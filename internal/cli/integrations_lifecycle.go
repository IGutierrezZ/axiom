package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/builtin"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Núcleo común de los verbos de tracker `axiom integrations tasks|summary|time` (eventos-ado, D14 y
// D15): resuelve, por este orden y antes de cualquier petición, las precondiciones compartidas, y da
// el sobre de salida (rechazos y avisos tipados) y el diario de avisos de `continuous` (D20).
// Códigos de salida: 0 hecho o no aplicable, 1 rechazo o resultado parcial, 2 error de uso.

const (
	integrationsTasksSchema   = "axiom.integrations-tasks/v1"
	integrationsSummarySchema = "axiom.integrations-summary/v1"
	integrationsTimeSchema    = "axiom.integrations-time/v1"

	// Estados del sobre común de salida.
	trackerStatusDone          = "done"
	trackerStatusPartial       = "partial"
	trackerStatusRefused       = "refused"
	trackerStatusNotApplicable = "not_applicable"

	// trackerCallTimeout acota cada verbo de tracker: son síncronos y fuera de la outbox.
	trackerCallTimeout = 2 * time.Minute
)

// Errores de diseño de los verbos de tracker: ninguna orden de axiom los resuelve.
var (
	errTrackerODDLane = errors.New("el carril ODD no crea Tasks ni imputa horas en el tracker") // refusal:by-design human-authority: es una decisión de producto que el carril ODD nunca escribe en el tracker
	errTrackerNoPort  = errors.New("el plugin activo no ofrece el puerto de Tasks")             // refusal:by-design world-action: hace falta una versión de axiom cuyo plugin ofrezca el puerto, no un comando de este binario
)

// trackerRefusal es el rechazo tipado de un verbo: código estable, mensaje y la continuación que
// nombra el comando que lo resuelve (D15). Los campos no se llaman Message ni Reason a propósito: el
// texto lo compone siempre un constructor con su continuación.
type trackerRefusal struct {
	Code string `json:"code"`
	Text string `json:"message"`
	Next string `json:"continuation"`
}

func (r *trackerRefusal) String() string {
	return fmt.Sprintf("%s: %s Continuación: %s", r.Code, r.Text, r.Next)
}

// trackerWarning es un aviso tipado: no detiene el verbo y deja el código de salida en 0.
type trackerWarning struct {
	Code string `json:"code"`
	Text string `json:"message"`
	Next string `json:"continuation,omitempty"`
	// Role es el rol al que se refiere el aviso, para el diario de continuous (no se serializa).
	Role string `json:"-"`
}

// trackerItemRef es la referencia a un elemento del tracker en la salida JSON.
type trackerItemRef struct {
	ID  string `json:"id"`
	URL string `json:"url,omitempty"`
}

func newTrackerRefusal(code, text, next string) *trackerRefusal {
	return &trackerRefusal{Code: code, Text: text, Next: next}
}

func newTrackerWarning(code, text, next string) trackerWarning {
	return trackerWarning{Code: code, Text: text, Next: next}
}

// trackerContext es el resultado de las precondiciones comunes: el cambio, su carpeta, el plugin con
// el puerto, el vínculo del work item y el estilo sellado.
type trackerContext struct {
	Loc      workspace.Location
	Change   string
	Root     string
	Folder   integrations.Folder
	Archived bool
	PluginID string
	Plugin   integrations.Plugin
	Config   workspace.IntegrationPluginConfig
	WorkItem integrations.Link
	Sealed   *kickoff.Kickoff
	Roster   []string
	Stderr   io.Writer
}

// Continuous indica un cambio sellado `continuous`: el único estilo con diario de avisos (D20).
func (c *trackerContext) Continuous() bool {
	return c != nil && c.Sealed != nil && c.Sealed.Config.ExecutionStyle == kickoff.ExecutionContinuous && !c.Archived
}

// note añade un suceso al diario `.tracker-notes.yaml` solo en `continuous` (D20). Un fallo al
// escribirlo se avisa por stderr y nunca cambia el resultado del verbo.
func (c *trackerContext) note(verb, role, code, text, next string) {
	if !c.Continuous() {
		return
	}
	err := integrations.AppendTrackerNote(c.Folder, integrations.TrackerNote{Verb: verb, Role: role, Code: code, Message: text, Continuation: next})
	if err != nil && c.Stderr != nil {
		fmt.Fprintf(c.Stderr, "Aviso: no se ha podido anotar el suceso %s en %s: %v. Repite el verbo con `axiom integrations %s --change %s`.\n",
			code, integrations.TrackerNotesFile, err, verb, c.Change)
	}
}

