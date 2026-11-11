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

// Frases nuevas del juicio y de las reglas de cobertura (DD-08, DD-09, DD-10).
const (
	judgmentSkillActivation = "The diff of one role of an SDD change is a valid target: accepting the orchestrator's offer, or a session preference that includes judgment, counts as that explicit request (see `SDD Role Target` in `references/prompts-and-formats.md`)."
	judgmentSkillGate       = "| Target is an SDD role diff | Follow `SDD Role Target` in `references/prompts-and-formats.md`: only after a clean verify, fixes only inside the authorized edit roots. |"
	coverageOwnTableRule    = "- Your own coverage table does not replace the independent coverage review (`../_shared/sdd-coverage-review.md`); never present it as that review."
)

// sectionUntilNextH2 devuelve el cuerpo de un encabezado `## <title>` hasta el
// siguiente encabezado `## ` (o el final del fichero).
func sectionUntilNextH2(content, title string) string {
	start := strings.Index(content, "\n## "+title+"\n")
	if start < 0 {
		return ""
	}
	body := content[start+1:]
	if next := strings.Index(body[3:], "\n## "); next >= 0 {
		return body[:next+3]
	}
	return body
}

func TestIndependentReviewBoundaryJudgmentAndCoveragePassages(t *testing.T) {
	judgment := MustRead("skills/judgment-day/SKILL.md")
	formats := MustRead("skills/judgment-day/references/prompts-and-formats.md")
	coverage := MustRead("skills/_shared/sdd-coverage-review.md")
	specSkill := MustRead("skills/sdd-spec/SKILL.md")
	designSkill := MustRead("skills/sdd-design/SKILL.md")
	convention := MustRead("skills/_shared/openspec-convention.md")

	for _, c := range []struct{ name, content, phrase string }{
		{"activación del juicio", judgment, judgmentSkillActivation},
		{"fila de decisión del juicio", judgment, judgmentSkillGate},
		{"regla de sdd-spec", specSkill, coverageOwnTableRule},
		{"regla de sdd-design", designSkill, coverageOwnTableRule},
	} {
		if !strings.Contains(c.content, c.phrase) {
			t.Errorf("%s: la frase fijada no está en su fichero: %q", c.name, c.phrase)
		}
	}

	roleTarget := sectionUntilNextH2(formats, "SDD Role Target")
	assertReviewBoundary(t, []reviewPassage{
		{"activación del juicio", "skills/judgment-day/SKILL.md", judgmentSkillActivation},
		{"fila de decisión del juicio", "skills/judgment-day/SKILL.md", judgmentSkillGate},
		{"SDD Role Target", "skills/judgment-day/references/prompts-and-formats.md", roleTarget},
		{"revisión de cobertura", "skills/_shared/sdd-coverage-review.md", coverage},
		{"regla de sdd-spec", "skills/sdd-spec/SKILL.md", coverageOwnTableRule},
		{"regla de sdd-design", "skills/sdd-design/SKILL.md", coverageOwnTableRule},
	})

	// Matriz de amenazas, fila «Selección de repositorio git».
	assertContainsAll(t, "SDD Role Target", roleTarget, append([]string{"axiom branch status", "allowedEditRoots", "technicalContext.reference", "**Estado:** limpio"}, reviewScopeMarkers...))
	assertNoHeadRangeScope(t, "SDD Role Target", roleTarget)

	// Mandato de cobertura (DD-09).
	assertContainsAll(t, "sdd-coverage-review.md", coverage, []string{
		"jd-judge-a", "jd-judge-b", `{"gaps":[`, "do not apply", "Ronda N", "coverage-review.spec.md", "coverage-review.design.md",
		"sdd/{change}/coverage-review/spec", "sdd/{change}/coverage-review/design",
	})

	// Los informes nuevos constan en la convención compartida (DD-07, DD-08, DD-09).
	assertContainsAll(t, "openspec-convention.md", convention, []string{
		"verify-report.<role>.md", "review-ledger.<role>.md", "coverage-review.spec.md", "coverage-review.design.md",
		"sdd/{change-name}/coverage-review/spec", "sdd/{change-name}/review-ledger/{role}",
	})

	// La prueba del juicio autónomo sigue vigente: la frase sin recibo no se toca.
	if !strings.Contains(strings.ToLower(judgment), "issues no receipt") {
		t.Error("judgment-day/SKILL.md debe conservar «issues no receipt»")
	}
}
