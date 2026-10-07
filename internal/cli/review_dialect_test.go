package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// The dialect tests prove both halves of the axiom dialect: every negotiated
// decision point reads axiom.review-integration/v2 exactly like
// gentle-ai.review-integration/v2, and every producer echoes the dialect the
// caller negotiated, with v1 and gentle-ai v2 bytes untouched.

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

func TestReviewDialectForContract(t *testing.T) {
	for _, tt := range []struct {
		name, contract, tool, contractV2, refresh string
		axiom                                     bool
	}{
		{name: "gentle-ai v2", contract: ReviewIntegrationContractV2, tool: "gentle-ai", contractV2: ReviewIntegrationContractV2,
			refresh: reviewNextTransitionRefreshCommandV21},
		{name: "axiom v2", contract: AxiomReviewIntegrationContractV2, tool: "axiom", contractV2: AxiomReviewIntegrationContractV2,
			refresh: "axiom review status --cwd <repo> --contract axiom.review-integration/v2 --next-transition", axiom: true},
		{name: "v1 stays gentle-ai", contract: ReviewIntegrationContractV1, tool: "gentle-ai", contractV2: ReviewIntegrationContractV2,
			refresh: reviewNextTransitionRefreshCommandV21},
		{name: "no contract stays gentle-ai", contract: "", tool: "gentle-ai", contractV2: ReviewIntegrationContractV2,
			refresh: reviewNextTransitionRefreshCommandV21},
		{name: "unresolvable stays gentle-ai", contract: "other/v9", tool: "gentle-ai", contractV2: ReviewIntegrationContractV2,
			refresh: reviewNextTransitionRefreshCommandV21},
		{name: "axiom family member is not the v2 lifecycle", contract: reviewtransaction.AxiomReviewIntegrationConsentV3Contract,
			tool: "gentle-ai", contractV2: ReviewIntegrationContractV2, refresh: reviewNextTransitionRefreshCommandV21},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dialect := reviewDialectForContract(tt.contract)
			if dialect.axiom != tt.axiom || dialect.Tool() != tt.tool || dialect.ContractV2() != tt.contractV2 || dialect.refreshCommand() != tt.refresh {
				t.Fatalf("dialect(%q) = {axiom:%v tool:%q contract:%q refresh:%q}", tt.contract, dialect.axiom, dialect.Tool(), dialect.ContractV2(), dialect.refreshCommand())
			}
			if !dialect.ownsCommand(tt.tool+" review status") || dialect.ownsCommand("other review status") {
				t.Fatalf("dialect %q ownership is wrong", tt.tool)
			}
		})
	}
	if !reviewNoContractDialect.axiom {
		t.Fatal("the no-contract dialect must default to axiom")
	}
}

// swapReviewDialect rewrites gentle-ai output into its axiom spelling: the
// command tool and the v2 contract identifier, nothing else. Schema ids and
// every other byte are wire contract and must be equal across dialects.
func swapReviewDialect(payload []byte) []byte {
	return []byte(strings.NewReplacer(
		"gentle-ai review ", "axiom review ",
		ReviewIntegrationContractV2, AxiomReviewIntegrationContractV2,
	).Replace(string(payload)))
}

func assertNoForeignDialect(t *testing.T, name string, payload []byte, foreign ...string) {
	t.Helper()
	for _, needle := range foreign {
		if bytes.Contains(payload, []byte(needle)) {
			t.Fatalf("%s names %q:\n%s", name, needle, payload)
		}
	}
}

// reviewContinuationStatus runs a published STATUS continuation verbatim (its
// own tokens, the repository as --cwd) and returns the raw bytes.
func reviewContinuationStatus(t *testing.T, repo string, started ReviewIntegrationStartResult) []byte {
	t.Helper()
	if started.NextTransition == nil || started.NextTransition.Execute == nil {
		t.Fatalf("START published no status continuation: %#v", started)
	}
	args := []string{"status"}
	for _, argument := range started.NextTransition.Execute.Arguments {
		args = append(args, argument.Token)
	}
	var output bytes.Buffer
	if err := RunReview(append(args, "--cwd="+repo), &output); err != nil {
		t.Fatalf("continuation STATUS: %v\n%s", err, output.String())
	}
	return output.Bytes()
}

