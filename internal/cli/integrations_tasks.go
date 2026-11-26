package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/trackersync"
)

// `axiom integrations tasks create|bind` (eventos-ado, prov:INT:2 y prov:INT:3): crean o vinculan la
// Task hija del work item de origen de cada rol del roster sellado. Son verbos síncronos, fuera de la
// outbox, que registran el vínculo `kind: task` en integrations.yaml tras cada respuesta con éxito.

const integrationsTasksHelp = `Uso: axiom integrations tasks <create|bind> --change <cambio> [banderas]

  create   Crea o reutiliza la Task hija del work item de origen de cada rol del roster sellado
  bind     Vincula a mano una Task existente a un rol (verbo de rescate; con --replace sustituye el vínculo)

Ejecuta ` + "`axiom integrations tasks <create|bind> --help`" + ` para ver las banderas de cada verbo.
Todos aceptan --cwd y --json (` + integrationsTasksSchema + `).
`

const integrationsTasksCreateHelp = `Uso: axiom integrations tasks create --change <cambio> [--estimate <rol>=<horas>]... [--cwd <ruta>] [--json]

Crea, por cada rol del roster sellado, la Task hija del work item vinculado al cambio, o reutiliza la
que ya existe: primero el vínculo kind task de integrations.yaml y, sin él, la Task que lleva la marca
axiom-task:<cambio>:<rol>. Registra el vínculo en cuanto recibe la respuesta. La estimación de cada rol
es la de --estimate o, si falta, la de la sección "## Estimación por rol" de design.md. Un rol que ya
tiene vínculo y cuya estimación confirmada difiere de la del vínculo actualiza OriginalEstimate y
RemainingWork de la Task con una sola escritura (test /rev). Un rol que falla no impide los demás.

Exige el plugin activo con el puerto de Tasks (sin él responde not_applicable con código 0 y sin red),
el vínculo del work item del cambio y, en checkpointed, la compuerta design aprobada con su digest
vigente; en continuous, que exista design.md. El carril ODD no está admitido.

Banderas:
  --change <cambio>          Cambio SDD (obligatorio)
  --estimate <rol>=<horas>   Estimación del rol: decimal con punto, positiva, de dos decimales como mucho y
                             de 1000 como máximo (repetible, un rol una sola vez)
  --cwd <ruta>               Directorio de partida (por defecto, el directorio de trabajo)
  --json                     Imprime el resultado como ` + integrationsTasksSchema + `

Sale con 0 si todo quedó hecho o no aplica; con 1 si algún rol se rechazó (status partial o refused);
con 2 por un error de uso.
`

const integrationsTasksBindHelp = `Uso: axiom integrations tasks bind --change <cambio> --role <rol>=<id> [--estimate <rol>=<horas>] [--replace] [--cwd <ruta>] [--json]

Vincula a mano la Task <id> al rol. Antes de escribir comprueba con el tracker que el elemento existe
(task_not_found), que es una Task (not_a_task) y que cuelga del work item del cambio
(not_child_of_work_item). Con una estimación (--estimate o la sección de design.md) y una Task sin
OriginalEstimate escribe OriginalEstimate y RemainingWork; si la Task ya los tenía no toca las horas.
Vincular otra Task a un rol ya vinculado se rechaza con task_already_bound salvo con --replace, que
revalida la Task nueva y sustituye el vínculo conservando started_at y las imputaciones. No exige
compuerta ni artefacto en ningún estilo.

Banderas:
  --change <cambio>          Cambio SDD (obligatorio)
  --role <rol>=<id>          Rol del roster sellado y Task a vincular (obligatorio)
  --estimate <rol>=<horas>   Estimación del mismo rol (opcional; mismo dominio que en create)
  --replace                  Sustituye el vínculo de Task del rol si ya existe
  --cwd <ruta>               Directorio de partida (por defecto, el directorio de trabajo)
  --json                     Imprime el resultado como ` + integrationsTasksSchema + `

Sale con 0 si quedó vinculada o no aplica; con 1 si se rechazó; con 2 por un error de uso.
`

