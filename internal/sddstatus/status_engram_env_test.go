package sddstatus

import "testing"

// TestShouldTryEngramEnvPrecedence pins AXIOM_SDD_STATUS_ENGRAM as the only
// opt-in variable; the retired GENTLE_AI_SDD_STATUS_ENGRAM is ignored (its
// literal stays here only to prove that).
func TestShouldTryEngramEnvPrecedence(t *testing.T) {
	tests := []struct {
		name   string
		axiom  string
		legacy string
		want   bool
	}{
		{name: "neither set stays off without .engram"},
		{name: "axiom primary", axiom: "1", want: true},
		{name: "legacy variable is ignored", legacy: "1", want: false},
		{name: "both set", axiom: "1", legacy: "1", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvSDDStatusEngramAxiom, tt.axiom)
			t.Setenv("GENTLE_AI_SDD_STATUS_ENGRAM", tt.legacy)
			if got := shouldTryEngram(t.TempDir()); got != tt.want {
				t.Fatalf("shouldTryEngram() = %v, want %v", got, tt.want)
			}
		})
	}
}
