## Exploración: bateria-pruebas (C-06-c, batería de pruebas de aceptación `pruebas.md`)

Todas las rutas son relativas a `C:\repos\axiom-wt\bateria-pruebas\axiom`.

### Estado actual

1. **Esquema de `axiom.yaml`.**
   - `internal/workspace/declarations.go:33-50` define `Declarations`, con un par bloque/error por cada bloque de primer nivel registrado.
   - `readDeclarationBlocks` (`declarations.go:99-142`) localiza cada bloque con `mappingValue(root, "<clave>")`. Las claves de primer nivel no registradas se ignoran sin validar.
   - `sdd:` es un bloque de claves cerradas: `sddKeys = {"iso","id_pattern","explainer_html"}` (`workspace/sdd.go:35`). Su decodificador es `decodeSDD` (`sdd.go:50`).
   - El patrón de C-06-b es `ExplainerHTMLMode` (`sdd.go:26`) más el accesor `Declarations.ExplainerHTML()` (`declarations.go:73`), que devuelve error envolviendo `ParseErr` o `SDDErr` y nunca asume un valor.
   - Todo error de bloque se suma en `joinLoadErrors` (`workspace/loader.go:43`), y `internal/hub/init.go:65-75` (`declares()`) lista los bloques para no perderlos con `--force`.
   - **Topología.** `readDeclarationBlocks` solo recibe `displacement`, no la topología. La regla análoga «roles especializados solo en multirepo» (RF-TOP-04) NO vive en la carga: está en `workspace.Validate`, paso 3b (`workspace/validator.go:122-138`), y la carga no la aplica a propósito. Ahí va la regla `acceptance_tests.enabled: true` solo con `multirepo`.
   - Fixtures de multirepo: `workspace/workspacetest/fixtures.go:29`.

2. **Digest de la puerta `tasks`.** Dos cálculos espejo que deben coincidir fichero a fichero.
   - Lado `gate record`: `cli/sdd_gate_digest.go:50-51` y `tasksArtifactPathsForDigest` (`:65-76`), que devuelve `tasks.md` o los `RoleTaskFiles` (`kickoff/types.go:327`). `digestIfAllPresent` (`:101`) exige que todos existan.
   - Lado status: `sddstatus/governance.go:472-532` (`governanceArtifactInputs`). Parte de `resolveArtifactPaths` (`sddstatus/status.go:1533`), cuyos `ArtifactPaths` (`status.go:84-91`) no tienen campo `pruebas`.
   - Dos usuarios más de los ficheros de tareas: `cli/sdd_trace.go:345` y `sddstatus/role_tasks.go:24`.
   - **Hallazgo crítico sobre el criterio «editar tras aprobar reabre la puerta».**
     - `kickoff/machine.go:252-266` (`resolveGateStatus`): una puerta `approved` es TERMINAL; solo una puerta `rejected` se reabre (pasa a `pending` con `Reopened`) cuando cambia el digest.
     - `kickoff/machine_test.go:250` (`TestEvaluateGatesApprovedGateNeverInvalidatedByDigestChange`) lo fija a propósito: reabrir una aprobada bloquearía `sdd-apply` contra sus propias casillas de `tasks.md`.
     - RF-GAT-29 (`openspec/specs/sdd-block-review-gates/spec.md:1070-1074`) dice «el digest cambia» al tocar el md, no «se reabre».
     - Los tests existentes que demuestran la reapertura: `kickoff/machine_test.go:222` (rechazada reabre al cambiar el digest) y `cli/sdd_gate_artifact_digest_test.go:94` (`TestRunSDDGateRecordRejectionReopensAfterRealRemediation`), además de `:217` (el digest de tasks cubre cada fichero de rol) y `:352,:428` (el HTML explicativo no entra en el digest).

