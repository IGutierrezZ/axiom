# Tareas: preflight y sellado derivados de la topología (preflight-topologia, C-01-b)

> Fase `tasks` · rama `inc/preflight-topologia` · worktree `C:\repos\axiom-wt\preflight-topologia\axiom` · kickoff `continuous`, rol único `fullstack` (`blocking`): un solo `tasks.md`. Entradas: `proposal.md`, `design.md` (autoridad técnica: unidades U1 a U6, D1 a D11), `exploration.md` y las deltas de `specs/` (8 capacidades). Sin `technical_context` mantenido en `axiom.yaml`: no hay `context-impact.md`.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈ 2.200–2.400 de autor (tests y doctrina incluidos; goldens regenerados excluidos) |
| 400-line budget risk | High |
| Chained PRs recommended | No (estrategia `exception-ok`: una sola PR con `size:exception`, aceptada por el usuario) |
| Suggested split | Una sola PR; diez commits (planificación, U1 a U6, fidelidad de ramas, cierre), revertibles por separado |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Nota: `exception-ok` ya recoge la aceptación de `size:exception` por el mantenedor, así que no hay decisión pendiente antes de `sdd-apply`. Nada de PR encadenadas.

### Previsión por unidad (líneas de autor, adiciones más borrados)

| Unidad | Líneas | Nota |
|---|---|---|
| U0 | ≈ 0 de código | Solo artefactos de planificación (Markdown/YAML ya escritos) |
| U1 | ≈ 180 | `workspace`: `GoverningTopology` y reglas del validador |
| U2 | ≈ 370 | `kickoff` + CLI de sellado + migración de pruebas CLI |
| U3 | ≈ 130 | Hub y dashboard: alta monorepo solo `fullstack` |
| U4 | ≈ 600 | Hook: tabla de grupos, marcadores, topología, cuestionario vacío |
| U5 | ≈ 460 | Plugin de OpenCode (dos copias), digests y pruebas bun |
| U6 | ≈ 330 | Texto proyectado, doctrina de todos los runtimes (goldens generados excluidos) |
| U7 | ≈ 120 | Fidelidad de las pruebas de ramas/archive/status a la regla de roles |
| U8 | ≈ 0 | Verificación y cierre |

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| U0 | Commitear los artefactos de planificación del cambio | PR única, commit 1 | `go test ./internal/sddstatus/... -run Tasks` (el parser cuenta las casillas) | `axiom sdd status --cwd C:\repos\axiom-wt\preflight-topologia\axiom --json` ya instalado en el `PATH` (sin compilar binarios): `tasks` detectadas | Revertir el commit: solo documentos |
| U1 | `Location.GoverningTopology()` y reglas del validador (roles de monorepo, almacén `engram`/`none`) | PR única, commit 2 | `go test ./internal/workspace/... -run 'GoverningTopology\|Validate\|Locate'` | `go test` en proceso con `axiom.yaml` temporales | Revertir el commit: el resto no depende de las reglas del validador; U2, U3 y U4 sí dependen de `GoverningTopology` (revertirlos antes) |
| U2 | `ApplyTopology` en `kickoff`, llamada en `runSDDKickoffSeal`, ayuda y migración de pruebas CLI | PR única, commit 3 | `go test ./internal/kickoff/...` y `go test ./internal/cli/ -run 'TestRunSDDKickoff\|TestKickoffSeal\|TestWriter\|TestGate\|TestRunSDDGate'` | `go test` con workspaces temporales reales (monorepo y multirepo), sin binario suelto | Revertir el commit (U1 queda intacta) |
| U3 | El alta del hub y el formulario del dashboard solo escriben `fullstack` en monorepo | PR única, commit 4 | `go test ./internal/hub/... ./internal/dashboard/...` | `go test`: `ValidateInit` rechaza roles explícitos en monorepo; formulario → `roles: []` | Revertir el commit |
| U4 | Hook de preflight: tabla de grupos y campos, marcadores nominales y heredados, topología, cuestionario vacío | PR única, commit 5 | `go test ./internal/cli/ -run 'SDDPreflight'` | `go test` con payloads y transcripciones sintéticas (`workspacetest.Embedded`, `workspacetest.Canonical`) | Revertir el commit: el texto proyectado antiguo sigue siendo válido con el hook antiguo |
| U5 | Plugin de OpenCode (dos copias): contrato de grupos, lector de topología, enlace por etiqueta y digests | PR única, commit 6 | `go test ./internal/sddtaskresult/ ./internal/assets/ -run 'OpenCode'` y `go test ./internal/components/sdd/ -run 'OpenCode\|Digest'` | `bun` vía los arneses de `internal/sddtaskresult` (paridad con `GoverningTopology`) | Revertir el commit (el digest antiguo sigue registrado) |
| U6 | Texto proyectado global, reescrituras de reserva, doctrina de todos los runtimes, contrato de persistencia y goldens | PR única, commit 7 | `go test ./internal/components/sdd/` y `go test ./internal/components/ -run TestGolden` | Goldens regenerados con `-update` y revisados; luego sin `-update` | Revertir el commit (hook y plugin siguen aceptando heredados) |
| U7 | Fidelidad de las pruebas de ramas, archive y status: roles de rama solo en `multirepo`, `fullstack` en monorepo | PR única, commit 8 | `go test ./internal/branch/... ./internal/sddstatus/...` y `go test ./internal/cli/ -run 'TestBranch'` | `go test` con repositorios git temporales | Revertir el commit: solo pruebas |
| U8 | Verificación global y cierre | PR única, commit 9 (solo si hay ajustes) | Batería de cierre (ver Convenciones) | Revisión de `axiom workspace validate` sobre este repositorio (`fullstack`, sin `artifact_store`) en proceso | Sin código propio |

