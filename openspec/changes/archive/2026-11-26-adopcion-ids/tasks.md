# Tareas: la adopción conserva los IDs del cliente y valida contra el patrón declarado (C-03-c)

Rol único `fullstack` (`kickoff.yaml`, `tasks_file: tasks.md`). Fuentes: `proposal.md`, las tres deltas de `specs/` (MODIFIED REQ-30.9, REQ-30.10, REQ-30.11, REQ-30.12, REQ-30.13, REQ-44.4 y REQ-30.31), `design.md` (D1 a D10, unidades U1 a U6, riesgos R1 a R6, tabla de resolución de cobertura) y `coverage-review.spec.md` / `coverage-review.design.md`.

Normas de ejecución para todo el plan:

- Modo TDD: `strict_tdd: false` en `openspec/config.yaml` (líneas 16 y 42) y runner `go test`. El design (Estrategia de pruebas) exige igualmente observar el RED antes de la producción, así que cada unidad abre con sus pruebas en RED y después viene el GREEN; REFACTOR solo con todo en verde. Se anota el RED observado como evidencia.
- Verificación localizada, nunca la batería completa: no se ejecuta `go test ./...` ni `go test ./internal/cli` entero; en `internal/cli`, `internal/assets` y `internal/components` siempre con `-run`. Tampoco se compilan binarios sueltos de `axiom` (`go build -o`): los arneses corren en proceso con `go test`.
- Cada unidad cierra con un único commit convencional, sin atribución a IA, con código, pruebas y documentación de la misma conducta juntos. Los ficheros de `openspec/specs/` no se tocan aquí: lo hace el archive (ver «Notas para el archive»).
- Todas las rutas son del worktree `C:\repos\axiom-wt\adopcion-ids\axiom` (rama `inc/adopcion-ids`); en las tareas se escriben relativas a la raíz del repositorio. Las rutas con `(read-only)` solo se leen.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | L |
| Estimated changed lines | 3.000 a 4.500 (adiciones + eliminaciones, sin ficheros generados; cerca de dos tercios son pruebas) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| U1 | Inferencia del patrón: `IdentityIDs`, `ProposePattern`, conflictos, campo `proposed_id_pattern` rellenado por `Scan` | `feat(adoption): proponer el patrón de identificadores del cliente al inventariar` | `go test ./internal/adoption/...` | N/A: biblioteca sin comando propio; la cubren las pruebas de tabla y `Scan` sobre árbol temporal | `internal/adoption/{pattern,inventory,scan}.go` y sus pruebas |
| U2 | `check` mide con el patrón efectivo: firma, `projectIDPattern`, registro con `Pattern`, formato, unicidad por clave, retirada de `series` y de `SeriesRefs` | `feat(adoptcheck): medir el cambio de adopción con el patrón efectivo del proyecto` | `go test ./internal/adoption/adoptcheck` y `go test ./internal/livespec -run 'Registry\|Lookup'` y `go test ./internal/cli -run 'ArchiveAdopt'` | `RunArchiveAdopt check` en proceso sobre fixture de `t.TempDir()` | `internal/livespec/registry.go`, `internal/adoption/adoptcheck/*`, tramo de `check` de `internal/cli/archive_adopt.go` |
| U3 | Conservación del ID: `origin_id`, `pattern_undeclared`, `foreign_id_conflict` | `feat(adoptcheck): conservar el ID del cliente y exigir el patrón declarado` | `go test ./internal/adoption/adoptcheck` | `Check` en proceso sobre inventarios de fixture | `internal/adoption/adoptcheck/{check.go,check_ids_test.go}` |
| U4 | Textos de la CLI: bloque del patrón en `scan`, patrón declarado frente al propuesto, ayuda | `feat(cli): mostrar el patrón propuesto y el declarado en archive adopt scan` | `go test ./internal/cli -run 'ArchiveAdopt\|IDPattern'` | `RunArchiveAdopt scan` y `scan --change` en proceso, texto y `--json` | `internal/cli/archive_adopt.go` y pruebas de CLI |
| U5 | Prueba de integración del criterio `RF-001…RF-120` → `RF-121` y del origen sin IDs | `test(cli): integrar adopción con IDs del cliente conservados y provisionales` | `go test ./internal/cli -run 'AdoptIDs'` | `RunArchiveAdopt` + `RunSDDArchiveIntegrate` en proceso sobre árbol temporal | `internal/cli/archive_adopt_ids_test.go` |
| U6 | Guía, manual y pins: `openspec-adoption.md`, `assets_test.go`, `docs/manual-de-inicio.md`, goldens, paridad entre runtimes | `docs(spec): guía de adopción con IDs conservados y patrón declarado` | `go test ./internal/assets -run 'SDDSpecAdoption\|SDDSpecAndPropose'` y `go test ./internal/components -run TestGolden` y `go test ./internal/workspace -run DocsPin` | N/A: solo texto; los pins comprueban el contenido renderizado | Textos de la guía, manual y pins |

Entrega `single-pr`: los seis commits forman una sola PR de código; la PR del archive va aparte. Orden: U1 y U2 no dependen entre sí (una tras otra, U1 primero); U3 depende de U1 y U2; U4 de U1 y U2; U5 de U1 a U4; U6 de U3 y U4.

---

## Fase 1 (U1): Inferencia del patrón y campo del inventario

Depende de: nada. Paquete: `internal/adoption`.

