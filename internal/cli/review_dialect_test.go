package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// The dialect tests below prove the reading half of the axiom dialect: every
// negotiated decision point treats axiom.review-integration/v2 exactly like
// gentle-ai.review-integration/v2, while v1 keeps its frozen behavior.

func TestIsReviewContractV2(t *testing.T) {
	for _, tt := range []struct {
		name     string
		contract string
		want     bool
	}{
		{name: "gentle-ai v2", contract: ReviewIntegrationContractV2, want: true},
		{name: "axiom v2", contract: AxiomReviewIntegrationContractV2, want: true},
		{name: "gentle-ai v1", contract: ReviewIntegrationContractV1},
		{name: "empty", contract: ""},
		{name: "unknown", contract: "gentle-ai.review-integration/v3"},
		{name: "surrounding whitespace", contract: " " + AxiomReviewIntegrationContractV2},
		{name: "other contract family", contract: reviewtransaction.AxiomReviewIntegrationConsentV3Contract},
		{name: "legacy other contract family", contract: reviewtransaction.LegacyReviewIntegrationConsentV3Contract},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReviewContractV2(tt.contract); got != tt.want {
				t.Fatalf("isReviewContractV2(%q) = %v, want %v", tt.contract, got, tt.want)
			}
		})
	}
}

