# SDD Orchestrator Assets Specification

## Purpose

Defines required behavior for agent-specific SDD orchestrator guidance assets so Claude and OpenCode guidance remain semantically aligned where intended, while preserving platform-accurate delegation wording.

---

## Requirements

### Requirement: Chain Strategy Guidance Parity

Los activos del orquestador SDD de Claude DEBEN incluir una guía explícita de estrategia de cadena equivalente a la de OpenCode para la planificación de entregas revisables. La guía DEBE enumerar exactamente estas estrategias seleccionables:
- `stacked-to-main`
- `feature-branch-chain`

La sección DEBE indicar que la estrategia de cadena solo se activa cuando la persona pide encadenar, apilar o rebanar la entrega (`delivery_strategy: chained`); NO DEBE activarla ninguna otra condición, ni un valor heredado, ni un tamaño, ni un riesgo.
(Previously: la guía enumeraba las dos estrategias sin indicar qué la activa; la activaban `ask-on-risk` y `auto-chain`)

#### Scenario: Claude guidance lists both required strategies

- **DADO** que la guía del orquestador de Claude se genera desde los activos embebidos
- **CUANDO** se renderiza la sección de estrategia de cadena
- **ENTONCES** incluye `stacked-to-main`
- **Y** incluye `feature-branch-chain`

#### Scenario: Strategy naming remains canonical

- **DADO** una validación estática de las etiquetas de estrategia de cadena en la guía de Claude
- **CUANDO** se ejecuta la comprobación
- **ENTONCES** ningún alias ni valor renombrado sustituye a los nombres canónicos de estrategia

#### Scenario: La cadena solo se activa a petición

- **DADO** la sección de estrategia de cadena del orquestador de Claude
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que solo se activa cuando la persona pide encadenar, apilar o rebanar
- **Y** no menciona `ask-on-risk` ni `auto-chain` como condición de activación

---

### Requirement: Chain Strategy Propagation to Downstream Phases

La guía del orquestador de Claude DEBE exigir pasar `chain_strategy` junto con `delivery_strategy` a `sdd-tasks` y `sdd-apply` cuando la persona haya pedido `chained`. Si la entrega es `single-pr` (valor por defecto), el orquestador DEBE reenviar solo `delivery_strategy: single-pr` y NO DEBE reenviar `chain_strategy`. Un `delivery_strategy: chained` reenviado con su `chain_strategy` prevalece sobre el valor por defecto `single-pr` del bloque de pre-vuelo.
(Previously: el reenvío de `chain_strategy` era incondicional siempre que la planificación de entrega fuera relevante)

#### Scenario: Guidance forwards both strategy fields to sdd-tasks on chained

- **DADO** que la persona pidió entrega encadenada y el orquestador invoca `sdd-tasks`
- **CUANDO** se renderiza la guía de invocación
- **ENTONCES** se reenvía `delivery_strategy: chained`
- **Y** se reenvía `chain_strategy`

#### Scenario: Guidance forwards both strategy fields to sdd-apply on chained

- **DADO** que la persona pidió entrega encadenada y el orquestador invoca `sdd-apply` tras la planificación
- **CUANDO** se renderiza la guía de invocación
- **ENTONCES** se reenvía `delivery_strategy: chained`
- **Y** se reenvía `chain_strategy`

#### Scenario: Con la entrega por defecto no se reenvía `chain_strategy`

- **DADO** un cambio sin petición de encadenar
- **CUANDO** el orquestador invoca `sdd-tasks` o `sdd-apply`
- **ENTONCES** reenvía `delivery_strategy: single-pr`
- **Y** no reenvía `chain_strategy`

---

### Requirement: Non-Claude Chain Strategy Guidance Parity

Los activos del orquestador SDD que no son de Claude y admiten planificación de entregas DEBEN incluir una guía explícita de estrategia de cadena equivalente en intención a la guía de referencia de Claude y OpenCode. La guía DEBE enumerar exactamente estas estrategias seleccionables:
- `stacked-to-main`
- `feature-branch-chain`

La sección DEBE indicar, igual que en Claude, que la estrategia de cadena solo se activa cuando la persona pide encadenar, apilar o rebanar la entrega.
(Previously: la guía enumeraba las dos estrategias sin indicar qué la activa)

#### Scenario: Non-Claude delivery-planning guidance lists both required strategies

- **DADO** un activo de orquestador no Claude que incluye guía de planificación de entregas
- **CUANDO** se renderiza la sección de estrategia de cadena
- **ENTONCES** incluye `stacked-to-main`
- **Y** incluye `feature-branch-chain`

#### Scenario: Non-Claude strategy naming remains canonical

- **DADO** una validación estática de las etiquetas de estrategia de cadena en los activos no Claude
- **CUANDO** se ejecuta la comprobación
- **ENTONCES** ningún alias ni valor renombrado sustituye a los nombres canónicos de estrategia

#### Scenario: La cadena solo se activa a petición en los activos no Claude

