package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// archiveCheckSchema identifica la salida JSON de `axiom sdd archive-check`.
const archiveCheckSchema = "axiom.sdd-archive-check/v1"

// RunSDDArchiveCheck es `axiom sdd archive-check --cwd <ruta> --change <c> [--json]`
// (design D13). Es de solo lectura: aplica la regla del archivado
// ISO (`sddstatus.ISOArchiveRefusal`, la misma del status y de archive-compose) y
// sale con error, con los motivos tipados, si hay hallazgos. Un cambio no ISO, o
// sin hallazgos, sale sin error. La skill `sdd-archive` lo invoca antes de
// componer y antes de cualquier movimiento de carpeta. Informa además, por
// dominio, `alreadyComposed` para reanudar un archivado interrumpido (D17).
//
// Falla cerrado: un kickoff ilegible no es «cambio no ISO», es un error. El
// kickoff lo carga este verbo por su cuenta porque `ISOArchiveRefusal` lo trata
// como ausente (el status ya lo reporta con su error nombrado).
func RunSDDArchiveCheck(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderSDDArchiveCheckHelp(stdout)
	}
	flags := flag.NewFlagSet("sdd-archive-check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", "", "Directorio desde el que resolver el workspaceRoot")
	change := flags.String("change", "", "Cambio activo cuyo archivado se comprueba")
	asJSON := flags.Bool("json", false, "Salida JSON (esquema "+archiveCheckSchema+")")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("sdd archive-check: %w; ejecuta `axiom sdd archive-check --help`", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("sdd archive-check: argumento inesperado %q; ejecuta `axiom sdd archive-check --cwd <dir> --change <cambio> [--json]`", flags.Arg(0))
	}
	if strings.TrimSpace(*change) == "" {
		return errors.New("sdd archive-check requiere --change; ejecuta `axiom sdd archive-check --cwd <dir> --change <cambio> [--json]`")
	}
	if err := gitref.ValidBranchComponent(*change); err != nil || *change == "archive" {
		reason := "es un nombre reservado"
		if err != nil {
			reason = err.Error()
		}
		return fmt.Errorf("sdd archive-check: --change %q no es un nombre de cambio válido: %s; usa el nombre de la carpeta del cambio que proyecta `axiom sdd status --cwd <dir> --json`", *change, reason)
	}
	start := *cwd
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, changeRoot, err := resolveGovernanceChangeRoot(start, *change)
	if err != nil {
		return fmt.Errorf("sdd archive-check: %w", err)
	}

	report := archiveCheckReport{Schema: archiveCheckSchema, Change: *change, Findings: []archiveCheckFinding{}, Domains: []archiveCheckDomain{}}
	sealed, loadErr := kickoff.Load(changeRoot)
	switch {
	case loadErr != nil:
		report.Error = fmt.Sprintf("el kickoff del cambio no se puede leer: %v; corrige o restaura kickoff.yaml y vuelve a ejecutar `axiom sdd archive-check --cwd <dir> --change %s`", loadErr, *change)
	case sealed != nil && sealed.IsISO():
		report.ISO = true
		for _, finding := range sddstatus.ISOArchiveRefusal(loc.WorkspaceRoot, changeRoot) {
			report.Findings = append(report.Findings, archiveCheckFinding{
				Token: string(finding.Code), Subject: finding.Subject, Reason: finding.Reason(),
			})
		}
		// REQ-44.9: sin el patrón accesible no se decide qué está compuesto; el verbo termina con
		// error y la continuación de `axiom workspace validate`.
		pattern, patternErr := projectIDPattern("sdd archive-check", loc)
		if patternErr != nil {
			report.Error = patternErr.Error()
			break
		}
		delta, baseline, readErr := sddstatus.ISOChangeRequirements(loc.PlanningHome, changeRoot)
		if readErr != nil {
			report.Error = fmt.Sprintf("no se pueden leer los artefactos del cambio: %v; comprueba la carpeta del cambio y vuelve a ejecutar `axiom sdd archive-check --cwd <dir> --change %s`", readErr, *change)
			break
		}
		report.Domains = alreadyComposed(delta, baseline, pattern)
	}
	if report.Error == "" {
		assignments, pending, idErr := archiveCheckIdentifiers(loc, changeRoot)
		if idErr != nil {
			report.Error = idErr.Error()
		}
		report.Assignments, report.PendingProvisionals = assignments, pending
	}
	report.OK = report.Error == "" && len(report.Findings) == 0

	if err := renderArchiveCheck(stdout, *asJSON, report); err != nil {
		return err
	}
	switch {
	case report.Error != "":
		return fmt.Errorf("sdd archive-check: %s; revisa el cambio con `axiom sdd status %s --cwd <dir> --json` y corrige lo que no se puede leer antes de archivar", report.Error, *change)
	case !report.OK:
		return fmt.Errorf("sdd archive-check: el archivado del cambio %q está bloqueado por %d motivo(s); corrígelos y vuelve a ejecutar `axiom sdd archive-check --cwd <dir> --change %s` antes de componer o mover la carpeta", *change, len(report.Findings), *change)
	}
	return nil
}

