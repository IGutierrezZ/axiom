package sddstatus

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// sealRolesKickoff seals a kickoff naming every role in order, each with the
// default tasks.<role>.md / verify-report.<role>.md pair, through the real
// kickoff.Seal API.
func sealRolesKickoff(t *testing.T, changeRoot, changeName string, style kickoff.ExecutionStyle, roles ...string) {
	t.Helper()
	sealed := make([]kickoff.KickoffRole, 0, len(roles))
	for _, role := range roles {
		sealed = append(sealed, kickoff.KickoffRole{
			Role: role, GatePolicy: "blocking",
			TasksFile: "tasks." + role + ".md", VerifyFile: "verify-report." + role + ".md",
		})
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: changeName, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: style, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone, Roles: sealed,
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("sealRolesKickoff: kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
}

// seedMultiRoleChange seeds a planned change whose plan lives only in the
// tasks.<role>.md files of the sealed roles (no shared tasks.md).
func seedMultiRoleChange(t *testing.T, root, name string, style kickoff.ExecutionStyle, tasksByRole map[string]string, roles ...string) string {
	t.Helper()
	changeRoot := seedReadyChange(t, root, name, "")
	if err := os.Remove(filepath.Join(changeRoot, "tasks.md")); err != nil {
		t.Fatal(err)
	}
	sealRolesKickoff(t, changeRoot, name, style, roles...)
	for role, text := range tasksByRole {
		write(t, filepath.Join(changeRoot, "tasks."+role+".md"), text)
	}
	return changeRoot
}

func TestMultiRoleTasksResolveFromSealedKickoff(t *testing.T) {
	root := t.TempDir()
	seedMultiRoleChange(t, root, "two-roles", kickoff.ExecutionContinuous, map[string]string{
		"backend":  "- [x] 1.1 Model\n- [ ] 1.2 Endpoint\n",
		"frontend": "- [ ] 1.1 Screen\n",
	}, "backend", "frontend")

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "two-roles"})
	if err != nil {
		t.Fatal(err)
	}
	assertArtifact(t, status, "tasks", ArtifactDone)
	if want := (TaskProgress{Total: 3, Completed: 1, Pending: 2}); status.TaskProgress != want {
		t.Fatalf("TaskProgress = %+v, want the aggregate %+v", status.TaskProgress, want)
	}
	if status.ApplyState != ApplyReady || status.NextRecommended != "apply" || len(status.BlockedReasons) != 0 {
		t.Fatalf("applyState = %q nextRecommended = %q blockedReasons = %v, want ready/apply without blockers", status.ApplyState, status.NextRecommended, status.BlockedReasons)
	}
	wantLocators := []string{
		filepath.Join(root, "openspec", "changes", "two-roles", "tasks.backend.md"),
		filepath.Join(root, "openspec", "changes", "two-roles", "tasks.frontend.md"),
	}
	if !reflect.DeepEqual(status.ArtifactPaths.Tasks, wantLocators) {
		t.Fatalf("ArtifactPaths.Tasks = %v, want the role files in kickoff order %v", status.ArtifactPaths.Tasks, wantLocators)
	}
}

func TestMultiRoleTasksFollowKickoffRoleOrder(t *testing.T) {
	root := t.TempDir()
	changeRoot := seedMultiRoleChange(t, root, "ordered", kickoff.ExecutionContinuous, map[string]string{
		"backend": "- [ ] 1.1 A\n", "frontend": "- [ ] 1.1 B\n",
	}, "frontend", "backend")

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "ordered"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(changeRoot, "tasks.frontend.md"), filepath.Join(changeRoot, "tasks.backend.md")}
	if !reflect.DeepEqual(status.ArtifactPaths.Tasks, want) {
		t.Fatalf("ArtifactPaths.Tasks = %v, want kickoff role order %v", status.ArtifactPaths.Tasks, want)
	}
}

