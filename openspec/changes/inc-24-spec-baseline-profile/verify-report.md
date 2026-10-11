# Informe de verificación consolidado: inc-24-spec-baseline-profile

> Fase `verify` global (REQ-21.13, REQ-21.15). Informe consolidado, distinto de los `verify-report.<rol>.md`. Diagnóstico práctico opcional: no gobierna el archivado y no abre bucles de revisión, refutación o corrección.
> Rama `feat/inc-24-spec-baseline-profile`, cabeza `6e8d824b`, worktree `C:\repos\axiom-wt\inc-24-spec-baseline-profile`. Base de la rama: `main` 902b4487. Fecha: 2026-10-11. Rol único `fullstack`. `strict_tdd: false` (`openspec/config.yaml`): la comprobación de evidencia RED/GREEN no aplica.

## 1. Veredicto

**0 CRITICAL, 4 WARNING, 4 SUGGESTION.** Los 79 requisitos tienen implementación y prueba: 77 cumplidos y 2 parciales solo en prueba (REQ-24.12 y REQ-21.16, escenario `pr_merged` sobre el espacio canónico, implementado por construcción). Ninguno queda sin cubrir ni fuera de alcance. Todas las comprobaciones ejecutadas en esta fase están en verde. El estado nativo sigue bloqueando el archivado solo por la compuerta `integration` (evidencia `pr_merged`), que se resuelve al fusionar el PR #133.

## 2. Estado de las tareas (observado, sin modificar)

- `tasks.md`: 186 casillas `[x]` y 0 `[ ]`. `axiom sdd status` coincide: `taskProgress` 186/186, `allComplete: true`, `applyState: all_done`.
- Fases 0 a 8 completas. No hay tareas incompletas.
- Compuertas aprobadas: `spec`, `design`, `tasks` y `role-apply:fullstack`. La compuerta `integration` no está registrada (bloquea `archive`).
- Artefactos del status: `proposal`, `specs` (11 capacidades), `design` y `tasks` en `done`; `applyProgress` y `verifyReport` figuran `missing` porque el progreso de apply vive en Engram (`sdd/inc-24-spec-baseline-profile/apply-progress`, observación 562) y este informe es el que crea `verifyReport`.

## 3. Ejecución real (en serie, cwd del worktree)

Resultados literales de esta fase:

| Comando | Resultado |
|---|---|
| `go build ./...` | exit 0 (7 s) |
| `go vet ./...` | exit 0 (5 s) |
| `go test ./internal/specbaseline -count=1 -timeout 30m` | `ok` (148 s) |
| `go test ./internal/workspace ./internal/sddstatus ./internal/kickoff ./internal/hub ./internal/livingdoc ./internal/knowledge ./internal/assets ./internal/components/agentguidance ./internal/components/sdd ./cmd/axiom -count=1 -p 2` | los 10 paquetes `ok` (141 s en total: sddstatus 92,6 s, components/sdd 101,6 s, cmd/axiom 27,1 s, assets 21,6 s) |
| `go test ./internal/cli -run 'ArchiveCompose\|Integrate\|InPlace\|Verify\|Show\|Trace\|SDDStatus\|Kickoff\|Gate\|Attempt\|Handoff\|Role\|TestEveryProductionRefusal\|TestRefusalRatchet\|TestEveryNamedReviewContinuation' -count=1` | `ok` (587,9 s; 591 s de reloj) |
| `git status --porcelain` antes y después de la batería | idéntico: solo `?? openspec/changes/inc-24-spec-baseline-profile/.gates.lock` |

No hubo ningún fallo, de modo que no se activó ninguna de las excepciones ambientales conocidas (los paquetes afectados por ellas no estaban en el alcance pedido).

**Evidencia previa citada, no repetida** (tarea 8.1 y `pr-description.md`, cabeza `9f4bd977`, y verificación posterior al rebase sobre `main` 60e75cb2):

- `go run ./internal/gofmtcheck` (exit 0), `bash scripts/deadcode-ratchet.sh` (salida «no new unreachable functions») y `go test ./... -count=1 -p 4` sin `internal/cli`, `internal/reviewtransaction` ni `internal/specbaseline`: 75 paquetes en verde salvo los fallos ambientales de la base (backup, filemerge junctions, opencode CancelsOverflowingChild, pathidentity, releasepolicy, semantic ReindexCodeGraph, tui TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults).
- `e2e/organicruntime` en verde (551 s) e `internal/cli` con 791 pruebas fuera de los `review*_test.go` en dos tandas (solo fallan las 3 ambientales de `sync_test.go`).
- Límites declarados: las 860 pruebas `review*_test.go` de `internal/cli` y `internal/reviewtransaction` completo no terminan en esta máquina y la rama no los toca (`git diff main...HEAD -- internal/reviewtransaction` vacío, comprobado en esta fase); las pruebas de enlaces simbólicos se omiten en Windows (las de `specbaseline` se ejecutaron en WSL con `GOOS=linux` durante C2); `bench/` falla 19 pruebas idénticas a las de la base; `go test -race` no disponible; cobertura no medida en porcentaje.

