package isoevidence

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
)

// fakeCommit es un commit de la línea --first-parent con el árbol completo que
// deja.
type fakeCommit struct {
	id    string
	files []string
}

// fakeHistory simula un repositorio: la línea --first-parent (del más antiguo
// al más reciente) y, para los commits de merge, los ancestros que quedan fuera
// de esa línea.
type fakeHistory struct {
	line      []fakeCommit
	ancestors map[string][]string // descendiente → ancestros que no están en la línea
	err       error
	ancestorE error
	calls     int
}

func (f *fakeHistory) tree(commit string) map[string]bool {
	for _, c := range f.line {
		if c.id == commit {
			set := map[string]bool{}
			for _, file := range c.files {
				set[file] = true
			}
			return set
		}
	}
	return nil
}

func (f *fakeHistory) resolve(ref string) string {
	if ref == "main" && len(f.line) > 0 {
		return f.line[len(f.line)-1].id
	}
	return ref
}

func under(files map[string]bool, path string) bool {
	if files[path] {
		return true
	}
	for file := range files {
		if strings.HasPrefix(file, path+"/") {
			return true
		}
	}
	return false
}

func (f *fakeHistory) FirstParentAdded(_ context.Context, ref, path string) ([]string, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	head := f.resolve(ref)
	var out []string
	prev := map[string]bool{}
	for _, c := range f.line {
		cur := f.tree(c.id)
		if under(cur, path) && !under(prev, path) {
			out = append(out, c.id)
		}
		prev = cur
		if c.id == head {
			break
		}
	}
	return out, nil
}

func (f *fakeHistory) TreeHas(_ context.Context, commit, path string) (bool, error) {
	return under(f.tree(f.resolve(commit)), path), f.err
}

