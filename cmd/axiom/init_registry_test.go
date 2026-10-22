package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/update"
)

// TestInitMutatesSelfToolEntryFieldByField pins the D-01 invariant: init() must
// mutate update.Tools field by field — never replace the whole ToolInfo struct —
// so that GoModulePath and any future registry fields survive the rename.
func TestInitMutatesSelfToolEntryFieldByField(t *testing.T) {
	var selfTool *update.ToolInfo
	for i := range update.Tools {
		if update.IsSelfToolName(update.Tools[i].Name) {
			selfTool = &update.Tools[i]
			break
		}
	}
	if selfTool == nil {
		t.Fatal("no self-tool entry found in update.Tools after init()")
	}

	if selfTool.Name != "axiom" {
		t.Errorf("Name = %q, want %q", selfTool.Name, "axiom")
	}
	if selfTool.Owner != "IGutierrezZ" {
		t.Errorf("Owner = %q, want %q", selfTool.Owner, "IGutierrezZ")
	}
	if selfTool.Repo != "axiom" {
		t.Errorf("Repo = %q, want %q", selfTool.Repo, "axiom")
	}
	if selfTool.GoImportPath != "github.com/IGutierrezZ/axiom/v3/cmd/axiom" {
		t.Errorf("GoImportPath = %q, want %q", selfTool.GoImportPath, "github.com/IGutierrezZ/axiom/v3/cmd/axiom")
	}
	// The import path must live inside the declared module so the go install
	// self-upgrade stays available.
	if !selfTool.GoInstallResolvable() {
		t.Errorf("GoInstallResolvable() = false for GoImportPath %q and GoModulePath %q", selfTool.GoImportPath, selfTool.GoModulePath)
	}
	if selfTool.DetectCmd != nil {
		t.Errorf("DetectCmd = %v, want nil", selfTool.DetectCmd)
	}
	if selfTool.VersionPrefix != "v" {
		t.Errorf("VersionPrefix = %q, want %q", selfTool.VersionPrefix, "v")
	}

	// Key assertion: GoModulePath must survive the rename. Full-struct
	// replacement would zero this field (D-01).
	if selfTool.GoModulePath != "github.com/IGutierrezZ/axiom/v3" {
		t.Errorf("GoModulePath = %q, want %q (must be preserved by field-by-field mutation)",
			selfTool.GoModulePath, "github.com/IGutierrezZ/axiom/v3")
	}
}

// TestInitPreservesSelfToolNameRecognition verifies that IsSelfToolName still
// recognizes both historical names after the mutation renames the entry.
func TestInitPreservesSelfToolNameRecognition(t *testing.T) {
	if !update.IsSelfToolName("axiom") {
		t.Error(`IsSelfToolName("axiom") = false, want true`)
	}
	if !update.IsSelfToolName("gentle-ai") {
		t.Error(`IsSelfToolName("gentle-ai") = false, want true`)
	}
}

// Preferencias personales de worktree al registrar (INC-25, REQ-25.30, S7): `init` y
// `project add` las aceptan por opción, las preguntan solo con terminal interactiva y
// validan antes de registrar o crear nada.

// unreadable es una entrada que no debe leerse nunca: su lectura es el fallo de la prueba.
type unreadable struct{ t *testing.T }

func (u unreadable) Read([]byte) (int, error) {
	u.t.Helper()
	u.t.Error("la CLI no debe leer la entrada sin terminal interactiva ni con opciones de preferencia")
	return 0, errors.New("lectura no permitida")
}

// registrationConsole sustituye la terminal de la pregunta de registro y devuelve lo que la
// CLI escribe en ella. input nil es una entrada que falla la prueba si se lee.
func registrationConsole(t *testing.T, interactive bool, input io.Reader) *bytes.Buffer {
	t.Helper()
	if input == nil {
		input = unreadable{t}
	}
	prevInteractive, prevInput, prevOutput := registrationInteractive, registrationInput, registrationOutput
	t.Cleanup(func() {
		registrationInteractive, registrationInput, registrationOutput = prevInteractive, prevInput, prevOutput
	})
	var out bytes.Buffer
	registrationInteractive = func() bool { return interactive }
	registrationInput = input
	registrationOutput = &out
	return &out
}