Evidencia U1: RED observado = `go test ./internal/adoption -run 'ProposePattern|IdentityIDs|Scan_Propone|Inventory_Lectores'` sin `pattern.go` falla de compilación (`undefined: ProposePattern`, `PatternEvidence`, `ConflictMixedPrefixes`, `ownedID`…). GREEN: `go test ./internal/adoption/...` ok y `go vet ./internal/adoption/...` limpio. Límite de ancho: `foreignIDPattern` admite `d{1,5}`, así que `<NNNNNN>` (k=6) es inalcanzable y se prueba como tal.

- [x] 1.1 RED: crear `internal/adoption/pattern_test.go` con la tabla de inferencia de D2, una fila por caso: `RF-001…RF-120` → `RF-<NNN>`; `RF-01…RF-45` → `RF-<NN>`; `RF-01`+`RF-100` → `RF-<NN>` (desbordamiento); `RF-1…RF-9` → `RF-<m>`; `RF-10…RF-99` → `RF-<m>`; `RF-1`,`RF-2`,`RF-010` → `RF-<m>`; `RF-001`+`RF-01` → `RF-<m>`; un único `RF-001` → `RF-<NNN>` y `RF-7` → `RF-<m>`; `RF-USU-01`/`RF-FAC-02` → `RF-<DOM>-<NN>`; `RF-USU-1`/`RF-FAC-010` → `RF-<DOM>-<m>`; `RF_001` → `RF_<NNN>`; `REQ-12`,`REQ-13` → `REQ-<m>`. Añade las filas de borde `RF-000` (número 0 con relleno → `RF-<NNN>`, y `RF-0`/`RF-000` mezclados → `RF-<m>`) y ancho 5 (`RF-00001`…`RF-00120` → `RF-<NNNNN>`; `RF-000001` con k=6 solo si `reqid` lo admite, anotando el límite superior). Prueba que `reqid.Parse(plantilla)` acepta la plantilla y `Valid` acepta todos los IDs de cada fila.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/adoption -run 'ProposePattern'` falla (RED observado)
- [x] 1.2 RED: en el mismo fichero, conflictos de D3 con su evidencia `{dimension, value, count}` en orden fijo (`prefix`, `separator`, `domain`, `width`): `RF-001`+`RNF-001` → `mixed_prefixes`; `RF-001`+`RF-USU-01` → `mixed_domain`; `RF-001`+`RF_002` → `mixed_separators` (el escenario de la spec escribe `RF.001`, imposible con `foreignIDPattern`: se usa `RF_001`; ver «Notas para el archive»). Con conflicto de esquema no hay plantilla y `Template` queda vacío. Prueba que `template_invalid` es inalcanzable por la vía normal (defensa) llamando a la función interna con un esquema forzado.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/adoption -run 'ProposePattern|Conflict'` falla (RED)
- [x] 1.3 RED: en el mismo fichero, `IdentityIDs` (D1): un ID en el título (el cuerpo no cuenta aunque cite otros); título sin ID y cuerpo con uno (cuenta el del cuerpo); varios IDs propios; ninguno; fichero entero sin secciones; origen binario (una unidad sin IDs). Conflictos de unidad: unidad con varios IDs propios → `ambiguous_unit_id` (la plantilla se sigue proponiendo); el mismo ID en dos unidades o en dos orígenes → `duplicate_foreign_id`, comparado por (serie, número) con plantilla (`RF-1` y `RF-001` con `<m>` se repiten) y por cadena sin ella. Determinismo: dos ejecuciones sobre las mismas unidades dan la misma propuesta byte a byte; sin IDs propios `ProposePattern` devuelve nil.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/adoption -run 'IdentityIDs|ProposePattern'` falla (RED)
- [x] 1.4 GREEN: crear `internal/adoption/pattern.go` con las constantes `Conflict*`, `PatternProposal`, `PatternEvidence`, `PatternConflict`, `IdentityIDs(u Unit) []string`, `ProposePattern(sources []Source) *PatternProposal`, `SchemeConflicts` y `UnitConflicts`. La expresión de forma `^([A-Z]+)([-_])(?:([A-Z]{2,5})([-_]))?([0-9]{1,5})$`, la elección `<N×k>`/`<m>` de D2 (k entre 2 y 6, la gramática real de `internal/reqid/template.go:108` y su mensaje en `:125-127`) y la validación final con `reqid.Parse`/`Valid`.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/adoption -run 'IdentityIDs|ProposePattern|Conflict'`
- [x] 1.5 RED y GREEN: añadir `ProposedIDPattern *PatternProposal` (`yaml`/`json`: `proposed_id_pattern,omitempty`) a `Inventory` en `internal/adoption/inventory.go` y rellenarlo al final de `Scan` en `internal/adoption/scan.go` (sin leer `axiom.yaml`). En `internal/adoption/scan_test.go`: propuesta `RF-<NNN>` con 120 unidades; `RF-<m>`; conflicto `RF`/`RNF`; sin IDs no hay campo; dos `Scan` idénticos dan el mismo inventario; un inventario con el campo se lee con un struct v1 sin error (lector no estricto); un inventario v1 sin el campo se lee con `ProposedIDPattern == nil`.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/adoption -run 'Scan|Inventory'`
- [x] 1.6 RED y GREEN: prueba de que `proposed_id_pattern` NO altera el SHA-256 de `inventory_stale` ni `Summarize`: el mismo inventario con y sin el campo (y editando `axiom.yaml`) produce el mismo resultado de obsolescencia. Se prueba en `internal/adoption/adoptcheck/check_test.go` o en `internal/adoption/inventory_test.go` según donde viva la comparación de hashes (`check.go:368-385`); si el campo la alterase, corregir la producción.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/adoption/... -run 'Stale|Summarize|Hash'`
- [x] 1.7 Cierre de U1: `go vet ./internal/adoption/...` y commit `feat(adoption): proponer el patrón de identificadores del cliente al inventariar`.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/adoption/...`

## Fase 2 (U2): `check` mide con el patrón efectivo

Depende de: nada (se ejecuta tras U1). Paquetes: `internal/livespec`, `internal/adoption/adoptcheck`, `internal/cli`.

Evidencia U1: commit `ae3709d4`.

Evidencia U2: RED observado = `go test ./internal/livespec -run Lookup` falla de compilación (`reg.LookupIdentity undefined`) y `go test ./internal/adoption/adoptcheck` no compila con la firma nueva de `Check(loc, change, pattern)` y la fixture con `RF-<NNN>`. GREEN: `go test ./internal/adoption/...` ok; `go test ./internal/reqid` ok; `go test ./internal/livespec -run 'Registry|Lookup|Check'` ok; `go test ./internal/cli -run 'Adopt|IDPattern'` ok; `go vet` de `internal/cli`, `internal/livespec` e `internal/adoption` limpio; `rg -n "RuleSeries|SeriesRefs" internal` sin resultados. Las tareas 2.2 y 2.7 se hicieron juntas (el índice `byKey` y la retirada de `SeriesRefs` van en el mismo cambio de `registry.go`); la cobertura de serie de `SeriesRefs` pasó a `LookupIdentity` (`registry_test.go`, `registry_pattern_test.go`). `Compact()` de `RF-<NNN>` es `RF-NNN` (sin ángulos): los mensajes de `format` citan `(RF-NNN)` y las pruebas lo fijan.

- [x] 2.1 RED: en `internal/livespec/registry_test.go`, `LookupIdentity`: con `RF-<m>` encuentra `RF-1` desde `RF-001`; con `RF-<NNN>` no encuentra `RF-1` (formato no válido); las referencias de cambios activos cuentan; `ExcludeChange` se respeta; orden estable (ruta y después ID); con el patrón por defecto se comporta como `Lookup` para `REQ-n.m`.
  Cubre: REQ-30.11, REQ-44.4
  Verificación: `go test ./internal/livespec -run 'Lookup'` falla (RED)
- [x] 2.2 GREEN: en `internal/livespec/registry.go`, añadir el índice `byKey map[identityKey][]IDRef` (llenado en `add`) y `func (r *Registry) LookupIdentity(id string) []IDRef`; `Lookup` no cambia.
  Cubre: REQ-30.11, REQ-44.4
  Verificación: `go test ./internal/livespec -run 'Registry|Lookup'`
- [x] 2.3 RED: reescribir `internal/adoption/adoptcheck/check_pattern_test.go` (invierte `:23` y `:38`): con `sdd.id_pattern` declarado, `Check` mide con él (un `(RF-001)` con `RF-<NNN>` pasa `format`); con patrón inválido no cae al defecto; sin declaración mide con `REQ-<n>.<m>` y el informe lo dice (`id_pattern_declared == false`). Ajusta `helpers_test.go` (fixture base con `RF-<NNN>`, `(RF-001)`/`(RF-002)`) y reescribe `MezclaDeSeries`, `SerieYaExistenteEnLasSpecs` y `UnaSolaSerieNuevaNoFalla` (`check_test.go:186-202`) como «ya no hay regla de serie».
  Cubre: REQ-30.11, REQ-30.12, REQ-44.4
  Verificación: `go test ./internal/adoption/adoptcheck` falla (RED)
- [x] 2.4 RED: casos de formato de D4 en `internal/adoption/adoptcheck/check_test.go`: definitivo válido; provisional compatible (`prov::<k>` con `RF-<NNN>`, `prov:<DOM>:<k>` con dominio, `prov:<n>:<k>` o etiqueta con el patrón por defecto); provisional incompatible (`prov:40:<k>` con `RF-<NNN>`) → `format` con `Reason` y `Code` de `reqid.ProvisionalError`; etiqueta con `RF-<NNN>` → `provisional_new_series_unsupported`; título sin ninguno → `format` nombrando el requisito, el último `(…)` del título, `pattern.Compact()` y `reqid.ProvisionalText("<clave>", "<ordinal>")`; título con definitivo y provisional a la vez → `format` por ambiguo.
  Cubre: REQ-30.10, REQ-30.11
  Verificación: `go test ./internal/adoption/adoptcheck -run 'Format|Provisional'` falla (RED)
- [x] 2.5 RED: casos de unicidad de D4: dos ADDED con la misma clave (`RF-1` y `RF-001` con `<m>`) → `uniqueness` nombrando ambos y la clave («serie `RF-*`, número 1»); dos requisitos con el mismo ID conservado (fusión: dos unidades del cliente que acaban en un único requisito, o dos requisitos que citan el mismo ID) → `uniqueness` con mensaje que sugiere usar un provisional para uno de ellos; dos provisionales con el mismo token → `uniqueness`; ADDED contra el registro con `LookupIdentity` (un `RF-001` vivo y otro `RF-001` ADDED falla; un definitivo libre pasa).
  Cubre: REQ-30.11, REQ-30.12
  Verificación: `go test ./internal/adoption/adoptcheck -run 'Uniqueness'` falla (RED)
- [x] 2.6 GREEN: en `internal/adoption/adoptcheck/check.go`, nueva firma `Check(loc workspace.Location, change string, pattern reqid.Pattern) (CheckReport, error)`; retirar `legacy` (`:24-26`), `RuleSeries` (`:32`) y el bloque de serie (`:200-232`); `LoadRegistry` con `Pattern: pattern` (`:183`); formato con `pattern.InTitle` y `ParseProvisional` (D4 pasos 1 a 4); unicidad con `idKey{reqid.Series; int}` por `pattern.Series` y `pattern.Number` y `reg.LookupIdentity`; `CheckReport` con `IDPattern` e `IDPatternDeclared` (`loc.Declarations.SDD != nil && loc.Declarations.SDD.IDPattern != nil`); el texto del informe abre con «Patrón de identificadores: `<plantilla>` (declarado en axiom.yaml | por defecto)».
  Cubre: REQ-30.10, REQ-30.11, REQ-30.12, REQ-44.4
  Verificación: `go test ./internal/adoption/adoptcheck`
- [x] 2.7 Retirar `SeriesRefs`: tras 2.6 su único consumidor de producción (`adoptcheck/check.go:219`) desaparece y el trinquete de código muerto de CI falla con funciones sin llamadas. Eliminar `func (r *Registry) SeriesRefs` (`internal/livespec/registry.go:152`) y sus pruebas (`registry_test.go:173-174` y `:189-206`, `registry_pattern_test.go:72-74`), reubicando la cobertura de serie que aporten a `LookupIdentity` si no queda cubierta. Comprobar con `rg -n SeriesRefs internal` que no queda ninguna referencia.
  Cubre: REQ-44.4
  Verificación: `go test ./internal/livespec -run 'Registry|Lookup'` y `go vet ./internal/livespec/...`
- [x] 2.8 RED y GREEN: en `internal/cli/archive_adopt.go` (tramo de `check`, `:253-256`), obtener el patrón con `projectIDPattern("archive adopt check", loc)` antes de llamar a `Check`, de modo que un patrón inaccesible o inválido dé exactamente el error de `idPatternRefusal` (`idpattern.go:45-51`). En `internal/cli/archive_adopt_test.go`: `check` con el patrón declarado; con patrón inválido (mismo error y continuación que el resto de verbos); el JSON incluye `id_pattern` e `id_pattern_declared`; `check` no escribe (hash del árbol, `axiom.yaml` incluido).
  Cubre: REQ-30.11, REQ-44.4
  Verificación: `go test ./internal/cli -run 'ArchiveAdopt'`
- [x] 2.9 Cierre de U2: `go vet ./internal/livespec/... ./internal/adoption/... ./internal/cli` y commit `feat(adoptcheck): medir el cambio de adopción con el patrón efectivo del proyecto`. Comprobar `rg -n "RuleSeries" internal` sin resultados.
  Cubre: REQ-30.11, REQ-30.12, REQ-44.4
  Verificación: `go test ./internal/adoption/... ./internal/livespec -run 'Registry|Lookup|Check'`

## Fase 3 (U3): Conservación del ID del cliente

Depende de: U1, U2. Paquete: `internal/adoption/adoptcheck`.

Evidencia U3: RED observado = `go test ./internal/adoption/adoptcheck -run 'OriginID|PatternUndeclared|ForeignIDConflict|InventoryCorrupt'` falla de compilación (`c.checkOriginIDs`, `RuleOriginID` sin definir) antes de la producción. GREEN: `go test ./internal/adoption/...` ok, `go vet ./internal/adoption/...` limpio, `go test ./internal/cli -run 'ArchiveAdopt|IDPattern|NoRequirementIDFormat'` ok. Código en `check_ids.go` (misma paquete que `check.go`). Interpretación: la regla de fusión (3.3a) cuenta solo los IDs propios de las unidades citadas, no los `<ID original>` escritos, así que el escenario «con varios orígenes basta que el título coincida» sigue verde cuando el segundo `Origen:` apunta a una unidad sin ID. Los fixtures de `internal/cli/archive_adopt_test.go` pasaron a `RF-001`/`RF-002` con `RF-<NNN>` declarado (con `REQ-40.n` y origen `RF-n` el nuevo `origin_id` los rechaza). Commit: `280fa4d6`. Ajuste posterior (commit `fix(adoptcheck)`): un `<ID original>` cuenta como ID si lo reconoce `foreignIDPattern` o si es un ID definitivo válido del patrón efectivo (`REQ-40.1` con el defecto); RED = `TestCheck_OriginID_IDDelPatronEfectivoCuentaComoID` fallaba con «un único identificador o ninguno»; GREEN tras `c.pattern.Valid(written)` en `check_ids.go`.

- [x] 3.1 RED: crear `internal/adoption/adoptcheck/check_ids_test.go` con los pasos de `origin_id` (D6): (2) `Origen:` con `(RF-001)` sobre una unidad cuyo ID propio es `RF-002` → fallo nombrando el `Origen:`, el ID escrito y el de la unidad; (3) provisional con ID esperado → «conserva el identificador del cliente `RF-001`»; (4) definitivo con unidad sin ID y todas las unidades citadas en el inventario → «usa un provisional»; (5) definitivo que no coincide con ningún elemento de `E`, nombrando el del título y todos los de `E`; con varios `Origen:` basta uno que coincida; la comparación por clave (`RF-1` frente a `RF-001` con `RF-<m>`) y por cadena exacta cuando algún ID no es válido; sin inventario legible solo se aplican 2, 3 y 5 con los `OriginalID`.
  Cubre: REQ-30.11
  Verificación: `go test ./internal/adoption/adoptcheck -run 'OriginID'` falla (RED)
- [x] 3.2 RED: `Origen:` con más de un ID o con texto no-ID. Comportamiento determinista fijado aquí: `Origen: docs/x.md#a (RF-001, RF-002)` y `Origen: docs/x.md#a (n/d)` → rechazo tipado `origin_id` cuyo mensaje pide «un único identificador o ninguno» y cita el texto recibido; no se interpreta ni se descarta en silencio. Un `Origen:` sin paréntesis no trae `OriginalID` (E se calcula solo con la unidad).
  Cubre: REQ-30.10, REQ-30.11
  Verificación: `go test ./internal/adoption/adoptcheck -run 'OriginID'` falla (RED)
