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
- **Estrategia elegida por el usuario: PRs independientes contra `main`**, uno por tarea y sin dependencias entre ellos. Este documento viaja en el PR de T1; los demás PRs no lo tocan. Al final se cierra con un PR pequeño de docs.

  | PR | Tarea | Rama | Worktree |
  |---|---|---|---|
  | — | T1 | `chore/retirar-skills-gentle-ai` | `axiom-wt/odd-gentle-t1` |
  | — | T2 | `fix/scope-for-path-windows` | `axiom-wt/odd-gentle-t2` |
  | — | T3 | `test/dashboard-aislado-del-repo` | `axiom-wt/odd-gentle-t3` |
  | — | T4 | `fix/doctor-lanzador-opencode` | `axiom-wt/odd-gentle-t4` |
  | — | Cierre | `docs/odd-limpieza-gentle-ai-cierre` | — |

- **Ejecución:** un writer cada vez, de forma secuencial.
- **RDD:** desactivado. Se evalúa el riesgo por commit con `axiom review assess`.

## Tareas

- [ ] **T1 · Retirar `branch-pr` y `gentle-ai-bench`**
  - Borrar `internal/assets/skills/branch-pr/` y `internal/assets/skills/gentle-ai-bench/`.
  - Quitar sus IDs de `internal/model/types.go`, `internal/catalog/skills.go`, `internal/components/skills/presets.go` y `internal/tui/screens/skill_picker.go`.
  - Documentación: `docs/components.md`. En `docs/usage.md`, el ejemplo de `--skill` pasa a usar `work-unit-commits`.
  - Tests y `e2e/e2e_test.sh` (inventario en «Notas de exploración»).
  - Instalaciones existentes: si sync, install o uninstall tienen un mecanismo para retirar skills gestionadas, registrar estos dos IDs. Si no lo tienen, documentarlo como seguimiento.
  - Regenerar `AGENTS.md` sin el espejo Engram.
  - Añadir una fila de descarte deliberado en `docs/upstream-absorption-ledger.md`, respetando su formato y sus reglas.
  - Ruta: delegada (writer). Disparadores: preparación de escritura y 2 o más ficheros no triviales.
- [ ] **T2 · `ScopeForPath` correcto en Windows**
  - En Windows, comparar prefijos sin distinguir mayúsculas y normalizar los nombres cortos 8.3 cuando la ruta exista. Sin cambios en otros sistemas.
  - Tests: diferencias de mayúsculas en la unidad y en los directorios, y nombre corto 8.3 si se puede obtener en el test.
  - Ruta: delegada (writer).
- [ ] **T3 · Tests de `dashboard` aislados del repositorio**
  - Localizar los tests que resuelven la raíz del repo o el cwd y escriben en él, y pasarlos a fixtures en `t.TempDir()` o a stubs.
  - `TestSemanticEndpoints` se salta cuando CodeGraph no está disponible o inicializado.
  - Verificación: ejecutar el paquete en un worktree limpio y comprobar que `git status --porcelain --ignored` sigue vacío.
  - Ruta: delegada (writer). Disparadores: exploración de 4 o más ficheros y preparación de escritura.
- [ ] **T4 · `doctor` reconoce el lanzador gestionado de `opencode`**
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

## Siguiente paso

T1, delegada en un writer.
