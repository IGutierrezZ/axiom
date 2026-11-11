# Diseño: Revisión independiente en SDD (sdd-revision-independiente)

> Fase `design` · rama `inc/sdd-revision-independiente` · 2026-11-11 · kickoff `continuous`, rol único `fullstack` (`blocking`).
> Entradas: `proposal.md` (D1–D8 ratificadas; P1 = (c), P2 = (b)) y las specs delta en curso (`sdd-independent-review`, `sdd-preflight-configuration`, `sdd-lane-kickoff`, `sdd-block-review-gates`), cuyos supuestos A-1 a A-4 se fijan aquí (§2, DD-07 a DD-09).
> Contexto técnico: la proyección de `axiom sdd status` no declara `technicalContext`, así que no hay `context-impact.md`.

## 1. Enfoque técnico

El cambio tiene dos mitades de naturaleza distinta y se diseñan por separado:

1. **Go, mínimo y en el borde del comando** (`internal/kickoff`, `internal/cli`): una validación de sellado `continuous` ⇒ roster exactamente `[fullstack]` que solo ejecuta el camino explícito de `axiom sdd kickoff seal`. La lectura (`ParseKickoff`, `Load`), la primitiva de persistencia (`Seal`) y el retrosellado (`InferKickoff`) no cambian de comportamiento (D7, D8, P2).
2. **Prosa del orquestador y de las skills** (D1): una sección compartida nueva, `SDD Independent Review`, en `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, proyectada a los doce orquestadores; una referencia nueva de `sdd-verify` para el cumplimiento técnico; una sección de objetivo SDD por rol en `judgment-day`; y un asset compartido con el mandato de los revisores de cobertura. Ningún estado, campo de ledger ni puerta nuevos en Go.

La preferencia de revisión independiente (P1 = (c)) se recoge en una **interacción propia, inmediatamente después** del pre-vuelo agrupado ya validado, y no como cuarto grupo: el pre-vuelo canónico, su hook corroborador y sus plugins quedan intactos (DD-04).

## 2. Decisiones de arquitectura

### DD-01. La regla de sellado vive en el borde del comando, no en `Validate()` ni en `Seal()`

**Elección.** Nuevo método `func (k Kickoff) ValidateForSeal() error` en `internal/kickoff/schema.go`: llama a `k.Validate()` y, salvo que `k.SealedBy == SealedByInferred`, aplica `checkContinuousRoster(style, roles)`. Lo usan:

- `ParseSealArgs` (`internal/kickoff/args.go`), al final de la rama sin `--infer`, sobre el estilo efectivo y el roster ya completado con el `fullstack` por defecto (línea 281). Es puro: rechaza antes de resolver `--change` contra el sistema de ficheros y antes de escribir nada.
- `runSDDKickoffSeal` (`internal/cli/sdd_kickoff.go`), justo antes de `kickoff.Seal`, sobre el `Kickoff` ya ensamblado (defensa en profundidad para cualquier productor futuro; el camino `--infer` queda exento por `SealedBy`).

`Seal()` sigue llamando solo a `Validate()`.

**Alternativas descartadas.**

- *En `Validate()`*: `ParseKickoff` dejaría de leer cualquier `kickoff.yaml` ya sellado o archivado que la incumpla (D7).
- *En `Seal()` con exención de `inferred`*: `Seal` es la primitiva de persistencia que comparten el sellado explícito, el retrosellado y los fixtures de prueba que modelan kickoffs ya existentes. Tres de ellos sellan hoy `continuous` con varios roles (`internal/branch/engine/snapshot_test.go:55`, `internal/branch/engine/status_test.go:198`, `internal/cli/branch_test.go:124`, con `SealedBy: "prueba"`); obligarlos a cambiar de estilo mezcla una regla de intención del usuario con la persistencia.

**Justificación.** Solo hay dos productores de `Kickoff` en producción (`SealArgs.ToKickoff` en `args.go:298` e `inferredKickoff` en `infer.go:83`, verificado con búsqueda en `internal/` y `cmd/`), y ambos pasan por `runSDDKickoffSeal`. La regla es de intención de sellado, así que se comprueba donde la intención entra.

**Cambio de comportamiento: `seal` repetido.** Hoy, repetir `axiom sdd kickoff seal` sobre un cambio ya sellado es idempotente: `Seal` relee el documento ganador y responde «Kickoff ya estaba sellado». Con la regla en `ParseSealArgs`, repetir un `seal` con los argumentos antiguos `--execution-style continuous` y roles especializados (o `--from-session-pace auto` con esos roles) se rechaza antes de llegar a `Seal`, aunque el cambio ya tuviera un kickoff sellado así antes de esta regla. El kickoff en disco no se toca y se sigue leyendo; para consultarlo, `axiom sdd kickoff show`. Se acepta porque el rechazo ocurre antes de cualquier lectura o escritura (la validación de argumentos es pura) y porque reintentar un sellado que hoy sería inválido no debe parecer correcto. La alternativa de comprobar la regla solo cuando no existe `kickoff.yaml` se descarta: haría que el mismo comando fuera válido o inválido según el estado del disco.

**Restricción de vocabulario en `internal/kickoff`.** `internal/kickoff/rdd_boundary_test.go` prohíbe en ese paquete el vocabulario reservado al otro mecanismo de revisión (las cinco palabras que su propia prueba ensambla por fragmentos para no escribirlas seguidas) y, según su comentario, la búsqueda en texto plano tampoco debe encontrarlas en los ficheros de prueba del paquete. El código, los comentarios, los mensajes y las pruebas nuevas de `internal/kickoff` (incluido `ErrContinuousRequiresFullstack` y su texto) no las usan.

### DD-02. La regla compara la identidad de los roles, no su política de compuerta

**Elección.** `continuous` exige `len(roles) == 1 && roles[0].Role == "fullstack"`. La política (`blocking|deferred|optional`) no se comprueba: `continuous` no tiene compuertas, así que no tiene efecto. Un roster repetido (`--role fullstack --role fullstack`) no es exactamente `[fullstack]` y se rechaza con el mismo mensaje.

**Alternativa descartada.** Exigir también `fullstack:blocking`: añadiría un rechazo sin efecto observable y un tercer caso de mensaje.

**Justificación.** Interpretación literal de D7 («el roster debe ser exactamente `[fullstack]`»), alineada con REQ-28.12 de la spec.

### DD-03. El retrosellado queda exento por construcción y por marca, sin nota de estado (P2 = (b))

**Elección.** `InferKickoff` no cambia de comportamiento: sigue sellando `continuous` con los roles detectados y `sealed_by: inferred`. Se añade la constante `SealedByInferred = "inferred"` (usada por `inferredKickoff` y por `ValidateForSeal`) y un párrafo en el comentario de `InferKickoff` que cita P2 y REQ-21.4. `axiom sdd status` no emite ninguna nota sobre kickoffs inferidos que incumplan la regla.

**Alternativas descartadas.** (a) inferir `checkpointed` y (c) exigir sellado explícito, rechazadas por el usuario (proposal §2.2). Una nota no bloqueante en `status`: amplía el alcance a `internal/sddstatus`, cambia el JSON de estado y sus goldens, y la regla es de sellado, no un invariante de lectura.

### DD-04. La preferencia de revisión independiente es una interacción propia tras el pre-vuelo, no un cuarto grupo

**Elección.** Tras validar los tres grupos canónicos del pre-vuelo y antes del primer lanzamiento de fase de la sesión, el orquestador presenta **una sola interacción** con dos preguntas de selección única:

| Pregunta | Token canónico | Etiquetas (se localizan junto con las descripciones) |
|---|---|---|
| Tras el apply de cada rol | `after_apply` | `Verify` → `verify`; `Verify and judgment` → `verify-judgment`; `None` → `none` |
| Cobertura tras spec y design | `coverage` | `Yes` → `yes`; `No` → `no` |

Cada opción lleva una descripción de efecto y coste; la de `Verify and judgment` dice que en `continuous` incluye las correcciones acotadas del juicio dentro de las raíces de edición autorizadas (REQ-28.14, REQ-28.7). En runtimes con interfaz nativa de preguntas se usa esa interfaz (en Claude Code, `AskUserQuestion` con dos preguntas: 3 y 2 opciones); sin ella, el respaldo en texto del contrato *Lossless Blocking Prompts* con la sintaxis `After apply: <label>; Coverage: <label>`. La respuesta se valida con *Answer validation*; nunca se toma un valor por defecto. El texto de esa interacción nunca reutiliza los marcadores de pregunta del pre-vuelo.

**Alternativa descartada: cuarto grupo con marcador `4/4`.**

- **No cabe.** Son dos dimensiones (3 × 2 combinaciones). `AskUserQuestion` admite como máximo 4 preguntas y 4 opciones por pregunta: dos preguntas nuevas suman 5; fundirlas en una sola pregunta de 4 opciones pierde combinaciones (por ejemplo, juicio sin cobertura).
- **Rompe la corroboración.** `resolveSDDPreflightHookBlock` (`internal/cli/sdd_preflight_hook.go:362-375`) exige exactamente tres preguntas con marcador `N/3:` y etiquetas byte-exactas; una cuarta convierte el pre-vuelo en `protocol error` y el hook deniega todo despacho SDD. Cambiarlo obliga a tocar, además del hook, el cuerpo canónico (`internal/components/sdd/session_preflight.go:16`, que dice expresamente «NEVER ask it as a fourth group»), su variante sin herramienta nativa (`:58-69`), los plugins de OpenCode (`internal/assets/opencode/plugins/sdd-task-result-artifacts.ts` y `plugins-v2/`), los arneses de `bench/` y `internal/sddtaskresult/preflight_question_harness_test.go`, y el bloque proyectado de 6 líneas que valida `validSDDPreflightHookBlock`.
- **Rompe instalaciones existentes.** El hook es el binario `axiom`; las instrucciones son assets que llegan con `axiom sync`. Un binario nuevo con instrucciones viejas (tres grupos) denegaría el despacho hasta sincronizar.

**Justificación.** El pre-vuelo existe para fijar **autoridad** de despacho (ritmo, almacén, estrategia), por eso se corrobora en el transcript. La preferencia no autoriza nada: solo decide si el orquestador lanza diagnósticos informativos. Que un modelo la «invente» solo produce más o menos revisiones informativas, nunca una escritura no autorizada (las correcciones del juicio siguen limitadas a las raíces autorizadas). El hook ignora las llamadas a `AskUserQuestion` sin el prefijo del pre-vuelo (`recognized == false`, `sdd_preflight_hook.go:353-361`), así que la interacción propia no altera la autoridad ya corroborada. Se cumple la «pregunta más en el pre-vuelo» de P1 (c) en el mismo arranque de sesión, sin la consecuencia de reabrir el contrato corroborado.

### DD-05. La preferencia es de sesión y no se persiste

**Elección.** Se cachea en la sesión igual que `delivery_strategy`. Sin campo en `kickoff.yaml` ni en `gates.yaml`, sin observación de Engram. Una sesión nueva la vuelve a preguntar una vez. Una sesión sin preferencia cacheada (orquestador anterior) se trata según REQ-28.2: en `continuous` no se lanza nada ni se pregunta; en `checkpointed` la elección se ofrece dentro de la presentación de la parada.

**Alternativas descartadas.** Campo en `kickoff.yaml` (excluido por P1 (c) y por D1). Guardarla en Engram: convertiría una memoria escrita por un modelo en un valor por defecto con apariencia de autoridad, la misma clase de fuente que el pre-vuelo rechaza explícitamente («prior sessions … do not»).

### DD-06. Proyección: una sección compartida nueva en los doce orquestadores

**Elección.**

- Sección nueva `<!-- sdd-orchestrator-section:SDD Independent Review:start/end -->` en `sdd-orchestrator-sections.md`, con la preferencia (DD-04, DD-05), su aplicación por estilo sellado (REQ-28.2), los tres puntos (REQ-28.1), el verify por rol (DD-07) y su excepción explícita al enrutado por dependencias (DD-07c), el juicio (DD-08), la cobertura (DD-09), la presentación en paradas (REQ-28.10), el resumen final en `continuous` y la indisponibilidad en runtimes sin subagentes (DD-11).
- Cada una de las doce plantillas (`internal/assets/{antigravity,claude,codex,cursor,gemini,generic,hermes,kimi,kiro,opencode,qwen,windsurf}/sdd-orchestrator.md`) añade, inmediatamente después del marcador `{{GENTLE_AI_SDD_SECTION:SDD Change Kickoff and Block Gates}}`, su propio encabezado `SDD Independent Review` (con el nivel que use para el de las puertas) y el marcador `{{GENTLE_AI_SDD_SECTION:SDD Independent Review}}`.
- Ediciones puntuales de textos compartidos ya proyectados: en `SDD Change Kickoff and Block Gates` (línea 80), la opción `continuous` se describe como válida solo con el rol único `fullstack`, y con roles especializados se sella `checkpointed` (el comando rechaza lo contrario); en `Dependency Graph` (líneas 36-43), la frase de verify remite a `SDD Independent Review`; en `internal/assets/claude/sdd-orchestrator-workflow.md` (línea 125, la copia propia de Claude del grafo) la misma remisión.

**Alternativas descartadas.**

- *Ampliar el cuerpo de `SDD Change Kickoff and Block Gates`*: no requiere tocar plantillas, pero mezcla la doctrina de puertas con la de revisiones, deja la nueva doctrina sin encabezado propio que las pruebas puedan aislar (el pin de DD-10 necesita un cuerpo delimitado) y sigue siendo texto siempre cargado en todos los runtimes, igual que con la sección nueva.
- *Prosa por runtime*: es exactamente la deriva que `orchestrator_drift_ratchet_test.go` mide y frena.
- *Solo en el flujo diferido de Claude (`sdd-orchestrator-workflow.md`)*: rompería la paridad; los demás runtimes cargan las puertas en su texto principal.

**Justificación.** Una sola fuente, sustituida por `substituteSharedOrchestratorSections` (`internal/components/sdd/orchestrator.go:79-93`) antes que cualquier otro paso; `composeFallbackSessionPreflight` no toca secciones con ese encabezado (solo reemplaza `Artifact Store Policy`, `Artifact Store Mode`, `Delivery Strategy` y los productores de modo de ejecución), así que la sección pasa igual a los runtimes de respaldo.

**Corrección a la propuesta: copias versionadas.** `AGENTS.md` y `GEMINI.md` de la raíz del repositorio no contienen las secciones proyectadas. `.codex/AGENTS.md` sí (`### SDD Change Kickoff and Block Gates`, línea 630), pero es una copia **heredada**: es el fichero que escribían las instalaciones workspace anteriores y que hoy el adaptador de Codex solo declara para retirarlo (`LegacySystemPromptFiles`, `internal/agents/codex/workspace.go:96-103`); el binario actual ya no lo genera. **Decisión: se deja intacto.** Regenerarlo con `axiom sync` no lo actualiza, sino que lo retira cuando prueba que es de Axiom, y esa retirada es una limpieza ajena a este cambio que además mezclaría en el diff un borrado de 929 líneas sin relación con la revisión independiente. Que no contenga la sección nueva no afecta a ningún runtime: Codex no lo lee. Las copias que sí cambian son los goldens de `testdata/golden/` (`sdd-*-claudemd`, `sdd-codex-agentsmd*`, `sdd-gemini-geminimd`, `sdd-cursor-rules`, `sdd-kiro-instructions`, `sdd-opencode-multi-settings`, `sdd-vscode-instructions`, `sdd-windsurf-global-rules`, `sdd-antigravity-rulesmd`, `combined-*`).

