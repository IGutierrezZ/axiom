# Tareas: una sola PR por defecto y retirada del umbral de 400 líneas (C-02-c)

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | XL |
| Estimated changed lines | 1.500-2.500 (adiciones más eliminaciones, sin goldens ni ficheros generados) |

Entrega ya decidida: una sola PR con `size:exception` y un commit convencional por unidad de trabajo (sin atribución a IA). TDD desactivado (`strict_tdd: false`); el ejecutor de pruebas es `go test` en proceso, sin compilar binarios sueltos (Cortex XDR los borra).

### Work Units

| Unit | Goal | Commit |
|------|------|--------|
| 1 | Bloque `delivery:` con `split_hint_lines` en `axiom.yaml` (esquema, validación, init, docs) | `feat(workspace): bloque delivery con split_hint_lines en axiom.yaml` |
| 2 | Pre-vuelo sin estrategia de PR y bloque derivado de cinco líneas (hook, texto, plugins, digests, comandos, bench) | `refactor(sdd): pre-vuelo sin estrategia de PR y bloque de cinco líneas` |
| 3 | Doctrina ODD: `single-pr` por defecto, sin número | `refactor(odd): entrega single-pr por defecto en la doctrina ODD` |
| 4 | Orquestadores, convención de fases y skills de fase (atómico) | `refactor(sdd): entrega single-pr en orquestadores y fases sin guard de carga` |
| 5 | Skills de entrega y documentación (`chained-pr`, `work-unit-commits`, `comment-writer`, `branch-pr`, `AGENTS.md`) | `docs(skills): chained-pr a petición y work-unit-commits sin presupuesto` |
| 6 | Guarda de residuos de umbrales de entrega | `test(assets): guarda contra umbrales de entrega residuales` |

### Reglas de ejecución comunes a todas las unidades

- Raíz de trabajo: `C:\repos\axiom-wt\single-pr\axiom` (nunca el checkout principal). Todos los comandos se lanzan desde esa raíz.
- Pruebas siempre acotadas por paquete o por `-run`. **Nunca** `go test ./internal/cli/` completo de una vez (los tests de git y subproceso son lentos en Windows); en `internal/cli` usar siempre `-run` con los nombres afectados.
- Antes de cada commit: el paquete de la unidad en verde, `go vet` de los paquetes tocados y los goldens regenerados si la unidad cambia texto generado (`go test ./internal/components/ -run 'TestGolden' -update`; el flag `update` está definido en `internal/components/golden_test.go` (read-only), línea 32). Revisar `git diff -- testdata/golden` antes de añadirlos: solo deben cambiar las secciones de la unidad.
- Verificación final de la unidad 6 (y del cambio completo): `go test ./...` y `go vet ./...` desde la raíz, más `go vet ./...` dentro de `bench/` (módulo aparte).
- Commits en castellano, convencionales, sin `Co-Authored-By` ni atribución a IA. Un commit por unidad; el árbol compila y sus pruebas pasan en cada commit.
- `internal/components/sdd/inject.go` (literales de migración, Q4) y `internal/reviewtransaction/risk.go` no se tocan.

---

## Unidad 1: `delivery:` en `axiom.yaml`

Objetivo: fijar el esquema que nombran después la doctrina y los orquestadores.
Comando de prueba acotado de la unidad: `go test ./internal/workspace/ ./internal/hub/`.
Arnés de ejecución real: N/A (esquema y validación, cubierto en proceso por tests).
Límite de reversión: los ficheros de esta unidad; un `axiom.yaml` con `delivery:` sigue siendo válido tras revertir (los binarios anteriores ignoran las claves de primer nivel desconocidas).

- [x] 1.1 Crear `internal/workspace/delivery_test.go` con tabla de subtests espejo de `sdd_test.go`/`workflow_test.go` (casos RED de D10): válidos `split_hint_lines: 600`, `1`, `delivery: {}`; rechazos con `delivery.split_hint_lines` en el mensaje para `0`, `-1`, `"600"`, `6e2`, `1.5`, `true`, nulo, lista, mapeo, desbordamiento de `int`; clave desconocida, clave repetida, bloque nulo (`delivery:`) y bloque escalar; sin bloque `Declarations.Delivery == nil` y sin error; proyección en `WorkspaceConfig` (nil sin bloque).
  Cubre: REQ-32.1
- [x] 1.2 Crear `internal/workspace/delivery.go` con `DeliveryConfig{SplitHintLines int}`, `deliveryKeys = []string{"split_hint_lines"}`, `decodeDelivery(*yaml.Node)` (recorrido del nodo, `unknownKeys`, repetidas, `errors.Join`) y `deliveryPositiveInt(label, node)` (solo escalar `!!int`, mayor que cero, desbordamiento como error), con los mensajes en castellano de «Interfaces / Contratos» del diseño.
  Cubre: REQ-32.1
- [x] 1.3 Modificar `internal/workspace/declarations.go`: campos `Delivery *DeliveryConfig` y `DeliveryErr error` en `Declarations`, rama en `readDeclarationBlocks` y comentarios de la lista de bloques.
  Cubre: REQ-32.1, REQ-2.1
