package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/skillcheck"
)

// skillCheckReport es el contrato estable de `axiom skill check --json`.
type skillCheckReport struct {
	Skills      []skillcheck.Report  `json:"skills"`
	SetFindings []skillcheck.Finding `json:"set_findings"`
	Errors      int                  `json:"errors"`
	Warnings    int                  `json:"warnings"`
}

// runSkillCheck implementa `axiom skill check [--json] [--path <dir>]`: pasa
// skillcheck por las skills de proyecto y usuario que el registro escanea e
// informa de lo que el registro tolera en silencio. Sale con 1 si hay algún
// error; los avisos y los hallazgos informativos no cambian el código.
func runSkillCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("skill check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pathFlag := fs.String("path", ".", "Ruta a la raíz del proyecto")
	jsonFlag := fs.Bool("json", false, "Informe en JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado %q. Uso: axiom skill check [--json] [--path <directorio>]\n", fs.Arg(0))
		return 1
	}

	root, err := filepath.Abs(*pathFlag)
	if err != nil {
		fmt.Fprintf(stderr, "Error resolviendo ruta: %v\n", err)
		return 1
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		fmt.Fprintf(stderr, "Error: no se pudo resolver la carpeta de usuario: %v\n", err)
		return 1
	}

	set := skillcheck.ScanSet(root, home, skillcheck.SetOptions{FolderRuntimes: cli.SkillFolderRuntimes(home, root)})
	report := skillCheckReport{
		Skills:      set.Skills,
		SetFindings: set.SetFindings,
		Errors:      set.Count(skillcheck.SeverityError),
		Warnings:    set.Count(skillcheck.SeverityWarning),
	}
	if report.Skills == nil {
		report.Skills = []skillcheck.Report{}
	}
	if report.SetFindings == nil {
		report.SetFindings = []skillcheck.Finding{}
	}
	for i := range report.Skills {
		if report.Skills[i].Findings == nil {
			report.Skills[i].Findings = []skillcheck.Finding{}
		}
	}

	if *jsonFlag {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(stderr, "Error escribiendo JSON: %v\n", err)
			return 1
		}
	} else {
		writeSkillCheckText(stdout, root, report)
	}
	if report.Errors > 0 {
		return 1
	}
	return 0
}

func writeSkillCheckText(w io.Writer, root string, r skillCheckReport) {
	fmt.Fprintf(w, "axiom skill check — %s\n\n", root)
	for _, s := range r.Skills {
		fmt.Fprintf(w, "%s\n  nombre: %s · perfil: %s\n", s.Path, s.Name, s.Profile)
		if len(s.Findings) == 0 {
			fmt.Fprintln(w, "  sin hallazgos")
		}
		for _, f := range s.Findings {
			fmt.Fprintf(w, "  %-7s %s: %s\n", f.Severity, f.Rule, f.Message)
		}
	}
	if len(r.SetFindings) > 0 {
		fmt.Fprintln(w, "\nConjunto de skills")
		for _, f := range r.SetFindings {
			fmt.Fprintf(w, "  %-7s %s: %s\n", f.Severity, f.Rule, f.Message)
		}
	}
	fmt.Fprintf(w, "\nResumen: %d skills, %d error(es), %d aviso(s)\n", len(r.Skills), r.Errors, r.Warnings)
}
