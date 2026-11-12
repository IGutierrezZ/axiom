# Optional SDD Diagnostic Report

Use only sections useful for the requested checks. This outline is not a schema or an archive certificate.

- **Scope:** change, implementation inspected, available artifacts, and TDD mode.
- **Observed progress:** completed and unfinished tasks; do not alter their recorded state.
- **Checks:** exact commands, exit codes, relevant output, and skipped/unavailable checks with reasons.
- **Findings:** concrete behavior, requirement or design discrepancies, severity, and supporting evidence. Separate runtime proof from static observations and assumptions.
- **Historical context:** identify prior report claims and their dates/sources; preserve unresolved findings and explain any changed conclusion. When a new verify replaces a report that holds a verification matrix, move the earlier matrix here and lower its heading to level 3 with the date, `### Matriz de verificación (histórica, AAAA-MM-DD)`, so the report never has two level-2 matrix sections.
- **Matriz de verificación:** only in an ISO change, in the consolidated report (the single role `fullstack`). A level-2 section with the exact heading `## Matriz de verificación`, additional to "Cumplimiento técnico". One row per affected scenario; automatic rows name a test reference (`ruta/fichero:símbolo` or `TestX`), the command and its integer exit code; manual rows name the person, the date and the justification. Never write a line that starts with `verdict:` in it. The full format is in `../../_shared/sdd-iso-convention.md`.
- **Avisos de contexto:** only with `technicalContext.maintained` declared and a context delta that says `Sin impacto`. List each path of the diff that matches the closed pattern list in `SKILL.md`, and invite the reader to re-evaluate the "Sin impacto" decision. The warning is informational and never alters `nextRecommended` or `dependencies`. Omit the section when the delta has entries or no path matches.
- **Cumplimiento técnico:** only in a per-role verify. Contrast the role's diff with the rule sources and use the finding shape of `technical-compliance.md`; with no findings, say so and list the sources consulted. Never write a line that starts with `verdict:` in this section: the multi-role barrier reads the first such line of the report and its meaning stays functional.
- **Barrera:** the functional `verdict:` line, if any, keeps its meaning and position; the new sections are additional content that the barrier ignores.
- **Summary:** what was verified, what failed or remains unknown, and recommended next work.
- **Estado:** only in a per-role verify, as the last line of the summary: `**Estado:** limpio` or `**Estado:** no limpio — <reason>`. Apply the rule of `technical-compliance.md`.

Do not invent output hashes, passing executions, or historical TDD evidence. A missing test or unavailable tool is a limitation, not proof of correctness. Findings are diagnostic; archive may record them without a verification verdict or mandatory remediation cycle. No RDD state is read or required.

When Strict TDD is active, use `strict-tdd-verify.md` to assess available TDD evidence for implemented work without claiming that unobserved RED/GREEN execution occurred.