- [x] 1.4 Modificar `internal/workspace/loader.go`: añadir `DeliveryErr` a `joinLoadErrors`, proyectar `Delivery` en `parseLayered` (nil sin bloque) y actualizar los comentarios de lista de bloques.
  Cubre: REQ-32.1, REQ-2.1
- [x] 1.5 Modificar `internal/workspace/types.go`: `WorkspaceConfig.Delivery *DeliveryConfig` con etiqueta `yaml:"-"` y su comentario.
  Cubre: REQ-32.1
- [x] 1.6 Modificar `internal/hub/init.go`: `ownMarkers.declares()` incluye `d.Delivery != nil || d.DeliveryErr != nil`; actualizar el comentario de la lista de bloques y el mensaje de `--force` para nombrar `delivery:`.
  Cubre: REQ-32.1
- [x] 1.7 Ampliar el test de preservación existente de `internal/hub` (localizarlo con `rg -n "declares\(|workflow:" internal/hub --glob "*_test.go"`; si hay varios, el que cubre `axiom init --force`) con un caso nuevo: `axiom init --force` conserva un `axiom.yaml` que declara `delivery:`, válido o no.
  Cubre: REQ-32.1
- [x] 1.8 Modificar `axiom.example.yaml`: bloque comentado `# delivery:` / `#   split_hint_lines: 600` con su explicación (desactivado por defecto; el aviso nunca bloquea).
  Cubre: REQ-32.1, REQ-32.2
- [x] 1.9 Modificar `internal/workspace/example_yaml_test.go`: el bloque de la plantilla, una vez descomentado, es válido (patrón de las líneas 35-60 del mismo fichero).
  Cubre: REQ-32.1
- [x] 1.10 Modificar `docs/manual-de-inicio.md`: entrada del índice (líneas 24-25) y sección «### Aviso de Cambio Grande (`delivery:`)» tras «Carril de Trabajo» (clave, dominio, ausencia = desactivado, validación estricta, el binario anterior la ignora, `init --force` la preserva).
  Cubre: REQ-32.1, REQ-32.2
- [x] 1.11 Modificar `internal/workspace/docs_contract_test.go`: añadir `TestDocsDocumentTheDeliveryBlock` con el patrón de `TestDocsDocumentTheWorkflowLaneBlock` (líneas 213-252 del mismo fichero).
  Cubre: REQ-32.1
- [x] 1.12 Comprobar la unidad: `go test ./internal/workspace/` y `go test ./internal/hub/`, luego `go vet ./internal/workspace/ ./internal/hub/`. Todo en verde.
- [x] 1.13 Commit de la unidad 1: `feat(workspace): bloque delivery con split_hint_lines en axiom.yaml`.

---

## Unidad 2: pre-vuelo sin estrategia de PR

Objetivo: `monorepo-embedded` pregunta solo `pace`; `multirepo` no pregunta nada; bloque derivado de 5 líneas con `Delivery strategy: single-pr`; `pr-strategy` queda como grupo heredado ignorado.
Comandos de prueba acotados: `go test ./internal/cli/ -run 'SDDPreflight'`, `go test ./internal/components/sdd/`, `go test ./internal/assets/ -run 'Preflight|OpenCode|Topology'`, `go test ./internal/sddtaskresult/`.
Arnés de ejecución real: el arnés con node de `internal/sddtaskresult/preflight_question_harness_test.go` ejecuta los plugins v1 y v2 reales; la *journey* de `bench/` ejecuta el binario real y **no** se lanza en este cambio.
Límite de reversión: hook Go, texto proyectado, dos plugins y digests se revierten juntos; no toca el esquema de la unidad 1.

- [x] 2.1 Modificar `internal/cli/sdd_preflight_hook.go` (D4): `pr-strategy` en `sddPreflightGroups` pierde `RequiredIn` (conserva `ID`, `Name`, `LegacyNumber: 3` y `Choices`); `Delivery strategy` en `sddPreflightBlockFields` pasa a `{Default: "single-pr", Allowed: []string{"single-pr"}}` sin `GroupID`; eliminar la constante `sddPreflightBlockReview` y su uso en `sddPreflightBuildBlock` y `validSDDPreflightHookBlock`, que pasa a exigir `len(lines) == 2+len(sddPreflightBlockFields)` sin comprobar línea final.
  Cubre: REQ-28.15, REQ-28.16, REQ-28.17, REQ-28.18, REQ-28.23
- [x] 2.2 Modificar `internal/cli/sdd_preflight_hook_test.go` (usos de `pr-strategy`, líneas 150 y 758): bloque de 5 líneas; casos nuevos: transcripción con `pr-strategy` respondida («Ask me», «Auto»; marcador nominal y numerado `3/3:`) aceptada y con `single-pr`; `validSDDPreflightHookBlock` rechaza un bloque de 6 líneas y `Delivery strategy: ask-on-risk`; se conservan las denegaciones de hijo y de bloque escrito por el modelo.
  Cubre: REQ-28.16, REQ-28.17, REQ-28.18, REQ-28.23
