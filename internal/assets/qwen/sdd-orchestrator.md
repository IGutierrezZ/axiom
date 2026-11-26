# Axiom — Orchestrator Rule for Qwen Code

Bind this to the dedicated `sdd-orchestrator` agent or rule only. Do NOT apply it to executor phase agents such as `sdd-apply` or `sdd-verify`.

## Agent Teams Orchestrator

{{AXIOM_SDD_SECTION:Agent Teams Orchestrator}}

### Lossless Blocking Prompts (MANDATORY)

{{AXIOM_SDD_SECTION:Lossless Blocking Prompts Contract}}
{{AXIOM_SDD_SECTION:Lossless Blocking Prompts Native Route (Plain Fallback Only)}}
{{AXIOM_SDD_SECTION:Lossless Blocking Prompts Fallback}}
{{AXIOM_SDD_SECTION:Lossless Blocking Prompts Answer Validation}}

#### SDD Edit-Authority Consent Relay (MANDATORY)

{{AXIOM_SDD_SECTION:SDD Edit-Authority Consent Relay (MANDATORY)}}

### Language Domain Contract

{{AXIOM_SDD_SECTION:Language Domain Contract}}

### Delegation Rules

{{AXIOM_SDD_SECTION:Delegation Rules}}

Use Qwen's native bounded worker for delegated-direct work; reserve `sdd-*` agents for a selected SDD route.

{{AXIOM_SDD_SECTION:Delegation Rules Handoff}}

#### Mandatory Delegation Triggers

{{AXIOM_SDD_SECTION:Mandatory Delegation Triggers}}

#### Delegated Verification Gate (MANDATORY)

{{AXIOM_SDD_SECTION:Delegated Verification Gate (MANDATORY)}}

#### Cost and Context Balance

{{AXIOM_SDD_SECTION:Cost and Context Balance}}

## SDD Workflow (Spec-Driven Development)

{{AXIOM_SDD_SECTION:SDD Workflow (Spec-Driven Development)}}

### Artifact Store Policy

{{AXIOM_SDD_SECTION:Artifact Store Policy}}

### Commands

{{AXIOM_SDD_SECTION:Commands}}

### Native SDD Dispatcher Guard

{{AXIOM_SDD_SECTION:Native SDD Dispatcher Guard}}

### SDD Change Kickoff and Block Gates

{{AXIOM_SDD_SECTION:SDD Change Kickoff and Block Gates}}

### SDD Independent Review

{{AXIOM_SDD_SECTION:SDD Independent Review}}

### SDD Init Guard (MANDATORY)

{{AXIOM_SDD_SECTION:SDD Init Guard (MANDATORY)}}

### Execution Mode

{{AXIOM_SDD_SECTION:Execution Mode}}

### Automatic Mode Gatekeeper (MANDATORY)

{{AXIOM_SDD_SECTION:Automatic Mode Gatekeeper (MANDATORY)}}

### Artifact Store Mode

{{AXIOM_SDD_SECTION:Artifact Store Mode}}

### Delivery Strategy

{{AXIOM_SDD_SECTION:Delivery Strategy}}

### Chain Strategy

{{AXIOM_SDD_SECTION:Chain Strategy}}

### Commit Review

{{AXIOM_SDD_SECTION:Commit Review}}

### Dependency Graph

{{AXIOM_SDD_SECTION:Dependency Graph}}

### Result Contract

{{AXIOM_SDD_SECTION:Result Contract}}

### Sub-Agent Launch Deduplication (MANDATORY)

{{AXIOM_SDD_SECTION:Sub-Agent Launch Deduplication (MANDATORY)}}

### Sub-Agent Launch Pattern

ALL sub-agent launch prompts that involve reading, writing, and reviewing code MUST include pre-resolved **skill paths** from the skill registry. Follow the **Skill Resolver Protocol** (see `_shared/skill-resolver.md` in the skills directory).

The orchestrator resolves skills from the registry ONCE (at session start or first delegation), caches the skill index, and passes matching `SKILL.md` paths into each sub-agent's prompt.

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

{{AXIOM_SDD_SECTION:Skill Resolution Feedback}}

### Sub-Agent Context Protocol

{{AXIOM_SDD_SECTION:Sub-Agent Context Protocol}}

#### Non-SDD Tasks (general delegation)

{{AXIOM_SDD_SECTION:Non-SDD Tasks (general delegation)}}

#### SDD Phases

{{AXIOM_SDD_SECTION:SDD Phases}}

#### Archive Final-State Handoff (MANDATORY)

{{AXIOM_SDD_SECTION:Archive Final-State Handoff (MANDATORY)}}

#### Strict TDD Forwarding (MANDATORY)

{{AXIOM_SDD_SECTION:Strict TDD Forwarding (MANDATORY)}}

#### Apply-Progress Continuity (MANDATORY)

{{AXIOM_SDD_SECTION:Apply-Progress Continuity (MANDATORY)}}

#### Engram Topic Key Format

When launching sub-agents for SDD phases with engram mode, pass these exact topic_keys as artifact references:

{{AXIOM_SDD_SECTION:Engram Topic Key Format (Generic and Cursor)}}

### State and Conventions

Convention files under `~/.qwen/skills/_shared/` (global) or `.agent/skills/_shared/` (workspace): `engram-convention.md`, `persistence-contract.md`, `openspec-convention.md`.

### Recovery Rule

{{AXIOM_SDD_SECTION:Recovery Rule}}
