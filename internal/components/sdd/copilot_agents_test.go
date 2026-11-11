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

// copilotProjectInject escribe los agentes por fase con el adaptador de VS Code Copilot
// ligado a un proyecto, que es quien los escribe cuando el proyecto lo tiene instalado.
func copilotProjectInject(t *testing.T, root string, opts InjectOptions) InjectionResult {
	t.Helper()
	opts.WorkspaceDir = root
	adapter := agents.BindWorkspace(copilotAdapter(t, model.AgentVSCodeCopilot), root)
	result, err := Inject(root, adapter, "", opts)
	if err != nil {
		t.Fatalf("Inject: %v", err)
	}
	return result
}

func TestInjectWritesOneCopilotAgentPerPhaseWithItsAssignment(t *testing.T) {
	home := copilotAgentsHome(t)
	root := t.TempDir()
	opts := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{
		"sdd-apply":   {Model: "Claude Sonnet 4.6 (copilot)", Effort: "medium"},
		"sdd-explore": {Model: "GPT-5.2 (copilot)"},
	}}
	copilotProjectInject(t, root, opts)

	dir := filepath.Join(root, ".github", "agents")
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
	if !strings.Contains(apply, ".agents/skills/sdd-apply/SKILL.md") {
		t.Errorf("el agente no apunta a las skills del proyecto:\n%s", apply)
	}
	if _, err := os.Stat(filepath.Join(home, ".copilot", "agents")); !os.IsNotExist(err) {
		t.Errorf("la instalación de proyecto no escribe en la carpeta de usuario: %v", err)
	}
}

func TestInjectCopilotAgentsAreIdempotent(t *testing.T) {
	copilotAgentsHome(t)
	root := t.TempDir()
	opts := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-apply": {Model: "GPT-5.2 (copilot)", Effort: "high"}}}
	copilotProjectInject(t, root, opts)
	dir := filepath.Join(root, ".github", "agents")
	before := readAgent(t, dir, "sdd-apply")

	second := copilotProjectInject(t, root, opts)
	for _, file := range second.Files {
		if filepath.Dir(file) == dir {
			t.Errorf("la segunda inyección reescribió %s", file)
		}
	}
	if readAgent(t, dir, "sdd-apply") != before {
		t.Error("los bytes del agente cambiaron entre dos inyecciones iguales")
	}

	changed := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-apply": {Model: "GPT-5.2 (copilot)"}}}
	copilotProjectInject(t, root, changed)
	if fm := frontmatterOf(t, readAgent(t, dir, "sdd-apply")); fm["reasoning-effort"] != nil {
		t.Errorf("quitar el esfuerzo debe regenerar el agente sin él: %v", fm)
	}
}

// Con solo Copilot CLI en el proyecto escribe InjectCopilotProjectAgents: el resultado es
// byte a byte el que escribe el adaptador de VS Code, así que da igual quién los escriba
// primero y el segundo no los reescribe.
func TestInjectCopilotProjectAgentsMatchVSCodeAndAreWrittenOnce(t *testing.T) {
	copilotAgentsHome(t)
	opts := InjectOptions{CopilotPhaseAssignments: map[string]model.CopilotPhaseAssignment{"sdd-design": {Model: "Claude Opus 4.6 (copilot)", Effort: "high"}}}

	viaVSCode := t.TempDir()
	copilotProjectInject(t, viaVSCode, opts)

	viaCLI := t.TempDir()
	adapter := agents.BindWorkspace(copilotAdapter(t, model.AgentVSCodeCopilot), viaCLI)
	first, err := InjectCopilotProjectAgents(viaCLI, adapter, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || len(first.Files) != len(model.CopilotPhaseNames()) {
		t.Fatalf("la primera escritura debe crear un agente por fase: %+v", first)
	}
	second, err := InjectCopilotProjectAgents(viaCLI, adapter, opts)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || len(second.Files) != 0 {
		t.Errorf("la segunda escritura no debe reescribir nada: %+v", second)
	}

	want := filepath.Join(viaVSCode, ".github", "agents")
	got := filepath.Join(viaCLI, ".github", "agents")
	entries, err := os.ReadDir(want)
	if err != nil || len(entries) != len(model.CopilotPhaseNames()) {
		t.Fatalf("agentes de VS Code: %d, %v", len(entries), err)
	}
	for _, entry := range entries {
		phase := copilotAgentPhase(entry.Name())
		if readAgent(t, got, phase) != readAgent(t, want, phase) {
			t.Errorf("%s difiere según quién lo escribió", entry.Name())
		}
	}
}

// En la carpeta de usuario ningún adaptador de Copilot escribe agentes por fase: el
// agente personal de ~/.copilot/agents gana al del proyecto en Copilot CLI y taparía el
// modelo elegido en el proyecto.
func TestInjectCopilotGlobalWritesNoPhaseAgents(t *testing.T) {
	home := copilotAgentsHome(t)
	custom := filepath.Join(t.TempDir(), "copilot-cli")
	t.Setenv("COPILOT_HOME", custom)

	for _, id := range []model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI} {
		adapter := copilotAdapter(t, id)
		if got := adapter.SubAgentsDir(home); got != "" {
			t.Errorf("%s: SubAgentsDir global = %q, quiero vacío", id, got)
		}
		if _, err := Inject(home, adapter, ""); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	for _, dir := range []string{filepath.Join(home, ".copilot", "agents"), filepath.Join(custom, "agents")} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("no se debe crear %s: %v", dir, err)
		}
	}
}

