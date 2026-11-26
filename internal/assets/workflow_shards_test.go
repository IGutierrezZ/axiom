package assets

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// workflowShard is one entry of a `matrix.shard` list. `packages` and
// `exclude` are space-separated `./` package paths; `run` is an optional
// `go test -run` selector.
type workflowShard struct {
	Name     string `yaml:"name"`
	Packages string `yaml:"packages"`
	Run      string `yaml:"run"`
	Exclude  string `yaml:"exclude"`
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
}

type workflowJob struct {
	Name     string `yaml:"name"`
	Needs    any    `yaml:"needs"`
	Strategy struct {
		Matrix struct {
			Shard []workflowShard `yaml:"shard"`
		} `yaml:"matrix"`
	} `yaml:"strategy"`
	Steps []workflowStep `yaml:"steps"`
}

func (job workflowJob) needs() []string {
	switch value := job.Needs.(type) {
	case string:
		return []string{value}
	case []any:
		var names []string
		for _, item := range value {
			if name, ok := item.(string); ok {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

func readWorkflowJobs(t *testing.T, file string) map[string]workflowJob {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", file))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]workflowJob `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return workflow.Jobs
}

// checkShardPartition proves that a sharded job runs every package exactly
// once. Shards that name packages run them; exactly one `rest` shard names
// none and runs `go list ./...` minus its `exclude`. The failure mode being
// guarded is silent: a package left out of `exclude` runs twice, and one
// excluded but run by no other shard never runs while the lane stays green.
// A package may appear in several shards only when every one of them splits
// it by `run` selector; that split is checked against the real test
// inventory by the Windows guard.
func checkShardPartition(t *testing.T, label string, shards []workflowShard) {
	t.Helper()
	if len(shards) == 0 {
		t.Fatalf("%s declares no shards; the guard would pass vacuously", label)
	}
	var rest []workflowShard
	named := map[string][]workflowShard{}
	for _, shard := range shards {
		if strings.TrimSpace(shard.Packages) == "" {
			rest = append(rest, shard)
			continue
		}
		if strings.TrimSpace(shard.Exclude) != "" {
			t.Fatalf("%s shard %q names packages and also excludes %q; only the rest shard excludes", label, shard.Name, shard.Exclude)
		}
		for _, pkg := range strings.Fields(shard.Packages) {
			named[pkg] = append(named[pkg], shard)
		}
	}
	if len(rest) != 1 {
		t.Fatalf("%s has %d shards without packages, want exactly one rest shard", label, len(rest))
	}
	if rest[0].Run != "" {
		t.Fatalf("%s rest shard %q selects by name (%q); it must select by package only", label, rest[0].Name, rest[0].Run)
	}
	for pkg, owners := range named {
		dir := filepath.Join("..", "..", filepath.FromSlash(strings.TrimPrefix(pkg, "./")))
		if goFiles, _ := filepath.Glob(filepath.Join(dir, "*.go")); len(goFiles) == 0 {
			t.Errorf("%s names %s, which is not a package of this module", label, pkg)
		}
		if len(owners) < 2 {
			continue
		}
		for _, owner := range owners {
			if owner.Run == "" {
				t.Errorf("%s runs %s in %d shards, and shard %q runs it whole", label, pkg, len(owners), owner.Name)
			}
		}
	}
	excluded := map[string]bool{}
	for _, pkg := range strings.Fields(rest[0].Exclude) {
		if excluded[pkg] {
			t.Errorf("%s rest shard excludes %s twice", label, pkg)
		}
		excluded[pkg] = true
		if len(named[pkg]) == 0 {
			t.Errorf("%s rest shard excludes %s, which no other shard runs: its tests would never run", label, pkg)
		}
	}
	for pkg := range named {
		if !excluded[pkg] {
			t.Errorf("%s runs %s in its own shard and again in the rest shard; add it to exclude", label, pkg)
		}
	}
}

// TestUnitTestShardsPartitionEveryPackage guards the Linux unit-test matrix of
// ci.yml: the parallel groups together are exactly `go test ./...`.
func TestUnitTestShardsPartitionEveryPackage(t *testing.T) {
	jobs := readWorkflowJobs(t, "ci.yml")
	job, ok := jobs["go-tests"]
	if !ok {
		t.Fatal("ci.yml has no go-tests job")
	}
	for _, shard := range job.Strategy.Matrix.Shard {
		if shard.Run != "" {
			t.Fatalf("go-tests shard %q selects tests by name; the Linux groups split by package only", shard.Name)
		}
	}
	checkShardPartition(t, "ci.yml go-tests", job.Strategy.Matrix.Shard)
}

// TestUnitTestsAggregatorRequiresEveryUnitJob guards the required checks of the
// `main` ruleset. It requires the checks by name ("Go Format" and "Unit
// Tests"), so renaming either job leaves every PR waiting for a check that no
// longer exists. "Unit Tests" runs no test itself: it must depend on every
// parallel unit job and fail unless each one succeeded.
func TestUnitTestsAggregatorRequiresEveryUnitJob(t *testing.T) {
	jobs := readWorkflowJobs(t, "ci.yml")
	for id, name := range map[string]string{"go-format": "Go Format", "unit-tests": "Unit Tests"} {
		if got := jobs[id].Name; got != name {
			t.Errorf("job %s is named %q, want the required check name %q", id, got, name)
		}
	}
	aggregator := jobs["unit-tests"]
	var checked []string
	for _, step := range aggregator.Steps {
		for _, value := range step.Env {
			checked = append(checked, value)
		}
	}
	for _, child := range []string{"go-tests", "binary-smoke", "deadcode-ratchet", "toolchain-meta-tests"} {
		if _, ok := jobs[child]; !ok {
			t.Errorf("ci.yml has no %s job", child)
			continue
		}
		if !slices.Contains(aggregator.needs(), child) {
			t.Errorf("unit-tests does not need %s, so its failure would not fail the required check", child)
		}
		if want := "${{ needs." + child + ".result }}"; !slices.Contains(checked, want) {
			t.Errorf("unit-tests does not check %s", want)
		}
	}
}

// TestToolchainMetaTestsRunInExactlyOneJob guards the split of the toolchain
// meta-tests out of the unit jobs (C-08-k). `Go Tests` skips the workflow's
// TOOLCHAIN_META_TESTS selector and `Toolchain Meta Tests` runs that same
// selector over its META_PACKAGES. Both failure modes are silent: a selected
// name declared outside those packages is skipped by `Go Tests` and run by
// nobody, and a new test in those packages that launches the toolchain lands
// back in every PR without anyone noticing.
func TestToolchainMetaTestsRunInExactlyOneJob(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Env  map[string]string      `yaml:"env"`
		Jobs map[string]workflowJob `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("parse ci.yml: %v", err)
	}
	selector := workflow.Env["TOOLCHAIN_META_TESTS"]
	if selector == "" {
		t.Fatal("ci.yml declares no workflow-level TOOLCHAIN_META_TESTS selector")
	}
	matcher, err := regexp.Compile(selector)
	if err != nil {
		t.Fatalf("TOOLCHAIN_META_TESTS %q does not compile as a -run regex: %v", selector, err)
	}

	stepRunning := func(job, command string) workflowStep {
		t.Helper()
		for _, step := range workflow.Jobs[job].Steps {
			if strings.Contains(step.Run, command) {
				return step
			}
		}
		t.Fatalf("ci.yml job %s has no step running %q", job, command)
		return workflowStep{}
	}
	stepRunning("go-tests", `go test "${packages[@]}" -skip "$TOOLCHAIN_META_TESTS"`)
	metaStep := stepRunning("toolchain-meta-tests", `-run "$TOOLCHAIN_META_TESTS"`)
	metaDirs := map[string]bool{}
	for _, pkg := range strings.Fields(metaStep.Env["META_PACKAGES"]) {
		metaDirs[filepath.FromSlash(strings.TrimPrefix(pkg, "./"))] = true
	}
	if len(metaDirs) == 0 {
		t.Fatal("toolchain-meta-tests names no META_PACKAGES")
	}

	declared := regexp.MustCompile(`(?m)^func (Test\w*)\(`)
	selected := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "testdata", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range declared.FindAllStringSubmatch(string(content), -1) {
			if !matcher.MatchString(match[1]) {
				continue
			}
			selected++
			dir, relErr := filepath.Rel(root, filepath.Dir(path))
			if relErr != nil {
				return relErr
			}
			if !metaDirs[dir] {
				t.Errorf("%s declares %s, selected by TOOLCHAIN_META_TESTS outside META_PACKAGES: Go Tests skips it and Toolchain Meta Tests never runs it", filepath.ToSlash(dir), match[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if selected == 0 {
		t.Fatal("TOOLCHAIN_META_TESTS selects no declared test; the split would pass vacuously")
	}

	fileSet := token.NewFileSet()
	for dir := range metaDirs {
		files, err := filepath.Glob(filepath.Join(root, dir, "*_test.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			parsed, err := parser.ParseFile(fileSet, file, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse %s: %v", file, err)
			}
			for _, decl := range parsed.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
					continue
				}
				if launchesGoToolchain(fn.Body) && !matcher.MatchString(fn.Name.Name) {
					t.Errorf("%s launches the Go toolchain but TOOLCHAIN_META_TESTS does not select it, so every PR pays for it", fn.Name.Name)
				}
			}
		}
	}
}

// launchesGoToolchain reports whether a test body calls the toolchain gate,
// the release-policy fixture (`go run`), or `go build|run|test` directly.
func launchesGoToolchain(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			found = fun.Name == "newReleasePolicyFixture"
		case *ast.SelectorExpr:
			switch fun.Sel.Name {
			case "RequireGoToolchain":
				found = true
			case "Command", "CommandContext":
				args := call.Args
				if fun.Sel.Name == "CommandContext" && len(args) > 0 {
					args = args[1:]
				}
				if len(args) >= 2 && goStringLiteral(args[0]) == "go" {
					switch goStringLiteral(args[1]) {
					case "build", "run", "test":
						found = true
					}
				}
			}
		}
		return !found
	})
	return found
}

func goStringLiteral(expr ast.Expr) string {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return value
}
