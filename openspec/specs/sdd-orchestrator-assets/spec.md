# SDD Orchestrator Assets Specification

## Purpose

Defines required behavior for agent-specific SDD orchestrator guidance assets so Claude and OpenCode guidance remain semantically aligned where intended, while preserving platform-accurate delegation wording.

---

## Requirements

### Requirement: Chain Strategy Guidance Parity (RF-ORQ-01)

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

### Requirement: Chain Strategy Propagation to Downstream Phases (RF-ORQ-02)

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

### Requirement: Non-Claude Chain Strategy Guidance Parity (RF-ORQ-03)

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

### Requirement: Non-Claude Strategy Propagation to Downstream Phases (RF-ORQ-04)

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

### Requirement: Platform-Native Solo Inline Semantics Preservation (RF-ORQ-05)

For non-Claude solo-inline and platform-native orchestrator assets, the system MUST preserve platform-accurate inline execution wording and MUST NOT add inaccurate subagent or OpenCode persistence claims.
(Previously: the first scenario also covered the Windsurf asset.)

#### Scenario: Antigravity keeps inline-accurate wording

- GIVEN the Antigravity orchestrator asset is reviewed
- WHEN delegation and execution wording is validated
- THEN wording remains accurate to inline/platform-native execution
- AND wording does not claim persisted OpenCode plugin-backed background delegation

#### Scenario: Other non-Claude platform-native assets avoid inaccurate persistence claims

- GIVEN non-Claude platform-native orchestrator assets are reviewed
- WHEN async/delegation wording is validated
- THEN wording stays platform-accurate
- AND wording does not introduce inaccurate subagent or OpenCode persistence guarantees

### Requirement: chained-pr Skill Binding in Chain Strategy Guidance (RF-ORQ-06)

Los activos del orquestador SDD DEBEN enlazar la skill `chained-pr` en su guía de estrategia de cadena. Cuando la persona pida PRs encadenadas o apiladas, el orquestador DEBE resolver la skill `chained-pr` por su nombre de registro (no por una ruta del sistema de ficheros fija) mediante su Sub-Agent Launch Pattern, y DEBE inyectar la ruta resuelta en los prompts de las fases `sdd-tasks` y `sdd-apply` bajo `## Skills to load before work`. Esas fases DEBEN leer y seguir la skill ANTES de planificar o crear cualquier PR. Con la entrega `single-pr` por defecto la skill NO se resuelve ni se inyecta, y la propia skill `chained-pr` se activa solo cuando la persona pide encadenar, apilar o rebanar, o con `delivery_strategy: chained`, sin ningún umbral numérico.

El enlace DEBE referenciar la skill por el nombre de registro `chained-pr` (frontmatter `chained-pr`). No se permite ninguna ruta fija a un fichero `SKILL.md` como referencia del enlace.
(Previously: decía «Los 12 activos» y el escenario enumeraba también las plantillas de hermes, kimi, kiro y windsurf, retiradas por C-08-d.)

#### Scenario: Orchestrator resolves chained-pr skill by registry name on chained delivery

- **DADO** un activo de orquestador SDD con guía de estrategia de cadena
- **CUANDO** la persona pide entrega encadenada o apilada
- **ENTONCES** el orquestador resuelve la skill `chained-pr` por su nombre de registro mediante el Sub-Agent Launch Pattern
- **Y** la ruta resuelta se inyecta en el prompt de `sdd-tasks` bajo `## Skills to load before work`
- **Y** la ruta resuelta se inyecta en el prompt de `sdd-apply` bajo `## Skills to load before work`
- **Y** no se usa ninguna ruta fija como referencia del enlace

#### Scenario: Binding is present in all orchestrator templates

- **DADO** las plantillas del orquestador SDD (antigravity, claude, codex, cursor, gemini, generic, opencode, qwen)
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

### Requirement: Inline Chain Strategy Summary Retained (RF-ORQ-07)

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

### Requirement: Cursor Chain Strategy Section Parity (RF-ORQ-08)

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

### Requirement: Platform-Accurate Binding Wording for Solo-Inline and Platform-Native Hosts (RF-ORQ-09)

For solo-inline and platform-native orchestrator hosts (antigravity), the skill-binding instruction MUST phrase "inject under `## Skills to load before work`" in a way that maps to each host's actual phase-context mechanism. The binding MUST NOT reintroduce OpenCode-style persisted-delegation claims that the existing spec forbids for these hosts.
(Previously: the hosts were windsurf, antigravity, kimi and kiro, with a second scenario for kimi and kiro.)

#### Scenario: Solo-inline binding wording is platform-accurate

- GIVEN a solo-inline orchestrator asset (antigravity) is reviewed
- WHEN the skill binding instruction is read
- THEN the binding references the `chained-pr` skill by registry name
- AND the wording maps skill injection to the host's inline or platform-native phase-context mechanism
- AND the wording does not claim persisted OpenCode plugin-backed background delegation

### Requirement: Golden Fixture and Static Assertion Coverage for the Binding (RF-ORQ-10)

Static-validation tests and golden fixtures that cover chain strategy content MUST be updated to assert the presence of the `chained-pr` skill binding. Golden churn MUST be limited to the intended binding wording and the new Cursor chain strategy section; no unrelated content changes are permitted.
(Previously: the scenarios counted 12 templates and 12 golden fixtures.)

#### Scenario: New static assertions verify binding presence in all templates

- GIVEN static-validation tests run after the change is applied
- WHEN binding assertions execute
- THEN each orchestrator template has a binding assertion that passes
- AND the assertions verify the registry-name reference, not a hardcoded path

