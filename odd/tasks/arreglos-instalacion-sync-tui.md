# ODD: Absorción upstream — arreglos de instalación, sync, TUI y Windows

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/arreglos-instalacion-sync-tui.md`.
> Espejo de recuperación en Engram: topic `odd/arreglos-instalacion-sync-tui/tasks`, proyecto `axiom`.
> Estado: **en implementación desde el 2026-11-05**: el usuario autorizó arrancar las tareas desbloqueadas (T1, T2, T3, T5, T6 y T9). T4 y T7 siguen bloqueadas.

## Objetivo

Absorber de upstream gentle-ai, revisado hasta `origin/main` 3ea08cdf (después de v4.3.1), los arreglos pequeños de instalación, sync, TUI y Windows que corrigen defectos presentes hoy en Axiom.

## Problema y porqué

Hoy Axiom tiene estos defectos, verificados contra `main` f7158af0:

- `axiom uninstall -h` y `axiom restore -h` devuelven `flag: help requested` sin texto de uso.
- `scripts/install.sh` sustituye el binario con un `cp` directo, sin escritura atómica.
- Un `variant` vacío pisa el que configuró el usuario en OpenCode.
- La TUI ejecuta `exec.LookPath` en cada frame.
- Sync, desinstalación y TUI leen, modifican y escriben el estado sin cerrojo.
- El revisor de RDD ignora el esfuerzo asignado, y Sonnet no ofrece `xhigh`.
- El sync aplica los componentes en un orden distinto del de install.
- GGA:
  - reescribe el proveedor configurado;
  - aborta en macOS sin Homebrew;
  - no reconoce los layouts `ucrt64` y `clang64` de Git for Windows.
- `brew` hereda el auto-update.
- Los instaladores no ponen el directorio en el PATH.
- `doctor` da por sana una herramienta solo porque está en el PATH.

## Fuera de alcance (decidido)

- **Agentes que el equipo no usa** (decisión del usuario, 2026-11-04; Engram `upstream/gentle-ai-review-2026-11/adaptadores-agentes`):
  - Pi: #4893, retirar `rpiv-ask-user-question`.
  - Codex: #5069, persona dentro de un marcador, y la parte de #5017 que reordena el TOML de Codex.
  - Kimi: #5063, `~/.kimi-code`.
- **Descartado:**
  - #5760 (4302006c): solo toca Gentle Shell/`shellinstaller`, que Axiom no tiene.
  - #5008 (6ee5df7e): el defecto no existe. `removeManagedAgentEntries` ya poda por `__managed_by` (`internal/components/uninstall/service.go:1236`, y Kilo en `global_agents.go:123-136`).
  - Cualquier parte de telemetría (#4838, #5012).
- **Aplazado:** #3637 (31fe20f1), pantalla TUI para gestionar agentes personalizados. Es una funcionalidad nueva, no un arreglo: unas 900 líneas, borra skills en varios agentes y choca con #5868.
- **Coordinación con otros bloques:**
  - #5625/#5632 (PATH de Windows con `REG_EXPAND_SZ`) es del bloque seguridad y datos. T7 debe respetar ese mismo criterio en `install.ps1`.
  - #5639 se asigna a este bloque; el de adaptadores lo excluye.
- **Prohibido:** tocar `docs/upstream-absorption-ledger.md` (decisión D4).

## Restricciones

- Se porta la intención, no se copian diffs a ciegas. Axiom conserva SDD/OpenSpec y RDD opcional apagado por defecto.
- Sin telemetría. Marca propia: binario `axiom`, marcadores `<!-- axiom:`.
- Módulo `github.com/IGutierrezZ/axiom/v3`: hay que reescribir los imports de `/v4`.
- No se compilan binarios sueltos, porque el antivirus borra `axiom.exe`. La verificación se hace con `go test` en proceso.
- Worktree `C:\repos\axiom-wt\arreglos-instalacion-sync-tui\axiom`, rama `inc/arreglos-instalacion-sync-tui`.

## Veredictos verificados

| Upstream (sha) | Evidencia en Axiom | Veredicto | Líneas |
|---|---|---|---|
| #4881 (05626be3) | `internal/cli/uninstall.go:35-58` y `restore.go:57-75`: `ContinueOnError` con salida descartada; `hasHelpFlag` (`internal/app/app.go:377`) no se usa en esas rutas | ADAPTAR (solo uninstall y restore) | ~120 |
| #3629 (b2ed7201) | `scripts/install.sh:374-379` (tras P1): `cp`/`chmod` y `sudo cp` directos | ADOPTAR | ~40 |
| #5639 (2a3129aa) | `internal/model/claude_model.go:76`: Sonnet sin `ClaudeEffortXHigh` | ADOPTAR | ~35 |
| #5360 (eb81278b) | `internal/reviewerprovider/claude_adapter.go:18,67`: solo pasa `--model`; `internal/cli/review_provider_runtime.go:75-108`: solo devuelve el modelo | ADAPTAR (el estado se lee con `EffectiveStateStore`) | ~140 |
| #5868 (af917f92) | `internal/tui/model.go:5679-5698` (`detectAgentBuilderEngines`) llamado desde `View` (`:1547`) y en cada pulsación (`:4466`) | ADOPTAR | ~100 |
| #4838 (79cd0099) | `internal/components/uninstall/service.go:2264-2295`, `internal/tui/model.go:3898-3903` y `internal/cli/sync.go:1739` hacen read-modify-write sin cerrojo; `statecoord.WithLock` ya existe (`internal/statecoord/statecoord.go:43`) | ADAPTAR (sin telemetría) | ~200 |
| #4078 (6c31484a) | `internal/components/sdd/inject.go:3926-3929`: escribe `"variant": assignment.Effort` aunque esté vacío | ADOPTAR | ~70 |
| #3906 (f0782af2) | Sin `LastSyncedAt`; upstream tampoco tiene consumidor (`internal/state/state.go:161` en upstream) | DESCARTAR (D-3) | 0 |
| #5017 (44be4761) | `internal/cli/sync.go:622` recorre `r.selection.Components` sin ordenar; `internal/cli/run.go:840-841` instala CodeGraph antes que la persona | ADAPTAR (sin la parte de Codex) | ~180 |
| #5012 (ed90f1e2) | `internal/installcmd/resolver.go:224,286,436`: `brew install` sin `HOMEBREW_NO_AUTO_UPDATE` ni `HOMEBREW_NO_INSTALL_CLEANUP` | ADAPTAR (solo `commandEnv`) | ~60 |
| #5724 (7d31db09) | `internal/cli/run.go:1903-1913` e `installcmd/resolver.go:283-287`: `brew tap/reinstall gga` sin comprobar que existe `brew` | ADAPTAR | ~150 |
| #5783 (a6942c7d) | `internal/assets/gga/gga.ps1:8`: solo acepta `mingw64` | ADOPTAR | 2 |
| #5626 (8f3b4c99) | `internal/components/gga/config.go:132-159`: `Inject` recalcula el proveedor (`config.go:70`) y reescribe sin leer el existente | ADAPTAR (solo la preservación) | ~250 |
| #5759 (d4e64a94) | Tras P1: `scripts/install.sh:219-221,386` e `install.ps1:120-123` solo avisan | ADAPTAR (decisión D-2) | ~250 |
| #5359 (46837ec8) | `internal/cli/doctor.go:173` (`checkOneTool`) solo resuelve el PATH | ADAPTAR acotado (D-4), se entrega en P3 | ~200 |
| #4173 (61125bbe) | `internal/assets/claude/persona-axiom.md:4` y `persona-gentleman.md:4`, con sus goldens | ADOPTAR (D-1) | ~6 + goldens |

**Dependencias:**
- #5639 va antes que #5360.
- #5012 va antes que #5724, y este antes que #5626: los tres tocan el paso GGA/brew de `run.go`.
- #5868 choca con #3637, si algún día se retoma.

## Decisiones del usuario

- **D-1 (#4173) — RESUELTA (2026-11-04): se retira el mandato.** La persona instalada usa las herramientas nativas del agente y, en shell, las disponibles en la plataforma. La regla personal del usuario sigue en su `CLAUDE.md`.
- **D-2 (#5759) — RESUELTA (2026-11-04): se añade al PATH.**
  - `install.ps1` lo añade al PATH de usuario y conserva `REG_EXPAND_SZ`.
  - `install.sh` añade al rc de la shell detectada una línea idempotente y marcada.
- **D-3 (#3906) — RESUELTA (2026-11-04): se descarta.** Ni upstream ni Axiom leen `last_synced_at`, y un campo sin lector es estado muerto. Se retoma si algún día lo necesitan `doctor` o la TUI.
- **D-4 (#5359) — RESUELTA (2026-11-04): sonda acotada.**
  - Se ejecuta `--version` con `context.WithTimeout` y `exec.CommandContext`, sin Job Object ni `NtResumeProcess`.
  - Riesgo aceptado: en Windows puede quedar vivo un proceso nieto colgado.

## Tareas

Cada tarea es una rebanada de PR de unas 400 líneas como máximo y cierra con un commit de unidad de trabajo que incluye sus tests.

- [x] **T1 · `--help` en la CLI** — #4881 (~120)
  - `uninstall -h` y `restore -h` imprimen el uso y salen con 0.
  - Hecho: `dcc15ea4` (+266/−28; ruta: escritor delegado). `go test ./internal/cli/ -run 'TestHelpRequest|TestRestore|TestUnknownFlag|Uninstall'` ok; `go vet` y `gofmt -l` limpios. `app.go` no se toca: el flujo ya funciona sin él.
- [x] **T2 · Esfuerzo de modelos** — #5639 → #5360 (~175)
  - Sonnet acepta `xhigh`.
  - El revisor en proceso pasa `--effort` con el esfuerzo asignado, leído desde `EffectiveStateStore`.
  - Hecho: `e8af9f56` (+117/−54; ruta: escritor delegado). `go test ./internal/model/... ./internal/reviewerprovider/... ./internal/tui/screens/...` ok; `go vet` y `gofmt -l` limpios. Sin esfuerzo asignado no se pasa `--effort`.
  - Pendiente común a T1 y T2: `go test ./internal/cli/...` completo agota el timeout en esta máquina (503 tests ok, 0 fallos antes del corte; los lentos son del ciclo de revisión, no tocados). Repetir en CI o en una máquina descargada antes del PR.
- [x] **T3 · TUI y estado** — #5868 + #4838 (~300)
  - La detección de motores se cachea en `NewModel` y se refresca al pulsar «Crear agente».
  - Los read-modify-write del estado en sync, desinstalación y TUI pasan por `statecoord`.
  - Hecho: `b1a10670` y `536f4be0` (~350; ruta: escritor delegado). `b1a10670` incluye también `markPendingSyncUnderLock` de la TUI, porque `model.go` no se pudo partir sin `git add -p`. `go test ./internal/tui/... ./internal/statecoord/... ./internal/components/uninstall/...` ok; `go test ./internal/cli/ -run 'MigratePersisted|ApplyResolvedPersona|InstallStateLock'` ok; `go vet` y `gofmt -l` limpios. Cinco pruebas de regresión nuevas (no se observó el RED: stash y reset están vetados). Fuera del encargo, sin portar: el preflight descartado de `run.go` que upstream quita en #4838.
- [ ] **T4 · Sync** — #4078 + #5017 sin Codex (~250)
  - Un `variant` vacío conserva el del usuario.
  - El sync usa el orden de install, con CodeGraph después de la persona.
- [x] **T5 · GGA y Homebrew** — #5012 → #5724 + #5783 (~210)
  - Hecho: `cad99f6e` (brew sin auto-update, en `commandEnv` de `run.go`), `55b50699` (GGA pospuesto sin brew en macOS) y `eb558af6` (`ucrt64` y `clang64`). Ruta: escritor delegado. `go test ./internal/installcmd/... ./internal/assets/... ./internal/components/gga/...` ok; `go test ./internal/cli/ -run 'TestCommandEnv|TestRunInstallMacOS|TestGGA|PostInstallNote|ReadyAgent|FailedVerification|TestRepro4020'` ok; `go vet` y `gofmt -l` limpios.
  - Sin verificar en local: los casos nuevos de `TestPowerShellShimResolvesGitBash` necesitan `AXIOM_TEST_BUILD_BINARIES=1` (compilan un binario); quedan para CI.
  - Pendiente fuera de alcance (`internal/tui`, `internal/pipeline`): la pantalla final de la TUI puede contar GGA como instalado cuando se pospone; upstream lo resuelve con `DeferredComponents`. El aviso sí llega por `ManualActions`.
  - **Limitada a `internal/cli` e `internal/installcmd`.** No toca `internal/update` (`strategy.go`) hasta que P1 se fusione; véase «Coordinación con P1 y P3».
  - `brew` se ejecuta sin auto-update.
  - Sin `brew` en macOS, GGA queda pospuesto con aviso y no aborta.
  - `gga.ps1` acepta `ucrt64` y `clang64`.
- [ ] **T6 · GGA conserva el proveedor** — #5626 (~250)
  - Install y sync respetan el proveedor que ya hay en el config de GGA.
- [ ] **T7 · Instaladores: binario atómico y PATH** — #3629 + #5759, según D-2 (~290). P1 ya está fusionado (PR #220, v3.6.0) y el worktree está rebasado sobre él. **Entra después de seguridad T2**, para aplicar su mismo criterio de `REG_EXPAND_SZ` al escribir el PATH.
  - `install.sh` deja el binario en una ruta temporal, aplica `install -m 0755` y hace `mv` atómico, sin interpolar en `sudo`.
  - `install.ps1` añade el directorio al PATH de usuario y respeta `REG_EXPAND_SZ`, con el mismo criterio que #5625 del bloque seguridad.
  - `install.sh` añade al rc de la shell detectada (bash, zsh o fish) una línea idempotente con marcador `axiom`.
- [ ] ~~**T8 · `doctor` con sondas acotadas**~~ — **se traslada a P3 (doctor por proyecto).** #5359, con el alcance de D-4: `--version` con timeout mediante `exec.CommandContext`, sin código específico de Windows. Se entrega allí, porque P3 reestructura `doctor` con informe estructurado y parametrizado por raíz.
- [ ] **T9 · Persona de Claude** — #4173, aprobada en D-1 (~6 + goldens). Afecta a `persona-axiom.md:4`, `persona-gentleman.md:4` y sus goldens.

## Coordinación con P1 y P3

Fuentes: hoja de ruta `docs/audits/2026-11-03-vision-producto.md` (rama `inc/auditoria-vision-producto`) y Engram `odd/distribucion-un-comando/tasks`.

- **P1, distribución con un comando** (rama `inc/distribucion-un-comando`, 8 commits sin PR). El diff contra `origin/main` toca:
  - `scripts/install.sh`: quita Homebrew propio y `git`; el método por defecto pasa a `binary`.
  - `scripts/install.ps1`;
  - `internal/update/**`: el tap ajeno, `effectiveMethod` y `checkSingleTool`;
  - `cmd/axiom/main.go`, README y docs.
  - **No toca** `internal/installcmd`, `internal/cli` ni `doctor`. Las pruebas de `installcmd` solo se ejecutan allí.
- **Efecto en este bloque:**
  - T7 (#3629 + #5759) toca los dos instaladores: depende de P1 y se rebasa cuando se fusione.
  - #3629 sale de T1 para no bloquear el arreglo de `--help`.
  - T5 (#5012, #5724) no comparte ficheros con P1, porque `run.go` e `installcmd/resolver.go` no cambian allí. Upstream #5012 sí toca `internal/update/upgrade/strategy_test.go`, así que esa parte espera a P1.
- **P3, doctor por proyecto** (pendiente): reestructura `doctor` (G3.4 y G3.5). #5359 se entrega dentro de P3, con el alcance de D-4.
- **Actualización 2026-11-05:** P1 está fusionado (PR #220, merge fbea8965, release v3.6.0). Este worktree está rebasado sobre `origin/main` 7235e6b6 y T7 queda desbloqueada por esa parte.

## Orden de ejecución entre bloques

Fijado el 2026-11-05 por la sesión de revisión upstream a partir del cruce de ficheros. Bloques: INS (este bloque), seguridad, adaptadores, revisión, guía y carril SDD. **No se implementa nada sin autorización del usuario.**

| Tarea INS | Cuándo entra | Motivo (fichero compartido) |
|---|---|---|
| T1 (`--help`) y T2 (esfuerzo) | Ya, en paralelo | Sin solape |
| T3 (TUI y estado) | Antes de seguridad T5 (`WriteFileAtomic`), que se rebasa sobre ella | `uninstall/service.go:2264` y `cli/sync.go:1739` |
| T4 (sync) | Antes de seguridad T5 y **después de adaptadores T1** (confirmado el 2026-11-05; `odd/tasks/adaptadores-agentes-upstream.md:143`, commit 273a5329). T4 se rebasa sobre adaptadores T1 cuando esta se fusione | `sdd/inject.go:3926`, `cli/sync.go:622` y `cli/run.go:840`. Las entradas de agente de OpenCode coinciden: `variant` frente a la purga de `__managed_by` |
| T5 (GGA y brew) | Antes de adaptadores T5 (Copilot CLI) | `installcmd/resolver.go` |
| T6 (proveedor de GGA) | Después de T5 de este bloque | `cli/run.go`, paso GGA |
| T7 (instaladores) | Después de seguridad T2 | Mismo criterio de `REG_EXPAND_SZ` al escribir el PATH |
| T9 (personas y goldens) | Eslabón 1 de la cadena en serie de goldens del orquestador | Goldens compartidos |

**Cadena de goldens del orquestador:**

1. INS T9 y adaptadores T1.
2. Revisión T6.
3. Guía T1.
4. Guía T2 y T3.
5. Carril SDD.

T9 no se mezcla con los demás eslabones.

## Criterios de aceptación y comprobaciones

- Cada tarea añade tests que fallan antes del cambio y pasan después. Comandos:
  - `go test ./internal/<paquetes tocados>/...`;
  - antes de cada PR, `go test ./...`.
- Para `install.sh`, test de shell existente o nuevo si hay arnés; si no, comprobación con `bash -n`.
- **TDD:** desactivado (2026-11-05). `axiom.yaml` no declara modo TDD ni hay elección explícita del usuario. Se aplican comprobaciones funcionales ordinarias más una prueba de regresión por arreglo; cuando es barato, se observa que la prueba falla antes del cambio. Ejecutor: `go test`.
- **RDD:** apagado. `axiom review mode status`: off, decided by default. No hay revisión nativa; la entrega sigue la política ordinaria del repositorio.
- **Ruta por tarea:**
  - T1, T2, T5 y T9: inline o un único escritor delegado.
  - T3, T4, T6 y T7: escritor delegado, porque tocan 2 o más ficheros no triviales.

## Entrega

- Previsión: unas 1550 líneas en 8 rebanadas, sin contar T8, que pasa a P3.
- Orden: según la sección «Orden de ejecución entre bloques».
- Estrategia: `ask-on-risk`. La estrategia de cadena (`stacked-to-main` o `feature-branch-chain`) se pregunta antes del primer PR.
- RDD: solo si el usuario lo activa; está apagado por defecto.

## Progreso

- 2026-11-04: se verificaron 22 candidatos con tres exploradores de solo lectura, y se comprobaron a mano las evidencias clave. Notas completas en la carpeta de notas compartidas de la sesión (`arreglos-instalacion-sync-tui.md`).
- Carril ODD elegido por el usuario.
- 2026-11-04: el usuario ha resuelto D-1 a D-4:
  - D-1: se retira el mandato.
  - D-2: se añade el directorio al PATH.
  - D-3: se descarta #3906.
  - D-4: sonda acotada.
- 2026-11-04: coordinación con P1 a petición de la sesión de revisión upstream. Tras leer el diff de `inc/distribucion-un-comando`:
  - #3629 pasa de T1 a T7, que depende de P1;
  - T5 queda limitada a `internal/cli` e `internal/installcmd`;
  - T8 (#5359) se traslada a P3.
- 2026-11-05: T1 (`dcc15ea4`) y T2 (`e8af9f56`) hechas. Fallo ambiental conocido: `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` en `internal/app`.

## Siguiente paso

T6 y T9, en ese orden. T4 espera a adaptadores T1 y T7 a seguridad T2.
