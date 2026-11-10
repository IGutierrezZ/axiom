# Propuesta: Carril elegido en el proyecto, ODD sin paradas y publicación ligera de documentación SDD (carril-proyecto-publicacion-docs)

> Fase `propose` · rama `inc/carril-proyecto-publicacion-docs` · 2026-11-06 · entrada: `exploration.md` (autoritativo).
> Kickoff sellado: `continuous`, rol único `fullstack` (`blocking`). Este cambio no tiene puertas propias.
> Las decisiones de producto 1 a 5 de §1.3 vienen fijadas por el usuario; los supuestos S1 a S10 de §3 los fija esta propuesta y el usuario puede corregirlos.

## 0. Resumen

Hoy cada petición sustancial empieza con la misma pregunta de carril (ODD o SDD), ODD tiene una parada de plan que nadie pidió mantener, y la salida 2 de una parada SDD («aprobar y que siga otra persona») solo imprime órdenes de `commit`/`push` para el humano y exige una PR fusionada antes de que otra persona pueda continuar. Este cambio (1) permite declarar el carril en `axiom.yaml` (`odd | sdd | ask`, `sdd` por defecto) y lo proyecta en `axiom sdd status` para que el orquestador no pregunte; (2) retira la parada de ODD; (3) hace que toda aprobación de una parada SDD publique los documentos del cambio con un verbo nativo nuevo, `axiom sdd publish`, que confirma solo rutas de documentación y sube solo la rama del cambio; y (4) hace que `axiom branch ensure` adopte la rama publicada cuando ya existe como rama de seguimiento remoto, de modo que otra persona continúa sin PR fusionada. Objetivo del usuario: menos clics.

## 1. Propósito (Intent)

### 1.1 Problema

| Hueco | Evidencia |
|---|---|
| El carril se pregunta en cada petición sustancial; no existe ninguna preferencia de proyecto y la doctrina no puede leer `axiom.yaml`. | `internal/components/agentguidance/routing.go:28` (`RenderRouting` solo recibe el agente), `routing.go:54` (paso 0); `axiom.yaml` sin campo de carril; `exploration.md` «Doctrina de carril y paradas» |
| La política «SDD solo por petición explícita o propuesta aceptada» no admite un carril declarado por el proyecto. | `internal/agents/capabilitymanifest/manifest.go:45,220` (`explicit_request_or_accepted_proposal`); `routing.go:35-36,48,79-80`; `routing_test.go:95-115` |
| ODD tiene una parada única de plan que añade un clic a todo trabajo sustancial ODD. | REQ-26.10 (`organic-agent-trigger-rules`), excepción en REQ-21.2 y mención en REQ-25.40; `routing.go:61,64,117,143-145`; `internal/assets/generic/stop-exits.md` |
| La salida 2 de una parada SDD no publica nada: `gate record --hand-off` calcula los pasos `add/commit/push` y los imprime; `gates.yaml` queda sin confirmar. | `internal/branch/engine/handoff.go`, `internal/cli/sdd_gate_handoff.go` (`writeHandOffDelivery`), `internal/cli/sdd_gate.go:53-190`; REQ-26.4, REQ-26.5 |
| Quien recoge el relevo no obtiene la rama publicada: `ensure` crea la rama desde la base si no existe localmente y nunca consulta `refs/remotes/origin/<rama>`. | `internal/branch/engine/ensure.go:306-319`; `internal/branch/gitcli/ops.go:305-317`; aviso `MergeNotice` y REQ-26.1 («PR fusionada») |
| No existe ninguna operación de push en el producto; el puerto de escritura de `branch` lo prohíbe por diseño. | `internal/branch/ports.go:7-9`, `engine.go:56-70`; REQ-25.1; `engine/archive_test.go:692-712` (`TestEnsureArchiveNeverPublishes`) |

### 1.2 Por qué ahora

La parada ODD de `paradas-tres-salidas` (archivado 2026-11-03) se ha percibido como fricción, y la salida 2 de SDD sigue costando una PR fusionada por cada relevo. El usuario quiere que el proyecto fije el carril una vez, que ODD no se detenga y que aprobar una parada deje los documentos publicados para quien siga.

### 1.3 Decisiones de producto fijadas por el usuario

