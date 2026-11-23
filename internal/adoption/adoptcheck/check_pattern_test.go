package adoptcheck

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// withDeclaredPattern declara `sdd.id_pattern` en la ubicación del fixture.
func withDeclaredPattern(t *testing.T, f *checkFixture, template string) {
	t.Helper()
	p, err := reqid.Parse(template)
	if err != nil {
		t.Fatal(err)
	}
	f.loc.Declarations = workspace.Declarations{SDD: &workspace.SDDConfig{IDPattern: &p}}
}

// REQ-30.31 y REQ-44.4: `adopt check` no lee `sdd.id_pattern`: con un proyecto que declara
// RF-<DOM>-<NN> sigue aplicando `REQ-<n>.<m>` hasta C-03-c.
func TestCheck_ConUnPatronDeclaradoSigueAplicandoElDefecto(t *testing.T) {
	f := newCheckFixture(t)
	withDeclaredPattern(t, f, "RF-<DOM>-<NN>")

	if report := f.check(t); !report.Passed || len(report.Failures) != 0 {
		t.Fatalf("la delta con REQ-<n>.<m> debe pasar aunque el proyecto declare otro patrón: %+v", report)
	}

	f.write(t, "specs/proyecto/spec.md", strings.ReplaceAll(strings.ReplaceAll(validDelta, "REQ-40.1", "RF-USU-01"), "REQ-40.2", "RF-USU-02"))
	report := f.check(t)
	requireFailure(t, report, RuleFormat, "Alta de usuario", "(REQ-n.m)")
	requireFailure(t, report, RuleFormat, "Baja de usuario", "(REQ-n.m)")
}

// Los textos de las reglas de formato y de serie son los de siempre.
func TestCheck_LosTextosDelDefectoNoCambian(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "REQ-40.2", "REQ-41.1", 1))
	report := f.check(t)
	requireFailure(t, report, RuleSeries, "series", "40 y 41", "`REQ-n.*`")
}
