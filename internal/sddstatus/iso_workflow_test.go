package sddstatus

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// REQ-29.1 y REQ-29.2: ante la contradicción `sdd.iso: true` con `workflow.lane:
// odd` el status no asume el carril odd, sino que pregunta (ask / invalid); un
// error ordinario de `sdd:` no cambia el carril del proyecto.
func TestProjectWorkflowISOContradiction(t *testing.T) {
	tests := []struct {
		name       string
		blocks     string
		wantLane   string
		wantSource string
	}{
		{"contradicción iso con lane odd", "sdd:\n  iso: true\nworkflow:\n  lane: odd\n", "ask", "invalid"},
		{"sdd inválido no cambia el carril declarado", "sdd:\n  iso: quizás\nworkflow:\n  lane: odd\n", "odd", "declared"},
		{"sdd inválido sin workflow no cambia el carril", "sdd:\n  archive: true\n", "sdd", "default"},
		{"iso true con lane sdd", "sdd:\n  iso: true\nworkflow:\n  lane: sdd\n", "sdd", "declared"},
		{"iso false con lane odd", "sdd:\n  iso: false\nworkflow:\n  lane: odd\n", "odd", "declared"},
		{"iso true sin workflow", "sdd:\n  iso: true\n", "sdd", "default"},
	}
	for _, tt := range tests {
		for _, change := range []string{"demo", ""} {
			name := tt.name + " con cambio"
			if change == "" {
				name = tt.name + " sin cambio"
			}
			t.Run(name, func(t *testing.T) {
				ws := embeddedWithDeclaration(t, tt.blocks)
				location, err := workspace.Locate(ws.Master)
				if err != nil {
					t.Fatal(err)
				}
				if got := projectWorkflow(location); got.Lane != tt.wantLane || got.Source != tt.wantSource {
					t.Errorf("projectWorkflow = %+v, se esperaba {%s %s}", got, tt.wantLane, tt.wantSource)
				}
				got := workflowOf(t, resolveFrom(t, ws.Master, change))
				if got.Lane != tt.wantLane || got.Source != tt.wantSource {
					t.Errorf("status.workflow = %+v, se esperaba {%s %s}", got, tt.wantLane, tt.wantSource)
				}
			})
		}
	}
}