- [x] 3.3 RED: fusión y división. (a) Un requisito que cita dos unidades con IDs distintos (`RF-001` y `RF-002`) y lleva un definitivo que coincide con uno solo → `origin_id` con mensaje que nombra los IDs no conservados (`RF-002`) y sugiere una de las vías (un requisito por unidad o provisional); (b) dos unidades que se dividen en dos requisitos con el mismo ID conservado → fallo de unicidad (de 2.5) con sugerencia de provisional para uno.
  Cubre: REQ-30.10, REQ-30.11
  Verificación: `go test ./internal/adoption/adoptcheck -run 'OriginID|Merge|Split'` falla (RED)
- [x] 3.4 RED: `pattern_undeclared` (D7). Un fallo de proyecto (sujeto `sdd.id_pattern`) cuando algún ID propio no es `Valid` para el patrón efectivo; el motivo nombra el patrón efectivo y su origen, el número de IDs no reconocidos con hasta tres ejemplos y, con plantilla propuesta, «copia en axiom.yaml las líneas `sdd:` e `  id_pattern: "RF-<NNN>"` … en el repositorio donde se trabaja la adopción y vuelve a ejecutar `adopt check`». El texto es válido también en multirepo (repositorio de specs en su rama de integración), no solo «rama del cambio»: dice «en la rama de trabajo de la adopción (la del cambio, o la de integración del repositorio de specs en multirepo)». Con conflicto de esquema: los conflictos con su recuento y «normaliza a mano los identificadores del origen (Axiom no los reescribe), vuelve a generar el inventario con `<scanCommand>`». Un mismo ID ajeno nunca aparece en dos motivos de proyecto. Las unidades de `## Descartado` (con o sin motivo) quedan fuera y los IDs de unidades descartadas no cuentan ni para la propuesta recalculada ni para `pattern_undeclared`. Con el patrón declarado que cubre los IDs, no falla. Inventario v1 sin `proposed_id_pattern` funciona (se recalcula).
  Cubre: REQ-30.9, REQ-30.11
  Verificación: `go test ./internal/adoption/adoptcheck -run 'PatternUndeclared'` falla (RED)
