package reqtrace

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// sddCandidates compone la lectura y la clasificación SDD, como hará Collect.
func sddCandidates(t *testing.T, files fs.FS, raw string, aliases AliasResolver) ([]candidate, []Warning, equivalence) {
	t.Helper()
	p := reqid.Default()
	q, err := ParseID(raw, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	src := Source{Files: files, GitPrefix: "openspec"}
	folders, warnings := listSDDFolders(src)
	traces, w := readSDDTraces(src, folders)
	warnings = append(warnings, w...)
	eq, w := resolveEquivalents(q, p, traces, aliases)
	warnings = append(warnings, w...)
	var out []candidate
	for _, f := range folders {
		c, w := classifySDD(src, p, eq, f)
		out = append(out, c...)
		warnings = append(warnings, w...)
	}
	return out, warnings, eq
}

func deltaSpec(section, title string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("## " + section + " Requirements\n\n### Requirement: " + title + "\n\nEl sistema MUST algo.\n\n#### Scenario: x\n\n- **DADO** a\n")}
}

func roles(cs []candidate) []Role {
	var out []Role
	for _, c := range cs {
		out = append(out, c.Role)
	}
	return out
}

func TestRequirementTraceSDDRolesByDeltaSection(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/specs/pagos/spec.md": deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-02-b/specs/pagos/spec.md": deltaSpec("MODIFIED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-03-c/specs/pagos/spec.md": deltaSpec("REMOVED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-04-d/specs/pagos/spec.md": deltaSpec("RENAMED", "Cobro (REQ-1.1) → Cobro nuevo (REQ-1.1)"),
		"changes/archive/2026-01-05-e/specs/pagos/spec.md": deltaSpec("RENAMED", "Viejo (REQ-9.9) → Cobro (REQ-1.1)"),
		"changes/archive/2026-01-06-f/specs/pagos/spec.md": deltaSpec("ADDED", "Otro (REQ-1.10)"),
		"changes/archive/2026-01-07-g/specs/pagos/spec.md": deltaSpec("ADDED", "Otro (REQ-11.1)"),
		"changes/no-archivado/specs/pagos/spec.md":         deltaSpec("ADDED", "Cobro (REQ-1.1)"),
	}
	got, warns, _ := sddCandidates(t, files, "REQ-1.1", nil)
	if len(warns) != 0 {
		t.Fatalf("advertencias = %v", warns)
	}
	if want := []Role{RoleCreated, RoleModified, RoleRemoved, RoleRenamed, RoleRenamed}; !reflect.DeepEqual(roles(got), want) {
		t.Fatalf("roles = %v; quería %v (REQ-1.10, REQ-11.1 y el cambio activo no cuentan)", roles(got), want)
	}
	first := got[0]
	if first.Lane != LaneSDD || first.Name != "a" || first.Path != "openspec/changes/archive/2026-01-01-a" ||
		first.Type != KindIncrement || !reflect.DeepEqual(first.IDs, []string{"REQ-1.1"}) || !reflect.DeepEqual(first.Capabilities, []string{"pagos"}) {
		t.Fatalf("incremento = %+v", first)
	}
}

func TestRequirementTraceSDDFullSpecCountsAsAdded(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/specs/nueva/spec.md": {Data: []byte("# Spec\n\n## Purpose\n\nx\n\n## Requirements\n\n### Requirement: Cobro (REQ-1.1)\n\nEl sistema MUST.\n\n#### Scenario: x\n\n- **DADO** a\n")},
	}
	got, _, _ := sddCandidates(t, files, "REQ-1.1", nil)
	if !reflect.DeepEqual(roles(got), []Role{RoleCreated}) {
		t.Fatalf("roles = %v", roles(got))
	}
}

func TestRequirementTraceSDDCubreWithTasksAndSeveralFiles(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/tasks.md":        {Data: []byte("- [x] 1.1 algo\n  Cubre: REQ-1.1\n- [x] 1.2 otra\n  Cubre: REQ-2.1, REQ-1.1\n```\nCubre: REQ-1.1\n```\n")},
		"changes/archive/2026-01-01-a/tasks.api.md":    {Data: []byte("Cubre: REQ-1.1\nCubre: REQ-1.10\n")},
		"changes/archive/2026-01-01-a/tasksx.md":       {Data: []byte("Cubre: REQ-1.1\n")},
		"changes/archive/2026-01-01-a/kickoff.yaml":    {Data: []byte("roles: []\n")},
		"changes/archive/2026-01-02-b/tasks.md":        {Data: []byte("Cubre: REQ-1.10\n")},
		"changes/archive/2026-01-03-a/specs/x/spec.md": deltaSpec("ADDED", "Cobro (REQ-1.1)"),
	}
	got, _, _ := sddCandidates(t, files, "REQ-1.1", nil)
	if !reflect.DeepEqual(roles(got), []Role{RoleCovered, RoleCreated}) {
		t.Fatalf("roles = %v", roles(got))
	}
	covered := got[0]
	want := []Task{
		{File: "openspec/changes/archive/2026-01-01-a/tasks.api.md", Line: 1},
		{File: "openspec/changes/archive/2026-01-01-a/tasks.md", Line: 2},
		{File: "openspec/changes/archive/2026-01-01-a/tasks.md", Line: 4},
	}
	if !reflect.DeepEqual(covered.Tasks, want) {
		t.Fatalf("tareas = %+v; quería %+v (las líneas vaciadas no cuentan)", covered.Tasks, want)
	}
	if got[1].Tasks != nil {
		t.Fatalf("solo covered lista tareas: %+v", got[1].Tasks)
	}
}

