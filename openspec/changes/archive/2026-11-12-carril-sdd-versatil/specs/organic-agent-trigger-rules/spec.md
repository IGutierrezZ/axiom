# Delta para organic-agent-trigger-rules

> Parte del estado vigente: el protocolo ODD no tiene ninguna parada (REQ-27.17) y el paso 0 lee `workflow` de `axiom sdd status` (REQ-27.16). Este delta no reintroduce ninguna parada.
> Decisión D4 (tercera enmienda, 2026-11-12): **ODD integra en la spec viva; los proyectos ISO solo usan SDD.** Este requisito cubre solo la mitad de la doctrina; la integración de ODD en la spec viva la diseña P6 y queda fuera.

## ADDED Requirements

### Requirement: La Doctrina ODD Remite Todo el Trabajo de un Proyecto ISO al SDD (REQ-29.18)

La doctrina ODD generada (el paso 0, que lee `workflow` de `axiom sdd status`, o el cierre del protocolo) DEBE expresar que, cuando `axiom sdd status` proyecta que el proyecto declara ISO (REQ-29.2 de `sdd-iso-evidence`; todavía no hay cambio sellado, así que se lee la declaración vigente del proyecto), todo el trabajo se tramita por el carril SDD, sin preguntar el carril, también cuando el carril efectivo sea `ask`, y sin crear el documento `odd/tasks/<feature>.md`; y que un bug se tramita siempre por el SDD con el perfil de bug (REQ-29.15 de `axiom-sdd-cli-integration`), que tiene tres rutas: sin delta y con `Cubre:` a un requisito existente, cuando restaura un requisito que ya existe; sin delta y con la sección `## Impacto en la spec` (línea `Sin impacto en la spec` y línea `Motivo:`), cuando no toca ningún requisito; y con delta, como cambio normal, cuando el requisito estaba mal especificado o faltaba. En las dos primeras el bug lleva además la sección `## Informe de bug` en `proposal.md`. La integración de ODD en la spec viva (el verbo de integración de ODD y la revisión de impacto en ODD) la diseña P6 y queda fuera de este requisito: la doctrina no afirma ni niega ninguna regla sobre el mecanismo con el que ODD la integra. El texto DEBE ser idéntico para todos los proyectos y adaptadores (REQ-27.16): no contiene el valor del carril ni la condición ISO de ningún proyecto, solo la instrucción condicionada a lo que proyecta el status. NO DEBE contener ninguna parada ni el texto de las tres salidas (REQ-27.17), y el resto del protocolo ODD (preparación de la rama con `axiom branch ensure`, commit por unidad de trabajo, reanudación desde `odd/tasks/<feature>.md`) NO cambia para los proyectos que no declaran ISO. La combinación de `sdd.iso: true` con `workflow.lane: odd` no llega a la doctrina: `axiom.yaml` la rechaza como declaración contradictoria (REQ-29.1 de `sdd-iso-evidence`), así que la doctrina no necesita arbitrar entre ambos valores.

(Previously: este requisito solo remitía los bugs de un proyecto ISO al SDD con el perfil de bug, y dejaba el trabajo que no era un bug en el carril que eligiera `workflow`; se apoyaba en el supuesto visible de que el perfil de bug prevalecía sobre `workflow.lane: odd`. Ese supuesto desaparece porque la combinación se rechaza al validar.)

#### Scenario: La doctrina remite todo el trabajo de un proyecto ISO al SDD según el status

- **DADO** las instrucciones de enrutamiento gestionadas renderizadas para un runtime soportado
- **CUANDO** se inspeccionan (pin de texto) el paso 0 o el cierre del protocolo ODD
- **ENTONCES** indican que, si el status proyecta que el proyecto declara ISO, todo el trabajo se tramita por el carril SDD, sin preguntar el carril
- **Y** indican que en ese caso no se crea el documento `odd/tasks/<feature>.md`

#### Scenario: La doctrina remite los bugs de un proyecto ISO al SDD con el perfil de bug

- **DADO** las mismas instrucciones renderizadas
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican que, si el status proyecta que el proyecto declara ISO, un bug se tramita siempre por el SDD con el perfil de bug: sin delta cuando restaura un requisito existente, sin delta y con `## Impacto en la spec` (`Sin impacto en la spec` y `Motivo:`) cuando no toca ningún requisito, y con delta cuando estaba mal especificado o faltaba

#### Scenario: La doctrina no pregunta el carril en un proyecto ISO con carril `ask`

- **DADO** las mismas instrucciones renderizadas
- **CUANDO** se inspeccionan (pin de texto) la lectura de `workflow` del paso 0 y la remisión ISO
- **ENTONCES** la remisión ISO se expresa como condición previa a la pregunta de carril: con ISO proyectado no se formula la pregunta bloqueante de carril

#### Scenario: La doctrina nombra la sección y la línea literal de la ruta sin impacto

- **DADO** las mismas instrucciones renderizadas
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** contienen la sección `## Impacto en la spec` y la línea literal `Sin impacto en la spec` exactamente como las fija REQ-29.15, y la sección `## Informe de bug` con `Observado:` y `Esperado:`

#### Scenario: La doctrina no fija regla sobre la integración de ODD en la spec viva

- **DADO** las mismas instrucciones renderizadas
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** la remisión ISO no afirma ni niega ninguna regla sobre el mecanismo con el que ODD integra en la spec viva
- **Y** no se introduce ningún verbo de integración de ODD ni revisión de impacto en ODD (los diseña P6)

#### Scenario: La doctrina es idéntica en proyectos ISO y no ISO

- **DADO** dos proyectos, uno ISO y otro no, con el mismo adaptador
- **CUANDO** se renderizan las instrucciones de enrutamiento
- **ENTONCES** el texto es idéntico byte a byte

#### Scenario: La doctrina no menciona la combinación contradictoria

- **DADO** las mismas instrucciones renderizadas
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no contienen ninguna regla de precedencia entre `sdd.iso` y `workflow.lane: odd`, porque esa combinación la rechaza la validación (REQ-29.1 de `sdd-iso-evidence`)

#### Scenario: La adición no reintroduce ninguna parada

- **DADO** las instrucciones renderizadas con esta adición
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no contienen `Plan stop`, `Stops and exits` ni el texto de las tres salidas
- **Y** la preparación de la rama, el commit por unidad de trabajo y la reanudación sin espejo de Engram se expresan igual que antes
