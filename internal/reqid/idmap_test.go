package reqid_test

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestIDMapSubstitute(t *testing.T) {
	one := []reqid.Assignment{{Provisional: "prov:30:1", Definitive: "REQ-30.7"}}
	cases := []struct {
		name       string
		pairs      []reqid.Assignment
		text       string
		want       string
		idempotent bool
	}{
		{"token completo", one, "Cobro (prov:30:1)", "Cobro (REQ-30.7)", true},
		{"no altera un prefijo de otro token", one, "prov:30:11 y prov:30:1", "prov:30:11 y REQ-30.7", true},
		{"el más largo primero", []reqid.Assignment{
			{Provisional: "prov:30:1", Definitive: "REQ-30.7"},
			{Provisional: "prov:30:11", Definitive: "REQ-30.9"},
		}, "prov:30:11 prov:30:1", "REQ-30.9 REQ-30.7", true},
		{"pegado a letra", one, "xprov:30:1", "xprov:30:1", true},
		{"pegado a cifra", one, "9prov:30:1", "9prov:30:1", true},
		{"pegado a guion bajo", one, "prov:30:1_x _prov:30:1", "prov:30:1_x _prov:30:1", true},
		{"pegado a guion", one, "prov:30:1-x", "prov:30:1-x", true},
		{"pegado a puntuación sí se sustituye", one, "`prov:30:1`, prov:30:1.", "`REQ-30.7`, REQ-30.7.", true},
		{"sin provisionales no cambia", one, "Cobro (REQ-30.7)", "Cobro (REQ-30.7)", true},
		{"reasignación definitiva a definitiva", []reqid.Assignment{{Provisional: "REQ-30.5", Definitive: "REQ-30.8"}},
			"(REQ-30.5) y REQ-30.55 y REQ-30.5.", "(REQ-30.8) y REQ-30.55 y REQ-30.8.", false},
		{"una sola pasada: un intercambio no se deshace", []reqid.Assignment{
			{Provisional: "REQ-30.5", Definitive: "REQ-30.6"},
			{Provisional: "REQ-30.6", Definitive: "REQ-30.5"},
		}, "REQ-30.5 REQ-30.6", "REQ-30.6 REQ-30.5", false},
		{"mapa vacío", nil, "prov:30:1", "prov:30:1", true},
		{"texto no ASCII alrededor", one, "ñprov:30:1 y «prov:30:1»", "ñprov:30:1 y «REQ-30.7»", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := reqid.NewIDMap(tc.pairs)
			got := m.Substitute(tc.text)
			if got != tc.want {
				t.Errorf("Substitute = %q, quería %q", got, tc.want)
			}
			if tc.idempotent {
				if again := m.Substitute(got); again != got {
					t.Errorf("no idempotente: %q", again)
				}
			}
		})
	}
}

func TestIDMapCount(t *testing.T) {
	if n := reqid.NewIDMap(nil).Count(); n != 0 {
		t.Errorf("Count vacío = %d", n)
	}
	m := reqid.NewIDMap([]reqid.Assignment{
		{Provisional: "prov:30:1", Definitive: "REQ-30.7"},
		{Provisional: "prov:30:1", Definitive: "REQ-30.9"},
		{Provisional: "prov:30:2", Definitive: "REQ-30.8"},
	})
	if n := m.Count(); n != 2 {
		t.Errorf("Count = %d, quería 2 (la clave repetida vale una vez)", n)
	}
	if got := m.Substitute("prov:30:1"); got != "REQ-30.7" {
		t.Errorf("con clave repetida vale la primera: %q", got)
	}
}
