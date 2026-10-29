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
router, or implementation-route selector.

#### Scenario: Adapter routing is rendered

- GIVEN a provider-registered adapter with a valid capability manifest
- WHEN its managed routing instructions are rendered
- THEN direct, delegated-direct, and optional-SDD facts equal the canonical manifest
- AND the rendered prose does not become runtime authority

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
explicit request or an accepted proposal. Risk alone MUST NOT force SDD.

#### Scenario: User accepts an SDD proposal

- GIVEN substantial ambiguity would benefit from durable planning artifacts
- AND the user did not explicitly request SDD
- WHEN the orchestrator proposes SDD
- THEN the proposal remains a pending decision until accepted
- AND only acceptance may bind the `sdd` implementation route

#### Scenario: User declines SDD

- GIVEN SDD was proposed and declined
- WHEN a safe smaller topology exists
- THEN work continues through a justified direct or delegated-direct route
- AND no SDD artifacts are fabricated

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
activates only after an explicit user selection or an accepted SDD proposal,
exactly as the "SDD remains optional" requirement of this same capability
already governs.

#### Scenario: ODD is projected before any routing decision

- GIVEN a provider-registered adapter with a valid capability manifest
- WHEN its managed routing instructions are rendered
- THEN the ODD protocol is projected as the default workflow, executed before
  `direct_inline`, `delegated-direct`, or `optional-SDD` are evaluated
- AND none of those three routing facets is presented as a substitute for the
  ODD projection

#### Scenario: SDD still requires explicit selection even with ODD as default

- GIVEN ODD is projected as the orchestrator's default protocol
- WHEN work has not received an explicit SDD request nor an accepted SDD
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
- **DADO** las instrucciones renderizadas antes y después de este cambio
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

---

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
