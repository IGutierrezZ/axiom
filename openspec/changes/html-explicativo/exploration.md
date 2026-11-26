## Exploración: html-explicativo (C-06-b)

Estado de `main` en el worktree `C:\repos\axiom-wt\html-explicativo\axiom`. Las rutas se dan relativas a él. C-06-a ya está fusionado (`internal/dashboard/service.go:583` `readArtifactFiles`). No hay `.codegraph/`.

### Estado actual

**1. Hook de preflight.**
- Solo reconoce preguntas cuyo texto empieza por `Axiom SDD preflight ` (`internal/cli/sdd_preflight_hook.go:21`, `hasSDDPreflightQuestionPrefix:391`, `sddPreflightQuestionPrefixes:385`). Es un `PreToolUse(Agent)` que corrobora en el transcript el último par `AskUserQuestion`/`tool_result` reconocido (`:252-323`). Cualquier otro `AskUserQuestion` se salta con `continue` (`:299-301`). No intercepta el texto del sellado ni la pregunta de execution style.
- El hook ya valida por marcador de grupo e ignora marcadores desconocidos (`sddPreflightQuestionGroup:476`, `resolveSDDPreflightHookBlock:555-599`). Los grupos son `pace`, `artifacts`, `pr-strategy` (`:422-430`); solo `pace` es requerido y solo en `monorepo-embedded`. En `multirepo` el hook no exige transcript y deriva el bloque (`:159-163`).
- Hay precedente exacto de interacción propia: la preferencia de Independent Review (RF-PRE-06), cuyo test fija que el hook la ignora sin alterar la autoridad (`internal/cli/sdd_preflight_hook_test.go:745`). El plugin OpenCode usa el mismo prefijo (`internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts:8,74`).
- **Conclusión: no hace falta tocar el hook.** Una pregunta con marcador `Axiom SDD kickoff explainer:` (distinto prefijo, no empieza por `Axiom SDD preflight `) es invisible para el hook y el plugin. Solo hace falta un test gemelo del de `:745` y fijar la regla de no reutilizar prefijo ni grupos.

