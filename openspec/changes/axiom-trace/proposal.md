# Propuesta: `axiom trace <ID>`, trazabilidad de un requisito de punta a punta (C-03-d)

## Intención

Hoy no hay forma de preguntar a Axiom «qué tocó este requisito». `axiom sdd trace` (`internal/cli/sdd_trace.go`, esquema `axiom.sdd-trace/v1`) está atado a un cambio (`--change`) y no conoce el carril ODD, así que para reconstruir la historia de un ID hay que recorrer a mano `openspec/changes/archive/`, los documentos `odd/tasks/*.md`, las líneas `Cubre:` y el historial de git.

El tema 3 del origen del plan maestro pide «que los incrementos y los bugs se relacionen con ese ID para que todo quede trazado». C-03-a (patrón de ID configurable) y C-03-b (IDs provisionales y traza `axiom.requirement-ids/v1`) ya están fusionados: ya existen los datos y falta la consulta. Además, la sección 07 (integraciones con terceros) necesita una fuente de trazabilidad por requisito que pueda consumir sin repetir el escaneo.

Éxito: `axiom trace <ID> --json` devuelve, para un requisito tocado por un cambio SDD archivado y por una feature ODD, ambos incrementos con su commit de integración en la línea `--first-parent`, y un bug SDD con `Cubre:` a ese ID aparece como bug.

## Alcance

### Dentro del alcance

- Paquete de dominio nuevo `internal/reqtrace`: consulta por ID que recorre los cambios SDD archivados y los documentos ODD, clasifica cada aparición y ordena el resultado por la línea `--first-parent`. Depende solo de puertos: historia git (reutiliza el contrato `isoevidence.History`, con `FirstParentAdded`) y lectura de ficheros.
- Adaptador fino `internal/cli/trace.go` (`RunTrace`): argumentos, resolución del patrón del proyecto (`projectIDPattern`), rama de integración (misma lógica y remedio que `traceIntegrationBranch`), implementación git de solo lectura (reutiliza `gitHistory`) y salida en texto o JSON.
- Verbo `axiom trace <ID> [--json] [--integration-branch <rama>] [--cwd <dir>]`, registrado en `cmd/axiom/main.go` con `runSimpleCommand`, con línea en `axiomHelpText()` y `--help` propio.
- Fuentes leídas:
  - Cambios archivados `openspec/changes/archive/<fecha>-<cambio>/`: deltas `specs/*/spec.md` (ADDED, MODIFIED, REMOVED, RENAMED), líneas `Cubre:` de `tasks*.md` e `ids.yaml`.
  - Documentos `odd/tasks/*.md`: `## Spec viva`, `Tipo: bug`, `## Identificadores asignados` y líneas `Cubre:`.
- Rol de cada aparición: `created | modified | removed | renamed | covered`. Tipo del incremento: `increment | bug`. En SDD el bug se detecta con `isoevidence.ParseBugDeclaration` sobre `proposal.md`; en ODD, con la línea `Tipo: bug`.
- Resolución de IDs provisionales por la traza `axiom.requirement-ids/v1`: consultar por el definitivo o por el provisional (`prov:…`) devuelve el mismo resultado, con la asignación visible en el incremento.
- Punto de extensión explícito para alias (el mapa `REQ-n.m` → ID nuevo de C-03-e), sin implementarlo: hoy solo devuelve el propio ID y los alias de las trazas.
- Contrato JSON versionado `axiom.trace/v1` (ID consultado, IDs resueltos, rama de integración, incrementos con carril, ruta, tipo, rol, tareas, commit de integración con `found`, asignación y advertencias). Forma exacta en el design.
- Tolerancia:
  - Commit de integración no encontrado: `found: false`, nunca error.
  - Documento ODD que no parsea (`livespec.ParseFeatureDoc` es estricto) o sin traza: advertencia en `warnings`, nunca error.
  - ID que no cumple el patrón del proyecto ni es un provisional válido: error tipado con continuación y código de salida distinto de 0.
  - ID válido sin apariciones: resultado vacío y código de salida 0.
- Topología: soporte pleno en `monorepo-embedded`; en `multirepo`, cada ruta se ancla con la raíz git del repositorio que la contiene (detalle en el design).
- Manual `docs/manual-de-inicio.md`: subsección nueva en §8 («Trazabilidad de un requisito: `axiom trace`») y referencia junto al bullet de `sdd trace`.
- Pruebas: dominio con dobles de los puertos y adaptador con los fixtures git existentes (`traceRepo`, `traceIntegrate`, `observeGit`), incluidos merge, squash y rebase; enrutado y ayuda en `cmd/axiom`.

