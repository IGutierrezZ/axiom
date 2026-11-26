# Guard population declarations

Guard population declarations make the accepted input set of selected production guards explicit at the check itself. The contract covers the evidenced guard families in `internal/cli`, `internal/fileprim`, and `internal/sddstatus`; the families of the retired review-transaction package left with it.

## Review rule

A production Go guard qualifies when all of these are true:

1. It is in one of the three scoped packages.
2. It decides whether external, repository, filesystem, or persisted state is legitimate for a security, integrity, admission, repair, or governance boundary.
3. It belongs to one of the registered families below.

Arbitrary control flow does not qualify. Shell and workflow guards are out of scope for v2.2.1.

| Family | Legitimate population under review |
|---|---|
| `darwin-search-ancestor` | Directory permission shapes allowed during secure ancestry traversal |
| `persisted-sync-state-integrity` | Persisted sync state admitted before persona mutation |

Reviewers own identification of a new or omitted qualifying guard. The mechanism cannot derive the real-world population from source and MUST NOT be described as semantic completeness.

## Declaration contract

Place one declaration immediately above the `if`, `switch`, or `return` node that enforces the population boundary:

```go
// guard:population <family> <too-tight|too-loose|fail-closed>: <legitimate population and exclusion boundary>
```

Use `too-tight` when drift is expected to reject legitimate inputs, `too-loose` when it may admit illegitimate inputs, and `fail-closed` when the important contract is safe refusal under uncertainty.

`TestEveryRegisteredGuardPopulationDeclarationMatchesProduction` AST-binds each declaration to the adjacent guard node. `.guard-population-baseline.txt` freezes the source path, family, direction, claim, node kind, and guard-node fingerprint. The test fails in both directions: a declaration missing from the registry and a registry entry missing from production are both drift.

After an intentional reviewed declaration or guard-node change, regenerate the registry before final verification:

```bash
AXIOM_GUARD_POPULATION_UPDATE=1 go test ./internal/cli -run TestEveryRegisteredGuardPopulationDeclarationMatchesProduction -count=1
```

## Proof boundary

The contract proves declaration presence, AST adjacency, and exact registry agreement. It does not prove that the population claim is true, that every qualifying guard was identified, or that tests sample the outside world. Review must challenge the claim against production platforms, repository shapes, persisted states, and incident evidence.
