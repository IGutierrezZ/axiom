# Propuesta: Eventos del ciclo de vida en Azure DevOps (C-07-e)

## Intención

Hoy el plugin `azure-devops` solo lee: importa el work item (WI) de origen y le pone una etiqueta de vuelta (C-07-d). Todo lo que el launcher de KVP hace a mano en cada fase sigue siendo manual. Ese trabajo es crear una Task de Azure por rol al aprobar el design, estimarla, sustituir la descripción del WI por un resumen de la spec, mover estados, asignar la Task e imputar horas al cerrar el rol. `Deliver` devuelve un error permanente (RF-ADO-14) y `TimeEntry` existe en el esquema, pero nadie la produce.

Este cambio automatiza esas escrituras con verbos deterministas. Pide confirmación humana donde el usuario la pidió (estimación e imputación en `checkpointed`) y no hace paradas en `continuous`, donde el resumen final informa de lo escrito. Por qué ahora: C-07-a, C-07-b, C-07-c, C-07-d y C-03-b ya están fusionados, y C-07-f (`{task_ref}`) y C-07-h (vista de Tasks e imputaciones) dependen de este cambio.

Éxito: con el doble `adotest`, aprobar el design crea una Task por rol con su estimación. Si falla, la aprobación queda registrada, se ofrecen las dos salidas y `ensure --role` se niega hasta que haya vínculo. Aprobar la spec sustituye la descripción y envía los IDs funcionales aparte. Cerrar el rol propone horas, las imputa sin pisar cambios concurrentes y registra quién lo hizo. En `continuous` no hay paradas. En ODD no se crea ninguna Task.

## Alcance

### Dentro del alcance

1. **Puerto opcional de escritura de Tasks en el núcleo** (`internal/integrations`). Cubre estas operaciones:
   - leer las Tasks hijas del WI;
   - crear una Task hija;
   - leer una Task (tipo, padre, estado, `rev` y horas);
   - actualizar las horas con `test /rev`;
   - cambiar el estado y asignar;
   - sustituir la descripción;
   - sustituir un conjunto de etiquetas con prefijo.

   Se detecta por aserción de tipo, igual que `WorkItemSource` (RF-INT-24).
2. **`integrations.yaml` con vínculos de Task y con imputaciones que llevan actor.**
   - El vínculo `kind: task` incluye `role`, `estimate_hours` (estimación confirmada) y `started_at` (primera entrada del rol).
   - `TimeEntry` gana `actor` (identidad git, C-07-a) y cada imputación tiene un `source` único por cierre (por ejemplo `role.close/<rol>/<n>`). Así un rol rechazado y aprobado de nuevo puede imputar dos veces sin chocar con la identidad `(plugin, source, item_id)`.
   - El esquema sube a `axiom.integrations/v2`. Los binarios nuevos siguen leyendo `v1`.
3. **Verbos nuevos bajo `axiom integrations`**, todos síncronos (fuera de la outbox), con salida JSON versionada y rechazos tipados que nombran su resolución:
   - `tasks create --change <c> [--estimate <rol>=<h> ...]`: crea o reutiliza la Task de cada rol del roster sellado y escribe `OriginalEstimate` y `RemainingWork` con la estimación.
   - `tasks bind --change <c> --role <rol>=<id> [--estimate <rol>=<h>]`: vincula a mano una Task. Valida que existe, que es de tipo Task y que cuelga del WI del cambio.
   - `summary push --change <c>`: sustituye `System.Description` por el resumen HTML de la spec y sincroniza las etiquetas de IDs funcionales. También funciona sobre la carpeta archivada, con los IDs definitivos.
   - `time propose --change <c> --role <rol>`: solo lectura. Muestra el reloj desde el inicio del rol recortado a la jornada, la estimación y el `CompletedWork` acumulado leído de ADO.
   - `time record --change <c> --role <rol> --hours <h>`: suma `<h>` a `CompletedWork`, ajusta `RemainingWork` con `test /rev` y registra la imputación con actor, minutos y fecha.