func TestReviewCommandCanonicalTool(t *testing.T) {
	for _, tt := range []struct{ name, command, want string }{
		{name: "gentle-ai unchanged", command: "gentle-ai review status --contract=x", want: "gentle-ai review status --contract=x"},
		{name: "axiom rewritten", command: "axiom review status --contract=x", want: "gentle-ai review status --contract=x"},
		{name: "bare tool name is not a command", command: "axiom", want: "axiom"},
		{name: "tool must lead", command: "run axiom review status", want: "run axiom review status"},
		{name: "empty", command: "", want: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := reviewCommandCanonicalTool(tt.command); got != tt.want {
				t.Fatalf("reviewCommandCanonicalTool(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

// TestDualContract_FreshStatusParityAcrossV2Dialects negotiates the same fresh
// candidate under {gentle-ai v2, axiom v2, v1}: both v2 dialects get the START
// transition with the exact consent relay and the forecast; v1 keeps its
// frozen shape without either.
func TestDualContract_FreshStatusParityAcrossV2Dialects(t *testing.T) {
	reviewEnabledHome(t)
	repo := initReviewCLIRepo(t)
	writeReviewStartCandidate(t, repo, "scripts/deploy.sh", "echo deploy\n", 0o644)

	var gentle, axiom ReviewTargetStatusResult
	for _, tt := range []struct {
		name, contract string
		v2             bool
	}{
		{name: "gentle-ai v2", contract: ReviewIntegrationContractV2, v2: true},
		{name: "axiom v2", contract: AxiomReviewIntegrationContractV2, v2: true},
		{name: "v1", contract: ReviewIntegrationContractV1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status := negotiatedStartStatusForContract(t, repo, tt.contract, "--lineage", "dialect-fresh-status")
			if status.Contract != tt.contract {
				t.Fatalf("status contract = %q, want %q", status.Contract, tt.contract)
			}
			if err := status.Validate(); err != nil {
				t.Fatalf("status does not validate: %v", err)
			}
			command := status.NextTransition.Execute.Command
			if !strings.Contains(command, " --contract="+tt.contract+" ") {
				t.Fatalf("START command %q does not carry the negotiated contract", command)
			}
			args := []string{"cwd", "contract", "target", "target-evidence", "projection", "lineage"}
			if tt.v2 {
				args = append(args, "consent")
				if !strings.HasSuffix(command, " --consent=relay") {
					t.Fatalf("v2 START command %q lacks the exact consent relay", command)
				}
				if status.Forecast == nil {
					t.Fatal("v2 status lost its forecast")
				}
			} else if strings.Contains(command, "--consent") || status.Forecast != nil {
				t.Fatalf("v1 status gained v2 surface: command %q forecast %#v", command, status.Forecast)
			}
			assertStartTransition(t, status, args)
			switch tt.contract {
			case ReviewIntegrationContractV2:
				gentle = status
			case AxiomReviewIntegrationContractV2:
				axiom = status
			}
		})
	}
	if gentle.Action != axiom.Action || !reflect.DeepEqual(gentle.Forecast, axiom.Forecast) ||
		gentle.NextTransition.Kind != axiom.NextTransition.Kind || gentle.NextTransition.ReasonCode != axiom.NextTransition.ReasonCode {
		t.Fatalf("v2 dialects diverge: gentle-ai %#v / axiom %#v", gentle, axiom)
	}
}

// runNegotiatedReviewStartForContract is runNegotiatedReviewStartWith with the
// negotiated contract as a parameter.
func runNegotiatedReviewStartForContract(t *testing.T, repo, contract, lineage string) ReviewIntegrationStartResult {
	t.Helper()
	var output bytes.Buffer
	if err := RunReview(boundNegotiatedStartArgs(t, []string{
		"start", "--contract", contract, "--cwd", repo, "--lineage", lineage,
	}), &output); err != nil {
		t.Fatal(negotiatedReviewStartFailure(err, output.String()))
	}
	result := decodeNegotiatedReviewStart(t, output.Bytes())
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
	return result
}

// TestDualContract_ReviewingLifecycleParityAcrossV2Dialects drives START,
// its exact replay, and the continuation STATUS of a reviewing lineage under
// both v2 dialects, and the v1 compatibility START beside them.
func TestDualContract_ReviewingLifecycleParityAcrossV2Dialects(t *testing.T) {
	reviewEnabledHome(t)

	type shape struct {
		reason   string
		kind     string
		action   reviewtransaction.TargetStatusAction
		forecast *ReviewForecast
	}
	shapes := map[string]shape{}
	for _, tt := range []struct{ name, contract string }{
		{name: "gentle-ai v2", contract: ReviewIntegrationContractV2},
		{name: "axiom v2", contract: AxiomReviewIntegrationContractV2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := initReviewCLIRepo(t)
			writeReviewStartCandidate(t, repo, "candidate.go", "package candidate\n\nfunc Candidate() int { return 1 }\n", 0o644)

			started := runNegotiatedReviewStartForContract(t, repo, tt.contract, "dialect-lifecycle")
			if started.Action != "created" || started.State != reviewtransaction.StateReviewing || started.RepositoryContext == nil {
				t.Fatalf("START = %#v", started)
			}
			transition := started.NextTransition
			if transition == nil || transition.Kind != reviewNextTransitionExecute || transition.Execute == nil || transition.Execute.Operation != "review.status" {
				t.Fatalf("reviewing START continuation = %#v", transition)
			}
			if replayed := runNegotiatedReviewStartForContract(t, repo, tt.contract, "dialect-lifecycle"); replayed.Action != "replayed" || replayed.LineageID != started.LineageID {
				t.Fatalf("exact replay = %#v", replayed)
			}

			// Run the published continuation tokens under this dialect.
			args := []string{"status"}
			for _, argument := range transition.Execute.Arguments {
				token := argument.Token
				if argument.Name == "contract" {
					token = "--contract=" + tt.contract
				}
				args = append(args, token)
			}
			args = append(args, "--cwd="+repo)
			var output bytes.Buffer
			if err := RunReview(args, &output); err != nil {
				t.Fatalf("continuation STATUS: %v\n%s", err, output.String())
			}
			var status ReviewTargetStatusResult
			decodeStrictReviewJSON(t, output.Bytes(), &status)
			if err := status.Validate(); err != nil {
				t.Fatalf("continuation STATUS does not validate: %v", err)
			}
			if status.Contract != tt.contract || status.Authority == nil || status.Authority.LineageID != started.LineageID ||
				status.Authority.State != reviewtransaction.StateReviewing {
				t.Fatalf("continuation STATUS authority = %#v (contract %q)", status.Authority, status.Contract)
			}
			if status.RepositoryContext == nil || status.NextTransition == nil || status.Forecast == nil {
				t.Fatalf("v2 continuation STATUS lost repository_context/next_transition/forecast: %#v", status)
			}
			shapes[tt.name] = shape{reason: status.NextTransition.ReasonCode, kind: status.NextTransition.Kind, action: status.Action, forecast: status.Forecast}
		})
	}
	if !reflect.DeepEqual(shapes["gentle-ai v2"], shapes["axiom v2"]) {
		t.Fatalf("v2 dialect lifecycle shapes diverge: %#v", shapes)
	}

	t.Run("v1 keeps the frozen compatibility shape", func(t *testing.T) {
		repo := initReviewCLIRepo(t)
		writeReviewStartCandidate(t, repo, "candidate.go", "package candidate\n\nfunc Candidate() int { return 1 }\n", 0o644)
		started := runNegotiatedReviewStartForContract(t, repo, ReviewIntegrationContractV1, "dialect-lifecycle-v1")
		if started.NextTransition != nil {
			t.Fatalf("v1 START published a next_transition: %#v", started.NextTransition)
		}
		if resumed := runNegotiatedReviewStartForContract(t, repo, ReviewIntegrationContractV1, "dialect-lifecycle-v1"); resumed.Action == "replayed" {
			t.Fatalf("v1 START replay reported the v2 action: %#v", resumed)
		}
	})
}

// TestDualContract_AxiomConsentRelayGetsTheV2Question is the regression for
// the axiom contract falling through to the v1 consent question: negotiating
// axiom.review-integration/v2 with --consent relay must return the same
// typed v3 question gentle-ai v2 gets, and its answers must run and replay.
func TestDualContract_AxiomConsentRelayGetsTheV2Question(t *testing.T) {
	reviewEnabledHome(t)
	stubReviewConsole(t, false, "")

	type questionShape struct {
		schema, headline, reason, value, offPath string
		risk                                     reviewtransaction.RiskLevel
		evidence                                 []string
	}
	questions := map[string]questionShape{}
	for _, tt := range []struct{ name, contract string }{
		{name: "gentle-ai v2", contract: ReviewIntegrationContractV2},
		{name: "axiom v2", contract: AxiomReviewIntegrationContractV2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := initReviewCLIRepo(t)
			writeReviewStartCandidate(t, repo, "scripts/deploy.sh", "echo deploy\n", 0o644)

			output := runConsentRelayStart(t, boundNegotiatedStartArgs(t, []string{
				"start", "--contract", tt.contract, "--cwd", repo,
				"--lineage", "dialect-consent", "--consent", "relay",
			}))
			question := decodeConsentQuestion(t, output.Bytes())
			if question.Schema != ReviewIntegrationConsentSchemaV3 || question.Contract != ReviewIntegrationContractV2 || question.Agent != "claude-code" {
				t.Fatalf("%s relay question identity = %#v, want the v3 question", tt.contract, question)
			}
			if err := question.Validate(); err != nil {
				t.Fatal(err)
			}
			for _, choice := range question.Choices {
				if !strings.Contains(choice.Invocation, " --contract "+tt.contract+" ") {
					t.Fatalf("choice %q invocation %q lost the negotiated contract", choice.Answer, choice.Invocation)
				}
			}
			validatePublishedReviewSchema(t, compileWholePublishedReviewSchema(t, "v2", "consent-v3.schema.json"), output.Bytes())

			questions[tt.name] = questionShape{
				schema: question.Schema, headline: question.Headline, reason: question.Reason, value: question.Value,
				offPath: question.OffPath.Note, risk: question.RiskLevel, evidence: question.RiskEvidence,
			}
			// Answering the question is already proven for gentle-ai v2 by the
			// consent relay tests; only the axiom answer is new here.
			if tt.contract != AxiomReviewIntegrationContractV2 {
				return
			}
			grantArgs := invocationArgs(t, question.Choices[0].Invocation)
			started := decodeNegotiatedReviewStart(t, runConsentRelayStart(t, grantArgs).Bytes())
			if started.Action != "created" || started.LineageID != "dialect-consent" || started.RiskLevel != reviewtransaction.RiskHigh {
				t.Fatalf("granted consent did not reach the lens plan: %#v", started)
			}
			if replayed := decodeNegotiatedReviewStart(t, runConsentRelayStart(t, grantArgs).Bytes()); replayed.Action != "replayed" || replayed.LineageID != started.LineageID {
				t.Fatalf("replayed grant = %#v", replayed)
			}
		})
	}
	if !reflect.DeepEqual(questions["gentle-ai v2"], questions["axiom v2"]) {
		t.Fatalf("consent questions diverge across the v2 dialects: %#v", questions)
	}
}

// TestDualContract_ConsentValidateAcceptsEveryV2Dialect proves the validator
// accepts a consent question in any combination of command tool and contract,
// and still refuses a tool or contract that is neither dialect.
func TestDualContract_ConsentValidateAcceptsEveryV2Dialect(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "contracts", "review-integration", "v2", "fixtures", "consent-v3.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	consentSchema := compileWholePublishedReviewSchema(t, "v2", "consent-v3.schema.json")

	for _, tt := range []struct {
		name, tool, contract string
		valid                bool
	}{
		{name: "gentle-ai tool, gentle-ai contract", tool: "gentle-ai", contract: ReviewIntegrationContractV2, valid: true},
		{name: "axiom tool, axiom contract", tool: "axiom", contract: AxiomReviewIntegrationContractV2, valid: true},
		{name: "axiom tool, gentle-ai contract", tool: "axiom", contract: ReviewIntegrationContractV2, valid: true},
		{name: "gentle-ai tool, axiom contract", tool: "gentle-ai", contract: AxiomReviewIntegrationContractV2, valid: true},
		{name: "unknown tool", tool: "other", contract: ReviewIntegrationContractV2},
		{name: "v1 contract on a v3 question", tool: "gentle-ai", contract: ReviewIntegrationContractV1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var question ReviewIntegrationConsentResult
			if err := json.Unmarshal(fixture, &question); err != nil {
				t.Fatal(err)
			}
			question.Contract = tt.contract
			for index := range question.Choices {
				question.Choices[index].Invocation = strings.Replace(question.Choices[index].Invocation, "gentle-ai review start ", tt.tool+" review start ", 1)
				question.Choices[index].Invocation = strings.Replace(question.Choices[index].Invocation, "--contract "+ReviewIntegrationContractV2, "--contract "+tt.contract, 1)
			}
			question.OffPath.Command = strings.Replace(question.OffPath.Command, "gentle-ai review", tt.tool+" review", 1)
			err := question.Validate()
			if tt.valid && err != nil {
				t.Fatalf("Validate rejected %s: %v", tt.name, err)
			}
			if !tt.valid && err == nil {
				t.Fatalf("Validate accepted %s", tt.name)
			}
			if tt.valid {
				payload, marshalErr := json.Marshal(question)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				validatePublishedReviewSchema(t, consentSchema, payload)
			}
		})
	}
}

// TestDualContract_V2SchemasAdmitTheAxiomDialect pins the in-place widening of
// the published v2 schemas: the consent question and the capabilities
// bootstrap admit the axiom command tool and contract next to the gentle-ai
// ones, and still refuse a mixed-up or unknown command.
func TestDualContract_V2SchemasAdmitTheAxiomDialect(t *testing.T) {
	const (
		gentleBootstrap = "gentle-ai review status --cwd <repo> --contract gentle-ai.review-integration/v2 --next-transition"
		axiomBootstrap  = "axiom review status --cwd <repo> --contract axiom.review-integration/v2 --next-transition"
	)
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2", "fixtures")
	withBootstrap := func(t *testing.T, fixture string, command string) []byte {
		t.Helper()
		payload, err := os.ReadFile(filepath.Join(root, fixture))
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(payload, &document); err != nil {
			t.Fatal(err)
		}
		document["bootstrap"].(map[string]any)["command"] = command
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	for _, tt := range []struct{ schema, fixture string }{
		{schema: "capabilities.schema.json", fixture: "capabilities.fixture.json"},
		{schema: "capabilities-v2.1.schema.json", fixture: "capabilities-v2.1.fixture.json"},
		{schema: "capabilities-v2.2.schema.json", fixture: "capabilities-v2.2.fixture.json"},
		{schema: "capabilities-v2.3.schema.json", fixture: "capabilities-v2.3.fixture.json"},
	} {
		t.Run(tt.schema, func(t *testing.T) {
			schema := compileWholePublishedReviewSchema(t, "v2", tt.schema)
			validatePublishedReviewSchema(t, schema, withBootstrap(t, tt.fixture, gentleBootstrap))
			validatePublishedReviewSchema(t, schema, withBootstrap(t, tt.fixture, axiomBootstrap))
			for _, mixed := range []string{
				"axiom review status --cwd <repo> --contract v1 --next-transition",
				"other review status --cwd <repo> --contract axiom.review-integration/v2 --next-transition",
			} {
				var document any
				if err := json.Unmarshal(withBootstrap(t, tt.fixture, mixed), &document); err != nil {
					t.Fatal(err)
				}
				if err := schema.Validate(document); err == nil {
					t.Fatalf("%s admitted the bootstrap command %q", tt.schema, mixed)
				}
			}
		})
	}

	// The derived v2.4-v2.6 schemas inherit the widening through $ref.
	surface := reviewCapabilitiesStaticSurface(AxiomReviewIntegrationContractV2)
	surface.Bootstrap.Command = axiomBootstrap
	surface.Package = ReviewCapabilitiesPackage{Name: "gentle-ai", Version: "3.1.0", ReleaseChannel: "stable"}
	surface.Build = ReviewCapabilitiesBuild{
		ID:            reviewCapabilitiesBuildDigest("3.1.0", ReviewCapabilitiesBuild{GoVersion: "go1.25.0", VCSModified: "false"}),
		GoVersion:     "go1.25.0",
		ModuleVersion: "v3.1.0",
		VCS:           "git",
		VCSRevision:   "0123456789abcdef0123456789abcdef01234567",
		VCSTime:       "2026-09-25T12:00:00Z",
		VCSModified:   "false",
	}
	surface.Executable = ReviewCapabilitiesExecutable{
		SHA256:       "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Evidence:     "self-reported",
		Verification: "compare-with-published-manifest",
	}
	payload, err := json.Marshal(surface)
	if err != nil {
		t.Fatal(err)
	}
	validatePublishedReviewSchema(t, compileWholePublishedReviewSchema(t, "v2", "capabilities-v2.6.schema.json"), payload)

	// The consent off-path admits both tools and the deliberate off command only.
	consent, err := os.ReadFile(filepath.Join(root, "consent-v3.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	consentSchema := compileWholePublishedReviewSchema(t, "v2", "consent-v3.schema.json")
	for command, admitted := range map[string]bool{
		"gentle-ai review mode disable": true,
		"axiom review mode disable":     true,
		"axiom review mode enable":      false,
		"other review mode disable":     false,
	} {
		var document map[string]any
		if err := json.Unmarshal(consent, &document); err != nil {
			t.Fatal(err)
		}
		document["off_path"].(map[string]any)["command"] = command
		var instance any = document
		if err := consentSchema.Validate(instance); (err == nil) != admitted {
			t.Fatalf("consent off_path command %q admitted = %v, want %v (%v)", command, err == nil, admitted, err)
		}
	}
}