- [x] 2.3 Modificar `internal/cli/sdd_preflight_hook_topology_test.go` (líneas 92, 124 y 254): bloque de 5 líneas; `monorepo-embedded` solo `pace`; `multirepo` sin grupos y sin transcripción.
  Cubre: REQ-28.15, REQ-28.17, REQ-28.18
- [x] 2.4 Comprobar el hook: `go test ./internal/cli/ -run 'SDDPreflight'` (si el patrón no cubre los nombres reales, localizarlos con `rg -n "^func Test" internal/cli/sdd_preflight_hook_test.go internal/cli/sdd_preflight_hook_topology_test.go` y pasarlos a `-run`). En verde.
- [x] 2.5 Modificar `internal/assets/opencode/plugins/sdd-task-result-artifacts.ts` (v1, D4): `requiredIn: []` en `pr-strategy`, `groupId: ""` en el campo `Delivery strategy`, fuera `SDD_PREFLIGHT_REVIEW_LINE` y su `lines.push` (hacia las líneas 53-69 y 132). Las descripciones de las opciones no cambian (la tabla TS se compara textualmente con la de Go).
  Cubre: REQ-28.21, REQ-28.23
- [x] 2.6 Modificar `internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts` (v2): la misma transformación (hacia las líneas 51-67 y 130).
  Cubre: REQ-28.21, REQ-28.23
- [x] 2.7 Modificar `internal/components/sdd/opencode_plugin_digests.go`: registrar los digests históricos de las revisiones sustituidas, v1 `c6e03fc1c7a03725cc1645b8285018f5dacd53fba8eb03cd51dfb26fe63fc28e // 43094d2d` y v2 `8c8bf972f4334dba3f8c947ef9b82eb34d6e931828887fb41b31271bd333d155 // 43094d2d`. Recalcular antes con `git show cb92574f:internal/assets/opencode/plugins/sdd-task-result-artifacts.ts` (y `plugins-v2/`) pasado por un hash SHA-256 en proceso (sin binarios sueltos) y comprobar que coinciden; si la base ha cambiado, usar el valor recalculado.
  Cubre: REQ-28.21
- [x] 2.8 Modificar `internal/components/sdd/opencode_runtime_test.go` (línea 101): `sdd-task-result-artifacts.ts` pasa a `{"v1": 10, "v2": 5}`.
  Cubre: REQ-28.21
- [x] 2.9 Modificar `internal/assets/opencode_sdd_preflight_topology_test.go` (líneas 49-51, 97, 107, 138, 175 y 180): constructor del bloque sin `Review policy`, expectativas de 5 líneas y de `multirepo` sin preguntas.
  Cubre: REQ-28.21
- [x] 2.10 Modificar `internal/sddtaskresult/preflight_question_harness_test.go`: arnés con node para v1 y v2: `multirepo` sin preguntas; respuesta `pr-strategy` heredada ignorada; bloque con `Delivery strategy: single-pr`; paridad de tablas TS/Go.
  Cubre: REQ-28.21, REQ-28.16
- [x] 2.11 Modificar `internal/assets/opencode_v2_sdd_test.go` solo si fija la tabla o la línea `Review policy` (comprobar con `rg -n "Review policy|pr-strategy" internal/assets/opencode_v2_sdd_test.go`; si no hay coincidencias, no se toca).
  Cubre: REQ-28.21
- [x] 2.12 Comprobar plugins: `go test ./internal/sddtaskresult/`, `go test ./internal/assets/ -run 'Preflight|Topology|OpenCode'` y `go test ./internal/components/sdd/ -run 'OpenCode'`. En verde (el arnés con node necesita `node` en el `PATH`).
- [x] 2.13 Modificar `internal/components/sdd/session_preflight.go` (D5): línea 18 (`monorepo-embedded` pregunta Pace; `multirepo` no pregunta nada), línea 20 (marcador único `Axiom SDD preflight pace:`), línea 22 (eliminar el bullet `PR strategy`), línea 24 (la entrega nunca se pregunta: siempre `single-pr`, la persona puede pedir PRs encadenadas en el chat), línea 25 (mapeos `Interactive -> interactive`, `Automatic -> auto`), líneas 29 y 79 (valores fijados por la topología: pace `interactive`, almacén `hybrid`, delivery strategy `single-pr`), línea 76 (sintaxis de reserva `Pace: <label>`, sin política de revisión). El ancla de la línea 75 no cambia.
  Cubre: REQ-28.14, REQ-28.15, REQ-28.20, REQ-28.23
- [x] 2.14 Modificar `internal/components/sdd/session_preflight_test.go` (líneas 25, 30, 120, 260 y 304-345): textos nuevos de las dos variantes (con y sin herramienta de preguntas) y aserción de ausencia de `400` en ambas.
  Cubre: REQ-28.14, REQ-28.20, REQ-28.23
- [x] 2.15 Modificar `internal/components/sdd/inject_test.go` (cláusulas de frescura del pre-vuelo, líneas 1423-1424, 1914-1915 y 2009-2010): ajustar a la redacción nueva sin tocar las aserciones 4R ni las de migración.
  Cubre: REQ-28.20
