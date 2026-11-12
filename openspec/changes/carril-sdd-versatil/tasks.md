# Tareas: Evidencias ISO y correcciones del SDD (carril-sdd-versatil)

> Fase `tasks` · rol único `fullstack` (`blocking`) · rama `inc/carril-sdd-versatil` · 2026-11-12. Entradas: `proposal.md`, `specs/*/spec.md` (REQ-29.1 a REQ-29.19, más los MODIFIED REQ-21.13, REQ-21.15, REQ-24.38, REQ-28.3, REQ-28.4 y REQ-28.6) y `design.md` (aprobado, con H1 a H6 aplicados).
> La **Review Workload Forecast** está al final del documento.

## Convenciones del plan

- **Formato de cada tarea.** Una línea de casilla con los ficheros que toca; debajo, `Cubre: REQ-x`, el criterio de aceptación (`Aceptación`) y la comprobación exacta (`Comprobación`). Este cambio no es ISO, pero usa el formato que introduce.
- **TDD.** Cada tarea con comportamiento escribe antes la prueba en rojo (RED) y después el código (GREEN) en la misma tarea y el mismo commit. Las filas aplicables de la matriz de amenazas (design §8) tienen una tarea RED propia antes de la tarea de producción.
- **Comprobaciones.** Solo `go test ./internal/<paquete>/ -run <Patrón>`, acotado y en proceso. **Nunca** `go build`, `go install` ni `go run`: Cortex XDR borra `axiom.exe`.
- **Commits.** Una unidad de commit por grupo de tareas (U1 a U24), con mensaje Conventional Commit en castellano, sin `Co-Authored-By` ni atribución de IA. Las unidades marcadas **«solape P6»** son pequeñas y van aisladas para que quien rebase (este cambio o `spec-viva-openspec`) lo haga mecánicamente.
- **Goldens.** Se regeneran solo con `-update` (`go test ./internal/components/ -run TestGolden -update`), nunca a mano, y en el mismo commit que cambia la prosa que los afecta, para que ningún commit deje rojas las pruebas de golden. La unidad final (7.8) hace una pasada sin `-update`.
- **Fuera del plan.** No se tocan `openspec/specs/`, `internal/specbaseline/`, `internal/multirole/barrier.go`, `internal/branch/` (salvo la línea de inventario de `internal/branch/engine/writer_inventory_test.go`) ni `internal/workspace/workflow.go`.
- **Rutas de solo lectura.** Los ficheros que una tarea solo consulta llevan `(read-only)` tras la ruta.

---

## Fase 1: paquete hoja `internal/isoevidence` (U1 a U4)

Sin dependencias internas; todo son funciones puras con pruebas por tablas.

### U1 · `feat(isoevidence): modelo, identificadores REQ y escenarios`

- [x] 1.1 Crear `internal/isoevidence/doc.go`, `internal/isoevidence/model.go` y `internal/isoevidence/imports_test.go`: tipos `Section`, `Requirement`, `Stage`, `Code` (los 14 tokens de design §5.3), `Finding` y `Finding.Reason()` con la forma `blocked(<token>): <detalle>; <continuación>`.
  Cubre: REQ-29.16
  Aceptación: `Reason()` compone exactamente la línea de design §5.3; una prueba de importaciones falla si el paquete importa `os/exec` o cualquier paquete `internal/...`.
  Comprobación: `go test ./internal/isoevidence/ -run 'TestFindingReason|TestPackageImports'`
- [x] 1.2 Crear `internal/isoevidence/reqid.go` y `internal/isoevidence/reqid_test.go`: `RequirementIDs` (válido `^REQ-\d+\.\d+$` y mal formado en límite de palabra) y `CheckRequirementIDs` con las reglas de design §6.1 (ausente, ambiguo, repetido que cuenta como uno, ADDED único entre dominios y ausente de la baseline, MODIFIED/REMOVED/RENAMED con su continuación RENAMED) y la tolerancia D17 (mismo dominio, mismo nombre y bloque idéntico tras `ensureTrailingNewline`).
  Cubre: REQ-29.3
  Aceptación: RED primero con cada fila de la tabla de REQ-29.3 y los negativos de la tolerancia de design §7 (un byte distinto, otro nombre, otro dominio, diferencia solo en el salto de línea final); GREEN con todas en verde.
  Comprobación: `go test ./internal/isoevidence/ -run 'TestRequirementIDs|TestCheckRequirementIDs'`
- [x] 1.3 Crear `internal/isoevidence/scenarios.go` y `internal/isoevidence/scenarios_test.go`: `Scenarios` (solo `#### Scenario:` fuera de vallas) y `CheckScenarios` (ADDED/MODIFIED sin escenarios y nombres de escenario repetidos; REMOVED y RENAMED exentos).
  Cubre: REQ-29.10, REQ-29.6
  Aceptación: escenario dentro de una valla no cuenta; el motivo nombra dominio y requisito; REMOVED/RENAMED no se rechazan.
  Comprobación: `go test ./internal/isoevidence/ -run 'TestScenarios|TestCheckScenarios'`

### U2 · `feat(isoevidence): perfil de bug y cobertura Cubre`

- [x] 2.1 Crear `internal/isoevidence/bug.go` y `internal/isoevidence/bug_test.go`: `ParseBugDeclaration`, `BugDeclaration.Findings()` según design §5.6 y la tabla de D9 (`bug_report_incomplete`, `bug_no_impact_incomplete`; marca literal `Sin impacto en la spec` sensible a mayúsculas; sin delta y con `## Impacto en la spec`, ruta sin impacto).
  Cubre: REQ-29.15
  Aceptación: cada escenario de REQ-29.15 en tabla, incluidos el punto final, las minúsculas, valores vacíos y secciones dentro de vallas; si faltan cosas de las dos secciones salen los dos tokens.
  Comprobación: `go test ./internal/isoevidence/ -run TestParseBugDeclaration`
- [x] 2.2 Crear `internal/isoevidence/cubre.go` y `internal/isoevidence/cubre_test.go`: `ParseCubre`, `BaselineIndex`/`NewBaselineIndex`, `CheckCoverage`, `CheckExistingRequirementBug`, `CheckNoImpactBug`, `AffectedFromDelta`, `AffectedFromBaseline`.
  Cubre: REQ-29.4, REQ-29.5, REQ-29.14, REQ-29.19
  Aceptación: unión de varios ficheros de tareas, varios identificadores por línea, identificadores mal formados, `iso_cubre_unknown` una vez por identificador, baseline sin delta, requisito citado con cero escenarios, ambigüedad en la baseline y `iso_bug_contradiction` ante un bug sin impacto con `Cubre:`.
  Comprobación: `go test ./internal/isoevidence/ -run 'TestParseCubre|TestCheckCoverage|TestCheckExistingRequirementBug|TestCheckNoImpactBug|TestAffected'`

