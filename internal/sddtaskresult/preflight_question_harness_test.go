package sddtaskresult

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
)

// preflightQuestionHarnessSource drives the actual embedded OpenCode plugin
// (v1 generation), calling its tool.execute hooks the way the host would: it
// canonicalizes a localized grouped preflight question set, accepts tolerant
// answers and prepends the confirmed block to a later SDD task dispatch, and
// surfaces the errors thrown for a skipped answer and for a missing required
// group. The groups the plugin requires depend on the topology of the project
// directory: this harness runs in a monorepo-embedded project (axiom.yaml with
// no topology), which asks Pace and PR strategy. The plugin's
// `import type { Plugin }` line is type-only; the copy under test keeps it, so
// this harness only needs a default export that returns the two tool.execute
// hooks.
const preflightQuestionHarnessSource = `import Factory from "./sdd-task-result-artifacts.ts"

type Hooks = {
  "tool.execute.before": (input: any, output: any) => Promise<void>
  "tool.execute.after": (input: any, output: any) => Promise<void>
}

async function main() {
  const client = {
    session: {
      get: async ({ path }: { path: { id: string } }) => ({ data: { id: path.id, parentID: undefined } }),
    },
  }
  // The plugin is inert outside an Axiom project; the test directory carries an axiom.yaml.
  const hooks = (await (Factory as any)({ client, directory: process.cwd() })) as Hooks

  const out: Record<string, unknown> = {}

  // Scenario A: a localized question set whose first question carries the
  // legacy numbered marker with the legacy "Gentle AI" prefix and whose second
  // one carries the nominal marker becomes two canonical questions with the
  // nominal Axiom marker. A question with no marker is never assigned to a
  // group by position: it is left untouched.
  const argsA: any = {
    questions: [
      { question: "Gentle AI SDD preflight 1/3: ¿Qué ritmo prefieres?", options: [{ label: "x" }], multiple: true },
      { question: "¿Qué artefactos usamos?", options: [{ label: "a" }, { label: "b" }, { label: "c" }] },
      { question: "Axiom SDD preflight pr-strategy: ¿Cómo entregamos el PR?", options: [] },
    ],
  }
  await hooks["tool.execute.before"]({ tool: "question", sessionID: "root-a", callID: "call-a" }, { args: argsA })
  out.canonicalized = argsA.questions

  // Scenario B: tolerant answers are accepted, bound by label, and the
  // confirmed block is prepended to a later SDD task dispatch with the store
  // fixed to hybrid. The answers are positional to the questions, so the
  // unmarked second question carries an unused answer.
  await hooks["tool.execute.after"](
    { tool: "question", sessionID: "root-a", args: argsA },
    { title: "q", output: "", metadata: { answers: [["automatic"], ["ignored"], ["ask me"]] } },
  )
  const dispatchArgs: any = { subagent_type: "sdd-explore", prompt: "do the sdd-explore work" }
  await hooks["tool.execute.before"]({ tool: "task", sessionID: "root-a", callID: "call-b" }, { args: dispatchArgs })
  out.dispatchedPrompt = dispatchArgs.prompt

  // Scenario C: an empty answer array (a skipped tab) throws.
  const argsC: any = {
    questions: [
      { question: "Gentle AI SDD preflight 1/3: Pace", options: [{ label: "Interactive" }, { label: "Automatic" }] },
      { question: "Gentle AI SDD preflight 3/3: PR strategy", options: [{ label: "Ask me" }, { label: "Single PR" }, { label: "Auto" }] },
    ],
  }
  await hooks["tool.execute.before"]({ tool: "question", sessionID: "root-c", callID: "call-c" }, { args: argsC })
  try {
    await hooks["tool.execute.after"](
      { tool: "question", sessionID: "root-c", args: argsC },
      { title: "q", output: "", metadata: { answers: [[], ["Ask me"]] } },
    )
    out.emptyAnswerError = null
  } catch (err: any) {
    out.emptyAnswerError = err instanceof Error ? err.message : String(err)
  }

  // Scenario D: a question set without the pr-strategy group throws in the
  // before-hook, naming the missing required group.
  const argsD: any = {
    questions: [
      { question: "Gentle AI SDD preflight 1/3: Pace", options: [{ label: "Interactive" }, { label: "Automatic" }] },
      { question: "Gentle AI SDD preflight 2/3: Artifacts", options: [{ label: "OpenSpec" }, { label: "Engram" }, { label: "Both" }] },
    ],
  }
  try {
    await hooks["tool.execute.before"]({ tool: "question", sessionID: "root-d", callID: "call-d" }, { args: argsD })
    out.missingGroupError = null
  } catch (err: any) {
    out.missingGroupError = err instanceof Error ? err.message : String(err)
  }

  // Scenario E: model-supplied descriptions follow their matching label,
  // never their array position. Q1 offers the canonical labels in swapped
  // order; Q3 offers localized labels that match nothing and must fall back
  // to the built-in descriptions. The legacy Artifacts question is ignored
  // and left untouched.
  const argsE: any = {
    questions: [
      { question: "Gentle AI SDD preflight 1/3: Pace", header: "Pace", options: [{ label: "automatic", description: "DESC-AUTO" }, { label: "Interactive", description: "DESC-INTERACTIVE" }] },
      { question: "Gentle AI SDD preflight 2/3: Artifacts", header: "Artifacts", options: [{ label: "Ambos", description: "DESC-AMBOS" }, { label: "Engram", description: "DESC-ENGRAM" }, { label: "OpenSpec", description: "DESC-OPENSPEC" }] },
      { question: "Gentle AI SDD preflight 3/3: PR strategy", header: "PR strategy", options: [{ label: "Pregúntame", description: "DESC-ASK" }, { label: "PR único", description: "DESC-SINGLE" }, { label: "Automático", description: "DESC-AUTO-PR" }] },
    ],
  }
  await hooks["tool.execute.before"]({ tool: "question", sessionID: "root-e", callID: "call-e" }, { args: argsE })
  out.reordered = argsE.questions

  // Scenario F: the Axiom-branded prefix with the nominal markers (the ones
  // the plugin emits) is recognized, canonicalized, answered and the confirmed
  // block reaches the next SDD task dispatch. Scenarios A-E above cover the
  // legacy prefix and the legacy numbered markers.
  const argsF: any = {
    questions: [
      { question: "Axiom SDD preflight pace: Pace", options: [{ label: "Interactive" }, { label: "Automatic" }] },
      { question: "Axiom SDD preflight pr-strategy: PR strategy", options: [{ label: "Ask me" }, { label: "Single PR" }, { label: "Auto" }] },
    ],
  }
  await hooks["tool.execute.before"]({ tool: "question", sessionID: "root-f", callID: "call-f" }, { args: argsF })
  out.axiomCanonicalized = argsF.questions
  await hooks["tool.execute.after"](
    { tool: "question", sessionID: "root-f", args: argsF },
    { title: "q", output: "", metadata: { answers: [["Automatic"], ["Single PR"]] } },
  )
  const dispatchF: any = { subagent_type: "sdd-explore", prompt: "do the sdd-explore work" }
  await hooks["tool.execute.before"]({ tool: "task", sessionID: "root-f", callID: "call-f2" }, { args: dispatchF })
  out.axiomDispatchedPrompt = dispatchF.prompt

  process.stdout.write(JSON.stringify(out))
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
`

