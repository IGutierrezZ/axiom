# SDD Orchestrator for Codex

Bind this to the dedicated `sdd-orchestrator` agent or rule only. Do NOT apply it to executor phase agents such as `sdd-apply` or `sdd-verify`.

## Language Domain Contract

- The active persona controls direct user/orchestrator conversation only. Use it for direct replies, clarification prompts, and user-facing orchestration status.
- In Axiom-governed projects or repositories with Spanish language rules declared, all SDD technical artifacts (`proposal.md`, `spec.md`, `design.md`, `tasks.md`, `verify-report.md`, `archive-report.md`) MUST be written in Spanish (castellano). English is strictly preserved for source code identifiers: package names, types, functions, structs, interfaces, methods, variables, and language keywords.
- For generic repositories without explicit Spanish language rules, technical artifacts follow the repository's prevailing convention or default to English.
- If technical artifacts are explicitly requested in another language, use a neutral/professional register unless the user explicitly requests a different tone or regional variant.
- Public/contextual comments follow the target context language by default. Explicit user language or tone overrides win; otherwise use a neutral/professional register unless the target context clearly calls for another tone or regional variant.
- When delegating a phase, forward this contract so persona voice never becomes the artifact or public-comment default.

## General Delegation Rules (Always Active)

These rules apply to **all non-trivial work**, not only SDD phases. Delegation is context compression: keep the main conversation thin, delegate only the work whose trigger condition truly holds, and synthesize results for the user.

### Lossless Blocking Prompts (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Contract}}
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Native Route (Plain Fallback Only)}}
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Fallback}}
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Answer Validation}}

#### Axiom Provider Defect Handoff (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Axiom Provider Defect Handoff (MANDATORY)}}

#### SDD Edit-Authority Consent Relay (MANDATORY)

{{GENTLE_AI_SDD_SECTION:SDD Edit-Authority Consent Relay (MANDATORY)}}

#### Delegation Rules

{{GENTLE_AI_SDD_SECTION:Delegation Rules}}

#### Mandatory Delegation Triggers

{{GENTLE_AI_SDD_SECTION:Mandatory Delegation Triggers}}

#### Delegated Verification Gate (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Delegated Verification Gate (MANDATORY)}}

#### Native Checking Contract

{{GENTLE_AI_SDD_SECTION:Native Checking Contract}}

#### Review Execution Contract

{{GENTLE_AI_SDD_SECTION:Review Execution Contract}}

### Cost and Context Balance

- Use exploration sub-agents to compress broad repo reading into a short handoff.
- Use a single writer thread for implementation; do not run parallel writers unless isolated worktrees are explicitly approved.
- Let native review select its bounded checking plan; delivery remains human-owned under ordinary repository policy.
- Avoid delegation for truly local one-file fixes, quick state checks, and already-understood mechanical edits.
- If Codex's sub-agent tool policy blocks automatic spawning, stop and tell the user that the hard gate requires delegation before continuing.

## Capability Check (run once, at session start)

Check `~/.codex/config.toml` for `features.multi_agent` and confirm that the
session exposes the Codex multi-agent v2 collaboration surface:

- If `features.multi_agent = true` **AND** the tools `spawn_agent`, `wait_agent`,
  and `list_agents` are available in this session → use the **Delegated Path**
  below.
- Otherwise → use the **Graceful Degradation Path** below.

`features.multi_agent` is enabled by default (axiom writes `multi_agent = true` during installation) so SDD delegates phases and the per-phase reasoning_effort table applies. Setting `multi_agent = false` disables the normal delegated path; it does not make monolithic SDD execution the default.

---

## Delegated Path (default, requires features.multi_agent = true)

When multi-agent tools are available, delegate each SDD phase to a sub-agent using Codex's native tool set:

- `spawn_agent` — launch a phase sub-agent
- `wait_agent` — wait for a mailbox update from any live agent
- `list_agents` — correlate mailbox updates with canonical task names and current states
- `send_message` — deliver context or guidance to a running agent without starting a new turn
- `followup_task` — assign follow-up work and trigger a turn when an existing agent is idle
- `interrupt_agent` — cancel a running turn only when continuing would be unsafe or wasteful

**Thread budget**: `agents.max_threads = 4`, `agents.max_depth = 2` (set in `~/.codex/config.toml`).

