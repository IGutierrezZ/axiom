# Axiom — Orchestrator Rule for Gemini

Bind this to the dedicated `sdd-orchestrator` agent or rule only. Do NOT apply it to executor phase agents such as `sdd-apply` or `sdd-verify`.

## Agent Teams Orchestrator

{{GENTLE_AI_SDD_SECTION:Agent Teams Orchestrator}}

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

Use Gemini's native bounded worker for delegated-direct work; reserve `sdd-*` agents for a selected SDD route.

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

{{GENTLE_AI_SDD_SECTION:Cost and Context Balance}}

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

### SDD Independent Review

{{GENTLE_AI_SDD_SECTION:SDD Independent Review}}

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

{{GENTLE_AI_SDD_SECTION:Review Workload Guard (MANDATORY)}}

### Sub-Agent Launch Deduplication (MANDATORY)

{{GENTLE_AI_SDD_SECTION:Sub-Agent Launch Deduplication (MANDATORY)}}

### Sub-Agent Launch Pattern

ALL sub-agent launch prompts that involve reading, writing, or reviewing code MUST include pre-resolved **skill paths** from the skill registry. Follow the **Skill Resolver Protocol** (see `_shared/skill-resolver.md` in the skills directory).

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

When launching sub-agents for SDD phases with engram mode, pass these exact topic_keys as artifact references:

{{GENTLE_AI_SDD_SECTION:Engram Topic Key Format (Generic and Cursor)}}

### State and Conventions

Convention files under `{{AXIOM_SKILLS_DIR}}/_shared/`: `engram-convention.md`, `persistence-contract.md`, `openspec-convention.md`.

### Recovery Rule

{{GENTLE_AI_SDD_SECTION:Recovery Rule}}
