# Diseño: Eventos del ciclo de vida en Azure DevOps (C-07-e)

> **Cambio:** `eventos-ado`. **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> **Entradas:** `proposal.md`, `exploration.md`, las cinco deltas de `specs/` y `coverage-review.spec.md` (22 huecos, resueltos o clasificados al final).
> **Kickoff:** `continuous`, rol único `fullstack:blocking`, `monorepo-embedded`, entrega `single-pr` con un commit por unidad.

## Enfoque técnico

El cambio sigue el enfoque A1 de la propuesta. Los verbos de `axiom integrations` son deterministas y síncronos, y la doctrina los invoca igual en `checkpointed` y en `continuous`. Las transiciones de estado ligadas a eventos (`role.start`, `role.apply.approved` y `archive`) siguen el despachador duradero existente: outbox y `sync`. Se reparte en cinco capas, cada una con una frontera ya presente en el repositorio:

1. **Núcleo neutral** (`internal/integrations`):
   - esquema `axiom.integrations/v2` de `integrations.yaml`, con lectura de `v1`;
   - vínculo `kind: task` con estimación e inicio;
   - `TimeEntry.Actor`;
   - puerto opcional `TaskTracker`, detectado por aserción de tipo;
   - diario local de imputaciones pendientes;
   - avisos de entrega y datos de entrega nuevos (`Deliverer`, `Identities`).
2. **Adaptador** (`internal/integrations/azuredevops`): implementación de `TaskTracker`, `Deliver` para los tres eventos y doble `adotest` ampliado.
3. **Lógica de los verbos** (`internal/trackersync`, paquete nuevo) y **renderizador** (`internal/specsummary`, paquete hoja nuevo):
   - estimación de `design.md`;
   - reloj recortado a la jornada;
   - extracción de IDs funcionales;
   - HTML determinista.
4. **CLI** (`internal/cli`): los verbos `tasks create|bind`, `summary push` y `time propose|record`, la barrera `task_not_bound` en `validateEnsureRole` y el registro de `started_at` tras `ensure --role`.
5. **Doctrina**: sección única en `_shared/sdd-orchestrator-sections.md`, una frase en `agentguidance/routing.go` para ODD, las skills `sdd-design`, `sdd-apply` y `sdd-archive`, las copias espejo y los goldens.

Ninguna capa nueva rompe las reglas de importación de `internal/integrations/imports_test.go`:
- el núcleo no conoce `internal/cli` ni al adaptador;
- `internal/branch` no importa `internal/integrations`;
- `manifest` sigue siendo una hoja de la biblioteca estándar.

## Decisiones de arquitectura

### D1. Puerto `TaskTracker` de grano medio con un único envoltorio genérico

**Elección:** una interfaz opcional `integrations.TaskTracker` con ocho métodos: hijas, buscar o crear, leer, validar, estimación, horas, transición y resumen. El núcleo ofrece un único envoltorio genérico, `integrations.WithTaskTracker`, que hace cuatro cosas:
- detecta el puerto por aserción de tipo;
- compone `PortRequest` con secretos acotados y rastreados;
- recupera pánicos;
- sanea errores mediante `invokePort`.

**Alternativas descartadas:**
- Un método por operación elemental (`AddTag`, `SetField`…). Sacaría al núcleo reglas que la spec asigna al adaptador: marca determinista, `test /rev` y una única escritura para descripción y etiquetas.
- Una función envoltorio por método, como en `workitems.go`. Serían ocho copias del mismo patrón.

**Justificación:**
- `prov:INT:1` exige tipos neutrales, `invokePort` y `port_unsupported`.
- `prov:ADO:1` y `prov:ADO:5` exigen que la búsqueda por marca y la sincronización de etiquetas sean reglas del adaptador.
- El genérico mantiene una sola ruta de saneado.

### D2. `integrations.yaml` v2 con lectura de v1 y reescritura solo al cambiar

**Elección:**
- `LinksSchemaV2 = "axiom.integrations/v2"`; `parseLinks` acepta `v1` y `v2`.
- Un esquema distinto falla con `esquema "<leído>" no admitido; admitidos: axiom.integrations/v1, axiom.integrations/v2`, antes de validar campos. Para ello se decodifica primero solo `schema` y después se hace la lectura estricta.
- `mergeLinks` escribe siempre `v2`, pero solo cuando la fusión cambia algo. Sin cambios no se escribe, así que un fichero `v1` intacto sigue siendo `v1`.
- Campos nuevos:
  - `Link.EstimateHours *float64` (`estimate_hours,omitempty`);
  - `Link.StartedAt *time.Time` (`started_at,omitempty`);
  - `TimeEntry.Actor string` (`actor,omitempty`).
- La fusión de un vínculo con identidad ya presente pasa de «conservar intacto» a dos reglas:
  - `EstimateHours` entrante no nulo y distinto → se actualiza;
  - `StartedAt` solo se fija si el existente es nulo.
- Validación:
  - `estimate_hours > 0` con, como mucho, dos decimales;
  - `started_at` en UTC;
  - una imputación cuyo `source` empieza por `role.close/` exige `actor` no vacío y `role` no vacío.

**Alternativas descartadas:**
- Mantener `v1` con campos opcionales: un binario antiguo daría un error de campo desconocido en lugar de un rechazo claro del esquema.
- Leer siempre de Azure DevOps (B2): el resumen final de `continuous` dependería de la red (`prov:ARC:2` lo prohíbe).

**Justificación:** RF-INT-09 modificado.

### D3. Funciones de vinculación exportadas, bajo el mismo cerrojo

**Elección:** funciones nuevas en `links.go`, todas sobre `Folder.acquire()` y `mergeLinks`:
- `BindTask(f, plugin, l Link, now)`: `source` `tasks.create` o `tasks.bind`. También actualiza `estimate_hours` de un vínculo existente (D17).
- `ReplaceTaskLink(f, plugin, role string, l Link, now)`: es la única función que retira un vínculo. Bajo el cerrojo, quita el `kind: task` del rol y añade `l`; el `started_at` anterior se copia si `l` no lo trae. `time_entries` no se toca (D18).
- `MarkRoleStarted(f, plugin, role, at)`: fija `started_at` solo si falta; devuelve si cambió.
- `RecordTimeEntry(f, plugin, e TimeEntry, now)`.
- `TaskLinks(f)` y `TimeEntries(f)`: lectura sin cerrojo.

Cada una respeta `f.guard` (carpeta archivada inmutable, RF-ARC-03).

**Alternativa descartada:** escribir el YAML desde la CLI. Duplicaría la escritura atómica y el cerrojo de RF-INT-08.

**Justificación:** RF-INT-09 («bajo el mismo cerrojo que las entregas»).

### D4. Lógica de verbos en `internal/trackersync` y renderizador hoja en `internal/specsummary`

**Elección:**
- `internal/specsummary` (solo biblioteca estándar) recibe un modelo ya extraído (capacidades → requisitos → título, primer párrafo y operación) y emite HTML. Es puro y determinista.
- `internal/trackersync` contiene:
  - `ParseEstimates` (sección `## Estimación por rol`);
  - `ProposeHours` (reloj);
  - `SpecModel` (deltas → modelo y lista de IDs, con `openspecdoc.ParseChangeSpec` y `livespec.ReadTraceFile`);
  - el cálculo de `role.close/<rol>/<n>`.
- La CLI orquesta: precondiciones, puerto, JSON y rechazos. Es el mismo reparto que `internal/changestart` y `cli/change_start.go`.

**Alternativa descartada:** todo en `internal/cli`. Mezclaría reglas puras con E/S, alargaría el trinquete de rechazos (que solo audita `internal/cli`, `internal/fileprim` e `internal/sddstatus`) y dificultaría las pruebas de tabla del reloj y del renderizador.

**Justificación:** el patrón existente y la propuesta («paquete Go hoja nuevo»).

### D5. Barrera `task_not_bound` en `validateEnsureRole`, sin red

**Elección:**
- Tras `role_not_in_roster`, `validateEnsureRole` comprueba, en el mismo planning home donde encontró el kickoff:
  1. que hay un plugin activo cuya entrada del catálogo declara `items.create` (`manifest.Lookup`);
  2. que `integrations.yaml` de la carpeta del cambio tiene un vínculo `kind: work_item` de ese plugin;
  3. que existe un vínculo `kind: task` del rol.
- Si se cumplen 1 y 2 y falta 3, devuelve `branch.TaskNotBound(change, role, dir)`, con el código nuevo `CodeTaskNotBound = "task_not_bound"` en `internal/branch`. El paquete de ramas solo conoce el código; no importa `integrations`.
- Un `integrations.yaml` ilegible con un plugin activo con `items.create` es el rechazo `integrations_links_invalid`, que nombra la causa y `axiom integrations sync --change <c>` (falla cerrado: la barrera de P8 no se esquiva con un fichero roto).
- No hay red: no se revalida la Task en el tracker (ver hueco 15).

**Alternativas descartadas:**
- La barrera en el motor (`engine/ensure.go`): el motor no conoce el rol (RF-BRA-33 y la prueba de arquitectura).
- Revalidar contra Azure DevOps: haría a `ensure` dependiente de la red y del PAT.

**Justificación:** RF-BRA-33 modificado y `prov:ORQ:2`.

### D6. `started_at` antes de despachar `role.start`

**Elección:**
- En `runBranchEnsure`, con `roleGiven` y resultado `ready`, `recordEnsureRoleStarted` corre antes de `emitEnsureRoleStart`.
- `recordEnsureRoleStarted` llama a `integrations.MarkRoleStarted`:
  - solo si hay plugin activo con el puerto de Tasks y vínculo de Task del rol;
  - con la guarda A de ramas (`integrationWriterGuard`).
- Un fallo se avisa por `integrationWarn` con el código `started_at_not_recorded` y no altera stdout ni el código de salida.
- El orden garantiza que la instantánea de `Delivery.Links` de `role.start` ya lleva el vínculo con `started_at`.

**Alternativa descartada:** registrarlo desde `Deliver`. `role.start` no persiste nada útil y solo se entrega una vez por clave.

**Justificación:** `prov:BRA:1`.

### D7. Propuesta de horas: ventana diaria anclada a la hora de inicio, zona local y cuartos de hora

**Elección (cierra los huecos 1 y 20 y el supuesto de `prov:INT:7`):**
- **Zona horaria:** `time.Local` del proceso (respeta `TZ`). La salida la nombra (`timezone`, con nombre y desfase) para que la propuesta sea auditable.
- **Inicio del intervalo:** `since = max(started_at, recorded_at de la última imputación role.close/<rol>/* del rol)`. Un segundo cierre no vuelve a contar el tiempo ya imputado (hueco 1).
- **Ventana de cada día:** el ancla es la hora local de `started_at`, nunca la de `since`, para que todos los cierres compartan ventana. Para cada fecha local D entre `fecha(since) − 1` y `fecha(now)`:
  - ventana `[D + ancla, D + ancla + workday_hours)`, construida con `time.Date(…, loc)` más una duración absoluta: un día con cambio de hora sigue sumando `workday_hours` reales;
  - cuenta solo si el día de la semana de D (el del inicio de la ventana) está en `workdays`;
  - el aporte de D es la intersección de la ventana con `[since, now)`.
  - Una jornada que cruza la medianoche se imputa al día en que empieza.
- **Redondeo:**
  - suma exacta en segundos → minutos por truncamiento;
  - `hours` propuestas = minutos redondeados al cuarto de hora más próximo (media hacia arriba).
  - La salida da ambos valores.
- Con los escenarios de la spec:
  - lunes 09:00 → martes 12:00 da 11 h (09–17 y 09–12);
  - 00:00 → 23:59 del lunes da 8 h;
  - sábado → domingo da 0 h.

**Alternativas descartadas:**
- «Recortar cada día natural a `workday_hours`» sin ancla: daría 16 h en el primer escenario de la spec, en contradicción con su valor de 11 h.
- Una clave de configuración de hora de inicio: la spec cierra la lista de claves (RF-ADO-02).

