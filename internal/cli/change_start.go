package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/changestart"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/builtin"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// `axiom change start` (C-07-d): arranque de un cambio desde un work item existente, en dos fases y un
// modo de descarte. Este fichero adapta banderas, aplicabilidad, catálogo de rechazos y salida; la
// lógica vive en internal/changestart (D-01). El verbo no ejecuta git ni el resto del arranque.

const (
	changeStartSchema = "axiom.change-start/v1"

	changeStartExitOK    = 0
	changeStartExitFail  = 1
	changeStartExitUsage = 2

	changeStartPhaseProbe   = "probe"
	changeStartPhaseFrom    = "from"
	changeStartPhaseAdopt   = "adopt"
	changeStartPhaseDiscard = "discard"

	changeStartStatusNotApplicable = "not_applicable"
	changeStartStatusRefused       = "refused"
)

const changeStartHelp = `Uso: axiom change start [--from <id|url> | --adopt --change <nombre> [--lane sdd|odd] | --discard --change <nombre>] [--cwd <ruta>] [--json]

Arranca un cambio desde un work item existente del tracker. Tiene dos fases y un modo de descarte:

  --from <id|url>     Fase 1. Lee el work item una vez, lo valida, deriva el nombre del cambio y deja el
                      contexto importado en staging bajo el common dir de git. No escribe en el árbol
                      de trabajo ni en el work item. Devuelve el nombre, el perfil y la referencia.
  --adopt             Fase 2. Con el cambio ya preparado (axiom branch ensure), copia y verifica el contexto en
                      el cambio, vincula el work item y le añade la etiqueta axiom:<cambio>. Exige --change.
  --discard           Retira un staging abandonado. Exige --change.

Sin ningún modo, el verbo es la consulta del orquestador: dice si hay un plugin de work items activo
(no aplica, código 0) o si hace falta indicar el work item (código 1).

Banderas:
  --from <id|url>     Identificador numérico o URL del work item de la organización configurada
  --adopt             Ejecuta la fase 2
  --discard           Retira el staging del cambio
  --change <nombre>   Cambio de --adopt o --discard
  --lane sdd|odd      Carril del cambio en --adopt (por defecto sdd)
  --cwd <ruta>        Directorio de partida (por defecto, el directorio de trabajo)
  --json              Imprime el resultado como ` + changeStartSchema + `

El verbo no ejecuta axiom branch ensure ni axiom sdd kickoff seal, no crea proposal.md ni el documento
ODD y no ejecuta git. Sale con 0 si terminó o no aplica; con 1 por un rechazo tipado; con 2 por un
error de uso.
`

// Catálogo cerrado de rechazos (D-18). Cada centinela nombra un `axiom …` o declara por qué no hay
// comando que lo resuelva. El texto que ve la persona lo compone changeStartRefusalText.
var (
	ErrChangeStartUsage                 = errors.New("uso incorrecto de `axiom change start`: revisa `axiom change start --help`")
	ErrChangeStartWorkItemRequired      = errors.New("hace falta el work item de origen: indícalo con `axiom change start --from <id|url>`")
	ErrChangeStartReferenceInvalid      = errors.New("la referencia no es un work item configurado: repite `axiom change start --from <id|url>`")
	ErrChangeStartNotFound              = errors.New("el tracker no encontró el work item: repite `axiom change start --from <id|url>`")
	ErrChangeStartTypeUnmapped          = errors.New("el tipo del work item no está en item_keys: añádelo y repite `axiom change start --from <id|url>`")
	ErrChangeStartClosed                = errors.New("el work item está cerrado: reábrelo o usa otro con `axiom change start --from <id|url>`")
	ErrChangeStartLinked                = errors.New("el work item ya está enlazado a otro cambio abierto: resuélvelo y repite `axiom change start --from <id|url>`")
	ErrChangeStartTrackerUnavailable    = errors.New("el tracker no está disponible: reintenta `axiom change start --from <id|url>`")
	ErrChangeStartStagingConflict       = errors.New("el staging del cambio es de otro work item: retíralo con `axiom change start --discard --change <cambio>`")
	ErrChangeStartBusy                  = errors.New("otro proceso arranca este cambio: reintenta `axiom change start` con los mismos argumentos")
	ErrChangeStartGitRepositoryRequired = errors.New("la raíz del workspace no está en un repositorio git: repite `axiom change start` con `--cwd <repositorio>`")
	ErrChangeStartWriteFailed           = errors.New("no se pudo escribir el staging o el contexto del cambio: reintenta `axiom change start` con los mismos argumentos")
	ErrChangeStartNotPrepared           = errors.New("el cambio no está preparado: prepáralo con `axiom branch ensure --change <cambio> --json` y repite `axiom change start --adopt --change <cambio>`")
	ErrChangeStartStagingNotFound       = errors.New("no hay staging ni contexto adoptado de ese cambio: ejecuta `axiom change start --from <id|url>`")
	ErrChangeStartPortUnsupported       = errors.New("el plugin activo no ofrece el puerto de work items") // refusal:by-design world-action: hace falta una versión de axiom cuyo plugin ofrezca el puerto; no hay comando que lo resuelva
)

