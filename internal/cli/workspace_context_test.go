package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const workspaceContextBase = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"

func workspaceContextDir(t *testing.T, blocks string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte(workspaceContextBase+blocks), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runWorkspaceContext(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := RunWorkspaceContext(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestWorkspaceContextJSONProjectsAbsolutePaths(t *testing.T) {
	dir := workspaceContextDir(t, "technical_context:\n  maintained:\n    entry: context/ENTRY.md\n    paths: [context/]\n  reference:\n    paths: [00_Glosario.md]\n")
	code, stdout, stderr := runWorkspaceContext(t, "--cwd", dir, "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	var got struct {
		SchemaName       string `json:"schemaName"`
		SchemaVersion    int    `json:"schemaVersion"`
		WorkspaceRoot    string `json:"workspaceRoot"`
		TechnicalContext *struct {
			Maintained *struct {
				Entry string   `json:"entry"`
				Paths []string `json:"paths"`
			} `json:"maintained"`
			Reference *struct {
				Paths []string `json:"paths"`
			} `json:"reference"`
		} `json:"technicalContext"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, stdout)
	}
	if got.SchemaName != "axiom.workspace-context" || got.SchemaVersion != 1 || !filepath.IsAbs(got.WorkspaceRoot) {
		t.Fatalf("cabecera inesperada: %+v", got)
	}
	if got.TechnicalContext == nil || got.TechnicalContext.Maintained == nil || got.TechnicalContext.Reference == nil {
		t.Fatalf("faltan modos: %s", stdout)
	}
	if entry := got.TechnicalContext.Maintained.Entry; !filepath.IsAbs(entry) || filepath.Base(entry) != "ENTRY.md" {
		t.Errorf("entry debe ser absoluta: %q", entry)
	}
	for _, p := range append(got.TechnicalContext.Maintained.Paths, got.TechnicalContext.Reference.Paths...) {
		if !filepath.IsAbs(p) {
			t.Errorf("ruta no absoluta: %q", p)
		}
	}
}

func TestWorkspaceContextWithoutDeclarationIsNullAndSucceeds(t *testing.T) {
	dir := workspaceContextDir(t, "")
	code, stdout, _ := runWorkspaceContext(t, "--cwd", dir, "--json")
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, `"technicalContext": null`) || strings.Contains(stdout, `"invalid"`) {
		t.Errorf("JSON inesperado: %s", stdout)
	}
	code, stdout, _ = runWorkspaceContext(t, "--cwd", dir)
	if code != 0 || !strings.Contains(stdout, "Sin contexto técnico declarado") {
		t.Errorf("texto: code=%d %q", code, stdout)
	}
}

func TestWorkspaceContextInvalidDeclarationFailsClosed(t *testing.T) {
	dir := workspaceContextDir(t, "technical_context:\n  maintained:\n    paths: []\n")
	code, stdout, _ := runWorkspaceContext(t, "--cwd", dir, "--json")
	if code == 0 {
		t.Fatal("una declaración inválida debe salir con código distinto de cero")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}
	if v, present := got["technicalContext"]; !present || v != nil {
		t.Errorf("technicalContext debe ser null: %v", got)
	}
	invalid, _ := got["invalid"].(string)
	if !strings.Contains(invalid, "paths") || !strings.Contains(invalid, "axiom workspace validate") {
		t.Errorf("invalid debe dar el motivo y la indicación: %q", invalid)
	}

	code, _, stderr := runWorkspaceContext(t, "--cwd", dir)
	if code == 0 || !strings.Contains(stderr, "axiom workspace validate") {
		t.Errorf("texto: code=%d stderr=%q", code, stderr)
	}
}

func TestWorkspaceContextTextListsEntryAndPaths(t *testing.T) {
	dir := workspaceContextDir(t, "technical_context:\n  maintained:\n    entry: context/ENTRY.md\n    paths: [context/]\n")
	code, stdout, _ := runWorkspaceContext(t, "--cwd", dir)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	for _, want := range []string{"Contexto técnico mantenido:", "entry: ", "ENTRY.md", "ruta: "} {
		if !strings.Contains(stdout, want) {
			t.Errorf("falta %q en:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "referencia") {
		t.Errorf("un modo no declarado no aparece:\n%s", stdout)
	}
}

func TestWorkspaceContextRejectsUnexpectedArguments(t *testing.T) {
	if code, _, stderr := runWorkspaceContext(t, "sobra"); code == 0 || !strings.Contains(stderr, "inesperado") {
		t.Errorf("code=%d stderr=%q", code, stderr)
	}
}
