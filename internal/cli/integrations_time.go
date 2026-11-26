package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/trackersync"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// `axiom integrations time propose|record` (eventos-ado, prov:INT:7 y prov:INT:8): proponen e imputan
// las horas de un rol a la Task vinculada. `propose` es de solo lectura; `record` escribe con
// `test /rev`, deja una intención en el diario local `.time-pending.yaml` antes de tocar el tracker
// (D9) y solo registra la imputación en integrations.yaml tras una escritura con éxito.

const integrationsTimeHelp = `Uso: axiom integrations time <propose|record> --change <cambio> --role <rol> [banderas]

  propose   Propone las horas imputables del rol (solo lectura)
  record    Imputa horas a la Task del rol y, en continuous, la pasa al estado de cierre

Ejecuta ` + "`axiom integrations time <propose|record> --help`" + ` para ver las banderas de cada verbo.
Todos aceptan --cwd y --json (` + integrationsTimeSchema + `).
`

const integrationsTimeProposeHelp = `Uso: axiom integrations time propose --change <cambio> --role <rol> [--cwd <ruta>] [--json]

Propone las horas que el rol puede imputar. Es de solo lectura: no escribe en el tracker, en
integrations.yaml ni en .time-pending.yaml. Exige el vínculo de Task del rol (task_not_bound) con su
started_at registrado (role_not_started). Muestra el reloj desde el inicio (o desde la última
imputación de cierre del rol) hasta ahora, recortado a la jornada, la zona horaria usada, la
estimación del vínculo y OriginalEstimate, RemainingWork y CompletedWork leídos del tracker.

El recorte suma, por cada jornada (workday_hours, anclada a la hora local del inicio del rol), solo los
días de workdays; los minutos se truncan y las horas se redondean al cuarto de hora más próximo. La
propuesta no es vinculante: time record toma las horas de su propio --hours. Con una intención
pendiente del rol avisa time_pending sin tocarla.

Exige el plugin activo con el puerto de Tasks (sin él responde not_applicable con código 0 y sin red),
el vínculo del work item y, en checkpointed, la compuerta role-apply:<rol> aprobada con su digest
vigente. El carril ODD no está admitido.

Banderas:
  --change <cambio>  Cambio SDD (obligatorio)
  --role <rol>       Rol del roster sellado (obligatorio)
  --cwd <ruta>       Directorio de partida (por defecto, el directorio de trabajo)
  --json             Imprime el resultado como ` + integrationsTimeSchema + `

Sale con 0 si quedó hecho o no aplica; con 1 si se rechazó; con 2 por un error de uso.
`

const integrationsTimeRecordHelp = `Uso: axiom integrations time record --change <cambio> --role <rol> (--hours <horas> | --resolve-pending applied|discarded | --close-only) [--cwd <ruta>] [--json]

Imputa las horas a la Task del rol: lee su rev, suma las horas a CompletedWork, fija RemainingWork en
max(0, RemainingWork - horas) y escribe con test /rev. Solo tras una escritura con éxito registra en
integrations.yaml una imputación con actor (la identidad git), minutos, fecha, rol, item_id y un source
único por cierre (role.close/<rol>/<n>). Un conflicto de rev termina con rev_conflict, sin reaplicar.
En continuous, además, pasa la Task al estado task_state_close (un fallo de esa transición es un aviso);
en checkpointed no toca el estado, que aplica role.apply.approved.

Antes de escribir deja la intención en .time-pending.yaml (estado local, nunca publicado). Si el
resultado es incierto (red, tiempo agotado, 5xx, conflicto tras reintento no verificado) la conserva y
avisa time_pending; el siguiente time record del rol la reconcilia releyendo la Task de la intención:
sin aplicar, la borra y sigue; aplicada, registra la imputación y avisa time_record_recovered (esa
ejecución no aplica sus --hours); en otro caso rechaza con time_pending_ambiguous.

Banderas:
  --change <cambio>             Cambio SDD (obligatorio)
  --role <rol>                  Rol del roster sellado (obligatorio)
  --hours <horas>               Horas a imputar: decimal con punto, de dos decimales como mucho, 0 < h <= 1000
  --resolve-pending <decisión>  applied registra la imputación de la intención sin tocar el tracker;
                                discarded la borra. Incompatible con --hours y --close-only
  --close-only                  Solo en continuous: pasa la Task a task_state_close sin imputar horas.
                                Incompatible con --hours y --resolve-pending
  --cwd <ruta>                  Directorio de partida (por defecto, el directorio de trabajo)
  --json                        Imprime el resultado como ` + integrationsTimeSchema + `

Sale con 0 si quedó hecho o no aplica; con 1 si se rechazó; con 2 por un error de uso.
`

// Costuras de prueba del reloj: el instante actual y la zona horaria local (D7).
var (
	trackerNow      = time.Now
	trackerLocation = func() *time.Location { return time.Local }

	// trackerRecordTimeEntry es la costura de prueba sobre el registro de la imputación.
	trackerRecordTimeEntry = integrations.RecordTimeEntry
)