3. **IDs y parseo de specs delta. Ya existe casi todo.**
   - Puerto único de IDs: `internal/reqid` (`reqid.go`: `Pattern`, `Valid`, `InTitle`, `Scan`; `provisional.go`: `ScanProvisional`, `ProvisionalInTitle`). El patrón del proyecto sale de `Declarations.IDPattern()` (`declarations.go:57`) y del envoltorio `projectIDPattern` en cli.
   - `internal/openspecdoc`: `ParseChangeSpec` (`declared.go:9`) y `Declared` (`declared.go:44`).
   - `internal/isoevidence` es el precedente directo de este cambio:
     - `Scenarios(blockText)` (`scenarios.go:12`).
     - `RequirementKeys` (`reqid.go:16`).
     - `AffectedFromDelta(delta, pattern) []ScenarioKey` (`cubre.go:243`), con los escenarios de los bloques ADDED y MODIFIED. Omite los requisitos sin exactamente un ID.
     - `ParseCubre`, `CheckCoverage` y `Finding` (`cubre.go`, `model.go`).
     - `ParseMatrix` (`matrix.go:87`): lector determinista de tabla markdown con cabeceras por nombre y avisos de forma. Es la plantilla para `P-NN`. Sus ayudantes (`splitLines`, `isTableLine`, `splitCells`) no están exportados.
     - Es un paquete hoja: `isoevidence/imports_test.go` solo admite importar `reqid`.
   - Adaptador exportado para leer la delta del cambio: `sddstatus.ISOChangeRequirements(planningHome, changeRoot)` (`sddstatus/iso.go:292`), sobre `readDeltaRequirements` (`iso.go:122`).
   - **No existe parser de `pruebas.md` ni de «cubre» con escenarios.** Las líneas `Cubre:` de las tareas solo citan IDs de requisito.
   - **Interacción con el archivado (riesgo).** `livespec/change.go:176-187` y `:207-247` reescriben los provisionales de los `Cubre:` de las tareas y de la matriz de verify al integrar el cambio (`RewriteCubre`, `RewriteMatrix`). Si `pruebas.md` cita provisionales, hay que añadirlo como tercer `changeFileKind`, o quedará con citas huérfanas tras el archivado.

4. **Registro de `axiom sdd ...`.**
   - `cmd/axiom/main.go:1986-2050` (`runSDD`): ayuda con una línea por verbo, `switch` con `cli.RunSDDxxx` y mensaje de error con la lista de verbos.
   - `cmd/axiom/main_test.go:52` y `:67` fijan la lista de verbos documentados en la ayuda.
   - Hay verbos de dos palabras con subdespachador: `RunSDDGate` (`cli/sdd_gate.go:28`: `record|show`) y `kickoff seal|show`. Así encaja `axiom sdd tests check`.
   - Plantilla de un verbo de solo lectura con `--cwd`, `--change`, `--json`: `cli/sdd_archive_check.go:37-71` y `cli/sdd_trace.go:39`. Resuelven con `resolveGovernanceChangeRoot` (`cli/sdd_governance.go:34`) y `projectIDPattern`.

5. **Proyección en `axiom sdd status`** (patrón de C-06-b).
   - Tipos y lógica en `sddstatus/explainer.go`: `ExplainerProjection` (`:17`), `ExplainerFreshness` (`:62`, compartida con el dashboard) y `decorateExplainer` (`:123`).
   - Se engancha en `Resolve` (`status.go:555-559`), con el campo `Explainer` en `Status` (`status.go:229`) y su copia a `statusV2` (`status_v2.go:58` y `:270`). Es aditiva: no toca `nextRecommended`, dependencias ni digests.
   - El contrato para las skills se documenta en `assets/skills/_shared/sdd-status-contract.md:192-200`.

6. **Dashboard.**
   - C-06-b: ruta `/api/explainer` (`server.go:93`, handler `:1036`, CSP `:1024`), excluida en el canario (`secrets_canary_test.go:59`), `ReadExplainer` (`service.go:1732`), `incrementExplainer` (`:1749`), tipos `IncrementExplainerDTO` (`types.go:586`). En el front: pestaña `explicacion` (`index.html:228,234`, `core/router.js:13`, `screens/detalle.js:205`, módulo `detalle-explicacion.js`). Sus tests pinean las pestañas: `assets_test.go:282` y `workflow_assets_test.go:103,393-421`.
   - C-06-a: `GET /api/increments/<n>` devuelve `IncrementDetailDTO` (`types.go:85-104`) con `Specs` y `Tasks` como `[]ArtifactFileDTO`, leídos por `readArtifactFiles` (`service.go:585`) con la regla canónica `sddstatus.ChangeArtifactPaths`. `detalle.js:103-118` (`availableArtifacts`) los lista en «Artefactos».
   - **Para `pruebas.md` no hace falta ruta nueva.** Basta un campo del DTO de detalle (contenido, más la cobertura M/M calculada con la misma función que cli y status), y la ruta `/api/increments/` ya está cubierta por el canario, así que no hay que tocar `excludedRoutes`.
   - No hay renderizado de tablas en `core/markdown.js` confirmado; hay que comprobarlo o renderizar la tabla de `P-NN` ya parseada.

