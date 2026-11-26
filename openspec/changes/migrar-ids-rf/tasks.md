# Tareas: Migración de las specs propias de Axiom de `REQ-n.m` a `RF-<DOM>-<NN>` (C-03-e)

> **Cambio:** `migrar-ids-rf`. **Rol:** `fullstack` (único). **Entradas:** `proposal.md`, deltas de `requirement-trace` y `requirement-identifiers`, `design.md` (D1-D13) y `coverage-review.design.md` (huecos 1-9 absorbidos como criterios de las tareas).
> **Convenciones de verificación (valen para todas las tareas):**
> - No se compilan ni ejecutan binarios sueltos (EDR Cortex): solo `go test`. La herramienta de migración se ejecuta en proceso con `AXIOM_IDMIGRATION=map|plan|write go test ./internal/idmigration -run TestMigrateRepository -count=1`.
> - Pruebas localizadas: solo los paquetes tocados y, en `./internal/cli`, solo con `-run`. Nunca `go test ./...` ni `./internal/cli` entero (la suite completa corre en la CI).
> - `go vet` de los paquetes tocados al cerrar cada tarea.
> - Trinquete de código muerto: ningún código de producción sin llamadas desde `./cmd/axiom` (la herramienta es solo código de prueba).
> - Cada tarea cierra con un commit convencional, sin atribución de IA ni `Co-Authored-By`.
> - `strict_tdd: false`; excepción: la corrección de `ParseDelta` exige RED observado antes del arreglo (1.1 antes de 1.2) y, por criterio del design, también RED para las reglas nuevas de la traza.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | XL |
| Estimated changed lines | 4.500-6.000 líneas escritas a mano (producción ~700-900; pruebas y `testdata` ~3.200-4.200; YAML curado, docs y ajustes ~400). Fuera de la cifra, por ser generado: `openspec/id-map.yaml` (~2.100 líneas), la reescritura mecánica de d3 (~7.000-14.000 líneas entre 6.633 apariciones en cientos de ficheros), `INDEX.md` y 10 goldens |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| a | `ParseDelta` y `Compose` ignoran cabeceras dentro de vallas | `fix(openspecdoc): ignorar requisitos dentro de vallas en la delta` | `go test ./internal/openspecdoc ./internal/livespec -count=1` | RED observado antes del arreglo; GREEN después | Independiente: `compose.go` y sus pruebas |
| b | Resolución de IDs heredados con mapa versionado (`legacyid`, `reqtrace`, CLI) | `feat(trace): resolver identificadores heredados con un mapa versionado` | `go test ./internal/legacyid ./internal/reqtrace -count=1` y `go test ./internal/cli -run 'TestTrace' -count=1` | Repositorio git temporal con `id-map.yaml` en la prueba de la CLI (sin binario) | Revertible mientras no estén aplicados d1/e; sin mapa el comportamiento es el actual |
| c | Herramienta de migración (solo pruebas) y modelo de residuos | `test(idmigration): herramienta de migración de identificadores` | `go test ./internal/idmigration -count=1` | `AXIOM_IDMIGRATION=plan` sobre el árbol miniatura de `testdata` | Solo pruebas; revertible sin efecto en producto |
| d1 | Generar `openspec/id-map.yaml` | `chore(specs): generar el mapa de identificadores heredados` | `go test ./internal/idmigration ./internal/legacyid -count=1` | `AXIOM_IDMIGRATION=map go test ./internal/idmigration -run TestMigrateRepository -count=1` | Un solo fichero; revertir con d2/d3/e revertidos |
| d2 | Residuos, contextos y mensaje retirado | `chore(idmigration): residuos, contextos y mensajes retirados` | `go test ./internal/idmigration ./internal/techcontext -count=1` | `AXIOM_IDMIGRATION=plan ... -v` hasta cero pendientes | Con d3 |
| d3 | Reescritura masiva generada | `refactor(specs): migrar los identificadores a RF-<DOM>-<NN>` | Lote de paquetes tocados (véase 6.4) | `AXIOM_IDMIGRATION=write` + `-update` de goldens + segunda pasada sin cambios | Commit único revertible conservando a, b, c |
| e | Pruebas permanentes de residuos, mapa y corpus RF | `test(specs): residuos y corpus con el patrón RF` | `go test ./internal/idmigration ./internal/openspecdoc ./internal/reqid ./internal/reqtrace -count=1` | Pruebas sobre el repositorio real con `git ls-files` | Solo pruebas |
| f | Documentación | `docs: patrón RF de Axiom y consulta de IDs antiguos` | `go test ./internal/idmigration -run TestRepositoryLegacyIDResidues -count=1` | N/A: solo Markdown | Dos ficheros de docs |

