package assets

import (
	"strings"
	"testing"
)

// carril-sdd-versatil U21 (REQ-29.17): pins de texto de la guía ISO, el perfil de
// bug, la matriz de verify y la excepción ISO. Las skills describen los formatos y
// el motor los valida; estas pruebas fijan que la guía conserve los literales
// exactos, porque el carril ODD usa la misma sección y la misma línea.

const sddISOConventionPath = "skills/_shared/sdd-iso-convention.md"

// sddSharedSection devuelve el cuerpo de una sección de sdd-orchestrator-sections.md.
func sddSharedSection(t *testing.T, name string) string {
	t.Helper()
	shared := MustRead("skills/_shared/sdd-orchestrator-sections.md")
	start := "<!-- sdd-orchestrator-section:" + name + ":start -->"
	end := "<!-- sdd-orchestrator-section:" + name + ":end -->"
	from := strings.Index(shared, start)
	to := strings.Index(shared, end)
	if from < 0 || to < from {
		t.Fatalf("sdd-orchestrator-sections.md has no %q section", name)
	}
	return shared[from+len(start) : to]
}

func requireContains(t *testing.T, label, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s missing %q", label, want)
		}
	}
}

func TestSDDISOConvention(t *testing.T) {
	content := MustRead(sddISOConventionPath)

	requireContains(t, sddISOConventionPath, content,
		"## Informe de bug",
		"## Impacto en la spec",
		"Sin impacto en la spec",
		"## Matriz de verificación",
		"Observado:",
		"Esperado:",
		"Referencia:",
		"Motivo:",
		"Cubre: REQ-x[, REQ-y ...]",
		"`ruta/fichero:símbolo`",
		"`TestX`",
		"axiom sdd archive-check --cwd <workspaceRoot> --change <name>",
		"compose only the domains with `alreadyComposed: false`",
		"axiom sdd trace --cwd <dir> --change <name>",
		// La matriz histórica baja a nivel 3 con fecha (REQ-28.8, D10).
		"### Matriz de verificación (histórica, AAAA-MM-DD)",
		"A report with history never has two level-2 matrix sections.",
		// Las tres rutas del bug.
		"| Existing requirement |",
		"| No impact on the spec |",
		"| Delta |",
		// REQ-29.19: en ISO el motivo es la evidencia y la matriz vacía es válida.
		"the `Motivo:` is the evidence of a bug without impact",
		"an absent or empty matrix is valid",
	)

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "verdict:") {
			t.Errorf("%s has a line that starts with `verdict:`: %q", sddISOConventionPath, line)
		}
	}
}

func TestSDDSkillsReferenceISOConvention(t *testing.T) {
	for _, path := range []string{
		"skills/sdd-propose/SKILL.md",
		"skills/sdd-spec/SKILL.md",
		"skills/sdd-tasks/SKILL.md",
		"skills/sdd-verify/SKILL.md",
		"skills/sdd-onboard/SKILL.md",
	} {
		t.Run(path, func(t *testing.T) {
			requireContains(t, path, MustRead(path), "_shared/sdd-iso-convention.md")
		})
	}

	t.Run("skills/sdd-verify/references/report-format.md", func(t *testing.T) {
		requireContains(t, "report-format.md", MustRead("skills/sdd-verify/references/report-format.md"), "_shared/sdd-iso-convention.md")
	})
}

func TestReportFormatHistoricalMatrix(t *testing.T) {
	report := MustRead("skills/sdd-verify/references/report-format.md")

	requireContains(t, "report-format.md", report,
		"**Historical context:**",
		"lower its heading to level 3 with the date",
		"### Matriz de verificación (histórica, AAAA-MM-DD)",
		"never has two level-2 matrix sections",
		"## Matriz de verificación",
		"Never write a line that starts with `verdict:` in it.",
	)

	// La skill de verify declara la matriz como sección adicional sin `verdict:`.
	requireContains(t, "sdd-verify/SKILL.md", MustRead("skills/sdd-verify/SKILL.md"),
		"additional section of the consolidated report, next to \"Cumplimiento técnico\"",
		"without any line that starts with `verdict:`",
		"A bug without impact on the spec has no matrix.",
	)
}

// carril-sdd-versatil U22 (REQ-29.17, REQ-29.10, REQ-29.11): la negativa del
// archivado ISO la decide Go, pero la skill es quien invoca el verbo; este pin
// falla si pierde el paso previo a componer o mover.
func TestSDDArchiveRequiresArchiveCheck(t *testing.T) {
	skill := MustRead("skills/sdd-archive/SKILL.md")

	from := strings.Index(skill, "### Archive Readiness")
	to := strings.Index(skill, "### Honest Partial Archive")
	if from < 0 || to < from {
		t.Fatal("sdd-archive/SKILL.md lost the Archive Readiness section")
	}
	readiness := skill[from:to]
	requireContains(t, "Archive Readiness", readiness,
		"**Archive check (mandatory, before composing or moving).**",
		"axiom sdd archive-check --cwd <workspaceRoot> --change <name>",
		"before any `axiom sdd archive-compose` and before any move",
		"On a non-zero exit, STOP and return `blocked`, quoting its typed reasons",
		"compose only the domains with `alreadyComposed: false`",
		"`../_shared/sdd-iso-convention.md`",
	)

	// El paso precede a toda composición y a todo movimiento de la skill.
	check := strings.Index(skill, "axiom sdd archive-check --cwd <workspaceRoot> --change <name>")
	for _, later := range []string{"### Archive Branch Flow", "### Step 2: Sync Delta Specs to Main Specs", "### Step 3: Move to Archive"} {
		if at := strings.Index(skill, later); at < 0 || check > at {
			t.Errorf("archive-check (offset %d) must come before %q (offset %d)", check, later, at)
		}
	}

	// REQ-29.17: el flujo de archivado no cambia; la rama `-archive` y el PR
	// documental aparte siguen siendo el único modo.
	requireContains(t, "sdd-archive/SKILL.md", skill,
		"`axiom branch ensure --cwd \"<workspaceRoot>\" --change <c> --archive --json`",
		"separate documentation pull request",
	)
}