4. **Barrera de P8 en `ensure --role`**: el nuevo rechazo tipado `task_not_bound` vive en `validateEnsureRole`, antes de mutar nada. Solo aplica cuando el cambio tiene vínculo `kind: work_item` y hay activo un plugin con la capacidad `items.create`. Su resolución nombra `axiom integrations tasks create --change <c>` y `axiom integrations tasks bind --change <c> --role <rol>=<id>`.
5. **Adaptador `azure-devops`**:
   - Creación de Task hija (POST no reintentable) con control de duplicados por marca determinista, sin heurísticas de título.
   - Validación de la Task en `bind`.
   - Lectura y escritura de horas con `test /rev`.
   - Sustitución de la descripción y sincronización de etiquetas.
   - `Deliver` deja de estar vacío para `role.start` (pasa la Task a `Active` y la asigna a la identidad ADO de C-07-c), `role.apply.approved` (pasa la Task al estado configurado, por defecto `Tech Review`) y `archive` (transición del WI).
   - Manifiesto con las capacidades `items.create`, `relations`, `estimation`, `time`, `items.transition` y `assignment`, y con esos tres eventos.
6. **Configuración nueva de `integrations.azure-devops`**, con valores por defecto documentados en `axiom.example.yaml`:
   - estado de la Task al iniciar el rol, al aprobar su apply y al cerrarlo en `continuous`;
   - estado del WI al archivar;
   - prefijo de las etiquetas de IDs (por defecto `req:`);
   - tope de esas etiquetas;
   - jornada: horas por día (por defecto 8) y días laborables (por defecto de lunes a viernes).
7. **Renderizador determinista de markdown a HTML simple** del resumen de la spec, en un paquete Go hoja nuevo. Solo emite `h2`, `h3`, `p`, `ul`, `li`, `code` y `strong`, con todo el texto escapado: sin `style`, sin SVG y sin scripts. A igual entrada produce la misma salida byte a byte. No reutiliza `resumen-spec.html`.
8. **Doctrina**:
   - `sdd-design` propone la estimación por rol en una sección fija de `design.md`.
   - El orquestador invoca los verbos en ambos estilos: en `checkpointed` tras registrar la compuerta y en `continuous` al terminar la fase. También invoca `ensure --role` antes del apply cuando hay tracker activo.
   - En `sdd-apply`, propuesta e imputación al cerrar el rol.
   - En `sdd-archive`, `summary push` con los IDs definitivos y un resumen final de `continuous` con la estimación por Task, las horas imputadas, quién y cuándo.
   - Se actualizan `routing.go`, las copias espejo y los goldens.
9. **Rechazo de ODD**: los verbos nuevos rechazan un nombre del carril ODD con un rechazo tipado. Este cambio no crea Tasks ni horas en ODD.
10. **Deltas de spec** que contradicen las specs vivas (ver «Capacidades»), y pruebas herméticas con `adotest` e `integrationstest`.

### Fuera del alcance

- Verbo libre de imputación `axiom time log`: el usuario eligió «al cerrar el rol».
- Tasks, estimación o imputación en ODD.
- Comentarios en el WI (`System.History`).
- Plantillas de rama, `{task_ref}` y `ArtifactLink` de rama (C-07-f).
- Cualquier interfaz web o del dashboard (C-07-h).
- Transportar los IDs funcionales dentro de `System.Description` como sustituto de las etiquetas: P12 exige enviarlos aparte (se descarta C2).
- Crear el WI de origen, o enlazar commits y PR (`vcs_links`).
- Cambiar el contrato de `gates.yaml` en `continuous` y que `emit` fabrique aprobaciones.

## Capacidades

### Capacidades nuevas

Ninguna. El comportamiento nuevo se reparte entre las capacidades vivas que ya lo gobiernan. Los requisitos nuevos llevarán IDs provisionales (`prov:ADO:n`, `prov:INT:n`, `prov:BRA:n`, `prov:ORQ:n`, `prov:ARC:n`).

### Capacidades modificadas

- `azure-devops-adapter`:
  - MODIFIED RF-ADO-01: capacidades de escritura y los eventos `role.start`, `role.apply.approved` y `archive`. Se sustituye el escenario «Ninguna capacidad de escritura del ciclo de vida».
  - MODIFIED RF-ADO-02: claves de configuración nuevas y sus formatos.
  - MODIFIED RF-ADO-14: `Deliver` actúa sobre los tres eventos, `change.start` sigue siendo error permanente, y se reescribe el escenario «Verbos de despacho sin cambios».
  - Se ajusta el párrafo de propósito: hoy dice «ningún evento del ciclo de vida… no crea work items».
  - RF-ADO-07 se mantiene como regla, porque toda escritura de horas, estado, descripción y etiquetas la cumple. La spec decide si su texto necesita algún retoque.
  - ADDED: creación de Task hija con control de duplicados; validación de `bind`; lectura y escritura de horas; sustitución de la descripción; etiquetas de IDs funcionales; estado y asignación por evento.
