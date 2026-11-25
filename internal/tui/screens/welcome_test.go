package screens_test

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/tui/screens"
)

// ─── WelcomeOptions ──────────────────────────────────────────────────────────

// TestWelcomeOptions_NoRDDEntry: RDD está suspendido (C-01-c), el menú no ofrece
// «Revisión formal RDD».
func TestWelcomeOptions_NoRDDEntry(t *testing.T) {
	for _, showProfiles := range []bool{false, true} {
		for _, opt := range screens.WelcomeOptions(nil, true, showProfiles, 1) {
			if strings.Contains(opt, "RDD") || strings.Contains(opt, "Revisión formal") {
				t.Errorf("el menú no debe ofrecer RDD; opción %q", opt)
			}
		}
	}
}

// TestWelcomeOptions_WithoutProfiles verifies that when showProfiles is false,
// the "Perfiles SDD de OpenCode" option is NOT present.
func TestWelcomeOptions_WithoutProfiles(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, false, 0)
	if !containsOption(opts, "Plugins comunitarios de OpenCode") {
		t.Fatalf("expected dedicated Plugins comunitarios de OpenCode option; got: %v", opts)
	}
	for _, opt := range opts {
		if strings.Contains(opt, "Perfiles SDD de OpenCode") {
			t.Errorf("expected no 'Perfiles SDD de OpenCode' option when showProfiles=false; got: %v", opts)
			break
		}
	}
}

// TestWelcomeOptions_WithProfiles_ZeroCount shows "Perfiles SDD de OpenCode" without a badge.
func TestWelcomeOptions_WithProfiles_ZeroCount(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, true, 0)
	found := false
	for _, opt := range opts {
		if opt == "Perfiles SDD de OpenCode" {
			found = true
		}
		if strings.HasPrefix(opt, "Perfiles SDD de OpenCode (") {
			t.Errorf("expected no badge for 0 profiles, got: %q", opt)
		}
	}
	if !found {
		t.Errorf("expected 'Perfiles SDD de OpenCode' option when showProfiles=true, profileCount=0; got: %v", opts)
	}
}

// TestWelcomeOptions_WithProfiles_CountTwo shows "Perfiles SDD de OpenCode (2)".
func TestWelcomeOptions_WithProfiles_CountTwo(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, true, 2)
	found := false
	for _, opt := range opts {
		if opt == "Perfiles SDD de OpenCode (2)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Perfiles SDD de OpenCode (2)' in options; got: %v", opts)
	}
}

// TestWelcomeOptions_WithProfiles_CountOne shows "Perfiles SDD de OpenCode (1)".
func TestWelcomeOptions_WithProfiles_CountOne(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, true, 1)
	found := false
	for _, opt := range opts {
		if opt == "Perfiles SDD de OpenCode (1)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Perfiles SDD de OpenCode (1)' in options; got: %v", opts)
	}
}

// TestWelcomeOptions_OptionCount_WithoutProfiles verifies 12 options when showProfiles=false.
func TestWelcomeOptions_OptionCount_WithoutProfiles(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, false, 0)
	// Includes the Governance entry.
	want := 12
	if len(opts) != want {
		t.Errorf("WelcomeOptions(showProfiles=false) = %d options, want %d; opts: %v", len(opts), want, opts)
	}
}

// TestWelcomeOptions_OptionCount_WithProfiles verifies 13 options when showProfiles=true.
func TestWelcomeOptions_OptionCount_WithProfiles(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, true, 2)
	// Includes the Governance entry.
	want := 13
	if len(opts) != want {
		t.Errorf("WelcomeOptions(showProfiles=true) = %d options, want %d; opts: %v", len(opts), want, opts)
	}
}