func (c *trackerContext) noteRefusal(verb, role string, r *trackerRefusal) {
	if r != nil {
		c.note(verb, role, r.Code, r.Text, r.Next)
	}
}

func (c *trackerContext) noteWarnings(verb string, warnings []trackerWarning) {
	for _, w := range warnings {
		c.note(verb, w.Role, w.Code, w.Text, w.Next)
	}
}

// trackerOptions son las precondiciones propias de cada verbo.
type trackerOptions struct {
	// Verb es el verbo que se muestra en los rechazos, por ejemplo "tasks create".
	Verb string
	// Gate es la compuerta que exige `checkpointed` (última decisión approved con digest vigente).
	// Vacía, el verbo no exige compuerta en ningún estilo.
	Gate kickoff.GateKey
	// Artifact comprueba, en `continuous`, que existe el artefacto del verbo; devuelve su nombre y si
	// existe. Nil, no se exige artefacto.
	Artifact func(root string) (string, bool)
	// AllowArchived admite la carpeta archivada del cambio (solo `summary push`).
	AllowArchived bool
}

// trackerResolution es el resultado de trackerTarget: el contexto (parcial en un rechazo, para poder
// anotar el diario), el rechazo o la marca de no aplicable.
type trackerResolution struct {
	Ctx           *trackerContext
	Refusal       *trackerRefusal
	NotApplicable bool
}

func trackerRefuseResolution(ctx *trackerContext, r *trackerRefusal) trackerResolution {
	return trackerResolution{Ctx: ctx, Refusal: r}
}

// trackerTarget resuelve las precondiciones comunes de D14 en este orden: carpeta del cambio y carril,
// plugin activo con el puerto, vínculo del work item y estilo sellado. No hace ninguna petición ni
// escribe ningún fichero. Las comprobaciones de uso (banderas) las hace antes cada verbo.
func trackerTarget(loc workspace.Location, cwd, change string, o trackerOptions, stderr io.Writer) trackerResolution {
	ctx := &trackerContext{Loc: loc, Change: change, Stderr: stderr}

	// 1. Carpeta.
	root, archived, refusal := resolveTrackerFolder(loc, cwd, change, o)
	if refusal != nil {
		return trackerRefuseResolution(ctx, refusal)
	}
	ctx.Root, ctx.Archived = root, archived
	ctx.Folder = integrations.Folder{WorkspaceRoot: loc.WorkspaceRoot, Root: root}
	if sealed, err := kickoff.Load(root); err == nil {
		ctx.Sealed = sealed
		if sealed != nil {
			ctx.Roster = rosterRoleNames(rosterFromKickoff(*sealed))
		}
	}

	// 2. Plugin activo con el puerto de Tasks.
	if refusal := resolveTrackerPlugin(ctx, o); refusal != nil {
		return trackerRefuseResolution(ctx, refusal)
	}
	if ctx.Plugin == nil {
		return trackerResolution{Ctx: ctx, NotApplicable: true}
	}

	// 3. Vínculo del work item.
	links, err := integrations.WorkItemLinks(ctx.Folder)
	if err != nil {
		return trackerRefuseResolution(ctx, newTrackerRefusal("integrations_links_invalid",
			fmt.Sprintf("no se puede leer integrations.yaml del cambio %s: %v.", change, err),
			fmt.Sprintf("axiom integrations sync --change %s", change)))
	}
	for _, link := range links {
		if link.Plugin == ctx.PluginID {
			ctx.WorkItem = link
			break
		}
	}
	if ctx.WorkItem.ID == "" {
		return trackerRefuseResolution(ctx, newTrackerRefusal("work_item_not_linked",
			fmt.Sprintf("el cambio %s no tiene vínculo kind work_item del plugin %s: no hay work item al que colgar Tasks ni resumen.", change, ctx.PluginID),
			fmt.Sprintf("axiom change start --from <id|url> o, si el cambio ya existe, axiom change start --adopt --change %s", change)))
	}

	// 4. Estilo sellado.
	if refusal := checkTrackerStyle(ctx, o); refusal != nil {
		return trackerRefuseResolution(ctx, refusal)
	}
	return trackerResolution{Ctx: ctx}
}