**2. Texto doctrinal del execution style.**
- Fuente única: `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, sección `SDD Change Kickoff and Block Gates` (línea 68: `kickoff seal`, las tres opciones de estilo). La sección `SDD Independent Review` (`:86-113`, «Preference» en `:89`) es el modelo de interacción propia. La sección se proyecta a cada orquestador con `{{AXIOM_SDD_SECTION:...}}` (p. ej. `internal/assets/claude/sdd-orchestrator.md:29`); `internal/components/sdd/orchestrator.go:21` la carga.
- `internal/components/sdd/session_preflight.go:20-28` solo describe el preflight (Pace). No es el sitio de la pregunta nueva.
- Goldens que contienen el texto del kickoff (11): `testdata/golden/sdd-{vscode-instructions,opencode-multi-settings,gemini-geminimd,cursor-rules,codex-agentsmd,codex-agentsmd-powerful,codex-agentsmd-lowcost,claude-claudemd,antigravity-rulesmd}.golden` y `combined-claude-claudemd.golden`.
- Tests que fijan la sección: `internal/assets/sdd_iso_pins_test.go:230,264`, `internal/assets/orchestrator_drift_ratchet_test.go:49`, `internal/assets/assets_test.go:1074-1094`, `internal/assets/independent_review_boundary_test.go`.
- `internal/assets/skills/_shared/sdd-status-contract.md:198` documenta `kickoff.docsPublication`: ahí iría el campo nuevo.

**3. Kickoff (patrón `DocsPublication`).**
- Tipo y constante: `internal/kickoff/types.go:125-137` (campo en `FlowConfig`, `yaml:"docs_publication,omitempty"`).
- Validación: `internal/kickoff/schema.go:31` (mapa cerrado) y `:105-113`. Bandera: `internal/kickoff/args.go:133,229-239`, rechazo con `--infer` en `:260`, coherencia con estilo en `:278`, copia en `ToKickoff:384`.
- CLI: ayuda `internal/cli/sdd_kickoff.go:38`, resumen `:214`, JSON `:251,275`. Status v2: `internal/sddstatus/status_v2.go:144,205` (`kickoffV2`).
- Copia desde `axiom.yaml` a `kickoff.yaml`: `sdd.iso` con `SDDSnapshot{ISO:true}` solo sin `--infer` (`sdd_kickoff.go:98-102`, tipo en `types.go:192`). El retrosellado `--infer` (`kickoff/infer.go:47`) no lee `axiom.yaml` para `sdd:`.
- `governance.kickoff` se proyecta también en `continuous` (`internal/sddstatus/governance.go:432-433`); en `continuous` no se calculan digests.

**4. `axiom.yaml`.**
- Registro de bloques `internal/workspace/declarations.go:99-118`; `sdd:` ya existe (C-03-a): `internal/workspace/sdd.go:14-22` (`SDDConfig{ISO, IDPattern}`, `sddKeys = {iso, id_pattern}`), `decodeSDD:37`. Las claves de primer nivel no registradas se ignoran sin error (`declarations.go:72`), así que un `explainer_html` de primer nivel quedaría sin validar. `SDDErr` ya se muestra en `sdd_kickoff.go:88`, `sddstatus/iso.go:446` y `hub/init.go:71`.
- Plantilla: `axiom.example.yaml:78-104` (bloque SDD) y su test `internal/workspace/example_config_test.go:51-58`.

**5. Status y digest.**
- Decoradores aditivos: `internal/sddstatus/status.go:552-553` (`decorateRequirements`, `decoratePublication`); patrón en `requirements.go:27`. Cada campo de `Status` se copia a `StatusV2Projection` (`status_v2.go:20-68`).
- Reglas de descubrimiento de specs: `sddstatus/status.go:1527-1598` (`resolveArtifactPaths`, `findSpecFiles`) y `cli/sdd_gate_digest.go:117-151` (`specArtifactPathsForDigest`); **están duplicadas, no compartidas**. Hay exportado `sddstatus.ChangeArtifactPaths` (usado en `dashboard/service.go:585`).
- Digest reutilizable: `kickoff.ArtifactDigest(paths)` (`kickoff/digest.go:23`): SHA-256 hex sobre contenido normalizado a LF, orden de rutas ordenado. `governanceArtifactInputs` (`governance.go:472-532`) ya calcula `spec` y `design` pero solo en `checkpointed`. Para el HTML basta `ArtifactDigest(artifactPaths.Specs)` y `ArtifactDigest(artifactPaths.Design)`. Matiz: el bug sin delta juzga `proposal.md` (`governance.go:490-493`); el HTML no aplica ahí.

**6. Puertas.** `gateArtifactDigest` (`cli/sdd_gate_digest.go:40-57`) solo hashea `design.md`, `specs/**/spec.md` (o `spec.md` plano), `tasks.md`/`tasks.<rol>.md`. Un `resumen-*.html` en la raíz del cambio no entra en ningún conjunto; `sddstatus` hace lo mismo (`governance.go:472-507`). Se confirma el criterio del plan. Falta el test que lo fije. Regla compartida que C-06-a reutiliza: `sddstatus.ChangeArtifactPaths`.

**7. Dashboard.**
- Rutas: `internal/dashboard/server.go:73-123` en un `mux`; `Router()` (`:67`) devuelve el `mux` sin `withHeaders` (que solo envuelve en `Start`, `:169`). Por eso la CSP debe fijarse dentro del handler nuevo: es testeable con `httptest.NewServer(server.Router())` y no choca con C-07-h. `withHeaders` (`:211-226`) pone `Access-Control-Allow-Origin: *`; MIME en `init:31-43`. `isLocalRequest` (`:954-980`) solo protege rutas mutantes; un iframe `sandbox` sin `allow-same-origin` envía `Origin: null`, que `isLocalRequest` rechaza.
- Resolución de carpeta: `Service.locateIncrement` (`service_listing.go:41-77`) resuelve por nombre con `filepath.Join(root,"changes",name)`. Hoy **no valida `..`** (el detalle `handleIncrementDetail:529` recorta el prefijo y lo pasa). El endpoint nuevo debe validar el nombre del cambio (rechazar separadores y `..`) y construir el fichero con una lista blanca `resumen-spec.html|resumen-design.html`, nunca con una ruta del cliente.
- Detalle: `GetIncrementDetail` (`service.go:529-578`) rellena `IncrementDetailDTO` (`types.go:85-102`) con `Specs/Tasks` de `readArtifactFiles` (`:583`). Aquí cabe un campo `explainers` (`kind`, `exists`, `state` fresco/desactualizado/ausente).
- Frontend: pestañas en `assets/index.html:225-232`, `assets/js/core/router.js:13` (`INCREMENT_TABS`) y `assets/js/screens/detalle.js` (`renderTabs:75`, `renderTabContent:197`). Tests que enumeran pestañas: `internal/dashboard/workflow_assets_test.go:100-110` y `internal/dashboard/assets_test.go:282`.
- Tests: `t.Parallel()` abunda con `httptest.NewServer(Router())` (`dashboard_test.go:127,255`). Los tests que levantan puerto real usan `freePort` y el hub compartido; no admiten `t.Parallel`.
- `assets/js/core/spec.js:25` (`REQ_ID`) solo reconoce `REQ`/`REQUISITO`: un requisito `RF-XXX-NN` no se renderiza con su ID. Es preexistente y no afecta al iframe, pero la pestaña de specs sigue sin mostrar IDs `RF`.

**8. Skills.**
- `internal/assets/skills/sdd-spec/SKILL.md` (pasos 2-6, persistencia en `:212`, IDs provisionales en `:178-186`) y `internal/assets/skills/sdd-design/SKILL.md` (paso 3 `:73`, persistencia `:155`).
- Copias por agente: `internal/assets/{claude,cursor}/agents/sdd-{spec,design}.md` y `internal/assets/copilot/agents/sdd-{spec,design}.agent.md`. No existen copias `kiro` ni `kimi`. Son resúmenes cortos que remiten al SKILL.
- Goldens: `testdata/golden/sdd-claude-agent-sdd-{spec,design}.golden` (solo la copia de claude; el texto del SKILL.md no está en ningún golden salvo la del agente spec).
- Las fases reciben valores del kickoff leyendo `axiom sdd status --json` (patrón `iso.change` en `sdd-spec/SKILL.md:267`, `sdd-iso-convention.md:5,10`), no por parámetros del orquestador. El valor sellado se proyectaría en `governance.kickoff`.

**9. Specs vivas y formato de delta.**
- Últimos IDs: `sdd-lane-kickoff` RF-KOF-12 (`spec.md:497`), `sdd-preflight-configuration` RF-PRE-15 (`:853`), `sdd-block-review-gates` RF-GAT-26 (`:932`), `local-web-dashboard` RF-WEB-14 (`:321`), y `sdd-docs-publication` RF-PUB-12 (`:632`). No hay deltas activos en `openspec/changes/`.
- RF-KOF-10 (`sdd-lane-kickoff/spec.md:402`) y RF-KOF-12 (`:497`) son el modelo de campo sellado y de instantánea.
- Formato: `## ADDED Requirements` / `## MODIFIED Requirements` con `### Requirement: <nombre> (<ID>)`, escenarios `#### Scenario:` con DADO/CUANDO/ENTONCES (p. ej. `openspec/changes/archive/2026-11-26-publicacion-pendiente/specs/sdd-docs-publication/spec.md:6-8`). Con el patrón `RF-<DOM>-<NN>` los requisitos nuevos usan identificador provisional literal por dominio: `(prov:KOF:1)`, `(prov:PRE:1)`, `(prov:GAT:1)`, `(prov:WEB:1)`; el archivo asigna el número (`sdd-spec/SKILL.md:178-186`). Los deltas archivados antiguos usan `REQ-27.x`: no imitarlos.

