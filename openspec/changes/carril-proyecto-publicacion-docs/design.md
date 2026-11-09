# Diseño: Carril elegido en el proyecto, ODD sin paradas y publicación ligera de documentación SDD (carril-proyecto-publicacion-docs)

> Fase `design` · rama `inc/carril-proyecto-publicacion-docs` · 2026-11-09.
> Entradas autoritativas: `proposal.md` (D1–D8; supuestos S2, S4, S8 y S9 confirmados por el usuario el 2026-11-09) y `exploration.md`. Las delta specs se escriben en paralelo; este documento usa los identificadores que ya fijan (REQ-27.1 a REQ-27.17) y los tokens que ya citan (`workflow`, `source: declared|default|no_workspace|invalid`, `docs_publication: auto`, `code_commits_unpublished`, `wrong_branch`).
> Kickoff sellado: `continuous`, rol único `fullstack`. Este cambio no tiene puertas propias.
> Idioma: castellano. Identificadores de código, rutas, flags y claves YAML/JSON en inglés.
> Las decisiones de este documento se numeran **DD-01…DD-16** para no confundirlas con las de la propuesta (**D1…D8**).
> Contexto técnico mantenido: `axiom.yaml` no declara `technical_context`, así que **no hay `context-impact.md`** (paso 2b de la skill).
> Este documento fija el CÓMO a nivel de arquitectura. No contiene tareas.

## 0. Resumen

El cambio se apoya en cuatro piezas nuevas y en piezas existentes que se reorientan:

1. **Bloque `workflow` en `axiom.yaml`** con la clave `lane`, decodificado estricto con el patrón del bloque `git:` y proyectado en `axiom sdd status` (v1 y v2, con o sin cambio) como `workflow: {lane, source}`. La doctrina lo lee en el paso 0; con `no_workspace`, `invalid` o un status inalcanzable se comporta como `ask`.
2. **Doctrina ODD sin paradas.** `routing.go` deja de proyectar `StopExits()`; `stop-exits.md` pasa a ser texto solo SDD y se reescribe para «aprobar publica».
3. **`axiom sdd publish`**, un verbo de `axiom sdd` con un paquete propio, `internal/sddpublish`, que es el único código del producto que ejecuta `git commit` y `git push`, siempre a través de `internal/gitcommon/exec.go`. La lista cerrada de rutas de documentación vive en un paquete puro, `internal/docpaths`. El destino (rama permitida, raíz git, rama de integración) lo calcula una lectura nueva del motor de ramas, `Engine.PublicationTarget`, que no publica nada.
4. **Adopción de la rama publicada** en `axiom branch ensure`: si la rama del cambio, de rol o de archivado no existe en local y existe `refs/remotes/origin/<rama>`, se crea con `git branch --track` desde esa referencia. El resultado lo indica por rama (`branches[].createdFrom: remote`), sin tocar `base.source`.

Lo demás reorienta lo que ya existe: el sobre de la puerta nombra el remoto y la rama y trae la orden de publicación; `gate record --hand-off` deja de exigir artefactos confirmados y de calcular `add`/`commit`/`push`; el kickoff gana `docs_publication: auto`.

### 0.1 Decisiones

| ID | Decisión | Alternativa descartada |
|---|---|---|
| DD-01 | `workflow: {lane: odd\|sdd\|ask}` con `decodeWorkflow` sobre `decodeStringKeys`; canal `Declarations.WorkflowErr`; `WorkspaceConfig.Workflow` resuelto (defecto `sdd`) | Decodificación tolerante; clave dentro de `git:` |
| DD-02 | `Status.Workflow *WorkflowProjection` (`lane`, `source`), calculado en `decorateWorkspaceDeclarations`; `no_workspace` e `invalid` proyectan `ask` | Omitir el campo sin `axiom.yaml`; proyectar `lane` vacío |
| DD-03 | Literal `explicit_request_project_lane_or_accepted_proposal` y constante `SDDSelectionExplicitRequestProjectLaneOrAcceptedProposal`; se retira el anterior | Conservar los dos literales |
| DD-04 | Paso 0 lee `axiom sdd status --json`; se retiran la parada ODD, la viñeta `Plan stop:` y `### Stops and exits` de `routing.go` | Hornear el carril en la doctrina |
| DD-05 | `FlowConfig.DocsPublication` (`docs_publication: auto`, `omitempty`), flag `--docs-publication auto` solo con `continuous`; proyectado en `governance.kickoff.docsPublication` | Campo de nivel raíz en `kickoff.yaml`; inferirlo de la *pace* |
| DD-06 | Paquete puro `internal/docpaths` con la lista cerrada S6, usado por `sddpublish` | Lista dentro de `sddpublish`; ponerla en `branch` |
| DD-07 | `Engine.PublicationTarget` (solo lecturas) calcula raíz git, rama permitida, integración e *in place* con `gather` + `Plan` | Que `sddpublish` replanifique; leer solo la instantánea del kickoff |
| DD-08 | Paquete `internal/sddpublish` con su puerto `Git` (lecturas, `Commit`, `Push`) y adaptador sobre `gitcommon.Run`/`RunWithTimeout` | Ampliar `branch.Git` o el `gitWriter` del motor |
| DD-09 | Algoritmo de `publish`: lecturas → commit `add --all` + `commit --only` por raíces de documentación → commits sin subir → push `refs/heads/<b>:refs/heads/<b>` | Commit de ficheros sueltos; comprobar el código antes de confirmar |
| DD-10 | CLI `axiom sdd publish --cwd --change [--archive] [--branch] [--json]`, esquema `axiom.sdd-publish/v1`, salida 0/1 | Subcomando de `axiom branch`; flag `--publish` en `gate record` |
| DD-11 | Sobre de la puerta con `publication {remote, branch, invocation}` y etiquetas «publish»; `gate record --json` devuelve `publish`; la doctrina encadena | Ejecutar `publish` desde `gate record` |
| DD-12 | `--hand-off` sin ninguna comprobación de «confirmado» (tampoco del código del rol); se retira `uncommitted_artifacts`; `HandOffPlan` sin `MergeRequired` y con `publish` (con `prTarget`) y `resume` | Mantener `uncommitted_artifacts` para la carpeta del cambio o para el código del rol |
| DD-13 | Adopción en `ensure` (ramas de cambio, de rol y de archivado) con `CreateTrackingBranch` y `branches[].createdFrom`/`createdFromRef`; `checkIntegrated` sin cambios | `base.source: remote_branch`; `ensure --adopt-remote` con fetch |
| DD-14 | Aislamiento: prueba de arquitectura ampliada a `internal/sddpublish`, prueba de importaciones y observador dinámico sobre `gate record` y `axiom branch` | Solo la prueba de arquitectura existente |
| DD-15 | `continuous` publica al cerrar `tasks` y tras el archivado solo con `docs_publication: auto`; el archivado usa `publish --archive` | Detectar el contexto de archivado por la rama actual |
| DD-16 | Entrega en cuatro commits de unidad de trabajo en el orden 1 → 2 → 4 → 3; regeneración sin binarios | Orden 1 → 2 → 3 → 4 literal |

## 1. Enfoque técnico

| Propuesta | Pieza de diseño | Paquetes |
|---|---|---|
| D1, S1 | DD-01 | `internal/workspace` |
| D2, S3, S10 | DD-02 | `internal/sddstatus`, `sdd-status-contract.md` |
| D3, S2 | DD-03, DD-04 | `internal/agents/capabilitymanifest`, `internal/components/agentguidance` |
| D4 | DD-04 | `agentguidance`, `internal/assets/generic/stop-exits.md` |
| D5, S6, S9 | DD-06 a DD-10 | `internal/docpaths`, `internal/branch/engine`, `internal/sddpublish`, `internal/gitcommon`, `internal/cli`, `cmd/axiom` |
| D6 | DD-11, DD-12 | `internal/sddstatus`, `internal/cli`, `internal/branch`, `sdd-orchestrator-sections.md` |
| D7, S7 | DD-13 | `internal/branch/engine`, `internal/branch/gitcli`, `internal/branch` |
| S4, S5 | DD-05, DD-15 | `internal/kickoff`, `internal/sddstatus`, doctrina |
| D8 | DD-16 | entrega |

Tres reglas atraviesan todo el diseño:

- **Fallo cerrado hacia preguntar.** Un `axiom.yaml` ausente o inválido, o un status que no responde, nunca activan SDD automático (S10).
- **Decisión y E/S separadas.** `gate record` solo registra; `publish` confirma y sube. El único código que ejecuta `commit` o `push` está en `internal/sddpublish`, y las pruebas lo fijan (DD-14).
- **Sin red implícita.** Ni `ensure` ni `publish` hacen `fetch`, `pull` ni `ls-remote`. `publish` solo hace `push`, y `ensure` solo lee referencias remotas que ya existen en local.

## 2. Decisiones de arquitectura

### 2.1 DD-01 — Bloque `workflow` de `axiom.yaml`

**Elección.** En un fichero nuevo, `internal/workspace/workflow.go`:

```go
type WorkflowLane string

const (
	WorkflowLaneODD WorkflowLane = "odd"
	WorkflowLaneSDD WorkflowLane = "sdd"
	WorkflowLaneAsk WorkflowLane = "ask"
	// DefaultWorkflowLane es el carril cuando axiom.yaml no declara `workflow.lane` (REQ-27.1).
	DefaultWorkflowLane = WorkflowLaneSDD
)

// WorkflowConfig es el bloque `workflow:` (REQ-27.1). En Declarations.Workflow conserva lo
// escrito (Lane vacío si no se declaró la clave); en WorkspaceConfig.Workflow, Lane está siempre
// relleno.
type WorkflowConfig struct {
	Lane WorkflowLane `yaml:"lane,omitempty"`
}

var workflowKeys = []string{"lane"}

func decodeWorkflow(node *yaml.Node) (*WorkflowConfig, error)
func resolveWorkflow(declared *WorkflowConfig) *WorkflowConfig
```

