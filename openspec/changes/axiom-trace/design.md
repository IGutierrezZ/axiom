# Diseño: `axiom trace <ID>`, trazabilidad de un requisito de punta a punta (C-03-d)

## Enfoque técnico

Se sigue el enfoque B de la propuesta: un paquete de dominio nuevo, `internal/reqtrace`, sin E/S propia, y un adaptador fino, `internal/cli/trace.go`, que resuelve el proyecto, valida la entrada, construye los puertos y pinta la salida. Es la misma forma que `isoevidence.DeriveTrace` (dominio sobre el puerto `History`, `internal/isoevidence/trace.go:16-33,78`) y `RunSDDTrace` (adaptador, `internal/cli/sdd_trace.go:39-124`).

- **Puertos del dominio.** Dos, ambos ya existentes: `isoevidence.History` para git (`trace.go:16-33`, sin cambios) e `io/fs.FS` de la biblioteca estándar para leer ficheros (`os.DirFS` en el adaptador y `fstest.MapFS` en las pruebas). El dominio no importa `os` ni `os/exec`: la solo-lectura (prov:trace:10) queda fijada por arquitectura con una prueba de importaciones, como `internal/isoevidence/imports_test.go:19-50`.
- **Analizadores reutilizados.** `isoevidence.RequirementKeys` (`internal/isoevidence/reqid.go:16`), `ParseCubre` (`cubre.go:27`), `ParseBugDeclaration` (`bug.go:50`); `openspecdoc.ParseChangeSpec` (`declared.go:9`), `ParseDelta` (`parse.go:49`), `SplitRename` (`parse.go:31`), `FenceMarker` (`compose.go:117`); `livespec.ParseFeatureDoc` (`odddoc.go:191`), `ExtractODDTrace` (`trace.go:151`), `ParseTrace` (`trace.go:52`); `reqid.Pattern.Valid`, `ParseProvisional` (`reqid.go:75`, `provisional.go:155`).
- **Reutilización en `package cli` sin tocar `sdd_trace.go`.** `gitHistory` (`sdd_trace.go:198-284`) implementa el puerto; `traceIntegrationBranch` (`:163`) elige la rama con `sealed = nil`; `planningHomePrefix` (`:129`) da el prefijo de cada fuente; `projectIDPattern` (`idpattern.go:17`) da el patrón; `hasGovernanceHelpFlag` (`sdd_governance.go:64`), `isDirectory` (`:56`) y `encodeJSON` (`sdd_kickoff.go:223`).
- **Registro.** `case "trace"` en `cmd/axiom/main.go` junto a `case "archive"` (`:380`), a través de un `runTrace` probable que envuelve `runSimpleCommand` (`:2241`).

La consulta recalcula todo en cada ejecución: lee el árbol de trabajo y consulta git solo para los incrementos que coinciden.

## Decisiones de arquitectura

### Decisión 1: dominio en `internal/reqtrace` con `isoevidence.History` e `io/fs.FS` como puertos

**Elección**: `reqtrace.Collect(ctx, Input) (Result, error)`. `Input` lleva dos `Source` (SDD y ODD); cada una tiene un `fs.FS` con raíz en su directorio, su prefijo dentro de la raíz git y su `isoevidence.History`.
**Alternativas consideradas**: (a) toda la lógica en `internal/cli/trace.go` (enfoque A): mezcla dominio y adaptador y la sección 07 no podría reutilizarlo; (b) un puerto de ficheros propio (`ReadDir`/`ReadFile`): duplica `io/fs`, que ya da `fstest.MapFS` para las pruebas; (c) ampliar `isoevidence.History`: obligaría a tocar `gitHistory` y el contrato de `sdd trace`.
**Justificación**: el puerto de git existe y basta con sus cuatro métodos (decisiones 6 y 7). `io/fs.FS` es de solo lectura por definición, y sin `os` en el paquete la regla de solo lectura no depende de la disciplina de quien escribe.

### Decisión 2: validar el ID antes de tocar git o escanear

**Elección**: `reqtrace.ParseID(raw, pattern) (Query, error)`. Primero `pattern.Valid(raw)`; si falla, `pattern.ParseProvisional(raw)`; si falla también, devuelve `*reqtrace.IDError{ID, Template, Code: "id_invalid", Cause}`. El ID se compara tal cual, sin normalizar mayúsculas, espacios ni ceros (un `REQ-01.1` no es `REQ-1.1`). El adaptador llama a `ParseID` después de `workspace.Locate` y de `projectIDPattern`, que solo leen ficheros (`gitcommon.LinkedWorktreeRoot` lee ficheros, `internal/gitcommon/files.go:166`), y antes de cualquier `gitcommon.TopLevel`.
**Alternativas consideradas**: validar dentro de `Collect`, que obligaría a resolver antes las raíces git (y por tanto a ejecutar `git rev-parse`); normalizar el ID, que contradice la comparación por token exacto de `reqid`.
**Justificación**: prov:trace:2 exige el rechazo «sin consultar git ni escanear ningún fichero». Sin `axiom.yaml` el patrón es el de por defecto; con un `axiom.yaml` cuyo patrón es inaccesible, `projectIDPattern` ya se niega con la continuación `axiom workspace validate --path …` (`idpattern.go:45-51`).

### Decisión 3: resolución de equivalentes por cierre sobre las trazas y un punto de extensión

