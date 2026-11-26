# Revisión de cobertura: spec de `axiom-trace`

Informe informativo (`sdd-coverage-review.md`): no bloquea ninguna fase ni el archivado, y no corrige la spec. En estilo `continuous` los huecos pasan al resumen final.

## Ronda 1 — 2026-11-26

- Entradas: `proposal.md` y `specs/requirement-trace/spec.md` (prov:trace:1 a prov:trace:15).
- Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Los dos devolvieron un resultado utilizable.
- Unión deduplicada: 13 huecos.

| # | Tipo | Ubicación | Hueco | Revisor |
|---|---|---|---|---|
| 1 | edge_case | prov:trace:5, prov:trace:7 | La coincidencia del ID no se exige por token exacto: `REQ-1.1` no debe casar con `REQ-1.10` ni con `REQ-11.1` en títulos, `Cubre:`, `ids.yaml` o la búsqueda textual. Tampoco se define una línea `Cubre:` con varios IDs ni una cita dentro de un bloque de código. | A, B |
| 2 | error | prov:trace:5, prov:trace:6 | La tolerancia a datos corruptos solo se define para ODD. En una carpeta archivada falta qué pasa con un `ids.yaml`, una delta, un `tasks*.md` o un `proposal.md` ilegible o ausente, y con un `kickoff.yaml` ausente en cambios antiguos: advertencia o aborto. La clasificación bug/incremento queda sin definir sin `proposal.md`. | A, B |
| 3 | edge_case | prov:trace:8 | Hay dos casos sin escenario: un cambio SDD archivado que no está en la línea `--first-parent` (omitirlo, como el ODD de prov:trace:7, o informarlo con `found:false`) y un archivado con solo uno de sus dos commits encontrado. Tampoco se cubre un clon superficial ni un directorio que no es repositorio git. | A, B |
| 4 | edge_case | prov:trace:3 | Trazas contradictorias o transitivas: cadenas de `Previous` (A→B→C), un mismo provisional asignado a definitivos distintos, ciclos, o un `Previous` compartido que fusionaría IDs no equivalentes. No se dice si el conflicto llega a `warnings`. | A, B |
| 5 | edge_case | prov:trace:2 | Proyecto sin `axiom.yaml` o con un patrón inválido (¿patrón por defecto o error?). Tampoco se define la normalización del ID de entrada: mayúsculas, espacios o ceros a la izquierda. | A, B |
| 6 | error | prov:trace:1 | Entradas inválidas sin escenario: varios posicionales, `--integration-branch` sin valor, `--cwd` inexistente o fuera de un proyecto Axiom, y HEAD desacoplado. Solo se cubre la falta de ID. | A, B |
| 7 | edge_case | prov:trace:1, prov:trace:10 | Con `--json` no se define la forma de los errores: JSON de error o texto por stderr. | A, B |
| 8 | edge_case | prov:trace:7 | La búsqueda textual de respaldo no fija qué rol y qué tipo recibe un documento hallado solo por texto, aunque el rol es obligatorio en el JSON. | B |
| 9 | edge_case | prov:trace:7 | En ODD no se fija si hay un incremento por rol, como en SDD, ni que las tareas `Cubre:` se listen con fichero y línea. Faltan también el documento borrado o renombrado después de integrarse y la deduplicación de un ID presente como provisional y como definitivo en el mismo incremento. | A |
| 10 | edge_case | prov:trace:13, prov:trace:5 | No se define el orden entre incrementos que comparten commit de integración (squash conjunto) ni el rol de una aparición solo en `ids.yaml`, que no está en la lista created, modified, removed, renamed y covered. | B |
| 11 | edge_case | prov:trace:14 | En multirepo falta el escenario de repositorio inaccesible o sin rama de integración que el requisito exige tratar. Tampoco hay regla para el mismo cambio en dos repositorios ni para un workspace sin raíz git en `odd/tasks`. | A, B |
| 12 | edge_case | prov:trace:5 | Sin escenario para los archivados bajo el prefijo `planningHome` ni para `tasks.<rol>.md`. Tampoco lo hay para RENAMED con origen y destino ni para REMOVED por ID. | B |
| 13 | uncovered_requirement | prov:trace:1 | Dos criterios de éxito de la propuesta no tienen requisito ni escenario: que `axiom sdd trace` y `axiom.sdd-trace/v1` no cambian, y la documentación en el manual (§8). | A |
