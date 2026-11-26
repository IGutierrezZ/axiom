package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// testsCheckSchema identifica la salida JSON de `axiom sdd tests check`.
const testsCheckSchema = "axiom.sdd-tests-check/v1"

// RunSDDTests es el subdespachador `axiom sdd tests <check>` (RF-KOF-21).
func RunSDDTests(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("tests requiere un subcomando: check; ejecuta `axiom sdd tests --help`")
	}
	if args[0] == "--help" || args[0] == "-h" {
		return renderSDDTestsHelp(stdout)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "check":
		return runSDDTestsCheck(rest, stdout)
	default:
		return fmt.Errorf("subcomando %q no reconocido para tests; opciones: check; ejecuta `axiom sdd tests --help`", sub)
	}
}

func renderSDDTestsHelp(stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "Uso: axiom sdd tests check --cwd <dir> --change <cambio> [--json]")
	_, _ = fmt.Fprintln(stdout, "Comprueba, sin escribir nada, la batería de pruebas de aceptación (pruebas.md) de un cambio contra sus specs delta.")
	_, _ = fmt.Fprintln(stdout, "Sale con error si hay algún hallazgo, o si la función está activa y falta pruebas.md después de tasks (pruebas_ausente).")
	_, _ = fmt.Fprintln(stdout, "Con la función no activa y sin pruebas.md sale con éxito indicando que no está activada.")
	_, _ = fmt.Fprintln(stdout, "  --json        Salida JSON ("+testsCheckSchema+").")
	return nil
}

// testsCheckReport es la salida de `axiom sdd tests check`, en texto y en JSON. Las claves
// `schema`, `change`, `file` y `error` son de identificación y las admite cualquier consumidor.
type testsCheckReport struct {
	Schema       string                   `json:"schema"`
	Change       string                   `json:"change"`
	Active       bool                     `json:"active"`
	Environments []string                 `json:"environments"`
	Present      bool                     `json:"present"`
	Cases        int                      `json:"cases"`
	Scenarios    acceptancetests.Coverage `json:"scenarios"`
	Requirements acceptancetests.Coverage `json:"requirements"`
	Findings     []testsCheckFinding      `json:"findings"`
	Error        string                   `json:"error,omitempty"`
	// Notice es el mensaje informativo de los casos sin batería; no es un error.
	Notice string `json:"notice,omitempty"`
}

type testsCheckFinding struct {
	Code    string `json:"code"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func runSDDTestsCheck(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderSDDTestsHelp(stdout)
	}
	flags := flag.NewFlagSet("sdd-tests-check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", "", "Directorio desde el que resolver el workspaceRoot")
	change := flags.String("change", "", "Cambio activo cuya batería de pruebas se comprueba")
	asJSON := flags.Bool("json", false, "Salida JSON (esquema "+testsCheckSchema+")")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("sdd tests check: %w; ejecuta `axiom sdd tests --help`", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("sdd tests check: argumento inesperado %q; ejecuta `axiom sdd tests check --cwd <dir> --change <cambio> [--json]`", flags.Arg(0))
	}
	if strings.TrimSpace(*change) == "" {
		return errors.New("sdd tests check requiere --change; ejecuta `axiom sdd tests check --cwd <dir> --change <cambio> [--json]`")
	}
	if err := gitref.ValidBranchComponent(*change); err != nil || *change == "archive" {
		reason := "es un nombre reservado"
		if err != nil {
			reason = err.Error()
		}
		return fmt.Errorf("sdd tests check: --change %q no es un nombre de cambio válido: %s; usa el nombre de la carpeta del cambio que proyecta `axiom sdd status --cwd <dir> --json`", *change, reason)
	}
	start := *cwd
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, changeRoot, err := resolveGovernanceChangeRoot(start, *change)
	if err != nil {
		return fmt.Errorf("sdd tests check: %w", err)
	}
	paths, err := sddstatus.ChangeArtifactPaths(changeRoot)
	if err != nil {
		return fmt.Errorf("sdd tests check: no se pueden leer los artefactos del cambio %q: %w; comprueba su carpeta y vuelve a ejecutar `axiom sdd tests check --cwd <dir> --change %s`", *change, err, *change)
	}
	if len(paths.Specs) == 0 {
		return fmt.Errorf("sdd tests check: el cambio %q no tiene specs delta contra las que cruzar %s; crea su spec con la fase sdd-spec y vuelve a ejecutar `axiom sdd tests check --cwd <dir> --change %s`", *change, acceptancetests.FileName, *change)
	}

	result := sddstatus.AcceptanceTestsFor(loc, changeRoot)
	report := testsCheckReport{
		Schema: testsCheckSchema, Change: *change, Active: result.Active,
		Environments: append([]string{}, result.Environments...), Present: result.Present,
		Findings: []testsCheckFinding{},
	}
	if result.Present && result.Error == "" {
		report.Cases = result.Report.Cases
		report.Scenarios = result.Report.Scenarios
		report.Requirements = result.Report.Requirements
	}
	for _, finding := range result.Findings {
		report.Findings = append(report.Findings, testsCheckFinding{
			Code: string(finding.Code), File: finding.File, Line: finding.Line, Message: finding.Message,
		})
	}
	failure := ""
	switch {
	case result.DeclarationError != "":
		failure = result.DeclarationError
	case result.Error != "":
		failure = result.Error
	}
	report.Error = failure
	switch {
	case failure != "":
	case !result.Present && !result.Missing && result.Active:
		report.Notice = "todavía no hay batería: " + acceptancetests.FileName + " se genera en la fase sdd-tasks"
	case !result.Present && !result.Missing:
		report.Notice = "las pruebas de aceptación no están activadas"
	}

	if err := renderTestsCheck(stdout, *asJSON, report); err != nil {
		return err
	}
	switch {
	case failure != "":
		return fmt.Errorf("sdd tests check: %s; corrígelo y vuelve a ejecutar `axiom sdd tests check --cwd <dir> --change %s`", failure, *change)
	case result.Missing || (result.Present && !result.Report.OK()):
		return fmt.Errorf("sdd tests check: la batería del cambio %q tiene %d hallazgo(s); corrígelos en %s y vuelve a ejecutar `axiom sdd tests check --cwd <dir> --change %s`", *change, len(report.Findings), acceptancetests.FileName, *change)
	}
	return nil
}

func renderTestsCheck(stdout io.Writer, asJSON bool, report testsCheckReport) error {
	if asJSON {
		return encodeJSON(stdout, report)
	}
	var b strings.Builder
	if report.Notice != "" {
		fmt.Fprintf(&b, "Cambio %q: %s.\n", report.Change, report.Notice)
	}
	if report.Present {
		fmt.Fprintf(&b, "Escenarios: %d/%d\n", report.Scenarios.Covered, report.Scenarios.Total)
		fmt.Fprintf(&b, "Requisitos: %d/%d\n", report.Requirements.Covered, report.Requirements.Total)
		fmt.Fprintf(&b, "Casos: %d\n", report.Cases)
	}
	for _, finding := range report.Findings {
		fmt.Fprintf(&b, "  - %s (línea %d): %s\n", finding.Code, finding.Line, finding.Message)
	}
	if report.Error != "" {
		fmt.Fprintf(&b, "Error: %s\n", report.Error)
	}
	if report.Present && report.Error == "" && len(report.Findings) == 0 {
		fmt.Fprintf(&b, "La batería del cambio %q es válida.\n", report.Change)
	}
	_, err := io.WriteString(stdout, b.String())
	return err
}