Ruta de ejecución por tarea (design, delegación solo por motivo nombrado): a, f y 5.x en línea (unidades pequeñas); b, c y e en línea secuencial (un único escritor; superficies acopladas, sin paralelismo útil); d1-d3 en línea porque son ejecuciones de la herramienta, sin escritura manual en d3. Si el contexto del padre rebasa el umbral del respaldo de sesión, c es la candidata a escritor delegado único.

## Fase 1: Corrección de `ParseDelta` (commit a)

- [x] 1.1 RED: añadir en `internal/openspecdoc/parse_test.go` y `internal/openspecdoc/compose_test.go` pruebas de que una delta con `### Requirement:` y `## ADDED Requirements` dentro de una valla de código (incluidas vallas anidadas) no cuenta como requisito ni sección, tanto para `ParseDelta` como para `Compose`. Ejecutar y registrar el fallo observado antes de tocar producción.
  - Verificación: `go test ./internal/openspecdoc -run 'Fence|Fenced' -count=1` falla (RED observado y anotado en la evidencia).
  - Evidencia: RED observado, 3 pruebas fallan (TestParseDeltaIgnoresRequirementHeadingsInsideFences: Removed = [Falso en valla]; TestParseDeltaIgnoresHeadingsInsideNestedFences: Added = [Real Interno Falso]; TestComposeIgnoresRequirementHeadingsInsideFencedDeltaBlocks: Widget Expiration reemplazado y Widget Tagging perdido).
- [x] 1.2 GREEN: en `internal/openspecdoc/compose.go`, calcular `fencedRanges(text)` una vez en `parseOpenSpecDelta` y descartar cabeceras de sección y de requisito que empiezan dentro de una valla; pasar el desplazamiento a `parseDeltaRequirementBlocks` (D9). `ParseDelta` y `Compose` comparten la función.
  - Verificación: `go test ./internal/openspecdoc ./internal/livespec -count=1` y `go vet ./internal/openspecdoc ./internal/livespec`.
  - Evidencia: GREEN, `go test ./internal/openspecdoc ./internal/livespec -count=1` ok y `go vet` ok.
  - Commit: `fix(openspecdoc): ignorar requisitos dentro de vallas en la delta`.

## Fase 2: Resolución de IDs heredados en `axiom trace` (commit b)

- [x] 2.1 RED: crear `internal/legacyid/legacyid_test.go` con tabla sobre `fstest.MapFS`: fichero ausente (`nil, nil`); ilegible; YAML inválido; campo desconocido; esquema desconocido; ID nuevo repetido o inválido; antiguo repetido en una capacidad (y permitido en dos); entrada sin `old` ni `title`; capacidad vacía; resolutor (`Aliases` en ambos sentidos, `InCapability`, `Capabilities`, `Legacy`, `Known`); token `REQ-27.2b` como clave antigua (consulta exacta, hueco 7).
  - Cubre: prov:TRC:1, prov:TRC:2
  - Evidencia: RED observado, `go test ./internal/legacyid` no compila (`undefined: FileName`, `Map`, `Load`, `CodeUnreadable`...); la tabla cubre ausente, ilegible, vacío, YAML inválido, campo desconocido, esquema desconocido/ausente, nuevo repetido/inválido/vacío, antiguo repetido (en la capacidad y en la misma entrada), sin `old` ni `title`, capacidad vacía, y el resolutor.
- [x] 2.2 RED: añadir en `internal/legacyid/legacyid_test.go` los casos de patrón (hueco 4): con un mapa cuyos IDs nuevos son `RF-<DOM>-<NN>` y el patrón por defecto `REQ-<n>.<m>`, `Load` no falla por la forma de los IDs nuevos (se validan con el patrón declarado del mapa o con `RF-<DOM>-<NN>`, nunca exigiendo `REQ-<n>.<m>`); y precedencia fijada: un mapa inválido gana siempre sobre un ID de consulta inválido.
  - Cubre: prov:TRC:2
  - Evidencia: RED en el mismo fallo de compilación (`TestLegacyIDNewIDsUseRFPatternNotDefault`); `Load` acepta IDs `RF-<DOM>-<NN>` con el patrón por defecto y con el RF, y rechaza uno inválido bajo ambos. La precedencia mapa inválido sobre ID inválido se fija en la CLI (2.6, `TestAxiomTraceInvalidMapFailsTypedWithoutGit`).