- `integration-plugins`:
  - MODIFIED RF-INT-09: esquema `v2`, vínculo `kind: task` con estimación e inicio, `TimeEntry` con actor y la regla de creación del fichero. Se mantiene «ODD no escribe `integrations.yaml`».
  - MODIFIED RF-INT-18: alta y ayuda de los verbos nuevos.
  - ADDED: puerto opcional de Tasks; verbos `tasks create|bind`, `summary push` y `time propose|record`; rechazo de ODD; precondiciones por estilo de ejecución.
  - RF-INT-12 (`emit`) no cambia con el enfoque elegido, y la spec debe confirmarlo.
- `change-branch-lifecycle`: MODIFIED RF-BRA-33, con el nuevo código `task_not_bound` y su resolución nombrada, y el registro de `started_at` la primera vez que el rol queda `ready`. RF-BRA-34 no cambia.
- `sdd-orchestrator-assets`: ADDED, la doctrina de invocación de los verbos en `checkpointed` y en `continuous`, y `ensure --role` antes del apply con tracker activo.
- `sdd-archive-lifecycle`: ADDED, el `summary push` final con los IDs definitivos y el resumen de `continuous` con estimaciones e imputaciones.

## Enfoque

**Disparo (A1).** Los verbos son deterministas y dedicados, y la doctrina los invoca de la misma forma en ambos estilos:

| Momento | `checkpointed` (tras registrar la compuerta) | `continuous` (al terminar la fase) |
|---|---|---|
| spec | `summary push` | `summary push` |
| design | `tasks create` (estimación de `design.md`) | `tasks create` (estimación aceptada sin parada) |
| entrada del rol | `ensure --role <rol>`: barrera, `started_at` y `role.start` | igual, invocado por la doctrina antes del apply |
| cierre del rol | `time propose`, confirmación y `time record` | `time propose` y `time record` con la propuesta, sin parada |
| archivo | `summary push` sobre la carpeta archivada; transición del WI por el evento `archive` | igual |

- `gate record` no cambia. Los cambios de estado de la Task (`Active`, `Tech Review`) y del WI siguen el despachador duradero existente (outbox, RF-INT-06/10): nunca hacen fallar la compuerta, y un estado inexistente en el proceso termina como abandono tipado visible en `sync`.
- En `continuous` no existe `role.apply.approved` (sin compuertas). Por eso `time record` aplica allí el estado de cierre configurado, con la misma operación idempotente del adaptador. En `checkpointed` lo aplica el evento y `time record` no toca el estado, para no tener dos orígenes.
- Precondiciones, sin fabricar aprobaciones:
  - En `checkpointed`, `summary push` exige `spec` aprobada, `tasks create` exige `design` aprobada y `time *` exige `role-apply:<rol>` aprobada.
  - En `continuous` basta con que el artefacto exista.

**Estimación.** `sdd-design` escribe una sección fija `## Estimación por rol` en `design.md` (una línea `- <rol>: <h> h` por rol del roster).
- Como forma parte del digest de la compuerta `design`, aprobar el design en `checkpointed` es confirmar las horas. Para cambiarlas se itera el design o se pasa `--estimate` explícito, que prevalece.
- `tasks create`, el reintento y `bind` leen esa sección cuando no se pasa `--estimate`. El reintento no depende de que el agente recuerde las horas.
- La estimación confirmada queda en el vínculo `kind: task`.

