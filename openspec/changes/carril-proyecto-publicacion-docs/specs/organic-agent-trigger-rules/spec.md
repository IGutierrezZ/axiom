# Delta para Organic Agent Routing Projection (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** delta sobre la capacidad viva `organic-agent-trigger-rules`.
> **Idioma:** los bloques modificados conservan el idioma del bloque vivo (inglés o castellano); los nuevos van en castellano de España.

Cambios de este delta:

- REQ-27.16 (nuevo): el paso 0 de la doctrina proyecta la lectura del carril del proyecto y la política de selección de SDD admite el carril declarado como selección explícita.
- REQ-27.17 (nuevo): ninguna parte de la doctrina ODD generada contiene una parada; el texto de las tres salidas es exclusivo de SDD.
- REQ-26.10 se retira (parada única de ODD).
- «Canonical provider-owned routing facts», «SDD remains optional» y «Organic Driven Development is the orchestrator's mandatory default protocol»: el carril declarado en el proyecto es selección explícita de SDD.
- REQ-25.40: se retira la referencia a la parada de REQ-26.10 en el escenario «El resto del protocolo no cambia» y en la nota.

## ADDED Requirements

### Requirement: El Paso 0 de la Doctrina Proyecta el Carril del Proyecto (REQ-27.16)

Las instrucciones de enrutamiento gestionadas DEBEN proyectar un paso 0 que instruya al orquestador a leer el campo `workflow` de `axiom sdd status` (REQ-27.2 de `axiom-sdd-cli-integration`) antes de decidir el carril de un trabajo sustancial, y que defina: con `sdd`, entrada directa al pre-vuelo de SDD sin preguntar el carril; con `odd`, trabajo en ODD sin preguntar el carril; con `ask`, la pregunta bloqueante actual (REQ-21.1 de `sdd-lane-kickoff`); y con `no_workspace`, `invalid` o sin proyección disponible, el comportamiento de `ask`. El texto es el mismo para todos los proyectos y adaptadores: NO DEBE contener el valor del carril de ningún proyecto, porque la doctrina no recibe datos del proyecto. El valor de la política de selección de SDD del manifiesto canónico de capacidades DEBE describir que SDD se selecciona por petición explícita, por carril declarado en el proyecto o por propuesta aceptada, y la proyección DEBE fallar cerrado (`ErrUnknownRoutingPolicy`) ante cualquier otro valor de la política. El riesgo, el tamaño o el número de ficheros NO DEBEN forzar SDD. La proyección DEBE seguir siendo determinista.

#### Scenario: El paso 0 instruye a leer la proyección

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el paso 0 instruye a leer `workflow` de `axiom sdd status`
- **Y** define el comportamiento de `sdd`, `odd` y `ask` y el de `no_workspace`, `invalid` o ausencia de proyección

#### Scenario: La doctrina es idéntica para todos los proyectos

- **DADO** dos proyectos con `workflow.lane` distinto y el mismo adaptador
- **CUANDO** se renderizan las instrucciones de enrutamiento
- **ENTONCES** el texto es idéntico byte a byte
- **Y** no contiene el valor del carril de ninguno de los dos

#### Scenario: La política del manifiesto admite el carril declarado

- **DADO** el manifiesto canónico de capacidades
- **CUANDO** se inspecciona (prueba) la política de selección de SDD
- **ENTONCES** su valor describe petición explícita, carril declarado en el proyecto o propuesta aceptada
- **Y** el valor anterior (`explicit_request_or_accepted_proposal`) ya no es aceptado

#### Scenario: Una política desconocida falla cerrado

- **DADO** un manifiesto con un valor de la política de selección de SDD distinto del vigente
- **CUANDO** se renderizan las instrucciones de enrutamiento
- **ENTONCES** el renderizado falla con `ErrUnknownRoutingPolicy`
- **Y** no se emite ningún texto parcial

#### Scenario: El riesgo no fuerza SDD

- **DADO** un proyecto con carril proyectado `ask` y un trabajo de alto riesgo
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** no selecciona SDD por el riesgo, el tamaño o el número de ficheros
- **Y** solo lo selecciona por petición explícita, carril declarado o propuesta aceptada

---