Comprobaciones estáticas de esta fase (solo lectura):

- R1: `git diff main...HEAD` de `internal/sddstatus/openspec_archive_compose.go`, su prueba y `internal/reviewtransaction` está vacío. `openspec/INDEX.md`, `openspec/config.yaml`, el `AGENTS.md` de la raíz y `.kiro/` no cambian. Los únicos ficheros fuera de `openspec/`, `testdata/`, `internal/`, `cmd/` y `docs/` son `.codex/AGENTS.md` y `.github/workflows/ci.yml`, ambos previstos.
- Los dos ficheros con parser de `### Requirement:` que aparecen en el diff (`internal/livingdoc/indexer.go`, `internal/knowledge/query.go`) cambian solo el nombre de proyecto del índice (m11), la resolución de raíz y las notas de guarda; los cuatro parsers no se tocan.
- Total del diff `main...HEAD`: 187 ficheros, +35.238 / -319 líneas (incluye artefactos de planificación, goldens y `.codex/AGENTS.md`).

## 4. Matriz de cumplimiento (79 requisitos)

Convención: ruta relativa al repositorio; `Test*` son pruebas Go; «pin» = prueba de texto sobre un asset; «previa» = prueba anterior a inc-24. Estado: Cumplido, Parcial, No cubierto, Fuera de alcance justificado.

### 4.1 planning-home-resolution (12)

| REQ | Implementación | Pruebas | Estado |
|---|---|---|---|
| 24.1 | `internal/workspace/locate.go`: `Locate`, `LocateAt`, `LocateError` | `TestLocateDoesNotCreateAPlanningHome`, `TestLocateRejectsUnresolvableConfiguration`, `TestLocateCanonicalResolvesTheSamePlanningHomeFromEveryStart` | Cumplido |
| 24.2 | `locate.go`: recorrido ascendente, precedencia, `MarkerDir` | `TestLocateNearestMarkerWins`, `TestLocatePrecedenceBetweenMarkersOfTheSameDirectory`, `TestLocateSubdirectoryFallbackNeverActsAsMarker`, `TestLocateWithoutMarkersRootsAtTheStart`, `TestLocateStartInvalid` | Cumplido |
| 24.3 | `locate.go`: `Roots` | `TestRoots`, `TestLocateRootsByTopology` (incluye `docs/openspec`) | Cumplido |
| 24.4 | `locate.go` (embebido); costura en `internal/sddstatus/status.go` | `TestLocateEmbeddedLayoutIsInvariantUnderSpecsRepository`, `TestResolveEmbeddedLayoutIsEquivalentForEverySpecsRepository`, `TestSDDStatusV2CleanBreak` (previa, sin cambios) | Cumplido |
| 24.5 | `locate.go`: deducción de carpeta maestra en cuatro pasos y excepción embebida | `TestLocateDeducesTheMasterFolderFromTheSpecsRepository` | Cumplido |
| 24.6 | `locate.go`: `ambiguous_planning_home`; `internal/hub/init.go` | `TestLocateAmbiguousPlanningHome`, `TestResolveRejectsAnAmbiguousPlanningHomeFromEveryStartingPoint`, `TestInitRejectsAnOwnPointerToAnotherPlanningHome` | Cumplido |
| 24.7 | `locate.go` (cuatro casos); `loader.go`: `parseLayered` | `TestLocateRejectsUnresolvableConfiguration`, `TestLocateKeepsTheEmbeddedResolutionForAnUnloadableDirectConfig`, `TestResolveReportsLocationNotesWithoutBlocking` | Cumplido |
| 24.8 | `status.go` `Resolve`; `internal/cli/sdd_governance.go`, `sdd_kickoff.go`, `sdd_attempt.go`; `cmd/axiom/main.go` (handoff, role) | `TestGovernanceVerbsOperateOnTheSamePlanningHomeFromEveryStart`, `TestKickoffSealInferReadsTheConfigurationBehindThePointer`, `TestRoleListHonoursThePointerFromEveryStart`, `TestHandoffVerbsUseThePlanningHomeFromEveryStart`, `TestResolveCanonicalWorkspaceFromThreeStartingPoints`, `TestRunSDDArchiveComposeCanonicalWithCwdResolvesRelativePaths` | Cumplido |
| 24.9 | decorador de `status.go`; `internal/kickoff/archived.go`: `ArchivedChangeRoot`; `sdd_attempt.go` | `TestResolveCanonicalWorkspaceFromThreeStartingPoints` (`allowedEditRoots`, `changeRoot`, `mode`), `TestArchivedChangeRoot`, `TestGovernanceVerbsRejectAnArchivedChangeWithADatePrefix`, `TestSDDAttemptGrantOperatesOnTheSpecsRepositoryFromACodeRepository`; el escenario del repositorio hermano se apoya en `edit_authority_test.go` (previa) | Cumplido |
| 24.10 | `internal/livingdoc/service.go`, `internal/knowledge/specs_root.go`, `internal/dashboard/service.go` (`planningRoot`), `internal/tui/model.go`, `Location.LegacySpecsDivergence` | `TestServiceReadsThePlanningHomeRegardlessOfSpecsRepository`, `TestServiceWarnsAboutADivergentLegacyLocation`, `TestDefaultSpecsRootTakesThePlanningHome`, `TestSweepAndQueryEmitTheLegacyLocationNote`, `TestPlanningHomeResolvesFromTheRegisteredPath`, `TestGovernanceLoadersResolveThePlanningHome`, `TestArchiveListReadsThePlanningHomeCatalog` | Cumplido |
| 24.11 | `internal/assets/skills/_shared/sdd-phase-common.md` | pin en `internal/assets/assets_test.go` (bloque REQ-24.11) | Cumplido |
| 24.12 | `docs/manual-de-inicio.md`; `internal/cli/sdd_gate.go:63-107` llama a `verifyGateEvidence(loc.WorkspaceRoot)` | `TestManualDeInicioDocumentsTheCanonicalWorkspaceRecipe`, `TestManualDeInicioDocumentsPRMergedEvidence`; `pr_merged` de un solo repositorio en `sdd_gate_test.go` (previa) | **Parcial (prueba)**: el escenario «commit solo en `repo-backend` se rechaza» no tiene prueba con fixture canónico (W1) |

