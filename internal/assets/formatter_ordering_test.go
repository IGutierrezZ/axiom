package assets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/testenv"
	"github.com/IGutierrezZ/axiom/v3/internal/versions"
)

// TestSDDOrchestratorsCarryNoFormatterOrderingContract: the normalization ordering
// rule only protected the frozen candidate of the native review, which is
// suspended (C-01-c), so no orchestrator projects it.
func TestSDDOrchestratorsCarryNoFormatterOrderingContract(t *testing.T) {
	for _, path := range allSDDOrchestratorAssetPaths(t) {
		t.Run(path, func(t *testing.T) {
			content := mustReadResolved(path)
			for _, retired := range []string{
				"Normalization ordering rule",
				"before review START and its identity freeze",
				"any byte, path, or mode change invalidates the receipt",
			} {
				if strings.Contains(content, retired) {
					t.Fatalf("%s retains formatter-ordering contract %q", path, retired)
				}
			}
		})
	}
}

func TestRequiredChecksFailClosedWhenFormatFails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	guard := "    steps:\n" +
		"      - name: Require successful Go format\n" +
		"        if: needs.go-format.result != 'success'\n" +
		"        run: exit 1\n"

	jobs := []struct {
		id, next string
		// needs and ifTokens override the manual-only default for a job whose
		// trigger legitimately changed. ifTokens must all appear in the job's
		// `if:` expression, so the fail-closed guard below stays mandatory.
		needs    string
		ifTokens []string
	}{
		// unit-tests is omitted on a PR that only touches planning artifacts
		// (odd/, openspec/, docs/plan/), so it uses !cancelled() instead of
		// always(). A red go-format still runs it and the guard still fails it.
		// It is the aggregator of the parallel unit jobs; its children do not
		// wait for go-format, so this guard is what keeps format mandatory.
		{
			id:    "unit-tests",
			next:  "windows-runtime",
			needs: "[go-format, changes, go-tests, binary-smoke, deadcode-ratchet]",
			ifTokens: []string{
				"!cancelled()",
				"needs.changes.outputs.artifacts_only != 'true'",
			},
		},
		// The sharded Windows full suite is deliberately NOT here: it lives in
		// its own workflow while the pre-existing Windows failures it exposed
		// are paid down, so it is not one of ci.yml's required checks and owes
		// no fail-closed contract to a go-format job in another file.
		//
		// windows-runtime is no longer manual-only: it also runs when the diff
		// touches Windows paths (the `changes` job) or the PR carries the
		// ci:windows label. It still owes the format guard, so that a red
		// go-format cannot be bypassed by a Windows run that was selected.
		{
			id:    "windows-runtime",
			next:  "darwin-runtime",
			needs: "[go-format, changes]",
			ifTokens: []string{
				"!cancelled()",
				"github.event_name == 'workflow_dispatch'",
				"needs.changes.outputs.windows == 'true'",
				"needs.changes.result != 'success'",
				"contains(github.event.pull_request.labels.*.name, 'ci:windows')",
			},
		},
		// darwin-runtime is a required check like its siblings, so it owes the
		// same fail-closed contract. Listing it here is also what keeps the
		// section slicing honest: a job this list does not know about gets
		// swallowed into its predecessor's section, and its own guard step
		// then reads as the predecessor bypassing the format gate.
		{id: "darwin-runtime", next: "organic-runtime-e2e"},
		{id: "organic-runtime-e2e", next: "e2e-tests"},
		{id: "e2e-tests"},
	}
	for _, job := range jobs {
		t.Run(job.id, func(t *testing.T) {
			marker := "  " + job.id + ":\n"
			start := strings.Index(workflow, marker)
			if start < 0 {
				t.Fatalf("missing required job %q", job.id)
			}
			section := workflow[start:]
			if job.next != "" {
				end := strings.Index(section, "\n  "+job.next+":")
				if end < 0 {
					t.Fatalf("missing job boundary %q", job.next)
				}
				section = section[:end]
			}
			needs := "go-format"
			if job.needs != "" {
				needs = job.needs
			}
			required := []string{"    needs: " + needs + "\n", guard}
			if job.ifTokens == nil {
				expectedIf := "    if: always()\n"
				if !strings.Contains(section, expectedIf) {
					expectedIf = "    if: github.event_name == 'workflow_dispatch'\n"
				}
				required = append(required, expectedIf)
			} else {
				// The expression is folded over several lines, so check its
				// tokens against the normalized `if:` block, not one line.
				const ifKey = "\n    if: "
				ifStart := strings.Index(section, ifKey)
				if ifStart < 0 {
					t.Fatalf("%s has no job-level if", job.id)
				}
				ifBlock := section[ifStart+len(ifKey):]
				if end := strings.Index(ifBlock, "\n    runs-on:"); end >= 0 {
					ifBlock = ifBlock[:end]
				}
				ifBlock = strings.Join(strings.Fields(ifBlock), " ")
				for _, token := range job.ifTokens {
					if !strings.Contains(ifBlock, token) {
						t.Fatalf("%s if expression %q missing %q", job.id, ifBlock, token)
					}
				}
			}
			for _, want := range required {
				if !strings.Contains(section, want) {
					t.Fatalf("%s missing fail-closed contract %q", job.id, want)
				}
			}

			expensiveSteps := section[strings.Index(section, guard)+len(guard):]
			if strings.Contains(expensiveSteps, "\n        if:") || strings.Contains(section, "continue-on-error:") {
				t.Fatalf("%s can bypass the failed format guard", job.id)
			}
		})
	}
}

