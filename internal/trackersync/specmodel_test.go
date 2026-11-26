package trackersync

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/specsummary"
)

func testPattern(t *testing.T) reqid.Pattern {
	t.Helper()
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return p
}

func writeSpec(t *testing.T, root, capability, text string) {
	t.Helper()
	dir := filepath.Join(root, "specs", capability)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mixedDelta pone MODIFIED antes de ADDED y un REMOVED en medio, como pide D12.
const mixedDelta = `## MODIFIED Requirements

### Requirement: Cambiado (prov:ADO:2)
El sistema DEBE cambiar ` + "`algo`" + `.

#### Scenario: s
- **WHEN** x

## REMOVED Requirements

### Requirement: Retirado (prov:ADO:3)
Ya no aplica.

## ADDED Requirements

### Requirement: Nuevo (prov:ADO:1)
- lista sin párrafo
`

func TestSpecModelProvisionalKeepsFileOrderAndOperations(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "beta", mixedDelta)
	writeSpec(t, root, "alfa", "## ADDED Requirements\n\n### Requirement: Solo (prov:ADO:9)\nTexto uno\ntexto dos\n")

	got, err := SpecModel(root, SpecModelOptions{Pattern: testPattern(t)})
	if err != nil {
		t.Fatalf("SpecModel: %v", err)
	}
	if len(got.Model.Capabilities) != 2 || got.Model.Capabilities[0].Name != "alfa" || got.Model.Capabilities[1].Name != "beta" {
		t.Fatalf("capacidades inesperadas: %+v", got.Model.Capabilities)
	}
	beta := got.Model.Capabilities[1].Requirements
	want := []specsummary.Requirement{
		{Title: "Cambiado (prov:ADO:2)", Paragraph: "El sistema DEBE cambiar `algo`.", Operation: specsummary.OpModified},
		{Title: "Retirado (prov:ADO:3)", Paragraph: "Ya no aplica.", Operation: specsummary.OpRemoved},
		{Title: "Nuevo (prov:ADO:1)", Operation: specsummary.OpAdded},
	}
	if !reflect.DeepEqual(beta, want) {
		t.Fatalf("requisitos de beta = %+v\nwant %+v", beta, want)
	}
	if alfa := got.Model.Capabilities[0].Requirements[0]; alfa.Paragraph != "Texto uno texto dos" {
		t.Fatalf("párrafo de alfa = %q", alfa.Paragraph)
	}
	// Los REMOVED no llevan ID; el orden sigue al del resumen.
	wantIDs := []string{"prov:ADO:9", "prov:ADO:2", "prov:ADO:1"}
	if !reflect.DeepEqual(got.IDs, wantIDs) {
		t.Fatalf("IDs = %v, want %v", got.IDs, wantIDs)
	}
}

func TestSpecModelRenamedUsesNewNameAndKeepsOldOne(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "c", "## RENAMED Requirements\n\n### Requirement: Viejo (RF-ADO-01) → Nuevo (RF-ADO-01)\nTexto.\n")
	got, err := SpecModel(root, SpecModelOptions{Pattern: testPattern(t)})
	if err != nil {
		t.Fatalf("SpecModel: %v", err)
	}
	r := got.Model.Capabilities[0].Requirements[0]
	if r.Operation != specsummary.OpRenamed || r.Title != "Nuevo (RF-ADO-01)" || r.RenamedFrom != "Viejo (RF-ADO-01)" {
		t.Fatalf("RENAMED inesperado: %+v", r)
	}
	if !reflect.DeepEqual(got.IDs, []string{"RF-ADO-01"}) {
		t.Fatalf("IDs = %v", got.IDs)
	}
}

func TestSpecModelWithoutSpecsDirIsEmpty(t *testing.T) {
	got, err := SpecModel(t.TempDir(), SpecModelOptions{})
	if err != nil || len(got.Model.Capabilities) != 0 || len(got.IDs) != 0 {
		t.Fatalf("got %+v, err %v", got, err)
	}
}

const traceYAML = `schema: axiom.requirement-ids/v1
pattern: RF-<DOM>-<NN>
assignments:
  - provisional: prov:ADO:1
    definitive: RF-ADO-01
    capability: beta
    requirement: Nuevo
`

func TestSpecModelArchivedTranslatesResidualProvisionalWithIDsYAML(t *testing.T) {
	root := t.TempDir()
	// Una delta ya reescrita (definitivo) y otra con un provisional residual.
	writeSpec(t, root, "alfa", "## ADDED Requirements\n\n### Requirement: Hecho (RF-ADO-07)\nTexto.\n")
	writeSpec(t, root, "beta", "## ADDED Requirements\n\n### Requirement: Nuevo (prov:ADO:1)\nTexto.\n")
	if err := os.WriteFile(filepath.Join(root, "ids.yaml"), []byte(traceYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := SpecModel(root, SpecModelOptions{Pattern: testPattern(t), Archived: true})
	if err != nil {
		t.Fatalf("SpecModel: %v", err)
	}
	if title := got.Model.Capabilities[1].Requirements[0].Title; title != "Nuevo (RF-ADO-01)" {
		t.Fatalf("título traducido = %q", title)
	}
	if want := []string{"RF-ADO-07", "RF-ADO-01"}; !reflect.DeepEqual(got.IDs, want) {
		t.Fatalf("IDs = %v, want %v", got.IDs, want)
	}
}

func TestSpecModelArchivedResidualProvisionalWithoutEntryFails(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "beta", "## ADDED Requirements\n\n### Requirement: Nuevo (prov:ADO:5)\nTexto.\n")
	if err := os.WriteFile(filepath.Join(root, "ids.yaml"), []byte(traceYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := SpecModel(root, SpecModelOptions{Pattern: testPattern(t), Archived: true})
	var sme *SpecModelError
	if !errors.As(err, &sme) || sme.Code != CodeProvisionalIDsUnresolved {
		t.Fatalf("err = %v, want %s", err, CodeProvisionalIDsUnresolved)
	}
	if !strings.Contains(err.Error(), "prov:ADO:5") {
		t.Fatalf("el error debe nombrar el provisional: %v", err)
	}
}

func TestSpecModelArchivedWithoutIDsYAMLFailsOnResidualButNotOnDefinitive(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "alfa", "## ADDED Requirements\n\n### Requirement: Hecho (RF-ADO-07)\nTexto.\n")
	if _, err := SpecModel(root, SpecModelOptions{Pattern: testPattern(t), Archived: true}); err != nil {
		t.Fatalf("solo definitivos no necesitan ids.yaml: %v", err)
	}
	writeSpec(t, root, "beta", "## ADDED Requirements\n\n### Requirement: Nuevo (prov:ADO:1)\nTexto.\n")
	_, err := SpecModel(root, SpecModelOptions{Pattern: testPattern(t), Archived: true})
	var sme *SpecModelError
	if !errors.As(err, &sme) || sme.Code != CodeProvisionalIDsUnresolved {
		t.Fatalf("err = %v, want %s", err, CodeProvisionalIDsUnresolved)
	}
}

func TestSpecModelFeedsRendererEndToEnd(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "beta", mixedDelta)
	got, err := SpecModel(root, SpecModelOptions{Pattern: testPattern(t)})
	if err != nil {
		t.Fatal(err)
	}
	html := specsummary.Render(got.Model)
	i, j, k := strings.Index(html, "Cambiado"), strings.Index(html, "Retirado"), strings.Index(html, "Nuevo")
	if !(i >= 0 && i < j && j < k) {
		t.Fatalf("orden del fichero perdido:\n%s", html)
	}
}
