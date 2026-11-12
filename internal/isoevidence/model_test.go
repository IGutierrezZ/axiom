package isoevidence

import "testing"

func TestFindingReason(t *testing.T) {
	cases := []struct {
		name string
		in   Finding
		want string
	}{
		{
			name: "forma completa de design 5.3",
			in: Finding{
				Code:         CodeRequirementID,
				Detail:       `dominio "pagos", sección ADDED, requisito "Cobro": sin identificador`,
				Continuation: "añade un identificador REQ-<n>.<m>",
			},
			want: `blocked(iso_requirement_id): dominio "pagos", sección ADDED, requisito "Cobro": sin identificador; añade un identificador REQ-<n>.<m>`,
		},
		{
			name: "sin continuación no deja punto y coma",
			in:   Finding{Code: CodeMatrixMissing, Detail: "falta la matriz"},
			want: "blocked(iso_matrix_missing): falta la matriz",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Reason(); got != tc.want {
				t.Fatalf("Reason() = %q, quería %q", got, tc.want)
			}
		})
	}
}

func TestAllCodesClosedList(t *testing.T) {
	codes := AllCodes()
	seen := map[Code]bool{}
	for _, c := range codes {
		if c == "" || seen[c] {
			t.Fatalf("token vacío o repetido: %q", c)
		}
		seen[c] = true
	}
	if len(codes) != 13 {
		t.Fatalf("la lista cerrada de design 5.3 tiene 13 tokens y hay %d", len(codes))
	}
}
