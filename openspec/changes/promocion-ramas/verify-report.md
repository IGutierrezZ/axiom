# Informe de verificación: promocion-ramas (C-02-a)

**Veredicto:** implementación completa; 0 CRITICAL, 2 WARNING, 2 SUGGESTION. Las 5 fases (tareas 1.1-5.7) están marcadas y respaldadas por 6 commits (`c29db67b`..`001da130`) sobre `591840be`.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./internal/branch/... ./internal/workspace/ ./internal/kickoff/ ./internal/cli/ ./cmd/axiom/` | exit 0 |
| `go test ./internal/workspace/ ./internal/branch/... ./internal/kickoff/ ./internal/assets/... ./internal/components/sdd/` | ok (todos los paquetes) |
| `go test ./cmd/axiom/ -run "Workspace\|Branch"` | ok |
| `go test ./internal/cli/ -run "Branch\|Kickoff\|Gate\|Sdd"` | ok (245 s) |
| `go test $(go list ./... \| rg -v "internal/cli$")` | fallos solo ambientales (ver abajo) |

Fallos de la pasada amplia:
- `filemerge` TestWriteFileAtomicFollows{,Relative}DirectoryJunction: ambiental conocido de `main`.
- `backup` TestSnapshotterSkipsUnixSockets: falla igual en `C:\repos\axiom` (main): ambiental (acceso denegado a `broker.sock` en Windows).
- `reviewtransaction`: `panic: test timed out` (10 min, `TestAssessAuthorityRepairStops...`); también agota el tiempo en `C:\repos\axiom` (main, 9 min): ambiental/lentitud, paquete no tocado por el cambio.
- `app` TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation: no falló en esta pasada.

## 1. Requisitos de las deltas

Cada REQ-25.46-25.50 y los MODIFIED 25.3, 25.8, 25.10, 25.14-25.17, 25.24, 25.26, 25.27, 25.36, 25.45 tienen implementación y tests (ficheros `*_promotion_test.go` en `internal/branch`, `engine`, `registry`, `workspace`, `kickoff`, `cli`; `snapshot_test.go`). Resolución y precedencia: `internal/branch/promotion.go`; base efectiva: `planner.go` (`applyPromotion`, `Base()`); `ensure`: `engine/prefs.go` + `ensure.go`; `status`: `status.go` (`describePromotionStage`); `done`: `done.go` + `merge_layers.go`.

## 2. Criterio de terminado de C-02-a

- Sin `main_branch` los tests existentes pasan: de los `_test.go` previos solo cambiaron firmas y listas de claves: `ensure_decision_test.go` (3 llamadas a `DecisionRequired(EnsureArgs{...})`), `done_result_test.go` (2 llamadas a `MergeNotConfirmed` con `destination`), `gitpolicy_test.go:146,176,539` (claves en `wantErrorContains`). `scripts_axiom_worktree_test.go` solo añade `IntegrationBranch` al stub y tests nuevos; `docs_contract_test.go` solo añade casos.
- `branch` con master/develop: `ensure_promotion_test.go`, `done_promotion_test.go`, `status_promotion_test.go`. `release`, `ask` (decision_required, respuesta registrada, conflicto): `ensure_promotion_test.go:180`. Multirepo: `:345` (dos repos secuenciales); specs exento: `planner_promotion_test.go`, `snapshot_test.go:197`. `done` omite specs: comportamiento previo intacto, no hay test nuevo específico (ver W2).

## 3. Squash y `pr_merged`

`done_promotion_test.go:30` (merge commit y squash por `equivalent_tree` contra `master`), `:85` (`pr_merged` sin ascendencia en `master` conserva la rama; `--force` fuerza), `:157` (`ask` sin registro → destino `main_branch`). `done.go` solo rellena `PRMerged` si `MergeDestination` devuelve `prMergedApplies`.

## 4. Contratos que no cambian

`publication.go`, `handoff.go`, `verdict.go`, `planner_archive.go` sin diff; siguen usando `repo.IntegrationBranch`. `RepoPlan.IntegrationBranch` y `PRTarget` de la rama del cambio no cambian (`planner_promotion_test.go:54`); `pending.go` resuelve la base con `Base()` = integración.

## 5. Scripts

`axiom-worktree.ps1/.sh` leen `integrationBranch` de `status --json` con `main` por defecto; tests `TestScriptsAxiomWorktreeDoneSyncsTheResolvedIntegrationBranch` (sh y ps1, `develop` y sin campo) en verde.

## Hallazgos

### WARNING
- W1: REQ-25.50, escenarios «La publicación y el relevo siguen apuntando a la integración» y «El archivado se rechaza si el cambio no está en la integración aunque esté en la rama principal» (y REQ-25.45 «Con dos etapas en modo `branch` el archivado espera solo a la integración»): no hay test de motor con `main_branch` declarado que ejerza `publish`, `handoff` ni `ensure --archive`. Solo `TestPlanArchiveHasNoPromotion` (`internal/branch/planner_promotion_test.go:135`) cubre el plan de archivado. El código no se tocó y lee la integración, por lo que la garantía es por ausencia de diff, no por test de regresión.
- W2: REQ-25.49, escenario «En multirepo `done` omite el repositorio de specs y evalúa cada repositorio con su modo»: sin test de `done` multirepo con modos distintos por repositorio en `done_promotion_test.go` (los tests de `done` son monorepo).

### SUGGESTION
- S1: añadir tests de `publish`/`handoff`/archivado con `main_branch` (cierra W1) y un `done` multirepo (cierra W2).
- S2: el cambio de `reviewtransaction` no tiene relación; conviene aislar su tiempo de espera ambiental en `main` aparte.

## Limitaciones
No se ejecutaron los scripts contra un remoto real ni binarios sueltos de axiom (restricción Cortex). `go test ./internal/cli/` se ejecutó solo con `-run`.

next_recommended: sdd-archive (los WARNING son huecos de cobertura, no incumplimientos).

Remediación: W1 y W2 cubiertos por TestDoneMultirepoEvaluatesEachRepositoryWithItsOwnPromotionMode (done_promotion_multirepo_test.go), TestPublicationAndHandOffKeepTargetingTheIntegrationInBranchMode y TestEnsureArchiveWaitsOnlyForTheIntegrationInBranchMode (promotion_contracts_test.go) (este commit; sin cambios de código de producción).
