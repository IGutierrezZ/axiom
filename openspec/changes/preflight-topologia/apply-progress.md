# Progreso de apply: preflight-topologia (C-01-b)

Modo: estándar (TDD estricto no activo; RED observado igualmente en cada tarea). Estrategia de entrega: `exception-ok` (una PR con `size:exception`). Lotes 1 a 5: U1 a U6.

## Unidades completadas

| Unidad | Estado | Commit |
|---|---|---|
| U0 | hecha | `1ca2b85c` |
| U1 | hecha | `37da0abf` |
| U2 | hecha | `9383953d` |
| U3 | hecha | `494049a5` |
| U4 | hecha | `9b53f58f` |
| U5 | hecha | `43094d2d` |
| U6 | hecha | ver `git log` (`feat(sdd): el texto y la doctrina fijan el almacén hybrid y el ritmo por topología`) |

## U1 · Work Unit Evidence

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/workspace/ -run 'GoverningTopology\|ValidateMonorepoRejects\|ValidateMultirepoKeeps\|LocateLoadsMonorepoRoles\|LegacyArtifactStore\|DeclarationPaths'`: ok |
| Arnés de ejecución | `go test ./internal/workspace/... ./internal/sddstatus/...`: ok (el status no cambia) |
| Límite de reversión | `internal/workspace/{locate,validator}.go` y sus pruebas; U2 a U4 dependen de `GoverningTopology` |

Notas de diseño aplicadas: con contexto técnico mantenido, `storeErrors` emite el error incondicional de REQ-2.6 (que nombra también `technical_context.maintained`) y además el error existente de REQ-24.62; sin contexto mantenido, solo el primero. `ValidateDeclarationPaths` ya no sale pronto cuando no hay `technical_context`.

## U2 · Sellado del kickoff por topología

Hecha (commit `feat(kickoff): el sellado aplica la topología al estilo de ejecución y a los roles`; U1 es `37da0abf`).

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/kickoff/...`: ok; `go test ./internal/cli/ -run 'TestKickoffSeal\|TestWriter\|TestGate\|TestRunSDDGate\|Seal\|Kickoff\|WriterGuard'`: ok (82 s) |
| Arnés de ejecución | Sellados reales en proceso sobre workspaces temporales (`workspacetest.Embedded`, `Canonical`, git aislado), sin binario suelto |
| Cierre | `gofmt -l ./internal ./cmd` vacío; `go vet` de kickoff, workspace y cli limpio; ratchet de negativas y `deadcode-ratchet.sh` en verde; `go test ./cmd/axiom/ -run HandoffGuard` ok |
| Límite de reversión | `internal/kickoff/args.go`, `internal/cli/sdd_kickoff.go` y sus pruebas (U1 queda intacta) |

Notas:
- Las pruebas nuevas de `ApplyTopology` viven en `internal/kickoff/apply_topology_test.go` (fichero nuevo, no `args_test.go`); `args_test.go` solo adapta las dos pruebas heredadas.
- El ayudante `sealLegacyKickoff` (en `sdd_kickoff_branch_test.go`) lo usan también `sdd_writer_guard_test.go` y, por transitividad, `sdd_gate_handoff_test.go` (`handOffProject`), que sellaban `core`/`qa` en monorepo y no estaban listadas en `tasks.md`.
- `TestKickoffSealOfAMultirepoRequiresTheSpecsIntegrationBranch...` sellaba `continuous` en multirepo; pasa a `checkpointed`.
- `TestKickoffSealDoesNotRewriteTheSnapshotOnASecondSeal` ya no repite el segundo sellado con `core`/`qa` (el rechazo por topología lo impediría); lo repite con `checkpointed` sin roles.
- Nuevas: `TestKickoffSealRejectsSpecializedRolesInAMonorepoBeforeTheBranchGuard` y `TestRunSDDKickoffSealUnknownTopologyAppliesMonorepoRule`.
- `ParseSealArgs` sigue rechazando `--docs-publication` sin `continuous` (con estilo vacío también).

## U3 · Alta del proyecto (hub y dashboard)