### DD-07. Verify por rol: ámbito resuelto con `axiom branch status`, detalle en una referencia nueva y estado «limpio» explícito

**Elección.**

1. **Lanzamiento.** El orquestador lanza `sdd-verify` con el rol, su `verify_file` (del kickoff: `verify-report.md` para `fullstack`, `verify-report.<rol>.md` para el resto; `internal/kickoff/types.go:218-223`) y la indicación de que el ámbito es el diff de ese rol. Con el rol único `fullstack`, el verify del rol y el del cambio son el mismo informe: no se lanza dos veces. En Engram, el informe del rol `fullstack` sigue en `sdd/{change}/verify-report`, y el de cualquier otro rol en `sdd/{change}/verify-report/{rol}`.
2. **Diff del rol.** `sdd-verify` lo resuelve sin escribir nada: lee `axiom branch status --cwd "<workspaceRoot>" --change <c> --json` y, por cada repositorio, toma la rama del rol (`kind: role` de ese rol) o, con el rol único `fullstack`, la rama del cambio (`kind: change`), con `worktree` y `comparedTo` tomados del JSON (campos verificados en la salida real de `branch status`). El diff se toma **contra el árbol de trabajo desde el merge-base**, porque el apply y `jd-fix-agent` dejan cambios sin commit en ficheros ya versionados y `diff <comparedTo>...HEAD` solo vería lo confirmado:
   - `base = git -C "<worktree>" merge-base <comparedTo> HEAD`;
   - `git -C "<worktree>" diff <base>`: confirmados, preparados y sin preparar de ficheros versionados, frente al árbol de trabajo;
   - más los ficheros sin seguimiento de `git -C "<worktree>" ls-files --others --exclude-standard`, leídos completos.

   Se excluye la carpeta del cambio (`<planningHome>/changes/<c>/`), que es planificación, no trabajo del rol. Si no hay rama resoluble (cambio sin instantánea git), el ámbito cae a los ficheros que el `apply-progress` del cambio declara tocados por ese rol, y el informe lo declara como limitación.
