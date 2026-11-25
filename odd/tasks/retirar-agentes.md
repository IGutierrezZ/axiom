# ODD: Retirar los agentes que no se usan (retirar-agentes)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/retirar-agentes.md`. Espejo Engram: `odd/retirar-agentes/tasks`.
> Rama: `inc/retirar-agentes` · worktree `C:\repos\axiom-wt\retirar-agentes\axiom`. Cambio C-08-d del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A3; decisiones P17 y P20 (apartado 6.2 del README y coordinación).

## Objetivo

Retirar seis agentes: `kiro-ide`, `windsurf`, `kimi`, `hermes`, `openclaw` y `kilocode`. Se quedan once: `claude-code`, `vscode-copilot`, `opencode`, `github-copilot-cli`, `gemini-cli`, `codex`, `cursor`, `antigravity`, `pi`, `trae-ide` y `qwen-code` (este sigue oculto al instalar).

## Problema y porqué

Los seis agentes no se usan y cuestan unas 12.300 líneas (adaptadores, assets, goldens, ramas por agente en la inyección, la desinstalación y la TUI). Decisión del usuario (P17, 2026-11-24): de los agentes heredados se van todos menos Gemini CLI, Codex, Cursor, Antigravity, Pi, Copilot CLI y los que apenas ahorran (Trae y Qwen).

## Decisiones

- **P20 (usuario: «Sí, borrarlas ya»).** Sin lápidas: `uninstall` deja de conocer las rutas de los agentes retirados y no hay comandos tumba. Lo que Axiom escribió en equipos con instalaciones antiguas lo limpia la persona a mano (lista al cerrar, en `## Siguiente paso`).
- **Lector tolerante.** Un estado, copia o selección guardados con un `AgentID` retirado siguen cargando: el agente se ignora con un aviso y se sincronizan los demás.
- **Código compartido.** Se queda con el agente que lo conserva; solo se borra la rama del retirado. Kilo Code comparte con OpenCode `agents/opencode/adapter.go`, `agents/skills_dir.go` y la familia OpenCode de `components/sdd/inject.go` y `uninstall`: lo generado para OpenCode no cambia (goldens de OpenCode sin diferencias).
- **Specs vivas.** Solo se retiran menciones a estos agentes (requisitos y escenarios que solo hablan de ellos, y listas que los nombran). Si hiciera falta reescribir requisitos, el cambio pasaría a SDD.
- **Históricos.** No se tocan `docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `docs/releases` ni los `odd/tasks/` cerrados, salvo marcar como superados `odd/tasks/kiro-workspace-parity.md` y la parte de Windsurf de `odd/tasks/codex-sync-sin-temas-y-rollback.md`.

## Alcance autorizado

Lo que lista C-08-d en la sección 08: registro (`model`, `catalog`, `agents/factory.go`, `capabilitymanifest`, `discovery`, paquetes de adaptador), instalación y CLI (`installcmd`, `cli/{validate,run,sync,workspace_*}`), TUI (selector de modelos de Kiro y referencias), componentes (`sdd`, `persona`, `permissions`, `engram`, `mcp`, `uninstall`), assets, goldens, docs, specs vivas y e2e. Más las rutas de configuración de esos agentes en `system/config_scan.go`, `skillregistry`, `semantic/detector.go` y `dashboard`.

Fuera de alcance: los once agentes que se quedan, persona/presets/temas/agent builder (C-08-f), GGA (C-08-e).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones con `go vet` y `go test` por paquetes, sin binarios sueltos de `axiom` (el EDR los borra). El lector tolerante (T7) se prueba con un test nuevo.

## Entrega

`single-pr`: una PR con un commit por tarea (un commit por agente). Etiqueta `type:refactor`. Previsión: unas 12.300 líneas, casi todo borrado.

## Tareas

- [x] T1 · Retirar Kilo Code (adaptador, registro, ramas de la familia OpenCode, instalación, desinstalación, tests). Goldens de OpenCode sin diferencias.
- [x] T2 · Retirar OpenClaw.
- [x] T3 · Retirar Hermes (incluida la skill `hermes-ephemeral-delegation`).
- [x] T4 · Retirar Kimi (assets y agentes `review-*` que queden).
- [x] T5 · Retirar Windsurf (assets y goldens).
- [x] T6 · Retirar Kiro IDE (modelos de Kiro, selector de la TUI, assets y goldens).
- [x] T7 · Lector tolerante: un estado con agentes retirados carga y sincroniza el resto con un aviso.
- [x] T8 · Docs, e2e, documentos ODD superados, `## Spec viva` integrada con `axiom archive integrate --odd retirar-agentes` y criterio `rg`.