// archiveCheckReport es la salida de archive-check, en texto y en JSON.
type archiveCheckReport struct {
	Schema   string                `json:"schema"`
	Change   string                `json:"change"`
	ISO      bool                  `json:"iso"`
	OK       bool                  `json:"ok"`
	Error    string                `json:"error,omitempty"`
	Findings []archiveCheckFinding `json:"findings"`
	// Assignments es el mapa de provisional a definitivo guardado en ids.yaml y PendingProvisionals
	// los provisionales que las deltas declaran y aún no tienen número (REQ-47.6, REQ-47.8).
	Assignments         []livespec.AssignmentReport `json:"assignments,omitempty"`
	PendingProvisionals []string                    `json:"pendingProvisionals,omitempty"`
	Domains             []archiveCheckDomain        `json:"domains"`
}

type archiveCheckFinding struct {
	Token   string `json:"token"`
	Subject string `json:"subject,omitempty"`
	Reason  string `json:"reason"`
}

// archiveCheckDomain dice si la baseline ya contiene la delta entera de un
// dominio (D17).
type archiveCheckDomain struct {
	Domain          string `json:"domain"`
	AlreadyComposed bool   `json:"alreadyComposed"`
}

func renderArchiveCheck(stdout io.Writer, asJSON bool, report archiveCheckReport) error {
	if asJSON {
		return encodeJSON(stdout, report)
	}
	var b strings.Builder
	switch {
	case !report.ISO && report.Error == "":
		fmt.Fprintf(&b, "El cambio %q no es ISO: el archivado no tiene comprobaciones adicionales.\n", report.Change)
	case report.OK:
		fmt.Fprintf(&b, "El cambio %q cumple las reglas del archivado ISO.\n", report.Change)
	default:
		fmt.Fprintf(&b, "El archivado del cambio %q está bloqueado:\n", report.Change)
		for _, finding := range report.Findings {
			fmt.Fprintf(&b, "  - %s\n", finding.Reason)
		}
		if report.Error != "" {
			fmt.Fprintf(&b, "  - %s\n", report.Error)
		}
	}
	for _, a := range report.Assignments {
		fmt.Fprintf(&b, "Asignación guardada: %s → %s (%s)\n", a.Provisional, a.Definitive, a.Capability)
	}
	for _, token := range report.PendingProvisionals {
		fmt.Fprintf(&b, "Provisional pendiente de asignar: %s\n", token)
	}
	for _, domain := range report.Domains {
		name := domain.Domain
		if name == "" {
			name = "(delta plana)"
		}
		fmt.Fprintf(&b, "Dominio %s: alreadyComposed: %t\n", name, domain.AlreadyComposed)
	}
	_, err := io.WriteString(stdout, b.String())
	return err
}

