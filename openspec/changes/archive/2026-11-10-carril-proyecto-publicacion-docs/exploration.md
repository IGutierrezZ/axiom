## Exploración: carril elegido en el proyecto, ODD sin paradas y publicación ligera de documentación SDD

Cambio: `carril-proyecto-publicacion-docs` (monorepo-embedded, rama `inc/carril-proyecto-publicacion-docs`).
Alcance: investigación únicamente; ningún fichero de código modificado.

### Estado actual

**Doctrina de carril y paradas (texto estático por runtime).**
- `internal/components/agentguidance/routing.go` genera el bloque `## Implementation Routing`. `RenderRouting(agent model.AgentID)` solo recibe el agente (`routing.go:28`): no hay ninguna entrada de proyecto, y la inyección es global o de workspace por adaptador (`inject.go:85-118`). Por tanto la doctrina NO puede leer `axiom.yaml`; es el mismo texto para todos los proyectos.
- Paso 0 (pregunta de carril) en `routing.go:54`; parada única de ODD en el paso 5 (`routing.go:61`), el párrafo de reanudación con cesión (`routing.go:64`) y la viñeta `Plan stop:` (`routing.go:117`); sección `### Stops and exits` (`routing.go:143-145`) que proyecta `StopExits()`.
- Política "SDD solo por petición explícita o propuesta aceptada": `routing.go:35-36` (falla cerrado con `ErrUnknownRoutingPolicy` si el manifiesto trae otra política), `routing.go:48` y `routing.go:79-80`; la política vive en `internal/agents/capabilitymanifest/manifest.go:45,220`.
- `internal/assets/generic/stop-exits.md` es la fuente única del texto de las tres salidas; la leen `agentguidance.StopExits()` (`stop_exits.go:19`) y los orquestadores SDD por el marcador `{{AXIOM_STOP_EXITS}}` (`internal/components/sdd/orchestrator.go:25,84`; sección `internal/assets/skills/_shared/sdd-orchestrator-sections.md:71-83`, línea 76 contiene la regla de `--hand-off`). El texto menciona explícitamente «the single ODD plan stop» y que push, fetch, PR y merge son del humano.
- Pines de prueba: `routing_stops_test.go` (10 menciones), `routing_test.go:95-115` (`TestRenderRoutingKeepsSDDSelectionExplicit`, exige «explicit request», «accepted proposal», «never select»), `stop_exits_test.go:68-75` (prohíbe en `stop-exits.md` nombrar contrato SDD porque ODD lo comparte), `orchestrator_stop_exits_test.go`, `assets_test.go:2468`.
- Goldens afectados (texto de doctrina): 13 ficheros en `testdata/golden/` (`sdd-*` ×11: windsurf, vscode, opencode-multi-settings, kiro, gemini, cursor, codex ×3, claude, antigravity; `combined-claude-claudemd`, `combined-windsurf-global-rules`). También `.codex/AGENTS.md` (salida generada versionada) y `docs/manual-de-inicio.md` (líneas ~230, 500-524).

**`axiom.yaml`: esquema, carga y validación.**
- Raíz `WorkspaceConfig` (`internal/workspace/types.go:65`): `workspace.topology`, `roles`, `governance` se decodifican en base; `baseline`, `technical_context`, `git`, `repositories` se decodifican aparte y estrictos (`yaml:"-"`) con canal de error propio (`Declarations`, `declarations.go:83-93`). El resto de claves desconocidas se ignoran (REQ-24.17), así que un binario anterior ignora una clave nueva sin error.
- Patrón exacto para añadir un bloque: tipo + `decodeX` por lista cerrada de claves (`gitpolicy.go:75-97`, `decodeStringKeys`), campo `X`/`XErr` en `Declarations`, `readDeclarationBlocks` (`declarations.go:164-193`), `joinLoadErrors` (`loader.go:28-43`), proyección con defecto en `parseLayered` (`loader.go:95-98`, `resolveGit`), plantilla `axiom.example.yaml` y manual (pines en `internal/workspace/docs_contract_test.go`, p. ej. `TestDocsDocumentTheGitBranchPolicyBlock`), requisito en `workspace-topology` (REQ-25.26 es el modelo). Otros lectores explícitos de los canales de error: `internal/branch/policy.go:100` (`config_invalid`), `internal/hub/init.go:67`, `sddstatus/workspace_declarations.go:212` (bloquea archivado ante declaraciones inválidas).
- Este repo (`axiom.yaml`): `topology: monorepo-embedded`, `git.branch_prefix: inc`, sin ningún campo de carril.

