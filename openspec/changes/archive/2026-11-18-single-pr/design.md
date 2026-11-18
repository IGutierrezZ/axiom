# Diseño: una sola PR por defecto y retirada del umbral de 400 líneas (C-02-c)

Base: `main` en `cb92574f` (worktree `C:\repos\axiom-wt\single-pr\axiom`). Todas las referencias `fichero:línea` se han comprobado en ese worktree. La propuesta (`proposal.md`) es la autoridad: este diseño no reabre P10, P5, Q1-Q6 ni el criterio ampliado; solo fija el «cómo».

## Enfoque técnico

Alternativa A de la propuesta: una reescritura completa en un solo cambio, entregada en una PR con `size:exception` y seis commits por unidad de trabajo, cada uno con el árbol compilable y `go test ./...` en verde.

El cambio tiene cinco superficies independientes y una regla de autoridad que las une:

1. **Esquema de `axiom.yaml`** (`internal/workspace`): nuevo bloque opcional `delivery:` con la única clave `split_hint_lines`, decodificado con el mismo patrón que `workflow:` y `sdd:` (lista cerrada de claves, recorrido del `yaml.Node`, canal de error propio en `Declarations`).
2. **Pre-vuelo** (hook Go, dos generaciones del plugin de OpenCode, texto proyectado): `pr-strategy` pasa a grupo heredado (sin `RequiredIn`, igual que `artifacts`); el bloque derivado pierde la línea `Review policy` y queda en 5 líneas; la línea `Delivery strategy` se conserva con el dominio cerrado `single-pr`.
3. **Doctrina ODD** (`internal/components/agentguidance/routing.go`): dos bullets reescritos sin número; el de entrega queda autocontenido para que C-02-d lo extienda.
4. **Orquestadores y convención de fases**: una sola sección canónica `Delivery Strategy` en `sdd-orchestrator-sections.md`, referenciada por plantilla desde los 12 `sdd-orchestrator.md` y desde el flujo de Claude; `Chain Strategy` se conserva con una sustitución textual idéntica en todas sus copias; `Review Workload Guard` desaparece de todos los runtimes y de `sdd-phase-common.md` §E, que pasa a describir la estrategia de entrega.
5. **Skills** (`chained-pr`, `work-unit-commits`, `sdd-tasks`, `sdd-apply`, `comment-writer`, `skills/branch-pr`): sin número, cadena solo a petición, previsión cualitativa y sin parada por carga.

La regla de autoridad (decisión D2) resuelve el riesgo principal de la propuesta: el bloque confirmado por el runtime siempre dice `Delivery strategy: single-pr`, mientras que la petición de encadenar solo existe en el chat y en la caché de sesión del orquestador.

## Decisiones de arquitectura

### D1. Dominio canónico, alias de lectura y valor inválido

**Elección**: el dominio de `delivery_strategy` es `single-pr | chained` (Q1). Los alias `ask-on-risk` → `single-pr`, `exception-ok` → `single-pr` y `auto-chain` → `chained` se aplican **solo al leer** un valor existente: un bloque de pre-vuelo antiguo (desfase de versiones entre el hook, el plugin y los orquestadores) o la fila `Delivery strategy` / línea `Chain strategy` del `tasks.md` de un cambio en curso. Ningún componente escribe ni reenvía un alias: lo que se reenvía es siempre el valor canónico ya traducido. Cualquier otro valor sigue siendo inválido: no se mapea al más parecido, se informa y la fase se detiene (es una validación de contrato, no una parada por carga).

La declaración del dominio vive, exactamente una vez y con la forma `` `single-pr | chained` ``, en `sdd-tasks/SKILL.md`, `sdd-apply/SKILL.md` y `_shared/sdd-phase-common.md` §E; los alias se enuncian en una línea aparte con `→` (nunca con `|`), para que la expresión regular del test de vocabulario no los confunda con el dominio.

**Alternativas consideradas**: (a) conservar los cuatro valores como sinónimos permanentes; (b) rechazar los valores heredados como inválidos.
**Justificación**: (a) mantiene vivo el vocabulario retirado y obliga a seguir documentándolo; (b) rompe cambios en curso con `tasks.md` generados por la versión anterior. El alias de solo lectura cumple «se sigue leyendo» sin persistir nada nuevo (Q6).

### D2. Precedencia del valor de entrega

**Elección**: el orquestador es la única autoridad que resuelve la entrega de una fase; las fases solo la consumen. Orden de precedencia, de mayor a menor:

| Nivel | Fuente | Quién la lee | Efecto |
|---|---|---|---|
| 1 | Petición explícita de la persona en el chat de esta sesión («encadena», «apila», «divide en PRs»), cacheada por el orquestador como `chained` + `chain_strategy` | Orquestador | Prevalece sobre todo lo demás |
| 2 | Valor heredado en el `tasks.md` del cambio (fila `Delivery strategy` y línea o fila `Chain strategy`), leído con los alias de D1 al reanudar | Orquestador | Solo para ese cambio; nunca se reescribe |
| 3 | Línea `- Delivery strategy:` del bloque de pre-vuelo confirmado por el runtime | Orquestador y fases | Valor por defecto de la sesión (`single-pr`; un bloque antiguo se lee con alias) |

- El orquestador reenvía siempre `delivery_strategy` (`single-pr` o `chained`) a `sdd-tasks` y `sdd-apply`, y `chain_strategy` solo cuando es `chained`.
- `sdd-apply` resuelve con este orden reducido: `delivery_strategy`/`chain_strategy` del prompt de lanzamiento; si faltan, la línea del bloque. **No** lee los campos de entrega de `tasks.md`: así hay una sola autoridad (el orquestador) y la fase no puede contradecirla.
- `sdd-apply` usa la estrategia **solo** para elegir la rama destino (`stacked-to-main`: la rama de la porción anterior o la de integración; `feature-branch-chain`: la rama *tracker* para la porción 1 y la porción anterior para las siguientes). Nunca se detiene por tamaño, previsión o ausencia de decisión.
- `chained` sin `chain_strategy` concreta (petición sin estrategia, `auto-chain` heredado o `Chain strategy: pending`): el orquestador hace **una** pregunta por la estrategia de cadena antes del siguiente `sdd-tasks` o `sdd-apply`. Si, pese a ello, `sdd-apply` recibe `chained` sin estrategia, apunta a la rama del cambio como en `single-pr` y lo anota en su resultado; no bloquea.
- `chained` vive solo en la caché de la sesión (Q6): no se escribe en `kickoff.yaml`, `gates.yaml` ni como campo de `tasks.md`. Una sesión nueva vuelve a `single-pr` salvo que el nivel 2 aporte un valor heredado.