func TestOrganicRuntimeE2EUsesInstalledOpenCodePin(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	install := "npm install --global opencode-ai@" + versions.OpenCode
	if strings.Count(string(data), install) != 1 {
		t.Fatalf("organic runtime E2E must install exact supported OpenCode pin %q once", install)
	}
}

// TestWindowsReleaseBlockerPackagesExist guards the Windows release-blocker step
// of ci.yml against naming a package that no longer exists: a stale entry would
// make `go test` fail on the runner or, worse, be reworded until it passes
// vacuously. Every package of `$packages` must resolve in `go list ./...`.
func TestWindowsReleaseBlockerPackagesExist(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	step := string(workflow)
	start := strings.Index(step, "- name: Run Windows release-blocker tests")
	if start < 0 {
		t.Fatal("ci.yml has no Windows release-blocker step")
	}
	step = step[start:]
	declaration := regexp.MustCompile(`\$packages = @\(([^)]*)\)`).FindStringSubmatch(step)
	if declaration == nil {
		t.Fatal("the Windows release-blocker step declares no $packages list")
	}
	packages := regexp.MustCompile(`'\./([^']+)'`).FindAllStringSubmatch(declaration[1], -1)
	if len(packages) == 0 {
		t.Fatal("the Windows release-blocker step lists no package; the guard would pass vacuously")
	}

	command := exec.Command("go", "list", "./...")
	command.Dir = filepath.Join("..", "..")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list ./...: %v\n%s", err, output)
	}
	existing := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if index := strings.Index(line, "/internal/"); index >= 0 {
			existing["internal/"+line[index+len("/internal/"):]] = true
		}
	}
	for _, match := range packages {
		if !existing[match[1]] {
			t.Errorf("the Windows release-blocker step lists ./%s, which is not a package of this module", match[1])
		}
	}
}

type formatterCandidate struct {
	path    string
	mode    string
	content string
}

type formatterOrderingFixture struct {
	candidate formatterCandidate
	reviewed  formatterCandidate
	receipt   bool
	commits   int
}

func normalizeFixture(candidate *formatterCandidate) {
	candidate.content = strings.ReplaceAll(candidate.content, "x=1", "x = 1")
}

func (fixture *formatterOrderingFixture) startReview() {
	fixture.reviewed = fixture.candidate
	fixture.receipt = true
}

func (fixture *formatterOrderingFixture) runCommitHook(hook func(*formatterCandidate)) bool {
	frozen := fixture.identity(fixture.reviewed)
	hook(&fixture.candidate)
	if fixture.identity(fixture.candidate) != frozen {
		fixture.receipt = false
		return false
	}
	fixture.commits++
	return true
}

func (fixture formatterOrderingFixture) identity(candidate formatterCandidate) string {
	return fmt.Sprintf("%s\x00%s\x00%s", candidate.path, candidate.mode, candidate.content)
}

func TestFormatterOrderingBehaviorContract(t *testing.T) {
	t.Run("formatter mutates before review then converges at commit", func(t *testing.T) {
		fixture := formatterOrderingFixture{candidate: formatterCandidate{path: "main.go", mode: "100644", content: "var x=1\n"}}
		normalizeFixture(&fixture.candidate)
		fixture.startReview()

		if !fixture.runCommitHook(normalizeFixture) {
			t.Fatal("convergent formatter hook rejected frozen reviewed bytes")
		}
		if !fixture.receipt || fixture.commits != 1 || fixture.candidate != fixture.reviewed {
			t.Fatalf("receipt=%t commits=%d candidate=%+v reviewed=%+v", fixture.receipt, fixture.commits, fixture.candidate, fixture.reviewed)
		}
	})

	mutations := map[string]func(*formatterCandidate){
		"byte": func(candidate *formatterCandidate) { candidate.content += "// changed\n" },
		"path": func(candidate *formatterCandidate) { candidate.path = "renamed.go" },
		"mode": func(candidate *formatterCandidate) { candidate.mode = "100755" },
	}
	for name, mutate := range mutations {
		t.Run(name+" mutation after freeze fails closed", func(t *testing.T) {
			fixture := formatterOrderingFixture{candidate: formatterCandidate{path: "main.go", mode: "100644", content: "var x = 1\n"}}
			fixture.startReview()

			if fixture.runCommitHook(mutate) || fixture.receipt || fixture.commits != 0 {
				t.Fatalf("mutation passed: receipt=%t commits=%d", fixture.receipt, fixture.commits)
			}
		})
	}
}

