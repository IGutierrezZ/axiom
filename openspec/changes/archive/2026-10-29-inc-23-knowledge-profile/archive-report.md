# Reporte de Archivado Formal: inc-23-knowledge-profile

## Resumen de Cierre
- Fecha de Archivado: 2026-10-29
- Estado de Verificación: Aprobado (PASS) según `verify-report.md`, fechado el 2026-09-29 (16/16 requisitos satisfechos). Es una instantánea de ese momento; no se ha reejecutado la verificación en este archivado.
- Estado de Implementación: 11/11 tareas marcadas en `tasks.md` (archivado); 0 pendientes. La implementación está en `main` desde `bd7cf24c` (`feat(knowledge): implementar perfil de inicialización y capacidades de descubrimiento y consulta (#56)`), comprobado como ascendiente de `origin/main`.
- Rama de archivado: `inc/inc-23-knowledge-profile-archive`, sobre `main` 54573ff4. Sin push ni PR: la entrega documental en PR aparte es decisión del usuario bajo la política ordinaria del repositorio.
- Baseline del proyecto: ninguna proyectada por el status. `technicalContext.maintained`: no proyectado. Kickoff `continuous`, sin compuertas de bloque.
- Destino en Archivo: `openspec/changes/archive/2026-10-29-inc-23-knowledge-profile/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento puramente aditivo.

## Especificaciones Vivas Promovidas
| Dominio | Acción | Requisitos | Escenarios | Origen en el `spec.md` archivado |
|---------|--------|------------|------------|----------------------------------|
| `knowledge-profile-init` | Creado | 6 | 8 | REQ-23.1 a REQ-23.6 |
| `knowledge-sweep-command` | Creado | 5 | 5 | REQ-23.7 a REQ-23.11 |
| `knowledge-query-engine` | Creado | 5 | 5 | REQ-23.12 a REQ-23.16 |

Método: la composición nativa no aplica. `axiom sdd archive-compose` exige una spec canónica existente y un delta con secciones ADDED, MODIFIED, REMOVED o RENAMED; ante una spec canónica inexistente falló con salida 1 y no escribió nada. El `spec.md` del cambio es un documento completo con tres capacidades, así que se partió por capacidad con copia mecánica (`head` y `tail` sobre el fichero fuente). Cada spec viva tiene una cabecera nueva (título, metadatos y «Propósito» tomado de la frase introductoria de su capacidad) y el cuerpo de requisitos y escenarios sin cambios. Se omitieron el encabezado de sección «## N. Capacidad» y la línea «Estado: En desarrollo (fase de especificación)». El `spec.md` archivado conserva sus bytes originales.

## Regeneración del Catálogo
`axiom archive sync --path <workspaceRoot>`: salida 0. `openspec/INDEX.md` pasa de 64 a 67 dominios, de 478 a 494 requisitos y de 1176 a 1194 escenarios BDD.

## Artefactos Consolidados
- Archivados y presentes en la carpeta: `proposal.md`, `spec.md` (monolítico, íntegro), `design.md`, `tasks.md`, `verify-report.md`, `kickoff.yaml`.
- Faltante: `apply-progress.md` (el status lo reporta como `missing`).

## Hechos del Estado Final
Fuentes por orden de prelación (Final-State Authority):
1. `tasks.md` archivado: 11 casillas `[x]` y 0 `[ ]`.
2. Hechos del encargo de archivado y estado nativo antes del archivado: `archive: ready`, `blockedReasons: []`, sin notas; implementación en `main` desde `bd7cf24c`, comprobado con git.
3. `verify-report.md` (2026-09-29): PASS con 16/16 requisitos. Se cita como historial; no se ha reejecutado la batería en este archivado. El informe no registra hallazgos abiertos.

## Contradicciones Registradas
1. Almacén de artefactos: el status reporta `artifactStore: openspec`; la preflight del orquestador indica `hybrid`. Prevalece el status, que es la fuente declarada del workspace. El reporte se persiste también en Engram por petición explícita.
2. Cabecera del `spec.md` archivado: «Estado: En desarrollo (fase de especificación)», frente a la implementación en `main` desde `bd7cf24c`. Es texto de un documento histórico y no se reescribe. Las specs vivas no heredan esa línea.

## Trazabilidad
- Observaciones de Engram leídas en esta fase: ninguna. Los artefactos se leyeron desde los ficheros que indicó el status (store `openspec`).
- Observaciones citadas por los artefactos del cambio: ninguna identificada.
- Persistencia de este reporte: fichero en esta carpeta y topic key `sdd/inc-23-knowledge-profile/archive-report`, proyecto `axiom`, tipo `architecture`.

## Verificación del Archivado
- `axiom workspace validate --path <workspaceRoot>`: `COMPLIANT`, salida 0.
- `go test -count=1 ./internal/workspace ./internal/assets`: `ok` en ambos paquetes.
- `axiom sdd status inc-23-knowledge-profile --json` tras el archivado: sin `changeRoot`, `archive: all_done`, `nextRecommended: archived`, `blockedReasons: []`.
- `axiom branch status --json`: salida 0. El cambio sigue listado en `pendingArchive` mientras la carpeta activa permanezca en el árbol de la rama de integración; la prueba `TestDocsDogfoodingListsPreexistingChangesAsPendingArchive` deriva su expectativa de las referencias git (inc-23 debe figurar si y solo si su carpeta está en el árbol de la integración), por lo que no depende de que el cambio siga activo y no se ha modificado.
- `diff -r` entre la instantánea previa al movimiento y la carpeta archivada: sin diferencias (salida 0). Las tres verificaciones de cuerpo de las specs vivas (una por capacidad): sin diferencias.
- `axiom sdd integrate verify`: no aplica, porque el cambio no tiene baseline del proyecto ni contexto técnico mantenido.

## Seguimientos Abiertos
- Entrega: PR documental aparte pendiente. Push, PR y fusión son decisiones del usuario bajo la política ordinaria del repositorio.
- Hasta fusionar el PR documental, `axiom branch status` seguirá mostrando inc-23 como pendiente de archivar, porque la rama de integración aún contiene la carpeta activa.
