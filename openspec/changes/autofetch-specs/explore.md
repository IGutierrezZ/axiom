# Exploración: autofetch-specs (C-05-d)

Cambio: «Autofetch del repo de specs: avisar y avanzar». Rama `inc/autofetch-specs`. Solo lectura del código; las líneas son las del worktree (algunas difieren del backlog).

## 1. Estado actual (mapa con evidencia)

### Git y guardas
- `internal/branch/gitcli/ops.go:449` `Fetch` = `runWrite(ctx, dir, "fetch", remote)` (sin `--prune`); `runWrite` en `:458` usa `gitcommon.Run`.
- `internal/gitcommon/exec.go:21` `runTimeout = 30s`; `:109` `Run`; `:124` `RunWithTimeout(ctx, dir, timeout, args...)` ya existe y devuelve `ErrTimeout` (`:116`). `gitEnvironment` (`:60-69`) ya fija `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`, `LC_ALL=C`. Para el timeout de ~8 s basta `RunWithTimeout`; no hay que tocar el entorno.
- Guardas que NO bloquean `merge --ff-only`:
  - `internal/sddpublish/isolation_test.go:136` `publishingLiterals = {push, commit}`; `:241` solo vigila esos dos literales en `internal/branch/**`.
  - `internal/gitcommon/architecture_test.go:106` prohíbe `os/exec`, `net`, `net/http` fuera de `exec.go`; `:247-297` define `remoteSubcommand` (fetch, pull, push, ls-remote).
  - `internal/branch/gitcli/ops_write_test.go:165` `TestWritesLaunchOnlyBranchAndWorktreeAdd` y `:195` `TestFetchBringsTheRemoteCommitsLaunchingOnlyFetchOrigin` solo comprueban sus propias operaciones: añadir `FastForward` es aditivo (con prueba nueva propia).
- Pruebas del motor que fijan «sin `--fetch` no hay red» y siguen válidas con el interruptor apagado: `engine/ensure_test.go:422`, `engine/ensure_fetch_test.go:138`, `engine/done_test.go:550`, `engine/archive_test.go:248`, `engine/status_archive_test.go:218`, `engine/ensure_adopt_test.go:52`.

### Motor
- `engine/engine.go:70-79` interfaz `gitWriter` (`Fetch` en `:78`); `:90-91` campo `prefs branch.Prefs`; `:99-101` `Default()` con `hubPrefs{}`. Comentario `:62-69` y `branch/ports.go:40` («Fetch, la única operación remota») hay que reescribirlos.
- `engine/ensure.go:17-31` `EnsureRequest.Fetch`; `:171` `checkPlan(..., r.Fetch)`; `:290-305` `checkRepository` hace `e.writer.Fetch(pending.mainCheckout, originRemote)` y, si falla, rechaza con `FetchFailed` (`branch/ensure_refusal.go:100`). Hoy ya trae todos los repos del plan con ramas, incluidos los de código: P14(b) = ensure con interruptor encendido ≡ `--fetch` implícito. `facts.prefs` (de `workPrefs`, `prefs.go:53-73`) llega a `resolveMode`; el booleano del interruptor puede viajar ahí.
- `engine/done.go:104` y `ensure.go` archive: `--fetch` también en `done` y `ensure --archive`; el backlog solo nombra `ensure`.
- `engine/status.go:308` `AheadBehind` solo con referencias locales; no existe sección `remote` en el JSON (`branch/result.go:36` `Result`, `:173` `RepositoryResult`).
- `engine/prefs.go:21-44` `hubPrefs.WorktreePrefs(projectMaster)` abre `hub.NewManager("")` por llamada; `branch/ports.go:53-56` `Prefs` solo tiene `WorktreePrefs`.
- Lecturas disponibles para relevancia (`engine.go:40-58`): `AheadBehind`, `RevList`, `CommitFiles` (`gitcli/scope.go:54`, devuelve `CommitPaths{SHA,Parents,Paths}`; las fusiones no llevan rutas), `DirtyPaths` (`ops.go:147`), `IsAncestor`, `BranchExists`, `Worktrees`.
- Roles: `branch/planner.go:480` `repositoryRole`: en monorepo todo es `RepoChangeBranch`; en multirepo el planning home es `RepoSpecsInPlace` y el resto `RepoCode`. «Repo de specs» = planning home. Rama a avanzar: `Policy.IntegrationBranch` (`branch/policy.go:17-18`); `MainBranch` (`:23`) nunca se avanza (C-02-a ya integrado).