3. **Cumplimiento técnico.** El procedimiento (fuentes a–d de D4/REQ-28.5, forma del hallazgo con fuente, regla y `path:línea`, observaciones sin regla, fuentes ausentes como «no aplicable», comprobaciones no ejecutadas) vive en una referencia nueva, `internal/assets/skills/sdd-verify/references/technical-compliance.md`. `SKILL.md` solo gana la activación ampliada, una regla dura que remite a la referencia y una fila de compuerta de decisión, porque su cuerpo ya ronda el máximo duro de la guía de estilo (763 palabras ≈ 1000 tokens; `docs/skill-style-guide.md`, «Body Budget»).
4. **Informe.** `references/report-format.md` gana las secciones **Cumplimiento técnico** y **Estado** (DD-07b). La sección técnica nunca escribe una línea que empiece por `verdict:`: la barrera de `internal/multirole/barrier.go:12` toma la primera coincidencia de `(?m)^verdict:` y su semántica (funcional) no cambia (REQ-28.4).

**DD-07b. Determinación mecánica de «verify limpio» (supuesto A-1).** El resumen del informe termina con una línea `**Estado:** limpio` o `**Estado:** no limpio — <motivo>`, que escribe el propio verify: es `limpio` si y solo si no hay ningún hallazgo `CRITICAL` (ni funcional ni técnico) y todas las comprobaciones obligatorias se ejecutaron y pasaron. Comprobaciones obligatorias: los comandos de `rules.verify` de `openspec/config.yaml` y, cuando existen, `testing.runner.command` y `testing.quality.type_checker.command`. Una comprobación no disponible o no ejecutada hace el informe `no limpio`. `WARNING` y `SUGGESTION` no lo impiden. Un diff de rol vacío produce `no limpio — diff del rol vacío` (no hay nada que juzgar) y no se ofrece juicio. El orquestador decide el juicio leyendo esa línea del informe vigente, nunca infiriéndola del resumen de la fase.