### Requirement: La Doctrina ODD Generada no Contiene Ninguna Parada (REQ-27.17)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD NO DEBEN contener ninguna parada: ni tras crear el documento de feature, ni la viñeta `Plan stop:`, ni el párrafo de reanudación con cesión del plan, ni la sección de salidas de las paradas (`Stops and exits`). El texto de las tres salidas (fuente única `stop-exits.md`) DEBE ser exclusivo de las paradas de SDD (REQ-26.8 de `sdd-block-review-gates`) y NO DEBE mencionar ODD. El orquestador NO DEBE detenerse en ODD tras crear `odd/tasks/<feature>.md`: tras registrar el documento y su espejo continúa con la implementación tarea a tarea. La regla de reanudación DEBE conservarse: parte del fichero `odd/tasks/<feature>.md` y tolera la ausencia del espejo de Engram. El paso 0 DEBE seguir sin formular ninguna pregunta de gobernanza una vez elegido el carril ODD. La proyección DEBE seguir siendo determinista.

#### Scenario: Ningún texto ODD generado contiene una parada

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto no contiene `Plan stop`, `Stops and exits` ni la mención de una parada del protocolo ODD
- **Y** el paso de registro del protocolo no ordena presentar ninguna parada

#### Scenario: El texto de las salidas es exclusivo de SDD

- **DADO** el recurso `stop-exits.md` y la doctrina ODD generada
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** `stop-exits.md` no menciona ODD
- **Y** la doctrina ODD generada no incluye el texto de las tres salidas

#### Scenario: Los orquestadores SDD conservan el texto de las salidas

- **DADO** las plantillas de orquestador SDD de todos los runtimes
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** todas contienen el mismo texto de las tres salidas procedente de `stop-exits.md`

#### Scenario: La reanudación sin espejo de Engram se conserva

- **DADO** una feature ODD con `odd/tasks/<feature>.md` y sin espejo de Engram en esa máquina
- **CUANDO** el orquestador la reanuda según la doctrina
- **ENTONCES** parte del fichero y no falla por la ausencia del espejo
- **Y** continúa por la primera tarea sin completar

#### Scenario: El renderizado sigue siendo determinista

- **DADO** una sincronización de adaptador
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado

## MODIFIED Requirements

### Requirement: Canonical provider-owned routing facts

Managed routing instructions MUST render from the canonical typed capability
manifest. They MUST NOT maintain an independent prompt event catalog, risk
router, or implementation-route selector. The SDD selection policy fact of the
manifest MUST state that SDD is selected only by an explicit request, by a
lane declared by the project, or by an accepted proposal (REQ-27.16).
(Previously: the manifest policy fact allowed only an explicit request or an accepted proposal)

#### Scenario: Adapter routing is rendered

- GIVEN a provider-registered adapter with a valid capability manifest
- WHEN its managed routing instructions are rendered
- THEN direct, delegated-direct, and optional-SDD facts equal the canonical manifest
- AND the rendered prose does not become runtime authority

#### Scenario: The SDD selection policy includes the project lane

- GIVEN the canonical capability manifest
- WHEN the SDD selection policy fact is read
- THEN it names explicit request, project-declared lane, and accepted proposal as the only ways to select SDD

### Requirement: SDD remains optional

The routing projection MUST propose SDD only when work is genuinely substantial
or ambiguous and durable proposal, specification, design, or task artifacts
would materially reduce uncertainty. SDD MUST be selected only after an
explicit request, an accepted proposal, or a lane declared by the project
(`workflow.lane: sdd`, REQ-27.3 of `sdd-lane-kickoff`), which counts as an
explicit request. Risk alone MUST NOT force SDD.
(Previously: SDD was selected only after an explicit request or an accepted proposal)

#### Scenario: User accepts an SDD proposal

- GIVEN substantial ambiguity would benefit from durable planning artifacts
- AND the user did not explicitly request SDD
- AND the project lane is `ask`
- WHEN the orchestrator proposes SDD
- THEN the proposal remains a pending decision until accepted
- AND only acceptance may bind the `sdd` implementation route

#### Scenario: User declines SDD

- GIVEN SDD was proposed and declined
- WHEN a safe smaller topology exists
- THEN work continues through a justified direct or delegated-direct route
- AND no SDD artifacts are fabricated

#### Scenario: The project lane counts as explicit selection

- GIVEN the project declares `workflow.lane: sdd` and the work is substantial
- WHEN the orchestrator routes the work
- THEN SDD is selected without a separate proposal
- AND small, understood work still creates no durable artifacts

#### Scenario: A declared `odd` lane keeps SDD out unless requested

- GIVEN the project declares `workflow.lane: odd`
- AND the user did not explicitly request SDD for this request
- WHEN the orchestrator routes substantial work
- THEN routing toward `optional-SDD` remains unactivated

### Requirement: Organic Driven Development is the orchestrator's mandatory default protocol

