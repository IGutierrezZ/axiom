package assets

import (
	"strings"
	"testing"
)

// bateria-pruebas (C-06-c): pins de texto de la doctrina de la batería de pruebas de aceptación (`pruebas.md`).

func TestAcceptanceTestsDoctrineShared(t *testing.T) {
	shared := MustRead("skills/_shared/sdd-acceptance-tests.md")
	requireContains(t, "sdd-acceptance-tests.md", shared,
		"the SDD lane",
		"`acceptanceTests.active: true`",
		"An ODD change NEVER generates `pruebas.md`",
		"does not change this rule",
		"axiom sdd tests check --cwd",
		"MUST NOT widen the scope of `pruebas.md`",
		"refine paths or URLs inside `Pasos`",
		"Return `pruebas.md` in `artifacts`",
		"the result of the check",
		"declare its absence",
		"never as clean",
		"do not write the file and do not mention it",
	)
	for _, col := range []string{"`ID`", "`Cubre`", "`Rol`", "`Tipo`", "`Prioridad`", "`Entornos`", "`Precondiciones`", "`Pasos`", "`Resultado esperado`"} {
		requireContains(t, "sdd-acceptance-tests.md columnas", shared, col)
	}
	// Las cuatro capas, en su orden.
	last := -1
	for _, layer := range []string{"1. Mandatory traceability", "2. Equivalence partitions", "3. Pairwise", "4. Merge cases"} {
		i := strings.Index(shared, layer)
		if i < 0 {
			t.Fatalf("falta la capa %q", layer)
		}
		if i < last {
			t.Errorf("la capa %q está fuera de orden", layer)
		}
		last = i
	}
}

func TestAcceptanceTestsDoctrineSkillAndAgents(t *testing.T) {
	requireContains(t, "sdd-tasks skill", MustRead("skills/sdd-tasks/SKILL.md"),
		"### Step 3b: Acceptance test battery",
		"../_shared/sdd-acceptance-tests.md",
		"`acceptanceTests.active: true`",
		"Never in ODD",
	)
	for _, path := range []string{
		"claude/agents/sdd-tasks.md",
		"cursor/agents/sdd-tasks.md",
		"copilot/agents/sdd-tasks.agent.md",
	} {
		requireContains(t, path, MustRead(path),
			"## Acceptance Tests",
			"`acceptanceTests.active: true`",
			"`pruebas.md`",
			"axiom sdd tests check",
		)
	}
}
