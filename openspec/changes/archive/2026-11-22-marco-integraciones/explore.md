# Exploración: marco-integraciones (C-07-b · Marco de plugins de integración)

Fase `explore`. Rutas relativas a la raíz del worktree `C:\repos\axiom-wt\marco-integraciones\axiom`. No se ha modificado código.

## 1. Estado actual

### 1.1 Bloques estrictos de `axiom.yaml` (`internal/workspace`)

- `readDeclarationBlocks` (`declarations.go:59-96`) localiza cada bloque de primer nivel con `mappingValue` y lo decodifica con una función propia (`decodeGit`, `decodeWorkflow`, `decodeSDD`, `decodeRepositories`, `decodeTechnicalContext`). Cada bloque tiene su canal de error en `Declarations` (`declarations.go:31-44`): `XxxErr`. Un bloque nuevo se añade en cuatro sitios del propio paquete:
  1. Campos `Integrations` / `IntegrationsErr` en `Declarations` (`declarations.go:31-44`).
  2. Rama nueva en `readDeclarationBlocks` (`declarations.go:59-96`).
  3. `joinLoadErrors` (`loader.go:43`): la lista `declarations.XxxErr` que convierte el error en error de carga de `ParseConfig`/`LoadConfig`/`workspace validate`.
  4. Proyección a `WorkspaceConfig` en `parseLayered` (`loader.go:102-117`) con campo `yaml:"-"` en `types.go:70-78`.
- Patrón de clave estricta: `unknownKeys(node, label, allowed)` (`declarations.go:276-285`) produce «clave desconocida 'x' en 'bloque' (línea N); claves admitidas: …». `decodeStringKeys` (`gitpolicy.go:291-314`) añade claves duplicadas y validación de valores. `sdd.go:27-58` y `workflow.go:55-66` son los ejemplos mínimos más recientes. Un bloque nulo o que no es mapeo se rechaza.
- **Matiz importante:** `Locate` (`locate.go:300-345`) NO rechaza por errores de declaraciones (D-03): solo por el error de la base. Así que un `integrations:` inválido es error de carga en `LoadConfig`/`ParseConfig`/`workspace validate` (`cmd/axiom/main.go:578-587`), pero llega a los verbos de `internal/cli` (que usan `Locate`) solo como `loc.Declarations.IntegrationsErr`. El despachador de eventos debe comprobarlo por sí mismo.
- Otros consumidores que deben conocer el bloque:
  - `internal/hub/init.go:65-73` (`ownMarkers.declares()`): lista los bloques que `init --force` no debe sobrescribir. Hay que añadir `Integrations`/`IntegrationsErr`.
  - `internal/sddstatus/workspace_declarations.go:206-220` (`invalidDeclarationErrors`) solo bloquea el archivado por `ParseErr`, `baseline` y `technical_context`. Los bloques `git`/`workflow`/`sdd` no lo hacen; `integrations` tampoco debería.
- No existe JSON Schema de `axiom.yaml` (los `*.schema.json` son de `contracts/review-integration` y `contracts/sdd-integration`). No hay catálogo de claves hoy: el catálogo lo introduce C-06-e. El único «catálogo» es doc + plantilla + pruebas de contrato:
  - `axiom.example.yaml:32-96` (bloques comentados, prueba `internal/workspace/example_config_test.go:46-90` y `docs_contract_test.go:208-252`).
  - `docs/manual-de-inicio.md:272-379` (sección 6, un subapartado por bloque: `git:` `:319`, `workflow:` `:360`, `sdd:` `:379`). `docs_contract_test.go` fija frases literales del manual y de la plantilla: el bloque nuevo necesita su propio apartado y su prueba de contrato.
- Los binarios anteriores ignoran claves de primer nivel desconocidas (REQ-24.17), así que `integrations:` no rompe versiones viejas.

### 1.2 `runSDDGateRecord` tras C-07-a (`internal/cli/sdd_gate.go:54-203`)

