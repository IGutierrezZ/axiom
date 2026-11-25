# Delta for organic-agent-trigger-rules

## MODIFIED Requirements

### Requirement: Public progress remains outcome-first

Managed routing instructions MUST present only **Working**, **Checking**,
**Ready**, and **Needs your decision** as normal progress states. Hashes, internal identifiers, and recovery operations MUST remain internal unless
maintainer diagnostics are explicitly requested.
(Previously: the internal-only list also named lineages, lenses, and receipts, and the scenario described provider-owned review; RDD is retired (C-01-e).)

#### Scenario: Functional verification is in progress

- GIVEN implementation has entered functional verification (tests, builds, and checks)
- WHEN progress is presented to the user
- THEN the visible state is **Checking**
- AND prompt-level review ceremonies are not exposed as extra workflow states

### Requirement: No prompt-owned lifecycle router

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

### Requirement: Runtime transitions are provider-issued

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

### Requirement: La Doctrina ODD de Entrega es `single-pr` por Defecto, la Cadena Solo a Petición y la Heurística por Tarea no Lleva Ningún Número (REQ-32.4)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN declarar la doctrina de entrega del trabajo organic en estos términos, en el mismo texto para todos los proyectos y adaptadores:

1. **Una PR por defecto.** La entrega es `single-pr`: una sola PR con un commit convencional por tarea (unidad de trabajo). La estrategia NO se pregunta.
2. **Cadena solo a petición.** Las PRs encadenadas (`chained`, con `stacked-to-main` o `feature-branch-chain`) solo se usan cuando la persona las pide; si las pide sin indicar la estrategia de cadena, el orquestador formula una única pregunta por ella. Las skills se resuelven por su nombre de registro: `work-unit-commits` siempre y `chained-pr` solo al encadenar.
3. **La unidad de revisión es la tarea, no un conteo de líneas.** La heurística por tarea DEBE describir la tarea como el comportamiento coherente más pequeño con sus tests y su documentación, DEBE prohibir borrar espacios, líneas en blanco, comentarios o tests para ahorrar líneas y partir artificialmente, y NO DEBE contener ningún número de líneas ni referencia a «size gates» de PR.
4. **Aviso opcional de cambio grande.** Solo si `axiom.yaml` declara `delivery.split_hint_lines` (REQ-32.1 de `workspace-topology`) y la previsión de líneas del documento de la feature la supera, el orquestador avisa una vez de que el cambio es grande y de que la persona puede pedir PRs encadenadas, y continúa sin esperar respuesta (REQ-32.2 de `workspace-topology`). Sin la clave no hay aviso ni ningún número en el texto.

La doctrina NO DEBE contener los valores `ask-on-risk`, `auto-chain` ni `exception-ok` como estrategias ofrecidas, ni la elección de una estrategia de cadena condicionada a un umbral numérico. Los valores heredados que se lean de un documento existente se tratan con los alias de lectura de REQ-32.3 de `sdd-orchestrator-assets`. El resto del protocolo ODD (rama preparada con `axiom branch ensure`, commit por unidad de trabajo, reanudación desde `odd/tasks/<feature>.md`, TDD configurado) NO cambia. La proyección DEBE seguir siendo determinista. Quedan fuera de este requisito, por no ser umbrales de entrega, el umbral de barrido de judgment-day.
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
- **ENTONCES** conserva la preparación de rama con `axiom branch ensure`, el commit por unidad de trabajo, la reanudación desde `odd/tasks/<feature>.md` y la ausencia de paradas (REQ-27.17)