- [x] 3.5 RED: `foreign_id_conflict` (D7). Un fallo por cada conflicto de unidad: unidad con varios IDs propios → `foreign_id_conflict` con la unidad como sujeto y continuación «normaliza el origen o descarta la unidad» (normalizar o descartar); el mismo ID en dos unidades; los IDs de unidades descartadas no cuentan; los conflictos de esquema NO aparecen aquí (solo en `pattern_undeclared`). Inventario ilegible o corrupto: un único fallo tipado (`inventory_missing`/regla existente), sin errores en cascada de `origin_id`, `pattern_undeclared` ni `foreign_id_conflict`; unidad citada inexistente en el inventario → fallo de la regla `origin` existente y no de `origin_id`.
  Cubre: REQ-30.11
  Verificación: `go test ./internal/adoption/adoptcheck -run 'ForeignIDConflict|InventoryCorrupt'` falla (RED)
- [x] 3.6 GREEN: en `internal/adoption/adoptcheck/check.go` añadir `RuleOriginID`, `RulePatternUndeclared` y `RuleForeignIDConflict`; leer el inventario una sola vez (`c.inv`, `c.invErr`) e indexarlo por `<ruta>#<ancla>`; implementar `origin_id` (D6 pasos 1 a 5 y los rechazos de 3.2 y 3.3), el cálculo de `pattern_undeclared` y `foreign_id_conflict` con `adoption.ProposePattern` sobre las unidades no descartadas; el orden del informe de D7 (`format`, `uniqueness`, `scenarios`, `origin`, `origin_id`, `coverage`, `discard_reason`, `inventory_missing`/`inventory_stale`, `pattern_undeclared`, `foreign_id_conflict`); si el inventario no se lee, un único fallo y se omiten las reglas que dependen de él.
  Cubre: REQ-30.10, REQ-30.11
  Verificación: `go test ./internal/adoption/adoptcheck`
