package uninstall

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

// Pruebas de los restos globales de Kilo, Kiro y Codex (H3) y del criterio de borrado
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

const kiloAxiomConfig = `{
  "agent": {
    "axiom-orchestrator": {"__managed_by": "axiom/sdd", "mode": "primary"},
    "mi-agente": {"mode": "subagent", "description": "del usuario"}
  },
  "mcp": {
    "engram": {"type": "local", "command": ["engram", "mcp", "--tools=agent"]},
    "context7": {"type": "remote", "url": "https://mcp.context7.com/mcp"},
    "mio": {"type": "local", "command": ["node", "s.js"]}
  },
  "share": "disabled",
  "theme": "mi-tema"
}`

// La desinstalación global retira de Kilo lo que tiene la forma de Axiom (agentes con
// la marca, servidores MCP, share por defecto y los plugins gestionados) y conserva lo
// del usuario.
func TestCompleteUninstallGlobalRetiresKiloLeftoversAndKeepsTheUsersConfig(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "kilo", "opencode.json")
	writeGlobalFile(t, configPath, kiloAxiomConfig)
	pluginsDir := filepath.Join(home, ".config", "kilo", "plugins")
	for _, name := range []string{"skill-registry.ts", "model-variants.ts"} {
		writeGlobalFile(t, filepath.Join(pluginsDir, name), "// plugin de Axiom\n")
	}
	writeGlobalFile(t, filepath.Join(pluginsDir, "mi-plugin.ts"), "// plugin del usuario\n")

	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 2 {
		t.Fatalf("antes de desinstalar debe haber dos restos de Kilo (plugins y configuración), got %+v", leftovers)
	}

	globalCompleteUninstall(t, home)

	got := readJSONFileForTest(t, configPath)
	agentsObject, _ := got["agent"].(map[string]any)
	if _, kept := agentsObject["mi-agente"]; !kept || len(agentsObject) != 1 {
		t.Errorf("solo debe quedar el agente del usuario: %#v", agentsObject)
	}
	servers, _ := got["mcp"].(map[string]any)
	if _, kept := servers["mio"]; !kept || len(servers) != 1 {
		t.Errorf("solo debe quedar el servidor MCP del usuario: %#v", servers)
	}
	if got["theme"] != "mi-tema" {
		t.Errorf("el tema del usuario debe conservarse: %#v", got)
	}
	if _, gone := got["share"]; gone {
		t.Errorf("share = disabled (el valor por defecto de Axiom) debe retirarse: %#v", got)
	}
	for _, name := range []string{"skill-registry.ts", "model-variants.ts"} {
		if pathExists(filepath.Join(pluginsDir, name)) {
			t.Errorf("el plugin gestionado %s debe retirarse", name)
		}
	}
	if !pathExists(filepath.Join(pluginsDir, "mi-plugin.ts")) {
		t.Error("el plugin del usuario debe conservarse")
	}
	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 0 {
		t.Errorf("tras desinstalar no debe quedar ningún resto de Axiom: %+v", leftovers)
	}
}

// Sin nada del usuario, el opencode.json de Kilo que queda vacío se borra (mismo
// criterio que el de OpenCode) y también su carpeta de plugins.
func TestCompleteUninstallGlobalDeletesAnEmptiedKiloConfig(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "kilo", "opencode.json")
	writeGlobalFile(t, configPath, `{"agent":{"axiom-orchestrator":{"__managed_by":"axiom/sdd"}},"mcp":{"engram":{"type":"local","command":["engram","mcp","--tools=agent"]}},"share":"disabled"}`)
	writeGlobalFile(t, filepath.Join(home, ".config", "kilo", "plugins", "skill-registry.ts"), "// plugin\n")

	globalCompleteUninstall(t, home)

	if pathExists(configPath) {
		t.Errorf("el opencode.json de Kilo vacío debe borrarse:\n%s", readGlobalFile(t, configPath))
	}
	if pathExists(filepath.Join(home, ".config", "kilo")) {
		t.Error("la carpeta de Kilo que queda vacía debe quitarse")
	}
}