#### Scenario: Cursor is added to chain-strategy parity static assertions

- GIVEN the existing chain-strategy parity assertions are updated to include cursor
- WHEN static tests execute
- THEN a chain-strategy parity assertion for the cursor template passes
- AND the assertion verifies that canonical strategy names are present

#### Scenario: Golden fixtures updated without unrelated churn

- GIVEN the golden fixtures that reference Chain Strategy are regenerated
- WHEN the regenerated fixtures are compared with the previous versions
- THEN the diff is limited to the binding wording and the new Cursor chain strategy section
- AND no unrelated content is modified in the golden fixtures

#### Scenario: inject_test.go binding assertion passes

- GIVEN `internal/components/sdd/inject_test.go` is updated with a binding assertion
- WHEN the test executes
- THEN the assertion verifies that the `chained-pr` skill binding is present in generated prompts for `sdd-tasks` and `sdd-apply` when chain strategy is active

### Requirement: Claude-Native Delegation Semantics (RF-ORQ-11)

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

### Requirement: Claude Golden and Static Validation Coverage (RF-ORQ-12)

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

### Requirement: Orchestrator-Owned Discovery and Confirmed Handoff (RF-ORQ-13)

Orchestrators MUST own discovery after exploration/research. Interactive mode MAY question; automatic mode MUST use confirmed facts or block. They MUST invoke `sdd-propose` only with selected research `done`, decisions `confirmed`, valid references/readiness; proposal executors MUST NOT interview.

#### Scenario: Confirmed proposal handoff

- GIVEN research is done/unselected, state matches, and decisions are confirmed
- WHEN the orchestrator reaches proposal
- THEN proposal starts without questions

#### Scenario: Automatic unresolved choices

- GIVEN automatic mode has unresolved product choices
- WHEN the pre-proposal gate runs
- THEN it emits one lossless grouped prompt, persists state, and blocks proposal

### Requirement: Optional-Lane and Status-v1 Compatibility (RF-ORQ-14)

Research MUST be optional before selection; no-request changes MUST skip only that branch. Native status-v1 shape, tokens, and routing MUST stay unchanged; orchestrators gate `propose` with readiness.

#### Scenario: Legacy change without research

- GIVEN a change has no research request
- WHEN status-v1 returns its existing `propose` recommendation
- THEN the route is preserved and research is not required

#### Scenario: Status-v1 remains frozen

- GIVEN a consumer asks for a research route
- WHEN status-v1 is validated
- THEN no research token is emitted/accepted; the orchestrator gate handles it

### Requirement: Optional Research Lifecycle Parity Across Catalog Agents (RF-ORQ-15)

All source orchestrator templates MUST render equivalent optional research lifecycle guidance; mappings MAY be shared. Every AgentID returned by `catalog.AllAgents()` MUST render that guidance, including Pi/VS Code Copilot/Trae using the generic runtime family.
(Previously: «All 12 source orchestrator templates» and «all 16 supported IDs», including OpenCode/Kilocode sharing and OpenClaw in the generic family; the requirement also demanded that every rendered variant carry the canonical bounded review execution contract, which belonged to RDD and is retired.)

#### Scenario: Catalog-derived parity test renders all agents

- GIVEN the current catalog's supported AgentIDs
- WHEN each ID is passed through the real orchestrator asset renderer
- THEN every rendered variant contains the optional research lifecycle guidance
- AND no rendered variant contains a `Review Execution Contract` section, its marker, `review/start(target)` or a refuter batch
- AND optional research guidance is rendered identically across the equivalent runtime families

### Requirement: Read-Only Judgment Day Judges (RF-ORQ-16)

Every Judgment Day judge (`jd-judge-a`, `jd-judge-b`) MUST be structurally read-only for its adapter. OpenCode overlays MUST explicitly disable write, edit, bash, and task for these roles; Claude/Cursor assets MUST use their equivalent read-only capability boundary.
(Previously: Kiro and Kimi assets were also listed; the requirement covered every dedicated reviewer, Judgment Day judge, and refuter, and reviewers and refuters belonged to RDD and are retired.)

#### Scenario: Judge finishes

- GIVEN a Judgment Day judge receives its target
- WHEN its sweep completes
- THEN it returns structured findings and terminates
- AND it cannot modify files or launch another role

### Requirement: Guía y Doctrina del Modo ISO, `Cubre:`, la Matriz y el Bug sin Delta, con la Excepción ISO, en Paridad entre Runtimes (RF-ORQ-17)