// alreadyComposed dice, por dominio de la delta, si la baseline ya contiene la
// delta entera (D17, REQ-29.3): todo ADDED con el mismo nombre y bloque, todo
// MODIFIED con el bloque de la delta, todo REMOVED ausente (borrado, o superado:
// con --supersede el compositor renombra el requisito) y todo RENAMED con el
// nombre nuevo presente y el antiguo ausente. La comparación de bloques es la que hace el compositor al
// insertar: bloque de la delta con salto de línea final frente al texto del
// segmento de la baseline, sin ninguna otra normalización. Los dominios salen en
// orden alfabético.
func alreadyComposed(delta, baseline []isoevidence.Requirement, pattern reqid.Pattern) []archiveCheckDomain {
	byDomain := map[string]map[string]string{}
	namesByDomain := map[string][]string{}
	for _, requirement := range baseline {
		if byDomain[requirement.Domain] == nil {
			byDomain[requirement.Domain] = map[string]string{}
		}
		byDomain[requirement.Domain][requirement.Name] = requirement.Text
		namesByDomain[requirement.Domain] = append(namesByDomain[requirement.Domain], requirement.Name)
	}
	composed := map[string]bool{}
	for _, requirement := range delta {
		present, ok := composed[requirement.Domain]
		if !ok {
			present = true
		}
		composed[requirement.Domain] = present && requirementComposed(requirement, byDomain[requirement.Domain], namesByDomain[requirement.Domain], pattern)
	}
	domains := make([]string, 0, len(composed))
	for domain := range composed {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	out := make([]archiveCheckDomain, 0, len(domains))
	for _, domain := range domains {
		out = append(out, archiveCheckDomain{Domain: domain, AlreadyComposed: composed[domain]})
	}
	return out
}

func requirementComposed(requirement isoevidence.Requirement, canonical map[string]string, names []string, pattern reqid.Pattern) bool {
	text, present := canonical[requirement.Name]
	switch requirement.Section {
	case isoevidence.SectionAdded, isoevidence.SectionModified:
		return present && text == ensureNewline(requirement.Text)
	case isoevidence.SectionRemoved:
		// Borrado o superado: en ambos casos el requisito deja de resolverse (con
		// --supersede el compositor lo renombra), y volver a componer fallaría con
		// «no canonical requirement named». Se resuelve como el compositor, por
		// identificador con vuelta al título exacto (REQ-30.22): un REMOVED por
		// identificador con el título obsoleto sigue sin componer mientras el requisito
		// existe bajo otro título.
		_, match := openspecdoc.Resolve(names, requirement.Name, pattern)
		return match == openspecdoc.MatchNone
	case isoevidence.SectionRenamed:
		_, oldPresent := canonical[requirement.OldName]
		return present && !oldPresent
	}
	return false
}

func ensureNewline(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

func renderSDDArchiveCheckHelp(stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "Uso: axiom sdd archive-check --cwd <dir> --change <cambio> [--json]")
	_, _ = fmt.Fprintln(stdout, "Comprueba, sin escribir nada, si un cambio ISO puede archivarse: declaración válida, al menos un escenario por requisito")
	_, _ = fmt.Fprintln(stdout, "y matriz de verificación válida y de éxito. Sale con error y los motivos tipados si hay hallazgos; un cambio no ISO sale sin error.")
	_, _ = fmt.Fprintln(stdout, "Informa por dominio alreadyComposed: la baseline ya contiene la delta entera (reanudación de un archivado interrumpido).")
	_, _ = fmt.Fprintln(stdout, "Informa además assignments (el mapa guardado en ids.yaml) y pendingProvisionals (provisionales de las deltas sin número todavía).")
	_, _ = fmt.Fprintln(stdout, "  --json        Salida JSON ("+archiveCheckSchema+").")
	return nil
}

// archiveCheckIdentifiers lee lo que el cambio dice de sus identificadores provisionales
// (REQ-47.6, REQ-47.8): el mapa guardado en `ids.yaml`, que reconoce una asignación ya hecha
// tras un archivado interrumpido, y los provisionales que sus deltas declaran en un ADDED y
// el mapa aún no asigna. Con el patrón inaccesible no decide nada (lo reporta la rama ISO); una
// traza ilegible es un error tipado.
func archiveCheckIdentifiers(loc workspace.Location, changeRoot string) (assignments []livespec.AssignmentReport, pending []string, err error) {
	if _, patternErr := loc.Declarations.IDPattern(); patternErr != nil {
		return nil, nil, nil
	}
	trace, err := livespec.ReadTraceFile(filepath.Join(changeRoot, "ids.yaml"))
	if err != nil {
		return nil, nil, err
	}
	assigned := map[string]bool{}
	for _, e := range trace.Assignments {
		assigned[e.Provisional] = true
		assignments = append(assignments, livespec.AssignmentReport{
			Provisional: e.Provisional, Definitive: e.Definitive, Capability: e.Capability,
			Requirement: e.Requirement, State: livespec.StateKept, Previous: e.Previous,
		})
	}
	entries, readErr := os.ReadDir(filepath.Join(changeRoot, "specs"))
	if readErr != nil {
		return assignments, nil, nil
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(changeRoot, "specs", entry.Name(), "spec.md"))
		if readErr != nil {
			continue
		}
		delta, _ := openspecdoc.ParseChangeSpec(string(data))
		for _, r := range delta.Added {
			if token, ok := reqid.ProvisionalInTitle(r.Name); ok && !assigned[token] && !seen[token] {
				seen[token] = true
				pending = append(pending, token)
			}
		}
	}
	return assignments, pending, nil
}
