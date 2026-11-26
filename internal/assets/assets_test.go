package assets

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSDDOrchestratorAssetsCarryTheKickoffAndBlockGatesMarker is task 23.1:
// each of the every SDD-orchestrator assets carries the shared-section
// placeholder that resolves to INC-21's per-change kickoff and block-gates
// doctrine (RF-PRE-01, RF-PRE-02, RF-GAT-04). internal/components/sdd owns
// substitution and cannot be imported here without a cycle; its own
// TestEveryRuntimeRendersTheSharedSections confirms the resolved body
// actually reaches the rendered prompt once this marker exists.
func TestSDDOrchestratorAssetsCarryTheKickoffAndBlockGatesMarker(t *testing.T) {
	const marker = "{{AXIOM_SDD_SECTION:SDD Change Kickoff and Block Gates}}"
	paths := allSDDOrchestratorAssetPaths(t)
	if len(paths) != orchestratorTemplateCount {
		t.Fatalf("orchestrator coverage sees %d assets, want %d", len(paths), orchestratorTemplateCount)
	}
	for _, path := range paths {
		if content := MustRead(path); !strings.Contains(content, marker) {
			t.Fatalf("%s is missing the %q marker", path, marker)
		}
	}
}

// TestAllEmbeddedAssetsAreReadable verifies that every expected embedded file
// can be loaded via Read() without error. This catches missing/misnamed files
// at test time rather than at runtime.
func TestAllEmbeddedAssetsAreReadable(t *testing.T) {
	expectedFiles := []string{
		// Canonical Engram protocol asset (full/slim/passive-capture/compact
		// marker sections — see design.md Decision 3).
		"engram/protocol.md",

		// Claude agent files
		"claude/output-style-axiom.md",
		"claude/persona-axiom.md",
		"claude/sdd-orchestrator.md",
		"claude/commands/sdd-apply.md",
		"claude/commands/sdd-archive.md",
		"claude/commands/sdd-continue.md",
		"claude/commands/sdd-explore.md",
		"claude/commands/sdd-ff.md",
		"claude/commands/sdd-init.md",
		"claude/commands/sdd-new.md",
		"claude/commands/sdd-onboard.md",
		"claude/commands/sdd-research.md",
		"claude/commands/sdd-status.md",
		"claude/commands/sdd-verify.md",
		"claude/agents/sdd-init.md",
		"claude/agents/sdd-onboard.md",
		"claude/agents/sdd-research.md",

		// OpenCode agent files
		"opencode/persona-axiom.md",
		"opencode/background-subagents.md",
		"opencode/sdd-orchestrator.md",
		"opencode/sdd-overlay-single.json",
		"opencode/sdd-overlay-multi.json",
		"opencode/commands/sdd-apply.md",
		"opencode/commands/sdd-archive.md",
		"opencode/commands/sdd-continue.md",
		"opencode/commands/sdd-explore.md",
		"opencode/commands/sdd-ff.md",
		"opencode/commands/sdd-init.md",
		"opencode/commands/sdd-new.md",
		"opencode/commands/sdd-onboard.md",
		"opencode/commands/sdd-research.md",
		"opencode/commands/sdd-status.md",
		"opencode/commands/sdd-verify.md",

		// Gemini agent files
		"gemini/sdd-orchestrator.md",

		// Antigravity agent files
		"antigravity/sdd-orchestrator.md",

		// Codex agent files
		"codex/sdd-orchestrator.md",

		// Cursor agent files
		"cursor/sdd-orchestrator.md",
		"cursor/agents/sdd-init.md",
		"cursor/agents/sdd-explore.md",
		"cursor/agents/sdd-propose.md",
		"cursor/agents/sdd-research.md",
		"cursor/agents/sdd-spec.md",
		"cursor/agents/sdd-design.md",
		"cursor/agents/sdd-tasks.md",
		"cursor/agents/sdd-apply.md",
		"cursor/agents/sdd-verify.md",
		"cursor/agents/sdd-archive.md",

		// SDD skills
		"skills/sdd-init/SKILL.md",
		"skills/sdd-init/references/init-details.md",
		"skills/sdd-apply/SKILL.md",
		"skills/sdd-archive/SKILL.md",
		"skills/sdd-archive/references/technical-context-integration.md",
		"skills/sdd-design/SKILL.md",
		"skills/sdd-explore/SKILL.md",
		"skills/sdd-propose/SKILL.md",
		"skills/sdd-research/SKILL.md",
		"skills/sdd-spec/SKILL.md",
		"skills/sdd-spec/references/openspec-adoption.md",
		"skills/sdd-tasks/SKILL.md",
		"skills/sdd-verify/SKILL.md",
		"skills/sdd-verify/references/report-format.md",
		"skills/sdd-verify/references/joint-verification.md",
		"skills/skill-registry/SKILL.md",
		"skills/judgment-day/references/prompts-and-formats.md",
		"skills/_shared/README.md",
		"skills/_shared/persistence-contract.md",
		"skills/_shared/engram-convention.md",
		"skills/_shared/openspec-convention.md",
		"skills/_shared/sdd-phase-common.md",
		"skills/_shared/sdd-status-contract.md",
		"skills/_shared/research-lifecycle.md",
		"skills/_shared/project-baseline-convention.md",

		// Foundation skills
		"skills/go-testing/SKILL.md",
		"skills/go-testing/references/examples.md",
		"skills/skill-creator/SKILL.md",
		"skills/skill-creator/references/skill-style-guide.md",
		"skills/skill-improver/SKILL.md",
		"skills/skill-improver/references/skill-style-guide.md",
		"skills/chained-pr/references/chaining-details.md",
		"skills/systemic-issue-triage/SKILL.md",
	}

	for _, path := range expectedFiles {
		t.Run(path, func(t *testing.T) {
			content, err := Read(path)
			if err != nil {
				t.Fatalf("Read(%q) error = %v", path, err)
			}

			if len(strings.TrimSpace(content)) == 0 {
				t.Fatalf("Read(%q) returned empty content", path)
			}

			// Real content should be substantial, not a one-line stub.
			if len(content) < 50 {
				t.Fatalf("Read(%q) content is suspiciously short (%d bytes) — possible stub", path, len(content))
			}
		})
	}
}

