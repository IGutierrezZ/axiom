package reqtrace

import (
	"errors"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestRequirementTraceParseID(t *testing.T) {
	custom, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatalf("plantilla personalizada: %v", err)
	}
	def := reqid.Default()
	for _, tc := range []struct {
		name    string
		raw     string
		pattern reqid.Pattern
		want    Query
		wantErr bool
	}{
		{"válido", "REQ-1.1", def, Query{ID: "REQ-1.1"}, false},
		{"sin normalizar ceros", "REQ-01.1", def, Query{ID: "REQ-01.1"}, false},
		{"provisional", "prov:trace:1", def, Query{ID: "prov:trace:1", Provisional: true}, false},
		{"provisional mal formado", "prov:trace:0", def, Query{}, true},
		{"inválido", "RF-USU-01", def, Query{}, true},
		{"patrón del proyecto manda", "RF-USU-01", custom, Query{ID: "RF-USU-01"}, false},
		{"el patrón por defecto no vale en el personalizado", "REQ-1.1", custom, Query{}, true},
		{"espacios", " REQ-1.1", def, Query{}, true},
		{"minúsculas", "req-1.1", def, Query{}, true},
		{"vacío", "", def, Query{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseID(tc.raw, tc.pattern, nil)
			if tc.wantErr {
				var idErr *IDError
				if !errors.As(err, &idErr) {
					t.Fatalf("quería *IDError, obtuve %v", err)
				}
				if idErr.Code != CodeIDInvalid || idErr.ID != tc.raw || idErr.Template != tc.pattern.Template() || idErr.Cause == nil {
					t.Fatalf("IDError incompleto: %+v", idErr)
				}
				if !errors.Is(err, idErr.Cause) {
					t.Fatalf("Unwrap no devuelve la causa")
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("ParseID(%q) = %+v, %v; quería %+v", tc.raw, got, err, tc.want)
			}
		})
	}
}

func TestRequirementTraceResultNormalizeEmitsEmptyLists(t *testing.T) {
	r := Result{Increments: []Increment{{Lane: LaneSDD}}}
	r.normalize()
	if r.Schema != Schema || r.ResolvedIDs == nil || r.Warnings == nil || r.Increments[0].IDs == nil || r.Increments[0].Tasks == nil {
		t.Fatalf("normalize dejó listas nil: %+v", r)
	}
}