### 4.2 spec-baseline-declaration (15)

| REQ | Implementación | Pruebas | Estado |
|---|---|---|---|
| 24.13 | `internal/workspace/declarations.go`, `loader.go` | `TestDeclarationsAbsentAreTheZeroValue`, `TestDeclarationsValidBlocksAreLoaded`, `TestDeclarationsBaselineValidation` (baseline vacía) | Cumplido |
| 24.14 | `declarations.go`; `validator.go`: `ValidateDeclarationPaths` | `TestDeclarationsBaselineValidation` (dominio duplicado, documento en dos dominios, rutas), `TestValidateDeclarationPathsBaselineDocuments`, `TestWorkspaceValidateReportsOneErrorPerInvalidPath` | Cumplido |
| 24.15 | `declarations.go` (perfil); `internal/specbaseline/profile.go`: `CompileProfile` | `TestDeclarationsRequirementProfileCompile`, `TestDeclarationsBaselineValidation` (grupos, patrón, nivel, render), `TestProfile*` | Cumplido |
| 24.16 | `declarations.go`: `EffectiveIntegration`, `validateDocumentProfile` | `TestDeclarationsEffectiveValuesOfAnEmptyProfile`, `TestDeclarationsBaselineValidation` (cuatro casos de `integration`) | Cumplido (sin etiqueta `REQ-` en el código; comprobado por lectura) |
| 24.17 | `loader.go`: `parseLayered`; `cmd/axiom/main.go`: `runWorkspaceValidate` | `TestDeclarationsErrorsAccumulate`, `TestDeclarationsUnknownKeysOutsideTheNewBlocksAreIgnored`, `TestWorkspaceValidateAcceptsAValidDeclaration`, `TestWorkspaceValidateReportsAMissingDocument` | Cumplido |
| 24.18 | canales `BaselineErr` y `ParseErr` en `loader.go`; `internal/sddstatus/workspace_declarations.go`: `blockArchiveOnDeclarations` | `TestLayeredErrorChannels`, `TestArchiveBlockedByAnInvalidDeclaration`, `TestArchiveNotBlockedWithoutDeclarations`, `TestSyncAndColdStartRejectAProjectBaseline`, `TestAnIndeterminateBaselineRejectsWritesAndWarnsOnReads` | Cumplido |
| 24.19 | `internal/sddstatus/status_v2.go`; `workspace_declarations.go`: `projectBaseline` | `TestDeclarationsStatusProjectsBothBlocksWithAbsolutePaths`, `TestDeclarationsStatusProjectsWithoutASelectedChange`, pin del contrato en `assets_test.go` | Cumplido |
| 24.20 | campos opcionales `omitempty` en `status.go` y `status_v2.go` | `TestDeclarationsStatusWithoutDeclarationHasNoNewKeys`, `TestDeclarationsStatusDoesNotAlterPlanningHomeOrArtifacts`, `TestSDDStatusV2CleanBreak` | Cumplido |
| 24.21 | el decorador solo lee `axiom.yaml`; `docs/openspec-config.md` | `TestDeclarationsStatusIgnoresADeclarationInOpenSpecConfig`, `TestOpenSpecConfigDocumentsTheConfigurationBoundary` | Cumplido |
| 24.22 | `internal/workspace/baseline_guard.go`: `Declarations.Ownership`, `Location.BaselineOwnership` | `TestOwnershipAnswers`, `TestLocationOwnershipAgreesFromEveryStart`, `TestOwnershipHasNoAnswerWhenLocateRejects`, `TestOwnershipIgnoresTheArtifactStore` | Cumplido |
| 24.23 | `Location.GuardSpecsWrite`; `internal/hub/init.go`; `livingdoc/service.go`; `knowledge/sweep.go`, `recorder.go`, `finalize.go`, `query.go`; `dashboard/server.go` (409); `tui` | `TestArchiveSyncAndColdstartAreRejectedWithAProjectBaseline`, `TestKnowledgeSweepSkipsTheSeedingWithAProjectBaseline`, `TestKnowledgeCrawlRecordAndFinalizeAreRejectedWithAProjectBaseline`, `TestKnowledgeQueryEnrichIsSkippedWithAProjectBaseline`, `TestArchiveSyncIsRejectedWithAProjectOrIndeterminateBaseline`, `TestLivingDocSyncShowsTheRejectionWithItsContinuation`, `TestInitWithAProjectBaselineSkipsTheSpecs` | Cumplido |
| 24.24 | `sdd-init` y `sdd-archive` (rama de baseline), `_shared/project-baseline-convention.md` | `TestSDDArchiveProjectBaselineBranch` y pin de `sdd-init` en `assets_test.go` | Cumplido (pin de texto) |
| 24.25 | `livingdoc/types.go`: `ExternalBaseline`; `cmd/axiom/main.go` (`archive list` y `archive show`); `knowledge/query.go`; `dashboard/assets/app.js`; `tui/screens/living_doc.go` | `TestGetCatalogWithAProjectBaselineReportsItAsExternal`, `TestArchiveListAndShowReportAnExternalBaseline`, `TestQueryEnrichWithAProjectOrIndeterminateBaseline`, `TestLivingDocScreenShowsAnExternalBaseline`, `TestLivingDocsTabPresentsTheExternalBaseline` | Cumplido (`app.js` por marcadores) |
| 24.60 | `declarations.go`: `EffectiveScenarios` y regla de nivel | `TestDeclarationsBaselineValidation` (ocho casos de `scenarios` y nivel) | Cumplido |
| 24.62 | `internal/workspace/artifact_store.go`: `DeclaredArtifactStore`; `validator.go`; `workspace_declarations.go`: `effectiveArtifactStore` | `TestValidateDeclarationPathsArtifactStore`, `TestWorkspaceValidateRejectsAFileDeclarationWithAFilelessStore`, `TestArchiveBlockedByAFilelessStore` | Cumplido |

