package assets

import (
	"strings"
	"testing"
)

// The automated defect handoff was filing other projects' failures, and it only
// ever served the RDD consent envelope. With RDD suspended (C-01-c) it is
// retired entirely: no orchestrator carries its admissibility test, its excluded
// classes or its silent out-of-scope path, so the superseded subjective test can
// never come back either.
func TestDefectHandoffIsRetiredFromEveryOrchestrator(t *testing.T) {
	for _, path := range allSDDOrchestratorAssetPaths(t) {
		t.Run(path, func(t *testing.T) {
			content := mustReadResolved(path)
			for _, forbidden := range []string{
				"what produced the failure",
				"an Axiom invocation produced it",
				"hosting a failure is not enough",
				"there is no report and no handoff",
				"appears blocked by a Gentle AI provider or tool defect",
			} {
				if strings.Contains(content, forbidden) {
					t.Errorf("%s still carries the retired defect handoff: %q", path, forbidden)
				}
			}
		})
	}
}
