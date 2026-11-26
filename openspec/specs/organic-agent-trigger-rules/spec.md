# Organic Agent Routing Projection

## Purpose

Define the managed, provider-neutral instructions that project Gentle AI's
canonical implementation-routing facts into each supported adapter. The
projection helps an orchestrator choose the smallest useful implementation
topology; it does not own runtime route admission, verification, review,
delivery, or lifecycle transitions.

## Requirements

### Requirement: Canonical provider-owned routing facts (RF-RUT-01)

Managed routing instructions MUST render from the canonical typed capability
manifest. They MUST NOT maintain an independent prompt event catalog, risk
router, or implementation-route selector. The SDD selection policy fact of the
manifest MUST state that SDD is selected only by an explicit request, by a
lane declared by the project, or by an accepted proposal (RF-RUT-17).

#### Scenario: Adapter routing is rendered

- GIVEN a provider-registered adapter with a valid capability manifest
- WHEN its managed routing instructions are rendered
- THEN direct, delegated-direct, and optional-SDD facts equal the canonical manifest
- AND the rendered prose does not become runtime authority

#### Scenario: The SDD selection policy includes the project lane

- GIVEN the canonical capability manifest
- WHEN the SDD selection policy fact is read
- THEN it names explicit request, project-declared lane, and accepted proposal as the only ways to select SDD

### Requirement: Direct-inline boundary (RF-RUT-02)

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

### Requirement: Delegated-direct boundary (RF-RUT-03)

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

### Requirement: Delegation applies per action (RF-RUT-04)

Implementation route and action-level delegation MUST remain independent.
Tests, builds, installs, and owner-selected checking actors MAY use fresh
workers without changing a direct implementation route or creating an SDD run.

#### Scenario: Direct implementation delegates a build

- GIVEN implementation is already bound to `direct_inline`
- WHEN an execution-heavy build needs a fresh worker
- THEN the build may be delegated
- AND the implementation route remains `direct_inline`
- AND no synthetic SDD state is created

### Requirement: SDD remains optional (RF-RUT-05)

The routing projection MUST propose SDD only when work is genuinely substantial
or ambiguous and durable proposal, specification, design, or task artifacts
would materially reduce uncertainty. SDD MUST be selected only after an
explicit request, an accepted proposal, or a lane declared by the project
(`workflow.lane: sdd`, RF-KOF-09 of `sdd-lane-kickoff`), which counts as an
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

### Requirement: Capability manifests describe facts, not outcomes (RF-RUT-06)

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

### Requirement: Contract exposure fails closed (RF-RUT-07)

Typed work-routing capability MUST distinguish dormant from advertised
exposure. A dormant, disabled, unknown, or read-only capability MUST NOT be
presented as writable runtime authority.

#### Scenario: Work routing is dormant

- GIVEN an adapter manifest whose work-routing contract is dormant
- WHEN managed guidance is used
- THEN no productive transition is advertised
- AND the orchestrator cannot infer a transition from the static routing facts

### Requirement: Runtime transitions are provider-issued (RF-RUT-08)

When a managed WorkRun and advertised capability exist, instructions MUST
request the exact typed work-status contract and apply only its zero-or-one
provider-issued `authorizedTransition` through the typed work-transition
contract. Instructions MUST NOT reconstruct flags, infer
PASS from prose or process exit, or retry stale or mismatched authorization.
(Previously: the prohibition also covered selecting review lenses; RDD review lenses are retired (C-01-e).)

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

### Requirement: Public progress remains outcome-first (RF-RUT-09)

Managed routing instructions MUST present only **Working**, **Checking**,
**Ready**, and **Needs your decision** as normal progress states. Hashes, internal identifiers, and recovery operations MUST remain internal unless
maintainer diagnostics are explicitly requested.
(Previously: the internal-only list also named lineages, lenses, and receipts, and the scenario described provider-owned review; RDD is retired (C-01-e).)

