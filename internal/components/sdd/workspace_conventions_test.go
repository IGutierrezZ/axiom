package sdd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/sharedskills"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// En un proyecto, el orquestador de cada agente sitúa los ficheros de convención en la
// carpeta de skills efectiva y no en .agent/skills; con la instalación global el texto
// no cambia.
func TestAdaptConventionsClauseForWorkspace(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		agent model.AgentID
		want  string
	}{
		{model.AgentGeminiCLI, "Convention files under `.agents/skills/_shared/`:"},
		{model.AgentCursor, "Convention files under `.agents/skills/_shared/`:"},
		{model.AgentKiroIDE, "Convention files under `.kiro/skills/_shared/`:"},
	} {
		adapter, err := agents.NewAdapter(tc.agent)
		if err != nil {
			t.Fatal(err)
		}
		prompt, err := renderSessionPreflightPrompt(adapter, InjectOptions{})
		if err != nil {
			t.Fatalf("%s: %v", tc.agent, err)
		}
		if !workspaceConventionsClause.MatchString(prompt) {
			t.Fatalf("%s: el orquestador ya no trae la frase de convenciones que se adapta", tc.agent)
		}
		got := adaptConventionsClauseForWorkspace(prompt, agents.BindWorkspace(adapter, root), root)
		if strings.Contains(got, ".agent/skills") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: la ruta de convenciones no apunta a %q", tc.agent, tc.want)
		}
		if same := adaptConventionsClauseForWorkspace(prompt, adapter, ""); same != prompt {
			t.Errorf("%s: con la instalación global el texto no debe cambiar", tc.agent)
		}
	}
	// Un agente sin esa frase no cambia.
	claudeAdapter := claude.NewAdapter()
	if got := adaptConventionsClauseForWorkspace("sin frase", claudeAdapter, root); got != "sin frase" {
		t.Errorf("un texto sin la frase no debe cambiar: %q", got)
	}
}

// La carpeta compartida .agents/skills se escribe con la identidad genérica de runtime,
// igual que la copia de compatibilidad, para que su contenido no dependa de qué agente
// la escriba. La carpeta propia de Claude conserva su identidad.
func TestInjectWritesSharedSkillsWithGenericRuntimeIdentity(t *testing.T) {
	for _, agent := range []model.AgentID{model.AgentOpenCode, model.AgentGeminiCLI, model.AgentCodex} {
		root := t.TempDir()
		adapter, err := agents.NewAdapter(agent)
		if err != nil {
			t.Fatal(err)
		}
		bound := agents.BindWorkspace(adapter, root)
		if _, err := Inject(root, bound, model.SDDModeSingle, InjectOptions{WorkspaceDir: root}); err != nil {
			t.Fatalf("%s: %v", agent, err)
		}
		shared := sharedskills.Dir(root)
		ledger, err := os.ReadFile(filepath.Join(shared, "_shared", "review-ledger-contract.md"))
		if err != nil {
			t.Fatalf("%s: %v", agent, err)
		}
		if strings.Contains(string(ledger), runtimeAgentIDPlaceholder) || strings.Contains(string(ledger), string(agent)+"\"") {
			t.Errorf("%s: la copia compartida no debe llevar la identidad del agente ni el marcador sin resolver", agent)
		}
		if _, err := os.Stat(filepath.Join(shared, "sdd-apply", "SKILL.md")); err != nil {
			t.Errorf("%s: falta sdd-apply en la copia compartida: %v", agent, err)
		}
	}
}

// SkillsAlreadyWritten evita repetir la escritura de la carpeta de skills: el segundo
// agente que la comparte no la reescribe ni la declara entre los ficheros que toca,
// pero el resto de su inyección sigue igual.
func TestInjectSkipsSkillsDirWhenAlreadyWritten(t *testing.T) {
	root := t.TempDir()
	shared := sharedskills.Dir(root)
	inject := func(agent model.AgentID, already bool) InjectionResult {
		t.Helper()
		adapter, err := agents.NewAdapter(agent)
		if err != nil {
			t.Fatal(err)
		}
		result, err := Inject(root, agents.BindWorkspace(adapter, root), model.SDDModeSingle, InjectOptions{WorkspaceDir: root, SkillsAlreadyWritten: already})
		if err != nil {
			t.Fatalf("%s: %v", agent, err)
		}
		return result
	}

	first := inject(model.AgentGeminiCLI, false)
	second := inject(model.AgentOpenCode, true)

	underShared := func(files []string) (n int) {
		for _, f := range files {
			if strings.HasPrefix(f, shared+string(filepath.Separator)) {
				n++
			}
		}
		return n
	}
	if underShared(first.Files) == 0 {
		t.Errorf("el primer agente debería declarar las skills compartidas: %v", first.Files)
	}
	if n := underShared(second.Files); n != 0 {
		t.Errorf("el segundo agente no debe repetir %d ficheros de la carpeta compartida", n)
	}
}
