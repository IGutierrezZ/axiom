# 01 · Poda de fricción

Análisis de la sesión 1, cerrado con el usuario el 2026-11-11. Las rutas se han comprobado contra `main` en `5f6b05b4`. Detalle en Engram: `analisis-backlog/poda-friccion` y sus subtemas `/multirepo-interactivo`, `/roles-monorepo` y `/rdd-prosa`.

## Origen

Temas del backlog original, en palabras del usuario, resumidas:

1. **RDD, ¿lo queremos en Axiom?** «RDD es solo de ODD y ODD no es el por defecto de Axiom; además tenemos sdd-verify y judgment-day; por si sobra y lo podemos quitar».
2. **Quitar Serena.** «Con la búsqueda semántica que tiene Axiom y CodeGraph creo que es suficiente; quitamos fricción de instalaciones y mantenimiento, y solo se mantienen Axiom, Engram y CodeGraph».
3. **Multirepo siempre interactivo.** En multirepo, la pregunta automatic/interactive debería ser siempre interactiva.
4. **Persistencia siempre both.** La pregunta engram/openspec/both debe desaparecer, porque en Axiom siempre será both, y el hook no debe saltar con error.
5. **Hook de preflight que rechaza por el orden de las opciones.** «Pasa mucho: el hook rechazó el preflight porque las opciones no estaban en el orden canónico». Si el usuario ya ha contestado, tiene que valer. El usuario pide eliminar esta fricción como principio de diseño, no solo para estas preguntas.

Durante el análisis surgió un tema más, a propuesta del usuario:

6. **Roles solo en multirepo.** «En un monorepo no hay opción de roles; podemos incluir esto también y quitamos problemas; los roles son para multirepo».

## Cambios resultantes

### C-01-a · El hook del preflight valida por etiqueta, no por posición

- **Objetivo.** Hoy el hook rechaza un preflight bien contestado si el agente presenta las opciones en otro orden, y el rechazo es genérico, así que el modelo tiene que adivinar la causa. El usuario lo sufre a menudo. El texto «el hook rechazó porque Interactive tenía que ir antes que Automatic» lo redacta el modelo, no el hook. Resultado esperado: una respuesta válida vale con cualquier orden de opciones, y si hay rechazo, dice exactamente qué corregir.
- **Decisiones.**
  - (Usuario) «Si el usuario ya ha contestado, tiene que valer.» Se elimina la fricción como principio de diseño, no solo para estas preguntas.
  - (Análisis, aceptado por el usuario) Cada pregunta se identifica por su marcador `Axiom SDD preflight N/M:` y cada respuesta por su etiqueta, por igualdad de conjuntos: se mapea etiqueta → token, nunca índice → token. El orden de las opciones deja de importar.
  - (Análisis) El rechazo nombra la causa concreta: pregunta que falta, etiqueta fuera del dominio, multiselección o respuesta ambigua. Hoy esa causa se descarta y el usuario solo ve el mensaje genérico de `internal/cli/sdd_preflight_hook.go:160`.
  - (Análisis) Se conserva lo que el hook protege de verdad: la autoridad confirmada por el padre. Un hijo no puede despachar (`:149`), un texto del modelo no crea autoridad (`:152`) y el payload debe enlazar la transcripción (`:155`).
  - (Análisis) La causa probable de la reordenación es la guía de `AskUserQuestion` de Claude Code, que pide poner primero la opción recomendada y añadirle «(Recommended)». Por eso la comparación de etiquetas debe tolerar ese sufijo o rechazarlo con un mensaje que lo nombre; se decide al implantar con una prueba. Es una inferencia sin reproducir.
  - (Análisis) Principio de diseño que se extiende al resto del producto: validar el significado, no la presentación; derivar antes de preguntar; todo rechazo dice cómo corregirlo.
- **Fuera de alcance.** Quitar preguntas del preflight (lo hace C-01-b). Cambiar el contrato de autoridad del hook.
- **Carril propuesto.** ODD. No hay ambigüedad de diseño, es un validador acotado con sus pruebas y no cambia ninguna spec viva: `openspec/specs/sdd-preflight-configuration/spec.md` no fija el orden de las opciones.
- **Tamaño.** S.
- **Depende de / bloquea a.**
  - Bloquea a C-01-b.
  - Bloquea a C-06-b, el HTML explicativo de la sección de UI (`explainer_html: ask|always|never`, que se pregunta al iniciar el cambio): la consolidación lo deja dependiendo del hook por etiqueta.
  - Lo ideal es implantarlo antes que cualquier cambio que añada o quite preguntas del preflight, incluida la estrategia de PR de C-02-c.
