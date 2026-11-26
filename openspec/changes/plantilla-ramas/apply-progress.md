# Progreso de apply: plantilla-ramas (C-07-f)

## Tareas completadas

- [x] 1.1 Motor `nametemplate` (paquete puro `internal/branch/nametemplate`). Commit `feat(branch): motor puro de plantillas de nombre de rama` (bba16f6d4).
- [x] 2.1 Esquema y carga de `branch_template` (`GitConfig`, `RepositoryPolicy`, `RoleConfig`, `Declarations.BranchTemplatesErr`, `ValidateBranchTemplate`, `BranchTemplateWarnings`, `ConfigRefusal`), `axiom.example.yaml` y `docs/usage.md`. Commit `feat(workspace): branch_template en git, repositories y roles`.

- [x] 2.2 Tres filas `branch_template` en el catálogo de Ajustes (`git.`, `repositories[].`, `roles[].`). Commit `feat(settings): filas de branch_template en el catálogo`. RED observado (3 pruebas fallan sin las filas); GREEN: `go test ./internal/settings/ -run 'Catalog|Patch|Write'` y `go test ./internal/dashboard/ -run Settings` ok; goldens sin cambios.

- [x] 2.3 Instantánea del kickoff: `GitSnapshotBranch.Template`, `GitSnapshotRepo.PendingBranches`, `GitSnapshot.TemplateValues` y su `Validate()`. Commit `feat(kickoff): la instantánea congela plantilla, pendientes y valores`. RED observado (fallo de compilación); GREEN: `go test ./internal/kickoff/ -run 'GitSnapshot|Snapshot'` ok, `go vet ./internal/kickoff/` ok, sin plantilla no se emite ninguna clave nueva.

- [x] 3.1 `branch.ResolveChangeBranch(change, prefix, ...ChangeBranchSource)` y `ArchiveBranchFor(changeBranch)` (`change_branch.go`), adoptados por planner, planner_archive, archive_plan, verdict y autofetch_relevance conservando su orden de fuentes. Commit `refactor(branch): ResolveChangeBranch como único punto de composición`. Las pruebas existentes pasan sin tocar expectativas.

- [x] 3.2 Códigos `branch_template_unresolved`/`branch_template_not_applicable`, `PendingReasonAwaitingRole`, etiquetas, constructores en `template_refusal.go` y `Verdict.Details`. Commit `feat(branch): códigos y constructores de rechazo de plantilla`.

- [x] 3.3 Plantilla en el plan: `Policy`/`RepoPolicy` con plantilla, `PlanInput.EnteringRole/Sealing/TemplateValues/Describe`, `RepoFacts.Frozen`, `BranchPlan.Template`, `RepoPlan.Pending`, `planner_template.go` (atribución D-04, precedencia D-05, pendientes D-07, rechazos tipados, colisiones, `Describe`), archivado con plantilla, `engine/template_values.go` (`resolveTemplateValues`, `templatesInPlay`), `changestart.WorkItemFacts`, instantánea con `Template`, `PendingBranches` y `TemplateValues`, y pruebas de planner, motor y seal. Commit `feat(branch): plantilla efectiva, valores de variables y rechazos tipados`.

## Pendientes

Ninguna: 10/10 tareas hechas.

## Desviaciones

