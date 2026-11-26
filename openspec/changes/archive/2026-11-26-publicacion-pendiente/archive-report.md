# Reporte de Archivado Formal: publicacion-pendiente

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: Aprobado (PASS) en el informe de verify (`**Estado:** limpio`, 0 CRITICAL, 2 WARNING abiertos y sin corregir, registrados como seguimiento). Verificación leída como instantánea de su momento; ver «Hechos de estado final».
- Destino en Archivo: `openspec/changes/archive/2026-11-26-publicacion-pendiente/`
- Cambio: C-05-c, «Estado "aprobada sin publicar" en `axiom sdd status`» (carril SDD, monorepo-embedded, almacén hybrid, kickoff `continuous`, un solo rol `fullstack`).

## Hechos de estado final
Fuentes ordenadas por rango (la más autorizada primero):
1. Tabla de tareas `tasks.md` (archivada, sin modificar): 24 casillas de tarea, todas marcadas (`[x]`), 0 pendientes.
2. Hechos explícitos del lanzamiento de archivado: implementación fusionada en `main` mediante la PR #342 (merge commit fe46b903), CI en verde. Commits de la implementación: planificación e97b7b57, U1 a2ee55b0, U2 c0f1465f, U3 8afe9e82, U4 6dce8f07, U5 574ffb1b, fase 6 e6d6fe67, verify 79dcc56a. Durante el rebase sobre `main` se resolvió un conflicto en `internal/assets/assets_test.go` (main había retirado las constantes `jointVerify*`).
3. Instantáneas `verify-report.md` (de su momento) y `coverage-review.*.md`.

**Contradicción de recuento (sin resolver):** el lanzamiento indica 28/28 tareas completas; el `tasks.md` archivado contiene 24 casillas, todas completas. El artefacto de tareas tiene rango superior y se toma como recuento de cierre (24/24). Las 4 diferencias no se pueden corroborar desde el fichero; quedan registradas sin resolver en ambas direcciones.

**Seguimiento abierto (WARNING de `verify-report.md`, instantánea de su momento; no corregido):**
- W1: la condición (a) de `publicationDue` (`internal/sddstatus/publication.go`) se cumple con cualquier compuerta aprobada. Con `spec` aprobada y `design.md` en iteración, el status proyecta `pending` y la doctrina podría publicar un `design` no aprobado. Coincide con el hueco 1 de `coverage-review.design.md`.
- W2: límite aceptado: `CommitFiles` no lista las rutas de los commits de fusión (`publication_state.go`); una fusión con conflictos resueltos sobre código puede proyectar `pending` donde `publish` se negaría. Coincide con el hueco 6 de `design.md`.

**Decisiones del usuario registradas:**
- D1: estado aparte `code_unpublished` con `pushCommand`, sin `invocation`, para los commits de código sin subir.
- El kickoff se selló SIN `--docs-publication auto`; el usuario rechazó esa opción.

**Cobertura (informativa, no bloqueante):** `coverage-review.spec.md` (10 huecos) y `coverage-review.design.md` (8 huecos). Ningún hueco se corrigió durante el archivado.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento puramente aditivo: la delta `specs/sdd-docs-publication/spec.md` contiene solo `## ADDED Requirements` (REQ-27.18 y REQ-27.19). No hay REMOVED, MODIFIED ni RENAMED.

| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| — | — | — | — | — |

## Integración de especificaciones vivas
- Capacidad `sdd-docs-publication`: `openspec/specs/sdd-docs-publication/spec.md` compuesta con `axiom sdd archive-integrate --cwd <worktree> --change publicacion-pendiente` (exit 0; sin provisionales `prov:` en tasks ni en la delta; `archive-check` exit 0, cambio no ISO).
- Verificación de la composición: `git diff --stat` = 250 inserciones, 0 eliminaciones; requisitos añadidos: REQ-27.18 («El Status Nativo Proyecta la Publicación Pendiente») y REQ-27.19 («El Orquestador Resuelve la Publicación Pendiente Antes de la Fase Siguiente»).
- Catálogo maestro: `axiom archive sync --path <worktree>` (exit 0): 49 especificaciones consolidadas, 513 requisitos activos, 2293 escenarios; `openspec/INDEX.md` regenerado.
- Contexto técnico mantenido: no proyectado por el status; no aplica `axiom archive context`.

## Trazabilidad de observaciones Engram
- Observation IDs de `sdd/publicacion-pendiente/proposal`, `spec`, `design`, `tasks` y `verify-report`: no recuperados en esta fase. La herramienta `mem_current_project` no estaba disponible en el entorno de este agente, por lo que no se ejecutaron las lecturas de Engram (regla de proyecto actual). Las rutas de fichero sí se leyeron directamente en el worktree.

## Artefactos Consolidados
- `proposal.md`
- `specs/sdd-docs-publication/spec.md` (delta, integrado en la spec viva)
- `design.md`
- `tasks.md`
- `verify-report.md`
- `coverage-review.spec.md`
- `coverage-review.design.md`
- `kickoff.yaml`

## Verificación mecánica
- Movimiento: `git mv openspec/changes/publicacion-pendiente openspec/changes/archive/2026-11-26-publicacion-pendiente`. El origen queda ausente.
- Readback `diff -r` (instantánea previa al movimiento vs. destino): sin diferencias (salida vacía, código 0).
- Este reporte es aditivo y queda fuera de la comparación.