- **DADO** la sección de estrategia de cadena de cada orquestador no Claude
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que solo se activa a petición de la persona
- **Y** no menciona `ask-on-risk` ni `auto-chain` como condición de activación

---

### Requirement: Non-Claude Strategy Propagation to Downstream Phases

Los activos del orquestador SDD que no son de Claude y lanzan fases de entrega DEBEN exigir pasar `chain_strategy` junto con `delivery_strategy` a `sdd-tasks` y `sdd-apply` cuando la persona haya pedido `chained`; con la entrega por defecto `single-pr` solo se reenvía `delivery_strategy` y NO se reenvía `chain_strategy`.
(Previously: el reenvío de `chain_strategy` era incondicional siempre que la planificación de entrega fuera relevante)

#### Scenario: Non-Claude guidance forwards both strategy fields to sdd-tasks on chained

- **DADO** un activo de orquestador no Claude y una petición de entrega encadenada al invocar `sdd-tasks`
- **CUANDO** se renderiza la guía de invocación
- **ENTONCES** se reenvía `delivery_strategy`
- **Y** se reenvía `chain_strategy`

#### Scenario: Non-Claude guidance forwards both strategy fields to sdd-apply on chained

- **DADO** un activo de orquestador no Claude y una petición de entrega encadenada al invocar `sdd-apply` tras la planificación
- **CUANDO** se renderiza la guía de invocación
- **ENTONCES** se reenvía `delivery_strategy`
- **Y** se reenvía `chain_strategy`

#### Scenario: Non-Claude con la entrega por defecto no reenvía `chain_strategy`

- **DADO** un activo de orquestador no Claude y ninguna petición de encadenar
- **CUANDO** invoca `sdd-tasks` o `sdd-apply`
- **ENTONCES** reenvía solo `delivery_strategy: single-pr`

---

### Requirement: Platform-Native Solo Inline Semantics Preservation

For non-Claude solo-inline and platform-native orchestrator assets, the system MUST preserve platform-accurate inline execution wording and MUST NOT add inaccurate subagent or OpenCode persistence claims.

#### Scenario: Windsurf and Antigravity keep inline-accurate wording

- GIVEN Windsurf and Antigravity orchestrator assets are reviewed
- WHEN delegation and execution wording is validated
- THEN wording remains accurate to inline/platform-native execution
- AND wording does not claim persisted OpenCode plugin-backed background delegation

#### Scenario: Other non-Claude platform-native assets avoid inaccurate persistence claims

- GIVEN non-Claude platform-native orchestrator assets are reviewed
- WHEN async/delegation wording is validated
- THEN wording stays platform-accurate
- AND wording does not introduce inaccurate subagent or OpenCode persistence guarantees

---

### Requirement: chained-pr Skill Binding in Chain Strategy Guidance

Los 12 activos del orquestador SDD DEBEN enlazar la skill `chained-pr` en su guía de estrategia de cadena. Cuando la persona pida PRs encadenadas o apiladas, el orquestador DEBE resolver la skill `chained-pr` por su nombre de registro (no por una ruta del sistema de ficheros fija) mediante su Sub-Agent Launch Pattern, y DEBE inyectar la ruta resuelta en los prompts de las fases `sdd-tasks` y `sdd-apply` bajo `## Skills to load before work`. Esas fases DEBEN leer y seguir la skill ANTES de planificar o crear cualquier PR. Con la entrega `single-pr` por defecto la skill NO se resuelve ni se inyecta, y la propia skill `chained-pr` se activa solo cuando la persona pide encadenar, apilar o rebanar, o con `delivery_strategy: chained`, sin ningún umbral numérico.

El enlace DEBE referenciar la skill por el nombre de registro `chained-pr` (frontmatter `chained-pr`). No se permite ninguna ruta fija a un fichero `SKILL.md` como referencia del enlace.
(Previously: la skill se resolvía cuando la planificación de entrega producía PRs encadenadas, y la skill se activaba por un presupuesto de revisión numérico)

#### Scenario: Orchestrator resolves chained-pr skill by registry name on chained delivery

- **DADO** un activo de orquestador SDD con guía de estrategia de cadena
- **CUANDO** la persona pide entrega encadenada o apilada
- **ENTONCES** el orquestador resuelve la skill `chained-pr` por su nombre de registro mediante el Sub-Agent Launch Pattern
- **Y** la ruta resuelta se inyecta en el prompt de `sdd-tasks` bajo `## Skills to load before work`
- **Y** la ruta resuelta se inyecta en el prompt de `sdd-apply` bajo `## Skills to load before work`
- **Y** no se usa ninguna ruta fija como referencia del enlace

#### Scenario: Binding is present in all 12 orchestrator templates