**Cómo puede consumir la configuración el agente.** Solo hay un precedente de «configuración de proyecto que llega al agente»: `axiom sdd status` proyecta `baseline` y `technicalContext` (JSON v1 `status.go:201-202` y v2 `status_v2.go:45-46,233-234`, decorador `workspace_declarations.go:31-33`), y REQ-24.19 garantiza que existe con o sin cambio seleccionado (`declarations_status_test.go:230-244`, `changeName: null`). Ese es el cauce natural: añadir una proyección opcional (p. ej. `workflow`/`lane`) al status, en v1 y v2, y documentarla en `internal/assets/skills/_shared/sdd-status-contract.md`. Alternativa: verbo nuevo ligero. No existe hoy ningún mecanismo por el que el texto generado conozca el proyecto.

**Ramas y traspaso (`internal/branch/`).**
- Puertos (`ports.go`): `Git` tiene `Fetch` como única operación remota; el puerto de escritura del motor (`engine.go:56-70`) no tiene push y su comentario lo fija (REQ-25.1). `TestEnsureArchiveNeverPublishes` (`engine/archive_test.go:692-712`) y la prueba de arquitectura (`internal/gitcommon/architecture_test.go`, solo permite lanzar procesos desde `gitcommon/exec.go`, cubre `internal/branch/**` y `internal/gitcommon`) sostienen la ausencia de operaciones remotas.
- `ensure`: la rama del cambio sale de `base.sha` (`origin/<integración>` local si existe) solo si NO existe como rama local (`engine/ensure.go:306-319`, `CreateBranch --no-track`, `gitcli/ops.go:305-317`). No consulta `refs/remotes/origin/<rama>`: tras un push, `ensure` en otra máquina crea una rama nueva desde la base, divergente de la subida. `--fetch` es opt-in y exige autorización (REQ-25.10).
- `prTarget`: `planner.go:392` (rama del cambio → integración) y `planner.go:411` (rama de rol → rama del cambio). En multirepo el repo de specs se trabaja in situ en la rama de integración (`RepoSpecsInPlace`, REQ-25.5/25.13).
- Traspaso (`branch/handoff.go`, `branch/engine/handoff.go`, `cli/sdd_gate_handoff.go`): `Engine.HandOff` solo calcula, con lecturas, los pasos `git -C <raíz> add/commit/push` (`HandOffAddStep`, `HandOffCommitStep`, `HandOffPushStep`) y los datos de PR; `writeHandOffDelivery` los imprime y dice «Axiom no ejecuta ninguna de estas órdenes». Rechaza con `uncommitted_artifacts` si hay artefactos sin confirmar (`engine/handoff.go:54-61`) y con `wrong_branch` fuera de la rama esperada. `MergeNotice`/`InPlaceNotice` exigen PR fusionada o push de integración antes de la fase siguiente.
- `gate record --hand-off` (`cli/sdd_gate.go:53-190`): `requireHandOffStops` (solo `checkpointed`), `prepareHandOff` (almacén con ficheros, artefacto presente, puerta sin resolver, entrega calculada), después escribe `gates.yaml` con `handedOff`/`handedOffTo` (`kickoff.AppendGateChecked`) y presenta la entrega. La orden NO hace commit ni push, y `gates.yaml` queda sin confirmar tras ella.
- Kickoff sellado de este cambio: `flow_mode: sdd`, `execution_style: continuous` (`kickoff.yaml`). `FlowMode` ya existe en el kickoff (`kickoff/types.go:17-56`) como carril sellado por cambio; no sirve de preferencia de proyecto.

**CI.** `.github/workflows/ci.yml:3-12`: solo `pull_request`, `push` a `main` y `workflow_dispatch`; un push de rama sin PR no lanza CI (confirmado).

### Áreas afectadas (por capa)

