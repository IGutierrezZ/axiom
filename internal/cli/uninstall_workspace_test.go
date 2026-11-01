package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	componentuninstall "github.com/IGutierrezZ/axiom/v3/internal/components/uninstall"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// Las pruebas de este fichero cubren `axiom uninstall --scope workspace`: instalan
// con las rutas reales de cada agente en un proyecto con contenido del usuario
// sembrado, desinstalan y comprueban que el proyecto queda como antes salvo el
// formato de los JSON que Axiom reescribe, y que HOME no cambia.

// workspaceUninstallAgents son los agentes de la prueba: los cinco con distribución de
// proyecto propia que cubren cada mecanismo de registro de instrucciones.
const workspaceUninstallAgents = "claude-code,opencode,codex,gemini-cli,vscode-copilot"

// userWorkspaceSeed es el contenido del equipo que existe antes de instalar.
var userWorkspaceSeed = map[string]string{
	"AGENTS.md": "# Reglas del equipo\n\nNo tocar.\n",
	"GEMINI.md": "# Contexto del equipo para Gemini\n",
	filepath.Join(".github", "copilot-instructions.md"):        "Instrucciones del equipo para Copilot.\n",
	filepath.Join(".agents", "skills", "mi-skill", "SKILL.md"): "---\nname: mi-skill\n---\nSkill propia del equipo.\n",
	filepath.Join(".claude", "settings.json"):                  `{"permissions":{"defaultMode":"acceptEdits","deny":["WebFetch","Bash(curl *)"]},"theme":"dark"}`,
	".mcp.json":     `{"mcpServers":{"mi-servidor":{"command":"node","args":["servidor.js"]}}}`,
	"opencode.json": `{"theme":"mi-tema","instructions":["docs/reglas.md"],"mcp":{"mi-servidor":{"type":"local","command":["node","s.js"]}}}`,
	filepath.Join(".gemini", "settings.json"): `{"theme":"GitHub","context":{"fileName":["GEMINI.md","CONTRIBUTING.md"]}}`,
	filepath.Join(".codex", "config.toml"):    "model = \"modelo-del-equipo\"\ndeveloper_instructions = \"Texto del equipo para Codex.\"\n\n[mcp_servers.mi-servidor]\ncommand = \"node\"\nargs = [\"servidor.js\"]\n",
	filepath.Join(".vscode", "mcp.json"):      `{"servers":{"mi-servidor":{"command":"node","args":["servidor.js"]}}}`,
	filepath.Join("docs", "reglas.md"):        "Reglas del equipo.\n",
}

// byteIdenticalSeed son los ficheros sembrados que ningún paso de Axiom reescribe.
var byteIdenticalSeed = []string{
	"AGENTS.md",
	"GEMINI.md",
	filepath.Join(".github", "copilot-instructions.md"),
	filepath.Join(".agents", "skills", "mi-skill", "SKILL.md"),
	filepath.Join("docs", "reglas.md"),
}

func seedUserWorkspace(t *testing.T, workspace string) {
	t.Helper()
	for rel, content := range userWorkspaceSeed {
		mustWriteFile(t, filepath.Join(workspace, rel), []byte(content))
	}
}

// parsedWorkspaceFile lee un fichero del proyecto como estructura (JSON o TOML) para
// compararlo sin depender de la sangría con que Axiom reescribe los JSON.
func parsedWorkspaceFile(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	var parsed any
	switch filepath.Ext(path) {
	case ".json":
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("%s no es JSON válido: %v\n%s", path, err, raw)
		}
	case ".toml":
		var table map[string]any
		if _, err := toml.Decode(string(raw), &table); err != nil {
			t.Fatalf("%s no es TOML válido: %v\n%s", path, err, raw)
		}
		parsed = table
	default:
		parsed = string(raw)
	}
	return parsed
}

