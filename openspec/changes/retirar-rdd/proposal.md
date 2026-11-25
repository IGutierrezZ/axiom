# Propuesta: Retirada completa de RDD (C-01-e)

## Intención

RDD (*receipt-driven development*, la revisión con recibos de `axiom review`) está apagado desde C-01-c: `internal/rddharness` solo lo activa con la etiqueta de compilación `rdd_harness` (bench, e2e y CI) o con `SetForTesting`. En el binario publicado todo el código RDD es inalcanzable, salvo las lápidas que retiran lo que dejaron versiones anteriores. Aun así, el repositorio sigue cargando con él:

- unas 61.300 líneas de Go de producción y unas 100.000 de pruebas (`internal/reviewtransaction`, `internal/cli/review*`, `internal/reviewerprovider`, `internal/providercontractbundle*`, `internal/rddharness` y la parte RDD de paquetes ajenos);
- 11.175 líneas de contratos publicados (`contracts/review-integration`, `contracts/review-provider-contract`) y un artefacto de release (`gentle-ai-review-provider-contract-<semver>.tar.gz`);
- 21 specs vivas, ~3.200 líneas de docs, 25 agentes `review-*`, un plugin de OpenCode, una skill y prosa residual en 12 orquestadores;
- el módulo `bench/` (24.148 líneas de Go), que mide la fricción del ciclo de revisión y es el único consumidor del binario con `rdd_harness`;
- CI que compila, estresa (`-count=100`) y reparte en shards de Windows todo lo anterior.

El usuario lo ha decidido («no quiero telemetría, ni GGA, ni RDD», sección 08 del plan) y C-08-c ya extrajo las primitivas compartidas (`internal/fileprim`, `internal/gitcommon`), de modo que `reviewtransaction` solo depende de código RDD.

**Por qué ahora.** C-01-e se adelantó a la ola A2 (sección 08, «C-01-e adelantada»). Bloquea a C-08-d, C-08-g, C-08-h y C-08-i: todos tocarían o paralelizarían ficheros que este cambio borra.

**Resultado esperado.** El producto, sus contratos publicados, la release, la CI, los assets, las specs vivas y la documentación viva dejan de contener RDD y `bench/`, sin lápidas. Toda la cobertura no RDD se conserva y la suite completa queda en verde.

## Decisiones vinculantes

- **P20 (usuario): «Sí, borrarlas ya».** No quedan lápidas RDD. Se borran `internal/components/sdd/rdd_retire.go`, la limpieza de la rama `!rddharness.Enabled()` de `inject.go` y el subcomando `axiom review stop-hook`, sin no-op. Las instalaciones existentes se limpian a mano (sección *Migración manual*).
- **P21 (usuario): «Sí, retirarlo».** `bench/` se retira entero, con sus pasos de CI y lo que lo referencia (`skills/axiom-bench`, `axiom bench-model-picker`, la comprobación del ratchet que lee `bench/classify.go`).
- **Entrega `single-pr`:** una PR con un commit por unidad de trabajo; sin PR encadenadas.
- **Orquestación, coherentes con P20:**
  1. La migración del plugin heredado `review-result-artifacts.ts` (`LegacyOpenCodeReviewPluginName`, `removeLegacyOpenCodeReviewPlugin`) es lápida RDD: se borra.
  2. La skill `rdd-defect-workflow` se borra sin entrada en `components/skills/retired.go`. Una selección persistida que la nombre no debe romper la carga ni `sync`; `design` decide el mecanismo (ignorar ids desconocidos si ya ocurre, o filtrarla al leer).
  3. El requisito de Judgment Day que hoy vive en `review-findings-ledger` («Explicit Judgment Day replacement») se traslada a `sdd-independent-review` antes de retirar esa spec, sin su acoplamiento a estado nativo y recibos de RDD. Los artefactos `review-ledger` de Judgment Day no son RDD y se quedan.
  4. `axiom sdd-attempt grant` (consentimiento de edición SDD, `axiom.sdd-integration.consent/v1`) e `internal/consentenvelope` se quedan. Solo desaparece su uso desde `cli/review_consent_contract.go`.

### Desviación del plan