| Capa | Ficheros |
|---|---|
| Esquema `axiom.yaml` | `internal/workspace/{types,declarations,gitpolicy|nuevo,loader}.go`, `axiom.example.yaml`, `axiom.yaml` (dogfooding), `docs/manual-de-inicio.md`, `internal/workspace/docs_contract_test.go`, `internal/branch/policy.go:100` (decidir si la clave inválida es `config_invalid`), `internal/hub/init.go:67` |
| Proyección al agente | `internal/sddstatus/{status.go,status_v2.go,workspace_declarations.go}`, `internal/assets/skills/_shared/sdd-status-contract.md`, pruebas tipo `declarations_status_test.go` |
| Doctrina | `routing.go` (paso 0, 5, 64, 79-80, 117, 143-145), `internal/assets/generic/stop-exits.md`, `internal/assets/skills/_shared/sdd-orchestrator-sections.md:71-83`, `internal/components/sdd/orchestrator.go`, `capabilitymanifest/manifest.go` (política de selección), 13 goldens, `.codex/AGENTS.md`, pruebas `routing_test.go`, `routing_stops_test.go`, `stop_exits_test.go`, `orchestrator_stop_exits_test.go`, `assets_test.go` |
| Publicación | nuevo verbo `axiom sdd publish` (`cmd/axiom/main.go:2214-2234` y `runSDD`; `internal/cli/sdd_*.go`), nuevo puerto de publicación (commit/push) fuera del puerto `branch.Git`, `branch/handoff.go` (textos y `MergeRequired`), `branch/engine/handoff.go`, `cli/sdd_gate_handoff.go` |
| Adopción de rama remota | `branch/engine/ensure.go:306-319` y `:371-381` (in_place), `branch/gitcli/ops.go`, `branch/result.go` (nuevo `base.source`/indicador de adopción) |
| Revertir parada ODD | `routing.go` (5 puntos), `stop-exits.md`, `routing_stops_test.go`, specs `organic-agent-trigger-rules` y `sdd-lane-kickoff` |

### Requisitos vivos que habría que modificar

- `sdd-lane-kickoff`: REQ-21.1 (la pregunta de carril solo con `ask`; con `odd`/`sdd` fijo no se pregunta), REQ-21.2 (revertir la excepción de la parada única; escenarios «La parada única de ODD no cuenta como pregunta de gobernanza» y «La parada de ODD no genera handoff.md» se eliminan), REQ-21.3 (entrada directa al pre-vuelo con `sdd` fijo), REQ-25.35 (el texto «una vez elegido el carril» debe contemplar carril por proyecto; sin cambio de fondo).
- `organic-agent-trigger-rules`: REQ-26.10 (eliminar entero), REQ-25.40 (escenario «El resto del protocolo no cambia» y nota «Previously» que citan REQ-26.10), requisito «SDD remains optional» y «Organic Driven Development is the orchestrator's mandatory default protocol» (sin ID; hoy dicen SDD solo por petición explícita o propuesta aceptada: hay que admitir el carril declarado en el proyecto como selección explícita), «Canonical provider-owned routing facts» (si cambia el manifiesto).
- `sdd-block-review-gates`: REQ-26.1 (la última frase remite a REQ-26.10; la salida 2/3 pasan a publicar), REQ-26.3 (la marca de cesión sigue; ya no implica «PR fusionada»), REQ-26.4 (hoy exige artefactos confirmados antes de registrar; si `publish` confirma, se relaja), REQ-26.5 (hoy prohíbe push/commit en `gate record`; hay que reescribirlo repartiendo: `gate record` sigue sin ejecutar nada remoto y `publish` sí), REQ-26.6 (disponibilidad de la publicación según almacén; `engram`/`none` no publican), REQ-26.7 (visibilidad de cesión en `status`; ¿proyectar «publicado»?), REQ-26.8 (texto único: reescribir el significado de «push» y quitar la mención ODD).
- `change-branch-lifecycle`: REQ-25.1 (cuatro verbos y «ningún subcomando publica»; el nuevo verbo vive en `sdd` y no en `branch`, pero el invariante del puerto de escritura y `TestEnsureArchiveNeverPublishes` hay que acotarlos), REQ-25.7 y REQ-25.10 (adopción de la rama remota ya presente como remote-tracking; sin fetch implícito), REQ-25.13 (ODD sin parada no cambia su rama; sin cambio de fondo), REQ-26.11 (fusión intermedia, revisar si la rama publicada se confunde con «integrada»).
- `workspace-topology`: requisito nuevo análogo a REQ-25.26 para el bloque del carril (y REQ-2.1 si se amplía el esquema).
- `axiom-sdd-cli-integration` / `sdd-archive-lifecycle`: nuevo verbo `sdd publish` y proyección en `status`; momento de publicar al archivar.
- `sdd-preflight-configuration`: REQ-26.9 no cambia; REQ-21.5/21.6 no cambian.