- Orden: parseo (`:55`), `resolveGovernanceChangeRoot` (`:59`), guarda A (`:71`), kickoff (`:75`), validación de rol (`:89-99`), actor git con `defaultActor` (`:103-106`, `sdd_actor_identity.go:40`), digest (`:107`), evidencia (`:135-144`), **`kickoff.AppendGate`/`AppendGateChecked` (`:145-154`)**, `emitActorIdentityWarning` (`:155`), publish (`:159`), salida (`:161-201`).
- **Punto de enganche:** inmediatamente después de `emitActorIdentityWarning(actorWarning)` (`:155`) y antes de cualquier código que pueda devolver error (`writeGateRecordHeader`, `lastBlockingRoleJustClosed`, `writeGateRecordJSON`). Ahí la decisión ya está en `gates.yaml` y el verbo no puede fallar por la integración.
- Datos disponibles en ese punto: `parsed.Gate` (`spec|design|tasks|integration|role-apply:<rol>`), `parsed.Decision`, `parsed.Change`, `changeRoot`, `workspaceRoot`, `loc` (con `loc.Config` y `loc.Declarations`), `actor`, `record.RecordedAt`, `digest`, `sealed` (kickoff; `Config.ExecutionStyle`, roster).
- Eventos: solo con `Decision == approved` y gate `spec` → `gate.spec.approved`, `design` → `gate.design.approved`, `role-apply:<r>` → `role.apply.approved` (rol = sufijo; `roleApplyGatePrefix` no es exportado, `internal/kickoff/types.go:92-100`; `sdd_gate_handoff.go:82` ya hace `strings.TrimPrefix`: conviene exportar un helper en `kickoff`). `tasks` e `integration` no tienen evento en el backlog.
- El contrato de salida (`axiom.sdd-gate-record/v1`, `sdd_gate_handoff.go:20`; REQ-42.9) no debe cambiar: los avisos del despacho van a stderr con una variable inyectable, como `actorIdentityWarnings` (`sdd_actor_identity.go:24`).
- Con `continuous` no hay paradas ni obligación de invocar `gate record`: la señal de fin de fase en continuo (C-07-e `:208-211`) necesita un disparador explícito distinto. Un verbo `emit` genérico lo cubre.

### 1.3 `ensure` tras C-02-a (`internal/branch/engine/ensure.go`)

- `Engine.Ensure` está en `ensure.go:83` (el `:79` del backlog está desplazado). **No tiene noción de rol**: `EnsureRequest` (`:27-40`) no lleva `Role`. Crea la rama del cambio y, **solo si el plan tiene ≥2 roles**, las ramas de rol (`internal/branch/planner.go:427-446`). Con un único rol (`fullstack`, la topología de este repo) no hay rama de rol.
- Hoy `ensure` se invoca al arrancar el cambio (antes del kickoff) y de nuevo antes de la primera unidad de trabajo del apply (`internal/assets/skills/sdd-apply/SKILL.md:341`), sin decir qué rol. La entrada por rol con rama diferida es C-07-f y el bloqueo sin Task es C-07-e. Por tanto **`role.start` no puede derivarse hoy de «ensure de un rol»**: ese concepto aún no existe.
- Llamadores de `Ensure`: `internal/cli/branch.go:297` (verbo) y `internal/dashboard/service_branch.go:38` (alta de incremento, sin rol y antes del kickoff). Ambos pasan por un `Result` con `Status == StatusReady`, `PlanningHome`, `Change`, `Lane`.
- Idempotencia: `ensure` es idempotente por diseño (`:42-70`); un `ensure` repetido devuelve el mismo estado y no crea nada. Para que `role.start` sea idempotente hace falta clave de outbox (`role.start/<cambio>/<rol>`), no el motor.
- `ensure.go` figura como solape duro de C-05-d, C-07-b, C-07-e y C-07-f (`README.md:182`).

### 1.4 Archivado

- **No existe un verbo determinista que archive.** El movimiento a `openspec/changes/archive/<fecha>-<cambio>/` lo hace la skill `sdd-archive` con comandos de shell (`internal/assets/skills/sdd-archive/SKILL.md:259-265`, `_shared/openspec-convention.md:56`).
- Verbos deterministas por cambio en el entorno del archivado: `axiom branch ensure --archive` (crea la rama de archivado, antes del archivado), `axiom archive context prepare|verify` (solo con contexto técnico mantenido), `axiom archive integrate --odd` (ODD) y **`axiom sdd publish --archive`** (`internal/cli/sdd_publish.go:90-105`: confirma los documentos ya archivados; resuelve la raíz archivada con `publishChangeRoot`, `:126-132`). `axiom archive sync/list/show/coldstart` no tienen cambio como argumento.
- El mejor candidato determinista para `archive` es `sdd publish --archive` con resultado no rechazado. Su límite: depende de que alguien lo invoque (en `continuous` con `docs_publication: auto` lo dispara la doctrina; en `checkpointed` lo hace la persona/orquestador). Un `emit --event archive` explícito sirve de respaldo.