### U3 · `feat(isoevidence): matriz de verificación y resolución de referencias`

- [x] 3.1 RED de amenazas, `internal/isoevidence/testref_test.go` y `internal/isoevidence/matrix_threat_test.go`: (a) fila con un comando que crearía un fichero centinela y referencia a un `README.sh`: tras validar, el centinela no existe y la referencia resuelve solo por texto; (b) referencias `C:\x:Sym`, `\\srv\x:Sym`, `../x_test.go:TestX`, ruta absoluta y enlace simbólico que sale del árbol rechazadas con el motivo «fuera del árbol» (el caso del enlace se omite si el sistema no permite crearlo).
  Cubre: REQ-29.7
  Aceptación: las pruebas existen y fallan (RED) antes de 3.2.
  Comprobación: `go test ./internal/isoevidence/ -run 'TestTreeResolverRejectsEscapes|TestMatrixNeverExecutesCommands'` (rojo esperado antes de 3.2)
- [x] 3.2 Crear `internal/isoevidence/testref.go`: `TreeResolver`, `NewTreeResolver` sobre `os.Root`, `ResolvePathSymbol` (identificador completo, lectura acotada a 4 MiB) y `ResolveGoTest` (recorrido único con prefiltro y `go/parser`, omitiendo `.git`, `vendor`, `node_modules` y `testdata`; índice construido una sola vez).
  Cubre: REQ-29.7
  Aceptación: pasan las pruebas de 3.1 más forma general, `TestX`, subcadena de otro identificador, fichero mayor del límite, Python y TypeScript.
  Comprobación: `go test ./internal/isoevidence/ -run 'TestTreeResolver|TestResolveGoTest'`
- [x] 3.3 Crear `internal/isoevidence/matrix.go` (parte `ParseMatrix`) y `internal/isoevidence/matrix_parse_test.go`: sección `## Matriz de verificación` de nivel 2 fuera de vallas, cabeceras por nombre sin importar orden ni mayúsculas, `\|` literal, celdas con `-` o `—`, matriz ambigua (dos encabezados de nivel 2) y matriz histórica de nivel 3 ignorada.
  Cubre: REQ-29.6, REQ-29.8
  Aceptación: tabla de casos de design §5.5; ninguna línea de la sección empieza por `verdict:`.
  Comprobación: `go test ./internal/isoevidence/ -run TestParseMatrix`
- [x] 3.4 Completar `internal/isoevidence/matrix.go` con `ValidateMatrix` y `internal/isoevidence/matrix_validate_test.go`: completitud (una fila por escenario afectado, ninguna sobrante ni repetida), tipo y resultado exactos, fila automática (referencia resoluble, comando, salida entera, `pass` exige `0`), fila manual (responsable, fecha `AAAA-MM-DD`, justificación), `Successful`, y bug sin impacto (`not_required`, cualquier fila es inválida), con un `TreeResolver` simulado.
  Cubre: REQ-29.6, REQ-29.7, REQ-29.8, REQ-29.11, REQ-29.19
  Aceptación: cada escenario de REQ-29.6 a REQ-29.8 y REQ-29.19 en tabla; filas `fail` producen `iso_matrix_failed`.
  Comprobación: `go test ./internal/isoevidence/ -run TestValidateMatrix`

### U4 · `feat(isoevidence): traza por first-parent`

- [x] 4.1 Crear `internal/isoevidence/trace.go` y `internal/isoevidence/trace_test.go`: puerto `History`, `TraceInput`, `Trace` y `DeriveTrace` con el algoritmo de design §6.4 (commit de código = primero con `verify-report.md` y algún `tasks*.md`; carpeta archivada `^\d{4}-\d{2}-\d{2}-<c>$`; commit de archivado; contraste `pr_merged` con `Discrepancy` sin error).
  Cubre: REQ-29.9
  Aceptación: squash, merge y rebase simulados en el puerto dan la misma cadena; discrepancia con la evidencia; ausencia de carpeta archivada.
  Comprobación: `go test ./internal/isoevidence/ -run TestDeriveTrace`

---

## Fase 2: configuración `sdd:` en `axiom.yaml` (U5 a U7)

### U5 · `feat(workspace): bloque sdd con la clave iso`

- [x] 5.1 Crear `internal/workspace/sdd.go` y `internal/workspace/sdd_test.go` (parte de decodificación y validación cruzada): `SDDConfig`, `sddKeys`, `decodeSDD` (booleano estricto), `ErrISOWorkflowContradiction` y `checkISOWorkflow` con el mensaje de design §5.1.
  Cubre: REQ-29.1
  Aceptación: RED con `quizás`, `"true"` entre comillas, nulo, lista, clave desconocida (`archive`, `design_triggers`) y duplicada; solo `iso: true` con `lane: odd` falla, nombrando `sdd.iso`, `workflow.lane` y las dos correcciones; el resto de la matriz `iso` × `lane` es válido.
  Comprobación: `go test ./internal/workspace/ -run 'TestDecodeSDD|TestCheckISOWorkflow'`

### U6 · `feat(workspace): cargar sdd y rechazar la contradicción con workflow.lane` (solape P6)

- [x] 6.1 Modificar `internal/workspace/declarations.go` (campos `SDD`/`SDDErr`, una rama en `readDeclarationBlocks`, llamada a `checkISOWorkflow` tras decodificar `sdd:` y `workflow:`), `internal/workspace/loader.go` (`SDDErr` en `joinLoadErrors` y en el comentario de `ParseConfig`) y completar `internal/workspace/sdd_test.go` con la carga por capas (`ParseConfig`, `ReadDeclarations`, `LoadConfig`).
  Cubre: REQ-29.1, REQ-29.2
  Aceptación: la contradicción y un `sdd:` inválido fallan al cargar con salida `1` en `workspace validate`; `workflow` se resuelve igual y `WorkflowErr` queda en `nil`; sin bloque `sdd:` nada cambia.
  Comprobación: `go test ./internal/workspace/ -run 'TestSDDLoad|TestWorkspaceValidateSDD'`