- [x] 2.3 GREEN: crear `internal/legacyid/legacyid.go` con `FileName`, `Schema`, `Load` (lectura estricta `KnownFields`), `Map` y sus métodos, `MapError` y códigos `id_map_unreadable`, `id_map_malformed`, `id_map_schema_unknown`, `id_map_invariant`; el `Error()` nombra fichero, código, motivo y continuación (D2, D8). Crear `internal/legacyid/imports_test.go` que prohíbe `os/exec`, `gitcommon` y la importación de `reqtrace` en producción.
  - Verificación: `go test ./internal/legacyid -count=1` y `go vet ./internal/legacyid`.
  - Cubre: prov:TRC:1, prov:TRC:2
  - Evidencia: GREEN, `go test ./internal/legacyid -count=1` ok y `go vet ./internal/legacyid` ok.
- [x] 2.4 RED: en `internal/reqtrace/*_test.go` (con fixtures de mapa en memoria): `ParseID` acepta un antiguo del mapa, rechaza uno fuera del mapa, rechaza sin mapa; cierre de equivalencias antiguo a nuevos y nuevo a antiguos con aviso `legacy_ambiguous` cuando un antiguo da varios nuevos; coincidencia acotada por capacidad (delta de `autoskills` frente a `workspace-topology` con `REQ-1.1`); `Cubre:` con y sin contexto; entradas de `ids.yaml` con `definitive` heredado; el token `REQ-27.2b` se consulta y se encuentra como token completo y no como `REQ-27.2` seguido de `b` (hueco 7).
  - Cubre: RF-TRC-02, RF-TRC-04
  - Evidencia: RED observado, `go vet ./internal/reqtrace` no compila (`ParseID` con 3 argumentos, `Query.Legacy`, `resolveEquivalentsLegacy`, `WarnLegacyAmbiguous` indefinidos) con `internal/reqtrace/legacy_test.go`.
- [x] 2.5 GREEN: en `internal/reqtrace/alias.go`, `id.go`, `classify.go`, `collect.go` y `model.go` añadir `LegacyResolver`, `Input.Legacy`, `Query.Legacy`, `ParseID(raw, p, legacy)`, `acceptableAlias` ampliado, `matchSet` sin heredados salvo el propio ID, búsqueda aparte de tokens heredados con `reqid.Default().Scan` y coincidencia acotada por capacidad, y `WarnLegacyAmbiguous` (D8). Si `Scan` o `isTokenRune` no leen `REQ-27.2b` como token único, ajustar el escaneo para que lo haga.
  - Verificación: `go test ./internal/reqtrace -count=1` y `go vet ./internal/reqtrace`; `imports_test.go` de `reqtrace` sigue sin `os` ni YAML.
  - Cubre: RF-TRC-02, RF-TRC-04
  - Evidencia: GREEN, `go test ./internal/legacyid ./internal/reqtrace -count=1` ok, `go vet` ok; `imports_test.go` de `reqtrace` sigue pasando. `Scan` ya lee `REQ-27.2b` como token único: no hizo falta tocar el escaneo. Decisiones: un antiguo alcanzado desde un ID nuevo no se vuelve a expandir (evita arrastrar los nuevos de otras capacidades); las entradas de `ids.yaml` con capacidad traducen sus IDs heredados al nuevo de (capacidad, antiguo); `legacy_ambiguous` de clasificación solo se emite si alguna candidata está en el cierre.
- [x] 2.6 RED y GREEN: en `internal/cli/trace.go`, cargar el mapa con `legacyid.Load(os.DirFS(loc.PlanningHome), legacyid.FileName, pattern)` tras `projectIDPattern` y antes de `ParseID`; mapa ausente sin alias ni aviso; mapa inválido devuelve `*legacyid.MapError` sin llamar a git; cablear `Aliases` y `Legacy` en `Input` sin interfaz con puntero nulo. Pruebas en `internal/cli/trace_legacy_test.go` (repositorio git temporal con `id-map.yaml`): ID antiguo aceptado, ambiguo, fuera del mapa, mapa inválido que falla sin ejecutar git, sin mapa sin aviso, y precedencia mapa inválido sobre ID inválido.
  - Verificación: `go test ./internal/cli -run 'TestTrace' -count=1` y `go vet ./internal/cli`; sin código de producción huérfano.
  - Evidencia: RED observado, `go vet ./internal/cli` no compila (`not enough arguments in call to reqtrace.ParseID`) con `trace_legacy_test.go`; GREEN, `go test ./internal/cli -run 'Trace' -count=1` ok (130 s) y `go vet ./internal/cli` ok; `go test ./internal/reqid -run TestNoRequirementIDFormatOutsideReqid` ok. Sin producción huérfana: `Map` se alcanza desde `RunTrace` y por la interfaz `LegacyResolver`.
  - Commit de las fases 2.1-2.6: `feat(trace): resolver identificadores heredados con un mapa versionado`.
  - Cubre: prov:TRC:1, prov:TRC:2, RF-TRC-02, RF-TRC-04

