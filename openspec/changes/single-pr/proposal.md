# Propuesta: una sola PR por defecto y retirada del umbral de 400 líneas (C-02-c)

## Intención

La doctrina, el pre-vuelo y las skills de Axiom imponen hoy un presupuesto de revisión de 400 líneas cambiadas por PR y hacen que la entrega gire alrededor de él:

- El pre-vuelo SDD pregunta la estrategia de PR («Ask me / Single PR / Auto») y el bloque derivado arrastra la línea `Review policy: 400 changed lines`.
- `sdd-tasks` emite un «Review Workload Forecast» con `400-line budget risk`, `Decision needed before apply` y `Chained PRs recommended`; `sdd-apply` se detiene (`workload-decision-required`) si el trabajo supera el presupuesto sin cadena ni `size:exception`.
- La doctrina ODD (`routing.go`) usa «unas 400 líneas» como heurística por tarea y como disparador de la estrategia de cadena (`ask-on-risk`, `auto-chain`, `single-pr`, `exception-ok`).
- `chained-pr`, `work-unit-commits`, `branch-pr` y `comment-writer` repiten el número.

El usuario decidió que la unidad de revisión es la tarea (un commit por tarea), no un conteo de líneas: «La PR mejor unificada, salvo que el usuario pida otra cosa» y «si quiero un control de "es mucho cambio, ¿quieres dividirlo?", vale, pero no definir lo de las 400». El cambio retira el número de la doctrina, del pre-vuelo y de las skills, fija `single-pr` como entrega por defecto sin preguntar, conserva la cadena solo a petición y añade un aviso opcional y no bloqueante configurable en `axiom.yaml`.

Éxito: el pre-vuelo no pregunta la estrategia de PR; la doctrina de todos los runtimes dice `single-pr` por defecto y `chained` solo si la persona lo pide; `sdd-apply` no se detiene por carga de revisión; un cambio con valores heredados (`ask-on-risk`, `auto-chain`, `exception-ok`) se sigue leyendo; y con `delivery.split_hint_lines` fijada el orquestador avisa una vez de un cambio grande sin bloquear.

## Decisiones cerradas (no se reabren)

| Id | Decisión | Origen |
|----|----------|--------|
| P10 | La pregunta de estrategia de PR sale del pre-vuelo. La entrega es `single-pr` sin preguntar; `chained` solo cuando la persona lo pide en el chat, y conserva `stacked-to-main` y `feature-branch-chain`. `ask-on-risk`, `auto-chain` y `exception-ok` dejan de ofrecerse. | Usuario, 2026-11-17 |
| P5 | Nueva clave `delivery.split_hint_lines` en `axiom.yaml`, desactivada por defecto. Si se fija, el orquestador avisa una vez de que el cambio es grande y nunca bloquea. C-06-e la incluirá en su catálogo. | Usuario, 2026-11-17 |
| P13 | El check de 400 del CI (`.github/workflows/pr-check.yml`) y la plantilla de PR los retira C-02-e, no este cambio. | Usuario, 2026-11-17 |
| Criterio ampliado | Además de `internal/reviewtransaction/risk.go:22`, se admiten como residuo del `rg "400"`: (1) el umbral de barrido de judgment-day «Full-4R … >400 changed lines» en los assets de `jd-judge-a`/`jd-judge-b`; (2) las lentes RDD `review-*` y la skill `rdd-defect-workflow` (las retira C-01-e); (3) los literales de migración «D1 400 lines» / «D1 400 líneas» de `internal/components/sdd/inject.go`, que detectan texto antiguo instalado. | Usuario, 2026-11-18 |
| Q1 | Dominio canónico `single-pr \| chained`. Alias de lectura: `ask-on-risk` y `exception-ok` → `single-pr`; `auto-chain` → `chained`. Cualquier otro valor sigue siendo inválido y se informa sin mapearlo. | Diseño delegado (exploración) |
| Q2 | Se retira la línea `Review policy` del bloque derivado (de 6 a 5 líneas). El grupo `pr-strategy` pasa a heredado: reconocido, tolerado e ignorado, igual que `artifacts`. | Diseño delegado |
| Q3 | `sdd-tasks` conserva solo una previsión cualitativa (tamaño estimado y unidades de trabajo), sin campos de riesgo ni de decisión y sin el número 400. | Diseño delegado |
| Q4 | Los literales de migración de `inject.go` se conservan. | Diseño delegado |
| Q5 | `skills/branch-pr` y `comment-writer` (sus dos copias) se reescriben en este cambio. | Diseño delegado |
| Q6 | `chained` se guarda solo en la caché de la sesión del orquestador; no se escribe en `kickoff.yaml`, `gates.yaml` ni como campo de `tasks.md`. En una sesión nueva la entrega vuelve a `single-pr`. | Diseño delegado |
| Reconciliación | Las dos copias de `work-unit-commits` se reconcilian tomando como canónica la embebida (`internal/assets/skills/work-unit-commits/SKILL.md`). | Diseño delegado |

