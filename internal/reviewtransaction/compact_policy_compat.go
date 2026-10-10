package reviewtransaction

// FacadeReviewPolicy is the default review policy frozen into new compact
// lineages when the caller does not supply a policy file.
const FacadeReviewPolicy = `Axiom native bounded review policy.

Only candidate-caused BLOCKER or CRITICAL findings may require correction. Pre-existing and base-only findings are follow-ups. One correction is bounded by the frozen original scope. Gates are informational and unmanaged; ordinary repository policy decides delivery.
`

// LegacyFacadeReviewPolicy is the exact default policy text that binaries
// produced before the product name changed froze into their lineages. It is
// kept only so those lineages still replay and carry evidence forward; it is
// never written into a new lineage.
const LegacyFacadeReviewPolicy = `Gentle AI native bounded review policy.

Only candidate-caused BLOCKER or CRITICAL findings may require correction. Pre-existing and base-only findings are follow-ups. One correction is bounded by the frozen original scope. Gates are informational and unmanaged; ordinary repository policy decides delivery.
`

var (
	compactCurrentFacadePolicyHash = compactPolicyContentHash(FacadeReviewPolicy)
	compactLegacyFacadePolicyHash  = compactPolicyContentHash(LegacyFacadeReviewPolicy)
)

// compactPolicyHashesEquivalent reports whether a policy hash frozen in an
// existing lineage satisfies a policy hash requested now. Equality always
// holds; the single extra pair accepted is a lineage frozen with the legacy
// default policy against a request for the current default policy. The
// relation is exact and one-directional: no other hash and no reverse pair is
// equivalent. Only the START replay and the evidence carry-over compare a
// frozen hash with a freshly built one; every frozen-versus-frozen check stays
// strict.
func compactPolicyHashesEquivalent(frozen, requested string) bool {
	return frozen == requested ||
		(frozen == compactLegacyFacadePolicyHash && requested == compactCurrentFacadePolicyHash)
}

// compactAtomicStartReplayMismatch is compactAtomicStartMismatch for the
// replay of an already frozen START: a legacy default policy hash replays under
// the current default policy. The existing binding is never rewritten.
func compactAtomicStartReplayMismatch(existing, requested CompactAtomicStartBinding) string {
	if existing.PolicyHash != requested.PolicyHash && compactPolicyHashesEquivalent(existing.PolicyHash, requested.PolicyHash) {
		requested.PolicyHash = existing.PolicyHash
	}
	return compactAtomicStartMismatch(existing, requested)
}
