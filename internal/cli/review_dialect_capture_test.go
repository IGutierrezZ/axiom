package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewerprovider"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// reviewV2DialectCase is one of the two spellings of the v2 lifecycle a caller
// can negotiate. The capture-* and acknowledge-approved commands run the exact
// tokens STATUS issued, so the tokens of an axiom caller lead with --contract and
// the tokens of a gentle-ai caller carry none: these cases replay each.
type reviewV2DialectCase struct {
	name     string
	contract string
	dialect  reviewDialect
}

var reviewV2DialectCases = []reviewV2DialectCase{
	{name: "gentle-ai v2", contract: ReviewIntegrationContractV2},
	{name: "axiom v2", contract: AxiomReviewIntegrationContractV2, dialect: reviewDialect{axiom: true}},
}

// captureExtra are the extra argv words a capture command receives for this
// dialect: --contract for axiom, nothing for gentle-ai.
func (tt reviewV2DialectCase) captureExtra() []string {
	if !tt.dialect.axiom {
		return nil
	}
	return []string{"--contract", tt.contract}
}

// assertOnlyDialect fails when payload names the other dialect's tool or v2
// contract identifier.
func (tt reviewV2DialectCase) assertOnlyDialect(t *testing.T, name string, payload []byte) {
	t.Helper()
	if tt.dialect.axiom {
		assertNoForeignDialect(t, name, payload, "gentle-ai review", ReviewIntegrationContractV2)
		return
	}
	assertNoForeignDialect(t, name, payload, "axiom review", AxiomReviewIntegrationContractV2)
}

// stripAxiomCaptureContract removes the one leading --contract argument and
// token an axiom caller's native capture inputs carry, so a STATUS can be
// compared with the gentle-ai one it otherwise equals.
func stripAxiomCaptureContract(t *testing.T, status *ReviewTargetStatusResult) {
	t.Helper()
	if status.NextTransition == nil || status.NextTransition.Collect == nil {
		return
	}
	lead := reviewDialect{axiom: true}.captureContractArguments()[0]
	for index := range status.NextTransition.Collect.Inputs {
		input := &status.NextTransition.Collect.Inputs[index]
		if _, native := reviewNativeCaptureVerb(input.CaptureOperation); !native {
			continue
		}
		if len(input.Arguments) == 0 || input.Arguments[0].Name != lead.Name || input.Arguments[0].Value != lead.Value ||
			input.Arguments[0].Token != reviewTransitionArgumentToken(lead) {
			t.Fatalf("capture input does not lead with the contract: %#v", input.Arguments)
		}
		input.Arguments = input.Arguments[1:]
		if submission := input.Submission; submission != nil {
			if len(submission.ArgumentTokens) == 0 || submission.ArgumentTokens[0] != reviewTransitionArgumentToken(lead) || submission.Value == nil {
				t.Fatalf("capture submission does not lead with the contract: %#v", submission)
			}
			submission.ArgumentTokens = submission.ArgumentTokens[1:]
			submission.Value.SubstitutionLocation--
		}
	}
}

func reviewExecutionTokens(execution *ReviewTransitionExecution) []string {
	tokens := make([]string, 0, len(execution.Arguments))
	for _, argument := range execution.Arguments {
		tokens = append(tokens, argument.Token)
	}
	return tokens
}

