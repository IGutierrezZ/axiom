// Package organicruntime_test proves the journeys a configured agent actually
// performs once the product stopped owning implementation: the agent implements
// organically and delivery follows ordinary repository policy.
//
// Every assertion here is driven through the real binary against real Git
// repositories and a real bare remote.
package organicruntime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/testenv"
	"github.com/IGutierrezZ/axiom/v3/internal/versions"
)

const (
	// realAgentE2EEnvironment gates the pinned real-agent journeys, which need a
	// pinned OpenCode plus network access to the pinned plugin package.
	realAgentE2EEnvironment = "AXIOM_REAL_AGENT_E2E"
	pinnedOpenCodeVersion   = versions.OpenCode

	organicCommandWaitDelay = 10 * time.Second

	// organicWithdrawalDeadline replaces the retired short-TTL harness. The
	// surviving authorization has no wall-clock lifetime: it is withdrawn by an
	// explicit user action instead of expiring, so the harness that used to sleep
	// out a TTL now performs the withdrawal instantly. The deadline only keeps CI
	// honest — a journey that needs longer than this has stopped being a journey.
	organicWithdrawalDeadline = 60 * time.Second
)

// Time budgets of the harness commands. They are variables so TestMain can
// scale them with AXIOM_TEST_TIMEOUT_SCALE (default 2 on Windows, where a loaded
// machine measured the local budget expiring at 118s of a 90s budget). Only the
// harness budgets move; the built binary keeps its production budgets.
var (
	organicLocalTimeout = 90 * time.Second
	organicSetupTimeout = 5 * time.Minute
	organicAgentTimeout = 4 * time.Minute
)

// scaleOrganicTimeBudgets applies the test time scale to the harness budgets.
// It runs once, in the process that drives the journeys.
func scaleOrganicTimeBudgets() error {
	for _, budget := range []*time.Duration{&organicLocalTimeout, &organicSetupTimeout, &organicAgentTimeout} {
		scaled, err := testenv.Scaled(*budget)
		if err != nil {
			return err
		}
		*budget = scaled
	}
	return nil
}

// Actor process contract. The delegated worker must be a real, separate OS
// process rather than an in-test closure, because the behaviour under test is
// exactly what a sub-agent does on its own: it implements and commits, and it
// never escalates its own route. Re-executing the compiled test binary keeps
// that real without adding a language runtime dependency to the suite.
const (
	organicActorRoleEnvironment    = "AXIOM_ORGANIC_ACTOR_ROLE"
	organicActorRepoEnvironment    = "AXIOM_ORGANIC_ACTOR_REPO"
	organicActorPathEnvironment    = "AXIOM_ORGANIC_ACTOR_PATH"
	organicActorBodyEnvironment    = "AXIOM_ORGANIC_ACTOR_BODY"
	organicActorMessageEnvironment = "AXIOM_ORGANIC_ACTOR_MESSAGE"
	organicActorBinaryEnvironment  = "AXIOM_ORGANIC_ACTOR_BINARY"
	organicTestBinaryEnvironment   = "AXIOM_ORGANIC_TEST_BINARY"

	organicActorRoleDirect    = "direct"
	organicActorRoleDelegated = "delegated"

	organicDirectActorMarker    = "ORGANIC_DIRECT_CANDIDATE_COMMITTED"
	organicDelegatedActorMarker = "ORGANIC_DELEGATED_CANDIDATE_COMMITTED"
)

var organicBinary string

// requireOrganicBinary skips the journey when TestMain did not build the
// product binary because testenv.BuildBinariesEnv is not set. A binary supplied
// through organicTestBinaryEnvironment needs no compilation and always runs.
func requireOrganicBinary(t testing.TB) {
	t.Helper()
	if organicBinary == "" {
		testenv.RequireBinaryBuilds(t)
		t.Fatal("the organic product binary was not built although binary builds are enabled")
	}
}

func TestMain(m *testing.M) {
	// The journeys run the real built binary (install, sync), which is not a test
	// binary and would otherwise write the developer's persistent Windows PATH.
	// Spawns built from os.Environ() inherit this opt-out; the closed
	// organicEnvironment whitelist sets it explicitly.
	if err := os.Setenv(system.NoPersistentPathEnvVar, "1"); err != nil {
		fmt.Fprintf(os.Stderr, "set %s: %v\n", system.NoPersistentPathEnvVar, err)
		os.Exit(1)
	}
	if role := strings.TrimSpace(os.Getenv(organicActorRoleEnvironment)); role != "" {
		os.Exit(runOrganicActor(role))
	}
	if err := scaleOrganicTimeBudgets(); err != nil {
		fmt.Fprintf(os.Stderr, "scale the organic time budgets: %v\n", err)
		os.Exit(1)
	}
	if binary := strings.TrimSpace(os.Getenv(organicTestBinaryEnvironment)); binary != "" {
		resolvedBinary, err := exec.LookPath(binary)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s=%q does not resolve to an executable: %v\n", organicTestBinaryEnvironment, binary, err)
			os.Exit(1)
		}
		organicBinary = resolvedBinary
		os.Exit(m.Run())
	}
	if !testenv.BinaryBuildsEnabled() {
		// Nothing is compiled without the opt-in: organicBinary stays empty and
		// every journey that needs it skips itself (requireOrganicBinary).
		os.Exit(m.Run())
	}
	workspace, err := os.MkdirTemp("", "organic-e2e-binary")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create organic binary workspace: %v\n", err)
		os.Exit(1)
	}
	binary, err := buildOrganicBinary(workspace)
	if err != nil {
		_ = os.RemoveAll(workspace)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	organicBinary = binary
	code := m.Run()
	_ = os.RemoveAll(workspace)
	os.Exit(code)
}

// runOrganicActor is the implementation actor. It edits exactly one already
// understood file and explicitly creates the candidate commit, which is the only
// thing an organic actor owes: the provider never creates or guesses a commit.
func runOrganicActor(role string) int {
	repo := os.Getenv(organicActorRepoEnvironment)
	relative := os.Getenv(organicActorPathEnvironment)
	body := os.Getenv(organicActorBodyEnvironment)
	message := os.Getenv(organicActorMessageEnvironment)
	if repo == "" || relative == "" || message == "" {
		fmt.Fprintln(os.Stderr, "organic actor requires repository, path, and message")
		return 1
	}

	marker := organicDirectActorMarker
	if role == organicActorRoleDelegated {
		marker = organicDelegatedActorMarker
	}

	target := filepath.Join(repo, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "organic actor mkdir: %v\n", err)
		return 1
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "organic actor write: %v\n", err)
		return 1
	}
	for _, arguments := range [][]string{
		{"add", "--", relative},
		{"commit", "-q", "-m", message},
	} {
		if _, err := organicGitOutput(context.Background(), repo, arguments...); err != nil {
			fmt.Fprintf(os.Stderr, "organic actor git: %v\n", err)
			return 1
		}
	}
	fmt.Print(marker)
	return 0
}

// ---------------------------------------------------------------------------
// Journeys
// ---------------------------------------------------------------------------

