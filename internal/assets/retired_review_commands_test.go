package assets

import (
	"io/fs"
	"path"
	"strings"
	"testing"
)

// retiredReviewAssetTokens are the strings no embedded asset may carry once the
// review by receipts (RDD) is retired. The only way to bring them back is to
// absorb upstream assets that still ship it, so this guard reads every embedded
// file, not just the directory an install writes to.
var retiredReviewAssetTokens = []string{
	"axiom review ",
	"gentle-ai review ",
	"review-ledger-contract",
	"rdd-defect-workflow",
	"rdd-advisory-transport",
	"axiom-bench",
}

// TestEmbeddedAssetsCarryNoRetiredReviewCommands pins REQ-46.3 and REQ-46.4 on
// the embedded tree: no asset names a retired review command, contract or
// skill, and no `review-*` agent or review transport plugin is embedded.
func TestEmbeddedAssetsCarryNoRetiredReviewCommands(t *testing.T) {
	inspected := 0
	err := fs.WalkDir(FS, ".", func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		inspected++
		base := path.Base(file)
		if strings.Contains(file, "/agents/") && strings.HasPrefix(base, "review-") {
			t.Errorf("%s: the review agent is still embedded", file)
		}
		if strings.Contains(base, "review-transport") {
			t.Errorf("%s: the review transport plugin is still embedded", file)
		}
		if strings.HasPrefix(file, "skills/rdd-") {
			t.Errorf("%s: an RDD skill is still embedded", file)
		}
		data, readErr := fs.ReadFile(FS, file)
		if readErr != nil {
			return readErr
		}
		for _, token := range retiredReviewAssetTokens {
			if strings.Contains(string(data), token) {
				t.Errorf("%s: carries the retired review token %q", file, token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded assets: %v", err)
	}
	if inspected < 100 {
		t.Fatalf("inspected only %d embedded files; the guard would pass vacuously", inspected)
	}
}
