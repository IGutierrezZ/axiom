# ODD: Retirada de la telemetría y de los restos del upstream Gentle AI

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/retirada-telemetria-y-restos-upstream.md`.
> Espejo de recuperación en Engram: topic `odd/retirada-telemetria-y-restos-upstream/tasks`, proyecto `axiom`.
> Rama del documento: `docs/odd-retirada-telemetria-y-restos-upstream`, subida a `origin`, con su worktree en `C:\repos\axiom-wt\odd-upstream-docs`. Este documento viaja **solo** en el PR de cierre.
> Abierto el 2026-10-04 desde `origin/main` `51740868`. Este documento está escrito para que **otra sesión pueda retomar el ODD sin el contexto de la sesión original**.

---

## 0. Cómo retomar este ODD en una sesión nueva

1. Recupera el contexto en este orden:
   - `mem_context`;
   - `mem_search "odd/retirada-telemetria-y-restos-upstream"` en el proyecto `axiom`;
   - `mem_get_observation` sobre la observación de topic `odd/retirada-telemetria-y-restos-upstream/tasks`;
   - **este fichero**, que es la fuente autoritativa. Está en `C:\repos\axiom-wt\odd-upstream-docs\odd\tasks\` o en `origin/docs/odd-retirada-telemetria-y-restos-upstream`.
2. Comprueba el estado real:
   - `git -C C:\repos\axiom fetch --prune origin`;
   - `git worktree list`;
   - `gh pr list --state open`;
   - el último PR de este ODD (sección 7, «Progreso»).
3. Concilia el documento con la realidad: PRs fusionados y ramas existentes. Después continúa con la **primera tarea sin marcar** de la sección 5.
4. Respeta las reglas de las secciones 3 y 4. **Varias se aprendieron por incidentes reales.**

---

## 1. Objetivo

Que Axiom **deje de enviar datos al upstream Gentle AI y de dirigir acciones contra él**, y que quede limpio de sus restos funcionales y de marca. En concreto:

1. Retirar la **telemetría por completo** (decisión del usuario, no negociable).
2. Corregir la skill propia `branch-pr`, que abría PRs en el repositorio del upstream (urgente).
3. Que los tests no contaminen la máquina del usuario: hub de workspaces y PATH.
4. Que `RefreshSkip` no regenere el índice dentro del HOME en Windows.
5. Retirar las skills descatalogadas (`branch-pr` y `gentle-ai-bench` embebidas) de las instalaciones existentes.
6. Sustituir la marca Gentle AI donde no sea un contrato, y migrar con lectura dual donde sí lo sea.

## 2. Decisiones del usuario (2026-10-04)

| Decisión | Valor |
|---|---|
| Carril | ODD |
| Telemetría | **Quitarla del todo**: cliente, hooks, plugins, CLI, documentación, colector y despliegue. «O tenemos nuestra propia telemetría o no tenemos, pero no podemos mantener envíos a servidores de upstream» |
| Alcance | Además de la telemetría: corregir `branch-pr`, el hub contaminado, el PATH con codegraph, `RefreshSkip`, las skills retiradas y los cambios de marca Gentle AI |
| Entrega | **PRs independientes** contra `main`, uno por tarea. T6 y T7 se dividen en PRs encadenados en un orden que no deje código muerto. El documento va en un PR de cierre al final |
| `size:exception` | Solo con aprobación explícita del usuario en cada PR, y únicamente si no existe un corte cohesivo |

**Decisiones pendientes**, a plantear cuando se llegue a cada tarea (una pregunta cada vez):

- **T6:** qué hacer con los hooks ya instalados. La recomendación es mantener un subcomando `axiom telemetry runtime …` que no haga nada durante una versión, para que los hooks antiguos no fallen mientras `sync` los retira.
- **T7:** si `skills/axiom-collab-perfect` se conserva o se retira. Qué copia de `chained-pr` es la canónica. Y el diseño de la lectura dual para los contratos `gentle-ai.*`.

---

## 3. Convenciones del repositorio y del usuario (obligatorias)

- **Idioma:**
  - el usuario recibe las respuestas en castellano peninsular;
  - los artefactos (documentos ODD, specs, cuerpos de PR, mensajes de commit) van en castellano;
  - los identificadores de código y los comentarios Go van en inglés, como el código que los rodea.
- **Commits:**
  - Conventional Commits, sin `Co-Authored-By` ni ninguna atribución de IA (regla del usuario, que prevalece sobre el recordatorio del sistema);
  - en el cuerpo del PR sí va la línea final `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- **Plantilla de PR** (`.github/PULL_REQUEST_TEMPLATE.md`):
  - exactamente una etiqueta `type:*`;
  - la sección «AI Assistance» marcada como «Material assistance used»;
  - **la casilla «I understand, reviewed, and take responsibility…» queda sin marcar**, porque es del usuario;
  - las casillas de `go test ./...` y E2E quedan sin marcar si no se han ejecutado en local.
- **Comprobaciones del CI** (`.github/workflows/ci.yml`, `pr-check.yml`):
  - «Check PR Cognitive Load»: como mucho 400 líneas (adiciones más borrados), salvo con la etiqueta `size:exception`;
  - «Check PR Has type:* Label»;
  - «Unit Tests» (`go test ./...`, con unos 7-10 minutos);
  - «Go Format»;
  - **`scripts/deadcode-ratchet.sh`**, que rechaza funciones nuevas sin llamadores. **Lección:** el PR #73 falló por separar un parser de su consumidor.
- **Ruleset de `main`:** solo exige la comprobación «Verify PR Origin & Authority». No es estricto, así que no hace falta que la rama esté al día. El auto-merge está desactivado en el repositorio.
- **Merge:** squash, que es la convención del repositorio: todo `main` es de un solo padre, con `(#N)` en el título.
  - En PRs encadenados, después de cada merge se hace `gh pr edit <hijo> --base main`, luego `git rebase --onto origin/main <cabeza antigua del padre> <rama hija>` y después `git push --force-with-lease=<rama>:<sha antiguo>`. Hay que esperar el CI del hijo antes de fusionarlo.
- **Flujo con el usuario:**
  - **no se consulta el CI en bucle**;
  - el usuario escribe «revisa si ya está verde y continúa», y entonces se mira el CI una vez y se sigue;
  - las preguntas se hacen de una en una, con `AskUserQuestion` cuando las opciones están cerradas;
  - push, apertura de PR y merge solo se hacen cuando el usuario los ha autorizado; en este ODD, la estrategia de PRs independientes autoriza a abrirlos;
  - tras fusionar, se eliminan el worktree, la rama local y la rama remota.
- **Worktrees:**
  - todo el trabajo va en `C:\repos\axiom-wt\<nombre>`, creados desde `origin/main`;
  - **el checkout principal `C:\repos\axiom` pertenece a otra sesión del usuario** (SDD inc-24, rama `feat/inc-24-spec-baseline-profile`): no se toca.
- **ODD:**
  - TDD desactivado (`openspec/config.yaml:16`, `strict_tdd: false`); runner `go test`;
  - RDD desactivado; el riesgo se evalúa por commit con `axiom review assess --cwd . --base-ref origin/main --committed-only --json`. Si hay ficheros sin trackear, se añade `--untracked-scope=exclude --expected-untracked-inventory=<sha>`, que el propio error indica;
  - riesgo `high` exige verificador independiente; riesgo `medium` con un writer Sonnet basta con autoverificación más una comprobación del orquestador;
  - delegación: un writer cada vez, cada uno en su worktree. El verificador es de solo lectura. Todas las llamadas a `Agent` llevan `model: sonnet`.

