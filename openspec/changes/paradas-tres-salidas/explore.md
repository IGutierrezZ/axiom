# Exploración: paradas con tres salidas (ODD y SDD) en lugar de `handoff_policy` y `handoff.md`

Cambio: `paradas-tres-salidas` (rama `inc/paradas-tres-salidas`). Decisión de origen: Engram #1811 (`product/paradas-tres-salidas`).
Todas las referencias son `fichero:línea` del worktree `C:\repos\axiom-wt\paradas-tres-salidas\axiom` (rutas relativas a su raíz). No se ha ejecutado ni compilado nada.

## 1. Estado actual

### 1.1 `handoff.md` final de integración (pregunta 1)

- **Quién lo escribe.** `internal/cli/sdd_gate.go:135-148`: al registrar `approved` de una puerta `role-apply:<rol>`, `lastRoleJustClosed` (`:242-264`, reutiliza `kickoff.EvaluateGates` + `kickoff.LastRoleClosed`) decide si ya están todos los roles cerrados; si sí, `writeIntegrationHandoff` (`:273-287`) construye el relevo con `kickoff.IntegrationHandoff` (`internal/kickoff/closure.go:44-76`, `from_phase: apply`, `to_phase: verify`, `status: ready`, `to_role` por la regla D-11 `:108-118`) y lo escribe con `handoff.WriteFile`. Imprime además el aviso «ultimo rol activo» (`:25`, `:144`, REQ-21.13).
- **Cómo desbloquea verify.** `internal/sddstatus/governance.go:491-546` (`verifyDependencyFromHandoff`, llamada en `internal/sddstatus/status.go:730`):
  - `!governanceGated` (continuous o sin kickoff) devuelve el baseline (`:510-512`).
  - Solo consulta `handoff.md` si el roster tiene más de un rol **o** `HandoffPolicy == per_checkpoint` (`:513-517`).
  - `handoff.md` ausente: bloquea **solo si** `LastRoleClosed` ya es cierto (`:526-528`); si no, baseline.
  - `status: ready` y `to_phase: verify` sube a `ready` incondicionalmente (`:533-537`); `blocked` o `needs_clarification` bloquea (`:538-542`).
- **Qué valida `internal/handoff/validator.go`.** Cinco metadatos obligatorios, estado en `ready|blocked|needs_clarification`, cinco secciones no vacías, transición de fase legal (`forwardTransitions` `:29-37`; retrocesos solo con `blocked|needs_clarification` `:40-43,110-121`) y, con `axiom.yaml` con roles, que `from_role`/`to_role` existan (`:126-139`).
- **Condición sustitutiva que ya existe.** `kickoff.LastRoleClosed` (`internal/kickoff/machine.go:152-178`): todas las `role-apply:<rol>` del roster en `approved` (roster vacío nunca cierra). Ya la usan `sdd_gate.go` y `governance.go:526,574-583`. Sustituir el relevo por «todos los role-apply aprobados» no necesita lógica nueva.
- **Matiz.** `dependencies.Verify` base es `ready` en cuanto `coreReady` (`internal/sddstatus/status.go:1694-1714`) y el contrato de status ya dice que verify es un diagnóstico opcional (`internal/assets/skills/_shared/sdd-status-contract.md:147`). Hoy el ruteo hacia `verify` ya lo frena `await-gate` (`status.go:1732-1736`), no el relevo.
- **`handoff.md` no es solo de SDD.** Lo consumen `axiom handoff show|create|validate` (`cmd/axiom/main.go:132-134,331-346,677-815`), el dashboard (`internal/dashboard/service.go:575-592,1095-1180`, `POST /api/handoffs`), la TUI (`internal/tui/model.go:6156-6173`, `internal/tui/screens/handoffs.go`), la spec `structured-handoffs` y el manual (`docs/manual-de-inicio.md:515-566`).

### 1.2 Puertas y estilo de ejecución (pregunta 2)