El plan (`docs/plan/backlog-2026-11/01-poda-friccion.md`, C-01-e, «Decisiones») conservaba como lápidas `rdd_retire.go` y `axiom review stop-hook` como no-op, y su criterio de terminado excluía «las lápidas de P20» de la búsqueda con `rg`. P20 se resolvió después como «borrarlas ya»: este cambio no deja lápidas y el criterio de terminado no admite esa excepción.

## Alcance

### Dentro del alcance

1. **Paquetes RDD completos:** `internal/reviewtransaction`, `internal/reviewerprovider`, `internal/providercontractbundle`, `internal/providercontractbundlecmd`, `internal/rddharness` (y sus `SetForTesting` en los 7 `main_test.go`), `internal/cli/review*.go` con sus tests y los ficheros `cli` RDD que no se llaman `review_*` (`branch_lineage*`, `codex_review_provider_model_test`, `sync_review_agents_verify_test`, `sdd_status_review_disabled_test`, `sdd_status_reenable_fresh_review_test`, `dual_contract_integration_test`, `protocol_probe_test`, `continuation_paths_test`, previa comprobación de que no cubren nada ajeno).
2. **Despachadores:** verbos `review*` de `cmd/axiom/main.go` y `internal/app/app.go`, incluido `review stop-hook`; `bench-model-picker` e `internal/app/bench_model_picker{,_fixture}.go`.
3. **Parte RDD de paquetes ajenos:**
   - `components/sdd`: `boundedreview.go`, `rdd_retire.go`, el cableado de `inject.go`, `orchestrator.go` y `profiles.go` (hook Stop/SessionStart, migración del plugin heredado, recortes `suspendedSharedSkillContent`) y `cli/run.go` (`IsRetiredReviewSubAgentPath`).
   - TUI: entrada del menú de bienvenida, `screens/{review_store_reset,review_mode,install_review_mode}.go`, `tui/model.go`, `tui/router.go`, `app.go` (`tuiReviewStoreSurvey/Reset`), filas de modelos RDD de los selectores de Claude, Codex y OpenCode. `ScreenReview` («Review and Confirm» del instalador) no es RDD y se queda.
   - Dashboard (`/api/project/review-mode`, `service_review_mode.go`, `reviewmode.js`), hub (`rddpref.go`, `types.go:RDDMode`) y `internal/state` (`RDDMode`, `RDDModeRecordedAt`), con lectura tolerante: un `rdd_mode` en disco se ignora.
   - `sddstatus/runtime_ledger.go`: sustituye `reviewtransaction.CompactAuthoritativeStore` por `gitcommon.CommonDir`, con prueba de equivalencia de ruta.
   - `branch`: `LineageProbe`, `DefaultWithLineages` y el aviso de linajes RDD abiertos de `branch done`.
   - `opencode` (`ReviewRefuterAgent`, `ReviewValidatorAgent`, `ReviewLensPhases`, `ReviewPhases`), `agents/capabilitymanifest` (`ContractReviewTransportV1`, `ContractImmutableReviewExecutorV1`, su exposición en los adaptadores y la guarda `adapter_forbidden_construction_guard_test`), `model.ValidCodexReviewModel`, `model.SkillRDDDefectWorkflow`, catálogo, presets y `skill_picker`.
   - Guardas de frontera RDD: `kickoff/rdd_boundary_test.go`, `sddstatus/review_*` de RDD.