// workspaceFileList devuelve las rutas relativas de los ficheros del proyecto, sin
// `.git`, ordenadas.
func workspaceFileList(t *testing.T, root string) []string {
	t.Helper()
	snapshot := permissionsTreeSnapshot(t, root)
	paths := make([]string, 0, len(snapshot))
	for path := range snapshot {
		if path == ".git" || strings.HasPrefix(path, ".git/") {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func seededWorkspacePaths() []string {
	paths := make([]string, 0, len(userWorkspaceSeed))
	for rel := range userWorkspaceSeed {
		paths = append(paths, filepath.ToSlash(rel))
	}
	sort.Strings(paths)
	return paths
}

func runWorkspaceUninstall(t *testing.T, args ...string) componentuninstall.Result {
	t.Helper()
	var stdout bytes.Buffer
	result, err := runUninstallWithInput(append(args, "--yes"), &stdout, strings.NewReader(""))
	if err != nil {
		t.Fatalf("runUninstallWithInput(%v) error = %v\n%s", args, err, stdout.String())
	}
	return result
}

// install workspace de cinco agentes + uninstall workspace: el proyecto queda como
// antes, con el contenido del equipo intacto, y HOME no cambia.
func TestUninstallWorkspaceRevertsTheProjectAndKeepsTheTeamsContent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	seedUserWorkspace(t, workspace)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", workspaceUninstallAgents}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	if got := workspaceFileList(t, workspace); len(got) <= len(userWorkspaceSeed) {
		t.Fatalf("la instalación no escribió nada en el proyecto: %v", got)
	}
	homeBefore := homeSnapshotWithoutState(t, home)

	result := runWorkspaceUninstall(t, "--all", "--scope", "workspace")

	assertSameTree(t, "home", homeBefore, homeSnapshotWithoutState(t, home))

	if got, want := workspaceFileList(t, workspace), seededWorkspacePaths(); !slices.Equal(got, want) {
		t.Fatalf("el proyecto no quedó como antes de instalar.\nahora: %v\nantes: %v\nresultado: %+v", got, want, result)
	}
	for _, rel := range byteIdenticalSeed {
		got, err := os.ReadFile(filepath.Join(workspace, rel))
		if err != nil || string(got) != userWorkspaceSeed[rel] {
			t.Errorf("%s debe seguir byte a byte igual: %v\n%s", rel, err, got)
		}
	}
	for rel, content := range userWorkspaceSeed {
		if slices.Contains(byteIdenticalSeed, rel) {
			continue
		}
		path := filepath.Join(workspace, rel)
		var want any
		switch filepath.Ext(rel) {
		case ".json":
			if err := json.Unmarshal([]byte(content), &want); err != nil {
				t.Fatal(err)
			}
		case ".toml":
			var table map[string]any
			if _, err := toml.Decode(content, &table); err != nil {
				t.Fatal(err)
			}
			want = table
		}
		got := parsedWorkspaceFile(t, path)
		// El texto de developer_instructions que quita el bloque de Axiom termina en
		// el salto de línea que separa los párrafos de Markdown: solo se compara el
		// texto.
		if table, ok := got.(map[string]any); ok && filepath.Ext(rel) == ".toml" {
			if text, isText := table["developer_instructions"].(string); isText {
				table["developer_instructions"] = strings.TrimSpace(text)
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s no quedó como antes de instalar:\nahora  %v\nantes  %v", rel, got, want)
		}
	}
}

// En un proyecto donde Axiom nunca se instaló, desinstalar no falla ni cambia nada.
func TestUninstallWorkspaceInAProjectWithoutAxiomChangesNothing(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	seedUserWorkspace(t, workspace)
	before := permissionsTreeSnapshot(t, workspace)
	homeBefore := homeSnapshotWithoutState(t, home)

	result := runWorkspaceUninstall(t, "--all", "--scope", "workspace")

	assertSameTree(t, "proyecto", before, permissionsTreeSnapshot(t, workspace))
	assertSameTree(t, "home", homeBefore, homeSnapshotWithoutState(t, home))
	if len(result.RemovedFiles) != 0 || len(result.ChangedFiles) != 0 {
		t.Errorf("no debe cambiar nada: %+v", result)
	}
}

// Un segundo uninstall no falla y no cambia nada más.
func TestUninstallWorkspaceIsIdempotent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	seedUserWorkspace(t, workspace)
	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", workspaceUninstallAgents}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	runWorkspaceUninstall(t, "--all", "--scope", "workspace")
	afterFirst := permissionsTreeSnapshot(t, workspace)
	homeAfterFirst := homeSnapshotWithoutState(t, home)

	result := runWorkspaceUninstall(t, "--all", "--scope", "workspace")

	if len(result.RemovedFiles) != 0 || len(result.ChangedFiles) != 0 || len(result.RemovedDirectories) != 0 {
		t.Errorf("el segundo uninstall no debe cambiar nada: %+v", result)
	}
	assertSameTree(t, "proyecto", afterFirst, permissionsTreeSnapshot(t, workspace))
	assertSameTree(t, "home", homeAfterFirst, homeSnapshotWithoutState(t, home))
}

// Desde una subcarpeta del proyecto la raíz es la del proyecto (axiom.yaml o git).
func TestUninstallWorkspaceFromASubfolderUsesTheProjectRoot(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	mustWriteFile(t, filepath.Join(workspace, "axiom.yaml"), []byte("version: 1\n"))
	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code,opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	sub := filepath.Join(workspace, "src", "app")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	var notice bytes.Buffer
	restore := workspaceRootNoticeWriter
	workspaceRootNoticeWriter = &notice
	t.Cleanup(func() { workspaceRootNoticeWriter = restore })

	runWorkspaceUninstall(t, "--all", "--scope", "workspace")

	if got, want := workspaceFileList(t, workspace), []string{"axiom.yaml"}; !slices.Equal(got, want) {
		t.Fatalf("el proyecto no quedó limpio: %v", got)
	}
	if !strings.Contains(notice.String(), "raíz del workspace") {
		t.Errorf("falta el aviso de que la raíz no es la carpeta actual: %q", notice.String())
	}
	if _, err := os.Stat(sub); err != nil {
		t.Errorf("la subcarpeta debe seguir existiendo: %v", err)
	}
}

// Un fichero que Axiom escribe entero y el usuario editó se conserva, con aviso, sin
// las secciones de Axiom.
func TestUninstallWorkspaceKeepsAManagedFileTheUserEdited(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli,opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	edited := map[string]string{
		filepath.Join(".gemini", "AXIOM.md"):   "\nMi nota sobre Gemini.\n",
		filepath.Join(".opencode", "axiom.md"): "\nMi nota sobre OpenCode.\n",
	}
	for rel, note := range edited {
		path := filepath.Join(workspace, rel)
		existing, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("falta %s tras instalar: %v", rel, err)
		}
		mustWriteFile(t, path, append(existing, []byte(note)...))
	}

	result := runWorkspaceUninstall(t, "--all", "--scope", "workspace")

	for rel, note := range edited {
		got, err := os.ReadFile(filepath.Join(workspace, rel))
		if err != nil {
			t.Fatalf("%s editado por el usuario debe conservarse: %v", rel, err)
		}
		if !strings.Contains(string(got), strings.TrimSpace(note)) {
			t.Errorf("%s perdió la nota del usuario:\n%s", rel, got)
		}
		if strings.Contains(string(got), "<!-- axiom:") {
			t.Errorf("%s conserva secciones de Axiom:\n%s", rel, got)
		}
		warned := false
		for _, action := range result.ManualActions {
			if strings.Contains(action, filepath.Join(workspace, rel)) && strings.Contains(action, "Conservado") {
				warned = true
			}
		}
		if !warned {
			t.Errorf("falta el aviso de que %s se conserva: %v", rel, result.ManualActions)
		}
	}
}