### 1.5 Patrones a seguir

- **Puertos:** `internal/branch/ports.go` (interfaces `Git`, `Registry`, `Prefs`, `LineageProbe` en el dominio; adaptadores en subpaquetes `gitcli`, `registry`; el motor solo consume interfaces).
- **Registro cerrado:** `internal/agents/registry.go:10-66` (`NewRegistry(adapters...)` con `ErrDuplicateAdapter` y manifiesto resuelto en `Register`) y `factory.go:87-104` (`NewDefaultRegistry` sobre la lista cerrada `defaultAgentIDs`). Es el patrón para `integrations.NewRegistry(plugins...)` + `DefaultRegistry()` (vacío en este cambio: el adaptador real es C-07-c; las pruebas inyectan el doble).
- **Fichero máquina por cambio con esquema versionado:** `internal/kickoff/ledger.go:15-147` (`gates.yaml`, `axiom.sdd-gate-ledger/v1`, `types.go:49-51`). Escritura: cerrojo `.gates.lock` con `reviewtransaction.AcquireAuthorityFileLock` y reintento con backoff y jitter (`:149-190`), lectura-modificación-escritura bajo cerrojo, temporal `.gates-*.yaml` en la misma carpeta y `reviewtransaction.ReplaceFileAtomic` + `SyncReviewDirectory` (`:120-145`), con costuras de prueba como variables de paquete (`:40-48`). `kickoff.yaml` es de escritura única (`seal.go`). `internal/filecoord` existe pero no lo importa ningún código no-test; el estándar real es el de `kickoff`.
- **Carpeta del cambio y `sdd publish`:** `internal/docpaths/docpaths.go:173-189` y `sddpublish/publish.go:161-193`. Todo fichero de la raíz de `changes/<cambio>/` cuyo nombre **no** empieza por punto es documentación y lo confirma `axiom sdd publish`; los ficheros con punto (`.gates.lock`, `.gates-*.yaml`) son estado local que nunca se versiona. Esto condiciona el diseño de `integrations.yaml` y de la outbox (ver 3.2).
- **Verbo nuevo en la CLI:** el adaptador fino `cli.RunWorkspaceContext` (`internal/cli/workspace_context.go:14-70`) y su alta en `cmd/axiom/main.go`: `case "branch": os.Exit(cli.RunBranch(...))` (`main.go:319-320`), el texto de ayuda global `axiomHelpText` (`main.go:112-198`, en el bloque «COMANDOS DE GOBERNANZA Y WORKSPACE») y pruebas que fijan la ayuda (`cmd/axiom/archive_help_test.go`, `branch_help_test.go`). Subcomandos con `runSimpleCommand` (`main.go:1781-1786`). Ya existe `case "change"` (`main.go:315`) para C-07-d.

### 1.6 Dependencias, tokens y configuración personal

- `go.mod`: módulo `github.com/IGutierrezZ/axiom/v3`, `go 1.25.10`. No hay keyring ni `vendor/`. Dependencias directas: bubbletea/lipgloss, `jsonschema/v6`, `golang.org/x/sys`, `yaml.v3`, etc.
- Tokens hoy: `internal/update/github.go:46-69` (`resolveGitHubToken`: `GITHUB_TOKEN` → `GH_TOKEN` → `gh auth token`), solo en memoria, enviado como cabecera `Authorization` (`:217`). No hay almacén.
- Config personal: `internal/hub/manager.go:25-33` (`workspaces.json` bajo `system.AxiomDir(home)` = `AXIOM_STATE_DIR` o `~/.axiom`). Preferencias de worktree, no secretos.
- `github.com/zalando/go-keyring` (MIT; Windows Credential Manager, `security` en macOS, Secret Service/D-Bus en Linux; `MockInit()` para pruebas; fuente: README en github.com/zalando/go-keyring). **Sin verificar** (hay que confirmarlo en diseño): versión actual a fijar, dependencias transitivas por plataforma (se esperan `wincred` y `dbus`), límite de tamaño por secreto y nombres exactos de errores (`ErrNotFound`, `ErrUnsupportedPlatform`).

### 1.7 Specs vivas