- **DADO** las 12 plantillas del orquestador SDD (antigravity, claude, codex, cursor, gemini, generic, hermes, kimi, kiro, opencode, qwen, windsurf)
- **CUANDO** se inspecciona la sección de estrategia de cadena de cada una
- **ENTONCES** cada plantilla contiene una instrucción de enlace que referencia la skill `chained-pr` por nombre de registro
- **Y** la instrucción está dentro de la sección de estrategia de cadena o inmediatamente junto a ella

#### Scenario: sdd-tasks sub-agent reads chained-pr skill before planning PRs

- **DADO** un sub-agente `sdd-tasks` lanzado con un `chain_strategy` por petición de encadenar
- **CUANDO** el sub-agente empieza a planificar
- **ENTONCES** el prompt incluye la skill `chained-pr` bajo `## Skills to load before work`
- **Y** el sub-agente la lee y la sigue antes de producir un plan de PR

#### Scenario: sdd-apply sub-agent reads chained-pr skill before creating PRs

- **DADO** un sub-agente `sdd-apply` lanzado para un cambio con entrega encadenada pedida
- **CUANDO** el sub-agente empieza la implementación
- **ENTONCES** el prompt incluye la skill `chained-pr` bajo `## Skills to load before work`
- **Y** el sub-agente la lee y la sigue antes de crear cualquier PR

#### Scenario: La skill chained-pr no menciona ningún umbral numérico

- **DADO** el `description` y el cuerpo de la skill `chained-pr` y su referencia de detalles
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** la activación se describe como una petición de la persona de encadenar, apilar o rebanar
- **Y** no contienen ningún número de líneas como umbral ni `400-line budget risk`

---

### Requirement: Inline Chain Strategy Summary Retained

Los 12 activos del orquestador SDD DEBEN conservar un resumen en línea breve de las estrategias `stacked-to-main` y `feature-branch-chain` en su sección de estrategia de cadena. El enlace complementa, y NO sustituye, el resumen en línea existente. Los nombres canónicos de estrategia DEBEN seguir apareciendo literalmente para que los escenarios «Strategy naming remains canonical» sigan cumpliéndose. El resumen NO DEBE condicionar la cadena a ningún tamaño, riesgo o valor heredado: la cadena es a petición de la persona.
(Previously: el requisito no precisaba qué condiciona la cadena)

#### Scenario: Chain strategy inline summary coexists with skill binding

- **DADO** un activo de orquestador SDD con una sección de estrategia de cadena con el enlace a la skill
- **CUANDO** se inspecciona la sección
- **ENTONCES** `stacked-to-main` aparece como nombre canónico de estrategia
- **Y** `feature-branch-chain` aparece como nombre canónico de estrategia
- **Y** aparece también una instrucción de enlace que referencia la skill `chained-pr` por nombre de registro
- **Y** la instrucción de enlace no sustituye a los resúmenes en línea de las estrategias

#### Scenario: Static validation passes after binding is added

- **DADO** pruebas de validación estática que afirman los nombres canónicos de estrategia y la presencia de la sección de cadena
- **CUANDO** las pruebas se ejecutan contra las plantillas actualizadas
- **ENTONCES** todas las aserciones de nombres canónicos siguen pasando
- **Y** pasan también las nuevas aserciones de enlace
- **Y** ninguna prueba exige que el enlace haya sustituido al resumen en línea

---

### Requirement: Cursor Chain Strategy Section Parity

La plantilla `cursor/sdd-orchestrator.md` DEBE incluir una sección `### Chain Strategy` en paridad con las otras 11 plantillas. La sección DEBE enumerar ambos nombres canónicos de estrategia, reenviar `chain_strategy` a las fases posteriores cuando la persona haya pedido `chained` e incluir el enlace a la skill `chained-pr`.
(Previously: el reenvío de `chain_strategy` era incondicional)

#### Scenario: Cursor template includes a Chain Strategy section

- **DADO** la plantilla del orquestador SDD de cursor
- **CUANDO** se revisa la estructura del documento
- **ENTONCES** está presente una sección `### Chain Strategy`
- **Y** incluye `stacked-to-main`
- **Y** incluye `feature-branch-chain`

#### Scenario: Cursor forwards chain_strategy to downstream phases on chained

- **DADO** la plantilla cursor y una petición de entrega encadenada al invocar `sdd-tasks` o `sdd-apply`
- **CUANDO** se renderiza la guía de invocación
- **ENTONCES** se reenvía `delivery_strategy`
- **Y** se reenvía `chain_strategy`

#### Scenario: Cursor includes the chained-pr skill binding

- **DADO** la sección de estrategia de cadena de la plantilla cursor
- **CUANDO** se revisa la instrucción de enlace
- **ENTONCES** la skill `chained-pr` se referencia por nombre de registro
- **Y** el enlace indica la inyección en los prompts de `sdd-tasks` y `sdd-apply`

#### Scenario: Cursor chain strategy naming is canonical

- **DADO** una validación estática de las etiquetas de estrategia de cadena en la plantilla cursor
- **CUANDO** se ejecuta la comprobación
- **ENTONCES** ningún alias ni valor renombrado sustituye a los nombres canónicos `stacked-to-main` y `feature-branch-chain`