**Alternativas consideradas**: (a) que el bloque del runtime sea la única autoridad (la persona no podría encadenar sin reabrir el pre-vuelo, contra P10); (b) que `sdd-apply` lea también `tasks.md` (dos autoridades que pueden discrepar); (c) persistir `chained` en `kickoff.yaml` (contra Q6).
**Justificación**: el bloque expresa lo que el runtime puede certificar (la sesión empezó sin petición de cadena); la petición posterior solo la conoce el orquestador. La entrega no es un límite de seguridad: un `delivery_strategy: chained` escrito por un modelo solo cambia la rama destino de commits locales, y el push y la PR siguen siendo de la persona. Por eso no necesita autoridad del runtime, a diferencia del ritmo.

### D3. Línea `Delivery strategy` en el bloque derivado: se conserva

**Elección**: el bloque pasa de 6 a 5 líneas: cabecera, autoridad, `Pace`, `Artifact store` y `Delivery strategy` (siempre `single-pr`). Se retira solo `- Review policy: 400 changed lines`.
**Alternativas consideradas**: retirar también `Delivery strategy` (bloque de 4 líneas).
**Justificación**: (1) REQ-28.18 ya fija «`Delivery strategy` derivado `single-pr`» y la ruta de cero grupos de C-01-b (`internal/cli/sdd_preflight_hook.go:160-165`, `sddPreflightDerivedBlock`) la usa para `multirepo` sin cambios; (2) da a las fases el valor del nivel 3 de D2 sin tener que deducirlo de la ausencia; (3) cada línea que se quita es otro cambio de forma del bloque en el hook, en dos plugins y en sus digests, sin beneficio funcional.

### D4. `pr-strategy` pasa a grupo heredado como `artifacts`

**Elección**:
- `sddPreflightGroups` (`sdd_preflight_hook.go:425-434`): el grupo `pr-strategy` conserva `ID`, `Name`, `LegacyNumber: 3` y `Choices` sin cambios y pierde `RequiredIn`. `resolveSDDPreflightHookBlock` (`:561-605`) ya ignora, sin validar sus opciones ni su respuesta, la pregunta cuyo marcador nombra un grupo no requerido; `sddPreflightQuestionGroup` (`:481-499`) sigue reconociendo el marcador nominal `pr-strategy:` y el numerado `3/3:`.
- `sddPreflightBlockFields` (`:450-454`): `{Label: "Delivery strategy", Default: "single-pr", Allowed: []string{"single-pr"}}`, sin `GroupID`. `sddPreflightBuildBlock` (`:502-512`) ya protege `values[field.GroupID]` con `field.GroupID != ""`.
- Se elimina la constante `sddPreflightBlockReview` (`:459`) y su uso en `sddPreflightBuildBlock` (`:511`) y en `validSDDPreflightHookBlock` (`:679-697`), que pasa a exigir `len(lines) == 2+len(sddPreflightBlockFields)` sin comprobar línea final.
- Mismo cambio en las dos generaciones del plugin (`plugins/sdd-task-result-artifacts.ts:53-69` y `plugins-v2/...:51-67`): `requiredIn: []` en `pr-strategy`, `groupId: ""` en el campo `Delivery strategy`, fuera `SDD_PREFLIGHT_REVIEW_LINE` y su `lines.push` (`:132` v1, `:130` v2). Las descripciones de las opciones no cambian (la tabla TS se compara textualmente con la de Go).
- Resultado por topología: `monorepo-embedded` pregunta solo `pace`; `multirepo` no tiene grupos y entra por la ruta de cero grupos, que no exige transcripción. Una transcripción antigua con la pregunta `pr-strategy` respondida («Ask me», «Auto») se acepta y el bloque dice `single-pr`.

**Alternativas consideradas**: borrar el grupo de la tabla.
**Justificación**: sin el grupo, un marcador `pr-strategy:` dejaría de reconocerse y una transcripción antigua de tres preguntas fallaría o se trataría como pregunta desconocida. Es exactamente el caso que resolvió `artifacts`.

### D5. Texto proyectado del pre-vuelo

**Elección** en `internal/components/sdd/session_preflight.go`:
- `:18`: «`monorepo-embedded` asks Pace. `multirepo` asks nothing: its pace is always Interactive and is never asked.» Se conserva «When the topology requires no group, ask nothing…» y «Never ask the artifact store: it is always `hybrid`.»
- `:20`: el marcador único `` `Axiom SDD preflight pace:` ``.
- `:22`: se elimina el bullet `- **PR strategy** (`pr-strategy`)…`.
- `:24`: «Review policy is fixed at 400…» se sustituye por «Delivery is never asked: it is always `single-pr`, and the person may ask for chained pull requests in chat later; NEVER ask it as a preflight group.»
- `:25`: los mapeos canónicos quedan en `Interactive -> interactive` y `Automatic -> auto`.
- `:29` y `:79` (párrafos de autoridad nativa y de reserva): los valores que fija la topología pasan a ser «`multirepo` pace `interactive`, artifact store `hybrid`, delivery strategy `single-pr`».
- `:76`: sintaxis de respuesta de reserva `` `Pace: <label>` ``; desaparece «the fixed review policy». El ancla `"\n\n- **Pace** (`pace`"` (`:75`) no cambia.

`migratePreservedSDDSessionPreflight` (`:103-133`) sustituye entero el bloque con marcadores de un prompt de OpenCode preservado, así que no hace falta un fragmento heredado nuevo en `inject.go`.

### D6. Una sola fuente para `Delivery Strategy`; el cuerpo de reserva desaparece

**Elección**:
- El cuerpo canónico de `Delivery Strategy` (ver «Interfaces / Contratos») vive en `internal/assets/skills/_shared/sdd-orchestrator-sections.md:261-263` y todas las superficies lo referencian con `{{GENTLE_AI_SDD_SECTION:Delivery Strategy}}`: las 8 que ya lo hacen y las copias en línea de `kiro`, `codex` y `opencode` (`sdd-orchestrator.md`), además de `claude/sdd-orchestrator-workflow.md:98-108`, que ya admite plantillas (`:229` usa `Recovery Rule`).
- `composeFallbackSessionPreflight` (`internal/components/sdd/orchestrator.go:137-158`) **deja de sustituir** el cuerpo de `### Delivery Strategy`: se elimina la tupla de `:152`. Se conservan los dos encabezados en la comprobación de cardinalidad (`:144`) porque `### Delivery Strategy` sigue siendo el ancla final de la tupla `### Artifact Store Mode` (`:151`) y `### Chain Strategy` garantiza que la sección de cadena sobrevive.
- `"Delivery Strategy"` se añade a `sharedOrchestratorSectionNames` (`orchestrator_shared_sections_test.go:103`), de modo que `TestSharedOrchestratorSectionsHaveOneSource` impide reintroducir una copia en línea y `TestEveryRuntimeRendersTheSharedSections` comprueba que todos los runtimes, también los de reserva, la renderizan.

