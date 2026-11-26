# Revisión de cobertura: spec y design de `axiom-trace`

Informe informativo (`sdd-coverage-review.md`): no bloquea ninguna fase ni el archivado, y no corrige los artefactos. En estilo `continuous` los huecos pasan al resumen final.

## Ronda 1 — 2026-11-26

- Entradas: `proposal.md`, `specs/requirement-trace/spec.md` y `design.md`.
- Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Los dos devolvieron un resultado utilizable.
- Unión deduplicada: 11 huecos.

| # | Tipo | Artefacto | Ubicación | Hueco | Revisor |
|---|---|---|---|---|---|
| 1 | uncovered_requirement | design | Decisiones 8 y 10, preguntas abiertas | prov:trace:14 pide la rama de integración de cada repositorio, pero el design usa una sola rama para todas las fuentes y deja la cuestión abierta. Si las ramas difieren, los commits son erróneos o la consulta falla. Falta decidir entre una rama por fuente o una limitación documentada con advertencia. | A, B |
| 2 | error | design | Decisión 5 | Una delta archivada que existe pero no se puede interpretar no tiene código de advertencia (el análogo SDD de `odd_unparseable`). `unreadable` solo cubre errores de lectura. | A |
| 3 | error | design | Decisiones 6 y 9 | No se fija la frontera entre `found:false`, advertencia y error cuando `History` falla por un error real de git, como un objeto ausente o un proceso fallido. La ordenación «propaga los errores de git» y aborta toda la consulta por un solo incremento. | A |
| 4 | error | design | Decisión 8 | La comprobación previa con `TreeHas` convierte cualquier fallo de `git ls-tree` en `RefError`, con el remedio `--integration-branch`, aunque la causa sea otra: repositorio corrupto, git ausente o clon sin commits. | B |
| 5 | edge_case | design | Decisiones 4, 6 y 7 | El escaneo lee el árbol de trabajo y el anclaje usa la rama de integración. No se declaran los archivados o documentos ODD integrados pero ausentes del árbol de trabajo, el contenido editado en una rama de feature ni la asimetría al ejecutar desde un worktree de feature: el SDD sale con `found:false` y el ODD se omite. | A, B |
| 6 | edge_case | design | Decisión 5 | No se define el caso de `openspec/changes/archive/` inexistente ni el de `odd/tasks/` vacío. Deben dar un resultado vacío y código 0 (prov:trace:11), no un error de `ReadDir`. | B |
| 7 | edge_case | design | Decisión 6 | El commit de código solo se busca por `verify-report.md`. Con un cambio multirrol que solo tiene `verify-report.<rol>.md`, o un archivado sin informe de verify, el incremento cae al anclaje `archive` sin advertencia. | B |
| 8 | error | spec, design | prov:trace:1, Decisión 12 | Los errores de entorno no tienen requisito ni escenario: `--cwd` inexistente o que no es un directorio, falta de repositorio git para la fuente SDD sin `axiom.yaml`, y `--integration-branch` sin valor o inválida. La tabla de la Decisión 12 no los lista todos. | A, B |
| 9 | edge_case | spec | prov:trace:5 | La spec no fija el rol de un incremento cuya única aparición es una entrada de `ids.yaml` o de `## Identificadores asignados`. Solo el design decide `created`. | A |
| 10 | edge_case | spec | prov:trace:13, prov:trace:14 | En multirepo el design ordena los incrementos de repositorios distintos solo por la clave estable, en contra de la lectura literal de prov:trace:13 (posición en `--first-parent`). La spec no declara esa excepción. | A |
| 11 | edge_case | spec | prov:trace:3 | Un mismo provisional repetido en cambios distintos con definitivos diferentes no tiene escenario. El design lo trata con `provisional_ambiguous` y con coincidencia por fuente, pero el invariante «mismo resultado» de la spec puede contradecirse ahí. | B |
