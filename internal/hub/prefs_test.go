package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Preferencias personales de worktree del hub (INC-25, REQ-25.29, design D-08): lectura por
// identidad de directorio, valores por defecto y rechazo de lo persistido inválido.

// newPrefsHub crea un hub con un proyecto registrado y devuelve el gestor, la ruta del
// proyecto y la del fichero.
func newPrefsHub(t *testing.T) (*Manager, string, string) {
	t.Helper()
	config := filepath.Join(t.TempDir(), "workspaces.json")
	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "mi-proyecto")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Register(project, "mi-proyecto", ""); err != nil {
		t.Fatal(err)
	}
	return manager, project, config
}

// storePrefs escribe las preferencias en la entrada del proyecto, como lo hará
// SetWorktreePrefs (S7): aquí solo se lee.
func storePrefs(t *testing.T, manager *Manager, mode, root string) {
	t.Helper()
	cfg, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Workspaces[0].WorktreeMode, cfg.Workspaces[0].WorktreeRoot = mode, root
	if err := manager.Save(cfg); err != nil {
		t.Fatal(err)
	}
}

func wantDefaultPrefs(t *testing.T, got WorktreePrefs, project string) {
	t.Helper()
	want := WorktreePrefs{Mode: WorktreeModeWorktree, Root: filepath.Join(filepath.Dir(project), filepath.Base(project)+"-wt")}
	if got != want {
		t.Errorf("preferencias = %+v, se esperaba %+v (modo worktree y <padre>/<nombre>-wt)", got, want)
	}
}

// REQ-25.29, escenario «Sin preferencias el modo efectivo es `worktree`».
func TestLookupWorktreePrefsWithoutPreferencesReturnsTheDefaults(t *testing.T) {
	manager, project, _ := newPrefsHub(t)

	prefs, stored, err := manager.LookupWorktreePrefs(project)

	if err != nil || stored {
		t.Fatalf("LookupWorktreePrefs = (%+v, %v, %v), se esperaba sin error y sin preferencia guardada", prefs, stored, err)
	}
	wantDefaultPrefs(t, prefs, project)
}

// REQ-25.29, escenario «Proyecto no registrado también recibe los valores por defecto»: ni
// el fichero del hub existe.
func TestLookupWorktreePrefsOfAnUnregisteredProjectAlsoGetsTheDefaults(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "no-existe", "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "sin-registrar")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	prefs, stored, err := manager.LookupWorktreePrefs(project)

	if err != nil || stored {
		t.Fatalf("LookupWorktreePrefs = (%+v, %v, %v), se esperaba sin error y sin preferencia guardada", prefs, stored, err)
	}
	wantDefaultPrefs(t, prefs, project)
}

// design D-08: la búsqueda es por identidad de directorio (pathidentity.SameDirectory), no
// por texto: una grafía distinta de la misma carpeta halla la entrada; otra carpeta no.
func TestLookupWorktreePrefsFindsTheProjectByDirectoryIdentity(t *testing.T) {
	manager, project, _ := newPrefsHub(t)
	root := filepath.Join(t.TempDir(), "worktrees")
	storePrefs(t, manager, WorktreeModeAsk, root)

	t.Run("la misma carpeta con otra grafía", func(t *testing.T) {
		respelled := filepath.Join(project, "sub", "..") + string(filepath.Separator)
		if err := os.MkdirAll(filepath.Join(project, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		prefs, stored, err := manager.LookupWorktreePrefs(respelled)
		if err != nil || !stored || prefs != (WorktreePrefs{Mode: WorktreeModeAsk, Root: root}) {
			t.Errorf("LookupWorktreePrefs(%q) = (%+v, %v, %v), se esperaba (ask, %q) guardada", respelled, prefs, stored, err, root)
		}
	})

	t.Run("otra carpeta recibe los valores por defecto", func(t *testing.T) {
		other := filepath.Join(t.TempDir(), "otro")
		if err := os.MkdirAll(other, 0o755); err != nil {
			t.Fatal(err)
		}
		prefs, stored, err := manager.LookupWorktreePrefs(other)
		if err != nil || stored {
			t.Fatalf("LookupWorktreePrefs = (%+v, %v, %v), se esperaba sin preferencia guardada", prefs, stored, err)
		}
		wantDefaultPrefs(t, prefs, other)
	})

	t.Run("solo el modo guardado deja la raíz por defecto", func(t *testing.T) {
		storePrefs(t, manager, WorktreeModeInPlace, "")
		prefs, stored, err := manager.LookupWorktreePrefs(project)
		if err != nil || !stored || prefs.Mode != WorktreeModeInPlace {
			t.Fatalf("LookupWorktreePrefs = (%+v, %v, %v), se esperaba in_place guardado", prefs, stored, err)
		}
		if want := filepath.Join(filepath.Dir(project), filepath.Base(project)+"-wt"); prefs.Root != want {
			t.Errorf("raíz = %q, se esperaba la de por defecto %q", prefs.Root, want)
		}
	})
}

// REQ-25.29: un valor persistido inválido se rechaza con un mensaje en castellano que nombra
// el fichero; el registro no se modifica.
func TestLookupWorktreePrefsRejectsAPersistedInvalidValue(t *testing.T) {
	tests := []struct {
		name, mode, root string
		want             []string
	}{
		{"modo desconocido", "siempre", "", []string{"siempre", "worktree", "in_place", "ask"}},
		{"raíz relativa", WorktreeModeWorktree, filepath.Join("..", "wt"), []string{"ruta absoluta", filepath.Join("..", "wt")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager, project, config := newPrefsHub(t)
			storePrefs(t, manager, tc.mode, tc.root)
			before, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}

			_, _, err = manager.LookupWorktreePrefs(project)

			if err == nil {
				t.Fatal("se esperaba un error por el valor persistido inválido")
			}
			for _, want := range append([]string{config}, tc.want...) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe contener %q", err, want)
				}
			}
			if after, _ := os.ReadFile(config); string(after) != string(before) {
				t.Error("leer las preferencias no debe modificar el hub")
			}
		})
	}
}

