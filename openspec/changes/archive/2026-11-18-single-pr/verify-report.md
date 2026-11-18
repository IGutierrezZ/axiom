# Informe de verificación: single-pr (C-02-c)

Estado: **PASS con observaciones** (0 CRITICAL, 0 WARNING, 3 SUGGESTION). Base `c18dd786`, HEAD `bf991897`. 17/17 grupos de tareas con todas las casillas marcadas (0 sin marcar). Verificación sin modificar código ni hacer commits.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go test -count=1 ./internal/workspace/ ./internal/hub/ ./internal/components/agentguidance/ ./internal/assets/ ./internal/skillregistry/ ./internal/sddtaskresult/...` | ok en los seis paquetes (exit 0) |
| `go test -count=1 ./internal/components/sdd/` | ok, 112 s |
| `go test -count=1 ./internal/components/ -run TestGolden` | ok |
| `go test -count=1 ./internal/cli/ -run 'SDDPreflight' -timeout 900s` | ok (110 líneas RUN/PASS en `-v`) |
| `go vet ./...` | exit 0, sin salida |
| `go vet ./...` en `bench/` (módulo aparte) | exit 0 |

No ejecutados (por indicación): `go test ./internal/cli/` completo, `bench/` tests, fallos ambientales conocidos. Sin procesos `cli.test.exe` huérfanos.

## Matriz requisito -> evidencia (resumen)

| Requisito | Implementación | Cobertura |
|---|---|---|
| workspace-topology: bloque `delivery:` con `split_hint_lines`, desactivado por defecto, rechaza 0/negativos/comillas/nulo/clave desconocida, solo avisa | `internal/workspace/delivery.go` (decodeDelivery, deliveryPositiveInt), `declarations.go`, `loader.go`, `types.go`, `hub/init.go` (preserva con --force), `axiom.example.yaml`, `docs/manual-de-inicio.md` | `delivery_test.go`, `example_yaml_test.go`, `docs_contract_test.go`, `hub/init_baseline_test.go` |
| REQ-28.14/28.15/28.18/28.20 (pre-vuelo: solo `pace` en monorepo, nada en multirepo, bloque de cinco líneas, sin estrategia de PR) | `internal/cli/sdd_preflight_hook.go` (`pr-strategy` sin `RequiredIn`, campo `Delivery strategy` fijo `single-pr`, sin `Review policy`), `components/sdd/session_preflight.go` | `sdd_preflight_hook_test.go`, `sdd_preflight_hook_topology_test.go`, `session_preflight_test.go`, `inject_test.go` |
| REQ-28.16/28.17 (marcadores y grupos heredados tolerados e ignorados; `Ask me`/`Auto` no producen entrega distinta de `single-pr`) | `sdd_preflight_hook.go` (grupo `pr-strategy` conservado solo como heredado) | `sdd_preflight_hook_test.go`, `..._topology_test.go:95-110,192,208` |
| REQ-28.21 (plugin OpenCode v1/v2 con el mismo contrato) | `internal/assets/opencode/plugins/` y `plugins-v2/sdd-task-result-artifacts.ts` (`requiredIn: []`, `groupId: ""`, sin línea de revisión), `opencode_plugin_digests.go` | `sddtaskresult/preflight_question_harness_test.go`, `opencode_sdd_preflight_topology_test.go`, `opencode_runtime_test.go` |
| REQ-32.3 (orquestadores: `single-pr` por defecto sin preguntar, `chained` solo a petición con una pregunta por estrategia de cadena, alias heredados, sin Review Workload Guard, sdd-apply sin parada por carga, sdd-tasks con previsión cualitativa, aviso único con `split_hint_lines`) | `assets/skills/_shared/sdd-orchestrator-sections.md:262-266`, `sdd-phase-common.md:114`, `sdd-apply/SKILL.md`, `sdd-tasks/SKILL.md:149`, 12 plantillas de orquestador, 9 comandos opencode | `delivery_strategy_vocabulary_test.go`, `orchestrator_shared_sections_test.go`, `renderer_invariants_test.go`, `review_ledger_contract_test.go`, goldens |
| REQ-32.4 (doctrina ODD: `single-pr` por defecto, cadena a petición, heurística por tarea sin número, aviso opcional no bloqueante) | `components/agentguidance/routing.go:129-130` | `agentguidance/routing_test.go:909-945` (incluye ausencia de `400`, `ask-on-risk`, `auto-chain`, `exception-ok`) |
| chained-pr / work-unit-commits sin umbral numérico | `assets/skills/chained-pr/**`, `work-unit-commits/SKILL.md`, `skills/*` copias | `assets/delivery_threshold_residue_test.go` (escaneo de assets y skills) |

## Criterio de hecho ampliado (residuos de `400`)

`rg -n "\b400\b" internal/assets internal/components internal/cli skills` solo muestra residuo permitido:
- Umbral de barrido `Full-4R review` en jd-judge-a/b y lentes `review-*` (claude, copilot, cursor, kiro, kimi) y en overlays `opencode/sdd-overlay-*.json`.
- `rdd-defect-workflow/SKILL.md` (ambas copias).
- Literales de migración `D1 400 lines` / `D1 400 líneas` en `internal/components/sdd/inject.go:1543,1560`.
- `internal/cli/review_process_boundary_test.go:16` (comentario).
- Aserciones negativas y rellenos en tests (`assets_test.go:1058`, `routing_test.go:942`, `windsurf_workflow_test.go:13`, `sdd_exits_wording_test.go:30`, `session_preflight_test.go`, `sdd_preflight_hook_topology_test.go:109`, `delivery_strategy_vocabulary_test.go:216`, `inject_test.go`, `review_*_test.go` con `Repeat(...,400)` y `lines: 400`, `delivery_threshold_residue_test.go`).
- `internal/reviewtransaction/risk.go:22` queda fuera del ámbito del `rg` pero está permitido.
No hay ninguna aparición viva fuera de esa lista: 0 CRITICAL por residuo. El guard `TestDeliveryThresholdResidue` lo fija por ruta y contenido.

## Comprobaciones puntuales

3. Pre-vuelo sin pregunta de estrategia de PR: verificado (monorepo solo `pace`, multirepo nada; bloque de cinco líneas con `Delivery strategy: single-pr`; el bloque con `ask-on-risk`/`chained` se rechaza, `..._topology_test.go:110-111`). `chained` solo a petición en el chat (`sdd-orchestrator-sections.md:262`, `routing.go:130`). `split_hint_lines` desactivada por defecto (`DeliveryConfig` nil sin bloque), rechaza 0 y solo avisa una vez sin bloquear.
4. Sin anticipar C-02-d (no hay avisos por commit ni modos P2 en el diff) y sin tocar C-02-e: `.github/` no figura en el diff y `.github/workflows/pr-check.yml:25` conserva `limit = 400`.
5. Atribución: ningún commit de `c18dd786..HEAD` contiene `Co-Authored-By`, `Claude`, `Anthropic` ni «Generated with».

## Hallazgos

**CRITICAL:** ninguno.

**WARNING:** ninguno.

**SUGGESTION:**
1. `internal/cli/sdd_preflight_hook.go:432` y las tablas TS conservan los tokens `ask-on-risk`/`auto-chain` como vocabulario heredado tolerado del grupo `pr-strategy`; es intencional (REQ-28.16), pero conviene un comentario que lo diga junto a la tabla Go para evitar que alguien lo borre o lo reactive.
2. `skills/branch-pr/SKILL.md` y `issue-creation` siguen citando `size:exception`; es coherente mientras C-02-e no retire el check de CI, y debe revisarse en ese cambio.
3. Cobertura no ejecutada: `bench/` tests y `go test ./internal/cli/` completo no se corrieron (por indicación); la ruta del preflight sí está cubierta por `-run SDDPreflight` y la compilación/vet de `bench/` es limpia.

## Tareas

Todas las tareas de `tasks.md` constan completadas ([x]); no se reescribió ninguna. Siguiente paso recomendado: `sdd-archive`.
