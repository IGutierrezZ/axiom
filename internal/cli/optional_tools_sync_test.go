package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// GGA y CodeGraph son opcionales: el preset por defecto no los incluye, pero la
// selección explícita sí y las instalaciones que ya los tienen no los pierden.

func TestNormalizeInstallFlagsDefaultNoGGANoCodeGraph(t *testing.T) {
	input, err := NormalizeInstallFlags(InstallFlags{}, system.DetectionResult{})
	if err != nil {
		t.Fatalf("NormalizeInstallFlags() error = %v", err)
	}
	if slices.Contains(input.Selection.Components, model.ComponentGGA) {
		t.Fatalf("el preset por defecto no debe incluir GGA: %v", input.Selection.Components)
	}
	if len(input.Selection.CommunityTools) != 0 {
		t.Fatalf("CodeGraph no debe venir marcado por defecto: %v", input.Selection.CommunityTools)
	}
}

func TestNormalizeInstallFlagsExplicitGGAIsInstalled(t *testing.T) {
	input, err := NormalizeInstallFlags(InstallFlags{Components: []string{"gga"}}, system.DetectionResult{})
	if err != nil {
		t.Fatalf("NormalizeInstallFlags() error = %v", err)
	}
	if !slices.Contains(input.Selection.Components, model.ComponentGGA) {
		t.Fatalf("la selección explícita de gga debe instalarlo: %v", input.Selection.Components)
	}
}

func TestBuildSyncSelectionDoesNotIncludeGGAByDefault(t *testing.T) {
	sel := BuildSyncSelection(SyncFlags{}, []model.AgentID{model.AgentClaudeCode})
	if slices.Contains(sel.Components, model.ComponentGGA) {
		t.Fatalf("sync no debe añadir GGA por su cuenta: %v", sel.Components)
	}
}

func TestRestoreDetectedGGA(t *testing.T) {
	provision := func(t *testing.T, home string) {
		t.Helper()
		t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
		mustWriteFile(t, gga.RuntimePRModePath(home), []byte("#!/bin/sh\n"))
	}
	tests := []struct {
		name      string
		provision bool
		persisted state.InstallState
		want      bool
	}{
		{name: "sin estado y sin GGA aprovisionado", want: false},
		{name: "sin estado y con GGA aprovisionado", provision: true, want: true},
		{
			name:      "selección persistida sin GGA manda sobre la detección",
			provision: true,
			persisted: state.InstallState{SelectionConfigured: true, Components: []model.ComponentID{model.ComponentSDD}},
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
			if tt.provision {
				provision(t, home)
			}
			selection := BuildSyncSelection(SyncFlags{}, []model.AgentID{model.AgentClaudeCode})
			RestorePersistedSelection(&selection, tt.persisted, SyncFlags{})
			RestoreDetectedGGA(home, &selection, tt.persisted)
			if got := slices.Contains(selection.Components, model.ComponentGGA); got != tt.want {
				t.Fatalf("GGA en la selección = %v, want %v (%v)", got, tt.want, selection.Components)
			}
		})
	}
}

func TestRestorePersistedSelectionKeepsGGAOfExistingInstall(t *testing.T) {
	selection := BuildSyncSelection(SyncFlags{}, []model.AgentID{model.AgentClaudeCode})
	RestorePersistedSelection(&selection, state.InstallState{
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentSDD, model.ComponentGGA},
	}, SyncFlags{})
	if !slices.Contains(selection.Components, model.ComponentGGA) {
		t.Fatalf("sync debe conservar GGA si figura en la selección persistida: %v", selection.Components)
	}
}

// runSyncWithState ejecuta `sync` global sobre un HOME temporal con el estado dado
// y devuelve la selección con la que sincronizó.
func runSyncWithState(t *testing.T, persisted *state.InstallState, provisionGGA bool) (home string, selection model.Selection) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks(temp home) error = %v", err)
	}
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	restoreCommand, restoreLookPath, restoreHome, restoreBackupHome := runCommand, cmdLookPath, osUserHomeDir, backup.UserHomeDirFn
	runCommand = func(string, ...string) error { return nil }
	cmdLookPath = func(name string) (string, error) { return "/usr/local/bin/" + name, nil }
	osUserHomeDir = func() (string, error) { return home, nil }
	backup.UserHomeDirFn = func() (string, error) { return home, nil }
	t.Cleanup(func() {
		runCommand, cmdLookPath, osUserHomeDir, backup.UserHomeDirFn = restoreCommand, restoreLookPath, restoreHome, restoreBackupHome
	})
	if provisionGGA {
		mustWriteFile(t, gga.RuntimePRModePath(home), []byte("#!/bin/sh\n# usuario\n"))
	}
	if persisted != nil {
		if err := state.Write(home, *persisted); err != nil {
			t.Fatalf("state.Write() error = %v", err)
		}
	}
	result, err := RunSync([]string{"--scope", "global", "--agents", "opencode"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	return home, result.Selection
}

func TestRunSyncKeepsGGAForExistingInstalls(t *testing.T) {
	t.Run("estado persistido con GGA", func(t *testing.T) {
		home, selection := runSyncWithState(t, &state.InstallState{
			InstalledAgents:     []string{"opencode"},
			SelectionConfigured: true,
			Components:          []model.ComponentID{model.ComponentSDD, model.ComponentGGA},
			Persona:             string(model.PersonaNeutral),
		}, false)
		if !slices.Contains(selection.Components, model.ComponentGGA) {
			t.Fatalf("sync perdió GGA del estado persistido: %v", selection.Components)
		}
		if _, err := os.Stat(gga.ConfigPath(home)); err != nil {
			t.Fatalf("sync debe seguir refrescando la configuración de GGA: %v", err)
		}
	})
	t.Run("sin estado y con GGA ya aprovisionado", func(t *testing.T) {
		home, selection := runSyncWithState(t, nil, true)
		if !slices.Contains(selection.Components, model.ComponentGGA) {
			t.Fatalf("sync perdió el GGA ya aprovisionado: %v", selection.Components)
		}
		if _, err := os.Stat(gga.ConfigPath(home)); err != nil {
			t.Fatalf("sync debe seguir refrescando la configuración de GGA: %v", err)
		}
	})
}

func TestRunSyncDoesNotAddGGAToNewInstalls(t *testing.T) {
	t.Run("estado persistido sin GGA", func(t *testing.T) {
		home, selection := runSyncWithState(t, &state.InstallState{
			InstalledAgents:     []string{"opencode"},
			SelectionConfigured: true,
			Components:          []model.ComponentID{model.ComponentSDD},
			Persona:             string(model.PersonaNeutral),
		}, false)
		if slices.Contains(selection.Components, model.ComponentGGA) {
			t.Fatalf("sync añadió GGA a una instalación que no lo eligió: %v", selection.Components)
		}
		if _, err := os.Stat(gga.ConfigPath(home)); !os.IsNotExist(err) {
			t.Fatalf("sync no debe escribir la configuración de GGA; stat err = %v", err)
		}
	})
	t.Run("sin estado ni GGA", func(t *testing.T) {
		home, selection := runSyncWithState(t, nil, false)
		if slices.Contains(selection.Components, model.ComponentGGA) {
			t.Fatalf("sync añadió GGA sin que existiera: %v", selection.Components)
		}
		if _, err := os.Stat(gga.ConfigPath(home)); !os.IsNotExist(err) {
			t.Fatalf("sync no debe escribir la configuración de GGA; stat err = %v", err)
		}
	})
}