- [x] 2.16 Modificar `internal/components/sdd/delivery_strategy_vocabulary_test.go` (tests de productor, líneas 183, 223, 399 y 430): el pre-vuelo no ofrece grupo ni mapeo de estrategia; el único productor es el hook (`single-pr`, dentro del dominio). Conservar los tests de migración (líneas 341 y 453) ajustando la expectativa del bloque reinyectado (línea 510). Los tests del guard (líneas 265-339) se dejan para la unidad 4.
  Cubre: REQ-28.23, REQ-32.3
- [x] 2.17 Modificar los nueve comandos `internal/assets/opencode/commands/sdd-new.md`, `sdd-ff.md`, `sdd-continue.md`, `sdd-explore.md`, `sdd-init.md`, `sdd-apply.md`, `sdd-verify.md`, `sdd-archive.md` y `sdd-onboard.md`: frase idéntica «SDD Session Preflight must already be complete for this session: the pace the topology asks (the artifact store is always `hybrid` and delivery is `single-pr` unless the person asks for chained PRs).»; en `sdd-apply.md` la línea 19 pasa a «5. Pass the resolved `delivery_strategy`, and `chain_strategy` when it is `chained`; delivery never blocks apply.» y la línea 33 a «- The resolved `delivery_strategy` and, when `chained`, its `chain_strategy`.»; en `sdd-onboard.md` la línea 19 pierde «and enforce the review budget before apply».
  Cubre: REQ-28.20, REQ-32.3
- [x] 2.18 Modificar `bench/journeys_issue_3336.go` (línea 18): deja de exigir `- **PR strategy**` y `fixed at 400 changed lines` y exige la frase de entrega fija del pre-vuelo nuevo.
  Cubre: REQ-28.23
- [x] 2.19 Comprobar `bench/`: ejecutar `go vet ./...` y `go test ./...` **desde `bench/`** (módulo aparte con `bench/go.mod`; `go test ./...` desde la raíz no lo cubre). La *journey* 3336 ejecuta el binario real: no se lanza; solo debe compilar y pasar el vet.
- [x] 2.20 Regenerar goldens: `go test ./internal/components/ -run 'TestGolden' -update`; revisar `git diff -- testdata/golden` (solo la proyección del pre-vuelo en los runtimes de reserva) y volver a ejecutar sin `-update`: `go test ./internal/components/ -run 'TestGolden'`.
- [x] 2.21 Comprobar la unidad completa: `go test ./internal/components/sdd/`, `go test ./internal/components/`, `go test ./internal/sddtaskresult/`, `go test ./internal/assets/`, los `-run` del hook de 2.4, y `go vet ./internal/cli/ ./internal/components/... ./internal/assets/... ./internal/sddtaskresult/`. En verde.
- [x] 2.22 Commit de la unidad 2: `refactor(sdd): pre-vuelo sin estrategia de PR y bloque de cinco líneas`.

---

## Unidad 3: doctrina ODD

Objetivo: dos bullets de `routing.go` reescritos sin número, el de entrega autocontenido para que C-02-d lo extienda.
Comando de prueba acotado: `go test ./internal/components/agentguidance/` y `go test ./internal/components/ -run 'TestGolden'`.
Arnés de ejecución real: N/A (texto proyectado; lo cubren las aserciones de `RenderRouting` por agente y los goldens).
Límite de reversión: `routing.go`, su test y los goldens de doctrina.

- [x] 3.1 Modificar `internal/components/agentguidance/routing_test.go` (RED): aserciones de la doctrina nueva en `RenderRouting` de cada agente (`single-pr` por defecto, `chained` solo a petición, `stacked-to-main` y `feature-branch-chain`, `delivery.split_hint_lines`, aviso único sin bloqueo, `work-unit-commits` siempre y `chained-pr` solo al encadenar) y de ausencia de `400`, `ask-on-risk`, `auto-chain`, `exception-ok` y «Existing PR size gates».
  Cubre: REQ-32.4, REQ-32.2
- [x] 3.2 Modificar `internal/components/agentguidance/routing.go` (línea 122): «Size each ODD task as the smallest coherent behavior with its tests and docs; there is no line-count target, cap, or forced split. Never delete spaces, blank lines, or comments for cosmetic line savings; never omit tests, minify, add gratuitous abstractions, or split artificially. Forward this same instruction when delegating tasks to subagents.»
  Cubre: REQ-32.4
- [x] 3.3 Modificar `internal/components/agentguidance/routing.go` (línea 130): bullet «Delivery follows work units» con el texto exacto de «Doctrina ODD (`routing.go`)» del diseño (`single-pr` por defecto, previsión al crear el documento de la feature, aviso único y no bloqueante con `delivery.split_hint_lines`, `chained` solo a petición con una única pregunta por la estrategia de cadena, `work-unit-commits` siempre y `chained-pr` solo al encadenar). Debe terminar en una frase a la que C-02-d pueda añadir texto sin reescribir.
  Cubre: REQ-32.4, REQ-32.2
