# ODD: Seguimientos del merge de overlays de OpenCode, del arnés crosslane y del scope de `axiom sync`

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/seguimientos-merge-overlays-y-sync.md`.
> Espejo de recuperación en Engram: topic `odd/seguimientos-merge-overlays-y-sync/tasks`, proyecto `axiom`.
> **Estado: LANZADO el 2026-10-12 por el carril ODD.** Rama del documento: `docs/odd-seguimientos-merge-sync` (worktree `axiom-wt/odd-seguimientos-merge-sync-docs`). W1 en curso.
> Origen: cierre del ODD `marca-restos-persona-overlays` (`odd/tasks/marca-restos-persona-overlays.md`). Al ejecutar `axiom sync` tras fusionar #130-#134 salieron S1 y S3; S2 ya estaba anotado como seguimiento menor.

---

## 0. Cómo lanzarlo o retomarlo

1. Recupera el contexto: `mem_context`; `mem_search "odd/seguimientos-merge-overlays-y-sync"` en el proyecto `axiom`; `mem_get_observation` sobre la observación de ese topic. Lee **este fichero**, que es la fuente autoritativa.
2. Concilia con la realidad: `git -C C:\repos\axiom fetch --prune origin`, `git -C C:\repos\axiom worktree list` y `gh pr list --repo IGutierrezZ/axiom --state open`. Comprueba que las líneas citadas en la sección 4 siguen siendo válidas en `origin/main`.
3. Pregunta el carril (ODD o SDD) antes de crear ramas; S1 toca la semántica del merge y puede justificar SDD.
4. Empieza por la sección 6 (primera tarea sin marcar) y respeta las secciones 2, 3 y 7.

## 1. Objetivo

Hacer que `axiom sync` retire de verdad las reglas gestionadas obsoletas de las instalaciones existentes. Eliminar la última tolerancia a la grafía `gentle-ai` del arnés crosslane. Conseguir que el scope efectivo de `axiom sync` sea visible para que nadie crea haber actualizado la instalación global cuando solo se actualizó el workspace.

## 2. Alcance y restricciones

- **Dentro:** S1, S2 y S3 (sección 4).
- **Fuera:** `contracts/review-integration/` (v1 congelado, FREEZE.md), el dialecto heredado de `internal/cli/review_dialect.go`, los marcadores que se escriben hoy y los IDs de contrato.
- S1 no puede borrar reglas de permiso que haya añadido el usuario. El merge de `permission` es deliberadamente conservador (`json_merge.go:900`, `permission || key == "permission"`). Hay que retirar solo claves gestionadas heredadas, enumeradas de forma explícita.
- Un cambio en `internal/assets/**` cambia `assets.ManagedDigest()`: hay que avisar en el PR. El hash fijado de Kilocode (`internal/components/sdd/review_ledger_contract_test.go`) cambia si el resultado renderizado para Kilocode cambia.
- Push, PR y merge solo con autorización explícita del usuario en cada caso.

## 3. Convenciones (heredadas de `marca-restos-persona-overlays`, sección 3)

- TDD desactivado (`openspec/config.yaml strict_tdd: false`); runner `go test -run`. Nunca `go test ./internal/cli/` completo; `internal/components/sdd` con `-timeout 40m`.
- RDD desactivado. Riesgo por PR con `axiom review assess --cwd . --base-ref origin/main --committed-only --json`; con `high`, verificador independiente de solo lectura.
- Un PR por unidad contra `main`, con la plantilla en castellano y una sola etiqueta `type:*`. Commits en Conventional Commits, en castellano y sin atribución. Merge por squash cuando el usuario diga «revisa si ya está verde y continúa».
- Goldens: `go test ./internal/components/ -run Golden -update`. Subagentes con `model: sonnet`.
- Verificar siempre la instalación real con `axiom sync --scope global`: el scope por defecto es `workspace` (ver S3).

## 4. Inventario (comprobado sobre `main` `8cf608bd` el 2026-10-11)

| ID | Qué | Dónde | Riesgo |
|---|---|---|---|
| S1 | **Bug:** el merge de JSON de los overlays de OpenCode es un deep-merge que nunca borra claves de la base. Una regla de permiso retirada del overlay sobrevive para siempre en las instalaciones existentes. Caso real: tras #132 y `axiom sync --scope global`, `~/.config/opencode/opencode.json` sigue teniendo `agent.review-validator.permission.bash["gentle-ai review inspect-candidate --purpose targeted-validation *"] = "allow"`, aunque el prompt sí se actualizó a `axiom`. El PR #132 decía que `axiom sync` la retiraba, y no es así. | `internal/components/sdd/inject.go:2670-2722` (`mergeOpenCodeJSONFile` → `mergeJSONFileContents`), migraciones heredadas en `:2704-2715`, `removeManagedOpenCodeAgentTools` en `:2735` (solo quita `tools`), `internal/components/filemerge/json_merge.go:855-909` (`mergeObjectScope`), escape `__replace__` en `:836-849` | medio |
| S2 | El arnés crosslane acepta todavía la grafía `gentle-ai review inspect-candidate` en el prompt hijo del validador. Es inocuo (no es un permiso), pero es el último resto de esa grafía fuera del dialecto heredado y de las listas de cadenas prohibidas. | `scripts/crosslane/opencode.go:357` | bajo |
| S3 | **UX:** `axiom sync` usa por defecto `--scope workspace` (`internal/cli/scope.go:27-39`). Desde un repo escribe en `<repo>/.config/opencode/` e informa «no managed sync actions needed», sin indicar el scope ni la ruta. El scope no se guarda en `~/.axiom/state.json`. Resultado: la instalación global de OpenCode llevaba sin tocarse desde el 2026-09-28 sin que nadie se diera cuenta. | `internal/cli/sync.go:1822-1824` (no-op = 0 ficheros cambiados), `:2269-2275` (mensaje), `internal/cli/run.go:2635-2678` (rutas destino), `sync.go:1844-1857` (digest solo en scope no-workspace) | bajo |

## 5. Decisiones del usuario

- D0 (2026-10-12): carril **ODD**, elegido por el usuario.
- S1 (2026-10-12): **migración de claves heredadas enumeradas.** La fija la restricción de la sección 2 (no borrar reglas del usuario); `__replace__` las pisaría, así que queda descartado.
- (pendiente) S3: ¿basta con mostrar el scope y la ruta efectiva en el informe de sync, o se quiere además guardar el scope en el estado o avisar si el global está desfasado?

## 6. Tareas (commit → PR → merge)

- [ ] **W1 · S1 · Retirar reglas gestionadas obsoletas** (rama `fix/opencode-retirar-permisos-heredados`, worktree `axiom-wt/w1-retirar-permisos-heredados`; ruta: delegada, disparador de escritor: `inject.go` + test): una migración en `mergeOpenCodeJSONFile`, junto a `migrateLegacyOpenCode*`, que elimine de los agentes gestionados las claves de permiso heredadas enumeradas, empezando por `gentle-ai review inspect-candidate --purpose targeted-validation *`. Tests: una base con la clave heredada y otra personalizada del usuario; tras el merge solo desaparece la heredada. Verificar con `axiom sync --scope global` que desaparece de `~/.config/opencode/opencode.json`.
- [ ] **W2 · S2 · Arnés crosslane:** exigir solo `axiom review inspect-candidate` en `scripts/crosslane/opencode.go:357` y ajustar el mensaje de fallo.
- [ ] **W3 · S3 · Scope visible en sync:** incluir el scope y la ruta raíz efectiva en el informe de `axiom sync`, también en el de no-op y en el de `--dry-run`. Lo demás, según la decisión del usuario.
- [ ] **W4 · Cierre:** actualizar este documento y abrir el PR `type:docs`.

Previsión: unas 80-200 líneas en total, sobre todo tests de S1.

## 7. Criterios de aceptación

- Tras `axiom sync --scope global`, ninguna instalación existente conserva reglas de permiso gestionadas que el overlay ya no declara, y las reglas añadidas por el usuario se mantienen.
- `rg "gentle-ai review inspect-candidate"` solo aparece en el dialecto heredado, en los contratos congelados y en las listas de cadenas prohibidas de los tests.
- El informe de `axiom sync` dice qué scope y qué ruta ha sincronizado.
- Build, tests dirigidos y goldens en verde; el CI de cada PR en verde antes de fusionar.

## 8. Progreso

- 2026-10-11: documento creado al cerrar `marca-restos-persona-overlays`, a petición del usuario, para que el seguimiento no se quede sin hacer.
  - Evidencia de S1 y S3: `axiom sync` desde el repo informó de un no-op (scope workspace). `axiom sync --scope global` actualizó `~/.config/opencode/opencode.json` (23 entradas `__managed_by: axiom`, prompt del validador con `axiom`), pero la clave `gentle-ai` siguió en la línea 157.
  - La investigación la hizo un subagente sonnet de solo lectura.
- Instalación accidental en el workspace: `C:\repos\axiom\.config\opencode\` (ignorada por git, `.gitignore:38`) también conserva la clave `gentle-ai`. Se puede borrar a mano; no forma parte del repo.
- 2026-10-12: lanzado. Conciliación: `origin/main` en `32683e0a`; los ficheros de la sección 4 no han cambiado desde `8cf608bd`. D0 = ODD; S1 = migración enumerada. Creados los worktrees del documento y de W1 desde `origin/main`.
