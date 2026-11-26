package kickoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func TestResolveExplainerHTML(t *testing.T) {
	tests := []struct {
		project       workspace.ExplainerHTMLMode
		flag          ExplainerHTML
		want          ExplainerHTML
		wantDefaulted bool
		wantErr       string
	}{
		{workspace.ExplainerHTMLAlways, "", ExplainerHTMLYes, false, ""},
		{workspace.ExplainerHTMLAlways, ExplainerHTMLYes, ExplainerHTMLYes, false, ""},
		{workspace.ExplainerHTMLAlways, ExplainerHTMLNo, "", false, "--explainer-html no contradice sdd.explainer_html: always de axiom.yaml; omite la bandera o cambia la declaración"},
		{workspace.ExplainerHTMLNever, "", ExplainerHTMLNo, false, ""},
		{workspace.ExplainerHTMLNever, ExplainerHTMLNo, ExplainerHTMLNo, false, ""},
		{workspace.ExplainerHTMLNever, ExplainerHTMLYes, "", false, "--explainer-html yes contradice sdd.explainer_html: never de axiom.yaml; omite la bandera o cambia la declaración"},
		{workspace.ExplainerHTMLAsk, ExplainerHTMLYes, ExplainerHTMLYes, false, ""},
		{workspace.ExplainerHTMLAsk, ExplainerHTMLNo, ExplainerHTMLNo, false, ""},
		{workspace.ExplainerHTMLAsk, "", ExplainerHTMLNo, true, ""},
		// Sin clave en axiom.yaml el modo efectivo es ask; el valor vacío se trata igual.
		{"", ExplainerHTMLYes, ExplainerHTMLYes, false, ""},
		{"", "", ExplainerHTMLNo, true, ""},
	}
	for _, tt := range tests {
		name := string(tt.project) + "+" + string(tt.flag)
		t.Run(name, func(t *testing.T) {
			got, defaulted, err := ResolveExplainerHTML(tt.project, tt.flag)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, se esperaba %q", err, tt.wantErr)
				}
				if got != "" || defaulted {
					t.Fatalf("con error no hay valor ni defecto: (%q, %v)", got, defaulted)
				}
				return
			}
			if err != nil || got != tt.want || defaulted != tt.wantDefaulted {
				t.Fatalf("ResolveExplainerHTML = (%q, %v, %v), se esperaba (%q, %v, nil)", got, defaulted, err, tt.want, tt.wantDefaulted)
			}
		})
	}
}