- `change-actor-identity` (REQ-42.1 a 42.9) es la última numeración (no hay REQ-43+ en `openspec/specs/`). Los eventos reutilizan `actor` de `gate record`; no requiere MODIFIED.
- `workspace-topology` ya documenta los bloques opcionales (`git:` REQ-25.26 `:252`, `repositories:` REQ-25.27 `:321`, `workflow:` REQ-27.1 `:381`, baseline retirada REQ-30.21 `:451`): el bloque `integrations:` es un requisito **ADDED** ahí (un delta de una capacidad existente).
- `sdd-block-review-gates`: requisitos de `gate record` (REQ-21.x/26.x/27.12). Añadir un requisito ADDED «la aprobación emite el evento y nunca falla por la integración» (no modifica las existentes).
- `change-branch-lifecycle` (REQ-25.x, último REQ-25.50): ADDED solo si se añade `--role` a `ensure`.
- `sdd-docs-publication`/`sdd-archive-lifecycle`: ADDED menor si `sdd publish --archive` dispara `archive`.
- **Capacidad nueva propuesta: `integration-plugins`** (REQ-43.x: puerto y manifiesto, registro cerrado, eventos, almacén de secretos, `integrations.yaml`, outbox, verbos). Riesgo de colisión de numeración con otros cambios en vuelo de la misma ola.

### 1.8 Dashboard (criterio «ningún secreto en la salida de un GET»)

- `GET /api/workspace` (`dashboard/service.go:149-216`) proyecta DTOs de lista blanca (`WorkspaceDTO`, `RoleMeta`, `BranchPolicyDTO`); **no serializa `axiom.yaml` ni `WorkspaceConfig`**. Un bloque nuevo no sale salvo que se añada al DTO a propósito (lo hará C-07-h).
- `GET /api/increments/<n>` (`server.go:556-574`, `service.go:330-475`) lee ficheros concretos (`tasks.md`, `design.md`, `proposal.md`, relevos), no vuelca la carpeta; `integrations.yaml` no se sirve.
- Ninguna ruta GET sirve ficheros arbitrarios del proyecto. El criterio se cumple por construcción; hace falta una prueba canario que lo fije (ver 5).

## 2. Áreas afectadas

| Ruta | Motivo |
|---|---|
| `internal/integrations/` (nuevo) | puerto `Plugin`, manifiesto JSON, eventos, registro cerrado, despachador, outbox, almacén de vínculos, secretos |
| `internal/integrations/integrationstest/` (nuevo) | doble en memoria para el criterio de terminado |
| `internal/workspace/declarations.go`, `loader.go`, `types.go`, nuevo `integrations.go` | bloque estricto, errores de carga, proyección |
| `internal/hub/init.go:65-73` | `declares()` |
| `internal/cli/sdd_gate.go:155`, nuevo `internal/cli/integrations_events.go` | enganche `gate.*` y `role.apply.approved` (una sola llamada, para minimizar conflicto con C-06-f/C-07-e) |
| `internal/cli/branch.go:297` (o `internal/branch/engine/ensure.go`) | `role.start` |
| `internal/cli/sdd_publish.go:104` | `archive` |
| `internal/cli/integrations.go` (nuevo), `cmd/axiom/main.go` | verbos `axiom integrations ...` y ayuda |
| `internal/kickoff/types.go` | helper exportado para extraer el rol de `role-apply:<rol>` |
| `axiom.example.yaml`, `docs/manual-de-inicio.md`, `internal/workspace/docs_contract_test.go` | plantilla, manual y prueba de contrato |
| `go.mod`, `go.sum` | `go-keyring` |
| `openspec/specs/` | capacidad nueva `integration-plugins`; deltas ADDED a `workspace-topology`, `sdd-block-review-gates` y, según decisión, `change-branch-lifecycle` |

## 3. Enfoques

### 3.1 Dónde vive el esquema de configuración de cada plugin y quién valida las claves

