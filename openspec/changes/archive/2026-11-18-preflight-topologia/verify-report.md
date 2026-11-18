# Informe de verificación: preflight-topologia (C-01-b)

Fase `sdd-verify` diagnóstica (no bloquea el archivo). Rama `inc/preflight-topologia`, base `668a72e8`, HEAD `06cb141e`. Sin cambios de código ni commits en esta fase.

## Veredicto

Cumple. 0 CRITICAL, 1 WARNING, 3 SUGGESTION. Las tareas U0 a U8 están completadas (todas `[x]`); no hay trabajo pendiente.

## Batería de comprobación (ejecutada con `-count=1`)

| Comando | Resultado |
|---|---|
| `go test ./internal/kickoff/... ./internal/workspace/... ./internal/hub/...` | ok (kickoff 1,3 s; workspace 60,8 s; hub 4,5 s) |
| `go test ./internal/cli/ -run 'SDDPreflight\|TestRunSDDKickoff\|TestKickoffSeal'` | ok (14,1 s) |
| `go test ./internal/components/ -run TestGolden` | ok |
| `go test ./internal/sddtaskresult/...` | ok |
| `go test ./internal/assets/ -run 'OpenCode\|SDD'` | ok |

## Requisitos de las deltas

| Requisito | Estado | Evidencia |
|---|---|---|
| REQ-2.2 / REQ-2.6 `workspace-topology` | Cumplido | `internal/workspace/validator.go` (paso 3b, `monorepoRole`, `legacyStoreError`); `TestValidateMonorepoRejectsSpecializedRoles`, `TestValidateMultirepoKeepsSpecializedRoles`, `TestLocateLoadsMonorepoRolesAndLegacyStore`, `TestValidateDeclarationPathsRejectsLegacyArtifactStore` |
| REQ-30.19 `project-technical-context` | Cumplido | `storeErrors` en `validator.go`; mismo test de almacén heredado |
| REQ-2.1, REQ-2.2, REQ-3.3 `multi-project-hub` | Cumplido | `internal/hub/init_validate.go:188`, `init.go:357` (roles detectados solo en multirepo); `TestPlanConfigMonorepoIgnoresDetectedRoles`, `TestPlanConfigMultirepoAppliesDetectedRoles`, `TestValidateInitRejectsExplicitRolesInMonorepo`, `TestPlanConfigKnowledgeProfileForcesMultirepoKnowledge`; dashboard: `projectform.js:122` (`roles: []` en monorepo) y `projects.js` (oculta la sección) |
| REQ-25.36 `sdd-lane-kickoff` | Cumplido | `TestRunSDDKickoffSeal*` (instantánea con el kickoff derivado), `gitsnapshot_test.go` en multirepo |
| REQ-25.4, 25.7, 25.11, 25.18, 27.15 `change-branch-lifecycle`, REQ-25.39, REQ-25.42 | Cumplido (solo pruebas) | Fixtures pasados a `multirepo`/`fullstack` (`planner_test.go`, `verdict_test.go`, `engine/*_test.go`, `branch_context_test.go`); sin código de producción nuevo, como declara el diseño |
| REQ-21.5, 21.6, 28.12, 28.13, 28.19 (sellado) | Cumplido | `internal/kickoff/args.go:312` `ApplyTopology` (multirepo rechaza `continuous` y deriva `checkpointed` con origen `topology`; monorepo solo `fullstack`); llamada en `internal/cli/sdd_kickoff.go:80` antes de `ToKickoff`, `SDDErr` y la guarda de rama; `--infer` exento; pruebas `apply_topology_test.go` y `TestRunSDDKickoffSeal*` |
| REQ-28.15, 28.16, 28.17 (hook) | Cumplido | `sdd_preflight_hook.go:425-433` (tabla de grupos: `pace` solo monorepo, `pr-strategy` ambos, `artifacts` heredado sin topología), `sddPreflightQuestionGroup` (nominal y `N/3`), tabla de campos del bloque; `sdd_preflight_hook_topology_test.go` (12 pruebas) |
| REQ-28.18 (cero grupos) | Cumplido | `sddPreflightDerivedBlock`; `TestSDDPreflightHookZeroRequiredGroupsAllowsDerivedBlock`, `...KeepsChildAndModelDenials` (protecciones de hijo y bloque del modelo intactas) |
| REQ-28.20 (doctrina) | Cumplido | `assets_test.go` (`TestSDDOrchestratorAssetsExecutionModeFollowsTopology`), `claude/sdd-orchestrator-workflow.md:7,96` (`hybrid` fijo), goldens regenerados; ya sin «default to Automatic» |
| REQ-28.21 (plugin OpenCode) | Cumplido | Tablas idénticas en v1 y v2 (`requiredIn`, campos, línea de revisión de 400); bloque `axiom-topology` idéntico (diff vacío); `TestOpenCodeTopologyReaderParity`, `opencode_sdd_preflight_topology_test.go`; digests anteriores registrados |
| REQ-28.22 (topología efectiva) | Cumplido | `locate.go:64` `GoverningTopology`; `TestLocationGoverningTopology`; hook cae a `monorepo-embedded` ante error de `Locate` |
| REQ-28.14 (renombrado) | Cumplido | Pin sin `pace`/`pr-strategy`/`N/3` en el texto de la preferencia de revisión (`assets_test.go`, `session_preflight_test.go`) |