**DD-07c. Excepción explícita al enrutado por dependencias (REQ-21.15).** El *Native SDD Dispatcher Guard* (`sdd-orchestrator-sections.md:21`) enruta las fases solo por `nextRecommended` y las dependencias, y `verifyDependencyFromRoleGates` (`internal/sddstatus/governance_handoff.go:248-266`) bloquea la dependencia `verify` de un roster multirrol mientras algún `role-apply` no esté aprobado. La sección `SDD Independent Review` declara que el verify **por rol** es una excepción explícita a ese enrutado: el orquestador lo lanza desde la propia sección, al concluir el `apply` de un rol, aunque `verify` figure bloqueado. No contradice REQ-21.15 porque no es la fase `verify` global del cambio:

- es un diagnóstico acotado al diff de un rol, que escribe en `verify-report.<rol>.md` (Engram `sdd/{change}/verify-report/{rol}`), y la proyección de estado no lee ninguno de los dos: solo considera `verify-report.md` (`internal/sddstatus/status.go:1475`, `governance.go:453`) y el título exacto `sdd/{change}/verify-report` (patrón `engramTitlePattern`, `status.go:1062`). Por tanto no cambia `artifacts.verifyReport`, `dependencies` ni `nextRecommended`;
- con el rol único `fullstack` no hay bloqueo que exceptuar: `verifyDependencyFromRoleGates` deja intacta la dependencia con un roster de un rol (`len(roles) <= 1`), y el verify del rol es el verify opcional del cambio de siempre;
- el verify global multirrol sigue esperando a que todos los `role-apply` estén aprobados y se enruta como hoy.

**Alternativas descartadas.** Un verbo Go que calcule el diff del rol (contradice D1 y añade superficie a `internal/branch`). Que el orquestador clasifique severidades del informe por su cuenta (no es determinista entre runtimes). Reutilizar `verdict:` (cambia lo que lee la barrera de `multi-role-fan-out`). Relajar `verifyDependencyFromRoleGates` para admitir el verify por rol (cambiaría REQ-21.15 y el enrutado global para todos los cambios multirrol).

### DD-08. Juicio sobre un objetivo SDD por rol

**Elección** (sección nueva **SDD Role Target** en `internal/assets/skills/judgment-day/references/prompts-and-formats.md`; `SKILL.md` gana una frase de activación y una fila de compuerta):

- **Activación.** Aceptar la oferta del orquestador, o una preferencia `verify-judgment`, cuenta como la petición explícita del contrato de activación. Solo con el informe vigente del rol en `**Estado:** limpio`; una petición explícita del usuario prevalece y se lanza avisando de que el verify no era limpio (A-2). Antes de lanzar, una línea de coste (dos jueces ciegos y hasta dos rondas de arreglo); en `continuous` no detiene.
- **Objetivo inmutable.** Identidad = `{rol, repositorios, merge-base, SHA de HEAD, sha256}`, donde el sha256 se calcula sobre la salida de `git -C "<worktree>" diff <merge-base>` más la lista y el contenido de los ficheros sin seguimiento, con el mismo ámbito y exclusiones de DD-07 paso 2; así cubre los cambios sin commit que dejan el apply y los arreglos. Los jueces (`jd-judge-a`, `jd-judge-b`) leen el árbol de trabajo en solo lectura mientras ningún escritor actúa; antes de fusionar, el padre recalcula el sha256: si cambió, la ronda se descarta y se informa, sin reintento automático.
- **Arreglos.** `jd-fix-agent` edita solo dentro de `actionContext.allowedEditRoots` ∩ el `worktree` del rol, y nunca bajo `technicalContext.reference`. Un hallazgo confirmado cuyo arreglo exige salir de esas raíces queda pendiente de decisión humana e impide `APPROVED` (REQ-28.7). Confirmación de la primera ronda (A-3): en `checkpointed` se pregunta antes de editar, como dice hoy el contrato del juicio; en `continuous` la preferencia `verify-judgment` ya la incluye.
- **Registro.** El padre fusiona y persiste el ledger en `review-ledger.md` (rol `fullstack`) o `review-ledger.<rol>.md`; en Engram, `sdd/{change}/review-ledger` o `sdd/{change}/review-ledger/{rol}` (A-4). El `apply-progress` es un artefacto único por cambio (`apply-progress.md` y `sdd/{change}/apply-progress`); los arreglos aplicados se anotan como unidades de trabajo en una subsección por rol, `### Correcciones del juicio — <rol>`, dentro de ese artefacto, que el padre fusiona sin sobrescribir lo anterior (como exige la continuidad de `apply-progress`), y **nunca** en el fichero de tareas, para no alterar el digest de `role-apply:<rol>` (`internal/cli/sdd_gate_digest.go:83-97`).
- **Verificación final.** Si hubo arreglos, el padre vuelve a lanzar `sdd-verify` del rol, que reescribe el `verify_file` como informe vigente y conserva el anterior como contexto histórico (regla 5 de `sdd-verify` y `report-format.md`, «Historical context»). Sin arreglos, el informe vigente no cambia y el veredicto del juicio se informa aparte.
- Se conservan intactas las reglas del juicio: dos jueces en paralelo, acuerdo como corroboración, solo severos confirmados, dos rondas como máximo, `APPROVED | ESCALATED`, sin recibo ni autoridad de entrega (el pin `judgment_day_standalone_test.go` sigue exigiendo «issues no receipt»).

