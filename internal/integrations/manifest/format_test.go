package manifest

import (
	"reflect"
	"testing"
)

func TestParseTypePrefixes(t *testing.T) {
	cases := []struct {
		name     string
		in       []string
		want     []TypePrefix
		problems int
	}{
		{"valido", []string{"User Story=US", "Bug=BUG"}, []TypePrefix{{"User Story", "US"}, {"Bug", "BUG"}}, 0},
		{"recorta", []string{"  Task = T1 "}, []TypePrefix{{"Task", "T1"}}, 0},
		{"sin igual", []string{"Bug"}, nil, 1},
		{"dos iguales", []string{"Bug=B=G"}, nil, 1},
		{"tipo vacio", []string{" =US"}, nil, 1},
		{"prefijo vacio", []string{"Bug="}, nil, 1},
		{"prefijo con guion", []string{"Bug=B-G"}, nil, 1},
		{"prefijo largo", []string{"Bug=ABCDEFGHIJK"}, nil, 1},
		{"tipo repetido", []string{"Bug=B1", "bug=B2"}, []TypePrefix{{"Bug", "B1"}}, 1},
		{"prefijo repetido sin mayusculas", []string{"Bug=BUG", "Task=bug"}, []TypePrefix{{"Bug", "BUG"}}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, problems := ParseTypePrefixes(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("pares = %v, esperaba %v", got, tc.want)
			}
			if len(problems) != tc.problems {
				t.Errorf("problemas = %v, esperaba %d", problems, tc.problems)
			}
			for _, p := range problems {
				if p.Element == "" && tc.in[p.Index] != "" {
					t.Errorf("el problema %+v no conserva el elemento", p)
				}
				if p.Index < 0 || p.Index >= len(tc.in) || p.Element != tc.in[p.Index] || p.Reason == "" {
					t.Errorf("problema incoherente: %+v", p)
				}
			}
		})
	}
}
