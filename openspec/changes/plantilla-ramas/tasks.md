# Tareas: plantillas de nombre de rama y ramas diferidas a la entrada del rol (C-07-f)

## Previsión de entrega

| Campo | Valor |
|-------|-------|
| Tamaño estimado | XL |
| Líneas modificadas estimadas | 2.000 a 2.600 (adiciones + eliminaciones, sin generados; la mitad en pruebas) |

### Unidades de trabajo

Raíz de trabajo: `C:\repos\axiom-wt\plantilla-ramas\axiom`. Todas las verificaciones son localizadas (`go test` de los paquetes tocados con `-run`, `go vet` del paquete). Nunca la batería completa ni `go build -o`.

| Unidad | Objetivo | Commit | Comando de prueba focalizado | Arnés de ejecución | Frontera de reversión |
|--------|----------|--------|------------------------------|--------------------|-----------------------|
| 1 | Motor puro `nametemplate` | `feat(branch): motor puro de plantillas de nombre de rama` | `go test ./internal/branch/nametemplate/ && go test ./internal/gitcommon/ -run Architecture` | N/A: paquete puro sin E/S | Borrar `internal/branch/nametemplate/` |
| 2 | Esquema y carga de `branch_template` | `feat(workspace): branch_template en git, repositories y roles` | `go test ./internal/workspace/ -run 'BranchTemplate\|TestGitUnknownKeysAreRejected\|TestRepositoriesInvalidBranchValues\|TestExampleGitBlockIsValidOnceUncommented\|TestExampleRepositories'` | `workspace validate` solo vía prueba en proceso (`go test`), sin binario | Revertir `gitpolicy.go`, `types.go`, `declarations.go`, `loader.go`, `main.go`, `axiom.example.yaml`, `docs/usage.md` |
| 3 | Catálogo de Ajustes | `feat(settings): filas de branch_template en el catálogo` | `go test ./internal/settings/ -run 'Catalog\|Patch\|Write'` y `go test ./internal/dashboard/ -run Settings` | `PUT /api/settings` con `httptest` en proceso | Revertir `catalog.go` y goldens |
| 4 | Instantánea del kickoff | `feat(kickoff): la instantánea congela plantilla, pendientes y valores` | `go test ./internal/kickoff/ -run 'GitSnapshot\|Snapshot'` | N/A: la decodificación con el struct anterior se simula en la prueba | Revertir `gitsnapshot.go` |
| 5 | Helper único de la rama del cambio | `refactor(branch): ResolveChangeBranch como único punto de composición` | `go test ./internal/branch/ -run 'ResolveChangeBranch\|ArchiveBranchFor\|Plan\|Archive'` y `go test ./internal/branch/engine/ -run 'Verdict\|ExpectedNames\|Archive\|Autofetch\|HandedOff'` | Las pruebas existentes sin cambios prueban identidad byte a byte | Revertir el helper y sus cinco consumidores |
| 6 | Plantilla en el plan, valores, códigos e ODD | `feat(branch): plantilla efectiva, valores de variables y rechazos tipados` | `go test ./internal/branch/ -run 'Plan\|Template\|Collision\|Describe\|Refusal\|Label'`, `go test ./internal/branch/engine/ -run 'Snapshot\|Template\|ODD'`, `go test ./internal/changestart/ -run WorkItemFacts`, `go test ./internal/cli/ -run 'SealBranch\|KickoffBranch\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` | `Engine.Snapshot` y seal en proceso con repositorio git temporal de prueba | Revertir los ficheros de plantilla del planificador, `template_values.go`, `WorkItemFacts` |
| 7 | Registro congelado, recuperación y guarda | `feat(branch): nombres congelados, recuperación y veredicto con plantilla` | `go test ./internal/branch/ -run 'Decide\|Verdict'` y `go test ./internal/branch/engine/ -run 'Ensure\|Verdict\|ExpectedNames\|Archive\|Template'` | `ensure` + borrado de registro con repositorio git temporal | Revertir `RecordBranch.Template`, `Frozen`, `recoverPending`, `expectedNames`, `Decide` |
| 8 | Diferido por rol | `feat(branch): ramas con task_ref diferidas a ensure --role` | `go test ./internal/branch/ -run 'Pending\|Plan\|Describe'`, `go test ./internal/branch/engine/ -run 'Ensure\|Status\|Pending'`, `go test ./internal/cli/ -run 'BranchEnsure\|EnsureRole\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` | `ensure` sin rol y con `--role` sobre repositorio temporal (JSON `pendingBranches`) | Revertir `RepoPlan.Pending`, `EnsureRequest.Role`, `describe`, `prefs.go`, `cli/branch.go` |
| 9 | Autofetch con plantilla | `feat(branch): el autofetch reconoce un cambio cedido con plantilla` | `go test ./internal/branch/engine/ -run 'Autofetch\|HandedOff'` | Referencias remotas simuladas en repositorio temporal | Revertir `autofetch_relevance.go` |