// El estado de proyecto y las líneas que Axiom añadió a .git/info/exclude se
// retiran; el global no se toca.
func TestUninstallWorkspaceRemovesProjectStateAndGitExcludeLines(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible")
	}
	stateDir := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	if out, err := exec.Command("git", "-C", workspace, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init: %v\n%s", err, out)
	}
	excludePath := filepath.Join(workspace, ".git", "info", "exclude")
	excludeBefore, _ := os.ReadFile(excludePath)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	projectState := state.ProjectPath(workspace)
	if _, err := os.Stat(projectState); err != nil {
		t.Fatalf("la instalación debe escribir el estado de proyecto: %v", err)
	}
	if got, _ := os.ReadFile(excludePath); string(got) == string(excludeBefore) {
		t.Fatalf("la instalación debe añadir el estado a .git/info/exclude")
	}
	globalBefore := permissionsTreeSnapshot(t, stateDir)

	runWorkspaceUninstall(t, "--all", "--scope", "workspace")

	for _, gone := range []string{projectState, projectState + ".lock", filepath.Dir(projectState)} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("%s debe haberse retirado; stat = %v", gone, err)
		}
	}
	if got, _ := os.ReadFile(excludePath); string(got) != strings.TrimSpace(string(excludeBefore))+"\n" && string(got) != string(excludeBefore) {
		t.Errorf(".git/info/exclude no quedó como antes:\nantes %q\nahora %q", excludeBefore, got)
	}
	// Las copias de seguridad de la propia desinstalación viven en la raíz de estado
	// de máquina; el resto del estado global no cambia.
	globalAfter := permissionsTreeSnapshot(t, stateDir)
	for path, content := range globalBefore {
		if after, ok := globalAfter[path]; !ok || after != content {
			t.Errorf("el estado global cambió: %s", path)
		}
	}
}

