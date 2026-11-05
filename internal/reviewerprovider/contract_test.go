package reviewerprovider

import (
	"strings"
	"testing"
)

// TestRuntimeBudgetLeavesContractResultLimitsUnchanged pins the output side of
// the byte policy: the runtime-context cap is an input budget owned by the
// runtime declaration, and it must never shrink Contract.ResultLimit, which
// owns the raw provider OUTPUT admission limit for every role.
func TestRuntimeBudgetLeavesContractResultLimitsUnchanged(t *testing.T) {
	for _, contract := range Contracts() {
		if contract.ResultLimit != 4<<20 {
			t.Fatalf("provider role %q ResultLimit = %d, want the unchanged %d byte output limit", contract.Role, contract.ResultLimit, 4<<20)
		}
	}
}

func TestTargetedValidatorContractDefinesPassedPolarity(t *testing.T) {
	contract, err := ContractFor(RoleTargetedValidator)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{"result schema", string(contract.ResultSchema), "true means the named check passed; false means the named check failed."},
		{"original criteria", contract.PromptInstruction, "Set `original_criteria.passed` to true only when every original criterion is met in the corrected candidate; set it to false when any original criterion remains unmet."},
		{"correction regression", contract.PromptInstruction, "Set `correction_regression.passed` to true only when the correction caused no regression; set it to false when you observe a regression."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.payload, tt.want) {
				t.Fatalf("targeted validator %s does not define passed polarity: missing %q", tt.name, tt.want)
			}
		})
	}
}

// TestTargetedValidatorContractDoesNotPromiseOmittedGeneratedContent pins the
// briefing against the evidence this role is actually handed.
// reviewProviderMaterializeEvidence gives it the same representation a lens
// receives, so a generated path arrives as a metadata summary with its content
// hunks omitted. A briefing that promised the complete patch for every path
// would have this role sign a verified verdict over bytes it never saw.
func TestTargetedValidatorContractDoesNotPromiseOmittedGeneratedContent(t *testing.T) {
	contract, err := ContractFor(RoleTargetedValidator)
	if err != nil {
		t.Fatal(err)
	}

	for _, forbidden := range []string{
		"already carries the complete frozen tree-to-tree patch",
		"for every path in `validation_request.correction_paths`. It is authoritative corrected-candidate content",
	} {
		if strings.Contains(contract.PromptInstruction, forbidden) {
			t.Fatalf("targeted validator briefing still promises complete content for every path: %q", forbidden)
		}
	}

	for _, required := range []string{
		`"content_omitted": true`,
		"Its content hunks are not in this input, so the summary alone never verifies a claim about what those hunks say.",
		"Use it whenever a check turns on a generated path's omitted content, because that content reaches you no other way.",
		"When a check turns on a generated path's omitted content and you cannot run that command, that check carries no verdict: mark it unavailable rather than reading one out of the summary.",
	} {
		if !strings.Contains(contract.PromptInstruction, required) {
			t.Fatalf("targeted validator briefing omits the generated-path route: missing %q", required)
		}
	}
}

// severityRulePhrases son las reglas concretas de severidad que toda superficie
// de revisor debe llevar. Las reglas concretas se mantienen estables entre
// modelos; las abstractas derivan.
var severityRulePhrases = []string{
	"must be caused by this change",
	"does not already happen at the baseline",
	"reachable with realistic input",
	"was not asked to change",
	"out-of-domain values",
	"at most WARNING",
	"ignoring an explicit option or argument while reporting success",
	"unrequested changes to existing command output or messages",
}

// observableHarmRulePhrases fijan la cualificación del daño observable: un
// hallazgo grave nombra su daño, el alcance por sí solo no es daño, una
// regresión no pedida sigue siendo grave sin una prohibición y un cambio pedido
// no es una regresión.
var observableHarmRulePhrases = []string{
	"must also name its observable harm",
	"a concrete violation of the requested behavior",
	"a regression on input or state that was valid at the baseline",
	"is not harm by itself and is at most WARNING",
	"at most WARNING unless the finding also shows a concrete violation of the requested behavior or a regression on input or state that was valid at the baseline.",
	"is a regression even when nothing prohibited it",
	"is not a regression merely because its results differ from the baseline",
}

// refuterHarmDecisionPhrases fijan la regla de decisión del refutador: refutar
// una afirmación sin daño observable y reservar inconclusive para la evidencia
// indecidible, porque abre una corrección en lugar de rebajar la severidad.
var refuterHarmDecisionPhrases = []string{
	"it demonstrates no observable harm",
	"Inconclusive is not a severity downgrade: it still opens a correction",
	"only when the supplied evidence cannot decide",
}

func TestSeverityRulesRequireObservableHarm(t *testing.T) {
	for _, required := range append(append([]string{}, severityRulePhrases...), observableHarmRulePhrases...) {
		if !strings.Contains(SeverityRules, required) {
			t.Fatalf("severity rules omit %q:\n%s", required, SeverityRules)
		}
	}
}

func TestRefuterPromptAppliesSeverityRules(t *testing.T) {
	contract, err := ContractFor(RoleRefuter)
	if err != nil {
		t.Fatal(err)
	}
	required := append(append([]string{}, severityRulePhrases...), observableHarmRulePhrases...)
	required = append(required, refuterHarmDecisionPhrases...)
	required = append(required, "Refute a BLOCKER or CRITICAL claim that fails these conditions", "inferential batch")
	for _, phrase := range required {
		if !strings.Contains(contract.PromptInstruction, phrase) {
			t.Fatalf("refuter prompt omits severity rule %q:\n%s", phrase, contract.PromptInstruction)
		}
	}
}

// TestRefuterPromptEchoesRequestHash fija #5597: la petición lleva request_hash,
// el esquema de resultado exige refuter_request_hash y la admisión los compara,
// así que el prompt debe nombrar la correspondencia.
func TestRefuterPromptEchoesRequestHash(t *testing.T) {
	contract, err := ContractFor(RoleRefuter)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Echo the input request_hash as refuter_request_hash."; !strings.Contains(contract.PromptInstruction, want) {
		t.Fatalf("refuter prompt omits %q:\n%s", want, contract.PromptInstruction)
	}
}