## Fase 1: Cimientos puros

- [x] 1.1 **Unidad 1. Motor `nametemplate` (RED primero).**
  - Ficheros: crear `internal/branch/nametemplate/nametemplate.go` y `internal/branch/nametemplate/nametemplate_test.go`.
  - Qué: `Variable`, `Variables()`, `Parse`, `Template` (`Raw`, `IsZero`, `Uses`, `Render`), `Values`, `CheckDistinguishing`, `CheckSample`, `Slug(change, ref)` y los errores tipados de D-01/D-03. Sin E/S; solo `internal/gitref` y la biblioteca estándar. Sin escape de llaves.
  - RED previstos (matriz de amenazas, composición de argumentos): un valor por clase (`-D`, `--force`, `T#12`, `a b`, `..`, `x.lock`, `CON`) rechazado con su variable y sin saneado; plantilla `-{ref}` y `{prefix}/` rechazadas por `ValidRefName`; llaves sin cerrar, sueltas, anidadas y variable desconocida; `{prefix}/{kind}` sin variable distintiva; `Slug` (`us-2525-login`→`login`, `US-2525-login`, `us-2525`, `us-2525-`, sin `ref`, sin prefijo que case); render de los escenarios de prov:BRA:1.
  - Cubre: prov:BRA:1, prov:TOP:1 (sintaxis y variable distintiva), prov:BRA:4 (`invalid_name` por valor no seguro).
  - Verificación: `go test ./internal/branch/nametemplate/`; `go vet ./internal/branch/nametemplate/`; `go test ./internal/gitcommon/ -run Architecture`.
  - Alcanzabilidad: las funciones exportadas las consumen las tareas 2.1, 4.1 y 6.x; `Variables()` se usa en los mensajes de error. El trinquete de código muerto se comprueba al cerrar la 6.1 (hasta entonces el paquete es hoja sin consumidor, aceptado dentro de la PR única).
  - Commit: `feat(branch): motor puro de plantillas de nombre de rama`.
  - Evidencia: hecho. `go test ./internal/branch/nametemplate/` ok; `go vet` limpio; `go test ./internal/gitcommon/ -run Architecture` ok. Nota de diseño: las constantes de variable se llaman `VarPrefix`, `VarKind`, `VarChange`, `VarSlug`, `VarRole`, `VarRef`, `VarTaskRef` porque `Slug` (constante) chocaría con la función `Slug`.

## Fase 2: Esquema, catálogo e instantánea

