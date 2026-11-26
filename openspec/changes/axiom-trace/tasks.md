# Tareas: `axiom trace <ID>`, trazabilidad de un requisito de punta a punta (C-03-d)

Rol único `fullstack` (`kickoff.yaml`). Fuentes: `proposal.md`, `specs/requirement-trace/spec.md` (prov:trace:1 a prov:trace:15) y `design.md` (decisiones 1 a 13, unidades U1 a U10).

Normas de ejecución para todo el plan:

- Pruebas funcionales normales (`strict_tdd: false`): cada comportamiento se entrega con su prueba en el mismo commit.
- Verificación localizada, nunca la batería completa: no se ejecuta `go test ./...` ni `go test ./internal/cli` sin `-run`. No se compilan binarios sueltos de axiom (las pruebas llaman a `RunTrace` y `runTrace` en proceso). La suite completa y `scripts/deadcode-ratchet.sh` los pasa la CI.
- Cada unidad cierra con un único commit convencional, sin atribución a IA, con código, pruebas y documentación juntos.
- Nombres de pruebas: `TestRequirementTrace*` (dominio), `TestAxiomTrace*` (adaptador), `TestTraceVerb*` (`cmd/axiom`). Los ayudantes nuevos no reutilizan `runTrace`, `traceDelta` ni `traceTasks` de `sdd_trace_flow_test.go`.
- `internal/cli/sdd_trace.go` y `isoevidence.History` no se tocan; `axiom.sdd-trace/v1` queda intacto.
- Los huecos de `coverage-review.spec.md` y `coverage-review.design.md` no cambian spec ni design: se resuelven aquí solo con detalle de implementación coherente con el design y se citan como «hueco spec N» o «hueco design N».

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | M |
| Estimated changed lines | 2.800 a 3.800 (adiciones + eliminaciones, sin generados; cerca de dos tercios son pruebas) |

### Work Units

| Unit | Goal | Commit | Focused test command | Rollback boundary |
|------|------|--------|----------------------|-------------------|
| U1 | Modelo, `ParseID`, regla de importaciones | `feat(reqtrace): modelo, validación del ID y regla de solo lectura` | `go test ./internal/reqtrace -run RequirementTrace -count=1` | `internal/reqtrace/{doc,model,id}.go` y pruebas |
| U2 | Cierre de equivalentes y `AliasResolver` | `feat(reqtrace): resolver equivalentes por la traza con punto de extensión de alias` | igual | `internal/reqtrace/alias.go` |
| U3 | Lectura y clasificación SDD | `feat(reqtrace): incrementos SDD archivados con rol, tipo y tareas` | igual | `scan.go`, `classify.go` (parte SDD) |
| U4 | Lectura y clasificación ODD tolerante | `feat(reqtrace): incrementos ODD con búsqueda tolerante y advertencias` | igual | `scan.go`, `classify.go` (parte ODD) |
| U5 | Anclaje y orden | `feat(reqtrace): anclar al commit --first-parent y ordenar de forma determinista` | igual | `anchor.go` |
| U6 | `Collect` de extremo a extremo | `feat(reqtrace): Collect con fuentes SDD y ODD` | igual | `collect.go` |
| U7 | Adaptador CLI | `feat(cli): verbo trace con salida de texto y JSON axiom.trace/v1` | `go test ./internal/cli -run AxiomTrace -count=1` | `internal/cli/trace.go` y `trace_test.go` |
| U8 | Pruebas de integración con git real | `test(cli): integración de axiom trace con merge, squash, rebase y multirepo` | `go test ./internal/cli -run Trace -count=1` | `internal/cli/trace_flow_test.go` |
| U9 | Registro en `main` | `feat(cmd): registrar axiom trace y su ayuda` | `go test ./cmd/axiom -run TraceVerb -count=1` | `cmd/axiom/main.go`, `trace_help_test.go` |
| U10 | Manual y pin | `docs(trace): documentar axiom trace en el manual de inicio` | `go test ./internal/workspace -run DocsPinManualDeInicio -count=1` | `docs/manual-de-inicio.md`, `docs_pin_test.go` |

Entrega `single-pr`: los diez commits forman una sola PR. Si U3 o U4 superan claramente las 100 líneas, se separan lectura y clasificación sin partir un comportamiento.

