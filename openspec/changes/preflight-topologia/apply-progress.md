# Progreso de apply: preflight-topologia (C-01-b)

Modo: estándar (TDD estricto no activo; RED observado igualmente en cada tarea). Estrategia de entrega: `exception-ok` (una PR con `size:exception`). Lote 1: U1 y U2.

## Unidades completadas

| Unidad | Estado | Commit |
|---|---|---|
| U0 | hecha | `1ca2b85c` |
| U1 | hecha | `37da0abf` |
| U2 | hecha | `9383953d` |
| U3 | hecha | ver `git log` (`feat(hub): el alta en monorepo-embedded escribe solo el rol fullstack`) |

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

## U3 · Alta del proyecto (hub y dashboard)

Hecha (lote 2; commit `feat(hub): el alta en monorepo-embedded escribe solo el rol fullstack`, el hash está en `git log`).

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/hub/... ./internal/dashboard/...`: ok; RED del dashboard observado (`TestProjectFormValidationRuntime`: monorepo devolvía los roles del formulario) |
| Arnés de ejecución | `Init`/`ValidateInit` reales sobre carpetas temporales con `NewDetector()`; formulario ejecutado con Node (`runNodeModules`) |
| Cierre | `gofmt -l` vacío; `go vet ./internal/hub/... ./internal/dashboard/...` limpio; `go test ./internal/workspace/...` ok |
| Límite de reversión | `internal/hub/{init,init_validate}.go`, `internal/dashboard/assets/{index.html,js/projects.js,js/core/projectform.js}` y sus pruebas |

Notas:
- `planConfig` y `buildAxiomYaml` aplican `tech.ConfiguredRoles` solo si la topología es `multirepo`; el perfil `knowledge`/`spec-only` no cambia; los roles explícitos se conservan y los rechaza `ValidateInit` (D9).
- Dashboard: se oculta la sección completa de roles (`#field-project-roles`, que envuelve a `#roles-builder-container`, el botón y su ayuda) en monorepo, no solo el contenedor; `validateProjectForm` devuelve `roles: []` únicamente con `topology === 'monorepo-embedded'` (sin topología declarada se comporta como antes).
- Pruebas heredadas adaptadas sin debilitarlas: las que ejercen roles especializados (claves seguras, violaciones juntas, roles personalizados, adopción con roles detectados, `tech` por rol y 400 con violaciones del dashboard) pasan a `Topology: "multirepo"`.
- Las cuatro pruebas nuevas del hub se escribieron tras la implementación (el RED de la conducta ya se observó en las pruebas heredadas que fallaron tras U1); el RED del formulario sí se observó antes del GREEN.
- `buildAxiomYaml` no tiene llamadores (código muerto previo); se aplicó igualmente la regla por el diseño.

## Riesgos abiertos
- Ninguno nuevo de U3.
