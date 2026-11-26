# Design: Estado «aprobada sin publicar» en `axiom sdd status` (C-05-c)

## Technical Approach

El status del cambio seleccionado gana un campo aditivo `publication`. Se calcula sin red y sin escrituras, en tres capas que ya existen:

1. **`internal/branch/engine`** reúne los hechos de git de la publicación. Parte del destino que ya calcula `PublicationTarget` y usa solo lecturas del puerto `branch.Git`.
2. **`internal/docpaths`** y **`internal/branch`** componen una única vez la lista cerrada de REQ-27.5. La usan tanto `publish` como el motor.
3. **`internal/sddstatus`** decide si procede proyectar (condición (a) y exclusiones), traduce los hechos al JSON y reutiliza la invocación de la pregunta de compuerta.

La doctrina del orquestador ordena ejecutar `publication.invocation` una sola vez antes de lanzar la fase siguiente y otra al reanudar. El estado `code_unpublished` (decisión D1, opción B) es solo informativo y lleva la orden de push para la persona.

Esto implementa el requisito nuevo de `sdd-docs-publication` sin cambiar REQ-27.5, REQ-27.8, REQ-27.12 ni REQ-27.13.

## Architecture Decisions

### Decision: Campo `publication` en la raíz del status, no dentro de `governance`

**Choice**: Se añade un campo `publication` en la raíz de `Status` y de `StatusV2Projection`, junto a `gateQuestion`, con `omitempty`.
**Alternatives considered**: `governance.publication`, como sugería la propuesta.
**Rationale**: El status de un cambio archivado (`archivedOpenSpecStatus`) no lleva `governance`. Meter ahí la publicación obligaría a proyectar `governance` en el archivado, y eso cambia bytes del JSON actual. En la raíz, el subconjunto `remote`, `branch`, `prTarget?` e `invocation` tiene la misma forma que `gateQuestion.publication`, y el consumidor reconoce la invocación.

### Decision: Lectura de rutas sucias con `--untracked-files=all` y pathspec literal (riesgo 1)

**Choice**: Se añade al puerto un método de lectura, `DirtyPathsUnder(ctx, dir, paths)`, en `branch.Git`, `gitReader` y `gitcli.Git`. Ejecuta `git status --porcelain=v1 -z --untracked-files=all --no-renames -- :(top,literal)<raíz>...` sobre `docpaths.Set.Candidates()`. Después filtra con `Root` y descarta `LocalState`. Es exactamente el paso 6 de `publish`.
**Alternatives considered**:
- Reutilizar `DirtyPaths` (`--untracked-files=normal`). Una carpeta sin seguimiento sale como una sola entrada `dir/`.
- Expandir esa entrada recorriendo el disco. No respeta `.gitignore` y divergiría de `publish`.
- Tratar `dir/` como documentación. Es falso si la carpeta solo tiene `.gates.lock` o `.axiom-instance`.
**Rationale**: Con `all`, git lista cada fichero, y `LocalState` ya sabe distinguir el estado local con punto. `kickoff.yaml` y `gates.yaml` no llevan punto: son documentación y `publish` los confirma. La paridad con `publish` es así estructural (mismo comando y mismo filtro), no aproximada. El coste es un `git status` limitado a las raíces candidatas, con `GIT_OPTIONAL_LOCKS=0` (`gitcommon`), así que no refresca el índice.

### Decision: `RemoteConfigured` como segunda lectura nueva del puerto

**Choice**: Se añade `RemoteConfigured(ctx, dir, remote)` (`git config --get remote.<r>.url`) a `branch.Git`, `gitReader` y `gitcli`. Sin `origin` configurado, el status no proyecta nada.
**Alternatives considered**:
- Deducir la ausencia de remoto porque faltan `refs/remotes/origin/*`. Falla en un remoto añadido y aún no traído, donde `publish` sí sube.
- Proyectar `pending`. Se repetiría la negativa `remote_missing` en cada transición.
**Rationale**: Sin `origin` no hay publicación posible, igual que con el almacén `engram`. Así se evita el bucle por la misma razón que motivó D1. Ningún nombre nuevo contiene `push`, `pull`, `lsremote`, `publish` ni `upload`, de modo que `TestTheGitPortHasNoPushMethodAndFetchIsTheOnlyRemoteOperation` sigue en verde.