**Alternativas descartadas.** Prohibir arreglos en el objetivo SDD y devolver todo a `sdd-apply` (contradice D5). Un actor de arreglos nuevo (D5 pide reutilizar los existentes). Anotar los arreglos en `tasks*.md` (reabriría la parada `role-apply`).

### DD-09. Cobertura: dos revisores reutilizando las identidades de juez, unión hecha por el padre

**Elección.**

- **Revisores.** Dos, en paralelo y ciegos, lanzados con las identidades `jd-judge-a` y `jd-judge-b` y el mandato del asset nuevo `internal/assets/skills/_shared/sdd-coverage-review.md`. Esas identidades ya son de contexto limpio y de solo lectura por construcción (en Claude, `tools: Read, Glob, Grep` y búsqueda en Engram; `internal/assets/claude/agents/jd-judge-a.md`), existen en todos los runtimes que ejecutan el juicio y su cuerpo ordena seguir el formato del prompt delegado. El mandato declara que la tarea revisa artefactos de planificación, que no se aplican el presupuesto de barridos, la puerta de precisión ni la persistencia de ledger de su definición, y fija su salida: `{"gaps":[{"kind":"use_case|edge_case|error|uncovered_requirement","artifact":"spec|design","location":"<ruta>#<encabezado>","description":"…","basis":"<cita de la propuesta o del spec>"}],"evidence":["…"]}`.
- **Entradas.** Tras `spec`: propuesta y specs. Tras `design`: propuesta, specs y design (incluye requisitos del spec que el design no cubre). Si spec y design corrieron en paralelo, la revisión de design espera a que existan ambos.
- **Unión.** El padre une y deduplica (misma ubicación y mismo comportamiento ausente ⇒ una fila con ambos revisores). Un revisor sin resultado utilizable se declara como tal y no se inventa su contenido.
- **Informe** (A-4): `coverage-review.spec.md` y `coverage-review.design.md` en la raíz de la carpeta del cambio; en Engram, `sdd/{change}/coverage-review/spec` y `sdd/{change}/coverage-review/design`; en `hybrid`, ambos; en `none`, en línea. Cada ejecución añade una sección fechada (`## Ronda N — AAAA-MM-DD`) en lugar de sobrescribir; un informe sin huecos lo declara. Ninguno de esos nombres entra en los digests: la puerta `spec` digiere `specs/**/spec.md` o `spec.md` y la `design` solo `design.md` (`internal/cli/sdd_gate_digest.go:39-55`, `:116-140`). `axiom sdd publish` los publica con el resto de la carpeta del cambio (`internal/docpaths`).
- **Sin autocorrección.** Las correcciones las hace `sdd-spec` o `sdd-design` por «iterar» en `checkpointed` o por una nueva ejecución que pida la persona en `continuous`; el padre nunca relanza la fase por su cuenta. Los hallazgos no activan el reintento del gatekeeper automático.

**Alternativas descartadas.** Identidades de agente nuevas (`sdd-coverage-*`): exigen alta en cada runtime (agentes de Claude y Kiro, `internal/opencode/models.go`, `internal/model/*_model.go`, desinstalación) y filas nuevas en la tabla de asignación de modelos de todos los orquestadores. Un agente generalista del runtime: no es de solo lectura por construcción en todos los runtimes. Un único revisor: D6 prioriza no perder huecos, y dos miradas ciegas con unión maximizan la exhaustividad. Guardar el informe dentro de `design.md` o bajo `specs/`: cambiaría el digest de la puerta.

### DD-10. Pin de vocabulario acotado al texto nuevo

**Elección.** Prueba nueva `internal/assets/independent_review_boundary_test.go`, acotada a los **pasajes nuevos o modificados** por este cambio, nunca a ficheros enteros:

- **Cuerpos delimitados nuevos**, extraídos por sus límites: la sección `SDD Independent Review` (marcadores `sdd-orchestrator-section`), los ficheros nuevos `_shared/sdd-coverage-review.md` y `sdd-verify/references/technical-compliance.md`, los elementos «Cumplimiento técnico» y «Estado» de `report-format.md`, y la sección `## SDD Role Target` de `prompts-and-formats.md` hasta el siguiente encabezado `## `.
- **Frases nuevas o reescritas dentro de ficheros existentes** (la descripción y la activación de `sdd-verify/SKILL.md`, su regla y su fila de decisión nuevas, la frase de activación y la fila nuevas de `judgment-day/SKILL.md`, la frase de `continuous` de las puertas, la remisión del `Dependency Graph` y de `claude/sdd-orchestrator-workflow.md`, y las reglas añadidas a `sdd-spec` y `sdd-design`): la prueba las declara como constantes, comprueba que están presentes en su fichero y aplica el pin solo a ellas. El resto de esos ficheros queda fuera, porque ya nombra RDD a propósito para declarar la frontera (por ejemplo, `SDD never offers, launches, or consumes RDD.` en `sdd-verify/SKILL.md` o la frase sin recibo de `judgment-day/SKILL.md:24`), y esas frases las fijan otras pruebas.

El pin comprueba la ausencia de `(?i)\b(lineage|candidates?|acknowledg\w*|burn(s|ed)?|receipts?|rdd)\b` y de los literales `axiom review`, `sdd-attempt` y `review-integration`; además, que la sección compartida no contenga `Axiom SDD preflight` ni `preflight 4/`, y que sí declare los resultados ausentes de REQ-28.10 (una frase que diga que una revisión no ejecutada, sin respuesta o no limpia se presenta como tal y nunca como limpia).

**Alternativa descartada.** Un barrido de ficheros completos o de todos los assets: los textos existentes nombran RDD a propósito para declarar la frontera, así que daría falsos positivos o forzaría excepciones.

**Justificación.** La frontera la declara ya la sección de puertas (REQ-21 y INC-18); el texto nuevo no necesita nombrar RDD y el pin garantiza que no lo hace. `internal/sddstatus/review_offer_absence_guard_test.go` no se toca.

### DD-11. Runtimes sin subagentes

