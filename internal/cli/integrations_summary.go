package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/specsummary"
	"github.com/IGutierrezZ/axiom/v3/internal/trackersync"
)

// `axiom integrations summary push` (eventos-ado, prov:INT:4 y D12): escribe en el work item de origen
// del cambio el resumen de su spec como HTML simple y las etiquetas `prefijo + ID funcional` de los
// requisitos, en una sola escritura y solo si algo difiere. Es el único verbo de tracker que admite la
// carpeta archivada del cambio: tras archivar, reenvía el resumen con los IDs definitivos.

const integrationsSummaryHelp = `Uso: axiom integrations summary push --change <cambio> [--cwd <ruta>] [--json]

Escribe en el work item vinculado al cambio la descripción (el resumen de la spec como HTML simple:
capacidades y, por cada requisito, su título y su primer párrafo) y las etiquetas de requisito
(req_tag_prefix + ID funcional, con tope en req_tag_max), con una sola escritura (test /rev) y solo si la
descripción o las etiquetas difieren de las que ya tiene. Las etiquetas ajenas se conservan; una
descripción editada a mano se sustituye por diseño y la salida lo indica con description.changed.

Resuelve la carpeta ordinaria del cambio y, si ya no existe, la archivada más reciente: sobre esta última
traduce los provisionales con ids.yaml y reenvía el resumen con los IDs definitivos (hazlo antes de
` + "`axiom sdd publish --archive`" + `, que cierra el work item). Exige el plugin activo con el puerto de
Tasks (sin él responde not_applicable con código 0 y sin red), el vínculo del work item del cambio y, en
checkpointed, la compuerta spec aprobada con su digest vigente; en continuous, que existan las deltas de
specs/. El carril ODD no está admitido.

Banderas:
  --change <cambio>  Cambio SDD (obligatorio)
  --cwd <ruta>       Directorio de partida (por defecto, el directorio de trabajo)
  --json             Imprime el resultado como ` + integrationsSummarySchema + `

Sale con 0 si quedó hecho o no aplica; con 1 si se rechazó; con 2 por un error de uso.
`

const integrationsSummaryFamilyHelp = `Uso: axiom integrations summary push --change <cambio> [banderas]

  push   Escribe el resumen de la spec y las etiquetas de requisito en el work item de origen del cambio

Ejecuta ` + "`axiom integrations summary push --help`" + ` para ver las banderas.
Acepta --cwd y --json (` + integrationsSummarySchema + `).
`

// Carpetas de la salida `axiom.integrations-summary/v1`.
const (
	summaryFolderOrdinary = "ordinary"
	summaryFolderArchived = "archived"

	defaultSummaryTagPrefix = "req:"
	defaultSummaryTagMax    = 50
)

type summaryDescriptionJSON struct {
	Changed bool   `json:"changed"`
	Bytes   int    `json:"bytes"`
	SHA256  string `json:"sha256"`
}

