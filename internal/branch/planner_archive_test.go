package branch

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas del plan de archivado (REQ-25.45, REQ-25.12, REQ-25.11, REQ-25.4, supuestos S-15
// y S-16): `Plan` con `Archive` es un plan aparte con solo la rama de archivado del
// repositorio del planning home, sin tocar las ramas del cambio ni las de rol. Tablas sin git
// ni disco, como las del resto del planificador.

// archiveInput es la entrada de `ensure --archive` en un monorepo embebido: con `Archive`,
// `Exceptions.Prefix` es el prefijo de la rama de archivado y `ChangeBranch`, la rama del
// cambio registrada.
func archiveInput(change string, sealed ...string) PlanInput {
	in := embeddedInput(change, sealed...)
	in.PlanningRepo = planningFacts(sealed...)
	in.Archive = true
	return in
}

func onlyArchiveBranch(t *testing.T, plan ChangePlan) (RepoPlan, BranchPlan) {
	t.Helper()
	if len(plan.Repositories) != 1 {
		t.Fatalf("el plan de archivado tiene un único repositorio, hay %d: %+v", len(plan.Repositories), plan.Repositories)
	}
	repo := plan.Repositories[0]
	if len(repo.Branches) != 1 {
		t.Fatalf("el plan de archivado tiene una única rama, hay %d: %+v", len(repo.Branches), repo.Branches)
	}
	return repo, repo.Branches[0]
}

// REQ-25.45 y REQ-25.11 (escenario «Ruta de la rama de archivado»): sin `--prefix`, la rama
// de archivado es `<rama del cambio registrada>-archive`, sale de la integración, su PR
// vuelve a la integración y su worktree está en `<raíz>/<cambio>-archive/<repo>`.
func TestPlanArchiveBranchDerivesFromTheRegisteredChangeBranch(t *testing.T) {
	tests := []struct {
		name         string
		changeBranch string
		want         string
	}{
		{"sin rama registrada rige la política", "", "feat/x-archive"},
		{"rama registrada con el prefijo de la política", "feat/x", "feat/x-archive"},
		{"rama registrada con --branch", "inc/inc-25-x", "inc/inc-25-x-archive"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := archiveInput("x", "core")
			in.ChangeBranch = tc.changeBranch
			plan := mustPlan(t, in)

			repo, archive := onlyArchiveBranch(t, plan)
			want := BranchPlan{
				Name:     tc.want,
				Kind:     KindArchive,
				PRTarget: "main",
				Worktree: filepath.Join(testWorktreeRoot, "x-archive", "axiom"),
			}
			if archive != want {
				t.Errorf("rama de archivado = %+v, se esperaba %+v", archive, want)
			}
			if repo.Role != RepoChangeBranch || repo.IntegrationBranch != "main" || repo.BaseRef != "" {
				t.Errorf("repositorio = %+v; se esperaba change_branch, integración main y la base de la integración (sin BaseRef)", repo)
			}
			if plan.Lane != LaneSDD || plan.Change != "x" || len(plan.Warnings) != 0 {
				t.Errorf("carril %q, cambio %q, avisos %v", plan.Lane, plan.Change, plan.Warnings)
			}
		})
	}
}

// REQ-25.12 (escenario «`--prefix` fija el prefijo de la rama de archivado»): con
// `--prefix p` la rama es `p/<cambio>-archive`, sin `plan_conflict` con el prefijo del cambio
// ni con el registrado, que no cambian.
func TestPlanArchivePrefixReplacesTheBranchPrefixWithoutConflictingWithTheChangeBranch(t *testing.T) {
	in := archiveInput("x", "core")
	in.ChangeBranch = "feat/x"
	in.Exceptions = Exceptions{Prefix: "doc"}

	plan := mustPlan(t, in)

	_, archive := onlyArchiveBranch(t, plan)
	if archive.Name != "doc/x-archive" || archive.Kind != KindArchive || archive.PRTarget != "main" {
		t.Errorf("rama = %+v, se esperaba doc/x-archive hacia main", archive)
	}

	t.Run("también con un prefijo de política distinto", func(t *testing.T) {
		in := archiveInput("x")
		in.Policy = ResolvePolicy(policyConfig("main", "feat"))
		in.ChangeBranch = "inc/x"
		in.Exceptions = Exceptions{Prefix: "docs"}
		if _, archive := onlyArchiveBranch(t, mustPlan(t, in)); archive.Name != "docs/x-archive" {
			t.Errorf("rama = %q, se esperaba docs/x-archive", archive.Name)
		}
	})
}