- **Ficheros y paquetes que toca.**
  - `internal/cli/sdd_preflight_hook.go:351-411`: `resolveSDDPreflightHookBlock`, con `expectedLabels` en `:362`, la comparación posicional en `:370-373` y el mapeo por índice en `:399-401`.
  - `:160` y `resolveSDDPreflightTranscriptAuthority`: hoy descarta el error del resolutor con `continue`.
  - `internal/cli/sdd_preflight_hook_test.go`: `TestSDDPreflightHookDeniesReorderedCanonicalLabels`, que pasa a aceptar el orden invertido.
  - Referencia de paridad: el plugin de OpenCode ya canoniza el orden (`internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts` y su copia en `plugins/`) [COMPARTIDO].
- **Criterio de terminado.**
  - Un preflight con las opciones en cualquier orden y las etiquetas canónicas produce el mismo bloque `## SDD Session Preflight` que en el orden canónico.
  - Una etiqueta fuera del dominio produce un `deny` cuyo texto nombra la pregunta y la etiqueta.
  - Siguen denegando los casos de autoridad (hijo, texto del modelo, payload sin transcripción).
  - `go test ./internal/cli/ -run SDDPreflight` en verde.
- **Riesgos y avisos.**
  - El texto proyectado del preflight (`internal/components/sdd/session_preflight.go:16`) dice «Keep options in the exact order below». Conviene suavizarlo en C-01-b, o aquí si no rompe goldens; el hook deja de exigirlo de todos modos.

### C-01-b · Preflight y sellado derivados de la topología: hybrid fijo, multirepo checkpointed y monorepo solo fullstack

- **Objetivo.** Quitar del preflight las preguntas cuya respuesta Axiom puede deducir y hacer que la topología decida el ritmo y los roles. Así desaparecen la pregunta de persistencia, la de ritmo en multirepo y el rechazo «auto con roles» al sellar.
- **Decisiones.**
  - (Usuario) «La pregunta engram / openspec / both debe desaparecer porque en Axiom siempre será both.» El almacén es siempre `hybrid` y no se pregunta.
  - (Usuario, 2026-11-11) El ritmo interactivo forzado se decide **por topología** («Por topología»), no por roles. En `multirepo` el preflight no pregunta el ritmo: deriva `interactive` y el cambio se sella `checkpointed`.
  - (Usuario, 2026-11-11) «En un monorepo no hay opción de roles, podemos incluir esto también y quitamos problemas; los roles son para multirepo», confirmado con «Sí, solo fullstack». Un `monorepo-embedded` solo admite el rol `fullstack`. Los roles especializados son exclusivos de `multirepo`.
  - Regla resultante (análisis, aceptada por el usuario):
    - **multirepo**: roles, siempre `checkpointed`, sin pregunta de ritmo.
    - **monorepo**: solo `fullstack`, con `continuous` o `checkpointed` a elección y sin ningún rechazo posible.
  - (Usuario, al aceptar) Pérdida asumida: en monorepo no hay `tasks.<rol>.md`, ni parada `role-apply` por rol, ni ramas por rol.
  - (Análisis) La regla de `#249` (`continuous` solo con el roster `[fullstack]`, `internal/kickoff/schema.go:122-143`, aplicada en `internal/kickoff/args.go:293`) se cumple por construcción en monorepo, y en multirepo queda inalcanzable porque el ritmo ya no se pregunta. Se conserva como red de seguridad.
  - (Análisis) Un multirepo debe rechazar `--execution-style continuous` al sellar, con un mensaje que diga que en multirepo el estilo es siempre `checkpointed`.
- **Fuera de alcance.**
  - La estrategia de PR y el umbral de 400 líneas: los decide C-02-c, que mantiene la estrategia de PR en el preflight con `single-pr` por defecto y retira el 400. Tras C-01-b y C-02-c, el preflight queda así: en multirepo, solo la estrategia de PR; en monorepo, el ritmo y la estrategia de PR. Nunca queda vacío mientras C-02-c conserve ese grupo.
  - La pantalla de ajustes del proyecto de la sección de UI, que ya excluye `artifact_store` y los roles en monorepo.
- **Carril propuesto.** SDD. Cambia el comportamiento contractual del sellado y del alta, y toca specs vivas: `change-branch-lifecycle`, `multi-project-hub` y probablemente `sdd-preflight-configuration` y `sdd-lane-kickoff`. Además necesita una migración de los monorepos con roles. Las deltas y la verificación por requisito reducen el riesgo de dejar specs incoherentes.
- **Tamaño.** M.
- **Depende de / bloquea a.**
  - Depende de C-01-a.
  - Bloquea a C-02-c, que toca el mismo bloque del preflight y del hook. Van uno tras otro, nunca en paralelo.
  - Bloquea a C-06-c y, por inferencia, a C-06-b, C-06-e y C-07-d.
  - No depende de RDD.
