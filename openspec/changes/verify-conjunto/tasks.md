# Tareas: Verificación conjunta del cambio antes de la PR a la rama principal (C-02-b)

Rol único `fullstack` (`tasks.md`). Entrega `single-pr`: un commit de unidad de trabajo por tarea, con sus pruebas y documentación. TDD: RED (prueba que falla) antes de cada producción. Prohibido compilar binarios sueltos de `axiom`: la CLI se prueba en proceso con `cli.RunBranch`. Decisión P1: un repositorio declarado que no existe en esta máquina es `unverifiable` / `branch_missing` con `details.path` y la ruta en el mensaje; el dominio de motivos no se amplía.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | L |
| Estimated changed lines | 1.300-1.400 (sin goldens) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| 1 | Primitivas git `RevList`, `CommitFiles`, `BranchCreation` | `feat(branch): añadir lecturas git de alcance conjunto` | `go test ./internal/branch/gitcli/ -run 'RevList\|CommitFiles\|BranchCreation\|ClosedList'` | N/A: se prueba con git real temporal | `internal/branch/gitcli/scope*.go` y puertos | 
| 2 | Contrato `axiom.branch-scope/v1` y `SummarizeScope` | `feat(branch): definir el contrato branch-scope y la oferta` | `go test ./internal/branch/ -run 'Scope\|Labels\|Code'` | N/A: tipos puros | `internal/branch/scope*.go` y etiquetas |
| 3 | `Engine.Scope`: rechazos, exentos, referencias, P1 | `feat(branch): calcular el esqueleto del alcance conjunto` | `go test ./internal/branch/engine/ -run 'Scope'` | N/A: git real temporal | `internal/branch/engine/scope.go` |
| 4 | Punto de partida por primer padre, commits, ficheros y fusión (E1, E2) | `feat(branch): delimitar commits propios y ficheros del cambio` | `go test ./internal/branch/engine/ -run 'ScopeRelease\|ScopeBranch'` | N/A: git real temporal | Cambios de la unidad en `engine/scope.go` |
| 5 | Avance rápido por reflog (E3) y multirepo (E4) | `feat(branch): resolver avance rápido y multirrepo en scope` | `go test ./internal/branch/engine/ -run 'ScopeFastForward\|ScopeMultirepo'` | N/A: git real temporal | Cambios de la unidad en `engine/scope.go` |
| 6 | Verbo `axiom branch scope` en la CLI | `feat(cli): añadir el verbo axiom branch scope` | `go test ./internal/cli/ -run 'BranchScope\|BranchHelp' && go test ./cmd/axiom/ -run BranchHelp` | `cli.RunBranch` en proceso | `internal/cli/branch_scope*.go` y ayudas |
| 7 | `sdd-verify` en modo conjunto | `feat(sdd-verify): añadir la verificación conjunta` | `go test ./internal/assets/ -run 'Asset\|JointVerification'` | N/A: texto de skill con pines | `sdd-verify/**` |
| 8 | Doctrina del orquestador (punto d y estados de `integration`) | `feat(sdd): declarar el estado de la verificación conjunta` | `go test ./internal/assets/ -run 'IndependentReviewBoundary\|StopExits'` | N/A: texto con pines | `sdd-orchestrator-sections.md` |
| 9 | No bloqueo en `sddstatus` | `test(sddstatus): fijar que la verificación conjunta no bloquea` | `go test ./internal/sddstatus/ -run JointVerifyNonblocking` | En proceso | Fichero de prueba nuevo |
| 10 | Documentación y pines de docs | `docs: documentar axiom branch scope` | `go test ./internal/workspace/ -run DocsContract` | N/A | Docs y pines |
| 11 | Pines, goldens y comprobación final | `chore: regenerar goldens y pines de verificación conjunta` | `go test ./...` y `go vet ./...` | Revisión del diff de goldens | Goldens y hash de Kilocode |

Tests lentos (señalar para segundo plano): `go test ./internal/branch/engine/` (git real), `go test ./... -update` y `go test ./...` final.

## Phase 1: Fundamentos de `internal/branch`

