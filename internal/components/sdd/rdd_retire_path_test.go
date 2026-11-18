package sdd

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
)

// Con RDD suspendido, la verificación debe tratar como retirados los agentes de
// revisión del directorio de subagentes, y solo esos.
func TestIsRetiredReviewSubAgentPath(t *testing.T) {
	agentsDir := filepath.Join("repo", ".claude", "agents")
	tests := []struct {
		name string
		rdd  bool
		path string
		want bool
	}{
		{"md suspendido", false, filepath.Join(agentsDir, "review-risk.md"), true},
		{"yaml suspendido", false, filepath.Join(agentsDir, "review-refuter.yaml"), true},
		{"RDD activo", true, filepath.Join(agentsDir, "review-risk.md"), false},
		{"otro agente", false, filepath.Join(agentsDir, "sdd-apply.md"), false},
		{"prefijo compartido", false, filepath.Join(agentsDir, "review-risk-extra.md"), false},
		{"fuera de agents", false, filepath.Join("repo", ".claude", "skills", "review-risk.md"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(rddharness.SetForTesting(tt.rdd))
			if rddharness.Enabled() != tt.rdd {
				t.Skip("binario compilado con rdd_harness: el apagado no aplica")
			}
			if got := IsRetiredReviewSubAgentPath(tt.path); got != tt.want {
				t.Fatalf("IsRetiredReviewSubAgentPath(%q) = %v; quiere %v", tt.path, got, tt.want)
			}
		})
	}
}