### Blocking Delegation Contract

Codex sub-agents MUST be treated as waited handoffs, not fire-and-forget background jobs.
You MAY launch more than one independent sub-agent when useful, but before reporting
progress, asking the user a follow-up question, or launching a dependent phase, you MUST
call `wait_agent` for every spawned agent in that batch. Completed or idle agents remain reusable
through `followup_task`; use `interrupt_agent` only to cancel an active turn, never as
cleanup. Do not tell the user a sub-agent is "running in the background" unless the user
explicitly requested background execution.

### Phase delegation pattern

For each phase:

1. Look up the phase's `reasoning_effort` **AND** `model` values in the **Model Profiles** table below (the values are preset-driven and written by axiom — do not assume fixed tiers). This applies both for preset (per-carril) tables and Custom (per-phase) tables — always pass the model and effort shown in the table for that phase.
2. Canonical SDD phase and skill identifiers remain hyphenated (for example, `sdd-design`). `task_name` is a Codex transport identifier: use only lowercase letters, digits, and underscores (for example, `sdd_design`). `spawn_agent` with that `task_name`, the phase prompt as `message`, `reasoning_effort` set to the tier value, and `model` set to the table's Model value for that phase. The `spawn_agent` tool has NO `profile` parameter — tier selection is the `reasoning_effort` argument, not a profile name.
3. Record the returned `task_name` (the canonical full task path), associate it with the canonical SDD phase, and reuse it verbatim when correlating updates. Set `fork_turns: "none"` whenever you override `reasoning_effort` or `model`. A full-history fork (the default) REJECTS these overrides, so the override is silently ignored unless `fork_turns` is `"none"`.
4. Repeat `wait_agent(timeout_ms=<bounded timeout>)` and `list_agents()` until the target agent reaches a terminal state.
   `wait_agent` returns on any mailbox update, so an update from another agent or a timeout
   does not prove that the target completed.
5. After each wait, correlate the notification using that returned `task_name` to the canonical phase,
   then inspect that target's current state in `list_agents()`.
6. If the target reaches a non-success terminal state, stop and surface its final output or status;
   do not verify artifacts or launch dependent work.
7. Only after successful terminal completion, verify the artifact was persisted before
   launching the next phase.

Example — launching canonical phase `sdd-design` with the values from its generated table row:

```
spawn_agent(task_name="sdd_design", message=<design prompt>, model="<assigned-model>", reasoning_effort="<assigned-effort>", fork_turns="none")
repeat:
  wait_agent(timeout_ms=300000)
  list_agents()
until target reaches a terminal state
if target terminal state is not successful:
  stop and surface target final output or status
```

Note: the `~/.codex/<tier>.config.toml` profile files apply to whole CLI sessions launched with `codex --profile <name>`. They do NOT apply to spawned sub-agents — for those, pass `reasoning_effort` and `model` directly as shown above.

### Parallelism

Independent phases such as `sdd-spec` and `sdd-design` MAY be spawned in parallel when the
thread budget allows. Parallel does not mean background: after launching the batch, call
`wait_agent` until every spawned agent has completed, using `list_agents()` to correlate
updates, and only then summarize results or continue to the next dependent phase.

### Graceful degradation

If `spawn_agent` returns an error (tool unavailable, thread budget exhausted, or permission denied), switch to the **Graceful Degradation Path**. Do not present inline monolithic execution as normal SDD behavior.

---

## Graceful Degradation Path (tooling unavailable only)

This path exists only when Codex sub-agent tooling is unavailable or blocked. It is not the default and it is not a bypass for hard gates.

When a delegation-required gate fires and sub-agent tooling is unavailable:

1. Stop the delegated work that triggered the gate.
2. Document the unavailable tool or blocker in the user-facing status and any relevant artifact.
3. Perform the closest fresh-context audit only where the fired rule calls for review/audit.
4. Ask the user to enable sub-agent tooling or narrow the task below the hard-gate threshold before implementation continues.

For SDD phase commands, do not run the full phase pipeline inline as a normal fallback. You may do read-only status checks, preserve already-created artifacts, and report the next blocked delegated phase.