Ruta de todas las tareas: inline (borrado guiado por el compilador; sin unidades independientes con superficies disjuntas que justifiquen escritores en paralelo).

## Criterio de terminado

- `internal/model` y el catálogo solo conocen los once agentes conservados.
- La instalación (TUI y CLI) ofrece los diez seleccionables; Qwen Code sigue oculto.
- Un estado con agentes retirados carga y sincroniza con un aviso.
- `rg -il 'kiro|windsurf|kimi|hermes|openclaw|kilocode|kilo code'` no encuentra nada fuera de los históricos y del lector tolerante.
- Goldens regenerados y revisados por diff; los de OpenCode, sin cambios. `go vet ./...` y `go test` de los paquetes tocados en verde.


## Spec viva

Solo retira menciones a los agentes retirados (requisitos y escenarios que solo hablaban de ellos, y enumeraciones que los nombraban); ningún requisito cambia de comportamiento para los agentes que se quedan.

### Capacidad: persona-behavior-contract

```openspec
## REMOVED Requirements

### Requirement: Kimi Neutral Output Style Content

(Reason: C-08-d retira Kimi; el contrato de su estilo de salida neutral deja de tener consumidor.)

### Requirement: Kimi Non-Duplication and Documentation Accuracy

(Reason: C-08-d retira Kimi; ya no hay módulos de inclusión ni fila de Kimi en docs/agents.md.)

## MODIFIED Requirements

### Requirement: Generic Neutral Asset Parity

All neutral consumers that are not covered by an agent-specific override MUST receive parity through the generic neutral persona or output-style asset. Agent-specific assets MAY adapt wording to platform mechanics, but MUST NOT weaken the neutral behavior contract. For adapters with an active output-style channel (Claude Code), parity with the generic neutral requirements MUST be evaluated over the COMBINED persona-residual + output-style channel, not over the persona file in isolation — the residual file alone deliberately omits tone/language/philosophy content that now lives exclusively in the output style.
(Previously: Kimi was also listed as an adapter with an active output-style channel.)

#### Scenario: Non-agent-specific consumers receive generic neutral parity

- GIVEN an agent or surface consumes the generic neutral persona asset
- WHEN neutral instructions are rendered for that consumer
- THEN the rendered content includes the neutral mentor behavior contract
- AND it includes brevity, one-question, no-menu, verification-first, and artifact-language constraints

#### Scenario: Agent-specific neutral assets do not weaken generic behavior

- GIVEN an agent has its own neutral persona or output-style asset
- WHEN that asset is compared against the generic neutral behavior contract
- THEN it preserves all generic neutral requirements
- AND any agent-specific differences are limited to platform-accurate wording or installation mechanics

#### Scenario: Output-style-capable adapters are evaluated on the combined channel

- GIVEN Claude Code has a residual neutral persona section plus a reconciled neutral output-style asset
- WHEN that adapter's neutral behavior contract is compared against the generic neutral requirements
- THEN the combined content of the residual persona section and the output-style asset preserves all generic neutral requirements
- AND the residual persona section alone is not required to independently preserve tone/language/philosophy content

### Requirement: Canonical Tone Channel for Output-Style-Capable Adapters

For any adapter with an active output-style channel — Claude Code (gated by `SupportsOutputStyles()`) — the output-style asset MUST be the exclusive source of persona tone, language, and philosophy. The system-prompt persona section for these adapters MUST be reduced to a residual block containing only `## Rules`, `## Expertise`, `## Contextual Skill Loading (MANDATORY)` and a one-line pointer to the output style. The residual MUST NOT duplicate tone/language/philosophy content.
(Previously: Kimi was an explicit carve-out with its own `## Kimi-native notes` section.)

