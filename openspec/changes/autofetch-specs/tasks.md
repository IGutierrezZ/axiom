# Tareas: Autofetch del repositorio de specs — avisar y avanzar (C-05-d)

> Rol único `fullstack`. TDD en modo estándar: las pruebas van en el mismo commit que el comportamiento. Una tarea = una unidad de trabajo = un commit (Conventional Commit, sin atribución de IA). Solo pruebas localizadas con `-run`; nunca la suite entera ni binarios sueltos (Cortex XDR los borra). En `internal/cli` las pruebas git comparten el HOME de `TestMain`: sin `t.Parallel`, sin escribir en `GIT_CONFIG_GLOBAL`, sin suponer un hub vacío (se inyecta la ruta del hub).

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | XL |
| Estimated changed lines | 3.200-3.600 escritas (sin goldens regenerados) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| T1 | Preferencia `autofetch` en el hub y puerto `Prefs.Autofetch` | `feat(hub): preferencia autofetch por proyecto` | `go test ./internal/hub -run Autofetch` y `go test ./internal/branch/engine -run Prefs` | N/A: capa de datos, se prueba en proceso con hub temporal | `hub/types.go`, `hub/prefs.go`, `engine/prefs.go`, `ports.go` (Prefs) |
| T2 | Primitivas `FetchWithin`, `FastForward`, `RemoteBranches` | `feat(branch): primitivas de fetch acotado y avance ff-only` | `go test ./internal/branch/gitcli -run "FetchWithin\|FastForward\|RemoteBranches"` | git real con remoto `file://` en `t.TempDir()` | `gitcli/ops.go` y ampliación de puertos |
| T3 | Fetch implícito en `ensure` y `ensure --archive` | `feat(branch): fetch implicito en ensure con autofetch encendido` | `go test ./internal/branch/engine -run "EnsureAutofetch\|Archive"` | repos reales con remoto desnudo local | `engine/ensure.go`, `archive.go`, `archive_refusal.go`, `result.go` |
| T4 | Ciclo `Engine.Autofetch`: resultado tipado, identidad por ficheros, sello, intervalo, fetch y presupuesto | `feat(branch): ciclo autofetch con sello, intervalo y presupuesto` | `go test ./internal/branch/engine -run "Autofetch(Cycle\|Stamp\|Project\|Budget)"` | repos reales + `Observe` | `branch/autofetch.go`, `engine/autofetch.go`, `autofetch_stamp.go` |
| T5 | Relevancia: cambio actual (4 reglas), avisos (a) y (b) | `feat(branch): avisos de relevancia del autofetch` | `go test ./internal/branch/engine -run AutofetchRelevance` | repos reales con commits entrantes | `engine/autofetch_relevance.go` |
| T6 | Avance fast-forward estricto y tabla D-08 | `feat(branch): avance ff-only de la integracion con avisos de no avance` | `go test ./internal/branch/engine -run AutofetchAdvance` | git real: `rebase`/`merge` en curso, `switch --detach` | `engine/autofetch_advance.go` |
| T7 | Verbo `axiom autofetch` (enable/disable/status/run, modo hook) | `feat(cli): verbo axiom autofetch` | `go test ./internal/cli -run Autofetch` | `cli.RunAutofetch` en proceso con `DefaultWithHubPath` | `cli/autofetch.go`, `cmd/axiom/main.go` |
| T8 | Hook `SessionStart`, poda por `sync` y retirada | `feat(sdd): hook SessionStart de autofetch con poda y desinstalacion` | `go test ./internal/components/sdd -run Autofetch` y `go test ./internal/components/uninstall -run Autofetch` | ficheros temporales de configuración | `inject.go`, `uninstall/service.go` |
| T9 | Doctrina y goldens | `docs(sdd): doctrina de autofetch y goldens regenerados` | `go test ./internal/components -run TestGolden`; `go test ./internal/assets -run "Fetch\|Orchestrator"`; `go test ./internal/components/agentguidance -run Routing` | regeneración con `-update` y revisión del diff | frase canónica y párrafo en 3 textos + goldens |
| T10 | Documentación de usuario y guardas de arquitectura | `docs: documentar axiom autofetch en AGENTS, GEMINI y manual` | `go test ./internal/workspace -run "Agents\|Manual"`; `go test ./internal/gitcommon -run TestArchitecture`; `go test ./internal/sddpublish -run TestIsolation` | N/A: documentación y guardas | `AGENTS.md`, `GEMINI.md`, `docs/manual-de-inicio.md` |