type preflightHarnessQuestion struct {
	Question string `json:"question"`
	Header   string `json:"header"`
	Multiple bool   `json:"multiple"`
	Options  []struct {
		Label       string `json:"label"`
		Description string `json:"description"`
	} `json:"options"`
}

type preflightHarnessResult struct {
	Canonicalized         []preflightHarnessQuestion `json:"canonicalized"`
	DispatchedPrompt      string                     `json:"dispatchedPrompt"`
	EmptyAnswerError      *string                    `json:"emptyAnswerError"`
	MissingGroupError     *string                    `json:"missingGroupError"`
	Reordered             []preflightHarnessQuestion `json:"reordered"`
	AxiomCanonicalized    []preflightHarnessQuestion `json:"axiomCanonicalized"`
	AxiomDispatchedPrompt string                     `json:"axiomDispatchedPrompt"`
}

// typeScriptRuntime returns the command that runs a TypeScript module: bun
// when it is on PATH, otherwise node, which strips the types itself. The
// plugin harness is skipped only when neither is installed.
func typeScriptRuntime(t *testing.T) (string, []string) {
	t.Helper()
	if bunPath, err := exec.LookPath("bun"); err == nil {
		return bunPath, nil
	}
	if nodePath, err := exec.LookPath("node"); err == nil {
		return nodePath, []string{"--experimental-strip-types"}
	}
	t.Skip("neither bun nor node is on PATH; skipping OpenCode plugin harness test")
	return "", nil
}

