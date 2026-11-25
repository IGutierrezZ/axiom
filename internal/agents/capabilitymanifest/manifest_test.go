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

	const wantRoutingDigest = "sha256:3106087a9bc8fefe5369ff8d1ad63f58ad3c47785710640408c2c87a62c2944d"
	// Digests pin the canonical JSON of every supported agent's manifest.
	wantManifestDigests := map[model.AgentID]string{
		model.AgentAntigravity:      "sha256:eec50d3f26afd75cb742954c2faa608ba2a6a9b20e6f1e3d8bc71761871b25c6",
		model.AgentClaudeCode:       "sha256:317eaa1be5e6ae9192aa72962f3313a115fe4504cb267a76eff47854b563461f",
		model.AgentCodex:            "sha256:856e4c93372f56b571eb539f6af6cbd9c0f6b17c82570857fddf91e55278da01",
		model.AgentCursor:           "sha256:13589ef70b28e58333bf1986ce3cb11107fa05b17533aac19cc26da30841c720",
		model.AgentGeminiCLI:        "sha256:32e03065615b5ef3b790310637c226833688c30130c9632ebba86145fd35f644",
		model.AgentGitHubCopilotCLI: "sha256:43d174deebc551fc4436f919de41a093884b6eef15e2d21a40afb09bcc44ea26",
		model.AgentHermes:           "sha256:a994cb71d6afd71244a5a6c26aa1831863209f780375cb95306efd1bcf31b933",
		model.AgentKilocode:         "sha256:d9d92d526c407ebde1bf4de4ea20197d3941af37baad776ae612f51d066f5552",
		model.AgentKimi:             "sha256:7a468b6da268f28d7d925b99801a61e0c0d31052031780d883e76e7fc7529b23",
		model.AgentKiroIDE:          "sha256:ea10acb87385dd682950639adfda39be9689fa65f4ea62431f51ea5134a19013",
		model.AgentOpenClaw:         "sha256:455fbe0fb711a7326eb5a6c04a2862c4b0b6c8fb0c4d4e45d0a34f0237fe1607",
		model.AgentOpenCode:         "sha256:81864464d366d6324745a56c8909ca9241b9822b551a51a811d5136723b2b904",
		model.AgentPi:               "sha256:17121ab9c73594bfa9c518e617b6314ed80fe81da564c16f1ed709c8292ca6ce",
		model.AgentQwenCode:         "sha256:a48c55be99e8d4371d0883811edb2e0e2319049bc728a598629f3d3d3bdb65c1",
		model.AgentTrae:             "sha256:eaecdec856d7c7aeadb9832857692162508212d6f94efead51a9ff92f7d6cb97",
		model.AgentVSCodeCopilot:    "sha256:39b13327a98a30dbb1076910b84c27068386ef95622282fd21f896f16e266a1c",
		model.AgentWindsurf:         "sha256:47cf8c37a07d1e32543f689334e6c6d8d2ead2f975dc108724b87b5220a19aa4",
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