## Convenciones de verificación (valen para todas las unidades)

- Runner: `go test` en proceso, por paquete, desde `C:\repos\axiom-wt\preflight-topologia\axiom`. En `internal/cli` SIEMPRE filtrado con `-run` (el paquete entero no cabe). NUNCA compilar binarios sueltos de `axiom` (`go build -o`, `go install`): Cortex XDR borra `axiom.exe`.
- TDD estricto: cada tarea RED va antes de su GREEN y se observa el rojo antes de implementar.
- Goldens (solo U6): `go test ./internal/components/ -run <Test> -update`, revisar el diff y repetir sin `-update`. Nunca editar goldens a mano.
- Comandos de cierre de cada unidad: `gofmt -l ./internal ./cmd` (salida vacía); `go vet` de los paquetes tocados; `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` (ratchet de negativas) si la unidad toca rechazos; `bash scripts/deadcode-ratchet.sh` si añade o retira funciones.
- Fallos ajenos conocidos en `main` (no bloquean ni cuentan como regresión): `internal/components/filemerge` `TestWriteFileAtomicFollowsDirectoryJunction` y `TestWriteFileAtomicFollowsRelativeDirectoryJunction`; `internal/app` `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`.
- Cada unidad cierra con un commit Conventional en castellano, sin `Co-Authored-By` ni atribución de IA, con sus pruebas y la doctrina o goldens que le correspondan. Anotar la identidad del commit en la tarea de cierre.
- Una ruta citada con `(read-only)` solo se lee.
- Heurística de ~400 líneas por tarea: solo orientativa; no es tope ni motivo de dividir artificialmente.

---

## U0 · Artefactos de planificación

Commit sugerido: `docs(sdd): planificar preflight-topologia (propuesta, specs, diseño y tareas)`

- [x] 0.1 Comprobar que `openspec/changes/preflight-topologia/` contiene `kickoff.yaml`, `exploration.md`, `proposal.md`, `design.md`, `tasks.md` y las ocho deltas de `specs/*/spec.md`, y que `axiom sdd status --cwd C:\repos\axiom-wt\preflight-topologia\axiom --json` cuenta las tareas de `tasks.md`. Verificación: `axiom sdd status --cwd C:\repos\axiom-wt\preflight-topologia\axiom --json` muestra las casillas; `gofmt` no aplica.
- [x] 0.2 Commitear solo esa carpeta: `git -C C:\repos\axiom-wt\preflight-topologia\axiom add -- openspec/changes/preflight-topologia` y commit `docs(sdd): planificar preflight-topologia (propuesta, specs, diseño y tareas)`. Verificación: `git -C C:\repos\axiom-wt\preflight-topologia\axiom status --short` vacío. (Cierre de U0; anotar el commit.)

---

## U1 · `workspace`: topología gobernante y reglas del validador

Satisface REQ-28.22 (base `GoverningTopology`), REQ-2.2 y REQ-2.6 de `workspace-topology`, REQ-30.19 de `project-technical-context`. Diseño: D1, D9.
Commit sugerido: `feat(workspace): topología gobernante y reglas de roles y almacén en la validación`

- [x] 1.1 RED: crear `internal/workspace/governing_topology_test.go` con `TestLocationGoverningTopology` (cargado → su topología; `axiom.yaml` directo no cargable, con topología ausente o desconocida → `monorepo-embedded` y `true`; sin `axiom.yaml`, origen `openspec` o `start` → `("", false)`; puntero roto → `Locate` falla). Cubre REQ-28.22. Verificación: `go test ./internal/workspace/ -run GoverningTopology` falla (no compila).
- [x] 1.2 GREEN: añadir `Location.GoverningTopology() (TopologyType, bool)` en `internal/workspace/locate.go` según la tabla de D1. Verificación: `go test ./internal/workspace/ -run GoverningTopology`.
- [x] 1.3 RED: en `internal/workspace/validator_test.go` añadir `TestValidateMonorepoRejectsSpecializedRoles` (`core`+`qa`, `fullstack`+`web`: un error en español que nombra la topología, los roles y la migración; `fullstack` solo es conforme), `TestValidateMultirepoKeepsSpecializedRoles` y `TestLocateLoadsMonorepoRolesAndLegacyStore` (la carga no falla con roles especializados ni con `artifact_store: engram`). Cubre REQ-2.2 y REQ-2.6 (no rotura de la carga). Verificación: `go test ./internal/workspace/ -run 'ValidateMonorepoRejects|ValidateMultirepoKeeps|LocateLoadsMonorepoRoles'` en rojo.
- [x] 1.4 GREEN: añadir en `internal/workspace/validator.go` el paso 3b de `Validate` (monorepo con claves de `roles` distintas de `fullstack`, claves ordenadas, mensaje de D9). Verificación: las pruebas de 1.3 sobre roles en verde.
- [x] 1.5 RED: añadir `TestValidateDeclarationPathsRejectsLegacyArtifactStore` en `internal/workspace/validator_test.go` (`engram` y `none` sin contexto mantenido → error que nombra `artifact_store`, el valor, la causa y la migración; `openspec`, `hybrid` y sin declarar → sin error; con contexto mantenido el error nombra también `technical_context.maintained`) y adaptar los casos existentes de `validator_test.go:468-522` para que el almacén `engram`/`none` sin contexto mantenido ya no se acepte. Cubre REQ-2.6 y REQ-30.19. Verificación: `go test ./internal/workspace/ -run 'LegacyArtifactStore|DeclarationPaths'` en rojo.
- [x] 1.6 GREEN: ampliar `storeErrors` en `internal/workspace/validator.go` con el error incondicional de D9 (sin tocar `DeclaredArtifactStore` ni `sddstatus`). Verificación: `go test ./internal/workspace/...` en verde.
- [x] 1.7 Cierre de U1: `gofmt -l ./internal/workspace`, `go vet ./internal/workspace/...`, `go test ./internal/workspace/... ./internal/sddstatus/...` (el status no debe cambiar) y commit `feat(workspace): topología gobernante y reglas de roles y almacén en la validación`. Anotar el commit. Commit: `37da0abf`.