**Elección.** Donde el runtime no puede lanzar un agente de contexto limpio (Windsurf, «Delegated Verification Gate (Reduced Form)»), la cobertura y el juicio se declaran no disponibles una vez al recoger la preferencia y cada vez que tocaría lanzarlos; el verify por rol se ejecuta en línea como hoy. Nunca se simulan en el propio contexto.

**Justificación.** Coherente con «If an applicable verifier is unavailable, preserve the typed unavailable result» del *Native Checking Contract* compartido.

## 3. Flujo de datos

### 3.1 Arranque de sesión

```
usuario ──/sdd-new──▶ orquestador
                        │ 1. pre-vuelo agrupado (3 preguntas, marcadores N/3) ──▶ hook PreToolUse(Agent)
                        │                                                       corrobora en el transcript
                        │ 2. preferencia (interacción propia, 2 preguntas) ──▶ hook: no reconoce prefijo, la ignora
                        │    caché de sesión: after_apply ∈ {verify, verify-judgment, none}, coverage ∈ {yes, no}
                        │ 3. init guard ─▶ kickoff seal (DD-01) ─▶ propose ...
```

### 3.2 Tras `spec` y tras `design`

```
fase spec|design ──▶ gatekeeper (contrato) ──▶ ¿coverage = yes?
                                                 │ no ──▶ continuous: fase siguiente │ checkpointed: parada
                                                 │ sí
                                                 ▼
                         jd-judge-a ┐ (paralelo, solo lectura, mandato sdd-coverage-review.md)
                         jd-judge-b ┘
                                                 ▼
                         padre: unión deduplicada ──▶ coverage-review.{spec|design}.md (+ Engram)
                                                 ▼
                 continuous: fase siguiente (huecos al resumen final)
                 checkpointed: parada spec|design con huecos abiertos y tres salidas (iterar ⇒ sdd-spec|sdd-design)
```

### 3.3 Tras el `apply` de un rol

```
sdd-apply(rol) ──▶ ¿after_apply ≠ none?
                     │ none ──▶ continuous: sigue │ checkpointed: parada role-apply:<rol>
                     ▼
              sdd-verify(rol) [excepción al enrutado, DD-07c]:
                     │   branch status ─▶ git -C <worktree> diff $(merge-base <comparedTo> HEAD) + sin seguimiento
                     │                 ─▶ verify_file: Findings + Cumplimiento técnico + Estado
                     ▼
         ¿after_apply = verify-judgment y Estado = limpio?
                     │ no ──▶ (informa motivo) ──▶ continuous: sigue │ checkpointed: parada
                     ▼ sí (línea de coste)
              jd-judge-a ┐ objetivo inmutable del rol
              jd-judge-b ┘
                     ▼
         ¿severos confirmados por ambos?
                     │ no ──▶ APPROVED
                     ▼ sí
         checkpointed: pregunta antes de la ronda 1 │ continuous: autorizado por la preferencia
                     ▼
              jd-fix-agent (allowedEditRoots ∩ worktree del rol) ─▶ apply-progress «Correcciones del juicio — <rol>»
                     ▼
              re-juicio acotado (≤ 2 rondas) ─▶ APPROVED | ESCALATED ─▶ review-ledger[.<rol>].md
                     ▼ (si hubo arreglos)
              sdd-verify(rol) de nuevo ─▶ informe vigente + histórico
                     ▼
         continuous: sigue (resultados al resumen final) │ checkpointed: parada con verify, juicio y arreglos
```

### 3.4 Sellado

```
axiom sdd kickoff seal ─▶ ParseSealArgs ─(sin --infer)─▶ estilo efectivo + roster ─▶ checkContinuousRoster
                                                                    │ incumple ─▶ error, nada escrito
        runSDDKickoffSeal ─▶ ToKickoff | InferKickoff ─▶ ValidateForSeal (exento si sealed_by=inferred)
                                                                    ─▶ Seal ─▶ Validate ─▶ kickoff.yaml
ParseKickoff / Load ─▶ Validate (sin la regla) ─▶ lectura tolerante de kickoffs existentes y archivados
```

