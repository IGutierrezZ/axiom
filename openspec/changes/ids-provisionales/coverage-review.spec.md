# Revisión de cobertura · spec · ids-provisionales

Revisión informativa (`skills/_shared/sdd-coverage-review.md`): no bloquea ninguna fase, puerta ni archivado, y no modifica la spec. Estilo `continuous`: los huecos van al resumen final.

## Ronda 1 — 2026-11-25

- Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura, con resultado utilizable los dos.
- Entradas: `proposal.md` y las seis delta specs de `specs/**`.
- Unión deduplicada: 11 huecos.

| # | Tipo | Ubicación | Hueco | Base | Revisor |
|---|---|---|---|---|---|
| 1 | requisito sin cubrir | `specs/` (todas) | El alcance 11 incluye las guías `sdd-spec`, `sdd-archive`, `_shared/sdd-iso-convention.md` y la regeneración de goldens. Ninguna delta tiene requisito ni escenario para ellas, salvo el paso 7 de ODD (REQ-24.58) y `sdd-status-contract.md` (REQ-47.9). | Propuesta, alcance 11 | A, B |
| 2 | caso límite | `requirement-identifiers` REQ-47.2 | El siguiente libre se calcula solo sobre la spec viva. Puede asignar un número que ya ocupa un ADDED con definitivo del propio cambio. Ocurre en la misma serie o cuando una «serie nueva» coincide con una serie fijada en el cambio. Falta la regla de que los definitivos del propio cambio cuentan como ocupados, o el rechazo. | Propuesta, alcance 4 («Un ADDED con ID definitivo se sigue aceptando») y «Único dentro del cambio» | A, B |
| 3 | requisito sin cubrir | `sdd-iso-evidence` REQ-29.4 / REQ-29.6 | La matriz de verificación ISO de `verify-report.md` cita IDs antes de archivar. No se dice si admite provisionales ni si se reescribe al archivar. Una fila con un provisional podría bloquear el archivado ISO. | Spec REQ-29.3 («la matriz de verify (REQ-29.6)») | B |
| 4 | caso límite | `requirement-identifiers` REQ-47.3.4 / REQ-47.4 | La validación rechaza provisionales en «el texto del cambio», pero la reescritura solo cubre la spec viva, `Cubre:`, la valla y las `Cubre:` del ODD, y las deltas. No se define qué pasa con un provisional citado en prosa de `tasks.md`, fuera de la valla del ODD, o en `proposal.md`, `design.md` o `verify-report.md`: si se reescribe, se rechaza o se permite. | Propuesta, alcances 3 y 4 | A, B |
| 5 | caso límite | `axiom-sdd-cli-integration` REQ-47.9 | No se define el elemento de `requirements` cuando el requisito no tiene ID: MODIFIED o REMOVED por título en un cambio no ISO, RENAMED cuyo nombre antiguo no tiene ID, o ADDED sin ID. | Spec REQ-47.9 y REQ-29.3 («En un cambio no ISO el identificador no se exige») | B |
| 6 | caso límite | `sdd-archive-lifecycle` REQ-47.8 | La reintegración tras un rebase solo tiene escenarios en ODD (título cambiado, definitivo ocupado por otro cambio, mapa guardado). En SDD falta el caso de la segunda rama que ya asignó y se rebasa. | Propuesta, alcance 5 y criterio de éxito «(ODD y SDD)» | A, B |
| 7 | caso límite | `odd-living-spec-integration` REQ-47.7 | No se define `--check` sobre un documento ya integrado: si los requisitos salen vacíos o con el mapa guardado, y si devuelve código 0. Tampoco el caso de una reasignación ni si el mapa previsto distingue entradas nuevas de reasignadas. | Spec REQ-47.7 («exactamente como lo haría la integración real») | B |
| 8 | caso límite | `requirement-identifiers` REQ-47.2 / `odd-living-spec-integration` REQ-30.3 | Si al reintegrar choca una serie nueva, la spec solo reasigna «ese requisito». No se dice si se reasigna la serie entera, con números consecutivos y coherentes entre capacidades, ni cómo se actualiza el mapa. | Propuesta, alcances 2 y 5 | A |
| 9 | error | `requirement-identifiers` REQ-47.3 | No hay escenario para un bloque `sdd:` inválido o un patrón ilegible (REQ-44.9) en el plan, en `--odd`, en `--check` ni en el paso de archivado. Tampoco para un provisional cuya serie fijada no es válida en el patrón. | Spec REQ-47.3, punto 6 | A |
| 10 | requisito sin cubrir | `requirement-identifiers` REQ-47.1 | La propuesta promete pruebas de no regresión para los lectores de deltas activas (indexador `livingdoc` y dashboard). No hay requisito ni escenario de que ignoren el provisional como ID. | Propuesta, Riesgos («Se añaden pruebas de no regresión») | A |
| 11 | caso límite | `sdd-archive-lifecycle` REQ-47.8 / REQ-47.6 | El informe del paso de archivado SDD no tiene escenario de contenido ni de formato. La interrupción solo se cubre «después de asignar y componer». Falta el caso con una capacidad nueva ya escrita y el resto sin componer. | Propuesta, alcances 6 y 7 | A |