## Alcance

### Dentro del alcance

1. **Clave `delivery.split_hint_lines` en `axiom.yaml`.**
   - Nuevo bloque opcional de primer nivel `delivery:` con lista cerrada de claves; `split_hint_lines` es un entero ≥ 1. Su ausencia (o la del bloque) desactiva el aviso.
   - Validación estricta con el mismo canal de error que `workflow:` y `git:` (clave desconocida o repetida, no mapa, valor no entero, cero, negativo o nulo), tanto en `axiom workspace validate` como en la carga.
   - `axiom init --force` preserva el bloque igual que los demás bloques declarados.
   - Documentación de la clave en `docs/manual-de-inicio.md` (y su test de contrato de documentación).
2. **Pre-vuelo sin estrategia de PR.**
   - `monorepo-embedded` pregunta solo `pace`; `multirepo` queda sin grupos y usa la ruta de cuestionario suprimido de REQ-28.18.
   - `pr-strategy` se conserva como grupo heredado sin `RequiredIn`: una respuesta antigua («Ask me», «Auto») se ignora y el bloque dice siempre `Delivery strategy: single-pr`.
   - El bloque derivado pasa a 5 líneas, sin `Review policy`.
   - Paridad en el hook Go (`internal/cli/sdd_preflight_hook.go`), en el texto proyectado (`internal/components/sdd/session_preflight.go`, incluida la sintaxis de respuesta sin UI `Pace: <label>`) y en las dos generaciones del plugin de OpenCode, con registro de los digests sustituidos en `opencode_plugin_digests.go`.
   - Los nueve comandos de OpenCode `sdd-*.md` dejan de exigir «chained PR strategy and review budget» en el pre-vuelo.
3. **Doctrina ODD** (`internal/components/agentguidance/routing.go`).
   - La heurística por tarea pierde el número y conserva lo útil: la tarea es el comportamiento coherente más pequeño con sus tests y su documentación; no se borran espacios, comentarios ni tests para ahorrar líneas; no se parte artificialmente. Desaparece «Existing PR size gates remain unchanged».
   - El bullet «Delivery follows work units» se reescribe de forma autocontenida: `single-pr` por defecto (una PR con un commit por tarea); `chained` (`stacked-to-main` o `feature-branch-chain`) solo si la persona lo pide, con una única pregunta por la estrategia de cadena si no la indicó; aviso único si `delivery.split_hint_lines` está fijada y la previsión la supera; `work-unit-commits` siempre y `chained-pr` solo al encadenar.
4. **Orquestadores y convención compartida.**
   - `Delivery Strategy`: no se pregunta; `single-pr`; `chained` solo a petición; alias de lectura de Q1.
   - `Chain Strategy`: se conserva, activada por la petición de la persona (no por `ask-on-risk` ni `auto-chain`).
   - `Review Workload Guard`: se retira el bloque de decisión, el presupuesto de 400 y las cuatro ramas; el reenvío de `delivery_strategy`/`chain_strategy` a `sdd-tasks` y `sdd-apply` queda en las secciones de entrega y cadena.
   - Ficheros: `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `_shared/sdd-phase-common.md` §E, `internal/assets/claude/sdd-orchestrator-workflow.md` y los `internal/assets/<agente>/sdd-orchestrator.md`. `Review Workload Guard` va por plantilla solo en qwen, gemini y generic, y en línea en los otros nueve orquestadores.
   - El cuerpo de reserva de `### Delivery Strategy` en `internal/components/sdd/orchestrator.go` (`composeFallbackSessionPreflight`) se reescribe conservando los encabezados de los que depende.
   - Se ajusta el ratchet de deriva (`orchestrator_drift_ratchet_test.go`) al número real de variantes que resulte.