- `decodeWorkflow` sigue `decodeGit` línea a línea: un nodo que no es un mapeo (incluido `workflow:` vacío) se rechaza con «el bloque 'workflow' debe ser un mapeo con la clave opcional 'lane' (odd, sdd o ask); para usar el valor por defecto, elimina el bloque»; las claves se recorren con `decodeStringKeys(node, "workflow", workflowKeys, workflowValidator)`. El validador acepta exactamente `odd`, `sdd` y `ask`, distinguiendo mayúsculas, y si no responde «workflow.lane: el valor 'agil' no es válido: usa odd, sdd o ask».
- `Declarations` gana `Workflow *WorkflowConfig` y `WorkflowErr error`. `readDeclarationBlocks` lee `mappingValue(root, "workflow")`. `joinLoadErrors` añade `WorkflowErr`, de modo que `axiom workspace validate` y `LoadConfig` lo informan con la clave.
- `WorkspaceConfig` gana `Workflow *WorkflowConfig \`yaml:"-"\``. `parseLayered` lo proyecta con `resolveWorkflow` cuando `WorkflowErr == nil`, igual que `resolveGit`.
- `branch.ConfigRefusal` **no** lee `WorkflowErr`: un carril inválido no impide planificar ramas. `blockArchiveOnDeclarations` tampoco lo lee: según REQ-27.2, el resto del status se proyecta como hasta ahora.
- `hub/init.go` (`ownMarkers.declares`) añade `Workflow != nil || WorkflowErr != nil`, para que `init --force` no sobrescriba un bloque declarado.
- `axiom.example.yaml` documenta el bloque comentado, y el `axiom.yaml` de este repositorio declara `workflow: {lane: ask}` (S8).

**Alternativas descartadas.** Decodificación tolerante (un `lane: SDD` o un error tipográfico acabaría en `sdd` en silencio). Clave dentro de `git:` o de `governance:` (D1).

**Motivo.** Es el patrón con pruebas y pines ya existentes, y un binario anterior ignora la clave (REQ-24.17).

### 2.2 DD-02 — Proyección `workflow` en `axiom sdd status`

**Elección.** En `internal/sddstatus/status.go`:

```go
// WorkflowProjection es el carril efectivo del proyecto (REQ-27.2).
type WorkflowProjection struct {
	Lane   string `json:"lane"`   // odd | sdd | ask
	Source string `json:"source"` // declared | default | no_workspace | invalid
}
```

- `Status.Workflow *WorkflowProjection \`json:"workflow,omitempty"\`` se coloca tras `TechnicalContext`, y `statusV2.Workflow` tiene la misma forma (en v2 se copia sin tipo `*V2`, igual que `Baseline`).
- `decorateWorkspaceDeclarations` lo rellena **siempre** con `projectWorkflow(location)`, y por tanto existe con y sin cambio seleccionado. `omitempty` solo protege los `Status` construidos a mano en pruebas.
- Reglas de `projectWorkflow`, evaluadas en este orden:

| Condición | `lane` | `source` |
|---|---|---|
| `location.ConfigPath == ""` (sin `axiom.yaml`, o `LocateAt` falló con raíz explícita) | `ask` | `no_workspace` |
| `location.ConfigErr != nil`, `Declarations.ParseErr != nil` o `Declarations.WorkflowErr != nil` | `ask` | `invalid` |
| `Declarations.Workflow != nil` con `Lane` declarado | el declarado | `declared` |
| resto (`axiom.yaml` válido sin el bloque, o el bloque sin `lane`) | `sdd` | `default` |

- Un `axiom.yaml` cuya configuración base no carga se trata como `invalid`, aunque el bloque `workflow` fuera válido: es fallo cerrado.
- No se añade nota ni motivo bloqueante: el campo es informativo y no cambia la ruta.
- `sdd-status-contract.md` documenta el campo, sus valores y la regla «`no_workspace` e `invalid` proyectan `ask`; si el status no responde, el orquestador actúa como `ask`».

**Alternativas descartadas.** Omitir el campo sin `axiom.yaml`: obliga al consumidor a deducir el carril de una ausencia. `lane` vacío con `invalid`: el consumidor tendría que traducirlo y podría equivocarse hacia `sdd`.

**Motivo.** Sigue el precedente de `baseline`/`technicalContext` (REQ-24.19) y deja la regla de fallo cerrado en el binario, no en la prosa.

### 2.3 DD-03 — Política de selección del manifiesto

**Elección.** En `capabilitymanifest/manifest.go`, `SDDSelectionExplicitRequestOrAcceptedProposal` se sustituye por:

```go
const SDDSelectionExplicitRequestProjectLaneOrAcceptedProposal SDDSelectionPolicy = "explicit_request_project_lane_or_accepted_proposal"
```

`ForAgent` usa el valor nuevo en todos los adaptadores. `RenderRouting` compara contra la constante nueva y sigue devolviendo `ErrUnknownRoutingPolicy` ante cualquier otro valor, incluido el literal retirado (prueba explícita).

**Alternativa descartada.** Conservar el literal y cambiar solo la prosa (D3).

**Motivo.** El literal no tiene consumidores externos (verificado: solo `routing.go`, `manifest_test.go` y dos nombres de prueba en `internal/cli`), y el pin pasa a describir la política real.

### 2.4 DD-04 — Doctrina: paso 0 con carril y ODD sin paradas

**Elección.** Estos son los cambios en `routing.go`, todos en texto inglés estático e idéntico para todos los adaptadores:

1. **Frase introductoria.** «SDD is a branch inside ODD, entered only by an explicit request, by the project lane declared in `axiom.yaml`, or by an accepted proposal.»
2. **Paso 0** (texto completo):

   > 0. **Evaluate scope and lane.** Before creating any work artifact — `odd/tasks/<feature-name>.md` or `openspec/changes/<change>/proposal.md` — read the project lane from `axiom sdd status --cwd <repo> --json`: its `workflow` field carries `lane` (`odd`, `sdd` or `ask`) and `source`, with or without a selected change. Small, understood work stays small in every lane and creates no work artifact. For substantial work: with `lane: sdd`, enter the SDD pre-flight questionnaire directly, without asking the lane question; with `lane: odd`, work through ODD without asking, even for an architectural scope, unless the user explicitly asks for SDD in this request; with `lane: ask`, when `source` is `no_workspace` or `invalid`, or when the status cannot be read, ask the blocking lane-selection question — do you want to approach this through the agile ODD lane or the formal SDD lane? — and STOP to wait for the answer, creating no work artifact while it is pending; under `ask`, a scope that is unambiguously architectural or large skips that binary question and enters the SDD pre-flight questionnaire directly. Whenever the work goes through ODD, do not ask any further governance question: proceed straight into step 1 below, work it through without dividing it into roles, and keep it exclusively in `odd/tasks/<feature-name>.md`.

3. **Paso 5.** Se elimina la frase «Then present the plan stop: …».
4. **Párrafo de reanudación.** Se eliminan «(for example, the feature was handed off to you)» y «never review an approved plan again». Queda «When the Engram mirror does not exist, start from the task file alone, never fail for the missing mirror, and continue with the first unfinished task.»
5. **Viñeta «Optional SDD».** «SDD is selected only by an explicit request, by the project lane `workflow.lane: sdd` for substantial work, or by an accepted proposal.»
6. **Retiradas.** Se eliminan la viñeta `Plan stop:` con su comentario y la sección `### Stops and exits`, incluida la llamada a `StopExits()`. `agentguidance.StopExits()` sigue existiendo: la usa el compositor de orquestadores SDD (`internal/components/sdd/orchestrator.go`).
7. **Comentarios.** Los comentarios Go que citan REQ-26.10 o DD-13 de `paradas-tres-salidas` se sustituyen por referencias a REQ-27.16 y REQ-27.17.

`stop-exits.md` se reescribe solo para SDD. En el incremento 1 solo se retiran la mención de ODD y la «single ODD plan stop». La reescritura completa, con «publish», llega en el incremento 3 (DD-11). Las pruebas de texto cambian así:

- `stop_exits_test.go`: `TestStopExitsNamesNoGateInternals` deja de prohibir `gate record`. Pasa a prohibir `ODD`, `plan stop` y `odd/tasks`, y sigue prohibiendo `gateQuestion`, `gateCriteria` y `gateEvidence`.
- `routing_stops_test.go`: se reescribe como prueba negativa. Ningún texto de `RenderRouting` contiene `Plan stop`, `plan stop`, `### Stops and exits`, `three exits` ni `handed off`.
- `TestRenderRoutingKeepsSDDSelectionExplicit` conserva sus tres pines y añade `project lane` y `workflow`.

**Alternativa descartada.** Hornear el carril al sincronizar (D2): imposible en ámbito global y obsoleto en cuanto cambia `axiom.yaml`.

**Motivo.** Es la única vía por la que un texto idéntico para todos los proyectos conoce el proyecto. El trabajo pequeño queda fuera del carril, como fija S2.

### 2.5 DD-05 — Autorización de publicación en el kickoff

**Elección.** En `internal/kickoff/types.go`:

```go
// DocsPublication es la autorización, sellada, de publicar los documentos del cambio sin preguntar
// en el estilo continuous (REQ-27.4, REQ-27.13). Vacía significa «no se publica automáticamente».
type DocsPublication string

const DocsPublicationAuto DocsPublication = "auto"

type FlowConfig struct {
	// ... campos actuales ...
	DocsPublication DocsPublication `yaml:"docs_publication,omitempty"`
}
```