func TestParseSealArgsExplainerHTML(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    ExplainerHTML
		wantErr string
	}{
		{name: "yes", args: []string{"--change", "x", "--execution-style", "checkpointed", "--explainer-html", "yes"}, want: ExplainerHTMLYes},
		{name: "no", args: []string{"--change", "x", "--execution-style", "continuous", "--explainer-html", "no"}, want: ExplainerHTMLNo},
		{name: "sin bandera", args: []string{"--change", "x", "--execution-style", "continuous"}, want: ""},
		{name: "valor desconocido", args: []string{"--change", "x", "--execution-style", "continuous", "--explainer-html", "quizás"}, wantErr: "--explainer-html desconocido \"quizás\"; opciones: yes, no"},
		{name: "valor ausente", args: []string{"--change", "x", "--execution-style", "continuous", "--explainer-html"}, wantErr: "--explainer-html"},
		{name: "infer se rechaza", args: []string{"--change", "x", "--infer", "--explainer-html", "yes"}, wantErr: "--explainer-html no se puede combinar con --infer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSealArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseSealArgs() error = %v, se esperaba %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSealArgs() error = %v", err)
			}
			if parsed.ExplainerHTML != tt.want {
				t.Fatalf("ExplainerHTML = %q, want %q", parsed.ExplainerHTML, tt.want)
			}
			if got := parsed.ToKickoff("x").Config.ExplainerHTML; got != tt.want {
				t.Fatalf("ToKickoff().Config.ExplainerHTML = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKickoffValidateExplainerHTML(t *testing.T) {
	for _, style := range []ExecutionStyle{ExecutionContinuous, ExecutionCheckpointed} {
		for _, v := range []ExplainerHTML{"", ExplainerHTMLYes, ExplainerHTMLNo} {
			k := sampleKickoff("inc-99-example")
			k.Config.ExecutionStyle = style
			k.Config.ExplainerHTML = v
			if err := k.Validate(); err != nil {
				t.Errorf("Validate(%s, %q) = %v", style, v, err)
			}
		}
	}
	k := sampleKickoff("inc-99-example")
	k.Config.ExplainerHTML = ExplainerHTML("quizás")
	err := k.Validate()
	if err == nil || !strings.Contains(err.Error(), `explainer_html desconocido "quizás"`) || !strings.Contains(err.Error(), "yes, no") {
		t.Fatalf("Validate() = %v, se esperaba el rechazo con los admitidos", err)
	}
}

func TestEffectiveExplainerHTML(t *testing.T) {
	if EffectiveExplainerHTML("") != ExplainerHTMLNo || EffectiveExplainerHTML(ExplainerHTMLNo) != ExplainerHTMLNo || EffectiveExplainerHTML(ExplainerHTMLYes) != ExplainerHTMLYes {
		t.Fatal("EffectiveExplainerHTML debe tratar la clave ausente como no")
	}
}

// El valor sellado viaja por disco: yes y no se releen como cadenas (yaml.v3
// entrecomilla el texto no) y la ausencia se lee como no efectivo.
func TestSealRoundTripsExplainerHTML(t *testing.T) {
	for _, v := range []ExplainerHTML{ExplainerHTMLYes, ExplainerHTMLNo} {
		changeRoot := t.TempDir()
		k := sampleKickoff("inc-99-example")
		k.Config.ExplainerHTML = v
		if _, ok, err := Seal(changeRoot, k); err != nil || !ok {
			t.Fatalf("Seal() = (_, %v, %v)", ok, err)
		}
		loaded, err := Load(changeRoot)
		if err != nil || loaded == nil || loaded.Config.ExplainerHTML != v {
			t.Fatalf("Load() = %+v, %v; se esperaba explainer_html %q", loaded, err, v)
		}
	}

	// Sin la clave (sellado sin valor): Load da "" y el efectivo es no.
	changeRoot := t.TempDir()
	if _, ok, err := Seal(changeRoot, sampleKickoff("inc-99-example")); err != nil || !ok {
		t.Fatalf("Seal() = (_, %v, %v)", ok, err)
	}
	raw, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil || strings.Contains(string(raw), "explainer_html") {
		t.Fatalf("sin valor no se escribe la clave: %v\n%s", err, raw)
	}
	loaded, err := Load(changeRoot)
	if err != nil || loaded == nil || loaded.Config.ExplainerHTML != "" || EffectiveExplainerHTML(loaded.Config.ExplainerHTML) != ExplainerHTMLNo {
		t.Fatalf("Load() sin clave = %+v, %v", loaded, err)
	}
}

// El retrosellado nunca infiere la decisión del HTML explicativo.
func TestInferKickoffNeverWritesExplainerHTML(t *testing.T) {
	changeRoot := t.TempDir()
	designPath := writeDesignMD(t, changeRoot, "# Diseno\n")
	k, _, err := InferKickoff(designPath, repoLikeWorkspaceConfig())
	if err != nil {
		t.Fatalf("InferKickoff() error = %v", err)
	}
	if k.Config.ExplainerHTML != "" {
		t.Fatalf("ExplainerHTML = %q, el retrosellado no la infiere", k.Config.ExplainerHTML)
	}
	raw, err := yaml.Marshal(k)
	if err != nil || strings.Contains(string(raw), "explainer_html") {
		t.Fatalf("el kickoff inferido no lleva la clave: %v\n%s", err, raw)
	}
}

// legacyKickoffFlowConfig es el bloque kickoff tal y como lo declaraba el
// esquema anterior a explainer_html. Se congela a propósito: prueba que un
// binario anterior (decodificación sin KnownFields) sigue leyendo un kickoff con
// la clave nueva.
type legacyKickoffFlowConfig struct {
	FlowMode             FlowMode        `yaml:"flow_mode"`
	ExecutionStyle       ExecutionStyle  `yaml:"execution_style"`
	ExecutionStyleSource string          `yaml:"execution_style_source"`
	HandoffPolicy        HandoffPolicy   `yaml:"handoff_policy,omitempty"`
	Roles                []KickoffRole   `yaml:"roles"`
	DocsPublication      DocsPublication `yaml:"docs_publication,omitempty"`
}

type legacyKickoffDocument struct {
	Schema    string                  `yaml:"schema"`
	Change    string                  `yaml:"change"`
	SealedAt  time.Time               `yaml:"sealed_at"`
	SealedBy  string                  `yaml:"sealed_by"`
	Config    legacyKickoffFlowConfig `yaml:"kickoff"`
	Lifecycle Lifecycle               `yaml:"lifecycle"`
}

func TestLegacyBinaryReadsAKickoffWithExplainerHTML(t *testing.T) {
	changeRoot := t.TempDir()
	k := sampleKickoff("inc-99-example")
	k.Config.ExplainerHTML = ExplainerHTMLYes
	if _, ok, err := Seal(changeRoot, k); err != nil || !ok {
		t.Fatalf("Seal() = (_, %v, %v)", ok, err)
	}
	raw, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil || !strings.Contains(string(raw), `explainer_html: "yes"`) {
		t.Fatalf("el kickoff sellado debe llevar explainer_html entrecomillado: %v\n%s", err, raw)
	}
	var legacy legacyKickoffDocument
	if err := yaml.Unmarshal(raw, &legacy); err != nil {
		t.Fatalf("un binario anterior debe leer la clave nueva sin error: %v", err)
	}
	if legacy.Change != "inc-99-example" || legacy.Config.FlowMode != FlowSDD || len(legacy.Config.Roles) != 1 {
		t.Fatalf("lectura anterior inesperada: %+v", legacy)
	}
}

// Un kickoff sin la clave (anterior a esta función) sigue cargando y vale no.
func TestLegacyKickoffWithoutExplainerHTMLLoadsAsNo(t *testing.T) {
	parsed, err := ParseKickoff([]byte(validKickoffYAML()))
	if err != nil {
		t.Fatalf("ParseKickoff() error = %v", err)
	}
	if parsed.Config.ExplainerHTML != "" || EffectiveExplainerHTML(parsed.Config.ExplainerHTML) != ExplainerHTMLNo {
		t.Fatalf("ExplainerHTML = %q, se esperaba vacío y no efectivo", parsed.Config.ExplainerHTML)
	}
}