4. **Assets:** los 25 agentes `review-*` (Claude, Cursor, Kiro, Kimi), sus 6 entradas en `opencode/sdd-overlay-{single,multi}.json`, los plugins `opencode-review-transport.ts` (v1 y v2), `skills/_shared/review-ledger-contract{,-pi}.md`, la skill `rdd-defect-workflow` (embebida y del repositorio), `skills/rdd-advisory-transport`, la prosa `Review Execution Contract` y su marcador en los 12 orquestadores y en `sdd-orchestrator-sections.md`, y las menciones de `judgment-day`, `sdd-verify` y `openspec-convention.md`.
5. **Contratos y release:** `contracts/review-integration/{v1,v2}`, `contracts/review-provider-contract`, el hook y el archivo de `.goreleaser.yaml`, `release.yml`, `promote-stable-rc.yml`, `scripts/{promote-stable-preflight,verify-release-assets}.sh`, `scripts/darwin-release-blockers.*`, `internal/releasepolicy/policy.go` y los tests de `internal/update`. `contracts/sdd-integration` no es RDD y se queda.
6. **CI y scripts:** pasos de bench y `rdd_harness`/`bench_fixture` de `ci.yml`, el estrés `-count=100` de `reviewtransaction`, los shards `cli-review-*` y `reviewtransaction-*` de `windows-full-suite.yml` (reequilibrando el resto), `scripts/{test-review-capabilities,test-review-contract-package,cross-lane-battery}.sh`, `scripts/crosslane/`, `opencode_v2_loopback.py`, `test-opencode-v2-host.py`, `test_opencode_v2_host_test.py`; `test-opencode-v2-contracts.sh` y el job `opencode-v2-contracts` se recortan y se quedan para los plugins no RDD.
7. **`bench/`** entero, el `results.json` de la raíz, `skills/axiom-bench`, la plantilla de PR y la comprobación de `refusal_resolution_ratchet_test` contra `bench/classify.go`.
8. **`e2e/organicruntime`:** cirugía de los tests RDD y sus helpers huérfanos; se conservan los no RDD (enrutado, SDD opcional, ejecutor `sdd-apply` instalado, `opencode_trace_test.go`) y todo `sdd_consent_wiring_e2e_test.go`.
9. **Specs vivas:** retirada de las 21 RDD y deltas de las que pierden menciones RDD (ver *Capacidades*).
10. **Docs y baselines:** borrado de la documentación RDD viva (`docs/review-integration.md`, `docs/review-authority-threat-model.md`, `docs/architecture/{organic-rdd,rdd-*,the-organic-rdd-story}.md`, `docs/audits/*rdd*` y sus datos, `docs/testing/organic-rdd-testing-guide.md`, `docs/assets/diagrams/rdd-review.svg`); poda de `README.md`, `AGENTS.md`, `docs/{usage,trigger-rules,intended-usage,components,manual-de-inicio,upstream-absorption-ledger}.md`, `docs/testing/windows-tests.md`, `docs/architecture/guard-population.md` y `docs/assets/diagrams/sdd-cycle.svg`; regeneración de `.deadcode-baseline.txt`, `.refusal-ratchet-baseline.txt`, `.guard-population-baseline.txt` y `guardPopulationCount`; `go mod tidy`.
11. **Sección de migración manual** en esta propuesta y en las notas de la versión.

### Fuera del alcance

- Lápidas de telemetría (`telemetryruntime`, `telemetry_retire.go`): C-08-e.
- Limpieza de `uninstall` de agentes retirados: C-08-d. Consecuencia aceptada: al borrar los 25 agentes `review-*` embebidos, `uninstall` deja de borrar los ya instalados (se documenta en la migración manual).
- `internal/handoff`, `internal/recoverytrace` (C-08-a), `internal/releasepolicy` e `internal/releaseprovenance` como paquetes (solo pierden la parte del contrato del proveedor).
- `axiom sdd-attempt grant`, `internal/consentenvelope`, `kickoff.CommitReview*` y `sdd_gate_commit_review*`: son de SDD.
- Judgment Day y sus artefactos `review-ledger` (`sdd/{change}/review-ledger`).
- Paralelizar tests (C-08-g) y recortar tests de prosa no RDD (C-08-h).
- Históricos: `docs/plan/**`, `openspec/changes/archive/**` y `odd/tasks/**` no se reescriben. `design` decide si `docs/releases/v2.2.0-closure-ledger.md` y `docs/archive/ROADMAP.md` se tratan como históricos o se podan.
- Una lápida o no-op para instalaciones antiguas (descartada por P20).

## Capacidades

### Capacidades nuevas

- Ninguna.

### Capacidades retiradas

`sdd-spec` escribe para cada una un delta con todos sus requisitos `REMOVED`, con el motivo «retirada de RDD (C-01-e, P20)»; al archivar, la spec viva desaparece (el mecanismo exacto lo fija `sdd-spec` según `openspec-convention.md`).