// TestWindowsFullSuiteShardsCoverEveryTestName proves the sharded Windows lane
// still runs every test. Sharding trades one long job for several short ones,
// and its failure mode is silent: a selector edited to stop matching some tests
// skips them while every shard, and the lane, still reports green.
//
// The shards used to be letter ranges tiling A-Z, checked by range algebra.
// That was cheap to verify but impossible to balance: Go test names cluster by
// prefix and so does their cost, so one letter (TestReview*, 423 of the 649
// TestR* tests in internal/cli) can own more work than a whole shard's budget.
// Selectors are therefore arbitrary -run regexes now, and coverage is proved
// against the package's real `go test -list` inventory instead: every test that
// exists must be claimed by exactly one shard. That is strictly stronger than
// the range check -- it catches an uncovered name even when the ranges look
// contiguous -- and it leaves the split free to follow measured cost.
func TestWindowsFullSuiteShardsCoverEveryTestName(t *testing.T) {
	testenv.RequireGoToolchain(t)
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "windows-full-suite.yml"))
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(data), "  windows-full-suite:")
	if start < 0 {
		t.Fatal("missing windows-full-suite job")
	}
	section := string(data)[start:]
	// Stop at the step list: steps carry their own `run:` keys, and a shell
	// command is not a shard selector.
	if end := strings.Index(section, "\n    steps:"); end >= 0 {
		section = section[:end]
	}

	// Selectors are read straight out of the matrix, so this cannot drift from
	// what actually runs the way a copy of the list would.
	type shard struct {
		name     string
		pkg      string
		selector string
		matcher  *regexp.Regexp
	}
	var shards []shard
	var name string
	var pkg string
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(trimmed, "- name: "); ok {
			name = strings.Trim(value, `"`)
		}
		if value, ok := strings.CutPrefix(trimmed, "packages: "); ok {
			pkg = strings.Trim(value, `"`)
		}
		value, ok := strings.CutPrefix(trimmed, "run: ")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		if value == "" {
			// The `rest` shard selects by package, not by name.
			continue
		}
		matcher, err := regexp.Compile(value)
		if err != nil {
			t.Fatalf("shard %q selector %q does not compile as a -run regex: %v", name, value, err)
		}
		shards = append(shards, shard{name: name, pkg: pkg, selector: value, matcher: matcher})
	}
	if len(shards) == 0 {
		t.Fatal("no name-selecting shards found; the guard would pass vacuously")
	}

	byPackage := map[string][]shard{}
	for _, s := range shards {
		if s.pkg == "" {
			t.Fatalf("shard %q selects tests by name but names no package", s.name)
		}
		byPackage[s.pkg] = append(byPackage[s.pkg], s)
	}

	for pkg, packageShards := range byPackage {
		// Ask go test for the current top-level inventory instead of
		// maintaining a second list that would rot.
		command := exec.Command("go", "test", pkg, "-list", "^Test")
		command.Dir = filepath.Join("..", "..")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("list %s tests: %v\n%s", pkg, err, output)
		}
		inventory := 0
		claimed := map[string]int{}
		for _, testName := range strings.Split(string(output), "\n") {
			testName = strings.TrimSpace(testName)
			if !strings.HasPrefix(testName, "Test") {
				continue
			}
			inventory++
			var owners []string
			for _, s := range packageShards {
				// -run splits on "/" for subtests; these selectors address
				// top-level names only, so a plain match is the same rule.
				if s.matcher.MatchString(testName) {
					owners = append(owners, s.name)
				}
			}
			if len(owners) != 1 {
				t.Fatalf("%s test %q is claimed by %d shards (%v), want exactly one", pkg, testName, len(owners), owners)
			}
			claimed[owners[0]]++
		}
		if inventory == 0 {
			t.Fatalf("%s test inventory is empty; coverage guard would pass vacuously", pkg)
		}
		// A shard that matches nothing is the same silent hole seen from the
		// other side, and the lane's own runtime guard throws on it. Catch it
		// here instead of one push later.
		for _, s := range packageShards {
			if claimed[s.name] == 0 {
				t.Fatalf("%s shard %q selector %q matches no test", pkg, s.name, s.selector)
			}
		}
	}
}

func TestArchOpenCodeProvisioningRunsPublishedRepairBeforeVersionCheck(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "e2e", "Dockerfile.arch"))
	if err != nil {
		t.Fatal(err)
	}
	required := "RUN npm install --global opencode-ai@" + versions.OpenCode + " && \\\n" +
		"    (cd \"$(npm root --global)/opencode-ai\" && node postinstall.mjs) && \\\n" +
		"    opencode_version=\"$(opencode --version)\" && \\\n" +
		"    test \"$opencode_version\" = \"" + versions.OpenCode + "\" && \\\n"
	if !strings.Contains(string(data), required) {
		t.Fatal("Arch must run the package-published postinstall repair in its package directory and fail closed on the pinned version before installing other runtimes")
	}
}
