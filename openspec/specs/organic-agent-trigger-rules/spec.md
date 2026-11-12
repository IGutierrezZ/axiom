# Organic Agent Routing Projection

## Purpose

Define the managed, provider-neutral instructions that project Gentle AI's
canonical implementation-routing facts into each supported adapter. The
projection helps an orchestrator choose the smallest useful implementation
topology; it does not own runtime route admission, verification, review,
delivery, or lifecycle transitions.

## Requirements

### Requirement: Canonical provider-owned routing facts

Managed routing instructions MUST render from the canonical typed capability
manifest. They MUST NOT maintain an independent prompt event catalog, risk
router, or implementation-route selector. The SDD selection policy fact of the
manifest MUST state that SDD is selected only by an explicit request, by a
lane declared by the project, or by an accepted proposal (REQ-27.16).

#### Scenario: Adapter routing is rendered

- GIVEN a provider-registered adapter with a valid capability manifest
- WHEN its managed routing instructions are rendered
- THEN direct, delegated-direct, and optional-SDD facts equal the canonical manifest
- AND the rendered prose does not become runtime authority

#### Scenario: The SDD selection policy includes the project lane

- GIVEN the canonical capability manifest
- WHEN the SDD selection policy fact is read
- THEN it names explicit request, project-declared lane, and accepted proposal as the only ways to select SDD

### Requirement: Direct-inline boundary

The routing projection MUST allow inline decision or verification when
understanding requires one to three files. It MAY keep one mechanical file
change inline only when the change is already understood, requires no research,
and has no unresolved design decision.

#### Scenario: Small understood change remains inline

- GIVEN one already-understood mechanical file change
- AND no research or design decision remains
- WHEN implementation topology is considered
- THEN `direct_inline` is an available route
- AND no SDD artifact is created

#### Scenario: Understanding expands to four files

- GIVEN the work requires understanding four or more files
- WHEN implementation topology is considered
- THEN the understanding step is not kept inline

### Requirement: Delegated-direct boundary

The routing projection MUST delegate narrow exploration when understanding
requires four or more files. It MUST delegate one writer for two or more
non-trivial implementation files, and MUST delegate reading that prepares a
write or broad research when the adapter supports delegation.

#### Scenario: Multi-file understanding delegates

- GIVEN implementation requires understanding four files
- WHEN the route is projected
- THEN narrow exploration uses delegated-direct topology
- AND delegation does not create an SDD lifecycle

#### Scenario: Two non-trivial files require one writer

- GIVEN implementation changes two non-trivial files
- WHEN the write topology is projected
- THEN one delegated writer owns the implementation
- AND the file count alone does not select SDD

### Requirement: Delegation applies per action

Implementation route and action-level delegation MUST remain independent.
Tests, builds, installs, and owner-selected checking actors MAY use fresh
workers without changing a direct implementation route or creating an SDD run.

#### Scenario: Direct implementation delegates a build

- GIVEN implementation is already bound to `direct_inline`
- WHEN an execution-heavy build needs a fresh worker
- THEN the build may be delegated
- AND the implementation route remains `direct_inline`
- AND no synthetic SDD state is created

### Requirement: SDD remains optional

The routing projection MUST propose SDD only when work is genuinely substantial
or ambiguous and durable proposal, specification, design, or task artifacts
would materially reduce uncertainty. SDD MUST be selected only after an
explicit request, an accepted proposal, or a lane declared by the project
(`workflow.lane: sdd`, REQ-27.3 of `sdd-lane-kickoff`), which counts as an
explicit request. Risk alone MUST NOT force SDD.

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

### Requirement: Capability manifests describe facts, not outcomes

Each provider-registered adapter MUST expose its own canonical typed capability
manifest. A manifest MUST describe static adapter features and routing facts
only; it MUST NOT claim runtime observations, select a route, issue a review
decision, or assert PASS. Unsupported adapter identities MUST be rejected.

#### Scenario: Adapter-specific features differ

- GIVEN two registered adapters have different native integration mechanisms
- WHEN their capability manifests are read
- THEN their feature claims may differ truthfully
- AND their canonical implementation-routing facts remain semantically equal

#### Scenario: Unknown adapter is requested

- GIVEN an adapter identity not registered by the provider
- WHEN its capability manifest is requested
- THEN the request is rejected
- AND no generic capability claim is invented

### Requirement: Contract exposure fails closed

Typed work-routing capability MUST distinguish dormant from advertised
exposure. A dormant, disabled, unknown, or read-only capability MUST NOT be
presented as writable runtime authority.

#### Scenario: Work routing is dormant

- GIVEN an adapter manifest whose work-routing contract is dormant
- WHEN managed guidance is used
- THEN no productive transition is advertised
- AND the orchestrator cannot infer a transition from the static routing facts

### Requirement: Runtime transitions are provider-issued

When a managed WorkRun and advertised capability exist, instructions MUST
request the exact typed work-status contract and apply only its zero-or-one
provider-issued `authorizedTransition` through the typed work-transition
contract. Instructions MUST NOT reconstruct flags, select review lenses, infer
PASS from prose or process exit, or retry stale or mismatched authorization.

#### Scenario: Status returns one authorized transition