El criterio exige que una clave desconocida en `integrations.*` sea error de carga, y `parseLayered` no recibe parámetros.

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| **A. Paquete hoja de manifiestos** (`internal/integrations/manifest`, sin dependencias de `workspace`) con tipos `Manifest` y `ConfigField` y un catálogo cerrado en compilación; `workspace` lo importa y valida contra él | Cumple el criterio literalmente (error de carga en `ParseConfig`); sin ciclos; el manifiesto es la única fuente (UI C-07-h y C-06-e lo leen) | `workspace` pasa a depender de un paquete hoja nuevo; las pruebas necesitan inyectar el doble en el catálogo (costura de variable de paquete, como `kickoff/ledger.go:40`) | Media |
| B. `workspace` solo valida la forma (`integrations.<id>` mapeo de escalares) y `integrations` valida las claves al cargar el registro | `workspace` no cambia de dependencias | La clave desconocida deja de ser error de carga: incumple el criterio; dos validaciones | Baja |
| C. Inyección por parámetro (`ParseConfig(data, schemas)`) | Sin estado global | Cambia la firma de `ParseConfig`/`LoadConfig`/`Locate`/`ReadDeclarations` y todos sus llamadores | Alta |

### 3.2 `integrations.yaml` y la cola pendiente

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| **A. Dos ficheros:** `integrations.yaml` (versionado, publicado por `sdd publish`: vínculos WI, Task por rol, imputaciones) y `.integrations-outbox.yaml` (estado local, empieza por punto → `LocalState`, nunca se versiona; cerrojo `.integrations.lock`) | La cola es por máquina y no genera conflictos de fusión; `sync` no ensucia el árbol tras publicar; reutiliza la regla de puntos ya existente | Se aparta del texto del backlog (`:81` mete la cola en `integrations.yaml`); dos esquemas | Media |
| B. Un solo fichero versionado con la cola dentro (texto literal del backlog) | Un esquema | La cola se confirma y se publica; dos personas reintentan lo mismo; cada `sync` ensucia la carpeta del cambio y obliga a otro `sdd publish`; conflictos de fusión | Baja |

### 3.3 Punto de disparo de `role.start`

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| **A. `--role <id>` opcional en `axiom branch ensure`; el adaptador CLI (`cli/branch.go:297`) emite `role.start` tras `StatusReady` en carril `sdd` sin `--archive`** | El motor queda puro y no depende de `integrations`; la clave `role.start/<cambio>/<rol>` hace idempotente el reintento; coincide con el `ensure` previo al apply de `sdd-apply:341` | Añade una bandera pública a `ensure` (ADDED en `change-branch-lifecycle`); el motor de `ensure.go` no se toca ahora pero C-07-e/f sí lo tocarán; el dashboard no emite (correcto) | Baja-Media |
| B. Puerto notificador dentro del motor (`branch.EventSink` inyectado en `Engine`) | Cubre todo llamador | Acopla el motor (patrón de puertos válido, pero es solape duro de `ensure.go`) y sigue sin saber el rol | Media |
| C. Solo `axiom integrations emit --event role.start --role X` | No toca `ensure` | No es «desde el verbo»: depende de la doctrina; no cumple el criterio al pie de la letra | Baja |

### 3.4 Disparo de `archive`

| Enfoque | Pros | Contras |
|---|---|---|
| **A. `sdd publish --archive` (resultado no rechazado) + `emit` de respaldo** | Determinista y por cambio; ya resuelve la carpeta archivada | Depende de que se invoque `publish --archive` |
| B. `branch ensure --archive` | Siempre se ejecuta al archivar | Ocurre **antes** del archivado: semántica equivocada para pasar el WI a cerrado |
| C. Solo `emit` | Simple | No determinista (lo invoca la doctrina) |

### 3.5 Secretos

Puerto `SecretStore` con tres adaptadores: `env` (`AXIOM_INTEGRATION_<PLUGIN>_<CLAVE>`), `keyring` (go-keyring; servicio `axiom`, usuario `<plugin>/<clave>`) y una cadena que consulta primero el entorno y luego el keyring. Sin adaptador de texto plano: si el keyring no está disponible (Linux sin D-Bus, CI), error tipado `ErrKeyringUnavailable` que nombra la variable de entorno alternativa. Tipo `Secret` que implementa `fmt.Stringer`, `MarshalJSON`, `MarshalYAML` y `GoString` redactados. La outbox nunca guarda secretos: el plugin los pide al almacén en el momento de entregar. El token se introduce solo por stdin o prompt sin eco (nunca por argumento).

## 4. Recomendación