Las skills `sdd-*` (sobre todo `sdd-verify`, con su referencia del formato de informe, `sdd-tasks` y `sdd-archive`), la doctrina compartida (`_shared/persistence-contract.md`, `_shared/sdd-status-contract.md` y `_shared/sdd-orchestrator-sections.md`) y los orquestadores proyectados a cada runtime DEBEN describir las opciones de este cambio: el modo ISO y la proyección de `sdd.iso` en el status (RF-ISO-02), `kickoff seal` copiando `iso` del proyecto en `kickoff.yaml` (RF-KOF-12), la línea `Cubre:` de las tareas, la matriz de verify con sus filas automáticas y manuales y las formas de referencia de prueba admitidas (`ruta/fichero:símbolo` y `TestX`; RF-ISO-06 a RF-ISO-08), el perfil de bug con sus tres rutas (RF-CLI-10): bug con requisito existente (sin delta, con `Cubre:`), bug sin impacto en la spec (sin delta, sin `Cubre:`, con `## Impacto en la spec` y la línea literal `Sin impacto en la spec` más `Motivo:`; en ISO el motivo es la evidencia y la matriz vacía es válida, RF-ISO-10) y bug con delta como cambio normal; el formato de `## Informe de bug` (`Observado:` y `Esperado:` obligatorias, `Referencia:` opcional); y la derivación de la traza (RF-ISO-09). La guía de bug DEBE reproducir esos textos exactamente, porque el carril ODD usa la misma sección y la misma línea. Las skills describen los formatos; la validación reside en el motor. La doctrina `persistence-contract.md` DEBE conservar íntegra la regla «Missing or failed verification never gates archive» para los cambios no ISO y DEBE declarar la excepción del modo ISO (RF-ARC-10) sin reescribir la regla general: en ISO, la matriz de verify válida y con éxito bloquea el archivado, y los hallazgos de revisión siguen siendo informativos. La skill `sdd-verify` DEBE indicar que, en un cambio ISO, la matriz es una sección adicional del informe sin líneas `verdict:`. La skill `sdd-archive` NO DEBE cambiar su descripción del flujo de archivado: la rama `-archive` y el PR documental aparte siguen siendo el único modo. Las skills y la guía de kickoff DEBEN dejar de indicar que se cree a mano la carpeta del cambio antes de `kickoff seal` (RF-KOF-11). Toda esa guía DEBE salir de una fuente única, con pines de texto que verifiquen la paridad entre runtimes, y las copias versionadas y los goldens DEBEN regenerarse con la herramienta de actualización, nunca a mano.

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
- **ENTONCES** documenta la proyección de `sdd.iso`, los motivos tipados de ISO (RF-CLI-11) y que un cambio sin instantánea es no ISO

#### Scenario: Los goldens y las copias versionadas están en paridad

- **DADO** la guía actualizada
- **CUANDO** se regeneran los goldens y las copias versionadas con la herramienta de actualización
- **ENTONCES** las pruebas de paridad pasan sin ediciones manuales de los goldens
### Requirement: Skills y Convención Compartida sin Baseline Ajena (RF-ORQ-18)

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

### Requirement: Skill o Referencia de Adopción de OpenSpec (RF-ORQ-19)

Axiom DEBE distribuir una skill o una referencia compartida de adopción que describa, en un texto idéntico para todos los runtimes, el flujo de migración a OpenSpec: ejecutar `axiom archive adopt scan --change <cambio>`; revisar el patrón de identificadores propuesto y sus conflictos y declararlo la persona en `sdd.id_pattern` de `axiom.yaml` (el escáner no lo escribe) antes de `adopt check` y de archivar; redactar la delta de adopción con solo `## ADDED Requirements`, al menos un escenario y una línea `Origen:` por requisito, y en cada título el identificador del cliente conservado (que debe coincidir con el `<ID original>` de su `Origen:`) o, cuando la unidad no trae identificador, un provisional `prov:<series>:<ordinal>` cuyo número se asigna al archivar; listar con motivo las unidades descartadas en `Descartado:`; normalizar a mano las mezclas de identificadores que el escáner informa como conflicto; ejecutar `axiom archive adopt check --change <cambio>` hasta que no reporte fallos (incluida la regla `pattern_undeclared`); someter la compuerta `spec` a la aprobación humana; y archivar con `axiom sdd archive-integrate`. El texto NO DEBE afirmar que Go juzga la fidelidad semántica de la migración, NO DEBE exigir «un `(REQ-n.m)` de una serie nueva», NO DEBE contener la excepción «hasta C-03-c» y NO DEBE ordenar `archive-compose --canonical --delta` para la adopción.
(Previously: la delta se redactaba con «un identificador `(REQ-n.m)` de una serie nueva» y el texto no mencionaba el patrón del cliente, los provisionales ni `archive-integrate`.)

#### Scenario: La guía de adopción describe el flujo completo
- **DADO** el texto de la skill o referencia de adopción
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** cita `adopt scan`, `adopt check`, `Origen:`, `Descartado:`, `sdd.id_pattern`, `pattern_undeclared`, `prov:<series>:<ordinal>`, `archive-integrate` y la compuerta `spec`

#### Scenario: La guía conserva los identificadores del cliente
- **DADO** el mismo texto
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que el identificador del cliente se conserva en el título y coincide con el `<ID original>` de `Origen:`
- **Y** indica que la declaración del patrón la hace la persona y que `scan` no escribe `axiom.yaml`
- **Y** indica que las mezclas de esquemas se normalizan a mano

#### Scenario: La guía ya no fija una serie nueva ni la excepción temporal
- **DADO** el mismo texto, y el resto de skills que mencionen la adopción
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene «una serie nueva» como exigencia del título ni «hasta C-03-c»

#### Scenario: La guía no atribuye a Go la fidelidad semántica
- **DADO** el mismo texto
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que la fidelidad la aprueba un humano en la compuerta `spec`

#### Scenario: Paridad entre runtimes
- **DADO** dos runtimes soportados
- **CUANDO** se renderiza la guía de adopción
- **ENTONCES** el contenido normativo es idéntico entre ambos
### Requirement: La Entrega es `single-pr` por Defecto sin Preguntar, `chained` Solo a Petición y se Leen los Valores Heredados mediante Alias (RF-ORQ-20)