### 4.3 project-technical-context (12)

| REQ | Implementación | Pruebas | Estado |
|---|---|---|---|
| 24.26 | `declarations.go`: `TechnicalContext`, modos `maintained` y `reference` | `TestDeclarationsTechnicalContextValidation` | Cumplido |
| 24.27 | `validator.go`: `ValidateDeclarationPaths` (errores y advertencias) | `TestValidateDeclarationPathsContextPaths`, `TestValidateDeclarationPathsSymlinks` (omitida en Windows), `TestWorkspaceValidatePrintsWarningsWithCompliant` | Cumplido |
| 24.28 | `workspace_declarations.go`: `projectTechnicalContext` (sin proyectar el delta) | `TestDeclarationsStatusProjectsBothBlocksWithAbsolutePaths`, pin del contrato | Cumplido |
| 24.29 | skills de fase lectoras y `project-baseline-convention.md` | pin de `entry` y presupuesto de lectura en `assets_test.go` | Cumplido (pin de texto) |
| 24.30 | prohibición en skills; `specbaseline/verify.go` (alcance sin `reference`) | pin de skills, `TestVerifyScopeRejectsEveryPathOutsideTheAllowedSet` | Cumplido |
| 24.31 | `specbaseline/contextdelta.go`: `ParseContextDelta` | `TestParseContextDeltaWithEntries`, `TestParseContextDeltaNoImpact`, `TestParseContextDeltaRejections`, `TestParseContextDeltaReportsEveryProblem` | Cumplido |
| 24.32 | `specbaseline/anchors.go`: `ParseAnchor`, `ResolveAnchor`, `CheckAnchor` | `TestParseAnchor`, `TestResolveAnchor`, `TestResolveAnchorIgnoresFencesAndSetext`, `TestCheckAnchorNamesTheOccurrencesItCannotAddress` | Cumplido |
| 24.33 | skill `sdd-design` | pin en `assets_test.go` | Cumplido (pin de texto) |
| 24.34 | skill `sdd-apply` | pin en `assets_test.go` | Cumplido (pin de texto) |
| 24.35 | skill `sdd-verify` | pin en `assets_test.go` | Cumplido (pin de texto) |
| 24.36 | decorador (la falta de `context-impact.md` bloquea `archive`); skills y compuertas | `TestArchiveBlockedWithoutTheContextDelta`, `TestCheckWithoutContextDeltaWarnsWhilePrepareFails`, pins | Cumplido |
| 24.37 | `specbaseline/plan.go` (autorización fuera del repositorio de especificaciones); `contextdelta.go`: `ContextScope` | `TestPrepareRequiresEditAuthorizationOutsideTheSpecsRepository`, `TestIntegratePrepareTakesTheEditRootsFromTheStatus`, `TestContextScope*` | Cumplido |

### 4.4 spec-baseline-integration (19)