- [x] 1.1 (Unidad 1, RED) Crear `internal/branch/gitcli/scope_test.go` con git real: `RevList` (con y sin `--first-parent`), `CommitFiles` (ruta con espacio y no ASCII, renombrado con dos rutas, commit de fusión con `merge: true`), `BranchCreation` (rama creada con `branch` válida; adoptada de `origin/<rama>` no válida). Incluir en `internal/branch/gitcli/ops_test.go` la ampliación de `TestReadsLaunchOnlyTheClosedListOfReadSubcommands` con `log` y `reflog show`.
  Cubre: REQ-25.51, REQ-25.53
- [x] 1.2 (Unidad 1, GREEN) Añadir `RevEntry` y `CommitPaths` en `internal/branch/types.go`; `RevList`, `CommitFiles`, `BranchCreation` en `internal/branch/ports.go`; implementarlas en `internal/branch/gitcli/scope.go` con `gitcommon.Run`, `plainArgument` y `commandFailed`; actualizar el doc del paquete en `internal/branch/gitcli/ops.go`. Añadir los stubs a `fakeGit` en `internal/branch/engine/pending_test.go` y las tres lecturas a `gitReader` en `internal/branch/engine/engine.go`.
  Cubre: REQ-25.51, REQ-25.53
- [x] 1.3 (Unidad 2, RED) Crear `internal/branch/scope_test.go`: tabla de `SummarizeScope` (precedencia `unverifiable` > `pending_merge` > `nothing_to_verify` > `offerable`; `empty` y `exempt` no pendientes; `pending` nunca `null`), JSON estable, etiquetas de los tokens nuevos y `CodeScopeNotApplicable` en `internal/branch/result_test.go`.
  Cubre: REQ-25.53, REQ-25.54
- [x] 1.4 (Unidad 2, GREEN) Crear `internal/branch/scope.go` (`ScopeSchema`, `ScopeResult`, `ScopeSummary`, `ScopePending`, `RepositoryScope`, `ScopeRef`, `ScopeStart`, `ScopeCommit`, `SummarizeScope`, `WriteJSON`, `WriteText`, `ExitCode`); `CodeScopeNotApplicable` en `internal/branch/types.go`; etiquetas en `internal/branch/labels.go`.
  Cubre: REQ-25.52, REQ-25.53, REQ-25.54

## Phase 2: Motor `Engine.Scope`

- [x] 2.1 (Unidad 3, RED) Crear `internal/branch/engine/scope_test.go` con: sin kickoff y sin documento ODD → `kickoff_not_sealed`; con `odd/tasks/<cambio>.md` → `scope_not_applicable`; kickoff sin bloque `git:` → `scope_not_applicable`; `config_invalid` ante kickoff ilegible; repositorio de specs in situ → `exempt` sin lecturas git; instantánea solo de specs → `nothing_to_verify`; rama sin referencia local ni `origin/<rama>` → `unverifiable`/`branch_missing` con mensaje, `continuation` de `fetch` citada con `pathquote.Quote` y sin ejecutar `fetch`; base o integración ausente → `base_missing` con `details.missingRef`; repositorio de `plan.Skipped` (P1) → `unverifiable`/`branch_missing` con `details.path` y ruta en el mensaje; `--cwd` relativa resuelta a absoluta.
  Cubre: REQ-25.51, REQ-25.53
- [x] 2.2 (Unidad 3, GREEN) Crear `internal/branch/engine/scope.go` con `Engine.Scope(ctx, cwd, change)`: `gather(ctx, "scope", cwd, change, LaneSDD)`, `planWithPromotion` no interactivo, rama `KindChange` por `RepoPlan`, resolución de punta (`refs/heads` → `origin/<rama>`), `resolveBase`, `integrationReferences`, exentos y P1; todo git con `dir = RepoPlan.GitRoot` y referencias por `plainArgument`.
  Cubre: REQ-25.51, REQ-25.53