func TestOpenCodeEmbeddedAssetLayout(t *testing.T) {
	entries, err := FS.ReadDir("opencode")
	if err != nil {
		t.Fatalf("ReadDir(opencode) error = %v", err)
	}

	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Name()] = true
	}

	for _, name := range []string{"commands", "plugins", "persona-axiom.md", "background-subagents.md", "sdd-orchestrator.md", "sdd-overlay-single.json", "sdd-overlay-multi.json"} {
		if !seen[name] {
			t.Fatalf("opencode embedded assets missing %q", name)
		}
	}

	commandEntries, err := FS.ReadDir("opencode/commands")
	if err != nil {
		t.Fatalf("ReadDir(opencode/commands) error = %v", err)
	}
	if len(commandEntries) != 13 {
		t.Fatalf("opencode commands count = %d, want 13", len(commandEntries))
	}
	wantCommands := map[string]bool{"skill-creator.md": true, "skill-registry.md": true}
	for _, entry := range commandEntries {
		delete(wantCommands, entry.Name())
	}
	for name := range wantCommands {
		t.Fatalf("opencode embedded commands missing %q", name)
	}

	pluginEntries, err := FS.ReadDir("opencode/plugins")
	if err != nil {
		t.Fatalf("ReadDir(opencode/plugins) error = %v", err)
	}
	if len(pluginEntries) != 4 {
		t.Fatalf("opencode plugins count = %d, want 4", len(pluginEntries))
	}
	wantPlugins := map[string]bool{"axiom-branch-guard.ts": true, "model-variants.ts": true, "sdd-task-result-artifacts.ts": true, "skill-registry.ts": true}
	for _, entry := range pluginEntries {
		if !wantPlugins[entry.Name()] {
			t.Fatalf("unexpected plugin entry = %q", entry.Name())
		}
	}
}

func TestOpenCodeBackgroundPolicyMarkersAreBalanced(t *testing.T) {
	content := mustReadResolved("opencode/background-subagents.md")
	const (
		start = "<!-- axiom:opencode-background-subagents -->"
		end   = "<!-- /axiom:opencode-background-subagents -->"
	)
	trimmed := strings.TrimSpace(content)
	if strings.Count(trimmed, start) != 1 || strings.Count(trimmed, end) != 1 {
		t.Fatalf("background policy marker cardinality = (%d, %d), want (1, 1)", strings.Count(trimmed, start), strings.Count(trimmed, end))
	}
	if !strings.HasPrefix(trimmed, start+"\n") || !strings.HasSuffix(trimmed, "\n"+end) {
		t.Fatalf("background policy markers are not balanced around the complete asset")
	}
}

func TestSDDTaskResultArtifactsPluginContract(t *testing.T) {
	source, err := Read("opencode/plugins/sdd-task-result-artifacts.ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`const SDD_PHASES`, `"sdd-research"`, `const SDD_TASK_FAILURE_PREFIX`, `failedSDDSessions`, `confirmedPreflights`, `client.session.get({ path: { id: sessionID } })`, `only the interactive root session may carry parent-confirmed`, `Axiom SDD preflight `, `SDD_PREFLIGHT_ACCEPTED_PREFIXES`, `model-authored preflight text cannot create parent-confirmed authority`, "output.args.prompt = `${preflight}\\n\\n${output.args.prompt}`", `export default SDDTaskResultArtifactsPlugin`,
		`canonicalizeSDDPreflightQuestions`, `input.tool === "question"`,
		`{ label: "Interactive", description:`, `{ label: "Automatic", description:`,
		`{ label: "OpenSpec", description:`, `{ label: "Engram", description:`, `{ label: "Both", description:`,
		`{ label: "Ask me", description:`, `{ label: "Single PR", description:`, `{ label: "Auto", description:`,
		`.normalize("NFD")`, `\u0300-\u036f`,
		`typed chat answers cannot create preflight authority`} {
		if !strings.Contains(source, want) {
			t.Fatalf("SDD task plugin missing %q", want)
		}
	}
}

// TestOpenCodeBranchGuardPluginContract fija el contenido de axiom-branch-guard.ts en las dos
// generaciones de plugins de OpenCode (RF-BRA-23; design D-18; R10): el plugin solo transporta la
// carga a `axiom branch guard --agent opencode` y aplica la decisión que devuelve Go.
func TestOpenCodeBranchGuardPluginContract(t *testing.T) {
	for _, path := range []string{"opencode/plugins/axiom-branch-guard.ts", "opencode/plugins-v2/axiom-branch-guard.ts"} {
		t.Run(path, func(t *testing.T) {
			source, err := Read(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				// Proceso: array de argumentos fijo, sin shell, carga por stdin y tiempo límite.
				`import { spawn } from "node:child_process"`,
				`const GUARD_COMMAND = "axiom"`,
				`const GUARD_ARGS = ["branch", "guard", "--agent", "opencode"]`,
				`spawn(GUARD_COMMAND, GUARD_ARGS,`,
				`.end(JSON.stringify({ event, tool, filePath, cwd }))`,
				`GUARD_TIMEOUT_MS`, `GUARD_MAX_OUTPUT`, `code !== 0`,
				// Herramientas de escritura y eventos de la carga.
				`new Set(["write", "edit", "patch"])`,
				`"pre"`, `"post"`,
				// Decisión de Go: bloqueo con Error y aviso en la salida de la herramienta.
				`throw new Error(`,
				`"block"`, `"warn"`,
				// Fallo en abierto: nada de lo que falle se convierte en decisión.
				`catch {`, `finish()`,
			} {
				if !strings.Contains(source, want) {
					t.Errorf("%s no contiene %q", path, want)
				}
			}
			for _, forbidden := range []string{
				// La orden nunca pasa por un shell ni interpola datos de la carga.
				"shell: true", "exec(", "execSync", "execFile", "spawnSync",
				// El plugin no clasifica ni decide nada: eso es de Go.
				"openspec/changes", "odd/tasks", "writeFile", "readFile", "gentle-ai\"",
			} {
				if strings.Contains(source, forbidden) {
					t.Errorf("%s conserva %q", path, forbidden)
				}
			}
		})
	}
	v1, _ := Read("opencode/plugins/axiom-branch-guard.ts")
	for _, want := range []string{`"tool.execute.before"`, `"tool.execute.after"`, `output.output =`} {
		if !strings.Contains(v1, want) {
			t.Errorf("el plugin de OpenCode v1 no contiene %q", want)
		}
	}
	v2, _ := Read("opencode/plugins-v2/axiom-branch-guard.ts")
	for _, want := range []string{`ctx.tool.hook("execute.before"`, `ctx.tool.hook("execute.after"`} {
		if !strings.Contains(v2, want) {
			t.Errorf("el plugin de OpenCode v2 no contiene %q", want)
		}
	}
}

