# Informe de verificación: plantilla-ramas (C-07-f)

Rol `fullstack` (informe del rol y del cambio). Ámbito: `git diff origin/main...HEAD` de `inc/plantilla-ramas` (73 ficheros, 14 commits: 10 de unidad de trabajo y 4 de publicación de documentos). Cambio no ISO. Strict TDD desactivado (`strict_tdd: false`).

## Estado de las tareas (observado)

10/10 marcadas hechas en `tasks.md` y `apply-progress.md` (1.1, 2.1, 2.2, 2.3, 3.1, 3.2, 3.3, 4.1, 4.2, 5.1); sin pendientes. No se reescribe ningún estado.

## Comandos ejecutados (todos localizados, desde el worktree)

| Comando | Resultado |
|---|---|
| `go test ./internal/branch/nametemplate ./internal/branch -count=1 -run 'Template\|Pending\|Plan\|Frozen\|Recover\|Verdict\|Decide\|ResolveChangeBranch\|ArchiveBranchFor\|Archive\|Describe\|Refusal\|Label\|Collision\|Guard\|Literal'` | ok (1.0 s, 3.3 s) |
| `go test ./internal/branch/engine -count=1 -run 'Template\|Pending\|Frozen\|Recover\|Verdict\|Snapshot\|Ensure\|Status\|Scope\|Autofetch\|HandedOff\|Archive\|ExpectedNames\|ODD'` | ok (374.5 s), exit 0 |
| `go test ./internal/workspace -run 'BranchTemplate\|Example\|Git\|Repositories'` | ok (14.3 s) |
| `go test ./internal/settings -run 'BranchTemplate\|Catalog\|Write\|Patch'` | ok |
| `go test ./internal/kickoff -run 'GitSnapshot\|Seal\|Snapshot'` | ok |
| `go test ./internal/changestart -run WorkItemFacts` | ok |
| `go test ./internal/cli -run 'Branch\|Kickoff\|SealBranch\|EnsureRole\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` | ok (86.1 s) |
| `go test ./cmd/axiom -run TestWorkspaceValidate` | ok |
| `go test ./internal/gitcommon/ -run Architecture` | ok |
| `go test ./internal/idmigration/ -run Residue` | ok (sin tokens `REQ-` heredados en el código ni en los artefactos del cambio) |
| `bash scripts/deadcode-ratchet.sh` | «no new unreachable functions», exit 0 |
| `gofmt -l internal cmd` | sin salida |
| `go vet` de `internal/branch/...`, `workspace`, `settings`, `kickoff`, `changestart`, `cli`, `cmd/axiom` | exit 0, sin salida |
| `go build ./...` | exit 0 |

No se ejecutó la batería completa ni `go build -o`/`go install` (regla del encargo). `internal/dashboard -run Settings` (citado en la unidad 3) no se relanzó: su paquete no está en el diff y lo cubre la CI.

## Cumplimiento de la spec

Contraste por requisito entre los deltas y el diff, con las pruebas que lo ejercitan (todas verdes arriba):