// Valores por defecto de las claves del ciclo de vida que lee el verbo: los del manifiesto.
const (
	defaultTimeWorkdayHours = 8
	resolvePendingApplied   = "applied"
	resolvePendingDiscarded = "discarded"
)

var defaultTimeWorkdays = []string{"mon", "tue", "wed", "thu", "fri"}

type timeZoneJSON struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Offset       string `json:"offset"`
}

type timeClockJSON struct {
	StartedAt    string   `json:"started_at"`
	Since        string   `json:"since"`
	Now          string   `json:"now"`
	Seconds      int64    `json:"seconds"`
	WorkdayHours int      `json:"workday_hours"`
	Workdays     []string `json:"workdays"`
}

type timeProposalJSON struct {
	Hours   float64 `json:"hours"`
	Minutes int     `json:"minutes"`
}

type timeTrackerJSON struct {
	Rev              int      `json:"rev"`
	State            string   `json:"state,omitempty"`
	OriginalEstimate *float64 `json:"original_estimate"`
	RemainingWork    float64  `json:"remaining_work"`
	CompletedWork    float64  `json:"completed_work"`
}

type timeRecordedJSON struct {
	Hours         float64 `json:"hours"`
	Minutes       int     `json:"minutes"`
	Source        string  `json:"source"`
	Date          string  `json:"date"`
	Actor         string  `json:"actor"`
	ItemID        string  `json:"item_id"`
	CompletedWork float64 `json:"completed_work"`
	RemainingWork float64 `json:"remaining_work"`
	Recovered     bool    `json:"recovered"`
}

type timeTransitionJSON struct {
	State   string `json:"state"`
	Changed bool   `json:"changed"`
}

// timeJSON es la salida `axiom.integrations-time/v1`.
type timeJSON struct {
	Schema        string              `json:"schema"`
	Verb          string              `json:"verb"`
	Change        string              `json:"change"`
	Plugin        string              `json:"plugin"`
	Role          string              `json:"role"`
	Status        string              `json:"status"`
	Task          *trackerItemRef     `json:"task"`
	EstimateHours *float64            `json:"estimate_hours"`
	Timezone      *timeZoneJSON       `json:"timezone"`
	Clock         *timeClockJSON      `json:"clock"`
	Proposal      *timeProposalJSON   `json:"proposal"`
	Tracker       *timeTrackerJSON    `json:"tracker"`
	Recorded      *timeRecordedJSON   `json:"recorded"`
	Transition    *timeTransitionJSON `json:"transition"`
	// Pending es lo que ocurrió con la intención del diario: not_applied, recovered, applied,
	// discarded o kept (se conserva por un resultado incierto).
	Pending  string           `json:"pending"`
	Refusal  *trackerRefusal  `json:"refusal"`
	Warnings []trackerWarning `json:"warnings"`
}

func newTimeJSON(verb, change, role string) *timeJSON {
	return &timeJSON{Schema: integrationsTimeSchema, Verb: verb, Change: change, Role: role, Warnings: []trackerWarning{}}
}

// runIntegrationsTime ejecuta `axiom integrations time`.
func runIntegrationsTime(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, integrationsTimeHelp)
		return integrationsExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, integrationsTimeHelp)
		return integrationsExitOK
	case "propose":
		return runIntegrationsTimePropose(rest, stdout, stderr)
	case "record":
		return runIntegrationsTimeRecord(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para integrations time; los disponibles son propose y record.\n", subcommand)
		fmt.Fprint(stdout, integrationsTimeHelp)
		return integrationsExitUsage
	}
}

// timeRun reúne lo común de propose y record una vez resueltas las precondiciones.
type timeRun struct {
	ctx    *trackerContext
	verb   string
	role   string
	run    context.Context
	out    *timeJSON
	loc    *time.Location
	stderr io.Writer
}

func (r *timeRun) command(verb string, extra ...string) string {
	parts := append([]string{"axiom integrations time", verb, "--change", r.ctx.Change, "--role", r.role}, extra...)
	return strings.Join(parts, " ")
}

func (r *timeRun) recordRetry() string { return r.command("record") }

func (r *timeRun) warn(code, text, next string) {
	w := newTrackerWarning(code, text, next)
	w.Role = r.role
	r.out.Warnings = append(r.out.Warnings, w)
}

// classify traduce un error del puerto: el 404 de una Task nombra el rescate con tasks bind.
func (r *timeRun) classify(err error, retry string) *trackerRefusal {
	if errors.Is(err, integrations.ErrTaskNotFound) {
		return newTrackerRefusal("task_not_found", err.Error()+".",
			fmt.Sprintf("axiom integrations tasks bind --change %s --role %s=<id> --replace", r.ctx.Change, r.role))
	}
	return classifyTrackerError(r.ctx, err, retry)
}

// finish anota el diario de continuous, imprime y devuelve el código de salida.
func (r *timeRun) finish(asJSON bool, stdout, stderr io.Writer) int {
	verb := "time " + r.verb
	if r.out.Refusal != nil {
		r.ctx.noteRefusal(verb, r.role, r.out.Refusal)
	} else {
		r.ctx.note(verb, r.role, "done", timeDoneText(r.out), "")
	}
	r.ctx.noteWarnings(verb, r.out.Warnings)
	return printTimeOutput(r.out, asJSON, stdout, stderr)
}