func (f *fakeHistory) TreeEntries(_ context.Context, ref, dir string) ([]string, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	set := map[string]bool{}
	for file := range f.tree(f.resolve(ref)) {
		if rest, ok := strings.CutPrefix(file, dir+"/"); ok {
			set[strings.SplitN(rest, "/", 2)[0]] = true
		}
	}
	var out []string
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (f *fakeHistory) IsAncestor(_ context.Context, ancestor, descendant string) (bool, error) {
	if f.ancestorE != nil {
		return false, f.ancestorE
	}
	if ancestor == descendant {
		return true, nil
	}
	for _, a := range f.ancestors[descendant] {
		if a == ancestor {
			return true, nil
		}
	}
	// Dentro de la línea, un commit anterior es ancestro de uno posterior.
	ai, di := -1, -1
	for i, c := range f.line {
		if c.id == ancestor {
			ai = i
		}
		if c.id == descendant {
			di = i
		}
	}
	return ai >= 0 && di >= 0 && ai < di, nil
}

const (
	codeDir    = "openspec/changes/pagos-iva/"
	archiveDir = "openspec/changes/archive/2026-11-12-pagos-iva/"
)

func codeFiles(extra ...string) []string {
	files := []string{codeDir + "tasks.md", codeDir + "verify-report.md", codeDir + "kickoff.yaml"}
	return append(files, extra...)
}

func archiveFiles() []string {
	return []string{
		archiveDir + "tasks.md", archiveDir + "verify-report.md", archiveDir + "kickoff.yaml",
		archiveDir + "archive-report.md",
	}
}

var baseFile = []string{"openspec/specs/pagos/spec.md"}

func input(evidence string) TraceInput {
	return TraceInput{Change: "pagos-iva", Ref: "main", PlanningHomePrefix: "openspec", EvidenceCommit: evidence}
}

func TestDeriveTrace(t *testing.T) {
	squash := &fakeHistory{line: []fakeCommit{
		{"c1", baseFile},
		{"squash-code", append(append([]string{}, baseFile...), codeFiles()...)},
		{"squash-archive", append(append([]string{}, baseFile...), archiveFiles()...)},
	}}
	merge := &fakeHistory{
		line: []fakeCommit{
			{"c1", baseFile},
			{"merge-code", append(append([]string{}, baseFile...), codeFiles()...)},
			{"merge-archive", append(append([]string{}, baseFile...), archiveFiles()...)},
		},
		ancestors: map[string][]string{"merge-code": {"side-1", "side-2"}, "merge-archive": {"side-3"}},
	}
	rebase := &fakeHistory{line: []fakeCommit{
		{"c1", baseFile},
		{"r1", append(append([]string{}, baseFile...), codeDir+"tasks.md")},
		{"r2", append(append([]string{}, baseFile...), codeDir+"tasks.md", codeDir+"verify-report.md", codeDir+"kickoff.yaml")},
		{"r3", append(append([]string{}, baseFile...), codeDir+"tasks.md", codeDir+"verify-report.md", codeDir+"kickoff.yaml", "src/otro.go")},
		{"a1", append(append([]string{}, baseFile...), "src/otro.go", archiveDir+"tasks.md", archiveDir+"verify-report.md", archiveDir+"kickoff.yaml")},
		{"a2", append(append([]string{}, baseFile...), "src/otro.go", archiveDir+"tasks.md", archiveDir+"verify-report.md", archiveDir+"kickoff.yaml", archiveDir+"archive-report.md")},
	}}

	cases := []struct {
		name        string
		repo        *fakeHistory
		evidence    string
		wantCode    string
		wantArchive string
		wantFolder  string
		wantDiscrep string // fragmento; vacío = ninguna
	}{
		{"squash", squash, "", "squash-code", "squash-archive", "2026-11-12-pagos-iva", ""},
		{"commit de merge", merge, "", "merge-code", "merge-archive", "2026-11-12-pagos-iva", ""},
		{"rebase con commits atómicos", rebase, "", "r2", "a1", "2026-11-12-pagos-iva", ""},
		{"evidencia igual al commit de código", squash, "squash-code", "squash-code", "squash-archive", "2026-11-12-pagos-iva", ""},
		{"evidencia descendiente del commit de código (último de la serie rebasada)", rebase, "r3", "r2", "a1", "2026-11-12-pagos-iva", ""},
		{"evidencia que no es descendiente: discrepancia sin error", rebase, "c1", "r2", "a1", "2026-11-12-pagos-iva", "r2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DeriveTrace(context.Background(), tc.repo, input(tc.evidence))
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got.CodeCommit != (CommitRef{Commit: tc.wantCode, Found: true}) {
				t.Errorf("commit de código = %+v, quería %s", got.CodeCommit, tc.wantCode)
			}
			if got.ArchiveCommit != (CommitRef{Commit: tc.wantArchive, Found: true}) {
				t.Errorf("commit de archivado = %+v, quería %s", got.ArchiveCommit, tc.wantArchive)
			}
			if got.ArchiveFolder != tc.wantFolder {
				t.Errorf("carpeta archivada = %q, quería %q", got.ArchiveFolder, tc.wantFolder)
			}
			switch {
			case tc.wantDiscrep == "" && got.Discrepancy != "":
				t.Errorf("no debía haber discrepancia: %q", got.Discrepancy)
			case tc.wantDiscrep != "":
				if !strings.Contains(got.Discrepancy, tc.wantDiscrep) || !strings.Contains(got.Discrepancy, tc.evidence) {
					t.Errorf("la discrepancia %q debe nombrar ambos commits (%s y %s)", got.Discrepancy, tc.wantDiscrep, tc.evidence)
				}
			}
		})
	}

	t.Run("las tres estrategias dan la misma cadena", func(t *testing.T) {
		for _, repo := range []*fakeHistory{squash, merge, rebase} {
			got, err := DeriveTrace(context.Background(), repo, input(""))
			if err != nil || !got.CodeCommit.Found || !got.ArchiveCommit.Found || got.ArchiveFolder == "" {
				t.Fatalf("cadena incompleta: %+v %v", got, err)
			}
		}
	})
}

