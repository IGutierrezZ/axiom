package adoptcheck

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// withDeclaredPattern declara `sdd.id_pattern` en la ubicación del fixture y lo pasa a Check.
func withDeclaredPattern(t *testing.T, f *checkFixture, template string) {
	t.Helper()
	p, err := reqid.Parse(template)
	if err != nil {
		t.Fatal(err)
	}
	f.loc.Declarations = workspace.Declarations{SDD: &workspace.SDDConfig{IDPattern: &p}}
	f.pattern = p
}

// withoutDeclaredPattern deja el proyecto sin `sdd.id_pattern`: se mide con el patrón por defecto.
func withoutDeclaredPattern(f *checkFixture) {
	f.loc.Declarations = workspace.Declarations{}
	f.pattern = reqid.Default()
}

// REQ-30.11 y REQ-44.4: con `sdd.id_pattern` declarado, `adopt check` mide con él.
func TestCheck_ConUnPatronDeclaradoMideConEl(t *testing.T) {
	f := newCheckFixture(t)
	report := f.check(t)
	if !report.Passed || len(report.Failures) != 0 {
		t.Fatalf("un (RF-001) con RF-<NNN> debe pasar el formato: %+v", report)
	}
	if report.IDPattern != "RF-<NNN>" || !report.IDPatternDeclared {
		t.Fatalf("el informe debe decir el patrón efectivo y que está declarado: %+v", report)
	}
}

// El patrón declarado no cae al defecto: una delta con `REQ-n.m` ya no es válida.
func TestCheck_ElPatronDeclaradoNoCaeAlDefecto(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.ReplaceAll(strings.ReplaceAll(validDelta, "(RF-001)", "(REQ-40.1)"), "(RF-002)", "(REQ-40.2)"))
	report := f.check(t)
	requireFailure(t, report, RuleFormat, "Alta de usuario", "(RF-NNN)")
	requireFailure(t, report, RuleFormat, "Baja de usuario", "(RF-NNN)")
}

// Sin declaración se mide con `REQ-<n>.<m>` y el informe lo dice.
func TestCheck_SinDeclaracionMideConElDefecto(t *testing.T) {
	f := newCheckFixture(t)
	withoutDeclaredPattern(f)
	f.write(t, "specs/proyecto/spec.md", strings.ReplaceAll(strings.ReplaceAll(validDelta, "(RF-001)", "(REQ-40.1)"), "(RF-002)", "(REQ-40.2)"))
	report := f.check(t)
	if got := failuresOf(report, RuleFormat); len(got) != 0 {
		t.Fatalf("REQ-n.m es válido con el patrón por defecto: %+v", got)
	}
	if report.IDPattern != reqid.Default().Template() || report.IDPatternDeclared {
		t.Fatalf("id_pattern_declared debe ser false con el defecto: %+v", report)
	}
}