func timeDoneText(out *timeJSON) string {
	switch {
	case out.Verb == "propose" && out.Proposal != nil:
		return fmt.Sprintf("propuesta de %s h", formatHours(out.Proposal.Hours))
	case out.Recorded != nil:
		return fmt.Sprintf("imputadas %s h (%s)", formatHours(out.Recorded.Hours), out.Recorded.Source)
	case out.Pending != "":
		return "intención " + out.Pending
	case out.Transition != nil:
		return "cierre sin imputación: Task en " + out.Transition.State
	}
	return "hecho"
}

func formatHours(h float64) string { return fmt.Sprintf("%g", h) }

func roundHours2(h float64) float64 { return math.Round(h*100) / 100 }

func trackerJSONOf(task integrations.TaskItem) *timeTrackerJSON {
	out := &timeTrackerJSON{Rev: task.Rev, State: task.State, RemainingWork: task.Hours.Remaining, CompletedWork: task.Hours.Completed}
	if task.Hours.HasOriginal {
		out.OriginalEstimate = hoursPtr(task.Hours.Original)
	}
	return out
}

// timeTarget resuelve las precondiciones de D14 para `time *`: carpeta, plugin, vínculo del work item,
// pertenencia del rol al roster (error de uso) y compuerta role-apply:<rol> en checkpointed. Devuelve
// un mensaje de uso no vacío cuando una bandera no es válida frente al roster o al estilo.
func timeTarget(loc workspace.Location, cwd, change, role, verbHelp string, closeOnly bool, stderr io.Writer) (trackerResolution, string) {
	resolution := trackerTarget(loc, cwd, change, trackerOptions{Verb: verbHelp}, stderr)
	if resolution.NotApplicable || resolution.Refusal != nil {
		return resolution, ""
	}
	ctx := resolution.Ctx
	if ctx.Sealed != nil {
		if !trackerRosterHas(ctx.Roster, role) {
			return resolution, fmt.Sprintf("--role nombra el rol %s, que no pertenece al roster sellado (%s)", role, strings.Join(ctx.Roster, ", "))
		}
		if closeOnly && ctx.Sealed.Config.ExecutionStyle != kickoff.ExecutionContinuous {
			return resolution, "--close-only solo se admite en un cambio continuous: en checkpointed el estado lo aplica el evento role.apply.approved"
		}
	}
	if refusal := checkTrackerStyle(ctx, trackerOptions{Verb: verbHelp, Gate: kickoff.RoleApplyGate(role)}); refusal != nil {
		return trackerRefuseResolution(ctx, refusal), ""
	}
	return resolution, ""
}

// roleTaskLink devuelve el vínculo de Task del rol en el plugin activo.
func (r *timeRun) roleTaskLink() (*integrations.Link, *trackerRefusal) {
	links, err := integrations.TaskLinks(r.ctx.Folder)
	if err != nil {
		return nil, newTrackerRefusal("integrations_links_invalid",
			fmt.Sprintf("no se puede leer integrations.yaml del cambio %s: %v.", r.ctx.Change, err),
			fmt.Sprintf("axiom integrations sync --change %s", r.ctx.Change))
	}
	for i := range links {
		if links[i].Plugin == r.ctx.PluginID && links[i].Role == r.role {
			return &links[i], nil
		}
	}
	return nil, newTrackerRefusal("task_not_bound",
		fmt.Sprintf("el rol %s del cambio %s no tiene vínculo de Task: no hay Task a la que imputar horas.", r.role, r.ctx.Change),
		fmt.Sprintf("axiom integrations tasks create --change %s", r.ctx.Change))
}

func (r *timeRun) notStarted() *trackerRefusal {
	return newTrackerRefusal("role_not_started",
		fmt.Sprintf("el rol %s del cambio %s no tiene started_at registrado: no hay reloj que medir.", r.role, r.ctx.Change),
		fmt.Sprintf("axiom branch ensure --change %s --role %s", r.ctx.Change, r.role))
}

func (r *timeRun) pendingInvalid(err error) *trackerRefusal {
	return newTrackerRefusal("time_pending_invalid",
		fmt.Sprintf("no se puede leer %s del cambio %s: %v.", integrations.PendingTimeFile, r.ctx.Change, err), r.command("propose"))
}

func closeEntries(entries []integrations.TimeEntry) []trackersync.CloseEntry {
	out := make([]trackersync.CloseEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, trackersync.CloseEntry{Source: e.Source, RecordedAt: e.RecordedAt})
	}
	return out
}

func timeZoneOf(loc *time.Location, at time.Time) *timeZoneJSON {
	abbreviation, offset := at.In(loc).Zone()
	name := loc.String()
	if name == "Local" {
		if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" {
			name = tz
		} else {
			name = abbreviation
		}
	}
	sign := "+"
	if offset < 0 {
		sign, offset = "-", -offset
	}
	return &timeZoneJSON{Name: name, Abbreviation: abbreviation, Offset: fmt.Sprintf("%s%02d:%02d", sign, offset/3600, offset%3600/60)}
}