**Justificación:** es el único modelo, sin claves nuevas, que reproduce los tres escenarios y no imputa dos veces.

### D8. `--hours` y minutos

**Elección (hueco 6):**
- `--hours` admite un decimal con punto y como mucho dos decimales, con `0 < h ≤ 1000`.
- `minutes = round(h × 60)` (media hacia arriba) y debe ser ≥ 1.
- Fuera de dominio es un error de uso (salida 2) sin red.
- Al tracker se envían horas con dos decimales: `CompletedWork = round2(C + h)` y `RemainingWork = round2(max(0, R − h))`.

**Justificación:** RF-INT-09 exige `minutes` entero positivo, y dos decimales evitan el ruido de coma flotante en Azure DevOps.

### D9. Diario local de imputación pendiente

**Elección (hueco 5 y recuperación de la doble imputación):** `time record` usa `.time-pending.yaml` en la raíz de la carpeta del cambio:
- esquema `axiom.integrations-time-pending/v1`;
- un elemento por rol;
- es un fichero con punto, así que es estado local que nunca se publica (`docpaths.LocalState`);
- su nombre no casa con el patrón temporal `.integrations-*.yaml`.

Flujo de `time record`:
1. Leer la Task (rev R0 y horas C0, Rm0).
2. Escribir la intención `{role, item_id, hours, minutes, actor, date, source, base_rev: R0, expected_completed, expected_remaining}` bajo el cerrojo del cambio. Si no se puede escribir: rechazo `write_failed` antes de tocar el tracker.
3. `PATCH` con `test /rev = R0`.
4. Si el `PATCH` tiene éxito, o se recupera tras el reintento (D10), `RecordTimeEntry` y después se borra la intención.
5. **Cuándo se borra la intención sin registrar (hueco 13 de la revisión del design):** solo cuando consta que el `PATCH` no se aplicó:
   - un `RevConflictError` en el primer intento (`AfterRetry: false`);
   - una respuesta 4xx que no es conflicto (validación, autorización, 404).
6. **Cuándo se conserva:** ante todo resultado incierto. Son inciertos el `TransportError` sin respuesta (en el intento o en el reintento), el tiempo agotado, un 5xx tras el presupuesto y un conflicto `AfterRetry` cuya relectura no verifica D10. En esos casos el verbo rechaza con `tracker_unavailable` y añade el aviso `time_pending`; la imputación no se registra.

Una intención que sobrevive (fallo local tras el `PATCH`, resultado incierto, proceso interrumpido) se reconcilia **solo al principio del siguiente `time record`** de ese rol (hueco 8 de la revisión del design: `time propose` es de solo lectura, como dice `prov:INT:7`).
- `time propose` con una intención pendiente no la toca: muestra la propuesta y avisa `time_pending`, con la resolución `axiom integrations time record --change <c> --role <r>`.
- La reconciliación relee **la Task de la intención** (`item_id` de la intención), aunque `tasks bind --replace` haya cambiado ya el vínculo del rol (hueco 14). Por eso `--replace` no necesita rechazarse con una intención pendiente.

| Lectura actual de la Task de la intención | Interpretación | Acción |
|---|---|---|
| `rev == base_rev` | el `PATCH` no se aplicó | se borra la intención y se sigue |
| `rev > base_rev` y `CompletedWork == expected_completed` y `RemainingWork == expected_remaining` | se aplicó | se registra la imputación de la intención, se borra y se avisa `time_record_recovered`; `time record` NO aplica sus `--hours` en esa ejecución (termina con 0 y nombra `axiom integrations time propose`); en `continuous` aplica la transición de cierre (D21) |
| cualquier otro caso | ambiguo | rechazo `time_pending_ambiguous` con los valores releídos; la resolución es `axiom integrations time record --change <c> --role <r> --resolve-pending applied|discarded` |

`--resolve-pending applied` registra la imputación de la intención sin tocar el tracker; `discarded` la descarta. Es incompatible con `--hours`, y sin intención pendiente del rol es error de uso. Este flujo está en la spec (`prov:INT:8`): diario, `time_pending`, conservación ante un resultado incierto, reconciliación solo en `time record` y sobre la Task de la intención, `time_pending_ambiguous` y `--resolve-pending`.

**Alternativas descartadas:**
- Deducir solo leyendo `CompletedWork`, sin base: no distingue imputaciones manuales.
- Registrar la imputación «pendiente» en `integrations.yaml`: RF-INT-09 prohíbe estado de entrega en el fichero versionado.

**Justificación:** «solo tras una escritura con éxito» se cumple (nada entra en `integrations.yaml` antes) y ninguna ruta reaplica horas sin decisión humana.

### D10. Respuesta perdida del `PATCH` de horas dentro de la misma ejecución

**Elección:**
- El cliente ya reintenta el `PATCH` con `test /rev`. Si el primer intento se aplicó y la respuesta se perdió, el reintento devuelve `*RevConflictError{AfterRetry: true}` (`workitems.go:181`).
- En ese caso, `RecordHours` relee la Task y la da por aplicada (`Recovered: true`) solo si `rev == base_rev + 1` y las dos horas coinciden con las esperadas.
- En cualquier otro conflicto devuelve `integrations.RevConflictError` con los valores releídos, sin reaplicar (`prov:ADO:3`).

**Justificación:** convierte la respuesta perdida en éxito verificable sin heurística de título y sin segunda escritura.

### D11. Creación de Task: buscar por marca (hijas y WIQL) y crear con un único `POST`

**Elección (`prov:ADO:1`, huecos 10, 16 y 21):**
- **Marca:** `axiom-task:<cambio>:<rol>`, siempre en minúsculas. La coincidencia ignora mayúsculas (como `LinkBack`), para no duplicar si Azure DevOps reescribe la grafía. Es un criterio de etiqueta, nunca de título.
- **Búsqueda, en orden:**
  1. `GetWorkItem(WI)` con relaciones → IDs de `System.LinkTypes.Hierarchy-Forward` → `GetWorkItems` en lotes de 200. Cubre cualquier número de hijas sin paginación de servidor.
  2. Se filtran las de tipo `Task` cuya categoría de estado no es `Removed` (estados del tipo por `GET workitemtypes/Task/states`, con caché por proceso) y que llevan la marca.
  3. Con cero coincidencias, consulta WIQL en el proyecto: `[System.WorkItemType] = 'Task' AND [System.Tags] CONTAINS '<marca>'`, con filtrado exacto posterior. Detecta una Task huérfana del camino alternativo.
- **Decisión:**

  | Coincidencias | Acción |
  |---|---|
  | 1 hija | `found_by_mark` |
  | 1 huérfana | `PATCH` de relación con `test /rev` → `repaired` |
  | 1 bajo otro padre, o > 1 | `task_ambiguous` con los IDs |
  | 0 | crear |

  Si WIQL falla, error del tracker y **no** se crea.
- **Creación:** `POST …/_apis/wit/workitems/$Task` (no reintentable) con:
  - `System.Title = "<cambio> / <rol>"`;
  - `System.Tags = <marca>`;
  - `System.AreaPath` y `System.IterationPath` copiados del WI padre (si los trae);
  - `OriginalEstimate` y `RemainingWork = estimación`;
  - `/relations/-` con `Hierarchy-Reverse` hacia la URL API del WI.
- **Camino alternativo:** si el `POST` responde 400, la Task no se ha creado; se repite una vez sin relaciones y después se hace `PATCH` de relación con `test /rev = 1`. Si ese `PATCH` falla, el error `task_parent_link_failed` nombra el ID. El siguiente `tasks create` la encuentra por WIQL y la repara.
- **Campos de horas:** antes de cualquier escritura con estimación, `GET workitemtypes/Task/fields` (con caché) comprueba que existen `Microsoft.VSTS.Scheduling.OriginalEstimate` y `RemainingWork`. Si falta alguno: `FieldMissingError` sin `POST`.
- **Concurrencia entre máquinas (hueco 16):** tras crear se relee la lista de hijas; si hay más de una con la marca, aviso `task_duplicated` que nombra `axiom integrations tasks bind`. No hay exclusión entre máquinas: Azure DevOps no ofrece unicidad.

**Alternativas descartadas:**
- Heurística de título (la spec la prohíbe).
- `POST` + `PATCH` siempre: duplica peticiones y deja más huérfanas.
- Reintentar el `POST` (`client.go:211`, RF-ADO-04).

### D12. Resumen y etiquetas en una sola escritura

**Elección (`prov:ADO:4`, `prov:ADO:5`, `prov:INT:4` y `prov:INT:5`, huecos 11, 12, 13 y 19):**
- `UpdateSummary` lee el WI, calcula la descripción y el conjunto de etiquetas y envía un único `PATCH` (`test /rev`) solo con las operaciones necesarias. Sin diferencias, no escribe. Ante un conflicto relee y repite hasta 3 veces.
- **Propiedad de las etiquetas:** una etiqueta es «nuestra» si empieza por `req_tag_prefix`, con distinción de mayúsculas (como dice la spec). Las demás se conservan.
- **Conjunto deseado:** `prefijo + ID` por cada ID de los requisitos `ADDED`, `MODIFIED` y `RENAMED` (el nombre nuevo). Los `REMOVED` no reciben etiqueta y, si ya la tenían, se retira.
- **Tope (hueco 11):**
  - si `len(deseadas) > req_tag_max`, o `ajenas + deseadas > 100`, o alguna deseada supera 400 caracteres o contiene `,` o `;`, el conjunto final son **solo las ajenas**: se retiran todas las nuestras, incluidas las `prov:` que ya estaban, y no se envía ninguna de ID;
  - aviso `req_tags_over_limit`, o `req_tag_invalid` con la etiqueta culpable;
  - la descripción se actualiza igualmente.
  - Así nunca queda un conjunto parcial ni un provisional obsoleto, y `prov:ARC:1` se cumple también por encima del tope.