- [x] 6.2 Modificar `internal/hub/init.go` y su prueba (`internal/hub/init_test.go`): `declares()` cuenta `SDD` y `SDDErr` para que `axiom init --force` no pierda `sdd:`.
  Cubre: REQ-29.1
  Aceptación: con solo `sdd:` declarado, `declares()` es verdadero.
  Comprobación: `go test ./internal/hub/ -run TestDeclaresCountsSDD`

### U7 · `feat(sddstatus): projectWorkflow no asume odd ante la contradicción ISO` (solape P6)

- [x] 7.1 Modificar `internal/sddstatus/workspace_declarations.go` (solo `projectWorkflow`: `ask` / `invalid` cuando `errors.Is(SDDErr, ErrISOWorkflowContradiction)`) y crear `internal/sddstatus/iso_workflow_test.go`.
  Cubre: REQ-29.1, REQ-29.2
  Aceptación: con la contradicción, `workflow.lane` proyecta `ask` / `invalid`; un `SDDErr` ordinario no cambia el carril.
  Comprobación: `go test ./internal/sddstatus/ -run TestProjectWorkflowISOContradiction`

---

## Fase 3: instantánea del kickoff y arranque con carpeta (U8 y U9)

### U8 · `feat(kickoff): instantánea sdd.iso y PrepareChangeRoot`

- [x] 8.1 Modificar `internal/kickoff/types.go` (`Kickoff.SDD *SDDSnapshot`, `SDDSnapshot`, `IsISO()`) y `internal/kickoff/types_test.go`.
  Cubre: REQ-29.13
  Aceptación: `omitempty` (con iso falso no se escribe `sdd:`), lectura sin clave, decodificación con el esquema previo (struct sin `SDD`) y `IsISO`.
  Comprobación: `go test ./internal/kickoff/ -run 'TestSDDSnapshot|TestIsISO'`
- [x] 8.2 Modificar `internal/kickoff/seal.go` (`PrepareChangeRoot(changeRoot) (rollback func(), err error)`) y `internal/kickoff/seal_prepare_test.go`.
  Cubre: REQ-29.12
  Aceptación: crea padres ausentes; la retirada borra en orden inverso solo lo creado y solo si está vacío; carpeta existente intacta y retirada sin efecto.
  Comprobación: `go test ./internal/kickoff/ -run TestPrepareChangeRoot`

### U9 · `feat(cli): kickoff seal prepara la carpeta del cambio y copia sdd.iso`

- [x] 9.1 RED de amenazas en `internal/cli/sdd_kickoff_test.go`: `--change ../x`, `--change archive`, nombres con `:(`, `*` y `..` rechazados sin crear nada; `wrong_branch` y `sdd:` inválido o contradictorio sin carpeta ni `kickoff.yaml`.
  Cubre: REQ-29.12, REQ-29.1
  Aceptación: pruebas escritas y en rojo antes de 9.2; la prueba de `TestRunSDDKickoffSealOnNonexistentChangeNamesResolvedPath` invertida.
  Comprobación: `go test ./internal/cli/ -run 'TestRunSDDKickoffSeal'` (rojo esperado antes de 9.2)
- [x] 9.2 Modificar `internal/cli/sdd_kickoff.go` y `internal/cli/sdd_kickoff_test.go`: `resolveSealChangeRoot` (nombre válido con `gitref.ValidBranchComponent`, `archive` reservado, rechazo de cambio archivado), creación con `kickoff.PrepareChangeRoot` tras todas las validaciones y justo antes de `kickoff.Seal`, retirada si `Seal` falla, lectura de `sdd:`, copia de `sdd.iso` (nunca con `--infer`, que sigue exigiendo carpeta) e `iso` en texto y JSON.
  Cubre: REQ-29.12, REQ-29.13, REQ-29.1
  Aceptación: escenarios de REQ-29.12 (idempotencia, carpeta existente intacta, sin carpeta ante rechazo) y REQ-29.13 (una sola escritura, `--infer` sin `iso`).
  Comprobación: `go test ./internal/cli/ -run 'TestRunSDDKickoffSeal'`
- [x] 9.3 Modificar `internal/branch/engine/writer_inventory_test.go`: añadir `{"internal/kickoff", "PrepareChangeRoot"}` a `writerCalls`.
  Cubre: REQ-29.12
  Aceptación: el inventario de escritores sigue en verde; `runSDDKickoffSeal` queda cubierto por `applySealBranchPolicy`.
  Comprobación: `go test ./internal/branch/engine/ -run TestWriterInventory`

---

## Fase 4: estado (`status`) y motivos tipados (U10 a U13)

### U10 · `feat(sddstatus): perfil de bug sin delta y compuerta spec sobre proposal.md`

- [x] 10.1 Crear `internal/sddstatus/iso.go` (parte del perfil de bug: adaptador de D9) y modificar `internal/sddstatus/status.go` (llamada en `resolveOpenSpecStatus` tras calcular `artifacts`); pruebas en `internal/sddstatus/iso_test.go`.
  Cubre: REQ-29.15, REQ-29.16
  Aceptación: sin las secciones, estado idéntico al actual; declaración válida, `specs` pasa a `done`; incompleta, `specs` sigue `missing` con un motivo por cada falta; con delta, las secciones no se analizan; vale en todo proyecto, sea o no ISO.
  Comprobación: `go test ./internal/sddstatus/ -run TestISOBugProfile`
- [x] 10.2 Modificar `internal/sddstatus/governance.go` (huella de la compuerta `spec` sobre `proposal.md` en un bug sin delta), `internal/cli/sdd_gate_digest.go` (mismo criterio) y `internal/sddstatus/governance_test.go`.
  Cubre: REQ-29.15
  Aceptación: la compuerta `spec` de un bug sin delta se abre, se reabre al cambiar `proposal.md` y la huella coincide entre status y `gate record`; con delta, juzga las specs como hoy.
  Comprobación: `go test ./internal/sddstatus/ -run TestGovernanceSpecGateBugWithoutDelta` y `go test ./internal/cli/ -run TestGateDigestBugWithoutDelta`

### U11 · `feat(sddstatus): proyección iso y declaración ISO en el status`

