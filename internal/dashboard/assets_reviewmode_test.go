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

// RDD está suspendido (C-01-c): ni el modal de alta ni la pantalla de ecosistema ofrecen
// el selector del modo de revisión, y nada del frontend referencia el módulo puro (queda
// latente hasta C-01-e).
func TestReviewModeUIIsAbsent(t *testing.T) {
	html, projects, eco := readAsset(t, "index.html"), readAsset(t, "js/projects.js"), readAsset(t, "js/screens/ecosystem.js")
	for _, marker := range []string{
		"select-project-review-mode", "form-review-mode", "select-review-mode", "review-mode-effective",
		"btn-save-review-mode", "eco-review-state", "eco-rdd-h", "Revisión (RDD)",
	} {
		if strings.Contains(html, marker) {
			t.Errorf("index.html no debe contener %q", marker)
		}
	}
	for _, marker := range []string{"select-project-review-mode", "review_mode", "review_mode_error"} {
		if strings.Contains(projects, marker) {
			t.Errorf("projects.js no debe contener %q", marker)
		}
	}
	for _, marker := range []string{"/api/project/review-mode", "describeReviewMode", "loadReviewMode", "saveReviewMode", "reviewmode.js"} {
		if strings.Contains(eco, marker) {
			t.Errorf("ecosystem.js no debe contener %q", marker)
		}
	}
}
