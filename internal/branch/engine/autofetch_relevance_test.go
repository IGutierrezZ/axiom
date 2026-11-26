package engine

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de la relevancia del autofetch (RF-BRA-39 paso 3, design D-07): cambio actual por sus
// cuatro reglas y avisos (a) y (b) calculados antes del avance, con git real y remotos desnudos
// locales. Ninguna es paralela: comparten el observador global de gitcommon.

var relevanceClock = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

// relevanceProject es un proyecto embebido con `origin` y la rama `main` publicada.
func relevanceProject(t *testing.T) (repo, remote string) {
	t.Helper()
	repo = embeddedCoreProject(t)
	remote = remoteWithMain(t, repo)
	return repo, remote
}

// registerChange guarda el registro de un cambio activo con las ramas dadas en el repositorio.
func registerChange(t *testing.T, repo string, lane branch.Lane, change string, branches ...string) {
	t.Helper()
	commonDir, err := gitcommon.CommonDirFromFiles(repo)
	if err != nil {
		t.Fatal(err)
	}
	record := branch.Record{
		Schema: branch.RegistrySchema, Lane: lane, Change: change, Mode: branch.ModeInPlace, ModeSource: branch.ModeSourceDefault,
		PlanningRepo: repo,
		Repositories: []branch.RecordRepo{{Name: filepath.Base(repo), GitRoot: repo, CommonDir: commonDir, Role: branch.RepoSpecsInPlace}},
	}
	for _, name := range branches {
		record.Repositories[0].Branches = append(record.Repositories[0].Branches, branch.RecordBranch{Name: name, Kind: branch.KindChange})
	}
	if err := registry.New().Save(commonDir, record); err != nil {
		t.Fatal(err)
	}
}

// publishFromClone publica en el remoto un commit nuevo de `target` desde otro clon (la rama nace
// de origin/main si no existe) y devuelve su SHA. Con merge, el commit entra en `main` por una
// fusión sin avance rápido.
func publishFromClone(t *testing.T, remote, target string, files map[string]string, message string) string {
	t.Helper()
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	runGit(t, scratch, "clone", "-q", remote, clone)
	if target != "main" {
		start := "origin/main"
		if strings.Contains(runGit(t, clone, "branch", "-r", "--list", "origin/"+target), "origin/"+target) {
			start = "origin/" + target
		}
		runGit(t, clone, "checkout", "-q", "-B", target, start)
	}
	for name, content := range files {
		writeFile(t, filepath.Join(clone, filepath.FromSlash(name)), content)
	}
	runGit(t, clone, "add", "-A")
	commit(t, clone, message)
	runGit(t, clone, "push", "-q", "origin", target)
	return runGit(t, clone, "rev-parse", "HEAD")
}

func noticesOf(result branch.AutofetchResult, kind string) []branch.AutofetchNotice {
	var found []branch.AutofetchNotice
	for _, notice := range result.Notices {
		if notice.Kind == kind {
			found = append(found, notice)
		}
	}
	return found
}

// RF-BRA-39 paso 3, regla 1: el cambio actual es el registrado que tiene la rama de `HEAD`.
func TestAutofetchRelevanceCurrentChangeByHead(t *testing.T) {
	repo, _ := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	registerChange(t, repo, branch.LaneSDD, "y", "feat/y")
	runGit(t, repo, "checkout", "-q", "-b", "feat/y")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if result.Change == nil || result.Change.Name != "y" || result.Change.Lane != branch.LaneSDD {
		t.Errorf("cambio actual %+v, se esperaba y por la rama de HEAD", result.Change)
	}
}

// Regla 2: un único cambio activo es el actual aunque HEAD no esté en sus ramas; con varios y sin
// coincidencia no hay cambio actual y no hay avisos (a) ni (b), pero el ciclo sigue.
func TestAutofetchRelevanceSingleActiveOrAmbiguous(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	publishFromClone(t, remote, "main", map[string]string{"openspec/changes/x/proposal.md": "p\n"}, "propuesta")

	single := autofetchRun(t, autofetchOn, repo, relevanceClock)
	if single.Change == nil || single.Change.Name != "x" || len(noticesOf(single, branch.AutofetchNoticeIncomingRelevant)) != 1 {
		t.Fatalf("un único cambio activo debía ser el actual con su aviso (a): %+v", single)
	}

	registerChange(t, repo, branch.LaneODD, "z", "feat/z")
	several := autofetchRun(t, autofetchOn, repo, relevanceClock.Add(autofetchInterval))
	if several.Change != nil || len(several.Notices) != 0 || several.Fetch != branch.AutofetchFetchDone {
		t.Errorf("varios cambios sin coincidencia: %+v, se esperaba sin cambio actual ni avisos y con el ciclo hecho", several)
	}
}