func TestSDDISOPins(t *testing.T) {
	t.Run("persistence contract keeps the general rule and declares the ISO exception", func(t *testing.T) {
		content := MustRead("skills/_shared/persistence-contract.md")
		requireContains(t, "persistence-contract.md", content,
			"Missing or failed verification never gates archive.",
			"The one exception is an ISO change",
			"must be valid and successful before archive",
			"review findings stay informational",
		)
	})

	t.Run("dependency graph makes verify mandatory in ISO", func(t *testing.T) {
		section := sddSharedSection(t, "Dependency Graph")
		requireContains(t, "Dependency Graph", section,
			"Verification is opt-in and may inspect partial work.",
			"In an ISO change (`iso.change: iso`), verify is mandatory before archive",
			"status routes to `verify` while the verification matrix is missing or invalid",
			"archive stays blocked until it is valid and successful",
			"Every other change keeps verification opt-in.",
		)
	})

	t.Run("kickoff section sends the ISO project to SDD and seals without a hand-made folder", func(t *testing.T) {
		section := sddSharedSection(t, "SDD Change Kickoff and Block Gates")
		requireContains(t, "SDD Change Kickoff and Block Gates", section,
			"the `iso` field of the status projects `project: iso`",
			"the project is ISO and only uses SDD",
			"without asking the lane question, also with `lane: ask`",
			"The seal creates the change folder and copies the project's `sdd.iso` into `kickoff.yaml`: never create the folder by hand beforehand.",
		)
	})

	t.Run("independent review keeps findings informational and only the matrix blocks", func(t *testing.T) {
		section := sddSharedSection(t, "SDD Independent Review")
		requireContains(t, "SDD Independent Review", section,
			"The one exception is an ISO change (`iso.change: iso`)",
			"there `verify` is mandatory before archive (status routes to `verify` while the matrix is missing or invalid)",
			"the verification matrix of the report does condition archive",
			"the block comes from the validity of the matrix and never from a finding, a `JUDGMENT: ESCALATED` verdict or a coverage review, which stay informational",
			// REQ-28.4 remite a la condición de REQ-21.15 sin repetirla.
			"is enabled by the condition that status projects (REQ-21.15)",
			// REQ-28.6: en ISO una matriz que falla hace que el verify no sea limpio.
			"counts as a failed required check: the verify is not clean",
		)
		if strings.Contains(section, "keeps waiting for every `role-apply` approval") {
			t.Error("SDD Independent Review repeats the verify condition of REQ-21.15 instead of referring to it")
		}
	})

	t.Run("no skill tells the agent to create the change folder by hand", func(t *testing.T) {
		for _, path := range []string{"skills/sdd-propose/SKILL.md", "skills/sdd-onboard/SKILL.md"} {
			content := MustRead(path)
			for _, banned := range []string{"create the change folder structure", "Create the change folder and write"} {
				if strings.Contains(content, banned) {
					t.Errorf("%s still tells the agent to create the change folder by hand: %q", path, banned)
				}
			}
			requireContains(t, path, content, "axiom sdd kickoff seal")
		}
	})

	// Paridad entre runtimes: todo orquestador que compone la sección de revisión
	// independiente (la lleva cada runtime) recibe la excepción ISO, la remisión del
	// proyecto ISO a SDD y la regla de sellado; el que además compone el grafo de
	// dependencias recibe el verify obligatorio. Las plantillas no cambian.
	t.Run("every orchestrator carries the ISO doctrine", func(t *testing.T) {
		composed := 0
		for _, path := range allSDDOrchestratorAssetPaths(t) {
			raw := MustRead(path)
			if !strings.Contains(raw, "{{GENTLE_AI_SDD_SECTION:SDD Independent Review}}") {
				continue
			}
			composed++
			t.Run(path, func(t *testing.T) {
				resolved := mustReadResolved(path)
				requireContains(t, path, resolved,
					"The one exception is an ISO change (`iso.change: iso`)",
					"there `verify` is mandatory before archive (status routes to `verify` while the matrix is missing or invalid)",
					"the project is ISO and only uses SDD",
					"never create the folder by hand beforehand",
				)
				if strings.Contains(raw, "{{GENTLE_AI_SDD_SECTION:Dependency Graph}}") {
					requireContains(t, path, resolved,
						"In an ISO change (`iso.change: iso`), verify is mandatory before archive",
						"Every other change keeps verification opt-in.",
					)
				}
			})
		}
		if composed == 0 {
			t.Fatal("no orchestrator composes the SDD Independent Review section")
		}
	})
}