- **Ficheros y paquetes que toca.**
  - Texto del preflight: `internal/components/sdd/session_preflight.go:16` (`sddSessionPreflightBody`: grupos, marcadores `N/3`, mapeos y «Keep options in the exact order»), `:58-69` (variante sin UI nativa) y `:153-193` (validación de la proyección, que incluye el rechazo heredado `Both -> both` en `:187`).
  - Hook: `internal/cli/sdd_preflight_hook.go:362-401`. Hoy exige exactamente 3 preguntas (`:363`); pasa a admitir un número variable según la topología.
  - Asset del orquestador [COMPARTIDO]: `internal/assets/claude/sdd-orchestrator-workflow.md:73` («If the user doesn't specify, default to **Automatic**») y `:97-99` (Artifact Store Mode: «default to `engram` when Engram is available; otherwise use `none`»), con sus equivalentes en los demás runtimes.
  - Plugin de OpenCode [COMPARTIDO]: `internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts` y `plugins/`.
  - Sellado: `internal/kickoff/args.go:140-300` (`ParseSealArgs`, `--from-session-pace` en `:193`, la falta de estilo en `:264` y `checkContinuousRoster` en `:293`) e `internal/kickoff/schema.go:122-143`.
  - Validador del workspace: `internal/workspace/validator.go:99-116`. Hoy valida los repositorios de cada rol sin mirar la topología; tiene que rechazar en `monorepo-embedded` cualquier rol distinto de `fullstack`.
  - Alta:
    - `internal/hub/init_validate.go:174-195`: el perfil `knowledge`/`spec-only` fuerza `multirepo` con el rol `knowledge`; los roles del formulario o los detectados se aplican a cualquier topología; si no hay ninguno, se usa `fullstack`.
    - `internal/hub/detector.go:245-275`: detecta `core`, `web` y `qa`.
    - `internal/hub/init.go:356-387`.
    - Formulario de alta: `internal/dashboard/assets/js/core/projectform.js` (`buildRoles`). En monorepo no debe ofrecer roles.
  - Almacén: `internal/workspace/artifact_store.go:22` (`DeclaredArtifactStore`). Hay que decidir si un `artifact_store` declarado distinto de `hybrid` se ignora con aviso o se rechaza. Hoy lo usan `internal/workspace/validator.go:271-285`, `internal/sddpublish/publish.go:67` e `internal/cli/sdd_gate_handoff.go:55,177`.
  - Specs vivas:
    - `openspec/specs/change-branch-lifecycle/spec.md`: los escenarios de monorepo con roles `core` y `qa` en `:268`, `:300` y `:549`, y la tabla de `:36`.
    - `openspec/specs/multi-project-hub/spec.md:91`: el alta monorepo genera los roles `core` y `qa`.
    - `openspec/specs/sdd-preflight-configuration/spec.md`.
  - Goldens [COMPARTIDO]: `testdata/golden/*` que proyectan el preflight (hoy uno contiene el marcador `Axiom SDD preflight 1/3`), más los de las plantillas del orquestador que cambien.
- **Criterio de terminado.**
  - En un `multirepo`, el preflight no muestra la pregunta de ritmo ni la de almacén. El bloque derivado dice `Pace: interactive` y `Artifact store: hybrid`, y el kickoff se sella `checkpointed`.
  - `kickoff seal --execution-style continuous` en multirepo se rechaza con un mensaje que nombra la topología.
  - En un `monorepo-embedded`, el preflight solo pregunta el ritmo (más lo que deje la sección de entrega Git), el almacén es `hybrid` y el sellado solo admite `fullstack`.
  - `axiom workspace validate` rechaza un monorepo con roles distintos de `fullstack`, con un mensaje que explica cómo migrar.
  - El alta monorepo (CLI y dashboard) solo escribe `fullstack`.
  - Las specs vivas no contienen escenarios de monorepo con roles especializados.
  - `go test ./internal/kickoff/... ./internal/cli/... ./internal/workspace/... ./internal/hub/... ./internal/components/...` y `go test ./internal/components/ -run TestGolden` en verde.
- **Riesgos y avisos.**
  - Migración: hay monorepos con varios roles. Este repositorio no (solo `fullstack`, `axiom.yaml:20-27`), pero el detector de alta crea `core`, `web` y `qa` en cualquier topología. Hace falta un aviso con la orden de migración, o una migración explícita; no se debe romper la carga.
  - El perfil `knowledge`/`spec-only` ya fuerza `multirepo` con el rol `knowledge` (`internal/hub/init_validate.go:175-181`), así que no choca con la regla. Hay que comprobarlo con una prueba.
  - El hook pasa de exigir exactamente 3 preguntas a exigir las que correspondan a la topología: en multirepo, Artifacts y Pace se derivan y no se preguntan. C-02-c parte de este hook ya variable, y el texto de la sección 02 que dice «el hook exige exactamente 3» queda obsoleto tras este cambio.
  - El esquema de marcadores (`N/3` hoy) debe admitir un número de preguntas que cambie por topología y preguntas que se añadan después, como `explainer_html` de C-06-b. Conviene un marcador por nombre de grupo, o un `N/M` con `M` variable, para que añadir o quitar un grupo no rompa a los demás.
  - Las paradas de tres salidas no están disponibles con `engram` o `none`. Con `hybrid` fijo siempre lo están.

### C-01-c · RDD apagado de fábrica y sin posibilidad de activarlo

- **Objetivo.** RDD gasta tokens (contrato de revisión, traspaso de defectos y verificación delegada proyectados en cada orquestador) y añade fricción (selector en la instalación, la TUI y el alta, y un hook Stop), aunque está apagado por defecto. El usuario quiere retirarlo, pero como es un cambio XL, de momento solo busca que no moleste ni gaste tokens.
- **Decisiones.**
  - (Usuario, 2026-11-11) «Mi idea es quitar RDD, pero como es un cambio grande quiero simplemente que no moleste y no gaste tokens, por lo que si lo más fácil es no habilitar la posibilidad de habilitar el RDD me vale.» Esto sustituye la propuesta previa de carga diferida (resguardo de pocas líneas más un asset que se lee solo si RDD está activo), que el usuario había aceptado y luego cambió.
  - (Análisis, aceptado) Corrección de la premisa: RDD no es «solo de ODD». Es la revisión del trabajo que no es SDD (ODD y trabajo directo), y SDD lo prohíbe de forma explícita (`internal/assets/skills/_shared/sdd-orchestrator-sections.md:53`, `:65` y `:92`; `internal/kickoff/rdd_boundary_test.go`).
  - (Análisis, aceptado) Alcance:
    1. El modo efectivo es siempre off. `axiom review mode enable`, en cualquier ámbito, lo rechaza con un mensaje claro, y las preferencias on ya guardadas (global, de proyecto en el hub o del clon) se ignoran.
    2. Desaparece el selector de RDD en la instalación, la TUI y el dashboard (alta de proyecto).
    3. No se proyecta ninguna prosa RDD en los orquestadores ni en la guía de agentes.
    4. No se instalan los agentes `review-*` ni el hook `axiom review stop-hook`, que en Claude se registra en `Stop` y en `SessionStart` (`internal/components/sdd/inject.go:2355-2420`).
    5. El código Go queda dormido hasta su retirada completa (C-01-e).
  - (Usuario, comunicado a la sesión de entrega Git) La revisión commit a commit no se apoya en RDD ni en `axiom review assess`.
- **Fuera de alcance.**
  - Borrar `internal/reviewtransaction`, `internal/cli/review*` y los paquetes satélite (lo hace C-01-e).
  - Retirar `contracts/review-*`, los journeys de `bench/` y los pasos de CI.
- **Carril propuesto.** ODD. Las decisiones están cerradas y no hay ambigüedad de diseño. El impacto en las specs `openspec/specs/rdd-*` y de review (23 ficheros) se limita a marcar la capacidad como suspendida, y puede integrarse con `axiom archive integrate --odd`. Si al implantar resulta que hay que reescribir requisitos de esas specs, pasar a SDD.
- **Tamaño.** S-M. El grueso son goldens y pruebas de texto.
- **Depende de / bloquea a.**
  - No depende de nada.
  - Bloquea a C-01-e.
  - Conviene que vaya antes que C-02-c: C-02-c no toca las referencias RDD al 400 de `routing.go:129` y espera que las retire este cambio.
  - Bloquea, por inferencia, a C-05-d: el autofetch necesita su propio SessionStart, no el de RDD.
  - C-06-e (ajustes del proyecto) ya excluye RDD, en coherencia con este cambio.
- **Ficheros y paquetes que toca.**
  - Modo:
    - `internal/reviewtransaction/rdd_mode.go:340` (`ResolveRDDMode`) y `:385` (`ResolveRDDModeStrict`). Fuente por defecto en `:101-102`.
    - `internal/cli/review_mode.go:60`, `:104` y `:149`: los verbos `enable|disable|status|inherit`, donde `enable` pasa a rechazar.
    - `internal/hub/rddpref.go`.
  - Prosa del orquestador [COMPARTIDO]:
    - Con que `rendersReviewLifecycle` (`internal/components/sdd/boundedreview.go:240`) devuelva `false`, ya se retira el `Review Execution Contract`, porque `:212-215` llama a `removeBoundedReviewSection`.
    - Hay que retirar además, de `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, las secciones `Delegated Verification Gate (MANDATORY)` (`:52-62`), `Delegated Verification Gate (Reduced Form)` (`:64-73`), `Axiom Provider Defect Handoff (MANDATORY)` (`:143-171`) y `Review Execution Contract` (`:448-450`), junto con sus marcadores `{{GENTLE_AI_SDD_SECTION:…}}` en las plantillas `internal/assets/*/sdd-orchestrator.md` (13 runtimes; en Claude, `internal/assets/claude/sdd-orchestrator.md:16-18`, `:49-51` y `:57-59`).
  - Guía de agentes [COMPARTIDO]: `internal/components/agentguidance/routing.go`:
    - `:129`, la viñeta «Preserve existing native risk selection…», que define el candidato de revisión nativa por commit o por rebanada de PR con `axiom review assess --committed-only` y el presupuesto de unas 400 líneas. Se conserva solo «run applicable functional checks per task»; C-02-c cuenta con que se retire aquí.
    - `:131`, el párrafo «When RDD is enabled…».
    - `:152-166`, la sección «Receipt-driven development is user-owned» y su comentario.
    - Lo que haya de RDD en `delegation.go` y `approval.go`.
  - Hook en Claude: `internal/components/sdd/inject.go:1868` y `:2355-2420`. `axiom review stop-hook` se registra en `Stop` y en `SessionStart` (`internal/cli/review_stop_hook.go`).
  - Agentes e instalación:
    - `internal/assets/*/agents/review-{risk,resilience,readability,reliability,refuter}.*` (25 ficheros) y su expansión para OpenCode en `internal/components/sdd/inject.go`.
    - Skill `rdd-defect-workflow` en `internal/catalog/skills.go` e `internal/components/skills/presets.go`.
  - UI:
    - TUI: `internal/tui/screens/review_mode.go` e `internal/tui/screens/install_review_mode.go`.
    - Dashboard: `internal/dashboard/service.go:909-926` (`ReviewMode` en el alta) y `internal/dashboard/types.go`.
  - Goldens [COMPARTIDO]: 17 en `testdata/golden` mencionan RDD (`rg -l -i "rdd|receipt-driven"`), y 6 contienen `Review Execution Contract`.
  - Pruebas de contrato que fijan la prosa: `internal/assets/blocking_prompt_contract_test.go`, `internal/components/sdd/{bounded_review_contract,orchestrator_composition,orchestrator_shared_sections,renderer_invariants,review_ledger_contract}_test.go` e `internal/cli/review_integration_contract_guard_test.go`.
- **Criterio de terminado.**
  - `axiom review mode enable --scope global|project|clone` sale con error y un mensaje que explica que RDD no está disponible.
  - `axiom review mode status` informa off aunque haya preferencias on guardadas.
  - `axiom sync --scope workspace` en un proyecto limpio no escribe agentes `review-*`, no registra `axiom review stop-hook` (ni en `Stop` ni en `SessionStart`) y no deja ninguna de las secciones RDD en los orquestadores. Se comprueba con `rg -i "receipt-driven|Review Execution Contract|Provider Defect Handoff"` sobre lo generado.
  - La TUI y el dashboard no muestran el selector.
  - Goldens regenerados y revisados por diff. `go test ./internal/components/... ./internal/assets/... ./internal/cli/... ./internal/tui/... ./internal/dashboard/...` en verde.
- **Riesgos y avisos.**
  - `Delegated Verification Gate` mezcla reglas generales (cuándo hace falta un verificador según el riesgo y el tamaño del escritor) con reglas RDD. Hay que conservar la parte general y reescribirla sin `axiom review assess`, o decidir que sobra.
  - `Axiom Provider Defect Handoff` depende del envelope `gentle-ai.review-integration.consent/v3`. Su rama de defecto del proveedor podría aplicarse a otros envelopes de Axiom; comprobar que nada ajeno a RDD la usa antes de retirarla.
  - En los proyectos ya sincronizados quedan los agentes `review-*` y las entradas `Stop` y `SessionStart` del hook. `axiom sync` debe retirarlas como gestionadas, o hay que documentar su borrado.
  - Mientras esas entradas existan en proyectos antiguos, `axiom review stop-hook` debe seguir terminando sin error, como no-op.
  - Tratar la compatibilidad del JSON persistido (`rdd_mode` en `internal/state/state.go`).

### C-01-d · Quitar Serena

- **Objetivo.** Quitar Serena del producto para quedarse con Axiom, Engram y CodeGraph.
- **Decisiones.**
  - (Usuario) Se quita Serena: «solo se mantienen Axiom, Engram y CodeGraph».
  - (Análisis, aceptado) Correcciones de la premisa:
    - Axiom no instala Serena. Solo la detecta (`internal/semantic/detector.go:147-227`), la prioriza en `auto` (`internal/semantic/service.go:73-97`) y la escribe en `.mcp.json` del perfil `knowledge` (`internal/hub/mcp.go:17-20`, llamada desde `internal/hub/init.go:291`). Quitarla ahorra menos fricción de la que se esperaba.
    - No existe «ugrep». El motor propio es un análisis AST nativo de Go (`internal/semantic/engine.go`, con `go/parser` y `go/ast`), sin embeddings.
  - (Análisis, aceptado) No queda hueco funcional:
    - La edición simbólica de Serena nunca se integró en Axiom.
    - `find_referencing_symbols` lo cubre CodeGraph de forma aproximada (`callers`/`impact`).
    - Los símbolos que no son de Go los cubre CodeGraph.
    - Las memorias de Serena las sustituye Engram.
- **Fuera de alcance.** Instalar CodeGraph de serie: hoy es una herramienta de comunidad opcional (`internal/model/types.go`, `CommunityToolCodeGraph`).
- **Carril propuesto.** ODD con integración en la spec viva (`axiom archive integrate --odd`). Las decisiones están cerradas. Toca cuatro specs vivas, pero solo para retirar menciones. Si se prefiere que esas specs pasen por deltas formales, SDD; en este caso es una cuestión de política, no de ambigüedad.
- **Tamaño.** S-M (unos 17-20 ficheros).
- **Depende de / bloquea a.** Es independiente y puede ir en paralelo con C-01-a, C-01-b o C-01-c, siempre que no toque los mismos goldens a la vez.
- **Ficheros y paquetes que toca.**
  - Go:
    - `internal/semantic/types.go:7-8` (`ConnectorSerena`) y `:66-69` (`SerenaAvailable`, `SerenaInstalled`, `SerenaConfigured`).
    - `internal/semantic/detector.go:11` y `:147-227`.
    - `internal/semantic/service.go:68-70`, `:73-97` (selector, con Serena primero en `auto`) y `:276` (mensaje de reindexado).
    - `internal/hub/mcp.go:17-20` y `:29-30` (`DefaultKnowledgeMCPServers`, `InjectKnowledgeMCPServers`).
    - `cmd/axiom/main.go:148` y `:1597`.
    - `internal/knowledge/types.go:59`.
  - Dashboard: `internal/dashboard/assets/js/screens/semantic.js:14`, `:17`, `:31` y `:75`.
  - Skills: `skills/knowledge-query/SKILL.md:3`, `:26` y `:61`, y `skills/knowledge-crawl/SKILL.md:3` y `:22-23`. Hay que reescribirlas sobre CodeGraph y el motor nativo.
  - Índice de skills: `AGENTS.md:59-60`.
  - Configuración: `axiom.example.yaml:149` (`semantic_analysis: "serena"`).
  - Pruebas: `internal/semantic/semantic_test.go`, `internal/hub/init_knowledge_test.go` e `internal/dashboard/knowledge_assets_test.go`.
  - Specs vivas: `openspec/specs/semantic-code/spec.md` (18 menciones), `openspec/specs/knowledge-profile-init/spec.md`, `openspec/specs/knowledge-query-engine/spec.md`, `openspec/specs/knowledge-sweep-command/spec.md` y `openspec/INDEX.md:76` y `:2553-2555`.
  - Docs: `docs/manual-de-inicio.md` y `docs/architecture/AXIOM_ENTERPRISE_VISION.md`.
  - Coincidencias falsas de «serena» que no hay que tocar: `internal/reviewtransaction/*_test.go` (vienen de «se-RENAMED») y el historial archivado (`docs/archive`, `openspec/changes/archive`, `docs/audits`).
- **Criterio de terminado.**
  - `rg -i serena` sin resultados fuera de los históricos archivados y de las coincidencias falsas.
  - `axiom semantic status` no informa de Serena.
  - En `auto` gana CodeGraph si está disponible y, si no, `native-ast`.
  - Un `axiom.yaml` con `semantic_analysis: serena` sigue cargando, con un aviso de valor obsoleto y caída a CodeGraph o `native-ast`.
  - El alta del perfil `knowledge` escribe solo `engram` y `codegraph` en `.mcp.json`, y retira una entrada `serena` gestionada que ya existiera.
  - `go test ./internal/semantic/... ./internal/hub/... ./internal/dashboard/...` en verde.
- **Riesgos y avisos.**
  - `InjectKnowledgeMCPServers` solo añade claves y nunca borra. Los `.mcp.json` existentes conservan `serena` si no se añade su retirada. Al retirarla, borrar solo la entrada exacta que escribió Axiom (`command: serena`, `args: [serve]`).
  - `docs/manual-de-inicio.md` documenta `semantic_analysis: "enabled"`, un valor que el código no reconoce. Conviene corregirlo de paso.

### C-01-e · Retirada completa de RDD (adelantada a la ola A2)

Corregido el 2026-11-24 con el inventario de la sección 08 (`main` en `d580368d`). El motivo del adelanto está en la sección 08, «C-01-e adelantada».

- **Objetivo.** Eliminar del producto el código, los contratos, los assets, las specs y la CI de RDD una vez apagado.
- **Decisiones.**
  - (Usuario) «Mi idea es quitar RDD», cuando sea posible, después de C-01-c.
  - (Usuario, 2026-11-24) Se adelanta: «Sí, redacta la 08», sobre la propuesta de la sección 08 con C-01-e adelantada.
  - (Análisis) Las primitivas que usan paquetes ajenos a RDD las saca antes C-08-c. Son `AcquireAuthorityFileLock`, `ReplaceFileAtomic`, `SyncReviewDirectory` y `PublishFileNoReplace`, más `SnapshotBuilder` y las demás que lista la 08. Este cambio parte de un `reviewtransaction` que ya solo importa código de RDD o usos clasificados como «parte RDD».
  - (Análisis) Se quedan como lápidas, según P20:
    - `components/sdd/rdd_retire.go`, que retira los agentes `review-*` y el hook de instalaciones antiguas.
    - `axiom review stop-hook` como no-op mientras existan proyectos con esa entrada.
  - (Análisis) `internal/consentenvelope` no se borra: lo usa el consentimiento de edición de SDD. Solo se quita su uso desde `cli/review_consent_contract.go:8`.
  - (Análisis) `bench/` se trata según P21.
- **Fuera de alcance.**
  - `internal/handoff` (traspaso de SDD) e `internal/recoverytrace` (lo retira C-08-a): ninguno es de RDD.
  - `internal/releasepolicy` e `internal/releaseprovenance`, que son de la distribución.
- **Carril propuesto.** SDD. Es XL y retira capacidades publicadas (`contracts/review-integration` y `contracts/review-provider-contract`) y 21 specs vivas (`rdd-*` y `review-findings-ledger`). Necesita un plan por fases con verificación.
- **Tamaño.** XL. Unas 61.300 líneas de Go de producción y unas 100.000 de pruebas. Además, 11.175 de contratos, unas 3.500 de specs, unas 3.200 de docs y `bench/`.
- **Depende de / bloquea a.**
  - Depende de C-01-c, de C-08-c, de P20 y de P21. Depende también de que la revisión commit a commit de C-02-c y C-02-d no use RDD (ya acordado).
  - Bloquea a C-08-d, C-08-g, C-08-h y C-08-i.
  - Retira `internal/reviewtransaction/risk.go:22` (`LargeChangeLines = 400`), que C-02-c deja expresamente para la sección 01.
- **Ficheros y paquetes que toca.**
  - Paquetes de RDD (confirmados por imports):
    - `internal/reviewtransaction` (36.077 + 38.307).
    - `internal/cli/review*` (21.666 + 50.879), más 12 ficheros de `cli` con RDD (`sdd_gate_commit_review*`, `sdd_no_review_test`, `sdd_status_*review*` y `codex_review_provider_model_test`).
    - `internal/reviewerprovider`, `internal/providercontractbundle` y `internal/providercontractbundlecmd`.
    - `internal/rddharness`, que va el último: tiene 28 importadores, entre ellos los `main_test.go` de seis paquetes.
  - Despachador: `cmd/axiom/main.go:173` y `:432`.
  - Parte RDD de paquetes ajenos:
    - `components/sdd`: `boundedreview.go`, `review_*` y `rdd_*`, salvo la lápida. Cableado en `inject.go`, `orchestrator.go` y `profiles.go`.
    - TUI: `review_mode`, `install_review_mode` y `review_store_reset`, con sus referencias en `tui/model.go` y `app/app.go:1146-1166`.
    - Dashboard: `service_review_mode.go` y `js/core/reviewmode.js`.
    - Hub: `hub/rddpref.go` y sus referencias en `hub/init.go` y `hub/types.go`.
    - `internal/state/state.go` (`rdd_mode`), con lectura tolerante.
    - Las guardas `sddstatus/review_offer_absence_guard_test.go` y `kickoff/rdd_boundary_test.go`.
    - Los usos clasificados como «parte RDD» por C-08-c.
  - Assets:
    - Los 25 agentes `review-*`.
    - `review-ledger-contract*.md`, enlazado desde `judgment-day/SKILL.md`.
    - La skill `rdd-defect-workflow`.
    - El plugin `opencode-review-transport.ts` (v1 y v2).
    - La prosa RDD residual de los orquestadores, `sdd-overlay-*.json`, `judgment-day`, `sdd-apply` y `sdd-verify`.
  - Contratos: `contracts/review-*`.
  - CI [COMPARTIDO]:
    - `ci.yml`: `:271-299` y `:555-623` (etiqueta, bench y el estrés `-count=100`) y el paso de `opencode-v2-contracts`, en la parte que sea RDD.
    - `windows-full-suite.yml`: `:143-211` y `:257`, los shards `cli-review-*` y `reviewtransaction-*`.
    - `release.yml`: `:67`, `:83` y `:148`.
    - `promote-stable-rc.yml`: `:73-109`.
  - Scripts:
    - `scripts/test-review-*.sh`, `opencode_v2_loopback.py`, `test-opencode-v2-*` y `test_opencode_v2_host_test.py`.
    - Revisar uno a uno, porque son mixtos: `scripts/crosslane` y `e2e/organicruntime`.
  - `bench/`: según P21.
  - Specs: `openspec/specs/rdd-*` (20) y `review-findings-ledger`. `sdd-block-review-gates` y `sdd-independent-review` son de SDD: solo pierden sus menciones a RDD.
  - Docs: `review-integration.md`, `review-authority-threat-model.md`, `architecture/rdd-*` y `audits/*rdd*`.
  - `.deadcode-baseline.txt`.
- **Criterio de terminado.**
  - `go vet ./...` y `go test ./...` en verde.
  - `rg -l 'reviewtransaction|reviewerprovider|rddharness'` sin resultados fuera de los históricos y de las lápidas de P20.
  - La release no publica el bundle del contrato de revisión, y `update/release_security_test.go:44` está ajustado.
  - Las specs RDD están retiradas.
  - Los shards de `windows-full-suite.yml` cubren todos los tests que quedan.
- **Riesgos y avisos.**
  - Los nombres no bastan para saber qué es de RDD: la pertenencia se confirma por importaciones. `handoff`, `recoverytrace` y `consentenvelope` no lo son, o no del todo.
  - El hook `Stop` instalado en proyectos antiguos llama a `axiom review stop-hook`. Hay que mantener un no-op o retirarlo antes con `sync` (P20).
  - Retirar el bundle cambia los assets de la release. Lo comprueban `release.yml` y `release_security_test.go`.
  - `scripts/crosslane`, `e2e/organicruntime` y `opencode-v2-contracts` cubren también OpenCode, la guarda de ramas y los carriles de agentes. No se pierde esa cobertura.

## Descartado

- **Carga diferida de la prosa RDD** (resguardo de pocas líneas más un asset que se lee si `axiom review mode status` da on). La había aceptado el usuario y la sustituyó por C-01-c («no habilitar la posibilidad de habilitar el RDD me vale»), porque es más simple.
- **Renderizar la prosa RDD solo si está activo al hacer `axiom sync`**, con re-sync al cambiar el modo. Descartado por análisis: se desfasa cuando el modo cambia por otra vía y ata el cambio de modo al sync. Además, `axiom sync` escribe por defecto en el workspace (`internal/cli/sync.go:141`; `docs/usage.md:90`), así que no era necesario para el ámbito global.
- **Forzar el ritmo interactivo por roles** en vez de por topología. Descartado por el usuario («Por topología»).
- **Retirar RDD ya, de golpe.** Descartado por el usuario por tamaño: primero C-01-c, después C-01-e.
- **Un hook de persistencia aparte.** No existe. El «error del hook» de persistencia es el mismo validador por orden de C-01-a.

## Correcciones a la consolidación

Contra la observación de Engram `analisis-backlog/consolidacion`, revisión del 2026-11-13:

- Dice «Un solo cambio M, que va primero: el hook de preflight valida por etiqueta…». La agrupación era una recomendación del análisis, no una decisión del usuario. Aquí se separa en C-01-a (hook, S, ODD) y C-01-b (preflight, sellado y roles, M, SDD). C-01-a es independiente, desbloquea antes el HTML explicativo de la UI y reduce el riesgo del cambio grande. Las decisiones de fondo no cambian.
- Dice «al alta (que hoy crea core/qa)». En `main` el alta crea roles en tres casos: los del formulario, los detectados por el stack (`core`, `web` y `qa`, en `internal/hub/detector.go:263`) o, si no hay ninguno, `fullstack` (`internal/hub/init_validate.go:186-195`). La que dice literalmente `core` y `qa` es la spec `openspec/specs/multi-project-hub/spec.md:91`. El efecto en C-01-b es el mismo: el alta en monorepo debe escribir solo `fullstack`.
- La regla «continuous solo con [fullstack]» ya está en `main` (PR #249, `internal/kickoff/schema.go:122-143`), no en vuelo como se dijo durante el análisis.
- La línea base ajena se retiró (`1166efca`). El choque entre `artifact_store: engram|none` y las declaraciones del proyecto ahora solo afecta a `technical_context.maintained` (`internal/workspace/validator.go:271-285`), no a `baseline`.