- **Renderizador (hueco 12 de la spec y hueco 15 del design, alineado con `prov:INT:5`):**
  - capacidades por orden alfabético (`h2`);
  - dentro de cada una, **todos los requisitos en el orden del fichero**, sin agrupar por operación. Cada requisito lleva un `h3` con su título (en `RENAMED`, el nombre nuevo) y, si lo tiene, un `p` con su primer párrafo normativo.
  - `REMOVED` y `RENAMED` siguen esa misma regla de `h3` más `p`. Su `p` empieza con un marcador en `strong` (`<strong>Retirado.</strong>`, o `<strong>Renombrado desde:</strong> <título anterior>`) seguido del primer párrafo, si lo hay. Así se distingue la operación sin salir del orden ni de la lista blanca.
  - El orden del fichero es el de `openspecdoc.ParseDelta`, que agrupa por operación. `trackersync.SpecModel` lo reconstruye ordenando los requisitos de todas las secciones por `Requirement.Line`.
  - El renderizador no emite `ul` ni `li`: la lista blanca los admite, pero ninguna regla los necesita.
  - **Primer párrafo:** las líneas no vacías consecutivas tras el título, antes de cualquier línea que empiece por `#`, `-`, `|`, `>`, una valla o `(`. Si el primer bloque no es párrafo, no hay `p`.
  - **Marcado en línea:** solo pares completos de `` ` `` y `**`; un delimitador sin cerrar se emite literal y escapado.
  - **Tamaño máximo:** 64 KiB. Si se supera, se vuelve a renderizar sin los `p`; si aún se supera, se cortan requisitos y se cierra con `<p>… y N requisitos más</p>`. Es determinista.
- **IDs tras archivar (hueco 13):**
  - Se lee la carpeta archivada que devuelve `kickoff.ArchivedChangeRoot`: con varias del mismo cambio, la de fecha mayor.
  - Los IDs salen de sus deltas reescritas; un `prov:` residual se traduce con su `ids.yaml`, y sin entrada da `provisional_ids_unresolved`.
  - La descripción editada a mano se sustituye por diseño; la salida JSON lo indica con `description.changed: true`. El original queda en el historial del WI y en `context/`.
- **Orden respecto a la transición del WI:** la doctrina invoca `summary push` sobre la carpeta archivada antes de `axiom sdd publish --archive`, que es quien emite el evento `archive` y cierra el WI. Un WI que ya no admite edición devuelve `tracker_rejected` con el reintento.

**Alternativas descartadas:** dos `PATCH` (descripción y etiquetas), que rompen «una sola escritura»; un envío parcial de etiquetas (la spec lo prohíbe).

### D13. Transiciones y asignación por evento

**Elección (`prov:ADO:6`, `prov:ADO:7`, huecos 2, 3 y 4):**
- **Regla de avance, sin retrocesos:**
  - `role.start` cambia el estado solo si la categoría actual es `Proposed`. Si la Task está ya en `InProgress`, `Resolved` o `Completed` (por ejemplo, reentrada tras `Tech Review`), no la mueve y avisa `task_state_kept`.
  - `role.apply.approved` y el cierre de `continuous` la mueven salvo que esté en `Completed` o `Removed` (mismo aviso).
  - En el estado destino, no se escribe.
- **Estado inexistente:** `StateUnknownError{State, Valid}` con los estados del tipo (`workitemtypes/Task/states`); error permanente, sin escritura.
- **Transición prohibida por reglas del proceso (hueco 3):** un 400 de reglas en el `PATCH` de estado es `ErrTrackerRejected` con el mensaje saneado del servidor:
  - en `Deliver`, permanente (abandonada y visible en `sync`, con `--retry-abandoned` para reactivarla);
  - en el cierre de `continuous`, aviso `state_transition_failed`.
- **Asignación (supuesto de la máquina que entrega):**
  - `Dispatcher.Deliverer` es la identidad git de quien entrega (`defaultActor` en la CLI) y `Dispatcher.Identities` consulta `identitystore` por (plugin, organización). Ambos viajan en `Delivery`.
  - Solo se asigna si: `Event.Actor` no está vacío, no es `cli` y es igual a `Delivery.Deliverer`; hay identidad guardada; y en el equipo no hay dos miembros con su mismo nombre visible (aviso `assignment_ambiguous`).
  - El valor de `System.AssignedTo` es el nombre visible guardado. Tras el `PATCH` se compara el `id` devuelto con el guardado; si no coincide, aviso `assignment_mismatch`.
  - Si la Task ya está asignada a otro `id`, no se reasigna (aviso `task_assigned_to_other`).
  - Si falta cualquiera de las condiciones, se aplica el estado y se avisa `assignment_skipped` nombrando `axiom integrations connect --plugin azure-devops`.
  - Nunca se lee ni se emite el `uniqueName` (correo) del tracker: la decisión de privacidad de `discovery.go:83` se mantiene.
- **`task_not_bound` en `Deliver` (hueco 4):** se mantiene permanente, como dice `prov:ADO:6`. El mensaje nombra la salida real: obtener el vínculo (`git pull` del `integrations.yaml` publicado, o `axiom integrations tasks create|bind`) y reactivar con `axiom integrations sync --retry-abandoned`. En la máquina de origen no puede ocurrir, porque `ensure --role` exige el vínculo (D5).
- **Avisos de entrega:** `Outcome.Warnings []string` (campo nuevo y aditivo), con la forma `"<código>: <mensaje>"`. El despachador los añade saneados a `Report.Warnings`, que `sync` y `integrationWarn` ya muestran.
- **Cómo llega la asignación al puerto (hueco 1 de la revisión del design):**
  - `TransitionRequest.Assignee *TrackerIdentity`: nil significa no asignar.
  - `Deliver` de `role.start` decide antes si procede: actor igual a quien entrega e identidad guardada. Si procede, pasa la identidad; si no, pasa nil y añade `assignment_skipped`.
  - `TransitionTask` aplica en el adaptador la comprobación de homónimos, la de «ya asignada a otro `id`» y la verificación del `id` devuelto. Devuelve `Assigned` y sus avisos en `TransitionResult.Warnings`, que `Deliver` copia a `Outcome.Warnings`.
  - El cierre de `continuous` (CLI) nunca pasa `Assignee`.
- **Asignación independiente de la regla de estado (hueco 11):**
  - Una Task que ya está en el estado destino, o retenida por `task_state_kept`, pero sin asignar recibe un `PATCH` solo con `System.AssignedTo` si se cumplen las condiciones.
  - Una Task en el destino y ya asignada a esa persona no recibe ninguna escritura.
- **`archive` sin vínculo `kind: work_item` en la instantánea (hueco 11):** éxito sin red, con el aviso `work_item_not_linked`; nada que transicionar.

**Alternativa descartada:** guardar `uniqueName` en `identitystore` para asignar por correo. Cambia el contrato de privacidad de C-07-c y RF-ADO-11 sin pedirlo la spec.

### D14. Precondiciones comunes y orden de comprobación

**Elección (`prov:INT:9`, huecos 9, 14 y 22):** `trackerTarget(cwd, change, verb)` resuelve, en este orden:
0. Uso: banderas, `--estimate` de un rol fuera del roster o, en `bind`, de un rol distinto del de `--role` (hueco 9). Error de uso, salida 2. **Dominio de `--estimate` (hueco 7 de la revisión del design):**
   - forma `<rol>=<h>`, con `h` decimal con punto, positiva, de dos decimales como mucho y `≤ 1000`;
   - un rol repetido en las banderas es error;
   - todo se valida antes de cualquier petición, así que una estimación inválida nunca deja una Task creada.
   - El mismo dominio (`≤ 1000`) se aplica a las líneas de `## Estimación por rol` (`estimate_invalid`).
1. Carpeta:
   - ordinaria en el planning home: el del estado del motor, como `validateEnsureRole`, y si no `workspace.Locate`; en `multirepo`, el repositorio de specs (hueco 14);
   - si no, la archivada, solo para `summary push`;
   - si no hay carpeta SDD y existe `odd/tasks/<c>.md`: `odd_lane_not_supported`;
   - si no: `change_not_found`.
2. Plugin activo con el puerto: el primero, por orden de declaración, que implemente `TaskTracker` y cuyo manifiesto declare `items.create`. Si no hay: `not_applicable`, salida 0, sin red ni ficheros.
3. Vínculo `kind: work_item` del plugin. Si falta: `work_item_not_linked`, cuya continuación nombra `axiom change start --from <id|url>` y `axiom change start --adopt --change <c>` (hueco 22).
4. Estilo sellado:
   - **`checkpointed`:** última decisión `approved` de la compuerta **y** digest vigente (`record.ArtifactDigest == gateArtifactDigest(...)`). Una edición de `design.md` posterior a la aprobación deja la compuerta en `pending` y el verbo responde `gate_not_approved` (hueco 22).
   - **`continuous`:** basta con que el artefacto exista.
   - **`tasks bind` (hueco 2):** no tiene precondición de estilo en ningún estilo; no exige compuerta ni artefacto, porque es el verbo de rescate (`prov:INT:9` solo la fija para `summary push`, `tasks create` y `time *`). Le bastan los pasos 0 a 3 y el roster.
5. Específicas de cada verbo (roster, estimación, vínculo de Task, `started_at`).

**Una Task creada sin vínculo (hueco 7):** si `BindTask` falla tras un `POST` con éxito, el rol se rechaza con `write_failed`, nombrando el ID creado y el reintento `axiom integrations tasks create --change <c>`. El reintento la encuentra por la marca (`found_by_mark`) sin duplicarla.

**`estimate_not_recorded` (hueco 5):**
- Lo emite `tasks bind`, también con `--replace`, cuando no hay `--estimate`, la sección de `design.md` no tiene línea del rol y no hay vínculo anterior con estimación.
- El vínculo se escribe sin `estimate_hours`, y el rol en el JSON lleva `estimate_hours: null` y `estimate_source: "none"`.
- El aviso nombra `axiom integrations tasks create --change <c> --estimate <rol>=<h>`. Por D17, con un vínculo sin estimación cualquier estimación confirmada «difiere», así que se escribe en el tracker y en el vínculo.
- `time propose` muestra `estimate_hours: null` («sin estimación») y el resumen final de `continuous` escribe «sin estimación» en esa Task.

El rol se canoniza como en `canonicalEnsureRole`.

### D15. Salida JSON versionada, avisos tipados y códigos de salida

**Elección:**
- Tres esquemas (`axiom.integrations-tasks/v1`, `axiom.integrations-summary/v1` y `axiom.integrations-time/v1`) con un sobre común `status`, `refusal{code,message,continuation}` y `warnings[{code,message,continuation?}]`.
- Salida 0 con `done` o `not_applicable`, también con avisos; 1 con `refused` o `partial`; 2 con error de uso.
- Los textos que construye `internal/cli` nombran siempre un comando `axiom …` o llevan `// refusal:by-design <forma>: <motivo>`, para el trinquete `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`.
- Las continuaciones nombradas deben existir para `TestEveryNamedReviewContinuationIsStructurallyReal`.

### D16. Configuración: formatos y rangos nuevos en el manifiesto

**Elección (RF-ADO-02):** el manifiesto (hoja) gana:
- `ConfigField.Min` y `ConfigField.Max *int`, aplicables a `int`;
- tres formatos de la lista cerrada:
  - `non_empty`: `string` no vacía tras recortar;
  - `tag_prefix`: `string` de 1 a 100 caracteres, sin `,` ni `;`, que no empieza por `axiom:` ni por `axiom-task:` (sin distinguir mayúsculas);
  - `weekday_set`: `string_list` no vacía, sin duplicados, de `mon`…`sun`; usa `Enum`.

`workspace.decodeIntegrationValue` los aplica con errores que nombran la clave y el valor o el elemento.

Claves nuevas del manifiesto de Azure DevOps:

| Clave | Tipo / formato | Valor por defecto |
|---|---|---|
| `task_state_start` | `string`, `non_empty` | `Active` |
| `task_state_apply_approved` | `string`, `non_empty` | `Tech Review` |
| `task_state_close` | `string`, `non_empty` | sin valor en el manifiesto: el adaptador usa el efectivo de `task_state_apply_approved` |
| `work_item_state_archive` | `string`, `non_empty` | sin valor por defecto (sin transición) |
| `req_tag_prefix` | `string`, `tag_prefix` | `req:` |
| `req_tag_max` | `int`, 1–100 | **50** |
| `workday_hours` | `int`, 1–24 | `8` |
| `workdays` | `string_list`, `weekday_set` | `[mon, tue, wed, thu, fri]` |

- `req_tag_max = 50` deja la mitad del límite de 100 del WI para las etiquetas ajenas y cubre con holgura un cambio grande (este declara 32 requisitos entre `ADDED` y `MODIFIED`).
- El valor derivado de `task_state_close` se resuelve en `azuredevops.lifecycleConfigFrom(cfg)`, que también da los demás valores ya tipados.

**Alternativa descartada:** validar en el adaptador. La spec exige un error de carga del workspace.

### D17. Estimación confirmada distinta con vínculo ya presente

**Elección (`prov:INT:2`, hueco 8):** con un vínculo `kind: task` ya registrado, `tasks create` resuelve la estimación confirmada del rol (`--estimate`, o si falta la sección de `design.md`).
- **Igual a `estimate_hours`:** ninguna petición ni escritura. Se conserva la idempotencia (`result: reused_link`).
- **Distinta:**
  1. `ReadTask` (rev R, `CompletedWork` C).
  2. `SetEstimate` en modo `replace`: un único `PATCH` con `test /rev = R` que fija `OriginalEstimate = e` y `RemainingWork = round2(max(0, e − C))`.
  3. Solo tras el éxito, `BindTask` con el nuevo `estimate_hours` (`result: estimate_updated`).