### Enfoques

**A. Fuente del carril (cómo lo conoce el agente)**

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| A1. Proyección en `axiom sdd status` (sin cambio seleccionado) | Precedente exacto (`baseline`/`technicalContext`, REQ-24.19), aditivo, cubre v1/v2, un solo comando | `status` sin cambio resuelve más de lo necesario; nombre «sdd status» para un dato que usa también ODD | Medio |
| A2. Verbo nuevo mínimo (p. ej. `axiom workspace lane --json`) | Barato, sin riesgo de contrato `status` | Nuevo contrato y nueva superficie en doctrina y docs | Medio |
| A3. Hornear el valor en la doctrina al sincronizar (workspace scope) | Sin comando extra | Estático, exige re-sync al cambiar `axiom.yaml`, imposible en ámbito global | Alto y frágil |

Recomendación: A1.

**B. Publicación ligera (`axiom sdd publish`)**

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| B1. Verbo separado que invoca el orquestador tras `gate record` | `gate record` queda puro (REQ-26.5 casi intacto); la autorización del push es explícita y trazable | Dos pasos; el orquestador debe encadenarlos | Medio |
| B2. `gate record --publish` implícito en la aprobación | Un solo paso atómico | Rompe «`gate record` no ejecuta git remoto»; mezcla decisión y E/S | Alto |
| B3. Mantener la entrega impresa y solo reforzar la doctrina | Sin código | No cumple la intención (subida siempre) | Bajo |

Recomendación: B1, con puerto de publicación nuevo (commit con pathspec explícito, push de la rama concreta, sin `--force`) separado de `branch.Git`, para no romper el invariante «el puerto de escritura no publica» de `ensure`/`done`/`archive`.

**C. Continuación por otra persona**

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| C1. `ensure` crea la rama local desde `refs/remotes/origin/<rama>` si existe localmente | Sin red; coherente con REQ-25.10; reutiliza `ResolveCommit` | La persona debe haber hecho `git fetch` (o `ensure --fetch`, con autorización); nuevo resultado en el JSON | Medio |
| C2. `ensure --adopt-remote` explícito que hace fetch de esa rama | Intención clara | Más superficie, operación remota nueva | Alto |
| C3. Dejar como está (merge de PR antes de continuar) | Sin cambios | Contradice la publicación en rama | Nulo |

Recomendación: C1, más `--track`/upstream al adoptar. Hoy NO funciona: sin remote-tracking local `ensure` crea una rama nueva desde la base, y con remote-tracking tampoco la usa.

**D. Carril por proyecto y ODD sin paradas**

- Clave nueva de primer nivel decodificada como `git:` (lista cerrada, valores `odd|sdd|ask`, defecto `sdd`, error accionable nombrando la clave). Proyección con defecto en `parseLayered`.
- Revertir ODD: retirar de `routing.go` las menciones de parada (pasos 0 y 5, línea 64, viñeta 117 y sección `### Stops and exits`) y la referencia ODD de `stop-exits.md`; el texto deja de compartirse con ODD (se relaja la restricción de `stop_exits_test.go:68-75`). Retirar REQ-26.10 y sus escenarios.

### Recomendación

Cuatro incrementos encadenables, de menor a mayor riesgo: (1) bloque de carril en `axiom.yaml` + proyección en `status` + doctrina del paso 0 + revertir parada ODD (todo texto/esquema, goldens); (2) puerto y verbo `axiom sdd publish` con su política de rechazo; (3) rediseño de las salidas 2 y 3 de la parada para publicar siempre al aprobar (doctrina + `gate record` + REQ-26.x); (4) adopción de rama remota en `ensure`. El 1 y el 2 son independientes; el 3 depende del 2; el 4 puede ir antes del 3 pero solo tiene valor con 2/3.