- **Modelo.** Cuatro claves fijas `spec|design|tasks|integration` y `role-apply:<rol>` (`internal/kickoff/types.go:59-62,76-84`). Decisión cerrada `approved|rejected` (`types.go:64-65`, `internal/kickoff/schema.go:48-51,143-145`). `gate record` exige `--reason` solo al rechazar (`internal/kickoff/args.go:458-460`).
- **Ledger.** Solo-anexar (`gates.yaml`, REQ-21.12). `resolveGateStatus` (`internal/kickoff/machine.go:198-212`): una aprobación en cualquier punto es **terminal**; sin aprobación, el último registro manda mientras el digest del artefacto coincida; si el digest cambió, vuelve a `pending` con `Reopened`.
- **Evaluación y ruteo.** `EvaluateGates` devuelve `nil` en continuous (`machine.go:51-54`). `governanceGated` (`internal/sddstatus/governance.go:325-327`) es la comprobación única. Una puerta abierta tiene precedencia sobre cualquier ruta (`status.go:1732-1736`): `pending` va a `await-gate`; `rejected` va a la fase dueña (`status.go:1773-1806`).
- **Pregunta tipada.** `newGovernanceGateQuestion` (`governance.go:235-282`) construye el sobre `gateQuestion` con dos opciones, `approved` y `rejected`, que comparten el prefijo `axiom sdd gate record`. `Validate` (`:107-141`) y el núcleo compartido `consentenvelope.Core.ValidateCompleteness` (`internal/consentenvelope/envelope.go:57-78`) **exigen exactamente dos opciones** (`len(core.Choices) != 2`). Ese núcleo lo comparte también el sobre de autoridad de edición.
- **Estilos.** `continuous` no tiene paradas (REQ-21.7); `checkpointed` sí. `--from-session-pace interactive|auto` se traduce a `checkpointed|continuous` (`args.go:186-196,245-256`). El ritmo de sesión `interactive` («tras cada fase, resumen y preguntar si ajustar o continuar») ya es una parada de dos salidas (`internal/assets/claude/sdd-orchestrator-workflow.md:68-73`).
- **Encaje de las tres salidas.**

| Salida | Encaje hoy |
|---|---|
| 1. Iterar | Equivale a `rejected` con motivo: el ledger ya conserva el rechazo, `status` enruta a la fase dueña (`status.go:1778-1781`) y la remediación reabre la puerta por digest. No hace falta vocabulario nuevo. |
| 3. Seguir uno mismo | Equivale a `approved` (terminal, desbloquea la fase siguiente). |
| 2. Subir a main y parar | No existe. Es una acción de git más el fin de la sesión. |

- **`approved` no sirve para la salida 2.** Es terminal y el receptor no podría rechazar después: `resolveGateStatus` devuelve `approved` aunque haya registros posteriores (`machine.go:199-203`).
- **Hoy hay precursores a nivel de skill.** `internal/assets/skills/sdd-spec/SKILL.md:242-250,269` y `internal/assets/skills/sdd-design/SKILL.md:183-193,208` formulan la «Pregunta de Transición» de dos salidas («¿generar relevo formal `handoff.md` o continuar?») y mandan hacer `git commit` y dejar push/PR al humano. Son el `handoff.md` intermedio, ligado a ODD-1.3. No hay pregunta equivalente en `proposal`, `tasks` ni `apply`.

### 1.3 `handoff_policy` y kickoff (pregunta 3)

