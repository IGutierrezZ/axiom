# ODD: Seguimientos de la retirada de los restos del upstream Gentle AI

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/seguimientos-restos-upstream.md`.
> Espejo de recuperación en Engram: topic `odd/seguimientos-restos-upstream/tasks`, proyecto `axiom`.
> Rama del documento: `docs/odd-seguimientos-restos-upstream`, con su worktree en `C:\repos\axiom-wt\odd-seg-docs`. Este documento viaja **solo** en el PR de cierre.
> Abierto el 2026-10-07 desde `origin/main` `741b731e`. Recoge lo que quedó fuera del ODD `retirada-telemetria-y-restos-upstream` (cerrado con el PR #103).

---

## 0. Cómo retomar este ODD en una sesión nueva

1. Recupera el contexto: `mem_context`; `mem_search "odd/seguimientos-restos-upstream"` en el proyecto `axiom`; `mem_get_observation` sobre la observación de ese topic; y **este fichero**, que es la fuente autoritativa (`C:\repos\axiom-wt\odd-seg-docs\odd\tasks\` o `origin/docs/odd-seguimientos-restos-upstream`).
2. Comprueba el estado real: `git -C C:\repos\axiom fetch --prune origin`, `git worktree list`, `gh pr list --state open`.
3. Concilia este documento con la realidad y continúa con la **primera tarea sin marcar** de la sección 6.
4. Respeta las secciones 3 y 4.

## 1. Objetivo

Cerrar los seguimientos que dejó el ODD anterior: defectos funcionales que siguen dirigiendo a artefactos del upstream (`cmd/gentle-ai` en el auto-upgrade, el canal de actualización, ficheros de prompt huérfanos), la deuda de aislamiento y fiabilidad de los tests, la migración de instalaciones antiguas y los restos de marca, decidiendo qué contratos `gentle-ai.*` se migran y cuáles se mantienen.

## 2. Decisiones del usuario

| Decisión | Valor |
|---|---|
| Carril | ODD (2026-10-07) |
| Entrega | PRs independientes contra `main`; el documento va en un PR de cierre |
| `size:exception` | Aprobado (2026-10-07) para los PRs unificados U1-U5 de la sección 6. Cualquier otro PR sigue necesitando aprobación explícita |
| T7m dentro de este ODD o en uno propio | **Dentro de este ODD** (2026-10-07). El usuario pide unificar todas las tareas unificables y aprueba `size:exception` para esos PRs unificados, aunque superen 400 líneas |
| `cmd/gentle-ai`: retirarlo o mantenerlo | **Retirarlo del todo** (2026-10-07): borrar `cmd/gentle-ai` y el build `gentle-ai-deprecated`, ajustar `releasepolicy`, sus tests y la spec de identidad de distribución, y corregir los usos indebidos (registry, e2e, bench, guard del CI) |
| Alcance de los IDs de protocolo `gentle-ai.*/vN` (incluye `__managed_by` y variables `GENTLE_AI_ENGRAM_*`/`GENTLE_AI_SDD_STATUS_ENGRAM`) | **Mínimo** (2026-10-07): variables `AXIOM_ENGRAM_*` y `AXIOM_SDD_STATUS_ENGRAM` con respaldo `GENTLE_AI_*`; `__managed_by` escrito como `axiom/sdd`; T7m con eco del dialecto. Los ~156 IDs `gentle-ai.*/vN` sin gemelo se quedan como contrato *wire* documentado; los separadores de hash y los persistidos no se tocan |
| T7m: activar el dialecto en los prompts | **En este PR** (2026-10-07): commit (c) cambia `review-ledger-contract.md` y demás assets a `axiom.review-integration/v2` |
| T7m: herramienta por defecto en `capture-*` y `acknowledge-approved` (sin `--contract`) | ~~`axiom` por defecto~~ → **revisada (2026-10-07)**. El verificador demostró que rompía la reproducción exacta de la confirmación para los negociadores `gentle-ai` y les cambiaba el contrato tras `correction_required`. Decisión nueva: STATUS añade `--contract axiom.review-integration/v2` a los tokens de `capture-*` y `acknowledge-approved` solo para quien negoció `axiom`. Sin ese flag, `capture-*` emite `gentle-ai` como hasta ahora |

## 3. Convenciones (vigentes del ODD anterior, sección 3)

- **Idioma:** respuestas y artefactos en castellano peninsular; identificadores y comentarios Go en inglés.
- **Commits:** Conventional Commits en castellano, escritos a un fichero y usados con `git commit -F`, sin atribución de IA. El cuerpo del PR sí lleva la línea final `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- **PR:** plantilla `.github/PULL_REQUEST_TEMPLATE.md`, una sola etiqueta `type:*`, «AI Assistance» como «Material assistance used», la casilla de responsabilidad sin marcar, y las de `go test ./...`/E2E sin marcar si no se ejecutaron en local.
- **CI:** «Check PR Cognitive Load» ≤ 400 líneas salvo `size:exception`; `type:*`; Unit Tests; Go Format; `scripts/deadcode-ratchet.sh` (no separar una función de su consumidor).
- **Merge:** squash. No se consulta el CI en bucle: el usuario dice «revisa si ya está verde y continúa». Tras cada merge se borran el worktree y las ramas local y remota.
- **Worktrees:** `C:\repos\axiom-wt\<nombre>`, creados con **ruta absoluta** desde `origin/main`. El checkout principal `C:\repos\axiom` es de otra sesión (inc-24) y no se toca.
- **ODD:** TDD desactivado (`openspec/config.yaml:16`, `strict_tdd: false`), runner `go test`. RDD desactivado (`axiom review mode status`: off, decidido por global). Riesgo por commit con `axiom review assess --cwd . --base-ref origin/main --committed-only --json`; `high` exige verificador independiente de solo lectura. Un writer cada vez, en su worktree, ~60 min, tests por nombre exacto, máximo 2 procesos, `sdd` en solitario. Todas las llamadas a `Agent` llevan `model: sonnet`.