### Hub y CLI
- `hub/types.go:10-24` `WorkspaceRecord` con `WorktreeMode`/`WorktreeRoot` (`omitempty`, esquema `1.0` intacto); `hub/prefs.go:59` `LookupWorktreePrefs`, `:90` `SetWorktreePrefs`, `:37` `ValidateWorktreePrefs`. Los valores están por proyecto (entrada del registro), no globales.
- Quien escribe prefs hoy: `cmd/axiom/main.go:1926` (`project add`/init), `hub/init.go:314`, `dashboard/service.go:895` y `dashboard/worktree_prefs.go:103` (`GET|PUT /api/projects/active/worktree-prefs`, `dashboard/server.go:81`). No existe CLI para editar prefs ya registradas.
- `cli/branch.go:27-37` `branchHelp` lista cinco verbos; `:211-228` `switch` de `runBranch`. El mensaje de error `:226` y la ayuda enumeran los verbos. RF-BRA-01 (spec `:134`) dice «exactamente los subcomandos `ensure`, `status`, `scope`, `done` y `guard`» y hay un escenario «La ayuda lista los cinco verbos»: añadir `autofetch` ahí obliga a modificar RF-BRA-01.
- `cmd/axiom/main.go:255` `case "project"`, `:282` `case "branch"`, `:368` `sdd-preflight-hook`, `:378` `skill-registry`: patrón de verbos de hook de primer nivel.

### Hooks SessionStart (realidad tras C-01-c)
- RDD ya está retirado: no hay `review_stop_hook.go` ni `axiom review stop-hook` en `inject.go` (solo lo limpia `uninstall/service.go:1495`).
- `components/sdd/inject.go:1402-1446` `installSkillRegistryAutomation`: Codex (`ensureCodexSkillRegistryHook` `:1448`, ya usa `SessionStart` en `:1470-1489`, más `ensureCodexBranchGuardHook` `:1859`) y Claude (`ensureClaudeSkillRegistryHook` `:1545`, evento `UserPromptSubmit`; `ensureClaudeSDDPreflightHook` `:1702`, `PreToolUse(Agent)`; `ensureClaudeBranchGuardHook` `:1852`, `PreToolUse/PostToolUse`). VS Code Copilot: `VSCodeBranchGuardHooksJSON` `:1818` (fichero propio `.github/hooks/axiom-branch-guard.json`). **Claude no tiene hoy ningún `SessionStart`**; Kiro, OpenCode (no por este camino), Gemini, Cursor, Antigravity no instalan hooks. La afirmación del backlog «Codex y Kiro» es inexacta.
- Patrón a copiar: `ensureBranchGuardHookEntries` (`:1867-1920`; identidad = comando exacto, idempotente, `claudeHookListContains` `:1941`, escritura con `filemerge.WriteFileAtomic`). Desinstalación: `uninstall/service.go:1472` ya recorre `SessionStart`; hay que añadir el comando nuevo a la lista de `:1493-1499`.
- Guarda de referencia de runtime de hook: `cli/branch_guard.go:52-84` + `engine.RunGuard` (stdin JSON del runtime, `--agent`, salida `additionalContext`). El SessionStart de Claude acepta stdout como contexto; el aviso debe salir ahí. Código de salida 2 bloquea en Claude/Codex: el hook debe terminar siempre en 0 (fail-open), como pide el comentario `:1770-1773`.