- [x] 3.7 Cierre de U3: `go vet ./internal/adoption/...` y commit `feat(adoptcheck): conservar el ID del cliente y exigir el patrón declarado`.
  Cubre: REQ-30.10, REQ-30.11
  Verificación: `go test ./internal/adoption/...`

## Fase 4 (U4): Textos de la CLI

Depende de: U1, U2. Paquete: `internal/cli`.

Evidencia U4: RED observado = `go vet ./internal/cli` falla de compilación (`report.DeclaredIDPattern undefined`) con los tests nuevos de `archive_adopt_test.go` y la ayuda invertida en `archive_idpattern_test.go` (`TestArchiveAdoptHelpDescribesTheDeclaredPattern`, antes `…IsUnchangedByTheDeclaredPattern`). GREEN: `go test ./internal/cli -run 'ArchiveAdopt|IDPattern|NoRequirementIDFormat'` ok, `go vet ./internal/cli` limpio, `rg -n "hasta C-03-c|RuleSeries" internal` sin resultados. `adoptScanText` recibe ahora el `adoptScanReport` entero. Evidencia U3: commit `280fa4d6`. Commit U4: `b667b70f`.

- [x] 4.1 RED: en `internal/cli/archive_adopt_test.go`, texto y JSON del bloque del patrón de `scan` (D8): plantilla, evidencia, líneas de copia `sdd:` / `id_pattern: "RF-<NNN>"`, conflictos; sin plantilla, «sin plantilla: normaliza a mano…»; sin IDs, sin bloque; `declared_id_pattern` con `{template, matches_proposal, uncovered[]}` cuando `axiom.yaml` declara (coincide / difiere / no reconoce N IDs); `declared_id_pattern_error` y continuación `axiom workspace validate` cuando `IDPattern()` falla, sin negarse; `scan --change` deja `axiom.yaml` byte a byte igual y no hay opción `--write-pattern`.
  Cubre: REQ-30.9
  Verificación: `go test ./internal/cli -run 'ArchiveAdoptScan'` falla (RED)
