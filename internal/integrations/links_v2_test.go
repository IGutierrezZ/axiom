package integrations

import (
	"strings"
	"testing"
	"time"
)

func f64(v float64) *float64 { return &v }

func taskLink(role, id string, estimate *float64) Link {
	return Link{Kind: TaskLinkKind, ID: id, Role: role, Source: TasksBindSource, EstimateHours: estimate}
}

func TestLinksSchemaV1IsReadWithoutRewriting(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	writeTestFile(t, folder.path(LinksFile), validLinksYAML)

	// Una fusión sin cambios no reescribe: un v1 intacto sigue siendo v1.
	changed, err := mergeLinks(folder, "fake", "k", fixedNow, Outcome{Links: []Link{{Kind: "work_item", ID: "12345"}}})
	if err != nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if got := readTestFile(t, folder.path(LinksFile)); got != validLinksYAML {
		t.Fatalf("el v1 intacto no se reescribe:\n%s", got)
	}

	// Una fusión que cambia algo lo escribe como v2 y conserva lo anterior.
	changed, err = mergeLinks(folder, "fake", "k", fixedNow, Outcome{Links: []Link{{Kind: "work_item", ID: "99"}}})
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	got := readTestFile(t, folder.path(LinksFile))
	if !strings.Contains(got, "schema: "+LinksSchemaV2) || strings.Contains(got, LinksSchemaV1) {
		t.Fatalf("al cambiar se escribe v2:\n%s", got)
	}
	doc, err := readLinks(folder)
	if err != nil || len(doc.Links) != 2 || len(doc.TimeEntries) != 1 {
		t.Fatalf("doc = %+v, %v", doc, err)
	}
}

func TestReadLinksRejectsUnknownSchemaNamingIt(t *testing.T) {
	f := newWorkspace(t)
	// Con campos desconocidos: gana el rechazo del esquema, no el error de campo.
	writeTestFile(t, f.Folder().path(LinksFile), "schema: axiom.integrations/v9\nchange: x\nextra: 1\n")
	_, err := readLinks(f.Folder())
	want := `esquema "axiom.integrations/v9" no admitido; admitidos: axiom.integrations/v1, axiom.integrations/v2`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, se esperaba %q", err, want)
	}
}

func TestLinksV2ValidationTable(t *testing.T) {
	base := "schema: axiom.integrations/v2\nchange: x\n"
	tests := []struct {
		name  string
		yaml  string
		wants string // vacío: válido
	}{
		{"estimación válida", base + "links:\n  - {plugin: fake, kind: task, id: \"7\", role: be, source: tasks.bind, recorded_at: 2026-11-18T22:30:00Z, estimate_hours: 2.25}\n", ""},
		{"estimación cero", base + "links:\n  - {plugin: fake, kind: task, id: \"7\", source: tasks.bind, recorded_at: 2026-11-18T22:30:00Z, estimate_hours: 0}\n", "estimate_hours"},
		{"estimación con tres decimales", base + "links:\n  - {plugin: fake, kind: task, id: \"7\", source: tasks.bind, recorded_at: 2026-11-18T22:30:00Z, estimate_hours: 1.125}\n", "dos decimales"},
		{"started_at en UTC", base + "links:\n  - {plugin: fake, kind: task, id: \"7\", source: tasks.bind, recorded_at: 2026-11-18T22:30:00Z, started_at: 2026-11-18T09:00:00Z}\n", ""},
		{"started_at con desfase", base + "links:\n  - {plugin: fake, kind: task, id: \"7\", source: tasks.bind, recorded_at: 2026-11-18T22:30:00Z, started_at: 2026-11-18T09:00:00+01:00}\n", "UTC"},
		{"role.close sin actor", base + "time_entries:\n  - {plugin: fake, role: be, item_id: \"7\", minutes: 60, date: 2026-11-18, source: role.close/be/1, recorded_at: 2026-11-18T22:30:00Z}\n", "actor"},
		{"role.close sin role", base + "time_entries:\n  - {plugin: fake, actor: ana, item_id: \"7\", minutes: 60, date: 2026-11-18, source: role.close/be/1, recorded_at: 2026-11-18T22:30:00Z}\n", "role"},
		{"role.close completa", base + "time_entries:\n  - {plugin: fake, role: be, actor: ana, item_id: \"7\", minutes: 60, date: 2026-11-18, source: role.close/be/1, recorded_at: 2026-11-18T22:30:00Z}\n", ""},
		{"otro origen sin actor", base + "time_entries:\n  - {plugin: fake, item_id: \"7\", minutes: 60, date: 2026-11-18, source: role.apply.approved/x, recorded_at: 2026-11-18T22:30:00Z}\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newWorkspace(t)
			writeTestFile(t, f.Folder().path(LinksFile), tt.yaml)
			_, err := readLinks(f.Folder())
			switch {
			case tt.wants == "" && err != nil:
				t.Fatalf("debía ser válido: %v", err)
			case tt.wants != "" && (err == nil || !strings.Contains(err.Error(), tt.wants)):
				t.Fatalf("se esperaba un error con %q, obtuvo %v", tt.wants, err)
			}
		})
	}
}

