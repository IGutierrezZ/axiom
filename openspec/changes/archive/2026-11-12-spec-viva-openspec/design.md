# Diseño: Spec viva única en OpenSpec (spec-viva-openspec)

> Fase `design` · rama `inc/spec-viva-openspec` · base `8a704c9e` · 2026-11-12. Entradas: `proposal.md` (autoritativa), `explore.md` y las deltas de `specs/` (REQ-30.1 a REQ-30.30), leídas al cerrar este diseño. `axiom.yaml` de este repositorio no declara `technical_context`: no hay `context-impact.md`.
> Restricciones: Go 1.x del módulo `github.com/IGutierrezZ/axiom/v3`; solo `go test` acotado por paquete (sin binarios sueltos); goldens con `-update`; ratchets de negativas y de código muerto sin empeorar; una PR con `size:exception`, un commit por unidad.

## 1. Enfoque técnico

Cuatro paquetes nuevos, sin estado propio y con dependencias en un solo sentido, y adaptadores CLI finos en el grupo `axiom archive`:

```
cmd/axiom (archive integrate|adopt|context) ──► internal/cli/archive_*.go
                                                   │
        ┌──────────────────┬───────────────────────┼──────────────────────┐
        ▼                  ▼                       ▼                      ▼
internal/livespec   internal/adoption      internal/techcontext   internal/hub (detector)
 (ODD: secciones,    (scan, inventario,      (delta de contexto,           │
  IDs, plan, apply)   check)                  plan/traza, git)             │
        │   └──────┬───────┘   └──── adoption.Scan ◄────────────────────────┘
        ▼          ▼
internal/openspecdoc (hoja: análisis y composición OpenSpec, sin dependencias internas)
        ▲
internal/sddstatus (conserva ComposeOpenSpecCanonicalSpecWithOptions como envoltorio)
```

`livespec` y `techcontext` leen `workspace.Location`; `livespec` usa `techcontext.Scope` y `techcontext.ChangedSince` para REQ-30.6. Todo git pasa por `gitcommon.Run`. Ningún paquete nuevo importa `specbaseline`, así que U6 lo borra sin tocarlos.

## 2. Decisiones de arquitectura

### D1. `internal/openspecdoc`: extraer el analizador y el compositor OpenSpec de `sddstatus`
**Elección**: mover `internal/sddstatus/openspec_archive_compose.go` (410 líneas) y su prueba a un paquete hoja con API exportada mínima; `sddstatus` conserva `ComposeOpenSpecCanonicalSpecWithOptions`, `ComposeOpenSpecCanonicalSpec`, `ComposeOptions` y `UnappliedDeltaError` como envoltorio y alias de tipo.
**Alternativas**: exportar `parseSpecDocument`/`parseOpenSpecDelta` desde `sddstatus` (obliga a `hub` y `adoption` a importar `sddstatus`, que arrastra `branch/engine`, `reviewtransaction`, `kickoff`); duplicar un analizador por regex en `livespec` (dos verdades sobre qué es un requisito).
**Rationale**: `livespec`, `adoption` y el detector del hub necesitan el mismo análisis que la composición del archivado (REQ-30.2 exige «la misma composición nativa»). El movimiento es mecánico, con las pruebas existentes como red.

### D2. U1 extrae el contexto técnico en paralelo, sin refactorizar `specbaseline`
**Elección**: `internal/techcontext` nace como extracción dirigida de la rama de contexto (`contextdelta.go`, `anchors.go`, el subconjunto de `document.go` que usan las anclas, la evaluación de contexto de `plan.go:1158-1240`, las comprobaciones de contexto de `verify.go` y la vista git) sobre `gitcommon.Run`, sin baseline, sin perfiles y sin IDs provisionales. `specbaseline` y `axiom sdd integrate` no cambian hasta U6. `axiom archive context` usa `techcontext`; con un `baseline:` declarado se niega remitiendo a `axiom sdd integrate` (solo U1–U5).
**Alternativas**: que `specbaseline` importe `techcontext` y delegue (la propuesta dice «delega»): `ContextEntry` depende de `Anchor`, que depende de `Document`, que es el núcleo de la composición por perfil con métodos propios (`Requirements(Profile)`); separar eso es reescribir código que U6 borra.
**Rationale**: el invariante de la propuesta (sin ventana rota, conducta de contexto idéntica) se garantiza por el formato compartido (D3) y por pruebas cruzadas, no por compartir código. La duplicación es temporal y acotada a U1–U5. **Desviación consciente** del «delega» de la propuesta §5.1.