#### Scenario: Functional verification is in progress

- GIVEN implementation has entered functional verification (tests, builds, and checks)
- WHEN progress is presented to the user
- THEN the visible state is **Checking**
- AND prompt-level review ceremonies are not exposed as extra workflow states

### Requirement: No prompt-owned lifecycle router (RF-RUT-10)

Managed routing instructions MUST NOT bind pre-commit, pre-push, pre-PR,
release, SDD-phase, CI, or schedule events to review actions. They MUST NOT
select a delivery decision.
That decision remains with its typed owner context.
(Previously: the prohibition also covered selecting a reviewer, refuter, correction, or receipt outcome, and the scenario a lens-selection table; those RDD roles are retired (C-01-e).)

#### Scenario: Managed rules are installed

- GIVEN adapter synchronization renders the managed routing block
- WHEN the block is inspected
- THEN it contains no event-to-review binding table
- AND later gates consume owner-issued evidence and authorization

### Requirement: Deterministic text-only installation (RF-RUT-11)

Routing rendering MUST remain deterministic, marker-free text generation.
Installation MAY update managed configuration files, but MUST NOT create Git
hooks, evaluate events in a daemon, invoke an agent, or start runtime work.

#### Scenario: Routing rules are installed

- GIVEN an adapter synchronization
- WHEN managed routing instructions are rendered and injected
- THEN repeated rendering produces identical text
- AND no review, verification, or delivery effect launches

### Requirement: User-owned model configuration (RF-RUT-12)

Routing policy MUST keep model, provider, profile, and effort as user-owned
choices. Those choices MUST NOT change the canonical routing thresholds or
become inputs to typed safety decisions.

#### Scenario: Runtime preferences change

- GIVEN the user selects a different model or effort
- WHEN routing instructions are evaluated
- THEN canonical direct, delegated-direct, and optional-SDD facts are unchanged
- AND no safety outcome is inferred from the runtime preference

### Requirement: Organic Driven Development is the orchestrator's mandatory default protocol (RF-RUT-13)

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

### Requirement: El cierre de ODD mantiene al día el contexto técnico mantenido (RF-RUT-14)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN incluir, en sus pasos de exploración y de cierre, estas reglas mínimas, en un texto idéntico para todos los proyectos y adaptadores:

1. Lectura del contexto técnico: el paso de exploración DEBE indicar que, si el proyecto declara un contexto técnico mantenido, se obtienen los localizadores con `axiom workspace context --cwd <repo> --json`, se abre primero `technicalContext.maintained.entry` y se consultan los demás `technicalContext.maintained.paths` solo bajo demanda. Sin contexto técnico mantenido declarado, el paso NO DEBE exigir ninguna lectura adicional. La regla NO DEBE depender de `axiom sdd status --json` ni de un cambio SDD.
2. Contexto técnico: el cierre DEBE indicar que, si el proyecto declara un contexto técnico mantenido y una unidad de trabajo cambia algo que ese contexto documenta, la documentación afectada se actualiza en el mismo commit de esa unidad de trabajo, con los localizadores leídos en la exploración (la salida de `axiom workspace context --cwd <repo> --json`). La regla NO DEBE exigir nada cuando el proyecto no declara contexto mantenido y NO DEBE autorizar la modificación de documentación declarada como de referencia (`technical_context.reference`).
3. Spec viva: el cierre DEBE indicar que, cuando el trabajo cambia o debe revisar la spec viva, la delta OpenSpec se escribe en la sección `## Spec viva` del documento `odd/tasks/<feature>.md`; que todo requisito ADDED lleva un identificador provisional (o, si el autor lo prefiere, un identificador definitivo, que se sigue admitiendo con la comprobación de colisión) y que `axiom archive integrate --odd <feature>` asigna el número definitivo y reescribe el documento; que todo bug (señalado con `Tipo: bug`) aporta una `## Spec viva` con contenido o el marcador de la sección `## Impacto en la spec` (la línea `Sin impacto en la spec` más una línea `Motivo:` no vacía, el mismo contrato que el carril SDD; el bug ODD no exige `## Informe de bug`); y que se ejecuta `axiom archive integrate --odd <feature>` en el commit de la unidad de trabajo que cierra la feature, sin paradas (RF-RUT-18). La integración en `openspec/specs/` y la reescritura de los identificadores del propio documento ODD son las únicas escrituras permitidas fuera del código.
4. Remisión: el cierre NO DEBE contener el valor del carril ni la condición ISO de ningún proyecto; la negativa de la integración en proyectos ISO la emite el verbo (RF-OLS-07 de `odd-living-spec-integration`).

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
### Requirement: El Protocolo ODD Prepara la Rama con `axiom branch ensure` (RF-RUT-15)

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

