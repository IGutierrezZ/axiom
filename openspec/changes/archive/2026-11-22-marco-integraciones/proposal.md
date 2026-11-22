# Propuesta: marco de plugins de integración (C-07-b)

## Intención

Axiom no tiene ningún punto de extensión hacia trackers externos: no hay puerto, ni bloque en `axiom.yaml`, ni almacén de secretos, ni registro de vínculos por cambio, ni eventos del ciclo de vida. Los cambios C-07-c (adaptador Azure DevOps), C-07-d, C-07-e, C-07-g y C-07-h dependen de esa base, y el usuario pidió que funcione «como un plugin» y que todo lo que hacía el launcher sea «parametrización de proyecto».

Este cambio deja el núcleo preparado para cualquier tracker sin acoplarlo a ninguno: un puerto con capacidades y manifiesto serializable, un registro cerrado en compilación, un bloque estricto `integrations.<plugin>` en `axiom.yaml`, un almacén seguro de secretos, un fichero versionado de vínculos por cambio, una cola de salida local idempotente con reintento y la emisión de eventos desde los verbos deterministas que ya existen.

Éxito (criterio de terminado del backlog, `docs/plan/backlog-2026-11/07-integraciones.md:95-98`):

- Un plugin de prueba (doble en memoria) registrado y configurado en `axiom.yaml` recibe los eventos de cada verbo, persiste los vínculos en `integrations.yaml` y reintenta desde la outbox.
- Una clave desconocida en `integrations.*` es un error de carga.
- Ningún secreto llega a disco versionado ni a la salida de un GET.

## Alcance

### Dentro del alcance

1. **Paquete `internal/integrations`** (patrón de puertos de `internal/branch/ports.go` y registro cerrado de `internal/agents/registry.go`):
   - Puerto `Plugin` y tipo `Event`.
   - Catálogo cerrado de eventos: `change.start` (solo catálogo: lo dispara C-07-d), `gate.spec.approved`, `gate.design.approved`, `role.start`, `role.apply.approved` y `archive`.
   - Catálogo cerrado de capacidades: `items.read`, `items.transition`, `items.create`, `relations`, `assignment`, `estimation`, `time`, `tags`, `attachments`, `vcs_links` y `discovery`.
   - `Registry` con `NewRegistry(plugins...)` (rechaza duplicados) y `DefaultRegistry()`, vacío en este cambio: el primer adaptador real es C-07-c.
   - Despachador (`Dispatcher`): encola de forma duradera y hace un intento inmediato acotado.
   - Outbox idempotente con reintento.
   - Almacén de vínculos (`integrations.yaml`).
2. **Subpaquete hoja de manifiesto** (`internal/integrations/manifest`, sin dependencias de `workspace`):
   - Manifiesto serializable en JSON con esquema versionado (`axiom.integration-manifest/v1`, nombre definitivo en diseño).
   - Declara identificador, capacidades, eventos que escucha, esquema de configuración (alimenta C-07-h y C-06-e), nombres de los secretos que necesita y cómo se resuelve la identidad.
   - El registro rechaza capacidades o eventos fuera de las listas cerradas y campos de configuración marcados como secretos.
   - Lo importa `workspace` para validar las claves de `integrations.<plugin>`.
3. **Bloque estricto `integrations` en `axiom.yaml`** (`internal/workspace`):
   - Solo admite identificadores del catálogo de manifiestos y, dentro de cada uno, solo las claves de su manifiesto.
   - Una clave o un plugin desconocidos, un bloque nulo o que no es mapeo, o una clave duplicada son error de carga en `ParseConfig`, `LoadConfig` y `workspace validate`, con línea y claves admitidas, como el resto de bloques.
   - Canal de error propio (`IntegrationsErr`), proyección a `WorkspaceConfig` y alta en `hub/init.go` (`declares()`).
   - No bloquea el archivado (como `git`, `workflow` y `sdd`).