// Resultados por rol y orígenes de la estimación de la salida `axiom.integrations-tasks/v1`.
const (
	tasksResultCreated         = "created"
	tasksResultFoundByMark     = "found_by_mark"
	tasksResultRepaired        = "repaired"
	tasksResultReusedLink      = "reused_link"
	tasksResultEstimateUpdated = "estimate_updated"
	tasksResultBound           = "bound"
	tasksResultReplaced        = "replaced"
	tasksResultUnchanged       = "unchanged"
	tasksResultRefused         = "refused"

	estimateSourceFlag   = "flag"
	estimateSourceDesign = "design"
	estimateSourceLink   = "link"
	estimateSourceNone   = "none"
)

// trackerBindTask es la costura de prueba sobre el registro del vínculo de Task.
var trackerBindTask = integrations.BindTask

// tasksRoleJSON es el resultado de un rol.
type tasksRoleJSON struct {
	Role           string          `json:"role"`
	Result         string          `json:"result"`
	Task           *trackerItemRef `json:"task"`
	EstimateHours  *float64        `json:"estimate_hours"`
	EstimateSource string          `json:"estimate_source"`
	Refusal        *trackerRefusal `json:"refusal"`
}

// tasksJSON es la salida `axiom.integrations-tasks/v1`.
type tasksJSON struct {
	Schema   string           `json:"schema"`
	Verb     string           `json:"verb"`
	Change   string           `json:"change"`
	Plugin   string           `json:"plugin"`
	Status   string           `json:"status"`
	WorkItem *trackerItemRef  `json:"work_item"`
	Roles    []tasksRoleJSON  `json:"roles"`
	Refusal  *trackerRefusal  `json:"refusal"`
	Warnings []trackerWarning `json:"warnings"`
}

func newTasksJSON(verb, change string) *tasksJSON {
	return &tasksJSON{Schema: integrationsTasksSchema, Verb: verb, Change: change, Roles: []tasksRoleJSON{}, Warnings: []trackerWarning{}}
}

// tasksStatus resume el sobre: done si ningún rol se rechazó, refused si todos, partial si unos sí y
// otros no.
func (o *tasksJSON) finalizeStatus() {
	if o.Status != "" {
		return
	}
	refused := 0
	for _, role := range o.Roles {
		if role.Refusal != nil {
			refused++
		}
	}
	switch {
	case o.Refusal != nil, refused > 0 && refused == len(o.Roles):
		o.Status = trackerStatusRefused
	case refused > 0:
		o.Status = trackerStatusPartial
	default:
		o.Status = trackerStatusDone
	}
}

// runIntegrationsTasks ejecuta `axiom integrations tasks`.
func runIntegrationsTasks(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, integrationsTasksHelp)
		return integrationsExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, integrationsTasksHelp)
		return integrationsExitOK
	case "create":
		return runIntegrationsTasksCreate(rest, stdout, stderr)
	case "bind":
		return runIntegrationsTasksBind(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para integrations tasks; los disponibles son create y bind.\n", subcommand)
		fmt.Fprint(stdout, integrationsTasksHelp)
		return integrationsExitUsage
	}
}

// trackerUsage imprime un error de uso y devuelve la salida 2.
func trackerUsage(stderr io.Writer, verbHelp, format string, args ...any) int {
	fmt.Fprintf(stderr, "Error: %s. Ejecuta `axiom integrations %s --help`.\n", fmt.Sprintf(format, args...), verbHelp)
	return integrationsExitUsage
}

