package integrations

import (
	"strings"
	"testing"
	"time"
)

const validLinksYAML = `schema: axiom.integrations/v1
change: x
links:
  - plugin: fake
    kind: work_item
    id: "12345"
    url: https://tracker.example/items/12345
    source: gate.spec.approved/x/d0123456789abcdef
    recorded_at: 2026-11-18T22:30:00Z
time_entries:
  - plugin: fake
    role: fullstack
    item_id: "12346"
    minutes: 90
    date: 2026-11-18
    source: role.apply.approved/x/fullstack/d0123456789abcdef
    recorded_at: 2026-11-18T22:30:00Z
`

func TestReadLinksAbsentFileIsEmpty(t *testing.T) {
	f := newWorkspace(t)
	doc, err := readLinks(f.Folder())
	if err != nil {
		t.Fatalf("readLinks sin fichero: %v", err)
	}
	if doc.Schema != LinksSchemaV1 || doc.Change != "x" || len(doc.Links) != 0 || len(doc.TimeEntries) != 0 {
		t.Fatalf("doc = %+v", doc)
	}
}

func TestReadLinksParsesTheExactSchema(t *testing.T) {
	f := newWorkspace(t)
	writeTestFile(t, f.Folder().path(LinksFile), validLinksYAML)
	doc, err := readLinks(f.Folder())
	if err != nil {
		t.Fatalf("readLinks: %v", err)
	}
	if len(doc.Links) != 1 || doc.Links[0].ID != "12345" || doc.Links[0].Kind != "work_item" || doc.Links[0].URL != "https://tracker.example/items/12345" {
		t.Fatalf("links = %+v", doc.Links)
	}
	if len(doc.TimeEntries) != 1 || doc.TimeEntries[0].Minutes != 90 || doc.TimeEntries[0].Date != "2026-11-18" {
		t.Fatalf("time_entries = %+v", doc.TimeEntries)
	}
}

func TestReadLinksIsStrict(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		wants string
	}{
		{"esquema distinto", strings.Replace(validLinksYAML, "axiom.integrations/v1", "axiom.integrations/v2", 1), "esquema"},
		{"sin esquema", strings.Replace(validLinksYAML, "schema: axiom.integrations/v1\n", "", 1), "esquema"},
		{"clave desconocida en la raíz", validLinksYAML + "attempts: 3\n", "attempts"},
		{"clave desconocida en un vínculo", strings.Replace(validLinksYAML, "    kind: work_item\n", "    kind: work_item\n    last_error: boom\n", 1), "last_error"},
		{"fichero vacío", "", "vacío"},
		{"YAML roto", "schema: [sin cerrar\n", "integrations.yaml"},
		{"kind inválido", strings.Replace(validLinksYAML, "kind: work_item", "kind: Work-Item", 1), "kind"},
		{"id vacío", strings.Replace(validLinksYAML, `id: "12345"`, `id: ""`, 1), "id"},
		{"url con userinfo", strings.Replace(validLinksYAML, "https://tracker.example", "https://usuario:clave@tracker.example", 1), "userinfo"},
		{"url con esquema distinto", strings.Replace(validLinksYAML, "https://tracker.example", "ftp://tracker.example", 1), "http"},
		{"minutos no positivos", strings.Replace(validLinksYAML, "minutes: 90", "minutes: 0", 1), "minutes"},
		{"fecha inválida", strings.Replace(validLinksYAML, "date: 2026-11-18", "date: ayer", 1), "date"},
		{"cambio distinto del de la carpeta", strings.Replace(validLinksYAML, "change: x", "change: otro", 1), "change"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newWorkspace(t)
			writeTestFile(t, f.Folder().path(LinksFile), tt.yaml)
			_, err := readLinks(f.Folder())
			if err == nil || !strings.Contains(err.Error(), tt.wants) {
				t.Fatalf("se esperaba un error con %q, obtuvo %v", tt.wants, err)
			}
		})
	}
}

