# Progreso de apply: bateria-pruebas

## T1 completada: bloque `acceptance_tests` en `workspace`

- Creado `internal/workspace/acceptance_tests.go` (`AcceptanceTestsConfig`, `decodeAcceptanceTests`, `acceptanceTestsTopologyError`, `Declarations.EffectiveAcceptanceTests`) y su test.
- Cableado en `declarations.go`, `types.go`, `loader.go`, `validator.go` (paso 3b bis) y `hub/init.go` (`declares()`).
- Documentado en `axiom.example.yaml`, `docs/manual-de-inicio.md` y `docs/openspec-config.md`, con pines en `example_yaml_test.go` y `docs_contract_test.go`; tests de `hub` y `Locate`.
- Decisión: la clave `enabled` es obligatoria en el bloque (no se asume valor). Los duplicados de `environments` solo se exigen con `enabled: true`.
- Verificación: `go test ./internal/workspace ./internal/hub -run 'AcceptanceTests|Validate|ExampleYAML|DocsContract|Declares|Locate|Docs' -count=1` ok; `go vet` ok; `go build ./...` ok.

## T2 completada: ayudantes de `isoevidence` exportados

- Renombrado mecánico en `internal/isoevidence/{fence,matrix,bug,cubre,scenarios}.go`: `TextLine`, `SplitLines`, `IsTableLine`, `IsSeparatorLine`, `SplitCells`, `RawCellSpan`, con comentario de doc en cada símbolo. Sin cambio de comportamiento; los tests no usaban los nombres antiguos.
- Nota para el ratchet de código muerto: `IsTableLine`, `IsSeparatorLine`, `SplitLines`, `SplitCells` y `RawCellSpan` ya son alcanzables (los usa `isoevidence` internamente desde `cmd/axiom`); `TextLine` es tipo. Los consumidores externos llegan con T3.
- Verificación: `go test ./internal/isoevidence ./internal/livespec -count=1` ok; `go vet ./internal/isoevidence` ok; `go build ./...` ok.

## T3 completada: núcleo `internal/acceptancetests`

- Creado `internal/acceptancetests/{doc,model,parse,check,rewrite}.go` y tests (`helpers`, `parse`, `check`, `rewrite`, `imports`); cobertura 97,8 %.
- API exportada final: `FileName`; `Code` y 15 constantes `Code*`; `CubreEntry`, `Case`, `Battery`, `Finding`, `Coverage`, `Input`, `Report`; `Report.OK()`; `Parse(text) Battery`; `Check(text, Input) Report`; `RewriteCubre(text, reqid.IDMap) string`; `MissingFinding() Finding`. Las columnas son un detalle interno (`columns`, no exportado).
- Ratchet de código muerto: `Parse` lo usará T7, `Check`, `Report.OK` y `MissingFinding` T5 y T6, `RewriteCubre` T7. Nada se ha exportado que no vaya a usarse fuera del paquete; si T6 no usa `Report.OK`, eliminarlo en T10.
- Decisiones: `ID#` sin nombre y las entradas vacías se representan como `CubreEntry{}` (ID vacío) y dan `caso_sin_cubre`; una cita repetida se cuenta una vez; los escenarios con `;` cuentan en el total y solo dan `escenario_ambiguo`; un requisito con más de un ID se informa como `requisito_sin_id`; las filas con otro número de celdas se descartan de `Parse` y `RewriteCubre` por igual.
- Verificación: `go test ./internal/acceptancetests -count=1` ok; `go vet ./internal/acceptancetests` ok; `go build ./...` ok.

## T4 completada: digest de la puerta `tasks` con cálculo único