### Requirement: Platform-Accurate Binding Wording for Solo-Inline and Platform-Native Hosts

For solo-inline and platform-native orchestrator hosts (windsurf, antigravity, kimi, kiro), the skill-binding instruction MUST phrase "inject under `## Skills to load before work`" in a way that maps to each host's actual phase-context mechanism. The binding MUST NOT reintroduce OpenCode-style persisted-delegation claims that the existing spec forbids for these hosts.

#### Scenario: Solo-inline binding wording is platform-accurate

- GIVEN a solo-inline orchestrator asset (windsurf or antigravity) is reviewed
- WHEN the skill binding instruction is read
- THEN the binding references the `chained-pr` skill by registry name
- AND the wording maps skill injection to the host's inline or platform-native phase-context mechanism
- AND the wording does not claim persisted OpenCode plugin-backed background delegation

#### Scenario: Platform-native non-Claude binding wording avoids inaccurate persistence claims

- GIVEN a platform-native non-Claude orchestrator asset (kimi or kiro) is reviewed
- WHEN the skill binding instruction is read
- THEN the binding references the `chained-pr` skill by registry name
- AND the wording does not introduce inaccurate subagent or OpenCode persistence guarantees
- AND the wording stays platform-accurate per the existing solo-inline semantics requirement

---

### Requirement: Golden Fixture and Static Assertion Coverage for the Binding

Static-validation tests and golden fixtures that cover chain strategy content MUST be updated to assert the presence of the `chained-pr` skill binding. Golden churn MUST be limited to the intended binding wording and the new Cursor chain strategy section; no unrelated content changes are permitted.

#### Scenario: New static assertions verify binding presence in all 12 templates

- GIVEN static-validation tests run after the change is applied
- WHEN binding assertions execute
- THEN each of the 12 orchestrator templates has a binding assertion that passes
- AND the assertions verify the registry-name reference, not a hardcoded path

#### Scenario: Cursor is added to chain-strategy parity static assertions

- GIVEN the existing chain-strategy parity assertions are updated to include cursor
- WHEN static tests execute
- THEN a chain-strategy parity assertion for the cursor template passes
- AND the assertion verifies that canonical strategy names are present

#### Scenario: Golden fixtures updated without unrelated churn

- GIVEN the 12 golden fixtures that reference Chain Strategy are regenerated
- WHEN the regenerated fixtures are compared with the previous versions
- THEN the diff is limited to the binding wording and the new Cursor chain strategy section
- AND no unrelated content is modified in the golden fixtures

#### Scenario: inject_test.go binding assertion passes

- GIVEN `internal/components/sdd/inject_test.go` is updated with a binding assertion
- WHEN the test executes
- THEN the assertion verifies that the `chained-pr` skill binding is present in generated prompts for `sdd-tasks` and `sdd-apply` when chain strategy is active

---

### Requirement: Claude-Native Delegation Semantics

Claude-targeted guidance MUST describe delegation behavior accurately for Claude Code and MUST NOT imply OpenCode plugin-backed persisted background delegation.

#### Scenario: Guidance avoids persisted-plugin delegation claims

- GIVEN Claude orchestrator guidance content is reviewed for async/delegation wording
- WHEN delegation behavior is described
- THEN wording does not claim OpenCode plugin persistence or background task storage

#### Scenario: Guidance states Claude-accurate delegation behavior

- GIVEN a reader follows the Claude guidance
- WHEN they interpret delegation expectations
- THEN wording reflects Claude Code semantics without OpenCode-specific guarantees

---

### Requirement: Claude Golden and Static Validation Coverage

The system MUST keep Claude golden outputs and static text assertions synchronized with the current Claude orchestrator guidance asset.

The system MUST also keep direct static assertions and impacted golden fixtures synchronized for non-Claude orchestrator assets affected by this capability.

#### Scenario: Claude SDD golden reflects updated chain strategy and forwarding guidance

- GIVEN Claude standalone and combined golden fixtures are generated
- WHEN compared with expected outputs
- THEN both include the updated chain strategy and propagation guidance

#### Scenario: Static assertions align with revised Claude delegation wording

- GIVEN static tests assert expected Claude guidance text fragments
- WHEN tests execute after wording changes
- THEN assertions reflect Claude-native delegation wording
- AND no assertion requires OpenCode plugin-backed persistence phrasing

#### Scenario: Non-Claude static assertions validate strategy and platform wording

- GIVEN direct static tests assert expected non-Claude orchestrator text fragments
- WHEN tests execute after non-Claude asset updates
- THEN assertions verify canonical chain strategy names where delivery planning is present
- AND assertions verify platform-native wording constraints for solo-inline hosts

#### Scenario: Impacted non-Claude golden fixtures reflect current orchestrator guidance

