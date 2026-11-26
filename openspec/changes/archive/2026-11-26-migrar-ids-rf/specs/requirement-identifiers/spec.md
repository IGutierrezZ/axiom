# Delta para Identificadores de Requisito (requirement-identifiers)

> **Cambio:** `migrar-ids-rf` (C-03-e). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> Solo añade un requisito; los existentes (`RF-RID-01` a `RF-RID-15`, antes REQ-44.1 a REQ-47.10) no cambian de texto. El patrón por defecto `REQ-<n>.<m>` de `RF-RID-02` tampoco cambia: Axiom declara su propio patrón.

## ADDED Requirements

### Requirement: Las Specs Propias de Axiom Usan el Patrón Declarado y no Citan Identificadores Heredados Fuera de los Residuos Declarados (RF-RID-16)

`axiom.yaml` del repositorio de Axiom DEBE declarar `sdd.id_pattern: RF-<DOM>-<NN>` (RF-RID-01). Bajo ese patrón, todo título `### Requirement:` de toda capacidad de `openspec/specs/` DEBE llevar un identificador válido; ninguno DEBE conservar un `REQ-n.m`, porque bajo un patrón distinto del por defecto un título con `REQ-n.m` queda sin identificador sin error alguno (sale del índice, del cálculo del siguiente libre y de la evidencia ISO). Las líneas `Cubre:` de los documentos vivos DEBEN citar identificadores válidos bajo ese patrón.

Ninguna cita a un requisito propio de Axiom en la forma `REQ-n.m` DEBE aparecer fuera de: `openspec/changes/archive/` (historia, no se reescribe), el mapa versionado de identificadores heredados (RF-TRC-16) y una lista de residuos legítimos versionada en el repositorio. Cada entrada de esa lista DEBE llevar una categoría de un conjunto cerrado y un motivo; las categorías son: fixture de pruebas del patrón por defecto, definición o ejemplo del patrón por defecto, ejemplo del patrón por defecto dirigido a otros proyectos, cita deliberada de un ID antiguo como tal, e historia fechada fuera de OpenSpec. Una cita en un mensaje de producción NUNCA es residuo admisible.

Una prueba automatizada DEBE recorrer el repositorio y fallar ante cualquier `REQ-n.m` que no esté en una de las tres ubicaciones anteriores, y ante cualquier entrada de la lista que ya no case con ninguna aparición. La prueba DEBE ejecutarse en proceso.

#### Scenario: Axiom declara su patrón

- **DADO** el `axiom.yaml` del repositorio de Axiom
- **CUANDO** se lee el bloque `sdd:`
- **ENTONCES** `id_pattern` es `RF-<DOM>-<NN>`

#### Scenario: Ningún título de la spec viva queda sin identificador

- **DADO** las capacidades de `openspec/specs/` y el patrón `RF-<DOM>-<NN>`
- **CUANDO** se recorren todos los títulos `### Requirement:`
- **ENTONCES** cada título lleva un identificador válido bajo el patrón
- **Y** ninguno contiene un `REQ-n.m`

#### Scenario: Una cita heredada no declarada rompe la prueba

- **DADO** un fichero versionado fuera de `openspec/changes/archive/`, que no es el mapa ni figura en la lista de residuos, y que cita un `REQ-n.m`
- **CUANDO** se ejecuta la prueba de residuos
- **ENTONCES** la prueba falla y nombra el fichero y la línea

#### Scenario: Una entrada de la lista sin aparición rompe la prueba

- **DADO** una entrada de la lista de residuos que ya no casa con ninguna aparición en el repositorio
- **CUANDO** se ejecuta la prueba de residuos
- **ENTONCES** la prueba falla y nombra la entrada obsoleta

#### Scenario: Los residuos declarados se aceptan

- **DADO** una aparición de `REQ-n.m` en un fixture de pruebas del patrón por defecto, declarada en la lista con su categoría y su motivo
- **CUANDO** se ejecuta la prueba de residuos
- **ENTONCES** la aparición no produce fallo