1. Carril en el proyecto: preferencia `odd | sdd | ask` en `axiom.yaml`, `sdd` por defecto en `multirepo` y `monorepo-embedded`; ODD sigue permitido en `multirepo`.
2. ODD sin paradas: se revierte la parada única de ODD (D2 de `paradas-tres-salidas`, REQ-26.10 y sus menciones en REQ-21.2 y REQ-25.40).
3. Paradas SDD (`checkpointed`): iterar / aprobar y seguir aquí / aprobar y parar para que siga otra persona. Al aprobar, siempre se suben los documentos SDD (también en «aprobar y seguir»). La aprobación de la parada autoriza ese push de documentación, y por eso la pregunta nombra el remoto y la rama; el push de código sigue siendo del humano.
4. Subida ligera: verbo nativo `axiom sdd publish --change <x>` con puerto propio de commit/push; commit solo de rutas de documentación del cambio, push solo de la rama del cambio (nunca `main`, nunca forzado); en `multirepo`, push a la rama de integración del repositorio de specs. Se niega, de forma informativa, si algún commit sin subir toca rutas fuera de la lista de documentación.
5. Continuación por otra persona: `axiom branch ensure` adopta `refs/remotes/origin/<rama>` cuando existe y no hay rama local, sin fetch implícito; la salida 2 deja de exigir PR fusionada en `monorepo-embedded`.

### 1.4 Resultado esperado

- Con `workflow.lane: sdd` (o sin la clave), el orquestador entra al pre-vuelo SDD para trabajo sustancial sin preguntar el carril; con `odd`, trabaja en ODD sin preguntar; con `ask`, se comporta como hoy.
- La doctrina ODD generada no contiene ninguna parada.
- Aprobar una parada SDD (salida 2 o 3) deja los documentos del cambio confirmados y subidos a `origin/<rama del cambio>`.
- Otra persona ejecuta `git fetch` (o `ensure --fetch` con su autorización) y `axiom branch ensure`, y continúa sobre la rama publicada sin PR fusionada.

## 2. Decisiones que la aprobación ratifica