#### Scenario: Claude residual section carries no tone content

- GIVEN Claude assets are generated with persona `gentleman` or `neutral`
- WHEN the CLAUDE.md persona section is inspected
- THEN it contains only Rules, Expertise, Contextual Skill Loading and a pointer to the output style
- AND it contains no tone, language, or philosophy prose

### Requirement: Persona Section Parity for Non-Output-Style Adapters

Adapters without an active output-style channel MUST continue to receive the full persona section — tone, language, philosophy, rules, expertise, and skill loading — unchanged by this change.
(Previously: the scenario excluded Kimi as well as Claude Code.)

#### Scenario: Non-output-style adapter keeps full persona section

- GIVEN an adapter other than Claude Code is generated with a persona
- WHEN its persona asset is inspected
- THEN it contains the full persona section, including tone/language/philosophy, as before this change

### Requirement: Legacy Fingerprint Continuity

Uninstall and auto-heal fingerprint checks that rely on literal markers from the persona section (e.g., `## Rules`, `Senior Architect`) MUST continue to match on-disk content after tone/philosophy content is removed, or the fingerprint constants MUST be updated so uninstall and auto-heal continue to function correctly.
(Previously: the scenario also covered Kimi installations.)

#### Scenario: Uninstall and auto-heal fingerprints still match after slimming

- GIVEN a Claude installation has the residual persona section on disk
- WHEN the uninstall cleaner and auto-heal each check their persona-section fingerprint
- THEN both checks match the residual on-disk content
- AND auto-heal does not false-positive a repair

### Requirement: Per-Adapter Test and Golden Coverage