- `rdd-authority-disposition-plan`, `rdd-authority-graph-classification`, `rdd-authority-store`, `rdd-backlog-disposition`, `rdd-candidate-identity`, `rdd-candidate-relation-algebra`, `rdd-closure-disposition-execution`, `rdd-decoupling-v3-stability`, `rdd-delivery-exception-removal`, `rdd-freeze-expansion-policy`, `rdd-leaf-disposition-execution`, `rdd-new-lineage-activation`, `rdd-ownership-inventory`, `rdd-post-verify-review-offer`, `rdd-receipt-only-gates`, `rdd-review-core-transitions`, `rdd-sdd-receipt-consumption`, `rdd-shadow-evaluation`, `rdd-simplification-design`, `rdd-transport-capability`.
- `review-findings-ledger` (después de trasladar su requisito de Judgment Day).

### Capacidades modificadas

- `sdd-independent-review`:
  - **ADDED** un requisito de Judgment Day trasladado desde `review-findings-ledger` («Explicit Judgment Day replacement»): selección explícita, dos jueces ciegos distintos de solo lectura, como máximo dos rondas de corrección y dos rejuicios acotados, y escalado sin tercera ronda. Se elimina el acoplamiento a RDD (sustitución de la revisión 4R, `review-refuter`, estado nativo y recibos).
  - **MODIFIED** los requisitos que citan RDD (12 menciones), para quitar esas menciones sin cambiar el comportamiento SDD.
- `change-branch-lifecycle`: **MODIFIED** los requisitos que citan linajes RDD o su aviso en `branch done` (el aviso desaparece).
- `sdd-orchestrator-assets`: **MODIFIED** los requisitos que citan RDD o la skill `rdd-defect-workflow`.
- `organic-agent-trigger-rules`: **MODIFIED** las menciones RDD y de `rdd-defect-workflow`.
- `axiom-distribution-identity`: **MODIFIED** los requisitos que citan el contrato del proveedor de revisión o su artefacto de release.
- `sdd-block-review-gates`, `sdd-preflight-configuration`, `tui-spanish-localization`, `upstream-absorption-protocol` (cita `bench`): **MODIFIED** si la mención está en el texto normativo de un requisito o escenario; si solo está en prosa no normativa, `sdd-spec` lo declara y no escribe delta.

`sdd-spec` confirma cada mención contra la spec viva antes de escribir el delta.

## Enfoque

Enfoque **B** de la exploración: borrado de fuera hacia dentro, en unidades que compilan, una PR con un commit por unidad. Cada unidad deja `go build ./... && go vet ./...` en verde, pasa `go test` de los paquetes tocados y regenera los baselines que toque con su mecanismo (`GENTLE_AI_REFUSAL_RATCHET_UPDATE=1`, `scripts/deadcode-ratchet.sh`), nunca a mano. No se compilan binarios sueltos (el EDR los borra); las pruebas que compilan `cmd/axiom` en proceso ya existen.

| # | Unidad | Contenido principal |
|---|---|---|
| 0 | Desacoplar lo no RDD | `sddstatus/runtime_ledger.go` a `gitcommon.CommonDir` con prueba de equivalencia; mover `runReviewCLIGit` a un helper de test común; sustituir el centinela `RDDMode` de los tests de preservación de estado por otro campo de máquina sin borrar esos tests. No borra nada. |
| 1 | `bench/` (P21) | Módulo, `results.json` raíz, `skills/axiom-bench`, `bench-model-picker`, pasos de `ci.yml`, plantilla de PR, fila de `AGENTS.md`, comprobación del ratchet contra `bench/classify.go`. |
| 2 | Scripts y CI RDD puros | Scripts y `crosslane/`, recorte de `opencode-v2-contracts`, estrés `-count=100`, shards `cli-review-*`/`reviewtransaction-*` con reequilibrio de los demás para cubrir todo `internal/cli`. |
| 3 | TUI | Entrada del menú, pantallas, `model.go`, `router.go`, `app.go`, goldens de `tui/testdata`. |
| 4 | Dashboard, hub y `state` | Endpoint y servicio de review-mode, `reviewmode.js`, `hub/rddpref*`, campos `RDDMode*`, prueba de lectura tolerante. |
| 5 | `components/sdd` y assets | `rddharness.Enabled()` y lápidas (`rdd_retire.go`, hook Stop/SessionStart, migración del plugin heredado) junto con `cli/run.go:IsRetiredReviewSubAgentPath` en el mismo commit; `boundedreview.go`; agentes, overlays, plugins, `review-ledger-contract*`, skill `rdd-defect-workflow` (con el tratamiento de selecciones persistidas), prosa de orquestadores, `judgment-day`, `sdd-verify`; `opencode.Review*`; manifiesto de capacidades; selectores de modelos. |
| 6 | `cli/review*` y despachadores | Ficheros `cli` RDD, `review stop-hook`, `branch_lineage*` con `LineageProbe`/`DefaultWithLineages`/aviso de `done.go`, `main.go`, `app.go`, guardas `kickoff`/`sddstatus`. `reviewtransaction` pierde su último importador. |
| 7 | Paquetes y release | `providercontractbundle*`, `reviewtransaction`, `reviewerprovider`, `rddharness`, `model.ValidCodexReviewModel`; en el mismo commit `contracts/review-*`, `.goreleaser.yaml`, `release.yml`, `promote-stable-rc.yml`, scripts de release, `releasepolicy/policy.go` y tests de `update`. |
| 8 | `e2e/organicruntime` | Cirugía de tests RDD y helpers huérfanos; se conservan los no RDD. |
| 9 | Specs, docs y guardas | Deltas de specs (con el traslado de Judgment Day aplicado antes del `REMOVED` de `review-findings-ledger`), docs, podas, baselines finales, `go mod tidy`, búsqueda de ausencia. |

