package reqtrace

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
)

// fakeHistory es una historia lineal: line es la línea --first-parent de la rama, del más
// antiguo al más reciente.
type fakeHistory struct {
	line    []string
	added   map[string][]string // ruta -> commits que la añaden, del más antiguo al más reciente
	entries map[string][]string // commit|directorio -> entradas
	badRef  string
	failOn  string // commit cuyo IsAncestor falla
	calls   []string
}

func (f *fakeHistory) pos(c string) int {
	for i, l := range f.line {
		if l == c {
			return i
		}
	}
	return -1
}

func (f *fakeHistory) FirstParentAdded(_ context.Context, ref, path string) ([]string, error) {
	f.calls = append(f.calls, "added "+path)
	if ref == f.badRef {
		return nil, errors.New("fatal: ambiguous argument: unknown revision")
	}
	return f.added[path], nil
}

func (f *fakeHistory) TreeHas(_ context.Context, commit, path string) (bool, error) {
	f.calls = append(f.calls, "has "+commit+" "+path)
	if commit == f.badRef {
		return false, errors.New("git ls-tree falló (salida 128): fatal: Not a valid object name " + commit)
	}
	return false, nil
}

func (f *fakeHistory) TreeEntries(_ context.Context, ref, dir string) ([]string, error) {
	f.calls = append(f.calls, "entries "+ref+" "+dir)
	return f.entries[ref+"|"+dir], nil
}

func (f *fakeHistory) IsAncestor(_ context.Context, a, d string) (bool, error) {
	f.calls = append(f.calls, "ancestor "+a+" "+d)
	if a == f.failOn || d == f.failOn {
		return false, errors.New("git merge-base falló")
	}
	return f.pos(a) <= f.pos(d), nil
}

func sddCand(change, folder string) candidate {
	return candidate{
		Increment: Increment{Lane: LaneSDD, Name: change, Path: "openspec/changes/archive/" + folder, Type: KindIncrement, Role: RoleCreated},
		change:    change,
	}
}

func TestRequirementTraceAnchorSDDCodeAndArchive(t *testing.T) {
	h := &fakeHistory{
		line: []string{"c1", "c2", "c3"},
		added: map[string][]string{
			"openspec/changes/demo/verify-report.md":                {"c1"},
			"openspec/changes/archive/2026-01-01-demo/kickoff.yaml": {"c3"},
		},
		entries: map[string][]string{"c1|openspec/changes/demo": {"tasks.md", "verify-report.md"}},
	}
	sdd := Source{History: h, GitPrefix: "openspec"}
	incs, warns, err := anchorCandidates(context.Background(), "main", sdd, Source{}, []candidate{sddCand("demo", "2026-01-01-demo")})
	if err != nil || len(warns) != 0 || len(incs) != 1 {
		t.Fatalf("incs=%v warns=%v err=%v", incs, warns, err)
	}
	got := incs[0]
	if got.Integration != (Commit{Commit: "c1", Found: true, Anchor: AnchorCode}) || got.Archive == nil || *got.Archive != (Commit{Commit: "c3", Found: true}) {
		t.Fatalf("anclaje = %+v archive=%+v", got.Integration, got.Archive)
	}
}

