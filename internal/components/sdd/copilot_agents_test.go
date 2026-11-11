package sdd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"gopkg.in/yaml.v3"
)

// copilotAgentsHome aísla un home que es además el home real del proceso: así
// COPILOT_HOME y las rutas de VS Code no se salen de él.
func copilotAgentsHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("COPILOT_HOME", "")
	return home
}

func copilotAdapter(t *testing.T, id model.AgentID) agents.Adapter {
	t.Helper()
	adapter, err := agents.NewAdapter(id)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

// frontmatterOf devuelve el frontmatter YAML de un `.agent.md` ya decodificado y
// falla si el fichero no abre y cierra el bloque con `---`.
func frontmatterOf(t *testing.T, content string) map[string]any {
	t.Helper()
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		t.Fatalf("el agente no empieza por ---:\n%s", content)
	}
	block, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("el frontmatter no se cierra:\n%s", content)
	}
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
		t.Fatalf("frontmatter YAML no válido: %v\n%s", err, block)
	}
	return fm
}

func readAgent(t *testing.T, dir, phase string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, phase+copilotAgentFileSuffix))
	if err != nil {
		t.Fatalf("falta el agente %s: %v", phase, err)
	}
	return string(data)
}

func TestCopilotAgentTemplatesMatchPhaseNames(t *testing.T) {
	entries, err := fs.ReadDir(assets.FS, "copilot/agents")
	if err != nil {
		t.Fatal(err)
	}
	var phases []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), copilotAgentFileSuffix) {
			t.Errorf("%s no es un .agent.md", entry.Name())
		}
		phases = append(phases, copilotAgentPhase(entry.Name()))
	}
	want := model.CopilotPhaseNames()
	if len(phases) != len(want) {
		t.Fatalf("plantillas = %v, fases asignables = %v", phases, want)
	}
	for _, phase := range want {
		if _, err := fs.Stat(assets.FS, "copilot/agents/"+phase+copilotAgentFileSuffix); err != nil {
			t.Errorf("la fase %s no tiene plantilla: %v", phase, err)
		}
	}
}

func TestCopilotAgentTemplatesOnlyUseFieldsOfBothHarnesses(t *testing.T) {
	allowed := map[string]bool{"name": true, "description": true, "model": true, "reasoning-effort": true}
	entries, _ := fs.ReadDir(assets.FS, "copilot/agents")
	for _, entry := range entries {
		raw := assets.MustRead("copilot/agents/" + entry.Name())
		content := injectCopilotModelFrontmatter(raw, model.CopilotPhaseAssignment{Model: "GPT-5.2 (copilot)", Effort: "high"})
		fm := frontmatterOf(t, content)
		for key := range fm {
			if !allowed[key] {
				t.Errorf("%s: el campo %q no vale en VS Code y en Copilot CLI a la vez", entry.Name(), key)
			}
		}
		if fm["name"] != copilotAgentPhase(entry.Name()) {
			t.Errorf("%s: name = %v", entry.Name(), fm["name"])
		}
		if desc, _ := fm["description"].(string); strings.TrimSpace(desc) == "" {
			t.Errorf("%s: sin description", entry.Name())
		}
	}
}