### Decision: Rango sin subir con la misma regla que `publish` y lectura con `CommitFiles`

**Choice**: Las exclusiones se eligen como en el paso 5 de `publish`:
- con `refs/remotes/origin/<b>`, se excluyen esa referencia y, si existe, `refs/remotes/origin/<integración>`;
- sin ella, se excluye solo `refs/remotes/origin/<integración>`;
- sin ninguna de las dos, se excluye `refs/heads/<integración>`;
- sin ninguna, el rango es desconocido.

Los commits y sus rutas se leen en una sola llamada: `CommitFiles(dir, "HEAD", "^<excl>"...)`. Un commit es de código si alguna de sus rutas no cumple `set.Contains`. Por encima de 500 commits, el resultado es `range_too_big`.
**Alternatives considered**:
- `RevList` más un `CommitFiles` por commit: más lecturas.
- Portar `UnpublishedCommits` y `CommitPaths --cc` de `sddpublish`: amplía el puerto. Además, `Unpublished` contiene `publish` y rompería la prueba de nombres.
**Rationale**: Son lecturas que el puerto ya ofrece. **Límite aceptado:** `CommitFiles` no lista rutas en un commit de fusión, mientras que `publish` usa `--cc` y lista las que resolvió la fusión. En una fusión limpia ambos coinciden (sin rutas). Solo una fusión con conflictos resueltos sobre código puede dar `pending` donde `publish` se negaría. La doctrina acota el efecto: una sola ejecución por transición, y la negativa es informativa.

### Decision: Composición única de la lista cerrada (riesgo 3)

**Choice**: Se separan dos piezas:
- `docpaths.FromWorkspace(planningRel, change, maintained, reference)`. Es pura y recibe las rutas tal como se declaran en `axiom.yaml`, relativas al `workspaceRoot`. Absorbe `workspaceRelative` y `fromWorkspace`, que hoy están en `sddpublish/publish.go:341-371`.
- `branch.PublicationTarget.DocPaths(declarations workspace.Declarations, change string) (docpaths.Set, error)`. Desempaqueta `technical_context`.

`sddpublish.composeSet` pasa a ser una línea que llama a `target.DocPaths(location.Declarations, change)`, y el motor llama a lo mismo.
**Alternatives considered**: Que el motor duplique `composeSet`. Abre la puerta a la deriva que la propuesta quiere cerrar.
**Rationale**: Con un único punto de composición no hace falta una prueba de paridad textual. `docpaths` sigue siendo una hoja, porque no importa `workspace`. Las dependencias quedan sin ciclos:

```
sddstatus ──→ branch/engine ──→ branch ──→ docpaths (hoja)
     │               │              └────→ workspace
     └──→ branch     └──→ docpaths
sddpublish ──→ branch, docpaths          (solo internal/cli/sdd_publish.go importa sddpublish)
```

Ni `engine`, ni `branch`, ni `docpaths` importan `sddstatus` ni `sddpublish`. `sddstatus` ya obtiene hoy el motor con `engine.Default()` o con la variable de prueba `branchEngine` (`governance_handoff.go:97-115` y `branch_context.go:24`), y la decoración nueva usa la misma vía.

### Decision: `PublicationTarget` no cambia de semántica (riesgo 5)

**Choice**: El cuerpo actual de `PublicationTarget` pasa a un método interno, `resolvePublication(ctx, req) (branch.PublicationTarget, workspace.Location, *branch.Refusal)`. `PublicationTarget` lo envuelve y descarta la ubicación. El método nuevo, `PublicationState`, usa ese helper para no llamar dos veces a `gather`.
**Alternatives considered**: Volver a llamar a `workspace.Locate` desde `PublicationState`. Duplica lecturas.
**Rationale**: El destino, las negativas y su orden no cambian. Lo fijan sin modificación `TestPublicationAndHandOffKeepTargetingTheIntegrationInBranchMode` (`promotion_contracts_test.go:17-43`) y `publication_test.go`.