**Alternativas consideradas**: (a) reescribir el cuerpo de reserva de `orchestrator.go`, como sugería la propuesta; (b) editar en línea las cuatro copias.
**Justificación**: el cuerpo de reserva existía para quitar a los runtimes de reserva el «ask once for and cache delivery strategy» de la plantilla. El cuerpo nuevo no pregunta nada, así que mantenerlo duplicaría la doctrina en Go y en Markdown, y además haría fallar `TestEveryRuntimeRendersTheSharedSections` al registrar la sección como compartida. El ratchet de `Delivery Strategy` baja de 4 a 1.

### D7. `Chain Strategy`: sustitución idéntica, sin converger

**Elección**: se conservan las 6 variantes (plantilla en qwen, codex, hermes, generic, cursor y gemini; en línea en antigravity, kimi, kiro, opencode y windsurf, más el flujo de Claude, que el ratchet no cuenta). En todas se aplica la **misma** sustitución textual de la frase inicial y del cierre (texto exacto en «Interfaces / Contratos»). Las coletillas propias de cada runtime («Kimi custom-agent prompt context», «Kiro phase context», «inline phase context», «dynamic subagent context», la ruta `question` de OpenCode) no se tocan.
**Alternativas consideradas**: converger todas a la plantilla.
**Justificación**: `inject_test.go:2569-2614` exige esas coletillas por runtime y la spec «Non-Claude Chain Strategy Guidance Parity» las ampara. Una sustitución idéntica no puede aumentar el número de variantes; converger sería otro cambio, ajeno a este.

### D8. Retirada de `Review Workload Guard`

**Elección**:
- Se eliminan el encabezado y el cuerpo en los 12 `sdd-orchestrator.md` (en gemini y qwen es encabezado más plantilla; en los otros diez, en línea), en `claude/sdd-orchestrator-workflow.md:138-153` y el bloque `<!-- sdd-orchestrator-section:Review Workload Guard (MANDATORY) -->` de `sdd-orchestrator-sections.md:405-420`.
- Lo que valía del guard (reenviar `delivery_strategy`/`chain_strategy` a `sdd-apply` y rechazar un valor desconocido) pasa al cuerpo canónico de `Delivery Strategy`.
- La frase «The gatekeeper runs in addition to the Review Workload Guard and the Mandatory Delegation Triggers» (`sdd-orchestrator-sections.md:254` y las copias en línea de `Automatic Mode Gatekeeper`) se sustituye de forma idéntica en todas por «The gatekeeper runs in addition to the Mandatory Delegation Triggers».
- `claude/sdd-orchestrator.md:68` deja de listar «review workload guard» en la superficie perezosa.
- `orchestrator_drift_ratchet_test.go:28`: se elimina la entrada (el test exige retirarla a propósito cuando la sección desaparece, `:92-95`). `Delivery Strategy` pasa a 1. `Chain Strategy` (6) y `Automatic Mode Gatekeeper (MANDATORY)` (8) no cambian.
- `sdd-phase-common.md` §E (`:109-125`) pasa a «## E. Delivery Strategy»: dominio, alias, valor inválido y precedencia (D1 y D2), sin presupuesto ni líneas de guarda.

**Alternativas consideradas**: dejar el encabezado con un cuerpo mínimo.
**Justificación**: una sección «MANDATORY» vacía es ruido normativo; su única función residual (el reenvío) pertenece a la sección de entrega.

### D9. Previsión de `sdd-tasks` y aviso de cambio grande

**Elección**:
- `sdd-tasks` sustituye «Review Workload Forecast» por **«## Delivery Forecast»**, cerca del principio de `tasks.md`, con solo: tamaño estimado (`S | M | L | XL`), líneas cambiadas estimadas (rango aproximado de adiciones más eliminaciones, sin ficheros generados) y unidades de trabajo (una por commit). Con `delivery_strategy: chained`, cada unidad indica la porción de PR a la que pertenece y su rama base. No hay campos de riesgo, decisión, estrategia ni número de umbral, y nunca se escriben `Delivery strategy` ni `Chain strategy` como campos (Q6).
- **Quién compara**: el orquestador, nunca la fase. En SDD, inmediatamente después de que `sdd-tasks` devuelva su previsión; en ODD, al crear el documento de la feature, que es cuando ODD hace su previsión.
- **Dónde lee la clave**: directamente de `axiom.yaml` del proyecto (en SDD, el de la carpeta maestra que resuelve el status), igual que el pre-vuelo lee `workspace.topology` (`session_preflight.go:18`). Si la clave falta, o su valor no es un entero mayor que cero, no hay aviso y no se cita ningún número; el valor inválido ya lo rechaza `axiom workspace validate`.
- **Comparación**: se compara el extremo superior del rango estimado con `split_hint_lines`; si lo supera, el orquestador emite un único aviso: el cambio es grande y la persona puede pedir PRs encadenadas. Después continúa sin esperar respuesta, también en `continuous`.
- **«Una vez por cambio»**: una vez por cambio y sesión del orquestador (SDD) o por feature (ODD, ligada a la creación del documento). No se persiste ninguna marca: repetir el aviso en una sesión nueva tras regenerar las tareas es inocuo porque nunca bloquea.

**Alternativas consideradas**: (a) que `sdd-tasks` lea la clave y emita el aviso; (b) proyectar `delivery` en `axiom sdd status --json` o en `axiom workspace context --json`; (c) persistir «avisado» en `tasks.md`.
**Justificación**: (a) devolvería a la fase una decisión de interacción que pertenece al orquestador; (b) está fuera de alcance por la propuesta (queda para C-06-e) y la lectura directa ya es el patrón de la topología; (c) contradice Q6 y no aporta nada a un aviso no bloqueante.

### D10. Bloque `delivery:` en `internal/workspace`

**Elección**: nuevo `internal/workspace/delivery.go`, espejo de `sdd.go` (porque la clave no es una cadena; `decodeStringKeys` solo sirve para escalares de texto):
- `DeliveryConfig{SplitHintLines int}`: el valor 0 significa «sin aviso».
- `deliveryKeys = []string{"split_hint_lines"}`.
- `decodeDelivery(*yaml.Node) (*DeliveryConfig, error)`: rechaza un bloque que no es un mapeo, incluido el nulo (`delivery:` sin contenido), las claves desconocidas (`unknownKeys`) y las repetidas, y acumula todos los errores con `errors.Join`. Un mapeo vacío (`delivery: {}`) es válido y equivale a no declarar la clave.
- `deliveryPositiveInt(label, node)`: solo un escalar con etiqueta `!!int` (no `"600"`, `6e2`, `true`, nulo, lista ni mapeo), decodificado a `int` (un desbordamiento es un error) y mayor que cero. Todos los mensajes nombran `delivery.split_hint_lines`.
- `Declarations` gana `Delivery *DeliveryConfig` y `DeliveryErr error` (`declarations.go:31-44`), con una rama en `readDeclarationBlocks` (`:59-96`).
- `joinLoadErrors` añade `DeliveryErr` (`loader.go:43`), de modo que `axiom workspace validate` y la carga fallan por el mismo canal que `workflow:`.
- `WorkspaceConfig.Delivery *DeliveryConfig `yaml:"-"`` (`types.go:70-78`) se proyecta tal como se declaró (nil sin bloque), como `Repositories` (`loader.go:113-117`).
- `ownMarkers.declares()` (`internal/hub/init.go:65-73`) añade `d.Delivery != nil || d.DeliveryErr != nil`; el comentario de `:57-60` y el mensaje de `:200` nombran `delivery:`.
- Los comentarios de lista de bloques de `loader.go:13-16` y `:69-71` y de `readDeclarationBlocks` (`declarations.go:55-58`) se actualizan.