// changeStartKinds asocia cada código tipado con su centinela.
var changeStartKinds = map[string]error{
	"usage_invalid":                          ErrChangeStartUsage,
	"work_item_required":                     ErrChangeStartWorkItemRequired,
	changestart.CodeWorkItemReferenceInvalid: ErrChangeStartReferenceInvalid,
	changestart.CodeWorkItemNotFound:         ErrChangeStartNotFound,
	changestart.CodeWorkItemTypeUnmapped:     ErrChangeStartTypeUnmapped,
	changestart.CodeWorkItemClosed:           ErrChangeStartClosed,
	changestart.CodeWorkItemLinked:           ErrChangeStartLinked,
	changestart.CodeTrackerUnavailable:       ErrChangeStartTrackerUnavailable,
	changestart.CodeStagingConflict:          ErrChangeStartStagingConflict,
	changestart.CodeChangeStartBusy:          ErrChangeStartBusy,
	"git_repository_required":                ErrChangeStartGitRepositoryRequired,
	changestart.CodeWriteFailed:              ErrChangeStartWriteFailed,
	changestart.CodeChangeNotPrepared:        ErrChangeStartNotPrepared,
	changestart.CodeStagingNotFound:          ErrChangeStartStagingNotFound,
	"port_unsupported":                       ErrChangeStartPortUnsupported,
}

const (
	continuationFrom    = "axiom change start --from <id|url>"
	continuationRetry   = "axiom change start …"
	continuationUsage   = "axiom change start --help"
	continuationDiscard = "axiom change start --discard --change <cambio>"
	continuationCwd     = "axiom change start … --cwd <repositorio>"
)

// changeStartRefusalJSON es el rechazo de la salida: código, mensaje y continuación. Fields solo lleva
// las dos excepciones de texto del tracker de D-20, acotadas por D-28.
type changeStartRefusalJSON struct {
	Code         string            `json:"code"`
	Message      string            `json:"message"`
	Continuation string            `json:"continuation,omitempty"`
	Fields       map[string]string `json:"fields,omitempty"`
}

type changeStartWorkItemJSON struct {
	Plugin  string `json:"plugin"`
	ID      string `json:"id"`
	TypeKey string `json:"type_key,omitempty"`
	URL     string `json:"url,omitempty"`
}

type changeStartAttachmentsJSON struct {
	Downloaded int `json:"downloaded"`
	Omitted    int `json:"omitted"`
}

// changeStartJSON es el documento axiom.change-start/v1. Los campos vacíos se omiten salvo schema,
// phase, applicable, status y warnings.
type changeStartJSON struct {
	Schema      string                      `json:"schema"`
	Phase       string                      `json:"phase"`
	Applicable  bool                        `json:"applicable"`
	Status      string                      `json:"status"`
	Change      string                      `json:"change,omitempty"`
	Profile     string                      `json:"profile,omitempty"`
	ExternalRef string                      `json:"external_ref,omitempty"`
	WorkItem    *changeStartWorkItemJSON    `json:"work_item,omitempty"`
	Staging     string                      `json:"staging,omitempty"`
	Context     string                      `json:"context,omitempty"`
	Lane        string                      `json:"lane,omitempty"`
	Link        string                      `json:"link,omitempty"`
	Backlink    string                      `json:"backlink,omitempty"`
	Attachments *changeStartAttachmentsJSON `json:"attachments,omitempty"`
	Refusal     *changeStartRefusalJSON     `json:"refusal,omitempty"`
	Warnings    []string                    `json:"warnings"`
}