4. **Ficheros por cambio** (decisión 3 del usuario):
   - `integrations.yaml`, versionado y publicado por `axiom sdd publish`: solo vínculos e imputaciones, con esquema versionado (`axiom.integrations/v1`).
   - `.integrations-outbox.yaml`, estado local (con punto, nunca se publica): operaciones pendientes, intentos y último error, con esquema versionado (`axiom.integrations-outbox/v1`).
   - Escritura con cerrojo, temporal en la misma carpeta y reemplazo atómico, como `kickoff/ledger.go`, bajo la guarda A del contexto que corresponda.
   - La outbox nunca guarda secretos.
5. **Coste cero sin integración.** Sin bloque `integrations` o sin plugin configurado, el despachador vuelve antes de crear ningún fichero: los proyectos actuales y los goldens no cambian.
6. **Secretos** (`internal/integrations/secrets`):
   - Puerto `SecretStore` con adaptador de entorno (`AXIOM_INTEGRATION_<PLUGIN>_<CLAVE>`, formato exacto en diseño), adaptador de keyring del sistema (`github.com/zalando/go-keyring`: Windows Credential Manager, Keychain, Secret Service) y una cadena entorno → keyring.
   - Nunca hay caída a texto plano: si el keyring no está disponible, error tipado que nombra la variable de entorno alternativa.
   - Tipo `Secret` redactado en `String`, `GoString`, `MarshalJSON` y `MarshalYAML`.
   - El plugin pide el secreto al almacén en el momento de entregar.
7. **Verbos nuevos** (`internal/cli/integrations.go`, alta en `cmd/axiom/main.go`, ayuda global y pruebas de ayuda):
   - `axiom integrations sync`: drena la outbox con reintento; termina con código distinto de cero si queda algo pendiente.
   - `axiom integrations emit`: respaldo explícito para disparar un evento del catálogo sobre un cambio. `change.start` queda fuera: lo dispara C-07-d.
   - `axiom integrations secret set|status|delete`: el valor solo entra por stdin, nunca por argumento; `status` dice si hay secreto y de qué fuente, nunca el valor.
8. **Emisión de eventos desde los verbos deterministas.** Ninguna emisión hace fallar el verbo que la origina: los fallos y un bloque `integrations` inválido (comprobado explícitamente con `Declarations.IntegrationsErr`, porque `Locate` no lo rechaza) se avisan por stderr con un `io.Writer` sustituible, y el contrato de stdout no cambia.
   - `axiom sdd gate record` con `--decision approved`, tras escribir `gates.yaml` (`internal/cli/sdd_gate.go`, después de `emitActorIdentityWarning`):
     - `spec` → `gate.spec.approved`;
     - `design` → `gate.design.approved`;
     - `role-apply:<rol>` → `role.apply.approved`.
     - `tasks` e `integration` no emiten. `axiom.sdd-gate-record/v1` no cambia.
   - `axiom branch ensure --role <id>` (decisión 1 del usuario):
     - Bandera opcional, solo en carril `sdd` e incompatible con `--archive`.
     - El adaptador CLI (`internal/cli/branch.go`) emite `role.start` tras un `ensure` con estado `ready`, con clave de idempotencia `role.start/<cambio>/<rol>`.
     - El motor (`internal/branch/engine/ensure.go`) no cambia de semántica ni conoce el rol.
     - El dashboard, que llama al motor directamente, no emite.
     - Contrato provisional que heredarán C-07-e y C-07-f.
   - `axiom sdd publish --archive` (decisión 2 del usuario): emite `archive` cuando el resultado no es un rechazo (archivado confirmado).
9. **Doble de prueba** `internal/integrations/integrationstest`: plugin en memoria que registra los eventos recibidos, devuelve vínculos y puede fallar a voluntad para ejercitar la outbox. Se inyecta en el catálogo y en el registro mediante costuras de prueba.
10. **Documentación y contrato:** bloque comentado en `axiom.example.yaml`, apartado `integrations:` en la sección 6 de `docs/manual-de-inicio.md` y pruebas de contrato en `internal/workspace/docs_contract_test.go` y `example_config_test.go`.
11. **Pruebas canario de secretos:** un valor distintivo en el almacén simulado no aparece, tras cada flujo, en `axiom.yaml`, en la carpeta del cambio (incluidos `integrations.yaml` y la outbox), en stdout/stderr ni en las respuestas de los GET del dashboard.
12. **Specs vivas:** capacidad nueva `integration-plugins` y requisitos ADDED en cuatro capacidades existentes (ver «Capacidades»).