| ID | Decisión | Alternativa descartada |
|---|---|---|
| D1 | **Bloque `workflow` en `axiom.yaml`** con la clave `lane: odd \| sdd \| ask`, decodificado estricto por lista cerrada de claves con el patrón del bloque `git:` (`internal/workspace/gitpolicy.go:75-97`, canal `Declarations.WorkflowErr`, `joinLoadErrors`, proyección con defecto en `parseLayered`). Ausente = `sdd`. Valor desconocido = error accionable que nombra la clave en `axiom workspace validate`. Las demás claves desconocidas de la raíz se siguen ignorando (REQ-24.17), así que un binario anterior ignora el bloque. | Clave dentro de `git:` o de `governance:` (mezcla conceptos: ni es política de ramas ni de gobierno de idioma/memoria); decodificación tolerante (un error tipográfico pasaría silenciosamente a `sdd`). |
| D2 | **Proyección en `axiom sdd status`** (A1 de la exploración): campo opcional `workflow` (forma exacta en design; p. ej. `{"lane":"sdd","source":"declared\|default\|no_workspace\|invalid"}`) en JSON v1 y v2, con o sin cambio seleccionado, siguiendo el precedente de `baseline`/`technicalContext` (REQ-24.19, `sddstatus/workspace_declarations.go:31-33`); documentado en `internal/assets/skills/_shared/sdd-status-contract.md`. El paso 0 de la doctrina instruye a leerla antes de decidir el carril. | Verbo nuevo (`axiom workspace lane`): más superficie y otro contrato. Hornear el valor en la doctrina al sincronizar: imposible en ámbito global y obsoleto en cuanto cambia `axiom.yaml`. |
| D3 | **Política de selección de SDD**: el carril declarado cuenta como petición explícita. Se sustituye el valor del manifiesto por uno nuevo (nombre en design, p. ej. `explicit_request_project_lane_or_accepted_proposal`), sin consumidores externos (solo `routing.go` y `manifest_test.go`); `routing.go` sigue fallando cerrado con `ErrUnknownRoutingPolicy` ante cualquier otro valor. El trabajo pequeño y entendido sigue sin artefactos en cualquier carril. | Mantener el literal y cambiar solo la prosa: el pin del manifiesto dejaría de describir la política real. |
| D4 | **ODD sin paradas.** Se retiran de `routing.go` la parada del paso 5, el párrafo de reanudación con cesión, la viñeta `Plan stop:` y la sección `### Stops and exits`; `stop-exits.md` deja de mencionar ODD y pasa a ser texto exclusivo de SDD (se reescribe la restricción de `stop_exits_test.go:68-75`). Se retira REQ-26.10 y se enmiendan REQ-21.2 y REQ-25.40. La reanudación ODD desde el fichero sin espejo de Engram se conserva (no depende de la parada). | Dejar la parada como opcional: el usuario pide quitarla. |
| D5 | **Verbo `axiom sdd publish --change <x>`** (B1 de la exploración) en `internal/cli`, con un puerto de publicación propio (commit y push) separado de `branch.Git` y del puerto de escritura del motor de ramas, que conservan su invariante «no publica» acotado a `axiom branch`. Ejecución por `internal/gitcommon/exec.go` (la prueba de arquitectura lo exige). Comportamiento: (a) confirma con pathspec explícito y semántica `--only` solo las rutas de documentación del cambio (S6) que tengan cambios, sin tocar otras rutas preparadas en el índice; (b) calcula los commits sin subir contra `refs/remotes/origin/<rama>` local, o contra la base de integración si la rama aún no existe en el remoto; (c) si alguno toca una ruta fuera de la lista, se niega sin push con un código informativo (nombre en design, p. ej. `code_commits_unpublished`) que incluye la orden de push exacta para el humano; (d) sube con refspec explícito `<rama>:<rama>` a `origin`, sin `--force` y sin `--no-verify`; (e) se niega en HEAD desacoplado, en rama distinta de la esperada (`wrong_branch`), en la rama de integración de un repositorio de código y con almacén `engram`/`none`; (f) un push rechazado por no ser *fast-forward*, un hook que rechaza o una identidad de autor ausente producen negativas tipadas sin reintento. Nada queda a medias: si el push falla, el commit local se conserva y se informa. | `gate record --publish` (B2): mezcla la decisión con E/S remota y rompe REQ-26.5. Mantener la entrega impresa (B3): no cumple la decisión 3. |
| D6 | **Aprobar publica siempre.** Las salidas 2 y 3 de `spec`, `design`, `tasks` y `role-apply:<rol>` son `gate record` seguido de `axiom sdd publish`; la doctrina del orquestador encadena ambas y la pregunta de la parada nombra la operación, el remoto y la rama (p. ej. «aprobar y subir los documentos a `origin/inc/<cambio>` con las credenciales de git configuradas»), lo que la convierte en la autorización explícita que exige la regla de operaciones remotas. `gate record` sigue sin ejecutar git (REQ-26.5 se reescribe repartiendo responsabilidades). REQ-26.4 se relaja: `--hand-off` ya no exige artefactos confirmados porque `publish` los confirma, incluido `gates.yaml`. Una negativa de `publish` no anula la aprobación registrada: el orquestador la presenta como informativa. La puerta `integration` no cambia. | Publicar solo en la salida 2: el usuario pide publicar en toda aprobación. |
| D7 | **Adopción de la rama publicada** (C1 de la exploración). Si la rama del cambio (o de rol) no existe localmente y existe `refs/remotes/origin/<rama>`, `ensure` crea la rama local desde esa referencia con upstream y el JSON lo indica (forma en design, p. ej. `base.source: remote_branch`). Sin esa referencia, comportamiento actual y sugerencia de `--fetch`, que sigue exigiendo autorización (REQ-25.10). Nunca hay fetch implícito. En `monorepo-embedded` la salida 2 deja de exigir PR fusionada: `MergeRequired`/`MergeNotice` se sustituyen por la indicación de que quien siga ejecute `git fetch` y `axiom branch ensure`. | `ensure --adopt-remote` con fetch propio (C2): operación remota nueva en `branch`. Mantener la PR fusionada (C3): contradice la decisión 5. |
| D8 | **Entrega en cuatro incrementos** dentro de una sola PR con `size:exception` (estrategia `exception-ok`), en este orden: (1) bloque `workflow` + proyección en `status` + paso 0 + política del manifiesto + retirada de la parada ODD; (2) puerto y verbo `publish` con su política de negativas; (3) aprobar publica siempre (doctrina, sobre de la parada, textos de `--hand-off`, REQ-26.x); (4) adopción de la rama remota en `ensure`. (1) y (2) son independientes; (3) depende de (2); (4) puede adelantarse a (3) pero solo tiene valor con ambos. Cada incremento es un commit de unidad de trabajo con sus pruebas, específicas y goldens. | PRs encadenadas: descartadas por preferencia expresa del usuario (§9). |

