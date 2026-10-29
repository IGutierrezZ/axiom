package engine

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func TestPlanningIntegrationResolvesTheRepositoryPolicy(t *testing.T) {
	master := t.TempDir()
	specs := filepath.Join(master, "repo-specs")
	tests := []struct {
		name string
		loc  workspace.Location
		want string
	}{
		{
			name: "sin configuración rige el valor por defecto de la política",
			loc:  workspace.Location{Master: master, WorkspaceRoot: master},
			want: workspace.DefaultIntegrationBranch,
		},
		{
			name: "git.integration_branch del workspace",
			loc: workspace.Location{Master: master, WorkspaceRoot: master,
				Config: &workspace.WorkspaceConfig{Git: &workspace.GitConfig{IntegrationBranch: "develop"}}},
			want: "develop",
		},
		{
			name: "la entrada de repositories del repositorio de specs manda sobre el workspace",
			loc: workspace.Location{Master: master, WorkspaceRoot: specs,
				Config: &workspace.WorkspaceConfig{
					Git:          &workspace.GitConfig{IntegrationBranch: "develop"},
					Repositories: []workspace.RepositoryPolicy{{Path: "repo-specs", IntegrationBranch: "release"}},
				}},
			want: "release",
		},
		{
			name: "fuera de la carpeta maestra solo rige el workspace",
			loc: workspace.Location{Master: master, WorkspaceRoot: t.TempDir(),
				Config: &workspace.WorkspaceConfig{Git: &workspace.GitConfig{IntegrationBranch: "develop"}}},
			want: "develop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlanningIntegration(tt.loc); got != tt.want {
				t.Fatalf("PlanningIntegration() = %q, se esperaba %q", got, tt.want)
			}
		})
	}
}
