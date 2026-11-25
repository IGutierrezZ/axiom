package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pruebas de los restos globales de Codex (H3) y del criterio de borrado
// de los mcp.json que quedan vacíos.

func globalCompleteUninstall(t *testing.T, home string) Result {
	t.Helper()
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(t.TempDir(), "estado"))
	t.Setenv("LOCALAPPDATA", t.TempDir())
	fakeUserPath(t, "linux")
	result, err := newGlobalService(t, home, Options{}).CompleteUninstall()
	if err != nil {
		t.Fatalf("CompleteUninstall() error = %v", err)
	}
	return result
}

func writeGlobalFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readGlobalFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// En config.toml de Codex se retiran solo las claves con los valores que escribe el
// instalador; lo del usuario, y los valores distintos, se quedan.
func TestCompleteUninstallGlobalCleansCodexConfigKeepingTheUsersKeys(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	writeGlobalFile(t, configPath, strings.Join([]string{
		`model = "modelo-del-usuario"`,
		``,
		`[features]`,
		`multi_agent = true`,
		`otra_funcion = true`,
		``,
		`[agents]`,
		`max_threads = 8`,
		`max_depth = 2`,
		``,
		`[mcp_servers.context7]`,
		`url = "https://mcp.context7.com/mcp"`,
		``,
		`[mcp_servers.mio]`,
		`command = "node"`,
		``,
	}, "\n"))

	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 1 || leftovers[0].Paths[0] != configPath {
		t.Fatalf("config.toml de Codex debe informarse como resto: %+v", leftovers)
	}

	globalCompleteUninstall(t, home)

	got := readGlobalFile(t, configPath)
	for _, kept := range []string{`model = "modelo-del-usuario"`, "otra_funcion = true", "[agents]", "max_threads = 8", "[mcp_servers.mio]"} {
		if !strings.Contains(got, kept) {
			t.Errorf("debe conservar %q:\n%s", kept, got)
		}
	}
	for _, gone := range []string{"multi_agent", "max_depth", "[mcp_servers.context7]"} {
		if strings.Contains(got, gone) {
			t.Errorf("debe retirar %q:\n%s", gone, got)
		}
	}
	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 0 {
		t.Errorf("tras desinstalar no debe quedar ningún resto de Codex: %+v", leftovers)
	}
}

// Un context7 de Codex que no apunta al servidor de Axiom es del usuario.
func TestCompleteUninstallGlobalKeepsACodexContext7TheUserDefined(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	seed := "[mcp_servers.context7]\nurl = \"https://contexto.interno.example/mcp\"\n"
	writeGlobalFile(t, configPath, seed)

	globalCompleteUninstall(t, home)

	if got := readGlobalFile(t, configPath); got != seed {
		t.Errorf("config.toml del usuario debe quedar intacto:\n%s", got)
	}
}

// Un mcp.json que queda vacío se borra solo si lo único que tenía eran servidores con
// la forma de Axiom; si el servidor retirado era del usuario, el documento vacío se
// conserva.
func TestCompleteUninstallGlobalDeletesAnEmptiedMCPFileOnlyWhenItHeldNothingButAxiom(t *testing.T) {
	cases := []struct {
		name       string
		seed       string
		wantExists bool
	}{
		{"solo servidores de Axiom", `{"mcpServers":{"engram":{"command":"engram","args":["mcp","--tools=agent"]},"context7":{"url":"https://mcp.context7.com/mcp"}}}`, false},
		{"engram con otra definición", `{"mcpServers":{"engram":{"command":"node","args":["engram-propio.js"]}}}`, true},
		{"con un servidor del usuario al lado", `{"mcpServers":{"engram":{"command":"engram","args":["mcp","--tools=agent"]},"mio":{"command":"node"}}}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			mcpPath := filepath.Join(home, ".cursor", "mcp.json")
			writeGlobalFile(t, mcpPath, tc.seed)

			globalCompleteUninstall(t, home)

			if got := pathExists(mcpPath); got != tc.wantExists {
				t.Fatalf("mcp.json existe = %v, want %v", got, tc.wantExists)
			}
			if tc.wantExists {
				servers, _ := readJSONFileForTest(t, mcpPath)["mcpServers"].(map[string]any)
				if strings.Contains(tc.name, "servidor del usuario") && (len(servers) != 1 || servers["mio"] == nil) {
					t.Errorf("debe conservar el servidor del usuario: %#v", servers)
				}
			}
		})
	}
}

func TestGlobalAgentLeftoversIsEmptyWithoutAxiom(t *testing.T) {
	home := t.TempDir()
	writeGlobalFile(t, filepath.Join(home, ".codex", "config.toml"), "model = \"x\"\n[features]\nmulti_agent = false\n")

	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 0 {
		t.Fatalf("no hay nada de Axiom: %+v", leftovers)
	}
}