## 3. Supuestos a confirmar

Fijados por esta propuesta a partir de la intención del usuario. El usuario confirmó S2, S4, S8 y S9 el 2026-11-09; el resto son decisiones técnicas que ratifica la aprobación.

| ID | Supuesto |
|---|---|
| S1 | **Clave**: bloque propio `workflow.lane` en `axiom.yaml`; ausente = `sdd`. Las claves desconocidas se siguen ignorando (REQ-24.17). |
| S2 | **Confirmado.** **Carril como petición explícita**: el carril declarado cuenta como «petición explícita» de SDD (se enmiendan la política y el pin de `capabilitymanifest`). El trabajo pequeño y entendido sigue sin artefactos en cualquier carril: el carril aplica solo a trabajo sustancial. Con `odd` fijo no se pregunta nunca, ni siquiera ante un alcance arquitectónico; el usuario puede pedir SDD explícitamente en una petición concreta. Con `ask`, comportamiento actual del paso 0. |
| S3 | **Proyección**: la preferencia llega al agente proyectada en `axiom sdd status` (v1 y v2, con o sin cambio), no horneada en la doctrina; el paso 0 instruye a leerla. |
| S4 | **Confirmado: campo en el kickoff.** **Estilo `continuous`**: sin paradas; publica los documentos una vez al cerrar `tasks` y una vez al archivar, sin preguntar, solo si el kickoff lo autoriza. **Motivo**: la regla de operaciones remotas exige autorización explícita del destino y de la operación, y la elección de `continuous` en el pre-vuelo actual no nombra ni remoto ni rama; además, los kickoffs `continuous` ya sellados (incluido este cambio) se sellaron sin esa autorización. **Solución elegida**: la opción `continuous` de la pregunta de pre-vuelo nombra la publicación («sin paradas; sube los documentos a `origin/<rama>` al cerrar tasks y al archivar») y el sello registra un campo opcional en `kickoff.yaml` (nombre en design, p. ej. `docs_publication: auto`) con `omitempty`; solo los kickoffs con ese campo publican automáticamente. Los kickoffs sin él no publican solos: el orquestador ofrece `axiom sdd publish` como siguiente paso. Mismo número de clics; un binario anterior ignora el campo. |
| S5 | **Archivado**: publica la carpeta archivada, las specs vivas y la baseline o el contexto técnico mantenido que toque el archivado, sobre la rama de archivado que prepara `ensure --archive` (en `multirepo`, la integración del repositorio de specs). |
| S6 | **Lista cerrada de rutas de documentación**: `openspec/changes/<x>/**`, la carpeta de `openspec/changes/archive/**` del cambio, `openspec/specs/**`, `openspec/INDEX.md` y las rutas de baseline y de contexto técnico mantenido declaradas en `axiom.yaml` (nunca las de `technical_context.reference`). `odd/tasks/**` queda fuera: ODD no publica en este cambio. |
| S7 | **Sin rama de seguimiento remoto**, `ensure` no adopta y sugiere `--fetch`, que sigue exigiendo autorización del humano. |
| S8 | **Confirmado.** **Este repositorio** declara `workflow.lane: ask` en su `axiom.yaml` para conservar su flujo actual. |
| S9 | **Confirmado.** **`main` en `multirepo`**: «nunca `main`» y «push a la integración del repositorio de specs» chocan cuando esa integración es `main`. Se interpreta como «solo la rama que el planificador asigna al cambio en ese repositorio»: en `monorepo-embedded` y en repositorios de código se rechaza la rama de integración; en el repositorio de specs de un `multirepo` se admite su integración aunque sea `main`. |
| S10 | **Sin `axiom.yaml`** (repositorio no gobernado por Axiom) o con bloque `workflow` inválido: la proyección indica la fuente y la doctrina se comporta como `ask`, nunca como `sdd` automático. |

## 4. Alcance (Scope)

### 4.1 Dentro de alcance (In Scope)

