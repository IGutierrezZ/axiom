package assets

import (
	"strings"
	"testing"
)

// TestJudgmentDaySkillStartsWithoutReviewTransaction guards the separation
// from issue #2487: Judgment Day is a developer tool, so its skill body must
// never route judges through the delivery-receipt machinery. The negotiated
// review lifecycle stays the only producer of delivery authority, and a
// judgment must disclaim that authority instead of impersonating it.
func TestJudgmentDaySkillStartsWithoutReviewTransaction(t *testing.T) {
	body := MustRead("skills/judgment-day/SKILL.md")
	lower := strings.ToLower(body)

	for _, phrase := range []string{
		// The fused step that held the tool hostage to the authority.
		"review/start",
		"mode=judgment_day",
		"persist the transaction",
		"terminal transaction states",
		"terminal receipt",
	} {
		if strings.Contains(lower, phrase) {
			t.Errorf("skills/judgment-day/SKILL.md routes the tool into delivery-receipt machinery via %q", phrase)
		}
	}

	if !strings.Contains(lower, "carries no delivery authority") {
		t.Error("skills/judgment-day/SKILL.md must state that a judgment carries no delivery authority")
	}
}