## 4. Reglas de seguridad (vigentes del ODD anterior, sección 4)

1. Nunca matar procesos por nombre de imagen; solo PIDs propios.
2. Nunca ejecutar `install`, `setup`, `sync`, `upgrade` ni `doctor` contra el HOME o el PATH reales: en pruebas, `HOME`, `USERPROFILE`, `PATH` temporales.
3. Nunca `go test ./internal/cli/` completo en local; subconjuntos con `-run`.
4. Pruebas Linux por compilación cruzada y ejecución en una sola invocación de WSL (ver ODD anterior, 4.5).
5. Nada de perl con rutas Windows en el reemplazo; usar `Edit` o Python.
6. Fallos de entorno conocidos en esta máquina que también fallan en `main`: ODD anterior 4.8, más `deadcode-ratchet.sh` y `sddstatus` `TestCharacterization_EditAuthorityMissingBlocksApplyAndArchive` (precisamente D6 de este ODD).

## 5. Inventario conciliado con `origin/main` `741b731e`

Verificado con dos exploraciones de solo lectura y comprobación directa del orquestador.

| ID | Punto | Estado real y líneas actuales | Clase | Tamaño | Riesgo |
|---|---|---|---|---|---|
| A1a | `cmd/gentle-ai` en uso | `cmd/gentle-ai/main.go` **ya no es un shim**: imprime «retirado definitivamente» y sale con 1. `internal/update/registry.go:32` (`GoImportPath`) hace que el auto-upgrade con Go ejecute `go install …/cmd/gentle-ai@vX` e instale ese stub; `instructions.go:20` lo recomienda. Los Dockerfiles `e2e/Dockerfile.{ubuntu:75,fedora:67,arch:69,claude-network-none:6}` y `e2e/lib.sh:51-54` construyen el stub como producto. `bench/record.go:93` crea un shim `gentle-ai` mientras el agente ejecuta `axiom`. `ci.yml:212` busca un aviso «'gentle-ai' está deprecado» que ya no existe | defecto | ~40 | medio |
| A1b | Retirar el build `gentle-ai-deprecated` | `.goreleaser.yaml:12-15,35-52`; `internal/releasepolicy/policy.go:348-351,597-599` exige dos builds; `windows_distribution_policy_test.go:544-545`; `cmd/axiom/canonical_binary_test.go:26`; `openspec/specs/axiom-distribution-identity/spec.md:64` | marca/contrato de release | ~150 | alto |
| A2 | Docs con nombres antiguos | `docs/intended-usage.md:66,78,80,84,185` y `docs/prd-opencode-profiles.md:119,131,265,707`; nombres vigentes en `docs/opencode-profiles.md:11,77-79`. Por revisar: `docs/pi.md:94` (`/gentle-sdd-init`) | marca | ~10 | bajo |
| A3 | Banner de `install.sh` | `scripts/install.sh:528-532` (arte «Gentle-AI») y `:534`; ningún test lo fija | marca | ~6 | bajo (`assess` puede dar `high` por `install.sh`) |
| A4a | Prompts huérfanos de Kiro y VS Code | Ya escriben `axiom.md` (`kiro/adapter.go:94`) y `axiom.instructions.md` (`vscode/adapter.go:87`), pero no retiran `gentle-ai.md` ni `gentle-ai.instructions.md`: duplican el prompt. Patrón de limpieza: `persona/inject.go:740` | defecto | 40-70 | medio (borra ficheros: verificar propiedad) |
| A4b | Cursor `gentle-ai.mdc` | `cursor/adapter.go:77`, decisión documentada en `docs/agents.md:138`, fijada en ~8 tests | marca | ~15 + migración | bajo-medio |
| A5a | Variables `GENTLE_AI_ENGRAM_SETUP_MODE/STRICT` y `GENTLE_AI_SDD_STATUS_ENGRAM` | No existen `AXIOM_*`: `engram/setup.go:78-79`, `cli/run.go:1616-1617`, `sddstatus/status.go:788`. Helper dual disponible: `system.Getenv` (`system/env.go:16,30`) | contrato | ~10 + tests | bajo |
| A5b | `__managed_by: gentle-ai/sdd` | Se escribe 46 veces en `internal/assets/opencode/sdd-overlay-{multi,single}.json`; la lectura ya es dual (`opencode/config.go:322`) | contrato | ~60 + goldens y hashes | medio |
| A5c | IDs `gentle-ai.*/vN` sin lectura dual | 156 sin gemelo `axiom.*` (review-* 78, review-integration* 30, verification-* 12, rdd-* 6, rar-* 6, sdd-* 5, release-* 2, varios 17). **No se tocan** los 24 usados como separador de hash ni los persistidos (`review-state/v2`, `review-state-record/v2`, `review-snapshot/v1,v2`, `review-transport/v2`, `compact-trace-entry/v1`) | contrato | grande | alto |
| T7m | Dialecto `axiom` en los comandos del contrato de revisión | Ver sección 5.1 | contrato | 650-800 | medio-alto |
| B1 | `AXIOM_CHANNEL` sin fallback | Solo `internal/update/check.go:15,182`. `upgrade/strategy.go:742` ya es dual vía `cli.ResolveInstallChannel` | defecto | ~11 | bajo |
| B2 | Canal en los instaladores | `scripts/install.sh:97,544,567` solo `GENTLE_AI_CHANNEL`; `scripts/install.ps1:166,181` solo `AXIOM_CHANNEL` (al revés); `docs/platforms.md:52-53` documenta `AXIOM_CHANNEL` | defecto | ~6 | bajo (`assess` puede dar `high`) |
| C1 | Digests históricos de plugins | `internal/components/sdd/opencode_plugin_digests.go:16-43` solo cubre `skill-registry.ts` y `opencode-review-transport.ts`. Faltan 13 digests: `model-variants.ts` (6 v1) y `sdd-task-result-artifacts.ts` (6 v1 + 1 v2). Efecto en `opencode_runtime.go:52` | defecto | ~25 + test | bajo |
| C2 | Aviso de `sync` perdido en la TUI | `internal/app/app.go:611-646` (`tuiSync` devuelve solo `ChangedFiles`); `tui/model.go:345,468`; avisos en `cli/sync.go:2205,2209,2286,2381-2391` | defecto | ~140 | medio |
| D1 | `LOCALAPPDATA`/`APPDATA` fuera del sandbox | `internal/app/testmain_test.go:17-38` | deuda de tests | ~6 | bajo |
| D2 | `AXIOM_STATE_DIR` en `TestMain` | `internal/app/testmain_test.go`, `internal/cli/protocol_probe_test.go:50`, `internal/dashboard/main_test.go:32`, `cmd/axiom/testmain_test.go:31`, `internal/reviewtransaction/snapshot_test.go:31` | deuda de tests | ~15 | bajo |
| D3 | Bench sin `AXIOM_NO_PERSISTENT_PATH` | `bench/runner.go:87-116` (`Sandbox.env()`); `bench` es módulo aparte: literal | deuda de tests | ~2 + test | bajo |
| D4 | Tests intermitentes | `internal/assets/opencode_v2_plugins_test.go:38` (espera fija `setTimeout(30)` y lectura del `.tmp`); `internal/update/cooldown_concurrency_test.go:18` (subproceso real con 10 s y sin `AXIOM_STATE_DIR`) | deuda de tests | pequeño | bajo |
| D5 | `TestDocumentedInvocationsRunAsDocumented` no termina en Windows | `internal/app/documented_invocation_test.go:385`: verbos «seguros» que caen al flujo real sin timeout | deuda de tests | pequeño-medio | bajo |
| D6 | Fallos locales en Windows | `sddstatus/verify_archive_gate_characterization_test.go:138` (nombres 8.3 frente a `EvalSymlinks`); `deadcode-ratchet.sh` (GOOS=windows frente a baseline Linux, y fallo silencioso de `go run` sin red) | deuda de tests | pequeño | bajo |