// resolveTrackerFolder es el paso 1 de D14: la carpeta ordinaria del planning home (el del estado del
// motor y, si no, el de workspace.Locate; en multirepo, el repositorio de specs), si no la archivada
// (solo con AllowArchived), y si no el carril ODD o un cambio inexistente.
func resolveTrackerFolder(loc workspace.Location, cwd, change string, o trackerOptions) (root string, archived bool, refusal *trackerRefusal) {
	var homes []string
	if status := engine.Default().Status(context.Background(), cwd, change, branch.LaneSDD); status.Status != branch.StatusError && status.Status != branch.StatusRefused && status.PlanningHome != "" {
		homes = append(homes, status.PlanningHome)
	}
	homes = append(homes, loc.PlanningHome)
	for _, home := range homes {
		if candidate := filepath.Join(home, "changes", change); isDirectory(candidate) {
			return candidate, false, nil
		}
	}
	if o.AllowArchived {
		if archivedRoot, ok := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, change); ok {
			return archivedRoot, true, nil
		}
	}
	if fileExists(filepath.Join(loc.WorkspaceRoot, "odd", "tasks", change+".md")) {
		return "", false, newTrackerRefusal("odd_lane_not_supported", errTrackerODDLane.Error()+": "+change+" es una característica ODD.",
			"el carril ODD no tiene continuación; para sincronizar con el tracker usa un cambio SDD (axiom sdd status)")
	}
	return "", false, newTrackerRefusal("change_not_found",
		fmt.Sprintf("el cambio %s no existe en %s.", change, filepath.Join(loc.PlanningHome, "changes")),
		"axiom sdd status")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// resolveTrackerPlugin es el paso 2 de D14: el primer plugin activo, por orden de declaración, que
// implementa el puerto de Tasks y cuyo manifiesto declara items.create. Sin él, deja ctx.Plugin nil.
func resolveTrackerPlugin(ctx *trackerContext, o trackerOptions) *trackerRefusal {
	loc := ctx.Loc
	if loc.Declarations.Integrations == nil {
		return nil
	}
	registry, err := builtin.DefaultRegistry()
	if err != nil {
		return newTrackerRefusal("tracker_unavailable", fmt.Sprintf("no se puede cargar el registro de plugins: %v.", err),
			fmt.Sprintf("axiom integrations %s --change %s", o.Verb, ctx.Change))
	}
	for _, cfg := range loc.Declarations.Integrations.Active() {
		plugin, ok := registry.Get(cfg.ID)
		if !ok {
			continue
		}
		if _, ok := plugin.(integrations.TaskTracker); !ok {
			continue
		}
		if !manifestHasCapability(plugin.Manifest(), manifest.CapItemsCreate) {
			continue
		}
		ctx.PluginID, ctx.Plugin, ctx.Config = cfg.ID, plugin, cfg
		return nil
	}
	return nil
}

