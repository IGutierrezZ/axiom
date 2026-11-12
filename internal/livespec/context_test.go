package livespec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Las pruebas de la comprobación estructural del contexto técnico (REQ-30.6, D7).
// Amenazas «selección de repositorio git» (rutas relativa, absoluta y `../repo` hermano,
// resueltas contra workspaceRoot) y «estado de commit» (confirmado, en índice, sin
// confirmar y no seguido). Todos los textos son inventados.

func gitIn(t testing.TB, dir string, args ...string) string {
	t.Helper()
	result, err := gitcommon.Run(context.Background(), dir, args...)
	if err != nil {
		t.Fatalf("git %v en %s: %v", args, dir, err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("git %v en %s terminó con %d: %s", args, dir, result.ExitCode, result.Stderr)
	}
	return string(result.Stdout)
}

func isolateGit(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

// newGitTree crea un repositorio con la rama main, los ficheros dados en su primer commit
// y la rama `feat` activa.
func newGitTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "config", "core.autocrlf", "false")
	writeTree(t, root, files)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "base")
	gitIn(t, root, "checkout", "-q", "-b", "feat")
}

func commitAll(t *testing.T, root string) {
	t.Helper()
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "feature")
}

func contextLocation(workspaceRoot string, maintained, reference []string) workspace.Location {
	loc := workspace.Location{WorkspaceRoot: workspaceRoot, PlanningHome: filepath.Join(workspaceRoot, "openspec")}
	if maintained == nil && reference == nil {
		return loc
	}
	config := &workspace.TechnicalContextConfig{}
	if maintained != nil {
		config.Maintained = &workspace.ContextMode{Paths: maintained}
	}
	if reference != nil {
		config.Reference = &workspace.ContextMode{Paths: reference}
	}
	loc.Declarations.TechnicalContext = config
	return loc
}

// useBase sustituye el resolutor de la base por uno fijo y lo restaura al terminar.
func useBase(t *testing.T, ref string) {
	t.Helper()
	previous := repositoryBase
	repositoryBase = func(context.Context, workspace.Location, string, string) (string, error) { return ref, nil }
	t.Cleanup(func() { repositoryBase = previous })
}

const (
	archivo   = "# Arquitectura\n\nTexto.\n"
	referenia = "# Referencia\n\nTexto.\n"
)

func baseFiles() map[string]string {
	return map[string]string{
		"context/ARQUITECTURA.md": archivo,
		"docs/referencia/ADR.md":  referenia,
		"src/main.go":             "package main\n",
	}
}

func TestCheckContextWithoutTechnicalContextProducesNoReport(t *testing.T) {
	previous := repositoryBase
	repositoryBase = func(context.Context, workspace.Location, string, string) (string, error) {
		t.Error("sin technical_context no se resuelve ninguna base")
		return "", nil
	}
	t.Cleanup(func() { repositoryBase = previous })

	report, err := checkContext(context.Background(), contextLocation(t.TempDir(), nil, nil), "mi-feature")
	if err != nil || report != nil {
		t.Errorf("checkContext = %+v, %v; sin technical_context no hay informe ni error", report, err)
	}
}

func TestCheckContextRejectsAReferenceEditInEachCommitState(t *testing.T) {
	states := []struct {
		name string
		edit func(t *testing.T, root string)
	}{
		{"confirmada", func(t *testing.T, root string) {
			writeTree(t, root, map[string]string{"docs/referencia/ADR.md": referenia + "cambio\n"})
			commitAll(t, root)
		}},
		{"en el índice", func(t *testing.T, root string) {
			writeTree(t, root, map[string]string{"docs/referencia/ADR.md": referenia + "cambio\n"})
			gitIn(t, root, "add", "-A")
		}},
		{"sin confirmar", func(t *testing.T, root string) {
			writeTree(t, root, map[string]string{"docs/referencia/ADR.md": referenia + "cambio\n"})
		}},
		{"no seguida", func(t *testing.T, root string) {
			writeTree(t, root, map[string]string{"docs/referencia/NUEVO.md": "# nuevo\n"})
		}},
	}
	for _, state := range states {
		t.Run(state.name, func(t *testing.T) {
			isolateGit(t)
			root := t.TempDir()
			newGitTree(t, root, baseFiles())
			useBase(t, "main")
			state.edit(t, root)

			_, err := checkContext(context.Background(), contextLocation(root, []string{"context/"}, []string{"docs/referencia/"}), "mi-feature")
			var refErr *ReferencePathError
			if !errors.As(err, &refErr) || len(refErr.Files) != 1 || !strings.HasPrefix(refErr.Files[0], "docs/referencia/") {
				t.Fatalf("error = %v, se esperaba *ReferencePathError con el fichero de referencia", err)
			}
			for _, want := range []string{refErr.Files[0], "reference", "axiom.yaml"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el rechazo debe nombrar %q: %v", want, err)
				}
			}
		})
	}
}

