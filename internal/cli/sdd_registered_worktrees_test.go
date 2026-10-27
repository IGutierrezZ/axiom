package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// Arnés de S9-c4 (REQ-25.44, design D-16): `axiom sdd status x --json --cwd <checkout principal>` y
// `axiom sdd continue x --cwd <checkout principal>` con el cambio solo en su worktree registrado. Entra por
// RunSDDStatus y RunSDDContinue, los puntos de entrada de la CLI, con el checkout principal y el worktree
// montados solo con ficheros y el registro escrito con el almacén real; sin registro el cambio no existe
// desde el checkout principal, con él se resuelve en el worktree.

// registeredWorktreeProject monta el checkout principal (sin ningún cambio, en `main`) y un worktree
// enlazado con el cambio x sellado y en su rama, y devuelve sus raíces y el common dir del principal.
func registeredWorktreeProject(t *testing.T) (main, worktree, common string) {
	t.Helper()
	main = evalSymlinks(t, t.TempDir())
	wrongBranchSwitch(t, main, "main")
	worktree = evalSymlinks(t, t.TempDir())
	gitDir := filepath.Join(worktree, ".gitdir")
	writeSDDStatusFile(t, filepath.Join(worktree, ".git"), "gitdir: "+gitDir+"\n")
	writeSDDStatusFile(t, filepath.Join(gitDir, "commondir"), filepath.Join(main, ".git")+"\n")
	writeSDDStatusFile(t, filepath.Join(gitDir, "HEAD"), "ref: refs/heads/feat/x\n")
	wrongBranchSealed(t, worktree, "- [ ] 1.1 Trabajo de ejemplo\n", false)
	common, err := gitcommon.CommonDirFromFiles(main)
	if err != nil {
		t.Fatal(err)
	}
	return main, worktree, common
}

func evalSymlinks(t *testing.T, dir string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func registerWorktree(t *testing.T, main, common, worktree string) {
	t.Helper()
	record := branch.Record{
		Schema: branch.RegistrySchema, Lane: branch.LaneSDD, Change: wrongBranchChange,
		Mode: branch.ModeWorktree, ModeSource: branch.ModeSourceDefault, PlanningRepo: main,
		Repositories: []branch.RecordRepo{{
			Name: filepath.Base(main), GitRoot: main, CommonDir: common, Role: branch.RepoChangeBranch,
			Branches: []branch.RecordBranch{{Name: "feat/x", Kind: branch.KindChange, Worktree: worktree}},
		}},
	}
	if err := registry.New().Save(common, record); err != nil {
		t.Fatal(err)
	}
}

func runStatus(t *testing.T, run func([]string, *bytes.Buffer) error, args ...string) sddstatus.Status {
	t.Helper()
	var stdout bytes.Buffer
	if err := run(args, &stdout); err != nil {
		t.Fatalf("la orden %v falló: %v", args, err)
	}
	var status sddstatus.Status
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("decodificar el status: %v\n%s", err, stdout.String())
	}
	return status
}

func statusRunner(t *testing.T, name string) func([]string, *bytes.Buffer) error {
	t.Helper()
	switch name {
	case "status":
		return func(args []string, out *bytes.Buffer) error { return RunSDDStatus(args, out) }
	default:
		return func(args []string, out *bytes.Buffer) error { return RunSDDContinue(args, out) }
	}
}

func wantWorktreeResolution(t *testing.T, label string, status sddstatus.Status, main, worktree string) {
	t.Helper()
	wantChangeRoot := filepath.Join(worktree, "openspec", "changes", wrongBranchChange)
	if status.ChangeRoot == nil || *status.ChangeRoot != wantChangeRoot {
		t.Errorf("%s: changeRoot = %v, se esperaba %s", label, status.ChangeRoot, wantChangeRoot)
	}
	if status.ActionContext.WorkspaceRoot != worktree || status.PlanningHome.Path != filepath.Join(worktree, "openspec") {
		t.Errorf("%s: workspaceRoot %s y planningHome %s, se esperaba el worktree %s", label, status.ActionContext.WorkspaceRoot, status.PlanningHome.Path, worktree)
	}
	if status.ApplyState != sddstatus.ApplyReady || status.NextRecommended != "apply" || slices.ContainsFunc(status.BlockedReasons, func(reason string) bool {
		return strings.Contains(reason, "wrong_branch") || strings.Contains(reason, "edit_authority_missing")
	}) {
		t.Errorf("%s: applyState %q, nextRecommended %q, motivos %v; se esperaba ready y apply sin bloqueos", label, status.ApplyState, status.NextRecommended, status.BlockedReasons)
	}
	if _, err := os.Stat(filepath.Join(main, "openspec", "changes", wrongBranchChange)); err == nil {
		t.Errorf("%s: el checkout principal no debe ganar una carpeta del cambio", label)
	}
}

func TestSDDRegisteredWorktreeStatusAndContinueHarness(t *testing.T) {
	main, worktree, common := registeredWorktreeProject(t)

	before := runStatus(t, statusRunner(t, "status"), wrongBranchChange, "--cwd", main, "--json")
	if before.ChangeRoot != nil || before.NextRecommended != "sdd-new" {
		t.Fatalf("sin registro el cambio no existe desde el checkout principal: changeRoot %v, nextRecommended %q", before.ChangeRoot, before.NextRecommended)
	}

	registerWorktree(t, main, common, worktree)

	for _, command := range []string{"status", "continue"} {
		run := statusRunner(t, command)
		named := runStatus(t, run, wrongBranchChange, "--cwd", main, "--json")
		wantWorktreeResolution(t, command+" x --cwd <principal>", named, main, worktree)
		unnamed := runStatus(t, run, "--cwd", main, "--json")
		wantWorktreeResolution(t, command+" --cwd <principal>", unnamed, main, worktree)
		inside := runStatus(t, run, wrongBranchChange, "--cwd", worktree, "--json")
		wantWorktreeResolution(t, command+" x --cwd <worktree>", inside, main, worktree)
		t.Logf("axiom sdd %s x --json --cwd <principal>: changeRoot = %s, nextRecommended = %s", command, *named.ChangeRoot, named.NextRecommended)
	}
}