// parseEstimateFlags valida todas las banderas `--estimate` antes de cualquier petición: forma
// <rol>=<horas>, dominio de las horas y un rol una sola vez.
func parseEstimateFlags(values []string) (map[string]float64, string) {
	estimates := map[string]float64{}
	for _, value := range values {
		role, hours, err := trackersync.ParseEstimateFlag(value)
		if err != nil {
			return nil, fmt.Sprintf("--estimate %v", err)
		}
		if _, dup := estimates[role]; dup {
			return nil, fmt.Sprintf("--estimate repite el rol %s", role)
		}
		estimates[role] = hours
	}
	return estimates, ""
}

func equalHours(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func hoursPtr(h float64) *float64 { return &h }

func taskRef(id, url string) *trackerItemRef {
	if id == "" {
		return nil
	}
	return &trackerItemRef{ID: id, URL: url}
}

// printTasksOutput imprime la salida y devuelve el código de salida: 0 con done o not_applicable, 1 con
// refused o partial.
func printTasksOutput(out *tasksJSON, asJSON bool, stdout, stderr io.Writer) int {
	out.finalizeStatus()
	if asJSON {
		writeIntegrationsJSON(stdout, out)
	} else {
		switch out.Status {
		case trackerStatusNotApplicable:
			fmt.Fprintf(stdout, "No hay plugin activo con el puerto de Tasks: `tasks %s` no aplica al cambio %s.\n", out.Verb, out.Change)
		default:
			fmt.Fprintf(stdout, "tasks %s del cambio %s (plugin %s): %s\n", out.Verb, out.Change, out.Plugin, out.Status)
		}
		if out.WorkItem != nil {
			fmt.Fprintf(stdout, "  Work item: %s\n", out.WorkItem.ID)
		}
		for _, role := range out.Roles {
			line := fmt.Sprintf("  rol %s: %s", role.Role, role.Result)
			if role.Task != nil {
				line += fmt.Sprintf(", Task %s", role.Task.ID)
			}
			if role.EstimateHours != nil {
				line += fmt.Sprintf(", estimación %s h (%s)", strconv.FormatFloat(*role.EstimateHours, 'f', -1, 64), role.EstimateSource)
			}
			fmt.Fprintln(stdout, line)
			if role.Refusal != nil {
				fmt.Fprintf(stdout, "    %s\n", role.Refusal)
			}
		}
		printTrackerWarnings(stdout, out.Warnings)
	}
	if out.Refusal != nil {
		fmt.Fprintf(stderr, "Error: %s\n", out.Refusal)
	}
	if out.Status == trackerStatusDone || out.Status == trackerStatusNotApplicable {
		return integrationsExitOK
	}
	return integrationsExitFail
}

// tasksRun reúne lo común de create y bind una vez resueltas las precondiciones.
type tasksRun struct {
	ctx     *trackerContext
	verb    string
	run     context.Context
	out     *tasksJSON
	estFlag map[string]float64
}

func newTasksRun(ctx *trackerContext, verb string, run context.Context, estFlag map[string]float64) *tasksRun {
	out := newTasksJSON(verb, ctx.Change)
	out.Plugin = ctx.PluginID
	out.WorkItem = taskRef(ctx.WorkItem.ID, ctx.WorkItem.URL)
	return &tasksRun{ctx: ctx, verb: verb, run: run, out: out, estFlag: estFlag}
}

// designEstimates lee la sección de estimaciones de design.md. Sin fichero o sin sección no es un
// error: devuelve un mapa vacío. Una línea inválida es *trackersync.EstimateError.
func (r *tasksRun) designEstimates() (map[string]float64, error) {
	data, err := os.ReadFile(filepath.Join(r.ctx.Root, "design.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]float64{}, nil
		}
		return nil, fmt.Errorf("no se puede leer design.md: %w", err)
	}
	return trackersync.ParseEstimates(string(data), r.ctx.Roster)
}

// estimateInvalid es el rechazo del verbo por una sección de estimaciones inválida.
func (r *tasksRun) estimateInvalid(err error) *trackerRefusal {
	return newTrackerRefusal(trackersync.CodeEstimateInvalid,
		fmt.Sprintf("la sección %q de design.md no es válida: %v.", trackersync.EstimateSectionTitle, err),
		fmt.Sprintf("axiom integrations tasks %s --change %s --estimate <rol>=<horas>", r.verb, r.ctx.Change))
}

// finish anota el diario de continuous, imprime y devuelve el código de salida.
func (r *tasksRun) finish(asJSON bool, stdout, stderr io.Writer) int {
	r.out.finalizeStatus()
	if r.out.Refusal != nil {
		r.ctx.noteRefusal("tasks "+r.verb, "", r.out.Refusal)
	}
	for _, role := range r.out.Roles {
		if role.Refusal != nil {
			if role.Refusal != r.out.Refusal {
				r.ctx.noteRefusal("tasks "+r.verb, role.Role, role.Refusal)
			}
			continue
		}
		r.ctx.note("tasks "+r.verb, role.Role, "done", fmt.Sprintf("rol %s: %s", role.Role, role.Result), "")
	}
	r.ctx.noteWarnings("tasks "+r.verb, r.out.Warnings)
	return printTasksOutput(r.out, asJSON, stdout, stderr)
}

// callTasks invoca una operación del puerto de Tasks del plugin activo con secretos acotados.
func callTasks[T any](ctx *trackerContext, call func(t integrations.TaskTracker, req integrations.PortRequest) (T, error)) (T, error) {
	return integrations.WithTaskTracker(ctx.Plugin, ctx.Config, secrets.Default(), call)
}

// ---------------------------------------------------------------- create

func runIntegrationsTasksCreate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations tasks create", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "cambio")
	var estimates branchRepeatedFlag
	flags.Var(&estimates, "estimate", "estimación <rol>=<horas>; repetible")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsTasksCreateHelp, stdout, stderr); done {
		return code
	}
	const verbHelp = "tasks create"
	if msg := trackerChangeUsageError(*change); msg != "" {
		return trackerUsage(stderr, verbHelp, "%s", msg)
	}
	estFlag, msg := parseEstimateFlags(estimates)
	if msg != "" {
		return trackerUsage(stderr, verbHelp, "%s", msg)
	}
	loc, ok := locateIntegrationsWorkspace(*cwd, stderr)
	if !ok {
		return integrationsExitFail
	}

	resolution := trackerTarget(loc, *cwd, *change, trackerOptions{
		Verb: verbHelp,
		Gate: kickoff.GateDesign,
		Artifact: func(root string) (string, bool) {
			return "design.md", fileExists(filepath.Join(root, "design.md"))
		},
	}, stderr)
	out := newTasksJSON("create", *change)
	switch {
	case resolution.NotApplicable:
		out.Status = trackerStatusNotApplicable
		return printTasksOutput(out, *asJSON, stdout, stderr)
	case resolution.Refusal != nil:
		out.Refusal = resolution.Refusal
		if resolution.Ctx != nil {
			out.Plugin = resolution.Ctx.PluginID
			resolution.Ctx.noteRefusal(verbHelp, "", resolution.Refusal)
		}
		return printTasksOutput(out, *asJSON, stdout, stderr)
	}
	ctx := resolution.Ctx
	for role := range estFlag {
		if !trackerRosterHas(ctx.Roster, role) {
			return trackerUsage(stderr, verbHelp, "--estimate nombra el rol %s, que no pertenece al roster sellado (%s)", role, strings.Join(ctx.Roster, ", "))
		}
	}

	run, cancel := trackerRunContext()
	defer cancel()
	t := newTasksRun(ctx, "create", run, estFlag)
	return t.create(*asJSON, stdout, stderr)
}