- GIVEN a managed WorkRun
- AND typed status returns one current `authorizedTransition`
- WHEN the orchestrator advances the work
- THEN it applies that exact authorization and expected revision
- AND it does not synthesize an alternate operation

#### Scenario: Runtime authority is unavailable

- GIVEN the capability is unavailable, disabled, unknown, or read-only
- WHEN the orchestrator requests progress
- THEN it reports the typed stop
- AND it does not fall back to prompt-owned lifecycle authority

### Requirement: Public progress remains outcome-first

Managed routing instructions MUST present only **Working**, **Checking**,
**Ready**, and **Needs your decision** as normal progress states. Hashes,
lineages, lenses, receipts, and recovery operations MUST remain internal unless
maintainer diagnostics are explicitly requested.

#### Scenario: Provider performs bounded review

- GIVEN implementation has entered provider-owned verification and review
- WHEN progress is presented to the user
- THEN the visible state is **Checking**
- AND prompt-level review ceremonies are not exposed as extra workflow states

### Requirement: No prompt-owned lifecycle router

Managed routing instructions MUST NOT bind pre-commit, pre-push, pre-PR,
release, SDD-phase, CI, or schedule events to review actions. They MUST NOT
select a reviewer, refuter, correction, receipt outcome, or delivery decision.
Those decisions remain with their typed owner contexts.

#### Scenario: Managed rules are installed

- GIVEN adapter synchronization renders the managed routing block
- WHEN the block is inspected
- THEN it contains no event-to-review binding or lens-selection table
- AND later gates consume owner-issued evidence and authorization

### Requirement: Deterministic text-only installation

Routing rendering MUST remain deterministic, marker-free text generation.
Installation MAY update managed configuration files, but MUST NOT create Git
hooks, evaluate events in a daemon, invoke an agent, or start runtime work.

#### Scenario: Routing rules are installed

- GIVEN an adapter synchronization
- WHEN managed routing instructions are rendered and injected
- THEN repeated rendering produces identical text
- AND no review, verification, or delivery effect launches

### Requirement: User-owned model configuration

Routing policy MUST keep model, provider, profile, and effort as user-owned
choices. Those choices MUST NOT change the canonical routing thresholds or
become inputs to typed safety decisions.

#### Scenario: Runtime preferences change

- GIVEN the user selects a different model or effort
- WHEN routing instructions are evaluated
- THEN canonical direct, delegated-direct, and optional-SDD facts are unchanged
- AND no safety outcome is inferred from the runtime preference

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
### Requirement: El Protocolo ODD Prepara la Rama con `axiom branch ensure` (REQ-25.40)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN sustituir la regla «branch first when on the default branch» por la ejecución de `axiom branch ensure --lane odd --change <feature>` antes de la primera escritura del documento vivo `odd/tasks/<feature>.md` y de cualquier escritura de código, y DEBEN indicar que las órdenes posteriores usan como `--cwd` las rutas absolutas que devuelve su JSON. El cierre de cada tarea mantiene el commit por unidad de trabajo, ahora en la rama que `ensure` preparó. La regla NO DEBE alterar los demás pasos del protocolo ODD ni el carril SDD, NO DEBE exigir interacción con el usuario salvo el resultado `decision_required` del modo `ask`, y la proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Esta regla no activa SDD ni crea artefactos SDD.

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

### Requirement: Prohibición de `EnterWorktree` y `ExitWorktree` en los Assets del Producto (REQ-25.41)

La doctrina de enrutamiento generada (punto único de generación) DEBE prohibir expresamente el uso de las primitivas `EnterWorktree` y `ExitWorktree` y DEBE prescribir el mecanismo común para todos los runtimes: el JSON de `axiom branch ensure` con rutas absolutas, y el uso de `--cwd`, `git -C` y rutas absolutas en lugar de cambiar el directorio de trabajo. Esa línea de prohibición es la **única** aparición permitida de los dos nombres en los assets del producto: ningún otro asset (skills, orquestadores por runtime, comandos, guías) DEBE contenerlos ni prescribir cualquier primitiva propia de un runtime para crear o cambiar de worktree. El texto generado DEBE ser idéntico para todos los adaptadores, de modo que el comportamiento no dependa de primitivas propias de uno de ellos.

#### Scenario: La doctrina generada contiene la prohibición
- **DADO** un adaptador registrado
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** contienen la prohibición de `EnterWorktree` y `ExitWorktree` y el mecanismo de rutas absolutas, `--cwd` y `git -C`

#### Scenario: Ningún otro asset contiene los nombres
- **DADO** el conjunto completo de assets del producto (skills, orquestadores de cada runtime, comandos y guías)
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** `EnterWorktree` y `ExitWorktree` solo aparecen en la línea de prohibición generada por la doctrina de enrutamiento

#### Scenario: La doctrina es idéntica en todos los adaptadores
- **DADO** dos adaptadores registrados distintos (por ejemplo Claude Code y OpenCode)
- **CUANDO** se comparan las líneas de prohibición y de mecanismo común de su doctrina renderizada
- **ENTONCES** son idénticas

#### Scenario: Los goldens fijan el texto
- **DADO** los goldens de doctrina y de assets regenerados
- **CUANDO** se ejecuta la batería de pruebas
- **ENTONCES** una modificación de la línea de prohibición falla por diferencia con el golden
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