Antes de borrar cada fichero de test mixto o dudoso (lista de la exploración, §1 y §5), la unidad comprueba si cubre comportamiento no RDD; si lo hace, la cobertura se mueve, no se pierde. Los helpers definidos en ficheros `review_*_test.go` y usados por otros se descubren con `go vet` del paquete en cada unidad.

Referencias de inventario: `openspec/changes/retirar-rdd/explore.md` (rutas, líneas e importadores verificados en `3c520778`).

## Zonas afectadas

| Zona | Impacto | Descripción |
|---|---|---|
| `internal/reviewtransaction`, `internal/reviewerprovider`, `internal/providercontractbundle`, `internal/providercontractbundlecmd`, `internal/rddharness` | Eliminado | Paquetes RDD completos |
| `internal/cli/review*.go`, `internal/cli/branch_lineage*.go` y tests RDD de `cli` | Eliminado | Verbos `axiom review*`, incluido `stop-hook` |
| `internal/cli/{run,branch_done,sdd_planning_home_test,install_test,sync_test,...}.go` | Modificado | Quitar referencias RDD; tests mixtos recortados |
| `cmd/axiom/main.go`, `internal/app/app.go`, `internal/app/bench_model_picker*.go` | Modificado/Eliminado | Despachadores sin `review*` ni `bench-model-picker` |
| `internal/components/sdd/{boundedreview,rdd_retire}.go`, `inject.go`, `orchestrator.go`, `profiles.go` | Eliminado/Modificado | Sin lápidas ni cableado RDD |
| `internal/tui/`, `internal/tui/screens/` | Modificado/Eliminado | Menú, pantallas RDD, selectores de modelos, goldens |
| `internal/dashboard/`, `internal/hub/`, `internal/state/` | Modificado/Eliminado | Review-mode, `rddpref`, campos `RDDMode*` |
| `internal/sddstatus/`, `internal/kickoff/`, `internal/branch/` | Modificado | `CommonDir`, guardas RDD, `LineageProbe` |
| `internal/opencode/`, `internal/agents/capabilitymanifest/`, `internal/agents/*/adapter.go`, `internal/model/`, `internal/catalog/`, `internal/components/skills/` | Modificado | Fases, contratos de capacidad, skill y modelo RDD |
| `internal/assets/**` | Modificado/Eliminado | Agentes, overlays, plugins, skills y prosa RDD |
| `internal/releasepolicy/`, `internal/update/` (tests) | Modificado | Release sin el contrato del proveedor |
| `contracts/review-integration/`, `contracts/review-provider-contract/` | Eliminado | Capacidades publicadas retiradas |
| `.goreleaser.yaml`, `.github/workflows/{ci,windows-full-suite,release,promote-stable-rc}.yml`, `.github/PULL_REQUEST_TEMPLATE.md` | Modificado | CI y release sin RDD ni bench |
| `scripts/` | Modificado/Eliminado | Scripts RDD y de release |
| `bench/`, `results.json`, `skills/axiom-bench/`, `skills/rdd-*` | Eliminado | P21 y skills RDD |
| `e2e/organicruntime/` | Modificado | Solo tests no RDD |
| `openspec/specs/` | Eliminado/Modificado | 21 retiradas y deltas de las que citan RDD |
| `docs/`, `README.md`, `AGENTS.md` | Eliminado/Modificado | Docs RDD vivas |
| `.deadcode-baseline.txt`, `.refusal-ratchet-baseline.txt`, `.guard-population-baseline.txt`, `go.mod`/`go.sum` | Modificado | Regenerados |