---

## Fase 1 (U1): Modelo e ID

Depende de: nada. Paquete: `internal/reqtrace`.

- [x] 1.1 Crear `doc.go` (con `Schema = "axiom.trace/v1"`) y `model.go`: `Result`, `Increment`, `Commit`, `Task`, `Assignment`, `Warning`; constantes `Lane`, `Kind`, `Role`, `Anchor` y los ocho códigos de advertencia de la decisión 5. Listas nunca `null` (constructores o normalización al emitir).
  Cubre: prov:trace:9
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 1.2 Crear `id.go` con `Query`, `ParseID` e `IDError` (`Code: "id_invalid"`, `Cause`) y su prueba de tabla: válido, provisional, inválido, patrón `RF-<DOM>-<NN>`, `REQ-01.1` frente a `REQ-1.1` (sin normalizar, hueco spec 5), espacios y minúsculas rechazados.
  Cubre: prov:trace:2
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 1.3 Crear `imports_test.go` (modelo: `internal/isoevidence/imports_test.go:19-50`): `reqtrace` no importa `os`, `os/exec` ni `internal/gitcommon`.
  Cubre: prov:trace:10
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`; `go vet ./internal/reqtrace`

## Fase 2 (U2): Equivalentes

Depende de: Fase 1.

- [x] 2.1 Crear `alias.go`: `AliasResolver`, `NoAliases`, lectura de trazas con `livespec.ParseTrace` (`ids.yaml`) y `livespec.ExtractODDTrace`, semillas, cierre hasta punto fijo, `resolvedIds` en orden de bytes y conjunto de coincidencia por fuente (el provisional de otra fuente no cuenta). Una traza ilegible da `trace_invalid` y se ignora. Pruebas con `fstest.MapFS`: provisional y definitivo dan el mismo conjunto, cadena A→B→C, ciclo, `Previous` compartido, alias inyectado que cumple el patrón y alias que no lo cumple (descartado), `NoAliases` (solo el propio ID), ID sin traza.
  Cubre: prov:trace:3, prov:trace:4
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 2.2 Provisional con varios definitivos: se añaden todos y se emite `provisional_ambiguous` con las fuentes; el mismo provisional repetido en cambios distintos no trae falsos positivos de la otra fuente (hueco spec 4, hueco design 11).
  Cubre: prov:trace:3
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`

## Fase 3 (U3): SDD

Depende de: Fase 2.

