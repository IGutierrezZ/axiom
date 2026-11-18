package branch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

const promotionTestCWD = `C:\repos\mi proyecto`

// REQ-25.47 (D10): sin banderas opcionales, EnsureArgs.Invocation produce exactamente la cadena
// de la orden de modo de siempre, byte a byte.
func TestEnsureArgsInvocationWithoutOptionalFlagsKeepsTheCurrentString(t *testing.T) {
	for _, tc := range []struct {
		name string
		args EnsureArgs
		want string
	}{
		{"sdd con modo", EnsureArgs{Operation: "ensure", Change: "x", CWD: promotionTestCWD, Lane: LaneSDD, Mode: ModeWorktree},
			`axiom branch ensure --change x --mode worktree --cwd "C:\repos\mi proyecto"`},
		{"odd con modo", EnsureArgs{Operation: "ensure", Change: "mi-feature", CWD: promotionTestCWD, Lane: LaneODD, Mode: ModeInPlace},
			`axiom branch ensure --change mi-feature --lane odd --mode in_place --cwd "C:\repos\mi proyecto"`},
		{"sin modo", EnsureArgs{Operation: "ensure", Change: "x", CWD: promotionTestCWD},
			`axiom branch ensure --change x --cwd "C:\repos\mi proyecto"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.args.Invocation(); got != tc.want {
				t.Errorf("Invocation() = %q, se esperaba %q", got, tc.want)
			}
		})
	}
	if got, want := modeInvocation("ensure", "x", promotionTestCWD, LaneSDD, ModeWorktree), "axiom branch ensure --change x --mode worktree --cwd "+pathquote.Quote(promotionTestCWD); got != want {
		t.Errorf("modeInvocation = %q, se esperaba %q", got, want)
	}
}

func TestEnsureArgsInvocationComposesEveryOptionalFlagInOrder(t *testing.T) {
	args := EnsureArgs{
		Operation:  "ensure",
		Change:     "x",
		CWD:        promotionTestCWD,
		Lane:       LaneODD,
		Mode:       ModeWorktree,
		Exceptions: Exceptions{Prefix: "inc", Branch: "inc/x", Base: "origin/main"},
		Repos:      []string{`C:\repos\otro repo`},
		Promotions: []PromotionRequest{{Mode: PromotionRelease}, {Repository: "web app", Mode: PromotionBranch}},
	}
	want := `axiom branch ensure --change x --lane odd --mode worktree --prefix inc --branch inc/x --base origin/main` +
		` --repo "C:\repos\otro repo" --promotion release --promotion "web app=branch" --cwd "C:\repos\mi proyecto"`
	if got := args.Invocation(); got != want {
		t.Errorf("Invocation() =\n%s\nse esperaba\n%s", got, want)
	}
}

// DecisionRequired conserva las excepciones, los `--repo` y los `--promotion` dados.
func TestDecisionRequiredKeepsExceptionsReposAndPromotions(t *testing.T) {
	args := EnsureArgs{
		Operation: "ensure", Change: "x", CWD: promotionTestCWD, Lane: LaneSDD,
		Exceptions: Exceptions{Branch: "inc/x"},
		Repos:      []string{`C:\otro`},
		Promotions: []PromotionRequest{{Repository: "web", Mode: PromotionBranch}},
	}
	refusal := DecisionRequired(args)
	for i, mode := range []Mode{ModeWorktree, ModeInPlace} {
		invocation := refusal.Decision.Choices[i].Invocation
		for _, want := range []string{"--mode " + string(mode), "--branch inc/x", `--repo "C:\otro"`, "--promotion web=branch"} {
			if !strings.Contains(invocation, want) {
				t.Errorf("la invocación %q debe contener %q", invocation, want)
			}
		}
		if !strings.Contains(refusal.Continuation, invocation) {
			t.Errorf("la continuación debe contener %q", invocation)
		}
	}
}

func promotionRepo(name string) RepoPlan {
	return RepoPlan{Name: name, GitRoot: `C:\repos\` + name, IntegrationBranch: "develop", MainBranch: "master", Promotion: PromotionAsk}
}

// REQ-25.47: la pregunta de promoción nombra las ramas reales y lleva una invocación por respuesta.
func TestPromotionDecisionRequiredNamesTheRealBranches(t *testing.T) {
	args := EnsureArgs{Operation: "ensure", Change: "x", CWD: promotionTestCWD, Lane: LaneSDD, Mode: ModeWorktree}
	refusal := PromotionDecisionRequired(args, promotionRepo("api"), false)

	if refusal.Code != CodeDecisionRequired || refusal.Decision == nil || len(refusal.Decision.Choices) != 2 {
		t.Fatalf("rechazo = %+v, se esperaba decision_required con dos opciones", refusal)
	}
	wantDetails := map[string]string{"change": "x", "repository": `C:\repos\api`, "name": "api", "promotion": "ask"}
	for key, want := range wantDetails {
		if refusal.Details[key] != want {
			t.Errorf("Details[%s] = %q, se esperaba %q", key, refusal.Details[key], want)
		}
	}
	branchChoice, releaseChoice := refusal.Decision.Choices[0], refusal.Decision.Choices[1]
	if branchChoice.Answer != "branch" || releaseChoice.Answer != "release" {
		t.Errorf("respuestas = %q y %q", branchChoice.Answer, releaseChoice.Answer)
	}
	for _, want := range []string{"origin/master", "develop", "master"} {
		if !strings.Contains(branchChoice.Effect, want) {
			t.Errorf("el efecto de branch %q debe nombrar %q", branchChoice.Effect, want)
		}
	}
	if !strings.Contains(releaseChoice.Effect, "develop") || !strings.Contains(releaseChoice.Effect, "master") {
		t.Errorf("el efecto de release %q debe nombrar las dos ramas", releaseChoice.Effect)
	}
	if want := `axiom branch ensure --change x --mode worktree --promotion branch --cwd "C:\repos\mi proyecto"`; branchChoice.Invocation != want {
		t.Errorf("invocación branch = %q, se esperaba %q", branchChoice.Invocation, want)
	}
	if !strings.Contains(releaseChoice.Invocation, "--promotion release") {
		t.Errorf("invocación release = %q", releaseChoice.Invocation)
	}
	for _, choice := range refusal.Decision.Choices {
		if !strings.Contains(refusal.Continuation, choice.Invocation) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, choice.Invocation)
		}
	}
}

// Con dos repositorios, la respuesta se cualifica y conserva la respuesta anterior.
func TestPromotionDecisionRequiredQualifiesAndKeepsPreviousAnswers(t *testing.T) {
	args := EnsureArgs{Operation: "ensure", Change: "x", CWD: promotionTestCWD, Lane: LaneSDD,
		Promotions: []PromotionRequest{{Repository: "api", Mode: PromotionBranch}}}
	refusal := PromotionDecisionRequired(args, promotionRepo("web"), true)

	for i, mode := range []string{"branch", "release"} {
		invocation := refusal.Decision.Choices[i].Invocation
		if !strings.Contains(invocation, "--promotion api=branch") || !strings.Contains(invocation, "--promotion web="+mode) {
			t.Errorf("la invocación %q debe conservar api=branch y dar web=%s", invocation, mode)
		}
	}
	if len(args.Promotions) != 1 {
		t.Errorf("la decisión no modifica los argumentos recibidos: %+v", args.Promotions)
	}

	var out bytes.Buffer
	if err := NewResult("ensure").Refused(refusal).WriteJSON(&out); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil || decoded["decision"] == nil {
		t.Errorf("el JSON debe llevar `decision` (%v):\n%s", err, out.String())
	}
}

func TestPromotionConflictRepeatsWithTheMaterializedMode(t *testing.T) {
	args := EnsureArgs{Operation: "ensure", Change: "x", CWD: promotionTestCWD, Lane: LaneSDD,
		Promotions: []PromotionRequest{{Repository: "api", Mode: PromotionRelease}, {Repository: "web", Mode: PromotionBranch}}}
	refusal := PromotionConflict(args, promotionRepo("api"), PromotionBranch, PromotionRelease, PromotionSourceAnswer)

	if refusal.Code != CodePlanConflict {
		t.Fatalf("código = %s, se esperaba plan_conflict", refusal.Code)
	}
	for _, want := range []string{"api", "branch", "release", "respuesta registrada"} {
		if !strings.Contains(refusal.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
		}
	}
	if !strings.Contains(refusal.Continuation, "--promotion api=branch") || strings.Contains(refusal.Continuation, "api=release") {
		t.Errorf("la continuación %q debe repetir con el modo materializado", refusal.Continuation)
	}
	if !strings.Contains(refusal.Continuation, "--promotion web=branch") {
		t.Errorf("la continuación conserva las respuestas de los demás repositorios: %q", refusal.Continuation)
	}
	if refusal.Details["promotion"] != "branch" || refusal.Details["requested"] != "release" || refusal.Details["name"] != "api" {
		t.Errorf("Details = %+v", refusal.Details)
	}
}

func TestUnknownPromotionRepositoryListsTheValidNames(t *testing.T) {
	args := EnsureArgs{Operation: "ensure", Change: "x", CWD: promotionTestCWD, Lane: LaneSDD,
		Promotions: []PromotionRequest{{Repository: "nope", Mode: PromotionBranch}}}
	refusal := UnknownPromotionRepository(args, "nope", []string{"web", "api"})

	if refusal.Code != CodeInvalidName {
		t.Fatalf("código = %s, se esperaba invalid_name", refusal.Code)
	}
	if !strings.Contains(refusal.Message, "«nope»") || !strings.Contains(refusal.Message, "api, web") {
		t.Errorf("el mensaje %q debe nombrar el repositorio y la lista válida, ordenada", refusal.Message)
	}
	if strings.Contains(refusal.Continuation, "--promotion nope") {
		t.Errorf("la continuación no repite la petición inválida: %q", refusal.Continuation)
	}
	if !strings.Contains(UnknownPromotionRepository(args, "nope", nil).Message, "ninguno") {
		t.Errorf("sin repositorios en el plan, el mensaje lo dice")
	}
}

func TestInvalidPromotionOffersBothAnswers(t *testing.T) {
	args := EnsureArgs{Operation: "ensure", Change: "x", CWD: promotionTestCWD, Lane: LaneSDD,
		Promotions: []PromotionRequest{{Mode: PromotionAsk}}}
	refusal := InvalidPromotion(args, "ask")

	if refusal.Code != CodeConfigInvalid || !strings.Contains(refusal.Message, "«ask»") {
		t.Fatalf("rechazo = %+v", refusal)
	}
	if !strings.Contains(refusal.Continuation, "--promotion branch") || !strings.Contains(refusal.Continuation, "--promotion release") || strings.Contains(refusal.Continuation, "--promotion ask") {
		t.Errorf("la continuación %q debe ofrecer branch y release", refusal.Continuation)
	}
}

// BaseNotFound nombra la rama principal y `git.main_branch` en modo `branch`; con una etapa el
// mensaje de siempre.
func TestBaseNotFoundNamesTheMainBranchInBranchMode(t *testing.T) {
	two := RepoPlan{Name: "api", GitRoot: `C:\repos\api`, IntegrationBranch: "develop", MainBranch: "master", Promotion: PromotionBranch, BaseBranch: "master"}
	refusal := BaseNotFound("ensure", "x", promotionTestCWD, two)
	for _, want := range []string{"«master»", "git.main_branch"} {
		if !strings.Contains(refusal.Message+refusal.Continuation, want) {
			t.Errorf("el rechazo debe nombrar %q:\n%s\n%s", want, refusal.Message, refusal.Continuation)
		}
	}
	if refusal.Details["branch"] != "master" || strings.Contains(refusal.Continuation, "git.integration_branch") {
		t.Errorf("Details = %+v, continuación %q", refusal.Details, refusal.Continuation)
	}

	release := two
	release.Promotion, release.BaseBranch = PromotionRelease, "develop"
	if got := BaseNotFound("ensure", "x", promotionTestCWD, release); got.Details["branch"] != "develop" || !strings.Contains(got.Continuation, "git.integration_branch") {
		t.Errorf("en modo release se nombra la integración: %+v", got)
	}

	single := RepoPlan{Name: "api", GitRoot: `C:\repos\api`, IntegrationBranch: "main"}
	if got := BaseNotFound("ensure", "x", promotionTestCWD, single); got.Details["branch"] != "main" || !strings.Contains(got.Continuation, "git.integration_branch") {
		t.Errorf("sin main_branch el mensaje no cambia: %+v", got)
	}

	fixed := two
	fixed.BaseRef = "origin/otra"
	if got := BaseNotFound("ensure", "x", promotionTestCWD, fixed); got.Details["branch"] != "origin/otra" {
		t.Errorf("--base prevalece: %+v", got)
	}
}

// MergeNotConfirmed usa el destino dado en el mensaje, en la orden `git log` y en Details.target.
func TestMergeNotConfirmedUsesTheGivenDestination(t *testing.T) {
	repo := RepoPlan{Name: "api", GitRoot: `C:\mi repo`, IntegrationBranch: "develop", MainBranch: "master"}
	planned := BranchPlan{Name: "feat/x", Kind: KindChange, PRTarget: "develop"}

	refusal := MergeNotConfirmed("x", `C:\mi repo`, LaneSDD, repo, planned, "master", []string{"origin/master", "master"})

	if !strings.Contains(refusal.Message, "origin/master ni en master") || !strings.Contains(refusal.Continuation, "log --oneline master..feat/x") || refusal.Details["target"] != "master" {
		t.Errorf("merge_not_confirmed = %+v", refusal)
	}
	if empty := MergeNotConfirmed("x", `C:\mi repo`, LaneSDD, repo, planned, "master", nil); !strings.Contains(empty.Message, "no existe master") {
		t.Errorf("sin destino, el mensaje nombra master: %q", empty.Message)
	}
}

// NewRepositoryResult rellena los campos de promoción solo con dos etapas, y el texto los muestra.
func TestRepositoryResultCarriesThePromotionOnlyInTwoStages(t *testing.T) {
	two := RepoPlan{
		Name: "api", GitRoot: "/r/api", CommonDir: "/r/api/.git", Role: RepoChangeBranch, IntegrationBranch: "develop",
		MainBranch: "master", Promotion: PromotionBranch, PromotionSource: PromotionSourceAnswer,
		Branches: []BranchPlan{{Name: "feat/x", Kind: KindChange, PRTarget: "develop"}},
	}
	result := NewRepositoryResult(two)
	if result.MainBranch != "master" || result.Promotion != "branch" || result.PromotionLabel != "rama a rama" ||
		result.PromotionSource != "answer" || result.PromotionSourceLabel != "respuesta registrada" {
		t.Errorf("resultado = %+v", result)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"mainBranch":"master"`, `"promotion":"branch"`, `"promotionLabel"`, `"promotionSource":"answer"`, `"promotionSourceLabel"`} {
		if !strings.Contains(string(data), key) {
			t.Errorf("el JSON debe contener %s: %s", key, data)
		}
	}
	var text bytes.Buffer
	result.writeText(&text)
	if !strings.Contains(text.String(), "Rama principal: master") || !strings.Contains(text.String(), "Promoción: rama a rama (respuesta registrada)") {
		t.Errorf("texto:\n%s", text.String())
	}

	single := NewRepositoryResult(RepoPlan{Name: "api", IntegrationBranch: "main"})
	data, err = json.Marshal(single)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"mainBranch", "promotion", "promotionLabel", "promotionSource"} {
		if strings.Contains(string(data), `"`+key+`"`) {
			t.Errorf("sin main_branch el JSON no lleva %q: %s", key, data)
		}
	}
}

// La etapa de la rama sale con su etiqueta y no usa los nombres que leen los scripts.
func TestBranchResultPromotionStageIsAdditive(t *testing.T) {
	branchResult := BranchResult{Name: "feat/x", Kind: KindChange, KindLabel: "rama del cambio", PRTarget: "master"}
	branchResult.SetPromotionStage(PromotionStageMain)
	data, err := json.Marshal(branchResult)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"promotionStage":"main"`, `"promotionStageLabel":"hacia la rama principal"`, `"prTarget":"master"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("el JSON debe contener %s: %s", want, data)
		}
	}
	if parts := branchResult.textParts(); !strings.Contains(strings.Join(parts, ", "), "siguiente PR hacia master") {
		t.Errorf("textParts = %v", parts)
	}
	if without, _ := json.Marshal(BranchResult{Name: "a"}); strings.Contains(string(without), "promotionStage") {
		t.Errorf("sin etapa el JSON no la lleva: %s", without)
	}
}