1. **Esquema**: bloque `workflow` (`lane`) en `internal/workspace` con decodificación estricta, error accionable, defecto `sdd`; plantilla `axiom.example.yaml`; `axiom.yaml` de este repositorio con `ask` (S8); manual.
2. **Proyección** del carril efectivo y su fuente en `axiom sdd status` v1 y v2, con o sin cambio; contrato en `sdd-status-contract.md`.
3. **Doctrina**: paso 0 que lee la proyección (`sdd` → pre-vuelo directo para trabajo sustancial; `odd` → ODD sin preguntar; `ask` → pregunta actual); política del manifiesto (D3); retirada de la parada ODD (D4).
4. **Verbo `axiom sdd publish`** y su puerto, con la política de rutas (S6), de ramas (S9) y de negativas tipadas (D5), en `monorepo-embedded` y `multirepo`.
5. **Paradas SDD**: el sobre de la puerta nombra remoto y rama en las salidas 2 y 3; la doctrina del orquestador encadena `gate record` y `publish`; `--hand-off` sin exigencia de artefactos confirmados ni de PR fusionada; textos de entrega de `gate record --hand-off`.
6. **`continuous` y archivado**: publicación al cerrar `tasks` y al archivar según S4 y S5.
7. **Adopción** de la rama publicada en `ensure` (rama del cambio y de rol; `in_place` sin cambio) con indicación en el JSON.
8. **Specs, goldens** (`testdata/golden/`, 13 ficheros), `.codex/AGENTS.md`, `AGENTS.md`/`GEMINI.md` si cambian, y `docs/manual-de-inicio.md`, incluida la nota de que el CI de otros proyectos puede ejecutarse con el push de rama y queda fuera del control de Axiom.

### 4.2 Fuera de alcance (Out of Scope)

- Publicación para ODD (`odd/tasks/**`) y cualquier `--lane odd` en `publish`.
- Push de código, apertura de PR o merge desde Axiom: siguen siendo del humano.
- Fetch implícito en `ensure` o en `publish`; `--force`; `--no-verify`.
- Proyectar en `status` un estado «publicado» (ledger o comparación con el remoto): se deja para un cambio posterior salvo que el design lo encuentre imprescindible.
- Depender de `paths-ignore` u otra configuración de CI de proyectos ajenos; en este repositorio un push de rama sin PR no lanza CI (`.github/workflows/ci.yml:3-12`).
- Cambiar la puerta `integration`, el vocabulario del ledger (`approved|rejected`) o RDD.
- Editar `openspec/INDEX.md` a mano (lo regenera `axiom archive sync`).

### 4.3 Qué no cambia

REQ-21.7 (`continuous` sin paradas); la reapertura por digest; el carácter terminal de una aprobación; `handedOff` en `gates.yaml` como marca de cesión; el invariante «ningún subcomando de `axiom branch` publica»; `--fetch` opt-in con autorización (REQ-25.10); el trabajo del repositorio de specs in situ en `multirepo` (REQ-25.5/25.13).

## 5. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Baseline: `openspec/specs/` (este repositorio no declara `baseline:`). Los requisitos nuevos usan la serie REQ-27.x, libre en `openspec/`.

### 5.1 Nuevas capacidades (New Capabilities)

- `sdd-docs-publication`: verbo `axiom sdd publish`, lista cerrada de rutas de documentación, política de ramas y remoto, negativas tipadas (código sin subir, *non-fast-forward*, hooks, identidad, rama equivocada, HEAD desacoplado, almacén sin ficheros), comportamiento en `monorepo-embedded` y `multirepo`, publicación en `continuous` y al archivar.

### 5.2 Capacidades modificadas (Modified Capabilities)