- [x] 3.4 Regenerar goldens de doctrina: `go test ./internal/components/ -run 'TestGolden' -update`; revisar `git diff -- testdata/golden` (solo los dos bullets) y comprobar sin `-update`.
- [x] 3.5 Comprobar la unidad: `go test ./internal/components/agentguidance/`, `go test ./internal/components/ -run 'TestGolden'` y `go vet ./internal/components/...`. En verde.
- [x] 3.6 Commit de la unidad 3: `refactor(odd): entrega single-pr por defecto en la doctrina ODD`.

---

## Unidad 4: orquestadores, convención de fases y skills de fase (atómica)

Objetivo: una sola sección canónica `Delivery Strategy`, `Chain Strategy` con sustitución idéntica, retirada de `Review Workload Guard` en los 12 runtimes y en la convención de fases, y `sdd-tasks`/`sdd-apply` sin parada por carga. Es atómica a propósito: el test de vocabulario deriva el dominio de `sdd-tasks`/`sdd-apply` y lo exige en los orquestadores; partirla deja el árbol en rojo.
Comandos de prueba acotados: `go test ./internal/components/sdd/`, `go test ./internal/assets/`, `go test ./internal/components/ -run 'TestGolden'`.
Arnés de ejecución real: N/A (texto de orquestadores y skills; lo cubren los tests de renderizado de los 13 runtimes y los goldens).
Límite de reversión: todos los ficheros de esta unidad se revierten juntos con los goldens.

### 4A. Tests primero (RED)

- [x] 4.1 Modificar `internal/components/sdd/orchestrator_shared_sections_test.go`: añadir `"Delivery Strategy"` a `sharedOrchestratorSectionNames` (hacia la línea 103), de modo que `TestSharedOrchestratorSectionsHaveOneSource` impida copias en línea y `TestEveryRuntimeRendersTheSharedSections` compruebe que los 13 runtimes la renderizan.
  Cubre: REQ-32.3
- [x] 4.2 Modificar `internal/assets/orchestrator_drift_ratchet_test.go`: eliminar la entrada de `Review Workload Guard` (línea 28; el test exige retirarla a propósito, líneas 92-95) y bajar `Delivery Strategy` de 4 a 1; `Chain Strategy` (6) y `Automatic Mode Gatekeeper (MANDATORY)` (8) no cambian.
  Cubre: REQ-32.3
- [x] 4.3 Modificar `internal/components/sdd/renderer_invariants_test.go` (líneas 104-122): centinelas nuevos de `Delivery Strategy` (`single-pr`, `chained`, ``Pass `delivery_strategy` ``) y fuera la entrada del guard.
  Cubre: REQ-32.3
- [x] 4.4 Modificar `internal/components/sdd/inject_test.go` (líneas 2639-2650): la sección de cadena acaba en el siguiente `### ` (ya no en el guard) y se afirman las frases nuevas; conservar las coletillas por runtime de las líneas 2569-2614.
  Cubre: REQ-32.3, Chain Strategy Guidance Parity, Non-Claude Chain Strategy Guidance Parity
- [x] 4.5 Modificar `internal/components/sdd/orchestrator_composition_test.go`: ajustar lo que fije del guard o de la entrega; conservar la negativa de la línea 543.
  Cubre: REQ-32.3
- [x] 4.6 Modificar `internal/components/sdd/delivery_strategy_vocabulary_test.go`: el dominio se lee de `sdd-apply`, `sdd-tasks` y `sdd-phase-common` con el patrón `` `single-pr | chained` ``; los tests del guard (líneas 265-339) pasan a las secciones `Delivery Strategy`: cubren el dominio, enuncian los alias (`→`, nunca con `|`) y rechazan un valor desconocido.
  Cubre: REQ-32.3
- [x] 4.7 Modificar `internal/components/sdd/planning_detail_test.go` (línea 23): `## Delivery Forecast` y `Estimated changed lines` en lugar de `## Review Workload Forecast` y `400 changed-line review budget`.
  Cubre: REQ-32.3

### 4B. Fuente canónica y convención de fases

- [x] 4.8 Modificar `internal/assets/skills/_shared/sdd-orchestrator-sections.md`: reescribir `Delivery Strategy` (líneas 261-263) con el cuerpo canónico de «Interfaces / Contratos» del diseño (entrega nunca preguntada, dominio `single-pr | chained`, alias de lectura, valor inválido detiene, precedencia, reenvío de `delivery_strategy`/`chain_strategy`, caché solo de sesión, aviso único tras la previsión con `delivery.split_hint_lines`); aplicar la sustitución idéntica a `Chain Strategy` (líneas 265-274); sustituir la frase del gatekeeper (línea 254) por «The gatekeeper runs in addition to the Mandatory Delegation Triggers»; eliminar el bloque `Review Workload Guard (MANDATORY)` (líneas 405-420).
  Cubre: REQ-32.3, REQ-32.2, Chain Strategy Propagation to Downstream Phases
