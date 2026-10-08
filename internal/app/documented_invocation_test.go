package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
)

// This corpus exists because of #2506: the product states runnable commands
// in shipped assets and docs, and none of them was ever executed by a test.
// The invocation #2473 broke lived only in documentation, so it broke without
// a single test going red. The corpus is EXTRACTED, never enumerated: adding
// a command to an asset or doc adds it to the corpus automatically.
//
// Every extracted invocation lands in exactly one tier, decided mechanically:
//
//   - executed: a verb app.RunArgs dispatches without system detection, with
//     no placeholder left after the repository slot is filled. It runs
//     in-process against a fresh repository and must not be refused by the
//     parser or answered with a non-retryable stop failure envelope.
//   - parse-only: same safe verbs, but state-bearing placeholders remain
//     (<lineage>, <token>, ...) whose valid values a test cannot invent.
//     Placeholders are dummy-filled and the REAL parser must accept the flag
//     surface; negotiated review verbs are additionally checked against the
//     operation registry's declared flags.
//   - presence-only: everything a test cannot honestly run: shell-composed
//     examples (redirection, expansion, alternation), verbs that reach the
//     network, prompt, or another binary. Counted so the tiers stay visible.

type documentedInvocation struct {
	source  string
	command string
}

// --- extraction -----------------------------------------------------------

var inlineInvocationRegexp = regexp.MustCompile("`((?:gentle-ai|axiom) [^`\n]+)`")

func extractInvocations(source, content string) []documentedInvocation {
	var out []documentedInvocation
	lines := strings.Split(content, "\n")
	fenced := false
	for index := 0; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		at := fmt.Sprintf("%s:%d", source, index+1)
		if strings.HasPrefix(trimmed, "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			command := strings.TrimPrefix(trimmed, "$ ")
			for strings.HasSuffix(command, "\\") && index+1 < len(lines) {
				index++
				command = strings.TrimSuffix(command, "\\") + " " + strings.TrimSpace(lines[index])
			}
			if strings.HasPrefix(command, "gentle-ai ") || strings.HasPrefix(command, "axiom ") {
				out = append(out, documentedInvocation{source: at, command: command})
			}
			continue
		}
		for _, match := range inlineInvocationRegexp.FindAllStringSubmatch(lines[index], -1) {
			out = append(out, documentedInvocation{source: at, command: match[1]})
		}
	}
	return out
}

