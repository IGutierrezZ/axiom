package hub

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Preferencia `autofetch` del hub (C-05-d, RF-HUB-07, RF-HUB-13, design D-03): apagada por
// defecto, persistente, idempotente y buscada por identidad de proyecto.

func TestAutofetchIsOffByDefaultAndWhenTheKeyIsAbsent(t *testing.T) {
	manager, project, config := newPrefsHub(t)

	enabled, registered, err := manager.LookupAutofetch(project)

	if err != nil || enabled || !registered {
		t.Fatalf("LookupAutofetch = (%v, %v, %v), se esperaba apagado, registrado y sin error", enabled, registered, err)
	}
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "autofetch") {
		t.Errorf("el hub no debería guardar la clave autofetch apagada:\n%s", data)
	}
}

func TestAutofetchPersistsAndTurnsOffAgain(t *testing.T) {
	manager, project, _ := newPrefsHub(t)

	rec, changed, err := manager.SetAutofetch(project, true)
	if err != nil || !changed || !rec.Autofetch {
		t.Fatalf("SetAutofetch(true) = (%+v, %v, %v), se esperaba encendido y cambiado", rec, changed, err)
	}
	// Un gestor nuevo sobre el mismo fichero lo ve encendido.
	reopened, err := NewManager(manager.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if enabled, registered, err := reopened.LookupAutofetch(project); err != nil || !enabled || !registered {
		t.Fatalf("LookupAutofetch tras reabrir = (%v, %v, %v), se esperaba encendido", enabled, registered, err)
	}

	if _, changed, err := manager.SetAutofetch(project, false); err != nil || !changed {
		t.Fatalf("SetAutofetch(false) = (%v, %v), se esperaba un cambio", changed, err)
	}
	if enabled, _, _ := manager.LookupAutofetch(project); enabled {
		t.Error("el autofetch debería estar apagado tras apagarlo")
	}
}

func TestAutofetchSetIsIdempotentAndDoesNotRewriteTheHub(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	if _, _, err := manager.SetAutofetch(project, true); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}

	_, changed, err := manager.SetAutofetch(project, true)

	if err != nil || changed {
		t.Fatalf("SetAutofetch repetido = (%v, %v), se esperaba sin cambio", changed, err)
	}
	after, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("workspaces.json cambió con una orden idempotente:\nantes:\n%s\ndespués:\n%s", before, after)
	}
}

func TestAutofetchSurvivesRegisterAndGoesAwayWithUnregister(t *testing.T) {
	manager, project, _ := newPrefsHub(t)
	rec, _, err := manager.SetAutofetch(project, true)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Register(project, "otro-nombre", ""); err != nil {
		t.Fatal(err)
	}
	if enabled, _, _ := manager.LookupAutofetch(project); !enabled {
		t.Error("el autofetch debería sobrevivir a un nuevo Register")
	}
	// Las preferencias de worktree tampoco lo pisan.
	if _, err := manager.SetWorktreePrefs(project, WorktreePrefs{Mode: WorktreeModeInPlace}); err != nil {
		t.Fatal(err)
	}
	if enabled, _, _ := manager.LookupAutofetch(project); !enabled {
		t.Error("el autofetch debería sobrevivir a SetWorktreePrefs")
	}

	if err := manager.Unregister(rec.ID); err != nil {
		t.Fatal(err)
	}
	if enabled, registered, err := manager.LookupAutofetch(project); err != nil || enabled || registered {
		t.Errorf("LookupAutofetch tras Unregister = (%v, %v, %v), se esperaba apagado y no registrado", enabled, registered, err)
	}
}

func TestAutofetchOfAnUnregisteredProjectIsRejectedWithoutWriting(t *testing.T) {
	config := filepath.Join(t.TempDir(), "workspaces.json")
	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()

	if enabled, registered, err := manager.LookupAutofetch(project); err != nil || enabled || registered {
		t.Errorf("LookupAutofetch = (%v, %v, %v), se esperaba apagado y no registrado sin error", enabled, registered, err)
	}
	_, changed, err := manager.SetAutofetch(project, true)

	if !errors.Is(err, ErrProjectNotRegistered) || changed {
		t.Fatalf("SetAutofetch = (%v, %v), se esperaba ErrProjectNotRegistered", changed, err)
	}
	if _, statErr := os.Stat(config); !os.IsNotExist(statErr) {
		t.Errorf("SetAutofetch de un proyecto no registrado no debe crear el hub: %v", statErr)
	}
}

func TestAutofetchWithAnUnreadableHubDoesNotWrite(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	broken := []byte("{ esto no es json")
	if err := os.WriteFile(config, broken, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := manager.LookupAutofetch(project); err == nil {
		t.Error("LookupAutofetch con un hub ilegible debería dar error")
	}
	_, _, err := manager.SetAutofetch(project, true)
	if err == nil || errors.Is(err, ErrProjectNotRegistered) {
		t.Fatalf("SetAutofetch con un hub ilegible = %v, se esperaba el error de lectura", err)
	}
	after, _ := os.ReadFile(config)
	if !bytes.Equal(after, broken) {
		t.Errorf("el hub ilegible se modificó:\n%s", after)
	}
}

// Un binario anterior desconoce la clave: al leer la entrada la ignora.
func TestAnOlderBinaryIgnoresTheAutofetchKey(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	if _, _, err := manager.SetAutofetch(project, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}

	var old struct {
		Workspaces []struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		t.Fatalf("un struct sin el campo debería leer el hub: %v", err)
	}
	if len(old.Workspaces) != 1 || old.Workspaces[0].Path == "" {
		t.Errorf("la entrada se leyó mal: %+v", old)
	}
}

// design D-03: `axiom.yaml` nunca decide el interruptor; ni el hub lo lee ni el modelo lo conoce.
func TestAutofetchIgnoresAnAutofetchKeyInAxiomYaml(t *testing.T) {
	manager, project, _ := newPrefsHub(t)
	if err := os.WriteFile(filepath.Join(project, "axiom.yaml"), []byte("autofetch: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if enabled, _, err := manager.LookupAutofetch(project); err != nil || enabled {
		t.Errorf("LookupAutofetch = (%v, %v), una clave en axiom.yaml no debe encenderlo", enabled, err)
	}
}

// hueco 8: la entrada registrada con otra grafía de mayúsculas se encuentra en Windows.
func TestAutofetchFindsTheProjectRegardlessOfCaseOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("la comparación sin distinguir mayúsculas solo rige en Windows")
	}
	manager, project, _ := newPrefsHub(t)
	if _, _, err := manager.SetAutofetch(project, true); err != nil {
		t.Fatal(err)
	}

	respelled := strings.ToUpper(project)
	if enabled, registered, err := manager.LookupAutofetch(respelled); err != nil || !enabled || !registered {
		t.Errorf("LookupAutofetch(%q) = (%v, %v, %v), se esperaba encontrar el proyecto", respelled, enabled, registered, err)
	}
	if _, changed, err := manager.SetAutofetch(strings.ToLower(project), true); err != nil || changed {
		t.Errorf("SetAutofetch con otra grafía = (%v, %v), se esperaba la misma entrada sin cambio", changed, err)
	}
}
