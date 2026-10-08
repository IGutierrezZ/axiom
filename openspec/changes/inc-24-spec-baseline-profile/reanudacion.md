# Reanudación de inc-24-spec-baseline-profile

> Estado a 2026-10-04. Documento para retomar el cambio en una sesión nueva. No es un relevo entre roles (`handoff.md`); el kickoff tiene `handoff_policy: none`.

## Estado

| Elemento | Estado |
|---|---|
| Rama | `feat/inc-24-spec-baseline-profile`, en `3a35fc66` (= `main` el 2026-10-04), sin commits |
| Artefactos | Todos sin versionar en `openspec/changes/inc-24-spec-baseline-profile/` |
| Kickoff | `checkpointed`, `handoff_policy: none`, roles `[fullstack:blocking]`, `deployment_target: local` |
| Gates | `spec` aprobada y `design` aprobada (historial completo en `gates.yaml`) |
| Status nativo | `nextRecommended: tasks`, sin bloqueos |
| Siguiente fase | `sdd-tasks`. El usuario la dejó en pausa a propósito, para lanzarla en una sesión nueva |

## Artefactos

| Fichero | Contenido | Engram |
|---|---|---|
| `exploration.md` | Exploración inicial | `sdd/inc-24-spec-baseline-profile/explore` (#433) |
| `proposal.md` | Propuesta aprobada tal cual (D1–D5, S1–S6, Q3–Q16) | `.../proposal` (#437) |
| `spike.md` | Estructura de la baseline del proyecto de referencia (C8) | `.../spike` |
| `specs/*/spec.md` | 11 capacidades, 79 requisitos, 339 escenarios | Índice `.../spec` (#453) y una observación por capacidad |
| `design.md` | Diseño aprobado, 22 decisiones, trazabilidad 79/79 | Índice `.../design` (#460) y secciones |
| `kickoff.yaml`, `gates.yaml` | Configuración sellada y decisiones de las gates | — |

## Decisiones de la sesión

- **Preflight**: ritmo `Interactive` y artefactos `Both` (`hybrid`).
- **Entrega**: `exception-ok`. El usuario aceptó `size:exception` de antemano y quiere los PRs agrupados en unidades con sentido.
- **Encadenado**: `stacked-to-main`. La cadena de PRs del diseño (§12) es A1 → A2 → B1 → B2 → C1 → C2 → C3; B2 y C1 pueden avanzar en paralelo tras B1. La fase de tareas puede refinarla.
- **Tamaño estimado**: de 8.550 a 12.350 líneas de autoría (R13).

## Qué debe hacer la fase de tareas

1. Planificar las tareas por PR según §5 y §12 del diseño, con su previsión de carga de revisión.
2. Incluir como tareas los menores pendientes de la §13 del diseño: m2, m4, m6, m7, m8, m9, m10, m11, m12 y m13.
3. Incluir en A1:
   - el spike de cwd de ejecutores (R17) en Claude Code y OpenCode;
   - el aislamiento de `TestRunSDDStatusJSON`, `TestRunSDDStatusNonexistentChange` y `TestArchiveEndpoints`;
   - el paso de CI "checkout limpio tras `go test ./...`".
4. Cargar las skills `skills/chained-pr/SKILL.md` y `skills/work-unit-commits/SKILL.md`.
5. Al terminar: gate `tasks` (checkpointed), con el plan del rol `fullstack` presentado entero. No lanzar `sdd-apply` hasta que esté aprobada.

## Coordinación y precauciones

- **Sesión concurrente.** Otra sesión del usuario trabaja en worktrees (`C:\repos\axiom-wt\*`), en la limpieza de restos de gentle-ai (ODD `limpieza-gentle-ai-y-seguimientos`): retira skills de `internal/assets/skills`, toca `skillregistry.ScopeForPath`, aísla los tests de dashboard (T3) y ajusta doctor. Esa sesión no toca este checkout. Antes de tareas y antes de implementar, comprueba `git log HEAD..main` y, si la rama sigue sin commits, avánzala con `git merge --ff-only main`.
- **Confidencialidad.** El remoto del repositorio está en GitHub. Los artefactos nuevos llaman al cliente "proyecto de referencia", pero `exploration.md` y `proposal.md` todavía nombran al cliente y sus repositorios. Antes del primer commit, pregunta al usuario si se sanean.
- **TDD.** `strict_tdd: false` (`openspec/config.yaml`): modo estándar, con las pruebas junto al código.
- **Nota para el diseño.** Archive debe avisar si el repo de specs está por detrás de su remoto antes de asignar IDs definitivos. Está en el diseño como información; no bloquea.
