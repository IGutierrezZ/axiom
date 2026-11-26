package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

// Un axiom.yaml con la clave retirada `governance.semantic_analysis` sigue
// cargando y validando; la validación solo avisa de que se ignora (C-08-l).
func TestValidateWarnsAboutObsoleteSemanticAnalysis(t *testing.T) {
	const base = "workspace:\n  name: \"Demo\"\n  topology: \"monorepo-embedded\"\n  specs_repository: \".\"\nroles:\n  fullstack:\n    name: \"Fullstack\"\n    repositories:\n      - path: \".\"\ngovernance:\n  language: \"es\"\n"
	baseDir := filepath.Clean("/workspace")

	for name, tc := range map[string]struct {
		yaml string
		warn bool
	}{
		"con la clave": {yaml: base + "  semantic_analysis: \"codegraph\"\n", warn: true},
		"sin la clave": {yaml: base, warn: false},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("la configuración debe cargar: %v", err)
			}
			fs := newMockFS()
			fs.addDir(baseDir)
			report, err := Validate(fs, cfg, baseDir)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if !report.Valid {
				t.Fatalf("la clave obsoleta no debe invalidar: %v", report.Errors)
			}
			got := strings.Contains(strings.Join(report.Warnings, "\n"), "governance.semantic_analysis")
			if got != tc.warn {
				t.Errorf("aviso de semantic_analysis = %v, se esperaba %v (avisos: %q)", got, tc.warn, report.Warnings)
			}
		})
	}
}
