package kickoff

import "testing"

// TestValidateGateRecordArgs fija que la validación tipada compartida rechaza con el mismo
// mensaje que la CLI (prov:GAT:1).
func TestValidateGateRecordArgs(t *testing.T) {
	ok := GateRecordArgs{Change: "inc-99-example", Gate: GateSpec, Decision: DecisionApproved}
	with := func(mutate func(*GateRecordArgs)) GateRecordArgs {
		a := ok
		mutate(&a)
		return a
	}
	tests := []struct {
		name string
		args GateRecordArgs
		want string // vacío: sin error
	}{
		{"válida", ok, ""},
		{"sin change", with(func(a *GateRecordArgs) { a.Change = "" }), "gate record requiere --change"},
		{"sin gate", with(func(a *GateRecordArgs) { a.Gate = "" }), "gate record requiere --gate"},
		{"sin decision", with(func(a *GateRecordArgs) { a.Decision = "" }), "gate record requiere --decision"},
		{"gate desconocida", with(func(a *GateRecordArgs) { a.Gate = "foo" }), `--gate "foo" desconocido; opciones: spec, design, tasks, role-apply:<rol>, integration`},
		{"role-apply sin rol", with(func(a *GateRecordArgs) { a.Gate = "role-apply:" }), `--gate "role-apply:" no declara un nombre de rol tras "role-apply:"`},
		{"decision desconocida", with(func(a *GateRecordArgs) { a.Decision = "maybe" }), `--decision desconocida "maybe"; opciones: approved, rejected`},
		{"evidencia desconocida", with(func(a *GateRecordArgs) { a.EvidenceKind = "x" }), `--evidence-kind desconocido "x"; opciones: pr_merged, deployment, attestation`},
		{"rejected sin motivo", with(func(a *GateRecordArgs) { a.Decision = DecisionRejected }), "gate record --decision rejected requiere --reason (RF-GAT-06 exige registrar el motivo)"},
		{"rejected con motivo", with(func(a *GateRecordArgs) { a.Decision = DecisionRejected; a.Reason = "falta" }), ""},
		{"pr_merged sin commit", with(func(a *GateRecordArgs) { a.Gate = GateIntegration; a.EvidenceKind = EvidencePRMerged }), "gate record --evidence-kind pr_merged requiere --commit"},
		{"pr_merged con commit", with(func(a *GateRecordArgs) { a.Gate = GateIntegration; a.EvidenceKind = EvidencePRMerged; a.Commit = "abc" }), ""},
		{"hand-off con rejected", with(func(a *GateRecordArgs) { a.HandOff = true; a.Decision = DecisionRejected; a.Reason = "x" }), "`--hand-off` solo acompaña a `--decision approved`; para iterar usa `axiom sdd gate record --change inc-99-example --gate spec --decision rejected --reason <motivo>` sin `--hand-off`"},
		{"hand-off en integration", with(func(a *GateRecordArgs) { a.HandOff = true; a.Gate = GateIntegration }), ""},
		{"hand-off válido", with(func(a *GateRecordArgs) { a.HandOff = true }), ""},
		{"change con barra", with(func(a *GateRecordArgs) { a.Change = "a/b" }), "?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGateRecordArgs(tt.args)
			switch {
			case tt.want == "" && tt.name != "hand-off en integration":
				if err != nil {
					t.Fatalf("error = %v, se esperaba nil", err)
				}
			case tt.want == "?" || tt.name == "hand-off en integration":
				if err == nil {
					t.Fatal("se esperaba un rechazo")
				}
			default:
				if err == nil || err.Error() != tt.want {
					t.Fatalf("error = %v, want %q", err, tt.want)
				}
			}
		})
	}
}