- Tipos: `internal/kickoff/types.go:26-28` (`HandoffPolicy`), `:56-57` (`none|per_checkpoint`), `:105` (`yaml:"handoff_policy"`, sin `omitempty`).
- Validación: `internal/kickoff/schema.go:25-28,89-91`.
- Flag obligatorio: `internal/kickoff/args.go:197-207` (parseo) y `:257-259` (`kickoff seal requiere --handoff-policy`); `ToKickoff` lo escribe en `:296`.
- Retro-sello: `internal/kickoff/infer.go:22,89` (`none`).
- Salida: `internal/cli/sdd_kickoff.go:36` (ayuda), `:131-132` (resumen de texto), `:160,182` (JSON `handoffPolicy`); `internal/sddstatus/status_v2.go:125,172` (JSON de `governance.kickoff`); `internal/assets/skills/_shared/sdd-status-contract.md:127`.
- Único lector con efecto: `governance.go:514`.
- Mensaje de error que sugiere el flag: `internal/cli/sdd_gate.go:89`.
- Texto proyectado a los 12 orquestadores desde una sola fuente: `internal/assets/skills/_shared/sdd-orchestrator-sections.md:74` (`--handoff-policy none|per_checkpoint`). Se compone vía marcador `{{GENTLE_AI_SDD_SECTION:SDD Change Kickoff and Block Gates}}` (`internal/assets/claude/sdd-orchestrator.md:57-59`) y `internal/components/sdd/orchestrator.go`.
- Copias renderizadas y versionadas con el flag: 11 goldens (`testdata/golden/sdd-*.golden`, `combined-*.golden`; se regeneran con `-update`, `internal/components/golden_test.go:32`) y `.codex/AGENTS.md:634`.
- Pruebas y fixtures: unas 25 referencias en `internal/kickoff/*_test.go`, `internal/cli/*_test.go`, `internal/sddstatus/*_test.go`, `internal/branch/engine/{status,snapshot}_test.go`, `e2e/organicruntime/sdd_consent_wiring_e2e_test.go:72`, `cmd/axiom/handoff_branch_guard_test.go:55`, `bench/journeys_sdd.go:151` y `bench/journeys_issue_2891.go:36`.
- **Lectura de kickoffs ya sellados.** `ParseKickoff` usa `yaml.Unmarshal` no estricto (`schema.go:67`; no hay `KnownFields`): una clave desconocida se ignora. Pero `Validate` rechaza hoy cualquier valor fuera de `none|per_checkpoint` (`schema.go:89`), así que **dejar `handoff_policy` vacío o ausente fallaría** hasta cambiar la validación. Hay kickoffs archivados con la clave (`openspec/changes/archive/*/kickoff.yaml`) y el kickoff de este propio cambio (`handoff_policy: none`).
- **Dato de compatibilidad.** El flag es obligatorio hoy. Cualquier orquestador con instrucciones renderizadas por un `axiom sync` anterior seguirá pasando `--handoff-policy`; un binario que lo rechazara como «bandera desconocida» (`args.go:229-230`) rompería el sellado hasta que el usuario resincronice.

### 1.4 «Subir a main y parar» (pregunta 4)

- `internal/cli/branch.go:193-199`: `ensure`, `status`, `done`, `guard`. No existe `pr` ni `push` en el producto (solo el envoltorio local `scripts/axiom-worktree.ps1 pr`, que no es producto). Invariante: Axiom nunca hace push ni PR (`internal/branch/planner.go:201`, `docs/manual-de-inicio.md:230`).
- `status` ya entrega por rama `prTarget`, `ahead/behind`, `dirty`, rama de integración (`internal/branch/result.go:173-236`).
- **`monorepo-embedded`.** Rama `<prefijo>/<cambio>` con `PRTarget = integración` (`internal/branch/planner.go:389-394`). Con 2 o más roles hay rama de seguimiento y PR por rol contra ella (`:395-414`; `openspec/specs/change-branch-lifecycle/spec.md:249-251`). El archive va en otra rama `<cambio>-archive` y otro PR documental (spec `:251`, REQ-25.45).
- **`multirepo`, repo de specs.** Se trabaja **in situ** en la rama de integración, sin rama de cambio, rol ni worktree (`planner.go:367-369`, `spec.md:307`); `ensure` rechaza si no está en integración (`specs_repo_not_on_integration`). Allí «subir a main» es commit directo sobre la integración más push humano, **no hay PR posible desde una rama de cambio**.
- **Qué falta para la salida 2 sin que Axiom publique.** Nada del motor de ramas. Haría falta, en el orquestador, (a) un commit convencional de los artefactos en la rama del cambio (ya lo mandan los skills), (b) leer `axiom branch status --json` y presentar al humano la rama, el `prTarget` y las órdenes `git push` / PR exactas, y (c) parar. Opcional: una verificación de que los artefactos están confirmados, para la que hay precedente (`kickoff.AncestryChecker`, `internal/kickoff/integration.go`; `internal/cli/sdd_gate_evidence.go`).
- **Efectos laterales a vigilar.**
  - «Cambio pendiente de archivar» se define como «activo y presente en el árbol de la integración» (`change-branch-lifecycle/spec.md:31`, REQ-25.20), y `ensure --archive` acepta la base cuando su árbol contiene `changes/<cambio>/` (S-16, `:1292`). Una fusión **intermedia** (solo spec y plan) haría que el cambio aparezca como «ya fusionado» y que `ensure --archive` no devuelva `change_not_integrated` aunque falte implementar.
  - Quien recoge el testigo en otra máquina ejecuta `ensure`, que crea `<prefijo>/<cambio>` **desde la base** y nunca mueve una rama existente (`internal/branch/engine/ensure.go:38-66`). Con solo la rama empujada y sin fusión, ese `ensure` no recuperaría los artefactos. Por eso la salida 2 debe significar «fusionado en integración», como dice la decisión.
  - Con el almacén `engram` o `none` no hay ficheros que subir (`sdd-status-contract.md:118` ya lo trata para el archive). La salida 2 solo es real con `openspec` o `hybrid`.