- `kickoff seal` gana el flag `--docs-publication auto` y `SealArgs.DocsPublication`. `ParseSealArgs` rechaza cualquier valor distinto de `auto`. Tras resolver el estilo efectivo (también con `--from-session-pace`), si el estilo no es `continuous` lo rechaza con «`--docs-publication` solo acompaña a `--execution-style continuous`». Con `--infer` (retrosellado) lo rechaza siempre: la autorización nunca se infiere.
- `Kickoff.Validate` admite `""` y `auto`, y rechaza `auto` con `checkpointed` (fichero editado a mano).
- El sellado único no cambia: un segundo `seal` devuelve «Kickoff ya estaba sellado» y no reescribe el campo.
- `ParseKickoff` usa `yaml.Unmarshal` no estricto: un binario anterior ignora la clave.
- Proyección: `kickoffV2.DocsPublication string \`json:"docsPublication,omitempty"\``. En v1, `Governance.Kickoff` se serializa por reflexión y el campo aparece sin trabajo adicional. `sdd-status-contract.md` lo documenta en `governance.kickoff`.
- Doctrina (sección de kickoff, `sdd-orchestrator-sections.md`): cuando el orquestador pregunta el estilo de ejecución, la opción `continuous` dice «continuous: no stops; pushes the change documents to `origin/<branch>` when `tasks` closes and after archive», con la rama real del JSON de `axiom branch ensure`. Solo si la persona elige esa opción se sella con `--docs-publication auto`. Si el estilo sale de la *pace* de sesión sin esa pregunta, se sella sin el flag.

**Alternativas descartadas.** Campo de nivel raíz: mezcla la autorización con la identidad del documento. Inferirlo de la *pace* `auto`: la *pace* no nombra remoto ni rama y no es autorización (S4).

**Motivo.** La autorización queda sellada junto al estilo que la justifica, y el cuestionario de sesión de cada runtime (plugins de OpenCode, preflight de Claude) no se toca.

### 2.6 DD-06 — Lista cerrada de rutas de documentación: `internal/docpaths`

**Elección.** Un paquete hoja y puro (sin git ni ficheros) que fija S6:

```go
package docpaths

// Set es la lista cerrada de rutas de documentación de un cambio en un repositorio (S6,
// REQ-27.5). Todas las rutas son relativas a la raíz git, con «/».
type Set struct { /* planningRel, change, baseline, maintained, reference */ }

// New compone el conjunto. planningRel es el planning home relativo a la raíz git ("" si es la
// raíz); baseline, maintained y reference son las rutas declaradas en axiom.yaml, ya relativas a la
// raíz git (las que caen fuera de ella se descartan antes de llamar).
func New(planningRel, change string, baseline, maintained, reference []string) (Set, error)

// Contains dice si rel es documentación del cambio. Toda ruta que no lo es es código.
func (s Set) Contains(rel string) bool

// Root devuelve la raíz de pathspec que cubre rel (la carpeta del cambio, la carpeta archivada,
// `<p>/specs`, `<p>/INDEX.md`, el documento o la ruta mantenida) y false si rel no es documentación.
func (s Set) Root(rel string) (string, bool)

// LocalState dice si rel es estado local de la carpeta del cambio que nunca se versiona: un
// fichero de su raíz cuyo nombre empieza por punto (`.gates.lock`, `.axiom-instance`, temporales).
func (s Set) LocalState(rel string) bool
```

Reglas de `Contains`, con `<p>` = `planningRel`:

| Raíz | Coincidencia |
|---|---|
| `<p>/changes/<x>` | prefijo `<p>/changes/<x>/` |
| carpeta archivada | `^<p>/changes/archive/\d{4}-\d{2}-\d{2}-<x>/` (el nombre del cambio va escapado: `foo-x` no coincide con `x`) |
| `<p>/specs` | prefijo `<p>/specs/` |
| `<p>/INDEX.md` | igualdad exacta |
| `baseline.documents[].path` | igualdad exacta |
| `technical_context.maintained.paths[]` | igualdad o prefijo `<ruta>/` |
| `technical_context.reference.paths[]` | **nunca**: una ruta bajo `reference` es código aunque coincida con otra regla |
| `odd/tasks/**` | no está en la lista: es código |

Las comparaciones distinguen mayúsculas, como git. `change` se valida con `branch.ValidateChangeName` antes de llamar.

**Alternativas descartadas.** Dejar la lista dentro de `sddpublish`: mezcla la política de rutas, que se prueba en tabla sin git, con el adaptador que ejecuta git. Ponerla en `internal/branch`: `sddpublish` tendría que importar el dominio de ramas solo por esto, y `HandOff` ya no la necesita (DD-12 retira su comprobación).

**Motivo.** Una sola definición, comprobable en tabla, y sin dependencias en ninguna dirección problemática.

### 2.7 DD-07 — Destino de la publicación: `Engine.PublicationTarget`

**Elección.** Es una lectura nueva del motor de ramas, en `internal/branch/engine/publication.go`, construida sobre `gather` + `branch.Plan` (y `useArchivePlan` con `Archive`), con la misma disciplina que `Status` y `HandOff`: sin cerrojo, sin registro y sin `fetch`.

```go
// internal/branch/publication.go
type PublicationRequest struct {
	CWD     string // workspaceRoot resuelto por la CLI
	Change  string
	Archive bool   // plan de archivado (rama `-archive` o integración in situ)
	Branch  string // opcional: la rama que autorizó la persona; si no coincide, wrong_branch
}

type PublicationTarget struct {
	Repository        string // nombre del repositorio del planning home
	GitRoot           string // raíz git absoluta del worktree donde vive la carpeta del cambio
	Branch            string // rama que se publica: la actual, que el plan permite
	IntegrationBranch string
	InPlace           bool   // repositorio de specs de un multirepo (o archivado in situ)
	PRTarget          string // BranchPlan.PRTarget de Branch, el mismo `prTarget` de `axiom branch status`; vacío si InPlace
	Topology          workspace.TopologyType
	PlanningRel       string // planning home relativo a GitRoot, con «/»
}

func (e *Engine) PublicationTarget(ctx context.Context, req branch.PublicationRequest) (branch.PublicationTarget, *branch.Refusal)
```

Reglas de rama (S9 y REQ-27.7), en este orden, con la rama actual leída con `gitcommon.ReadHead` del repositorio del planning home:

1. HEAD desacoplado → `detached_head` (código nuevo de `branch`).
2. `InPlace` (repositorio de specs de un `multirepo`, o archivado in situ): la rama permitida es `IntegrationBranch`. Si la actual es otra → `wrong_branch`.
3. Si no: si la actual es `IntegrationBranch` → `integration_branch` (código nuevo). Si no es ninguna de las ramas planificadas del repositorio (cambio y rol; con `Archive`, solo la de archivado) → `wrong_branch`, con la continuación de `ensure` que ya usa `HandOffWrongBranch`. Una rama de archivado sin `--archive`, o una rama del cambio con `--archive`, también es `wrong_branch`, y la continuación nombra el flag que falta.
4. Con `req.Branch` no vacío y distinto de la rama permitida → `wrong_branch`. Así la publicación queda atada a la rama que nombró la autorización (DD-11).

**Alternativas descartadas.** Que `sddpublish` replanifique por su cuenta, porque duplicaría el planificador. Leer solo la instantánea `git` del kickoff, que no existe en kickoffs retrosellados y no conoce la rama actual.

**Motivo.** La pregunta de la puerta (DD-11), `gate record --json` y `publish` consultan la misma función, así que la rama que se nombra es exactamente la que se sube. Es una lectura: el invariante «`axiom branch` no publica» sigue intacto.

### 2.8 DD-08 — Puerto de publicación: `internal/sddpublish`

**Elección.** Un paquete nuevo con su propio puerto. Ni `branch.Git` ni el `gitWriter` del motor ganan métodos que publiquen.

```go
package sddpublish

// Git son las operaciones de git de la publicación. dir es siempre la raíz git del destino
// (PublicationTarget.GitRoot), nunca el directorio del proceso. Es el único puerto del producto con
// Commit y Push (REQ-27.11).
type Git interface {
	RemoteConfigured(ctx context.Context, dir, remote string) (bool, error)             // git config --get remote.<r>.url
	ResolveCommit(ctx context.Context, dir, ref string) (string, bool, error)             // git rev-parse --verify --quiet <ref>^{commit}
	DirtyPaths(ctx context.Context, dir string, pathspecs []string) ([]string, error)     // git status --porcelain=v1 -z --untracked-files=all --no-renames -- <pathspecs>
	IdentityConfigured(ctx context.Context, dir string) (bool, error)                     // git var GIT_AUTHOR_IDENT y GIT_COMMITTER_IDENT
	HookPresent(ctx context.Context, dir, hook string) (bool, error)                      // git rev-parse --git-path hooks/<hook> + stat
	UnpublishedCommits(ctx context.Context, dir, head string, exclude []string, limit int) ([]string, error) // git rev-list --reverse --max-count=<limit+1> <head> ^<ref>...
	CommitPaths(ctx context.Context, dir, sha string) ([]string, error)                   // git diff-tree -r --no-commit-id --name-only -z --no-renames --root --cc <sha>
	Commit(ctx context.Context, dir, message string, pathspecs []string) (string, error)  // git add --all -- <ps>; git commit --only -m <msg> -- <ps>; rev-parse HEAD
	Push(ctx context.Context, dir, remote, branch string, setUpstream bool) (PushOutcome, error) // git push --porcelain [--set-upstream] origin refs/heads/<b>:refs/heads/<b>
}
```