- [x] 4.9 Modificar `internal/assets/skills/_shared/sdd-phase-common.md` (§E, líneas 109-125): pasa a «## E. Delivery Strategy» con dominio `single-pr | chained` (declaración única en esa forma), alias en línea aparte con `→`, valor inválido y precedencia de D1/D2; sin presupuesto ni líneas de guarda.
  Cubre: REQ-32.3
- [x] 4.10 Modificar `internal/assets/skills/sdd-tasks/SKILL.md`: línea 38 (dominio `single-pr | chained`); plantilla (líneas 84-101) y reglas (líneas 149-183) sustituidas por `## Delivery Forecast` (tamaño `S | M | L | XL`, rango de líneas cambiadas, unidades de trabajo con porción de PR y rama base solo si `chained`; sin campos de riesgo, decisión, estrategia ni número; nunca escribir `Delivery strategy` ni `Chain strategy` como campos); resumen (líneas 238-245) y regla de la línea 262 sin guard de 400.
  Cubre: REQ-32.3, REQ-32.2
- [x] 4.11 Modificar `internal/assets/skills/sdd-apply/SKILL.md`: línea 40 (dominio); reescribir el paso 2a (líneas 82-104) como «Step 2a: Follow the Delivery Strategy» (solo rama destino: `stacked-to-main` rama de la porción anterior o de integración, `feature-branch-chain` rama *tracker* para la porción 1; sin parada por tamaño; si recibe `chained` sin estrategia apunta a la rama del cambio y lo anota; conserva «no minificar» sin número); no renumerar el paso 2b; eliminar la regla de la línea 333 y quitar el número de la línea 334.
  Cubre: REQ-32.3

### 4C. Orquestadores por runtime

- [x] 4.12 Modificar `internal/assets/claude/sdd-orchestrator-workflow.md`: `Delivery Strategy` (líneas 98-108) pasa a plantilla `{{GENTLE_AI_SDD_SECTION:Delivery Strategy}}`; cadena (líneas 109-120) con las mismas cláusulas en redacción compacta; eliminar `Review Workload Guard` (líneas 138-153); frase del gatekeeper idéntica.
  Cubre: REQ-32.3, Chain Strategy Guidance Parity
- [x] 4.13 Modificar `internal/assets/claude/sdd-orchestrator.md` (línea 68): dejar de listar «review workload guard» en la superficie perezosa.
  Cubre: REQ-32.3
- [x] 4.14 Modificar `internal/assets/gemini/sdd-orchestrator.md` y `internal/assets/qwen/sdd-orchestrator.md` (plantillas): eliminar el encabezado de `Review Workload Guard` y su plantilla; frase del gatekeeper idéntica; `Chain Strategy` ya por plantilla.
  Cubre: REQ-32.3
- [x] 4.15 Modificar `internal/assets/codex/sdd-orchestrator.md`, `internal/assets/kiro/sdd-orchestrator.md` y `internal/assets/opencode/sdd-orchestrator.md`: `### Delivery Strategy` en línea pasa a `{{GENTLE_AI_SDD_SECTION:Delivery Strategy}}`; eliminar `Review Workload Guard` en línea; sustitución idéntica en `Chain Strategy`; frase del gatekeeper (copias en línea). Conservar las coletillas por runtime (Kiro «phase context», ruta `question` de OpenCode).
  Cubre: REQ-32.3, Non-Claude Chain Strategy Guidance Parity, Inline Chain Strategy Summary Retained
- [x] 4.16 Modificar `internal/assets/antigravity/sdd-orchestrator.md`, `internal/assets/cursor/sdd-orchestrator.md`, `internal/assets/generic/sdd-orchestrator.md`, `internal/assets/hermes/sdd-orchestrator.md`, `internal/assets/kimi/sdd-orchestrator.md` y `internal/assets/windsurf/sdd-orchestrator.md`: eliminar `Review Workload Guard` (en línea; en `generic` líneas 106-117), sustitución idéntica en `Chain Strategy` (plantilla o en línea según el runtime; conservar coletillas «Kimi custom-agent prompt context», «inline phase context», «dynamic subagent context»), frase del gatekeeper idéntica; `Delivery Strategy` ya por plantilla.
  Cubre: REQ-32.3, Cursor Chain Strategy Section Parity, Chain Strategy Guidance Parity
- [x] 4.17 Verificar con `rg -n "Review Workload Guard|review workload guard" internal/assets internal/components` que no queda ninguna referencia viva (solo pueden quedar aserciones negativas de tests); corregir cualquier resto en los orquestadores.
  Cubre: REQ-32.3
- [x] 4.18 Modificar `internal/components/sdd/orchestrator.go` (D6): eliminar la tupla de la línea 152 de `composeFallbackSessionPreflight`; conservar la comprobación de cardinalidad de la línea 144 y los dos encabezados `### Delivery Strategy` y `### Chain Strategy` (el segundo sigue siendo el ancla de la cadena y el primero el de `### Artifact Store Mode`).
  Cubre: REQ-32.3

### 4D. Verificación y goldens