Hecha (lote 2; commit `494049a5`, `feat(hub): el alta en monorepo-embedded escribe solo el rol fullstack`).

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/hub/... ./internal/dashboard/...`: ok; RED del dashboard observado (`TestProjectFormValidationRuntime`: monorepo devolvía los roles del formulario) |
| Arnés de ejecución | `Init`/`ValidateInit` reales sobre carpetas temporales con `NewDetector()`; formulario ejecutado con Node (`runNodeModules`) |
| Cierre | `gofmt -l` vacío; `go vet ./internal/hub/... ./internal/dashboard/...` limpio; `go test ./internal/workspace/...` ok |
| Límite de reversión | `internal/hub/{init,init_validate}.go`, `internal/dashboard/assets/{index.html,js/projects.js,js/core/projectform.js}` y sus pruebas |

Notas:
- `planConfig` y `buildAxiomYaml` aplican `tech.ConfiguredRoles` solo si la topología es `multirepo`; el perfil `knowledge`/`spec-only` no cambia; los roles explícitos se conservan y los rechaza `ValidateInit` (D9).
- Dashboard: se oculta la sección completa de roles (`#field-project-roles`, que envuelve a `#roles-builder-container`, el botón y su ayuda) en monorepo, no solo el contenedor; `validateProjectForm` devuelve `roles: []` únicamente con `topology === 'monorepo-embedded'` (sin topología declarada se comporta como antes).
- Pruebas heredadas adaptadas sin debilitarlas: las que ejercen roles especializados (claves seguras, violaciones juntas, roles personalizados, adopción con roles detectados, `tech` por rol y 400 con violaciones del dashboard) pasan a `Topology: "multirepo"`.
- Las cuatro pruebas nuevas del hub se escribieron tras la implementación (el RED de la conducta ya se observó en las pruebas heredadas que fallaron tras U1); el RED del formulario sí se observó antes del GREEN.
- `buildAxiomYaml` no tiene llamadores (código muerto previo); se aplicó igualmente la regla por el diseño.

## U4 · Hook de preflight por topología

Hecha (lote 3; commit `9b53f58f`, `feat(cli): el hook de preflight valida los grupos que exige la topología`).

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/cli/ -run "SDDPreflight\|Preflight"`: ok (165 s). RED observado: la compilación de las pruebas nuevas fallaba (`group.ID` sin definir) antes de implementar |
| Arnés de ejecución | Payloads y transcripciones sintéticas con `cwd` explícito (`workspacetest.Embedded`, `Canonical`, `WithBrokenPointer`); los payloads sin `cwd` resuelven el `axiom.yaml` de este repositorio (monorepo) |
| Cierre | `gofmt -l internal/cli` vacío; `go vet ./internal/cli/` limpio; `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` ok; `go test ./internal/sddtaskresult/...` ok; `deadcode-ratchet.sh`: sin funciones inalcanzables nuevas |
| Límite de reversión | `internal/cli/sdd_preflight_hook.go`, `sdd_preflight_hook_test.go` y `sdd_preflight_hook_topology_test.go` |

Notas:
- Las pruebas nuevas viven en `internal/cli/sdd_preflight_hook_topology_test.go` (fichero nuevo); `sdd_preflight_hook_test.go` solo adapta las existentes (almacén `hybrid`, mensajes con el id del grupo, causa de «grupo requerido ausente»). Las pruebas de opción renombrada, omitida y repetida pasan al grupo `pr-strategy`, porque el grupo heredado `artifacts` ya se ignora.
- `sddPreflightDerivedBlock()` no recibe la topología (desviación menor del diseño): con cero grupos requeridos todos los valores son los por defecto, así que la topología no aporta nada. `sddPreflightHookTopology(cwd)` devuelve `monorepo-embedded` ante un error de `Locate`.
- «Un prompt que ya trae el bloque no se duplica»: el hook deniega siempre un prompt con el encabezado del bloque (denegación de bloque del modelo, también con cero grupos); la prueba afirma que el prompt permitido contiene el encabezado exactamente una vez.
- `internal/sddtaskresult/preflight_question_harness_test.go` ejercita el plugin de OpenCode (U5), no el hook: no cambia.

## U5 · Plugin de OpenCode (dos copias)

Hecha (lote 4; commit `feat(opencode): el plugin aplica el contrato de grupos por topología del hook`, el hash está en `git log`).

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/assets/ -run "OpenCode|Opencode|SDD"`: ok; `go test ./internal/sddtaskresult/...`: ok; `go test ./internal/components/sdd/ -run "OpenCode|Opencode|Digest|Plugin"`: ok (48 s). RED observado: con las dos copias restauradas a la revisión anterior (`git show HEAD:...`), fallan `TestOpenCodeSDDPreflightTopologyContract`, `TestOpenCodeV2SDDDispatch`, `TestOpenCodeV2SDDPreflightAcceptsBothPrefixes`, las dos pruebas de sincronía de bloques y tablas, `TestOpenCodeTopologyReaderParity` y el arnés de v1 |
| Arnés de ejecución | Los plugins reales (v1 y v2) ejecutados con Node (`--experimental-strip-types`) con el mismo guion sobre un adaptador por generación; el arnés de v1 de `internal/sddtaskresult` ejecuta también con Node cuando `bun` no está en el `PATH` (en esta máquina no lo está); paridad del lector de topología frente a `workspace.Locate`/`GoverningTopology` sobre 34 directorios más tres rutas relativas |
| Límite de reversión | Las dos copias de `sdd-task-result-artifacts.ts`, `opencode_plugin_digests.go`, la cuenta de la tabla en `opencode_runtime_test.go` y las pruebas de plugin (el digest antiguo sigue registrado) |