**Elección**:
1. Se leen todas las trazas: el `ids.yaml` de cada carpeta archivada (`livespec.ParseTrace`, porque `ReadTraceFile` usa `os`) y la sección `## Identificadores asignados` de cada documento ODD (`livespec.ExtractODDTrace`).
2. Semillas: con un ID definitivo, el propio ID. Con un provisional, los definitivos de las entradas cuyo `Provisional` coincide; si no hay ninguna, el propio provisional; si hay más de un definitivo distinto, se añaden todos y la advertencia `provisional_ambiguous` nombra las fuentes (los provisionales son locales a un cambio y se repiten entre cambios).
3. Cierre hasta punto fijo sobre un conjunto: toda entrada cuyo `Definitive` o algún `Previous` esté en el conjunto aporta su `Definitive` y sus `Previous`, y cada ID nuevo pasa una vez por `AliasResolver.Aliases(id)` (se descartan los alias que no cumplen el patrón). Las cadenas (A→B→C) y los ciclos terminan porque solo se añade lo que no estaba.
4. `resolvedIds` = el cierre ∪ los provisionales de las entradas que lo tocan ∪ el ID consultado, sin repetidos y en orden de bytes. Consultar por el provisional o por el definitivo de una asignación produce el mismo conjunto.
5. Conjunto de coincidencia por fuente: el cierre de definitivos, el ID consultado y los provisionales de las entradas de esa misma fuente. El provisional de otra fuente no cuenta, para que un `prov:<serie>:1` repetido en otro cambio no traiga falsos positivos.

```go
// AliasResolver es el punto de extensión de equivalentes (C-03-e añadirá el mapa de IDs antiguos).
type AliasResolver interface{ Aliases(id string) []string }
// NoAliases es la implementación por defecto: no añade nada más allá de las trazas.
type NoAliases struct{}
```

**Alternativas consideradas**: una función `func(string) []string` (equivalente, pero la interfaz deja nombrar la implementación por defecto); un `union-find` global que incluya los provisionales, que fusionaría cambios distintos con el mismo provisional.
**Justificación**: cumple prov:trace:3 y prov:trace:4. El adaptador pasa `reqtrace.NoAliases{}`, que llega a `main` y no queda como código muerto (`scripts/deadcode-ratchet.sh` mide desde `./cmd/axiom`). Un `Previous` compartido por dos entradas fusiona sus IDs, que es lo que la spec manda («los `Previous` cuentan como equivalentes»).

### Decisión 4: detección y rol por fuente, por token exacto

Toda comparación usa los tokens que devuelve `isoevidence.RequirementKeys` (válidos y provisionales) o `reqid.Pattern.Scan`, comparados por igualdad exacta con el conjunto de coincidencia: `REQ-1.1` nunca casa con `REQ-1.10` ni con `REQ-11.1`.

**SDD** (`changes/archive/<fecha>-<cambio>/`; los cambios activos no se leen):

| Aparición | Origen | Rol |
|---|---|---|
| Título de un requisito ADDED en `specs/*/spec.md` (`openspecdoc.ParseChangeSpec`: una especificación completa cuenta como ADDED) | delta | `created` |
| Título MODIFIED | delta | `modified` |
| Título REMOVED | delta | `removed` |
| Título RENAMED, nombre origen o destino (`openspecdoc.SplitRename`; si no separa, el título entero) | delta | `renamed` |
| Línea `Cubre:` de `tasks.md` o `tasks.<rol>.md` (`isoevidence.ParseCubre`, fuera de vallas, varios IDs por línea) | tareas | `covered` |
| Solo una entrada de `ids.yaml` (sin aparición en la delta) | traza | `created` |

Los ficheros de tareas se eligen por nombre (`^tasks(\.[^.]+)?\.md$`, la regla de `isoevidence/trace.go:164-174`), no por el `kickoff.yaml`, que falta en cambios antiguos. Tipo: `bug` si `isoevidence.ParseBugDeclaration(proposal.md).Kind != BugKindNone`; si no, o sin `proposal.md`, `increment`. Se aplica a todos los roles del cambio. Un bug con requisito mal especificado (delta y sin secciones de bug) sale como `increment`: limitación documentada en la ayuda y el manual.

**ODD** (`odd/tasks/*.md`): con `livespec.ParseFeatureDoc` correcto, roles por la delta de cada `### Capacidad:` con la misma tabla, `covered` por las líneas `Cubre:` del documento entero y `created` por una entrada de `## Identificadores asignados` sin aparición en la delta. Tipo `bug` si `Kind == livespec.KindBug`. Los documentos anteriores a la traza se encuentran por los títulos de su delta, sin error.

**ODD tolerante.** Si `ParseFeatureDoc` falla, se añade la advertencia `odd_unparseable` con la ruta y la búsqueda sigue en el texto: la delta de cada bloque vallado con etiqueta `openspec` (con `openspecdoc.FenceMarker` y `openspecdoc.ParseDelta`, o el texto entero si no hay bloques) da los roles de la tabla, las líneas `Cubre:` dan `covered` y el tipo sale de una línea `Tipo: bug` antes del primer `## `. Si el ID solo aparece en prosa, fuera de un título de requisito o de una línea `Cubre:`, no hay rol que asignar: se añade la advertencia `unclassified_mention` con la ruta y no se crea incremento. Así el rol sigue dentro del conjunto cerrado de prov:trace:9.

**Un incremento por (fuente, rol).** Cada incremento lleva los IDs que coincidieron con ese rol (sin repetidos: un ID presente como provisional y como definitivo cuenta una vez por token) y las capacidades. Solo el rol `covered` lista tareas. Las asignaciones de la traza que tocan el cierre se adjuntan a todos los incrementos de esa fuente.

**Alternativas consideradas**: un sexto rol `mentioned` para la prosa (se sale del conjunto cerrado de la spec); usar `openspecdoc.Declared` (solo mira el nombre origen de un RENAMED, `declared.go:57-63`, y prov:trace:5 pide los dos); leer las tareas desde `kickoff.RoleTaskFiles` (falla en archivados sin `kickoff.yaml`).
**Justificación**: los analizadores ya son los del archivado y del status, de modo que el verbo ve los mismos requisitos que vio el cambio.