- `workspace-topology`: requisito nuevo del bloque `workflow` análogo a REQ-25.26 (claves, valores, defecto, error accionable, compatibilidad).
- `axiom-sdd-cli-integration`: proyección `workflow` en `axiom sdd status` v1 y v2 con o sin cambio (REQ-13.1 o requisito nuevo).
- `sdd-lane-kickoff`: REQ-21.1 (la pregunta de carril solo con `ask` o sin `axiom.yaml`), REQ-21.2 (se retira la excepción de la parada ODD y sus dos escenarios), REQ-21.3 (entrada directa al pre-vuelo con `sdd`), REQ-25.35 (redacción «una vez elegido el carril»); S4 si se adopta el campo de publicación en el kickoff.
- `organic-agent-trigger-rules`: se retira REQ-26.10; REQ-25.40 (escenario y nota que lo citan); requisitos «SDD remains optional» y «Organic Driven Development is the orchestrator's mandatory default protocol» (el carril declarado es selección explícita); «Canonical provider-owned routing facts» (valor de la política).
- `sdd-block-review-gates`: REQ-26.1 (aprobar publica; sin referencia a REQ-26.10; sin «PR fusionada»), REQ-26.3 (la cesión no implica fusión), REQ-26.4 (se relaja), REQ-26.5 (se reescribe: `gate record` no ejecuta git, `publish` sí), REQ-26.6 (disponibilidad de la publicación según almacén y estilo), REQ-26.8 (texto único solo SDD con remoto y rama).
- `change-branch-lifecycle`: REQ-25.1 (invariante acotado a `axiom branch`), REQ-25.7 y REQ-25.10 (adopción de la rama de seguimiento remoto sin fetch implícito), REQ-26.11 (una rama publicada no es una rama integrada).
- `sdd-archive-lifecycle`: publicación tras el archivado (S5).
- `sdd-orchestrator-assets`: solo si la spec fija el texto proyectado de las puertas; la fase de spec lo confirma.

## 6. Enfoque (Approach)

- **Configuración aditiva con fallo cerrado hacia preguntar.** Bloque estricto con defecto; un bloque inválido o la ausencia de `axiom.yaml` nunca activan SDD automático (S10).
- **Un único cauce de proyecto al agente.** `axiom sdd status` ya proyecta declaraciones del proyecto sin cambio seleccionado; se añade un campo opcional y la doctrina, idéntica en todos los proyectos, lo consulta.
- **Publicación aislada.** El push vive en un puerto nuevo usado solo por `axiom sdd publish`; `ensure`, `done`, `status`, `archive` y `gate record` no pueden alcanzarlo (prueba de arquitectura o de dependencias que lo fije, además de `TestEnsureArchiveNeverPublishes`).
- **Decisión y E/S separadas.** `gate record` registra; `publish` confirma y sube. El orquestador encadena ambos tras la autorización de la parada.
- **Adopción sin red.** `ensure` reutiliza la resolución de commits existente sobre `refs/remotes/origin/<rama>`; solo cambia el origen de la rama nueva.
- **Pruebas en proceso.** `go test` por paquete con remotos *bare* temporales (`remoteWithMain` en `engine/archive_test.go` como precedente); nunca binarios sueltos. Goldens regenerados al final con `-update`.

## 7. Áreas afectadas (Affected Areas)

| Área | Impacto | Qué cambia |
|---|---|---|
| `internal/workspace/` (`types.go`, `declarations.go`, `loader.go`, fichero nuevo del bloque, `docs_contract_test.go`) | Modificado / Nuevo | Bloque `workflow`, error, defecto, pines de documentación |
| `axiom.example.yaml`, `axiom.yaml`, `docs/manual-de-inicio.md` | Modificado | Plantilla, `ask` en este repositorio, manual (carril, publicación, adopción, CI ajeno) |
| `internal/sddstatus/` (`status.go`, `status_v2.go`, `workspace_declarations.go`, pruebas) | Modificado | Proyección `workflow` v1 y v2 |
| `internal/agents/capabilitymanifest/` (`manifest.go`, `manifest_test.go`) | Modificado | Valor de la política de selección |
| `internal/components/agentguidance/` (`routing.go`, `routing_test.go`, `routing_stops_test.go`, `stop_exits_test.go`) | Modificado | Paso 0 con carril; retirada de la parada ODD |
| `internal/assets/generic/stop-exits.md`, `internal/assets/skills/_shared/` (`sdd-orchestrator-sections.md`, `sdd-status-contract.md`), `internal/assets/assets_test.go`, `internal/components/sdd/` (`orchestrator.go`, `orchestrator_stop_exits_test.go`) | Modificado | Texto de salidas solo SDD, encadenado `gate record` + `publish`, contrato de status |
| `internal/sddstatus/governance.go` | Modificado | Sobre de la parada con remoto y rama |
| `internal/cli/` (verbo nuevo `sdd_publish.go`, `sdd_gate.go`, `sdd_gate_handoff.go`), `cmd/axiom/main.go` | Nuevo / Modificado | Verbo `publish`; textos de `--hand-off` sin PR fusionada ni exigencia de confirmación |
| Puerto nuevo de publicación (paquete en design, sobre `internal/gitcommon/exec.go`) | Nuevo | Commit `--only` por pathspec y push sin forzar |
| `internal/branch/` (`handoff.go`, `engine/handoff.go`, `engine/ensure.go`, `gitcli/ops.go`, `result.go`, pruebas) | Modificado | `MergeRequired`/`MergeNotice`; adopción de la rama remota; indicación en el JSON |
| `internal/kickoff/` | Modificado (solo con S4 alternativa) | Campo opcional de publicación automática en `continuous` |
| `testdata/golden/` (13), `.codex/AGENTS.md`, `AGENTS.md`, `GEMINI.md` | Regenerado | Copias versionadas |
| `openspec/specs/` (capacidades de §5) | Modificado / Nuevo | Deltas y capacidad nueva |