- Creado `internal/sddstatus/acceptance_tests.go` con `TasksGateArtifactPaths(changeRoot, roles)`: `tasks.md` o los `TasksFile` por rol, más `pruebas.md` solo si existe y no está en blanco. `governanceArtifactInputs` y `tasksArtifactPathsForDigest` (cli) delegan en ella; `internal/kickoff/machine.go` intacto. Los digests por rol y `role-apply` no incluyen `pruebas.md`.
- Tests: `TestTasksGateArtifactPaths` (sddstatus), `TestGateArtifactDigestTasksGateAcceptanceTests` (sin `pruebas.md` digest idéntico a `tasks.md`, en blanco fuera, editar cambia el digest, paridad registro/status, aprobada terminal) y `TestGateArtifactDigestTasksGateRejectedReopensOnAcceptanceTestsEdit` (rechazada pasa a pending reabierta).
- Verificación: `go test ./internal/sddstatus -run 'TasksGate|Governance'`, `go test ./internal/cli -run 'GateArtifactDigest|RunSDDGateRecord|GateDigest'`, `go test ./internal/kickoff -run EvaluateGates` ok; `go vet ./internal/sddstatus ./internal/cli` ok.

## T5 completada: `AcceptanceTestsFor` y proyección `acceptanceTests`

- En `internal/sddstatus/acceptance_tests.go`: `AcceptanceTestsFor(location workspace.Location, changeRoot string) AcceptanceTestsResult` (campos `Active`, `Environments`, `DeclarationError`, `Present`, `Missing`, `Error`, `Report`, `Findings`; `Findings` incluye `pruebas_ausente` si `Missing`), `AcceptanceTestsProjection` (`active`, `environments`, `error`, `warning`, `battery{path,cases,scenarios,requirements,findings,error}`) y `decorateAcceptanceTests`, enganchado en `Resolve` tras `decorateExplainer` y copiado en `status_v2.go`.
- Con la función no activa y `pruebas.md` presente se ejecuta `Check` con `SkipEnvironments`; con `changeRoot` vacío solo se lee la declaración; fallos de kickoff, patrón o delta van a `Error` sin informe parcial. Omitida para archivados y sin bloque ni `pruebas.md`.
- Contrato documentado en `internal/assets/skills/_shared/sdd-status-contract.md`.
- Tests: `acceptance_tests_projection_test.go` (14 casos: omisión con JSON idéntico, activa con batería, `enabled:false`, bloque inválido y topología, aviso sin cambiar enrutado, no activa con fichero, kickoff ilegible, archivado no bloqueado, archivados omitidos).
- Verificación: `go test ./internal/sddstatus -run 'AcceptanceTests|TasksGate|Governance|Explainer|Declarations|Requirements|StatusV2'`, `go test ./internal/assets -run StatusContract`, `go test ./internal/components -run TestGolden` ok; `go vet ./internal/sddstatus`; `go build ./...` ok.
- Para T6/T8: `AcceptanceTestsResult.Findings` ya trae `pruebas_ausente`; el verbo decide «antes de tasks» con `Missing == false && !Present`.

## T6 completada: verbo `axiom sdd tests check`

- Creado `internal/cli/sdd_tests.go` (`RunSDDTests` subdespachador, `runSDDTestsCheck`, esquema `axiom.sdd-tests-check/v1`; JSON con `active`, `environments`, `present`, `cases`, `scenarios`, `requirements`, `findings[{code,file,line,message}]`, `error?`, `notice?`). Registrado en `runSDD` y en la ayuda de `cmd/axiom/main.go`; `main_test.go` con `tests` en `expectedSubcmds` y `TestSDDTestsRouting`.
- Salida: `Report.OK()` y `Missing` deciden el código 1; sin delta o cambio inexistente es error de uso (la delta se comprueba antes, también sin `pruebas.md`); sin batería informa «no están activadas» o «todavía no hay batería» con 0; declaración inválida o `Error` salen 1 tras imprimir el informe (con bloque inválido y fichero presente se ejecuta el check y sale 1 igualmente por el error de declaración).
- Tests: `internal/cli/sdd_tests_test.go` (9 casos: texto/JSON, hallazgos, ausente, inactiva sin fichero, inactiva con fichero sin validar entornos, antes de tasks, errores de uso, declaración inválida, solo lectura).
- Verificación: `go test ./internal/cli -run SDDTests`, `go test ./cmd/axiom -run 'RunSDDHelp|SDDVerbRouting|SDDTests|RunSDDUnknown'` ok; `go vet ./internal/cli ./cmd/axiom`; `go build ./...` ok.

