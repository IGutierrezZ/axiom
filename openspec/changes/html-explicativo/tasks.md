# Tareas: HTML explicativo de spec y design (C-06-b)

Rol único `fullstack`. Cada tarea es una unidad de trabajo con su código, tests y docs, y cierra con un commit convencional. Verificación siempre localizada (`go test <paquete> -run '<Patrón>'`); nunca la suite entera ni binarios sueltos; con TDD, cada tarea parte del test en rojo. Los tests nuevos de `internal/dashboard` usan `t.Parallel()` pero ninguno usa `freePort` ni el hub compartido.

## Pronóstico de entrega

| Campo | Valor |
|-------|-------|
| Tamaño estimado | XL |
| Líneas modificadas estimadas | 3200-4200 (código, tests, docs y doctrina; goldens regenerados excluidos) |

### Unidades de trabajo

| Unidad | Objetivo | Commit | Comando de prueba localizado | Arnés de ejecución | Límite de reversión | Ruta |
|--------|----------|--------|------------------------------|--------------------|---------------------|------|
| T1 | Paquete hoja `internal/explainer` | `feat(explainer): contrato del HTML explicativo` | `go test ./internal/explainer` | N/A: biblioteca pura | `internal/explainer/` | Delegada o inline según tamaño (~450 líneas): inline salvo que el padre supere ~150k tokens; sin paralelismo posible |
| T2 | Declaración `sdd.explainer_html` y ejemplo | `feat(workspace): declarar sdd.explainer_html` | `go test ./internal/workspace -run 'TestDecodeSDD\|TestSDDLoad\|TestExample\|TestDeclarationsExplainer'` | `axiom workspace validate` lo cubre el test | `internal/workspace/*`, `axiom.example.yaml` | Inline (~200 líneas, comportamiento único) |
| T3 | Campo sellado y bandera `--explainer-html` | `feat(kickoff): sellar explainer_html y resolverlo contra el proyecto` | `go test ./internal/kickoff -run 'Explainer\|TestParseSealArgs\|TestInfer\|TestLegacy'` y `go test ./internal/cli -run 'TestRunSDDKickoffSeal.*Explainer'` | Pruebas CLI en proceso | `internal/kickoff/*`, `internal/cli/sdd_kickoff*.go` | Inline (unidad cohesionada ~500 líneas; depende de T2) |
| T4 | Proyección y frescura en `sddstatus` | `feat(sddstatus): proyectar explainer y su frescura` | `go test ./internal/sddstatus -run 'Explainer\|TestSDDStatusV2CleanBreak\|TestExplainerISO'` | Pruebas en proceso sobre carpetas temporales | `internal/sddstatus/explainer*.go` y los dos campos añadidos | Inline (~600 líneas con tests; depende de T1 y T3) |
| T5 | El HTML queda fuera del digest y entra en la publicación | `test(sdd): el HTML explicativo no altera el digest y se publica` | `go test ./internal/cli -run 'TestGateArtifactDigest.*Explainer'` y `go test ./internal/sddpublish -run 'Explainer'` | N/A | Solo tests | Inline (~150 líneas) |
| T6 | Dashboard: endpoint, DTO y servicio | `feat(dashboard): servir el HTML explicativo con CSP` | `go test ./internal/dashboard -run 'Explainer\|TestIncrementDetailExplainer'` | `httptest.NewServer(server.Router())` | `server.go`, `service.go`, `types.go`, `explainer_test.go` | Inline (~600 líneas; depende de T1 y T4) |
| T7 | Dashboard: pestaña «Explicación» | `feat(dashboard): pestaña Explicación con iframe en sandbox` | `go test ./internal/dashboard -run 'TestIncrementDetailTabsAndRoutes\|TestRouterDefinesEveryRoute\|TestExplainerTab'` | N/A: sin navegador; test estático de assets | `assets/index.html`, `router.js`, `detalle*.js` | Inline (~250 líneas; depende de T6) |
| T8 | Doctrina del orquestador, contrato de estado, hook y plugin | `feat(sdd): pregunta del HTML explicativo y contrato de estado` | `go test ./internal/assets -run 'Explainer\|TestOrchestratorSectionDriftDoesNotGrow\|TestOpenCodePluginIgnoresExplainerQuestion\|TestOpenCodeSDDPreflightTopologyContract'` y `go test ./internal/cli -run 'TestSDDPreflightHookIgnores'` | Arnés de node del plugin (se omite sin node) | `sdd-orchestrator-sections.md`, `sdd-status-contract.md`, tests | Inline (~300 líneas; depende de T4) |
| T9 | Documento compartido, skills y copias por agente | `feat(sdd): sdd-spec y sdd-design generan el HTML explicativo` | `go test ./internal/assets -run 'Explainer\|TestSDDISOConvention\|TestSDDSkillsReferenceISOConvention'` | N/A | `sdd-explainer-html.md`, 2 skills, 6 copias, `explainer_html_doctrine_test.go` | Inline (~450 líneas; depende de T1 y T8) |
| T10 | Goldens regenerados | `test(components): regenerar goldens por la doctrina del HTML explicativo` | `go test ./internal/components -run 'TestGolden'` | Regeneración con `-update` y confirmación sin él | 12 goldens | Inline (mecánica; depende de T8 y T9) |
| T11 | Docs de usuario | `docs: documentar sdd.explainer_html y el HTML explicativo` | N/A (docs); comprobación estructural por lectura | N/A | Solo docs | Inline (~80 líneas) |
| T12 | Verificación localizada final | sin commit | Todos los `-run` anteriores | Ver T12 | N/A | Delegada por acción (tests) o inline |