- El adaptador (`gitport.go`) ejecuta todo con `gitcommon.Run`, salvo `Push`, que usa `gitcommon.RunWithTimeout(ctx, dir, pushTimeout, …)` con `pushTimeout = 120 * time.Second`. Es una variable de paquete para que las pruebas la reduzcan.
- `RunWithTimeout` se añade a `internal/gitcommon/exec.go` (el único punto de ejecución), y `Run` delega en ella con `runTimeout`. `Observe` sigue viendo todas las invocaciones.
- Toda referencia y rama pasan por `gitref.ValidRefName`, y las rutas se citan con la magia `:(top,literal)`. Ningún argumento empieza por `-`, y se usa `--end-of-options` donde git lo admite (`rev-list`, `rev-parse`).
- `gitEnvironment` ya fija `GIT_TERMINAL_PROMPT=0` y `LC_ALL=C`: un push que necesita credenciales interactivas falla y se clasifica como `remote_unavailable`, sin colgarse.

**Alternativas descartadas.** Añadir `Push` a `branch.Git` o al `gitWriter`: rompe el invariante de REQ-25.1 y la prueba `TestEnsureArchiveNeverPublishes` perdería su sentido. Un subproceso `axiom branch publish`: vuelve a colocar la publicación bajo `branch`.

**Motivo.** El aislamiento se puede verificar con importaciones (DD-14) y la ejecución sigue pasando por el único punto auditado.

### 2.9 DD-09 — Algoritmo de `publish`

**Elección.** `sddpublish.Publisher.Run(ctx, Request) Result`, en este orden. Los pasos 1 a 6 solo leen, y una negativa en ellos no escribe nada:

1. **Almacén**: el declarado (`workspace.DeclaredArtifactStore`) `engram` o `none` → `store_without_files` (REQ-27.10). Sin declaración cuenta como `openspec`, como en `prepareHandOff`.
2. **Cambio**: con `Archive` falso, `openspec/changes/<x>/` existe. Con `Archive`, existe la carpeta archivada `<p>/changes/archive/<fecha>-<x>/` o hay entradas sucias bajo `<p>/changes/<x>/`. Si no → `change_not_found`.
3. **Destino**: `PublicationTarget` (DD-07). Sus negativas se reenvían con su código (`detached_head`, `wrong_branch`, `integration_branch`, `config_invalid`, `not_a_git_repository`, `git_failed`).
4. **Remoto**: `RemoteConfigured(origin)` falso → `remote_missing`. Va antes del commit a propósito: sin `origin` no se crea ningún commit.
5. **Rango sin subir**: la cabeza es `HEAD`. La referencia superior es `refs/remotes/origin/<rama>` si existe y, si no, `refs/remotes/origin/<integración>` y, si no, `refs/heads/<integración>` (REQ-27.8). Si no existe ninguna → `unpublished_range_unknown`. Cuando existe `refs/remotes/origin/<integración>`, se excluye también: lo que ya está en la integración publicada no es trabajo del cambio. Esto evita negativas falsas tras fusionar `main` en la rama.
6. **Conjunto de documentación**: `docpaths.New` con `PlanningRel`, el cambio y las rutas de `baseline` y `technical_context` de `axiom.yaml`, convertidas de relativas a `workspaceRoot` a relativas a `GitRoot`. `DirtyPaths` sobre las raíces candidatas filtra lo que `Contains` acepta y descarta `LocalState`.
7. **Commit** (solo si hay rutas sucias de documentación):
   - Si `IdentityConfigured` es falso → `identity_missing`, sin escribir.
   - Si no, `Commit` con las **raíces** que cubren esas rutas (`Root`), más la exclusión `:(top,exclude,glob)<p>/changes/<x>/.*`, en dos pasos: `git add --all -- <ps>`, que solo toca el índice de esas rutas e incluye borrados (archivado), y `git commit --only -m <msg> -- <ps>`, que confirma solo esas rutas y deja preparado lo demás.
   - Mensaje: `docs(sdd): publicar los documentos de <x>`, o `docs(sdd): publicar el archivado de <x>` con `Archive`.
   - Si falla: con un hook `pre-commit`, `prepare-commit-msg` o `commit-msg` presente → `commit_hook_rejected`; si no, `commit_failed` con la cola de `stderr`. Si `git add` ya preparó las rutas y el commit falla, quedan preparadas: es el estado que deja git y la negativa lo dice.
8. **Commits sin subir**: `UnpublishedCommits(HEAD, excluye: [superior, origin/<integración>?], límite 500)`. Con más de 500 → `unpublished_range_too_large`. Para cada commit, `CommitPaths`: en un commit de fusión, `--cc` deja solo lo que la fusión resolvió por su cuenta. Si alguna ruta no es documentación → `code_commits_unpublished`, con `details.commits` (SHA cortos), `details.paths` (hasta 20 y el recuento) y la continuación `git -C <raíz> push origin <rama>:<rama>`. El commit del paso 7, si se creó, se conserva y se informa.
9. **Nada que subir**: cero commits y ningún commit en el paso 7 → `up_to_date` (código de salida 0, sin push).
10. **Push**: `git push --porcelain [--set-upstream] origin refs/heads/<b>:refs/heads/<b>`, sin `+`, sin `--force*` y sin `--no-verify`. `--set-upstream` solo fuera de `InPlace`. La clasificación del resultado:

| Señal (salida `--porcelain` y `stderr` con `LC_ALL=C`) | Código |
|---|---|
| línea `!` con `[rejected]` y `(non-fast-forward)` o `(fetch first)` | `push_non_fast_forward` |
| línea `!` con `[remote rejected]` | `push_hook_rejected` (hook del servidor o rama protegida) |
| sin línea de estado de ref y con un hook `pre-push` local presente | `push_hook_rejected` |
| salida 128 con `Could not read from remote repository`, `does not appear to be a git repository`, `unable to access`, `Authentication failed` o `could not read Username` | `remote_unavailable` |
| superado `pushTimeout` | `remote_unavailable` (con `details.timeout`) |
| cualquier otro fallo | `push_failed` con la cola de `stderr` |

Ninguna negativa reintenta, hace `fetch` ni fuerza. Tras un push correcto se devuelve el SHA subido (`rev-parse HEAD`) y si la rama remota se creó (no existía `refs/remotes/origin/<b>` antes).

**Alternativas descartadas.**

- Pasar al commit los ficheros sueltos: con muchos ficheros (archivado) supera el límite de línea de órdenes de Windows, y `--pathspec-from-file` exige `stdin`, que `gitcommon.Run` no ofrece.
- Comprobar el código sin subir antes de confirmar: REQ-27.8 pide conservar el commit de documentación, y así el push del humano lo incluye.
- Usar `--no-verify`: lo prohíbe REQ-27.6.

**Motivo.** El orden deja todas las negativas sin efectos antes del commit, y después del commit solo quedan dos efectos, el commit y el push, ambos informados. Excluir el estado local y las rutas de `reference` evita confirmar ficheros que nunca deben versionarse, cosa que hoy ocurriría porque `.gates.lock` y `.axiom-instance` no están en `.gitignore`.

### 2.10 DD-10 — Verbo `axiom sdd publish`

**Elección.** `internal/cli/sdd_publish.go` con `RunSDDPublish(args []string, stdout io.Writer) error`, registrado en `runSDD` (`cmd/axiom/main.go`) y en su ayuda y su lista de opciones.

```
Uso: axiom sdd publish --cwd <ruta> --change <nombre> [--archive] [--branch <rama>] [--json]
```

- `--change` es obligatorio. Sin él, vacío o con un nombre inválido, es error de uso sin efectos (salida 1, como el resto de `axiom sdd`).
- `--cwd` se resuelve con `workspace.Locate`, como en `gate record`.
- `--archive` pide el plan de archivado y es **obligatorio** para publicar la rama de archivado `<x>-archive` (o la integración in situ tras archivar en `multirepo`): sin él, esa rama es `wrong_branch` y la continuación nombra el flag. `--branch` ata la publicación a la rama autorizada (DD-07, regla 4).
- La guarda de escritura de la CLI (`guardChangeWriter`) se aplica con `ContextOrdinary` o `ContextArchive`, como en `gate record`.
- La salida es 0 con `published` o `up_to_date` y 1 con `refused`. Con `--json`, el documento se escribe en `stdout` también en una negativa, y la función devuelve `errPublishRefused` (un error con la línea resumen) para que `runSDD` salga con 1.

Esquema `axiom.sdd-publish/v1`, en §5.3. Texto en castellano:

```
Publicación de <x> en origin/<rama> (repositorio <nombre>):
  Commit <sha corto>: docs(sdd): publicar los documentos de <x> (<n> rutas)
  Push: refs/heads/<rama> -> origin (rama creada)
Quien siga: `git fetch origin` y `axiom branch ensure --change <x> --json`.
```

Con una negativa, la etiqueta, el mensaje y la continuación; si hubo commit, una línea «el commit <sha> se conserva en local».

**Alternativas descartadas.** Un subcomando de `axiom branch`, que rompe REQ-25.1. Un flag en `gate record` (B2 de la exploración).

**Motivo.** El verbo vive donde vive la decisión que lo autoriza (`axiom sdd`), y su salida tipada permite a la doctrina tratar las negativas como informativas.

### 2.11 DD-11 — Aprobar publica: sobre de la puerta, `gate record --json` y doctrina

**Elección.**

- **Sobre** (`SDDGovernanceGateResult`, `governance.go`): campo nuevo `Publication *GatePublication \`json:"publication,omitempty"\``.