func TestRequirementTraceAnchorSDDFallbacks(t *testing.T) {
	t.Run("sin kickoff.yaml ancla por la carpeta", func(t *testing.T) {
		h := &fakeHistory{
			line:  []string{"c1", "c2"},
			added: map[string][]string{"openspec/changes/archive/2026-01-01-demo": {"c2"}},
		}
		incs, _, err := anchorCandidates(context.Background(), "main", Source{History: h, GitPrefix: "openspec"}, Source{}, []candidate{sddCand("demo", "2026-01-01-demo")})
		if err != nil || incs[0].Integration != (Commit{Commit: "c2", Found: true, Anchor: AnchorArchive}) || !incs[0].Archive.Found {
			t.Fatalf("incs=%+v err=%v", incs, err)
		}
	})
	t.Run("sin verify-report ancla al archivado", func(t *testing.T) {
		h := &fakeHistory{line: []string{"c1"}, added: map[string][]string{"openspec/changes/archive/2026-01-01-demo/kickoff.yaml": {"c1"}}}
		incs, _, _ := anchorCandidates(context.Background(), "main", Source{History: h, GitPrefix: "openspec"}, Source{}, []candidate{sddCand("demo", "2026-01-01-demo")})
		if incs[0].Integration.Anchor != AnchorArchive {
			t.Fatalf("anclaje = %+v", incs[0].Integration)
		}
	})
	t.Run("ninguno da found false sin error", func(t *testing.T) {
		h := &fakeHistory{}
		incs, warns, err := anchorCandidates(context.Background(), "main", Source{History: h, GitPrefix: "openspec"}, Source{}, []candidate{sddCand("demo", "2026-01-01-demo")})
		if err != nil || len(warns) != 0 || incs[0].Integration.Found || incs[0].Archive == nil || incs[0].Archive.Found {
			t.Fatalf("incs=%+v warns=%v err=%v", incs, warns, err)
		}
	})
	t.Run("verify-report sin tasks en el árbol se salta", func(t *testing.T) {
		h := &fakeHistory{
			line:    []string{"c1", "c2"},
			added:   map[string][]string{"openspec/changes/demo/verify-report.md": {"c1"}, "openspec/changes/archive/2026-01-01-demo/kickoff.yaml": {"c2"}},
			entries: map[string][]string{"c1|openspec/changes/demo": {"verify-report.md"}},
		}
		incs, _, _ := anchorCandidates(context.Background(), "main", Source{History: h, GitPrefix: "openspec"}, Source{}, []candidate{sddCand("demo", "2026-01-01-demo")})
		if incs[0].Integration.Anchor != AnchorArchive {
			t.Fatalf("anclaje = %+v", incs[0].Integration)
		}
	})
}

func TestRequirementTraceAnchorSDDTwoFoldersSameChangeNameTakeOwnCommit(t *testing.T) {
	h := &fakeHistory{
		line: []string{"a-code", "a-arch", "b-code", "b-arch"},
		added: map[string][]string{
			"openspec/changes/demo/verify-report.md":                {"a-code", "b-code"},
			"openspec/changes/archive/2026-01-01-demo/kickoff.yaml": {"a-arch"},
			"openspec/changes/archive/2026-03-01-demo/kickoff.yaml": {"b-arch"},
		},
		entries: map[string][]string{
			"a-code|openspec/changes/demo": {"tasks.md"},
			"b-code|openspec/changes/demo": {"tasks.md"},
		},
	}
	incs, _, err := anchorCandidates(context.Background(), "main", Source{History: h, GitPrefix: "openspec"}, Source{},
		[]candidate{sddCand("demo", "2026-01-01-demo"), sddCand("demo", "2026-03-01-demo")})
	if err != nil || incs[0].Integration.Commit != "a-code" || incs[1].Integration.Commit != "b-code" {
		t.Fatalf("incs=%+v err=%v", incs, err)
	}
}

func TestRequirementTraceAnchorODD(t *testing.T) {
	h := &fakeHistory{added: map[string][]string{"odd/tasks/a.md": {"c1", "c9"}}}
	odd := Source{History: h, GitPrefix: "odd/tasks"}
	cands := []candidate{
		{Increment: Increment{Lane: LaneODD, Name: "a", Path: "odd/tasks/a.md", Role: RoleModified}},
		{Increment: Increment{Lane: LaneODD, Name: "a", Path: "odd/tasks/a.md", Role: RoleCovered}},
		{Increment: Increment{Lane: LaneODD, Name: "b", Path: "odd/tasks/b.md", Role: RoleModified}},
	}
	incs, warns, err := anchorCandidates(context.Background(), "main", Source{}, odd, cands)
	if err != nil || len(incs) != 2 || incs[0].Integration != (Commit{Commit: "c1", Found: true, Anchor: AnchorDocument}) {
		t.Fatalf("incs=%+v err=%v", incs, err)
	}
	if len(warns) != 1 || warns[0].Code != WarnODDNotIntegrated || warns[0].Path != "odd/tasks/b.md" {
		t.Fatalf("advertencias = %+v", warns)
	}
	// Sin historia ODD, todos se omiten.
	incs, _, err = anchorCandidates(context.Background(), "main", Source{}, Source{GitPrefix: "odd/tasks"}, cands)
	if err != nil || len(incs) != 0 {
		t.Fatalf("sin historia: %+v %v", incs, err)
	}
}

