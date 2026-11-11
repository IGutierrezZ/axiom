package dashboard

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// G5.4: la lógica de la pantalla de modelos por fase (opciones, validación y parche) vive en
// un módulo puro de js/core/ y se ejecuta con Node.
const phaseModelsFixtureJS = `
const claude = {
  id: 'claude-code', name: 'Claude Code',
  phases: ['sdd-apply', 'sdd-archive', 'default'],
  assignments: { 'sdd-apply': { model: 'opus', effort: 'high' } },
  defaults: { 'sdd-archive': { model: 'haiku' } },
  options: { models: [
    { id: 'opus', efforts: ['low', 'high', 'max'] },
    { id: 'haiku', efforts: [] }
  ] }
};
const opencode = {
  id: 'opencode', name: 'OpenCode',
  phases: ['sdd-apply', 'sdd-spec'],
  assignments: { 'sdd-apply': { model: 'anthropic/claude-sonnet-4', effort: 'high' } },
  options: { providers: [
    { id: 'anthropic', name: 'Anthropic', models: [{ id: 'claude-sonnet-4', name: 'Claude Sonnet 4', variants: ['low', 'high'] }] },
    { id: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5', name: 'GPT-5', variants: [] }] }
  ] }
};
const opencodeNoCatalog = { ...opencode, options: { catalog_error: 'unavailable' } };
const copilot = {
  id: 'copilot', name: 'GitHub Copilot',
  phases: ['sdd-apply', 'sdd-spec'],
  assignments: {},
  options: { suggestions: ['GPT-5.2 (copilot)'], free_text: true }
};
`

func TestPhaseModelsLogicRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"phasemodels"},
		"import * as pm from './phasemodels.mjs';\n"+phaseModelsFixtureJS+
			"console.log(JSON.stringify({\n"+
			"  split: [pm.splitModelSpec('anthropic/claude'), pm.splitModelSpec('a:b'), pm.splitModelSpec('sinbarra'), pm.splitModelSpec('/x'), pm.splitModelSpec('x/'), pm.splitModelSpec(null)],\n"+
			"  claudeInput: pm.modelInput(claude),\n"+
			"  ocInput: pm.modelInput(opencode),\n"+
			"  ocTextInput: pm.modelInput(opencodeNoCatalog),\n"+
			"  cpInput: pm.modelInput(copilot),\n"+
			"  eff: [pm.effortInput(claude, 'opus'), pm.effortInput(claude, 'haiku'), pm.effortInput(claude, ''), pm.effortInput(claude, 'raro'),\n"+
			"        pm.effortInput(opencode, 'anthropic/claude-sonnet-4'), pm.effortInput(opencode, 'openai/gpt-5'), pm.effortInput(opencode, 'nube/x'),\n"+
			"        pm.effortInput(opencodeNoCatalog, 'anthropic/claude-sonnet-4'), pm.effortInput(copilot, 'GPT-5.2 (copilot)'), pm.effortInput(copilot, '  ')],\n"+
			"  labels: [pm.defaultLabel(claude, 'sdd-archive'), pm.defaultLabel(claude, 'sdd-apply'), pm.defaultLabel(copilot, 'sdd-apply'), pm.defaultLabel(opencode, 'sdd-apply')],\n"+
			"  rows: pm.rowsFor(claude),\n"+
			"}));\n")
	var res struct {
		Split       []*struct{ Provider, Model string } `json:"split"`
		ClaudeInput struct {
			Kind    string
			Options []struct{ Value, Label string }
		} `json:"claudeInput"`
		OcInput struct {
			Kind   string
			Groups []struct {
				Label   string
				Options []struct{ Value, Label string }
			}
		} `json:"ocInput"`
		OcTextInput struct{ Kind string } `json:"ocTextInput"`
		CpInput     struct {
			Kind        string
			Suggestions []string
		} `json:"cpInput"`
		Eff []struct {
			Kind   string
			Values []string
		} `json:"eff"`
		Labels []string `json:"labels"`
		Rows   []struct{ Phase, Model, Effort string }
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if res.Split[0] == nil || res.Split[0].Provider != "anthropic" || res.Split[0].Model != "claude" || res.Split[1] == nil || res.Split[1].Model != "b" {
		t.Errorf("splitModelSpec válido = %+v %+v", res.Split[0], res.Split[1])
	}
	for i := 2; i < 6; i++ {
		if res.Split[i] != nil {
			t.Errorf("splitModelSpec caso %d debe ser null: %+v", i, res.Split[i])
		}
	}
	if res.ClaudeInput.Kind != "select" || len(res.ClaudeInput.Options) != 2 || res.ClaudeInput.Options[0].Value != "opus" {
		t.Errorf("modelInput(claude) = %+v", res.ClaudeInput)
	}
	if res.OcInput.Kind != "grouped" || len(res.OcInput.Groups) != 2 || res.OcInput.Groups[0].Label != "Anthropic" || res.OcInput.Groups[0].Options[0].Value != "anthropic/claude-sonnet-4" || res.OcInput.Groups[0].Options[0].Label != "Claude Sonnet 4" {
		t.Errorf("modelInput(opencode) = %+v", res.OcInput)
	}
	if res.OcTextInput.Kind != "text" || res.CpInput.Kind != "text" || len(res.CpInput.Suggestions) != 1 {
		t.Errorf("sin catálogo y Copilot son texto libre: %+v %+v", res.OcTextInput, res.CpInput)
	}
	wantEff := []struct {
		kind   string
		values string
	}{
		{"select", "low,high,max"}, {"select", ""}, {"none", ""}, {"select", ""},
		{"select", "low,high"}, {"select", ""}, {"select", ""},
		{"text", ""}, {"text", ""}, {"none", ""},
	}
	for i, want := range wantEff {
		if got := res.Eff[i]; got.Kind != want.kind || strings.Join(got.Values, ",") != want.values {
			t.Errorf("effortInput caso %d = %+v, quiero %+v", i, got, want)
		}
	}
	if !strings.Contains(res.Labels[0], "haiku") || !strings.Contains(res.Labels[1], "sin asignar") || !strings.Contains(res.Labels[2], "conversación") || !strings.Contains(res.Labels[3], "hereda") {
		t.Errorf("defaultLabel = %q", res.Labels)
	}
	if len(res.Rows) != 3 || res.Rows[0].Model != "opus" || res.Rows[0].Effort != "high" || res.Rows[1].Model != "" {
		t.Errorf("rowsFor = %+v", res.Rows)
	}
}

func TestPhaseModelsValidationAndPatchRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"phasemodels"},
		"import * as pm from './phasemodels.mjs';\n"+phaseModelsFixtureJS+
			"const unchanged = pm.rowsFor(claude);\n"+
			"console.log(JSON.stringify({\n"+
			"  ok: pm.validateRows(claude, [{ phase: 'sdd-apply', model: 'opus', effort: 'max' }, { phase: 'default', model: '', effort: '' }]),\n"+
			"  claudeBad: pm.validateRows(claude, [\n"+
			"    { phase: 'a', model: 'gpt' }, { phase: 'b', model: 'haiku', effort: 'high' }, { phase: 'c', model: 'opus', effort: 'ultra' }, { phase: 'd', model: '', effort: 'high' },\n"+
			"    { phase: 'e', model: 'opus\\nx: y' }]),\n"+
			"  ocBad: pm.validateRows(opencode, [\n"+
			"    { phase: 'a', model: 'claude' }, { phase: 'b', model: 'nube/x' }, { phase: 'c', model: 'anthropic/claude-sonnet-4', effort: 'ultra' }, { phase: 'd', model: 'openai/gpt-5', effort: 'low' }]),\n"+
			"  ocNoCatalogOk: pm.validateRows(opencodeNoCatalog, [{ phase: 'a', model: 'nube/x', effort: 'cualquiera' }]),\n"+
			"  ocNoCatalogBad: pm.validateRows(opencodeNoCatalog, [{ phase: 'a', model: 'sin-barra' }]),\n"+
			"  cpOk: pm.validateRows(copilot, [{ phase: 'sdd-apply', model: 'Claude Sonnet 4.6 (copilot)', effort: 'lo-que-sea' }]),\n"+
			"  cpBad: pm.validateRows(copilot, [{ phase: 'sdd-apply', model: 'GPT\\ntools: []' }]),\n"+
			"  none: pm.buildPatch(claude, unchanged),\n"+
			"  edit: pm.buildPatch(claude, [{ phase: 'sdd-apply', model: ' opus ', effort: 'low' }, { phase: 'sdd-archive', model: 'haiku', effort: '' }, { phase: 'default', model: '', effort: '' }]),\n"+
			"  clear: pm.buildPatch(claude, [{ phase: 'sdd-apply', model: '', effort: '' }, { phase: 'sdd-archive', model: '', effort: '' }]),\n"+
			"  dropEffort: pm.buildPatch(claude, [{ phase: 'sdd-apply', model: 'opus', effort: '' }]),\n"+
			"  invalid: pm.buildPatch(claude, [{ phase: 'sdd-apply', model: 'gpt', effort: '' }]),\n"+
			"  saved: [pm.savedMessage({ regenerated: ['a.md'] }), pm.savedMessage({ regenerated: ['a', 'b', 'c', 'd', 'e', 'f'] }), pm.savedMessage({ regenerated: [] }), pm.savedMessage(null)],\n"+
			"  merged: pm.withSavedAssignments(claude, { assignments: { x: { model: 'opus' } } }).assignments,\n"+
			"  mergedKeeps: pm.withSavedAssignments(claude, { assignments: { x: { model: 'opus' } } }).name,\n"+
			"  noChanges: pm.NO_CHANGES_MESSAGE,\n"+
			"}));\n")
	var res struct {
		Ok             []string `json:"ok"`
		ClaudeBad      []string `json:"claudeBad"`
		OcBad          []string `json:"ocBad"`
		OcNoCatalogOk  []string `json:"ocNoCatalogOk"`
		OcNoCatalogBad []string `json:"ocNoCatalogBad"`
		CpOk           []string `json:"cpOk"`
		CpBad          []string `json:"cpBad"`
		None           patchRes `json:"none"`
		Edit           patchRes `json:"edit"`
		Clear          patchRes `json:"clear"`
		DropEffort     patchRes `json:"dropEffort"`
		Invalid        patchRes `json:"invalid"`
		Saved          []string `json:"saved"`
		Merged         map[string]struct{ Model string }
		MergedKeeps    string `json:"mergedKeeps"`
		NoChanges      string `json:"noChanges"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if len(res.Ok) != 0 || len(res.OcNoCatalogOk) != 0 || len(res.CpOk) != 0 {
		t.Errorf("filas válidas rechazadas: %v %v %v", res.Ok, res.OcNoCatalogOk, res.CpOk)
	}
	wantContains(t, "Claude", res.ClaudeBad, []string{"a: el modelo «gpt»", "b: haiku no admite esfuerzo", "c: el esfuerzo «ultra»", "d: elige un modelo", "e: el modelo y el esfuerzo no pueden"})
	wantContains(t, "OpenCode", res.OcBad, []string{"a: el modelo «claude» debe tener el formato", "b: el modelo «nube/x» no está en el catálogo", "c: la variante «ultra»", "d: openai/gpt-5 no admite variantes"})
	wantContains(t, "OpenCode sin catálogo", res.OcNoCatalogBad, []string{"proveedor/modelo"})
	wantContains(t, "Copilot", res.CpBad, []string{"caracteres de control"})

	if len(res.None.Changed) != 0 || len(res.None.Violations) != 0 || res.None.Body.Adapter != "claude-code" {
		t.Errorf("sin cambios: %+v", res.None)
	}
	if strings.Join(res.Edit.Changed, ",") != "sdd-apply,sdd-archive" || res.Edit.Body.Assignments["sdd-apply"].Model != "opus" || res.Edit.Body.Assignments["sdd-apply"].Effort != "low" || res.Edit.Body.Assignments["sdd-archive"].Model != "haiku" {
		t.Errorf("edición: solo cambian las fases editadas y se recorta el texto: %+v", res.Edit)
	}
	if strings.Join(res.Clear.Changed, ",") != "sdd-apply" || res.Clear.Body.Assignments["sdd-apply"].Model != "" {
		t.Errorf("quitar: solo la fase que tenía asignación, con modelo vacío: %+v", res.Clear)
	}
	if strings.Join(res.DropEffort.Changed, ",") != "sdd-apply" || res.DropEffort.Body.Assignments["sdd-apply"].Effort != "" || res.DropEffort.Body.Assignments["sdd-apply"].Model != "opus" {
		t.Errorf("quitar solo el esfuerzo debe enviar el modelo con esfuerzo vacío: %+v", res.DropEffort)
	}
	if len(res.Invalid.Violations) != 1 {
		t.Errorf("un parche con violaciones debe llevarlas: %+v", res.Invalid)
	}
	if !strings.Contains(res.Saved[0], "1 fichero regenerado: a.md") || !strings.Contains(res.Saved[1], "6 ficheros regenerados") || !strings.Contains(res.Saved[1], "y 2 más") ||
		!strings.Contains(res.Saved[2], "ningún fichero") || !strings.Contains(res.Saved[3], "ningún fichero") {
		t.Errorf("savedMessage = %q", res.Saved)
	}
	if res.Merged["x"].Model != "opus" || res.MergedKeeps != "Claude Code" || res.NoChanges == "" {
		t.Errorf("withSavedAssignments = %+v %q", res.Merged, res.MergedKeeps)
	}
}

type patchRes struct {
	Body struct {
		Adapter     string
		Assignments map[string]struct{ Model, Effort string }
	}
	Changed    []string
	Violations []string
}

func wantContains(t *testing.T, name string, got, wants []string) {
	t.Helper()
	joined := strings.Join(got, "\n")
	for _, want := range wants {
		if !strings.Contains(joined, want) {
			t.Errorf("%s: faltan %q en:\n%s", name, want, joined)
		}
	}
	if len(got) != len(wants) {
		t.Errorf("%s: hay %d violaciones, quiero %d:\n%s", name, len(got), len(wants), joined)
	}
}

// La pantalla pinta una tabla por adaptador, valida con el módulo puro antes de enviar y usa
// solo las rutas del propio servidor (sin CDN).
func TestPhaseModelsUIWiring(t *testing.T) {
	eco, screen, html, css := readAsset(t, "js/screens/ecosystem.js"), readAsset(t, "js/screens/ecosystem-models.js"), readAsset(t, "index.html"), readAsset(t, "style.css")
	for _, marker := range []string{"from './ecosystem-models.js'", "modelsHtml(data.adapters)", "bindModelsEditor("} {
		if !strings.Contains(eco, marker) {
			t.Errorf("ecosystem.js debe contener %q", marker)
		}
	}
	for _, marker := range []string{"/api/ecosystem/models", "buildPatch(", "effortInput(", "modelInput(", "violationsFromError(", "savedMessage(", "data-pm-save", "data-pm-errors", `role="status"`, `role="alert"`} {
		if !strings.Contains(screen, marker) {
			t.Errorf("ecosystem-models.js debe contener %q", marker)
		}
	}
	if !strings.Contains(html, `id="eco-models-body"`) {
		t.Error("index.html debe seguir teniendo #eco-models-body")
	}
	for _, class := range regexp.MustCompile(`class="(eco-[\w-]+)`).FindAllStringSubmatch(screen, -1) {
		if !strings.Contains(css, "."+class[1]) {
			t.Errorf("la clase %s se usa pero style.css no la define", class[1])
		}
	}
	if strings.Contains(screen+readAsset(t, "js/core/phasemodels.js"), "http://") || strings.Contains(screen, "https://") {
		t.Error("las pantallas no cargan nada de fuera: sin CDN")
	}
}