### Requirement: Prohibición de `EnterWorktree` y `ExitWorktree` en los Assets del Producto (RF-RUT-16)

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
### Requirement: El Paso 0 de la Doctrina Proyecta el Carril del Proyecto (RF-RUT-17)

Las instrucciones de enrutamiento gestionadas DEBEN proyectar un paso 0 que instruya al orquestador a leer el campo `workflow` de `axiom sdd status` (RF-CLI-08 de `axiom-sdd-cli-integration`) antes de decidir el carril de un trabajo sustancial, y que defina: con `sdd`, entrada directa al pre-vuelo de SDD sin preguntar el carril; con `odd`, trabajo en ODD sin preguntar el carril; con `ask`, la pregunta bloqueante actual (RF-KOF-01 de `sdd-lane-kickoff`); y con `no_workspace`, `invalid` o sin proyección disponible, el comportamiento de `ask`. El texto es el mismo para todos los proyectos y adaptadores: NO DEBE contener el valor del carril de ningún proyecto, porque la doctrina no recibe datos del proyecto. El valor de la política de selección de SDD del manifiesto canónico de capacidades DEBE describir que SDD se selecciona por petición explícita, por carril declarado en el proyecto o por propuesta aceptada, y la proyección DEBE fallar cerrado (`ErrUnknownRoutingPolicy`) ante cualquier otro valor de la política. El riesgo, el tamaño o el número de ficheros NO DEBEN forzar SDD. La proyección DEBE seguir siendo determinista.

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

### Requirement: La Doctrina ODD Generada no Contiene Ninguna Parada (RF-RUT-18)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD NO DEBEN contener ninguna parada: ni tras crear el documento de feature, ni la viñeta `Plan stop:`, ni el párrafo de reanudación con cesión del plan, ni la sección de salidas de las paradas (`Stops and exits`). El texto de las tres salidas (fuente única `stop-exits.md`) DEBE ser exclusivo de las paradas de SDD (RF-GAT-14 de `sdd-block-review-gates`) y NO DEBE mencionar ODD. El orquestador NO DEBE detenerse en ODD tras crear `odd/tasks/<feature>.md`: tras registrar el documento y su espejo continúa con la implementación tarea a tarea. La regla de reanudación DEBE conservarse: parte del fichero `odd/tasks/<feature>.md` y tolera la ausencia del espejo de Engram. El paso 0 DEBE seguir sin formular ninguna pregunta de gobernanza una vez elegido el carril ODD. La proyección DEBE seguir siendo determinista.

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

### Requirement: La Doctrina ODD Remite Todo el Trabajo de un Proyecto ISO al SDD (RF-RUT-19)

