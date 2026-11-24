package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `axiom branch scope` (C-02-b, REQ-25.51 a REQ-25.54): el alcance del cambio por repositorio, solo
// con lecturas de git local. Con git real, entorno aislado y sin red.

func scopeOf(t *testing.T, cwd, change string) branch.ScopeResult {
	t.Helper()
	return Default().Scope(context.Background(), cwd, change)
}

// scopeProject crea un monorepo embebido con el rol `fullstack` y el kickoff de `x` sellado con
// su instantánea de política git, como lo deja `axiom sdd kickoff seal`.
func scopeProject(t *testing.T, extraGit string) string {
	t.Helper()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, "git:\n  integration_branch: main\n  branch_prefix: feat\n"+extraGit))
	sealWithSnapshot(t, repo, "x", snapshotOf(t, repo, "x", "fullstack"), "fullstack")
	return repo
}

// scopeMultirepo crea un multirepo con los repositorios de código dados, un rol por repositorio
// (con el nombre del repositorio sin el prefijo `repo-`) y el kickoff de `x` sellado en el repositorio
// de specs con su instantánea.
func scopeMultirepo(t *testing.T, code ...string) (master, specs string) {
	t.Helper()
	roles := map[string][]string{}
	var names []string
	for _, repo := range code {
		role := strings.TrimPrefix(repo, "repo-")
		roles[role] = []string{repo}
		names = append(names, role)
	}
	master, specs = newMultirepo(t, roles, code...)
	sealWithSnapshot(t, specs, "x", snapshotOf(t, master, "x", names...), names...)
	return master, specs
}

func wantScopeRefusal(t *testing.T, result branch.ScopeResult, code branch.Code) {
	t.Helper()
	if result.Code != code || result.Status != branch.StatusRefused || result.ExitCode() != branch.ExitRefused {
		t.Fatalf("estado %q, código %q (%s), salida %d; se esperaba refused con %s y salida 1", result.Status, result.Code, result.Message, result.ExitCode(), code)
	}
	if result.Message == "" || result.Continuation == "" {
		t.Errorf("todo rechazo lleva mensaje y continuación: %+v", result)
	}
	if len(result.Repositories) != 0 || result.Summary != nil {
		t.Errorf("un rechazo no lleva repositorios ni resumen: %+v", result)
	}
}

func scopeEntryOf(t *testing.T, result branch.ScopeResult, name string) branch.RepositoryScope {
	t.Helper()
	for _, repo := range result.Repositories {
		if repo.Name == name {
			return repo
		}
	}
	t.Fatalf("el resultado no tiene el repositorio %s: %+v", name, result.Repositories)
	return branch.RepositoryScope{}
}

func wantComputed(t *testing.T, result branch.ScopeResult) {
	t.Helper()
	if result.Status != branch.StatusReady || result.ExitCode() != branch.ExitReady || result.Summary == nil || result.Schema != branch.ScopeSchema {
		t.Fatalf("estado %q, código %q (%s), salida %d, resumen %+v; se esperaba un alcance calculado", result.Status, result.Code, result.Message, result.ExitCode(), result.Summary)
	}
}

// scopeReads son los subcomandos de lectura propios del alcance: ninguno puede lanzarse en un
// repositorio exento ni puede ser remoto.
func wantNoScopeReadIn(t *testing.T, seen [][]string, repo string) {
	t.Helper()
	for _, invocation := range seen {
		if !strings.EqualFold(filepath.Clean(invocation[0]), filepath.Clean(repo)) {
			continue
		}
		switch sub := invocation[1]; {
		case sub == "rev-list", sub == "log", sub == "reflog", sub == "merge-base", sub == "merge-tree",
			sub == "rev-parse" && len(invocation) > 2 && invocation[2] == "--verify":
			t.Errorf("el repositorio %s es exento y no admite lecturas del alcance, pero se lanzó %v", repo, invocation)
		}
	}
}

