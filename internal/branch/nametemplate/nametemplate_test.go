package nametemplate

import (
	"errors"
	"testing"
)

func mustParse(t *testing.T, raw string) Template {
	t.Helper()
	tpl, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%q): %v", raw, err)
	}
	return tpl
}

func TestVariablesOrder(t *testing.T) {
	got := Variables()
	want := []Variable{VarPrefix, VarKind, VarChange, VarSlug, VarRole, VarRef, VarTaskRef}
	if len(got) != len(want) {
		t.Fatalf("Variables() = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Variables()[%d] = %q, quiero %q", i, got[i], want[i])
		}
	}
}

func TestParseSyntaxErrors(t *testing.T) {
	for _, raw := range []string{"{prefix", "prefix}", "{prefix/{change}}", "{{change}}", "{}", ""} {
		_, err := Parse(raw)
		var se *SyntaxError
		if !errors.As(err, &se) {
			t.Errorf("Parse(%q) = %v, quiero *SyntaxError", raw, err)
		}
	}
}

func TestParseUnknownVariable(t *testing.T) {
	_, err := Parse("{prefix}/{branch}")
	var ue *UnknownVariableError
	if !errors.As(err, &ue) || ue.Name != "branch" {
		t.Fatalf("Parse = %v, quiero *UnknownVariableError{branch}", err)
	}
}

func TestParseUsesAndRaw(t *testing.T) {
	tpl := mustParse(t, "feature/TASK-{task_ref}")
	if tpl.IsZero() || tpl.Raw() != "feature/TASK-{task_ref}" {
		t.Fatalf("Raw/IsZero incorrectos: %q", tpl.Raw())
	}
	if !tpl.Uses(VarTaskRef) || tpl.Uses(VarRef) {
		t.Fatal("Uses incorrecto")
	}
	if !(Template{}).IsZero() {
		t.Fatal("el valor cero debe ser IsZero")
	}
}

func TestRenderScenarios(t *testing.T) {
	cases := []struct {
		name, tpl string
		values    Values
		want      string
	}{
		{"defecto", "{prefix}/{change}", Values{VarPrefix: "inc", VarChange: "login"}, "inc/login"},
		{"kind y ref", "{kind}/{ref}-{slug}", Values{VarKind: "feature", VarRef: "2525", VarSlug: "login"}, "feature/2525-login"},
		{"fix", "{kind}/{change}", Values{VarKind: "fix", VarChange: "crash"}, "fix/crash"},
		{"task_ref", "feature/TASK-{task_ref}", Values{VarTaskRef: "1235"}, "feature/TASK-1235"},
		{"rol", "{prefix}/{change}/{role}", Values{VarPrefix: "equipo/inc", VarChange: "c", VarRole: "backend"}, "equipo/inc/c/backend"},
	}
	for _, tc := range cases {
		got, err := mustParse(t, tc.tpl).Render(tc.values)
		if err != nil || got != tc.want {
			t.Errorf("%s: Render = %q, %v; quiero %q", tc.name, got, err, tc.want)
		}
	}
}

func TestRenderRejectsUnsafeValuesWithoutSanitising(t *testing.T) {
	for _, value := range []string{"-D", "--force", "T#12", "a b", "..", "x.lock", "CON"} {
		_, err := mustParse(t, "feature/{ref}").Render(Values{VarRef: value})
		var ue *UnsafeValueError
		if !errors.As(err, &ue) {
			t.Errorf("valor %q: Render = %v, quiero *UnsafeValueError", value, err)
			continue
		}
		if ue.Variable != VarRef || ue.Value != value {
			t.Errorf("valor %q: error con variable %q y valor %q", value, ue.Variable, ue.Value)
		}
	}
}