## 8. Riesgos (Risks)

| Riesgo | Prob. | Mitigación |
|---|---|---|
| Cambio de comportamiento silencioso: proyectos con `axiom.yaml` sin la clave pasan de preguntar el carril a `sdd` fijo. | Alta | Nota en el manual y en la versión; `ask` disponible; S10 evita el efecto en repositorios sin `axiom.yaml`. |
| Primera operación remota ejecutada por el producto: push indebido de código, de `main` o forzado. | Media | Puerto aislado, refspec explícito, sin `--force`, negativa ante código sin subir y ante ramas no asignadas; pruebas con remoto *bare* que lo fijan; prueba de que `axiom branch` no alcanza el puerto. |
| Autorización remota insuficiente en `continuous` (S4). | Media | Alternativa recomendada de S4: autorización nombrada en el pre-vuelo y registrada en el kickoff; sin ella no hay publicación automática. |
| Rama de seguimiento remoto desactualizada o dos personas en la misma rama: push rechazado por no ser *fast-forward*. | Media | Negativa tipada sin reintento ni fetch; la aprobación queda registrada; se indica `--fetch` con autorización. |
| `role-apply` con commits de código sin subir: la publicación se niega siempre. | Alta (esperada) | Negativa informativa con la orden de push de código para el humano; la doctrina no la trata como fallo de la aprobación. |
| Hooks locales o identidad de git ausente bloquean el commit. | Baja | Negativa tipada; nunca `--no-verify` (REQ-25.22 no se toca). |
| Push directo a la integración del repositorio de specs en `multirepo` (ramas protegidas, carreras). | Media | S9; negativa tipada si el remoto rechaza; documentado en el manual. |
| CI de proyectos ajenos que se dispara con push de rama. | Media | Documentado como fuera del control de Axiom; sin dependencia de `paths-ignore`. |
| Proyección v1/v2 y `sdd-status-contract.md` desincronizados. | Media | Pruebas de ambos JSON y pin del contrato. |
| Pines de texto (`routing_test.go:95-115`, `stop_exits_test.go:68-75`) relajados a ciegas. | Media | Se reescriben con el texto nuevo; criterios §10. |
| Tamaño muy por encima de 400 líneas. | Alta | Una sola PR con `size:exception` (§9), por preferencia expresa del usuario. |

Riesgo global: **medio-alto**, concentrado en la nueva operación remota y en el cambio de comportamiento por defecto.

## 9. Entrega (Review Workload Forecast)

