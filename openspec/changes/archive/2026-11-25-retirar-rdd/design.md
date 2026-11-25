# Diseño: Retirada completa de RDD y de `bench/` (C-01-e)

## Enfoque técnico

Se aplica el enfoque B de la propuesta: borrado de fuera hacia dentro, una PR (`single-pr`) con un commit por unidad, y cada commit deja `go build ./... && go vet ./...` en verde y pasa `go test` de los paquetes que toca, **enteros**. No se compila ningún binario suelto de `axiom` (el EDR los borra).

El inventario de la exploración (`explore.md`, `3c520778`) se ha contrastado con el árbol y con las importaciones reales. Ese contraste cambia tres cosas respecto a la propuesta, todas de **orden y frontera**, no de alcance:

1. **El orden de commits no sigue la numeración 0-9.** La propuesta borraba proveedores antes que sus consumidores: `cli/review_mode.go` consume `state.RDDMode` y `hub.SetProjectRDD` (unidad 4); `providercontractbundle` consume `sdd.ReviewExecutionContractFor` de `boundedreview.go` (unidad 5); `e2e/organicruntime` importa `reviewerprovider` y ejecuta los verbos `review` (unidades 6 y 7); y los tests de `cli/review*` leen en tiempo de ejecución los activos `review-*` que borra la unidad 5. Se conserva la numeración como identidad de unidad y se fija otro orden de commits (D1).
2. **La unidad 7 se parte en dos.** El contrato del proveedor (`providercontractbundle*`, pipeline de release, `releasepolicy`, `releaseprovenance`) va en un commit atómico temprano (7a); `reviewtransaction`, `reviewerprovider` y `rddharness` se borran al final (7b) (D2).
3. **`boundedreview.go` no es RDD puro.** Aloja el renderizador de todos los activos gestionados (`renderSDDOrchestratorAsset`, `renderBoundedReviewAsset*`, `bindRuntimeAgentIdentity`, `researchLifecycleContract`, `judgmentDayReviewerContract`, `replaceAgentBody`). Se parte (D6).

Piezas que la propuesta no nombraba y que se incorporan: el job `claude-network-none` y `e2e/Dockerfile.claude-network-none`; la verificación de `sync` que **falla** si existe el plugin heredado `review-result-artifacts.ts` (`cli/sync.go:2642`); funciones de producción de `cli` definidas en ficheros `review_*`; `internal/releaseprovenance` (`provider_contract_semver`); la ruta `/api/project/review-mode` en `dashboard/secrets_canary_test.go:45`.

**Regla de commit.** Cada test que valida un activo o un comportamiento que se borra sale, como tarde, en el mismo commit que lo borra. La comprobación de cada commit ejecuta los paquetes que toca **enteros**: `-run` solo aparece en las órdenes que regeneran baselines, y siempre va seguida de la ejecución completa del paquete. Ningún filtro sirve para ocultar tests en rojo.

## Orden de commits

| Commit | Unidad | Título | Por qué en este lugar |
|---|---|---|---|
| C1 | 0 | Desacoplar lo no RDD | No borra nada; deja a los demás commits como borrados puros. |
| C2 | 1 | Retirar `bench/` | Elimina el único consumidor de `-tags rdd_harness` y de `bench_fixture`. |
| C3 | 2 | Scripts y CI RDD puros | Sin consumidores en Go. Los shards **no** van aquí (D8). |
| C4 | 8 | Cirugía de `e2e/organicruntime` | Debe preceder a C8 (verbos `review`), C9 (activos) y C11 (`reviewerprovider`). |
| C5 | 7a | Release sin contrato del proveedor | Debe preceder a C9, que borra `ReviewExecutionContractFor`. |
| C6 | 3 | TUI | Consumidor puro. |
| C7 | 4a | Dashboard | Consumidor puro de `hub`. |
| C8 | 6 | `cli/review*`, despachadores, `branch` | Sus tests leen los activos `review-*`: salen antes que ellos. No depende de nada que borre C9 (de `components/sdd` solo usa `sdd.Inject`). |
| C9 | 5 | `components/sdd`, activos, skill y contratos RDD de `capabilitymanifest` | Sin consumidores ya; `boundedreview.go` es el último consumidor de los contratos de capacidad RDD. |
| C10 | 4b | `hub` y `state` sin `RDDMode*` | Sus consumidores desaparecieron en C7 y C8. |
| C11 | 7b | Paquetes RDD y su CI | `reviewtransaction`, `reviewerprovider` y `rddharness` ya no tienen importadores. |
| C12 | 9 | Docs, baseline de deadcode, `go mod tidy`, ausencia | Cierre. Las specs vivas no se tocan aquí (D14). |

## Decisiones de arquitectura

### D1: Orden de commits por dependencias verificadas

**Elección**: la tabla anterior. Cada commit borra solo proveedores cuyos consumidores ya se borraron antes o en el mismo commit.
**Alternativas consideradas**: la numeración 0-9; juntar 4, 5 y 6 en un solo commit.
**Justificación**: con 0-9, `go vet ./...` falla en las unidades 4 y 5 (`cli/review_mode.go`, `providercontractbundle/bundle.go`), y la unidad 5 deja en rojo `go test ./internal/cli` porque `sync_review_agents_verify_test.go` y una docena de `review_*_test.go` leen los activos que borra. Juntar 4-6 daría un commit de ~80.000 líneas imposible de revisar o revertir por partes.

### D2: La unidad 7 se parte en 7a (release) y 7b (paquetes)

**Elección**: C5 retira en un solo commit `internal/providercontractbundle{,cmd}`, `contracts/review-provider-contract`, el hook y el archivo `review-provider-contract` de `.goreleaser.yaml` (y las entradas `docs/review-integration.md` y `contracts/review-integration/**` del archivo principal), `release.yml`, `promote-stable-rc.yml`, `scripts/{promote-stable-preflight,verify-release-assets}.sh`, `internal/releasepolicy/policy.go`, `internal/releaseprovenance{,cmd}`, `internal/update/{release_security,windows_distribution_policy}_test.go` y la línea `.goreleaser-provider-contract/` de `.gitignore`. C11 borra el resto de paquetes y `contracts/review-integration`.
**Alternativas consideradas**: todo al final, manteniendo `ReviewExecutionContractFor` vivo hasta entonces.
**Justificación**: pipeline, `releasepolicy` (que fija como texto `.goreleaser.yaml` y los pasos de `release.yml`) y tests de `update` cambian juntos o `go test ./internal/update ./internal/releasepolicy` rompe; esa atomicidad se conserva en C5. Adelantarlo libera `boundedreview.go` para C9.