// TestDualContract_CaptureAcknowledgementIsTheOneRestartedStatusReoffers drives
// one lineage per dialect to approval. The final capture runs the tokens STATUS
// issued to that caller, so an axiom caller passes --contract and a gentle-ai
// caller does not; the acknowledgement it returns must be, field for field and
// command for command, the one the restarted STATUS re-offers, and replaying it
// verbatim burns the authority.
func TestDualContract_CaptureAcknowledgementIsTheOneRestartedStatusReoffers(t *testing.T) {
	for _, tt := range reviewV2DialectCases {
		t.Run(tt.name, func(t *testing.T) {
			reviewEnabledHome(t)
			repo := initReviewCLIRepo(t)
			started := startHighRiskCLIReview(t, repo)
			store, err := reviewtransaction.CompactAuthoritativeStore(context.Background(), repo, started.LineageID)
			if err != nil {
				t.Fatal(err)
			}
			statusArgs := []string{"status", "--cwd", repo, "--lineage", started.LineageID, "--contract", tt.contract, "--next-transition"}
			statusSchema := compileWholeNativeStatusSchema(t, "status-v9.schema.json")

			// The capture tokens of the reviewing authority lead with --contract
			// only for the axiom dialect, and the whole STATUS stays in one dialect.
			var reviewing bytes.Buffer
			if err := RunReview(statusArgs, &reviewing); err != nil {
				t.Fatalf("reviewing STATUS: %v\n%s", err, reviewing.String())
			}
			var reviewingStatus ReviewTargetStatusResult
			decodeStrictReviewJSON(t, reviewing.Bytes(), &reviewingStatus)
			if err := reviewingStatus.Validate(); err != nil {
				t.Fatalf("reviewing STATUS does not validate: %v", err)
			}
			if reviewingStatus.NextTransition == nil || reviewingStatus.NextTransition.Collect == nil || len(reviewingStatus.NextTransition.Collect.Inputs) == 0 {
				t.Fatalf("reviewing STATUS transition = %#v, want the reviewer collection", reviewingStatus.NextTransition)
			}
			for _, input := range reviewingStatus.NextTransition.Collect.Inputs {
				if input.CaptureOperation != reviewCaptureResultCaptureOperation {
					t.Fatalf("collect input operation = %q, want capture-result", input.CaptureOperation)
				}
				lead := tt.dialect.captureContractArguments()
				if len(lead) == 0 {
					for _, argument := range input.Arguments {
						if argument.Name == "contract" {
							t.Fatalf("%s capture input carries a contract token: %#v", tt.name, input.Arguments)
						}
					}
				} else if input.Arguments[0].Name != lead[0].Name || input.Arguments[0].Value != lead[0].Value ||
					input.Arguments[0].Token != reviewTransitionArgumentToken(lead[0]) {
					t.Fatalf("%s capture input does not lead with its contract token: %#v", tt.name, input.Arguments)
				}
			}
			validatePublishedReviewSchema(t, statusSchema, reviewing.Bytes())
			tt.assertOnlyDialect(t, tt.name+" reviewing STATUS", reviewing.Bytes())

			last := len(started.SelectedLenses) - 1
			for order := 0; order < last; order++ {
				captureCLIReviewerResultWithFindings(t, repo, started, order, []facadeFinding{}, &bytes.Buffer{}, tt.captureExtra()...)
			}
			var terminalOutput bytes.Buffer
			captureCLIReviewerResultWithFindings(t, repo, started, last, []facadeFinding{}, &terminalOutput, tt.captureExtra()...)
			validatePublishedReviewSchema(t, compileWholePublishedReviewSchema(t, "v2", "last-event-closure.schema.json"), terminalOutput.Bytes())
			var terminal reviewLastEventClosureResult
			decodeStrictReviewJSON(t, terminalOutput.Bytes(), &terminal)
			if terminal.State != reviewtransaction.StateApproved || terminal.StatusContinuation != nil {
				t.Fatalf("last capture terminal result = %#v, want approved pending acknowledgement", terminal)
			}
			assertApprovedAcknowledgementTransitionFor(t, tt.dialect, terminal.Acknowledgement, repo, started.LineageID, started.TargetIdentity, terminal.StoreRevision)
			if want := tt.dialect.Tool() + " review acknowledge-approved "; !strings.HasPrefix(terminal.Acknowledgement.Command, want) {
				t.Fatalf("acknowledgement command = %q, want the %s tool", terminal.Acknowledgement.Command, tt.dialect.Tool())
			}
			tt.assertOnlyDialect(t, tt.name+" terminal capture", terminalOutput.Bytes())

			var restartOutput bytes.Buffer
			if err := RunReview(statusArgs, &restartOutput); err != nil {
				t.Fatalf("restart status: %v\n%s", err, restartOutput.String())
			}
			var restarted ReviewTargetStatusResult
			decodeStrictReviewJSON(t, restartOutput.Bytes(), &restarted)
			if err := restarted.Validate(); err != nil {
				t.Fatalf("restart STATUS does not validate: %v", err)
			}
			if restarted.NextTransition == nil || restarted.NextTransition.Kind != reviewNextTransitionExecute ||
				restarted.NextTransition.ReasonCode != "approved_acknowledgement_required" {
				t.Fatalf("restart status transition = %#v, want pending acknowledgement", restarted.NextTransition)
			}
			if !reflect.DeepEqual(terminal.Acknowledgement, restarted.NextTransition.Execute) {
				t.Fatalf("restarted STATUS acknowledgement differs from the capture one:\ncapture=%#v\nstatus=%#v", terminal.Acknowledgement, restarted.NextTransition.Execute)
			}
			validatePublishedReviewSchema(t, statusSchema, restartOutput.Bytes())
			tt.assertOnlyDialect(t, tt.name+" restarted STATUS", restartOutput.Bytes())

			// A foreign contract is refused before the burn and leaves the
			// authority replayable.
			tokens := reviewExecutionTokens(terminal.Acknowledgement)
			if tt.dialect.axiom {
				tokens = tokens[1:]
			}
			if err := RunReview(append([]string{"acknowledge-approved", "--contract=" + ReviewIntegrationContractV1}, tokens...), &bytes.Buffer{}); err == nil ||
				!strings.Contains(err.Error(), "accepts only") {
				t.Fatalf("acknowledge-approved with a v1 contract = %v, want the contract refusal", err)
			}
			if _, err := store.Load(); err != nil {
				t.Fatalf("a refused contract mutated the authority: %v", err)
			}

			var acknowledged bytes.Buffer
			if err := RunReview(append([]string{"acknowledge-approved"}, reviewExecutionTokens(terminal.Acknowledgement)...), &acknowledged); err != nil {
				t.Fatalf("replay the issued acknowledgement verbatim: %v", err)
			}
			assertAcknowledgedEnvelope(t, acknowledged.Bytes(), started.LineageID, started.TargetIdentity, terminal.StoreRevision)
			assertApprovedCompactAuthorityBurned(t, store, started.LineageID)
		})
	}
}