- GIVEN golden fixtures for directly affected non-Claude orchestrator outputs are generated
- WHEN compared with expected outputs
- THEN fixtures include required strategy guidance and propagation where relevant
- AND fixtures preserve accurate platform-native wording without unrelated churn

---

### Requirement: Orchestrator-Owned Discovery and Confirmed Handoff

Orchestrators MUST own discovery after exploration/research. Interactive mode MAY question; automatic mode MUST use confirmed facts or block. They MUST invoke `sdd-propose` only with selected research `done`, decisions `confirmed`, valid references/readiness; proposal executors MUST NOT interview.

#### Scenario: Confirmed proposal handoff

- GIVEN research is done/unselected, state matches, and decisions are confirmed
- WHEN the orchestrator reaches proposal
- THEN proposal starts without questions

#### Scenario: Automatic unresolved choices

- GIVEN automatic mode has unresolved product choices
- WHEN the pre-proposal gate runs
- THEN it emits one lossless grouped prompt, persists state, and blocks proposal

### Requirement: Optional-Lane and Status-v1 Compatibility

Research MUST be optional before selection; no-request changes MUST skip only that branch. Native status-v1 shape, tokens, and routing MUST stay unchanged; orchestrators gate `propose` with readiness.

#### Scenario: Legacy change without research

- GIVEN a change has no research request
- WHEN status-v1 returns its existing `propose` recommendation
- THEN the route is preserved and research is not required

#### Scenario: Status-v1 remains frozen

- GIVEN a consumer asks for a research route
- WHEN status-v1 is validated
- THEN no research token is emitted/accepted; the orchestrator gate handles it

### Requirement: Bounded Review Contract Parity Across Catalog Agents

All 12 source orchestrator templates MUST render equivalent optional research lifecycle guidance; mappings MAY be shared. Every AgentID returned by `catalog.AllAgents()` MUST render the canonical bounded review execution contract, and all 16 supported IDs MUST be covered, including OpenCode/Kilocode sharing and Pi/VS Code Copilot/OpenClaw/Trae using the generic runtime family.
(Previously: bounded-review parity covered all 16 IDs but not research lifecycle.)

#### Scenario: Catalog-derived parity test renders all agents

- GIVEN the current catalog contains 16 supported AgentIDs
- WHEN each ID is passed through the real orchestrator asset renderer
- THEN every rendered variant contains explicit `review/start(target)`, one transaction-wide refuter batch, non-iterative ordinary scoped validation, Judgment Day-only re-judgment, exact persistence references, and user-owned runtime selection
- AND no variant contains the old standard=1/full-4R=3 refuter fan-out
- AND optional research guidance does not change these guarantees

### Requirement: Shared Contract Expansion

Reusable bounded-review algorithms MUST live in `skills/_shared/review-ledger-contract.md`. Orchestrator and native review-agent generation MUST expand that canonical source instead of maintaining divergent hand-copied algorithms.

#### Scenario: Dedicated reviewer is installed

- GIVEN an adapter supports native review sub-agents
- WHEN SDD injection writes the reviewer file
- THEN the installed prompt contains the canonical bounded contract
- AND the role-specific source contributes only lens and tool-boundary instructions

### Requirement: Read-Only Reviewer and Refuter Roles

Every dedicated reviewer, Judgment Day judge, and refuter MUST be structurally read-only for its adapter. OpenCode overlays MUST explicitly disable write, edit, bash, and task for these roles; Claude/Kiro/Kimi/Cursor assets MUST use their equivalent read-only capability boundary.

#### Scenario: Reviewer finishes

- GIVEN a native review lens receives an immutable target
- WHEN its one sweep completes
- THEN it returns neutral structured claims and terminates
- AND it cannot modify files or launch another role
### Requirement: Guía y Doctrina del Modo ISO, `Cubre:`, la Matriz y el Bug sin Delta, con la Excepción ISO, en Paridad entre Runtimes (REQ-29.17)

