# Delta para organic-agent-trigger-rules

La regla 3 («Spec viva») del cierre de ODD pasa a indicar que el ADDED lleva un identificador provisional y que `axiom archive integrate --odd` asigna el número y reescribe el documento (C-03-b).

Notación de los escenarios: el formato literal del identificador provisional lo fija el diseño y el texto de la regla lo nombra de forma genérica («identificador provisional»); NO se fija aquí.

## MODIFIED Requirements

### Requirement: El cierre de ODD mantiene al día el contexto técnico mantenido (REQ-24.58)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN incluir, en sus pasos de exploración y de cierre, estas reglas mínimas, en un texto idéntico para todos los proyectos y adaptadores:

1. Lectura del contexto técnico: el paso de exploración DEBE indicar que, si el proyecto declara un contexto técnico mantenido, se obtienen los localizadores con `axiom workspace context --cwd <repo> --json`, se abre primero `technicalContext.maintained.entry` y se consultan los demás `technicalContext.maintained.paths` solo bajo demanda. Sin contexto técnico mantenido declarado, el paso NO DEBE exigir ninguna lectura adicional. La regla NO DEBE depender de `axiom sdd status --json` ni de un cambio SDD.
2. Contexto técnico: el cierre DEBE indicar que, si el proyecto declara un contexto técnico mantenido y una unidad de trabajo cambia algo que ese contexto documenta, la documentación afectada se actualiza en el mismo commit de esa unidad de trabajo, con los localizadores leídos en la exploración (la salida de `axiom workspace context --cwd <repo> --json`). La regla NO DEBE exigir nada cuando el proyecto no declara contexto mantenido y NO DEBE autorizar la modificación de documentación declarada como de referencia (`technical_context.reference`).
3. Spec viva: el cierre DEBE indicar que, cuando el trabajo cambia o debe revisar la spec viva, la delta OpenSpec se escribe en la sección `## Spec viva` del documento `odd/tasks/<feature>.md`; que todo requisito ADDED lleva un identificador provisional (o, si el autor lo prefiere, un identificador definitivo, que se sigue admitiendo con la comprobación de colisión) y que `axiom archive integrate --odd <feature>` asigna el número definitivo y reescribe el documento; que todo bug (señalado con `Tipo: bug`) aporta una `## Spec viva` con contenido o el marcador de la sección `## Impacto en la spec` (la línea `Sin impacto en la spec` más una línea `Motivo:` no vacía, el mismo contrato que el carril SDD; el bug ODD no exige `## Informe de bug`); y que se ejecuta `axiom archive integrate --odd <feature>` en el commit de la unidad de trabajo que cierra la feature, sin paradas (REQ-27.17). La integración en `openspec/specs/` y la reescritura de los identificadores del propio documento ODD son las únicas escrituras permitidas fuera del código.
4. Remisión: el cierre NO DEBE contener el valor del carril ni la condición ISO de ningún proyecto; la negativa de la integración en proyectos ISO la emite el verbo (REQ-30.7 de `odd-living-spec-integration`).

Estas reglas NO DEBEN alterar los demás pasos del protocolo ODD ni el carril SDD. La proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Estas reglas son la extensión de `project-technical-context` y de `odd-living-spec-integration` al carril ODD, que no tiene archive.
(Previously: la regla 3 decía que la delta se escribe en `## Spec viva` y que el ADDED lleva el identificador elegido por el autor, y que la integración en `openspec/specs/` era la única escritura permitida fuera del documento ODD y del código; ahora el ADDED lleva un provisional que la integración asigna y reescribe en el documento, y esa reescritura del documento también está permitida.)

#### Scenario: La exploración lee la entrada del contexto mantenido

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el paso de exploración del protocolo ODD indica obtener los localizadores con `axiom workspace context --cwd <repo> --json`, abrir primero `technicalContext.maintained.entry` y consultar `technicalContext.maintained.paths` solo bajo demanda
- **Y** la regla indica que sin contexto mantenido declarado no se lee nada más

#### Scenario: La regla figura en el paso de cierre del protocolo ODD

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el paso de cierre del protocolo ODD incluye la regla de actualizar el contexto técnico mantenido en el mismo commit de la unidad de trabajo
- **Y** la regla indica que solo aplica cuando el proyecto declara un contexto técnico mantenido

#### Scenario: El cierre remite a los localizadores de la exploración

- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de cierre
- **ENTONCES** remite a los localizadores leídos en la exploración con `axiom workspace context --cwd <repo> --json`
- **Y** no cita `axiom sdd status --json` como localizador del contexto técnico

#### Scenario: El cierre manda integrar la spec viva y declarar el impacto de los bugs

- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de cierre
- **ENTONCES** indica la sección `## Spec viva`, el marcador de `## Impacto en la spec` para los bugs sin delta (la línea literal `Sin impacto en la spec` y una línea `Motivo:`) y la ejecución de `axiom archive integrate --odd <feature>`
- **Y** no contiene ninguna parada ni el texto de las tres salidas

#### Scenario: El cierre indica el identificador provisional del ADDED y la asignación

- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de cierre
- **ENTONCES** indica que el requisito ADDED lleva un identificador provisional y que `axiom archive integrate --odd <feature>` asigna el número definitivo y reescribe el documento
- **Y** indica que un identificador definitivo en el ADDED sigue admitiéndose
- **Y** no fija el formato literal del identificador provisional

#### Scenario: La regla no exige nada sin contexto mantenido

- **DADO** un proyecto sin `technical_context.maintained`
- **CUANDO** el orquestador explora y cierra una unidad de trabajo bajo ODD
- **ENTONCES** la regla no exige ninguna lectura ni actualización de documentación de contexto
- **Y** la exploración y el cierre proceden como antes

#### Scenario: La documentación de referencia no se modifica

- **DADO** un proyecto con `technical_context.reference` declarado
- **CUANDO** una unidad de trabajo bajo ODD cambia algo que esa documentación describe
- **ENTONCES** la regla no autoriza a modificarla

#### Scenario: El renderizado sigue siendo determinista e idéntico entre proyectos

- **DADO** una sincronización de adaptador y dos proyectos, uno ISO y otro no
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado y entre proyectos
- **Y** los pasos del protocolo ODD distintos de la exploración y el cierre, y el carril SDD, no mencionan el contexto técnico