5. **Skills.**
   - `chained-pr`, `SKILL.md` y `references/chaining-details.md`: se activa solo cuando la persona pide encadenar, apilar o rebanar, o con `delivery_strategy: chained`; sin número; su `description` se proyecta a `AGENTS.md`.
   - `work-unit-commits`: un commit por unidad de trabajo dentro de una sola PR y promoción a cadena solo a petición; se conserva «no code-golf». La copia de `skills/` se reconcilia con la embebida.
   - `sdd-tasks`: previsión cualitativa (Q3).
   - `sdd-apply`: se elimina el paso 2a y la regla `workload-decision-required`; si el prompt trae `chain_strategy`, la sigue para elegir la rama destino; se conserva «no minificar».
   - `comment-writer` (`skills/` y `internal/assets/skills/`): se sustituye el ejemplo de tono que cita el presupuesto.
   - `skills/branch-pr`: se retira el presupuesto y la regla de `size:exception` como política propia.
6. **Specs vivas y documentación.** Deltas de las capacidades listadas abajo; `docs/components.md` (Chained PR «a petición»); `AGENTS.md` (descripción de `chained-pr`).
7. **Tests y goldens.**
   - Reescribir `delivery_strategy_vocabulary_test.go` para el dominio nuevo con alias, conservando los tests de migración de vocabulario retirado y el de valor no reconocido.
   - Actualizar los tests de pre-vuelo (hook, topología, `session_preflight`, tabla TS, arnés del plugin con node), composición, invariantes del renderizador, `planning_detail_test.go` y `routing_test.go`, que gana aserciones de la doctrina nueva.
   - Regenerar los goldens afectados con `-update`.

### Fuera del alcance

- **C-02-d**: aviso y parada por commit, modos de parada (P2) y validación implícita al pedir la PR. Este cambio no los anticipa; el bullet de entrega de `routing.go` queda autocontenido para que C-02-d lo extienda sin reescribirlo.
- **C-01-e**: `internal/reviewtransaction/risk.go:22` (`LargeChangeLines`), las lentes `review-*`, `rdd-defect-workflow` (sus dos copias y su test), las aserciones 4R de `inject_test.go` («Large PR, hot path, or >400 changed lines») y `openspec/specs/review-findings-ledger/spec.md:35`.
- **C-02-e (P13)**: el check de tamaño de `.github/workflows/pr-check.yml`, `.github/PULL_REQUEST_TEMPLATE.md`, la memoria local `entrega-una-pr-size-exception.md` y la preferencia `preference/delivery-strategy` de Engram.
- El umbral de barrido de judgment-day en `jd-judge-a`/`jd-judge-b` y sus proyecciones (overlays de OpenCode, goldens): no es un umbral de entrega.
- `size:exception` como etiqueta de política protegida en `issue-creation` (gobierno de etiquetas).
- Proyectar `delivery` en `axiom sdd status --json`: queda como opción para C-06-e si la necesita.
- Retirar la capacidad de encadenar PRs: se mantiene a petición.
- `docs/usage.md` e `internal/app/help.go`: no contienen el umbral; el plan los listaba por error.
- Reescribir `tasks.md`, `kickoff.yaml` o artefactos archivados con valores heredados: se siguen leyendo mediante los alias.

## Capacidades

### Nuevas capacidades

Ninguna.

### Capacidades modificadas

- `sdd-preflight-configuration`:
  - REQ-28.14: retira «la política de revisión de 400 líneas sigue sin preguntarse» y su escenario.
  - REQ-28.15: la tabla vigente pasa a ser `monorepo-embedded` → `pace` y `multirepo` → ninguno; el bloque de 5 líneas no incluye política de revisión.
  - REQ-28.16: `pr-strategy` se suma a `artifacts` como grupo heredado reconocido y tolerado.
  - REQ-28.17 y REQ-28.18: `multirepo` entra siempre por la ruta sin grupos; una respuesta heredada de estrategia se ignora.
  - REQ-28.20: la doctrina dice `single-pr` fijo.
  - REQ-28.21: paridad del plugin.
  - REQ-28.22: «seis líneas» pasa a cinco.
- `sdd-orchestrator-assets`:
  - MODIFIED: los requisitos de cadena («Chain Strategy Guidance Parity», «Chain Strategy Propagation to Downstream Phases», sus variantes no-Claude, «chained-pr Skill Binding…», «Inline Chain Strategy Summary Retained» y «Cursor Chain Strategy Section Parity») conservan ambas estrategias canónicas en todas las plantillas, pero el reenvío de `chain_strategy` pasa a ser condicional a `chained` pedido por la persona.
  - ADDED: entrega `single-pr` por defecto sin pregunta, alias de lectura (Q1), sin presupuesto numérico ni parada por carga en ningún runtime, y `chained` solo en la caché de sesión (Q6).