// Un servidor MCP de Kilo con el nombre de los de Axiom pero otra definición es del
// usuario: se conserva y se avisa.
func TestCompleteUninstallGlobalKeepsAKiloServerTheUserDefined(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "kilo", "opencode.json")
	writeGlobalFile(t, configPath, `{"mcp":{"context7":{"type":"remote","url":"https://contexto.interno.example/mcp"}}}`)

	result := globalCompleteUninstall(t, home)

	servers, _ := readJSONFileForTest(t, configPath)["mcp"].(map[string]any)
	if _, kept := servers["context7"]; !kept {
		t.Errorf("el context7 del usuario debe conservarse: %#v", servers)
	}
	if !strings.Contains(strings.Join(result.ManualActions, "\n"), "context7") {
		t.Errorf("debe avisar de que conserva context7: %v", result.ManualActions)
	}
}

func kiroSteeringPath(home string) string {
	return filepath.Join(home, ".kiro", "steering", "axiom.md")
}

// El steering de Kiro es la persona entera con `inclusion: always`: se borra si sigue
// siendo lo que escribió Axiom.
func TestCompleteUninstallGlobalRemovesTheManagedKiroSteering(t *testing.T) {
	personas := embeddedPersonas()
	if len(personas) == 0 {
		t.Skip("no hay personas incrustadas")
	}
	home := t.TempDir()
	writeGlobalFile(t, kiroSteeringPath(home), filemerge.SDDSteeringFrontmatter+"\n"+personas[0]+"\n")

	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 1 || leftovers[0].Label == "" {
		t.Fatalf("el steering de Kiro debe informarse como resto: %+v", leftovers)
	}

	globalCompleteUninstall(t, home)

	if pathExists(kiroSteeringPath(home)) {
		t.Errorf("el steering gestionado debe borrarse:\n%s", readGlobalFile(t, kiroSteeringPath(home)))
	}
	if pathExists(filepath.Join(home, ".kiro", "steering")) {
		t.Error("la carpeta steering que queda vacía debe quitarse")
	}
}

// Si el usuario añadió contenido al steering de Kiro, se conserva con aviso y el
// diagnóstico no lo da por resto de Axiom.
func TestCompleteUninstallGlobalKeepsAKiroSteeringTheUserEdited(t *testing.T) {
	personas := embeddedPersonas()
	if len(personas) == 0 {
		t.Skip("no hay personas incrustadas")
	}
	home := t.TempDir()
	edited := filemerge.SDDSteeringFrontmatter + "\n" + personas[0] + "\n\n## Mi regla\n\nSiempre en castellano.\n"
	writeGlobalFile(t, kiroSteeringPath(home), edited)

	result := globalCompleteUninstall(t, home)

	if got := readGlobalFile(t, kiroSteeringPath(home)); got != edited {
		t.Errorf("el steering editado debe quedar intacto:\n%s", got)
	}
	if !strings.Contains(strings.Join(result.ManualActions, "\n"), kiroSteeringPath(home)) {
		t.Errorf("debe avisar de que conserva el steering: %v", result.ManualActions)
	}
	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 0 {
		t.Errorf("un steering editado por el usuario no es un resto de Axiom: %+v", leftovers)
	}
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
			mcpPath := filepath.Join(home, ".kiro", "settings", "mcp.json")
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
	writeGlobalFile(t, filepath.Join(home, ".config", "kilo", "opencode.json"), `{"theme":"mi-tema"}`)
	writeGlobalFile(t, filepath.Join(home, ".config", "kilo", "plugins", "mi-plugin.ts"), "// propio\n")
	writeGlobalFile(t, kiroSteeringPath(home), "---\ninclusion: always\n---\n\nReglas propias.\n")
	writeGlobalFile(t, filepath.Join(home, ".codex", "config.toml"), "model = \"x\"\n[features]\nmulti_agent = false\n")

	if leftovers := GlobalAgentLeftovers(home); len(leftovers) != 0 {
		t.Fatalf("no hay nada de Axiom: %+v", leftovers)
	}
	if slices.Contains(kiloPluginNames(), "mi-plugin.ts") {
		t.Fatal("mi-plugin.ts no es un plugin de Axiom")
	}
}
