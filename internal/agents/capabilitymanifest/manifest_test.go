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
			MinUnderstandingFiles:                    1,
			MaxUnderstandingFiles:                    3,
			MaxMechanicalWriteFiles:                  1,
			MechanicalWriteMustBeAlreadyUnderstood:   true,
			MechanicalWriteMustNotRequireResearch:    true,
			MechanicalWriteMustNotHaveOpenDesignWork: true,
		},
		DelegatedDirect: DelegatedDirectFacts{
			MappingMinUnderstandingFiles:  4,
			WriterMinNonTrivialFiles:      2,
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
			name: "direct understanding starts below one file",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DirectInline.MinUnderstandingFiles = 0
			},
		},
		{
			name: "direct understanding exceeds three files",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DirectInline.MaxUnderstandingFiles = 4
			},
		},
		{
			name: "mapping starts after four files",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.MappingMinUnderstandingFiles = 5
			},
		},
		{
			name: "writer starts after two non-trivial files",
			weaken: func(manifest *AgentCapabilityManifest) {
				manifest.ImplementationRouting.DelegatedDirect.WriterMinNonTrivialFiles = 3
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

	const wantRoutingDigest = "sha256:e114fbf7265d24cd5db30409b99d5bda6025d91871246762d612e097c0ea0754"
	// Digests pin the four providers with an enforceable fresh-reviewer
	// boundary: Claude Code's generated reviewer has no live tools, OpenCode
	// relays one ordinary task through Go-owned admission, Codex's provider
	// subprocess reaches the same contract, and gentle-pi's host relay
	// forwards the Go-issued opaque task to a fresh locked-down pi
	// subprocess (gentle-pi#311, gentle-ai#3249).
	wantManifestDigests := map[model.AgentID]string{
		model.AgentAntigravity:   "sha256:b965a31aec2a39433a987f93dd9fb9ae84ebc756bbb1277dc43dcfaa5fa0cd09",
		model.AgentClaudeCode:    "sha256:3784060e708364dc6263dc01cd2bbdd92ca0bbe55382a12a55155df1ec2fa239",
		model.AgentCodex:         "sha256:52ac15f7d2f077e85d07e70965d000072441254afe5c7e3c756dfeacb2b38e55",
		model.AgentCursor:        "sha256:fe3dafc89ca71df79166b483e2726062e9b12f0585462401f88ae4da2b1409b8",
		model.AgentGeminiCLI:     "sha256:2fe3db0e9d8726c16b7aea8101c7a8a41377c8c41aef2acb89095a125d90da9d",
		model.AgentHermes:        "sha256:092b369652cdaf22233910bc4c11afebab7ec889f6526d9d44e2b0920f70c0f2",
		model.AgentKilocode:      "sha256:31a32cd9c79a8cc8a6594e735e3c2cab7a39ef4a1bf5e344d97a140b3e9af0b7",
		model.AgentKimi:          "sha256:1f06502934b765f22b97e52ce2150f0877f62e0eb598b3350cfe2886554681e6",
		model.AgentKiroIDE:       "sha256:ce6d1b33d122c0002bc9fd0d63405d749c514f15daec8ba59af392b74836c498",
		model.AgentOpenClaw:      "sha256:c62c5e827f25bed8f96570c9efeb5aaffafd1f939a8c4c0c9e6cddf7479a1384",
		model.AgentOpenCode:      "sha256:2ff5e0860b2657daefabae78de531b0e4bc620a9e64191fddf45a371b3b7fb9b",
		model.AgentPi:            "sha256:6022f66d1415fde731951e5ab024f60cdca213a8cbaed97e65652a8a18efa4f1",
		model.AgentQwenCode:      "sha256:9c79d3e8c331d2211586e57b62c28ac6e3880c330be29891cdacacb8a1ba66d5",
		model.AgentTrae:          "sha256:8c4dffe4d6d5f4e80e4ebddd95456dfaaa6e3723a2553a94762aeeaf364eb57b",
		model.AgentVSCodeCopilot: "sha256:06f7cc118863d1c28939e0715c8989cb2854b078ea3e9d13da6ac44f1994acbc",
		model.AgentWindsurf:      "sha256:86b8246b107505971caa296ffb4f24eb634d14ff4606e188003398bb20caa8c7",
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
		model.AgentAntigravity:   "sha256:b965a31aec2a39433a987f93dd9fb9ae84ebc756bbb1277dc43dcfaa5fa0cd09",
		model.AgentClaudeCode:    "sha256:3784060e708364dc6263dc01cd2bbdd92ca0bbe55382a12a55155df1ec2fa239",
		model.AgentCodex:         "sha256:52ac15f7d2f077e85d07e70965d000072441254afe5c7e3c756dfeacb2b38e55",
		model.AgentCursor:        "sha256:fe3dafc89ca71df79166b483e2726062e9b12f0585462401f88ae4da2b1409b8",
		model.AgentGeminiCLI:     "sha256:2fe3db0e9d8726c16b7aea8101c7a8a41377c8c41aef2acb89095a125d90da9d",
		model.AgentHermes:        "sha256:092b369652cdaf22233910bc4c11afebab7ec889f6526d9d44e2b0920f70c0f2",
		model.AgentKilocode:      "sha256:31a32cd9c79a8cc8a6594e735e3c2cab7a39ef4a1bf5e344d97a140b3e9af0b7",
		model.AgentKimi:          "sha256:1f06502934b765f22b97e52ce2150f0877f62e0eb598b3350cfe2886554681e6",
		model.AgentKiroIDE:       "sha256:ce6d1b33d122c0002bc9fd0d63405d749c514f15daec8ba59af392b74836c498",
		model.AgentOpenClaw:      "sha256:c62c5e827f25bed8f96570c9efeb5aaffafd1f939a8c4c0c9e6cddf7479a1384",
		model.AgentOpenCode:      "sha256:2ff5e0860b2657daefabae78de531b0e4bc620a9e64191fddf45a371b3b7fb9b",
		model.AgentQwenCode:      "sha256:9c79d3e8c331d2211586e57b62c28ac6e3880c330be29891cdacacb8a1ba66d5",
		model.AgentTrae:          "sha256:8c4dffe4d6d5f4e80e4ebddd95456dfaaa6e3723a2553a94762aeeaf364eb57b",
		model.AgentVSCodeCopilot: "sha256:06f7cc118863d1c28939e0715c8989cb2854b078ea3e9d13da6ac44f1994acbc",
		model.AgentWindsurf:      "sha256:86b8246b107505971caa296ffb4f24eb634d14ff4606e188003398bb20caa8c7",
	}

	nonPiAgents := make([]model.AgentID, 0, len(wantNonPiDigests))
	for agent := range wantNonPiDigests {
		nonPiAgents = append(nonPiAgents, agent)
	}

	if got := len(nonPiAgents); got != 15 {
		t.Fatalf("want 15 non-Pi agents, got %d", got)
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