func TestMultiRoleTasksMissingRoleFileIsPartialAndNamed(t *testing.T) {
	root := t.TempDir()
	seedMultiRoleChange(t, root, "one-missing", kickoff.ExecutionContinuous, map[string]string{
		"backend": "- [ ] 1.1 Endpoint\n",
	}, "backend", "frontend")

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "one-missing"})
	if err != nil {
		t.Fatal(err)
	}
	assertArtifact(t, status, "tasks", ArtifactPartial)
	if status.ApplyState != ApplyBlocked || status.NextRecommended != "tasks" {
		t.Fatalf("applyState = %q nextRecommended = %q, want blocked/tasks", status.ApplyState, status.NextRecommended)
	}
	if got, want := planningReasons(t, root, "one-missing", status, "backend", "frontend"), "tasks.frontend.md is missing or partial."; !reflect.DeepEqual(got, []string{want}) {
		t.Fatalf("expected planning reasons = %v, want only the missing tasks.frontend.md named", got)
	}

	// With no role file at all the artifact is missing, and both are named.
	empty := t.TempDir()
	seedMultiRoleChange(t, empty, "none-written", kickoff.ExecutionContinuous, nil, "backend", "frontend")
	status, err = Resolve(ResolveOptions{CWD: empty, ChangeName: "none-written"})
	if err != nil {
		t.Fatal(err)
	}
	assertArtifact(t, status, "tasks", ArtifactMissing)
	got := planningReasons(t, empty, "none-written", status, "backend", "frontend")
	if want := "tasks.backend.md, tasks.frontend.md are missing or partial."; !reflect.DeepEqual(got, []string{want}) {
		t.Fatalf("expected planning reasons = %v, want [%q]", got, want)
	}
}

// planningReasons returns the expected-planning blockers artifactBlockedReasons
// produces for a multi-role change. The tasks route drops them from the status
// output (blockerReasons.finalize), so the naming is asserted where it is made.
func planningReasons(t *testing.T, root, change string, status Status, roles ...string) []string {
	t.Helper()
	changeRoot := filepath.Join(root, "openspec", "changes", change)
	files := make([]string, 0, len(roles))
	for _, role := range roles {
		files = append(files, filepath.Join(changeRoot, "tasks."+role+".md"))
	}
	resolved, err := resolveRoleTasks(files)
	if err != nil {
		t.Fatal(err)
	}
	return artifactBlockedReasonsForRoleTasks(status.Artifacts, status.TaskProgress, change, nil, &resolved).expectedPlanning
}

func TestMultiRoleTasksRoleWithoutCheckboxesBlocksApply(t *testing.T) {
	root := t.TempDir()
	seedMultiRoleChange(t, root, "plain-prose", kickoff.ExecutionContinuous, map[string]string{
		"backend":  "- [ ] 1.1 Endpoint\n",
		"frontend": "just prose, no checklist\n",
	}, "backend", "frontend")

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "plain-prose"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ApplyState != ApplyBlocked {
		t.Fatalf("applyState = %q, want blocked while one role plan has no checkbox", status.ApplyState)
	}
	if reasons := strings.Join(status.BlockedReasons, "\n"); !strings.Contains(reasons, "tasks.frontend.md has no markdown task checkboxes.") {
		t.Fatalf("blockedReasons = %q, want tasks.frontend.md named", reasons)
	}
}

func TestMultiRoleTasksAllCompleteReportsAllDone(t *testing.T) {
	root := t.TempDir()
	seedMultiRoleChange(t, root, "finished", kickoff.ExecutionContinuous, map[string]string{
		"backend": "- [x] 1.1 Endpoint\n", "frontend": "- [x] 1.1 Screen\n- [x] 1.2 Route\n",
	}, "backend", "frontend")

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "finished"})
	if err != nil {
		t.Fatal(err)
	}
	if want := (TaskProgress{Total: 3, Completed: 3, AllComplete: true}); status.TaskProgress != want {
		t.Fatalf("TaskProgress = %+v, want %+v", status.TaskProgress, want)
	}
	if status.ApplyState != ApplyAllDone || status.Dependencies.Archive != DependencyReady {
		t.Fatalf("applyState = %q archive = %q, want all-done and archive ready", status.ApplyState, status.Dependencies.Archive)
	}
}