## Criterio de terminado de la sección (líneas 92-99)

1. Multirepo sin pregunta de ritmo ni de almacén, bloque `Pace: interactive`/`Artifact store: hybrid`, sellado `checkpointed`: observado (tabla del hook, texto proyectado, `ApplyTopology`).
2. `continuous` en multirepo rechazado nombrando la topología: observado (`ErrTopologyForbidsContinuous`, mensaje con `multirepo`).
3. Monorepo pregunta ritmo y estrategia de PR, almacén `hybrid`, sellado solo `fullstack`: observado.
4. `axiom workspace validate` rechaza monorepo con roles distintos de `fullstack` con migración: observado (mensaje de `validator.go:136`).
5. Alta monorepo (CLI y dashboard) solo escribe `fullstack`: observado.
6. Specs vivas sin escenarios de monorepo con roles especializados: pendiente hasta `sdd-archive` (ver WARNING 1). Las deltas no conservan ninguno.
7. Baterías de tests en verde: observado para la batería pedida; el cierre completo de apply (`apply-progress.md` U8) también en verde, pero `go test ./internal/cli/...` y `./internal/components/...` enteros no se han ejecutado (restricción del entorno).

## Coherencia

- Paridad hook/plugin: mismas tablas de grupos, marcadores nominal/heredado, campos del bloque y línea de revisión; v1 y v2 idénticas en el lector de topología.
- Texto proyectado (`session_preflight.go`) coincide con lo que exige el hook: marcadores `Axiom SDD preflight pace:`/`pr-strategy:`, etiquetas byte-exactas, sin grupo de almacén, mapeos y política de 400 líneas.
- Ninguna delta conserva escenarios de monorepo con roles especializados como comportamiento soportado; los que los mencionan son de rechazo (`workspace-topology`, REQ-28.19) o del retro-sellado `--infer` exento.

## Hallazgos

### WARNING

1. Las specs vivas (`openspec/specs/change-branch-lifecycle/spec.md:268,294,300,549`, `axiom-sdd-cli-integration/spec.md:90`, `sdd-lane-kickoff/spec.md:202`, `multi-project-hub/spec.md:91`) conservan los escenarios antiguos hasta que el archive fusione las deltas. El criterio 6 solo se cumple tras `sdd-archive`; comprobarlo allí.

### SUGGESTION

1. Divergencia aceptada del lector de topología del plugin (multirepo cuyo repositorio de specs falta: el plugin lee `multirepo`, el binario `monorepo-embedded`); está documentada en el código y el proyecto afectado ya falla `Locate`.
2. `sddPreflightDerivedBlock()` no recibe la topología (desviación menor de D5, documentada en `apply-progress.md`); inocua porque con cero grupos todos los valores son los por defecto.
3. `tasks.md` cita `screens/projects.js` para el dashboard, pero el fichero real es `internal/dashboard/assets/js/projects.js`; corregir la referencia al archivar si se quiere trazabilidad exacta.

## Limitaciones

- Una primera ejecución en segundo plano no produjo salida de las pruebas del hook; se repitió en primer plano con `-count=1` y pasó. No se ejecutaron paquetes enteros de `internal/cli` ni `internal/components`.
- `bun` no está instalado: los arneses del plugin v1 corren con Node (según `apply-progress.md`).
