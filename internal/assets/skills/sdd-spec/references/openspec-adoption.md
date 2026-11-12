# OpenSpec adoption: migrating pre-existing requirements

Read this reference when `openspec/changes/{change-name}/adoption-inventory.yaml` exists. That file is the deterministic trigger: the change is an **adoption change**, an ordinary SDD change (kickoff, gates, archive) whose spec phase migrates a project's pre-existing requirement documents, in any format, into `openspec/specs/`. OpenSpec is the only living-spec format.

## Who checks what

- Go (`axiom archive adopt scan` and `axiom archive adopt check`) verifies form and coverage: format, unique identifiers, one new series, scenarios, resolvable `Origen:` lines, and that every inventory unit is migrated or discarded with a reason.
- A human approves, at the `spec` gate, that the migration is **faithful** to its sources. No command judges semantic fidelity: a requirement can pass every check and still misstate its source. `adopt check` never records or replaces a gate decision.

## Flow

1. **Inventory.** From the workspace root of the specs repository, run `axiom archive adopt scan --path "<workspaceRoot>" --change "{change-name}"`. It is a guarded writer: it writes only `adoption-inventory.yaml` in the change folder, and outside the change branch it refuses with `wrong_branch` and the `axiom branch ensure` continuation (execute that continuation, then repeat). Without `--change` it only reports and writes nothing. Read the inventory: each source with its `path`, each unit with its `anchor`, its title and the foreign identifiers (for example `RF-120`) found in it, and the `context_candidates` it proposes. If it lists no sources, report that there is nothing to migrate.
2. **Read every unit.** Open each source and read each unit before writing: the inventory is the coverage reference, not the content.
3. **Write one delta per capability** at `openspec/changes/{change-name}/specs/<capability>/spec.md`. For an adoption change the capability file is a delta with **only `## ADDED Requirements`** (never MODIFIED, REMOVED or RENAMED), even for a new capability: this takes precedence over the full-spec shape that Step 2 describes for new capabilities. Each requirement carries:
   - A title `### Requirement: <name> (REQ-n.m)`. Choose one series `REQ-n.*` that is new for the migrated project: `n` is not used in `openspec/specs/` and every requirement of the change uses the same `n`. The original identifier (`RF-120`, `US-4`, ...) is never the identity of the migrated requirement.
   - A normative statement (RFC 2119 keywords) in the artifact language of the project.
   - At least one line `Origen: <path>#<anchor> (<original ID>)`, one line per source unit it migrates (several lines are allowed). `<path>` is relative to the workspace root, with `/`; put it between backticks if it contains spaces; `<anchor>` is copied from the inventory; omit `#<anchor>` only for a whole-file unit; `(<original ID>)` is the foreign identifier of the unit when it has one. The archive keeps these lines verbatim in the living spec.
   - At least one `#### Scenario:` with verifiable GIVEN/WHEN/THEN steps, in the artifact language of the project.
4. **Discard with a reason.** Create `openspec/changes/{change-name}/adoption.md` with a `## Descartado` section: one bullet `- <path>#<anchor>: <reason>` per inventory unit that you do not migrate (a whole-file unit is `- <path>: <reason>`). The reason is never empty. Do not discard to silence a coverage failure: discard only what is not a requirement (a title, a glossary, an introduction).
5. **Check.** Run `axiom archive adopt check --path "<workspaceRoot>" --change "{change-name}"`. It reports every failure, not only the first, each with the rule (`format`, `uniqueness`, `series`, `scenarios`, `origin`, `coverage`, `discard_reason`, `inventory_missing`, `inventory_stale`) and the requirement or unit involved. Fix them all and repeat until it exits `0`. On `inventory_stale` a source changed after the scan: repeat step 1 and re-evaluate coverage against the new units.
6. **Gate.** Present the migration at the `spec` gate for human approval of its fidelity. Do not claim that `adopt check` verified fidelity, and do not record the gate yourself.
7. **Archive.** The adoption change is archived like any other: `axiom sdd archive-compose --canonical --delta` composes it and the living spec keeps each `Origen:` line byte for byte.

## Limits

- Never edit the source documents.
- Never invent a requirement without an `Origen:`; never migrate Axiom's own `REQ-x.y` identifiers.
- Never write `technical_context:` into `axiom.yaml` on your own: present the `context_candidates` (`maintained` or `reference`) to the user as a proposal.
- In a `multirepo`, the commands operate on the specs repository: its `docs/` is inventoried, the change lives in its `openspec/changes/`, and sources in sibling code repositories are out of the inventory.