### Fuera del alcance

- Adaptador Azure DevOps, cliente REST, conexión, identidad resuelta y persistencia de esa identidad en `~/.axiom/` (C-07-c). Aquí solo se fija el contrato del manifiesto (`identity`).
- Arranque de un cambio desde un work item, verbo `axiom change start` y emisión de `change.start` (C-07-d).
- Tasks por rol, estimación, resumen de la spec, imputación, bloqueo del `ensure` de un rol sin Task, señales de fin de fase en `continuous` y cambios de doctrina o skills para invocar `ensure --role` o `emit` (C-07-e).
- Plantillas de nombre de rama y ramas de código diferidas a la entrada del rol (C-07-f).
- MCP de Azure DevOps para el agente (C-07-g).
- UI de integraciones, vista de outbox y vínculos y endpoints del dashboard (C-07-h). El dashboard no se modifica: solo se añade la prueba canario.
- Catálogo de claves de `axiom.yaml` para la UI (C-06-e).
- Plugins fuera de proceso o con carga dinámica de código: el manifiesto queda serializable para habilitarlos en el futuro.
- Un verbo de inspección de la outbox distinto de `sync`.
- Extraer las primitivas de fichero de `internal/reviewtransaction` (C-01-e): aquí solo se envuelven tras costuras.
- Cambios en guías de agente, skills y goldens.

## Capacidades

### Nuevas capacidades

- `integration-plugins`: puerto `Plugin`, manifiesto JSON versionado y sus listas cerradas de capacidades y eventos, registro cerrado en compilación, despachador (encolado duradero e intento inmediato acotado), clave de idempotencia por operación, outbox local con reintento y `axiom integrations sync`, `integrations.yaml` versionado con vínculos e imputaciones, almacén de secretos sin caída a texto plano y tipo redactado, verbos `axiom integrations emit` y `axiom integrations secret set|status|delete`, coste cero sin integración y garantía de no fuga de secretos. Prefijo propuesto **REQ-43** (máximo actual en `openspec/specs`: REQ-42, `change-actor-identity`; ningún otro worktree reclama REQ-43).

### Capacidades modificadas

Son requisitos ADDED, salvo dos MODIFIED que añadió la resolución de B-1 (opción B, `design.md` §0.1): REQ-21.18 en `sdd-archive-lifecycle` y REQ-27.5 en `sdd-docs-publication`. Se numeran también en la familia **REQ-43.x**, igual que `workflow:` entró en `workspace-topology` como REQ-27.1.

- `workspace-topology`: bloque opcional `integrations:` de primer nivel, estricto contra el catálogo de manifiestos; error de carga en `ParseConfig`/`LoadConfig`/`workspace validate`; `Locate` lo expone como error de declaración sin rechazar; no bloquea el archivado.
- `sdd-block-review-gates`: la aprobación de `spec`, `design` y `role-apply:<rol>` emite su evento tras persistir la decisión y nunca falla por la integración; `tasks` e `integration` no emiten; el contrato `axiom.sdd-gate-record/v1` no cambia.
- `change-branch-lifecycle`: `ensure --role <id>` opcional (carril `sdd`, incompatible con `--archive`), sin cambio de semántica del motor; emite `role.start` idempotente tras un `ensure` listo.
- `sdd-docs-publication`: `sdd publish --archive` con resultado no rechazado emite `archive`, sin alterar el resultado ni el código de salida de la publicación.

## Enfoque

Enfoques recomendados por la exploración (3.1-A, 3.2-A, 3.3-A y 3.4-A), con las decisiones del usuario del 2026-11-18.