Orden y dependencias: T1 y T2 independientes; T3 depende de T1+T2; T4 depende de T1+T2; T5 y T6 dependen de T4; T7 depende de T6; T8 independiente del motor; T9 y T10 después de T7. No hay tarea de verificación del rol: la skill no la pide (la ejecuta `sdd-verify`).

## Fase 1: Fundamentos

- [x] T1 Preferencia `autofetch` en el hub (~260 líneas)
  - Ficheros: `internal/hub/types.go` (campo `Autofetch bool \`json:"autofetch,omitempty"\``), `internal/hub/prefs.go` (`LookupAutofetch`, `SetAutofetch` sin escritura si no cambia, `ErrProjectNotRegistered`, búsqueda con `sameProject`), `internal/branch/ports.go` (`Prefs.Autofetch(projectMaster) (bool, AutofetchPrefState, string)`), `internal/branch/engine/prefs.go` (`hubPrefs` con `Autofetch`, `DefaultWithHubPath`), `internal/branch/engine/ensure_mode_test.go` (doble de `Prefs`), `internal/hub/prefs_autofetch_test.go` (crear).
  - Cubre: RF-HUB-07, RF-HUB-13 (parte hub), huecos 8 y 12 (D-03).
  - Pruebas: apagado por defecto y clave ausente; persistencia; idempotencia de `SetAutofetch` (bytes de `workspaces.json` iguales); sobrevive a `Register` y al dashboard, desaparece con `Unregister`; hub ilegible no escribe; no registrado devuelve `errors.Is(err, hub.ErrProjectNotRegistered)`; binario anterior (unmarshal en struct sin el campo); `axiom.yaml` con clave `autofetch` ignorada; en Windows `C:\Repos\X` se encuentra desde `c:\repos\x`; los cuatro `state` (`enabled|disabled|unregistered|unreadable`) con aviso de `axiom project add "<maestra>"`.
  - Verificación: `go test ./internal/hub -run Autofetch`; `go test ./internal/branch/engine -run Prefs`.
  - Evidencia: `go test ./internal/hub -run Autofetch`: ok; `go test ./internal/branch/engine -run Prefs`: ok; `go vet ./internal/hub ./internal/branch/... ./internal/cli ./internal/dashboard`: sin salida; `gofmt -l internal/hub internal/branch`: vacío. Ruta: inline.

- [x] T2 Primitivas de git (~220 líneas)
  - Ficheros: `internal/branch/gitcli/ops.go` (`FetchWithin` sobre `gitcommon.RunWithTimeout` sin `--prune`, envolviendo `gitcommon.ErrTimeout`; `FastForward` = `merge --ff-only` con `plainArgument`/`runWrite`; `RemoteBranches` = `for-each-ref refs/remotes/origin/<prefix>/`, sin red), `internal/branch/ports.go` (`Git`), `internal/branch/engine/engine.go` (`gitReader`, `gitWriter`, campo `now`, comentarios de operaciones remotas), `internal/branch/gitcli/ops_write_test.go`.
  - Cubre: RF-BRA-01 (único punto de ejecución), base de RF-BRA-10 y RF-BRA-39.
  - Pruebas: argumentos exactos vía `gitcommon.Observe` (sin `--prune`, sin refspec); `ErrTimeout` con remoto lento/inexistente; `FastForward` rechaza la divergencia sin crear fusión; `RemoteBranches` no es subcomando remoto (sin red).
  - Verificación: `go test ./internal/branch/gitcli -run "FetchWithin|FastForward|RemoteBranches"`.
  - Evidencia: `go test ./internal/branch/gitcli -run "FetchWithin|FastForward|RemoteBranches"`: ok; `go test ./internal/gitcommon -run TestArchitecture`: ok; `go test ./internal/sddpublish -run TestIsolation`: ok; `go test ./internal/branch/engine -run "Prefs|EnsureReports"`: ok; `go vet ./internal/branch/...`: sin salida; `gofmt -l internal/branch`: vacío. Ruta: inline.

