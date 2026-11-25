package capabilitymanifest

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

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
	// Digests pin the canonical JSON of every supported agent's manifest.
	wantManifestDigests := map[model.AgentID]string{
		model.AgentAntigravity:      "sha256:93147d813f87f038f0b486098c7d39fc01b6200b0b89e2aa55bddfbbecd36842",
		model.AgentClaudeCode:       "sha256:6386624d7ec694d7ed79928b193ef6bc611564ee5421b32941aa2bb1f496c5b2",
		model.AgentCodex:            "sha256:d082469ad7eb4fb6426951d4f9b5163558552b0e0448d7b2bfb94cd421ab9f54",
		model.AgentCursor:           "sha256:da44d33b397550c4483966fb4eb466beba8c65737338cb9b0fe634709de57c67",
		model.AgentGeminiCLI:        "sha256:619fa29a43d429b9d73aec4b131eeb8c33c325cd1f589c22dc583d7b0205564b",
		model.AgentGitHubCopilotCLI: "sha256:8cb612203fa1e999505dc915f7ae4cfb87afafa96a09543a25b55ee47b9c4497",
		model.AgentHermes:           "sha256:31e521642248d9a1946587d6a36346c09f827bdaeb08d56f0ffef3338e60caac",
		model.AgentKilocode:         "sha256:4bea71ac2fa786e868906b916fbdea1fa2d2c4407fbec8eb9f2adaa33734c855",
		model.AgentKimi:             "sha256:f5de46e14b67e155af07745190c098b9c4b497137fb7cc15165ae090e15dee05",
		model.AgentKiroIDE:          "sha256:249818d423d03731195e79627df348858a0772ff3e9d401d9e9876d4026c3a8f",
		model.AgentOpenClaw:         "sha256:b4f06b78e3e2e12e378ba0d45ecdceba2eff572adbec7c2ee680af292126c66b",
		model.AgentOpenCode:         "sha256:49f3f6f0755335e58fd2f9acca0cfac75f762b84a15b9af90ee3726f000b93b4",
		model.AgentPi:               "sha256:24ffc4968f6eab2800edefd09a99eb4249c5308e685ffdbb41df78504b6a348f",
		model.AgentQwenCode:         "sha256:76982f969755bcc23dccf909d0cd77ed7b84f40f23d437247167fe0c4f87b133",
		model.AgentTrae:             "sha256:0ee0722c20698dcf4edbf18c2c3c7ab0c4a128cf2513c16f750c1ee3b2080dc8",
		model.AgentVSCodeCopilot:    "sha256:11162d72d627970fdaac530c70ba4a0acfe49d35a4b60df589290e9885244029",
		model.AgentWindsurf:         "sha256:bb3083e660c6ac6c43996711c14c97444e98453e01d8319793ad1f214b1061f0",
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

func TestForAgentRejectsUnknownAgent(t *testing.T) {
	t.Parallel()

	_, err := ForAgent(model.AgentID("unknown"))
	if !errors.Is(err, ErrUnsupportedAgent) {
		t.Fatalf("ForAgent() error = %v, want ErrUnsupportedAgent", err)
	}
}
