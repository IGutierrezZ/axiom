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

## Pendientes

T4 a T10.
