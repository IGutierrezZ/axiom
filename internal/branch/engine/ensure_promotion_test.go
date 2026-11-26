package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// `ensure` y la promoción de ramas (C-02-a, RF-BRA-28, RF-BRA-29, RF-BRA-15, RF-BRA-10, design D4,
// D5 y D10): con `git.main_branch` distinta de la integración la rama del cambio sale de la
// principal (`branch`) o de la integración (`release`); `ask` pregunta por repositorio sin crear
// nada; el modo se registra con su origen y no cambia. Con git real y sin red. Usan `in_place`,
// que no crea worktrees (cada `git worktree add` cuesta segundos en Windows).

const twoStageGit = "git:\n  integration_branch: develop\n  main_branch: master\n"

// twoStageBranches deja el repositorio con `master` y `develop` (esta, un commit por delante y
// desplegada) y devuelve el SHA de cada una.
func twoStageBranches(t *testing.T, repo string) (master, develop string) {
	t.Helper()
	runGit(t, repo, "branch", "-m", "main", "master")
	master = runGit(t, repo, "rev-parse", "master")
	runGit(t, repo, "switch", "-q", "-c", "develop")
	commit(t, repo, "integración")
	return master, runGit(t, repo, "rev-parse", "develop")
}

// twoStageProject crea un monorepo embebido de dos etapas: `master` y `develop` (un commit por
// delante) con el axiom.yaml (git: de twoStageGit más extra) confirmado en `master`, de modo que
// toda rama que sale de cualquiera de las dos lo conserva al desplegarse en `in_place`.
func twoStageProject(t *testing.T, extra string) (repo, master, develop string) {
	t.Helper()
	repo = newRepository(t)
	writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, twoStageGit+extra))
	runGit(t, repo, "add", "axiom.yaml")
	commit(t, repo, "configuración")
	master, develop = twoStageBranches(t, repo)
	return repo, master, develop
}

func ensurePromoted(t *testing.T, request EnsureRequest) branch.Result {
	t.Helper()
	request.Mode = branch.ModeInPlace
	return ensureWith(t, hubWorktree, request)
}

func wantPromotion(t *testing.T, repository branch.RepositoryResult, mode branch.Promotion, source string) {
	t.Helper()
	if repository.Promotion != string(mode) || repository.PromotionSource != source || repository.PromotionLabel == "" || repository.PromotionSourceLabel == "" {
		t.Errorf("promoción %q (%q) con origen %q (%q); se esperaba %s con origen %s y las dos etiquetas",
			repository.Promotion, repository.PromotionLabel, repository.PromotionSource, repository.PromotionSourceLabel, mode, source)
	}
}

func wantWarning(t *testing.T, result branch.Result, fragment string) {
	t.Helper()
	for _, warning := range result.Warnings {
		if strings.Contains(warning, fragment) {
			return
		}
	}
	t.Errorf("avisos %q; se esperaba uno con %q", result.Warnings, fragment)
}

// RF-BRA-28 y RF-BRA-10: sin nada declarado rige el defecto `branch` (origen `default`): la rama
// del cambio sale de la rama principal, su PR sigue yendo a la integración y la base del
// resultado es la principal. El registro guarda el modo y su origen. Sin red.
func TestEnsurePromotionBranchIsTheDefaultAndCreatesTheChangeFromTheMainBranch(t *testing.T) {
	repo, master, _ := twoStageProject(t, "")
	seen := observe(t)

	result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, result)
	wantNoRemote(t, *seen)
	repository := result.Repositories[0]
	wantPromotion(t, repository, branch.PromotionBranch, branch.PromotionSourceDefault)
	if repository.MainBranch != "master" || repository.IntegrationBranch != "develop" || repository.Base == nil || repository.Base.Ref != "master" {
		t.Errorf("rama principal %q, integración %q, base %+v; se esperaba master, develop y base master", repository.MainBranch, repository.IntegrationBranch, repository.Base)
	}
	created := branchByName(t, repository, "feat/x")
	if created.PRTarget != "develop" {
		t.Errorf("la primera PR va a %q, se esperaba develop", created.PRTarget)
	}
	if got := runGit(t, repo, "rev-parse", "feat/x"); got != master {
		t.Errorf("feat/x sale de %s, se esperaba master (%s)", got, master)
	}
	record := loadRecord(t, repo, "x")
	if got := record.Repositories[0]; got.Promotion != branch.PromotionBranch || got.PromotionSource != branch.PromotionSourceDefault {
		t.Errorf("registro = %q con origen %q, se esperaba branch con origen default", got.Promotion, got.PromotionSource)
	}
}

// RF-BRA-28 (precedencia de política): `git.promotion` y `repositories[].promotion` fijan el modo
// con su origen; `release` crea desde la integración. La referencia remota ya traída prevalece
// sobre la rama local de la base.
func TestEnsurePromotionPolicyFixesTheModeAndTheBase(t *testing.T) {
	t.Run("git.promotion release sale de la integración con origen git", func(t *testing.T) {
		t.Parallel()
		repo, _, develop := twoStageProject(t, "  promotion: release\n")

		result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

		wantReady(t, result)
		wantPromotion(t, result.Repositories[0], branch.PromotionRelease, branch.PromotionSourceGit)
		if got := runGit(t, repo, "rev-parse", "feat/x"); got != develop {
			t.Errorf("feat/x sale de %s, se esperaba develop (%s)", got, develop)
		}
		if base := result.Repositories[0].Base; base == nil || base.Ref != "develop" {
			t.Errorf("base = %+v, se esperaba develop", base)
		}
	})

	t.Run("repositories[].promotion prevalece sobre git.promotion con origen repository", func(t *testing.T) {
		t.Parallel()
		repo, _, develop := twoStageProject(t, "  promotion: branch\nrepositories:\n  - path: \".\"\n    promotion: release\n")

		result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

		wantReady(t, result)
		wantPromotion(t, result.Repositories[0], branch.PromotionRelease, branch.PromotionSourceRepository)
		if got := runGit(t, repo, "rev-parse", "feat/x"); got != develop {
			t.Errorf("feat/x sale de %s, se esperaba develop (%s)", got, develop)
		}
	})

	t.Run("origin/master traído a local prevalece sobre master", func(t *testing.T) {
		t.Parallel()
		repo, _, develop := twoStageProject(t, "")
		runGit(t, repo, "update-ref", "refs/remotes/origin/master", develop)

		result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

		wantReady(t, result)
		if base := result.Repositories[0].Base; base == nil || base.Ref != "origin/master" {
			t.Errorf("base = %+v, se esperaba origin/master", base)
		}
		if got := runGit(t, repo, "rev-parse", "feat/x"); got != develop {
			t.Errorf("feat/x sale de %s, se esperaba origin/master (%s)", got, develop)
		}
	})
}

// RF-BRA-29: `--promotion` fija el modo con origen `flag`; con la política en `ask` es la
// respuesta y el origen es `answer`.
func TestEnsurePromotionFlagFixesTheModeWithItsSource(t *testing.T) {
	t.Run("con la política fija es flag", func(t *testing.T) {
		t.Parallel()
		repo, _, develop := twoStageProject(t, "  promotion: branch\n")

		result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Promotions: []branch.PromotionRequest{{Mode: branch.PromotionRelease}}})

		wantReady(t, result)
		wantPromotion(t, result.Repositories[0], branch.PromotionRelease, branch.PromotionSourceFlag)
		if got := runGit(t, repo, "rev-parse", "feat/x"); got != develop {
			t.Errorf("feat/x sale de %s, se esperaba develop (%s)", got, develop)
		}
	})

	t.Run("cualificado con el nombre del repositorio", func(t *testing.T) {
		t.Parallel()
		repo, master, _ := twoStageProject(t, "  promotion: release\n")

		result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Promotions: []branch.PromotionRequest{{Repository: filepath.Base(repo), Mode: branch.PromotionBranch}}})

		wantReady(t, result)
		wantPromotion(t, result.Repositories[0], branch.PromotionBranch, branch.PromotionSourceFlag)
		if got := runGit(t, repo, "rev-parse", "feat/x"); got != master {
			t.Errorf("feat/x sale de %s, se esperaba master (%s)", got, master)
		}
	})
}

// RF-BRA-29 (escenario `ask`): sin respuesta, `decision_required` sin crear nada ni escribir el
// registro; al repetir con la invocación propuesta la rama sale de la base correcta y el registro
// guarda `answer`; un `--promotion` distinto del registrado es `plan_conflict` sin tocar nada.
func TestEnsurePromotionAskAsksOnceRecordsTheAnswerAndRefusesAChangeOfMode(t *testing.T) {
	repo, master, _ := twoStageProject(t, "  promotion: ask\n")
	state := repositoryState(t, repo)

	asked := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

	wantRefusal(t, asked, branch.CodeDecisionRequired, branch.StatusDecisionRequired)
	if asked.Details["promotion"] != "ask" || asked.Details["name"] != filepath.Base(repo) {
		t.Errorf("detalles = %v, se esperaba el repositorio y promotion=ask", asked.Details)
	}
	decision := asked.Decision
	if decision == nil || decision.Question == "" || len(decision.Choices) != 2 || decision.Choices[0].Answer != "branch" || decision.Choices[1].Answer != "release" {
		t.Fatalf("decision = %+v, se esperaban las respuestas branch y release", decision)
	}
	for _, choice := range decision.Choices {
		if !strings.Contains(choice.Effect, "master") || !strings.Contains(choice.Effect, "develop") {
			t.Errorf("el efecto de %s debe nombrar las ramas reales: %q", choice.Answer, choice.Effect)
		}
	}
	if want := "--promotion branch --cwd"; !strings.Contains(decision.Choices[0].Invocation, want) || strings.Contains(decision.Choices[0].Invocation, "=") {
		t.Errorf("invocación %q: con un solo repositorio la respuesta no se cualifica y lleva %q", decision.Choices[0].Invocation, want)
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("la pregunta no crea nada:\n%s\n---\n%s", state, got)
	}
	wantNoRegistryFile(t, repo, "x")

	answered := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Promotions: []branch.PromotionRequest{{Mode: branch.PromotionBranch}}})

	wantReady(t, answered)
	wantPromotion(t, answered.Repositories[0], branch.PromotionBranch, branch.PromotionSourceAnswer)
	if got := runGit(t, repo, "rev-parse", "feat/x"); got != master {
		t.Errorf("feat/x sale de %s, se esperaba master (%s)", got, master)
	}
	if got := loadRecord(t, repo, "x").Repositories[0]; got.Promotion != branch.PromotionBranch || got.PromotionSource != branch.PromotionSourceAnswer {
		t.Errorf("registro = %q con origen %q, se esperaba branch con origen answer", got.Promotion, got.PromotionSource)
	}

	t.Run("repetir sin --promotion conserva la respuesta y no vuelve a preguntar", func(t *testing.T) {
		again := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

		wantReady(t, again)
		wantPromotion(t, again.Repositories[0], branch.PromotionBranch, branch.PromotionSourceAnswer)
	})

	t.Run("un modo distinto del registrado es plan_conflict y no toca nada", func(t *testing.T) {
		before := repositoryState(t, repo)
		file := recordFile(commonDir(repo), "sdd", "x")
		recorded, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		conflict := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Promotions: []branch.PromotionRequest{{Mode: branch.PromotionRelease}}})

		wantRefusal(t, conflict, branch.CodePlanConflict, branch.StatusRefused)
		if conflict.Details["promotion"] != "branch" || conflict.Details["requested"] != "release" {
			t.Errorf("detalles = %v, se esperaba el modo materializado branch y el pedido release", conflict.Details)
		}
		after, err := os.ReadFile(file)
		if err != nil || string(after) != string(recorded) {
			t.Errorf("el registro cambió (%v):\n%s\n---\n%s", err, recorded, after)
		}
		if got := repositoryState(t, repo); got != before {
			t.Errorf("el conflicto no toca nada:\n%s\n---\n%s", before, got)
		}
	})
}

// RF-BRA-15: sin `main_branch` ni el registro ni el resultado ni el plan cambian un byte, y un
// `--promotion` sin efecto solo avisa.
func TestEnsureWithoutMainBranchKeepsTheRegistryAndTheResultUntouched(t *testing.T) {
	repo := embeddedCoreProject(t)

	result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, result)
	recorded, err := os.ReadFile(recordFile(commonDir(repo), "sdd", "x"))
	if err != nil {
		t.Fatal(err)
	}
	for what, text := range map[string]string{"registro": string(recorded), "resultado": resultJSON(t, result)} {
		if strings.Contains(strings.ToLower(text), "promotion") || strings.Contains(text, "mainBranch") {
			t.Errorf("sin main_branch el %s no lleva campos de promoción:\n%s", what, text)
		}
	}

	t.Run("--promotion sin main_branch avisa y no cambia el registro", func(t *testing.T) {
		again := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Promotions: []branch.PromotionRequest{{Mode: branch.PromotionBranch}}})

		wantReady(t, again)
		wantWarning(t, again, "no tiene efecto")
		after, err := os.ReadFile(recordFile(commonDir(repo), "sdd", "x"))
		if err != nil || string(after) != string(recorded) {
			t.Errorf("el registro cambió (%v):\n%s\n---\n%s", err, recorded, after)
		}
	})
}

// RF-BRA-10 y matriz de amenazas («Base de creación»): en modo `branch` sin la rama principal,
// `base_not_found` nombra `git.main_branch` y no deja ni rama ni registro; `--base` manda sobre la
// rama principal.
func TestEnsurePromotionBranchWithoutTheMainBranchIsBaseNotFoundAndBaseWins(t *testing.T) {
	repo := newRepository(t)
	runGit(t, repo, "switch", "-q", "-c", "develop")
	writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, twoStageGit))
	runGit(t, repo, "add", "axiom.yaml")
	commit(t, repo, "configuración")
	state := repositoryState(t, repo)

	refused := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

	wantRefusal(t, refused, branch.CodeBaseNotFound, branch.StatusRefused)
	if !strings.Contains(refused.Message, "git.main_branch") || !strings.Contains(refused.Continuation, "git.main_branch") || refused.Details["branch"] != "master" {
		t.Errorf("el rechazo debe nombrar la rama principal y git.main_branch: %q / %q / %v", refused.Message, refused.Continuation, refused.Details)
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("el rechazo no deja efectos:\n%s\n---\n%s", state, got)
	}
	wantNoRegistryFile(t, repo, "x")

	t.Run("con --base manda la excepción", func(t *testing.T) {
		result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Exceptions: branch.Exceptions{Base: "develop"}})

		wantReady(t, result)
		if base := result.Repositories[0].Base; base == nil || base.Ref != "develop" {
			t.Errorf("base = %+v, se esperaba develop", base)
		}
	})
}

// RF-BRA-29: un `--promotion <repo>=…` cuyo repositorio no es del plan es `invalid_name` con los
// nombres válidos y sin efectos.
func TestEnsurePromotionWithAnUnknownRepositoryIsInvalidName(t *testing.T) {
	t.Parallel()
	repo, _, _ := twoStageProject(t, "")
	state := repositoryState(t, repo)

	result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Promotions: []branch.PromotionRequest{{Repository: "zz", Mode: branch.PromotionBranch}}})

	wantRefusal(t, result, branch.CodeInvalidName, branch.StatusRefused)
	if result.Details["repository"] != "zz" || !strings.Contains(result.Details["available"], filepath.Base(repo)) {
		t.Errorf("detalles = %v, se esperaba zz y los repositorios válidos", result.Details)
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("el rechazo no deja efectos:\n%s\n---\n%s", state, got)
	}
	wantNoRegistryFile(t, repo, "x")
}

// RF-BRA-29: un modo pedido que no es una respuesta (`ask`) es `config_invalid` por defensa: la
// CLI ya lo rechaza como error de uso.
func TestEnsurePromotionRevalidatesTheRequestedMode(t *testing.T) {
	t.Parallel()
	repo, _, _ := twoStageProject(t, "")

	result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x", Promotions: []branch.PromotionRequest{{Mode: branch.PromotionAsk}}})

	wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
	wantNoRegistryFile(t, repo, "x")
}

// RF-BRA-29 (multirepo): el repositorio de specs está exento; con dos repositorios de código en
// `ask`, dos llamadas secuenciales preguntan de uno en uno con `Details.name` y la segunda
// invocación conserva la primera respuesta; las respuestas se registran por repositorio.
func TestEnsurePromotionMultirepoAsksOneRepositoryPerCallAndKeepsThePreviousAnswers(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-a"}, "frontend": {"repo-b"}}, "repo-a", "repo-b")
	heads := map[string][2]string{}
	for _, name := range []string{"repo-specs", "repo-a", "repo-b"} {
		dir := filepath.Join(master, name)
		masterSHA, developSHA := twoStageBranches(t, dir)
		heads[name] = [2]string{masterSHA, developSHA}
	}
	writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", map[string][]string{"backend": {"repo-a"}, "frontend": {"repo-b"}}, twoStageGit+"  promotion: ask\n"))
	sealChangeAt(t, specs, "x", "backend", "frontend")

	first := ensurePromoted(t, EnsureRequest{CWD: master, Change: "x"})

	wantRefusal(t, first, branch.CodeDecisionRequired, branch.StatusDecisionRequired)
	if first.Details["name"] != "repo-a" {
		t.Fatalf("detalles = %v, se esperaba la pregunta de repo-a (el de specs está exento)", first.Details)
	}
	for _, choice := range first.Decision.Choices {
		if want := "--promotion repo-a=" + choice.Answer; !strings.Contains(choice.Invocation, want) {
			t.Errorf("la invocación %q debe cualificar la respuesta con %q", choice.Invocation, want)
		}
	}
	wantNoRegistryFile(t, specs, "x")

	second := ensurePromoted(t, EnsureRequest{CWD: master, Change: "x", Promotions: []branch.PromotionRequest{{Repository: "repo-a", Mode: branch.PromotionBranch}}})

	wantRefusal(t, second, branch.CodeDecisionRequired, branch.StatusDecisionRequired)
	if second.Details["name"] != "repo-b" {
		t.Fatalf("detalles = %v, se esperaba la pregunta de repo-b", second.Details)
	}
	for _, choice := range second.Decision.Choices {
		for _, want := range []string{"--promotion repo-a=branch", "--promotion repo-b=" + choice.Answer} {
			if !strings.Contains(choice.Invocation, want) {
				t.Errorf("la invocación %q debe conservar la primera respuesta y cualificar la segunda: falta %q", choice.Invocation, want)
			}
		}
	}
	wantNoRegistryFile(t, specs, "x")

	done := ensurePromoted(t, EnsureRequest{CWD: master, Change: "x", Promotions: []branch.PromotionRequest{
		{Repository: "repo-a", Mode: branch.PromotionBranch}, {Repository: "repo-b", Mode: branch.PromotionRelease},
		{Repository: "repo-specs", Mode: branch.PromotionBranch},
	}})

	wantReady(t, done)
	wantWarning(t, done, "repositorio de specs")
	byName := map[string]branch.RepositoryResult{}
	for _, repository := range done.Repositories {
		byName[repository.Name] = repository
	}
	if byName["repo-specs"].Promotion != "" || byName["repo-specs"].MainBranch != "" {
		t.Errorf("el repositorio de specs está exento: %+v", byName["repo-specs"])
	}
	wantPromotion(t, byName["repo-a"], branch.PromotionBranch, branch.PromotionSourceAnswer)
	wantPromotion(t, byName["repo-b"], branch.PromotionRelease, branch.PromotionSourceAnswer)
	if got := runGit(t, filepath.Join(master, "repo-a"), "rev-parse", "feat/x"); got != heads["repo-a"][0] {
		t.Errorf("repo-a: feat/x sale de %s, se esperaba master (%s)", got, heads["repo-a"][0])
	}
	if got := runGit(t, filepath.Join(master, "repo-b"), "rev-parse", "feat/x"); got != heads["repo-b"][1] {
		t.Errorf("repo-b: feat/x sale de %s, se esperaba develop (%s)", got, heads["repo-b"][1])
	}
	record := loadRecordAt(t, specs, branch.LaneSDD, "x")
	recorded := map[string]branch.RecordRepo{}
	for _, repository := range record.Repositories {
		recorded[repository.Name] = repository
	}
	if recorded["repo-a"].Promotion != branch.PromotionBranch || recorded["repo-b"].Promotion != branch.PromotionRelease || recorded["repo-specs"].Promotion != "" {
		t.Errorf("registro = %+v, se esperaba branch, release y el de specs sin modo", record.Repositories)
	}
}

// Un registro anterior a la promoción (sin el campo y con la rama del cambio ya creada) es de una
// etapa y equivale a `release` (origen `legacy`): `ensure` no rehace la base ni pregunta.
func TestEnsurePromotionTreatsARecordWithoutTheFieldAsLegacyRelease(t *testing.T) {
	t.Parallel()
	repo, _, develop := twoStageProject(t, "  promotion: ask\n")
	runGit(t, repo, "branch", "feat/x", develop)
	record := registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange})
	record.Mode = branch.ModeInPlace
	writeRegistry(t, commonDir(repo), record)

	result := ensurePromoted(t, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, result)
	wantPromotion(t, result.Repositories[0], branch.PromotionRelease, branch.PromotionSourceLegacy)
	if got := loadRecord(t, repo, "x").Repositories[0]; got.Promotion != branch.PromotionRelease || got.PromotionSource != branch.PromotionSourceLegacy {
		t.Errorf("registro = %q con origen %q, se esperaba release con origen legacy", got.Promotion, got.PromotionSource)
	}
}

// La pregunta del modo de trabajo conserva en sus invocaciones las excepciones, los `--repo` y los
// `--promotion` ya dados: responderla no los pierde.
func TestEnsureWorkModeDecisionKeepsTheExceptionsAndThePromotionAnswers(t *testing.T) {
	t.Parallel()
	repo, _, _ := twoStageProject(t, "")

	asked := ensureWith(t, hubAsk, EnsureRequest{
		CWD: repo, Change: "x", Lane: branch.LaneSDD, Exceptions: branch.Exceptions{Prefix: "inc"},
		Promotions: []branch.PromotionRequest{{Mode: branch.PromotionRelease}},
	})

	wantRefusal(t, asked, branch.CodeDecisionRequired, branch.StatusDecisionRequired)
	for _, choice := range asked.Decision.Choices {
		for _, want := range []string{"--prefix inc", "--promotion release", "--mode " + choice.Answer} {
			if !strings.Contains(choice.Invocation, want) {
				t.Errorf("la invocación %q debe conservar %q", choice.Invocation, want)
			}
		}
	}
}