- `organic-agent-trigger-rules`, ADDED: la doctrina ODD de entrega es `single-pr` por defecto, `chained` solo a petición, sin número en la heurística por tarea y con aviso único no bloqueante cuando `delivery.split_hint_lines` está fijada.
- `workspace-topology`:
  - MODIFIED REQ-2.1: la lista de bloques opcionales de primer nivel admite `delivery`.
  - ADDED: requisito del bloque `delivery:` (clave `split_hint_lines`, dominio, ausencia = desactivado, validación estricta, un binario anterior lo ignora y `axiom init --force` lo preserva). Esta spec es la dueña del esquema de `axiom.yaml`, como `git:` (REQ-25.26) y `workflow:` (REQ-27.1).

La numeración de los requisitos nuevos la asigna `sdd-spec`.

## Enfoque

Alternativa A de la exploración: reescritura completa en un solo cambio, con commits por unidad de trabajo.

1. **`delivery:` en `internal/workspace`** siguiendo el patrón de `workflow.go`/`sdd.go`:
   - `delivery.go` con `DeliveryConfig{SplitHintLines int}`, `deliveryKeys` y `decodeDelivery`.
   - Campos `Delivery`/`DeliveryErr` en `Declarations` y una rama en `readDeclarationBlocks`.
   - Integración en `joinLoadErrors`, la proyección en `WorkspaceConfig` y la lista de bloques preservados de `internal/hub/init.go`.
   - Tests espejo de `workflow_test.go`.
   - El agente lee la clave directamente de `axiom.yaml`, como ya hace con `workspace.topology` y `technical_context`; no se amplía el status.
2. **Pre-vuelo.**
   - `sddPreflightGroups`: `pr-strategy` sin `RequiredIn`.
   - `sddPreflightBlockFields`: `Delivery strategy` sin `GroupID`, con `Allowed: single-pr`.
   - Se elimina `sddPreflightBlockReview`, ajustando `sddPreflightBuildBlock` y `validSDDPreflightHookBlock`.
   - Misma transformación en ambos plugins TS, digests históricos y texto de `session_preflight.go`.