### Doctrina de fetch
- Regla: `components/agentguidance/routing.go:144`; `assets/skills/_shared/sdd-orchestrator-sections.md:66`; `assets/skills/sdd-apply/SKILL.md:111`; `assets/generic/stop-exits.md:7` («(or `axiom branch ensure --fetch` with the user's authorization)»). `sdd-orchestrator-sections.md:108` (verificación conjunta: «fetch que la persona debe autorizar», `scope` nunca hace fetch) no cambia.
- Tests que fijan la frase exacta: `assets/assets_test.go:1096,1103`; `agentguidance/routing_test.go:119`; `agentguidance/stop_exits_test.go:16`. `internal/workspace/docs_contract_test.go` NO contiene «fetch» (solo helpers `readManualDeInicio`/`readRepoFile`; los pins de AGENTS.md y manual están en otros `_test.go` de ese paquete: revisar con `Grep "autorización" internal/workspace/*_test.go` antes de tocar el texto).
- Goldens con la frase (10, no 13): `testdata/golden/` `combined-claude-claudemd`, `sdd-claude-claudemd`, `sdd-codex-agentsmd`, `sdd-codex-agentsmd-powerful`, `sdd-codex-agentsmd-lowcost`, `sdd-gemini-geminimd`, `sdd-cursor-rules`, `sdd-antigravity-rulesmd`, `sdd-vscode-instructions`, `sdd-opencode-multi-settings` (todos `.golden`).
- Docs: `AGENTS.md:17` (sincronización al inicio: `git fetch` solo con autorización, `git pull` con autorización) y `:20` (`--fetch` solo con autorización); `GEMINI.md:48` (punto 2 «Sincronización al Inicio»); `docs/manual-de-inicio.md:227` (`ensure ... [--fetch]`, «no usa la red salvo con `--fetch`»), `:229` (scope sin `--fetch`), `:238` (otra persona: `git fetch` o `ensure --fetch` con autorización), `:247` (`--archive` sugiere `--fetch` «solo con tu autorización»). Las líneas del backlog (226/236/243) están desplazadas.

### Spec viva
- `change-branch-lifecycle/spec.md`: RF-BRA-10 `:537` (título «Base sin Red y `--fetch` Explícito»; «El `--fetch` sigue exigiendo la autorización de la persona»); RF-BRA-01 `:134`; tabla `:97` y «No cubre» `:118`; matices en RF-BRA-13 (`:701`, `change_not_integrated` «sin fetch implícito»), RF-BRA-16 `status` solo local (`:843-845`), `done --fetch` (`:896`), adopción RF-BRA-27 (`:1531-1561`), RF-BRA-35 `scope` sin red (`:2054`, se mantiene). Siguiente ID libre: **RF-BRA-39** (último RF-BRA-38 en `:2168`).
- `multi-project-hub/spec.md`: RF-HUB-07 `:218` (preferencias personales por proyecto; «worktree»), RF-HUB-08 `:267` (se piden al registrar y se editan desde el dashboard). Siguiente libre: **RF-HUB-13** (último RF-HUB-12 `:352`).
- `sdd-block-review-gates/spec.md:410` (`gate record` no ejecuta fetch; salida: «tras `git fetch` (o `axiom branch ensure --fetch` con su autorización)»), `:420`, `:433` (observador sin fetch), `:961-962`. `:410` sigue siendo válido (gate record nunca toca red) pero su redacción menciona la autorización: solo cambia si se reescribe `stop-exits.md`.
- `local-web-dashboard/spec.md:205` (RF-HUB-07 citado); C-06-e depende de esta preferencia.

## 2. Preguntas del alcance y evidencia

- Topología: monorepo = un repo, el checkout principal está en la integración (`CLAUDE.md` local); las sesiones suelen estar en un worktree de rama, así que el ff debe ejecutarse con `dir` = checkout principal (`Worktrees()[0].Path`, como `ensure.go:296`). Multirepo: el repo de specs trabaja in situ en la integración (`specs_repo_not_on_integration` si no).
- Relevancia con primitivas existentes: tras el fetch, `CommitFiles(dir, "<integración>..origin/<integración>")` da commits entrantes con rutas (filtrar por prefijos `openspec/changes/<cambio>/`, `openspec/specs/<capacidad afectada>/`, `odd/tasks/<feature>.md`); rama del cambio: `AheadBehind(rama, origin/<rama>)` y `RevList`. Las fusiones aparecen sin rutas: sus padres se listan aparte, bastan para el filtro. Las «specs afectadas» salen de las carpetas `specs/<capacidad>/` del delta del cambio (`TreeDirs` en el árbol del cambio).
- Intervalo mínimo entre fetches: no existe estado. Opciones: (a) mtime de `<commonDir>/FETCH_HEAD` (cero estado nuevo, cuenta también los fetch manuales; `gitcommon` ya lee ficheros del common dir sin lanzar git); (b) fichero propio bajo `<git common dir>/axiom/autofetch/v1/` (como el registro `branch/registry`); (c) campo en el hub (descartado: escritura del hub por cada arranque, bloqueos y mezcla estado/preferencia).

## 3. Enfoques

| # | Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|---|
| A | Nuevo paquete `internal/autofetch` (política + relevancia + ff + aviso) que usa un puerto git mínimo; `branch/engine` solo consume el booleano para el fetch implícito de `ensure` | Aísla el aviso y el ff; no engorda `engine`; fácil de probar con dobles | Hay que decidir su relación con RF-BRA-01 (la prueba de arquitectura cubre `internal/branch/**`); duplica `Fetch`/`DirtyPaths` si no reutiliza `gitcli` | M |
| B | Todo en `internal/branch/engine` (`Autofetch(ctx, cwd)` + `FastForward` en `gitWriter`) | Reutiliza `gitReader`/`gitWriter`, `Policy`, roles, planner y guardas existentes; una sola ruta de git | `engine` ya es grande; RF-BRA-01 y la prueba de arquitectura quedan afectadas | M |
| C (recomendado) | Híbrido: primitiva `FastForward` en `gitcli` + método de motor `Autofetch` (fetch, avance, relevancia) en `engine`, expuesto por un verbo propio fuera de la familia `branch` | Coherente con la arquitectura (un único punto de git); RF-BRA-01 se matiza en lugar de ampliar la familia | Verbo nuevo de primer nivel | M |

### Nombre del CLI
- Opción 1: `axiom branch autofetch enable|disable|status|run`. Contra: obliga a modificar RF-BRA-01 («exactamente cinco subcomandos»), `branchHelp`, el mensaje de `:226`, el escenario de ayuda y las pruebas.
- Opción 2 (recomendada): `axiom autofetch enable|disable|status|run [--cwd] [--agent] [--json]`. El hook ejecuta `axiom autofetch run --agent <runtime>`. Respeta RF-BRA-01 sin ampliar la familia y la preferencia es del hub, no de las ramas. Coste: un `case` en `cmd/axiom/main.go`.
- Opción 3: `axiom project autofetch ...` (junto a `project add`); correcta por semántica del hub, pero `project` hoy es de registro.

### Hub
- Campo nuevo `Autofetch bool` (`json:"autofetch,omitempty"`) en `WorkspaceRecord` (apagado por defecto; esquema `1.0` intacto), con `LookupAutofetch(projectPath)` y `SetAutofetch(idOrPath, bool)` en `hub/prefs.go`; puerto `branch.Prefs` ampliado con un método (`Autofetch(projectMaster) (enabled bool, warning string)`). Dashboard: `GET|PUT /api/projects/active/worktree-prefs` o un endpoint hermano para C-06-e (no es de este cambio, pero el DTO debe permitirlo).
- Proyecto no registrado o hub ilegible ⇒ apagado (fail-closed) con aviso: nunca red por defecto.

### Triggers
- SessionStart: Claude (`settings.json`, `SessionStart`, matcher vacío o `startup|resume`) y Codex (`hooks.json`, `SessionStart`); VS Code Copilot es opcional vía el fichero JSON existente. Para el resto de agentes, solo doctrina (el agente ejecuta `axiom autofetch run`).
- Antes de `ensure`: `fetch := r.Fetch || facts.prefs.autofetch`; con el interruptor, un fallo debería degradar a aviso (`warnings`) y no a `fetch_failed`, para no bloquear con red caída (decisión de diseño; el `--fetch` explícito conserva el rechazo).
- Antes de cada parada: doctrina que ejecuta `axiom autofetch run --quiet` (respeta intervalo).
- Timeout ~8 s con `RunWithTimeout`; intervalo mínimo por defecto 5-10 min (valor a fijar en diseño).

## 4. Requisitos a tocar

MODIFIED:
- RF-BRA-10 `change-branch-lifecycle`: «ninguna orden hace red sin opción explícita» ⇒ la preferencia persistente cuenta como opción explícita; con interruptor apagado, comportamiento idéntico; `fetched` sigue indicando el fetch real.
- RF-BRA-01: «salvo la solicitada por `--fetch`» ⇒ también la autorizada por el interruptor; nueva escritura `merge --ff-only` (no es `push`/`commit`); mantener la lista de cinco subcomandos si se elige la opción 2.
- Tabla `:97`, «No cubre» `:118` y, si procede, RF-BRA-13 `:701` (la sugerencia de `--fetch` en `change_not_integrated`), RF-BRA-16 `:843-845` y `done --fetch` (decidir si `done` también usa el interruptor; recomendado NO: fuera de alcance).
- RF-HUB-07 `multi-project-hub`: nueva preferencia personal `autofetch` (apagada por defecto), nunca en `axiom.yaml`; escenarios de persistencia/default/re-registro/desregistro. Hacer constar que el binario anterior la ignora.
- `sdd-block-review-gates:410/420`: solo si cambia la redacción de `stop-exits.md`.

ADDED (recomendado): **RF-BRA-39** «Autofetch del repo de specs con el interruptor personal encendido» (fetch con timeout y intervalo, avance ff estricto de `integration_branch` solo con árbol limpio y sin divergencia, jamás de `main_branch` ni de ramas de cambio ni de repos de código, degradación a aviso con la orden exacta, aviso solo de lo relevante, remoto inalcanzable no bloquea, hook fail-open). **RF-HUB-13** opcional si el interruptor tiene requisito propio en el hub (alternativa: solo MODIFIED de RF-HUB-07).

## 5. Riesgos

- `merge --ff-only` en el checkout principal puede chocar con `scripts/axiom-worktree.ps1 done`, que también avanza `main`; es idempotente si ya está al día, pero comprobar `index.lock`/carrera. Con `GIT_OPTIONAL_LOCKS=0` no basta para escrituras.
- Avance con la rama de integración desplegada en un worktree distinto del checkout principal: ejecutar con `dir` del worktree que la tiene (el de `Worktrees()` con `Branch == integración`); si no está desplegada en ninguno, no avanzar con `merge` (alternativa: `fetch origin <int>:<int>`, que ya es ff-only, pero cambia la política «solo merge --ff-only»).
- Sucio: usar `DirtyPaths` (`--untracked-files=normal`) y degradar a aviso; en multirepo el árbol sucio es frecuente (documentos directos en la integración).
- Credenciales/red: `GIT_TERMINAL_PROMPT=0` ya está; añadir `GIT_SSH_COMMAND`/`BatchMode` no es seguro de forzar (no tocar el entorno del usuario). Un proceso colgado por credenciales se corta por el timeout.
- Hook SessionStart nuevo cambia `settings.json` del usuario (Claude) y la confianza por hash en Codex (comentario `:1766-1768`: el comando es la identidad; no cambiarlo después). Un `axiom` antiguo sin el verbo devolvería 2 ⇒ el comando del hook debe terminar siempre en 0 (`; exit 0` / `|| true`, patrón de `VSCodeBranchGuardHooksJSON`).
- El fetch trae también ramas de otros cambios: `refs/remotes/origin/*` crece; sin `--prune` los avisos de «rama ya no está» quedan obsoletos (aceptado por el backlog).
- Ambigüedad residual de P14(b): ¿el fetch previo a `ensure` aplica solo con `ensure` o también a `ensure --archive` y `done`? Recomendado: `ensure` y `ensure --archive` (ya comparten `checkRepository`), no `done`.
- Los marcadores «no tocar sin autorización» de las reglas de agente (`AGENTS.md:17,20`, `GEMINI.md:48`, manual, routing, stop-exits, sdd-apply) deben decir lo mismo: riesgo de deriva entre 6 textos y 10 goldens.
- Estado ya integrado: C-01-c (RDD retirado) y C-02-a (`main_branch`/`promotion`) están en `main`; C-05-a también (según `AGENTS.md:17` ya en «solo con autorización»). No hay dependencias abiertas.

## 6. Tests y goldens afectados (lista exacta)

Pruebas a modificar/ampliar: `assets/assets_test.go:1096,1103`; `agentguidance/routing_test.go:119` (y `:90`); `agentguidance/stop_exits_test.go:16` (si cambia `stop-exits.md`); `branch/gitcli/ops_write_test.go` (nueva prueba de `FastForward`; `remoteOperationFor` en `:16`); `engine/ensure_fetch_test.go` (fetch implícito con interruptor, y que sin él no hay red: `:138`); `engine/ensure_test.go:422`; pruebas de hub (`hub/prefs_test.go` o equivalente) para el campo nuevo; `sddpublish/isolation_test.go:241` no cambia pero debe seguir verde; `gitcommon/architecture_test.go:106` idem; pruebas de `inject.go`/`uninstall/service.go` para el hook; pruebas de docs de `internal/workspace` que fijen AGENTS.md y manual (localizar con `Grep`).
Goldens (regenerar): los 10 de §1 «Doctrina de fetch».
Ficheros de texto: `routing.go:144`, `sdd-orchestrator-sections.md:66`, `sdd-apply/SKILL.md:111`, `assets/generic/stop-exits.md:7`, `AGENTS.md:17,20`, `GEMINI.md:48`, `docs/manual-de-inicio.md:227,229,238,247`.
