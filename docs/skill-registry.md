# Skill Registry

← [Back to README](../README.md)

The skill registry is a project-local index that lets every supported agent find the same skills without rewriting them. It stores skill names, full descriptions, scopes, and exact `SKILL.md` paths.

## When To Use It

Use `axiom skill-registry refresh` after you add, remove, rename, or move skills. Normal installs wire this refresh into startup hooks where the agent supports them, including Codex, Claude Code, OpenCode, and Pi through `gentle-pi`.

## Runtime Flow

```text
User task
   │
   ▼
Orchestrator reads .atl/skill-registry.md
   │
   ▼
Matches task + file context against full skill descriptions
   │
   ▼
Passes exact SKILL.md paths to subagent
   │
   ▼
Subagent reads full skills before work
   │
   ▼
Subagent executes with original skill intent preserved
```

## Refresh Flow

```text
axiom skill-registry refresh
   │
   ├─ Scan project skill roots first
   │     skills/, workspace.skill_roots from axiom.yaml, .opencode/skills/, .claude/skills/, ...
   │
   ├─ Scan global agent skill roots second
   │     ~/.config/opencode/skills/, ~/.claude/skills/, ...
   │
   ├─ Deduplicate by skill name
   │     project skill wins over global skill
   │
   ├─ Parse frontmatter
   │     name + full description + path + scope
   │
   └─ Write .atl/skill-registry.md + cache
```

## Declared Skill Roots

A project that keeps its canonical skills outside `skills/` declares those directories in `axiom.yaml`:

```yaml
workspace:
  skill_roots:
    - "internal/assets/skills"
```

Each entry must be a relative path that stays inside the project. Absolute paths, paths that leave the project (`../x`) and `.` are ignored by the scan, and `axiom workspace validate` reports them. Declared roots are scanned right after `skills/` and before the per-agent directories (`.claude/skills`, `.gemini/skills`, ...), so the declared copy wins when a skill name appears in both.

## Versioned View

One scan feeds two views, because the destinations have different audiences:

| Destination | View | Contains |
| --- | --- | --- |
| `.atl/skill-registry.md` (gitignored, per machine) | Full | Every scanned skill: project skills, per-agent copies such as `.claude/skills`, and user-scope skills from `$HOME`, with the paths this machine resolves. |
| `## Skills` section of `AGENTS.md` and the Engram `skill-registry` topic | Versioned | Only the skills every clone can resolve, with paths relative to the project. |

`AGENTS.md` is committed and read by every collaborator, so it must not change with the machine that regenerated it. A skill is versioned when both conditions hold:

- Its `SKILL.md` is inside the project. User-scope skills and roots outside the project (`~/.claude/skills`, `../shared`) never appear in `AGENTS.md`.
- Git does not ignore it. The check is `git check-ignore` without `--no-index`, so a file that is tracked but sits under an ignored directory still counts as versioned.

Entries that are not versioned are dropped before the name deduplication. When a skill exists both as a committed copy and as an ignored copy, `AGENTS.md` lists the committed one.

The classification costs one `git` call per refresh. If git is not installed, the directory is not a repository or git does not answer in time, nothing is filtered by git and every skill inside the project counts as versioned. A git failure never fails the refresh.

The cache fingerprint includes which entries are versioned, so editing `.gitignore` regenerates the files on the next refresh without `--force`.

> **Generated agent copies.** `axiom setup` writes copies of the skills into per-agent directories such as `.claude/skills` or `.gemini/skills`. They only stay out of `AGENTS.md` when git ignores them. If your project does not ignore one of those directories, its copies are listed as versioned and the index changes with whoever ran setup last. Add the directory to `.gitignore`, or keep the canonical skills in `skills/` or in a declared root so that they win the deduplication.

## Registry Contract

The registry is an **index**, not a generated summary.

| Field | Meaning |
| --- | --- |
| `Skill` | Skill `name` from frontmatter, or directory name fallback |
| `Trigger / description` | Full `description`, including YAML folded multiline descriptions |
| `Scope` | `project` or `user` |
| `Path` | Exact `SKILL.md` file to load |

## Skill Loading Contract

Delegators pass paths, not digested rules:

```markdown
## Skills to load before work

Read these exact files before reading, writing, reviewing, testing, or creating artifacts:

- /path/to/skills/go-testing/SKILL.md
- /path/to/skills/docs-writer/SKILL.md
```

The subagent then reads those files. This keeps the original `SKILL.md` as the source of truth and avoids breaking author intent through automatic summarization.

## Skill Authoring Flow

```text
New reusable pattern
   │
   ▼
skill-creator creates SKILL.md
   │
   ▼
skill-registry indexes SKILL.md path and full description
   │
   ▼
orchestrator passes matching paths to agents
```

## Skill Improvement Flow

```text
Existing skills
   │
   ▼
skill-improver reads .atl/skill-registry.md
   │
   ▼
Audits each indexed SKILL.md against docs/skill-style-guide.md
   │
   ├─ Audit mode: report issues only
   │
   └─ Apply mode: safely refactor skills and preserve intent
   │
   ▼
Run axiom skill-registry refresh again
```

## Why Not Compact Rules?

Compact rules were cheaper per delegation but could distort skills. The index-first design spends tokens only when a subagent actually needs a skill, and it preserves the complete runtime contract.

| Design | Benefit | Tradeoff |
| --- | --- | --- |
| Compact summaries | Small prompt injection | Can lose nuance and break custom skills |
| Index + paths | Preserves full skill intent | Subagents read selected full skills |

## Excluded Skills

The registry never indexes `_shared`, `skill-registry`, or any `sdd-*` skill.
The first two are internal plumbing; `sdd-*` skills are orchestrator-managed by
the SDD workflow, not delegator-selected. This exclusion is intentional and
silent, so a user skill whose name collides with these prefixes is dropped
without a warning.

## Inspecting Without Writing

`skill-registry list` resolves the same deduplicated skill set as `refresh` (the
full view of `.atl/skill-registry.md`), but prints it instead of writing
`.atl/skill-registry.md`, the cache, or `.gitignore`. Handy for debugging what a
delegator would see.

```bash
axiom skill-registry list          # name<TAB>scope<TAB>path
axiom skill-registry list --json   # machine-readable, includes descriptions
```

## Quick Check

```bash
axiom skill-registry refresh --force
```

Open `.atl/skill-registry.md` and verify each row has a useful description and a real `SKILL.md` path.

← [Back to README](../README.md)