func (r *tasksRun) create(asJSON bool, stdout, stderr io.Writer) int {
	ctx := r.ctx
	design := map[string]float64{}
	needDesign := false
	for _, role := range ctx.Roster {
		if _, ok := r.estFlag[role]; !ok {
			needDesign = true
		}
	}
	if needDesign {
		parsed, err := r.designEstimates()
		if err != nil {
			var invalid *trackersync.EstimateError
			if errors.As(err, &invalid) {
				r.out.Refusal = r.estimateInvalid(err)
			} else {
				r.out.Refusal = newTrackerRefusal("artifact_missing", err.Error()+".", fmt.Sprintf("axiom sdd status --change %s", ctx.Change))
			}
			return r.finish(asJSON, stdout, stderr)
		}
		design = parsed
	}

	links, err := integrations.TaskLinks(ctx.Folder)
	if err != nil {
		r.out.Refusal = newTrackerRefusal("integrations_links_invalid", fmt.Sprintf("no se puede leer integrations.yaml: %v.", err),
			fmt.Sprintf("axiom integrations sync --change %s", ctx.Change))
		return r.finish(asJSON, stdout, stderr)
	}
	for _, role := range ctx.Roster {
		var link *integrations.Link
		for i := range links {
			if links[i].Plugin == ctx.PluginID && links[i].Role == role {
				link = &links[i]
				break
			}
		}
		result, warnings := r.createRole(role, link, design)
		r.out.Roles = append(r.out.Roles, result)
		r.out.Warnings = append(r.out.Warnings, warnings...)
	}
	return r.finish(asJSON, stdout, stderr)
}