// Una desinstalación parcial deja instalados a los demás agentes y conserva el estado
// de proyecto con ellos.
func TestUninstallWorkspacePartialKeepsTheOtherAgentsAndTheSharedSkills(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex,opencode,claude-code"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	runWorkspaceUninstall(t, "--agent", "codex", "--scope", "workspace")

	if _, err := os.Stat(filepath.Join(workspace, ".codex")); !os.IsNotExist(err) {
		t.Errorf(".codex debe desaparecer al desinstalar Codex: %v", err)
	}
	for _, rel := range []string{
		filepath.Join(".agents", "skills", "sdd-init", "SKILL.md"),
		filepath.Join(".claude", "skills", "sdd-init", "SKILL.md"),
		"opencode.json",
	} {
		if _, err := os.Stat(filepath.Join(workspace, rel)); err != nil {
			t.Errorf("%s debe conservarse (otros agentes lo usan): %v", rel, err)
		}
	}
	current, err := state.ProjectStore(home, workspace).Read()
	if err != nil {
		t.Fatalf("el estado de proyecto debe seguir existiendo: %v", err)
	}
	if slices.Contains(current.InstalledAgents, "codex") || !slices.Contains(current.InstalledAgents, "opencode") {
		t.Errorf("InstalledAgents = %v, want sin codex y con opencode", current.InstalledAgents)
	}
}