| REQ | Implementación | Pruebas | Estado |
|---|---|---|---|
| 24.38 | `internal/cli/sdd_archive_compose.go` (motor nativo sin perfil); `openspec_archive_compose.go` intacto | `TestRunSDDArchiveComposeDocumentWithoutBaselineUsesTheNativeEngine`, 9 `TestComposeOpenSpecCanonicalSpec*` y 3 `TestRunSDDArchiveCompose*` sin cambios, `TestComposeTextIdentityMatchesTheNativeEngine` | Cumplido |
| 24.39 | `specbaseline/document.go`, `profile.go`, `compose.go` | `TestBlockRequirementsOfTheFunctionalReference`, `TestProfileMatchesTheWholeHeadingLine`, `TestBlockIdentityByTextUsesTheTitleAsKey` | Cumplido |
| 24.40 | `compose.go` | `TestComposeRejectsOperationsOnDuplicatedIDsButKeepsTheRestOperable`, `TestBlockDuplicatesAreGroupedByKey` | Cumplido |
| 24.41 | `document.go` (bloque y cola contigua), `compose.go` | `TestBlockBoundaryDependsOnTheHeadingLevel`, `TestComposeAddedKeepsTheContiguousTailWithItsRequirement`, `TestComposeWarnsOncePerTailSection` | Cumplido |
| 24.42 | `specbaseline/delta.go`, `compose.go`, `plan.go`: `Check`; `internal/cli/sdd_integrate.go`; skill `sdd-spec` | `TestDeltaParse*`, `TestComposeModifiedBodyArrivesWholeWithTheTemplateHeading`, `TestComposeRejectsBodyHeadingsThatWouldCloseTheBlock`, `TestCheckReportsEveryConflictingBodyWithItsLine`, `TestIntegrateCheckExitsNonZeroOnAConflictingHeading`, `TestIntegrateCheckDuringTheSpecPhaseNeedsNoGateNorAuthorization`, pin de `sdd-spec` | Cumplido (la parte de skill vive en C3 y está fijada por pin) |
| 24.43 | `compose.go` (ADDED), `ids.go`: `PlanIDs`; `sdd_archive_compose.go` | `TestComposeAdded*` (27), `TestPlanIDs*`, `TestRunSDDArchiveComposeDocumentAddsARequirementWithItsDefinitiveID` | Cumplido |
| 24.44 | `compose.go` (MODIFIED, REMOVED y RENAMED por ID) | `TestComposeModifiedReplacesOnlyTheBlockAndDropsTheNotes`, `TestComposeRemovedDeletesOnlyTheBlockAndKeepsItsTail`, `TestComposeRenamedWithIDIdentityIsRejected`, `TestComposeRenamedWithTextIdentityRetitlesTheBlock` | Cumplido (sin etiqueta `REQ-` en las pruebas) |
| 24.45 | `document.go` (líneas, EOL por línea, BOM) | `TestComposeKeepsEveryByteOutsideTheEditedBlocks`, `TestComposeKeepsTheBOMAndNeverAddsOne`, `TestComposeEmitsTheEditedBlocksWithTheDocumentEOL`, `TestDocumentRoundTripKeepsEveryByte` | Cumplido |
| 24.46 | `compose.go`, `plan.go`: `ApplyCompose` | `TestComposeIsAllOrNothing`, `TestComposeAddedIsAllOrNothing`, `TestApplyComposeIsAllOrNothingWhenARenameFails` | Cumplido |
| 24.47 | `sdd_archive_compose.go` (`--cwd` y `--document`); skill `sdd-archive` | `TestRunSDDArchiveComposeDocument*` (excluyente con `--canonical`, prosa, sin baseline, rutas), `TestSDDArchiveProjectBaselineBranch` (la skill no plantilla rutas) | Cumplido (la parte de skill vive en C3 y está fijada por pin) |
| 24.48 | `plan.go`, `sdd_archive_compose.go` | `TestRunSDDArchiveComposeDocumentRejectsUndeclaredDomains`, `TestCheckRejectsAnUndeclaredDomainWithoutCreatingIt` | Cumplido |
| 24.49 | `plan.go`: `Prepare`, captura de repositorios; `gitview.go` | `TestCaptureRepositoriesStopsOnAnyStateOfAnAffectedDocument`, `TestCaptureRepositoriesKeepsForeignChangesInThePhoto`, `TestCaptureRepositoriesStopsWhenAnAffectedDocumentIsNotInAGitRepository`, `TestPrepareStopsOnGitPreconditionsWithoutWritingAnything` | Cumplido |
| 24.50 | `plan.go` (entradas dirigidas), `verify.go`; referencia de `sdd-archive` | `TestVerifyAnchorsOfTheDirectedSection`, `TestVerifyPassesAndMarksTheDirectedAndContextEntriesIntegrated`, pin de la referencia | Cumplido (la edición por agente no es comprobable en una prueba unitaria: la cubren el verificador nativo y el pin) |
| 24.51 | `specbaseline/verify.go`: `Verify` | `TestVerify*` (26: presencia, anclas, alcance, traza, IDs, `INDEX.md`, `reference`, `no_impact`, índice de git intacto) | Cumplido |
| 24.52 | `specbaseline/trace.go`; `sdd_integrate.go` (`integrate show`) | `TestRenderTrace*`, `TestIntegrateShowRendersTheTraceInMarkdownAndJSON`, `TestIntegrateFullCycleEndsWithTheTraceOfTheArchivedChange` | Cumplido |
| 24.53 | `plan.go`, `verify.go` (modo `register`, estado `pending`) | `TestVerifyScopeRejectsEveryPathOutsideTheAllowedSet` (documento `register`), `TestVerifyPassesAndMarksTheDirectedAndContextEntriesIntegrated` (una entrada `register`) | Cumplido |
| 24.54 | skills `sdd-explore`, `sdd-propose`, `sdd-spec`, `sdd-onboard`; `sdd_integrate.go` | pins de `assets_test.go`, `TestIntegrateCheckTextReport` | Cumplido (pin de texto) |
| 24.59 | `ids.go`: `IDMap.Substitute`, `CountProvisional`; `plan.go`; `verify.go` | `TestSubstitute*`, `TestPlanIDsStops*` (cuatro causas), `TestVerifyRejectsAProvisionalIDLeftInTheAddedText`, `TestComposeSubstitutesProvisionalIDsOnlyInTheIntegratedText` | Cumplido |
| 24.61 | `delta.go`, `compose.go`: `omit` solo en ADDED y MODIFIED | `TestComposeScenariosOmitIntegratesOnlyTheDescription`, `TestComposeScenariosOmitDoesNotChangeRemoved`, `TestComposeScenariosIncludeIntegratesTheWholeBody`, `TestOmitScenarios` | Cumplido |