La doctrina de los orquestadores SDD de todos los runtimes (plantillas, copias en línea, reservas que compone el instalador y la convención compartida `sdd-phase-common.md`) DEBE declarar que la estrategia de entrega por defecto es `single-pr` (una PR con un commit por unidad de trabajo), que NO se pregunta, y que `chained` (con `stacked-to-main` o `feature-branch-chain`) solo se activa cuando la persona lo pide durante la sesión; si lo pide sin indicar la estrategia de cadena, el orquestador formula una única pregunta por la estrategia de cadena. El dominio canónico es `single-pr | chained`. Al **leer** un valor de entrega ya existente (un `tasks.md`, un prompt o una caché heredados), el orquestador y las fases DEBEN aplicar estos alias de lectura: `ask-on-risk` y `exception-ok` → `single-pr`; `auto-chain` → `chained`. Cualquier otro valor DEBE seguir siendo inválido y reportarse sin mapearlo. NO se escribe ningún valor nuevo con vocabulario heredado, y no se reescribe ningún artefacto existente.

`chained` se mantiene solo en la caché de la sesión del orquestador (como las demás preferencias de sesión, RF-REV-02): NO se escribe en `kickoff.yaml`, en `gates.yaml` ni como campo de `tasks.md`; en una sesión nueva la entrega vuelve a `single-pr`. El bloque de pre-vuelo expresa el valor por defecto de la sesión: un `delivery_strategy: chained` reenviado por el orquestador con su `chain_strategy` prevalece sobre él, y `sdd-apply` solo lo usa para elegir la rama destino y nunca se detiene por ello.

Ninguna doctrina ni skill DEBE contener un presupuesto numérico de revisión ni una parada por carga de revisión. En concreto: la sección `Review Workload Guard` y su bloque de decisión no existen en ningún runtime; `sdd-tasks` conserva solo una previsión cualitativa (tamaño estimado y unidades de trabajo) sin los campos `400-line budget risk`, `Decision needed before apply` ni `Chained PRs recommended`; y `sdd-apply` no contiene el paso de decisión de carga de trabajo ni la regla `workload-decision-required`. Si `delivery.split_hint_lines` está fijada, se aplica el aviso único no bloqueante de RF-TOP-15 de `workspace-topology`. Quedan fuera de este requisito, por no ser umbrales de entrega, los literales de migración de prompts antiguos de `internal/components/sdd/inject.go`, el umbral de barrido de `jd-judge-a` y `jd-judge-b` de judgment-day.
(Previously: la lista de exclusiones citaba también las lentes `review-*` y `rdd-defect-workflow` de RDD y el umbral de riesgo del paquete de revisión de RDD; RDD se retira (C-01-e, P20).)

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

### Requirement: Sección «Commit Review» de Fuente Única Proyectada a Todos los Runtimes (RF-ORQ-21)

La doctrina SDD DEBE incluir una sección «Commit Review» con una única fuente en `_shared/sdd-orchestrator-sections.md`, referenciada solo por su marcador desde el orquestador de cada runtime soportado y desde el flujo del orquestador de Claude, de modo que el texto proyectado sea idéntico en todos. La sección DEBE registrarse en la lista de secciones compartidas, de modo que la prueba de fuente única y la de ausencia de deriva la cubran. La sección «SDD Change Kickoff and Block Gates» DEBE enlazar la interacción de la puerta `tasks` de RF-GAT-23. La sección NO DEBE mencionar RDD ni recibos.

#### Scenario: Todos los orquestadores proyectan la sección

- **DADO** los orquestadores generados de todos los runtimes
- **CUANDO** se comprueba la presencia de la sección «Commit Review»
- **ENTONCES** cada uno contiene el mismo texto, expandido desde la fuente única, y ninguno tiene una copia propia

#### Scenario: Un marcador sin sección registrada falla

- **DADO** un orquestador que referencia el marcador de «Commit Review»
- **CUANDO** la sección no figura en la lista de secciones compartidas
- **ENTONCES** la prueba de fuente única falla

#### Scenario: Los goldens reflejan la sección

- **DADO** los goldens de los orquestadores
- **CUANDO** se regeneran con la herramienta de actualización y se ejecutan las pruebas
- **ENTONCES** pasan sin ediciones manuales

### Requirement: Contenido Común de la Doctrina «Commit Review» (RF-ORQ-22)

La sección DEBE establecer, como mínimo: (a) al cerrar cada tarea con commit, el orquestador avisa en el chat con el sha, la tarea y la orden `git show <sha>`; (b) los tres modos de RF-GAT-19 y su efecto, aclarando que en `stop` y `stop-before-commit` la pausa por tarea no es una puerta, no usa las tres salidas y no escribe en `gates.yaml`; (c) en `notify` el trabajo continúa sin parar; (d) pedir la PR da por validados todos los commits pendientes, no existe aprobación commit a commit y nada bloquea la PR; (e) solo se registran los commits sobre los que la persona pidió cambios; (f) para cambios sobre un commit ya superado, si la rama no está publicada (ningún remoto la contiene) se usa `git commit --fixup=<sha>` y un autosquash no interactivo diferido hasta antes de abrir la PR, y si ya está publicada se usa un commit correctivo nuevo, y nunca se reescribe historia publicada; (g) para intercalar un `fixup!` se termina o se aparca primero la tarea en curso; (h) se recomienda fusionar con merge commit, sin squash, para que los commits revisados lleguen a la rama destino, y Axiom no fusiona. Los modos 2 y 3 DEBEN describir además el avance tarea a tarea y, en el modo 3, que el commit solo se hace tras la aprobación de la persona. El comportamiento por tarea DEBE cumplirse aunque la fase de implementación devuelva varias tareas juntas: el aviso es uno por commit y se emite por el orquestador al recuperar el control.

#### Scenario: La sección contiene el aviso y la orden de revisión

- **DADO** la sección proyectada
- **CUANDO** se lee la regla del aviso
- **ENTONCES** pide sha, tarea y `git show <sha>` por cada commit de tarea