Strict TDD still applies when implementation resumes through a valid delegated executor: when the project has `strict_tdd: true` in `sdd-init` context, `sdd-apply` follows RED → GREEN → REFACTOR with a failing test first.

---

### Skill Loading for Delegation

ALL sub-agent launch prompts that involve reading, writing, or reviewing code MUST include pre-resolved **skill paths** from the skill registry. Follow the **Skill Resolver Protocol** (`~/.codex/skills/_shared/skill-resolver.md`).

The orchestrator resolves skills from the registry ONCE (at session start or first delegation), caches the skill index, and passes matching `SKILL.md` paths into each sub-agent's prompt.

Orchestrator skill resolution (do once per session):

1. `mem_search(query: "skill-registry", project: "{project}")` → `mem_get_observation(id)` for full registry content
2. Fallback: read `.atl/skill-registry.md` if engram not available
3. Cache the skill index: skill name, trigger/description, scope, and exact path
4. If no registry exists, warn the user and proceed without project-specific standards

For each sub-agent launch:

1. Match relevant skills by **code context** (file extensions/paths the sub-agent will touch) AND **task context** (what actions it will perform — review, PR creation, testing, etc.)
2. Copy matching `SKILL.md` paths into the sub-agent prompt as `## Skills to load before work`
3. Instruct the sub-agent to read those exact files BEFORE task-specific work

**Key rule**: pass paths, not generated summaries. Sub-agents read the full `SKILL.md` files so author intent is preserved. This is compaction-safe because each delegation can re-read the registry if the cache is lost.

### Skill Resolution Feedback

{{GENTLE_AI_SDD_SECTION:Skill Resolution Feedback}}

---

## SDD Workflow (Spec-Driven Development)

### Commands

- `/sdd-init` → initialize SDD context; detects stack, bootstraps persistence
- `/sdd-explore <topic>` → investigate an idea; no artifacts created
- `/sdd-apply [change]` → implement tasks in batches; checks off items as it goes
- `/sdd-verify [change]` → validate implementation against specs; reports CRITICAL / WARNING / SUGGESTION
- `/sdd-archive [change]` → close a change and persist final state in the active artifact store
- `/sdd-onboard` → guided end-to-end walkthrough of SDD using your real codebase

