# Axiom — Orchestrator Instructions

Bind this to the Claude Code orchestrator rule only. Do NOT apply it to executor phase agents such as `sdd-apply` or `sdd-verify`.

## Agent Teams Orchestrator

{{GENTLE_AI_SDD_SECTION:Agent Teams Orchestrator}}

### Lossless Blocking Prompts (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Contract}}
- Native route: The classified native question UI is `AskUserQuestion`. Use it only when it is available in the current interactive runtime and the complete choice envelope is exactly representable in one grouped interaction without truncation or reshaping. When the closed domain of a single-select envelope is representable as the classified native question UI, use it; otherwise fall through to the Fallback clause below.
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Fallback}} Native-tool-only wording elsewhere never disables this fallback.
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Answer Validation}}

#### Axiom Provider Defect Handoff (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Axiom Provider Defect Handoff (MANDATORY)}}

#### SDD Edit-Authority Consent Relay (MANDATORY)

{{GENTLE_AI_SDD_SECTION:SDD Edit-Authority Consent Relay (MANDATORY)}}


### Language Domain Contract

{{GENTLE_AI_SDD_SECTION:Language Domain Contract}}

### SDD Change Kickoff and Block Gates

{{GENTLE_AI_SDD_SECTION:SDD Change Kickoff and Block Gates}}

### Delegation Rules

{{GENTLE_AI_SDD_SECTION:Delegation Rules}}

Use Claude Code's native Agent/Task mechanism for delegated-direct work; reserve `sdd-*` agents for a selected SDD route. These results are not persisted by OpenCode's background-agent plugin, so summarize any needed handoff explicitly.

{{GENTLE_AI_SDD_SECTION:Delegation Rules Handoff}}

#### Mandatory Delegation Triggers

{{GENTLE_AI_SDD_SECTION:Mandatory Delegation Triggers}}

#### Delegated Verification Gate (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Delegated Verification Gate (MANDATORY)}}

#### Native Checking Contract

{{GENTLE_AI_SDD_SECTION:Native Checking Contract}}

#### Review Execution Contract

{{GENTLE_AI_SDD_SECTION:Review Execution Contract}}

#### Cost and Context Balance

- Use exploration sub-agents to compress broad repo reading into a short handoff.
- Use a single writer thread for implementation; do not run parallel writers unless isolated worktrees are explicitly approved.
- Let native review select its bounded checking plan; delivery remains human-owned under ordinary repository policy.
- Avoid delegation for truly local one-file fixes, quick state checks, and already-understood mechanical edits.

{{GENTLE_AI_RESEARCH_LIFECYCLE}}

<!-- gentle-ai:sdd-model-assignments -->
<!-- /gentle-ai:sdd-model-assignments -->

## SDD Workflow (lazy-loaded)

The detailed SDD procedure is intentionally NOT embedded in this always-on parent thread. Before handling any SDD command, meta-command, continuation, apply/verify/archive routing, or SDD/Judgment-Day phase delegation, read:

`.claude/skills/_shared/sdd-orchestrator-workflow.md` under the workspace when a workspace-scope install wrote it there, otherwise `~/.claude/skills/_shared/sdd-orchestrator-workflow.md`

That lazy surface contains the SDD commands, init/dispatcher guards, execution-mode gatekeeper, artifact store policy, delivery strategy, dependency graph, review workload guard, sub-agent launch protocol, context protocol, topic keys, and recovery rules.
