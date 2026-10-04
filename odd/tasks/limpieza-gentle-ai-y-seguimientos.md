# ODD: Limpieza de restos de Gentle AI y seguimientos del índice de skills

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/limpieza-gentle-ai-y-seguimientos.md`.
> Espejo de recuperación en Engram: topic `odd/limpieza-gentle-ai-y-seguimientos/tasks`, proyecto `axiom`.
> Abierto el 2026-10-04 desde `origin/main` (`3a35fc66`).

## Objetivo

Retirar de Axiom las skills específicas de Gentle AI y cerrar tres seguimientos que surgieron al entregar la vista versionada del índice de skills (#64-#68):

- la clasificación de ámbito en Windows (D4);
- los tests de `dashboard` que escriben en el repositorio;
- un falso positivo de `axiom doctor`.

## Problema

- **Skills de Gentle AI.** `internal/assets/skills/branch-pr` («Create Gentle AI pull requests») y `internal/assets/skills/gentle-ai-bench` son skills del upstream, y Axiom ya tiene sus propios equivalentes (`skills/branch-pr`, con nombre `axiom-branch-pr`, y `skills/axiom-bench`). Aun así aparecen en el índice de `AGENTS.md`, en el catálogo, en los presets de contribución y en el selector de la TUI. **Decisión del usuario, no negociable: deben desaparecer.**
- **D4.** `ScopeForPath` (`internal/skillregistry/registry.go`, ~520) compara prefijos de ruta de forma léxica y distinguiendo mayúsculas. En Windows, si la letra de unidad cambia de mayúsculas (`C:` frente a `c:`) o se usa un nombre corto 8.3 (`IGUTIE~1` frente a `igutierrezz`), una skill del proyecto se clasifica como de usuario. Desde #64 eso la saca **sin avisar** de `AGENTS.md` y de Engram, y la muestra con ruta absoluta.
- **Tests de `dashboard`.** `go test ./internal/dashboard/` modifica el checkout, como se comprobó en un worktree limpio:
  - cambia `openspec/INDEX.md`;
  - crea `.atl/`, `.claude/*` (`CLAUDE.md`, agents, commands, mcp, output-styles, `settings.json` y skills), `.config/`, `.gemini/`, `.kiro/` y `.mcp.json`;
  - lanza `codegraph index` en la raíz.

  Además, `TestSemanticEndpoints` falla cuando CodeGraph no está inicializado.
- **`doctor`.** Avisa de que `opencode` está duplicado en el PATH, pero una de las dos copias es el lanzador que gestiona el propio Axiom (`~/.axiom/bin/opencode.cmd`, con la marca `gentle-ai:managed-opencode-launcher/v1`), que envuelve al de npm. El aviso es un falso positivo del producto.

## Alcance

- **T1:** retirar `branch-pr` y `gentle-ai-bench` de los assets, el catálogo, los presets, la TUI, la documentación, los tests y el e2e. Regenerar `AGENTS.md` y registrar el descarte en el registro de absorción del upstream.
- **T2:** hacer que `ScopeForPath` sea correcto en Windows ante diferencias de mayúsculas y nombres cortos 8.3, sin cambiar el comportamiento en otros sistemas.
- **T3:** que los tests de `internal/dashboard` no modifiquen el checkout y que el test semántico se salte cuando CodeGraph no esté disponible.
- **T4:** que `doctor` no avise de un duplicado de `opencode` cuando una de las copias es el lanzador gestionado por Axiom.

## Restricciones

- `bench/` y el CI usan `gentle-ai-bench` como **nombre del binario y del esquema**. No se tocan.
- No se modifican los identificadores de protocolo `gentle-ai.*/vN` ni la marca `gentle-ai:managed-opencode-launcher/v1`: son contratos con instalaciones existentes. Solo se señalan.
- Las referencias a `skills/branch-pr/` (la skill propia de Axiom) se mantienen.
- Los fixtures que usan «branch-pr» como nombre arbitrario se mantienen.
- El histórico (`odd/tasks/*`, `docs/archive`, `openspec/changes/archive`) no se toca.
- El checkout principal `C:\repos\axiom` pertenece a otra sesión del usuario (SDD inc-24). Todo el trabajo se hace en worktrees aparte.
- Unas 400 líneas por tarea es solo una heurística orientativa.

## Configuración de ejecución

- **TDD:** desactivado. Fuente: `openspec/config.yaml:16` (`strict_tdd: false`). Runner: `go test`.
- **Entrega:** `ask-on-risk`. Previsión: unas 800 líneas de autor (T1 unas 370, de las que 222 son de borrado de assets; T2 unas 120; T3 entre 200 y 300; T4 unas 80).
- **Estrategia elegida por el usuario: PRs independientes contra `main`**, uno por tarea y sin dependencias entre ellos. **Ajuste del 2026-10-04:** este documento viaja solo en el PR de cierre (`docs/odd-limpieza-gentle-ai-cierre`), no en el de T1. Con el documento, T1 sumaba 465 líneas y superaba el presupuesto; sin él se queda en 318, sin necesidad de `size:exception`.

  | PR | Tarea | Rama | Worktree |
  |---|---|---|---|
  | #70 | T1 | `chore/retirar-skills-gentle-ai` | `axiom-wt/odd-gentle-t1` |
  | — | T2 | `fix/scope-for-path-windows` | `axiom-wt/odd-gentle-t2` |
  | #71 + #72 | T3 (aislamiento + guarda encima) | `test/dashboard-aislado-del-repo` + `test/dashboard-guarda-del-checkout` | `axiom-wt/odd-gentle-t3` |
  | — | T4 | `fix/doctor-lanzador-opencode` | `axiom-wt/odd-gentle-t4` |
  | — | Cierre (con este documento) | `docs/odd-limpieza-gentle-ai-cierre` | `axiom-wt/odd-gentle-docs` |

- **Ejecución:** un writer cada vez, de forma secuencial.
- **RDD:** desactivado. Se evalúa el riesgo por commit con `axiom review assess`.

## Tareas

- [x] **T1 · Retirar `branch-pr` y `gentle-ai-bench`** (`c8ce5937`, PR #70, fusionado como `c990e57d`)
  - Borrar `internal/assets/skills/branch-pr/` y `internal/assets/skills/gentle-ai-bench/`.
  - Quitar sus IDs de `internal/model/types.go`, `internal/catalog/skills.go`, `internal/components/skills/presets.go` y `internal/tui/screens/skill_picker.go`.
  - Documentación: `docs/components.md`. En `docs/usage.md`, el ejemplo de `--skill` pasa a usar `work-unit-commits`.
  - Tests y `e2e/e2e_test.sh` (inventario en «Notas de exploración»).
  - Instalaciones existentes: si sync, install o uninstall tienen un mecanismo para retirar skills gestionadas, registrar estos dos IDs. Si no lo tienen, documentarlo como seguimiento.
  - Regenerar `AGENTS.md` sin el espejo Engram.
  - Añadir una fila de descarte deliberado en `docs/upstream-absorption-ledger.md`, respetando su formato y sus reglas.
  - Ruta: delegada (writer). Disparadores: preparación de escritura y 2 o más ficheros no triviales.
- [x] **T2 · `ScopeForPath` correcto en Windows** (PR #75 con `size:exception`, fusionado como `c504f2e6`)
  - En Windows, comparar prefijos sin distinguir mayúsculas y normalizar los nombres cortos 8.3 cuando la ruta exista. Sin cambios en otros sistemas.
  - Tests: diferencias de mayúsculas en la unidad y en los directorios, y nombre corto 8.3 si se puede obtener en el test.
  - Ruta: delegada (writer).
- [x] **T3 · Tests de `dashboard` aislados del repositorio** (PR #71, fusionado como `ccff924c`; PR #72 fusionado como `7aa6e9bc`)
  - Localizar los tests que resuelven la raíz del repo o el cwd y escriben en él, y pasarlos a fixtures en `t.TempDir()` o a stubs.
  - `TestSemanticEndpoints` se salta cuando CodeGraph no está disponible o inicializado.
  - Verificación: ejecutar el paquete en un worktree limpio y comprobar que `git status --porcelain --ignored` sigue vacío.
  - Ruta: delegada (writer). Disparadores: exploración de 4 o más ficheros y preparación de escritura.
- [x] **T4 · `doctor` reconoce el lanzador gestionado de `opencode`** (PR #74 con `size:exception`, fusionado como `7b6b7abd`; sustituye a #73)
  - Si una de las copias del PATH es el lanzador gestionado por Axiom (marca `gentle-ai:managed-opencode-launcher/v1`) y envuelve a la otra, no se informa de duplicado. Puede quedar, como mucho, una nota informativa.
  - Tests.
  - Ruta: delegada (writer).

## Criterios de aceptación

- `branch-pr` y `gentle-ai-bench` no aparecen en los assets, el catálogo, los presets, la TUI, `AGENTS.md`, la documentación activa ni los tests como skills instalables. Pasan `go test ./...` en el CI y los tests afectados en local.
- `ScopeForPath` clasifica como proyecto una skill del repo aunque haya diferencias de mayúsculas o nombres 8.3 en Windows.
- `go test ./internal/dashboard/` deja el checkout sin cambios.
- `doctor` no muestra un aviso de duplicado de `opencode` por el lanzador gestionado.

## Notas de exploración (mapeo previo, `origin/main` `3a35fc66`)

- **T1. Tests que referencian estas skills:**
  - `internal/assets/assets_test.go:482`
  - `internal/assets/skills_frontmatter_test.go:73-74`
  - `internal/catalog/skills_test.go:51`
  - `internal/cli/install_test.go:98,104`
  - `internal/cli/openclaw_orchestration_test.go:325`
  - `internal/cli/run_component_paths_test.go:370,378,387,516,527,538`
  - `internal/cli/run_integration_test.go:2136,2150-2155,2185,2596`
  - `internal/components/skills/presets_test.go:74-75,93,97,146`
  - `internal/components/skills/inject_test.go:339,486`
  - `internal/tui/screens/skill_picker_test.go:12,26`
  - `e2e/e2e_test.sh`: 695, 724, 741-749, 785-798 y 973.
- **T1. Instalaciones existentes:** `uninstall/service.go:925` enumera el directorio embebido de skills, así que tras el borrado ya no limpiaría las copias instaladas de estas dos skills.
- **Restos de Gentle AI que solo se señalan:**
  - la cabecera «Gentle AI™ — Agent Skills Index» de `AGENTS.md`;
  - la skill embebida `chained-pr`, que duplica a `axiom-chained-pr`;
  - `axiom-collab-perfect` y `skills/branch-pr`, que apuntan a `Gentleman-Programming/gentle-ai`;
  - los identificadores `gentle-ai.*/vN`;
  - la marca del lanzador.
- **Entorno local del usuario, fuera de alcance:**
  - `go\bin\engram.exe` es la versión 1.20.0, obsoleta; la vigente es la de `AppData`, la 3.0.0.
  - `gemini-cli` figura en el estado, pero no está su binario.

## Progreso

- 2026-10-04: ODD abierto. Decisiones del usuario: carril ODD, las cuatro tareas, la retirada no negociable de las skills de Gentle AI y PRs independientes.
- **T1: completada (`c8ce5937`, PR #70).** Ruta delegada en un writer.
  - Tamaño: 222 líneas borradas de assets, más +37/−59 en 24 ficheros de código, tests, docs, registro de absorción, `AGENTS.md` y goldens.
  - Cambios adicionales que no estaban en el inventario: `TestEmbeddedAssetCount` pasa de 27 a 25, y se regeneran dos goldens de la TUI.
  - Sustituciones en tests y docs: `work-unit-commits` en los tests de cli y en `docs/usage.md`, `rdd-defect-workflow` en `install_test.go`, e `issue-creation` en el e2e.
  - **Registro de absorción:** es un registro cerrado, con una fila por sha, y su validador rechaza filas ajenas. Por eso el descarte se anota en una sección en prosa al final, «Descartes de contenido fuera del universo congelado», sin cambiar `## Recuento`.
  - **`AGENTS.md`:** solo desaparecen las dos filas. La línea manual 33, sobre la convención `gentle-ai-*`, queda obsoleta; solo se señala.
  - **Comprobaciones del writer:**
    - Pasan `go build`, `gofmt`, `vet`, `gofmtcheck` y `bash -n` del e2e.
    - Pasan los paquetes assets, catalog, `components/skills`, `tui/screens`, model y `absorptionledger`, y `internal/app -run Documented|SkillIndex`.
    - De `internal/cli` se ejecutaron los 373 tests de skills: 362 pasan, 8 se saltan y 3 fallan, los tres preexistentes en `3a35fc66` por symlinks y CodeGraph en Windows. El paquete completo no termina en local (más de 40 minutos) y queda para el CI.
  - **Riesgo evaluado:** `high`, por un único motivo: `e2e/e2e_test.sh` lanza procesos de shell.
  - **Verificación independiente: PASS.**
    - El e2e se probó con el binario real: los recuentos asertados (2, 8 y 8) coinciden, e `issue-creation` es una sustitución válida.
    - Ningún test se debilita.
    - El grep de completitud no deja restos.
    - El registro de absorción cumple sus reglas.
  - **Seguimientos detectados, fuera de alcance:**
    - Axiom no tiene mecanismo para retirar skills gestionadas que ya no distribuye. Quien las instaló conserva sus copias, `uninstall` deja de limpiarlas (`uninstall/service.go:925-936`) y la verificación de `sync` (`run.go:2473-2481`) podría exigir el fichero de un ID que siga guardado en `state.json`.
    - La telemetría de Axiom está activada por defecto y envía a `telemetry.gentlemanprogramming.com`, el servidor del upstream.
    - El PATH de usuario contiene entradas `AppData\Local\Temp\codegraph-install-test-*`, restos de tests de instalación de CodeGraph.
  - **Incidentes con efectos fuera del worktree:**
    - El writer ejecutó `taskkill /IM go.exe`, que puede haber cortado procesos Go de otra sesión. Lección registrada: no matar procesos por nombre.
    - El verificador ejecutó `install` en el worktree. Creó `.claude/`, que luego borró, y reinstaló `engram.exe` 3.0.0 en `AppData`. El PATH final es correcto: `engram\bin` va primero.