func lifecycleInt(values map[string]any, key string, fallback int) int {
	switch n := values[key].(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return fallback
}

func lifecycleList(values map[string]any, key string, fallback []string) []string {
	switch l := values[key].(type) {
	case []string:
		if len(l) > 0 {
			return append([]string(nil), l...)
		}
	case []any:
		var out []string
		for _, v := range l {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return append([]string(nil), fallback...)
}

// closeState es el estado de cierre efectivo: task_state_close y, sin valor, task_state_apply_approved.
func closeState(values map[string]any) string {
	for _, key := range []string{"task_state_close", "task_state_apply_approved"} {
		if s, ok := values[key].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func timeUsageFlag(flags *flag.FlagSet, name string) bool {
	set := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// ---------------------------------------------------------------- propose

func runIntegrationsTimePropose(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations time propose", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "cambio")
	roleFlag := flags.String("role", "", "rol")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsTimeProposeHelp, stdout, stderr); done {
		return code
	}
	const verbHelp = "time propose"
	if flags.NArg() > 0 {
		return trackerUsage(stderr, verbHelp, "sobran argumentos: %s", strings.Join(flags.Args(), " "))
	}
	if msg := trackerChangeUsageError(*change); msg != "" {
		return trackerUsage(stderr, verbHelp, "%s", msg)
	}
	role := canonicalEnsureRole(*roleFlag)
	if role == "" {
		return trackerUsage(stderr, verbHelp, "--role es obligatorio")
	}
	loc, ok := locateIntegrationsWorkspace(*cwd, stderr)
	if !ok {
		return integrationsExitFail
	}

	resolution, usage := timeTarget(loc, *cwd, *change, role, verbHelp, false, stderr)
	if usage != "" {
		return trackerUsage(stderr, verbHelp, "%s", usage)
	}
	out := newTimeJSON("propose", *change, role)
	switch {
	case resolution.NotApplicable:
		out.Status = trackerStatusNotApplicable
		return printTimeOutput(out, *asJSON, stdout, stderr)
	case resolution.Refusal != nil:
		out.Refusal = resolution.Refusal
		if resolution.Ctx != nil {
			out.Plugin = resolution.Ctx.PluginID
			resolution.Ctx.noteRefusal(verbHelp, role, resolution.Refusal)
		}
		return printTimeOutput(out, *asJSON, stdout, stderr)
	}
	ctx := resolution.Ctx
	out.Plugin = ctx.PluginID
	run, cancel := trackerRunContext()
	defer cancel()
	r := &timeRun{ctx: ctx, verb: "propose", role: role, run: run, out: out, loc: trackerLocation(), stderr: stderr}
	r.propose()
	return r.finish(*asJSON, stdout, stderr)
}

func (r *timeRun) propose() {
	ctx := r.ctx
	link, refusal := r.roleTaskLink()
	if refusal != nil {
		r.out.Refusal = refusal
		return
	}
	r.out.Task = taskRef(link.ID, link.URL)
	r.out.EstimateHours = link.EstimateHours
	if link.StartedAt == nil {
		r.out.Refusal = r.notStarted()
		return
	}

	// Solo lectura: una intención pendiente se avisa, nunca se reconcilia ni se toca.
	pending, hasPending, err := integrations.PendingTimeFor(ctx.Folder, r.role)
	if err != nil {
		r.out.Refusal = r.pendingInvalid(err)
		return
	}
	if hasPending {
		r.out.Pending = "kept"
		r.warn("time_pending", fmt.Sprintf("el rol %s tiene una imputación pendiente de %s h sobre la Task %s: se reconcilia al empezar el siguiente time record.",
			r.role, formatHours(pending.Hours), pending.ItemID), r.recordRetry())
	}

	task, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.TaskItem, error) {
		return t.ValidateTask(r.run, integrations.TaskRequest{PortRequest: req, TaskID: link.ID, ParentID: ctx.WorkItem.ID})
	})
	if err != nil {
		r.out.Refusal = classifyBindError(ctx, err, fmt.Sprintf("axiom integrations tasks bind --change %s --role %s=<id> --replace", ctx.Change, r.role))
		return
	}
	r.out.Tracker = trackerJSONOf(task)

	entries, err := integrations.TimeEntries(ctx.Folder)
	if err != nil {
		r.out.Refusal = newTrackerRefusal("integrations_links_invalid", fmt.Sprintf("no se puede leer integrations.yaml del cambio %s: %v.", ctx.Change, err),
			fmt.Sprintf("axiom integrations sync --change %s", ctx.Change))
		return
	}
	now := trackerNow()
	workdayHours := lifecycleInt(ctx.Config.Values, "workday_hours", defaultTimeWorkdayHours)
	workdays := lifecycleList(ctx.Config.Values, "workdays", defaultTimeWorkdays)
	proposal, err := trackersync.ProposeHours(trackersync.ClockInput{
		StartedAt: *link.StartedAt, LastClose: trackersync.LastCloseAt(r.role, closeEntries(entries)), Now: now,
		Loc: r.loc, WorkdayHours: workdayHours, Workdays: workdays,
	})
	if err != nil {
		r.out.Refusal = newTrackerRefusal("clock_invalid", fmt.Sprintf("no se puede calcular el reloj de jornada: %v.", err),
			"axiom workspace validate")
		return
	}
	r.out.Timezone = timeZoneOf(r.loc, now)
	r.out.Clock = &timeClockJSON{
		StartedAt: link.StartedAt.UTC().Format(time.RFC3339), Since: proposal.Since.UTC().Format(time.RFC3339), Now: now.UTC().Format(time.RFC3339),
		Seconds: proposal.Seconds, WorkdayHours: workdayHours, Workdays: workdays,
	}
	r.out.Proposal = &timeProposalJSON{Hours: proposal.Hours, Minutes: proposal.Minutes}
	r.out.Status = trackerStatusDone
}

// ---------------------------------------------------------------- record

func runIntegrationsTimeRecord(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations time record", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "cambio")
	roleFlag := flags.String("role", "", "rol")
	hoursFlag := flags.String("hours", "", "horas a imputar")
	resolveFlag := flags.String("resolve-pending", "", "applied o discarded")
	closeOnly := flags.Bool("close-only", false, "cierra la Task sin imputar horas")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsTimeRecordHelp, stdout, stderr); done {
		return code
	}
	const verbHelp = "time record"
	if flags.NArg() > 0 {
		return trackerUsage(stderr, verbHelp, "sobran argumentos: %s", strings.Join(flags.Args(), " "))
	}
	if msg := trackerChangeUsageError(*change); msg != "" {
		return trackerUsage(stderr, verbHelp, "%s", msg)
	}
	role := canonicalEnsureRole(*roleFlag)
	if role == "" {
		return trackerUsage(stderr, verbHelp, "--role es obligatorio")
	}
	hoursSet, resolveSet := timeUsageFlag(flags, "hours"), timeUsageFlag(flags, "resolve-pending")
	switch {
	case !hoursSet && !resolveSet && !*closeOnly:
		return trackerUsage(stderr, verbHelp, "hace falta una de --hours, --resolve-pending o --close-only")
	case hoursSet && resolveSet:
		return trackerUsage(stderr, verbHelp, "--resolve-pending es incompatible con --hours")
	case *closeOnly && (hoursSet || resolveSet):
		return trackerUsage(stderr, verbHelp, "--close-only es incompatible con --hours y con --resolve-pending")
	}
	var (
		hours   float64
		minutes int
	)
	if hoursSet {
		var err error
		if hours, minutes, err = trackersync.ParseHours(*hoursFlag); err != nil {
			return trackerUsage(stderr, verbHelp, "%v", err)
		}
	}
	resolve := strings.ToLower(strings.TrimSpace(*resolveFlag))
	if resolveSet && resolve != resolvePendingApplied && resolve != resolvePendingDiscarded {
		return trackerUsage(stderr, verbHelp, "--resolve-pending admite applied o discarded y recibió %q", *resolveFlag)
	}
	loc, ok := locateIntegrationsWorkspace(*cwd, stderr)
	if !ok {
		return integrationsExitFail
	}

	resolution, usage := timeTarget(loc, *cwd, *change, role, verbHelp, *closeOnly, stderr)
	if usage != "" {
		return trackerUsage(stderr, verbHelp, "%s", usage)
	}
	out := newTimeJSON("record", *change, role)
	switch {
	case resolution.NotApplicable:
		out.Status = trackerStatusNotApplicable
		return printTimeOutput(out, *asJSON, stdout, stderr)
	case resolution.Refusal != nil:
		out.Refusal = resolution.Refusal
		if resolution.Ctx != nil {
			out.Plugin = resolution.Ctx.PluginID
			resolution.Ctx.noteRefusal(verbHelp, role, resolution.Refusal)
		}
		return printTimeOutput(out, *asJSON, stdout, stderr)
	}
	ctx := resolution.Ctx
	out.Plugin = ctx.PluginID
	run, cancel := trackerRunContext()
	defer cancel()
	r := &timeRun{ctx: ctx, verb: "record", role: role, run: run, out: out, loc: trackerLocation(), stderr: stderr}

	pending, hasPending, err := integrations.PendingTimeFor(ctx.Folder, role)
	if err != nil {
		out.Refusal = r.pendingInvalid(err)
		return r.finish(*asJSON, stdout, stderr)
	}
	switch {
	case resolveSet:
		if !hasPending {
			return trackerUsage(stderr, verbHelp, "el rol %s no tiene ninguna intención pendiente que resolver", role)
		}
		r.resolvePending(pending, resolve)
	case hasPending && r.reconcile(pending):
		// La reconciliación terminó el verbo: recuperada o rechazada.
	case *closeOnly:
		r.closeOnly()
	default:
		r.recordHours(hours, minutes)
	}
	return r.finish(*asJSON, stdout, stderr)
}

// reconcile es D9: al empezar `time record`, relee la Task de la intención (su item_id, aunque el
// vínculo del rol haya cambiado) y decide. Devuelve true cuando el verbo debe terminar aquí.
func (r *timeRun) reconcile(p integrations.PendingTime) bool {
	ctx := r.ctx
	task, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.TaskItem, error) {
		return t.ReadTask(r.run, integrations.TaskRequest{PortRequest: req, TaskID: p.ItemID})
	})
	if err != nil {
		r.out.Pending = "kept"
		r.out.Refusal = r.classify(err, r.recordRetry())
		return true
	}
	r.out.Tracker = trackerJSONOf(task)
	switch {
	case task.Rev == p.BaseRev:
		if err := integrations.ClearPendingTime(ctx.Folder, r.role); err != nil {
			r.out.Refusal = newTrackerRefusal("write_failed",
				fmt.Sprintf("no se ha podido borrar la intención no aplicada de %s: %v.", integrations.PendingTimeFile, err), r.recordRetry())
			return true
		}
		r.out.Pending = "not_applied"
		return false
	case task.Rev > p.BaseRev && equalHours(task.Hours.Completed, p.ExpectedCompleted) && equalHours(task.Hours.Remaining, p.ExpectedRemaining):
		if !r.registerPending(p, true) {
			return true
		}
		r.out.Pending = "recovered"
		r.warn("time_record_recovered", fmt.Sprintf("la imputación pendiente de %s h sobre la Task %s ya estaba aplicada en el tracker: se ha registrado (%s) y esta ejecución no aplica horas nuevas.",
			formatHours(p.Hours), p.ItemID, p.Source), r.command("propose"))
		r.closeAfter(p.ItemID)
		return true
	}
	r.out.Pending = "kept"
	r.out.Refusal = newTrackerRefusal("time_pending_ambiguous",
		fmt.Sprintf("la Task %s está en la revisión %d con CompletedWork %s y RemainingWork %s; la intención pendiente leyó la revisión %d y esperaba %s y %s: no consta si se aplicó.",
			p.ItemID, task.Rev, formatHours(task.Hours.Completed), formatHours(task.Hours.Remaining), p.BaseRev, formatHours(p.ExpectedCompleted), formatHours(p.ExpectedRemaining)),
		r.command("record", "--resolve-pending", "applied|discarded"))
	return true
}