### 1.5 ODD (pregunta 5)

- ODD es doctrina de prompt, no código: la CLI `axiom odd` se retiró (`openspec/specs/odd-cli-commands`, `odd-living-document`, `odd-sdd-promotion`: todo REMOVED).
- Fuente única: `internal/components/agentguidance/routing.go` (`RenderRouting`; se inyecta en todos los agentes por `inject.go:95`). Protocolo de 7 pasos más paso 0 en `:48-64`; detalle en `:103-121`; ramas en `:130-135`.
- No hay parada formal en ODD. Los únicos puntos de espera son la pregunta de selección de carril (paso 0, `:54`) y la pregunta de producto (paso 3, `:59`). El paso 0 dice «If the user chooses ODD, do not ask any further governance question» (`:54`), igual que REQ-21.2 de `sdd-lane-kickoff` («del tirón»).
- Pruebas que lo fijan: `internal/components/agentguidance/routing_test.go:258` (lista de pasos), `:364` (el paso 6 es el único que nombra `axiom branch`), `:473-508` (cláusulas del paso 0, incluida «do not ask any further governance question»).
- La reanudación de ODD usa `mem_context`/`mem_search` y el espejo de Engram (`routing.go:64,116`). El espejo es local a cada persona; el fichero `odd/tasks/<feature>.md` es el autoritativo. Para una persona distinta, solo el fichero existe.
- Otras copias de texto de ODD: `.codex/AGENTS.md`, `AGENTS.md:16` (ODD-5, «Antes de ceder el testigo en `handoff.md`…»), `GEMINI.md`, `docs/manual-de-inicio.md:230`.
- Textos de las puertas SDD (una sola fuente, 12 orquestadores): `internal/assets/skills/_shared/sdd-orchestrator-sections.md:71-81`. Pruebas que lo fijan: `internal/assets/assets_test.go:2422-2470` (frases de las puertas y delta de contexto) y `:3798-3840` (orden `ensure` → pre-vuelo → `seal`). Los goldens y `.codex/AGENTS.md` lo reproducen.
- Pi (`model.AgentPi`): no existe `internal/assets/pi/sdd-orchestrator.md`; `AgentPi` aparece en `internal/components/sdd/orchestrator.go`. Falta confirmar en design por qué ruta recibe las puertas. El routing de ODD sí le llega por `agentguidance` como a todos.

### 1.6 Specs afectadas (pregunta 6)

