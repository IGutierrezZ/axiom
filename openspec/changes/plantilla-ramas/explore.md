## Exploración: plantilla-ramas (C-07-f)

Plantillas de nombre de rama (`git.branch_template`) por proyecto, repositorio y rol, con las variables `{prefix}`, `{kind}`, `{change}`, `{role}`, `{slug}`, `{ref}` y `{task_ref}`, y ramas de código diferidas a la entrada del rol.

### Estado actual (con evidencia)

**1. Bloque `git:` y `repositories:`**
- `GitConfig` (`internal/workspace/gitpolicy.go:45`) y `RepositoryPolicy` (`:61`) son structs planos de cuatro y cuatro campos de cadena. Las listas de claves son cerradas: `gitKeys` y `repositoryKeys` (`:70-71`).
- `decodeGit` (`:103`) y `decodeRepositories` (`:133`) recorren el `yaml.Node` con `decodeStringKeys` (`:298`), que acumula errores y valida cada valor con `branchValidator(key)` (`:196`, un `switch` por clave). `unknownKeys` está en `declarations.go:327`. Añadir `branch_template` a ambos bloques es mecánico: una clave más en cada lista, un campo más y un caso nuevo en `branchValidator` (la validación de plantilla).
- Los roles NO pasan por `decodeStringKeys`: son `map[string]RoleConfig` de la decodificación base (`types.go:36-41`, `Roles` en `types.go:81`), y las claves desconocidas fuera de los bloques estrictos se ignoran (`workspace-topology/spec.md:103`). Un campo `BranchTemplate string yaml:"branch_template,omitempty"` en `RoleConfig` basta para la sobrescritura por rol, pero hay que añadir su validación en la carga (hoy `RoleConfig` no se valida clave a clave).
- `axiom.example.yaml:36-46` (bloque `git:`) y `:182-195` (bloque `repositories:`) son comentarios; `TestExampleGitBlockIsValidOnceUncommented` (`gitpolicy_test.go:346`) descomenta y valida el ejemplo, así que cualquier línea nueva en el ejemplo se ejecuta como prueba. No hay esquema JSON de `axiom.yaml` (la búsqueda de `branch_prefix` solo devuelve Go, YAML de ejemplo, kickoffs archivados y `scripts/axiom-worktree.ps1`).
- Valores por defecto: `DefaultBranchPrefix = "feat"` (`gitpolicy.go:24`); el `axiom.yaml` de este repositorio declara `inc`.

**2. Planificador y nombres**
- `planRepository` (`internal/branch/planner.go:381-448`): rama del cambio `prefix + "/" + change` (`:412`), sustituida por `--branch` (`:413-415`) y, por encima de ambas, por la instantánea (`:417-420`, D-10). Rama de rol `changeBranch + "-" + role` (`:434`), sustituida por la instantánea (`:435-437`). Los roles solo generan ramas con dos o más sobre el mismo repositorio (`:427`).
- `--prefix` sustituye el prefijo de la política (`:405`). La rama de archivado se deriva de la del cambio o del prefijo (`planner_archive.go:147-168`): con plantilla, `<rama del cambio>-archive` sigue valiendo (usa `in.ChangeBranch`, que sale de registro/instantánea, `engine/archive_plan.go:28,49-70`).
- `snapshotBranch`/`snapshotRepo` (`planner_snapshot.go:14-35`) buscan por `(kind, role)`. `kickoff.GitSnapshot` (`kickoff/gitsnapshot.go:24-70`) guarda **nombres** (`GitSnapshotBranch{Name, Kind, Role}`) y `Validate()` exige que `Name` sea un `ValidRefName` y que role vaya si y solo si `kind == role` (`:146-155`). No hay campo de plantilla.
- La instantánea la calcula `Engine.Snapshot` (`engine/snapshot.go:30-84`) con el mismo planificador y `gatherOptions{sealing: true, sealRoles}`. Hoy copia `repo.Branches` tal cual (`:72-77`).
- **No existe noción de `kind` feature|fix en `branch`**. El perfil bug solo vive en `proposal.md` (`isoevidence.ParseBugDeclaration`, `isoevidence/bug.go:50`) y en el staging de `change start` (`ContextDoc.Profile`, `changestart/staging.go:112`), no en `kickoff.yaml`. Al sellar, `proposal.md` aún no existe (el seal lo precede), así que `{kind}` no se puede leer del cambio en el seal.

