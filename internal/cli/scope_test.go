package cli

import (
	"strings"
	"testing"
)

func TestResolveInstallScope(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		envValue  string
		want      InstallScope
		wantErr   bool
	}{
		{
			name:      "empty flag and no env defaults to workspace",
			flagValue: "",
			envValue:  "",
			want:      ScopeWorkspace,
		},
		{
			name:      "flag global returns global",
			flagValue: "global",
			envValue:  "",
			want:      ScopeGlobal,
		},
		{
			name:      "flag workspace returns workspace",
			flagValue: "workspace",
			envValue:  "",
			want:      ScopeWorkspace,
		},
		{
			name:      "env global is refused: global needs --scope global",
			flagValue: "",
			envValue:  "global",
			wantErr:   true,
		},
		{
			name:      "flag global wins over env workspace",
			flagValue: "global",
			envValue:  "workspace",
			want:      ScopeGlobal,
		},
		{
			name:      "env workspace returns workspace",
			flagValue: "",
			envValue:  "workspace",
			want:      ScopeWorkspace,
		},
		{
			name:      "flag takes precedence over env",
			flagValue: "workspace",
			envValue:  "global",
			want:      ScopeWorkspace,
		},
		{
			name:      "invalid flag value returns error",
			flagValue: "project",
			wantErr:   true,
		},
		{
			name:     "invalid env value returns error",
			envValue: "local",
			wantErr:  true,
		},
		{
			name:      "whitespace-only flag treated as unset",
			flagValue: "   ",
			envValue:  "",
			want:      ScopeWorkspace,
		},
	}

	// El arnés del paquete (TestMain) deja que la variable elija el global; aquí se
	// prueba el comportamiento de producción.
	withScopeEnvGlobalOverride(t, false)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(ScopeAxiomEnvVar, "")
			if tt.envValue != "" {
				t.Setenv(scopeEnvVar, tt.envValue)
			}

			got, err := ResolveInstallScope(tt.flagValue)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveInstallScope(%q) error = %v, wantErr %v", tt.flagValue, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("ResolveInstallScope(%q) = %q, want %q", tt.flagValue, got, tt.want)
			}
		})
	}
}

func withScopeEnvGlobalOverride(t *testing.T, value bool) {
	t.Helper()
	orig := ScopeEnvGlobalOverride
	ScopeEnvGlobalOverride = value
	t.Cleanup(func() { ScopeEnvGlobalOverride = orig })
}

func TestResolveInstallScopeRefusesGlobalFromEnv(t *testing.T) {
	withScopeEnvGlobalOverride(t, false)
	t.Setenv(ScopeAxiomEnvVar, "global")
	got, err := ResolveInstallScope("")
	if err == nil {
		t.Fatalf("ResolveInstallScope(\"\") con %s=global = %q, se esperaba un error", ScopeAxiomEnvVar, got)
	}
	if !strings.Contains(err.Error(), "--scope global") {
		t.Errorf("el error debe indicar cómo pedir el ámbito global: %v", err)
	}
	if got, err := ResolveInstallScope("global"); err != nil || got != ScopeGlobal {
		t.Fatalf("ResolveInstallScope(\"global\") = %q, %v; --scope global explícito debe seguir valiendo", got, err)
	}
}

func TestResolveInstallScopeOverrideLetsTheHarnessUseTheEnv(t *testing.T) {
	withScopeEnvGlobalOverride(t, true)
	t.Setenv(ScopeAxiomEnvVar, "global")
	if got, err := ResolveInstallScope(""); err != nil || got != ScopeGlobal {
		t.Fatalf("con ScopeEnvGlobalOverride, ResolveInstallScope(\"\") = %q, %v, want %q", got, err, ScopeGlobal)
	}
}

func TestResolveAgentConfigDir(t *testing.T) {
	tests := []struct {
		name         string
		scope        InstallScope
		homeDir      string
		workspaceDir string
		want         string
	}{
		{
			name:         "global scope uses homeDir",
			scope:        ScopeGlobal,
			homeDir:      "/home/user",
			workspaceDir: "/projects/myapp",
			want:         "/home/user",
		},
		{
			name:         "workspace scope uses workspaceDir",
			scope:        ScopeWorkspace,
			homeDir:      "/home/user",
			workspaceDir: "/projects/myapp",
			want:         "/projects/myapp",
		},
		{
			name:         "workspace scope with empty workspaceDir falls back to homeDir",
			scope:        ScopeWorkspace,
			homeDir:      "/home/user",
			workspaceDir: "",
			want:         "/home/user",
		},
		{
			name:         "workspace scope with whitespace-only workspaceDir falls back to homeDir",
			scope:        ScopeWorkspace,
			homeDir:      "/home/user",
			workspaceDir: "   ",
			want:         "/home/user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAgentConfigDir(tt.scope, tt.homeDir, tt.workspaceDir)
			if got != tt.want {
				t.Fatalf("ResolveAgentConfigDir(%q, %q, %q) = %q, want %q",
					tt.scope, tt.homeDir, tt.workspaceDir, got, tt.want)
			}
		})
	}
}
