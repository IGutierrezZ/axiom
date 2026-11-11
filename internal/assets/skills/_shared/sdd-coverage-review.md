# SDD Coverage Review

Mandate for the optional coverage review that the orchestrator runs after `spec` and after `design`, when the session preference asks for it or the user requests it. The review is informational: it never blocks a phase, a gate, or archive, and it never edits the artifact it reviews.

## Reviewers

Launch two blind reviewers in parallel, with the identities `jd-judge-a` and `jd-judge-b`. They are clean-context and read-only by construction. Both receive the same inputs and neither sees the other's result.

State this in the prompt of each reviewer: the task reviews planning artifacts, so the sweep budget, the precision gate, and the ledger persistence of the judge definition do not apply; follow only this mandate and return the result shape below.

## Inputs

| After | Inputs |
|---|---|
| `spec` | the proposal and the specs |
| `design` | the proposal, the specs, and the design (include spec requirements that the design does not cover) |

When `spec` and `design` ran in parallel, wait until both exist before the review that follows `design`.

## Result shape

Each reviewer returns one JSON object and no prose:

```json
{"gaps":[{"kind":"use_case|edge_case|error|uncovered_requirement","artifact":"spec|design","location":"<path>#<heading>","description":"...","basis":"<quote from the proposal or the spec>"}],"evidence":["what was inspected"]}
```

A gap is a use case, edge case, or error scenario that the reviewed artifact lacks, with the quote that justifies it. Return `{"gaps":[],"evidence":["..."]}` when the artifact is complete.

## Union by the parent

The parent merges the results: the report is the union of every reviewer's gaps, deduplicated, without requiring agreement. Two gaps are the same when they share location and the missing behavior; keep one row and name both reviewers. A gap reported by one reviewer stays in the report. A reviewer without a usable result is declared as such in the report; never invent its content.

## Report

- Files: `coverage-review.spec.md` and `coverage-review.design.md` at the root of the change folder.
- Engram: `sdd/{change}/coverage-review/spec` and `sdd/{change}/coverage-review/design`. With the `hybrid` store write both; with `none`, present the report inline.
- Each run appends a section `## Ronda N — YYYY-MM-DD`; never overwrite a previous round.
- A report without gaps exists and says so.
- The report is outside the artifacts that the gates digest: it never changes a gate digest and never reopens a gate.

## No self-correction

The parent never launches `sdd-spec` or `sdd-design` because of a gap. In `checkpointed`, the person iterates the stop and the phase updates its artifact; in `continuous`, gaps go to the final summary and a later run of the phase is requested by the person. Gaps never trigger the retry of the automatic gatekeeper.