// One role with pending work keeps apply open even when the other role is done.
func TestMultiRoleTasksOnePendingRoleKeepsApplyReady(t *testing.T) {
	root := t.TempDir()
	seedMultiRoleChange(t, root, "half-done", kickoff.ExecutionContinuous, map[string]string{
		"backend": "- [x] 1.1 Endpoint\n", "frontend": "- [ ] 1.1 Screen\n",
	}, "backend", "frontend")

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "half-done"})
	if err != nil {
		t.Fatal(err)
	}
	if status.TaskProgress.AllComplete || status.ApplyState != ApplyReady || status.NextRecommended != "apply" {
		t.Fatalf("progress = %+v applyState = %q nextRecommended = %q, want ready/apply", status.TaskProgress, status.ApplyState, status.NextRecommended)
	}
}

func TestMultiRoleEditAuthorityUnitesTheRootsOfEveryRole(t *testing.T) {
	workspace := t.TempDir() // the workspace itself is NOT a Git repository
	planning := filepath.Join(workspace, "planning")
	apiRepo := filepath.Join(workspace, "api-repo")
	uiRepo := filepath.Join(workspace, "ui-repo")
	initEditAuthorityGitRepo(t, planning, true)
	initEditAuthorityGitRepo(t, apiRepo, false)
	initEditAuthorityGitRepo(t, uiRepo, false)
	changeRoot := seedMultiRoleChange(t, planning, "two-repos", kickoff.ExecutionContinuous, map[string]string{
		"backend": strings.Join([]string{
			"- [ ] 1.1 Add the handler in `../api-repo/src/x.cs`",
			"- [ ] 2.1 Later work in `../later-repo/src/z.cs`",
			"",
		}, "\n"),
		"frontend": "- [ ] 1.1 Add the view in `../ui-repo/src/y.ts`\n",
	}, "backend", "frontend")

	status, err := Resolve(ResolveOptions{CWD: planning, ChangeName: "two-repos"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ApplyState != ApplyBlocked || status.NextRecommended == "apply" {
		t.Fatalf("applyState = %q nextRecommended = %q, want blocked by edit authority", status.ApplyState, status.NextRecommended)
	}
	wantAPI, wantUI := realPath(t, apiRepo), realPath(t, uiRepo)
	reasons := strings.Join(status.BlockedReasons, "\n")
	for _, want := range []string{"blocked(edit_authority_missing): tasks.backend.md, tasks.frontend.md target edit paths", wantAPI, wantUI} {
		if !strings.Contains(reasons, want) {
			t.Fatalf("blockedReasons = %q, want it to contain %q", reasons, want)
		}
	}

	// The prepared marker lets the status carry one envelope for both roots.
	if err := PrepareChangeInstanceConsent(status); err != nil {
		t.Fatal(err)
	}
	status, err = Resolve(ResolveOptions{CWD: planning, ChangeName: "two-repos"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Consent == nil {
		t.Fatalf("blocked status carries no consent envelope; blockedReasons = %v", status.BlockedReasons)
	}
	if !reflect.DeepEqual(status.Consent.MissingRoots, []string{wantAPI, wantUI}) {
		t.Fatalf("Consent.MissingRoots = %v, want both sibling roots [%s %s]", status.Consent.MissingRoots, wantAPI, wantUI)
	}
	// The later work unit of the backend plan is informational only.
	notes := strings.Join(status.Notes, "\n")
	if !strings.Contains(notes, "note(future_edit_roots): a later work unit in tasks.backend.md") || strings.Contains(strings.Join(status.BlockedReasons, "\n"), "later-repo") {
		t.Fatalf("notes = %q, want the future root reported in notes naming tasks.backend.md, not as a blocker", notes)
	}

	token, err := readChangeInstanceMarker(changeRoot)
	if err != nil || token == "" {
		t.Fatalf("no change-instance marker: %q, %v", token, err)
	}
	opened, err := OpenRuntimeStore(context.Background(), planning, "two-repos")
	if err != nil {
		t.Fatal(err)
	}
	store, err := opened.ForInstance(token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Grant(context.Background(), GrantRootsRequest{
		RequestID: "grant-two-repos", Roots: []string{wantAPI, wantUI},
		Reason: "maintainer authorized both code repositories", Actor: "maintainer",
	}); err != nil {
		t.Fatal(err)
	}
	granted, err := Resolve(ResolveOptions{CWD: planning, ChangeName: "two-repos"})
	if err != nil {
		t.Fatal(err)
	}
	if granted.ApplyState != ApplyReady || granted.NextRecommended != "apply" || len(granted.BlockedReasons) != 0 {
		t.Fatalf("post-grant status = %q/%q reasons %v, want ready/apply", granted.ApplyState, granted.NextRecommended, granted.BlockedReasons)
	}
}

// A single sealed role whose plan lives in tasks.<role>.md is per-role too, and
// its diagnostics name that one file in the singular.
func TestSingleSealedRoleWithOwnTasksFileResolvesIt(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	apiRepo := filepath.Join(workspace, "api-repo")
	initEditAuthorityGitRepo(t, planning, true)
	initEditAuthorityGitRepo(t, apiRepo, false)
	seedMultiRoleChange(t, planning, "one-role", kickoff.ExecutionContinuous, map[string]string{
		"backend": "- [ ] 1.1 Add the handler in `../api-repo/src/x.cs`\n",
	}, "backend")

	status, err := Resolve(ResolveOptions{CWD: planning, ChangeName: "one-role"})
	if err != nil {
		t.Fatal(err)
	}
	assertArtifact(t, status, "tasks", ArtifactDone)
	if reasons := strings.Join(status.BlockedReasons, "\n"); !strings.Contains(reasons, "blocked(edit_authority_missing): tasks.backend.md targets edit paths") {
		t.Fatalf("blockedReasons = %q, want the singular tasks.backend.md wording", reasons)
	}
}

// In checkpointed mode the shared tasks digest covers every role file, so the
// tasks gate opens once both exist (machine.go tasksGateOpen needs it) and not
// before.
func TestMultiRoleCheckpointedTasksGateOpensOnceEveryRoleFileExists(t *testing.T) {
	root := t.TempDir()
	changeRoot := seedMultiRoleChange(t, root, "gated", kickoff.ExecutionCheckpointed, map[string]string{
		"backend": "- [ ] 1.1 Endpoint\n",
	}, "backend", "frontend")

	tasksGate := func() kickoff.GateState {
		t.Helper()
		status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "gated"})
		if err != nil || status.Governance == nil {
			t.Fatalf("Resolve() = (governance=%v, err=%v), want projected governance", status.Governance != nil, err)
		}
		for _, gate := range status.Governance.Gates {
			if gate.Key == kickoff.GateTasks {
				return gate
			}
		}
		return kickoff.GateState{}
	}
	if gate := tasksGate(); gate.Key == kickoff.GateTasks {
		t.Fatalf("tasks gate %+v is open while tasks.frontend.md is missing", gate)
	}

	write(t, filepath.Join(changeRoot, "tasks.frontend.md"), "- [ ] 1.1 Screen\n")
	if gate := tasksGate(); gate.Key != kickoff.GateTasks || gate.Status != "pending" {
		t.Fatalf("tasks gate = %+v, want it open and pending once both role files exist", gate)
	}
}

func TestGateEvidenceNamesTheRoleTaskFiles(t *testing.T) {
	paths := ArtifactPaths{Tasks: []string{"c/tasks.backend.md", "c/tasks.frontend.md"}}
	if got := gateEvidence(kickoff.GateState{Key: kickoff.GateTasks}, paths); !reflect.DeepEqual(got, paths.Tasks) {
		t.Fatalf("tasks gate evidence = %v, want both role files %v", got, paths.Tasks)
	}
	role := kickoff.GateState{Key: kickoff.RoleApplyGate("frontend"), Blocks: "frontend"}
	if got := gateEvidence(role, paths); !reflect.DeepEqual(got, []string{"c/tasks.frontend.md"}) {
		t.Fatalf("role-apply:frontend evidence = %v, want only its own plan", got)
	}
}