```go
type GatePublication struct {
	Remote     string `json:"remote"`             // siempre "origin"
	Branch     string `json:"branch"`             // PublicationTarget.Branch
	PRTarget   string `json:"prTarget,omitempty"` // prTarget de la rama en el plan (`axiom branch status`); vacío en la integración in situ de multirepo
	Invocation string `json:"invocation"`         // axiom sdd publish --cwd <q> --change <x> --branch <b>
}
```

- Presente en `spec`, `design`, `tasks` y `role-apply:<rol>` cuando el almacén tiene ficheros y `PublicationTarget` responde. Si no responde (por ejemplo `wrong_branch`), el sobre lleva `publicationUnavailable: "<código>"` y las opciones siguen siendo las de `gate record`. La puerta `integration` no cambia.
- `Validate` exige que `Invocation` empiece por `axiom sdd publish ` y lleve `--branch <Branch>`. Las invocaciones de las opciones siguen siendo `axiom sdd gate record …` (prefijo validado, sin cambios).
- `newGovernanceGateQuestion` llama a `branchEngine()`/`engine.Default()`. Con un motor nulo (pruebas antiguas), `publication` se omite.
- **Etiquetas y efectos** (`gateQuestionChoices`):
  - `rejected`: «Iterate on this artifact», sin cambios.
  - `hand_off`: «Approve, publish and stop». Efecto: «Records an approved decision for gate %q marked as handed off, then publishes the change documents: `axiom sdd publish` commits only the documentation paths of the change and pushes `%s` to `origin` without forcing, using the git credentials already configured. Whoever continues runs `git fetch origin` and `axiom branch ensure`; no merged pull request is needed. A refused publication keeps the approval. Code pushes, pull requests and merges remain yours.»
  - `approved`: «Approve, publish and continue». Efecto: «Records an approved decision for gate %q, publishes the change documents to `origin/%s` with `axiom sdd publish`, and starts the next phase in this session. A refused publication keeps the approval. Code pushes, pull requests and merges remain yours.»
  - Con almacén `engram` o `none`, `approved` mantiene «Approve and continue» sin publicación y `hand_off` no existe, como hoy.
  - `gateQuestionNarrative` actualiza `value` con el remoto y la rama, y con la misma frase final (REQ-27.12).
  - `prTarget` del sobre es informativo: la PR hacia ese destino sigue siendo decisión de la persona.
- **`gate record --json`** (`gateRecordJSON`) gana `Publish *gatePublishJSON \`json:"publish,omitempty"\`` (la misma forma que `GatePublication`) y `PublishUnavailable string \`json:"publishUnavailable,omitempty"\``. Aparecen en toda aprobación de una puerta distinta de `integration`. `gate record` **calcula** la orden y no la ejecuta (REQ-26.5 reescrito).
- **Doctrina** (`sdd-orchestrator-sections.md`, sección de kickoff y puertas): «Approve, publish and stop is `--decision approved --hand-off`; approve, publish and continue is `--decision approved`. After either, run the `publish.invocation` the record returns (the same as `gateQuestion.publication.invocation`) exactly once. A refused publication never undoes the recorded approval: present its label, message and continuation as information, and keep going (exit 3) or stop (exit 2).» Se elimina «it refuses without writing anything when the artifacts are not committed».
- **`stop-exits.md`** (texto final, solo SDD):

  > **Stops and exits.** A stop is a point where the orchestrator presents what it just produced and waits for a human decision before any further work: every SDD block-review gate except integration (spec, design, tasks, and each role-apply) of a `checkpointed` change. A `continuous` change has no stops. Every stop offers exactly three exits, always in this order:
  >
  > 1. **Iterate:** keep working on the current artifact and present the stop again. There is no limit to the iterations.
  > 2. **Approve, publish and stop:** approve the artifact with `axiom sdd gate record --hand-off`, publish the change documents, hand the next phase to another person, and stop. The session never starts the next phase.
  > 3. **Approve, publish and continue:** approve the artifact with `axiom sdd gate record`, publish the change documents, and continue yourself with the next phase.
  >
  > "Publish" in exits 2 and 3 is `axiom sdd publish`, run once right after the record with the invocation it returns: it commits only the documentation paths of the change and pushes only the change branch to `origin` (in the specs repository of a `multirepo`, its integration branch), never forced. The stop names that remote and branch, so choosing exit 2 or 3 is the explicit authorization of that push with the git credentials already configured, and of nothing else. A refused publication, such as `code_commits_unpublished` after a role-apply, never undoes the approval: present it as information, with the push command it returns for the human. Whoever continues runs `git fetch origin` and `axiom branch ensure --change <name>`, which adopts the published branch; no merged pull request is needed.
  >
  > Pushing code, `fetch`, opening a pull request, and merging still belong to the human: present the exact commands and data, and never run them yourself. With the `engram` or `none` artifact store the artifacts are not files: exit 2 is not available and exit 3 publishes nothing, so offer only exits 1 and 3. The person who picks up a handed-off phase starts the next phase without reviewing the previous one again.

**Alternativas descartadas.** Ejecutar `publish` dentro de `gate record` (rompe REQ-26.5 y mezcla decisión y red). Nombrar la rama solo en prosa (el relevo sin pérdidas exige el dato tipado).

**Motivo.** La autorización, el texto de la pregunta, nombra exactamente lo que ejecuta la orden atada con `--branch`, y eso cumple la regla de operaciones remotas (destino, operación y credencial). Una negativa no deshace nada porque el registro ya está escrito y es terminal.

### 2.12 DD-12 — `gate record --hand-off` relajado y entrega nueva

**Elección.**

- `Engine.HandOff` deja de comprobar si algo está confirmado, en todas las puertas: REQ-26.4 prohíbe que `gate record` inspeccione el trabajo del rol. Ya no lee `DirtyPaths`. El código sin subir de un `role-apply` lo detecta después `publish`, como negativa informativa `code_commits_unpublished` (DD-09, paso 8); el código que ni siquiera está en un commit no viaja en ningún push y no es asunto de la cesión.
- Se retiran, sin otro uso en el producto (verificado con `rg`): `branch.UncommittedArtifacts`, `CodeUncommittedArtifacts` (`uncommitted_artifacts`) y su etiqueta en `labels.go`, y en `engine/handoff.go` `uncommittedArtifacts`, `exemptEntry` y la lectura de `DirtyPaths` de `codeDelivery`.
- `HandOffRequest.Exempt` desaparece: `gates.yaml` es documentación.
- `HandOffPlan` (`internal/branch/handoff.go`) queda así:

```go
type HandOffPlan struct {
	Topology   workspace.TopologyType `json:"topology"`
	Publish    HandOffPublish         `json:"publish"`              // destino de los documentos: lo sube `axiom sdd publish`
	Deliveries []HandOffDelivery      `json:"deliveries,omitempty"` // solo repositorios de código del rol: push del humano
	Resume     []string               `json:"resume"`               // lo que ejecuta quien siga
}

type HandOffPublish struct {
	Remote     string `json:"remote"`
	Branch     string `json:"branch"`
	PRTarget   string `json:"prTarget,omitempty"` // prTarget del plan; vacío en la integración in situ de multirepo
	InPlace    bool   `json:"inPlace,omitempty"`
	Invocation string `json:"invocation"`
}
```

- Se eliminan `MergeRequired`, `MergeNotice`, `InPlaceNotice`, `HandOffAddStep`, `HandOffCommitStep` y `HandOffCommitMessage`, con sus pruebas. Se conservan `HandOffPushStep` (entregas de código) y `HandOffWrongBranch`. `HandOffDelivery` pierde `PR` (la cesión no exige PR) y conserva `PRTarget`, informativo.
- `Resume = ["git fetch origin", "axiom branch ensure --change <x> --json"]`. No llevan `-C`, porque la ruta de quien siga no se conoce.
- `writeHandOffDelivery` escribe: «Cesión registrada. Siguiente paso: publica los documentos con `<invocation>` (rama `<branch>` en `origin`; destino de su PR: `<prTarget>`). Quien siga ejecuta `git fetch origin` y `axiom branch ensure --change <x>`: `ensure` adopta la rama publicada; no hace falta una PR fusionada.» Con `prTarget` vacío (integración in situ de multirepo) dice «se trabaja in situ en la integración, sin PR». Si hay entregas de código, añade sus órdenes de push numeradas, el `prTarget` de cada una y «Axiom no sube código ni abre PR: ejecútalo tú» (REQ-26.5).
- `requireHandOffStops`, la comprobación del almacén, la del artefacto presente, la de la puerta sin resolver y `AppendGateChecked` no cambian.

**Alternativas descartadas.** Mantener `uncommitted_artifacts` para la carpeta del cambio: contradice D6, porque `publish` confirma esos ficheros. Mantenerlo para el código del rol: REQ-26.4 prohíbe comprobar el trabajo del rol en `gate record`, y `publish` ya lo informa sin bloquear la aprobación.

**Motivo.** La salida 2 ya no entrega órdenes `git` de documentación al humano; entrega la orden de publicación y la reanudación. El JSON `axiom.sdd-gate-record/v1` pierde `delivery.mergeRequired`. Su único consumidor es la doctrina, que se sincroniza con el binario. La retirada queda en el contrato y en la nota de versión (§8).

### 2.13 DD-13 — Adopción de la rama publicada en `axiom branch ensure`

**Elección.**

- En `checkRepository` y `checkInPlaceRepository` (`engine/ensure.go`), cuando una rama planificada de tipo `change`, `role` o `archive` no existe en local, el motor resuelve `refs/remotes/origin/<nombre>` con `ResolveCommit`, que lee ficheros de referencias locales.
  - Si existe: `step.createFrom = "refs/remotes/origin/<nombre>"` y `step.adopt = true`.
  - Si no: la regla actual (la base para la del cambio y la de archivado, `refs/heads/<from>` para la de rol).
