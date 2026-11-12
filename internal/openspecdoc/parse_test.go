package openspecdoc

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseDeltaReadsTheFourSectionsInOrder(t *testing.T) {
	delta := `## ADDED Requirements

### Requirement: Alta (REQ-31.1)

El sistema DEBE dar de alta.

#### Scenario: Alta

- **DADO** un formulario

### Requirement: Baja (REQ-31.2)

El sistema DEBE dar de baja.

## MODIFIED Requirements

### Requirement: Consulta (REQ-31.3)

El sistema DEBE consultar.

## REMOVED Requirements

### Requirement: Vieja

(Reason: obsoleta)

## RENAMED Requirements

### Requirement: Antigua → Nueva

(Reason: nombre mejor)
`
	got := ParseDelta(delta)
	if names := requirementNames(got.Added); !reflect.DeepEqual(names, []string{"Alta (REQ-31.1)", "Baja (REQ-31.2)"}) {
		t.Errorf("Added = %v", names)
	}
	if names := requirementNames(got.Modified); !reflect.DeepEqual(names, []string{"Consulta (REQ-31.3)"}) {
		t.Errorf("Modified = %v", names)
	}
	if names := requirementNames(got.Removed); !reflect.DeepEqual(names, []string{"Vieja"}) {
		t.Errorf("Removed = %v", names)
	}
	if names := requirementNames(got.Renamed); !reflect.DeepEqual(names, []string{"Antigua → Nueva"}) {
		t.Errorf("Renamed = %v", names)
	}
	if got.Added[0].ID != "REQ-31.1" || got.Added[1].ID != "REQ-31.2" || got.Modified[0].ID != "REQ-31.3" {
		t.Errorf("IDs = %q %q %q", got.Added[0].ID, got.Added[1].ID, got.Modified[0].ID)
	}
	if got.Removed[0].ID != "" {
		t.Errorf("un título sin (REQ-n.m) no lleva ID: %q", got.Removed[0].ID)
	}
	if got.Added[0].Line != 3 || got.Added[1].Line != 11 {
		t.Errorf("Line = %d, %d; se esperaba 3 y 11", got.Added[0].Line, got.Added[1].Line)
	}
	if !strings.HasPrefix(got.Added[0].Body, "### Requirement: Alta (REQ-31.1)") || !strings.Contains(got.Added[0].Body, "#### Scenario: Alta") {
		t.Errorf("Body debe ser el bloque completo con su título: %q", got.Added[0].Body)
	}
	if strings.Contains(got.Added[0].Body, "Baja") {
		t.Errorf("Body no debe incluir el requisito siguiente: %q", got.Added[0].Body)
	}
}

func TestParseDeltaReportsUnknownSections(t *testing.T) {
	got := ParseDelta("## ADDED Requirements\n\n### Requirement: A\n\ncuerpo\n\n## Notas\n\nlibre\n\n```\n## En una valla\n```\n")
	if !reflect.DeepEqual(got.Unknown, []string{"## Notas"}) {
		t.Errorf("Unknown = %v; solo un encabezado de nivel 2 fuera de vallas y que no es una operación", got.Unknown)
	}
}

func TestParseDeltaWithoutSectionsIsEmpty(t *testing.T) {
	got := ParseDelta("Sin secciones.\n")
	if len(got.Added)+len(got.Modified)+len(got.Removed)+len(got.Renamed) != 0 {
		t.Errorf("Delta = %+v, se esperaba vacía", got)
	}
}

func TestParseCanonicalListsEveryRequirementWithItsLine(t *testing.T) {
	got := ParseCanonical(composeCanonicalFixture)
	if names := requirementNames(got); !reflect.DeepEqual(names, []string{"Unrelated Listing", "Widget Expiration"}) {
		t.Fatalf("requisitos = %v", names)
	}
	if got[0].Line != 5 || got[1].Line != 15 {
		t.Errorf("Line = %d, %d; se esperaba 5 y 15", got[0].Line, got[1].Line)
	}
	if !strings.Contains(got[1].Body, "THEN the widget is removed") {
		t.Errorf("Body debe llegar hasta el final del bloque: %q", got[1].Body)
	}
}

func TestParseCanonicalIgnoresHeadingsInsideFences(t *testing.T) {
	canonical := "# T\n\n## Requirements\n\n### Requirement: Uno (REQ-1.1)\n\n```\n### Requirement: Falso (REQ-1.2)\n```\n"
	got := ParseCanonical(canonical)
	if len(got) != 1 || got[0].ID != "REQ-1.1" {
		t.Errorf("requisitos = %+v, se esperaba solo REQ-1.1", got)
	}
}

func TestNormalizeBody(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"lf y espacios finales", "\n\nlinea   \r\notra\t\r\n\n\n", "linea\notra"},
		{"se conservan las líneas en blanco internas", "a\n\nb\n", "a\n\nb"},
		{"vacío", "  \n\n", ""},
		{"no toca la sangría", "  - a\n    b\n", "  - a\n    b"},
	}
	for _, tt := range tests {
		if got := NormalizeBody(tt.in); got != tt.want {
			t.Errorf("%s: NormalizeBody(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

func requirementNames(reqs []Requirement) []string {
	names := make([]string, 0, len(reqs))
	for _, r := range reqs {
		names = append(names, r.Name)
	}
	return names
}