func TestCheckContextNamesEveryReferenceFileIncludingDeletionsAndRenames(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	useBase(t, "main")
	if err := os.Remove(filepath.Join(root, "docs", "referencia", "ADR.md")); err != nil {
		t.Fatal(err)
	}
	writeTree(t, root, map[string]string{"docs/referencia/OTRO.md": "# otro\n"})

	_, err := checkContext(context.Background(), contextLocation(root, []string{"context/"}, []string{"docs/referencia/"}), "mi-feature")
	var refErr *ReferencePathError
	if !errors.As(err, &refErr) {
		t.Fatalf("error = %v", err)
	}
	want := []string{"docs/referencia/ADR.md", "docs/referencia/OTRO.md"}
	if !reflect.DeepEqual(refErr.Files, want) {
		t.Errorf("Files = %v, se esperaba %v", refErr.Files, want)
	}
}

func TestCheckContextReportsMaintainedEditsWithoutBlocking(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	useBase(t, "main")
	writeTree(t, root, map[string]string{"context/ARQUITECTURA.md": archivo + "cambio\n", "src/main.go": "package main\n// cambio\n"})
	commitAll(t, root)

	report, err := checkContext(context.Background(), contextLocation(root, []string{"context/"}, []string{"docs/referencia/"}), "mi-feature")
	if err != nil {
		t.Fatalf("checkContext: %v", err)
	}
	if report == nil || !report.Declared || !reflect.DeepEqual(report.MaintainedTouched, []string{"context/ARQUITECTURA.md"}) {
		t.Errorf("report = %+v", report)
	}
	if report.ReferenceTouched == nil || len(report.ReferenceTouched) != 0 {
		t.Errorf("ReferenceTouched = %#v, debe ser una lista vacía no nula", report.ReferenceTouched)
	}
}

func TestCheckContextReportsNoMaintainedPathTouched(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	useBase(t, "main")
	writeTree(t, root, map[string]string{"src/main.go": "package main\n// cambio\n"})
	commitAll(t, root)

	report, err := checkContext(context.Background(), contextLocation(root, []string{"context/"}, []string{"docs/referencia/"}), "mi-feature")
	if err != nil || report == nil || !report.Declared {
		t.Fatalf("checkContext = %+v, %v", report, err)
	}
	if report.MaintainedTouched == nil || len(report.MaintainedTouched) != 0 {
		t.Errorf("MaintainedTouched = %#v, debe ser una lista vacía no nula", report.MaintainedTouched)
	}
}

func TestCheckContextFailsClosedWhenTheBaseCannotBeResolved(t *testing.T) {
	tests := []struct {
		name     string
		resolver func(context.Context, workspace.Location, string, string) (string, error)
	}{
		{"el resolutor falla", func(context.Context, workspace.Location, string, string) (string, error) {
			return "", errors.New("el repositorio no forma parte del plan de la feature")
		}},
		{"la referencia no existe", func(context.Context, workspace.Location, string, string) (string, error) {
			return "origin/inexistente", nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGit(t)
			root := t.TempDir()
			newGitTree(t, root, baseFiles())
			previous := repositoryBase
			repositoryBase = tt.resolver
			t.Cleanup(func() { repositoryBase = previous })

			_, err := checkContext(context.Background(), contextLocation(root, []string{"context/"}, []string{"docs/referencia/"}), "mi-feature")
			var unresolved *BaseUnresolvedError
			if !errors.As(err, &unresolved) {
				t.Fatalf("error = %v, se esperaba *BaseUnresolvedError", err)
			}
			if unresolved.Repository == "" || !strings.Contains(err.Error(), "axiom branch") {
				t.Errorf("el rechazo debe nombrar el repositorio y una continuación: %v", err)
			}
		})
	}
}