Managed routing instructions MUST project the Organic Driven Development (ODD)
protocol as the orchestrator's default workflow for every request, executed
first and without the user having to request it explicitly or ask about
planning or task tracking, per the upstream absorption (commits `70c774f8`,
`1b202d77`, `cfc415ce`). This projection MUST include, at minimum,
commit-per-work-unit, configured TDD, and feature continuity on resume.
Routing instructions toward SDD (`direct_inline`, `delegated-direct`,
`optional-SDD`) MUST NOT replace, precede, or defer the ODD projection; SDD
activates only after an explicit user selection, a lane declared by the
project (`workflow.lane: sdd`, which counts as an explicit selection) or an
accepted SDD proposal, exactly as the "SDD remains optional" requirement of
this same capability already governs.
(Previously: SDD activated only after an explicit user selection or an accepted SDD proposal)

#### Scenario: ODD is projected before any routing decision

- GIVEN a provider-registered adapter with a valid capability manifest
- WHEN its managed routing instructions are rendered
- THEN the ODD protocol is projected as the default workflow, executed before
  `direct_inline`, `delegated-direct`, or `optional-SDD` are evaluated
- AND none of those three routing facets is presented as a substitute for the
  ODD projection

#### Scenario: SDD still requires explicit selection even with ODD as default

- GIVEN ODD is projected as the orchestrator's default protocol
- WHEN work has not received an explicit SDD request, a project-declared `sdd` lane, nor an accepted SDD
  proposal
- THEN routing toward `optional-SDD` remains unactivated
- AND work continues under the ODD projection or another direct
  implementation route

### Requirement: El Protocolo ODD Prepara la Rama con `axiom branch ensure` (REQ-25.40)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN sustituir la regla «branch first when on the default branch» por la ejecución de `axiom branch ensure --lane odd --change <feature>` antes de la primera escritura del documento vivo `odd/tasks/<feature>.md` y de cualquier escritura de código, y DEBEN indicar que las órdenes posteriores usan como `--cwd` las rutas absolutas que devuelve su JSON. El cierre de cada tarea mantiene el commit por unidad de trabajo, ahora en la rama que `ensure` preparó. La regla NO DEBE alterar los demás pasos del protocolo ODD ni el carril SDD, NO DEBE exigir interacción con el usuario salvo el resultado `decision_required` del modo `ask`, y la proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Esta regla no activa SDD ni crea artefactos SDD.
(Previously: el escenario «El resto del protocolo no cambia» tenía en cuenta la parada única tras el plan de REQ-26.10, que se retira)

#### Scenario: El paso de implementación invoca `ensure`
- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el protocolo ODD indica ejecutar `axiom branch ensure --lane odd --change <feature>` antes de la primera escritura
- **Y** el texto ya no contiene «branch first when on the default branch»

#### Scenario: Las rutas del resultado se usan como `--cwd`
- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de implementación
- **ENTONCES** indica usar las rutas absolutas del JSON de `ensure` como `--cwd` de las órdenes posteriores

#### Scenario: El resto del protocolo no cambia
- **DADO** las instrucciones renderizadas antes y después de la regla de rama de este requisito
- **CUANDO** se comparan los demás pasos del protocolo ODD (autorizar, explorar, resolver incertidumbre, clasificar, registrar, implementar, cerrar) y el carril SDD
- **ENTONCES** son idénticos salvo la sustitución de la regla de rama

#### Scenario: El renderizado sigue siendo determinista
- **DADO** una sincronización de adaptador
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado

#### Scenario: El modo `ask` no añade una pregunta de gobernanza
- **DADO** las instrucciones renderizadas
- **CUANDO** se inspecciona (pin de texto) la regla
- **ENTONCES** la única interacción que prevé es presentar el resultado `decision_required` y repetir `ensure` con la respuesta

## REMOVED Requirements

### Requirement: El Protocolo ODD Incluye una Parada Única con Tres Salidas tras Crear el Documento de Feature (REQ-26.10)

(Reason: el carril ODD deja de tener paradas; la parada única tras el plan se percibió como fricción y el usuario pidió revertirla. Las paradas con tres salidas son exclusivas de SDD en modalidad `checkpointed`.)
(Migration: el comportamiento de ODD sin parada lo fija REQ-27.17; la regla de reanudación desde el fichero sin espejo de Engram se conserva en REQ-27.17; REQ-21.2 de `sdd-lane-kickoff` y REQ-26.1 y REQ-26.8 de `sdd-block-review-gates` dejan de citar REQ-26.10; el texto `stop-exits.md` pasa a ser exclusivo de SDD.)
