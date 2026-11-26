# persona-behavior-contract Specification

## Requirements

### Requirement: Artifact Language Independence

Persona voice MUST govern only direct chat replies to the user. Generated technical artifacts MUST default to English and neutral professional wording regardless of the persona, unless the user explicitly requests a different artifact language or the existing project artifact convention requires it.
(Previously: the scenarios used the retired `neutral` and `gentleman` personas.)

#### Scenario: Axiom keeps generated artifacts free of conversational voice

- GIVEN the Axiom persona is installed
- WHEN the system generates code, identifiers, comments, UI copy, documentation, commit messages, PR descriptions, SDD artifacts, or tests without an explicit request for another language or tone
- THEN the generated artifact uses neutral professional wording without the persona's conversational register
- AND it defaults to English unless the project artifact convention requires otherwise

### Requirement: Safe Persona Fallback Semantics

When the `persona` component is selected, install and sync MUST apply the Axiom persona. Persisted persona state that is missing, empty, `axiom`, or a retired or unknown value (such as `gentleman`, `neutral`, `gentleman-neutral-artifacts` or `custom`) MUST NOT block sync and MUST NOT select any other persona. A retired or unknown persisted value MUST be reported with a warning and MUST NOT be written back to the state.
(Previously: the fallback for missing or invalid state was the `neutral` persona, and an explicit `gentleman` selection was honored.)

#### Scenario: Missing persisted persona applies Axiom

- GIVEN persisted persona state is absent
- WHEN sync applies the `persona` component
- THEN it applies the Axiom persona

#### Scenario: Retired persisted persona applies Axiom with a warning

- GIVEN persisted state contains persona `gentleman`
- WHEN sync loads the state and applies the `persona` component
- THEN it applies the Axiom persona
- AND it reports that the persisted persona no longer exists
- AND the next state write no longer contains the retired persona

### Requirement: Canonical Tone Channel for Output-Style-Capable Adapters

For any adapter with an active output-style channel — Claude Code (gated by `SupportsOutputStyles()`) — the output-style asset MUST be the exclusive source of persona tone, language, and philosophy. The system-prompt persona section for these adapters MUST be reduced to a residual block containing only `## Rules`, `## Expertise`, `## Contextual Skill Loading (MANDATORY)` and a one-line pointer to the output style. The residual MUST NOT duplicate tone/language/philosophy content.
(Previously: the scenario covered the retired `gentleman` and `neutral` personas; Kimi was an explicit carve-out before C-08-d.)

#### Scenario: Claude residual section carries no tone content

- GIVEN Claude assets are generated with the Axiom persona
- WHEN the CLAUDE.md persona section is inspected
- THEN it contains only Rules, Expertise, Contextual Skill Loading and a pointer to the output style
- AND it contains no tone, language, or philosophy prose

### Requirement: Persona Section Parity for Non-Output-Style Adapters

Adapters without an active output-style channel MUST continue to receive the full persona section — tone, language, philosophy, rules, expertise, and skill loading — unchanged by this change.
(Previously: the scenario excluded Kimi as well as Claude Code.)

#### Scenario: Non-output-style adapter keeps full persona section

- GIVEN an adapter other than Claude Code is generated with a persona
- WHEN its persona asset is inspected
- THEN it contains the full persona section, including tone/language/philosophy, as before this change

### Requirement: Idempotent Convergence on Re-Injection

Upgrading an installation that has an existing full (pre-slim) persona section MUST converge the on-disk system-prompt persona section to the residual block via the marker-delimited section-replace mechanism. Repeated injection MUST be idempotent.

#### Scenario: Existing full section converges to residual, then stays stable

- GIVEN an installation has a full, pre-slim persona section on disk
- WHEN injection runs after this change, then runs again
- THEN the first run replaces the section with the residual block, leaving no orphaned tone content outside the marker boundaries
- AND the second run leaves the on-disk content unchanged

---

### Requirement: Legacy Fingerprint Continuity

Uninstall and auto-heal fingerprint checks that rely on literal markers from the persona section (e.g., `## Rules`, `Senior Architect`) MUST continue to match on-disk content after tone/philosophy content is removed, or the fingerprint constants MUST be updated so uninstall and auto-heal continue to function correctly.
(Previously: the scenario also covered Kimi installations.)

#### Scenario: Uninstall and auto-heal fingerprints still match after slimming

- GIVEN a Claude installation has the residual persona section on disk
- WHEN the uninstall cleaner and auto-heal each check their persona-section fingerprint
- THEN both checks match the residual on-disk content
- AND auto-heal does not false-positive a repair

### Requirement: Per-Adapter Test and Golden Coverage

Automated tests and golden fixtures MUST cover the residual persona section, the reconciled canonical output-style content, and unaffected non-output-style adapter behavior, and MUST be updated in strict-TDD lockstep with implementation changes.
(Previously: Kimi's residual section and output style were covered by unit assertions.)

#### Scenario: Claude goldens capture the residual section

- GIVEN Claude golden fixtures are regenerated after this change
- WHEN the persona section of the golden is inspected
- THEN it matches the residual block, not the prior full section

#### Scenario: Non-output-style adapter goldens remain unchanged

- GIVEN a golden fixture for a non-output-style adapter existed before this change
- WHEN the same fixture is regenerated after this change
- THEN its persona section content is unchanged

### Requirement: Axiom Persona Definition and Default Selection (REQ-10.1 & REQ-10.2)

The system MUST support `axiom` as its only managed persona. The persona MUST prescribe senior architect guidance in Peninsular Spanish (castellano peninsular) with professional tuteo ("tienes", "haz", "revisa"), strictly prohibiting forced Rioplatense voseo ("che", "tenés", "podés", "hacé") and informal regionalisms. The CLI MUST NOT offer a persona choice: selecting the `persona` component installs the Axiom persona, and deselecting it installs none.
(Previously: `axiom` was the default among several personas, and the TUI and CLI presented it as the default selection.)

#### Scenario: Axiom is the only persona and uses Peninsular Spanish

- GIVEN the `persona` component is selected
- WHEN Axiom installs it
- THEN the installed persona is `axiom`
- AND its instructions enforce Peninsular Spanish without voseo or Rioplatense regionalisms

### Requirement: SDD Language Domain Contract for Axiom Projects (REQ-10.3 & REQ-10.4)

In Axiom-governed projects and repositories with Spanish language rules declared, all SDD technical artifacts (`proposal.md`, `spec.md`, `design.md`, `tasks.md`, `verify-report.md`, `archive-report.md`) MUST be authored in Spanish (castellano peninsular). English is strictly preserved for source code identifiers and language keywords. Orchestrators MUST NOT enforce unconditional "default to English" rules for documentation when operating under Axiom governance.

#### Scenario: SDD technical artifacts generated in Spanish in Axiom projects

- GIVEN an SDD orchestration session runs in an Axiom-governed workspace
- WHEN SDD artifacts are authored
- THEN all proposals, specifications, designs, tasks, verification reports, and archive reports are written in Spanish
- AND English is reserved exclusively for source code identifiers