### 4.5 sdd-archive-lifecycle (5), sdd-block-review-gates (2), organic-agent-trigger-rules (1)

| REQ | Implementación | Pruebas | Estado |
|---|---|---|---|
| 21.16 | `sdd_gate.go` llama a `verifyGateEvidence(loc.WorkspaceRoot)`; bloqueo de `archive` en el status | `sdd_gate_test.go` (`pr_merged` con ancestro confirmado y sin confirmar, previas); el status real de este cambio bloquea `archive` por la compuerta `integration` | **Parcial (prueba)**: mismo hueco que REQ-24.12 (W1) |
| 21.17 | skill `sdd-archive` y referencia; `archive-compose --change --in-place`; `integrate show` | `TestSDDArchiveProjectBaselineBranch`, `TestSDDArchiveProjectBaselineReferenceDescribesTheFlows`, `TestInPlaceAppliesThePlanPrepareFixedAndRefusesAReapplication`, `TestIntegrateFullCycleEndsWithTheTraceOfTheArchivedChange` | Cumplido |
| 24.55 | `plan.go` (documentos nuevos), `anchors.go`, `verify.go` (`created`) | `TestPrepareAddsContextEntriesForANewMaintainedDocumentAndRecordsTheOptions`, `TestVerifyRecordsCreatedForTheAddEntryThatCreatedTheDocument`, `TestContextScope*` | Cumplido |
| 24.56 | causas (a), (f) y (g) en `workspace_declarations.go`; (b) a (e) en `plan.go` y `verify.go`; tabla en la referencia de `sdd-archive` | `TestArchiveBlockedWithoutTheContextDelta`, `TestArchiveBlockedByAnInvalidDeclaration`, `TestArchiveBlockedByAFilelessStore`, `TestPrepare*`, pin de las causas (a) a (g) | Cumplido |
| 24.57 | `verify.go` (`no_impact`), `trace.go` | `TestVerifyNoImpactDecisionRejectsAnEditOfTheMaintainedPaths`, `TestPrepareRecordsTheNoImpactDecisionWithoutNeedingAuthorization`, `TestIntegrateShowRecordsTheNoImpactDecisionWithItsReason` | Cumplido |
| 21.9 | `_shared/sdd-orchestrator-sections.md` (sección de compuertas, 12 plantillas) | `TestSharedGateSectionPresentsAndRemediatesTheContextDelta`, goldens regenerados, constante SHA-256 de Kilocode | Cumplido (pin de texto) |
| 21.11 | ídem | ídem | Cumplido (pin de texto) |
| 24.58 | `internal/components/agentguidance/routing.go` (paso 7), `.codex/AGENTS.md` | `TestRenderRoutingClosesODDWithTheMaintainedTechnicalContext` (todos los adaptadores, determinista) | Cumplido |

### 4.6 workspace-topology (4), living-documentation (7), multi-project-hub (1), tui-ui-parity (1)