- [x] 4.2 RED: invertir `internal/cli/archive_idpattern_test.go:136`: la ayuda de `archive adopt` no contiene «de una serie nueva» y cita `sdd.id_pattern`; la ayuda de `check` lista `origin_id` y `pattern_undeclared`; la ayuda de `scan` menciona que propone el patrón «sin escribir axiom.yaml»; sin «hasta C-03-c».
  Cubre: REQ-30.9, REQ-30.11, REQ-44.4
  Verificación: `go test ./internal/cli -run 'IDPattern|ArchiveAdoptHelp'` falla (RED)
- [x] 4.3 GREEN: en `internal/cli/archive_adopt.go`, `adoptScanReport` con `DeclaredIDPattern` y `DeclaredIDPatternError`, tipo `adoptDeclaredPattern`, bloque de texto tras los candidatos a `technical_context`, lectura de `loc.Declarations` solo para el informe; ayuda (`:55-57`) construida con `reqid.Default().Compact()` y `reqid.ProvisionalText("<clave>", "<ordinal>")` para respetar `TestNoRequirementIDFormatOutsideReqid`; quitar el comentario «hasta C-03-c» de la CLI y el de `adoptcheck/check.go:24-25` si aún existiera.
  Cubre: REQ-30.9, REQ-30.11, REQ-44.4
  Verificación: `go test ./internal/cli -run 'ArchiveAdopt|IDPattern|NoRequirementIDFormat'`
- [x] 4.4 Cierre de U4: `go vet ./internal/cli` y commit `feat(cli): mostrar el patrón propuesto y el declarado en archive adopt scan`.
  Cubre: REQ-30.9, REQ-30.11
  Verificación: `go test ./internal/cli -run 'ArchiveAdopt|IDPattern'`

## Fase 5 (U5): Prueba de integración del criterio

Depende de: U1 a U4. Fichero: `internal/cli/archive_adopt_ids_test.go`.

Evidencia U5: prueba de integración (`TestAdoptIDsCriterion`, `…ProvisionalBeforeTheHighestDefinitive`, `…PatternUndeclaredFails`, `…OriginWithoutIDsUsesTheDefaultPattern`) escrita sobre `guardProject` (rama `feat/x`: scan/check; rama de archivado: `RunSDDArchiveIntegrate`; segundo cambio con `PlanChange`+`Apply`). RED: el primer intento falló solo por la fixture (un provisional sin `Origen:` incumple la regla `origin`; se le dio `Origen: docs/requisitos.md#requisitos`); no hubo RED de producción porque U1 a U4 ya implementan la conducta: las pruebas son de caracterización y pasaron sin cambios de producción. GREEN: `go test ./internal/cli -run AdoptIDs` ok y `-run ArchiveAdopt` ok; `go vet ./internal/cli` limpio. No se destapó ningún fallo de producción.

- [x] 5.1 RED: `TestAdoptIDsCriterion` con el andamiaje de `sddIntegrateProject` (`internal/cli/sdd_archive_integrate_test.go:29-36`: `guardProject`, cambio sellado, git de prueba): (1) `docs/requisitos.md` con 120 secciones `## RF-%03d …` y `sdd: {id_pattern: "RF-<NNN>"}` en `axiom.yaml`; (2) `scan --change` → `proposed_id_pattern.template == "RF-<NNN>"`, sin conflictos, `axiom.yaml` intacto; (3) delta con 120 ADDED `(RF-%03d)`, cada uno con `Origen: docs/requisitos.md#<ancla> (RF-%03d)` y un escenario, y `adoption.md` con el título descartado; (4) `check` → código 0 y `id_pattern == "RF-<NNN>"`; (5) en la rama de archivado, `RunSDDArchiveIntegrate`: la spec viva contiene `(RF-001)`…`(RF-120)` y las 120 líneas `Origen:` byte a byte.
  Cubre: REQ-30.9, REQ-30.11, REQ-30.12, REQ-30.13
  Verificación: `go test ./internal/cli -run 'AdoptIDs'`
