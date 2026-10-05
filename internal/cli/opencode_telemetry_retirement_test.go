package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/components/mutationjournal"
	"github.com/IGutierrezZ/axiom/v3/internal/components/telemetryruntime"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
	"github.com/IGutierrezZ/axiom/v3/internal/planner"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/telemetry"
)

// The plugin is no longer shipped, so the legacy pair is reproduced from the
// vendored copies of what earlier releases installed.
// Resolved at package initialisation: tests that change directory would break
// a relative path.
var legacyTelemetryFixtureDir, _ = filepath.Abs(filepath.Join("..", "components", "telemetryruntime", "testdata"))

var legacyTelemetryPlugins = []string{"telemetry-runtime-axiom-v1.ts", "telemetry-runtime-axiom-v2.ts", "telemetry-runtime-a9cab7dd.ts"}

func writeLegacyTelemetryPair(t *testing.T, configDir, fixture string) []string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(legacyTelemetryFixtureDir, fixture))
	if err != nil {
		t.Fatal(err)
	}
	paths := telemetryruntime.ManagedPaths(configDir)
	if err := os.MkdirAll(filepath.Dir(paths[0]), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[0], content, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.MarshalIndent(map[string]any{
		"schema": "gentle-ai.telemetry-runtime-ownership/v1",
		"file": mutationjournal.OwnedFile{
			After: string(content), AfterHash: fmt.Sprintf("%x", sha256.Sum256(content)), Mode: 0o644,
		},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[1], append(manifest, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return paths
}

func openCodeTestConfig(t *testing.T) (home, config string) {
	t.Helper()
	home = t.TempDir()
	setOpenCodeTestHome(t, home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("DO_NOT_TRACK", "1")
	return home, opencode.NewAdapter().GlobalConfigDir(home)
}

func TestOpenCodeTelemetryRetirementStepRemovesOwnedPlugin(t *testing.T) {
	for _, fixture := range legacyTelemetryPlugins {
		t.Run(fixture, func(t *testing.T) {
			_, config := openCodeTestConfig(t)
			paths := writeLegacyTelemetryPair(t, config, fixture)
			sibling := filepath.Join(filepath.Dir(paths[0]), "community.ts")
			if err := os.WriteFile(sibling, []byte("community"), 0o600); err != nil {
				t.Fatal(err)
			}
			runState := &runtimeState{}
			step := openCodeTelemetryRetirementStep{id: "retire", configDir: config, state: runState}
			if err := step.Run(); err != nil {
				t.Fatalf("retirement must never fail the run: %v", err)
			}
			for _, path := range paths {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("owned legacy file remains: %s: %v", path, err)
				}
				if !slices.Contains(runState.retiredFiles, path) {
					t.Fatalf("removal of %s was not reported", path)
				}
			}
			if len(runState.retirementNotes) != 0 {
				t.Fatalf("unexpected warnings: %v", runState.retirementNotes)
			}
			if data, err := os.ReadFile(sibling); err != nil || string(data) != "community" {
				t.Fatalf("retirement touched an unrelated plugin: %v", err)
			}

			// A second run has nothing left to do.
			again := &runtimeState{}
			if err := (openCodeTelemetryRetirementStep{id: "retire", configDir: config, state: again}).Run(); err != nil {
				t.Fatal(err)
			}
			if len(again.retiredFiles) != 0 || len(again.retirementNotes) != 0 {
				t.Fatalf("second retirement is not a no-op: %+v", again)
			}
		})
	}
}

func TestOpenCodeTelemetryRetirementStepKeepsUnprovenPlugin(t *testing.T) {
	for _, kind := range []string{"edited", "unowned"} {
		t.Run(kind, func(t *testing.T) {
			_, config := openCodeTestConfig(t)
			var paths []string
			if kind == "edited" {
				paths = writeLegacyTelemetryPair(t, config, "telemetry-runtime-axiom-v1.ts")
			} else {
				paths = telemetryruntime.ManagedPaths(config)
				if err := os.MkdirAll(filepath.Dir(paths[0]), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(paths[0], []byte("user plugin"), 0o600); err != nil {
				t.Fatal(err)
			}
			runState := &runtimeState{}
			if err := (openCodeTelemetryRetirementStep{id: "retire", configDir: config, state: runState}).Run(); err != nil {
				t.Fatalf("an unproven plugin must not fail the run: %v", err)
			}
			if data, err := os.ReadFile(paths[0]); err != nil || string(data) != "user plugin" {
				t.Fatalf("unproven plugin was not preserved: %v", err)
			}
			if len(runState.retiredFiles) != 0 {
				t.Fatalf("nothing should be reported removed: %v", runState.retiredFiles)
			}
			if len(runState.retirementNotes) != 1 || !strings.Contains(runState.retirementNotes[0], paths[0]) || !strings.Contains(runState.retirementNotes[0], "kept") {
				t.Fatalf("warning does not name the kept plugin: %v", runState.retirementNotes)
			}
		})
	}
}

func TestOpenCodeTelemetryRetirementStepWithoutPluginDoesNothing(t *testing.T) {
	_, config := openCodeTestConfig(t)
	runState := &runtimeState{}
	if err := (openCodeTelemetryRetirementStep{id: "retire", configDir: config, state: runState}).Run(); err != nil {
		t.Fatal(err)
	}
	if len(runState.retiredFiles) != 0 || len(runState.retirementNotes) != 0 {
		t.Fatalf("empty config produced output: %+v", runState)
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatalf("retirement created the OpenCode config directory: %v", err)
	}
}

func TestOpenCodeInstallRetiresTelemetryPluginWithoutInstallingOne(t *testing.T) {
	for _, selected := range []bool{true, false} {
		t.Run(map[bool]string{true: "selected", false: "absent"}[selected], func(t *testing.T) {
			home, config := openCodeTestConfig(t)
			if err := telemetry.Save(home, telemetry.State{InstallID: "existing", Enabled: false, NoticeShown: true}); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(telemetry.Path(home))
			agents := []model.AgentID{model.AgentClaudeCode}
			if selected {
				agents = []model.AgentID{model.AgentOpenCode}
			}
			selection := model.Selection{Agents: agents}
			resolved := planner.ResolvedPlan{Agents: agents}
			rt := &installRuntime{homeDir: home, workspaceDir: t.TempDir(), scope: ScopeGlobal, selection: selection, resolved: resolved, state: &runtimeState{}}
			plan := rt.stagePlan()

			for _, step := range plan.Prepare {
				if strings.Contains(step.ID(), "telemetry") {
					t.Fatalf("install still plans a telemetry prepare step: %s", step.ID())
				}
			}
			var retire []string
			for i, step := range plan.Apply {
				if strings.Contains(step.ID(), "telemetry") {
					retire = append(retire, step.ID())
					if i != len(plan.Apply)-1 {
						t.Fatalf("retirement step %s is not last", step.ID())
					}
				}
			}
			if selected && !slices.Equal(retire, []string{"opencode:retire-telemetry-plugin"}) || !selected && len(retire) != 0 {
				t.Fatalf("retirement steps = %v, selected = %v", retire, selected)
			}

			paths := writeLegacyTelemetryPair(t, config, "telemetry-runtime-axiom-v2.ts")
			for _, step := range plan.Apply {
				if strings.Contains(step.ID(), "telemetry") {
					if err := step.Run(); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, path := range paths {
				_, err := os.Lstat(path)
				if selected && !os.IsNotExist(err) {
					t.Fatalf("install did not retire %s: %v", path, err)
				}
				if !selected && err != nil {
					t.Fatalf("install touched the plugin without OpenCode selected: %v", err)
				}
			}
			backups, err := backupTargets(home, rt.workspaceDir, ScopeGlobal, selection, resolved)
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range paths {
				if slices.Contains(backups, path) {
					t.Fatalf("install backs up the retired plugin path %s", path)
				}
			}
			if after, _ := os.ReadFile(telemetry.Path(home)); string(before) != string(after) {
				t.Fatal("installation changed telemetry opt-out")
			}
		})
	}
}

func TestOpenCodeInstallWarnsAboutKeptTelemetryPlugin(t *testing.T) {
	home, config := openCodeTestConfig(t)
	paths := writeLegacyTelemetryPair(t, config, "telemetry-runtime-axiom-v1.ts")
	if err := os.WriteFile(paths[0], []byte("user plugin"), 0o600); err != nil {
		t.Fatal(err)
	}
	agents := []model.AgentID{model.AgentOpenCode}
	rt := &installRuntime{homeDir: home, workspaceDir: t.TempDir(), scope: ScopeGlobal, selection: model.Selection{Agents: agents}, resolved: planner.ResolvedPlan{Agents: agents}, state: &runtimeState{}}
	for _, step := range rt.stagePlan().Apply {
		if strings.Contains(step.ID(), "telemetry") {
			if err := step.Run(); err != nil {
				t.Fatal(err)
			}
		}
	}
	rendered := RenderInstallManualActions(InstallResult{Execution: pipeline.ExecutionResult{ManualActions: rt.state.retirementNotes}})
	if !strings.Contains(rendered, "Manual actions required") || !strings.Contains(rendered, paths[0]) {
		t.Fatalf("kept plugin is not surfaced to the user: %q", rendered)
	}
}

func TestSyncRetiresTelemetryPluginAndReportsIt(t *testing.T) {
	workspace, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(workspace)
	home, config := openCodeTestConfig(t)
	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"opencode"}, SelectionConfigured: true, Persona: "neutral",
	}); err != nil {
		t.Fatal(err)
	}
	restoreHome := osUserHomeDir
	osUserHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { osUserHomeDir = restoreHome })
	paths := writeLegacyTelemetryPair(t, config, "telemetry-runtime-axiom-v1.ts")

	first, err := RunSync([]string{"--agents", "opencode"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	for _, path := range paths {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("sync did not retire %s: %v", path, err)
		}
		if !slices.Contains(first.ChangedFiles, path) {
			t.Fatalf("retirement of %s is missing from ChangedFiles %v", path, first.ChangedFiles)
		}
	}
	second, err := RunSync([]string{"--agents", "opencode"})
	if err != nil {
		t.Fatalf("second RunSync() error = %v", err)
	}
	for _, path := range paths {
		if slices.Contains(second.ChangedFiles, path) {
			t.Fatalf("second sync reports %s again", path)
		}
	}
	if len(second.Execution.ManualActions) != 0 {
		t.Fatalf("second sync warns: %v", second.Execution.ManualActions)
	}
}

func TestSyncWarnsAboutKeptTelemetryPlugin(t *testing.T) {
	home, config := openCodeTestConfig(t)
	paths := writeLegacyTelemetryPair(t, config, "telemetry-runtime-axiom-v2.ts")
	if err := os.WriteFile(paths[0], []byte("user plugin"), 0o600); err != nil {
		t.Fatal(err)
	}
	selection := model.Selection{Agents: []model.AgentID{model.AgentOpenCode}}
	rt, err := newSyncRuntime(home, selection)
	if err != nil {
		t.Fatal(err)
	}
	plan := rt.stagePlan()
	last := plan.Apply[len(plan.Apply)-1]
	if last.ID() != "sync:opencode:retire-telemetry-plugin" {
		t.Fatalf("last sync step = %s, want the retirement step", last.ID())
	}
	for _, path := range rt.managedPaths {
		if slices.Contains(paths, path) {
			t.Fatalf("sync backs up the retired plugin path %s", path)
		}
	}
	if err := last.Run(); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(paths[0]); string(data) != "user plugin" {
		t.Fatal("edited plugin was not preserved")
	}
	rendered := RenderSyncReport(SyncResult{Agents: selection.Agents, NoOp: true, Execution: pipeline.ExecutionResult{ManualActions: rt.state.retirementNotes}})
	if !strings.Contains(rendered, "WARNING:") || !strings.Contains(rendered, paths[0]) {
		t.Fatalf("kept plugin is not surfaced in the sync report: %q", rendered)
	}
}

type failAfterOpenCodePluginStep struct{ path string }

func (s failAfterOpenCodePluginStep) ID() string { return "test:fail-after-opencode-plugin" }
func (s failAfterOpenCodePluginStep) Run() error {
	// The failing run has already rewritten the managed plugin when it fails.
	if err := os.WriteFile(s.path, []byte("bytes written by the failing run"), 0o644); err != nil {
		return err
	}
	return fmt.Errorf("forced failure after the OpenCode plugin was rewritten")
}

// XDG_CONFIG_HOME places the OpenCode config outside HOME. The snapshot still
// covers managed files under it, so rollback must be allowed to restore there.
func TestOpenCodeRollbackRestoresConfigOutsideHomeViaXDG(t *testing.T) {
	for _, flow := range []string{"install", "sync"} {
		t.Run(flow, func(t *testing.T) {
			home := t.TempDir()
			setOpenCodeTestHome(t, home)
			xdg := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("DO_NOT_TRACK", "1")
			config := opencode.NewAdapter().GlobalConfigDir(home)
			if !strings.HasPrefix(config, xdg) {
				t.Fatalf("OpenCode config %q is not under XDG_CONFIG_HOME %q", config, xdg)
			}
			plugin := filepath.Join(config, "plugins", "skill-registry.ts")
			if err := os.MkdirAll(filepath.Dir(plugin), 0o755); err != nil {
				t.Fatal(err)
			}
			const original = "pre-existing plugin bytes"
			if err := os.WriteFile(plugin, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}
			restoreBackupHome := backup.UserHomeDirFn
			backup.UserHomeDirFn = func() (string, error) { return home, nil }
			t.Cleanup(func() { backup.UserHomeDirFn = restoreBackupHome })

			agents := []model.AgentID{model.AgentOpenCode}
			selection := model.Selection{Agents: agents, Components: []model.ComponentID{model.ComponentSDD}}
			var plan pipeline.StagePlan
			var rollbackState *runtimeState
			if flow == "install" {
				rt := &installRuntime{homeDir: home, workspaceDir: t.TempDir(), backupRoot: filepath.Join(home, "backups"), scope: ScopeGlobal, selection: selection, resolved: planner.ResolvedPlan{Agents: agents, OrderedComponents: selection.Components}, state: &runtimeState{}}
				plan = rt.stagePlan()
				rollbackState = rt.state
			} else {
				rt, err := newSyncRuntime(home, selection)
				if err != nil {
					t.Fatal(err)
				}
				plan = rt.stagePlan()
				rollbackState = rt.state
			}
			// Only the snapshot and the rollback owner take part: the point is the restore roots.
			var steps []pipeline.Step
			for _, step := range plan.Apply {
				if _, ok := step.(rollbackRestoreStep); ok {
					steps = append(steps, step)
				}
			}
			var prepare []pipeline.Step
			for _, step := range plan.Prepare {
				if _, ok := step.(prepareBackupStep); ok {
					prepare = append(prepare, step)
				}
			}
			steps = append(steps, failAfterOpenCodePluginStep{path: plugin})
			result := pipeline.NewOrchestrator(pipeline.DefaultRollbackPolicy()).Execute(pipeline.StagePlan{Prepare: prepare, Apply: steps})
			if result.Err == nil {
				t.Fatal("forced failure did not fail the pipeline")
			}
			if !result.Rollback.Success {
				t.Fatalf("rollback failed: err=%v steps=%#v", result.Rollback.Err, result.Rollback.Steps)
			}
			if data, err := os.ReadFile(plugin); err != nil || string(data) != original {
				t.Fatalf("plugin outside HOME was not restored: %q, %v", data, err)
			}
			if rollbackState.rollbackSnapshotDir != "" {
				t.Fatalf("rollback left a snapshot at %q", rollbackState.rollbackSnapshotDir)
			}
		})
	}
}