// TestDualContract_CorrectionContinuationKeepsTheNegotiatedContract proves the
// re-entry after a severe final event does not switch the caller's contract: the
// status_continuation of a capture run with the tokens of either dialect names
// that dialect's tool and contract, runs verbatim into a STATUS answered in the
// same dialect, whose correction-plan tokens are in turn executable.
func TestDualContract_CorrectionContinuationKeepsTheNegotiatedContract(t *testing.T) {
	for _, tt := range reviewV2DialectCases {
		t.Run(tt.name, func(t *testing.T) {
			reviewEnabledHome(t)
			repo := initReviewCLIRepo(t)
			started := startHighRiskCLIReview(t, repo)
			last := len(started.SelectedLenses) - 1
			for order := 0; order < last; order++ {
				captureCLIReviewerResultWithFindings(t, repo, started, order, []facadeFinding{}, &bytes.Buffer{}, tt.captureExtra()...)
			}
			var correctionOutput bytes.Buffer
			captureCLIReviewerResultWithFindings(t, repo, started, last, []facadeFinding{{
				ID: "R3-001", Location: "internal/auth/session.go:4", Severity: "CRITICAL",
				Claim:         "the candidate introduces an observable authentication failure",
				ProofRefs:     []string{"the changed line deterministically causes the reproduced failure"},
				EvidenceClass: reviewtransaction.EvidenceDeterministic, CausalDisposition: reviewtransaction.CausalIntroduced,
			}}, &correctionOutput, tt.captureExtra()...)
			validatePublishedReviewSchema(t, compileWholePublishedReviewSchema(t, "v2", "last-event-closure.schema.json"), correctionOutput.Bytes())
			var closure reviewLastEventClosureResult
			decodeStrictReviewJSON(t, correctionOutput.Bytes(), &closure)
			if closure.State != reviewtransaction.StateCorrectionRequired || closure.StatusContinuation == nil {
				t.Fatalf("last capture correction result = %#v, want the status continuation", closure)
			}
			continuation := closure.StatusContinuation
			var contract string
			for _, argument := range continuation.Arguments {
				if argument.Name == "contract" {
					contract = argument.Value
				}
			}
			if contract != tt.contract {
				t.Fatalf("status continuation contract = %q, want the negotiated %q", contract, tt.contract)
			}
			if want := tt.dialect.Tool() + " review status "; !strings.HasPrefix(continuation.Command, want) {
				t.Fatalf("status continuation command = %q, want the %s tool", continuation.Command, tt.dialect.Tool())
			}
			tt.assertOnlyDialect(t, tt.name+" correction closure", correctionOutput.Bytes())

			// Published schemas load relative to the package directory, so compile
			// them before the capture below needs the repository as process cwd.
			statusSchema := compileWholeNativeStatusSchema(t, "status-v9.schema.json")
			t.Chdir(repo)
			var statusOutput bytes.Buffer
			if err := RunReview(append([]string{"status"}, reviewExecutionTokens(continuation)...), &statusOutput); err != nil {
				t.Fatalf("run the status continuation verbatim: %v\n%s", err, statusOutput.String())
			}
			var status ReviewTargetStatusResult
			decodeStrictReviewJSON(t, statusOutput.Bytes(), &status)
			if err := status.Validate(); err != nil {
				t.Fatalf("continuation STATUS does not validate: %v", err)
			}
			if status.Contract != tt.contract {
				t.Fatalf("continuation STATUS contract = %q, want the negotiated %q", status.Contract, tt.contract)
			}
			validatePublishedReviewSchema(t, statusSchema, statusOutput.Bytes())
			tt.assertOnlyDialect(t, tt.name+" continuation STATUS", statusOutput.Bytes())
			if status.NextTransition == nil || status.NextTransition.ReasonCode != "correction_plan_required" ||
				status.NextTransition.Collect == nil || len(status.NextTransition.Collect.Inputs) != 1 {
				t.Fatalf("continuation STATUS transition = %#v, want the correction plan collection", status.NextTransition)
			}
			input := status.NextTransition.Collect.Inputs[0]
			lead := tt.dialect.captureContractArguments()
			if len(lead) != 0 && (input.Arguments[0].Name != lead[0].Name || input.Arguments[0].Value != lead[0].Value) {
				t.Fatalf("correction-plan input does not lead with the contract: %#v", input.Arguments)
			}
			if input.Submission == nil {
				t.Fatalf("correction-plan input has no submission descriptor: %#v", input)
			}
			if (len(input.Submission.ArgumentTokens) == 7) != tt.dialect.axiom {
				t.Fatalf("correction-plan submission tokens = %v, want a leading --contract token only for axiom", input.Submission.ArgumentTokens)
			}

			// The issued submission, executed as STATUS rendered it, records the plan.
			plan := make([]string, 0, len(input.Submission.ArgumentTokens)+1)
			plan = append(plan, input.Submission.OperationToken)
			for _, token := range input.Submission.ArgumentTokens {
				plan = append(plan, strings.ReplaceAll(token, reviewSubmissionValuePlaceholder, "2"))
			}
			var planOutput bytes.Buffer
			if err := RunReview(plan, &planOutput); err != nil {
				t.Fatalf("run the issued correction-plan submission: %v\n%s", err, planOutput.String())
			}
			var planned reviewCorrectionPlanCaptureResult
			decodeStrictReviewJSON(t, planOutput.Bytes(), &planned)
			if planned.Operation != reviewCaptureCorrectionPlanOperation || planned.State != reviewtransaction.StateCorrectionRequired || planned.CorrectionLines != 2 {
				t.Fatalf("correction-plan capture = %#v", planned)
			}
		})
	}
}