**3. Congelado: el kickoff es de escritura única**
- `kickoff.Seal` publica con `PublishNoReplace` y devuelve el ganador previo sin sobrescribir (`kickoff/seal.go:26,61-112`). **No se puede reescribir la instantánea en el `ensure` del rol.** Es el hallazgo que más cambia el planteamiento de C-07-f: el criterio «el nombre queda congelado en la instantánea tras el primer `ensure`» (`07-integraciones.md:277`) no es implementable tal cual.
- Hay un registro local por cambio (`branch.Record`, `types.go:188-210`, `RecordRepo.Branches`), en el git common dir (estado derivado, no versionado), que `ensure` guarda siempre (`engine/ensure.go:203-210`) y que ya lleva los nombres creados. El orden de precedencia documentado es «instantánea, registro, política» (`planner_snapshot.go:7`, `verdict.go:229-232`), pero el plan ordinario **no lee el registro para el nombre del cambio ni de rol**: solo para excepciones (`engine.go:186`) y para el archivado (`archive_plan.go`). La guarda sí lo lee (`verdict.go:243-289`, `recordedBranches`). Congelar el nombre en el registro es viable, pero hay que meterlo en `PlanInput`.

**4. `ensure` y rol**
- `engine.Ensure` (`ensure.go:84-242`) **no conoce el rol** (`EnsureRequest` sin campo `Role`, `:28-41`; lo dice `cli/branch_role.go:16-18`). `--role` es un adaptador: valida antes (`validateEnsureRole`, `branch_role.go:28-61`), llama a `ensureRoleTaskBarrier` (`:67-92`, la barrera `task_not_bound` de C-07-e/P8: plugin activo con `items.create`, vínculo `work_item` presente y sin Task del rol, `:81-90`) y, tras un `ensure` listo, escribe `started_at` y emite `role.start` (`branch.go:327-329`).
- Lo que crea el motor hoy: **todo el plan a la vez**. Antes del sellado, solo la rama del cambio en el planning home (`TestPlanBeforeSealingHasOnlyTheChangeBranchInThePlanningHome`, `planner_test.go:228`). Tras el sellado, el primer `ensure` crea la rama del cambio de **cada repositorio de código** (`declaredRepositories(sealed)`, `engine.go:199`) y **todas** las ramas de rol de los repos con dos o más roles (`planner.go:430-446`), con independencia del rol que entre. Es decir: en multirepo las ramas de código ya no nacen al arrancar el cambio, sino en el primer `ensure` posterior al sellado; lo que NO existe es el diferido por rol.
- Consecuencia: la opción B del usuario («las ramas de código se crean cuando cada rol entra en el apply») exige un comportamiento nuevo: pasar el rol al motor y no crear (ni nombrar) las ramas que dependen de `{task_ref}` de un rol que no entra.