// registerPending registra la imputación de la intención en integrations.yaml (idempotente por source)
// y borra la intención. Devuelve false, con el rechazo puesto, si el registro falla.
func (r *timeRun) registerPending(p integrations.PendingTime, recovered bool) bool {
	ctx := r.ctx
	entry := integrations.TimeEntry{Plugin: ctx.PluginID, Role: r.role, ItemID: p.ItemID, Minutes: p.Minutes, Date: p.Date, Source: p.Source, Actor: p.Actor}
	if _, err := trackerRecordTimeEntry(ctx.Folder, ctx.PluginID, entry, trackerNow().UTC()); err != nil {
		r.out.Pending = "kept"
		r.out.Refusal = newTrackerRefusal("write_failed",
			fmt.Sprintf("la imputación %s consta en la intención pero no se ha podido registrar en integrations.yaml: %v.", p.Source, err), r.recordRetry())
		return false
	}
	if err := integrations.ClearPendingTime(ctx.Folder, r.role); err != nil {
		r.warn("time_pending", fmt.Sprintf("la imputación %s está registrada pero no se ha podido borrar la intención de %s: %v.", p.Source, integrations.PendingTimeFile, err), r.recordRetry())
	}
	r.out.Recorded = &timeRecordedJSON{Hours: p.Hours, Minutes: p.Minutes, Source: p.Source, Date: p.Date, Actor: p.Actor, ItemID: p.ItemID,
		CompletedWork: p.ExpectedCompleted, RemainingWork: p.ExpectedRemaining, Recovered: recovered}
	return true
}