// workItemTracker liga el plugin activo, su configuración y los secretos al puerto de work items.
type workItemTracker struct {
	manifest manifest.Manifest
	plugin   integrations.Plugin
	config   workspace.IntegrationPluginConfig
	store    secrets.Store
	keys     []manifest.TypePrefix
}

func (t *workItemTracker) PluginID() string                { return t.manifest.ID }
func (t *workItemTracker) RefAlias() string                { return t.manifest.RefAlias }
func (t *workItemTracker) ItemKeys() []manifest.TypePrefix { return t.keys }
func (t *workItemTracker) ParseRef(raw string) (string, error) {
	return integrations.ParseItemRef(t.plugin, t.config, raw)
}

func (t *workItemTracker) Read(ctx context.Context, id string) (integrations.ExternalItem, error) {
	return integrations.ReadItem(ctx, t.plugin, t.config, t.store, id)
}

func (t *workItemTracker) StateCategory(ctx context.Context, configuredType, state string) (integrations.StateCategory, error) {
	return integrations.ReadStateCategory(ctx, t.plugin, t.config, t.store, configuredType, state)
}

func (t *workItemTracker) Download(ctx context.Context, url string, max int64) (integrations.Attachment, error) {
	return integrations.DownloadAttachment(ctx, t.plugin, t.config, t.store, url, max)
}

func (t *workItemTracker) LinkBack(ctx context.Context, id, tag string, allow func([]string) error) (integrations.BackLink, error) {
	return integrations.LinkBack(ctx, t.plugin, t.config, t.store, id, tag, allow)
}

// workItemApplicability es el resultado de la aplicabilidad (D-17, paso 4).
type workItemApplicability struct {
	tracker         *workItemTracker
	portUnsupported bool
	warnings        []string
}

// resolveWorkItemTracker decide si el verbo aplica: sin axiom.yaml, con un bloque de integraciones
// inválido, sin plugins activos o sin el plugin compilado, no aplica (con aviso cuando la exigencia del
// work item queda desactivada por un fallo). Un plugin activo y compilado sin el puerto es
// port_unsupported. Se elige el primer plugin activo, en orden de declaración, que implemente el puerto.
func resolveWorkItemTracker(cwd string) (workspace.Location, workItemApplicability) {
	invalid := "axiom.yaml o su bloque `integrations:` no es válido, así que no se puede saber si el work item es obligatorio: revísalo con `axiom workspace validate`"
	loc, err := workspace.Locate(cwd)
	if err != nil {
		return loc, workItemApplicability{warnings: []string{invalid}}
	}
	if loc.ConfigErr != nil || loc.Declarations.IntegrationsErr != nil {
		return loc, workItemApplicability{warnings: []string{invalid}}
	}
	if loc.Declarations.Integrations == nil {
		return loc, workItemApplicability{}
	}
	active := loc.Declarations.Integrations.Active()
	if len(active) == 0 {
		return loc, workItemApplicability{}
	}
	registry, err := builtin.DefaultRegistry()
	if err != nil {
		return loc, workItemApplicability{warnings: []string{"no se pudo cargar el registro de plugins de integración: el work item de origen no es obligatorio en esta ejecución"}}
	}
	var out workItemApplicability
	for _, cfg := range active {
		plugin, ok := registry.Get(cfg.ID)
		if !ok {
			out.warnings = append(out.warnings, "el plugin activo "+cfg.ID+" no está compilado en esta versión de axiom: el work item de origen no es obligatorio")
			continue
		}
		if _, ok := plugin.(integrations.WorkItemSource); !ok {
			out.portUnsupported = true
			continue
		}
		keys, _ := manifest.ParseTypePrefixes(itemKeysOf(cfg))
		out.tracker = &workItemTracker{manifest: plugin.Manifest(), plugin: plugin, config: cfg, store: secrets.Default(), keys: keys}
		out.portUnsupported = false
		return loc, out
	}
	return loc, out
}