Las skills `sdd-*` (sobre todo `sdd-verify`, con su referencia del formato de informe, `sdd-tasks` y `sdd-archive`), la doctrina compartida (`_shared/persistence-contract.md`, `_shared/sdd-status-contract.md` y `_shared/sdd-orchestrator-sections.md`) y los orquestadores proyectados a cada runtime DEBEN describir las opciones de este cambio: el modo ISO y la proyección de `sdd.iso` en el status (REQ-29.2), `kickoff seal` copiando `iso` del proyecto en `kickoff.yaml` (REQ-29.13), la línea `Cubre:` de las tareas, la matriz de verify con sus filas automáticas y manuales y las formas de referencia de prueba admitidas (`ruta/fichero:símbolo` y `TestX`; REQ-29.6 a REQ-29.8), el perfil de bug con sus tres rutas (REQ-29.15): bug con requisito existente (sin delta, con `Cubre:`), bug sin impacto en la spec (sin delta, sin `Cubre:`, con `## Impacto en la spec` y la línea literal `Sin impacto en la spec` más `Motivo:`; en ISO el motivo es la evidencia y la matriz vacía es válida, REQ-29.19) y bug con delta como cambio normal; el formato de `## Informe de bug` (`Observado:` y `Esperado:` obligatorias, `Referencia:` opcional); y la derivación de la traza (REQ-29.9). La guía de bug DEBE reproducir esos textos exactamente, porque el carril ODD usa la misma sección y la misma línea. Las skills describen los formatos; la validación reside en el motor. La doctrina `persistence-contract.md` DEBE conservar íntegra la regla «Missing or failed verification never gates archive» para los cambios no ISO y DEBE declarar la excepción del modo ISO (REQ-29.11) sin reescribir la regla general: en ISO, la matriz de verify válida y con éxito bloquea el archivado, y los hallazgos de revisión siguen siendo informativos. La skill `sdd-verify` DEBE indicar que, en un cambio ISO, la matriz es una sección adicional del informe sin líneas `verdict:`. La skill `sdd-archive` NO DEBE cambiar su descripción del flujo de archivado: la rama `-archive` y el PR documental aparte siguen siendo el único modo. Las skills y la guía de kickoff DEBEN dejar de indicar que se cree a mano la carpeta del cambio antes de `kickoff seal` (REQ-29.12). Toda esa guía DEBE salir de una fuente única, con pines de texto que verifiquen la paridad entre runtimes, y las copias versionadas y los goldens DEBEN regenerarse con la herramienta de actualización, nunca a mano.

#### Scenario: La doctrina conserva la regla general y declara la excepción ISO

- **DADO** el texto de `_shared/persistence-contract.md` proyectado
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** contiene la regla «Missing or failed verification never gates archive» para cambios no ISO
- **Y** declara que en un cambio ISO la matriz de verify válida y con éxito bloquea el archivado y que los hallazgos de revisión no lo bloquean

#### Scenario: Paridad de la guía en todos los runtimes

- **DADO** cada runtime soportado y su orquestador proyectado
- **CUANDO** se renderizan los activos mediante el renderizador real
- **ENTONCES** todas las variantes contienen la misma guía de `Cubre:`, matriz, perfil de bug (tres rutas) y modo ISO
- **Y** la excepción ISO figura en todas

#### Scenario: La guía de bug describe las tres rutas y los textos literales

- **DADO** el texto de la guía de bug de las skills y de la doctrina compartida
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** describe las rutas con requisito existente, sin impacto en la spec y con delta
- **Y** contiene literalmente `## Informe de bug`, `Observado:`, `Esperado:`, `Referencia:`, `## Impacto en la spec`, `Sin impacto en la spec` y `Motivo:`
- **Y** indica que en ISO el motivo es la evidencia del bug sin impacto y que la matriz vacía es válida

#### Scenario: La skill de verify describe la matriz sin líneas `verdict:`

- **DADO** el texto de `sdd-verify` y de su referencia del formato de informe
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** describen la matriz como una sección adicional que convive con «Cumplimiento técnico»
- **Y** prohíben escribir en ella líneas que empiecen por `verdict:`
- **Y** nombran las dos formas de referencia de prueba, el comando y el código de salida de las filas automáticas y la justificación de las manuales

#### Scenario: La skill de tareas describe `Cubre:`

- **DADO** el texto de `sdd-tasks`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** describe la línea `Cubre: REQ-x[, REQ-y ...]` y que en un cambio ISO cada requisito de la delta debe figurar en alguna

#### Scenario: La skill de archive conserva su flujo

- **DADO** el texto de `sdd-archive/SKILL.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** sigue indicando la rama `-archive` y el PR documental aparte
- **Y** no describe ningún archivado en la rama del cambio

#### Scenario: La guía de kickoff no pide crear la carpeta

- **DADO** los textos de las skills y plantillas que describen el sellado
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no indican crear a mano la carpeta del cambio antes de `axiom sdd kickoff seal`

#### Scenario: El contrato del status documenta la proyección ISO

- **DADO** `_shared/sdd-status-contract.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** documenta la proyección de `sdd.iso`, los motivos tipados de ISO (REQ-29.16) y que un cambio sin instantánea es no ISO

#### Scenario: Los goldens y las copias versionadas están en paridad

- **DADO** la guía actualizada
- **CUANDO** se regeneran los goldens y las copias versionadas con la herramienta de actualización
- **ENTONCES** las pruebas de paridad pasan sin ediciones manuales de los goldens
### Requirement: Skills y Convención Compartida sin Baseline Ajena (REQ-30.28)