- La rama de archivado `<x>-archive` también adopta `refs/remotes/origin/<x>-archive` (REQ-27.15), para que otra persona continúe un archivado ya publicado. `checkIntegrated` no cambia: comprueba que la **base** de integración contiene el cambio fusionado, una condición previa que no depende del punto de partida de la rama, y se sigue ejecutando antes (con `--fetch`, sobre la base recién traída). Si la base no contiene el cambio, `ensure --archive` rechaza igual que hoy aunque exista la rama remota.
- Con `--fetch` (autorizado), el `fetch` va antes de esta comprobación (`checkRepository` ya lo ordena así), de modo que `ensure --fetch` adopta en la misma ejecución. Sin `--fetch` no hay red.
- `gitWriter` (`engine/engine.go`) y `gitcli.Git` ganan `CreateTrackingBranch(ctx, dir, name, remoteRef string) error`, que ejecuta `git branch --track <name> <remoteRef>`. Exige `remoteRef == "refs/remotes/origin/" + name` y `plainArgument` en ambos. Configurar el *upstream* es una escritura local de `.git/config`, no una operación remota.
- El comentario del puerto se actualiza: «Fetch sigue siendo la única operación remota; ningún método publica».
- `BranchResult` (`internal/branch/result.go`) gana dos campos, solo en el resultado de `ensure` y solo para las ramas que creó esa ejecución:

```go
CreatedFrom    string `json:"createdFrom,omitempty"`    // base | change_branch | remote
CreatedFromRef string `json:"createdFromRef,omitempty"` // la referencia de partida
```

- `BaseResult.Source` no cambia: describe la base de integración del repositorio (`remote-tracking` o `local`), y mezclarlo con el origen de una rama rompería su significado y sus pines.
- Texto de `ensure`: «Rama <b> adoptada desde origin/<b> (seguimiento configurado)». Si una rama del cambio se crea desde la base sin `--fetch`, sale una línea informativa: «Rama <b> creada desde la base: no había `origin/<b>` en local; si otra persona ya la publicó, borra esta rama local, ejecuta `git fetch origin` (o `--fetch` con autorización) y repite `ensure`». REQ-25.7 prohíbe mover una rama existente, así que la reparación es manual.

**Alternativas descartadas.** `base.source: remote_branch` (sugerencia de la propuesta): confunde la base del repositorio con el origen de una rama. `ensure --adopt-remote` con *fetch* propio (C2): añade una operación remota a `branch`.

**Motivo.** Es una adopción sin red y sin superficie nueva de flags. El motor ya resolvía commits sobre referencias remotas locales.

### 2.14 DD-14 — Aislamiento de la operación remota

**Elección.** Tres pruebas complementarias:

1. **Arquitectura (estática, ampliada).** `TestArchitectureOnlyExecGoLaunchesProcesses` (`internal/gitcommon/architecture_test.go`) recorre también `internal/sddpublish` y `internal/docpaths`: solo `exec.go` lanza procesos o abre la red.
2. **Importaciones (estática, nueva).** `internal/sddpublish/isolation_test.go` recorre con `go/parser` los `.go` de producción de `internal/` y `cmd/`.
   - El único paquete que importa `internal/sddpublish` es `internal/cli`, y dentro de él el único fichero que usa el selector `sddpublish.` es `sdd_publish.go`.
   - `internal/branch/**` y `internal/docpaths` no importan `sddpublish`.
   - Ningún fichero de producción de `internal/branch/**` contiene el literal `"push"` ni `"commit"` como argumento de `gitcommon.Run`. Esto se comprueba con `ast.CallExpr` sobre `gitcommon.Run`/`RunWithTimeout` y `runWrite`/`runRead`.
3. **Observador (dinámica, nueva y existentes).** `internal/cli/sdd_publish_isolation_test.go` instala `gitcommon.Observe` y ejecuta, sobre un proyecto con remoto *bare*:
   - `RunBranch ensure|status|done`, con y sin `--archive`;
   - `RunSDDGate record` con `--decision approved`, con `--hand-off` y con `rejected`;
   - `RunSDDStatus` con la puerta abierta (que calcula `publication`).

   Ninguno ejecuta `push` ni `commit`, ni `fetch` sin `--fetch`. `TestEnsureArchiveNeverPublishes` y la prueba de `status` sin `fetch` siguen tal cual.

**Alternativa descartada.** Solo la prueba de arquitectura existente: permite el único punto de ejecución pero no distingue quién llama a `push`.

**Motivo.** La regla «solo `axiom sdd publish` alcanza el puerto» (REQ-27.11) queda fijada por la estructura (importaciones) y por el comportamiento (observador).

### 2.15 DD-15 — Momentos de publicación: `continuous` y archivado

**Elección.**

- **`checkpointed`.** Publica tras cada aprobación de `spec`, `design`, `tasks` y `role-apply:<rol>` (DD-11).
- **`continuous` con `docs_publication: auto`.** El orquestador ejecuta, sin preguntar:
  - `axiom sdd publish --cwd <root> --change <x> --json` una vez, cuando `tasks` está hecho y antes de lanzar `apply`;
  - `axiom sdd publish --cwd <archive-root> --change <x> --archive --json` una vez, cuando la fase `archive` termina, en el worktree de archivado de `ensure --archive` (en `multirepo`, el repositorio de specs in situ).
- **`continuous` sin el campo, o `checkpointed` al archivar.** El orquestador ofrece esas mismas órdenes como siguiente paso y espera la decisión de la persona.
- **Negativas.** Siempre informativas: se presentan y el encadenado continúa (REQ-27.13).
- **Contenido del archivado (S5).** La carpeta archivada, los borrados de `<p>/changes/<x>/`, `<p>/specs/**`, `<p>/INDEX.md` y las rutas de baseline y de contexto mantenido declaradas: todo lo que `docpaths` acepta y está sucio.
- **Lugar en la doctrina.** Todo esto vive en la sección «SDD Change Kickoff and Block Gates» de `sdd-orchestrator-sections.md`. Las skills de fase no publican. En `sdd-spec/SKILL.md` y `sdd-design/SKILL.md`, la regla «Antes de la salida 2 de la parada, los artefactos quedan confirmados en la rama del cambio; publicarla es decisión de la persona» se sustituye por «La fase no hace commit ni push: tras la aprobación de la parada el orquestador publica los documentos con `axiom sdd publish`; el push de código y la PR son de la persona» (pin en `assets_test.go`).

**Alternativa descartada.** Detectar el archivado por el nombre de la rama actual: obliga a planificar dos veces y oculta el contexto. El flag explícito replica `ensure --archive`.

**Motivo.** Se cumplen S4 y S5 sin paradas nuevas en `continuous` (REQ-21.7).

### 2.16 DD-16 — Entrega y regeneración

**Elección.** Una sola PR con `size:exception` (estrategia `exception-ok`) y cuatro commits de unidad de trabajo. Se recomienda el orden **1 → 2 → 4 → 3**, que D8 permite («(4) puede adelantarse a (3)»): así ningún commit deja una doctrina que promete una adopción que aún no existe.

| Commit | Contenido | Depende de |
|---|---|---|
| 1 `feat(workflow)` | DD-01 a DD-04: bloque, proyección, manifiesto, paso 0 y retirada de la parada ODD; `stop-exits.md` sin ODD; `axiom.yaml` con `ask`; plantilla, manual y goldens | — |
| 2 `feat(sdd-publish)` | DD-06 a DD-10, DD-14: `docpaths`, `PublicationTarget`, `sddpublish`, `RunWithTimeout`, verbo y pruebas | — |
| 4 `feat(branch)` | DD-13: adopción en `ensure` | — |
| 3 `feat(sdd-gates)` | DD-05, DD-11, DD-12, DD-15: kickoff, sobre, `gate record`, doctrina de puertas y `stop-exits.md` final, skills y goldens | 2 y 4 |

Regeneración, siempre sin binarios sueltos (el antivirus Cortex los borra):

- Goldens: `go test ./internal/components -run Golden -update` (solo el paquete `internal/components`, sin subpaquetes que compilan binarios).
- `.codex/AGENTS.md`: se actualiza el bloque gestionado con los mismos bytes del golden regenerado `sdd-codex-agentsmd.golden`. No se ejecuta `axiom sync`.
- `AGENTS.md` y `GEMINI.md`: se retira la mención de la parada de plan. `docs/manual-de-inicio.md`: carril, publicación, adopción y la nota de CI ajeno.

**Motivo.** Cada commit compila, pasa sus pruebas y describe un comportamiento real.

## 3. Flujo de datos

### 3.1 Carril

```
axiom.yaml ──parseLayered──▶ Declarations.Workflow / WorkflowErr
                                   │
workspace.Locate ──▶ Location ─────┘
                                   │ decorateWorkspaceDeclarations
                                   ▼
                    Status.Workflow {lane, source} ──▶ JSON v1 / v2
                                   │
               doctrina, paso 0 ◀──┘  (status inalcanzable ⇒ ask)
```

### 3.2 Aprobar y publicar (`checkpointed`)

```
status (await-gate)
  └─ gateQuestion ──── PublicationTarget (lectura) ──▶ publication{origin, rama, invocation}
         │
   persona elige salida 2 o 3  (autorización: nombra origin/<rama>)
         │
  axiom sdd gate record ... [--hand-off] --json
         │   escribe gates.yaml; nunca git commit/push
         ▼
   { publish: {invocation} }
         │
  axiom sdd publish --cwd R --change x --branch b --json
         │
  1-6 lecturas ─▶ 7 commit (docs) ─▶ 8 commits sin subir ─▶ 10 push refs/heads/b:refs/heads/b
         │                         └─ código ⇒ code_commits_unpublished (informativa)
         ▼
  JSON axiom.sdd-publish/v1 ──▶ orquestador: sigue (3) o se detiene (2)
```