// TestModelVariantsPluginContract verifies the embedded model-variants.ts
// plugin keeps the contract enforced by PR #440 review: atomic write via
// tmp+rename, always-write semantics (no early return on empty variants),
// and visible error logging instead of silent failure.
func TestModelVariantsPluginContract(t *testing.T) {
	source, err := Read("opencode/plugins/model-variants.ts")
	if err != nil {
		t.Fatalf("Read(model-variants.ts) error = %v", err)
	}
	src := string(source)

	// Atomic write: must import rename and write to a .tmp file before renaming.
	if !strings.Contains(src, "rename") {
		t.Errorf("model-variants.ts must use rename() for atomic write")
	}
	if !strings.Contains(src, ".tmp") {
		t.Errorf("model-variants.ts must write to a .tmp file before rename()")
	}

	// Always-write semantics: the cache must be written unconditionally so an
	// empty variants object overwrites a stale cache from a previous run.
	// Reject any guard on `Object.keys(variants).length` that could short-circuit
	// the write path.
	if strings.Contains(src, "Object.keys(variants).length") {
		t.Errorf("model-variants.ts must not gate the write on variants length (allows stale cache to survive)")
	}
	if !strings.Contains(src, "JSON.stringify(variants") {
		t.Errorf("model-variants.ts must serialize the variants object — even when empty — to overwrite stale cache")
	}

	// Errors must be logged, not swallowed silently.
	if strings.Contains(src, "} catch {") {
		t.Errorf("model-variants.ts must not have a parameterless `catch {}` block (silences ENOSPC/EACCES)")
	}
	if !strings.Contains(src, "console.error") {
		t.Errorf("model-variants.ts must log errors via console.error so users see failures")
	}

	// Per-invocation tmp path: OpenCode loads the plugin twice within the
	// same process when started with `--port`. Both loads share the same
	// PID, so a fixed `.tmp` name races with itself and the second rename()
	// fails with ENOENT. The tmp name must include a per-invocation random
	// suffix (randomBytes) to be unique across both loads, and it must be
	// constructed from cacheDir plus the cache basename so this invocation can
	// track and clean only its own temp file if the write path fails.
	for _, want := range []string{
		`const MODEL_VARIANTS_CACHE_FILE = "model-variants.json"`,
		"const finalPath = path.join(cacheDir, MODEL_VARIANTS_CACHE_FILE)",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("model-variants.ts missing constant-based cache path contract %q", want)
		}
	}
	tmpPathPattern := regexp.MustCompile("tmpPath\\s*=\\s*path\\.join\\(\\s*cacheDir\\s*,\\s*`\\$\\{\\s*MODEL_VARIANTS_CACHE_FILE\\s*\\}\\.\\$\\{\\s*randomBytes\\([^)]*\\)\\s*\\.\\s*toString\\(\\s*[\"']hex[\"']\\s*\\)\\s*\\}\\.tmp`\\s*\\)")
	if !tmpPathPattern.MatchString(src) {
		t.Errorf("model-variants.ts tmp path must use path.join(cacheDir, randomized basename) to be unique across plugin double-loads within the same process")
	}

	// Own-temp cleanup: this randomized temp path has not shipped yet, so there
	// are no previous randomized orphan files to scan at startup. The plugin
	// should only best-effort remove the temp file created by this invocation
	// when it still exists after failure; after rename, the temp file is consumed.
	for _, want := range []string{
		"finally",
		"removeOwnTempFile(tmpPath)",
		"await rm(tmpPath, { force: true })",
		"tmpPath = undefined",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("model-variants.ts missing own-temp cleanup contract %q", want)
		}
	}
	for _, forbidden := range []string{
		"removeStaleModelVariantsTempFiles",
		"STALE_TEMP_FILE_AGE_MS",
		"mtimeMs",
		"Date.now()",
	} {
		if strings.Contains(src, forbidden) {
			t.Errorf("model-variants.ts must not use stale temp cleanup by age; found %q", forbidden)
		}
	}
	if strings.Contains(src, "setTimeout") {
		t.Errorf("model-variants.ts must not use setTimeout for temp cleanup")
	}
}