## T7 completada: archivado reescribe `Cubre` de `pruebas.md`

- `internal/livespec/change.go`: tercer `changeFileKind` (`kindAcceptanceTests`) para `acceptancetests.FileName`; `changeArtifacts` extrae como citas los provisionales de la parte ID de cada entrada de `Cubre` (línea de la fila, vía `acceptancetests.Parse`); la reescritura usa `acceptancetests.RewriteCubre`. Tabla inválida: sin citas, sin reescritura, sin error propio; provisional sin correspondencia lo rechaza `PlanIDs`. Comentario de `changeArtifacts` actualizado.
- Tests en `internal/livespec/change_acceptance_tests_test.go`: reescritura solo de `Cubre` (con `ID#nombre`, ID suelto, mezcla; byte a byte fuera, CRLF), idempotencia, definitivos y ausencia de fichero, provisional no declarado rechazado con la línea de la fila, tabla inválida.
- Verificación: `go test ./internal/livespec -run 'AcceptanceTests|ChangeArtifacts|PlanChange' -count=1` ok; `go vet ./internal/livespec` ok.

## T8 completada: dashboard, campo `acceptance_tests` y pestaña «Pruebas»

- `internal/dashboard/types.go`: `IncrementDetailDTO.AcceptanceTests *IncrementAcceptanceTestsDTO` (`acceptance_tests,omitempty`) y DTOs de cobertura, caso (nueve columnas y línea) y hallazgo. `service.go`: `incrementAcceptanceTests` (`workspace.Locate` + `sddstatus.AcceptanceTestsFor`; si `Locate` falla lleva `error` y el detalle se sirve). El campo es nil para archivados; con `Locate` correcto se informa para todo cambio SDD activo (la pestaña trata activo falso y sin fichero como no activado).
- Front de solo lectura: pestaña `pruebas` en `index.html`, `INCREMENT_TABS`, rama en `detalle.js` y módulo `detalle-pruebas.js` (resumen, aviso, hallazgos y tabla reutilizando `table-wrap`/`data-table`, cada celda con `escapeHtml`, sin literales de color). Sin ruta nueva.
- Tests: `acceptance_tests_test.go` (DTO igual al compartido, hallazgos y texto hostil literal, no activada, activa sin fichero con aviso en multirepo, archivado omitido, error de `Locate`), `TestAcceptanceTestsTab` y pines de pestañas/router/assets.
- Verificación: `go test ./internal/dashboard -run 'IncrementDetail|AcceptanceTests|Tabs|Router|Canary|AssetsAreServed|ExplainerTab' -count=1` ok; `go vet ./internal/dashboard`; `go build ./...` ok.

## T9 completada: doctrina de `sdd-tasks`

- Nuevo `_shared/sdd-acceptance-tests.md` (condición SDD y `acceptanceTests.active`, ODD nunca, nueve columnas con ejemplo, cuatro capas en orden, `axiom sdd tests check`, devolución en `artifacts`, ausencia, refinado de `sdd-apply` solo en `Pasos`); «Step 3b» en la skill `sdd-tasks`; sección `## Acceptance Tests` en los agentes claude, cursor y copilot; pines en `internal/assets/acceptance_tests_doctrine_test.go`; golden `sdd-claude-agent-sdd-tasks` regenerado.
- No se tocaron `sdd-orchestrator-sections.md` ni la skill `sdd-apply`. Riesgo anotado: la frase de la parada `tasks` del orquestador (RF-GAT-32) queda fuera; se cumple vía skill y status.
- Verificación: `go test ./internal/assets -run 'AcceptanceTestsDoctrine|TestOrchestratorSectionDriftDoesNotGrow|Explainer'` ok; goldens con y sin `-update` ok; `go vet ./internal/assets` ok.

## Pendientes

T10.
