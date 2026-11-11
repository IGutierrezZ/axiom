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

// reviewBoundaryViolations devuelve las violaciones de frontera de los pasajes; es la lógica que
// assertReviewBoundary reporta y que la prueba de mutación ejercita sin tocar ficheros.
func reviewBoundaryViolations(passages []reviewPassage) []string {
	var violations []string
	for _, p := range passages {
		if strings.TrimSpace(p.text) == "" {
			violations = append(violations, p.name+" ("+p.file+"): pasaje vacío")
			continue
		}
		if m := reviewBoundaryVocabulary.FindString(p.text); m != "" {
			violations = append(violations, p.name+" ("+p.file+"): usa vocabulario de la otra frontera: "+m)
		}
		for _, lit := range reviewBoundaryLiterals {
			if strings.Contains(p.text, lit) {
				violations = append(violations, p.name+" ("+p.file+"): contiene el literal "+lit)
			}
		}
	}
	return violations
}

func assertReviewBoundary(t *testing.T, passages []reviewPassage) {
	t.Helper()
	for _, v := range reviewBoundaryViolations(passages) {
		t.Error(v)
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

// Frases nuevas del orquestador compartido (DD-06, REQ-28.10, REQ-28.12).
const (
	gatesContinuousFullstackPhrase = "`continuous` is valid only with the single role `fullstack`: with specialized roles, seal `checkpointed`; the command rejects `continuous` (also when derived from `--from-session-pace auto`) with any other roster and writes nothing, and only the retro-sealing `--infer` is exempt."
	gatesStopResultsPhrase         = "Each stop also presents the independent review results that exist at that moment and declares the ones that are absent, as `SDD Independent Review` describes; they never change the gate or its three exits."
	dependencyGraphPhrase          = "The per-role verify, the judgment, and the coverage review are offered or applied as `SDD Independent Review` describes; they are not phases of this graph."
	absentResultsPhrase            = "An independent review that did not run, did not answer, or was not clean is presented as such and never as clean."
)

const (
	independentReviewSectionStart = "<!-- sdd-orchestrator-section:SDD Independent Review:start -->"
	independentReviewSectionEnd   = "<!-- sdd-orchestrator-section:SDD Independent Review:end -->"
	independentReviewTemplateTag  = "{{GENTLE_AI_SDD_SECTION:SDD Independent Review}}"
	blockGatesTemplateTag         = "{{GENTLE_AI_SDD_SECTION:SDD Change Kickoff and Block Gates}}"
)

var independentReviewRuntimes = []string{
	"antigravity", "claude", "codex", "cursor", "gemini", "generic", "hermes", "kimi", "kiro", "opencode", "qwen", "windsurf",
}

func TestIndependentReviewBoundaryOrchestratorPassages(t *testing.T) {
	shared := MustRead("skills/_shared/sdd-orchestrator-sections.md")
	start := strings.Index(shared, independentReviewSectionStart)
	end := strings.Index(shared, independentReviewSectionEnd)
	if start < 0 || end < start {
		t.Fatal("la sección compartida SDD Independent Review no está delimitada por sus marcadores")
	}
	section := shared[start+len(independentReviewSectionStart) : end]
	if strings.Count(shared, independentReviewSectionStart) != 1 {
		t.Error("la sección compartida debe declararse una sola vez")
	}

	for _, c := range []struct{ name, phrase string }{
		{"frase de continuous en las puertas", gatesContinuousFullstackPhrase},
		{"frase de resultados en las paradas", gatesStopResultsPhrase},
		{"remisión del grafo de dependencias", dependencyGraphPhrase},
	} {
		if !strings.Contains(shared, c.phrase) {
			t.Errorf("%s: la frase fijada no está en sdd-orchestrator-sections.md: %q", c.name, c.phrase)
		}
	}
	workflow := MustRead("claude/sdd-orchestrator-workflow.md")
	if !strings.Contains(workflow, dependencyGraphPhrase) {
		t.Errorf("claude/sdd-orchestrator-workflow.md: falta la remisión del grafo: %q", dependencyGraphPhrase)
	}

	assertReviewBoundary(t, []reviewPassage{
		{"SDD Independent Review", "skills/_shared/sdd-orchestrator-sections.md", section},
		{"frase de continuous en las puertas", "skills/_shared/sdd-orchestrator-sections.md", gatesContinuousFullstackPhrase},
		{"frase de resultados en las paradas", "skills/_shared/sdd-orchestrator-sections.md", gatesStopResultsPhrase},
		{"remisión del grafo de dependencias", "skills/_shared/sdd-orchestrator-sections.md", dependencyGraphPhrase},
	})

	// La interacción de la preferencia no reutiliza los marcadores del pre-vuelo (REQ-28.14).
	for _, forbidden := range []string{"Axiom SDD preflight", "preflight 4/", "Gentle AI SDD preflight"} {
		if strings.Contains(section, forbidden) {
			t.Errorf("la sección contiene el marcador del pre-vuelo %q", forbidden)
		}
	}
	// Resultados ausentes (REQ-28.10) y contrato de la preferencia (REQ-28.14, DD-04, DD-05, DD-07c, DD-11).
	assertContainsAll(t, "SDD Independent Review", section, []string{
		absentResultsPhrase,
		"`after_apply`", "`coverage`", "`verify`", "`verify-judgment`", "`none`", "`yes`", "`no`",
		"After apply: <label>; Coverage: <label>",
		"never write it to `kickoff.yaml`, `gates.yaml`, or memory",
		"explicit exception to routing only by `nextRecommended` and dependencies",
		"**Estado:** limpio", "SDD Role Target", "skills/_shared/sdd-coverage-review.md",
		"Never simulate them in your own context",
	})
}

// TestIndependentReviewBoundaryTemplateParity: toda plantilla que proyecta las puertas proyecta una vez la sección nueva,
// detrás de ellas.
func TestIndependentReviewBoundaryTemplateParity(t *testing.T) {
	for _, runtime := range independentReviewRuntimes {
		path := runtime + "/sdd-orchestrator.md"
		content := MustRead(path)
		gates := strings.Index(content, blockGatesTemplateTag)
		if gates < 0 {
			continue
		}
		if got := strings.Count(content, independentReviewTemplateTag); got != 1 {
			t.Errorf("%s: la sección SDD Independent Review se proyecta %d veces, se esperaba 1", path, got)
			continue
		}
		if strings.Index(content, independentReviewTemplateTag) < gates {
			t.Errorf("%s: la sección SDD Independent Review debe ir detrás de las puertas", path)
		}
		if !strings.Contains(content, "\n### SDD Independent Review\n") {
			t.Errorf("%s: falta el encabezado de la sección SDD Independent Review", path)
		}
	}
}

// pinnedReviewPhrases agrupa todas las frases nuevas o reescritas que el pin fija, con el fichero
// donde deben estar (REQ-28.11, DD-10).
var pinnedReviewPhrases = []struct{ file, phrase string }{
	{"skills/sdd-verify/SKILL.md", verifySkillDescription},
	{"skills/sdd-verify/SKILL.md", verifySkillActivation},
	{"skills/sdd-verify/SKILL.md", verifySkillRoleRule},
	{"skills/sdd-verify/SKILL.md", verifySkillRoleGate},
	{"skills/judgment-day/SKILL.md", judgmentSkillActivation},
	{"skills/judgment-day/SKILL.md", judgmentSkillGate},
	{"skills/sdd-spec/SKILL.md", coverageOwnTableRule},
	{"skills/sdd-design/SKILL.md", coverageOwnTableRule},
	{"skills/_shared/sdd-orchestrator-sections.md", gatesContinuousFullstackPhrase},
	{"skills/_shared/sdd-orchestrator-sections.md", gatesStopResultsPhrase},
	{"skills/_shared/sdd-orchestrator-sections.md", dependencyGraphPhrase},
	{"claude/sdd-orchestrator-workflow.md", dependencyGraphPhrase},
}

// TestIndependentReviewBoundaryPinDetectsMutations comprueba, solo en memoria y sin modificar
// ficheros, que el pin falla si una frase fijada se retira o si un pasaje adopta el vocabulario
// de la otra frontera, y que con el contenido real pasa.
func TestIndependentReviewBoundaryPinDetectsMutations(t *testing.T) {
	for _, pinned := range pinnedReviewPhrases {
		content := MustRead(pinned.file)
		if !strings.Contains(content, pinned.phrase) {
			t.Errorf("%s: la frase fijada no está: %q", pinned.file, pinned.phrase)
			continue
		}
		retired := strings.Replace(content, pinned.phrase, "", 1)
		// La mutación retira la frase del contenido en memoria: la comprobación de presencia
		// de las pruebas de pasajes fallaría con ese contenido.
		if strings.Contains(retired, pinned.phrase) {
			t.Errorf("%s: la frase %q aparece más de una vez o la mutación no la retiró", pinned.file, pinned.phrase)
		}
	}

	clean := reviewPassage{"pasaje", "mutacion.md", "Run the per-role verify and present the result."}
	if got := reviewBoundaryViolations([]reviewPassage{clean}); len(got) != 0 {
		t.Fatalf("un pasaje limpio no debe tener violaciones: %v", got)
	}
	for _, injected := range []string{"issue a receipt", "keep the lineage", "run axiom review start", "burn the candidate", "RDD enabled", "see sdd-attempt", "acknowledged"} {
		mutated := clean
		mutated.text += " " + injected
		if got := reviewBoundaryViolations([]reviewPassage{mutated}); len(got) == 0 {
			t.Errorf("el pin no detecta %q", injected)
		}
	}
}

// TestIndependentReviewBoundaryIsNotAWholeFileSweep: los ficheros preexistentes nombran la frontera a propósito, de modo que
// un barrido de fichero completo fallaría; el pin inspecciona solo los pasajes nuevos o modificados (REQ-28.11).
func TestIndependentReviewBoundaryIsNotAWholeFileSweep(t *testing.T) {
	deliberate := []struct{ file, mention string }{
		{"skills/sdd-verify/SKILL.md", "SDD never offers, launches, or consumes RDD."},
		{"skills/judgment-day/SKILL.md", "issues no receipt"},
		{"skills/_shared/sdd-orchestrator-sections.md", "receipt-driven development (RDD)"},
	}
	for _, d := range deliberate {
		content := MustRead(d.file)
		if !strings.Contains(content, d.mention) {
			t.Errorf("%s: falta la mención preexistente que declara la frontera: %q", d.file, d.mention)
		}
		if len(reviewBoundaryViolations([]reviewPassage{{"fichero completo", d.file, content}})) == 0 {
			t.Errorf("%s: se esperaba que el fichero completo incumpliera el vocabulario (mención deliberada de la frontera)", d.file)
		}
		for _, pinned := range pinnedReviewPhrases {
			if pinned.file == d.file && len(pinned.phrase) >= len(content) {
				t.Errorf("%s: un pasaje pineado cubre el fichero completo", d.file)
			}
		}
	}
}