- [x] 2.1 **Unidad 2. Esquema y carga de `branch_template` (RED primero).**
  - Ficheros: `internal/workspace/gitpolicy.go`, `internal/workspace/types.go`, `internal/workspace/declarations.go`, `internal/workspace/loader.go`, `cmd/axiom/main.go`, `internal/branch/policy.go` (solo `ConfigRefusal` incluye `BranchTemplatesErr`; los campos de política van en 6.1), `axiom.example.yaml`, `docs/usage.md`, y las pruebas en `internal/workspace/` (p. ej. `gitpolicy_template_test.go`) y `cmd/axiom/` si procede.
  - Qué: `BranchTemplate` en `GitConfig`, `RepositoryPolicy` y `RoleConfig`; claves en `gitKeys`/`repositoryKeys` y mensajes de claves admitidas; caso en `branchValidator` (canales `GitErr`/`RepositoriesErr`); `Declarations.BranchTemplatesErr` calculado en `parseLayered` tras `validateBaseConfig` e incluido en `joinLoadErrors` (rol: sintaxis/distintiva/muestra; regla de la rama con `openspec/` S11/G6: entrada `.` en monorepo, desplazamiento léxico de `lexicalDisplacement` en multirepo, «existe y declara», `specs_repository` absoluto → `git:`, aplicada también con specs in situ); `ValidateBranchTemplate`; `BranchTemplateWarnings` llamada desde `cmd/axiom/main.go` (`workspace validate`); `resolveGit` copia la plantilla.
  - Docs (RF-TOP-09): `axiom.example.yaml` con `#   branch_template: "{prefix}/{change}"` en `git:`, `#     branch_template: "feature/TASK-{task_ref}"` en `repositories:` y línea doblemente comentada en el rol `backend`, con explicación de las siete variables y de la variable distintiva; sección de usuario del bloque `git:` en `docs/usage.md` (variables, precedencia rol > repositorio > `git:`, diferido por rol, limitaciones S3/S6/D-16, binario anterior) y añadir las tres claves a la lista de claves editables de la línea 388-389.
  - RED previstos: tres claves aceptadas y desconocidas rechazadas; errores acumulados con la ruta de la clave; plantilla sin variable distintiva → `config_invalid`; `{role}`/`{task_ref}` en la plantilla que aloja `openspec/` (monorepo `.`, multirepo con y sin `branch_template`, solo `branch_prefix`, specs absoluto, specs in situ); `git:` con `{task_ref}` válido en multirepo solo si specs declara la suya; aviso de rol sin efecto en `monorepo-embedded`.
  - Cubre: prov:TOP:1, RF-TOP-03, RF-TOP-09, RF-TOP-10, RF-BRA-05.
  - Verificación: `go test ./internal/workspace/ -run 'BranchTemplate\|TestGitUnknownKeysAreRejected\|TestRepositoriesInvalidBranchValues\|TestExampleGitBlockIsValidOnceUncommented\|TestExampleRepositories'`; `go test ./internal/branch/ -run 'ConfigRefusal'`; `go vet ./internal/workspace/ ./cmd/axiom/`. `TestExampleGitBlockIsValidOnceUncommented` debe seguir en verde con los bloques descomentados.
  - Commit: `feat(workspace): branch_template en git, repositories y roles`.
  - Evidencia: hecho. RED observado (fallo de compilación de `gitpolicy_template_test.go`: campos y `BranchTemplatesErr` inexistentes) y luego GREEN. `go test ./internal/workspace/ -run 'BranchTemplate|TestGitUnknownKeysAreRejected|TestRepositoriesInvalidBranchValues|TestExampleGitBlockIsValidOnceUncommented|TestExampleRepositories'` ok; `go test ./internal/workspace/ -run 'Git|Repositories|Declarations|Example|LoadConfig|Layered|Validate'` ok; `go test ./internal/branch/ -run ConfigRefusal` ok; `go test ./cmd/axiom/ -run TestWorkspaceValidate` ok; `go vet ./internal/workspace/ ./cmd/axiom/ ./internal/branch/` limpio; `go test ./internal/gitcommon/ -run Architecture` ok. Nota: la línea del rol `backend` en `axiom.example.yaml` queda como `# # branch_template: ...` y una prueba la descomenta para validarla. `ValidateBranchTemplate` y `BranchTemplateWarnings` son alcanzables desde `main` (decodificador y `workspace validate`).

- [x] 2.2 **Unidad 3. Catálogo de Ajustes.**
  - Ficheros: `internal/settings/catalog.go`, `internal/settings/testdata/*` (goldens si enumeran claves) y pruebas en `internal/settings/` y `internal/dashboard/`.
  - Qué: tres filas `TypeString` (`git.branch_template`; `repositories[].branch_template` con `scopeRepo`+`multirepoOnly`; `roles[].branch_template` con `scopeRole`+`multirepoOnly`). Sin validador nuevo: `settings.Write` ya revalida con `workspace.ParseConfig`.
  - RED previstos: filtrado por topología; `PUT` de `repositories[].branch_template` y `roles[].branch_template` con la ruta del elemento en el error; rechazo por la API de `{role}`/`{task_ref}` en la plantilla de la rama con `openspec/` (S10).
  - Cubre: RF-TOP-21, prov:TOP:1.
  - Verificación: `go test ./internal/settings/ -run 'Catalog\|Patch\|Write'`; `go test ./internal/dashboard/ -run Settings`.
  - Dependencia: 2.1.
  - Commit: `feat(settings): filas de branch_template en el catálogo`.

