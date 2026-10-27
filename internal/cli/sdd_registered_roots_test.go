package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// Arnés de S9-c3 (REQ-25.43, REQ-24.9, design D-14): `axiom sdd status x --json` con tareas que nombran
// el worktree que `axiom branch ensure` registró para el cambio. Entra por RunSDDStatus, el punto de
// entrada de la CLI, con el repositorio montado solo con ficheros y el registro escrito con el almacén
// real; sin él, el worktree exige concesión (`blocked(edit_authority_missing)`), con él no.
func TestSDDRegisteredRootsStatusHarness(t *testing.T) {
	worktree, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSDDStatusFile(t, filepath.Join(worktree, ".git"), "gitdir: "+filepath.Join(worktree, ".gitdir")+"\n")
	task := "- [ ] 1.1 Editar `" + filepath.Join(worktree, "internal", "api", "handler.go") + "`\n"
	root, _ := wrongBranchProject(t, "feat/x", task, false)

	before := wrongBranchStatus(t, RunSDDStatus, root)
	if !slices.ContainsFunc(before.BlockedReasons, func(reason string) bool { return strings.Contains(reason, "blocked(edit_authority_missing)") }) {
		t.Fatalf("sin registro el worktree exige concesión; blockedReasons = %v", before.BlockedReasons)
	}

	commonDir, err := gitcommon.CommonDirFromFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	record := branch.Record{
		Schema: branch.RegistrySchema, Lane: branch.LaneSDD, Change: wrongBranchChange,
		Mode: branch.ModeWorktree, ModeSource: branch.ModeSourceDefault, PlanningRepo: root,
		Repositories: []branch.RecordRepo{{
			Name: filepath.Base(root), GitRoot: root, CommonDir: commonDir, Role: branch.RepoChangeBranch,
			Branches: []branch.RecordBranch{{Name: "feat/x", Kind: branch.KindChange, Worktree: worktree}},
		}},
	}
	if err := registry.New().Save(commonDir, record); err != nil {
		t.Fatal(err)
	}

	after := wrongBranchStatus(t, RunSDDStatus, root)
	if slices.ContainsFunc(after.BlockedReasons, func(reason string) bool { return strings.Contains(reason, "edit_authority_missing") }) {
		t.Errorf("con el worktree registrado no hay edit_authority_missing; blockedReasons = %v", after.BlockedReasons)
	}
	if after.ApplyState != sddstatus.ApplyReady || after.NextRecommended != "apply" {
		t.Errorf("applyState %q y nextRecommended %q, se esperaba ready y apply", after.ApplyState, after.NextRecommended)
	}
	if !slices.Contains(after.ActionContext.AllowedEditRoots, worktree) {
		t.Errorf("allowedEditRoots = %v debe incluir el worktree registrado %s", after.ActionContext.AllowedEditRoots, worktree)
	}
	t.Logf("sdd status x --json con el worktree registrado: allowedEditRoots = %v", after.ActionContext.AllowedEditRoots)
}
