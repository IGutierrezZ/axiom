package capabilitymanifest

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestCanonicalImplementationRoutingBoundaries(t *testing.T) {
	t.Parallel()

	got := CanonicalImplementationRouting()
	want := ImplementationRoutingFacts{
		DirectInline: DirectInlineFacts{
			MaxEvidenceBatches:                       1,
			MaxEvidenceCalls:                         3,
			ApproxEvidenceTokens:                     10000,
			EvidenceMustUseBoundedRanges:             true,
			MaxInlineUnitLines:                       100,
			MechanicalWriteMustBeAlreadyUnderstood:   true,
			MechanicalWriteMustNotRequireResearch:    true,
			MechanicalWriteMustNotHaveOpenDesignWork: true,
		},
		DelegatedDirect: DelegatedDirectFacts{
			ApproxHandoffTokens:           2000,
			MaxParentSpotChecks:           1,
			ApproxParentContextTokens:     150000,
			ContextBackstopIsAdvisory:     true,
			WriterRequiresNamedReason:     true,
			DelegateWhenReadPreparesWrite: true,
			DelegateWhenBroadResearch:     true,
		},
		SDD: SDDProposalFacts{
			ProposeWhenSubstantialOrAmbiguous:     true,
			DurableArtifactsMustReduceUncertainty: true,
			SelectionPolicy:                       SDDSelectionExplicitRequestProjectLaneOrAcceptedProposal,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CanonicalImplementationRouting() = %#v, want %#v", got, want)
	}
}

func TestManifestRejectsWeakenedRoutingFacts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		weaken func(*AgentCapabilityManifest)
	}{
		{
			name: "inline evidence budget grows past three calls",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DirectInline.MaxEvidenceCalls = 4
			},
		},
		{
			name: "inline evidence budget grows past ten thousand tokens",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DirectInline.ApproxEvidenceTokens = 20000
			},
		},
		{
			name: "inline evidence may use unbounded reads",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DirectInline.EvidenceMustUseBoundedRanges = false
			},
		},
		{
			name: "inline units are no longer capped at one hundred lines",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DirectInline.MaxInlineUnitLines = 50
			},
		},
		{
			name: "explorer handoff grows past two thousand tokens",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.ApproxHandoffTokens = 5000
			},
		},
		{
			name: "parent spot check is unbounded",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.MaxParentSpotChecks = 3
			},
		},
		{
			name: "context backstop becomes enforced",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.ContextBackstopIsAdvisory = false
			},
		},
		{
			name: "writer no longer needs a named reason",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.WriterRequiresNamedReason = false
			},
		},
		{
			name: "read preparing write no longer delegates",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.DelegateWhenReadPreparesWrite = false
			},
		},
		{
			name: "broad research no longer delegates",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.DelegateWhenBroadResearch = false
			},
		},
		{
			name: "substantial ambiguity no longer proposes SDD",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.SDD.ProposeWhenSubstantialOrAmbiguous = false
			},
		},
		{
			name: "SDD proposal need not reduce durable uncertainty",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.SDD.DurableArtifactsMustReduceUncertainty = false
			},
		},
		{
			name: "SDD selection bypasses explicit consent",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.SDD.SelectionPolicy = "automatic"
			},
		},
		{
			name: "retired selection literal is no longer canonical (REQ-27.16)",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.SDD.SelectionPolicy = "explicit_request_or_accepted_proposal"
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			manifest := MustForAgent(model.AgentClaudeCode)
			test.weaken(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate() = nil, want non-canonical routing rejection")
			}
		})
	}
}