7. **Skills y goldens.**
   - `assets/skills/sdd-tasks/SKILL.md`: «Step 3: Write tasks.md» (`:62`), roster multirrol (`:74`), formato (`:81-123`).
   - Agentes delgados de `sdd-tasks`: solo existen `assets/claude/agents/sdd-tasks.md`, `assets/cursor/agents/sdd-tasks.md` y `assets/copilot/agents/sdd-tasks.agent.md`. **No hay copias para kiro ni kimi** (el plan está desfasado en esto).
   - Golden: solo `testdata/golden/sdd-claude-agent-sdd-tasks.golden` pinea un agente de tasks.
   - La skill NO lee `axiom.yaml`: lee el status. C-06-b lo hace con `governance.kickoff.explainerHtml` (sellado) y `explainer.*`. Para este cambio la skill necesita una proyección en el status, que no existe todavía.
   - El texto de la parada `tasks` vive en `_shared/sdd-orchestrator-sections.md` y se proyecta a 10 goldens de orquestador (`*-claudemd`, `*-cursor-rules`, `*-codex-agentsmd*`, `*-gemini-geminimd`, `*-opencode-multi-settings`, `*-vscode-instructions`, `*-antigravity-rulesmd`, `combined-claude-claudemd`). Tocarlo es lo caro.
   - `_shared/sdd-explainer-html.md` y `assets/explainer_html_doctrine_test.go` son la plantilla de doctrina compartida y de su test de pines.

8. **Specs vivas a tocar** (siguiente ID libre entre paréntesis).
   - `workspace-topology`: RF-TOP-01..18 (siguiente RF-TOP-19). Candidatos de referencia: RF-TOP-04 (embebido, solo `fullstack`, `:126`), RF-TOP-06 (multirepo, `:191`), RF-TOP-16 (bloque `integrations:` estricto, `:635`), RF-TOP-17 (`Locate` expone el error sin rechazar, `:686`).
   - `sdd-lane-kickoff`: RF-KOF-01..16 (siguiente RF-KOF-17). Solo aplica si se sella algo en `kickoff.yaml`; ver decisión abierta 2.
   - `sdd-block-review-gates`: RF-GAT-01..30 (siguiente RF-GAT-31). RF-GAT-04 (puerta `tasks`, `:115`), RF-GAT-06 (rechazo y reapertura, `:197`), RF-GAT-29 (digest, `:1053`) y RF-GAT-30 (frescura en status, `:1078`).
   - Probable spec nueva o extensión de una existente para la capacidad `acceptance-tests`; la decide la fase de propuesta.

9. **Riesgos de CI.**
   - Ratchet de código muerto: `scripts/deadcode-ratchet.sh` más `.deadcode-baseline.txt`; mide alcanzabilidad desde `./cmd/axiom` (`deadcode@v0.49.0`). Toda función exportada nueva debe estar llamada desde el binario (cli, status o dashboard); un ayudante usado solo por tests falla. C-06-b no dejó entradas.
   - Canario de secretos de `/api`: `dashboard/secrets_canary_test.go:41` exige clasificar toda ruta nueva. Si se reutiliza `/api/increments/<n>` no hay cambio.
   - `TestOrchestratorSectionDriftDoesNotGrow` (`assets/orchestrator_drift_ratchet_test.go:91`): cuenta variantes de cuerpo por sección entre los `sdd-orchestrator.md` de los runtimes. Editar «Agent Teams Orchestrator» (ratchet 6) o «SDD Independent Review» (ratchet 1) en un solo runtime lo rompe; hay que editar la fuente compartida.
   - Hay pines de doctrina por texto en `assets/*_test.go` (p. ej. `sdd_iso_pins_test.go`, `explainer_html_doctrine_test.go`).