- [x] 2.3 **Unidad 4. Instantánea del kickoff.**
  - Ficheros: `internal/kickoff/gitsnapshot.go` y su prueba en `internal/kickoff/`.
  - Qué: `GitSnapshotBranch.Template`, `GitSnapshotRepo.PendingBranches []GitSnapshotPendingBranch{Kind, Role, Template, Prefix}`, `GitSnapshot.TemplateValues *GitTemplateValues{Kind, Ref}`, todo `omitempty`; `Validate()` con `nametemplate.Parse` + `CheckDistinguishing`, pendiente con `role` y `{task_ref}`, `kind` ∈ {change, role}, `TemplateValues.Kind` ∈ {feature, fix}, `Ref` con `ValidBranchComponent`, `Prefix` con `ValidPrefix`.
  - RED previstos: validación de cada campo; un binario anterior (struct previo) ignora `pending_branches` y `template_values` sin error; el rechazo de `name` vacío en `branches` se mantiene; sin plantilla, el YAML es idéntico byte a byte.
  - Cubre: prov:KOF:1, RF-KOF-06.
  - Verificación: `go test ./internal/kickoff/ -run 'GitSnapshot\|Snapshot'`; `go vet ./internal/kickoff/`.
  - Dependencia: 1.1.
  - Commit: `feat(kickoff): la instantánea congela plantilla, pendientes y valores`.

## Fase 3: Helper único y plan con plantilla

- [x] 3.1 **Unidad 5. `ResolveChangeBranch` y `ArchiveBranchFor` (sin plantilla, refactor).**
  - Ficheros: crear `internal/branch/change_branch.go` y `internal/branch/change_branch_test.go`; modificar `internal/branch/planner.go` (composición de la línea ~412), `internal/branch/planner_archive.go`, `internal/branch/engine/archive_plan.go`, `internal/branch/engine/verdict.go`, `internal/branch/engine/autofetch_relevance.go`.
  - Qué: helper puro con las fuentes con nombre en el orden de cada consumidor (D-14); `ArchiveBranchFor` único que añade `-archive`; los cinco consumidores lo adoptan conservando su orden actual. Aquí solo la rama sin plantilla (composición `<prefijo>/<cambio>`); la rama con plantilla se completa en 4.1.
  - Prueba de identidad: las pruebas existentes de `planner_test.go`, archivado, veredicto y autofetch pasan sin tocar sus expectativas. RED nuevas: orden de fuentes con nombre; `ArchiveBranchFor`; composición idéntica a `prefijo + "/" + cambio`.
  - Cubre: prov:BRA:3, RF-BRA-03, RF-BRA-15.
  - Verificación: `go test ./internal/branch/ -run 'ResolveChangeBranch\|ArchiveBranchFor\|Plan\|Archive'`; `go test ./internal/branch/engine/ -run 'Verdict\|ExpectedNames\|Archive\|Autofetch\|HandedOff'`.
  - Alcanzabilidad: ambos símbolos se llaman desde el planificador, el archivado, la guarda y el autofetch.
  - Commit: `refactor(branch): ResolveChangeBranch como único punto de composición`.

- [x] 3.2 **Unidad 6a. Códigos, etiquetas y constructores de rechazo.**
  - Ficheros: `internal/branch/types.go`, `internal/branch/labels.go`, crear `internal/branch/template_refusal.go` y su prueba.
  - Qué: `CodeBranchTemplateUnresolved`, `CodeBranchTemplateNotApplicable`, `PendingReasonAwaitingRole`; etiquetas; constructores `TemplateUnresolved`, `TemplateNotApplicable` (`Details.reason` = `shared_branch` | `openspec_branch` | `odd_lane`), `TemplateInvalidName`, `TemplateCollision`, `TemplateReservedName`, `TemplateCandidateCollision`, `IntegrationsLinksUnreadable`; todos con `Continuation` no vacía (D-11).
  - RED previstos: cada código tiene etiqueta y continuación; el trinquete de rechazos de producción en `internal/cli` (`TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`) en verde con los códigos nuevos (ejercitado en 3.3, cuando existen las rutas de producción).
  - Cubre: prov:BRA:4.
  - Verificación: `go test ./internal/branch/ -run 'Refusal\|Label'`.
  - Commit: se integra con 3.3 en el commit de la unidad 6 si no compila sola con el trinquete de código muerto; en otro caso `feat(branch): códigos y constructores de rechazo de plantilla`.
  - Evidencia: `go test ./internal/branch/ -run 'Refusal|Label|Decide|Verdict'` ok; `go vet ./internal/branch/...` ok; `go build ./...` ok; trinquete `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` ok. Se añadió también `Verdict.Details`.

