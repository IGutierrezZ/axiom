# Diseño: HTML explicativo de spec y design (`resumen-spec.html` y `resumen-design.html`)

Cambio C-06-b. Entradas: `proposal.md`, `exploration.md` y la spec delta de `specs/`. Trazabilidad con los requisitos provisionales:

| Requisito | Decisiones y secciones |
|---|---|
| prov:KOF:1 (declaración `sdd.explainer_html`), RF-ISO-01 (MODIFIED) | `internal/workspace` |
| prov:KOF:2 (campo sellado), prov:KOF:3 (bandera y resolución) | Decisiones 5 y 6; `internal/kickoff`, `internal/cli/sdd_kickoff.go` |
| prov:KOF:4 (proyección del valor de proyecto y del sellado) | Decisiones 3 y 4; `explainer.project`, `explainer.reason`, `governance.kickoff.explainerHtml` |
| prov:PRE:1 (interacción propia) | Doctrina del orquestador; tests del hook y del plugin de OpenCode |
| prov:GAT:1 (contrato del HTML), prov:GAT:2 (cita de IDs, aviso, regeneración) | Decisiones 1, 9 y 12; `internal/explainer`, documento compartido, skills y copias por agente |
| prov:GAT:3 (fuera del digest) | Decisión 10; test de `gateArtifactDigest` |
| prov:GAT:4 (frescura) | Decisiones 2, 3, 10 y 11; `sddstatus.ExplainerFreshness` |
| prov:WEB:1 (endpoint), prov:WEB:2 (pestaña) | Decisiones 7, 8, 11 y 13; dashboard |

## Enfoque técnico

Se sigue el enfoque 1 de la propuesta, en cuatro capas que no se mezclan:

1. **Declaración** (`internal/workspace`): `sdd.explainer_html: ask|always|never` en el decodificador estricto del bloque `sdd:`, con un accesor `Declarations.ExplainerHTML()` calcado de `Declarations.IDPattern()`.
2. **Instantánea** (`internal/kickoff` + `internal/cli/sdd_kickoff.go`): campo `explainer_html: yes|no` en `FlowConfig` con el patrón exacto de `DocsPublication` (tipo, mapa cerrado, bandera, rechazo con `--infer`, JSON), y una función pura `kickoff.ResolveExplainerHTML` que cruza la bandera con el valor de proyecto.
3. **Contrato del HTML** (paquete hoja nuevo `internal/explainer`): nombres de fichero, tipos (`spec`, `design`), estados (`fresh|stale|missing`), formato del digest, lectura acotada del fichero, lectura del `<meta>`, extracción de IDs citados y `Validate`. No importa nada del repositorio, de modo que `sddstatus`, `dashboard` y los tests de `internal/assets` lo usan sin ciclos.
4. **Proyección informativa** (`internal/sddstatus/explainer.go`, nuevo): una única función exportada `ExplainerFreshness(changeRoot, store)` que aplica la condición (kickoff sellado `yes` y almacén `openspec`/`hybrid`) y calcula la frescura; la llaman el decorador aditivo `decorateExplainer` (tras `decoratePublication`, `status.go:553`) y el dashboard. Calcula el digest esperado con `kickoff.ArtifactDigest` sobre `ChangeArtifactPaths` y lo proyecta como `expectedDigest`, que es lo que copia la fase al `<meta>`: el modelo nunca calcula un SHA-256.

La doctrina (orquestador, contrato de estado, fases y copias por agente) y el dashboard consumen esas cuatro piezas sin duplicar reglas.

## Decisiones de arquitectura

### Decisión 1: paquete hoja `internal/explainer`

**Elección**: un paquete nuevo, solo con la biblioteca estándar, que define el contrato del fichero (nombres, límite de tamaño, meta, IDs citados, validación).
**Alternativas consideradas**: ponerlo en `sddstatus` (los tests de `internal/assets` arrastrarían todo `sddstatus`); ponerlo en `kickoff` (mezcla gobierno sellado con formato de un fichero).
**Motivo**: tres consumidores (`sddstatus`, `dashboard`, tests de assets) necesitan la misma lectura acotada, el mismo `<meta>` y la misma lista blanca; un paquete hoja evita ciclos y una segunda copia de la regla.

### Decisión 2: el digest esperado lo proyecta `axiom sdd status`

**Elección**: `explainer.spec.expectedDigest` y `explainer.design.expectedDigest` con el valor literal que va en el `<meta>` (`sha256:<64 hex>`). La fase escribe el md, relee el estado, copia el valor y vuelve a leer el estado para confirmar `state: fresh`.
**Alternativas consideradas**: un verbo nuevo `axiom sdd explainer digest` (más superficie de CLI y de doctrina); que el modelo calcule SHA-256 con la shell (frágil entre plataformas y con la normalización CRLF→LF de `ArtifactDigest`).
**Motivo**: `axiom sdd status` ya es la fuente que leen las fases (patrón `iso.change`) y es de solo lectura; el digest sale de la misma función que juzga la frescura, así que no puede divergir.

### Decisión 3: presencia del bloque `explainer` como la del bloque `iso`

**Elección**: `explainer` se omite (nunca `null`) salvo que se cumpla una de estas condiciones: (a) `axiom.yaml` tiene bloque `sdd:` (válido o inválido) o es ilegible; (b) el cambio seleccionado, no archivado, está sellado con `explainer_html: yes`. La mera existencia de un `resumen-*.html` NO hace aparecer el bloque ni su frescura (prov:GAT:4). Ausente equivale a `project: ask` y a no haber frescura que informar. Con la declaración inválida o ilegible, `project` es `invalid` y `reason` lleva el error (incluido el de `sdd.explainer_html`) con su continuación; nunca se asume `ask` (prov:KOF:4).
**Alternativas consideradas**: proyectarlo siempre (cambia la salida de todo `status` y rompe el test de claves exactas `status_v2_clean_break_test.go:90`); hacerlo aparecer también por la existencia del HTML (lo prohíbe prov:GAT:4).
**Motivo**: es el precedente de `decorateISO` (`iso.go:538`, `ISOProjection.Reason`): un consumidor anterior no observa diferencias y la doctrina se escribe con una regla simple («ausente o `ask`: pregunta»).

### Decisión 4: valor sellado en `governance.kickoff.explainerHtml`

**Elección**: `kickoffV2.ExplainerHTML` (`json:"explainerHtml,omitempty"`), presente solo cuando el `kickoff.yaml` lleva la clave. Ausente equivale a `no`. Se proyecta en `continuous` y en `checkpointed` porque `governance` existe con cualquier kickoff sellado (`governance.go:432`).
**Alternativas consideradas**: duplicarlo en `explainer.change`.
**Motivo**: las fases ya leen el kickoff sellado de `governance.kickoff` (`docsPublication`); una sola fuente evita discrepancias.