### Decisión 5: tolerancia; advertencias en lugar de errores

**Elección**: un fichero ausente, ilegible o mal formado nunca aborta la consulta. Los códigos de `warnings` son estables:

| Código | Cuándo | Efecto |
|---|---|---|
| `odd_unparseable` | `ParseFeatureDoc` falla | búsqueda tolerante (decisión 4) |
| `unclassified_mention` | el ID aparece solo en prosa | sin incremento |
| `trace_invalid` | `ids.yaml` o `## Identificadores asignados` ilegibles | se ignora esa traza |
| `provisional_ambiguous` | un provisional consultado con varios definitivos | se resuelven todos |
| `odd_not_integrated` | un documento ODD que coincide no está en la línea `--first-parent` | se omite (prov:trace:7) |
| `odd_repository_unavailable` | `odd/tasks` existe pero no está en un repositorio git | se omiten todos los ODD |
| `unsafe_name` | una carpeta archivada o un documento con un nombre fuera del juego seguro | se omite sin tocar git |
| `unreadable` | un error de lectura que no es «no existe» | se omite ese fichero |

Un `proposal.md`, un `tasks*.md` o una delta ausentes se tratan como vacíos (sin advertencia); un `kickoff.yaml` ausente se cubre en el anclaje (decisión 6). Cada advertencia es `{code, path, message}` con la ruta relativa a la raíz git; el mensaje lo compone `reqtrace`, que no entra en el trinquete de rechazos (`refusal_resolution_ratchet_test.go:773-777` audita `internal/cli`, `internal/fileprim` e `internal/sddstatus`), y aun así nombra la acción cuando la hay.
**Alternativas consideradas**: error con salida distinta de 0 ante cualquier dato roto (contradice prov:trace:7 y prov:trace:8).
**Justificación**: hay 94 documentos ODD y 61 carpetas archivadas en este repositorio; la mayoría anteriores a la traza. La consulta es de diagnóstico: informa de lo que hay.

### Decisión 6: anclaje SDD por carpeta archivada, con el algoritmo de `DeriveTrace`

**Elección**: para la carpeta `<p>/changes/archive/<fecha>-<cambio>` (`<p>` = prefijo del planning home):
1. Commit de archivado: el primero de `FirstParentAdded(ref, <carpeta>/kickoff.yaml)` o, sin resultados, de `FirstParentAdded(ref, <carpeta>)`, como `trace.go:108-111`.
2. Commit de código: de `FirstParentAdded(ref, <p>/changes/<cambio>/verify-report.md)`, del más reciente al más antiguo, el primero cuyo árbol (`TreeEntries(c, <p>/changes/<cambio>)`) tiene `tasks.md` o `tasks.<rol>.md` y que, si hay commit de archivado, es ancestro suyo (`IsAncestor(c, archivado)`).
3. `integrationCommit` = el commit de código con `anchor: "code"`; sin él, el de archivado con `anchor: "archive"` (un cambio archivado en la misma PR que su código); sin ninguno, `found: false`. `archiveCommit` se informa aparte, con su propio `found`.

Una carpeta archivada que la línea `--first-parent` no contiene se informa con `found: false` y no se omite: prov:trace:8 lo pide así para SDD, y la omisión es solo para ODD (prov:trace:7). Un clon superficial que corta la historia da el mismo `found: false`, nunca un error.
**Alternativas consideradas**: llamar a `isoevidence.DeriveTrace` (elige la carpeta más reciente del nombre, `trace.go:143-162`, y el candidato de código más antiguo: confunde dos carpetas con el mismo nombre de cambio, que prov:trace:5 prohíbe); exportar una variante en `isoevidence` (la propuesta deja ese paquete sin cambios).
**Justificación**: con un nombre de cambio único el resultado coincide con `sdd trace`; con nombres repetidos, cada carpeta toma el código integrado justo antes de su archivado.

### Decisión 7: anclaje ODD por el documento; sin pickaxe

**Elección**: `integrationCommit` de un ODD = primer commit de `FirstParentAdded(ref, <prefijo>/<feature>.md)`, con `anchor: "document"`. Sin commit, el documento se omite con `odd_not_integrated`. La ayuda, el texto (nota final cuando hay incrementos ODD) y el manual dicen que es el commit que trae el documento y no necesariamente el que añadió su `## Spec viva`.
**Alternativas consideradas**: pickaxe (`git log --first-parent -S<ID> -- openspec/specs/<cap>/spec.md`). Se descarta porque: `-S` solo detecta commits que cambian el número de apariciones del texto, así que un MODIFIED, que conserva el ID, nunca aparece; exigiría un método nuevo en el puerto (o un segundo puerto) y en `gitHistory`; y añade un proceso git por capacidad y coincidencia.
**Justificación**: con merge, squash y rebase el primer commit de la línea `--first-parent` que contiene el documento es la fusión, el squash o el último commit rebasado que lo añade: cumple el criterio de terminado con el puerto existente. Es la opción mínima.

### Decisión 8: rama de integración y comprobación previa de la referencia