func TestRequirementTraceSDDSameChangeNameInTwoFolders(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-demo/specs/x/spec.md": deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-03-01-demo/specs/x/spec.md": deltaSpec("ADDED", "Otro (REQ-2.1)"),
	}
	got, _, _ := sddCandidates(t, files, "REQ-1.1", nil)
	if len(got) != 1 || got[0].Path != "openspec/changes/archive/2026-01-01-demo" || got[0].Name != "demo" {
		t.Fatalf("incrementos = %+v", got)
	}
}

func TestRequirementTraceSDDTypeBugAndTolerance(t *testing.T) {
	cubre := &fstest.MapFile{Data: []byte("Cubre: REQ-1.1\n")}
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/tasks.md":    cubre,
		"changes/archive/2026-01-01-a/proposal.md": {Data: []byte("# P\n\n## Informe de bug\n\nObservado: x\nEsperado: y\n")},
		"changes/archive/2026-01-02-b/tasks.md":    cubre,
		"changes/archive/2026-01-02-b/proposal.md": {Data: []byte("# P\n\n## Impacto en la spec\n\nSin impacto en la spec\nMotivo: x\n")},
		"changes/archive/2026-01-03-c/tasks.md":    cubre,
		"changes/archive/2026-01-03-c/proposal.md": {Data: []byte("# P\n\n## Por qué\n\nx\n")},
		"changes/archive/2026-01-04-d/tasks.md":    cubre,
		// Bug con delta y sin secciones de bug: sale increment (limitación documentada).
		"changes/archive/2026-01-05-e/specs/x/spec.md": deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-05-e/proposal.md":     {Data: []byte("# P\n\nUn bug.\n")},
	}
	got, warns, _ := sddCandidates(t, files, "REQ-1.1", nil)
	if len(warns) != 0 {
		t.Fatalf("advertencias = %v", warns)
	}
	var kinds []Kind
	for _, c := range got {
		kinds = append(kinds, c.Type)
	}
	if want := []Kind{KindBug, KindBug, KindIncrement, KindIncrement, KindIncrement}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("tipos = %v; quería %v", kinds, want)
	}
}

func TestRequirementTraceSDDIdsYamlOnlyIsCreated(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/ids.yaml": {Data: []byte(validTrace)},
	}
	for _, raw := range []string{"REQ-30.1", "prov:trace:1"} {
		got, _, eq := sddCandidates(t, files, raw, nil)
		if len(got) != 1 || got[0].Role != RoleCreated || !reflect.DeepEqual(got[0].IDs, []string{"REQ-30.1"}) ||
			!reflect.DeepEqual(got[0].Capabilities, []string{"pagos"}) {
			t.Fatalf("%s: incrementos = %+v", raw, got)
		}
		if len(got[0].Assignments) != 1 || got[0].Assignments[0].Provisional != "prov:trace:1" {
			t.Fatalf("%s: asignaciones = %+v", raw, got[0].Assignments)
		}
		if !reflect.DeepEqual(eq.resolved, []string{"REQ-30.1", "prov:trace:1"}) {
			t.Fatalf("%s: resueltos = %v", raw, eq.resolved)
		}
	}
}

func TestRequirementTraceSDDEmptyAndMissingArchive(t *testing.T) {
	for name, files := range map[string]fs.FS{
		"inexistente": fstest.MapFS{"otra/cosa": {Data: []byte("x")}},
		"vacío":       fstest.MapFS{"changes/archive/.gitkeep": {Data: []byte("")}},
	} {
		got, warns, _ := sddCandidates(t, files, "REQ-1.1", nil)
		if len(got) != 0 || len(warns) != 0 {
			t.Fatalf("%s: %v %v", name, got, warns)
		}
	}
}

func TestRequirementTraceSDDUnsafeNamesAreSkippedWithWarning(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a*/specs/x/spec.md":   deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-01-a[1]/specs/x/spec.md": deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/-x/specs/x/spec.md":              deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-01-a..b/specs/x/spec.md": deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/sin-fecha/specs/x/spec.md":       deltaSpec("ADDED", "Cobro (REQ-1.1)"),
	}
	got, warns, _ := sddCandidates(t, files, "REQ-1.1", nil)
	if len(got) != 0 || len(warns) != 5 {
		t.Fatalf("incrementos=%v advertencias=%v", got, warns)
	}
	for _, w := range warns {
		if w.Code != WarnUnsafeName || w.Path == "" {
			t.Fatalf("advertencia = %+v", w)
		}
	}
}

// failingFS devuelve un error de lectura que no es «no existe» para una ruta.
type failingFS struct {
	fs.FS
	path string
}

func (f failingFS) Open(name string) (fs.File, error) {
	if name == f.path {
		return nil, errors.New("acceso denegado")
	}
	return f.FS.Open(name)
}

func TestRequirementTraceSDDUnreadableDeltaWarnsAndContinues(t *testing.T) {
	base := fstest.MapFS{
		"changes/archive/2026-01-01-a/specs/x/spec.md": deltaSpec("ADDED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-01-a/specs/y/spec.md": deltaSpec("MODIFIED", "Cobro (REQ-1.1)"),
	}
	files := failingFS{FS: base, path: "changes/archive/2026-01-01-a/specs/x/spec.md"}
	got, warns, _ := sddCandidates(t, files, "REQ-1.1", nil)
	if len(warns) != 1 || warns[0].Code != WarnUnreadable || warns[0].Path != "openspec/changes/archive/2026-01-01-a/specs/x/spec.md" {
		t.Fatalf("advertencias = %+v", warns)
	}
	if !reflect.DeepEqual(roles(got), []Role{RoleModified}) {
		t.Fatalf("la búsqueda debe seguir con el resto del cambio: %v", roles(got))
	}
}