// confirmedEstimate resuelve la estimación confirmada del rol: --estimate, si no design.md.
func (r *tasksRun) confirmedEstimate(role string, design map[string]float64) (float64, string) {
	if hours, ok := r.estFlag[role]; ok {
		return hours, estimateSourceFlag
	}
	if hours, ok := design[role]; ok {
		return hours, estimateSourceDesign
	}
	return 0, estimateSourceNone
}

func (r *tasksRun) createRole(role string, link *integrations.Link, design map[string]float64) (tasksRoleJSON, []trackerWarning) {
	ctx := r.ctx
	result := tasksRoleJSON{Role: role, Result: tasksResultRefused, EstimateSource: estimateSourceNone}
	retry := fmt.Sprintf("axiom integrations tasks create --change %s", ctx.Change)
	bindHint := fmt.Sprintf("%s o, para vincular a mano, axiom integrations tasks bind --change %s --role %s=<id>", retry, ctx.Change, role)
	estimate, source := r.confirmedEstimate(role, design)

	if link != nil {
		result.Task = taskRef(link.ID, link.URL)
		return r.reuseLink(role, link, estimate, source, result, retry)
	}

	if source == estimateSourceNone {
		result.Refusal = newTrackerRefusal("estimate_missing",
			fmt.Sprintf("el rol %s no tiene estimación: ni --estimate ni una línea en %q de design.md.", role, trackersync.EstimateSectionTitle),
			fmt.Sprintf("axiom integrations tasks create --change %s --estimate %s=<horas>", ctx.Change, role))
		return result, nil
	}
	result.EstimateHours, result.EstimateSource = hoursPtr(estimate), source

	created, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.CreateTaskResult, error) {
		return t.EnsureChildTask(r.run, integrations.CreateTaskRequest{
			PortRequest: req, ParentID: ctx.WorkItem.ID,
			Title: ctx.Change + " / " + role, Mark: strings.ToLower("axiom-task:" + ctx.Change + ":" + role), EstimateHours: estimate,
		})
	})
	if err != nil {
		result.Refusal = classifyTrackerError(ctx, err, bindHint)
		return result, nil
	}
	warnings := fromPortWarnings(role, created.Warnings)
	result.Task = taskRef(created.Task.ID, created.Task.URL)
	switch created.Outcome {
	case integrations.TaskFoundByMark:
		result.Result = tasksResultFoundByMark
	case integrations.TaskRepaired:
		result.Result = tasksResultRepaired
	default:
		result.Result = tasksResultCreated
	}
	bound := integrations.Link{Plugin: ctx.PluginID, Kind: integrations.TaskLinkKind, ID: created.Task.ID, URL: created.Task.URL,
		Role: role, Source: integrations.TasksCreateSource, EstimateHours: hoursPtr(estimate)}
	if _, err := trackerBindTask(ctx.Folder, ctx.PluginID, bound, time.Now().UTC()); err != nil {
		result.Result = tasksResultRefused
		result.Refusal = newTrackerRefusal("write_failed",
			fmt.Sprintf("la Task %s del rol %s existe en el tracker pero no se ha podido registrar su vínculo en integrations.yaml: %v.", created.Task.ID, role, err), retry)
	}
	return result, warnings
}