- [ ] 2.3 (Unidad 4, RED) Ampliar `internal/branch/engine/scope_test.go` con E1 `release` y commit ajeno `F` (sin fusionar, `--no-ff`, squash: mismos commits y ficheros, sin `F`, sin `otro.go` ni `openspec/changes/<c>/`; `merged` tras fusionar, squash solo con `merge-tree` disponible), E2 `branch` con `main_branch` (rama fusionada en `develop`; `develop` fusionada dentro de la rama → `F` en `excluded` y aviso `foreign_commits_possible` cuando aplique), `fork_point_undetermined` en historia sin relación, y en E1 el observador de `gitcommon` sin operaciones remotas con referencias, índice y worktrees idénticos antes y después, con cambios con y sin stage y un fichero sin seguimiento fuera de `files`.
  Cubre: REQ-25.52, REQ-25.53, REQ-25.54
- [ ] 2.4 (Unidad 4, GREEN) En `internal/branch/engine/scope.go`: `firstParentFork` sobre `RevList`, revisiones propias y excluidos según la tabla de D5, `CommitFiles` con la unión de rutas sin `<WorkspaceRel>/openspec/changes/<cambio>/`, resultado `ok`/`empty`, `mergeLayers` solo para entradas `ok`, `SummarizeScope`.
  Cubre: REQ-25.52, REQ-25.53, REQ-25.54
- [ ] 2.5 (Unidad 5, RED) Ampliar `internal/branch/engine/scope_test.go` con E3 (rama local fusionada por avance rápido → `ok` con `start.source: reflog`; rama recién creada → `empty`; rama solo remota fusionada por avance rápido → `fork_point_undetermined`) y E4 multirepo (`api` fusionada → `ok`; `web` sin rama → `branch_missing` sin `fetch`; specs → `exempt`; `offer: unverifiable`; subcaso `web` presente sin fusionar → `pending_merge` con `web` en `pending`).
  Cubre: REQ-25.53, REQ-25.54
- [ ] 2.6 (Unidad 5, GREEN) En `internal/branch/engine/scope.go`: lectura de `BranchCreation` solo cuando `fB == tip` y la base es la integración, validez por asunto `branch: Created from ` y origen distinto de la copia remota, reglas `C == tip` → `empty` e `IsAncestor(C, tip)` → `start = C`.
  Cubre: REQ-25.52, REQ-25.53

## Phase 3: CLI

- [ ] 3.1 (Unidad 6, RED) Crear `internal/cli/branch_scope_test.go` con `cli.RunBranch`: `--fetch`, bandera desconocida, `--change` ausente o nombre inválido → salida `2` sin leer repositorios; sin kickoff → `kickoff_not_sealed` (`1`); documento ODD → `scope_not_applicable` (`1`); `--json` con `schema: axiom.branch-scope/v1`, `--cwd` relativa. Ampliar el escenario «La ayuda lista los cinco verbos» en `internal/cli/branch_test.go` y añadir `scope` a `implemented` en `cmd/axiom/branch_help_test.go`.
  Cubre: REQ-25.1, REQ-25.2, REQ-25.51
- [ ] 3.2 (Unidad 6, GREEN) Crear `internal/cli/branch_scope.go` (`runBranchScope`, `branchScopeHelp` con `--change`, `--cwd`, `--json`; sin `--fetch` ni `--lane`); `case "scope"`, ayuda de cinco verbos y mensaje «los disponibles son ensure, status, scope, done y guard» en `internal/cli/branch.go`; línea `branch scope` en la ayuda global de `cmd/axiom/main.go`.
  Cubre: REQ-25.1, REQ-25.2, REQ-25.51

## Phase 4: Skill y doctrina

- [ ] 4.1 (Unidad 7, RED) En `internal/assets/assets_test.go` registrar `joint-verification.md` en la lista de assets y añadir los pines: `joint-verification.md` contiene `axiom branch scope`, `git -C`, «Verificación conjunta», `**Resultado conjunto:**`, `branch_missing` y «consulta no disponible», y no contiene `ls-files --others` ni rangos `...HEAD` (`assertNoHeadRangeScope`); `sdd-verify/SKILL.md` contiene la regla y la fila de *Decision Gates* del modo conjunto.
  Cubre: REQ-28.4, REQ-28.41