- Estimación no medida: 1.500 a 2.500 líneas autoradas sin goldens, repartidas en los cuatro incrementos de D8.
- Estrategia de entrega: `exception-ok`. El usuario fijó el 2026-11-09 que un cambio por encima del presupuesto va siempre en una sola PR con `size:exception`, sin PRs encadenadas. Los cuatro incrementos de D8 son commits de unidad de trabajo dentro de esa PR.

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: High (aceptado con size:exception)
```

## 10. Plan de reversión (Rollback Plan)

- **Por código.** `git revert` de los commits del cambio, incremento a incremento si hace falta (D8). No hay migraciones.
- **Por configuración.** Un binario revertido ignora el bloque `workflow` (REQ-24.17) y vuelve a preguntar el carril; el campo opcional del kickoff (S4) lo ignora el decodificador no estricto.
- **Por remoto.** Las ramas ya publicadas siguen siendo ramas normales; un binario revertido las ignora al ejecutar `ensure` y crea la rama desde la base como hoy. Ningún push se ha forzado, así que no hay historia remota que reparar.
- **Por doctrina.** `axiom sync` reproyecta las instrucciones del binario revertido (vuelve la parada ODD y la salida 2 impresa).
- **Señal para revertir.** Un push de ruta fuera de la lista de documentación, de una rama no asignada o forzado; `ensure` que adopta sin rama de seguimiento remoto o con fetch implícito; un proyecto sin `axiom.yaml` que entra en SDD sin preguntar.

## 11. Dependencias (Dependencies)

- `axiom branch status --json` (`prTarget`, ramas asignadas), `internal/gitcommon/exec.go` y los pasos de `HandOff`, ya existentes.
- `paradas-tres-salidas` archivado (REQ-26.x vigentes en `openspec/specs/`).
- Restricciones de trabajo: TDD desactivado (`strict_tdd: false`), sin compilar binarios sueltos (solo `go test` por paquete con remotos *bare* temporales), commits Conventional sin atribución de IA.

## 12. Criterios de éxito (Success Criteria)

- [ ] **Esquema.** `workflow.lane` acepta `odd`, `sdd` y `ask`; ausente resulta `sdd`; un valor desconocido produce un error que nombra la clave en `axiom workspace validate`; un `axiom.yaml` con el bloque se carga sin error con el decodificador de un binario anterior (prueba).
- [ ] **Proyección.** `axiom sdd status --json` v1 y v2 incluye el carril y su fuente con y sin cambio seleccionado; sin `axiom.yaml` o con bloque inválido la fuente lo indica y la doctrina lo trata como `ask`.
- [ ] **Doctrina.** El paso 0 generado instruye a leer la proyección y define el comportamiento de `sdd`, `odd` y `ask`; la política del manifiesto tiene el valor nuevo y `routing.go` falla cerrado con cualquier otro.
- [ ] **ODD sin paradas.** Ningún texto generado para ODD contiene `Plan stop` ni una parada; `stop-exits.md` no menciona ODD; REQ-26.10 retirado.
- [ ] **Publish confirma solo documentación.** Con cambios en rutas de documentación y en código, el commit de `publish` contiene solo las rutas de S6 y deja el resto intacto en el árbol y en el índice (prueba con remoto *bare*).
- [ ] **Publish sube solo la rama asignada.** Sube `<rama>:<rama>` a `origin` sin forzar; se niega en la rama de integración de un repositorio de código, en HEAD desacoplado, en rama equivocada y con almacén `engram`/`none`; en el repositorio de specs de un `multirepo` sube a su integración (S9).
- [ ] **Negativa por código.** Con un commit sin subir que toca código, `publish` se niega sin push con un código informativo y la orden de push para el humano.
- [ ] **Negativas tipadas.** *Non-fast-forward*, hook que rechaza e identidad ausente producen negativas tipadas sin reintento.
- [ ] **Aprobar publica.** En `checkpointed`, las salidas 2 y 3 nombran remoto y rama y la doctrina encadena `gate record` y `publish`; `gate record` no ejecuta git (prueba); `--hand-off` no exige artefactos confirmados ni PR fusionada.
- [ ] **`continuous` y archivado.** Se publica al cerrar `tasks` y al archivar según S4 y S5 (o el supuesto corregido).
- [ ] **Adopción.** Con `refs/remotes/origin/<rama>` y sin rama local, `ensure` crea la rama desde esa referencia y el JSON lo indica; sin ella, comportamiento actual y sugerencia de `--fetch`; ninguna prueba observa fetch implícito.
- [ ] **Aislamiento.** `axiom branch` (`ensure`, `done`, `status`, `archive`) y `gate record` no alcanzan el puerto de publicación (prueba de arquitectura o de dependencias) y `TestEnsureArchiveNeverPublishes` sigue pasando.
- [ ] **Paridad y goldens.** Goldens, `.codex/AGENTS.md` y manual regenerados o actualizados; texto de salidas idéntico en todos los runtimes.
- [ ] **CI verde.** `go test` de los paquetes tocados y `go vet ./...` pasan; los ratchets no empeoran.