func wantNoRemoteOrWrite(t *testing.T, seen [][]string) {
	t.Helper()
	for _, invocation := range seen {
		if remoteOperation(invocation[1]) || isWriteInvocation(invocation) || invocation[1] == "commit" {
			t.Errorf("`scope` es de solo lectura y sin red, pero lanzó %v", invocation)
		}
	}
}

// REQ-25.51 y D3: sin kickoff ni documento ODD el cambio no está sellado.
func TestScopeRefusesAChangeWithoutKickoff(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))

	result := scopeOf(t, repo, "x")

	wantScopeRefusal(t, result, branch.CodeKickoffNotSealed)
	if result.Details["change"] != "x" {
		t.Errorf("detalles = %v, se esperaba el cambio x", result.Details)
	}
}

// REQ-25.51: un cambio del carril ODD (con su documento y sin kickoff) y un kickoff sin bloque
// `git:` no son aplicables.
func TestScopeIsNotApplicableToAnOddChangeNorToAKickoffWithoutASnapshot(t *testing.T) {
	t.Parallel()
	t.Run("documento ODD sin kickoff", func(t *testing.T) {
		t.Parallel()
		repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))
		writeFile(t, filepath.Join(repo, "odd", "tasks", "x.md"), "# x\n")

		wantScopeRefusal(t, scopeOf(t, repo, "x"), branch.CodeScopeNotApplicable)
	})
	t.Run("kickoff sin instantánea de repositorios", func(t *testing.T) {
		t.Parallel()
		repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))
		sealChange(t, repo, "x", "fullstack")

		result := scopeOf(t, repo, "x")

		wantScopeRefusal(t, result, branch.CodeScopeNotApplicable)
		if !strings.Contains(result.Message, "instantánea") {
			t.Errorf("el mensaje debía explicar la falta de instantánea: %s", result.Message)
		}
	})
}

// REQ-25.2: un kickoff ilegible es `config_invalid`, no «sin sellar».
func TestScopeRefusesAnUnreadableKickoffWithConfigInvalid(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "kickoff.yaml"), "esto: [no es yaml\n")

	wantScopeRefusal(t, scopeOf(t, repo, "x"), branch.CodeConfigInvalid)
}

// REQ-25.51: un nombre de cambio inválido y un directorio inexistente se rechazan antes de leer.
func TestScopeRefusesAnInvalidChangeNameAndAMissingDirectory(t *testing.T) {
	t.Parallel()
	repo := scopeProject(t, "")

	wantScopeRefusal(t, scopeOf(t, repo, "../x"), branch.CodeInvalidName)
	wantScopeRefusal(t, scopeOf(t, filepath.Join(repo, "no-existe"), "x"), branch.CodeNotAGitRepository)
}

// REQ-25.51: el repositorio de specs in situ de un multirepo es `exempt`, sin ninguna lectura del
// alcance; con solo ese repositorio en la instantánea, la oferta es `nothing_to_verify` y la salida, 0.
func TestScopeExemptsTheSpecsRepositoryAndOffersNothingToVerifyWhenItIsTheOnlyOne(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"docs": {"repo-specs"}})
	sealWithSnapshot(t, specs, "x", snapshotOf(t, master, "x", "docs"), "docs")
	seen := observe(t)

	result := scopeOf(t, master, "x")

	wantComputed(t, result)
	if len(result.Repositories) != 1 {
		t.Fatalf("repositorios = %+v, se esperaba solo el de specs", result.Repositories)
	}
	entry := result.Repositories[0]
	if entry.Role != branch.RepoSpecsInPlace || entry.Outcome != branch.ScopeOutcomeExempt || entry.OutcomeLabel == "" ||
		entry.Range != "" || len(entry.Files) != 0 || len(entry.Commits) != 0 || entry.Merged != nil {
		t.Errorf("entrada = %+v, se esperaba exenta, sin rango, commits, ficheros ni fusión", entry)
	}
	if result.Summary.Offer != branch.ScopeOfferNothingToVerify || len(result.Summary.Pending) != 0 {
		t.Errorf("resumen = %+v, se esperaba nothing_to_verify sin pendientes", result.Summary)
	}
	wantNoScopeReadIn(t, *seen, specs)
	wantNoRemoteOrWrite(t, *seen)
}