- **Conflicto de `rev`:** el rol se rechaza con `rev_conflict`, sin reaplicar ni tocar el vínculo, con la resolución `axiom integrations tasks create --change <c>`.
- **Campos:** si falta uno de los dos, `field_missing` sin escritura.
- **Respuesta perdida:** el `PATCH` recuperado tras un reintento se verifica igual que en D10: `rev == R + 1` y valores iguales.
- **Sin estimación confirmada** (ni bandera ni sección): el vínculo existente no se toca. Solo un rol **sin** vínculo exige estimación (`estimate_missing`).

**Alternativa descartada:** el aviso `estimate_drift` sin escritura, que proponía la primera versión de este diseño. La spec ahora pide propagar.

### D18. `tasks bind --replace`

**Elección (`prov:INT:3`, hueco 7):**
- Sin `--replace`, vincular otra Task a un rol ya vinculado da `task_already_bound` sin escritura, con la resolución `axiom integrations tasks bind --change <c> --role <rol>=<id> --replace`.
- Con `--replace`:
  1. `ValidateTask` de la Task nueva: mismos rechazos y orden (`task_not_found`, `not_a_task`, `not_child_of_work_item`).
  2. Regla de horas de `bind`: solo escribe `OriginalEstimate` y `RemainingWork` si la Task nueva no tiene `OriginalEstimate`.
  3. `ReplaceTaskLink` bajo el cerrojo:
     - el vínculo nuevo lleva `source: tasks.bind` y el `started_at` del anterior;
     - el `estimate_hours` sale de `--estimate`, si falta de `design.md` y si falta del vínculo anterior.
- Las imputaciones ya registradas se conservan con su `item_id` original. La sección del resumen final de `continuous` las lista por rol, con el `item_id` de cada imputación.
- Un fallo de validación deja intacto el vínculo existente.
- `--replace` sin vínculo previo equivale a un `bind`, y con la misma Task es idempotente.
- La Task anterior no se modifica en el tracker.

**Alternativa descartada:** editar a mano `integrations.yaml`, que no revalida la Task y es propenso a errores.

### D19. Transición del WI al archivar

**Elección (`prov:ADO:7`, hueco 12 de la revisión del design):** `Deliver` de `archive` hace lo siguiente:
1. Sin `work_item_state_archive`: éxito sin red.
2. Sin vínculo `kind: work_item` en la instantánea: éxito sin red y aviso `work_item_not_linked`.
3. `GetWorkItem(WI)`.
4. Estados **del tipo del propio WI** (`System.WorkItemType`, por ejemplo `User Story` o `Bug`), con `TypeStates(tipo)` en caché.
5. Según el estado:
   - destino no válido para ese tipo: `StateUnknownError` permanente con la lista de estados válidos, sin escritura;
   - WI ya en el destino: sin escritura;
   - WI en categoría `Completed` o `Removed` con un estado distinto del destino: sin escritura, aviso `work_item_state_kept` (no se reabre ni se mueve un WI cerrado);
   - en otro caso, `PATCH` de `System.State` con `test /rev`.
6. Si la escritura falla:
   - un conflicto de `rev` es un error transitorio que `sync` reintenta;
   - un 400 de reglas es `tracker_rejected`, permanente.
7. Nunca devuelve vínculos ni imputaciones (RF-INT-21).

### D20. Diario de avisos de `continuous` para el resumen final

**Elección (hueco 10):** en un cambio sellado `continuous`, los cinco verbos añaden cada rechazo y cada aviso a `.tracker-notes.yaml` en la raíz de la carpeta del cambio:
- esquema `axiom.integrations-tracker-notes/v1`;
- un elemento por suceso: `{at, verb, role, code, message, continuation}`;
- solo se añade; es estado local con punto, que no se publica ni casa con `.integrations-*.yaml`;
- se escribe bajo el cerrojo del cambio y en `time propose` no altera su carácter de solo lectura respecto al tracker y a `integrations.yaml` (`prov:INT:7`);
- en `checkpointed` no se escribe.

`sdd-archive` lee `.tracker-notes.yaml` y el `integrations.yaml` **antes de mover la carpeta** (paso previo al paso 3 de la skill). Con ellos compone `## Seguimiento en el tracker`, sin red, y muestra cada suceso con su comando de resolución.

Un suceso de una ejecución posterior resuelto deja de ser pendiente cuando el diario tiene una entrada posterior, del mismo verbo y rol, con `code: done`. Ese `done` lo añaden los verbos en `continuous` cuando terminan bien.

**Alternativa descartada:** confiar en el contexto de la conversación del orquestador, que se pierde entre sesiones y no es comprobable.

**Prueba:**
- la CLI comprueba que el diario se escribe solo en `continuous`, que es solo de añadir y que no se publica;
- la doctrina comprueba que `sdd-archive` nombra `.tracker-notes.yaml` antes del movimiento.

### D21. Cierre sin horas en `continuous`: `time record --close-only`

**Elección (hueco 3):** la transición a `task_state_close` en `continuous` ocurre siempre al cerrar el rol, haya o no horas.
- **En `time record`:** tras una imputación con éxito, tras `time_record_recovered` y tras `--resolve-pending applied`.
- **Bandera nueva `--close-only`:**
  - solo en `continuous`; en `checkpointed` es error de uso, porque el estado lo aplica `role.apply.approved`;
  - incompatible con `--hours` y con `--resolve-pending`;
  - transiciona sin imputar, con la regla `AdvanceUnlessClosed` y sin `Assignee`;
  - exige el vínculo del rol (`task_not_bound`), pero no `started_at`.
- **Doctrina** (`sdd-apply` y el orquestador):
  - propuesta > 0 h → `time record --hours <propuesta>`;
  - propuesta de 0 h → `time record --close-only`, informado como «sin imputación»;
  - `time propose` fallido por cualquier motivo distinto de `task_not_bound` (por ejemplo `role_not_started` o la red) → `time record --close-only`, y el fallo queda en el diario de D20;
  - `time propose` con `task_not_bound` → nada que cerrar.
- **Spec:** ajuste mínimo en `prov:INT:8` (bandera, transición tras recuperar y escenario); no cambia ningún ID.

**Alternativa descartada:** que la doctrina pidiera `time record --hours 0`, que la spec rechaza como error de uso.

## Flujo de datos

```
checkpointed: gate record ─► (outbox) role.apply.approved ─► Deliver ─► PATCH estado
continuous / ambos: doctrina ─► axiom integrations <verbo> ─► trackerTarget (D14)
                                                │
             ┌──────────────────────────────────┼─────────────────────────────┐
        tasks create|bind                  summary push                 time propose|record
             │                                  │                              │
   trackersync.ParseEstimates      trackersync.SpecModel ─► specsummary    trackersync.ProposeHours
             │                                  │                              │
   WithTaskTracker ─► EnsureChildTask /   WithTaskTracker ─► UpdateSummary   .time-pending.yaml (D9)
                      ValidateTask/SetEstimate           (1 PATCH test /rev)     │
             │                                                          ReadTask ─► RecordHours
   integrations.BindTask ─► integrations.yaml v2                               │
                                                             RecordTimeEntry ─► integrations.yaml v2
                                                             (continuous) TransitionTask(close)

branch ensure --role ─► validateEnsureRole (+ task_not_bound) ─► motor ─► ready
                     ─► MarkRoleStarted (started_at) ─► dispatch role.start ─► Deliver (Active + asignación)
sdd publish --archive ─► dispatch archive ─► Deliver (WI → work_item_state_archive)
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/integrations/links.go` | Modificar | Esquema v2 con lectura de v1; campos nuevos; fusión con actualización de estimación y `started_at` de escritura única; `BindTask`, `ReplaceTaskLink`, `MarkRoleStarted`, `RecordTimeEntry`, `TaskLinks` y `TimeEntries` |
| `internal/integrations/plugin.go` | Modificar | `Link.EstimateHours/StartedAt`, `TimeEntry.Actor`, `Outcome.Warnings`, `Delivery.Deliverer/Identities`; se retira el comentario «nadie la produce aún» |
| `internal/integrations/tasks.go` | Crear | `PortTasks`, tipos neutrales, `TaskTracker`, `WithTaskTracker` y errores neutrales |
| `internal/integrations/pending.go` | Crear | Diario `.time-pending.yaml`: `PendingTime`, `PutPendingTime`, `PendingTimeFor`, `ClearPendingTime` (bajo cerrojo) |
| `internal/integrations/notes.go` | Crear | Diario de avisos `.tracker-notes.yaml` de `continuous` (D20): `AppendTrackerNote` bajo cerrojo y `TrackerNotes` en lectura |
| `internal/integrations/dispatcher.go` | Modificar | Campos `Deliverer` e `Identities`, que pasa a `Delivery`; propaga `Outcome.Warnings` saneados |
| `internal/integrations/ports.go` | Modificar | `portLabel` de `PortTasks` |
| `internal/integrations/integrationstest/fake.go` | Modificar | Fake de `TaskTracker` programable (hijas, creación, lectura, horas, conflicto, transición y resumen) |
| `internal/integrations/manifest/manifest.go`, `format.go` | Modificar | `Min/Max`, formatos `non_empty`, `tag_prefix` y `weekday_set` con su validación |
| `internal/integrations/manifest/azuredevops.go` | Modificar | Capacidades, eventos y claves nuevas (D16); comentario actualizado |
| `internal/workspace/integrations.go` | Modificar | Aplicación de rangos y formatos nuevos |
| `internal/integrations/azuredevops/tasks.go` | Crear | `TaskTracker`: hijas, WIQL, creación y reparación, validación, estimación, horas y transición |
| `internal/integrations/azuredevops/summary.go` | Crear | `UpdateSummary` (descripción y etiquetas en un `PATCH`) |
| `internal/integrations/azuredevops/lifecycle.go` | Crear | `lifecycleConfigFrom`, `Deliver` por evento y reglas de avance y asignación |
| `internal/integrations/azuredevops/plugin.go` | Modificar | `Deliver` delega en `lifecycle.go`; `var _ integrations.TaskTracker` |
| `internal/integrations/azuredevops/workitems.go`, `client.go`, `workitem_port.go` | Modificar | `CreateWorkItem` (`POST`, `retryable: false`), `QueryWIQL` (`retryable: true`), `TypeFields` y `TypeStates` con caché, constantes de campos de horas, `AreaPath`, `IterationPath` y `AssignedTo`; se retira «ningún evento invoca esta operación» |
| `internal/integrations/azuredevops/errors.go` | Modificar | Se retira `ErrLifecycleUnsupported` de la ruta de los tres eventos; `change.start` sigue siendo permanente |
| `internal/integrations/azuredevops/adotest/workitems.go`, `states.go`, `tasks.go` (nuevo) | Modificar/Crear | `POST $Task` (asigna id y rev 1, aplica relaciones), `LoseNextCreateResponse`, `RejectCreateRelations`, WIQL por etiqueta, `SeedTypeFields`, `PATCH` de campos no sembrados de una lista blanca (horas, estado, `AssignedTo`, descripción, etiquetas, relaciones) y `SeedMembers` |
| `internal/specsummary/render.go` | Crear | Renderizador hoja (D12) |
| `internal/trackersync/estimate.go`, `clock.go`, `specmodel.go`, `source.go` | Crear | Estimación, reloj, modelo de la spec e IDs, `role.close/<rol>/<n>` |
| `internal/branch/role_refusal.go`, códigos | Modificar | `CodeTaskNotBound`, `TaskNotBound(...)` y `IntegrationsLinksInvalid(...)` |
| `internal/cli/branch_role.go` | Modificar | Barrera D5 y `recordEnsureRoleStarted` (D6) |
| `internal/cli/branch.go` | Modificar | Llamada a `recordEnsureRoleStarted` antes de `emitEnsureRoleStart` |
| `internal/cli/integrations.go` | Modificar | Despacho y ayuda de `tasks`, `summary` y `time` (RF-INT-18) |
| `internal/cli/integrations_events.go` | Modificar | `newIntegrationDispatcher` rellena `Deliverer` e `Identities` |
| `internal/cli/integrations_lifecycle.go` | Crear | `trackerTarget` (D14), sobre JSON, catálogo de rechazos y avisos |
| `internal/cli/integrations_tasks.go`, `integrations_summary.go`, `integrations_time.go` | Crear | Los cinco verbos |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Párrafo «Tracker writes» tras «Work item of origin» |
| `internal/components/agentguidance/routing.go` | Modificar | Frase ODD: nunca se invocan los verbos `tasks`, `summary` ni `time` |
| `internal/assets/skills/sdd-design/SKILL.md` | Modificar | Sección `## Estimación por rol` condicionada al tracker activo |
| `internal/assets/skills/sdd-apply/SKILL.md` | Modificar | Cierre del rol: `time propose` y `time record` por estilo |
| `internal/assets/skills/sdd-archive/SKILL.md` | Modificar | `summary push` sobre la carpeta archivada antes de publicar; sección `## Seguimiento en el tracker` en `continuous` |
| copias espejo (`.claude/skills/...` y demás) y `testdata/golden/*` | Regenerar | Paridad y goldens |
| `axiom.example.yaml`, `docs/manual-de-inicio.md` | Modificar | Claves nuevas con valores por defecto; flujo de los verbos |

