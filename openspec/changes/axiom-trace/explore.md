# Exploración: axiom-trace (C-03-d)

Verbo nuevo `axiom trace <ID> [--cwd] [--integration-branch] [--json]`: dado un ID de requisito, responde qué incrementos (cambios SDD archivados, documentos ODD), tareas (`Cubre:`), bugs y commits de integración lo crearon o tocaron. Solo lectura, sin estado nuevo, anclado en `--first-parent`.

## 1. Estado actual (evidencia)

### `axiom sdd trace` (`internal/cli/sdd_trace.go`)
- Esquema `axiom.sdd-trace/v1` (`:25`); informe `traceReport` (`:287-297`): `codeCommit`, `archiveCommit`, `archiveFolder`, `discrepancy`, `chainSource`, `chain[]`.
- Está atado a `--change` (`:55-64`). `resolveTraceChangeRoot` (`:150-159`): carpeta activa o archivada más reciente vía `kickoff.ArchivedChangeRoot` (`internal/kickoff/archived.go:36`).
- Commit de integración: `isoevidence.DeriveTrace` (`internal/isoevidence/trace.go:78`) sobre el puerto `History` (`trace.go:16-33`), implementado por `gitHistory` (`sdd_trace.go:198-284`) con `git log --first-parent --reverse --no-renames --diff-filter=A` (`:222`). Código = primer commit de la línea que añade `verify-report.md` con `tasks*.md` en la carpeta (`trace.go:86-99`); archivado = primer commit que añade `kickoff.yaml` o la carpeta archivada (`trace.go:101-118`).
- Reutilizables tal cual, sin extraer, porque están en `package cli`: `planningHomePrefix` (`:129`), `traceIntegrationBranch` (`:163`), `prMergedEvidence` (`:181`), `gitHistory` (`:198`), `requirementKeys` (`:334`), `readTraceFile` (`:406`), `projectIDPattern` (`internal/cli/idpattern.go:17`), `encodeJSON`.
- Cadena existente: `traceChain` (`:342-402`) usa `sddstatus.ISOChangeRequirements` (`internal/sddstatus/iso.go:292`), `isoevidence.ParseCubre` (`cubre.go:27`) y `ParseMatrix`; solo ADDED y MODIFIED (`:368`).
- NO sirve para consultar por ID: itera un solo cambio y no conoce ODD.

### Registro en `cmd/axiom/main.go`
- Ayuda global: `axiomHelpText()` (`:86-177`); línea `sdd trace` en `:147`; `archive integrate` en `:138`.
- Conmutador de `main()` (`:241-415`): `case "archive"` en `:380`, `case "sdd"` en `:386`, `case "branch"` con `os.Exit(cli.RunBranch(...))` en `:294`. El `:2240` del backlog es hoy `runSimpleCommand` (`:2241`), el adaptador `func([]string, io.Writer) error` → código de salida; usarlo: `case "trace": os.Exit(runSimpleCommand(cli.RunTrace, os.Args[2:], os.Stdout, os.Stderr))`.
- Tests de ayuda: `cmd/axiom/archive_help_test.go:24,64,96` (patrón `strings.Contains(axiomHelpText(), "\n  archive integrate ")`), `branch_help_test.go:14`, `main_test.go:52-90` (`TestSDDVerbRouting`: sin argumentos el verbo falla con su mensaje, `--help` imprime `axiom <verbo>`). No hay goldens de la ayuda (`testdata/golden` no cita `sdd trace`).

### ODD y traza provisional→definitivo (C-03-b, fusionado)
- `internal/livespec/odddoc.go`: `ParseFeatureDoc(feature, data)` (`:191`) → `FeatureDoc{Kind, Capabilities[]{Name, Delta openspecdoc.Delta{Added,Modified,Removed,Renamed}}, Impact, SpecSection}` (`:53-82`); `Kind` bug por línea `Tipo: bug` (`:22,48,205-213`). Es estricto: devuelve `*DocError` con defectos (`:240-242`), así que un documento antiguo/mal formado falla → hay que tolerarlo.
- Traza: `internal/livespec/trace.go`. Esquema `axiom.requirement-ids/v1` (`:17`); `TraceEntry{Provisional, Definitive, Capability, Requirement, Previous[]}` (`:28-34`). SDD: `ids.yaml` en la carpeta del cambio, también archivada (`internal/livespec/change.go:22`, leída con `livespec.ReadTraceFile`, p. ej. `internal/cli/sdd_archive_integrate.go:129`; ausente = traza vacía, `trace.go:65-74`). ODD: sección `## Identificadores asignados` con un único bloque `yaml` (`trace.go:20,151-179`, `ExtractODDTrace` devuelve `found=false` si falta). Tras integrar, la delta y las líneas `Cubre:` del documento llevan ya los definitivos (`odd_ids.go:69-84`, `isoevidence.RewriteCubre`).
- Los ODD antiguos (anteriores a C-03-b) llevan el ID escrito a mano en el título y ninguna sección de traza: se encuentran por título (`pattern.InTitle`), no por traza.
- ODD en el repo: 100+ ficheros `odd/tasks/*.md`; solo 18 tienen `Spec viva`/`Tipo: bug`/`Impacto`/`Cubre:` (conteo propio): la mayoría no tocan requisitos.