---

## U2 · Sellado del kickoff por topología

Satisface REQ-28.19, REQ-28.22 (sellado), REQ-21.5, REQ-21.6, REQ-28.12, REQ-28.13. Depende de U1. Diseño: D8.
Commit sugerido: `feat(kickoff): el sellado aplica la topología al estilo de ejecución y a los roles`

- [x] 2.1 RED: en `internal/kickoff/args_test.go` añadir `TestSealArgsApplyTopologyMultirepo` (sin estilo deriva `checkpointed` con `ExecutionStyleSource` `topology`; `continuous` explícito, `--from-session-pace auto` y estilo explícito sobre el ritmo → `ErrTopologyForbidsContinuous` con mensaje que nombra `multirepo`; `checkpointed` e `interactive` aceptados con `core`+`web`), `TestSealArgsApplyTopologyMonorepo` (rol distinto de `fullstack`, también con `fullstack` + `qa` → `ErrTopologyRequiresFullstack` con la topología y los roles; sin `--role` sella `fullstack`; `continuous` con `fullstack` y política `deferred` se acepta), `TestSealArgsApplyTopologyPrecedence` (monorepo + `continuous` + `core` → gana el mensaje de topología) y `TestSealArgsApplyTopologyWithoutTopology` (`""`: estilo obligatorio REQ-21.5 y regla de roster REQ-28.12/28.13 intactas). Cubre REQ-28.19, REQ-28.22, REQ-28.12, REQ-28.13, REQ-21.5. Verificación: `go test ./internal/kickoff/ -run 'ApplyTopology'` en rojo.
- [x] 2.2 GREEN: en `internal/kickoff/args.go` hacer que `ParseSealArgs` deje de rechazar la ausencia de `--execution-style`/`--from-session-pace` y de aplicar `checkContinuousRoster`; añadir `(*SealArgs).ApplyTopology`, `ErrTopologyForbidsContinuous` y `ErrTopologyRequiresFullstack` según D8 (orden: topología, estilo obligatorio ampliado, red de seguridad) y actualizar el comentario de `ExecutionStyleSource`. Verificación: `go test ./internal/kickoff/ -run 'ApplyTopology'`.
- [x] 2.3 Adaptar `TestParseSealArgsRequiresExecutionStyleOrSessionPace` y `TestParseSealArgsContinuousRosterRule` en `internal/kickoff/args_test.go` para que ejerzan la regla vía `ApplyTopology("")`, y confirmar que `--infer`, `--docs-publication` solo con `continuous` y el resto de la validación sintáctica de `ParseSealArgs` no cambian. Cubre REQ-21.5, REQ-21.6, REQ-28.12, REQ-28.13. Verificación: `go test ./internal/kickoff/...` en verde.
- [x] 2.4 RED (integración CLI): en `internal/cli/sdd_kickoff_test.go` añadir `TestRunSDDKickoffSealMultirepoDerivesCheckpointed`, `TestRunSDDKickoffSealMultirepoRejectsContinuous`, `TestRunSDDKickoffSealMonorepoRejectsRolesBeforeBranchGuard` (el rechazo no escribe `kickoff.yaml` ni emite avisos de rama ni ejecuta `applySealBranchPolicy`), `TestRunSDDKickoffSealInferIsExemptFromTopology` y un caso de `axiom.yaml` con topología desconocida que aplica la regla de monorepo, con fixtures reales (`workspacetest.Embedded`, `workspacetest.Canonical`). Cubre REQ-28.19, REQ-28.22 y REQ-25.36 (la instantánea se escribe con el kickoff derivado). Verificación: `go test ./internal/cli/ -run 'TestRunSDDKickoffSeal'` en rojo.
- [x] 2.5 GREEN: en `internal/cli/sdd_kickoff.go` llamar a `parsed.ApplyTopology(topology)` con `loc.GoverningTopology()` en la rama sin `--infer`, antes de `ToKickoff`, de `SDDErr` y de `applySealBranchPolicy`; actualizar la ayuda (`:38-39`) con la regla de topología. Verificación: `go test ./internal/cli/ -run 'TestRunSDDKickoff'`.
- [x] 2.6 Migrar los sellados de monorepo con `core`/`qa` en `internal/cli/sdd_kickoff_branch_test.go` (líneas ~156, 242, 293) y `internal/cli/sdd_writer_guard_test.go` (`guardProjectSealedWith`) al ayudante de prueba `sealLegacyKickoff(t, ws, args...)` (replica `runSDDKickoffSeal` sin `ApplyTopology`; documenta que simula un kickoff sellado antes de C-01-b), y hacer que `TestKickoffSealAcceptsTheRoleBranchesOfTheRolesBeingSealed` afirme el rechazo por topología. Los sellados sin `axiom.yaml` (`sdd_gate*_test.go`, `newGovernanceWorkspace`) y los de multirepo no cambian. Cubre REQ-25.36, REQ-28.19. Verificación: `go test ./internal/cli/ -run 'TestKickoffSeal|TestWriter|TestGate|TestRunSDDGate'`.
- [x] 2.7 Cierre de U2: `gofmt -l ./internal/kickoff ./internal/cli`, `go vet ./internal/kickoff/... ./internal/cli/`, `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`, `bash scripts/deadcode-ratchet.sh` y commit `feat(kickoff): el sellado aplica la topología al estilo de ejecución y a los roles`. Anotar el commit. Commit: `9383953d`.