La doctrina ODD generada (el paso 0, que lee `workflow` de `axiom sdd status`, o el cierre del protocolo) DEBE expresar que, cuando `axiom sdd status` proyecta que el proyecto declara ISO (RF-ISO-02 de `sdd-iso-evidence`; todavía no hay cambio sellado, así que se lee la declaración vigente del proyecto), todo el trabajo se tramita por el carril SDD, sin preguntar el carril, también cuando el carril efectivo sea `ask`, y sin crear el documento `odd/tasks/<feature>.md`; y que un bug se tramita siempre por el SDD con el perfil de bug (RF-CLI-10 de `axiom-sdd-cli-integration`), que tiene tres rutas: sin delta y con `Cubre:` a un requisito existente, cuando restaura un requisito que ya existe; sin delta y con la sección `## Impacto en la spec` (línea `Sin impacto en la spec` y línea `Motivo:`), cuando no toca ningún requisito; y con delta, como cambio normal, cuando el requisito estaba mal especificado o faltaba. En las dos primeras el bug lleva además la sección `## Informe de bug` en `proposal.md`. La integración de ODD en la spec viva (el verbo de integración de ODD y la revisión de impacto en ODD) la diseña P6 y queda fuera de este requisito: la doctrina no afirma ni niega ninguna regla sobre el mecanismo con el que ODD la integra. El texto DEBE ser idéntico para todos los proyectos y adaptadores (RF-RUT-17): no contiene el valor del carril ni la condición ISO de ningún proyecto, solo la instrucción condicionada a lo que proyecta el status. NO DEBE contener ninguna parada ni el texto de las tres salidas (RF-RUT-18), y el resto del protocolo ODD (preparación de la rama con `axiom branch ensure`, commit por unidad de trabajo, reanudación desde `odd/tasks/<feature>.md`) NO cambia para los proyectos que no declaran ISO. La combinación de `sdd.iso: true` con `workflow.lane: odd` no llega a la doctrina: `axiom.yaml` la rechaza como declaración contradictoria (RF-ISO-01 de `sdd-iso-evidence`), así que la doctrina no necesita arbitrar entre ambos valores.

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
- **ENTONCES** contienen la sección `## Impacto en la spec` y la línea literal `Sin impacto en la spec` exactamente como las fija RF-CLI-10, y la sección `## Informe de bug` con `Observado:` y `Esperado:`

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
- **ENTONCES** no contienen ninguna regla de precedencia entre `sdd.iso` y `workflow.lane: odd`, porque esa combinación la rechaza la validación (RF-ISO-01 de `sdd-iso-evidence`)

#### Scenario: La adición no reintroduce ninguna parada

- **DADO** las instrucciones renderizadas con esta adición
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no contienen `Plan stop`, `Stops and exits` ni el texto de las tres salidas
- **Y** la preparación de la rama, el commit por unidad de trabajo y la reanudación sin espejo de Engram se expresan igual que antes
### Requirement: La Doctrina ODD de Entrega es `single-pr` por Defecto, la Cadena Solo a Petición y la Heurística por Tarea no Lleva Ningún Número (RF-RUT-20)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN declarar la doctrina de entrega del trabajo organic en estos términos, en el mismo texto para todos los proyectos y adaptadores:

1. **Una PR por defecto.** La entrega es `single-pr`: una sola PR con un commit convencional por tarea (unidad de trabajo). La estrategia NO se pregunta.
2. **Cadena solo a petición.** Las PRs encadenadas (`chained`, con `stacked-to-main` o `feature-branch-chain`) solo se usan cuando la persona las pide; si las pide sin indicar la estrategia de cadena, el orquestador formula una única pregunta por ella. Las skills se resuelven por su nombre de registro: `work-unit-commits` siempre y `chained-pr` solo al encadenar.
3. **La unidad de revisión es la tarea, no un conteo de líneas.** La heurística por tarea DEBE describir la tarea como el comportamiento coherente más pequeño con sus tests y su documentación, DEBE prohibir borrar espacios, líneas en blanco, comentarios o tests para ahorrar líneas y partir artificialmente, y NO DEBE contener ningún número de líneas ni referencia a «size gates» de PR.
4. **Aviso opcional de cambio grande.** Solo si `axiom.yaml` declara `delivery.split_hint_lines` (RF-TOP-14 de `workspace-topology`) y la previsión de líneas del documento de la feature la supera, el orquestador avisa una vez de que el cambio es grande y de que la persona puede pedir PRs encadenadas, y continúa sin esperar respuesta (RF-TOP-15 de `workspace-topology`). Sin la clave no hay aviso ni ningún número en el texto.