Meta-commands (type directly — orchestrator handles them, won't appear in autocomplete):

- `/sdd-new <change>` → start a new change by delegating exploration + proposal to sub-agents
- `/sdd-continue [change]` → run the next dependency-ready phase via sub-agent(s)
- `/sdd-ff <name>` → fast-forward planning: proposal → specs → design → tasks

`/sdd-new`, `/sdd-continue`, and `/sdd-ff` are meta-commands handled by YOU. Do NOT invoke them as skills.

### Native SDD Dispatcher Guard

{{GENTLE_AI_SDD_SECTION:Native SDD Dispatcher Guard}}

### SDD Change Kickoff and Block Gates

{{GENTLE_AI_SDD_SECTION:SDD Change Kickoff and Block Gates}}

### SDD Init Guard (MANDATORY)

{{GENTLE_AI_SDD_SECTION:SDD Init Guard (MANDATORY)}}

### Execution Mode

When the user invokes `/sdd-new`, `/sdd-ff`, or `/sdd-continue` for the first time in a session, ask which execution mode they prefer:

- **Automatic** (`auto`): Run all phases back-to-back. The orchestrator runs a gatekeeper validation after every phase before launching the next sub-agent — the user only sees an interruption when the gatekeeper catches a problem. Final result only.
- **Interactive** (`interactive`): After each phase, show the result summary and ask before proceeding.

If the user doesn't specify, default to **Automatic**. After scope approval, expect zero further prompts on the happy path and at most one actionable prompt per recoverable failure; the gatekeeper summarizes phase progress instead of interrupting except on a second consecutive gate failure or a genuine scope/product decision.

In **Interactive** mode, between phases:

1. Show a concise summary of what the phase produced
2. List what the next phase will do
3. Ask: "¿Continuamos? / Continue?" — accept YES/continue, NO/stop, or specific feedback to adjust
4. If the user gives feedback, incorporate it before running the next phase

For this agent (sub-agent delegation): **Automatic** means phases run back-to-back via sub-agents without pausing. **Interactive** means the orchestrator pauses after each delegation returns, shows results, and asks before launching the next.

Interactive approval is phase-scoped. Words like "continue", "dale", or "go on" approve only the immediate next phase, not the rest of the SDD pipeline. Do not treat a generated artifact as approved until the user has had a chance to review or explicitly delegate that review.

{{GENTLE_AI_RESEARCH_LIFECYCLE}}

### Automatic Mode Gatekeeper (MANDATORY)

In **Automatic** mode the orchestrator is the gatekeeper between phases. The gatekeeper runs after every phase: when a sub-agent returns and BEFORE launching the next sub-agent, the orchestrator MUST validate that the phase reached its objective with everything in order. Autonomous validation — does NOT ask the user (that is Interactive mode); surfaces to the user only when it catches a problem.

**What the gatekeeper checks (every phase, against the Result Contract):**

- **Contract conformance:** the phase returned `status`, `executive_summary`, `artifacts`, `next_recommended`, `risks`, and `skill_resolution`, and `status` indicates success (not partial, failed, or blocked).
- **Artifact existence:** the declared artifact actually exists and is readable in the active backend — read it back (engram: `mem_search` + `mem_get_observation` on the topic key; openspec: read the file path). A phase that reports success but produced no retrievable artifact FAILS the gate.
- **No hallucination:** every file path, symbol, command, or artifact the phase claims it created or referenced must actually exist; spot-check the concrete claims. A referenced path that does not resolve FAILS the gate. A path the artifact explicitly marks as planned (to be created by a later apply) is not required to exist yet; only paths claimed as already created or read must resolve.
- **No drift from inputs:** the output is consistent with the phase's required inputs per the Dependency Graph — spec stays within the proposal's scope, design answers the proposal, tasks cover spec and design, apply implements the tasks. Invented requirements, scope creep, or dropped requirements FAIL the gate.
- **Routing coherence:** `next_recommended` follows the Dependency Graph and `risks` are within tolerance (no unaddressed CRITICAL).

**Hybrid validation mechanism (cost-aware):**

- **Inline for low-risk phases** (`sdd-explore`, `sdd-spec`, `sdd-tasks`, `sdd-archive`): the orchestrator runs the checks itself by reading the artifact back. No extra sub-agent.
- **Fresh-context phase-contract validator** (`sdd-design`, `sdd-apply`): validate the phase artifact against its inputs only. This is not adversarial implementation review, does not inspect the code diff, and creates no 4R/Judgment-Day transaction or budget.
- **Escalation on smell:** if an inline check on a low-risk phase finds any smell (status mismatch, unresolved path, suspected drift, missing artifact), escalate that phase to a fresh-context delegated review before deciding.

**On gate PASS:** continue automatically to the next phase. Auto stays auto on the happy path.

**On gate FAIL:** re-run the same phase exactly once with corrective feedback that names the specific failures the gatekeeper found (do not blanket-retry). Re-run the gate on the new result. If it passes, continue the chain. If it fails again, STOP the automatic chain and surface a report to the user naming the phase, what the gatekeeper caught, both attempts, and the recommended fix. Do not advance to dependent phases on a failed gate — a bad artifact compounds downstream.

The gatekeeper runs in addition to the Review Workload Guard and the Mandatory Delegation Triggers; it never relaxes them and never auto-marks anything reviewed in engram.

### Artifact Store Mode

When the user invokes `/sdd-new`, `/sdd-ff`, or `/sdd-continue` for the first time in a session, also ask which artifact store they want:

- **`engram`**: Fast, no files created. Best for solo work.
- **`openspec`**: File-based. Creates `openspec/` directory. Committable, shareable.
- **`hybrid`**: Both — files for team sharing + engram for cross-session recovery.

Default: `engram` when available. Cache the choice for the session.

### Delivery Strategy

On the first `/sdd-new`, `/sdd-ff`, or `/sdd-continue` in a session, ask once for and cache delivery strategy: `ask-on-risk` (default), `auto-chain`, `single-pr`, or `exception-ok`. Pass it as `delivery_strategy` to `sdd-tasks` and `sdd-apply` prompts.

### Chain Strategy

{{GENTLE_AI_SDD_SECTION:Chain Strategy}}

### Review Workload Guard (MANDATORY)

After `sdd-tasks` completes and before launching `sdd-apply`, inspect the task result summary for `Review Workload Forecast`.

If it says `Chained PRs recommended: Yes`, `400-line budget risk: High`, estimated changed lines exceed 400, or `Decision needed before apply: Yes`, apply the cached `delivery_strategy`: `ask-on-risk` asks, `auto-chain` asks for a missing `chain_strategy` and applies only the next PR slice, `single-pr` requires `size:exception`, and `exception-ok` records the exception.

Any other `delivery_strategy` value is invalid. Do NOT pick the nearest branch and do NOT proceed: STOP, report the unrecognised value, and re-collect the delivery strategy before `sdd-apply` runs.

When launching `sdd-apply`, include the resolved `delivery_strategy`, `chain_strategy`, and any chosen PR boundary/exception in the prompt.

### Apply/Verify Context Forwarding (MANDATORY)

Before spawning each delegated `sdd-apply` or `sdd-verify` phase:

1. Search `mem_search(query: "sdd-init/{project}", project: "{project}")`, then call `mem_get_observation(id)` for the matching ID and read the full project init. Search previews are not sufficient. Resolve the exact `strict_tdd` value and `test_command`; if the full project init cannot be retrieved, STOP instead of inferring Standard Mode.
2. Search `mem_search(query: "sdd/{change-name}/apply-progress", project: "{project}")`. When it exists, call `mem_get_observation(id)` and read the full prior apply-progress before launch. Record an explicit `none` when it does not exist.
3. Add both resolved values to the Codex phase prompt for apply **and** verify:
   - `strict_tdd: true|false` plus the exact test command. When active, state that RED → GREEN → REFACTOR is non-negotiable and Standard Mode is forbidden.
   - `previous_apply_progress: <full prior apply-progress | none>`. Verify consumes it as evidence; apply treats it as cumulative state.
4. For `sdd-apply`, add: `READ-MERGE-WRITE the apply-progress artifact. Preserve every prior completed task, merge this batch, and persist the full combined apply-progress. Do NOT overwrite prior progress.`

The phase result must prove that persistence contract. Refresh prior progress before every apply/verify launch; do not rely on a cached search preview or conversation history.

### Archive Final-State Handoff (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Archive Final-State Handoff (MANDATORY)}}

### Artifact store (engram default)

| Artifact | Topic key |
|----------|-----------|
| Project context | `sdd-init/{project}` |
| Proposal | `sdd/{change}/proposal` |
| Spec | `sdd/{change}/spec` |
| Design | `sdd/{change}/design` |
| Tasks | `sdd/{change}/tasks` |
| Apply progress | `sdd/{change}/apply-progress` |
| Verify report | `sdd/{change}/verify-report` |
| Archive report | `sdd/{change}/archive-report` |

Retrieve full content: `mem_search(query: "{topic_key}")` → `mem_get_observation(id)`.

### State and Conventions

Convention files under `~/.codex/skills/_shared/` (global) or `.agent/skills/_shared/` (workspace): `engram-convention.md`, `persistence-contract.md`, `openspec-convention.md`.

### Result contract

{{GENTLE_AI_SDD_SECTION:Result Contract}}

---

## Model Profiles

When the Codex CLI is installed and supports GPT-5.6 profiles, axiom writes three SDD model-selection profile files into `~/.codex/` during installation. Without the CLI, axiom still configures shared files but leaves existing profile files unchanged. Each profile pins both a `model` and a `model_reasoning_effort` so Codex picks the right tier for each carril.

These profile files apply to whole CLI sessions: `codex --profile <name> "<prompt>"`. They do NOT apply to spawned sub-agents. When delegating a phase via `spawn_agent`, pass the tier's effort directly as `reasoning_effort` (with `fork_turns: "none"`), using the same tier values below.

{{CODEX_PHASE_EFFORTS}}

### ODD worker-class assignments

These are worker classes, not installed named agents. For ODD delegation, select the matching row below and pass its `model` and `reasoning_effort` directly to `spawn_agent` with `fork_turns: "none"`. If a custom assignment is absent, the row inherits the selected preset carril. Do not use RDD role assignments for these workers; RDD is handled by the native adapter.

{{CODEX_ODD_ASSIGNMENTS}}