| Spec | Requisito | Efecto |
|---|---|---|
| `sdd-preflight-configuration` | REQ-21.5 (modalidad **y política de relevos** selladas antes de la propuesta) y propósito (`:7,13,17`) | **Modificar.** Solo se sella la modalidad (`execution_style`) y los roles. Se retiran la «política de relevos» y el escenario correspondiente. |
| `sdd-lane-kickoff` | REQ-21.2 (ODD sin preguntas de gobernanza ni `handoff.md`, `:36-49`); REQ-21.4 (escenarios `:74-83` citan «política de relevos» y «sin `handoff.md` intermedios») | **Modificar.** REQ-21.2 choca con las paradas de ODD (ver preguntas abiertas). REQ-21.4: retirar menciones a la política de relevos. |
| `sdd-block-review-gates` | REQ-21.7 a 21.12 | **Ampliar.** REQ-21.7 (continuous sin paradas) puede mantenerse. REQ-21.8 a 21.11 pasan de «aprobar o rechazar» a «tres salidas». REQ-21.12 (rechazo = remediación) queda como la salida 1. |
| `sdd-role-closure-handoff` | REQ-21.13 (aviso del último rol), 21.14 (genera `handoff.md`), 21.15 (verify no arranca sin `handoff.md` ready) | **Retirar REQ-21.14** y sus escenarios; **reescribir REQ-21.15** («verify global tras todos los `role-apply` aprobados»); **conservar el aviso de REQ-21.13** sin generar fichero. La capacidad pierde su razón de ser y puede fusionarse en `sdd-block-review-gates` o quedar reducida. |
| `change-branch-lifecycle` | REQ-25.20 y S-16 | **Posible enmienda** si se aceptan fusiones intermedias (ver riesgos). |
| `structured-handoffs`, `local-web-dashboard`, `dashboard-sdd-orchestration`, `tui-ui-parity`, `planning-home-resolution`, `axiom-binary-ci-coverage` | `axiom handoff`, dashboard y TUI | **Sin cambio** si se conserva el artefacto como utilidad general (recomendado). |
| `organic-agent-trigger-rules` | Proyección del routing/ODD | **Ampliar** con las paradas de ODD (REQ nuevo). |
| `sdd-orchestrator-assets` | Plantillas del orquestador | Posible requisito de pin de texto de la pregunta de tres salidas. |

`openspec/INDEX.md` lo regenera `axiom archive sync`; no se edita a mano.

### 1.7 Solapamiento con `fix/bloqueos-prueba-multirepo-r2` (pregunta 7)

- **Limitación.** En esta fase no tengo `Bash`, así que **no he podido ejecutar `git diff main...fix/bloqueos-prueba-multirepo-r2 --stat`**. La orquestadora debe lanzarlo.
- **Lo que sí he comprobado leyendo el worktree `C:\repos\axiom-wt\bloqueos-prueba-multirepo-r2\axiom`.** `internal/cli/sdd_governance.go` (`resolveGovernanceChangeRoot`, que hoy rechaza crear la carpeta del cambio) es idéntico al de mi worktree; `internal/sddstatus/governance.go` tiene el mismo número de líneas (541) y `status.go` las mismas coincidencias de `apply-progress|verify-report|proposal` (30). Es decir, ese checkout no muestra todavía esos arreglos en los ficheros comprobados (o están sin confirmar o en otros ficheros). Su `odd/tasks/bloqueos-prueba-multirepo.md` es la ronda 1, ya cerrada, y no describe la ronda 2.
- **Solapamiento previsible por el contenido de los arreglos descritos.**

| Arreglo de la otra rama | Ficheros que este cambio también toca | Riesgo |
|---|---|---|
| Retransmitir `gateQuestion` | `internal/sddstatus/governance.go` (`newGovernanceGateQuestion`), `status.go:751-760`, `sdd-status-contract.md`, `sdd-orchestrator-sections.md`, `assets_test.go`, goldens | **Alto**: mismas funciones y mismo texto compartido; los goldens solo se concilian regenerándolos. |
| `apply-progress`/`verify-report` por rol | `internal/sddstatus/governance.go` (`governanceArtifactInputs`, `verifyDependencyFromHandoff`), `status.go` | **Medio**: `governance.go` lo toca la retirada de `verifyDependencyFromHandoff`. |
| `kickoff seal` crea la carpeta del cambio | `internal/cli/sdd_kickoff.go`, `sdd_governance.go`, `internal/kickoff/args.go` | **Bajo-medio**: este cambio retira `--handoff-policy` de `args.go` y de `sdd_kickoff.go:36,131,160,182`; hunks distintos pero en los mismos ficheros. |
| Plantilla de proposal que deja de contar como `done` | `internal/sddstatus/status.go` (resolución de artefactos) | **Bajo.** |

