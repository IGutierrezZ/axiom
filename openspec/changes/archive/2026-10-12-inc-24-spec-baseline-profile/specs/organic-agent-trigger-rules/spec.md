# Delta para Organic Agent Routing Projection (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Tipo:** delta sobre la capacidad viva `organic-agent-trigger-rules`

Cambios de este delta: se añade una regla mínima al paso de cierre del protocolo ODD (decisión Q15, aceptada por el usuario). Se ubica en esta capacidad porque es la especificación viva que ya gobierna la proyección del protocolo ODD (el requisito «Organic Driven Development is the orchestrator's mandatory default protocol»); la semántica del contexto técnico mantenido y de referencia se define en `project-technical-context`. El requisito es aditivo: no cambia ningún requisito existente de esta capacidad.

Nota de idioma: la especificación viva de destino está redactada en inglés; este delta sigue la regla de idioma del proyecto (castellano), de modo que tras el archivado esta capacidad contendrá un requisito en castellano.

## ADDED Requirements

### Requirement: El cierre de ODD mantiene al día el contexto técnico mantenido (REQ-24.58)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN incluir, en su paso de cierre, una regla mínima: si el proyecto declara un contexto técnico mantenido (`technical_context.maintained` en `axiom.yaml`) y una unidad de trabajo cambia algo que ese contexto documenta, la documentación afectada se actualiza en el mismo commit de esa unidad de trabajo. La regla NO DEBE exigir nada cuando el proyecto no declara contexto mantenido, NO DEBE autorizar la modificación de documentación declarada como de referencia (`technical_context.reference`) y NO DEBE alterar los demás pasos del protocolo ODD ni el carril SDD. La proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Esta regla es la extensión mínima de `project-technical-context` al carril ODD, que no tiene archive.

#### Scenario: La regla figura en el paso de cierre del protocolo ODD

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el paso de cierre del protocolo ODD incluye la regla de actualizar el contexto técnico mantenido en el mismo commit de la unidad de trabajo
- **Y** la regla indica que solo aplica cuando el proyecto declara `technical_context.maintained`

#### Scenario: La regla no exige nada sin contexto mantenido

- **DADO** un proyecto sin `technical_context.maintained`
- **CUANDO** el orquestador cierra una unidad de trabajo bajo ODD
- **ENTONCES** la regla no exige ninguna actualización de documentación de contexto
- **Y** el cierre procede como antes

#### Scenario: La documentación de referencia no se modifica

- **DADO** un proyecto con `technical_context.reference` declarado
- **CUANDO** una unidad de trabajo bajo ODD cambia algo que esa documentación describe
- **ENTONCES** la regla no autoriza a modificarla

#### Scenario: El renderizado sigue siendo determinista

- **DADO** una sincronización de adaptador
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado
- **Y** los demás pasos del protocolo ODD y el carril SDD no cambian