// TestOrganicDirectoryIdentityAcceptsCanonicalAliases keeps the repository
// selection boundary: relative, absolute, aliased, and `git -C` forms all denote
// exactly one repository, and a non-directory never does.
func TestOrganicDirectoryIdentityAcceptsCanonicalAliases(t *testing.T) {
	t.Parallel()
	harness := newOrganicHarness(t)
	worktree := harness.repo.worktree

	canonical := harness.commonDir()
	relative := harness.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(relative) {
		relative = filepath.Join(worktree, relative)
	}
	if !sameOrganicDirectory(canonical, relative) {
		t.Fatalf("relative and absolute common-dir forms denote different repositories: %q vs %q", relative, canonical)
	}

	parent := filepath.Dir(worktree)
	viaParent, err := organicGitOutput(context.Background(), parent, "-C", worktree, "rev-parse", "--absolute-git-dir")
	if err != nil {
		t.Fatal(err)
	}
	if !sameOrganicDirectory(canonical, viaParent) {
		t.Fatalf("git -C form denotes a different repository: %q vs %q", viaParent, canonical)
	}

	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(worktree, alias); err != nil {
		t.Skipf("directory aliases are unavailable: %v", err)
	}
	viaAlias, err := organicGitOutput(context.Background(), alias, "rev-parse", "--absolute-git-dir")
	if err != nil {
		t.Fatal(err)
	}
	if !sameOrganicDirectory(canonical, viaAlias) {
		t.Fatalf("aliased worktree denotes a different repository: %q vs %q", viaAlias, canonical)
	}

	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if sameOrganicDirectory(canonical, file) {
		t.Fatal("a regular file was accepted as the repository directory")
	}
}

// TestOrganicConfiguredAgentReceivesRoutingGuidance is the optional-SDD
// "proposed" leg. Every configured agent is told the same thing through its own
// delivery strategy: three routes exist, SDD is only ever proposed, and it is
// selected only by an explicit request or an accepted proposal.
// organicRoutingGuidanceRequiredFragments is the routing-guidance content
// every configured agent must receive, shared between this file's Cursor
// case and organic_runtime_real_agent_detection_test.go's Claude Code /
// OpenCode cases (see that file for why they're split).
var organicRoutingGuidanceRequiredFragments = []string{
	"Direct inline",
	"Delegated direct",
	"Optional SDD",
	"never selects SDD",
	"never create SDD artifacts",
}

// organicRoutingGuidanceForbiddenFragments is RDD prose that routing guidance
// must no longer carry while RDD is suspended (C-01-c).
var organicRoutingGuidanceForbiddenFragments = []string{
	"review mode enable|disable|status",
	"disabled/unmanaged",
}

// TestOrganicConfiguredAgentReceivesRoutingGuidanceCursor proves the
// markdown-rules delivery strategy for Cursor. Cursor's Detect is
// config-dir-only (~/.cursor, no PATH lookup — see
// internal/agents/cursor/adapter.go), so this case needs no real agent
// binary and runs unconditionally in the ordinary unit sweep.
//
// Claude Code and OpenCode's equivalent cases used to live in this same
// table-driven test, but their detection follows the inherited PATH to a
// real installed binary — install refuses instead of installing a missing
// runtime now (agentInstallStep in internal/cli/run.go) — so running them
// here depended on those binaries happening to be on the machine running
// `go test ./...`, which is true on developer machines but not on every CI
// runner. They now live in
// organic_runtime_real_agent_detection_test.go, gated behind the
// real_agent_e2e build tag so they run only in the organic-runtime-e2e CI
// job, which installs both runtimes first.
func TestOrganicConfiguredAgentReceivesRoutingGuidanceCursor(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	home := t.TempDir()
	if _, err := organicGitOutput(context.Background(), workspace, "init", "--quiet", "--initial-branch=main", "."); err != nil {
		t.Fatal(err)
	}
	// Cursor's Detect looks for ~/.cursor, which this fake isolated HOME
	// never has. Simulate Cursor as already installed so axiom does not
	// correctly refuse an undetected agent here — this test targets
	// routing-guidance delivery, not agent install behavior.
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	const path = ".cursor/rules/axiom.mdc"
	output, stderr, err := runOrganicCommand(
		t, organicBinary, workspace, organicEnvironment(home),
		"install", "--agent", "cursor", "--scope", "workspace", "--components", "permissions",
	)
	if err != nil {
		t.Fatalf("install cursor: %v\nstdout:\n%s\nstderr:\n%s", err, output, stderr)
	}
	rendered, readErr := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(path)))
	if readErr != nil {
		t.Fatalf("configured agent cursor received no routing guidance at %s: %v", path, readErr)
	}
	for _, fragment := range organicRoutingGuidanceRequiredFragments {
		if !bytes.Contains(rendered, []byte(fragment)) {
			t.Fatalf("routing guidance for cursor omits %q:\n%s", fragment, rendered)
		}
	}
	for _, fragment := range organicRoutingGuidanceForbiddenFragments {
		if bytes.Contains(rendered, []byte(fragment)) {
			t.Fatalf("routing guidance for cursor still carries RDD prose %q:\n%s", fragment, rendered)
		}
	}
}

// TestOrganicImplementationRoutesReachDelivery walks the two organic
// implementation routes end to end: a real actor process produces the candidate
// and ordinary repository policy delivers it under compare-and-swap.
func TestOrganicImplementationRoutesReachDelivery(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		role   string
		marker string
		path   string
		body   string
	}{
		{
			name:   "direct inline",
			role:   organicActorRoleDirect,
			marker: organicDirectActorMarker,
			path:   "docs/direct-note.md",
			body:   organicLines("direct implementation line", 10),
		},
		{
			name:   "delegated direct",
			role:   organicActorRoleDelegated,
			marker: organicDelegatedActorMarker,
			path:   "docs/delegated-note.md",
			body:   organicLines("delegated implementation line", 10),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			harness := newOrganicHarness(t)
			harness.runActor(test.role, test.path, test.body, "docs: add an organic note", test.marker)

			candidate := harness.git("rev-parse", "HEAD")
			if candidate == harness.repo.baseRevision {
				t.Fatal("the actor never created a candidate commit")
			}

			harness.pushWithLease(harness.repo.baseRevision)
			harness.assertRemoteBlob(test.path, test.body)
			harness.assertOnlyMainRef()
			harness.assertStaleLeaseIsRejected(harness.repo.baseRevision)

			// The route is what this journey selects; SDD is what it must never
			// select. A delegated worker in particular must not promote itself.
			harness.assertNoSDDArtifacts()
		})
	}
}

