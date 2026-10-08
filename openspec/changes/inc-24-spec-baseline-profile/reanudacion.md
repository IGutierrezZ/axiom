# Reanudación de inc-24-spec-baseline-profile

> Estado a 2026-10-09. Documento para retomar el cambio en una sesión nueva. No es un relevo entre roles (`handoff.md`); el kickoff tiene `handoff_policy: none`.

## Estado

| Elemento | Estado |
|---|---|
| Rama | `feat/inc-24-spec-baseline-profile`, en el worktree `C:\repos\axiom-wt\inc-24-spec-baseline-profile`, rebasada el 2026-10-09 sobre `origin/main` 770914de. Copia de seguridad previa al rebase: `backup/inc-24-pre-rebase` |
| Sesión concurrente | Ya fusionó en main (T3 como #71/#72, retirada de skills y resto). Ya no aplica `git merge --ff-only main`: la rama avanza solo con commits propios sobre main |
| Artefactos | Versionados en `openspec/changes/inc-24-spec-baseline-profile/` |
| Kickoff | `checkpointed`, `handoff_policy: none`, roles `[fullstack:blocking]`, `deployment_target: local` |
| Gates | `spec`, `design` y `tasks` aprobadas (historial completo en `gates.yaml`) |
| Progreso de tareas | 49/186 casillas marcadas según `axiom sdd status` (nextRecommended: `apply`). Las 4 de la fase 0 y las 37 de A1 están hechas (41 tareas); A2 está en curso, con 8 de las 19 tareas 2.x marcadas |
| Tramo A2 | Hechos 2 commits de `internal/livingdoc` (`feat(livingdoc): resolver el catálogo con Locate…` y `fix(livingdoc): tomar el nombre del proyecto…`). Hay trabajo en curso en `internal/knowledge/` sin confirmar (`planner.go`, `query.go`, `sweep.go`, `types.go`, `main_test.go`, `specs_root.go`, `specs_root_test.go`); no se ha hecho commit de ese WIP |

## Artefactos

| Fichero | Contenido | Engram |
|---|---|---|
| `exploration.md` | Exploración inicial | `sdd/inc-24-spec-baseline-profile/explore` (#433) |
| `proposal.md` | Propuesta aprobada tal cual (D1–D5, S1–S6, Q3–Q16) | `.../proposal` (#437) |
| `spike.md` | Estructura de la baseline del proyecto de referencia (C8) | `.../spike` |
| `specs/*/spec.md` | 11 capacidades, 79 requisitos, 339 escenarios | Índice `.../spec` (#453) y una observación por capacidad |
| `design.md` | Diseño aprobado, 22 decisiones, trazabilidad 79/79 | Índice `.../design` (#460) y secciones |
| `tasks.md` | Plan del rol `fullstack` (186 tareas, 7 tramos) | `.../tasks` |
| `kickoff.yaml`, `gates.yaml` | Configuración sellada y decisiones de las gates | — |

## Decisiones de la sesión

- **Preflight**: ritmo `Interactive` y artefactos `Both` (`hybrid`).
- **Entrega**: `single-pr` con `size:exception` aprobado de antemano (ver cabecera de `tasks.md`). La cadena A1 → A2 → B1 → B2 → C1 → C2 → C3 del diseño (§12) se conserva como tramos de revisión.
- **Tamaño estimado**: de 8.660 a 12.900 líneas de autoría (R13).

## Cómo retomar

1. Trabajar en el worktree `C:\repos\axiom-wt\inc-24-spec-baseline-profile`, no en `C:\repos\axiom`.
2. Comprobar `git log --oneline origin/main..HEAD` y `git status --short`; el WIP de `internal/knowledge/` es el tramo A2 interrumpido (tareas 2.x del commit A2-c3).
3. Continuar por `tasks.md`: skills `go-testing`, `work-unit-commits` y `internal/assets/skills/chained-pr/SKILL.md`. `gentleman-bubbletea` no está en el registro: las pruebas de TUI usan `Model.Update` directo.
4. `bash scripts/deadcode-ratchet.sh` analiza con `GOOS=linux`; ante un fallo, compararlo con `origin/main` en un `git worktree` temporal.

## Precauciones

- **Confidencialidad.** El remoto del repositorio está en GitHub. Los artefactos nuevos llaman al cliente «proyecto de referencia», pero `exploration.md` y `proposal.md` todavía nombran al cliente y sus repositorios; la tarea 0.2 recoge el saneamiento y el alcance del commit de artefactos.
- **TDD.** `strict_tdd: false` (`openspec/config.yaml`): modo estándar, con las pruebas junto al código.
- **Nota para el diseño.** Archive debe avisar si el repo de specs está por detrás de su remoto antes de asignar IDs definitivos. Está en el diseño como información; no bloquea.
