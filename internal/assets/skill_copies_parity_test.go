package assets

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The delivery skills ship twice: the embedded asset that `axiom sync`
// distributes and the public copy under `skills/` that the repository itself
// loads. Without a guard the two drift apart, as work-unit-commits did before
// single-pr (two lines existed only in the embedded copy).
func TestDeliverySkillCopiesAreByteIdentical(t *testing.T) {
	for _, skill := range []string{"work-unit-commits", "comment-writer"} {
		t.Run(skill, func(t *testing.T) {
			embedded, err := Read("skills/" + skill + "/SKILL.md")
			if err != nil {
				t.Fatalf("Read(embedded %s) error = %v", skill, err)
			}
			source, err := os.ReadFile(filepath.Join("..", "..", "skills", skill, "SKILL.md"))
			if err != nil {
				t.Fatalf("ReadFile(source %s) error = %v", skill, err)
			}
			if !bytes.Equal(source, []byte(embedded)) {
				t.Fatalf("skills/%s/SKILL.md and the embedded asset differ; the embedded asset is canonical, copy it byte for byte", skill)
			}
		})
	}
}