**5. Origen de `{ref}` y `{task_ref}`**
- `{ref}`: vínculo `kind: work_item` de `integrations.yaml` (`integrations.WorkItemLinks`, `links.go:348`), escrito por `change start --adopt` (`changestart/adopt.go:180`: `Link{Kind: "work_item", ID: doc.ID, URL: doc.URL}`). Para Azure DevOps `ID` es el entero «2525» (`ParseItemRef` acepta `^[1-9][0-9]{0,9}$`, `azuredevops/ref.go:17`), no `ado:2525`. El `external_ref` completo (`ado:2525`, `RefAlias`, `manifest/azuredevops.go:24`, `changestart/start.go:89`) solo se persiste en el staging y en `context/work-item.md`, que no se versiona (`EnsureIgnored`). Por tanto `{ref}` debe ser el `ID` del vínculo, no el `external_ref`.
- **Problema de orden**: la fase 1 de `change start` va antes del `ensure` y la fase 2 (`--adopt`, que escribe el vínculo en `integrations.yaml`) va **después** del seal. El primer `ensure` (el que crea la rama del cambio con `openspec/`) corre con el vínculo aún sin escribir; `{ref}` solo está en el staging (`<commonDir>/axiom/change-start/v1/<cambio>`, `staging.go:97-100`, `ContextDoc.ExternalRef/ID`, `:109-111`), que la fase 2 retira.
- `{task_ref}`: `integrations.TaskLinks` (`links.go:485`), vínculos `kind: task` con `Role` (`TaskLinkKind`, `links.go:35`; escritos por `BindTask`, `:366`). `hasPluginLink(links, plugin, role)` (`branch_role.go:106`) es la búsqueda por rol. `Link.ID` es texto libre hasta 128 caracteres sin controles (`validateItemID`, `links.go:179-192`), así que para otro plugin puede no ser seguro para git: hay que validarlo con `gitref.ValidBranchComponent` (`gitref.go:103`: letras ASCII, dígitos, `.`, `_`, `-`; sin `#`) y rechazar con tipado, nunca sanear en silencio.
- `{slug}`: no existe como concepto en el código; `DeriveName(tp.Prefix, id, item.Title)` (`changestart/start.go:87`) compone el nombre del cambio desde el título. Hay que definirlo (propuesta: el nombre del cambio sin el prefijo `us-<id>-`/`bug-<id>-`; o alias de `{change}` sin plugin).

**6. Catálogo de ajustes (C-06-e)**
- `editableSpecs` (`settings/catalog.go:101-118`) es una tabla de `keySpec` con `scope` `scopeBlock|scopeRepo|scopeRole` (`:52-77`). Existen ya `git.branch_prefix` (`:104`), `repositories[].branch_prefix` (`:115`) y `roles[].gate_policy` (`:117`, `scopeRole`, `multirepoOnly`). `parseKey` entiende `repositories[<ruta>].<campo>` y `roles[<id>].<campo>` (`:155-171`). Registrar `git.branch_template`, `repositories[].branch_template` y `roles[].branch_template` son tres filas `TypeString`, sin vocabulario. `engineChecked` solo aplica a enums; para el texto de la plantilla el parche no valida contra el motor, así que conviene que `patch.go` llame a `workspace.ValidateBranchTemplate` (o el validador de `workspace.Write` ya revalida al recargar; comprobar en diseño).
- Pruebas: `catalog_test.go` (`TestCatalogFiltersByTopology`, `TestCatalogDefaultsAndEffects`, `TestCatalogVocabularyIsCoherentWithParseConfig`), `patch_test.go`, `write_test.go`, `layout_test.go`, `document_test.go`, y los golden `testdata/commented.yaml`/`commented.lane-sdd.golden`; también `dashboard/settings_api_test.go` y `assets_settings_test.go`.

**7. Spec viva**
- `change-branch-lifecycle/spec.md`: último ID `RF-BRA-41` (`:2528`). Requisitos que cambian: RF-BRA-03 (precedencia de política, `:239`; entra la plantilla), RF-BRA-04 (modelo de ramas, `:289`; rama de rol con plantilla y rol que entra), RF-BRA-12 (excepciones, `:717`, si se admite `--kind`), RF-BRA-33 y RF-BRA-41 (`ensure --role`, `:2059`, `:2528`; el rol llega al motor). El «registro compartido» (RF-KOF-06, `:41`) y el congelado D-10 se citan aquí.
- `sdd-lane-kickoff/spec.md:196` RF-KOF-06 (instantánea de política git, escrita una vez): se modifica para añadir la plantilla por rama de código.
- `workspace-topology/spec.md:262` RF-TOP-09 (bloque `git:`) y `:331` RF-TOP-10 (bloque `repositories:`): entra `branch_template`. La entrada de roles de RF-TOP-04..08 si se añade la clave por rol.
- Nuevos requisitos: `prov:BRA:1..n` (los provisionales se numeran al archivar tras el 41).

