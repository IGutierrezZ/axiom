package dashboard

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// La lógica de la pantalla de exclusiones MCP vive en un módulo puro de js/core/ y se
// ejecuta con Node.
func TestMCPExclusionsModuleRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"mcpexclusions"},
		"import { normalizeNames, addName, buildBody, reportLines, hasFailures } from './mcpexclusions.mjs';\n"+
			"const report = { targets: [{ dir: 'C:/p', status: 'written' }, { dir: 'C:/p-wt', status: 'failed', reason: 'JSON ilegible' }], unsupported: ['codex'] };\n"+
			"console.log(JSON.stringify({\n"+
			"  norm: normalizeNames([' work-copilot ', '', 'engram', 'engram', null, 'claude.ai Microsoft 365']),\n"+
			"  add: addName(['engram'], ['engram'], '  claude.ai Microsoft 365 '),\n"+
			"  addEmpty: addName(['b', 'a'], ['a'], '   '),\n"+
			"  body: buildBody({ 'claude-code': ['x'], codex: ['y'] }, ['z', ' z ', 'a']),\n"+
			"  bodyEmpty: buildBody(null, []),\n"+
			"  lines: reportLines(report),\n"+
			"  noLines: reportLines(null),\n"+
			"  failed: [hasFailures(report), hasFailures({ targets: [{ status: 'unchanged' }] }), hasFailures(null)],\n"+
			"}));\n")
	var res struct {
		Norm []string `json:"norm"`
		Add  struct {
			Candidates []string `json:"candidates"`
			Selected   []string `json:"selected"`
			Added      string   `json:"added"`
		} `json:"add"`
		AddEmpty struct {
			Candidates []string `json:"candidates"`
			Selected   []string `json:"selected"`
			Added      string   `json:"added"`
		} `json:"addEmpty"`
		Body      map[string]map[string][]string `json:"body"`
		BodyEmpty map[string]map[string][]string `json:"bodyEmpty"`
		Lines     []string                       `json:"lines"`
		NoLines   []string                       `json:"noLines"`
		Failed    []bool                         `json:"failed"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if want := []string{"claude.ai Microsoft 365", "engram", "work-copilot"}; !reflect.DeepEqual(res.Norm, want) {
		t.Errorf("normalizeNames = %v", res.Norm)
	}
	if res.Add.Added != "claude.ai Microsoft 365" || !reflect.DeepEqual(res.Add.Candidates, []string{"claude.ai Microsoft 365", "engram"}) || !reflect.DeepEqual(res.Add.Selected, []string{"claude.ai Microsoft 365", "engram"}) {
		t.Errorf("addName = %+v", res.Add)
	}
	if res.AddEmpty.Added != "" || !reflect.DeepEqual(res.AddEmpty.Candidates, []string{"a", "b"}) || !reflect.DeepEqual(res.AddEmpty.Selected, []string{"a"}) {
		t.Errorf("addName vacío = %+v", res.AddEmpty)
	}
	if got := res.Body["exclusions"]; !reflect.DeepEqual(got["claude-code"], []string{"a", "z"}) || !reflect.DeepEqual(got["codex"], []string{"y"}) {
		t.Errorf("buildBody = %v: debe sustituir claude-code y conservar codex", res.Body)
	}
	if got := res.BodyEmpty["exclusions"]; len(got["claude-code"]) != 0 || len(got) != 1 {
		t.Errorf("buildBody vacío = %v", res.BodyEmpty)
	}
	if len(res.Lines) != 3 || !strings.Contains(res.Lines[0], "escrito") || !strings.Contains(res.Lines[1], "falló (JSON ilegible)") || !strings.Contains(res.Lines[2], "codex") || len(res.NoLines) != 0 {
		t.Errorf("reportLines = %q, sin informe = %q", res.Lines, res.NoLines)
	}
	if !reflect.DeepEqual(res.Failed, []bool{true, false, false}) {
		t.Errorf("hasFailures = %v", res.Failed)
	}
}

// La sección vive en Ecosistema, junto a las preferencias de worktree, y su pantalla se
// carga y se inicializa desde ecosystem.js.
func TestMCPExclusionsSectionIsWiredIntoTheEcosystemScreen(t *testing.T) {
	eco, screen, html := readAsset(t, "js/screens/ecosystem.js"), readAsset(t, "js/screens/ecosystem-mcp.js"), readAsset(t, "index.html")
	for _, marker := range []string{"from './ecosystem-mcp.js'", "loadMcpExclusions()", "initMcpExclusions()"} {
		if !strings.Contains(eco, marker) {
			t.Errorf("ecosystem.js debe contener %q", marker)
		}
	}
	for _, marker := range []string{"/api/projects/active/mcp-exclusions", "from '../core/mcpexclusions.js'", "buildBody("} {
		if !strings.Contains(screen, marker) {
			t.Errorf("ecosystem-mcp.js debe contener %q", marker)
		}
	}
	for _, marker := range []string{"Servidores MCP excluidos en este proyecto", `id="form-mcp-exclusions"`, `id="mcp-excl-list"`, `id="input-mcp-excl-name"`, "claude.ai Microsoft 365", "comprueba en <code>/mcp</code>", "sesiones nuevas"} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html debe contener %q", marker)
		}
	}
}