### Fuera del alcance

- Envío del ID o de la traza a un tercero (sección 07). Este verbo es la fuente que esa sección podrá consumir.
- El mapa `REQ-n.m` → ID nuevo de C-03-e: solo queda el punto de extensión.
- Cambios activos no archivados y documentos ODD no integrados en la rama de integración: solo cuentan los archivados e integrados.
- Cualquier cambio en `axiom sdd trace`, `sdd_trace.go` o el esquema `axiom.sdd-trace/v1`.
- Estado nuevo, caché o índice persistente: la consulta recalcula en cada ejecución.
- Distinguir el bug SDD cuyo requisito estaba mal especificado (va con delta y sin `## Informe de bug` ni `## Impacto en la spec`): es indistinguible de un incremento y así se informa.

## Capacidades

### Capacidades nuevas

- `requirement-trace`: consulta de solo lectura `axiom trace <ID>` que, dado un ID de requisito (definitivo o provisional), devuelve los incrementos SDD archivados y ODD integrados que lo crearon, modificaron, retiraron, renombraron o cubrieron, su tipo (incremento o bug), sus tareas y su commit de integración `--first-parent`, con el contrato JSON `axiom.trace/v1`, su tolerancia a datos incompletos, su registro como verbo y su documentación.

  Serie nueva con el patrón por defecto del proyecto (`REQ-<n>.<m>`, `axiom.yaml`): la spec viva ocupa hasta la serie 47, así que los ADDED llevan provisionales con etiqueta de serie nueva compartida, `prov:trace:1`, `prov:trace:2`, … (REQ-47.1 y REQ-47.2 de `requirement-identifiers`; la etiqueta cumple `^[a-z][a-z0-9]*$`). El número definitivo se asigna al archivar; no se fija a mano.

### Capacidades modificadas

- Ninguna. `sdd-iso-evidence` (REQ-29.x de `axiom sdd trace`) y `requirement-identifiers` (REQ-47.6, traza legible) se consumen sin cambiar sus requisitos.

## Enfoque

Enfoque B de la exploración (arquitectura hexagonal, como `isoevidence.DeriveTrace` y `livespec`):

1. **Entrada.** El adaptador lee el patrón del proyecto y valida el ID con `reqid.Pattern.Valid`; si no es válido, intenta `ParseProvisional`. Ninguno de los dos: error tipado.
2. **Resolución de alias.** `reqtrace` reúne el conjunto de IDs equivalentes: el propio, su pareja provisional/definitivo y los `Previous` de cada `TraceEntry` (trazas de `ids.yaml` y de `## Identificadores asignados`). Una interfaz de alias inyectable deja el hueco para el mapa de C-03-e.
3. **Escaneo.** Se recorren las carpetas archivadas y los documentos ODD del árbol de trabajo de la rama de integración. Solo se leen ficheros pequeños (deltas, `tasks*.md`, `ids.yaml`, `proposal.md`, el documento ODD), y git solo se consulta para los incrementos que coinciden.
4. **Clasificación.** Rol por sección de la delta o por línea `Cubre:`; tipo por la declaración de bug. Un documento ODD que no parsea cae a una búsqueda textual del ID y deja una advertencia.
5. **Anclaje.** SDD: commit de código y de archivado como `DeriveTrace`, anclado por la carpeta archivada (`kickoff.yaml`) para evitar la ambigüedad de nombres de cambio repetidos. ODD: primer commit de la línea `--first-parent` que añade `odd/tasks/<feature>.md`. El design decide si se añade una consulta de pickaxe sobre `openspec/specs/<cap>/spec.md` para precisar el commit que introdujo el ID.
6. **Salida.** Orden por posición en la línea `--first-parent` (los no encontrados al final, en orden estable); texto legible por defecto y `axiom.trace/v1` con `--json`.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/reqtrace/` | Nuevo | Dominio: consulta, resolución de alias, clasificación, ordenación y puertos. |
| `internal/cli/trace.go` (+ `trace_test.go`) | Nuevo | Adaptador `RunTrace`: argumentos, patrón, rama, git de solo lectura y render texto/JSON. |
| `internal/cli/sdd_trace.go` | Sin cambios | Se reutilizan `gitHistory`, `traceIntegrationBranch`, `planningHomePrefix` y `readTraceFile` sin tocar su contrato. |
| `internal/isoevidence` | Consumido | `History`, `DeriveTrace`, `ParseCubre`, `ParseBugDeclaration`. Sin cambios previstos. |
| `internal/livespec` | Consumido | `ParseFeatureDoc`, `ReadTraceFile`, `ExtractODDTrace`. Sin cambios previstos. |
| `internal/reqid`, `internal/openspecdoc`, `internal/kickoff` | Consumido | Patrón, provisionales, parseo de deltas, ficheros de tareas por rol. |
| `cmd/axiom/main.go` (+ tests de ayuda y enrutado) | Modificado | `case "trace"` y línea en `axiomHelpText()`. |
| `docs/manual-de-inicio.md` | Modificado | Subsección en §8 y referencia junto a `sdd trace`. |
| `openspec/changes/axiom-trace/specs/requirement-trace/spec.md` | Nuevo | Capacidad nueva con provisionales `prov:trace:<n>`. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| ODD con squash o rebase: el commit que trae el documento no es necesariamente el que añadió `## Spec viva`. | Media | Documentarlo en el contrato; el design valora el pickaxe sobre la spec viva. |
| Documentos ODD antiguos sin traza ni formato estricto (100+ ficheros). | Alta | Búsqueda textual de respaldo y advertencias, nunca error; prueba con un documento antiguo. |
| Bug SDD con delta indistinguible de un incremento. | Media | Se clasifica como incremento y se documenta la limitación. |
| Coste del escaneo de cientos de carpetas archivadas. | Baja | Solo ficheros pequeños; git solo para coincidencias. |
| Nombres de cambio repetidos en `archive/`. | Baja | Anclar por carpeta, no por nombre. |
| Rama de integración ausente en el clon. | Media | Mismo error y remedio `--integration-branch` que `sdd trace`; nunca `fetch`. |
| Multirepo: `odd/tasks` y las specs pueden vivir en repos git distintos. | Media | Anclar cada ruta con la raíz git que la contiene; detalle y pruebas en el design. |
| Dependencia de C-03-e para IDs `REQ-n.m` antiguos. | Baja | Solo punto de extensión; el resultado hoy es correcto para los IDs vigentes. |
| Prueba intermitente conocida del consent snapshot. | Baja | Pruebas focalizadas con `-run`; la batería completa, en CI. |