func TestEveryManifestKeepsWorkRoutingDormantAndHashesCanonically(t *testing.T) {
	t.Parallel()

	const wantRoutingDigest = "sha256:24fb8cbd6f38e7c4a6917e3abdc523f4ae51eb4fea5da76705b383ade8f13ed3"
	// Digests pin the four providers with an enforceable fresh-reviewer
	// boundary: Claude Code's generated reviewer has no live tools, OpenCode
	// relays one ordinary task through Go-owned admission, Codex's provider
	// subprocess reaches the same contract, and gentle-pi's host relay
	// forwards the Go-issued opaque task to a fresh locked-down pi
	// subprocess (gentle-pi#311, gentle-ai#3249).
	wantManifestDigests := map[model.AgentID]string{
		model.AgentAntigravity:      "sha256:03a35fd1c8b95f1ef5575e492ecdf3ae0719c165586663aa86a0d68c42797a7b",
		model.AgentClaudeCode:       "sha256:4ff85fc5eb5278ef4a7552650270c012448e822c33bab8d442bc9bd849ab328e",
		model.AgentCodex:            "sha256:72718322f2be7cab668922d44403cf622fa0ff550444cbd7c7879b2cbcc05ec5",
		model.AgentCursor:           "sha256:14d2288f56d6efd8493b6a2fe2f907817a2363cee6604b78c0673f48815fd9ee",
		model.AgentGeminiCLI:        "sha256:da66d357b74598aaf95e98e15af3ce709f41dc30b2a39a340c7f25d8f6158d2a",
		model.AgentGitHubCopilotCLI: "sha256:56c2bb26d29547ac910483a1676d0ed92557dd1e01aa27afe2150c507f3aeb42",
		model.AgentHermes:           "sha256:e39d7b505f57db5a968087770cd5d2b7c08d3a95dc4c8500ae3a207bd4490d58",
		model.AgentKilocode:         "sha256:7e8858b732e0a0e637bd21f80d9757092f426bc903a69e856437e5dae545597c",
		model.AgentKimi:             "sha256:3ad829e58930d03f0bdb2da3ba1d089912c3b7c0fd0dff5cf2d029515294a280",
		model.AgentKiroIDE:          "sha256:f47244d8a8a480fc6c279a0f86acee1f2e3580493d42a08f0573613400921d41",
		model.AgentOpenClaw:         "sha256:52a24440ec541fa3b5c35b0ad3593c2cd21c792881f52504855ee2830b50ae6c",
		model.AgentOpenCode:         "sha256:303e1d4cd22cf891093b2b4d5d119a12190e7070f88fadeef240c666307a96cc",
		model.AgentPi:               "sha256:049740b83550b87958131a157fd1303e3f32743028e1ced2f5d924990d4dad80",
		model.AgentQwenCode:         "sha256:d5b3c45d7939810c3dd94c1838505825aab0f395e3a1b009a46030ca3b04fc5b",
		model.AgentTrae:             "sha256:d91de263fb1dcb3f013d811dbad2973e33450b1d193a262ee5937fd68c7419af",
		model.AgentVSCodeCopilot:    "sha256:da345ae4c631fdd99d92b77b21415ced213cd8c1fef2511de09c228ef4f01c7f",
		model.AgentWindsurf:         "sha256:4b57475c602cfeafd91308c66b9360ae135b0459e2544d28c4ab2c13d06aa173",
	}

	for agent, wantDigest := range wantManifestDigests {
		agent := agent
		wantDigest := wantDigest
		t.Run(string(agent), func(t *testing.T) {
			t.Parallel()

			manifest := MustForAgent(agent)
			if err := manifest.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if manifest.Contracts.WorkRoutingV1.Exposure != ContractExposureDormant {
				t.Fatalf("work-routing exposure = %q, want %q", manifest.Contracts.WorkRoutingV1.Exposure, ContractExposureDormant)
			}
			if manifest.Advertises(ContractWorkRoutingV1) {
				t.Fatal("work-routing must remain unadvertised before final activation")
			}
			wantImmutableExecutor := agent == model.AgentClaudeCode || agent == model.AgentOpenCode || agent == model.AgentCodex || agent == model.AgentPi
			if got := manifest.Advertises(ContractImmutableReviewExecutorV1); got != wantImmutableExecutor {
				t.Fatalf("immutable reviewer execution advertised = %t, want %t", got, wantImmutableExecutor)
			}
			wantExposure := ContractExposureDormant
			if wantImmutableExecutor {
				wantExposure = ContractExposureAdvertised
			}
			if got := manifest.Contracts.ImmutableReviewExecutorV1.Exposure; got != wantExposure {
				t.Fatalf("immutable reviewer execution exposure = %q, want %q", got, wantExposure)
			}

			payload, err := manifest.CanonicalJSON()
			if err != nil {
				t.Fatalf("CanonicalJSON() error = %v", err)
			}
			var roundTrip AgentCapabilityManifest
			if err := json.Unmarshal(payload, &roundTrip); err != nil {
				t.Fatalf("Unmarshal(CanonicalJSON()) error = %v", err)
			}
			if roundTrip != manifest {
				t.Fatalf("canonical JSON round trip = %#v, want %#v", roundTrip, manifest)
			}

			gotDigest, err := roundTrip.Digest()
			if err != nil {
				t.Fatalf("Digest() error = %v", err)
			}
			if gotDigest != wantDigest {
				t.Fatalf("Digest() = %q, want %q", gotDigest, wantDigest)
			}

			gotRoutingDigest, err := manifest.RoutingDigest()
			if err != nil {
				t.Fatalf("RoutingDigest() error = %v", err)
			}
			if gotRoutingDigest != wantRoutingDigest {
				t.Fatalf("RoutingDigest() = %q, want %q", gotRoutingDigest, wantRoutingDigest)
			}
		})
	}
}