// REQ-25.3: sin rama registrada, el prefijo de la política (y el de un repositorio con
// ajustes propios) compone el nombre.
func TestPlanArchiveWithoutARegisteredBranchUsesTheConfiguredPolicy(t *testing.T) {
	in := archiveInput("x")
	in.Policy = ResolvePolicy(policyConfig("develop", "inc"))

	repo, archive := onlyArchiveBranch(t, mustPlan(t, in))

	if archive.Name != "inc/x-archive" || archive.PRTarget != "develop" || repo.IntegrationBranch != "develop" {
		t.Errorf("rama %q hacia %q con integración %q; se esperaba inc/x-archive hacia develop", archive.Name, archive.PRTarget, repo.IntegrationBranch)
	}

	t.Run("un valor inválido de la política es config_invalid", func(t *testing.T) {
		in := archiveInput("x")
		in.Policy = ResolvePolicy(policyConfig("main", "a b"))
		refusal := mustRefuse(t, in, CodeConfigInvalid)
		if refusal.Details["field"] != "branch_prefix" {
			t.Errorf("details = %v, se esperaba el campo branch_prefix", refusal.Details)
		}
	})
	t.Run("con --prefix el prefijo inválido de la política no se usa", func(t *testing.T) {
		in := archiveInput("x")
		in.Policy = ResolvePolicy(policyConfig("main", "a b"))
		in.Exceptions = Exceptions{Prefix: "doc"}
		if _, archive := onlyArchiveBranch(t, mustPlan(t, in)); archive.Name != "doc/x-archive" {
			t.Errorf("rama = %q, se esperaba doc/x-archive", archive.Name)
		}
	})
}

// REQ-25.12, supuesto S-16 (escenario «`--branch` y `--base` no se admiten con `--archive`»):
// con `--archive` solo se admite `--prefix`; las otras dos excepciones son `plan_conflict`
// y el rechazo nombra la bandera y la orden que sí se admite.
func TestPlanArchiveRejectsBranchAndBaseExceptionsWithPlanConflict(t *testing.T) {
	tests := []struct {
		name       string
		exceptions Exceptions
		wantFlags  []string
	}{
		{"--branch", Exceptions{Branch: "doc/x"}, []string{"--branch"}},
		{"--base", Exceptions{Base: "release/2"}, []string{"--base"}},
		{"las dos, con --prefix", Exceptions{Prefix: "doc", Branch: "doc/x", Base: "release/2"}, []string{"--branch", "--base"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := archiveInput("x", "core")
			in.Exceptions = tc.exceptions

			refusal := mustRefuse(t, in, CodePlanConflict)

			for _, flag := range tc.wantFlags {
				if !strings.Contains(refusal.Message, flag) {
					t.Errorf("el mensaje %q debe nombrar %s", refusal.Message, flag)
				}
			}
			if !strings.Contains(refusal.Message, "--prefix") {
				t.Errorf("el mensaje %q debe decir que solo se admite --prefix", refusal.Message)
			}
			for _, want := range []string{"axiom branch ensure --archive --change x", "--cwd " + pathquote.Quote(testMaster)} {
				if !strings.Contains(refusal.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
				}
			}
			for _, notWanted := range []string{"--branch", "--base"} {
				if strings.Contains(refusal.Continuation, notWanted) {
					t.Errorf("la continuación %q no debe repetir %s", refusal.Continuation, notWanted)
				}
			}
			if tc.exceptions.Prefix != "" && !strings.Contains(refusal.Continuation, "--prefix doc") {
				t.Errorf("la continuación %q debe conservar --prefix doc", refusal.Continuation)
			}
		})
	}
}

// REQ-25.45 (escenarios «Multirepo no tiene rama de archivado» y «El carril ODD no tiene
// rama de archivado»): `archive_branch_not_applicable`, con la orden que sí resuelve el
// archivado de esa topología, sin plan.
func TestPlanArchiveIsNotApplicableToMultirepoOrTheODDLane(t *testing.T) {
	t.Run("multirepo", func(t *testing.T) {
		in := archiveInput("x", "core")
		in.Topology = workspace.TopologyMultirepo
		refusal := mustRefuse(t, in, CodeArchiveBranchNotApplicable)
		for _, want := range []string{"multirepo", "repositorio de specs", "integración"} {
			if !strings.Contains(refusal.Message, want) {
				t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
			}
		}
		for _, want := range []string{"axiom sdd status x", pathquote.Quote(testMaster)} {
			if !strings.Contains(refusal.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
			}
		}
	})
	t.Run("carril ODD", func(t *testing.T) {
		in := archiveInput("mi-feature")
		in.Lane = LaneODD
		refusal := mustRefuse(t, in, CodeArchiveBranchNotApplicable)
		if !strings.Contains(refusal.Message, "ODD") {
			t.Errorf("el mensaje %q debe nombrar el carril ODD", refusal.Message)
		}
		if want := "axiom branch status --change mi-feature --lane odd --cwd " + pathquote.Quote(testMaster); !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	})
	t.Run("la aplicabilidad se decide antes que las banderas", func(t *testing.T) {
		in := archiveInput("x")
		in.Topology = workspace.TopologyMultirepo
		in.Exceptions = Exceptions{Branch: "doc/x"}
		mustRefuse(t, in, CodeArchiveBranchNotApplicable)
	})
}