### Decision: Condición de proyección y exclusiones en `sddstatus`, como decoración

**Choice**: Se añade `decoratePublication(&status, location)` en `Resolve`, después de `decorateRequirements`. Se proyecta solo si se cumplen, en orden:

| # | Comprobación | Si falla |
|---|--------------|----------|
| 1 | Hay cambio seleccionado (`status.ChangeName != nil`) | nada |
| 2 | `effectiveArtifactStore` no es `engram` ni `none` | nada |
| 3 | Condición (a), que depende del caso (ver abajo) | nada |
| 4 | Hay motor (`branchEngine`/`engine.Default`); en las pruebas antiguas es nulo | nada |
| 5 | `PublicationState` resuelve | `unavailable` con el código |
| 6 | Hay `origin` configurado y se conoce el rango | nada (sin `origin`) o `unavailable` (`unpublished_range_unknown`) |
| 7 | Condición (b): hay documentación sucia o commits sin subir (más de 0) | nada |

Condición (a) según el caso:
- **Cambio activo `checkpointed`** (`governanceGated`): al menos una compuerta con estado `approved` y clave distinta de `integration`.
- **Cambio activo `continuous`**: `docsPublication == auto`, al menos uno de los artefactos `specs`, `design` o `tasks` en `done`, `taskProgress.completed == 0` y `verifyReport` distinto de `done`. Las dos últimas condiciones excluyen la ventana `apply`/`verify` (REQ-27.13: «No se publica tras `apply`»).
- **Cambio archivado** (`status.Archived != nil`): el kickoff se lee de `kickoff.ArchivedChangeRoot` y debe ser `continuous` con `docsPublication: auto`. La petición lleva `Archive: true`. Un cambio `checkpointed` archivado no proyecta nada, porque la doctrina ya ofrece la publicación del archivado y espera a la persona.

**Alternatives considered**:
- Calcular la publicación dentro de `resolveStatus`. Mezcla una lectura de git con el enrutado.
- Proyectar siempre que haya kickoff. Contradice REQ-27.13 en `continuous` sin `auto` y durante `apply`.

**Rationale**: El patrón `decorate*` ya separa las proyecciones aditivas, que no tocan `nextRecommended`, las dependencias ni `blockedReasons` (`requirements.go:27`). La condición (a) se evalúa antes de cualquier lectura de git, así que un cambio que no cumple (a) no paga nada. La condición (b) exige que haya algo que `publish` haría (confirmar o subir). Con la referencia remota ausente y el rango vacío, `publish` responde `up_to_date` sin subir nada. Proyectar `pending` en ese caso crearía otro bucle, así que «sin referencia remota» es un motivo (`remote_branch_missing`) y no basta por sí solo.

### Decision: Estados y precedencia (D1, opción B)

**Choice**: Se aplica la primera regla que se cumpla:

| Hechos | `state` | `reasons` | Campos propios |
|--------|---------|-----------|----------------|
| Negativa de `PublicationTarget` o fallo de lectura de git | `unavailable` | — | `unavailable: <código>` (`git_failed` en una lectura) |
| Rama de integración ausente en `origin` y en local | `unavailable` | — | `unavailable: "unpublished_range_unknown"` |
| Documentación sucia | `pending` | `docs_uncommitted` y, si procede, `branch_ahead`, `remote_branch_missing` y `code_commits` | `invocation` |
| Más de 500 commits sin subir | `code_unpublished` | `range_too_big` | `pushCommand` |
| Algún commit sin subir toca código | `code_unpublished` | `code_commits` y, si procede, `remote_branch_missing` | `pushCommand` y `codeCommits` |
| Solo commits de documentación sin subir | `pending` | `branch_ahead` y, si procede, `remote_branch_missing` | `invocation` |
| Nada que confirmar ni subir | (campo ausente) | | |

