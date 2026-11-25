# Organic Implementation Trigger Rules

<- [Back to README](../README.md)

Ask for the outcome. Axiom keeps already-understood work inline, delegates only
the actions that benefit from fresh context, and offers SDD only when durable
planning would materially reduce uncertainty. Native providers own verification
and lifecycle authority; ordinary repository policy owns delivery.

## Quick path

1. Describe the outcome in natural language.
2. Axiom uses the smallest useful implementation route: direct inline,
   delegated direct, or an optional SDD proposal.
3. The normal interaction reports only **Working**, **Checking**, **Ready**, or
   **Needs your decision**.

The user does not choose internals, hashes, or lifecycle
transitions. A question is necessary only when the answer changes requested
scope, destructive or irreversible impact, permission or security exposure,
verification cost or external side effects, accepted residual risk, or delivery.

## Implementation routes

| Route | Use it when | What happens |
|---|---|---|
| **Direct inline** | Deciding or verifying fits the **evidence budget** (one parallel batch, at most 3 calls, about 10k tokens); or the unit is **about 100 changed lines or less**; or the change is mechanical and already understood, with no research or unresolved design decision. | Keep the bounded action inline. |
| **Delegated direct** | Understanding exceeds the evidence budget; or a writer has a **named reason** (independent units in parallel, or protecting the parent context); or reading prepares a delegated write, or broad research is needed. | Delegate one read-only explorer (handoff of at most about 2k tokens with `path:line` evidence, one parent spot check) and/or one bounded writer per unit. |
| **Optional SDD** | The work has substantial ambiguity, or durable proposal, spec, design, or task artifacts would materially reduce uncertainty. | Propose SDD. Select it only after an explicit request or an accepted proposal. |

The evidence budget describes the context needed for the current action, not a risk
score and not an SDD threshold. Risk may strengthen native verification or
review, but it never forces SDD. Each trigger holds only when its condition truly holds, and when in doubt the work stays inline; the number of files never fires delegation.

Delegation also applies per action. Tests, builds, installs, and native review
actors may use fresh workers without changing the implementation route or
creating an SDD run. Direct and delegated work create no SDD artifacts, phase
attempts, or synthetic SDD lifecycle.

If apparently simple work reveals substantial ambiguity, Axiom may offer SDD
at the next safe boundary. Declining it leads to a safely reduced scope, a
justified direct or delegated route, or **Needs your decision**—never silent SDD
enrollment.

## Informed approval and continuity

A short affirmative reply ("yes", "go ahead", "ok", or the equivalent in your
language) is read against the proposal or question immediately before it and
approves exactly that, within its stated scope; it never widens the scope. If that
proposal was ambiguous or offered several options, a short reply does not choose
and Axiom asks again. Contextual approval never replaces native consent, the exact
allowed answers of a blocking prompt, or the separate explicit permission that
publishing, remote, and destructive actions need. Resolving a subdecision does not
cancel the objective you already authorized: Axiom continues with the authorized
work that remains.

## Native progress and authority

| Public state | Meaning |
|---|---|
| **Working** | The implementation can still change. |
| **Checking** | Axiom is performing the applicable functional proof and bounded review. |
| **Ready** | The exact candidate has sufficient evidence for the selected delivery route. |
| **Needs your decision** | Safe automatic convergence is impossible; Axiom presents the cause, impact, and concrete choices. |

The user still asks only for the outcome. Repository identity, route, policy,
candidate, delivery mechanism, and authority references remain owner-derived.
Adapters do not reconstruct recovery policy or infer
success from prose. Existing SDD v1 runs continue through their SDD-specific
status contract; direct and delegated runs do not create or consume an SDD run.

## Installation and refresh

`axiom install` and `axiom sync` project the same canonical rules into
every supported adapter, independently of whether the optional SDD component is
selected:

- Standard adapters receive the managed `agent-routing` marker in their
  adapter-owned system-prompt file.
- OpenCode and Kilocode receive it inside
  `agent.axiom-orchestrator.prompt` in their adapter-owned `opencode.json`.
- Jinja-backed adapters receive an `agent-routing.md` module included by their
  managed router template.

```bash
axiom install   # full install
axiom sync      # refresh managed content
```

Refresh is idempotent: the managed projection is replaced without duplication.

## Source of truth

The rendered projection comes from
[`internal/components/agentguidance/routing.go`](../internal/components/agentguidance/routing.go),
and its adapter delivery is owned by
[`internal/components/agentguidance/inject.go`](../internal/components/agentguidance/inject.go).
Canonical route facts come from
[`internal/agents/capabilitymanifest/manifest.go`](../internal/agents/capabilitymanifest/manifest.go).
