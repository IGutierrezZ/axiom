package sdd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestCommittedKiroSkillRegistryHookMatchesGenerator guards the one Kiro file
// this repository versions (.kiro/hooks/axiom-skill-registry.json). The rest of
// .kiro/ is ignored because Axiom regenerates it; the hook is committed, so it
// can drift silently from what ensureKiroSkillRegistryHook writes.
func TestCommittedKiroSkillRegistryHookMatchesGenerator(t *testing.T) {
	committedPath := filepath.Join(kiroHookDriftRepoRoot(t), ".kiro", "hooks", skillRegistryHookFileName)
	committed, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatalf("read committed Kiro hook %q: %v\nregenerate it with: axiom setup --agent kiro-ide (from the repository root) and commit .kiro/hooks/%s", committedPath, err, skillRegistryHookFileName)
	}
	// A Windows checkout can convert LF to CRLF despite .gitattributes.
	committed = bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n"))

	generatedPath := filepath.Join(t.TempDir(), ".kiro", "hooks", skillRegistryHookFileName)
	if _, err := ensureKiroSkillRegistryHook(generatedPath); err != nil {
		t.Fatalf("ensureKiroSkillRegistryHook() error = %v", err)
	}
	generated, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read generated Kiro hook %q: %v", generatedPath, err)
	}

	if !bytes.Equal(committed, generated) {
		t.Fatalf("committed .kiro/hooks/%s drifted from ensureKiroSkillRegistryHook output.\n"+
			"Regenerate it with: axiom setup --agent kiro-ide (from the repository root), then commit the result.\n"+
			"committed:\n%s\ngenerated:\n%s", skillRegistryHookFileName, committed, generated)
	}
}

// kiroHookDriftRepoRoot walks upward from the test working directory until it
// finds go.mod, so the check runs from any package directory.
func kiroHookDriftRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
}
