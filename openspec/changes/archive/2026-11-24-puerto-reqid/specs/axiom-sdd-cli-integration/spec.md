# Delta para axiom-sdd-cli-integration

> Parte del estado vigente de `openspec/specs/axiom-sdd-cli-integration/spec.md`. REQ-29.16 formula el motivo de identificador ausente, mal formado, ambiguo o repetido según el patrón declarado del proyecto (REQ-44.1 y REQ-44.5 de `requirement-identifiers`); con el patrón por defecto el texto de los motivos y continuaciones no cambia. La instantánea de `kickoff.yaml` sigue gobernando la regla ISO (`sdd.iso`), pero el patrón se lee en vivo de `axiom.yaml` (D-04). El título del requisito no cambia.

## MODIFIED Requirements

### Requirement: Motivos de Bloqueo Tipados de ISO, Cobertura, Escenarios y Matriz en `status` (REQ-29.16)

`axiom sdd status` DEBE proyectar, con un motivo tipado que nombre el elemento afectado y la continuación, los bloqueos de un cambio ISO (sellado con `iso`, REQ-29.13 de `sdd-lane-kickoff`; la condición ISO y las reglas de proceso se leen de su instantánea y no de `axiom.yaml` en vivo, mientras que el patrón de identificador se lee siempre en vivo de `axiom.yaml`, nunca de la instantánea: REQ-44.1 de `requirement-identifiers`) definidos en `sdd-iso-evidence` y `sdd-archive-lifecycle`: identificador válido según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`; con el patrón por defecto, `REQ-<n>.<m>`) ausente, mal formado, ambiguo o repetido en el nombre de los requisitos de la delta (REQ-29.3), formulado con el patrón efectivo en su texto legible, cobertura `Cubre:` y `Cubre:` a un requisito inexistente (REQ-29.4), requisito de un bug con requisito existente inexistente, sin citar o sin escenarios (REQ-29.5), `Cubre:` en un bug sin impacto en la spec (REQ-29.19), requisito ADDED o MODIFIED sin escenarios (REQ-29.10), y matriz de verify ausente, inválida o con filas fallidas en la fase `archive` (REQ-29.6 a REQ-29.8 y REQ-29.11). Fuera de un cambio ISO, ninguna de esas reglas DEBE proyectar ni bloqueo ni aviso. `axiom sdd continue` DEBE reflejar cualquier bloqueo sin autorizar la fase siguiente. Los nuevos motivos son solo aditivos: ninguna clave existente del status v2 cambia, y el orden de precedencia de los motivos existentes (por ejemplo, `wrong_branch` antes de `edit_authority_missing`) se conserva. Cuando concurren varios bloqueos, el status DEBE proyectar todos con su continuación, no solo el primero. Si el patrón de identificador es inaccesible (bloque `sdd:` inválido o configuración ilegible), un cambio ISO NO DEBE evaluarse con un patrón no declarado: la etapa `specs` proyecta el mismo hallazgo que ya lleva el archivado, con el código `iso_declaration_invalid`, en lugar de las reglas de identificador; la etapa `tasks` no se evalúa; y la proyección `iso.matrix` se omite (REQ-44.9 de `requirement-identifiers`). No se crea ningún código de motivo nuevo.
(Previously: la instantánea de `kickoff.yaml` gobernaba todas las reglas; ahora gobierna la condición ISO y el patrón se lee en vivo. El motivo de identificador se formulaba siempre con la forma literal `REQ-<n>.<m>`; ahora se formula con el patrón declarado del proyecto.)

#### Scenario: Un bloqueo ISO nombra el elemento y la continuación

- **DADO** un cambio ISO y un requisito de la delta sin cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de cobertura nombrando el requisito
- **Y** incluye la continuación (añadir un `Cubre:` en alguna tarea)

#### Scenario: Un requisito sin escenarios se proyecta antes del archivado

- **DADO** un cambio ISO y una delta con un ADDED sin escenarios
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de escenarios nombrando el dominio, la sección y el requisito

#### Scenario: Una matriz inválida bloquea la dependencia `archive`

- **DADO** un cambio ISO en la fase `archive` con una matriz con un escenario sin fila
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la dependencia `archive` aparece bloqueada con el motivo tipado de matriz que nombra el escenario
- **Y** incluye la continuación (completar la matriz del `verify-report.md`)

#### Scenario: Fuera de ISO la cobertura y la matriz no proyectan nada

- **DADO** un cambio no ISO, requisitos sin `Cubre:` y un `verify-report.md` libre
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ni bloqueo ni aviso de cobertura o de matriz

#### Scenario: `axiom sdd continue` refleja el bloqueo ISO

- **DADO** un cambio ISO bloqueado por cobertura
- **CUANDO** se ejecuta `axiom sdd continue <cambio>`
- **ENTONCES** no emite ninguna fase siguiente autorizada
- **Y** reporta el mismo motivo tipado

#### Scenario: Varios bloqueos concurrentes se proyectan todos

- **DADO** un cambio ISO con un requisito ADDED sin identificador válido en su nombre y otro requisito sin cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyectan los motivos de identificador y de cobertura, cada uno con su continuación

#### Scenario: Un `Cubre:` junto a la marca de bug sin impacto se proyecta como motivo tipado

- **DADO** un cambio ISO declarado como bug sin impacto en la spec y una tarea con `Cubre: REQ-13.1`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de declaración contradictoria, con el fichero de tareas, el requisito y la continuación

#### Scenario: La precedencia de los motivos existentes se conserva

- **DADO** un cambio ISO bloqueado por `wrong_branch` y por cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `wrong_branch` se proyecta con la precedencia que ya tenía
- **Y** el motivo de cobertura se proyecta además

#### Scenario: El contrato del status solo crece

- **DADO** el status de un cambio no ISO
- **CUANDO** se compara con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor


#### Scenario: Con un patrón declarado, el motivo de identificador nombra ese patrón

- **DADO** un cambio ISO de un proyecto con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}` y un requisito ADDED llamado `Alta de usuario` (sin identificador)
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de identificador, con la continuación formulada con el patrón `RF-<DOM>-<NN>`
- **Y** no menciona `REQ-<n>.<m>`