## 4. Reglas de seguridad (aprendidas por incidentes de la sesión original)

1. **Nunca matar procesos por nombre de imagen** (`taskkill /IM go.exe`, `pkill go`): hay otras sesiones con Go en la máquina. Solo se mata un PID propio. *Incidente: un writer lo hizo y pudo cortar procesos de la otra sesión.*
2. **Nunca ejecutar `axiom install`, `setup`, `sync`, `upgrade` ni `doctor` contra el HOME o el PATH reales.** En pruebas de extremo a extremo se usan `HOME`, `USERPROFILE` y `PATH` temporales y `DO_NOT_TRACK=1`. *Incidente: un verificador ejecutó `install` y reinstaló `engram.exe`.* Excepción: las acciones locales autorizadas de T2 y T3.
3. **No ejecutar `go test ./internal/cli/` completo en local**: tarda más de 40 minutos en esta máquina. Se usan subconjuntos con `-run`; el paquete completo lo pasa el CI.
4. **Tests de `internal/dashboard`:** ya están aislados (#71) y protegidos por una guarda (#72). Antes de esos PRs ejecutaban `upgrade` y `sync` reales.
5. **Pruebas en Linux sin Go en WSL:** se compila de forma cruzada con `GOOS=linux GOARCH=amd64 go test -c -o x.test ./pkg/` y se ejecuta **en una sola invocación**, porque `/tmp` de WSL se vacía entre llamadas:

   ```
   MSYS_NO_PATHCONV=1 wsl.exe -d Ubuntu -e sh -c "cp /mnt/c/<ruta>/x.test /tmp/x && chmod +x /tmp/x && cd /mnt/c/<ruta>/pkg && /tmp/x -test.run '<patrón>' -test.count=1" | tr -d '\0'
   ```

   Después hay que borrar el binario.
6. **Edición con perl:** no se ponen rutas Windows con `\L`, `\c` o similares en el texto de reemplazo, porque se interpretan como escapes y corrompen el fichero. Hay que usar `Edit` o Python. *Incidente: se corrompió el documento de cierre del ODD anterior.*
7. **Herramientas en Git Bash:** `fd`, `sd` y `eza` no están instaladas; `rg` y Python sí. Se usan las herramientas `Grep`, `Glob`, `Read` y `Edit` cuando se pueda.
8. **Fallos de entorno conocidos** en esta máquina Windows, que también fallan en `main`:
   - `internal/tui` `TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults`, porque un proveedor `litellm` llega desde el `~/.config/opencode/opencode.json` real;
   - `internal/cli` `TestCodeGraphGuidanceSyncStep*` y `TestRunSyncMigratesLegacyManagedPiCodeGraphSelection`, por symlinks y CodeGraph;
   - `internal/opencode` `TestRunCatalogCommandCancelsOverflowingChild`, por tiempos;
   - `internal/components/filemerge` `TestWriteFileAtomicFollowsDirectoryJunction`;
   - `internal/pathidentity` `TestSameDirectoryAcceptsTwoSpellingsOfOneDirectory` y tres `TestContainsAccepts*`, porque la cuenta no tiene el privilegio para crear symlinks (comprobado en la base el 2026-10-04).
9. **Identidad git del usuario:** `~/.gitconfig` tiene `user.name = igutierrezz@hiberus.com` y `user.email = Inigo.GUTIERREZZUFIA@berger-levrault.com`. Ya se le ha avisado y no se cambia.

---

## 5. Tareas

> Estado: `[ ]` pendiente · `[~]` en curso o con PR abierto · `[x]` fusionado.

### [x] T1 · 🔴 La skill `branch-pr` no debe apuntar al upstream: **PR #77**, fusionado como `1bed2bc1`

- **Problema:** `skills/branch-pr/SKILL.md` líneas 16, 249, 288 y 289 usaban `--repo Gentleman-Programming/gentle-ai` en `gh pr create`, `gh pr checks` y `gh pr view`.
- **Arreglo:** ahora apuntan a `IGutierrezZ/axiom` y exigen `--repo` explícito, porque GitHub marca este repositorio como fork (`isFork=true`, sin padre visible).
- **Comprobado:**
  - `internal/assets/skills/issue-creation` obtiene `TARGET` de un objetivo explícito o del remoto autenticado (`SKILL.md:45`), así que está bien.
  - `skills/axiom-collab-perfect` también menciona el upstream, pero es una skill de contribución al upstream; se decide en T7.
- **Verificación:** `go test ./internal/assets/ -run 'IssueCreation|Bundled|Skill'` pasa. Riesgo `medium`. 4 líneas cambiadas.
- **Cerrada (2026-10-04):** fusionado con squash con autorización del usuario. Eliminados el worktree `odd-up-t1` y la rama `fix/branch-pr-sin-upstream`, local y remota.

### [x] T2 · Aislar los tests de `cmd/axiom` del hub real y limpiar el hub local: **PR #78**, fusionado como `1feda4b1`

- **Hecho (2026-10-04, sesión de relevo):**
  - **Código** (ruta delegada, un writer Sonnet; disparador: preparación de escritura): nuevo `cmd/axiom/testmain_test.go` con `TestMain` (`HOME`, `USERPROFILE` y `DO_NOT_TRACK=1` en un directorio temporal) y dos tests de contrato, `TestSandboxHomeIsInForce` y `TestRunInitRegistersInSandboxHub`. 109 líneas, sin código de producción. `hub.NewManager` no lee ninguna otra variable, así que no se fijan más.
  - **Verificación:** `go test ./cmd/axiom/ -count=1` pasa en el writer y en la comprobación del orquestador; el sha256 del hub real es idéntico antes y después (`dd7758af…`) en ambas ejecuciones; `gofmt` y `go vet` limpios. Riesgo `medium` (`executable_change`); RDD desactivado.
  - **Auditoría:** solo `cmd/axiom`, `dashboard`, `knowledge` y `tui` importan `internal/hub`. `internal/tui` **lee** el hub real en `TestGovernanceScreensNavigationAndActions` (`model_test.go:9265-9275`), pero ningún test dispara su escritura (`model.go:3155-3157`). Se deja así.
  - **Limpieza local hecha:** copia en `~/.axiom/workspaces.json.bak-2026-10-04`; `project prune` (37 eliminadas), `project remove` de `testaxiomapp-8`, `-9` y `-17`, y `project switch axiom`, con un binario compilado desde `origin/main`. Resultado: 4 proyectos (`app-knowledge-agent`, `kvp25`, `kvp25-2`, `axiom`) y `axiom` activo.
  - **Seguimiento posible, fuera de alcance:** `AXIOM_STATE_DIR` (`internal/system/user_paths.go:11`) no lo fija ningún `TestMain`; si un desarrollador lo tiene definido, los tests de copia de seguridad y estado de `cmd/axiom` leerían su directorio real.
  - `scripts/deadcode-ratchet.sh` en local (Windows) señala `processVerifiedDead` y `secureLockRoot` de `internal/reviewtransaction`, ajenos a este PR; probablemente la línea base se genera en Linux. Lo decide el CI.
- **Cerrada (2026-10-04):** CI en verde, fusionado con squash y eliminados el worktree `odd-up-t2` y la rama `test/aislar-hub-cmd-axiom`, local y remota.

- **Causa:**
  - `cmd/axiom/knowledge_cli_test.go:11,48,119` (TestCLIInitKnowledgeProfile, TestCLIKnowledgeSweepAndQuery y el de Crawl) llama a `runInit` dentro del proceso (líneas 15, 52 y 123).
  - `runInit` (`cmd/axiom/main.go:1722`) llama a `hub.Register` y `SetActive` (`internal/hub/init.go:152`).
  - `hub.NewManager("")` usa `os.UserHomeDir()` (`internal/hub/manager.go:~26`), que en Windows lee `USERPROFILE`.
  - **`cmd/axiom` no tiene `TestMain`.**
  - El `TestMain` de `internal/cli` (`protocol_probe_test.go:58`) sí aísla bien; no es el culpable.
- **Estado del hub real** (`~/.axiom/workspaces.json`):
  - 44 entradas, de las que 37 apuntan a directorios que ya no existen y solo 4 son legítimas: `axiom`, `app-knowledge-agent`, `kvp25` y `kvp25-2`.
  - 13 vienen de estos tests y 27 son `axiom-test-*` del 29/09, creadas por scripts manuales; 3 de ellas aún existen en disco.
  - **El workspace activo es `testcrawlapp-4`, un huérfano.**
- **Arreglo de código:**
  - `cmd/axiom/main_test.go` (o uno nuevo `cmd/axiom/testmain_test.go`) con un `TestMain` que fije `HOME`, `USERPROFILE` y `DO_NOT_TRACK=1` en un directorio temporal, siguiendo el patrón de `internal/dashboard/main_test.go:28`, más un test que compruebe que el aislamiento está activo.
  - Revisar si hay otros paquetes de test sin aislamiento que llamen a `hub.NewManager("")`, `os.UserHomeDir` o `runInit`: con `rg -l 'NewManager\(""\)|runInit\('` sobre `*_test.go`.
- **Limpieza local, autorizada por el usuario:**
  1. Copia de seguridad: `cp ~/.axiom/workspaces.json ~/.axiom/workspaces.json.bak-<fecha>`.
  2. `axiom project prune`. Borra las 37 entradas sin directorio y reasigna el activo a la primera. No tiene `--dry-run`, así que antes conviene revisar su código: `internal/hub/manager.go` `Prune`, `cmd/axiom/main.go:1882`.
  3. `axiom project remove <id>` para las 3 `axiom-test-*` que aún existen.
  4. `axiom project switch axiom`.
  5. Comprobar con `axiom project list` y relacionar el resultado.
- **Verificación:**
  - `go test ./cmd/axiom/ -count=1` deja `~/.axiom/workspaces.json` sin cambios: se compara su sha256 antes y después.
  - El CI.
- **Tamaño:** S. Ruta: un writer para el código y el orquestador para la limpieza local.

### [x] T3 · PATH: quitar las entradas muertas y añadir una salvaguarda preventiva: **PR #79**, fusionado como `a7350417`

- **Entradas muertas en el PATH de usuario** (`HKCU\Environment` `Path`):
  - `C:\Users\igutierrezz\AppData\Local\Temp\codegraph-install-test-kvp10\current\bin`
  - `C:\Users\igutierrezz\AppData\Local\Temp\codegraph-install-test-kvp25\current\bin`

  Ningún código de Axiom crea `codegraph-install-test-*`. Probablemente vienen del instalador externo de CodeGraph (`%LOCALAPPDATA%\codegraph`), ejecutado con una raíz temporal. Los directorios ya no existen.
- **Escritor de PATH de Axiom:**
  - `internal/system/path.go:36-92` (PowerShell `SetEnvironmentVariable(...,'User')`, línea 76) y `:138-162`.
  - Lo llaman `internal/cli/run.go:1612` (mensaje «moved … ahead», `:1615`) e `internal/opencode/background.go:672`.
  - Está protegido en `go test` por `userPathRunningInGoTest()` (`path.go:186`, `flag.Lookup("test.v")`) y por un seam de runner falso (`path_test.go:20-33`).
  - **Riesgo:** un futuro test que ejecute el binario real con `install` o `sync` como subproceso esquivaría esa protección. Hoy los subprocesos de `cmd/axiom/main_test.go:169-273` solo hacen `--help` o status.
- **Arreglo de código:** una variable `AXIOM_NO_PERSISTENT_PATH`. Si vale `1`, `path.go` no escribe el PATH persistente, deja un log y no falla. Los `TestMain` (`internal/cli`, `internal/app`, `cmd/axiom`, `internal/dashboard`) la fijan. Con tests.
- **Limpieza local, autorizada por el usuario:**
  1. Copia de seguridad del valor actual: `reg.exe query "HKCU\Environment" /v Path`, guardado en un fichero.
  2. Quitar exactamente esas dos entradas con PowerShell, `[Environment]::SetEnvironmentVariable('Path', <valor sin ellas>, 'User')`, manteniendo el orden del resto.
  3. Releer y comparar.

  No se toca nada más del PATH. Ojo: la entrada `C:\Users\igutierrezz\AppData\Local\gentle-ai\bin` **se queda**, porque decidirlo corresponde a T7 y al usuario.
- **Tamaño:** S.
- **Hecho (2026-10-04, sesión de relevo):**
  - **Limpieza local hecha**, confirmada de nuevo por el usuario en esta sesión: el valor era `REG_SZ` sin variables `%…%`. Copia literal en `~/.axiom/path-user-backup-2026-10-04.txt`; quitadas solo las dos entradas (26 → 24), sigue siendo `REG_SZ`, la relectura coincide y el orden se conserva. Se observa `.dotnet\tools` duplicado (posiciones 13 y 21); no se toca.
  - **Código** (ruta delegada, writer Sonnet): el primer writer dejó `path.go` y `path_test.go` sin commit y terminó con un aviso de la organización sobre credenciales en lugar de su informe; el diff revisado por el orquestador no contiene credenciales. Diseño: constante `NoPersistentPathEnvVar` y helper `skipPersistentUserPathWrite(operation)`, que agrupa la guarda de `go test` y la variable, con `log.Printf` como diagnóstico; las lecturas (`UserPathEntries`) no cambian. Un segundo writer terminó los `TestMain` (`cmd/axiom`, `internal/app`, `internal/cli`, `internal/dashboard`), la fila de `docs/non-interactive.md` y el commit.
  - **Añadido por el orquestador:** `e2e/organicruntime` (sin etiqueta de build, entra en `go test ./...` y ejecuta `install` con el binario real).
  - **Verificación:** registro `Path` idéntico antes y después de los tests; `internal/system`, `cmd/axiom`, `internal/app`, `internal/dashboard` y un subconjunto de `internal/cli` en verde; `gofmt` y `go vet` limpios; ratchet solo con los dos avisos ajenos conocidos. Riesgo **`high`** (`process_boundary`), así que hubo **verificador independiente** (Sonnet, solo lectura): mutación confirmada (los tests fallan sin la guarda) y **un defecto bloqueante**, ya corregido: `organicEnvironment` es una lista blanca cerrada y no heredaba la variable; ahora la fija explícitamente, con el test `TestOrganicEnvironmentDisablesPersistentPathWrites`. 8 ficheros, 296 líneas.
  - **Seguimiento posible:** `bench/runner.go` (`Sandbox.env()`) es otro entorno cerrado sin la variable; la journey `bench/journeys_issue_3043.go` ejecuta `install` con subagentes en segundo plano, pero usa un stub `#!/bin/sh`.
- **Cerrada (2026-10-04):** CI en verde, fusionado con squash y eliminados el worktree `odd-up-t3` y la rama `fix/path-persistente-salvaguarda`, local y remota.

### [x] T4 · `RefreshSkip` compara por identidad, no de forma léxica: **PR #80**, fusionado como `83b438cf`

- **Causa:**
  - `internal/skillregistry/guard.go:36-53` compara `cwd == filepath.Dir(cwd)` (raíz del sistema de ficheros) y `cwd == home` tras `cleanPathArg` (`registry.go:174`), que solo normaliza separadores. El home sale de `%USERPROFILE%`.
  - En Windows, `c:\Users\…` frente a `C:\…`, `C:\Users\IGUTIE~1` (así aparece `%TEMP%`) o un prefijo `\\?\` hacen que la comparación falle.
  - Además, `hasProjectMarker(home)` es verdadero por `~/.claude/skills`, así que `Regenerate` crea `~/.atl` (`registry.go:236`), escribe el registro, lo refleja en Engram y toca `~/AGENTS.md` si existe.
  - Los llamadores son `internal/app/skill_index.go:232` e `internal/cli/sync.go:1805`.
- **Arreglo:**
  - Tras el atajo léxico, comparar por identidad con `pathidentity.SameDirectory` (`identity.go:65-75`) o con un helper equivalente a `relativeByIdentity` (`internal/skillregistry/scope.go:91-109`, de #75).
  - `projectRelativePath` no sirve tal cual, porque excluye la igualdad (`scope.go:62-64`).
  - Aplicarlo en todos los sistemas operativos (APFS tampoco distingue mayúsculas) y añadir una guarda defensiva en `Regenerate`, que se niegue a escribir en el HOME.
  - Tests: mayúsculas, nombres 8.3 (Windows, con `GetShortPathName`), `\\?\` y la raíz.
- **Tamaño:** S (2-3 h).
- **Hecho (2026-10-04, sesión de relevo):**
  - **Código** (ruta delegada, writer Sonnet): `guard.go` con `protectedDirectory`, `isFilesystemRoot` e `isHomeDirectory` (léxico primero y después `pathidentity.SameDirectory`; un HOME vacío o relativo no se compara por identidad). `Regenerate` aplica la misma regla tras `cleanPathArg` y `filepath.Abs`, en ese orden (lección de #68). Ningún llamante regenera legítimamente en el HOME.
  - **Tests:** tabla en `guard_test.go`, `guard_windows_test.go` nuevo (letra de unidad, 8.3, `\\?\`, uniones al HOME y a la raíz) y un test de extremo a extremo en `internal/app` y otro en `internal/cli`. Sin el arreglo fallan: se crea `.atl` en el HOME falso y se reescribe `AGENTS.md`. Sin cubrir: la raíz `\\?\UNC\server\share`.
  - **Verificación:** `internal/skillregistry` en verde (writer y orquestador), subconjuntos de `internal/app` e `internal/cli` y `internal/autoskill` en verde; `go vet` también para `linux/amd64` y `darwin/arm64`; ratchet solo con los dos avisos conocidos; el HOME real sin cambios. Riesgo `medium`.
  - **Tamaño:** 457 líneas (unas 87 de código y unas 370 de tests). **`size:exception` aprobado explícitamente por el usuario**, porque los tests comprueban las dos capas a la vez.
  - **Limpieza local:** el `~/.atl` del HOME real (del 2026-06-29, `skill-registry.md` y `.skill-registry.cache.json`), restos de este mismo fallo, se ha movido a `~/.atl.bak-2026-10-04` por decisión del usuario.
- **Cerrada (2026-10-04):** CI en verde, fusionado con squash y eliminados el worktree `odd-up-t4` y la rama `fix/refreshskip-identidad`, local y remota.

### [~] T5 · Retirar las skills descatalogadas de las instalaciones existentes

- **Problema:**
  - Las copias instaladas de `branch-pr` y `gentle-ai-bench` (retiradas en #70) siguen cargándose y compiten con `axiom-branch-pr` por el mismo trigger.
  - `uninstall/service.go:935` enumera `assets.FS`, así que ya no las borra.
  - `inject.go:~73-76` solo deja un log ante un ID sin asset.
  - No hay marcador ni checksum por skill; solo el `ManagedAssetDigest` global (`state/state.go:63`).
  - Si `state.json` guarda `Skills` explícitos (la TUI lo hace en `tui/model.go:2857`, `state.go:230`), `componentPaths` (`run.go:2473-2481`) exige `skills/branch-pr/SKILL.md`, y `verificationComponentPaths` (`run.go:2584`) no lo filtra. La verificación de `sync` falla si falta la copia.
  - Precedente: el commit upstream `79998d96` borró 10 skills sin código de limpieza.
- **Diseño recomendado:**
  - Un registro de retiradas `{ID, []sha256 conocidos}`: 5 versiones históricas del `SKILL.md` de `branch-pr` y 3 de `gentle-ai-bench`, extraídas con `git log --follow -p` de `internal/assets/skills/<id>/SKILL.md`.
  - **`sync`:** por cada agente, si `skills/<id>/SKILL.md` coincide con una huella conocida, borra el directorio; si está modificado, lo conserva y avisa.
  - **`uninstall`:** añade esos IDs en `service.go:935`.
  - **Estado:** se podan de `Skills` en `RestorePersistedSelection`, como ya se hace con ComponentTheme (`sync.go:~436`).
  - **Verificación:** la de ausencia reutiliza `isRetiredManagedPath` (`run.go:2863`).
- **Alternativas descartadas:** borrar por nombre sin huella, que podría eliminar copias editadas o un `branch-pr` ajeno; o solo podar el estado, que deja la copia en disco.
- **Tamaño:** M (1-2 días). Probablemente necesita `size:exception` o dos PRs (registro y `sync` en uno, `uninstall` y estado en otro). No puede quedar código muerto.
- **Plan validado (2026-10-04, agente Plan de solo lectura sobre `83b438cf`).** Correcciones al diseño:
  - **Huellas: 4 versiones de `branch-pr` y 2 de `gentle-ai-bench`**, no 5 y 3. Solo tuvieron `SKILL.md`. `sha256` de los blobs LF: `branch-pr` `e6c67d06…`, `90ab6ec4…`, `8553fdde…`, `61924090…`; `gentle-ai-bench` `a7c9576c…`, `49ac6728…`. Se comparan con el contenido normalizado a LF, porque una compilación local en Windows anterior al 2026-06-11 pudo instalar copias CRLF.
  - La instalación copia `SKILL.md` byte a byte para estas skills (sin transformación).
  - **No reutilizar `isRetiredManagedPath`:** es una comprobación dura de ausencia; una copia editada que se conserva haría fallar la verificación y revertiría todo el `sync`. En su lugar: filtrar en `selectedSkillIDs` y una comprobación *Soft* (aviso) para las copias editadas.
  - **Podar en `selectedSkillIDs` (`run.go:2130`)**, el punto común de sus 9 llamadores, después de la comprobación de longitud, para que una lista explícita formada solo por IDs retirados no caiga al preset.
  - **El ámbito por defecto de `sync` es `workspace`**: si se copia la guarda del precedente de temas (`scope != ScopeWorkspace`), las copias heredadas del HOME no se tocarían nunca.
  - Borrado seguro: solo si `<id>` es un directorio real (sin enlaces), `SKILL.md` es un fichero regular de 64 KiB o menos y su huella coincide; se vuelve a comprobar justo antes de borrar, se borra `SKILL.md` y luego el directorio solo si queda vacío (nunca `RemoveAll`). La skill propia `skills/branch-pr` (`axiom-branch-pr`) no coincide con ninguna huella y ningún adaptador apunta a `<ws>/skills`.
  - **Fallo activo en `main` (regresión de #70):** si `state.json` tiene `Skills` explícitos con `branch-pr`, `componentPaths` exige un `SKILL.md` que ya nunca se escribe y la verificación de `sync` falla siempre en ámbito `workspace`. Por eso S1 va primero.
  - En esta máquina no hay copias instaladas de ninguna de las dos skills y `state.json` no tiene `Skills` explícitos.
- **Corte en PRs (sin `size:exception`):**
  - **S1** — podar los IDs retirados de la selección (`skills/retired.go` con `IsRetired` y `WithoutRetired`, y `selectedSkillIDs`). ~100 líneas. Corrige el fallo activo.
  - **S2** — huellas, biblioteca (`InspectRetired`, `RetireInstalled`) y paso de `sync` con copia de seguridad para el *rollback*. ~380-430 líneas; si pasa de 400, se cambia el corte: biblioteca más `uninstall`, y después `sync` con el aviso.
  - **S3** — `uninstall` (borra las copias propias y conserva las editadas), aviso *Soft* en la verificación de `sync` y nota en el registro de absorción. ~200-260 líneas.
- **Decisiones de producto** (recomendaciones del plan):
  1. Que un `sync` en ámbito `workspace` limpie también las copias de los directorios de usuario del HOME de los agentes seleccionados: recomendado sí.
  2. El directorio de compatibilidad `~/.agents/skills`: recomendado dejarlo para un seguimiento.
  3. Avisar de una copia editada en cada `sync` (sin estado) o una sola vez.
  4. `uninstall`: conservar las copias editadas (recomendado) o borrar por nombre.
  5. Retirar solo en `sync`, no en `install` (recomendado).
- **Decisiones tomadas (2026-10-05):**
  - 1: **sí**, decisión explícita del usuario. Un `sync` en `workspace` también retira, por huella, las copias del HOME de los agentes seleccionados.
  - 2-5: se aplican las recomendaciones del plan, comunicadas al usuario: `~/.agents/skills` queda para un seguimiento; el aviso es sin estado, en cada `sync`, y solo si el `name` del frontmatter sigue siendo el ID retirado; `uninstall` conserva las copias editadas; la retirada solo se hace en `sync`.
- **S1 hecho: PR #81** (`e550e21c`, 199 líneas, riesgo `medium`). El test de `sync` falla sin el arreglo en los dos ámbitos (`post-sync verification failed: … branch-pr\SKILL.md`) y pasa con él. Revisado y reejecutado por el orquestador. Worktree `odd-up-t5a`, rama `fix/podar-skills-retiradas-seleccion`.
- **S2 y S3 hechos, encadenados** (worktree `odd-up-t5b`). El primer intento de S2 con el paso de `sync` ocupaba 521 líneas, así que se aplicó el corte de reserva del plan:
  - **S2: PR #82** (`ba35cf31`, rama `feat/retirar-skills-instaladas-uninstall`, base S1). Biblioteca (`InspectRetired`, `RetireInstalled` y variantes `*Against` con huellas inyectables) más `uninstall` como primer llamador: un borrado planificado por copia propia, propiedad comprobada de nuevo al aplicar y rutas en la copia de seguridad. 438 líneas: **`size:exception` aprobado por el usuario**, porque mover tests a S3 lo pasaría de 400 y recortarlos restaría cobertura.
  - **S3: PR #83** (`cae8f4ee`, rama `feat/retirar-skills-instaladas-sync`, base S2). Paso `sync:retire-installed-skills` en los dos ámbitos, con las raíces del ámbito actual más el HOME de cada agente; copia de seguridad para el *rollback*; aviso *Soft* (`WARNING:`), sin estado, solo si el `name` sigue siendo el ID retirado; viñeta en el registro de absorción. 332 líneas.
  - **Verificación:** huellas recalculadas desde el historial (las 6 coinciden); tests de `skills`, `uninstall`, `absorptionledger` y un subconjunto de `cli` en verde (writer y orquestador); ratchet limpio en cada commit; riesgo `medium` en los dos commits. El orquestador revisó la lógica de borrado: propiedad por huella, comprobación justo antes de borrar, sin seguir enlaces, sin `RemoveAll` y, ante un fallo, la copia se conserva.
- **Orden de merge:** #81, después #82 (retarget a `main` y `rebase --onto`) y después #83.
- **2026-10-05:** #81 fusionado (`6c97589a`); worktree `odd-up-t5a` y rama de S1 eliminados. #82 retargeteado a `main` y rebasado (`661abb22`); #83 rebasado encima (`bb315b5a`); los dos con `--force-with-lease` y pendientes de su CI.
- **2026-10-05:** #82 en verde y fusionado (`0ea9aec6`); rama de S2 eliminada. #83 retargeteado a `main` y rebasado (`71c3466c`), pendiente de su CI.

### [ ] T6 · Retirar la telemetría por completo (tamaño L, varios PRs)

**Inventario** (mapeo sobre `origin/main` `c504f2e6`):

- **Endpoint:** constante `internal/telemetry/telemetry.go:36`. Override con `AXIOM_TELEMETRY_ENDPOINT` (`telemetry.go:40`, `killswitch.go:65-70`). El canal runtime cambia el path por `/v1/runtime-events` (`runtime_send.go:49-53`). La documentación lo repite en `docs/telemetry.md:583`, junto a variables `GENTLE_AI_*` que ya no existen.
- **Eventos legacy** (`event.go:73-85`):
  - campos `install_id`, `sent_at`, `version`, `os`, `arch`, `agents`, `components` y `rdd_enabled`;
  - tipos `install` y `heartbeat` cada 24 h (`state.go:23-29`);
  - disparadores en `internal/app/app.go:470`, `internal/cli/run.go:306`, `sync.go:2038-2039` y `review_last_event_closure.go:144,148,296,303,307`.
- **Runtime** (`runtime.go:39-56`): modelo, esfuerzo, clase de agente, tokens, duración, categoría de error y host. No comprueba `IsDevBuild` (`runtime.go:465-471`) y exige `notice_shown=true`.
- **Estado:** `install_id` (UUID v4, `state.go:208`) se guarda en `~/.gentle-ai/telemetry.json` (`state.go:17-18`), fichero **compartido con el upstream**; la decisión se tomó en `inc-20/design.md:225`. Además, `axiom telemetry status` crea ese fichero como efecto colateral (`cli/telemetry.go:156`).
- **Consentimiento:**
  - activa por defecto (`state.go:93`, `:75`);
  - opt-out con `DO_NOT_TRACK`, `AXIOM_TELEMETRY=0`, `CI`/`GITHUB_ACTIONS` o `axiom telemetry disable` (`killswitch.go:33-49`);
  - el aviso está en inglés y se muestra una vez (`telemetry.go:48`, `opportunistic.go:107-120`).
- **Hooks escritos en los agentes:**
  - Claude Stop y SubagentStop con `axiom telemetry runtime claude --json` (`internal/components/sdd/inject.go:1900`, `:2412-2430`, `ensureClaudeTelemetryHooks`);
  - Codex con `axiom telemetry runtime codex --json` (`inject.go:1952-1971`);
  - un plugin de OpenCode (`plugins/telemetry-runtime.ts:53`, `plugins-v2/telemetry-runtime.ts:63`, instalado en `run.go:759` y `sync.go:576`), que ejecuta **`execFile("gentle-ai", …)`**: en Axiom es un shim retirado (`cmd/gentle-ai/main.go:13-16`), y si hay un `gentle-ai` del upstream en el PATH se ejecuta ese.
  - **`sync` no retira nada** (`inject.go:2048` solo migra `gentle-ai` a `axiom`).
  - El borrado ya existe y es reutilizable: `uninstall/service.go:1402-1405` y `telemetryruntime.RemoveManaged` (`managed.go:359`, con comprobación de propiedad y lista de digests en `managed.go:23-36`).
- **Componentes:**

  | Componente | Ficheros | Líneas aprox. |
  |---|---|---|
  | `internal/telemetry` | 42 | 5,5 k |
  | `internal/cli/telemetry*.go` | 9 | 2,3 k |
  | `internal/components/telemetryruntime` | 12 | 2,2 k |
  | Dos assets de plugin OpenCode, `contracts/telemetry` y `cmd/axiom/main.go:410` | — | — |
  | **Colector:** `cmd/gentle-telemetry` e `internal/telemetrycollector` | 31 | 6,7 k |
  | **Despliegue:** `deploy/telemetry/` (install.sh, systemd, Apache, VictoriaMetrics, rclone, Grafana) | 15 | 5,9 k |

  El colector importa `internal/telemetry` (`metrics.go:13`, `runtime_handlers.go:9`, `runtime_storage.go:10`). Ni `doctor` ni `dashboard` leen telemetría.
- **Tests:**
  - `killswitch_test.go:19-31,46-57`;
  - `opportunistic_test.go:198,335`;
  - `cli/telemetry_test.go:37-43`;
  - `runtime_state_test.go`, `runtime_send_test.go:103`;
  - `inject_test.go:8222-8244,8519-8684`;
  - `uninstall/service_test.go:1972-2003`;
  - `assets_test.go:18-73`;
  - los `TestMain` de `internal/app`, `internal/cli` (`protocol_probe_test.go:127-144`) e `internal/telemetry`;
  - en bench: la journey `j4395` (`bench/journeys_issue4395.go`, `journeys.manifest:38`, `review_declarations.go:112`).
- **Documentación y registro:**
  - `README.md:177` enlaza `docs/telemetry.md`, que está en inglés y con la marca Gentle.
  - `docs/upstream-absorption-ledger.md:66-84,230` marca 19 filas de telemetría como `absorbido`. **Regla 7: pasarlas a `revertido`, nunca borrarlas.**

**Plan propuesto de PRs.** Hay que validarlo con un agente Plan antes de implementar, sobre todo el orden que exige `deadcode-ratchet`:

- **T6a · Dejar de enviar y de instalar, con migración.**
  - Quitar los disparadores (app, run, sync, review closure).
  - Dejar de instalar los hooks de Claude y Codex y el plugin de OpenCode.
  - En `sync`, retirar los hooks y el plugin ya instalados, reutilizando la lógica de uninstall y `RemoveManaged`.
  - **Decisión pendiente, con recomendación:** mantener `axiom telemetry runtime <agente>` como subcomando que no hace nada y sale con 0 durante una versión, para que los hooks antiguos no fallen en las sesiones de Claude y Codex hasta que `sync` los retire.
  - Las funciones que queden sin llamadores **se borran en el mismo PR**, o el PR no pasará el ratchet.
- **T6b · Eliminar el cliente y el CLI:** `internal/telemetry`, `internal/cli/telemetry*`, `internal/components/telemetryruntime`, los assets del plugin y `contracts/telemetry`, con sus tests. Hay que extraer o borrar antes lo que importa el colector, o borrar el colector antes (T6c antes que T6b).
- **T6c · Eliminar el colector y el despliegue:** `cmd/gentle-telemetry`, `internal/telemetrycollector` y `deploy/telemetry/`.
- **T6d · Documentación y registro:** borrar `docs/telemetry.md` y su enlace en el README, pasar las filas del registro de absorción a `revertido` y resolver la journey `j4395` del bench (retirarla o adaptarla).
- **No se borra `~/.gentle-ai/telemetry.json` de los usuarios:** es compartido con el upstream y no es de Axiom. Simplemente se deja de leer y de crear.
- **Mientras tanto, en la máquina del usuario:** la telemetría sigue activa. Se le ha sugerido ejecutar `axiom telemetry disable`.

Cada PR lleva riesgo `high` probable (borrado masivo y hooks), así que necesita verificador independiente. Probablemente haga falta `size:exception` en T6b y T6c, que son borrados masivos, con aprobación del usuario.

**Plan validado (2026-10-05, agente Plan de solo lectura sobre `6c97589a`).** Sustituye al plan propuesto de arriba.

- **Correcciones al inventario:**
  - **El plugin de OpenCode es una fuga real.** Ejecuta `gentle-ai telemetry runtime opencode --json`. Si el `gentle-ai` real del upstream está en el PATH, **se siguen enviando datos a través de nuestro plugin**.
  - **El `sync` diferido no cubre la migración.** Solo se consume al arrancar la TUI sin argumentos (`app.go:239-263`). Ni `axiom upgrade`, ni los scripts de instalación, ni el uso sin interfaz lo ejecutan, y los hooks se invocan por `cmd/axiom/main.go:410`, que no pasa por la autoactualización. **Hace falta un stub.**
  - `managed.go` (`inspect`, `:129-192`) demuestra la propiedad comparando con los assets embebidos. Al borrarlos hay que sustituirlo por una lista fija con los 4 digests distribuidos.
  - El colector no está en ningún workflow, ni en `.goreleaser.yaml`, ni en un Makefile, ni en la línea base de código muerto. Borrar un paquete huérfano entero nunca dispara el ratchet; solo lo hacen las funciones que se quedan sin llamadores dentro de paquetes que siguen importados.
  - `modernc.org/sqlite` y sus dependencias indirectas solo los usa el colector: `go mod tidy` los quita.
  - No hay ratchet ni golden test que enumere símbolos de telemetría. `.guard-population-baseline.txt` fija el hash de una guarda de `sync.go`: no tocar esa sentencia.
  - Otros puntos afectados: `scripts/test-opencode-v2-host.py:19` (`PLUGIN_IDS`), `assets_test.go:649-656` (5 plugins), `docs/opencode-compatibility.md:10` y el requisito REQ-20.12 (`openspec/specs/axiom-distribution-identity/spec.md:74-79`), que queda sin objeto.
- **Forma de los hooks instalados:**
  - Claude: `axiom telemetry runtime claude --json` (async, timeout 5), en `Stop` y `SubagentStop`.
  - Codex: `axiom telemetry runtime codex --json` (async, timeout 4), y debe callar en stdout.
  - Instalaciones antiguas: `gentle-ai …`, que ya fallan por el shim.
  - **Stub mínimo:** `axiom telemetry <lo que sea>` sale con 0. `runtime …` no imprime nada, no valida argumentos y no lee stdin. El resto de subcomandos imprime una línea en stderr («telemetry removed»). Se conecta en `cmd/axiom/main.go:410` y `internal/app/app.go:134`, en un fichero pequeño de `internal/cli` que no importa nada de telemetría. El plugin de OpenCode no necesita stub: basta con borrarlo.
- **Cadena de PRs** (tres `size:exception`, en T6b, T6d y T6e):
  - **T6a** — dejar de enviar los eventos legacy automáticos: quitar `TelemetryTrigger` de `app.go:470`, `run.go:306` y `sync.go:2038-2039` y las seis llamadas a `telemetryRecordReviewOutcome`; borrar los helpers que quedan sin llamadores y `counters.go`. ~380 líneas.
  - **T6b** — el corte: el stub y el borrado de `internal/cli/telemetry*.go`, de `telemetryruntime/{claude,codex,opencode}.go` y de la journey `j4395` del bench. Al terminar, `internal/telemetry` sale del grafo de imports. ~3,3 k líneas borradas, `size:exception`.
  - **T6c** — retirar los hooks de Claude y Codex: dejar de escribirlos y, en `sync` o `install`, quitar los existentes (coincidencia exacta con los prefijos `axiom` y `gentle-ai`, sin tocar entradas del usuario). `uninstall` ya los quita. ~350 líneas.
  - **T6d** — retirar el plugin de OpenCode: lista fija de 4 digests en `managed.go`; un paso de retirada no fatal en `install` y en `sync`; borrar los assets `.ts` y `Reconcile*`, con sus tests. ~1-1,3 k líneas, `size:exception`.
  - **T6e** — borrado final de `cmd/gentle-telemetry`, `internal/telemetrycollector`, `internal/telemetry`, `deploy/telemetry`, `contracts/telemetry` y `docs/telemetry-collector.md`, más `go mod tidy`. También quita la telemetría de los `TestMain`, manteniendo su aislamiento del HOME y del PATH, y añade un test de guarda contra la reaparición del endpoint. ~20 k líneas, `size:exception`.
  - **T6f** — `README.md:177`; `docs/telemetry.md`, reducido a una declaración de ~15 líneas de que Axiom no envía telemetría; ledger con 20 filas de `absorbido` a `revertido` (19 en F2 más `80c927ae`), con motivo no vacío, y el recuento a 60/11/20; nota de retirada de REQ-20.12. ~100 líneas.
  - **Orden:** T6a → T6b (con el stub) → T6c y T6d (independientes) → T6e → T6f.
- **Decisión del usuario (2026-10-05): sin stub.** «No hay ese alguien: el único proyecto con Axiom instalado es Axiom, por lo que se puede quitar y hacer sync.»
  - **Nuevo orden: T6a → T6c → T6d → T6b → T6e → T6f.** Primero se deja de enviar y se retiran los hooks y el plugin mediante `sync`. El comando se borra solo cuando el usuario ya ha ejecutado `sync`, para que ningún hook llame a un comando inexistente. T6b deja de llevar stub.
  - Se aplican las demás recomendaciones del plan: retirar el plugin en `install` y en `sync`; ante un conflicto de propiedad, avisar y saltar; mantener un `docs/telemetry.md` mínimo; no fusionar T6b con T6e; sin hotfix del endpoint; el ledger, después de T6e; retirar REQ-20.12 en T6f. Los `size:exception` se piden PR a PR.
  - **Instalación real (lectura del 2026-10-05):** 2 hooks en `~/.claude/settings.json`; el plugin `~/.config/opencode/plugins/telemetry-runtime.ts`; `~/.gentle-ai/telemetry.json` (no se borra, es compartido con el upstream); y `C:\repos\axiom\.claude\settings.json`, que no está versionado y pertenece al checkout de la otra sesión. Ningún fichero versionado del repositorio lleva hooks de telemetría.
- **Progreso:**
  - **T6a: PR #84** (`600350b3`, rama `fix/telemetria-dejar-de-enviar`, worktree `odd-up-t6a`). Quita los disparadores automáticos de `app`, `run` y `sync` y las seis llamadas del cierre de revisiones, y borra lo que queda sin llamadores (`counters.go` incluido). +5 y −383 líneas, riesgo `medium`. Verificado: build y vet limpios; `internal/telemetry`, `internal/app` y subconjuntos de `internal/cli` en verde; ratchet sin funciones nuevas inalcanzables. Se mantienen `runTelemetryTriggerCommand` y sus helpers, que siguen siendo alcanzables desde `axiom telemetry trigger` hasta T6b.
  - **T6c: PR #85** (`ed9fff1d`, rama `fix/telemetria-retirar-hooks`, worktree `odd-up-t6c`), con 359 líneas y riesgo `medium`.
    - **Cambio:** deja de instalar los hooks de Claude y Codex y los retira en cada `install` o `sync` que inyecta SDD.
    - **Coincidencia exacta** con 4 comandos: `axiom` y `gentle-ai`, para `claude` y `codex`. Solo en `Stop` y `SubagentStop`. Conserva los hooks del usuario, borra los contenedores que quedan vacíos, es idempotente y nunca crea `settings.json`.
    - **Verificación:** paquete `sdd` completo y subconjuntos de `uninstall` y `cli` en verde. El orquestador revisó el helper.
  - **T6d: PR #86** (`7dc41fd4`, worktree `odd-up-t6d`, rama `fix/telemetria-retirar-plugin-opencode`). El verificador independiente encontró un **defecto bloqueante, ya corregido**: el directorio de configuración de OpenCode había dejado de ser raíz de restauración del *rollback*, que solo llegaba ahí por el código de telemetría. Con `XDG_CONFIG_HOME` fuera del HOME, un `install` o `sync` fallido no restauraba `skill-registry.ts`. Ahora `rollbackRestoreStep.openCodeConfigDir`, con el test de regresión `TestOpenCodeRollbackRestoresConfigOutsideHomeViaXDG` en `install` y `sync`. También se añadió el test de `AfterHash == sha256(After)`, cuya mutación sobrevivía. Tamaño final: 1658 líneas (+670 / −988), con `size:exception` aprobado. **Seguimiento:** el `sync` de la TUI pierde el aviso, porque `tui.SyncFunc` solo devuelve los ficheros cambiados.
    - **Tamaño:** 1518 líneas (+531 / −987). **`size:exception` aprobado por el usuario.**
    - **Cambios:**
      - `openCodeTelemetryRetirementStep` en `install` y `sync`, al final y no fatal;
      - lista fija con 4 digests (`ff05ed23…`, `8053fc82…`, `902fe092…`, `54150f7d…`, recalculados);
      - borrados `Reconcile*`, `guardedFile` y el *rollback*;
      - assets movidos a `testdata`;
      - avisos a través de `retirementNotes`, que llegan a `ManualActions` en `install` y a `WARNING:` en `sync`.
    - **Riesgo `high`** (`process_boundary`).
    - **Hallazgos del writer:**
      - `docs/telemetry.md:255-256` queda obsoleto; se corrige en T6f.
      - El plugin `skill-registry.ts` (v1 y v2) también ejecuta `gentle-ai skill-registry refresh`; pasa a T7.
- **Riesgos:**
  - Las sesiones de OpenCode abiertas mantienen el plugin hasta que se reinician.
  - Los hooks editados por el usuario sobreviven y dependen del stub.
  - `state.json` no tiene campos de telemetría.
  - Mantener `DO_NOT_TRACK` en los `TestMain` hasta T6e.
  - El código de retirada debe reconocer los ID legacy (`gentle-ai.telemetry-*` y las marcas `// gentle-ai:managed telemetry-runtime/v1|v2`).

### [ ] T7 · Marca Gentle AI (tamaño M-L, varios PRs)

| Elemento | Clase | Acción | Tamaño |
|---|---|---|---|
| `AGENTS.md:29` («# Gentle AI™ — Agent Skills Index») y `:33` (convención `gentle-ai-*`, ya obsoleta) | Marca pura, fuera del bloque generado (línea 42) | Cambiarlas a Axiom | S |
| 164 mensajes `` `gentle-ai <verbo>` `` en 58 ficheros de producción, frente a 69 con `axiom` (por ejemplo `run.go:2816`, `review_asset_provenance.go:11`) | Marca pura con riesgo: el PATH del usuario tiene `AppData\Local\gentle-ai\bin`, así que el consejo puede lanzar el binario del upstream | Pasarlos a `axiom <verbo>`, regenerar los goldens y revisar los tests: 301 ficheros de test mencionan gentle-ai | M (1-2 días) |
| `internal/assets/skills/chained-pr`, que duplica a `skills/chained-pr` (`axiom-chained-pr`) | La copia embebida es más nueva (2 reglas y 1 fila más) | **Decisión del usuario**: borrar `skills/chained-pr` o añadir un test de deriva. Renombrar el ID embebido rompe instalaciones | S, o M con el registro de T5 |
| `skills/axiom-collab-perfect` (skill para contribuir a `Gentleman-Programming/gentle-ai`) | Funcional, apunta al upstream | **Decisión del usuario**: conservarla o retirarla. Se mantiene `author: ardelperal` | S |
| IDs de protocolo `gentle-ai.*/vN` | Contrato. Los de revisión y telemetría ya tienen lectura dual (`reviewtransaction/contract.go:10-47`, `telemetry.go:18-19`); quedan unos 40 sin migrar (`verification-*`, `provider-transport`, …). Los persistidos o con hash son contrato duro: `compact.go:16`, `compact_store.go:1888`, `artifact_subject.go:11-12,160,183` | Diseñar la lectura dual: escribir `axiom.*` y aceptar los dos | M-L, y L con migración de los persistidos |
| Marca `gentle-ai:managed-opencode-launcher/v1` (`opencode/background.go:27,754`, `launcher_target.go:53`, `uninstall/service.go:1870`) | Contrato con instalaciones | Aceptar las dos marcas y escribir la nueva | S-M |
| Plugin de OpenCode `skill-registry.ts` (v1 y v2), que ejecuta `gentle-ai skill-registry refresh` (hallazgo de T6d) | Funcional: con el `gentle-ai` real del upstream en el PATH ejecuta su binario; con el shim de Axiom falla | Pasar a `axiom`, con migración de las copias instaladas | S-M |
| Hooks legacy (`sdd/inject.go:1918,2085-2087`) y `.gentle-ai/bin` (`background.go:334`) | Migración existente | Se mantienen | — |
| `metadata.author: gentleman-programming` en las skills | Atribución de licencia | Se mantiene | — |

---

## 6. Criterios de aceptación

- Ningún binario, hook ni plugin de Axiom envía datos a dominios del upstream. Tras `sync`, las instalaciones existentes quedan sin hooks ni plugin de telemetría y sin errores en las sesiones de los agentes.
- Ninguna skill propia dirige acciones (PRs, issues) contra el repositorio del upstream.
- `go test ./...` no modifica el hub (`~/.axiom/workspaces.json`) ni el PATH persistente del usuario.
- `RefreshSkip` no regenera el índice dentro del HOME por diferencias de forma en la ruta.
- `sync` retira `branch-pr` y `gentle-ai-bench` sin modificar de las instalaciones existentes y conserva y avisa de las copias editadas.
- No quedan mensajes visibles que recomienden `gentle-ai <verbo>`. Los contratos se leen con los dos nombres.
- El CI está en verde en todos los PRs. El hub y el PATH del usuario quedan limpios (acciones locales de T2 y T3).

## 7. Progreso

- **2026-10-04, apertura:**
  - Mapeo previo con dos agentes de exploración, uno para la telemetría y otro para los seguimientos.
  - Decisiones: carril ODD; quitar la telemetría del todo e incluir todos los seguimientos; PRs independientes.
- **2026-10-04, T1 implementada:**
  - `9454f652` y **PR #77**, con 4 líneas y riesgo `medium`.
  - Pendiente de que el CI esté en verde para fusionarlo y eliminar `axiom-wt/odd-up-t1`.
- **Relevo:** a petición del usuario, este ODD continúa en **otra sesión**, porque la original acumulaba demasiado contexto. Estado en el momento del relevo:
  - **Worktrees de este ODD:**
    - `C:\repos\axiom-wt\odd-upstream-docs`, rama `docs/odd-retirada-telemetria-y-restos-upstream`, con este documento;
    - `C:\repos\axiom-wt\odd-up-t1`, rama `fix/branch-pr-sin-upstream`, del PR #77.
  - **Ramas remotas:** `main`, `feat/cli-json-query` (del usuario, no se toca), `fix/branch-pr-sin-upstream` y `docs/odd-retirada-telemetria-y-restos-upstream`.
  - **Checkout principal:** sigue siendo de la otra sesión (inc-24) y no se toca.
- **2026-10-04, sesión de relevo:**
  - Contexto recuperado de Engram (#457, #458) y de este documento; conciliado con `origin`.
  - #77 en verde: fusionado (`1bed2bc1`) con autorización del usuario y limpiado. **T1 cerrada.**
  - T2: código en **PR #78** (`43935d61`, 109 líneas, riesgo `medium`) y limpieza local del hub hecha.
  - #78 en verde: fusionado (`1feda4b1`) y limpiado. **T2 cerrada.**
  - T3: limpieza local del PATH hecha y código en **PR #79** (`ad1f5095`, riesgo `high` con verificador independiente).
  - #79 en verde: fusionado (`a7350417`) y limpiado. **T3 cerrada.**
  - T4: **PR #80** (`dcb077da`, 457 líneas con `size:exception` aprobado, riesgo `medium`). Pendiente del CI de #80.

## 8. Siguiente paso

1. T5: fusionar #83 cuando su CI esté en verde (ya apunta a `main`). Después, eliminar el worktree `odd-up-t5b` y sus ramas.
2. T6: plan de PRs con un agente Plan (lanzado el 2026-10-05), plantear al usuario la decisión del stub de `axiom telemetry runtime` y después ejecutar T6a a T6d.
3. T7: plantear al usuario las decisiones de `axiom-collab-perfect` y `chained-pr` y diseñar la lectura dual de los contratos.
4. PR de cierre con este documento.

## 9. Historia relacionada (ODDs anteriores de la misma sesión)

- `odd/tasks/skill-index-versioned-view.md`: vista versionada del índice de skills (#64-#67) y hotfix #68 (`--cwd` normalizado antes de `Abs`).
- `odd/tasks/limpieza-gentle-ai-y-seguimientos.md`: retirada de `branch-pr` y `gentle-ai-bench` embebidas (#70), `ScopeForPath` en Windows (#75), tests de `dashboard` aislados (#71 y #72) y `doctor` con el lanzador gestionado (#74).
- `odd/tasks/kiro-workspace-parity.md`: solo se versiona el hook de Kiro (#69).