- **Recomendación de orden.** Fusionar la rama r2 primero (o rebasar este cambio sobre ella) y regenerar los goldens al final.

## 2. Enfoques

### 2.1 Qué hacer con `handoff_policy`

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| **A. Eliminar del todo** (campo, flag, validación, JSON, textos) | Superficie limpia; no queda un campo muerto. | El flag es obligatorio hoy: un binario nuevo rompe el sellado con orquestadores sincronizados antes. Cambia el contrato JSON de status y de `kickoff show`. Toca ~25 fixtures de test. | Medio-alto |
| **B. Deprecado: aceptado y ignorado** (recomendado) | Compatible con orquestadores y kickoffs antiguos. Se deja de exigir el flag; no se escribe en `kickoff.yaml` nuevos (`omitempty`); `Validate` acepta `""`, `none` y `per_checkpoint`; `governance.go:514` deja de leerlo. Los textos, la ayuda y los tests nuevos ya no lo nombran. Las pruebas existentes que lo pasan siguen compilando. Se elimina del todo en una versión posterior. | Queda un campo muerto temporal. Hay que decidir si el JSON de status lo sigue emitiendo (propongo `omitempty`). | Bajo-medio |
| **C. Reutilizar el campo** para configurar la salida por defecto de las paradas | Reaprovecha el sello. | Mezcla dos conceptos y la decisión de producto dice que se sustituye, no que se reconvierte. | Medio |

Lectura de un kickoff ya sellado con `per_checkpoint`: en B se acepta y se ignora (sin efecto, sin aviso bloqueante). `kickoff show` puede imprimirlo marcado como obsoleto solo si está presente.

### 2.2 Cómo representar la salida 2 en las puertas

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| **P1. Solo texto** (orquestador/skills): la pregunta de la puerta sigue con dos opciones; el orquestador ofrece un «parar aquí» fuera del sobre | Mínimo; sin cambios en ledger ni envelope. | El contrato de «Lossless Blocking Prompt» prohíbe añadir opciones que no vengan del sobre; no hay auditoría; nada verifica que lo subido esté confirmado. | Bajo |
| **P2. Tercera decisión en el ledger** (`handed_off`) y tercera opción del sobre | Auditable; el sobre lo relata de forma literal. | `ParseGateLedger` rechaza decisiones desconocidas (`schema.go:143-145`): un `axiom` antiguo de otra persona del mismo repo **no podría leer `gates.yaml`**. Obliga a generalizar `consentenvelope.Core.ValidateCompleteness` (hoy exactamente 2 opciones, compartido con el sobre de autoridad de edición) y a ajustar `resolveGateStatus` (un último registro no-aprobado hoy cuenta como `rejected`). | Alto |
| **P3. Tercera opción del sobre sin tocar el ledger** (recomendado): la opción «parar y subir» invoca un verbo nuevo de solo lectura/verificación (p. ej. `axiom sdd gate handoff`), que comprueba que los artefactos de la puerta están confirmados y emite la rama, el `prTarget` y las órdenes `git push`/PR exactas, **sin escribir** `gates.yaml` ni ejecutar nada remoto | Sin riesgo de compatibilidad del ledger; la puerta sigue `pending` y quien recoge el testigo ve el mismo `await-gate`; el rastro queda en git. Respeta «push/PR son decisión humana». | Hay que generalizar el validador del sobre a N opciones (cambio local, con prueba de que el sobre de autoridad de edición sigue con 2). No queda registro de la parada en `gates.yaml`. | Medio |
| **P4. Salida 2 = `approved` + parar** | Sin vocabulario nuevo. | Semánticamente incorrecta: `approved` es terminal (`machine.go:199-203`), el receptor no podría rechazar. | Bajo |

La salida 1 reutiliza `rejected`; la 3, `approved`. En ambos casos no cambia el comportamiento actual.