## Fase 3: Herramienta de migración, solo pruebas (commit c)

- [x] 3.1 Crear `internal/idmigration/domains_test.go` con la tabla aprobada de 49 códigos de dominio y el árbol miniatura `internal/idmigration/testdata/**` (capacidades, ambiguos, títulos no finales, `(A & B)`, `REQ-27.2b`, sin ID, ODD con vallas `openspec`, Go, Markdown, ficheros binarios, ficheros con CRLF y con BOM).
  - Verificación: `go test ./internal/idmigration -run TestDomains -count=1`.
  - Evidencia: GREEN, `TestDomainsTable` (49 códigos únicos, tres mayúsculas, válidos bajo `RF-<DOM>-<NN>`) y `TestDomainsSeriesNumbering` ok. Árbol miniatura en `internal/idmigration/testdata/mini` (5 capacidades, ambiguos `REQ-1.1`/`REQ-2.1`, título no final, `(A & B)`, `REQ-27.2`/`REQ-27.2b`, título sin ID, ODD con vallas `openspec` y capacidad retirada, Go de producción y de pruebas, Markdown, binario con NUL). Los CRLF y el BOM se fabrican en `t.TempDir()` dentro de las pruebas, porque `.gitattributes` (`* text=auto eol=lf`) normaliza los ficheros versionados.
- [x] 3.2 RED y GREEN del generador: `internal/idmigration/generate_test.go` con doble generación idéntica byte a byte, numeración por aparición, ambigüedad calculada, emisor determinista (cabecera fija, LF, salto final), ida y vuelta con `legacyid.Load(Generate(x))` sin error, fallo si el fichero ya existe o si un título ya tiene un ID que no es `REQ-n.m`, y `REQ-27.2b` como token del mapa (hueco 7).
  - Cubre: prov:TRC:1
  - Verificación: `go test ./internal/idmigration -run 'TestGenerate' -count=1`.
  - Evidencia: GREEN, 9 pruebas `TestGenerate*` ok (numeración por aparición, ambigüedad calculada, doble generación idéntica, emisor LF con cabecera fija, ida y vuelta con `legacyid.Load`, `REQ-27.2b` distinto de `REQ-27.2`, falla si el mapa existe, falla ante títulos con ID no heredado, sin código de dominio o claves repetidas, ignora títulos dentro de vallas). RED: el generador y las pruebas se escribieron juntos y pasaron a la primera, así que el RED se observó por mutación (véase 3.3); no hubo fallo previo de compilación.
- [x] 3.3 RED y GREEN del reescritor: `internal/idmigration/rewrite_test.go` con las reglas D3-D6: token completo (`REQ-1.1` no toca `REQ-1.10`), títulos (D6, incluido `REQ-27.2b` por sustitución y títulos sin ID), continuaciones y rangos D5 (contiguo y no contiguo), cadena de contexto D3 (cada eslabón y fallo sin escribir), Go con `go/scanner` (comentario, literal fixture, literal fijado, `go/format`), ODD (vallas `openspec`, capacidad retirada), Markdown de specs (código en línea conservado), idempotencia (segunda pasada sin cambios). Criterios de bytes (hueco 5): se preservan CRLF/LF y BOM salvo los tokens sustituidos; un fichero sin sustituciones no se reescribe (mismo contenido y mismo mtime); rutas listadas por git que no existen y ficheros binarios se omiten sin error.
  - Cubre: prov:RID:1
  - Verificación: `go test ./internal/idmigration -run 'TestRewrite' -count=1`.
  - Evidencia: RED observado por mutación: con el límite derecho de `findTokens` anulado (`if r == 0`), `TestRewriteWholeTokenOnly` falla (`REQ-1.10`/`REQ-11.1` se tocarían); restaurado, GREEN con 14 pruebas `TestRewrite*` ok: token completo, sufijo `REQ-27.2b`, títulos D6 (no final, `(A & B)`, sin ID, idempotente, ID distinto), listas y rangos D5 (contiguo, no contiguo y extremo sin mapa), cadena D3 (línea, fichero, tabla de contexto y fallo `ruta:línea token [candidatas]`), Go (`go/scanner`, literal fijado y fixture, `gofmt` solo si el original ya era canónico), ODD (vallas `openspec` anidadas y capacidad retirada), specs (código en línea y vallas conservados), residuo, idempotencia, CRLF y BOM preservados, fichero sin sustituciones con mismo contenido y mtime, rutas inexistentes y binario omitidos, fallo sin escribir ante ambiguo, rango no contiguo, literal de producción y título con ID distinto.