## Migración manual (instalaciones existentes)

Esta versión no limpia lo que instalaron versiones anteriores (P20). Hay que hacerlo **antes de instalar la versión que retira `axiom review`**; si no, Claude Code ejecutará en cada fin de turno una orden que ya no existe y mostrará un error.

1. **Hooks de Claude Code** en `~/.claude/settings.json` (y en `.claude/settings.json` o `.claude/settings.local.json` del proyecto, si existen): en `hooks.Stop` y `hooks.SessionStart`, borrar solo los hooks cuyo `command` sea exactamente una de estas órdenes, y la entrada que quede sin hooks:
   - `axiom review stop-hook --agent claude-code`
   - `gentle-ai review stop-hook --agent claude-code`

   Repetir con el mismo patrón (`<binario> review stop-hook --agent <id>`) en cualquier otro agente con hooks donde se haya registrado.
2. **Agentes de revisión instalados:** borrar `review-readability`, `review-refuter`, `review-reliability`, `review-resilience` y `review-risk` (`.md`, y `.yaml` en Kimi) de los directorios `agents` de Claude (`~/.claude/agents`), Cursor, Kiro y Kimi. En OpenCode, quitar de `opencode.json` (o del overlay instalado) las definiciones `review-risk`, `review-readability`, `review-reliability`, `review-resilience`, `review-refuter` y `review-validator`, y sus permisos de `task`. `axiom uninstall` ya no los borra.
3. **Plugins de OpenCode:** borrar `opencode-review-transport.ts` y el heredado `review-result-artifacts.ts` del directorio de plugins de OpenCode (`~/.config/opencode/plugins` o el del proyecto).
4. **Skill:** borrar el directorio `rdd-defect-workflow` de los directorios de skills de cada agente.
5. **Opcional, inofensivo:** el campo `rdd_mode` del estado global y del hub se ignora y desaparece en la siguiente escritura; el almacén de revisiones en `<git common dir>/gentle-ai/...` y las asignaciones de modelo de roles `rdd-*` de Codex pueden borrarse a mano.

Las notas de la versión repiten esta lista.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Se pierde cobertura no RDD en tests y scripts mixtos (`e2e/organicruntime`, `opencode-v2-contracts`, tests mixtos de `cli`, `components/sdd`, `assets`) | Media | Clasificación test a test antes de borrar; los no RDD de la exploración se conservan; `sdd_consent_wiring_e2e_test.go` intacto; `tsc` de los plugins v2 no RDD se mantiene |
| Se pierde cobertura de ejes de `bench/` no RDD (`model-picker`, `sdd-task-result`, `journeys_sdd*`, `journeys_managed_assets`) | Media | Antes de la unidad 1, comprobar un equivalente en `go test` por eje; si falta, `design` decide si se añade una prueba mínima o se acepta la pérdida de forma expresa |
| Shards de Windows: retirar `cli-review-*` deja sin cubrir parte de `internal/cli` | Media | Reequilibrar shards en la unidad 2; criterio de terminado de cobertura completa de shards |
| Release rota o assets esperados incoherentes (`promote-stable-rc.yml`, `verify-release-assets.sh`, `releasepolicy`) | Media | Pipeline, `releasepolicy` y tests de `update` en el mismo commit que los paquetes (unidad 7); `go test ./internal/update ./internal/releasepolicy` en verde |
| Consumidores externos del artefacto `review-provider-contract` | Baja | Capacidad retirada de forma explícita en la PR y en las notas de la versión |
| Goldens de TUI (menú de bienvenida, «RDD Defect Workflow») | Alta, acotado | Regenerar con `-update` y revisar el diff; los goldens de `components` no deberían cambiar porque el binario ya renderiza sin RDD (si cambian, se revisa la causa) |
| Selección persistida con `rdd-defect-workflow` rompe `sync`/verificación | Media | Requisito explícito y prueba con un `state.json` que la nombra (decisión 2) |
| Hook `Stop` roto en instalaciones antiguas | Alta en quien no migre | Sección de migración manual y notas de la versión (consecuencia aceptada de P20) |
| Ratchets y baselines fallan al cambiar ficheros | Alta | Regenerar con sus mecanismos en cada unidad; nunca editar a mano |
| `sddstatus` cambia de ruta al sustituir `CompactAuthoritativeStore` por `gitcommon.CommonDir` | Baja | Prueba de equivalencia (absoluta, mismos enlaces simbólicos) en la unidad 0 |
| Confundir Judgment Day (`review-ledger`) con RDD (`review-ledger-contract`) | Media | Decisión 3; `review-ledger.md` y `jd-judge-*` fuera del borrado |
| Dependencias huérfanas en `go.mod` | Baja | `go mod tidy` en la unidad 9 |

