package marks

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
)

func testBattery() acceptancetests.Battery {
	return acceptancetests.Battery{Cases: []acceptancetests.Case{
		{ID: "P-01", Environments: []string{"pre", "local"}},
		{ID: "P-02", Environments: []string{"pre", "prod"}}, // prod no declarado
		{ID: "P-03", Environments: []string{"pre"}},
		{ID: "P-03", Environments: []string{"local"}}, // ID repetido: unión
	}}
}

var testEnvs = []string{"pre", "local"}

func TestValidateInputRejections(t *testing.T) {
	ok := Input{Case: "P-01", Environment: "pre", Result: "ok"}
	tests := []struct {
		name string
		mut  func(*Input)
		code string
	}{
		{"caso desconocido", func(i *Input) { i.Case = "P-99" }, CodeCaseUnknown},
		{"entorno no declarado", func(i *Input) { i.Case = "P-02"; i.Environment = "prod" }, CodeEnvironmentInvalid},
		{"entorno no aplicable", func(i *Input) { i.Case = "P-02"; i.Environment = "local" }, CodeEnvironmentInvalid},
		{"resultado", func(i *Input) { i.Result = "OK" }, CodeResultInvalid},
		{"nota con LF", func(i *Input) { i.Note = "a\nb" }, CodeNoteInvalid},
		{"nota con TAB", func(i *Input) { i.Note = "a\tb" }, CodeNoteInvalid},
		{"nota UTF-8 inválido", func(i *Input) { i.Note = "a\xffb" }, CodeNoteInvalid},
		{"nota larga", func(i *Input) { i.Note = strings.Repeat("ñ", 501) }, CodeNoteTooLong},
		{"commit con espacio", func(i *Input) { i.Commit = "a b" }, CodeCommitInvalid},
		{"commit con comilla", func(i *Input) { i.Commit = "a'b" }, CodeCommitInvalid},
		{"commit largo", func(i *Input) { i.Commit = strings.Repeat("a", 101) }, CodeCommitInvalid},
	}
	for _, tc := range tests {
		in := ok
		tc.mut(&in)
		_, rej := ValidateInput(in, testBattery(), testEnvs)
		if rej == nil || rej.Code != tc.code || rej.Message == "" || rej.Continuation == "" {
			t.Errorf("%s: rechazo = %+v, quería %s", tc.name, rej, tc.code)
		}
	}
}

func TestValidateInputAccepts(t *testing.T) {
	e, rej := ValidateInput(Input{Case: "P-03", Environment: "local", Result: "n/a",
		Commit: " v1.2.3 ", Note: "  " + strings.Repeat("ñ", 500) + "  "}, testBattery(), testEnvs)
	if rej != nil {
		t.Fatalf("rechazo: %+v", rej)
	}
	if e.Commit != "v1.2.3" || e.Result != ResultNA || len([]rune(e.Note)) != 500 || e.By != "" || e.At != "" || e.PlanDigest != "" {
		t.Fatalf("entrada = %+v", e)
	}
	if _, rej := ValidateInput(Input{Case: "P-01", Environment: "pre", Result: "ok", Commit: "build#123@x/y+z:1_"}, testBattery(), testEnvs); rej != nil {
		t.Fatalf("commit permitido rechazado: %+v", rej)
	}
}