**8. Trinquetes y pruebas**
- `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` (`cli/refusal_resolution_ratchet_test.go:617`): todo rechazo de producción nuevo debe nombrar su resolución (campo `Continuation`) o declararse «por diseño»; los rechazos de `branch` (`Refusal{Code, Message, Continuation}`) cumplen esto si siempre llevan continuación. Un código nuevo exige alta en `types.go:134-158` y `labels.go:53` (`CodeTaskNotBound` es el patrón).
- Deadcode ratchet (`scripts/deadcode-ratchet.sh`, job de `ci.yml:489`): toda función nueva exportada debe ser alcanzable desde `main`; la validación de plantilla debe usarse desde la carga y el planificador.
- `internal/idmigration/residues.yaml`: son citas `REQ-n.m`; no afecta si el código no cita IDs heredados. No citar IDs provisionales en comentarios de Go.
- Pruebas a ampliar: `internal/branch/planner_test.go` (helpers `embeddedInput`, `mustPlan`, `mustRefuse`, `wantNames`, `policyConfig`, `TestPlanResolvesTheBranchPolicyPerRepository`, `TestPlanMultirepoWithTwoRolesCreatesTrackingAndRoleBranches`), `branch/policy_test.go`, `branch/engine/ensure_multirepo_test.go` (`TestEnsureMultirepoSharedRepositoryOfTwoRolesGetsTheTrackingAndTheRoleBranches`), `engine/snapshot_test.go`, `engine/verdict_test.go`, `kickoff/gitsnapshot_test.go`, `workspace/gitpolicy_test.go` (`TestGitUnknownKeysAreRejected`, `TestRepositoriesInvalidBranchValuesNameTheEntryTheKeyAndTheValue`, `TestExampleGitBlockIsValidOnceUncommented`), `cli/branch_ensure_test.go` y `cli/sdd_kickoff_branch_test.go`.

### Zonas afectadas
- `internal/workspace/gitpolicy.go`, `types.go`: claves, decodificación y validación de la plantilla.
- `internal/branch/policy.go`, `planner.go`, `planner_snapshot.go`, `naming.go` (nuevo `template.go` con el motor de plantillas), `types.go`/`labels.go` (códigos de rechazo).
- `internal/branch/engine/engine.go` (recoger vínculos y `Role`), `ensure.go` (rol y diferido), `snapshot.go`, `verdict.go` (nombre esperado con plantilla).
- `internal/kickoff/gitsnapshot.go`: campo de plantilla en `GitSnapshotBranch`/`GitSnapshotRepo`.
- `internal/cli/branch.go`, `branch_role.go` (pasar `Role` al motor; la barrera ya existe).
- `internal/settings/catalog.go`: tres filas.
- `axiom.example.yaml`, y la documentación de usuario del bloque `git:` si existe.
- Spec: `change-branch-lifecycle`, `sdd-lane-kickoff`, `workspace-topology`.

### Enfoques

**A. Forma del YAML**
1. **`branch_template` escalar en `git:`, en `repositories[]` y en `roles.<id>`.** Precedencia: rol > repositorio > `git:` > sin plantilla. Encaja con `decodeStringKeys` (una clave más por lista) y con las tres plantillas de catálogo ya existentes. Pros: homogéneo con `branch_prefix`/`promotion`, editable desde Ajustes, ninguna estructura nueva. Contras: el rol tiene que validarse a mano (no pasa por `decodeStringKeys`); un rol que toca varios repositorios usa la misma plantilla en todos (y para variar por repositorio se usa la del repositorio, que es la regla de precedencia).
2. **Mapa `branch_templates: {rol: plantilla}` bajo `git:`.** Pros: todo el bloque de ramas en un sitio. Contras: `decodeStringKeys` no lo soporta (decoder propio), no encaja con el catálogo por elemento (`roles[].…`), duplica el selector de rol que ya existe en `roles:`.
3. **Plantilla única con condicionales por repositorio/rol dentro del texto.** Descartada: complica la validación y no es editable como ajuste.

**B. Dónde se resuelve el nombre**
1. **Función pura `branch.RenderBranchName(template, vars)` en el planificador**, con las variables resueltas por el motor (que lee `integrations.yaml`/staging) y entregadas en `PlanInput` (`Refs{Change, TaskByRole}`, `Kind`). Mantiene el paquete `branch` sin E/S, igual que hoy. Es la recomendada.
2. Resolver en el adaptador de CLI y pasar nombres ya hechos al motor. Contra: duplica la lógica entre `Snapshot` (seal) y `ensure`, justo lo que hoy se evita («`ensure` y la instantánea nunca discrepan», `snapshot.go:22`).