- [x] 3.3 **Unidad 6b. Plantilla en el plan, valores de variables, instantánea con plantilla e ODD (RED primero).**
  - Ficheros: `internal/branch/policy.go` (`Policy.BranchTemplate`, `RoleTemplates`, `RepoPolicy.BranchTemplate`/`BranchTemplateKey`, `For` hereda), `internal/branch/planner.go` (`TemplateValues`, `PlanInput.EnteringRole/TemplateValues/Describe`, `RepoFacts.Frozen`, `BranchPlan.Template`, precedencia D-05), crear `internal/branch/planner_template.go` (`effectiveTemplate`, `renderBranch`, `checkTemplateCollisions`), `internal/branch/planner_snapshot.go`, `internal/branch/planner_archive.go` (con plantilla, `Exceptions.Prefix` no aplica), `internal/branch/change_branch.go` (rama con plantilla), `internal/branch/engine/engine.go` (`gatherWith`: `needs` y `resolveTemplateValues`), crear `internal/branch/engine/template_values.go` (`templatesInPlay`, `resolveTemplateValues`), `internal/branch/engine/snapshot.go` (escribe `Template`, `PendingBranches` y `TemplateValues`; plan con `sealing=true`), `internal/changestart/staging.go` (`WorkItemFacts` exportada) y pruebas en `internal/branch/`, `internal/branch/engine/`, `internal/changestart/`, `internal/cli/sdd_kickoff_branch_test.go`.
  - Qué: `effectiveTemplate` de D-04 (incluye `fullstack`); precedencia D-05 (`--branch` > instantánea > `Frozen` > plantilla sellada o de política > defecto); `--prefix` solo alimenta `{prefix}`; R2 (`shared_branch`), R3 (`odd_lane`) y `openspec_branch` → `branch_template_not_applicable`; variable sin valor → `branch_template_unresolved`; `LinksErr` → `integrations_links_invalid` (D-11); colisiones con `checkTemplateCollisions` (entre ramas, rama de integración, `main_branch`); sin plantilla no se evalúa nada nuevo. Valores D-13 (`kind` y `ref` según sellado / sin sellar / ODD; `slug` con `Slug`; `task_ref` del vínculo `kind: task`); `changestart.WorkItemFacts(commonDir, change, adoptedDir)`. El seal planifica con `sealing=true`, propaga el rechazo y no escribe `kickoff.yaml` (S5, sin cambio de código en `cli/sdd_kickoff.go`, solo pruebas); `kind` siempre congelado, `ref` solo si se conoce; cambio sellado sin plantilla nunca aplica la de la política (D5).
  - RED previstos: atribución de D-04; precedencia con mezcla «instantánea sin nombre + registro» y «instantánea con nombre + registro distinto»; `--branch` con `--role`; colisiones; `LinksErr` con `{ref}`, `{slug}`, `{kind}` congelados o no; `{ref}` del staging antes de `--adopt` y vínculo después; `kind` sellado sin `template_values` = `feature` (G9); ODD con `{ref}` del vínculo, del staging y sin work item; seal que propaga `branch_template_not_applicable` y `branch_template_unresolved` sin escribir `kickoff.yaml`; `WorkItemFacts` (staging, documento adoptado, ninguno); sin plantilla, el plan y la instantánea son idénticos byte a byte.
  - Cubre: prov:BRA:1, prov:BRA:2, prov:BRA:3, prov:BRA:4, prov:KOF:1, RF-BRA-03, RF-BRA-04, RF-BRA-14, RF-BRA-15, RF-KOF-06.
  - Verificación: `go test ./internal/branch/ -run 'Plan\|Template\|Collision\|Describe\|Snapshot\|Archive\|Refusal\|Label'`; `go test ./internal/branch/engine/ -run 'Snapshot\|Template\|ODD'`; `go test ./internal/changestart/ -run WorkItemFacts`; `go test ./internal/cli/ -run 'SealBranch\|KickoffBranch\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'`; `go vet` de los paquetes tocados; `scripts/deadcode-ratchet.sh` (funciones nuevas alcanzables desde `main`: `nametemplate.*`, `ValidateBranchTemplate`, `ResolveChangeBranch`, `WorkItemFacts`, constructores).
  - Dependencias: 1.1, 2.1, 2.3, 3.1, 3.2.
  - Commit: `feat(branch): plantilla efectiva, valores de variables y rechazos tipados`.
  - Evidencia: hecho (parcial en lo conectado a 4.1/4.2, ver abajo). RED observado (fallo de compilación de `planner_template_test.go`: `PlanInput.TemplateValues` inexistente) y luego GREEN. `go test ./internal/branch/` ok (planner, instantánea, archivado, etiquetas y rechazos); `go test ./internal/branch/engine/ -run 'Snapshot|Template|ODD|Odd'` ok y `go test ./internal/branch/engine/` completo ok; `go test ./internal/changestart/ -run WorkItemFacts` ok; `go test ./internal/cli/ -run 'KickoffSeal|SealBranch|KickoffBranch|TestBranch|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` ok; `go vet` de los paquetes tocados limpio; `bash scripts/deadcode-ratchet.sh` solo señala `TemplateCandidateCollision`, que consume `recoverPending` en 4.1. Desviaciones: el tipo de `WorkItemFacts` se llama `WorkItemOrigin` (un tipo y una función no pueden compartir nombre); `PlanInput.Sealing` distingue el plan del sellado (kickoff ya presente pero sin instantánea) del de un cambio sellado; el orden `--branch` > instantánea del plan ya es el de D-05 (las pruebas existentes lo toleran); en ODD `{ref}` sale del staging y del `work-item.json` adoptado (solo SDD escribe el vínculo `work_item`). Pendiente para 4.1: leer `Frozen` del registro en `gatherWith` (el planificador ya lo consume), `registeredChangeBranch` con plantilla y la guarda `expectedNames`.