- [x] 5.2 Segundo cambio, una sola vía: integrar un ADDED `(prov::<k>)` de la misma capacidad con `livespec.PlanChange` + `Apply` con `LoadRegistry(Pattern)` (mismo motor que `archive-integrate`; no se usa un segundo cambio sellado con `sddIntegrateProject`, para no depender de que el andamiaje ofrezca dos cambios). El resultado es `RF-121`.
  Cubre: REQ-30.12
  Verificación: `go test ./internal/cli -run 'AdoptIDs'`
- [x] 5.3 Caso de orden: un mismo cambio de adopción con definitivos conservados y provisionales, con el definitivo más alto DESPUÉS del provisional en la delta (p. ej. `(prov::<k>)` antes de `(RF-120)`, con `RF-001…RF-120` conservados) → el provisional recibe máximo+1 (`RF-121`), porque los definitivos ocupan primero (`livespec/ids.go:496-641`); los provisionales se numeran por orden de aparición.
  Cubre: REQ-30.12
  Verificación: `go test ./internal/cli -run 'AdoptIDs'`
- [x] 5.4 Casos negativo y sin IDs: el mismo árbol sin `id_pattern` → `check` falla `pattern_undeclared` citando `id_pattern: "RF-<NNN>"`; un origen sin IDs, sin `id_pattern` y con títulos `(prov:40:<k>)` y `(prov:40:<k+1>)` → `check` pasa y el archivado da `REQ-40.1` y `REQ-40.2`, sin provisionales en la spec viva.
  Cubre: REQ-30.9, REQ-30.11, REQ-30.12
  Verificación: `go test ./internal/cli -run 'AdoptIDs'`
- [x] 5.5 Cierre de U5: `go vet ./internal/cli` y commit `test(cli): integrar adopción con IDs del cliente conservados y provisionales`. Si 5.1 a 5.4 destapan un fallo de producción, corregirlo en el commit de la unidad que lo introdujo (fixup) o como corrección dentro de este commit documentada en la evidencia.
  Cubre: REQ-30.9, REQ-30.11, REQ-30.12
  Verificación: `go test ./internal/cli -run 'AdoptIDs|ArchiveAdopt'`

## Fase 6 (U6): Guía, manual y pins

Depende de: U3, U4. Paquetes: `internal/assets`, `docs`, `internal/components`.

Evidencia U6: RED observado = `go test ./internal/assets -run 'SDDSpecAdoption|SDDSpecAndPropose'` falla con los pins nuevos (faltan `Declare the identifier pattern`, `proposed_id_pattern`, `origin_id`…; sobran `one new series`) y `go test ./internal/workspace -run DocsPin` falla con `TestDocsPinManualAdopcionConservaLosIDs` (pin nuevo del manual). GREEN: ambos ok tras reescribir `openspec-adoption.md` y la sección de adopción de `docs/manual-de-inicio.md` (6 pasos: inventario, declaración del patrón, delta, comprobación, compuerta, archivado). Goldens: no hay goldens con `openspec-adoption|adopt scan|archive adopt`; `go test ./internal/components -run TestGolden` ok sin `-update`; `go test ./internal/components/sdd -run Adoption` ok. Restos de `rg -n "hasta C-03-c|serie nueva|RuleSeries" internal docs`: solo legítimos (etiqueta de serie nueva de `reqid`/`livespec` y sus pruebas, pins negativos de pruebas y el backlog `docs/plan`); ninguno en código o assets de adopción. `sdd-spec/SKILL.md:180` cita `prov:a:<k>` (etiqueta de serie nueva de provisionales), no contradice. Commits: ajuste U3 `2cfd35c4`, U5 `ea086e52`, U6 `06544055`.

- [x] 6.1 RED: en `internal/assets/assets_test.go` (`:3544-3570`), reescribir los pins al flujo nuevo y añadir pins negativos: presentes el paso «Declare the identifier pattern», `proposed_id_pattern`, `origin_id`, `pattern_undeclared`, `foreign_id_conflict`, `axiom sdd archive-integrate`, `prov::<ordinal>` para un patrón de serie fija `RF-<NNN>`; ausentes «one new series», «C-03-c», «never the identity» y `archive-compose --canonical --delta` en el paso de archivado.
  Cubre: REQ-30.31
  Verificación: `go test ./internal/assets -run 'SDDSpecAdoption|SDDSpecAndPropose'` falla (RED)