### 3.3 Continuación por otra persona

```
máquina B:  git fetch origin            (decisión de la persona)
            axiom branch ensure --change x --json
               │  rama local ausente + refs/remotes/origin/b presente
               ▼
            git branch --track b refs/remotes/origin/b ──▶ worktree
            JSON: branches[].createdFrom = "remote"
```

## 4. Cambios de ficheros

| Fichero | Acción | Commit | Descripción |
|---|---|---|---|
| `internal/workspace/workflow.go` | Crear | 1 | Tipo, constantes, `decodeWorkflow`, `resolveWorkflow` |
| `internal/workspace/declarations.go` | Modificar | 1 | `Workflow`/`WorkflowErr`; lectura en `readDeclarationBlocks` |
| `internal/workspace/loader.go` | Modificar | 1 | `joinLoadErrors`, proyección en `parseLayered`, comentarios |
| `internal/workspace/types.go` | Modificar | 1 | `WorkspaceConfig.Workflow` |
| `internal/workspace/workflow_test.go`, `docs_contract_test.go` | Crear / Modificar | 1 | Decodificación y pin del manual y la plantilla |
| `internal/hub/init.go` | Modificar | 1 | `declares()` cuenta `workflow` |
| `axiom.example.yaml`, `axiom.yaml`, `docs/manual-de-inicio.md` | Modificar | 1 | Plantilla, `lane: ask` y manual |
| `internal/sddstatus/status.go`, `status_v2.go`, `workspace_declarations.go` | Modificar | 1 | `WorkflowProjection` y `projectWorkflow` |
| `internal/sddstatus/workflow_status_test.go` | Crear | 1 | v1/v2, con y sin cambio, cuatro orígenes |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | 1, 3 | `workflow` (1); `publication`, `docsPublication` (3) |
| `internal/agents/capabilitymanifest/manifest.go`, `manifest_test.go` | Modificar | 1 | Literal nuevo |
| `internal/components/agentguidance/routing.go` | Modificar | 1 | Paso 0, retirada de la parada ODD y de `### Stops and exits` |
| `internal/components/agentguidance/routing_test.go`, `routing_stops_test.go`, `stop_exits_test.go`, `stop_exits.go` | Modificar | 1, 3 | Pines nuevos; comentario de `stop_exits.go` sin ODD |
| `internal/assets/generic/stop-exits.md` | Modificar | 1, 3 | Sin ODD (1); salidas con publicación (3) |
| `internal/cli/run_component_paths_test.go`, `sync_test.go` | Modificar | 1 | Nombres o pines de la política, si citan el literal |
| `AGENTS.md`, `GEMINI.md` | Modificar | 1 | Sin parada de plan |
| `testdata/golden/*` (13), `.codex/AGENTS.md` | Regenerar | 1, 3 | Doctrina |
| `internal/docpaths/docpaths.go`, `docpaths_test.go` | Crear | 2 | Lista cerrada S6 |
| `internal/branch/publication.go`, `engine/publication.go`, `engine/publication_test.go` | Crear | 2 | `PublicationTarget` |
| `internal/branch/types.go` | Modificar | 2 | Códigos `detached_head`, `integration_branch` |
| `internal/gitcommon/exec.go`, `exec_test.go`, `architecture_test.go` | Modificar | 2 | `RunWithTimeout`; arquitectura sobre `sddpublish` y `docpaths` |
| `internal/sddpublish/{publish.go,ports.go,gitport.go,classify.go,result.go}` | Crear | 2 | Puerto, adaptador, algoritmo y clasificación |
| `internal/sddpublish/*_test.go`, `isolation_test.go` | Crear | 2 | Remotos *bare* y aislamiento |
| `internal/cli/sdd_publish.go`, `sdd_publish_test.go`, `sdd_publish_isolation_test.go` | Crear | 2 | Verbo, JSON y observador |
| `cmd/axiom/main.go` | Modificar | 2 | Registro y ayuda de `publish` |
| `internal/branch/engine/ensure.go`, `engine/engine.go`, `gitcli/ops.go`, `result.go` | Modificar | 4 | Adopción, `CreateTrackingBranch`, `createdFrom` |
| `internal/branch/engine/ensure_adopt_test.go` | Crear | 4 | Adopción con remoto *bare* |
| `internal/kickoff/types.go`, `args.go`, `schema.go`, pruebas | Modificar | 3 | `docs_publication` y `--docs-publication` |
| `internal/sddstatus/governance.go`, `governance_handoff.go`, pruebas | Modificar | 3 | `publication`, etiquetas y efectos |
| `internal/branch/handoff.go`, `engine/handoff.go`, `labels.go`, `types.go`, pruebas | Modificar | 3 | `HandOffPlan` nuevo con `prTarget`; retirada de `uncommitted_artifacts` y de la lectura de `DirtyPaths` |
| `internal/cli/sdd_gate.go`, `sdd_gate_handoff.go`, `sdd_kickoff.go`, pruebas | Modificar | 3 | `publish` en el JSON y texto de la cesión; flag del kickoff |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | 1, 3 | «SDD lane selected» (1); encadenado y momentos (3) |
| `internal/assets/skills/sdd-spec/SKILL.md`, `sdd-design/SKILL.md`, `internal/assets/assets_test.go` | Modificar | 3 | La fase no publica |
| `internal/components/sdd/orchestrator_stop_exits_test.go` | Modificar | 3 | Pin del texto nuevo |

No se borra ningún fichero de producción. Las pruebas de `MergeNotice` e `InPlaceNotice` (`internal/branch/handoff_test.go`) se reescriben.

## 5. Interfaces y contratos

### 5.1 `axiom.yaml`

```yaml
# Carril del proyecto (REQ-27.1): odd | sdd | ask. Ausente: sdd.
workflow:
  lane: ask
```

### 5.2 `axiom sdd status --json` (v1 y v2)

```json
{
  "changeName": null,
  "workflow": { "lane": "ask", "source": "declared" },
  "governance": {
    "kickoff": { "executionStyle": "continuous", "docsPublication": "auto" }
  },
  "gateQuestion": {
    "gate": "design",
    "choices": ["…rejected…", "…hand_off…", "…approved…"],
    "publication": {
      "remote": "origin",
      "branch": "inc/x",
      "prTarget": "main",
      "invocation": "axiom sdd publish --cwd '<workspaceRoot>' --change x --branch inc/x"
    }
  }
}
```

En el ejemplo solo se muestran los campos nuevos de cada bloque. `publicationUnavailable` sustituye a `publication` cuando `PublicationTarget` responde con una negativa.

### 5.3 `axiom sdd publish --json` (`axiom.sdd-publish/v1`)

```json
{
  "schema": "axiom.sdd-publish/v1",
  "change": "x",
  "context": "change",
  "status": "published",
  "repository": { "name": "axiom", "gitRoot": "<gitRoot absoluta>" },
  "remote": "origin",
  "branch": "inc/x",
  "inPlace": false,
  "commit": {
    "sha": "…",
    "message": "docs(sdd): publicar los documentos de x",
    "paths": ["openspec/changes/x/design.md", "openspec/changes/x/gates.yaml"]
  },
  "push": { "refspec": "refs/heads/inc/x:refs/heads/inc/x", "sha": "…", "created": true },
  "unpublishedCommits": ["…"],
  "resume": ["git fetch origin", "axiom branch ensure --change x --json"],
  "refusal": null
}
```

- `context` vale `change` o `archive`, y `status` vale `published`, `up_to_date` o `refused`.
- `commit` y `push` se omiten cuando no ocurrieron.
- Con `refused`, `refusal` es `{code, label, message, continuation, details}`, y `commit` sigue presente si el commit local se creó.

Códigos de negativa (tokens estables en inglés con etiqueta en castellano):

| Código | Etiqueta | Escribe algo |
|---|---|---|
| `store_without_files` | almacén sin ficheros | no |
| `change_not_found` | cambio inexistente | no |
| `config_invalid`, `not_a_git_repository`, `git_failed` | reenviados de `branch` | no |
| `detached_head` | HEAD desacoplado | no |
| `wrong_branch` | rama distinta de la asignada (REQ-25.19) | no |
| `integration_branch` | rama de integración no publicable | no |
| `remote_missing` | no existe el remoto `origin` | no |
| `unpublished_range_unknown` | sin referencia para calcular lo no subido | no |
| `identity_missing` | identidad de git ausente | no |
| `commit_hook_rejected` | un hook rechazó el commit | no (rutas quizá preparadas) |
| `commit_failed` | el commit falló | no (rutas quizá preparadas) |
| `unpublished_range_too_large` | más de 500 commits sin subir | commit, si lo hubo |
| `code_commits_unpublished` | commits de código sin subir | commit, si lo hubo |
| `push_non_fast_forward` | el remoto avanzó | commit, si lo hubo |
| `push_hook_rejected` | un hook (local o remoto) rechazó el push | commit, si lo hubo |
| `remote_unavailable` | remoto inalcanzable o sin credenciales | commit, si lo hubo |
| `push_failed` | el push falló | commit, si lo hubo |

### 5.4 `axiom sdd gate record --json` (`axiom.sdd-gate-record/v1`, campos nuevos)

```json
{
  "decision": "approved",
  "handedOff": true,
  "publish": { "remote": "origin", "branch": "inc/x", "prTarget": "main", "invocation": "axiom sdd publish …" },
  "delivery": {
    "topology": "monorepo-embedded",
    "publish": { "remote": "origin", "branch": "inc/x", "prTarget": "main", "invocation": "axiom sdd publish …" },
    "resume": ["git fetch origin", "axiom branch ensure --change x --json"]
  }
}
```