- [ ] 11.1 Completar `internal/sddstatus/iso.go` (`ISOProjection`, `decorateISO`: `iso.project`, `iso.reason`, nota de discrepancia, nota de declaración inválida con o sin cambio, `ParseErr`, `iso_declaration_invalid`, `iso_store_unsupported`), modificar `internal/sddstatus/status.go` (campo `ISO`, llamada desde `Resolve` tras `decorateWorkspaceDeclarations`) y `internal/sddstatus/status_v2.go` (`ISO` en `StatusV2Projection`).
  Cubre: REQ-29.1, REQ-29.2, REQ-29.13
  Aceptación: `iso` solo aparece con bloque `sdd:` o cambio ISO; `invalid` con `reason` aunque no haya cambio seleccionado; el archivado se estrecha solo cuando la ruta sería `archive`; un cambio ISO con almacén `engram` o `none` bloquea con `iso_store_unsupported`.
  Comprobación: `go test ./internal/sddstatus/ -run 'TestDecorateISO|TestISOProjection'`
- [ ] 11.2 Prueba de continuidad en `internal/sddstatus/iso_test.go`: sin bloque `sdd:` el JSON del status es idéntico byte a byte; las pruebas existentes de regresión no cambian.
  Cubre: REQ-29.2
  Aceptación: `kickoff_absence_regression_test.go` y `status_v2_clean_break_test.go` siguen en verde sin editarlos.
  Comprobación: `go test ./internal/sddstatus/ -run 'TestISOStatusContinuity|TestKickoffAbsenceRegression|TestStatusV2CleanBreak'`

### U12 · `feat(sddstatus): etapas specs y tasks de las reglas ISO`

- [ ] 12.1 Completar `internal/sddstatus/iso.go` con el adaptador de la etapa `specs`: índice de baseline con `parseSpecDocument` y modelo de la delta con `parseOpenSpecDelta` (ambas en `internal/sddstatus/openspec_archive_compose.go` (read-only)); `CheckRequirementIDs` y `CheckScenarios`; estrechamiento de `specs` y de `coreReady`.
  Cubre: REQ-29.3, REQ-29.10, REQ-29.6, REQ-29.16
  Aceptación: motivos `iso_requirement_id` e `iso_requirement_scenarios` en `blockedReasons`, ruta vuelve a `spec`; archivado interrumpido tras componer un ADDED con el compositor real no proyecta `iso_requirement_id` ni ruta a `spec`; fuera de ISO no aparece ningún token `iso_*`.
  Comprobación: `go test ./internal/sddstatus/ -run 'TestISOSpecsStage|TestISOInterruptedArchive'`
- [ ] 12.2 Completar `internal/sddstatus/iso.go` y `internal/sddstatus/status.go` con la etapa `tasks`: lectura de todos los ficheros de tareas (`kickoff.RoleTaskFiles` o `tasks.md`), `CheckCoverage`, `CheckExistingRequirementBug`, `CheckNoImpactBug`; estrechamiento de `tasks`.
  Cubre: REQ-29.4, REQ-29.5, REQ-29.19, REQ-29.16
  Aceptación: `iso_coverage`, `iso_cubre_unknown`, `iso_bug_requirement` e `iso_bug_contradiction` solo cuando existen todos los ficheros de tareas; un motivo por hallazgo; `axiom sdd continue` refleja los mismos motivos; `wrong_branch` sigue delante.
  Comprobación: `go test ./internal/sddstatus/ -run 'TestISOTasksStage|TestISOReasonsOrdering'`

### U13 · `feat(sddstatus): etapa archive, matriz y ruta verify en cambios ISO`

- [ ] 13.1 Completar `internal/sddstatus/iso.go` con `ISOArchiveRefusal(workspaceRoot, changeRoot) []isoevidence.Finding` (declaración válida, escenarios de todas las deltas y matriz válida y de éxito, con `NewTreeResolver` sobre `Location.Master`) y `archiveDependencyFromISO` (solo estrecha de `ready` a `blocked`).
  Cubre: REQ-29.10, REQ-29.11, REQ-29.6, REQ-29.7, REQ-29.8
  Aceptación: con tareas pendientes, `archive` bloqueado, motivo solo en `notes`, ruta `apply` sin cambios (D12); la regla no evalúa REQ-29.3.
  Comprobación: `go test ./internal/sddstatus/ -run 'TestISOArchiveRefusal|TestISOArchiveWithPendingTasks'`
- [ ] 13.2 Completar `internal/sddstatus/iso.go` y `internal/sddstatus/status.go` con la ruta `verify` (D19) y `iso.matrix`: con `apply` terminado y matriz ausente o inválida, `nextRecommended: verify` (también sobre `await-gate` de `integration`); filas `fail`, `resolve-blockers`; `ISOMatrixProjection`.
  Cubre: REQ-29.11, REQ-29.16, REQ-29.19
  Aceptación: las compuertas `spec`, `design`, `tasks` y `role-apply` conservan su precedencia; bug sin impacto sin matriz queda `not_required` y de éxito; motivos `iso_matrix_missing`, `iso_matrix_invalid` e `iso_matrix_failed` con su continuación.
  Comprobación: `go test ./internal/sddstatus/ -run 'TestISOVerifyRoute|TestISOMatrixProjection'`

---

## Fase 5: archivado, `archive-check` y `trace` (U14 a U17)

### U14 · `feat(cli): axiom sdd archive-check de solo lectura`

- [ ] 14.1 Crear `internal/cli/sdd_archive_check.go`, `internal/cli/sdd_archive_check_test.go` y modificar `cmd/axiom` (registro del verbo y su ayuda junto a los demás `sdd`): verbo `axiom sdd archive-check --cwd --change [--json]` sobre `sddstatus.ISOArchiveRefusal`, texto y JSON.
  Cubre: REQ-29.10, REQ-29.11
  Aceptación: salida `0` en un cambio no ISO o sin hallazgos; salida `1` con motivos tipados con matriz ausente (también con tareas pendientes), bug sin delta, capacidad nueva sin composición y declaración inválida; no escribe nada.
  Comprobación: `go test ./internal/cli/ -run TestSDDArchiveCheck`
- [ ] 14.2 Completar `internal/cli/sdd_archive_check.go` con `alreadyComposed` por dominio (D17: todo ADDED con mismo nombre y bloque, todo MODIFIED con el bloque de la delta, todo REMOVED ausente o superado con `--supersede`, todo RENAMED con el nuevo presente y el antiguo ausente).
  Cubre: REQ-29.3, REQ-29.10
  Aceptación: tras componer un dominio con el compositor real, informa `alreadyComposed: true` para él y `false` para el resto; el resto se compone y se mueve.
  Comprobación: `go test ./internal/cli/ -run TestSDDArchiveCheckAlreadyComposed`