### Puerto de IDs `internal/reqid`
- `Pattern` configurable (`reqid.go:34-195`): `Valid(id)` (`:75`), `InTitle(title)` (`:95`), `Scan(text)` (`:108`), `ParseProvisional` (`provisional.go:155`), `ScanProvisional` (`:107`). El patrón sale de `axiom.yaml` con `loc.Declarations.IDPattern()` → `projectIDPattern(command, loc)` (`idpattern.go:17`). Un ID de entrada válido = `pattern.Valid(id)`; opcionalmente un provisional vía `ParseProvisional`.

### Qué toca un cambio archivado (`openspec/changes/archive/<fecha>-<cambio>/`)
- Deltas `specs/*/spec.md` (ADDED/MODIFIED/REMOVED/RENAMED: `isoevidence/model.go:10-13`, `ISOChangeRequirements`).
- `Cubre:` en `tasks*.md` (`kickoff.RoleTaskFiles`, `ParseCubre`, usado en `sdd_trace.go:343-386`). Tras C-03-b se reescriben a definitivos.
- `ids.yaml` (asignaciones; los `Previous` son alias históricos).
- Filas de la matriz en `verify-report*.md` (`ParseMatrix`).
- Bug SDD: no hay `Tipo: bug`; se distingue por `isoevidence.ParseBugDeclaration(proposal.md)` (`bug.go:50`): `## Informe de bug` → `existing_requirement`; `## Impacto en la spec` → `no_impact`; `none` = no es bug. Limitación: un bug cuyo requisito estaba mal especificado va con delta y sin esas secciones, indistinguible de un incremento.

### Commit de integración
- SDD: reutilizar `DeriveTrace` (código y archivado). Para el verbo por ID conviene una llamada directa a `FirstParentAdded` sobre la carpeta archivada (`kickoff.yaml`) para evitar la ambigüedad de nombres de cambio repetidos (`archivedFolder` toma la más reciente, `trace.go:143-162`).
- ODD: `archive integrate --odd` no crea commit (lo hace el commit de trabajo de la feature, `CLAUDE.md` ODD paso 6-7). Ancla viable: `FirstParentAdded(ref, "odd/tasks/<f>.md")` = primer commit de la línea principal que incorpora el documento (la fusión de la PR; con squash/rebase también). Matiz: es el commit que trae el documento, no necesariamente el que añadió `## Spec viva`. Para precisar se puede añadir al puerto una consulta de pickaxe (`git log --first-parent -S"<definitivo>" -- openspec/specs/<cap>/spec.md`) que da el commit que introdujo el ID en la spec viva, válido para ambos carriles.

### Pruebas y manual
- Fixtures en `package cli`: `traceGit`/`traceRepo` (`sdd_trace_test.go:20-44`), `traceIntegrate` (merge/squash/rebase, `sdd_trace_flow_test.go:20-39`), `traceDiverge` (`:43`), `newTraceFlowWith` (`:75-125`, escribe `axiom.yaml` ISO con `writeSDDAxiomYAML`, `sealTestChange`, kickoff, archivado con `git mv`), `runTrace` (`:156`), `writeCLIFixture`. `observeGit` (`sdd_trace_test.go:48`) comprueba que solo se ejecutan lecturas de git. Nombres de prueba con prefijo y ejecución focalizada con `-run`.
- Manual `docs/manual-de-inicio.md`: bullet de `sdd trace` en `:412` (sección «Modo ISO y Perfil de Bug»); §8 «La Spec Viva es OpenSpec» `:578`; ODD en `:653-665`. Recomendado: subsección nueva en §8 («Trazabilidad de un requisito: `axiom trace`») y una línea en `:412`. Pines que citan `sdd trace`: `internal/assets/sdd_iso_pins_test.go`, `internal/workspace/example_config_test.go`, `internal/cli/sdd_idpattern_test.go`, `openspec/specs/requirement-identifiers/spec.md`.