func TestMergeLinksUpdatesEstimateAndFixesStartedAtOnce(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	first := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	later := first.Add(48 * time.Hour)

	l := taskLink("be", "7", f64(4))
	l.StartedAt = &first
	if _, err := mergeLinks(folder, "fake", "tasks.bind", fixedNow, Outcome{Links: []Link{l}}); err != nil {
		t.Fatal(err)
	}

	// Misma estimación: sin cambios. Otra distinta: se actualiza. Nula: se conserva.
	if changed, _ := mergeLinks(folder, "fake", "tasks.bind", fixedNow, Outcome{Links: []Link{taskLink("be", "7", f64(4))}}); changed {
		t.Fatal("la misma estimación no cambia el fichero")
	}
	if changed, err := mergeLinks(folder, "fake", "tasks.bind", fixedNow, Outcome{Links: []Link{taskLink("be", "7", f64(6.5))}}); err != nil || !changed {
		t.Fatalf("una estimación distinta se actualiza: changed=%v err=%v", changed, err)
	}
	if changed, _ := mergeLinks(folder, "fake", "tasks.bind", fixedNow, Outcome{Links: []Link{taskLink("be", "7", nil)}}); changed {
		t.Fatal("una estimación nula no cambia nada")
	}

	// started_at es de escritura única.
	other := taskLink("be", "7", nil)
	other.StartedAt = &later
	if changed, _ := mergeLinks(folder, "fake", "tasks.bind", fixedNow, Outcome{Links: []Link{other}}); changed {
		t.Fatal("started_at ya fijado no se sustituye")
	}
	links, err := TaskLinks(folder)
	if err != nil || len(links) != 1 {
		t.Fatalf("TaskLinks = %+v, %v", links, err)
	}
	if *links[0].EstimateHours != 6.5 || !links[0].StartedAt.Equal(first) {
		t.Fatalf("estimación %v, inicio %v", *links[0].EstimateHours, links[0].StartedAt)
	}
}

func TestMarkRoleStartedIsWriteOnce(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	at := time.Date(2026, 11, 18, 10, 30, 0, 0, time.FixedZone("CET", 3600))

	if _, err := MarkRoleStarted(folder, "fake", "be", at); err == nil {
		t.Fatal("sin vínculo de Task del rol es un error")
	}
	if _, err := BindTask(folder, "fake", taskLink("be", "7", f64(3)), fixedNow); err != nil {
		t.Fatal(err)
	}
	changed, err := MarkRoleStarted(folder, "fake", "be", at)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	changed, err = MarkRoleStarted(folder, "fake", "be", at.Add(time.Hour))
	if err != nil || changed {
		t.Fatalf("la segunda vez no cambia: changed=%v err=%v", changed, err)
	}
	links, _ := TaskLinks(folder)
	if !links[0].StartedAt.Equal(at) || links[0].StartedAt.Location() != time.UTC {
		t.Fatalf("started_at = %v, se guarda en UTC", links[0].StartedAt)
	}
}

func TestBindTaskChecksSourceAndKind(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	bad := taskLink("be", "7", nil)
	bad.Source = "gate.spec.approved/x"
	if _, err := BindTask(folder, "fake", bad, fixedNow); err == nil {
		t.Fatal("el source debe ser tasks.create o tasks.bind")
	}
	if _, err := BindTask(folder, "fake", Link{Kind: "work_item", ID: "1", Source: TasksBindSource}, fixedNow); err == nil {
		t.Fatal("solo vincula kind task")
	}
	if fileExists(folder.path(LinksFile)) {
		t.Fatal("nada se escribe si el vínculo es inválido")
	}
	for _, src := range []string{TasksCreateSource, TasksBindSource} {
		l := taskLink("be", "7", f64(2))
		l.Source = src
		if _, err := BindTask(folder, "fake", l, fixedNow); err != nil {
			t.Fatalf("source %s: %v", src, err)
		}
	}
}