## Plan de rollback

- Cada unidad es un commit que compila; antes del merge, un fallo se corrige o se revierte la unidad afectada (`git revert <sha>`) sin tocar las demás.
- Tras el merge, `git revert -m 1 <merge>` restaura código, contratos, CI, release, specs y docs. No hay migración de datos: el estado en disco solo pierde campos que se ignoraban y vuelve a leerlos tras el revert.
- No hay nada que restaurar en las instalaciones: RDD ya estaba apagado (C-01-c), así que la versión revertida tampoco instala hooks, agentes ni plugins de revisión. Su `axiom sync` vuelve a retirar con las lápidas los restos que la persona no limpiara a mano.
- Si solo falla la release, puede revertirse la unidad 7 entera (paquetes y pipeline van juntos) dejando las demás.

## Dependencias

- C-01-c (RDD apagado) y C-08-c (primitivas extraídas): en `main`.
- P20 y P21: resueltas por el usuario.
- Revisión commit a commit de C-02-c y C-02-d sin RDD: acordado.

## Estrategia de entrega

`single-pr`: una PR desde `inc/retirar-rdd` contra `main`, con un commit por unidad (0-9). Tamaño XL, casi todo borrado; sin umbral de 400 líneas ni PR encadenadas (decisión del repositorio).

## Criterios de éxito

- [ ] `rg -l 'reviewtransaction|reviewerprovider|rddharness'` no devuelve resultados fuera de los históricos (`docs/plan/**`, `openspec/changes/archive/**`, `openspec/changes/retirar-rdd/**`, `odd/tasks/**` y los que `design` declare históricos). Sin excepción para lápidas.
- [ ] `go build ./...`, `go vet ./...` y `go test ./...` en verde.
- [ ] `axiom review` y `axiom review stop-hook` no existen; `axiom bench-model-picker` tampoco. La ayuda no los cita.
- [ ] La release no publica `gentle-ai-review-provider-contract-<semver>.tar.gz`; `.goreleaser.yaml`, `release.yml`, `promote-stable-rc.yml`, `verify-release-assets.sh` y `releasepolicy` son coherentes, y `update/release_security_test.go` está ajustado.
- [ ] `contracts/review-integration` y `contracts/review-provider-contract` no existen; `contracts/sdd-integration` sigue intacto.
- [ ] Las 21 specs RDD están retiradas y `sdd-independent-review` contiene el requisito de Judgment Day sin acoplamiento a RDD.
- [ ] `bench/` y `results.json` no existen y la CI no los invoca.
- [ ] Los shards de `windows-full-suite.yml` cubren todos los tests que quedan.
- [ ] Un `state.json` con `rdd_mode` y con la skill `rdd-defect-workflow` seleccionada carga y `sync` termina sin error.
- [ ] `axiom sdd-attempt grant`, `kickoff.CommitReview*`, Judgment Day y `sdd_consent_wiring_e2e_test.go` siguen funcionando con sus tests en verde.
- [ ] Baselines (`.deadcode`, `.refusal-ratchet`, `.guard-population`) regenerados y sus tests en verde.
- [ ] La sección de migración manual figura en las notas de la versión.