1. **Dirección de dependencias.** `internal/integrations/manifest` es una hoja pura. `workspace` la importa para validar `integrations.<plugin>` sin cambiar la firma de `ParseConfig`, `LoadConfig` ni `Locate`. `internal/integrations` sí puede importar `workspace`, pero nunca al revés. El catálogo de manifiestos es cerrado en compilación y lleva una costura de variable de paquete para que las pruebas registren el doble.
2. **Despacho.** Una sola llamada por verbo a un adaptador fino (`internal/cli/integrations_events.go`), para minimizar los conflictos con C-06-f y C-07-e en `sdd_gate.go`:
   1. Comprueba `IntegrationsErr`. Si hay error, avisa y no despacha.
   2. Sin plugin configurado, vuelve sin tocar disco.
   3. Con plugin configurado, encola en `.integrations-outbox.yaml` con su clave de idempotencia e intenta entregar de inmediato con un tiempo acotado.
   4. Si la entrega tiene éxito, persiste los vínculos que devuelve el plugin en `integrations.yaml` y retira la operación.
   5. Si falla, la operación queda pendiente para `sync`.
3. **Idempotencia.** Clave estable por operación:
   - Compuertas y `archive`: `<evento>/<cambio>[/<rol>]/<huella>`. La huella se deriva del ledger (digest o ordinal de la aprobación), de modo que reaprobar un artefacto cambiado crea una operación nueva y un reintento no.
   - `role.start`: `role.start/<cambio>/<rol>`, fijada por el usuario.
   - El diseño concreta la huella, la política de reintento (intentos máximos, abandono) y el tiempo del intento inmediato.