func sampleOutcome() Outcome {
	return Outcome{
		Links: []Link{
			{Kind: "work_item", ID: "1", URL: "https://tracker.example/1"},
			{Kind: "work_item", ID: "2", Role: "backend"},
		},
		TimeEntries: []TimeEntry{{Role: "backend", ItemID: "1", Minutes: 30, Date: "2026-11-19"}},
	}
}

func TestMergeLinksWritesAndIsIdempotentByteForByte(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	key := "gate.spec.approved/x/dabc"

	changed, err := mergeLinks(folder, "fake", key, fixedNow, sampleOutcome())
	if err != nil || !changed {
		t.Fatalf("primera fusión: changed=%v err=%v", changed, err)
	}
	first := readTestFile(t, folder.path(LinksFile))

	changed, err = mergeLinks(folder, "fake", key, fixedNow.Add(time.Hour), sampleOutcome())
	if err != nil || changed {
		t.Fatalf("repetir la fusión no cambia nada: changed=%v err=%v", changed, err)
	}
	if second := readTestFile(t, folder.path(LinksFile)); second != first {
		t.Fatalf("repetir la fusión debe dejar el fichero byte a byte igual:\n%s\n---\n%s", first, second)
	}

	doc, err := readLinks(folder)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Links) != 2 || doc.Links[0].Plugin != "fake" || doc.Links[0].Source != key || !doc.Links[0].RecordedAt.Equal(fixedNow) {
		t.Fatalf("los vínculos reciben plugin, origen y fecha: %+v", doc.Links)
	}
	if len(doc.TimeEntries) != 1 || doc.TimeEntries[0].Plugin != "fake" || doc.TimeEntries[0].Source != key {
		t.Fatalf("las imputaciones reciben plugin y origen: %+v", doc.TimeEntries)
	}
}

func TestMergeLinksIdentity(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	_, _ = mergeLinks(folder, "fake", "k1", fixedNow, Outcome{Links: []Link{{Kind: "work_item", ID: "1"}}})

	t.Run("mismo (plugin, kind, id, role) no se duplica aunque cambie el origen", func(t *testing.T) {
		changed, err := mergeLinks(folder, "fake", "k2", fixedNow, Outcome{Links: []Link{{Kind: "work_item", ID: "1", URL: "https://t.example/1"}}})
		if err != nil || changed {
			t.Fatalf("changed=%v err=%v", changed, err)
		}
		doc, _ := readLinks(folder)
		if len(doc.Links) != 1 || doc.Links[0].Source != "k1" || doc.Links[0].URL != "" {
			t.Fatalf("el vínculo existente se conserva intacto: %+v", doc.Links)
		}
	})
	t.Run("otro role o otro plugin o otro kind es otro vínculo", func(t *testing.T) {
		out := Outcome{Links: []Link{
			{Kind: "work_item", ID: "1", Role: "backend"},
			{Kind: "task", ID: "1"},
		}}
		if changed, err := mergeLinks(folder, "fake", "k3", fixedNow, out); err != nil || !changed {
			t.Fatalf("changed=%v err=%v", changed, err)
		}
		doc, _ := readLinks(folder)
		if len(doc.Links) != 3 {
			t.Fatalf("links = %+v", doc.Links)
		}
	})
	t.Run("imputación por (plugin, source, item_id)", func(t *testing.T) {
		entry := TimeEntry{ItemID: "9", Minutes: 10, Date: "2026-11-19"}
		if changed, _ := mergeLinks(folder, "fake", "k4", fixedNow, Outcome{TimeEntries: []TimeEntry{entry}}); !changed {
			t.Fatal("la primera imputación se añade")
		}
		if changed, _ := mergeLinks(folder, "fake", "k4", fixedNow, Outcome{TimeEntries: []TimeEntry{entry}}); changed {
			t.Fatal("la misma imputación no se repite")
		}
		if changed, _ := mergeLinks(folder, "fake", "k5", fixedNow, Outcome{TimeEntries: []TimeEntry{entry}}); !changed {
			t.Fatal("otro origen es otra imputación")
		}
	})
}