#### Scenario: Con el patrón por defecto el motivo de identificador no cambia

- **DADO** un cambio ISO de un proyecto sin declaración de patrón y un requisito ADDED llamado `Cobro con IVA`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el motivo tipado de identificador y su continuación son idénticos byte a byte a los de antes de este cambio

#### Scenario: El patrón se lee en vivo y la regla ISO de la instantánea

- **DADO** un cambio ISO sellado cuya delta nombra `Alta de usuario (RF-USU-01)` con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}`
- **CUANDO** el proyecto cambia `axiom.yaml` a `sdd: {iso: true, id_pattern: "HU-<NNN>"}` y se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio sigue siendo ISO por su instantánea
- **Y** el nombre se evalúa con `HU-<NNN>`: se proyecta el motivo tipado de identificador ausente nombrando el patrón `HU-<NNN>`

#### Scenario: Con el patrón inaccesible la etapa `specs` lleva `iso_declaration_invalid`

- **DADO** un cambio ISO sellado y un `axiom.yaml` cuyo `sdd.id_pattern` es inválido (`RF-`)
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la etapa `specs` proyecta el motivo `iso_declaration_invalid`, no el de identificador
- **Y** la etapa `tasks` no se evalúa
- **Y** en la fase `archive` no se proyecta `iso.matrix`

#### Scenario: Un patrón inválido se rechaza en `status` de un cambio no ISO

- **DADO** un cambio no ISO de un proyecto cuyo `sdd.id_pattern` es inválido
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se reporta el error de carga del patrón de REQ-44.1, sin asumir el patrón por defecto
