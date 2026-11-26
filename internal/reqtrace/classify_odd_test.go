package reqtrace

import (
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// oddCandidates compone la lectura y la clasificación ODD, como hará Collect.
func oddCandidates(t *testing.T, files fs.FS, raw string) ([]candidate, []Warning, equivalence) {
	t.Helper()
	p := reqid.Default()
	q, err := ParseID(raw, p)
	if err != nil {
		t.Fatal(err)
	}
	src := Source{Files: files, GitPrefix: "odd/tasks"}
	docs, warnings := listODDDocs(src)
	traces, w := readODDTraces(docs)
	warnings = append(warnings, w...)
	eq, w := resolveEquivalents(q, p, traces, nil)
	warnings = append(warnings, w...)
	var out []candidate
	for _, d := range docs {
		c, w := classifyODD(p, eq, d)
		out = append(out, c...)
		warnings = append(warnings, w...)
	}
	return out, warnings, eq
}

func oddDocText(header, sections string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("# Feature\n\n" + header + "\n## Objetivo\n\nx\n\n" + sections)}
}

const oddSpecViva = "## Spec viva\n\n### Capacidad: pagos\n\n```openspec\n## MODIFIED Requirements\n\n### Requirement: Cobro (REQ-1.1)\n\nEl sistema MUST.\n\n#### Scenario: x\n\n- **DADO** a\n```\n"

func TestRequirementTraceODDRolesTypeAndCubre(t *testing.T) {
	files := fstest.MapFS{
		"modifica.md": oddDocText("", oddSpecViva),
		"bug.md":      oddDocText("Tipo: bug\n", "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: x\n\n## Tareas\n\n- [x] T1\n  Cubre: REQ-1.1\n"),
		"otro.md":     oddDocText("", "## Tareas\n\n- [x] T1\n  Cubre: REQ-1.10\n"),
	}
	got, warns, _ := oddCandidates(t, files, "REQ-1.1")
	if len(warns) != 0 || len(got) != 2 {
		t.Fatalf("incrementos=%+v advertencias=%v", got, warns)
	}
	bug, mod := got[0], got[1]
	if bug.Name != "bug" || bug.Type != KindBug || bug.Role != RoleCovered || bug.Path != "odd/tasks/bug.md" ||
		len(bug.Tasks) != 1 || bug.Tasks[0].File != "odd/tasks/bug.md" || bug.Tasks[0].Line != 17 {
		t.Fatalf("bug ODD = %+v", bug)
	}
	if mod.Name != "modifica" || mod.Type != KindIncrement || mod.Role != RoleModified || mod.Lane != LaneODD ||
		!reflect.DeepEqual(mod.Capabilities, []string{"pagos"}) || !reflect.DeepEqual(mod.IDs, []string{"REQ-1.1"}) {
		t.Fatalf("ODD modificado = %+v", mod)
	}
}

func TestRequirementTraceODDTraceAssignmentsAndProvisional(t *testing.T) {
	doc := "## Spec viva\n\n### Capacidad: pagos\n\n```openspec\n## ADDED Requirements\n\n### Requirement: Cobro (REQ-30.1)\n\nEl sistema MUST.\n\n#### Scenario: x\n\n- **DADO** a\n```\n\n## Identificadores asignados\n\n```yaml\n" + validTrace + "```\n"
	files := fstest.MapFS{"cobro.md": oddDocText("", doc)}
	for _, raw := range []string{"REQ-30.1", "prov:trace:1"} {
		got, warns, _ := oddCandidates(t, files, raw)
		if len(warns) != 0 || len(got) != 1 || got[0].Role != RoleCreated || len(got[0].Assignments) != 1 ||
			got[0].Assignments[0].Provisional != "prov:trace:1" {
			t.Fatalf("%s: incrementos=%+v advertencias=%v", raw, got, warns)
		}
	}
	// Solo la traza, sin aparición en la delta: created.
	onlyTrace := "## Identificadores asignados\n\n```yaml\n" + validTrace + "```\n"
	got, _, _ := oddCandidates(t, fstest.MapFS{"t.md": oddDocText("", onlyTrace)}, "REQ-30.1")
	if len(got) != 1 || got[0].Role != RoleCreated || !reflect.DeepEqual(got[0].IDs, []string{"REQ-30.1"}) {
		t.Fatalf("solo traza = %+v", got)
	}
}

