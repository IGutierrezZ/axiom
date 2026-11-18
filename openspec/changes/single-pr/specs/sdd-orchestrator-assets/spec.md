# Delta para sdd-orchestrator-assets

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta fija la entrega `single-pr` por defecto sin preguntar y conserva la capacidad de encadenar PRs solo a petición de la persona (decisión P10): las estrategias de cadena canónicas `stacked-to-main` y `feature-branch-chain` siguen apareciendo textualmente en las doce plantillas, pero el reenvío de `chain_strategy` a las fases pasa a ser condicional a que la persona haya pedido `chained`. Se retira el Review Workload Guard y su presupuesto numérico. Los requisitos nuevos usan el grupo de numeración REQ-32 (el resto de esta capacidad no está numerado).

## MODIFIED Requirements

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

## ADDED Requirements

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
