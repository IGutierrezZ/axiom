package cli

import "testing"

// The GENTLE_AI_* variables are retired: they are no longer read, so setting
// one must not change what the resolver reports. The literals stay here on
// purpose; the test only exists to prove the old names are ignored.
func TestLegacyGentleAIEnvIgnored(t *testing.T) {
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
			legacyKey: "GENTLE_AI_INSTALL_SCOPE",
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
			legacyKey: "GENTLE_AI_CHANNEL",
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
			legacyKey: "GENTLE_AI_OPENCODE_BACKGROUND_SUBAGENTS",
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
			legacyKey: "GENTLE_AI_PI_BACKGROUND_SUBAGENTS",
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
			t.Setenv(tc.axiomKey, "")
			defaultValue := tc.resolve(t)

			t.Run("legacy alone is ignored", func(t *testing.T) {
				t.Setenv(tc.axiomKey, "")
				t.Setenv(tc.legacyKey, tc.legacyVal)
				if got := tc.resolve(t); got != defaultValue {
					t.Fatalf("resolved %q with only %s=%q, want the default %q", got, tc.legacyKey, tc.legacyVal, defaultValue)
				}
			})
			t.Run("axiom is honoured with the legacy one set", func(t *testing.T) {
				t.Setenv(tc.axiomKey, tc.axiomVal)
				t.Setenv(tc.legacyKey, tc.legacyVal)
				if got := tc.resolve(t); got != tc.axiomVal {
					t.Fatalf("resolved %q with %s=%q and %s=%q, want %q", got, tc.axiomKey, tc.axiomVal, tc.legacyKey, tc.legacyVal, tc.axiomVal)
				}
			})
		})
	}
}