func TestSkillRegistryPluginContract(t *testing.T) {
	source, err := Read("opencode/plugins/skill-registry.ts")
	if err != nil {
		t.Fatalf("Read(skill-registry.ts) error = %v", err)
	}
	src := string(source)

	for _, want := range []string{
		"execFile",
		"skill-registry",
		"refresh",
		"--quiet",
		"--no-gitignore",
		"--cwd",
		"input.directory",
		"input.worktree",
		"timeout: 30_000",
		"console.error",
		// Non-project guard: a fresh OpenCode directory can resolve to "/" or
		// another non-project location; the plugin must skip silently instead
		// of spawning a refresh that pollutes or fails at startup (#skill-registry-root-guard).
		"isProjectRoot",
		"homedir()",
		".git",
		".atl",
		"console.error",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("skill-registry.ts missing %q", want)
		}
	}
	// stdout belongs to OpenCode commands whose output axiom parses
	// (`opencode models --verbose`); plugin logging must stay on stderr.
	for _, forbidden := range []string{"console.info", "console.log"} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("skill-registry.ts must not log to stdout via %q", forbidden)
		}
	}
	if strings.Contains(src, "exec(") {
		t.Fatal("skill-registry.ts must use execFile, not shell exec")
	}
	if guardIdx, spawnIdx := strings.Index(src, "isProjectRoot"), strings.Index(src, "execFileAsync("); guardIdx == -1 || spawnIdx == -1 || guardIdx >= spawnIdx {
		t.Fatalf("skill-registry.ts must guard before spawning; isProjectRoot@%d execFileAsync(@%d", guardIdx, spawnIdx)
	}
	worktreeIdx := strings.Index(src, "input.worktree")
	directoryIdx := strings.Index(src, "input.directory")
	if worktreeIdx == -1 || directoryIdx == -1 {
		t.Fatal("skill-registry.ts must contain both input.worktree and input.directory")
	}
	if worktreeIdx >= directoryIdx {
		t.Errorf("skill-registry.ts must use input.worktree before input.directory; got worktree@%d >= directory@%d", worktreeIdx, directoryIdx)
	}
}

func TestClaudeEmbeddedAssetLayout(t *testing.T) {
	entries, err := FS.ReadDir("claude")
	if err != nil {
		t.Fatalf("ReadDir(claude) error = %v", err)
	}

	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Name()] = true
	}

	for _, name := range []string{"agents", "commands", "persona-axiom.md", "sdd-orchestrator.md"} {
		if !seen[name] {
			t.Fatalf("claude embedded assets missing %q", name)
		}
	}

	commandEntries, err := FS.ReadDir("claude/commands")
	if err != nil {
		t.Fatalf("ReadDir(claude/commands) error = %v", err)
	}
	if len(commandEntries) != 11 {
		t.Fatalf("claude commands count = %d, want 11", len(commandEntries))
	}

	agentEntries, err := FS.ReadDir("claude/agents")
	if err != nil {
		t.Fatalf("ReadDir(claude/agents) error = %v", err)
	}
	if len(agentEntries) != 14 {
		t.Fatalf("claude agents count = %d, want 14", len(agentEntries))
	}
}

// TestEngramEmbeddedAssetLayout verifies the canonical protocol asset
// directory introduced by the consolidation (design.md Decision 3).
func TestEngramEmbeddedAssetLayout(t *testing.T) {
	entries, err := FS.ReadDir("engram")
	if err != nil {
		t.Fatalf("ReadDir(engram) error = %v", err)
	}

	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Name()] = true
	}

	if !seen["protocol.md"] {
		t.Fatal("engram embedded assets missing \"protocol.md\"")
	}
}

// TestMustReadPanicsOnMissingFile verifies that MustRead panics for a
// nonexistent file, confirming the safety mechanism works.
func TestMustReadPanicsOnMissingFile(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustRead() did not panic for missing file")
		}
	}()

	MustRead("nonexistent/file.md")
}

// TestEmbeddedAssetCount verifies we have the expected number of embedded files.
// This catches accidental deletions of asset files.
func TestEmbeddedAssetCount(t *testing.T) {
	// Count skill files.
	entries, err := FS.ReadDir("skills")
	if err != nil {
		t.Fatalf("ReadDir(skills) error = %v", err)
	}

	skillDirs := 0
	for _, entry := range entries {
		if entry.IsDir() {
			skillDirs++
		}
	}

	// We expect 23 skill directories (11 SDD + judgment-day + 10 foundation + _shared).
	if skillDirs != 23 {
		t.Fatalf("expected 23 skill directories, got %d", skillDirs)
	}

	// Verify each skill directory has a SKILL.md.
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "_shared" {
			for _, sharedFile := range []string{"README.md", "persistence-contract.md", "engram-convention.md", "openspec-convention.md", "sdd-phase-common.md", "sdd-status-contract.md", "research-lifecycle.md", "skill-resolver.md", "project-baseline-convention.md"} {
				sharedPath := "skills/_shared/" + sharedFile
				if _, err := Read(sharedPath); err != nil {
					t.Fatalf("shared directory missing %q: %v", sharedFile, err)
				}
			}
			continue
		}
		skillPath := "skills/" + entry.Name() + "/SKILL.md"
		if _, err := Read(skillPath); err != nil {
			t.Fatalf("skill directory %q missing SKILL.md: %v", entry.Name(), err)
		}
	}
}

func TestOpenCodeSDDOverlaySubagentsAreExplicitExecutors(t *testing.T) {
	for _, assetPath := range []string{"opencode/sdd-overlay-single.json", "opencode/sdd-overlay-multi.json"} {
		t.Run(assetPath, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal([]byte(mustReadResolved(assetPath)), &root); err != nil {
				t.Fatalf("Unmarshal(%q) error = %v", assetPath, err)
			}

			agents, ok := root["agent"].(map[string]any)
			if !ok {
				t.Fatalf("%q missing agent map", assetPath)
			}

			// multi overlay uses __PROMPT_FILE_{phase}__ placeholders that are
			// replaced at runtime with absolute {file:...} references by
			// inlineOpenCodeSDDPrompts. Verify the placeholder format.
			// single overlay still uses inline prompt strings.
			isMulti := assetPath == "opencode/sdd-overlay-multi.json"

			orchestrator, ok := agents["axiom-orchestrator"].(map[string]any)
			if !ok {
				t.Fatalf("%q missing axiom-orchestrator agent", assetPath)
			}
			permissions, ok := orchestrator["permission"].(map[string]any)
			if !ok || permissions["question"] != "allow" {
				t.Fatalf("%q axiom-orchestrator must allow question permission", assetPath)
			}
			for name, raw := range agents {
				if _, exists := raw.(map[string]any)["tools"]; exists {
					t.Fatalf("%q agent %q emits deprecated tools", assetPath, name)
				}
			}

			for _, phase := range []string{"sdd-init", "sdd-explore", "sdd-propose", "sdd-spec", "sdd-design", "sdd-tasks", "sdd-apply", "sdd-verify", "sdd-archive"} {
				agentDef, ok := agents[phase].(map[string]any)
				if !ok {
					t.Fatalf("%q missing %s agent", assetPath, phase)
				}
				prompt, _ := agentDef["prompt"].(string)
				if isMulti {
					// Multi overlay uses placeholders — verify the placeholder exists.
					expectedPlaceholder := "__PROMPT_FILE_" + phase + "__"
					if prompt != expectedPlaceholder {
						t.Fatalf("%q phase %s prompt = %q, want placeholder %q", assetPath, phase, prompt, expectedPlaceholder)
					}
				} else if strings.TrimSpace(prompt) == "" {
					t.Fatalf("%q phase %s has an empty inline prompt", assetPath, phase)
				}
			}
		})
	}
}

