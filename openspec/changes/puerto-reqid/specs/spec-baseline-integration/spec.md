# Delta para spec-baseline-integration

> Parte del estado vigente de `openspec/specs/spec-baseline-integration/spec.md`. REQ-29.14 resuelve el requisito existente por el identificador válido según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`) en lugar de la forma literal `REQ-<n>.<m>`; con el patrón por defecto el comportamiento no cambia. El título del requisito no cambia.

## MODIFIED Requirements

### Requirement: Consulta de un Requisito Existente en la Baseline para el Bug sin Delta (REQ-29.14)

El motor DEBE poder resolver si un identificador de requisito existe en la baseline (`openspec/specs/`, la spec viva del proyecto) y, si existe, devolver sus escenarios. La resolución se hace por el identificador válido, según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`), que contiene el nombre del requisito (REQ-29.3 de `sdd-iso-evidence`), y un requisito cuyo nombre no contiene ese identificador no puede citarse (supuesto visible). La consulta es de solo lectura y NO DEBE modificar ningún documento. La usa el bug con requisito existente (REQ-29.5 de `sdd-iso-evidence`) para comprobar, en un cambio ISO, que el `Cubre:` apunta a un requisito existente y para conocer los escenarios que la matriz de verify debe cubrir (REQ-29.6). Un identificador que no existe se informa como tal, sin error de ejecución, y un requisito existente sin escenarios se informa como existente con una lista de escenarios vacía.
(Previously: la resolución se hacía por la forma literal `REQ-<n>.<m>`; ahora es por el identificador válido según el patrón declarado del proyecto, que con el patrón por defecto coincide con `REQ-<n>.<m>`.)

#### Scenario: Un identificador existente se resuelve con sus escenarios

- **DADO** la baseline con el requisito `Subcomando axiom sdd status (REQ-13.1)` con dos escenarios
- **CUANDO** se consulta `REQ-13.1`
- **ENTONCES** se informa de que existe y se devuelven sus dos escenarios

#### Scenario: Un identificador inexistente se informa sin error

- **DADO** la misma baseline y el identificador `REQ-13.99`
- **CUANDO** se consulta `REQ-13.99`
- **ENTONCES** se informa de que no existe
- **Y** la consulta no falla ni modifica ningún documento

#### Scenario: Un requisito sin identificador en su nombre no puede citarse

- **DADO** la baseline con un requisito cuyo nombre no contiene un identificador válido según el patrón
- **CUANDO** se consulta cualquier identificador
- **ENTONCES** ese requisito no se devuelve para ningún identificador

#### Scenario: Un requisito existente sin escenarios se informa como tal

- **DADO** un requisito existente sin ningún `#### Scenario:`
- **CUANDO** se consulta su identificador
- **ENTONCES** se informa de que existe con una lista de escenarios vacía

#### Scenario: La consulta no escribe

- **DADO** cualquier consulta
- **CUANDO** termina
- **ENTONCES** el árbol de trabajo no presenta ningún cambio

#### Scenario: Con un patrón declarado, un identificador existente se resuelve con sus escenarios

- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}` y la baseline con el requisito `Alta de usuario (RF-USU-01)` con dos escenarios
- **CUANDO** se consulta `RF-USU-01`
- **ENTONCES** se informa de que existe y se devuelven sus dos escenarios

#### Scenario: Con un patrón declarado, un identificador con el formato por defecto no es válido

- **DADO** el mismo proyecto y una baseline cuyo requisito se llama `Alta de usuario (REQ-30.1)`
- **CUANDO** se consulta `REQ-30.1`
- **ENTONCES** ese requisito no se devuelve, porque su nombre no contiene un identificador válido según el patrón del proyecto
- **Y** la consulta no falla ni modifica ningún documento