## Fase 1: Contrato del HTML

- [x] T1 Crear `internal/explainer/explainer.go` y `explainer_test.go` (tipos `Kind`, `State`, `FileName`, `MaxBytes`, `ReadFile`, `SourceDigest`, `FormatDigest`, `CitedIDs`, `Validate`, errores y expresiones del diseño; decisión 1).
  - Cubre: prov:GAT:1, prov:GAT:2, prov:GAT:4, prov:WEB:1
  - Ficheros: `internal/explainer/explainer.go`, `internal/explainer/explainer_test.go`
  - Pruebas (RED primero): `Validate` (`<SCRIPT>` en mayúsculas, `src="https://"`, `href='//x'`, `xlink:href`, `srcset`, `url(//x)`, `@import`, `data:` admitido, meta ausente/vacía/mal formada, aviso ausente, sin `data-axiom-ref`); `SourceDigest` (orden de atributos, comillas, mayúsculas, varias etiquetas meta; absorbe hueco spec #10); `CitedIDs` (orden, sin repetidos); `ReadFile` (inexistente `fs.ErrNotExist`, enlace simbólico → `ErrNotRegular` omitido si la plataforma no lo permite, exactamente `MaxBytes` admitido, `MaxBytes+1` → `ErrTooLarge` sin contenido).
  - Terminado cuando: el paquete solo importa la biblioteca estándar y la prueba pasa.
  - Verificación: `go test ./internal/explainer`
  - Ruta: inline (unidad cohesionada; sin paralelismo; sin motivo de delegación).

## Fase 2: Declaración y sellado

- [x] T2 Declaración `sdd.explainer_html` en el decodificador estricto del bloque `sdd:`, accesor `Declarations.ExplainerHTML()` y `axiom.example.yaml` (RF-ISO-01 modificado).
  - Cubre: prov:KOF:1, RF-ISO-01
  - Ficheros: `internal/workspace/sdd.go`, `internal/workspace/declarations.go`, `internal/workspace/sdd_test.go`, `axiom.example.yaml`, `internal/workspace/example_config_test.go`
  - Pruebas (RED primero): `ask`, `always`, `never`; `quizás`, `true`, nulo, lista, mapeo, duplicado → error con clave, valor y admitidos; sin clave → `ask`; independencia de `iso` y `workflow.lane`; `ExplainerHTML()` con `SDDErr`/`ParseErr` devuelve error envuelto; el bloque descomentado del ejemplo declara `explainer_html: ask`.
  - Terminado cuando: `axiom.example.yaml` documenta los tres valores y los tests pasan.
  - Verificación: `go test ./internal/workspace -run 'TestDecodeSDD|TestSDDLoad|TestExample|TestDeclarationsExplainer'`
  - Ruta: inline (~200 líneas).

- [x] T3 Campo sellado `explainer_html`, bandera, `ResolveExplainerHTML` y cableado en `seal` (decisiones 5 y 6).
  - Cubre: prov:KOF:2, prov:KOF:3
  - Ficheros: `internal/kickoff/types.go`, `internal/kickoff/schema.go`, `internal/kickoff/args.go`, `internal/kickoff/explainer.go`, `internal/kickoff/{args,schema,infer}_test.go`, `internal/kickoff/explainer_test.go`, `internal/cli/sdd_kickoff.go`, `internal/cli/sdd_kickoff_test.go`
  - Pruebas (RED primero):
    - kickoff: bandera válida e inválida, `--infer` + bandera rechazado, `ToKickoff`, `Validate` (`quizás` nombra `yes, no`), tabla de `ResolveExplainerHTML` (always/never/ask × yes/no/vacío, contradicción con el mensaje exacto), `--infer` sin campo (también con `always` en el proyecto), lectura heredada `TestLegacyBinaryReadsAKickoffWithExplainerHTML` con struct congelado sin el campo y `Load` sin clave → `no` efectivo (absorbe hueco spec #4).
    - CLI: always/never/contradicción sin crear la carpeta ni escribir nada, `ask` con y sin bandera, carpeta nueva con `yes`, aviso en stderr solo al escribir (costura `kickoffSealWarnings`; absorbe hueco spec #5: `ask` sin bandera desde un llamante sin orquestador, como el dashboard, sella `no` y avisa), JSON `explainerHtml`, ayuda, y `seal` sobre un cambio ya sellado con otro `--explainer-html` (se comporta como hoy: rechaza sin modificar; absorbe hueco spec #9 sin añadir verbo de reactivación).
  - Terminado cuando: un rechazo no deja carpeta ni avisos de rama y todos los tests pasan.
  - Verificación: `go test ./internal/kickoff -run 'Explainer|TestParseSealArgs|TestInfer|TestLegacy'` y `go test ./internal/cli -run 'TestRunSDDKickoffSeal.*Explainer'`
  - Ruta: inline (~500 líneas, una sola unidad de comportamiento; depende de T2).

## Fase 3: Proyección y puertas

- [x] T4 `sddstatus`: `ExplainerFreshness`, `DeclaredArtifactStore`, `decorateExplainer`, `Status.Explainer`, `StatusV2Projection.Explainer` y `kickoffV2.ExplainerHTML` (decisiones 2, 3, 4, 10 y 11).
  - Cubre: prov:KOF:4, prov:GAT:4, prov:GAT:3, RF-ISO-01
  - Ficheros: `internal/sddstatus/explainer.go`, `internal/sddstatus/explainer_test.go`, `internal/sddstatus/status.go`, `internal/sddstatus/status_v2.go`
  - Pruebas (RED primero): con `yes`, fresh → stale al editar el md → missing al borrar; meta vacía → stale; varios `spec.md` por dominio y alta/baja de dominio cambian el digest (hueco spec #10); sin md de origen → tipo omitido; HTML de `MaxBytes+1` → `error` sin `state`; `specs/` ilegible o kickoff corrupto → `explainer.error`; con `no`, sin campo o con almacén `engram`/`none` no hay frescura aunque exista el HTML; `continuous` proyecta; `nextRecommended`, `dependencies` y `blockedReasons` idénticos con `yes`/`no` y con/sin HTML; valor de proyecto sin cambio seleccionado; `invalid` con `reason` (hueco diseño #7); `governance.kickoff.explainerHtml`; copia en v2; claves exactas de `status_v2_clean_break_test.go` intactas; `TestExplainerISO`: cambio ISO con HTML `stale` y `missing` no bloquea y `dependencies.archive` e `iso` idénticos a los de `fresh` (hueco diseño #10).
  - Terminado cuando: ninguna rama de `decorateExplainer` toca `NextRecommended`, `Dependencies`, `BlockedReasons`, `Notes`, `ISO` ni `Governance.Gates`.
  - Verificación: `go test ./internal/sddstatus -run 'Explainer|TestSDDStatusV2CleanBreak|TestExplainerISO'`
  - Ruta: inline (~600 líneas; un solo comportamiento; depende de T1 y T3).

- [x] T5 Pruebas de que el HTML queda fuera del digest de las puertas y entra en la publicación.
  - Cubre: prov:GAT:3, prov:GAT:2 (publicación)
  - Ficheros: `internal/cli/sdd_gate_artifact_digest_test.go`, `internal/sddpublish/publish_test.go`
  - Pruebas: crear, modificar y borrar `resumen-spec.html` y `resumen-design.html` no cambia `gateArtifactDigest` de `spec` ni de `design`, y modificar el md sí; una publicación de la carpeta del cambio incluye los `*.html` junto al resto de documentos (hueco spec #1, solo si basta una prueba; si exigiera cambiar el código de `sddpublish`, se registra como riesgo y no se amplía el alcance).
  - Verificación: `go test ./internal/cli -run 'TestGateArtifactDigest.*Explainer'` y `go test ./internal/sddpublish -run 'Explainer'`
  - Ruta: inline (~150 líneas).

## Fase 4: Dashboard

- [ ] T6 Endpoint `GET|HEAD /api/explainer`, `ReadExplainer`, `ErrExplainerNotFound`, DTO y relleno en `GetIncrementDetail` (decisiones 7, 8, 11 y 13).
  - Cubre: prov:WEB:1, prov:WEB:2 (DTO)
  - Ficheros: `internal/dashboard/server.go`, `internal/dashboard/service.go`, `internal/dashboard/types.go`, `internal/dashboard/explainer_test.go`
  - Pruebas (RED primero): 200 con `Content-Type`, CSP exacta, `nosniff`, `no-store`, `Referrer-Policy`; CSP y `nosniff` también en 400/404; 400 con `..`, `../x`, `x/..`, `a/b`, `a\b`, `%2e%2e`, `a%2fb`, `%5c`, `C:x`, `%00`, vacío; 400 con `kind` fuera de lista (`spec.md`, `kickoff.yaml`, `tasks`); 404 sin fichero, sin cambio o enlace simbólico (se omite si no se puede crear); 405 con POST, PUT y DELETE; `HEAD`; directorio o fichero no regular → 404; `MaxBytes+1` → 500 con el mensaje y sin bytes del fichero; directorio hermano con `resumen-spec.html` nunca se sirve; con `explainer_html` ausente o `no` y el fichero presente, el endpoint lo sirve (la lista blanca no depende del sellado; hueco spec #3); cambio archivado servido; DTO: `enabled` falso con `no` y sin campo (`items: []`), con `yes` estados iguales a `sddstatus.ExplainerFreshness` del mismo directorio (hueco diseño #11), sin md → `state` vacío, archivado con `yes` mismo cálculo.
  - Terminado cuando: ningún test nuevo usa `freePort` ni el hub; las pruebas con `httptest.NewServer(server.Router())`.
  - Verificación: `go test ./internal/dashboard -run 'Explainer|TestIncrementDetailExplainer'`
  - Ruta: inline (~600 líneas; depende de T1 y T4).

- [ ] T7 Pestaña «Explicación»: `index.html`, `router.js`, `detalle.js`, `detalle-explicacion.js` y tests de assets.
  - Cubre: prov:WEB:2
  - Ficheros: `internal/dashboard/assets/index.html`, `internal/dashboard/assets/js/core/router.js`, `internal/dashboard/assets/js/screens/detalle.js`, `internal/dashboard/assets/js/screens/detalle-explicacion.js`, `internal/dashboard/workflow_assets_test.go`, `internal/dashboard/assets_test.go`
  - Pruebas: pestaña y panel `explicacion`; `'explicacion'` en `INCREMENT_TABS`; `sandbox=""` literal sin `allow-same-origin` ni `allow-scripts`; textos «no está activada», «Todavía no hay documento de origen», «No hay explicación generada» y «desactualizado»; sin controles de edición, guardado ni regeneración; la pestaña va tras «Handoff» y siempre es visible.
  - Verificación: `go test ./internal/dashboard -run 'TestIncrementDetailTabsAndRoutes|TestRouterDefinesEveryRoute|TestExplainerTab'`
  - Ruta: inline (~250 líneas; depende de T6).

## Fase 5: Doctrina

- [ ] T8 Doctrina del orquestador (párrafo **Explainer HTML.** al final de `SDD Change Kickoff and Block Gates`), contrato de estado, test del hook y test del plugin de OpenCode.
  - Cubre: prov:PRE:1, prov:KOF:4, prov:GAT:2 (presentación en paradas)
  - Ficheros: `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `internal/assets/skills/_shared/sdd-status-contract.md`, `internal/cli/sdd_preflight_hook_test.go`, `internal/assets/opencode_sdd_preflight_topology_test.go`, `internal/assets/explainer_html_doctrine_test.go` (parte del orquestador y del contrato)
  - Pruebas: el orquestador contiene el marcador `Axiom SDD kickoff explainer:`, el orden (tras `axiom branch ensure`, antes de `kickoff seal`), la restricción a `ask`, una vez por cambio, nunca en ODD ni con `--infer` ni al reanudar, y `--explainer-html` (hueco spec #11); presenta el HTML y su frescura en las paradas `spec` y `design` de `checkpointed` sin sustituir al md (hueco spec #7); el contrato de estado nombra `explainerHtml`, `expectedDigest`, `reason`, `fresh|stale|missing`; gemelo del hook con pregunta y con cabecera que empiezan por el marcador (autoridad intacta; alinea la palabra «cabecera» de la pregunta abierta del diseño); plugin v1 y v2 (`SDD_PREFLIGHT_ACCEPTED_PREFIXES = [SDD_PREFLIGHT_QUESTION_PREFIX]`, el marcador no empieza por el prefijo, arnés de node); `TestOrchestratorSectionDriftDoesNotGrow` sigue verde.
  - Verificación: `go test ./internal/assets -run 'Explainer|TestOrchestratorSectionDriftDoesNotGrow|TestOpenCodePluginIgnoresExplainerQuestion|TestOpenCodeSDDPreflightTopologyContract'` y `go test ./internal/cli -run 'TestSDDPreflightHookIgnores'`
  - Ruta: inline (~300 líneas; depende de T4).

- [ ] T9 Documento compartido `sdd-explainer-html.md`, pasos en `sdd-spec` y `sdd-design`, y seis copias por agente (decisiones 9 y 12).
  - Cubre: prov:GAT:1, prov:GAT:2
  - Ficheros: `internal/assets/skills/_shared/sdd-explainer-html.md`, `internal/assets/skills/sdd-spec/SKILL.md`, `internal/assets/skills/sdd-design/SKILL.md`, `internal/assets/claude/agents/sdd-{spec,design}.md`, `internal/assets/cursor/agents/sdd-{spec,design}.md`, `internal/assets/copilot/agents/sdd-{spec,design}.agent.md`, `internal/assets/explainer_html_doctrine_test.go` (parte de skills, copias y esqueleto)
  - Pruebas (RED primero): skills, seis copias y documento compartido contienen `castellano`, `<script>`, `external`, `axiom-source-digest` y `data-axiom-ref` (falla si falta uno en cualquiera); esqueleto `html` sin hallazgos de `explainer.Validate`; `CitedIDs` no vacío, cada ID citado existe en el md de ejemplo y cada requisito del md de ejemplo está citado (hueco diseño #9); el paso cubre: HTML que no se logra escribir o incumple termina igualmente con el md, avisa y deja `missing`/`stale` sin bloquear (hueco spec #2); regeneración tras rechazo, con HTML previo ausente y tras edición manual del md (hueco spec #6, solo como regla de regeneración); `engram`/`none` no escriben HTML.
  - Verificación: `go test ./internal/assets -run 'Explainer|TestSDDISOConvention|TestSDDSkillsReferenceISOConvention'`
  - Ruta: inline (~450 líneas, mayoría texto; depende de T1 y T8).

## Fase 6: Goldens y docs

- [ ] T10 Regenerar los 12 goldens (10 de orquestador y 2 de agentes) y revisar el diff.
  - Cubre: prov:PRE:1, prov:GAT:1 (doctrina desplegada)
  - Ficheros: `testdata/golden/*` listados en «Goldens» del diseño
  - Orden: `go test ./internal/components -run 'TestGolden' -update`, después `go test ./internal/components -run 'TestGolden'` sin `-update`.
  - Terminado cuando: el diff solo muestra el párrafo **Explainer HTML.** y la sección de los agentes; nunca se edita un golden a mano.
  - Ruta: inline (mecánica).

- [ ] T11 Documentación de usuario donde el diseño la pide: `axiom.example.yaml` ya cubierto en T2; revisar y, si existe una guía de `kickoff seal` o de configuración SDD en `docs/`, añadir `sdd.explainer_html`, `--explainer-html` y la pestaña «Explicación». Si no hay documento que hable de `docs_publication` o `sdd.iso`, no se crea ninguno nuevo y se anota en el resultado.
  - Cubre: prov:KOF:1, prov:KOF:3, prov:WEB:2
  - Verificación: lectura estructural; sin comando de prueba (docs).
  - Ruta: inline.

## Fase 7: Verificación

- [ ] T12 Verificación localizada final: ejecutar cada `-run` de T1 a T10 una vez, en el orden de las tareas, sin suite completa, sin compilar binarios sueltos y sin `go build` de `axiom.exe`:
  - `go test ./internal/explainer`
  - `go test ./internal/workspace -run 'TestDecodeSDD|TestSDDLoad|TestExample|TestDeclarationsExplainer'`
  - `go test ./internal/kickoff -run 'Explainer|TestParseSealArgs|TestInfer|TestLegacy'`
  - `go test ./internal/cli -run 'TestRunSDDKickoffSeal.*Explainer|TestGateArtifactDigest.*Explainer|TestSDDPreflightHookIgnores'`
  - `go test ./internal/sddstatus -run 'Explainer|TestSDDStatusV2CleanBreak|TestExplainerISO'`
  - `go test ./internal/sddpublish -run 'Explainer'`
  - `go test ./internal/dashboard -run 'Explainer|TestIncrementDetailExplainer|TestIncrementDetailTabsAndRoutes|TestRouterDefinesEveryRoute|TestExplainerTab'`
  - `go test ./internal/assets -run 'Explainer|TestOrchestratorSectionDriftDoesNotGrow|TestOpenCodePluginIgnoresExplainerQuestion|TestOpenCodeSDDPreflightTopologyContract|TestSDDISOConvention|TestSDDSkillsReferenceISOConvention'`
  - `go test ./internal/components -run 'TestGolden'`
  - Cubre: todos los requisitos.
  - Terminado cuando: todos pasan; cualquier fallo o prueba omitida (node, enlaces simbólicos) se declara. La batería completa la ejecuta CI.

## Huecos de cobertura que quedan fuera, con motivo

- Hueco spec #8 (qué muestra la pestaña sin md de origen): ya lo resuelve la decisión 13; se cubre en T6 y T7, no es una exclusión.
- Hueco spec #9 (camino para activar el HTML en un cambio ya sellado): fuera del alcance de la propuesta; T3 solo fija que `seal` sobre un cambio sellado no cambia el valor. Reactivar exigiría un verbo nuevo.
- Hueco spec #6 (md editado a mano sin rechazo): no hay acción propia; el estado `stale` solo informa y la regeneración se hace al reabrir la fase. No se añade comando de regeneración.
- Hueco spec #2 (fallo de escritura del HTML): solo doctrina y su test de texto en T9; no hay verbo de validación en producción (`Validate` solo se usa en tests, según el diseño).
- Hueco spec #1 (publicación): T5 solo prueba; si `sddpublish` no incluyera los `*.html`, se escala como riesgo, no se modifica aquí.
- Ejecución real de un modelo generando el HTML: fuera; solo se valida el esqueleto y la doctrina.