// TestDualContract_CaptureCommandsAcceptOnlyAV2Contract proves the optional
// --contract of the six commands: v1, an unknown value, and an empty value are
// refused before anything is read or written, a v2 contract in either spelling
// is accepted, and the failure envelope of a capture-* refusal follows the
// dialect the invocation named.
func TestDualContract_CaptureCommandsAcceptOnlyAV2Contract(t *testing.T) {
	collectVerbs := []string{"capture-result", "capture-refuter", "capture-validation", "capture-correction-plan", "capture-unachievable"}
	for _, verb := range append(append([]string{}, collectVerbs...), "acknowledge-approved") {
		for _, contract := range []string{ReviewIntegrationContractV1, "other/v9", ""} {
			t.Run(verb+" refuses "+contract, func(t *testing.T) {
				var output bytes.Buffer
				err := RunReview([]string{verb, "--contract=" + contract}, &output)
				if err == nil || !strings.Contains(err.Error(), "accepts only") {
					t.Fatalf("%s --contract=%q = %v, want the contract refusal", verb, contract, err)
				}
				if verb == "acknowledge-approved" {
					return
				}
				var failure ReviewIntegrationFailure
				decodeStrictReviewJSON(t, output.Bytes(), &failure)
				if err := failure.Validate(); err != nil || failure.Schema != ReviewIntegrationFailureSchemaV2 || failure.Contract != ReviewIntegrationContractV2 {
					t.Fatalf("%s refusal envelope = %#v (%v), want the gentle-ai v2 identity", verb, failure, err)
				}
			})
		}
	}
	for _, verb := range collectVerbs {
		t.Run(verb+" answers in the axiom dialect", func(t *testing.T) {
			var output bytes.Buffer
			err := RunReview([]string{verb, "--contract=" + AxiomReviewIntegrationContractV2}, &output)
			if err == nil || strings.Contains(err.Error(), "accepts only") {
				t.Fatalf("%s with the axiom contract and no binding = %v, want the missing-binding refusal", verb, err)
			}
			var failure ReviewIntegrationFailure
			decodeStrictReviewJSON(t, output.Bytes(), &failure)
			if err := failure.Validate(); err != nil || failure.Schema != reviewtransaction.AxiomReviewFailureV2Contract || failure.Contract != AxiomReviewIntegrationContractV2 {
				t.Fatalf("%s refusal envelope = %#v (%v), want the axiom v2 identity", verb, failure, err)
			}
		})
	}
	// The legacy spelling is accepted too and keeps the gentle-ai identity.
	var output bytes.Buffer
	if err := RunReview([]string{"capture-refuter", "--contract=" + ReviewIntegrationContractV2}, &output); err == nil || strings.Contains(err.Error(), "accepts only") {
		t.Fatalf("capture-refuter with the gentle-ai v2 contract = %v, want the missing-binding refusal", err)
	}
	var failure ReviewIntegrationFailure
	decodeStrictReviewJSON(t, output.Bytes(), &failure)
	if failure.Schema != ReviewIntegrationFailureSchemaV2 || failure.Contract != ReviewIntegrationContractV2 {
		t.Fatalf("gentle-ai v2 refusal envelope = %#v", failure)
	}
}