- [x] 4.19 Comprobar los orquestadores: `go test ./internal/components/sdd/` y `go test ./internal/assets/`. En verde (ratchet con `Delivery Strategy` a 1; `TestEveryRuntimeRendersTheSharedSections` con los 13 runtimes).
- [x] 4.20 Regenerar goldens: `go test ./internal/components/ -run 'TestGolden' -update`; revisar `git diff -- testdata/golden` (las 11 proyecciones de orquestador; solo deben cambiar `Delivery Strategy`, `Chain Strategy`, el gatekeeper y la retirada del guard) y comprobar sin `-update`.
- [x] 4.21 Comprobar la unidad completa: `go test ./internal/components/...`, `go test ./internal/assets/...`, `go vet ./internal/components/... ./internal/assets/...`.
- [x] 4.22 Commit de la unidad 4: `refactor(sdd): entrega single-pr en orquestadores y fases sin guard de carga`.

---

## Unidad 5: skills de entrega y documentación

Objetivo: `chained-pr` solo a petición, `work-unit-commits` sin presupuesto, `comment-writer` y `branch-pr` sin número, copias del proyecto alineadas.
Comandos de prueba acotados: `go test ./internal/assets/ -run 'SkillCopies'`, `go test ./internal/skillregistry/ -run 'TestRepoAgentsIndexMatchesVersionedSkills'`, `go test ./internal/components/ -run 'TestGolden'`.
Arnés de ejecución real: N/A (texto de skills y documentación). No ejecutar `axiom skill-registry refresh` (exige compilar el binario).
Límite de reversión: skills, `AGENTS.md`, `docs/components.md` y test de paridad.

- [x] 5.1 Crear `internal/assets/skill_copies_parity_test.go` (RED): paridad byte a byte de `work-unit-commits` y `comment-writer` entre el asset embebido y `../../skills/<skill>/SKILL.md`, con el patrón de `rdd_defect_workflow_skill_test.go` (líneas 11-25 del mismo directorio). Debe fallar hoy en `work-unit-commits` (divergencia en dos líneas).
  Cubre: REQ-32.3
- [x] 5.2 Modificar `internal/assets/skills/chained-pr/SKILL.md`: `description`, activación, reglas y tabla de decisión sin número; activación solo cuando la persona pide encadenar, apilar o rebanar o con `delivery_strategy: chained`; frase «When `delivery_strategy` is `chained`, …» de la sustitución idéntica.
  Cubre: REQ-32.3, chained-pr Skill Binding in Chain Strategy Guidance
- [x] 5.3 Modificar `internal/assets/skills/chained-pr/references/chaining-details.md`: quitar la fila «Review budget» del contrato de salida (línea 62) y la referencia de la línea 96.
  Cubre: REQ-32.3
- [x] 5.4 Modificar `AGENTS.md` (línea 51): `description` de `chained-pr` idéntico al nuevo de 5.2, a mano.
  Cubre: REQ-32.3
- [x] 5.5 Modificar `docs/components.md` (línea 77): «Planifica PR encadenadas cuando la persona las pide.»
  Cubre: REQ-32.3
- [x] 5.6 Modificar `internal/assets/skills/work-unit-commits/SKILL.md` (canónica; líneas 20, 32-33, 64 y 66-74): un commit por unidad dentro de una sola PR; cadena solo a petición; conserva «no code-golf» sin número.
  Cubre: REQ-32.3
- [x] 5.7 Modificar `skills/work-unit-commits/SKILL.md`: copia byte a byte de la embebida de 5.6 (hoy divergen en las líneas 33 y 74 de la embebida).
  Cubre: REQ-32.3
- [x] 5.8 Modificar `internal/assets/skills/comment-writer/SKILL.md` y `skills/comment-writer/SKILL.md` (línea 64 de ambas): ejemplo de tono sin presupuesto, idéntico en las dos copias.
  Cubre: REQ-32.3
- [x] 5.9 Modificar `skills/branch-pr/SKILL.md` (D11): retirar el punto 3 de la línea 23, la casilla de la línea 126, el ejemplo de la línea 274 y el número de la fila de la línea 143 (la fila remite a la política de tamaño del repositorio sin citar número); conservar el punto 2 de la línea 22 (gobierno de la etiqueta protegida `size:exception`).
  Cubre: REQ-32.3
- [x] 5.10 Regenerar goldens si alguna proyección de skills cambia: `go test ./internal/components/ -run 'TestGolden' -update`; revisar el diff (puede no haber cambios) y comprobar sin `-update`.
- [x] 5.11 Comprobar la unidad: `go test ./internal/assets/`, `go test ./internal/skillregistry/ -run 'TestRepoAgentsIndexMatchesVersionedSkills'`, `go test ./internal/components/ -run 'TestGolden'` y `go vet ./internal/assets/... ./internal/skillregistry/...`. En verde.
- [x] 5.12 Commit de la unidad 5: `docs(skills): chained-pr a petición y work-unit-commits sin presupuesto`.

---

## Unidad 6: guarda de residuos

Objetivo: que ningún umbral de entrega vuelva a colarse en los assets ni en `skills/`.
Comando de prueba acotado: `go test ./internal/assets/ -run 'DeliveryThresholdResidue'`.
Arnés de ejecución real: la verificación con `rg` de 6.3 sobre el árbol real.
Límite de reversión: el fichero de test nuevo.