- [x] 3.4 RED y GREEN del modelo de residuos y las guardas de raíz: `internal/idmigration/residues_test.go` y la lectura de `residues.yaml` y `contexts.yaml` (D7): cita no declarada falla con `ruta:línea`; entrada o token caducado falla; la caducidad cuenta el archivo; validación estructural (categoría, ámbito, motivo, `tokens` obligatorio); prohibición de entradas sobre literales de Go de producción y sobre los assets embebidos de producción `internal/assets/skills/**` salvo ejemplo explícito del patrón por defecto (`default-pattern-example`, hueco 9); la prueba falla (no se omite) si no hay `git` o la raíz no es un repositorio (hueco 8); resolución de raíz con `filepath.Abs("../..")` y `git -C <raíz> rev-parse --show-toplevel`, rechazo de una ruta con `..` o absoluta de la lista.
  - Cubre: prov:RID:1
  - Verificación: `go test ./internal/idmigration -run 'TestResidue|TestRepoRoot' -count=1`.
  - Evidencia: GREEN, 9 pruebas `TestResidue*` y 4 `TestRepoRoot*` ok: cita no declarada falla con `ruta:línea`, residuos declarados aceptados, entrada y token caducados, caducidad que cuenta el archivo, ficheros de entrada que no mantienen viva una entrada, validación estructural (categoría, ámbito, motivo, `tokens`), veto de literales de producción y assets embebidos (con excepción del ejemplo del patrón por defecto), ámbitos por `go/scanner` y Markdown; raíz con `git -C <raíz> rev-parse --show-toplevel`, falla (no se omite) sin git (`PATH` vacío) y fuera de un repositorio, y rechaza `..`, rutas absolutas y vacías.
- [x] 3.5 `TestMigrateRepository` (`internal/idmigration/migrate_test.go`) con los modos `map`, `plan` y `write` por la variable `AXIOM_IDMIGRATION` (sin variable: `t.Skip`); `plan` y `write` fallan sin escribir nada ante ambiguos sin resolver, rangos no contiguos, literal de producción no mapeado o título con ID distinto; inserción del bloque `sdd:` en `axiom.yaml` (D10) y regeneración de `INDEX.md` con `livingdoc.NewService(raíz, nil, nil).Sync` solo si cambia algo más que la línea de fecha. Probar los tres modos sobre el árbol miniatura.
  - Verificación: `go test ./internal/idmigration -count=1` y `go vet ./internal/idmigration`; el paquete no contiene ficheros de producción.
  - Evidencia: `go test ./internal/idmigration -count=1` ok (`TestMigrateRepository` se omite sin variable) y `go vet` ok; `find internal/idmigration -name '*.go' ! -name '*_test.go' ! -path '*/testdata/*'` vacío. Los tres modos sobre el árbol miniatura (`TestMigrateMiniatureMapPlanWriteEndToEnd`): `plan` no escribe y sin mapa usa el generado en memoria, `write` sin mapa pide `map`, `map` solo crea `openspec/id-map.yaml`, `write` reescribe, inserta el bloque `sdd:`, regenera `INDEX.md` y la segunda ejecución no cambia nada. `AXIOM_IDMIGRATION=plan` contra el repositorio real (solo lectura, sin mapa: usa el generado en memoria): 1969 ficheros analizados, 583 reescribirían 4326 sustituciones, 39 ambiguos sin resolver (los 39 medidos en el design), 1 literal de producción sin equivalente (`internal/techcontext/plan.go:137`, `REQ-24.62`, D13), 43 tokens distintos sin equivalente en 151 apariciones y 44 ficheros candidatos a residuo; `plan` falla, como debe, hasta que d2 aporte `contexts.yaml` y `residues.yaml`.
  - Commit de las fases 3.1-3.5: `test(idmigration): herramienta de migración de identificadores`.
  - Cubre: prov:TRC:1, prov:RID:1