// TestDualContract_ProducersLeadCaptureTokensWithTheContractOnlyForAxiom pins
// every producer of capture tokens at once: the reviewer, refuter, validator and
// correction-plan inputs and descriptors, the unachievable-slot withdraw, the
// acknowledgement and the correction status continuation. Each leads with the
// contract only under the axiom dialect, validates in its own, and is refused
// when read under the other.
func TestDualContract_ProducersLeadCaptureTokensWithTheContractOnlyForAxiom(t *testing.T) {
	t.Setenv(reviewPiHostRelayContractEnvironment, reviewPiHostRelayContract)
	binding := ReviewTransitionBinding{
		LineageID: "dialect-producers", Revision: descriptorTestSHA("a"), TargetIdentity: descriptorTestSHA("b"),
		RepositoryContext: "rctx1_" + strings.Repeat("c", 64),
	}
	validation := &reviewtransaction.TargetedValidationRequest{RequestHash: descriptorTestSHA("d")}
	request := reviewtransaction.CorrectionPlanRequest{
		LineageID: binding.LineageID, ExpectedRevision: binding.Revision, TargetIdentity: binding.TargetIdentity,
		RequestHash: descriptorTestSHA("d"), CorrectionBudget: 7,
	}
	for _, tt := range reviewV2DialectCases {
		other := reviewV2DialectCases[0]
		if tt.dialect.axiom == other.dialect.axiom {
			other = reviewV2DialectCases[1]
		}
		t.Run(tt.name, func(t *testing.T) {
			wantLead := tt.dialect.captureContractArguments()
			assertLead := func(label string, arguments []ReviewTransitionArgument) {
				t.Helper()
				if len(wantLead) == 0 {
					for _, argument := range arguments {
						if argument.Name == "contract" {
							t.Fatalf("%s carries a contract argument: %#v", label, arguments)
						}
					}
					return
				}
				if len(arguments) == 0 || arguments[0].Name != wantLead[0].Name || arguments[0].Value != wantLead[0].Value {
					t.Fatalf("%s does not lead with the contract: %#v", label, arguments)
				}
			}
			assertTokens := func(label string, tokens []string, valueSlot int) {
				t.Helper()
				if len(wantLead) == 0 {
					for _, token := range tokens {
						if strings.HasPrefix(token, "--contract=") {
							t.Fatalf("%s carries a contract token: %v", label, tokens)
						}
					}
				} else if tokens[0] != reviewTransitionArgumentToken(wantLead[0]) {
					t.Fatalf("%s does not lead with the contract token: %v", label, tokens)
				}
				if valueSlot >= 0 && !strings.Contains(tokens[valueSlot], reviewSubmissionValuePlaceholder) {
					t.Fatalf("%s value slot %d does not point at the placeholder: %v", label, valueSlot, tokens)
				}
			}
			// readsAs validates the transition under this dialect and refuses it
			// under the other: a result never carries the other dialect's tokens.
			readsAs := func(label string, transition ReviewNextTransition) {
				t.Helper()
				if err := validateReviewDialectCaptureTokens(tt.contract, &transition); err != nil {
					t.Fatalf("%s is refused under its own dialect: %v", label, err)
				}
				if err := validateReviewDialectCaptureTokens(other.contract, &transition); err == nil {
					t.Fatalf("%s is accepted under the other dialect", label)
				}
			}

			refuter := reviewProviderRoleTransition(tt.dialect, "provider_refuter_required", binding, reviewerprovider.RoleRefuter, model.AgentClaudeCode, nil)
			if err := refuter.Validate(); err != nil {
				t.Fatalf("compiled refuter transition: %v", err)
			}
			assertLead("compiled refuter", refuter.Collect.Inputs[0].Arguments)
			readsAs("compiled refuter", refuter)

			relayRefuter := reviewProviderRoleTransition(tt.dialect, "provider_refuter_required", binding, reviewerprovider.RoleRefuter, model.AgentPi, nil)
			if err := relayRefuter.Validate(); err != nil {
				t.Fatalf("host-relay refuter transition: %v", err)
			}
			assertLead("host-relay refuter", relayRefuter.Collect.Inputs[0].Arguments)
			relayInput := relayRefuter.Collect.Inputs[0]
			assertTokens("host-relay refuter submission", relayInput.Submission.ArgumentTokens, relayInput.Submission.Value.SubstitutionLocation)
			readsAs("host-relay refuter", relayRefuter)

			relayValidator, err := reviewProviderRoleMaterializeSubmissionInput(tt.dialect, binding, reviewerprovider.RoleTargetedValidator, model.AgentPi, validation)
			if err != nil {
				t.Fatal(err)
			}
			assertLead("host-relay validator", relayValidator.Arguments)
			assertTokens("host-relay validator submission", relayValidator.Submission.ArgumentTokens, relayValidator.Submission.Value.SubstitutionLocation)
			if err := relayValidator.Submission.Validate(); err != nil {
				t.Fatalf("host-relay validator descriptor: %v", err)
			}
			compiledValidator, err := reviewProviderCompiledRoleExecuteInput(tt.dialect, binding, reviewerprovider.RoleTargetedValidator, model.AgentClaudeCode, validation)
			if err != nil {
				t.Fatal(err)
			}
			assertLead("compiled validator", compiledValidator.Arguments)

			lens := reviewCaptureInput(tt.dialect, binding, reviewtransaction.LensReliability, 0, nil, model.AgentPi)
			assertLead("host-relay lens", lens.Arguments)
			assertTokens("host-relay lens submission", lens.Submission.ArgumentTokens, lens.Submission.Value.SubstitutionLocation)
			if err := lens.Submission.Validate(); err != nil {
				t.Fatalf("host-relay lens descriptor: %v", err)
			}
			readsAs("host-relay lens", reviewCollectTransition("reviewer_results_required", lens))

			plan := reviewCorrectionPlanSubmission(tt.contract, binding, request)
			if err := plan.Validate(); err != nil {
				t.Fatalf("correction-plan descriptor: %v", err)
			}
			assertTokens("correction-plan submission", plan.ArgumentTokens, plan.Value.SubstitutionLocation)
			readsAs("correction-plan input", reviewCollectTransition("correction_plan_required", ReviewTransitionInput{
				Name: "correction_lines", Schema: "gentle-ai.review-correction-plan/v1", CaptureOperation: reviewCaptureCorrectionPlanOperation,
				Arguments: tt.dialect.withCaptureContract([]ReviewTransitionArgument{{Name: "lineage", Value: binding.LineageID}}), Submission: plan,
			}))

			slot := reviewUnachievableLensSlotEntry(tt.dialect, binding, reviewtransaction.CompactUnachievableLensAttempt{
				Lens: reviewtransaction.LensReliability, SelectedOrder: 0, SubjectHash: descriptorTestSHA("e"), Reason: "relay_transport_bound_exceeded",
			})
			assertLead("unachievable withdraw", slot.Withdraw.Arguments)
			if want := tt.dialect.Tool() + " review capture-unachievable "; !strings.HasPrefix(slot.Withdraw.Command, want) {
				t.Fatalf("withdraw command = %q, want the %s tool", slot.Withdraw.Command, tt.dialect.Tool())
			}
			if len(wantLead) != 0 && !strings.Contains(slot.Withdraw.Command, " --contract="+tt.contract+" ") {
				t.Fatalf("withdraw command = %q, want the contract token", slot.Withdraw.Command)
			}
			withdraw := reviewStopTransition("unachievable_lens_slot")
			withdraw.UnachievableLensSlots = &[]ReviewUnachievableLensSlot{slot}
			readsAs("unachievable withdraw", withdraw)

			acknowledgement := reviewApprovedAcknowledgementTransition(tt.dialect, "/repo", reviewtransaction.ApprovedCompactAcknowledgement{
				LineageID: binding.LineageID, TargetIdentity: binding.TargetIdentity, ExpectedRevision: binding.Revision, Token: strings.Repeat("a", 64),
			})
			assertLead("acknowledgement", acknowledgement.Arguments)
			if err := validateReviewApprovedAcknowledgementExecution(*acknowledgement); err != nil {
				t.Fatalf("acknowledgement does not validate: %v", err)
			}
			readsAs("acknowledgement", ReviewNextTransition{Kind: reviewNextTransitionExecute, ReasonCode: "approved_acknowledgement_required", Execute: acknowledgement})

			continuation := reviewCorrectionStatusContinuation(tt.dialect, "/repo", reviewtransaction.CompactState{
				LineageID:       binding.LineageID,
				InitialSnapshot: reviewtransaction.Snapshot{Kind: reviewtransaction.TargetCurrentChanges, Identity: binding.TargetIdentity},
			}, binding.Revision, "")
			if continuation == nil || continuation.Arguments[1].Name != "contract" || continuation.Arguments[1].Value != tt.contract {
				t.Fatalf("correction status continuation = %#v, want the negotiated contract", continuation)
			}
		})
	}
}

