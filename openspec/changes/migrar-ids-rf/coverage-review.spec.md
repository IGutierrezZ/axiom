# Revisión de cobertura de la spec — migrar-ids-rf

Revisión informativa (`skills/_shared/sdd-coverage-review.md`): no bloquea fases ni el archivado y no edita la spec. En `continuous` los huecos van al resumen final.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Ambos devolvieron un resultado utilizable. Entradas: `proposal.md` y las deltas `specs/requirement-trace/spec.md` y `specs/requirement-identifiers/spec.md`. Ninguno consultó las specs vivas.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | requisito no cubierto | `requirement-identifiers` (sin requisito) | La herramienta de reescritura no tiene requisito ni escenario: sustitución por token completo (`REQ-1.1` no toca `REQ-1.10`), idempotencia, resolución del ambiguo por capacidad de contexto y fallo sin escribir ante un ambiguo sin contexto. | A, B |
| 2 | requisito no cubierto | ninguna delta | La corrección de `openspecdoc.ParseDelta` (ignorar `### Requirement:` en vallas) no tiene requisito, escenario ni capacidad declarada; falta la no regresión de `livespec`. | A, B |
| 3 | requisito no cubierto | `requirement-identifiers` | No se especifica la asignación a los 48 requisitos sin ID ni la migración completa de `Cubre:`, prosa y rangos H1 («REQ-42.1 a REQ-42.9»); tampoco un `Cubre:` con ID antiguo ambiguo. | A, B |
| 4 | requisito no cubierto | `requirement-identifiers` | Sin escenario para las variantes RF de las pruebas de corpus, la regeneración de `INDEX.md` sin diferencias inesperadas, la estabilidad de los goldens, la unicidad de los IDs RF entre capacidades ni las referencias huérfanas. | A, B |
| 5 | caso límite | `requirement-trace#prov:TRC:1` | Entradas sin ID antiguo identificadas por título: títulos repetidos en una capacidad, comparación del título, uso en trace; el invariante de completitud (513) ante requisitos añadidos después y la inmutabilidad (invariante 5) no tienen escenario. | A, B |
| 6 | caso límite | `requirement-trace#RF-TRC-04` | Cita heredada ambigua sin contexto de capacidad (prosa ODD, `Previous` de `ids.yaml`, `Cubre:` sin capacidad): ¿cuenta para todos, ninguno o avisa? Consulta por un ID nuevo sin ID antiguo. | A, B |
| 7 | caso límite | `requirement-trace#RF-TRC-02` | Interacción del mapa con un proyecto de patrón por defecto (un `REQ-n.m` válido y a la vez en el mapa); formas distintas del ID antiguo (ceros a la izquierda, mayúsculas). | A, B |
| 8 | error | `requirement-trace#prov:TRC:2` | Casos de error del mapa sin escenario: `REQ-n.m` consultado sin mapa, fichero vacío o ilegible, invariante violado al ejecutar `axiom trace`, capacidad inexistente, ID nuevo no válido; salida (código o JSON) y aviso ante una consulta ambigua con varios IDs resueltos; si un mapa inválido falla también en consultas ajenas a él. | A, B |
| 9 | caso límite | `requirement-identifiers#prov:RID:1` | Lista de residuos: granularidad (fichero, línea o literal); la carpeta del propio cambio y su paso a `archive/`; `docs/archive/**` y `docs/audits/**`; categoría fuera del conjunto cerrado o sin motivo; ficheros mixtos (un ejemplo del defecto no puede ocultar una cita propia). | A, B |
| 10 | caso límite | `requirement-identifiers#prov:RID:1` | Sin escenario de que una cita en un mensaje de producción nunca es residuo, aunque se declare. | A, B |
| 11 | requisito no cubierto | `requirement-identifiers` | Ejecución en proceso (sin binarios, sin código muerto para el trinquete) y `axiom trace REQ-24.3` sobre las specs migradas reales, no solo con fixtures. | A |
| 12 | error | `requirement-trace` (delta compuesta al archivar) | Sin criterio que valide antes de archivar que `RF-TRC-02`/`RF-TRC-04` existen tras migrar y que los ADDED reciben `RF-TRC-16`/`RF-TRC-17`, que el texto ya cita. | A, B |

Evidencia: ambos leyeron la propuesta completa y las dos deltas.
