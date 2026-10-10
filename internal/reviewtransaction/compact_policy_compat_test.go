package reviewtransaction

import (
	"strings"
	"testing"
)

func TestCompactFacadePolicyHashesArePinnedAndNeverDrift(t *testing.T) {
	const legacyPinned = "sha256:36e477a871ace064f20f06aa9c669315dad46b9732a0a3092cf740f78ced3434"
	const currentPinned = "sha256:25e6bf0ead2a5951a143f615dda54cd5108d923d32d8f40aef7294995c79e076"
	if got := compactPolicyContentHash(LegacyFacadeReviewPolicy); got != legacyPinned || compactLegacyFacadePolicyHash != legacyPinned {
		t.Fatalf("legacy policy hash = %q (const %q), want pinned %q", got, compactLegacyFacadePolicyHash, legacyPinned)
	}
	if got := compactPolicyContentHash(FacadeReviewPolicy); got != currentPinned || compactCurrentFacadePolicyHash != got {
		t.Fatalf("current policy hash = %q (const %q), want %q", got, compactCurrentFacadePolicyHash, currentPinned)
	}
	if !strings.Contains(LegacyFacadeReviewPolicy, "Gentle AI") {
		t.Fatal("legacy policy text must keep the exact historical product name")
	}
	if !strings.Contains(FacadeReviewPolicy, "Axiom") || strings.Contains(FacadeReviewPolicy, "Gentle AI") {
		t.Fatal("current policy text must name Axiom and never Gentle AI")
	}
	if strings.Replace(LegacyFacadeReviewPolicy, "Gentle AI", "Axiom", 1) != FacadeReviewPolicy {
		t.Fatal("current policy must differ from the legacy text only by the product name")
	}
}

func TestCompactPolicyHashesEquivalentIsExactAndOneDirectional(t *testing.T) {
	other := compactAtomicStartHash("f")
	tests := []struct {
		name              string
		frozen, requested string
		want              bool
	}{
		{"identical custom hash", other, other, true},
		{"identical legacy hash", compactLegacyFacadePolicyHash, compactLegacyFacadePolicyHash, true},
		{"identical current hash", compactCurrentFacadePolicyHash, compactCurrentFacadePolicyHash, true},
		{"legacy frozen accepts current request", compactLegacyFacadePolicyHash, compactCurrentFacadePolicyHash, true},
		{"current frozen refuses legacy request", compactCurrentFacadePolicyHash, compactLegacyFacadePolicyHash, false},
		{"legacy frozen refuses unrelated request", compactLegacyFacadePolicyHash, other, false},
		{"unrelated frozen refuses current request", other, compactCurrentFacadePolicyHash, false},
		{"unrelated frozen refuses legacy request", other, compactLegacyFacadePolicyHash, false},
		{"empty frozen refuses current request", "", compactCurrentFacadePolicyHash, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compactPolicyHashesEquivalent(tt.frozen, tt.requested); got != tt.want {
				t.Fatalf("compactPolicyHashesEquivalent(%q, %q) = %t, want %t", tt.frozen, tt.requested, got, tt.want)
			}
		})
	}
}

func TestCompactAtomicStartReplayMismatchOnlyRelaxesThePolicyHashPair(t *testing.T) {
	base := CompactAtomicStartBinding{
		LineageID: "replay-mismatch", WorktreeIdentity: "worktree", TargetIdentity: compactAtomicStartHash("a"),
		PolicyHash: compactLegacyFacadePolicyHash, Tier: RiskLow, SelectedLenses: []string{LensRisk},
	}
	requested := base
	requested.PolicyHash = compactCurrentFacadePolicyHash
	if field := compactAtomicStartReplayMismatch(base, requested); field != "" {
		t.Fatalf("legacy to current replay mismatch = %q, want none", field)
	}
	if requested.PolicyHash != compactCurrentFacadePolicyHash || base.PolicyHash != compactLegacyFacadePolicyHash {
		t.Fatal("replay comparison rewrote a binding")
	}
	if field := compactAtomicStartMismatch(base, requested); field != "policy_hash" {
		t.Fatalf("strict comparator = %q, want policy_hash", field)
	}
	requested.Tier = RiskHigh
	if field := compactAtomicStartReplayMismatch(base, requested); field != "tier" {
		t.Fatalf("equivalent policy must not mask another field, got %q", field)
	}
	reverse, current := base, base
	reverse.PolicyHash, current.PolicyHash = compactCurrentFacadePolicyHash, compactLegacyFacadePolicyHash
	if field := compactAtomicStartReplayMismatch(reverse, current); field != "policy_hash" {
		t.Fatalf("reverse pair = %q, want policy_hash", field)
	}
}