func TestDeriveTraceAbsences(t *testing.T) {
	t.Run("sin carpeta archivada todavía", func(t *testing.T) {
		repo := &fakeHistory{line: []fakeCommit{
			{"c1", baseFile},
			{"code", append(append([]string{}, baseFile...), codeFiles()...)},
		}}
		got, err := DeriveTrace(context.Background(), repo, input(""))
		if err != nil {
			t.Fatal(err)
		}
		if !got.CodeCommit.Found || got.ArchiveCommit.Found || got.ArchiveFolder != "" {
			t.Fatalf("traza = %+v", got)
		}
	})

	t.Run("sin carpeta del cambio en la rama", func(t *testing.T) {
		repo := &fakeHistory{line: []fakeCommit{{"c1", baseFile}}}
		got, err := DeriveTrace(context.Background(), repo, input(""))
		if err != nil {
			t.Fatal(err)
		}
		if got.CodeCommit.Found || got.ArchiveCommit.Found || got.Discrepancy != "" {
			t.Fatalf("traza = %+v", got)
		}
	})

	t.Run("verify-report sin tasks no cuenta como commit de código", func(t *testing.T) {
		repo := &fakeHistory{line: []fakeCommit{
			{"c1", baseFile},
			{"solo-verify", append(append([]string{}, baseFile...), codeDir+"verify-report.md")},
		}}
		got, err := DeriveTrace(context.Background(), repo, input(""))
		if err != nil || got.CodeCommit.Found {
			t.Fatalf("traza = %+v, err = %v", got, err)
		}
	})

	t.Run("los ficheros de tareas por rol cuentan", func(t *testing.T) {
		repo := &fakeHistory{line: []fakeCommit{
			{"c1", baseFile},
			{"roles", append(append([]string{}, baseFile...), codeDir+"tasks.core.md", codeDir+"tasks.web.md", codeDir+"verify-report.md")},
		}}
		got, err := DeriveTrace(context.Background(), repo, input(""))
		if err != nil || got.CodeCommit != (CommitRef{Commit: "roles", Found: true}) {
			t.Fatalf("traza = %+v, err = %v", got, err)
		}
	})

	t.Run("una carpeta archivada de otro cambio no cuenta", func(t *testing.T) {
		repo := &fakeHistory{line: []fakeCommit{
			{"c1", baseFile},
			{"otro", append(append([]string{}, baseFile...), "openspec/changes/archive/2026-11-12-pagos-iva-extra/kickoff.yaml", "openspec/changes/archive/2026-11-12x-pagos-iva/kickoff.yaml")},
		}}
		got, err := DeriveTrace(context.Background(), repo, input(""))
		if err != nil || got.ArchiveFolder != "" || got.ArchiveCommit.Found {
			t.Fatalf("traza = %+v, err = %v", got, err)
		}
	})

	t.Run("con dos carpetas archivadas se toma la más reciente", func(t *testing.T) {
		repo := &fakeHistory{line: []fakeCommit{
			{"c1", baseFile},
			{"viejo", append(append([]string{}, baseFile...), "openspec/changes/archive/2026-01-01-pagos-iva/kickoff.yaml")},
			{"nuevo", append(append([]string{}, baseFile...), "openspec/changes/archive/2026-01-01-pagos-iva/kickoff.yaml", "openspec/changes/archive/2026-11-12-pagos-iva/kickoff.yaml")},
		}}
		got, err := DeriveTrace(context.Background(), repo, input(""))
		if err != nil || got.ArchiveFolder != "2026-11-12-pagos-iva" || got.ArchiveCommit.Commit != "nuevo" {
			t.Fatalf("traza = %+v, err = %v", got, err)
		}
	})

	t.Run("sin kickoff.yaml en la carpeta archivada se usa la carpeta", func(t *testing.T) {
		repo := &fakeHistory{line: []fakeCommit{
			{"c1", baseFile},
			{"archivado", append(append([]string{}, baseFile...), archiveDir+"tasks.md")},
		}}
		got, err := DeriveTrace(context.Background(), repo, input(""))
		if err != nil || got.ArchiveCommit != (CommitRef{Commit: "archivado", Found: true}) {
			t.Fatalf("traza = %+v, err = %v", got, err)
		}
	})
}

func TestDeriveTraceEvidenceWithoutCode(t *testing.T) {
	repo := &fakeHistory{line: []fakeCommit{{"c1", baseFile}}}
	got, err := DeriveTrace(context.Background(), repo, input("deadbeef"))
	if err != nil {
		t.Fatalf("la ausencia del commit de código no es un error: %v", err)
	}
	if !strings.Contains(got.Discrepancy, "deadbeef") {
		t.Fatalf("debía informar la evidencia sin commit de código: %q", got.Discrepancy)
	}
}

