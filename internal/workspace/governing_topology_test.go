package workspace

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// RF-PRE-14: la topología que gobierna un directorio sale de Locate; un
// axiom.yaml directo no cargable conserva la resolución embebida, y sin
// axiom.yaml no hay topología que aplicar.
func TestLocationGoverningTopology(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(t *testing.T) string
		want   TopologyType
		wantOK bool
	}{
		{
			name: "multirepo cargado por puntero",
			setup: func(t *testing.T) string {
				return workspacetest.Canonical(t).Master
			},
			want: TopologyMultirepo, wantOK: true,
		},
		{
			name: "monorepo embebido cargado",
			setup: func(t *testing.T) string {
				return workspacetest.Embedded(t, ".").Master
			},
			want: TopologyMonorepoEmbedded, wantOK: true,
		},
		{
			name: "axiom.yaml directo sin topologia",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), "workspace: {name: x}\n")
				return dir
			},
			want: TopologyMonorepoEmbedded, wantOK: true,
		},
		{
			name: "axiom.yaml directo con topologia desconocida",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), retiredTopologyConfig("."))
				return dir
			},
			want: TopologyMonorepoEmbedded, wantOK: true,
		},
		{
			name: "axiom.yaml directo ilegible",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), "workspace: [sin cerrar\n")
				return dir
			},
			want: TopologyMonorepoEmbedded, wantOK: true,
		},
		{
			name: "solo openspec sin axiom.yaml",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				workspacetest.MkdirAll(t, filepath.Join(dir, "openspec"))
				return dir
			},
			want: "", wantOK: false,
		},
		{
			name: "sin marcadores",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			want: "", wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc := mustLocate(t, tc.setup(t))
			got, ok := loc.GoverningTopology()
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("GoverningTopology() = (%q, %v), se esperaba (%q, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}

	t.Run("puntero roto hace fallar Locate", func(t *testing.T) {
		ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
		if _, err := Locate(ws.Master); err == nil {
			t.Fatal("Locate con puntero roto debe fallar: no hay topología que consultar")
		}
	})
}