3. **Doctrina ODD** en `routing.go` (dos bullets) con aserciones en `routing_test.go`.
4. **Orquestadores**: editar `sdd-orchestrator-sections.md` (fuente de las plantillas), luego cada copia en línea y el texto de reserva de `orchestrator.go`. `sdd-design` decide si converge copias en línea a la plantilla al tocarlas y fija el ratchet resultante.
5. **Skills** según el alcance, reconciliando `work-unit-commits`.
6. **Aviso de cambio grande.** Si `delivery.split_hint_lines` está fijada, el orquestador compara con ella la previsión de líneas: en SDD, la estimación cualitativa de `sdd-tasks`; en ODD, la previsión del documento de la feature. Si la supera, emite un único aviso de que el cambio es grande y de que puede pedir PRs encadenadas, y continúa sin esperar respuesta. Sin la clave no hay aviso ni número.
7. **Specs, documentación, tests y goldens**, regenerados con `go test ./internal/components/ -run <Test> -update` y revisando el diff. Todo en proceso con `go test`, sin compilar binarios sueltos.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/workspace/` (`delivery.go`, `declarations.go`, `loader.go`, `types.go`) y tests | Nuevo/Modificado | Bloque `delivery:` y su validación |
| `internal/hub/init.go` | Modificado | `axiom init --force` preserva `delivery:` |
| `docs/manual-de-inicio.md` y su test de contrato | Modificado | Documentación de la clave |
| `internal/cli/sdd_preflight_hook.go` y tests | Modificado | `pr-strategy` heredado; bloque de 5 líneas |
| `internal/components/sdd/session_preflight.go` y tests | Modificado | Texto del cuestionario sin estrategia de PR |
| `internal/assets/opencode/plugins/` y `plugins-v2/sdd-task-result-artifacts.ts` | Modificado | Paridad con el hook |
| `internal/components/sdd/opencode_plugin_digests.go` y `opencode_runtime_test.go` | Modificado | Digests de las revisiones sustituidas |
| `internal/assets/opencode/commands/sdd-*.md` (9) | Modificado | Contenido exigido del pre-vuelo |
| `internal/components/agentguidance/routing.go` y `routing_test.go` | Modificado | Doctrina ODD de entrega |
| `internal/components/sdd/orchestrator.go` | Modificado | Cuerpo de reserva de `### Delivery Strategy` |
| `internal/components/sdd/inject.go` | Sin cambio | Literales de migración conservados (Q4) |
| `internal/assets/skills/_shared/{sdd-orchestrator-sections.md,sdd-phase-common.md}` | Modificado | Entrega, cadena; retirada del guard de carga |
| `internal/assets/claude/sdd-orchestrator-workflow.md` y `internal/assets/<agente>/sdd-orchestrator.md` | Modificado | Misma doctrina en todos los runtimes |
| `internal/assets/skills/{chained-pr,work-unit-commits,sdd-tasks,sdd-apply,comment-writer}` | Modificado | Sin número; cadena a petición; sin parada por carga |
| `skills/{work-unit-commits,branch-pr,comment-writer}/SKILL.md` | Modificado | Copias del proyecto alineadas |
| `AGENTS.md`, `docs/components.md` | Modificado | Descripción de `chained-pr` y de la capacidad |
| Tests de `internal/components/sdd`, `internal/assets`, `internal/sddtaskresult`, `internal/components` | Modificado | Vocabulario, invariantes, composición, ratchet |
| `testdata/golden/*` (unos 11) | Regenerado | Proyecciones de la doctrina |
| `openspec/specs/{sdd-preflight-configuration,sdd-orchestrator-assets,organic-agent-trigger-rules,workspace-topology}` | Delta | Según «Capacidades» |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| El bloque de pre-vuelo, confirmado por el runtime, dice `Delivery strategy: single-pr` aunque la persona haya pedido `chained` en el chat; una fase podría tomar el bloque como autoridad | Media | `sdd-design` fija la precedencia: el bloque expresa el valor por defecto de la sesión y un `delivery_strategy: chained` con su `chain_strategy`, reenviado por el orquestador, prevalece. `sdd-apply` solo lo usa para elegir la rama destino y nunca se detiene por ello |
| Un `tasks.md` antiguo con `auto-chain` o `ask-on-risk` choca con Q6 (la sesión nueva vuelve a `single-pr`) | Baja | Los alias se aplican al leer valores existentes de cambios ya en curso; no se escribe ningún valor nuevo. El diseño lo explicita |
| Ratchet de deriva al editar en línea los orquestadores | Media | Editar las 13 superficies en el mismo commit y ajustar el ratchet a la cifra real; converger a la plantilla donde sea barato |
| `composeFallbackSessionPreflight` hace `panic` si cambian los encabezados `### Delivery Strategy` / `### Chain Strategy` | Media | Se conservan los encabezados; los tests de composición de los runtimes de reserva lo cubren |
| Olvidar los digests históricos de los plugins | Media | Añadir el sha256 de la revisión sustituida en v1 y v2 y subir el conteo del test |
| Transcripciones con `pr-strategy` respondido generan un bloque con `ask-on-risk` | Baja | Prueba explícita: una respuesta «Ask me» o «Auto» se ignora y el bloque dice `single-pr` |
| `skills/branch-pr` deja de citar la regla de `size:exception` mientras el check del CI sigue activo hasta C-02-e | Media (transitoria) | La skill remite a la política de etiquetas del repositorio sin citar el número; C-02-e retira el check justo después |
| Previsión cualitativa (Q3) frente a la comparación numérica del aviso | Baja | La previsión conserva una estimación aproximada de líneas, sin umbral, riesgo ni decisión; solo se compara si la clave está fijada |
| Colisión con C-02-d (mismos bullets de `routing.go` y `work-unit-commits`) y con C-06-b/C-06-e (pre-vuelo y catálogo de ajustes) | Media | Bullet autocontenido; esquema de `delivery:` estable y cerrado |
| Tamaño de la PR (supera ampliamente 400 líneas) | Alta | Entrega en una PR con `size:exception` (el check del CI sigue activo hasta C-02-e) y commits por unidad de trabajo |

## Plan de reversión

- Revertir la PR única (o sus commits por unidad de trabajo, en orden inverso) restaura el grupo `pr-strategy`, la línea `Review policy`, el guard de carga, la doctrina con 400 y los goldens anteriores.
- Tras revertir, ejecutar `axiom sync` para reproyectar orquestadores, skills y plugins; los digests añadidos se retiran con la reversión.
- Compatibilidad durante un despliegue parcial: el hook nuevo ignora respuestas `pr-strategy` de transcripciones antiguas; si se revierte el hook, `pr-strategy` vuelve a ser requerido y el texto proyectado antiguo lo vuelve a preguntar.
- `delivery:` es opcional: un binario revertido ignora el bloque, porque las claves de primer nivel no reconocidas se ignoran (`internal/workspace/loader.go:68-71`, REQ-2.1); el aviso simplemente deja de emitirse. Ningún proyecto la declara hoy.
- No hay migraciones de datos: no se reescribe ningún `axiom.yaml`, `kickoff.yaml` ni `tasks.md`.