## Fase 4: Generación del mapa y datos curados (commits d1 y d2)

- [x] 4.1 Ejecutar `AXIOM_IDMIGRATION=map go test ./internal/idmigration -run TestMigrateRepository -count=1` para generar `openspec/id-map.yaml` (513 entradas). Comprobar la ida y vuelta con `legacyid.Load` y que `REQ-27.2b` y las dos entradas con dos antiguos están presentes. Sin ediciones a mano.
  - Verificación: `go test ./internal/idmigration ./internal/legacyid -count=1`.
  - Evidencia: `openspec/id-map.yaml` generado (513 entradas, 1542 líneas; segunda generación idéntica byte a byte con `cmp`), `go test ./internal/idmigration ./internal/legacyid -count=1` ok, `plan` lo carga sin error; `REQ-27.2b` → `RF-CLI-09`, `REQ-48.2` → `RF-TRC-02`, `REQ-48.4` → `RF-TRC-04`, `REQ-24.3` (planning-home-resolution) → `RF-PLH-03`, 2 entradas con dos antiguos.
  - Commit: `chore(openspec): generar el mapa de identificadores heredados`.
  - Cubre: prov:TRC:1
- [x] 4.2 Corregir a mano el mensaje de producción con el requisito retirado en `internal/techcontext/plan.go` (línea 137 al diseñar, sustituir la cita retirada por el requisito vivo de contexto técnico mantenido, D13) y ajustar la prueba que fija ese mensaje en `internal/techcontext`.
  - Verificación: `go test ./internal/techcontext -count=1` y `go vet ./internal/techcontext`.
  - Evidencia: `plan.go:137` cita ahora `REQ-30.19` (único en el mapa, `project-technical-context`, «El Contexto Técnico Mantenido Exige un Almacén de Artefactos de Ficheros»). Ninguna prueba fija la cita retirada (`TestCheckRequiresAFileArtifactStoreWithMaintainedContext` comprueba `technical_context.maintained`, el almacén y `artifact_store: openspec`), así que no hubo que ajustar pruebas. `go test ./internal/techcontext -count=1` ok y `go vet` ok.
  - Cubre: prov:RID:1
- [x] 4.3 Crear `internal/idmigration/residues.yaml` (esquema `axiom.id-residues/v1`) y `internal/idmigration/contexts.yaml` (`axiom.id-contexts/v1`) a partir del informe de `AXIOM_IDMIGRATION=plan go test ./internal/idmigration -run TestMigrateRepository -count=1 -v`: fixtures del patrón por defecto, ejemplos para otros proyectos, retirados sin equivalente como `legacy-citation` con motivo «requisito retirado sin equivalente vivo», historia fechada, la carpeta de este cambio (rutas activa y archivada) y las ~10 entradas de contexto de Go con evidencia. Ninguna entrada alcanza literales de Go de producción ni los assets embebidos de producción salvo el ejemplo del patrón por defecto. Repetir `plan` hasta cero pendientes.
  - Verificación: `AXIOM_IDMIGRATION=plan go test ./internal/idmigration -run TestMigrateRepository -count=1` pasa sin pendientes; `go test ./internal/idmigration -count=1`.
  - Evidencia: `contexts.yaml` con 28 entradas (resuelven los 39 ambiguos, cada una con la evidencia de la spec o del comentario) y `residues.yaml` con 94 entradas (retirados sin equivalente vivo como `legacy-citation`, fixtures y comentarios de fixtures del patrón por defecto, ejemplos en specs e `INDEX.md`, historia fechada, bloques `openspec` de ODD conservados, árbol miniatura y carpeta de este cambio). `AXIOM_IDMIGRATION=plan` termina con 0 ambiguos, 0 fallos y 0 candidatos a residuo: 1975 ficheros analizados, 587 reescritos, 4345 sustituciones. Comprobación adicional sin tocar el árbol: un `write` simulado sobre una copia temporal más `checkResidues` sobre el resultado solo dejó pendientes los 10 goldens de `testdata/golden`, que regenera `-update` en d3. Ajuste de la herramienta: `classify` trata un `.go` bajo `testdata/` como fixture (el literal de `testdata/mini/internal/foo/foo.go` no podía declararse residuo) con la prueba `TestResidueTestdataGoFileIsAFixtureNotProduction`. `go test ./internal/idmigration -count=1` ok y `go vet` ok.
  - Commit de 4.2-4.3: `chore(idmigration): curar contextos y residuos de la migración`.
  - Cubre: prov:RID:1