**Creación de Task sin duplicados ni heurística de título.**
- Cada Task creada lleva la etiqueta marca `axiom-task:<cambio>:<rol>`. No usa el prefijo `axiom:`, por la trampa de `foreignAxiomTags` en `linkback.go:70`.
- Antes de cualquier POST, `tasks create`:
  1. Reutiliza el vínculo ya registrado en `integrations.yaml`.
  2. Si no hay vínculo, lee las Tasks hijas del WI y busca la marca exacta. Con una coincidencia la vincula. Con varias se niega y nombra `bind`.
  3. Solo con cero coincidencias crea la Task con la marca, la relación `System.LinkTypes.Hierarchy-Reverse` al WI y las horas en el mismo POST, y vincula la Task en cuanto recibe la respuesta.
- Así, un POST con la respuesta perdida se recupera en el siguiente intento por la marca, sin duplicar.
- El POST no se reintenta automáticamente, como ya prevé `client.go:211`.

**IDs funcionales aparte (P12).**
- Una etiqueta `System.Tags` por ID con el prefijo configurable, por defecto `req:` (`req:prov:ADO:3` antes de archivar y `req:RF-ADO-20` después).
- Se sustituye el conjunto de forma idempotente: se retiran las etiquetas con ese prefijo que ya no correspondan y se conservan las ajenas y la `axiom:<cambio>`. Todo va en un solo PATCH con `test /rev`.
- Si se supera el tope configurado o el límite de 100 etiquetas del WI, no se envía ninguna etiqueta de ID y se emite un aviso tipado: un envío parcial engañaría. La descripción se actualiza igualmente.
- Al archivar, `summary push` lee los IDs definitivos asignados (`ids.yaml` de la carpeta archivada) y reenvía la descripción y las etiquetas, para que no queden `prov:` obsoletos en el WI.

**Resumen de la spec.**
- Un renderizador Go puro recorre las deltas del cambio (capacidad, título del requisito con su ID y primer párrafo normativo) y produce HTML de la lista blanca indicada.
- Sustituye `System.Description` completo, de forma idempotente: si el HTML no cambia, no se envía PATCH.
- El original sigue en `context/` y en el historial de revisiones del WI.

**Imputación.**
- `time propose` calcula el reloj desde `started_at` hasta ahora. Cada día laborable configurado suma como mucho las horas de la jornada, y los demás días no suman.
- Muestra esa propuesta junto a `OriginalEstimate`, `RemainingWork` y `CompletedWork`, leídos en el momento de ADO; incluye las horas metidas a mano en Azure.
- `time record` lee el `rev`, suma a `CompletedWork`, hace `RemainingWork = max(0, RemainingWork − h)` y escribe con `test /rev`.
- Si hay conflicto de `rev`, no reaplica. Devuelve un error tipado y muestra los valores releídos para que la persona decida.
- Solo tras una escritura con éxito registra `TimeEntry{actor, minutes, date, source}`.