func itemKeysOf(cfg workspace.IntegrationPluginConfig) []string {
	keys, _ := cfg.Values["item_keys"].([]string)
	return keys
}

// changeStartOptions son las banderas ya analizadas.
type changeStartOptions struct {
	cwd, from, change, lane string
	hasFrom, adopt, discard bool
	asJSON                  bool
}

func (o changeStartOptions) phase() string {
	switch {
	case o.hasFrom:
		return changeStartPhaseFrom
	case o.adopt:
		return changeStartPhaseAdopt
	case o.discard:
		return changeStartPhaseDiscard
	}
	return changeStartPhaseProbe
}

// parseChangeStartFlags analiza las banderas y aplica D-17, pasos 1 a 3. Devuelve el mensaje de uso
// incorrecto o vacío.
func parseChangeStartFlags(args []string) (changeStartOptions, string, bool) {
	var o changeStartOptions
	flags := flag.NewFlagSet("change start", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&o.cwd, "cwd", ".", "directorio de partida")
	flags.StringVar(&o.from, "from", "", "work item de origen")
	flags.BoolVar(&o.adopt, "adopt", false, "fase 2")
	flags.BoolVar(&o.discard, "discard", false, "retira el staging")
	flags.StringVar(&o.change, "change", "", "cambio")
	flags.StringVar(&o.lane, "lane", "", "carril")
	flags.BoolVar(&o.asJSON, "json", false, "salida JSON")
	if err := flags.Parse(args); err != nil {
		o.asJSON = slices.Contains(args, "--json") || slices.Contains(args, "-json")
		if errors.Is(err, flag.ErrHelp) {
			return o, "", true
		}
		return o, "banderas no válidas: " + err.Error(), false
	}
	if flags.NArg() > 0 {
		return o, fmt.Sprintf("argumento posicional inesperado %q", flags.Arg(0)), false
	}
	seen := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	o.hasFrom = seen["from"]

	modes := 0
	for _, on := range []bool{o.hasFrom, o.adopt, o.discard} {
		if on {
			modes++
		}
	}
	switch {
	case modes > 1:
		return o, "los modos --from, --adopt y --discard son excluyentes", false
	case (o.adopt || o.discard) && !seen["change"]:
		return o, "--adopt y --discard exigen --change <nombre>", false
	case seen["change"] && !o.adopt && !o.discard:
		return o, "--change solo se acepta con --adopt o --discard", false
	case seen["lane"] && !o.adopt:
		return o, "--lane solo se acepta con --adopt", false
	case seen["lane"] && o.lane != "sdd" && o.lane != "odd":
		return o, "--lane debe valer sdd u odd", false
	}
	if seen["change"] {
		if err := changestart.ValidChangeName(o.change); err != nil {
			return o, "--change no es un nombre de cambio válido: " + err.Error(), false
		}
	}
	return o, "", false
}

