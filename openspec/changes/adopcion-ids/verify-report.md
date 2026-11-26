# Informe de verificación · adopcion-ids (fullstack)

- **Fecha:** 2026-11-26
- **Rol:** `fullstack`. Su verify es el del cambio.
- **Alcance:** diff del rol, `7a6aa9f6..c9780134`.
- **Commits:**
  - U1 `ae3709d4`
  - U2 `5f0d0b7e`
  - U3 `280fa4d6` y `2cfd35c4`
  - U4 `b667b70f`
  - U5 `ea086e52`
  - U6 `06544055`
  - Evidencia: `c9780134`

Veredicto: 0 CRITICAL, 0 WARNING, 2 SUGGESTION.

## Cumplimiento técnico

Tareas: 37/37 marcadas `[x]`, con evidencia RED/GREEN anotada por unidad en `tasks.md`.

Pruebas localizadas. Todas terminaron con código de salida 0:

| Comando | Resultado |
|---|---|
| `go test ./internal/adoption/...` | ok |
| `go test ./internal/reqid` | ok |
| `go test ./internal/livespec` | ok |
| `go test ./internal/assets -run 'SDDSpecAdoption\|SDDSpecAndPropose'` | ok |
| `go test ./internal/components -run TestGolden` | ok, sin `-update` |
| `go test ./internal/workspace -run DocsPin` | ok |
| `go test ./internal/cli -run 'AdoptIDs\|ArchiveAdopt\|IDPattern\|NoRequirementIDFormat\|ArchiveIntegrate'` | ok |
| `go vet` de adoption, livespec, cli, assets, workspace y reqid | limpio |
| `gofmt -l internal` | vacío |

No se ejecutó `go test ./...`: la batería completa la pasa la CI. Tampoco se compiló ningún binario.

Código muerto: todas las funciones exportadas nuevas tienen al menos una llamada de producción.

| Función | Llamadas de producción |
|---|---|
| `ProposePattern` | `internal/adoption/scan.go:155`, `internal/adoption/adoptcheck/check_ids.go:162` |
| `SchemeConflicts` | `check_ids.go:204` |
| `UnitConflicts` | `check_ids.go:168`, `:189` |
| `LookupIdentity` | `check.go:280` |
| `IdentityIDs` | `internal/cli/archive_adopt.go:167`, `check_ids.go` |

`rg "hasta C-03-c|RuleSeries|SeriesRefs" internal docs` no devuelve resultados.

## Criterio de terminado

- **Origen `RF-001…RF-120`.** `TestAdoptIDsCriterion` (`internal/cli/archive_adopt_ids_test.go:119`) comprueba que se conservan, y en `:165` que el siguiente requisito integrado recibe `RF-121`. `scan` no escribe `axiom.yaml`.
- **Origen sin IDs.** `TestAdoptIDsOriginWithoutIDsUsesTheDefaultPattern` comprueba que se usa el patrón por defecto, mediante provisionales.
- **`check` falla con duplicados o con IDs fuera del patrón.** Lo cubren `TestCheck_ForeignIDConflict_*`, `TestCheckFormatoDelTitulo`, `TestCheck_UnicidadPorClave` y `TestAdoptIDsPatternUndeclaredFails`.

## Matriz requisito → pruebas

| Requisito | Pruebas |
|---|---|
| REQ-30.9 | `internal/adoption/pattern_test.go` (`TestProposePattern*`, `TestIdentityIDs`); `scan_test.go` (`TestScan_*`, `TestInventory_LectoresV1YElCampoNuevo`) |
| REQ-30.10 y REQ-30.11 | `adoptcheck/check_test.go` (formato, unicidad por clave); `adoptcheck/check_ids_test.go` (`origin_id`, `pattern_undeclared`, `foreign_id_conflict`) |
| REQ-30.12 | `adoptcheck/check_pattern_test.go`; `TestArchiveAdoptCheck*` |
| REQ-30.13 | `TestAdoptIDs*` (archivado con `archive-integrate`) |
| REQ-44.4 | `livespec/registry_pattern_test.go`; `TestRegistryLookupIdentity*` |
| REQ-30.31 | `TestArchiveAdoptHelpDescribesTheDeclaredPattern`. Los pins de la guía (`assets_test.go`) y del manual (`TestDocsPinManualAdopcionConservaLosIDs`) se retiraron al integrar `main`, en coherencia con C-08-h (#333), que borró los centinelas de redacción |

La matriz es por requisito: no se revisó cada escenario de la delta frente a su prueba línea a línea.

## Riesgos del design

- **R1.** `IdentityIDs` (`internal/adoption/pattern.go:106`) toma los IDs del título y, si el título no trae ninguno, los del cuerpo. Las pruebas de los escenarios de la delta siguen en verde.
- **R2.** `TestCheck_PatternUndeclared_UnIDNoApareceEnDosMotivos` cubre el caso en que coinciden los motivos `format` y `pattern_undeclared`.
- **R4.** Reglas sin escenario en la delta, pero con prueba:
  - La exclusión de las unidades descartadas está en `check_ids.go:143-152` y la prueba `TestCheck_PatternUndeclared_IgnoraLasUnidadesDescartadas`.
  - `foreign_id_conflict` está en `check_ids.go:190`, con sus pruebas.
  - El paso 4 de `origin_id` está en `check_ids.go:116` y `:122-132`, con `TestCheck_OriginID_FusionDeUnidades` y `TestCheck_OriginID_DivisionConElMismoID`.
- **Fusión (tarea 3.3a).** `check_ids.go:124-131` cuenta solo los IDs propios de las unidades citadas. `TestCheck_OriginID_ConVariosOrigenesBastaUnoQueCoincida` sigue en verde.
- **Ajuste `2cfd35c4`.** `check_ids.go:95` acepta como `<ID original>` un ID para el que `c.pattern.Valid(written)` es cierto. Lo prueba `TestCheck_OriginID_IDDelPatronEfectivoCuentaComoID`.

## Hallazgos

- **SUGGESTION.** El escenario de separadores mixtos de la spec usa `RF.001`, pero `foreignIDPattern` (`internal/adoption/scan.go:34`) solo admite `-` y `_`. La prueba usa `RF_001`. El texto del escenario se corrige al archivar, como recoge «Notas para el archive» de `tasks.md`.
- **SUGGESTION.** La colisión por cadena en `archive-integrate` (`internal/livespec/ids.go:427`, riesgo R3 del design) queda fuera de alcance. Con `RF-<m>`, un `RF-001` frente a un `RF-1` vivo no se detectaría al archivar. Se propone como seguimiento.

**Estado:** limpio