// resolvePending es `--resolve-pending applied|discarded`: nunca toca el tracker salvo la transición
// de cierre de continuous tras `applied`.
func (r *timeRun) resolvePending(p integrations.PendingTime, decision string) {
	r.out.Task = taskRef(p.ItemID, "")
	if decision == resolvePendingDiscarded {
		if err := integrations.ClearPendingTime(r.ctx.Folder, r.role); err != nil {
			r.out.Refusal = newTrackerRefusal("write_failed", fmt.Sprintf("no se ha podido borrar la intención de %s: %v.", integrations.PendingTimeFile, err), r.recordRetry())
			return
		}
		r.out.Pending = resolvePendingDiscarded
		r.out.Status = trackerStatusDone
		return
	}
	if !r.registerPending(p, false) {
		return
	}
	r.out.Pending = resolvePendingApplied
	r.out.Status = trackerStatusDone
	r.closeAfter(p.ItemID)
}

// currentTaskID es la Task que cierra la transición: la del vínculo vigente del rol y, sin él, la dada.
func (r *timeRun) currentTaskID(fallback string) string {
	if link, refusal := r.roleTaskLink(); refusal == nil {
		return link.ID
	}
	return fallback
}

// transition lleva la Task al estado de cierre (D21) con la regla AdvanceUnlessClosed y sin asignar.
func (r *timeRun) transition(taskID string) error {
	ctx := r.ctx
	state := closeState(ctx.Config.Values)
	if state == "" {
		return errors.New("el plugin no define task_state_close ni task_state_apply_approved: revisa la configuración con `axiom workspace validate`")
	}
	result, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.TransitionResult, error) {
		return t.TransitionTask(r.run, integrations.TransitionRequest{PortRequest: req, TaskID: taskID, State: state, Rule: integrations.AdvanceUnlessClosed})
	})
	if err != nil {
		return err
	}
	r.out.Transition = &timeTransitionJSON{State: state, Changed: result.StateChanged}
	r.out.Warnings = append(r.out.Warnings, fromPortWarnings(r.role, result.Warnings)...)
	return nil
}

