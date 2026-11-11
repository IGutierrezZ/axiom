package assets

import (
	"regexp"
	"strings"
	"testing"
)

// Pin de la frontera de la revisión independiente de SDD (REQ-28.11, DD-10).
//
// Solo inspecciona los pasajes que el cambio sdd-revision-independiente añade
// o modifica, nunca ficheros enteros: los textos preexistentes nombran el otro
// mecanismo de revisión a propósito para declarar la frontera y los fijan otras
// pruebas. Un pasaje es o un cuerpo delimitado nuevo o una constante de frase
// cuya presencia se comprueba en su fichero.

// reviewBoundaryVocabulary es el vocabulario que ningún pasaje nuevo puede usar.
var reviewBoundaryVocabulary = regexp.MustCompile(`(?i)\b(lineage|candidates?|acknowledg\w*|burn(s|ed)?|receipts?|rdd)\b`)

// reviewBoundaryLiterals son los literales de comandos y contratos de la otra
// frontera.
var reviewBoundaryLiterals = []string{"axiom review", "sdd-attempt", "review-integration"}

// reviewScopeMarkers son las marcas de selección de repositorio git que los
// pasajes que leen diffs deben nombrar (matriz de amenazas de DD-10).
var reviewScopeMarkers = []string{"git -C", "worktree", "comparedTo", "merge-base", "ls-files --others"}

// reviewPassage es un pasaje pineado: su texto ya extraído y el fichero de origen.
type reviewPassage struct {
	name string
	file string
	text string
}

func assertReviewBoundary(t *testing.T, passages []reviewPassage) {
	t.Helper()
	for _, p := range passages {
		if strings.TrimSpace(p.text) == "" {
			t.Errorf("%s (%s): pasaje vacío", p.name, p.file)
			continue
		}
		if m := reviewBoundaryVocabulary.FindString(p.text); m != "" {
			t.Errorf("%s (%s): usa vocabulario de la otra frontera: %q", p.name, p.file, m)
		}
		for _, lit := range reviewBoundaryLiterals {
			if strings.Contains(p.text, lit) {
				t.Errorf("%s (%s): contiene el literal %q", p.name, p.file, lit)
			}
		}
	}
}

func assertNoHeadRangeScope(t *testing.T, name, text string) {
	t.Helper()
	if strings.Contains(text, "...HEAD") {
		t.Errorf("%s: no debe tomar `...HEAD` como ámbito del diff (perdería los cambios sin commit)", name)
	}
}

func assertContainsAll(t *testing.T, name, text string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s: falta %q", name, want)
		}
	}
}

// linesWithPrefix devuelve las líneas de text que empiezan por prefix.
func linesWithPrefix(text, prefix string) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// Frases nuevas o reescritas en sdd-verify/SKILL.md (DD-07, DD-10).
const (
	verifySkillDescription = `description: "Trigger: SDD verification requested, accepted as an offer, or set by preference. Run practical diagnostics, per role when a role is given."`
	verifySkillActivation  = "Run when verification is requested by the user, accepted as an offer, or selected by the session review preference. Verification is optional, not a prerequisite for archive."
	verifySkillRoleRule    = "- When a role is supplied, inspect only that role's diff and add the technical compliance section; follow `references/technical-compliance.md`."
	verifySkillRoleGate    = "| A role is supplied | Resolve its diff and write its `verify_file` with technical compliance and `**Estado:**`; see `references/technical-compliance.md`. |"
)

func TestIndependentReviewBoundaryVerifyPassages(t *testing.T) {
	skill := MustRead("skills/sdd-verify/SKILL.md")
	compliance := MustRead("skills/sdd-verify/references/technical-compliance.md")
	report := MustRead("skills/sdd-verify/references/report-format.md")

	phrases := []reviewPassage{
		{"descripción", "skills/sdd-verify/SKILL.md", verifySkillDescription},
		{"activación", "skills/sdd-verify/SKILL.md", verifySkillActivation},
		{"regla de ámbito por rol", "skills/sdd-verify/SKILL.md", verifySkillRoleRule},
		{"fila de decisión por rol", "skills/sdd-verify/SKILL.md", verifySkillRoleGate},
	}
	for _, p := range phrases {
		if !strings.Contains(skill, p.text) {
			t.Errorf("%s: la frase fijada no está en %s: %q", p.name, p.file, p.text)
		}
	}

	technicalSection := linesWithPrefix(report, "- **Cumplimiento técnico:**")
	stateSection := linesWithPrefix(report, "- **Estado:**")
	verdictRule := linesWithPrefix(report, "- **Barrera:**")

	passages := append(phrases,
		reviewPassage{"referencia de cumplimiento técnico", "skills/sdd-verify/references/technical-compliance.md", compliance},
		reviewPassage{"elemento Cumplimiento técnico", "skills/sdd-verify/references/report-format.md", technicalSection},
		reviewPassage{"elemento Estado", "skills/sdd-verify/references/report-format.md", stateSection},
		reviewPassage{"regla de la barrera", "skills/sdd-verify/references/report-format.md", verdictRule},
	)
	assertReviewBoundary(t, passages)

	// Matriz de amenazas, fila «Selección de repositorio git»: la resolución del
	// diff del rol sale de `axiom branch status --json`, siempre con `git -C`.
	assertContainsAll(t, "technical-compliance.md", compliance, append([]string{"axiom branch status"}, reviewScopeMarkers...))
	assertNoHeadRangeScope(t, "technical-compliance.md", compliance)
	assertNoHeadRangeScope(t, "report-format.md", technicalSection+stateSection)

	// Forma del informe (REQ-28.4, DD-07b).
	assertContainsAll(t, "report-format.md (Cumplimiento técnico)", technicalSection, []string{"verdict:"})
	assertContainsAll(t, "report-format.md (Estado)", stateSection, []string{"**Estado:** limpio", "**Estado:** no limpio"})
	assertContainsAll(t, "technical-compliance.md", compliance, []string{
		"verify-report/{role}", "comparedTo", "no limpio — diff del rol vacío", "no aplicable",
	})
}