**Alternatives considered**: La opción A, con `pending` siempre. La descartó el usuario por el bucle de negativas.
**Rationale**: Con documentación sucia y commits de código a la vez, `publish` confirma en local y después se niega. El siguiente status ya proyecta `code_unpublished` y el bucle termina. `pushCommand` reproduce el formato de `sddpublish.pushCommand`: `git -C <raíz> push origin <b>:<b>`, con `pathquote.Quote` y `pathquote.ShellWord`.

### Decision: Invocación compartida con la pregunta de compuerta

**Choice**: Se extrae `gatePublicationFrom(target branch.PublicationTarget, workspaceRoot, change string) *GatePublication` de `resolveGatePublication`, y la usan los dos sitios. En el archivado, la invocación es `branch.PublishArchiveInvocation(...)`, que es `PublishInvocation(...) + " --archive"`, función nueva en `branch/handoff.go`.
**Alternatives considered**: Llamar a `resolveGatePublication` tal cual. Repite `PublicationTarget`, y por tanto `gather`, en la misma resolución.
**Rationale**: Hay una sola función que da forma a `remote`, `branch`, `prTarget` e `invocation`, así que no hay deriva. La invocación del estado `pending` es idéntica, byte a byte, a `gateQuestion.publication.invocation` cuando ambas aparecen. `axiom sdd publish` ya acepta `--archive` con `--branch` (`internal/cli/sdd_publish.go:25`).

### Decision: Doctrina del orquestador (riesgo 4)

