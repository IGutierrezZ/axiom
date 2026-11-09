# Tareas: Carril elegido en el proyecto, ODD sin paradas y publicación ligera de documentación SDD (carril-proyecto-publicacion-docs)

> Fase `tasks` · rama `inc/carril-proyecto-publicacion-docs` · 2026-11-09.
> Entradas: `proposal.md` (D1–D8, S1–S10), `design.md` (DD-01 a DD-16, §5 contratos, §7 matriz de amenazas, §8 migración) y las ocho deltas de `specs/*/spec.md` (REQ-27.1 a REQ-27.17; REQ-21.x, REQ-25.x y REQ-26.x enmendados o retirados).
> Kickoff sellado: `continuous`, rol único `fullstack` → un solo `tasks.md`. Sin puertas propias.
> Idioma: castellano. Identificadores de código, rutas, flags y claves en inglés.
> Orden de los incrementos: el de DD-16, **1 → 2 → 4 → 3** (los incrementos 1, 2, 3 y 4 son los de D8). Para no confundirlos, las unidades de trabajo se numeran **WU-1 a WU-4 con el número de D8** y se ejecutan en ese orden: WU-1, WU-2, WU-4, WU-3.

## Reglas de ejecución (válidas para todas las tareas)

- **TDD desactivado** (`strict_tdd: false`, confirmado en `proposal.md` §11): cada tarea lleva su prueba junto al comportamiento, en la misma unidad de trabajo; no se exige RED previo. Las filas aplicables de la matriz de amenazas (design §7) se listan como casos de prueba dentro de T-08, T-09, T-13, T-14 y T-15, con los nombres que fija el diseño.
- **Verificación solo con `go test` acotado por paquete y en proceso.** PROHIBIDO `go test ./...` completo y compilar o ejecutar binarios sueltos (`go build -o`, `axiom.exe`, `go run` a ejecutable temporal): el antivirus Cortex XDR los borra. Las pruebas de CLI usan los puntos de entrada en proceso (`RunSDDPublish`, `RunSDDGate`, `RunSDDKickoff`, `RunSDDStatus`, `RunBranch`). Las pruebas con git usan repositorios y **remotos bare temporales** (`git init --bare` en `t.TempDir()`, precedente `remoteWithMain` de `internal/branch/engine/archive_test.go`).
- **`go test ./internal/cli/` completo tarda más de una hora**: en `internal/cli` se usa **siempre** `-run` acotado (los patrones figuran en cada tarea). Los demás paquetes se ejecutan completos.
- **Goldens**: se regeneran en proceso con `go test ./internal/components/ -run Golden -update -count=1` (el flag `update` está declarado en `internal/components/golden_test.go:32`) y se vuelven a pasar sin él. Solo el paquete `internal/components`, nunca sus subpaquetes que compilan binarios. Los goldens no se concilian a mano. `.codex/AGENTS.md` se actualiza copiando los mismos bytes del bloque gestionado del golden `sdd-codex-agentsmd.golden`; no se ejecuta `axiom sync`.
- **Aislamiento (DD-14, REQ-27.11):** el código que ejecuta `git commit` y `git push` vive solo en `internal/sddpublish`, y solo `internal/cli/sdd_publish.go` lo importa. Ningún método de `branch.Git`, de `gitWriter` ni de `gitcli.Git` publica.
- **Fuera de alcance:** publicación para ODD (`odd/tasks/**`), push de código, PR o merge desde Axiom, `fetch` implícito, `--force`, `--no-verify`, la puerta `integration`, el vocabulario del ledger y RDD (proposal §4.2). No se edita `openspec/INDEX.md` a mano.
- **Cada unidad de trabajo cierra con un commit Conventional** (sin atribución de IA) con sus pruebas, deltas de goldens y doctrina regenerada. La heurística de unas 400 líneas por tarea es orientativa, no un límite.
- **Orden y compilación:** cada commit deja el árbol compilando y con sus pruebas en verde. Ningún commit deja una doctrina que promete algo que aún no existe (por eso WU-4 va antes que WU-3).
- Este cambio no toca las fixtures que pasan `--handoff-policy` ni las de `e2e/`, `bench/` o `cmd/axiom/` que compilan binarios.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 4.000 a 5.500 (central ≈ 4.800), autoradas (adiciones + eliminaciones, pruebas incluidas); sin contar goldens ni `.codex/AGENTS.md` regenerados. La estimación de `proposal.md` §9 (1.500 a 2.500) no incluía las pruebas ni los textos de doctrina |
| 400-line budget risk | High (aceptado con `size:exception`) |
| Chained PRs recommended | No (una sola PR por preferencia fija del usuario) |
| Suggested split | PR única con 4 commits de unidad de trabajo, revisables commit a commit, en el orden WU-1, WU-2, WU-4, WU-3 |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

La PR lleva la etiqueta `size:exception`. La carga de revisión se reparte por commit: WU-2 es la mayor en código nuevo (≈2.000 l, casi todo paquete nuevo y sus pruebas) y WU-3 la mayor en textos de doctrina y contratos.

### Suggested Work Units

| Unit | Orden de ejecución | Goal | Tareas | Líneas | Focused test command | Rollback boundary |
|------|--------------------|------|--------|--------|----------------------|-------------------|
| WU-1 | 1.º | Bloque `workflow`, proyección en `status`, paso 0 y ODD sin paradas | T-01 a T-07 (7) | ≈ 800 | `go test ./internal/workspace/ ./internal/sddstatus/ ./internal/agents/capabilitymanifest/ ./internal/components/agentguidance/ -count=1` | `internal/workspace/workflow*.go`, `workflow_status*`, `routing.go`, `stop-exits.md`, `axiom.yaml` |
| WU-2 | 2.º | Verbo `axiom sdd publish`, `docpaths`, `PublicationTarget` y puerto aislado | T-08 a T-15 (8) | ≈ 2.200 | `go test ./internal/docpaths/ ./internal/sddpublish/ ./internal/gitcommon/ ./internal/branch/... -count=1` y `go test ./internal/cli/ -run 'SDDPublish\|PublishIsolation' -count=1` | `internal/docpaths`, `internal/sddpublish`, `internal/cli/sdd_publish*.go`, `publication.go` |
| WU-4 | 3.º | Adopción de la rama publicada en `axiom branch ensure` | T-16 a T-19 (4) | ≈ 450 | `go test ./internal/branch/... -count=1` | `engine/ensure.go`, `engine/engine.go`, `gitcli/ops.go`, `result.go` |
| WU-3 | 4.º | Aprobar publica siempre: kickoff, sobre, `gate record`, doctrina de puertas | T-20 a T-28 (9) | ≈ 1.500 (+ goldens) | `go test ./internal/kickoff/ ./internal/sddstatus/ ./internal/branch/... ./internal/assets/ ./internal/components/... -count=1` y `go test ./internal/cli/ -run 'Gate\|Kickoff\|HandOff' -count=1` | `kickoff/types.go`, `governance*.go`, `handoff*.go`, `sdd_gate*.go`, doctrina y skills |