### D3: `sddstatus` resuelve el git common dir con `gitcommon`

**Elección**: `OpenRuntimeStore` (`internal/sddstatus/runtime_ledger.go`) sustituye la sonda `reviewtransaction.CompactAuthoritativeStore(ctx, root, "sdd-runtime-probe")` y los cuatro `filepath.Dir` por:

```go
// runtimeGitCommonDir devuelve el git common dir de root en forma canónica:
// absoluto y con los enlaces resueltos, igual que lo resolvía la sonda RDD.
func runtimeGitCommonDir(ctx context.Context, root string) (string, error) {
	common, err := gitcommon.CommonDir(ctx, root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(common)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}
```

**Alternativas consideradas**: `CommonDir` sin `EvalSymlinks`; `gitcommon.CommonDirFromFiles`.
**Justificación**: la sonda devolvía `EvalSymlinks(Abs(--git-common-dir))` (`canonicalLocatorDirectory`); `CommonDir` devuelve `Clean(--path-format=absolute --git-common-dir)` sin resolver enlaces. Sin `EvalSymlinks`, `RuntimeStore.Dir` deja de ser la misma cadena en macOS (`/var` frente a `/private/var`) y en Windows con nombres 8.3: el directorio físico es el mismo, pero `root` y `workspace` sí se canonizan y las comparaciones de rutas dejarían de ser homogéneas. `CommonDirFromFiles` cambiaría la semántica que el fichero ya usa (`RepositoryRoot` ejecuta git). Se pierden dos comprobaciones de la sonda que eran de RDD: el rechazo ante el marcador de reconciliación de lotes de `review-transactions` y la verificación de identidad por `os.SameFile`; el almacén SDD ya comprueba cada segmento con `Lstat` en `ensureDirectories`.

### D4: Selección persistida con una skill sin activo embebido

**Hallazgo**: hoy un id desconocido que no está en `retiredSkills` se **salta** al inyectar (`components/skills/inject.go:69-73`, con `log.Printf`), pero la verificación (`run.go:2826`) exige su `SKILL.md`, y `sync` falla y se revierte (el defecto que `TestRunSyncRetiredSkillInPersistedSelectionDoesNotFailVerification` fijó para `branch-pr`).
**Elección**: filtro genérico, sin lápida. `internal/components/skills` gana `Installable(ids []model.SkillID) []model.SkillID`: conserva el orden, no comparte el slice y quita los retirados y los ids sin directorio embebido `skills/<id>` en `assets.FS`. `cli.selectedSkillIDs` lo usa en lugar de `WithoutRetired`, que se borra. `IsRetired` y `RetireInstalled` se quedan. **Lectores de la selección persistida**: install, sync y los dos caminos de `compatibility_*` pasan todos por `selectedSkillIDs` (`run.go`, `sync.go:932,1439`, `compatibility_skills.go`, `compatibility_transaction_windows.go`), así que quedan cubiertos con un solo punto; el selector de la TUI parte del catálogo, no de la selección guardada (`tui/model.go:4844`), y `doctor_skills.go` inspecciona directorios instalados, no la selección. Ninguno necesita otra llamada; la prueba de C9 lo fija para install y sync.
**Alternativas consideradas**: entrada en `retired.go` (lápida, contraria a P20); filtrar por `catalog` (dejaría fuera ids embebidos que no están en él, como `hermes-ephemeral-delegation`); filtrar al leer `state.json` (no cubre `--skills`).
**Justificación**: la verificación pasa a coincidir con lo que la inyección ya hace y solo afecta a ids que hoy rompen `sync`; al filtrar antes de inyectar desaparece también el aviso del log.

### D5: Lectura tolerante por eliminación de campos y centinelas de máquina

**Elección**: se borran `InstallState.RDDMode`, `RDDModeRecordedAt` (con su copia en `merge`, `ProjectView` y `WithMachineFields`) y `hub.WorkspaceRecord.RDDMode`. `state.UnmarshalJSON` y `hub/manager.go` usan `encoding/json` sin `DisallowUnknownFields`: un `rdd_mode` en disco se ignora y desaparece en la siguiente escritura. Los tests con `RDDMode` como centinela de «un campo de máquina sobrevive a escritores concurrentes» pasan en C1 a `PendingSync: true`, o a `LastUpdateCheck` fijo si ya manipulan `PendingSync`; `update/cooldown_concurrency_test.go`, que prueba `LastUpdateCheck`, usa `PendingSync`. `TestInstallStateLockAcceptsSymlinkedHome` deja de escribir con `RunReviewMode` y usa un escritor no RDD que tome el mismo bloqueo (el de `TestPersistInstallStateMergesExplicitAgentsFromLatestState`).
**Alternativas consideradas**: campos `json:"-"`; un `UnmarshalJSON` que los descarte.
**Justificación**: serían lápidas en el modelo de datos sin aportar nada que `encoding/json` no haga.

### D6: `boundedreview.go` se parte; los goldens prueban la equivalencia

**Elección**: C9 crea `internal/components/sdd/asset_render.go` con lo no RDD, renombrado: `renderSDDOrchestratorAsset`, `renderBoundedReviewAsset{,Body,BodyFromContent}` → `renderManagedAsset{,Body,BodyFromContent}`, `bindRuntimeAgentIdentity`, `runtimeAgentIDPlaceholder`, `researchLifecyclePlaceholder`, `researchLifecycleContract`, `judgmentDayReviewerContract`, `nativeReviewerResultSchema`, `replaceAgentBody`, `replaceBoundedReviewSection` → `replaceManagedSection`. `renderManagedAssetBodyFromContent` conserva solo: la sustitución de `{{GENTLE_AI_RESEARCH_LIFECYCLE}}`, la de `## Review ledger contract` por `judgmentDayReviewerContract()` en `*/agents/jd-judge-*`, el paso sin cambios de `jd-fix-agent` y la retirada de `## Review ledger contract` del resto. Se borra lo demás de `boundedreview.go` y `rdd_retire.go` entero.

La prosa RDD se quita **en la fuente** (los 12 `sdd-orchestrator.md`, `skills/_shared/sdd-orchestrator-sections.md`, `claude|opencode/commands/sdd-apply.md`) y se borran los recortadores (`removeBoundedReviewSection`, `removeSuspendedAuthorityFirstProcedure`, `suspendedSharedSkillContent`, `stripPreservedOpenCodeReviewExecutionContract`, `stripOpenCodeReviewAgents`). Equivalencia en dos pasos dentro de C9:

