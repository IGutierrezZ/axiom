# Informe de verificación consolidado: Ramas y worktrees por topología (inc-25-ramas-y-worktrees-por-topologia)

> **Fase:** `verify` global (REQ-21.13, REQ-21.14, REQ-21.15). Distinto de los informes `verify-report.<rol>.md`: el roster es el rol único `fullstack` y su verificación se consolida aquí.
> **Fecha:** 2026-10-29 · **Idioma:** castellano peninsular (identificadores de código en inglés).
> **Árbol verificado:** worktree `inc/inc-25-ramas-y-worktrees-por-topologia-archive`, sobre `main` en `99ee35a1` (merge de #161) más `4f606065` (compuerta `integration`). Rango de implementación: `17d4029e..99ee35a1` (335 ficheros, +52.979/−984; 145 ficheros `_test.go` tocados).
> **Alcance de esta fase:** diagnóstico sin escritura de código ni de otros artefactos. No se ha hecho commit ni push; no se ha usado red.

## 1. Veredicto global

**PASS con avisos.** 0 CRÍTICOS, 3 AVISOS, 4 SUGERENCIAS.

- `go build`, `go vet` y `gofmtcheck` limpios; todos los paquetes que inc-25 añadió o tocó de forma sustancial pasan con `-count=1`, y un subconjunto dirigido de `internal/cli` (pruebas de `branch`, guardas, raíces y worktrees registrados y ratchet de rechazos) también.
- Los 45 requisitos nuevos (REQ-25.1 a REQ-25.45) y los requisitos modificados tienen al menos una prueba que los referencia y que ha pasado en esta ejecución (§4).
- Los criterios de éxito de `proposal.md` §12 se cumplen en lo observable, con dos matices declarados: el criterio «Invariante», sustituido por D17, y el criterio «Decoupled retirado», cuya parte de «spec viva» se completa al archivar.
- Estado de tareas observado: 213/214 marcadas. S12.26 queda sin marcar a propósito: la ejecuta `sdd-archive` (decisión T10). No es un defecto de apply.

## 2. Resultado de los comandos

Todos desde el worktree `C:\repos\axiom-wt\inc-25-ramas-y-worktrees-por-topologia-archive\axiom`.

| Comando | Resultado observado |
|---|---|
| `go build ./... && go vet ./... && go run ./internal/gofmtcheck` | Salida 0; sin diagnósticos. |
| `go test -count=1` sobre `./cmd/axiom`, `./internal/branch`, `./internal/branch/gitcli`, `./internal/branch/registry`, `./internal/components/agentguidance`, `./internal/components/sdd`, `./internal/components/uninstall`, `./internal/dashboard`, `./internal/gitcommon`, `./internal/gitref`, `./internal/hub`, `./internal/kickoff`, `./internal/sddstatus`, `./internal/skillregistry`, `./internal/tui`, `./internal/tui/screens`, `./internal/workspace`, `./internal/assets/...` | Salida 0; 18 paquetes `ok` (el más lento, `components/sdd`, 113 s). |
| `go test -count=1 ./internal/branch/engine` | `ok` en 248,3 s (salida 0). |
| `go test -count=1 ./internal/cli -run` con el filtro `Branch`, `WrongBranch`, `WriterGuard`, `RegisteredRoots`, `RegisteredWorktrees`, `KickoffBranch`, `Ratchet`, `Refusal`, `GatePlanningHome` | `ok` en 326,3 s (salida 0). Subconjunto dirigido, no el paquete completo. |

Paquetes tocados por inc-25 que no se han ejecutado, por ser fallos de entorno conocidos (fallan también en la base): `internal/app`, `internal/backup`, `internal/filemerge`, `internal/semantic`, `internal/reviewtransaction` y el paquete `internal/cli` completo (supera 60 min). El diff de inc-25 modifica ficheros en `internal/cli` (59), `internal/reviewtransaction` (6) e `internal/semantic` (4).

Evidencia de la suite completa (citada, no ejecutada aquí por no usar red): CI de #161 sobre `ef165c97` (merge ref con `main`): Unit Tests, Go Format, Windows Runtime, Claude Network-None Runtime y OpenCode V2 SDK Contracts en SUCCESS; E2E, Darwin y Windows Full Suite SKIPPED por la propia CI. Cada rebanada S1 a S12 pasó además su batería de la regla 3 de `tasks.md` y su CI (según el relevo y el ledger).

Comprobaciones estáticas adicionales (lectura del repositorio):

- El diff `17d4029e..99ee35a1` no toca `go.mod`, `go.sum`, `Makefile`, `Dockerfile`, `.goreleaser.yaml` ni `.github/workflows/*`.
- `axiom.yaml` no declara `technical_context.maintained`: no aplica el aviso de contexto y no hay `context-impact.md` en la carpeta del cambio.
- `openspec/config.yaml` `rules.verify` pide `go test ./...`, `go vet ./...` y comparar contra cada escenario: lo primero se cubre por CI y por la ejecución parcial descrita; lo segundo, por la matriz de §4.

## 3. Criterios de éxito de `proposal.md` §12

| # | Criterio | Estado | Evidencia |
|---|---|---|---|
| 1 | Decoupled retirado | Cumplido en código; ver AVISO A2 | `internal/workspace/loader.go` (`retiredTopologyMonorepoDecoupled`), `loader_test.go`, `validator_test.go`, `cmd/axiom/workspace_validate_test.go` (código de salida 1 y mensaje que nombra `multirepo`), `docs_contract_test.go` (ninguna doc, ejemplo o visión de arquitectura presenta el valor sin declararlo retirado). `git grep monorepo-decoupled` fuera de `openspec/changes/` solo devuelve pruebas, el rechazo del cargador, `docs/archive/ROADMAP.md` (histórico) y las cinco specs vivas pendientes de archivar. |
| 2 | Esquema validado | Cumplido | `internal/workspace/gitpolicy_test.go`, `internal/gitref/gitref_test.go`, `cmd/axiom/workspace_validate_test.go` (REQ-25.26, REQ-25.27). |
| 3 | Monorepo de un rol e idempotencia | Cumplido | `internal/branch/engine/ensure_test.go`, `ensure_registry_test.go`, `internal/branch/planner_test.go`, `internal/cli/branch_ensure_test.go`. |
| 4 | Monorepo multirrol | Cumplido | `internal/branch/planner_test.go`, `engine/ensure_test.go`, `engine/snapshot_test.go` (rama de seguimiento `<prefijo>/x` y ramas `<prefijo>/x-<rol>`). |
| 5 | Multirepo | Cumplido | `engine/ensure_multirepo_test.go`, `engine/status_registry_test.go`, `internal/cli/branch_registry_test.go`. |
| 6 | Sin red | Cumplido | `engine/ensure_fetch_test.go`, `engine/ensure_test.go` (observador sin operaciones remotas; `base.source`, `sha`, `fetched:false`), `gitcli/ops_test.go`; el puerto `Git` no tiene método de `push` (REQ-25.1). |
| 7 | `ask` tipado | Cumplido | `internal/branch/ensure_decision_test.go`, `engine/ensure_mode_test.go`. |
| 8 | `in_place` sucio | Cumplido | `internal/branch/ensure_dirty_test.go`, `engine/ensure_inplace_test.go`, `gitcli/ops_switch_test.go`, `internal/cli/branch_inplace_test.go`. |
| 9 | `done` seguro | Cumplido | `engine/done_test.go`, `engine/done_layers_test.go`, `internal/branch/merge_test.go`, `gitcli/ops_done_test.go`, `gitcli/capability_test.go`, `internal/cli/branch_done_test.go`, `branch_done_layers_test.go`. |
| 10 | Guardas | Cumplido | `engine/verdict_test.go`, `internal/branch/verdict_test.go`, `internal/cli/sdd_kickoff_branch_test.go`, `sdd_wrong_branch_test.go`, `sdd_writer_guard_test.go`, `engine/writer_inventory_test.go` (inventario de 12 escritores Go, uno por escritor), `internal/sddstatus/branch_context_test.go`. Un cambio sin instantánea no se bloquea: `engine/snapshot_test.go`, `verdict_test.go` (REQ-25.37). |
| 11 | Invariante (`axiom branch check`) | Sustituido por D17; ver SUGERENCIA S2 | El verbo `check` y el invariante se retiraron en la reapertura de spec del 2026-10-17 (compuerta `spec` aprobada de nuevo). REQ-25.1 fija cuatro verbos (`ensure`, `status`, `done`, `guard`) y `internal/cli/branch_test.go` prueba que `push`, `pr` y `check` no existen. En su lugar, REQ-25.20 lista los cambios pendientes de archivar (informativo): `engine/pending_test.go`, `internal/branch/result_pending_test.go`, `internal/cli/branch_pending_test.go`, `internal/dashboard/listing_test.go`, `internal/tui/model_test.go`. |
| 12 | Autoridad de edición | Cumplido | `internal/sddstatus/edit_authority_registered_test.go`, `internal/cli/sdd_registered_roots_test.go` (REQ-25.43, REQ-24.9). |
| 13 | Paridad | Cumplido | `internal/assets/assets_test.go` (pin: ningún asset del producto contiene `EnterWorktree` ni `ExitWorktree` salvo la prohibición explícita), `internal/components/agentguidance/routing_test.go` (la doctrina ODD invoca `axiom branch ensure --lane odd` y el texto «branch first when on the default branch» está retirado). |
| 14 | Hook | Cumplido | `engine/guard_hook_test.go` (fallo en abierto: carga de 1 MiB, JSON inválido, tiempo agotado, pánico; ningún proceso lanzado; protocolo de salida por agente), `guard_classify_test.go`, `guard_warn_test.go`, `internal/assets/opencode_branch_guard_plugin_test.go`, `internal/components/sdd/inject_test.go` (instalación por `sync` en Claude, Codex y OpenCode), `internal/components/uninstall/service_test.go`. Pi queda fuera por diseño (sin canal demostrado). |
| 15 | Hub y dashboard | Cumplido | `internal/hub/prefs_test.go`, `hub_test.go`, `cmd/axiom/init_registry_test.go`, `internal/dashboard/server_test.go`, `service_branch_test.go`, `assets_branch_test.go`, `listing_test.go`, `listing_tui_parity_test.go`, `internal/tui/model_test.go`, `internal/tui/screens/sdd_increments_test.go`, `cmd/axiom/change_create_branch_test.go`. |
| 16 | Dogfooding | Cumplido | `axiom.yaml` declara `git: {integration_branch: main, branch_prefix: inc}`; `CLAUDE.md`, `AGENTS.md` y `GEMINI.md` remiten a `axiom branch`; `internal/workspace/dogfooding_branch_status_test.go`. La transición de los cambios activos queda según Q5: `inc-23-knowledge-profile` se archiva en una tarea aparte (fuera de inc-25) y, sin `axiom branch check`, nada la bloquea. |
| 17 | CI verde | Cumplido por CI y por esta ejecución | Ver §2. Los ratchets de rechazos pasan en la ejecución dirigida de `internal/cli`; el ratchet de `deadcode` y los goldens completos se apoyan en la CI de #161 (no se han ejecutado aquí). |

La propuesta enumera 17 casillas; la tabla las recorre en el mismo orden.

## 4. Matriz de cumplimiento REQ → evidencia, por delta

Criterio de la columna «Evidencia»: ficheros de prueba que citan el requisito por su identificador y que han pasado en §2 (los de `internal/cli` pasan por el subconjunto dirigido). Además, la trazabilidad requisito → tareas de `tasks.md` está completa y todas sus tareas están marcadas salvo S12.26.

### 4.1 `change-branch-lifecycle` (nueva; REQ-25.1 a REQ-25.24 y REQ-25.45)

| REQ | Tema | Evidencia principal |
|---|---|---|
| 25.1 | Cuatro verbos, sin red ni publicación, un único punto de ejecución de git | `cmd/axiom/branch_help_test.go`, `internal/cli/branch_test.go`, `gitcli/ops_test.go`, `ops_write_test.go` |
| 25.2 | Salida común, rutas absolutas, códigos tipados | `internal/branch/result_test.go`, `labels_test.go`, `done_result_test.go`, `engine/ensure_test.go` |
| 25.3 | Resolución de la política | `internal/branch/policy_test.go`, `ensure_exception_test.go`, `engine/ensure_exception_test.go`, `gitref/gitref_test.go` |
| 25.4 | Modelo de ramas por número de roles | `internal/branch/planner_test.go`, `planner_archive_test.go`, `engine/snapshot_test.go` |
| 25.5 | Papel del repositorio según topología | `engine/ensure_multirepo_test.go`, `guard_test.go`, `internal/dashboard/planning_home_test.go` |
| 25.6 | Raíz git real | `engine/ensure_test.go`, `gitcommon/exec_test.go` |
| 25.7 | `ensure` determinista e idempotente | `engine/ensure_test.go`, `ensure_registry_test.go`, `archive_test.go` |
| 25.8 | Modos `worktree`, `in_place`, `ask` | `engine/ensure_mode_test.go`, `ensure_inplace_test.go`, `internal/branch/ensure_decision_test.go` |
| 25.9 | `in_place` rechaza árbol sucio | `internal/branch/ensure_dirty_test.go`, `engine/ensure_inplace_test.go` |
| 25.10 | Sin red; `--fetch` explícito | `engine/ensure_fetch_test.go`, `ensure_test.go`, `pending_test.go` |
| 25.11 | Rutas de worktree y límite de longitud | `engine/ensure_multirepo_test.go`, `internal/branch/planner_test.go`, `gitref/gitref_test.go` |
| 25.12 | Excepciones `--prefix`, `--branch`, `--base` | `internal/branch/ensure_exception_test.go`, `engine/ensure_exception_test.go` |
| 25.13 | Carril ODD | `engine/ensure_odd_test.go`, `internal/cli/branch_exception_test.go` |
| 25.14 | Registro local en el git common dir | `internal/branch/registry/registry_test.go`, `engine/status_registry_test.go` |
| 25.15 | `status` | `engine/status_test.go`, `status_archive_test.go`, `status_registry_test.go` |
| 25.16 | `done`: detección de fusión por capas | `engine/done_layers_test.go`, `internal/branch/merge_test.go`, `gitcli/capability_test.go` |
| 25.17 | `done`: seguridad y eliminación | `engine/done_test.go`, `gitcli/ops_done_test.go`, `internal/cli/branch_lineage_test.go` |
| 25.18 | Veredicto único de rama | `engine/verdict_test.go`, `internal/branch/verdict_test.go` |
| 25.19 | Guarda A en todo escritor Go | `engine/writer_inventory_test.go`, `internal/cli/sdd_writer_guard_test.go`, `cmd/axiom/handoff_branch_guard_test.go` |
| 25.20 | Pendientes de archivar (informativo) | `engine/pending_test.go`, `internal/cli/branch_pending_test.go`, `internal/dashboard/listing_test.go` |
| 25.21 | `guard`: bloquea documentos de Axiom, avisa el resto | `engine/guard_classify_test.go`, `guard_hook_test.go`, `guard_warn_test.go` |
| 25.22 | Instalación del hook por `sync`; sin hooks de git | `internal/components/sdd/inject_test.go`, `uninstall/service_test.go`, `internal/assets/opencode_branch_guard_plugin_test.go` |
| 25.23 | Skills y doctrina delegan en el verbo | `internal/assets/assets_test.go`, `internal/cli/sdd_gate_planning_home_test.go` |
| 25.24 | Dogfooding | `internal/workspace/dogfooding_branch_status_test.go`, `docs_contract_test.go` |
| 25.45 | Rama de archivado y `ensure --archive` | `engine/archive_test.go`, `status_archive_test.go`, `internal/branch/planner_archive_test.go`, `internal/cli/branch_archive_test.go` |

### 4.2 Deltas sobre capacidades vivas

| Capacidad | REQ | Evidencia principal |
|---|---|---|
| workspace-topology | REQ-1.2, REQ-2.1 | `internal/workspace/gitpolicy_test.go`, `loader_test.go`, `cmd/axiom/workspace_validate_test.go` |
| | REQ-2.3 (retirado) y REQ-25.25 | `workspace/loader_test.go`, `validator_test.go`, `locate_test.go`, `docs_contract_test.go` |
| | REQ-25.26, REQ-25.27 | `workspace/gitpolicy_test.go`, `gitcommon/files_test.go`, `cmd/axiom/workspace_validate_test.go` |
| planning-home-resolution | REQ-24.3, 24.5, 24.7 | `workspace/locate_test.go`, `sddstatus/planning_home_test.go`, `declarations_archive_block_test.go` |
| | REQ-24.9 | `sddstatus/edit_authority_registered_test.go`, `internal/cli/sdd_planning_home_test.go` |
| | REQ-25.28 | `workspace/locate_test.go`, `cmd/axiom/workspace_validate_test.go` |
| spec-baseline-declaration | REQ-24.13 | `workspace/declarations_test.go`, `docs_contract_test.go` |
| multi-project-hub | REQ-3.1, 3.2 | `internal/dashboard/server_test.go` |
| | REQ-25.29, 25.30, 25.31 | `internal/hub/prefs_test.go`, `hub_test.go`, `main_test.go`, `cmd/axiom/init_registry_test.go` |
| local-web-dashboard | REQ-1.1 | `dashboard/assets_branch_test.go`, `server_test.go` |
| | REQ-25.32, 25.33, 25.34 | `dashboard/service_branch_test.go`, `assets_branch_test.go`, `listing_test.go`, `listing_tui_parity_test.go`, `tui/model_test.go` |
| sdd-lane-kickoff | REQ-25.35 | `internal/assets/assets_test.go`, `internal/components/sdd/review_ledger_contract_test.go` |
| | REQ-25.36, 25.37, 25.38 | `internal/kickoff/gitsnapshot_test.go`, `engine/snapshot_test.go`, `internal/cli/sdd_kickoff_branch_test.go`, `engine/verdict_test.go` |
| sdd-archive-lifecycle | REQ-21.16 | `internal/sddstatus/governance_continuous_test.go`, `status_test.go`, `internal/cli/sdd_gate_planning_home_test.go` |
| | REQ-25.39 | `internal/cli/sdd_writer_guard_test.go`, `sdd_wrong_branch_test.go`, `sddstatus/branch_context_test.go` |
| organic-agent-trigger-rules | REQ-25.40, 25.41 | `internal/components/agentguidance/routing_test.go`, `internal/assets/assets_test.go` |
| axiom-sdd-cli-integration | REQ-25.42 | `sddstatus/branch_context_test.go`, `internal/cli/sdd_wrong_branch_test.go` |
| | REQ-25.43 | `sddstatus/edit_authority_registered_test.go`, `internal/cli/sdd_registered_roots_test.go` |
| | REQ-25.44 | `sddstatus/worktree_changes_test.go`, `internal/cli/sdd_registered_worktrees_test.go`, `engine/listing_test.go` |

Salvedad metodológica: la matriz se apoya en que cada prueba cita su requisito y en que la suite pasa; no se han releído uno a uno los escenarios Dado/Cuando/Entonces contra el cuerpo de cada prueba. La cobertura por escenario se delega en la trazabilidad de `tasks.md` (tareas de pruebas RED por escenario, todas marcadas) y en la batería de cada rebanada.

## 5. Desviaciones, avisos y sugerencias

### Avisos

- **A1. Cobertura no medida.** No se ha ejecutado `go test -cover` ni hay umbral declarado; la compuerta `role-apply:fullstack` ya constaba como «cobertura no medida».
- **A2. Specs vivas aún con `monorepo-decoupled`.** Las cinco specs de `openspec/specs/` (`workspace-topology`, `planning-home-resolution`, `sdd-archive-lifecycle`, `spec-baseline-declaration`, `local-web-dashboard`) mencionan todavía la topología como válida. Es lo esperado antes de archivar: los deltas las modifican al fusionarse y S12.26 cubre las tres ediciones a mano de `planning-home-resolution`. Hasta entonces, la parte «ninguna spec viva» del primer criterio de §12 no se cumple literalmente.
- **A3. Verificación local parcial de la suite.** No se ha ejecutado `go test ./...`: `internal/cli` completo y los paquetes con fallo de entorno conocido (`app`, `backup`, `filemerge`, `semantic`, `reviewtransaction`) quedan sin evidencia local, y el diff de inc-25 toca ficheros de `internal/cli`, `internal/reviewtransaction` e `internal/semantic`. La evidencia de suite completa es la CI de #161 citada (Unit Tests y Windows Runtime en SUCCESS; E2E, Darwin y Windows Full Suite SKIPPED), no una ejecución propia. El ratchet de `deadcode` tampoco se ha ejecutado aquí.

### Sugerencias

- **S1. S12.26 pendiente.** 213/214 tareas marcadas; S12.26 la ejecuta `sdd-archive` (decisión T10). Se recomienda que el archivo la deje cerrada y confirme que `workspace-topology` y `spec-baseline-declaration` no necesitan edición a mano (notas de archivado de sus deltas).
- **S2. Criterio «Invariante» de la propuesta, obsoleto por D17.** `proposal.md` §12 sigue describiendo `axiom branch check` y un invariante de CI; la reapertura de spec del 2026-10-17 los retiró (REQ-25.1: cuatro verbos; REQ-25.20: pendientes informativos). No es un incumplimiento, pero el texto de la propuesta no se actualizó; conviene que el archivo lo anote.
- **S3. Redacción de REQ-25.36.** El spec dice «un segundo `seal` se rechaza como hoy»; la implementación conserva el comportamiento de hoy (un segundo `seal` devuelve al ganador sin reescribir) y solo lo rechaza la guarda A desde otra rama (decisión registrada en `tasks.md`, fase S8). Es coherente con «como hoy» y con el sellado único; conviene fijar la redacción del requisito al archivar.
- **S4. Modo TDD.** `strict_tdd: false` en `openspec/config.yaml`: no se ha evaluado evidencia de TDD estricto. `tasks.md` registra RED observados en varias tareas, pero no se han auditado aquí.

### Riesgos residuales

- La capa D del hook es de mejor esfuerzo (settings globales en Claude, confianza explícita en Codex, excepciones tragadas en OpenCode, sin canal en Pi): las garantías son las capas A, B y C, tal como declara la propuesta (R10).
- `go test ./internal/branch/engine` tarda unos 250 s en esta máquina (170 s esperados): coste de pruebas con git real, no un fallo.
- Las operaciones remotas (PR, CI, fetch) quedan fuera de este informe: se citan como evidencia, no se han reverificado.

## 6. Recomendación siguiente

`sdd-archive`. Implementación completa e integrada en `main` (compuerta `integration` aprobada, PR #161). Las precondiciones de REQ-21.16 y REQ-25.39 se cumplen: evidencia `pr_merged` registrada (`99ee35a1`) y trabajo en la rama de archivado `inc/inc-25-ramas-y-worktrees-por-topologia-archive`. El archivo debe ejecutar S12.26 y entregarse en un PR documental aparte.