## Fase 2: Motor

- [x] T3 Fetch implícito en `ensure` y `ensure --archive` (~450 líneas; depende de T1, T2)
  - Ficheros: `internal/branch/engine/ensure.go` (`fetchMode` none/explicit/implicit, 8 s por repositorio en serie, `RemoteConfigured` antes de traer, repositorio de specs in situ en `checkPlan`, lectura de la preferencia tras `resolveMode`), `internal/branch/engine/archive.go`, `internal/branch/archive_refusal.go` (`ChangeNotIntegrated(..., autofetch bool)`), `internal/branch/result.go` (`RepositoryResult.Fetched *bool omitempty`), `internal/branch/engine/ensure_autofetch_test.go` (crear).
  - Cubre: RF-BRA-10, RF-BRA-13, RF-BRA-01 (`scope` sin red), RF-HUB-07 (ensure sin aviso nuevo).
  - Pruebas: fetch implícito en todos los repositorios del plan incluido specs in situ (`fetched:true`); código sin mover ramas; sin `origin` omitido sin aviso; `--base <ref>` con fetch; corte a 8 s con `autofetchFetchTimeout` reducido y doble que devuelve `ErrTimeout`; fallo con aviso `el autofetch no pudo traer origin…` y `fetched:false`; `--fetch` explícito con `fetch_failed` y límite 30 s; `decision_required` sin fetch; `done` y `scope` sin fetch; intervalo ignorado; `--archive` sin sugerir `--fetch` (ambos mensajes); proyecto no registrado/hub ilegible como apagado; salida idéntica byte a byte con el interruptor apagado; `prefs == nil` equivale a apagado.
  - Verificación: `go test ./internal/branch/engine -run "EnsureAutofetch|Archive|Ensure"`; `go test ./internal/branch -run ChangeNotIntegrated`.
  - Evidencia: `go test ./internal/branch/engine -run "EnsureAutofetch|Archive|Ensure"`: ok; `go test ./internal/branch -run "ChangeNotIntegrated|Archive"`: ok; `go vet ./internal/branch/... ./internal/cli`: sin salida; `gofmt -l internal/branch`: vacío. Ruta: inline.

- [x] T4 Ciclo `Engine.Autofetch` base (~450 líneas; depende de T1, T2)
  - Ficheros: `internal/branch/autofetch.go` (`AutofetchResult`, `AutofetchIntegration`, `AutofetchChange`, `AutofetchNotice`, `AutofetchResultSchema`, `WriteText`), `internal/branch/engine/autofetch.go` (`Autofetch`, `AutofetchProject` solo por ficheros: `IsAxiomProject`→`Locate`→`GitRootFromFiles`/`CommonDirFromFiles`/`MainWorktreeFromFiles`; presupuesto `autofetchRunBudget` 10 s con `recover`), `internal/branch/engine/autofetch_stamp.go` (`<commonDir>/axiom/autofetch/v1/state.json`, escritura atómica, constantes `autofetchInterval`/`autofetchFetchTimeout`/`autofetchAdvanceReserve` como variables de paquete, reloj `now`), `internal/branch/engine/autofetch_test.go` (crear).
  - Cubre: RF-BRA-39 (apagado sin red ni escritura, intervalo, remoto inalcanzable, multirepo solo specs, sin push/prune), RF-HUB-07.
  - Pruebas: apagado sin procesos git (`Observe` vacío); identidad desde worktree de feature enlazado (el fetch va al checkout principal y no toca el worktree), con `GIT_DIR` en el entorno, desde subdirectorio y desde repositorio de código con puntero `.axiom-workspace`; sello ausente/ilegible/futuro = sin fetch previo; intervalo omite la red y la repite pasado el tiempo; sello solo tras fetch correcto; fallo de escritura del sello = aviso; `no_origin`; `timeout` con doble `ErrTimeout`, exit 0, aviso y sin avance; `Observe` sin `push`, `pull`, `ls-remote` ni `--prune`; pánico simulado devuelto como resultado; proyecto no registrado/hub ilegible.
  - Verificación: `go test ./internal/branch/engine -run "AutofetchCycle|AutofetchStamp|AutofetchProject|AutofetchBudget"`.
  - Evidencia: `go test ./internal/branch/engine -run "AutofetchCycle|AutofetchStamp|AutofetchProject|AutofetchBudget"`: ok; `go test ./internal/branch -run Autofetch`: ok; `go test ./internal/gitcommon -run TestArchitecture`: ok; `go test ./internal/sddpublish -run TestIsolation`: ok; `go vet ./internal/branch/...`: sin salida; `gofmt -l internal/branch`: vacío. Ruta: inline. Puntos de extensión de T5 y T6 marcados en `autofetchCycle`.