- [x] 6.2 GREEN: en `internal/assets/skills/sdd-spec/references/openspec-adoption.md`: «Who checks what» sin «one new series»; paso 1b «Declare the identifier pattern» (revisar `proposed_id_pattern` y conflictos; la persona declara `sdd.id_pattern` en `axiom.yaml` antes de `check`; el escáner no lo escribe; las mezclas se normalizan a mano); paso 3 con el ID del cliente conservado, que coincide con el `<original ID>` de `Origen:`, o un provisional `prov:<series>:<ordinal>` con ejemplo `prov::<ordinal>` para un patrón de serie fija `RF-<NNN>`; aviso de convivencia: declarar el patrón del cliente cuando ya hay specs vivas con otro esquema deja de indexar los `REQ-n.m` existentes (declarar el patrón es una decisión del proyecto, no solo de la adopción); paso 5 con las reglas `format`, `uniqueness`, `scenarios`, `origin`, `origin_id`, `coverage`, `discard_reason`, `inventory_missing`, `inventory_stale`, `pattern_undeclared`, `foreign_id_conflict`; paso 7 archivar con `axiom sdd archive-integrate`; límite de prefijos de `foreignIDPattern` (con otro prefijo, declarar a mano y escribir `(<ID original>)` en `Origen:`). La guía y el texto de `pattern_undeclared` valen para multirepo (repositorio de specs en su rama de integración), no solo «rama del cambio». Comprobar con `rg -n "new series|serie nueva|C-03-c" internal/assets/skills` que `sdd-spec/SKILL.md:81`, `sdd-propose/SKILL.md:196`, `sdd-archive/SKILL.md:135` y `_shared/sdd-iso-convention.md:18` no contradicen y no se tocan.
  Cubre: REQ-30.31
  Verificación: `go test ./internal/assets -run 'SDDSpecAdoption|SDDSpecAndPropose'`
- [x] 6.3 RED y GREEN: en `docs/manual-de-inicio.md:587-605`, paso 1 con la propuesta del patrón, nuevo paso de declaración, paso 2 con IDs conservados o provisionales (sin «El identificador ajeno nunca es la identidad»), paso 3 con las reglas nuevas, paso 5 con `axiom sdd archive-integrate`; conservar las cadenas fijadas por `internal/workspace/docs_pin_test.go:17-18`. Si hay pin del manual para el flujo nuevo, añadirlo antes.
  Cubre: REQ-30.31
  Verificación: `go test ./internal/workspace -run DocsPin`
- [x] 6.4 Goldens y paridad: `go test ./internal/components -run TestGolden` y `go test ./internal/components/sdd -run Adoption` (paridad entre runtimes, `inject_test.go:8148`). Acotar la comprobación de goldens a los que contengan `openspec-adoption|adopt scan|archive adopt` (`rg -l "openspec-adoption|adopt scan|archive adopt" testdata/golden`); regenerar con `go test ./internal/components -run TestGolden -update` solo si falla por este cambio y revisar el diff de los goldens afectados.
  Cubre: REQ-30.31
  Verificación: `go test ./internal/components -run TestGolden`
- [x] 6.5 Cierre de U6 y comprobación final de restos: `go vet` de los paquetes tocados; `rg -n "hasta C-03-c|serie nueva|RuleSeries" internal docs` (código y assets) sin restos; commit `docs(spec): guía de adopción con IDs conservados y patrón declarado`.
  Cubre: REQ-30.31
  Verificación: `go test ./internal/assets -run 'SDDSpecAdoption|SDDSpecAndPropose'` y `go test ./internal/workspace -run DocsPin`

## Notas para el archive

Archivar con `axiom sdd archive-integrate` (C-03-b), que compone solo los bloques de requisitos. En la rama de archivado, tras componer y antes de `axiom sdd archive-check`, aplicar a mano y dejar anotado en `archive-report.md`:

1. `openspec/specs/openspec-adoption/spec.md:38` («No cubre»): sustituir la línea por el texto que fija la cabecera de la delta de `openspec-adoption`.
2. `openspec/specs/openspec-adoption/spec.md:30` (tabla «de un vistazo», fila REQ-30.12): sustituir el resumen por «Los IDs del cliente se conservan según el patrón declarado; sin IDs, provisionales». Revisar que las filas de REQ-30.9, REQ-30.10, REQ-30.11 y REQ-30.13 no contradigan el texto nuevo.
3. `openspec/specs/requirement-identifiers/spec.md:5` (`## Purpose`): quitar «la única excepción deliberada y temporal es `adoptcheck` con `archive adopt` (REQ-44.4)» y «no conserva los identificadores de un cliente al adoptar (C-03-c)». La delta de `requirement-identifiers` no lo anota: es una corrección del texto vivo.
4. Texto del escenario `mixed_separators` de la delta de `openspec-adoption`: la spec dice `RF.001`, que `foreignIDPattern` no reconoce; el comportamiento implementado y probado usa `RF-001` frente a `RF_001`. Corregir el texto del escenario en la spec viva al archivar (corrección de texto, sin cambio de comportamiento).
5. Seguimiento (R3): `livespec/ids.go:427` compara colisiones por cadena y no por clave de identidad; proponerlo para C-03-d o un cambio de `livespec`.

## Integración de main (2026-11-26)

- La PR #335 tenía conflictos con `main` por C-08-l (#332) y C-08-h (#333). Se resolvieron con un merge de `origin/main`, sin rebase porque la rama ya está publicada.
- C-08-h borró `internal/workspace/docs_pin_test.go` y `TestSDDSpecAdoptionReferenceDescribesTheWholeFlow` de `internal/assets/assets_test.go`, que eran centinelas de redacción. Se aceptan los dos borrados, de modo que los pins de U6 (guía y manual) desaparecen. La guía y el manual reescritos se conservan.