---

## U3 · Alta del proyecto (hub y dashboard)

Satisface REQ-2.1, REQ-2.2 y REQ-3.3 de `multi-project-hub`. Depende de U1 (D9 rechaza los roles explícitos). Diseño: D10.
Commit sugerido: `feat(hub): el alta en monorepo-embedded escribe solo el rol fullstack`

- [x] 3.1 RED: en `internal/hub/init_validate_test.go`, `internal/hub/init_test.go` y `internal/hub/init_multirepo_test.go` añadir `TestPlanConfigMonorepoIgnoresDetectedRoles` (`go.mod`+`package.json` → solo `fullstack`; `axiom workspace validate` sin errores de roles), `TestPlanConfigMultirepoAppliesDetectedRoles`, `TestValidateInitRejectsExplicitRolesInMonorepo` y `TestPlanConfigKnowledgeProfileForcesMultirepoKnowledge`; adaptar las expectativas de roles por topología en `internal/hub/hub_test.go`. Cubre REQ-2.1 y REQ-2.2 (hub). Verificación: `go test ./internal/hub/ -run 'PlanConfig|ValidateInit'` en rojo.
- [x] 3.2 GREEN: en `internal/hub/init_validate.go` (`planConfig`) y `internal/hub/init.go` (`buildAxiomYaml`) aplicar los roles detectados (`tech.ConfiguredRoles`) solo si la topología normalizada es `multirepo`; el perfil `knowledge`/`spec-only` sigue forzando `multirepo` con `knowledge`; los roles explícitos se conservan y los rechaza `ValidateInit` por D9. `detector.go` no cambia. Verificación: `go test ./internal/hub/...`.
- [x] 3.3 RED: en `internal/dashboard/assets_projectform_test.go` añadir a `TestProjectFormValidationRuntime` los casos de REQ-3.3: `monorepo-embedded` → `roles: []` sin violaciones de roles; `multirepo` conserva los roles del formulario; cambiar de `multirepo` a `monorepo-embedded` con `core` y `qa` definidos devuelve `roles: []`. Verificación: `go test ./internal/dashboard/ -run ProjectForm` en rojo.
- [x] 3.4 GREEN: en `internal/dashboard/assets/js/core/projectform.js` hacer que `validateProjectForm` devuelva `roles: []` sin violaciones en `monorepo-embedded`, y en `internal/dashboard/assets/js/screens/projects.js` ocultar `#roles-builder-container` cuando `#select-project-topology` vale `monorepo-embedded` (mostrarlo en `multirepo`, también al cambiar la topología). `buildRoles` no cambia. Verificación: `go test ./internal/dashboard/...`.
- [x] 3.5 Cierre de U3: `gofmt -l ./internal/hub ./internal/dashboard`, `go vet ./internal/hub/... ./internal/dashboard/...`, `go test ./internal/hub/... ./internal/dashboard/...` y commit `feat(hub): el alta en monorepo-embedded escribe solo el rol fullstack`. Anotar el commit. Commit: `494049a5`.

---

## U4 · Hook de preflight por topología

Satisface REQ-28.15, REQ-28.16, REQ-28.17, REQ-28.18, REQ-28.22 (hook), REQ-28.14 (renombrado: el hook ignora la interacción de preferencia). Depende de U1. Diseño: D2, D3, D4, D5.
Commit sugerido: `feat(cli): el hook de preflight valida los grupos que exige la topología`

Amenazas aplicables de la matriz de diseño: selección de repositorio (cwd relativo, absoluto, ausente, puntero roto) y autoridad de despacho (hijo, bloque del modelo, transcripción no ligada, cuestionario vacío). Las filas N/A se omiten.