- [x] T5 Relevancia y avisos (a)/(b) (~400 líneas; depende de T4)
  - Ficheros: `internal/branch/engine/autofetch_relevance.go`, ampliación de `autofetch_test.go` (o `autofetch_relevance_test.go`).
  - Cubre: RF-BRA-39 (relevancia: cambio, spec, ODD, nada relevante; rama del cambio nunca se mueve), huecos 5 y 9 parcial.
  - Pruebas: cambio actual por `HEAD`; único activo; cesión sin registro ni rama local (aviso `change_branch_not_adopted` con orden `axiom branch ensure --change <c> --cwd "<maestra>"`, tope 50 referencias); dos candidatas o varios activos sin coincidencia = sin (a) ni (b); aviso (a) calculado antes del avance con `CommitFiles` por prefijos SDD/ODD, fusiones ignoradas, orden `git log --oneline <sha previo>..origin/<int> -- <prefijos>`; (b) con `AheadBehind` y orden ff-only o log según divergencia; rama del cambio con SHA intacto; sin nada relevante = cero avisos; ordenación determinista; repositorios de código solo lecturas locales.
  - Verificación: `go test ./internal/branch/engine -run AutofetchRelevance`.
  - Evidencia: `go test ./internal/branch/engine -run AutofetchRelevance`: ok; `go test ./internal/branch/engine -run "AutofetchCycle|AutofetchStamp|AutofetchProject|AutofetchBudget"`: ok; `go vet ./internal/branch/...`: sin salida; `gofmt -l internal/branch`: vacío. Ruta: inline. La relevancia se calcula en `autofetchRelevance` antes del punto de avance; `autofetchCycle` añade los avisos (a), deja el hueco de T6 y añade los (b).

- [x] T6 Avance ff-only y tabla D-08 (~350 líneas; depende de T4)
  - Ficheros: `internal/branch/engine/autofetch_advance.go`, ampliación de las pruebas del motor.
  - Cubre: RF-BRA-39 (ff con árbol limpio, ya al día, sucio, divergencia, no desplegada, operación en curso, `HEAD` desacoplado, `main_branch` distinta o igual a la integración, sin presupuesto).
  - Pruebas por fila: 1 (sin `R`/sin `L`/igual), 2 (local por delante), 3 `integration_diverged`, 4 `integration_operation_in_progress` con `rebase` y `merge` con conflicto reales (índice y ficheros intactos) y `cherry-pick`, 5 `integration_detached` con `switch --detach`, 6 `integration_not_deployed` con `fetch . refs/remotes/origin/<int>:refs/heads/<int>`, 7 `integration_dirty` (preparado y sin seguimiento), 8 sin presupuesto (`autofetchAdvanceReserve`), 9 avance sin crear commits (`RevList` igual al remoto), 10 `integration_ff_failed` por `index.lock` sin reintento; integración al día con árbol sucio sin aviso; `main_branch` distinta intocable (SHA comparado, sin `merge`); iguales avanza; worktree enlazado con la integración desplegada.
  - Verificación: `go test ./internal/branch/engine -run AutofetchAdvance`.