// Aviso (a): se calcula antes del avance, cuenta solo commits no de fusión bajo los prefijos del
// cambio y de sus capacidades, e ignora lo ajeno; la orden usa el SHA local previo.
func TestAutofetchRelevanceIncomingCommitsByPrefixes(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	// La capacidad del cambio existe en la rama del cambio: `openspec/specs/cap/` es relevante.
	runGit(t, repo, "branch", "feat/x")
	runGit(t, repo, "checkout", "-q", "feat/x")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "specs", "cap", "spec.md"), "d\n")
	runGit(t, repo, "add", "-A")
	commit(t, repo, "delta")
	runGit(t, repo, "checkout", "-q", "main")
	localMain := runGit(t, repo, "rev-parse", "refs/heads/main")

	publishFromClone(t, remote, "main", map[string]string{"openspec/changes/x/design.md": "d\n"}, "diseño de x")
	publishFromClone(t, remote, "main", map[string]string{"openspec/specs/cap/spec.md": "s\n"}, "spec viva")
	publishFromClone(t, remote, "main", map[string]string{"openspec/changes/otro/design.md": "o\n"}, "ajeno")
	publishFromClone(t, remote, "main", map[string]string{"README.md": "r\n"}, "irrelevante")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	incoming := noticesOf(result, branch.AutofetchNoticeIncomingRelevant)
	if len(incoming) != 1 {
		t.Fatalf("se esperaba un único aviso (a): %+v", result.Notices)
	}
	notice := incoming[0]
	if notice.Branch != "origin/main" || notice.Commits != 2 || !slices.Equal(notice.Scope, []string{"openspec/changes/x", "openspec/specs/cap"}) {
		t.Errorf("aviso (a) %+v, se esperaban 2 commits en origin/main con el alcance de x y de cap", notice)
	}
	wantCommand := "git -C " + pathquote.Quote(result.Repository) + " log --oneline " + localMain + "..origin/main -- \"openspec/changes/x\" \"openspec/specs/cap\""
	if notice.Command != wantCommand {
		t.Errorf("orden %q, se esperaba %q", notice.Command, wantCommand)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/main"); got != localMain {
		t.Errorf("la relevancia no avanza la integración: main pasó de %s a %s", localMain, got)
	}
}

// Las fusiones no cuentan: el commit de la fusión no trae rutas y sus padres ya están en el rango.
func TestAutofetchRelevanceIgnoresMergeCommits(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	runGit(t, scratch, "clone", "-q", remote, clone)
	runGit(t, clone, "checkout", "-q", "-b", "lateral")
	writeFile(t, filepath.Join(clone, "openspec", "changes", "x", "tasks.md"), "t\n")
	runGit(t, clone, "add", "-A")
	commit(t, clone, "tareas")
	runGit(t, clone, "checkout", "-q", "main")
	writeFile(t, filepath.Join(clone, "otro.txt"), "o\n")
	runGit(t, clone, "add", "-A")
	commit(t, clone, "otro")
	runGit(t, clone, "merge", "-q", "--no-ff", "-m", "fusión", "lateral")
	runGit(t, clone, "push", "-q", "origin", "main")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	incoming := noticesOf(result, branch.AutofetchNoticeIncomingRelevant)
	if len(incoming) != 1 || incoming[0].Commits != 1 {
		t.Errorf("avisos %+v, se esperaba un aviso con 1 commit (la fusión no cuenta)", result.Notices)
	}
}

// Cambio ODD: el prefijo es `odd/tasks/<c>.md`.
func TestAutofetchRelevanceOddChangeUsesItsTaskDocument(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneODD, "x", "feat/x")
	publishFromClone(t, remote, "main", map[string]string{"odd/tasks/x.md": "t\n", "odd/tasks/y.md": "y\n"}, "documento odd")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	incoming := noticesOf(result, branch.AutofetchNoticeIncomingRelevant)
	if result.Change == nil || result.Change.Lane != branch.LaneODD || len(incoming) != 1 || !slices.Equal(incoming[0].Scope, []string{"odd/tasks/x.md"}) {
		t.Errorf("resultado %+v, se esperaba el cambio ODD x con alcance odd/tasks/x.md", result)
	}
}

// Sin nada relevante no hay ningún aviso, y sin rama local de integración se omite (a).
func TestAutofetchRelevanceNothingRelevantIsSilent(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	publishFromClone(t, remote, "main", map[string]string{"README.md": "r\n"}, "irrelevante")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if result.Change == nil || len(result.Notices) != 0 {
		t.Errorf("resultado %+v, se esperaba el cambio actual y cero avisos", result)
	}
}