### Huecos de diseño reales (técnicos)

1. **Invariante «no push» en tres sitios.** REQ-25.1, REQ-26.5 y los comentarios/pruebas de `ports.go:7-9`, `engine.go:60-62`, `engine/handoff.go:20-21`, `archive_test.go:692`. La prueba de arquitectura no prohíbe `push` en sí (permite el único punto de ejecución), pero la prosa de las specs sí; hay que reescribirla y acotarla al verbo `branch`.
2. **«Commits sin subir» sin red.** Se calcula contra `refs/remotes/origin/<rama>` local (o `origin/<integración>` si la rama es nueva); si está desactualizada, el push puede ser rechazado por no ser fast-forward y no hay `--force` ni fetch autorizado. Hay que definir el rechazo (`push_rejected`) y qué se informa.
3. **Qué es «documentación».** No existe hoy una lista de rutas de documentación. Candidatos: carpeta del cambio, `odd/tasks/`, `openspec/specs/**`, rutas de `baseline.documents`, `technical_context.maintained`. Hay que fijar una lista cerrada y qué hace el rechazo con rutas ambiguas (p. ej. `.gitkeep`, `index.md`).
4. **`role-apply` y código.** Con la regla de rechazo, la aprobación de un `role-apply` nunca publicará documentación mientras haya commits de código sin subir; la aprobación queda registrada y la publicación se niega. El orquestador debe tratar esa negativa como informativa y no como fallo de la aprobación.
5. **Orden commit/gates.** `gate record` escribe `gates.yaml` después de `prepareHandOff`; `publish` debe ir después y confirmar también `gates.yaml` (hoy `HandOff` lo exime de «sin confirmar»). REQ-26.4 («artefactos confirmados antes de registrar») pasa a ser innecesario para la parte que `publish` confirma.
6. **Autoría y hooks.** Un commit requiere `user.name`/`user.email`; hooks de git locales del usuario pueden bloquearlo (REQ-25.22 prohíbe que Axiom instale hooks de git, no que existan). Hay que decidir si se usa `--no-verify` (no recomendado) o se rechaza con mensaje.
7. **Multirepo in situ.** Publicar en la rama de integración del repo de specs choca con ramas protegidas y con carrera entre personas; sin PR no hay revisión. `HandOff` ya modela `InPlace` (`engine/handoff.go:159-170`).
8. **Doble proyección de estado.** `sddstatus` v1 y v2 deben llevar el campo nuevo; si se proyecta «publicado», hay que decidir su origen (ledger vs. comparación con remote-tracking).
9. **Compatibilidad.** Binario anterior con `axiom.yaml` nuevo: ignora la clave (REQ-24.17), sin error. Binario nuevo con `axiom.yaml` viejo: defecto `sdd`, cambio de comportamiento silencioso respecto a la pregunta actual (ver riesgos).
10. **Verificación local.** Por política de Cortex (memoria del usuario) las pruebas deben ser en proceso con `go test` y remotos bare temporales (ya hay `remoteWithMain` en `engine/archive_test.go`), sin compilar binarios sueltos.

### Decisiones de producto abiertas (no decididas aquí)