// design D-08: si el hub no se puede leer, Lookup devuelve el error (el motor lo traduce a
// los valores por defecto con un aviso).
func TestLookupWorktreePrefsReportsAnUnreadableHub(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	if err := os.WriteFile(config, []byte("{esto no es json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := manager.LookupWorktreePrefs(project); err == nil {
		t.Error("se esperaba un error con un workspaces.json corrupto")
	}
}

// REQ-25.29: las preferencias son campos opcionales de la entrada y el esquema sigue en
// `"1.0"`: sin ellas el JSON no cambia (un binario anterior los ignora).
func TestWorkspaceRecordStoresTheWorktreePrefsWithoutChangingTheSchema(t *testing.T) {
	manager, _, config := newPrefsHub(t)

	read := func() (string, map[string]any) {
		raw, err := os.ReadFile(config)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		return string(raw), decoded
	}

	raw, decoded := read()
	if strings.Contains(raw, "worktree_mode") || strings.Contains(raw, "worktree_root") {
		t.Errorf("sin preferencias el JSON no lleva los campos nuevos:\n%s", raw)
	}
	if decoded["version"] != "1.0" {
		t.Errorf("version = %v, se esperaba 1.0", decoded["version"])
	}

	root := filepath.Join(t.TempDir(), "wt")
	storePrefs(t, manager, WorktreeModeAsk, root)

	raw, decoded = read()
	entry := decoded["workspaces"].([]any)[0].(map[string]any)
	if entry["worktree_mode"] != WorktreeModeAsk || entry["worktree_root"] != root {
		t.Errorf("la entrada debe guardar worktree_mode y worktree_root:\n%s", raw)
	}
	if decoded["version"] != "1.0" {
		t.Errorf("guardar preferencias no cambia el esquema: version = %v", decoded["version"])
	}
	if entry["name"] != "mi-proyecto" || entry["topology"] == "" {
		t.Errorf("el resto de campos se conserva: %v", entry)
	}
}

// REQ-25.29 y design D-08: el modo admite los tres valores (o vacío, sin preferencia) y la
// raíz, una ruta absoluta (o vacía); el rechazo está en castellano y lista lo válido.
func TestValidateWorktreePrefs(t *testing.T) {
	absolute := filepath.Join(t.TempDir(), "wt")
	tests := []struct {
		name    string
		prefs   WorktreePrefs
		wantErr []string
	}{
		{"vacío es sin preferencia", WorktreePrefs{}, nil},
		{"worktree", WorktreePrefs{Mode: "worktree"}, nil},
		{"in_place", WorktreePrefs{Mode: "in_place"}, nil},
		{"ask", WorktreePrefs{Mode: "ask"}, nil},
		{"modo y raíz absoluta", WorktreePrefs{Mode: "ask", Root: absolute}, nil},
		{"modo desconocido", WorktreePrefs{Mode: "siempre"}, []string{"«siempre»", "worktree", "in_place", "ask"}},
		{"el modo distingue mayúsculas", WorktreePrefs{Mode: "Worktree"}, []string{"«Worktree»"}},
		{"raíz relativa", WorktreePrefs{Root: "../wt"}, []string{"ruta absoluta", "../wt"}},
		{"raíz de solo espacios", WorktreePrefs{Root: "  "}, []string{"ruta absoluta"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWorktreePrefs(tc.prefs)
			if len(tc.wantErr) == 0 {
				if err != nil {
					t.Errorf("ValidateWorktreePrefs(%+v) = %v, se esperaba nil", tc.prefs, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateWorktreePrefs(%+v) = nil, se esperaba un error", tc.prefs)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe contener %q", err, want)
				}
			}
		})
	}
}