## 2. Enfoques

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| A. Fichero único `internal/cli/trace.go`, reutiliza helpers de `package cli` y parsers de `livespec`/`isoevidence` | Mínimo; sin refactor; `sdd_trace.go` intacto | Lógica de dominio (escaneo, rol por requisito) dentro del adaptador; difícil de reutilizar para sección 07 | M-bajo |
| B. Paquete de dominio `internal/reqtrace` (Collect con puerto `isoevidence.History` + lectores de ficheros) + adaptador fino `cli/trace.go` | Sigue la arquitectura hexagonal del repo (como `isoevidence.DeriveTrace` y `livespec`); probable consumo por C-07; punto de extensión limpio para el mapa de C-03-e (`aliases(id)`) | Algo más de código y un paquete nuevo | M |
| C. Ampliar `axiom sdd trace` con `--requirement` | Cero verbos nuevos | No cumple «verbo `axiom trace`»; mezcla ámbitos; rompe contrato `sdd-trace/v1` | S, descartado |

## 3. Recomendación

Enfoque B (dominio en `internal/reqtrace`, adaptador `internal/cli/trace.go`, sin tocar `sdd_trace.go` salvo exportar nada: todo vive en `package cli`).

Diseño propuesto:
1. Entrada: `ID` posicional (o `--id`), validado con `pattern.Valid` (o provisional resuelto vía trazas). Error con continuación si no es válido. Punto de extensión `resolveAliases(id) []string`: devuelve el ID, sus `Previous` de las trazas y, tras C-03-e, el mapa `REQ-n.m`→nuevo (hoy sin implementar).
2. Escaneo (árbol de trabajo): `openspec/changes/archive/*/` y `odd/tasks/*.md` (más cambios activos, marcados `active`, sin commit). Un incremento entra si: la delta cita el ID en un título ADDED/MODIFIED/REMOVED/RENAMED; `ids.yaml`/traza ODD lo asigna; o una línea `Cubre:` lo cita. Rol: `created | modified | removed | renamed | covered`.
3. Bug: ODD por `Kind==KindBug`; SDD por `ParseBugDeclaration.Kind != none`; campo `type: "increment"|"bug"`.
4. Commits: `FirstParentAdded` por carpeta archivada (SDD: código vía `DeriveTrace`, archivado) y por `odd/tasks/<f>.md` (ODD). Ausente → `found:false`, nunca error (tolera ODD sin traza y cambios no integrados). Orden de salida por posición en la línea `--first-parent`.
5. JSON `axiom.trace/v1`: `{schema, id, resolvedIds[], integrationBranch, increments:[{lane, name, path, type, role, tasks:[{file,line}], integrationCommit{commit,found}, archiveCommit?, assignment?{provisional,definitive}}], warnings[]}`; texto legible sin `--json`.
6. Tolerancia: `ParseFeatureDoc` fallido → advertencia en `warnings`, no error; fallback a búsqueda textual del ID en el documento.
7. Registro: `case "trace"` en `main.go` (junto a `:380`) con `runSimpleCommand`; línea en `axiomHelpText` (cerca de `:147`); `--help` propio.
8. Pruebas: reutilizar `traceRepo`/`traceIntegrate`; casos: SDD archivado + ODD que tocan el mismo ID con commits; bug SDD con `Cubre:`; ODD antiguo sin traza; ID inválido; solo lectura (`observeGit`); merge/squash/rebase; enrutado y ayuda en `cmd/axiom`.
9. Docs: manual (§8 y `:412`), delta de la capacidad `requirement-identifiers` (REQ nuevo) y guías de agente si citan `sdd trace`.

## 4. Riesgos
- Topología multirepo: `odd/tasks` vive en la raíz del workspace y las specs en otro repo; el commit de un ODD y el de un SDD pueden pertenecer a repositorios git distintos. Decidir en el design (propuesta: soporte pleno en monorepo-embedded, y en multirepo resolver cada ruta con su propia raíz git).
- ODD con squash: el commit de integración es el que trae el documento, no el de `## Spec viva`; documentarlo o usar pickaxe.
- Bug SDD con delta indistinguible de incremento (sin `Tipo`).
- Coste del escaneo de todos los archivados (cientos de carpetas): leer solo ficheros pequeños y consultar git solo para los incrementos coincidentes.
- Nombres de cambio repetidos en `archive/` (carpetas `<fecha>-<cambio>`): anclar por carpeta, no por nombre.
- Rama de integración ausente en el clon: mismo error con remedio `--integration-branch` que `sdd trace`.
- IDs antiguos `REQ-n.m` dependen del mapa de C-03-e (no existe): fuera de alcance, solo punto de extensión.
- Prueba intermitente conocida del consent snapshot (memoria): focalizar con `-run`, no batería completa.