// TestOrganicOptionalSDDDeclineAndAccept covers both answers to the one optional
// route question. Declining leaves the repository free of SDD state; accepting
// may use its OpenSpec state without archive routing being blocked by it.
func TestOrganicOptionalSDDDeclineAndAccept(t *testing.T) {
	t.Parallel()

	t.Run("declined", func(t *testing.T) {
		t.Parallel()
		harness := newOrganicHarness(t)
		harness.runActor(organicActorRoleDirect, "docs/declined.md", organicLines("declined line", 8), "docs: implement directly", organicDirectActorMarker)

		// This is the proposal's core claim, so it stays verbatim: direct and
		// delegated work never create SDD artifacts, prompts, phase attempts, or
		// synthetic SDD runs.
		harness.assertNoSDDArtifacts()
		if _, err := os.Stat(filepath.Join(harness.repo.worktree, "openspec")); !os.IsNotExist(err) {
			t.Fatalf("declined route created OpenSpec artifacts: %v", err)
		}
	})

	t.Run("accepted", func(t *testing.T) {
		t.Parallel()
		harness := newOrganicHarness(t)
		const change = "organic-accepted-change"
		harness.seedOrganicSDDChange(change)

		status := harness.sddStatus(change)
		if status.Dependencies.Archive == "blocked" {
			t.Fatalf("accepted SDD archive stayed blocked: reasons=%v", status.BlockedReasons)
		}
	})
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

type organicRepository struct {
	worktree     string
	bare         string
	baseRevision string
}

type organicHarness struct {
	t    *testing.T
	repo organicRepository
	home string
}

// newOrganicHarness builds the shared fixture every journey runs against: a
// seeded repository and an isolated HOME.
func newOrganicHarness(t *testing.T) *organicHarness {
	t.Helper()
	requireOrganicBinary(t)
	harness := &organicHarness{t: t, repo: initOrganicRepository(t), home: t.TempDir()}
	return harness
}

// environment isolates the run from the developer's own global state. A
// suite that reads the real user state would pass or fail for reasons that have
// nothing to do with the product.
func (harness *organicHarness) environment() []string {
	return organicEnvironment(harness.home)
}

func organicEnvironment(home string) []string {
	environment := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME=" + filepath.Join(home, ".local", "state"),
		"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
		"PATH=" + os.Getenv("PATH"),
		"LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		// CI makes the one-time consent question deterministically unanswerable,
		// which is exactly the non-interactive path this suite asserts on.
		"CI=1",
		// This whitelist does not inherit the TestMain opt-out, so it is set
		// explicitly: the real binary must never write the developer's
		// persistent Windows PATH.
		system.NoPersistentPathEnvVar + "=1",
	}
	if value := os.Getenv("SYSTEMROOT"); value != "" {
		environment = append(environment, "SYSTEMROOT="+value)
	}
	if value := os.Getenv("TMPDIR"); value != "" {
		environment = append(environment, "TMPDIR="+value)
	}
	for _, name := range []string{"OPENCODE_DISABLE_PROJECT_CONFIG", "OPENCODE_DISABLE_EXTERNAL_SKILLS"} {
		if value := os.Getenv(name); value != "" {
			environment = append(environment, name+"="+value)
		}
	}
	return environment
}

func TestOrganicEnvironmentDisablesPersistentPathWrites(t *testing.T) {
	want := system.NoPersistentPathEnvVar + "=1"
	for _, entry := range organicEnvironment(t.TempDir()) {
		if entry == want {
			return
		}
	}
	t.Fatalf("organicEnvironment() does not contain %q; journeys could write the real persistent PATH", want)
}

func (harness *organicHarness) axiom(arguments ...string) []byte {
	harness.t.Helper()
	stdout, stderr, err := runOrganicCommand(harness.t, organicBinary, harness.repo.worktree, harness.environment(), arguments...)
	if err != nil {
		harness.t.Fatalf("axiom %v: %v\nstdout:\n%s\nstderr:\n%s", arguments, err, stdout, stderr)
	}
	return []byte(stdout)
}

func runOrganicCommand(t *testing.T, binary, dir string, environment []string, arguments ...string) (string, string, error) {
	t.Helper()
	requireOrganicBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), organicLocalTimeout)
	defer cancel()
	command := organicCommandContext(ctx, binary, arguments...)
	command.Dir = dir
	command.Env = environment
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.String(), stderr.String(), err
}

// organicSDDStatus is the slice of `sdd-status --json` the archive journeys
// consume: the archive dependency and the routing.
type organicSDDStatus struct {
	Dependencies struct {
		Archive string `json:"archive"`
	} `json:"dependencies"`
	NextRecommended string   `json:"nextRecommended"`
	BlockedReasons  []string `json:"blockedReasons"`
}

func (harness *organicHarness) sddStatus(change string) organicSDDStatus {
	harness.t.Helper()
	payload := harness.axiom("sdd-status", change, "--cwd", harness.repo.worktree, "--json")
	var status organicSDDStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		harness.t.Fatalf("decode sdd-status: %v\n%s", err, payload)
	}
	return status
}

// organicSDDVerifyReport is the fenced envelope a completed independent
// verification writes. Its exact shape matters: a report the product cannot
// parse routes as "verification is missing", which is a different journey.
const organicSDDVerifyReport = "```yaml\n" +
	"schema: gentle-ai.verify-result/v1\n" +
	"evidence_revision: sha256:1111111111111111111111111111111111111111111111111111111111111111\n" +
	"verdict: pass\n" +
	"blockers: 0\n" +
	"critical_findings: 0\n" +
	"requirements: 1/1\n" +
	"scenarios: 1/1\n" +
	"test_command: go test ./internal/example\n" +
	"test_exit_code: 0\n" +
	"test_output_hash: sha256:2222222222222222222222222222222222222222222222222222222222222222\n" +
	"build_command: go test ./cmd/axiom\n" +
	"build_exit_code: 0\n" +
	"build_output_hash: sha256:3333333333333333333333333333333333333333333333333333333333333333\n" +
	"```\n"

// seedOrganicSDDChange commits a complete OpenSpec change at its archive
// decision: planning done, every task checked, and a parseable passing
// verification report.
func (harness *organicHarness) seedOrganicSDDChange(change string) {
	harness.t.Helper()
	root := "openspec/changes/" + change + "/"
	harness.writeFiles(map[string]string{
		root + "proposal.md": "# " + change + "\n\n## Why\n\nthe journey drives a delivery cycle.\n",
		root + "design.md":   "# design\n\n## Approach\n\nplain prose, no executable content.\n",
		root + "tasks.md":    "# tasks\n\n- [x] 1.1 write the prose\n",
		root + "specs/prose/spec.md": "### Requirement: prose exists\n" +
			"#### Scenario: prose is present\n\n- **WHEN** the reader opens the docs\n- **THEN** the prose is there\n",
		root + "verify-report.md": organicSDDVerifyReport,
	})
	// The change sits at its archive decision in a repository that stays off any archive branch, and a
	// change with no kickoff.yaml is not exempt from the status branch guard (S-7): the retro-sealed
	// kickoff without a git snapshot exempts it (REQ-25.37).
	retroSealConsentChange(harness.t, harness.repo.worktree, change)
	harness.git("add", "--", "openspec")
	harness.git("commit", "-q", "-m", "test: seed the SDD change")
}

// assertNoSDDArtifacts is the proposal's core claim and survives verbatim:
// direct and delegated work never create SDD, trace, or evaluation state.
func (harness *organicHarness) assertNoSDDArtifacts() {
	harness.t.Helper()
	if name, found := harness.sddArtifact(); found {
		harness.t.Fatalf("organic implementation created forbidden SDD/trace/evaluation artifact %q", name)
	}
}

func (harness *organicHarness) sddArtifact() (string, bool) {
	harness.t.Helper()
	root := filepath.Join(harness.commonDir(), "axiom")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return "", false
	}
	if err != nil {
		harness.t.Fatal(err)
	}
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.HasPrefix(name, "sdd-") || name == "sdd" || name == "trace" || name == "evaluation" {
			return filepath.Join(root, entry.Name()), true
		}
	}
	return "", false
}

// commonDir resolves the repository the way the product does, so an aliased or
// relative invocation cannot silently point the assertions at another clone.
func (harness *organicHarness) commonDir() string {
	harness.t.Helper()
	common := harness.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(harness.repo.worktree, common)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(common))
	if err != nil {
		harness.t.Fatal(err)
	}
	return resolved
}

func (harness *organicHarness) runActor(role, path, body, message, marker string) {
	harness.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), organicLocalTimeout)
	defer cancel()
	command := organicCommandContext(ctx, os.Args[0])
	command.Dir = harness.repo.worktree
	command.Env = append(harness.environment(),
		organicActorRoleEnvironment+"="+role,
		organicActorRepoEnvironment+"="+harness.repo.worktree,
		organicActorPathEnvironment+"="+path,
		organicActorBodyEnvironment+"="+body,
		organicActorMessageEnvironment+"="+message,
		organicActorBinaryEnvironment+"="+organicBinary,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		harness.t.Fatalf("%s actor: %v\nstdout:\n%s\nstderr:\n%s", role, err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), marker) {
		harness.t.Fatalf("%s actor did not report %q: %s", role, marker, stdout.String())
	}
}