- [x] 4.1 RED (modelo de datos): en `internal/cli/sdd_preflight_hook_test.go` añadir `TestSDDPreflightHookRequiredGroupsByTopology` (`monorepo-embedded` → `pace`,`pr-strategy`; `multirepo` → `pr-strategy`; `cwd` multirepo, monorepo, sin `cwd`, puntero roto → grupos de monorepo; error de `Locate` nunca falla abierto) y `TestValidSDDPreflightHookBlockFollowsFieldTable` (el bloque tiene `3 + len(fields)` líneas; `Artifact store` distinto de `hybrid` es inválido). Cubre REQ-28.15, REQ-28.22 y la amenaza «selección de repositorio». Verificación: `go test ./internal/cli/ -run 'SDDPreflightHookRequiredGroupsByTopology|ValidSDDPreflightHookBlockFollowsFieldTable'` en rojo.
- [x] 4.2 GREEN: en `internal/cli/sdd_preflight_hook.go` crear `sddPreflightGroup`, `sddPreflightGroups`, `sddPreflightBlockField`, `sddPreflightBlockFields`, `sddPreflightRequiredGroups`, `sddPreflightHookTopology`, `sddPreflightBuildBlock` y `validSDDPreflightHookBlock` contra la tabla de campos; el bloque conserva los seis rótulos y `Delivery strategy` por defecto `single-pr` (D2, D3). Verificación: las pruebas de 4.1 en verde.
- [x] 4.3 RED (marcadores y validación por significado): añadir `TestSDDPreflightHookAcceptsNominalMarkers` (ambos prefijos), `TestSDDPreflightHookAcceptsLegacyNumberedMarkersAndIgnoresArtifacts` (monorepo con `1/3`,`2/3`,`3/3`; almacén `Engram` ignorado, bloque `hybrid`), `TestSDDPreflightHookMultirepoAsksOnlyPRStrategy`, `TestSDDPreflightHookMultirepoIgnoresAnsweredPace` (ritmo `Automatic` contestado → `Pace: interactive`), `TestSDDPreflightHookDeniesMissingRequiredGroup` (causa que nombra `pace`), `TestSDDPreflightHookDeniesGroupAskedTwice` (nominal+heredado), `TestSDDPreflightHookIgnoresUnknownNamedGroup` y la prueba de la interacción de preferencia ignorada (REQ-28.14); adaptar `…AllowsCorroboratedCanonicalPreflight` (espera `hybrid`), `…RejectsMalformedMarkerUnderEveryPrefix`, `…BindsAnswersByLabelNotPosition`, `…DenialNamesTheCause`, `…DeniesOutsideDomainAnswer` y `…MultiSelectAnswer`. Cubre REQ-28.15, REQ-28.16, REQ-28.17, REQ-28.22. Verificación: `go test ./internal/cli/ -run 'SDDPreflightHook'` en rojo.
- [x] 4.4 GREEN: implementar `sddPreflightQuestionGroup`, `resolveSDDPreflightHookBlock(questions, answers, topology)` (sin el error «must ask exactly N questions», con los mensajes de la tabla de D4), `resolveSDDPreflightTranscriptAuthority(path, sessionID, topology)` y `sddPreflightHookMissingAuthorityReason(rejection, topology)`; grupos extra, heredados y marcadores desconocidos se ignoran. Verificación: `go test ./internal/cli/ -run 'SDDPreflightHook'`.
- [x] 4.5 RED (cuestionario vacío): añadir `TestSDDPreflightHookZeroRequiredGroupsAllowsDerivedBlock` (sustituyendo `sddPreflightGroups` por una copia sin `multirepo` en `RequiredIn` de `pr-strategy`, con `t.Cleanup` y sin `t.Parallel`: permite sin transcripción y antepone el bloque con `Delivery strategy: single-pr`; un prompt que ya trae el bloque no se duplica) y `TestSDDPreflightHookZeroRequiredGroupsKeepsChildAndModelDenials` (hijo y bloque escrito por el modelo siguen denegados con cero grupos); confirmar que las pruebas actuales de hijo, bloque del modelo y transcripción no ligada siguen en verde con grupos requeridos. Cubre REQ-28.18, REQ-28.17 y la amenaza «autoridad de despacho». Verificación: `go test ./internal/cli/ -run 'ZeroRequiredGroups'` en rojo.
- [x] 4.6 GREEN (cuestionario vacío): reordenar `runSDDPreflightHook` según D5 (denegaciones de hijo y bloque del modelo siempre; topología y grupos requeridos; con cero grupos `sddPreflightDerivedBlock(topology)` validado y despacho sin transcripción; con ≥1 grupo el flujo actual) y añadir `sddPreflightDerivedBlock`. Verificación: `go test ./internal/cli/ -run 'SDDPreflight'`.
- [x] 4.7 Cierre de U4: `gofmt -l ./internal/cli`, `go vet ./internal/cli/`, `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`, `bash scripts/deadcode-ratchet.sh` y commit `feat(cli): el hook de preflight valida los grupos que exige la topología`. Anotar el commit. Commit: `9b53f58f`.

---

## U5 · Plugin de OpenCode (dos copias)

Satisface REQ-28.21, REQ-28.15, REQ-28.16, REQ-28.17, REQ-28.18 y REQ-28.22 (plugin). Depende de U4 (contrato). Diseño: D7.
Commit sugerido: `feat(opencode): el plugin aplica el contrato de grupos por topología del hook`