**Elección**: `ref := traceIntegrationBranch(flag, nil, loc.Declarations)` (`sdd_trace.go:163-173`: la indicada, la de `axiom.yaml` o `main`) y `gitref.ValidRefName(ref)` antes de llamar a git. Al empezar, `Collect` comprueba la referencia en el repositorio de cada fuente presente con `History.TreeHas(ctx, ref, <prefijo>/changes)` (SDD) o `TreeHas(ctx, ref, <prefijo>)` (ODD): `git ls-tree` falla con una referencia inexistente y devuelve `false` sin error si la ruta no está. Un fallo es `*reqtrace.RefError{Lane, Ref, Err}`, y el adaptador lo convierte en un error que nombra la rama y el repositorio y da el remedio `axiom trace <ID> --integration-branch <rama>`. Nunca hay `fetch`.
**Alternativas consideradas**: dejar que falle el primer `FirstParentAdded` (con un ID sin coincidencias no se llamaría a git y una rama inexistente pasaría por resultado vacío, contra prov:trace:12); un método nuevo `RefExists` en el puerto.
**Justificación**: dos llamadas de lectura baratas que reutilizan el puerto y hacen que prov:trace:12 se cumpla siempre.

### Decisión 9: orden determinista con `IsAncestor`

**Elección**: los incrementos con `integrationCommit.found` van primero, ordenados por la posición de su commit en la línea `--first-parent` con el comparador `a ≺ b ⇔ a ≠ b ∧ IsAncestor(a, b)`. Es un orden total entre commits de una misma línea `--first-parent`, con memoria de las comparaciones y una ordenación por inserción que propaga los errores de git. Los empates (mismo commit, por ejemplo un squash conjunto) y los `found: false`, que van al final, se ordenan por `(lane, path, rol)`, con `lane` y `path` en orden de bytes y el rol en el orden `created, modified, removed, renamed, covered`. `resolvedIds`, `ids`, `capabilities`, `tasks` (fichero y línea) y `warnings` (ruta, código y mensaje) se ordenan también. `fs.ReadDir` ya devuelve las entradas ordenadas y no se itera ningún mapa para emitir.
**Alternativas consideradas**: un método `FirstParentLine` (`git rev-list --first-parent`) en el puerto: un solo proceso, pero cambia `isoevidence.History` y `gitHistory`; ordenar por fecha del commit (no determinista con rebase y no es la posición en la línea).
**Justificación**: k commits distintos cuestan como mucho k(k-1)/2 lecturas `merge-base --is-ancestor`, con k pequeño (los incrementos de un requisito). En multirepo cada fuente se ordena con su propia historia y los incrementos de repositorios distintos solo se comparan por la clave estable.

### Decisión 10: topología; cada fuente con su raíz git

**Elección**: el adaptador construye cada `Source` por separado:
- SDD: `Files = os.DirFS(loc.PlanningHome)`; raíz git = `gitcommon.TopLevel(loc.PlanningHome)` (o de `loc.WorkspaceRoot` si el planning home no existe); prefijo = `planningHomePrefix(raíz, loc.PlanningHome)`. Sin raíz git, error con remedio, como `sdd_trace.go:86-89`.
- ODD: si `<loc.WorkspaceRoot>/odd/tasks` no es un directorio, fuente ausente. Si existe, raíz git propia con `TopLevel` y su prefijo; si no está en un repositorio git, `History = nil` y el dominio emite `odd_repository_unavailable`.

Cada `Source` lleva su `gitHistory{root}`, y `path` y `tasks[].file` son relativos a la raíz git que los contiene (prov:trace:14). En el código actual `odd/tasks` cuelga de `loc.WorkspaceRoot` (`internal/cli/archive_integrate.go:77`, `internal/branch/engine/guard.go:141`). En `multirepo`, `workspace.Roots` (`internal/workspace/locate.go:386,459-467`) hace de `WorkspaceRoot` el repositorio de specs, y entonces las dos fuentes comparten repositorio; pero si `specs_repository` se llama `openspec`, el planning home es ese repositorio y `WorkspaceRoot` es su padre (la carpeta maestra), de modo que `odd/tasks` puede vivir en otro repositorio o en ninguno. El anclaje por fuente cubre los dos casos. Se usa una sola rama de integración para todas las fuentes.
**Alternativas consideradas**: una sola raíz git para todo (mezclaría historias si `odd/tasks` y `openspec` llegaran a estar en repositorios distintos).
**Justificación**: anclar por fuente cuesta una llamada `rev-parse` más y cubre las dos topologías sin ramas de código distintas.

### Decisión 11: nombres seguros antes de git

**Elección**: las rutas que llegan a git salen de nombres de directorio y de fichero del árbol de trabajo. `reqtrace` solo acepta carpetas archivadas `^\d{4}-\d{2}-\d{2}-[A-Za-z0-9][A-Za-z0-9._-]*$` y documentos `^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`, sin `..`; el resto se omite con `unsafe_name`. Así ningún nombre lleva magia de pathspec (`:(`), comodines (`*?[`) ni un guion inicial. Los 61 archivados y los 94 documentos actuales cumplen la regla.
**Alternativas consideradas**: `--literal-pathspecs` en `gitHistory` (cambia `sdd_trace.go`).
**Justificación**: defensa en el dominio sin tocar el adaptador git compartido; `gitHistory` ya valida las revisiones (`sdd_trace.go:206-211`).

### Decisión 12: errores del adaptador con continuación nombrada

**Elección**: todo rechazo de `internal/cli/trace.go` nombra un `axiom …` ejecutable o es un `%w` de propagación, para pasar el trinquete `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` (`refusal_resolution_ratchet_test.go:617`) sin tocar `.refusal-ratchet-baseline.txt`:

| Caso | Mensaje (resumen) | Salida |
|---|---|---|
| Sin ID | `trace requiere un ID de requisito; ejecuta `axiom trace <ID> [--json] [--integration-branch <rama>] [--cwd <dir>]`` | 1 |
| Varios posicionales u opción desconocida | `trace: argumento inesperado %q; ejecuta `axiom trace --help`` | 1 |
| `--integration-branch` no válida | `trace: --integration-branch %q no es una referencia válida; ejecuta `axiom trace <ID> --integration-branch <rama>` con un nombre de rama` | 1 |
| ID no válido (`*reqtrace.IDError`, envuelto con `%w`) | nombra el ID, `pattern.Template()`, la forma `reqid.ProvisionalText("<serie>", "<n>")` y `axiom trace --help` | 1 |
| Rama inexistente (`*reqtrace.RefError`) | nombra la rama y el repositorio y el remedio `axiom trace <ID> --integration-branch <rama>` | 1 |
| Patrón inaccesible | el de `projectIDPattern` (`axiom workspace validate --path …`) | 1 |

Los errores van siempre por stderr como texto, también con `--json`, como `sdd trace` y `runSimpleCommand` (`main.go:2241-2247`): stdout solo lleva el documento JSON de un resultado. Ningún literal de producción contiene `REQ-` seguido de algo que no sea una cifra ni `prov:` (pruebas `TestNoRequirementIDFormatOutsideReqid` y `TestProvisionalLiteralOnlyInReqid` de `internal/reqid/arch_test.go:57,124`): la ayuda y los mensajes usan `pattern.Template()`, `reqid.Default().Template()` y `reqid.ProvisionalText`.
**Alternativas consideradas**: un JSON de error con `--json` (ningún verbo vecino lo hace y la spec no lo pide); marcas `refusal:by-design` (aquí siempre existe una continuación ejecutable).

### Decisión 13: argumentos con posicional antes o después de las opciones

**Elección**: si `args[0]` no empieza por `-`, es el ID y el resto se analiza con `flag`; si no, se analizan las opciones y se exige exactamente un posicional. `--help`/`-h` en cualquier posición imprime la ayuda (`hasGovernanceHelpFlag`).
**Justificación**: `flag.Parse` se detiene en el primer posicional, y la forma documentada es `axiom trace <ID> --json`.

## Flujo de datos

```
axiom trace <ID> [--json] [--integration-branch r] [--cwd d]
        │
cmd/axiom main ── case "trace" ──> runTrace ──> runSimpleCommand(cli.RunTrace)
        │
cli.RunTrace
  1. ayuda / opciones / ID presente / ValidRefName(flag)        (sin git)
  2. workspace.Locate(cwd) ─> projectIDPattern ─> reqtrace.ParseID   (sin git)
  3. ref = traceIntegrationBranch(flag, nil, decl)
  4. fuentes: SDD{DirFS(planningHome), prefijo, gitHistory{raíz}}
              ODD{DirFS(odd/tasks),    prefijo, gitHistory{raíz} | nil}
  5. reqtrace.Collect(ctx, Input{Query, Pattern, Ref, NoAliases{}, SDD, ODD})
        │
reqtrace.Collect
  a. comprobar ref ──(TreeHas)──────────────────────────> History (git ls-tree)
  b. leer fuentes (fs.ReadDir/ReadFile): archivados, docs ODD, trazas
  c. resolver equivalentes: semillas → cierre (trazas + AliasResolver)
  d. clasificar: (fuente, rol) → incrementos + warnings
  e. anclar: SDD FirstParentAdded/TreeEntries/IsAncestor;
             ODD FirstParentAdded (omitir si no integrado) ──> History (git log, ls-tree, merge-base)
  f. ordenar (IsAncestor) → Result{schema axiom.trace/v1}
        │
cli: --json → encodeJSON(Result) | texto → renderRequirementTrace
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/reqtrace/doc.go` | Crear | Documentación del paquete y `Schema = "axiom.trace/v1"`. |
| `internal/reqtrace/model.go` | Crear | `Result`, `Increment`, `Commit`, `Task`, `Assignment`, `Warning`; constantes `Lane`, `Kind`, `Role`, `Anchor`; códigos de advertencia. |
| `internal/reqtrace/id.go` | Crear | `Query`, `ParseID`, `IDError`. |
| `internal/reqtrace/alias.go` | Crear | `AliasResolver`, `NoAliases`, cierre de equivalentes. |
| `internal/reqtrace/scan.go` | Crear | `Source`; lectura de archivados SDD y documentos ODD con `fs.FS`; nombres seguros. |
| `internal/reqtrace/classify.go` | Crear | Roles, tipo, tareas, asignaciones; ODD tolerante. |
| `internal/reqtrace/anchor.go` | Crear | Comprobación de la referencia (`RefError`), anclaje SDD y ODD, orden. |
| `internal/reqtrace/collect.go` | Crear | `Input`, `Collect`. |
| `internal/reqtrace/*_test.go` | Crear | Unitarias con `fstest.MapFS` y un `History` falso; `imports_test.go`. |
| `internal/cli/trace.go` | Crear | `RunTrace`, argumentos, fuentes, `renderRequirementTrace`, `renderRequirementTraceHelp`. |
| `internal/cli/trace_test.go`, `internal/cli/trace_flow_test.go` | Crear | Adaptador e integración con repositorios git reales. |
| `internal/cli/sdd_trace.go` | Sin cambios | Se reutilizan sus funciones; `axiom.sdd-trace/v1` intacto. |
| `cmd/axiom/main.go` | Modificar | `case "trace"`, `runTrace` y línea `  trace ` en `axiomHelpText()` junto a `sdd trace` (`:147`). |
| `cmd/axiom/trace_help_test.go` | Crear | Ayuda global, `trace --help` y enrutado sin ID. |
| `docs/manual-de-inicio.md` | Modificar | Subsección `### Trazabilidad de un Requisito (`axiom trace`)` al final de §8, tras «Baseline Retirada y Migración» (no tras «La Spec Viva es OpenSpec», cuyo texto remite a «la sección siguiente», la de adopción); una línea tras el bullet de `sdd trace` (`:412`). |
| `internal/workspace/docs_pin_test.go` | Modificar | Añadir `"axiom trace <ID>"` a la lista de `TestDocsPinManualDeInicio` (`:16-24`). |