Las skills `sdd-*` y los ficheros compartidos (`_shared/`) distribuidos por Axiom NO DEBEN contener ramas, prosa ni escenarios que describan `baseline:`, `baseline.documents`, los modos de integración `compose`, `directed` o `register`, `axiom sdd integrate` ni `axiom sdd archive-compose --document` o `--in-place`. La skill `sdd-spec` DEBE leer la spec viva de `openspec/specs/{dominio}/spec.md` como única fuente del comportamiento vigente. La convención compartida del proyecto (`_shared/project-baseline-convention.md`) DEBE quedar reducida al contexto técnico (declaración, lectura, delta de contexto e integración con `axiom archive context`) y a la regla de «Sin impacto». La skill `sdd-archive` DEBE prescribir `axiom archive context check|prepare|verify|show` para el contexto mantenido. La prosa que se proyecta a los runtimes se regenera con la herramienta de goldens y permanece en paridad entre runtimes.

#### Scenario: Ninguna skill describe la baseline ajena
- **DADO** los textos de las skills `sdd-*` y de `_shared/` distribuidos
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene `baseline.documents`, `integration: directed`, `axiom sdd integrate` ni `--in-place`

#### Scenario: `sdd-spec` lee `openspec/specs/`
- **DADO** el texto de la skill `sdd-spec`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica leer `openspec/specs/{dominio}/spec.md` para conocer el comportamiento vigente
- **Y** no condiciona la lectura a ninguna proyección `baseline`

#### Scenario: La convención compartida solo trata el contexto técnico
- **DADO** el texto de `_shared/project-baseline-convention.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** describe el contexto técnico, el delta de contexto y `axiom archive context`
- **Y** no describe documentos de baseline ni perfiles de requisito

#### Scenario: `sdd-archive` prescribe el verbo de contexto
- **DADO** el texto de la skill `sdd-archive`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** prescribe `axiom archive context` para la integración del contexto mantenido

#### Scenario: Paridad entre runtimes
- **DADO** dos runtimes soportados
- **CUANDO** se renderizan las skills y las instrucciones
- **ENTONCES** el contenido normativo es idéntico entre ambos y los goldens coinciden con la regeneración

---

### Requirement: Skill o Referencia de Adopción de OpenSpec (REQ-30.31)

Axiom DEBE distribuir una skill o una referencia compartida de adopción que describa, en un texto idéntico para todos los runtimes, el flujo de migración a OpenSpec: ejecutar `axiom archive adopt scan --change <cambio>`; redactar la delta de adopción con solo `## ADDED Requirements`, un identificador `(REQ-n.m)` de una serie nueva, al menos un escenario y una línea `Origen:` por requisito; listar con motivo las unidades descartadas en `Descartado:`; ejecutar `axiom archive adopt check --change <cambio>` hasta que no reporte fallos; y someter la compuerta `spec` a la aprobación humana. El texto NO DEBE afirmar que Go juzga la fidelidad semántica de la migración.

#### Scenario: La guía de adopción describe el flujo completo
- **DADO** el texto de la skill o referencia de adopción
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** cita `adopt scan`, `adopt check`, `Origen:`, `Descartado:`, la serie `REQ-n.m` y la compuerta `spec`

#### Scenario: La guía no atribuye a Go la fidelidad semántica
- **DADO** el mismo texto
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que la fidelidad la aprueba un humano en la compuerta `spec`

#### Scenario: Paridad entre runtimes
- **DADO** dos runtimes soportados
- **CUANDO** se renderiza la guía de adopción
- **ENTONCES** el contenido normativo es idéntico entre ambos
### Requirement: La Entrega es `single-pr` por Defecto sin Preguntar, `chained` Solo a Petición y se Leen los Valores Heredados mediante Alias (REQ-32.3)

La doctrina de los orquestadores SDD de todos los runtimes (plantillas, copias en línea, reservas que compone el instalador y la convención compartida `sdd-phase-common.md`) DEBE declarar que la estrategia de entrega por defecto es `single-pr` (una PR con un commit por unidad de trabajo), que NO se pregunta, y que `chained` (con `stacked-to-main` o `feature-branch-chain`) solo se activa cuando la persona lo pide durante la sesión; si lo pide sin indicar la estrategia de cadena, el orquestador formula una única pregunta por la estrategia de cadena. El dominio canónico es `single-pr | chained`. Al **leer** un valor de entrega ya existente (un `tasks.md`, un prompt o una caché heredados), el orquestador y las fases DEBEN aplicar estos alias de lectura: `ask-on-risk` y `exception-ok` → `single-pr`; `auto-chain` → `chained`. Cualquier otro valor DEBE seguir siendo inválido y reportarse sin mapearlo. NO se escribe ningún valor nuevo con vocabulario heredado, y no se reescribe ningún artefacto existente.

`chained` se mantiene solo en la caché de la sesión del orquestador (como las demás preferencias de sesión, REQ-28.2): NO se escribe en `kickoff.yaml`, en `gates.yaml` ni como campo de `tasks.md`; en una sesión nueva la entrega vuelve a `single-pr`. El bloque de pre-vuelo expresa el valor por defecto de la sesión: un `delivery_strategy: chained` reenviado por el orquestador con su `chain_strategy` prevalece sobre él, y `sdd-apply` solo lo usa para elegir la rama destino y nunca se detiene por ello.

