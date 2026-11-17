# ODD: Limpieza de restos del «Engram compartido» (C-05-a)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/limpiar-engram.md`.
> Espejo de recuperación en Engram: topic `odd/limpiar-engram/tasks`, proyecto `axiom`.
> Origen: `docs/plan/backlog-2026-11/05-conocimiento-equipo.md` (C-05-a) y decisión P7 del README del plan (ampliar).

## Objetivo

El repositorio versiona memoria de Engram heredada de upstream y varias guías invitan a compartirla con `engram sync` o a hacer `git fetch` sin autorización. Ambas cosas contradicen lo decidido (compartir la memoria de Engram entre personas está descartado; el fetch solo con autorización). Resultado esperado: no queda memoria de terceros versionada, no hay documentación del flujo de equipo con `engram sync` y ninguna guía del repositorio ordena un fetch sin autorización.

## Alcance autorizado

- Borrar `.engram/chunks/*.jsonl.gz` (5 ficheros) y `.engram/manifest.json`; conservar `.engram/config.json`.
- `.gitignore:53-55`: cambiar el comentario e ignorar también `.engram/chunks/` y `.engram/manifest.json`.
- `docs/engram.md`: quitar la sección «Team Sharing» (`:50-62`) y la entrada de `engram sync` de los comandos esenciales (`:24-25`).
- `docs/intended-usage.md:30-31,172` y `docs/codebase/sync-and-cloud.md:12,39,52`: dejar de presentar `engram sync` como reparto de memoria.
- `AGENTS.md:17` y `GEMINI.md:48`: «fetch solo con autorización de la persona».
- `docs/assets/diagrams/engram-memory.svg:3,91,93`: añadido por decisión del usuario (2026-11-17, «Sí, inclúyelo»). No figuraba en la lista de ficheros de C-05-a ni en P7, pero el objetivo y el criterio `rg -n "engram sync" docs/` lo alcanzan.

## Fuera de alcance

- Reescribir el historial de git.
- `internal/assets/skills/_shared/persistence-contract.md` (lo recoge C-01-b).
- `governance.shared_memory` (`axiom.yaml:31`, `internal/workspace/types.go:46`).
- Nombrar el interruptor del autofetch en `AGENTS.md`/`GEMINI.md` (es de C-05-d).

## Restricciones

- TDD: desactivado (`openspec/config.yaml`, `strict_tdd: false`); comprobaciones funcionales ordinarias. Runner: `go test`.
- Estrategia de entrega: `exception-ok` (una sola PR; `size:exception` si supera 400 líneas). Previsión: muy por debajo de 400 líneas de autor (los `.jsonl.gz` son binarios borrados).
- Commits convencionales sin atribución de IA.

## Tareas

- [x] **T1 · Retirar la memoria versionada** (`.engram/chunks/`, `.engram/manifest.json`, `.gitignore`). Ruta: inline (mecánico, 1 fichero de texto).
- [x] **T2 · Alinear la documentación de Engram** (`docs/engram.md`, `docs/intended-usage.md`, `docs/codebase/sync-and-cloud.md`). Ruta: inline (ediciones mecánicas y acotadas ya localizadas).
- [x] **T3 · Fetch solo con autorización** (`AGENTS.md:17`, `GEMINI.md:48`). Ruta: inline (una línea por fichero).
- [x] **T4 · Diagrama `docs/assets/diagrams/engram-memory.svg`**: reescribir solo las cadenas `:3`, `:91` y `:93`, que presentan `engram sync` como reparto con el equipo, sin cambiar la geometría. Lo enlaza `README.md:67`. Ruta: inline (cadenas de texto). También se cambió el título de esa franja (`:90`, «Hand it to the team» → «What the team shares»), que formaba parte del mismo mensaje. El texto nuevo de `:91` es más corto que el original, así que cabe en la franja.
- [ ] **T5 · Verificar el criterio de terminado** (ver «Comprobaciones»).

## Comprobaciones (criterio de terminado)

- `git ls-files .engram` devuelve solo `.engram/config.json`.
- `rg -n "engram sync" docs/` no presenta `engram sync` como flujo de equipo.
- `rg -n "pasivamente .git fetch" AGENTS.md GEMINI.md` no devuelve nada.
- `mem_current_project` desde el repositorio sigue devolviendo `project: axiom` con `project_source: config`.
- `go test ./internal/workspace/...` pasa.

## Progreso

- Worktree: `C:\repos\axiom-wt\limpiar-engram\axiom`, rama `inc/limpiar-engram`.
- Plan stop: approved 2026-11-17; implementation continues

## Siguiente paso

Presentar la parada de plan.