func manifestHasCapability(m manifest.Manifest, capability manifest.Capability) bool {
	for _, c := range m.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

// checkTrackerStyle es el paso 4 de D14: en `checkpointed`, la última decisión de la compuerta debe
// ser approved con el digest vigente; en `continuous`, basta con que exista el artefacto. Sobre la
// carpeta archivada no se exige nada.
func checkTrackerStyle(ctx *trackerContext, o trackerOptions) *trackerRefusal {
	if ctx.Archived || (o.Gate == "" && o.Artifact == nil) {
		return nil
	}
	if ctx.Sealed == nil {
		return newTrackerRefusal("kickoff_not_sealed",
			fmt.Sprintf("el cambio %s no tiene kickoff sellado: sin él no hay roster ni estilo de ejecución.", ctx.Change),
			fmt.Sprintf("axiom sdd kickoff seal --change %s --execution-style checkpointed", ctx.Change))
	}
	switch ctx.Sealed.Config.ExecutionStyle {
	case kickoff.ExecutionCheckpointed:
		if o.Gate == "" {
			return nil
		}
		approved, err := trackerGateApproved(ctx.Root, o.Gate, ctx.Sealed)
		if err != nil {
			return newTrackerRefusal("gate_not_approved",
				fmt.Sprintf("no se puede leer la compuerta %s de %s: %v.", o.Gate, ctx.Change, err),
				fmt.Sprintf("axiom sdd gate show --change %s", ctx.Change))
		}
		if !approved {
			return newTrackerRefusal("gate_not_approved",
				fmt.Sprintf("la compuerta %s del cambio %s no tiene como última decisión una aprobación con el digest vigente: `%s` no fabrica aprobaciones.", o.Gate, ctx.Change, o.Verb),
				fmt.Sprintf("axiom sdd gate show --change %s", ctx.Change))
		}
	case kickoff.ExecutionContinuous:
		if o.Artifact == nil {
			return nil
		}
		if name, ok := o.Artifact(ctx.Root); !ok {
			return newTrackerRefusal("artifact_missing",
				fmt.Sprintf("el cambio %s (continuous) no tiene todavía %s: `%s` lo necesita.", ctx.Change, name, o.Verb),
				fmt.Sprintf("axiom sdd status --change %s", ctx.Change))
		}
	}
	return nil
}

// trackerGateApproved dice si la última decisión de gate en gates.yaml es approved y su digest sigue
// siendo el del artefacto actual (una edición posterior a la aprobación la deja en pending).
func trackerGateApproved(root string, gate kickoff.GateKey, sealed *kickoff.Kickoff) (bool, error) {
	ledger, err := kickoff.LoadGates(root)
	if err != nil {
		return false, err
	}
	var last *kickoff.GateRecord
	for i := range ledger.Records {
		if ledger.Records[i].Gate == gate {
			last = &ledger.Records[i]
		}
	}
	if last == nil || last.Decision != kickoff.DecisionApproved {
		return false, nil
	}
	current, err := gateArtifactDigest(root, gate, sealed)
	if err != nil {
		return false, err
	}
	return last.ArtifactDigest == current, nil
}

// trackerRosterHas dice si role pertenece al roster sellado.
func trackerRosterHas(roster []string, role string) bool {
	for _, r := range roster {
		if r == role {
			return true
		}
	}
	return false
}

// classifyTrackerError traduce un error del puerto de Tasks al rechazo tipado del catálogo (D15). retry
// es la continuación por defecto de los fallos que se resuelven repitiendo el verbo.
func classifyTrackerError(ctx *trackerContext, err error, retry string) *trackerRefusal {
	var (
		ambiguous *integrations.TaskAmbiguousError
		missing   *integrations.FieldMissingError
		unknown   *integrations.StateUnknownError
		conflict  *integrations.RevConflictError
		noPort    *integrations.PortUnsupportedError
	)
	discover := fmt.Sprintf("axiom integrations discover --plugin %s", ctx.PluginID)
	switch {
	case errors.As(err, &noPort):
		return newTrackerRefusal("port_unsupported", errTrackerNoPort.Error()+".", "hace falta una versión de axiom cuyo plugin ofrezca el puerto de Tasks (sin comando que lo resuelva)")
	case errors.As(err, &ambiguous):
		return newTrackerRefusal("task_ambiguous", err.Error(),
			fmt.Sprintf("axiom integrations tasks bind --change %s --role <rol>=<id> con una de las Tasks %s", ctx.Change, strings.Join(ambiguous.IDs, ", ")))
	case errors.Is(err, integrations.ErrTaskParentLinkFailed):
		return newTrackerRefusal("task_parent_link_failed", err.Error(), fmt.Sprintf("axiom integrations tasks create --change %s", ctx.Change))
	case errors.As(err, &missing):
		return newTrackerRefusal("field_missing", err.Error(), discover)
	case errors.As(err, &unknown):
		return newTrackerRefusal("state_unknown", err.Error(), discover)
	case errors.As(err, &conflict) && !conflict.AfterRetry:
		return newTrackerRefusal("rev_conflict", err.Error(), retry)
	case errors.Is(err, integrations.ErrTrackerRejected):
		return newTrackerRefusal("tracker_rejected", err.Error(), retry)
	}
	return newTrackerRefusal("tracker_unavailable", err.Error(), retry)
}

// trackerChangeUsageError valida el nombre del cambio como lo hace `emit`: una carpeta simple.
func trackerChangeUsageError(change string) string {
	switch {
	case strings.TrimSpace(change) == "":
		return "--change es obligatorio"
	case change == "." || change == ".." || strings.ContainsAny(change, `/\`):
		return fmt.Sprintf("el nombre de cambio %q no es un nombre de carpeta simple", change)
	}
	return ""
}

// trackerRunContext acota un verbo de tracker en el tiempo.
func trackerRunContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), trackerCallTimeout)
}

// fromPortWarnings convierte los avisos del puerto en avisos de la salida.
func fromPortWarnings(role string, warnings []integrations.Warning) []trackerWarning {
	out := make([]trackerWarning, 0, len(warnings))
	for _, w := range warnings {
		text := w.Message
		if role != "" {
			text = fmt.Sprintf("rol %s: %s", role, text)
		}
		warning := newTrackerWarning(w.Code, text, w.Continuation)
		warning.Role = role
		out = append(out, warning)
	}
	return out
}

func printTrackerWarnings(stdout io.Writer, warnings []trackerWarning) {
	for _, w := range warnings {
		line := fmt.Sprintf("Aviso %s: %s", w.Code, w.Text)
		if w.Next != "" {
			line += " Continuación: " + w.Next
		}
		fmt.Fprintln(stdout, line)
	}
}
