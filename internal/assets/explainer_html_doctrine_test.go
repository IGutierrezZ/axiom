package assets

import (
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/explainer"
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

// Los cuatro términos de las restricciones del HTML (prov:GAT:1) deben estar en el documento compartido, en las dos
// skills y en las seis copias por agente: si falta uno en cualquiera, el modelo de ese agente no los recibe.
var explainerHTMLTerms = []string{"castellano", "<script>", "external", "axiom-source-digest", "data-axiom-ref"}

var explainerHTMLCarriers = []string{
	"skills/_shared/sdd-explainer-html.md",
	"skills/sdd-spec/SKILL.md",
	"skills/sdd-design/SKILL.md",
	"claude/agents/sdd-spec.md",
	"claude/agents/sdd-design.md",
	"cursor/agents/sdd-spec.md",
	"cursor/agents/sdd-design.md",
	"copilot/agents/sdd-spec.agent.md",
	"copilot/agents/sdd-design.agent.md",
}

func TestExplainerHTMLConstraintsInEveryCarrier(t *testing.T) {
	for _, path := range explainerHTMLCarriers {
		requireContains(t, path, MustRead(path), explainerHTMLTerms...)
	}
}

func TestExplainerHTMLPhaseSteps(t *testing.T) {
	for _, kind := range []string{"spec", "design"} {
		skill := MustRead("skills/sdd-" + kind + "/SKILL.md")
		requireContains(t, "sdd-"+kind+" skill", skill,
			"governance.kickoff.explainerHtml: yes",
			"explainer."+kind+".expectedDigest",
			"resumen-"+kind+".html",
			"../_shared/sdd-explainer-html.md",
			"`state: fresh`",
			"With `engram` or `none`, write no HTML",
			"reopened by a rejection or a correction",
			"never blocks the phase",
		)
	}
	shared := MustRead("skills/_shared/sdd-explainer-html.md")
	requireContains(t, "sdd-explainer-html.md", shared,
		"after a manual edit of the md",
		"it blocks nothing",
		"With `engram` or `none`, write no HTML",
		"Never compute a digest yourself",
		"Este resumen no sustituye al documento: lo que se aprueba es el md.",
	)
}

// fencedBlock devuelve el contenido del primer bloque con la etiqueta dada.
func fencedBlock(t *testing.T, text, label string) string {
	t.Helper()
	open := "```" + label + "\n"
	from := strings.Index(text, open)
	if from < 0 {
		t.Fatalf("no hay bloque ```%s", label)
	}
	rest := text[from+len(open):]
	to := strings.Index(rest, "\n```")
	if to < 0 {
		t.Fatalf("el bloque ```%s no se cierra", label)
	}
	return rest[:to]
}

// El esqueleto que lee el modelo es el que valida el contrato, y cita exactamente los requisitos del md de ejemplo.
func TestExplainerHTMLSkeletonMatchesExample(t *testing.T) {
	shared := MustRead("skills/_shared/sdd-explainer-html.md")
	md := fencedBlock(t, shared, "markdown")
	html := []byte(fencedBlock(t, shared, "html"))

	if findings := explainer.Validate(html); len(findings) != 0 {
		t.Fatalf("el esqueleto incumple el contrato: %+v", findings)
	}
	cited := explainer.CitedIDs(html)
	if len(cited) == 0 {
		t.Fatal("el esqueleto no cita ningún ID")
	}
	var required []string
	for _, match := range regexp.MustCompile(`(?m)^#{2,4} (RF-[A-Z]+-\d+)`).FindAllStringSubmatch(md, -1) {
		required = append(required, match[1])
	}
	if len(required) == 0 {
		t.Fatal("el md de ejemplo no tiene requisitos con ID")
	}
	inMD := map[string]bool{}
	for _, id := range required {
		inMD[id] = true
	}
	inHTML := map[string]bool{}
	for _, id := range cited {
		inHTML[id] = true
		if !inMD[id] {
			t.Errorf("el esqueleto cita %q, que no existe en el md de ejemplo", id)
		}
	}
	for _, id := range required {
		if !inHTML[id] {
			t.Errorf("el requisito %q del md de ejemplo no está citado en el esqueleto", id)
		}
	}
}
