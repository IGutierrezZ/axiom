# Informe de verificación: verify-conjunto (C-02-b), rol `fullstack`

Fecha: 2026-11-24 · Rama `inc/verify-conjunto` · Diff `origin/main...HEAD` (12 commits, 32553049..52c3d0a3) · TDD: pruebas RED previstas en `tasks.md`.

## Alcance y progreso observado

Artefactos inspeccionados: proposal, tres deltas de spec, design (D1-D13) y tasks. Tareas 1.1 a 5.4: las 22 figuran `[x]` (estado leído, no reescrito). Lo declarado «Fuera de alcance» en las specs no se evalúa como incumplimiento.

## Comprobaciones ejecutadas (todas en el worktree, sin compilar binarios)

| Comando | Resultado |
|---|---|
| `go vet ./...` | salida 0, sin avisos |
| `go test ./internal/branch/... -count=1` | ok: branch 0.9s, engine 387.9s, gitcli 35.6s, registry 5.8s |
| `go test ./internal/cli/ -run Branch -count=1` | ok (125.5s) |
| `go test ./cmd/axiom/ ./internal/assets/ ./internal/workspace/ ./internal/components/sdd/ -count=1` | ok los cuatro |
| `go test ./internal/sddstatus/ -count=1` | ok (58.9s) |
| `go test ./internal/components/ -run TestGolden -count=1` | ok (3.7s) |

Fallos preexistentes del entorno (filemerge, app, backup, mcp, reviewtransaction, suite completa de cli): no ejecutados, no cuentan.

## Matriz de requisitos