// RunChangeStart ejecuta `axiom change start`. El orden de comprobación es el de D-17: banderas,
// exclusión de modos, --change, aplicabilidad, work item requerido, repositorio git y despacho.
func RunChangeStart(args []string, stdout, stderr io.Writer) int {
	opts, usage, help := parseChangeStartFlags(args)
	if help {
		fmt.Fprint(stdout, changeStartHelp)
		return changeStartExitOK
	}
	doc := changeStartJSON{Schema: changeStartSchema, Phase: opts.phase(), Warnings: []string{}}
	if usage != "" {
		return emitChangeStartRefusal(opts, doc, "usage_invalid", nil, "", usage, stdout, stderr)
	}

	loc, app := resolveWorkItemTracker(opts.cwd)
	switch {
	case app.portUnsupported && app.tracker == nil:
		return emitChangeStartRefusal(opts, doc, "port_unsupported", nil, "", "", stdout, stderr)
	case app.tracker == nil:
		doc.Applicable = false
		doc.Status = changeStartStatusNotApplicable
		doc.Warnings = append(doc.Warnings, app.warnings...)
		return emitChangeStartDoc(opts, doc, stdout, stderr, changeStartExitOK)
	}
	doc.Applicable = true
	doc.Warnings = append(doc.Warnings, app.warnings...)
	plugin := app.tracker.PluginID()

	if !opts.hasFrom && !opts.adopt && !opts.discard || opts.hasFrom && strings.TrimSpace(opts.from) == "" {
		return emitChangeStartRefusal(opts, doc, "work_item_required", nil, plugin, "", stdout, stderr)
	}

	gitRoot, common, err := changestart.ResolveRepository(loc.WorkspaceRoot)
	if err != nil {
		return emitChangeStartRefusal(opts, doc, "git_repository_required", nil, plugin, "", stdout, stderr)
	}
	env := changestart.Env{WorkspaceRoot: loc.WorkspaceRoot, PlanningHome: loc.PlanningHome, GitRoot: gitRoot, CommonDir: common}

	switch {
	case opts.adopt:
		return runChangeStartAdopt(opts, doc, env, app.tracker, stdout, stderr)
	case opts.discard:
		return runChangeStartDiscard(opts, doc, env, plugin, stdout, stderr)
	}
	res, err := changestart.Start(context.Background(), env, app.tracker, opts.from)
	if err != nil {
		return emitChangeStartError(opts, doc, err, plugin, stdout, stderr)
	}
	doc.Status = res.Status
	doc.Change = res.Change
	doc.Profile = string(res.Profile)
	doc.ExternalRef = res.ExternalRef
	doc.WorkItem = &changeStartWorkItemJSON{Plugin: res.Plugin, ID: res.ID, TypeKey: res.TypeKey, URL: res.URL}
	doc.Staging = res.Staging
	doc.Context = res.Context
	doc.Lane = string(res.Lane)
	if res.Status == changestart.StatusStaged || res.Status == changestart.StatusAlreadyStaged {
		doc.Attachments = &changeStartAttachmentsJSON{Downloaded: res.Downloaded, Omitted: res.Omitted}
	}
	doc.Warnings = append(doc.Warnings, res.Warnings...)
	return emitChangeStartDoc(opts, doc, stdout, stderr, changeStartExitOK)
}

// emitChangeStartError traduce el error de una operación a su rechazo tipado: un *changestart.Refusal con
// su código y sus datos de confianza, el puerto no soportado o, en cualquier otro caso, un fallo de
// escritura. Añade el cambio pedido a los datos del rechazo si la lógica no lo nombró.
func emitChangeStartError(opts changeStartOptions, doc changeStartJSON, err error, plugin string, stdout, stderr io.Writer) int {
	var refusal *changestart.Refusal
	switch {
	case errors.As(err, &refusal):
		fields := map[string]string{}
		for k, v := range refusal.Fields {
			fields[k] = v
		}
		if fields["change"] == "" && opts.change != "" {
			fields["change"] = opts.change
		}
		usage := ""
		if refusal.Code == changestart.CodeUsageInvalid {
			usage = "--change no es un nombre de cambio válido"
		}
		return emitChangeStartRefusal(opts, doc, refusal.Code, fields, plugin, usage, stdout, stderr)
	case errors.Is(err, integrations.ErrPortUnsupported):
		return emitChangeStartRefusal(opts, doc, "port_unsupported", nil, plugin, "", stdout, stderr)
	}
	return emitChangeStartRefusal(opts, doc, changestart.CodeWriteFailed, map[string]string{"change": opts.change}, plugin, "", stdout, stderr)
}

