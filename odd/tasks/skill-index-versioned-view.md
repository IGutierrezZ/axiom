# ODD: Vista versionada del índice de skills en AGENTS.md

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/skill-index-versioned-view.md`.
> Espejo de recuperación en Engram: topic `odd/skill-index-versioned-view/tasks`, proyecto `axiom`.
> Rama: `fix/skill-index-versioned-view` (desde `origin/main` `305010c9`).

## Objetivo

Que el bloque `axiom:skills-index` de `AGENTS.md`, que está versionado, sea determinista y portable entre clones. Debe listar solo skills versionables del propio repositorio, con rutas relativas canónicas. El registro local `.atl/skill-registry.md` conserva la vista completa de la máquina.

## Problema

`Regenerate` (`internal/skillregistry/registry.go`) alimenta los tres destinos (`.atl`, `AGENTS.md` y Engram) con un único escaneo. Ese escaneo recorre directorios locales de cada máquina: `.claude/skills` y `.gemini/skills`, generados por `axiom setup`, y los directorios de `$HOME`. Consecuencias observadas:

- Filas que cambian entre `.gemini/skills/…` y `.claude/skills/…` según la máquina. En un clon limpio son enlaces rotos. `main` ya tiene commiteadas 5 filas que apuntan a `.gemini/skills/…`.
- Filas con rutas absolutas de usuario (p. ej. `C:\Users\<usuario>\.config\opencode\skills\…`).
- `issue-creation` desaparece al regenerar, porque `internal/assets/skills` no es raíz de escaneo. Su fila está escrita a mano y `TestIssueCreationAuthorityBoundary` exige un texto literal que no coincide con el frontmatter.

## Por qué

`AGENTS.md` se versiona y lo consumen todos los agentes y colaboradores. Cada regeneración introduce ruido en los diffs y rompe enlaces. Además, obliga a parches manuales, como el de #60 (`0c2196b4`).

## Alcance

- **Vista versionada** para `AGENTS.md` y el espejo Engram. Una entrada entra solo si:
  - su `SKILL.md` está dentro del workspace (`cwd`);
  - y git no la ignora (`git check-ignore`, sin `--no-index`, de modo que un fichero ignorado pero trackeado cuenta como versionado).
- **Deduplicación por nombre dentro de la vista versionada:** se descartan las entradas no versionables antes de deduplicar, de modo que gana la copia canónica.
- **Vista completa** (comportamiento actual) para `.atl/skill-registry.md`.
- **Sin git o fuera de un repositorio git:** todas las entradas dentro de `cwd` cuentan como versionables, como hasta ahora salvo `$HOME`.
- **Caché:** la huella (`Fingerprint`) debe invalidarse cuando cambia qué entradas son versionables, por ejemplo al editar `.gitignore`. Si cambia la salida, se sube `RegistrySchema`.
- **Raíces versionadas declaradas:** nueva clave `workspace.skill_roots` en `axiom.yaml`, con rutas relativas a `cwd`; se rechazan las absolutas y las que salen de `cwd`. Estas raíces se escanean justo después de `skills/` y antes de los directorios propios de cada agente. En este repo se declara `internal/assets/skills`.
- **Repositorio:**
  - regenerar el bloque de `AGENTS.md`;
  - adaptar `TestIssueCreationAuthorityBoundary` para que compruebe que existe una fila `issue-creation` cuya ruta es `internal/assets/skills/issue-creation/SKILL.md`, en lugar del texto literal;
  - actualizar REQ-22.11 en `openspec/specs/axiom-skills-index-governance/spec.md` y la documentación afectada.

## Restricciones

- Fuera de alcance: cambiar el orden estático de `ProjectSkillDirs("")`, porque está fijado por `internal/assets/skill_registry_plugin_guard_test.go:90-100` y el plugin `opencode/plugins/skill-registry.ts`. Tampoco se modifican la exclusión por nombre (`_shared`, `skill-registry`, `sdd-*`) ni el formato de fila.
- Hay que mantener la idempotencia byte a byte de REQ-22.12 y la no fatalidad de Engram de REQ-22.11.
- Una sola llamada a git por regeneración. Si git falla, se aplica la vista sin filtro git, nunca un error fatal.
- Heurística orientativa de unas 400 líneas por tarea. No es un límite, ni justifica recortar tests o comentarios.

## Configuración de ejecución

- **TDD:** desactivado. Fuente: `openspec/config.yaml:16` (`strict_tdd: false`). Runner: `go test`. Se ejecutan las comprobaciones funcionales ordinarias.
- **Estrategia de entrega:** `ask-on-risk`. Previsión: unas 500 líneas de autor, superior al presupuesto de 400. **Estrategia de encadenado: `stacked-to-main`** (decisión del usuario).
- **Slices:**

  | PR | Contenido | Rama | Base |
  |---|---|---|---|
  | 1 | T1 | `fix/skill-index-versioned-view` | `main` |
  | 2 | T2a: declarar y validar `workspace.skill_roots` | `fix/skill-index-skill-roots-config` | PR 1 |
  | 3 | T2b: escanear las raíces declaradas, con docs | `fix/skill-index-declared-roots` | PR 2 |
  | 4 | T3 | `fix/skill-index-apply-repo` | PR 3 |

  T2 se ha partido en dos PRs porque sumaba 521 líneas y admitía un corte cohesivo: T2a tiene 297 líneas y T2b, 224.
- **`size:exception` aceptado para PR 1 (T1)** por el usuario y mantenedor el 2026-10-02.
  - Motivo: unas 860 líneas de autor (código y tests de la vista versionada, más la corrección D1-D3). Tras un intento honesto, no admite un corte cohesivo: el núcleo con sus tests también supera las 400 líneas, y separar los tests de su comportamiento no forma una unidad de trabajo.
  - La etiqueta `size:exception` se aplicará al abrir PR 1.
- **RDD:** desactivado (`rdd_mode: off`). Se evalúa el riesgo por commit con `axiom review assess`, sin revisión.

## Tareas

- [x] **T1 · Vista versionada en `internal/skillregistry`** (`bb72a460`, más la corrección `37bdb403`)
  - Clasificación versionable/no versionable con una sola llamada `git check-ignore -z --stdin` y su fallback.
  - Separar las entradas de `.atl` (completas) de las de `AGENTS.md` y Engram (versionadas). Deduplicar en cada vista. Incluir la versionabilidad en la huella y subir `RegistrySchema` si cambia la salida.
  - Tests:
    - copias en `.claude/skills` ignoradas que quedan fuera, con lo que gana la canónica;
    - un proyecto que versiona `.claude/skills` y conserva sus filas;
    - skills de `$HOME` fuera de `AGENTS.md` pero presentes en `.atl`;
    - un directorio sin git que conserva el comportamiento;
    - un cambio de `.gitignore` que invalida la caché;
    - idempotencia.
  - Ruta: delegada (writer). Disparadores: preparación de escritura y 2 o más ficheros no triviales (`registry.go`, `agents.go`, tests).
- [x] **T2 · Raíces versionadas declaradas en `axiom.yaml` (`workspace.skill_roots`)**: T2a `8affb459` y T2b `0454b7dd`
  - Campo en `internal/workspace`, carga en `ProjectSkillDirs(cwd)` inmediatamente después de `skills/` y validación de rutas relativas contenidas.
  - Tests en `internal/workspace` e `internal/skillregistry`.
  - Ruta: delegada (writer). Disparador: 2 o más ficheros no triviales.
- [ ] **T3 · Aplicar en el repositorio**
  - Declarar `internal/assets/skills` en `axiom.yaml`.
  - Regenerar `AGENTS.md` y adaptar `TestIssueCreationAuthorityBoundary`.
  - Actualizar REQ-22.11 y los escenarios en la spec viva, y la documentación afectada.
  - Ruta: delegada (writer). Disparador: preparación de escritura sobre 3 o más ficheros.

## Criterios de aceptación

- En un clon limpio, `axiom skill index refresh` produce un bloque `AGENTS.md` idéntico con o sin `.claude/skills` y `.gemini/skills` locales presentes.
- El bloque no contiene rutas fuera del repositorio ni rutas ignoradas por git.
- La fila `issue-creation` apunta a `internal/assets/skills/issue-creation/SKILL.md` y `TestIssueCreationAuthorityBoundary` pasa.
- `.atl/skill-registry.md` mantiene la vista completa.
- Pasan `go test ./internal/skillregistry/... ./internal/workspace/... ./internal/assets/... ./internal/cli/... ./internal/app/...` y `go run ./internal/gofmtcheck`.

## Progreso

- Exploración completada. El mapa del generador ha sido delegado en un agente de exploración.
- Decisiones del usuario: carril ODD; criterio de versionabilidad «no ignorada por git»; encadenado `stacked-to-main`.
- **T1: commit `bb72a460`**, ruta delegada.
  - Cambios: `registry.go` +17/−3; `versioned.go` nuevo (170 líneas); `versioned_test.go` nuevo (439 líneas, 11 tests). En total, 629 líneas de autor.
  - Decisiones: `SkillCount` cuenta la vista completa; `RegistrySchema` no cambia porque el formato de `.atl` es el mismo (la huella nueva provoca una única regeneración al actualizar); el runner git es la variable de paquete `runGitCheckIgnore`, con un timeout de 5 s; los symlinks se clasifican por la propia entrada del enlace.
  - Comprobaciones del writer:
    - `gofmt` y `go vet` limpios.
    - Pasan `go test ./internal/skillregistry/...`, los tests filtrados de `app`, `cli` y `assets`, y `gofmtcheck`.
    - WSL Linux con git 2.53: PASS.
    - `go test ./internal/dashboard/` falla solo en `TestSemanticEndpoints` («CodeGraph not initialized»). Es un fallo del entorno: también ocurre con el cambio retirado.
  - Riesgo evaluado: **high** (lanza un proceso). Con RDD desactivado, eso exige un verificador independiente además de la autoverificación del writer.
  - Comprobaciones del verificador independiente:
    - Extremo a extremo con `axiom.exe` en un repo temporal: la vista versionada, la idempotencia, el cambio de `.gitignore`, el fallback sin git y sin binario, los ficheros ignorados pero trackeados, la deduplicación y `$HOME`.
    - `.atl` es byte-idéntico al de la base.
  - Defectos confirmados por el verificador, resueltos en la corrección acotada **`37bdb403`** (+225/−7):
    - **D1:** un submódulo devolvía 128 en `check-ignore` y desactivaba todo el filtro. Ahora `gitVisiblePath` corta en el primer ancestro con `.git`.
    - **D2:** en Windows el timeout no acotaba la llamada, porque `git.exe` es un lanzador. Ahora se fija `WaitDelay` a 1 s.
    - **D3:** las variables `GIT_*` heredadas no se limpiaban. Ahora `gitCheckIgnoreEnvironment` elimina las de ubicación del repositorio.
  - **Nueva verificación independiente: PASS.**
    - D1, D2 y D3 se han repetido con sus reproducciones: D2 vuelve en 6,7 s, frente a los 21,6 s de antes, con un lanzador simulado.
    - La pasada de extremo a extremo no muestra regresiones.
    - `go test ./internal/skillregistry/...` pasa, con 4 tests nuevos.
    - Los tests de symlink solo se ejecutan en WSL, porque esta cuenta de Windows no tiene permiso para crear symlinks.
  - **Resultado para T1:** riesgo `high`; con RDD desactivado, verificador independiente con resultado PASS.
  - **Seguimiento fuera de alcance:** D4, `ScopeForPath` distingue mayúsculas en Windows. Es preexistente, pero con este cambio una raíz absoluta escrita con distinta capitalización desaparece en silencio de `AGENTS.md`.
  - **Presupuesto:** PR 1 supera las 400 líneas (unas 630 + la corrección). Tras un intento honesto de partirlo, el núcleo (registro, clasificación y tests básicos) también lo supera, y separar los tests de su comportamiento no es una unidad de trabajo. Se recomienda `size:exception` para PR 1.
- **Consecuencias para T3:**
  - Con el criterio «no ignorada», `.gemini/`, `.kiro/` y `.agents/`, que están sin trackear y sin ignorar en `main`, seguirían generando filas. T3 debe asegurar que estén ignorados: llega con #62 (`.gemini/` y `.agents/tasks/`) o se añade en T3.
  - `go test ./internal/dashboard/` reescribe `AGENTS.md` y `openspec/INDEX.md` y crea `.gemini/` y `.kiro/` en la raíz. No hay que ejecutarlo antes de regenerar, o hay que limpiar después.
  - Cambio de comportamiento: las raíces multirrepo fuera de `cwd` (`../specs` o rutas absolutas) dejan de aparecer en `AGENTS.md` y Engram, aunque se mantienen en `.atl`.

- **T2: completada.** Ruta delegada. El writer lo entregó en un único commit de 521 líneas, que el orquestador ha dividido en dos sin cambiar el contenido (mismo árbol, `8efd71ce`):
  - **T2a `8affb459`** (297 líneas): `workspace.SkillRoots` (`yaml:"skill_roots,omitempty"`) y `CleanSkillRoot`.
    - `CleanSkillRoot` aplica la misma regla en todos los sistemas operativos: rechaza `/x`, `\x`, `C:\x`, `C:x`, UNC, la cadena vacía, `.`, `..` y `../x`.
    - Añade `ValidSkillRoots`, y `workspace.Validate` (detrás de `axiom workspace validate`) reporta cada raíz inválida como error.
    - Tests en `skillroots_test.go`, `loader_test.go` y `validator_test.go`.
  - **T2b `0454b7dd`** (224 líneas):
    - `ProjectSkillDirs(cwd)` carga la configuración una sola vez y escanea las raíces declaradas justo después de `skills/` y antes de los directorios de agentes. Las inválidas se ignoran.
    - `ProjectSkillDirs("")` no cambia, y el test de guarda sigue en verde.
    - Tests en `declared_roots_test.go`, incluido el caso en que la raíz declarada gana a copias en `.claude` (ignorada) y en `.gemini` (sin trackear).
    - Docs en `axiom.example.yaml`, `docs/manual-de-inicio.md` y `docs/skill-registry.md`.
  - Comprobaciones del writer:
    - `gofmt` y `vet` limpios.
    - Pasan `go test ./internal/skillregistry/... ./internal/workspace/...`, el test de guarda de `assets` y los tests filtrados de `app` y `cli`.
    - `gofmtcheck` limpio; en WSL pasan `skillregistry` y `workspace`.
  - Comprobaciones del orquestador:
    - Rebase limpio sobre `1d4f8fe5`.
    - Pasan `go test ./internal/skillregistry/... ./internal/workspace/...` y el test de guarda.
    - T2a compila y sus tests pasan por sí solo en un worktree temporal.
    - Revisión del diff.
  - **Riesgo evaluado:** `medium`. Con RDD desactivado basta la autoverificación del writer más la comprobación del orquestador.
  - **Notas para T3:**
    - `docs/skill-registry.md` no describe todavía la vista versionada de T1; hay que añadirla junto a «Declared Skill Roots».
    - `ProjectSkillDirs("")` lee `axiom.yaml` del cwd del proceso, y eso ya ocurría antes de este cambio.
    - El plugin de OpenCode (`PROJECT_MARKERS`) no ve las raíces declaradas, la misma limitación que con specs y roles.

## Siguiente paso

T3: declarar `internal/assets/skills` en `axiom.yaml` y asegurar que `.gemini/`, `.kiro/` y `.agents/` estén ignorados. Después, regenerar `AGENTS.md`, adaptar `TestIssueCreationAuthorityBoundary`, actualizar REQ-22.11 y documentar la vista versionada.