- [ ] 4.2 (Unidad 7, GREEN) Crear `internal/assets/skills/sdd-verify/references/joint-verification.md` (D11); modificar `internal/assets/skills/sdd-verify/SKILL.md` (regla, fila de *Decision Gates*, entrada en *References*) y `internal/assets/skills/sdd-verify/references/report-format.md` (ronda fechada de solo añadir, sin `verdict:` ni `**Estado:**`).
  Cubre: REQ-28.4, REQ-28.41
- [ ] 4.3 (Unidad 8, RED) Ampliar `internal/assets/independent_review_boundary_test.go` con los pasajes nuevos (vocabulario de frontera limpio) y los pines: «Four points, and only these.», `axiom branch scope`, la frase de `integration`, los tres estados (hecha, ofrecida, no verificable), no ejecutar `fetch` y no bloqueo; comprobar que las frases fijadas de `integration` de `internal/assets/assets_test.go` y `orchestrator_stop_exits_test.go` siguen intactas.
  Cubre: REQ-28.1, REQ-28.42, REQ-43.32
- [ ] 4.4 (Unidad 8, GREEN) Modificar `internal/assets/skills/_shared/sdd-orchestrator-sections.md`: frase de `integration` tras la de `Commit Review` (sin reflujo de la línea 81), «Four points, and only these.» con el punto (d) y el párrafo **Joint verify.** (D12).
  Cubre: REQ-28.1, REQ-28.42, REQ-43.32
- [ ] 4.5 (Unidad 9) Crear `internal/sddstatus/joint_verify_nonblocking_test.go`: con y sin ronda «Verificación conjunta» en `verify-report.md`, `nextRecommended` y dependencias idénticos, sin cambios en `internal/sddstatus/governance.go`.
  Cubre: REQ-28.1, REQ-43.32

## Phase 5: Documentación, pines y cierre

- [ ] 5.1 (Unidad 10, RED) Actualizar los pines de `internal/workspace/docs_contract_test.go` (manual `:463-497` y visión `:520`, «`ensure`, `status`, `scope`, `done` y `guard`») y comprobar que fallan.
  Cubre: REQ-25.1
- [ ] 5.2 (Unidad 10, GREEN) Documentar `axiom branch scope` en «Ramas y Worktrees» y en la lista de verbos de `docs/manual-de-inicio.md` y `docs/architecture/AXIOM_ENTERPRISE_VISION.md`.
  Cubre: REQ-25.1
- [ ] 5.3 (Unidad 11) Regenerar los orquestadores con `go test ./... -update` (en segundo plano) sobre `testdata/golden/**` y revisar el diff; actualizar el SHA-256 de los ajustes de Kilocode en `internal/components/sdd/review_ledger_contract_test.go` (`TestKilocodeReviewSettingsMatchCurrentMainBaseline`) con su comentario de cambio deliberado.
  Cubre: REQ-28.42, REQ-43.32
- [ ] 5.4 (Unidad 11) Comprobación final: `go vet ./...` y `go test ./...` en el worktree, sin compilar binarios sueltos; registrar los resultados observados antes de pasar a `sdd-verify`.
  Cubre: REQ-25.1, REQ-25.2, REQ-25.51, REQ-25.52, REQ-25.53, REQ-25.54, REQ-28.1, REQ-28.4, REQ-28.41, REQ-28.42, REQ-43.32

## Matriz de amenazas: pruebas RED previstas

- Selección del repositorio git: 2.1 (`--cwd` relativa resuelta) y 4.1 (pin `git -C`).
- Estado de commit: 2.3 (cambios con y sin stage y sin seguimiento fuera de `files`; referencias, índice y worktrees idénticos).
- Estado de push: 2.3 (observador sin operaciones remotas), 2.5 (`branch_missing` sin `fetch`) y 3.1 (`--fetch` rechazada con `2`).
- Rutas con aspecto de documentación y órdenes de PR: N/A según el diseño.