## Interfaces y contratos

### Dominio (`internal/reqtrace`)

```go
const Schema = "axiom.trace/v1"

type Lane string   // LaneSDD = "sdd", LaneODD = "odd"
type Kind string   // KindIncrement = "increment", KindBug = "bug"
type Role string   // RoleCreated, RoleModified, RoleRemoved, RoleRenamed, RoleCovered
type Anchor string // AnchorCode = "code", AnchorArchive = "archive", AnchorDocument = "document"

type Query struct {
	ID          string
	Provisional bool
}

// ParseID valida el ID contra el patrón: válido, provisional válido o *IDError.
func ParseID(raw string, p reqid.Pattern) (Query, error)

type IDError struct {
	ID, Template string
	Cause        error // el *reqid.ProvisionalError si tenía forma de provisional
}

type AliasResolver interface{ Aliases(id string) []string }
type NoAliases struct{}

type Source struct {
	Files     fs.FS               // nil: fuente ausente
	GitPrefix string              // raíz de Files relativa a la raíz git, con '/'
	History   isoevidence.History // nil con Files: sin repositorio git
}

type Input struct {
	Query   Query
	Pattern reqid.Pattern
	Ref     string
	Aliases AliasResolver
	SDD     Source // raíz en el planning home: changes/archive/...
	ODD     Source // raíz en odd/tasks
}

type RefError struct {
	Lane Lane
	Ref  string
	Err  error
}

func Collect(ctx context.Context, in Input) (Result, error)
```

### JSON `axiom.trace/v1`

Las listas nunca son `null`. Los campos con `omitempty` faltan cuando no aplican.

```json
{
  "schema": "axiom.trace/v1",
  "id": "REQ-30.1",
  "resolvedIds": ["REQ-30.1"],
  "integrationBranch": "main",
  "increments": [
    {
      "lane": "sdd",
      "name": "demo",
      "path": "openspec/changes/archive/2026-11-12-demo",
      "type": "increment",
      "role": "created",
      "ids": ["REQ-30.1"],
      "capabilities": ["pagos"],
      "tasks": [],
      "integrationCommit": {"commit": "<sha>", "found": true, "anchor": "code"},
      "archiveCommit": {"commit": "<sha>", "found": true},
      "assignments": [{"provisional": "<provisional>", "definitive": "REQ-30.1", "capability": "pagos"}]
    },
    {
      "lane": "odd",
      "name": "cobro",
      "path": "odd/tasks/cobro.md",
      "type": "increment",
      "role": "covered",
      "ids": ["REQ-30.1"],
      "tasks": [{"file": "odd/tasks/cobro.md", "line": 12}],
      "integrationCommit": {"commit": "<sha>", "found": true, "anchor": "document"}
    }
  ],
  "warnings": [
    {"code": "odd_unparseable", "path": "odd/tasks/viejo.md", "message": "..."}
  ]
}
```

| Campo | Tipo | Notas |
|---|---|---|
| `id` | string | El ID tal como se consultó. |
| `resolvedIds` | string[] | Cierre de equivalentes (decisión 3), en orden de bytes. |
| `increments[].lane` | `sdd` \| `odd` | |
| `increments[].name` | string | Cambio (carpeta sin fecha) o feature (sin `.md`). |
| `increments[].path` | string | Relativa a la raíz git que la contiene, con `/`. |
| `increments[].type` | `increment` \| `bug` | |
| `increments[].role` | `created` \| `modified` \| `removed` \| `renamed` \| `covered` | Un incremento por rol. |
| `increments[].ids` | string[] | IDs del cierre que coincidieron con ese rol. |
| `increments[].capabilities` | string[] | Solo en roles de delta. |
| `increments[].tasks` | `{file, line}`[] | Solo en `covered`; `file` relativo a la raíz git. |
| `increments[].integrationCommit` | `{commit?, found, anchor?}` | `anchor`: `code`, `archive` (SDD) o `document` (ODD). |
| `increments[].archiveCommit` | `{commit?, found}` | Solo SDD. |
| `increments[].assignments` | `{provisional, definitive, capability?, previous?}`[] | Entradas de traza de la fuente que tocan el cierre. |
| `warnings` | `{code, path?, message}`[] | Códigos de la decisión 5. |

El esquema es versionado: añadir un campo opcional es compatible; quitar o cambiar uno exige `axiom.trace/v2`.

### Texto

No es un contrato estable. Forma:

