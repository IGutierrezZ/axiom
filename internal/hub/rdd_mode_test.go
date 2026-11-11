package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Preferencia personal del modo RDD (D2): vive junto a las de worktree y se valida igual.

func TestValidateRDDMode(t *testing.T) {
	for _, test := range []struct {
		mode    string
		wantErr bool
	}{
		{mode: ""}, {mode: "on"}, {mode: "off"},
		{mode: "ON", wantErr: true}, {mode: "sometimes", wantErr: true}, {mode: " on", wantErr: true},
	} {
		err := ValidateRDDMode(test.mode)
		if (err != nil) != test.wantErr {
			t.Errorf("ValidateRDDMode(%q) = %v, wantErr %v", test.mode, err, test.wantErr)
		}
	}
}

func TestRDDModeRoundTripsAndLeavesTheOtherPreferencesAlone(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	if _, err := manager.SetWorktreePrefs(project, WorktreePrefs{Mode: WorktreeModeInPlace}); err != nil {
		t.Fatal(err)
	}

	if mode, registered, err := manager.LookupRDDMode(project); err != nil || mode != "" || !registered {
		t.Fatalf("sin preferencia = %q, %v, %v; se esperaba vacío, registrado y sin error", mode, registered, err)
	}
	if _, err := manager.SetRDDMode(project, RDDModeOn); err != nil {
		t.Fatalf("SetRDDMode(on) error = %v", err)
	}
	if mode, _, err := manager.LookupRDDMode(project); err != nil || mode != RDDModeOn {
		t.Fatalf("tras fijar on = %q, %v", mode, err)
	}
	prefs, stored, err := manager.LookupWorktreePrefs(project)
	if err != nil || !stored || prefs.Mode != WorktreeModeInPlace {
		t.Fatalf("fijar el RDD alteró las preferencias de worktree: %+v, %v, %v", prefs, stored, err)
	}
	if _, err := manager.SetWorktreePrefs(project, WorktreePrefs{}); err != nil {
		t.Fatal(err)
	}
	if mode, _, err := manager.LookupRDDMode(project); err != nil || mode != RDDModeOn {
		t.Fatalf("fijar las preferencias de worktree alteró el RDD: %q, %v", mode, err)
	}

	if _, err := manager.SetRDDMode(project, ""); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "rdd_mode") {
		t.Errorf("un modo vacío debe retirar el campo del JSON:\n%s", payload)
	}
}

func TestSetRDDModeRejectsInvalidValuesAndUnregisteredProjectsWithoutWriting(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.SetRDDMode(project, "sometimes"); err == nil || !strings.Contains(err.Error(), "no es válido") {
		t.Errorf("un modo inválido debía rechazarse con un mensaje en castellano: %v", err)
	}
	if _, err := manager.SetRDDMode(filepath.Join(t.TempDir(), "otro"), RDDModeOn); err == nil || !strings.Contains(err.Error(), "no está registrado") {
		t.Errorf("un proyecto sin registrar debía rechazarse: %v", err)
	}
	after, err := os.ReadFile(config)
	if err != nil || string(after) != string(before) {
		t.Errorf("un rechazo modificó el hub: %v", err)
	}
}

func TestLookupRDDModeWithoutHubOrProjectFallsBackWithoutError(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mode, registered, err := manager.LookupRDDMode(t.TempDir()); err != nil || mode != "" || registered {
		t.Fatalf("hub inexistente = %q, %v, %v; se esperaba vacío, sin registrar y sin error", mode, registered, err)
	}
	other, _, _ := newPrefsHub(t)
	if mode, registered, err := other.LookupRDDMode(filepath.Join(t.TempDir(), "ajeno")); err != nil || mode != "" || registered {
		t.Fatalf("proyecto sin registrar = %q, %v, %v", mode, registered, err)
	}
}

func TestLookupRDDModeRejectsAStoredInvalidValueNamingTheFile(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	cfg, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Workspaces[0].RDDMode = "maybe"
	if err := manager.Save(cfg); err != nil {
		t.Fatal(err)
	}
	_, _, err = manager.LookupRDDMode(project)
	if err == nil || !strings.Contains(err.Error(), config) || !strings.Contains(err.Error(), "no es válido") {
		t.Fatalf("error = %v, se esperaba uno que nombre %s y diga que el valor no es válido", err, config)
	}
}
