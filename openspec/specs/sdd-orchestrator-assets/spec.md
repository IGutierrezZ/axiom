# SDD Orchestrator Assets Specification

## Purpose

Defines required behavior for agent-specific SDD orchestrator guidance assets so Claude and OpenCode guidance remain semantically aligned where intended, while preserving platform-accurate delegation wording.

---

## Requirements

### Requirement: Chain Strategy Guidance Parity

For Claude-targeted SDD orchestrator assets, the system MUST include explicit chain strategy guidance equivalent to OpenCode for reviewer-safe delivery planning.

The guidance MUST enumerate exactly these selectable strategies:
- `stacked-to-main`
- `feature-branch-chain`

#### Scenario: Claude guidance lists both required strategies

- GIVEN Claude orchestrator guidance is generated from embedded assets
- WHEN the chain strategy section is rendered
- THEN it includes `stacked-to-main`
- AND it includes `feature-branch-chain`

#### Scenario: Strategy naming remains canonical

- GIVEN a static validation checks chain strategy labels in Claude guidance
- WHEN the check runs
- THEN no alias or renamed value replaces the canonical strategy names

---

### Requirement: Chain Strategy Propagation to Downstream Phases

Claude orchestrator guidance MUST require passing `chain_strategy` together with `delivery_strategy` into `sdd-tasks` and `sdd-apply` whenever delivery planning is relevant.

#### Scenario: Guidance forwards both strategy fields to sdd-tasks

- GIVEN the orchestrator invokes `sdd-tasks` for a change requiring delivery planning
- WHEN invocation guidance is rendered
- THEN `delivery_strategy` is forwarded
- AND `chain_strategy` is forwarded

#### Scenario: Guidance forwards both strategy fields to sdd-apply

- GIVEN the orchestrator invokes `sdd-apply` after planning
- WHEN invocation guidance is rendered
- THEN `delivery_strategy` is forwarded
- AND `chain_strategy` is forwarded

---

### Requirement: Non-Claude Chain Strategy Guidance Parity

For non-Claude SDD orchestrator assets that support SDD delivery planning, the system MUST include explicit chain strategy guidance equivalent in intent to the Claude/OpenCode reference guidance.

The guidance MUST enumerate exactly these selectable strategies:
- `stacked-to-main`
- `feature-branch-chain`

#### Scenario: Non-Claude delivery-planning guidance lists both required strategies

- GIVEN a non-Claude orchestrator asset that includes delivery planning guidance
- WHEN the chain strategy section is rendered
- THEN it includes `stacked-to-main`
- AND it includes `feature-branch-chain`

#### Scenario: Non-Claude strategy naming remains canonical

- GIVEN static validation checks chain strategy labels in non-Claude orchestrator assets
- WHEN the check runs
- THEN no alias or renamed value replaces the canonical strategy names

---

### Requirement: Non-Claude Strategy Propagation to Downstream Phases

For non-Claude SDD orchestrator assets that invoke downstream delivery phases, the system MUST require passing `chain_strategy` together with `delivery_strategy` into `sdd-tasks` and `sdd-apply` whenever delivery planning is relevant.

#### Scenario: Non-Claude guidance forwards both strategy fields to sdd-tasks

- GIVEN a non-Claude orchestrator asset invokes `sdd-tasks` for a change requiring delivery planning
- WHEN invocation guidance is rendered
- THEN `delivery_strategy` is forwarded
- AND `chain_strategy` is forwarded

#### Scenario: Non-Claude guidance forwards both strategy fields to sdd-apply

- GIVEN a non-Claude orchestrator asset invokes `sdd-apply` after planning
- WHEN invocation guidance is rendered
- THEN `delivery_strategy` is forwarded
- AND `chain_strategy` is forwarded

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

All 12 SDD orchestrator assets MUST bind the `chained-pr` skill into their chain-strategy guidance. When delivery planning produces chained or stacked PRs, the orchestrator MUST resolve the `chained-pr` skill by its registry name (not a hardcoded filesystem path) through its existing Sub-Agent Launch Pattern, and MUST inject the resolved skill path into `sdd-tasks` and `sdd-apply` phase prompts under `## Skills to load before work`. Those phases MUST read and follow the skill BEFORE planning or creating any PR.

The binding MUST reference the skill by registry name `chained-pr` (frontmatter `chained-pr`). No hardcoded path to a `SKILL.md` file is permitted as the binding reference.

