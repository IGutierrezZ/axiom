// Package agentguidance owns the always-installed organic routing guidance
// projected into every configured agent. Routing is unconditional: it must not
// depend on the optional SDD component, on SDD mode, or on any SDD asset being
// present, because an agent without routing guidance has no way to choose
// between direct, delegated, and proposed work.
package agentguidance

import (
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/capabilitymanifest"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// ErrUnknownRoutingPolicy fails closed when the canonical manifest carries a
// selection policy this renderer was not written for. Rendering stale prose
// over a changed policy would silently misinform every installed agent.
var ErrUnknownRoutingPolicy = errors.New("unknown sdd selection policy")

// checkRoutingPolicy admits only the current selection policy (REQ-27.16): the
// retired `explicit_request_or_accepted_proposal` literal and any other value
// fail closed with ErrUnknownRoutingPolicy.
func checkRoutingPolicy(policy string) error {
	if capabilitymanifest.SDDSelectionPolicy(policy) != capabilitymanifest.SDDSelectionExplicitRequestProjectLaneOrAcceptedProposal {
		return fmt.Errorf("%w: %q", ErrUnknownRoutingPolicy, policy)
	}
	return nil
}

// RenderRouting projects the canonical implementation-routing facts of one
// supported adapter into its guidance block.
//
// The rendered block deliberately carries routing semantics only. It states no
// runtime observation, issues no lifecycle authority, and activates no remote
// mechanism, so an offline agent reading it can still route work correctly.
func RenderRouting(agent model.AgentID) (string, error) {
	manifest, err := capabilitymanifest.ForAgent(agent)
	if err != nil {
		return "", fmt.Errorf("render routing guidance for %q: %w", agent, err)
	}

	routing := manifest.ImplementationRouting
	if err := checkRoutingPolicy(string(routing.SDD.SelectionPolicy)); err != nil {
		return "", err
	}

	var output strings.Builder
	output.WriteString("## Implementation Routing\n\n")

	// ODD protocol: absorbed from upstream (1b202d77, 70c774f8, cfc415ce,
	// dcd2fa07 — INC-20 F6.1). Organic Driven Development becomes the
	// orchestrator's predefined, mandatory-by-default workflow; SDD remains a
	// branch inside it, selected only by explicit request, by the project lane or an accepted
	// proposal, never a substitute, predecessor, or postponement of this
	// projection (organic-agent-trigger-rules, ADDED requirement).
	output.WriteString("Organic Driven Development (ODD) is the predefined workflow of this orchestrator. Every request enters it, on every runtime, without the user asking for a workflow, a plan, or task tracking. SDD is a branch inside ODD, entered only by an explicit request, by the project lane declared in `axiom.yaml` (including an ISO project, `iso.project: iso`), or by an accepted proposal. Never describe this workflow only when asked about it: run it.\n\n")
	output.WriteString("### ODD protocol (MANDATORY, in this order, on every request)\n\n")
	// Step 0: absorbed from INC-21 (design.md S1.2/S4.7, H-2; REQ-21.1-21.3).
	// It stays numbered zero, ahead of the seven-step protocol below, instead
	// of renumbering steps 1-7: those numbers are already cross-referenced
	// verbatim elsewhere in this doctrine and in SDD's own shared sections.
	// The lane comes from the `workflow` field of `axiom sdd status` (REQ-27.16):
	// the binary projects it and fails closed toward `ask`, so this identical text
	// serves every project. ODD has no stop of its own (REQ-27.17): the plan stop
	// and the three-exit section belong to the SDD gates only.
	output.WriteString("0. **Evaluate scope and lane.** Before creating any work artifact — `odd/tasks/<feature-name>.md` or `openspec/changes/<change>/proposal.md` — first read `axiom sdd status --cwd <repo> --json`. When its `iso` field projects `project: iso`, the project is ISO and only uses SDD: all work, including small, understood work, goes through the SDD lane without asking the lane question, also when `workflow.lane` is `ask`, and no `odd/tasks/<feature-name>.md` is created. A bug in an ISO project always goes through SDD with the bug profile, along one of three routes: when it restores a requirement that already exists, no delta and a `Cubre:` line citing that requirement; when it touches no requirement, no delta and a `## Impacto en la spec` section holding the line `Sin impacto en la spec` and a `Motivo:` line; when the requirement was wrongly specified or missing, a delta, as a normal change. The first two routes also carry a `## Informe de bug` section with `Observado:` and `Esperado:` lines in `proposal.md`. Otherwise, read the project lane from its `workflow` field, which carries `lane` (`odd`, `sdd` or `ask`) and `source`, with or without a selected change. Small, understood work stays small in every lane and creates no work artifact, except in an ISO project (above). For substantial work: with `lane: sdd`, enter the SDD pre-flight questionnaire directly, without asking the lane question; with `lane: odd`, work through ODD without asking, even for an architectural scope, unless the user explicitly asks for SDD in this request; with `lane: ask`, when `source` is `no_workspace` or `invalid`, or when the status cannot be read, ask the blocking lane-selection question — do you want to approach this through the agile ODD lane or the formal SDD lane? — and STOP to wait for the answer, creating no work artifact while it is pending; under `ask`, a scope that is unambiguously architectural or large skips that binary question and enters the SDD pre-flight questionnaire directly. Whenever the work goes through ODD, do not ask any further governance question: proceed straight into step 1 below, work it through without dividing it into roles, and keep it exclusively in `odd/tasks/<feature-name>.md`.\n")
	output.WriteString("1. **Authorize.** First establish whether the requested outcome explicitly authorizes a change. Investigation, explanation, review, audit, comparison, and solution-proposal or planning-only requests are read-only unless the user explicitly requests implementation or another mutation.\n")
	output.WriteString("   - Read-only work may inspect, explain, compare, and recommend, but must not write or edit files, delegate a writer, invoke apply, or create implementation artifacts.\n")
	output.WriteString("   - If change intent is ambiguous or conditional, ask one clarification and remain read-only until answered.\n")
	output.WriteString("2. **Explore.** Explore the existing code and requirements first, proportionately to the request, before proposing or writing anything. When the project declares a maintained technical context, read its entry before proposing or writing: get the locators from `axiom workspace context --cwd <repo> --json`, open `technicalContext.maintained.entry` first and consult the remaining `technicalContext.maintained.paths` only on demand; without a declared maintained context this step reads nothing more.\n")
	output.WriteString("3. **Resolve uncertainty.** Recommend optional research only for a named uncertainty; ask one focused user question only for a real unresolved product decision, then stop and wait; use at most one scoped read-only assumption challenge for a high-consequence unproven premise.\n")
	output.WriteString("4. **Classify.** The work is substantial when exploration yields two or more meaningful implementation steps, or progress worth recovering after an interruption. Small, understood work stays small and creates no durable task artifacts.\n")
	output.WriteString("5. **Track before the first write.** For substantial authorized implementation, create `odd/tasks/<feature-name>.md` and its Engram mirror `odd/<feature-name>/tasks` automatically, before the first source write, without asking permission for tasks or storage. Tell the user in one line which feature document was created and how many tasks it holds.\n")
	output.WriteString("6. **Implement task by task.** Route each task through the smallest useful topology below, honoring its delegation triggers when their conditions truly hold, with the configured TDD mode and applicable checks. Check an item off only after its outcome and checks were observed; update the file and the mirror after each task. Every task closes with at least one work-unit commit on the branch that `axiom branch ensure --lane odd --change <feature-name>` prepared before the first write of `odd/tasks/<feature-name>.md` (see *Branches and worktrees*), with tests and docs alongside the behavior, using a Conventional Commit message; record the commit identity in the feature document as evidence. Work-unit commits on the feature branch are part of authorized substantial ODD implementation; push, pull request creation, and merge remain the user's decisions under ordinary repository policy. After each work-unit commit, tell the person in one line the short sha, the task ID and `git show <sha>`, then continue with the next task without waiting.\n")
	output.WriteString("7. **Close.** Report the verified outcome, every failed, skipped, or pending check, and the next step. When the project declares a maintained technical context, a work unit that changes something that context documents updates the affected documentation in that same work-unit commit; documentation declared as reference (`technical_context.reference`) is never modified, and without a maintained context this adds no requirement. The locators of that maintained context are the ones read in step 2, from the output of `axiom workspace context --cwd <repo> --json`. When the work changes or must review the living specification, write the OpenSpec delta in the `## Spec viva` section of `odd/tasks/<feature>.md` (one `### Capacidad: <domain>` per capability, with its delta inside one fenced code block labelled `openspec`); every bug (marked with the line `Tipo: bug`) provides a non-empty `## Spec viva` or a `## Impacto en la spec` section with the exact line `Sin impacto en la spec` plus a non-empty `Motivo: <one line>` line (the marker is ignored when a non-empty `## Spec viva` is present, and a bug needs no `## Informe de bug`); every new requirement in an ADDED section carries a provisional identifier instead of a number picked by hand; run `axiom archive integrate --odd <feature>` in the work-unit commit that closes the feature, without stopping: it assigns the number of each provisional identifier and rewrites it in the ODD document, and that integration into `openspec/specs/` is the only write allowed outside the ODD document and the code.\n\n")
	output.WriteString("Resume an interrupted feature with `mem_context`, then project- and feature-scoped `mem_search`, then `mem_get_observation` for the full document, then the task file itself; reconcile before continuing the next unfinished task. When the Engram mirror does not exist, start from the task file alone, never fail for the missing mirror, and continue with the first unfinished task.\n\n")
	output.WriteString("After explicit change intent is established, route work for the requested outcome with the smallest useful topology. Every authorized change takes exactly one implementation route: direct inline, delegated direct, or optional SDD.\n\n")

	_, _ = fmt.Fprintf(
		&output,
		"- **Direct inline:** decide or verify within the evidence budget: one parallel batch of at most %d calls and approximately %dk tokens, using bounded searches or line ranges. A unit of about %d changed lines or less is always written inline, and a mechanical, already-understood change stays inline when it needs no research and has no unresolved design decision. The number of files never decides.\n",
		routing.DirectInline.MaxEvidenceCalls,
		routing.DirectInline.ApproxEvidenceTokens/1000,
		routing.DirectInline.MaxInlineUnitLines,
	)
	_, _ = fmt.Fprintf(
		&output,
		"- **Delegated direct:** understanding that exceeds the evidence budget takes one read-only explorer that returns a handoff of at most approximately %dk tokens; a writer is delegated only for a named reason (parallelism or protecting the parent context), never by file count. Reading that prepares a delegated write and broad research also delegate.\n",
		routing.DelegatedDirect.ApproxHandoffTokens/1000,
	)
	output.WriteString("- **Optional SDD:** propose SDD only when durable proposal, spec, design, and tasks would materially reduce substantial ambiguity. SDD is selected only by an explicit request, by the project lane `workflow.lane: sdd` for substantial work, by an ISO project (`iso.project: iso`) for all work, or by an accepted proposal.\n")
	output.WriteString("- File count, changed lines, size, or perceived risk alone never selects SDD and never forces a heavier route.\n")
	output.WriteString("- Automatic SDD pace is not mutation authorization; once implementation is explicitly authorized, it continues under the selected route.\n")
	output.WriteString("- These are implementation routes, not a ban on per-action delegation. Tests, builds, and installs may still use fresh workers without changing the selected route.\n")
	output.WriteString("- Direct and delegated work never create SDD artifacts, prompts, phase attempts, or synthetic SDD runs.\n")

	// Delegation triggers: conditional, rendered from the single source in
	// delegation.go and shared with the SDD orchestrators. Each trigger is
	// mandatory only while its condition truly holds; when in doubt, inline. They
	// follow the route descriptions so they compete with the permissive "smallest
	// useful topology" framing instead of being buried pages away from it.
	output.WriteString("\n### Mandatory Delegation Triggers\n\n")
	output.WriteString(renderDelegationTriggers(routing))
	output.WriteString("\n")

	// Organic Driven Development detail: absorbed from upstream 1b202d77,
	// amended by cfc415ce (INC-20 F6.1). This is the reference detail for the
	// mandatory protocol above — feature continuity, configured TDD, and
	// and work-unit commits, all previously undocumented in
	// this fork's routing guidance (zero ODD mentions before this task).
	output.WriteString("\n### Organic Driven Development\n\n")
	output.WriteString("Use this flow for direct and delegated organic work, not explicitly selected SDD. Explore the existing code and requirements first, proportionately to the request; the mutation-authorization guard above still applies.\n\n")
	output.WriteString("This section is the reference detail for the protocol above.\n\n")
	output.WriteString("- Recommend optional research only for a named uncertainty. If declined, continue within authorized scope only where safe without the missing evidence; disclose unresolved uncertainty and pause affected unsafe decisions. Offer a concise proposal only when a real scope or product decision needs it. Neither research nor a proposal is mandatory.\n")
	output.WriteString("- Establish the problem, intended outcome, constraints, and current evidence; inspect relevant code. Adapt depth to uncertainty and consequence, not a fixed questionnaire or mandatory rounds. The parent owns product decisions: ask one focused user question only for a real unresolved product decision, then stop and wait. Workers return gaps to the parent rather than assuming choices.\n")
	output.WriteString("- When the question needs external evidence, use available authorized documentation/web tools and prefer primary sources. Attribute material claims to source URLs or code locations; distinguish verified facts, assumptions, contradictions, freshness, and gaps. If tools are unavailable, disclose limitations without inventing access or evidence; pause only unsafe decisions dependent on missing evidence.\n")
	output.WriteString("- Return concise findings, recommendation, tradeoffs, open questions, and implementation implications. When delegating research, forward these research instructions to a fresh general exploration/research worker through existing delegation; do not create a specialized agent or invoke sdd-research. Keep research read-only; its findings do not authorize implementation or require new persistence or readiness machinery.\n")
	output.WriteString("- Use at most one scoped independent read-only assumption challenge for a high-consequence unproven premise, even in a small security-critical change. Name the premise, evidence, and consequence; do not start a debate loop. Deterministic failures need fixes, not model debate.\n")
	output.WriteString("- Small, understood work creates no durable task artifacts. Substantial means coordinated steps or progress worth recovering, not a line-count threshold. For substantial authorized implementation, automatically create the feature document after exploration, without a task or storage permission prompt.\n")
	output.WriteString("- Size each ODD task as the smallest coherent behavior with its tests and docs; there is no line-count target, cap, or forced split. Never delete spaces, blank lines, or comments for cosmetic line savings; never omit tests, minify, add gratuitous abstractions, or split artificially. Forward this same instruction when delegating tasks to subagents.\n")
	output.WriteString("- Keep `odd/tasks/<feature-name>.md` and an Engram recovery copy under topic `odd/<feature-name>/tasks`, scoped to the current project. Use a descriptive filename-safe feature name. Reuse the same feature identity; never overwrite another feature. Keep one feature document, not a separate plan file or topic: objective, problem, why, scope, constraints, actionable checklist with stable task IDs, authorized scope, acceptance criteria, and applicable checks. Include progress, verification evidence, and next step, plus concise rationale for meaningful accepted changes. Routine corrections stay with their tasks; no exhaustive decision journal. Mirror the full current document and repository-relative file locator, not just a summary or completion notice.\n")
	output.WriteString("- Accepted user, review, or verification changes automatically update affected intent and TODOs: preserve valid completed and unrelated work; add genuinely new tasks or reopen invalidated items with a reason, and revise their checks. Findings alone never authorize scope expansion or automatic acceptance. Business scope changes still require user authorization. Check off only observed outcomes with applicable proof; record failed, unavailable, skipped, or pending checks honestly. Checkboxes grant no approval or receipt.\n")
	output.WriteString("- Read back both writes; they are not atomic. If Engram is unavailable, preserve local progress and explicitly mark the mirror pending; do not claim success or block unrelated safe work. Resynchronize when available. If a file write is unsafe or unavailable, preserve existing state and report the limitation. Preserve both versions on irreconcilable edits and ask only about the real conflict.\n")
	output.WriteString("- On resume, use `mem_context`, then `mem_search` scoped to the current project and feature, and `mem_get_observation` for the full saved document; read the actual task file. Do not infer active work from the newest global memory. Reconcile current requirements, code, and proof before resuming the next unfinished task; preserve pending mirrors and conflicting edits.\n")
	output.WriteString("- Before implementation or resume, the parent reads both the actual file and full observation, reconciles them, and passes the locator and relevant context; workers read the document before edits. Small work without a document still receives its authorized scope and checks.\n")
	output.WriteString("- Resolve effective TDD on/off from existing project/session configuration or explicit user choice; retain its source and exact test runner. Record resolved mode, source, and runner in the feature document when present. Tests or frameworks being present does not enable TDD. Forward mode, source, and runner on every implementation delegation; refresh on resume. When enabled, require observed RED before implementation, GREEN, then REFACTOR; never invent evidence. When disabled, run ordinary functional checks, not no checks. If mode is unknown/conflicting or the runner is missing, disclose and resolve only the ambiguity affecting the next action; never invent precedence or a command, and never invoke sdd-init to determine ODD TDD.\n")
	output.WriteString("- Preserve applicable functional verification. Run applicable functional checks per task, not a verification cycle per TODO checkbox.\n")
	output.WriteString("- Delivery follows work units. By default a feature ships as `single-pr`: one pull request holding one work-unit commit per task. At feature-document creation, forecast the approximate authored changed lines (additions plus deletions, generated files excluded) from the task list. When the project's `axiom.yaml` declares `delivery.split_hint_lines` and the forecast exceeds it, tell the person once for this feature that the change is large and that they can ask for chained pull requests, then continue without waiting; without that key, give no size warning and cite no line number. Use `chained` only when the person asks to chain, stack, or slice the pull requests: if they did not name a chain strategy, ask once whether `stacked-to-main` or `feature-branch-chain`, cache it, and record slice boundaries, which commits each pull request holds, in the feature document. Resolve the `work-unit-commits` skill by registry name for every feature and the `chained-pr` skill only when chaining, never hardcoding their paths.\n")
	output.WriteString("- Commits and the pull request. Asking for the pull request validates every commit of the feature: there is no commit-by-commit approval and nothing blocks it. Record in a `## Commits con cambios pedidos` section of `odd/tasks/<feature>.md` only the commits the person asked to change, one line each, never a checkbox. To change a commit already made, first finish the task in progress, or park it with `git stash push --include-untracked` and restore it afterwards; when no remote-tracking branch contains the commit (`git branch -r --contains <sha>`, never fetch to decide), use `git commit --fixup=<sha>` and squash the fixups with `git -c sequence.editor=true rebase -i --autosquash --autostash <base>` only when the pull request is requested; once it is published, add a new corrective commit and never rewrite published history. Recommend merging with a merge commit, without squash.\n")

	// Branches and worktrees (inc-25, REQ-25.40, REQ-25.41, design D-19): the
	// single point of the ODD doctrine that prepares the branch. It is rendered
	// identically for every adapter and it holds the only line of the product
	// that bans the runtime-specific worktree primitives; no skill and no
	// per-runtime orchestrator may restate them. The `ask` mode adds no
	// governance question: the one interaction is the decision the command
	// itself returns.
	output.WriteString("\n### Branches and worktrees\n\n")
	output.WriteString("- Before the first write of `odd/tasks/<feature-name>.md` or of any code, run `axiom branch ensure --lane odd --change <feature-name> --json`. It prepares the branch and, when the effective mode asks for it, the worktree; repeating it when you resume is idempotent.\n")
	output.WriteString("- Use the absolute paths of its JSON as `--cwd` of every later `axiom` command, as the repository of every `git -C` command, and as the root of every file you write.\n")
	output.WriteString("- `decision_required` is the only interaction this rule can add: present the decision it returns, then repeat the same command with the answer; do not add a governance question of your own.\n")
	output.WriteString("- Never pass `--fetch` without the explicit authorization of the user.\n")
	output.WriteString("- Never use `EnterWorktree` or `ExitWorktree`; work in another worktree only through the absolute paths that `axiom branch ensure` returns, with `--cwd`, `git -C`, and absolute paths.\n")

	// Informed approval and continuity (upstream #5850, #5486): one source in
	// approval.go, rendered once per routing block.
	output.WriteString("\n")
	output.WriteString(InformedApprovalAndContinuity())
	output.WriteString("\n")

	return output.String(), nil
}
