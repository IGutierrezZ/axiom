# Revisión de cobertura · design · adopcion-ids

Revisión informativa tras la fase `design` (`skills/_shared/sdd-coverage-review.md`). No bloquea ninguna fase y no cambia ningún artefacto revisado.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Entradas: `proposal.md`, las tres deltas de `specs/` y `design.md`. Los dos dieron un resultado utilizable. B contrastó además `internal/adoption/scan.go:34` y `internal/adoption/discards.go:26,150`. A no consultó código.

Huecos (unión deduplicada):

| # | Tipo | Artefacto | Ubicación | Hueco | Revisores |
|---|---|---|---|---|---|
| 1 | uncovered_requirement | design | Estrategia de pruebas (U5) | U5 no prueba el escenario de REQ-30.13 en el que un mismo cambio de adopción mezcla definitivos conservados y provisionales. Los provisionales deben recibir números posteriores al máximo conservado aunque el definitivo más alto aparezca después en la delta. | A |
| 2 | edge_case | design | D6 (`origin_id`) | `Origin.OriginalID` es el texto crudo entre paréntesis (`discards.go:150`). No se define qué pasa con `(RF-001, RF-002)`, `(RF-001 y RF-002)` ni con valores que no son ID (`(n/d)`, `(sin ID)`), y no hay pruebas para ninguno. | B |
| 3 | edge_case | design | D6 pasos 4-5 | Fusión: un requisito que cita varias unidades con IDs distintos acepta el título si coincide con cualquiera, y el resto de IDs del cliente se pierde sin aviso. División: dos requisitos que citan la misma unidad chocan en unicidad sin guía. El mensaje nombra E pero no sugiere qué unidad corresponde. | A, B |
| 4 | edge_case | design | D1 / D7 (unidad ambigua) | Una unidad con varios IDs propios (una tabla de requisitos bajo un único encabezado) bloquea `check`. Solo puede salir normalizando o descartando la unidad, así que no hay camino que conserve los IDs. Tampoco se dice si los IDs de una unidad descartada cuentan como ocupados. | A |
| 5 | use_case | design | D4 / Migración | Convivencia con una spec viva previa: un proyecto con `REQ-n.m` vivos que adopta un origen `RF-nnn` y declara `RF-<NNN>` deja de indexar sus `REQ-n.m`. `LookupIdentity` no los ve como ocupados y los demás verbos (`archive-check`, `trace`, `livespec`) los tratan distinto. El design no dice cómo se avisa ni cómo se protege. | A, B |
| 6 | edge_case | design | D2 (inferencia) | No se trata el número 0 (`RF-000`, `RF-0`), ni el relleno de ancho 5-6, ni un `REQ-12` frente a un `REQ-<n>.<m>` ya declarado (proponer `REQ-<m>` desplazaría los existentes). Solo lo cubriría la defensa `template_invalid`. | A, B |
| 7 | edge_case | design | D7 / D8 / D10 (multirepo) | En multirepo no hay rama de cambio en el repositorio de specs, porque trabaja en la rama de integración. El motivo de `pattern_undeclared` y la guía dicen «en la rama del cambio» sin matizarlo, y no hay ninguna prueba de `scan`/`check` multirepo con patrón declarado. | A, B |
| 8 | uncovered_requirement | spec | REQ-30.11 (g) / D7 | La exclusión de unidades descartadas, `foreign_id_conflict` y el paso 4 de `origin_id` no tienen escenario en la delta (lo reconoce el riesgo R4 del design). El verify contra la spec no los puede fijar. | B |
| 9 | error | spec | REQ-30.9, escenario «Esquemas incompatibles son un conflicto» | El escenario usa `RF-001` frente a `RF.001`, pero `foreignIDPattern` solo admite `-` y `_` como separador (`scan.go:34`). `RF.001` nunca es un ID ajeno, así que la prueba no se puede escribir tal cual: el conflicto real de separador es `-` frente a `_`. | B |
| 10 | error | design | D6 / D7 | Solo se describe el inventario ausente. Falta qué hacen `origin_id`, `pattern_undeclared` y `foreign_id_conflict` con un inventario ilegible, corrupto o de esquema desconocido, y cuando una unidad citada ya no existe en el inventario. | A |
| 11 | edge_case | design | D4 / D10 (provisionales) | Falta cómo elige la persona el `<DOM>` del provisional cuando el patrón declarado tiene dominio y el origen no lo trae. La guía solo cita `prov:<series>:<ordinal>` y no pone el ejemplo `prov::<ordinal>` para un patrón de serie fija como `RF-<NNN>`. | A |

Total: 11 huecos (6 comunes, 3 solo de A, 2 solo de B). En `continuous` no se relanza `sdd-design`. Se entregan a `sdd-tasks` y `sdd-apply` como casos de prueba cuando caben en las decisiones cerradas. Los que no caben van al resumen final:

- El hueco 9 se prueba con `-` frente a `_`. El texto del escenario se corrige al archivar.
- El hueco 5 queda como aviso en la guía; la protección completa sería un seguimiento.
- La colisión por cadena en `archive-integrate` (riesgo R3 del design) se propone como seguimiento.
