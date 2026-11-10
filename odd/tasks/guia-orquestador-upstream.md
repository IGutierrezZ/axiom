# ODD: Guía del orquestador — absorción upstream (delegación y mecánica)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/guia-orquestador-upstream.md`.
> Espejo de recuperación en Engram: topic `odd/guia-orquestador-upstream/tasks`, proyecto `axiom`.
> Ramas: `feat/guia-orquestador-upstream` (T4 y T5, fusionada en el PR #226) e `inc/guia-orquestador-upstream` (T1, worktree `C:\repos\axiom-wt\guia-orquestador-upstream\axiom`).

Plan stop: approved 2026-11-04; implementation continues

## Estado

**En curso (2026-11-10).**
- T4 y T5 están fusionadas (PR #226).
- T1 está implementada en `inc/guia-orquestador-upstream`, pendiente de PR.
- T2 y T3 van después de que se fusione T1.
- Las decisiones P1–P3 están resueltas.

## Objetivo

Absorber de upstream gentle-ai (revisado hasta `origin/main` 3ea08cdf) la parte de la guía del orquestador que afecta a la delegación y a la mecánica del orquestador. La absorción es selectiva: se conservan las divergencias de Axiom.

Divergencias de Axiom que se conservan:
- SDD y OpenSpec vivos.
- RDD opcional y apagado por defecto.
- Autorización de operaciones remotas.
- Elección de TDD estricto.
- Marca propia y módulo `/v3`.

## Problema

- **Regla de delegación.** Se basa en el número de ficheros: 1–3 inline, 4 o más se delega. Es un mal indicador del coste real. La regla está en tres sitios de Go y en las 12 plantillas:
  - `internal/agents/capabilitymanifest/manifest.go:204-213`
  - `internal/components/agentguidance/routing.go:70-95`
  - `internal/components/sdd/inject.go:1661-1700`, bloque de migración.
  - En las 12 plantillas, como tabla y como reglas 1–6.
- **Disparadores incondicionales.** Los «Mandatory Delegation Triggers» empujan a sobredelegar.
- **Plantillas duplicadas.** Las 12 `internal/assets/<runtime>/sdd-orchestrator.md` suman 4747 líneas, en gran parte duplicadas. Gemini, qwen, hermes, generic y cursor son casi copias. Ya hay deriva:
  - «¿Continuamos?» frente a «¿Seguimos?».
  - «Write rule» en las plantillas frente a «Writer trigger» y «Long-session backstop», que solo existen en `routing.go`.
- **Fuente compartida parcial.** Solo existe para 6 secciones SDD: `internal/assets/skills/_shared/sdd-orchestrator-sections.md` junto con `internal/components/sdd/orchestrator.go:21-80`.
- **Protocolo de Engram.** `internal/assets/engram/protocol.md:7-16` detecta el proyecto, pero no obliga a resolverlo antes del primer acceso ni prohíbe hacerlo en paralelo. Además, `protocol.md:54` describe `content` como una lista y no como un único string.
- **Skill `issue-creation`.** `internal/assets/skills/issue-creation/SKILL.md:20` impone Issue Forms YAML a cualquier repositorio destino.

## Por qué

- **Coste y latencia.** Upstream midió en banco que los disparadores incondicionales y la regla por recuento de ficheros encarecen y ralentizan sin mejorar la calidad.
- **Deriva.** Las 12 plantillas divergen con cada cambio.

## Evidencia upstream verificada

- **Los disparadores no se retiraron.** Upstream los hizo condicionales: «Each trigger holds only when its condition truly holds… when in doubt work inline» (`gentle-ai/internal/components/agentguidance/routing.go:121`). Sigue vigente en `origin/main`.
- **Presupuesto de evidencia (#5147, sha 30ff0afd).** La regla nueva es: «Inline evidence uses one parallel batch, at most 3 calls and approximately 10k tokens». Si no cabe, un explorador devuelve un traspaso de ≤2k tokens con `path:line` y el padre hace una sola comprobación puntual.
  - Evidencia en el issue #5139: 146 delegaciones contrafactuales y 256 respuestas correctas.
  - Delegar toda la exploración cuesta un 46–65 % más de tokens y unas 3× de latencia.
  - Con la regla de presupuesto, el coste queda dentro de ±2 % y el contexto del padre baja unos 27k tokens.
  - Solo hubo 2 repeticiones; el propio issue lo llama «weak signal».
- **Delegar por razón (#5250, #5522, #5530).** Las mediciones están en los issues #5521 y #5529, en un banco privado del mantenedor, con un solo modelo en esfuerzo bajo:
  - En la tarea x6, el coste pasó de 1/3 aceptadas, unos 740 s y 1,83 USD a 3/3 aceptadas, 392 s y 1,24 USD.
  - En #5522 el tiempo y el coste bajaron entre un 27 y un 35 %.
  - Limitaciones: n pequeño, brazos que agrupan todos los cambios y metodología no auditable.
  - Lectura: la dirección es consistente, pero la prueba es débil.
- **Contrato compartido (#5005 y #5006).** En upstream la fuente única es `internal/assets/skills/_shared/odd-orchestrator-sections.md`, con marcadores `odd-orchestrator-section:<nombre>`. Las plantillas usan `{{GENTLE_AI_ODD_SECTION:<nombre>}}`, el render está en `agentguidance/orchestrator.go` y la paridad se prueba en `orchestrator_parity_test.go`.
  - El render depende de la retirada de SDD (#4967) y de RDD por defecto (#4803), con gating por runtime y `stripReceiptDrivenDevelopment`.
  - Por eso no se porta el código, solo el patrón.

## Veredictos

| PR upstream | Veredicto | Alcance en Axiom |
|---|---|---|
| #5005/#5006 contrato compartido | ADAPTAR (patrón) | Se amplía el mecanismo existente `{{GENTLE_AI_SDD_SECTION:…}}` (T1). |
| #5147 presupuesto de evidencia | ADAPTAR | T2. |
| #5250/#5522 writer por razón y disparadores condicionales | ADAPTAR (parte) | T2. Se adopta la condicionalidad y que el número de ficheros nunca dispare la delegación. |
| #4900 proporcionalidad | ADAPTAR (parte) | T2. Se adopta validar premisas, reutilizar la investigación hermana, hacer comprobaciones focales al iterar y completas al cerrar. La cláusula del «segundo fichero mecánico» queda absorbida por el writer por razón. Sin las líneas de RDD. |
| #5530 rebanadas | ADAPTAR (parte) | T2. Las unidades de ≤100 líneas van siempre inline. Axiom ya tiene la heurística de unas 400 líneas. No se adoptan las ramas «RDD runtimes / others». |
| #5850 + #5486 aprobación informada y continuidad | ADOPTAR | T3. |
| #2022 resolver el proyecto antes del primer acceso | ADOPTAR | T4. |
| #5663 `mem_save` con `content` como string JSON | ADOPTAR | T4. |
| #5907 `issue-creation` sigue la política del destino | ADOPTAR (adaptado) | T5. Sin reglas propias para `IGutierrezZ/axiom`, que no recibe issues (Engram #520). Axiom no tiene la skill `branch-pr`. |
| #4978 TDD siempre instalado | DESCARTAR | Elimina la elección de TDD estricto (pantalla TUI y flag retirado), que Axiom conserva (`internal/model/selection.go:11`, `internal/state/state.go:69`). |
| #5522/#5250 «Risk line» y verificador solo por ítem de riesgo nombrado | FUERA (dependencia) | Choca con la decisión de verificar por nivel de riesgo (Engram #1922). Lo decide la sesión de revisión nativa. |
| #5218 Task Size | FUERA (dependencia) | Cambia la clasificación de «sustancial» del paso 4 de ODD y roza la selección de carril. Se coordina con el carril SDD versátil. |
| #5215 spec por referencia (`## Specs`, `## Tasks`, `## Log`, `S#`) | FUERA (dependencia) | Cambia el formato del documento ODD y se solapa con la spec viva. Se coordina con el carril SDD versátil. |

## Alcance propuesto (tareas)

- [x] **T1 · Contrato compartido de la guía del orquestador** (talla L). Commit `999e6172`, ruta delegada (writer; más de 4 ficheros y más de 2 no triviales). Véase el registro del 2026-11-10.
  - Ampliar la fuente única: llevar a secciones compartidas lo que hoy se duplica en las 12 plantillas:
    - Delegation Rules (tabla) y Mandatory Delegation Triggers.
    - Lossless Blocking Prompts.
    - Native Checking Contract.
    - Las demás secciones no divergentes que se midan.
  - Las secciones SDD (kickoff, compuertas, Dispatcher Guard) se mueven como bloques opacos, sin cambiar su texto.
  - Las secciones propias de cada runtime se quedan en su plantilla: Kiro steering, Windsurf Plan/Code, Codex capability check, Hermes native delegation y OpenCode entry routing.
  - Test de no deriva: cada sección compartida aparece una sola vez por plantilla y no quedan marcadores sin resolver.
  - Se conservan la autorización remota (`internal/assets/generic/remote-authorization-contract.md`, que se inyecta aparte) y RDD opcional.
  - Decisión de diseño abierta (P2): ¿la fuente única de la tabla y los disparadores es `RenderRouting` en Go o el markdown compartido?
- [ ] **T2 · Delegación por presupuesto de evidencia y por razón** (depende de T1)
  - Sustituir 1–3/4+ por el presupuesto de evidencia:
    - Inline, un lote paralelo de ≤3 llamadas y unos 10k tokens.
    - Si no cabe, explorador con traspaso de ≤2k tokens con `path:line` y una comprobación puntual del padre.
  - Disparadores condicionales: «solo obligatorio si su condición se cumple; ante la duda, inline».
  - Writer solo por razón nombrada (paralelismo o contexto); el número de ficheros nunca lo dispara.
  - Unidades de ≤100 líneas, siempre inline.
  - Proporcionalidad del gasto (#4900, parte).
  - Revisar el «Long-session backstop», que pasa a ser orientativo.
  - Toca:
    - `manifest.go` (campos y digests).
    - `routing.go`.
    - El bloque de migración de `sdd/inject.go:1661-1700`.
    - La sección compartida de T1.
    - `internal/assets/assets_test.go`.
    - Goldens (`go test ./internal/components/ -update`).
    - `internal/assets/skills/hermes-ephemeral-delegation/SKILL.md:14,34`.
- [ ] **T3 · Aprobación informada y continuidad** (#5850, #5486)
  - Texto en la sección compartida y en `routing.go`.
  - Interpretar «sí», «adelante» o «eso mismo» contra la propuesta inmediatamente anterior.
  - Nunca sustituye el consentimiento nativo, los tokens exactos ni los permisos de publicación o destructivos.
  - Resolver una subdecisión no cancela el objetivo autorizado.
- [x] **T4 · Protocolo de Engram** (#2022, #5663). Commit `0a118489`, ruta delegada (writer, más de 4 ficheros), upstream `1c76e86a` y `4bae61e2`.
  - En `internal/assets/engram/protocol.md`:
    - Resolver el proyecto con `mem_current_project` antes del primer `mem_context`, `mem_search` o `mem_review`.
    - Esperar el resultado, nunca en paralelo.
    - Ramas: ambiguo, único o sin proyecto.
    - `content` como un único string JSON.
  - Regenerar `internal/components/engram/testdata/protocol-full.md` y `protocol_test.go`.
- [x] **T5 · `issue-creation` sigue la política del repositorio destino** (#5907). Commit `48117e8b`, ruta delegada (mismo writer), upstream `6e02c6ed`, solo la skill y sus tests.
  - En `internal/assets/skills/issue-creation/SKILL.md`, la política del destino manda: formulario, plantilla Markdown o ruta sin plantilla.
  - Actualizar las referencias y los tests de autoridad (`internal/assets/issue_creation_authority_test.go`).

## Fuera de alcance

- `docs/upstream-absorption-ledger.md`, por la decisión D4.
- Carriles y fases SDD: los decide la sesión «carril SDD versátil».
- Risk line y Delegated Verification Gate: los decide la sesión de revisión nativa.
- Retirada de SDD (#4967), RDD por defecto (#4803) y #4978.

## Dependencias con otras sesiones

- **Carril SDD versátil.**
  - T1 mueve secciones SDD de las plantillas. Si ese carril reescribe las secciones SDD, que lo haga sobre la fuente compartida que deje T1, o T1 espera.
  - #5218 y #5215 quedan para esa sesión.
- **Revisión nativa (lote C, #5351 parcial).** Edita `internal/assets/skills/_shared/sdd-orchestrator-sections.md:46-58` (Delegated Verification Gate), el mismo fichero que T1. Hay que ordenar los PRs para evitar conflictos.
- **Adaptadores de agentes.** El equipo usa Claude Code (`claude/`), OpenCode (`opencode/`) y Copilot CLI y VS Code Copilot (`generic/`) (Engram #1928).
- **Hoja de ruta de producto** (`docs/audits/2026-11-03-vision-producto.md` en la rama `inc/auditoria-vision-producto`, Engram `planning/vision-producto-hoja-de-ruta`):
  - **D3 / P5.** ¿Hay que distinguir la extensión de VS Code, Desktop y CLI de Claude Code? Si se distinguen, podría hacer falta una variante de plantilla o de entrega del orquestador por entorno de Claude. Eso afecta a T1: qué secciones son propias de cada runtime y cómo se seleccionan en `sddOrchestratorAsset`.
  - **D4 / P6.** ¿El trabajo ODD integra en la spec viva? Afecta a T2 (bloque ODD de `routing.go`) y a #5215 (formato del documento ODD). Lo explora ahora la sesión del carril SDD versátil.
  - **P1, distribución** (rama `inc/distribucion-un-comando`, sin PR): no se solapa. Comprobado con `git diff origin/main...inc/distribucion-un-comando --stat`, que no toca plantillas del orquestador, `agentguidance`, `_shared`, el protocolo de Engram ni `issue-creation`.

## Orden de ejecución entre bloques

Lo fija la sesión coordinadora a partir del cruce de ficheros entre bloques (2026-11-05). No se implementa nada hasta que el usuario lo autorice y estén resueltas P1–P3.

- **T4 y T5:** en paralelo, ya. No comparten ficheros con los demás bloques.
- **T1** es el eslabón 3 de la cadena en serie de goldens del orquestador:
  1. Arreglos T9 + adaptadores T1.
  2. Revisión T6. Edita la sección «Delegated Verification Gate» de `internal/assets/skills/_shared/sdd-orchestrator-sections.md:46-69`, así que debe entrar antes de que T1 mueva esas secciones como bloques opacos.
  3. **T1 de este bloque.**
  4. T2 y T3 de este bloque.
  5. Carril SDD versátil. Reescribirá las secciones de kickoff y compuertas, por eso T1 las mueve sin tocar su texto.
- **Techos de tamaño.** T1 y las tareas T14, T16 y T17 de revisión pueden mover los techos de `review_ledger_contract_test.go`. Hay que recalcularlos en el PR que entre segundo.
- **T2 y T3** (`internal/components/agentguidance/routing.go`, digests de `internal/agents/capabilitymanifest/manifest.go` y `internal/components/sdd/inject.go:1661-1700`) van antes que adaptadores T5 (Copilot CLI). Adaptadores T5 añade una entrada en `manifest.go:334` y toca `internal/components/sdd/orchestrator.go:114,189`, la misma función (`sddOrchestratorAsset`) que T1 cambia si se adopta D3.
- **Con P6 (carril SDD versátil)** se comparte el bloque ODD de `routing.go`.
- Con este orden, P3 queda resuelta: el PR de revisión nativa (T6) entra antes que T1.

## Decisiones abiertas para validar

- **P1 (resuelta, 2026-11-05).** ¿T1 cubre las 12 plantillas o solo `claude`, `opencode` y `generic`, que son las que usa el equipo? **Las 12.** Dejar plantillas fuera mantendría la deriva que T1 quiere eliminar.
- **P2 (resuelta, 2026-11-05).** ¿Dónde vive la fuente única de la delegación: en `RenderRouting` (Go, ya unificado y con el manifiesto) o en el markdown compartido? **En `RenderRouting`.** Ya concentra la regla junto con el manifiesto y sus digests; el markdown compartido se limita a referenciarla.
- **P3 (resuelta, 2026-11-05).** ¿En qué orden se hace T1 frente al cambio de `sdd-orchestrator-sections.md` de la sesión de revisión nativa? Primero revisión T6, luego T1 (véase «Orden de ejecución entre bloques»).

## Restricciones

- TDD desactivado (`openspec/config.yaml`, `strict_tdd: false`); el runner es `go test` con `-run`.
- Nunca se ejecuta `go test ./internal/cli/` completo en local.
- No se compilan binarios sueltos.
- RDD apagado (configuración global).
- El riesgo se evalúa con `axiom review assess --cwd <worktree> --base-ref origin/main --committed-only --json`.
- Artefactos en castellano y commits convencionales sin atribución de IA.

## Entrega

- **Previsión.**
  - T1: más de 400 líneas, sobre todo borrado de duplicados.
  - T2: unas 300 líneas.
  - T3, T4 y T5: menos de 100 líneas cada una.
- **Estrategia.** `ask-on-risk`. Al superar unas 400 líneas se pregunta una vez por la cadena (`stacked-to-main` o `feature-branch-chain`).
- **Rebanadas previstas.** T1 | T2+T3 | T4+T5. T4 y T5 son independientes y pueden ir antes.

## Criterios de aceptación

- No queda ninguna regla de delegación por número de ficheros en Go, plantillas ni goldens.
- Cada sección compartida se renderiza una vez por plantilla y lo cubre un test de no deriva.
- Las secciones SDD, la autorización remota y el texto de RDD opcional quedan intactos byte a byte, salvo la reubicación.
- Pasan `go test` de `./internal/assets/...`, `./internal/components/...` y `./internal/agents/capabilitymanifest/...`, y `go vet` de los paquetes tocados.

## Registro

- 2026-11-04: exploración y veredictos. Las notas están en la carpeta compartida de sesiones (`guia-orquestador-veredictos.md`) y en Engram (`upstream/gentle-ai-review-2026-11/guia-orquestador`). El usuario elige el carril ODD.
- 2026-11-04: se anotan las dependencias D3 / P5 y D4 / P6. Se comprueba que `inc/distribucion-un-comando` no se solapa con este bloque. El usuario valida el alcance T1–T5.
- 2026-11-05: se anota el orden de ejecución entre bloques que fija la sesión coordinadora. T4 y T5 van ya en paralelo; T1 es el eslabón 3 de la cadena de goldens y T2/T3 el 4. P3 queda resuelta por ese orden.
- 2026-11-05: la coordinadora autoriza arrancar T4 y T5. Ninguna de las dos depende de P1 ni de P2.
  - **T4** (`0a118489`, +133/−35):
    - `protocol.md` resuelve el proyecto con `mem_current_project` antes del primer acceso, espera el resultado y aplica las ramas por orden: comprobación del workspace, ambiguo, único y sin proyecto.
    - La recuperación entre proyectos solo se hace si el usuario la pide.
    - `content` pasa a ser un único string.
    - Se regeneran los fixtures `protocol-full.md` y `protocol-slim.md` y 5 goldens.
  - **T5** (`48117e8b`, +49/−16): la skill sigue la política del repositorio destino, que puede ser formulario, plantilla Markdown o ninguna plantilla. Se añade el test `TestIssueCreationFollowsDestinationRepositoryPolicy`.
  - **Verificación del writer** (todo ok): `go test ./internal/components/engram/...`, `go test ./internal/assets/...`, `go test ./internal/components/` (tras `-update`) y `go vet`.
  - **Comprobación puntual del padre:** `go test ./internal/components/engram/... ./internal/assets/ -run "IssueCreation|Protocol"`, ok.
  - **RDD apagado:** no se abre revisión nativa.
  - **Seguimiento fuera de alcance:** `.codex/AGENTS.md` sigue con el texto antiguo del protocolo de Engram.
- 2026-11-05: el usuario resuelve P1 (las 12 plantillas) y P2 (fuente única en `RenderRouting`) según la recomendación, y autoriza el push y el PR de T4 y T5. T1 sigue esperando a que se fusione revisión T6.
- 2026-11-10: revisión T6 entra con el PR #233. Se retira el worktree fusionado con `done`. T1 se implementa sobre `origin/main` b7ddbf51, que ya incluye #233, #234, #235 y #236.
  - **Rama:** el commit nació en `feat/guia-orquestador-t1`, pero la guarda de ramas exige que el documento de este cambio viva en `inc/guia-orquestador-upstream`. Se recrea el worktree con esa identidad y se trae el commit con cherry-pick; el árbol es idéntico.
  - **T1** (`999e6172`, 31 ficheros, +956/−1781):
    - Las 12 plantillas pasan de unas 4750 a unas 3375 líneas.
    - `_shared/sdd-orchestrator-sections.md` gana 37 secciones compartidas. Entre ellas están Delegation Rules y Mandatory Delegation Triggers, en dos secciones; T2 las sustituirá desde `RenderRouting`.
    - Las secciones SDD (kickoff, compuertas, Dispatcher Guard y Delegated Verification Gate) se quedan como bloques opacos, sin cambiar ni un byte.
    - Se quedan propias de cada runtime: la frase de la herramienta de delegación, Kiro steering, Windsurf Plan/Code, Codex capability check, Hermes native delegation y OpenCode entry routing.
  - **Deriva unificada:**
    - Native Checking Contract: «and the terminal receipt» en claude, codex, generic y opencode.
    - Regla 5 de los disparadores en generic: «native review actors».
    - Execution Mode en qwen: «¿Continuamos?».
    - Espacios en hermes.
    - Cambian 7 goldens, y solo por las dos primeras unificaciones.
  - **Consumidores:** todos pasan por `composeOrchestratorPrompt`, que ahora resuelve las secciones antes de `composeFallbackSessionPreflight`. Los tests que leían la plantilla en crudo leen la versión resuelta.
  - **Test de no deriva** (`internal/components/sdd/orchestrator_no_drift_test.go`) en las 12 plantillas:
    - No quedan marcadores sin resolver.
    - Cada sección se expande una vez por referencia.
    - Ninguna plantilla copia el cuerpo de una sección compartida.
    - Toda sección compartida se usa.
  - **Ajustes en tests:**
    - Se bajan los techos del ratchet de deriva.
    - Cambia el SHA de `TestKilocodeReviewSettingsMatchCurrentMainBaseline`.
    - `TestSDDSkillsNameTheIntegrationBranchNotALiteral` admite 4 patrones nuevos de prosa de canales de release.
    - No cambian los techos de `review_ledger_contract_test.go`.
  - **Verificación del writer:** `go test` de `./internal/components/...`, `./internal/assets/...` y `./internal/agents/capabilitymanifest/...`, todo ok salvo `filemerge`. `go vet` limpio.
  - **Fallo preexistente:** `TestWriteFileAtomicFollowsDirectoryJunction` y `…RelativeDirectoryJunction`, en `filemerge`, por junctions de Windows («file already exists»). Falla igual en `main`, comprobado por el padre.
  - **Comprobación puntual del padre:** `go test ./internal/components/sdd/ ./internal/components/ ./internal/assets/ -count=1`, ok. Con word-diff, los goldens solo cambian en «+the» ×7 y «checking»→«review».
  - **Riesgo:** `axiom review assess` da `medium` y RDD está apagado. Basta la autoverificación del writer (sonnet, no es perfil pequeño) y la comprobación puntual.
  - **Entrega:** 2737 líneas, por encima de 400 → un solo PR con `size:exception` (preferencia del usuario).