// Una versión anterior escribió los agentes por fase en la carpeta de usuario. El sync
// global los retira, también bajo COPILOT_HOME, y conserva los agentes del usuario.
func TestInjectCopilotGlobalRetiresLegacyManagedAgentsAndKeepsUserOnes(t *testing.T) {
	for _, id := range []model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI} {
		t.Run(string(id), func(t *testing.T) {
			home := copilotAgentsHome(t)
			custom := filepath.Join(t.TempDir(), "copilot-cli")
			t.Setenv("COPILOT_HOME", custom)
			legacy := []string{filepath.Join(home, ".copilot", "agents"), filepath.Join(custom, "agents")}
			const own = "---\nname: mi-agente\n---\npropio\n"
			for _, dir := range legacy {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				for _, phase := range model.CopilotPhaseNames() {
					if err := os.WriteFile(filepath.Join(dir, phase+copilotAgentFileSuffix), []byte("---\nname: "+phase+"\n---\n"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			// Solo la carpeta de COPILOT_HOME tiene un agente del usuario: la otra se queda vacía.
			userAgent := filepath.Join(custom, "agents", "mi-agente.agent.md")
			if err := os.WriteFile(userAgent, []byte(own), 0o644); err != nil {
				t.Fatal(err)
			}

			adapter := copilotAdapter(t, id)
			if _, err := Inject(home, adapter, ""); err != nil {
				t.Fatal(err)
			}
			for _, dir := range legacy {
				for _, phase := range model.CopilotPhaseNames() {
					if _, err := os.Stat(filepath.Join(dir, phase+copilotAgentFileSuffix)); !os.IsNotExist(err) {
						t.Errorf("el agente gestionado %s/%s debe retirarse: %v", dir, phase, err)
					}
				}
			}
			if data, err := os.ReadFile(userAgent); err != nil || string(data) != own {
				t.Errorf("el agente del usuario cambió: %v %q", err, data)
			}
			if _, err := os.Stat(legacy[0]); !os.IsNotExist(err) {
				t.Errorf("la carpeta vacía debe quitarse: %v", err)
			}

			// Idempotente: sin nada que retirar no hace nada ni falla.
			if _, err := Inject(home, adapter, ""); err != nil {
				t.Fatalf("segunda ejecución: %v", err)
			}
			if data, err := os.ReadFile(userAgent); err != nil || string(data) != own {
				t.Errorf("el agente del usuario cambió en la segunda ejecución: %v %q", err, data)
			}
		})
	}
}

func TestInjectCopilotProjectAgentsKeepUserAuthoredAgents(t *testing.T) {
	copilotAgentsHome(t)
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const own = "---\nname: mi-agente\n---\npropio\n"
	path := filepath.Join(dir, "mi-agente.agent.md")
	if err := os.WriteFile(path, []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}
	copilotProjectInject(t, root, InjectOptions{})
	if data, err := os.ReadFile(path); err != nil || string(data) != own {
		t.Errorf("el agente del usuario cambió: %v %q", err, data)
	}
}