// reuseLink aplica D17: con un vínculo ya registrado, la estimación igual no escribe nada y la
// distinta se propaga al tracker con un único PATCH con test /rev y, solo tras el éxito, al vínculo.
func (r *tasksRun) reuseLink(role string, link *integrations.Link, estimate float64, source string, result tasksRoleJSON, retry string) (tasksRoleJSON, []trackerWarning) {
	ctx := r.ctx
	result.Result = tasksResultReusedLink
	if link.EstimateHours != nil {
		result.EstimateHours, result.EstimateSource = hoursPtr(*link.EstimateHours), estimateSourceLink
	}
	if source == estimateSourceNone || (link.EstimateHours != nil && equalHours(*link.EstimateHours, estimate)) {
		if source != estimateSourceNone {
			result.EstimateSource = source
		}
		return result, nil
	}

	base, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.TaskItem, error) {
		return t.ReadTask(r.run, integrations.TaskRequest{PortRequest: req, TaskID: link.ID})
	})
	if err != nil {
		result.Result, result.Refusal = tasksResultRefused, classifyTrackerError(ctx, err, retry)
		return result, nil
	}
	if _, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.EstimateResult, error) {
		return t.SetEstimate(r.run, integrations.EstimateRequest{PortRequest: req, TaskID: link.ID, Hours: estimate, Mode: integrations.EstimateReplace, Base: base})
	}); err != nil {
		result.Result, result.Refusal = tasksResultRefused, classifyTrackerError(ctx, err, retry)
		return result, nil
	}
	updated := *link
	updated.EstimateHours = hoursPtr(estimate)
	if _, err := trackerBindTask(ctx.Folder, ctx.PluginID, updated, time.Now().UTC()); err != nil {
		result.Result = tasksResultRefused
		result.Refusal = newTrackerRefusal("write_failed",
			fmt.Sprintf("la estimación de la Task %s se escribió en el tracker pero no se ha podido actualizar el vínculo del rol %s: %v.", link.ID, role, err), retry)
		return result, nil
	}
	result.Result, result.EstimateHours, result.EstimateSource = tasksResultEstimateUpdated, hoursPtr(estimate), source
	return result, nil
}

// ---------------------------------------------------------------- bind

var bindTaskIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func runIntegrationsTasksBind(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations tasks bind", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "cambio")
	roleFlag := flags.String("role", "", "rol y Task: <rol>=<id>")
	var estimates branchRepeatedFlag
	flags.Var(&estimates, "estimate", "estimación <rol>=<horas>")
	replace := flags.Bool("replace", false, "sustituye el vínculo de Task del rol")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsTasksBindHelp, stdout, stderr); done {
		return code
	}
	const verbHelp = "tasks bind"
	if msg := trackerChangeUsageError(*change); msg != "" {
		return trackerUsage(stderr, verbHelp, "%s", msg)
	}
	role, taskID, ok := strings.Cut(strings.TrimSpace(*roleFlag), "=")
	role, taskID = canonicalEnsureRole(role), strings.TrimSpace(taskID)
	if !ok || role == "" || !bindTaskIDPattern.MatchString(taskID) {
		return trackerUsage(stderr, verbHelp, "--role es obligatorio y tiene la forma <rol>=<id>")
	}
	estFlag, msg := parseEstimateFlags(estimates)
	if msg != "" {
		return trackerUsage(stderr, verbHelp, "%s", msg)
	}
	for estimated := range estFlag {
		if estimated != role {
			return trackerUsage(stderr, verbHelp, "--estimate nombra el rol %s y --role es %s: en bind solo se admite la estimación del rol vinculado", estimated, role)
		}
	}
	loc, ok := locateIntegrationsWorkspace(*cwd, stderr)
	if !ok {
		return integrationsExitFail
	}

	resolution := trackerTarget(loc, *cwd, *change, trackerOptions{Verb: verbHelp}, stderr)
	out := newTasksJSON("bind", *change)
	switch {
	case resolution.NotApplicable:
		out.Status = trackerStatusNotApplicable
		return printTasksOutput(out, *asJSON, stdout, stderr)
	case resolution.Refusal != nil:
		out.Refusal = resolution.Refusal
		if resolution.Ctx != nil {
			out.Plugin = resolution.Ctx.PluginID
			resolution.Ctx.noteRefusal(verbHelp, "", resolution.Refusal)
		}
		return printTasksOutput(out, *asJSON, stdout, stderr)
	}
	ctx := resolution.Ctx
	run, cancel := trackerRunContext()
	defer cancel()
	t := newTasksRun(ctx, "bind", run, estFlag)
	if ctx.Sealed == nil {
		t.out.Refusal = newTrackerRefusal("kickoff_not_sealed",
			fmt.Sprintf("el cambio %s no tiene kickoff sellado: sin él no hay roster de roles.", ctx.Change),
			fmt.Sprintf("axiom sdd kickoff seal --change %s --execution-style checkpointed", ctx.Change))
		return t.finish(*asJSON, stdout, stderr)
	}
	return t.bind(role, taskID, *replace, *asJSON, stdout, stderr)
}