- **T2: completada (`a1a2ceff`).** Ruta delegada en un writer.
  - Diseño: una sola regla, `projectRelativePath` (`internal/skillregistry/scope.go`), usada por `ScopeForPath`, la ruta de la tabla de `AGENTS.md`, `versionableFiles` y `dedupeBySkillName`.
    - Fuera de Windows se mantiene la comprobación léxica exacta, sin cambios.
    - En Windows, solo si la comprobación exacta falla, se prueba `filepath.Rel` (que no distingue mayúsculas) y después la identidad con `os.SameFile` del ancestro que está a la profundidad de `cwd`, para cubrir los nombres 8.3.
    - No se usa `EvalSymlinks`, para no sacar del proyecto las skills enlazadas mediante junctions.
  - Tamaño: 6 ficheros, +592/−13. Los tests suman 458 líneas (el 76 %).
  - Comprobaciones del writer:
    - Los tests de Windows se ejecutaron sin saltarse ninguno: mayúsculas, nombres 8.3 (activos en este volumen), junctions, identidad y `Regenerate` con el cwd escrito de otra forma.
    - Con la rama de Windows desactivada fallan 6 tests.
    - En WSL, PASS. `gofmtcheck` y `vet` limpios (incluido `GOOS=linux/darwin`).
  - Comprobaciones del orquestador: los tests pasan. Riesgo `medium`.
  - **Presupuesto:** no admite un corte cohesivo. El mecanismo es uno solo y los tests del comportamiento en Windows superan las 400 líneas por sí solos. Se recomienda `size:exception`, pendiente de que el mantenedor lo acepte.
  - Fuera de alcance:
    - `RefreshSkip` compara `cwd == home` de forma léxica, así que tiene el mismo hueco con mayúsculas y nombres 8.3.
    - «Sources scanned» de `.atl` usa `filepath.Rel` sin normalizar, lo que solo afecta a cómo se muestra.