func TestCheckContextResolvesRelativeAbsoluteAndSiblingPathsAgainstTheWorkspaceRoot(t *testing.T) {
	isolateGit(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "specs")
	sibling := filepath.Join(parent, "backend")
	newGitTree(t, root, baseFiles())
	newGitTree(t, sibling, map[string]string{"docs/API.md": "# API\n", "docs/ref/ADR.md": "# ADR\n", "x.txt": "x\n"})
	useBase(t, "main")

	// El directorio del proceso no es ninguno de los dos repositorios.
	t.Chdir(t.TempDir())

	writeTree(t, sibling, map[string]string{"docs/API.md": "# API\n\ncambio\n"})
	commitAll(t, sibling)

	t.Run("hermano relativo mantenido", func(t *testing.T) {
		report, err := checkContext(context.Background(), contextLocation(root, []string{"context/", "../backend/docs/"}, []string{"docs/referencia/"}), "mi-feature")
		if err != nil {
			t.Fatalf("checkContext: %v", err)
		}
		if !reflect.DeepEqual(report.MaintainedTouched, []string{"../backend/docs/API.md"}) {
			t.Errorf("MaintainedTouched = %v", report.MaintainedTouched)
		}
	})

	t.Run("absoluta mantenida", func(t *testing.T) {
		report, err := checkContext(context.Background(), contextLocation(root, []string{filepath.Join(sibling, "docs")}, []string{"docs/referencia/"}), "mi-feature")
		if err != nil {
			t.Fatalf("checkContext: %v", err)
		}
		if !reflect.DeepEqual(report.MaintainedTouched, []string{"../backend/docs/API.md"}) {
			t.Errorf("MaintainedTouched = %v", report.MaintainedTouched)
		}
	})

	t.Run("referencia en un repositorio hermano", func(t *testing.T) {
		writeTree(t, sibling, map[string]string{"docs/ref/ADR.md": "# ADR\n\ncambio\n"})
		_, err := checkContext(context.Background(), contextLocation(root, []string{"context/"}, []string{"../backend/docs/ref/"}), "mi-feature")
		var refErr *ReferencePathError
		if !errors.As(err, &refErr) || !reflect.DeepEqual(refErr.Files, []string{"../backend/docs/ref/ADR.md"}) {
			t.Fatalf("error = %v, se esperaba el fichero hermano en *ReferencePathError", err)
		}
	})
}

func TestBaseOfPrefersTheRemoteTrackingTargetThenTheLocalOne(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	repo := branch.RepositoryResult{
		CurrentBranch:     "feat",
		IntegrationBranch: "main",
		Branches:          []branch.BranchResult{{Name: "feat", PRTarget: "main"}},
	}

	got, err := baseOf(context.Background(), root, repo)
	if err != nil || got != "main" {
		t.Errorf("baseOf = %q, %v; sin origin/main se usa la rama local main", got, err)
	}

	gitIn(t, root, "update-ref", "refs/remotes/origin/main", "main")
	got, err = baseOf(context.Background(), root, repo)
	if err != nil || got != "origin/main" {
		t.Errorf("baseOf = %q, %v; con origin/main se prefiere la referencia remota", got, err)
	}

	repo.Branches = nil
	got, err = baseOf(context.Background(), root, repo)
	if err != nil || got != "origin/main" {
		t.Errorf("baseOf = %q, %v; sin rama de feature se usa la rama de integración", got, err)
	}

	repo.Branches = []branch.BranchResult{{Name: "feat", PRTarget: "no-existe"}}
	if got, err = baseOf(context.Background(), root, repo); err == nil {
		t.Errorf("baseOf = %q; un destino que no existe en local no es una base", got)
	}
}

