package engine

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `axiom branch scope` (C-02-b, RF-BRA-35 a RF-BRA-38): el alcance del cambio por repositorio, solo
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

// RF-BRA-35 y D3: sin kickoff ni documento ODD el cambio no está sellado.
func TestScopeRefusesAChangeWithoutKickoff(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))

	result := scopeOf(t, repo, "x")

	wantScopeRefusal(t, result, branch.CodeKickoffNotSealed)
	if result.Details["change"] != "x" {
		t.Errorf("detalles = %v, se esperaba el cambio x", result.Details)
	}
}

// RF-BRA-35: un cambio del carril ODD (con su documento y sin kickoff) y un kickoff sin bloque
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

// RF-BRA-02: un kickoff ilegible es `config_invalid`, no «sin sellar».
func TestScopeRefusesAnUnreadableKickoffWithConfigInvalid(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "kickoff.yaml"), "esto: [no es yaml\n")

	wantScopeRefusal(t, scopeOf(t, repo, "x"), branch.CodeConfigInvalid)
}

// RF-BRA-35: un nombre de cambio inválido y un directorio inexistente se rechazan antes de leer.
func TestScopeRefusesAnInvalidChangeNameAndAMissingDirectory(t *testing.T) {
	t.Parallel()
	repo := scopeProject(t, "")

	wantScopeRefusal(t, scopeOf(t, repo, "../x"), branch.CodeInvalidName)
	wantScopeRefusal(t, scopeOf(t, filepath.Join(repo, "no-existe"), "x"), branch.CodeNotAGitRepository)
}

// RF-BRA-35: el repositorio de specs in situ de un multirepo es `exempt`, sin ninguna lectura del
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

// RF-BRA-37, escenario «Rama borrada o ausente»: sin la rama en local ni como `origin/<rama>`, la
// entrada es `unverifiable` con `branch_missing`, el mensaje de la spec y la continuación `fetch`
// citada, que nunca se ejecuta; el resto de repositorios se calcula.
func TestScopeDeclaresABranchThatIsGoneUnverifiableWithoutFetching(t *testing.T) {
	master, _ := scopeMultirepo(t, "repo-api", "repo-web")
	api := filepath.Join(master, "repo-api")
	runGit(t, api, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, api, "a.go", "propio")
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
	if api := scopeEntryOf(t, result, "repo-api"); api.Outcome != branch.ScopeOutcomeOK || len(api.Files) != 1 {
		t.Errorf("repo-api = %+v, un repositorio no verificable no impide calcular el resto", api)
	}
	if result.Summary.Offer != branch.ScopeOfferUnverifiable || len(result.Summary.Pending) != 2 || result.Summary.Pending[1].Repository != "repo-web" {
		t.Errorf("resumen = %+v, se esperaba unverifiable con repo-api sin fusionar y repo-web ausente", result.Summary)
	}
	for _, invocation := range *seen {
		if invocation[1] == "fetch" {
			t.Errorf("scope nunca ejecuta fetch: %v", invocation)
		}
	}
	wantNoRemoteOrWrite(t, *seen)
}

// RF-BRA-37, escenario «Rama solo como referencia remota ya descargada»: se usa `origin/<rama>` y
// la entrada lo declara.
func TestScopeUsesTheDownloadedRemoteTrackingBranchAndSaysSo(t *testing.T) {
	t.Parallel()
	repo := scopeProject(t, "")
	runGit(t, repo, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, repo, "a.go", "propio")
	runGit(t, repo, "update-ref", "refs/remotes/origin/feat/x", "feat/x")
	runGit(t, repo, "switch", "-q", "main")
	runGit(t, repo, "branch", "-D", "feat/x")

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

// RF-BRA-37, escenario «Base o integración ausentes»: la entrada es `unverifiable` con
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

// P1 (RF-BRA-37): un repositorio declarado que no existe en esta máquina no se omite: es
// `unverifiable` con `branch_missing`, `details.path` y la ruta en el mensaje.
func TestScopeDeclaresARepositoryThatIsNotOnThisMachineUnverifiable(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"api": {"repo-api"}, "web": {"repo-web"}}, "repo-api")
	sealWithSnapshot(t, specs, "x", snapshotOf(t, master, "x", "api", "web"), "api", "web")
	api := filepath.Join(master, "repo-api")
	runGit(t, api, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, api, "a.go", "propio")

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

// RF-BRA-02: una `--cwd` relativa se resuelve a absoluta y toda ruta del resultado lo es.
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

// ---- Unidad 4: punto de partida por primer padre, commits propios, ficheros y fusión (E1, E2) ----

func commitShas(commits []branch.ScopeCommit) []string {
	shas := make([]string, 0, len(commits))
	for _, c := range commits {
		shas = append(shas, c.SHA)
	}
	slices.Sort(shas)
	return shas
}

func sortedShas(shas ...string) []string {
	shas = slices.Clone(shas)
	slices.Sort(shas)
	return shas
}

// wantOwnScope comprueba el alcance de un repositorio calculado: los commits propios, los ficheros
// exactos y que nada ajeno ni de la carpeta del cambio entra.
func wantOwnScope(t *testing.T, entry branch.RepositoryScope, ownCommits []string, files []string) {
	t.Helper()
	if entry.Outcome != branch.ScopeOutcomeOK {
		t.Fatalf("entrada = %+v, se esperaba el resultado ok", entry)
	}
	if got := commitShas(entry.Commits); !slices.Equal(got, sortedShas(ownCommits...)) {
		t.Errorf("commits = %v, se esperaban los propios %v", got, ownCommits)
	}
	if !slices.Equal(entry.Files, files) {
		t.Errorf("ficheros = %v, se esperaban exactamente %v", entry.Files, files)
	}
	for _, file := range entry.Files {
		if strings.HasPrefix(file, "openspec/changes/x/") {
			t.Errorf("la carpeta del cambio no entra en los ficheros: %s", file)
		}
	}
	if entry.Range != entry.Start.SHA+".."+entry.Tip || len(entry.Revisions) < 2 || entry.Revisions[0] != entry.Tip || entry.Revisions[1] != "^"+entry.Start.SHA {
		t.Errorf("rango %q y revisiones %v, se esperaba <inicio>..<punta> y [punta, ^inicio, …]", entry.Range, entry.Revisions)
	}
}

// releaseWithForeignCommit es E1: `feat/x` sale de `main` (integración y base, una etapa), tiene dos
// commits propios (el segundo toca también `openspec/changes/x/tasks.md`) y después llega a `main`
// un commit ajeno `F` que modifica `otro.go`. Devuelve el repositorio, el punto de bifurcación, los
// commits propios y `F`.
func releaseWithForeignCommit(t *testing.T) (repo, fork string, own []string, foreign string) {
	t.Helper()
	repo = scopeProject(t, "")
	fork = runGit(t, repo, "rev-parse", "main")
	runGit(t, repo, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, repo, "a.go", "propio uno")
	own = append(own, runGit(t, repo, "rev-parse", "HEAD"))
	writeFile(t, filepath.Join(repo, "b.go"), "propio dos\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "x", "tasks.md"), "tareas\n")
	runGit(t, repo, "add", "b.go", "openspec/changes/x/tasks.md")
	commit(t, repo, "propio dos")
	own = append(own, runGit(t, repo, "rev-parse", "HEAD"))
	runGit(t, repo, "switch", "-q", "main")
	commitFile(t, repo, "otro.go", "ajeno")
	foreign = runGit(t, repo, "rev-parse", "HEAD")
	return repo, fork, own, foreign
}

// RF-BRA-36, escenario «Commit ajeno en develop, modo release, rama sin fusionar», y RF-BRA-35,
// escenario «La orden no cambia nada»: el rango y los ficheros son solo los de la rama, sin `F`, sin
// `otro.go` y sin la carpeta del cambio; scope no lanza nada remoto ni cambia referencias, índice ni
// worktrees, con cambios con y sin stage y un fichero sin seguimiento que no entran en `files`.
func TestScopeReleaseBeforeTheMergeListsOnlyTheOwnCommitsAndChangesNothing(t *testing.T) {
	repo, fork, own, foreign := releaseWithForeignCommit(t)
	runGit(t, repo, "switch", "-q", "feat/x")
	writeFile(t, filepath.Join(repo, "a.go"), "modificado sin stage\n")
	writeFile(t, filepath.Join(repo, "b.go"), "modificado con stage\n")
	runGit(t, repo, "add", "b.go")
	writeFile(t, filepath.Join(repo, "sin-seguimiento.go"), "no confirmado\n")
	before := detachableState(t, repo)
	seen := observe(t)

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	wantOwnScope(t, entry, own, []string{"a.go", "b.go"})
	if entry.Start.SHA != fork || entry.Start.Source != branch.ScopeStartFirstParent || entry.Tip != own[1] {
		t.Errorf("inicio %+v y punta %s, se esperaban la bifurcación %s (first_parent) y %s", entry.Start, entry.Tip, fork, own[1])
	}
	if slices.Contains(commitShas(entry.Commits), foreign) || len(entry.Excluded) != 0 || len(entry.Warnings) != 0 {
		t.Errorf("el commit ajeno %s no entra ni se excluye ni avisa: commits %v, excluidos %v, avisos %v", foreign, entry.Commits, entry.Excluded, entry.Warnings)
	}
	if entry.Merged == nil || *entry.Merged {
		t.Errorf("merged = %v, se esperaba false", entry.Merged)
	}
	if result.Summary.Offer != branch.ScopeOfferPendingMerge || len(result.Summary.Pending) != 1 || result.Summary.Pending[0].Reason != branch.ScopeReasonNotMerged {
		t.Errorf("resumen = %+v, se esperaba pending_merge con la rama sin fusionar", result.Summary)
	}
	wantNoRemoteOrWrite(t, *seen)
	gitcommon.Observe = nil
	if after := detachableState(t, repo); after != before {
		t.Errorf("scope no cambia nada:\nantes:\n%s\ndespués:\n%s", before, after)
	}
}

// RF-BRA-36, escenario «Modo release, rama ya fusionada, mismo alcance» y RF-BRA-38: tras fusionar
// con un commit de fusión (o con un squash) el alcance es el mismo, sin `F`, y la rama figura
// fusionada.
func TestScopeReleaseAfterTheMergeKeepsTheSameScope(t *testing.T) {
	for _, tc := range []struct {
		name  string
		merge func(t *testing.T, repo string)
		by    branch.MergeLayer
	}{
		{"commit de fusión", func(t *testing.T, repo string) { mergeInto(t, repo, "feat/x") }, branch.LayerAncestry},
		{"squash", func(t *testing.T, repo string) {
			runGit(t, repo, "merge", "--squash", "feat/x")
			commit(t, repo, "squash de x")
			commitFile(t, repo, "posterior.go", "avanza main")
		}, branch.LayerEquivalentTree},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.by == branch.LayerEquivalentTree {
				requireMergeTree(t)
			}
			repo, fork, own, foreign := releaseWithForeignCommit(t)
			tc.merge(t, repo)

			result := scopeOf(t, repo, "x")

			wantComputed(t, result)
			entry := scopeEntryOf(t, result, filepath.Base(repo))
			wantOwnScope(t, entry, own, []string{"a.go", "b.go"})
			if entry.Start.SHA != fork {
				t.Errorf("inicio = %s, se esperaba la misma bifurcación %s antes y después de fusionar", entry.Start.SHA, fork)
			}
			if slices.Contains(commitShas(entry.Commits), foreign) || slices.Contains(entry.Files, "otro.go") {
				t.Errorf("ni el commit ajeno ni otro.go aparecen: %v %v", entry.Commits, entry.Files)
			}
			if entry.Merged == nil || !*entry.Merged || entry.MergedBy != tc.by || entry.MergedByLabel == "" {
				t.Errorf("merged = %v por %q, se esperaba true por %s", entry.Merged, entry.MergedBy, tc.by)
			}
			if result.Summary.Offer != branch.ScopeOfferOfferable {
				t.Errorf("oferta = %q, se esperaba offerable", result.Summary.Offer)
			}
		})
	}
}

// RF-BRA-36: los commits ajenos que llegan a la rama por una fusión de la integración se excluyen
// del rango, de los commits y de los ficheros, y se listan como excluidos.
func TestScopeReleaseExcludesTheForeignCommitsMergedIntoTheBranch(t *testing.T) {
	repo, _, own, foreign := releaseWithForeignCommit(t)
	runGit(t, repo, "switch", "-q", "feat/x")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "integra main", "main")
	mergeCommit := runGit(t, repo, "rev-parse", "HEAD")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	wantOwnScope(t, entry, append(own, mergeCommit), []string{"a.go", "b.go"})
	if !slices.Equal(entry.Excluded, []string{foreign}) {
		t.Errorf("excluidos = %v, se esperaba solo el ajeno %s", entry.Excluded, foreign)
	}
	if slices.Contains(entry.Files, "otro.go") {
		t.Errorf("otro.go del commit ajeno no entra: %v", entry.Files)
	}
	for _, c := range entry.Commits {
		if c.SHA == mergeCommit && !c.Merge {
			t.Errorf("el commit de fusión %s debe llevar merge: true", mergeCommit)
		}
	}
	if len(entry.Warnings) != 0 {
		t.Errorf("se distinguen los ajenos: no hay aviso, y hay %v", entry.Warnings)
	}
}

// RF-BRA-36: si el punto de bifurcación no puede determinarse (historias sin relación), la entrada
// es `unverifiable` con `fork_point_undetermined` y no presenta un rango vacío como válido.
func TestScopeReleaseWithUnrelatedHistoriesIsForkPointUndetermined(t *testing.T) {
	repo := scopeProject(t, "")
	root := runGit(t, repo, "commit-tree", "-m", "raíz sin relación", runGit(t, repo, "rev-parse", "main^{tree}"))
	runGit(t, repo, "branch", "feat/x", root)

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	if entry.Outcome != branch.ScopeOutcomeUnverifiable || entry.Reason != branch.ScopeReasonForkPointUndetermined || entry.Range != "" || len(entry.Commits) != 0 {
		t.Errorf("entrada = %+v, se esperaba unverifiable con fork_point_undetermined y sin rango", entry)
	}
	if result.Summary.Offer != branch.ScopeOfferUnverifiable {
		t.Errorf("oferta = %q, se esperaba unverifiable", result.Summary.Offer)
	}
}

// RF-BRA-37: un commit propio vacío (o de fusión, sin ficheros) cuenta como commit propio: la entrada
// se decide por los commits, no por la lista de ficheros. Solo es `empty` cuando todos los commits
// propios tocan únicamente la carpeta del cambio.
func TestScopeReleaseDecidesTheOutcomeByOwnCommitsAndNotByFiles(t *testing.T) {
	t.Run("un commit vacío es propio y la entrada es ok con ficheros vacíos", func(t *testing.T) {
		repo := scopeProject(t, "")
		runGit(t, repo, "switch", "-q", "-c", "feat/x", "main")
		commit(t, repo, "commit vacío")
		empty := runGit(t, repo, "rev-parse", "HEAD")

		result := scopeOf(t, repo, "x")

		wantComputed(t, result)
		entry := scopeEntryOf(t, result, filepath.Base(repo))
		if entry.Outcome != branch.ScopeOutcomeOK || !slices.Equal(commitShas(entry.Commits), []string{empty}) || len(entry.Files) != 0 {
			t.Errorf("entrada = %+v, se esperaba ok con el commit vacío y sin ficheros", entry)
		}
	})
	t.Run("todos los commits tocan solo la carpeta del cambio: empty", func(t *testing.T) {
		repo := scopeProject(t, "")
		runGit(t, repo, "switch", "-q", "-c", "feat/x", "main")
		commitFile(t, repo, "openspec/changes/x/tasks.md", "tareas")

		result := scopeOf(t, repo, "x")

		wantComputed(t, result)
		entry := scopeEntryOf(t, result, filepath.Base(repo))
		if entry.Outcome != branch.ScopeOutcomeEmpty || entry.Range != "" || len(entry.Commits) != 0 || len(entry.Files) != 0 || entry.Merged != nil {
			t.Errorf("entrada = %+v, se esperaba empty sin rango, commits, ficheros ni fusión", entry)
		}
		if result.Summary.Offer != branch.ScopeOfferNothingToVerify {
			t.Errorf("oferta = %q, se esperaba nothing_to_verify", result.Summary.Offer)
		}
	})
}

// twoStageScope es E2: monorepo de dos etapas (`master` y `develop`, modo `branch` por defecto) con
// el kickoff de `x` sellado. `feat/x` sale de `master` con dos commits propios y a `develop` llega
// después un commit ajeno `F` que modifica `otro.go`.
func twoStageScope(t *testing.T) (repo string, own []string, foreign string) {
	t.Helper()
	repo, _, _ = twoStageProject(t, "")
	sealWithSnapshot(t, repo, "x", snapshotOf(t, repo, "x", "core"), "core")
	runGit(t, repo, "switch", "-q", "-c", "feat/x", "master")
	commitFile(t, repo, "a.go", "propio uno")
	own = append(own, runGit(t, repo, "rev-parse", "HEAD"))
	commitFile(t, repo, "b.go", "propio dos")
	own = append(own, runGit(t, repo, "rev-parse", "HEAD"))
	runGit(t, repo, "switch", "-q", "develop")
	commitFile(t, repo, "otro.go", "ajeno")
	foreign = runGit(t, repo, "rev-parse", "HEAD")
	return repo, own, foreign
}

// RF-BRA-36 y RF-BRA-38, modo `branch`: la rama fusionada en `develop` se mide frente a `master`
// (solo sus commits propios) y figura fusionada en la integración, no en la rama principal.
func TestScopeBranchMergedIntoTheIntegrationHasOnlyTheOwnCommits(t *testing.T) {
	repo, own, foreign := twoStageScope(t)
	mergeInto(t, repo, "feat/x")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	wantOwnScope(t, entry, own, []string{"a.go", "b.go"})
	if slices.Contains(commitShas(entry.Commits), foreign) || len(entry.Excluded) != 0 || len(entry.Warnings) != 0 {
		t.Errorf("commits %v, excluidos %v, avisos %v; el ajeno %s no entra", entry.Commits, entry.Excluded, entry.Warnings, foreign)
	}
	if entry.Merged == nil || !*entry.Merged || entry.MergedBy != branch.LayerAncestry {
		t.Errorf("merged = %v por %q, se esperaba true por ascendencia", entry.Merged, entry.MergedBy)
	}
	if entry.Base == nil || entry.Base.Name != "master" || entry.Integration == nil || entry.Integration.Name != "develop" {
		t.Errorf("base %+v e integración %+v, se esperaban master y develop", entry.Base, entry.Integration)
	}
	if result.Summary.Offer != branch.ScopeOfferOfferable {
		t.Errorf("oferta = %q, se esperaba offerable", result.Summary.Offer)
	}
}

// RF-BRA-36, modo `branch`: con `develop` fusionada dentro de la rama, los commits de `develop` se
// excluyen del alcance y se listan como excluidos.
func TestScopeBranchExcludesTheIntegrationCommitsMergedIntoTheBranch(t *testing.T) {
	repo, own, foreign := twoStageScope(t)
	integration := runGit(t, repo, "rev-parse", "develop~1") // el commit «integración» de develop
	runGit(t, repo, "switch", "-q", "feat/x")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "integra develop", "develop")
	mergeCommit := runGit(t, repo, "rev-parse", "HEAD")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	wantOwnScope(t, entry, append(own, mergeCommit), []string{"a.go", "b.go"})
	if !slices.Equal(sortedShas(entry.Excluded...), sortedShas(foreign, integration)) {
		t.Errorf("excluidos = %v, se esperaban los de develop %s y %s", entry.Excluded, foreign, integration)
	}
	if slices.Contains(entry.Files, "otro.go") || len(entry.Warnings) != 0 {
		t.Errorf("otro.go no entra y no hay aviso: ficheros %v, avisos %v", entry.Files, entry.Warnings)
	}
	if len(entry.Revisions) != 3 || entry.Revisions[2] != "^"+runGit(t, repo, "rev-parse", "develop") {
		t.Errorf("revisiones = %v, se esperaba [punta, ^inicio, ^integración]", entry.Revisions)
	}
}

// RF-BRA-36, escenario «Commits ajenos no distinguibles se avisan»: si `develop` entra en la rama por
// avance rápido, sus commits no se distinguen de los propios: el alcance es el rango disponible y la
// entrada lleva `foreign_commits_possible`.
func TestScopeBranchWarnsWhenTheIntegrationEntersTheBranchByFastForward(t *testing.T) {
	repo, _, foreign := twoStageScope(t)
	runGit(t, repo, "switch", "-q", "feat/x")
	runGit(t, repo, "reset", "-q", "--hard", "master")
	runGit(t, repo, "merge", "-q", "--ff-only", "develop")
	commitFile(t, repo, "c.go", "propio tras avanzar")
	own := runGit(t, repo, "rev-parse", "HEAD")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	if !slices.Equal(entry.Warnings, []string{branch.ScopeWarningForeignCommitsPossible}) || len(entry.Excluded) != 0 {
		t.Fatalf("avisos %v y excluidos %v, se esperaba solo foreign_commits_possible", entry.Warnings, entry.Excluded)
	}
	if !slices.Contains(commitShas(entry.Commits), own) || !slices.Contains(commitShas(entry.Commits), foreign) {
		t.Errorf("commits = %v, se esperaba el rango disponible (con el propio %s y el ajeno posible %s)", entry.Commits, own, foreign)
	}
}

// RF-BRA-36: la rama que entra en `develop` por avance rápido no tiene ajenos que no se distingan:
// `develop` apunta a la punta de la rama y no hay aviso.
func TestScopeBranchFastForwardedIntoTheIntegrationDoesNotWarn(t *testing.T) {
	repo, _, _ := twoStageProject(t, "")
	sealWithSnapshot(t, repo, "x", snapshotOf(t, repo, "x", "core"), "core")
	runGit(t, repo, "switch", "-q", "-c", "feat/x", "master")
	commitFile(t, repo, "a.go", "propio uno")
	own := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "branch", "-f", "develop", "feat/x")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	wantOwnScope(t, entry, []string{own}, []string{"a.go"})
	if len(entry.Warnings) != 0 || len(entry.Excluded) != 0 {
		t.Errorf("avisos %v y excluidos %v, no debía haber ninguno: develop es la propia rama", entry.Warnings, entry.Excluded)
	}
	if entry.Merged == nil || !*entry.Merged {
		t.Errorf("merged = %v, se esperaba true: develop contiene la rama", entry.Merged)
	}
}

// RF-BRA-37, escenario «Rama sin commits propios», modo `branch`: una rama recién creada desde
// `master` es `empty`.
func TestScopeBranchWithoutOwnCommitsIsEmpty(t *testing.T) {
	repo, _, _ := twoStageProject(t, "")
	sealWithSnapshot(t, repo, "x", snapshotOf(t, repo, "x", "core"), "core")
	runGit(t, repo, "branch", "feat/x", "master")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	if entry.Outcome != branch.ScopeOutcomeEmpty || entry.Range != "" || len(entry.Commits) != 0 {
		t.Errorf("entrada = %+v, se esperaba empty y no ok con un rango vacío", entry)
	}
}

// ---- Unidad 5: avance rápido por reflog (E3) y multirepo (E4) ----

// fastForwarded es E3: `feat/x` sale de `main` con dos commits propios y `main` avanza hasta ella
// por avance rápido. Devuelve el repositorio, el punto de bifurcación y los commits propios.
func fastForwarded(t *testing.T) (repo, fork string, own []string) {
	t.Helper()
	repo = scopeProject(t, "")
	fork = runGit(t, repo, "rev-parse", "main")
	runGit(t, repo, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, repo, "a.go", "propio uno")
	own = append(own, runGit(t, repo, "rev-parse", "HEAD"))
	commitFile(t, repo, "b.go", "propio dos")
	own = append(own, runGit(t, repo, "rev-parse", "HEAD"))
	runGit(t, repo, "switch", "-q", "main")
	runGit(t, repo, "merge", "-q", "--ff-only", "feat/x")
	return repo, fork, own
}

// RF-BRA-36, escenario «Modo release, rama ya fusionada, mismo alcance» con avance rápido: la rama
// local fusionada por avance rápido se resuelve con el reflog de su creación, que es el punto de
// partida estable.
func TestScopeFastForwardOfALocalBranchStartsAtItsCreationInTheReflog(t *testing.T) {
	repo, fork, own := fastForwarded(t)
	seen := observe(t)

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	wantOwnScope(t, entry, own, []string{"a.go", "b.go"})
	if entry.Start.SHA != fork || entry.Start.Source != branch.ScopeStartReflog || entry.Start.SourceLabel == "" {
		t.Errorf("inicio = %+v, se esperaba la bifurcación %s por reflog", entry.Start, fork)
	}
	if entry.Merged == nil || !*entry.Merged || entry.MergedBy != branch.LayerAncestry {
		t.Errorf("merged = %v por %q, se esperaba true por ascendencia", entry.Merged, entry.MergedBy)
	}
	wantNoRemoteOrWrite(t, *seen)
}

// RF-BRA-37: una rama local recién creada, sin commits, es `empty` y no `fork_point_undetermined`: el
// reflog dice que la rama se creó en su propia punta.
func TestScopeFastForwardOfANewlyCreatedBranchIsEmpty(t *testing.T) {
	t.Parallel()
	repo := scopeProject(t, "")
	runGit(t, repo, "branch", "feat/x", "main")

	result := scopeOf(t, repo, "x")

	wantComputed(t, result)
	entry := scopeEntryOf(t, result, filepath.Base(repo))
	if entry.Outcome != branch.ScopeOutcomeEmpty || entry.Range != "" || len(entry.Commits) != 0 {
		t.Errorf("entrada = %+v, se esperaba empty", entry)
	}
	if result.Summary.Offer != branch.ScopeOfferNothingToVerify {
		t.Errorf("oferta = %q, se esperaba nothing_to_verify", result.Summary.Offer)
	}
}

// RF-BRA-36, escenario «Modo release, punto de bifurcación no determinable»: sin reflog de creación
// válido (rama solo remota, o adoptada de `origin/<rama>`, RF-BRA-27) y fusionada por avance rápido,
// la entrada es `unverifiable` con `fork_point_undetermined` y no presenta un rango vacío.
func TestScopeFastForwardWithoutAValidCreationIsForkPointUndetermined(t *testing.T) {
	t.Run("rama solo remota", func(t *testing.T) {
		t.Parallel()
		repo, _, _ := fastForwarded(t)
		runGit(t, repo, "update-ref", "refs/remotes/origin/feat/x", "feat/x")
		runGit(t, repo, "branch", "-D", "feat/x")

		result := scopeOf(t, repo, "x")

		wantComputed(t, result)
		entry := scopeEntryOf(t, result, filepath.Base(repo))
		if entry.Outcome != branch.ScopeOutcomeUnverifiable || entry.Reason != branch.ScopeReasonForkPointUndetermined || entry.Range != "" || len(entry.Commits) != 0 {
			t.Errorf("entrada = %+v, se esperaba unverifiable con fork_point_undetermined y sin rango", entry)
		}
		if entry.Branch == nil || entry.Branch.Source != branch.ScopeSourceRemoteTracking {
			t.Errorf("rama = %+v, se esperaba la referencia remota declarada", entry.Branch)
		}
	})
	t.Run("rama adoptada de origin", func(t *testing.T) {
		t.Parallel()
		repo, _, _ := fastForwarded(t)
		runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
		runGit(t, repo, "update-ref", "refs/remotes/origin/feat/x", "feat/x")
		runGit(t, repo, "branch", "-D", "feat/x")
		runGit(t, repo, "branch", "--track", "feat/x", "refs/remotes/origin/feat/x")

		result := scopeOf(t, repo, "x")

		wantComputed(t, result)
		entry := scopeEntryOf(t, result, filepath.Base(repo))
		if entry.Outcome != branch.ScopeOutcomeUnverifiable || entry.Reason != branch.ScopeReasonForkPointUndetermined {
			t.Errorf("entrada = %+v, se esperaba unverifiable con fork_point_undetermined: la creación es una adopción", entry)
		}
	})
}

// E4 (RF-BRA-35, RF-BRA-37, RF-BRA-38) en un multirepo con `api`, `web` y el repositorio de specs in
// situ: `api` fusionada es `ok`; `web` sin rama es `branch_missing` sin ejecutar `fetch`; specs es
// `exempt`; y la oferta es `unverifiable`.
func TestScopeMultirepoCombinesMergedMissingAndExemptRepositories(t *testing.T) {
	master, specs := scopeMultirepo(t, "repo-api", "repo-web")
	api := filepath.Join(master, "repo-api")
	runGit(t, api, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, api, "a.go", "propio de api")
	own := runGit(t, api, "rev-parse", "HEAD")
	runGit(t, api, "switch", "-q", "main")
	mergeInto(t, api, "feat/x")
	seen := observe(t)

	result := scopeOf(t, master, "x")

	wantComputed(t, result)
	if len(result.Repositories) != 3 {
		t.Fatalf("repositorios = %+v, se esperaban specs, api y web", result.Repositories)
	}
	entry := scopeEntryOf(t, result, "repo-api")
	wantOwnScope(t, entry, []string{own}, []string{"a.go"})
	if entry.Merged == nil || !*entry.Merged || entry.Role != branch.RepoCode {
		t.Errorf("repo-api = %+v, se esperaba código fusionada", entry)
	}
	web := scopeEntryOf(t, result, "repo-web")
	if web.Outcome != branch.ScopeOutcomeUnverifiable || web.Reason != branch.ScopeReasonBranchMissing || web.Continuation != "git -C "+pathquote.Quote(web.GitRoot)+" fetch origin" {
		t.Errorf("repo-web = %+v, se esperaba unverifiable con branch_missing y la continuación fetch", web)
	}
	wantSameDirectory(t, "gitRoot de specs", scopeEntryOf(t, result, "repo-specs").GitRoot, specs)
	if exempt := scopeEntryOf(t, result, "repo-specs"); exempt.Outcome != branch.ScopeOutcomeExempt || exempt.Range != "" || len(exempt.Files) != 0 {
		t.Errorf("repo-specs = %+v, se esperaba exenta y sin cálculo", exempt)
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
	wantNoScopeReadIn(t, *seen, specs)
}

// E4, subcaso: con `web` presente y sin fusionar y `api` fusionada, la oferta es `pending_merge` y
// el resumen nombra `web` y su rama.
func TestScopeMultirepoWithAnUnmergedRepositoryIsPendingMerge(t *testing.T) {
	t.Parallel()
	master, _ := scopeMultirepo(t, "repo-api", "repo-web")
	api, web := filepath.Join(master, "repo-api"), filepath.Join(master, "repo-web")
	runGit(t, api, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, api, "a.go", "propio de api")
	runGit(t, api, "switch", "-q", "main")
	mergeInto(t, api, "feat/x")
	runGit(t, web, "switch", "-q", "-c", "feat/x", "main")
	commitFile(t, web, "w.go", "propio de web")

	result := scopeOf(t, master, "x")

	wantComputed(t, result)
	if result.Summary.Offer != branch.ScopeOfferPendingMerge || len(result.Summary.Pending) != 1 {
		t.Fatalf("resumen = %+v, se esperaba pending_merge con un pendiente", result.Summary)
	}
	pending := result.Summary.Pending[0]
	if pending.Repository != "repo-web" || pending.Branch != "feat/x" || pending.Reason != branch.ScopeReasonNotMerged {
		t.Errorf("pendiente = %+v, se esperaba repo-web con feat/x sin fusionar", pending)
	}
}