// TestCommandsDoNotUseEchoNPwd guards against the nested-subshell pattern
// `echo -n "$(pwd)"` (and the basename variant) that causes Claude Code v2.1.113+
// to reject slash commands with "Unhandled node type: string". Use the plain pwd
// or basename command forms instead — both are accepted by old and new parsers.
func TestCommandsDoNotUseEchoNPwd(t *testing.T) {
	forbidden := `echo -n "$(pwd)"`

	for _, dir := range []string{"claude/commands", "opencode/commands"} {
		entries, err := FS.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir(%s) error = %v", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := dir + "/" + entry.Name()
			content := mustReadResolved(path)
			if strings.Contains(content, forbidden) {
				t.Errorf("%s contains banned pattern %q — use a safer detection mechanism instead", path, forbidden)
			}
		}
	}
}

// TestOpenCodeCommandsDetectWorkspaceAgentSide guards against parse-time shell
// interpolation for the working directory in OpenCode command files. In
// OpenCode Desktop (Electron), patterns like !pwd and !basename $(pwd) evaluate
// against the Electron app data directory rather than the project workspace
// (issue #74). Command files must instruct the agent to detect the workspace
// via its bash tool (e.g. git rev-parse --show-toplevel) and treat that
// returned path as authoritative.
func TestOpenCodeCommandsDetectWorkspaceAgentSide(t *testing.T) {
	forbiddenPatterns := []string{
		"!`pwd`",
		"!`basename \"$(pwd)\"`",
	}
	const requiredHint = "git rev-parse --show-toplevel"

	entries, err := FS.ReadDir("opencode/commands")
	if err != nil {
		t.Fatalf("ReadDir(opencode/commands) error = %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := "opencode/commands/" + entry.Name()
		content := mustReadResolved(path)
		for _, pat := range forbiddenPatterns {
			if strings.Contains(content, pat) {
				t.Errorf("%s contains banned shell interpolation %q — detect the workspace via the agent's bash tool instead (see #74)", path, pat)
			}
		}
		if strings.Contains(content, "Working directory:") && !strings.Contains(content, requiredHint) {
			t.Errorf("%s mentions \"Working directory:\" without the agent-side detection hint %q (see #74)", path, requiredHint)
		}
	}
}

// TestClaudeCommandsDetectWorkspaceAgentSide guards against parse-time shell
// interpolation for workspace/project context in Claude slash commands. Claude
// Code performs static permission validation before running commands, so forms
// like !`basename "$(pwd)"` can be rejected before the agent starts. Command
// files must instruct the agent to detect the workspace from inside the session.
func TestClaudeCommandsDetectWorkspaceAgentSide(t *testing.T) {
	forbiddenPatterns := []string{
		"!pwd",
		"!`pwd`",
		"!basename $(pwd)",
		"!basename \"$(pwd)\"",
		"!basename '$(pwd)'",
		"!`basename $(pwd)`",
		"!`basename \"$(pwd)\"`",
		"!`basename '$(pwd)'`",
		"!git rev-parse --show-toplevel",
		"!`git rev-parse --show-toplevel`",
	}
	const requiredHint = "git rev-parse --show-toplevel"

	entries, err := FS.ReadDir("claude/commands")
	if err != nil {
		t.Fatalf("ReadDir(claude/commands) error = %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := "claude/commands/" + entry.Name()
		content := mustReadResolved(path)
		for _, pat := range forbiddenPatterns {
			if strings.Contains(content, pat) {
				t.Errorf("%s contains banned Claude parse-time shell interpolation %q — detect workspace/project context agent-side instead (see #837)", path, pat)
			}
		}
		for _, line := range strings.Split(content, "\n") {
			if (strings.Contains(line, "Working directory:") || strings.Contains(line, "Current project:")) && strings.Contains(line, "!") {
				t.Errorf("%s contains parse-time shell interpolation in workspace/project context line %q — detect it agent-side instead (see #837)", path, line)
			}
		}
		if strings.Contains(content, "Working directory:") && !strings.Contains(content, requiredHint) {
			t.Errorf("%s mentions \"Working directory:\" without the agent-side detection hint %q (see #837)", path, requiredHint)
		}
	}
}

func TestSDDArchiveMoveTransactionPreservesFilesystemOnCollisions(t *testing.T) {
	shell := requireArchiveShell(t)
	t.Setenv("BASH_ENV", "repository-sentinel.txt")
	for _, tracked := range []bool{true, false} {
		sourceMode := "untracked"
		if tracked {
			sourceMode = "tracked"
		}
		t.Run(sourceMode+" source moves to absent destination", func(t *testing.T) {
			root, source, destination, sentinel := setupArchiveFixture(t, tracked)
			output, err := runArchiveMoveTransaction(shell, root)
			if err != nil {
				t.Fatalf("archive transaction failed: %v\n%s", err, output)
			}
			if _, err := os.Lstat(source); !os.IsNotExist(err) {
				t.Fatalf("%s remains after archive move: %v", source, err)
			}
			assertFileContents(t, filepath.Join(destination, "tasks.md"), "archive task bytes\n")
			assertFileContents(t, sentinel, "exit 99\nrepository sentinel\n")
			if tracked {
				assertGitCommandFails(t, root, "ls-files", "--error-unmatch", "openspec/changes/change/tasks.md")
				runGit(t, root, "ls-files", "--error-unmatch", "openspec/changes/archive/2030-01-02-change/tasks.md")
				staged := runGit(t, root, "diff", "--cached", "--name-status")
				if !strings.Contains(staged, "R100") {
					t.Fatalf("tracked archive move did not stage a rename:\n%s", staged)
				}
			} else {
				status := runGit(t, root, "status", "--porcelain", "--untracked-files=all")
				if strings.Contains(status, "openspec/changes/change/") || !strings.Contains(status, "openspec/changes/archive/") {
					t.Fatalf("untracked archive move has unexpected Git state:\n%s", status)
				}
			}
		})
		for _, collision := range []string{"directory", "regular file", "live symlink", "dangling symlink"} {
			t.Run(sourceMode+" source preserves "+collision+" collision", func(t *testing.T) {
				root, source, destination, sentinel := setupArchiveFixture(t, tracked)
				createArchiveCollision(t, root, destination, collision)
				beforeStatus := runGit(t, root, "status", "--porcelain")
				output, err := runArchiveMoveTransaction(shell, root)
				if err == nil {
					t.Fatalf("archive transaction unexpectedly succeeded for %s collision:\n%s", collision, output)
				}
				for _, required := range []string{
					"source openspec/changes/change and destination openspec/changes/archive/2030-01-02-change remain unchanged",
					"Resolve the destination collision, then rerun this archive step.",
				} {
					if !strings.Contains(output, required) {
						t.Fatalf("collision failure missing %q:\n%s", required, output)
					}
				}
				assertFileContents(t, filepath.Join(source, "tasks.md"), "archive task bytes\n")
				assertArchiveCollision(t, root, destination, collision)
				assertFileContents(t, sentinel, "exit 99\nrepository sentinel\n")
				if afterStatus := runGit(t, root, "status", "--porcelain"); afterStatus != beforeStatus {
					t.Fatalf("collision changed Git state:\nbefore:\n%safter:\n%s", beforeStatus, afterStatus)
				}
			})
		}
	}
}
func TestSDDArchiveHistoricalRecoveryRefusesDanglingActiveSourceSymlink(t *testing.T) {
	shell := requireArchiveShell(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "repository-sentinel.txt"), []byte("exit 99\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BASH_ENV", "repository-sentinel.txt")
	activeSource := filepath.Join(root, "openspec", "changes", "change")
	nestedSource := filepath.Join(root, "openspec", "changes", "archive", "2030-01-02-change", "change")
	if err := os.MkdirAll(nestedSource, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedSource, "tasks.md"), []byte("historical task bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing-active-source"), activeSource); err != nil {
		t.Skipf("dangling symlink fixture is unavailable: %v", err)
	}
	recovery := strings.ReplaceAll(archiveFencedShellBlock("### Historical Malformed Nesting Recovery (Manual Only)"), "{change-name}", "change")
	recovery = strings.ReplaceAll(recovery, "YYYY-MM-DD-change", "2030-01-02-change")
	command := exec.Command(shell, "-c", recovery)
	command.Dir = root
	command.Env = withoutBashEnv(isolatedGitEnvironment())
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "active source must be absent") {
		t.Fatalf("historical recovery did not fail closed for a dangling active-source symlink: %v\n%s", err, output)
	}
	if info, err := os.Lstat(activeSource); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("dangling active-source symlink was not preserved: %v, %v", info, err)
	}
	assertFileContents(t, filepath.Join(nestedSource, "tasks.md"), "historical task bytes\n")
}

