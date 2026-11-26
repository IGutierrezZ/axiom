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

## Pendientes

T3 a T10.