// closeAfter aplica la transición de cierre tras una imputación, una recuperación o `applied`, solo en
// continuous. Un fallo no deshace nada: es el aviso `state_transition_failed`.
func (r *timeRun) closeAfter(fallbackTaskID string) {
	if !r.ctx.Continuous() {
		return
	}
	if err := r.transition(r.currentTaskID(fallbackTaskID)); err != nil {
		r.warn("state_transition_failed", fmt.Sprintf("la imputación quedó registrada pero la Task no ha pasado al estado de cierre: %v.", err),
			r.command("record", "--close-only"))
	}
}

// closeOnly es `--close-only` (solo continuous): transiciona sin imputar; exige vínculo, no started_at.
func (r *timeRun) closeOnly() {
	link, refusal := r.roleTaskLink()
	if refusal != nil {
		r.out.Refusal = refusal
		return
	}
	r.out.Task = taskRef(link.ID, link.URL)
	r.out.EstimateHours = link.EstimateHours
	if err := r.transition(link.ID); err != nil {
		r.out.Refusal = r.classify(err, r.command("record", "--close-only"))
		return
	}
	r.out.Status = trackerStatusDone
}

// recordHours es el flujo normal de D9: leer, escribir la intención, PATCH con test /rev y registrar.
func (r *timeRun) recordHours(hours float64, minutes int) {
	ctx := r.ctx
	link, refusal := r.roleTaskLink()
	if refusal != nil {
		r.out.Refusal = refusal
		return
	}
	r.out.Task = taskRef(link.ID, link.URL)
	r.out.EstimateHours = link.EstimateHours
	if link.StartedAt == nil {
		r.out.Refusal = r.notStarted()
		return
	}
	retry := r.command("record", "--hours", formatHours(hours))

	base, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.TaskItem, error) {
		return t.ValidateTask(r.run, integrations.TaskRequest{PortRequest: req, TaskID: link.ID, ParentID: ctx.WorkItem.ID})
	})
	if err != nil {
		r.out.Refusal = classifyBindError(ctx, err, fmt.Sprintf("axiom integrations tasks bind --change %s --role %s=<id> --replace", ctx.Change, r.role))
		return
	}
	if base.ID == "" {
		base.ID = link.ID
	}
	r.out.Tracker = trackerJSONOf(base)

	entries, err := integrations.TimeEntries(ctx.Folder)
	if err != nil {
		r.out.Refusal = newTrackerRefusal("integrations_links_invalid", fmt.Sprintf("no se puede leer integrations.yaml del cambio %s: %v.", ctx.Change, err),
			fmt.Sprintf("axiom integrations sync --change %s", ctx.Change))
		return
	}
	actor, actorWarning := defaultActor(ctx.Loc.WorkspaceRoot, gateActorTarget)
	now := trackerNow()
	intent := integrations.PendingTime{
		Role: r.role, ItemID: link.ID, Hours: hours, Minutes: minutes, Actor: actor, Date: now.In(r.loc).Format("2006-01-02"),
		Source: trackersync.NextCloseSource(r.role, closeEntries(entries)), BaseRev: base.Rev,
		ExpectedCompleted: roundHours2(base.Hours.Completed + hours),
		ExpectedRemaining: roundHours2(math.Max(0, base.Hours.Remaining-hours)),
	}
	if err := integrations.PutPendingTime(ctx.Folder, intent); err != nil {
		r.out.Refusal = newTrackerRefusal("write_failed",
			fmt.Sprintf("no se ha podido escribir la intención en %s antes de tocar el tracker: %v.", integrations.PendingTimeFile, err), retry)
		return
	}

	result, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.HoursResult, error) {
		return t.RecordHours(r.run, integrations.HoursRequest{PortRequest: req, Base: base, Hours: hours})
	})
	if err != nil {
		r.writeFailed(err, retry)
		return
	}
	if !r.registerPending(intent, result.Recovered) {
		return
	}
	emitTimeActorWarning(r.stderr, actorWarning)
	r.out.Status = trackerStatusDone
	r.out.Tracker = trackerJSONOf(result.Task)
	r.closeAfter(link.ID)
}

func emitTimeActorWarning(stderr io.Writer, warning string) {
	if warning != "" && stderr != nil {
		fmt.Fprintln(stderr, warning)
	}
}

