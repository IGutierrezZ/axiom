# Informe de verificación: carril-sdd-versatil

> Fase `verify` · 2026-11-12 · commit verificado `b16ff42e` (merge de la PR #256, ya en `main`) · cambio no ISO (su `kickoff.yaml` no tiene instantánea `sdd:`): informe de formato libre, sin matriz ni líneas `verdict:`.
> Almacén `hybrid`. STRICT TDD desactivado. Verificación sin construir ni ejecutar binarios (solo `go test` y `go vet` en proceso).

## Resumen

Veredicto: **cumple**. 0 CRITICAL, 2 WARNING, 3 SUGGESTION. Las 46/46 tareas figuran hechas y cada requisito REQ-29.1 a REQ-29.19 tiene al menos una prueba existente que pasa.

## Estado de las tareas (observado, sin reescribir)

`tasks.md`: 46/46 casillas marcadas (U1 a U24 y el cierre 24.2); cada tarea lleva `Cubre:`. La tabla de cobertura REQ → tareas incluye los 19 requisitos nuevos y los MODIFIED REQ-21.13, REQ-21.15, REQ-24.38, REQ-28.3, REQ-28.4 y REQ-28.6.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go test` de isoevidence, workspace, kickoff, hub, sddstatus, assets, components/agentguidance y components/sdd (todos con `/...`) | salida 0; todos `ok` (isoevidence 3,6 s, workspace 37 s, kickoff 3,7 s, hub 7,6 s, sddstatus 80 s, assets 46 s, agentguidance 4,1 s, components/sdd 122 s) |
| `go test ./internal/cli/ -timeout 9m -run` con el patrón anclado `^(TestSDD, TestRunSDD, TestGateDigest, TestArchive, TestGateRecord, TestEveryProductionRefusal)` | salida 0; `ok` en 195 s |
| `go test ./cmd/axiom -run` con `^(TestSDDVerbRouting, TestRunSDD, TestHelp)` | salida 0; `ok` en 11 s |
| `go vet ./...` | salida 0, sin diagnósticos |

Limitaciones: la suite completa de `internal/cli` no se ejecutó (supera los 10 minutos); solo el subconjunto acotado. `internal/components/filemerge` falla en Windows (uniones de directorio) también en `main`; no se atribuye al cambio y no se ejecutó aquí. El CI en verde de la PR se toma del encargo, no se reverificó.

## Cumplimiento por requisito (REQ → prueba)

Todas las pruebas se localizaron con `rg` en el árbol y se ejecutaron dentro de los comandos anteriores.

| REQ | Prueba (paquete: símbolo) |
|---|---|
| REQ-29.1 | `workspace`: `TestDecodeSDD`, `TestCheckISOWorkflow`, `TestSDDLoad`, `TestWorkspaceValidateSDD` (`sdd_test.go`); `hub`: `TestDeclaresCountsSDD`; `cli`: `TestRunSDDKickoffSealRejectsAnInvalidOrContradictorySDDDeclaration` |
| REQ-29.2 | `sddstatus`: `TestISOProjection`, `TestDecorateISO` (`iso_projection_test.go`), `TestISOStatusContinuity` (`iso_test.go`), `TestProjectWorkflowISOContradiction` (`iso_workflow_test.go`); `assets`: `TestSDDStatusContractISO` |
| REQ-29.3 | `isoevidence`: `TestRequirementIDs`, `TestCheckRequirementIDs` (`reqid_test.go`); `sddstatus`: `TestISOSpecsStage`, `TestISOInterruptedArchive` (`iso_stages_test.go`); `cli`: `TestSDDArchiveCheckAlreadyComposed`, `TestSDDArchiveCheckAlreadyComposedNegatives` |
| REQ-29.4 | `isoevidence`: `TestParseCubre`, `TestCheckCoverage` (`cubre_test.go`); `sddstatus`: `TestISOTasksStage` |
| REQ-29.5 | `isoevidence`: `TestCheckExistingRequirementBug` (`cubre_test.go`); `sddstatus`: `TestISOTasksStage` |
| REQ-29.6 | `isoevidence`: `TestParseMatrix`, `TestValidateMatrix`, `TestScenarios`; `sddstatus`: `TestISOArchiveRefusal`, `TestISOMatrixProjection` (`iso_matrix_test.go`) |
| REQ-29.7 | `isoevidence`: `TestTreeResolver`, `TestResolveGoTest` (`testref_test.go`), `TestMatrixNeverExecutesCommands` (`matrix_threat_test.go`), `TestValidateMatrix` |
| REQ-29.8 | `isoevidence`: `TestParseMatrix`, `TestValidateMatrix` (filas manuales) |
| REQ-29.9 | `isoevidence`: `TestDeriveTrace`; `cli`: `TestSDDTraceRejectsUnsafeInput`, `TestSDDTraceSameChainWithSquashMergeAndRebase`, `TestSDDTraceDiscrepancyWithEvidence`, `TestSDDTraceBeforeArchiving`, `TestSDDTraceChangeNotIntegratedYet` (`sdd_trace_flow_test.go`); `cmd/axiom`: `TestSDDVerbRouting` |
| REQ-29.10 | `isoevidence`: `TestCheckScenarios`; `sddstatus`: `TestISOSpecsStage`; `cli`: `TestSDDArchiveCheck`, `TestArchiveComposeISORefusal`, `TestArchiveComposeISORefusesEndToEndWithoutWriting`, `TestArchiveComposeISORefusalWithoutChangeContext` |
| REQ-29.11 | `sddstatus`: `TestISOArchiveRefusal`, `TestISOArchiveWithPendingTasks`, `TestISOVerifyRoute`; `cli`: `TestSDDArchiveCheck`, `TestArchiveComposeISORefusal`; `assets`: `TestSDDArchiveRequiresArchiveCheck` |
| REQ-29.12 | `kickoff`: `TestPrepareChangeRoot` (`seal_prepare_test.go`); `cli` (`sdd_kickoff_iso_test.go`): `TestRunSDDKickoffSealOnNonexistentChangeCreatesTheFolder`, `...KeepsAnExistingFolderAndIsIdempotent`, `...RejectsUnsafeChangeNamesWithoutCreatingAnything`, `...WrongBranchLeavesNoFolder`, `...RemovesTheCreatedFolderWhenSealFails`; `branch/engine`: `TestWriterInventoryEveryWriterIsGuardedOrExempt` |
| REQ-29.13 | `kickoff`: `TestSDDSnapshot`, `TestIsISO` (`types_test.go`); `cli`: `TestRunSDDKickoffSealCopiesSDDISOToTheSnapshot`, `TestRunSDDKickoffSealInferNeverCreatesTheFolderNorCopiesISO` |
| REQ-29.14 | `isoevidence`: `TestCheckExistingRequirementBug`, `TestAffected*` (`cubre_test.go`) |
| REQ-29.15 | `isoevidence`: `TestParseBugDeclaration`; `sddstatus`: `TestISOBugProfile`, `TestGovernanceSpecGateBugWithoutDelta`; `cli`: `TestGateDigestBugWithoutDelta` |
| REQ-29.16 | `isoevidence`: `TestFindingReason`, `TestAllCodesClosedList`; `sddstatus`: `TestISOReasonsOrdering`, `TestISOVerifyRoute`, `TestISOBugProfile`; `assets`: `TestSDDStatusContractISO` |
| REQ-29.17 | `assets`: `TestSDDISOConvention`, `TestSDDSkillsReferenceISOConvention`, `TestReportFormatHistoricalMatrix`, `TestSDDISOPins`, `TestSDDArchiveRequiresArchiveCheck`, `TestSDDStatusContractISO`; `workspace`: `TestExampleConfigLoads`; `components/sdd`: pruebas de composición y plantillas de orquestador |
| REQ-29.18 | `agentguidance`: `TestRoutingISOSendsAllWorkToSDD`, `TestRoutingISOBugRoutes`, `TestRoutingISONamesTheRouteInEveryEntryPoint`, `TestRoutingISOStaysNeutral`, `TestUnknownRoutingPolicyFailsClosed` (`routing_stops_test.go`) |
| REQ-29.19 | `isoevidence`: `TestCheckNoImpactBug`, `TestValidateMatrix`; `sddstatus`: `TestISOTasksStage`, `TestISOMatrixProjection` |
| REQ-21.13 (MOD) | `cli`: `TestGateRecordLastRoleNotice*` (5 pruebas, `sdd_gate_notice_test.go`) |
| REQ-21.15 (MOD) | `kickoff`: `TestLastBlockingRoleClosed`, `TestPendingNonBlockingRoles`, `TestIntermediateMergeCountsOnlyBlockingRoles`, `TestHandedOffToIgnoresNonBlockingRoles` (`blocking_roles_test.go`); `sddstatus`: `TestRoleGatesDependency*`, `TestVerifyDependencyFromRoleGatesByPolicy` (`role_gates_policy_test.go`) |
| REQ-24.38 (MOD) | `cli`: `TestArchiveComposeISOAcceptedOutputIsByteIdentical`, `TestArchiveComposeNonISOKeepsComposingWithoutScenarios`, `TestArchiveComposeInPlaceIsNotTouchedByTheISORefusal` |
| REQ-28.3, 28.4, 28.6 (MOD) | `assets` y `components/sdd`: pines de doctrina (`TestSDDISOPins`) y composición de orquestador (`TestCanonicalComposition*`) |

Las pruebas son de comportamiento (tablas, repos git reales en `t.TempDir()`, compositor real de archivado), salvo REQ-29.17, REQ-29.18 y los MODIFIED REQ-28.x, que son pines de prosa por naturaleza.

## Coherencia con el design (muestreo D1 a D21)

| Decisión | Pieza observada | Resultado |
|---|---|---|
| D1 paquete hoja | `internal/isoevidence` con `TestPackageImports` (sin `os/exec` ni `internal/...`) | Conforme |
| D3 y D18 bloque `sdd:` y contradicción con `workflow.lane` | `workspace/sdd.go`, `ErrISOWorkflowContradiction`; `projectWorkflow` ask/invalid | Conforme |
| D4 instantánea `sdd.iso` | `kickoff.SDDSnapshot`, `IsISO()`, `TestSDDSnapshot` | Conforme |
| D5 `PrepareChangeRoot` y retirada | `kickoff/seal.go`; entrada en `writerCalls` de `writer_inventory_test.go:63` | Conforme |
| D6, D12 y D19 estrechamiento y ruta `verify` | `sddstatus/iso.go`; `TestISOVerifyRoute`, `TestISOArchiveWithPendingTasks` | Conforme |
| D9 y D20 perfil de bug y compuerta `spec` | `TestISOBugProfile`, `TestGovernanceSpecGateBugWithoutDelta`, `TestGateDigestBugWithoutDelta` | Conforme |
| D13 `archive-check` y negativa en `archive-compose` | `cli/sdd_archive_check.go`, `sdd_archive_compose_iso.go`; paso obligatorio en `sdd-archive/SKILL.md:78` | Conforme |
| D14 `sdd trace` | `cli/sdd_trace.go`; verbo enrutado en `cmd/axiom/main.go:2228` | Conforme |
| D15 QA diferido | `LastBlockingRoleClosed`, `PendingNonBlockingRoles`, `HandedOffTo`, `IntermediateMerge` | Conforme |
| D16 doctrina | pines en `agentguidance` y `assets`; `_shared/sdd-iso-convention.md` | Conforme |
| D17 ADDED ya compuesto | `TestCheckRequirementIDs`, `TestISOInterruptedArchive`, `TestSDDArchiveCheckAlreadyComposed*` | Conforme |
| D21 almacenes sin ficheros | `blocked(iso_store_unsupported)` cubierto en `iso_projection_test.go:233-249` | Conforme |

Desviaciones conocidas respecto al texto del design y de `tasks.md`:

- Hay **13 tokens**, no 14 (`isoevidence.AllCodes()`; la prueba de `assets` fija que el design lista 13). El design §5.3 y la tarea 23.1 hablan de 14.
- `archive-check` **no lleva `--supersede`**: el design D17 menciona un REMOVED marcado como superado con `--supersede`, pero el verbo no expone el indicador; `TestSDDArchiveCheckAlreadyComposedNegatives` cubre el comportamiento resultante.
- `kickoff.LastRoleClosed` se retiró tras el apply (queda solo el campo JSON `lastRoleClosed` de `gate record`); `isoevidence.AllCodes` pasó a `.deadcode-baseline.txt` porque solo lo consumen pruebas.
- La tarea 21.2 cita un patrón `-run` con `TestOrchestratorNoDrift`, nombre que ninguna prueba lleva (`orchestrator_no_drift_test.go` define `TestOrchestratorTemplates...` y `TestSharedOrchestratorSections...`); la cobertura real se obtuvo al ejecutar el paquete `components/sdd` completo (verde).

## Cambios de comportamiento visibles fuera de ISO

1. `HandedOffTo` con un rol `deferred` u `optional`: la salida 2 solo cede `verify` cuando se cierra el último rol `blocking`; `IntermediateMerge` ignora los no bloqueantes. Con todos los roles `blocking` o con `fullstack` el resultado es el anterior.
2. `lastRoleClosed` en `gate record`: ahora significa último rol bloqueante cerrado y se añade `pendingNonBlockingRoles` (`omitempty`); el aviso de REQ-21.13 ya no salta al cerrar un rol no bloqueante.

Ambos están respaldados por pruebas (REQ-21.13 y REQ-21.15).

## Hallazgos

CRITICAL: ninguno.

WARNING
1. `internal/components/filemerge` falla en Windows (uniones de directorio), también en `main`. No atribuible al cambio; sigue pendiente de seguimiento aparte.
2. `.codex/AGENTS.md` está desfasada: copia generada versionada, anterior al cambio, que ninguna prueba compara y que solo regenera `axiom sync` desde un binario de la rama. Fuera de alcance por decisión del usuario (tarea 24.2); queda como seguimiento.

SUGGESTION
1. El design y la tarea 23.1 nombran 14 tokens; el código y la prueba, 13. Dejar constancia de la corrección al archivar para no confundir a un auditor.
2. El patrón `-run` con `TestOrchestratorNoDrift` de la tarea 21.2 no coincide con ninguna prueba; corregirlo en la documentación del plan.
3. En el worktree de archivado, `gates.yaml` figura modificado y `.gates.lock` sin seguimiento (no los tocó esta fase); revisarlos antes del commit de archivado.

## Aviso de contexto

No aplica: el cambio no declara contexto técnico mantenido.

## Recomendación

`sdd-archive`. Implementación completa (46/46), pruebas acotadas y `go vet` en verde, sin hallazgos CRITICAL.
