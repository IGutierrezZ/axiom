# Delta para organic-agent-trigger-rules

> Parte del estado vigente: el protocolo ODD no tiene ninguna parada (REQ-27.17) y el paso 0 lee `workflow` de `axiom sdd status` (REQ-27.16). Este delta no reintroduce ninguna parada.

## ADDED Requirements

### Requirement: La Doctrina ODD Expresa que ODD no Integra en la Spec Viva y Remite los Bugs de un Proyecto ISO al SDD (REQ-29.18)

La doctrina ODD generada (el paso 0, que lee `workflow` de `axiom sdd status`, o el cierre del protocolo) DEBE expresar que el carril ODD nunca integra en la spec viva: no compone deltas, no modifica la baseline declarada ni archiva cambios en ella. DEBE expresar también que, cuando `axiom sdd status` proyecta que el proyecto declara ISO (REQ-29.2 de `sdd-iso-evidence`; todavía no hay cambio sellado, así que se lee la declaración vigente del proyecto), un bug se tramita por el carril SDD: sin delta, con `Cubre:` a un requisito existente, cuando restaura un requisito que ya existe (REQ-29.15 de `axiom-sdd-cli-integration`), y con delta cuando el requisito estaba mal especificado o faltaba. El texto DEBE ser idéntico para todos los proyectos y adaptadores (REQ-27.16): no contiene el valor del carril ni la condición ISO de ningún proyecto, solo la instrucción condicionada a lo que proyecta el status. NO DEBE contener ninguna parada ni el texto de las tres salidas (REQ-27.17), y el resto del protocolo ODD (preparación de la rama con `axiom branch ensure`, commit por unidad de trabajo, reanudación desde `odd/tasks/<feature>.md`) NO cambia. Supuesto visible: en un proyecto ISO, el perfil de bug del SDD prevalece sobre un `workflow.lane: odd` declarado cuando el trabajo es la corrección de un bug.

#### Scenario: La doctrina ODD generada expresa que no integra en la spec viva

- **DADO** las instrucciones de enrutamiento gestionadas renderizadas para un runtime soportado
- **CUANDO** se inspecciona (pin de texto) el paso 0 o el cierre del protocolo ODD
- **ENTONCES** indica que ODD nunca integra en la spec viva

#### Scenario: La doctrina remite los bugs de un proyecto ISO al SDD según el status

- **DADO** las mismas instrucciones renderizadas
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican que, si el status proyecta que el proyecto declara ISO, un bug se tramita por el SDD: sin delta cuando restaura un requisito existente y con delta cuando estaba mal especificado o faltaba

#### Scenario: La doctrina es idéntica en proyectos ISO y no ISO

- **DADO** dos proyectos, uno ISO y otro no, con el mismo adaptador
- **CUANDO** se renderizan las instrucciones de enrutamiento
- **ENTONCES** el texto es idéntico byte a byte

#### Scenario: La adición no reintroduce ninguna parada

- **DADO** las instrucciones renderizadas con esta adición
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no contienen `Plan stop`, `Stops and exits` ni el texto de las tres salidas
- **Y** la preparación de la rama, el commit por unidad de trabajo y la reanudación sin espejo de Engram se expresan igual que antes