// writeFailed clasifica un fallo del PATCH de horas (D9, puntos 5 y 6): solo cuando consta que no se
// aplicó (conflicto en el primer intento o 4xx que no es conflicto) se borra la intención; en todo
// resultado incierto se conserva y se avisa time_pending.
func (r *timeRun) writeFailed(err error, retry string) {
	var conflict *integrations.RevConflictError
	firmConflict := errors.As(err, &conflict) && !conflict.AfterRetry
	notApplied := firmConflict || errors.Is(err, integrations.ErrTrackerRejected) ||
		errors.Is(err, integrations.ErrTaskNotFound) || errors.Is(err, integrations.ErrFieldMissing)

	switch {
	case firmConflict:
		r.out.Tracker = &timeTrackerJSON{Rev: conflict.Rev, RemainingWork: conflict.Hours.Remaining, CompletedWork: conflict.Hours.Completed}
		if conflict.Hours.HasOriginal {
			r.out.Tracker.OriginalEstimate = hoursPtr(conflict.Hours.Original)
		}
		r.out.Refusal = newTrackerRefusal("rev_conflict",
			fmt.Sprintf("la Task cambió desde la lectura: ahora está en la revisión %d con CompletedWork %s y RemainingWork %s; no se reaplica la escritura ni se registra la imputación.",
				conflict.Rev, formatHours(conflict.Hours.Completed), formatHours(conflict.Hours.Remaining)),
			r.command("propose"))
	default:
		r.out.Refusal = r.classify(err, retry)
	}

	if notApplied {
		if clearErr := integrations.ClearPendingTime(r.ctx.Folder, r.role); clearErr != nil {
			r.out.Pending = "kept"
			r.warn("time_pending", fmt.Sprintf("no se ha podido borrar la intención no aplicada de %s: %v.", integrations.PendingTimeFile, clearErr), r.recordRetry())
		}
		return
	}
	r.out.Pending = "kept"
	r.warn("time_pending", "no consta si la escritura de horas se aplicó: la intención queda en "+integrations.PendingTimeFile+
		" y el siguiente time record del rol la reconcilia releyendo la Task.", r.recordRetry())
}

// printTimeOutput imprime la salida y devuelve el código: 0 con done o not_applicable, 1 con refused.
func printTimeOutput(out *timeJSON, asJSON bool, stdout, stderr io.Writer) int {
	if out.Status == "" {
		out.Status = trackerStatusDone
		if out.Refusal != nil {
			out.Status = trackerStatusRefused
		}
	}
	if out.Refusal != nil {
		out.Status = trackerStatusRefused
	}
	if asJSON {
		writeIntegrationsJSON(stdout, out)
	} else {
		printTimeText(out, stdout)
	}
	if out.Refusal != nil {
		fmt.Fprintf(stderr, "Error: %s\n", out.Refusal)
	}
	if out.Status == trackerStatusDone || out.Status == trackerStatusNotApplicable {
		return integrationsExitOK
	}
	return integrationsExitFail
}

func printTimeText(out *timeJSON, stdout io.Writer) {
	if out.Status == trackerStatusNotApplicable {
		fmt.Fprintf(stdout, "No hay plugin activo con el puerto de Tasks: `time %s` no aplica al cambio %s.\n", out.Verb, out.Change)
		return
	}
	fmt.Fprintf(stdout, "time %s del cambio %s, rol %s (plugin %s): %s\n", out.Verb, out.Change, out.Role, out.Plugin, out.Status)
	if out.Task != nil {
		fmt.Fprintf(stdout, "  Task: %s\n", out.Task.ID)
	}
	if out.EstimateHours != nil {
		fmt.Fprintf(stdout, "  Estimación del vínculo: %s h\n", formatHours(*out.EstimateHours))
	}
	if out.Tracker != nil {
		original := "sin estimación"
		if out.Tracker.OriginalEstimate != nil {
			original = formatHours(*out.Tracker.OriginalEstimate) + " h"
		}
		fmt.Fprintf(stdout, "  Tracker (rev %d): OriginalEstimate %s, RemainingWork %s h, CompletedWork %s h\n",
			out.Tracker.Rev, original, formatHours(out.Tracker.RemainingWork), formatHours(out.Tracker.CompletedWork))
	}
	if out.Clock != nil && out.Timezone != nil {
		fmt.Fprintf(stdout, "  Reloj: desde %s hasta %s (zona %s, %s %s), jornada de %d h\n",
			out.Clock.Since, out.Clock.Now, out.Timezone.Name, out.Timezone.Abbreviation, out.Timezone.Offset, out.Clock.WorkdayHours)
	}
	if out.Proposal != nil {
		fmt.Fprintf(stdout, "  Propuesta: %s h (%d min)\n", formatHours(out.Proposal.Hours), out.Proposal.Minutes)
	}
	if out.Recorded != nil {
		fmt.Fprintf(stdout, "  Imputación %s: %s h (%d min) el %s por %s\n", out.Recorded.Source, formatHours(out.Recorded.Hours), out.Recorded.Minutes, out.Recorded.Date, out.Recorded.Actor)
	}
	if out.Transition != nil {
		fmt.Fprintf(stdout, "  Estado de cierre: %s (cambiado: %t)\n", out.Transition.State, out.Transition.Changed)
	}
	if out.Pending != "" {
		fmt.Fprintf(stdout, "  Intención pendiente: %s\n", out.Pending)
	}
	printTrackerWarnings(stdout, out.Warnings)
}