// prefsHub aísla el hub de la prueba (HOME y USERPROFILE temporales) y devuelve su gestor y
// un proyecto sin registrar.
func prefsHub(t *testing.T) (*hub.Manager, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	manager, err := hub.NewManager("")
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "mi-proyecto")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	return manager, project
}

func projectAdd(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := projectAddCommand(args, &stdout, &stderr)
	return code, stdout.String() + stderr.String()
}

func initProject(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := initCommand(args, &stdout, &stderr)
	return code, stdout.String() + stderr.String()
}

func mustLookup(t *testing.T, manager *hub.Manager, project string) (hub.WorktreePrefs, bool) {
	t.Helper()
	prefs, stored, err := manager.LookupWorktreePrefs(project)
	if err != nil {
		t.Fatalf("LookupWorktreePrefs: %v", err)
	}
	return prefs, stored
}

// REQ-25.30, escenario «Registro con las opciones de preferencia».
func TestProjectAddAcceptsTheWorktreeOptions(t *testing.T) {
	manager, project := prefsHub(t)
	registrationConsole(t, false, nil)
	root := filepath.Join(t.TempDir(), "wt-propio")

	code, out := projectAdd(t, "--worktree-mode", "in_place", "--worktree-root", root, project)

	if code != 0 {
		t.Fatalf("project add salió con %d:\n%s", code, out)
	}
	prefs, stored := mustLookup(t, manager, project)
	if !stored || prefs.Mode != hub.WorktreeModeInPlace || prefs.Root != root {
		t.Errorf("preferencias = (%+v, guardadas %v), se esperaba (in_place, %q)", prefs, stored, root)
	}
	for _, want := range []string{hub.WorktreeModeInPlace, root} {
		if !strings.Contains(out, want) {
			t.Errorf("el resumen debe mostrar %q:\n%s", want, out)
		}
	}
}

// REQ-25.30, escenario «Registro interactivo pregunta con valores por defecto»: pregunta el
// modo (con worktree preseleccionado) y la raíz (con su valor por defecto), y aceptarlos con
// Enter deja el proyecto registrado sin preferencias.
func TestProjectAddAsksInteractivelyAndAcceptingTheDefaultsStoresNothing(t *testing.T) {
	manager, project := prefsHub(t)
	prompts := registrationConsole(t, true, strings.NewReader("\n\n"))

	code, out := projectAdd(t, project)

	if code != 0 {
		t.Fatalf("project add salió con %d:\n%s", code, out)
	}
	wantMode := "Modo de trabajo [worktree] (worktree, in_place, ask):"
	wantRoot := "Raíz de worktrees [" + branch.DefaultWorktreeRoot(project) + "]:"
	for _, want := range []string{wantMode, wantRoot} {
		if !strings.Contains(prompts.String(), want) {
			t.Errorf("la pregunta debe contener %q:\n%s", want, prompts)
		}
	}
	if _, stored := mustLookup(t, manager, project); stored {
		t.Error("aceptar los valores por defecto no debe guardar preferencias")
	}
	if raw, _ := os.ReadFile(manager.GetConfigPath()); strings.Contains(string(raw), "worktree_") {
		t.Errorf("el JSON no debe llevar preferencias:\n%s", raw)
	}
	if strings.Contains(out, "Modo de worktree") || strings.Contains(out, "Raíz de worktrees") {
		t.Errorf("sin preferencias el resumen no las muestra:\n%s", out)
	}
}