**Orden de commits (`single-pr`):**
1. Núcleo: puerto, esquema `v2` y rechazo `task_not_bound`.
2. Adaptador ADO y doble `adotest` ampliado.
3. Renderizador del resumen.
4. Verbos `tasks`, `summary` y `time`.
5. Doctrina y goldens.
6. Deltas de spec.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/integrations/` (`ports.go`, `workitems.go`, `links.go`, `plugin.go`) | Modificado | Puerto opcional de Tasks; `Link` de Task con `estimate_hours` y `started_at`; `TimeEntry.Actor`; esquema `v2` con lectura de `v1` |
| `internal/integrations/manifest/` (`azuredevops.go`, `capabilities.go`, `format.go`) | Modificado | Capacidades y eventos del manifiesto ADO; claves de configuración nuevas y su validación |
| `internal/integrations/azuredevops/` (`plugin.go`, `workitems.go`, `workitem_port.go`, nuevos `tasks.go` y `description.go`) | Modificado/Nuevo | POST de Task hija con marca; validación de `bind`; campos de horas; descripción; etiquetas con prefijo; estado y asignación; `Deliver` por evento |
| `internal/integrations/azuredevops/adotest/` | Modificado | Doble con creación, hijos, horas, estados y validación de `test /rev` para las escrituras nuevas |
| `internal/integrations/integrationstest/` | Modificado | Fake del puerto de Tasks para el núcleo y la CLI |
| `internal/specsummary/` (nombre a fijar en el diseño) | Nuevo | Paquete hoja: deltas de spec a HTML simple y determinista |
| `internal/reqid/`, `internal/livespec/` | Solo lectura | Extracción de IDs provisionales y definitivos (`ids.yaml`) |
| `internal/cli/integrations.go` y nuevos `integrations_tasks.go`, `integrations_summary.go`, `integrations_time.go` | Modificado/Nuevo | Verbos, ayuda, salida JSON versionada y rechazos con resolución nombrada |
| `internal/cli/branch_role.go`, `internal/branch/role_refusal.go` | Modificado | Rechazo `task_not_bound` en `validateEnsureRole`; `started_at` al quedar `ready` |
| `internal/cli/sdd_actor_identity.go` | Solo lectura | `defaultActor` para el actor de la imputación |
| `internal/integrations/identitystore/` | Solo lectura | Identidad ADO para la asignación en `role.start` |
| `internal/assets/skills/{sdd-design,sdd-apply,sdd-archive}/SKILL.md`, `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificado | Doctrina de estimación, invocación de verbos, cierre del rol y resumen final |
| `internal/components/agentguidance/routing.go`, `testdata/golden/*`, `internal/components/golden_test.go` | Modificado | Texto de enrutado junto al de `change start`; goldens regenerados |
| `axiom.example.yaml` | Modificado | Claves nuevas de `integrations.azure-devops` con sus valores por defecto |
| `openspec/specs/{azure-devops-adapter,integration-plugins,change-branch-lifecycle,sdd-orchestrator-assets,sdd-archive-lifecycle}` | Deltas | Ver «Capacidades» |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| **Sin verificar:** que el POST de creación acepte la relación padre (`Hierarchy-Reverse`) en el mismo cuerpo | Media | El diseño lo fija con el doble y, si es posible, con un WI de prueba que la persona autorice. Alternativa sin huérfanas: POST seguido de PATCH de relación, más búsqueda de la marca por WIQL en el proyecto para recuperar una Task sin padre |
| **Sin verificar:** qué HTML acepta `System.Description` | Media | Lista blanca mínima sin `style`, SVG ni scripts, y todo escapado. Si ADO filtra algo, se pierde formato, nunca contenido |
| **Sin verificar:** nombres de los campos de horas (`Microsoft.VSTS.Scheduling.OriginalEstimate`, `RemainingWork`, `CompletedWork`) y su presencia en la Task según el proceso | Media | Confirmarlos con `discovery` (campos por tipo). Si falta un campo, rechazo tipado que lo nombra, sin escritura parcial |
| **Sin verificar:** `Tech Review` no es un estado estándar de Agile ni de Scrum | Alta | Estado configurable y validado contra `discovery`. El fallo se reporta como abandono tipado en la outbox, sin romper la compuerta |
| La asignación en `role.start` reentregada por `sync` desde otra máquina asignaría a quien sincroniza | Media | El diseño decide entre fijar la identidad ADO en la emisión y asignar solo en el intento inmediato. Nunca asignar con una identidad distinta de la del actor del evento |
| Doble imputación si se pierde la respuesta del PATCH de horas y la persona reintenta | Baja | No hay reaplicación automática. El conflicto de `rev` muestra los valores releídos y la persona decide. La `TimeEntry` solo se registra tras éxito |
| Esquema `v2` estricto: un binario antiguo rechaza el fichero | Media | La subida de versión da un rechazo claro en lugar de un error de campo desconocido. Los binarios nuevos leen `v1` y `v2` |
| `continuous` depende de que la doctrina invoque los verbos | Media | Barrera mecánica en `ensure --role`, invocado por la doctrina antes del apply con tracker activo. Los rechazos nombran el verbo que falta |
| Límite de 100 etiquetas por WI y reserva compartida de etiquetas del proyecto | Baja | Tope configurable. Sin envío parcial. Una única marca por Task |
| La lectura de las horas en el cierre puede reflejar imputaciones manuales concurrentes | Baja | Lectura inmediatamente anterior y `test /rev` en la escritura (RF-ADO-07) |
| Trinquetes: código muerto, rechazos con resolución, canarios de PAT, residuos de IDs heredados | Media | Todo es alcanzable desde `cmd/axiom`; cada rechazo nombra su `axiom …`; las rutas `/api` nuevas entran en `integrations_ado_canary_test.go`; solo se usan IDs `prov:` en el cambio |
| Solape con C-06-f en `sdd_gate.go` | Baja | Este enfoque no toca `sdd_gate.go` |

