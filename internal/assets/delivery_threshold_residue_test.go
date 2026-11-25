package assets

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Delivery is `single-pr` by default and never asked, so no shipped asset and
// no public skill may state a numeric delivery threshold again. The textual
// residue is guarded here, in process and without external tools, so it is part
// of `go test ./...`. The text generated from Go is guarded in its own package:
// `routing_test.go` (agentguidance), `session_preflight_test.go` (sdd) and
// `sdd_preflight_hook_test.go` (cli).

// retiredDeliveryLiterals are the phrases of the retired review-workload guard.
var retiredDeliveryLiterals = []string{
	"400-line budget risk",
	"Decision needed before apply",
	"Chained PRs recommended",
	"workload-decision-required",
	"Review Workload Guard",
	"review_budget_lines",
}

var deliveryThresholdNumber = regexp.MustCompile(`\b400\b`)

// admittedDeliveryResidue reports whether a line that carries the retired number
// is an admitted residue. It is decided by path AND content, so a new `400` can
// never slip into a file that is admitted for another reason.
func admittedDeliveryResidue(file, line string) bool {
	switch {
	case strings.Contains(line, "Full-4R review") && isFull4RSweepSource(file):
		// Sweep threshold of the judges.
		return true
	}
	return false
}

func isFull4RSweepSource(file string) bool {
	dir, name := path.Split(file)
	if strings.HasSuffix(dir, "/agents/") {
		return strings.HasPrefix(name, "jd-judge-")
	}
	return file == "opencode/sdd-overlay-single.json" || file == "opencode/sdd-overlay-multi.json"
}

func scanDeliveryResidue(t *testing.T, file string, content []byte) {
	t.Helper()
	if bytes.IndexByte(content, 0) >= 0 {
		return // binary asset
	}
	for number, line := range strings.Split(string(content), "\n") {
		found := ""
		if deliveryThresholdNumber.MatchString(line) {
			found = "the number 400"
		}
		for _, literal := range retiredDeliveryLiterals {
			if strings.Contains(line, literal) {
				found = "the retired literal " + literal
			}
		}
		if found == "" || admittedDeliveryResidue(file, line) {
			continue
		}
		t.Errorf("%s:%d carries %s: %s", file, number+1, found, strings.TrimSpace(line))
	}
}

func TestDeliveryThresholdResidue(t *testing.T) {
	t.Run("embedded assets", func(t *testing.T) {
		scanned := 0
		err := fs.WalkDir(FS, ".", func(file string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			content, err := fs.ReadFile(FS, file)
			if err != nil {
				return err
			}
			scanned++
			scanDeliveryResidue(t, file, content)
			return nil
		})
		if err != nil {
			t.Fatalf("walk embedded assets: %v", err)
		}
		if scanned == 0 {
			t.Fatal("no embedded asset was scanned; the guard has lost its subject")
		}
	})

	t.Run("public skills", func(t *testing.T) {
		root := filepath.Join("..", "..", "skills")
		scanned := 0
		err := filepath.WalkDir(root, func(file string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			content, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(filepath.Join("..", ".."), file)
			if err != nil {
				return err
			}
			scanned++
			scanDeliveryResidue(t, filepath.ToSlash(relative), content)
			return nil
		})
		if err != nil {
			t.Fatalf("walk public skills: %v", err)
		}
		if scanned == 0 {
			t.Fatal("no public skill was scanned; the guard has lost its subject")
		}
	})
}

// The admission rules must stay narrow: a number outside the admitted files, or
// inside them without the admitted content, is a failure.
func TestDeliveryThresholdResidueAdmissionIsNarrow(t *testing.T) {
	for _, tc := range []struct {
		name, file, line string
		want             bool
	}{
		{"judge sweep line", "claude/agents/jd-judge-a.md", "Full-4R review (or >400 changed lines)", true},
		{"judge line without the sweep text", "claude/agents/jd-judge-a.md", "split above 400 lines", false},
		{"overlay sweep line", "opencode/sdd-overlay-multi.json", "Full-4R review >400", true},
		{"sweep text in an unrelated asset", "claude/sdd-orchestrator-workflow.md", "Full-4R review >400", false},
		{"another skill", "skills/chained-pr/SKILL.md", "PRs over 400 lines", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := admittedDeliveryResidue(tc.file, tc.line); got != tc.want {
				t.Fatalf("admittedDeliveryResidue(%q, %q) = %v, want %v", tc.file, tc.line, got, tc.want)
			}
		})
	}
}