func TestRequirementTraceAnchorUsesEachSourcesOwnHistory(t *testing.T) {
	specs := &fakeHistory{added: map[string][]string{"openspec/changes/archive/2026-01-01-demo/kickoff.yaml": {"s1"}}}
	odd := &fakeHistory{added: map[string][]string{"odd/tasks/a.md": {"o1"}}}
	cands := []candidate{sddCand("demo", "2026-01-01-demo"), {Increment: Increment{Lane: LaneODD, Name: "a", Path: "odd/tasks/a.md", Role: RoleModified}}}
	incs, _, err := anchorCandidates(context.Background(), "main", Source{History: specs, GitPrefix: "openspec"}, Source{History: odd, GitPrefix: "odd/tasks"}, cands)
	if err != nil || incs[0].Integration.Commit != "s1" || incs[1].Integration.Commit != "o1" {
		t.Fatalf("incs=%+v err=%v", incs, err)
	}
	for _, c := range specs.calls {
		if strings.Contains(c, "odd/tasks") {
			t.Fatalf("la historia de specs recibió una ruta ODD: %v", specs.calls)
		}
	}
	for _, c := range odd.calls {
		if strings.Contains(c, "openspec") {
			t.Fatalf("la historia ODD recibió una ruta de specs: %v", odd.calls)
		}
	}
}

func TestRequirementTraceCheckRefKeepsCause(t *testing.T) {
	h := &fakeHistory{badRef: "nope"}
	src := Source{Files: nilFS{}, History: h, GitPrefix: "openspec", Repo: "/r/specs"}
	err := checkRef(context.Background(), "nope", LaneSDD, src, "openspec/changes")
	var refErr *RefError
	if !errors.As(err, &refErr) || refErr.Lane != LaneSDD || refErr.Ref != "nope" || refErr.Repo != "/r/specs" ||
		!strings.Contains(refErr.Error(), "Not a valid object name") || !strings.Contains(refErr.Error(), "/r/specs") {
		t.Fatalf("error = %v", err)
	}
	if err := checkRef(context.Background(), "main", LaneSDD, src, "openspec/changes"); err != nil {
		t.Fatalf("referencia válida: %v", err)
	}
	if err := checkRef(context.Background(), "nope", LaneODD, Source{Files: nilFS{}}, "odd/tasks"); err != nil {
		t.Fatalf("sin historia no se comprueba: %v", err)
	}
	if err := checkRef(context.Background(), "nope", LaneODD, Source{History: h}, "odd/tasks"); err != nil {
		t.Fatalf("fuente ausente no se comprueba: %v", err)
	}
}

func inc(lane Lane, path string, role Role, commit string) Increment {
	return Increment{Lane: lane, Name: path, Path: path, Role: role, Integration: Commit{Commit: commit, Found: commit != ""}}
}