// TestDirectReviewStartKeepsTheLegacyDialectOnItsAcknowledgement pins the START
// that negotiates nothing: its zero-lens acknowledgement names the gentle-ai
// tool, carries no contract token, and mentions no axiom command, exactly as it
// did before the axiom dialect existed.
func TestDirectReviewStartKeepsTheLegacyDialectOnItsAcknowledgement(t *testing.T) {
	reviewEnabledHome(t)
	repo := initReviewCLIRepo(t)
	writeZeroLensDocumentationCandidate(t, repo)
	var output bytes.Buffer
	if err := RunReviewFacadeStart([]string{"--cwd", repo}, &output); err != nil {
		t.Fatalf("direct zero-lens START: %v\n%s", err, output.String())
	}
	var started struct {
		Acknowledgement *ReviewTransitionExecution `json:"acknowledgement"`
	}
	if err := json.Unmarshal(output.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	acknowledgement := started.Acknowledgement
	if acknowledgement == nil || len(acknowledgement.Arguments) != 5 ||
		!strings.HasPrefix(acknowledgement.Command, "gentle-ai review acknowledge-approved ") || acknowledgement.Arguments[0].Name != "cwd" {
		t.Fatalf("direct START acknowledgement = %#v, want the legacy gentle-ai command with five arguments", acknowledgement)
	}
	assertNoForeignDialect(t, "direct START", output.Bytes(), "axiom review", AxiomReviewIntegrationContractV2, "--contract=")
}

// TestDualContract_OpenCodeTaskBindingsCarryTheDialectOnlyForAxiom pins the
// opaque Task bindings of the OpenCode relay, which holds no token list: the
// binding names the dialect only for axiom (a gentle-ai binding keeps its exact
// bytes), the transport decodes it back and rebuilds the very same Go-issued
// prompt, and a binding naming any other contract is refused.
func TestDualContract_OpenCodeTaskBindingsCarryTheDialectOnlyForAxiom(t *testing.T) {
	binding := ReviewTransitionBinding{
		LineageID: "dialect-opencode", Revision: descriptorTestSHA("a"), TargetIdentity: descriptorTestSHA("b"),
		RepositoryContext: "rctx1_" + strings.Repeat("c", 64),
	}
	subject := &reviewtransaction.ArtifactSubject{
		LineageID: binding.LineageID, AuthorityRevision: binding.Revision, TargetIdentity: binding.TargetIdentity,
		Lens: reviewtransaction.LensReliability, SubjectHash: descriptorTestSHA("e"), SelectedOrder: 0,
	}
	// assertPromptContract checks the contract field only exists for axiom.
	assertPromptContract := func(t *testing.T, label, prompt string, axiom bool) {
		t.Helper()
		if has := strings.Contains(prompt, `"contract":"`+AxiomReviewIntegrationContractV2+`"`); has != axiom {
			t.Fatalf("%s prompt = %q, contract present = %v, want %v", label, prompt, has, axiom)
		}
		if !axiom && strings.Contains(prompt, "contract") {
			t.Fatalf("%s prompt of the legacy dialect mentions a contract: %q", label, prompt)
		}
	}
	for _, tt := range reviewV2DialectCases {
		t.Run(tt.name, func(t *testing.T) {
			role, err := newReviewProviderTask(tt.dialect, reviewerprovider.RoleRefuter, binding)
			if err != nil {
				t.Fatal(err)
			}
			assertPromptContract(t, "role task", role.Prompt, tt.dialect.axiom)
			decoded, err := decodeOpenCodeTransportBinding(role.Prompt)
			if err != nil || decoded.Contract != tt.dialect.bindingContract() || decoded.canonicalTaskPrompt != role.Prompt {
				t.Fatalf("decoded role binding = %#v (%v), want the dialect echoed and the issued prompt rebuilt", decoded, err)
			}
			forgedRole := strings.Replace(role.Prompt, `"role":"refuter"`, `"role":"refuter","contract":"`+ReviewIntegrationContractV2+`"`, 1)
			if tt.dialect.axiom {
				forgedRole = strings.Replace(role.Prompt, AxiomReviewIntegrationContractV2, ReviewIntegrationContractV2, 1)
			}
			if _, err := decodeOpenCodeTransportBinding(forgedRole); err == nil {
				t.Fatalf("a role binding naming the contract %s was admitted: %q", ReviewIntegrationContractV2, forgedRole)
			}

			arguments := tt.dialect.withCaptureContract([]ReviewTransitionArgument{
				{Name: "lineage", Value: binding.LineageID}, {Name: "expected-revision", Value: binding.Revision},
				{Name: "target", Value: binding.TargetIdentity}, {Name: "repository-context", Value: binding.RepositoryContext},
				{Name: "lens", Value: subject.Lens}, {Name: "order", Value: "0"}, {Name: "subject-hash", Value: subject.SubjectHash},
			})
			lens, err := newReviewLensProviderTask(arguments, subject)
			if err != nil {
				t.Fatal(err)
			}
			assertPromptContract(t, "lens task", lens.Prompt, tt.dialect.axiom)
			decodedLens, err := decodeOpenCodeTransportBinding(lens.Prompt)
			if err != nil || decodedLens.Contract != tt.dialect.bindingContract() || decodedLens.SubjectHash != subject.SubjectHash {
				t.Fatalf("decoded lens binding = %#v (%v), want the dialect echoed", decodedLens, err)
			}
			unknown := strings.TrimSuffix(strings.Replace(lens.Prompt, AxiomReviewIntegrationContractV2, "other/v9", 1), "}")
			if !tt.dialect.axiom {
				unknown += `,"contract":"other/v9"`
			}
			if _, err := decodeOpenCodeTransportBinding(unknown + "}"); err == nil {
				t.Fatalf("a lens binding naming an unknown contract was admitted: %q", unknown)
			}
			if session := (openCodeTransportSession{binding: decodedLens}); session.dialect() != tt.dialect {
				t.Fatalf("session dialect = %#v, want %#v", session.dialect(), tt.dialect)
			}
		})
	}
}