| Requisito | Estado | Evidencia (código / prueba) |
|---|---|---|
| REQ-25.1 cinco verbos, sin `--fetch` | Cumple | `internal/cli/branch.go` (`case "scope"`, ayuda y mensaje de cinco verbos); `internal/cli/branch_scope.go:53-85` (FlagSet sin `--fetch`/`--lane`, salida 2); `cmd/axiom/main.go` (línea `branch scope`); pruebas `internal/cli/branch_scope_test.go`, `branch_test.go`, `cmd/axiom/branch_help_test.go`; `internal/branch/gitcli/ops_test.go` (lista cerrada de lecturas con `log` y `reflog show`) |
| REQ-25.2 `--json`, `--cwd`, rutas absolutas, salida 0 con `unverifiable` | Cumple | `internal/cli/branch_scope.go:79-98`; `internal/branch/engine/scope.go:39-41` (cwd absoluta); `branch.ScopeResult.ExitCode`; `TestScopeResolvesARelativeCwdToAbsolutePaths`; `TestScopeMultirepoCombinesMergedMissingAndExemptRepositories` |
| REQ-25.51 solo lectura, contrato versionado, rechazos | Cumple | `engine/scope.go:34-77` (`kickoff_not_sealed`, `scope_not_applicable` ODD y sin instantánea, `config_invalid` vía `gather`, `git_failed` por `gitRefusal` sin degradar); specs in situ `exempt` (`scope.go:111-114`); carpeta del cambio excluida (`changeFolderPrefix`, `ownFiles` `scope.go:336-366`); `gitcli/scope.go` solo `rev-list`, `log`, `reflog show`; pruebas `TestScopeRefuses*`, `TestScopeIsNotApplicable*`, `TestScopeReleaseBeforeTheMerge...ChangesNothing` |
| REQ-25.52 punto de partida estable y commits ajenos | Cumple | `firstParentFork` (`scope.go:305-318`); tabla D5 (`scope.go:220-267`); `foreign_commits_possible` (`scope.go:256-258`); `fork_point_undetermined` (`scope.go:183-186`); pruebas `TestScopeRelease{Before,After}...`, `...ExcludesTheForeignCommits...`, `...UnrelatedHistories...`, `TestScopeBranch*` |
| REQ-25.53 resultado tipado y motivos | Cumple | dominio cerrado `ok/empty/unverifiable/exempt` y tres motivos (`scope.go:121-155`, `scope.go:166-298`); `branch_missing` con frase fijada (`scope.go:19,127`) y `fetch` solo como continuación citada; origen `origin/<rama>` declarado (`scope.go:390-404`); P1 `scopeAbsentRepository` (`scope.go:88-98`) para que ningún repositorio se omita; pruebas `TestScopeDeclaresABranchThatIsGone...`, `...UsesTheDownloadedRemoteTracking...`, `...MissingBaseOrIntegration...`, `...RepositoryThatIsNotOnThisMachine...`, `TestScopeFastForward*` |
| REQ-25.54 fusión y `offer` | Cumple | `mergeLayers` solo ancestry y equivalent_tree (`scope.go:285-296`); `SummarizeScope` con precedencia correcta (`internal/branch/scope.go:193-222`); pruebas `internal/branch/scope_test.go`, `TestScopeMultirepoWithAnUnmergedRepositoryIsPendingMerge` |
| REQ-28.1 cuatro puntos | Cumple | `internal/assets/skills/_shared/sdd-orchestrator-sections.md` («Four points, and only these.», punto (d) solo oferta, sin pregunta nueva); `internal/assets/independent_review_boundary_test.go`; goldens regenerados y `TestGolden` ok |
| REQ-28.4 alcance del verify sin rol | Cumple | `sdd-verify/SKILL.md` (regla y fila de Decision Gates); `sdd-verify/references/joint-verification.md` §2-3 (usa `axiom branch scope`, fallo declarado sin cambiar veredicto, suite completa no acotada); pines en `internal/assets/assets_test.go` |
| REQ-28.41 verify conjunto = ronda fechada | Cumple | `joint-verification.md` §3-5 (solo commits, `git -C`, excluidos y avisos reproducidos, sin `fetch`, ronda de solo añadir, sin `verdict:` ni `**Estado:**`, sin ronda si todo `empty/exempt`); `report-format.md`; pines `assertNoHeadRangeScope` |
| REQ-28.42 nunca bloquea | Cumple | párrafo **Joint verify.** (5) en `sdd-orchestrator-sections.md`; `continuous` sin oferta; `internal/sddstatus/joint_verify_nonblocking_test.go` (`nextRecommended`, dependencias y razones de bloqueo idénticos con y sin ronda); `sddstatus/governance.go` sin cambios |
| REQ-43.32 tres estados en `integration` | Cumple | **Joint verify.** (1)-(3) en `sdd-orchestrator-sections.md` (hecha con fecha, ofrecida, no verificable con motivos y «consulta no disponible»); frase en el párrafo de stops; pines en `independent_review_boundary_test.go` y frases de `integration` intactas |

## Hallazgos

CRITICAL: ninguno.

WARNING: ninguno.

SUGGESTION:
1. Los escenarios de comportamiento del orquestador (REQ-28.1, 28.42, 43.32) y del verify conjunto (REQ-28.41) se cubren con pines de texto en skills y doctrina, no con ejecución real de un agente; es la técnica prevista en `tasks.md`. Verificado estáticamente, no como prueba de runtime.
2. `joint-verification.md` §6 busca las rondas en `<planningHome>/changes/<change>/verify-report.md`; conviene confirmar en un caso real que `planningHome` equivale a la carpeta de `openspec/` en `monorepo-embedded`.
3. La mezcla de sustantivo `tip`/«punta» entre la ronda (`tip`) y el texto en castellano es coherente con el contrato, solo se anota por legibilidad.

## Limitaciones

No se ejecutó la suite completa de `internal/cli` ni `go test ./...` (cuelgue y fallos preexistentes conocidos). No se ejercitó el verbo contra un repositorio real mediante un binario (prohibido compilar).

## Resumen

Implementación completa y conforme a los tres deltas; todas las comprobaciones solicitadas pasan. Siguiente paso recomendado: `sdd-archive`.

**Estado:** limpio