func TestRequirementTraceSortByFirstParentPositionAndStableKey(t *testing.T) {
	h := &fakeHistory{line: []string{"c1", "c2", "c3"}}
	src := Source{History: h, Repo: "r"}
	input := []Increment{
		inc(LaneODD, "odd/tasks/z.md", RoleModified, "c3"),
		inc(LaneSDD, "openspec/changes/archive/2026-01-01-b", RoleCovered, ""),
		inc(LaneSDD, "openspec/changes/archive/2026-01-01-a", RoleCovered, "c2"),
		inc(LaneSDD, "openspec/changes/archive/2026-01-01-a", RoleCreated, "c2"),
		inc(LaneODD, "odd/tasks/y.md", RoleModified, "c1"),
		inc(LaneSDD, "openspec/changes/archive/2026-01-01-c", RoleCreated, ""),
	}
	reversed := append([]Increment(nil), input...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	first, err := sortIncrements(context.Background(), src, src, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := sortIncrements(context.Background(), src, src, reversed)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatalf("el orden depende de la entrada:\n%s\n%s", a, b)
	}
	var got []string
	for _, i := range first {
		got = append(got, i.Path+"|"+string(i.Role))
	}
	want := []string{
		"odd/tasks/y.md|modified",
		"openspec/changes/archive/2026-01-01-a|created",
		"openspec/changes/archive/2026-01-01-a|covered",
		"odd/tasks/z.md|modified",
		"openspec/changes/archive/2026-01-01-b|covered",
		"openspec/changes/archive/2026-01-01-c|created",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orden = %v\nquería %v", got, want)
	}
}

func TestRequirementTraceSortSquashedTogetherTiesByKeyAndSDDBeforeODDByPosition(t *testing.T) {
	h := &fakeHistory{line: []string{"c1", "c2"}}
	src := Source{History: h, Repo: "r"}
	got, err := sortIncrements(context.Background(), src, src, []Increment{
		inc(LaneODD, "odd/tasks/a.md", RoleModified, "c2"),
		inc(LaneSDD, "openspec/changes/archive/2026-01-01-a", RoleCreated, "c1"),
		inc(LaneSDD, "openspec/changes/archive/2026-01-01-b", RoleCreated, "c2"),
	})
	if err != nil || got[0].Lane != LaneSDD || got[1].Lane != LaneODD || got[2].Lane != LaneSDD {
		t.Fatalf("orden = %+v err=%v", got, err)
	}
}

func TestRequirementTraceSortPropagatesGitErrorsAndMemoizes(t *testing.T) {
	h := &fakeHistory{line: []string{"c1", "c2"}, failOn: "c2"}
	src := Source{History: h, Repo: "r"}
	if _, err := sortIncrements(context.Background(), src, src, []Increment{
		inc(LaneSDD, "a", RoleCreated, "c2"), inc(LaneSDD, "b", RoleCreated, "c1"),
	}); err == nil || !strings.Contains(err.Error(), "merge-base") {
		t.Fatalf("error = %v", err)
	}
	h = &fakeHistory{line: []string{"c1", "c2"}}
	src = Source{History: h, Repo: "r"}
	if _, err := sortIncrements(context.Background(), src, src, []Increment{
		inc(LaneSDD, "a", RoleCreated, "c2"), inc(LaneSDD, "b", RoleCreated, "c1"),
		inc(LaneSDD, "c", RoleCreated, "c2"), inc(LaneSDD, "d", RoleCreated, "c1"),
	}); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, call := range h.calls {
		if seen[call] {
			t.Fatalf("comparación repetida %q: falta memoria (%v)", call, h.calls)
		}
		seen[call] = true
	}
}

func TestRequirementTraceSortDifferentRepositoriesCompareByKeyOnly(t *testing.T) {
	specs := &fakeHistory{line: []string{"s1"}}
	odd := &fakeHistory{line: []string{"o1"}}
	got, err := sortIncrements(context.Background(), Source{History: specs, Repo: "specs"}, Source{History: odd, Repo: "odd"}, []Increment{
		inc(LaneSDD, "openspec/changes/archive/2026-01-01-a", RoleCreated, "s1"),
		inc(LaneODD, "odd/tasks/a.md", RoleModified, "o1"),
	})
	if err != nil || got[0].Lane != LaneODD {
		t.Fatalf("orden = %+v err=%v", got, err)
	}
	if len(specs.calls)+len(odd.calls) != 0 {
		t.Fatalf("no debe comparar commits de repositorios distintos: %v %v", specs.calls, odd.calls)
	}
}

func TestRequirementTraceSortWarningsIsStable(t *testing.T) {
	ws := []Warning{{Code: "b", Path: "z"}, {Code: "a", Path: "z"}, {Code: "c", Path: "a"}, {Code: "a"}}
	sortWarnings(ws)
	if ws[0].Code != "a" || ws[0].Path != "" || ws[1].Path != "a" || ws[2].Code != "a" || ws[3].Code != "b" {
		t.Fatalf("advertencias = %+v", ws)
	}
}

// nilFS es un sistema de ficheros vacío que solo sirve para marcar una fuente como presente.
type nilFS struct{}

func (nilFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }
