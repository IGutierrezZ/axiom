# Diseño: Evidencias ISO y mejoras del SDD (carril-sdd-versatil)

> Fase `design` · rama `inc/carril-sdd-versatil` · 2026-11-10 · kickoff `checkpointed`, rol único `fullstack` (`blocking`).
> Entrada: `proposal.md` y las 13 deltas de `specs/` (compuerta `spec` aprobada). Decisiones de producto en Engram `architecture/sdd-carril-versatil/*`.
> Contexto técnico mantenido: el status no proyecta `technicalContext.maintained` (este repositorio no declara `technical_context:`), así que no hay `context-impact.md`.
> Este documento usa el formato de design proporcional que él mismo define (§3.4): secciones mínimas, tabla de disparadores y, al haber disparadores marcados, el nivel completo.

## Enfoque

El cambio **extiende** el SDD sin retirar nada. Se organiza en tres capas:

1. **Configuración** (`internal/workspace`): un bloque nuevo de primer nivel `sdd:` en `axiom.yaml`, decodificado con lista cerrada de claves al estilo de `workflow:`, y un punto único de resolución del destino del archivado (frontera D5).
2. **Evidencia** (paquete nuevo `internal/sddevidence`): analizadores puros de Markdown (`story.md`, deltas, `Cubre:`, matriz de `verify-report.md`, `design.md`, plan de `tasks.<rol>.md`), la consulta de un requisito existente en la baseline (REQ-28.25), la derivación del commit de integración (REQ-28.6) y un evaluador que devuelve **hallazgos tipados** con severidad `block` o `note`.
3. **Proyección y guardas** (`internal/sddstatus`, `internal/branch`, `internal/kickoff`, `internal/cli`): el status aplica los hallazgos como motivos `blocked(<token>)` o avisos `note(<token>)`, ajusta la ruta recomendada y proyecta el bloque `sdd`; el motor de ramas consulta un único predicado para saber si existe rama de archivado; la composición del archivado rechaza en ISO los requisitos sin escenarios.

Las correcciones independientes (piezas 1 y 3) no dependen de ninguna de estas capas. La prosa (skills, doctrina y orquestador) va al final; la del orquestador espera a la cadena de prosa (§9).

## Roles

| Rol | Política | Alcance |
|---|---|---|
| `fullstack` | `blocking` | Todo el cambio: Go, pruebas, skills, doctrina, prosa del orquestador, goldens, manual y ejemplo. |

```yaml
roles:
  - role: fullstack
    gate_policy: blocking
```

Con un solo rol basta este `design.md`; no hay `design.<rol>.md` ni plan propio por rol.

## Componentes

| Componente | Paquete o fichero | Nuevo o modificado |
|---|---|---|
| Bloque `sdd:` y su resolución | `internal/workspace/sdd.go` (nuevo), `declarations.go`, `loader.go`, `types.go` | Nuevo y modificado |
| Frontera del destino del archivado (D5) | `internal/workspace/archive_target.go` (nuevo) | Nuevo |
| Analizadores y evaluador de evidencia | `internal/sddevidence/` (nuevo) | Nuevo |
| Decorador de evidencia del status | `internal/sddstatus/evidence.go` (nuevo), `status.go`, `status_v2.go`, `workspace_declarations.go`, `governance.go`, `governance_handoff.go` | Nuevo y modificado |
| Partición `blocking` común | `internal/multirole/types.go`, `barrier.go` | Modificado |
| Predicado «existe rama de archivado» | `internal/branch/` (`planner.go`, `planner_archive.go`, `verdict.go`), `internal/branch/engine/` (`engine.go`, `verdict.go`, `publication.go`, `archive_plan.go`) | Modificado |
| Compuerta `integration` no aplicable | `internal/kickoff/machine.go`, `internal/cli/sdd_gate.go`, `sdd_gate_handoff.go` | Modificado |
| `kickoff seal` con carpeta | `internal/cli/sdd_kickoff.go`, `sdd_governance.go` | Modificado |
| Regla de escenarios en la composición | `internal/cli/sdd_archive_compose.go`, `internal/specbaseline/plan.go` | Modificado |
| Verbo `axiom sdd trace` | `internal/cli/sdd_trace.go` (nuevo), `cmd/axiom/main.go` | Nuevo y modificado |
| Prosa | `internal/assets/skills/**`, `internal/components/agentguidance/routing.go` | Modificado |

## Riesgos

| Riesgo | Prob. | Mitigación |
|---|---|---|
| Ciclo de importación en pruebas: `internal/specbaseline` tiene pruebas internas que importan `sddstatus`; si `sddstatus` importara `specbaseline` (directa o transitivamente) esas pruebas dejarían de compilar. | Alta si no se respeta | `sddevidence` solo importa `workspace` y `gitcommon`, nunca `specbaseline` (decisión D2). Una prueba de arquitectura lo fija. |
| Las notas nuevas fuera de ISO (design sin marcas) cambian el status de cambios activos y rompen aserciones exactas de `notes` en pruebas antiguas. | Alta | Variable de prueba `evidenceEvaluator` (patrón de `branchEngine`): el `TestMain` de `sddstatus` la desactiva para las pruebas anteriores; las nuevas la activan. |
| `axiom sdd status` ejecuta `git diff` en la fase verify (aviso de disparadores no marcados) y lee la baseline entera en ISO. | Media | Solo en ISO o en verify; `gitcommon.RunWithTimeout` acotado; sin git, sin aviso y sin bloqueo. |
| `sdd.archive` se lee en vivo de `axiom.yaml`: cambiarlo con cambios a medio archivar mezcla reglas. | Media | La skill y el manual lo advierten; la opción es por proyecto (Q11). Pregunta abierta P5. |
| `axiom branch done` con `same_pr` no tiene la capa 3 (`pr_merged` en `gates.yaml`): con squash y git < 2.38 no confirma la fusión. | Media | Queda `--force` y la capa 2; pregunta abierta P6. |
| Ratchet de negativas (`.refusal-ratchet-baseline.txt`) y de código muerto. | Media | Cada negativa nueva lleva su anotación `refusal:by-design`; la interfaz D5 tiene un único implementador usado. |
| Cadena de prosa del orquestador sin cerrar. | Media | La unidad U15 espera; el resto no toca esa prosa (§9). |

