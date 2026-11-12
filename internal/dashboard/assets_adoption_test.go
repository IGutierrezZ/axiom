package dashboard

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// REQ-30.27: el panel de adopción que aparece tras el alta se calcula en un módulo puro de
// js/core/ (se ejecuta con Node) y la UI solo lo muestra: sin ejecutar nada, sin CDN externos.
func TestAdoptionAssetPanelModelRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"projectform"},
		"import { adoptionPanel } from './projectform.mjs';\n"+
			"const full = { adoption: { source_count: 1, unit_count: 3, sources: [{ path: 'docs/requisitos.md', kind: 'markdown', units: 3 }],\n"+
			"  context_candidates: [{ path: 'ARCHITECTURE.md', mode: 'maintained', reason: 'documento de arquitectura' }, { path: 'adr/', mode: 'reference', reason: 'decisiones' }],\n"+
			"  warnings: ['aviso'] }, adoption_instruction: 'Adopta los requisitos' };\n"+
			"const onlyContext = { adoption: { source_count: 0, unit_count: 0, sources: [], context_candidates: [{ path: 'ARCHITECTURE.md', mode: 'maintained', reason: 'x' }], warnings: [] }, adoption_instruction: 'no debe salir' };\n"+
			"console.log(JSON.stringify({\n"+
			"  full: adoptionPanel(full),\n"+
			"  onlyContext: adoptionPanel(onlyContext),\n"+
			"  none: adoptionPanel({}),\n"+
			"  nullResponse: adoptionPanel(null),\n"+
			"  empty: adoptionPanel({ adoption: { source_count: 0, unit_count: 0, sources: [], context_candidates: [], warnings: [] }, adoption_instruction: 'no debe salir' }),\n"+
			"}));\n")
	var res struct {
		Full struct {
			SourceCount int `json:"sourceCount"`
			UnitCount   int `json:"unitCount"`
			Sources     []struct {
				Path  string `json:"path"`
				Units int    `json:"units"`
			} `json:"sources"`
			Candidates []struct {
				Path      string `json:"path"`
				ModeLabel string `json:"modeLabel"`
			} `json:"candidates"`
			Warnings    []string `json:"warnings"`
			Instruction string   `json:"instruction"`
		} `json:"full"`
		OnlyContext *struct {
			Sources     []any  `json:"sources"`
			Instruction string `json:"instruction"`
		} `json:"onlyContext"`
		None         any `json:"none"`
		NullResponse any `json:"nullResponse"`
		Empty        any `json:"empty"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if res.Full.SourceCount != 1 || res.Full.UnitCount != 3 || len(res.Full.Sources) != 1 || res.Full.Sources[0].Path != "docs/requisitos.md" || res.Full.Sources[0].Units != 3 {
		t.Errorf("modelo de orígenes = %+v", res.Full)
	}
	if len(res.Full.Candidates) != 2 || res.Full.Candidates[0].ModeLabel != "mantenida" || res.Full.Candidates[1].ModeLabel != "de referencia" {
		t.Errorf("candidatos = %+v", res.Full.Candidates)
	}
	if res.Full.Instruction != "Adopta los requisitos" || len(res.Full.Warnings) != 1 {
		t.Errorf("instrucción o avisos = %q %v", res.Full.Instruction, res.Full.Warnings)
	}
	if res.OnlyContext == nil || len(res.OnlyContext.Sources) != 0 || res.OnlyContext.Instruction != "" {
		t.Errorf("sin orígenes no se ofrece la instrucción: %+v", res.OnlyContext)
	}
	if res.None != nil || res.NullResponse != nil || res.Empty != nil {
		t.Errorf("un informe ausente o vacío no muestra panel: %v %v %v", res.None, res.NullResponse, res.Empty)
	}
}

func TestAdoptionAssetPanelIsWiredAfterTheProjectInit(t *testing.T) {
	html, projects := readAsset(t, "index.html"), readAsset(t, "js/projects.js")

	dialog := html[strings.Index(html, `id="modal-adoption"`):]
	dialog = dialog[:strings.Index(dialog, "</dialog>")]
	for _, marker := range []string{`aria-labelledby="adoption-title"`, `id="adoption-body"`, `id="btn-close-adoption"`, "data-close"} {
		if !strings.Contains(dialog, marker) {
			t.Errorf("el diálogo de adopción debe contener %q", marker)
		}
	}

	for _, marker := range []string{"adoptionPanel(response)", "showAdoptionPanel(", "data-copy=", "Copiar la instrucción", "Axiom no ejecuta nada"} {
		if !strings.Contains(projects, marker) {
			t.Errorf("projects.js debe contener %q", marker)
		}
	}
	submit := regexp.MustCompile(`(?s)api\('/api/projects/init',.*?adoptionPanel\(response\)`).FindString(projects)
	if submit == "" {
		t.Error("el panel debe mostrarse tras la respuesta del alta (después de api('/api/projects/init', …))")
	}

	// Solo muestra el informe: no hay ninguna petición que lance la adopción ni crea el cambio.
	panel := projects[strings.Index(projects, "function showAdoptionPanel"):]
	panel = panel[:strings.Index(panel, "\n}\n")]
	for _, forbidden := range []string{"api(", "fetch(", "adopt scan", "XMLHttpRequest"} {
		if strings.Contains(panel, forbidden) {
			t.Errorf("showAdoptionPanel no ejecuta nada: contiene %q", forbidden)
		}
	}
	// Sin CDN ni recursos externos en el diálogo ni en el panel.
	if strings.Contains(dialog, "http://") || strings.Contains(dialog, "https://") || strings.Contains(panel, "http://") || strings.Contains(panel, "https://") {
		t.Error("el panel de adopción no puede cargar recursos externos")
	}
}