```
Traza de REQ-30.1 en main
IDs resueltos: REQ-30.1
Incrementos (2):
  1. sdd demo [increment, created] 1a2b3c4d5e6f (código)
     openspec/changes/archive/2026-11-12-demo
  2. odd cobro [increment, covered] 9f8e7d6c5b4a (documento)
     odd/tasks/cobro.md
     Cubre: odd/tasks/cobro.md:12
Advertencias:
  - [odd_unparseable] odd/tasks/viejo.md: ...
Nota: en ODD el commit es el primero de la línea --first-parent que trae odd/tasks/<feature>.md, no necesariamente el que añadió su `## Spec viva`.
```

Sin commit: `sin commit en la línea --first-parent de <rama>`. Sin incrementos: `Ningún cambio SDD archivado ni documento ODD integrado cita <ID>.` La nota ODD solo aparece si hay incrementos ODD. El commit se abrevia a 12 caracteres.

### Verbo

```
Uso: axiom trace <ID> [--json] [--integration-branch <rama>] [--cwd <dir>]
```

Línea de la ayuda global, junto a `sdd trace`:
`  trace                Traza un requisito por su ID: incrementos SDD archivados y ODD integrados, bugs, tareas y commits --first-parent (--json)`

## Estrategia de pruebas

Todas las pruebas son localizadas. Los nombres llevan los prefijos `TestRequirementTrace` (dominio), `TestAxiomTrace` (adaptador) y `TestTraceVerb` (`cmd/axiom`), así que `-run Trace` las incluye y también las `TestSDDTrace*` existentes, que deben seguir pasando sin cambios. Los ayudantes de prueba nuevos no reutilizan nombres de `package cli` (`runTrace`, `traceDelta`, `traceTasks` ya existen en `sdd_trace_flow_test.go`).

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitaria `reqtrace` | `ParseID`: válido, provisional, inválido, patrón `RF-<DOM>-<NN>` | Tabla; `go test ./internal/reqtrace -run RequirementTrace -count=1` |
| Unitaria | Cierre: provisional y definitivo dan lo mismo; cadena y ciclo de `Previous`; provisional ambiguo; alias inyectado; `NoAliases` | Trazas en `fstest.MapFS` |
| Unitaria | SDD: cuatro roles de delta, RENAMED por origen y por destino, especificación completa como ADDED, `Cubre:` en `tasks.md` y `tasks.<rol>.md` con varias líneas, solo `ids.yaml`, bug `existing_requirement` y `no_impact`, sin `proposal.md`, dos carpetas con el mismo nombre, cambio activo ignorado, `REQ-1.1` frente a `REQ-1.10` | `fstest.MapFS` y un `History` falso que registra las llamadas |
| Unitaria | ODD: delta, `Tipo: bug`, traza, documento sin traza, documento que no parsea (advertencia y roles por búsqueda tolerante), mención solo en prosa (`unclassified_mention`) | Igual |
| Unitaria | Anclaje: código y archivado; archivado sin `kickoff.yaml`; código ausente (ancla `archive`); ninguno (`found: false`); ODD no integrado omitido; referencia inexistente (`RefError`); dos historias distintas por fuente (prov:trace:14) | `History` falso por fuente |
| Unitaria | Orden: cronológico, `found: false` al final, empates por `(lane, path, rol)`, dos ejecuciones idénticas | Comparando el JSON serializado |
| Unitaria | Arquitectura: sin `os`, `os/exec` ni `internal/gitcommon` | `imports_test.go`, como `isoevidence/imports_test.go` |
| Unitaria | Nombres inseguros (`2026-01-01-a*`, `-x.md`, `a[1].md`, `..`) omitidos con `unsafe_name` y sin llamada a `History` con esa ruta | `History` falso |
| Integración `cli` | ID inválido: error con el patrón y la continuación, código 1, ninguna llamada git (`observeGit`) | `traceRepo`, `writeSDDAxiomYAML`; `go test ./internal/cli -run AxiomTrace -count=1` |
| Integración | Sin ID, posicional extra, `--help`, ID válido sin apariciones (lista vacía), rama inexistente (error con `--integration-branch`, sin `fetch`), `--integration-branch` explícita | Igual |
| Integración | SDD archivado que añade el ID y ODD que lo modifica, integrados por merge, squash y rebase: los dos con `found: true` y el SDD antes que el ODD; bug SDD con `## Informe de bug` y `Cubre:` como `bug`/`covered`; consulta por provisional y por definitivo iguales; ODD que no parsea (advertencia, salida 0); ODD no integrado omitido | `traceRepo`, `traceIntegrate`, `traceDiverge`; documentos ODD escritos en las ramas de fixture |
| Integración | Solo lectura: solo `rev-parse`, `log`, `ls-tree` y `merge-base`; árbol, índice y lista de ficheros iguales antes y después, con y sin `--json` | `observeGit`, `treeSnapshotWithoutGit`, `git status --porcelain` |
| Integración | Repositorio de `--cwd`: con `t.Chdir` a un repositorio ajeno, git solo se ejecuta en el del proyecto | Como `sdd_trace_test.go:85-108` |
| Integración | Multirepo: carpeta archivada y documento ODD en el repositorio de specs; commits de su historia y rutas relativas a su raíz; git solo en ese repositorio | `multirepoSealProject` (`sdd_kickoff_branch_test.go:428`) |
| Integración | Multirepo con `specs_repository: openspec`: el planning home es un repositorio y `odd/tasks` cuelga de la carpeta maestra, que es otro repositorio; cada incremento toma el commit de su historia y su ruta es relativa a su raíz; sin git en la carpeta maestra, `odd_repository_unavailable` | Repositorios creados con `traceRepo`/`traceGit` y un `axiom.yaml` multirepo propio |
| `cmd/axiom` | `axiomHelpText()` contiene `"\n  trace "`; `runTrace(nil)` devuelve 1 con «requiere un ID»; `runTrace({"--help"})` devuelve 0 con `Uso: axiom trace` | `go test ./cmd/axiom -run TraceVerb -count=1` |
| Documentación | El manual cita `axiom trace <ID>` | `go test ./internal/workspace -run DocsPinManualDeInicio -count=1` |

Además: `go vet ./internal/reqtrace ./internal/cli ./cmd/axiom` y `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign -count=1` (trinquete de rechazos). El trinquete de código muerto se valida en CI (`scripts/deadcode-ratchet.sh`): todas las funciones de `reqtrace` deben alcanzarse desde `RunTrace`, que solo es alcanzable desde `main` tras la unidad U9. Nunca se ejecuta `./internal/cli` entero ni la batería completa en local; no se compilan binarios sueltos (las pruebas llaman a `RunTrace` y a `runTrace` en proceso).

