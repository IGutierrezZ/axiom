package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #4119: builds a minimal openspec tree (canonical spec with two
// requirements, a change delta with an ADDED and a MODIFIED requirement) and
// runs the archive merge through the CLI surface the sdd-archive skill
// calls. Before ComposeOpenSpecCanonicalSpec existed, composition was
// model-driven prose with no Go command to call, so this exact shape
// (dropped requirement / unapplied delta) could not be exercised or gated.
func TestRunSDDArchiveComposeMinimalOpenSpecTree(t *testing.T) {
	root := t.TempDir()
	canonicalPath := filepath.Join(root, "openspec", "specs", "widgets", "spec.md")
	deltaPath := filepath.Join(root, "openspec", "changes", "add-widget-tagging", "specs", "widgets", "spec.md")

	writeCLIFixture(t, canonicalPath, "## Requirements\n\n"+
		"### Requirement: Unrelated Listing\n\nThe system MUST list widgets.\n\n"+
		"### Requirement: Widget Expiration\n\nThe system MUST expire widgets after 30 days.\n")
	writeCLIFixture(t, deltaPath, "## ADDED Requirements\n\n"+
		"### Requirement: Widget Tagging\n\nThe system MUST support tagging widgets.\n\n"+
		"## MODIFIED Requirements\n\n### Requirement: Widget Expiration\n\nThe system MUST expire widgets after 90 days.\n")

	var output bytes.Buffer
	if err := runSDDArchiveCompose([]string{"--canonical", canonicalPath, "--delta", deltaPath}, &output); err != nil {
		t.Fatalf("runSDDArchiveCompose() error = %v", err)
	}

	composed := output.String()
	if !strings.Contains(composed, "### Requirement: Unrelated Listing") {
		t.Fatalf("composed spec dropped the unrelated requirement:\n%s", composed)
	}
	if !strings.Contains(composed, "expire widgets after 90 days") || strings.Contains(composed, "expire widgets after 30 days") {
		t.Fatalf("composed spec did not apply the MODIFIED delta:\n%s", composed)
	}
	if !strings.Contains(composed, "### Requirement: Widget Tagging") {
		t.Fatalf("composed spec did not apply the ADDED delta:\n%s", composed)
	}
}

func TestRunSDDArchiveComposeRefusesUnappliedDeltaWithoutWriting(t *testing.T) {
	root := t.TempDir()
	canonicalPath, deltaPath, outputPath := filepath.Join(root, "spec.md"), filepath.Join(root, "delta.md"), filepath.Join(root, "out.md")

	writeCLIFixture(t, canonicalPath, "## Requirements\n\n### Requirement: Existing\n\nBody.\n")
	writeCLIFixture(t, deltaPath, "## MODIFIED Requirements\n\n### Requirement: Missing\n\nNew body.\n")

	err := runSDDArchiveCompose([]string{"--canonical", canonicalPath, "--delta", deltaPath, "--output", outputPath}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "MODIFIED") || !strings.Contains(err.Error(), "Missing") {
		t.Fatalf("error = %v, want it to name the MODIFIED delta for %q", err, "Missing")
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("output file was written despite refusal: statErr = %v", statErr)
	}
}

func writeCLIFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestRunSDDArchiveComposeSupersedeFlag(t *testing.T) {
	root := t.TempDir()
	canonicalPath := filepath.Join(root, "openspec", "specs", "widgets", "spec.md")
	deltaPath := filepath.Join(root, "openspec", "changes", "prune-widgets", "specs", "widgets", "spec.md")

	writeCLIFixture(t, canonicalPath, "## Requirements\n\n"+
		"### Requirement: Old Feature\n\nOld body.\n\n"+
		"### Requirement: Active Feature\n\nActive body.\n")
	writeCLIFixture(t, deltaPath, "## REMOVED Requirements\n\n"+
		"### Requirement: Old Feature\n\n(Reason: Reemplazado por nueva arquitectura)\n")

	var output bytes.Buffer
	err := runSDDArchiveCompose([]string{
		"--canonical", canonicalPath,
		"--delta", deltaPath,
		"--supersede",
		"--superseded-requirements", "Active Feature",
	}, &output)
	if err != nil {
		t.Fatalf("runSDDArchiveCompose() error = %v", err)
	}

	composed := output.String()
	if !strings.Contains(composed, "### Requirement: Old Feature [SUPERSEDED / DEPRECADO]") {
		t.Errorf("se esperaba Old Feature marcado como SUPERSEDED:\n%s", composed)
	}
	if !strings.Contains(composed, "### Requirement: Active Feature [SUPERSEDED / DEPRECADO]") {
		t.Errorf("se esperaba Active Feature marcado como SUPERSEDED:\n%s", composed)
	}
}

// REQ-47.3, REQ-47.8: un identificador inexistente nombra la capacidad por la carpeta de la
// delta, y un provisional lleva la continuación al verbo que asigna; sin escribir en ambos.
func TestRunSDDArchiveComposeTypedIDErrors(t *testing.T) {
	cases := []struct {
		name, delta string
		wantParts   []string
	}{
		{"identificador inexistente", "## MODIFIED Requirements\n\n### Requirement: Fantasma (REQ-9.9)\n\nNew body.\n",
			[]string{"capacidad pagos", "REQ-9.9", "MODIFIED"}},
		{"provisional", "## ADDED Requirements\n\n### Requirement: Nuevo (prov:1:1)\n\nBody.\n",
			[]string{"prov:1:1", "axiom sdd archive-integrate", "--check"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			canonicalPath := filepath.Join(root, "openspec", "specs", "pagos", "spec.md")
			deltaPath := filepath.Join(root, "openspec", "changes", "c", "specs", "pagos", "spec.md")
			outputPath := filepath.Join(root, "out.md")
			writeCLIFixture(t, canonicalPath, "## Requirements\n\n### Requirement: Existing (REQ-1.1)\n\nBody.\n")
			writeCLIFixture(t, deltaPath, tc.delta)

			err := runSDDArchiveCompose([]string{"--canonical", canonicalPath, "--delta", deltaPath, "--output", outputPath}, &bytes.Buffer{})
			if err == nil {
				t.Fatal("se esperaba un rechazo")
			}
			for _, part := range tc.wantParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("el error no contiene %q: %v", part, err)
				}
			}
			if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
				t.Errorf("se escribió la salida pese al rechazo: %v", statErr)
			}
		})
	}
}

// REQ-30.23: un nombre superado inexistente sigue ignorándose en archive-compose.
func TestRunSDDArchiveComposeStillIgnoresUnknownSupersededName(t *testing.T) {
	root := t.TempDir()
	canonicalPath, deltaPath := filepath.Join(root, "spec.md"), filepath.Join(root, "delta.md")
	writeCLIFixture(t, canonicalPath, "## Requirements\n\n### Requirement: Existing (REQ-1.1)\n\nBody.\n")
	writeCLIFixture(t, deltaPath, "## ADDED Requirements\n\n### Requirement: Nuevo (REQ-1.2)\n\nBody.\n")
	var out bytes.Buffer
	if err := runSDDArchiveCompose([]string{"--canonical", canonicalPath, "--delta", deltaPath, "--superseded-requirements", "No existe"}, &out); err != nil {
		t.Fatalf("un nombre superado inexistente se ignora: %v", err)
	}
}
