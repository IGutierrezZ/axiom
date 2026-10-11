package specbaseline

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func TestProfileCompileFromTheDeclaration(t *testing.T) {
	valid := workspace.BaselineDocument{
		Domain: "requisitos-funcionales",
		Path:   "docs/01_Requisitos_Funcionales.md",
		Requirement: &workspace.RequirementProfile{
			Heading: `^#+ (?P<id>RF-\d+): (?P<title>.+)$`,
			Level:   3,
			Render:  "### {id}: {title}",
		},
	}

	t.Run("los valores por defecto son identidad id y escenarios include", func(t *testing.T) {
		profile, err := CompileProfile(valid)
		if err != nil {
			t.Fatalf("CompileProfile() error = %v", err)
		}
		if profile.Identity != workspace.IdentityID {
			t.Errorf("Identity = %q, se esperaba id", profile.Identity)
		}
		if profile.Scenarios != workspace.ScenariosInclude {
			t.Errorf("Scenarios = %q, se esperaba include", profile.Scenarios)
		}
		if profile.Level != 3 || profile.Render != "### {id}: {title}" || profile.Heading == nil {
			t.Errorf("perfil = %#v, no refleja la declaración", profile)
		}
	})

	t.Run("la identidad y los escenarios declarados se respetan", func(t *testing.T) {
		doc := valid
		requirement := *valid.Requirement
		requirement.Identity = workspace.IdentityText
		requirement.Level = 4
		doc.Requirement = &requirement
		doc.Scenarios = workspace.ScenariosOmit
		profile, err := CompileProfile(doc)
		if err != nil {
			t.Fatalf("CompileProfile() error = %v", err)
		}
		if profile.Identity != workspace.IdentityText || profile.Scenarios != workspace.ScenariosOmit || profile.Level != 4 {
			t.Errorf("perfil = %#v, no refleja la declaración", profile)
		}
	})

	failures := []struct {
		name   string
		mutate func(*workspace.BaselineDocument)
		want   string
	}{
		{name: "documento de prosa", mutate: func(d *workspace.BaselineDocument) { d.Requirement = nil }, want: "prosa"},
		{name: "patrón que no compila", mutate: func(d *workspace.BaselineDocument) { d.Requirement.Heading = `^(?P<id>RF` }, want: "heading"},
		{name: "patrón sin los grupos id y title", mutate: func(d *workspace.BaselineDocument) { d.Requirement.Heading = `^### RF-\d+` }, want: "id"},
		{name: "nivel por debajo de 1", mutate: func(d *workspace.BaselineDocument) { d.Requirement.Level = 0 }, want: "level"},
		{name: "nivel por encima de 6", mutate: func(d *workspace.BaselineDocument) { d.Requirement.Level = 7 }, want: "level"},
		{name: "plantilla sin marcadores", mutate: func(d *workspace.BaselineDocument) { d.Requirement.Render = "### texto" }, want: "render"},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			doc := valid
			requirement := *valid.Requirement
			doc.Requirement = &requirement
			tt.mutate(&doc)
			_, err := CompileProfile(doc)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, se esperaba que mencionara %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), "requisitos-funcionales") {
				t.Errorf("el error debe nombrar el dominio: %v", err)
			}
		})
	}
}

func TestProfileMatchesTheWholeHeadingLine(t *testing.T) {
	profile := requirementProfile(t)
	tests := []struct {
		name      string
		line      string
		wantID    string
		wantTitle string
		wantOK    bool
	}{
		{name: "requisito funcional", line: "### RF-100: Alta de cliente", wantID: "RF-100", wantTitle: "Alta de cliente", wantOK: true},
		{name: "requisito no funcional", line: "### NFR-002: Capacidad", wantID: "NFR-002", wantTitle: "Capacidad", wantOK: true},
		{name: "el título conserva los dos puntos", line: "### RF-1: Cobro: tarjeta", wantID: "RF-1", wantTitle: "Cobro: tarjeta", wantOK: true},
		{name: "sección que no es requisito", line: "### Notas de implantación"},
		{name: "ID sin título", line: "### RF-100"},
		{name: "ID de otra familia", line: "### US-014: Cliente"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, title, ok := profile.Match(tt.line)
			if ok != tt.wantOK || id != tt.wantID || title != tt.wantTitle {
				t.Errorf("Match(%q) = (%q, %q, %v), se esperaba (%q, %q, %v)", tt.line, id, title, ok, tt.wantID, tt.wantTitle, tt.wantOK)
			}
		})
	}

	if _, _, ok := (Profile{}).Match("### RF-1: A"); ok {
		t.Error("un perfil sin patrón no debe reconocer ningún requisito")
	}
}

func TestProfileIdentityChoosesTheKey(t *testing.T) {
	byID := profileFor(t, `^### (?P<id>RF-\d+): (?P<title>.+)$`, 3, workspace.IdentityID, "### {id}: {title}", workspace.ScenariosInclude)
	byText := profileFor(t, `^### (?P<id>RF-\d+): (?P<title>.+)$`, 3, workspace.IdentityText, "### {id}: {title}", workspace.ScenariosInclude)
	if got := byID.Key("RF-1", "Alta"); got != "RF-1" {
		t.Errorf("Key con identidad id = %q, se esperaba RF-1", got)
	}
	if got := byText.Key("RF-1", "Alta"); got != "Alta" {
		t.Errorf("Key con identidad text = %q, se esperaba Alta", got)
	}
}

func TestProfileRenderSubstitutesIDAndTitleOnce(t *testing.T) {
	tests := []struct {
		name   string
		render string
		id     string
		title  string
		want   string
	}{
		{name: "plantilla habitual", render: "### {id}: {title}", id: "RF-120", title: "Alta de cliente con validación", want: "### RF-120: Alta de cliente con validación"},
		{name: "orden invertido", render: "### {title} ({id})", id: "RF-120", title: "Alta", want: "### Alta (RF-120)"},
		{name: "marcador repetido", render: "### {id} {id} {title}", id: "RF-1", title: "A", want: "### RF-1 RF-1 A"},
		{name: "el título no se vuelve a expandir", render: "### {id}: {title}", id: "RF-1", title: "usa {id} literal", want: "### RF-1: usa {id} literal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Profile{Render: tt.render}
			if got := profile.RenderHeading(tt.id, tt.title); got != tt.want {
				t.Errorf("RenderHeading() = %q, se esperaba %q", got, tt.want)
			}
		})
	}
}

func TestProfileHeadingLineRebuildsTheLevelPrefix(t *testing.T) {
	for level, want := range map[int]string{1: "# RF-1: A", 3: "### RF-1: A", 6: "###### RF-1: A"} {
		if got := (Profile{Level: level}).HeadingLine("RF-1: A"); got != want {
			t.Errorf("HeadingLine con level %d = %q, se esperaba %q", level, got, want)
		}
	}
}