// REQ-25.53, escenario «Rama borrada o ausente»: sin la rama en local ni como `origin/<rama>`, la
// entrada es `unverifiable` con `branch_missing`, el mensaje de la spec y la continuación `fetch`
// citada, que nunca se ejecuta; el resto de repositorios se calcula.
func TestScopeDeclaresABranchThatIsGoneUnverifiableWithoutFetching(t *testing.T) {
	master, _ := scopeMultirepo(t, "repo-api", "repo-web")
	api := filepath.Join(master, "repo-api")
	runGit(t, api, "branch", "feat/x", "main")
	seen := observe(t)

	result := scopeOf(t, master, "x")

	wantComputed(t, result)
	web := scopeEntryOf(t, result, "repo-web")
	if web.Outcome != branch.ScopeOutcomeUnverifiable || web.Reason != branch.ScopeReasonBranchMissing || web.ReasonLabel == "" {
		t.Fatalf("repo-web = %+v, se esperaba unverifiable con branch_missing", web)
	}
	if web.Message != "la rama ya no está o falta un fetch autorizado" && !strings.Contains(web.Message, "la rama ya no está o falta un fetch autorizado") {
		t.Errorf("mensaje = %q, se esperaba la frase de la spec", web.Message)
	}
	wantFetch := "git -C " + pathquote.Quote(web.GitRoot) + " fetch origin"
	if web.Continuation != wantFetch {
		t.Errorf("continuación = %q, se esperaba %q", web.Continuation, wantFetch)
	}
	wantSameDirectory(t, "gitRoot de repo-web", web.GitRoot, filepath.Join(master, "repo-web"))
	if scopeEntryOf(t, result, "repo-api").Outcome == branch.ScopeOutcomeUnverifiable {
		t.Error("un repositorio no verificable no impide calcular el resto")
	}
	if result.Summary.Offer != branch.ScopeOfferUnverifiable || len(result.Summary.Pending) != 1 || result.Summary.Pending[0].Repository != "repo-web" {
		t.Errorf("resumen = %+v, se esperaba unverifiable con repo-web pendiente", result.Summary)
	}
	for _, invocation := range *seen {
		if invocation[1] == "fetch" {
			t.Errorf("scope nunca ejecuta fetch: %v", invocation)
		}
	}
	wantNoRemoteOrWrite(t, *seen)
}

// REQ-25.53, escenario «Rama solo como referencia remota ya descargada»: se usa `origin/<rama>` y
// la entrada lo declara.
func TestScopeUsesTheDownloadedRemoteTrackingBranchAndSaysSo(t *testing.T) {
	t.Parallel()
	repo := scopeProject(t, "")
	runGit(t, repo, "update-ref", "refs/remotes/origin/feat/x", "main")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	if entry.Outcome == branch.ScopeOutcomeUnverifiable || entry.Branch == nil {
		t.Fatalf("entrada = %+v, no debía ser unverifiable", entry)
	}
	if entry.Branch.Source != branch.ScopeSourceRemoteTracking || entry.Branch.Ref != "refs/remotes/origin/feat/x" || entry.Branch.SourceLabel == "" {
		t.Errorf("rama = %+v, se esperaba la referencia remota origin/feat/x declarada como remote-tracking", entry.Branch)
	}
}

