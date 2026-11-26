package changestart

import (
	"os"
	"path/filepath"
	"testing"
)

func writeWorkItemDoc(t *testing.T, dir, id, profile string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `{"schema":"` + ContextSchema + `","change":"x","plugin":"azure-devops","id":"` + id + `","profile":"` + profile + `","attachments":{"downloaded":0,"omitted":0}}`
	if err := os.WriteFile(filepath.Join(dir, "work-item.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

// WorkItemFacts lee primero el staging, después el documento adoptado y, sin ninguno, no devuelve nada.
func TestWorkItemFacts(t *testing.T) {
	common := t.TempDir()
	adopted := filepath.Join(t.TempDir(), "context")

	if _, ok := WorkItemFacts(common, "x", adopted); ok {
		t.Fatal("sin staging ni documento adoptado no hay work item")
	}

	writeWorkItemDoc(t, adopted, "300", "")
	got, ok := WorkItemFacts(common, "x", adopted)
	if !ok || got.ID != "300" || got.Plugin != "azure-devops" || got.Profile != "" {
		t.Errorf("documento adoptado = %+v, %v", got, ok)
	}

	writeWorkItemDoc(t, filepath.Join(common, "axiom", "change-start", "v1", "x", "context"), "2525", "bug")
	got, ok = WorkItemFacts(common, "x", adopted)
	if !ok || got.ID != "2525" || got.Profile != "bug" {
		t.Errorf("el staging manda sobre el adoptado: %+v, %v", got, ok)
	}

	if _, ok := WorkItemFacts(common, "otro", ""); ok {
		t.Error("el staging de otro cambio no cuenta")
	}
}