## Fase 5: Reescritura masiva generada (commit d3)

- [ ] 5.1 Ejecutar `AXIOM_IDMIGRATION=write go test ./internal/idmigration -run TestMigrateRepository -count=1`: reescribe specs, Go, ODD, docs vivos, assets, `axiom.yaml` (bloque `sdd:` con `id_pattern: "RF-<DOM>-<NN>"`) y regenera `openspec/INDEX.md` en proceso. Sin ediciones a mano. Comprobar la segunda pasada en memoria sin cambios (idempotencia) y que `git diff --stat` no toca `openspec/changes/archive/` ni `openspec/id-map.yaml`.
  - Verificación: `AXIOM_IDMIGRATION=plan go test ./internal/idmigration -run TestMigrateRepository -count=1` tras escribir informa cero sustituciones.
  - Cubre: prov:RID:1
- [ ] 5.2 Regenerar los 10 goldens: `go test ./internal/components -run TestGolden -count=1 -update` y confirmar estabilidad con `go test ./internal/components -run TestGolden -count=1` sin `-update`.
  - Verificación: ambas ejecuciones pasan.
- [ ] 5.3 Pasar, por lotes y con `go vet`, los paquetes cuyos pines de mensajes cambian por la reescritura: `./internal/kickoff`, `./internal/sddstatus`, `./internal/techcontext`, `./internal/branch/...` (engine), `./internal/assets`, `./internal/components`, `./internal/livespec`, `./internal/openspecdoc`, `./internal/reqid`, `./internal/reqtrace`, `./internal/legacyid`, `./internal/livingdoc` y el resto de paquetes con ficheros tocados según `git diff --name-only`. En `./internal/cli` solo con `-run` de las pruebas afectadas (p. ej. `go test ./internal/cli -run 'TestTrace|TestSDDStatus|TestKickoff' -count=1`); nunca el paquete entero ni `./...`. Si un pin falla, la causa es un literal fijado mal reescrito en la herramienta: corregir la herramienta (commit c) y repetir `write`, no editar el resultado a mano.
  - Verificación: `go test <paquetes tocados> -count=1` y `go vet <paquetes tocados>`; el trinquete de código muerto no se ve afectado (no hay producción nueva).
  - Commit de 5.1-5.3 (una sola unidad, salida exacta de la herramienta y goldens): `refactor(specs): migrar los identificadores a RF-<DOM>-<NN>`.
  - Cubre: prov:RID:1

## Fase 6: Pruebas permanentes (commit e)

- [ ] 6.1 Crear `TestIDMapMatchesLivingSpecs` en `internal/idmigration`: carga el mapa real con `legacyid.Load` y comprueba solo mapa a spec viva (cada entrada existe en la spec viva de su capacidad con su ID nuevo; 513 entradas en el momento de la migración). No comprueba spec a mapa, para no romper al archivar este cambio ni con requisitos posteriores (huecos 1; RF-TRC-16, RF-TRC-17, RF-RID-16 no deben estar en el mapa). Antes del archivado, comprueba además que `RF-TRC-02` y `RF-TRC-04` existen en `requirement-trace`.
  - Cubre: prov:TRC:1, RF-TRC-02, RF-TRC-04
- [ ] 6.2 Huella de inmutabilidad (hueco 2): fijar en `internal/idmigration` una prueba con el `sha256` de `openspec/id-map.yaml` tal como queda tras 4.1, con mensaje de fallo que explica que el mapa es inmutable una vez publicado y que un cambio exige actualizar la huella de forma deliberada y revisada. Es el mecanismo barato; queda anotado como pendiente refinar a una huella por conjunto de entradas originales si hiciera falta añadir entradas legítimas después.
  - Cubre: prov:TRC:1
- [ ] 6.3 Crear `TestRepositoryLegacyIDResidues` (siempre activa): recorre `git -C <raíz> ls-files -z`, busca `REQ-\d+\.\d+`, clasifica por ámbito y exige cobertura por `residues.yaml` fuera de `openspec/changes/archive/**` y de `openspec/id-map.yaml`; caducidad de entradas y tokens sobre cualquier fichero versionado; falla (no omite) sin git o fuera de un repositorio; veta los literales de Go de producción y los assets embebidos de producción como residuo (huecos 8 y 9).
  - Cubre: prov:RID:1