func TestDeriveTraceContrastFailureIsADiscrepancy(t *testing.T) {
	repo := &fakeHistory{
		line:      []fakeCommit{{"code", append(append([]string{}, baseFile...), codeFiles()...)}},
		ancestorE: errors.New("commit desconocido"),
	}
	got, err := DeriveTrace(context.Background(), repo, input("ffff"))
	if err != nil {
		t.Fatalf("un fallo al contrastar no debe ser un error de ejecución: %v", err)
	}
	if !strings.Contains(got.Discrepancy, "commit desconocido") || !strings.Contains(got.Discrepancy, "code") {
		t.Fatalf("discrepancia = %q", got.Discrepancy)
	}
}

func TestDeriveTraceHistoryErrorPropagates(t *testing.T) {
	repo := &fakeHistory{err: errors.New("git no disponible")}
	if _, err := DeriveTrace(context.Background(), repo, input("")); err == nil || !strings.Contains(err.Error(), "git no disponible") {
		t.Fatalf("un fallo del historial debía propagarse: %v", err)
	}
}

func TestDeriveTraceTrailersAreNotRead(t *testing.T) {
	// El puerto no ofrece ningún acceso a los mensajes de commit: los trailers
	// intermedios nunca se leen ni se validan.
	repo := &fakeHistory{line: []fakeCommit{
		{"c1", baseFile},
		{"sin-trailers", append(append([]string{}, baseFile...), codeFiles()...)},
	}}
	if _, err := DeriveTrace(context.Background(), repo, input("")); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveTraceRejectsUnsafeInput(t *testing.T) {
	repo := &fakeHistory{line: []fakeCommit{{"c1", baseFile}}}
	cases := []struct {
		name string
		in   TraceInput
	}{
		{"cambio vacío", TraceInput{Ref: "main", PlanningHomePrefix: "openspec"}},
		{"cambio con barra", TraceInput{Change: "a/b", Ref: "main", PlanningHomePrefix: "openspec"}},
		{"cambio con punto punto", TraceInput{Change: "..", Ref: "main", PlanningHomePrefix: "openspec"}},
		{"cambio con pathspec mágico", TraceInput{Change: ":(top)x", Ref: "main", PlanningHomePrefix: "openspec"}},
		{"cambio con comodín", TraceInput{Change: "a*", Ref: "main", PlanningHomePrefix: "openspec"}},
		{"cambio que parece una opción", TraceInput{Change: "-x", Ref: "main", PlanningHomePrefix: "openspec"}},
		{"referencia vacía", TraceInput{Change: "x", PlanningHomePrefix: "openspec"}},
		{"referencia con punto punto", TraceInput{Change: "x", Ref: "a..b", PlanningHomePrefix: "openspec"}},
		{"referencia que parece una opción", TraceInput{Change: "x", Ref: "--all", PlanningHomePrefix: "openspec"}},
		{"planning home absoluto", TraceInput{Change: "x", Ref: "main", PlanningHomePrefix: "/etc"}},
		{"planning home con punto punto", TraceInput{Change: "x", Ref: "main", PlanningHomePrefix: "../fuera"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := repo.calls
			_, err := DeriveTrace(context.Background(), repo, tc.in)
			if !errors.Is(err, ErrInvalidTraceInput) {
				t.Fatalf("error = %v, quería ErrInvalidTraceInput", err)
			}
			if repo.calls != before {
				t.Fatal("la entrada no válida debía rechazarse antes de consultar el historial")
			}
		})
	}
}

func TestDeriveTraceEmptyPlanningHome(t *testing.T) {
	repo := &fakeHistory{line: []fakeCommit{
		{"c1", []string{"changes/x/tasks.md", "changes/x/verify-report.md"}},
	}}
	got, err := DeriveTrace(context.Background(), repo, TraceInput{Change: "x", Ref: "main"})
	if err != nil || got.CodeCommit != (CommitRef{Commit: "c1", Found: true}) {
		t.Fatalf("traza = %+v, err = %v", got, err)
	}
}