La doctrina NO DEBE contener los valores `ask-on-risk`, `auto-chain` ni `exception-ok` como estrategias ofrecidas, ni la elección de una estrategia de cadena condicionada a un umbral numérico. Los valores heredados que se lean de un documento existente se tratan con los alias de lectura de RF-ORQ-20 de `sdd-orchestrator-assets`. El resto del protocolo ODD (rama preparada con `axiom branch ensure`, commit por unidad de trabajo, reanudación desde `odd/tasks/<feature>.md`, TDD configurado) NO cambia. La proyección DEBE seguir siendo determinista. Quedan fuera de este requisito, por no ser umbrales de entrega, el umbral de barrido de judgment-day.
(Previously: la lista de exclusiones citaba también el umbral de riesgo del paquete de revisión de RDD y las lentes `review-*` y `rdd-defect-workflow` de RDD; RDD se retira (C-01-e, P20).)

#### Scenario: La doctrina ODD declara `single-pr` por defecto sin preguntar

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto declara la entrega `single-pr` (una PR con un commit por tarea) como valor por defecto
- **Y** no ordena preguntar la estrategia de PR

#### Scenario: La cadena solo se usa a petición

- **DADO** la doctrina ODD generada
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** declara `chained` con `stacked-to-main` y `feature-branch-chain` solo cuando la persona lo pide
- **Y** indica una única pregunta por la estrategia de cadena si la persona no la indicó
- **Y** indica resolver `work-unit-commits` siempre y `chained-pr` solo al encadenar

#### Scenario: La heurística por tarea no contiene ningún número

- **DADO** la doctrina ODD generada
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** la heurística de tamaño de tarea describe el comportamiento coherente más pequeño con sus tests y su documentación
- **Y** prohíbe borrar espacios, comentarios o tests por ahorrar líneas
- **Y** no contiene ningún número de líneas ni la frase `PR size gates`

#### Scenario: La doctrina no ofrece las estrategias heredadas

- **DADO** la doctrina ODD generada
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** no contiene `ask-on-risk`, `auto-chain` ni `exception-ok`

#### Scenario: El aviso de cambio grande es opcional y no bloquea

- **DADO** la doctrina ODD generada
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que, solo si `axiom.yaml` declara `delivery.split_hint_lines` y la previsión del documento de la feature la supera, se avisa una vez y se continúa sin bloquear
- **Y** no contiene ningún valor por defecto de esa clave

#### Scenario: La doctrina no contiene ningún umbral numérico de entrega

- **DADO** la doctrina ODD generada para todos los adaptadores y sus goldens regenerados
- **CUANDO** se busca un número de líneas como umbral de entrega
- **ENTONCES** no aparece ninguno
- **Y** el texto es idéntico en todos los adaptadores y determinista entre renderizados

#### Scenario: El resto del protocolo ODD no cambia

- **DADO** la doctrina ODD generada
- **CUANDO** se compara con la anterior en lo que no es entrega
- **ENTONCES** conserva la preparación de rama con `axiom branch ensure`, el commit por unidad de trabajo, la reanudación desde `odd/tasks/<feature>.md` y la ausencia de paradas (RF-RUT-18)
### Requirement: La Doctrina ODD Avisa por Commit en el Modo 1 Fijo, sin Preguntas ni Paradas (RF-RUT-21)

La doctrina ODD generada (paso 6 del protocolo) DEBE indicar que, al cerrar cada tarea con su commit de unidad de trabajo, el orquestador avisa en el chat con el sha, la tarea y la orden `git show <sha>`, y continúa sin esperar. ODD DEBE funcionar siempre en el comportamiento `notify`: la doctrina ODD NO DEBE preguntar el modo de revisión por commit, NO DEBE ofrecerlo ni aceptarlo a petición en el chat, y NO DEBE contener ninguno de los tokens `stop` o `stop-before-commit` como modo ni vocabulario de paradas, de modo que los pines de RF-RUT-18 sigan en verde. La doctrina ODD DEBE incluir además las reglas comunes de RF-ORQ-22: pedir la PR da por validados todos los commits pendientes y nada la bloquea; solo se registran, en `odd/tasks/<feature>.md`, los commits sobre los que la persona pidió cambios; los cambios sobre un commit ya superado usan `git commit --fixup=<sha>` con autosquash no interactivo diferido antes de abrir la PR si la rama no está publicada, o un commit correctivo nuevo si ya lo está, sin reescribir nunca historia publicada; y se recomienda fusionar sin squash. La doctrina ODD NO DEBE mencionar RDD ni recibos.

#### Scenario: El paso 6 contiene el aviso por commit

- **DADO** la doctrina ODD generada
- **CUANDO** se lee el paso 6
- **ENTONCES** pide avisar con sha, tarea y `git show <sha>` al cerrar cada tarea y continuar sin esperar

#### Scenario: ODD no contiene vocabulario de paradas ni del modo

- **DADO** la doctrina ODD generada
- **CUANDO** se busca `stop-before-commit`, una pregunta del modo o las tres salidas
- **ENTONCES** no aparece ninguno
- **Y** las pruebas de RF-RUT-18 (`routing_test.go`, `routing_stops_test.go`) siguen en verde sin cambios

#### Scenario: Una petición de parada en el chat no cambia el modo

- **DADO** una feature ODD en curso
- **CUANDO** la persona pide en el chat que se pare tras cada tarea
- **ENTONCES** la doctrina no define ningún mecanismo de parada por tarea y el comportamiento sigue siendo avisar y continuar

#### Scenario: El registro de commits con cambios pedidos va al documento ODD

- **DADO** la doctrina ODD generada
- **CUANDO** se lee la regla de registro
- **ENTONCES** indica `odd/tasks/<feature>.md` como destino y limita el registro a los commits con cambios pedidos

#### Scenario: La doctrina ODD declara validación implícita y merge sin squash

- **DADO** la doctrina ODD generada
- **CUANDO** se leen las reglas de la PR y de la fusión
- **ENTONCES** declaran que pedir la PR valida todos los commits pendientes, que nada la bloquea y que se recomienda merge commit sin squash
### Requirement: La Doctrina ODD Exige el WI con Plugin Activo y Rodea `ensure` con las Dos Fases (RF-RUT-22)

La doctrina ODD generada para todos los runtimes MUST indicar que, con el plugin de work items activo (la respuesta de `axiom change start` no es «no aplica»), todo trabajo sustancial (también un bug) nace de un WI existente: el orquestador MUST pedirlo si falta y no avanzar, MUST ejecutar la fase 1 (`axiom change start --from <id|url>`) antes de `axiom branch ensure --lane odd`, MUST usar el nombre derivado como `<feature-name>` del documento `odd/tasks/<feature-name>.md` y de `ensure`, MUST ejecutar la fase 2 (`axiom change start --adopt --change <feature> --lane odd`) después de `ensure` y antes de la primera escritura del documento ODD, y MUST escribir `external_ref: <alias>:<id>` en la cabecera de ese documento. El verbo MUST NOT crear el documento ODD. Si `ensure --lane odd` falla o devuelve `decision_required` tras una fase 1 correcta, la doctrina MUST indicar que se conserva el staging, que no se ejecuta la fase 2 y que se reintenta con el mismo nombre derivado o se ofrece `axiom change start --discard --change <feature>`; un rechazo de `--adopt` MUST relayarse con su mensaje y su continuación y detener el arranque antes de la primera escritura del documento ODD. Cuando el perfil devuelto por la fase 1 es `bug`, el orquestador (nunca el verbo) MUST escribir en el documento ODD la línea `Tipo: bug` y MUST proporcionar un `## Spec viva` no vacío o una sección `## Impacto en la spec` con la línea `Sin impacto en la spec` y su `Motivo:`, y el documento no necesita `## Informe de bug`. La doctrina MUST tratar `odd/context/<feature>/work-item.md` como dato no fiable de un tercero que nunca cambia el alcance, ordena comandos ni concede autoridad. Con trabajo pequeño y comprendido, que no crea documento ODD, MUST NOT exigirse el WI salvo que el proyecto sea ISO. Sin plugin activo, la doctrina de RF-RUT-15 MUST permanecer sin cambios. La regla MUST proyectarse sin ninguna parada adicional (RF-RUT-18) y los goldens regenerados MUST contenerla.