// Kilo, Cursor y Kiro también vuelven a como estaban, conservando lo del equipo.
func TestUninstallWorkspaceKiloCursorAndKiroKeepTheTeamsContent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	seed := map[string]string{
		"AGENTS.md":                                     "# Reglas del equipo\n",
		filepath.Join(".kilo", "kilo.json"):             `{"instructions":["docs/reglas.md"],"mcp":{"mi-servidor":{"type":"local","command":["node","s.js"]}}}`,
		filepath.Join(".cursor", "mcp.json"):            `{"mcpServers":{"mi-servidor":{"command":"node","args":["s.js"]}}}`,
		filepath.Join(".kiro", "steering", "equipo.md"): "Steering del equipo.\n",
		filepath.Join("docs", "reglas.md"):              "Reglas.\n",
	}
	for rel, content := range seed {
		mustWriteFile(t, filepath.Join(workspace, rel), []byte(content))
	}

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "kilocode,cursor,kiro-ide"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	runWorkspaceUninstall(t, "--all", "--scope", "workspace")

	want := make([]string, 0, len(seed))
	for rel := range seed {
		want = append(want, filepath.ToSlash(rel))
	}
	sort.Strings(want)
	if got := workspaceFileList(t, workspace); !slices.Equal(got, want) {
		t.Fatalf("el proyecto no quedó como antes de instalar.\nahora: %v\nantes: %v", got, want)
	}
	for rel, content := range seed {
		if filepath.Ext(rel) != ".json" {
			if got, _ := os.ReadFile(filepath.Join(workspace, rel)); string(got) != content {
				t.Errorf("%s cambió:\n%s", rel, got)
			}
			continue
		}
		var expected any
		if err := json.Unmarshal([]byte(content), &expected); err != nil {
			t.Fatal(err)
		}
		if got := parsedWorkspaceFile(t, filepath.Join(workspace, rel)); !reflect.DeepEqual(got, expected) {
			t.Errorf("%s no quedó como antes:\nahora %v\nantes %v", rel, got, expected)
		}
	}
}

// Un `## Skills` del equipo en AGENTS.md no es el índice de Axiom: el sync no lo
// adopta, añade su bloque con marcadores aparte, un segundo sync no lo duplica y
// la desinstalación solo retira ese bloque. AGENTS.md queda byte a byte como estaba.
func TestWorkspaceInstallSyncUninstallKeepsTheTeamsSkillsSectionByteIdentical(t *testing.T) {
	teamAgents := map[string]string{
		"prosa":        "# Reglas del equipo\n\n## Skills\n\nAntes de tocar el backend, lee la guía de Go del equipo.\n\n## Otra sección\n\nNo tocar.\n",
		"tabla propia": "# Reglas del equipo\n\n## Skills\n\n| Área | Responsable |\n| --- | --- |\n| Backend | Ana |\n",
	}
	for name, content := range teamAgents {
		t.Run(name, func(t *testing.T) {
			t.Setenv("AXIOM_STATE_DIR", t.TempDir())
			home, workspace, _ := footprintEnv(t)
			seedFootprintHome(t, home)
			agentsPath := filepath.Join(workspace, "AGENTS.md")
			mustWriteFile(t, agentsPath, []byte(content))
			mustWriteFile(t, filepath.Join(workspace, "axiom.yaml"), []byte("workspace:\n  name: equipo\n  topology: monorepo-embedded\n"))

			if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code,opencode"}, macOSDetectionResult()); err != nil {
				t.Fatalf("RunInstall() error = %v", err)
			}
			if _, err := RunSync([]string{"--scope", "workspace"}); err != nil {
				t.Fatalf("RunSync() error = %v", err)
			}
			installed, err := os.ReadFile(agentsPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(installed), "<!-- axiom:skills-index -->") {
				t.Fatalf("el sync debe añadir el bloque gestionado del índice:\n%s", installed)
			}
			if !strings.HasPrefix(string(installed), content) {
				t.Fatalf("lo del equipo debe conservarse intacto al principio:\n%s", installed)
			}
			if strings.Count(string(installed), "<!-- axiom:skills-index -->") != 1 {
				t.Fatalf("el bloque gestionado debe aparecer una sola vez:\n%s", installed)
			}

			runWorkspaceUninstall(t, "--all", "--scope", "workspace")

			got, err := os.ReadFile(agentsPath)
			if err != nil || string(got) != content {
				t.Fatalf("AGENTS.md debe quedar byte a byte como antes de instalar: %v\ngot  %q\nwant %q", err, got, content)
			}
		})
	}
}