// REQ-25.4 (escenario «Un rol llamado `archive` colisiona con la rama de archivado») y
// supuesto S-15: un rol `archive` junto a otro sobre el mismo repositorio es `invalid_name`
// y el rechazo nombra la colisión con `<prefijo>/<cambio>-archive`, también al planificar el
// archivado; un solo rol, que no genera rama de rol, no colisiona.
func TestPlanArchiveRejectsARoleNamedArchiveNamingTheCollision(t *testing.T) {
	for _, role := range []string{"archive", "Archive"} {
		t.Run(role, func(t *testing.T) {
			in := archiveInput("x", "core", role)
			in.ChangeBranch = "inc/x"

			refusal := mustRefuse(t, in, CodeInvalidName)

			if refusal.Details["field"] != "role" || refusal.Details["value"] != role {
				t.Errorf("details = %v, se esperaba el rol %q", refusal.Details, role)
			}
			for _, want := range []string{"«" + role + "»", "inc/x-archive"} {
				if !strings.Contains(refusal.Message, want) {
					t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
				}
			}
		})
	}
	t.Run("el plan ordinario lo rechaza igual", func(t *testing.T) {
		in := embeddedInput("x", "core", "archive")
		in.PlanningRepo = planningFacts("core", "archive")
		refusal := mustRefuse(t, in, CodeInvalidName)
		if !strings.Contains(refusal.Message, "feat/x-archive") {
			t.Errorf("el mensaje %q debe nombrar feat/x-archive", refusal.Message)
		}
	})
	t.Run("con --prefix el mensaje nombra la rama de archivado pedida", func(t *testing.T) {
		in := archiveInput("x", "core", "archive")
		in.Exceptions = Exceptions{Prefix: "doc"}
		refusal := mustRefuse(t, in, CodeInvalidName)
		if !strings.Contains(refusal.Message, "doc/x-archive") {
			t.Errorf("el mensaje %q debe nombrar doc/x-archive", refusal.Message)
		}
	})
	t.Run("un solo rol llamado archive no genera rama de rol", func(t *testing.T) {
		in := archiveInput("x", "archive")
		if _, archive := onlyArchiveBranch(t, mustPlan(t, in)); archive.Name != "feat/x-archive" {
			t.Errorf("rama = %q, se esperaba feat/x-archive", archive.Name)
		}
	})
}

// REQ-25.4 y REQ-25.7: el plan de archivado no toca las ramas del cambio ni las de rol: las
// del plan ordinario no cambian porque exista el de archivado, y planificar el archivado no
// altera la entrada.
func TestPlanArchiveLeavesTheChangeAndRoleBranchesUntouched(t *testing.T) {
	ordinary := embeddedInput("x", "core", "qa")
	ordinary.PlanningRepo = planningFacts("core", "qa")
	before := mustPlan(t, ordinary)

	in := ordinary
	in.Archive = true
	in.ChangeBranch = "feat/x"
	snapshot := in
	plan := mustPlan(t, in)

	if !reflect.DeepEqual(in, snapshot) {
		t.Errorf("Plan alteró su entrada:\nantes: %+v\ndespués: %+v", snapshot, in)
	}
	_, archive := onlyArchiveBranch(t, plan)
	if archive.Kind != KindArchive || archive.Name != "feat/x-archive" {
		t.Errorf("rama = %+v, se esperaba solo la rama de archivado feat/x-archive", archive)
	}
	if again := mustPlan(t, ordinary); !reflect.DeepEqual(again, before) {
		t.Errorf("el plan ordinario cambió:\nantes: %+v\ndespués: %+v", before, again)
	}
	wantNames(t, "ramas del plan ordinario", branchNames(before.Repositories[0]), "feat/x", "feat/x-core", "feat/x-qa")
	if before.Repositories[0].Branches[1].Kind == KindArchive {
		t.Error("el plan ordinario no incluye la rama de archivado")
	}
}