Total: **28 tareas** (WU-1: 7 · WU-2: 8 · WU-4: 4 · WU-3: 9). Dependencias: WU-1 y WU-2 son independientes entre sí; WU-4 es independiente de ambos; WU-3 depende de WU-2 (`PublicationTarget`, `publish`) y de WU-4 (la doctrina promete la adopción). No hay paralelismo autorizado (un solo escritor); el orden indicado es el de ejecución.

---

## WU-1 · Bloque `workflow`, proyección en `status`, paso 0 y ODD sin paradas (incremento 1 de D8)

Commit: `feat(workflow): declarar el carril del proyecto en axiom.yaml y retirar la parada de ODD`

- [x] T-01 Decodificador y tipos del bloque `workflow` (≈220 l)
  - Ficheros: `internal/workspace/workflow.go` (nuevo), `internal/workspace/workflow_test.go` (nuevo).
  - Cambio (DD-01): `WorkflowLane` y constantes `odd|sdd|ask`, `DefaultWorkflowLane = sdd`, `WorkflowConfig`, `workflowKeys`, `decodeWorkflow` sobre `decodeStringKeys(node, "workflow", workflowKeys, workflowValidator)` con los mensajes de design §2.1 y `resolveWorkflow`.
  - Cubre: REQ-27.1 · DD-01.
  - Aceptación: tablas con `odd`, `sdd`, `ask` válidos; `SDD` (mayúsculas), `agil`, `null`, lista, `workflow:` vacío, clave duplicada y clave desconocida rechazados con un error que nombra `workflow.lane`/`workflow`; `resolveWorkflow(nil)` y bloque sin `lane` resultan `sdd`; un decodificador congelado sin el bloque carga el mismo `axiom.yaml` ignorando la clave (compatibilidad REQ-24.17).
  - Comprobación: `go test ./internal/workspace/ -run 'Workflow' -count=1`

- [x] T-02 Integración en declaraciones, carga y `init`; plantilla y `axiom.yaml` de este repositorio (≈160 l)
  - Ficheros: `internal/workspace/declarations.go`, `internal/workspace/loader.go`, `internal/workspace/types.go`, `internal/hub/init.go` (`ownMarkers.declares`), `internal/hub/init_test.go` (o el fichero de pruebas existente del paquete), `axiom.example.yaml`, `axiom.yaml`, `internal/workspace/workflow_test.go`.
  - Cambio: `Declarations.Workflow`/`WorkflowErr`; lectura en `readDeclarationBlocks`; `joinLoadErrors` añade `WorkflowErr`; `WorkspaceConfig.Workflow` (`yaml:"-"`) proyectado en `parseLayered` con `resolveWorkflow`; `declares()` cuenta `workflow`; plantilla con el bloque comentado; **este repositorio declara `workflow: {lane: ask}` (S8)**. `branch.ConfigRefusal` y `blockArchiveOnDeclarations` no leen `WorkflowErr` (design §2.1).
  - Cubre: REQ-27.1, REQ-2.1 (estructura del esquema), S8 · DD-01.
  - Aceptación: `axiom workspace validate`/`LoadConfig` informan un `lane` inválido con la clave; `WorkspaceConfig.Workflow.Lane` siempre relleno; un `lane` inválido no impide planificar ramas; `init --force` no sobrescribe un bloque `workflow` declarado; el `axiom.yaml` del repositorio carga con `lane: ask`; la plantilla carga sin error.
  - Comprobación: `go test ./internal/workspace/ ./internal/hub/ -count=1`

- [x] T-03 Proyección `workflow` en `axiom sdd status` v1 y v2 y contrato (≈230 l)
  - Ficheros: `internal/sddstatus/status.go`, `internal/sddstatus/status_v2.go`, `internal/sddstatus/workspace_declarations.go`, `internal/sddstatus/workflow_status_test.go` (nuevo), `internal/assets/skills/_shared/sdd-status-contract.md`.
  - Cambio (DD-02): `WorkflowProjection{Lane, Source}`, `Status.Workflow` y `statusV2.Workflow` (`omitempty`, tras `TechnicalContext`), `projectWorkflow(location)` con las cuatro filas de design §2.2 en ese orden, rellenado siempre en `decorateWorkspaceDeclarations` (con y sin cambio). El contrato documenta el campo, sus valores y la regla «`no_workspace` e `invalid` proyectan `ask`; si el status no responde, el orquestador actúa como `ask`».
  - Cubre: REQ-27.2, REQ-27.2b (parte de contrato) · DD-02.
  - Aceptación: v1 y v2, con y sin cambio seleccionado, proyectan `lane`/`source`; sin `axiom.yaml` → `ask`/`no_workspace`; `ConfigErr`, `ParseErr` o `WorkflowErr` → `ask`/`invalid` (incluido un bloque `workflow` válido con base rota); bloque declarado → `declared`; válido sin bloque o sin `lane` → `sdd`/`default`; ninguna nota ni bloqueo nuevos; la ruta recomendada no cambia (comparación contra un status existente).
  - Comprobación: `go test ./internal/sddstatus/ -count=1` y `go test ./internal/assets/ -run 'StatusContract' -count=1`

- [x] T-04 Política de selección del manifiesto (≈90 l)
  - Ficheros: `internal/agents/capabilitymanifest/manifest.go`, `internal/agents/capabilitymanifest/manifest_test.go`, y los nombres o pines que citen el literal en `internal/cli/run_component_paths_test.go` y `internal/cli/sync_test.go`.
  - Cambio (DD-03): `SDDSelectionExplicitRequestProjectLaneOrAcceptedProposal = "explicit_request_project_lane_or_accepted_proposal"`; `ForAgent` lo usa en todos los adaptadores; se retira la constante anterior.
  - Cubre: REQ-27.16 (política), REQ-27.3 · DD-03.
  - Aceptación: todos los adaptadores devuelven el valor nuevo; ningún texto fuera de pruebas conserva el literal retirado (`rg explicit_request_or_accepted_proposal` sin resultados en producción).
  - Comprobación: `go test ./internal/agents/capabilitymanifest/ -count=1` y `go test ./internal/cli/ -run 'RunComponentPaths|SyncRouting|Manifest' -count=1`

- [x] T-05 Paso 0 con carril y retirada de la parada ODD en `routing.go` (≈260 l)
  - Ficheros: `internal/components/agentguidance/routing.go`, `routing_test.go`, `routing_stops_test.go`.
  - Cambio (DD-04, puntos 1 a 7): frase introductoria, paso 0 con el texto de design §2.4, paso 5 sin «Then present the plan stop…», párrafo de reanudación sin «handed off»/«approved plan», viñeta «Optional SDD», retirada de la viñeta `Plan stop:` y de `### Stops and exits` (incluida la llamada a `StopExits()`), comentarios Go con REQ-27.16 y REQ-27.17. `RenderRouting` compara contra la constante nueva y devuelve `ErrUnknownRoutingPolicy` ante cualquier otra, incluido el literal retirado.
  - Cubre: REQ-27.16, REQ-27.17, REQ-27.3, REQ-21.1 a REQ-21.3, REQ-25.40 (escenario) · DD-03, DD-04.
  - Aceptación: pines del paso 0 (`workflow`, `lane`, `source`, `no_workspace`, `invalid`, «cannot be read», las tres ramas `sdd`/`odd`/`ask`, la pregunta bloqueante conservada); `TestRenderRoutingKeepsSDDSelectionExplicit` conserva sus tres pines y añade `project lane` y `workflow`; `routing_stops_test.go` pasa a prueba negativa (ningún texto contiene `Plan stop`, `plan stop`, `### Stops and exits`, `three exits`, `handed off`); política desconocida y literal retirado → `ErrUnknownRoutingPolicy`; salida idéntica en todos los adaptadores.
  - Comprobación: `go test ./internal/components/agentguidance/ -count=1`