- [ ] 6.4 Prueba de líneas `Cubre:` (hueco 3): recorre los documentos vivos (`odd/tasks/*.md` y `openspec/specs/**/spec.md`), extrae los IDs de cada línea `Cubre:` y exige que sean válidos bajo `RF-<DOM>-<NN>` y existan en la spec viva, salvo residuos de requisitos retirados declarados explícitamente en una lista de la propia prueba (cada uno con motivo). No inspecciona `openspec/changes/archive/**`.
  - Cubre: prov:RID:1
- [ ] 6.5 Prueba permanente de `axiom.yaml` (hueco 8, RF-RID-01): `axiom.yaml` declara `sdd.id_pattern: RF-<DOM>-<NN>`; se lee con el cargador de configuración del proyecto o `yaml.v3` en `internal/idmigration`.
  - Cubre: prov:RID:1
- [ ] 6.6 Variantes RF del corpus: en `internal/openspecdoc/compose_corpus_test.go` añadir `TestComposeCorpusIdentityModifiedRF` con `reqid.Parse("RF-<DOM>-<NN>")` exigiendo `p.InTitle(nombre) != ""` y `reqid.Default().InTitle(nombre) == ""` en todo requisito; en `internal/reqid/legacy_equivalence_test.go` pasar el corpus de equivalencia a los títulos de `openspec/changes/archive/*/specs/*/spec.md` y añadir `TestRFCorpusTitlesHaveIDs` sobre `openspec/specs`.
  - Cubre: prov:RID:1
- [ ] 6.7 `TestTraceRealMapResolvesLegacyID` en `internal/reqtrace`: lee `../../openspec/id-map.yaml` y el patrón de `../../axiom.yaml`, comprueba que `REQ-24.3` se acepta y resuelve a `RF-PLH-03`, que `REQ-1.1` resuelve a siete IDs nuevos con `legacy_ambiguous` y que `REQ-27.2b` se acepta y resuelve a su ID nuevo.
  - Cubre: RF-TRC-02, RF-TRC-04, prov:TRC:1
  - Verificación 6.1-6.7: `go test ./internal/idmigration ./internal/openspecdoc ./internal/reqid ./internal/reqtrace -count=1` y `go vet` de esos paquetes.
  - Commit: `test(specs): residuos y corpus con el patrón RF`.

## Fase 7: Documentación (commit f)

- [ ] 7.1 Actualizar a mano `docs/openspec-config.md` y `docs/manual-de-inicio.md`: el `sdd.id_pattern` de Axiom, el fichero `openspec/id-map.yaml` y la consulta de IDs antiguos con `axiom trace`. Declarar en `residues.yaml` los ejemplos del patrón por defecto que dichos documentos conserven (solo con `tokens`, nunca por fichero entero).
  - Verificación: `go test ./internal/idmigration -run TestRepositoryLegacyIDResidues -count=1`.
  - Commit: `docs: patrón RF de Axiom y consulta de IDs antiguos`.
  - Cubre: prov:TRC:1, prov:RID:1

## Fase 8: Comprobación final

- [ ] 8.1 Comprobar con `rg 'REQ-[0-9]+\.[0-9]+'` fuera de `openspec/changes/archive/` que toda aparición restante figura en la lista de residuos (`residues.yaml`) y que no queda ninguna en literales de Go de producción ni en `internal/assets/skills/**` salvo el ejemplo declarado; confirmar que `TestRepositoryLegacyIDResidues` pasa y que la revisión de d3 es reproducible (`git checkout <d2>`, `AXIOM_IDMIGRATION=write ...`, `-update` de goldens, `git diff <d3>` solo difiere en la línea de fecha de `INDEX.md`).
  - Verificación: `go test ./internal/idmigration ./internal/legacyid ./internal/openspecdoc ./internal/livespec ./internal/reqid ./internal/reqtrace -count=1`, `go vet` de los mismos y `go test ./internal/cli -run 'TestTrace' -count=1`.
  - Cubre: prov:RID:1, prov:TRC:1, prov:TRC:2, RF-TRC-02, RF-TRC-04

## Orden y dependencias

Estrictamente secuencial por commits: a, b y c son independientes entre sí a nivel de código, pero se ejecutan en ese orden con un solo escritor; d1 requiere b y c; d2 requiere d1 (el informe `plan`); d3 requiere d2; e requiere d3 (usa el mapa real y la spec migrada); f requiere e. La fase 8 es verificación sin commit.