- [x] 3.1 Crear `scan.go` (parte SDD): `fs.ReadDir` de `changes/archive` con nombres seguros (`^\d{4}-\d{2}-\d{2}-[A-Za-z0-9][A-Za-z0-9._-]*$`, sin `..`; el resto se omite con `unsafe_name`, sin llamada a `History`). Los cambios activos no se leen. `changes/archive` inexistente o vacío da resultado vacío sin error ni advertencia (hueco design 6). Una lectura que falla por otra causa distinta de «no existe» da `unreadable` y se omite.
  Cubre: prov:trace:5, prov:trace:11
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 3.2 Crear `classify.go` (parte SDD): roles por título de requisito con `openspecdoc.ParseChangeSpec` (ADDED `created`, MODIFIED `modified`, REMOVED `removed`, RENAMED por origen y por destino con `SplitRename`; especificación completa como ADDED). Comparación siempre por token exacto con `isoevidence.RequirementKeys`/`Scan`: `REQ-1.1` no casa con `REQ-1.10` ni `REQ-11.1` (hueco spec 1). Un incremento por rol, `ids` sin repetidos por token y capacidades.
  Cubre: prov:trace:5
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 3.3 Tareas: ficheros `^tasks(\.[^.]+)?\.md$` (no por `kickoff.yaml`), líneas `Cubre:` con `isoevidence.ParseCubre` (varias por fichero, varios IDs por línea, ignorando las de bloques vallados, hueco spec 1), rol `covered` con fichero y línea. Solo `ids.yaml` sin aparición en delta: rol `created`. Pruebas: `tasks.md` y `tasks.<rol>.md`, dos carpetas con el mismo nombre de cambio (solo aparece la que cita el ID, con su ruta completa), cambio activo ignorado, RENAMED por origen y por destino.
  Cubre: prov:trace:5
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 3.4 Tipo y tolerancia: `bug` si `isoevidence.ParseBugDeclaration(proposal.md).Kind != BugKindNone`, si no `increment`, aplicado a todos los roles; `proposal.md` ausente o ilegible da `increment` sin abortar. Una delta archivada que existe pero no se puede interpretar emite advertencia `unreadable` con la ruta y la búsqueda sigue con el resto del cambio, sin abortar (hueco design 2; reutiliza un código del conjunto cerrado de la decisión 5). `proposal.md`, `tasks*.md` o delta ausentes se tratan como vacíos sin advertencia. Pruebas: `existing_requirement`, `no_impact`, sin declaración, bug con delta (sale `increment`).
  Cubre: prov:trace:6
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`

## Fase 4 (U4): ODD

Depende de: Fase 3.

- [x] 4.1 Parte ODD de `scan.go`: leer `odd/tasks/*.md` con nombres seguros (`^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`; el resto `unsafe_name`). `odd/tasks` inexistente o vacío da resultado vacío (hueco design 6). `livespec.ParseFeatureDoc` correcto: roles por la delta de cada `### Capacidad:`, `covered` por las líneas `Cubre:` del documento entero con fichero y línea, `created` por una entrada de `## Identificadores asignados` sin aparición en delta. Tipo `bug` si `Kind == livespec.KindBug`. Documento anterior a la traza: se encuentra por los títulos de su delta, sin error.
  Cubre: prov:trace:7
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 4.2 ODD tolerante: si `ParseFeatureDoc` falla, advertencia `odd_unparseable` con la ruta y búsqueda en el texto (bloques `openspec` con `FenceMarker` y `ParseDelta`, `Cubre:`, `Tipo: bug` antes del primer `## `). Mención solo en prosa: advertencia `unclassified_mention` y ningún incremento. Una traza ilegible: `trace_invalid`.
  Cubre: prov:trace:7
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`

## Fase 5 (U5): Anclaje y orden

Depende de: Fases 3 y 4.

- [x] 5.1 Crear `anchor.go`: comprobación previa de la referencia con `History.TreeHas` en cada fuente presente (`<prefijo>/changes` y `<prefijo>`) y `RefError{Lane, Ref, Err}`; el mensaje conserva la causa `Err` para distinguir referencia inexistente de otros fallos de `git ls-tree` (hueco design 4). Con varias fuentes y ramas distintas por repositorio, el error nombra el repositorio que no tiene la rama (hueco design 1; limitación ya anotada en la pregunta abierta del design).
  Cubre: prov:trace:12
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 5.2 Anclaje SDD por carpeta archivada (decisión 6): commit de archivado por `FirstParentAdded` de `kickoff.yaml` y, sin resultados, de la carpeta; commit de código por `verify-report.md`, `TreeEntries` con `tasks*.md` e `IsAncestor`; `integrationCommit` con ancla `code`, o `archive`, o `found: false` (no se omite, también en clon superficial). Un archivado sin `kickoff.yaml` o sin `verify-report.md` cae al ancla `archive` o a `found: false` sin error. Dos carpetas con el mismo nombre toman cada una su propio commit.
  Cubre: prov:trace:8
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 5.3 Anclaje ODD (decisión 7): primer commit de `FirstParentAdded(ref, <prefijo>/<feature>.md)` con ancla `document`; sin commit, el documento se omite con `odd_not_integrated`. Con `History == nil`, `odd_repository_unavailable` y se omiten todos los ODD. Cada fuente usa su propia historia (prov:trace:14).
  Cubre: prov:trace:7, prov:trace:8, prov:trace:14
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`
- [x] 5.4 Orden determinista (decisión 9): comparador `IsAncestor` con memoria y ordenación por inserción que propaga errores de git; `found: false` al final; empates por `(lane, path, rol)`; `resolvedIds`, `ids`, `capabilities`, `tasks` y `warnings` ordenados. Prueba con historia falsa de squash conjunto (mismo commit) y comparación del JSON de dos ejecuciones. Documentar en el comentario del comparador que los incrementos de repositorios distintos solo se comparan por la clave estable (hueco spec 10).
  Cubre: prov:trace:13
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`; `go vet ./internal/reqtrace`

## Fase 6 (U6): Collect

Depende de: Fase 5.

- [x] 6.1 Crear `collect.go` con `Input` y `Collect(ctx, in) (Result, error)` que encadena: referencia, lectura, cierre, clasificación, anclaje, orden. Prueba de extremo a extremo con `MapFS` e `History` falso: SDD `created` + ODD `modified` + bug SDD `covered`; ID válido sin apariciones (listas vacías, no `nil`); referencia inexistente (`RefError`); asignación visible en los incrementos de su fuente; consulta por provisional y por definitivo idénticas.
  Cubre: prov:trace:11, prov:trace:15
  Verificación: `go test ./internal/reqtrace -run RequirementTrace -count=1`; `go vet ./internal/reqtrace`

## Fase 7 (U7): Adaptador CLI

Depende de: Fase 6. Paquete: `internal/cli`.

- [x] 7.1 Crear `trace.go` (argumentos): posicional antes o después de las opciones (decisión 13), `--json`, `--integration-branch`, `--cwd`, ayuda con `hasGovernanceHelpFlag`, `gitref.ValidRefName` de la rama. Errores de la tabla de la decisión 12, todos con continuación `axiom …` y por stderr también con `--json`: sin ID, varios posicionales, opción desconocida, `--integration-branch` sin valor o inválida (`../x`, `a..b`, `-x`, `x^`, `x:y`), `--cwd` inexistente o que no es un directorio, directorio fuera de un proyecto Axiom (hueco spec 6, hueco design 8). Ninguna llamada git en estos rechazos.
  Cubre: prov:trace:1, prov:trace:12
  Verificación: `go test ./internal/cli -run AxiomTrace -count=1`
- [x] 7.2 Resolución previa a git: `workspace.Locate`, `projectIDPattern` (sin `axiom.yaml` el patrón por defecto, hueco spec 5; patrón inaccesible con la continuación de `idpattern.go`), `reqtrace.ParseID` y error de ID inválido con ID, `pattern.Template()`, `reqid.ProvisionalText("<serie>","<n>")` y `axiom trace --help`; sin literales `REQ-` ni `prov:` en producción. Prueba con `observeGit`: cero llamadas git con ID inválido.
  Cubre: prov:trace:2
  Verificación: `go test ./internal/cli -run AxiomTrace -count=1`
- [x] 7.3 Fuentes y topología (decisión 10): SDD con `os.DirFS(loc.PlanningHome)`, `gitcommon.TopLevel` y `planningHomePrefix`; ODD solo si `odd/tasks` es directorio, con raíz git propia o `History = nil`. `ref := traceIntegrationBranch(flag, nil, loc.Declarations)`; `*reqtrace.RefError` convertido a error con rama, repositorio y remedio `axiom trace <ID> --integration-branch <rama>`; sin `fetch`. Pasa `reqtrace.NoAliases{}`.
  Cubre: prov:trace:12, prov:trace:14
  Verificación: `go test ./internal/cli -run AxiomTrace -count=1`
- [x] 7.4 Salida: JSON con `encodeJSON(Result)` (solo el documento en stdout) y `renderRequirementTrace` (texto, abreviatura de 12 caracteres, `sin commit en la línea --first-parent de <rama>`, `Ningún cambio SDD archivado ni documento ODD integrado cita <ID>.`, nota ODD solo con incrementos ODD) y `renderRequirementTraceHelp`. Pruebas de formato con resultado construido a mano.
  Cubre: prov:trace:1, prov:trace:9, prov:trace:11
  Verificación: `go test ./internal/cli -run AxiomTrace -count=1`; `go vet ./internal/cli`; `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign -count=1`

## Fase 8 (U8): Integración con git real

Depende de: Fase 7. Fichero: `internal/cli/trace_flow_test.go`.

- [x] 8.1 Ayudantes `traceRepo`, `traceIntegrate` (merge, squash, rebase), `traceDiverge`, sin reutilizar nombres de `sdd_trace_flow_test.go`. Caso SDD archivado que añade el ID y ODD que lo modifica integrados por los tres métodos: ambos `found: true`, SDD antes que ODD; bug SDD con `## Informe de bug` y `Cubre:` como `bug`/`covered`; consulta por provisional y por definitivo iguales; ODD que no parsea (advertencia, salida 0); ODD no integrado omitido; ID válido sin apariciones.
  Cubre: prov:trace:6, prov:trace:7, prov:trace:8, prov:trace:11, prov:trace:15
  Verificación: `go test ./internal/cli -run AxiomTrace -count=1`
- [x] 8.2 Rama: inexistente (error con `--integration-branch`, sin `fetch`) y explícita (`integrationBranch` del JSON igual a ella). Orden determinista: dos ejecuciones con `--json` idénticas byte a byte.
  Cubre: prov:trace:12, prov:trace:13
  Verificación: `go test ./internal/cli -run AxiomTrace -count=1`
- [x] 8.3 Solo lectura: con `observeGit`, solo `rev-parse`, `log`, `ls-tree` y `merge-base`; árbol, índice y lista de ficheros idénticos antes y después (`treeSnapshotWithoutGit`, `git status --porcelain`), con y sin `--json`. Con `t.Chdir` a un repositorio ajeno, git solo en el del proyecto. Nombres inseguros (`2026-01-01-a*`, `-x.md`, `a[1].md`) omitidos sin llamada git con esa ruta.
  Cubre: prov:trace:10
  Verificación: `go test ./internal/cli -run Trace -count=1`
- [x] 8.4 Multirepo con `multirepoSealProject`: carpeta archivada y documento ODD en el repositorio de specs, commits de su historia y rutas relativas a su raíz, git solo en ese repositorio. Variante `specs_repository: openspec` (planning home y `odd/tasks` en repositorios distintos): cada incremento toma el commit de su historia; sin git en la carpeta maestra, `odd_repository_unavailable`. Los `TestSDDTrace*` existentes siguen pasando sin cambios (hueco spec 13).
  Cubre: prov:trace:14
  Verificación: `go test ./internal/cli -run Trace -count=1`

## Fase 9 (U9): Registro en `main`

Depende de: Fase 7.

- [x] 9.1 En `cmd/axiom/main.go`: `case "trace"` junto a `case "archive"`, `runTrace` que envuelve `runSimpleCommand(cli.RunTrace)` y la línea `  trace ` de `axiomHelpText()` junto a `sdd trace`. Crear `trace_help_test.go`: la ayuda global contiene `"\n  trace "`; `runTrace(nil)` devuelve 1 con «requiere un ID»; `runTrace({"--help"})` devuelve 0 con `Uso: axiom trace`. Esto hace alcanzable `reqtrace` desde `main` para el trinquete de código muerto.
  Cubre: prov:trace:1
  Verificación: `go test ./cmd/axiom -run TraceVerb -count=1`; `go vet ./cmd/axiom`

## Fase 10 (U10): Documentación

Depende de: Fase 9.

- [x] 10.1 Manual `docs/manual-de-inicio.md`: subsección `### Trazabilidad de un Requisito (`axiom trace`)` al final de §8, tras «Baseline Retirada y Migración», y una línea tras el bullet de `sdd trace`. Debe documentar: uso y opciones, roles y tipos, que el commit ODD es el que trae el documento y no necesariamente el que añadió su `## Spec viva`, la limitación del bug con requisito mal especificado (sale `increment`), que se lee el árbol de trabajo mientras el anclaje usa la rama de integración (archivados o documentos ausentes del árbol no aparecen; ejecutar desde un worktree de feature da `found: false` o omisión; hueco design 5), el JSON `axiom.trace/v1`, que los errores van por stderr con `--json` y que `axiom sdd trace` no cambia.
  Cubre: prov:trace:6, prov:trace:8, prov:trace:9
  Verificación: `go test ./internal/workspace -run DocsPinManualDeInicio -count=1`
- [x] 10.2 Añadir `"axiom trace <ID>"` a la lista de `TestDocsPinManualDeInicio` en `internal/workspace/docs_pin_test.go`.
  Cubre: prov:trace:1
  Verificación: `go test ./internal/workspace -run DocsPinManualDeInicio -count=1`

---

## Paralelismo y dependencias

Todo es secuencial salvo: U3 y U4 (lectura/clasificación SDD y ODD) pueden ir en cualquier orden entre sí tras U2; U10 puede adelantarse tras U7 (no depende de U8 ni U9 salvo por la redacción). Con un solo escritor y un commit por unidad se recomienda el orden U1 a U10.