type summaryTagsJSON struct {
	Prefix  string   `json:"prefix"`
	Desired []string `json:"desired"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Sent    bool     `json:"sent"`
}

// summaryJSON es la salida `axiom.integrations-summary/v1`.
type summaryJSON struct {
	Schema      string                  `json:"schema"`
	Change      string                  `json:"change"`
	Plugin      string                  `json:"plugin"`
	Status      string                  `json:"status"`
	Folder      string                  `json:"folder"`
	WorkItem    *trackerItemRef         `json:"work_item"`
	Description *summaryDescriptionJSON `json:"description"`
	Tags        *summaryTagsJSON        `json:"tags"`
	Wrote       bool                    `json:"wrote"`
	Refusal     *trackerRefusal         `json:"refusal"`
	Warnings    []trackerWarning        `json:"warnings"`
}

func newSummaryJSON(change string) *summaryJSON {
	return &summaryJSON{Schema: integrationsSummarySchema, Change: change, Warnings: []trackerWarning{}}
}

// runIntegrationsSummary ejecuta `axiom integrations summary`.
func runIntegrationsSummary(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, integrationsSummaryFamilyHelp)
		return integrationsExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, integrationsSummaryFamilyHelp)
		return integrationsExitOK
	case "push":
		return runIntegrationsSummaryPush(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para integrations summary; el único disponible es push.\n", subcommand)
		fmt.Fprint(stdout, integrationsSummaryFamilyHelp)
		return integrationsExitUsage
	}
}

// summaryDeltasExist dice si el cambio tiene alguna delta `specs/<capacidad>/spec.md`.
func summaryDeltasExist(root string) (string, bool) {
	matches, _ := filepath.Glob(filepath.Join(root, "specs", "*", "spec.md"))
	return "las deltas de specs/", len(matches) > 0
}

// summaryTagSettings lee el prefijo y el tope de etiquetas de la configuración del plugin; la carga de
// axiom.yaml ya aplicó los valores por defecto del manifiesto y esta lectura los repite por seguridad.
func summaryTagSettings(values map[string]any) (string, int) {
	prefix, max := defaultSummaryTagPrefix, defaultSummaryTagMax
	if s, ok := values["req_tag_prefix"].(string); ok && s != "" {
		prefix = s
	}
	switch n := values["req_tag_max"].(type) {
	case int:
		max = n
	case int64:
		max = int(n)
	case float64:
		max = int(n)
	}
	return prefix, max
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func runIntegrationsSummaryPush(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations summary push", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "cambio")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsSummaryHelp, stdout, stderr); done {
		return code
	}
	const verbHelp = "summary push"
	if flags.NArg() > 0 {
		return trackerUsage(stderr, verbHelp, "sobran argumentos: %s", strings.Join(flags.Args(), " "))
	}
	if msg := trackerChangeUsageError(*change); msg != "" {
		return trackerUsage(stderr, verbHelp, "%s", msg)
	}
	loc, ok := locateIntegrationsWorkspace(*cwd, stderr)
	if !ok {
		return integrationsExitFail
	}

	resolution := trackerTarget(loc, *cwd, *change, trackerOptions{
		Verb:          verbHelp,
		Gate:          kickoff.GateSpec,
		Artifact:      summaryDeltasExist,
		AllowArchived: true,
	}, stderr)
	out := newSummaryJSON(*change)
	switch {
	case resolution.NotApplicable:
		out.Status = trackerStatusNotApplicable
		return printSummaryOutput(out, *asJSON, stdout, stderr)
	case resolution.Refusal != nil:
		out.Refusal = resolution.Refusal
		if resolution.Ctx != nil {
			out.Plugin = resolution.Ctx.PluginID
			resolution.Ctx.noteRefusal(verbHelp, "", resolution.Refusal)
		}
		return printSummaryOutput(out, *asJSON, stdout, stderr)
	}
	ctx := resolution.Ctx
	out.Plugin = ctx.PluginID
	out.WorkItem = taskRef(ctx.WorkItem.ID, ctx.WorkItem.URL)
	out.Folder = summaryFolderOrdinary
	if ctx.Archived {
		out.Folder = summaryFolderArchived
	}
	return pushSummary(ctx, out, loc.Declarations.IDPattern, *asJSON, stdout, stderr)
}

// pushSummary arma el modelo, lo renderiza y lo envía con UpdateSummary.
func pushSummary(ctx *trackerContext, out *summaryJSON, idPattern func() (reqid.Pattern, error), asJSON bool, stdout, stderr io.Writer) int {
	const verb = "summary push"
	retry := fmt.Sprintf("axiom integrations summary push --change %s", ctx.Change)
	finish := func() int {
		if out.Refusal != nil {
			ctx.noteRefusal(verb, "", out.Refusal)
		} else {
			ctx.note(verb, "", "done", summaryDoneText(out), "")
		}
		ctx.noteWarnings(verb, out.Warnings)
		return printSummaryOutput(out, asJSON, stdout, stderr)
	}

	// Un sdd.id_pattern ilegible no impide el resumen: el modelo cae al patrón predeterminado.
	pattern, _ := idPattern()
	model, err := trackersync.SpecModel(ctx.Root, trackersync.SpecModelOptions{Pattern: pattern, Archived: ctx.Archived})
	if err != nil {
		var modelErr *trackersync.SpecModelError
		if errors.As(err, &modelErr) {
			out.Refusal = newTrackerRefusal(modelErr.Code, modelErr.Reason+".",
				fmt.Sprintf("axiom sdd archive-check --cwd %s --change %s --json", ctx.Loc.WorkspaceRoot, ctx.Change))
		} else {
			out.Refusal = newTrackerRefusal("artifact_missing", fmt.Sprintf("no se pueden leer las deltas del cambio %s: %v.", ctx.Change, err),
				fmt.Sprintf("axiom sdd status --change %s", ctx.Change))
		}
		return finish()
	}
	if len(model.Model.Capabilities) == 0 {
		out.Refusal = newTrackerRefusal("artifact_missing",
			fmt.Sprintf("el cambio %s no tiene deltas de spec que resumir en %s.", ctx.Change, filepath.Join(ctx.Root, "specs")),
			fmt.Sprintf("axiom sdd status --change %s", ctx.Change))
		return finish()
	}

	prefix, tagMax := summaryTagSettings(ctx.Config.Values)
	html := specsummary.Render(model.Model)
	desired := make([]string, 0, len(model.IDs))
	for _, id := range model.IDs {
		desired = append(desired, prefix+id)
	}
	digest := sha256.Sum256([]byte(html))

	run, cancel := trackerRunContext()
	defer cancel()
	result, err := callTasks(ctx, func(t integrations.TaskTracker, req integrations.PortRequest) (integrations.SummaryResult, error) {
		return t.UpdateSummary(run, integrations.SummaryRequest{
			PortRequest: req, ItemID: ctx.WorkItem.ID, HTML: html, TagPrefix: prefix, Tags: desired, TagMax: tagMax,
		})
	})
	if err != nil {
		out.Refusal = classifyTrackerError(ctx, err, retry)
		return finish()
	}
	out.Description = &summaryDescriptionJSON{Changed: result.DescriptionChanged, Bytes: len(html), SHA256: hex.EncodeToString(digest[:])}
	out.Tags = &summaryTagsJSON{Prefix: prefix, Desired: nonNilStrings(desired), Added: nonNilStrings(result.Added),
		Removed: nonNilStrings(result.Removed), Sent: result.TagsSent}
	out.Wrote = result.Wrote
	out.Warnings = append(out.Warnings, fromPortWarnings("", result.Warnings)...)
	out.Status = trackerStatusDone
	return finish()
}

func summaryDoneText(out *summaryJSON) string {
	if out.Wrote {
		return fmt.Sprintf("resumen escrito (%s)", out.Folder)
	}
	return fmt.Sprintf("resumen sin cambios (%s)", out.Folder)
}

// printSummaryOutput imprime la salida y devuelve el código: 0 con done o not_applicable, 1 con refused.
func printSummaryOutput(out *summaryJSON, asJSON bool, stdout, stderr io.Writer) int {
	if out.Status == "" {
		out.Status = trackerStatusDone
		if out.Refusal != nil {
			out.Status = trackerStatusRefused
		}
	}
	if asJSON {
		writeIntegrationsJSON(stdout, out)
	} else {
		if out.Status == trackerStatusNotApplicable {
			fmt.Fprintf(stdout, "No hay plugin activo con el puerto de Tasks: `summary push` no aplica al cambio %s.\n", out.Change)
		} else {
			fmt.Fprintf(stdout, "summary push del cambio %s (plugin %s): %s\n", out.Change, out.Plugin, out.Status)
		}
		if out.WorkItem != nil {
			fmt.Fprintf(stdout, "  Work item: %s\n", out.WorkItem.ID)
		}
		if out.Description != nil {
			fmt.Fprintf(stdout, "  Descripción: %d bytes, sha256 %s, cambiada: %t\n", out.Description.Bytes, out.Description.SHA256, out.Description.Changed)
		}
		if out.Tags != nil {
			fmt.Fprintf(stdout, "  Etiquetas (%s): %d deseadas, %d añadidas, %d retiradas, enviadas: %t\n",
				out.Tags.Prefix, len(out.Tags.Desired), len(out.Tags.Added), len(out.Tags.Removed), out.Tags.Sent)
		}
		if out.Status == trackerStatusDone {
			if out.Wrote {
				fmt.Fprintln(stdout, "  Escrito en el tracker.")
			} else {
				fmt.Fprintln(stdout, "  Sin cambios: no se ha escrito nada.")
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