// runChangeStartAdopt ejecuta la fase 2 (--adopt) y compone la salida axiom.change-start/v1.
func runChangeStartAdopt(opts changeStartOptions, doc changeStartJSON, env changestart.Env, tracker changestart.Tracker, stdout, stderr io.Writer) int {
	lane := changestart.LaneSDD
	if opts.lane == "odd" {
		lane = changestart.LaneODD
	}
	res, err := changestart.Adopt(context.Background(), env, tracker, opts.change, lane)
	if err != nil {
		return emitChangeStartError(opts, doc, err, tracker.PluginID(), stdout, stderr)
	}
	doc.Status = res.Status
	doc.Change = res.Change
	doc.Profile = string(res.Profile)
	doc.ExternalRef = res.ExternalRef
	doc.WorkItem = &changeStartWorkItemJSON{Plugin: res.Plugin, ID: res.ID, TypeKey: res.TypeKey, URL: res.URL}
	doc.Context = res.Context
	doc.Lane = string(res.Lane)
	doc.Link = res.Link
	doc.Backlink = res.Backlink
	doc.Attachments = &changeStartAttachmentsJSON{Downloaded: res.Downloaded, Omitted: res.Omitted}
	doc.Warnings = append(doc.Warnings, res.Warnings...)
	return emitChangeStartDoc(opts, doc, stdout, stderr, changeStartExitOK)
}

// runChangeStartDiscard ejecuta --discard: sin red, sin tracker y sin tocar el repositorio.
func runChangeStartDiscard(opts changeStartOptions, doc changeStartJSON, env changestart.Env, plugin string, stdout, stderr io.Writer) int {
	res, err := changestart.Discard(env, opts.change)
	if err != nil {
		return emitChangeStartError(opts, doc, err, plugin, stdout, stderr)
	}
	doc.Status = res.Status
	doc.Change = res.Change
	doc.Staging = res.Staging
	doc.Warnings = append(doc.Warnings, res.Warnings...)
	return emitChangeStartDoc(opts, doc, stdout, stderr, changeStartExitOK)
}

// changeStartRefusalText compone el mensaje y la continuación de un rechazo (D-18, D-20). Solo usa
// datos de confianza; el tipo y la etiqueta salen ya neutralizados o validados (D-28).
func changeStartRefusalText(code string, fields map[string]string, plugin, usage string) (message, continuation string) {
	switch code {
	case "usage_invalid":
		return usage, continuationUsage
	case "work_item_required":
		return "Hay un plugin de work items activo: indica el work item de origen, su identificador numérico o su URL. Crear el work item no es una opción de este verbo.", continuationFrom
	case changestart.CodeWorkItemReferenceInvalid:
		return "La referencia no es un work item del host, la organización y el proyecto configurados: usa su identificador numérico o la URL que copias del navegador.", continuationFrom
	case changestart.CodeWorkItemNotFound:
		return "El tracker no encontró el work item: revisa el identificador.", continuationFrom
	case changestart.CodeWorkItemTypeUnmapped:
		return fmt.Sprintf("El tipo %s del work item no está mapeado en `integrations.%s.item_keys` de axiom.yaml.", fields["type"], plugin),
			fmt.Sprintf("Añade `<Tipo>=<PREFIJO>` a integrations.%s.item_keys y repite `%s`", plugin, continuationFrom)
	case changestart.CodeWorkItemClosed:
		return "El work item está cerrado (su estado pertenece a la categoría Completed o Removed).", "Reabre el work item en el tracker o usa otro: `" + continuationFrom + "`"
	case changestart.CodeWorkItemLinked:
		return linkedText(fields)
	case changestart.CodeTrackerUnavailable:
		if fields["secret"] != "" {
			return "Falta el secreto del tracker: no se hizo ninguna petición.",
				fmt.Sprintf("axiom integrations secret set --plugin %s --name %s, y reintenta `%s`", plugin, fields["secret"], continuationFrom)
		}
		return "El tracker no está disponible, la credencial no es válida o se agotó el tiempo: no se escribió nada.",
			fmt.Sprintf("Reintenta `%s`; si falta el PAT, `axiom integrations secret set --plugin %s --name pat`", continuationFrom, plugin)
	case changestart.CodeStagingConflict:
		return fmt.Sprintf("Ya existe un staging del cambio %s asociado a otro work item.", fields["change"]), "axiom change start --discard --change " + fields["change"]
	case changestart.CodeChangeStartBusy:
		return "Otro proceso está arrancando este work item o este cambio.", "Reintenta `" + continuationRetry + "` con los mismos argumentos"
	case "git_repository_required":
		return "La raíz del workspace no pertenece a un repositorio git: no hay common dir donde dejar el staging.", continuationCwd
	case changestart.CodeWriteFailed:
		return writeFailedText(fields)
	case changestart.CodeChangeNotPrepared:
		return notPreparedText(fields)
	case changestart.CodeStagingNotFound:
		return fmt.Sprintf("No hay staging del cambio %s ni un contexto ya adoptado en su destino: no hay qué adoptar.", fields["change"]), continuationFrom
	case "port_unsupported":
		return "El plugin activo no ofrece el puerto de work items: hace falta una versión de axiom cuyo plugin lo ofrezca.", ""
	}
	return "Rechazo no catalogado.", continuationUsage
}