## Disparadores

| Disparador | Marca | Motivo |
|---|---|---|
| `public-contract` | sí | Bloque `sdd:` de `axiom.yaml`, claves nuevas del status v2, verbo `axiom sdd trace`, tokens tipados nuevos. |
| `new-component` | sí | Paquete `internal/sddevidence` y frontera `workspace.ArchiveTargetResolver`. |
| `data-model` | no | Sin estado persistido nuevo ni migración: `kickoff.yaml`, `gates.yaml` e `integration.yaml` no cambian de esquema. |
| `security-performance` | sí | El status pasa a ejecutar `git diff` en verify y a leer la baseline en ISO; el verbo `trace` ejecuta `git log`. |
| `maintained-context` | no | El proyecto no declara `technical_context.maintained`. |

Revisor esperado en la compuerta `design`: el arquitecto (REQ-28.14).

## Decisiones

### D1. Bloque `sdd:` (Q3b): claves y valores

**Elección**:

```yaml
sdd:
  iso: true               # booleano YAML; por defecto false
  archive: same_pr        # branch | same_pr; por defecto branch
  design_triggers:        # lista de identificadores; amplía los base
    - accessibility
```

- `iso`: escalar con etiqueta `!!bool` y valor `true` o `false`. Cualquier otro valor (`"true"` entrecomillado, `yes`, `1`) es error.
- `archive`: `branch` (rama `-archive` y PR documental aparte, el comportamiento actual) o `same_pr` (REQ-28.22). `same_pr` con topología `multirepo` es error que nombra opción y topología.
- `design_triggers`: secuencia de escalares no vacíos, únicos, con el patrón de dominio `^[a-z0-9]+(-[a-z0-9]+)*$` y distintos de los base (`public-contract`, `new-component`, `data-model`, `security-performance`, `maintained-context`).
- `sdd:` nulo, escalar o lista: error «el bloque 'sdd' debe ser un mapeo con las claves opcionales 'iso', 'archive' y 'design_triggers'».

**Alternativas**: `iso` como enumerado (`none|pa2.2`) para dejar sitio a Q6; `archive` booleano (`same_pr: true`); identificadores de disparador en castellano.
**Motivo**: un booleano es la forma mínima y Q6 (modelo y nivel) se añadiría como claves propias sin cambiar `iso`; un enumerado en `archive` deja nombrado el valor por defecto y admite un tercer flujo sin romper; los identificadores en inglés y kebab-case siguen la convención de las claves y valores de `axiom.yaml` (`in_place`, `odd`).

### D2. Paquete `internal/sddevidence` sin dependencia de `specbaseline`

**Elección**: un paquete nuevo con analizadores puros y un evaluador. Importa solo `internal/workspace` (perfiles, `RequirementProfile.Compile`, `Location`) y `internal/gitcommon` (único punto de ejecución de git). Incluye su propio escáner mínimo de encabezados ATX fuera de vallas.
**Alternativas**: (a) meter los validadores en `sddstatus` (ya muy grande, y `specbaseline` no podría reutilizarlos); (b) que `sddevidence` reutilice el parser de `specbaseline` y mover a un paquete externo las dos pruebas de paridad de `specbaseline` que importan `sddstatus` (`compose_test.go:650`, `compose_added_test.go:613`), lo que exige exportar sus ayudantes internos.
**Motivo**: (b) rompería la compilación de las pruebas de `specbaseline` por el ciclo `specbaseline (test) → sddstatus → sddevidence → specbaseline`. El escáner propio repite unas 40 líneas de detección de vallas, una duplicación que el propio `specbaseline/doc.go` ya documenta respecto del motor nativo. `specbaseline` y `cli` sí pueden importar `sddevidence`.

### D3. Hallazgos tipados y su proyección

**Elección**: `sddevidence.Evaluate` devuelve `[]Finding{Token, Severity, Phase, Subject, Detail, Continuation}`. El status los proyecta con el formato existente de los motivos tipados (`blocked(wrong_branch): …`):

- `Severity == block` → `blockedReasons` con `blocked(<token>): <detalle>; <continuación>`.
- `Severity == note` → `notes` con `note(<token>): <detalle>` (precedente: `note(branch_unchecked)`).

Tokens y ruta (la ruta es la fase dueña de la remediación; `resolve-blockers` cuando la corrige una persona fuera del flujo):

| Token | REQ | ISO | Fuera de ISO | Desde | Ruta |
|---|---|---|---|---|---|
| `sdd_declaration_invalid` | 28.1, 28.23, 28.24 | block | block | siempre | `resolve-blockers` |
| `iso_store_unsupported` | 28.1 | block | — | siempre | `resolve-blockers` |
| `story_profile_invalid` | 28.10, 28.11 | block | block | `story.md` existe | `propose` |
| `iso_identity_required` | 28.2 | block | — | delta escrita | `resolve-blockers` |
| `iso_requirement_id_missing` | 28.3, 28.25 | block | — | delta escrita | `spec` |
| `iso_scenarios_missing` | 28.21 | block | — | delta escrita | `spec` |
| `design_incomplete` | 28.12, 28.17 | block | note | `design.md` existe | `design` |
| `design_role_unknown` | 28.15, 28.17 | block | note | `design.<rol>.md` existe | `design` |
| `role_plan_missing` | 28.16, 28.17 | block | note | `tasks.<rol>.md` existe | `tasks` |
| `iso_coverage_missing` | 28.3 | block | — | tareas escritas | `tasks` |
| `iso_coverage_unknown_requirement` | 28.3, 28.11 | block | — | tareas escritas | `tasks` |
| `iso_matrix_invalid` | 28.4 | block | — | fase archive | `verify` |
| `iso_matrix_failed` | 28.5 | block | — | fase archive | `resolve-blockers` |
| `design_triggers_marked` | 28.14 | note | note | `continuous` con marcas | — |
| `design_trigger_unmarked` | 28.13 | note | note | fase verify | — |
| `verify_nonblocking_roles` | 21.15 | note | note | verify listo con roles no bloqueantes | — |

Reglas de composición:

1. `blocked(wrong_branch)` sigue el primero y, con él, la ruta sigue siendo `resolve-blockers` (REQ-25.42).
2. Se proyectan **todos** los bloqueos concurrentes (REQ-28.27), en el orden de la tabla.
3. La ruta es `resolve-blockers` si algún bloqueo tiene esa ruta; si no, la fase más temprana entre las de los bloqueos (`propose < spec < design < tasks < verify`). Esta ruta prevalece sobre `await-gate`: no se presenta una compuerta sobre un artefacto que no supera su validación.
4. Un bloqueo de las fases `archive` (matriz) solo bloquea `dependencies.archive` y entra en `blockedReasons` cuando la ruta sería `archive`, con el mismo patrón que `blockArchiveOnDeclarations` (una matriz pendiente no bloquea la planificación ni apply).
5. Las notas nunca cambian la ruta ni las dependencias.

**Alternativas**: un campo estructurado nuevo (`findings: [...]`) en el status v2; enrutar todo a `resolve-blockers`.
**Motivo**: los consumidores ya entienden `blockedReasons` como compuerta y `notes` como información (contrato de `blockerReasons`); un campo nuevo duplicaría el canal. Enrutar a la fase dueña sigue el precedente de una compuerta rechazada (`governanceNextRecommended`) y deja al orquestador relanzar la fase que corrige el artefacto.

### D4. Fallo cerrado de la declaración `sdd:`

**Elección**: un `sdd:` inválido produce `blocked(sdd_declaration_invalid)` en cualquier fase, con la continuación `axiom workspace validate --path <maestra>`, y el status proyecta `sdd.source: invalid`. Además entra en `invalidDeclarationErrors` (bloquea el archivado como las demás declaraciones).
**Alternativas**: bloquear solo el archivado (como `baseline:` y `technical_context:`, REQ-24.18); tratar el proyecto como ISO.
**Motivo**: REQ-28.1 exige no degradar a «no ISO»; sin saber si el proyecto es ISO no se puede decidir si la identidad, la cobertura o el design bloquean, así que no se avanza. Tratarlo como ISO convertiría una errata en decenas de bloqueos. Es una asimetría deliberada respecto de REQ-24.18 (pregunta abierta P3).

### D5. ISO solo con almacén de ficheros

**Elección**: un proyecto ISO cuyo almacén declarado es `engram` o `none` recibe `blocked(iso_store_unsupported)` con la continuación «declara `artifact_store: openspec` o `hybrid`».
**Alternativas**: validar el texto de las observaciones de Engram.
**Motivo**: la trazabilidad de REQ-28.6 se deriva de la ruta archivada en la rama principal; sin ficheros versionados no hay evidencia auditable. Pregunta abierta P4.

### D6. Perfil del cambio: `story.md` con front matter

**Elección**: el perfil se declara en el propio `story.md`, con front matter YAML de lista cerrada:

```markdown
---
profile: bug            # story | bug; por defecto story
external_ref: JIRA-123  # opcional, texto libre no vacío
---
```

- Sin `story.md`, el perfil es el actual (`proposal`). Con `proposal.md` y `story.md` a la vez, prevalece `proposal.md` y se emite `note(story_ignored)`.
- Front matter ausente: `story`. Clave desconocida, `profile` fuera de `story|bug` o YAML ilegible: `blocked(story_profile_invalid)`.
- `resolveArtifactPaths` toma `proposal.md` y, si falta, `story.md` como localizador de `artifactPaths.proposal`: la clave `artifacts.proposal` no cambia de nombre (REQ-28.26) y el topic de Engram en `hybrid` sigue siendo `sdd/<cambio>/proposal`.
- `story.md` se añade a los marcadores de cambio (`openSpecChangeContainer`, `explorationOnlyChangeDir`, `status.go:442` y `:491`).
- Bug sin delta: `artifacts.specs` sigue `missing` (es lo que hay en disco), pero `dependencies.specs` es `all_done` y `coreReady` no exige delta. La compuerta `spec` no se abre (no hay digest de specs).

**Alternativas**: una bandera `--profile` en `kickoff seal` guardada en `kickoff.yaml`; inferir «bug sin delta» de la ausencia de delta.
**Motivo**: `kickoff.yaml` es de escritura única y un binario anterior rechazaría la clave nueva; el perfil se conoce al escribir la historia, no al sellar. Inferirlo convertiría una delta olvidada en un bug sin delta silencioso, inaceptable para una auditoría.

### D7. `Cubre:` y cobertura

**Elección**: una línea cuyo texto, sin espacios iniciales ni marcador de lista, empieza por `Cubre:` y pertenece a una tarea (las líneas entre un `- [ ]`/`- [x]` y el siguiente elemento de nivel igual o superior o encabezado):

```markdown
- [ ] 1.2 Filtrar la partición por `gate_policy`
  Cubre: REQ-21.15, REQ-2.1
```

- Identificadores separados por comas con el patrón `[A-Za-z][A-Za-z0-9]*-[0-9]+(\.[0-9]+)*` (`REQ-28.1`, `RF-120`, `NFR-001`).
- Una línea `Cubre:` fuera de una tarea no cuenta y produce `note(iso_coverage_orphan)` en ISO.
- Requisitos de la delta: ADDED (ID provisional con baseline declarada; el del nombre en la implícita), MODIFIED y REMOVED (ID existente) y RENAMED (ID del nombre anterior).
- Identificador del requisito: con perfil declarado, se reconstruye la línea `strings.Repeat("#", level) + " " + nombre` y se aplica el patrón `heading` compilado (mismo criterio que `specbaseline.Profile.HeadingLine`); en la baseline implícita, el token final entre paréntesis del nombre, `\((REQ-[0-9]+\.[0-9]+)\)\s*$`. Un requisito de la delta sin identificador extraíble produce `iso_requirement_id_missing`.
- Bug sin delta: cada `Cubre:` debe existir en la baseline (consulta REQ-28.25); la cobertura exige al menos un `Cubre:`.

**Alternativas**: alias `Covers:`; contar las líneas huérfanas.
**Motivo**: la spec fija `Cubre:`; atribuir cada línea a una tarea es lo que da la traza requisito → tarea.

### D8. Matriz de `verify-report.md` (ISO)

**Elección**: una sección `## Matriz de verificación` con una tabla de nueve columnas, en este orden y con estos encabezados exactos:

```markdown
## Matriz de verificación

| REQ | Escenario | Tipo | Referencia | Resultado | Commit | Quién | Cuándo | Motivo |
|---|---|---|---|---|---|---|---|---|
| REQ-28.3 | Un requisito de la delta sin tarea que lo cubra bloquea | unitario | internal/sddevidence/coverage_test.go:TestCoverageMissing | pass | 1a2b3c4 | | | |
| REQ-28.6 | Cambio fusionado por squash | manual | Revisión del historial de main | pass | | ana.garcia | 2026-11-20 | Requiere un PR real fusionado en GitHub |
```

- `Tipo`: `unitario`, `integración`, `e2e` o `manual` (valores de la spec). `Resultado`: `pass` o `fail` (mismo vocabulario que `verdict:`).
- `Commit` es opcional e informativo. `Quién`, `Cuándo` (`AAAA-MM-DD`) y `Motivo` son obligatorios solo en las filas `manual`.
- Clave de fila: (REQ, Escenario), con el texto del escenario normalizado (recorte, espacios colapsados, `\|` → `|`). Varias filas para el mismo escenario se admiten y todas deben pasar.
- Escenarios exigidos: los `#### Scenario:` de los bloques ADDED y MODIFIED; en un bug sin delta, los del requisito existente citado. Si ese requisito no tiene escenarios en su documento (`scenarios: omit`), se exige una fila con el escenario `—`.
- Inválida (`iso_matrix_invalid`): falta `verify-report.md`, falta la sección, encabezados distintos, escenario sin fila, fila con REQ o escenario inexistente (supuesto aceptado en la compuerta `spec`), `Tipo` o `Resultado` fuera del dominio, manual sin justificación completa. Con resultado `fail`: `iso_matrix_failed`.
- No se comprueba que el test exista ni que el commit esté en git.

**Alternativas**: seis columnas con la justificación manual en la celda `Referencia` (`quién=…; cuándo=…`); una segunda tabla de evidencias manuales.
**Motivo**: columnas explícitas dan un análisis determinista y una tabla legible para el auditor; las tres últimas vacías en filas automáticas cuestan poco.

### D9. Design proporcional

**Elección**: secciones H2 reconocidas por su texto (sin distinguir mayúsculas), con alias en inglés para los proyectos que redactan en inglés:

| Nivel | Sección canónica | Alias |
|---|---|---|
| Mínimo | `Enfoque` | `Approach`, `Technical Approach` |
| Mínimo | `Roles` | `Affected Roles` |
| Mínimo | `Componentes` | `Components`, `File Changes` |
| Mínimo | `Riesgos` | `Risks` |
| Mínimo | `Disparadores` | `Design Triggers` |
| Mínimo bug | `Causa raíz`, `Corrección`, `Disparadores` | `Root Cause`, `Fix`, `Design Triggers` |
| Completo | `Decisiones`, `Flujo de datos`, `Interfaces y contratos`, `Estrategia de pruebas`, `Migración` | `Architecture Decisions`, `Data Flow`, `Interfaces / Contracts`, `Testing Strategy`, `Migration / Rollout` |