| REQ | Implementación | Pruebas | Estado |
|---|---|---|---|
| topology 1.2 | `cmd/axiom/main.go`: `runWorkspaceValidate` | `TestWorkspaceValidate*` (8) | Cumplido |
| topology 2.1 | esquema `axiom.yaml` con los dos bloques opcionales | `TestDeclarationsUnknownKeysOutsideTheNewBlocksAreIgnored`, `TestDeclarationsValidBlocksAreLoaded` | Cumplido |
| topology 2.3 | `locate.go`: `Roots`, `Locate` | `TestLocateRootsByTopology`, `TestLocateCanonicalResolvesTheSamePlanningHomeFromEveryStart` | Cumplido |
| topology 2.4 | ídem, con `workspacetest.Canonical` | `TestLocateRootsByTopology`, `TestLocateAmbiguousPlanningHome` | Cumplido |
| livingdoc 1.1 | `livingdoc/service.go` (`<planning home>/specs`) | `TestServiceReadsThePlanningHomeRegardlessOfSpecsRepository` | Cumplido |
| livingdoc 1.2 | `livingdoc/indexer.go`: `RenderIndexMarkdown` con nombre de proyecto | `TestRenderIndexMarkdownUsesTheProjectName`, `TestServiceTakesTheProjectNameFromTheWorkspace` | Cumplido |
| livingdoc 2.1 | `livingdoc` coldstart sobre el `workspaceRoot` resuelto | `TestColdStartUsesTheResolvedWorkspaceRoot`, `TestSyncAndColdStartRejectAProjectBaseline` | Cumplido |
| livingdoc 3.1 | `axiom archive sync` con guarda | `TestArchiveSyncAndColdstartAreRejectedWithAProjectBaseline`, `TestSyncAndColdStartWithoutAProjectBaselineStillWrite` | Cumplido |
| livingdoc 3.4 | `axiom archive coldstart` con guarda | ídem | Cumplido |
| livingdoc 4.1 | `dashboard/server.go` (200 con `external_baseline`, 4xx en detalle, 409 en sync) | `TestArchiveSpecsWithAProjectBaselineReportsItAsExternal`, `TestArchiveSpecDetailOfADeclaredDomain`, `TestArchiveSyncIsRejectedWithAProjectOrIndeterminateBaseline` | Cumplido |
| livingdoc 4.2 | `dashboard/assets/app.js` (aviso y rechazo) | `TestLivingDocsTabPresentsTheExternalBaseline` (marcadores del texto, sin ejecutar JS) | Cumplido (S2) |
| hub 2.2 | `internal/hub/init.go` | `TestInitRejectsAnOwnPointerToAnotherPlanningHome`, `TestInitRejectsAnOwnPointerThatCannotBeResolved`, `TestInitAcceptsAnOwnPointerToItsOwnPlanningHome`, `TestInitIgnoresAPointerInAnAncestor`, `TestInitWithAProjectBaselineSkipsTheSpecs`, `TestInitForceKeepsAConfigWithDeclarations`, `TestInitReportsTheProjectBaselineAndRejectsAForeignPointer` | Cumplido |
| tui 17.10 | `internal/tui/screens/living_doc.go`, `model.go` | `TestLivingDocScreenShowsAnExternalBaseline`, `TestLivingDocScreenWithoutAProjectBaselineKeepsItsPresentation`, `TestLivingDocSyncShowsTheRejectionWithItsContinuation` | Cumplido |

Recuento: 79 requisitos = 12 + 15 + 12 + 19 + 8 (archive, gates y ODD) + 13 (topology, living-documentation, hub y TUI). 77 cumplidos, 2 parciales (24.12 y 21.16), 0 no cubiertos, 0 fuera de alcance. Los gaps aceptados de REQ-24.42 y REQ-24.47 (partes que viven en skills de C3) se comprobaron: están cubiertos por los pins de `sdd-spec` y `sdd-archive`.

## 5. Coherencia con el diseño (D-01 a D-22)

Comprobado en código y pruebas; no se encontraron desviaciones de fondo no registradas.

- D-01 a D-03: `Locate`, `LocateAt`, `ResolveOwnPointer`, `Roots` y `parseLayered` con la firma `(cfg, baseErr, decl)` existen; sin variable de entorno de techo (solo `DiscoveryCeilingOverride`, fijada por los `TestMain`); `resolveGovernanceChangeRoot` devuelve `(Location, changeRoot, error)`.
- D-04: `workspace.Validate` conserva su firma y las comprobaciones de identidad y enlaces simbólicos viven en `ValidateDeclarationPaths` (decisión m9 de la tabla).
- D-05: decorador fuera de `baseStatus`; causas en `notes` y en `blockedReasons` solo cuando la ruta sería `archive`; `TestSDDStatusV2CleanBreak` pasa sin cambios. El status real de este cambio lo confirma: `notes` vacío y un único bloqueo (`integration`).
- D-06 y D-07: una sola implementación de la consulta (`Declarations.Ownership`), guardas en todos los creadores y resolución en cada llamada en dashboard y TUI.
- D-08: `internal/specbaseline` nuevo y los cuatro parsers de `### Requirement:` sin tocar.
- D-09 a D-12, D-15 y D-16: modelo por líneas, bloque y cola, IDs, plan, verificador sobre git de solo lectura (con `GIT_INDEX_FILE` sobre copia, decisión registrada) y delta de contexto con anclas.
- D-13, D-14 y D-17: ciclo `prepare`, `archive-compose --in-place`, `verify` y `show` probado (`TestIntegrateFullCycle...`); regla de modelo del archivado en la plantilla de Claude Code, `project-baseline-convention.md` y `references/project-baseline-integration.md`.
- D-18 y D-19: sección compartida de compuertas con la remediación por línea fechada, sin tocar `gateQuestion`; la oración del paso 7 de ODD coincide con el texto del diseño.
- D-20: los tres ratchets de refusals (`TestEveryProductionRefusal`, `TestRefusalRatchet`, `TestEveryNamedReviewContinuation`) pasan en esta fase; deadcode y gofmtcheck son evidencia previa.
- D-21 y D-22: causa (g) y `workspace validate` por almacén; `sdd-tasks` solo gana la línea de prohibición (+1); `openspec/INDEX.md`, `openspec/config.yaml`, `AGENTS.md` de la raíz, `.kiro/` y `reviewtransaction` sin cambios.