**C. Cómo se congela**
1. **Instantánea con plantilla + nombre en el registro local.** El seal guarda la plantilla de cada rama de código (campo aditivo `template` en `GitSnapshotBranch`; el nombre se omite si depende de `{task_ref}`) y, para la rama con `openspec/`, el nombre (resoluble con `{ref}` desde el staging en el seal). El primer `ensure` que resuelve el nombre de una rama de rol o de código lo escribe en el registro (`RecordBranch`), y el plan posterior lo lee del registro antes que de la plantilla (precedencia: nombre de instantánea > nombre del registro > plantilla de la instantánea > política). Sin reescribir `kickoff.yaml`. Es la recomendada. Hace falta tocar `planRepository` para leer el registro (hoy solo lo hace el archivado).
2. Reescribir `kickoff.yaml` en el `ensure` del rol. Imposible sin romper la escritura única (D-01/D-02, `seal.go:52-60`) ni la compuerta; descartado.
3. Congelar el nombre de todas las ramas en el seal. Imposible: `{task_ref}` aún no existe (las Tasks se crean al aprobar el design, C-07-e).

**D. Diferido por rol**
1. **`EnsureRequest.Role` (nuevo) y rama «pendiente».** El planificador marca una rama cuya plantilla usa `{task_ref}` como `Deferred` si el rol que la reclama no es el que entra; el motor no la crea ni la nombra y `status` la lista como pendiente. Sin `--role`, nada diferido se crea. Pros: cumple la opción B, y sigue siendo un solo `ensure` idempotente. Contras: toca `EnsureRequest`, `checkPlan`/`apply`, `describe` y el JSON de salida (campo aditivo).
2. Crear la rama en cuanto exista la Task (sin mirar el rol). Pros: sin cambio de firma. Contras: rompe «cuando cada rol entra en el apply» si `bind` ocurre antes del apply de otro rol (en `continuous`, `tasks` y `apply` encadenan).

### Recomendación
A1 + B1 + C1 + D1:
- `branch_template` escalar en `git:`, `repositories[]` y `roles.<id>` (precedencia rol > repositorio > `git:`), validado al cargar con un `workspace.ValidateBranchTemplate` (variables conocidas; chequeo del nombre resultante renderizando con valores de muestra por `gitref.ValidRefName`) y de nuevo en el `ensure` con los valores reales (`invalid_name`/`config_invalid` existentes, más un código nuevo para `{task_ref}` sin resolver cuando haga falta, p. ej. `branch_template_unresolved`, con `Continuation`).
- Un módulo `internal/branch/template.go` puro, con `Render` y `Variables(template)`. Cada valor sustituido pasa por `gitref.ValidBranchComponent` (salvo `{prefix}`, que admite `/`); `{kind}` en `feature|fix`.
- La rama del cambio con `openspec/` (planning home) solo admite `{prefix}`, `{kind}`, `{change}`, `{slug}` y `{ref}`; con `{task_ref}`/`{role}` se rechaza al cargar o en el seal (el nombre ha de existir desde el arranque).
- Regla del sin plantilla: `if template == "" { current code path }` para garantizar nombres idénticos (`prefix + "/" + change`, `changeBranch + "-" + role`).
- Una rama de código en un repo tocado por un solo rol es de tipo `change`, pero su rol efectivo es ese rol: usa la plantilla del rol y su `{task_ref}`; con `fullstack` toca todos los repos con la misma Task.
- `{ref}` sale del vínculo `work_item` de `integrations.yaml`; en el primer `ensure` (antes de la fase 2) se lee del staging de `change start` (mismo git common dir). Sin plugin o sin WI, `{ref}` sin valor es un rechazo tipado (no se sustituye por vacío).
- `{kind}`: decisión pendiente (véase preguntas). Propuesta mínima: `feature` por defecto, `fix` si la invocación lo indica (`--kind`, excepción registrada como `--prefix`), que el orquestador pasa cuando `change start` devuelve `profile: bug`.