## Matriz de amenazas

El verbo ejecuta git (procesos) y añade una ruta de enrutado.

| Frontera | Casos adversos mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con aspecto de documentación | `requirements.txt`, Markdown ejecutable, `README.sh` | N/A: los `.md` y `.yaml` se leen como datos con `fs.FS`; nada se ejecuta ni se clasifica como ejecutable | — | — |
| Selección del repositorio git | `git -C`, rutas relativas, rutas absolutas | Aplicable | git solo a través de `gitcommon.Run` con la raíz absoluta de `TopLevel` de cada fuente, nunca el directorio del proceso; revisiones validadas (`gitref.ValidRefName` y `revision`); rutas construidas con nombres del juego seguro (decisión 11), siempre tras `--`; ID y referencia validados antes de llamar a git | (1) `t.Chdir` a un repositorio ajeno: todas las llamadas en la raíz del proyecto; (2) `--integration-branch` con `../x`, `a..b`, `-x`, `x^`, `x:y`: rechazo sin llamadas git; (3) ID inválido sin llamadas git; (4) carpeta `2026-01-01-a*` y documentos `-x.md`, `a[1].md`: omitidos con `unsafe_name` y sin llamada a `History` con esa ruta; (5) multirepo: llamadas solo en el repositorio de specs |
| Estado del commit | `commit -a`, índice vacío o con cambios | N/A como operación (el verbo no escribe); la garantía de solo lectura se prueba igualmente en prov:trace:10 | Solo `rev-parse`, `log`, `ls-tree`, `merge-base` | Cubierto por la prueba de solo lectura |
| Estado del push | rama de seguimiento, primer push, refspec | N/A: no hay push ni `fetch` | — | — |
| Órdenes de PR | `--head`, prefijos de entorno | N/A: no hay automatización de PR | — | — |

## Unidades de trabajo

Una unidad por commit (Conventional Commits), cada una con su código y sus pruebas, con el objetivo de unas 100 líneas o menos:

| Unidad | Alcance | Verificación |
|---|---|---|
| U1 | `reqtrace`: `doc.go`, `model.go`, `id.go`, `imports_test.go`, pruebas de `ParseID` | `go test ./internal/reqtrace -run RequirementTrace -count=1` |
| U2 | `alias.go`: cierre de equivalentes, `NoAliases`, `provisional_ambiguous` | Igual |
| U3 | `scan.go` (SDD) y la parte SDD de `classify.go`: archivados, deltas, tareas, `ids.yaml`, bug, nombres seguros | Igual |
| U4 | Parte ODD de `scan.go`/`classify.go`: `ParseFeatureDoc`, traza, búsqueda tolerante, advertencias | Igual |
| U5 | `anchor.go`: `RefError`, anclaje SDD y ODD, orden por `IsAncestor` | Igual |
| U6 | `collect.go`: `Input`, `Collect`, prueba de extremo a extremo con `MapFS` y `History` falsos | Igual |
| U7 | `internal/cli/trace.go`: argumentos, fuentes, salida texto/JSON, ayuda; pruebas del adaptador | `go test ./internal/cli -run AxiomTrace -count=1`; trinquete de rechazos |
| U8 | `internal/cli/trace_flow_test.go`: merge/squash/rebase, bug, provisional, ODD tolerante y no integrado, solo lectura, determinismo, `t.Chdir`, multirepo | `go test ./internal/cli -run Trace -count=1` (incluye `TestSDDTrace*` sin cambios) |
| U9 | `cmd/axiom/main.go` (`case "trace"`, `runTrace`, línea de ayuda) y `trace_help_test.go` | `go test ./cmd/axiom -run TraceVerb -count=1`; `go vet ./cmd/axiom` |
| U10 | Manual (§8 y línea tras `:412`) y pin en `docs_pin_test.go` | `go test ./internal/workspace -run DocsPinManualDeInicio -count=1` |

U3 y U4 son las que más se acercan al límite; si superan claramente las 100 líneas, se separan clasificación y lectura sin partir un comportamiento.

## Migración y despliegue

No hace falta migración. El cambio es aditivo y de solo lectura: no escribe estado, no cambia `axiom.sdd-trace/v1` ni `isoevidence.History` y no afecta a ningún verbo existente.

## Preguntas abiertas

Ninguna bloquea. Decisiones tomadas que la spec deja abiertas o que la alinean, para que `sdd-tasks` y `sdd-verify` las vean:

- [x] Mención solo en prosa en un ODD que no parsea: advertencia `unclassified_mention` y sin incremento, para que el rol siga en el conjunto cerrado de prov:trace:9 (hueco 8 de `coverage-review.spec.md`).
- [x] Aparición solo en `ids.yaml` o en `## Identificadores asignados`: rol `created` (hueco 10).
- [x] Cambio SDD archivado fuera de la línea `--first-parent`: se informa con `found: false`, no se omite; solo un commit encontrado: `integrationCommit` usa el de código o, sin él, el de archivado (hueco 3).
- [x] Errores con `--json`: texto por stderr y salida 1 (hueco 7).
- [x] ID de entrada sin normalizar (hueco 5).
- [ ] Multirepo con rama de integración distinta por repositorio: hoy se usa una sola rama (`axiom.yaml` o `--integration-branch`) para todas las fuentes. Solo importa cuando `odd/tasks` y el planning home están en repositorios distintos (`specs_repository: openspec`) y sus ramas de integración difieren; entonces la referencia se comprueba en cada repositorio y el error nombra el repositorio que no la tiene (prov:trace:12 y prov:trace:14). No bloquea.
