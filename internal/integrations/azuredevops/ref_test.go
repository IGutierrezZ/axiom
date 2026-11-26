package azuredevops

import (
	"errors"
	"strings"
	"testing"
)

func TestParseItemRef(t *testing.T) {
	cfg := map[string]any{"organization": "Acme", "project": "Proyecto", "team": "Equipo"}
	p := New(Options{})

	cases := []struct {
		name string
		raw  string
		want string // vacío: se espera error
	}{
		{"id", "2525", "2525"},
		{"id con espacios", "  2525 ", "2525"},
		{"id máximo int32", "2147483647", "2147483647"},
		{"id cero", "0", ""},
		{"id cero inicial", "0123", ""},
		{"id mayor que int32", "2147483648", ""},
		{"id con signo", "+5", ""},
		{"id decimal", "5.0", ""},
		{"id vacío", "", ""},
		{"texto", "abc", ""},
		{"edit con proyecto", "https://dev.azure.com/acme/Proyecto/_workitems/edit/2525", "2525"},
		{"edit sin proyecto", "https://dev.azure.com/Acme/_workitems/edit/2525", "2525"},
		{"edit con barra final", "https://dev.azure.com/acme/Proyecto/_workitems/edit/2525/", "2525"},
		{"edit con fragmento", "https://dev.azure.com/acme/Proyecto/_workitems/edit/2525#comments", "2525"},
		{"edit proyecto en minúsculas", "https://dev.azure.com/acme/proyecto/_workitems/edit/7", "7"},
		{"api con proyecto", "https://dev.azure.com/acme/Proyecto/_apis/wit/workitems/2525", "2525"},
		{"api sin proyecto", "https://dev.azure.com/acme/_apis/wit/workitems/2525", "2525"},
		{"tablero", "https://dev.azure.com/acme/Proyecto/_boards/board/t/Equipo/Stories?workitem=2525", "2525"},
		{"backlog", "https://dev.azure.com/acme/Proyecto/_backlogs/backlog/Equipo/Stories/?workitem=9&x=y", "9"},
		{"tablero con dos workitem", "https://dev.azure.com/acme/Proyecto/_boards/board?workitem=1&workitem=2", ""},
		{"tablero con id inválido", "https://dev.azure.com/acme/Proyecto/_boards/board?workitem=0", ""},
		{"otra organización", "https://dev.azure.com/otra/Proyecto/_workitems/edit/9", ""},
		{"otro proyecto", "https://dev.azure.com/acme/Otro/_workitems/edit/9", ""},
		{"otro host", "https://evil.example/acme/Proyecto/_workitems/edit/9", ""},
		{"visualstudio.com", "https://acme.visualstudio.com/Proyecto/_workitems/edit/9", ""},
		{"http", "http://dev.azure.com/acme/Proyecto/_workitems/edit/9", ""},
		{"userinfo", "https://user:pw" + "@dev.azure.com/acme/Proyecto/_workitems/edit/9", ""},
		{"sin esquema", "dev.azure.com/acme/Proyecto/_workitems/edit/9", ""},
		{"ruta sin id", "https://dev.azure.com/acme/Proyecto/_workitems/edit/", ""},
		{"ruta ajena sin workitem", "https://dev.azure.com/acme/Proyecto/_git/repo", ""},
		{"id no numérico en la ruta", "https://dev.azure.com/acme/Proyecto/_workitems/edit/abc", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := p.ParseItemRef(cfg, tc.raw)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("ParseItemRef(%q) = %q, esperaba un error", tc.raw, got)
				}
				if !errors.Is(err, ErrItemRef) {
					t.Errorf("el error debe responder a ErrItemRef: %v", err)
				}
				if tc.raw != "" && strings.Contains(err.Error(), tc.raw) {
					t.Errorf("el error no debe repetir la referencia recibida: %v", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("ParseItemRef(%q) = %q, %v; esperaba %q", tc.raw, got, err, tc.want)
			}
		})
	}
}

func TestParseItemRefHonoursTheEndpointScheme(t *testing.T) {
	cfg := map[string]any{"organization": "acme", "project": "P", "team": "T"}
	p := New(Options{Endpoint: "http://127.0.0.1:8080"})
	if got, err := p.ParseItemRef(cfg, "http://127.0.0.1:8080/acme/P/_workitems/edit/5"); err != nil || got != "5" {
		t.Fatalf("la URL del endpoint de pruebas debe aceptarse: %q, %v", got, err)
	}
	if _, err := p.ParseItemRef(cfg, "https://127.0.0.1:8080/acme/P/_workitems/edit/5"); err == nil {
		t.Fatal("un esquema distinto del del endpoint debe rechazarse")
	}
	if _, err := p.ParseItemRef(map[string]any{"organization": "acme"}, "5"); err == nil {
		t.Fatal("una configuración incompleta debe rechazarse")
	}
}