func TestReplaceTaskLinkKeepsStartedAtAndEntriesAndRemovesOnlyTheRole(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	started := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

	old := taskLink("be", "205", f64(5))
	old.StartedAt = &started
	for _, l := range []Link{{Kind: "work_item", ID: "100", Source: "change.start"}, old, taskLink("fe", "300", nil)} {
		if _, err := mergeLinks(folder, "fake", "tasks.bind", fixedNow, Outcome{Links: []Link{l}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, src := range []string{"role.close/be/1", "role.close/be/2"} {
		e := TimeEntry{Role: "be", Actor: "ana", ItemID: "205", Minutes: 60, Date: "2026-11-18", Source: src}
		if _, err := RecordTimeEntry(folder, "fake", e, fixedNow); err != nil {
			t.Fatal(err)
		}
	}

	changed, err := ReplaceTaskLink(folder, "fake", "be", taskLink("be", "206", f64(8)), fixedNow)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	doc, err := readLinks(folder)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, l := range doc.Links {
		kinds = append(kinds, l.Kind+":"+l.ID)
		if l.Kind == TaskLinkKind && l.Role == "be" && (l.StartedAt == nil || !l.StartedAt.Equal(started)) {
			t.Fatalf("el started_at anterior se copia: %+v", l)
		}
	}
	if strings.Join(kinds, ",") != "work_item:100,task:300,task:206" {
		t.Fatalf("vínculos tras sustituir = %v", kinds)
	}
	if len(doc.TimeEntries) != 2 {
		t.Fatalf("time_entries no se toca: %+v", doc.TimeEntries)
	}

	// Sustituir por la misma Task es idempotente.
	if changed, err := ReplaceTaskLink(folder, "fake", "be", taskLink("be", "206", f64(8)), fixedNow); err != nil || changed {
		t.Fatalf("idempotente: changed=%v err=%v", changed, err)
	}
	// Rol que no corresponde con el vínculo.
	if _, err := ReplaceTaskLink(folder, "fake", "fe", taskLink("be", "1", nil), fixedNow); err == nil {
		t.Fatal("el rol del vínculo debe ser el indicado")
	}
}

func TestReplaceTaskLinkWithoutPreviousLinkBehavesAsBind(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	changed, err := ReplaceTaskLink(folder, "fake", "be", taskLink("be", "5", nil), fixedNow)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	links, _ := TaskLinks(folder)
	if len(links) != 1 || links[0].ID != "5" {
		t.Fatalf("links = %+v", links)
	}
}

func TestTwoRoleCloseEntriesCoexistAndRequireActor(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	for i, src := range []string{"role.close/be/1", "role.close/be/2"} {
		e := TimeEntry{Role: "be", Actor: "ana", ItemID: "205", Minutes: 30 * (i + 1), Date: "2026-11-18", Source: src}
		if changed, err := RecordTimeEntry(folder, "fake", e, fixedNow); err != nil || !changed {
			t.Fatalf("%s: changed=%v err=%v", src, changed, err)
		}
	}
	entries, err := TimeEntries(folder)
	if err != nil || len(entries) != 2 || entries[0].Actor != "ana" {
		t.Fatalf("entries = %+v, %v", entries, err)
	}
	// Repetir la primera es idempotente.
	if changed, _ := RecordTimeEntry(folder, "fake", TimeEntry{Role: "be", Actor: "ana", ItemID: "205", Minutes: 30, Date: "2026-11-18", Source: "role.close/be/1"}, fixedNow); changed {
		t.Fatal("repetir una imputación no cambia el fichero")
	}
	before := readTestFile(t, folder.path(LinksFile))
	_, err = RecordTimeEntry(folder, "fake", TimeEntry{Role: "be", ItemID: "205", Minutes: 30, Date: "2026-11-18", Source: "role.close/be/3"}, fixedNow)
	if err == nil || !IsPermanent(err) || !strings.Contains(err.Error(), "actor") {
		t.Fatalf("role.close/* sin actor es un error permanente: %v", err)
	}
	if readTestFile(t, folder.path(LinksFile)) != before {
		t.Fatal("el rechazo no escribe")
	}
}

func TestTaskWritersRespectTheArchivedGuard(t *testing.T) {
	f := newGitWorkspace(t)
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	writeTestFile(t, archived.path(LinksFile), validLinksYAML)
	before := readTestFile(t, archived.path(LinksFile))

	checks := map[string]func() error{
		"BindTask": func() error { _, err := BindTask(archived, "fake", taskLink("be", "7", nil), fixedNow); return err },
		"ReplaceTaskLink": func() error {
			_, err := ReplaceTaskLink(archived, "fake", "be", taskLink("be", "7", nil), fixedNow)
			return err
		},
		"MarkRoleStarted": func() error { _, err := MarkRoleStarted(archived, "fake", "be", fixedNow); return err },
		"RecordTimeEntry": func() error {
			_, err := RecordTimeEntry(archived, "fake", TimeEntry{ItemID: "7", Minutes: 5, Date: "2026-11-18", Source: "role.close/be/9", Role: "be", Actor: "a"}, fixedNow)
			return err
		},
	}
	for name, run := range checks {
		if err := run(); err == nil || !strings.Contains(err.Error(), "RF-ARC-03") {
			t.Errorf("%s en la archivada debe rechazarse con RF-ARC-03: %v", name, err)
		}
	}
	if readTestFile(t, archived.path(LinksFile)) != before {
		t.Fatal("integrations.yaml archivado no cambia")
	}
}
