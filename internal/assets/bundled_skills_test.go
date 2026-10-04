package assets

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicBundledSkillsMatchEmbeddedAssets(t *testing.T) {
	for _, skill := range []string{"systemic-issue-triage"} {
		t.Run(skill, func(t *testing.T) {
			sourcePath := filepath.Join("..", "..", "skills", skill, "SKILL.md")
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				t.Skipf("public source skill %q does not exist on disk", skill)
				return
			}
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("ReadFile(public source) error = %v", err)
			}

			embedded, err := Read("skills/" + skill + "/SKILL.md")
			if err != nil {
				t.Fatalf("Read(embedded asset) error = %v", err)
			}
			if !bytes.Equal(source, []byte(embedded)) {
				t.Fatal("public source skill and embedded distribution asset differ")
			}
		})
	}
}