func TestMergeLinksEmptyOutcomeCreatesNoFile(t *testing.T) {
	f := newWorkspace(t)
	changed, err := mergeLinks(f.Folder(), "fake", "k", fixedNow, Outcome{})
	if err != nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if fileExists(f.Folder().path(LinksFile)) {
		t.Fatal("un Outcome vacío no crea integrations.yaml")
	}
}

func TestMergeLinksRejectsUserinfoAsAPermanentError(t *testing.T) {
	f := newWorkspace(t)
	out := Outcome{Links: []Link{{Kind: "work_item", ID: "1", URL: "https://usuario:clave@tracker.example/1"}}}
	_, err := mergeLinks(f.Folder(), "fake", "k", fixedNow, out)
	if err == nil || !IsPermanent(err) {
		t.Fatalf("una URL con userinfo es un error permanente: %v", err)
	}
	if strings.Contains(err.Error(), "clave@") || strings.Contains(err.Error(), "usuario:clave") {
		t.Fatalf("el error no debe repetir las credenciales: %v", err)
	}
	if fileExists(f.Folder().path(LinksFile)) {
		t.Fatal("nada se escribe si un vínculo es inválido")
	}
}

func TestMergeLinksRejectsInvalidOutcomesAsPermanent(t *testing.T) {
	tests := map[string]Outcome{
		"kind inválido":           {Links: []Link{{Kind: "Work Item", ID: "1"}}},
		"id vacío":                {Links: []Link{{Kind: "work_item", ID: ""}}},
		"id con control":          {Links: []Link{{Kind: "work_item", ID: "a\nb"}}},
		"id demasiado largo":      {Links: []Link{{Kind: "work_item", ID: strings.Repeat("a", 129)}}},
		"url no http":             {Links: []Link{{Kind: "work_item", ID: "1", URL: "javascript:alert(1)"}}},
		"plugin ajeno":            {Links: []Link{{Plugin: "otro", Kind: "work_item", ID: "1"}}},
		"imputación sin minutos":  {TimeEntries: []TimeEntry{{ItemID: "1", Minutes: 0, Date: "2026-11-19"}}},
		"imputación sin fecha":    {TimeEntries: []TimeEntry{{ItemID: "1", Minutes: 5}}},
		"imputación sin elemento": {TimeEntries: []TimeEntry{{Minutes: 5, Date: "2026-11-19"}}},
	}
	for name, out := range tests {
		t.Run(name, func(t *testing.T) {
			f := newWorkspace(t)
			_, err := mergeLinks(f.Folder(), "fake", "k", fixedNow, out)
			if err == nil || !IsPermanent(err) {
				t.Fatalf("un resultado inválido del plugin es un error permanente: %v", err)
			}
			if fileExists(f.Folder().path(LinksFile)) {
				t.Fatal("nada se escribe")
			}
		})
	}
}

func TestLinksFileHoldsNoAttemptsErrorsOrOperations(t *testing.T) {
	f := newWorkspace(t)
	_, _ = mergeLinks(f.Folder(), "fake", "k", fixedNow, sampleOutcome())
	content := readTestFile(t, f.Folder().path(LinksFile))
	for _, forbidden := range []string{"attempts", "last_error", "operations", "state:", "pending", "abandoned"} {
		if strings.Contains(content, forbidden) {
			t.Errorf("integrations.yaml no guarda %q:\n%s", forbidden, content)
		}
	}
	if !strings.HasPrefix(content, "schema: axiom.integrations/v1\nchange: x\n") {
		t.Errorf("cabecera inesperada:\n%s", content)
	}
}

