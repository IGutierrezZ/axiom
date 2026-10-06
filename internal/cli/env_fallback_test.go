package cli

import "testing"

// The help text of install and sync promises a GENTLE_AI_* fallback for these
// four variables; these tests pin that the code honours it.
func TestGentleAIEnvFallback(t *testing.T) {
	cases := []struct {
		name      string
		axiomKey  string
		legacyKey string
		axiomVal  string
		legacyVal string
		resolve   func(t *testing.T) string
	}{
		{
			name:      "install scope",
			axiomKey:  ScopeAxiomEnvVar,
			legacyKey: ScopeGentleAIEnvVar,
			axiomVal:  "workspace",
			legacyVal: "global",
			resolve: func(t *testing.T) string {
				got, err := ResolveInstallScope("")
				if err != nil {
					t.Fatalf("ResolveInstallScope() error = %v", err)
				}
				return string(got)
			},
		},
		{
			name:      "channel",
			axiomKey:  ChannelAxiomEnvVar,
			legacyKey: ChannelGentleAIEnvVar,
			axiomVal:  "stable",
			legacyVal: "beta",
			resolve: func(t *testing.T) string {
				got, err := ResolveInstallChannel("")
				if err != nil {
					t.Fatalf("ResolveInstallChannel() error = %v", err)
				}
				return string(got)
			},
		},
		{
			name:      "opencode background subagents",
			axiomKey:  OpenCodeBackgroundSubagentsAxiomEnv,
			legacyKey: OpenCodeBackgroundSubagentsGentleAIEnv,
			axiomVal:  "on",
			legacyVal: "off",
			resolve: func(t *testing.T) string {
				got, _ := lookupOpenCodeBackgroundEnv()
				return got
			},
		},
		{
			name:      "pi background subagents",
			axiomKey:  PiBackgroundSubagentsAxiomEnv,
			legacyKey: PiBackgroundSubagentsGentleAIEnv,
			axiomVal:  "on",
			legacyVal: "off",
			resolve: func(t *testing.T) string {
				got, _ := lookupPiBackgroundEnv()
				return got
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each pair differs from the default on the legacy side, so a fallback that
			// silently never fires cannot pass. defaultValue is what the resolver reports when neither variable is set.
			t.Setenv(tc.axiomKey, "")
			t.Setenv(tc.legacyKey, "")
			defaultValue := tc.resolve(t)

			steps := []struct {
				name      string
				axiomVal  string
				legacyVal string
				want      string
			}{
				{name: "both unset gives the default", want: defaultValue},
				{name: "legacy used when axiom unset", legacyVal: tc.legacyVal, want: tc.legacyVal},
				{name: "axiom wins over legacy", axiomVal: tc.axiomVal, legacyVal: tc.legacyVal, want: tc.axiomVal},
				{name: "axiom alone is honoured", axiomVal: tc.axiomVal, want: tc.axiomVal},
			}
			for _, step := range steps {
				t.Run(step.name, func(t *testing.T) {
					t.Setenv(tc.axiomKey, step.axiomVal)
					t.Setenv(tc.legacyKey, step.legacyVal)
					if got := tc.resolve(t); got != step.want {
						t.Fatalf("resolved %q with %s=%q and %s=%q, want %q", got, tc.axiomKey, step.axiomVal, tc.legacyKey, step.legacyVal, step.want)
					}
				})
			}
		})
	}
}