## Plan de reversión

El cambio es aditivo y de solo lectura: no escribe estado, no migra datos y no modifica contratos existentes.

1. Revertir el commit de fusión de la PR (`git revert -m 1 <merge>`), o los commits de trabajo, elimina `internal/reqtrace/`, `internal/cli/trace.go`, el `case "trace"`, la línea de ayuda y la subsección del manual.
2. Si la capacidad `requirement-trace` ya se archivó en `openspec/specs/`, la reversión incluye esa carpeta. Como ningún otro requisito ni código cita sus IDs, retirarla no deja referencias colgantes; el cambio archivado conserva su `ids.yaml` como historia.
3. `axiom sdd trace` y `axiom.sdd-trace/v1` siguen funcionando igual porque no se tocan. Ningún otro paquete depende de `reqtrace` tras este cambio.

## Dependencias

- C-03-a (puerto `internal/reqid`) y C-03-b (provisionales y traza `axiom.requirement-ids/v1`): fusionados.
- C-03-e (mapa `REQ-n.m` → nuevo): dependencia blanda, solo para resolver IDs antiguos; fuera de alcance.

## Criterios de éxito

- [ ] `axiom trace <ID> --json` devuelve, para un requisito tocado por un cambio SDD archivado y por una feature ODD, ambos incrementos con su commit de integración `--first-parent` (merge, squash y rebase).
- [ ] Un bug SDD con `Cubre:` a ese ID aparece con `type: "bug"` y rol `covered`.
- [ ] Consultar por el provisional y por el definitivo de una asignación devuelve el mismo resultado.
- [ ] Un documento ODD que no parsea o sin traza produce una advertencia, nunca un error; un commit no encontrado da `found: false`.
- [ ] Un ID fuera del patrón da un error tipado y código de salida distinto de 0; un ID válido sin apariciones da resultado vacío y código 0.
- [ ] La ejecución solo lanza lecturas de git (`observeGit`) y no escribe ningún fichero.
- [ ] `axiom --help` y `axiom trace --help` muestran el verbo; el manual lo documenta.
- [ ] `axiom sdd trace` y `axiom.sdd-trace/v1` no cambian (sus pruebas pasan sin modificación).
- [ ] `go test` focalizado de `internal/reqtrace`, `internal/cli` (con `-run`) y `cmd/axiom` en verde; `go vet` de los paquetes tocados limpio.