### U15 · `feat(cli): archive-compose rechaza el archivado ISO sin escenarios o sin matriz`

- [ ] 15.1 Crear `internal/cli/sdd_archive_compose_iso.go` y `internal/cli/sdd_archive_compose_iso_test.go`: determinación del cambio con `changeRootOfDelta` (modo `--canonical`, como `guardComposeLooseDelta`) y negativa con `ISOArchiveRefusal`, antes de leer la canónica y sin escribir; sin cambio determinable, no aplica nada.
  Cubre: REQ-29.10, REQ-29.11
  Aceptación: la función se prueba directamente: ISO con y sin escenarios, matriz inválida y sin contexto de cambio.
  Comprobación: `go test ./internal/cli/ -run TestArchiveComposeISORefusal`

### U16 · `feat(cli): enganchar la negativa ISO en archive-compose` (solape P6)

- [ ] 16.1 Modificar `internal/cli/sdd_archive_compose.go` (llamada de 5 a 8 líneas antes de leer la canónica) y completar `internal/cli/sdd_archive_compose_iso_test.go` con las pruebas de extremo a extremo.
  Cubre: REQ-29.10, REQ-29.11, REQ-24.38
  Aceptación: rechazo ISO sin escribir; la salida de una composición aceptada y la de un cambio no ISO son idénticas byte a byte; el modo `--change --in-place` no se toca.
  Comprobación: `go test ./internal/cli/ -run 'TestArchiveCompose'`

### U17 · `feat(cli): axiom sdd trace de solo lectura`

- [ ] 17.1 RED de amenazas en `internal/cli/sdd_trace_test.go`: traza lanzada desde un directorio ajeno con `--cwd` al repositorio; nombre de cambio con `:(`, `*` o `..` y referencia con `..` rechazados antes de llamar a git.
  Cubre: REQ-29.9
  Aceptación: pruebas escritas y en rojo antes de 17.2.
  Comprobación: `go test ./internal/cli/ -run TestSDDTraceRejectsUnsafeInput` (rojo esperado antes de 17.2)
- [ ] 17.2 Crear `internal/cli/sdd_trace.go` y completar `internal/cli/sdd_trace_test.go`: verbo `axiom sdd trace --cwd --change [--integration-branch] [--json]`, puerto `History` sobre `gitcommon.Run` con `dir` = raíz git resuelta desde `--cwd`, referencia validada con `gitref.ValidRefName`, esquema `axiom.sdd-trace/v1`; repos git reales en `t.TempDir()` con squash, merge y rebase, PR del cambio y PR documental, discrepancia y ausencia.
  Cubre: REQ-29.9
  Aceptación: misma cadena REQ → `Cubre:` → fila → commit de código → commit de archivado con los tres modos de fusión; ningún fichero nuevo en el árbol tras ejecutar.
  Comprobación: `go test ./internal/cli/ -run 'TestSDDTrace'`
- [ ] 17.3 Modificar `cmd/axiom` para registrar `sdd trace` y su ayuda.
  Cubre: REQ-29.9
  Aceptación: el verbo se enruta junto a los demás `sdd` y aparece en la ayuda.
  Comprobación: `go test ./cmd/axiom/ -run 'TestSDDVerbRouting'`

---

## Fase 6: QA diferido (U18 y U19)

### U18 · `fix(sddstatus): verify solo exige los role-apply de los roles blocking`

- [ ] 18.1 Modificar `internal/kickoff/machine.go` (`LastBlockingRoleClosed`, `RoleGateStatus`, `PendingNonBlockingRoles`) y `internal/kickoff/handed_off.go` (`HandedOffTo` con `LastBlockingRoleClosed`; `IntermediateMerge` solo con roles `blocking`) y sus pruebas (`internal/kickoff/machine_test.go`, `internal/kickoff/handed_off_test.go`).
  Cubre: REQ-21.15, REQ-21.13
  Aceptación: todos `blocking`, mezcla, sin bloqueantes y rechazado; `IntermediateMerge` con un rol `deferred` pendiente no rechaza el archivado; con el rol único `fullstack` el resultado es el actual.
  Comprobación: `go test ./internal/kickoff/ -run 'TestLastBlockingRoleClosed|TestPendingNonBlockingRoles|TestIntermediateMerge|TestHandedOffTo'`
- [ ] 18.2 Modificar `internal/sddstatus/governance_handoff.go` (`verifyDependencyFromRoleGates` por `gate_policy`, con notas), `internal/sddstatus/governance.go` (`lastRoleClosedForGovernance` pasa a bloqueantes) y `internal/sddstatus/role_gates_dependency_test.go`.
  Cubre: REQ-21.15
  Aceptación: con un `blocking` aprobado y un `deferred` sin aprobar, `verify` queda desbloqueado y el rol diferido se informa en notas con su política y estado; con un `blocking` sin aprobar sigue bloqueado.
  Comprobación: `go test ./internal/sddstatus/ -run 'TestVerifyDependencyFromRoleGates|TestRoleGatesDependency'`

### U19 · `fix(cli): aviso de último rol activo solo al cerrar el último bloqueante`

- [ ] 19.1 Modificar `internal/cli/sdd_gate.go`, `internal/cli/sdd_gate_handoff.go` y `internal/cli/sdd_gate_test.go`: aviso de REQ-21.13 solo cuando la compuerta aprobada es de un rol `blocking` y con ella quedan aprobados todos los bloqueantes; el aviso nombra los no bloqueantes pendientes; el JSON conserva `lastRoleClosed` y añade `pendingNonBlockingRoles` con `omitempty`.
  Cubre: REQ-21.13
  Aceptación: escenarios con `deferred` y `optional`; sin segundo aviso; con todos los roles `blocking` el JSON no cambia.
  Comprobación: `go test ./internal/cli/ -run 'TestGateRecordLastRoleNotice'`

---

## Fase 7: doctrina, skills, goldens y manual (U20 a U24)

### U20 · `feat(agentguidance): el proyecto ISO tramita todo por SDD` (solape P6)

