package sddstatus

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// This file pins the shared extractor fix for the "prose read as a
// filesystem edit path" defect family:
//
//   - #4096: an oxlint `../../` import-glob inside backticked tasks.md prose
//     is treated as an unauthorized external edit root.
//   - #4192: a canonical web route literal `/` (and routes like
//     `/api/health`) in prose is treated as a filesystem path requiring
//     authority.
//   - gp#598: backticked HTTP routes like `GET /api/health` are parsed as
//     edit-authority filesystem paths.
//   - #4103: edit authority is evaluated across the whole task plan, so a
//     future task naming an external root blocks an unrelated current task.
//
// editTargetTokens is the one derivation both detectUnauthorizedEditRoots and
// the runtime-topology guard route through, so fixing its candidate gating
// fixes the whole family at the source.

func TestExtractEditAuthorityTargetTokensExcludesNonPathProseAndKeepsRealPaths(t *testing.T) {
	workspace := t.TempDir()
	mkdir(t, filepath.Join(workspace, "internal", "sddstatus"))
	mkdir(t, filepath.Join(workspace, "Repos EMT", "api"))

	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "oxlint import-glob with a double-dot traversal is not a filesystem edit path (#4096)",
			line: "- [x] 3.2 `.oxlintrc.json`: MODIFY `src/some/dir/*` — add `!../../i18n/*`. Test-first: no — injection-probed.",
			want: nil,
		},
		{
			name: "canonical route literal `/` is not a filesystem edit path (#4192)",
			line: "- [ ] 1.1 Document that the canonical route set includes `/` and other routes like `/api/health`.",
			want: nil,
		},
		{
			name: "backticked HTTP method and route is not a filesystem edit path (gp#598)",
			line: "- [ ] 1.1 Add the readiness check for `GET /api/health`.",
			want: nil,
		},
		{
			name: "bare route path without a method prefix is still not a filesystem edit path (gp#598)",
			line: "- [ ] 1.1 Document the route `/api/health` for the readiness check.",
			want: nil,
		},
		{
			name: "real repository-relative Go path is still extracted",
			line: "- [ ] 1.1 Update `internal/cli/foo.go` with the new flag",
			want: []string{"internal/cli/foo.go"},
		},
		{
			name: "sibling repository TypeScript path is still extracted",
			line: "- [ ] 1.1 Update `../shared/lib.ts` for the shared client",
			want: []string{"../shared/lib.ts"},
		},
		{
			name: "new markdown doc under an existing directory is still extracted",
			line: "- [ ] 1.1 Write `docs/new-file.md` describing the rollout",
			want: []string{"docs/new-file.md"},
		},
		{
			name: "existing repository directory without an extension is still extracted",
			line: "- [ ] 1.1 Update the tests under `internal/sddstatus`",
			want: []string{"internal/sddstatus"},
		},
		{
			name: "explicit Files marker authorizes an extensionless declared path",
			line: "- [ ] 1.1 Files: `docs/adr` will hold the new decision record",
			want: []string{"docs/adr"},
		},
		{
			name: "extensionless path mentioned without a marker or an existing directory is not extracted",
			line: "- [ ] 1.1 Mention `docs/adr` as a possible location",
			want: nil,
		},
		{
			name: "Next.js dynamic-route directory segment is still extracted, not treated as a glob",
			line: "- [ ] 1.1 Add the loader to `src/app/[id]/page.tsx`",
			want: []string{"src/app/[id]/page.tsx"},
		},
		{
			name: "spaced relative path whose first segment exists under the workspace is extracted",
			line: "- [ ] 1.1 Update `Repos EMT/api/handler.go` with the new route",
			want: []string{"Repos EMT/api/handler.go"},
		},
		{
			name: "spaced path anchored with ./ is extracted",
			line: "- [ ] 1.1 Update `./mi carpeta/handler.go` with the new route",
			want: []string{"./mi carpeta/handler.go"},
		},
		{
			name: "spaced path anchored with ../ is extracted",
			line: "- [ ] 1.1 Update `../otro repo/lib.ts` for the shared client",
			want: []string{"../otro repo/lib.ts"},
		},
		{
			name: "spaced relative path with no anchor and no existing first segment is not extracted",
			line: "- [ ] 1.1 See `Some Folder/handler.go` for the old behaviour",
			want: nil,
		},
		{
			name: "go test command line is not a path",
			line: "- [ ] 1.1 Run `go test ./...` before merging",
			want: nil,
		},
		{
			name: "command with a flag is not a path even when it starts with a script path",
			line: "- [ ] 1.1 Regenerate the fixtures with `scripts/x.sh --update`",
			want: nil,
		},
		{
			name: "command chained with && is not a path",
			line: "- [ ] 1.1 Build with `cd web && ./build.sh`",
			want: nil,
		},
		{
			name: "command with a redirection is not a path even when anchored with ./",
			line: "- [ ] 1.1 Capture `./mi carpeta/run.sh > out.log` for the report",
			want: nil,
		},
		{
			name: "command with a flag is not a path even under an explicit Files marker",
			line: "- [ ] 1.1 Files: `./mi carpeta/run.sh --update` regenerates the fixtures",
			want: nil,
		},
		{
			name: "spaced URL is not a path",
			line: "- [ ] 1.1 Fetch `https://example.com/a b/c.json` for the fixture",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := editTargetTokens(tt.line, workspace)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("editTargetTokens(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

// TestExistingAbsoluteExtensionlessDirectoryOutsideRootsIsStillFlagged pins
// the R3-absolute-extensionless-excluded fix: a real absolute directory with
// no extension that already exists on disk outside every authorized root must
// still be caught, not swallowed by the route-ish exclusion meant only for
// URL literals like bare `/` or `/api/health`.
func TestExistingAbsoluteExtensionlessDirectoryOutsideRootsIsStillFlagged(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	external := filepath.Join(workspace, "external-app", "deploy")
	initEditAuthorityGitRepo(t, planning, false)
	mkdir(t, external)

	tasks := "- [ ] 1.1 Run the rollout script in `" + external + "`\n"
	flagged := detectUnauthorizedEditRoots(tasks, planning, []string{planning}, identityRebase)
	want := realPath(t, external)
	if len(flagged) != 1 || flagged[0] != want {
		t.Fatalf("detectUnauthorizedEditRoots() = %v, want exactly [%s]", flagged, want)
	}
}

// TestExplicitlyDeclaredWildcardTargetIsStillFlagged pins the
// R3-glob-exclusion-false-negative fix: a wildcard target named under an
// explicit Files:/Edit:/Touch: declaration is a genuine authorization
// question, not prose to silently ignore.
func TestExplicitlyDeclaredWildcardTargetIsStillFlagged(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	shared := filepath.Join(workspace, "shared")
	initEditAuthorityGitRepo(t, planning, false)
	initEditAuthorityGitRepo(t, shared, false)

	tasks := "- [ ] 1.1 Files: `../shared/*.ts` for the generated client types\n"
	flagged := detectUnauthorizedEditRoots(tasks, planning, []string{planning}, identityRebase)
	want := realPath(t, shared)
	if len(flagged) != 1 || flagged[0] != want {
		t.Fatalf("detectUnauthorizedEditRoots() = %v, want exactly [%s]", flagged, want)
	}
}

// TestSpacedAbsolutePathOutsideRootsIsFlaggedWithQuotedReason pins the fail-open
// this fix closes: a workspace folder with a space in its name used to make the
// whole token "a command", so a task editing a real file outside every
// authorized root was never detected. The path is flagged as one root, and the
// blocked reason prints it quoted so the exits stay copy-pasteable.
func TestSpacedAbsolutePathOutsideRootsIsFlaggedWithQuotedReason(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	external := filepath.Join(workspace, "Repos EMT", "otro repo")
	initEditAuthorityGitRepo(t, planning, false)
	mkdir(t, external)
	mkdir(t, filepath.Join(planning, "mi carpeta"))

	tasks := strings.Join([]string{
		"- [ ] 1.1 Update `" + filepath.Join(external, "handler.go") + "` with the new route",
		"- [ ] 1.2 Update `" + filepath.Join(planning, "mi carpeta", "notes.md") + "` inside the authorized root",
		"",
	}, "\n")
	flagged := detectUnauthorizedEditRoots(tasks, planning, []string{planning}, identityRebase)
	want := realPath(t, external)
	if len(flagged) != 1 || flagged[0] != want {
		t.Fatalf("detectUnauthorizedEditRoots() = %v, want exactly [%s]", flagged, want)
	}
	if reason := editAuthorityBlockedReason(flagged); !strings.Contains(reason, pathquote.Quote(want)) {
		t.Fatalf("blocked reason = %q, want it to name %s quoted", reason, pathquote.Quote(want))
	}
}

// TestSpacedExternalPathBlocksApplyWithEditAuthorityMissing is the end-to-end
// form of the same defect: native status for a tasks.md whose only target is a
// spaced path outside the authorized roots must block apply.
func TestSpacedExternalPathBlocksApplyWithEditAuthorityMissing(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	external := filepath.Join(workspace, "Repos EMT", "otro repo")
	initEditAuthorityGitRepo(t, planning, true)
	mkdir(t, external)

	seedReadyChange(t, planning, "spaced-external-target", strings.Join([]string{
		"- [ ] 1.1 Update `" + filepath.Join(external, "handler.go") + "` with the new route",
		"",
	}, "\n"))

	status, err := Resolve(ResolveOptions{CWD: planning, ChangeName: "spaced-external-target"})
	if err != nil {
		t.Fatal(err)
	}
	reasons := strings.Join(status.BlockedReasons, "\n")
	if status.ApplyState != ApplyBlocked || !strings.Contains(reasons, "blocked(edit_authority_missing)") {
		t.Fatalf("applyState = %q, blockedReasons = %v; want apply blocked by edit_authority_missing", status.ApplyState, status.BlockedReasons)
	}
	if want := pathquote.Quote(realPath(t, external)); !strings.Contains(reasons, want) {
		t.Fatalf("blocked reasons = %q, want them to name %s", reasons, want)
	}
}

// TestReportedFalsePositiveProseDoesNotBlockEditAuthorityApply is the
// end-to-end regression for all three prose-shaped false positives in one
// fixture: #4096's oxlint import glob, #4192's canonical route literal, and
// gp#598's backticked HTTP method and route.
func TestReportedFalsePositiveProseDoesNotBlockEditAuthorityApply(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	seedReadyChange(t, repo, "false-positive-prose", strings.Join([]string{
		"- [x] 1.1 Add the header check",
		"- [ ] 1.2 `.oxlintrc.json`: MODIFY `src/some/dir/*` — add `!../../i18n/*`. Test-first: no — injection-probed.",
		"- [ ] 1.3 Document that the canonical route set includes `/` and other routes like `/api/health`.",
		"- [ ] 1.4 Add the readiness check for `GET /api/health`.",
		"- [ ] 1.5 Update `internal/auth/login.go` with the route table",
		"",
	}, "\n"))

	status, err := Resolve(ResolveOptions{CWD: repo, ChangeName: "false-positive-prose"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ApplyState != ApplyReady || status.NextRecommended != "apply" {
		t.Fatalf("prose-shaped false positive falsely blocked apply: applyState = %q, nextRecommended = %q, blockedReasons = %v",
			status.ApplyState, status.NextRecommended, status.BlockedReasons)
	}
	if len(status.BlockedReasons) != 0 {
		t.Fatalf("prose-shaped false positive produced unexpected blocked reasons: %v", status.BlockedReasons)
	}
}

// TestFutureWorkUnitEditAuthorityRootIsInformationalNotBlocking pins #4103's
// scoping fix and #4372's channel split: a future work unit (a higher leading
// task number, e.g. "2.1" following "1.x") that genuinely targets an external
// root must not block the current work unit's apply, and the informational
// report of that root must travel on `notes` — never on the gated
// `blockedReasons` a consumer contract reads as "stop".
func TestFutureWorkUnitEditAuthorityRootIsInformationalNotBlocking(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	// deploy is deliberately NOT a Git repository: this test isolates the
	// edit-authority scoping fix (#4103) from the unrelated runtime-topology
	// guard, which independently blocks apply for a future task naming a
	// foreign Git repository regardless of work-unit scope.
	deploy := filepath.Join(workspace, "deploy-config")
	initEditAuthorityGitRepo(t, planning, true)
	mkdir(t, deploy)
	mkdir(t, filepath.Join(planning, "internal", "auth"))
	write(t, filepath.Join(planning, "internal", "auth", "login.go"), "package auth\n")

	seedReadyChange(t, planning, "future-unit-rollout", strings.Join([]string{
		"- [x] 1.1 Update `internal/auth/login.go` with the new claim check",
		"- [ ] 1.2 Update `internal/auth/session.go` with the refreshed token flow",
		"- [ ] 2.1 Deploy the change via `../deploy-config/scripts/rollout.sh`",
		"",
	}, "\n"))

	status, err := Resolve(ResolveOptions{CWD: planning, ChangeName: "future-unit-rollout"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ApplyState != ApplyReady || status.NextRecommended != "apply" {
		t.Fatalf("future work unit's external root falsely blocked the current work unit: applyState = %q, nextRecommended = %q, blockedReasons = %v",
			status.ApplyState, status.NextRecommended, status.BlockedReasons)
	}
	reasons := strings.Join(status.BlockedReasons, "\n")
	if strings.Contains(reasons, "blocked(edit_authority_missing)") {
		t.Fatalf("future work unit's external root produced a blocking reason: %v", status.BlockedReasons)
	}
	if len(status.BlockedReasons) != 0 {
		t.Fatalf("a non-blocking informational note reached the gated blockedReasons: %v", status.BlockedReasons)
	}
	wantDeploy := realPath(t, deploy)
	notes := strings.Join(status.Notes, "\n")
	if !strings.Contains(notes, "note(future_edit_roots)") || !strings.Contains(notes, wantDeploy) {
		t.Fatalf("notes must carry an informational future-edit-roots note naming %q: %v", wantDeploy, status.Notes)
	}

	// The consumer-visible document is the surface that actually contradicted the
	// contracts in #4372, so pin the split after projection too.
	projected, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	if _, ok := document["notes"]; !ok {
		t.Fatalf("v2 document omitted the notes field: %s", payload)
	}
	var projectedBlockers, projectedNotes []string
	if err := json.Unmarshal(document["blockedReasons"], &projectedBlockers); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(document["notes"], &projectedNotes); err != nil {
		t.Fatal(err)
	}
	if len(projectedBlockers) != 0 {
		t.Fatalf("v2 blockedReasons retained the informational note: %v", projectedBlockers)
	}
	if len(projectedNotes) != len(status.Notes) {
		t.Fatalf("v2 notes = %v, want the resolved notes %v", projectedNotes, status.Notes)
	}
}

// TestDetectUnauthorizedEditAuthorityRootsForCurrentWorkUnitScopesToCurrentUnit pins the
// lower-level split: the current (lowest-numbered) pending work unit's targets
// block, a later work unit's targets are reported as future only, and an
// already-completed task's target stays in scope regardless of its number.
func TestDetectUnauthorizedEditAuthorityRootsForCurrentWorkUnitScopesToCurrentUnit(t *testing.T) {
	workspace := t.TempDir()
	planning := filepath.Join(workspace, "planning")
	serviceA := filepath.Join(workspace, "service-a")
	serviceB := filepath.Join(workspace, "service-b")
	initEditAuthorityGitRepo(t, planning, false)
	initEditAuthorityGitRepo(t, serviceA, false)
	initEditAuthorityGitRepo(t, serviceB, false)

	tasks := strings.Join([]string{
		"- [x] 1.1 Already touched `../service-a/legacy/done.go`",
		"- [ ] 1.2 Still touching `../service-a/internal/api/handler.go`",
		"- [ ] 2.1 Later touches `../service-b/internal/worker/consume.go`",
		"",
	}, "\n")

	blocking, future := detectUnauthorizedEditRootsForCurrentWorkUnit(tasks, planning, []string{planning}, identityRebase)
	wantA := realPath(t, serviceA)
	wantB := realPath(t, serviceB)
	if !reflect.DeepEqual(blocking, []string{wantA}) {
		t.Fatalf("blocking = %v, want [%s] (completed task stays in scope, current pending unit 1 stays in scope)", blocking, wantA)
	}
	if !reflect.DeepEqual(future, []string{wantB}) {
		t.Fatalf("future = %v, want [%s] (unit 2 has not started)", future, wantB)
	}
}