func TestRequirementTraceODDOldDocumentWithoutTrace(t *testing.T) {
	got, warns, _ := oddCandidates(t, fstest.MapFS{"viejo.md": oddDocText("", oddSpecViva)}, "REQ-1.1")
	if len(got) != 1 || len(warns) != 0 {
		t.Fatalf("incrementos=%+v advertencias=%v", got, warns)
	}
}

func TestRequirementTraceODDTolerantSearchOnUnparseableDocument(t *testing.T) {
	// Dos secciones `## Spec viva` rompen el parseo estricto; el texto sigue teniendo delta, Cubre: y Tipo: bug.
	broken := "# Feature\n\nTipo: bug\n\n" + oddSpecViva + "\n## Spec viva\n\nduplicada\n\n## Tareas\n\n- [x] T1\n  Cubre: REQ-1.1\n\n```\nCubre: REQ-1.1\n```\n"
	files := fstest.MapFS{"roto.md": {Data: []byte(broken)}}
	got, warns, _ := oddCandidates(t, files, "REQ-1.1")
	if len(warns) != 1 || warns[0].Code != WarnODDUnparseable || warns[0].Path != "odd/tasks/roto.md" {
		t.Fatalf("advertencias = %+v", warns)
	}
	if !reflect.DeepEqual(roles(got), []Role{RoleModified, RoleCovered}) || got[0].Type != KindBug ||
		!reflect.DeepEqual(got[0].Capabilities, []string{"pagos"}) || len(got[1].Tasks) != 1 {
		t.Fatalf("incrementos = %+v", got)
	}
	// Un documento que no parsea y no cita el ID no produce ruido.
	other := fstest.MapFS{"roto.md": {Data: []byte("Tipo: raro\n\nnada\n")}}
	if got, warns, _ := oddCandidates(t, other, "REQ-1.1"); len(got) != 0 || len(warns) != 0 {
		t.Fatalf("sin cita: %v %v", got, warns)
	}
}

func TestRequirementTraceODDProseOnlyMentionWarns(t *testing.T) {
	files := fstest.MapFS{"prosa.md": {Data: []byte("Tipo: raro\n\nSe habla de REQ-1.1 sin tocarlo.\n")}}
	got, warns, _ := oddCandidates(t, files, "REQ-1.1")
	if len(got) != 0 || len(warns) != 2 || warns[0].Code != WarnODDUnparseable || warns[1].Code != WarnUnclassifiedMention {
		t.Fatalf("incrementos=%+v advertencias=%+v", got, warns)
	}
}

func TestRequirementTraceODDBrokenTraceWarnsAndIgnoresIt(t *testing.T) {
	doc := oddSpecViva + "\n## Identificadores asignados\n\nsin bloque yaml\n"
	got, warns, _ := oddCandidates(t, fstest.MapFS{"x.md": oddDocText("", doc)}, "REQ-1.1")
	if len(got) != 1 || len(warns) != 1 || warns[0].Code != WarnTraceInvalid {
		t.Fatalf("incrementos=%+v advertencias=%+v", got, warns)
	}
}

func TestRequirementTraceODDUnsafeNamesAndEmptyDirectory(t *testing.T) {
	files := fstest.MapFS{
		"-x.md":    oddDocText("", oddSpecViva),
		"a[1].md":  oddDocText("", oddSpecViva),
		"a b.md":   oddDocText("", oddSpecViva),
		"nota.txt": {Data: []byte("x")},
		"sub/y.md": oddDocText("", oddSpecViva),
	}
	got, warns, _ := oddCandidates(t, files, "REQ-1.1")
	if len(got) != 0 || len(warns) != 3 {
		t.Fatalf("incrementos=%v advertencias=%v", got, warns)
	}
	for _, w := range warns {
		if w.Code != WarnUnsafeName {
			t.Fatalf("advertencia = %+v", w)
		}
	}
	if got, warns, _ := oddCandidates(t, fstest.MapFS{}, "REQ-1.1"); len(got) != 0 || len(warns) != 0 {
		t.Fatalf("directorio vacío: %v %v", got, warns)
	}
}