### 2.3 Paradas en ODD

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| **O1. Sin parada formal** (documentar solo en el paso 3) | No cambia nada. | No cumple el objetivo. | Nulo |
| **O2. Una parada en ODD tras el paso 5** (documento de feature creado, antes de la primera escritura de código) con las tres salidas, bajo un bloque común «Stops and exits» en `routing.go` que SDD referencia | Es el equivalente de «esta es la spec / el plan». Un único texto para ODD y SDD. Salida 2: commit de `odd/tasks/<feature>.md` en la rama ODD y presentar `prTarget`. | Choca con «del tirón» (REQ-21.2 y paso 0). Hay que decidir si es por defecto o a petición. | Medio |
| **O3. O2 más paradas al cierre de cada unidad de trabajo** | Cubre PRs encadenados (`delivery_strategy`). | Más fricción; solapa con la estrategia de entrega ya existente. | Medio-alto |

## 3. Recomendación

1. **`handoff_policy`: enfoque B** (aceptar y ignorar, no exigir, no escribir). Retirada completa en un cambio posterior.
2. **`handoff.md` de integración:** retirar `writeIntegrationHandoff` y `kickoff.IntegrationHandoff` (`closure.go`) y reescribir `verifyDependencyFromHandoff` como «en un roster multirol con puertas, verify bloqueado mientras no estén todos los `role-apply` aprobados» (`LastRoleClosed`). Conservar el aviso de último rol (sin generar fichero) y **conservar `internal/handoff`, `axiom handoff`, dashboard y TUI** como utilidad general: retirarlos ensancha el cambio sin que la decisión lo pida. Retirar la «Pregunta de Transición» de `sdd-spec` y `sdd-design`.
3. **Salida 2 en SDD: enfoque P3.** Salida 1 = `rejected`, salida 3 = `approved`, salida 2 = tercera opción del sobre que invoca un verbo de verificación/presentación, sin tocar el ledger.
4. **ODD: O2**, con el texto de las tres salidas definido una vez en `routing.go` y referenciado desde la sección compartida de puertas, y con la regla de que la parada de ODD es la excepción documentada a «sin preguntas de gobernanza».
5. **Orden de entrega:** tras la rama r2; goldens al final con `-update`.

## 4. Mapa de ficheros afectados

**Código (retirar o cambiar):**
- `internal/kickoff/types.go`, `schema.go`, `args.go`, `infer.go` (handoff_policy opcional y tolerante).
- `internal/kickoff/closure.go` y `closure_test.go` (retirar).
- `internal/cli/sdd_gate.go` (retirar `writeIntegrationHandoff`, `buildIntegrationArtifactInventory`; mensaje de `:89`; conservar aviso `:144`; nuevo verbo de salida 2), `sdd_kickoff.go` (`:36,131,160,182`).
- `internal/sddstatus/governance.go` (`verifyDependencyFromHandoff`, `newGovernanceGateQuestion`, `Validate`), `status.go:730`, `status_v2.go:125,172`.
- `internal/consentenvelope/envelope.go` (validador de N opciones).
- `internal/branch/engine/writer_inventory_test.go:113-114` (entradas del inventario de escritores).

**Textos proyectados:**
- `internal/assets/skills/_shared/sdd-orchestrator-sections.md:74,76`, `sdd-status-contract.md:127,135`, `openspec-convention.md:26,47`.
- `internal/assets/skills/sdd-spec/SKILL.md:242-250,269`, `sdd-design/SKILL.md:183-193,208`.
- `internal/components/agentguidance/routing.go` (paradas de ODD).
- Copias versionadas: 11 goldens, `.codex/AGENTS.md`, `AGENTS.md:16`, `GEMINI.md`, `docs/manual-de-inicio.md:230,499-500,515-566`.

**Pruebas a reescribir o retirar:**
- `internal/sddstatus/handoff_dependency_test.go` (entero), `governance_continuous_test.go:93-126,161`, `governance_test.go`, `internal/cli/sdd_gate_test.go:211-213`.
- `internal/components/agentguidance/routing_test.go:258,364,473-508`; `internal/assets/assets_test.go:2422-2470,3798-3840`.
- Fixtures con `HandoffPolicy` (compatibles con el enfoque B sin tocarlos).

