# Revisión de cobertura — spec de `bateria-pruebas`

Revisión informativa: no bloquea fases, puertas ni archivado, y no entra en ningún digest. En `continuous`, los huecos pasan al resumen final y no relanzan `sdd-spec`.

## Ronda 1 — 2026-11-26

Revisores: A (`jd-judge-a`) y B (`jd-judge-b`), ciegos y en paralelo. Los dos devolvieron resultado utilizable. Entradas: `proposal.md` y las cuatro specs delta.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | `sdd-lane-kickoff` RF-KOF-21 | Con la función inactiva, sin bloque o con bloque inválido, y `pruebas.md` presente, no se fija contra qué dominio se validan los `Entornos`. Las opciones serían que cada fila dé `entorno_invalido`, omitir la comprobación o dar un error propio. Lo mismo vale para el status y el dashboard con el bloque inválido. | A, B |
| 2 | edge_case | `sdd-lane-kickoff` RF-KOF-18 | No hay escenario para un `pruebas.md` con dos o más tablas. No se dice si se toma la primera o si es `tabla_invalida`. Tampoco hay escenario para una tabla sin filas de datos. | A, B |
| 3 | edge_case | `sdd-lane-kickoff` RF-KOF-18 | No hay escenario para un `kickoff.yaml` ausente, sin sellar o ilegible al validar `Rol`. B pregunta además si un caso puede cubrir requisitos de otro rol en multirepo. | A, B |
| 4 | uncovered_requirement | `sdd-lane-kickoff` RF-KOF-20 | En multirepo no se dice cómo se resuelven las rutas de `pruebas.md` y de las specs delta (*planning home*) desde `--cwd`, ni en el status, el digest o el dashboard. | A |
| 5 | edge_case | `sdd-lane-kickoff` RF-KOF-22 | No se define cuándo está «terminado» `tasks` para `pruebas_ausente`. Queda abierto el caso multirrol con solo algunos `tasks.<rol>.md`. | A |
| 6 | error | `sdd-lane-kickoff` RF-KOF-22 | No se dice qué proyectan el status y el dashboard cuando `Check` no puede ejecutarse, por ejemplo sin specs delta o con una spec ilegible. Falta garantizar que el status no falla ni cambia `nextRecommended`. | B |
| 7 | edge_case | `sdd-lane-kickoff` RF-KOF-19 | No se cubren estos casos: la entrada `ID#` con escenario vacío y su código; las citas repetidas en un caso o en varios; la cita a un escenario de un requisito REMOVED o RENAMED. | B |
| 8 | uncovered_requirement | `local-web-dashboard` RF-WEB-17 | No se cubre la pestaña en estos casos: incremento ODD o sin carpeta de cambio; función activa con `tasks` sin terminar; `pruebas.md` ilegible, enorme o con tabla inválida; bloque inválido. | A, B |
| 9 | edge_case | `sdd-block-review-gates` RF-GAT-31 | No hay escenario para el digest al crear o borrar `pruebas.md` con la puerta `tasks` en `rejected` o `pending`. Afecta al plan de reversión de la propuesta. | B |
| 10 | edge_case | `sdd-lane-kickoff` RF-KOF-23 | El archivado no dice qué hace con una `pruebas.md` de tabla inválida o sin columna `Cubre`: omitir la reescritura, fallar o bloquear. Tampoco cubre provisionales sin correspondencia ni la función desactivada tras generar el fichero. | B |

Total: 10 huecos únicos (A: 6, B: 8; 4 compartidos).