Ninguna doctrina ni skill DEBE contener un presupuesto numérico de revisión ni una parada por carga de revisión. En concreto: la sección `Review Workload Guard` y su bloque de decisión no existen en ningún runtime; `sdd-tasks` conserva solo una previsión cualitativa (tamaño estimado y unidades de trabajo) sin los campos `400-line budget risk`, `Decision needed before apply` ni `Chained PRs recommended`; y `sdd-apply` no contiene el paso de decisión de carga de trabajo ni la regla `workload-decision-required`. Si `delivery.split_hint_lines` está fijada, se aplica el aviso único no bloqueante de REQ-32.2 de `workspace-topology`. Quedan fuera de este requisito, por no ser umbrales de entrega, los literales de migración de prompts antiguos de `internal/components/sdd/inject.go`, el umbral de barrido de `jd-judge-a` y `jd-judge-b` de judgment-day, las lentes `review-*` y `rdd-defect-workflow` de RDD y `internal/reviewtransaction/risk.go`.

#### Scenario: La entrega por defecto es `single-pr` y no se pregunta

- **DADO** la doctrina SDD proyectada a cada runtime y la convención compartida
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** declaran `single-pr` como entrega por defecto y que no se pregunta
- **Y** declaran `chained` solo a petición de la persona, con `stacked-to-main` y `feature-branch-chain`
- **Y** el dominio canónico declarado es `single-pr | chained`

#### Scenario: Una petición de encadenar sin estrategia provoca una única pregunta

- **DADO** una persona que pide encadenar las PRs sin indicar la estrategia de cadena
- **CUANDO** el orquestador aplica su doctrina
- **ENTONCES** formula una única pregunta por `stacked-to-main` o `feature-branch-chain`
- **Y** cachea `delivery_strategy: chained` y la `chain_strategy` elegida solo en la sesión

#### Scenario: Los valores heredados se leen mediante alias

- **DADO** un `tasks.md` o un prompt con `Delivery strategy: ask-on-risk` (y otros con `exception-ok` y con `auto-chain`)
- **CUANDO** el orquestador o una fase leen el valor
- **ENTONCES** `ask-on-risk` y `exception-ok` se interpretan como `single-pr` y `auto-chain` como `chained`
- **Y** no se reescribe el artefacto existente

#### Scenario: Un valor desconocido se sigue informando sin mapearlo

- **DADO** un valor de entrega que no es canónico ni alias (por ejemplo `bulk`)
- **CUANDO** se valida
- **ENTONCES** se reporta como inválido
- **Y** no se mapea a ningún valor canónico

#### Scenario: `chained` no se persiste

- **DADO** una sesión en la que la persona pidió `chained`
- **CUANDO** se sella el kickoff o se registra una puerta
- **ENTONCES** `kickoff.yaml` y `gates.yaml` no contienen la estrategia de entrega
- **Y** en una sesión nueva la entrega vuelve a `single-pr`

#### Scenario: Un `chained` reenviado prevalece sobre el valor del bloque de pre-vuelo

- **DADO** un bloque de pre-vuelo con `Delivery strategy: single-pr` y un prompt de fase con `delivery_strategy: chained` y su `chain_strategy`
- **CUANDO** la fase resuelve la entrega
- **ENTONCES** prevalece `chained` para elegir la rama destino
- **Y** la fase no se detiene por ello

#### Scenario: No existe ningún Review Workload Guard ni presupuesto numérico

- **DADO** la doctrina proyectada a los doce runtimes, los goldens regenerados y las skills `sdd-tasks`, `sdd-apply`, `chained-pr` y `work-unit-commits`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no contienen la sección `Review Workload Guard` ni ningún número de líneas como umbral de entrega
- **Y** `sdd-tasks` no contiene `400-line budget risk`, `Decision needed before apply` ni `Chained PRs recommended`
- **Y** `sdd-apply` no contiene la regla `workload-decision-required`

#### Scenario: `sdd-apply` no se detiene por carga de revisión

- **DADO** un cambio con una previsión de tamaño grande, sin cadena pedida ni `size:exception`
- **CUANDO** se ejecuta `sdd-apply`
- **ENTONCES** continúa con la implementación sin emitir ninguna parada por carga de revisión

#### Scenario: `sdd-tasks` emite una previsión cualitativa

- **DADO** la skill `sdd-tasks`
- **CUANDO** se inspecciona su previsión de entrega
- **ENTONCES** pide el tamaño estimado y las unidades de trabajo
- **Y** no incluye campos de riesgo ni de decisión ni ningún umbral numérico

#### Scenario: Con `delivery.split_hint_lines` fijada la doctrina indica el aviso único

- **DADO** la doctrina del orquestador SDD
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que, si `axiom.yaml` declara `delivery.split_hint_lines` y la previsión la supera, se avisa una vez de que el cambio es grande y se continúa sin bloquear
- **Y** no contiene ningún valor por defecto de esa clave