**Alternativas consideradas**: decodificar con `node.Decode(&DeliveryConfig{})`.
**Justificación**: `yaml.v3` no ofrece `KnownFields` en `Node.Decode` (`declarations.go:46-48`) y aceptaría `"600"` entre comillas. El recorrido explícito es el patrón del paquete.

**Corrección a la propuesta (no bloqueante)**: el plan de reversión dice que un `axiom.yaml` con `delivery:` «no es válido para un binario revertido (lista cerrada de bloques)». No es así: las claves de primer nivel fuera de los bloques reconocidos se ignoran (`loader.go:69-71`; REQ-2.1 en `openspec/specs/workspace-topology/spec.md:98`), y el manual lo dice de cada bloque («Los binarios anteriores ignoran el bloque, como el de `git:`», `docs/manual-de-inicio.md:356`, `:377` y `:394`). La sección «Capacidades» de la propia propuesta ya lo recoge bien («un binario anterior lo ignora»). Tras revertir no hace falta retirar el bloque.

### D11. Copias de skills y su paridad

**Elección**:
- `work-unit-commits`: se reescribe la copia embebida (canónica) y se copia byte a byte a `skills/work-unit-commits/SKILL.md`. Hoy divergen en dos líneas (`internal/assets/skills/work-unit-commits/SKILL.md:33` y `:74` no están en `skills/`).
- `comment-writer`: las dos copias son hoy idénticas; se cambia el ejemplo de `:64` en ambas.
- Nuevo test de paridad byte a byte en `internal/assets` para `work-unit-commits` y `comment-writer`, con el patrón de `rdd_defect_workflow_skill_test.go:11-25` (`Read` del asset embebido frente a `os.ReadFile("../../skills/<skill>/SKILL.md")`).
- `chained-pr`: el `description` nuevo se refleja **a mano** en `AGENTS.md:51` en el mismo commit; `TestRepoAgentsIndexMatchesVersionedSkills` (`internal/skillregistry/repo_agents_index_test.go:24-55`) lo verifica. No se ejecuta `axiom skill-registry refresh` porque exige compilar el binario (memoria del proyecto: Cortex XDR borra binarios sueltos).
- `skills/branch-pr`: se retiran el punto 3 de `:23`, la casilla de `:126`, el ejemplo de `:274` y el número de la fila de `:143`. El punto 2 (`:22`, gobierno de la etiqueta protegida `size:exception`) se conserva, como `issue-creation`, que está fuera de alcance. La fila del check del CI remite a la política de tamaño del repositorio sin citar número (riesgo transitorio aceptado hasta C-02-e).

**Alternativas consideradas**: sin test de paridad.
**Justificación**: la deriva actual de `work-unit-commits` demuestra que, sin guarda, las copias se separan. El test es barato y fija el criterio de éxito «las dos copias coinciden».

### D12. Guarda contra umbrales residuales

**Elección**: un test nuevo en `internal/assets` recorre `assets.FS` y `../../skills` y falla ante cualquier línea que contenga `400` como palabra (`\b400\b`) o los literales retirados (`400-line budget risk`, `Decision needed before apply`, `Chained PRs recommended`, `workload-decision-required`, `Review Workload Guard`, `review_budget_lines`). Residuo admitido, por ruta **y** contenido:
- líneas con `Full-4R review` en `*/agents/jd-judge-*`, `*/agents/review-*` y `opencode/sdd-overlay-{single,multi}.json` (barrido de los jueces y lentes RDD);
- cualquier línea de `skills/rdd-defect-workflow/` (sus dos copias; las retira C-01-e).

Los textos generados desde Go se vigilan en su paquete: `routing_test.go` (sin `400`, `ask-on-risk`, `auto-chain` ni `exception-ok` en `RenderRouting` de cada agente), `session_preflight_test.go` (sin `400` en las dos variantes del bloque) y `sdd_preflight_hook_test.go` (bloque de 5 líneas). Los literales de migración de `inject.go:1543,1560` (Q4) y `internal/reviewtransaction/risk.go:22` no son assets y quedan fuera del recorrido.

**Alternativas consideradas**: un test que ejecute `rg`.
**Justificación**: un test en proceso no depende de herramientas externas y forma parte de `go test ./...`. La lista blanca por ruta y contenido impide que un `400` nuevo se cuele en un fichero admitido.

## Flujo de datos

```
                 axiom.yaml (delivery.split_hint_lines?)
                        │  lectura directa (D9)
                        ▼
persona ──chat «encadena»──► orquestador ──cache de sesión: chained + chain_strategy (D2-1)
                               │   ▲
  tasks.md heredado ──alias────┘   │ (D2-2, solo al reanudar)
                               │
  hook / plugin ──bloque 5 líneas (Delivery strategy: single-pr) (D2-3)
                               │
          ┌────────────────────┴─────────────────────┐
          ▼                                          ▼
     sdd-tasks                                  sdd-apply
  recibe delivery_strategy                 recibe delivery_strategy
  (+ chain_strategy si chained)            (+ chain_strategy si chained)
  escribe «Delivery Forecast»              usa la estrategia solo para
  (tamaño, rango, unidades)                elegir la rama destino; nunca para
          │                                 por carga
          ▼
  orquestador compara el extremo superior del rango
  con split_hint_lines ──► un aviso no bloqueante ──► continúa
```

En ODD el flujo es el mismo sin fases: el agente prevé las líneas al crear `odd/tasks/<feature>.md`, compara con la clave y avisa una vez. La entrega es `single-pr` salvo petición.

## Cambios de ficheros