#### Scenario: La sección describe los tres modos sin convertirlos en puerta

- **DADO** la sección proyectada
- **CUANDO** se leen los modos `stop` y `stop-before-commit`
- **ENTONCES** declaran que la pausa no es una puerta y no escribe en `gates.yaml`

#### Scenario: La sección declara la validación implícita

- **DADO** la sección proyectada
- **CUANDO** se lee la regla de la PR
- **ENTONCES** dice que pedir la PR valida todos los commits pendientes y que nada bloquea la PR

#### Scenario: La sección distingue publicada de no publicada

- **DADO** la sección proyectada
- **CUANDO** se leen las reglas de `fixup!`
- **ENTONCES** indican autosquash diferido antes de la PR si la rama no está publicada, commit correctivo si lo está, y la prohibición de reescribir historia publicada

#### Scenario: La sección recomienda merge sin squash

- **DADO** la sección proyectada
- **CUANDO** se lee la recomendación de fusión
- **ENTONCES** recomienda merge commit sin squash y declara que Axiom no fusiona

#### Scenario: Un lote devuelto de golpe sigue produciendo un aviso por commit

- **DADO** un lote de `apply` que devuelve tres commits de tarea en modo `notify`
- **CUANDO** el orquestador recupera el control
- **ENTONCES** emite tres avisos, uno por sha, cada uno con su tarea y su `git show <sha>`

### Requirement: Contrato `commit_review` de `sdd-apply` (RF-ORQ-23)

La skill `sdd-apply` DEBE documentar el parámetro de lanzamiento `commit_review` con el vocabulario de RF-GAT-19, tomado de la proyección de `status` (RF-GAT-22), con `notify` por defecto cuando falta. En `notify` DEBE devolver, junto a cada tarea completada, el sha de su commit. En `stop` y `stop-before-commit` DEBE devolver el control al orquestador tras cada tarea. En `stop-before-commit` NO DEBE hacer el commit de la tarea hasta recibir la aprobación de la persona, dejando el trabajo de esa tarea sin confirmar y dejando constancia del estado de la tarea para poder reanudar. La skill DEBE indicar dónde se registran, en SDD, los commits con cambios pedidos (RF-ORQ-22 e), en un artefacto de seguimiento del rol cuya ubicación fija design, sin reabrir la puerta `tasks`.

#### Scenario: `notify` devuelve un sha por tarea

- **DADO** `apply` lanzado con `commit_review: notify` o sin el parámetro
- **CUANDO** completa varias tareas
- **ENTONCES** el resultado lista cada tarea con el sha de su commit

#### Scenario: `stop` devuelve el control tras cada tarea

- **DADO** `apply` lanzado con `commit_review: stop`
- **CUANDO** completa una tarea con su commit
- **ENTONCES** devuelve el control con ese sha y no empieza la tarea siguiente

#### Scenario: `stop-before-commit` no confirma antes de la aprobación

- **DADO** `apply` lanzado con `commit_review: stop-before-commit`
- **CUANDO** termina el trabajo de una tarea
- **ENTONCES** devuelve el control con el cambio sin confirmar y sin sha
- **Y** el commit se hace solo tras la aprobación de la persona

#### Scenario: Registrar cambios pedidos no reabre `tasks`

- **DADO** una puerta `tasks` aprobada
- **CUANDO** se registra un commit con cambios pedidos
- **ENTONCES** la puerta `tasks` sigue `approved`, no se reabre y conserva su `commit_review`, aunque cambie el contenido del fichero de tareas

### Requirement: La Skill `work-unit-commits` Incluye `fixup!`, Autosquash y Merge sin Squash, en Paridad (RF-ORQ-24)

Las dos copias de la skill `work-unit-commits` (la embebida en `internal/assets/skills/` y la del repositorio en `skills/`) DEBEN recibir las reglas de RF-ORQ-22 (f) y (h) con el mismo texto, y la paridad entre ambas DEBE seguir vigilada por la prueba de paridad de copias.

#### Scenario: Las copias mantienen la paridad

- **DADO** las dos copias de `work-unit-commits`
- **CUANDO** se ejecuta la prueba de paridad
- **ENTONCES** pasa y ambas contienen las reglas de `fixup!`, autosquash diferido, commit correctivo y merge sin squash

#### Scenario: Divergir entre copias falla

- **DADO** una copia con las reglas y otra sin ellas
- **CUANDO** se ejecuta la prueba de paridad
- **ENTONCES** falla
### Requirement: La Doctrina Proyectada No Contiene la Prosa de RDD (RF-ORQ-25)

La doctrina de los orquestadores SDD y ODD proyectada a los doce runtimes, la fuente compartida `_shared/sdd-orchestrator-sections.md`, las skills `judgment-day` y `sdd-verify`, la convención `openspec-convention.md` y los goldens regenerados NO DEBEN contener la sección `Review Execution Contract` ni su marcador, la sección «Receipt-driven development is user-owned», el vocabulario de revisión por recibos (`lineage`, `receipt`, `acknowledge`, `burn`, `review/start(target)`, `review-refuter`) ni comandos `axiom review`. La prosa de entrega (`single-pr`), de Commit Review y de Judgment Day NO cambia salvo por la retirada de sus menciones a RDD.

#### Scenario: Ningún orquestador contiene la prosa de revisión por recibos

- **DADO** el orquestador renderizado para cada AgentID de `catalog.AllAgents()`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** no contiene `Review Execution Contract`, `receipt-driven`, `axiom review`, `review-refuter` ni `rdd-defect-workflow`
- **Y** conserva intactas las secciones de entrega, Commit Review y Judgment Day