- prov:BRA:1 (motor y variables): `nametemplate` (`Parse`, `Render`, `Slug`, `CheckDistinguishing`, `LiteralPrefixes`) con `nametemplate_test.go` y `literal_prefix_test.go`. Cumple.
- prov:BRA:2 (origen de los valores): `engine/template_values.go` + `changestart.WorkItemFacts`; pruebas `template_values_test.go`, `work_item_facts_test.go`, `TestVerdictForTheODDLaneRendersFromTheStagingOrFailsClosed`, `TestStatusWithUnreadableIntegrationsReportsTheCodeOnTheEntry`. Cumple.
- prov:BRA:3 (plantilla efectiva y precedencia): `planner_template.go`, `change_branch.go`; `planner_template_test.go`, `change_branch_test.go`, `TestRegisteredChangeBranchWithAndWithoutTemplate`, `TestExpectedNamesIgnoresTheArchivePrefixWithATemplate`. Cumple.
- prov:BRA:4 (rechazos tipados): `template_refusal.go` con constructores y códigos `branch_template_unresolved`/`branch_template_not_applicable`; `template_refusal_test.go`, trinquete de rechazos con resolución (`TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` verde) y `sdd_kickoff_branch_test.go` (el seal propaga sin escribir `kickoff.yaml`). Cumple.
- prov:BRA:5 (diferido por rol): `RepoPlan.Pending`, `EnsureRequest.Role`, `prefs.go`, `status`; `ensure_role_test.go` (entrada del rol, idempotencia, `status` sin mutar, `ask`, alcance `branch_missing`) y `frozen_test.go` (recuperación sin registro). Cumple.
- prov:KOF:1 (instantánea del kickoff): `kickoff/gitsnapshot.go` (`Template`, `PendingBranches`, `TemplateValues`, `Validate`); `gitsnapshot_test.go`. Cumple.
- prov:TOP:1 (`branch_template` en `axiom.yaml`): `gitpolicy.go`, `types.go`, `declarations.go`, `loader.go`, catálogo de Ajustes, `workspace validate`; `gitpolicy_template_test.go`, `catalog_test.go`, `write_test.go`, `workspace_validate_branch_template_test.go`. Cumple.
- MODIFIED (RF-BRA-03, 04, 14, 15, 16, 19): `policy.go`, `verdict.go`/`verdict_names.go`, `frozen.go`, `result.go`; `policy_test.go`, `verdict_template_test.go`, `branch_role_template_test.go` y las pruebas existentes de `branch`/`cli` actualizadas (`branch_test.go`, `branch_role_test.go`). Sin plantilla los nombres son idénticos a los actuales (las pruebas existentes pasan sin tocar sus expectativas, salvo la inversión declarada de `TestEnsureRequestHasNoRoleField`). Cumple.

Límite declarado: la comprobación fue por mapeo de requisito a código y a pruebas verdes del paquete tocado, no una ejecución uno a uno de los ~150 escenarios.

## Conformidad con el diseño

La tabla «Cambios de ficheros» del diseño coincide con el diff: ficheros creados (`nametemplate.go`, `change_branch.go`, `planner_template.go`, `template_refusal.go`, `template_values.go`; y los adicionales `frozen.go`, `verdict_names.go`, `planner_guard.go` que materializan D-09/D-10) y modificados (políticas, planificador, veredicto, autofetch D-16, kickoff, changestart, cli, ajustes, docs, `axiom.example.yaml`). Las desviaciones de `apply-progress.md` (constantes `Var*`, `WorkItemOrigin`, `recoverPending` dentro del cierre `gather`) son menores y no cambian el contrato observable. Sin hallazgos de divergencia.

## Cumplimiento técnico

Fuentes consultadas: (a) skills declaradas por apply: no constan en `skill_resolution` de `apply-progress.md` (no aplicable); (b) `go-testing` por los `_test.go` tocados: pruebas con tablas y repositorios git temporales, coherentes; (c) `technicalContext`: no aplicable (no proyectado); (d) `openspec/config.yaml` (`rules.verify`: tests localizados, `go vet`, contraste con escenarios): ejecutado, ver tabla.

- SUGGESTION · openspec/config.yaml / explore.md (no citar IDs provisionales en comentarios de Go) · `internal/branch/planner_template.go`, `internal/cli/sdd_kickoff_branch_test.go`, `internal/settings/write_test.go`, `internal/workspace/gitpolicy_template_test.go` · comentarios que citan `prov:KOF:1`, `prov:BRA:2`, `prov:TOP:1`; el integrador de archivado asigna números definitivos y el comentario quedará con un identificador provisional obsoleto. No lo detecta el trinquete de residuos (pasa).

Sin hallazgos CRITICAL ni WARNING. Comprobaciones mandatorias (`rules.verify`) ejecutadas y verdes.

## Avisos de contexto

`technicalContext.maintained` no declarado: no aplica.

## Hallazgos

- CRITICAL: 0
- WARNING: 0
- SUGGESTION: 1 (comentarios con IDs provisionales, arriba)

## Siguiente paso recomendado

`sdd-archive`.

**Estado:** limpio