**Choice**: Se añade una frase al final del párrafo «When to publish» de `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (línea 80). También se añade una viñeta en `internal/assets/skills/_shared/sdd-status-contract.md`, en la sección *Governance*, después de la viñeta de `gateQuestion`. Texto propuesto, en inglés como el resto del asset:

> When the status projects `publication` with `state: pending`, run its `publication.invocation` exactly once before launching the next phase and exactly once when resuming the change; its refusal is informational, never undoes an approval and never stops the chain. With `state: code_unpublished`, present its `pushCommand` to the person and never run it; with `state: unavailable`, present its code as information. The field never changes `nextRecommended`.

Después se regeneran los goldens con `go test ./internal/components -run TestGolden -update` y se vuelven a ejecutar sin `-update`.
**Alternatives considered**: Un párrafo nuevo. Fragmenta la regla de cuándo publicar.
**Rationale**: La frase completa la regla existente sin cambiar REQ-27.12 ni REQ-27.13.

## Data Flow

```
axiom sdd status --json
  Resolve ─→ resolveStatus (sin cambios) ─→ decorate* ─→ decoratePublication
                                                            │ (a) gobierno/kickoff, almacén, ventana
                                                            ▼
                                       engine.PublicationState(ctx, {CWD, Change, Archive})
                                            │ resolvePublication (= PublicationTarget)
                                            │ target.DocPaths(declarations) ─→ docpaths.Set
                                            │ RemoteConfigured · ResolveCommit×≤3
                                            │ DirtyPathsUnder(Candidates) · CommitFiles(HEAD ^excl)
                                            ▼
                                       branch.PublicationState (hechos)
                                            ▼
                         projectPublication ─→ Status.Publication ─→ StatusV2Projection.Publication
                                                                       │
                       orquestador: pending → ejecuta invocation 1 vez │ code_unpublished → muestra pushCommand
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/docpaths/docpaths.go` | Modify | `FromWorkspace` más `workspaceRelative` y `fromWorkspace`, movidos desde `sddpublish`. |
| `internal/branch/publication.go` | Modify | Tipo `PublicationState`, método `PublicationTarget.DocPaths` y constantes de código `PublicationCodeRangeUnknown`. |
| `internal/branch/handoff.go` | Modify | `PublishArchiveInvocation`. |
| `internal/branch/ports.go` | Modify | `RemoteConfigured` y `DirtyPathsUnder` en `branch.Git`. |
| `internal/branch/gitcli/ops.go` | Modify | Implementación de las dos lecturas, con pathspec `:(top,literal)` y validación de rutas. |
| `internal/branch/engine/engine.go` | Modify | Dos métodos más en `gitReader`. |
| `internal/branch/engine/publication.go` | Modify | Extracción de `resolvePublication`; `PublicationTarget` lo envuelve. |
| `internal/branch/engine/publication_state.go` | Create | `Engine.PublicationState`: hechos de solo lectura. |
| `internal/branch/engine/pending_test.go` | Modify | Dos stubs en `fakeGit`. |
| `internal/sddpublish/publish.go` | Modify | `composeSet` delega en `target.DocPaths`; se eliminan los dos helpers movidos. |
| `internal/sddstatus/publication.go` | Create | `PublicationProjection`, `decoratePublication`, `publicationDue`, `projectPublication` y `publicationPushCommand`. |
| `internal/sddstatus/governance_handoff.go` | Modify | `gatePublicationFrom`, compartido. |
| `internal/sddstatus/status.go` | Modify | Campo `Publication` y llamada a `decoratePublication` en `Resolve`. |
| `internal/sddstatus/status_v2.go` | Modify | Campo `Publication` y copia en `ProjectStatusV2`. |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modify | Frase de ejecución única. |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modify | Viñeta del campo `publication`. |
| `testdata/golden/*.golden` (10 ficheros con «When to publish») | Modify | Regenerados. |

## Interfaces / Contracts

```go
// internal/branch/ports.go (y gitReader en engine.go)
RemoteConfigured(ctx context.Context, dir, remote string) (bool, error)           // git config --get remote.<r>.url
DirtyPathsUnder(ctx context.Context, dir string, paths []string) ([]string, error) // status --porcelain=v1 -z --untracked-files=all --no-renames -- :(top,literal)<p>...

// internal/branch/publication.go
const PublicationCodeRangeUnknown Code = "unpublished_range_unknown" // = sddpublish.CodeUnpublishedRangeUnknown

type PublicationState struct {
	Target        PublicationTarget
	RemoteMissing bool     // origin sin URL: no hay publicación posible
	RangeUnknown  bool     // ni origin/<integración> ni <integración> local
	RemoteBranch  bool     // existe refs/remotes/origin/<Branch>
	DirtyDocs     []string // documentación sin confirmar, sin estado local, ordenada
	Unpushed      int      // commits de HEAD fuera de las exclusiones de REQ-27.8
	CodeCommits   []string // SHA cortos (7) de los commits sin subir que tocan código, del más antiguo al más reciente
	RangeTooBig   bool     // Unpushed > 500
}

func (t PublicationTarget) DocPaths(declarations workspace.Declarations, change string) (docpaths.Set, error)
func PublishArchiveInvocation(workspaceRoot, change, branchName string) string

// internal/docpaths/docpaths.go
func FromWorkspace(planningRel, change string, maintained, reference []string) (Set, error)

// internal/branch/engine/publication_state.go
func (e *Engine) PublicationState(ctx context.Context, req branch.PublicationRequest) (branch.PublicationState, *branch.Refusal)

// internal/sddstatus/publication.go
type PublicationProjection struct {
	State       string   `json:"state"`                 // pending | code_unpublished | unavailable
	Reasons     []string `json:"reasons,omitempty"`     // docs_uncommitted, branch_ahead, remote_branch_missing, code_commits, range_too_big
	Archive     bool     `json:"archive,omitempty"`     // momento de archivo (rama de archivado)
	Remote      string   `json:"remote,omitempty"`      // siempre "origin" cuando hay destino
	Branch      string   `json:"branch,omitempty"`
	PRTarget    string   `json:"prTarget,omitempty"`    // informativo; vacío en la integración in situ
	Invocation  string   `json:"invocation,omitempty"`  // solo pending: la misma de gateQuestion.publication
	PushCommand string   `json:"pushCommand,omitempty"` // solo code_unpublished: para la persona
	CodeCommits []string `json:"codeCommits,omitempty"` // solo code_unpublished
	Unavailable string   `json:"unavailable,omitempty"` // solo unavailable
}
// Status y StatusV2Projection: Publication *PublicationProjection `json:"publication,omitempty"` (después de GateQuestion)
```

Ejemplo `pending`:

```json
"publication": {
  "state": "pending",
  "reasons": ["docs_uncommitted", "remote_branch_missing"],
  "remote": "origin",
  "branch": "inc/x",
  "prTarget": "main",
  "invocation": "axiom sdd publish --cwd C:/repo --change x --branch inc/x"
}
```

**Compatibilidad**: el puntero con `omitempty` hace que, sin estado, el JSON no cambie ni un byte. Las pruebas existentes de `sddstatus` ejecutan con `branchEngine` devolviendo un motor nulo (`main_test.go`), así que la decoración no hace nada en ellas.

## Testing Strategy

Todas las pruebas son localizadas: solo se ejecutan los paquetes tocados y, en `internal/cli`, solo con `-run`. Nunca la suite entera.

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (`docpaths`) | `FromWorkspace`: `planningRel` `""`, `openspec` y `sub/openspec`; rutas declaradas fuera de la raíz se descartan; mismo `Set` que el `composeSet` anterior. | Tabla, `go test ./internal/docpaths/...` |
| Unit (`sddstatus`) | `publicationDue`: `checkpointed` con `spec` aprobada (sí); solo `integration` aprobada (no); `continuous` sin `auto` (no); `auto` con `specs` en `done` y 0 completadas (sí); `completed > 0` (no); `verifyReport` en `done` (no); sin kickoff (no). `projectPublication`: una fila por estado de la tabla de precedencia; `pushCommand` exacto con una ruta con espacios. | Tabla, `go test ./internal/sddstatus/... -run Publication` |
| Integration (`gitcli`) | `RemoteConfigured` con y sin `origin`. `DirtyPathsUnder`: una carpeta sin seguimiento se expande a ficheros, cambios preparados y sin preparar, borrados, y una ruta con espacios o `*` tomada como literal. | Git real en `t.TempDir()`, `go test ./internal/branch/gitcli/... -run 'RemoteConfigured\|DirtyPathsUnder'` |
| Integration (`engine`) | `PublicationState`: (1) carpeta del cambio sin seguimiento con solo `.gates.lock` → sin `DirtyDocs`; con `kickoff.yaml` → `[…/kickoff.yaml]`; (2) commit de documentación sin `origin/<b>` → `Unpushed=1`, `RemoteBranch=false`; (3) commit de `requirements.txt` → `CodeCommits`; (4) tras subir al remoto desnudo → todo vacío; (5) sin `origin` → `RemoteMissing`; (6) fusión limpia de `origin/main` → sin código; (7) multirepo de specs in situ → rango contra `origin/<integración>`; (8) rama de archivado con `Archive:true`; (9) solo lecturas: el observador no ve `push`, `commit`, `fetch`, `pull`, `ls-remote` ni `add`, y `repositoryState` no cambia. | Helpers existentes (`handOffMonorepo`, `remoteWithMain`, `plannedWorktree`, `observe`), `go test ./internal/branch/engine/... -run 'PublicationState\|PublicationTarget\|PublicationAndHandOff'` |
| Integration (`sddpublish`) | Las pruebas existentes de `publish` pasan sin cambios. Paridad: tras un `Run` publicado, `engine.PublicationState` no ve nada; tras editar un documento, `DirtyDocs` coincide con `CommitResult.Paths`. `branch.PublicationCodeRangeUnknown == sddpublish.CodeUnpublishedRangeUnknown`. | Git real con remoto desnudo; importar el motor solo desde la prueba. `go test ./internal/sddpublish/...` (incluye `isolation_test.go`) |
| Integration (`sddstatus`) | Con el motor real: `checkpointed` con `spec` aprobada y un documento sucio → `pending`, con `invocation` igual a `gateQuestion.publication.invocation`; almacén `engram` → sin campo; la clave `"publication"` no aparece en un status sin estado. | `go test ./internal/sddstatus/... -run Publication` |
| E2E (`cli`) | Extender `TestPublishIsolationNoVerbButPublishPushesCommitsOrFetches`: `sdd status --json` proyecta `pending` y no ejecuta `push`, `commit` ni `fetch`. | `go test ./internal/cli -run 'TestPublishIsolation'` |
| Doctrina | Prueba de `assets` y goldens. | `go test ./internal/assets/...`; `go test ./internal/components -run TestGolden -update` y después sin `-update` |
| Regresión | El destino no cambia. | `go test ./internal/branch/engine/... -run 'PromotionContracts\|PublicationAndHandOff\|TheGitPortHasNoPush'` y `go test ./internal/branch/ -run Publish` |

**Estimación**: unas 360 líneas de producción añadidas y unas 50 eliminadas (por `sddpublish`), más unas 450 de pruebas. Si hay que recortar hacia el objetivo de 250 a 400 líneas totales, hay dos piezas separables:
- el momento de archivo (unas 40 líneas de producción y 60 de pruebas);
- `RemoteConfigured` (unas 30 y 30), aceptando el bucle `remote_missing` en repositorios sin `origin`.

## Threat Matrix

| Boundary | Minimum adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| Documentation-like paths | `requirements.txt`, `CMakeLists.txt`, Markdown o MDX ejecutable, `README.sh` | Applicable: clasifica commits sin subir como documentación o código | Solo `docpaths.Set.Contains`, sobre la raíz de la ruta y nunca sobre la extensión, compuesto en un único punto (`target.DocPaths`). Las clases ya están probadas en `docpaths_test.go` | `engine`: un commit de `requirements.txt` y otro de `docs/x.mdx` sin declarar dan `CodeCommits`; uno en `openspec/changes/x/design.md` no |
| Git repository selection | `git -C`, rutas relativas y absolutas | Applicable: lecturas de git en el repositorio del planning home | Todas las lecturas usan `target.GitRoot` como `dir` de `gitcommon.Run`, nunca el directorio del proceso. `CWD` es `status.ActionContext.WorkspaceRoot` | `engine`: status desde un worktree del cambio y desde el repositorio de specs de un multirepo. `pushCommand` cita `GitRoot` absoluto |
| Commit state | Preparado, `commit -a`, índice vacío | Applicable (solo lectura) | Preparado, sin preparar, sin seguimiento y borrado cuentan como documentación sucia. Nunca se confirma nada | `gitcli` y `engine`: un documento preparado, un documento borrado y una carpeta sin seguimiento con solo estado local |
| Push state | Rama con seguimiento, primera subida, refspec explícito | Applicable: rango sin subir y orden de push para la persona | Exclusiones idénticas a REQ-27.8. Primera subida → motivo `remote_branch_missing`. `pushCommand` con refspec explícito `<b>:<b>`, sin forzar. Nunca se ejecuta | `engine`: sin `origin/<b>`, adelantada y tras subir. `sddstatus`: formato exacto de `pushCommand` |
| PR commands | `--head` explícito, prefijo de entorno, órdenes compuestas | N/A: no se compone ninguna orden de PR; `prTarget` es solo un dato | — | — |

## Migration / Rollout

No migration required. Es aditivo: el campo es opcional y no cambia el formato de `kickoff.yaml` ni de `gates.yaml`. Basta revertir la PR.

**Orden de fusión**:
- C-06-d también toca `sddpublish/publish.go`. Aquí solo cambia `composeSet` y desaparecen dos helpers, así que conviene fusionar antes uno de los dos y rebasar el otro.
- C-03-b y C-06-b/c/d tocan `sddstatus`. Esta decoración vive en un fichero nuevo y solo añade una línea a `Resolve`.

## Open Questions

- [ ] Ninguna bloqueante. Decisión a validar en verify: la representación en texto del status (sin `--json`) no muestra `publication`. Solo lo lee el orquestador.