#### Scenario: Las skills compartidas no mencionan RDD

- **DADO** las skills `judgment-day` y `sdd-verify` y la convención `openspec-convention.md`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no contienen `RDD`, `receipt` ni `axiom review`
- **Y** `judgment-day` conserva sus dos jueces ciegos y sus artefactos `review-ledger`

### Requirement: `axiom sync` No Instala Agentes, Plugins, Skill ni Hook de RDD (RF-ORQ-26)

`axiom sync` y la instalación NO DEBEN escribir ningún agente `review-*` (`review-risk`, `review-resilience`, `review-readability`, `review-reliability`, `review-refuter`, `review-validator`) en ningún runtime, ni sus entradas en los overlays de OpenCode, ni los plugins `opencode-review-transport.ts`, ni la skill `rdd-defect-workflow`, ni ningún hook `Stop` o `SessionStart` cuya orden sea `axiom review stop-hook` (o `gentle-ai review stop-hook`). Tampoco DEBEN modificar ni borrar los que ya estuvieran instalados por una versión anterior: la limpieza es manual (sección de migración manual de la propuesta C-01-e). Los agentes de Judgment Day (`jd-judge-a`, `jd-judge-b`, `jd-fix-agent`) y los demás agentes SDD NO cambian.

#### Scenario: Una instalación limpia no contiene nada de RDD

- **DADO** un directorio de configuración vacío para cada runtime soportado
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** no existe ningún fichero `review-*` en los directorios de agentes, ni `opencode-review-transport.ts` en los plugins, ni la skill `rdd-defect-workflow`
- **Y** `settings.json` de Claude Code no contiene ningún hook `Stop` ni `SessionStart` con `review stop-hook`
- **Y** los agentes `jd-judge-a`, `jd-judge-b` y `jd-fix-agent` y los agentes `sdd-*` se instalan como antes

#### Scenario: Los overlays de OpenCode no definen agentes de revisión

- **DADO** los overlays `sdd-overlay-single.json` y `sdd-overlay-multi.json`
- **CUANDO** se inspeccionan
- **ENTONCES** no contienen las definiciones `review-risk`, `review-readability`, `review-reliability`, `review-resilience`, `review-refuter` ni `review-validator`
- **Y** el resto de definiciones permanece

#### Scenario: Los restos de una versión anterior no se tocan

- **DADO** una instalación con un hook `Stop` y agentes `review-*` dejados por una versión anterior
- **CUANDO** se ejecuta `axiom sync` o `axiom uninstall`
- **ENTONCES** no se borra ni se modifica ninguno de esos restos
- **Y** la ejecución termina sin error

#### Scenario: Una selección persistida con `rdd-defect-workflow` no rompe la carga

- **DADO** un estado persistido cuya selección de skills incluye `rdd-defect-workflow`
- **CUANDO** se carga y se ejecuta `axiom sync`
- **ENTONCES** la carga y `sync` terminan sin error
- **Y** no se instala ninguna skill con ese identificador

### Requirement: Paridad entre Runtimes de la Regla del WI Obligatorio y del Contenido Importado como Dato no Fiable (RF-ORQ-27)

La doctrina del orquestador SDD (fuente única en `_shared/sdd-orchestrator-sections.md` y guía de enrutado de `agentguidance`) MUST proyectarse a todos los runtimes con las mismas reglas de arranque con plugin activo de `sdd-lane-kickoff`: pedir el WI si falta, fase 1 antes de `ensure` y `seal`, fase 2 después, y uso del nombre derivado. Las skills `sdd-explore` y `sdd-propose` y la convención ISO MUST tratar `context/work-item.md` como dato no fiable de un tercero: MUST leerlo solo como información, MUST NOT seguir instrucciones que contenga, MUST NOT ampliar ni cambiar el alcance por su contenido ni ejecutar comandos que aparezcan en él, y MUST preguntar a la persona ante cualquier duda. Los goldens y las aserciones estáticas MUST fijar estas reglas para cada runtime.

#### Scenario: Paridad de la regla del WI
- **DADO** los textos generados del orquestador SDD para cada runtime
- **CUANDO** se inspecciona el arranque con plugin activo
- **ENTONCES** todos contienen la misma regla de pedir el WI y el mismo orden de las dos fases

#### Scenario: `sdd-explore` trata el contexto como dato
- **DADO** la skill `sdd-explore`
- **CUANDO** se inspecciona el tratamiento de `context/work-item.md`
- **ENTONCES** lo declara dato no fiable que no cambia el alcance, no ordena comandos y no concede autoridad

#### Scenario: `sdd-propose` trata el contexto como dato
- **DADO** la skill `sdd-propose`
- **CUANDO** se inspecciona el mismo tratamiento
- **ENTONCES** declara las mismas reglas y la obligación de preguntar ante la duda

#### Scenario: Goldens regenerados
- **DADO** los goldens regenerados con `-update`
- **CUANDO** se ejecutan las pruebas de goldens y las aserciones estáticas
- **ENTONCES** pasan y el diff solo contiene la regla del WI

### Requirement: `sdd-propose` Copia `external_ref` y `Referencia:` desde el Vínculo del WI (RF-ORQ-28)