1. **Paquetes** (sigue `branch/ports.go` y `agents/registry.go`): `internal/integrations/manifest` (hoja, JSON: `Manifest{Schema, ID, Capabilities, Events, Config []ConfigField, Identity}` con esquema `axiom.integration-manifest/v1`; el registro rechaza capacidades o eventos fuera de las listas cerradas y campos de configuración con nombre de secreto o marcados como secretos), `internal/integrations` (puerto `Plugin`, `Event`, `Registry` con `NewRegistry`/`DefaultRegistry`, `Dispatcher`, `Outbox`, `LinkStore`), `internal/integrations/secrets` (adaptadores), `internal/integrations/integrationstest` (doble).
2. **Bloque `integrations:`** con el enfoque 3.1-A. Admite solo ids del catálogo y claves del manifiesto; errores acumulados con línea. Lo no configurado es coste cero: sin bloque o sin plugin habilitado, el despachador vuelve antes de crear ningún fichero (los proyectos actuales y sus goldens no cambian). Bloque inválido en el momento de la compuerta: aviso por stderr y sin despacho, nunca fallo de la compuerta.
3. **Ficheros:** enfoque 3.2-A, con el patrón de `kickoff/ledger.go` (cerrojo, temporal, reemplazo atómico, esquemas `axiom.integrations/v1` y `axiom.integrations-outbox/v1`). Escritura bajo guarda A (`guardChangeWriter`, `ContextArchive` para el archivado). Clave de idempotencia: `<evento>/<cambio>[/<rol>]/<huella>` donde la huella se deriva del ledger (digest del artefacto o ordinal de la aprobación), de modo que reaprobar una spec cambiada genera operación nueva y un reintento no.
4. **Eventos:** `gate.spec.approved`, `gate.design.approved`, `role.apply.approved` desde `sdd_gate.go:155`; `role.start` con 3.3-A; `archive` con 3.4-A. Entrega = encolar de forma duradera + intento inmediato acotado si hay plugin; el fallo deja la operación pendiente. `axiom integrations sync` drena la cola con reintento y devuelve código distinto de cero si queda algo pendiente.
5. **Verbos:** `axiom integrations sync|status|emit` y `axiom integrations secret set|status|delete` (stdin). Alta en `main.go`, ayuda global y pruebas de ayuda.
6. **Primitivas de fichero:** envolver `reviewtransaction.AcquireAuthorityFileLock` y `ReplaceFileAtomic` tras costuras de variable de paquete: el plan indica que C-01-e (ola 16) extrae esas primitivas, y así el cambio posterior es de una línea.
7. **Entrega:** un solo PR con `size:exception` (preferencia registrada del usuario). Commits por unidad de trabajo sugeridos: (1) bloque `integrations:` + plantilla + manual + prueba de contrato; (2) manifiesto, puerto, registro y doble; (3) vínculos y outbox con cerrojo; (4) secretos; (5) verbos y ayuda; (6) enganches de los cuatro verbos; (7) specs vivas.

## 5. Riesgos

- **Publicación y ensuciado:** `integrations.yaml` es documentación publicable; si algún `sync` lo reescribe tras `sdd publish`, la carpeta queda sucia. Mitigación: la cola va a un fichero con punto (3.2-A) y los vínculos solo cambian en el despacho sincrónico de la propia compuerta, antes del `publish` que la doctrina ejecuta a continuación (`resolveGatePublish`, `sdd_gate.go:159`).
- **`role.start` sin concepto de rol en el motor** y semántica que C-07-e/f redefinirán (bloqueo sin Task, rama diferida). El `--role` debe documentarse como contrato provisional compatible.
- **Dependencia de RDD:** las primitivas de fichero salen de `internal/reviewtransaction`, que C-01-e retira al final del plan.
- **`Locate` no rechaza declaraciones inválidas:** sin la comprobación explícita de `IntegrationsErr` en el despachador, un bloque roto se ignoraría en silencio en los verbos.
- **Fugas de secretos:** argumentos de proceso, errores que envuelven el valor, logs, payload de la outbox, salida `--json`. Mitigación: tipo `Secret` redactado, prueba canario (valor distintivo en el almacén simulado; tras cada flujo, buscarlo en `axiom.yaml`, la carpeta del cambio, `integrations.yaml`, la outbox, stdout/stderr y las respuestas de los `GET` del dashboard).
- **Regresión de ayuda y docs:** `axiomHelpText` y `docs_contract_test.go` fijan texto literal; el manual (`manual-de-inicio.md:272+`) y la plantilla deben actualizarse en el mismo cambio.
- **Dependencia nueva:** `go-keyring` y sus transitivas (sin verificar). Las pruebas deben usar `MockInit()`; no hay D-Bus en CI Linux.
- **Colisión de numeración de REQ-43.x** con otros cambios de la ola 6 (`C-02-b`, `C-03-a`).
- **Duplicados entre máquinas:** el mismo cambio en dos clones tiene dos colas locales; la idempotencia debe poder apoyarse en el plugin (C-07-c) además de en la clave local.
- **Entorno (memoria del usuario):** Cortex XDR borra `axiom.exe` sueltos: verificar solo con `go test`, sin compilar binarios.
- **Dirección de importación:** `workspace` → `integrations/manifest` debe ser hoja pura; si `integrations` importa `workspace`, se rompe el ciclo.