// REQ-25.30: un valor distinto del de por defecto sí se guarda; escribir el valor por
// defecto equivale a Enter.
func TestProjectAddStoresWhatTheUserTypesAndOnlyThat(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wt-personal")
	tests := []struct {
		name       string
		input      string
		wantMode   string
		wantRoot   string
		wantStored bool
	}{
		{"modo distinto", "ask\n\n", hub.WorktreeModeAsk, "", true},
		{"raíz distinta", "\n" + root + "\n", "", root, true},
		{"ambos", "in_place\n" + root + "\n", hub.WorktreeModeInPlace, root, true},
		{"el modo por defecto escrito no se guarda", "worktree\n\n", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager, project := prefsHub(t)
			registrationConsole(t, true, strings.NewReader(tc.input))

			if code, out := projectAdd(t, project); code != 0 {
				t.Fatalf("project add salió con %d:\n%s", code, out)
			}

			raw, err := manager.Load()
			if err != nil {
				t.Fatal(err)
			}
			rec := raw.Workspaces[0]
			if rec.WorktreeMode != tc.wantMode || rec.WorktreeRoot != tc.wantRoot {
				t.Errorf("guardado = (%q, %q), se esperaba (%q, %q)", rec.WorktreeMode, rec.WorktreeRoot, tc.wantMode, tc.wantRoot)
			}
		})
	}
}

// REQ-25.30, escenarios «Registro sin terminal no pregunta» y «Registro con las opciones»:
// ni se escribe una pregunta ni se lee la entrada.
func TestInitAndProjectAddDoNotAskWithoutATerminalOrWithOptions(t *testing.T) {
	t.Run("project add sin terminal", func(t *testing.T) {
		manager, project := prefsHub(t)
		prompts := registrationConsole(t, false, nil)

		code, out := projectAdd(t, project)

		if code != 0 {
			t.Fatalf("project add salió con %d:\n%s", code, out)
		}
		if prompts.Len() != 0 {
			t.Errorf("sin terminal no se pregunta:\n%s", prompts)
		}
		if _, stored := mustLookup(t, manager, project); stored {
			t.Error("sin preferencias el proyecto queda con las de por defecto")
		}
	})

	t.Run("init sin terminal", func(t *testing.T) {
		manager, project := prefsHub(t)
		prompts := registrationConsole(t, false, nil)

		code, out := initProject(t, "--path", project)

		if code != 0 {
			t.Fatalf("init salió con %d:\n%s", code, out)
		}
		if prompts.Len() != 0 {
			t.Errorf("sin terminal no se pregunta:\n%s", prompts)
		}
		prefs, stored := mustLookup(t, manager, project)
		if stored || prefs.Mode != hub.WorktreeModeWorktree {
			t.Errorf("preferencias = (%+v, guardadas %v), se esperaba el modo efectivo worktree sin guardar", prefs, stored)
		}
	})

	t.Run("con terminal pero con una opción no se pregunta nada", func(t *testing.T) {
		for _, args := range [][]string{{"--worktree-mode", "ask"}, {"--worktree-root", filepath.Join(t.TempDir(), "wt")}} {
			manager, project := prefsHub(t)
			prompts := registrationConsole(t, true, nil)

			code, out := projectAdd(t, append(args, project)...)

			if code != 0 {
				t.Fatalf("project add %v salió con %d:\n%s", args, code, out)
			}
			if prompts.Len() != 0 {
				t.Errorf("con %v no se pregunta:\n%s", args, prompts)
			}
			cfg, err := manager.Load()
			if err != nil {
				t.Fatal(err)
			}
			if rec := cfg.Workspaces[0]; rec.WorktreeMode == "" && rec.WorktreeRoot == "" {
				t.Errorf("la opción %v debe guardarse", args)
			}
		}
	})
}