- `Disparadores` contiene una tabla `| Disparador | Marca | Motivo |`; la marca es `sí`, `si` o `no`. Cada disparador de la lista efectiva (base más `design_triggers`) debe tener exactamente una fila.
- Con alguna marca `sí`, se exige el nivel completo; `maintained-context` en `sí` mantiene además la exigencia actual de `context-impact.md`.
- `design.<rol>.md` cuyo rol no está en el roster sellado: `design_role_unknown`. Su ausencia nunca es hallazgo.
- Disparadores marcados: en `continuous`, `note(design_triggers_marked)`; en `checkpointed`, la pregunta de la compuerta `design` añade a `criteria` «Disparadores marcados: `<ids>`; revisor esperado: arquitecto». El status proyecta además `designTriggers: {marked: [...], reviewer: "architect"}` cuando hay marcas.
- Aviso de verify (REQ-28.13): solo cuando `nextRecommended` es `verify` o la fase archive, se calcula `git diff --name-only <merge-base(origen, HEAD)> HEAD` en el repositorio del planning home y se aplican heurísticas por ruta solo a disparadores base marcados `no`: `data-model` (`**/migrations/**`, `*.sql`), `new-component` (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`), `public-contract` (`*.proto`, `openapi*.y*ml`, `**/api/**`), `security-performance` (`**/auth/**`, `**/security/**`), `maintained-context` (rutas de `technical_context.maintained`). Los disparadores propios no tienen heurística.

**Alternativas**: secciones solo en castellano; marcas en un bloque YAML; calcular el diff en la skill `sdd-verify`.
**Motivo**: la plantilla actual de `sdd-design` ya está en inglés y sus secciones cubren el nivel completo, así que los design existentes ya cumplen parte; la tabla de disparadores se lee en la compuerta tal cual; validar en Go y no en prosa es un principio de la propuesta (§5.3).

### D10. Plan propio de `tasks.<rol>.md`

**Elección**: con varios roles, cada `tasks.<rol>.md` contiene:

```markdown
## Plan

| Unidad | Commit previsto | Depende de | En paralelo con |
|---|---|---|---|
| U1 | feat(core): exponer la partición blocking | — | U2 |
| U2 | test(core): fijar la barrera | — | U1 |
| U3 | feat(core): proyectar el aviso | U1 | — |
```

Validación: identificadores únicos (`[A-Za-z0-9._-]+`), commit no vacío, referencias existentes, sin autodependencia ni ciclos, y ningún par «en paralelo» con dependencia entre sí. `—` significa ninguno. Las tareas y su cómputo de progreso no cambian.
**Motivo**: la spec pide unidades con commit y paralelismo explícito; la tabla da las dos cosas y se valida sin heurísticas.

### D11. Corrección del QA diferido (pieza 1)

**Elección**: `multirole.GatePolicy.Blocks()` (verdadero para `blocking` y para el valor vacío o desconocido, fallo cerrado) es la única partición. `EvaluateBarrier` la usa en su `switch` y `verifyDependencyFromRoleGates` deja de recorrer `governance.Roster.Roles` para recorrer `governance.Kickoff.Config.Roles` y exigir solo los `role-apply` de los roles bloqueantes. Los no bloqueantes se informan en `note(verify_nonblocking_roles): <rol> (<política>, role-apply <pending|approved|rejected>)`.
**Motivo**: REQ-2.1 y REQ-21.15 exigen la misma partición; una función compartida impide que diverjan.

### D12. `kickoff seal` crea la carpeta (pieza 3)

**Elección**: sin `--infer`, `runSDDKickoffSeal` resuelve la raíz con una variante de `resolveGovernanceChangeRoot` que no exige existencia, valida el nombre (`gitref.ValidBranchComponent`), rechaza un cambio ya archivado, ejecuta la guarda A (`applySealBranchPolicy`, que ya resuelve el repositorio desde el directorio existente más cercano) y solo después hace `os.MkdirAll`. Si `kickoff.Seal` falla y la carpeta la creó esta ejecución y sigue vacía, se elimina. `--infer` sigue exigiendo la carpeta.
**Motivo**: REQ-28.20 pide crearla tras la guarda de rama y no tocar una carpeta existente.

### D13. Archivado en el mismo PR (Q11, Q11b)

**Elección**: un solo predicado en `internal/branch`:

```go
// ArchiveBranchApplies dice si el cambio tiene rama de archivado: carril SDD, topología con
// rama de cambio y proyecto sin `sdd.archive: same_pr` (REQ-25.45, REQ-28.22).
func ArchiveBranchApplies(in PlanInput) bool
```

`PlanInput` gana `SamePRArchive bool` (relleno en `gatherWith` desde `location.Config.SDD`). Lo consultan `CheckArchiveRequest` (rechazo `archive_branch_not_applicable` con el motivo del mismo PR, antes de evaluar `change_not_integrated`), `withArchiveBranch`, `PublicationTarget` (`archiveApplies`) y `Decide`: `VerdictInput.SamePRArchive` hace que, en el contexto de archivado, la única rama correcta sea la del cambio, sin ramas de rol, y que la continuación sea `axiom branch ensure --cwd <ruta> --change <c>` sin `--archive`. En `PublicationTarget`, con `--archive` y `same_pr`, solo la rama de tipo cambio es planificada (una rama de rol es `wrong_branch` nombrando la del cambio).

La compuerta `integration`: `kickoff.Inputs.SkipIntegration` (valor cero = comportamiento actual) omite su entrada en `EvaluateGates`; `loadGovernance` la recibe desde `ResolveOptions` y `archiveDependencyFromGovernance` no bloquea con `same_pr`. `gate record --gate integration` en un proyecto `same_pr` se rechaza con `integration_gate_not_applicable` sin escribir en `gates.yaml`; `gate show` no la lista. La guarda D (`engine/guard.go`) no cambia: con `same_pr` el contexto de archivado acepta la rama del cambio, que ya es correcta en el ordinario.

**Alternativas**: un indicador por cambio en la instantánea de `kickoff.yaml`; rechazar `publish --archive` con `same_pr`.
**Motivo**: Q11 descarta la opción por cambio; Q11b fija reutilizar el precedente de `multirepo`. Un único predicado sustituye las tres comprobaciones sueltas de hoy (`publication.go:46`, `archive_plan.go:102`, `planner_archive.go:125`).

### D14. Regla de escenarios en la composición (pieza 2)

**Elección**: `sddevidence.ScenarioViolations(domain string, delta []byte) []Finding` (ADDED y MODIFIED sin `#### Scenario:`). Se aplica en ISO:
- en `archive-compose --document` cuando la delta vive en la carpeta de un cambio (`changeRootOfDelta`): se validan **todas** las deltas del cambio antes de escribir, para que un dominio sin escenarios impida modificar los demás también en la baseline implícita, donde la skill compone dominio a dominio;
- en `specbaseline.Prepare` y `ApplyCompose`, dentro de la validación previa a cualquier escritura (REQ-24.46);
- nunca con `--canonical` (sin contexto de proyecto, REQ-24.38).
La negativa es `sdd-archive-compose: blocked(iso_scenarios_missing): dominio <d>, sección <ADDED|MODIFIED>, requisito <nombre>; …`. El status adelanta el mismo hallazgo desde que existe la delta.
**Motivo**: la validación conjunta da el «todo o nada» que pide REQ-24.46; adelantarlo en status evita descubrirlo al archivar (decisión a confirmar en la compuerta, P2).

### D15. Frontera del destino del archivado (D5)

**Elección**:

```go
// internal/workspace/archive_target.go
type ArchiveTargetKind string // "native" | "profiled" | "prose"

type ArchiveTarget struct {
    Domain   string
    Path     string             // absoluta
    Kind     ArchiveTargetKind
    Document *BaselineDocument  // nil en "native"
}

type ArchiveTargetResolver interface {
    Resolve(loc Location, domain string) (ArchiveTarget, error)
}

// ResolveArchiveTarget usa la única implementación: Markdown por dominio.
func ResolveArchiveTarget(loc Location, domain string) (ArchiveTarget, error)
```

La usan `composeDocumentMode`, la resolución de dominios de `specbaseline/plan.go` y `sddevidence` (identidad y consulta de requisitos). Los mensajes de error actuales se conservan byte a byte.
**Motivo**: D5 conectará otros formatos implementando la interfaz sin tocar a los consumidores; `workspace` ya conoce `PlanningHome` y la declaración.

### D16. Trazabilidad y verbo `axiom sdd trace`

**Elección**: `sddevidence.IntegrationCommit(ctx, gitRoot, ref, archivedRel)` ejecuta, con `gitcommon.Run`, `git log --first-parent --diff-filter=A --reverse --format=%H%x00%cI%x00%s <ref> -- <archivedRel>` y toma la primera línea; `ref` es `origin/<integración>` si existe en local y, si no, `<integración>`, sin `fetch`. Ruta ausente: `found=false`, sin error. Un verbo nuevo de solo lectura:

```
axiom sdd trace --cwd <ruta> --change <x> [--json]
```

reconstruye la cadena requisito → tareas con `Cubre:` → filas de la matriz → commit de integración (más la evidencia `pr_merged` de `gates.yaml` cuando existe, y la correspondencia de IDs de `integration.yaml`), para un cambio activo o archivado. JSON con esquema `axiom.sdd-trace/v1`. No escribe nada. `sddevidence` entra en `TestArchitectureOnlyExecGoLaunchesProcesses`.
**Alternativas**: ampliar `axiom sdd integrate show` (solo existe con baseline declarada); proyectar el commit en `status.archived`.
**Motivo**: la cadena tiene que poder reconstruirse también con la baseline implícita, que no tiene `integration.yaml`; un verbo propio no encarece cada `status`. Decisión de contrato visible para la compuerta (P1).

### D17. Proyección del bloque en el status

**Elección**: clave aditiva `sdd` en `Status` y `StatusV2`, **omitida** cuando `axiom.yaml` no declara el bloque:

```json
"sdd": {
  "iso": true,
  "archive": "same_pr",
  "designTriggers": ["public-contract", "new-component", "data-model", "security-performance", "maintained-context", "accessibility"],
  "source": "declared"
}
```

Con el bloque inválido: `{"source": "invalid"}` y D4. Además, `designTriggers` (marcas) de D9 y la nota del perfil.
**Alternativas**: proyectarla siempre, como `workflow`.
**Motivo**: REQ-28.24 admite omitirla; proyectarla siempre cambiaría el JSON de todas las pruebas y fixtures del status. La ausencia significa «no ISO, rama `-archive`, disparadores base», y las skills lo describen.

## Flujo de datos

```
axiom.yaml ──parseLayered──> Declarations.SDD / SDDErr ──> WorkspaceConfig.SDD (resuelto)
                                    │
workspace.Locate ──> Location ──────┼──> ResolveOptions.sdd (SkipIntegration, ISO)
                                    │
sddstatus.Resolve
  ├─ resolveStatus (openspec/hybrid)
  │    ├─ resolveArtifactPaths ── proposal.md | story.md
  │    ├─ sddevidence.ReadProfile ── bug sin delta ⇒ specs satisfecha
  │    ├─ loadGovernance(skipIntegration) ── kickoff.EvaluateGates
  │    ├─ verifyDependencyFromRoleGates ── multirole.GatePolicy.Blocks()
  │    └─ resolveBranchContext ── engine.VerdictFor ── branch.Decide(SamePRArchive)
  ├─ decorateWorkspaceDeclarations ── sdd projection, declaration causes
  └─ decorateEvidence
       └─ sddevidence.Evaluate(Input)
            ├─ deltas + workspace.ResolveArchiveTarget ── identidad, IDs, escenarios
            ├─ LookupRequirement (baseline) ── Cubre: inexistente, bug sin delta
            ├─ ParseCubre(tasks*.md) ── cobertura
            ├─ ParseDesign / ParseRolePlan ── severidad por ISO
            ├─ ParseMatrix(verify-report.md) ── fase archive
            └─ ChangedPaths (gitcommon) ── aviso de verify
       ⇒ blockedReasons / notes / nextRecommended / dependencies.archive

archive-compose ──> ResolveArchiveTarget ──> ScenarioViolations (ISO, todo el cambio) ──> componer
axiom sdd trace ──> artefactos (activos o archivados) + IntegrationCommit (git log --first-parent)
axiom sdd publish --archive ──> PublicationTarget ── ArchiveBranchApplies ⇒ rama del cambio con same_pr
```

## Interfaces y contratos

### `internal/workspace`

```go
type ArchiveMode string

const (
    ArchiveModeBranch ArchiveMode = "branch"
    ArchiveModeSamePR ArchiveMode = "same_pr"
)

// SDDConfig es el bloque `sdd:` (REQ-28.23). En Declarations conserva lo escrito; en
// WorkspaceConfig está resuelto (Archive nunca vacío, DesignTriggers con los base primero).
type SDDConfig struct {
    ISO            bool
    Archive        ArchiveMode
    DesignTriggers []string
}

var BaseDesignTriggers = []string{"public-contract", "new-component", "data-model", "security-performance", "maintained-context"}

// Declarations gana SDD *SDDConfig y SDDErr error; WorkspaceConfig gana SDD *SDDConfig.
// readDeclarationBlocks recibe la topología leída para rechazar same_pr en multirepo.
func decodeSDD(node *yaml.Node, topology TopologyType) (*SDDConfig, error)
func resolveSDD(declared *SDDConfig) *SDDConfig
func SamePRArchive(loc Location) bool // false sin config o con bloque inválido
```

`decodeStringKeys` solo admite escalares, así que `decodeSDD` recorre el mapeo con `unknownKeys` y la detección de claves repetidas, y decodifica `design_triggers` como secuencia propia.

### `internal/sddevidence`

```go
type Severity string // "block" | "note"

type Finding struct {
    Token        string
    Severity     Severity
    Phase        string // propose|spec|design|tasks|verify|resolve-blockers|"" (notas)
    Subject      string
    Detail       string
    Continuation string
}

func (f Finding) Reason() string // "blocked(<token>): …; …" o "note(<token>): …"

type ProfileKind string // "proposal" | "story" | "bug"
type Profile struct{ Kind ProfileKind; ExternalRef string; Source string }
func ReadProfile(changeRoot string) (Profile, []Finding)

type DeltaRequirement struct {
    Domain, Section, Name, ID string // Section: ADDED|MODIFIED|REMOVED|RENAMED
    Scenarios []string
}
func ParseDelta(domain string, data []byte, idOf func(name string) (string, bool)) []DeltaRequirement
func ScenarioViolations(domain string, data []byte) []Finding

type Requirement struct{ ID, Domain, Name string; Scenarios []string }
func LookupRequirement(loc workspace.Location, id string) (Requirement, bool, error)

type CubreLine struct{ File, Task string; Line int; IDs []string; Orphan bool }
func ParseCubre(file string, data []byte) []CubreLine

type MatrixRow struct{ Line int; REQ, Scenario, Type, Reference, Result, Commit, Who, When, Why string }
func ParseMatrix(data []byte) ([]MatrixRow, []Finding)

type Design struct{ Sections map[string]bool; Marks map[string]bool; MissingMarks []string }
func ParseDesign(data []byte) Design
func ParseRolePlan(file string, data []byte) []Finding

type Input struct {
    Location     workspace.Location
    ChangeRoot   string
    Config       workspace.SDDConfig // resuelto; ISO=false sin bloque
    Store        string
    Execution    string              // continuous | checkpointed | ""
    Roster       []string
    Phase        PhaseFacts          // qué artefactos existen y si es verify o archive
    ChangedPaths func() ([]string, error)
}
func Evaluate(in Input) []Finding

type Commit struct{ SHA, Date, Subject string }
func IntegrationCommit(ctx context.Context, gitRoot, ref, archivedRel string) (Commit, bool, error)
func ChangedPaths(ctx context.Context, gitRoot, base string) ([]string, error)
```

### Otros contratos

- `kickoff.Inputs.SkipIntegration bool`; `multirole.GatePolicy.Blocks() bool`.
- `branch.PlanInput.SamePRArchive`, `branch.VerdictInput.SamePRArchive`, `branch.ArchiveBranchApplies(PlanInput) bool`.
- Negativas nuevas: `integration_gate_not_applicable` (`gate record`), `blocked(iso_scenarios_missing)` (`archive-compose`), `blocked(iso_matrix_invalid|iso_matrix_failed)` (`archive-compose` en ISO con la delta de un cambio, defensa en profundidad de REQ-28.5).
- Status v2: `sdd` y `designTriggers` aditivos y omitibles; ninguna clave existente cambia (REQ-24.20).
- `axiom sdd trace`: esquema `axiom.sdd-trace/v1` (`change`, `archivedPath`, `requirements[{id, section, tasks[{file,task}], rows[...]}]`, `integrationCommit{sha,date,subject}|null`, `prMerged{commit}|null`, `idMap[]`).

## Estrategia de pruebas

TDD: desactivado (`strict_tdd: false` en `openspec/config.yaml`); cada unidad incluye las pruebas de sus escenarios en el mismo commit. Las pruebas RED de la matriz de amenazas sí se escriben antes del código de producción (regla de `sdd-design`). Solo `go test` acotado por paquete; **nunca** `go build` ni `go run` de binarios sueltos (Cortex XDR borra `axiom.exe`). La regla `rules.design` de diagramas de secuencia para flujos TUI no aplica: el cambio no toca `internal/tui/`.

| Capa | Qué | Cómo |
|---|---|---|
| Unitaria | Decodificación de `sdd:` (todos los escenarios de REQ-28.23) | `go test ./internal/workspace -run 'SDD'`, tablas sobre `parseLayered` |
| Unitaria | Analizadores y evaluador | `go test ./internal/sddevidence/...`, tablas por escenario de REQ-28.2 a REQ-28.17 y REQ-28.25 |
| Unitaria | Veredicto, predicado y publicación con `same_pr` | `go test ./internal/branch/...` (tablas de `Decide`, `CheckArchiveRequest`, `PublicationTarget`) |
| Unitaria | Partición y compuertas | `go test ./internal/multirole/... ./internal/kickoff/...` |
| Integración | Status y continue | `go test ./internal/sddstatus/...` con fixtures en `t.TempDir()`; `evidenceEvaluator` activo solo en las pruebas nuevas |
| Integración | CLI (`kickoff seal`, `gate record`, `archive-compose`, `trace`) | `go test ./internal/cli -run '<Verbo>'` |
| Integración | Trazabilidad con squash, merge y rebase | repositorios git en `t.TempDir()` con `workspacetest`; una prueba por estrategia |
| Arquitectura | `sddevidence` solo ejecuta git con `gitcommon.Run`; no importa `specbaseline` | `go test ./internal/gitcommon -run Architecture`; prueba nueva en `sddevidence` |
| Paridad | Composición nativa byte a byte sin ISO | pruebas existentes de `specbaseline` y `cli` sin modificar aserciones |
| Prosa | Pines de texto y paridad entre runtimes | `go test ./internal/components/... ./internal/assets/...`; goldens con `-update`, una sola vez (U16) |
| Manual | `docs/manual-de-inicio.md` | `go test ./internal/workspace -run ManualDeInicio` (pines existentes intactos) |

CI final: `go vet ./...` y `go test` de los paquetes tocados; ratchets de negativas y código muerto sin empeorar.

## Migración

No hay migración de datos ni de cambios activos. Sin bloque `sdd:` el comportamiento es el actual salvo las notas no bloqueantes de design y plan (REQ-28.17) y las correcciones 1 y 3. Despliegue: binario nuevo y `axiom sync` para la prosa. Reversión: retirar el bloque `sdd:` o volver a la versión anterior (`proposal.md` §10). Antes de aplicar, la rama debe avanzar en rápido a `origin/main` (b7ddbf51, que archiva #235 con #236), para que los MODIFIED partan del texto ya integrado en `openspec/specs`.

## Ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/sdd.go` | Crear | `SDDConfig`, `decodeSDD`, `resolveSDD`, `SamePRArchive`, `BaseDesignTriggers` |
| `internal/workspace/archive_target.go` | Crear | Frontera D5 con su única implementación |
| `internal/workspace/{declarations,loader,types}.go` | Modificar | Canal `SDD`/`SDDErr`, topología en `readDeclarationBlocks`, `joinLoadErrors` |
| `internal/sddevidence/{profile,delta,lookup,cubre,matrix,design,plan,evaluate,git,trace}.go` | Crear | Analizadores, evaluador, consulta y git de solo lectura |
| `internal/sddstatus/evidence.go` | Crear | `decorateEvidence`, `evidenceEvaluator`, ruta y dependencias |
| `internal/sddstatus/{status,status_v2,workspace_declarations,governance,governance_handoff}.go` | Modificar | Localizador de `story.md`, perfil, `sdd`, `designTriggers`, `SkipIntegration`, partición, criterios de la compuerta `design` |
| `internal/multirole/{types,barrier}.go` | Modificar | `GatePolicy.Blocks()` |
| `internal/kickoff/machine.go` | Modificar | `Inputs.SkipIntegration` |
| `internal/branch/{planner,planner_archive,verdict}.go` | Modificar | `SamePRArchive`, `ArchiveBranchApplies`, veredicto de archivado |
| `internal/branch/engine/{engine,verdict,publication,archive_plan}.go` | Modificar | Relleno del indicador y uso del predicado |
| `internal/cli/{sdd_kickoff,sdd_governance,sdd_gate,sdd_gate_handoff,sdd_archive_compose}.go` | Modificar | Carpeta en `seal`, compuerta `integration`, escenarios y matriz en ISO |
| `internal/cli/sdd_trace.go`, `cmd/axiom/main.go` | Crear y modificar | Verbo `axiom sdd trace` |
| `internal/specbaseline/plan.go` | Modificar | Frontera D5 y regla de escenarios en la validación previa |
| `internal/gitcommon/architecture_test.go` | Modificar | Incluir `internal/sddevidence` |
| `internal/assets/skills/sdd-{propose,spec,design,tasks,apply,verify,archive}/SKILL.md` | Modificar | `story.md`, bug, design proporcional, `Cubre:`, plan, matriz, mismo PR |
| `internal/assets/skills/_shared/{persistence-contract,sdd-status-contract,openspec-convention}.md` | Modificar | Excepción ISO, tokens y claves nuevas, carpeta creada por `seal` |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Compuerta `design` con disparadores, `integration` no aplicable, `story.md`/bug, mismo PR (espera a la cadena) |
| `internal/components/agentguidance/routing.go` | Modificar | D4 en el protocolo ODD (espera a la cadena) |
| `testdata/golden/*`, `.codex/AGENTS.md` y demás copias versionadas | Regenerar | Con la herramienta de actualización, una vez |
| `docs/manual-de-inicio.md`, `axiom.example.yaml` | Modificar | Bloque `sdd:`, ISO, perfiles, mismo PR, `axiom sdd trace` |

## Matriz de amenazas

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta del diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas tipo documentación | `requirements.txt`, Markdown ejecutable | N/A: no se clasifica ni ejecuta ningún fichero; los analizadores solo leen texto de la carpeta del cambio y de la baseline | — | — |
| Selección del repositorio git | `git -C`, rutas relativas y absolutas | Aplicable | `IntegrationCommit` y `ChangedPaths` reciben la raíz git absoluta resuelta desde `Location` (nunca el cwd del proceso) y pasan la ruta archivada como `pathspec` tras `--`, validada con el mismo criterio que `gitcli.pathspecArgument`; `ref` validado con `plainArgument` | Raíz relativa rechazada; ruta con `..` o que empieza por `-` rechazada; ejecución desde otro cwd da el mismo commit |
| Estado del commit | índice preparado, `commit -a`, índice vacío | Aplicable (solo `publish --archive` con `same_pr`) | Sin código nuevo de commit: `sddpublish` confirma solo las rutas de `docpaths`; el cambio es la rama esperada | `publish --archive` en la rama del cambio con un fichero de código modificado: el commit no lo incluye |
| Estado del push | rama con seguimiento, primer push, refspec explícito | Aplicable (solo destino) | `PublicationTarget` con `same_pr` devuelve la rama del cambio; en rama de rol o integración, `wrong_branch` sin publicar | Rama de rol y `main` con `same_pr`: `wrong_branch` nombrando la rama del cambio y sin `push` (observador de git) |
| Órdenes de PR | `--head` explícito, prefijo de entorno | N/A: Axiom no abre PR ni ejecuta `gh` (invariante del motor de ramas) | — | — |

## Unidades de trabajo

Cada unidad es un commit Conventional con sus pruebas. Entrega: PR única con `size:exception`.

| Unidad | Pieza | Contenido | Depende de | Toca prosa del orquestador |
|---|---|---|---|---|
| U1 | 1 | `GatePolicy.Blocks()`, barrera y `verifyDependencyFromRoleGates` | — | No |
| U2 | 3 | `kickoff seal` crea la carpeta | — | No |
| U3 | 4 | Bloque `sdd:` en `workspace` y `workspace validate` | — | No |
| U4 | 4 | Proyección `sdd`, `sdd_declaration_invalid`, `iso_store_unsupported`, decorador y `evidenceEvaluator` | U3 | No |
| U5 | 11 | Frontera D5 y su adopción sin cambio observable | — | No |
| U6 | 4 | Delta, IDs, consulta REQ-28.25, identidad ISO | U4, U5 | No |
| U7 | 2 | Regla de escenarios (status, `archive-compose`, `specbaseline`) | U6 | No |
| U8 | 8 | `story.md` y perfil de bug | U4 | No |
| U9 | 5 | `Cubre:` y cobertura (incluido bug sin delta) | U6, U8 | No |
| U10 | 9 | Design proporcional, `design.<rol>.md`, plan propio, criterios de la compuerta, aviso de verify | U4 | No |
| U11 | 6 | Matriz y bloqueo del archivado ISO | U6, U9 | No |
| U12 | 10 | Archivado en el mismo PR (rama, veredicto, publicación, compuerta `integration`) | U3 | No |
| U13 | 7 | `IntegrationCommit` y `axiom sdd trace` | U9, U11 | No |
| U14 | 13 | Skills `sdd-*`, doctrina compartida, manual y ejemplo | U1–U13 | No |
| U15 | 12, 13 | `sdd-orchestrator-sections.md` y `routing.go` (D4) | U14 y cadena de prosa | **Sí** |
| U16 | 13 | Regeneración de goldens y copias versionadas | U15 | Sí (proyección) |

## Preguntas abiertas

- [ ] P1. ¿Se acepta el verbo nuevo `axiom sdd trace` (D16) frente a ampliar `integrate show`?
- [ ] P2. ¿Se acepta adelantar en status el bloqueo `iso_scenarios_missing` desde que existe la delta (D14), además del rechazo al componer que exige REQ-28.21?
- [ ] P3. ¿Se acepta que un `sdd:` inválido bloquee todas las fases (D4), a diferencia de las demás declaraciones (REQ-24.18)?
- [ ] P4. ¿Se acepta que el modo ISO exija almacén `openspec` o `hybrid` (D5)?
- [ ] P5. ¿Basta con advertir que cambiar `sdd.archive` con cambios a medio archivar mezcla reglas, o debe fijarse en la instantánea del kickoff?
- [ ] P6. ¿Debe `axiom branch done` aceptar, con `same_pr`, la derivación por `--first-parent` como evidencia de fusión (hoy no lo pide ninguna spec)?
- [ ] P7. Mecanismo de regeneración de las copias versionadas de la raíz (`.codex/AGENTS.md`, `AGENTS.md`, `GEMINI.md`) sin compilar binarios sueltos: a confirmar en `tasks`.
- [ ] Q7 de la propuesta («Listo»/«Hecho») sigue abierta; este diseño no depende de ella.
