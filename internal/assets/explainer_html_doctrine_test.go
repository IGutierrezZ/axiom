package assets

import (
	"strings"
	"testing"
)

// html-explicativo (C-06-b): pins de texto de la doctrina del HTML explicativo. El orquestador pregunta una vez
// por cambio, el contrato de estado nombra los campos que consumen las fases y la frescura nunca gobierna el flujo.

func TestExplainerOrchestratorDoctrine(t *testing.T) {
	section := sddSharedSection(t, "SDD Change Kickoff and Block Gates")
	from := strings.Index(section, "**Explainer HTML.**")
	if from < 0 {
		t.Fatal("la sección del kickoff no tiene el párrafo **Explainer HTML.**")
	}
	paragraph := section[from:]
	requireContains(t, "párrafo Explainer HTML", paragraph,
		"`Axiom SDD kickoff explainer:`",
		"after `axiom branch ensure` and before `axiom sdd kickoff seal`",
		"`project: ask`",
		"once per change",
		"Never ask it in ODD",
		"`--infer`",
		"when resuming",
		"never a pre-flight group",
		"`--explainer-html yes|no`",
		"`governance.kickoff.explainerHtml`",
		"`spec` and `design` stop",
		"never in place of the md",
		"never changes `nextRecommended`",
	)
	if strings.Contains(paragraph, "<!-- sdd-orchestrator-section") {
		t.Error("el párrafo Explainer HTML. debe ser el último de la sección, antes del marcador de cierre")
	}
	if !strings.HasSuffix(strings.TrimSpace(section), "digest.") {
		t.Error("el párrafo Explainer HTML. debe cerrar la sección")
	}
	// El orden: la pregunta va antes del sellado.
	if strings.Index(paragraph, "after `axiom branch ensure`") > strings.Index(paragraph, "`kickoff seal`") {
		t.Error("la pregunta debe preceder al sellado")
	}
	// Los párrafos ajenos no se tocan: el de publicación sigue antes del nuevo.
	if strings.Index(section, "A gate approval never authorizes delivery") > from {
		t.Error("el párrafo Explainer HTML. debe ir después del párrafo final de las puertas")
	}
}

func TestExplainerStatusContractDoctrine(t *testing.T) {
	contract := MustRead("skills/_shared/sdd-status-contract.md")
	requireContains(t, "sdd-status-contract.md", contract,
		"explainerHtml?",
		"`kickoff.explainerHtml`",
		"`explainer` (optional, at the root of the status)",
		"`expectedDigest`",
		"`reason`",
		"`fresh`, `stale` or `missing`",
		"also in an ISO change",
		"never compute a digest by hand",
	)
}
