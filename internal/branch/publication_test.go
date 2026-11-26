package branch

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/docpaths"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// DocPaths compone la lista cerrada de REQ-27.5 en un único punto: debe dar el mismo conjunto que
// `docpaths.FromWorkspace` con las rutas de `technical_context`, en monorepo y en multirepo.
func TestPublicationTargetDocPathsComposesTheClosedList(t *testing.T) {
	declarations := workspace.Declarations{TechnicalContext: &workspace.TechnicalContextConfig{
		Maintained: &workspace.ContextMode{Paths: []string{"docs/contexto", "../fuera"}},
		Reference:  &workspace.ContextMode{Paths: []string{"docs/contexto/ref"}},
	}}
	cases := []struct {
		name         string
		target       PublicationTarget
		declarations workspace.Declarations
		maintained   []string
		reference    []string
	}{
		{"monorepo embebido", PublicationTarget{Topology: workspace.TopologyMonorepoEmbedded, PlanningRel: "openspec"}, declarations, declarations.TechnicalContext.Maintained.Paths, declarations.TechnicalContext.Reference.Paths},
		{"multirepo con specs en la raíz", PublicationTarget{Topology: workspace.TopologyMultirepo, InPlace: true}, declarations, declarations.TechnicalContext.Maintained.Paths, declarations.TechnicalContext.Reference.Paths},
		{"sin technical_context", PublicationTarget{PlanningRel: "openspec"}, workspace.Declarations{}, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.target.DocPaths(tc.declarations, "x")
			if err != nil {
				t.Fatalf("DocPaths: %v", err)
			}
			want, err := docpaths.FromWorkspace(tc.target.PlanningRel, "x", tc.maintained, tc.reference)
			if err != nil {
				t.Fatalf("FromWorkspace: %v", err)
			}
			if !reflect.DeepEqual(got.Candidates(), want.Candidates()) {
				t.Errorf("Candidates() = %v, se esperaba %v", got.Candidates(), want.Candidates())
			}
			for _, rel := range []string{"requirements.txt", "docs/x.mdx", "docs/contexto/a.md", "docs/contexto/ref/b.md", tc.target.PlanningRel + "/changes/x/design.md"} {
				if got.Contains(rel) != want.Contains(rel) {
					t.Errorf("Contains(%q) = %v, se esperaba %v", rel, got.Contains(rel), want.Contains(rel))
				}
			}
		})
	}
}

func TestPublicationTargetDocPathsRejectsAnUnusableChangeName(t *testing.T) {
	if _, err := (PublicationTarget{PlanningRel: "openspec"}).DocPaths(workspace.Declarations{}, "../x"); err == nil {
		t.Fatal("DocPaths aceptó un nombre de cambio que sale de la carpeta")
	}
}