## Interfaces y contratos

### Puerto de Tasks (`internal/integrations/tasks.go`)

```go
const PortTasks = "tasks"

// TaskHours son las horas de una Task; un campo ausente en el elemento se lee como 0.
type TaskHours struct {
	Original, Remaining, Completed float64
	HasOriginal                    bool
}

// TaskItem es la lectura neutral de una Task. AssignedToID es solo el id del tracker: nunca el
// correo ni el uniqueName.
type TaskItem struct {
	ID, URL, Type, State, StateCategory, ParentID string
	AssignedToID                                  string
	Rev                                           int
	Tags                                          []string
	Hours                                         TaskHours
}

type TaskOutcome string

const (
	TaskCreated     TaskOutcome = "created"
	TaskFoundByMark TaskOutcome = "found_by_mark"
	TaskRepaired    TaskOutcome = "repaired"
)

// Warning es un aviso tipado de un puerto: no detiene la operación.
type Warning struct{ Code, Message, Continuation string }

type ChildTasksRequest struct {
	PortRequest
	ParentID string
}
type CreateTaskRequest struct {
	PortRequest
	ParentID, Title, Mark string
	EstimateHours         float64 // 0: sin estimación
}
type CreateTaskResult struct {
	Task     TaskItem
	Outcome  TaskOutcome
	Warnings []Warning
}
type TaskRequest struct {
	PortRequest
	TaskID, ParentID string // ParentID solo lo usa ValidateTask
}
type EstimateMode string

const (
	EstimateIfMissing EstimateMode = "if_missing" // bind: solo si la Task no tiene OriginalEstimate
	EstimateReplace   EstimateMode = "replace"    // create con estimación distinta (D17)
)

type EstimateRequest struct {
	PortRequest
	TaskID string
	Hours  float64
	Mode   EstimateMode
	Base   TaskItem // lectura previa (Rev y Completed); obligatoria con EstimateReplace
}
type EstimateResult struct {
	Task      TaskItem
	Written   bool // false en if_missing si la Task ya tenía OriginalEstimate
	Recovered bool // respuesta perdida verificada, como en D10
}
type HoursRequest struct {
	PortRequest
	Base  TaskItem // lectura previa: Rev y Hours
	Hours float64
}
type HoursResult struct {
	Task      TaskItem
	Recovered bool // respuesta perdida verificada (D10)
}

type TransitionRule string

const (
	AdvanceFromProposed TransitionRule = "advance_from_proposed" // role.start
	AdvanceUnlessClosed TransitionRule = "advance_unless_closed" // apply aprobado y cierre
)

type TransitionRequest struct {
	PortRequest
	TaskID, State string
	Rule          TransitionRule
	// Assignee es la identidad a la que asignar; nil = no asignar. Deliver de role.start la pasa
	// solo si el actor del evento es quien entrega y hay identidad guardada (D13).
	Assignee *TrackerIdentity
}
type TransitionResult struct {
	Task         TaskItem
	StateChanged bool
	Assigned     bool
	// Avisos: task_state_kept, assignment_ambiguous, assignment_mismatch, task_assigned_to_other.
	Warnings []Warning
}
type SummaryRequest struct {
	PortRequest
	ItemID, HTML, TagPrefix string
	Tags                    []string // IDs ya con prefijo, en orden estable
	TagMax                  int
}
type SummaryResult struct {
	Wrote, DescriptionChanged, TagsSent bool
	Added, Removed                      []string
	Warnings                            []Warning
}

type TaskTracker interface {
	ChildTasks(ctx context.Context, req ChildTasksRequest) ([]TaskItem, error)
	EnsureChildTask(ctx context.Context, req CreateTaskRequest) (CreateTaskResult, error)
	ReadTask(ctx context.Context, req TaskRequest) (TaskItem, error)
	ValidateTask(ctx context.Context, req TaskRequest) (TaskItem, error)
	SetEstimate(ctx context.Context, req EstimateRequest) (EstimateResult, error)
	RecordHours(ctx context.Context, req HoursRequest) (HoursResult, error)
	TransitionTask(ctx context.Context, req TransitionRequest) (TransitionResult, error)
	UpdateSummary(ctx context.Context, req SummaryRequest) (SummaryResult, error)
}

// WithTaskTracker detecta el puerto (sin él: *PortUnsupportedError{Port: PortTasks}, sin red),
// compone la PortRequest con secretos acotados y rastreados, y ejecuta call bajo invokePort.
func WithTaskTracker[T any](p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store,
	call func(t TaskTracker, req PortRequest) (T, error)) (T, error)

// Errores neutrales (errors.Is / errors.As a través de *PortError).
var (
	ErrTaskNotFound        = errors.New("task_not_found")
	ErrNotATask            = errors.New("not_a_task")
	ErrNotChildOfWorkItem  = errors.New("not_child_of_work_item")
	ErrTaskAmbiguous       = errors.New("task_ambiguous")
	ErrTaskParentLinkFailed = errors.New("task_parent_link_failed")
	ErrFieldMissing        = errors.New("field_missing")
	ErrStateUnknown        = errors.New("state_unknown")
	ErrRevConflict         = errors.New("rev_conflict")
	ErrTrackerRejected     = errors.New("tracker_rejected")
	ErrTaskNotBound        = errors.New("task_not_bound")
)

type TaskAmbiguousError struct{ Mark string; IDs []string }
type FieldMissingError struct{ Field string }
type StateUnknownError struct{ State string; Valid []string }
type RevConflictError struct{ Rev int; Hours TaskHours }
```

Las cadenas de los centinelas son códigos y no frases. El texto que lee la persona lo compone la CLI con su continuación (D15), de modo que el núcleo no queda dentro del trinquete de rechazos.

### Entrega (`plugin.go` y `dispatcher.go`)

```go
type IdentityLookup interface {
	Lookup(plugin, account string) (TrackerIdentity, bool, error)
}
type Delivery struct {
	Event      Event
	Config     map[string]any
	Links      []Link
	Secrets    SecretSource
	Deliverer  string         // identidad git de quien entrega (nuevo)
	Identities IdentityLookup // nil: sin identidades (nuevo)
}
type Outcome struct {
	Links       []Link
	TimeEntries []TimeEntry
	Warnings    []string // nuevo; saneados y añadidos a Report.Warnings
}
```

### `integrations.yaml` v2

```yaml
schema: axiom.integrations/v2
change: x
links:
  - plugin: azure-devops
    kind: work_item
    id: "100"
    url: https://dev.azure.com/org/proj/_workitems/edit/100
    source: change.start
    recorded_at: 2026-11-26T10:00:00Z
  - plugin: azure-devops
    kind: task
    id: "205"
    url: https://dev.azure.com/org/proj/_workitems/edit/205
    role: fullstack
    source: tasks.create
    estimate_hours: 6
    started_at: 2026-11-26T11:00:00Z
    recorded_at: 2026-11-26T10:30:00Z
time_entries:
  - plugin: azure-devops
    role: fullstack
    item_id: "205"
    minutes: 120
    date: "2026-11-27"
    actor: Ana <ana@x.test>
    source: role.close/fullstack/1
    recorded_at: 2026-11-27T16:00:00Z
```

**Migración de lectura:** `v1` se acepta sin campos nuevos (los nuevos son opcionales y una fusión sin cambios no reescribe). La primera fusión que cambia algo lo escribe como `v2`. Bajar a `v1` (reversión) consiste en quitar los vínculos `kind: task`, los campos `estimate_hours`, `started_at` y `actor`, y poner `schema: axiom.integrations/v1`.

### Diario `.time-pending.yaml` (local, no publicado)

```yaml
schema: axiom.integrations-time-pending/v1
change: x
pending:
  - role: fullstack
    plugin: azure-devops
    item_id: "205"
    hours: 2
    minutes: 120
    date: "2026-11-27"
    actor: Ana <ana@x.test>
    source: role.close/fullstack/1
    base_rev: 4
    expected_completed: 5
    expected_remaining: 3
    created_at: 2026-11-27T16:00:00Z
```

### Verbos (CLI)

```
axiom integrations tasks create --change <c> [--estimate <rol>=<h>]... [--cwd <ruta>] [--json]
axiom integrations tasks bind   --change <c> --role <rol>=<id> [--estimate <rol>=<h>] [--replace] [--cwd <ruta>] [--json]
axiom integrations summary push --change <c> [--cwd <ruta>] [--json]
axiom integrations time propose --change <c> --role <rol> [--cwd <ruta>] [--json]
axiom integrations time record  --change <c> --role <rol> (--hours <h> | --resolve-pending applied|discarded | --close-only) [--cwd <ruta>] [--json]
```

### Esquemas JSON

`axiom.integrations-tasks/v1`:

```json
{
  "schema": "axiom.integrations-tasks/v1",
  "verb": "create",
  "change": "x",
  "plugin": "azure-devops",
  "status": "done",
  "work_item": {"id": "100", "url": "https://…/100"},
  "roles": [
    {"role": "fullstack", "result": "created",
     "task": {"id": "205", "url": "https://…/205"},
     "estimate_hours": 6, "estimate_source": "design",
     "refusal": null}
  ],
  "refusal": null,
  "warnings": []
}
```

- `result`: `created`, `found_by_mark`, `repaired`, `reused_link`, `estimate_updated`, `bound`, `replaced`, `unchanged` o `refused`.
- `estimate_source`: `flag`, `design`, `link` o `none`.
- `status`: `partial` cuando unos roles terminan y otros no.

`axiom.integrations-summary/v1`:

```json
{
  "schema": "axiom.integrations-summary/v1",
  "change": "x", "plugin": "azure-devops", "status": "done",
  "folder": "ordinary",
  "work_item": {"id": "100", "url": "https://…/100"},
  "description": {"changed": true, "bytes": 1834, "sha256": "…"},
  "tags": {"prefix": "req:", "desired": ["req:prov:ADO:1"], "added": ["req:prov:ADO:1"], "removed": [], "sent": true},
  "wrote": true,
  "refusal": null, "warnings": []
}
```

- `folder`: `ordinary` o `archived`.

`axiom.integrations-time/v1`:

```json
{
  "schema": "axiom.integrations-time/v1",
  "verb": "propose",
  "change": "x", "plugin": "azure-devops", "status": "done", "role": "fullstack",
  "task": {"id": "205", "url": "https://…/205"},
  "estimate_hours": 6,
  "proposal": {"hours": 11, "minutes": 660, "since": "…", "until": "…", "timezone": "Europe/Madrid +01:00",
               "anchor": "09:00", "workday_hours": 8, "workdays": ["mon","tue","wed","thu","fri"], "rounding": "quarter_hour"},
  "tracker": {"rev": 4, "original": 8, "remaining": 5, "completed": 3},
  "recorded": null,
  "transition": null,
  "refusal": null, "warnings": []
}
```

