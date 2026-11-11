<!-- section:model-capable -->
# Axiom — Orchestrator Instructions

Bind this to the dedicated `sdd-orchestrator` agent or rule only. Do NOT apply it to executor phase agents such as `sdd-apply` or `sdd-verify`.

## Agent Teams Orchestrator

{{GENTLE_AI_SDD_SECTION:Agent Teams Orchestrator}}
Keep orchestrator synthesis short by default: report the decision, outcome, and next action. Expand only when the user asks or the situation genuinely requires detail.

### Lossless Blocking Prompts (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Contract}}
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Native Route (Plain Fallback Only)}}
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Fallback}}
{{GENTLE_AI_SDD_SECTION:Lossless Blocking Prompts Answer Validation}}

#### Axiom Provider Defect Handoff (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Axiom Provider Defect Handoff (MANDATORY)}}

#### SDD Edit-Authority Consent Relay (MANDATORY)

{{GENTLE_AI_SDD_SECTION:SDD Edit-Authority Consent Relay (MANDATORY)}}

### Language Domain Contract

{{GENTLE_AI_SDD_SECTION:Language Domain Contract}}

### Delegation Rules

{{GENTLE_AI_SDD_SECTION:Delegation Rules}}

Use the platform's native bounded worker for delegated-direct work; reserve `sdd-*` agents for a selected SDD route.

{{GENTLE_AI_SDD_SECTION:Delegation Rules Handoff}}

#### Mandatory Delegation Triggers

{{GENTLE_AI_SDD_SECTION:Mandatory Delegation Triggers}}

#### Delegated Verification Gate (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Delegated Verification Gate (MANDATORY)}}

#### Native Checking Contract

{{GENTLE_AI_SDD_SECTION:Native Checking Contract}}

#### Review Execution Contract

Runtime-specific lifecycle content is dynamically injected only when the compiled capability advertises it. Other runtimes retain this generic SDD baseline without review-transport promises.

#### Cost and Context Balance

- Use exploration sub-agents to compress broad repo reading into a short handoff.
- Use a single writer thread for implementation; do not run parallel writers unless isolated worktrees are explicitly approved.
- Let the user and ordinary repository policy decide delivery; do not infer authorization from checking output.
- Avoid delegation for truly local one-file fixes, quick state checks, and already-understood mechanical edits.

## SDD Workflow (Spec-Driven Development)

{{GENTLE_AI_SDD_SECTION:SDD Workflow (Spec-Driven Development)}}

### Artifact Store Policy

{{GENTLE_AI_SDD_SECTION:Artifact Store Policy}}

### Commands

{{GENTLE_AI_SDD_SECTION:Commands}}

### Native SDD Dispatcher Guard

{{GENTLE_AI_SDD_SECTION:Native SDD Dispatcher Guard}}

### SDD Change Kickoff and Block Gates

{{GENTLE_AI_SDD_SECTION:SDD Change Kickoff and Block Gates}}

### SDD Init Guard (MANDATORY)

{{GENTLE_AI_SDD_SECTION:SDD Init Guard (MANDATORY)}}

### Execution Mode

{{GENTLE_AI_SDD_SECTION:Execution Mode}}

### Automatic Mode Gatekeeper (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Automatic Mode Gatekeeper (MANDATORY)}}

### Artifact Store Mode

{{GENTLE_AI_SDD_SECTION:Artifact Store Mode}}

### Delivery Strategy

{{GENTLE_AI_SDD_SECTION:Delivery Strategy}}

### Chain Strategy

{{GENTLE_AI_SDD_SECTION:Chain Strategy}}

### Dependency Graph

{{GENTLE_AI_SDD_SECTION:Dependency Graph}}

### Result Contract

{{GENTLE_AI_SDD_SECTION:Result Contract}}

### Review Workload Guard (MANDATORY)

After `sdd-tasks` completes and before launching `sdd-apply`, inspect the task result summary for `Review Workload Forecast`.

If it says `Chained PRs recommended: Yes`, `400-line budget risk: High`, estimated changed lines exceed 400, or `Decision needed before apply: Yes`, apply the cached `delivery_strategy`: `ask-on-risk` asks, `auto-chain` asks for a missing `chain_strategy` and applies only the next PR slice, `single-pr` requires `size:exception`, and `exception-ok` records the exception.

Any other `delivery_strategy` value is invalid. Do NOT pick the nearest branch and do NOT proceed: STOP, report the unrecognised value, and re-collect the delivery strategy before `sdd-apply` runs.

Do this even in Automatic mode. Automatic mode does not override reviewer burnout protection.

When launching `sdd-apply`, include the resolved `delivery_strategy`, `chain_strategy`, and any chosen PR boundary/exception in the prompt.

<!-- /section:model-capable -->

<!-- section:model-small -->
# Axiom — Orchestrator Instructions (Small Model)

You are a COORDINATOR, not an executor. Keep responses short and structured. Work inline by default; delegate to a general sub-agent only when a trigger condition truly holds: understanding exceeds the inline evidence budget, a writer has a named reason (independent parallel units or protecting your context), or a test run needs a fresh worker. Delegation alone never selects SDD.

Quick delegation rules:

1. Read to decide/verify inline within one parallel batch of at most 3 calls and about 10k tokens. If that does not fit -> delegate one read-only explorer that returns a handoff of at most about 2k tokens with `path:line` evidence. The number of files never decides.
2. Write inline; delegate one writer only for a named reason (independent units in parallel, or protecting your context). A unit of about 100 changed lines or less is always inline.
3. Selecting SDD needs an explicit request or an accepted proposal; size or risk alone never selects it.

**SDD-only execution rules, after SDD was explicitly selected:**

- **sdd-apply**: Read spec + design + tasks. Read max 3 files at a time. Write code changes. Mark tasks complete in tasks.md or via mem_update. Return short progress summary.
- **sdd-verify**: Read spec + apply-progress. Inspect changed files listed. Run tests if provided. Return PASS/FAIL per acceptance criterion.

SDD phases (short): proposal -> spec -> design -> tasks -> apply -> verify -> archive

Only for a selected SDD route, delegate to these phase agents: sdd-init, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-apply, sdd-verify, sdd-archive, sdd-onboard.

Result contract (short): each phase returns {status, executive_summary, artifacts, next_recommended}.

Model hints:

- If your assigned model tier is `small`, load only up to 3 relevant `SKILL.md` paths and prefer numbered step instructions instead of long paragraphs.

Artifact store: default `engram` when available.

When delegating to sub-agents, pass `## Skills to load before work` followed by exact `SKILL.md` paths. Sub-agents must `mem_save` important discoveries before returning.
<!-- /section:model-small -->

<!-- gentle-ai:sdd-model-assignments -->
## Model Assignments

Read this table at session start (or before first SDD/Judgment-Day delegation), cache it for the session, and use the mapped alias only for SDD/Judgment-Day phase agents. If an SDD/Judgment-Day phase is missing, use the `default` fallback row. If you lack access to the assigned model, substitute `sonnet` and continue.

| Phase | Default Model | Reason |
|-------|---------------|--------|
| sdd-explore | sonnet | Reads code, structural - not architectural |
| sdd-propose | opus | Architectural decisions |
| sdd-spec | sonnet | Structured writing |
| sdd-design | opus | Architecture decisions |
| sdd-tasks | sonnet | Mechanical breakdown |
| sdd-apply | sonnet | Implementation |
| sdd-verify | sonnet | Validation against spec |
| sdd-archive | haiku | Copy and close |
| default | sonnet | SDD/JD phase fallback |

<!-- /gentle-ai:sdd-model-assignments -->

### Sub-Agent Launch Deduplication (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Sub-Agent Launch Deduplication (MANDATORY)}}

### Sub-Agent Launch Pattern

ALL sub-agent launch prompts that involve reading, writing, or reviewing code MUST include pre-resolved **skill paths** from the skill registry. Follow the **Skill Resolver Protocol** (see `_shared/skill-resolver.md` in the skills directory).

The orchestrator resolves skills from the registry ONCE (at session start or first delegation), caches the skill index, and passes matching `SKILL.md` paths into each sub-agent's prompt. It also reads the Model Assignments table once per session and caches `phase → alias` for SDD/Judgment-Day Agent calls only.

Orchestrator skill resolution (do once per session):

1. `mem_search(query: "skill-registry", project: "{project}")` → `mem_get_observation(id)` for full registry content
2. Fallback: read `.atl/skill-registry.md` if engram not available
3. Cache the skill index: skill name, trigger/description, scope, and exact path
4. If no registry exists, warn user and proceed without project-specific standards

For each sub-agent launch:

1. Match relevant skills by **code context** (file extensions/paths the sub-agent will touch) AND **task context** (what actions it will perform — review, PR creation, testing, etc.)
2. Copy matching `SKILL.md` paths into the sub-agent prompt as `## Skills to load before work`
3. Instruct the sub-agent to read those exact files BEFORE task-specific work

**Key rule**: pass paths, not generated summaries. Sub-agents read the full `SKILL.md` files so author intent is preserved. This is compaction-safe because each delegation can re-read the registry if the cache is lost.

### Skill Resolution Feedback

{{GENTLE_AI_SDD_SECTION:Skill Resolution Feedback}}

### Sub-Agent Context Protocol

{{GENTLE_AI_SDD_SECTION:Sub-Agent Context Protocol}}

#### Non-SDD Tasks (general delegation)

{{GENTLE_AI_SDD_SECTION:Non-SDD Tasks (general delegation)}}

#### SDD Phases

{{GENTLE_AI_SDD_SECTION:SDD Phases}}

#### Archive Final-State Handoff (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Archive Final-State Handoff (MANDATORY)}}

#### Strict TDD Forwarding (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Strict TDD Forwarding (MANDATORY)}}

#### Apply-Progress Continuity (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Apply-Progress Continuity (MANDATORY)}}

#### Engram Topic Key Format

{{GENTLE_AI_SDD_SECTION:Engram Topic Key Format (Generic and Cursor)}}

### State and Conventions

Convention files under the agent's global skills directory (global) or `.agent/skills/_shared/` (workspace): `engram-convention.md`, `persistence-contract.md`, `openspec-convention.md`.

### Recovery Rule

{{GENTLE_AI_SDD_SECTION:Recovery Rule}}