// TestWelcomeOptions_ProfilesInsertedBeforeManageBackups verifies the ordering:
// profiles option sits between "Plugins comunitarios de OpenCode" / "Desinstalar plugin de OpenCode"
// and "📁 Proyectos y Gobernanza SDD ➔" / "Gestionar respaldos".
func TestWelcomeOptions_ProfilesInsertedBeforeManageBackups(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, true, 1)

	modelsIdx := -1
	pluginsIdx := -1
	uninstallIdx := -1
	profilesIdx := -1
	governanceIdx := -1
	manageBackupsIdx := -1
	for i, opt := range opts {
		if opt == "Configurar modelos" {
			modelsIdx = i
		}
		if opt == "Plugins comunitarios de OpenCode" {
			pluginsIdx = i
		}
		if opt == "Desinstalar plugin de OpenCode" {
			uninstallIdx = i
		}
		if strings.HasPrefix(opt, "Perfiles SDD de OpenCode") {
			profilesIdx = i
		}
		if strings.Contains(opt, "Proyectos y Gobernanza SDD") {
			governanceIdx = i
		}
		if opt == "Gestionar respaldos" {
			manageBackupsIdx = i
		}
	}

	if modelsIdx < 0 {
		t.Fatal("option 'Configurar modelos' not found")
	}
	if pluginsIdx < 0 {
		t.Fatal("option 'Plugins comunitarios de OpenCode' not found")
	}
	if uninstallIdx < 0 {
		t.Fatal("option 'Desinstalar plugin de OpenCode' not found")
	}
	if profilesIdx < 0 {
		t.Fatal("option 'Perfiles SDD de OpenCode' not found")
	}
	if governanceIdx < 0 {
		t.Fatal("option 'Proyectos y Gobernanza SDD' not found")
	}
	if manageBackupsIdx < 0 {
		t.Fatal("option 'Gestionar respaldos' not found")
	}

	if pluginsIdx != modelsIdx+1 {
		t.Errorf("plugins option at index %d, expected %d (right after 'Configurar modelos' at %d)",
			pluginsIdx, modelsIdx+1, modelsIdx)
	}
	if uninstallIdx != pluginsIdx+1 {
		t.Errorf("'Desinstalar plugin de OpenCode' at index %d, expected %d (right after plugins at %d)",
			uninstallIdx, pluginsIdx+1, pluginsIdx)
	}
	if profilesIdx != uninstallIdx+1 {
		t.Errorf("profiles option at index %d, expected %d (right after uninstall at %d)",
			profilesIdx, uninstallIdx+1, uninstallIdx)
	}
	if governanceIdx != profilesIdx+1 {
		t.Errorf("governance option at index %d, expected %d (right after profiles at %d)",
			governanceIdx, profilesIdx+1, profilesIdx)
	}
	if manageBackupsIdx != governanceIdx+1 {
		t.Errorf("'Gestionar respaldos' at index %d, expected %d (right after governance at %d)",
			manageBackupsIdx, governanceIdx+1, governanceIdx)
	}
}

func containsOption(opts []string, want string) bool {
	for _, opt := range opts {
		if opt == want {
			return true
		}
	}
	return false
}

func TestWelcomeOptions_IncludesManagedUninstall(t *testing.T) {
	opts := screens.WelcomeOptions(nil, true, false, 0)

	found := false
	for _, opt := range opts {
		if opt == "Desinstalación gestionada" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected 'Desinstalación gestionada' option; got: %v", opts)
	}
}

// ─── RenderWelcome ────────────────────────────────────────────────────────────

// TestRenderWelcome_WithoutProfiles verifies no "Perfiles SDD de OpenCode" in output.
func TestRenderWelcome_WithoutProfiles(t *testing.T) {
	output := screens.RenderWelcome(0, "1.0.0", "", nil, true, false, 0)
	if strings.Contains(output, "Perfiles SDD de OpenCode") {
		snippet := output
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		t.Errorf("RenderWelcome(showProfiles=false) should not contain 'Perfiles SDD de OpenCode'; output snippet: %q", snippet)
	}
}

// TestRenderWelcome_WithProfiles_ZeroCount contains "Perfiles SDD de OpenCode" but no badge.
func TestRenderWelcome_WithProfiles_ZeroCount(t *testing.T) {
	output := screens.RenderWelcome(0, "1.0.0", "", nil, true, true, 0)
	if !strings.Contains(output, "Perfiles SDD de OpenCode") {
		t.Errorf("RenderWelcome(showProfiles=true, count=0) missing 'Perfiles SDD de OpenCode'")
	}
	if strings.Contains(output, "Perfiles SDD de OpenCode (") {
		t.Errorf("RenderWelcome(showProfiles=true, count=0) should NOT have badge")
	}
}

// TestRenderWelcome_WithProfiles_CountTwo contains "Perfiles SDD de OpenCode (2)".
func TestRenderWelcome_WithProfiles_CountTwo(t *testing.T) {
	output := screens.RenderWelcome(0, "1.0.0", "", nil, true, true, 2)
	if !strings.Contains(output, "Perfiles SDD de OpenCode (2)") {
		t.Errorf("RenderWelcome(showProfiles=true, count=2) missing 'Perfiles SDD de OpenCode (2)'")
	}
}

// TestRenderWelcome_WithProfiles_CountOne contains "Perfiles SDD de OpenCode (1)".
func TestRenderWelcome_WithProfiles_CountOne(t *testing.T) {
	output := screens.RenderWelcome(0, "1.0.0", "", nil, true, true, 1)
	if !strings.Contains(output, "Perfiles SDD de OpenCode (1)") {
		t.Errorf("RenderWelcome(showProfiles=true, count=1) missing 'Perfiles SDD de OpenCode (1)'")
	}
}