- [x] 6.1 Crear `internal/assets/delivery_threshold_residue_test.go` (D12): recorre `assets.FS` y `../../skills` y falla ante cualquier línea con `400` como palabra (`\b400\b`) o con los literales retirados (`400-line budget risk`, `Decision needed before apply`, `Chained PRs recommended`, `workload-decision-required`, `Review Workload Guard`, `review_budget_lines`). Residuo admitido por ruta **y** contenido: líneas con `Full-4R review` en `*/agents/jd-judge-*`, `*/agents/review-*` y `opencode/sdd-overlay-{single,multi}.json`; cualquier línea de `skills/rdd-defect-workflow/` (sus dos copias; las retira C-01-e).
  Cubre: REQ-32.3, REQ-28.23
- [x] 6.2 Ajustar los tests de texto generado desde Go que vigilan el residuo en su paquete, si no quedaron ya cubiertos: `internal/components/agentguidance/routing_test.go` (sin `400`, `ask-on-risk`, `auto-chain`, `exception-ok`), `internal/components/sdd/session_preflight_test.go` (sin `400` en las dos variantes) y `internal/cli/sdd_preflight_hook_test.go` (bloque de 5 líneas). Ejecutar `go test ./internal/components/agentguidance/ ./internal/components/sdd/ -run 'Routing|Preflight'` y `go test ./internal/cli/ -run 'SDDPreflight'`.
  Cubre: REQ-32.4, REQ-28.23
- [x] 6.3 Criterio de hecho: ejecutar `rg -n "\b400\b" internal/assets internal/components internal/cli skills` y confirmar que solo queda residuo admitido: umbral de barrido «Full-4R» de `jd-judge-a`/`jd-judge-b` y sus proyecciones; lentes RDD `review-*` y overlays de OpenCode; `rdd-defect-workflow` (sus dos copias y su test); literales de migración de `internal/components/sdd/inject.go` y sus aserciones en `internal/components/sdd/inject_test.go`; la aserción 4R de `inject_test.go`; la línea 16 de `internal/cli/review_process_boundary_test.go` («400-line size rule», residuo RDD de C-01-e igual que `internal/reviewtransaction/risk.go:22`, que no está bajo estas rutas); y coincidencias sin relación con la entrega (HTTP 400, relleno de pruebas). Cualquier otra coincidencia se corrige antes del commit (volviendo a la unidad que la dejó) o se reporta como desviación.
  Cubre: REQ-32.3, REQ-28.23
- [x] 6.4 Verificación final del cambio completo, desde la raíz: `go test ./...` y `go vet ./...`; después, dentro de `bench/`: `go vet ./...` y `go test ./...`. Todo en verde; anotar cualquier fallo preexistente con evidencia.
- [x] 6.5 Comprobar que los goldens no tienen cambios pendientes: `go test ./internal/components/ -run 'TestGolden'` sin `-update` y `git status` sin ficheros sin añadir.
- [x] 6.6 Commit de la unidad 6: `test(assets): guarda contra umbrales de entrega residuales`.

---

## Trazabilidad de requisitos

| Requisito | Tareas |
|-----------|--------|
| REQ-2.1 (`workspace-topology`, bloque `delivery` admitido) | 1.3, 1.4 |
| REQ-32.1 (bloque `delivery:`) | 1.1-1.11 |
| REQ-32.2 (aviso único no bloqueante) | 1.8, 1.10, 3.1, 3.3, 4.8, 4.10 |
| REQ-32.3 (entrega `single-pr`, `chained` a petición, alias) | 2.16, 2.17, 4.1-4.11, 4.12-4.18, 5.1-5.9, 6.1, 6.3 |
| REQ-32.4 (doctrina ODD) | 3.1-3.3, 6.2 |
| REQ-28.14, REQ-28.15, REQ-28.16, REQ-28.17, REQ-28.18, REQ-28.20, REQ-28.21 | 2.1-2.15 |
| REQ-28.23 (sin umbral numérico en pre-vuelo) | 2.1, 2.2, 2.6, 2.13, 2.14, 2.16, 2.18, 6.1-6.3 |
| Requisitos de cadena de `sdd-orchestrator-assets` (Chain Strategy Guidance Parity, Propagation, Non-Claude, chained-pr Skill Binding, Inline Summary, Cursor Parity) | 4.4, 4.8, 4.12, 4.15, 4.16, 5.2 |

REQ-28.22 (topología efectiva) no cambia de comportamiento; solo lo ejercen indirectamente 2.3 y 2.9.

## Orden y paralelismo

Secuencial por unidad (1 → 2 → 3 → 4 → 5 → 6): la 3 solo depende del nombre de la clave de la 1; la 4 es atómica; la 5 necesita los textos de fase de la 4; la 6 solo pasa cuando 2-5 han retirado todo el residuo. Dentro de una unidad, las tareas de test (RED) preceden a las de producción. Sin ejecución en paralelo: un único escritor, un worktree, y los goldens se regeneran de forma serie (una regeneración por unidad).
