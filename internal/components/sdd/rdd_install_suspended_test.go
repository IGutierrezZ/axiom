package sdd

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
)

// rddInstallProse es el criterio de terminado de C-01-c: ningún fichero que
// escribe la instalación contiene estas expresiones, sin distinguir mayúsculas.
var rddInstallProse = regexp.MustCompile(`(?i)receipt-driven|Review Execution Contract|Provider Defect Handoff`)

func suspendedInstallEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	t.Cleanup(rddharness.SetForTesting(false))
}

// walkInstalledFiles devuelve la ruta y el contenido de cada fichero de root.
func walkInstalledFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		files[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s) error = %v", root, err)
	}
	return files
}

// TestInjectWorkspaceInstallsNoRDDWhileSuspended pins C-01-c: instalar SDD en
// un proyecto limpio no escribe agentes review-*, no registra el hook de parada
// de revisión y no deja prosa de RDD en ningún fichero, para todos los runtimes.
func TestInjectWorkspaceInstallsNoRDDWhileSuspended(t *testing.T) {
	for _, entry := range catalog.AllAgents() {
		agent := entry.ID
		modes := []model.SDDModeID{model.SDDModeSingle}
		if agent == model.AgentOpenCode {
			modes = append(modes, model.SDDModeMulti)
		}
		for _, mode := range modes {
			t.Run(string(agent)+"/"+string(mode), func(t *testing.T) {
				suspendedInstallEnv(t)
				root := t.TempDir()
				adapter, err := agents.NewAdapter(agent)
				if err != nil {
					t.Fatal(err)
				}
				bound := agents.BindWorkspace(adapter, root)
				if _, err := Inject(root, bound, mode, InjectOptions{WorkspaceDir: root}); err != nil {
					t.Fatalf("Inject(%s, %s) error = %v", agent, mode, err)
				}

				written := walkInstalledFiles(t, root)
				if len(written) == 0 {
					t.Skipf("%s no escribe nada en el proyecto", agent)
				}
				for path, content := range written {
					if isReviewSubAgentFile(filepath.Base(path)) {
						t.Errorf("%s: se escribió el agente de revisión %s", agent, path)
					}
					if strings.Contains(content, "review stop-hook") {
						t.Errorf("%s: %s registra el hook de parada de revisión", agent, path)
					}
					if match := rddInstallProse.FindString(content); match != "" {
						t.Errorf("%s: %s contiene prosa de RDD %q", agent, path, match)
					}
				}
			})
		}
	}
}

