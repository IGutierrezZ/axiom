---
name: hermes-ephemeral-delegation
description: "Trigger: exploration beyond the evidence budget, parallel writers, tests/builds, fresh review, multi-step debug. Orchestrate via delegate_task to save context."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## Activation Contract

Load this skill when you are acting as the parent orchestrator and the work ahead falls into any of these categories:

- Exploration that exceeds the inline evidence budget (one parallel batch, at most 3 calls, approximately 10k tokens; codebase mapping, approach comparison)
- Implementation with a named reason to delegate a writer (independent units in parallel, or protecting the parent context); the number of files never counts
- Test or build execution
- Fresh adversarial review (diffs, PR readiness, incident audit)
- Multi-step debugging that would flood the parent context

Do NOT load this skill if you are already inside a delegated child task — you are the executor, not the orchestrator.

## Hard Rules

- Use `delegate_task` for the work listed above only when the gate's condition truly holds; when in doubt, work inline. Do NOT execute work inline that a gate clearly delegates.
- Workers are EPHEMERAL: each `delegate_task` call creates a fresh context. Do NOT request persistent agent files or profiles.
- Pass a self-contained mission. Workers have no memory of the parent conversation.
- Treat worker output as self-report: verify file writes, test pass/fail, URLs, and IDs before reporting success to the user.
- Batch parallel calls only for INDEPENDENT workstreams. Sequential dependencies must run sequentially.

## Decision Gates

| Situation | Action |
|-----------|--------|
| Understanding exceeds the inline evidence budget | Delegate a read-only exploration worker that returns a handoff of at most approximately 2k tokens with `path:line` evidence; make one spot check |
| Independent units to write in parallel, or a long session whose parent context needs protecting | Delegate one bounded writer per unit with the full mission |
| Need to run tests or builds | Delegate an executor; do not run inline |
| Need an adversarial review of a diff | Delegate a fresh-context reviewer |
| Multi-step debug that grows the context | Delegate a debug worker; feed results back inline |
| A unit of about 100 changed lines or less, or a change you already understand | Do it inline, whatever the number of files; no delegation needed |
| Quick git/state check | Do it inline; no delegation needed |

## Execution Steps

1. Identify which gate applies. If none applies, skip delegation.
2. Draft a self-contained mission for the worker — include:
   - Exact goal (one sentence)
   - File paths or targets to act on
   - Relevant prior context the worker needs (decisions, conventions, prior findings)
   - Constraints (style, test runner, budget)
   - Expected evidence to return (e.g., file written, test output, URL found)
   - Allowed toolsets/MCP/skills the worker should use
   - Any `SKILL.md` paths to load before work
3. Call `delegate_task` with that mission.
4. Wait for the worker summary.
5. Verify the claimed output (check file existence, test result, side effect).
6. Synthesize the verified result into your orchestrator reply.

## Output Contract

After synthesizing worker results, return:

- What was delegated and to how many workers
- What each worker returned (verified, not just claimed)
- Any discrepancy between worker self-report and verified evidence
- Final answer or next step for the user

## References

- [references/tuning-knobs.md](references/tuning-knobs.md) — Full table of `delegate_task` configuration parameters and the explicit toolset/MCP/skill checklist for worker missions.
- [../../hermes/sdd-orchestrator.md](../../hermes/sdd-orchestrator.md) — SDD orchestrator protocol that uses this delegation standard for SDD phase work.