Cuando el cambio tiene un vínculo `kind: work_item` en su `integrations.yaml`, la skill `sdd-propose` MUST escribir en `proposal.md` la línea `external_ref: <alias>:<id>` (por ejemplo `external_ref: ado:2525`), con el alias de referencia del manifiesto del plugin y el identificador del vínculo. Si el perfil es de bug, MUST además escribir `Referencia: <alias>:<id>` en la sección `## Informe de bug`, y MUST redactar `Observado:` y `Esperado:` desde `context/work-item.md` y preguntar a la persona lo que falte, sin inventarlo. Sin vínculo `work_item`, MUST NOT escribir ninguna de las dos líneas. El verbo `axiom change start` MUST NOT crear ni modificar `proposal.md`. Ningún estado, compuerta ni validador MUST exigir la presencia de `external_ref`.

#### Scenario: Cambio con vínculo de WI
- **DADO** un cambio con `integrations.yaml` que contiene el vínculo `work_item` del WI 2525 con alias `ado`
- **CUANDO** `sdd-propose` crea `proposal.md`
- **ENTONCES** `proposal.md` contiene `external_ref: ado:2525`

#### Scenario: Bug con vínculo de WI
- **DADO** un cambio de perfil bug con el vínculo `work_item` del WI 7
- **CUANDO** `sdd-propose` crea `proposal.md`
- **ENTONCES** contiene `external_ref: ado:7` y, bajo `## Informe de bug`, la línea `Referencia: ado:7`
- **Y** `Observado:` y `Esperado:` proceden del contexto o de la respuesta de la persona, nunca del verbo

#### Scenario: Falta información del bug
- **DADO** un WI de bug sin pasos de reproducción ni resultado esperado
- **CUANDO** `sdd-propose` redacta el informe
- **ENTONCES** pregunta a la persona lo que falta en lugar de inventarlo

#### Scenario: Sin vínculo no hay referencia
- **DADO** un cambio sin vínculo `work_item`
- **CUANDO** `sdd-propose` crea `proposal.md`
- **ENTONCES** no escribe `external_ref` ni `Referencia:`

#### Scenario: La referencia no condiciona el estado
- **DADO** un cambio con `proposal.md` sin `external_ref`
- **CUANDO** se ejecuta `axiom sdd status`
- **ENTONCES** el estado no la exige ni la proyecta como bloqueo
### Requirement: El Orquestador Invoca los Verbos de Tasks y de Resumen con Tracker Activo (RF-ORQ-29)

Cuando el cambio tiene un vínculo `kind: work_item` y `axiom change start --json` no responde `not_applicable` (hay un plugin de work items activo), la doctrina DEBE ordenar al orquestador invocar los verbos de `axiom integrations` en estos momentos, con el mismo contenido en `checkpointed` y en `continuous` y sin añadir ninguna pregunta de gobierno: al terminar `spec`, `summary push`; al terminar `design`, `tasks create` (con la estimación de la sección `## Estimación por rol` de `design.md`, sin `--estimate` salvo que la persona pida cambiarla); al cerrar cada rol, `time propose` y `time record`; y al terminar `archive`, `summary push` sobre la carpeta archivada. En `checkpointed` DEBE invocarlos tras registrar la compuerta correspondiente (`gate record`), nunca antes ni sustituyéndola; en `continuous` DEBE invocarlos al terminar la fase, sin parada. Sin vínculo de WI o sin plugin activo la doctrina NO DEBE invocar ninguno ni anunciarlo. La doctrina NO DEBE ordenar ninguno de estos verbos en el carril ODD. Un fallo de un verbo NO DEBE deshacer ni repetir la aprobación de la compuerta: en `checkpointed` el orquestador presenta el rechazo tipado con sus resoluciones y la parada ofrece sus tres salidas habituales; en `continuous` no se detiene por el fallo de `summary push`, de `time record` ni de la transición de estado, y los recoge para el resumen final.

#### Scenario: `checkpointed` tras la compuerta
- **DADO** un cambio `checkpointed` con tracker activo y la compuerta `design` recién aprobada
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** invoca `axiom integrations tasks create --change <cambio>` después de `gate record`

#### Scenario: `continuous` sin parada
- **DADO** un cambio `continuous` con tracker activo y `design` terminado
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** invoca `tasks create` al terminar la fase y encadena la siguiente sin presentar ninguna parada ni pregunta

#### Scenario: Sin tracker activo no se invoca nada
- **DADO** un cambio sin plugin de work items activo
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** no invoca ningún verbo `tasks`, `summary` ni `time` y no lo menciona

#### Scenario: Fallo de `tasks create` en `checkpointed`
- **DADO** un `tasks create` que falla con la compuerta `design` aprobada
- **CUANDO** el orquestador presenta la parada
- **ENTONCES** muestra el rechazo con su resolución y la parada ofrece sus tres salidas
- **Y** la aprobación de `design` no se repite ni se deshace

#### Scenario: Fallo de `summary push` en `continuous`
- **DADO** un cambio `continuous` y un `summary push` que falla
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** no se detiene y registra el fallo para el resumen final

#### Scenario: Carril ODD
- **DADO** una característica ODD con tracker activo
- **CUANDO** el orquestador sigue la doctrina de ODD
- **ENTONCES** no invoca ninguno de los verbos nuevos

### Requirement: `ensure --role` Antes del Apply con Tracker Activo (RF-ORQ-30)

Con tracker activo y vínculo `kind: work_item`, la doctrina DEBE ordenar al orquestador invocar `axiom branch ensure --change <cambio> --role <rol>` antes de lanzar `apply` de cada rol, también con el único rol `fullstack` y en ambos estilos de ejecución, para que la barrera `task_not_bound` y el registro de `started_at` ocurran siempre. Si `ensure --role` rechaza con `task_not_bound`, el orquestador DEBE NO lanzar `apply`, presentar el rechazo con sus dos resoluciones (`tasks create` y `tasks bind`) y esperar: es un bloqueo reportado a la persona, no una parada de compuerta, y se produce igualmente en `continuous`. Sin tracker activo NO DEBE invocarlo por esta regla.