func requireArchiveShell(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("archive shell integration is skipped in short mode")
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("archive shell integration requires git: %v", err)
	}
	// Prefer Git's POSIX shell over a possible WSL launcher and verify candidates.
	candidates := []string{filepath.Join(filepath.Dir(gitPath), "..", "bin", "bash.exe")}
	if bashPath, err := exec.LookPath("bash"); err == nil {
		candidates = append(candidates, bashPath)
	}
	for _, shell := range candidates {
		if _, err := os.Stat(shell); err != nil {
			continue
		}
		if err := exec.Command(shell, "-c", "exit 0").Run(); err == nil {
			return shell
		}
	}
	t.Skip("archive shell integration requires a usable POSIX shell")
	return ""
}
func setupArchiveFixture(t *testing.T, tracked bool) (root, source, destination, sentinel string) {
	t.Helper()
	root = t.TempDir()
	sentinel = filepath.Join(root, "repository-sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("exit 99\nrepository sentinel\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source = filepath.Join(root, "openspec", "changes", "change")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "tasks.md"), []byte("archive task bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	destination = filepath.Join(root, "openspec", "changes", "archive", "2030-01-02-change")

	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.name", "Archive Test")
	runGit(t, root, "config", "user.email", "archive-test@example.invalid")
	if tracked {
		runGit(t, root, "add", "--", "repository-sentinel.txt", "openspec/changes/change/tasks.md")
	} else {
		runGit(t, root, "add", "--", "repository-sentinel.txt")
	}
	runGit(t, root, "commit", "-qm", "archive fixture")
	return root, source, destination, sentinel
}

func runArchiveMoveTransaction(shell, root string) (string, error) {
	const changeName = "change"
	transaction := archiveFencedShellBlock("### Step 3: Move to Archive")
	transaction = strings.ReplaceAll(transaction, "{change-name}", changeName)
	transaction = strings.ReplaceAll(transaction, "YYYY-MM-DD-"+changeName, "2030-01-02-"+changeName)
	command := exec.Command(shell, "-c", transaction)
	command.Dir = root
	command.Env = withoutBashEnv(isolatedGitEnvironment())
	output, err := command.CombinedOutput()
	return string(output), err
}

func archiveFencedShellBlock(heading string) string {
	skill := mustReadResolved("skills/sdd-archive/SKILL.md")
	start := strings.Index(skill, heading)
	if start < 0 {
		panic("sdd-archive shell section is missing")
	}
	opening := strings.Index(skill[start:], "```bash\n")
	if opening < 0 {
		panic("sdd-archive shell section is missing its opening fence")
	}
	start += opening + len("```bash\n")
	end := strings.Index(skill[start:], "\n```")
	if end < 0 {
		panic("sdd-archive shell block is missing its closing fence")
	}
	return skill[start : start+end]
}

func createArchiveCollision(t *testing.T, root, destination, collision string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	switch collision {
	case "directory":
		if err := os.MkdirAll(destination, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, "collision-sentinel"), []byte("directory sentinel\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	case "regular file":
		if err := os.WriteFile(destination, []byte("file sentinel\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	case "live symlink":
		target := filepath.Join(root, "live-symlink-target")
		if err := os.WriteFile(target, []byte("live symlink sentinel\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, destination); err != nil {
			t.Skipf("live symlink fixture is unavailable: %v", err)
		}
	case "dangling symlink":
		if err := os.Symlink(filepath.Join(root, "missing-symlink-target"), destination); err != nil {
			t.Skipf("dangling symlink fixture is unavailable: %v", err)
		}
	default:
		t.Fatalf("unknown collision type %q", collision)
	}
}

func assertArchiveCollision(t *testing.T, root, destination, collision string) {
	t.Helper()
	info, err := os.Lstat(destination)
	if err != nil {
		t.Fatalf("collision destination is missing: %v", err)
	}
	switch collision {
	case "directory":
		if !info.IsDir() {
			t.Fatalf("collision destination is %v, want directory", info.Mode())
		}
		assertFileContents(t, filepath.Join(destination, "collision-sentinel"), "directory sentinel\n")
	case "regular file":
		if !info.Mode().IsRegular() {
			t.Fatalf("collision destination is %v, want regular file", info.Mode())
		}
		assertFileContents(t, destination, "file sentinel\n")
	case "live symlink":
		if target, err := os.Readlink(destination); info.Mode()&os.ModeSymlink == 0 || err != nil || target != filepath.Join(root, "live-symlink-target") {
			t.Fatalf("collision destination is %v, want live symlink to %q: target %q, error %v", info.Mode(), filepath.Join(root, "live-symlink-target"), target, err)
		}
		assertFileContents(t, filepath.Join(root, "live-symlink-target"), "live symlink sentinel\n")
	case "dangling symlink":
		if target, err := os.Readlink(destination); info.Mode()&os.ModeSymlink == 0 || err != nil || target != filepath.Join(root, "missing-symlink-target") {
			t.Fatalf("collision destination is %v, want dangling symlink to %q: target %q, error %v", info.Mode(), filepath.Join(root, "missing-symlink-target"), target, err)
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatalf("dangling symlink target is unexpectedly available: %v", err)
		}
	}
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func withoutBashEnv(env []string) []string {
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.ToUpper(env[i]), "BASH_ENV=") {
			env = append(env[:i], env[i+1:]...)
		}
	}
	return env
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = isolatedGitEnvironment()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func assertGitCommandFails(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = isolatedGitEnvironment()
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("git %s unexpectedly succeeded:\n%s", strings.Join(args, " "), output)
	}
}

func isolatedGitEnvironment() []string {
	env := os.Environ()
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.ToUpper(env[i]), "GIT_") {
			env = append(env[:i], env[i+1:]...)
		}
	}
	return append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull, "GIT_CONFIG_COUNT=0")
}

// commitReviewSection returns the canonical body of the shared `Commit Review` section.
func commitReviewSection(t *testing.T) string {
	t.Helper()
	shared := MustRead("skills/_shared/sdd-orchestrator-sections.md")
	const start = "<!-- sdd-orchestrator-section:Commit Review:start -->"
	const end = "<!-- sdd-orchestrator-section:Commit Review:end -->"
	from := strings.Index(shared, start)
	to := strings.Index(shared, end)
	if from < 0 || to < from {
		t.Fatalf("sdd-orchestrator-sections.md has no %q section", "Commit Review")
	}
	return shared[from+len(start) : to]
}

// Threat matrix of aviso-commit: every git order of the doctrine selects its repository with
// `git -C <worktree>`, the staging is explicit, nothing fetches, pushes, forces or commits with -a,
// and the text never mentions RDD nor receipts.
func TestCommitReviewSectionKeepsGitOrdersInsideTheWorktree(t *testing.T) {
	section := commitReviewSection(t)

	orders := regexp.MustCompile("`git [^`]*`").FindAllString(section, -1)
	if len(orders) == 0 {
		t.Fatal("the Commit Review section names no git order")
	}
	for _, order := range orders {
		if order == "`git show <sha>`" || strings.HasPrefix(order, "`git -C <worktree> ") {
			continue
		}
		t.Errorf("git order %s does not start with `git -C <worktree>`", order)
	}
	for _, forbidden := range []string{"commit -a", "commit -m", "--force", "git push"} {
		if strings.Contains(section, forbidden) {
			t.Errorf("Commit Review section must not contain %q", forbidden)
		}
	}
}

// Los marcadores de la doctrina de delegación los resuelve el renderizador de
// internal/components/sdd; sin ellos la doctrina no llega al prompt.
func TestOrchestratorsProjectDelegationMarkers(t *testing.T) {
	paths := allSDDOrchestratorAssetPaths(t)
	if len(paths) != orchestratorTemplateCount {
		t.Fatalf("delegation marker coverage sees %d orchestrators, want %d", len(paths), orchestratorTemplateCount)
	}
	for _, path := range paths {
		content := mustReadResolved(path)
		for _, marker := range []string{"{{AXIOM_DELEGATION_RULES}}", "{{AXIOM_DELEGATION_TRIGGERS}}"} {
			if !strings.Contains(content, marker) {
				t.Errorf("%s missing marker %q", path, marker)
			}
		}
	}
}

// El marcador de la sección «Commit Review» aparece una vez en cada plantilla.
func TestEveryOrchestratorProjectsTheCommitReviewMarker(t *testing.T) {
	// claude/sdd-orchestrator.md es el arranque que remite al workflow: no lleva secciones.
	var paths []string
	for _, path := range allSDDOrchestratorAssetPaths(t) {
		if path != "claude/sdd-orchestrator.md" {
			paths = append(paths, path)
		}
	}
	paths = append(paths, "claude/sdd-orchestrator-workflow.md")
	if len(paths) != orchestratorTemplateCount {
		t.Fatalf("expected %d orchestrators, got %d: %v", orchestratorTemplateCount, len(paths), paths)
	}
	for _, path := range paths {
		if got := strings.Count(MustRead(path), "{{AXIOM_SDD_SECTION:Commit Review}}"); got != 1 {
			t.Errorf("%s carries the Commit Review marker %d times, want 1", path, got)
		}
	}
}

// La sección compartida de puertas lleva el marcador de las tres salidas una
// sola vez, y ninguna plantilla duplica su cuerpo.
func TestSharedGatesSectionCarriesTheStopExitsMarkerOnce(t *testing.T) {
	shared := mustReadResolved("skills/_shared/sdd-orchestrator-sections.md")
	const start = "<!-- sdd-orchestrator-section:SDD Change Kickoff and Block Gates:start -->"
	const end = "<!-- sdd-orchestrator-section:SDD Change Kickoff and Block Gates:end -->"
	from, to := strings.Index(shared, start), strings.Index(shared, end)
	if from < 0 || to < from {
		t.Fatal("sdd-orchestrator-sections.md has no SDD Change Kickoff and Block Gates section")
	}
	if got := strings.Count(shared[from:to], "{{AXIOM_STOP_EXITS}}"); got != 1 {
		t.Errorf("the stop-exits marker appears %d times in the shared gates section, want exactly once", got)
	}
}

// Invariante de seguridad: ningún `fetch` sin autorización explícita de la persona
// en la doctrina de ramas (kickoff, sdd-apply y revisión de commits).
func TestBranchDoctrineNeverFetchesWithoutAuthorization(t *testing.T) {
	shared := mustReadResolved("skills/_shared/sdd-orchestrator-sections.md")
	const start = "<!-- sdd-orchestrator-section:SDD Change Kickoff and Block Gates:start -->"
	const end = "<!-- sdd-orchestrator-section:SDD Change Kickoff and Block Gates:end -->"
	from, to := strings.Index(shared, start), strings.Index(shared, end)
	if from < 0 || to < from {
		t.Fatal("sdd-orchestrator-sections.md has no SDD Change Kickoff and Block Gates section")
	}
	if !strings.Contains(shared[from:to], "Never pass `--fetch` without the explicit authorization of the user.") {
		t.Error("the shared kickoff section must forbid `--fetch` without the explicit authorization of the user")
	}
	if !strings.Contains(commitReviewSection(t), "never fetch") {
		t.Error("the Commit Review section must keep `never fetch`")
	}
	capable := extractModelSectionForTest(mustReadResolved("skills/sdd-apply/SKILL.md"), "capable")
	if !strings.Contains(capable, "Never pass `--fetch` without the explicit authorization of the user.") {
		t.Error("sdd-apply must forbid `--fetch` without the explicit authorization of the user")
	}
}

// Concesión de herramientas del agente de investigación: es el campo `tools:` del
// frontmatter que lee Claude Code, y solo puede abrir lectura web.
func TestSDDResearchAgentToolGrantIsExact(t *testing.T) {
	const want = "tools: WebFetch, WebSearch"
	var tools string
	for _, line := range strings.Split(mustReadResolved("claude/agents/sdd-research.md"), "\n") {
		if strings.HasPrefix(line, "tools:") {
			tools = line
			break
		}
	}
	if tools != want {
		t.Fatalf("claude/agents/sdd-research.md tools = %q, want %q", tools, want)
	}
}

// RF-PUB-12 (C-05-c): la doctrina ordena ejecutar `publication.invocation` una sola vez antes de
// lanzar la fase siguiente y otra al reanudar, presenta sin ejecutar la orden de push de
// `code_unpublished`, trata `unavailable` como información y no deja que el campo cambie la ruta;
// el contrato del status documenta el campo `publication`.
func TestPendingPublicationDoctrineAndStatusContract(t *testing.T) {
	sections := mustReadResolved("skills/_shared/sdd-orchestrator-sections.md")
	for _, want := range []string{
		"When the status projects `publication` with `state: pending`, run its `publication.invocation` exactly once before launching the next phase and exactly once when resuming the change",
		"its refusal is informational, never undoes an approval and never stops the chain",
		"With `state: code_unpublished`, present its `pushCommand` to the person and never run it",
		"with `state: unavailable`, present its code as information",
		"The field never changes `nextRecommended`.",
	} {
		if !strings.Contains(sections, want) {
			t.Errorf("shared gates section missing the pending-publication rule %q", want)
		}
	}
	// El texto vive una sola vez, dentro del párrafo «When to publish», y no toca la envoltura de la compuerta.
	if got := strings.Count(sections, "run its `publication.invocation` exactly once"); got != 1 {
		t.Errorf("the pending-publication rule appears %d times in the shared gates section, want exactly once", got)
	}
	paragraph := ""
	for _, line := range strings.Split(sections, "\n") {
		if strings.HasPrefix(line, "When to publish.") {
			paragraph = line
		}
	}
	if !strings.Contains(paragraph, "run its `publication.invocation` exactly once") {
		t.Error("the pending-publication rule must close the \"When to publish\" paragraph, not a new one")
	}

	contract := mustReadResolved("skills/_shared/sdd-status-contract.md")
	for _, want := range []string{
		"- `publication` (optional, at the root of the status, next to `gateQuestion`)",
		"calculated with reads only (no network, no writes)",
		"`state` is `pending`",
		"`code_unpublished`",
		"`unavailable`",
		"`docs_uncommitted`, `branch_ahead`, `remote_branch_missing`, `code_commits` and `range_too_big`",
		"`invocation`, the same as `gateQuestion.publication.invocation`",
		"`pushCommand` for the person",
		"It is omitted when there is nothing to commit or push",
		"`nextRecommended`, `dependencies` and `blockedReasons` never depend on it",
	} {
		if !strings.Contains(contract, want) {
			t.Errorf("status contract missing the publication field rule %q", want)
		}
	}
}