// TestDualContract_FreshStatusEchoesTheNegotiatedDialect negotiates one fresh
// candidate under {gentle-ai v2, axiom v2, v1}: gentle-ai v2 keeps its bytes,
// axiom v2 is the same command in the axiom spelling with no gentle-ai tool
// left, and v1 keeps its frozen shape (no consent relay, no forecast, no axiom).
func TestDualContract_FreshStatusEchoesTheNegotiatedDialect(t *testing.T) {
	reviewEnabledHome(t)
	repo := initReviewCLIRepo(t)
	writeReviewStartCandidate(t, repo, "scripts/deploy.sh", "echo deploy\n", 0o644)

	startCommand := func(status ReviewTargetStatusResult, tool, contract string, consent bool) string {
		command := tool + " review start" +
			" " + reviewTransitionShellWord("--cwd="+repo) +
			" --contract=" + contract +
			" --target=" + status.TargetIdentity +
			" --target-evidence=" + reviewStatusTargetEvidenceToken(status) +
			" --projection=workspace" +
			" --lineage=dialect-fresh-status"
		if consent {
			command += " --consent=relay"
		}
		return command
	}
	for _, tt := range []struct {
		name, contract, tool, wantContract string
		v2                                 bool
	}{
		{name: "gentle-ai v2", contract: ReviewIntegrationContractV2, tool: "gentle-ai", wantContract: ReviewIntegrationContractV2, v2: true},
		{name: "axiom v2", contract: AxiomReviewIntegrationContractV2, tool: "axiom", wantContract: AxiomReviewIntegrationContractV2, v2: true},
		{name: "v1", contract: ReviewIntegrationContractV1, tool: "gentle-ai", wantContract: ReviewIntegrationContractV1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status := negotiatedStartStatusForContract(t, repo, tt.contract, "--lineage", "dialect-fresh-status")
			if status.Contract != tt.wantContract {
				t.Fatalf("status contract = %q, want %q", status.Contract, tt.wantContract)
			}
			if err := status.Validate(); err != nil {
				t.Fatalf("status does not validate: %v", err)
			}
			names := []string{"cwd", "contract", "target", "target-evidence", "projection", "lineage"}
			if tt.v2 {
				names = append(names, "consent")
				if status.Forecast == nil {
					t.Fatal("v2 status lost its forecast")
				}
			} else if status.Forecast != nil {
				t.Fatalf("v1 status gained a forecast: %#v", status.Forecast)
			}
			assertStartTransition(t, status, names)
			if want := startCommand(status, tt.tool, tt.wantContract, tt.v2); status.NextTransition.Execute.Command != want {
				t.Fatalf("START command = %q, want %q", status.NextTransition.Execute.Command, want)
			}
			payload, err := json.Marshal(status)
			if err != nil {
				t.Fatal(err)
			}
			if tt.tool == "axiom" {
				assertNoForeignDialect(t, "axiom v2 fresh STATUS", payload, "gentle-ai review", ReviewIntegrationContractV2)
			} else {
				assertNoForeignDialect(t, tt.name+" fresh STATUS", payload, "axiom review", AxiomReviewIntegrationContractV2)
			}

			// Strict coherence: the same result with the other dialect's tool
			// is refused, so a result never mixes the two spellings.
			mixed := status
			transition := *status.NextTransition
			execution := *transition.Execute
			if tt.tool == "axiom" {
				execution.Command = "gentle-ai" + strings.TrimPrefix(execution.Command, "axiom")
			} else {
				execution.Command = "axiom" + strings.TrimPrefix(execution.Command, "gentle-ai")
			}
			transition.Execute = &execution
			mixed.NextTransition = &transition
			if err := mixed.Validate(); err == nil {
				t.Fatalf("%s STATUS accepted the other dialect's command %q", tt.name, execution.Command)
			}
		})
	}
}