### Decisión 5: resolución pura en `kickoff`, aviso en la CLI

**Elección**: `kickoff.ResolveExplainerHTML(project workspace.ExplainerHTMLMode, flag ExplainerHTML) (ExplainerHTML, bool, error)`; el booleano indica «se selló `no` por defecto con `ask`». La CLI imprime el aviso en stderr (costura `kickoffSealWarnings io.Writer = os.Stderr`) solo si `wrote` es verdadero, como `emitActorIdentityWarning`, tanto en salida de texto como con `--json` (el JSON de stdout no cambia de forma).
**Alternativas consideradas**: resolverlo en `ParseSealArgs` (no conoce `axiom.yaml`); en `ToKickoff` (mezcla E/S de declaraciones con el ensamblado puro).
**Motivo**: `kickoff` ya importa `workspace` (`infer.go`); la función es testeable por tabla y la CLI solo orquesta.

### Decisión 6: `never` y `ask` sin bandera escriben `no` explícito

**Elección**: un sellado sin `--infer` siempre escribe `explainer_html: yes` o `explainer_html: no`. Solo `--infer` y los kickoffs previos carecen de la clave.
**Alternativas consideradas**: omitir la clave cuando el resultado es `no`.
**Motivo**: deja constancia de que la decisión se tomó al sellar; la ausencia queda reservada a «sellado antes de esta función o retrosellado», y ambas lecturas equivalen a `no`. yaml.v3 entrecomilla al serializar el texto `no`, de modo que se relee como cadena.

### Decisión 7: endpoint en ruta propia con parámetros de consulta