- Constantes de variable con prefijo `Var*` (`VarSlug`, etc.) en lugar de `Slug`, que chocaba con la función `Slug` del mismo paquete. Las tareas posteriores deben usar `nametemplate.VarRef`, etc.
- TDD: en 1.1 la implementación y las pruebas se escribieron juntas; en 2.1 se observó RED (fallo de compilación de las pruebas) antes de implementar.
- 2.1: `openspecHostPath` trata `specs_repository` vacío, `.` u `openspec` como entrada `.` (el desplazamiento léxico vacío del repositorio de specs es la propia carpeta maestra); solo un `specs_repository` absoluto cae a `git:`.
- 3.2: pruebas escritas junto con el código (sin RED separado). Firmas: `TemplateUnresolved(change, master, key, variable, role, lane)`, `TemplateNotApplicable(change, master, key, reason, lane)`, `TemplateInvalidName(change, master, key, field, value, cause)`, `TemplateCollision`/`TemplateCandidateCollision(change, master, repo, name, roles)`, `TemplateReservedName(change, master, repo, name, reserved)`, `IntegrationsLinksUnreadable(operation, change, master, cause)`; constantes `TemplateReason*`.
- 2.1: la sección de usuario se añadió como `## Plantillas de nombre de rama (branch_template)` en `docs/usage.md` y la lista de claves editables ya cita las tres claves, aunque las filas del catálogo llegan en 2.2 (misma PR).
- 3.3: la tarea se aplicó con TDD (RED por fallo de compilación). El tipo de `changestart.WorkItemFacts` se llama `WorkItemOrigin`. `PlanInput.Sealing` es el interruptor de «plan del sellado» (política aplica aunque haya kickoff sin instantánea). El planificador ya consume `RepoFacts.Frozen`, pero el motor aún no lo rellena (4.1). `EnteringRole` y `Describe` están en el planificador; el motor los cablea en 4.2. En ODD solo SDD escribe el vínculo `work_item`, así que `{ref}` sale del staging o del `work-item.json` adoptado.

- [x] 4.1 Registro congelado, recuperación y guarda. `RecordBranch.Template`; `engine/frozen.go` (`freezeFromRecord` rellena `RepoFacts.Frozen` en `gatherWith`, `recoverPending` con `BranchExists` solo en `refs/heads`, propagación de errores, `TemplateCandidateCollision`); `branch/planner_guard.go` (`RenderGuardBranches`, `ChangeBranchTemplated`, `FrozenFrom`); `engine/verdict_names.go` (`expectedNames` con orden D-10, render sin registro, `PendingRoles`, archivado sin `ArchivePrefix` con plantilla); `Decide` con `awaiting_role` y `Details.pendingRoles`; `registeredChangeBranch` y `newArchiveRecord` con plantilla (S9). Commit `feat(branch): nombres congelados, recuperación y veredicto con plantilla`. Sin RED separado. Desviaciones: `recoverPending` va dentro del cierre `gather` de `Ensure` y planifica con `InPlace=true`; la guarda reutiliza `repoNaming` del planificador. Pendientes: 4.2 (aquí `EnteringRole` y `Describe` siguen sin cablear), 5.1.

- [x] 4.2 Diferido por rol, `status` y CLI. `EnsureRequest.Role` → `PlanInput.EnteringRole` (en el cierre `gather` de `Ensure`, tras `recoverPending`); `status` con `Describe`; `planWithPromotion` salta repositorios con todas las ramas pendientes (la pregunta `ask` se hace en el `ensure --role` que crea la rama); `RepositoryResult.PendingBranches` / `PendingBranchResult` / `NewPendingBranchResult` (JSON aditivo `pendingBranches`, texto «Rama pendiente …»); `scopeRepository` trata un repositorio solo pendiente como `unverifiable` con `branch_missing` sin `fetch`; `internal/cli/branch.go` pasa `Role: *role` y su ayuda lo documenta; `docs/usage.md` actualizado; `TestEnsureRequestHasNoRoleField` invertido. Commit `feat(branch): ramas con task_ref diferidas a ensure --role`. Sin RED separado; GREEN observado. Pendiente: 5.1.

- [x] 5.1 Regla 3 del autofetch con plantilla (D-16): `nametemplate.LiteralPrefixes` y `templatedHandedOffChange`; el cambio cedido se reconoce por `ChangeBranch` de la instantánea de `kickoff.yaml` en cada referencia bajo el prefijo literal, con la cota `autofetchHandoffLimit`; sin prefijo literal, aviso; sin plantilla, igual que antes. Commit `feat(branch): el autofetch reconoce un cambio cedido con plantilla`. Sin RED separado; GREEN observado. Estado final: 10/10.
