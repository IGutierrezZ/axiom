package assets

import (
	"os"
	"path/filepath"
	"slices"
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
	for _, child := range []string{"go-tests", "binary-smoke", "deadcode-ratchet"} {
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