**Elección**: `GET /api/explainer?change=<nombre>&kind=spec|design` (también `HEAD`), con cabeceras fijadas en el handler.
**Alternativas consideradas**: `/api/increments/explainer` (lo taparía un cambio llamado `explainer`, porque `/api/increments/` resuelve el resto como nombre, `server.go:534`); `/api/increments/<nombre>/explainer/<kind>` (obliga a reescribir el troceado de `handleIncrementDetail`).
**Motivo**: un espacio de nombres propio no colisiona con nombres de cambios; `r.URL.Query()` entrega valores ya decodificados, así que `%2e%2e`, `%2f` y `%5c` llegan como `..`, `/` y `\` y la validación los rechaza sin leer del disco.

### Decisión 8: CSP y `nosniff` en el handler, no en `withHeaders`

**Elección**: el handler fija, antes de cualquier respuesta (también las de error), `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; img-src data:`, `X-Content-Type-Options: nosniff`, `Cache-Control: no-store` y `Referrer-Policy: no-referrer`; en éxito, `Content-Type: text/html; charset=utf-8`.
**Alternativas consideradas**: añadirlas en `withHeaders` (`server.go:211`).
**Motivo**: `Router()` no pasa por `withHeaders`, así que `httptest.NewServer(server.Router())` las comprueba; y C-07-h reescribirá `withHeaders` sin chocar con este cambio.

### Decisión 9: doctrina del HTML en un documento compartido

**Elección**: `internal/assets/skills/_shared/sdd-explainer-html.md` (nuevo) con el contrato, un md de ejemplo en un bloque `markdown` y el esqueleto HTML que lo resume en un bloque `html`. `sdd-spec/SKILL.md` y `sdd-design/SKILL.md` añaden un paso que remite a él; las copias por agente repiten las restricciones esenciales (decisión 12). El test de assets extrae los dos bloques, pasa el `html` por `explainer.Validate` y contrasta sus IDs citados con los del md de ejemplo.
**Alternativas consideradas**: repetir el contrato completo en las dos skills (deriva segura); un fichero de ejemplo en `testdata` (no lo ve el modelo).
**Motivo**: una sola fuente para las dos fases; `SharedSkillFileNames` (`internal/assets/shared_skill_files.go:22`) despliega cualquier fichero nuevo de `_shared` sin listas literales; y el ejemplo que lee el modelo es el mismo que valida el test.

### Decisión 10: frescura fuera de las puertas y del encadenado

**Elección**: `decorateExplainer` no toca `NextRecommended`, `Dependencies`, `BlockedReasons`, `Notes`, `ISO`, `Governance.Gates` ni las reglas de archivado (tampoco en un cambio ISO). El digest de las puertas sigue siendo `gateArtifactDigest` (`internal/cli/sdd_gate_digest.go:40`), que no ve el HTML.
**Alternativas consideradas**: una nota en `notes` cuando el HTML está desactualizado.
**Motivo**: prov:GAT:3 y prov:GAT:4 lo declaran solo informativo; el campo `state` basta y no añade ruido a consumidores que leen `notes`.

### Decisión 11: una sola función de frescura para status y dashboard

**Elección**: `sddstatus.ExplainerFreshness(changeRoot string, store ArtifactStore) ExplainerFreshnessResult` lee el `kickoff.yaml` de la carpeta, aplica la condición (sellado `yes` y almacén `openspec` o `hybrid`) y calcula cada artefacto. El status le pasa `status.ArtifactStore`; el dashboard le pasa `sddstatus.DeclaredArtifactStore(planningHome)` (envoltorio exportado de `declaredArtifactStore`, `status.go:1002`) o `openspec` cuando el proyecto no declara almacén, que es lo que resuelve el status para un cambio en disco.
**Alternativas consideradas**: que cada consumidor evalúe la condición (dos reglas que pueden divergir); que el dashboard llame a `sddstatus.Resolve` (caro y con efectos de descubrimiento de worktrees).
**Motivo**: prov:WEB:2 exige el mismo cálculo; con una función que incluye la condición, las dos proyecciones solo pueden diferir por el almacén que recibe, que en ambos casos sale de la declaración.

### Decisión 12: las copias por agente exigen las restricciones

**Elección**: cada copia (`claude`, `cursor`, `copilot`; spec y design) lleva una sección corta «Explainer HTML» que exige, en el idioma del resto del fichero (inglés), las cuatro restricciones de prov:GAT:1 con sus términos literales: HTML escrito en castellano (`Spanish (castellano)`), autocontenido sin `<script>` ni recursos externos, `<meta name="axiom-source-digest">` con `expectedDigest`, y cita del ID de cada bloque con `data-axiom-ref`. Un test de assets recorre las dos skills, las seis copias y el documento compartido y falla si falta cualquiera de esos términos.
**Alternativas consideradas**: una línea que solo remita al SKILL (no cumple prov:GAT:1, que pide que las copias exijan las restricciones); escribir la sección en castellano (rompería la coherencia de idioma de copias escritas en inglés).
**Motivo**: la exigencia es comprobable por texto literal, y la sección replica solo las cuatro restricciones; el detalle sigue en el documento compartido.

### Decisión 13: la pestaña distingue activación, ausencia de md y frescura

**Elección**: el DTO del detalle lleva `explainer: {sealed, enabled, error?, items[]}`. Con `enabled: false` la pestaña solo dice «La explicación no está activada para este cambio» y no crea ningún `iframe`. Con `enabled: true`, por tipo: sin md de origen (`state` vacío), «Todavía no hay documento de origen; el resumen se generará con él»; `missing`, «No hay explicación generada»; `stale`, aviso de desactualizado y `iframe`; `fresh`, `iframe`. Un cambio archivado se trata igual que uno activo: el dashboard calcula su frescura con la misma función sobre la carpeta archivada (que conserva `kickoff.yaml`, md y HTML) y el endpoint lo sirve; `axiom sdd status` no proyecta frescura de un cambio archivado (como `iso.change`, `iso.go:501`), así que no hay dos valores que comparar.
**Alternativas consideradas**: ocultar la pestaña sin activación (prov:WEB:2 pide indicar que no está activada); no mostrar archivados (el HTML sigue siendo útil para leer un cambio cerrado).
**Motivo**: cubre los cuatro escenarios de prov:WEB:2 y el caso sin md, sin ningún `iframe` roto.

## Flujo de datos

```
axiom.yaml  sdd.explainer_html ──► workspace.Declarations.ExplainerHTML()
                                          │
            ┌─────────────────────────────┴───────────────────────────┐
            ▼                                                         ▼
 axiom sdd kickoff seal [--explainer-html]                axiom sdd status --json
   kickoff.ResolveExplainerHTML ──► kickoff.yaml            explainer.project / explainer.reason
   (always→yes, never→no, contradicción→error,              governance.kickoff.explainerHtml
    ask sin bandera→no + aviso stderr)                      explainer.{spec,design} (solo con yes)
                                                                     ▲            │
 Orquestador: explainer ausente o ask ─► pregunta                    │            ▼
   «Axiom SDD kickoff explainer: …» ─► --explainer-html        sdd-spec / sdd-design (con yes):
                                                               md → status → copia expectedDigest
                                                               → resumen-*.html → status (fresh)
                                                                     │
 sddstatus.ExplainerFreshness(changeRoot, store) ◄── decorateExplainer│
            ▲                                                         │
 Dashboard: GetIncrementDetail ───────────────────────────────────────┘
            GET /api/explainer?change&kind ─► lista blanca ─► explainer.ReadFile ─► HTML con CSP ─► <iframe sandbox="">
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---------|--------|-------------|
| `internal/workspace/sdd.go` | Modificar | Tipo `ExplainerHTMLMode`, campo `SDDConfig.ExplainerHTML`, clave en `sddKeys`, `sddExplainerHTML`, mensaje del bloque con la clave nueva |
| `internal/workspace/declarations.go` | Modificar | Accesor `Declarations.ExplainerHTML()` |
| `internal/workspace/sdd_test.go` | Modificar | Valores válidos, inválidos, nulo, lista, booleano, duplicado, defecto `ask`, independencia de `iso` y `workflow.lane` |
| `axiom.example.yaml` | Modificar | `#   explainer_html: ask` y su explicación con los tres valores en el bloque SDD |
| `internal/workspace/example_config_test.go` | Modificar | El bloque descomentado declara `explainer_html: ask` |
| `internal/kickoff/types.go` | Modificar | Tipo `ExplainerHTML`, constantes, campo `FlowConfig.ExplainerHTML`, `EffectiveExplainerHTML` |
| `internal/kickoff/schema.go` | Modificar | Mapa `validExplainerHTMLs` y validación en `Validate` |
| `internal/kickoff/args.go` | Modificar | `SealArgs.ExplainerHTML`, bandera `--explainer-html`, rechazo con `--infer`, copia en `ToKickoff` |
| `internal/kickoff/explainer.go` | Crear | `ResolveExplainerHTML` |
| `internal/kickoff/{args,schema,infer}_test.go`, `internal/kickoff/explainer_test.go` | Modificar/Crear | Bandera, validación, `--infer` sin campo, resolución por tabla, lectura heredada |
| `internal/cli/sdd_kickoff.go` | Modificar | Ayuda, resolución tras la guarda de `SDDErr`, aviso, resumen y `kickoffJSON.ExplainerHTML` |
| `internal/cli/sdd_kickoff_test.go` | Modificar | always/never/contradicción/ask con y sin bandera/carpeta nueva/JSON/ayuda |
| `internal/explainer/explainer.go` | Crear | Contrato del fichero: `Kind`, `State`, `FileName`, `MaxBytes`, `ReadFile`, `SourceDigest`, `FormatDigest`, `CitedIDs`, `Validate` |
| `internal/explainer/explainer_test.go` | Crear | Tablas de `Validate`, `SourceDigest`, `CitedIDs` y `ReadFile` |
| `internal/sddstatus/explainer.go` | Crear | `ExplainerProjection`, `ExplainerFreshness`, `DeclaredArtifactStore`, `decorateExplainer` |
| `internal/sddstatus/status.go` | Modificar | Campo `Status.Explainer`; llamada a `decorateExplainer` en `Resolve` |
| `internal/sddstatus/status_v2.go` | Modificar | Campo `StatusV2Projection.Explainer`, copia en `ProjectStatusV2`, `kickoffV2.ExplainerHTML` |
| `internal/sddstatus/explainer_test.go` | Crear | Frescura, condición de sellado y almacén, sin md, errores, límite de tamaño, ISO, `nextRecommended` intacto, `continuous`, v2 |
| `internal/cli/sdd_gate_artifact_digest_test.go` | Modificar | El HTML no altera los digests `spec` ni `design` |
| `internal/cli/sdd_preflight_hook_test.go` | Modificar | Gemelo de `:745` con el marcador `Axiom SDD kickoff explainer:` |
| `internal/assets/opencode_sdd_preflight_topology_test.go` | Modificar | El plugin (v1 y v2) deja fuera el marcador nuevo; escenario en el arnés de node |
| `internal/dashboard/server.go` | Modificar | Ruta `/api/explainer`, `handleExplainer`, `validExplainerChangeName`, constante `explainerCSP` |
| `internal/dashboard/service.go` | Modificar | `ReadExplainer`, `ErrExplainerNotFound`, `Explainer` en `GetIncrementDetail` |
| `internal/dashboard/types.go` | Modificar | `IncrementExplainerDTO`, `ExplainerDTO` e `IncrementDetailDTO.Explainer` |
| `internal/dashboard/explainer_test.go` | Crear | Cabeceras, lista blanca, traversal, 404, límite de tamaño, DTO, coincidencia con status, archivado |
| `internal/dashboard/assets/index.html` | Modificar | Pestaña y panel `explicacion` |
| `internal/dashboard/assets/js/core/router.js` | Modificar | `INCREMENT_TABS` con `'explicacion'` |
| `internal/dashboard/assets/js/screens/detalle.js` | Modificar | Rama `explicacion` en `renderTabContent` |
| `internal/dashboard/assets/js/screens/detalle-explicacion.js` | Crear | `renderExplainer(panel, { change, detail })` |
| `internal/dashboard/{workflow_assets,assets}_test.go` | Modificar | La pestaña nueva, los textos de estado y el `sandbox` sin `allow-*` |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Párrafo **Explainer HTML.** al final de `SDD Change Kickoff and Block Gates` |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | `kickoff.explainerHtml` y bloque `explainer` |
| `internal/assets/skills/_shared/sdd-explainer-html.md` | Crear | Contrato del HTML, md de ejemplo y esqueleto |
| `internal/assets/skills/sdd-{spec,design}/SKILL.md` | Modificar | Paso del HTML explicativo |
| `internal/assets/{claude,cursor}/agents/sdd-{spec,design}.md`, `internal/assets/copilot/agents/sdd-{spec,design}.agent.md` | Modificar | Sección «Explainer HTML» con las cuatro restricciones |
| `internal/assets/explainer_html_doctrine_test.go` | Crear | Fija la doctrina en skills, copias y documento compartido; valida el esqueleto y sus IDs |
| `testdata/golden/*` (10 de orquestador + 2 de agentes) | Regenerar | Ver «Goldens» |