func TestRenderValueRulesPerVariable(t *testing.T) {
	// {prefix} admite «/», las demás no; {kind} solo feature o fix.
	if _, err := mustParse(t, "{prefix}/{change}").Render(Values{VarPrefix: "a/b", VarChange: "c"}); err != nil {
		t.Errorf("prefijo con «/»: %v", err)
	}
	if _, err := mustParse(t, "{change}").Render(Values{VarChange: "a/b"}); err == nil {
		t.Error("{change} con «/» debe rechazarse")
	}
	_, err := mustParse(t, "{kind}/{change}").Render(Values{VarKind: "bug", VarChange: "c"})
	var ue *UnsafeValueError
	if !errors.As(err, &ue) || ue.Variable != VarKind {
		t.Errorf("kind=bug: %v", err)
	}
	_, err = mustParse(t, "{prefix}/{change}").Render(Values{VarPrefix: "/x", VarChange: "c"})
	if !errors.As(err, &ue) || ue.Variable != VarPrefix {
		t.Errorf("prefijo con «/» inicial: %v", err)
	}
}

func TestRenderMissingValue(t *testing.T) {
	for _, values := range []Values{{}, {VarRef: ""}} {
		_, err := mustParse(t, "feature/{ref}").Render(values)
		var me *MissingValueError
		if !errors.As(err, &me) || me.Variable != VarRef {
			t.Errorf("Render(%v) = %v, quiero *MissingValueError{ref}", values, err)
		}
	}
}

func TestRenderInvalidName(t *testing.T) {
	_, err := mustParse(t, "-{ref}").Render(Values{VarRef: "1"})
	var ie *InvalidNameError
	if !errors.As(err, &ie) || ie.Name != "-1" {
		t.Errorf("-{ref}: %v", err)
	}
}

func TestCheckSampleRejectsBadShapes(t *testing.T) {
	for _, raw := range []string{"-{ref}", "{prefix}/", "/{change}", "a..b/{change}"} {
		if err := CheckSample(mustParse(t, raw)); err == nil {
			t.Errorf("CheckSample(%q) debe fallar", raw)
		}
	}
	for _, raw := range []string{"{prefix}/{change}", "feature/TASK-{task_ref}", "{kind}/{ref}-{slug}/{role}"} {
		if err := CheckSample(mustParse(t, raw)); err != nil {
			t.Errorf("CheckSample(%q): %v", raw, err)
		}
	}
}

func TestCheckDistinguishing(t *testing.T) {
	for _, raw := range []string{"{prefix}/{kind}", "release", "{role}"} {
		err := CheckDistinguishing(mustParse(t, raw))
		var ce *ConstantTemplateError
		if !errors.As(err, &ce) {
			t.Errorf("CheckDistinguishing(%q) = %v, quiero *ConstantTemplateError", raw, err)
		}
	}
	for _, raw := range []string{"{change}", "x/{slug}", "{prefix}/{ref}", "t-{task_ref}"} {
		if err := CheckDistinguishing(mustParse(t, raw)); err != nil {
			t.Errorf("CheckDistinguishing(%q): %v", raw, err)
		}
	}
}

func TestSlug(t *testing.T) {
	cases := []struct{ change, ref, want string }{
		{"us-2525-login", "2525", "login"},
		{"US-2525-login", "2525", "login"},
		{"us-2525", "2525", "us-2525"},
		{"us-2525-", "2525", "us-2525-"},
		{"us-2525-login", "", "us-2525-login"},
		{"login", "2525", "login"},
		{"us-9999-login", "2525", "us-9999-login"},
		{"2525-login", "2525", "2525-login"},
		{"AB-x1-login", "X1", "login"},
	}
	for _, tc := range cases {
		if got := Slug(tc.change, tc.ref); got != tc.want {
			t.Errorf("Slug(%q, %q) = %q, quiero %q", tc.change, tc.ref, got, tc.want)
		}
	}
}

func TestErrorMessagesNameTheOffender(t *testing.T) {
	_, err := Parse("{x}")
	if msg := err.Error(); !contains(msg, "{x}") || !contains(msg, "{task_ref}") {
		t.Errorf("mensaje sin variable ni vocabulario: %s", msg)
	}
	_, err = mustParse(t, "{ref}").Render(Values{VarRef: "a b"})
	if msg := err.Error(); !contains(msg, "{ref}") || !contains(msg, "a b") {
		t.Errorf("mensaje sin variable ni valor: %s", msg)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
