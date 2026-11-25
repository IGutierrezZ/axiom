# ODD: Publicar también tras spec y design en `continuous` + `auto` (publicar-spec-design)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/publicar-spec-design.md`. Espejo Engram: `odd/publicar-spec-design/tasks`.
> Rama: `inc/publicar-spec-design` · worktree `C:\repos\axiom-wt\publicar-spec-design\axiom`. Cambio C-05-b del plan maestro (`docs/plan/backlog-2026-11/05-conocimiento-equipo.md`), ola 7.

## Objetivo

En SDD `continuous` con `docs_publication: auto`, publicar los documentos del cambio también al terminar `spec` y al terminar `design`, además de al cerrar `tasks` y al archivar.

## Problema y porqué

Hoy, con la publicación autorizada, spec y design no llegan al equipo hasta que cierra `tasks`. Decisión del usuario: la documentación tiene que quedar subida en cada parada aprobada y, en SDD sin paradas con `docs_publication: auto`, «subir también tras spec y design».

## Decisiones

- La autorización sellada no cambia: solo publica documentos (REQ-27.4). Los kickoffs ya sellados con `auto` heredan los momentos nuevos sin volver a preguntar.
- `continuous` sigue sin paradas y una negativa de `publish` sigue siendo informativa (REQ-27.13).
- Fuera de alcance: `checkpointed` (ya publica en cada aprobación, REQ-27.12), ODD y los kickoffs `continuous` sin `docs_publication`.
- El texto del pre-vuelo vive hoy en `sdd-orchestrator-sections.md:68` (la opción 3 del estilo de ejecución); `session_preflight.go` y `sdd_preflight_hook.go` ya no lo contienen, así que no se tocan.

## Alcance autorizado

`internal/assets/skills/_shared/sdd-orchestrator-sections.md` (opción del estilo y «When to publish»), `sdd-status-contract.md`, el comentario de `internal/kickoff/types.go`, `internal/assets/assets_test.go`, goldens de `testdata/golden/`, `docs/manual-de-inicio.md`, una prueba en `internal/branch/engine/publication_test.go` y las specs `sdd-docs-publication` (REQ-27.13) y `sdd-lane-kickoff` (REQ-27.4).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones: `go test` de los paquetes tocados con `-run` (norma del usuario: nada de baterías completas en local; la suite entera, en la CI).

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:feature`. Previsión: unas 150 líneas.

## Tareas

- [x] T1 · Prueba: `PublicationTarget` resuelve la rama del cambio antes de `tasks` (cambio de un rol con solo `proposal.md` y spec). Evidencia: `go test ./internal/branch/engine -run TestPublicationTarget` ok.
- [x] T2 · Doctrina: la opción del pre-vuelo y «When to publish» publican tras `spec`, `design` y `tasks` y al archivar; contrato de status, comentario de `DocsPublicationAuto`, manual, prueba de assets y goldens. Evidencia: `go test ./internal/assets/... ./internal/kickoff/... ./internal/components/...` ok; goldens regenerados con `-update` (2 líneas por fichero).
- [ ] T3 · `## Spec viva` (REQ-27.13 y REQ-27.4) integrada con `axiom archive integrate --odd publicar-spec-design`.

Ruta de todas las tareas: inline (unidades de menos de ~100 líneas; sin unidades independientes que justifiquen escritores en paralelo).

## Criterio de terminado

- REQ-27.13 nombra cuatro momentos: spec, design, tasks y archivo.
- La sección del orquestador ordena `axiom sdd publish --cwd <root> --change <name> --json` al terminar `spec` y `design` con `docsPublication: auto`.
- La opción del pre-vuelo dice que sube los documentos a `origin/<rama>` al cerrar spec, design y tasks y al archivar.
- Goldens regenerados y `go test` de `./internal/assets/...`, `./internal/components/...`, `./internal/kickoff/...` y `./internal/branch/engine` (con `-run`) en verde.

## Progreso

- Worktree creado desde `origin/main` (`814b3ce5`).

## Siguiente paso

T1.