El plugin `internal/assets/opencode/plugins{,-v2}/sdd-task-result-artifacts.ts` NO cambia: su filtro `isSDDPreflightQuestion` solo acepta textos que empiezan por `SDD_PREFLIGHT_QUESTION_PREFIX = "Axiom SDD preflight "`, así que el marcador nuevo queda fuera. Tampoco cambia `internal/cli/sdd_preflight_hook.go` (`hasSDDPreflightQuestionPrefix`, `:391`).

## Interfaces y contratos

### `internal/workspace`

```go
// ExplainerHTMLMode es `sdd.explainer_html`: si los cambios SDD del proyecto generan el HTML explicativo.
type ExplainerHTMLMode string

const (
	ExplainerHTMLAsk    ExplainerHTMLMode = "ask"    // defecto: se pregunta una vez por cambio
	ExplainerHTMLAlways ExplainerHTMLMode = "always" // se sella yes sin preguntar
	ExplainerHTMLNever  ExplainerHTMLMode = "never"  // se sella no sin preguntar
)

type SDDConfig struct {
	ISO       bool
	IDPattern *reqid.Pattern
	// ExplainerHTML es lo declarado; vacío si la clave no está (rige ask).
	ExplainerHTML ExplainerHTMLMode
}

var sddKeys = []string{"iso", "id_pattern", "explainer_html"}

// sddExplainerHTML admite solo un escalar !!str con ask, always o never.
// Error: "sdd.explainer_html (línea %d): el valor %q no es válido: usa ask, always o never"
// (también para nulo, lista, mapeo o booleano sin comillas).
func sddExplainerHTML(line int, node *yaml.Node) (ExplainerHTMLMode, error)

// ExplainerHTML devuelve el modo efectivo: ask sin bloque o sin clave; error que envuelve
// ParseErr o SDDErr (%w) si la declaración no se pudo leer, como IDPattern().
func (d Declarations) ExplainerHTML() (ExplainerHTMLMode, error)
```

`decodeSDD` añade la rama `explainer_html` antes de la de `sddBool`. Un valor no admitido entra en `SDDErr`, de modo que `axiom workspace validate` y `kickoff seal` (guarda existente en `sdd_kickoff.go:88`) lo rechazan con el mismo mensaje y sin escribir nada.

### `internal/kickoff`

```go
// ExplainerHTML es la decisión sellada de generar el HTML explicativo. Vacía (kickoff previo o
// retrosellado) equivale a ExplainerHTMLNo.
type ExplainerHTML string

const (
	ExplainerHTMLYes ExplainerHTML = "yes"
	ExplainerHTMLNo  ExplainerHTML = "no"
)

// En FlowConfig, tras DocsPublication:
ExplainerHTML ExplainerHTML `yaml:"explainer_html,omitempty"`

func EffectiveExplainerHTML(v ExplainerHTML) ExplainerHTML // "" → no

// ResolveExplainerHTML cruza la bandera con el valor de proyecto:
//   always + (""|yes) → yes        always + no  → error
//   never  + (""|no)  → no         never  + yes → error
//   ask    + yes|no   → la bandera
//   ask    + ""       → no, defaulted = true (la CLI avisa)
// Error: "--explainer-html %s contradice sdd.explainer_html: %s de axiom.yaml; omite la bandera o cambia la declaración"
func ResolveExplainerHTML(project workspace.ExplainerHTMLMode, flag ExplainerHTML) (value ExplainerHTML, defaulted bool, err error)
```

- `schema.go`: `validExplainerHTMLs = map[ExplainerHTML]bool{yes, no}`; en `Validate`, un valor no vacío fuera del mapa da `kickoff: explainer_html desconocido %q; admitidos: yes, no`. Sin regla de estilo: vale en `continuous` y `checkpointed`.
- `args.go`: `case "--explainer-html"` con `requireFlagValue`; valor fuera del mapa → `--explainer-html desconocido %q; opciones: yes, no`. En el bloque `if parsed.Infer`, junto al de `--docs-publication`: `--explainer-html no se puede combinar con --infer: la decisión del HTML explicativo nunca se infiere`. `ToKickoff` copia `a.ExplainerHTML`.
- `infer.go` no cambia: `inferredKickoff` no escribe el campo (lo fija un test, también con `always` en el proyecto).
- Lectura heredada: test `TestLegacyBinaryReadsAKickoffWithExplainerHTML` con un struct congelado sin el campo (patrón de `ledger_test.go:185`), y `Load` de un `kickoff.yaml` sin la clave → `""` y `EffectiveExplainerHTML` → `no`.

### `internal/cli/sdd_kickoff.go`

En `runSDDKickoffSeal`, rama sin `--infer`, justo después de la guarda de `loc.Declarations.SDDErr` y antes de `applySealBranchPolicy` (un rechazo no crea la carpeta, no escribe nada ni emite avisos de rama):