- **T3: completada.** Ruta delegada en un writer. Se dividió en dos PRs para no pasar del presupuesto: el aislamiento (`1e13896c`, PR #71, 393 líneas) y la guarda encima (`d2372155`, PR #72, 127 líneas).
  - **Causa raíz:** once tests usaban `NewService("../..")`, la raíz real del repositorio, con el HOME real:
    - `TestEcosystemEndpoints` lanzaba un `sync --scope workspace` real (de ahí `.claude/*`, `.atl`, `.kiro`, `.gemini`, `.config` y `.mcp.json`), un **`upgrade` real contra la red** y una copia de seguridad del `~/.axiom` real;
    - `TestArchiveEndpoints` reescribía `openspec/INDEX.md`;
    - `TestSemanticEndpoints` ejecutaba `codegraph index` en la raíz;
    - `TestSpecsSyncStatusEndpoint` hacía un `git fetch` real;
    - dos tests de `service_sequence_test.go` ejecutaban un `upgrade` y un `sync` reales.
  - **Arreglo:**
    - `TestMain` aísla el HOME y fija `DO_NOT_TRACK=1`.
    - Fixtures en `t.TempDir()`, un `codegraph` falso en el PATH y stubs para `sync` y `upgrade`.
    - Una costura de producción, `runAppArgsFn`, sin efecto en el comportamiento.
    - La guarda (#72) hace fallar el paquete si `git status --porcelain --ignored` cambia. Puede dar falsos positivos si otro proceso escribe en el checkout durante la ejecución.
  - **Comprobaciones:**
    - El paquete pasa en varias ejecuciones, entre 4 y 12 s (antes, unos 80 s).
    - El diff del estado del checkout está vacío en cada ejecución.
    - La guarda se probó con un fichero desechable, que la hizo fallar.
    - WSL: PASS.
    - El orquestador repitió la prueba principal sobre el commit del aislamiento.
  - **Riesgo evaluado:** `medium`.
  - **Incidente:** la ejecución de estos tests que hizo el orquestador antes de T3, para confirmar que escribían en el repo, pudo lanzar un `upgrade` real de herramientas de la máquina y registrar el worktree temporal en el hub.
  - **Seguimiento:** los tests de knowledge de `internal/cli` (`TestCLIInitKnowledgeProfile`, `TestCLIKnowledgeSweepAndQuery`, ...) registran directorios temporales en el hub real. Hay 37 de 44 entradas con rutas que ya no existen.

- **T4: completada.** Ruta delegada en un writer.
  - **Dónde se genera el lanzador:** `internal/opencode/background.go`, con la constante `OwnershipMarker` y los generadores para cmd, sh y ps1.
  - **Lectura del destino:** `ManagedLauncherTarget` lee como mucho 4096 bytes de un fichero regular, sin ejecutarlo, y devuelve el destino que envuelve. Exige que la marca esté en la cabecera y que la invocación coincida con el entrecomillado del generador.
  - **En `doctor`:** solo para `opencode`, si el lanzador envuelve otra de las copias, deja de contarlo como duplicado. Si envuelve una ruta ajena o quedan más copias reales, el aviso se mantiene.
  - **Tamaño y división:** 590 líneas en total (unas 392 de tests). Se dividió en dos PRs encadenados: el parser en `internal/opencode` (`3fd5aca5`, 267 líneas) y su uso en `doctor` (`86844846`, 323 líneas). El árbol es idéntico al del commit original, y el commit del parser compila y pasa sus tests por sí solo.
  - **Comprobaciones del writer:** pasan los tests de cli de `doctor`, `CheckOneTool` y los nuevos, y los de `opencode`. En WSL pasa también el test del lanzador POSIX. Con `doctor.go` revertido, fallan los 4 tests de «lanzador excluido».
  - **Riesgo evaluado:** `high`, por la señal `shell_process` en `launcher_target.go`, que analiza scripts sin ejecutarlos. Verificación independiente en curso.
  - **Verificación independiente: PASS.**
    - El parser no ejecuta nada y lee como mucho 4096 bytes.
    - 900.000 destinos aleatorios hacen el round-trip sin ningún fallo.
    - `doctor` sigue avisando en todos los casos de duplicado real.
    - La prueba de extremo a extremo con PATH aislado pasa de `[!!]` a `[ok]` en el caso del bug.
    - La heurística de riesgo saltó por la palabra `exec` dentro del texto del formato POSIX; es un falso positivo.
  - **Corrección acotada `ea558625`** (+104/−8), a raíz de una observación del verificador:
    - Si el lanzador está en el PATH pero queda tapado por una copia anterior, `doctor` vuelve a avisar, porque la variable de background-subagents no se aplica. Antes del arreglo, ese caso habría salido en verde.
    - Remedio nuevo, `RemedyReorderPath` (`reorder-path`), para no reutilizar `remove-duplicate-tools`.
    - Casos de round-trip con `%`, `&`, `'`, unicode y `.PS1`.
  - **Reparto final:**
    - PR #73, el parser con sus casos de round-trip: `13a623ed`, 277 líneas.
    - Rama `fix/doctor-lanzador-opencode`, la exclusión y el aviso de lanzador tapado: `d86f5c83` y `ea558625`, 409 líneas.
    - El árbol es idéntico al ya verificado.
    - **El PR de `doctor` supera el presupuesto en 9 líneas.** No admite un corte cohesivo: separar el aviso de lanzador tapado publicaría temporalmente una regresión, y no se separan tests de su comportamiento. Pendiente de que el usuario acepte `size:exception`.
  - **Preexistente, sin relación con este cambio:** `internal/opencode` `TestRunCatalogCommandCancelsOverflowingChild` falla por tiempos en esta máquina, también sin el cambio.
  - **CI de los PRs ya abiertos:** #69, #70, #71 y #72 están en verde. #72 pasa con `go test ./...` ejecutando los paquetes en paralelo, así que la guarda no da falsos positivos.

### Entrega (2026-10-04)

- **Decisión del usuario:** fusionar en orden los PRs en verde y abrir T2 y `doctor` con `size:exception`.
- **Fusionados con squash:**
  - #69 (hook de Kiro) → `9cf5cfff`
  - #70 (T1) → `c990e57d`
  - #71 (T3, aislamiento) → `ccff924c`
- **#72 (guarda de T3):** se reapuntó a `main` y se rebasó como `f43d3ec8` (127 líneas). CI en curso.
- **#73 cerrado.** Su CI falló en `scripts/deadcode-ratchet.sh`: con el parser separado de su uso, `ManagedLauncherTarget` y sus funciones quedaban sin llamadores. Fue un error del corte del orquestador. T4 se entrega entero en **#74**: parser y `doctor`, 686 líneas, con `size:exception`.
  - **Lección:** este repo rechaza funciones inalcanzables, así que no se puede publicar un PR con código nuevo sin su consumidor.
- **T2 → #75**, 605 líneas, con `size:exception`.
- **Limpieza:** eliminados los worktrees y las ramas de #69, #70, #71 y #73.

- **Fusionados después, con el CI en verde y sin ficheros compartidos:**
  - #72 (guarda de T3) → `7aa6e9bc`
  - #74 (T4) → `7b6b7abd`
  - #75 (T2) → `c504f2e6`
- **Limpieza final:** eliminados los worktrees y las ramas de T2, T3 y T4. Solo queda este PR de cierre.

### Criterios de aceptación: resultado

- ✅ `branch-pr` y `gentle-ai-bench` no aparecen como skills instalables ni en `AGENTS.md`, la documentación activa, los assets, el catálogo, los presets ni la TUI (#70).
- ✅ `ScopeForPath` clasifica como proyecto las rutas que difieren en mayúsculas o usan nombres 8.3 en Windows (#75, con tests ejecutados en Windows sin saltos).
- ✅ `go test ./internal/dashboard/` deja el checkout sin cambios (#71), y la guarda #72 lo impone. Ha pasado en el CI con los paquetes en paralelo.
- ✅ `doctor` no avisa de duplicado por el lanzador gestionado de `opencode`, y avisa si queda tapado en el PATH (#74).
- ✅ El CI está en verde en todos los PRs fusionados.

### Seguimientos que quedan abiertos (fuera de alcance)

- **Skills retiradas en instalaciones existentes:** no hay mecanismo para retirar skills gestionadas que Axiom ya no distribuye. Quien instaló `branch-pr` o `gentle-ai-bench` conserva sus copias, y `uninstall` deja de limpiarlas.
- **Telemetría:** está activada por defecto y envía a `telemetry.gentlemanprogramming.com`, el servidor del upstream.
- **Hub contaminado por tests:** los tests de knowledge de `internal/cli` registran directorios temporales en el hub real (`~/.axiom/workspaces.json`).
- **PATH del usuario:** quedan entradas `AppData\Local\Temp\codegraph-install-test-*`, restos de tests de CodeGraph.
- **`RefreshSkip`:** compara `cwd == home` de forma léxica, así que tiene el mismo hueco con mayúsculas y nombres 8.3 que T2.
- **`AGENTS.md`:** la línea manual sobre la convención `gentle-ai-*` ha quedado obsoleta, y la cabecera sigue diciendo «Gentle AI™».

## Cierre

Fusionar este PR de cierre cuando su CI esté en verde y eliminar su worktree. El ODD queda cerrado.