// writeFailedText distingue el fallo de escritura de la fase 1 (no queda staging adoptable) de los de la
// fase 2 y el descarte, que dejan el staging intacto y se reintentan igual (D-25).
func writeFailedText(fields map[string]string) (string, string) {
	change := fields["change"]
	switch fields["step"] {
	case "copy", "verify", "rename", "link", "exclude", "workitem_json", "destination":
		return "No se pudo completar la adopción del contexto: el staging permanece intacto y el destino queda como estaba.",
			"Reintenta `axiom change start --adopt --change " + change + "`"
	case "discard":
		return "No se pudo retirar el staging: permanece intacto.", "Reintenta `axiom change start --discard --change " + change + "`"
	}
	return "No se pudo escribir el staging; no queda ningún staging adoptable.", "Reintenta `" + continuationRetry + "` con los mismos argumentos"
}

// notPreparedText compone change_not_prepared (D-14): el motivo, el carril del otro registro y el
// worktree salen de datos de confianza de la lógica; la continuación nunca ejecuta nada.
func notPreparedText(fields map[string]string) (string, string) {
	change, lane := fields["change"], fields["lane"]
	laneFlag := func(l string) string {
		if l == "odd" {
			return " --lane odd"
		}
		return ""
	}
	label := func(l string) string { return strings.ToUpper(l) }
	ensure := fmt.Sprintf("axiom branch ensure --change %s%s --json", change, laneFlag(lane))
	adoptHere := ""
	if wt := fields["worktree"]; wt != "" {
		adoptHere = fmt.Sprintf(" Desde su worktree, repite `axiom change start --adopt --change %s%s --cwd %s`.", change, laneFlag(lane), pathquote.Quote(wt))
	}
	switch fields["reason"] {
	case "lane_mismatch":
		record := fields["record_lane"]
		cont := fmt.Sprintf("axiom change start --adopt --change %s%s", change, laneFlag(record))
		if wt := fields["worktree"]; wt != "" {
			cont += " --cwd " + pathquote.Quote(wt)
		}
		return fmt.Sprintf("El cambio %s está preparado en el carril %s, no en %s.", change, label(record), label(lane)), cont
	case "record_unreadable":
		return fmt.Sprintf("El registro de ramas del cambio %s no se puede leer: %s.", change, oneLine(fields["detail"])), ensure
	case "head_other_branch":
		return fmt.Sprintf("El HEAD del checkout no es la rama del cambio %s: no se escribe en otra rama.%s", change, adoptHere), ensure
	case "folder_missing":
		return fmt.Sprintf("Falta la carpeta openspec/changes/%s/ del cambio: prepárala con el arranque del cambio (`axiom branch ensure` y `axiom sdd kickoff seal`).%s", change, adoptHere), ensure
	}
	return fmt.Sprintf("El cambio %s no está preparado: no hay su registro de ramas en el carril %s.", change, label(lane)), ensure
}

func linkedText(fields map[string]string) (string, string) {
	other := fields["change"]
	if tag := fields["tag"]; tag != "" {
		if fields["local"] == "false" {
			return fmt.Sprintf("El work item ya está enlazado a otro cambio abierto: lo reclama la etiqueta %s, y el cambio %s no consta en este equipo (puede estar abierto en otra máquina).", tag, other),
				fmt.Sprintf("Retira la etiqueta %s en el tracker, o actualiza el checkout principal y repite `%s`", tag, continuationFrom)
		}
		return fmt.Sprintf("El work item ya está enlazado a otro cambio abierto: lo reclama la etiqueta %s.", tag),
			fmt.Sprintf("Retira la etiqueta %s en el tracker o cierra el otro cambio con `axiom branch done --change %s`", tag, other)
	}
	return fmt.Sprintf("El work item ya está adoptado por el cambio abierto %s de este equipo.", other),
		fmt.Sprintf("Cierra ese cambio con `axiom branch done --change %s` o retira su vínculo", other)
}