```go
mode, err := loc.Declarations.ExplainerHTML()
if err != nil { return fmt.Errorf("no se sella el cambio %q: %w", parsed.Change, err) }
value, defaulted, err := kickoff.ResolveExplainerHTML(mode, parsed.ExplainerHTML)
if err != nil { return fmt.Errorf("no se sella el cambio %q: %w", parsed.Change, err) }
sealing.Config.ExplainerHTML = value
```

Tras `sealKickoffFile`, si `wrote && defaulted`: `fmt.Fprintln(kickoffSealWarnings, "aviso: sdd.explainer_html es ask y no se ha preguntado: se sella explainer_html: no; para activar el HTML explicativo, pregunta antes de sellar y pasa --explainer-html yes")`. Ayuda: `[--explainer-html yes|no]` en la línea de `seal` y una línea explicativa. Resumen de texto: ` explainer_html=<v>` cuando no está vacío. `kickoffJSON.ExplainerHTML string json:"explainerHtml,omitempty"`.

### `internal/explainer`

```go
package explainer

type Kind string   // KindSpec = "spec", KindDesign = "design"
type State string  // StateFresh = "fresh", StateStale = "stale", StateMissing = "missing"

const (
	MetaName     = "axiom-source-digest"
	DigestPrefix = "sha256:"
	RefAttribute = "data-axiom-ref"
	MaxBytes     = 5 << 20 // límite común del endpoint y de la frescura
)

var (
	ErrTooLarge   = errors.New("el resumen supera el límite de 5 MiB")
	ErrNotRegular = errors.New("el resumen no es un fichero regular")
)

func Kinds() []Kind                        // [spec, design], en este orden
func ValidKind(k Kind) bool
func FileName(k Kind) string               // spec → "resumen-spec.html", design → "resumen-design.html"; "" si no es válido
func FormatDigest(hexDigest string) string // DigestPrefix + hex

// ReadFile exige fichero regular (os.Lstat; un enlace simbólico da ErrNotRegular), lee como
// máximo MaxBytes+1 y devuelve ErrTooLarge sin contenido si se supera; nunca trunca. Un fichero
// inexistente devuelve un error que cumple errors.Is(err, fs.ErrNotExist).
func ReadFile(path string) ([]byte, error)

// SourceDigest devuelve el content del primer <meta name="axiom-source-digest">, recortado y en minúsculas.
func SourceDigest(html []byte) (string, bool)

// CitedIDs devuelve, en orden de aparición y sin repetir, los valores de data-axiom-ref.
func CitedIDs(html []byte) []string

type Finding struct{ Code, Detail string }
// Códigos: script_tag, external_url, digest_meta_missing, approval_notice_missing, cited_ids_missing.
func Validate(html []byte) []Finding
```

Expresiones (insensibles a mayúsculas, orden de atributos libre, comillas simples o dobles):