- [ ] 20.1 Modificar `internal/components/agentguidance/routing.go` (condición ISO al principio del paso 0, excepción en «Small, understood work…», preámbulo y viñeta **Optional SDD**), `internal/components/agentguidance/delegation.go` (proyecto ISO en **Optional SDD trigger**) y los pines en `internal/components/agentguidance/routing_stops_test.go`; regenerar con `-update` los goldens afectados.
  Cubre: REQ-29.18
  Aceptación: pin de texto con la condición antes de la pregunta de carril, también para trabajo pequeño, sin `odd/tasks/`, tres rutas del bug con sus literales y sin regla de precedencia entre `sdd.iso` y `workflow.lane: odd`; `checkRoutingPolicy` sigue aceptando `explicit_request_project_lane_or_accepted_proposal` y el bloque renderizado con ella nombra la vía ISO; texto idéntico entre agentes y entre proyectos ISO y no ISO.
  Comprobación: `go test ./internal/components/agentguidance/ -run 'TestRenderRouting|TestRoutingISO|TestUnknownRoutingPolicyFailsClosed'` y `go test ./internal/components/ -run TestGolden -update` seguido de `go test ./internal/components/ -run TestGolden`

### U21 · `docs(assets): convención ISO, doctrina del orquestador y skills`

- [ ] 21.1 Crear `internal/assets/skills/_shared/sdd-iso-convention.md`: formatos de `Cubre:`, matriz (sin `verdict:`; matriz histórica a nivel 3 con fecha), referencias de prueba, perfil de bug con sus textos literales y tres rutas, `archive-check`, reanudación de un archivado interrumpido (solo dominios con `alreadyComposed: false`) y `sdd trace`.
  Cubre: REQ-29.17
  Aceptación: contiene los literales `## Informe de bug`, `## Impacto en la spec`, `Sin impacto en la spec` y `## Matriz de verificación`, y la regla histórica de nivel 3.
  Comprobación: `go test ./internal/assets/ -run TestSDDISOConvention`
