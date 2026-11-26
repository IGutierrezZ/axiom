package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

// Puerto `Prefs.Autofetch` sobre el hub real (C-05-d, RF-HUB-13, design D-03): los cuatro
// estados con la ruta del hub inyectada, sin tocar el del usuario.

func TestHubPrefsAutofetchReportsTheFourStates(t *testing.T) {
	config := filepath.Join(t.TempDir(), "workspaces.json")
	manager, err := hub.NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	master := t.TempDir()
	prefs := hubPrefs{path: config}

	// unregistered: el hub aún no existe y el aviso indica cómo registrar el proyecto.
	enabled, state, warning := prefs.Autofetch(master)
	if enabled || state != branch.AutofetchUnregistered || !strings.Contains(warning, `axiom project add "`+master+`"`) {
		t.Errorf("sin registrar: (%v, %q, %q)", enabled, state, warning)
	}

	// disabled: registrado y sin encender.
	if _, err := manager.Register(master, "proyecto", ""); err != nil {
		t.Fatal(err)
	}
	if enabled, state, warning := prefs.Autofetch(master); enabled || state != branch.AutofetchDisabled || warning != "" {
		t.Errorf("registrado: (%v, %q, %q)", enabled, state, warning)
	}

	// enabled.
	if _, _, err := manager.SetAutofetch(master, true); err != nil {
		t.Fatal(err)
	}
	if enabled, state, warning := prefs.Autofetch(master); !enabled || state != branch.AutofetchEnabled || warning != "" {
		t.Errorf("encendido: (%v, %q, %q)", enabled, state, warning)
	}

	// unreadable: hub corrupto, apagado con aviso y sin modificarlo.
	if err := os.WriteFile(config, []byte("{ roto"), 0o644); err != nil {
		t.Fatal(err)
	}
	enabled, state, warning = prefs.Autofetch(master)
	if enabled || state != branch.AutofetchUnreadable || warning == "" {
		t.Errorf("ilegible: (%v, %q, %q)", enabled, state, warning)
	}
	if data, _ := os.ReadFile(config); string(data) != "{ roto" {
		t.Errorf("el hub ilegible se modificó: %s", data)
	}
}

func TestDefaultWithHubPathReadsTheInjectedHubPrefs(t *testing.T) {
	config := filepath.Join(t.TempDir(), "workspaces.json")
	engine := DefaultWithHubPath(config)

	hp, ok := engine.prefs.(hubPrefs)
	if !ok || hp.path != config {
		t.Fatalf("prefs = %#v, se esperaba hubPrefs con la ruta inyectada", engine.prefs)
	}
}