#### Scenario: Orchestrator resolves chained-pr skill by registry name on chained delivery

- GIVEN an SDD orchestrator asset has chain strategy guidance
- WHEN delivery planning determines that chained or stacked PRs are required
- THEN the orchestrator resolves the `chained-pr` skill by its registry name through the Sub-Agent Launch Pattern
- AND the resolved skill path is injected into the `sdd-tasks` prompt under `## Skills to load before work`
- AND the resolved skill path is injected into the `sdd-apply` prompt under `## Skills to load before work`
- AND no hardcoded filesystem path is used as the binding reference

#### Scenario: Binding is present in all 12 orchestrator templates

- GIVEN the 12 SDD orchestrator templates are reviewed (antigravity, claude, codex, cursor, gemini, generic, hermes, kimi, kiro, opencode, qwen, windsurf)
- WHEN the chain strategy section of each template is inspected
- THEN each template contains a binding instruction referencing the `chained-pr` skill by registry name
- AND the binding instruction is located inside or immediately adjacent to the chain strategy section

#### Scenario: sdd-tasks sub-agent reads chained-pr skill before planning PRs

- GIVEN an `sdd-tasks` sub-agent is launched with a `chain_strategy` requiring chained PRs
- WHEN the sub-agent begins planning
- THEN the prompt includes `chained-pr` skill under `## Skills to load before work`
- AND the sub-agent reads and follows the skill before producing a PR plan

#### Scenario: sdd-apply sub-agent reads chained-pr skill before creating PRs

- GIVEN an `sdd-apply` sub-agent is launched for a change requiring chained PRs
- WHEN the sub-agent begins implementation
- THEN the prompt includes `chained-pr` skill under `## Skills to load before work`
- AND the sub-agent reads and follows the skill before creating any PR

---

### Requirement: Inline Chain Strategy Summary Retained

All 12 SDD orchestrator assets MUST retain a short inline summary of both `stacked-to-main` and `feature-branch-chain` strategies in their chain strategy section. The binding supplements, and does NOT replace, the existing inline summary. Canonical strategy names MUST continue to appear verbatim so that existing "Strategy naming remains canonical" scenarios continue to hold.

#### Scenario: Chain strategy inline summary coexists with skill binding

- GIVEN an SDD orchestrator asset contains a chain strategy section with the skill binding
- WHEN the section is inspected
- THEN `stacked-to-main` appears as a canonical strategy name
- AND `feature-branch-chain` appears as a canonical strategy name
- AND a binding instruction referencing the `chained-pr` skill by registry name also appears
- AND the binding instruction does not replace the inline strategy summaries

#### Scenario: Static validation passes after binding is added

- GIVEN static-validation tests assert canonical strategy names and chain strategy section presence
- WHEN tests execute against the updated orchestrator templates
- THEN all assertions for canonical strategy names continue to pass
- AND new binding assertions also pass
- AND no test requires the binding to have replaced the inline summary

---

### Requirement: Cursor Chain Strategy Section Parity

The `cursor/sdd-orchestrator.md` template MUST include a `### Chain Strategy` section that is at parity with the other 10 templates. The section MUST enumerate both canonical strategy names, forward `chain_strategy` to downstream phases, and include the `chained-pr` skill binding.

#### Scenario: Cursor template includes a Chain Strategy section

- GIVEN the cursor SDD orchestrator template is inspected
- WHEN the document structure is reviewed
- THEN a `### Chain Strategy` section is present
- AND it includes `stacked-to-main`
- AND it includes `feature-branch-chain`

#### Scenario: Cursor forwards chain_strategy to downstream phases

- GIVEN the cursor SDD orchestrator template invokes `sdd-tasks` or `sdd-apply`
- WHEN the invocation guidance is rendered
- THEN `delivery_strategy` is forwarded
- AND `chain_strategy` is forwarded

#### Scenario: Cursor includes the chained-pr skill binding

- GIVEN the cursor SDD orchestrator template chain strategy section is inspected
- WHEN the binding instruction is reviewed
- THEN the `chained-pr` skill is referenced by registry name
- AND the binding instructs injection into `sdd-tasks` and `sdd-apply` prompts

#### Scenario: Cursor chain strategy naming is canonical

- GIVEN a static validation checks chain strategy labels in the cursor template
- WHEN the check runs
- THEN no alias or renamed value replaces the canonical strategy names `stacked-to-main` and `feature-branch-chain`

---

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