## Fase 4: Registro, guarda y diferido

- [x] 4.1 **Unidad 7. Registro congelado, recuperación y guarda (RED primero).**
  - Ficheros: `internal/branch/types.go` (`RecordBranch.Template`), `internal/branch/engine/ensure_registry.go` (`recordRepository` copia `BranchPlan.Template`), `internal/branch/engine/engine.go` (`Frozen` por identidad del git common dir), `internal/branch/engine/ensure.go` (`recoverPending`), `internal/branch/verdict.go` (`ExpectedNames.PendingRoles`, `Verdict.Details`, caso `awaiting_role` en `Decide`), `internal/branch/engine/verdict.go` (`expectedNames` con orden D-10, render sin registro, archivado sin `ArchivePrefix` con plantilla), `internal/branch/engine/archive_plan.go`, `internal/branch/engine/archive.go` (no guarda `ArchivePrefix` con plantilla), y pruebas en esos paquetes.
  - Qué: el registro congela el nombre con `Template`; `recoverPending` con `BranchExists` (error ≠ «no existe» se propaga; dos candidatos iguales → `invalid_name`; existe → `Frozen` y un único replan); la guarda calcula el nombre esperado igual que el plan, también sin registro (`feature/TASK-1235` correcto); `awaiting_role` → `not_applicable` con `Continuation` `--role`; plantilla irresoluble → `*RefusalError` (falla cerrado).
  - RED previstos: congelado y lectura del registro aunque cambie la plantilla; recuperación con registro borrado, error de `BranchExists`, candidatos repetidos y Task desvinculada (G5); `ensure --role` con `{task_ref}` = `-D` no crea rama, worktree ni registro (matriz de amenazas); veredicto con nombres de plantilla sin registro; `awaiting_role` con uno y varios roles y `Details`; repositorio con un rol dentro y otro pendiente; rama inexistente sin pendiente sigue `wrong_branch`; `ArchivePrefix` ignorado con plantilla; `registeredChangeBranch` con y sin plantilla; sin plantilla, el registro y el veredicto no cambian.
  - Cubre: prov:BRA:3, prov:BRA:5, RF-BRA-15, RF-BRA-19, RF-BRA-03, RF-BRA-22, RF-KOF-07.
  - Verificación: `go test ./internal/branch/ -run 'Decide\|Verdict'`; `go test ./internal/branch/engine/ -run 'Ensure\|Verdict\|ExpectedNames\|Archive\|Template'`; `go vet ./internal/branch/...`.
  - Dependencia: 3.3.
  - Commit: `feat(branch): nombres congelados, recuperación y veredicto con plantilla`.
  - Evidencia: hecho. Ruta: inline. TDD: no se observó RED separado (las pruebas de `frozen_test.go` y `verdict_template_test.go` se escribieron tras la implementación); GREEN observado. `go test ./internal/branch/ -run 'Decide|Verdict|Refusal|Label|Plan|Archive|Template'` ok; `go test ./internal/branch/engine/ -run 'Verdict|ExpectedNames|Registry|Record|Recover|Archive|Template'` ok; pruebas nuevas ok (congelado aunque cambie la plantilla, recuperación sin registro, error de `BranchExists` propagado, candidatos repetidos `invalid_name`, guarda sin registro con `feature/TASK-1235`, `awaiting_role`, ODD con y sin `{ref}`, `ArchivePrefix` ignorado con plantilla, `registeredChangeBranch` con y sin plantilla); `go vet ./internal/branch/...` limpio; `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` ok; `bash scripts/deadcode-ratchet.sh`: «no new unreachable functions» (`TemplateCandidateCollision` ya tiene consumidor en `recoverPending`). Desviaciones: `recoverPending` se invoca dentro del cierre `gather` de `Ensure` (tras cada lectura, también la posterior a `apply`) y planifica con `InPlace=true` solo para leer nombres; la guarda se apoya en `branch.RenderGuardBranches` (mismo `repoNaming` que el plan) en vez de duplicar el render; el orden de fuentes de la guarda pasa a D-10 (`--branch` registrada, instantánea, registro).