- Evidencia: `go test ./internal/branch/engine -run AutofetchAdvance`: ok; `go test ./internal/branch/engine -run "AutofetchCycle|AutofetchStamp|AutofetchProject|AutofetchBudget|AutofetchRelevance"`: ok (aserciones de no avance de T4 y T5 relajadas a «solo merge --ff-only»); `go test ./internal/gitcommon -run TestArchitecture`: ok; `go test ./internal/sddpublish -run TestIsolation`: ok; `go vet ./internal/branch/...`: sin salida; `gofmt -l internal/branch`: vacío. Ruta: inline. Los avisos de no avance se insertan en `autofetchCycle` entre (a) y (b).

## Fase 3: Superficie

- [x] T7 Verbo `axiom autofetch` (~540 líneas; depende de T6)
  - Ficheros: `internal/cli/autofetch.go` (`RunAutofetch(args, stdin, stdout, stderr)`, ayuda, `--cwd` relativo = error de uso, `--agent`, `--json`, texto en castellano, modo hook con `hookSpecificOutput.additionalContext`, lectura de stdin con límite 256 KiB y fallo en abierto), `cmd/axiom/main.go` (`case "autofetch"` y línea de ayuda), `internal/cli/autofetch_test.go` (crear).
  - Cubre: RF-HUB-13, RF-HUB-07 (tabla D-03), RF-BRA-39 (salida texto/`--json` de `run`), RF-BRA-40 (modo hook, exit 0, inerte), RF-BRA-01 (`branch --help` sin `autofetch`).
  - Pruebas: tabla D-03 completa para `status`, `run`, `run --agent`, `enable` y `disable` con no registrado e ilegible (códigos 0/1/2); `enable` desde worktree de feature, subdirectorio y puntero `.axiom-workspace`; idempotencia (bytes del hub); `disable` sin registrar no crea el hub; salida `--json` (`axiom.autofetch-result/v1`, `axiom.autofetch-preference/v1`) y texto de `run`; hook sin avisos = salida vacía; con avisos = JSON `SessionStart`; fuera de proyecto Axiom y con interruptor apagado: salida vacía, 0 y `Observe` vacío; carga ilegible o > 256 KiB sin fallo; presupuesto del hook (`autofetchRunBudget` reducido, con remoto lento termina en 0 dentro de la cota); `enable` nunca escribe `axiom.yaml`. Sin `t.Parallel`; hub inyectado con `DefaultWithHubPath`.
  - Verificación: `go test ./internal/cli -run Autofetch` (siempre con `-run`).
  - Evidencia: `go test ./internal/cli -run Autofetch`: ok (13 pruebas, incluida la del presupuesto del hook con un remoto que no responde); `go test ./cmd/axiom -run "Autofetch|BranchVerbs"`: ok; `go test ./internal/branch/engine -run Autofetch`: ok; `go test ./internal/branch/engine -run "EnsureAutofetch|EnsureFetch"`: ok; `go test ./internal/gitcommon`: ok; `go test ./internal/sddpublish -run TestIsolation`: ok; `go vet ./internal/cli ./internal/branch/... ./internal/gitcommon ./cmd/axiom`: sin salida; `gofmt -l internal cmd`: vacío. Ruta: inline. Desviaciones anotadas: (1) `AutofetchProject` devuelve `Checkout` normalizado con `resolveDirectory`, de modo que las rutas de los avisos y de `repository` salen largas y coherentes en Windows (cierra el riesgo heredado de T6; se ajustaron dos expectativas de `autofetch_advance_test.go`); (2) `gitcommon.RunWithTimeout` gana `cmd.WaitDelay` (500 ms) porque un `git-remote-http` colgado mantenía abiertas las tuberías y el fetch superaba su tiempo máximo y el presupuesto de 10 s de RF-BRA-40; (3) `engine.SetAutofetchRunBudget` exportado solo para pruebas de la CLI y `Engine.AutofetchPreference` para `status`.