- `recorded` (en `record`): `{hours, minutes, date, actor, source, recovered}`; `null` con `--close-only`.
- `transition` (en `continuous`): `{state, changed}`.

### Catálogo de rechazos y avisos

Todos con continuación `axiom …` o marca `refusal:by-design`.

| Código | Verbos | Continuación |
|---|---|---|
| `usage_invalid` (salida 2) | todos | `axiom integrations <familia> <verbo> --help` |
| `change_not_found` | todos | `axiom sdd status` |
| `odd_lane_not_supported` | todos | `// refusal:by-design human-authority:` el carril ODD no crea Tasks ni horas por decisión de producto |
| `work_item_not_linked` | todos | `axiom change start --from <id|url>` y `axiom change start --adopt --change <c>` |
| `gate_not_approved` | create, summary, time | `axiom sdd gate show --change <c>` |
| `artifact_missing` | create, summary | `axiom sdd status --change <c>` |
| `estimate_missing` / `estimate_invalid` | create, bind | `axiom integrations tasks create --change <c> --estimate <rol>=<h>` |
| `role_not_in_roster` | bind, time | `axiom integrations tasks bind --change <c> --role <rol>=<id>` con un rol del roster listado |
| `task_ambiguous` | create | `axiom integrations tasks bind --change <c> --role <rol>=<id>` |
| `task_parent_link_failed` | create | `axiom integrations tasks create --change <c>` (la reparación por WIQL) |
| `task_already_bound` | bind | `axiom integrations tasks bind --change <c> --role <rol>=<id> --replace` (D18) |
| `task_not_found` / `not_a_task` / `not_child_of_work_item` | bind, time | `axiom integrations tasks bind --change <c> --role <rol>=<id>` |
| `task_not_bound` | time, ensure | `axiom integrations tasks create --change <c>` y `axiom integrations tasks bind --change <c> --role <rol>=<id>` |
| `role_not_started` | time | `axiom branch ensure --change <c> --role <rol>` |
| `rev_conflict` | time record | `axiom integrations time propose --change <c> --role <rol>` |
| `rev_conflict` (por rol) | tasks create | `axiom integrations tasks create --change <c>` (D17) |
| `time_pending_ambiguous` | time | `axiom integrations time record --change <c> --role <rol> --resolve-pending applied|discarded` |
| `provisional_ids_unresolved` | summary | `axiom sdd archive-check --cwd <raíz> --change <c> --json` (verbo existente, `cli/sdd_archive_check.go`; su salida lista `assignments` de `ids.yaml` y `pendingProvisionals`), validado por `TestEveryNamedReviewContinuationIsStructurallyReal` |
| `field_missing` / `state_unknown` | create, bind, time | `axiom integrations discover --plugin azure-devops` |
| `tracker_unavailable` / `tracker_rejected` | todos | el mismo verbo repetido (`axiom integrations … --change <c>`) |
| `write_failed` | create, bind, time | el mismo verbo repetido |
| `integrations_links_invalid` | ensure | `axiom integrations sync --change <c>` |
| `port_unsupported` | todos | `// refusal:by-design world-action:` hace falta una versión de axiom cuyo plugin ofrezca el puerto |

Avisos (salida 0):
- `req_tags_over_limit`, `req_tag_invalid`;
- `estimate_not_recorded`;
- `task_duplicated`;
- `state_transition_failed`, `time_record_recovered`, `time_pending`;
- `started_at_not_recorded`;
- `assignment_skipped`, `assignment_ambiguous`, `assignment_mismatch`;
- `task_state_kept`, `task_assigned_to_other`;
- `work_item_not_linked` (aviso de `Deliver` de `archive`), `work_item_state_kept`.

### Estimación (`trackersync.ParseEstimates`)

- **Sección:** el título exacto `## Estimación por rol`, fuera de vallas, hasta el siguiente `## `.
- **Líneas de datos:** toda línea que empieza por `- ` es de datos y debe casar con `^- ([a-z][a-z0-9-]*): ([0-9]+(\.[0-9]{1,2})?) h$`. El resto del texto (la justificación) se ignora.
- **Rechazo `estimate_invalid`, nombrando la línea:** valor no positivo, rol repetido o rol fuera del roster.
- **Sin sección:** no es error; solo falta la estimación del rol (`estimate_missing`).

## Estrategia de pruebas

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad, núcleo | v1 → lectura sin reescritura; v2 escrito al cambiar; esquema `v9` rechazado nombrándolo; `estimate_hours` actualizado; `started_at` de escritura única; `ReplaceTaskLink` conserva `started_at` e imputaciones y retira solo el vínculo del rol; dos `role.close/*` conviven; `actor` obligatorio en `role.close/*`; `Outcome.Warnings` propagados; `WithTaskTracker` (sin puerto, pánico, saneado del secreto); diario pendiente bajo cerrojo; diario de avisos `.tracker-notes.yaml` solo de añadir (D20) | `go test ./internal/integrations/` |
| Unidad, manifiesto y workspace | rangos y formatos; valores por defecto; `axiom:` en el prefijo rechazado; `req_tag_max: 101`; `workdays` repetidos o desconocidos; catálogo idéntico al manifiesto. Escenario «Valores por defecto de las claves nuevas» de RF-ADO-02 (hueco 9): una única prueba con nombre propio, `TestADOLifecycleConfigDefaults`, en `internal/integrations/azuredevops/`. Carga un bloque sin claves nuevas con `workspace.Locate` y comprueba con `lifecycleConfigFrom` `Active`, `Tech Review` en apply y en cierre, `req:`, 50, 8 h, lunes a viernes y ningún estado de archivado | `go test ./internal/integrations/manifest/ ./internal/workspace/ -run 'Integration|Manifest|Format'`; `go test ./internal/integrations/azuredevops/ -run TestADOLifecycleConfigDefaults` |
| Unidad, renderizador | estructura, lista blanca sin atributos, escape, determinismo byte a byte, marcado sin cerrar; **orden del fichero** con una delta que mezcla `MODIFIED` antes de `ADDED` y un `REMOVED` en medio; `REMOVED` y `RENAMED` con `h3` + `p` y su marcador en `strong`; ningún `ul` ni `li`; tope de 64 KiB | `go test ./internal/specsummary/` (tabla y fuzz corto del escape) |
| Unidad, `trackersync` | los tres escenarios de reloj de la spec, segundo cierre desde la última imputación, cambio de hora (`Europe/Madrid`, último domingo de marzo y de octubre), ancla nocturna que cruza la medianoche, cuartos de hora; estimación (decimal, malformada, repetida, fuera del roster, en valla); `SpecModel` con `prov:` y con `ids.yaml` | `go test ./internal/trackersync/` con `time.FixedZone` y `time.LoadLocation` |
| Adaptador (doble) | creación con marca, relación, horas, `AreaPath` e `IterationPath`; reutilización por marca; `POST` con respuesta perdida sin reintento; dos marcas → `task_ambiguous`; el título no decide; `Removed` ignorada; huérfana reparada por WIQL; WIQL caído → sin `POST`; 400 en `POST` → camino alternativo; campo de horas ausente → sin `POST`; `ValidateTask` (404, tipo, otro padre, sin padre); horas con `test /rev`, piso en cero, conflicto con valores, `AfterRetry` recuperado; `RecordHours` con `CompletedWork` ausente en el proceso → `FieldMissingError` sin escritura, y ausente en el valor → leído como 0 (hueco 4); `TransportError` sin respuesta devuelto como incierto (hueco 13); `SetEstimate` `if_missing` (con y sin `OriginalEstimate`) y `replace` (`RemainingWork = max(0, e − C)`, conflicto sin reaplicar); resumen (sin cambios no escribe, una sola escritura, ajenas conservadas, tope que retira las `prov:`, prefijo configurado, 3 intentos); transiciones (Active, Tech Review, repetida sin escritura, estado inexistente lista los válidos, 400 de reglas → `tracker_rejected`, reentrada en `Tech Review` → `task_state_kept`); asignación (actor igual, distinto, sin identidad, homónimos, otro asignado, `id` devuelto distinto, Task ya `Active` sin asignar → solo `AssignedTo`, `Assignee` nil sin asignación); `Deliver` (`change.start` y `gate.spec.approved` sin red, `task_not_bound` permanente, `archive` sin vínculo de WI → aviso sin red); `archive` (D19: sin estado configurado, WI ya en el destino sin escritura, estado inexistente para el tipo del WI con la lista de válidos, WI `Completed` → `work_item_state_kept`, estados leídos del tipo `User Story` y del tipo `Bug`, conflicto transitorio); el PAT nunca aparece | `go test ./internal/integrations/azuredevops/...` con `adotest` (hermético: solo hosts locales) |
| CLI | los cinco verbos: rutas de la spec, orden de precondiciones, `not_applicable` sin ficheros, ODD rechazado sin ficheros, `gate_not_approved` con digest caducado, JSON por esquema, salidas 0, 1 y 2, diario pendiente (no aplicado, recuperado, ambiguo, `--resolve-pending applied|discarded`, incompatibilidad con `--hours`, sin publicar); `tasks create` con estimación igual (sin escritura) y distinta (D17, con `rev_conflict` por rol); `tasks bind --replace` (sustitución, validación fallida sin cambios, idempotente, sin vínculo previo), y `task_already_bound` nombrando `--replace`; `time *` sin vínculo de Task en `continuous` → `task_not_bound`; diario: `time propose` con intención solo avisa `time_pending` sin tocar ficheros, fallo de transporte conserva la intención, reconciliación sobre el `item_id` de la intención tras `--replace`; `--close-only` (transiciona sin imputar en `continuous`, error de uso en `checkpointed` y con `--hours`); transición de cierre también tras `time_record_recovered` y `--resolve-pending applied`; `tasks create` `partial` con un rol fallido sin impedir el otro y `gates.yaml` intacto; dominio de `--estimate` (0, negativo, tres decimales, 1001, rol repetido) como error de uso sin ninguna petición; `BindTask` fallido tras el `POST` → `write_failed` y el reintento encuentra la Task por la marca; `tasks bind` sin compuerta en `checkpointed` con `design` pendiente; `estimate_not_recorded` con `estimate_hours: null`; `.tracker-notes.yaml` solo en `continuous`; `ensure --role` con barrera, roster antes que la barrera, sin WI ni plugin no cambia nada, `started_at` en `ready`, sin `started_at` con `decision_required`, sin `--role` (fichero intacto) y desde la ruta del dashboard que llama al motor (fichero intacto) (hueco 6), fallo de escritura → aviso; `gate record` sin cambios con el bloque activo; ayuda | `go test ./internal/cli/ -run 'TestIntegrationsTasks|TestIntegrationsSummary|TestIntegrationsTime|TestEnsureRole|TestIntegrationsHelp|TestGateRecordADO'` |
| Canario de secretos | rutas REST nuevas (`POST $Task`, `wiql`, `fields`, `states`, `PATCH`) en el camino feliz, 401, 403, 429 y 500, con los cinco verbos y con y sin `--json` | `go test ./internal/cli/ -run 'TestADOCanary|TestIntegrationsCanary'` |
| Trinquetes | rechazos nuevos con continuación; código alcanzable desde `cmd/axiom`; residuos de IDs heredados | `go test ./internal/cli/ -run 'TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign|TestEveryNamedReviewContinuationIsStructurallyReal'`; `scripts/deadcode-ratchet.sh` en CI; `go test ./internal/idmigration/ -run TestRepositoryLegacyIDResidues` |
| Doctrina | párrafo en todos los runtimes, ODD sin verbos, paridad de copias, goldens; `--close-only` para 0 h y para un `time propose` fallido; `sdd-archive` lee `.tracker-notes.yaml` antes de mover la carpeta. **Sin secretos ni rutas del tracker (hueco 16):** `TestTrackerDoctrineNamesNoSecretsOrTrackerRoutes` recorre el texto proyectado a cada runtime y las tres skills, y falla si aparece `_apis/`, `dev.azure.com`, `workitems/`, `Authorization`, `Basic `, `pat=`, el valor del canario de PAT o el `uniqueName` de un fixture | `go test ./internal/components/sdd/ ./internal/components/agentguidance/ ./internal/assets/ -run 'Tracker|WorkItem|Parity|Shared'`; `go test ./internal/components/ -run TestGolden` (con `-update` solo para regenerar) |