10. **Roles solo en multirepo (C-01-b / RF-TOP-04).** Se aplica en `workspace.Validate` paso 3b (`validator.go:126`), con constante `monorepoRole` (`:40`) y el mensaje «solo admite el rol 'fullstack'». Es el sitio exacto y el precedente de redacción para el rechazo de `acceptance_tests.enabled: true` en monorepo.

### Áreas afectadas

- `internal/workspace/declarations.go`, `types.go`, `loader.go`, `validator.go`, y un fichero nuevo `acceptance_tests.go` (decodificador estricto).
- `internal/hub/init.go:65-75` (`declares()`), si el bloque es de primer nivel.
- `axiom.example.yaml`, `docs/manual-de-inicio.md` y `docs/openspec-config.md`: documentan las claves.
- Paquete nuevo `internal/acceptancetests`: analizador de `pruebas.md` y comprobador de cobertura.
- `internal/isoevidence`: exportar los ayudantes de tabla y de líneas, o duplicarlos.
- `internal/cli/sdd_gate_digest.go` y `internal/sddstatus/governance.go`: digest espejo.
- `internal/cli/sdd_tests.go` (nuevo) y `cmd/axiom/main.go`, `main_test.go`: verbo `sdd tests check`.
- `internal/sddstatus` (`acceptance_tests.go` nuevo, `status.go`, `status_v2.go`): proyección.
- `internal/livespec/change.go`: reescritura de provisionales en `pruebas.md`.
- `internal/dashboard` (`types.go`, `service.go`, `index.html`, `router.js`, `detalle.js`, módulo nuevo `detalle-pruebas.js`) y sus tests de assets.
- `internal/assets/skills/sdd-tasks/SKILL.md`, un compartido nuevo `_shared/sdd-acceptance-tests.md`, `_shared/sdd-status-contract.md`, y el golden `sdd-claude-agent-sdd-tasks.golden` solo si cambian los agentes delgados.
- Specs vivas citadas en el apartado 8.

### Enfoques

1. **Mínimo en superficie compartida (recomendado): bloque `acceptance_tests` de primer nivel, estado vivo en el status, sin sellar en `kickoff.yaml`, sin tocar el texto del orquestador.**
   - Bloque de primer nivel registrado (como `delivery:` o `integrations:`), tal como fija el plan, con `Declarations.AcceptanceTests`/`AcceptanceTestsErr` y accesor análogo a `ExplainerHTML()`. La regla de topología va en `Validate` 3c, igual que la de roles, y la carga tolera.
   - `pruebas.md` entra en el digest de `tasks` solo cuando existe, en `tasksArtifactPathsForDigest` y en `governanceArtifactInputs`, con una única función compartida que resuelva el conjunto para evitar que los espejos diverjan.
   - `acceptancetests.Check(delta, pruebas, pattern)` se reutiliza en `axiom sdd tests check`, en `status.acceptanceTests` y en el DTO del dashboard. Así no hay código muerto y la cobertura sale de un solo sitio.
   - La parada `tasks` presenta `pruebas.md` porque lo dice la skill `sdd-tasks` y lo proyecta el status; no se edita el texto compartido del orquestador, así que no se regeneran los 10 goldens ni se mueve el ratchet de deriva.
   - Pestaña «Pruebas» de solo lectura, con módulo propio, y campo nuevo en el DTO de detalle.
   - Pros: superficie mínima, una sola fuente de cobertura, no rompe ratchets.
   - Contras: la parada `tasks` no nombra `pruebas.md` en el texto del orquestador. Mitigado desde la skill y el status.
   - Esfuerzo: Medio.

2. **Doctrina completa en el orquestador y sellado en `kickoff.yaml`.**
   - Se sella `acceptance_tests` en `kickoff.yaml` (instantánea por cambio, como `explainerHtml`) y se edita el texto de la parada `tasks` en `_shared/sdd-orchestrator-sections.md`.
   - Pros: el cambio queda inmune a ediciones posteriores de `axiom.yaml`; coherente con `git.promotion`.
   - Contras: cambia el esquema de `kickoff.yaml` y de `kickoff seal`; arrastra 10 goldens de orquestador y la coordinación con la sección 02 (vocabulario de paradas); exige una pregunta o una derivación en el pre-vuelo, que C-01-b y C-02-c están rediseñando. Amplía mucho el alcance de una M.
   - Esfuerzo: Alto.