// REQ-25.45 y REQ-25.8: en `in_place` no hay worktree que colocar; el plan lleva la rama con
// la ruta vacía.
func TestPlanArchiveInPlaceHasNoWorktree(t *testing.T) {
	in := archiveInput("x")
	in.InPlace = true

	_, archive := onlyArchiveBranch(t, mustPlan(t, in))

	if archive.Worktree != "" || archive.Name != "feat/x-archive" {
		t.Errorf("rama = %+v, se esperaba feat/x-archive sin worktree", archive)
	}
}

// REQ-25.11: el workspaceRoot y el planning home del plan de archivado son los del worktree
// de la rama de archivado, donde se ejecuta el archive.
func TestPlanArchiveLocatesTheWorkspaceInsideTheArchiveWorktree(t *testing.T) {
	plan := mustPlan(t, archiveInput("x"))

	worktree := filepath.Join(testWorktreeRoot, "x-archive", "axiom")
	if plan.WorkspaceRoot != worktree || plan.PlanningHome != filepath.Join(worktree, "openspec") {
		t.Errorf("workspaceRoot %q y planningHome %q; se esperaba el worktree de archivado %q", plan.WorkspaceRoot, plan.PlanningHome, worktree)
	}
}

// REQ-25.3, REQ-25.11: nombres inválidos, longitud de ruta y colocación se rechazan como en el
// plan ordinario, antes de crear nada.
func TestPlanArchiveValidatesNamesLengthAndPlacement(t *testing.T) {
	t.Run("cambio inválido", func(t *testing.T) {
		mustRefuse(t, archiveInput("a b:c"), CodeInvalidName)
	})
	t.Run("--prefix inválido", func(t *testing.T) {
		in := archiveInput("x")
		in.Exceptions = Exceptions{Prefix: "a b"}
		refusal := mustRefuse(t, in, CodeInvalidName)
		if refusal.Details["field"] != "--prefix" {
			t.Errorf("details = %v, se esperaba --prefix", refusal.Details)
		}
	})
	t.Run("ruta de 221 caracteres en Windows", func(t *testing.T) {
		in := archiveInput("x")
		in.PathRules = PathRules{GOOS: "windows"}
		// `<raíz>/x-archive/axiom`: se ajusta la raíz para que la ruta mida exactamente 221.
		base := filepath.Join("x-archive", "axiom")
		root := `C:\` + strings.Repeat("r", 221-len(`C:\`)-1-len(base))
		in.WorktreeRoot = root
		if length := pathLength(filepath.Join(root, base)); length != 221 {
			t.Fatalf("la preparación debía dar 221 caracteres, dio %d", length)
		}
		refusal := mustRefuse(t, in, CodePathTooLong)
		if refusal.Details["length"] != "221" || refusal.Details["max"] != "220" {
			t.Errorf("details = %v, se esperaba longitud 221 y máximo 220", refusal.Details)
		}
		// Con 220 exactos se admite.
		in.WorktreeRoot = root[:len(root)-1]
		mustPlan(t, in)
	})
	t.Run("ruta dentro de un worktree existente que no es el suyo", func(t *testing.T) {
		in := archiveInput("x")
		in.ExistingWorktrees = []WorktreeEntry{{Path: filepath.Join(testWorktreeRoot, "x-archive")}}
		mustRefuse(t, in, CodePlanConflict)
	})
	t.Run("el worktree que ya está en su ruta es el suyo", func(t *testing.T) {
		in := archiveInput("x")
		in.ExistingWorktrees = []WorktreeEntry{{Path: filepath.Join(testWorktreeRoot, "x-archive", "axiom"), Branch: "feat/x-archive"}}
		mustPlan(t, in)
	})
}

// REQ-25.12 y design D-25: una rama de archivado que el registro ya guarda no cambia de nombre.
// Sin `--prefix` el plan la conserva aunque la rama del cambio o la política hayan cambiado; con
// un `--prefix` que da el mismo nombre es idempotente, y con uno que daría otro, `plan_conflict`
// que nombra las dos ramas y vuelve a la registrada sin `--prefix`.
func TestPlanArchiveKeepsTheRegisteredArchiveBranchAndRejectsAPrefixThatWouldRenameIt(t *testing.T) {
	registered := func() PlanInput {
		in := archiveInput("x", "core")
		in.ChangeBranch = "inc/x"
		in.ArchiveBranch = "feat/x-archive"
		return in
	}
	t.Run("sin --prefix se conserva el nombre registrado", func(t *testing.T) {
		if _, archive := onlyArchiveBranch(t, mustPlan(t, registered())); archive.Name != "feat/x-archive" {
			t.Errorf("rama = %q, se esperaba la registrada feat/x-archive y no inc/x-archive", archive.Name)
		}
	})
	t.Run("el mismo prefijo es idempotente", func(t *testing.T) {
		in := registered()
		in.ArchiveBranch = "doc/x-archive"
		in.Exceptions = Exceptions{Prefix: "doc"}
		if _, archive := onlyArchiveBranch(t, mustPlan(t, in)); archive.Name != "doc/x-archive" {
			t.Errorf("rama = %q, se esperaba doc/x-archive", archive.Name)
		}
	})
	t.Run("otro prefijo es plan_conflict", func(t *testing.T) {
		in := registered()
		in.Exceptions = Exceptions{Prefix: "doc"}

		refusal := mustRefuse(t, in, CodePlanConflict)

		for _, want := range []string{"feat/x-archive", "doc/x-archive"} {
			if !strings.Contains(refusal.Message, want) {
				t.Errorf("el mensaje %q debe nombrar %s", refusal.Message, want)
			}
		}
		for _, want := range []string{
			"axiom branch ensure --archive --change x --cwd " + pathquote.Quote(testMaster),
			"git -C " + pathquote.Quote(testMaster) + " branch -D feat/x-archive",
		} {
			if !strings.Contains(refusal.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
			}
		}
		if refusal.Details["registered"] != "feat/x-archive" || refusal.Details["planned"] != "doc/x-archive" {
			t.Errorf("details = %v", refusal.Details)
		}
	})
}

// REQ-25.45 y REQ-25.10: `change_not_integrated` nombra la base evaluada y la carpeta que falta,
// sugiere `--fetch` con una orden ejecutable que conserva el `--prefix`, y, con la base ya traída
// con `--fetch`, dice que falta fusionar el PR; si la base contiene el cambio archivado (G-13)
// nombra la entrada y dice que no hay nada que archivar.
func TestChangeNotIntegratedNamesTheBaseTheFolderAndAnExecutableContinuation(t *testing.T) {
	repo := RepoPlan{Name: "axiom", GitRoot: testMaster, IntegrationBranch: "main"}
	quoted := pathquote.Quote(testMaster)

	t.Run("sin fetch sugiere --fetch", func(t *testing.T) {
		refusal := ChangeNotIntegrated("x", testMaster, "", repo, "origin/main", "openspec/changes/x/", "", false)

		if refusal.Code != CodeChangeNotIntegrated {
			t.Fatalf("código = %s", refusal.Code)
		}
		for _, want := range []string{"origin/main", "openspec/changes/x/", "no se ha creado ninguna rama ni worktree"} {
			if !strings.Contains(refusal.Message, want) {
				t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
			}
		}
		if want := "axiom branch ensure --cwd " + quoted + " --change x --archive --fetch"; !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	})
	t.Run("la continuación conserva el prefijo", func(t *testing.T) {
		refusal := ChangeNotIntegrated("x", testMaster, "doc", repo, "main", "openspec/changes/x/", "", false)
		if want := "--change x --archive --fetch --prefix doc"; !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	})
	t.Run("tras el fetch falta fusionar el PR", func(t *testing.T) {
		refusal := ChangeNotIntegrated("x", testMaster, "", repo, "origin/main", "openspec/changes/x/", "", true)
		if !strings.Contains(refusal.Message, "tras el --fetch") || !strings.Contains(refusal.Continuation, "fusiona el PR del cambio en main") {
			t.Errorf("mensaje %q, continuación %q", refusal.Message, refusal.Continuation)
		}
	})
	t.Run("cambio ya archivado", func(t *testing.T) {
		entry := "openspec/changes/archive/2026-10-01-x/"
		refusal := ChangeNotIntegrated("x", testMaster, "", repo, "main", "openspec/changes/x/", entry, false)

		if refusal.Code != CodeChangeNotIntegrated || refusal.Details["archivedEntry"] != entry {
			t.Fatalf("código %s, details %v", refusal.Code, refusal.Details)
		}
		for _, want := range []string{entry, "nada que archivar"} {
			if !strings.Contains(refusal.Message, want) {
				t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
			}
		}
		if want := "axiom branch status --change x --cwd " + quoted; !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	})
}