// writeFiles writes candidate files and declares them. Since #2394 a new file
// only enters the review candidate once the user put it in the index, so a
// journey that means to have its files reviewed has to say so the same way a
// real user does.
func (harness *organicHarness) writeFiles(files map[string]string) {
	harness.t.Helper()
	for relative, body := range files {
		target := filepath.Join(harness.repo.worktree, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			harness.t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			harness.t.Fatal(err)
		}
		harness.git("add", "--", relative)
	}
}

func (harness *organicHarness) git(arguments ...string) string {
	harness.t.Helper()
	output, err := organicGitOutput(context.Background(), harness.repo.worktree, arguments...)
	if err != nil {
		harness.t.Fatal(err)
	}
	return output
}

func (harness *organicHarness) bareGit(arguments ...string) string {
	harness.t.Helper()
	output, err := organicBareGitOutput(context.Background(), harness.repo.bare, arguments...)
	if err != nil {
		harness.t.Fatal(err)
	}
	return output
}

// pushWithLease publishes under compare-and-swap against the exact revision the
// candidate was reviewed on top of.
func (harness *organicHarness) pushWithLease(expected string) {
	harness.t.Helper()
	harness.git("push", "--quiet", "--force-with-lease=refs/heads/main:"+expected, "origin", "HEAD:refs/heads/main")
	local := harness.git("rev-parse", "HEAD")
	if remote := harness.bareGit("rev-parse", "refs/heads/main"); remote != local {
		harness.t.Fatalf("remote ref = %s, want the delivered candidate %s", remote, local)
	}
}

// assertStaleLeaseIsRejected proves the publication really is a compare-and-swap.
// It needs something to publish, because an up-to-date push would succeed
// without ever consulting the lease and would prove nothing.
func (harness *organicHarness) assertStaleLeaseIsRejected(stale string) {
	harness.t.Helper()
	before := harness.bareGit("rev-parse", "refs/heads/main")
	harness.writeFiles(map[string]string{"docs/lease-probe.md": "lease probe\n"})
	harness.git("add", "--", "docs/lease-probe.md")
	harness.git("commit", "-q", "-m", "test: probe the publication lease")
	if _, err := organicGitOutput(
		context.Background(), harness.repo.worktree,
		"push", "--quiet", "--force-with-lease=refs/heads/main:"+stale, "origin", "HEAD:refs/heads/main",
	); err == nil {
		harness.t.Fatal("a stale compare-and-swap lease was accepted")
	}
	if after := harness.bareGit("rev-parse", "refs/heads/main"); after != before {
		harness.t.Fatalf("a rejected compare-and-swap still moved the remote: %s -> %s", before, after)
	}
	harness.git("reset", "--quiet", "--hard", before)
}

// assertRemoteBlob proves delivery reached the bare repository as exact content,
// not merely as a moved ref.
func (harness *organicHarness) assertRemoteBlob(path, body string) {
	harness.t.Helper()
	entry := harness.bareGit("ls-tree", "refs/heads/main", "--", path)
	if entry == "" {
		harness.t.Fatalf("delivered path %q is absent from the remote tree", path)
	}
	fields := strings.Fields(entry)
	if len(fields) < 3 {
		harness.t.Fatalf("unreadable remote tree entry %q", entry)
	}
	if fields[0] != "100644" {
		harness.t.Fatalf("delivered mode = %q, want 100644", fields[0])
	}
	blob := harness.bareGit("cat-file", "blob", fields[2])
	if blob != strings.TrimRight(body, "\n") {
		harness.t.Fatalf("delivered blob content differs:\nwant:\n%s\ngot:\n%s", body, blob)
	}
	tree := harness.bareGit("rev-parse", "refs/heads/main^{tree}")
	if localTree := harness.git("rev-parse", "HEAD^{tree}"); tree != localTree {
		harness.t.Fatalf("delivered tree = %s, want the reviewed tree %s", tree, localTree)
	}
}

func (harness *organicHarness) assertOnlyMainRef() {
	harness.t.Helper()
	refs := harness.bareGit("for-each-ref", "--format=%(refname)")
	if refs != "refs/heads/main" {
		harness.t.Fatalf("bare repository refs = %q, want only refs/heads/main", refs)
	}
}

// ---------------------------------------------------------------------------
// Wire projections
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Repository fixtures and shared utilities
// ---------------------------------------------------------------------------