- [x] T-06 `stop-exits.md` solo SDD y sección «SDD lane selected» (≈120 l)
  - Ficheros: `internal/assets/generic/stop-exits.md`, `internal/components/agentguidance/stop_exits.go` (comentario), `internal/components/agentguidance/stop_exits_test.go`, `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `internal/components/sdd/orchestrator_stop_exits_test.go`, `internal/assets/assets_test.go` (solo los pines que citen la parada ODD).
  - Cambio (DD-04, última parte): se retiran del texto la mención de ODD y la «single ODD plan stop» (la reescritura con «publish» llega en T-26); `sdd-orchestrator-sections.md` refleja «SDD lane selected» (la pregunta de carril solo con `ask` o sin `axiom.yaml`).
  - Cubre: REQ-27.17, REQ-27.3, REQ-21.1, REQ-25.35 · DD-04.
  - Aceptación: `TestStopExitsNamesNoGateInternals` pasa a prohibir `ODD`, `plan stop` y `odd/tasks` y sigue prohibiendo `gateQuestion`, `gateCriteria` y `gateEvidence`; `agentguidance.StopExits()` sigue disponible para el compositor de orquestadores SDD; los doce orquestadores y Pi compuestos siguen incluyendo `StopExits()`.
  - Comprobación: `go test ./internal/components/agentguidance/ ./internal/components/sdd/ ./internal/assets/ -count=1`

- [x] T-07 Documentación, regeneración de goldens y commit de WU-1 (≈20 l de autoría; resto generado)
  - Ficheros: `AGENTS.md`, `GEMINI.md` (sin la parada de plan), `docs/manual-de-inicio.md` (sección del carril: `workflow.lane`, `ask` para conservar la pregunta, nota de cambio de comportamiento por defecto a `sdd`), `internal/workspace/docs_contract_test.go`, `testdata/golden/*.golden` (13), `.codex/AGENTS.md`.
  - Cambio: regenerar con `go test ./internal/components/ -run Golden -update -count=1` y luego sin `-update`; copiar el bloque gestionado de `sdd-codex-agentsmd.golden` a `.codex/AGENTS.md`; pin del manual y de la plantilla en `docs_contract_test.go`. Cierra con el commit de WU-1.
  - Cubre: REQ-27.1 (documentación), REQ-27.17 (paridad de doctrina) · DD-16.
  - Aceptación: goldens en verde sin `-update`; `.codex/AGENTS.md` coincide con el golden; ningún golden contiene `Plan stop`; el manual describe el carril y la migración (proyectos sin la clave pasan a `sdd`).
  - Comprobación: `go test ./internal/components/ -run Golden -count=1`, `go test ./internal/workspace/ -count=1` y `go vet ./internal/workspace/ ./internal/sddstatus/ ./internal/components/... ./internal/hub/`

---

## WU-2 · Verbo `axiom sdd publish`, `docpaths`, `PublicationTarget` y puerto aislado (incremento 2 de D8)

Commit: `feat(sdd-publish): publicar con axiom sdd publish solo la documentación del cambio`

- [x] T-08 Paquete puro `internal/docpaths` (≈220 l)
  - Ficheros: `internal/docpaths/docpaths.go` (nuevo), `internal/docpaths/docpaths_test.go` (nuevo).
  - Cambio (DD-06): `Set`, `New(planningRel, change, baseline, maintained, reference)`, `Contains`, `Root`, `LocalState` con la tabla de raíces de design §2.6 (carpeta del cambio, carpeta archivada con nombre escapado, `<p>/specs`, `<p>/INDEX.md`, documentos de baseline, rutas mantenidas; `reference` y `odd/tasks/**` nunca).
  - Cubre: REQ-27.5 (lista cerrada), REQ-27.6 · DD-06 · matriz «Rutas que parecen documentación».
  - Aceptación: `TestDocPathsClassifiesByRootNotExtension` (`requirements.txt`, `CMakeLists.txt`, `docs/x.mdx` sin declarar y `README.sh` en la raíz son código; `README.sh` bajo `openspec/changes/<x>/` es documentación); `foo-x` no coincide con la carpeta archivada de `x`; `reference` gana sobre otra regla; `odd/tasks/**` es código; `LocalState` marca `.gates.lock`, `.axiom-instance` y temporales de la raíz del cambio; `planningRel` vacío (multirepo); comparaciones sensibles a mayúsculas.
  - Comprobación: `go test ./internal/docpaths/ -count=1`

- [x] T-09 `Engine.PublicationTarget` y códigos nuevos de `branch` (≈420 l)
  - Ficheros: `internal/branch/publication.go` (nuevo), `internal/branch/engine/publication.go` (nuevo), `internal/branch/engine/publication_test.go` (nuevo), `internal/branch/types.go`, `internal/branch/labels.go`.
  - Cambio (DD-07): `PublicationRequest`, `PublicationTarget`, `Engine.PublicationTarget` sobre `gather` + `Plan` (y `useArchivePlan` con `Archive`), solo lecturas (sin cerrojo, registro ni `fetch`); códigos `detached_head` e `integration_branch` con su etiqueta; reglas de rama 1 a 4 de design §2.7, con `wrong_branch` y la continuación de `HandOffWrongBranch` (que nombra `--archive` cuando falta).
  - Cubre: REQ-27.7, REQ-27.14 (rama de archivado), REQ-25.19 · DD-07 · matriz «Estado del push».
  - Aceptación: `monorepo-embedded` con rama de cambio, de rol y de archivado; multirepo con repositorio de specs in situ en `main` → rama permitida = integración; rama de integración de un repositorio de código → `integration_branch`; HEAD desacoplado → `detached_head`; rama de archivado sin `--archive` y rama de cambio con `--archive` → `wrong_branch`; `Branch` distinto de la rama permitida → `wrong_branch`; `PRTarget` igual al de `axiom branch status` y vacío en integración in situ; el observador de `gitcommon` no registra `push`, `commit` ni `fetch`.
  - Comprobación: `go test ./internal/branch/... -count=1`

- [x] T-10 `gitcommon.RunWithTimeout` (≈90 l)
  - Ficheros: `internal/gitcommon/exec.go`, `internal/gitcommon/exec_test.go`.
  - Cambio (DD-08): `RunWithTimeout(ctx, dir, timeout, args...)`; `Run` delega con `runTimeout`; `Observe` sigue viendo todas las invocaciones.
  - Cubre: REQ-27.9 (tiempo de espera → `remote_unavailable`), REQ-27.11 (único punto de ejecución) · DD-08.
  - Aceptación: un comando que supera el límite devuelve error de tiempo de espera distinguible; `Run` conserva su comportamiento; el observador registra las invocaciones de ambas funciones.
  - Comprobación: `go test ./internal/gitcommon/ -run 'Exec|Observe|Timeout' -count=1`

- [x] T-11 Puerto `sddpublish.Git` y adaptador sobre `gitcommon` (≈380 l)
  - Ficheros: `internal/sddpublish/ports.go` (nuevo), `internal/sddpublish/gitport.go` (nuevo), `internal/sddpublish/gitport_test.go` (nuevo).
  - Cambio (DD-08): interfaz `Git` con las nueve operaciones de design §2.8 y `PushOutcome`; adaptador con `gitcommon.Run` (y `RunWithTimeout` con `pushTimeout = 120 * time.Second`, variable de paquete); referencias validadas con `gitref.ValidRefName`, rutas con `:(top,literal)`, `--end-of-options` en `rev-list` y `rev-parse`; `Commit` en dos pasos (`git add --all -- <ps>` y `git commit --only -m <msg> -- <ps>`).
  - Cubre: REQ-27.6, REQ-27.9, REQ-27.11 · DD-08 · matriz «Selección del repositorio git» y «Estado del commit».
  - Aceptación: sobre un repositorio y un remoto bare temporales, cada operación devuelve lo esperado; ningún argumento empieza por `-`; `DirtyPaths` incluye borrados y no sigue renombrados; `UnpublishedCommits` respeta el límite; `CommitPaths` con `--cc` deja solo lo resuelto por una fusión; el adaptador ignora `GIT_DIR` y variables que redirigen el repositorio (`TestPublishIgnoresGitDirEnvironment`).
  - Comprobación: `go test ./internal/sddpublish/ -run 'GitPort|Adapter' -count=1`

- [x] T-12 Resultado, JSON `axiom.sdd-publish/v1` y clasificación de negativas (≈230 l)
  - Ficheros: `internal/sddpublish/result.go` (nuevo), `internal/sddpublish/classify.go` (nuevo), `internal/sddpublish/classify_test.go` (nuevo).
  - Cambio (DD-09, DD-10): tipos `Request`, `Result`, `Refusal{code,label,message,continuation,details}`, los 17 códigos de design §5.3 con su etiqueta en castellano; `classifyPush` y `classifyCommit` con la tabla de design §2.9 (paso 10).
  - Cubre: REQ-27.9, REQ-27.8 (forma de la negativa) · DD-09.
  - Aceptación: tabla con entradas `--porcelain` y `stderr` fijas (`LC_ALL=C`) para `push_non_fast_forward` (`non-fast-forward` y `fetch first`), `push_hook_rejected` (`remote rejected`, hook `pre-push` local), `remote_unavailable` (las cinco señales y el tiempo de espera, con `details.timeout`), `push_failed`; `commit_hook_rejected` frente a `commit_failed`; serialización JSON estable con `commit` y `push` omitidos cuando no ocurrieron.
  - Comprobación: `go test ./internal/sddpublish/ -run 'Classify|Result' -count=1`

- [x] T-13 Algoritmo `Publisher.Run` y matriz de pruebas con remoto bare (≈650 l)
  - Ficheros: `internal/sddpublish/publish.go` (nuevo), `internal/sddpublish/publish_test.go` (nuevo).
  - Cambio (DD-09): los diez pasos en orden (almacén → cambio → destino → remoto → rango sin subir → conjunto de documentación → commit `--only` por raíces con exclusión del estado local → commits sin subir → `up_to_date` → push `refs/heads/<b>:refs/heads/<b>` sin `+`, `--force*` ni `--no-verify`, con `--set-upstream` solo fuera de `InPlace`); mensajes `docs(sdd): publicar los documentos de <x>` / `publicar el archivado de <x>`; límite de 500 commits.
  - Cubre: REQ-27.5, REQ-27.6, REQ-27.7, REQ-27.8, REQ-27.9, REQ-27.10, REQ-27.14 · DD-09 · matriz «Rutas que parecen documentación», «Estado del commit» y «Estado del push».
  - Aceptación (nombres de design §7): `TestPublishKeepsStagedCodeStaged`, `TestPublishNeverCommitsUnstagedCode`, `TestPublishNoDocChangesMakesNoCommit`, `TestPublishExcludesLocalStateFiles`, `TestPublishCommitsArchiveDeletions`, `TestPublishFirstPushCreatesRemoteBranch`, `TestPublishTrackedBranchFastForward`, `TestPublishNonFastForwardRefusedWithoutRetry`, `TestPublishRefusesIntegrationBranchInCodeRepo`, `TestPublishMultirepoSpecsPushesIntegration`, `TestPublishBranchFlagMismatchIsWrongBranch`, `TestPublishNeverRunsFetchOrForce` (observador), `TestPublishRefusesUnpushedCommitTouchingReadmeSh`, `TestPublishIgnoresPublishedIntegrationCommits`, `TestPublishRunsInTargetRootFromLinkedWorktree`; además `store_without_files` (`engram`, `none`), `change_not_found`, `remote_missing` sin commit previo, `unpublished_range_unknown`, `identity_missing`, hooks `pre-commit` y `pre-push` locales y `pre-receive` remoto, remoto con URL inexistente, HEAD desacoplado, `up_to_date` y `unpublished_range_too_large`; todo con `t.Setenv` para la identidad y hooks `#!/bin/sh\nexit 1`. En cada negativa posterior al commit, el commit local se conserva y se informa. Comprobar durante esta tarea el supuesto abierto de design §9 (`git rev-parse --git-path hooks/<hook>` y `core.hooksPath`); si no se respeta, degradar a `commit_failed`/`push_failed` con la cola de `stderr`.
  - Comprobación: `go test ./internal/sddpublish/ -count=1`

- [x] T-14 Verbo `axiom sdd publish` en la CLI y registro en `cmd/axiom` (≈320 l)
  - Ficheros: `internal/cli/sdd_publish.go` (nuevo), `internal/cli/sdd_publish_test.go` (nuevo), `cmd/axiom/main.go` (registro en `runSDD`, ayuda y lista de opciones).
  - Cambio (DD-10): `RunSDDPublish(args []string, stdout io.Writer) error` con `--cwd`, `--change` (obligatorio), `--archive`, `--branch`, `--json`; `--cwd` por `workspace.Locate`; `guardChangeWriter` con `ContextOrdinary`/`ContextArchive`; salida de texto en castellano; JSON también en una negativa; `errPublishRefused` para el código de salida 1; `PublicationTarget` y `sddpublish.Publisher` cableados con el adaptador real.
  - Cubre: REQ-27.5, REQ-27.2b, REQ-27.7, REQ-27.9 · DD-10 · matriz «Selección del repositorio git».
  - Aceptación: sin `--change`, vacío o inválido → error de uso sin efectos; `published` y `up_to_date` → salida 0 y `refused` → 1; `--json` escribe `axiom.sdd-publish/v1` también en la negativa; `TestPublishRelativeCwdResolvesToWorkspace`; con `--archive` ausente la rama de archivado es `wrong_branch` y la continuación nombra el flag; el texto incluye commit, push y «Quien siga: `git fetch origin` y `axiom branch ensure --change <x> --json`»; la ayuda de `cmd/axiom` lista `publish`.
  - Comprobación: `go test ./internal/cli/ -run 'SDDPublish' -count=1` y `go test ./cmd/axiom/ -run 'Help|Usage|SDDDispatch' -count=1` (solo pruebas en proceso; si el paquete `cmd/axiom` no tiene pruebas de ayuda acotables, se omite y se anota en el commit)

- [x] T-15 Pruebas de aislamiento de la operación remota (≈260 l)
  - Ficheros: `internal/gitcommon/architecture_test.go`, `internal/sddpublish/isolation_test.go` (nuevo), `internal/cli/sdd_publish_isolation_test.go` (nuevo).
  - Cambio (DD-14): `TestArchitectureOnlyExecGoLaunchesProcesses` recorre también `internal/sddpublish` y `internal/docpaths`; `isolation_test.go` recorre con `go/parser` los `.go` de producción de `internal/` y `cmd/`: solo `internal/cli` importa `sddpublish` y solo `sdd_publish.go` usa el selector `sddpublish.`; `internal/branch/**` y `internal/docpaths` no lo importan; ningún fichero de `internal/branch/**` pasa `"push"` ni `"commit"` a `gitcommon.Run`/`RunWithTimeout`/`runWrite`/`runRead` (`ast.CallExpr`); el observador dinámico ejecuta `RunBranch ensure|status|done` (con y sin `--archive`), `RunSDDGate record` (`approved`, `--hand-off`, `rejected`) y `RunSDDStatus` con la puerta abierta sobre un proyecto con remoto bare y comprueba que no hay `push`, `commit` ni `fetch` sin `--fetch`.
  - Cubre: REQ-27.11, REQ-25.1 (invariante acotado a `axiom branch`) · DD-14.
  - Aceptación: las tres pruebas en verde; `TestEnsureArchiveNeverPublishes` y la prueba de `status` sin `fetch` siguen pasando sin cambios; si el observador `RunSDDGate` aún pasa por el comportamiento previo de `--hand-off`, la prueba no exige nada sobre confirmaciones (eso cambia en WU-3).
  - Comprobación: `go test ./internal/gitcommon/ ./internal/sddpublish/ -count=1`, `go test ./internal/branch/engine/ -run 'EnsureArchiveNeverPublishes' -count=1` y `go test ./internal/cli/ -run 'PublishIsolation' -count=1`. Cierra WU-2 con `go vet ./internal/docpaths/ ./internal/sddpublish/ ./internal/gitcommon/ ./internal/branch/... ./internal/cli/` y el commit.

---

## WU-4 · Adopción de la rama publicada en `axiom branch ensure` (incremento 4 de D8; se ejecuta antes que WU-3)

Commit: `feat(branch): adoptar en ensure la rama publicada como rama de seguimiento remoto`

- [x] T-16 `CreateTrackingBranch` en `gitcli` y `gitWriter` (≈110 l)
  - Ficheros: `internal/branch/gitcli/ops.go`, `internal/branch/gitcli/ops_test.go`, `internal/branch/engine/engine.go` (`gitWriter`), y los dobles de `branch.Git` que ya existan en `internal/branch` y `internal/branch/engine`.
  - Cambio (DD-13): `CreateTrackingBranch(ctx, dir, name, remoteRef string) error` con `git branch --track <name> <remoteRef>`; exige `remoteRef == "refs/remotes/origin/" + name` y `plainArgument` en ambos; el comentario del puerto pasa a «Fetch sigue siendo la única operación remota; ningún método publica».
  - Cubre: REQ-27.15, REQ-25.1 · DD-13.
  - Aceptación: con una referencia remota local la rama se crea con *upstream* configurado; `remoteRef` que no coincide, nombre con guion inicial o argumento no plano → rechazo sin escribir; el observador no registra operaciones remotas.
  - Comprobación: `go test ./internal/branch/gitcli/ ./internal/branch/engine/ -run 'Tracking|Ops' -count=1`

- [x] T-17 Adopción en `checkRepository` y `checkInPlaceRepository` (≈180 l)
  - Ficheros: `internal/branch/engine/ensure.go`, `internal/branch/result.go`, `internal/branch/engine/ensure_adopt_test.go` (nuevo).
  - Cambio (DD-13): para ramas `change`, `role` y `archive` ausentes en local, resolver `refs/remotes/origin/<nombre>` con `ResolveCommit`: si existe, `step.createFrom` y `step.adopt = true`; si no, la regla actual. `BranchResult` gana `CreatedFrom` (`base|change_branch|remote`) y `CreatedFromRef` (`omitempty`, solo en `ensure` y solo para las ramas que creó esa ejecución). `BaseResult.Source` no cambia. `checkIntegrated` no cambia y se sigue ejecutando antes. Con `--fetch` autorizado el `fetch` va antes de la comprobación (adopta en la misma ejecución); sin `--fetch` no hay red.
  - Cubre: REQ-27.15, REQ-25.7, REQ-25.10, REQ-26.11 (una rama publicada no es una rama integrada) · DD-13.
  - Aceptación: sobre remoto bare y segundo clon, con `git fetch` previo: rama de cambio, de rol y de archivado (`<x>-archive`) y repositorio `in_place` adoptan con `createdFrom: remote` y `createdFromRef`; sin referencia remota, `createdFrom: base` y comportamiento actual; `ensure --archive` con la base que no contiene el cambio rechaza igual que hoy aunque exista la rama remota; `ensure` repetido es idempotente (sin `createdFrom` la segunda vez); no mueve nunca una rama local existente; con `--fetch` adopta en una sola ejecución; un observador confirma que sin `--fetch` no hay `fetch`, `pull` ni `ls-remote`.
  - Comprobación: `go test ./internal/branch/... -count=1`

- [x] T-18 Texto de `ensure` y aviso de rama creada desde la base (≈90 l)
  - Ficheros: `internal/cli/branch_ensure_text.go` (o el fichero que escribe el texto de `ensure`; verificar con `rg 'ensure' internal/cli`), su prueba, `internal/branch/engine/ensure_adopt_test.go`.
  - Cambio (DD-13): líneas «Rama <b> adoptada desde origin/<b> (seguimiento configurado)» y, al crear una rama de cambio desde la base sin `--fetch`, la línea informativa «Rama <b> creada desde la base: no había `origin/<b>` en local; si otra persona ya la publicó, borra esta rama local, ejecuta `git fetch origin` (o `--fetch` con autorización) y repite `ensure`». Sin cambios de comportamiento.
  - Cubre: REQ-27.15, REQ-25.10 · DD-13.
  - Aceptación: el texto sale solo en los casos descritos; el JSON de design §5.5 es el de `--json`; los refusals nuevos (si los hay) llevan continuación ejecutable (ratchet de rechazos).
  - Comprobación: `go test ./internal/cli/ -run 'BranchEnsure|Ensure' -count=1`

- [x] T-19 Cierre de WU-4: verificación y commit (≈70 l de ajustes si aparecen)
  - Ficheros: ninguno de partida; ajustes mínimos de las pruebas anteriores.
  - Cambio: comprobar que la regresión completa del motor de ramas sigue en verde y que `TestEnsureArchiveNeverPublishes` pasa sin cambios; commit de WU-4.
  - Cubre: REQ-27.15, REQ-27.11 · DD-13, DD-16.
  - Aceptación: sin binarios compilados; sin `fetch` implícito observado; el commit contiene solo adopción (sin doctrina que dependa de WU-3).
  - Comprobación: `go test ./internal/branch/... -count=1`, `go test ./internal/cli/ -run 'BranchEnsure|Ensure|PublishIsolation' -count=1` y `go vet ./internal/branch/... ./internal/cli/`

---

## WU-3 · Aprobar publica siempre: kickoff, sobre de la puerta, `gate record` y doctrina (incremento 3 de D8)

Commit: `feat(sdd-gates): aprobar una parada publica los documentos del cambio`

- [ ] T-20 `docs_publication` en el kickoff y flag `--docs-publication` (≈220 l)
  - Ficheros: `internal/kickoff/types.go`, `internal/kickoff/args.go` (edición puntual), `internal/kickoff/schema.go`, `internal/kickoff/types_test.go`, `internal/kickoff/args_test.go`, `internal/kickoff/schema_test.go`, `internal/cli/sdd_kickoff.go` (ayuda y texto), `internal/cli/sdd_kickoff_test.go`.
  - Cambio (DD-05): `DocsPublication` y `DocsPublicationAuto`, `FlowConfig.DocsPublication` (`docs_publication,omitempty`), `SealArgs.DocsPublication`; `ParseSealArgs` rechaza cualquier valor distinto de `auto`; tras resolver el estilo efectivo (también con `--from-session-pace`) rechaza si no es `continuous`, y rechaza siempre con `--infer`; `Kickoff.Validate` admite `""` y `auto` y rechaza `auto` con `checkpointed`; el sellado único no reescribe el campo.
  - Cubre: REQ-27.4 · DD-05.
  - Aceptación: `seal --execution-style continuous --docs-publication auto` escribe el campo; con `checkpointed`, `--infer` o `otro` se rechaza sin escribir `kickoff.yaml`; sellar sin el flag no escribe el campo (serialización idéntica a la anterior); un segundo `seal` devuelve «Kickoff ya estaba sellado» y no lo reescribe; un `kickoff.yaml` con el campo se lee con un struct congelado anterior sin error.
  - Comprobación: `go test ./internal/kickoff/ -count=1` y `go test ./internal/cli/ -run 'Kickoff' -count=1`

- [ ] T-21 Proyección `governance.kickoff.docsPublication` (≈70 l)
  - Ficheros: `internal/sddstatus/status_v2.go` (`kickoffV2.DocsPublication`, `json:"docsPublication,omitempty"`), `internal/sddstatus/status_v2_test.go`, `internal/assets/skills/_shared/sdd-status-contract.md`.
  - Cambio (DD-05): en v1 el campo aparece por reflexión; en v2 se copia; el contrato lo documenta en `governance.kickoff`.
  - Cubre: REQ-27.4, REQ-27.13 · DD-05.
  - Aceptación: con el campo sellado aparece `docsPublication: "auto"` en v1 y v2; sin él, ausente; un kickoff antiguo no altera ninguna proyección.
  - Comprobación: `go test ./internal/sddstatus/ -count=1`

- [ ] T-22 `HandOffPlan` nuevo y retirada de `uncommitted_artifacts` (≈380 l, casi todo reescritura y eliminaciones)
  - Ficheros: `internal/branch/handoff.go`, `internal/branch/handoff_test.go`, `internal/branch/types.go`, `internal/branch/labels.go`, `internal/branch/engine/handoff.go`, `internal/branch/engine/handoff_test.go`.
  - Cambio (DD-12): `Engine.HandOff` deja de comprobar «confirmado» en todas las puertas y de leer `DirtyPaths`; se retiran `UncommittedArtifacts`, `CodeUncommittedArtifacts` y su etiqueta, `uncommittedArtifacts`, `exemptEntry`, `HandOffRequest.Exempt`, `MergeRequired`, `MergeNotice`, `InPlaceNotice`, `HandOffAddStep`, `HandOffCommitStep` y `HandOffCommitMessage`; `HandOffPlan{Topology, Publish HandOffPublish, Deliveries, Resume}` con `Publish.Invocation`, `PRTarget` e `InPlace`; `HandOffDelivery` pierde `PR` y conserva `PRTarget`; `Resume = ["git fetch origin", "axiom branch ensure --change <x> --json"]`; se conservan `HandOffPushStep` y `HandOffWrongBranch`. Verificar con `rg` que ningún otro consumidor usa lo retirado.
  - Cubre: REQ-26.4, REQ-26.5, REQ-26.11, REQ-27.12 · DD-12.
  - Aceptación: monorepo con rama de cambio → `Publish.PRTarget` = integración; con rama de rol → `prTarget` de la rama de seguimiento; repositorio de specs de un multirepo → `InPlace` sin `prTarget`; `role-apply` con código sin confirmar ya no rechaza (lo informa después `publish`); `Deliveries` solo lista repositorios de código del rol con su `prTarget`; ningún paso contiene `&&`, `gh ` ni `glab `; el observador de `gitcommon` no registra `push`, `fetch`, `pull`, `ls-remote` ni `commit`.
  - Comprobación: `go test ./internal/branch/... -count=1`

- [ ] T-23 Sobre de la puerta con `publication` y etiquetas «publish» (≈300 l)
  - Ficheros: `internal/sddstatus/governance.go`, `internal/sddstatus/governance_handoff.go`, `internal/sddstatus/governance_test.go`, `internal/sddstatus/status_v2_test.go`.
  - Cambio (DD-11): `GatePublication{Remote, Branch, PRTarget, Invocation}` y `SDDGovernanceGateResult.Publication`/`PublicationUnavailable`; `newGovernanceGateQuestion` consulta `engine.Default().PublicationTarget` y, con motor nulo, omite `publication`; etiquetas «Approve, publish and stop» y «Approve, publish and continue» con los efectos de design §2.11; con `engram` o `none`, `approved` mantiene «Approve and continue» y `hand_off` no existe; `Validate` exige `Invocation` con prefijo `axiom sdd publish ` y `--branch <Branch>`; `gateQuestionNarrative` nombra el remoto y la rama; `integration` sin cambios.
  - Cubre: REQ-27.12, REQ-27.7 (rama atada), REQ-26.1, REQ-26.6 · DD-11.
  - Aceptación: en `spec`, `design`, `tasks` y `role-apply:<rol>` con almacén de ficheros, el sobre lleva `publication` con la rama real y la invocación con `--branch`; con `wrong_branch` lleva `publicationUnavailable: "wrong_branch"` y las opciones de `gate record`; efectos con la frase «A refused publication keeps the approval. Code pushes, pull requests and merges remain yours.»; invocaciones de las opciones siguen con prefijo `axiom sdd gate record `; el contrato de *Lossless Blocking Prompt* se cumple con las etiquetas nuevas.
  - Comprobación: `go test ./internal/sddstatus/ -count=1` y `go test ./internal/assets/ -run 'BlockingPrompt' -count=1`

- [ ] T-24 `gate record`: `publish` en el JSON, entrega nueva y `--hand-off` sin comprobación de confirmado (≈330 l)
  - Ficheros: `internal/cli/sdd_gate.go` (ediciones mínimas), `internal/cli/sdd_gate_handoff.go`, `internal/cli/sdd_gate_test.go`, `internal/cli/sdd_gate_handoff_test.go`, `internal/cli/sdd_writer_guard_test.go`.
  - Cambio (DD-11, DD-12): `gateRecordJSON` gana `Publish *gatePublishJSON` y `PublishUnavailable` en toda aprobación distinta de `integration` (se calcula la orden, no se ejecuta); `writeHandOffDelivery` con el texto de design §2.12 (incluida la variante «se trabaja in situ en la integración, sin PR» y las órdenes de push de código numeradas con «Axiom no sube código ni abre PR: ejecútalo tú»); se retira `delivery.mergeRequired` del JSON `axiom.sdd-gate-record/v1`; se elimina el rechazo `uncommitted_artifacts` y se conservan guarda `wrong_branch`, estilo, almacén, digest y puerta ya aprobada.
  - Cubre: REQ-26.3, REQ-26.4, REQ-26.5, REQ-26.6, REQ-27.12 · DD-11, DD-12 · matriz «Estado del push».
  - Aceptación: salidas 2 y 3 devuelven `publish.invocation` (con `prTarget`, vacío en specs in situ de multirepo); `--hand-off` con artefactos sin confirmar y con `role-apply` con código sucio **no** rechaza; las salidas 1 y 3 siguen escribiendo registros sin `handed_off`; `gate record` no ejecuta `commit` ni `push` (observador); `gates.yaml` queda intacto en cada rechazo restante; con `integration` sin cambios; salida de texto sin PR fusionada.
  - Comprobación: `go test ./internal/cli/ -run 'Gate|HandOff|WriterGuard' -count=1`

- [ ] T-25 Pruebas de integración de `publish` encadenado tras `gate record` (≈220 l)
  - Ficheros: `internal/cli/sdd_publish_gate_test.go` (nuevo), `internal/cli/sdd_publish_isolation_test.go`.
  - Cambio: pruebas en proceso sobre remoto bare: `RunSDDGate record --decision approved --hand-off --json` seguido de `RunSDDPublish` con la `invocation` devuelta publica `gates.yaml` y los documentos y no sube código; un `role-apply` con commit de código sin subir devuelve `code_commits_unpublished` y la aprobación registrada se conserva; el observador de aislamiento se actualiza al comportamiento nuevo de `--hand-off`.
  - Cubre: REQ-27.8, REQ-27.12, REQ-27.15 (quien sigue adopta la rama con `ensure`), REQ-27.11 · DD-11, DD-14.
  - Aceptación: de extremo a extremo en proceso, una segunda copia del repositorio hace `git fetch origin` y `RunBranch ensure` adopta la rama con `createdFrom: remote`; la negativa por código no deshace la aprobación; ningún comando de `axiom branch` ni `gate record` ejecuta `push` o `commit`.
  - Comprobación: `go test ./internal/cli/ -run 'SDDPublishGate|PublishIsolation' -count=1`

- [ ] T-26 Doctrina de puertas, `stop-exits.md` final, skills y pines de assets (≈330 l)
  - Ficheros: `internal/assets/generic/stop-exits.md` (texto final de design §2.11, solo SDD), `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (encadenado `gate record` + `publish`, `--docs-publication auto` en la opción `continuous` de la pregunta de estilo, momentos de publicación de DD-15 y archivado con `publish --archive`; se elimina «it refuses without writing anything when the artifacts are not committed»), `internal/assets/skills/sdd-spec/SKILL.md`, `internal/assets/skills/sdd-design/SKILL.md` (regla «La fase no hace commit ni push…»), `internal/assets/skills/_shared/sdd-status-contract.md` (`publication`, `publicationUnavailable`), `internal/assets/assets_test.go`, `internal/components/sdd/orchestrator_stop_exits_test.go`, `internal/components/agentguidance/stop_exits_test.go`.
  - Cambio (DD-11, DD-15): el texto de las tres salidas pasa a «Approve, publish and stop / continue»; la sección de puertas pide ejecutar `publish.invocation` una vez tras el registro y tratar la negativa como informativa; `continuous` con `docs_publication: auto` publica al cerrar `tasks` y tras archivar (`--archive` en el worktree de `ensure --archive`), sin él ofrece las órdenes; las skills de fase no publican.
  - Cubre: REQ-27.12, REQ-27.13, REQ-27.14, REQ-27.4, REQ-26.8, REQ-26.1, REQ-21.7 · DD-11, DD-15.
  - Aceptación: los doce orquestadores compuestos y Pi incluyen `StopExits()` idéntico; `{{AXIOM_STOP_EXITS}}` resuelto; pines de `publish.invocation`, de «A refused publication never undoes the recorded approval», de `--docs-publication auto` y de `publish --archive`; ningún asset conserva «commit` y `push` impresos para el humano» ni «PR fusionada»; `StopExits()` sigue sin nombrar `gateQuestion`, `gateCriteria` ni `gateEvidence`; el pin de `sdd-spec`/`sdd-design` coincide con design §2.15.
  - Comprobación: `go test ./internal/assets/ ./internal/components/sdd/ ./internal/components/agentguidance/ -count=1`

- [ ] T-27 Documentación, regeneración de goldens y `.codex/AGENTS.md` (≈90 l de autoría; resto generado)
  - Ficheros: `docs/manual-de-inicio.md` (publicación con `axiom sdd publish`, adopción con `fetch` + `ensure`, retirada de «PR fusionada», `docs_publication`, nota de que el CI de otros proyectos puede ejecutarse con el push de rama y queda fuera del control de Axiom), `AGENTS.md`, `GEMINI.md` (solo si el texto cambia), `internal/workspace/docs_contract_test.go`, `testdata/golden/*.golden` (13), `.codex/AGENTS.md`.
  - Cambio: regenerar con `go test ./internal/components/ -run Golden -update -count=1` y volver a pasar sin `-update`; copiar el bloque gestionado de `sdd-codex-agentsmd.golden` a `.codex/AGENTS.md`.
  - Cubre: REQ-27.12, REQ-27.15, REQ-26.8 (paridad: texto de las tres salidas idéntico en todos los runtimes) · DD-16.
  - Aceptación: goldens en verde sin `-update`; `.codex/AGENTS.md` coincide con el golden; el texto de las tres salidas es idéntico al de `StopExits()`; el manual no contradice la doctrina generada; pines de `docs_contract_test.go` en verde.
  - Comprobación: `go test ./internal/components/ -run Golden -count=1` y `go test ./internal/workspace/ -count=1`

- [ ] T-28 Verificación final del cambio y commit de WU-3 (≈60 l de ajustes si aparecen)
  - Ficheros: ninguno de partida; ajustes mínimos que surjan de las pruebas.
  - Cambio: `go vet ./...`; `go test` paquete a paquete de todos los tocados; ratchets de rechazos de `cli` con `-run` acotado (`'Refusal|Ratchet|Continuation'`); comprobar `rg` de importaciones: solo `internal/cli/sdd_publish.go` usa `sddpublish`; `gates.yaml` y `kickoff.yaml` de este cambio sin tocar. `scripts/deadcode-ratchet.sh` no se ejecuta si compila un binario suelto (se anota igual que en `paradas-tres-salidas`).
  - Cubre: criterios de éxito de `proposal.md` §12 («CI verde», «Aislamiento», «Paridad y goldens») · DD-14, DD-16.
  - Aceptación: todo en verde; ningún binario compilado ni ejecutado; si `internal/components/filemerge` falla por las uniones de directorio de Windows, se anota como fallo ambiental previo (paquete no tocado).
  - Comprobación: `go vet ./...` y `go test ./internal/workspace/ ./internal/hub/ ./internal/sddstatus/ ./internal/agents/capabilitymanifest/ ./internal/components/agentguidance/ ./internal/components/sdd/ ./internal/assets/ ./internal/kickoff/ ./internal/docpaths/ ./internal/sddpublish/ ./internal/gitcommon/ ./internal/branch/... -count=1`, más `go test ./internal/components/ -run Golden -count=1` y `go test ./internal/cli/ -run 'SDDPublish|PublishIsolation|SDDPublishGate|Gate|HandOff|Kickoff|BranchEnsure|Refusal|Ratchet|Continuation' -count=1`

---

## Trazabilidad requisito → tarea

| Requisito | Tareas |
|---|---|
| REQ-27.1 (bloque `workflow`) | T-01, T-02, T-07 |
| REQ-27.2, REQ-27.2b (proyección y contrato del binario) | T-03, T-14 |
| REQ-27.3 (carril gobierna la pregunta) | T-04, T-05, T-06 |
| REQ-27.4 (`docs_publication` en el kickoff) | T-20, T-21, T-26 |
| REQ-27.5 (verbo y lista cerrada) | T-08, T-13, T-14 |
| REQ-27.6 (commit solo de documentación) | T-08, T-11, T-13 |
| REQ-27.7 (rama y remoto) | T-09, T-13, T-14, T-23 |
| REQ-27.8 (código sin subir) | T-12, T-13, T-25 |
| REQ-27.9 (negativas tipadas) | T-10, T-11, T-12, T-13, T-14 |
| REQ-27.10 (almacén) | T-13, T-23 |
| REQ-27.11 (aislamiento) | T-10, T-11, T-15, T-16, T-19, T-25 |
| REQ-27.12 (aprobar publica) | T-22, T-23, T-24, T-25, T-26, T-27 |
| REQ-27.13 (`continuous`) | T-21, T-26 |
| REQ-27.14 (archivado) | T-09, T-13, T-26 |
| REQ-27.15 (adopción) | T-16, T-17, T-18, T-19, T-25, T-27 |
| REQ-27.16 (paso 0) | T-04, T-05 |
| REQ-27.17 (ODD sin paradas) | T-05, T-06, T-07 |
| REQ-21.1 a REQ-21.3, REQ-25.35, REQ-25.40 (enmiendas de `sdd-lane-kickoff` y `organic-agent-trigger-rules`) | T-05, T-06 |
| REQ-26.1, REQ-26.3 a REQ-26.6, REQ-26.8 (enmiendas de `sdd-block-review-gates`) | T-22, T-23, T-24, T-26, T-27 |
| REQ-25.1, REQ-25.7, REQ-25.10, REQ-26.11 (enmiendas de `change-branch-lifecycle`) | T-15, T-16, T-17, T-22 |
| S8 (`workflow.lane: ask` en este repositorio) | T-02 |

## Matriz de amenazas (design §7) → tareas

| Frontera | Tareas con casos de prueba |
|---|---|
| Rutas que parecen documentación | T-08 (`TestDocPathsClassifiesByRootNotExtension`), T-13 (`TestPublishRefusesUnpushedCommitTouchingReadmeSh`) |
| Selección del repositorio git | T-11 (`TestPublishIgnoresGitDirEnvironment`), T-13 (`TestPublishRunsInTargetRootFromLinkedWorktree`), T-14 (`TestPublishRelativeCwdResolvesToWorkspace`) |
| Estado del commit | T-11, T-13 (`KeepsStagedCodeStaged`, `NeverCommitsUnstagedCode`, `NoDocChangesMakesNoCommit`, `ExcludesLocalStateFiles`, `CommitsArchiveDeletions`) |
| Estado del push | T-09, T-13 (primer push, avance rápido, `non-fast-forward`, rama de integración, multirepo, `--branch`, observador sin `fetch` ni `--force`) |
| Órdenes de PR | N/A según design §7 (T-22 comprueba que ningún paso contiene `&&`, `gh ` ni `glab `) |

## Riesgos de planificación

- **WU-2 es la unidad más grande** (≈2.200 l, la mayor parte paquete nuevo) y T-13 concentra el algoritmo y la matriz de pruebas con remoto bare; si T-13 supera con claridad la heurística de 400 líneas se mantiene entera (no se parte por tamaño), porque separar el algoritmo de sus pruebas dejaría un commit sin verificar.
- **Dependencia de WU-3 respecto a WU-2 y WU-4:** T-23 a T-25 usan `PublicationTarget`, `publish` y la adopción; el orden 1 → 2 → 4 → 3 evita que un commit intermedio prometa en doctrina algo que aún no existe. No reordenar WU-3 antes de WU-4.
- **Pruebas de `internal/cli` muy lentas** (más de una hora el paquete completo): cada tarea de CLI declara su `-run` acotado; si un patrón no encuentra pruebas, ajustar el patrón y no ejecutar el paquete entero.
- **Cuello de botella en `sdd_gate.go` y `governance.go`:** los edita solo WU-3 (T-23 y T-24), de forma secuencial y con ediciones mínimas.
- **Supuesto técnico abierto (design §9):** `git rev-parse --git-path hooks/<hook>` y `core.hooksPath`. Se valida en T-13; si falla, la clasificación degrada a `commit_failed`/`push_failed` sin cambiar la arquitectura.
- **Redacción final de tokens** (por ejemplo `createdFrom`) puede ajustarse en `apply` si las specs la exigen distinta (design §9); el ajuste no cambia la arquitectura.
- **Cambio de comportamiento silencioso:** proyectos con `axiom.yaml` sin `workflow` pasan a `sdd` (design §8); queda documentado en T-07 y T-27, con `ask` como salida.
- **Un solo escritor** (sin worktrees paralelos autorizados): no hay paralelismo entre unidades; WU-1, WU-2 y WU-4 serían paralelizables en teoría, pero se ejecutan en el orden indicado.