```go
var metaTag     = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
var metaName    = regexp.MustCompile(`(?is)\bname\s*=\s*["']axiom-source-digest["']`)
var metaContent = regexp.MustCompile(`(?is)\bcontent\s*=\s*["']([^"']*)["']`)
var digestValue = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var refAttr     = regexp.MustCompile(`(?is)\bdata-axiom-ref\s*=\s*["']([^"']+)["']`)
```

`Validate`:

- `script_tag`: `(?i)<script\b`.
- `external_url`: `(?is)\b(?:src|href|xlink:href|action|poster|srcset)\s*=\s*["']?\s*(?:https?:)?//`, y en CSS `(?i)url\(\s*["']?\s*(?:https?:)?//` y `(?i)@import\b`. `data:` y anclas `#...` se admiten. Cubre hojas de estilo, fuentes, imágenes y marcos externos.
- `digest_meta_missing`: sin `SourceDigest` o con un valor que no casa `digestValue`.
- `approval_notice_missing`: el texto no contiene `lo que se aprueba es el md` (insensible a mayúsculas, tras colapsar espacios).
- `cited_ids_missing`: `CitedIDs` vacío.

`Validate` no se ejecuta en producción (un verbo de validación queda fuera del alcance): la usan los tests.

### `internal/sddstatus/explainer.go`

```go
// ExplainerProjection es Status.Explainer. Informativa: nunca cambia nextRecommended,
// las dependencias, blockedReasons, la proyección iso ni el digest de una puerta.
type ExplainerProjection struct {
	Project string                       `json:"project"`          // ask|always|never|invalid
	Reason  string                       `json:"reason,omitempty"` // solo con invalid: error y continuación
	Error   string                       `json:"error,omitempty"`  // fallo al descubrir los md del cambio
	Spec    *ExplainerArtifactProjection `json:"spec,omitempty"`
	Design  *ExplainerArtifactProjection `json:"design,omitempty"`
}

type ExplainerArtifactProjection struct {
	State          string `json:"state,omitempty"`          // fresh|stale|missing; vacío solo junto a error
	Path           string `json:"path"`                     // ruta absoluta de resumen-*.html
	ExpectedDigest string `json:"expectedDigest,omitempty"` // sha256:<hex> de los md actuales
	Error          string `json:"error,omitempty"`          // HTML ilegible, mayor que 5 MiB o md ilegible
}

// ExplainerFreshnessResult es el resultado común de status y dashboard.
type ExplainerFreshnessResult struct {
	Sealed  kickoff.ExplainerHTML        // efectivo: yes o no ("" del fichero → no)
	Enabled bool                         // Sealed == yes y store openspec|hybrid
	Error   string                       // kickoff ilegible o ChangeArtifactPaths falló
	Spec    *ExplainerArtifactProjection // nil sin md de origen o con Enabled falso
	Design  *ExplainerArtifactProjection
}

func ExplainerFreshness(changeRoot string, store ArtifactStore) ExplainerFreshnessResult
func DeclaredArtifactStore(workspaceRoot string) (ArtifactStore, bool) // envoltorio de declaredArtifactStore
func decorateExplainer(status *Status, location workspace.Location)
```

Regla por tipo, solo con `Enabled` (fuentes: `ChangeArtifactPaths(changeRoot).Specs` para `spec`, `.Design` para `design`):

| Fuentes md | HTML (`explainer.ReadFile`) | `<meta>` | Proyección |
|---|---|---|---|
| ninguna | cualquiera | — | ninguna (el tipo se omite) |
| existen pero ilegibles | cualquiera | — | `error` con el motivo, sin `state` |
| existen | no existe | — | `missing` + `expectedDigest` |
| existen | `ErrTooLarge`, `ErrNotRegular` u otro error | — | `error` con el motivo + `expectedDigest`, sin `state` |
| existen | existe | ausente, vacío o mal formado | `stale` |
| existen | existe | distinto de `expectedDigest` | `stale` |
| existen | existe | igual | `fresh` |

Tratamiento de fallos: si `kickoff.Load` o `ChangeArtifactPaths` fallan, `Error` lleva el mensaje y no hay tipos; `decorateExplainer` lo copia a `explainer.error`. Ningún fallo se convierte en `blockedReasons`, `notes` ni error de `axiom sdd status`.

`decorateExplainer`:

1. `project`: `invalid` (con `reason`, como `isoDeclaration.reason`, `iso.go:463`) si `Declarations.ExplainerHTML()` falla; si no, el modo efectivo.
2. `declared := Declarations.SDD != nil || ParseErr != nil || SDDErr != nil`.
3. Solo con cambio seleccionado no archivado y `status.ChangeRoot != nil`: `r := ExplainerFreshness(*status.ChangeRoot, status.ArtifactStore)`.
4. Si `!declared && !r.Enabled && r.Error == ""`: no escribe nada.
5. Copia `r.Error`, `r.Spec` y `r.Design` (nil sin activación).

Forma JSON (cambio sellado `yes`, spec al día, design aún sin md):

```json
"explainer": {
  "project": "ask",
  "spec": { "state": "fresh", "path": "C:/.../changes/x/resumen-spec.html", "expectedDigest": "sha256:9f2c…" }
},
"governance": { "kickoff": { "...": "...", "explainerHtml": "yes" } }
```

Declaración inválida:

```json
"explainer": { "project": "invalid", "reason": "sdd.explainer_html (línea 3): el valor \"quizás\" no es válido: usa ask, always o never; corrígela con `axiom workspace validate --path …`" }
```

### Dashboard

```go
// server.go
const explainerCSP = "default-src 'none'; style-src 'unsafe-inline'; img-src data:"
var explainerChangeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$`)

// validExplainerChangeName: casa el patrón y no contiene "..". Rechaza vacío, ".", "..",
// "/", "\", ":", espacios, NUL y cualquier escape ya decodificado (%2e%2e, %2f, %5c).
func validExplainerChangeName(name string) bool

// GET|HEAD /api/explainer?change=<n>&kind=spec|design
// 405 otro método · 400 nombre o kind inválidos (sin tocar el disco) · 404 cambio o fichero
// inexistentes o no regular · 500 con cuerpo "el resumen supera el límite de 5 MiB" ante
// explainer.ErrTooLarge, sin ningún byte del fichero · 500 otra lectura fallida
func (s *Server) handleExplainer(w http.ResponseWriter, r *http.Request)

// service.go
var ErrExplainerNotFound = errors.New("resumen explicativo no encontrado")

// ReadExplainer resuelve el cambio con locateIncrement, construye la ruta con
// filepath.Join(location.path, explainer.FileName(kind)) (nunca con texto del cliente) y
// lee con explainer.ReadFile. fs.ErrNotExist y ErrNotRegular → ErrExplainerNotFound.
func (s *Service) ReadExplainer(name string, kind explainer.Kind) ([]byte, error)

// types.go
type IncrementExplainerDTO struct {
	Sealed  string         `json:"sealed"`          // yes|no
	Enabled bool           `json:"enabled"`         // sealed yes y almacén de ficheros
	Error   string         `json:"error,omitempty"`
	Items   []ExplainerDTO `json:"items"`           // spec y design con Enabled; [] sin activación
}
type ExplainerDTO struct {
	Kind  string `json:"kind"`            // spec|design
	State string `json:"state,omitempty"` // fresh|stale|missing; vacío sin md de origen o con error
	Error string `json:"error,omitempty"`
}
// IncrementDetailDTO: Explainer IncrementExplainerDTO `json:"explainer"`
```

`GetIncrementDetail` rellena `Explainer` con `sddstatus.ExplainerFreshness(path, store)`, para cambios activos y archivados; `store` es `DeclaredArtifactStore(loc.PlanningHome)` o `openspec` si no se declara.

Frontend (`detalle-explicacion.js`, `renderExplainer(panel, { change, detail })`):

- `enabled: false` → solo «La explicación no está activada para este cambio.», sin `iframe`.
- `error` global → «No se pudo calcular la explicación: <error>», sin `iframe`.
- Por cada `item` («Resumen de la spec» / «Resumen del diseño»): sin `state` ni `error`, «Todavía no hay documento de origen; el resumen se generará con él.»; con `error`, el mensaje, sin `iframe`; `missing`, «No hay explicación generada.», sin `iframe`; `stale`, aviso «Este resumen está desactualizado respecto del md. Lo que se aprueba es el md.» y el `iframe`; `fresh`, el `iframe`.

```html
<iframe sandbox="" referrerpolicy="no-referrer" loading="lazy" title="Resumen de la spec"
        src="/api/explainer?change=<encodeURIComponent(change)>&kind=spec"></iframe>
```

Sin `allow-same-origin` ni `allow-scripts`; el atributo se escribe literal `sandbox=""`. Sin controles de edición, guardado ni regeneración. La pestaña «Explicación» (`data-tab="explicacion"`) va tras «Handoff» y siempre es visible.

### Doctrina

**Orquestador** (`sdd-orchestrator-sections.md`): un párrafo nuevo, **Explainer HTML.**, como último párrafo de `SDD Change Kickoff and Block Gates`, justo antes del marcador `:end`, sin tocar los párrafos existentes. Contenido obligatorio:

- Cuándo: cuando `explainer` falta en `axiom sdd status` o proyecta `project: ask`; una sola vez por cambio, después de `axiom branch ensure` y antes de `axiom sdd kickoff seal`; nunca en ODD, nunca con `always`, `never` o `invalid`, nunca con `--infer`, nunca al reanudar y nunca otra vez con el cambio sellado.
- Forma: una interacción propia, una sola pregunta de selección única cuyo texto empieza por `Axiom SDD kickoff explainer:` (el marcador va en el texto de la pregunta, que es lo que inspeccionan el hook y el plugin; la cabecera corta es libre); nunca es un grupo del pre-vuelo y nunca reutiliza el prefijo `Axiom SDD preflight `.
- Opciones, en este orden, con etiqueta y descripción traducidas y el token canónico entre paréntesis: (1) «Sí, generar el resumen (`yes`)»: «`sdd-spec` y `sdd-design` escriben además `resumen-spec.html` y `resumen-design.html`, un resumen con diagramas para lectores no técnicos que se ve en el dashboard. Cuesta tokens adicionales en las dos fases; lo que se aprueba sigue siendo el md.»; (2) «No (`no`)»: «Solo se escribe el md.»
- Traducción: la respuesta se pasa como `--explainer-html yes|no` a `kickoff seal`; con `always` o `never` no se pasa bandera.
- Uso posterior: las fases leen `governance.kickoff.explainerHtml` (ausente es `no`); con `yes`, cada parada de `spec` y `design` en `checkpointed` presenta también su HTML con su `state`, nunca en lugar del md; la frescura nunca cambia `nextRecommended`, una puerta ni su digest.

**Contrato de estado** (`sdd-status-contract.md`): `explainerHtml?` en la línea de `kickoff` de la sección «Governance» y una viñeta con su semántica; una subsección `explainer` con `project`, `reason`, `error`, `spec`/`design` (`state`, `path`, `expectedDigest`, `error`), la regla de presencia de la decisión 3, la condición de la decisión 11, la omisión sin md de origen y la frase de que es informativo también en un cambio ISO.

**Fases** (`sdd-spec/SKILL.md` tras «Step 5: Persist Artifact»; `sdd-design/SKILL.md` tras «Step 4: Persist Artifact»): paso «Explainer HTML (only with `governance.kickoff.explainerHtml: yes`)»: con almacén `openspec` o `hybrid`, tras escribir el md, ejecutar `axiom sdd status --cwd "<workspaceRoot>" <change> --json`, copiar literal `explainer.<kind>.expectedDigest` al `<meta name="axiom-source-digest">`, escribir `<changeRoot>/resumen-<kind>.html` según `../_shared/sdd-explainer-html.md` (castellano, autocontenido sin `<script>` ni recursos externos, `data-axiom-ref` con el ID de cada bloque, aviso de que lo que se aprueba es el md), releer el estado y corregir hasta `state: fresh`; incluir la ruta en `artifacts`. Con `engram` o `none`, no escribir HTML y decirlo en el resultado. Cuando la fase se reabre por un rechazo o una corrección, regenerar el md y después el HTML. Una regla en «Rules» que lo resume.

**Documento compartido** (`sdd-explainer-html.md`): público no técnico; castellano neutro; secciones (qué cambia, para quién, flujo antes/después, diagrama SVG en línea, glosario); cada bloque lleva `data-axiom-ref="<ID>"` y muestra el ID (definitivo, provisional `prov:<serie>:<n>` o título de la decisión), y solo cita IDs que existen en el md; nada que no esté en el md; aviso literal «Este resumen no sustituye al documento: lo que se aprueba es el md.»; autocontenido (HTML, CSS y SVG en línea; imágenes solo `data:`); sin `<script>`, sin manejadores `on*`, sin URL `http://`, `https://` ni `//` en `src`, `href`, `xlink:href`, `srcset` ni CSS, sin `@import`; el `<meta>` con el valor literal de `expectedDigest`; menos de 5 MiB; regeneración tras cualquier cambio del md. Termina con un md de ejemplo (bloque `markdown`, dos requisitos con ID) y su esqueleto (bloque `html`) que cumple todo lo anterior.

**Agentes** (`claude`, `cursor`, `copilot`; spec y design): sección «Explainer HTML» tras «Instructions», en inglés como el resto del fichero, con este contenido (cambiando el nombre del fichero): «When `governance.kickoff.explainerHtml` is `yes`, after the md write `resumen-spec.html` as the skill's explainer step says: written in Spanish (castellano) for non-technical readers; self-contained, with no `<script>` and no external resources; with `<meta name="axiom-source-digest">` set to `expectedDigest` from `axiom sdd status`; and every block citing its requirement ID with `data-axiom-ref`.»

## Estrategia de pruebas

Todas localizadas por paquete y con `-run`; sin suite completa y sin compilar binarios sueltos (solo `go test`, que construye en proceso). Con TDD, cada tarea parte de su test en rojo.

| Capa | Qué se prueba | Cómo |
|------|---------------|------|
| Unidad | `sdd.explainer_html`: `ask`, `always`, `never`; `quizás`, `true`, nulo, lista, duplicado → error con clave, valor y admitidos; sin clave → `ask`; independencia de `iso`/`workflow.lane`; `Declarations.ExplainerHTML()` con `SDDErr`/`ParseErr` → error | `go test ./internal/workspace -run 'TestDecodeSDD|TestSDDLoad|TestExample|TestDeclarationsExplainer'` |
| Unidad | Bandera, valor inválido, `--infer` + bandera, `ToKickoff`, `Validate` (`quizás` nombra `yes, no`), `ResolveExplainerHTML` por tabla, `--infer` sin campo, lectura heredada | `go test ./internal/kickoff -run 'Explainer|TestParseSealArgs|TestInfer|TestLegacy'` |
| Integración CLI | `seal` con always/never/ask (con y sin bandera), contradicción sin crear la carpeta, carpeta nueva con `yes`, aviso solo al escribir (costura `kickoffSealWarnings`), JSON `explainerHtml`, ayuda | `go test ./internal/cli -run 'TestRunSDDKickoffSeal.*Explainer'` |
| Unidad | `Validate` (script en mayúsculas, `src="https://"`, `href='//x'`, `xlink:href`, `srcset`, `url(//x)`, `@import`, `data:` admitido, meta ausente/vacía/mal formada, aviso ausente, sin `data-axiom-ref`); `SourceDigest` (orden de atributos, comillas, varios meta); `CitedIDs` (orden, sin repetidos); `ReadFile` (inexistente, enlace simbólico, exactamente `MaxBytes` admitido, `MaxBytes+1` → `ErrTooLarge` sin contenido) | `go test ./internal/explainer` |
| Integración status | con `yes`: fresh → stale al editar el md → missing al borrar; meta vacía → stale; sin md de origen → tipo omitido; HTML de `MaxBytes+1` → `error` sin `state`; `specs/` ilegible o kickoff corrupto → `explainer.error`; con `no`, sin campo o con almacén `engram`/`none` no hay frescura aunque exista el HTML; `continuous` proyecta; `nextRecommended`, `dependencies` y `blockedReasons` iguales con `yes` y con `no` y con y sin HTML; valor de proyecto sin cambio seleccionado; `invalid` con `reason`; `governance.kickoff.explainerHtml`; copia en v2; claves exactas intactas | `go test ./internal/sddstatus -run 'Explainer|TestSDDStatusV2CleanBreak'` |
| ISO | cambio ISO sellado `yes` con HTML `stale` y con HTML `missing`: ninguna entrada de `blockedReasons` lo cita, `dependencies.archive` y `iso` son idénticos a los del mismo cambio con HTML `fresh` | `go test ./internal/sddstatus -run 'TestExplainerISO'` |
| Puertas | crear, modificar y borrar `resumen-spec.html`/`resumen-design.html` no cambia `gateArtifactDigest` de `spec` ni de `design`; modificar el md sí | test nuevo en `sdd_gate_artifact_digest_test.go`: `go test ./internal/cli -run 'TestGateArtifactDigest.*Explainer'` |
| Hook | gemelo de `TestSDDPreflightHookIgnoresIndependentReviewPreferenceInteraction` con una pregunta cuyo texto empieza por `Axiom SDD kickoff explainer:`: autoridad intacta | `go test ./internal/cli -run 'TestSDDPreflightHookIgnores'` |
| Plugin OpenCode | estático (v1 y v2): `SDD_PREFLIGHT_ACCEPTED_PREFIXES = [SDD_PREFLIGHT_QUESTION_PREFIX]`, el prefijo es `"Axiom SDD preflight "`, el marcador nuevo no empieza por él y el fuente no lo menciona; arnés de node (se omite sin node, como hoy): despachar tras una pregunta `Axiom SDD kickoff explainer:` produce el mismo bloque que sin ella | `go test ./internal/assets -run 'TestOpenCodePluginIgnoresExplainerQuestion|TestOpenCodeSDDPreflightTopologyContract'` |
| HTTP | 200 con `Content-Type`, CSP exacta, `nosniff`, `no-store`; CSP y `nosniff` también en 400/404; 400 con `..`, `../x`, `x/..`, `a/b`, `a\b`, `%2e%2e`, `a%2fb`, `%5c`, `C:x`, `%00`, vacío; 400 con `kind` fuera de lista (incluidos `spec.md`, `kickoff.yaml`, `tasks`); 404 sin fichero, sin cambio o con enlace simbólico (se omite si la plataforma no permite crearlo); 405 con POST, PUT y DELETE; HTML de `MaxBytes+1` → 500 con el mensaje y sin bytes del fichero; un directorio hermano con `resumen-spec.html` nunca se sirve | `go test ./internal/dashboard -run 'Explainer'` con `httptest.NewServer(server.Router())` y `t.Parallel()`; ningún test nuevo usa `freePort` ni el hub |
| DTO | `enabled` falso con `no` y sin campo (`items: []`); con `yes`, estados iguales a los de `sddstatus.ExplainerFreshness` para el mismo directorio; sin md → `state` vacío; cambio archivado con `yes` → mismo cálculo y endpoint 200 | `go test ./internal/dashboard -run 'TestIncrementDetailExplainer'` |
| Assets web | pestaña y panel `explicacion`, `'explicacion'` en `router.js`; en `detalle-explicacion.js`: `sandbox=""`, sin `allow-same-origin` ni `allow-scripts`, textos «no está activada», «Todavía no hay documento de origen», «No hay explicación generada» y «desactualizado», sin controles de edición | `go test ./internal/dashboard -run 'TestIncrementDetailTabsAndRoutes|TestRouterDefinesEveryRoute|TestExplainerTab'` |
| Doctrina | orquestador: marcador, una vez, antes de `kickoff seal`, nunca en ODD, `--explainer-html`; contrato de estado: `explainerHtml`, `expectedDigest`, `reason`, `fresh|stale|missing`; skills, seis copias y documento compartido contienen `castellano`, `<script>`, `external`, `axiom-source-digest` y `data-axiom-ref` (falla si falta uno en cualquiera); esqueleto `html` sin hallazgos de `explainer.Validate`; `CitedIDs(html)` no vacío, cada ID citado existe en el md de ejemplo y cada requisito del md de ejemplo está citado | `go test ./internal/assets -run 'Explainer|TestOrchestratorSectionDriftDoesNotGrow|TestSDDISOConvention|TestSDDSkillsReferenceISOConvention'` |
| Goldens | ver abajo | `go test ./internal/components -run 'TestGolden'` |

## Goldens

Contienen la sección del kickoff y cambian con el párrafo nuevo (10): `testdata/golden/combined-claude-claudemd.golden`, `sdd-antigravity-rulesmd.golden`, `sdd-claude-claudemd.golden`, `sdd-codex-agentsmd.golden`, `sdd-codex-agentsmd-lowcost.golden`, `sdd-codex-agentsmd-powerful.golden`, `sdd-cursor-rules.golden`, `sdd-gemini-geminimd.golden`, `sdd-opencode-multi-settings.golden` y `sdd-vscode-instructions.golden`. Cambian con la sección de los agentes (2): `sdd-claude-agent-sdd-spec.golden` y `sdd-claude-agent-sdd-design.golden`.

Regeneración: `go test ./internal/components -run 'TestGolden' -update` (bandera `update` de `internal/components/golden_test.go:30`), y después la misma orden sin `-update` para confirmar. Revisar el diff: solo debe aparecer el párrafo **Explainer HTML.** y la sección de los agentes.

Solape con C-05-d: toca la misma sección doctrinal y los mismos 10 goldens. El párrafo va en bloque propio al final de la sección para que el conflicto textual sea, como mucho, de contexto adyacente. Quien fusione en segundo lugar rebasa sobre `main`, resuelve el `.md` conservando ambos párrafos y regenera los goldens con la orden anterior; nunca resuelve a mano un golden.

## Matriz de amenazas

N/A: el cambio no toca enrutamiento de agentes, órdenes de shell, subprocesos, automatización de VCS o PR, clasificación de ficheros ejecutables ni integración de procesos. El riesgo propio del cambio (servir HTML generado por un modelo) se trata en las decisiones 7, 8 y 11 y en las pruebas HTTP: lista blanca de dos ficheros, nombre validado tras la decodificación, ruta construida en el servidor, rechazo de enlaces simbólicos, límite de 5 MiB sin truncar, CSP sin scripts ni recursos externos e `iframe sandbox` con origen opaco.

## Migración y despliegue

No requiere migración. `explainer_html` es `omitempty` y su ausencia equivale a `no`; `internal/kickoff` decodifica sin `KnownFields` (`schema.go:78`), así que un binario anterior ignora la clave (lo fija el test de lectura heredada). Sin `sdd.explainer_html` en `axiom.yaml` rige `ask`. Reversión: revertir la PR; los `resumen-*.html` existentes son inertes.

## Preguntas abiertas

Ninguna bloqueante.

- [ ] El escenario «El hook del pre-vuelo ignora la pregunta» de prov:PRE:1 dice «cabecera»; el hook y el plugin inspeccionan el texto de la pregunta (`hasSDDPreflightQuestionPrefix`, `isSDDPreflightQuestion`) y la cabecera de `AskUserQuestion` es corta. El diseño pone el marcador al principio del texto de la pregunta; el test del hook cubre también una cabecera con el marcador. `sdd-tasks` o una revisión de la spec pueden alinear la palabra.