func initOrganicRepository(t *testing.T) organicRepository {
	t.Helper()
	repo := t.TempDir()
	for _, arguments := range [][]string{
		{"init", "--quiet", "--initial-branch=main", "."},
		{"config", "user.name", "Organic E2E"},
		{"config", "user.email", "organic-e2e@example.invalid"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := organicGitOutput(context.Background(), repo, arguments...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("organic runtime\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := organicGitOutput(context.Background(), repo, "add", "--", "tracked.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := organicGitOutput(context.Background(), repo, "commit", "-q", "-m", "test: seed the organic repository"); err != nil {
		t.Fatal(err)
	}
	baseRevision, err := organicGitOutput(context.Background(), repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	bare := filepath.Join(t.TempDir(), "origin.git")
	if _, err := organicGitOutput(context.Background(), repo, "init", "--bare", "--quiet", bare); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{
		{"remote", "add", "origin", bare},
		{"push", "--quiet", "--set-upstream", "origin", "main:refs/heads/main"},
	} {
		if _, err := organicGitOutput(context.Background(), repo, arguments...); err != nil {
			t.Fatal(err)
		}
	}
	if err := requireOrganicOnlyMainRef(context.Background(), bare); err != nil {
		t.Fatal(err)
	}
	return organicRepository{worktree: repo, bare: bare, baseRevision: baseRevision}
}

func organicGitOutput(parent context.Context, repo string, arguments ...string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, organicLocalTimeout)
	defer cancel()
	command := organicCommandContext(ctx, "git", append([]string{"-C", repo}, arguments...)...)
	command.Env = append(os.Environ(), "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git -C %q %v: %w\n%s", repo, arguments, err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func organicBareGitOutput(parent context.Context, bare string, arguments ...string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, organicLocalTimeout)
	defer cancel()
	command := organicCommandContext(ctx, "git", append([]string{"--git-dir=" + bare}, arguments...)...)
	command.Env = append(os.Environ(), "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git --git-dir=%q %v: %w\n%s", bare, arguments, err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func requireOrganicOnlyMainRef(parent context.Context, bare string) error {
	refs, err := organicBareGitOutput(parent, bare, "for-each-ref", "--format=%(refname)")
	if err != nil {
		return err
	}
	if refs != "refs/heads/main" {
		return fmt.Errorf("bare repository refs = %q, want only refs/heads/main", refs)
	}
	return nil
}

func sameOrganicDirectory(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && leftInfo.IsDir() && rightInfo.IsDir() && os.SameFile(leftInfo, rightInfo)
}

func organicCommandContext(ctx context.Context, name string, arguments ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, name, arguments...)
	command.WaitDelay = organicCommandWaitDelay
	return command
}

func organicModuleRoot() (string, error) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("resolve the organic test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..")), nil
}

// buildOrganicBinary compiles the product once for the whole package. Every
// journey drives that one binary, so a per-journey build would only buy slower
// feedback for the same proof.
func buildOrganicBinary(workspace string) (string, error) {
	moduleRoot, err := organicModuleRoot()
	if err != nil {
		return "", err
	}
	// Build the canonical binary. The retired gentle-ai package printed a
	// notice to stderr on every invocation, and several journeys assert that a
	// silent transition writes nothing to stderr.
	name := "axiom"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(workspace, name)
	ctx, cancel := context.WithTimeout(context.Background(), organicSetupTimeout)
	defer cancel()
	command := organicCommandContext(ctx, "go", "build", "-trimpath", "-o", path, "./cmd/axiom")
	command.Dir = moduleRoot
	command.Env = os.Environ()
	if output, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build the axiom test binary: %w\n%s", err, output)
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("built axiom binary %q is unusable: %v", path, err)
	}
	return path, nil
}

func organicLines(prefix string, count int) string {
	var builder strings.Builder
	for index := 1; index <= count; index++ {
		fmt.Fprintf(&builder, "%s %03d\n", prefix, index)
	}
	return builder.String()
}

// ---------------------------------------------------------------------------
// Pinned real-agent journeys
// ---------------------------------------------------------------------------

// TestRealAgentOrganicJourneys runs the same organic journeys through a real
// configured agent. The agent runtime, its sub-agent mechanism, its tool calls,
// the axiom binary, and the repository are all real; only the model is a
// fixture, because a scripted model is what makes an agent journey repeatable.
func TestRealAgentOrganicJourneys(t *testing.T) {
	if os.Getenv(realAgentE2EEnvironment) != "1" {
		t.Skip("set AXIOM_REAL_AGENT_E2E=1 to run the pinned real-agent journeys")
	}
	requireOrganicExecutableVersion(t, "opencode", pinnedOpenCodeVersion)
	sharedConfig := prepareOpenCodeConfig(t)
	sharedCache := t.TempDir()

	tests := []struct {
		name         string
		outcome      string
		role         string
		marker       string
		path         string
		delegated    bool
		actorPrompt  string
		wantSubagent bool
	}{
		{
			name:    "direct inline implementation",
			outcome: "Apply one already-understood mechanical documentation change and deliver it.",
			role:    organicActorRoleDirect,
			marker:  organicDirectActorMarker,
			path:    "docs/real-direct.md",
		},
		{
			name:      "delegated direct implementation",
			outcome:   "Understand the documentation set, implement the bounded outcome, and deliver it.",
			role:      organicActorRoleDelegated,
			marker:    organicDelegatedActorMarker,
			path:      "docs/real-delegated.md",
			delegated: true,
			actorPrompt: "Act as the delegated-direct implementation worker. Implement the exact " +
				"admitted documentation scope, explicitly commit it, and return exactly " +
				organicDelegatedActorMarker + ". Never propose or create SDD state.",
			wantSubagent: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newOrganicHarness(t)
			body := organicLines("real agent line", 10)

			script := []openCodeTurn{
				{tool: "bash", arguments: map[string]any{"command": organicActorToolCommand(t)}},
			}
			if test.delegated {
				// The implementation step becomes a real sub-agent, and only the
				// sub-agent may commit the candidate.
				script[0] = openCodeTurn{tool: "task", arguments: map[string]any{
					"description":   "Run the delegated organic actor",
					"prompt":        test.actorPrompt,
					"subagent_type": "general",
				}}
			}

			model := newOpenCodeFixtureServer(t, script, test.actorPrompt)
			defer model.Close()

			home := t.TempDir()
			environment := append(harness.environment(),
				"XDG_CONFIG_HOME="+sharedConfig,
				"XDG_CACHE_HOME="+sharedCache,
				"OPENCODE_CONFIG_DIR="+filepath.Join(sharedConfig, "opencode"),
				"OPENCODE_TEST_HOME="+filepath.Join(home, "opencode"),
				"OPENCODE_CONFIG_CONTENT="+organicOpenCodeConfig(t, model.URL),
				"OPENCODE_AUTH_CONTENT={}",
				"OPENCODE_DISABLE_PROJECT_CONFIG=1",
				"OPENCODE_DISABLE_AUTOUPDATE=1",
				"OPENCODE_DISABLE_AUTOCOMPACT=1",
				"OPENCODE_DISABLE_CLAUDE_CODE=1",
				"OPENCODE_DISABLE_DEFAULT_PLUGINS=1",
				"OPENCODE_DISABLE_EXTERNAL_SKILLS=1",
				"OPENCODE_DISABLE_LSP_DOWNLOAD=1",
				"OPENCODE_DISABLE_MODELS_FETCH=1",
				"OPENCODE_EXPERIMENTAL_DISABLE_FILEWATCHER=1",
				"OPENCODE_FAST_BOOT=1",
				"OPENCODE_PURE=1",
				organicActorRoleEnvironment+"="+test.role,
				organicActorRepoEnvironment+"="+harness.repo.worktree,
				organicActorPathEnvironment+"="+test.path,
				organicActorBodyEnvironment+"="+body,
				organicActorMessageEnvironment+"=docs: implement the real-agent outcome",
				organicActorBinaryEnvironment+"="+organicBinary,
				"AXIOM_ORGANIC_ACTOR_EXECUTABLE="+os.Args[0],
				"AXIOM_ORGANIC_BINARY="+organicBinary,
			)

			ctx, cancel := context.WithTimeout(context.Background(), organicAgentTimeout)
			defer cancel()
			command := organicCommandContext(ctx, "opencode", "run", "--pure",
				"--format", "json", "--agent", "organic", "--model", "fixture/fixture",
				"--dir", harness.repo.worktree, test.outcome,
			)
			command.Dir = harness.repo.worktree
			command.Env = environment
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			if err := command.Run(); err != nil {
				t.Fatalf("opencode run: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
			}
			model.assertComplete(t, test.wantSubagent)

			transcript := stdout.String()
			if !strings.Contains(transcript, test.marker) {
				t.Fatalf("the real agent never reported %q:\n%s", test.marker, transcript)
			}
			if harness.git("rev-parse", "HEAD") == harness.repo.baseRevision {
				t.Fatal("the real agent never created a candidate commit")
			}
			// A real sub-agent must not escalate its own route either.
			harness.assertNoSDDArtifacts()
		})
	}
}

func TestRealAgentInstalledSDDApplyExecutorDoesNotDelegate(t *testing.T) {
	if os.Getenv(realAgentE2EEnvironment) != "1" {
		t.Skip("set AXIOM_REAL_AGENT_E2E=1 to run the pinned real-agent journeys")
	}
	requireOrganicExecutableVersion(t, "opencode", pinnedOpenCodeVersion)

	configRoot := prepareOpenCodeConfig(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	const (
		executorPrompt               = "Execute the assigned SDD apply phase without delegation."
		executorCompletionPrefix     = "SDD_APPLY_EXECUTOR_COMPLETED:"
		orchestratorCompletionMarker = "SDD_APPLY_ORCHESTRATOR_COMPLETED"
	)
	executorNonce := fmt.Sprintf("sdd-apply-executor-nonce-%d", time.Now().UnixNano())
	fixture := newOpenCodeFixtureServer(t, []openCodeTurn{
		{tool: "task", arguments: map[string]any{
			"description":   "Run the installed SDD apply executor",
			"prompt":        executorPrompt,
			"subagent_type": "sdd-apply",
		}},
	}, executorPrompt)
	defer fixture.Close()
	fixture.requireInstalledSDDApplyExecutor = true
	fixture.executorNonce = executorNonce
	fixture.executorCompletionPrefix = executorCompletionPrefix
	fixture.completion = orchestratorCompletionMarker
	fixture.actorCommand = "echo " + executorNonce

	settingsPath := filepath.Join(configRoot, "opencode", "opencode.json")
	if err := os.WriteFile(settingsPath, []byte(organicOpenCodeConfig(t, fixture.URL)), 0o600); err != nil {
		t.Fatalf("write OpenCode fixture config: %v", err)
	}
	if _, err := sdd.Inject(home, opencode.NewAdapter(), model.SDDModeMulti); err != nil {
		t.Fatalf("install SDD OpenCode assets: %v", err)
	}

	workdir := t.TempDir()
	environment := append(os.Environ(),
		"OPENCODE_CONFIG_DIR="+filepath.Join(configRoot, "opencode"),
		"OPENCODE_TEST_HOME="+filepath.Join(home, "opencode"),
		"OPENCODE_AUTH_CONTENT={}",
		"OPENCODE_DISABLE_PROJECT_CONFIG=1",
		"OPENCODE_DISABLE_AUTOUPDATE=1",
		"OPENCODE_DISABLE_AUTOCOMPACT=1",
		"OPENCODE_DISABLE_CLAUDE_CODE=1",
		"OPENCODE_DISABLE_DEFAULT_PLUGINS=1",
		"OPENCODE_DISABLE_EXTERNAL_SKILLS=1",
		"OPENCODE_DISABLE_LSP_DOWNLOAD=1",
		"OPENCODE_DISABLE_MODELS_FETCH=1",
		"OPENCODE_EXPERIMENTAL_DISABLE_FILEWATCHER=1",
		"OPENCODE_FAST_BOOT=1",
		"OPENCODE_PURE=1",
	)
	ctx, cancel := context.WithTimeout(context.Background(), organicAgentTimeout)
	defer cancel()
	command := organicCommandContext(ctx, "opencode", "run", "--pure",
		"--format", "json", "--agent", "gentle-orchestrator", "--model", "fixture/fixture",
		"--dir", workdir, "Delegate the assigned phase to the installed sdd-apply executor.",
	)
	command.Dir = workdir
	command.Env = environment
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("run installed sdd-apply executor: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	fixture.assertComplete(t, true)
	fixture.assertInstalledSDDApplyExecutorProof(t)
	if !strings.Contains(stdout.String(), orchestratorCompletionMarker) {
		t.Fatalf("orchestrator did not complete after the executor result round trip:\n%s", stdout.String())
	}
}

func TestInstalledSDDApplyExecutorProofRejectsOrchestratorSpoof(t *testing.T) {
	const (
		nonce            = "sdd-apply-executor-nonce-spoof-control"
		executorComplete = "SDD_APPLY_EXECUTOR_COMPLETED:" + nonce
	)
	fixture := &openCodeFixtureServer{
		requireInstalledSDDApplyExecutor: true,
		executorNonce:                    nonce,
		executorCompletionPrefix:         "SDD_APPLY_EXECUTOR_COMPLETED:",
		executorSubagentResult:           executorComplete,
	}
	recorder := httptest.NewRecorder()
	input := openAIRequest{Messages: []openAIMessage{{Role: "tool", Content: executorComplete}}}

	if fixture.acceptInstalledSDDApplyExecutorRoundTrip(recorder, input) {
		t.Fatal("an orchestrator-spoofed executor completion was accepted without an executor bash result")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("spoof response status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(fixture.failure, "without executor bash result") {
		t.Fatalf("spoof refusal = %q, want missing executor bash result", fixture.failure)
	}
}

func TestInstalledSDDApplyExecutorRoundTripRejectsMissingCredentials(t *testing.T) {
	tests := []struct {
		name   string
		nonce  string
		prefix string
	}{
		{name: "empty nonce", prefix: "SDD_APPLY_EXECUTOR_COMPLETED:"},
		{name: "empty completion prefix", nonce: "sdd-apply-executor-nonce-missing-prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const executorResult = "SDD_APPLY_EXECUTOR_COMPLETED:fixture-result"
			fixture := &openCodeFixtureServer{
				executorNonce:            tt.nonce,
				executorCompletionPrefix: tt.prefix,
				executorBashResult:       tt.nonce,
				executorSubagentResult:   executorResult,
			}
			recorder := httptest.NewRecorder()
			input := openAIRequest{Messages: []openAIMessage{{Role: "tool", Content: executorResult}}}

			if fixture.acceptInstalledSDDApplyExecutorRoundTrip(recorder, input) {
				t.Fatal("round trip accepted missing executor proof credentials")
			}
			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("response status = %d, want %d", recorder.Code, http.StatusInternalServerError)
			}
			if !strings.Contains(fixture.failure, "missing nonce or completion marker") {
				t.Fatalf("refusal = %q, want missing credentials", fixture.failure)
			}
		})
	}
}

func TestInstalledSDDApplyExecutorRoundTripRejectsUnrelatedBashOutput(t *testing.T) {
	const (
		nonce            = "sdd-apply-executor-nonce-unrelated-output"
		executorComplete = "SDD_APPLY_EXECUTOR_COMPLETED:" + nonce
	)
	fixture := &openCodeFixtureServer{
		executorNonce:            nonce,
		executorCompletionPrefix: "SDD_APPLY_EXECUTOR_COMPLETED:",
		executorBashResult:       "completed a different command successfully",
		executorSubagentResult:   executorComplete,
	}
	recorder := httptest.NewRecorder()
	input := openAIRequest{Messages: []openAIMessage{{Role: "tool", Content: executorComplete}}}

	if fixture.acceptInstalledSDDApplyExecutorRoundTrip(recorder, input) {
		t.Fatal("round trip accepted non-empty bash output without the executor nonce")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("response status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(fixture.failure, "without executor bash result") {
		t.Fatalf("refusal = %q, want nonce-bound executor bash result", fixture.failure)
	}
}

// organicActorToolCommand runs the compiled actor process from the agent's own
// bash tool, so the implementation step is a real child process of a real agent.
func organicActorToolCommand(t *testing.T) string {
	t.Helper()
	return organicToolCommand(t, "AXIOM_ORGANIC_ACTOR_EXECUTABLE")
}

// organicToolCommand turns one fixture-authored argv into the string the
// agent's bash tool executes, without round-tripping the argv through a shell
// flavour we do not control.
//
// On POSIX systems the agent's tool shell is sh-compatible, so the argv is
// authored directly in sh syntax — byte-identical to what the journeys always
// ran on Linux. On Windows the agent's tool shell is PowerShell (or cmd),
// neither of which parses POSIX single-quoting (PowerShell fails with
// "ParserError: Unexpected token" on a single-quoted argument such as
// 'docs'), so the argv is baked by Go into a generated .cmd trampoline
// and the command becomes that script's bare path: a single token that
// PowerShell and cmd both resolve as a plain invocation.
func organicToolCommand(t *testing.T, binaryEnvironment string, arguments ...string) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		command := `"$` + binaryEnvironment + `"`
		for _, argument := range arguments {
			command += " '" + strings.ReplaceAll(argument, "'", `'\''`) + "'"
		}
		return command
	}

	invocation := `"%` + binaryEnvironment + `%"`
	for _, argument := range arguments {
		if strings.ContainsAny(argument, `"%`) {
			t.Fatalf("tool argument %q cannot be embedded safely in a cmd trampoline", argument)
		}
		invocation += ` "` + argument + `"`
	}
	script := filepath.Join(t.TempDir(), "tool.cmd")
	if strings.ContainsAny(script, " \t'\"") {
		t.Fatalf("tool trampoline path %q would itself need shell quoting", script)
	}
	content := strings.Join([]string{"@echo off", invocation, "exit /b %ERRORLEVEL%"}, "\r\n") + "\r\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
	return script
}

type openCodeTurn struct {
	tool      string
	arguments map[string]any
}

type openCodeFixtureServer struct {
	*httptest.Server
	mu                               sync.Mutex
	script                           []openCodeTurn
	actorPrompt                      string
	actorCommand                     string
	completion                       string
	subagentCompletion               string
	requireInstalledSDDApplyExecutor bool
	sawInstalledSDDApplyExecutor     bool
	executorNonce                    string
	executorCompletionPrefix         string
	executorBashResult               string
	executorSubagentResult           string
	executorRoundTripResult          string
	mainCalls                        int
	subagentStarts                   int
	failure                          string
}

func newOpenCodeFixtureServer(t *testing.T, script []openCodeTurn, actorPrompt string) *openCodeFixtureServer {
	t.Helper()
	fixture := &openCodeFixtureServer{
		script:      script,
		actorPrompt: actorPrompt,
		// Precomputed with the test handle: the HTTP handler that serves the
		// delegated worker turn has no *testing.T of its own.
		actorCommand: organicActorToolCommand(t),
	}
	fixture.Server = httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	return fixture
}

type openAIRequest struct {
	Messages []openAIMessage   `json:"messages"`
	Tools    []json.RawMessage `json:"tools"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func (fixture *openCodeFixtureServer) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method", http.StatusMethodNotAllowed)
		return
	}
	var input openAIRequest
	if err := json.NewDecoder(io.LimitReader(request.Body, 8<<20)).Decode(&input); err != nil {
		fixture.fail(writer, "decode model request: %v", err)
		return
	}
	if len(input.Tools) == 0 {
		fixture.writeText(writer, "Organic runtime journey", "stop")
		return
	}
	if fixture.isSubagent(input) {
		if fixture.requireInstalledSDDApplyExecutor && !fixture.acceptInstalledSDDApplyExecutor(writer, input) {
			return
		}
		fixture.mu.Lock()
		fixture.subagentStarts++
		fixture.mu.Unlock()
		last := input.Messages[len(input.Messages)-1]
		if last.Role == "tool" {
			if fixture.requireInstalledSDDApplyExecutor && !fixture.captureInstalledSDDApplyExecutorBashResult(writer, messageText(last.Content)) {
				return
			}
			completion := fixture.subagentCompletion
			if fixture.requireInstalledSDDApplyExecutor {
				fixture.mu.Lock()
				completion = fixture.executorSubagentResult
				fixture.mu.Unlock()
			}
			if completion == "" {
				completion = organicDelegatedActorMarker
			}
			fixture.writeText(writer, completion, "stop")
			return
		}
		fixture.writeTool(writer, "delegated-actor", "bash", map[string]any{"command": fixture.actorCommand})
		return
	}

	fixture.mu.Lock()
	fixture.mainCalls++
	call := fixture.mainCalls
	fixture.mu.Unlock()
	if call > len(fixture.script) {
		if fixture.requireInstalledSDDApplyExecutor && !fixture.acceptInstalledSDDApplyExecutorRoundTrip(writer, input) {
			return
		}
		completion := fixture.completion
		if completion == "" {
			completion = "Organic journey complete."
		}
		fixture.writeText(writer, completion, "stop")
		return
	}
	turn := fixture.script[call-1]
	fixture.writeTool(writer, fmt.Sprintf("turn-%d", call), turn.tool, turn.arguments)
}

func (fixture *openCodeFixtureServer) acceptInstalledSDDApplyExecutor(writer http.ResponseWriter, input openAIRequest) bool {
	var transcript strings.Builder
	for _, message := range input.Messages {
		transcript.WriteString(messageText(message.Content))
	}
	content := transcript.String()
	// The proof asserts the role contract the executor actually received, not
	// one wording of it. Ordering between two blocks is no longer the property
	// under test: a single block whose every imperative follows its own
	// condition is, and the executor branch must come first because these
	// skills are delegate_only and the sub-agent is the intended reader.
	role := strings.Index(content, "## Execution Role")
	if role < 0 {
		fixture.fail(writer, "installed sdd-apply executor did not load its Execution Role block")
		return false
	}
	executor := strings.Index(content, "If you are the `sdd-apply` sub-agent")
	orchestrator := strings.Index(content, "If you loaded this skill through the `skill()` tool")
	if executor < 0 || orchestrator < 0 || executor > orchestrator {
		fixture.fail(writer, "installed sdd-apply executor role block does not state the executor branch before the orchestrator branch")
		return false
	}
	if !strings.Contains(content, "continue with the phase work below. Do not delegate. Do not call the Skill tool.") {
		fixture.fail(writer, "installed sdd-apply executor is missing its non-delegating continuation instruction")
		return false
	}
	for _, retraction := range []string{"does NOT apply to you", "the gate above", "the gate below"} {
		if strings.Contains(content, retraction) {
			fixture.fail(writer, "installed sdd-apply executor received a retraction phrase %q that undoes an earlier imperative", retraction)
			return false
		}
	}
	for _, rawTool := range input.Tools {
		var tool struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if err := json.Unmarshal(rawTool, &tool); err != nil {
			fixture.fail(writer, "decode installed sdd-apply tool definition: %v", err)
			return false
		}
		if tool.Function.Name == "task" {
			fixture.fail(writer, "installed sdd-apply executor was offered the task delegation tool")
			return false
		}
	}
	fixture.mu.Lock()
	fixture.sawInstalledSDDApplyExecutor = true
	fixture.mu.Unlock()
	return true
}

func (fixture *openCodeFixtureServer) captureInstalledSDDApplyExecutorBashResult(writer http.ResponseWriter, result string) bool {
	fixture.mu.Lock()
	nonce := fixture.executorNonce
	prefix := fixture.executorCompletionPrefix
	fixture.mu.Unlock()
	if nonce == "" || prefix == "" {
		fixture.fail(writer, "installed sdd-apply executor proof is missing its nonce or completion marker")
		return false
	}
	if !strings.Contains(result, nonce) {
		fixture.fail(writer, "installed sdd-apply executor bash result does not contain its nonce")
		return false
	}
	fixture.mu.Lock()
	fixture.executorBashResult = result
	fixture.executorSubagentResult = prefix + nonce
	fixture.mu.Unlock()
	return true
}

func (fixture *openCodeFixtureServer) acceptInstalledSDDApplyExecutorRoundTrip(writer http.ResponseWriter, input openAIRequest) bool {
	fixture.mu.Lock()
	nonce := fixture.executorNonce
	prefix := fixture.executorCompletionPrefix
	bashResult := fixture.executorBashResult
	executorResult := fixture.executorSubagentResult
	fixture.mu.Unlock()
	if nonce == "" || prefix == "" {
		fixture.fail(writer, "installed sdd-apply executor proof is missing nonce or completion marker")
		return false
	}
	if !strings.Contains(bashResult, nonce) {
		fixture.fail(writer, "orchestrator cannot complete the executor proof without executor bash result")
		return false
	}
	if executorResult == "" {
		fixture.fail(writer, "installed sdd-apply executor did not return a nonce-bound subagent result")
		return false
	}
	for index := len(input.Messages) - 1; index >= 0; index-- {
		message := input.Messages[index]
		if message.Role != "tool" {
			continue
		}
		result := messageText(message.Content)
		if strings.Contains(result, executorResult) && strings.Contains(result, nonce) {
			fixture.mu.Lock()
			fixture.executorRoundTripResult = result
			fixture.mu.Unlock()
			return true
		}
	}
	fixture.fail(writer, "orchestrator did not receive the nonce-bound executor subagent result")
	return false
}

func (fixture *openCodeFixtureServer) assertInstalledSDDApplyExecutorProof(t *testing.T) {
	t.Helper()
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	nonce := fixture.executorNonce
	prefix := fixture.executorCompletionPrefix
	if nonce == "" || prefix == "" {
		t.Fatal("installed sdd-apply executor proof is missing nonce or completion marker")
	}
	executorResult := prefix + nonce
	if fixture.completion == executorResult {
		t.Fatal("orchestrator and executor completion markers must be distinct")
	}
	if !strings.Contains(fixture.actorCommand, nonce) {
		t.Fatal("the installed sdd-apply executor did not receive its nonce-bearing bash command")
	}
	if !strings.Contains(fixture.executorBashResult, nonce) {
		t.Fatal("the installed sdd-apply executor did not return its bash-produced nonce")
	}
	if fixture.executorSubagentResult != executorResult {
		t.Fatalf("executor subagent result = %q, want %q", fixture.executorSubagentResult, executorResult)
	}
	if !strings.Contains(fixture.executorRoundTripResult, executorResult) {
		t.Fatal("orchestrator did not receive the executor result through its task round trip")
	}
}

// isSubagent recognises the delegated worker session. OpenCode gives the
// sub-agent its own conversation seeded with the delegation prompt, so the
// prompt's presence in a user message is what distinguishes the two sessions.
func (fixture *openCodeFixtureServer) isSubagent(input openAIRequest) bool {
	if strings.TrimSpace(fixture.actorPrompt) == "" {
		return false
	}
	for _, message := range input.Messages {
		if message.Role == "user" && strings.Contains(messageText(message.Content), fixture.actorPrompt) {
			return true
		}
	}
	return false
}

func (fixture *openCodeFixtureServer) fail(writer http.ResponseWriter, format string, arguments ...any) {
	fixture.mu.Lock()
	fixture.failure = fmt.Sprintf(format, arguments...)
	fixture.mu.Unlock()
	http.Error(writer, "fixture failure", http.StatusInternalServerError)
}

func (fixture *openCodeFixtureServer) writeTool(writer http.ResponseWriter, id, name string, arguments any) {
	encoded, _ := json.Marshal(arguments)
	fixture.writeChunks(writer, []any{
		map[string]any{
			"id": "chat", "object": "chat.completion.chunk", "created": 0, "model": "fixture",
			"choices": []any{map[string]any{
				"index": 0,
				"delta": map[string]any{
					"role": "assistant",
					"tool_calls": []any{map[string]any{
						"index": 0, "id": "call_" + id, "type": "function",
						"function": map[string]any{"name": name, "arguments": string(encoded)},
					}},
				},
				"finish_reason": nil,
			}},
		},
		organicFinishChunk("tool_calls"),
	})
}

func (fixture *openCodeFixtureServer) writeText(writer http.ResponseWriter, content, reason string) {
	fixture.writeChunks(writer, []any{
		map[string]any{
			"id": "chat", "object": "chat.completion.chunk", "created": 0, "model": "fixture",
			"choices": []any{map[string]any{
				"index":         0,
				"delta":         map[string]any{"role": "assistant", "content": content},
				"finish_reason": nil,
			}},
		},
		organicFinishChunk(reason),
	})
}

func organicFinishChunk(reason string) map[string]any {
	return map[string]any{
		"id": "chat", "object": "chat.completion.chunk", "created": 0, "model": "fixture",
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": reason}},
		"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
	}
}

func (fixture *openCodeFixtureServer) writeChunks(writer http.ResponseWriter, chunks []any) {
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)
	for _, chunk := range chunks {
		encoded, _ := json.Marshal(chunk)
		_, _ = fmt.Fprintf(writer, "data: %s\n\n", encoded)
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
	}
	_, _ = io.WriteString(writer, "data: [DONE]\n\n")
}

func (fixture *openCodeFixtureServer) assertComplete(t *testing.T, wantSubagent bool) {
	t.Helper()
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if fixture.failure != "" {
		t.Fatal(fixture.failure)
	}
	if fixture.mainCalls < len(fixture.script) {
		t.Fatalf("the agent issued %d of %d scripted turns", fixture.mainCalls, len(fixture.script))
	}
	if hadSubagent := fixture.subagentStarts > 0; hadSubagent != wantSubagent {
		t.Fatalf("real sub-agent used = %t, want %t", hadSubagent, wantSubagent)
	}
	if fixture.requireInstalledSDDApplyExecutor && !fixture.sawInstalledSDDApplyExecutor {
		t.Fatal("the installed sdd-apply executor never loaded its runtime prompt")
	}
}

func organicOpenCodeConfig(t *testing.T, serverURL string) string {
	t.Helper()
	config := map[string]any{
		"provider": map[string]any{
			"fixture": map[string]any{
				"npm":     "@ai-sdk/openai-compatible",
				"name":    "Organic E2E Fixture",
				"options": map[string]any{"baseURL": serverURL + "/v1", "apiKey": "fixture"},
				"models":  map[string]any{"fixture": map[string]any{"name": "Fixture"}},
			},
		},
		"agent": map[string]any{
			"organic": map[string]any{
				"description": "Organic runtime E2E",
				"mode":        "primary",
				"model":       "fixture/fixture",
				"permission":  map[string]any{"bash": "allow", "task": "allow", "edit": "deny"},
			},
		},
		"plugin":     []any{},
		"compaction": map[string]any{"auto": false},
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func prepareOpenCodeConfig(t *testing.T) string {
	t.Helper()
	requireOrganicExecutable(t, "npm")
	root := t.TempDir()
	config := filepath.Join(root, "opencode")
	if err := os.MkdirAll(config, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"private":true,"dependencies":{"@opencode-ai/plugin":"` + pinnedOpenCodeVersion + `"}}` + "\n")
	if err := os.WriteFile(filepath.Join(config, "package.json"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), organicSetupTimeout)
	defer cancel()
	command := organicCommandContext(ctx, "npm", "install", "--ignore-scripts", "--no-audit", "--no-fund", "--package-lock=false", "--prefix", config)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("prepare the pinned OpenCode plugin: %v\n%s", err, output)
	}
	return root
}

func requireOrganicExecutable(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Fatalf("required executable %s: %v", name, err)
	}
}

func requireOrganicExecutableVersion(t *testing.T, name, expected string) {
	t.Helper()
	requireOrganicExecutable(t, name)
	ctx, cancel := context.WithTimeout(context.Background(), organicLocalTimeout)
	defer cancel()
	command := organicCommandContext(ctx, name, "--version")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s --version: %v\n%s", name, err, output)
	}
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("%s version = %q, want %q", name, strings.TrimSpace(string(output)), expected)
	}
}

func messageText(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []any:
		var builder strings.Builder
		for _, part := range value {
			encoded, _ := json.Marshal(part)
			builder.Write(encoded)
		}
		return builder.String()
	default:
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
}

// Trace the real host once; -ff covers descendants, including relay version
// probes. A PATH wrapper would recursively invoke strace in an already traced
// child. The real OpenCode remains on PATH for the production capability check.
func openCodeTraceCommand(ctx context.Context, tracer, binary, traceBase string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, tracer, append([]string{"-ff", "-o", traceBase, "-e", "trace=connect", binary}, args...)...)
}