## 4. Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/kickoff/schema.go` | Modificar | `ErrContinuousRequiresFullstack`, `checkContinuousRoster`, `(Kickoff) ValidateForSeal()`. `Validate()` no cambia. |
| `internal/kickoff/types.go` | Modificar | Constante `SealedByInferred`. |
| `internal/kickoff/args.go` | Modificar | `ParseSealArgs` aplica `checkContinuousRoster` al estilo efectivo y al roster final; mensaje específico si el estilo se derivó de `--from-session-pace auto`. |
| `internal/kickoff/infer.go` | Modificar | Usa `SealedByInferred`; comentario que fija P2. Sin cambio de comportamiento. |
| `internal/cli/sdd_kickoff.go` | Modificar | `ValidateForSeal` antes de `kickoff.Seal`; línea de ayuda «continuous solo con el rol fullstack». |
| `internal/kickoff/schema_test.go` | Modificar | Tabla de `ValidateForSeal`; `ParseKickoff` lee `continuous` + `[core, web]`. |
| `internal/kickoff/args_test.go` | Modificar | Casos nuevos de sellado; `TestParseSealArgsRoleFlagPolicyParsing` pasa a `--execution-style checkpointed`. |
| `internal/kickoff/infer_test.go` | Modificar | Pin de P2: inferido `continuous` + roles detectados, `sealed_by: inferred`, `ValidateForSeal` sin error. |
| `internal/cli/sdd_kickoff_test.go` | Modificar | `seal continuous --role core --role web` falla sin crear `kickoff.yaml`; `seal --infer` con roles sigue sellando. |
| `internal/cli/sdd_kickoff_branch_test.go` | Modificar | Los sellados con roles (`:152`, `:238`, `:348`) usan `checkpointed`. |
| `internal/cli/sdd_gate_handoff_test.go` | Modificar | `continuousArgs` (`:23`) sin roles especializados. |
| `internal/cli/sdd_preflight_hook_test.go` | Modificar | Caso nuevo: pre-vuelo canónico seguido de la interacción de preferencia conserva la autoridad. El hook no cambia. |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Sección `SDD Independent Review`; frase de `continuous` en las puertas; remisión en `Dependency Graph`. |
| `internal/assets/{antigravity,claude,codex,cursor,gemini,generic,hermes,kimi,kiro,opencode,qwen,windsurf}/sdd-orchestrator.md` | Modificar | Encabezado y marcador de la sección nueva tras el de las puertas. |
| `internal/assets/claude/sdd-orchestrator-workflow.md` | Modificar | Remisión a `SDD Independent Review` junto a su copia del grafo. |
| `internal/assets/skills/sdd-verify/SKILL.md` | Modificar | Descripción y activación (petición, oferta aceptada o preferencia), regla de ámbito por rol y fila de decisión que remiten a la referencia; conserva todas las frases fijadas por pruebas. |
| `internal/assets/skills/sdd-verify/references/technical-compliance.md` | Crear | Resolución del diff del rol (merge-base contra el árbol de trabajo más los ficheros sin seguimiento), clave de Engram por rol, fuentes a–d, forma del hallazgo, observaciones, fuentes ausentes, comprobaciones no ejecutadas, regla de `Estado`. |
| `internal/assets/skills/sdd-verify/references/report-format.md` | Modificar | Secciones «Cumplimiento técnico» y «Estado»; prohibición de líneas `verdict:` en la sección técnica. |
| `internal/assets/skills/judgment-day/SKILL.md` | Modificar | Activación por oferta o preferencia SDD; fila de decisión para el objetivo SDD por rol; remisión a la referencia. |
| `internal/assets/skills/judgment-day/references/prompts-and-formats.md` | Modificar | Sección `SDD Role Target` (DD-08). |
| `internal/assets/skills/_shared/sdd-coverage-review.md` | Crear | Mandato de los revisores, salida, regla de unión, formato del informe y rondas. Se instala solo por `SharedSkillFileNames()`. |
| `internal/assets/skills/_shared/openspec-convention.md` | Modificar | `coverage-review.spec.md`, `coverage-review.design.md`, `verify-report.<rol>.md`, `review-ledger[.<rol>].md`. |
| `internal/assets/skills/sdd-spec/SKILL.md`, `internal/assets/skills/sdd-design/SKILL.md` | Modificar | Una regla: la cobertura propia no sustituye a la revisión de cobertura. |
| `internal/components/sdd/orchestrator_shared_sections_test.go` | Modificar | Registrar `SDD Independent Review` en `sharedOrchestratorSectionNames`. |
| `internal/assets/orchestrator_drift_ratchet_test.go` | Modificar | Entrada `{"SDD Independent Review", 1}`. |
| `internal/assets/independent_review_boundary_test.go` | Crear | Pin de DD-10 y paridad: toda plantilla que proyecta las puertas proyecta una vez la sección nueva. |
| `testdata/golden/*.golden` | Regenerar | Con `go test ./internal/components/ -run TestGolden -update`, nunca a mano. |
| `.codex/AGENTS.md` | Sin cambios (deliberado) | Copia heredada que el binario actual ya no genera (`internal/agents/codex/workspace.go:96-103`); ni se edita a mano ni se regenera ni se retira en este cambio (DD-06). |

Sin cambios: `internal/kickoff/seal.go`, `internal/kickoff/machine.go`, `internal/cli/sdd_preflight_hook.go`, `internal/components/sdd/session_preflight.go`, los plugins de OpenCode, `internal/sddstatus/**` (incluida `review_offer_absence_guard_test.go`) e `internal/multirole/barrier.go`.

## 5. Interfaces y contratos

```go
// internal/kickoff/types.go
const SealedByInferred = "inferred"

// internal/kickoff/schema.go

// ErrContinuousRequiresFullstack identifica el rechazo de sellado de D7/D8. Se
// envuelve con el roster recibido y las dos salidas; errors.Is lo reconoce.
var ErrContinuousRequiresFullstack = errors.New("execution_style continuous solo admite el rol único fullstack")

// checkContinuousRoster es la regla pura: nil si style != continuous o si roles
// es exactamente ["fullstack"]. fromSessionPace solo cambia el texto del mensaje.
func checkContinuousRoster(style ExecutionStyle, roles []string, fromSessionPace bool) error

// ValidateForSeal = Validate() + checkContinuousRoster, salvo SealedBy == SealedByInferred.
// Solo la usa el camino de sellado; ParseKickoff y Load siguen con Validate().
func (k Kickoff) ValidateForSeal() error
```

Mensaje (una línea, sin escribir nada):

```
kickoff seal: execution_style continuous solo admite el rol único fullstack (roles recibidos: core, web); sella con --execution-style checkpointed para conservar esos roles, o quita las banderas --role para sellar con fullstack
```

Con estilo derivado, el prefijo dice `--from-session-pace auto deriva continuous, que solo admite el rol único fullstack`; nunca deriva `checkpointed` (D8, REQ-28.13).

Contratos de texto (prosa, sin Go):

| Contrato | Forma |
|---|---|
| Preferencia | `after_apply ∈ {verify, verify-judgment, none}`, `coverage ∈ {yes, no}`; caché de sesión. |
| Estado del verify | Última línea del resumen: `**Estado:** limpio` \| `**Estado:** no limpio — <motivo>`. |
| Hallazgo técnico | `severidad · fuente (skill <nombre> \| technicalContext \| openspec/config.yaml) · regla · path:línea[-fin] · evidencia`. |
| Revisor de cobertura | JSON `{"gaps":[…],"evidence":[…]}` (DD-09). |
| Informes (fichero) | `verify-report[.<rol>].md`, `review-ledger[.<rol>].md`, `coverage-review.{spec,design}.md`; sufijo `.<rol>` solo para roles distintos de `fullstack`. |
| Informes (Engram) | `sdd/{change}/verify-report[/{rol}]`, `sdd/{change}/review-ledger[/{rol}]`, `sdd/{change}/coverage-review/{spec,design}`; ninguna clave con sufijo casa con `engramTitlePattern`, así que la proyección de estado no la lee. |
| Correcciones del juicio | Subsección `### Correcciones del juicio — <rol>` dentro del `apply-progress` único del cambio. |

## 6. Estrategia de pruebas