- [ ] T8 Hook `SessionStart`, poda por `sync` y retirada (~380 líneas; independiente del motor)
  - Ficheros: `internal/components/sdd/inject.go` (`AutofetchHookCommand(agent, goos)`, `ensureAutofetchHookEntries`, `pruneStaleAutofetchHooks`, llamadas en `installSkillRegistryAutomation` solo para Claude Code y Codex), `internal/components/uninstall/service.go` (retirar órdenes con `axiom autofetch run` en cualquier evento) y sus pruebas `internal/components/sdd/*_test.go`, `internal/components/uninstall/*_test.go`.
  - Cubre: RF-BRA-40 (instalación, idempotencia, otros runtimes, retirada por `sync` y desinstalación, `|| true`/`; exit 0`, `timeout` 10), RF-BRA-23 (coexistencia con `branch guard`).
  - Pruebas: órdenes por runtime y SO terminan en `|| true` o `; exit 0`; `timeout` 10 y matcher `startup|resume`; segundo `sync` byte a byte igual; hooks ajenos y `branch guard` intactos; poda: `UserPromptSubmit` con la orden vieja + otra forma bajo `SessionStart` + hook ajeno → solo queda la canónica; otros runtimes sin hook; retirada por `uninstall` conservando lo ajeno; formas inesperadas no se tocan.
  - Verificación: `go test ./internal/components/sdd -run Autofetch`; `go test ./internal/components/uninstall -run Autofetch`; `go test ./internal/components/sdd -run TestSyncNeverWritesOutsideSelectedAgents`.

## Fase 4: Doctrina, documentación y guardas

- [ ] T9 Doctrina y goldens (~170 líneas escritas + goldens; después de T7)
  - Ficheros: `internal/components/agentguidance/routing.go` (frase canónica), `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (frase y párrafo «Before presenting each stop, run `axiom autofetch run --cwd <root> --json`…»), `internal/assets/skills/sdd-apply/SKILL.md`, `internal/assets/assets_test.go`, `internal/components/agentguidance/routing_test.go`, `testdata/golden/*.golden` (10).
  - Cubre: RF-BRA-40 (doctrina condiciona `--fetch`); hueco 6 (las líneas 108 y 288 y `stop-exits.md` no se tocan; una prueba fija que siguen intactas).
  - Pasos: editar textos → `go test ./internal/components -run TestGolden -update` → revisar el diff de los 10 goldens → `go test ./internal/components -run TestGolden`.
  - Verificación: `go test ./internal/components -run TestGolden`; `go test ./internal/assets -run "Fetch|Orchestrator|Skill"`; `go test ./internal/components/agentguidance -run Routing`.

- [ ] T10 Documentación y guardas de arquitectura (~100 líneas)
  - Ficheros: `AGENTS.md` (líneas 17 y 20), `GEMINI.md` (línea 48), `docs/manual-de-inicio.md` (líneas 227, 238, 247 y nueva entrada `axiom autofetch`: subcomandos, hook, intervalo, prohibiciones).
  - Cubre: RF-BRA-40 (doctrina), RF-BRA-01 (guarda de único punto de ejecución).
  - Verificación: `go test ./internal/workspace -run "Agents|Manual|Guidance"` (pruebas que fijan AGENTS.md y el manual; si el patrón no casa, localizarlas con Grep y ajustarlo); `go test ./internal/gitcommon -run TestArchitecture`; `go test ./internal/sddpublish -run TestIsolation`.

## Criterios absorbidos de la revisión de cobertura

Huecos 1 (T8), 2 y 7 (T3), 3 (T4/T7: presupuesto 10 s), 4 (T6: regla previa), 5 (T5), 6 (T9), 8 (T1/T7), 9 (T6), 10 y 12 (T1/T3/T7), 11 (T6).

## Huecos de cobertura no absorbidos

- Límite documentado de D-07/R-4: con varios cambios activos y `HEAD` fuera de sus ramas, no hay avisos (a)/(b); un cambio ODD cedido sin registro no se detecta.
- R-5: el contrato `hookSpecificOutput` de `SessionStart` en Codex y su `timeout` se validan a mano; no hay prueba automatizada.
- R-3: `--fetch` explícito sigue sin traer el repositorio de specs in situ.
- R-10: con red caída `ensure` puede esperar hasta ~8·N s (cota por repositorio).
- Límite heredado de RF-BRA-23: un runtime que se deja de seleccionar solo se limpia con `axiom uninstall`.