func TestInjectCopilotModelFrontmatter(t *testing.T) {
	template := "---\nname: sdd-apply\ndescription: \"x\"\n" + copilotModelFrontmatterPlaceholder + "\n---\ncuerpo\n"
	tests := []struct {
		name       string
		assignment model.CopilotPhaseAssignment
		wantModel  any
		wantEffort any
	}{
		{name: "sin asignación"},
		{name: "modelo vacío ignora el esfuerzo", assignment: model.CopilotPhaseAssignment{Effort: "high"}},
		{name: "solo modelo", assignment: model.CopilotPhaseAssignment{Model: "Claude Sonnet 4.6 (copilot)"}, wantModel: "Claude Sonnet 4.6 (copilot)"},
		{
			name:       "modelo y esfuerzo",
			assignment: model.CopilotPhaseAssignment{Model: "GPT-5.2 (copilot)", Effort: "high"},
			wantModel:  "GPT-5.2 (copilot)", wantEffort: "high",
		},
		{
			name:       "se escapa lo que YAML no admite sin comillas",
			assignment: model.CopilotPhaseAssignment{Model: "Modelo: \"raro\" # no\\", Effort: "a: b"},
			wantModel:  "Modelo: \"raro\" # no\\", wantEffort: "a: b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := injectCopilotModelFrontmatter(template, tt.assignment)
			if strings.Contains(out, "{{") || strings.Contains(out, "\n\n---") {
				t.Fatalf("queda el marcador o una línea en blanco en el frontmatter:\n%s", out)
			}
			fm := frontmatterOf(t, out)
			if fm["model"] != tt.wantModel || fm["reasoning-effort"] != tt.wantEffort {
				t.Errorf("model = %v, reasoning-effort = %v, quiero %v y %v\n%s", fm["model"], fm["reasoning-effort"], tt.wantModel, tt.wantEffort, out)
			}
			wantKeys := 2
			if tt.wantModel != nil {
				wantKeys++
			}
			if tt.wantEffort != nil {
				wantKeys++
			}
			if len(fm) != wantKeys {
				t.Errorf("claves inesperadas en el frontmatter: %v", fm)
			}
		})
	}
}

func TestInjectWritesOneCopilotAgentPerPhaseWithItsAssignment(t *testing.T) {
	home := copilotAgentsHome(t)
	adapter := copilotAdapter(t, model.AgentVSCodeCopilot)
	opts := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{
		"sdd-apply":   {Model: "Claude Sonnet 4.6 (copilot)", Effort: "medium"},
		"sdd-explore": {Model: "GPT-5.2 (copilot)"},
	}}
	if _, err := Inject(home, adapter, "", opts); err != nil {
		t.Fatalf("Inject: %v", err)
	}

	dir := filepath.Join(home, ".copilot", "agents")
	for _, phase := range model.CopilotPhaseNames() {
		fm := frontmatterOf(t, readAgent(t, dir, phase))
		assignment, assigned := opts.CopilotPhaseAssignments[phase]
		if got, _ := fm["model"].(string); got != assignment.Model {
			t.Errorf("%s: model = %q, quiero %q", phase, got, assignment.Model)
		}
		if got, _ := fm["reasoning-effort"].(string); got != assignment.Effort {
			t.Errorf("%s: reasoning-effort = %q, quiero %q", phase, got, assignment.Effort)
		}
		if !assigned && (fm["model"] != nil || fm["reasoning-effort"] != nil) {
			t.Errorf("%s: sin asignación no debe llevar modelo ni esfuerzo: %v", phase, fm)
		}
	}
	apply := readAgent(t, dir, "sdd-apply")
	if strings.Contains(apply, "{{") {
		t.Errorf("quedan marcadores sin resolver:\n%s", apply)
	}
	if !strings.Contains(apply, "~/.copilot/skills/sdd-apply/SKILL.md") {
		t.Errorf("el agente no apunta a las skills de Copilot:\n%s", apply)
	}
}

func TestInjectCopilotAgentsAreIdempotent(t *testing.T) {
	home := copilotAgentsHome(t)
	adapter := copilotAdapter(t, model.AgentVSCodeCopilot)
	opts := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-apply": {Model: "GPT-5.2 (copilot)", Effort: "high"}}}
	if _, err := Inject(home, adapter, "", opts); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".copilot", "agents")
	before := readAgent(t, dir, "sdd-apply")

	second, err := Inject(home, adapter, "", opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range second.Files {
		if filepath.Dir(file) == dir {
			t.Errorf("la segunda inyección reescribió %s", file)
		}
	}
	if readAgent(t, dir, "sdd-apply") != before {
		t.Error("los bytes del agente cambiaron entre dos inyecciones iguales")
	}

	changed := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-apply": {Model: "GPT-5.2 (copilot)"}}}
	if _, err := Inject(home, adapter, "", changed); err != nil {
		t.Fatal(err)
	}
	if fm := frontmatterOf(t, readAgent(t, dir, "sdd-apply")); fm["reasoning-effort"] != nil {
		t.Errorf("quitar el esfuerzo debe regenerar el agente sin él: %v", fm)
	}
}