// TestEveryManifestDigestStaysByteStable pins every non-Pi row at the closed
// review-transport baseline. The 12 non-RDD rows change only because their
// transport claim becomes dormant; the three non-Pi RDD rows remain unchanged.
func TestEveryManifestDigestStaysByteStable(t *testing.T) {
	t.Parallel()

	wantNonPiDigests := map[model.AgentID]string{
		model.AgentAntigravity:      "sha256:03a35fd1c8b95f1ef5575e492ecdf3ae0719c165586663aa86a0d68c42797a7b",
		model.AgentClaudeCode:       "sha256:4ff85fc5eb5278ef4a7552650270c012448e822c33bab8d442bc9bd849ab328e",
		model.AgentCodex:            "sha256:72718322f2be7cab668922d44403cf622fa0ff550444cbd7c7879b2cbcc05ec5",
		model.AgentCursor:           "sha256:14d2288f56d6efd8493b6a2fe2f907817a2363cee6604b78c0673f48815fd9ee",
		model.AgentGeminiCLI:        "sha256:da66d357b74598aaf95e98e15af3ce709f41dc30b2a39a340c7f25d8f6158d2a",
		model.AgentGitHubCopilotCLI: "sha256:56c2bb26d29547ac910483a1676d0ed92557dd1e01aa27afe2150c507f3aeb42",
		model.AgentHermes:           "sha256:e39d7b505f57db5a968087770cd5d2b7c08d3a95dc4c8500ae3a207bd4490d58",
		model.AgentKilocode:         "sha256:7e8858b732e0a0e637bd21f80d9757092f426bc903a69e856437e5dae545597c",
		model.AgentKimi:             "sha256:3ad829e58930d03f0bdb2da3ba1d089912c3b7c0fd0dff5cf2d029515294a280",
		model.AgentKiroIDE:          "sha256:f47244d8a8a480fc6c279a0f86acee1f2e3580493d42a08f0573613400921d41",
		model.AgentOpenClaw:         "sha256:52a24440ec541fa3b5c35b0ad3593c2cd21c792881f52504855ee2830b50ae6c",
		model.AgentOpenCode:         "sha256:303e1d4cd22cf891093b2b4d5d119a12190e7070f88fadeef240c666307a96cc",
		model.AgentQwenCode:         "sha256:d5b3c45d7939810c3dd94c1838505825aab0f395e3a1b009a46030ca3b04fc5b",
		model.AgentTrae:             "sha256:d91de263fb1dcb3f013d811dbad2973e33450b1d193a262ee5937fd68c7419af",
		model.AgentVSCodeCopilot:    "sha256:da345ae4c631fdd99d92b77b21415ced213cd8c1fef2511de09c228ef4f01c7f",
		model.AgentWindsurf:         "sha256:4b57475c602cfeafd91308c66b9360ae135b0459e2544d28c4ab2c13d06aa173",
	}

	nonPiAgents := make([]model.AgentID, 0, len(wantNonPiDigests))
	for agent := range wantNonPiDigests {
		nonPiAgents = append(nonPiAgents, agent)
	}

	if got := len(nonPiAgents); got != 16 {
		t.Fatalf("want 16 non-Pi agents, got %d", got)
	}

	for _, agent := range nonPiAgents {
		agent := agent
		wantDigest := wantNonPiDigests[agent]
		t.Run(string(agent), func(t *testing.T) {
			t.Parallel()

			manifest := MustForAgent(agent)
			gotDigest, err := manifest.Digest()
			if err != nil {
				t.Fatalf("Digest() error = %v", err)
			}
			if gotDigest != wantDigest {
				t.Fatalf("Digest() = %q, want %q (byte-stable contract)", gotDigest, wantDigest)
			}
		})
	}
}

func TestReviewTransportAdvertisementIsClosedCatalogSet(t *testing.T) {
	const wantExposed = 4

	exposed := 0
	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			manifest := MustForAgent(agent.ID)
			want := agent.ID == model.AgentClaudeCode ||
				agent.ID == model.AgentOpenCode ||
				agent.ID == model.AgentCodex ||
				agent.ID == model.AgentPi
			if got := manifest.Advertises(ContractReviewTransportV1); got != want {
				t.Fatalf("review transport advertised = %t, want %t", got, want)
			}
			if want {
				exposed++
			}
		})
	}
	if exposed != wantExposed {
		t.Fatalf("advertised review transport runtimes = %d, want %d", exposed, wantExposed)
	}
}

func TestForAgentRejectsUnknownAgent(t *testing.T) {
	t.Parallel()

	_, err := ForAgent(model.AgentID("unknown"))
	if !errors.Is(err, ErrUnsupportedAgent) {
		t.Fatalf("ForAgent() error = %v, want ErrUnsupportedAgent", err)
	}
}