func collectDocumentedInvocations(t *testing.T) []documentedInvocation {
	t.Helper()
	var corpus []documentedInvocation
	// Embedded assets: the channel rendered into every installed agent's rules.
	err := fs.WalkDir(assets.FS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		content, readErr := fs.ReadFile(assets.FS, path)
		if readErr != nil {
			return readErr
		}
		corpus = append(corpus, extractInvocations("assets:"+path, string(content))...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// docs/: the human-facing instruction surface. docs/audits are dated
	// snapshots of past investigations, not living instructions, so they are
	// deliberately outside the runnable corpus.
	err = filepath.WalkDir(filepath.Join("..", "..", "docs"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "audits" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		corpus = append(corpus, extractInvocations(filepath.ToSlash(path), string(content))...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(corpus) == 0 {
		t.Fatal("extracted no documented invocations from assets or docs; the extractor is stale")
	}
	return corpus
}

// --- classification -------------------------------------------------------

var placeholderRegexp = regexp.MustCompile(`<[a-zA-Z][a-zA-Z0-9_ .-]*>`)
var optionalWordRegexp = regexp.MustCompile(`^\[[a-z-]+\]$`)

const documentedRuntimeAgentIDPlaceholder = "{{GENTLE_AI_RUNTIME_AGENT_ID}}"
const documentedRuntimeAgentID = "opencode"

func wordNeedsShell(word string) bool {
	switch word {
	case ">", ">>", "2>", "<", "|", "||", "&&", ";":
		return true
	}
	return strings.ContainsAny(word, "$`|;&<>")
}

// platformIndependentVerbs extracts the case labels app.RunArgs dispatches
// BEFORE system detection, straight from app.go's own dispatch source, the
// same way review_integration_contract_guard_test.go extracts dispatch verbs.
// The set is derived, not maintained; a verb added to the switch joins it.
func platformIndependentVerbs(t *testing.T) map[string]bool {
	t.Helper()
	source, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	begin := strings.Index(text, "// Platform-independent commands")
	end := strings.Index(text, "ensureCurrentOSSupported(")
	if begin < 0 || end < 0 || end < begin {
		t.Fatal("app.go's platform-independent dispatch section moved; update the extraction anchors")
	}
	verbs := map[string]bool{}
	for _, match := range regexp.MustCompile(`case "([a-z][a-z-]*)"`).FindAllStringSubmatch(text[begin:end], -1) {
		verbs[match[1]] = true
	}
	if len(verbs) == 0 {
		t.Fatal("extracted no platform-independent verbs from app.go")
	}
	return verbs
}

// documentedInvocationExecutionExclusions holds the platform-independent
// verbs still kept out of in-process execution, each with its reason. This is
// a policy judgment, not a mirror of truth computed elsewhere: nothing else
// in the tree records which verbs prompt, shell out, or fall through.
var documentedInvocationExecutionExclusions = map[string]string{
	"install":   "its case only prints help; a real install falls through to system detection and the TUI",
	"uninstall": "may prompt interactively and mutates installed agent state",
	"codegraph": "proxies to the external codegraph binary, which CI does not install",
}

type invocationTier string

const (
	tierExecuted invocationTier = "executed"
	tierParse    invocationTier = "parse-only"
	tierPresence invocationTier = "presence-only"
)

// classifyWords rewrites the repository slot to repo, drops optional
// positional words, and reports the tier plus the words to run. The shell
// check runs on each word with its placeholders stripped, so a templated
// value never reads as shell syntax while real redirection and expansion do.
func classifyWords(words []string, safeVerbs map[string]bool, repo string) ([]string, invocationTier) {
	if len(words) < 2 || !safeVerbs[words[1]] || documentedInvocationExecutionExclusions[words[1]] != "" {
		return nil, tierPresence
	}
	rewritten := make([]string, 0, len(words))
	placeholders := false
	for index := 0; index < len(words); index++ {
		word := words[index]
		if word == "..." {
			return nil, tierPresence
		}
		if optionalWordRegexp.MatchString(word) {
			continue
		}
		switch {
		case word == "--cwd" && index+1 < len(words):
			rewritten = append(rewritten, word, repo)
			index++
			continue
		case strings.HasPrefix(word, "--cwd="):
			rewritten = append(rewritten, "--cwd="+repo)
			continue
		case strings.Contains(word, documentedRuntimeAgentIDPlaceholder):
			// The placeholder appears only in the shared review contract. Its
			// OpenCode rendering is pinned by TestGoldenSDD_OpenCode_Multi, so
			// execute the documented command with that rendered runtime identity.
			word = strings.ReplaceAll(word, documentedRuntimeAgentIDPlaceholder, documentedRuntimeAgentID)
		}
		if wordNeedsShell(placeholderRegexp.ReplaceAllString(word, "")) {
			return nil, tierPresence
		}
		if placeholderRegexp.MatchString(word) {
			// A placeholder is only substitutable where a value belongs: in a
			// --flag word or as the value of the bare flag before it. In a
			// verb or positional slot the command's own identity is
			// templated ("gentle-ai review <verb>"), a reference to a family
			// of commands rather than a runnable claim.
			bareFlagBefore := index > 0 && strings.HasPrefix(words[index-1], "--") && !strings.Contains(words[index-1], "=")
			if !strings.HasPrefix(word, "--") && !bareFlagBefore {
				return nil, tierPresence
			}
			placeholders = true
			word = placeholderRegexp.ReplaceAllString(word, "dummy")
		}
		rewritten = append(rewritten, word)
	}
	if placeholders {
		return rewritten, tierParse
	}
	return rewritten, tierExecuted
}

// --- predicates -----------------------------------------------------------

// isParseRejection reports whether the CLI refused the invocation at the
// parser, which means the documented command does not run as printed. A
// semantic refusal (missing state, absent lineage) proves the opposite: the
// parser accepted every documented flag before the refusal happened.
func isParseRejection(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	for _, signature := range []string{
		"flag provided but not defined",
		"unknown command",
		"unknown review command",
		"unknown review mode command",
		"unexpected review",
	} {
		if strings.Contains(message, signature) {
			return true
		}
	}
	return false
}

// nonRetryableStop scans every JSON document in output for the failure-
// envelope signature #2473 shipped with: retry_safe false with next_action
// "stop". A negotiated next_transition of kind "stop" is a routed answer and
// deliberately not matched; the guard hunts dead ends, not routing.
func nonRetryableStop(output []byte) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var document any
		if err := decoder.Decode(&document); err != nil {
			return "", false
		}
		if found, ok := findNonRetryableStop(document); ok {
			return found, true
		}
	}
}

func findNonRetryableStop(document any) (string, bool) {
	switch value := document.(type) {
	case map[string]any:
		retry, hasRetry := value["retry_safe"].(bool)
		action, hasAction := value["next_action"].(string)
		if hasRetry && hasAction && !retry && action == "stop" {
			encoded, _ := json.Marshal(value)
			return string(encoded), true
		}
		for _, child := range value {
			if found, ok := findNonRetryableStop(child); ok {
				return found, true
			}
		}
	case []any:
		for _, child := range value {
			if found, ok := findNonRetryableStop(child); ok {
				return found, true
			}
		}
	}
	return "", false
}

// --- sandbox --------------------------------------------------------------

// newDocumentedInvocationSandbox builds the fixture repository and home for one
// documented invocation. The sandbox is removed by documentedSandbox.cleanup
// instead of t.TempDir so that a removal failure names what was still being
// written (see that method). command is the invocation about to run in it.
func newDocumentedInvocationSandbox(t *testing.T, command string) (string, *documentedSandbox) {
	t.Helper()
	root, err := os.MkdirTemp("", "documented-invocation-*")
	if err != nil {
		t.Fatal(err)
	}
	sandbox := &documentedSandbox{root: root, started: time.Now(), command: command, previous: lastDocumentedCommand}
	lastDocumentedCommand = command
	// Registered before every t.Setenv/t.Chdir below, so (cleanups run last in,
	// first out) the working directory has left the sandbox before it is removed.
	t.Cleanup(func() { sandbox.cleanup(t) })
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")
	for _, dir := range []string{repo, home} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	// Every home-like location the product resolves stays inside the sandbox.
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(home, ".axiom"))
	t.Setenv("AXIOM_NO_PERSISTENT_PATH", "1")
	// Dependency detection runs `go version` (internal/system/deps.go), and a
	// go toolchain with telemetry enabled writes counter files under its user
	// config dir from a detached child that outlives the command, racing the
	// sandbox removal ("directory not empty" in CI). Turn Go telemetry off in
	// every user config dir the toolchain may resolve, exactly as
	// `go telemetry off` would, so no go invocation writes here at all.
	for _, configDir := range []string{
		filepath.Join(home, ".config"),
		filepath.Join(home, "AppData", "Roaming"),
		filepath.Join(home, "Library", "Application Support"),
	} {
		modeDir := filepath.Join(configDir, "go", "telemetry")
		if err := os.MkdirAll(modeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(modeDir, "mode"), []byte("off"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) {
		command := exec.Command("git", args...)
		command.Dir = repo
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	git("init", "-q")
	git("config", "user.email", "corpus@example.invalid")
	git("config", "user.name", "corpus")
	if err := os.WriteFile(filepath.Join(repo, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-q", "-m", "seed")
	t.Chdir(repo)
	// Restrict PATH only after the fixture repository exists (git is needed to
	// build it): the verbs under test (sync, setup) probe agent runtimes and
	// helper tools through PATH, and on a developer machine that means running
	// the real npm-shimmed opencode/claude/engram binaries. Measured on Windows,
	// one `sync --agent opencode` took ~49s with the real PATH and ~6s with the
	// restricted one, so the whole corpus (~80 subprocess-heavy invocations)
	// ran for many minutes and looked like it never terminated.
	t.Setenv("PATH", hermeticDocumentedInvocationPath(t, root))
	return repo, sandbox
}

// hermeticDocumentedInvocationPath returns a PATH that holds an empty directory
// owned by the sandbox, the directory of the git binary the product legitimately
// shells out to, and the operating system's own tool directories. No agent
// runtime or package-manager shim is reachable through it.
func hermeticDocumentedInvocationPath(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	entries := []string{bin}
	if gitPath, err := exec.LookPath("git"); err == nil {
		entries = append(entries, filepath.Dir(gitPath))
	}
	if runtime.GOOS == "windows" {
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		entries = append(entries, filepath.Join(systemRoot, "System32"), systemRoot)
	} else {
		entries = append(entries, "/usr/bin", "/bin")
	}
	return strings.Join(entries, string(os.PathListSeparator))
}

// documentedInvocationBudget bounds one in-process documented invocation. A
// healthy one takes seconds even on a loaded machine; the budget exists so a
// future hang fails loudly, naming the command, instead of stalling the run
// until the package-level test timeout kills it with no attribution.
const documentedInvocationBudget = 3 * time.Minute

// runDocumentedInvocation runs app.RunArgs and fails the test when it does not
// return within documentedInvocationBudget. RunArgs has no context to cancel, so
// a stuck call keeps its goroutine until the test binary exits; the failure is
// still reported at once with the command and the budget.
func runDocumentedInvocation(t *testing.T, args []string) (output []byte, runErr error) {
	t.Helper()
	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, 1)
	go func() {
		var buffer bytes.Buffer
		err := app.RunArgs(args[1:], &buffer)
		done <- result{buffer.Bytes(), err}
	}()
	select {
	case r := <-done:
		return r.output, r.err
	case <-time.After(documentedInvocationBudget):
		t.Fatalf("documented invocation did not return within %v: %s", documentedInvocationBudget, strings.Join(args, " "))
		return nil, nil
	}
}

// lastDocumentedCommand is the invocation the previous sandbox ran, so a cleanup
// failure can also name the neighbour that may have left work behind.
var lastDocumentedCommand string

const (
	// documentedCleanupRetryDelay is how long a failed removal waits before the
	// single retry that exists only to show whether the tree was still changing.
	documentedCleanupRetryDelay = 250 * time.Millisecond
	// documentedGoroutineDumpLimit bounds the goroutine listing in the report.
	documentedGoroutineDumpLimit = 24 << 10
)

// documentedSandbox owns one invocation's scratch tree. Removing it used to be
// t.TempDir's job, which on Linux CI intermittently failed with "directory not
// empty" and no hint of who was still writing. cleanup keeps the same failure
// (it never swallows or hides the error) but attaches the evidence.
type documentedSandbox struct {
	root        string
	started     time.Time
	command     string
	previous    string
	runDuration time.Duration
}

func (s *documentedSandbox) cleanup(t *testing.T) {
	if t.Failed() {
		t.Logf("RunArgs wall time for %q: %v", s.command, s.runDuration)
	}
	removeStart := time.Now()
	err := os.RemoveAll(s.root)
	if err == nil {
		return
	}
	first := s.tree(removeStart)
	processes := s.processesInside()
	goroutines := goroutineDump()
	time.Sleep(documentedCleanupRetryDelay)
	retryStart := time.Now()
	second := s.tree(retryStart)
	retryErr := os.RemoveAll(s.root)
	t.Errorf("sandbox cleanup failed: %v\n"+
		"invocation:          %s\n"+
		"previous invocation: %s\n"+
		"RunArgs wall time:   %v\n"+
		"sandbox age at removal start: %v\n"+
		"remaining tree at the failure (mtime as time, offset from sandbox start, offset from RemoveAll start):\n%s"+
		"processes with cwd or an open fd inside the sandbox:\n%s"+
		"goroutines:\n%s\n"+
		"tree %v later, before the single retry (still changing => a live writer):\n%s"+
		"retry RemoveAll: %v",
		err, s.command, s.previous, s.runDuration, removeStart.Sub(s.started),
		first, processes, goroutines, retryStart.Sub(removeStart), second, retryErr)
}

// tree lists what is left under the sandbox, one entry per line.
func (s *documentedSandbox) tree(removeStart time.Time) string {
	var b strings.Builder
	_ = filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		rel, _ := filepath.Rel(s.root, path)
		if walkErr != nil {
			fmt.Fprintf(&b, "  %s: %v\n", rel, walkErr)
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			fmt.Fprintf(&b, "  %s: %v\n", rel, err)
			return nil
		}
		kind := "file"
		switch {
		case info.IsDir():
			kind = "dir"
		case info.Mode()&fs.ModeSymlink != 0:
			kind = "symlink"
		}
		mtime := info.ModTime()
		fmt.Fprintf(&b, "  %-48s %-7s %8d B  %s  %+.6fs  %+.6fs\n", rel, kind, info.Size(),
			mtime.Format("15:04:05.000000000"), mtime.Sub(s.started).Seconds(), mtime.Sub(removeStart).Seconds())
		return nil
	})
	return b.String()
}

// processesInside lists the processes whose working directory or open file
// descriptors point into the sandbox. Only Linux exposes this through /proc.
func (s *documentedSandbox) processesInside() string {
	if runtime.GOOS != "linux" {
		return "  (not collected on " + runtime.GOOS + ")\n"
	}
	root, err := filepath.EvalSymlinks(s.root)
	if err != nil {
		root = s.root
	}
	inside := func(target string) bool {
		target = strings.TrimSuffix(target, " (deleted)")
		return target == root || strings.HasPrefix(target, root+string(os.PathSeparator))
	}
	var b strings.Builder
	processes, _ := os.ReadDir("/proc")
	for _, process := range processes {
		dir := filepath.Join("/proc", process.Name())
		var holds []string
		if target, err := os.Readlink(filepath.Join(dir, "cwd")); err == nil && inside(target) {
			holds = append(holds, "cwd="+target)
		}
		fds, _ := os.ReadDir(filepath.Join(dir, "fd"))
		for _, fd := range fds {
			if target, err := os.Readlink(filepath.Join(dir, "fd", fd.Name())); err == nil && inside(target) {
				holds = append(holds, "fd"+fd.Name()+"="+target)
			}
		}
		if len(holds) == 0 {
			continue
		}
		comm, _ := os.ReadFile(filepath.Join(dir, "comm"))
		cmdline, _ := os.ReadFile(filepath.Join(dir, "cmdline"))
		fmt.Fprintf(&b, "  pid=%s ppid=%s comm=%s cmdline=%q holds %s\n", process.Name(), parentPID(dir),
			strings.TrimSpace(string(comm)), strings.ReplaceAll(string(cmdline), "\x00", " "), strings.Join(holds, ", "))
	}
	if b.Len() == 0 {
		return "  (none)\n"
	}
	return b.String()
}

// parentPID reads the parent from /proc/<pid>/stat, whose comm field may itself
// contain spaces and parentheses, hence the search from the last ')'.
func parentPID(procDir string) string {
	stat, err := os.ReadFile(filepath.Join(procDir, "stat"))
	if err != nil {
		return "?"
	}
	text := string(stat)
	fields := strings.Fields(text[strings.LastIndex(text, ")")+1:])
	if len(fields) < 2 {
		return "?"
	}
	return fields[1]
}

func goroutineDump() string {
	buffer := make([]byte, 1<<20)
	dump := string(buffer[:runtime.Stack(buffer, true)])
	if len(dump) > documentedGoroutineDumpLimit {
		dump = dump[:documentedGoroutineDumpLimit] + "\n... (truncated)"
	}
	return dump
}

// registrySurfaceViolation cross-checks a documented negotiated review verb
// against the operation registry's declared flag surface. --contract is the
// facade's own routing flag and is declared on the FlagSets, not the rows.
func registrySurfaceViolation(words []string) string {
	if len(words) < 3 || words[1] != "review" {
		return ""
	}
	surface, found := cli.ReviewOperationFlagSurface(words[2])
	if !found {
		return ""
	}
	for _, word := range words[3:] {
		if !strings.HasPrefix(word, "--") {
			continue
		}
		name := strings.TrimPrefix(word, "--")
		if index := strings.Index(name, "="); index >= 0 {
			name = name[:index]
		}
		if name != "contract" && !surface[name] {
			return fmt.Sprintf("documents --%s, which the operation registry does not declare for review %s", name, words[2])
		}
	}
	return ""
}

// --- the guard ------------------------------------------------------------

// documentedInvocationKnownFailures is the quarantined worklist for #2506:
// commands the product documents today that do not survive their own tier on
// current main. Every entry is a defect to burn down, never an accepted
// state. An entry whose invocation starts passing must be removed; the guard
// fails on stale entries so the list can only shrink.
var documentedInvocationKnownFailures = map[string]string{}

func TestDocumentedInvocationsRunAsDocumented(t *testing.T) {
	corpus := collectDocumentedInvocations(t)
	safeVerbs := platformIndependentVerbs(t)

	// Identical command text is exercised once; sources accumulate so a
	// failure names every place the broken form is documented.
	sources := map[string][]string{}
	var unique []string
	for _, invocation := range corpus {
		if _, seen := sources[invocation.command]; !seen {
			unique = append(unique, invocation.command)
		}
		sources[invocation.command] = append(sources[invocation.command], invocation.source)
	}
	sort.Strings(unique)

	counts := map[invocationTier]int{}
	quarantined := 0
	staleQuarantine := map[string]bool{}
	for command := range documentedInvocationKnownFailures {
		staleQuarantine[command] = true
	}

	for _, command := range unique {
		words, err := cli.SplitPrintedCommandWords(command)
		if err != nil {
			counts[tierPresence]++
			continue
		}
		repo := "<repo>"
		args, tier := classifyWords(words, safeVerbs, repo)
		counts[tier]++
		if tier == tierPresence {
			continue
		}
		t.Run(string(tier)+"/"+strings.Join(words[1:min(3, len(words))], " "), func(t *testing.T) {
			failure := ""
			if violation := registrySurfaceViolation(words); violation != "" {
				failure = violation
			} else {
				sandbox, diagnostics := newDocumentedInvocationSandbox(t, command)
				for index, arg := range args {
					args[index] = strings.ReplaceAll(arg, repo, sandbox)
				}
				began := time.Now()
				output, runErr := runDocumentedInvocation(t, args)
				diagnostics.runDuration = time.Since(began)
				if isParseRejection(runErr) {
					failure = fmt.Sprintf("the parser refuses it as printed: %v", runErr)
				} else if tier == tierExecuted {
					if envelope, dead := nonRetryableStop(output); dead && !strings.Contains(envelope, "immutable_review_transport_unsupported") {
						failure = "answers a non-retryable stop on a fresh repository: " + envelope
					}
				}
			}
			if reason, known := documentedInvocationKnownFailures[command]; known {
				delete(staleQuarantine, command)
				if failure == "" {
					t.Errorf("quarantined invocation now passes; remove it from documentedInvocationKnownFailures (%s)", reason)
					return
				}
				quarantined++
				t.Logf("quarantined (#2506 worklist, %s): %s", reason, failure)
				return
			}
			if failure != "" {
				t.Errorf("documented at %s\n  %s\n  %s", strings.Join(sources[command], ", "), command, failure)
			}
		})
	}

	for command := range staleQuarantine {
		t.Errorf("documentedInvocationKnownFailures names %q, which the corpus no longer contains", command)
	}
	if counts[tierExecuted] == 0 {
		t.Error("no documented invocation reached the executed tier; the classification is stale")
	}
	t.Logf("documented invocation corpus: %d unique commands (%d occurrences): executed %d, parse-only %d, presence-only %d, quarantined %d",
		len(unique), len(corpus), counts[tierExecuted], counts[tierParse], counts[tierPresence], quarantined)
}