- [x] 5.1 Registrar el digest previo antes de editar: en `internal/components/sdd/opencode_plugin_digests.go` añadir el sha256 de `git show HEAD:internal/assets/opencode/plugins/sdd-task-result-artifacts.ts` (lista `"v1"`) y de `git show HEAD:internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts` (lista `"v2"`), con el commit corto en el comentario. Cubre REQ-28.21 (revisión anterior registrada). Verificación: `go test ./internal/components/sdd/ -run 'Digest'`.
- [x] 5.2 RED: crear `internal/sddtaskresult/topology_reader_parity_test.go` con `TestOpenCodeTopologyReaderParity` (estilo de bloque, comillas, comentarios, puntero a otro `axiom.yaml`, valor desconocido y sin fichero frente a `GoverningTopology`; cwd relativo y puntero roto → `monorepo-embedded`) y adaptar a los comportamientos nuevos `internal/assets/opencode_v2_sdd_test.go` (`TestOpenCodeV2SDDDispatch`, `TestOpenCodeV2SDDPreflightAcceptsBothPrefixes`), `internal/assets/opencode_sdd_project_gate_test.go` (`TestOpenCodePreflightQuestionCanonicalizationAndTolerantMatching`, el escenario A pierde la asignación posicional), `internal/sddtaskresult/preflight_question_harness_test.go`, `internal/sddtaskresult/transport_parity_test.go`, `internal/components/sdd/opencode_runtime_test.go` y `internal/components/sdd/sdd_task_result_transport_test.go`: marcadores nominales, número variable de grupos, `multirepo` con solo `pr-strategy`, `monorepo-embedded` con solo `pr-strategy` denegado nombrando `pace`, heredados `1/3` a `3/3`, opciones reordenadas enlazadas por etiqueta, grupo `artifacts` heredado ignorado (`Artifact store: hybrid`), falta de grupo requerido y cero grupos (fixture de tabla) que antepone el bloque derivado. Cubre REQ-28.21, REQ-28.22. Verificación: `go test ./internal/sddtaskresult/ ./internal/assets/ -run 'OpenCode'` en rojo.
- [x] 5.3 GREEN (v2): en `internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts` añadir el bloque delimitado `// axiom-topology:begin … // axiom-topology:end` con `findAxiomConfig` y `readAxiomTopology`, `SDD_PREFLIGHT_GROUPS`/`SDD_PREFLIGHT_BLOCK_FIELDS`, marcadores nominal y heredado, `canonicalizeSDDPreflightQuestions` sin exigir tres preguntas ni asignar marcadores por posición, `sddPreflightBlock(args, metadata, topology)` con enlace etiqueta → token y los mismos motivos de rechazo que el hook, y `execute.before` que antepone el bloque derivado con cero grupos conservando las denegaciones de sesión hija y de bloque del modelo; el bloque `axiom-project-gate` no se toca. Verificación: `go test ./internal/sddtaskresult/ ./internal/assets/ -run 'OpenCodeV2|TopologyReaderParity'`.
- [x] 5.4 GREEN (v1): replicar el mismo contrato en `internal/assets/opencode/plugins/sdd-task-result-artifacts.ts`, conservando su normalización de respuestas actual (v1 con NFD). Verificación: `go test ./internal/sddtaskresult/ ./internal/assets/ -run 'OpenCode'`.
- [x] 5.5 Cierre de U5: `go test ./internal/sddtaskresult/ ./internal/assets/ -run 'OpenCode'`, `go test ./internal/components/sdd/ -run 'OpenCode|Digest'`, `gofmt -l ./internal` y commit `feat(opencode): el plugin aplica el contrato de grupos por topología del hook`. Anotar el commit. Commit: `43094d2d`.

---

## U6 · Texto proyectado, doctrina y goldens

Satisface REQ-28.15 (texto global y pin de lectura de topología), REQ-28.14 (renombrado y pin sin marcadores del preflight), REQ-28.18 (el orquestador no lanza la herramienta con cero grupos), REQ-28.20, REQ-21.5, REQ-21.6. Depende de U4. Diseño: D6, D11.
Commit sugerido: `feat(sdd): el texto y la doctrina fijan el almacén hybrid y el ritmo por topología`