- [ ] 21.2 Modificar `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (solo dentro de secciones existentes: «Dependency Graph» con el verify obligatorio en ISO, «SDD Change Kickoff and Block Gates» y «SDD Independent Review») y `internal/assets/skills/_shared/persistence-contract.md` (excepción ISO junto a la regla general).
  Cubre: REQ-29.17, REQ-29.11, REQ-28.3, REQ-28.4, REQ-28.6
  Aceptación: ningún marcador ni sección nuevos, así que `orchestrator_no_drift_test.go` pasa sin tocar las 12 plantillas; la excepción ISO declara que los hallazgos de revisión siguen siendo informativos y que solo la matriz bloquea; REQ-28.4 remite a la condición de REQ-21.15 sin repetirla.
  Comprobación: `go test ./internal/components/sdd/ -run 'TestOrchestratorNoDrift|TestCanonicalComposition'`
- [ ] 21.3 Modificar `internal/assets/skills/sdd-propose/SKILL.md`, `internal/assets/skills/sdd-spec/SKILL.md`, `internal/assets/skills/sdd-tasks/SKILL.md`, `internal/assets/skills/sdd-verify/SKILL.md`, `internal/assets/skills/sdd-verify/references/report-format.md` y `internal/assets/skills/sdd-onboard/SKILL.md`: remisión de una línea a la convención; `seal` sin crear la carpeta a mano; matriz sin `verdict:`; matriz anterior a nivel 3 con fecha en *Historical context*.
  Cubre: REQ-29.17, REQ-29.12
  Aceptación: cada skill cita `_shared/sdd-iso-convention.md`; `report-format.md` fija la regla histórica de nivel 3.
  Comprobación: `go test ./internal/assets/ -run 'TestSDDSkillsReferenceISOConvention|TestReportFormatHistoricalMatrix'`
- [ ] 21.4 Añadir los pines de REQ-29.17 en `internal/assets/sdd_iso_pins_test.go` (o en la prueba de pines vigente de assets) y regenerar con `-update` los goldens afectados.
  Cubre: REQ-29.17
  Aceptación: pines de textos literales, excepción ISO, sin `verdict:`, sin carpeta a mano, verify obligatorio en ISO y paridad entre runtimes; nueva pasada sin `-update` en verde; si una prueba enumera `_shared/`, se actualiza en este mismo commit.
  Comprobación: `go test ./internal/assets/ -run 'TestSDDISOPins'` y `go test ./internal/components/ -run TestGolden`

### U22 · `docs(assets): sdd-archive exige archive-check y reanuda archivados` (solape P6)

- [ ] 22.1 Modificar `internal/assets/skills/sdd-archive/SKILL.md` (paso obligatorio de *Archive Readiness*: `axiom sdd archive-check --cwd <workspaceRoot> --change <name>` y `STOP` con `blocked` ante salida distinta de cero; compone solo los dominios con `alreadyComposed: false`) y su pin en `internal/assets/sdd_iso_pins_test.go`.
  Cubre: REQ-29.17, REQ-29.10, REQ-29.11
  Aceptación: el pin falla si la skill pierde el paso previo a componer o mover.
  Comprobación: `go test ./internal/assets/ -run TestSDDArchiveRequiresArchiveCheck`

### U23 · `docs(assets): contrato de status con la proyección iso y los motivos` (solape P6)

- [ ] 23.1 Modificar `internal/assets/skills/_shared/sdd-status-contract.md` (proyección `iso` con `reason`, ruta `verify` en ISO y tabla de los 14 tokens) y su pin en `internal/assets/sdd_iso_pins_test.go`; regenerar con `-update` los goldens afectados.
  Cubre: REQ-29.2, REQ-29.16, REQ-29.17
  Aceptación: la tabla de tokens coincide con la lista cerrada de `isoevidence` (la prueba compara ambas).
  Comprobación: `go test ./internal/assets/ -run TestSDDStatusContractISO` y `go test ./internal/components/ -run TestGolden`

### U24 · `docs: declaración ISO, perfil de bug y traza en el manual y el ejemplo`

- [ ] 24.1 Modificar `docs/manual-de-inicio.md` y `axiom.example.yaml`: declaración `sdd: {iso: true}`, perfil de bug con sus tres rutas y traza.
  Cubre: REQ-29.17
  Aceptación: el ejemplo carga sin error y `iso: true` no convive con `workflow.lane: odd`.
  Comprobación: `go test ./internal/workspace/ -run TestExampleConfigLoads`

### Cierre sin commit propio

- [ ] 24.2 Verificación final de goldens, copias versionadas y paquetes tocados, sin `-update`. Si `AGENTS.md`, `GEMINI.md` o `.codex/AGENTS.md` solo se regeneran con `axiom sync`, no compilar ni ejecutar un binario: localizar la prueba que las compara con `rg`; si no existe ruta en proceso, parar y consultar al usuario.
  Cubre: REQ-29.17, REQ-29.18
  Aceptación: nada pendiente de regenerar; `git status` limpio tras la pasada.
  Comprobación: `go test ./internal/components/ -run TestGolden`, `go test ./internal/components/agentguidance/`, `go test ./internal/components/sdd/`, `go test ./internal/assets/`, `go test ./internal/isoevidence/`, `go test ./internal/workspace/`, `go test ./internal/kickoff/`, `go test ./internal/sddstatus/`, `go test ./internal/cli/`

---

## Cobertura REQ → tareas

| REQ | Tareas |
|---|---|
| REQ-29.1 | 5.1, 6.1, 6.2, 7.1, 9.1, 9.2, 11.1 |
| REQ-29.2 | 6.1, 7.1, 11.1, 11.2, 23.1 |
| REQ-29.3 | 1.2, 12.1, 14.2 |
| REQ-29.4 | 2.2, 12.2 |
| REQ-29.5 | 2.2, 12.2 |
| REQ-29.6 | 1.3, 3.3, 3.4, 13.1 |
| REQ-29.7 | 3.1, 3.2, 3.4, 13.1 |
| REQ-29.8 | 3.3, 3.4, 13.1 |
| REQ-29.9 | 4.1, 17.1, 17.2, 17.3 |
| REQ-29.10 | 1.3, 12.1, 13.1, 14.1, 14.2, 15.1, 16.1, 22.1 |
| REQ-29.11 | 3.4, 13.1, 13.2, 14.1, 15.1, 16.1, 21.2, 22.1 |
| REQ-29.12 | 8.2, 9.1, 9.2, 9.3, 21.3 |
| REQ-29.13 | 8.1, 9.2, 11.1 |
| REQ-29.14 | 2.2 |
| REQ-29.15 | 2.1, 10.1, 10.2 |
| REQ-29.16 | 1.1, 1.3, 10.1, 12.1, 12.2, 13.2, 23.1 |
| REQ-29.17 | 21.1 a 21.4, 22.1, 23.1, 24.1, 24.2 |
| REQ-29.18 | 20.1, 24.2 |
| REQ-29.19 | 2.2, 3.4, 12.2, 13.2 |
| REQ-21.13 | 18.1, 19.1 |
| REQ-21.15 | 18.1, 18.2 |
| REQ-24.38 | 16.1 |
| REQ-28.3, REQ-28.4, REQ-28.6 | 21.2 |

## Orden y paralelismo

Es un único rol (`fullstack`), así que **todo es secuencial** dentro de la misma rama y la misma PR; no hay paralelismo entre roles. Dependencias entre unidades:

- U1 a U4 son la raíz. U1, U2, U3 y U4 son independientes entre sí a nivel de código, pero se mantienen en orden para que la revisión sea por pieza.
- U5 a U7 no dependen de la Fase 1 y habilitan U9 (`SDDErr`) y U11. U8 precede a U9.
- U10 a U13 dependen de U1 a U4 y de U5 a U9; U13 depende de U10 a U12.
- U14 a U16 dependen de U13; U17 depende de U4 y de U8.
- U18 y U19 son independientes del modo ISO (corrección del QA diferido) y por eso van tras el núcleo, como pide el orden de revisión; podrían adelantarse sin romper nada.
- U20 a U24 van al final, para regenerar los goldens con la prosa ya cerrada.

---

## Review Workload Forecast

Líneas estimadas = autoradas, creadas más borradas, sin goldens ni copias versionadas; coherentes con design §10 (~5.730).

| Commit | Unidad | Líneas |
|---|---|---|
| 1 | U1 `isoevidence`: modelo, identificadores y escenarios | 555 |
| 2 | U2 `isoevidence`: perfil de bug y cobertura | 520 |
| 3 | U3 `isoevidence`: matriz y referencias | 760 |
| 4 | U4 `isoevidence`: traza | 290 |
| 5 | U5 `workspace`: bloque `sdd:` | 184 |
| 6 | U6 `workspace` y `hub`: carga y `init` (solape P6) | 107 |
| 7 | U7 `projectWorkflow` (solape P6) | 29 |
| 8 | U8 `kickoff`: instantánea y `PrepareChangeRoot` | 153 |
| 9 | U9 `kickoff seal` en `cli` | 278 |
| 10 | U10 perfil de bug y compuerta `spec` | 230 |
| 11 | U11 proyección `iso` | 206 |
| 12 | U12 etapas `specs` y `tasks` | 275 |
| 13 | U13 etapa `archive`, matriz y ruta `verify` | 235 |
| 14 | U14 `archive-check` | 262 |
| 15 | U15 negativa en `archive-compose` | 110 |
| 16 | U16 enganche en `archive-compose` (solape P6) | 68 |
| 17 | U17 `sdd trace` | 388 |
| 18 | U18 QA diferido en `kickoff` y `sddstatus` | 344 |
| 19 | U19 aviso de `gate record` | 131 |
| 20 | U20 `agentguidance` (solape P6) | 71 |
| 21 | U21 convención, secciones y skills | 349 |
| 22 | U22 skill `sdd-archive` (solape P6) | 50 |
| 23 | U23 `sdd-status-contract.md` (solape P6) | 85 |
| 24 | U24 manual y ejemplo | 50 |
| **Total** | **24 commits (47 tareas con commit, más 1 de cierre)** | **~5.730** |

Margen: de 4.900 a 5.750 líneas (design §10), según cuánto compriman las tablas de pruebas.

| Field | Value |
|-------|-------|
| Estimated changed lines | ~5.730 (rango 4.900 a 5.750) |
| 400-line budget risk | High |
| Chained PRs recommended | No (exception-ok aceptado por el usuario) |
| Suggested split | Una sola PR con 24 commits revisables por pieza |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No (exception-ok aceptado por el usuario)
Chain strategy: size-exception
400-line budget risk: High

> Nota `size:exception`: la PR supera con mucho las 400 líneas (la unidad más grande, U3, ronda 760) y debe llevar la etiqueta `size:exception`, aceptada por el mantenedor (preferencia permanente del usuario: una sola PR, nunca cadena). Se revisa por commits, en el orden de las unidades.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| U1 | Modelo, identificadores y escenarios | PR única | `go test ./internal/isoevidence/ -run 'TestRequirementIDs\|TestCheckRequirementIDs\|TestScenarios\|TestCheckScenarios\|TestFindingReason\|TestPackageImports'` | N/A: funciones puras, prueba en proceso | `internal/isoevidence/` (paquete nuevo y sin consumidores) |
| U2 | Perfil de bug y cobertura | PR única | `go test ./internal/isoevidence/ -run 'TestParseBugDeclaration\|TestParseCubre\|TestCheckCoverage'` | N/A: funciones puras | Ficheros `bug.go` y `cubre.go` con sus pruebas |
| U3 | Matriz y referencias | PR única | `go test ./internal/isoevidence/ -run 'TestTreeResolver\|TestParseMatrix\|TestValidateMatrix'` | Árbol real en `t.TempDir()` con enlace simbólico y fichero centinela | Ficheros `matrix.go` y `testref.go` |
| U4 | Traza | PR única | `go test ./internal/isoevidence/ -run TestDeriveTrace` | Puerto `History` simulado | `trace.go` y su prueba |
| U5 | Bloque `sdd:` | PR única | `go test ./internal/workspace/ -run 'TestDecodeSDD\|TestCheckISOWorkflow'` | N/A: decodificación pura | `sdd.go` y su prueba |
| U6 | Carga, validación y `init` (solape P6) | PR única | `go test ./internal/workspace/ -run 'TestSDDLoad\|TestWorkspaceValidateSDD'` y `go test ./internal/hub/ -run TestDeclaresCountsSDD` | `axiom workspace validate` simulado con `ParseConfig` y salida `1` | Líneas de `declarations.go`, `loader.go` y `init.go` |
| U7 | `projectWorkflow` (solape P6) | PR única | `go test ./internal/sddstatus/ -run TestProjectWorkflowISOContradiction` | Workspace en `t.TempDir()` | Un caso de `projectWorkflow` |
| U8 | Instantánea y `PrepareChangeRoot` | PR única | `go test ./internal/kickoff/ -run 'TestSDDSnapshot\|TestIsISO\|TestPrepareChangeRoot'` | Directorios reales en `t.TempDir()` | `types.go` y `seal.go` de `kickoff` |
| U9 | `kickoff seal` con carpeta e `iso` | PR única | `go test ./internal/cli/ -run TestRunSDDKickoffSeal` y `go test ./internal/branch/engine/ -run TestWriterInventory` | `seal` real sobre repo en `t.TempDir()` | `sdd_kickoff.go` y la línea del inventario |
| U10 | Perfil de bug y compuerta `spec` | PR única | `go test ./internal/sddstatus/ -run 'TestISOBugProfile\|TestGovernanceSpecGateBugWithoutDelta'` | Workspace con `proposal.md` de bug | Hook de D9 y huella D20 |
| U11 | Proyección `iso` | PR única | `go test ./internal/sddstatus/ -run 'TestDecorateISO\|TestISOStatusContinuity'` | Status completo con y sin `sdd:` | `decorateISO` y el campo `ISO` |
| U12 | Etapas `specs` y `tasks` | PR única | `go test ./internal/sddstatus/ -run 'TestISOSpecsStage\|TestISOTasksStage\|TestISOInterruptedArchive'` | Compositor real de archivado | Etapas `specs` y `tasks` de `iso.go` |
| U13 | Etapa `archive`, matriz y `verify` | PR única | `go test ./internal/sddstatus/ -run 'TestISOArchiveRefusal\|TestISOVerifyRoute'` | Cambio ISO con `apply` terminado y árbol real | Etapa `archive` y ruta `verify` |
| U14 | `archive-check` | PR única | `go test ./internal/cli/ -run 'TestSDDArchiveCheck'` | Cambios en `t.TempDir()` con salida de proceso `0` y `1` | `sdd_archive_check.go` y su registro |
| U15 | Negativa en `archive-compose` | PR única | `go test ./internal/cli/ -run TestArchiveComposeISORefusal` | Delta real en `t.TempDir()` | `sdd_archive_compose_iso.go` |
| U16 | Enganche (solape P6) | PR única | `go test ./internal/cli/ -run TestArchiveCompose` | Composición real, salida byte a byte | Una llamada de 8 líneas |
| U17 | `sdd trace` | PR única | `go test ./internal/cli/ -run TestSDDTrace` | Tres repos git reales (squash, merge, rebase) en `t.TempDir()` | `sdd_trace.go` y su registro |
| U18 | QA diferido en estado | PR única | `go test ./internal/kickoff/ -run 'TestLastBlockingRoleClosed\|TestIntermediateMerge'` y `go test ./internal/sddstatus/ -run TestRoleGatesDependency` | Roster con `deferred` y `optional` | `machine.go`, `handed_off.go` y `governance_handoff.go` |
| U19 | Aviso de cierre | PR única | `go test ./internal/cli/ -run TestGateRecordLastRoleNotice` | `gate record` con roles mixtos | `sdd_gate.go` y `sdd_gate_handoff.go` |
| U20 | Doctrina ODD (solape P6) | PR única | `go test ./internal/components/agentguidance/ -run 'TestRenderRouting\|TestRoutingISO'` | Render del bloque y goldens | `routing.go` y `delegation.go` |
| U21 | Convención, secciones y skills | PR única | `go test ./internal/assets/ -run 'TestSDDISOConvention\|TestSDDISOPins'` y `go test ./internal/components/sdd/ -run TestOrchestratorNoDrift` | Goldens con `-update` y nueva pasada | Ficheros `_shared/` y skills |
| U22 | Skill `sdd-archive` (solape P6) | PR única | `go test ./internal/assets/ -run TestSDDArchiveRequiresArchiveCheck` | N/A: pin de texto | `sdd-archive/SKILL.md` |
| U23 | Contrato de status (solape P6) | PR única | `go test ./internal/assets/ -run TestSDDStatusContractISO` | Goldens | `sdd-status-contract.md` |
| U24 | Manual y ejemplo | PR única | `go test ./internal/workspace/ -run TestExampleConfigLoads` | Carga del ejemplo | `manual-de-inicio.md` y `axiom.example.yaml` |

## Registro de riesgos para el apply

- **Tamaño y orden de revisión.** U3 (760), U1 (555) y U2 (520) son las unidades mayores; no se parten más porque son un solo paquete cohesionado y sus pruebas deben ir con el código.
- **Nombres de prueba.** Los patrones de `-run` de este plan son los nombres que el apply debe dar a las pruebas nuevas; si una prueba existente ya cubre el comportamiento con otro nombre, se ajusta el patrón, no el criterio.
- **Copias versionadas.** La regeneración de `AGENTS.md`, `GEMINI.md` y `.codex/AGENTS.md` sin compilar un binario no está garantizada (tarea 24.2).
