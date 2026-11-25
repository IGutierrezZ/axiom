package mcpexclusion

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// repoWithWorktree crea un repositorio git temporal con un worktree enlazado.
func repoWithWorktree(t *testing.T) (repo, worktree string) {
	t.Helper()
	repo = t.TempDir()
	git(t, repo, "init", "-q")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "inicial")
	worktree = filepath.Join(t.TempDir(), "wt")
	git(t, repo, "worktree", "add", "-q", "-b", "rama", worktree)
	return repo, worktree
}

func denied(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(claude.LocalSettingsPath(dir))
	if err != nil {
		t.Fatalf("falta settings.local.json en %s: %v", dir, err)
	}
	return string(raw)
}

func TestApplyWritesRootAndWorktreesAndReportsUnsupported(t *testing.T) {
	repo, worktree := repoWithWorktree(t)
	record := &hub.WorkspaceRecord{Path: repo}
	current := map[string][]string{
		"claude-code": {"work-copilot"},
		"codex":       {"otro"},
	}

	report := Apply(record, nil, current)

	if len(report.Targets) != 2 {
		t.Fatalf("destinos = %+v, se esperaban raíz y worktree", report.Targets)
	}
	for _, target := range report.Targets {
		if target.Status != StatusWritten {
			t.Errorf("destino %s: %s %s", target.Dir, target.Status, target.Reason)
		}
	}
	for _, dir := range []string{repo, worktree} {
		if !strings.Contains(denied(t, dir), `"serverName": "work-copilot"`) {
			t.Errorf("%s no recibió la exclusión", dir)
		}
	}
	if len(report.Unsupported) != 1 || report.Unsupported[0] != "codex" {
		t.Errorf("no soportados = %v", report.Unsupported)
	}
	if len(report.Warnings()) != 1 || !strings.Contains(report.Warnings()[0], "no soportado") {
		t.Errorf("avisos = %v", report.Warnings())
	}

	again := Apply(record, nil, current)
	for _, target := range again.Targets {
		if target.Status != StatusUnchanged {
			t.Errorf("segunda aplicación en %s: %s", target.Dir, target.Status)
		}
	}
}

func TestApplyRemovesOnlyPreviousNotInCurrent(t *testing.T) {
	repo, worktree := repoWithWorktree(t)
	record := &hub.WorkspaceRecord{Path: repo}
	Apply(record, nil, map[string][]string{"claude-code": {"a", "b"}})

	Apply(record, map[string][]string{"claude-code": {"a", "b"}}, map[string][]string{"claude-code": {"b"}})

	for _, dir := range []string{repo, worktree} {
		content := denied(t, dir)
		if strings.Contains(content, `"a"`) || !strings.Contains(content, `"b"`) {
			t.Errorf("%s: %s", dir, content)
		}
	}
}

func TestApplyReportsFailedTargetWithoutStoppingOthers(t *testing.T) {
	repo, worktree := repoWithWorktree(t)
	broken := claude.LocalSettingsPath(worktree)
	if err := os.MkdirAll(filepath.Dir(broken), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := Apply(&hub.WorkspaceRecord{Path: repo}, nil, map[string][]string{"claude-code": {"a"}})

	if len(report.Failed()) != 1 || len(report.Warnings()) != 1 {
		t.Fatalf("informe = %+v", report)
	}
	if !strings.Contains(denied(t, repo), `"a"`) {
		t.Errorf("la raíz debía recibir la exclusión")
	}
}

func TestApplyCurrentIsNoOpWithoutRecordOrPreference(t *testing.T) {
	mgr, err := hub.NewManager(filepath.Join(t.TempDir(), "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()

	report, err := ApplyCurrent(mgr, project)
	if err != nil || len(report.Targets) != 0 {
		t.Fatalf("proyecto sin registrar: %+v %v", report, err)
	}
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	if report, err = ApplyCurrent(mgr, project); err != nil || len(report.Targets) != 0 {
		t.Fatalf("sin preferencia: %+v %v", report, err)
	}
	if _, err := os.Stat(claude.LocalSettingsPath(project)); !os.IsNotExist(err) {
		t.Fatalf("no debía crearse nada: %v", err)
	}
}

func TestApplyCurrentAppliesStoredPreferenceToExtraDir(t *testing.T) {
	mgr, err := hub.NewManager(filepath.Join(t.TempDir(), "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	if _, err := mgr.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mgr.SetProjectMCPExclusions(project, map[string][]string{"claude-code": {"a"}}); err != nil {
		t.Fatal(err)
	}
	extra := t.TempDir()

	report, err := ApplyCurrent(mgr, project, extra)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Targets) != 2 {
		t.Fatalf("destinos = %+v", report.Targets)
	}
	for _, dir := range []string{project, extra} {
		if !strings.Contains(denied(t, dir), `"a"`) {
			t.Errorf("%s sin exclusión", dir)
		}
	}
}