// TestInjectWorkspaceRetiresInstalledReviewAgentsWhileSuspended pins que sync
// borra los agentes review-* gestionados que dejó una versión anterior y no
// toca los del usuario con otro nombre.
func TestInjectWorkspaceRetiresInstalledReviewAgentsWhileSuspended(t *testing.T) {
	for _, agent := range []model.AgentID{model.AgentClaudeCode, model.AgentCursor, model.AgentKiroIDE, model.AgentKimi} {
		t.Run(string(agent), func(t *testing.T) {
			suspendedInstallEnv(t)
			root := t.TempDir()
			adapter, err := agents.NewAdapter(agent)
			if err != nil {
				t.Fatal(err)
			}
			bound := agents.BindWorkspace(adapter, root)
			dir := bound.SubAgentsDir(root)
			if dir == "" {
				t.Skipf("%s no tiene carpeta de agentes", agent)
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			var installed []string
			embedded, err := assets.FS.ReadDir(bound.EmbeddedSubAgentsDir())
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range embedded {
				if !isReviewSubAgentFile(entry.Name()) {
					continue
				}
				path := filepath.Join(dir, entry.Name())
				if err := os.WriteFile(path, []byte("name: "+entry.Name()+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				installed = append(installed, path)
			}
			if len(installed) == 0 {
				t.Fatalf("%s no tiene agentes de revisión embebidos", agent)
			}
			mine := filepath.Join(dir, "review-mine.md")
			if err := os.WriteFile(mine, []byte("mine\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := Inject(root, bound, model.SDDModeSingle, InjectOptions{WorkspaceDir: root})
			if err != nil {
				t.Fatalf("Inject() error = %v", err)
			}
			if !result.Changed {
				t.Error("Inject().Changed = false al retirar agentes de revisión")
			}
			for _, path := range installed {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("%s sigue instalado (err = %v)", path, err)
				}
			}
			if _, err := os.Stat(mine); err != nil {
				t.Errorf("el agente del usuario review-mine.md se borró: %v", err)
			}
		})
	}
}

// TestInjectClaudeRetiresReviewStopHookWhileSuspended pins que sync retira el
// hook de parada de Stop y SessionStart, con el nombre actual y el heredado, y
// conserva los hooks del usuario.
func TestInjectClaudeRetiresReviewStopHookWhileSuspended(t *testing.T) {
	suspendedInstallEnv(t)
	root := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	bound := agents.BindWorkspace(adapter, root)
	settingsPath := bound.SettingsPath(root)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const seeded = `{
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [
        {"type": "command", "command": "axiom review stop-hook --agent claude-code", "timeout": 60},
        {"type": "command", "command": "echo mi-hook-de-usuario"}
      ]}
    ],
    "SessionStart": [
      {"matcher": "startup|resume|clear|compact", "hooks": [
        {"type": "command", "command": "gentle-ai review stop-hook --agent claude-code", "timeout": 30}
      ]}
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(seeded), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Inject(root, bound, model.SDDModeSingle, InjectOptions{WorkspaceDir: root}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "review stop-hook") {
		t.Errorf("settings.json conserva el hook de parada de revisión:\n%s", text)
	}
	if !strings.Contains(text, "echo mi-hook-de-usuario") {
		t.Errorf("settings.json perdió el hook del usuario:\n%s", text)
	}

}

// TestInjectOpenCodeRetiresManagedReviewAgentsWhileSuspended pins que sync
// quita de opencode.json los agentes review-* gestionados y sus permisos de
// delegación, conserva una definición del usuario con otra forma y no los
// vuelve a escribir.
func TestInjectOpenCodeRetiresManagedReviewAgentsWhileSuspended(t *testing.T) {
	suspendedInstallEnv(t)
	root := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentOpenCode)
	if err != nil {
		t.Fatal(err)
	}
	bound := agents.BindWorkspace(adapter, root)
	settingsPath := bound.SettingsPath(root)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}

	seed := map[string]any{"agent": map[string]any{
		"review-risk": map[string]any{
			"mode": "subagent", "prompt": "You are R1 Risk.", "permission": map[string]any{"*": "deny"},
		},
		"review-validator": map[string]any{
			"mode": "subagent", "prompt": "validator", "permission": map[string]any{"*": "deny"},
		},
		// Sin prompt ni permission: no tiene la forma que escribe Axiom.
		"review-refuter": map[string]any{"model": "provider/mine"},
		"axiom-orchestrator": map[string]any{
			"mode": "primary", "prompt": "orq",
			"permission": map[string]any{"task": map[string]any{"review-risk": "allow", "sdd-apply": "allow"}},
		},
	}}
	raw, err := json.Marshal(seed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Inject(root, bound, model.SDDModeSingle, InjectOptions{WorkspaceDir: root}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("opencode.json no es JSON válido: %v", err)
	}
	agentsMap, _ := settings["agent"].(map[string]any)
	for _, name := range []string{"review-risk", "review-validator", "review-readability", "review-reliability", "review-resilience"} {
		if _, has := agentsMap[name]; has {
			t.Errorf("opencode.json conserva el agente gestionado %s", name)
		}
	}
	if _, has := agentsMap["review-refuter"]; !has {
		t.Error("opencode.json perdió la definición del usuario review-refuter")
	}
	if strings.Contains(string(data), `"review-risk"`) {
		t.Errorf("opencode.json conserva un permiso o entrada de review-risk:\n%s", data)
	}
	if _, has := agentsMap["sdd-apply"]; !has {
		t.Error("opencode.json perdió el agente sdd-apply")
	}
	for _, name := range opencode.ReviewLensPhases() {
		if _, has := agentsMap[name]; has {
			t.Errorf("sync volvió a escribir %s", name)
		}
	}
}
