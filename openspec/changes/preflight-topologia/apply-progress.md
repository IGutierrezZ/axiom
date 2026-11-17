# Progreso de apply: preflight-topologia (C-01-b)

Modo: estándar (TDD estricto no activo; RED observado igualmente en cada tarea). Estrategia de entrega: `exception-ok` (una PR con `size:exception`). Lote 1: U1 y U2.

## Unidades completadas

| Unidad | Estado | Commit |
|---|---|---|
| U0 | hecha | `1ca2b85c` |
| U1 | hecha | ver commit `feat(workspace): topología gobernante y reglas de roles y almacén en la validación` |

## U1 · Work Unit Evidence

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/workspace/ -run 'GoverningTopology\|ValidateMonorepoRejects\|ValidateMultirepoKeeps\|LocateLoadsMonorepoRoles\|LegacyArtifactStore\|DeclarationPaths'`: ok |
| Arnés de ejecución | `go test ./internal/workspace/... ./internal/sddstatus/...`: ok (el status no cambia) |
| Límite de reversión | `internal/workspace/{locate,validator}.go` y sus pruebas; U2 a U4 dependen de `GoverningTopology` |

Notas de diseño aplicadas: con contexto técnico mantenido, `storeErrors` emite el error incondicional de REQ-2.6 (que nombra también `technical_context.maintained`) y además el error existente de REQ-24.62; sin contexto mantenido, solo el primero. `ValidateDeclarationPaths` ya no sale pronto cuando no hay `technical_context`.

## U2 · Sellado del kickoff por topología

Hecha (commit `feat(kickoff): el sellado aplica la topología al estilo de ejecución y a los roles`; U1 es `37da0abf`).

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/kickoff/...`: ok; `go test ./internal/cli/ -run 'TestKickoffSeal\|TestWriter\|TestGate\|TestRunSDDGate\|Seal\|Kickoff\|WriterGuard'`: ok (82 s) |
| Arnés de ejecución | Sellados reales en proceso sobre workspaces temporales (`workspacetest.Embedded`, `Canonical`, git aislado), sin binario suelto |
| Cierre | `gofmt -l ./internal ./cmd` vacío; `go vet` de kickoff, workspace y cli limpio; ratchet de negativas y `deadcode-ratchet.sh` en verde; `go test ./cmd/axiom/ -run HandoffGuard` ok |
| Límite de reversión | `internal/kickoff/args.go`, `internal/cli/sdd_kickoff.go` y sus pruebas (U1 queda intacta) |

Notas:
- Las pruebas nuevas de `ApplyTopology` viven en `internal/kickoff/apply_topology_test.go` (fichero nuevo, no `args_test.go`); `args_test.go` solo adapta las dos pruebas heredadas.
- El ayudante `sealLegacyKickoff` (en `sdd_kickoff_branch_test.go`) lo usan también `sdd_writer_guard_test.go` y, por transitividad, `sdd_gate_handoff_test.go` (`handOffProject`), que sellaban `core`/`qa` en monorepo y no estaban listadas en `tasks.md`.
- `TestKickoffSealOfAMultirepoRequiresTheSpecsIntegrationBranch...` sellaba `continuous` en multirepo; pasa a `checkpointed`.
- `TestKickoffSealDoesNotRewriteTheSnapshotOnASecondSeal` ya no repite el segundo sellado con `core`/`qa` (el rechazo por topología lo impediría); lo repite con `checkpointed` sin roles.
- Nuevas: `TestKickoffSealRejectsSpecializedRolesInAMonorepoBeforeTheBranchGuard` y `TestRunSDDKickoffSealUnknownTopologyAppliesMonorepoRule`.
- `ParseSealArgs` sigue rechazando `--docs-publication` sin `continuous` (con estilo vacío también).

## Riesgos abiertos
- `go test ./internal/hub/... ./internal/dashboard/...` falla desde U1 (D9 rechaza roles distintos de `fullstack` en monorepo): `TestProjectsInitWritesRoleTech`, `TestProjectsInitRejectsInvalidConfigurationWith400AndViolations` y las pruebas de hub con roles detectados. Es el alcance de U3 (tareas 3.1 a 3.5), que adapta esas expectativas.