## 6. Preguntas abiertas

### Decisión del usuario o de alcance (la coordinadora decide si las eleva)

1. **Verbo `emit` y verbos de secretos:** el backlog solo nombra `sync`. ¿Entran `axiom integrations emit` (necesario para `continuous` y como respaldo de `archive`) y `axiom integrations secret set|status|delete` en C-07-b o esperan a C-07-e/C-07-c/C-07-h? Recomendación: entran, porque `archive` y el almacén de secretos no se demuestran sin ellos.
2. **Bandera `--role` en `axiom branch ensure`** (contrato público): recomendada (3.3-A). Alternativa sin bandera: solo `emit` (3.3-C), que no cumple «desde el verbo».
3. **Semántica de `archive`:** «archivado confirmado» (`sdd publish --archive`) frente a «rama de archivado creada» (`ensure --archive`). Recomendación: la primera.
4. **Desviación del backlog:** la cola pendiente fuera de `integrations.yaml` (3.2-A). Es diseño de esquema (delegado), pero contradice el texto de `07-integraciones.md:81`: se avisa por ser visible.

### Diseño delegado a la sesión del cambio

- Nombres exactos de verbos y banderas; esquemas `axiom.integration-manifest/v1`, `axiom.integrations/v1`, `axiom.integrations-outbox/v1`; clave de idempotencia y huella; política de reintento (intentos máximos, qué cuenta como pendiente y cuándo se abandona) y tiempo máximo del intento inmediato.
- Contenido exacto del `Event` y del `Plugin` (métodos `Manifest`, `Handle`, `ResolveIdentity`); orden de precedencia entorno/keyring; formato de la variable de entorno.
- Persistencia de la identidad externa resuelta: fuera de alcance de C-07-b (es de C-07-c, `~/.axiom/`); aquí solo el contrato del manifiesto (`identity`).
- Nombre y estructura de los apartados de documentación y de la capacidad `integration-plugins`.

## 7. Estimación de tamaño

Núcleo `integrations` 800-1.000 líneas con pruebas; bloque de `workspace` 250-350; verbos CLI y ayuda 250-350; enganches 100-150; secretos 250-350; docs, plantilla y specs vivas 150-250 (no cuentan como código). Total **1.800-2.300 líneas de autoría con pruebas**, en el extremo alto del L del backlog (1.500-2.000). Un PR único con `size:exception`.

## 8. Listo para propuesta

Sí. Para `sdd-propose`: fijar el alcance de las preguntas 1-4 de 6, nombrar la capacidad `integration-plugins` (REQ-43.x) y declarar los deltas ADDED de `workspace-topology` y `sdd-block-review-gates` (más `change-branch-lifecycle` si se aprueba `--role`).

## Key Learnings

1. `Locate` no rechaza declaraciones inválidas: el error de carga del bloque `integrations:` solo aparece en `ParseConfig`/`LoadConfig`/`workspace validate`, así que el despachador comprueba `Declarations.IntegrationsErr` y degrada a aviso.
2. Todo fichero sin punto en `changes/<cambio>/` lo publica `axiom sdd publish`: los vínculos son versionables, la outbox no (fichero con punto, estado local).
3. No hay «ensure de un rol» ni verbo determinista que archive: `role.start` y `archive` necesitan una definición propia (`--role`, `sdd publish --archive`, `emit`).
4. Los GET del dashboard usan DTOs de lista blanca: el criterio de secretos se cumple por construcción y solo falta una prueba canario.
5. Las primitivas de fichero de `kickoff` viven en `reviewtransaction` (C-01-e las moverá): envolverlas tras costuras de variable de paquete.