### 5.1 T7m (resumen del mapa)

- **Schemas v2 que fijan solo `gentle-ai`:** `consent.schema.json:44,58`, `consent-v3.schema.json:45,59`, `capabilities{,-v2.1,-v2.2,-v2.3}.schema.json:101-103` (v2.4-v2.6 referencian v2.3). Ya duales: `transition-execution` y `status-v7:83`. **v1 está congelado** (`v1/FREEZE.md`): los flujos `--contract …/v1` siguen emitiendo `gentle-ai`.
- **Productores:** `reviewTransitionCommandTool` (`review_next_transition.go:1289`), `reviewNegotiatedStartCommand` (`review_facade.go:201`), `reviewConsentFollowUpBase` (`:2489`), `reviewConsentOffPathCommand` (`review_mode.go:589`), `reviewNextTransitionRefreshCommand{,V21}` (`review_capabilities.go:61-62`).
- **Defecto latente:** 26 comparaciones `== ReviewIntegrationContractV2` en producción sobre el flag crudo: con `--contract axiom.review-integration/v2` esas ramas no se activan (por ejemplo `review_next_transition.go:944` omite `--consent relay`). `MatchReviewDialect` e `IsReviewContractSupported` no tienen llamadores (`.deadcode-baseline.txt:112-113`).
- **Validadores exactos:** `review_consent_contract.go:338,344`, `review_status_contract.go:1633`, `review_capabilities.go:428`.
- **Regla propuesta:** eco del dialecto — `axiom` solo si el llamador negoció `axiom.*`.
- **Cortes:** PR-A dual-read + validadores + schemas ampliados en su sitio (~350); PR-B1 transiciones (~350); PR-B2 consent y capabilities (~300); PR-C fixtures, docs, bench, e2e (~250). Se planifica con un agente Plan de solo lectura antes de implementar.