// REQ-25.30: `init` acepta las mismas opciones y la misma pregunta, antes de escribir nada.
func TestInitAcceptsTheWorktreeOptionsAndAsksInteractively(t *testing.T) {
	t.Run("opciones", func(t *testing.T) {
		manager, project := prefsHub(t)
		registrationConsole(t, false, nil)
		root := filepath.Join(t.TempDir(), "wt-init")

		code, out := initProject(t, "--path", project, "--worktree-mode", "ask", "--worktree-root", root)

		if code != 0 {
			t.Fatalf("init salió con %d:\n%s", code, out)
		}
		prefs, stored := mustLookup(t, manager, project)
		if !stored || prefs.Mode != hub.WorktreeModeAsk || prefs.Root != root {
			t.Errorf("preferencias = (%+v, guardadas %v), se esperaba (ask, %q)", prefs, stored, root)
		}
		if !strings.Contains(out, "ask") || !strings.Contains(out, root) {
			t.Errorf("el resumen debe mostrar las preferencias guardadas:\n%s", out)
		}
	})

	t.Run("pregunta y un valor distinto se guarda", func(t *testing.T) {
		manager, project := prefsHub(t)
		prompts := registrationConsole(t, true, strings.NewReader("in_place\n\n"))

		code, out := initProject(t, "--path", project)

		if code != 0 {
			t.Fatalf("init salió con %d:\n%s", code, out)
		}
		if want := "Raíz de worktrees [" + branch.DefaultWorktreeRoot(project) + "]:"; !strings.Contains(prompts.String(), want) {
			t.Errorf("la pregunta debe contener %q:\n%s", want, prompts)
		}
		if prefs, stored := mustLookup(t, manager, project); !stored || prefs.Mode != hub.WorktreeModeInPlace {
			t.Errorf("preferencias = (%+v, guardadas %v), se esperaba in_place", prefs, stored)
		}
	})

	t.Run("aceptar los valores por defecto no guarda nada", func(t *testing.T) {
		manager, project := prefsHub(t)
		registrationConsole(t, true, strings.NewReader("\n\n"))

		if code, out := initProject(t, "--path", project); code != 0 {
			t.Fatalf("init salió con %d:\n%s", code, out)
		}
		if _, stored := mustLookup(t, manager, project); stored {
			t.Error("aceptar los valores por defecto no debe guardar preferencias")
		}
	})
}

// REQ-25.30: un valor inválido en una opción falla con salida 1 y `[ERROR]`, sin registrar
// el proyecto ni crear nada.
func TestInitAndProjectAddRejectAnInvalidWorktreeOptionWithoutSideEffects(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"modo siempre", []string{"--worktree-mode", "siempre"}, []string{"«siempre»", "worktree", "in_place", "ask"}},
		{"raíz relativa", []string{"--worktree-root", "../wt"}, []string{"ruta absoluta", "../wt"}},
	}
	for _, tc := range tests {
		t.Run("project add "+tc.name, func(t *testing.T) {
			manager, project := prefsHub(t)
			registrationConsole(t, false, nil)

			code, out := projectAdd(t, append(tc.args, project)...)

			if code != 1 || !strings.Contains(out, "[ERROR]") {
				t.Fatalf("project add = (%d, %q), se esperaba salida 1 con [ERROR]", code, out)
			}
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("el mensaje debe contener %q:\n%s", want, out)
				}
			}
			if _, err := os.Stat(manager.GetConfigPath()); !os.IsNotExist(err) {
				t.Error("un valor inválido no debe registrar el proyecto ni crear el hub")
			}
		})

		t.Run("init "+tc.name, func(t *testing.T) {
			manager, _ := prefsHub(t)
			registrationConsole(t, false, nil)
			target := filepath.Join(t.TempDir(), "nuevo", "proyecto")

			code, out := initProject(t, append(tc.args, "--path", target)...)

			if code != 1 || !strings.Contains(out, "[ERROR]") {
				t.Fatalf("init = (%d, %q), se esperaba salida 1 con [ERROR]", code, out)
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Error("un valor inválido no debe crear el directorio del proyecto")
			}
			if _, err := os.Stat(manager.GetConfigPath()); !os.IsNotExist(err) {
				t.Error("un valor inválido no debe registrar el proyecto ni crear el hub")
			}
		})
	}
}