### Riesgos
- **El criterio de «terminado» exige congelar «en la instantánea» y es inviable**: el kickoff es de escritura única. Hay que cambiarlo a «instantánea (plantilla) + registro (nombre)» en la propuesta y en RF-KOF-06.
- **El orden `ensure` → seal → adopt** deja el vínculo `work_item` sin escribir en el primer `ensure`; hay que leer `{ref}` del staging en el git common dir. Si el staging falta o está retirado y aún no hay vínculo, el `ensure` rechaza (no hay valor seguro).
- **Compatibilidad**: un binario anterior rechaza una clave desconocida en `git:` (y en `repositories[]`); mismo precedente que `promotion` (`TestGitBlockIsToleratedByThePreviousBinary`, `gitpolicy_test.go:220`). Los kickoff ya sellados no llevan plantilla: comportamiento actual sin migración, y el campo `template` de la instantánea debe ser aditivo (`omitempty`).
- **La guarda** (`expectedNames`, `verdict.go:235-304`) compone nombres con `prefix + "/" + change` en tres sitios: debe aprender a usar el nombre registrado y no recomponer la plantilla; si no, una rama con plantilla se marcará como inesperada.
- **Lectura de `integrations.yaml` ilegible** con plugin activo: la barrera ya falla cerrada (`branch_role.go:77-80`); el planificador debe hacer lo mismo para no producir nombres con `{ref}` incompleto.
- **Solape con la sección 02** (promoción, misma instantánea y mismo bloque `git:`): conviene integrar tras ella o resolver el conflicto en `gitsnapshot.go`/`gitpolicy.go` (marcado COMPARTIDO en el plan).
- **Tamaño**: la estimación M (500-800 líneas) se queda corta si entran D1 (rol al motor) y la guarda; plantear dos unidades de trabajo: (1) plantilla + política + catálogo + instantánea, (2) diferido por rol y registro.
- **Windows**: nombres reservados (`CON`, etc.) y rutas largas de worktree (`plannedWorktree` usa `in.Change`, no el nombre de rama), ya cubiertos por `ValidBranchComponent`; un `{task_ref}` largo no alarga el worktree.

### Preguntas abiertas (para propuesta)
1. `{kind}`: ¿flag `--kind feature|fix` por cambio, derivado de `profile: bug` del staging, o de la declaración de bug de `proposal.md` (no disponible al sellar)? Sin decisión, `{kind}` vale siempre `feature`.
2. `{slug}`: ¿nombre del cambio sin el prefijo del tipo de WI?
3. ¿Se admite `{role}` en la plantilla del repositorio/`git:` (se evalúa por rol) o solo en la del rol?
4. En `in_place`/`worktree`, ¿la carpeta del worktree usa el nombre del cambio (hoy) o el de la rama? Recomendación: el del cambio, sin cambios.

### Listo para propuesta
Sí. Carril SDD (proyecto en modo `continuous`, rol único `fullstack`). La propuesta debe fijar `{kind}`, el modelo de congelado (instantánea con plantilla + registro) y si D1 va en este cambio o en uno posterior.

## Key Learnings
- El kickoff es de escritura única (`kickoff/seal.go:61`); congelar el nombre «en la instantánea tras el primer `ensure`» no es posible: hay que congelarlo en el registro local y guardar solo la plantilla en la instantánea.
- `engine.Ensure` no conoce el rol (`EnsureRequest` sin `Role`): `--role` es un adaptador de CLI. Hoy el primer `ensure` posterior al sellado crea todas las ramas de código y de rol a la vez; el diferido por rol es comportamiento nuevo, no solo nombres.
- El plan ordinario no lee el registro para los nombres (solo el archivado y la guarda); hay que añadir esa precedencia a `PlanInput`.
- `{ref}` debe ser el `ID` del vínculo `work_item` (`2525`), no el `external_ref` (`ado:2525`), que no se versiona; y en el primer `ensure` solo está en el staging de `change start` porque la fase 2 va después del seal.
- No hay noción de `kind` feature|fix en `branch`; el perfil bug no está en `kickoff.yaml` ni en el seal.
- `roles[].gate_policy` es la plantilla a copiar en el catálogo (`scopeRole`); `roles` no usa `decodeStringKeys`.