**Specs:** ver 1.6.

**Tamaño.** Estimación gruesa, no medida: 1.500 a 2.500 líneas de texto autorado (más deleciones que adiciones por `closure.go`, su test y `handoff_dependency_test.go`, más la reescritura de cuatro specs), sin contar los goldens regenerados. Encaja con la entrega de una sola PR con `size:exception` que pidió el usuario.

## 5. Riesgos

- **Desfase binario/instrucciones.** El flag `--handoff-policy` es obligatorio hoy y sigue en orquestadores ya sincronizados. Mitigación: enfoque B.
- **Compatibilidad del ledger entre versiones** si se añadiera una decisión nueva (P2). Mitigación: P3.
- **Radio de acción del validador del sobre.** `ValidateCompleteness` es compartido; generalizarlo exige una prueba de no regresión del sobre de autoridad de edición y del contrato de «Lossless Blocking Prompt» (`internal/assets/blocking_prompt_contract_test.go`).
- **Fusiones intermedias y archive.** Una PR de solo spec y plan deja el cambio «ya fusionado» a ojos de `status`/dashboard (REQ-25.20) y hace que `ensure --archive` no rechace con `change_not_integrated` (S-16). Hay que decidir si se aceptan o se matizan.
- **Recuperar el testigo con `ensure`.** Crea la rama desde la base y no recupera artefactos que no estén fusionados en la integración.
- **Almacén de artefactos.** Con `engram` o `none` la salida 2 no tiene qué subir.
- **Reanudación de ODD por otra persona.** El espejo de Engram es local; la instrucción de reanudación debe tolerar su ausencia y partir del fichero.
- **Choque con REQ-21.2 («del tirón»).** Las paradas de ODD contradicen hoy el paso 0 y su prueba (`routing_test.go:496`).
- **Superficie de pruebas y goldens.** Doce plantillas, once goldens y varias pruebas de pin de texto; toda edición de la sección compartida exige regenerar y revisar.
- **Solapamiento con la rama r2** (ver 1.7), no medido por falta de `git`.

## 6. Preguntas abiertas de producto (decide el humano)

1. **ODD:** ¿en qué punto exacto hay parada (propongo tras el paso 5, plan listo) y es por defecto o solo a petición/ritmo interactivo? Hoy ODD es «del tirón» (REQ-21.2).
2. **`continuous`:** hoy no tiene paradas. ¿Las tres salidas solo existen en `checkpointed` (y en ODD según la respuesta anterior), o `continuous` también las ofrece en algún punto?
3. **Alcance de las paradas SDD:** hoy hay puertas en `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration`. ¿Se añade parada tras `proposal` (no tiene puerta) o tras `explore`?
4. **Salida 2, qué significa «subir»:** ¿exige PR **fusionada** en la integración (la decisión dice «subirlo a main») o basta empujar la rama? Con rama sin fusionar, `ensure` en otra máquina no la recupera.
5. **Salida 2 en multirepo (repo de specs in situ):** ¿commit directo en la rama de integración y push humano, sin PR?
6. **Registro de la parada:** ¿basta el rastro de git (P3) o se quiere constancia en `gates.yaml` aun a costa de la compatibilidad entre versiones (P2)?
7. **`axiom handoff`, dashboard y TUI:** ¿se conservan como utilidad general (recomendado) o se retiran también?
8. **Verify multirol:** ¿bloqueado hasta que estén todos los `role-apply` aprobados (equivalente a lo de hoy) o sin bloqueo (verify ya es diagnóstico opcional según el contrato de status)?
9. **Fusiones intermedias:** ¿se acepta que un cambio activo y fusionado parcialmente aparezca como «pendiente de archivar», o se enmienda REQ-25.20/S-16 para distinguirlo?

## Listo para propuesta

Sí, con las preguntas 1, 4 y 6 resueltas (o asumidas explícitamente en la propuesta). El resto puede resolverse en `sdd-propose` con la recomendación del apartado 3.
