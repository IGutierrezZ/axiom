# ODD: el detalle del cambio muestra `specs/<dominio>/spec.md` y `tasks.<rol>.md` (C-06-a)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/fix-detalle-specs.md`.
> Espejo de recuperación en Engram: topic `odd/fix-detalle-specs/tasks`, proyecto `axiom`.
> Origen: plan maestro `docs/plan/backlog-2026-11/README.md` (ola 1) y su sección `06-ui-human-loop.md`, cambio C-06-a.

## Objetivo

Que el detalle de un cambio en el dashboard muestre todas las specs de dominio (`specs/<dominio>/spec.md`, una por dominio) y todas las tareas por rol (`tasks.<rol>.md`), sin dejar de mostrar los cambios heredados con `spec.md` y `tasks.md` planos.

## Problema

`GetIncrementDetail` (`internal/dashboard/service.go:544` y `:555`) solo mira `spec.md` y `tasks.md` planos en la raíz del cambio. sdd-spec escribe en `specs/<capacidad>/spec.md` y, con varios roles sellados, sdd-tasks escribe `tasks.<rol>.md`. La UI no enseña la spec real de ningún cambio SDD actual ni las tareas por rol.

## Por qué

Es un bug que bloquea a C-06-b y C-06-c, que añaden pestañas al mismo detalle.

## Alcance

- Regla de descubrimiento única: el dashboard usa la misma que `axiom sdd status` (`sddstatus.resolveArtifactPaths`), de la que el digest de las puertas (`internal/cli/sdd_gate_digest.go`) es espejo exacto. Se exporta desde `internal/sddstatus` en un fichero nuevo; no se duplica la regla ni se toca el digest (el mapa de solapes dice que C-06-a solo lo lee).
  - Specs: todos los `spec.md` bajo `specs/**`, ordenados; si no hay ninguno, el `spec.md` plano con contenido.
  - Tareas: con kickoff sellado de varios roles (o un rol con fichero distinto de `tasks.md`), los `tasks.<rol>.md` existentes en el orden del kickoff (`kickoff.RoleTaskFiles`); si no, `tasks.md`.
- `internal/dashboard/types.go`: el DTO de detalle pasa a listas `specs` y `tasks` de `{path, content}` (ruta relativa al cambio, con `/`). Se conservan `has_spec` y `has_tasks` (lista no vacía) para el paso a paso; desaparecen `spec` y `tasks_content`.
- `internal/dashboard/assets/js/screens/detalle.js`: la lista fija `ARTIFACTS` pasa a construirse desde el detalle, un documento por spec y por fichero de tareas.

## Fuera de alcance

Renderizar HTML, cambiar el renderizador de markdown, pestañas nuevas, el listado (`inspectIncrement`) y el digest de las puertas.

## Restricciones

- Worktree `C:\repos\axiom-wt\fix-detalle-specs\axiom`, rama `inc/fix-detalle-specs`. Nada en `C:\repos\axiom`.
- Sin `--fetch` ni `git push` sin autorización. Sin binarios sueltos de `axiom`: solo `go test` por paquete.
- Commits convencionales sin atribución de IA.
- TDD: no configurado en `axiom.yaml` (modo off, fuente: configuración del proyecto); comprobaciones funcionales ordinarias con `go test`.
- RDD: off (`axiom review mode status`, decidido por defecto).
- Entrega: una sola PR (`single-pr`); previsión unas 250 líneas.

## Tareas

- [x] **T1 · Backend: regla única y DTO** (`internal/sddstatus`, `internal/dashboard/service.go`, `types.go` y tests). Ruta: escritor delegado (2+ ficheros no triviales).
  - Exportar la resolución de ficheros de spec y tareas desde `internal/sddstatus`.
  - `GetIncrementDetail` rellena `specs` y `tasks` con esa regla.
  - Tests en `internal/dashboard` de los tres casos del criterio de terminado; ajustar `listing_test.go`.
- [x] **T2 · Frontend: documentos dinámicos en el detalle** (`detalle.js`, `workflow_assets_test.go`). Ruta: mismo escritor delegado.
  - Un documento por spec y por fichero de tareas; marcador «no existe todavía» con `specs/<dominio>/spec.md` y `tasks.md` cuando faltan.

## Criterio de terminado

- Un cambio con `specs/a/spec.md` y `specs/b/spec.md` muestra ambas en el detalle.
- Un cambio multirrol con `tasks.core.md` y `tasks.qa.md` muestra las dos.
- Un cambio heredado con `spec.md` y `tasks.md` planos se sigue viendo.
- Hay tests en `internal/dashboard` que cubren los tres casos.

## Comprobaciones

- `go test ./internal/dashboard/`
- `go test ./internal/sddstatus/`
- `go vet ./internal/dashboard/ ./internal/sddstatus/`

## Progreso

- Plan stop: approved 2026-11-17; implementation continues
- Espejo Engram: pendiente (el guardado no se pudo ejecutar en modo automático; se resincroniza cuando esté disponible).
- T1 hecha: commit `4ba64c32`. Riesgo `axiom review assess`: high (RDD off: autoverificación del escritor más verificador independiente).
- T2 hecha: commit `47a0dc49`.

## Evidencia de verificación

- `go test ./internal/dashboard/`: ok (escritor). Repetido por el padre con `-run 'TestIncrementDetail|TestServiceIncrementDetail|Listing|Worktree'`: ok.
- `go test ./internal/sddstatus/`: ok (escritor). Repetido por el padre con `-run TestChangeArtifactPaths`: ok.
- `go vet ./internal/dashboard/ ./internal/sddstatus/`: limpio. `gofmt -l`: vacío.
- Tests de los tres casos del criterio: `TestIncrementDetailListsDomainSpecs`, `TestIncrementDetailListsPerRoleTasksInKickoffOrder`, `TestIncrementDetailKeepsLegacyFlatFiles` (`internal/dashboard/detail_artifacts_test.go`).
- Verificador independiente: sin hallazgos bloqueantes.

## Seguimientos (fuera de alcance)

- Un `spec.md` plano vacío ya no aparece en el detalle: la regla compartida exige contenido. Es coherente con `axiom sdd status`.
- Faltan tests del fallback con `kickoff.yaml` corrupto y del detalle de un cambio archivado.
- `assets/js/core/increment.js:191,193` sigue rotulando el paso a paso con `spec.md` y `tasks.md`.
- El digest de las puertas (`internal/cli/sdd_gate_digest.go`) sigue siendo un espejo de `sddstatus.resolveArtifactPaths` en vez de llamar a `ChangeArtifactPaths`.

## Siguiente paso

PR única lista para abrirse cuando el usuario lo autorice (`scripts/axiom-worktree.ps1 pr fix-detalle-specs -Lane odd`), con etiqueta `type:*`.
