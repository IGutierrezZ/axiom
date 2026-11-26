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

## Pendientes

T6 a T10.