Notas:
- Digests registrados antes de editar (5.1): v1 `5b259f9f…` y v2 `e12ef869…`, ambos de la revisión `386e0cc0`; `TestHistoricalOpenCodePluginDigestTable` pasa de 8/3 a 9/4 digests.
- Contrato replicado: tabla `SDD_PREFLIGHT_GROUPS` (`id`, `header`, `legacyNumber`, `requiredIn`, opciones con `token`) y `SDD_PREFLIGHT_BLOCK_FIELDS` espejo de las del hook; el grupo heredado `artifacts` se reconoce y se ignora (queda en la tabla con `requiredIn: []`, y su pregunta no se toca); las preguntas sin marcador nunca se asignan por posición. La canonicalización reescribe el marcador (heredado o con prefijo heredado) al nominal con el prefijo Axiom solo en los grupos requeridos.
- El `before` de `question` también rechaza un grupo requerido ausente o repetido nombrándolo (conserva el rechazo temprano que antes daba «exactamente tres preguntas»); el `after` es la red de seguridad y valida por etiqueta (conjunto exacto de etiquetas ofrecidas, selección única, respuesta en el dominio) con los mismos motivos de rechazo que el hook. Las respuestas del host son posicionales respecto a las preguntas, así que la respuesta de un grupo es la del índice de su pregunta.
- Con cero grupos requeridos el despacho antepone el bloque derivado sin exigir pregunta, y se conservan las denegaciones de sesión hija y de bloque del modelo (se comprueban antes). La prueba de cero grupos sustituye en el fuente la fila `requiredIn` de `pr-strategy` (fixture de tabla).
- Lector de topología (bloque `// axiom-topology:begin … end`, idéntico en las dos copias, comprobado por `TestSDDTaskResultArtifactsTopologyReaderIsInSyncAcrossGenerations`): sigue el marcador más cercano como `Locate` (puntero sobre `axiom.yaml` sobre `openspec/` suelto), un puntero roto o un inicio inexistente dan `monorepo-embedded`. Divergencias conocidas y aceptadas, documentadas en el código: estilo de flujo de YAML (se trata como ilegible: lado seguro) y no comprobar que exista el repositorio de especificaciones de un multirepo (el binario falla en `Locate` y gobierna monorepo; el plugin lee multirepo). El bloque `axiom-project-gate` no se tocó.
- `preflight_question_harness_test.go` (arnés de v1) pasa a usar `bun` si existe y, si no, `node --experimental-strip-types` (importa el plugin con extensión explícita); se omite solo sin ninguno de los dos. Así corre en esta máquina, donde `bun` no está instalado. Los escenarios se reescribieron al contrato nuevo (dos grupos, marcadores nominales y heredados, artefactos ignorados, sin asignación posicional).
- Pruebas nuevas en `internal/assets/opencode_sdd_preflight_topology_test.go` (mismo guion en v1 y v2) y `internal/sddtaskresult/topology_reader_parity_test.go`. `opencode_sdd_project_gate_test.go`, `transport_parity_test.go` y `sdd_task_result_transport_test.go` no necesitaron cambios (los escenarios de dos preguntas ya se rechazaban por grupo requerido ausente); `TestSDDTaskResultArtifactsPluginContract` sigue fijando las etiquetas, el `NFD` y las frases de rechazo.