- [ ] 4.2 **Unidad 8. Diferido por rol, `status` y CLI (RED primero).**
  - Ficheros: `internal/branch/engine/ensure.go` (`EnsureRequest.Role` → `EnteringRole`), `internal/branch/planner.go` / `internal/branch/planner_template.go` (`RepoPlan.Pending`, `PendingBranch`, `--branch` sobre rama pendiente), `internal/branch/engine/prefs.go` (`planWithPromotion` salta repositorios sin ramas), `internal/branch/engine/status.go` (`Describe`, D-15), `internal/branch/result.go` (`PendingBranchResult`, `RepositoryResult.PendingBranches`, líneas de texto), `internal/cli/branch.go` (pasa `Role: *role`), y pruebas en `internal/branch/`, `internal/branch/engine/` e `internal/cli/` (incluidos `internal/branch/scope_test.go` para la pregunta abierta).
  - Qué: rama con `{task_ref}` sin nombre congelado, sin `--branch` y de rol ≠ `EnteringRole` queda pendiente (no se crea, nombra, registra, mide, coloca ni materializa worktree); con `EnteringRole` igual se renderiza y entra en `Branches`; promoción `ask` en el `ensure --role` que crea la rama; `ensure --role fullstack` válido; `status` con `Describe` lista pendientes con motivo y continuación sin mutar; `pendingBranches` en JSON y texto, aditivo.
  - RED previstos: matriz de amenazas: una rama pendiente de un rol que no entra con `origin/<candidato>` traída no se crea en `ensure` sin `--role` ni con `--role` de otro; `--branch` sobre rama del cambio pendiente la crea; `--branch` nunca nombra rama de rol; la entrada de un rol no toca a otro; `ensure --role` idempotente; segundo clon con `ensure --role`, sin registro y con rama publicada (S2); `status` con `integrations.yaml` ilegible lista la entrada con código sin fallar; JSON `pendingBranches`; `ensure --role` pasa el rol y la barrera `task_not_bound` precede a la creación.
  - Pregunta abierta del diseño: confirmar con `scope_test.go` que `axiom branch scope` con ramas pendientes devuelve `unverifiable` con `branch_missing` sin código nuevo; si no, ajustar dentro de esta tarea.
  - Cubre: prov:BRA:5, RF-BRA-33, RF-BRA-16, RF-BRA-27, RF-BRA-28, RF-BRA-29, RF-BRA-30, RF-BRA-21, RF-INT-02.
  - Verificación: `go test ./internal/branch/ -run 'Pending\|Plan\|Describe'`; `go test ./internal/branch/engine/ -run 'Ensure\|Status\|Pending'`; `go test ./internal/cli/ -run 'BranchEnsure\|EnsureRole\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'`; `go vet ./internal/branch/... ./internal/cli/`; `scripts/deadcode-ratchet.sh`.
  - Dependencia: 4.1.
  - Commit: `feat(branch): ramas con task_ref diferidas a ensure --role`.