Se retira `delivery.mergeRequired`, y `delivery.deliveries` solo lista repositorios de código (cada uno con su `prTarget`). `publish.prTarget` es el `prTarget` que da `axiom branch status` para la rama publicada y se omite en la integración in situ de `multirepo` (REQ-26.5); `writeHandOffDelivery` lo imprime.

### 5.5 `axiom branch ensure --json` (campos nuevos por rama)

```json
{ "name": "inc/x", "kind": "change", "exists": true, "createdFrom": "remote", "createdFromRef": "refs/remotes/origin/inc/x" }
```

### 5.6 `kickoff.yaml`

```yaml
kickoff:
    flow_mode: sdd
    execution_style: continuous
    docs_publication: auto   # solo continuous y solo si la persona lo autorizó al sellar
```

## 6. Estrategia de pruebas

Todas las pruebas se ejecutan en proceso con `go test` por paquete, con remotos *bare* temporales (`git init --bare` en `t.TempDir()`, siguiendo `remoteWithMain` de `engine/archive_test.go`). **Nunca** se compilan binarios ni se ejecuta `go test ./...` en local: varios paquetes ajenos compilan ejecutables. Paquetes a ejecutar: `./internal/workspace ./internal/hub ./internal/sddstatus ./internal/agents/capabilitymanifest ./internal/components/agentguidance ./internal/components/sdd ./internal/assets ./internal/kickoff ./internal/docpaths ./internal/sddpublish ./internal/gitcommon ./internal/branch/... ./internal/cli`, más `go test ./internal/components -run Golden` y `go vet ./...`. TDD está desactivado (`strict_tdd: false`), pero las filas aplicables de la matriz de amenazas exigen su prueba roja antes del código de producción.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad | `decodeWorkflow`: valores válidos, desconocido, nulo, lista, mapeo, clave duplicada o desconocida; `resolveWorkflow`; un decodificador sin el bloque ignora la clave | Tablas en `internal/workspace` |
| Unidad | `projectWorkflow`: cuatro orígenes; `invalid` con base rota | `Location` construidas a mano |
| Unidad | `docpaths`: cada raíz; archivo de otro cambio (`foo-x`) excluido; `reference` gana; `odd/tasks` es código; `LocalState`; `planningRel` vacío (multirepo) | Tablas |
| Unidad | Clasificación de push y commit | Entradas `--porcelain`/`stderr` fijas |
| Unidad | `kickoff`: `--docs-publication` con `checkpointed`, `--infer` o un valor desconocido se rechaza; sellado único; `Validate` | Tablas |
| Unidad | Doctrina: paso 0 con `workflow`, `lane: sdd`/`odd`/`ask`, `no_workspace`, `invalid` y «status cannot be read»; sin `Plan stop`; política desconocida y literal retirado → `ErrUnknownRoutingPolicy` | `routing_test.go`, `manifest_test.go` |
| Integración | Status v1 y v2 con y sin cambio | Fixtures de `sddstatus` |
| Integración | `publish`: cada fila aplicable de la matriz, cada código de §5.3 y `up_to_date` | Remoto *bare* + clon, `t.Setenv` para la identidad, hooks `#!/bin/sh\nexit 1` |
| Integración | `PublicationTarget`: `monorepo-embedded`, `multirepo` (specs in situ en `main`), archivado y rama de rol | `internal/branch/engine` |
| Integración | Adopción: cambio, rol y archivado (`<x>-archive`, con `checkIntegrated` aún exigido sobre la base), `in_place`, `--fetch`, sin referencia remota (`createdFrom: base`) | Remoto *bare*, segundo clon |
| Integración | `gate record` (salidas 2 y 3) devuelve `publish` con `prTarget` (vacío en specs in situ de multirepo); `--hand-off` no comprueba nada confirmado, tampoco en `role-apply` con código sucio; ese código lo informa después `publish` con `code_commits_unpublished` | `internal/cli` |
| Arquitectura | DD-14: procesos, importaciones, literales y observador | `internal/gitcommon`, `internal/sddpublish`, `internal/cli` |
| Goldens | 13 goldens; `.codex/AGENTS.md` igual al golden; pines de `docs_contract_test.go` | `-update` en `internal/components` |
| E2E | No aplica en local: el arnés de binario está vetado por el antivirus; la CI de GitHub ejecuta la suite completa | — |

## 7. Matriz de amenazas

| Frontera | Casos adversariales mínimos | Aplicabilidad | Respuesta del diseño | Pruebas rojas planificadas |
|---|---|---|---|---|
| Rutas que parecen documentación | `requirements.txt`, `CMakeLists.txt`, Markdown/MDX ejecutable, `README.sh` | **Aplicable** | Solo cuenta la lista cerrada de `docpaths` (DD-06), por prefijo de ruta y no por extensión: `README.sh` en la raíz o `docs/x.mdx` sin declarar son código; un `README.sh` dentro de `openspec/changes/<x>/` es documentación del cambio y nunca se ejecuta. `odd/tasks/**` y `reference` son código. | `TestDocPathsClassifiesByRootNotExtension` (cuatro casos); `TestPublishRefusesUnpushedCommitTouchingReadmeSh` |
| Selección del repositorio git | `git -C`, rutas relativas y absolutas | **Aplicable** | Todo git corre con `dir` = `PublicationTarget.GitRoot` absoluto (`gitcommon.Run` rechaza rutas relativas) y sin las variables que redirigen el repositorio; `--cwd` relativo se resuelve con `filepath.Abs` y `Locate`. Pathspecs con `:(top,literal)`. | `TestPublishRunsInTargetRootFromLinkedWorktree`; `TestPublishIgnoresGitDirEnvironment`; `TestPublishRelativeCwdResolvesToWorkspace` |
| Estado del commit | índice con código preparado, `commit -a`, índice vacío | **Aplicable** | `add --all` y `commit --only` sobre raíces de documentación; el código preparado sigue preparado y el no preparado sigue intacto; nunca `-a`; sin cambios no hay commit vacío; excluye `.gates.lock` y `.axiom-instance`. | `TestPublishKeepsStagedCodeStaged`; `TestPublishNeverCommitsUnstagedCode`; `TestPublishNoDocChangesMakesNoCommit`; `TestPublishExcludesLocalStateFiles`; `TestPublishCommitsArchiveDeletions` |
| Estado del push | rama de seguimiento, primer push, refspec explícita | **Aplicable** | `refs/heads/<b>:refs/heads/<b>` a `origin`, sin `+` ni `--force*`; primer push con `--set-upstream`; rama de integración rechazada salvo specs de `multirepo`; `--branch` ata la autorización; sin `fetch`. | `TestPublishFirstPushCreatesRemoteBranch`; `TestPublishTrackedBranchFastForward`; `TestPublishNonFastForwardRefusedWithoutRetry`; `TestPublishRefusesIntegrationBranchInCodeRepo`; `TestPublishMultirepoSpecsPushesIntegration`; `TestPublishBranchFlagMismatchIsWrongBranch`; `TestPublishNeverRunsFetchOrForce` (observador) |
| Órdenes de PR | `--head` explícito, prefijo de entorno, órdenes compuestas | N/A: el producto deja de proponer datos de PR en la cesión y no ejecuta ninguna orden de plataforma; la única orden compuesta para el humano (`git -C <raíz> push origin <b>:<b>`) se construye con `pathquote` como `HandOffPushStep`, ya probado | — | — |

Casos adicionales derivados de las filas aplicables, también con prueba roja: hook `pre-commit` que rechaza (`commit_hook_rejected`), hook `pre-push` local y `pre-receive` remoto (`push_hook_rejected`), identidad ausente (`identity_missing`), remoto con URL inexistente (`remote_unavailable`), sin `origin` (`remote_missing`), HEAD desacoplado (`detached_head`) y fusión de `main` publicada en la rama sin negativa falsa (`TestPublishIgnoresPublishedIntegrationCommits`).

## 8. Migración y despliegue

- No hay migración de datos. `axiom.yaml` sin `workflow` pasa a proyectar `lane: sdd` (`default`): es el cambio de comportamiento silencioso de la propuesta y se documenta en el manual y en la nota de versión, con `workflow.lane: ask` como forma de conservar la pregunta. Los repositorios sin `axiom.yaml` no cambian (S10).
- Los kickoffs ya sellados (este cambio incluido) no tienen `docs_publication` y no publican de forma automática.
- `axiom.sdd-gate-record/v1` pierde `delivery.mergeRequired` y los pasos de documentación. El consumidor es la doctrina, que `axiom sync` reproyecta con el mismo binario.
- **Reversión**: `git revert` por commit de unidad (DD-16). Un binario revertido ignora `workflow` y `docs_publication`, y las ramas publicadas quedan como ramas normales.

## 9. Preguntas abiertas

- [ ] **Ninguna bloquea.** Un solo supuesto técnico necesita validación durante `apply`: que `git rev-parse --git-path hooks/<hook>` respete `core.hooksPath` en la versión mínima de git que admite el producto. Si no la respeta, el commit y el push fallidos se clasifican como `commit_failed`/`push_failed` con la cola de `stderr`, y la distinción con «hook» se degrada sin perder información.
- [ ] La fase `spec` decide el texto exacto de las enmiendas a REQ-26.1, REQ-26.4, REQ-26.5, REQ-26.8, REQ-25.1 y REQ-25.7/25.10. Este diseño fija el comportamiento; si la redacción final exige otros tokens (por ejemplo, otro nombre para `createdFrom`), se ajusta en `tasks` sin cambiar la arquitectura.