1. **Mecánico**: se quita de la fuente exactamente lo que hoy recortan los recortadores. `go test ./internal/components -run TestGolden -count=1` **sin** `-update` pasa: lo instalado es byte a byte lo de hoy.
2. **Prosa**: se poda lo que hoy sí llega al binario («not a `axiom review capture-result` lens artifact» de `judgmentDayReviewerContract`, «Do not call `axiom review status`» de los `sdd-verify.md`, el párrafo «SDD does not retain, read, or persist review lineage, receipt…» de los `sdd-apply.md`, `judgment-day/references/prompts-and-formats.md:18-21`, `openspec-convention.md`). Goldens con `-update`; el diff solo toca esas frases.

**Alternativas consideradas**: borrar el fichero entero (no compila); conservar los nombres `renderBoundedReview*` (el nombre afirma una revisión que ya no existe).

### D7: Los helpers compartidos se mueven en C1

**Elección**: C1 mueve, sin cambiar comportamiento, lo que vive en ficheros RDD y usan ficheros que se quedan:

| Símbolo | Hoy en | Lo usa | Destino en C1 |
|---|---|---|---|
| `managedAssetDigest` (prod) | `review_asset_provenance.go` | `run.go`, `sync.go`, `doctor_project.go` | `managed_asset_digest.go` |
| `reviewConsoleInteractive` y `reviewConsoleTerminal` (prod) | `review_mode.go:878-888` | `console_interactive.go` | `console_interactive.go` (`consoleInteractive`, `consoleTerminal`) |
| `reviewGentleAIVersionAndCommit` (prod) | `review_defect_report.go` | `doctor.go` | `build_version.go` (`buildVersionAndCommit`) y `build_version_test.go` con la parte no RDD de `review_version_consistency_test.go` |
| `runReviewCLIGit`, `initReviewCLIRepo`, `canonicalReviewCLITempDir`, `containsString` | `review_test.go`, `review_schema_test.go` | `sdd_planning_home_test`, `sdd_status_continue_consent_test`, `sdd_attempt_test`, `sdd_attempt_marker_binding_test` | `gittest_helpers_test.go` (`runCLIGit`, `initCLIGitRepo`, `canonicalCLITempDir`, `containsString`) |
| `TestDualContract_SDDStatusDualSerialization` | `dual_contract_integration_test.go:344` | — (REQ-46.7) | `sdd_status_serialization_test.go`, como `TestSDDStatusDualSerialization` |

Los ficheros `review_*` pasan a llamar a los nombres nuevos y se borran en C8. En `components/sdd`, `assertTextContainsClauses` y `readGentleOrchestratorPrompt` (`review_ledger_contract_test.go`) se mueven a `helpers_test.go` en C9. Lo que esta búsqueda no ve aparece con `go vet` del paquete en el commit que borra su fichero; se mueve, nunca se reimplementa.

### D8: Los shards de Windows se retiran con el código que cubren

**Elección**: `TestWindowsFullSuiteShardsCoverEveryTestName` exige que un único shard reclame cada test y que ningún shard quede vacío; por eso los shards cambian en el commit que borra sus tests. C8 sustituye `cli-review-a-m`, `cli-review-n-z` y `cli-r-other` por `cli-r` (`^TestR`). C11 borra `reviewtransaction-{a-c,d-k,l-z}` y la exclusión `/internal/reviewtransaction$` del shard `rest`. Matriz final: `cli-a-m` (`^Test[A-M]`), `cli-n-o` (`^Test[N-O]`), `cli-p-q-s-z` (`^Test[P-Q]|^Test[S-Z]`), `cli-r` (`^TestR`), `rest` (todo salvo `/internal/cli$`). La cabecera (10 shards) y la tabla de costes se reescriben, con los costes pendientes de volver a medir tras el merge.
**Alternativas consideradas**: retirar los shards en C3; tres shards de `cli`.
**Justificación**: en C3 los `TestReview*` siguen existiendo y la guarda falla. Fusionar solo los R no puede empeorar el peor shard (`cli-p-q-s-z`, 48,6 s en Linux con los tests RDD dentro).

### D9: La cirugía de `e2e/organicruntime` va antes que verbos y activos

**Elección**: C4 conserva `TestOrganicDirectoryIdentityAcceptsCanonicalAliases`, `TestOrganicConfiguredAgentReceivesRoutingGuidance{Cursor,RealAgents}`, `TestOrganicEnvironmentDisablesPersistentPathWrites`, los cuatro `*InstalledSDDApplyExecutor*`, `opencode_trace_test.go` y `sdd_consent_wiring_e2e_test.go` entero. Reescribe sin los pasos de revisión `TestOrganicImplementationRoutesReachDelivery` (fuera `startReview`, `approveReview`, `gate`), `TestOrganicOptionalSDDDeclineAndAccept` (fuera la aserción sobre `ReviewGate`) y `TestRealAgentOrganicJourneys` (fuera los turnos `review start|finalize|validate` y la puerta; quedan commit, marcador, subagente y ausencia de artefactos SDD). Borra los tests RDD de la exploración, sus helpers, la consulta `review mode status` de `assertOrganicDelegatedWorkerStaysInRoute`, la importación de `reviewerprovider` y `-tags rdd_harness` de `buildOrganicBinary`. Mismo commit: job `claude-network-none` y su Dockerfile; en `organic-runtime-e2e`, el paso «Verify live OpenCode provider isolation E2E is registered», la instalación de Codex y `GENTLE_AI_{CODEX,OPENCODE}_RUNTIME_E2E` y `GENTLE_AI_LARGE_WORKSPACE_E2E` (no la lee ningún test); en `formatter_ordering_test.go`, `unit-tests` pasa a `next: "windows-runtime"` y `TestOrganicRuntimeE2EUsesInstalledOpenCodePin` solo exige la instalación fijada de OpenCode.

### D10: Ejes de `bench/` no RDD: pérdida aceptada, eje a eje