### Recomendación

Enfoque 1. Comparte con C-06-b el esquema de proyección y las funciones puras, y evita las dos zonas con más conflicto de la ola (kickoff y texto del orquestador). Si más adelante se quiere la instantánea por cambio, se añade sin deshacer nada.

### Riesgos

- **El criterio «editar `pruebas.md` tras aprobar `tasks` reabre la puerta» es falso con el diseño actual.** Una puerta `approved` es terminal (D-08, `machine.go:254`). Solo se puede probar que el digest cambia y que una puerta `rejected` se reabre. Hay que decidirlo con el usuario antes de la spec.
- Dependencia blanda de C-03-a: el parser debe aceptar `REQ-n.m`, `RF-<DOM>-<NN>` (`sdd.id_pattern`) y provisionales `prov:`; hay que usar `reqid` y nada más.
- La cita de escenarios: el nombre del escenario es texto libre en castellano; `isoevidence` solo exige nombres únicos por requisito en ISO. Fuera de ISO puede haber duplicados.
- Requisitos sin ID en proyectos no ISO: `AffectedFromDelta` los omite, y la cobertura M/M los ignoraría en silencio.
- El archivado deja provisionales huérfanos en `pruebas.md` si no se extiende `livespec/change.go`.
- Ratchet de código muerto: cualquier ayudante exportado que solo usen los tests rompe CI.
- Más tokens en `sdd-tasks`, y la calidad de la minimización depende del modelo en la v1 (ya recogido en el plan).
- Este repositorio es `monorepo-embedded`: no puede activar la función con su propio `axiom.yaml`; las pruebas end-to-end usan fixtures multirepo y no hay dogfooding.
- Los tests de pines de pestañas del dashboard (`assets_test.go:282`, `workflow_assets_test.go:103`) y el espejo de ayuda de `main_test.go` hay que actualizarlos.

### Decisiones abiertas

1. ¿Qué significa «reabre la puerta»? Opciones: (a) mantener D-08 y probar solo digest distinto más rechazada que reabre; (b) hacer que `pruebas.md` reabra una puerta `tasks` aprobada, lo que rompe D-08 y el deadlock con `sdd-apply`. Recomendada: (a), y corregir el criterio de terminado.
2. ¿`acceptance_tests` de primer nivel (como dice el plan) o dentro de `sdd:`? Recomendada: primer nivel, con bloque registrado, porque `sdd:` es de claves cerradas y se le adjunta semántica ISO.
3. ¿Con `enabled: true` y sin `pruebas.md`, la puerta `tasks` está bloqueada, avisa o no hace nada? Recomendada: solo aviso en `notes` del status y `tests check` en rojo; la puerta no se bloquea.
4. Sintaxis de la columna «cubre»: recomendada `ID` (cubre el requisito) o `ID#<nombre exacto del escenario>`; M/M cuenta escenarios, y un requisito cuenta cubierto con su ID suelto o con cualquiera de sus escenarios.
5. ¿Qué hacer con los requisitos sin ID fuera de ISO? Recomendada: hallazgo «sin ID citable» que hace fallar el `check`, sin exigir ISO.
6. ¿Se cubren también los MODIFIED? Recomendada: sí (ADDED y MODIFIED, como `AffectedFromDelta`); REMOVED y RENAMED no llevan escenarios.
7. ¿Los entornos de `pruebas.md` se validan contra `acceptance_tests.environments`? Recomendada: sí, hallazgo si una fila cita un entorno no declarado.
8. ¿`pruebas.md` en `continuous`? En multirepo se sella siempre `checkpointed`, así que el caso es teórico; recomendada: se genera igual y se proyecta en el status.
9. ¿Extender `livespec/change.go` en este cambio o dejarlo para C-06-d? Recomendada: en este cambio, para no dejar citas huérfanas.
10. ¿Se reutilizan los ayudantes de `isoevidence` exportándolos, o se duplica en `acceptancetests`? Recomendada: exportar un pequeño `internal/mdtable` o los tres ayudantes existentes.

### Listo para propuesta

Sí, una vez el usuario resuelva la decisión 1 (el criterio de reapertura es incompatible con D-08).