// REQ-25.53, escenario «Base o integración ausentes»: la entrada es `unverifiable` con
// `base_missing`, `details.missingRef` y la continuación `fetch`, sin ejecutarlo.
func TestScopeDeclaresAMissingBaseOrIntegrationUnverifiable(t *testing.T) {
	t.Run("integración ausente", func(t *testing.T) {
		repo := scopeProject(t, "")
		runGit(t, repo, "branch", "feat/x", "main")
		runGit(t, repo, "branch", "-m", "main", "otra")
		seen := observe(t)

		result := scopeOf(t, repo, "x")

		wantComputed(t, result)
		entry := scopeEntryOf(t, result, filepath.Base(repo))
		if entry.Outcome != branch.ScopeOutcomeUnverifiable || entry.Reason != branch.ScopeReasonBaseMissing {
			t.Fatalf("entrada = %+v, se esperaba unverifiable con base_missing", entry)
		}
		if entry.Details["missingRef"] != "main" || entry.Continuation != "git -C "+pathquote.Quote(entry.GitRoot)+" fetch origin" {
			t.Errorf("detalles %v y continuación %q; se esperaba missingRef main y la continuación fetch", entry.Details, entry.Continuation)
		}
		wantNoRemoteOrWrite(t, *seen)
	})
	t.Run("rama principal ausente en un repositorio de dos etapas", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		sealWithSnapshot(t, repo, "x", snapshotOf(t, repo, "x", "core"), "core")
		runGit(t, repo, "switch", "-q", "develop")
		runGit(t, repo, "branch", "feat/x", "develop")
		runGit(t, repo, "branch", "-D", "master")

		result := scopeOf(t, repo, "x")

		wantComputed(t, result)
		entry := scopeEntryOf(t, result, filepath.Base(repo))
		if entry.Reason != branch.ScopeReasonBaseMissing || entry.Details["missingRef"] != "master" {
			t.Errorf("entrada = %+v, se esperaba base_missing con missingRef master", entry)
		}
	})
}

// P1 (REQ-25.53): un repositorio declarado que no existe en esta máquina no se omite: es
// `unverifiable` con `branch_missing`, `details.path` y la ruta en el mensaje.
func TestScopeDeclaresARepositoryThatIsNotOnThisMachineUnverifiable(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"api": {"repo-api"}, "web": {"repo-web"}}, "repo-api")
	sealWithSnapshot(t, specs, "x", snapshotOf(t, master, "x", "api", "web"), "api", "web")
	runGit(t, filepath.Join(master, "repo-api"), "branch", "feat/x", "main")

	result := scopeOf(t, master, "x")

	wantComputed(t, result)
	missing := scopeEntryOf(t, result, "repo-web")
	absent := filepath.Join(master, "repo-web")
	if missing.Outcome != branch.ScopeOutcomeUnverifiable || missing.Reason != branch.ScopeReasonBranchMissing {
		t.Fatalf("repo-web = %+v, se esperaba unverifiable con branch_missing", missing)
	}
	if missing.Details["path"] != absent || !strings.Contains(missing.Message, absent) {
		t.Errorf("detalles %v y mensaje %q; se esperaba la ruta %s en ambos", missing.Details, missing.Message, absent)
	}
	if len(result.Repositories) != 3 {
		t.Errorf("repositorios = %d, se esperaban specs, api y web: ningún repositorio se omite", len(result.Repositories))
	}
	if result.Summary.Offer != branch.ScopeOfferUnverifiable {
		t.Errorf("oferta = %q, se esperaba unverifiable", result.Summary.Offer)
	}
}

// REQ-25.2: una `--cwd` relativa se resuelve a absoluta y toda ruta del resultado lo es.
func TestScopeResolvesARelativeCwdToAbsolutePaths(t *testing.T) {
	repo := scopeProject(t, "")
	runGit(t, repo, "branch", "feat/x", "main")
	t.Chdir(filepath.Dir(repo))

	result := scopeOf(t, filepath.Base(repo), "x")

	wantComputed(t, result)
	if !filepath.IsAbs(result.WorkspaceRoot) {
		t.Errorf("workspaceRoot = %q, se esperaba una ruta absoluta", result.WorkspaceRoot)
	}
	for _, entry := range result.Repositories {
		if !filepath.IsAbs(entry.GitRoot) {
			t.Errorf("gitRoot de %s = %q, se esperaba una ruta absoluta", entry.Name, entry.GitRoot)
		}
	}
	if _, err := os.Stat(result.WorkspaceRoot); err != nil {
		t.Errorf("workspaceRoot no existe: %v", err)
	}
}
