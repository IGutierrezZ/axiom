# Revisión de cobertura del design — migrar-ids-rf

Revisión informativa (`skills/_shared/sdd-coverage-review.md`): no bloquea fases ni el archivado y no edita el design. En `continuous` los huecos van al resumen final; los marcados «absorber en tasks» se pasan a `sdd-tasks` como restricciones de implementación compatibles con la spec.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo, ambos con resultado utilizable. Entradas: `proposal.md`, las deltas y `design.md`. El validador de contrato de fase (contexto limpio) dio PASA, sin citas inexistentes.

| # | Tipo | Ubicación | Hueco | Revisores | Tratamiento |
|---|---|---|---|---|---|
| 1 | error | `design.md#D11` | `TestIDMapMatchesLivingSpecs` exige 513 entradas y que todo ID de la spec viva esté en el mapa: al archivar este cambio (RF-TRC-16, RF-TRC-17, RF-RID-16) o con cualquier requisito posterior, la CI fallaría. La completitud es «en el momento de la migración». | A, B | absorber en tasks: la prueba comprueba mapa → spec viva (cada ID nuevo del mapa existe) y no spec → mapa |
| 2 | requisito no cubierto | `design.md#D2`, `#D11` | El invariante 5 (inmutable una vez publicado) no tiene mecanismo ni prueba; el contador 513 no detecta un ID nuevo cambiado ni un intercambio. | A, B | absorber en tasks si es barato (huella del mapa fijada en prueba); si no, pendiente |
| 3 | requisito no cubierto | `design.md#D7`, `#D13` | Las líneas `Cubre:` deben citar IDs válidos bajo el patrón RF; la prueba de residuos solo busca `REQ-n.m` y deja pasar un `Cubre:` con ID RF inexistente o con un token retirado declarado como residuo. | A, B | absorber en tasks: prueba de validez y existencia de los IDs de `Cubre:` en documentos vivos |
| 4 | caso límite / error | `design.md#D8` | Mapa presente con patrón por defecto: los IDs nuevos RF serían inválidos bajo `REQ-<n>.<m>` y `Load` fallaría (`id_map_invariant`) en cualquier proyecto o checkout así; tampoco se fija la precedencia entre error de mapa y error de ID. | A, B | absorber en tasks: validar los IDs nuevos con su propia forma o solo bajo el patrón declarado; fijar y probar la precedencia |
| 5 | caso límite | `design.md#D4` | Finales de línea (CRLF/LF), BOM y codificación al reescribir cientos de ficheros en Windows; ficheros sin sustituciones no se reescriben; ficheros listados por git que no existen o son binarios. | A, B | absorber en tasks: preservar bytes salvo los tokens sustituidos |
| 6 | caso límite | `design.md#D8`, `#D2` | Requisitos sin ID antiguo (entradas por título): `axiom trace` por su ID nuevo no encuentra los incrementos archivados que los tocaron por título. | A | pendiente (informativo) |
| 7 | caso límite | `design.md#D8` | `REQ-27.2b`: `Scan`/`isTokenRune` pueden leerlo como `REQ-27.2` seguido de `b`; la consulta se aceptaría sin encontrar incrementos, y la sustitución en prosa podría fallar. | B | absorber en tasks: prueba específica del token |
| 8 | requisito no cubierto | `design.md#D10`, `#D11` | Ninguna prueba permanente afirma que `axiom.yaml` declara `sdd.id_pattern: RF-<DOM>-<NN>` (RF-RID-01). | B | absorber en tasks |
| 9 | error | `design.md#D7` | `TestRepositoryLegacyIDResidues` depende de `git ls-files`: sin git o fuera de un repositorio no se define si falla o se omite. | A | absorber en tasks: fallar (no omitir en silencio) |

Observaciones del validador (no bloqueantes): los assets embebidos de producción (`internal/assets/skills/**`) deberían quedar vetados como residuo igual que los `.go` de producción; la prueba de residuos usa `git ls-files` como subproceso dentro de `go test`; `legacy_ambiguous` es un aviso nuevo sin escenario en la spec.

Evidencia: ambos revisores leyeron la propuesta, las dos deltas y el design completo (D1–D13); no ejecutaron código.