// VS Code y Copilot CLI comparten ~/.copilot/agents: el segundo adaptador encuentra
// los ficheros ya escritos, idénticos byte a byte, y no los reescribe.
func TestInjectCopilotAgentsAreSharedByVSCodeAndCopilotCLI(t *testing.T) {
	home := copilotAgentsHome(t)
	opts := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-design": {Model: "Claude Opus 4.6 (copilot)", Effort: "high"}}}
	dir := filepath.Join(home, ".copilot", "agents")

	if _, err := Inject(home, copilotAdapter(t, model.AgentVSCodeCopilot), "", opts); err != nil {
		t.Fatal(err)
	}
	snapshot := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		snapshot[entry.Name()] = readAgent(t, dir, copilotAgentPhase(entry.Name()))
	}

	cli := copilotAdapter(t, model.AgentGitHubCopilotCLI)
	if got := cli.SubAgentsDir(home); got != dir {
		t.Fatalf("Copilot CLI escribe en %s, VS Code en %s", got, dir)
	}
	result, err := Inject(home, cli, "", opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range result.Files {
		if filepath.Dir(file) == dir {
			t.Errorf("Copilot CLI reescribió %s, que ya había dejado VS Code", file)
		}
	}
	after, _ := os.ReadDir(dir)
	if len(after) != len(snapshot) || len(snapshot) != len(model.CopilotPhaseNames()) {
		t.Fatalf("%d agentes tras VS Code, %d tras Copilot CLI, %d fases", len(snapshot), len(after), len(model.CopilotPhaseNames()))
	}
	for name, want := range snapshot {
		if got := readAgent(t, dir, copilotAgentPhase(name)); got != want {
			t.Errorf("%s difiere según el adaptador que lo escribió", name)
		}
	}
}

func TestInjectCopilotAgentsFollowCopilotHomeForTheCLI(t *testing.T) {
	home := copilotAgentsHome(t)
	custom := filepath.Join(t.TempDir(), "copilot-cli")
	t.Setenv("COPILOT_HOME", custom)

	if _, err := Inject(home, copilotAdapter(t, model.AgentGitHubCopilotCLI), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(custom, "agents", "sdd-apply.agent.md")); err != nil {
		t.Errorf("Copilot CLI debe escribir en COPILOT_HOME: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".copilot", "agents")); !os.IsNotExist(err) {
		t.Errorf("con COPILOT_HOME no se escribe en ~/.copilot/agents: %v", err)
	}
}

func TestInjectCopilotAgentsInWorkspaceGoToGithubAgents(t *testing.T) {
	home := copilotAgentsHome(t)
	root := t.TempDir()
	adapter := agents.BindWorkspace(copilotAdapter(t, model.AgentVSCodeCopilot), root)
	opts := InjectOptions{
		WorkspaceDir:            root,
		CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-verify": {Model: "GPT-5.2 (copilot)"}},
	}
	if _, err := Inject(root, adapter, "", opts); err != nil {
		t.Fatalf("Inject: %v", err)
	}

	dir := filepath.Join(root, ".github", "agents")
	if fm := frontmatterOf(t, readAgent(t, dir, "sdd-verify")); fm["model"] != "GPT-5.2 (copilot)" {
		t.Errorf("sdd-verify del proyecto: %v", fm)
	}
	if body := readAgent(t, dir, "sdd-apply"); !strings.Contains(body, ".agents/skills/sdd-apply/SKILL.md") || strings.Contains(body, "~/") {
		t.Errorf("en el proyecto las skills se leen de .agents/skills:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(home, ".copilot", "agents")); !os.IsNotExist(err) {
		t.Errorf("la instalación de proyecto no escribe en la carpeta de usuario: %v", err)
	}
}

func TestInjectCopilotAgentsKeepUserAuthoredAgents(t *testing.T) {
	home := copilotAgentsHome(t)
	dir := filepath.Join(home, ".copilot", "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const own = "---\nname: mi-agente\n---\npropio\n"
	path := filepath.Join(dir, "mi-agente.agent.md")
	if err := os.WriteFile(path, []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Inject(home, copilotAdapter(t, model.AgentVSCodeCopilot), ""); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != own {
		t.Errorf("el agente del usuario cambió: %v %q", err, data)
	}
}