// REQ-25.29: registrar de nuevo con solo uno de los valores conserva el otro que ya estaba
// guardado.
func TestProjectAddAgainWithOneOptionKeepsTheOtherStoredValue(t *testing.T) {
	manager, project := prefsHub(t)
	registrationConsole(t, false, nil)
	root := filepath.Join(t.TempDir(), "wt")
	if code, out := projectAdd(t, "--worktree-root", root, project); code != 0 {
		t.Fatalf("project add salió con %d:\n%s", code, out)
	}

	if code, out := projectAdd(t, "--worktree-mode", "ask", project); code != 0 {
		t.Fatalf("segundo project add salió con %d:\n%s", code, out)
	}

	if prefs, _ := mustLookup(t, manager, project); prefs.Mode != hub.WorktreeModeAsk || prefs.Root != root {
		t.Errorf("preferencias = %+v, se esperaba (ask, %q)", prefs, root)
	}
}

// resolveRegistrationWorktreePrefs: la regla de resolución, sin pasar por la CLI.
func TestResolveRegistrationWorktreePrefs(t *testing.T) {
	master := filepath.Join(t.TempDir(), "maestra")
	defaultRoot := branch.DefaultWorktreeRoot(master)
	custom := filepath.Join(t.TempDir(), "otra")

	t.Run("un modo inválido tecleado se vuelve a preguntar", func(t *testing.T) {
		var out bytes.Buffer

		prefs, err := resolveRegistrationWorktreePrefs("", "", true, strings.NewReader("siempre\nask\n\n"), &out, master)

		if err != nil || prefs != (hub.WorktreePrefs{Mode: hub.WorktreeModeAsk}) {
			t.Fatalf("= (%+v, %v), se esperaba (ask)", prefs, err)
		}
		if got := strings.Count(out.String(), "Modo de trabajo ["); got != 2 {
			t.Errorf("el modo se pregunta %d veces, se esperaba 2:\n%s", got, out.String())
		}
		if !strings.Contains(out.String(), "«siempre»") {
			t.Errorf("la nueva pregunta explica el rechazo:\n%s", out.String())
		}
	})

	t.Run("una raíz relativa tecleada se vuelve a preguntar", func(t *testing.T) {
		var out bytes.Buffer

		prefs, err := resolveRegistrationWorktreePrefs("", "", true, strings.NewReader("\n../wt\n"+custom+"\n"), &out, master)

		if err != nil || prefs != (hub.WorktreePrefs{Root: custom}) {
			t.Fatalf("= (%+v, %v), se esperaba la raíz %q", prefs, err, custom)
		}
	})

	t.Run("el fin de la entrada acepta los valores por defecto", func(t *testing.T) {
		var out bytes.Buffer

		prefs, err := resolveRegistrationWorktreePrefs("", "", true, strings.NewReader(""), &out, master)

		if err != nil || prefs != (hub.WorktreePrefs{}) {
			t.Fatalf("= (%+v, %v), se esperaba sin preferencias", prefs, err)
		}
	})

	t.Run("la raíz por defecto escrita no se guarda", func(t *testing.T) {
		var out bytes.Buffer

		prefs, err := resolveRegistrationWorktreePrefs("", "", true, strings.NewReader("\n"+defaultRoot+"\n"), &out, master)

		if err != nil || prefs != (hub.WorktreePrefs{}) {
			t.Fatalf("= (%+v, %v), se esperaba sin preferencias", prefs, err)
		}
	})

	t.Run("las opciones se devuelven tal cual aunque coincidan con el valor por defecto", func(t *testing.T) {
		prefs, err := resolveRegistrationWorktreePrefs(hub.WorktreeModeWorktree, defaultRoot, true, unreadable{t}, io.Discard, master)

		if err != nil || prefs != (hub.WorktreePrefs{Mode: hub.WorktreeModeWorktree, Root: defaultRoot}) {
			t.Fatalf("= (%+v, %v), se esperaban las opciones explícitas", prefs, err)
		}
	})
}