## 6. Tareas (unificadas tras las decisiones del 2026-10-07)

> Estado: `[ ]` pendiente · `[~]` en curso o con PR abierto · `[x]` fusionado. Estrategia de entrega: `single-pr` por tarea unificada, con `size:exception` aprobado. Ruta: delegada (un writer por tarea, en su worktree), porque todas tocan 2 o más ficheros no triviales. Orden: primero U2, para que los tests de las demás ya corran aislados.

- [x] **U2 · Aislamiento y fiabilidad de tests** (D1-D6): **PR #104, fusionado como `935aebb1`** (`a73b831a` + `5f13c8e9`, rama `test/aislamiento-y-fiabilidad`, worktree `odd-seg-u2`; +282/−18; riesgo `high`; verificador PASS WITH NOTES). Ruta: delegada (writer Sonnet, 12 ficheros); las dos notas del verificador se corrigieron inline (2 ediciones mecánicas). Pendiente del CI. `LOCALAPPDATA`/`APPDATA` y `AXIOM_STATE_DIR` en los `TestMain` señalados; `AXIOM_NO_PERSISTENT_PATH` en `bench/runner.go`; espera por condición en `opencode_v2_plugins_test.go` e ignorar `.tmp`; entorno aislado y presupuesto en `cooldown_concurrency_test.go`; timeout y sandbox completo en `documented_invocation_test.go`; canonicalizar rutas en el test de `sddstatus`; `deadcode-ratchet.sh` con `GOOS=linux` y fallo explícito si `go run` falla. ~150-250 líneas. Riesgo previsto bajo-medio.
- [x] **U1 · Variables y canal `AXIOM_*` con respaldo `GENTLE_AI_*`** (B1 + B2 + A5a + A3): **PR #105, fusionado como `f3bf9ed8`** (`a30dd755`, rama `fix/variables-y-canal-axiom`, worktree `odd-seg-u1`; 15 ficheros, +210/−41; `type:bug`; riesgo `high`; verificador PASS WITH NOTES). Ruta: delegada (writer Sonnet), más un comentario obsoleto de `engram/download.go` corregido inline. Pendiente del CI. `update/check.go` con `system.Getenv`; `AXIOM_ENGRAM_SETUP_MODE/STRICT` y `AXIOM_SDD_STATUS_ENGRAM`; `install.sh` y `install.ps1` con el mismo orden (`AXIOM_CHANNEL` y respaldo); banner y textos de ayuda de `install.sh`; docs de variables. ~80-120 líneas. Riesgo previsto `high` por `install.sh` → verificador.
- [x] **U3 · Migración de instalaciones en `sync`** (C1 + C2 + A4a + A4b): **PR #106, fusionado como `6335b56d`** (`25d37652`, `7ab87ed3`, `66ba2ae4`, `cb274e89`; rama `fix/migracion-instalaciones-sync`, worktree `odd-seg-u3`; 41 ficheros, +1305/−151; `type:bug` + `size:exception`; riesgo `high`). Ruta: delegada (writer Sonnet). Verificador: FAIL → una corrección acotada → validador del arreglo PASS WITH NOTES. Pendiente del CI. 13 digests históricos; aviso de `sync` en la TUI; retirada de `gentle-ai.md` y `gentle-ai.instructions.md` de Kiro y VS Code solo si son de Axiom; Cursor pasa a `axiom.mdc` con migración del `gentle-ai.mdc` gestionado. ~300-400 líneas. Riesgo previsto medio-alto (borra ficheros del usuario) → verificador.
- [~] **U4 · Retirada de `cmd/gentle-ai` y restos de marca** (A1a + A1b + A2 + A5b): **PR #107** (`42a8eb6c`, `614634fe`, `e1ff0e90`, `bfc3d344`, `8eb0a495`; rama `chore/retirar-cmd-gentle-ai`, worktree `odd-seg-u4`; 51 ficheros, +319/−333; `type:chore` + `size:exception`; riesgo `high`; verificador PASS WITH NOTES). Ruta: delegada (writer Sonnet); `bfc3d344` y `8eb0a495` inline (ediciones mecánicas). Pendiente del CI. `registry.go` e `instructions.go` a `cmd/axiom`; Dockerfiles y `lib.sh` de e2e; shim de `bench/record.go`; guard de `ci.yml:212`; borrar `cmd/gentle-ai` y el build `gentle-ai-deprecated`; `releasepolicy` con un solo build; tests y spec `axiom-distribution-identity`; docs con nombres vigentes; `__managed_by: axiom/sdd` con goldens y hashes. ~300-400 líneas más goldens. Riesgo previsto `high` (release) → verificador.
- [x] **U5 · T7m: dialecto `axiom` en los comandos del contrato de revisión:** **PR #108, fusionado como `250ff359`**. Commits `107b725e`, `baf78944`, `01639d2d` y `796d4cd2`, en la rama `feat/dialecto-axiom-contrato-revision` (worktree `odd-seg-u5`); 67 ficheros, +1970/−286; `type:feature` + `size:exception`. Verificador: FAIL → decisión del usuario → validador del arreglo PASS WITH NOTES. Ruta: agente Plan más writers delegados por etapas. Pendiente del CI. Antes, un agente Plan de solo lectura fija el diseño (eco del dialecto, helper de contrato, schemas v2 ampliados en su sitio frente a versiones nuevas, v1 intacto). Dos commits de unidad de trabajo en un solo PR: (a) lectura dual, las 26 comparaciones, los validadores exactos y los schemas; (b) productores con dialecto, fixtures, docs, bench, e2e y crosslane. ~650-800 líneas. Riesgo `high` → verificador.
- [ ] **Cierre:** PR con este documento.