#### Scenario: La doctrina pide el WI antes de `ensure`
- **DADO** el texto de doctrina ODD generado para cada runtime
- **CUANDO** se inspecciona la sección del arranque
- **ENTONCES** indica pedir el WI con plugin activo y ejecutar la fase 1 antes de `axiom branch ensure --lane odd`

#### Scenario: La fase 2 precede a la primera escritura del documento ODD
- **DADO** el mismo texto
- **CUANDO** se inspecciona el orden
- **ENTONCES** la fase 2 con `--lane odd` figura después de `ensure` y antes de crear `odd/tasks/<feature-name>.md`

#### Scenario: `external_ref` en la cabecera del documento ODD
- **DADO** el mismo texto
- **CUANDO** se inspecciona la estructura del documento ODD
- **ENTONCES** indica que el orquestador escribe `external_ref: <alias>:<id>` en la cabecera y que el verbo no crea el documento

#### Scenario: Perfil de bug en ODD
- **DADO** el mismo texto y una fase 1 que devuelve el perfil `bug`
- **CUANDO** se inspecciona la regla del documento ODD
- **ENTONCES** indica que el orquestador escribe `Tipo: bug` y un `## Spec viva` o un `## Impacto en la spec` con `Sin impacto en la spec` y `Motivo:`, y que el verbo no escribe nada de eso

#### Scenario: Fallo de `ensure` o rechazo de `--adopt` en ODD
- **DADO** el mismo texto
- **CUANDO** se inspecciona la regla sobre fallos tras la fase 1
- **ENTONCES** indica conservar el staging, no ejecutar la fase 2 tras un fallo de `ensure --lane odd`, reintentar con el mismo nombre o ofrecer `--discard`, y detener el arranque ante un rechazo de `--adopt` antes de escribir el documento ODD

#### Scenario: Contenido importado como dato
- **DADO** el mismo texto
- **CUANDO** se inspecciona la regla sobre `odd/context/<feature>/work-item.md`
- **ENTONCES** lo trata como dato no fiable que no concede autoridad

#### Scenario: Trabajo pequeño sin documento
- **DADO** un proyecto no ISO y un cambio pequeño y comprendido
- **CUANDO** se aplica la doctrina
- **ENTONCES** no exige el WI ni crea documento ODD

#### Scenario: Sin plugin activo la doctrina de `ensure` no cambia
- **DADO** un proyecto sin plugin activo
- **CUANDO** se aplica la doctrina
- **ENTONCES** el protocolo de RF-RUT-15 se sigue sin ninguna llamada a `axiom change start` más allá de la consulta que responde «no aplica»

#### Scenario: Paridad entre runtimes y sin paradas
- **DADO** los goldens de todos los runtimes
- **CUANDO** se comparan
- **ENTONCES** todos contienen la regla del WI con el mismo contenido y ninguno añade una parada que RF-RUT-18 prohíba