// TestOpenCodePreflightQuestionCanonicalizationAndTolerantMatching runs the
// actual embedded OpenCode plugin through bun (or node when bun is not
// installed), proving the runtime-owned canonicalization and tolerant-answer
// matching behave the way the plugin contract in this package describes,
// rather than only pinning literal source substrings. It is skipped when
// neither runtime is on PATH.
func TestOpenCodePreflightQuestionCanonicalizationAndTolerantMatching(t *testing.T) {
	runtimePath, runtimeArgs := typeScriptRuntime(t)

	pluginSource := assets.MustRead("opencode/plugins/sdd-task-result-artifacts.ts")
	// The Plugin import is type-only and the runtime erases it, but strip it
	// defensively in case it ever tries to resolve the package.
	pluginSource = strings.Replace(pluginSource, `import type { Plugin } from "@opencode-ai/plugin"`+"\n", "", 1)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sdd-task-result-artifacts.ts"), []byte(pluginSource), 0o644); err != nil {
		t.Fatalf("write plugin copy: %v", err)
	}
	// The plugin only acts inside an Axiom project, so the harness directory is one. With no declared
	// topology it is governed as monorepo-embedded, which asks the pace and pr-strategy groups.
	if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte("name: harness\n"), 0o644); err != nil {
		t.Fatalf("write axiom.yaml: %v", err)
	}
	harnessPath := filepath.Join(dir, "harness.ts")
	if err := os.WriteFile(harnessPath, []byte(preflightQuestionHarnessSource), 0o644); err != nil {
		t.Fatalf("write harness: %v", err)
	}

	cmd := exec.Command(runtimePath, append(runtimeArgs, harnessPath)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TMPDIR="+dir, "TEMP="+dir, "TMP="+dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("plugin harness failed: %v\n%s", err, output)
	}

	var result preflightHarnessResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode harness output: %v\nraw output:\n%s", err, output)
	}

	// Scenario 1: the two marked questions become canonical questions with
	// the nominal marker; the unmarked one is left exactly as the model wrote it.
	if len(result.Canonicalized) != 3 {
		t.Fatalf("expected 3 questions after canonicalization, got %d", len(result.Canonicalized))
	}
	canonical := map[int]struct {
		marker, header string
		labels         []string
	}{
		0: {"Axiom SDD preflight pace:", "Pace", []string{"Interactive", "Automatic"}},
		2: {"Axiom SDD preflight pr-strategy:", "PR strategy", []string{"Ask me", "Single PR", "Auto"}},
	}
	for index, want := range canonical {
		question := result.Canonicalized[index]
		if !strings.HasPrefix(question.Question, want.marker) {
			t.Errorf("question %d = %q, want prefix %q", index+1, question.Question, want.marker)
		}
		if question.Header != want.header {
			t.Errorf("question %d header = %q, want %q", index+1, question.Header, want.header)
		}
		if question.Multiple {
			t.Errorf("question %d multiple = true, want false", index+1)
		}
		if len(question.Options) != len(want.labels) {
			t.Fatalf("question %d has %d options, want %d", index+1, len(question.Options), len(want.labels))
		}
		for j, option := range question.Options {
			if option.Label != want.labels[j] {
				t.Errorf("question %d option %d label = %q, want %q", index+1, j+1, option.Label, want.labels[j])
			}
			if option.Description == "" {
				t.Errorf("question %d option %d has no description", index+1, j+1)
			}
		}
	}
	if untouched := result.Canonicalized[1]; untouched.Question != "¿Qué artefactos usamos?" || untouched.Header != "" || len(untouched.Options) != 3 || untouched.Options[0].Label != "a" {
		t.Errorf("an unmarked question was assigned to a group by position: %+v", untouched)
	}

	// Scenario 2: tolerant answers are accepted, bound by label, and the
	// confirmed block is prepended to the next SDD task dispatch with the
	// artifact store fixed to hybrid.
	for _, want := range []string{"## SDD Session Preflight", "Pace: auto", "Artifact store: hybrid", "Delivery strategy: ask-on-risk"} {
		if !strings.Contains(result.DispatchedPrompt, want) {
			t.Errorf("dispatched prompt missing %q; got:\n%s", want, result.DispatchedPrompt)
		}
	}

	// Scenario 3: an empty answer (a skipped tab) throws, naming the
	// question tool as the way to retry.
	if result.EmptyAnswerError == nil {
		t.Fatal("expected an error for an empty preflight answer")
	} else if !strings.Contains(*result.EmptyAnswerError, "question tool") {
		t.Errorf("empty answer error = %q, want it to mention the question tool", *result.EmptyAnswerError)
	}

	// Scenario 4: a question set without the pr-strategy group throws in the
	// before-hook and names the missing group.
	if result.MissingGroupError == nil {
		t.Fatal("expected an error for a preflight without the pr-strategy group")
	} else if !strings.Contains(*result.MissingGroupError, "pr-strategy") {
		t.Errorf("missing group error = %q, want it to name the pr-strategy group", *result.MissingGroupError)
	}

	// Scenario 5: a supplied description stays attached to the label it was
	// written for, so a swapped option order never shows a canonical label
	// next to another option's description; labels that match no canonical
	// option fall back to the built-in descriptions; the legacy Artifacts
	// question is ignored and left as the model wrote it.
	if len(result.Reordered) != 3 {
		t.Fatalf("expected 3 reordered questions, got %d", len(result.Reordered))
	}
	wantReordered := [][2]string{{"Interactive", "DESC-INTERACTIVE"}, {"Automatic", "DESC-AUTO"}}
	for j, want := range wantReordered {
		option := result.Reordered[0].Options[j]
		if option.Label != want[0] || option.Description != want[1] {
			t.Errorf("reordered pace option %d = %q/%q, want %q/%q", j+1, option.Label, option.Description, want[0], want[1])
		}
	}
	for j, option := range result.Reordered[1].Options {
		if want := []string{"Ambos", "Engram", "OpenSpec"}[j]; option.Label != want || !strings.HasPrefix(option.Description, "DESC-") {
			t.Errorf("legacy artifacts option %d = %q/%q, want it untouched (%q)", j+1, option.Label, option.Description, want)
		}
	}
	wantPR := []string{"Ask me", "Single PR", "Auto"}
	for j, option := range result.Reordered[2].Options {
		if option.Label != wantPR[j] {
			t.Errorf("localized pr strategy option %d label = %q, want %q", j+1, option.Label, wantPR[j])
		}
		if strings.HasPrefix(option.Description, "DESC-") {
			t.Errorf("localized pr strategy option %d kept a positional description %q; want the built-in default", j+1, option.Description)
		}
	}

	// Scenario 6: the Axiom-branded prefix with nominal markers is accepted
	// and emitted, and the authority is recorded so the confirmed block
	// reaches the dispatch.
	if len(result.AxiomCanonicalized) != 2 {
		t.Fatalf("expected 2 canonicalized Axiom-prefixed questions, got %d", len(result.AxiomCanonicalized))
	}
	for i, want := range []string{"Axiom SDD preflight pace:", "Axiom SDD preflight pr-strategy:"} {
		if question := result.AxiomCanonicalized[i]; !strings.HasPrefix(question.Question, want) {
			t.Errorf("Axiom-prefixed question %d = %q, want prefix %q", i+1, question.Question, want)
		}
	}
	for _, want := range []string{"## SDD Session Preflight", "Pace: auto", "Artifact store: hybrid", "Delivery strategy: single-pr"} {
		if !strings.Contains(result.AxiomDispatchedPrompt, want) {
			t.Errorf("Axiom-prefixed dispatched prompt missing %q; got:\n%s", want, result.AxiomDispatchedPrompt)
		}
	}
}
