# Joint verification (verify without a role, joint mode)

Load this reference only when the orchestrator launches you without a role and with `mode: joint`. It is informational: nothing here blocks a phase, a gate, or archive, and the verify never edits code, tasks, or planning artifacts. Its only write is a dated round appended to its own report.

The joint verification is the global verify without a role, not a new report or phase. Its scope is the union, per repository, of what `axiom branch scope` returns for the change, and it inspects commits only.

## 1. Launch conditions

The orchestrator launches this mode only when the person accepts the offer made at the `integration` gate or asks for it explicitly. A session preference never launches it. An explicit request is always honored, even with gaps; the round declares them.

## 2. Resolve the scope (read-only)

1. Run `axiom branch scope --cwd "<workspaceRoot>" --change <change> --json` now. The result recomputed at execution time wins over the one shown at the gate.
2. If the command fails, is not applicable, or its `schema` is not `axiom.branch-scope/v1`, behave as the verify without a role always does: declare the reason in the report (the refusal `code`, or `consulta no disponible`), never fail because of it, and never change the verdict.
3. Never run `fetch`, `pull`, `push`, or `ls-remote`. A gap is reported with the `fetch` the person must authorize, not filled.

## 3. Inspect commits only

For each repository whose `outcome` is `ok`, read the work with `git -C "<gitRoot>" log` and `git -C "<gitRoot>" show` over the `revisions`, `commits`, and `files` of the JSON, where `<gitRoot>` is the absolute path that the JSON returns.

- Verify commits only: never the working tree, the index, or files without tracking. Say so in the report.
- Never use the current directory, a relative path, or a path that the JSON did not return.
- List the `excluded` commits as excluded and reproduce a `foreign_commits_possible` warning as a warning. Never include them silently and never take findings or files from them.
- The full test suite that the global verify requires is not narrowed by this scope. Run it where the global verify runs it, and state in the report which `HEAD` it used and whether it equals the `tip`.

## 4. When no round is written

When every repository is `empty` or `exempt`, write no round and state in the answer that there are no own commits to verify.

## 5. The round

Append at the end of the consolidated report (the `verify_file` of `fullstack` when it is the only role; in a `multirepo`, the one in the change folder of the planning home):

```
## Verificación conjunta (AAAA-MM-DD)

Alcance: axiom.branch-scope/v1 · solo commits

### <repositorio>
- resultado: ok | empty | unverifiable (<motivo>) | exempt
- rama: <rama> · <local|remote-tracking> · fusionada en <integración>: sí | no
- tip: <sha>   rango: <start>..<tip>
- excluidos: <sha…> | ninguno   avisos: foreign_commits_possible | ninguno
- fetch a autorizar: git -C "<gitRoot>" fetch origin     (solo con branch_missing o base_missing)

**Resultado conjunto:** limpio | no limpio — <huecos>
```

- Append-only: never delete earlier rounds, never write a line that starts with `verdict:`, never touch the `**Estado:**` line, "Cumplimiento técnico", or the ISO matrix.
- The result is `no limpio` with any `unverifiable` repository (with its reason: `branch_missing`, `base_missing`, or `fork_point_undetermined`) or any branch not merged into the integration branch. `empty` and `exempt` are declared and are not gaps.
- The verify never runs `fetch` and never fills a gap on its own.

## 6. Where the orchestrator looks for rounds

The orchestrator looks for the heading `## Verificación conjunta (AAAA-MM-DD)` only in `<planningHome>/changes/<change>/verify-report.md`, the consolidated report in the change folder of the planning home. When that file does not exist or cannot be read, there is no round, and the state comes from `axiom branch scope` (offered or not verifiable).

## 7. Where the report goes

- File: the `verify_file` of the kickoff (`verify-report.md` for `fullstack`).
- Engram: `sdd/{change}/verify-report` for `fullstack`.
