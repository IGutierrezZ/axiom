# ODD: escenarios de la advertencia `legacy_ambiguous` en la spec de la traza (trace-legacy-ambiguous)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/trace-legacy-ambiguous.md`. Espejo Engram: `odd/trace-legacy-ambiguous/tasks`.
> Rama: `inc/trace-legacy-ambiguous` · worktree `C:\repos\axiom-wt\trace-legacy-ambiguous\axiom`.

## Objetivo

Que la spec viva de `requirement-trace` describa la advertencia `legacy_ambiguous` que `axiom trace` ya emite.

## Problema y porqué

La migración C-03-e (`migrar-ids-rf`) añadió a `axiom trace` la advertencia `legacy_ambiguous` (`internal/reqtrace/alias.go`, `internal/reqtrace/classify.go`), pero ninguna spec la menciona. Tampoco dice qué pasa con una cita de un ID antiguo repetido en varias capacidades cuando su contexto no nombra ninguna. Era un pendiente informativo del archive de `migrar-ids-rf` (hueco 6 de `coverage-review.spec.md`). El comportamiento ya está implementado y probado; este cambio solo lo deja por escrito.

## Decisiones

- **Solo spec.** No cambia código: las pruebas existentes ya fijan el comportamiento (`TestRequirementTraceLegacyCubreWithAndWithoutContext` y `TestRequirementTraceLegacyClosure` en `internal/reqtrace/legacy_test.go`; `TestAxiomTraceLegacyAmbiguousIDWarnsAndResolvesAll` en `internal/cli/trace_legacy_test.go`).
- **Dónde:** regla 6 de RF-TRC-04 (punto de extensión de alias), junto a la regla 3 que acota los IDs antiguos por capacidad, más un escenario nuevo y una línea en el escenario del ID antiguo ambiguo.
- **El código del aviso se nombra en la spec** (`legacy_ambiguous`), porque es un código estable de `axiom.trace/v1` y quien consume el JSON decide por él.

## Alcance autorizado

La spec viva `requirement-trace` (RF-TRC-04), este documento y, si la prueba de residuos lo exige, su entrada en `internal/idmigration/residues.yaml`.

## Modo TDD

No aplica: no hay cambio de comportamiento. Comprobaciones: `go test ./internal/reqtrace/ -run Legacy`, `go test ./internal/cli/ -run TraceLegacy` y `go test ./internal/idmigration/ -run TestRepository` después del commit. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit. Etiqueta `type:docs`.

## Tareas

- [x] T1 · Regla 6 y escenarios de `legacy_ambiguous` en RF-TRC-04, integrados con `axiom archive integrate --odd trace-legacy-ambiguous`. Ruta: inline (unidad pequeña). Evidencia: `go test ./internal/reqtrace/ -run Legacy` ok; `go test ./internal/cli/ -run TraceLegacy` ok; `go test ./internal/idmigration/` ok tras el commit, con el residuo `md-code` de este documento en `residues.yaml`. Integración `composed` de `requirement-trace` (más `INDEX.md`).
  Cubre: RF-TRC-04

## Criterio de terminado

- RF-TRC-04 describe cuándo una cita heredada sin contexto cuenta, cuándo no, y las dos situaciones en que se emite `legacy_ambiguous`.
- Las pruebas citadas pasan sin cambios y la CI está en verde.

## Spec viva

### Capacidad: requirement-trace

```openspec
## MODIFIED Requirements

### Requirement: Punto de Extensión de Alias (RF-TRC-04)

La resolución de equivalentes MUST pasar por un punto de extensión de alias sustituible, de modo que el mapa de identificadores heredados (RF-TRC-16) añada equivalentes sin modificar la consulta. La implementación por defecto MUST devolver únicamente el propio ID, su pareja provisional/definitivo y los `Previous` de las trazas. El mapa de identificadores heredados MUST estar cableado como resolutor de alias de `axiom trace` cuando existe, con estas reglas:

1. Un ID antiguo MUST resolverse a todos sus IDs nuevos: un ID antiguo repetido en varias capacidades (`REQ-1.1`) devuelve el ID nuevo de cada una de ellas.
2. Un ID nuevo MUST resolverse a su ID antiguo (cuando lo tiene), para encontrar los incrementos archivados que lo citan.
3. La coincidencia de un ID antiguo en los incrementos archivados MUST acotarse por la capacidad: un ID antiguo que aparece en el título de una delta archivada de la capacidad `C` (carpeta `specs/C/`), o en una línea `Cubre:` o una entrada de `ids.yaml` cuyo contexto nombra `C`, equivale solo al ID nuevo de (`C`, ID antiguo). MUST NOT contar como incremento del ID nuevo de otra capacidad que usaba el mismo ID antiguo.
4. Consultar por un ID nuevo MUST traer los incrementos archivados que tocaron su ID antiguo en su capacidad, con los roles de RF-TRC-05, además de los que citan el ID nuevo.
5. Consultar por el ID nuevo de una entrada del mapa sin ID antiguo (la que lleva el título) MUST traer también los incrementos, SDD archivados u ODD, cuya delta de esa misma capacidad nombra un requisito con ese título exacto (sin espacios al principio ni al final), con los roles de RF-TRC-05; en RENAMED cuentan el nombre origen y el destino. Una delta de otra capacidad con el mismo título MUST NOT contar, y las líneas `Cubre:` no casan por título.
6. Una cita de un ID antiguo cuyo contexto (la capacidad de la delta, las capacidades del cambio archivado o las `### Capacidad:` del documento ODD) no nombra ninguna capacidad del mapa para ese ID antiguo cuenta como el ID nuevo de su única capacidad cuando el ID antiguo figura solo en una, y cuenta siempre cuando se consulta ese mismo ID antiguo. Si el ID antiguo figura en varias capacidades y se consulta uno de sus IDs nuevos, la cita MUST NOT contar como incremento y `warnings` MUST incluir una advertencia `legacy_ambiguous` que nombre la fuente, el ID antiguo y las capacidades candidatas, y que proponga consultar por el ID antiguo. Consultar un ID antiguo que figura en varias capacidades MUST añadir a `warnings` una advertencia `legacy_ambiguous` que nombre sus IDs nuevos y sus capacidades.