| Eje | Equivalente en `go test` | Decisión |
|---|---|---|
| `model-picker` (j97, #3262) | `components/sdd/inject_test.go:4420-4456,5911`; `tui/screens/model_picker_test.go` (`TestRuntimeModelPickerStateDiscoversCustomAgents`, `TestHandleModelPickerNav_CustomAgent*`) | Se acepta la pérdida del recorrido de caja negra con `-tags bench_fixture`; la regla y la persistencia están cubiertas. |
| `sdd-task-result` | `sddtaskresult/{classify,transport_parity,preflight_question_harness}_test.go` | Se acepta: nunca corrió en CI (`GENTLE_AI_BENCH_SDD_PLUGIN` no está en `ci.yml`). |
| `journeys_sdd*` | parte no RDD en `sddstatus` | Cobertura RDD: autoridad, recibos e interruptor. |
| `journeys_managed_assets` (j93) | `cli/{run_state,project_state}_test.go` | Cobertura RDD: STATUS v2 de revisión. |

### D11: `releaseprovenance` conserva `v1` y pierde el campo del proveedor

**Elección**: se borran `Input.ProviderContractSemver`, `provider_contract_semver`, la entrada `provider-contract` de `artifacts` y su validación; esquema y asset siguen en `v1`.
**Alternativas consideradas**: subir a `v2` (renombra el asset en seis sitios).
**Justificación**: solo lo producen y verifican piezas que cambian en C5; no hay consumidor externo documentado. Se anota en las notas de la versión.

### D12: Ratchets y baselines, con su mecanismo y en la unidad que los mueve

| Artefacto | Mecanismo | Commits |
|---|---|---|
| `TestRefusalRatchetVocabularyMatchesBenchClassifier` | se borra (lee `bench/classify.go`) | C2 |
| `.refusal-ratchet-baseline.txt` | `GENTLE_AI_REFUSAL_RATCHET_UPDATE=1 go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign -count=1 -timeout 10m` | C8 y C11 (antes se quita `reviewtransaction` de `refusalRatchetProductionDirs` y los datos sintéticos de `:654-659` pasan a una ruta neutra) |
| `.guard-population-baseline.txt`, `guardPopulationCount` (6→2), `guardPopulationProductionDirs` | `GENTLE_AI_GUARD_POPULATION_UPDATE=1 go test ./internal/cli -run TestEveryRegisteredGuardPopulationDeclarationMatchesProduction -count=1` | C11, con `docs/architecture/guard-population.md` |
| `.deadcode-baseline.txt` | `bash scripts/deadcode-ratchet.sh --update` | C12; una función que aparezca **nueva** como inalcanzable se borra, no entra en el baseline |

Ningún baseline se edita a mano.

### D13: Una sola guarda de ausencia, en los activos

**Elección**: se borran las guardas de frontera y suspensión RDD (`kickoff/rdd_boundary_test.go`, los tests RDD de `sddstatus`, `components/sdd/rdd_*_test.go`, `cli/sdd_no_review_test.go`, `cli/sdd_attempt_rar_mode_read_refusal_{unix,windows}_test.go`, la parte RDD de `assets/independent_review_boundary_test.go`, `agentguidance/routing_test.go:TestRenderRoutingOmitsRDDGuidance`, `agents/adapter_forbidden_construction_guard_test.go`) y C9 añade `internal/assets/retired_review_commands_test.go`: ningún activo embebido contiene `axiom review `, `gentle-ai review `, `review-ledger-contract` ni `rdd-defect-workflow`.
**Justificación**: el único vector de reintroducción es la absorción de activos de upstream, que sigue teniendo RDD.

### D14: Documentos históricos frente a vivos; specs vivas

- **Se borran**: `docs/review-integration.md`, `docs/review-authority-threat-model.md`, `docs/architecture/{organic-rdd,rdd-backlog-disposition,rdd-freeze-expansion-policy,rdd-ownership-inventory,rdd-root-simplification-design,rdd-wave7-deletion-proof-tracker,the-organic-rdd-story}.md`, `docs/audits/{2026-07-21-rdd-system-audit,2026-07-24-organic-rdd-recovery-plan}.md`, `docs/audits/data/organic-rdd-recovery/**`, `docs/testing/organic-rdd-testing-guide.md`, `docs/assets/diagrams/rdd-review.svg`.
- **Se podan**: `README.md`, `AGENTS.md`, `docs/{usage,trigger-rules,intended-usage,components,manual-de-inicio,agents}.md`, `docs/testing/windows-tests.md`, `docs/architecture/guard-population.md` (C11), `docs/assets/diagrams/sdd-cycle.svg`, `.github/PULL_REQUEST_TEMPLATE.md`.
- **Se conservan sin tocar** (registros fechados o cerrados): `docs/plan/**`, `docs/releases/**`, `docs/archive/**`, los informes fechados no RDD de `docs/audits/` (`2026-07-23-*`, `2026-07-27-*`, `2026-07-30-*`, `2026-11-03-*`), `docs/upstream-absorption-ledger.md` (registro cerrado en `v3.4.0`: podarlo lo falsearía; desviación expresa de la propuesta), `openspec/changes/archive/**`, `odd/tasks/**`.
- **Specs vivas**: `apply` no toca `openspec/specs/**`; los deltas de `sdd-spec` se integran al archivar.

La búsqueda de ausencia excluye esos históricos, `openspec/changes/retirar-rdd/**` y, hasta el archivo, `openspec/specs/**`.

### D15: Notas de migración

No hay `CHANGELOG` y el `changelog` de goreleaser solo lista asuntos de commit. C12 añade `docs/releases/notas-retirada-rdd.md` (texto en *Migración*), lo enlaza desde `README.md` y lo repite en la PR; quien publique la primera versión con el cambio lo pega en las notas de la release. Se descarta `release.footer` en `.goreleaser.yaml` porque `releasepolicy` lo fija byte a byte.

### D16: Requisitos trasladados (REQ-46.7 a REQ-46.10) se sostienen con tests no RDD existentes

**Elección**: los cuatro requisitos que `sdd-spec` traslada desde `rdd-decoupling-v3-stability` no necesitan código nuevo; se fijan a tests no RDD que ya existen y que se ejecutan enteros en los commits indicados:

| Requisito | Test que lo cubre | Commits donde se comprueba en verde |
|---|---|---|
| REQ-46.7 (`axiom-sdd-cli-integration`, decodificación estructurada en `StatusV2Projection`) | `TestSDDStatusDualSerialization` (trasladado en C1 desde `dual_contract_integration_test.go:344`), `sddstatus/status_v2_test.go:TestStatusRenderersEmbedOnlyStatusV2Projection` | C1, C8, C12 |
| REQ-46.8 (`axiom-user-state-and-env`, aislamiento del CWD) | `cli/workspace_root_test.go:TestRunSyncWorkspaceScopeWritesToTheProjectRoot` y `TestRunInstallWorkspaceScopeWritesToTheProjectRoot`, `cli/install_in_workspace_test.go:TestInstallInWorkspaceInstallsInTheExplicitRootWhateverTheCwd`, `cli/uninstall_workspace_test.go:TestUninstallWorkspaceFromASubfolderUsesTheProjectRoot` | C1, C8, C9, C12 |
| REQ-46.9 (`axiom-skills-index-governance`, `contributorSkills` fuera de los presets) | `components/skills/presets_test.go:TestSkillsForPresetExcludesContributorSkills` y el golden `testdata/golden/skills-presets.json` | C9 (quita `rdd-defect-workflow` de `contributorSkills`), C12 |
| REQ-46.10 (`engram-protocol-injection`, `ambiguous_project`) | `components/engram/protocol_test.go:TestProtocolAmbiguousProjectRecoveryDistinction` | C9 (`./internal/components/...`), C12 |

`TestDualContract_SchemaValidationWithCompiledSchemas` (`:254`) **no se conserva**: compila `operation.schema.json`, `failure.schema.json` y `capabilities-v2.7.schema.json` de `contracts/review-integration/v2` y valida `ReviewValidateResult` y las superficies de `axiom review`; todo lo que valida desaparece. El equivalente para el contrato publicado que queda (`contracts/sdd-integration`) ya existe: `sddstatus/consent_contract_test.go` (`TestSDDIntegrationConsentSchemaIsStrictAndBound`, `TestSDDIntegrationConsentFixtureValidates`). El resto de `dual_contract_integration_test.go` se borra en C8.

## Unidades de trabajo

Comprobación común: `go build ./... && go vet ./...`, y `go test` de **cada paquete tocado, entero**, con `-count=1` y `-timeout` explícito. Abajo solo se añaden comprobaciones específicas.

### C1 · Unidad 0: desacoplar

- **Edita**: `sddstatus/runtime_ledger.go` (D3); movimientos de D7; centinelas de D5 en `state/{state,store}_test.go`, `app/{app_test,state_lock_test}.go`, `components/uninstall/state_lock_test.go`, `cli/{install_state_lock,sync_persona_migration,opencode_background,project_state}_test.go`, `tui/{model,pi_background_model}_test.go`, `update/cooldown_concurrency_test.go`.
- **Crea**: `sddstatus/runtime_common_dir_test.go`, `cli/sdd_status_serialization_test.go`.
- **Comprueba**: `go test ./internal/sddstatus ./internal/state ./internal/app ./internal/components/uninstall ./internal/update ./internal/tui/... ./internal/cli -count=1 -timeout 60m`.

### C2 · Unidad 1: `bench/` (P21)

- **Borra**: `bench/`, `results.json`, `skills/axiom-bench/`, `internal/app/bench_model_picker{,_fixture}.go`, `TestRefusalRatchetVocabularyMatchesBenchClassifier`.
- **Edita**: `cmd/axiom/main.go:451-470` e `internal/app/app.go:108`; `ci.yml` (pasos de bench y comentarios del paso de humo); `.github/PULL_REQUEST_TEMPLATE.md`; fila `axiom-bench` de `AGENTS.md`; comentario de `sddtaskresult/classify_test.go:11-13`.
- **Comprueba**: `go test ./cmd/axiom ./internal/app ./internal/sddtaskresult ./internal/cli -count=1 -timeout 60m`.

### C3 · Unidad 2: scripts y CI RDD puros

- **Borra**: `scripts/{test-review-capabilities,test-review-contract-package,cross-lane-battery}.sh`, `scripts/crosslane/`, `scripts/{opencode_v2_loopback,test-opencode-v2-host,test_opencode_v2_host_test}.py`.
- **Edita**: `ci.yml`, job `opencode-v2-contracts`: fuera «Test bounded V2 host activation polling». `scripts/test-opencode-v2-contracts.sh` no cambia: copia `plugins-v2/*.ts` y deja de ver el plugin de revisión cuando C9 lo borra.
- **Comprueba**: `go test ./internal/assets -count=1 -timeout 15m`.

### C4 · Unidad 8: `e2e/organicruntime`

- Contenido en D9.
- **Comprueba**: `go test ./e2e/organicruntime -list '.*'` (la lista conservada aparece completa); `go test ./e2e/organicruntime ./internal/assets -count=1 -timeout 20m`.

### C5 · Unidad 7a: release sin contrato del proveedor

- D2 y D11. En `promote-stable-rc.yml`: fuera la salida `provider_contract_semver`, «Read immutable provider contract semver», «Verify source provider contract bundle», las `PROVIDER_CONTRACT_SEMVER` y el asset de la lista (`:137`). En `release.yml`: las tres lecturas de `CONTRACT_SEMVER` y «Verify provider contract bundle». En `promote-stable-preflight.sh:51-52,76` y `verify-release-assets.sh:28-42`: semver y asset. `releasepolicy/policy.go`: texto fijado (`:570-640`, `:720-790`) y comprobación de artefactos (`:375-428`).
- **Comprueba**: `go test ./internal/update ./internal/releasepolicy ./internal/releaseprovenance ./internal/releaseprovenancecmd -count=1 -timeout 15m`; `rg -n "provider.contract|providercontractbundle|PROVIDER_CONTRACT" .goreleaser.yaml .github scripts internal/releasepolicy internal/releaseprovenance internal/update` vacío.

### C6 · Unidad 3: TUI

- **Borra**: `tui/screens/{review_store_reset,review_mode,install_review_mode}.go`, `screens/install_review_mode_test.go`, `tui/review_store_reset_test.go`.
- **Edita**: `tui/model.go` (entrada de bienvenida `:2131` y bloques de `explore.md` §3), `tui/router.go:36-37`, `app/app.go:299-305,1144-1166`, `tui/main_test.go`, `tui/model_test.go` (cursor). `ScreenReview` no se toca. El menú de bienvenida no tiene golden: `tui/testdata` no cambia.
- **Comprueba**: `go test ./internal/tui/... ./internal/app -count=1 -timeout 20m`.

### C7 · Unidad 4a: dashboard

- **Borra**: `dashboard/service_review_mode.go`, `server_review_mode_test.go`, `assets_reviewmode_test.go`, `assets/js/core/reviewmode.js`.
- **Edita**: `server.go:79,348-380`, `service.go:938-956`, `service_adapters.go:43,107`, `types.go:237-241` (`ReviewMode`, `ReviewModeError`), `main_test.go`, y `secrets_canary_test.go:45` (la ruta `/api/project/review-mode` sale de `excludedRoutes`). Un cliente que envíe `review_mode` no recibe error (el decodificador no rechaza campos desconocidos).
- **Comprueba**: `go test ./internal/dashboard/... -count=1 -timeout 15m`; restaurar `openspec/INDEX.md`, que esos tests reescriben (defecto previo).

### C8 · Unidad 6: `cli/review*`, despachadores y `branch`

- **Borra**: `cli/review*.go` (prod y tests, incluido `review_stop_hook*`), `branch_lineage{,_test}.go`, `codex_review_provider_model_test.go`, `sync_review_agents_verify_test.go`, `sdd_status_review_disabled_test.go`, `sdd_status_reenable_fresh_review_test.go`, `dual_contract_integration_test.go` (tras el traslado de D7), `protocol_probe_test.go`, `continuation_paths_test.go`, `sdd_no_review_test.go`, `sdd_attempt_rar_mode_read_refusal_{unix,windows}_test.go`; `app/review_validate_nondeciding_test.go`; `kickoff/rdd_boundary_test.go`; tests RDD de `sddstatus` (D13) y `review_reverify_test.go` (vacío).
- **Edita**: `cmd/axiom/main.go` (ayuda `:174`, casos `:433-446`, `runReview` `:2303-2375`) y tests; `app/app.go:134-152` y tests de ayuda; `cmd/axiom/testmain_test.go`, `app/testmain_test.go`, `sddstatus/main_test.go` (fuera `rddharness` y `reviewtransaction`); `cli/branch_done.go` (`:38`, `newBranchLineageProbe` `:114`); `branch/ports.go:55-58`, `branch/engine/engine.go:89-110`, `branch/engine/done.go` y tests; `cli/project_state_test.go` (fuera `checkManagedReviewerAssets`); `windows-full-suite.yml` (D8, primera mitad); release-blockers de `cli`/`sddstatus` en `ci.yml` y `darwin-release-blockers.manifest`; baseline de rechazos (D12). Opcional: `sddstatus/review_gate.go` → `read_text.go`.
- **Comprueba**: `go test ./internal/cli ./cmd/axiom ./internal/app ./internal/branch/... ./internal/kickoff ./internal/sddstatus ./internal/assets -count=1 -timeout 75m`.

### C9 · Unidad 5: `components/sdd`, activos, skill y contratos de capacidad

- **`components/sdd`**: D6; fuera `rddharness.Enabled()` y sus ramas de `inject.go` (Stop/SessionStart, migración de `review-result-artifacts.ts`, `isReviewSubAgentFile`, `opencode.Review*`), `orchestrator.go:119,233-275`, `profiles.go:310`; `rdd_retire.go` y tests RDD de `explore.md` §3; recortes de los mixtos (`inject_test`, `read_assignments_test`, `prompts_test`, `copilot_cli_test`, `opencode_v2_test`, `opencode_runtime_test`, `profiles_test`, `commands_test`, `shared_asset_deployment_test`, `workspace_conventions_test`, `adapter_minimality_guard_test`); `main_test.go` sin `rddharness`.
- **Contratos de capacidad**: `capabilitymanifest/manifest.go` (`ContractReviewTransportV1`, `ContractImmutableReviewExecutorV1`, mapas de exposición, campos de `ContractClaims`, ramas de `Advertises`), `agents/capability_manifest.go:24`, `manifest_test.go`; se borra `agents/adapter_forbidden_construction_guard_test.go`.
- **`cli`**: `run.go:3278` (`IsRetiredReviewSubAgentPath`), `sync.go:2636-2655` (verificación del plugin heredado), `selectedSkillIDs` → `skills.Installable` (D4); tests que leen activos borrados: `install_test.go`, `run_component_paths_test.go`, `sync_test.go` (`:3927-3963`), `workspace_shared_skills_test.go`.
- **Skill y modelos**: `model.SkillRDDDefectWorkflow`, `catalog/skills.go:38` y `skills_test.go`, `components/skills/presets.go:27,45`, `tui/screens/skill_picker.go:38`, `components/skills/retired.go` (fuera `WithoutRetired`), `opencode.{ReviewRefuterAgent,ReviewValidatorAgent,ReviewLensPhases,ReviewPhases}`, `opencode/config.go:389-407` con `{models,config}_test.go`, filas RDD de `tui/screens/{model_picker,claude_model_picker,codex_model_picker}.go` y sus tests, `components/uninstall/workspace_test.go`.
- **Activos**: los 25 `review-*` de `internal/assets/{claude,cursor,kiro,kimi}/agents/`, 6 entradas de `opencode/sdd-overlay-{single,multi}.json`, `opencode/plugins{,-v2}/opencode-review-transport.ts`, `skills/_shared/review-ledger-contract{,-pi}.md`, `internal/assets/skills/rdd-defect-workflow/`, `skills/{rdd-defect-workflow,rdd-advisory-transport}/`, filas de `AGENTS.md:62-63`; prosa (D6); tests de `internal/assets` de `explore.md` §4; guarda de D13.
- **Goldens**: D6 en dos pasos; `go test ./internal/tui -run TestCustomPresetPostComponentFlowMatrix -update` para los dos goldens con «RDD Defect Workflow»; después, todo sin `-update`.
- **Comprueba**: `go test ./internal/components/... ./internal/assets/... ./internal/opencode/... ./internal/catalog ./internal/model ./internal/agents/... ./internal/tui/... ./internal/cli -count=1 -timeout 90m`.

### C10 · Unidad 4b: `hub` y `state`

- **Borra**: `hub/rddpref{,_test}.go`.
- **Edita**: `hub/types.go:24-26`, `hub/main_test.go`, `state/state.go:183-196,341-342`, `state/store.go:136-164`.
- **Crea**: pruebas de tolerancia de D5.
- **Comprueba**: `go test ./internal/state ./internal/hub ./internal/statecoord ./internal/app ./internal/cli -count=1 -timeout 60m`.

### C11 · Unidad 7b: paquetes RDD y su CI

- **Borra**: `internal/{reviewtransaction,reviewerprovider,rddharness}/`, `contracts/review-integration/`, `model.ValidCodexReviewModel` y su test.
- **Edita**: `windows-full-suite.yml` (D8, segunda mitad); `ci.yml` `windows-runtime`: fuera «Run native Windows git process tree tests», «Stress repository-context sharing retry», `GENTLE_AI_REQUIRE_DISTINCT_WINDOWS_TOKEN_OWNER` y los release-blockers de `reviewtransaction` (quedan `./internal/fileprim` con 3 y `./internal/components/engram` con 5); `TestWindowsReleaseBlockerCannotSkipOwnerRebinding` pasa a `TestWindowsReleaseBlockerPackagesExist` (cada paquete de `$packages` existe en `go list ./...`); `darwin-release-blockers.manifest`; `.github/PULL_REQUEST_TEMPLATE.md`; ratchets de D12; comentarios que citan los paquetes en `fileprim/lock.go`, `pathidentity/identity.go`, `kickoff/{integration,types,integration_test}.go`, `state/state.go`, `e2e/organicruntime/organic_runtime_real_agent_detection_test.go:4`; se borra el subtest temporal de paridad de D3.
- **Comprueba**: `go list ./... | rg 'reviewtransaction|reviewerprovider|rddharness'` vacío; `go test ./internal/cli ./internal/assets ./internal/model ./internal/kickoff ./internal/fileprim ./internal/pathidentity ./internal/state ./internal/sddstatus -count=1 -timeout 75m`.

### C12 · Unidad 9: cierre

- Docs de D14, `docs/releases/notas-retirada-rdd.md` y enlace en `README.md`, `.deadcode-baseline.txt`, `go mod tidy`.
- **Comprueba**: `go build ./... && go vet ./... && go test ./... -count=1 -timeout 120m`; `go mod tidy` sin diff; búsqueda de ausencia con las exclusiones de D14; `rg -n 'bench-model-picker|bench_fixture|rdd_harness' --glob '!docs/plan/**' --glob '!openspec/changes/**' --glob '!odd/**'` vacío.

## Flujo de datos

```
OpenRuntimeStore(repo, change)
  ├─ gitcommon.RepositoryRoot(repo) ─────────────→ root (canónico)
  └─ gitcommon.CommonDir(root) → EvalSymlinks ──→ commonDir/gentle-ai/sdd-runtime/v1/<change>

state.json skills / --skills ─→ selectedSkillIDs ─→ skills.Installable
                                                     ├─ fuera: retiradas (branch-pr, gentle-ai-bench)
                                                     └─ fuera: sin skills/<id> embebido (rdd-defect-workflow)
                              ─→ inyección ─→ verificación (mismos ids)
```

## Cambios de ficheros (resumen)

| Zona | Acción | Commit |
|---|---|---|
| `sddstatus/runtime_ledger.go`; helpers de D7; centinelas de D5 | Modificar/Crear | C1 |
| `bench/`, `results.json`, `skills/axiom-bench/`, `app/bench_model_picker*.go` | Borrar | C2 |
| `scripts/` RDD, paso de `opencode-v2-contracts` | Borrar/Modificar | C3 |
| `e2e/organicruntime`, `e2e/Dockerfile.claude-network-none`, jobs de `ci.yml` | Modificar/Borrar | C4 |
| `providercontractbundle*`, `contracts/review-provider-contract`, release, `releasepolicy`, `releaseprovenance*`, tests de `update` | Borrar/Modificar | C5 |
| `internal/tui/**`, `app/app.go` (TUI) | Borrar/Modificar | C6 |
| `internal/dashboard/**` | Borrar/Modificar | C7 |
| `cli/review*` y afines, `cmd/axiom`, `internal/app`, `internal/branch/**`, `internal/kickoff`, tests de `sddstatus` | Borrar/Modificar | C8 |
| `components/sdd` (→ `asset_render.go`), `internal/assets/**`, `skills/rdd-*`, `components/skills`, `catalog`, `model`, `opencode`, pickers, `capabilitymanifest` | Borrar/Crear/Modificar | C9 |
| `hub/{rddpref*,types}.go`, `state/{state,store}.go` | Borrar/Modificar | C10 |
| `internal/{reviewtransaction,reviewerprovider,rddharness}`, `contracts/review-integration`, CI Windows/Darwin, ratchets | Borrar/Modificar | C11 |
| `docs/**`, `README.md`, `AGENTS.md`, `.deadcode-baseline.txt`, `go.mod`/`go.sum` | Borrar/Modificar/Crear | C12 |

## Interfaces / contratos

```go
// internal/components/skills/installable.go
// Installable devuelve ids sin las skills retiradas ni las que no tienen
// activo embebido, en el mismo orden y sin compartir el slice de entrada.
func Installable(ids []model.SkillID) []model.SkillID
```

Desaparecen: `axiom review*` (incluido `review stop-hook`), `axiom bench-model-picker`, `/api/project/review-mode`, el campo `review_mode` del inicio de proyecto (se ignora), `gentle-ai-review-provider-contract-<semver>.tar.gz`, `provider_contract_semver`, `contracts/review-integration/{v1,v2}` y los contratos de capacidad `gentle-ai.review-transport/v1` y `gentle-ai.immutable-review-executor/v1`. No cambian `contracts/sdd-integration`, `axiom sdd-attempt grant`, `internal/consentenvelope`, `kickoff.CommitReview*` ni Judgment Day con su `review-ledger`.

## Estrategia de pruebas

TDD: el modo no está fijado; donde hay comportamiento nuevo (C1, C9, C10) la prueba se escribe antes y se observa en rojo. El resto son borrados verificados con las suites existentes, ejecutadas enteras.

| Capa | Qué | Cómo |
|---|---|---|
| Unitaria | D3 | `sddstatus/runtime_common_dir_test.go`, tabla: checkout principal; worktree enlazado (resuelve al common dir del principal); ruta con enlace (se salta en Windows sin privilegio); temporal 8.3 en Windows; llamada desde un subdirectorio por `OpenRuntimeStore`. Oráculo: `EvalSymlinks(git rev-parse --path-format=absolute --git-common-dir)`. Subtest temporal de paridad con `CompactAuthoritativeStore` hasta C11. |
| Unitaria | D4 | `components/skills`: mixto (orden), solo retiradas, sin activo (`rdd-defect-workflow` e id inventado), sin mutar la entrada. `cli/selected_skill_ids_test.go` añade `rdd-defect-workflow`. |
| Integración | D4 | `TestRunSyncRetiredSkillInPersistedSelectionDoesNotFailVerification` con `rdd-defect-workflow` en `Skills` (global y workspace): `sync` sin error, sin directorio `rdd-defect-workflow`, `go-testing` instalado; equivalente para `RunInstall`. |
| Unitaria | D5 | `state` y `hub`: JSON crudo con `rdd_mode` (y `rdd_mode_recorded_at`) carga, conserva el resto, y tras escribir el fichero ya no los contiene. |
| Requisitos trasladados | D16 | Tests de la tabla de D16 en verde en sus commits; ninguno se modifica salvo el traslado de `TestSDDStatusDualSerialization`. |
| Golden | D6 | `TestGolden*` sin `-update` tras el paso mecánico; con `-update` tras la prosa, revisando el diff; de nuevo sin `-update`. |
| Guardas CI | D8, D9, C11 | `TestWindowsFullSuiteShardsCoverEveryTestName`, `TestRequiredChecksFailClosedWhenFormatFails`, `TestOrganicRuntimeE2EUsesInstalledOpenCodePin`, `TestWindowsReleaseBlockerPackagesExist`. |
| E2E | D9 | `go test ./e2e/organicruntime -list` y ejecución sin binarios; el job manual `organic-runtime-e2e` con agentes reales. |
| Release | C5 | `./internal/update ./internal/releasepolicy ./internal/releaseprovenance ./internal/releaseprovenancecmd`. |
| Completa | C12 | `go test ./...` y `windows-full-suite` con la etiqueta `ci:windows` en la PR. |

## Matriz de amenazas

| Frontera | Casos adversos mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Ficheros con aspecto de documentación | `requirements.txt`, `CMakeLists.txt`, Markdown ejecutable, `README.sh` | N/A: no se cambia ninguna clasificación de ejecutables. | — | — |
| Selección del repositorio git | `git -C`, rutas relativas, rutas absolutas | **Aplicable** (D3). | `root` sale de `gitcommon.RepositoryRoot` (canónico) y git se ejecuta en `root`; el resultado se canoniza con `EvalSymlinks`; fuera de un repositorio se mantiene `RuntimeRepositoryRequiredError`; un worktree enlazado resuelve al common dir del principal. | Los cinco casos de `runtime_common_dir_test.go`. |
| Estado del commit | staged, `commit -a`, índice vacío | N/A: sin automatización de commits nueva. | — | — |
| Estado del push | seguimiento, primer push, refspec | N/A: sin automatización de push nueva. | — | — |
| Órdenes de PR | `--head`, prefijo de entorno, compuestas | N/A: sin automatización de PR nueva. | — | — |

Fuera de la matriz: un hook `Stop` antiguo con `axiom review stop-hook` recibe «unknown command» por `app.RunArgs` y `cmd/axiom` sale con 1; Claude Code trata una salida distinta de 2 como error no bloqueante. Es la consecuencia aceptada de P20.

## Migración / despliegue

### Texto para `docs/releases/notas-retirada-rdd.md`, la PR y las notas de la release

> **Retirada de `axiom review` (RDD) y de `bench/`.** Esta versión no limpia lo que instalaron versiones anteriores. Antes de instalarla:
>
> 1. **Hooks de Claude Code** en `~/.claude/settings.json` y, si existen, `.claude/settings.json` y `.claude/settings.local.json` de cada proyecto: en `hooks.Stop` y `hooks.SessionStart`, borra solo los hooks cuyo `command` sea exactamente `axiom review stop-hook --agent claude-code` o `gentle-ai review stop-hook --agent claude-code`, y la entrada que quede vacía. Repite el patrón `<binario> review stop-hook --agent <id>` en cualquier otro agente. Si no, cada fin de turno mostrará un error no bloqueante («unknown command»).
> 2. **Agentes de revisión**: borra `review-readability`, `review-refuter`, `review-reliability`, `review-resilience` y `review-risk` (`.md`, y `.yaml` en Kimi) de los directorios `agents` de Claude (`~/.claude/agents`), Cursor, Kiro y Kimi. En OpenCode, quita de `opencode.json` (o del overlay instalado) `review-risk`, `review-readability`, `review-reliability`, `review-resilience`, `review-refuter` y `review-validator`, con sus permisos de `task`. `axiom uninstall` ya no los borra.
> 3. **Plugins de OpenCode**: borra `opencode-review-transport.ts` y el heredado `review-result-artifacts.ts` del directorio de plugins (`~/.config/opencode/plugins` o el del proyecto). `axiom sync` ya no los comprueba ni los migra.
> 4. **Skill y ficheros compartidos**: borra el directorio `rdd-defect-workflow` y los ficheros `_shared/review-ledger-contract.md` y `_shared/review-ledger-contract-pi.md` de los directorios de skills de cada agente. Una selección guardada que nombre la skill ya no hace fallar `sync`: se ignora.
> 5. **Prosa de orquestador**: tras el primer `axiom sync` con esta versión, comprueba que los ficheros de instrucciones de tus agentes (por ejemplo `~/.claude/CLAUDE.md` o `AGENTS.md`) no conservan la sección `#### Review Execution Contract` ni otra prosa de `axiom review`. `sync` reescribe la parte que gestiona; si la copiaste fuera de ella o la editaste a mano, bórrala tú.
> 6. **Opcional, inofensivo**: `rdd_mode` y `rdd_mode_recorded_at` del estado global y `rdd_mode` del hub se ignoran y desaparecen en la siguiente escritura; el almacén `<git common dir>/gentle-ai/review-transactions` y las asignaciones de modelo de los roles `rdd-*` de Codex pueden borrarse a mano.
>
> **Capacidades publicadas retiradas**: el asset `gentle-ai-review-provider-contract-<semver>.tar.gz`, los esquemas `contracts/review-integration/{v1,v2}` y el campo `provider_contract_semver` de `gentle-ai.release-provenance/v1`. Los verbos `axiom review*` y `axiom bench-model-picker` ya no existen.

### Acciones humanas fuera del repositorio

- **Protección de rama**: si «Claude Network-None Runtime (Required)» es comprobación obligatoria de `main`, hay que quitarla **antes** de fusionar; si no, la PR esperará una comprobación que ya no se ejecuta.
- **Candidatas de release**: una RC cortada antes del merge lleva el asset del contrato y no pasará la comparación de `promote-stable-rc.yml` nuevo; hace falta una RC nueva.

Sin migración de datos. Rollback: el de la propuesta (`git revert` por unidad antes del merge, `git revert -m 1` después), con un matiz de D1: un commit solo compila revertido si se revierten también los posteriores que borran lo que consume. Revertir C5 sola deja `providercontractbundle` sin `ReviewExecutionContractFor` (borrado en C9): un fallo de release se corrige hacia delante, o se revierten C5 y C9-C12.

## Preguntas abiertas

- [ ] **Redacción de la spec (no bloquea)**: REQ-46.5 dice «ya no exista en el catálogo»; D4 usa «sin activo embebido», que coincide con el catálogo para toda skill distribuida sin excluir ids embebidos fuera de `catalog`. Conviene alinear la redacción.
- [ ] **`deadcode` en local (no bloquea)**: `scripts/deadcode-ratchet.sh` instala la herramienta con `go install`. Si el EDR la bloquea, C12 usa la salida del paso de CI «Check for new unreachable functions», sin editar a mano.