**Política:** solo los paquetes tocados en cada unidad y, en `internal/cli`, siempre con `-run`. Nunca la batería completa en local: la pasa CI. Ninguna prueba llama a Azure DevOps real. Los fixtures solo usan IDs `prov:` o `RF-…`, nunca el prefijo heredado (si hiciera falta, se declara en `internal/idmigration/residues.yaml`).

## Matriz de amenazas

El cambio no introduce enrutado de shell, subprocesos nuevos, automatización de git ni de PR, ni clasificación de ejecutables. La integración con el proceso del tracker es HTTP mediante el cliente existente (RF-ADO-03 a RF-ADO-05).

| Frontera | Aplicabilidad | Respuesta |
|---|---|---|
| Rutas con apariencia de documentación | N/A: no se clasifican ni ejecutan ficheros | — |
| Selección de repositorio git | N/A: `ensure --role` solo añade una lectura de `integrations.yaml` en el planning home ya resuelto; no hay `git -C` nuevo | — |
| Estado de commit | N/A: los verbos no hacen commit | — |
| Estado de push | N/A: ningún push | — |
| Órdenes de PR | N/A | — |

Riesgos propios del tracker, con pruebas en la tabla anterior:
- el PAT (canario ampliado);
- HTML inyectado desde la spec (escape total y lista blanca);
- URL con credenciales en vínculos (`validateLink`, ya vigente);
- asignación a otra persona (D13).

## Migración y despliegue

- **Lectura:** los binarios nuevos leen `integrations.yaml` v1 y v2; un `v1` sin cambios no se reescribe.
- **Escritura:** toda fusión con cambios escribe `v2`. Un binario anterior rechaza el `v2` con un error que nombra el esquema.
- **Desactivación:** `integrations.azure-devops.enabled: false` apaga verbos, eventos y barrera.
- **Reversión:**
  - `git revert` del merge;
  - bajar los `integrations.yaml` escritos a `v1` (ver «Migración de lectura»);
  - `axiom integrations sync --discard <clave>` para las operaciones de `role.start`, `role.apply.approved` y `archive` que queden en las colas.
  - Lo escrito en Azure DevOps no se revierte.
- **Spec viva:** al integrar, hay que reescribir el `## Purpose` de `azure-devops-adapter` (lo pide la nota de su delta).

## Unidades de trabajo (un commit por unidad, en este orden)

| Unidad | Contenido (código, pruebas y docs juntos) | Pruebas de la unidad |
|---|---|---|
| U1 Núcleo de vínculos v2 | D2 y D3; `Outcome.Warnings` propagado; `Delivery/Dispatcher.Deliverer/Identities`; diario pendiente (D9, parte de núcleo) | `go test ./internal/integrations/` |
| U2 Puerto de Tasks en el núcleo y fake | `tasks.go`, `WithTaskTracker`, errores neutrales, `portLabel`; fake en `integrationstest` | `go test ./internal/integrations/ ./internal/integrations/integrationstest/` |
| U3 Configuración | D16 (manifiesto, workspace), claves y capacidades del manifiesto de Azure DevOps (sin eventos nuevos todavía), `axiom.example.yaml` | `go test ./internal/integrations/manifest/ ./internal/workspace/ -run 'Integration|Manifest|Format'` |
| U4 Adaptador: Tasks y horas | D10 y D11; `ValidateTask`, `SetEstimate` (`if_missing` y `replace`), `ReadTask`, `RecordHours`; `adotest` ampliado; `lifecycleConfigFrom` | `go test ./internal/integrations/azuredevops/...` |
| U5 Adaptador: resumen, transiciones y `Deliver` | D12 (`UpdateSummary`), D13 (`TransitionTask` y `Deliver` de los tres eventos), eventos del manifiesto; `newIntegrationDispatcher` con `Deliverer` e `Identities` | `go test ./internal/integrations/azuredevops/... ./internal/integrations/manifest/`; `go test ./internal/cli/ -run 'TestGateRecordADO|TestIntegrationsSync'` |
| U6 Renderizador y modelo de spec | `internal/specsummary`; `trackersync.SpecModel` | `go test ./internal/specsummary/ ./internal/trackersync/` |
| U7 Verbos `tasks create|bind` | `integrations_lifecycle.go` (D14 y D15), `integrations_tasks.go` (incluidas D17 y D18, con `--replace`), `trackersync.ParseEstimates`, ayuda | `go test ./internal/trackersync/`; `go test ./internal/cli/ -run 'TestIntegrationsTasks|TestIntegrationsHelp|TestEveryProductionRefusal'` |
| U8 Verbo `summary push` | `integrations_summary.go` (carpeta ordinaria y archivada) | `go test ./internal/cli/ -run 'TestIntegrationsSummary|TestEveryProductionRefusal'` |
| U9 Verbos `time propose|record` | `integrations_time.go`, `trackersync.ProposeHours`, `source.go`, uso del diario | `go test ./internal/trackersync/`; `go test ./internal/cli/ -run 'TestIntegrationsTime|TestEveryProductionRefusal'` |
| U10 Barrera de `ensure --role` y `started_at` | D5 y D6 | `go test ./internal/branch/ -run Role`; `go test ./internal/cli/ -run 'TestEnsureRole|TestBranchEnsure.*Role'` |
| U11 Canario de secretos | ampliación de los canarios a los verbos y rutas nuevas | `go test ./internal/cli/ -run 'TestADOCanary|TestIntegrationsCanary'` |
| U12 Doctrina, skills y goldens | párrafo «Tracker writes» del orquestador, frase ODD en `routing.go`, `sdd-design`, `sdd-apply` y `sdd-archive`, copias espejo, goldens y prueba de regla | `go test ./internal/components/sdd/ ./internal/components/agentguidance/ ./internal/assets/ -run 'Tracker|WorkItem|Parity|Shared'`; `go test ./internal/components/ -run TestGolden` |
| U13 Documentación | `docs/manual-de-inicio.md` (flujo y verbos), revisión final de `axiom.example.yaml` | lectura de comprobación |

El trinquete de código muerto se mide en CI sobre la cabeza de la PR. U1 y U2 dejan funciones exportadas que alcanzan U4 y U7. Ninguna queda sin consumidor al final.

### Contenido de la doctrina (U12)

- **Orquestador (`sdd-orchestrator-sections.md`, párrafo «Tracker writes», en inglés como el resto del fichero):**
  - **Condición:** el cambio tiene un vínculo `kind: work_item` y `axiom change start --cwd <root> --json` no responde `not_applicable`.
  - **Momentos:**
    - `spec` → `summary push`;
    - `design` → `tasks create`;
    - antes de cada `apply` → `axiom branch ensure --change <c> --role <rol>`;
    - cierre del rol → `time propose` y `time record`;
    - `archive` → `summary push` sobre la carpeta archivada, **antes** de `axiom sdd publish --archive`.
  - **En `checkpointed`:** después de `gate record`.
  - **En `continuous`:** al terminar la fase, sin parada.
  - **Fallos:**
    - nunca deshacen ni repiten la aprobación;
    - `task_not_bound` de `ensure --role` bloquea el `apply` y se presenta con sus dos resoluciones, también en `continuous`;
    - en `continuous`, los fallos de `summary push`, `time propose`, `time record` y la transición los registran los propios verbos en `.tracker-notes.yaml` (D20), y la cadena sigue (hueco 18);
    - una propuesta de 0 h no se imputa: se invoca `time record --close-only` y se informa como «sin imputación» (hueco 17 de la spec, hueco 3 del design);
    - un `time propose` fallido por un motivo distinto de `task_not_bound` también lleva a `time record --close-only` (D21).
  - **Prohibiciones:** ninguna de estas invocaciones en ODD ni sin tracker activo.
- **`routing.go` (ODD):** una frase: ODD nunca invoca `axiom integrations tasks`, `summary` ni `time`.
- **`sdd-design`:** con tracker activo, la sección `## Estimación por rol` con una línea `- <rol>: <h> h` por rol del roster y la justificación fuera de las líneas de datos. Sin tracker, no se escribe.
- **`sdd-apply`:** con tracker activo, al cerrar el rol se lanza `time propose`.
  - En `checkpointed`, se presenta la propuesta y se espera a las horas confirmadas, con `role-apply:<rol>` aprobada.
  - En `continuous`, se invoca `time record --hours <propuesta>` sin parada.
  - Con una propuesta de 0 h, o un `time propose` fallido salvo por `task_not_bound`, se invoca `time record --close-only` (D21).
  - `rev_conflict` y `time_pending_ambiguous` nunca se reintentan solos.
  - Un aviso `time_pending` de `time propose` se deja a `time record`, que reconcilia primero.
- **`sdd-archive`:**
  - `summary push` sobre la carpeta archivada; un fallo se anota en el informe con el reintento.
  - En `continuous`, **antes de mover la carpeta**, se leen `integrations.yaml` y `.tracker-notes.yaml` (D20). Con ellos se compone, sin red, la sección `## Seguimiento en el tracker`:
    - por Task: rol, ID, URL, estimación (o «sin estimación»), horas imputadas (suma de minutos / 60, con dos decimales) y cada imputación con actor, fecha e `item_id`;
    - un rol sin imputación figura con 0 h;
    - los sucesos del diario sin `done` posterior van como pendientes, con su comando de resolución.

## Riesgos y aspectos no verificados contra Azure DevOps real

| Riesgo | Estado | Mitigación en el diseño |
|---|---|---|
| El `POST` acepta `/relations/-` con `Hierarchy-Reverse` en el mismo cuerpo | **No verificado** | Camino alternativo ante un 400 (D11) y reparación de huérfanas por WIQL |
| `CONTAINS` de WIQL sobre `System.Tags` con la marca | **No verificado** | Filtrado exacto posterior; si WIQL falla, no se crea |
| Nombres de los campos `Microsoft.VSTS.Scheduling.*` en el tipo Task | **No verificado** | Comprobación con `workitemtypes/Task/fields` y `field_missing` sin escritura |
| HTML conservado por `System.Description`, y si Azure DevOps lo normaliza (lo que provocaría un `PATCH` en cada `summary push`) | **No verificado** | Lista blanca mínima; el riesgo de reescritura perpetua queda abierto (pregunta abierta 1) |
| Grafía de etiquetas: Azure DevOps puede reutilizar la grafía de una definición existente del proyecto | **No verificado** | Marca comparada sin distinguir mayúsculas; las etiquetas de ID siguen la spec (con distinción) y aceptan el riesgo de `PATCH` repetido |
| Existencia de `Tech Review` en el proceso | Conocido: no es estándar | Configurable; estado inexistente → permanente con la lista de válidos |
| `System.AssignedTo` por nombre visible | **No verificado** | Comprobación de homónimos antes y del `id` devuelto después; avisos |
| 400 de reglas del flujo distinguible de otros 400 | **No verificado** | Todo 400 en el `PATCH` de estado se trata como `tracker_rejected` |
| Un `role.start` reentregado desde otra máquina | Diseñado | Asignación solo si el actor del evento es quien entrega (D13) |
| Duplicado por dos `tasks create` simultáneos en máquinas distintas | Aceptado | Aviso `task_duplicated` y `task_ambiguous` posterior |
| Tamaño de la PR (13 unidades) | Aceptado | `single-pr` con un commit por unidad; la persona puede pedir PR encadenadas |

## Preguntas abiertas

- [ ] Si Azure DevOps normaliza el HTML de `System.Description`, `summary push` escribirá siempre. Si se observa con un WI real (requiere autorización explícita del destino y del PAT), se guardaría la huella del último HTML enviado en estado local (`.summary-pushed.yaml`). No bloquea el diseño.