Un alias que no sea válido bajo el patrón del proyecto MUST aceptarse cuando proviene del mapa de identificadores heredados.

#### Scenario: Un alias inyectado se incorpora a la consulta

- **DADO** un resolutor de alias que declara `«ID-b»` equivalente a `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** los IDs resueltos incluyen `«ID-b»`
- **Y** los incrementos que tocan `«ID-b»` forman parte del resultado

#### Scenario: Sin resolutor adicional solo cuentan las trazas

- **DADO** la implementación por defecto y un ID sin entrada de traza
- **CUANDO** se consulta
- **ENTONCES** los IDs resueltos son solo el ID consultado

#### Scenario: Un ID antiguo ambiguo resuelve a todos sus IDs nuevos

- **DADO** el mapa con `REQ-1.1` en siete capacidades
- **CUANDO** se consulta `axiom trace REQ-1.1`
- **ENTONCES** los IDs resueltos incluyen el ID nuevo de cada una de las siete capacidades y el propio `REQ-1.1`
- **Y** `warnings` contiene una advertencia `legacy_ambiguous` que nombra los siete IDs nuevos y sus capacidades

#### Scenario: Un ID antiguo no ambiguo resuelve a su ID nuevo

- **DADO** el mapa con `REQ-24.3` en una sola capacidad
- **CUANDO** se consulta `axiom trace REQ-24.3`
- **ENTONCES** los IDs resueltos incluyen el ID nuevo de esa capacidad y `REQ-24.3`

#### Scenario: La coincidencia en archivados se acota por capacidad

- **DADO** una delta archivada de la capacidad `autoskills` que modifica `REQ-1.1` y otra de `workspace-topology` que modifica el mismo `REQ-1.1`
- **CUANDO** se consulta el ID nuevo de `autoskills` para ese requisito
- **ENTONCES** el resultado incluye el incremento de `autoskills`
- **Y** no incluye el de `workspace-topology`

#### Scenario: Consultar por el ID nuevo trae los incrementos del ID antiguo

- **DADO** un cambio archivado cuya delta de la capacidad `C` modifica `«ID-antiguo»` y el mapa que lo asocia a `«ID-nuevo»` en `C`
- **CUANDO** se consulta `«ID-nuevo»`
- **ENTONCES** el resultado incluye ese cambio archivado con el rol `modified`

#### Scenario: Un requisito sin ID antiguo se encuentra por su título en su capacidad

- **DADO** el mapa con la entrada (`C`, título `«Título»`, `«ID-nuevo»`) sin ID antiguo, un cambio archivado cuya delta de `C` modifica `### Requirement: «Título»` y otro cuya delta de otra capacidad modifica un requisito con el mismo título
- **CUANDO** se consulta `«ID-nuevo»`
- **ENTONCES** el resultado incluye el primer cambio con el rol `modified`
- **Y** no incluye el segundo


#### Scenario: Una cita heredada sin contexto de capacidad no cuenta y avisa

- **DADO** el mapa con `REQ-1.1` en `autoskills` y en `workspace-topology`, y un cambio archivado sin deltas cuya línea `Cubre:` cita `REQ-1.1`
- **CUANDO** se consulta el ID nuevo de `workspace-topology` para ese requisito
- **ENTONCES** el resultado no incluye ese cambio
- **Y** `warnings` contiene una advertencia `legacy_ambiguous` que nombra el cambio y las capacidades `autoskills` y `workspace-topology`
- **Y** al consultar `REQ-1.1` el resultado sí incluye ese cambio
```