## Plan de reversión

- **Código**: un `git revert` del merge de la única PR devuelve el plugin a solo lectura.
  - El manifiesto vuelve a escuchar solo `change.start` y `Deliver` vuelve a ser un error permanente.
  - Los verbos nuevos desaparecen, y `ensure --role` pierde `task_not_bound`.
  - La doctrina y los goldens vuelven a su estado anterior.
- **Datos locales**: un `integrations.yaml` ya escrito en `v2` no lo lee el binario revertido. La reversión incluye un paso documentado para bajar el fichero a `v1`: retirar los vínculos `kind: task` y los campos `actor`, `estimate_hours` y `started_at`. Las operaciones pendientes de la outbox para eventos que el manifiesto ya no escucha se descartan con `sync`, que las abandona tipadas.
- **Datos en Azure DevOps**: las Tasks creadas, las horas imputadas, las descripciones sustituidas y las etiquetas `req:` y `axiom-task:` no se revierten automáticamente. La descripción original queda en el historial de revisiones del WI y en `context/`.
- **Desactivación sin revertir**: `integrations.azure-devops.enabled: false` (ya existe) apaga todos los efectos, y la barrera deja de aplicar porque no hay plugin activo con `items.create`.

## Dependencias

- Fusionados y vigentes: C-07-a (identidad git del actor), C-07-b (marco, outbox, `integrations.yaml`), C-07-c (cliente ADO, `discovery`, identidad ADO), C-07-d (vínculo `kind: work_item`, `LinkBack`) y C-03-b (IDs provisionales y `ids.yaml`).
- Blanda: C-06-b. No se reutiliza `resumen-spec.html`, así que no condiciona nada.
- Desbloquea a C-07-f (`{task_ref}`) y a C-07-h (vista de Tasks e imputaciones).
- No requiere red en pruebas: todo es hermético con `adotest`. Una comprobación contra Azure DevOps real, si se hace, necesita autorización explícita de la persona para el destino y el PAT.

## Criterios de éxito

- [ ] Con el doble ADO, aprobar el design (o terminarlo en `continuous`) crea una Task hija por rol con la marca `axiom-task:<cambio>:<rol>`, `OriginalEstimate` y `RemainingWork` iguales a la estimación, y el vínculo `kind: task` en `integrations.yaml`.
- [ ] Repetir `tasks create` tras una respuesta perdida no crea una segunda Task: la reutiliza por la marca.
- [ ] Un fallo de creación deja la compuerta `design` aprobada y la salida nombra `tasks create` y `tasks bind`. `ensure --role <rol>` se niega con `task_not_bound`, sin mutar nada, hasta que la Task está vinculada.
- [ ] `tasks bind` rechaza un ID inexistente, un ID que no es Task o una Task que no cuelga del WI del cambio.
- [ ] `summary push` sustituye `System.Description` por HTML sin `style`, SVG ni scripts, idéntico byte a byte en dos ejecuciones, y sincroniza las etiquetas `req:<ID>`: añade, retira las obsoletas y conserva las ajenas.
- [ ] Al archivar, las etiquetas y el resumen pasan de IDs `prov:` a los definitivos.
- [ ] `time propose` muestra la propuesta recortada a la jornada, la estimación y el `CompletedWork` leído de ADO. `time record` escribe con `test /rev`, y un `rev` obsoleto devuelve conflicto sin reaplicar.
- [ ] `time record` registra la imputación con actor, minutos y fecha.
- [ ] `role.start` deja la Task en `Active` y asignada. `role.apply.approved` (o `time record` en `continuous`) la deja en el estado configurado. Un estado inexistente se reporta sin romper ningún verbo de origen.
- [ ] En `continuous` no hay ninguna parada, y el resumen final del archivo lista la estimación por Task, las horas imputadas, quién y cuándo.
- [ ] Los verbos nuevos rechazan un nombre ODD con un rechazo tipado, y ninguna ruta de ODD crea Tasks ni escribe `integrations.yaml`.
- [ ] Sin `integrations.azure-devops` activo, la salida de `ensure --role` y la de `gate record` no cambian.
- [ ] Siguen en verde los trinquetes de código muerto, de resolución de rechazos, los canarios de PAT y `TestRepositoryLegacyIDResidues`.