## Estimación por rol

Estimación de este propio cambio: 13 unidades que tocan núcleo, adaptador con doble, dos paquetes nuevos, cinco verbos con su JSON, barrera, canario, doctrina con goldens y documentación.

- fullstack: 64 h

## Huecos de cobertura de la spec

| # | Resolución | Dónde |
|---|---|---|
| 1 | **Resuelto.** El reloj parte de `max(started_at, recorded_at de la última imputación role.close/<rol>/*)`; la ventana diaria sigue anclada a la hora de `started_at`. | D7; prueba en `trackersync` |
| 2 | **Resuelto.** Regla de avance sin retrocesos: `role.start` solo desde categoría `Proposed`; nunca se reasigna una Task asignada a otro `id` (`task_state_kept`, `task_assigned_to_other`). La reentrada no vuelve a emitir `role.start` (RF-BRA-34 sin cambios). | D13; pruebas del adaptador |
| 3 | **Resuelto.** Un 400 de reglas en el `PATCH` de estado es `tracker_rejected`: permanente en `Deliver` (visible en `sync`, reactivable con `--retry-abandoned`) y aviso en el cierre de `continuous`. | D13 |
| 4 | **Resuelto dentro de la spec.** Sigue permanente (`prov:ADO:6`), y su mensaje nombra obtener el vínculo y `axiom integrations sync --retry-abandoned`. En origen no puede ocurrir por la barrera. | D13 y D5 |
| 5 | **Resuelto en spec y diseño.** Diario local `.time-pending.yaml` con base y valores esperados; reconciliación al empezar el siguiente `time *`; `time_pending_ambiguous` y `--resolve-pending applied|discarded`, con escenarios en `prov:INT:8`. | D9, D10 y `prov:INT:8` |
| 6 | **Resuelto.** Máximo dos decimales, `0 < h ≤ 1000`, `round(h × 60) ≥ 1`; al tracker con dos decimales. | D8 |
| 7 | **Resuelto en spec y diseño.** `tasks bind --replace` revalida la Task nueva y sustituye el vínculo del rol, conservando `started_at` e imputaciones; sin `--replace`, `task_already_bound` nombra `tasks bind … --replace`. Escenarios en `prov:INT:3`. | D3, D18 y catálogo |
| 8 | **Resuelto en spec y diseño.** Con el vínculo presente y una estimación confirmada distinta, `tasks create` actualiza `OriginalEstimate` y `RemainingWork = max(0, e − CompletedWork)` con `test /rev` y después `estimate_hours`; con la misma estimación no escribe. Escenarios en `prov:INT:2`. Se retira el aviso `estimate_drift`. | D17 |
| 9 | **Resuelto.** `--estimate` de un rol fuera del roster, o en `bind` de un rol distinto de `--role`, es error de uso (salida 2) sin red. | D14, paso 0 |
| 10 | **Resuelto.** Título `<cambio> / <rol>`; hereda `System.AreaPath` y `System.IterationPath` del WI padre. | D11 |
| 11 | **Resuelto.** Por encima del tope se retiran todas las etiquetas con el prefijo propio, incluidas las `prov:`, y no se envía ninguna de ID. Cumple `prov:ADO:5` («ninguna etiqueta de ID») y `prov:ARC:1`. | D12 |
| 12 | **Resuelto.** Orden `ADDED`, `MODIFIED` y `RENAMED` con `h3` + `p`; `REMOVED` en `ul/li`, sin etiqueta y retirando la que tuvieran; primer bloque no párrafo → sin `p`; delimitadores sin cerrar literales; tope de 64 KiB determinista. | D12; pruebas de `specsummary` |
| 13 | **Resuelto.** La doctrina hace `summary push` antes de `sdd publish --archive` (que emite `archive`); WI sin edición → `tracker_rejected` con reintento; carpeta archivada más reciente por fecha; la sustitución de una descripción editada a mano es intencionada y consta en el JSON. | D12 y doctrina |
| 14 | **Resuelto.** `integrations.yaml` vive en la carpeta del cambio del planning home (repositorio de specs en `multirepo`); los verbos lo resuelven como `validateEnsureRole`; cada rol tiene su `source` y su reloj. | D14 |
| 15 | **Resuelto en parte.** `time propose` y `time record` validan la Task (existe, es Task y cuelga del WI) antes de leer horas. `ensure --role` no revalida, para no depender de la red (D5). Una Task cerrada o borrada se sustituye con `tasks bind --replace` (D18). | D5 y D14 |
| 16 | **Aceptado.** Sin exclusión entre máquinas; aviso `task_duplicated` tras crear y `task_ambiguous` después. | D11 |
| 17 | **Resuelto.** En `continuous`, una propuesta de 0 h no llama a `time record` y se informa como «sin imputación» en el resumen final. | Doctrina U12 |
| 18 | **Resuelto.** En `continuous`, un fallo de `time propose` no detiene la cadena ni lleva a `time record`, y queda como pendiente en el resumen final. | Doctrina U12 |
| 19 | **Resuelto en parte.** Una etiqueta de más de 400 caracteres o con `,` o `;` → `req_tag_invalid` sin enviar ninguna. La distinción de mayúsculas sigue la spec; el riesgo de `PATCH` repetido por normalización del tracker queda **no verificado**. | D12 y riesgos |
| 20 | **Resuelto.** Zona `time.Local` nombrada en la salida; ventana anclada con duración absoluta (cambio de hora correcto); jornada que cruza la medianoche imputada al día de inicio; minutos truncados y propuesta en cuartos de hora. | D7 |
| 21 | **Resuelto.** Hijas por relaciones y lotes de 200; se ignoran las de categoría `Removed` y las que no son de tipo Task; la marca incluye el cambio; huérfanas por WIQL; WIQL caído → sin creación. | D11 |
| 22 | **Resuelto.** `work_item_not_linked` nombra `axiom change start --from <id|url>` y `--adopt`. Una edición de `design.md` tras crear las Tasks deja `design` en `pending` y `tasks create` responde `gate_not_approved` (digest vigente exigido); tras reaprobar, `tasks create` propaga la nueva estimación (D17). | D14 |

**Contradicción entre requisitos (resuelta en la spec).** Sin vínculo de Task del rol, `prov:INT:9` decía `artifact_missing` para `time *` y `prov:INT:7`, `task_not_bound`. La spec queda con un único comportamiento: `task_not_bound` en ambos estilos, con escenario propio en `prov:INT:9`.

## Huecos de cobertura del design

Respuesta a `coverage-review.design.md` (ronda 1, 16 huecos). Todos quedan resueltos en el diseño; los que tocan la spec llevan un ajuste mínimo en `prov:INT:8`, sin cambiar IDs.

| # | Resolución | Dónde |
|---|---|---|
| 1 | **Resuelto.** `TransitionRequest.Assignee *TrackerIdentity` (nil = no asignar) y `TransitionResult.Assigned`; `Deliver` decide si procede (actor = quien entrega, identidad guardada) y `TransitionTask` aplica homónimos, «otro asignado» y verificación del `id` devuelto; los avisos van en `TransitionResult.Warnings` y `Deliver` los copia a `Outcome.Warnings`. | D13; contrato del puerto; pruebas del adaptador |
| 2 | **Resuelto.** `tasks bind` no tiene precondición de estilo (ni compuerta ni artefacto): es el verbo de rescate; le bastan uso, carpeta, plugin, vínculo de WI y roster. | D14; prueba de CLI con `design` pendiente |
| 3 | **Resuelto (con ajuste mínimo de spec).** `time record --close-only` en `continuous` transiciona sin imputar; la transición de cierre se aplica también tras `time_record_recovered` y `--resolve-pending applied`; la doctrina usa `--close-only` con 0 h y con un `time propose` fallido (salvo `task_not_bound`). | D21; `prov:INT:8`; doctrina U12; pruebas de CLI y de doctrina |
| 4 | **Resuelto.** Pruebas de `RecordHours` con `CompletedWork` ausente en el proceso (`FieldMissingError`, sin escritura) y ausente en el valor (leído como 0). | Estrategia de pruebas (adaptador) |
| 5 | **Resuelto.** `estimate_not_recorded` lo emite `tasks bind` sin ninguna estimación; JSON con `estimate_hours: null` y `estimate_source: "none"`; `time propose` y el resumen final muestran «sin estimación»; `tasks create --estimate` lo completa por D17. | D14; catálogo de avisos |
| 6 | **Resuelto.** Pruebas de `prov:BRA:1`: sin `started_at` con `decision_required`, sin `--role` y desde la ruta del dashboard que llama al motor, con `integrations.yaml` intacto. | Estrategia de pruebas (CLI); U10 |
| 7 | **Resuelto.** Dominio de `--estimate` (positivo, ≤ 2 decimales, ≤ 1000, sin roles repetidos) validado como error de uso antes de la red; `tasks create` `partial` con un rol fallido sin impedir el otro y `gates.yaml` intacto; `BindTask` fallido tras el `POST` → `write_failed` y el reintento encuentra la Task por la marca. | D14; pruebas de CLI; U7 |
| 8 | **Resuelto (con ajuste mínimo de spec).** Prevalece `prov:INT:7`: `time propose` es de solo lectura y con una intención pendiente solo avisa `time_pending`; la reconciliación, que escribe, ocurre únicamente en `time record`. | D9; `prov:INT:8` |
| 9 | **Resuelto.** Prueba con nombre propio `TestADOLifecycleConfigDefaults`: carga el workspace sin claves nuevas y comprueba los valores efectivos con `lifecycleConfigFrom`, incluido el cierre `Tech Review` derivado. | Estrategia de pruebas; U4 |
| 10 | **Resuelto.** Diario local `.tracker-notes.yaml` (solo en `continuous`, solo de añadir, no publicado) que escriben los cinco verbos; `sdd-archive` lo lee junto con `integrations.yaml` antes de mover la carpeta; prueba de CLI del diario y prueba de doctrina de la skill. | D20; doctrina U12 |
| 11 | **Resuelto.** `archive` sin vínculo de WI → éxito sin red con aviso `work_item_not_linked`; `role.start` sobre una Task ya `Active` sin asignar → `PATCH` solo de `System.AssignedTo` (asignación independiente de la regla de estado). | D13; pruebas del adaptador |
| 12 | **Resuelto.** Decisión propia D19: estados del tipo del WI (`User Story`, `Bug`…), WI `Completed`/`Removed` no se mueve (`work_item_state_kept`), destino alcanzado sin escritura, estado inexistente con la lista de válidos, conflicto transitorio, 400 de reglas permanente. | D19; pruebas del adaptador |
| 13 | **Resuelto (con ajuste mínimo de spec).** La intención solo se borra cuando consta que el `PATCH` no se aplicó (conflicto en el primer intento o 4xx no conflicto); un `TransportError` sin respuesta, tiempo agotado, 5xx o conflicto tras reintento no verificado la conservan. | D9; `prov:INT:8`; pruebas del adaptador y de CLI |
| 14 | **Resuelto (reconciliar contra el `item_id`).** La reconciliación relee siempre la Task de la intención, no la del vínculo vigente, así que `tasks bind --replace` no necesita rechazarse con una intención pendiente. | D9; `prov:INT:8`; prueba de CLI |
| 15 | **Resuelto.** El renderizador emite los requisitos de cada capacidad en el orden del fichero (`Requirement.Line`), cada uno con `h3` + `p`; `REMOVED` y `RENAMED` llevan un marcador en `strong` al inicio de su `p`; no se emiten `ul` ni `li`. Cumple `prov:INT:5` sin cambiarlo. | D12; pruebas de `specsummary` |
| 16 | **Resuelto.** `TestTrackerDoctrineNamesNoSecretsOrTrackerRoutes` recorre la doctrina proyectada y las tres skills y falla con rutas del tracker, cabeceras de credencial, el canario de PAT o un `uniqueName`. | Estrategia de pruebas (doctrina); U12 |
