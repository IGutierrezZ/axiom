package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

func derivedRangeTerminalStatus(t *testing.T, repo string) ReviewTargetStatusResult {
	t.Helper()
	var output bytes.Buffer
	if err := RunReview([]string{"status", "--cwd", repo, "--contract", ReviewIntegrationContractV2, "--next-transition"}, &output); err != nil {
		t.Fatalf("derived range STATUS: %v\n%s", err, output.String())
	}
	var status ReviewTargetStatusResult
	decodeStrictReviewJSON(t, output.Bytes(), &status)
	validatePublishedReviewSchema(t, compileWholeNativeStatusSchema(t, "status-v9.schema.json"), output.Bytes())
	return status
}

// TestSelectorlessStatusDoesNotRestartEscalatedCandidate reproduces the
// operator's exact emitted START, not a hand-assembled substitute.
func TestSelectorlessStatusDoesNotRestartEscalatedCandidate(t *testing.T) {
	reviewEnabledHome(t)
	repo := initReviewCLIRepo(t)
	writeReviewStartCandidate(t, repo, "tracked.txt", "base\none\ntwo\nthree\nfour\n", 0o644)
	// v1 emits a direct START; v2 relays candidate consent before START.
	fresh := negotiatedStartStatusForContract(t, repo, ReviewIntegrationContractV1)
	started := executeStartTransition(t, repo, fresh)
	lineage := started.LineageID
	if lineage != startTransitionArgumentValue(t, fresh, "lineage") {
		t.Fatalf("START lineage = %q, offered %q", lineage, startTransitionArgumentValue(t, fresh, "lineage"))
	}
	legacyStarted := ReviewFacadeStartResult{
		LineageID: lineage, TargetIdentity: negotiatedStartTarget(started), SelectedLenses: started.SelectedLenses,
	}
	args := cliReviewerCaptureArgs(t, repo, legacyStarted, 0, []facadeFinding{{
		Location: "tracked.txt:5", Severity: "CRITICAL", Claim: "candidate regression",
		ProofRefs: []string{"tracked.txt:5 changed hunk"}, EvidenceClass: reviewtransaction.EvidenceDeterministic,
		CausalDisposition: reviewtransaction.CausalIntroduced,
	}})
	if err := RunReviewCaptureResult(args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	store, err := reviewtransaction.CompactAuthoritativeStore(t.Context(), repo, lineage)
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := reviewtransaction.BuildCorrectionPlanRequest(record.State, record.State.CapturePhaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	if err := RunReviewCaptureCorrectionPlan([]string{
		"--cwd", repo, "--lineage", lineage, "--target", plan.TargetIdentity,
		"--expected-revision", record.State.CapturePhaseRevision, "--request-hash", plan.RequestHash, "--correction-lines", "2",
	}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("base\none\ntwo\nthree\nfixed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	record, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	request, err := reviewtransaction.BuildTargetedValidationRequest(t.Context(), repo, record.State, record.State.CapturePhaseRevision)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(facadeValidationResult{
		TargetedValidationRequestHash: request.RequestHash, CorrectionTargetIdentity: request.CorrectionTargetIdentity,
		OriginalCriteria:     facadeValidationCheck{Passed: false, Evidence: []string{"acceptance test failed"}},
		CorrectionRegression: facadeValidationCheck{Passed: true, Evidence: []string{"no regression observed"}},
		FollowUps:            []reviewtransaction.FollowUp{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(reviewPiHostRelayContractEnvironment, reviewPiHostRelayContract)
	resultFile := writeReviewCLIRawInput(t, payload)
	if err := RunReviewCaptureValidation([]string{
		"--cwd", repo, "--lineage", lineage, "--target", request.CorrectionTargetIdentity,
		"--expected-revision", record.State.CapturePhaseRevision, "--request-hash", request.RequestHash,
		"--agent", string(model.AgentPi), "--input", resultFile,
	}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	record, err = store.Load()
	if err != nil || record.State.State != reviewtransaction.StateEscalated {
		t.Fatalf("terminal authority = %#v, %v; want escalated", record, err)
	}

	// Correction changed the live candidate. Restore the exact original bytes
	// so selectorless STATUS assesses the candidate that occupied this lineage.
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("base\none\ntwo\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	status := runSelectorlessNegotiatedStatus(t, repo)
	if status.TargetIdentity != record.State.InitialSnapshot.Identity {
		t.Fatalf("restored target = %q, want frozen initial target %q", status.TargetIdentity, record.State.InitialSnapshot.Identity)
	}
	if status.Applicability != reviewtransaction.TargetApplicabilityCurrent || status.Authority == nil ||
		status.Authority.LineageID != lineage || status.Authority.State != reviewtransaction.StateEscalated {
		t.Fatalf("restored original target lost terminal authority: %#v", status)
	}
	if status.NextTransition == nil {
		t.Fatal("selectorless STATUS omitted the terminal transition")
	}
	if status.NextTransition.Execute == nil {
		if status.NextTransition.Kind != reviewNextTransitionStop || status.NextTransition.ReasonCode != "native_stop_required" {
			t.Fatalf("escalated authority did not offer a truthful terminal stop: %#v", status.NextTransition)
		}
		return
	}
	if status.NextTransition.ReasonCode != "fresh_target_ready" || status.NextTransition.Execute.Operation != "review.start" ||
		status.NextTransition.Execute.Binding.LineageID != lineage {
		t.Fatalf("unexpected executable STATUS after escalation: %#v", status.NextTransition)
	}
	var output bytes.Buffer
	emitted := []string{"start"}
	for _, argument := range status.NextTransition.Execute.Arguments {
		emitted = append(emitted, argument.Token)
	}
	if err := RunReview(emitted, &output); err != nil {
		t.Fatalf("emitted START consent relay: %v\n%s", err, output.String())
	}
	question := decodeConsentQuestion(t, output.Bytes())
	if question.Action != "consent_required" || question.TargetIdentity != status.TargetIdentity {
		t.Fatalf("emitted START did not relay consent for the same candidate: %#v", question)
	}
	invocation := ""
	for _, choice := range question.Choices {
		if choice.Answer == "granted" {
			invocation = choice.Invocation
			break
		}
	}
	if invocation == "" {
		t.Fatalf("consent relay omitted granted invocation: %#v", question)
	}
	output.Reset()
	err = RunReview(invocationArgs(t, invocation), &output)
	if err == nil {
		t.Fatalf("granted START unexpectedly succeeded: %s", output.String())
	}
	failure := decodeReviewIntegrationFailure(t, output.Bytes())
	if failure.Code != "atomic_start_conflict" || failure.MutationOutcome != ReviewMutationNotStarted {
		t.Fatalf("granted START failure = %#v, want atomic_start_conflict/not_started (error %v)", failure, err)
	}
	t.Fatalf("selectorless STATUS advertised fresh_target_ready for escalated lineage %q and target %q; exact granted START failed %s (%s)", lineage, status.TargetIdentity, failure.Code, failure.MutationOutcome)
}

func TestNextTransitionDerivedRangeAcknowledgementStaysTerminal(t *testing.T) {
	reviewEnabledHome(t)
	repo := initReviewStopHookRepo(t)
	runReviewCLIGit(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	runReviewCLIGit(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	writeZeroLensDocumentationCandidate(t, repo)
	runReviewCLIGit(t, repo, "add", "docs/ordinary-guide.md")
	runReviewCLIGit(t, repo, "commit", "-qm", "documentation candidate")
	offered := derivedRangeTerminalStatus(t, repo)
	if offered.NextTransition == nil || offered.NextTransition.Execute == nil {
		t.Fatalf("no committed-range START: %#v", offered)
	}
	args := []string{"start"}
	for _, argument := range offered.NextTransition.Execute.Arguments {
		args = append(args, argument.Token)
	}
	var output bytes.Buffer
	if err := RunReview(args, &output); err != nil {
		t.Fatalf("execute offered START: %v\n%s", err, output.String())
	}
	pending := derivedRangeTerminalStatus(t, repo)
	if pending.TargetIdentity != offered.TargetIdentity || pending.NextTransition == nil || pending.NextTransition.ReasonCode != "approved_acknowledgement_required" {
		t.Fatalf("derived range lost pending acknowledgement: %#v", pending)
	}
	args = []string{"acknowledge-approved"}
	for _, argument := range pending.NextTransition.Execute.Arguments {
		args = append(args, argument.Token)
	}
	output.Reset()
	if err := RunReview(args, &output); err != nil {
		t.Fatal(err)
	}
	// A new hook session has no reminder history to hide a duplicate START.
	payload := reviewStopHookTestPayload(t, "derived-range-session", repo, false, nil)
	output.Reset()
	var diagnostics bytes.Buffer
	if err := runReviewStopHook([]string{"--agent", "claude-code"}, strings.NewReader(payload), &output, &diagnostics); err != nil || output.Len() != 0 {
		t.Fatalf("derived range Stop = %s, %v", output.String(), err)
	}
	after := derivedRangeTerminalStatus(t, repo)
	if after.TargetIdentity != offered.TargetIdentity || after.NextTransition == nil || after.NextTransition.ReasonCode != "target_already_acknowledged" || after.Authority != nil {
		t.Fatalf("derived range not terminal: %#v", after)
	}
	consumed, err := reviewtransaction.CompactTargetConsumed(context.Background(), repo, offered.TargetIdentity)
	if err != nil || !consumed {
		t.Fatalf("derived range consumption = %v, %v", consumed, err)
	}
	// #4739: a later commit that only adds passive content to the acknowledged
	// candidate changes the identity but has nothing new to review. STATUS
	// stops without authority and the Stop hook stays silent.
	writeReviewStartCandidate(t, repo, "docs/ordinary-guide.md", "new committed range\n", 0o644)
	runReviewCLIGit(t, repo, "add", "docs/ordinary-guide.md")
	runReviewCLIGit(t, repo, "commit", "-qm", "passive follow-up")
	passive := derivedRangeTerminalStatus(t, repo)
	if passive.TargetIdentity == offered.TargetIdentity || passive.NextTransition == nil || passive.NextTransition.Execute != nil ||
		passive.NextTransition.ReasonCode != "acknowledged_predecessor_passive_delta" || passive.Authority != nil {
		t.Fatalf("passive delta after acknowledgement re-offered review: %#v", passive)
	}
	payload = reviewStopHookTestPayload(t, "derived-range-passive-session", repo, false, nil)
	output.Reset()
	if err := runReviewStopHook([]string{"--agent", "claude-code"}, strings.NewReader(payload), &output, &diagnostics); err != nil || output.Len() != 0 {
		t.Fatalf("passive delta Stop = %s, %v", output.String(), err)
	}
	if consumed, err := reviewtransaction.CompactTargetConsumed(context.Background(), repo, passive.TargetIdentity); err != nil || consumed {
		t.Fatalf("suppressed passive delta recorded as consumed = %v, %v", consumed, err)
	}
	// Anything non-passive after the acknowledgement is offered for review.
	writeReviewStartCandidate(t, repo, "cmd/app/main.go", "package main\n\nfunc main() {}\n", 0o644)
	runReviewCLIGit(t, repo, "add", "cmd/app/main.go")
	runReviewCLIGit(t, repo, "commit", "-qm", "code follow-up")
	changed := derivedRangeTerminalStatus(t, repo)
	if changed.TargetIdentity == passive.TargetIdentity || changed.NextTransition == nil || changed.NextTransition.Execute == nil || changed.NextTransition.Execute.Operation != "review.start" {
		t.Fatalf("non-passive committed range suppressed: %#v", changed)
	}
}

// grantedDerivedRangeStart executes STATUS's offered START and, for a medium
// or high candidate, the consent envelope's exact granted invocation.
func grantedDerivedRangeStart(t *testing.T, offered ReviewTargetStatusResult) ReviewFacadeStartResult {
	t.Helper()
	args := []string{"start"}
	for _, argument := range offered.NextTransition.Execute.Arguments {
		args = append(args, argument.Token)
	}
	var output bytes.Buffer
	if err := RunReview(args, &output); err != nil {
		t.Fatalf("execute START: %v\n%s", err, output.String())
	}
	var consent struct {
		Choices []struct {
			Answer     string `json:"answer"`
			Invocation string `json:"invocation"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(output.Bytes(), &consent); err != nil {
		t.Fatalf("decode START output: %v\n%s", err, output.String())
	}
	for _, choice := range consent.Choices {
		if choice.Answer != "granted" {
			continue
		}
		output.Reset()
		if err := RunReview(invocationArgs(t, choice.Invocation), &output); err != nil {
			t.Fatalf("granted START: %v\n%s", err, output.String())
		}
		break
	}
	var started ReviewFacadeStartResult
	if err := json.Unmarshal(output.Bytes(), &started); err != nil || started.LineageID == "" {
		t.Fatalf("START created no lineage: %v\n%s", err, output.String())
	}
	return started
}

// #4739 must only withhold a fresh offer: a review someone deliberately
// started on a passive delta is live authority and must stay visible.
func TestNextTransitionPassiveDeltaNeverHidesALiveReview(t *testing.T) {
	reviewEnabledHome(t)
	repo := initReviewStopHookRepo(t)
	runReviewCLIGit(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	runReviewCLIGit(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	writeReviewStartCandidate(t, repo, "tracked.txt", "candidate\n", 0o644)
	runReviewCLIGit(t, repo, "commit", "-qam", "medium candidate")

	// Approve and acknowledge the medium candidate through its real lens.
	started := grantedDerivedRangeStart(t, derivedRangeTerminalStatus(t, repo))
	store, err := reviewtransaction.CompactAuthoritativeStore(context.Background(), repo, started.LineageID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "result.json")
	if err := os.WriteFile(input, admittedReviewerPayloadForTest(t, repo, record, record.State.SelectedLenses[0], 0), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := RunReviewCaptureResult([]string{"--cwd", repo, "--lineage", started.LineageID, "--target", record.State.InitialSnapshot.Identity,
		"--lens", record.State.SelectedLenses[0], "--order", "0", "--input", input}, &output); err != nil {
		t.Fatal(err)
	}
	var terminal reviewLastEventClosureResult
	decodeStrictReviewJSON(t, output.Bytes(), &terminal)
	if err := RunReview([]string{"acknowledge-approved", "--cwd", repo, "--lineage", started.LineageID, "--target", record.State.InitialSnapshot.Identity,
		"--expected-revision", terminal.StoreRevision, "--token", terminal.Acknowledgement.Arguments[4].Value}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	writeReviewStartCandidate(t, repo, "docs/notes.md", "Tracking note.\n", 0o644)
	runReviewCLIGit(t, repo, "add", "docs/notes.md")
	runReviewCLIGit(t, repo, "commit", "-qm", "passive follow-up")
	passive := derivedRangeTerminalStatus(t, repo)
	if passive.NextTransition == nil || passive.NextTransition.ReasonCode != "acknowledged_predecessor_passive_delta" {
		t.Fatalf("passive delta after a medium acknowledgement re-offered review: %#v", passive.NextTransition)
	}

	// A deliberate review of the full range is live authority: the START a
	// maintainer builds from this STOP's own fields.
	deliberate := passive
	deliberate.NextTransition = &ReviewNextTransition{Kind: reviewNextTransitionExecute, Execute: &ReviewTransitionExecution{}}
	for _, token := range []string{"--cwd=" + repo, "--contract=" + ReviewIntegrationContractV2, "--agent=claude-code", "--target=" + passive.TargetIdentity,
		"--projection=" + string(passive.Projection.Projection), "--base-ref=" + passive.Projection.BaseTree, "--committed-only", "--consent=relay"} {
		deliberate.NextTransition.Execute.Arguments = append(deliberate.NextTransition.Execute.Arguments, ReviewTransitionArgument{Token: token})
	}
	live := grantedDerivedRangeStart(t, deliberate)
	status := derivedRangeTerminalStatus(t, repo)
	if status.NextTransition == nil || status.NextTransition.ReasonCode == "acknowledged_predecessor_passive_delta" {
		t.Fatalf("live review %s hidden behind the passive-delta STOP: %#v", live.LineageID, status.NextTransition)
	}
	payload := reviewStopHookTestPayload(t, "live-review-session", repo, false, nil)
	output.Reset()
	if err := runReviewStopHook([]string{"--agent", "claude-code"}, strings.NewReader(payload), &output, &bytes.Buffer{}); err != nil || output.Len() == 0 {
		t.Fatalf("Stop hook stayed silent over live review %s: %s, %v", live.LineageID, output.String(), err)
	}
}