func (r *tasksRun) bind(role, taskID string, replace, asJSON bool, stdout, stderr io.Writer) int {
	ctx := r.ctx
	result := tasksRoleJSON{Role: role, Result: tasksResultRefused, EstimateSource: estimateSourceNone}
	refuse := func(refusal *trackerRefusal) int {
		result.Refusal = refusal
		r.out.Roles = append(r.out.Roles, result)
		r.out.Refusal = refusal
		return r.finish(asJSON, stdout, stderr)
	}
	bindCommand := fmt.Sprintf("axiom integrations tasks bind --change %s --role %s=%s", ctx.Change, role, taskID)

	if !trackerRosterHas(ctx.Roster, role) {
		return refuse(newTrackerRefusal("role_not_in_roster",
			fmt.Sprintf("el rol %s no pertenece al roster sellado de %s (roster: %s).", role, ctx.Change, strings.Join(ctx.Roster, ", ")),
			fmt.Sprintf("axiom integrations tasks bind --change %s --role <rol>=%s con un rol del roster: %s", ctx.Change, taskID, strings.Join(ctx.Roster, ", "))))
	}

	links, err := integrations.TaskLinks(ctx.Folder)
	if err != nil {
		return refuse(newTrackerRefusal("integrations_links_invalid", fmt.Sprintf("no se puede leer integrations.yaml: %v.", err),
			fmt.Sprintf("axiom integrations sync --change %s", ctx.Change)))
	}
	var previous *integrations.Link
	for i := range links {
		if links[i].Plugin == ctx.PluginID && links[i].Role == role {
			previous = &links[i]
			break
		}
	}
	if previous != nil && previous.ID != taskID && !replace {
		result.Task = taskRef(previous.ID, previous.URL)
		return refuse(newTrackerRefusal("task_already_bound",
			fmt.Sprintf("el rol %s ya está vinculado a la Task %s; vincular la %s sin --replace no se hace.", role, previous.ID, taskID),
			bindCommand+" --replace"))
	}

	item, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.TaskItem, error) {
		return t.ValidateTask(r.run, integrations.TaskRequest{PortRequest: req, TaskID: taskID, ParentID: ctx.WorkItem.ID})
	})
	if err != nil {
		return refuse(classifyBindError(ctx, err, bindCommand))
	}
	result.Task = taskRef(item.ID, item.URL)

	// Estimación: --estimate, si no design.md, si no la del vínculo anterior.
	estimate, source := 0.0, estimateSourceNone
	if hours, ok := r.estFlag[role]; ok {
		estimate, source = hours, estimateSourceFlag
	} else if designPath := filepath.Join(ctx.Root, "design.md"); fileExists(designPath) {
		parsed, derr := r.designEstimates()
		if derr != nil {
			return refuse(r.estimateInvalid(derr))
		}
		if hours, ok := parsed[role]; ok {
			estimate, source = hours, estimateSourceDesign
		}
	}
	if source == estimateSourceNone && previous != nil && previous.EstimateHours != nil {
		estimate, source = *previous.EstimateHours, estimateSourceLink
	}
	var warnings []trackerWarning
	if source != estimateSourceNone {
		if _, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.EstimateResult, error) {
			return t.SetEstimate(r.run, integrations.EstimateRequest{PortRequest: req, TaskID: taskID, Hours: estimate, Mode: integrations.EstimateIfMissing, Base: item})
		}); err != nil {
			return refuse(classifyTrackerError(ctx, err, bindCommand))
		}
		result.EstimateHours, result.EstimateSource = hoursPtr(estimate), source
	} else {
		warnings = append(warnings, newTrackerWarning("estimate_not_recorded",
			fmt.Sprintf("rol %s: la Task %s se vincula sin estimación: no hay --estimate ni línea en design.md ni vínculo anterior con estimación.", role, taskID),
			fmt.Sprintf("axiom integrations tasks create --change %s --estimate %s=<horas>", ctx.Change, role)))
		warnings[len(warnings)-1].Role = role
	}

	link := integrations.Link{Plugin: ctx.PluginID, Kind: integrations.TaskLinkKind, ID: item.ID, URL: item.URL, Role: role, Source: integrations.TasksBindSource}
	if source != estimateSourceNone {
		link.EstimateHours = hoursPtr(estimate)
	}
	var changed bool
	if replace {
		changed, err = integrations.ReplaceTaskLink(ctx.Folder, ctx.PluginID, role, link, time.Now().UTC())
	} else {
		changed, err = trackerBindTask(ctx.Folder, ctx.PluginID, link, time.Now().UTC())
	}
	if err != nil {
		return refuse(newTrackerRefusal("write_failed",
			fmt.Sprintf("no se ha podido registrar el vínculo de la Task %s al rol %s en integrations.yaml: %v.", taskID, role, err), bindCommand))
	}
	switch {
	case !changed:
		result.Result = tasksResultUnchanged
	case previous != nil && replace && previous.ID != taskID:
		result.Result = tasksResultReplaced
	default:
		result.Result = tasksResultBound
	}
	r.out.Roles = append(r.out.Roles, result)
	r.out.Warnings = append(r.out.Warnings, warnings...)
	return r.finish(asJSON, stdout, stderr)
}

// classifyBindError añade a la clasificación común los rechazos de validación de bind, cuya
// continuación es el propio verbo con otra Task.
func classifyBindError(ctx *trackerContext, err error, bindCommand string) *trackerRefusal {
	for _, known := range []struct {
		target error
		code   string
	}{
		{integrations.ErrTaskNotFound, "task_not_found"},
		{integrations.ErrNotATask, "not_a_task"},
		{integrations.ErrNotChildOfWorkItem, "not_child_of_work_item"},
	} {
		if errors.Is(err, known.target) {
			return newTrackerRefusal(known.code, err.Error(), bindCommand+" con una Task hija del work item "+ctx.WorkItem.ID)
		}
	}
	return classifyTrackerError(ctx, err, bindCommand)
}