#### Scenario: Apply del rol único
- **DADO** un cambio `continuous` con rol `fullstack` y tracker activo
- **CUANDO** el orquestador va a lanzar `apply`
- **ENTONCES** invoca antes `axiom branch ensure --change x --role fullstack`

#### Scenario: Bloqueo por Task sin vincular
- **DADO** un `ensure --role` que rechaza con `task_not_bound`
- **CUANDO** el orquestador lo recibe
- **ENTONCES** no lanza `apply` y presenta las resoluciones `axiom integrations tasks create --change x` y `axiom integrations tasks bind --change x --role fullstack=<id>`

#### Scenario: Sin tracker
- **DADO** un cambio sin plugin activo
- **CUANDO** el orquestador va a lanzar `apply`
- **ENTONCES** no añade el `ensure --role` por esta regla

### Requirement: `sdd-design` Propone la Estimación por Rol (RF-ORQ-31)

Con tracker activo, la skill `sdd-design` DEBE escribir en `design.md` la sección fija `## Estimación por rol` con una línea `- <rol>: <horas> h` por cada rol del roster sellado, con horas decimales positivas y una justificación breve fuera de las líneas de datos. En `checkpointed`, la aprobación de la compuerta `design` confirma esas horas porque forman parte de su digest; para cambiarlas se itera el diseño o se pasa `--estimate`, que prevalece. Sin tracker activo la skill NO DEBE escribir la sección.

#### Scenario: Sección presente con tracker activo
- **DADO** un cambio con tracker activo y roster `fullstack`
- **CUANDO** `sdd-design` termina
- **ENTONCES** `design.md` contiene `## Estimación por rol` con una línea `- fullstack: <horas> h`

#### Scenario: Sin tracker
- **DADO** un cambio sin tracker activo
- **CUANDO** `sdd-design` termina
- **ENTONCES** `design.md` no contiene la sección

#### Scenario: Una línea por rol del roster
- **DADO** un roster sellado con dos roles
- **CUANDO** `sdd-design` escribe la sección
- **ENTONCES** hay exactamente una línea por cada uno de los dos roles

### Requirement: `sdd-apply` Propone e Imputa las Horas al Cerrar el Rol (RF-ORQ-32)

Con tracker activo, al cerrar el apply de un rol la doctrina DEBE ordenar `axiom integrations time propose --change <cambio> --role <rol>` y después `time record` con las horas definitivas. En `checkpointed` DEBE presentar la propuesta a la persona, aceptar sus horas (que PUEDEN diferir de la propuesta) y llamar a `time record` solo tras esa confirmación, y solo con la compuerta `role-apply:<rol>` aprobada. En `continuous` DEBE llamar a `time record` con las horas de la propuesta, sin parada y sin pedir confirmación. Un `rev_conflict` NO DEBE reintentarse ni reaplicarse automáticamente: se presenta con los valores releídos y la decisión es de la persona (en `continuous` queda en el resumen final como imputación pendiente). La skill NO DEBE imputar en el carril ODD ni sin tracker activo.

#### Scenario: `checkpointed` con confirmación
- **DADO** un rol cerrado con `role-apply:fullstack` aprobada y una propuesta de 11 horas
- **CUANDO** la persona confirma 9 horas
- **ENTONCES** el orquestador ejecuta `time record --hours 9` y no 11

#### Scenario: `continuous` sin parada
- **DADO** un rol cerrado en `continuous` con una propuesta de 11 horas
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** ejecuta `time record --hours 11` sin presentar ninguna parada

#### Scenario: Conflicto de revisión
- **DADO** un `time record` que termina con `rev_conflict`
- **CUANDO** el orquestador lo recibe
- **ENTONCES** no lo reintenta, presenta los valores releídos y deja la decisión a la persona

#### Scenario: Sin tracker o en ODD
- **DADO** un cambio sin tracker activo, o una característica ODD
- **CUANDO** se cierra el rol o la característica
- **ENTONCES** no se propone ni se imputa ninguna hora

### Requirement: Doctrina de los Verbos de Integración en Paridad entre Runtimes y Goldens (RF-ORQ-33)

La doctrina de los requisitos anteriores DEBE tener una única fuente proyectada a todos los runtimes que proyectan el orquestador (junto al texto de `axiom change start` en el enrutado), de modo que el orquestador, `sdd-design`, `sdd-apply` y `sdd-archive` contengan las mismas reglas con las mismas palabras de comando en cada runtime, y las copias espejo y los goldens DEBEN regenerarse con ellas. La doctrina NO DEBE nombrar ni valores de secretos ni rutas del tracker.

#### Scenario: Todos los runtimes contienen la doctrina
- **DADO** la proyección del orquestador a cada runtime
- **CUANDO** se comprueba su texto
- **ENTONCES** todos contienen los comandos `tasks create`, `summary push`, `time propose` y `time record` y la regla de `ensure --role` antes del apply

#### Scenario: Goldens actualizados
- **DADO** el texto de la doctrina cambiado
- **CUANDO** se ejecutan las pruebas de goldens y de paridad de las skills
- **ENTONCES** las copias espejo y los goldens coinciden con la fuente

#### Scenario: Sin secretos en la doctrina
- **DADO** el texto proyectado
- **CUANDO** se escanea
- **ENTONCES** no contiene ningún valor de secreto