// Aviso (b): la rama del cambio con commits nuevos en la remota avisa con la orden ff-only, o con
// el log si tiene commits propios; la rama local conserva su SHA y no hay avisos sin novedades.
func TestAutofetchRelevanceChangeBranchBehind(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	runGit(t, repo, "branch", "feat/x")
	runGit(t, repo, "push", "-q", "origin", "feat/x")
	local := runGit(t, repo, "rev-parse", "refs/heads/feat/x")

	quiet := autofetchRun(t, autofetchOn, repo, relevanceClock)
	if got := noticesOf(quiet, branch.AutofetchNoticeChangeBranchBehind); len(got) != 0 {
		t.Fatalf("sin novedades no hay aviso (b): %+v", got)
	}

	publishFromClone(t, remote, "feat/x", map[string]string{"a.txt": "a\n"}, "de otra persona")
	behind := autofetchRun(t, autofetchOn, repo, relevanceClock.Add(autofetchInterval))
	got := noticesOf(behind, branch.AutofetchNoticeChangeBranchBehind)
	if len(got) != 1 || got[0].Commits != 1 || got[0].Branch != "feat/x" {
		t.Fatalf("avisos %+v, se esperaba un aviso (b) de feat/x con 1 commit", behind.Notices)
	}
	if want := "git -C " + pathquote.Quote(repo) + " merge --ff-only origin/feat/x"; got[0].Command != want {
		t.Errorf("orden %q, se esperaba %q", got[0].Command, want)
	}
	if sha := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); sha != local {
		t.Errorf("la rama del cambio no se mueve: %s -> %s", local, sha)
	}

	// Con commits propios (divergencia) la orden es el log.
	runGit(t, repo, "checkout", "-q", "feat/x")
	writeFile(t, filepath.Join(repo, "propio.txt"), "p\n")
	runGit(t, repo, "add", "propio.txt")
	commit(t, repo, "propio")
	runGit(t, repo, "checkout", "-q", "main")
	diverged := autofetchRun(t, autofetchOn, repo, relevanceClock.Add(2*autofetchInterval))
	got = noticesOf(diverged, branch.AutofetchNoticeChangeBranchBehind)
	if len(got) != 1 || !strings.Contains(got[0].Command, "log --oneline feat/x..origin/feat/x") {
		t.Errorf("avisos %+v, se esperaba la orden log por la divergencia", diverged.Notices)
	}
}

// Una rama registrada sin rama local pero con referencia remota es una cesión pendiente de
// adoptar: `change_branch_not_adopted` con los commits sobre la integración y `axiom branch ensure`.
func TestAutofetchRelevanceChangeBranchNotAdopted(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	publishFromClone(t, remote, "feat/x", map[string]string{"a.txt": "a\n"}, "cesión")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	got := noticesOf(result, branch.AutofetchNoticeChangeBranchNotAdopted)
	if len(got) != 1 || got[0].Commits != 1 {
		t.Fatalf("avisos %+v, se esperaba change_branch_not_adopted con 1 commit", result.Notices)
	}
	if want := "axiom branch ensure --change x --cwd " + pathquote.Quote(result.Project); got[0].Command != want {
		t.Errorf("orden %q, se esperaba %q", got[0].Command, want)
	}
}

// Regla 3 (hueco 5): sin ningún cambio registrado, una única cesión con `kickoff.yaml` sin rama
// local es el cambio actual; con dos candidatas no hay cambio actual.
func TestAutofetchRelevanceHandoffWithoutRegistry(t *testing.T) {
	repo, remote := relevanceProject(t)
	publishFromClone(t, remote, "feat/x", map[string]string{"openspec/changes/x/kickoff.yaml": "k\n"}, "cesión x")
	publishFromClone(t, remote, "feat/x-archive", map[string]string{"openspec/changes/x-archive/kickoff.yaml": "k\n"}, "archivado")
	publishFromClone(t, remote, "feat/sin-kickoff", map[string]string{"otra.txt": "o\n"}, "sin kickoff")

	one := autofetchRun(t, autofetchOn, repo, relevanceClock)
	if one.Change == nil || one.Change.Name != "x" || one.Change.Lane != branch.LaneSDD || len(noticesOf(one, branch.AutofetchNoticeChangeBranchNotAdopted)) != 1 {
		t.Fatalf("resultado %+v, se esperaba la cesión x como cambio actual con su aviso", one)
	}

	publishFromClone(t, remote, "feat/y", map[string]string{"openspec/changes/y/kickoff.yaml": "k\n"}, "cesión y")
	two := autofetchRun(t, autofetchOn, repo, relevanceClock.Add(autofetchInterval))
	if two.Change != nil || len(two.Notices) != 0 {
		t.Errorf("con dos cesiones: %+v, se esperaba sin cambio actual ni avisos", two)
	}
}

// Una cesión con rama local ya no es candidata: el cambio está adoptado.
func TestAutofetchRelevanceHandoffWithLocalBranchIsNotACandidate(t *testing.T) {
	repo, remote := relevanceProject(t)
	publishFromClone(t, remote, "feat/x", map[string]string{"openspec/changes/x/kickoff.yaml": "k\n"}, "cesión x")
	runGit(t, repo, "fetch", "-q", "origin")
	runGit(t, repo, "branch", "feat/x", "origin/feat/x")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if result.Change != nil || len(result.Notices) != 0 {
		t.Errorf("resultado %+v, se esperaba sin cambio actual", result)
	}
}

// La relevancia solo lee referencias locales: el ciclo lanza un único fetch y nada que publique,
// pode o avance.
func TestAutofetchRelevanceOnlyLocalReads(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	publishFromClone(t, remote, "main", map[string]string{"openspec/changes/x/design.md": "d\n"}, "diseño")
	seen := observe(t)

	autofetchRun(t, autofetchOn, repo, relevanceClock)

	if fetches := fetchInvocations(*seen); len(fetches) != 1 {
		t.Errorf("se esperaba un único fetch: %v", fetches)
	}
	noRemoteOperations(t, *seen)
}