Automated tests and golden fixtures MUST cover the residual persona section, the reconciled canonical output-style content, and unaffected non-output-style adapter behavior, and MUST be updated in strict-TDD lockstep with implementation changes.
(Previously: Kimi's residual section and output style were covered by unit assertions.)

#### Scenario: Claude goldens capture the residual section

- GIVEN Claude golden fixtures are regenerated after this change
- WHEN the persona section of the golden is inspected
- THEN it matches the residual block, not the prior full section

#### Scenario: Non-output-style adapter goldens remain unchanged

- GIVEN a golden fixture for a non-output-style adapter existed before this change
- WHEN the same fixture is regenerated after this change
- THEN its persona section content is unchanged
```

### Capacidad: sdd-orchestrator-assets

```openspec
## MODIFIED Requirements

### Requirement: Platform-Native Solo Inline Semantics Preservation

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

### Requirement: chained-pr Skill Binding in Chain Strategy Guidance

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

### Requirement: Platform-Accurate Binding Wording for Solo-Inline and Platform-Native Hosts

For solo-inline and platform-native orchestrator hosts (antigravity), the skill-binding instruction MUST phrase "inject under `## Skills to load before work`" in a way that maps to each host's actual phase-context mechanism. The binding MUST NOT reintroduce OpenCode-style persisted-delegation claims that the existing spec forbids for these hosts.
(Previously: the hosts were windsurf, antigravity, kimi and kiro, with a second scenario for kimi and kiro.)

#### Scenario: Solo-inline binding wording is platform-accurate

- GIVEN a solo-inline orchestrator asset (antigravity) is reviewed
- WHEN the skill binding instruction is read
- THEN the binding references the `chained-pr` skill by registry name
- AND the wording maps skill injection to the host's inline or platform-native phase-context mechanism
- AND the wording does not claim persisted OpenCode plugin-backed background delegation

### Requirement: Golden Fixture and Static Assertion Coverage for the Binding

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

### Requirement: Optional Research Lifecycle Parity Across Catalog Agents

All source orchestrator templates MUST render equivalent optional research lifecycle guidance; mappings MAY be shared. Every AgentID returned by `catalog.AllAgents()` MUST render that guidance, including Pi/VS Code Copilot/Trae using the generic runtime family.
(Previously: «All 12 source orchestrator templates» and «all 16 supported IDs», including OpenCode/Kilocode sharing and OpenClaw in the generic family; the requirement also demanded that every rendered variant carry the canonical bounded review execution contract, which belonged to RDD and is retired.)

#### Scenario: Catalog-derived parity test renders all agents

- GIVEN the current catalog's supported AgentIDs
- WHEN each ID is passed through the real orchestrator asset renderer
- THEN every rendered variant contains the optional research lifecycle guidance
- AND no rendered variant contains a `Review Execution Contract` section, its marker, `review/start(target)` or a refuter batch
- AND optional research guidance is rendered identically across the equivalent runtime families

### Requirement: Read-Only Judgment Day Judges

Every Judgment Day judge (`jd-judge-a`, `jd-judge-b`) MUST be structurally read-only for its adapter. OpenCode overlays MUST explicitly disable write, edit, bash, and task for these roles; Claude/Cursor assets MUST use their equivalent read-only capability boundary.
(Previously: Kiro and Kimi assets were also listed; the requirement covered every dedicated reviewer, Judgment Day judge, and refuter, and reviewers and refuters belonged to RDD and are retired.)

#### Scenario: Judge finishes

- GIVEN a Judgment Day judge receives its target
- WHEN its sweep completes
- THEN it returns structured findings and terminates
- AND it cannot modify files or launch another role
```

### Capacidad: visual-decoupling

```openspec
## MODIFIED Requirements

### Requirement: Rutas de instrucciones con nomenclatura Axiom (REQ-09.4)
Los adaptadores de agentes DEBEN utilizar rutas y nombres de archivos que contengan el prefijo o identificador `axiom` en lugar de `gentle-ai`.
(Previously: también fijaba la ruta del steering de Kiro IDE, `~/.kiro/steering/axiom.md`, agente retirado por C-08-d.)

#### Scenario: Archivo de instrucciones en VS Code Copilot
- **DADO** el adaptador de VS Code Copilot
- **CUANDO** se consulta la ruta del archivo de system prompt mediante `SystemPromptFile()`
- **ENTONCES** el nombre del archivo retornado es `axiom.instructions.md` (ubicado en la carpeta de prompts del usuario)

#### Scenario: Directorio de modelos en Pi
- **DADO** el adaptador de Pi
- **CUANDO** se resuelve la configuración de modelos
- **ENTONCES** la ruta de configuración apunta a `~/.pi/axiom/`
```

### Capacidad: tui-ui-parity

```openspec
## MODIFIED Requirements

### Requirement: Diagnóstico de Salud del Sistema (Axiom Doctor) en Web UI (REQ-17.1)
El servidor HTTP local DEBE exponer el endpoint `GET /api/ecosystem/doctor` que recopila y retorna el estado de salud de los agentes compatibles (Claude Code, OpenCode, Codex, Pi), herramientas del sistema (Git, Go, Node.js), permisos del directorio de usuario (`~/.axiom/`) y configuración general. El frontend web DEBE renderizar estos diagnósticos en tarjetas con indicadores visuales claros (verde: saludable, amarillo: advertencia, rojo: error).
(Previously: la lista de agentes incluía Kiro, retirado por C-08-d.)

#### Scenario: Consulta de diagnóstico de salud del sistema
- **DADO** el servidor del Dashboard Web activo
- **CUANDO** se realiza una petición `GET /api/ecosystem/doctor`
- **ENTONCES** el servidor responde con código `200 OK`
- **Y** el cuerpo JSON contiene la lista de diagnósticos con campos `name`, `status` (`ok`, `warning`, `error`), `details` y `recommendation`
```

### Capacidad: tui-spanish-localization

```openspec
## MODIFIED Requirements

### Requirement: Localización de Submenús y Pantallas de Configuración (REQ-16.3)
Las pantallas subordinadas de configuración y gestión deben desplegar sus títulos, opciones y botones de acción en castellano.
(Previously: la pantalla de configuración de modelos ofrecía también `Configurar modelos de Kiro`, agente retirado por C-08-d.)

#### Scenario: Pantalla de Configuración de Modelos
- **DADO QUE** el usuario navega a "Configurar modelos"
- **CUANDO** se muestra la lista de opciones
- **ENTONCES** las opciones deben mostrarse en español (`Configurar modelos de Claude`, `Configurar modelos de OpenCode`, `Configurar modelos de Codex`, y opción "Volver").

#### Scenario: Pantalla de Gestión de Respaldos
- **DADO QUE** el usuario navega a "Gestionar respaldos"
- **CUANDO** se presenta la lista y las acciones disponibles
- **ENTONCES** los botones y estados deben mostrarse en español (`Crear respaldo`, `Restaurar`, `Cancelar`, `Eliminar`, `Volver`, `Sin respaldos registrados`).

#### Scenario: Pantallas de Desinstalación y Herramientas Comunitarias
- **DADO QUE** el usuario accede a "Desinstalación gestionada" o "Herramientas y plugins comunitarios"
- **CUANDO** se despliegan los diálogos y confirmaciones
- **ENTONCES** los textos explicativos y advertencias deben formularse en español peninsular claro y profesional.
```

### Capacidad: installer-picker-navigation

```openspec
## MODIFIED Requirements

### Requirement: Declarative picker flow ordering

The installer's conditional picker chain MUST be navigated forward from a single ordered source of truth, filtered by the per-selection `shouldShow*` predicates. The forward order is `Claude → Codex → SDDMode → ModelPicker → StrictTDD → OpenCodePlugins → DependencyTree`.

`ScreenModelPicker` MUST be included when `SDDMode == Multi`. It asynchronously discovers the active project's effective OpenCode catalog after opening, so no private model-cache file gates navigation. No screen may appear out of this order, and none may be skipped unless its predicate is false. When `Preset == PresetCustom`, `ScreenDependencyTree` precedes the picker sequence (component selector first) instead of being the terminal anchor.
(Previously: the chain included a Kiro picker between Claude and Codex; Kiro is retired by C-08-d.)

#### Scenario: All agents selected, full forward pass

- GIVEN all `shouldShow*` predicates return true and `SDDMode == Multi`
- WHEN the user confirms each picker screen in sequence
- THEN the screens visited in order are Claude → Codex → SDDMode → ModelPicker → StrictTDD → OpenCodePlugins → DependencyTree

#### Scenario: Eligible-only screens are visited

- GIVEN `shouldShowCodex` is false and all other predicates are true
- WHEN the user confirms Claude and continues
- THEN the next screen is SDDMode (Codex is skipped)

#### Scenario: ModelPicker is independent of private cache state

- GIVEN `SDDMode != Multi`
- WHEN the user confirms SDDMode
- THEN the next screen is StrictTDD and ModelPicker is not visited

- GIVEN `SDDMode == Multi` and no private OpenCode cache file exists
- WHEN the user confirms SDDMode
- THEN ModelPicker opens and begins runtime catalog discovery

#### Scenario: PresetCustom ordering inversion

- GIVEN `Preset == PresetCustom`
- WHEN navigation enters the picker chain
- THEN DependencyTree appears first (before Claude/Codex), not last

### Requirement: Symmetric reverse navigation

Backward navigation MUST be the exact reverse of forward navigation and MUST be identical whether triggered by Esc or by selecting the "← Back" row. Both mechanisms read the same flow slice.
(Previously: the Codex back-row scenario used the Kiro picker as the previous screen.)

#### Scenario: SDDMode back returns to Codex when Codex is in the flow

- GIVEN Codex is in the flow (`shouldShowCodex` is true) and the user reached ScreenSDDMode
- WHEN the user presses Esc OR selects "← Back" on ScreenSDDMode
- THEN the current screen is ScreenCodex (Codex is not skipped)

#### Scenario: Codex "← Back" row is not inert

- GIVEN Claude is in the flow (`shouldShowClaude` is true) and the user is on ScreenCodex
- WHEN the user selects "← Back" on ScreenCodex
- THEN the current screen is ScreenClaude

#### Scenario: Full round-trip symmetry for any agent subset

- GIVEN any combination of selected agents
- WHEN the user navigates forward through the whole slice and then backward through the whole slice
- THEN the backward sequence is the exact reverse of the forward sequence and the user returns to the pre-picker screen
```

## Progreso

### T1 · Kilo Code

- Borrado `internal/agents/kilocode` y sus ramas: registro, `capabilitymanifest`, `installcmd`, `validate`, familia OpenCode en `components/sdd` (overlay, perfiles, plugins), `persona`, `permissions`, `mcp`, `engram`, `agentguidance`, `uninstall` (restos globales de Kilo), `workspace_instructions`, `skillregistry` y `config_scan`. Funciones muertas retiradas: `stripOpenCodeNativeFallbackAgents`, `cleanupKilocodeProfileJDPermissions`, `restoreKilocodeManagedAgentTools*`, `usesManagedMCPShape`, `kiloPluginNames`, `kiloConfigHasAxiomEntries`.
- Pruebas: borradas las propias de Kilo; las mixtas se quedan con OpenCode; recuentos de agentes 17 → 16.
- Evidencia: `go vet ./...` limpio; `go test` ok en `agents/...`, `catalog`, `model`, `installcmd`, `components/...` (goldens de OpenCode sin diferencias), `skillregistry`, `system`, `opencode`, `assets`, `tui`, `cli`, `app`, `dashboard`, `update/...`, `state` y `workspace`.
- Commit: `6ead6a2e`.

### T2 · OpenClaw

- Borrado `internal/agents/openclaw`, `components/openclaw_integration_test.go` y `cli/openclaw_orchestration_test.go`; ramas en `mcp` (overlay `mcp.servers` y migración), `engram` (`InjectWithPromptDir`, validación del workspace, sonda), `persona` (`SOUL.md`), `sdd`, `uninstall`, `cli/{run,sync,workspace_agents}` y `skillregistry`; `.openclaw/skills` sale de `PROJECT_MARKERS` de los plugins `skill-registry.ts`.
- Corrige un recuento que T1 dejó mal en `agentguidance` (`markdownSectionAgents`: los excluidos son OpenCode y el router Jinja).
- Evidencia: `go vet ./...` limpio; `go test` ok en `agents/...`, `catalog`, `model`, `installcmd`, `components/...`, `skillregistry`, `system`, `assets`, `tui`, `dashboard`, `app`, `update/...` y `cli`.
- Commit: `09030779`.

### T3 · Hermes

- Borrados `internal/agents/hermes`, `internal/assets/hermes`, la skill `hermes-ephemeral-delegation` (y su fila en `AGENTS.md`) y `components/filemerge/yaml.go`, cuyo único usuario era Hermes; con él sale `model.StrategyMergeIntoYAML` y sus ramas en `mcp` y `engram`. `.hermes/skills` sale de `PROJECT_MARKERS` de los plugins `skill-registry.ts`.
- Las pruebas de lotes de desinstalación que usaban Hermes como agente que falla pasan a Gemini CLI (`settings.json` corrupto en el mismo lote que Claude Code). El recuento de plantillas de orquestador pasa a la constante `orchestratorTemplateCount` (11).
- Evidencia: `go vet ./...` limpio; `go test` ok en `agents/...`, `catalog`, `model`, `installcmd`, `components/...`, `skillregistry`, `system`, `assets`, `tui`, `dashboard`, `app`, `update/...` y `cli`.
- Commit: `be083a09`.

### T4 · Kimi

- Borrados `internal/agents/kimi` y `internal/assets/kimi` (hub `KIMI.md`, agentes YAML, estilos y personas). Kimi era el único usuario de `model.StrategyJinjaModules`: sale la estrategia y sus ramas en `persona`, `sdd`, `engram`, `agentguidance` (entrega por módulo Jinja y `ErrUnloadableGuidance`), `cli/{run,sync}` (paso del hub y limpieza heredada del módulo de reglas) y `update/upgrade`; también el instalador `uv` de `installcmd` y el overlay de Context7 de Kimi.
- Pruebas: borradas las propias de Kimi y del canal Jinja, con las instantáneas históricas de sus estilos; recuentos: 13 agentes, 10 plantillas de orquestador.
- Evidencia: `go vet ./...` limpio; `go test` ok en `agents/...`, `catalog`, `model`, `installcmd`, `components/...`, `skillregistry`, `system`, `assets`, `tui`, `dashboard`, `app`, `update/...` y `cli` (con `-timeout 25m`: tarda unos 11 minutos). `TestPiCodeGraphPendingProbePreservesConfiguredFiles` falló una vez por tiempo de espera del sondeo MCP y pasa al repetirlo (intermitente, ajeno al cambio).
- Commit: `d3f0d413`.

### T5 · Windsurf

- Borrados `internal/agents/windsurf`, `internal/assets/windsurf` (orquestador y `workflows/`), sus 8 goldens y `assets/windsurf_workflow_test.go`. Windsurf era el único agente con workflows nativos: salen la vía de inyección de workflows de `sdd` (`workflowInjector`, `findProjectRoot` y la entrada de preflight de `sdd-new`), su retirada en `uninstall` y la línea de la pantalla de desinstalación; también la sección compartida «Delegated Verification Gate (Reduced Form)», que solo usaba su orquestador. El esquema del manifiesto de capacidades conserva el campo `workflows`.
- Pruebas: las de `cli` que usaban Windsurf como agente sin modo workspace pasan a Trae; las de Engram con `StrategyMCPConfigFile`, a Cursor; la de preflight que falla antes de escribir se queda con el caso de Codex. `e2e`: retiradas las pruebas de Windsurf y las limpiezas de `.codeium` y `.kimi` en `lib.sh`. Recuentos: 12 agentes, 9 plantillas de orquestador.
- Evidencia: `go vet ./...` limpio; `go test` ok en `agents/...`, `catalog`, `model`, `installcmd`, `components/...` (goldens), `skillregistry`, `system`, `assets`, `tui`, `dashboard`, `app`, `update/...` y `cli`; `bash -n e2e/e2e_test.sh` ok.
- Commit: `f737cec1`.

### T6 · Kiro IDE

- Borrados `internal/agents/kiro`, `internal/assets/kiro`, `model/kiro_model*.go`, el selector de modelos de Kiro de la TUI (`ScreenKiroModelPicker`; el menú de modelos queda en Claude, OpenCode y Codex), sus 18 goldens y `sdd/kiro_hook_drift_test.go`. Kiro era el único usuario de `model.StrategySteeringFile`, del hook de skill-registry por fichero (`SkillRegistryHookPath`), del resolvedor de alias de modelo, del marcador `{{AXIOM_SKILLS_DIR_WIN}}` y de la sección compartida «Skill Resolution Feedback (Kiro and Windsurf)»: salen todos. `KiroModelAssignments` sale de `state`, `model.Selection`, `SyncOverrides`, `cli` y `app` (un `state.json` antiguo con esa clave sigue cargando: la clave se ignora). La desinstalación deja de conocer el steering de Kiro (P20).
- Assets compartidos: menciones de Kiro en los jueces de Judgment Day, `project-baseline-convention.md` y `PROJECT_MARKERS`; `sdd-init/references/init-details.md` pierde las rutas de los seis agentes. `docs/manual-de-inicio.md` deja de nombrar a Kiro (lo fija `docs_contract_test`).
- Pruebas: las de prompts heredados pasan a Cursor; las de MCP global y CodeGraph, a `~/.cursor/mcp.json`. Recuentos: 11 agentes, 8 plantillas de orquestador.
- Evidencia: `go vet ./...` limpio; `go test` ok en `agents/...`, `catalog`, `model`, `installcmd`, `components/...` (goldens), `skillregistry`, `system`, `assets`, `tui/...`, `dashboard`, `app`, `update/...`, `state`, `workspace` y `semantic`. `cli` completo no se repite aquí (decisión del usuario: ir más rápido); lo cubre la CI de la PR.
- Commit: `53115740`.

### T7 · Lector tolerante

- `state.InstallState.UnmarshalJSON` separa de `installed_agents` los agentes que el catálogo no conoce y los deja en `IgnoredAgents` (no se persiste): global y de proyecto, también la lectura del estado heredado. Todos los consumidores (sync, install, TUI, uninstall, doctor) ven solo agentes conocidos y la siguiente escritura del estado ya no los incluye. Es genérico: no nombra los agentes retirados.
- `sync` avisa por cada agente ignorado (`noteIgnoredStateAgents`) y sincroniza el resto.
- Pruebas nuevas: `state.TestReadIgnoresAgentsTheCatalogNoLongerKnows` y `cli.TestRunSyncWithRetiredAgentInStateWarnsAndSyncsTheRest`.
- Evidencia: `go vet` limpio; `go test` ok en `state` y en `cli -run` (las dos nuevas y las de agentes persistidos en workspace).
- Commit: `ceae0caa`. Corrección posterior `40b7fb5d`: `state` no puede importar `catalog` (ciclo en las pruebas de `catalog`); la lista de agentes admitidos pasa a `model.IsKnownAgentID`, y `catalog.TestModelKnownAgentsMatchCatalog` la mantiene igual que la del catálogo.
- CI de la PR: `TestProjectAdaptersCheckLooksAtTheProjectRoot` usaba un agente inexistente que el lector tolerante ya descarta; pasa a usar `codex`.

### T8 · Docs, e2e, specs vivas y criterio

- Docs (ruta delegada en un escritor, por paralelismo con las deltas de specs: superficies disjuntas): `docs/kiro.md` borrado; `docs/agents.md`, `README.md`, `docs/{usage,platforms,intended-usage,components,quickstart,trigger-rules,architecture,manual-de-inicio}.md`, `docs/codebase/interfaces.md`, `PRD.md` y `PRD-AGENT-BUILDER.md` sin los seis agentes. Revisado por el padre: recuento de portadores de `docs/components.md` (15 → 10) y comprobación `rg`.
- Notas para la persona usuaria: `docs/releases/notas-retirada-agentes.md` (qué limpiar a mano, P20).
- `## Spec viva` integrada con `axiom archive integrate --odd retirar-agentes`: 6 capacidades y `openspec/INDEX.md`. A mano: `openspec/config.yaml` y la tabla «Test Enforcement» de `installer-picker-navigation`.
- `odd/tasks/kiro-workspace-parity.md` marcado como superado y `odd/tasks/codex-sync-sin-temas-y-rollback.md` como parcialmente superado (Windsurf).
- Criterio `rg -il "kiro|windsurf|kimi|hermes|openclaw|kilocode|kilo code"` fuera de los históricos (`docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `docs/releases`, `odd/tasks`): solo quedan `CONTRIBUTORS.md` (atribución histórica), `docs/upstream-absorption-ledger.md` (registro histórico), las notas `(Previously: ...)` de las specs vivas y las pruebas del lector tolerante (`state`, `catalog`, `cli`).
- Evidencia: `go vet ./...` limpio; `go test` ok en `workspace` (contratos de docs), `state`, `catalog`, `model`, `components` (goldens) y `cli -run TestRunSyncWithRetiredAgentInStateWarnsAndSyncsTheRest`. Batería completa: CI de la PR.

## Siguiente paso

PR con `scripts/axiom-worktree.ps1 pr retirar-agentes -Lane odd` (etiqueta `type:refactor`), CI, merge commit y `done`.