Previsión total: ~1.500-1.900 líneas en 5 PRs más el de cierre.

## 7. Criterios de aceptación

- Ningún camino de actualización ni prueba de extremo a extremo instala o ejecuta el stub `gentle-ai` como si fuera el producto.
- `AXIOM_*` es la variable principal y `GENTLE_AI_*` el respaldo, de forma consistente en Go, `install.sh` e `install.ps1`.
- `sync` sustituye las copias históricas de los plugins de OpenCode sin bloquearse, y retira los prompts huérfanos de Kiro y VS Code solo si son de Axiom.
- La TUI muestra los avisos de `sync`.
- Los tests no leen ni escriben `LOCALAPPDATA`, `APPDATA` ni `~/.axiom` reales; los intermitentes identificados dejan de serlo o se documenta por qué no.
- CI en verde en todos los PRs.

## 8. Progreso

- **2026-10-07, apertura:** contexto recuperado (Engram #457, #522 y el documento del ODD anterior en `origin/main`); sin PRs abiertos ni worktrees previos. Worktree `odd-seg-docs` creado desde `741b731e`. Dos exploraciones de solo lectura (T7m y A-D) y comprobación directa de `cmd/gentle-ai/main.go`, `registry.go:32`, `check.go:182`, `install.sh:567`, Dockerfiles de e2e y `ci.yml:212`. Pendientes las tres decisiones de la sección 2.
- **2026-10-07, decisiones:** T7m entra en este ODD; se unifican las tareas y se aprueba `size:exception` para U1-U5; `cmd/gentle-ai` se retira del todo; alcance mínimo de los contratos `gentle-ai.*`. La lista S1-S11 se reagrupa en U1-U5.
- **2026-10-07, U2:** writer (~97 min) con estado *partial* y causas medidas:
  - **D1/D2:** fijar `AXIOM_STATE_DIR` de forma global rompía 14 `TestSelfUpdate_*`, así que se elimina en lugar de fijarse.
  - **D4b:** presupuestos de build y de actor demasiado ajustados.
  - **D5:** el test era lento por los shims reales del PATH, no estaba colgado.
  - **D6:** nombres 8.3 en `sddstatus`; `GOOS=windows` frente al baseline Linux en el ratchet.

  `assess`: `high` (`deadcode-ratchet.sh`, test de `update`). Verificador independiente: **PASS WITH NOTES**. El orquestador corrigió sus dos notas en `5f13c8e9`:
  - el presupuesto de build declarado no se usaba;
  - `GOCACHE`/`GOENV` quedaban fríos al aislar `LOCALAPPDATA`/`APPDATA`.

  Con eso, los tests de cooldown y `TestSelfUpdate|TestTUIExecuteWithBackground…` pasan en verde (22 s frente a 120 s). PR #104 abierto con `type:chore`, sin `size:exception` (300 líneas).
  - **Pendiente sin veredicto local:** `internal/reviewtransaction` completo (supera el timeout en Windows); la limpieza de `bench` falla por un `opencode.exe` bloqueado (preexistente).

- **2026-10-07, U1:** el writer (~33 min) implementa B1, A5a, B2 y A3.
  - No había docs que mencionaran las variables de Engram ni las de `sddstatus`.
  - `assess`: `high` (`install.sh`, `update/check.go`). Verificador: **PASS WITH NOTES**. Comprobó la precedencia en bash y PowerShell sin ejecutar los instaladores. `install.ps1` sigue en ASCII sin BOM.
  - PR #105 con `type:bug`.
  - **Para U4:** `bench/runner.go:113` fija `GENTLE_AI_INSTALL_SCOPE`, cuando `AXIOM_INSTALL_SCOPE` es la principal.
  - **Fuera de alcance:** lecturas de un solo nombre en fixtures de bench y de transporte (`GENTLE_AI_BENCH_SDD_PLUGIN`, `GENTLE_AI_BENCH_CRASH_AT_PHASE`, `GENTLE_AI_OPENCODE_RELAY_CONTRACT`), que son protocolo interno.
- **2026-10-07, U3:** el writer entrega C1, C2 y A4 en tres commits.
  - **Primer verificador: FAIL.** La prueba de propiedad aceptaba cualquier ID de sección, de modo que un bloque `axiom:<id-propio>` del usuario provocaba el borrado del heredado.
  - **Corrección acotada** (`cb274e89`): lista de IDs sacada de constantes compartidas con los inyectores; prompt vigente regular y con sección gestionada; heredados dentro del snapshot de install y sync; tests.
  - **Validador del arreglo: PASS WITH NOTES.** Repitió el ataque en 9 combinaciones; una instalación completa se retira sin falsos positivos; los 13 digests coinciden.
  - **Ratchet:** la versión corregida en U2 (`GOOS=linux`) da «no new unreachable functions» sobre U3.
  - **Pendiente:** los avisos del flujo de desinstalación con clean install en la TUI.
  - **Limitación conservadora:** el heredado creado solo con SDD, sin persona, se conserva con aviso.
  - PR #106 abierto con `size:exception`.
- **2026-10-07, U4:** el writer entrega A1a, A1b y A2+A5b en tres commits.
  - **`docs/pi.md` no se toca:** `/gentle-sdd-init` es del paquete externo `gentle-pi`.
  - **Defecto encontrado** y corregido inline en `bfc3d344`: el `init()` de `cmd/axiom` reescribía `GoImportPath` sin `/v3`, herencia de cuando el módulo era `gentle-ai/v3`. Con eso `GoInstallResolvable()` daba falso y la autoactualización con `go install` en Windows nunca estaba disponible.
  - **Verificador: PASS WITH NOTES.** Sus notas sobre los docs de `bench` (el shim seguía llamándose `gentle-ai`) y el regex del smoke se corrigieron en `8eb0a495`.
  - **Seguimientos nuevos:**
    - la constante `version` de `cmd/axiom/main.go` debe subirse en cada release, porque `go install` no inyecta ldflags;
    - `update` `TestCheckAllWithCooldown_ContendedPersistenceRejectsReviewModeDisable` es intermitente con 3 `go test` en paralelo;
    - los comentarios de `scripts/deadcode-ratchet.sh:33,39` y `PRD.md:1188` siguen nombrando `cmd/gentle-ai`.
  - PR #107.
- **2026-10-07, plan de U5 (agente Plan):**
  - **Mapa corregido:** son 10 puntos de decisión de un solo dialecto, no 26. Con `axiom.*`, el consent sale como v1.
  - **Dialecto:** se deriva en cada invocación del `--contract` y no se persiste.
  - **Schemas:** se amplían en su sitio (precedente #48), sin versiones nuevas; v1 queda intacto.
  - **Commits:** tres, (a) lectura dual, (b) productores con eco y (c) activación en los prompts.
  - **Decisiones del usuario:** activar en este PR; `axiom` por defecto en `capture-*`.
- **2026-10-07, U5 commit (a):** `107b725e` (+504/−39; producción +52/−21).
  - **Lectura dual:** los 10 puntos de decisión usan `isReviewContractV2`; los validadores aceptan los dos dialectos con `reviewCommandCanonicalTool`; se borra `IsReviewContractSupported` con su línea del baseline.
  - **Schemas:** consent, consent-v3 y capabilities hasta v2.3 se amplían en su sitio, con 5 hashes re-fijados.
  - **Tests:** `review_dialect_test.go`, que incluye la regresión de que `axiom.*` recibía el consent v1.
  - **Verificación:** 61 + 103 tests de `internal/cli` en verde por nombre exacto, salvo un caso de symlinks (entorno); ratchet limpio.
  - Etapa (b) en curso.
- **2026-10-07, U5 commit (b):** `baf78944`. Acumulado: +794/−121 en 36 ficheros.
  - **Tipo `reviewDialect`:** se deriva de `--contract` en cada invocación y se enhebra por las transiciones, START, capabilities y consent.
  - **Validadores:** exigen coherencia estricta entre herramienta y contrato.
  - **`capture-*` y `acknowledge-approved`:** emiten `axiom` por defecto, tal como decidió el usuario.
  - **Herramientas:** `-dialect` en crosslane, caso axiom en bench y variante en e2e (solo `go vet`).
  - **Guard:** un lineage gentle-ai y uno axiom producen bytes equivalentes salvo herramienta y contrato.
  - **Verificación:** ~250 tests de `internal/cli` en verde por nombre exacto. Quedan sin ejecutar en local, por timeout, 48 tests de transporte OpenCode y de STATUS negociado, que no tocan el código modificado; los cubre el CI.
  - Etapa (c), activación en los prompts, en curso con un writer nuevo.
- **2026-10-07, U5 commit (c):** `01639d2d` (+60/−18).
  - `review-ledger-contract.md` y `.codex/AGENTS.md` pasan a negociar `axiom.review-integration/v2`.
  - Se actualizan 6 goldens, con una línea cada uno.
  - Los prompts de OpenCode preservados migran al reemplazarse la sección (test nuevo).
  - El permiso del validador ya admitía ambos dialectos.
- **Verificador de U5 (a+b+c): FAIL** por la excepción de `capture-*`.
  - El resto de invariantes se cumplen: gentle-ai v2 idéntico en START, STATUS, consent y capabilities; v1 intacto; los hashes coinciden; los goldens solo cambian el token; no hay código muerto.
  - Notas: el START sin contrato pasaba a `axiom`; el stop-hook sigue negociando `gentle-ai`; la doc exagera al decir que nunca se mezclan; el envelope de fallo de `capture-*` publica el contrato `gentle-ai`.
  - El usuario elige `--contract` en los tokens. Commit (d) en curso.
- **2026-10-07, U5 commit (d):** `796d4cd2`.
  - Las capturas y `acknowledge-approved` aceptan un `--contract` opcional (solo v2), que STATUS y START les pasan solo a quien negoció `axiom`.
  - El START sin contrato vuelve a `gentle-ai`.
  - El stop-hook pasa a negociar `axiom`.
  - Eco del dialecto también en el relé de OpenCode (añadido por el writer, necesario con los prompts en `axiom`).
  - `transition-execution`, `status-v5` y `status-v9` se amplían en su sitio.
- **Validador del arreglo: PASS WITH NOTES.** Comparó los binarios de `main` y de la rama recorriendo el ciclo completo por CLI:
  - `gentle-ai` v2 y sin contrato: 0 diferencias.
  - `axiom`: la confirmación coincide con la que reofrece STATUS y la continuación mantiene el contrato.
  - Los 7 hashes coinciden.
  - Ratchet del orquestador: limpio.
  - Se abre el PR #108.
- **2026-10-08, primera tanda de merges:** el usuario pide revisar el CI y continuar.
  - **Fusionados con squash:** #105 (`f3bf9ed8`), #106 (`6335b56d`) y #104 (`935aebb1`).
  - **#104:** aparecía `BLOCKED`, porque una de sus dos ejecuciones simultáneas de «PR Validation» quedó cancelada. Se relanzó la cancelada y pasó a `CLEAN`.
  - **`main` combinado tras #105 y #106**, en el worktree temporal `odd-seg-check`: build y vet limpios; en verde `persona`, `engram`, `uninstall`, `system`, `update/upgrade` y los tests cruzados de `internal/cli`.
  - **Limpieza:** eliminados los worktrees `odd-seg-u1`, `odd-seg-u2` y `odd-seg-u3` y sus ramas, local y remota.
  - **#107:** conflicto previsto con #104 en `bench/runner.go`. Se resolvió conservando `AXIOM_NO_PERSISTENT_PATH` y quitando `GENTLE_AI_INSTALL_SCOPE`, y se rebasó sobre `main` (cabeza `9a524dbc`). En verde build, vet, `cmd/...` y `update/upgrade`. Fallos de entorno ya conocidos: `releasepolicy` (symlink) y la limpieza de `bench` (`opencode.exe` bloqueado). Subido con `--force-with-lease`; falta su CI.
  - **#108:** el CI dio rojo dos veces por fijaciones antiguas.
    - **Primera:** `TestNamedReviewStatusNextTransitionIsAlwaysComplete` y `TestRunSyncWithSelection_WritesExpectedFiles` esperaban el prompt negociando `gentle-ai`. Corregido en `3e7cb072`.
    - **Segunda:** el journey de bench `j125` esperaba `gentle-ai review start` del stop-hook. Corregido en `9f53ae37`; `j125` pasa en local.
    - **Lección:** el paso «Run benchmark evidence» del CI ejecuta journeys de bench que ningún verificador local había corrido. Para cambios en la superficie de comandos, hay que pasar `gentle-ai-bench run` antes del PR.
    - **Benchmark local en Windows, no concluyente:**
      - `j105` falla porque el shim del proveedor que crea bench no se ejecuta en Windows. Además, el arnés **llamó al `claude` real del PATH**, que respondió «Not logged in» sin hacer nada. Es un seguimiento nuevo: bench no aísla el PATH del proveedor en Windows.
      - En el eje de transición fallan 13 de 56 journeys (`operation_timeout`, capturas vacías) y no se han comparado con `main`.
      - La referencia es el CI de Linux.
- **2026-10-08, segunda revisión del CI:**
  - #108 en verde: fusionado (`250ff359`). Eliminados su worktree y sus ramas.
  - #107 en rojo por `TestDocumentedInvocationsRunAsDocumented`: la limpieza del `TempDir` falla con «directory not empty», cada vez en un subtest `executed/sync_*` distinto.
  - **`main` falla igual desde `935aebb1`** (#104, junto con #105 y #106), así que no es un fallo de #107. Algo sigue escribiendo en el sandbox cuando `sync` ya ha devuelto el control.
  - **Hotfix H1** en curso: worktree `odd-seg-fix`, rama `test/documented-invocations-tempdir`. Reproducción en WSL y causa raíz antes de tocar nada.
  - Después: relanzar el CI de #107 y fusionarlo.

## 9. Siguiente paso

Esperar el CI de #104-#108; el usuario avisa con «revisa si ya está verde y continúa».

Orden de merge propuesto, con squash: #104 (U2) → #105 (U1) → #106 (U3) → #107 (U4) → #108 (U5). Después de cada merge, rebase de los siguientes si hay conflicto:
- #107 con #104, en `bench/runner.go`;
- #108 con #107, en `e2e/organicruntime`.

Después, el PR de cierre con este documento (rama `docs/odd-seguimientos-restos-upstream`).

Seguimientos para después del ODD:
- bench en Windows: el shim del proveedor no se ejecuta y el arnés llega al `claude` real del PATH;
- la constante `version` de `cmd/axiom/main.go` en cada release;
- `ContendedPersistenceRejectsReviewModeDisable` y `TestRejectedTargetedValidatorCaptureRoutesEscalatedRecovery`, intermitentes bajo carga;
- los avisos del flujo de desinstalación con clean install en la TUI;
- el heredado creado solo con SDD, que se conserva con aviso;
- los comentarios de `deadcode-ratchet.sh:33,39` y `PRD.md:1188`.