4. **Ficheros.** Patrón de `kickoff/ledger.go`: cerrojo con `reviewtransaction.AcquireAuthorityFileLock`, temporal y `ReplaceFileAtomic`/`SyncReviewDirectory`, envueltos tras variables de paquete para que la extracción de C-01-e sea un cambio de una línea. `docpaths.LocalState` ya excluye de la publicación los ficheros con punto.
5. **`--role` en `ensure`.** Validación en el adaptador CLI: carril `sdd`, sin `--archive` y rol presente en el kickoff sellado; si no, rechazo tipado antes de mutar. El diseño fija los códigos. La emisión ocurre solo tras `StatusReady`.
6. **Secretos.** Orden entorno → keyring. Pruebas con `keyring.MockInit()`, sin D-Bus en CI. Errores tipados que nunca envuelven el valor. `secret set` lee de stdin; el diseño decide si admite un prompt sin eco con TTY.
7. **Verificación.** Solo con `go test` (sin compilar binarios sueltos, por el antivirus del entorno). Las pruebas e2e del criterio de terminado registran el doble, configuran `axiom.yaml` y recorren `gate record`, `ensure --role`, `publish --archive`, `emit` y `sync`.
8. **Unidades de trabajo sugeridas** (una PR, `single-pr` con `size:exception`):
   1. Manifiesto, puerto, registro y doble.
   2. Bloque `integrations:`, plantilla, manual y prueba de contrato.
   3. Vínculos y outbox con cerrojo.
   4. Secretos y dependencia.
   5. Verbos y ayuda.
   6. Enganches de `gate record`, `ensure --role` y `publish --archive`, con canarios.
   7. Specs vivas.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/integrations/` | Nuevo | Puerto `Plugin`, `Event`, catálogos cerrados, `Registry`, `Dispatcher`, outbox, almacén de vínculos |
| `internal/integrations/manifest/` | Nuevo | Manifiesto JSON versionado, catálogo cerrado con costura de prueba (hoja) |
| `internal/integrations/secrets/` | Nuevo | `SecretStore`, adaptadores de entorno y keyring, cadena, tipo `Secret` redactado |
| `internal/integrations/integrationstest/` | Nuevo | Doble en memoria con fallos programables |
| `internal/workspace/declarations.go`, `loader.go`, `types.go`, nuevo `integrations.go` | Modificado / Nuevo | Bloque estricto, `IntegrationsErr`, `joinLoadErrors`, proyección |
| `internal/hub/init.go` | Modificado | `declares()` conoce `Integrations` |
| `internal/cli/sdd_gate.go` | Modificado | Una llamada al despacho tras `emitActorIdentityWarning` |
| `internal/cli/branch.go` | Modificado | Bandera `--role`, validación y emisión de `role.start` |
| `internal/cli/sdd_publish.go` | Modificado | Emisión de `archive` con resultado no rechazado |
| `internal/cli/integrations.go`, `integrations_events.go` | Nuevo | Verbos `integrations sync`, `emit` y `secret` y adaptador de despacho |
| `internal/kickoff/types.go` | Modificado | Helper exportado para extraer el rol de `role-apply:<rol>` |
| `cmd/axiom/main.go` y pruebas de ayuda | Modificado | Alta del verbo `integrations` y texto de ayuda global |
| `axiom.example.yaml`, `docs/manual-de-inicio.md`, `internal/workspace/docs_contract_test.go`, `example_config_test.go` | Modificado | Plantilla, manual y contrato de docs |
| `internal/dashboard/` (solo pruebas) | Modificado | Prueba canario de que ningún GET devuelve el secreto |
| `go.mod`, `go.sum` | Modificado | `github.com/zalando/go-keyring` y transitivas |
| `openspec/changes/marco-integraciones/specs/{integration-plugins,workspace-topology,sdd-block-review-gates,change-branch-lifecycle,sdd-docs-publication}/spec.md` | Nuevo | Spec de la capacidad nueva y deltas ADDED |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| Fuga de secretos por argumentos, errores que envuelven el valor, logs, outbox o salida `--json` | Media | Tipo `Secret` redactado, valor solo por stdin, outbox sin secretos y pruebas canario sobre ficheros, stdout/stderr y GET del dashboard |
| `Locate` no rechaza declaraciones inválidas: un bloque roto se ignoraría en silencio en los verbos | Media | El despachador comprueba `Declarations.IntegrationsErr` y avisa por stderr; prueba dedicada |
| **`.integrations-outbox.yaml` persistente ensucia el árbol.** Un fichero sin seguimiento y no ignorado cuenta como árbol sucio: `ensure` en `in_place` (REQ-25.9) y la eliminación del worktree en `done` (REQ-25.17) lo rechazan. Si el worktree se elimina, se pierden las operaciones pendientes. | Media | Decisión de diseño, sin cambio de alcance: borrar la outbox al quedar vacía y elegir entre una regla de ignorado y excluir `docpaths.LocalState` de la comprobación de suciedad. Si la solución exige cambiar la semántica de suciedad de REQ-25.9/25.17, se eleva al usuario |
| **`archive` se emite después de que `sdd publish --archive` confirme.** Si el plugin devuelve vínculos, reescribir `integrations.yaml` dejaría sucia la carpeta archivada ya publicada. | Media | Decisión de diseño: o `archive` no modifica `integrations.yaml` en este cambio (solo outbox y entrega), o se despacha antes de la confirmación dentro de la misma publicación sin cambiar la semántica de «archivado confirmado». Los vínculos del criterio de terminado se demuestran con los eventos de compuerta |
| **`role.start` solo se dispara si alguien pasa `--role`.** La doctrina (`sdd-apply`) no lo hace hasta C-07-e/f. | Baja | Aceptado por la decisión 1 del usuario; queda documentado como contrato provisional, y `emit` cubre el respaldo manual |
| Dependencia nueva `go-keyring`: versión, transitivas por plataforma (`wincred`, `dbus`), límites de tamaño y nombres de error sin verificar | Media | Verificarlo en el diseño; `MockInit()` en pruebas; error tipado sin caída a texto plano |
| Primitivas de fichero en `internal/reviewtransaction`, que C-01-e retira al final del plan | Baja | Envolverlas tras costuras de variable de paquete |
| Regresión de texto literal en `axiomHelpText`, ayudas y `docs_contract_test.go` | Media | Actualizar ayuda, manual, plantilla y pruebas de contrato en la misma unidad de trabajo |
| Colisión de la familia REQ-43 con otros cambios de la ola 6 (C-02-b, C-03-a) | Media | Antes de archivar, confirmar en `main` el siguiente prefijo libre y renumerar si hace falta |
| Solape en `sdd_gate.go` y `branch.go`/`ensure.go` con C-05-d, C-06-f, C-07-e y C-07-f | Media | Una sola llamada por verbo a un adaptador fino; el motor de `ensure` no se toca |
| Ciclo de importación si `workspace` acaba importando `internal/integrations` | Baja | `manifest` hoja pura; prueba de arquitectura o revisión de importaciones en diseño |
| Duplicados entre máquinas: cada clon tiene su propia outbox local | Baja | Clave de idempotencia estable en el payload para que el plugin real (C-07-c) también deduplique |
| Tamaño por encima del L del backlog (1.800-2.300 líneas con pruebas) | Media | Una PR con `size:exception` (preferencia del usuario) y commits por unidad de trabajo |

## Plan de reversión

- Revertir la PR única: desaparecen `internal/integrations`, el bloque `integrations:`, los verbos `axiom integrations`, la bandera `--role` de `ensure` y las emisiones de `gate record` y `sdd publish --archive`. Las specs delta se retiran con la reversión.
- Sin migración de datos. Los proyectos sin bloque `integrations` no generaron ningún fichero (coste cero). Un `axiom.yaml` con `integrations:` sigue cargando en el binario revertido, porque las claves de primer nivel desconocidas se ignoran (REQ-24.17).
- Los `integrations.yaml` ya confirmados en carpetas de cambio son ficheros inertes para el binario revertido. Las `.integrations-outbox.yaml` locales se pueden borrar sin efecto.
- Los secretos guardados en el keyring del sistema no se borran solos: antes de revertir, `axiom integrations secret delete`; si no, se retiran a mano del gestor de credenciales (servicio `axiom`).
- Tras la reversión, C-07-c a C-07-h quedan bloqueados de nuevo.

## Dependencias

- C-07-a (`identidad-git-actor`, ya archivado), dependencia blanda: el actor de los eventos de compuerta reutiliza el `actor` resuelto por `gate record`.
- Dependencia externa nueva: `github.com/zalando/go-keyring` (MIT).
- Bloquea a C-07-c, C-07-d, C-07-e, C-07-g y C-07-h. Arista blanda con C-06-e, que registrará en su catálogo las claves de `integrations.<plugin>`.

## Criterios de éxito

- [ ] Un plugin de prueba en memoria, registrado y configurado en `axiom.yaml`, recibe:
  - `gate.spec.approved`, `gate.design.approved` y `role.apply.approved` desde `axiom sdd gate record --decision approved`;
  - `role.start` desde `axiom branch ensure --role <id>`;
  - `archive` desde `axiom sdd publish --archive`;
  - el evento indicado desde `axiom integrations emit`.
- [ ] Los vínculos que devuelve el plugin quedan en `integrations.yaml` de la carpeta del cambio, con su esquema versionado.
- [ ] Con el plugin fallando, la operación queda pendiente en `.integrations-outbox.yaml` y el verbo origen termina con éxito y un aviso por stderr. `axiom integrations sync` la reintenta y la retira al entregarla. Con algo pendiente, `sync` termina con código distinto de cero.
- [ ] Un reintento o un `ensure --role` repetido no duplica la operación (misma clave de idempotencia). Reaprobar un artefacto cambiado sí genera una operación nueva.
- [ ] Una clave o un plugin desconocidos en `integrations.*` son error de carga en `ParseConfig`/`LoadConfig`/`axiom workspace validate`, con línea y claves admitidas. En los verbos, el mismo bloque inválido produce un aviso y ningún despacho, sin hacer fallar el verbo.
- [ ] Sin bloque `integrations` no se crea ningún fichero nuevo y los goldens no cambian.
- [ ] Un valor canario del almacén de secretos no aparece en `axiom.yaml`, en la carpeta del cambio, en `integrations.yaml`, en la outbox, en stdout/stderr ni en las respuestas de los GET del dashboard. Sin keyring disponible, el error tipado nombra la variable de entorno y no cae a texto plano.
- [ ] `--role` fuera del carril `sdd`, junto con `--archive` o con un rol ausente del kickoff produce un rechazo tipado, sin mutar nada.
- [ ] `axiom.sdd-gate-record/v1` y el resultado de `sdd publish` no cambian.
- [ ] `go test ./internal/integrations/... ./internal/workspace/... ./internal/cli/... ./internal/kickoff/... ./internal/hub/... ./internal/dashboard/... ./cmd/axiom/...` en verde, y `go test ./...` sin regresiones.

## Estimación de tamaño

L alto: unas 1.800-2.300 líneas de autoría con pruebas, más docs y specs. Una sola PR (`single-pr`) con `size:exception`.
