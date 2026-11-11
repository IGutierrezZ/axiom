package dashboard

import (
	"encoding/json"
	"strings"
	"testing"
)

// G4.5: los textos del modo de revisión viven en un módulo puro de js/core/ y se
// ejecutan con Node.
func TestReviewModeTextsRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"reviewmode"},
		"import { sourceLabel, describeReviewMode, selectValue, overrideNote } from './reviewmode.mjs';\n"+
			"console.log(JSON.stringify({\n"+
			"  labels: ['clone_local', 'project', 'global', 'default', 'otro', '', undefined].map(sourceLabel),\n"+
			"  on: describeReviewMode({ effective: 'on', source: 'project' }),\n"+
			"  off: describeReviewMode({ effective: 'off', source: 'default' }),\n"+
			"  none: describeReviewMode(null),\n"+
			"  select: [{ project: 'on' }, { project: 'off' }, { project: '' }, {}, null].map(selectValue),\n"+
			"  win: overrideNote({ clone_local: 'off', project: 'on' }),\n"+
			"  noNote: [overrideNote({ clone_local: 'off', project: '' }), overrideNote({ clone_local: '', project: 'on' }), overrideNote(null)],\n"+
			"}));\n")
	var res struct {
		Labels []string `json:"labels"`
		On     string   `json:"on"`
		Off    string   `json:"off"`
		None   string   `json:"none"`
		Select []string `json:"select"`
		Win    string   `json:"win"`
		NoNote []string `json:"noNote"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	for i, want := range []string{"clon", "proyecto", "global", "por defecto"} {
		if !strings.Contains(res.Labels[i], want) {
			t.Errorf("sourceLabel caso %d = %q, debe contener %q", i, res.Labels[i], want)
		}
	}
	if res.Labels[4] != "otro" || !strings.Contains(res.Labels[5], "por defecto") || !strings.Contains(res.Labels[6], "por defecto") {
		t.Errorf("fuentes desconocidas o vacías: %q", res.Labels[4:])
	}
	if !strings.Contains(res.On, "encendida") || !strings.Contains(res.On, "tu preferencia para este proyecto") {
		t.Errorf("describeReviewMode(on/project) = %q", res.On)
	}
	if !strings.Contains(res.Off, "apagada") || !strings.Contains(res.Off, "por defecto") || res.None != "" {
		t.Errorf("describeReviewMode(off/default) = %q, null = %q", res.Off, res.None)
	}
	if strings.Join(res.Select, ",") != "on,off,inherit,inherit,inherit" {
		t.Errorf("selectValue = %q", res.Select)
	}
	if !strings.Contains(res.Win, "gana") {
		t.Errorf("overrideNote = %q", res.Win)
	}
	for i, note := range res.NoNote {
		if note != "" {
			t.Errorf("overrideNote sin conflicto caso %d = %q", i, note)
		}
	}
}

// El modal de alta ofrece la preferencia y la pantalla de ecosistema la muestra y la cambia.
func TestReviewModeUIWiring(t *testing.T) {
	html, projects, eco := readAsset(t, "index.html"), readAsset(t, "js/projects.js"), readAsset(t, "js/screens/ecosystem.js")
	modal := html[strings.Index(html, `id="modal-add-project"`):]
	modal = modal[:strings.Index(modal, "</dialog>")]
	if !strings.Contains(modal, `id="select-project-review-mode"`) {
		t.Error("el modal de alta debe ofrecer la preferencia de revisión")
	}
	for _, marker := range []string{"select-project-review-mode", "review_mode", "review_mode_error"} {
		if !strings.Contains(projects, marker) {
			t.Errorf("projects.js debe contener %q", marker)
		}
	}
	for _, marker := range []string{`id="form-review-mode"`, `id="select-review-mode"`, `id="review-mode-effective"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html debe contener %q", marker)
		}
	}
	for _, marker := range []string{"/api/project/review-mode", "describeReviewMode(", "loadReviewMode", "saveReviewMode"} {
		if !strings.Contains(eco, marker) {
			t.Errorf("ecosystem.js debe contener %q", marker)
		}
	}
}