## Fase 5: Autofetch

- [ ] 5.1 **Unidad 9. Regla 3 del autofetch con plantilla (RED primero).**
  - Ficheros: `internal/branch/engine/autofetch_relevance.go` y su prueba.
  - Qué: con plantilla efectiva en la rama del cambio del repositorio de specs, `handedOffChange` lista las referencias remotas bajo el prefijo literal (`{prefix}/` → prefijo efectivo; `{kind}/` → `feature/` y `fix/`), con la cota `autofetchHandoffLimit`, lee `TreeDirs` y `FileAt` del `kickoff.yaml` y toma el cambio cuya instantánea nombra esa rama; se detiene en el segundo candidato; sin prefijo literal no aplica y deja un aviso en `result.Warnings`. Sin plantilla, igual que hoy.
  - RED previstos: cambio cedido `feature/…` reconocido por la instantánea; dos candidatos; plantilla sin prefijo literal (aviso); sin plantilla (igual); la regla 3 con plantilla no crea ninguna rama local (matriz de amenazas); el prefijo literal pasa `plainArgument`.
  - Cubre: prov:BRA:3, RF-BRA-12.
  - Verificación: `go test ./internal/branch/engine/ -run 'Autofetch\|HandedOff'`; `go vet ./internal/branch/engine/`; `scripts/deadcode-ratchet.sh`.
  - Dependencia: 3.3.
  - Commit: `feat(branch): el autofetch reconoce un cambio cedido con plantilla`.

## Orden de implementación y paralelismo

Secuencial por dependencias: 1.1 → 2.1 → 2.2 / 2.3 (2.2 y 2.3 son independientes entre sí tras 1.1 y 2.1, pero con commit por tarea y un solo escritor se aplican en orden) → 3.1 → 3.2 → 3.3 → 4.1 → 4.2 y 5.1 (4.2 y 5.1 son independientes entre sí tras 3.3/4.1; sin escritores paralelos, en orden). Cada tarea es un commit.

## Ruta por tarea

| Tarea | Ruta | Evidencia del disparador |
|-------|------|--------------------------|
| 1.1 | Inline | Paquete nuevo y puro; una unidad de comportamiento coherente, escritura inline por el único escritor |
| 2.1 | Inline | Cambios dispersos pero acoplados (carga, ejemplo, docs); ningún motivo de paralelismo |
| 2.2 | Inline | Unidad pequeña (3 filas + goldens) |
| 2.3 | Inline | Unidad pequeña y acotada en `kickoff` |
| 3.1 | Inline | Refactor mecánico guiado por pruebas existentes |
| 3.2 | Inline | Constructores y etiquetas, unidad pequeña |
| 3.3 | Inline por defecto; delegar un escritor solo si el contexto del padre supera ~150k tokens (protección de contexto) | Es la unidad más grande; no es paralelizable por depender de 3.1 y 3.2 |
| 4.1 | Inline (mismo criterio de contexto que 3.3) | Depende de 3.3 |
| 4.2 | Inline (mismo criterio de contexto que 3.3) | Depende de 4.1 |
| 5.1 | Inline | Unidad aislada y pequeña |

Las verificaciones por tarea se ejecutan inline y son localizadas; la batería completa y los trinquetes de CI (arquitectura de `internal/gitcommon`, código muerto) corren en CI, salvo `go test ./internal/gitcommon/ -run Architecture` en 1.1 y `scripts/deadcode-ratchet.sh` donde se indica.

## Commits con cambios pedidos