- [x] 6.1 RED: en `internal/components/sdd/session_preflight_test.go` añadir `TestSessionPreflightBodyNamesTopologyGroups` (ordena leer `workspace.topology`, declara los grupos de cada topología, marcadores nominales `pace` y `pr-strategy`, sin grupo ni mapeos de Artifacts ni `Both -> hybrid`, frase de cuestionario vacío, política de revisión fija de 400 líneas) y `TestSessionPreflightFallbackAnswerSyntaxUsesRequiredGroups` (`Pace: <label>; PR strategy: <label>`), y adaptar los literales de `session_preflight_test.go:25-264`; adaptar los literales de doctrina en `orchestrator_composition_test.go`, `renderer_invariants_test.go`, `inject_test.go`, `orchestrator_shared_sections_test.go`, `persistence_recovery_test.go`, `delivery_strategy_vocabulary_test.go`, `research_state_contract_test.go`, `review_ledger_contract_test.go`, `internal/assets/assets_test.go` e `internal/assets/independent_review_boundary_test.go` (sin «default to Automatic», almacén `hybrid` fijo, idéntico en todos los runtimes, pin de REQ-28.14 sin `Axiom SDD preflight`/`pace`/`pr-strategy`/`N/3`). Cubre REQ-28.15, REQ-28.14, REQ-28.18, REQ-28.20. Verificación: `go test ./internal/components/sdd/ ./internal/assets/ -run 'SessionPreflight|Doctrine|Orchestrator|Persistence|Vocabulary|Contract'` en rojo.
- [x] 6.2 GREEN (texto): reescribir `sddSessionPreflightBody` y la variante sin UI (`sddSessionPreflightBlockWithTool("")`, nuevo ancla `"\n\n- **Pace** (`pace`"`, sintaxis solo con los grupos requeridos, resumen de mapeos con los valores fijados por la topología) en `internal/components/sdd/session_preflight.go`; conservar el rechazo específico de `Both -> both` en `validateSDDSessionPreflightProjection`. `inject.go` no cambia estructuralmente. Verificación: `go test ./internal/components/sdd/ -run 'SessionPreflight|Inject'`.
- [x] 6.3 GREEN (doctrina compartida): actualizar «Execution Mode» y «Artifact Store Mode» y la frase de kickoff en `internal/assets/claude/sdd-orchestrator-workflow.md` y `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, y la sección «Mode Resolution» de `internal/assets/skills/_shared/persistence-contract.md`, con los textos exactos de D11 y «Interfaces / Contratos» (la apertura de «Execution Mode» conserva el prefijo «When the user invokes `/sdd-new`»). Cubre REQ-28.20, REQ-21.5, REQ-21.6. Verificación: `go test ./internal/components/sdd/ -run 'Shared|Persistence|Composition'`.
- [x] 6.4 GREEN (runtimes): aplicar las mismas dos secciones en `internal/assets/codex/sdd-orchestrator.md`, `internal/assets/antigravity/sdd-orchestrator.md`, `internal/assets/windsurf/sdd-orchestrator.md`, `internal/assets/cursor/sdd-orchestrator.md`, `internal/assets/kiro/sdd-orchestrator.md`, `internal/assets/kimi/sdd-orchestrator.md`, `internal/assets/opencode/sdd-orchestrator.md` y `internal/assets/hermes/sdd-orchestrator.md`; revisar `generic`, `gemini` y `qwen` y tocar solo los que contengan el texto. Verificación: `go test ./internal/assets/...`.
- [x] 6.5 GREEN (reescrituras de reserva): en `internal/components/sdd/orchestrator.go` actualizar las reescrituras de `:148-223` («### Artifact Store Policy», «### Artifact Store Mode», Copilot, Codex y búsqueda de init) y la cota de inicio de `composeFallbackSessionPreflight` (`:170-180`) a la frase exacta nueva, en el mismo commit para que no entre en pánico. Verificación: `go test ./internal/components/sdd/ -run 'Fallback|Composition|Orchestrator'`.
- [x] 6.6 Regenerar goldens: `go test ./internal/components/ -run TestGolden -update`, revisar el diff completo (los 11 goldens de la exploración más `sdd-claude-claudemd.golden` y `combined-claude-claudemd.golden`) y repetir `go test ./internal/components/ -run TestGolden` sin `-update`. Cubre REQ-28.20 («todos los runtimes proyectan el mismo texto»). Verificación: ambas ejecuciones en verde.
- [x] 6.7 Cierre de U6: `gofmt -l ./internal`, `go vet ./internal/components/...`, `go test ./internal/components/sdd/ ./internal/assets/...`, `go test ./internal/app/ -run 'Documented|Invocation'` y commit `feat(sdd): el texto y la doctrina fijan el almacén hybrid y el ritmo por topología`. Anotar el commit. Commit: ver `git log` (mensaje `feat(sdd): el texto y la doctrina fijan el almacén hybrid y el ritmo por topología`).

---

## U7 · Fidelidad de ramas, archive y status a la regla de roles

Satisface REQ-25.4, REQ-25.7, REQ-25.11, REQ-25.18, REQ-27.15 (`change-branch-lifecycle`), REQ-25.39 (`sdd-archive-lifecycle`), REQ-25.42 (`axiom-sdd-cli-integration`) y REQ-25.36 (`sdd-lane-kickoff`). Estas deltas no cambian código de producción: la carga sigue aceptando monorepos con roles, así que las pruebas actuales pasan, pero los escenarios pasan a `multirepo` o a `fullstack` y las pruebas deben reflejarlo. Depende de U1.
Commit sugerido: `test(branch): los escenarios de ramas de rol y archive usan multirepo o fullstack`

- [x] 7.1 (alcance acotado por el orquestador: solo las pruebas que dicen explícitamente probar un escenario de monorepo con ramas de rol; detalle en `apply-progress.md`) Localizar con `rg -n "feat/x-core|x-core|feat/x-qa" internal/branch internal/sddstatus internal/kickoff internal/cli/branch_test.go` los fixtures de monorepo con roles `core`/`qa` y reclasificarlos: los escenarios de ramas de rol, de seguimiento, de `ensure` previo/posterior al sellado, de ruta con sufijo de rol, de adopción de rama de rol (`ensure_adopt_test.go`), de veredicto con rama de rol (`verdict_test.go`, `engine/verdict_test.go`) y de snapshot (`planner_snapshot_test.go`, `kickoff/gitsnapshot_test.go`) pasan a un fixture `multirepo`; los de monorepo pasan al rol `fullstack` (rama única, sin ramas de rol, sin rama de rol en el plan de `status`). Cubre REQ-25.4, REQ-25.7, REQ-25.11, REQ-25.18, REQ-27.15, REQ-25.36. Verificación: `go test ./internal/branch/... ./internal/kickoff/...`.
- [x] 7.2 RED→GREEN: añadir o adaptar los escenarios con nombre propio: monorepo `fullstack` → solo `feat/x` y destino de PR de `feat/x` y de `feat/x-archive` a la integración sin rama de rol (REQ-25.4); `ensure` en monorepo antes y después de sellar `fullstack` no añade ramas (REQ-25.7); `archive` en rama ajena `feat/otra` → `blocked(wrong_branch)` con esperada `feat/x-archive` y continuación `ensure --archive` en `internal/branch/engine/status_archive_test.go`, `internal/branch/engine/archive_test.go`, `internal/branch/verdict_test.go` e `internal/sddstatus/branch_context_test.go` (REQ-25.39, REQ-25.42, REQ-25.18); en el repositorio de specs de `multirepo` con rama `feat/x-core` → `wrong_branch` con esperada `main` y sin continuación de `--archive`. Verificación: `go test ./internal/branch/... ./internal/sddstatus/ -run 'Verdict|Archive|BranchContext|Status'`.
- [x] 7.3 Cierre de U7: `gofmt -l ./internal/branch ./internal/sddstatus ./internal/kickoff`, `go test ./internal/branch/... ./internal/sddstatus/... ./internal/kickoff/...`, `go test ./internal/cli/ -run 'TestBranch'` y commit `test(branch): los escenarios de ramas de rol y archive usan multirepo o fullstack`. Anotar el commit.

---

## U8 · Verificación global y cierre

Satisface todos los requisitos (verificación transversal). Depende de U1 a U7.

- [ ] 8.1 Batería de cierre completa sin binarios sueltos: `go test ./internal/workspace/... ./internal/kickoff/... ./internal/hub/... ./internal/dashboard/... ./internal/branch/... ./internal/sddstatus/... ./internal/sddtaskresult/... ./internal/assets/... ./internal/components/sdd/...`; `go test ./internal/components/ -run TestGolden`; `go test ./internal/cli/ -run 'SDDPreflight|TestRunSDDKickoff|TestKickoffSeal|TestWriter|TestGate|TestRunSDDGate|TestBranch'`; `go test ./internal/app/ -run 'Documented|Invocation'`. Los únicos fallos admitidos son los ajenos conocidos de `main` (junctions de `filemerge` y `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`).
- [ ] 8.2 Comprobar la conformidad de este repositorio con las reglas nuevas: `go test ./internal/workspace/ -run 'Validate'` más una comprobación en proceso de que el `axiom.yaml` del repositorio (solo `fullstack`, sin `artifact_store`) sigue siendo conforme; documentar el resultado en la evidencia.
- [ ] 8.3 Revisar `git -C C:\repos\axiom-wt\preflight-topologia\axiom log --oneline origin/main..HEAD` (un commit convencional en castellano por unidad, sin atribución de IA), anotar los commits de U0 a U7 en las tareas de cierre y dejar listos los datos de la PR única con `size:exception` (el empuje y la apertura de la PR son del usuario).

---

## Matriz requisito → tarea

| Requisito | Capacidad | Tareas |
|---|---|---|
| REQ-2.2 (roles de monorepo) | `workspace-topology` | 1.3, 1.4, 7.1, 8.2 |
| REQ-2.6 (`engram`/`none` reportados) | `workspace-topology` | 1.3, 1.5, 1.6 |
| REQ-30.19 (contexto mantenido y almacén) | `project-technical-context` | 1.5, 1.6 |
| REQ-2.1 (detección y roles solo en multirepo) | `multi-project-hub` | 3.1, 3.2 |
| REQ-2.2 (andamiaje `fullstack` en monorepo) | `multi-project-hub` | 3.1, 3.2 |
| REQ-3.3 (formulario del dashboard) | `multi-project-hub` | 3.3, 3.4 |
| REQ-25.36 (instantánea del kickoff) | `sdd-lane-kickoff` | 2.4, 2.6, 7.1 |
| REQ-25.4 (modelo de ramas por roles) | `change-branch-lifecycle` | 7.1, 7.2 |
| REQ-25.7 (`ensure` determinista e idempotente) | `change-branch-lifecycle` | 7.1, 7.2 |
| REQ-25.11 (rutas de worktree) | `change-branch-lifecycle` | 7.1 |
| REQ-25.18 (veredicto de rama) | `change-branch-lifecycle` | 7.1, 7.2 |
| REQ-27.15 (adopción de la rama publicada) | `change-branch-lifecycle` | 7.1 |
| REQ-25.39 (archive en rama de archivado) | `sdd-archive-lifecycle` | 7.2 |
| REQ-25.42 (`blocked(wrong_branch)` en status) | `axiom-sdd-cli-integration` | 7.2 |
| REQ-21.5 (modalidad sellada) | `sdd-preflight-configuration` | 2.1, 2.3, 6.3 |
| REQ-21.6 (rol `fullstack`) | `sdd-preflight-configuration` | 2.1, 2.3, 6.3 |
| REQ-28.12 (`continuous` solo con `fullstack`) | `sdd-preflight-configuration` | 2.1, 2.3 |
| REQ-28.13 (`--from-session-pace auto`) | `sdd-preflight-configuration` | 2.1, 2.3 |
| REQ-28.14 (renombrado: preferencia sin reutilizar marcadores) | `sdd-preflight-configuration` | 4.3, 6.1 |
| REQ-28.15 (grupos por topología) | `sdd-preflight-configuration` | 4.1, 4.2, 4.3, 5.2, 6.1, 6.2 |
| REQ-28.16 (marcadores nominales y heredados) | `sdd-preflight-configuration` | 4.3, 4.4, 5.2 |
| REQ-28.17 (el hook valida el significado) | `sdd-preflight-configuration` | 4.3, 4.4, 4.5, 4.6 |
| REQ-28.18 (cuestionario vacío) | `sdd-preflight-configuration` | 4.5, 4.6, 5.2, 6.1 |
| REQ-28.19 (sellado por topología) | `sdd-preflight-configuration` | 2.1, 2.2, 2.4, 2.5, 2.6 |
| REQ-28.20 (doctrina `hybrid` y ritmo por topología) | `sdd-preflight-configuration` | 6.1, 6.3, 6.4, 6.5, 6.6 |
| REQ-28.21 (plugin de OpenCode) | `sdd-preflight-configuration` | 5.1, 5.2, 5.3, 5.4 |
| REQ-28.22 (topología efectiva) | `sdd-preflight-configuration` | 1.1, 1.2, 2.1, 2.4, 4.1, 5.2 |

Dependencias entre unidades: U1 → U2, U3, U4, U7; U4 → U5, U6; U8 cierra tras U7. U2, U3 y U7 son independientes entre sí; el orden de commits propuesto es U0, U1, U2, U3, U4, U5, U6, U7, U8, con U4 antes de U6 para que ningún commit proyecte marcadores nominales que el hook aún no acepte.
