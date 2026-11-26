package acceptancetests

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const header = "| ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones | Pasos | Resultado esperado |\n" +
	"|---|---|---|---|---|---|---|---|---|\n"

// row construye una fila válida; los campos se pueden sobrescribir con los pares clave/valor.
func row(over ...string) string {
	f := map[string]string{
		"ID": "P-01", "Cubre": "REQ-1.1#Uno", "Rol": "fullstack", "Tipo": "manual", "Prioridad": "alta",
		"Entornos": "dev", "Precondiciones": "", "Pasos": "hacer", "Resultado esperado": "ok",
	}
	for i := 0; i+1 < len(over); i += 2 {
		f[over[i]] = over[i+1]
	}
	var cells []string
	for _, c := range columns {
		cells = append(cells, f[c])
	}
	return "| " + strings.Join(cells, " | ") + " |\n"
}

func req(domain string, section isoevidence.Section, name string, scenarios ...string) isoevidence.Requirement {
	var b strings.Builder
	b.WriteString("### Requirement: " + name + "\n\nTexto.\n\n")
	for _, s := range scenarios {
		b.WriteString("#### Scenario: " + s + "\n- **DADO** x\n\n")
	}
	return isoevidence.Requirement{Domain: domain, Section: section, Name: name, Text: b.String()}
}

func baseInput(delta ...isoevidence.Requirement) Input {
	return Input{
		Delta:        delta,
		Pattern:      reqid.Default(),
		Environments: []string{"dev", "pre"},
		Roles:        []string{"fullstack"},
	}
}

func codes(r Report) []Code {
	var out []Code
	for _, f := range r.Findings {
		out = append(out, f.Code)
	}
	return out
}

func has(r Report, c Code) bool {
	for _, f := range r.Findings {
		if f.Code == c {
			return true
		}
	}
	return false
}

func count(r Report, c Code) int {
	n := 0
	for _, f := range r.Findings {
		if f.Code == c {
			n++
		}
	}
	return n
}

func mustOnly(t *testing.T, r Report, want ...Code) {
	t.Helper()
	got := codes(r)
	if len(got) != len(want) {
		t.Fatalf("hallazgos = %v, se esperaban %v: %+v", got, want, r.Findings)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hallazgos = %v, se esperaban %v", got, want)
		}
	}
}