1. **Nombre y lugar de la clave**: bloque propio (p. ej. `workflow: {lane: …}`) frente a clave en `git:` o en `governance:`. Recomendación técnica: bloque propio decodificado estricto. Pendiente de elegir nombre.
2. **Interacción con «SDD solo por petición explícita o propuesta aceptada»**: con `sdd` por defecto, el carril declarado en el proyecto pasa a ser una selección explícita permanente. Falta decidir qué pasa con trabajo pequeño y comprendido bajo `sdd`: ¿sigue sin crear artefactos (comportamiento actual de ODD paso 4) o todo trabajo sustancial va a SDD? Y con `odd` fijo, ¿un alcance inequívocamente arquitectónico sigue yendo a SDD o se queda en ODD?
3. **Estilo `continuous`**: no tiene paradas ni compuertas, luego «subir al aprobar» no ocurre nunca. ¿Se publica algo (p. ej. solo con `axiom sdd publish` manual) o no?
4. **Salida 2 tras publicar**: ¿sigue exigiendo PR fusionada (`MergeRequired`, `MergeNotice`) o basta con adoptar la rama publicada? Hoy el aviso dice que `ensure` en otra máquina no recupera trabajo sin fusionar; con C1 eso deja de ser cierto.
5. **Archivado**: cuándo se publica (compuerta `integration`, `archive-compose`, rama `-archive`) y si incluye las specs vivas y la baseline fuera de la carpeta del cambio.
6. **Autorización remota**: la doctrina global exige autorización explícita del destino, la operación y la credencial/sesión antes de trabajo remoto. Para que la aprobación de la parada valga como autorización, la pregunta de la parada debe nombrar remoto, rama y operación («subir documentos a `origin/<rama>`»); falta aprobar esa redacción y cómo se refleja en el sobre tipado de la compuerta (REQ-26.2).
7. **Multirepo**: ¿push directo a la rama de integración del repo de specs o rama de cambio también en specs? La instrucción previa dice integración; riesgo de rama protegida.
8. **¿`publish` sirve también a ODD (`--lane odd`)?** Con ODD sin paradas, la intención dice solo SDD.
9. **Valor de `axiom.yaml` de este repositorio** tras el cambio (hoy se usa mucho ODD; el defecto `sdd` cambiaría el flujo de quien trabaje aquí).
10. **Adopción**: ¿basta con remote-tracking ya presente, o se autoriza `ensure --fetch` para esa rama?

### Riesgos

- Cambio silencioso de comportamiento: proyectos sin la clave pasan de «preguntar el carril» a `sdd` fijo. Mitigación: nota de versión y `ask` disponible.
- 13 goldens y `.codex/AGENTS.md` cambian a la vez; pines de texto (`routing_test.go:95-115`, `stop_exits_test.go`) deben reescribirse, no relajarse a ciegas.
- Romper el invariante de seguridad «ningún subcomando de `branch` publica»: el verbo debe quedar en `sdd`, con puerto separado y prueba que impida que `ensure`/`done`/`status`/`gate record` lleguen a `push`.
- Operación remota nueva ejecutada por el producto: requiere pruebas con remoto bare, rechazo de `--force`, `main`, rama desconocida y HEAD desacoplado; límites de pathspec (`pathspecArgument`, `gitcli/ops.go:436`).
- Rechazo por stale remote-tracking y carreras entre dos personas en la misma rama.
- Ramas protegidas en multirepo (push directo a integración).
- Doble proyección v1/v2 de `status` y contrato `sdd-status-contract.md` desincronizados.
- Tamaño: el conjunto supera con holgura 400 líneas; estrategia de entrega declarada `single-pr`, hay que vigilar el presupuesto o pedir cadena.

### Listo para propuesta

Sí, con las decisiones de producto 2, 3, 4, 5 y 6 resueltas antes (o fijadas como supuestos explícitos en la propuesta). Las demás pueden quedar como supuestos recomendados.

## Key Learnings

1. La doctrina generada por `RenderRouting` no recibe ningún dato de proyecto, así que cualquier preferencia por proyecto debe llegar al agente por un comando nativo como la proyección de `axiom sdd status`.
2. `axiom sdd status` ya proyecta `baseline` y `technicalContext` con o sin cambio seleccionado, lo que convierte un campo de carril en un cambio aditivo con precedente claro.
3. `axiom branch ensure` solo crea la rama desde la base cuando no existe localmente y nunca mira `refs/remotes/origin/<rama>`, por lo que no adopta una rama ya publicada.
4. `gate record --hand-off` solo calcula y escribe `gates.yaml`; los pasos de commit y push son texto para el humano, y el puerto de escritura no tiene método de publicación por diseño (REQ-25.1).
5. Revertir la parada única de ODD afecta a cinco puntos de `routing.go`, al texto compartido `stop-exits.md` y a REQ-26.10, REQ-21.2 y REQ-25.40, con 13 goldens y `.codex/AGENTS.md` regenerados.