// TestDualContract_V2DialectsAnswerOneAuthorityIdentically drives the typed
// consent question, the granted START, its exact replay, and the continuation
// STATUS of ONE lineage under both v2 dialects. The axiom answer to each step
// is exactly the gentle-ai answer with the tool and contract swapped (no other
// byte differs), and it never names a gentle-ai command.
func TestDualContract_V2DialectsAnswerOneAuthorityIdentically(t *testing.T) {
	reviewEnabledHome(t)
	stubReviewConsole(t, false, "")
	repo := initReviewCLIRepo(t)
	writeReviewStartCandidate(t, repo, "scripts/deploy.sh", "echo deploy\n", 0o644)

	relay := func(contract string) []byte {
		return runConsentRelayStart(t, boundNegotiatedStartArgs(t, []string{
			"start", "--contract", contract, "--cwd", repo, "--lineage", "dialect-authority", "--consent", "relay",
		})).Bytes()
	}
	gentleQuestionBytes, axiomQuestionBytes := relay(ReviewIntegrationContractV2), relay(AxiomReviewIntegrationContractV2)
	if want := swapReviewDialect(gentleQuestionBytes); !bytes.Equal(axiomQuestionBytes, want) {
		t.Fatalf("axiom consent question is not the gentle-ai one in the axiom spelling:\ngot=%s\nwant=%s", axiomQuestionBytes, want)
	}
	gentleQuestion, axiomQuestion := decodeConsentQuestion(t, gentleQuestionBytes), decodeConsentQuestion(t, axiomQuestionBytes)
	if gentleQuestion.Contract != ReviewIntegrationContractV2 || axiomQuestion.Contract != AxiomReviewIntegrationContractV2 ||
		axiomQuestion.Schema != ReviewIntegrationConsentSchemaV3 || axiomQuestion.OffPath.Command != "axiom review mode disable" {
		t.Fatalf("question identity: gentle-ai %#v / axiom %#v", gentleQuestion, axiomQuestion)
	}
	for _, question := range []ReviewIntegrationConsentResult{gentleQuestion, axiomQuestion} {
		if err := question.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	consentSchema := compileWholePublishedReviewSchema(t, "v2", "consent-v3.schema.json")
	validatePublishedReviewSchema(t, consentSchema, gentleQuestionBytes)
	validatePublishedReviewSchema(t, consentSchema, axiomQuestionBytes)
	assertNoForeignDialect(t, "axiom consent question", axiomQuestionBytes, "gentle-ai review", ReviewIntegrationContractV2)
	assertNoForeignDialect(t, "gentle-ai consent question", gentleQuestionBytes, "axiom review", AxiomReviewIntegrationContractV2)

	// The gentle-ai grant creates the authority; the axiom grant, run from the
	// axiom question's own invocation, replays it.
	gentleGrant := invocationArgs(t, gentleQuestion.Choices[0].Invocation)
	created := decodeNegotiatedReviewStart(t, runConsentRelayStart(t, gentleGrant).Bytes())
	if created.Action != "created" || created.LineageID != "dialect-authority" || created.RiskLevel != reviewtransaction.RiskHigh {
		t.Fatalf("granted START = %#v", created)
	}
	replayGentle := runConsentRelayStart(t, gentleGrant).Bytes()
	replayAxiom := runConsentRelayStart(t, invocationArgs(t, axiomQuestion.Choices[0].Invocation)).Bytes()
	if want := swapReviewDialect(replayGentle); !bytes.Equal(replayAxiom, want) {
		t.Fatalf("axiom START replay is not the gentle-ai one in the axiom spelling:\ngot=%s\nwant=%s", replayAxiom, want)
	}
	assertNoForeignDialect(t, "axiom START replay", replayAxiom, "gentle-ai review", ReviewIntegrationContractV2)
	assertNoForeignDialect(t, "gentle-ai START replay", replayGentle, "axiom review", AxiomReviewIntegrationContractV2)
	gentleStarted, axiomStarted := decodeNegotiatedReviewStart(t, replayGentle), decodeNegotiatedReviewStart(t, replayAxiom)
	if gentleStarted.Action != "replayed" || axiomStarted.Action != "replayed" || axiomStarted.Contract != AxiomReviewIntegrationContractV2 {
		t.Fatalf("replays: gentle-ai %#v / axiom %#v", gentleStarted.Action, axiomStarted)
	}
	for _, started := range []ReviewIntegrationStartResult{gentleStarted, axiomStarted} {
		if err := started.Validate(); err != nil {
			t.Fatal(err)
		}
	}

	// Each continuation runs verbatim and is answered in its own dialect.
	gentleStatus, axiomStatus := reviewContinuationStatus(t, repo, gentleStarted), reviewContinuationStatus(t, repo, axiomStarted)
	if want := swapReviewDialect(gentleStatus); !bytes.Equal(axiomStatus, want) {
		t.Fatalf("axiom continuation STATUS is not the gentle-ai one in the axiom spelling:\ngot=%s\nwant=%s", axiomStatus, want)
	}
	assertNoForeignDialect(t, "axiom continuation STATUS", axiomStatus, "gentle-ai review", ReviewIntegrationContractV2)
	assertNoForeignDialect(t, "gentle-ai continuation STATUS", gentleStatus, "axiom review", AxiomReviewIntegrationContractV2)
	var status ReviewTargetStatusResult
	decodeStrictReviewJSON(t, axiomStatus, &status)
	if err := status.Validate(); err != nil {
		t.Fatalf("axiom continuation STATUS does not validate: %v", err)
	}
	if status.Contract != AxiomReviewIntegrationContractV2 || status.RepositoryContext == nil || status.NextTransition == nil || status.Forecast == nil {
		t.Fatalf("axiom continuation STATUS = %#v", status)
	}
}

// TestDualContract_ConsentValidateRequiresACoherentDialect proves the consent
// validator accepts a question only when its command tool matches the dialect
// of the contract it names, in either dialect, and refuses a mixed one.
func TestDualContract_ConsentValidateRequiresACoherentDialect(t *testing.T) {
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
		{name: "axiom tool, gentle-ai contract", tool: "axiom", contract: ReviewIntegrationContractV2},
		{name: "gentle-ai tool, axiom contract", tool: "gentle-ai", contract: AxiomReviewIntegrationContractV2},
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

// TestDualContract_CapabilitiesBootstrapEchoesTheDialect pins the bootstrap
// command each dialect publishes and proves the published schemas admit it.
func TestDualContract_CapabilitiesBootstrapEchoesTheDialect(t *testing.T) {
	const axiomBootstrap = "axiom review status --cwd <repo> --contract axiom.review-integration/v2 --next-transition"
	for _, tt := range []struct{ contract, want string }{
		{contract: ReviewIntegrationContractV2, want: "gentle-ai review status --cwd <repo> --contract gentle-ai.review-integration/v2 --next-transition"},
		{contract: AxiomReviewIntegrationContractV2, want: axiomBootstrap},
	} {
		t.Run(tt.contract, func(t *testing.T) {
			surface := reviewCapabilitiesStaticSurface(tt.contract)
			if surface.Bootstrap == nil || surface.Bootstrap.Command != tt.want {
				t.Fatalf("bootstrap = %#v, want %q", surface.Bootstrap, tt.want)
			}
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
		})
	}
}

// TestDualContract_V2SchemasAdmitTheAxiomDialect pins the in-place widening of
// the older published v2 capabilities and consent schemas: both tools and
// contracts are admitted, and a mixed-up or unknown command is still refused.
func TestDualContract_V2SchemasAdmitTheAxiomDialect(t *testing.T) {
	const (
		gentleBootstrap = "gentle-ai review status --cwd <repo> --contract gentle-ai.review-integration/v2 --next-transition"
		axiomBootstrap  = "axiom review status --cwd <repo> --contract axiom.review-integration/v2 --next-transition"
	)
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2", "fixtures")
	withField := func(t *testing.T, fixture, object, command string) []byte {
		t.Helper()
		payload, err := os.ReadFile(filepath.Join(root, fixture))
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(payload, &document); err != nil {
			t.Fatal(err)
		}
		document[object].(map[string]any)["command"] = command
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
			validatePublishedReviewSchema(t, schema, withField(t, tt.fixture, "bootstrap", gentleBootstrap))
			validatePublishedReviewSchema(t, schema, withField(t, tt.fixture, "bootstrap", axiomBootstrap))
			for _, mixed := range []string{
				"axiom review status --cwd <repo> --contract v1 --next-transition",
				"other review status --cwd <repo> --contract axiom.review-integration/v2 --next-transition",
			} {
				var document any
				if err := json.Unmarshal(withField(t, tt.fixture, "bootstrap", mixed), &document); err != nil {
					t.Fatal(err)
				}
				if err := schema.Validate(document); err == nil {
					t.Fatalf("%s admitted the bootstrap command %q", tt.schema, mixed)
				}
			}
		})
	}

	// The consent off-path admits both tools and the deliberate off command only.
	consentSchema := compileWholePublishedReviewSchema(t, "v2", "consent-v3.schema.json")
	for command, admitted := range map[string]bool{
		"gentle-ai review mode disable": true,
		"axiom review mode disable":     true,
		"axiom review mode enable":      false,
		"other review mode disable":     false,
	} {
		var document any
		if err := json.Unmarshal(withField(t, "consent-v3.fixture.json", "off_path", command), &document); err != nil {
			t.Fatal(err)
		}
		if err := consentSchema.Validate(document); (err == nil) != admitted {
			t.Fatalf("consent off_path command %q admitted = %v, want %v (%v)", command, err == nil, admitted, err)
		}
	}
}