## U6 · Texto proyectado, doctrina y goldens

Hecha (lote 5; commit `feat(sdd): el texto y la doctrina fijan el almacén hybrid y el ritmo por topología`, el hash está en `git log`).

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/components/sdd/`: ok; `go test ./internal/assets/...`: ok; `go test ./internal/components/ -run TestGolden` (tras `-update` y revisión del diff): ok |
| Arnés de ejecución | Proyecciones reales de todos los runtimes (`composeOrchestratorPrompt`, `renderClaudeSessionPreflight`, inyección en `opencode.json`) y los 13 goldens regenerados con `-update` |
| Límite de reversión | `session_preflight.go`, `orchestrator.go`, las doctrinas de `internal/assets/**`, los goldens y sus pruebas (hook y plugin siguen aceptando los marcadores heredados) |

Notas:
- Cuerpo canónico y variante sin UI según D6 (el ancla de la variante es la línea de `Pace` de la lista); el párrafo de autoridad es una constante (`sddSessionPreflightAuthorityParagraph`) que la variante sin UI sustituye por su texto de solo prompt. Se conserva el rechazo específico de `Both -> both`.
- Doctrina: «Execution Mode» sin «default to Automatic» (frase nueva idéntica en los 12 assets) y «Artifact Store Mode» con la frase única de D11. Los assets cuyo cuerpo es idéntico al de la sección compartida pasan a la referencia `{{GENTLE_AI_SDD_SECTION:...}}` (la prueba de no-deriva lo exige); lo hacen antigravity, codex, kimi, kiro, opencode y windsurf. «Artifact Store Policy» de Claude, de OpenCode y de la sección compartida pasa al texto de política fija (decisión de este lote, coherente con REQ-28.20: ningún asset indica elegir `engram` o `none` según la disponibilidad de Engram). Kickoff y `persistence-contract.md` según D11.
- `orchestrator.go`: reescrituras de reserva y cota de «Execution Mode» a la frase nueva en el mismo commit; el encabezado de Codex pasa a «Artifact store (status-reported backend)».
- Pruebas: nuevas `TestSessionPreflightBodyNamesTopologyGroups`, `TestSessionPreflightFallbackAnswerSyntaxUsesRequiredGroups` y `TestPersistenceContractFixesTheHybridSessionStore`; `TestSDDOrchestratorAssetsDefaultToAutomatic` pasa a `TestSDDOrchestratorAssetsExecutionModeFollowsTopology`; el pin de REQ-28.14 prohíbe además `pace`, `pr-strategy` y `N/3`; la regexp de grupos de estrategia de PR y los literales de `inject_test`/`renderer_invariants_test` siguen la lista `- **PR strategy** (`pr-strategy`)`.
- `TestKilocodeReviewSettingsMatchCurrentMainBaseline`: nuevo hash con comentario de causa (deliberado).

## Riesgos abiertos
- Ninguno nuevo de U3 ni de U4.
- U5: la divergencia aceptada del lector de topología (multirepo cuyo repositorio de especificaciones falta) lee `multirepo` en el plugin y `monorepo-embedded` en el binario; un proyecto así ya falla `Locate` y no es utilizable.