func emitChangeStartRefusal(opts changeStartOptions, doc changeStartJSON, code string, fields map[string]string, plugin, usage string, stdout, stderr io.Writer) int {
	message, continuation := changeStartRefusalText(code, fields, plugin, usage)
	ref := &changeStartRefusalJSON{Code: code, Message: message, Continuation: continuation}
	switch code {
	case changestart.CodeWorkItemTypeUnmapped:
		ref.Fields = map[string]string{"type": fields["type"]}
	case changestart.CodeWorkItemLinked:
		ref.Fields = map[string]string{}
		for _, k := range []string{"tag", "change"} {
			if v := fields[k]; v != "" {
				ref.Fields[k] = v
			}
		}
	}
	doc.Status = changeStartStatusRefused
	doc.Refusal = ref
	exit := changeStartExitFail
	if code == "usage_invalid" {
		exit = changeStartExitUsage
	}
	return emitChangeStartDoc(opts, doc, stdout, stderr, exit)
}

// emitChangeStartDoc imprime el documento (JSON) o su forma de texto confinada a datos de confianza.
func emitChangeStartDoc(opts changeStartOptions, doc changeStartJSON, stdout, stderr io.Writer, exit int) int {
	if doc.Warnings == nil {
		doc.Warnings = []string{}
	}
	if opts.asJSON {
		writeIntegrationsJSON(stdout, doc)
		return exit
	}
	out := stdout
	if doc.Refusal != nil {
		out = stderr
		fmt.Fprintf(out, "Error: %s\n", oneLine(doc.Refusal.Message))
		if doc.Refusal.Continuation != "" {
			fmt.Fprintf(out, "Continuación: %s\n", doc.Refusal.Continuation)
		}
	} else {
		fmt.Fprintln(out, changeStartSummary(doc))
	}
	for _, w := range doc.Warnings {
		fmt.Fprintf(out, "Aviso: %s\n", oneLine(w))
	}
	return exit
}

func changeStartSummary(doc changeStartJSON) string {
	switch doc.Status {
	case changeStartStatusNotApplicable:
		return "No aplica: no hay un plugin de work items activo, así que el work item de origen no es obligatorio."
	case changestart.StatusStaged:
		return fmt.Sprintf("Contexto del work item preparado para el cambio %s (perfil %s, referencia %s).\nStaging: %s\nSiguiente: prepara el cambio con ese nombre y adóptalo con `axiom change start --adopt --change %s`.",
			doc.Change, doc.Profile, doc.ExternalRef, filepath.ToSlash(doc.Staging), doc.Change)
	case changestart.StatusAlreadyStaged:
		return fmt.Sprintf("Ya hay un staging de este work item para el cambio %s (perfil %s, referencia %s): %s", doc.Change, doc.Profile, doc.ExternalRef, filepath.ToSlash(doc.Staging))
	case changestart.StatusAlreadyAdopted:
		return fmt.Sprintf("El cambio %s ya adoptó este work item (perfil %s, referencia %s): %s", doc.Change, doc.Profile, doc.ExternalRef, filepath.ToSlash(doc.Context))
	case changestart.StatusAdopted:
		return fmt.Sprintf("Contexto del work item adoptado por el cambio %s (carril %s, referencia %s, enlace de vuelta: %s).\nContexto: %s",
			doc.Change, doc.Lane, doc.ExternalRef, doc.Backlink, filepath.ToSlash(doc.Context))
	case changestart.StatusDiscarded:
		return fmt.Sprintf("Staging del cambio %s retirado.", doc.Change)
	case changestart.StatusNothingToDiscard:
		return fmt.Sprintf("No había ningún staging del cambio %s que retirar.", doc.Change)
	}
	return doc.Status
}