TDD desactivado (`openspec/config.yaml`, `strict_tdd: false`): pruebas junto al cambio, por paquete y sin binarios sueltos.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad `internal/kickoff` | `ValidateForSeal`: `continuous`+`[fullstack]` sí; `+[core,web]`, `+[backend]`, `+[fullstack,backend]`, `+[fullstack,fullstack]` no (`errors.Is`, mensaje con `--execution-style checkpointed` y `--role`); `checkpointed`+`[core,web]` sí; inferido `continuous`+`[core]` sí. | Tabla. |
| Unidad `internal/kickoff` | `ParseSealArgs`: `continuous` sin roles ⇒ `fullstack`; `--from-session-pace auto` + `--role core` rechaza sin derivar `checkpointed`; `interactive` + roles ⇒ `checkpointed`; `checkpointed` explícito sobre `auto` + roles sella; `continuous` explícito sobre `auto` + roles rechaza. | Tabla. |
| Unidad `internal/kickoff` | Lectura tolerante: `ParseKickoff` y `Load` de un `continuous` + `[core,web]`. P2: `InferKickoff` con roles en `design.md`. `rdd_boundary_test.go` sigue verde: ni el código ni las pruebas nuevas del paquete usan su vocabulario prohibido. | YAML en línea y `t.TempDir()`. |
| Integración `internal/cli` | `RunSDDKickoff seal` rechaza sin crear `kickoff.yaml`; un `seal` repetido con `continuous` y roles sobre un kickoff ya sellado así se rechaza y deja el fichero intacto (cambio de comportamiento de DD-01); `seal --infer` con roles sigue sellando `continuous`; ayuda actualizada; pruebas adaptadas de `sdd_kickoff_branch_test.go` y `sdd_gate_handoff_test.go`. | Proceso, sin red. |
| Integración `internal/cli` | Hook: transcript con el pre-vuelo canónico y después la interacción de preferencia ⇒ `allow` con el mismo bloque de 6 líneas. | Transcript sintético. |
| Assets (`internal/assets`) | Pin de DD-10 sobre pasajes nuevos o modificados, con la frase de resultados ausentes; paridad de la sección en las doce plantillas; ratchet = 1; `judgment_day_standalone_test.go` y `assets_test.go` (frases de `sdd-verify`) siguen verdes; longitud de la descripción de `sdd-verify` (`skills_frontmatter_test.go`, ≤160). | `go test ./internal/assets/`. |
| Componentes (`internal/components/sdd`) | Registro en `sharedOrchestratorSectionNames`; `review_foundations_test.go` (frases de `sdd-verify`) sigue verde. | `go test ./internal/components/sdd/`. |
| Goldens | Proyección en todos los runtimes. | `go test ./internal/components/ -run TestGolden -update`, revisar el diff y repetir sin `-update`. |
| Sin cambios | `go test ./internal/sddstatus/ -run ReviewOfferAbsence`, `go test ./internal/components/sdd/ -run Preflight`, `go test ./internal/multirole/`. | Deben pasar sin tocar el código. |
| Cierre | `go test ./...` y `go vet ./...` (`rules.verify`). | — |

## 7. Matriz de amenazas

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta del diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con aspecto de documentación | `requirements.txt`, `CMakeLists.txt`, Markdown ejecutable, `README.sh` | N/A: nada se ejecuta ni se clasifica por contenido; la exclusión de la carpeta del cambio en el ámbito del verify es por raíz de ruta y solo reduce lo que se lee. | — | — |
| Selección de repositorio git | `git -C`, rutas relativas, absolutas | Aplicable (prosa): verify y juicio leen diffs y el actor de arreglos edita. | Repositorio y base solo de `axiom branch status --json` (`worktree` absoluto y `comparedTo`), siempre con `git -C "<worktree>"`; diff contra el árbol de trabajo desde `merge-base <comparedTo> HEAD` más los ficheros sin seguimiento, para no perder cambios sin commit; arreglos en `allowedEditRoots` ∩ `worktree` del rol; sin ruta resoluble, limitación declarada y `no limpio`. | Pin de texto en `independent_review_boundary_test.go`: la referencia de verify y la sección `SDD Role Target` nombran `git -C`, `worktree`, `comparedTo`, `merge-base`, `ls-files --others` y `allowedEditRoots`, y no usan `...HEAD` como ámbito. |
| Estado de commit | index, `commit -a`, índice vacío | N/A: ninguna pieza nueva hace commits; los arreglos del juicio quedan como el trabajo de `sdd-apply`, cuyo commit sigue ofreciéndolo esa fase. | — | — |
| Estado de push | rama de seguimiento, primer push, refspec | N/A: no se añade ningún push; los informes nuevos se publican con `axiom sdd publish` como el resto de la carpeta. | — | — |
| Comandos de PR | `--head`, prefijos de entorno, comandos compuestos | N/A: sin automatización de PR. | — | — |

## 8. Migración y despliegue

No hay migración de datos. Ningún `kickoff.yaml` ni `gates.yaml` existente cambia ni deja de leerse; un kickoff sellado con el binario nuevo cumple también las reglas del anterior. Los informes nuevos son Markdown que el binario anterior ignora. Despliegue con `axiom sync`; reversión con `git revert` y `axiom sync`.

## 9. Estimación de entrega

| Bloque | Líneas autoradas (estimación) |
|---|---|
| Go de `internal/kickoff` y CLI | 70–90 |
| Pruebas Go (nuevas y adaptadas, incluido el hook) | 200–260 |
| Prosa: sección compartida, plantillas (12 × 3), skills, referencias, asset de cobertura, convención | 260–340 |
| Pins y registros de secciones | 80–110 |
| Specs delta (ya escritas por la fase `spec`) | ≈ 750 |
| **Total sin goldens** | **≈ 1.360–1.550** |

`single-pr` con `size:exception`, como fija el pre-vuelo. Los goldens regenerados no cuentan en el presupuesto.

## 10. Preguntas abiertas

- [ ] Ninguna bloqueante. La spec debe recoger que las etiquetas de la preferencia se localizan (a diferencia de las del pre-vuelo, que el hook exige byte-exactas) y que la sintaxis de respuesta del respaldo en texto es la de la interacción propia (`After apply: <label>; Coverage: <label>`), lo que satisface el escenario «Un runtime sin interfaz nativa presenta el contenido completo» de REQ-28.14.
- [ ] La spec debe alinear REQ-28.12 con DD-02 (la política de compuerta del rol `fullstack` no se comprueba) y con el rechazo de rosters repetidos.