func TestLinksInArchivedFolderAreReadOnly(t *testing.T) {
	f := newWorkspace(t)
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	writeTestFile(t, archived.path(LinksFile), strings.Replace(validLinksYAML, "change: x", "change: x", 1))

	doc, err := readLinks(archived)
	if err != nil || len(doc.Links) != 1 {
		t.Fatalf("la archivada se lee: %+v %v", doc, err)
	}

	before := readTestFile(t, archived.path(LinksFile))
	for _, out := range []Outcome{sampleOutcome(), {Links: []Link{{Kind: "work_item", ID: "12345"}}}} {
		_, err := mergeLinks(archived, "fake", "archive/x/2026-11-19-x", fixedNow, out)
		if err == nil || !strings.Contains(err.Error(), "RF-ARC-03") {
			t.Fatalf("toda escritura en la archivada se rechaza con el mensaje de RF-ARC-03: %v", err)
		}
	}
	if after := readTestFile(t, archived.path(LinksFile)); after != before {
		t.Fatal("integrations.yaml archivado no cambia")
	}
}

func TestLinkWorkItemCreatesFile(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	link := Link{Kind: "work_item", ID: "2525", URL: "https://tracker.example/2525"}

	changed, err := LinkWorkItem(folder, "fake", link, fixedNow)
	if err != nil || !changed {
		t.Fatalf("LinkWorkItem: changed=%v err=%v", changed, err)
	}
	doc, err := readLinks(folder)
	if err != nil || len(doc.Links) != 1 {
		t.Fatalf("readLinks = %+v, %v", doc, err)
	}
	got := doc.Links[0]
	if got.Plugin != "fake" || got.Kind != "work_item" || got.ID != "2525" || got.Source != "change.start" || !got.RecordedAt.Equal(fixedNow) {
		t.Fatalf("vínculo = %+v", got)
	}
}

func TestLinkWorkItemIdempotent(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	link := Link{Kind: "work_item", ID: "2525"}
	if _, err := LinkWorkItem(folder, "fake", link, fixedNow); err != nil {
		t.Fatal(err)
	}
	first := readTestFile(t, folder.path(LinksFile))
	changed, err := LinkWorkItem(folder, "fake", link, fixedNow.Add(time.Hour))
	if err != nil || changed {
		t.Fatalf("repetir no cambia nada: changed=%v err=%v", changed, err)
	}
	if second := readTestFile(t, folder.path(LinksFile)); second != first {
		t.Fatalf("el fichero debe quedar byte a byte igual:\n%s\n---\n%s", first, second)
	}
}

func TestLinkWorkItemKeepsDeliveryLinks(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	if _, err := mergeLinks(folder, "fake", "gate.spec.approved/x/dabc", fixedNow, sampleOutcome()); err != nil {
		t.Fatal(err)
	}
	if _, err := LinkWorkItem(folder, "fake", Link{Kind: "work_item", ID: "2525"}, fixedNow); err != nil {
		t.Fatal(err)
	}
	links, err := WorkItemLinks(folder)
	if err != nil || len(links) != 3 {
		t.Fatalf("WorkItemLinks = %+v, %v; esperaba los 3 vínculos kind work_item", links, err)
	}
	doc, _ := readLinks(folder)
	if len(doc.TimeEntries) != 1 {
		t.Fatalf("las imputaciones deben conservarse: %+v", doc.TimeEntries)
	}
}

func TestLinkWorkItemRejectsUserinfo(t *testing.T) {
	f := newWorkspace(t)
	_, err := LinkWorkItem(f.Folder(), "fake", Link{Kind: "work_item", ID: "1", URL: "https://usuario:clave@tracker.example/1"}, fixedNow)
	if err == nil {
		t.Fatal("una URL con userinfo debe rechazarse")
	}
	if strings.Contains(err.Error(), "clave@") || strings.Contains(err.Error(), "usuario:clave") {
		t.Fatalf("el error no debe repetir las credenciales: %v", err)
	}
	if fileExists(f.Folder().path(LinksFile)) {
		t.Fatal("nada se escribe si el vínculo es inválido")
	}
}
