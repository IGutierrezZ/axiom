package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
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
		return fmt.Errorf("sdd archive-check: --change %q no es un nombre de cambio válido: %s", *change, reason)
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
		delta, baseline, readErr := sddstatus.ISOChangeRequirements(loc.PlanningHome, changeRoot)
		if readErr != nil {
			report.Error = fmt.Sprintf("no se pueden leer los artefactos del cambio: %v; comprueba la carpeta del cambio y vuelve a ejecutar `axiom sdd archive-check --cwd <dir> --change %s`", readErr, *change)
			break
		}
		report.Domains = alreadyComposed(delta, baseline)
	}
	report.OK = report.Error == "" && len(report.Findings) == 0

	if err := renderArchiveCheck(stdout, *asJSON, report); err != nil {
		return err
	}
	switch {
	case report.Error != "":
		return fmt.Errorf("sdd archive-check: %s", report.Error)
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
	Domains  []archiveCheckDomain  `json:"domains"`
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
func alreadyComposed(delta, baseline []isoevidence.Requirement) []archiveCheckDomain {
	byDomain := map[string]map[string]string{}
	for _, requirement := range baseline {
		if byDomain[requirement.Domain] == nil {
			byDomain[requirement.Domain] = map[string]string{}
		}
		byDomain[requirement.Domain][requirement.Name] = requirement.Text
	}
	composed := map[string]bool{}
	for _, requirement := range delta {
		present, ok := composed[requirement.Domain]
		if !ok {
			present = true
		}
		composed[requirement.Domain] = present && requirementComposed(requirement, byDomain[requirement.Domain])
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

func requirementComposed(requirement isoevidence.Requirement, canonical map[string]string) bool {
	text, present := canonical[requirement.Name]
	switch requirement.Section {
	case isoevidence.SectionAdded, isoevidence.SectionModified:
		return present && text == ensureNewline(requirement.Text)
	case isoevidence.SectionRemoved:
		// Borrado o superado: en ambos casos el nombre original deja de figurar
		// (con --supersede el compositor lo renombra), y volver a componer fallaría
		// con «no canonical requirement named».
		return !present
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
	_, _ = fmt.Fprintln(stdout, "  --json        Salida JSON ("+archiveCheckSchema+").")
	return nil
}