## Dependencias

- C-01-a, C-01-b y C-01-c fusionados en `main`. El pre-vuelo por etiqueta y por topología, con su cuestionario vacío (REQ-28.18), es la base. `routing.go:129` ya quedó sin 400.
- C-02-a fusionado (base del worktree `cb92574f`).
- Bloquea a C-02-d. C-02-e se implanta con este cambio o justo después. C-06-b y C-06-e van después y consumen el pre-vuelo reducido y el esquema de `delivery:`.

## Previsión de tamaño

- **Tamaño:** L, en el límite de XL.
- **Ficheros:** unos 85-90 tocados.
  - Unos 7 de Go de `workspace`/`hub` y 2 de documentación.
  - Unos 14 de pre-vuelo: hook, texto, 2 plugins, digests y 9 comandos.
  - 2 de doctrina ODD.
  - Unos 16 de orquestadores, convención compartida y reserva.
  - Unos 10 de skills.
  - Unos 20 de tests.
  - 4 deltas de spec.
  - Unos 11 goldens regenerados.
- **Líneas de autor:** del orden de 1.500 a 2.500 (adiciones más eliminaciones, sin goldens), con predominio de eliminaciones en doctrina y skills.
- **Entrega:** una sola PR con `size:exception` y commits convencionales por unidad de trabajo, sin atribución a IA.

## Criterios de éxito

- [ ] `rg -n "400" internal/assets internal/components internal/cli skills` no devuelve ninguna referencia al umbral de entrega. Solo quedan como residuo admitido:
  - el umbral de barrido «Full-4R … >400 changed lines» de `jd-judge-a`/`jd-judge-b` y sus proyecciones;
  - las lentes RDD `review-*`, `rdd-defect-workflow` (sus dos copias y su test) y las aserciones 4R de `inject_test.go`, que retira C-01-e;
  - los literales de migración «D1 400 lines» / «D1 400 líneas» de `internal/components/sdd/inject.go` y sus tests;
  - coincidencias sin relación con la entrega (timeouts de 4000, HTTP 400, relleno de pruebas).
  - `internal/reviewtransaction/risk.go:22` sigue siendo residuo RDD hasta C-01-e.
- [ ] El pre-vuelo ya no pregunta la estrategia de PR. `monorepo-embedded` pregunta solo el ritmo y `multirepo` no pregunta nada. El bloque derivado tiene 5 líneas, dice `Delivery strategy: single-pr` y no lleva `Review policy`.
- [ ] Una transcripción con una respuesta `pr-strategy` heredada se acepta y el bloque sigue diciendo `single-pr`, en el hook y en ambos plugins, con los digests registrados.
- [ ] La doctrina de todos los runtimes y la de ODD dicen `single-pr` por defecto y `chained` (`stacked-to-main` / `feature-branch-chain`) solo a petición de la persona, sin presupuesto numérico.
- [ ] `sdd-apply` no contiene ninguna parada por carga de revisión y `sdd-tasks` no emite `400-line budget risk`, `Decision needed before apply` ni `Chained PRs recommended`.
- [ ] Un cambio o prompt con `ask-on-risk`, `exception-ok` o `auto-chain` se lee mediante los alias de Q1; un valor desconocido se sigue informando sin mapearlo (`delivery_strategy_vocabulary_test.go`).
- [ ] `axiom workspace validate` acepta `delivery: {split_hint_lines: 600}` y rechaza, nombrando `delivery.split_hint_lines`, los valores `0`, negativos, no enteros o nulos y las claves desconocidas. Sin el bloque el comportamiento no cambia y `axiom init --force` preserva el bloque.
- [ ] Con `delivery.split_hint_lines` fijada, la doctrina indica un único aviso no bloqueante cuando la previsión la supera; sin ella no hay aviso.
- [ ] Las dos copias de `work-unit-commits` y de `comment-writer` coinciden en el contenido afectado; `skills/branch-pr` no contiene el presupuesto.
- [ ] Goldens regenerados y `go test ./...` en verde.
