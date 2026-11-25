# Delta for sdd-orchestrator-assets

## RENAMED Requirements

### Requirement: Bounded Review Contract Parity Across Catalog Agents → Optional Research Lifecycle Parity Across Catalog Agents

(Reason: la parte de contrato de revisión acotada pertenece a RDD y se retira (C-01-e, P20); solo queda la paridad de la guía de investigación opcional.)
(Migration: los pines de texto que buscaban `review/start(target)`, el lote de refutadores o `Review Execution Contract` en los orquestadores se eliminan; los pines de la guía de investigación opcional se conservan.)

### Requirement: Read-Only Reviewer and Refuter Roles → Read-Only Judgment Day Judges

(Reason: los revisores de lente y el refutador `review-*` pertenecen a RDD y se retiran (C-01-e, P20); solo los jueces de Judgment Day siguen siendo roles de solo lectura.)
(Migration: ninguna para los jueces `jd-judge-a` y `jd-judge-b`; el resto de agentes de revisión se limpian a mano según la sección de migración manual de la propuesta.)

## MODIFIED Requirements

### Requirement: Optional Research Lifecycle Parity Across Catalog Agents

All 12 source orchestrator templates MUST render equivalent optional research lifecycle guidance; mappings MAY be shared. Every AgentID returned by `catalog.AllAgents()` MUST render that guidance, and all 16 supported IDs MUST be covered, including OpenCode/Kilocode sharing and Pi/VS Code Copilot/OpenClaw/Trae using the generic runtime family.
(Previously: the requirement also demanded that every rendered variant carry the canonical bounded review execution contract; that contract belonged to RDD and is retired.)

#### Scenario: Catalog-derived parity test renders all agents

- GIVEN the current catalog contains 16 supported AgentIDs
- WHEN each ID is passed through the real orchestrator asset renderer
- THEN every rendered variant contains the optional research lifecycle guidance
- AND no rendered variant contains a `Review Execution Contract` section, its marker, `review/start(target)` or a refuter batch
- AND optional research guidance is rendered identically across the equivalent runtime families

### Requirement: Read-Only Judgment Day Judges

Every Judgment Day judge (`jd-judge-a`, `jd-judge-b`) MUST be structurally read-only for its adapter. OpenCode overlays MUST explicitly disable write, edit, bash, and task for these roles; Claude/Kiro/Kimi/Cursor assets MUST use their equivalent read-only capability boundary.
(Previously: the requirement covered every dedicated reviewer, Judgment Day judge, and refuter; reviewers and refuters belonged to RDD and are retired.)

#### Scenario: Judge finishes

- GIVEN a Judgment Day judge receives its target
- WHEN its sweep completes
- THEN it returns structured findings and terminates
- AND it cannot modify files or launch another role

### Requirement: La Entrega es `single-pr` por Defecto sin Preguntar, `chained` Solo a Petición y se Leen los Valores Heredados mediante Alias (REQ-32.3)

La doctrina de los orquestadores SDD de todos los runtimes (plantillas, copias en línea, reservas que compone el instalador y la convención compartida `sdd-phase-common.md`) DEBE declarar que la estrategia de entrega por defecto es `single-pr` (una PR con un commit por unidad de trabajo), que NO se pregunta, y que `chained` (con `stacked-to-main` o `feature-branch-chain`) solo se activa cuando la persona lo pide durante la sesión; si lo pide sin indicar la estrategia de cadena, el orquestador formula una única pregunta por la estrategia de cadena. El dominio canónico es `single-pr | chained`. Al **leer** un valor de entrega ya existente (un `tasks.md`, un prompt o una caché heredados), el orquestador y las fases DEBEN aplicar estos alias de lectura: `ask-on-risk` y `exception-ok` → `single-pr`; `auto-chain` → `chained`. Cualquier otro valor DEBE seguir siendo inválido y reportarse sin mapearlo. NO se escribe ningún valor nuevo con vocabulario heredado, y no se reescribe ningún artefacto existente.

`chained` se mantiene solo en la caché de la sesión del orquestador (como las demás preferencias de sesión, REQ-28.2): NO se escribe en `kickoff.yaml`, en `gates.yaml` ni como campo de `tasks.md`; en una sesión nueva la entrega vuelve a `single-pr`. El bloque de pre-vuelo expresa el valor por defecto de la sesión: un `delivery_strategy: chained` reenviado por el orquestador con su `chain_strategy` prevalece sobre él, y `sdd-apply` solo lo usa para elegir la rama destino y nunca se detiene por ello.

Ninguna doctrina ni skill DEBE contener un presupuesto numérico de revisión ni una parada por carga de revisión. En concreto: la sección `Review Workload Guard` y su bloque de decisión no existen en ningún runtime; `sdd-tasks` conserva solo una previsión cualitativa (tamaño estimado y unidades de trabajo) sin los campos `400-line budget risk`, `Decision needed before apply` ni `Chained PRs recommended`; y `sdd-apply` no contiene el paso de decisión de carga de trabajo ni la regla `workload-decision-required`. Si `delivery.split_hint_lines` está fijada, se aplica el aviso único no bloqueante de REQ-32.2 de `workspace-topology`. Quedan fuera de este requisito, por no ser umbrales de entrega, los literales de migración de prompts antiguos de `internal/components/sdd/inject.go`, el umbral de barrido de `jd-judge-a` y `jd-judge-b` de judgment-day.
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

## ADDED Requirements

### Requirement: La Doctrina Proyectada No Contiene la Prosa de RDD (REQ-46.3)

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

### Requirement: `axiom sync` No Instala Agentes, Plugins, Skill ni Hook de RDD (REQ-46.4)

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

## REMOVED Requirements

### Requirement: Shared Contract Expansion

(Reason: la expansión del contrato canónico `skills/_shared/review-ledger-contract.md` en los agentes de revisión nativos pertenece a RDD; ambos se retiran (C-01-e, P20). Los artefactos `review-ledger` de Judgment Day no son ese contrato y se conservan.)
(Migration: ninguna; Judgment Day mantiene su propio contrato en `skills/judgment-day`.)