### Áreas afectadas
- `internal/workspace/sdd.go`, `internal/workspace/declarations.go` (comentario), `axiom.example.yaml`: `sdd.explainer_html`.
- `internal/kickoff/{types,schema,args}.go`, `internal/cli/sdd_kickoff.go`: campo sellado y bandera `--explainer-html`.
- `internal/sddstatus/{status.go,status_v2.go}` + fichero nuevo `explainer.go`: proyección y frescura.
- `internal/dashboard/{server,service,types}.go` + `assets/index.html`, `assets/js/core/router.js`, `assets/js/screens/detalle.js` (y módulo nuevo para la pestaña).
- `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `sdd-status-contract.md`, `skills/sdd-{spec,design}/SKILL.md`, copias de agentes, goldens.
- Specs: `sdd-lane-kickoff`, `sdd-preflight-configuration`, `sdd-block-review-gates`, `local-web-dashboard`.

### Enfoques
1. **Interacción propia con campo sellado `explainer_html: yes|no`, valor de proyecto en `sdd.explainer_html: ask|always|never`.**
   - Pros: sin tocar hook ni plugin; patrón probado (RF-PRE-06, RF-KOF-10/12); la clave queda validada y solo afecta a SDD; la sesión solo pregunta en `ask`.
   - Cons: desvía el plan, que decía clave de primer nivel; añade una clave al bloque `sdd:`.
   - Esfuerzo: M.
2. **Clave de primer nivel `explainer_html` (literal del plan) y pregunta como cuarto grupo del preflight.**
   - Pros: sigue el texto del plan.
   - Cons: la clave de primer nivel no se valida (`declarations.go:72`); un grupo nuevo exige tocar hook, plugin TS y topología (P10 lo descarta); rompe en multirepo, que no pregunta nada.
   - Esfuerzo: L.
3. **Sin campo sellado: leer el valor vivo de `axiom.yaml` en cada fase.**
   - Pros: nada que sellar.
   - Cons: viola el principio de instantánea (RF-KOF-12); no hay respuesta por cambio con `ask`.
   - Esfuerzo: S, descartado.

### Recomendación
Enfoque 1.
- **Declaración.** `sdd.explainer_html: ask|always|never` en `sdd:` (defecto `ask`; cadena, no booleano; `workspace validate` rechaza otro valor). Se documenta en `axiom.example.yaml`.
- **Campo sellado.** `kickoff.explainer_html: yes|no` con `omitempty` (`ExplainerHTML` en `FlowConfig`, tipo `ExplainerHTML`). Ausente equivale a `no` (cambios previos y `--infer`, que no lo escribe ni lo hereda). Bandera `seal --explainer-html yes|no`; se rechaza con `--infer`. No depende del estilo de ejecución (vale en `continuous` y `checkpointed`).
- **Resolución en `seal`.** `always` sella `yes`, `never` sella `no`, sin bandera. Con `ask`, la bandera la aporta la respuesta de la persona; sin bandera se sella `no` (compatibilidad con llamadas existentes y el dashboard) y la doctrina obliga al orquestador a preguntar antes. El orquestador necesita saber si preguntar: se proyecta `sdd.explainerHtml` (valor de proyecto) en el estado, siguiendo `iso.project`.
- **Interacción propia.** Un `AskUserQuestion` de una sola pregunta, de marcador `Axiom SDD kickoff explainer:` (no reutiliza el prefijo del preflight, igual que RF-PRE-06), con opciones «Sí, generar el HTML explicativo» / «No»; se pregunta una vez al nombrar el cambio, después de `axiom branch ensure` y junto a la pregunta de estilo, antes de `kickoff seal`. Nunca en ODD ni con `always|never`. Se añade como párrafo a `SDD Change Kickoff and Block Gates` en el texto único proyectado.
- **Frescura.** `sddstatus.ChangeExplainers(changeRoot)` calcula por `spec` y `design` el estado `fresh|stale|missing` comparando `<meta name="axiom-source-digest" content="sha256:...">` con `kickoff.ArtifactDigest` de `artifactPaths.Specs` o `.Design`. Se proyecta en `Status.Explainer` (decorador `decorateExplainer` en `status.go:553`, copia en `status_v2.go`) y la reutiliza el dashboard. Solo informativo: no cambia `nextRecommended` ni dependencias.
- **Skills.** `sdd-spec` y `sdd-design` leen `governance.kickoff.explainerHtml` del status; si es `yes`, tras el md escriben el HTML (autocontenido, sin `<script>` ni URL externa, con el digest calculado de los md ya escritos). Una comprobación automática en tests de assets fija esas restricciones.
- **Dashboard.** `GET /api/increments/explainer?change=<n>&kind=spec|design`: valida el nombre (rechaza separadores y `..`), resuelve con `locateIncrement`, construye el fichero por lista blanca, y responde `text/html` con `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; img-src data:` fijada en el handler, más `X-Content-Type-Options: nosniff`. Pestaña «Explicación» con `<iframe sandbox="" srcdoc|src>` y aviso de desactualizado.

### Estimación (líneas netas, sin goldens)
- Workspace + ejemplo + tests: ~70.
- Kickoff + CLI + tests: ~110.
- Status/frescura + tests: ~120.
- Dashboard (Go + JS + tests): ~190.
- Doctrina, skills, agentes: ~60.
- Deltas de spec (cuatro dominios): ~150 (documentos).
- Total ~550 de código y tests; más goldens regenerados (11 + 2).

### Riesgos
- **C-05-d (autofetch, en paralelo).** Toca el motor de ramas, el hub, los hooks de `internal/components/sdd/inject.go` y la doctrina antes de `branch ensure` y «antes de presentar cada parada»: solape duro en `sdd-orchestrator-sections.md` (sección del kickoff y puertas) y en los 11 goldens. Mitigación: párrafo nuevo en bloque propio al final de la sección del kickoff y regenerar goldens al rebasar. Sin solape de ficheros Go con `internal/branch/engine`.
- **C-07-h / `withHeaders`.** CSP en el handler, no en `withHeaders`, evita conflicto; `Access-Control-Allow-Origin: *` sigue vigente hasta C-07-h.
- **Path traversal.** `locateIncrement` no valida `..`: el endpoint debe validar por su cuenta y los tests cubrir `..`, `%2e%2e`, separadores y otros nombres de fichero.
- **Doble implementación de la regla de specs** (`cli/sdd_gate_digest.go:117` y `sddstatus/status.go:1579`): el digest del HTML debe reutilizar `ChangeArtifactPaths`, no una tercera copia.
- **`continuous` sin digests de gobernanza** (`governance.go:432`): la frescura debe calcularse aparte, no de `Governance.Gates`.
- **`spec.js:25`** solo reconoce `REQ`: preexistente; no se toca en este cambio.
- **Contrato de HTML por modelo** (script/URLs externas): solo una comprobación automática en tests sobre ejemplos; la fase no se puede validar en ejecución sin otro verbo (fuera de alcance).
- **Tests dashboard con puerto real**: sin `t.Parallel`.
- **Opciones de pregunta fuera del hook:** al no pasar por el hook, su corrección depende de la doctrina; se fija con un test de assets y un test del hook que la ignora.

### Ready for Proposal
Sí. El orquestador debe confirmar con la persona dos desvíos del plan: clave dentro de `sdd:` (no de primer nivel) y bandera `--explainer-html` con ausencia = `no`.