### D3. El plan y la traza del contexto siguen en `integration.yaml` con el esquema `axiom.sdd-integration/v1`
**Elección**: `techcontext.Trace` es un subconjunto estricto de `specbaseline.Plan` (`schema`, `change`, `prepared_at`, `verified_at`, `inputs_sha256`, `repositories`, `context`, `entries` de tipo `context`). El decodificador tolera `options`, `id_map` y `documents` vacíos y rechaza con `*ForeignPlanError` un plan con documentos o entradas que no son de contexto. `prepare` escribe el plan; `verify` lo marca `integrated` y es la traza (REQ-30.15, REQ-30.18).
**Alternativas**: fichero nuevo `context-integration.yaml` (rompe los cambios en curso preparados con `sdd integrate` y las trazas archivadas que `show` debe leer).
**Rationale**: sin baseline declarada, `integration.yaml` ya es solo de contexto; conservarlo hace que el cambio de verbo en las skills (U6) no exija migración.

### D4. Formato ODD: valla con etiqueta `openspec` y cabecera `Tipo:`
**Elección**: `## Spec viva` contiene `### Capacidad: <dominio>` (dominio `^[a-z0-9][a-z0-9-]*$`) y, debajo, exactamente un bloque vallado con etiqueta `openspec` (``` o ~~~, longitud ≥ 3; si la delta contiene vallas, la exterior es más larga, regla CommonMark). `Tipo: bug` o `Tipo: incremento` (por defecto, incremento) es una línea propia antes del primer `## `. `## Impacto en la spec` sigue el contrato de marcador de REQ-29.15 de `carril-sdd-versatil`: una línea exacta `Sin impacto en la spec` (comparación sensible a mayúsculas, solo se ignoran los espacios al principio y al final; un punto final la invalida) más una línea `Motivo: <una línea>` con texto no vacío. Con una `## Spec viva` no vacía el marcador se ignora y el documento se evalúa como documento con delta (no se rechaza); sin marcador válido ni delta, un bug se rechaza. ODD conserva `Tipo: bug` y no exige el `## Informe de bug` del SDD. Una línea de casilla (`- [ ]`, `- [x]`) dentro de la delta se rechaza (`checkbox_in_delta`), porque `dashboard.parseODDDoc` cuenta casillas también dentro de vallas y REQ-30.1 prohíbe alterar el recuento.
**Alternativas**: ignorar vallas en `parseODDDoc` (cambia el recuento de documentos existentes); clasificar bug por la presencia de secciones (P3, alternativa descartada por la propuesta).
**Rationale**: los H2 de la delta no pueden ir sueltos; la etiqueta hace el bloque inequívoco aunque la sección tenga otros fragmentos de código.

### D5. Idempotencia por clasificación de cada requisito, sin huella persistida
**Elección**: antes de componer, cada requisito de la delta se clasifica contra la spec canónica con cuerpos normalizados (LF, sin espacios finales, sin líneas en blanco al principio y al final): ADDED o MODIFIED con cuerpo igual → `already`; ADDED presente con cuerpo distinto → `*DivergentRequirementError`; REMOVED ausente → `already`; RENAMED con el nombre nuevo presente y el viejo ausente → `already`; el resto → `pending`. Se compone solo una delta reducida con los `pending`; si no queda ninguno, la capacidad es `already_integrated` y no se escribe.
**Alternativas**: registrar `Integrado:` y una huella en el documento ODD (REQ-30.2 prohíbe modificarlo); huella en un fichero lateral (rompe el listado ODD, R4).
**Rationale**: el estado verdadero es la spec canónica; tras un rebase no hay registro que quede desfasado. Un MODIFIED cuyo nombre ya no casa sigue fallando cerrado con `UnappliedDeltaError` (REQ-30.3).

### D6. Registro único de identificadores `REQ-n.m`
**Elección**: `livespec.Registry` indexa los títulos `### Requirement: … (REQ-n.m)` de `<planningHome>/specs/*/spec.md` y de `<planningHome>/changes/*/specs/*/spec.md` (secciones ADDED, MODIFIED y RENAMED), excluido `changes/archive/`. Lo consumen `integrate --odd` (REQ-30.4) y `adopt check` (REQ-30.11 b, REQ-30.12, con la opción de excluir su propio cambio y de limitarse a `specs/`). `NextFree(n)` = máximo `m` de la serie `n` en todo el registro, más uno.
**Alternativas**: servicio de asignación de IDs (fuera de alcance, R2).
**Rationale**: una sola implementación del alcance de P5 y del «siguiente número libre».

### D7. Base de la rama y ficheros cambiados (REQ-30.6)
**Elección**: por cada repositorio que contiene una ruta `maintained` o `reference`, la base es el `prTarget` que devuelve `engine.Default().Status(ctx, cwd, feature, branch.LaneODD)` para ese repositorio, resuelto localmente como `origin/<prTarget>` si existe y si no `<prTarget>`; si `HEAD` ya es esa rama (repositorio de specs de un `multirepo`), la base es `@{upstream}`. Nunca `fetch`. Cambiados = `diff --name-status <base>...HEAD` ∪ índice (`--cached`) ∪ árbol de trabajo ∪ no seguidos (`ls-files --others --exclude-standard`); en un renombrado cuentan ambas rutas. Sin base resoluble → `*BaseUnresolvedError` (fallo cerrado).
**Alternativas**: comparar solo `HEAD` contra la base (una edición de referencia sin confirmar pasaría).
**Rationale**: la regla de referencia (REQ-24.30) es una garantía; un cambio pendiente de confirmar acabará en el commit de cierre.

### D8. Adopción: inventario YAML, descartes en `adoption.md` y referencia de skill
**Elección**: `adopt scan --change <c>` escribe solo `<changeRoot>/adoption-inventory.yaml` (`axiom.openspec-adoption-inventory/v1`); los descartes viven en `<changeRoot>/adoption.md`, sección `## Descartado`, una viñeta `- <ruta>#<ancla>: <motivo>` por unidad. Las anclas son *slugs* de encabezado al estilo GitHub (minúsculas, letras Unicode y dígitos, espacios a `-`, puntuación fuera, duplicados `-1`, `-2`); una unidad sin ancla es el fichero entero. `Origen:` sigue REQ-30.10: `Origen: <ruta>#<ancla> (<ID original>)`, con la ruta opcionalmente entre comillas invertidas si contiene espacios. La doctrina es una referencia, `internal/assets/skills/sdd-spec/references/openspec-adoption.md`, que `sdd-spec` sigue cuando existe el inventario en la carpeta del cambio, y una línea en `sdd-propose`.
**Alternativas**: skill nueva `sdd-adopt` (exige registrarla en `catalog`, `model/*`, selectores de la TUI, `uninstall`, `components/sdd`); anclas `Título > Título` de `techcontext` (la spec fija `#<ancla>`).
**Rationale**: un cambio de adopción es un cambio SDD normal; lo que cambia es cómo se redacta la spec, y la presencia del inventario es un disparador determinista.

### D9. Retirada de `baseline:` por el canal de error existente
**Elección**: `workspace.readDeclarationBlocks` convierte la mera presencia de la clave `baseline` en `*workspace.BaselineRetiredError` (código `baseline_retired`, con ruta, línea y la continuación `axiom archive adopt scan --path <ruta> --change <cambio>`) en el campo `Declarations.BaselineRetiredErr`, que sustituye a `BaselineErr` en las listas que ya fallan cerrado: `ParseConfig`/`LoadConfig`, `Validate` (`axiom workspace validate`), `invalidDeclarationErrors` del status (bloquea solo `archive`) e `hub.inspectOwnMarkers`.
**Alternativas**: ignorar el bloque (escribiría una segunda spec viva); un validador aparte (duplica canales).
**Rationale**: REQ-30.21 (a)–(d) se cumplen con los consumidores que ya existen; el archivado se detiene por el status y por `archive-compose` (§5, U6).

### D10. Convenciones de la superficie CLI
**Elección**: los verbos nuevos viven en `internal/cli` (`RunArchiveIntegrate`, `RunArchiveAdopt`, `RunArchiveContext`, firma `func(args []string, stdout io.Writer) error`), con `flag.ContinueOnError`. Aceptan `--path` (convención del grupo `archive`, REQ-30.9) y `--cwd` como alias (continuidad con `sdd integrate`, REQ-30.15); si se dan ambos con valores distintos, error. `--json` en todos. Toda negativa nombra una continuación `axiom …` o lleva `// refusal:by-design`. `openspec/INDEX.md` se regenera con `livingdoc.Service.Sync` solo si ya existe (criterio de REQ-30.2: «cuando el proyecto lo mantiene»).

### D11. `_shared/project-baseline-convention.md` conserva su nombre
**Elección**: en U6 se reduce a contexto técnico manteniendo las secciones 5 y 6 con su número.
**Rationale**: las instrucciones del orquestador citan «`project-baseline-convention.md`, sections 5 and 6»; renombrar obliga a regenerar todos los goldens del orquestador y los `CLAUDE.md` gestionados sin valor de producto.

## 3. Flujo de `axiom archive integrate --odd <feature>`

```
Locate(--path) ─► BaselineRetiredErr? ─► ISO? (U6) ─► guarda de rama (LaneODD)
   └► leer <workspaceRoot>/odd/tasks/<feature>.md ─► ParseFeatureDoc ─► reglas de impacto (U3)
        └► Registry ─► colisiones ─► Plan por capacidad (D5) ─► contexto (D7, U3)
             └► sin errores: Apply todo o nada (temporal + rename, restauración) ─► Sync INDEX si existe
```

Orden de rechazos: configuración, ISO, rama, documento, secciones, impacto, IDs, composición, contexto. Todo se calcula en memoria; la primera escritura ocurre tras la última comprobación. Informe `axiom.odd-integration/v1`: `feature`, `kind`, `impact` (`no_impact|delta|absent`), `capabilities[{name,path,outcome}]` (`created|composed|already_integrated`), `context{declared,maintainedTouched[],referenceTouched[]}`, `written[]`, `nothingToIntegrate`.

Capacidad nueva (solo ADDED): `openspecdoc.RenderNewSpec` escribe `# Especificación: <capacidad>`, `## Propósito` con «Integrada desde el documento ODD `<feature>`.» y `## Requirements` con los bloques en su orden.

## 4. Interfaces y contratos

```go
// internal/openspecdoc
type Requirement struct{ Name, ID, Body string; Line int } // ID "" si el título no lleva (REQ-…)
type Delta struct{ Added, Modified, Removed, Renamed []Requirement; Unknown []string }
func ParseDelta(text string) Delta
func ParseCanonical(text string) []Requirement
func Compose(canonical, delta string, opts ComposeOptions) (string, error) // *UnappliedDeltaError
func RenderNewSpec(capability, origin string, added []Requirement) string
func NormalizeBody(body string) string

// internal/livespec
type Kind string // "bug" | "increment"
type FeatureDoc struct{ Feature string; Kind Kind; Capabilities []CapabilityDelta; Impact *Impact; SpecSection bool }
type CapabilityDelta struct{ Name string; Line int; Delta openspecdoc.Delta; Text string }
type Impact struct{ NoImpact bool; Reason string; Line int }
func ParseFeatureDoc(feature string, data []byte) (FeatureDoc, error) // *DocError{Problems []Problem{Code, Line, Subject, Message}}
func CheckImpact(doc FeatureDoc) error                               // U3, *DocError
type Registry struct{ /* id -> []IDRef */ }
type RegistryOptions struct{ ExcludeChange string; SpecsOnly bool }
func LoadRegistry(planningHome string, opts RegistryOptions) (*Registry, error)
func (r *Registry) Lookup(id string) []IDRef
func (r *Registry) NextFree(series int) string
type Integration struct{ Capabilities []CapabilityPlan; Context *ContextReport }
func PlanIntegration(loc workspace.Location, doc FeatureDoc, reg *Registry) (Integration, error)
func (i Integration) Apply() ([]string, error) // todo o nada
// Errores: *DocError, *CollisionError{ID, Where, NextFree}, *DivergentRequirementError,
// *ReferencePathError{Files}, *BaseUnresolvedError, *ISOProjectError (U6)

// internal/techcontext
const TraceFileName, DeltaFileName = "integration.yaml", "context-impact.md"
type Target struct{ Location workspace.Location; Change, ChangeRoot, ArtifactStore string; AllowedRoots []string }
func Check(ctx context.Context, t Target) (Report, error)
func Prepare(ctx context.Context, t Target) (Trace, Report, error)
func Verify(ctx context.Context, t Target) (VerifyReport, error) // presence|anchor|scope|trace
func ReadTrace(file string) (Trace, error)
func RenderTrace(tr Trace, format string) ([]byte, error)        // markdown|json
type Scope struct{ WorkspaceRoot string; Maintained, Reference []string }
func ScopeOf(loc workspace.Location) Scope
func (s Scope) Classify(abs string) PathMode // maintained|reference|outside
func ChangedSince(ctx context.Context, repo, base string) ([]string, error)

// internal/adoption
const InventoryFileName, DiscardsFileName = "adoption-inventory.yaml", "adoption.md"
type Inventory struct{ Schema string; Sources []Source; ContextCandidates []ContextCandidate; Warnings []string }
type Source struct{ Path, SHA256, Kind string; Units []Unit } // Kind: markdown|file
type Unit struct{ Anchor, Title string; Line int; ForeignIDs []string }
type ContextCandidate struct{ Path, Mode, Reason string } // Mode: maintained|reference
func Scan(root string) (Inventory, error)
func Summarize(inv Inventory) Summary // para el hub: orígenes, recuento de unidades, candidatos
func Check(loc workspace.Location, change string) (CheckReport, error)
type CheckReport struct{ Change string; Passed bool; Failures []Failure } // Failure{Rule, Subject, Detail}
```

Reglas de `adoption.Check` (todas se acumulan): `format`, `uniqueness`, `series`, `scenarios`, `origin`, `coverage`, `discard_reason`, `inventory_missing` (remite a `axiom archive adopt scan --change <c>`), `inventory_stale` (SHA-256 de un origen distinto del inventario).

Reglas de `adoption.Scan` (deterministas, rutas con `/` ordenadas): recorre `workspaceRoot` excluyendo `openspec/`, `.openspec/`, `odd/`, `.git/`, `node_modules/`, `vendor/`, `testdata/` y directorios ocultos; como mucho 2.000 ficheros y 2 MiB por fichero (el exceso va a `Warnings`). Es origen un `.md`/`.markdown`/`.txt` cuyo nombre case `(?i)requisit|requirement|prd|srs|especificaci|specification|historias|user-?stor|casos?-de-uso|use-?case`, o que contenga al menos un ID ajeno, o un `.docx`/`.xlsx`/`.pdf` con nombre de ese patrón (unidad de fichero entero). Unidades: encabezados ATX H1–H3 fuera de vallas. ID ajeno: `\b(?:RF|RNF|RN|REQ|US|HU|UC|CU|FR|NFR)[-_](?:[A-Z]{2,5}[-_])?\d{1,5}\b`, descartando la coincidencia seguida de `.<dígito>` (es un `REQ-n.m` OpenSpec). Candidatos a contexto: `ARCHITECTURE.md` y `docs/arquitectura*`, `docs/architecture*`, `docs/design*` como `maintained`; `adr/`, `docs/adr/`, `docs/decisions/` como `reference`.

## 5. Plan por unidad

### U1 · Contexto técnico reubicado y `axiom archive context` (REQ-30.15 a REQ-30.20, REQ-30.29, REQ-30.30)

| Fichero | Acción | Contenido |
|---|---|---|
| `internal/techcontext/{delta,anchors,markdown,scope,gitview,plan,verify,trace}.go` | Crear | Extracción de la rama de contexto (D2), git por `gitcommon.Run` |
| `internal/techcontext/*_test.go` | Crear | Pruebas de contexto portadas de `specbaseline` (`contextdelta_test`, anclas, `gitview_test`, casos de contexto de `plan_test`/`verify_test`) |
| `internal/cli/archive_context.go` (+ prueba) | Crear | `check|prepare|verify|show`, guarda A en contexto `archive` para `prepare`/`verify`, negativa con `baseline:` declarado (U1–U5) |
| `cmd/axiom/main.go` | Modificar | `case "context"` en `archive`; `axiom archive --help` lista `integrate`, `adopt`, `context` (escenario de REQ-30.15) |
| `internal/gitcommon/architecture_test.go` | Modificar | Incluye `internal/techcontext` e `internal/livespec` en los paquetes sin `os/exec` |

### U2 · Integración ODD (REQ-30.1 a REQ-30.4, REQ-30.8; REQ-24.58 parte 2)

| Fichero | Acción | Contenido |
|---|---|---|
| `internal/openspecdoc/{compose,parse,render}.go` (+ pruebas movidas) | Crear | D1 |
| `internal/sddstatus/openspec_archive_compose.go` | Modificar | Envoltorio y alias; la prueba existente sigue en `sddstatus` contra el envoltorio |
| `internal/livespec/{odddoc,registry,plan,apply,report}.go` (+ pruebas) | Crear | D4–D6, §3 |
| `internal/cli/archive_integrate.go` (+ prueba) | Crear | Flags `--odd <feature>` (obligatoria), `--path/--cwd`, `--json`; guarda `RequireChangeBranch{Lane: LaneODD, Context: ContextOrdinary}` |
| `internal/components/agentguidance/routing.go` | Modificar | Paso 7: `## Spec viva`, `## Impacto en la spec` con el marcador `Sin impacto en la spec` y `Motivo:`, `axiom archive integrate --odd <feature>` en el commit que cierra la feature, sin paradas |
| `testdata/golden/**`, `AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md` | Regenerar | `go test ./internal/components/ -run Golden -update` |

### U3 · Impacto de los bugs y contexto en ODD (REQ-30.5, REQ-30.6; REQ-24.58 parte 1)

| Fichero | Acción | Contenido |
|---|---|---|
| `internal/livespec/impact.go`, `context.go` (+ pruebas) | Crear | `CheckImpact`; informe de contexto con `techcontext.Scope`/`ChangedSince` (D7) |
| `internal/livespec/plan.go` | Modificar | Encadena impacto y contexto antes de componer |
| `routing.go` y goldens | Modificar/regenerar | Paso 7: leer `technicalContext.maintained.entry|paths` de `axiom sdd status --json`, abrir `entry` primero |

### U4 · Adopción, hub y dashboard (REQ-30.9 a REQ-30.14, REQ-30.26, REQ-30.27)

| Fichero | Acción | Contenido |
|---|---|---|
| `internal/adoption/{scan,slug,inventory,discards,check}.go` (+ pruebas, `testdata/`) | Crear | §4 |
| `internal/cli/archive_adopt.go` (+ prueba) | Crear | `scan [--path] [--change] [--json]`, `check --change [--path] [--json]`; `scan --change` pasa la guarda A en contexto `ordinary` y escribe solo el inventario |
| `internal/hub/types.go`, `detector.go` | Modificar | `TechDetection.Adoption *adoption.Summary` (`json:"adoption,omitempty"`), relleno con `adoption.Scan` sin escribir |
| `internal/hub/types.go` (`InitResult`), `init.go` | Modificar | `Adoption *adoption.Summary` y `AdoptionInstruction string` (orden `axiom archive adopt scan --path <ruta> --change adopcion-openspec` y la frase para el agente) |
| `internal/dashboard/assets/js/projects.js` (+ prueba de asset) | Modificar | Panel tras el alta con orígenes, unidades, candidatos y la instrucción con botón de copiar; sin ejecutar nada |
| `internal/assets/skills/sdd-spec/SKILL.md`, `references/openspec-adoption.md`, `sdd-propose/SKILL.md`, `internal/assets/assets_test.go` | Modificar/crear | D8; la lista de referencias del test incluye el fichero nuevo |

### U5 · Documentación (G6.3, deriva)

`axiom.example.yaml` documenta `technical_context:` con los dos modos (sin `baseline:`); `docs/manual-de-inicio.md` §8–9 reescritos (adopción, integración ODD, contexto) y sin `axiom odd create|status|promote`. Prueba nueva en `internal/workspace`: `axiom.example.yaml` carga sin errores y declara `technical_context`.

### U6 · Alineación con carril y retirada (REQ-30.7, REQ-30.21 a REQ-30.25, REQ-2.2)

**Precondición dura**: `carril-sdd-versatil` fusionado en `main` con su segunda enmienda; este cambio rebasado sobre él.

1. **Reubicar lo que carril deje vivo en `specbaseline`**: `rg "specbaseline\." -g '!internal/specbaseline/**'` debe quedar vacío salvo en ficheros que esta unidad borra. Lo esperado (traza `--first-parent`, consulta de un REQ existente) se mueve al paquete de su consumidor de carril, no a los paquetes de este cambio.
2. **Negativa ISO**: `livespec` rechaza con `*ISOProjectError` antes de leer el documento cuando la declaración ISO de carril está activa (nombre del campo provisional, `Declarations.SDD.ISO`, se ajusta al rebasar); continuación: perfil de bug SDD o cambio SDD (`axiom sdd kickoff seal --cwd <ruta> --change <cambio> …`).
3. **Rechazo tipado** (D9): `internal/workspace/retired.go` nuevo; `declarations.go` sin `BaselineConfig`, `BaselineDocument`, `RequirementProfile`, `IntegrationMode`, `ScenariosMode`, `IdentityKind`, `decodeBaseline`, `validateDocuments*`, `validateRender`, `baselinePath` (en `contextOverlaps`); `validator.go` sin `checkBaselineDocuments` ni el error de almacén de baseline; `types.go` y `loader.go` sin `Baseline`.
4. **Borrar**: `internal/specbaseline/**` (incluido `testdata/reference`); `internal/workspace/baseline_guard.go` (+ pruebas); `internal/cli/sdd_integrate*.go`, `sdd_archive_compose_document_test.go`, `sdd_archive_compose_inplace_test.go`; en `sdd_archive_compose.go`, los modos `--document`, `--change`, `--in-place` y sus ayudantes (se conserva `--canonical --delta`, REQ-30.22–30.24); `sddstatus`: `BaselineProjection`, `projectBaseline`, el sujeto baseline de `declarationCauses`, `Status.Baseline`; `integratePrepareLine` pasa a nombrar `axiom archive context prepare` solo con contexto mantenido; `livingdoc/service.go` (`externalBaselineOf`, catálogo externo); `knowledge/specs_root.go`; `sddpublish/publish.go:345` y `docpaths.New` sin parámetro `baseline`; `hub/init.go` sin `BaselineProject`/`BaselineOwnership` (con `BaselineRetiredErr` no se crean `openspec/specs/` ni `INDEX.md` y la nota remite a la migración); `dashboard/archive_baseline_test.go`.
5. **Stubs**: `axiom sdd integrate` queda como negativa tipada que nombra `axiom archive context` y `axiom archive adopt` (REQ-30.25); `archive-compose` con salida a fichero resuelve el workspace del destino y se niega con `BaselineRetiredError` (REQ-30.21 d).
6. **Doctrina**: skills `sdd-archive` (referencia renombrada a `references/technical-context-integration.md`, con `axiom archive context`), `sdd-apply`, `sdd-spec`, `sdd-propose`, `_shared/project-baseline-convention.md` (D11), `_shared/sdd-status-contract.md` sin `baseline`; `axiom.example.yaml` con `sdd:` (R7); goldens regenerados.
7. **Ratchets**: `.deadcode-baseline.txt` y `.refusal-ratchet-baseline.txt` pierden las entradas de lo borrado (nunca ganan).

## 6. Estrategia de pruebas

| Unidad | Unidad (paquete) | Integración (repositorio git temporal) | Contrato y pines |
|---|---|---|---|
| U1 | Delta, anclas, alcance, decodificador de traza (incluido plan v1 con campos vacíos y `*ForeignPlanError`) | Plan escrito por `sdd integrate prepare` verificado por `archive context verify` y viceversa (se retira en U6); `check` no escribe (instantánea del árbol antes y después) | Ayuda de `axiom archive`; códigos de salida; ratchet de negativas |
| U2 | Tabla de `ParseFeatureDoc` (vallas de 3 y 4, `~~~`, valla sin etiqueta, H2 dentro, capacidad duplicada, casilla en la delta, `Tipo:` inválido); `Registry` (archive excluido, `NextFree`); clasificación D5 | Integrar dos veces → bytes idénticos; todo o nada con fallo de escritura inyectado; capacidad nueva; colisión con cambio activo; `parseODDDoc` da el mismo recuento con y sin `## Spec viva` | Golden del paso 7 |
| U3 | Marcador exacto (mayúsculas, espacios, punto final inválido), `Motivo:` vacío o ausente, bug sin marcador ni delta; marcador con `## Spec viva` no vacía se ignora y se integra la delta | Rama de feature sobre base: edición de referencia confirmada, en índice, sin confirmar y no seguida → rechazo; mantenida → informe; sin `technical_context` → sin informe; base irresoluble → rechazo | Golden del paso 7 con el localizador |
| U4 | `Scan` determinista (dos pasadas iguales), slugs con acentos y duplicados, filtro de IDs; cada regla de `Check` y su acumulación | `scan` sin `--change` no escribe; con `--change` solo el inventario; detector del hub; `POST /api/projects/init` devuelve `Adoption` | Asset `projects.js`; referencia en `assets_test` |
| U5 | `axiom.example.yaml` carga | — | Pin: el manual no contiene `axiom odd` |
| U6 | `BaselineRetiredError` en `LoadConfig`, `Validate` (salida 1), status (sin clave `baseline`, `archive` bloqueado, planificación no) | `archive-compose` a fichero con `baseline:` se niega; `sdd integrate` se niega sin escribir; ISO rechaza `integrate --odd` | Pines: skills sin `sdd integrate` ni `baseline:`; goldens; ratchets; `go vet ./...` |

Los comandos `axiom archive …` documentados quedan en el nivel *presence-only* del corpus de invocaciones (`archive` no está en la sección independiente de plataforma de `app.go`); no se amplía ese corpus en este cambio.

## 7. Matriz de amenazas

| Frontera | Aplicabilidad | Respuesta | Pruebas RED |
|---|---|---|---|
| Rutas tipo documentación | Aplicable: `adopt scan` clasifica ficheros por nombre y extensión | Solo lee; nunca ejecuta ni sigue enlaces simbólicos fuera de `workspaceRoot`; `README.sh`, `CMakeLists.txt`, `requirements.txt` no son origen (extensión fuera de la lista) | Una por clase: `.sh`, `requirements.txt`, `.mdx`, enlace simbólico saliente |
| Selección de repositorio git | Aplicable: D7 ejecuta git en los repositorios de las rutas de contexto | `gitcommon.Run` con `dir` = raíz resuelta por `gitcommon.TopLevel` de cada ruta declarada; nunca el cwd del proceso; rutas relativas resueltas contra `workspaceRoot` | Ruta relativa, absoluta y `../repo` hermano |
| Estado de commit | Aplicable: los cambiados incluyen índice, árbol y no seguidos | Unión de los cuatro conjuntos (D7) | Una por estado: confirmado, en índice, sin confirmar, no seguido |
| Estado de push | N/A: ningún verbo hace push ni fetch | — | — |
| Órdenes de PR | N/A: ningún verbo crea ni lee PR | — | — |

## 8. Colisión con `carril-sdd-versatil` y rebase

| Zona | Carril | Este cambio | Regla |
|---|---|---|---|
| `routing.go`, bloque ODD | REQ-29.18 (remisión de bugs ISO) | Paso 7 (U2, U3) | Frases en pasos distintos; quien llega segundo rebasa y regenera goldens |
| `internal/workspace/declarations.go` | Bloque `sdd:` | Retirada de `baseline:` (U6) | U6 tras la fusión de carril |
| `internal/sddstatus/status.go`, `workspace_declarations.go` | Proyección ISO, bug sin delta | Sin baseline (U6) | Ídem |
| `internal/specbaseline/trace.go`, `ids.go` | Reutiliza | Borra (U6) | Paso 1 de U6 |
| Skills `sdd-*`, `_shared/*` | `Cubre:`, matriz | Adopción (U4), retirada (U6) | Conflictos de prosa a mano; goldens nunca a mano |
| Spec viva `spec-baseline-integration` | Su delta la modifica | REMOVED entera | Si carril archiva antes, la delta REMOVED de este cambio se rehace sobre su versión |

Procedimiento: `git rebase origin/main` en el worktree (sin `fetch` salvo autorización); en conflictos de `testdata/golden/**`, `AGENTS.md`, `GEMINI.md` y `.codex/AGENTS.md` se toma cualquiera de los lados y se regenera con `-update`; después, `go test` de los paquetes tocados y los dos ratchets.

## 9. Migración y despliegue

Sin migración de datos de Axiom. `integration.yaml` conserva formato (D3). Las secciones ODD nuevas son texto que un binario anterior ignora. Un `axiom.yaml` con `baseline:` queda bloqueado en el archivado tras U6, con la vía `axiom archive adopt`; reversión: `git revert` del commit de U6 (U1–U5 no dependen de él).

## 10. Preguntas abiertas

- [x] Nombre definitivo del campo ISO de carril y paquete destino de su traza `--first-parent`: resuelto en U6 (`Declarations.SDD.ISO`; la traza vive en `internal/isoevidence`).
- [ ] D2 se desvía del «delega» de la propuesta §5.1: el resultado observable es el mismo; si el usuario exige delegación real, U1 crece con un refactor de `specbaseline` que U6 borra.