### Unidad 1: `delivery:` en `axiom.yaml`

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/delivery.go` | Crear | `DeliveryConfig`, `deliveryKeys`, `decodeDelivery`, `deliveryPositiveInt` (D10) |
| `internal/workspace/delivery_test.go` | Crear | Tabla de casos espejo de `sdd_test.go`/`workflow_test.go` |
| `internal/workspace/declarations.go` | Modificar | Campos `Delivery`/`DeliveryErr`, rama en `readDeclarationBlocks`, comentarios |
| `internal/workspace/loader.go` | Modificar | `joinLoadErrors` (`:43`), proyección en `parseLayered` (`:105-118`), comentarios (`:13-16`, `:69-71`) |
| `internal/workspace/types.go` | Modificar | `WorkspaceConfig.Delivery` con `yaml:"-"` y su comentario |
| `internal/hub/init.go` (+ test de preservación existente en `internal/hub`) | Modificar | `declares()` y mensaje de `--force` con `delivery:` |
| `axiom.example.yaml` | Modificar | Bloque comentado `# delivery:` / `#   split_hint_lines: 600` con su explicación |
| `internal/workspace/example_yaml_test.go` | Modificar | El bloque de la plantilla, una vez descomentado, es válido (patrón de `:35-60`) |
| `docs/manual-de-inicio.md` | Modificar | Entrada del índice (`:24-25`) y sección «### Aviso de Cambio Grande (`delivery:`)» tras «Carril de Trabajo» |
| `internal/workspace/docs_contract_test.go` | Modificar | `TestDocsDocumentTheDeliveryBlock`, patrón de `TestDocsDocumentTheWorkflowLaneBlock` (`:213-252`) |

### Unidad 2: pre-vuelo sin estrategia de PR

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/cli/sdd_preflight_hook.go` | Modificar | D4 |
| `internal/cli/sdd_preflight_hook_test.go` | Modificar | `:150`, `:758` y usos de `pr-strategy`; casos nuevos de respuesta heredada ignorada |
| `internal/cli/sdd_preflight_hook_topology_test.go` | Modificar | `:92`, `:124`, `:254`: bloque de 5 líneas; `multirepo` sin grupos |
| `internal/assets/opencode/plugins/sdd-task-result-artifacts.ts` | Modificar | D4 (v1) |
| `internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts` | Modificar | D4 (v2) |
| `internal/components/sdd/opencode_plugin_digests.go` | Modificar | v1: `c6e03fc1c7a03725cc1645b8285018f5dacd53fba8eb03cd51dfb26fe63fc28e // 43094d2d`; v2: `8c8bf972f4334dba3f8c947ef9b82eb34d6e931828887fb41b31271bd333d155 // 43094d2d` (calculados con `git show HEAD:<asset> \| sha256sum` sobre `cb92574f`; recalcular si la base cambia) |
| `internal/components/sdd/opencode_runtime_test.go` | Modificar | `:101`: `sdd-task-result-artifacts.ts` pasa a `{"v1": 10, "v2": 5}` |
| `internal/assets/opencode_sdd_preflight_topology_test.go` | Modificar | `:49-51`, `:97` (constructor del bloque sin `Review policy`), `:107`, `:138`, `:175`, `:180` |
| `internal/sddtaskresult/preflight_question_harness_test.go` | Modificar | Arnés con node: `multirepo` sin preguntas; respuesta `pr-strategy` heredada ignorada; bloque `single-pr` |
| `internal/assets/opencode_v2_sdd_test.go` | Modificar | Si fija la tabla o la línea `Review policy` |
| `internal/components/sdd/session_preflight.go` | Modificar | D5 |
| `internal/components/sdd/session_preflight_test.go` | Modificar | `:25`, `:30`, `:120`, `:260`, `:304-345`; aserción de ausencia de `400` |
| `internal/components/sdd/inject_test.go` | Modificar | Cláusulas de frescura del pre-vuelo (`:1423-1424`, `:1914-1915`, `:2009-2010`) |
| `internal/components/sdd/delivery_strategy_vocabulary_test.go` | Modificar | Tests de productor (`:183`, `:223`, `:399`, `:430`): el pre-vuelo no ofrece grupo ni mapeo de estrategia; el único productor es el hook (`single-pr`, dentro del dominio). Los de migración (`:341`, `:453`) se conservan ajustando la expectativa del bloque reinyectado (`:510`) |
| `internal/assets/opencode/commands/sdd-{new,ff,continue,explore,init,apply,verify,archive,onboard}.md` | Modificar | Frase idéntica en las nueve; `sdd-apply.md:19,33` y `sdd-onboard.md:19` (D8) |
| `bench/journeys_issue_3336.go` | Modificar | `:18`: deja de exigir `- **PR strategy**` y `fixed at 400 changed lines`, y exige la frase de entrega fija. Es un módulo aparte (`bench/go.mod`): `go test ./...` desde la raíz no lo cubre |
| `testdata/golden/*` | Regenerar | Proyección del pre-vuelo en los runtimes de reserva |

### Unidad 3: doctrina ODD

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/components/agentguidance/routing.go` | Modificar | `:122` y `:130` (texto en «Interfaces / Contratos») |
| `internal/components/agentguidance/routing_test.go` | Modificar | Aserciones de la doctrina nueva y de ausencia de `400`, `ask-on-risk`, `auto-chain`, `exception-ok` y «Existing PR size gates» |
| `testdata/golden/*` | Regenerar | Proyecciones de la doctrina |

### Unidad 4: orquestadores, convención de fases y skills de fase

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | `Delivery Strategy` canónica (`:261-263`), `Chain Strategy` (`:265-274`), frase del gatekeeper (`:254`), fuera `Review Workload Guard` (`:405-420`) |
| `internal/assets/skills/_shared/sdd-phase-common.md` | Modificar | §E (`:109-125`) → «E. Delivery Strategy» |
| `internal/assets/claude/sdd-orchestrator-workflow.md` | Modificar | `:98-108` → plantilla; `:109-120` (cadena); fuera `:138-153` |
| `internal/assets/claude/sdd-orchestrator.md` | Modificar | `:68` |
| `internal/assets/{antigravity,codex,cursor,gemini,generic,hermes,kimi,kiro,opencode,qwen,windsurf}/sdd-orchestrator.md` y el resto de `*/sdd-orchestrator.md` | Modificar | `Delivery Strategy` → plantilla donde está en línea (kiro, codex, opencode); sustitución idéntica en `Chain Strategy` en línea; fuera `Review Workload Guard`; frase del gatekeeper |
| `internal/components/sdd/orchestrator.go` | Modificar | Se elimina la tupla de `:152`; se conserva la comprobación de `:144` |
| `internal/assets/skills/sdd-tasks/SKILL.md` | Modificar | `:38` (dominio), `:84-101`, `:149-183`, `:238-245`, `:262` (D9) |
| `internal/assets/skills/sdd-apply/SKILL.md` | Modificar | `:40` (dominio); el paso 2a (`:82-104`) se reescribe como «Step 2a: Follow the Delivery Strategy» (solo rama destino, sin parada; conserva «no minificar» sin número); fuera la regla `:333`; `:334` sin número. Se mantiene la numeración: el paso 2b no se renumera |
| `internal/assets/orchestrator_drift_ratchet_test.go` | Modificar | Fuera `Review Workload Guard`; `Delivery Strategy` a 1 |
| `internal/components/sdd/orchestrator_shared_sections_test.go` | Modificar | `Delivery Strategy` en `sharedOrchestratorSectionNames` |
| `internal/components/sdd/renderer_invariants_test.go` | Modificar | `:104-122`: centinelas nuevos de `Delivery Strategy` (`single-pr`, `chained`, `Pass \`delivery_strategy\``); fuera la entrada del guard |
| `internal/components/sdd/inject_test.go` | Modificar | `:2639-2650`: la sección de cadena acaba en el siguiente `### ` (no en el guard); frases nuevas |
| `internal/components/sdd/orchestrator_composition_test.go` | Modificar | Lo que fije del guard o de la entrega; `:543` (negativa) se conserva |
| `internal/components/sdd/delivery_strategy_vocabulary_test.go` | Modificar | Dominio leído de `sdd-apply`, `sdd-tasks` y `sdd-phase-common` con el patrón `` `single-pr | chained` ``; los tests del guard (`:265-339`) pasan a las secciones `Delivery Strategy`: cubren el dominio, enuncian los alias y rechazan un valor desconocido |
| `internal/components/sdd/planning_detail_test.go` | Modificar | `:23`: `## Delivery Forecast` y `Estimated changed lines` en lugar de `## Review Workload Forecast` y `400 changed-line review budget` |
| `testdata/golden/*` | Regenerar | Las 11 proyecciones de orquestador |

