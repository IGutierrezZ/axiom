package dashboard

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// detailOf monta un cambio activo con los ficheros dados (ruta relativa -> contenido) y devuelve su
// detalle. Si seal no es nil, sella antes ese kickoff.
func detailOf(t *testing.T, files map[string]string, seal []kickoff.KickoffRole) *IncrementDetailDTO {
	t.Helper()
	ws := embeddedProject(t, hub.WorktreeModeInPlace)
	root := filepath.Join(ws.Home, "changes", "c")
	workspacetest.WriteFile(t, filepath.Join(root, "proposal.md"), "# c\n")
	for rel, content := range files {
		workspacetest.WriteFile(t, filepath.Join(root, filepath.FromSlash(rel)), content)
	}
	if seal != nil {
		_, ok, err := kickoff.Seal(root, kickoff.Kickoff{
			Schema: kickoff.KickoffSchemaV1, Change: "c", SealedBy: "test",
			Config: kickoff.FlowConfig{
				FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
				Roles: seal,
			},
			Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		})
		if err != nil || !ok {
			t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
		}
	}
	dto, err := NewService(ws.Master).GetIncrementDetail("c")
	if err != nil {
		t.Fatal(err)
	}
	return dto
}

func wantFiles(t *testing.T, label string, got []ArtifactFileDTO, want []ArtifactFileDTO) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %+v, se esperaba %+v", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %+v, se esperaba %+v", label, i, got[i], want[i])
		}
	}
}

// Las specs por dominio (specs/<dominio>/spec.md) aparecen todas y ordenadas.
func TestIncrementDetailListsDomainSpecs(t *testing.T) {
	dto := detailOf(t, map[string]string{
		"specs/b/spec.md": "# spec b\n",
		"specs/a/spec.md": "# spec a\n",
	}, nil)
	wantFiles(t, "Specs", dto.Specs, []ArtifactFileDTO{
		{Path: "specs/a/spec.md", Content: "# spec a\n"},
		{Path: "specs/b/spec.md", Content: "# spec b\n"},
	})
	if !dto.HasSpec || dto.HasTasks || dto.Tasks == nil {
		t.Errorf("HasSpec=%v HasTasks=%v Tasks=%v", dto.HasSpec, dto.HasTasks, dto.Tasks)
	}
}

// Con varios roles sellados, las tareas son los tasks.<rol>.md en el orden del kickoff.
func TestIncrementDetailListsPerRoleTasksInKickoffOrder(t *testing.T) {
	dto := detailOf(t, map[string]string{
		"tasks.core.md": "- [ ] 1.1 core\n",
		"tasks.qa.md":   "- [ ] 1.1 qa\n",
	}, []kickoff.KickoffRole{
		{Role: "core", GatePolicy: "blocking", TasksFile: "tasks.core.md", VerifyFile: "verify-report.core.md"},
		{Role: "qa", GatePolicy: "blocking", TasksFile: "tasks.qa.md", VerifyFile: "verify-report.qa.md"},
	})
	wantFiles(t, "Tasks", dto.Tasks, []ArtifactFileDTO{
		{Path: "tasks.core.md", Content: "- [ ] 1.1 core\n"},
		{Path: "tasks.qa.md", Content: "- [ ] 1.1 qa\n"},
	})
	if !dto.HasTasks {
		t.Error("HasTasks debía ser true")
	}
}

// Un cambio heredado con spec.md y tasks.md planos sigue mostrándose.
func TestIncrementDetailKeepsLegacyFlatFiles(t *testing.T) {
	dto := detailOf(t, map[string]string{"spec.md": "# spec\n", "tasks.md": "- [ ] 1.1 a\n"}, nil)
	wantFiles(t, "Specs", dto.Specs, []ArtifactFileDTO{{Path: "spec.md", Content: "# spec\n"}})
	wantFiles(t, "Tasks", dto.Tasks, []ArtifactFileDTO{{Path: "tasks.md", Content: "- [ ] 1.1 a\n"}})
	if !dto.HasSpec || !dto.HasTasks {
		t.Errorf("HasSpec=%v HasTasks=%v", dto.HasSpec, dto.HasTasks)
	}
}