Divergencias respecto a lo previsto, todas ya registradas en la tabla de decisiones de `tasks.md` o en `pr-description.md` (D-12 separación, tiempo de pruebas, índice de git en verify, regeneración de `.codex/AGENTS.md`, m9, m11, m12, m13, `--amend` de `296b7b40`, constante SHA-256 de Kilocode). Una precisión: el crecimiento de `internal/cli/sdd_archive_compose.go` (+397/-19) y de `cmd/axiom/main.go` (+161/-57) frente a «costuras pequeñas» consta solo en la tarea 8.2, no en la tabla de decisiones ni en `pr-description.md` (S3).

## 6. Hallazgos

### CRITICAL

Ninguno.

### WARNING

- **W1. Escenario `pr_merged` del espacio canónico sin prueba (REQ-24.12 y REQ-21.16).** El rechazo de un commit presente solo en `repo-backend` se cumple por construcción (`internal/cli/sdd_gate.go:63` toma `loc.WorkspaceRoot` y `verifyGateEvidence` abre el grafo de ese repositorio), pero ninguna prueba nueva lo ejerce con la fixture canónica; las pruebas de `sdd_gate_test.go` son anteriores y usan un solo repositorio.
- **W2. La rama está 4 commits por detrás de `main` local.** `main` avanzó a `8cf608bd` (#130, #131, #132 y #134) sobre la base `902b4487`. Coincide un fichero modificado por ambos lados, `testdata/golden/sdd-opencode-multi-settings.golden`, y `main` también tocó `internal/components/sdd/inject.go` y sus pruebas: es probable que el PR #133 necesite rebase y regenerar el golden (y revisar la constante SHA-256 de Kilocode) antes de fusionarse. No se evaluó la fusión.
- **W3. Límites de verificación en Windows.** Pruebas de enlaces simbólicos omitidas (ejecutadas en WSL durante C2, no repetidas aquí), 860 pruebas `review*_test.go` de `internal/cli` y `internal/reviewtransaction` completo sin ejecutar (la rama no los toca), `bench/` con 19 fallos idénticos a la base, `go test -race` no disponible y cobertura no medida.
- **W4. `.gates.lock` sin seguimiento y sin ignorar.** `openspec/changes/inc-24-spec-baseline-profile/.gates.lock` es un fichero de ejecución de las compuertas (no figura en `git ls-files`). No debe entrar en el commit del informe.

### SUGGESTION

- **S1.** `pr-description.md` quedó desfasada: dice «Sin push ni PR abierto», cita la cabeza `9f4bd977` y «61 commits» (y más abajo «63 por delante»), mientras que la cabeza real es `6e8d824b` con 64 commits por delante de `main` y el PR #133 ya está abierto.
- **S2.** REQ-4.2 (interfaz web) y la parte de `app.js` de REQ-24.25 solo se verifican por marcadores de texto, sin ejecutar JavaScript.
- **S3.** Anotar en la tabla de decisiones o en la descripción del PR el tamaño real de las costuras de `sdd_archive_compose.go` y `main.go`, hoy solo en la tarea 8.2.
- **S4.** Trazabilidad: REQ-24.16 no tiene etiqueta `REQ-` en el código y REQ-24.44 solo en `compose.go`; añadirlas en las pruebas facilitaría auditorías futuras (no afecta al cumplimiento).

## 7. Estado nativo

`axiom sdd status inc-24-spec-baseline-profile --cwd . --json` (ejecutado antes de persistir este informe): `taskProgress` 186/186, `applyState: all_done`, `dependencies.verify: ready`, `dependencies.archive: blocked`, `nextRecommended: resolve-blockers`, `notes` vacío.

`blockedReasons`: la compuerta `integration` no está aprobada; se resuelve con `axiom sdd gate record --gate integration --decision approved --evidence-kind pr_merged --commit <sha>` (o `deployment` o `attestation`) cuando se fusione el PR #133 (REQ-21.16). Este informe no intenta resolverlo.

Recomendación: la implementación está completa; el siguiente paso es `sdd-archive`, condicionado a la fusión del PR y al registro de la compuerta `integration`.