func TestBaseOfUsesTheUpstreamWhenHeadIsAlreadyTheIntegrationBranch(t *testing.T) {
	isolateGit(t)
	remote := t.TempDir()
	gitIn(t, remote, "init", "-q", "--bare", "-b", "main")
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	gitIn(t, root, "checkout", "-q", "main")
	gitIn(t, root, "remote", "add", "origin", remote)
	gitIn(t, root, "push", "-q", "-u", "origin", "main")
	useBase(t, "@{upstream}")

	repo := branch.RepositoryResult{CurrentBranch: "main", IntegrationBranch: "main"}
	base, err := baseOf(context.Background(), root, repo)
	if err != nil || base != "@{upstream}" {
		t.Fatalf("baseOf = %q, %v; con HEAD en la rama de integración la base es @{upstream}", base, err)
	}

	// Un commit sin publicar que edita una referencia se ve contra el upstream.
	writeTree(t, root, map[string]string{"docs/referencia/ADR.md": referenia + "cambio\n"})
	commitAll(t, root)
	_, err = checkContext(context.Background(), contextLocation(root, []string{"context/"}, []string{"docs/referencia/"}), "mi-feature")
	var refErr *ReferencePathError
	if !errors.As(err, &refErr) {
		t.Fatalf("error = %v, se esperaba *ReferencePathError", err)
	}
}

func TestBaseOfWithoutUpstreamFailsClosed(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	gitIn(t, root, "checkout", "-q", "main")

	repo := branch.RepositoryResult{CurrentBranch: "main", IntegrationBranch: "main"}
	_, err := baseOf(context.Background(), root, repo)
	var unresolved *BaseUnresolvedError
	if !errors.As(err, &unresolved) {
		t.Fatalf("error = %v, se esperaba *BaseUnresolvedError: sin upstream no hay base", err)
	}
}

func TestPlanIntegrationWritesNothingWhenTheContextCheckFails(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	useBase(t, "main")
	writeTree(t, root, map[string]string{"docs/referencia/ADR.md": referenia + "cambio\n"})
	loc := contextLocation(root, []string{"context/"}, []string{"docs/referencia/"})
	reg, err := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta", "REQ-31.1", "El sistema DEBE dar de alta.")

	_, err = PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	var refErr *ReferencePathError
	if !errors.As(err, &refErr) {
		t.Fatalf("PlanIntegration = %v, se esperaba *ReferencePathError", err)
	}
	if _, statErr := os.Stat(filepath.Join(loc.PlanningHome, "specs")); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("no se escribe nada si falla el contexto (%v)", statErr)
	}
}

func TestPlanIntegrationCarriesTheContextReport(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	newGitTree(t, root, baseFiles())
	useBase(t, "main")
	writeTree(t, root, map[string]string{"context/ARQUITECTURA.md": archivo + "cambio\n"})
	loc := contextLocation(root, []string{"context/"}, []string{"docs/referencia/"})
	reg, err := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta", "REQ-31.1", "El sistema DEBE dar de alta.")

	plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	if plan.Context == nil || !reflect.DeepEqual(plan.Context.MaintainedTouched, []string{"context/ARQUITECTURA.md"}) {
		t.Errorf("Context = %+v", plan.Context)
	}
	if got := plan.Report(nil).Context; got != plan.Context {
		t.Errorf("el informe debe llevar el informe de contexto: %+v", got)
	}
}

func TestPlanIntegrationRejectsABugWithoutImpactBeforeCheckingTheContext(t *testing.T) {
	previous := repositoryBase
	repositoryBase = func(context.Context, workspace.Location, string, string) (string, error) {
		t.Error("el rechazo por impacto precede a la comprobación del contexto")
		return "", nil
	}
	t.Cleanup(func() { repositoryBase = previous })
	loc := contextLocation(t.TempDir(), []string{"context/"}, nil)
	reg, err := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = PlanIntegration(loc, parseOK(t, featureDoc("Tipo: bug\n\n", "")), reg)
	var docErr *DocError
	if !errors.As(err, &docErr) {
		t.Fatalf("PlanIntegration = %v, se esperaba *DocError", err)
	}
}