### Unidad 5: skills de entrega y documentación

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/assets/skills/chained-pr/SKILL.md` | Modificar | `description`, activación, reglas y tabla de decisión sin número; activación a petición o con `delivery_strategy: chained` |
| `internal/assets/skills/chained-pr/references/chaining-details.md` | Modificar | `:62` (fila «Review budget» fuera del contrato de salida), `:96` |
| `AGENTS.md` | Modificar | `:51`, idéntico al `description` nuevo |
| `docs/components.md` | Modificar | `:77`: «Planifica PR encadenadas cuando la persona las pide.» |
| `internal/assets/skills/work-unit-commits/SKILL.md` | Modificar | `:20`, `:32-33`, `:64`, `:66-74`: un commit por unidad dentro de una sola PR; cadena solo a petición; conserva «no code-golf» sin número |
| `skills/work-unit-commits/SKILL.md` | Modificar | Copia byte a byte de la embebida |
| `internal/assets/skills/comment-writer/SKILL.md`, `skills/comment-writer/SKILL.md` | Modificar | `:64`: ejemplo de tono sin presupuesto |
| `skills/branch-pr/SKILL.md` | Modificar | D11 |
| `internal/assets/skill_copies_parity_test.go` | Crear | Paridad de `work-unit-commits` y `comment-writer` (D11) |
| `testdata/golden/*` | Regenerar | Si alguna proyección de skills cambia |

### Unidad 6: guarda de residuos

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/assets/delivery_threshold_residue_test.go` | Crear | D12 |

### Sin cambio (verificado)

- `internal/components/sdd/inject.go:1543,1560` (Q4) y `:1568-1569`.
- `internal/reviewtransaction/risk.go:22` y `internal/cli/review_process_boundary_test.go:16` (comentario de la regla de tamaño de 4R): residuo RDD de C-01-e.
- `internal/assets/assets_test.go:1058`, `windsurf_workflow_test.go:13`, `sdd_exits_wording_test.go:30` y `orchestrator_composition_test.go:543`: prohibiciones que siguen siendo válidas.
- `issue-creation` (gobierno de `size:exception`), `docs/usage.md` e `internal/app/help.go`.
- `openspec/INDEX.md:2575`: lo regenera el archivado a partir de las specs vivas.
- Documentos históricos (`docs/redesign/README.md`, `docs/architecture/rdd-*.md`, `docs/releases/*`, `odd/tasks/*`): registran decisiones pasadas; no son doctrina vigente.

## Interfaces / Contratos

### Go: bloque `delivery:`

```go
// DeliveryConfig es el bloque `delivery:` de axiom.yaml.
// SplitHintLines es el umbral del aviso no bloqueante de cambio grande;
// 0 significa que la clave no se declaró y no hay aviso.
type DeliveryConfig struct {
	SplitHintLines int
}

// deliveryKeys es la lista cerrada de claves del bloque `delivery:`.
var deliveryKeys = []string{"split_hint_lines"}

// decodeDelivery rechaza un nodo que no es un mapeo (incluido el nulo), las
// claves desconocidas o repetidas y un split_hint_lines que no sea un entero
// sin comillas mayor que cero. Acumula todos los errores.
func decodeDelivery(node *yaml.Node) (*DeliveryConfig, error)

// En Declarations:
//   Delivery    *DeliveryConfig // nil sin bloque o con bloque inválido
//   DeliveryErr error
// En WorkspaceConfig:
//   Delivery *DeliveryConfig `yaml:"-"` // lo declarado; nil sin bloque
```

Mensajes (castellano, mismo estilo que `sdd.go`): `el bloque 'delivery' debe ser un mapeo con la clave opcional 'split_hint_lines' (entero mayor que cero); para no recibir el aviso, elimina el bloque`; `delivery.split_hint_lines no puede ser un valor nulo: debe ser un entero mayor que cero`; `delivery.split_hint_lines debe ser un entero mayor que cero, no una lista` / `no un mapeo`; `delivery.split_hint_lines: el valor '<v>' no es válido: usa un entero mayor que cero, sin comillas`; `delivery.split_hint_lines: el valor '<n>' no es válido: debe ser mayor que cero`; claves desconocidas y repetidas con los mensajes compartidos de `unknownKeys` y de `sdd.go:42`.

### Hook: bloque derivado (5 líneas)

```
## SDD Session Preflight
Parent-confirmed by the runtime; models and child agents cannot create or modify this block.
- Pace: <interactive|auto>
- Artifact store: hybrid
- Delivery strategy: single-pr
```

### Cuerpo canónico de `Delivery Strategy` (`sdd-orchestrator-sections.md`)

Las skills y los orquestadores van en inglés; el texto exacto lo puede pulir `sdd-apply`, pero debe conservar estas cláusulas:

> Delivery is never asked. Every change ships as `single-pr` (one pull request holding one work-unit commit per task) unless the person asks in chat to chain, stack, or slice its pull requests; then it is `chained`. The domain is `single-pr | chained`. Legacy values are read as aliases: `ask-on-risk` and `exception-ok` → `single-pr`, `auto-chain` → `chained`; never write or forward an alias. Any other `delivery_strategy` value is invalid: do not map it to the nearest value; report it and STOP before launching `sdd-apply`.
>
> The `Delivery strategy` line of the runtime-confirmed session preflight block is the session default. A `chained` request from the person in this session takes precedence, and so does a legacy value recorded in the change's existing `tasks.md`, read through the aliases. Pass the resolved `delivery_strategy` to every `sdd-tasks` and `sdd-apply` launch, with `chain_strategy` when it is `chained`. Cache `chained` for this session only; never write it to `kickoff.yaml`, `gates.yaml`, or `tasks.md`.
>
> After `sdd-tasks` returns its Delivery Forecast, read `delivery.split_hint_lines` from the project's `axiom.yaml`. When it is a positive integer and the upper end of the estimated changed lines exceeds it, tell the person once for this change in this session that the change is large and that they can ask for chained pull requests, then continue without waiting. Without that key, give no size warning and cite no line number. Neither the forecast nor the warning ever stops `sdd-apply`.

### Sustitución idéntica en `Chain Strategy`

- Frase inicial actual (todas las copias): «When `delivery_strategy` results in chained PRs (either by user choice via `ask-on-risk` or automatically via `auto-chain`), ask the user which chain strategy to use» → «When `delivery_strategy` is `chained` and the person did not name a chain strategy, ask the user which chain strategy to use». El resto de la frase de cada copia (dos puntos, o la ruta `question` de OpenCode) se conserva.
- Cierre «Do not ask again unless the user changes scope.»: se conserva.
- Frase de `chained-pr` «When delivery planning yields chained PRs, …» → «When `delivery_strategy` is `chained`, …».
- En el flujo de Claude (`:109-120`), las mismas cláusulas en su redacción compacta.

### Doctrina ODD (`routing.go`)

- `:122` → «Size each ODD task as the smallest coherent behavior with its tests and docs; there is no line-count target, cap, or forced split. Never delete spaces, blank lines, or comments for cosmetic line savings; never omit tests, minify, add gratuitous abstractions, or split artificially. Forward this same instruction when delegating tasks to subagents.»
- `:130` → «Delivery follows work units. By default a feature ships as `single-pr`: one pull request holding one work-unit commit per task. At feature-document creation, forecast the approximate authored changed lines (additions plus deletions, generated files excluded) from the task list. When the project's `axiom.yaml` declares `delivery.split_hint_lines` and the forecast exceeds it, tell the person once for this feature that the change is large and that they can ask for chained pull requests, then continue without waiting; without that key, give no size warning and cite no line number. Use `chained` only when the person asks to chain, stack, or slice the pull requests: if they did not name a chain strategy, ask once whether `stacked-to-main` or `feature-branch-chain`, cache it, and record slice boundaries, which commits each pull request holds, in the feature document. Resolve the `work-unit-commits` skill by registry name for every feature and the `chained-pr` skill only when chaining, never hardcoding their paths.»

El bullet es autocontenido: C-02-d puede añadir sus modos de aviso y parada por commit con una frase nueva al final, sin reescribirlo.

### `sdd-tasks`: plantilla de la previsión

```markdown
## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | S / M / L / XL |
| Estimated changed lines | <approximate range, additions + deletions, generated files excluded> |

### Work Units

| Unit | Goal | Commit | PR slice and base (only when `delivery_strategy` is `chained`) |
|------|------|--------|------------------------------------------------------------------|
```

### Comandos de OpenCode

Frase idéntica en los nueve `sdd-*.md`: «SDD Session Preflight must already be complete for this session: the pace the topology asks (the artifact store is always `hybrid` and delivery is `single-pr` unless the person asks for chained PRs).» `sdd-apply.md:19` → «5. Pass the resolved `delivery_strategy`, and `chain_strategy` when it is `chained`; delivery never blocks apply.» `sdd-apply.md:33` → «- The resolved `delivery_strategy` and, when `chained`, its `chain_strategy`.» `sdd-onboard.md:19`: se elimina «and enforce the review budget before apply».

## Estrategia de pruebas

TDD desactivado (`openspec/config.yaml`: `strict_tdd: false`); runner `go test ./...` en proceso, sin compilar binarios sueltos.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitaria (`internal/workspace`) | `decodeDelivery`: válido (`600`, `1`, `delivery: {}`), y rechazos con `delivery.split_hint_lines` en el mensaje: `0`, `-1`, `"600"`, `6e2`, `1.5`, `true`, nulo, lista, mapeo, clave desconocida, clave repetida, bloque nulo o escalar; desbordamiento de `int`. Sin bloque, `Declarations.Delivery == nil` y sin error. Proyección en `WorkspaceConfig` | Tabla con subtests, espejo de `sdd_test.go` |
| Unitaria (`internal/hub`) | `axiom init --force` conserva un `axiom.yaml` que declara `delivery:`, válido o no | Caso nuevo en el test de preservación existente |
| Contrato de documentación | Sección, índice y ejemplo de `delivery:`; plantilla válida al descomentarla | `docs_contract_test.go`, `example_yaml_test.go` |
| Unitaria (`internal/cli`) | Bloque de 5 líneas; `monorepo-embedded` solo `pace`; `multirepo` sin transcripción; transcripción con `pr-strategy` respondida («Ask me», «Auto», marcador nominal y numerado `3/3:`) aceptada y con `single-pr`; `validSDDPreflightHookBlock` rechaza un bloque de 6 líneas o `Delivery strategy: ask-on-risk`; siguen las denegaciones de hijo y de bloque escrito por el modelo | `sdd_preflight_hook_test.go`, `sdd_preflight_hook_topology_test.go` |
| Integración (plugin real con node) | Las mismas expectativas en v1 y v2; paridad de tablas TS/Go; digests históricos (10 y 5) | `preflight_question_harness_test.go`, `opencode_sdd_preflight_topology_test.go`, `opencode_runtime_test.go` |
| Unitaria (`internal/components/sdd`) | Texto del pre-vuelo (con y sin herramienta), sin `400`; vocabulario: dominio `single-pr \| chained` coherente en tres fuentes, alias enunciados, valor desconocido rechazado y migraciones conservadas; renderizado de `Delivery Strategy` compartida en los 13 runtimes; cadena por runtime; previsión de `sdd-tasks` | Tests listados en «Cambios de ficheros» |
| Unitaria (`internal/assets`) | Ratchet; paridad de copias de skills; guarda de residuos (D12) | Tests nuevos y actualizados |
| Unitaria (`agentguidance`) | Doctrina ODD nueva por agente | `routing_test.go` |
| Golden | 11 proyecciones de orquestador y las de doctrina o skills que cambien | `go test ./internal/components/ -run 'TestGolden' -update` (flag `update` en `internal/components/golden_test.go:32`); revisar el diff, que solo debe tocar estas secciones |
| Índice de skills | `AGENTS.md` coherente con el `description` de `chained-pr` | `TestRepoAgentsIndexMatchesVersionedSkills` |
| Módulo `bench` | La aserción de la *journey* 3336 compila | `go vet ./...` dentro de `bench/` (no compila binario); la *journey* ejecuta el binario real y no se lanza en este cambio |

Verificación final de cada unidad: `go test ./...` y `go vet ./...` desde la raíz del worktree.

## Matriz de amenazas

El cambio toca integración de proceso (el hook `PreToolUse` y el plugin de OpenCode) y doctrina de ramas y PR, pero no añade ni modifica ningún comando git, `gh`, subproceso ni clasificación de ficheros ejecutables.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta de diseño | Tests RED previstos |
|---|---|---|---|---|
| Rutas tipo documentación | `requirements.txt`, `CMakeLists.txt`, Markdown/MDX ejecutable, `README.sh` | N/A: no se cambia ninguna clasificación de ficheros ni su ejecución | — | — |
| Selección de repositorio git | `git -C`, rutas relativas, absolutas | N/A: ningún código nuevo invoca git; la doctrina sigue remitiendo a las rutas absolutas de `axiom branch ensure` | — | — |
| Estado del commit | staged, `commit -a`, índice vacío | N/A: no hay automatización de commits | — | — |
| Estado del push | rama con seguimiento, primer push, refspec explícito | N/A: el push sigue siendo de la persona; `chained` solo elige la rama destino de commits locales en `sdd-apply` | — | — |
| Comandos de PR | `--head` explícito, prefijo de entorno, comandos compuestos | N/A: no se automatiza ninguna PR; `chained-pr` sigue siendo guía para la persona | — | — |

Frontera del hook (fuera de la matriz, pero con pruebas de regresión explícitas): la autoridad del pre-vuelo no se relaja. Se conservan la denegación de un lanzamiento hijo (`sdd_preflight_hook.go:148-150`) y la de un prompt con `## SDD Session Preflight` escrito por el modelo (`:151-153`). Una respuesta heredada `pr-strategy` nunca puede producir un valor distinto de `single-pr`, porque el campo no tiene grupo y su dominio permitido es solo ese.

## Migración y despliegue

No hay migración de datos: no se reescribe ningún `axiom.yaml`, `kickoff.yaml`, `gates.yaml` ni `tasks.md`.

- **Desfase de versiones durante la actualización**: un hook o plugin antiguo puede seguir emitiendo un bloque de 6 líneas con `ask-on-risk` o `auto-chain`. El orquestador y las fases nuevos lo leen con los alias (D1) e ignoran `Review policy`. Un hook nuevo con orquestadores antiguos (instalación sin `axiom sync`) emite `single-pr`, que el guard antiguo ya conocía.
- **Instalaciones de OpenCode**: `axiom sync` sustituye el plugin gestionado porque su digest anterior queda registrado como histórico; sin ese registro, la sustitución segura de v2 se negaría.
- **Prompts de OpenCode preservados**: el bloque con marcadores del pre-vuelo se reemplaza entero (`migratePreservedSDDSessionPreflight`).
- **Cambios en curso**: su `tasks.md` antiguo se lee con alias al reanudar (D2, nivel 2). `sdd-apply` deja de detenerse por la previsión antigua porque ya no la lee.
- **Reversión**: revertir los commits en orden inverso y ejecutar `axiom sync`. Un `axiom.yaml` que declare `delivery:` sigue siendo válido para el binario revertido, que ignora las claves de primer nivel no reconocidas (corrección de D10).
- **CI**: el check de tamaño de `.github/workflows/pr-check.yml` sigue activo hasta C-02-e; esta PR necesita la etiqueta `size:exception`.

## Unidades de trabajo (commits)

Una PR con `size:exception`; commits convencionales en castellano, sin atribución a IA. Cada commit deja `go test ./...` y `go vet ./...` en verde y regenera los goldens que su cambio afecte.

| # | Commit | Contenido | Por qué en este orden |
|---|---|---|---|
| 1 | `feat(workspace): bloque delivery con split_hint_lines en axiom.yaml` | Unidad 1 | Independiente; fija el esquema que nombran la doctrina y los orquestadores |
| 2 | `refactor(sdd): pre-vuelo sin estrategia de PR y bloque de cinco líneas` | Unidad 2, incluidos los digests, los comandos de OpenCode, `bench` y los tests de productor de vocabulario | El pre-vuelo es autónomo; los tests de guard del vocabulario siguen leyendo los orquestadores antiguos, todavía coherentes |
| 3 | `refactor(odd): entrega single-pr por defecto en la doctrina ODD` | Unidad 3 | Solo depende del nombre de la clave (commit 1) |
| 4 | `refactor(sdd): entrega single-pr en orquestadores y fases sin guard de carga` | Unidad 4 | Atómico a propósito: el test de vocabulario deriva el dominio de `sdd-tasks`/`sdd-apply` y lo exige en los orquestadores; separarlo dejaría el árbol en rojo |
| 5 | `docs(skills): chained-pr a petición y work-unit-commits sin presupuesto` | Unidad 5 | Necesita los textos de fase del commit 4 para ser coherente; incluye `AGENTS.md` por el test del índice |
| 6 | `test(assets): guarda contra umbrales de entrega residuales` | Unidad 6 | Solo pasa cuando los commits 2 a 5 han retirado todo el residuo |

Fuera de este cambio: C-02-d (modos de aviso y parada por commit), C-01-e (residuo RDD) y C-02-e (check del CI, plantilla de PR, memoria local y preferencia de Engram).

## Preguntas abiertas

Ninguna bloqueante. Hallazgos para las fases siguientes:

- [ ] `bench/journeys_issue_3336.go:18` no figuraba en la propuesta; se incluye en la unidad 2 porque fija el texto retirado del pre-vuelo. Vive en un módulo aparte y su *journey* ejecuta el binario real: no se lanza en este cambio.
- [ ] La propuesta cita `opencode/commands/sdd-*.md`, pero no `sdd-apply.md:19` («Review workload guard must have passed») ni `sdd-onboard.md:19` («enforce the review budget»); se incluyen (D8).
- [ ] `claude/sdd-orchestrator.md:68` y la frase del gatekeeper (`sdd-orchestrator-sections.md:254` y sus copias en línea) nombran el guard retirado; se incluyen (D8).
- [ ] `generic/sdd-orchestrator.md` lleva `Review Workload Guard` en línea (`:106-117`), no por plantilla como decía la exploración; da igual, porque se elimina.
- [ ] `internal/cli/review_process_boundary_test.go:16` contiene «400-line size rule» dentro de `internal/cli` (ruta del criterio `rg`): es un comentario de la regla de tamaño de 4R ligada a `risk.go:22`. Debe tratarse como residuo RDD de C-01-e; la spec puede nombrarlo junto a `risk.go`.
- [ ] Riesgo bajo: un `tasks.md` heredado con `auto-chain` y `Chain strategy: pending` (o un bloque antiguo con `auto-chain`) se lee como `chained` por Q1 y provoca una única pregunta por la estrategia de cadena aunque la previsión antigua fuera baja. Es transitorio y no bloquea.
