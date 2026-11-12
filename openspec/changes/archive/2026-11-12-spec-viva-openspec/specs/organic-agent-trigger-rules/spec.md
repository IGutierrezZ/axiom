# Delta para organic-agent-trigger-rules

> Parte del estado vigente: el protocolo ODD no tiene ninguna parada (REQ-27.17), el paso 0 lee `workflow` de `axiom sdd status` (REQ-27.16) y la rama se prepara con `axiom branch ensure` (REQ-25.40). Esta delta no reintroduce ninguna parada y no toca el carril SDD.
> Coordinación con `carril-sdd-versatil`: su REQ-29.18 (remisión de los bugs de un proyecto ISO al SDD) no afirma ni niega la integración de ODD en la spec viva; esta delta la fija, y REQ-30.7 de `odd-living-spec-integration` la niega en proyectos ISO.
> Nota: los escenarios de `sdd-lane-kickoff` que dicen «registra el trabajo exclusivamente en `odd/tasks/<feature>.md`» se refieren al registro de seguimiento del trabajo y siguen siendo ciertos; la integración en `openspec/specs/` de esta delta es una escritura de producto, no de seguimiento.

## MODIFIED Requirements

### Requirement: El cierre de ODD mantiene al día el contexto técnico mantenido (REQ-24.58)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN incluir, en su paso de cierre, estas reglas mínimas, en un texto idéntico para todos los proyectos y adaptadores:

1. Contexto técnico: el cierre DEBE indicar que se lea `technicalContext.maintained.entry` y `technicalContext.maintained.paths` de la salida de `axiom sdd status --json` (el localizador del contexto mantenido) y que, si el proyecto declara un contexto técnico mantenido y una unidad de trabajo cambia algo que ese contexto documenta, la documentación afectada se actualiza en el mismo commit de esa unidad de trabajo, abriendo primero `entry`. La regla NO DEBE exigir nada cuando el proyecto no declara contexto mantenido y NO DEBE autorizar la modificación de documentación declarada como de referencia (`technical_context.reference`).
2. Spec viva: el cierre DEBE indicar que, cuando el trabajo cambia o debe revisar la spec viva, la delta OpenSpec se escribe en la sección `## Spec viva` del documento `odd/tasks/<feature>.md`, que todo bug (señalado con `Tipo: bug`) aporta una `## Spec viva` con contenido o el marcador de la sección `## Impacto en la spec` (la línea `Sin impacto en la spec` más una línea `Motivo:` no vacía, el mismo contrato que el carril SDD; el bug ODD no exige `## Informe de bug`) y que se ejecuta `axiom archive integrate --odd <feature>` en el commit de la unidad de trabajo que cierra la feature, sin paradas (REQ-27.17). La integración en `openspec/specs/` es la única escritura permitida fuera del documento ODD y del código.
3. Remisión: el cierre NO DEBE contener el valor del carril ni la condición ISO de ningún proyecto; la negativa de la integración en proyectos ISO la emite el verbo (REQ-30.7 de `odd-living-spec-integration`).

El cierre NO DEBE alterar los demás pasos del protocolo ODD ni el carril SDD. La proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Estas reglas son la extensión de `project-technical-context` y de `odd-living-spec-integration` al carril ODD, que no tiene archive.
(Previously: el cierre solo exigía actualizar el contexto mantenido en el mismo commit, sin localizador de `axiom sdd status --json` y sin la integración de la spec viva)

#### Scenario: La regla figura en el paso de cierre del protocolo ODD

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el paso de cierre del protocolo ODD incluye la regla de actualizar el contexto técnico mantenido en el mismo commit de la unidad de trabajo
- **Y** la regla indica que solo aplica cuando el proyecto declara `technical_context.maintained`

#### Scenario: El cierre cita el localizador del contexto

- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de cierre
- **ENTONCES** indica leer `technicalContext.maintained.entry` y `technicalContext.maintained.paths` de `axiom sdd status --json`

#### Scenario: El cierre manda integrar la spec viva y declarar el impacto de los bugs

- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de cierre
- **ENTONCES** indica la sección `## Spec viva`, el marcador de `## Impacto en la spec` para los bugs sin delta (la línea literal `Sin impacto en la spec` y una línea `Motivo:`) y la ejecución de `axiom archive integrate --odd <feature>`
- **Y** no contiene ninguna parada ni el texto de las tres salidas

#### Scenario: La regla no exige nada sin contexto mantenido

- **DADO** un proyecto sin `technical_context.maintained`
- **CUANDO** el orquestador cierra una unidad de trabajo bajo ODD
- **ENTONCES** la regla no exige ninguna actualización de documentación de contexto
- **Y** el cierre procede como antes

#### Scenario: La documentación de referencia no se modifica

- **DADO** un proyecto con `technical_context.reference` declarado
- **CUANDO** una unidad de trabajo bajo ODD cambia algo que esa documentación describe
- **ENTONCES** la regla no autoriza a modificarla

#### Scenario: El renderizado sigue siendo determinista e idéntico entre proyectos

- **DADO** una sincronización de adaptador y dos proyectos, uno ISO y otro no
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado y entre proyectos
- **Y** los demás pasos del protocolo ODD y el carril SDD no cambian
