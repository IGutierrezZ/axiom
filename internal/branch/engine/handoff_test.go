package engine

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de `Engine.HandOff` (REQ-26.4, REQ-26.5, DD-03, DD-07 y DD-08): con git real, entorno
// aislado y sin red. Cubren la comprobación de «confirmado», las entregas por topología y que el
// cálculo es de solo lectura.

const handOffGates = "openspec/changes/x/gates.yaml"

// handOffMonorepo crea un monorepo embebido con el kickoff de x sellado y confirmado en main, y
// ejecuta `ensure` para tener la rama del cambio (y las de rol, con dos o más) desplegadas. Devuelve
// el checkout principal.
func handOffMonorepo(t *testing.T, roles ...string) string {
	t.Helper()
	declared := map[string][]string{}
	for _, role := range roles {
		declared[role] = []string{"."}
	}
	repo := newProject(t, configWithRoles(declared, ""))
	sealChange(t, repo, "x", roles...)
	runGit(t, repo, "add", "openspec")
	commit(t, repo, "kickoff de x")
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	return repo
}

// handOffRequest es la petición de la salida 2 de la puerta dada, desde el worktree dado.
func handOffRequest(worktree, gate, role, handedOffTo string) branch.HandOffRequest {
	return branch.HandOffRequest{
		CWD:         worktree,
		Change:      "x",
		ChangeRoot:  filepath.Join(worktree, "openspec", "changes", "x"),
		Role:        role,
		Gate:        gate,
		HandedOffTo: handedOffTo,
		Exempt:      []string{"gates.yaml"},
	}
}

func handOff(t *testing.T, request branch.HandOffRequest) (branch.HandOffPlan, *branch.Refusal) {
	t.Helper()
	return Default().HandOff(context.Background(), request)
}

func wantUncommitted(t *testing.T, refusal *branch.Refusal, wantPath string) {
	t.Helper()
	if refusal == nil {
		t.Fatalf("se esperaba el rechazo uncommitted_artifacts por %q y la entrega se calculó", wantPath)
	}
	if refusal.Code != branch.CodeUncommittedArtifacts {
		t.Fatalf("código %q (%s), se esperaba uncommitted_artifacts", refusal.Code, refusal.Message)
	}
	if !strings.Contains(refusal.Message, wantPath) {
		t.Errorf("el mensaje %q debe nombrar %q", refusal.Message, wantPath)
	}
	if !strings.Contains(refusal.Continuation, "git -C") || !strings.Contains(refusal.Continuation, wantPath) {
		t.Errorf("la continuación %q debe ser un commit ejecutable de %q", refusal.Continuation, wantPath)
	}
}

// DD-03 y matriz «Estado del commit»: lo preparado sin confirmar, lo que no tiene seguimiento y una
// carpeta entera sin seguimiento impiden la entrega; `gates.yaml` y los ficheros de estado con punto
// de la raíz de la carpeta no cuentan.
func TestHandOffRefusesUncommittedArtifactsOfTheChangeFolder(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	folder := filepath.Join(worktree, "openspec", "changes", "x")

	t.Run("el worktree limpio pasa", func(t *testing.T) {
		if _, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design")); refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
	})
	t.Run("solo gates.yaml y los ficheros de estado con punto sucios pasan", func(t *testing.T) {
		writeFile(t, filepath.Join(folder, "gates.yaml"), "schema: axiom.sdd-gate-ledger/v1\n")
		writeFile(t, filepath.Join(folder, ".gates.lock"), "")
		writeFile(t, filepath.Join(folder, ".axiom-instance"), "id\n")
		writeFile(t, filepath.Join(folder, ".gates-123.yaml"), "x\n")
		t.Cleanup(func() { resetTree(t, worktree) })
		if _, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design")); refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
	})
	t.Run("un fichero preparado sin confirmar se rechaza", func(t *testing.T) {
		writeFile(t, filepath.Join(folder, "proposal.md"), "propuesta\n")
		runGit(t, worktree, "add", "openspec/changes/x/proposal.md")
		t.Cleanup(func() { resetTree(t, worktree) })
		_, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design"))
		wantUncommitted(t, refusal, "openspec/changes/x/proposal.md")
	})
	t.Run("un fichero sin seguimiento se rechaza", func(t *testing.T) {
		writeFile(t, filepath.Join(folder, "design.md"), "diseño\n")
		t.Cleanup(func() { resetTree(t, worktree) })
		_, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design"))
		wantUncommitted(t, refusal, "openspec/changes/x/design.md")
	})
	t.Run("un fichero modificado se rechaza aunque gates.yaml también lo esté", func(t *testing.T) {
		writeFile(t, filepath.Join(folder, "proposal.md"), "propuesta\n")
		runGit(t, worktree, "add", "openspec/changes/x/proposal.md")
		commit(t, worktree, "propuesta")
		t.Cleanup(func() { runGit(t, worktree, "reset", "-q", "--hard", "HEAD~1") })
		writeFile(t, filepath.Join(folder, "gates.yaml"), "schema: axiom.sdd-gate-ledger/v1\n")
		writeFile(t, filepath.Join(folder, "proposal.md"), "propuesta editada\n")
		t.Cleanup(func() { resetTree(t, worktree) })
		_, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design"))
		wantUncommitted(t, refusal, "openspec/changes/x/proposal.md")
		if strings.Contains(refusal.Message, "gates.yaml") {
			t.Errorf("gates.yaml está exento y no debe figurar: %s", refusal.Message)
		}
	})
	t.Run("lo sucio fuera de la carpeta del cambio no impide spec, design ni tasks", func(t *testing.T) {
		writeFile(t, filepath.Join(worktree, "otro.txt"), "x\n")
		t.Cleanup(func() { resetTree(t, worktree) })
		if _, refusal := handOff(t, handOffRequest(worktree, "design", "", "tasks")); refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
	})
}

func TestHandOffRefusesAnEntirelyUntrackedChangeFolderWithOneEntry(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	// Otro cambio versionado evita que git agrupe toda la carpeta `openspec/changes/`.
	writeFile(t, filepath.Join(worktree, "openspec", "changes", "y", "proposal.md"), "otro\n")
	runGit(t, worktree, "add", "openspec/changes/y/proposal.md")
	commit(t, worktree, "otro cambio")
	runGit(t, worktree, "rm", "-r", "-q", "--cached", "openspec/changes/x")
	commit(t, worktree, "saca x del índice")

	_, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design"))

	wantUncommitted(t, refusal, "openspec/changes/x/")
	if refusal.Details["paths"] != "openspec/changes/x/" {
		t.Errorf("la carpeta entera cuenta como una sola entrada: %q", refusal.Details["paths"])
	}
}

// REQ-26.4 con role-apply: el árbol de trabajo entero de cada repositorio del rol debe estar
// confirmado, no solo la carpeta del cambio.
func TestHandOffOfARoleApplyGateRequiresTheWholeWorkingTreeOfTheRoleRepository(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))

	t.Run("trabajo del rol sin confirmar", func(t *testing.T) {
		writeFile(t, filepath.Join(worktree, "internal", "core.go"), "package core\n")
		t.Cleanup(func() { resetTree(t, worktree) })
		_, refusal := handOff(t, handOffRequest(worktree, "role-apply:core", "core", "verify"))
		wantUncommitted(t, refusal, "internal/")
	})
	t.Run("con el árbol limpio pasa", func(t *testing.T) {
		if _, refusal := handOff(t, handOffRequest(worktree, "role-apply:core", "core", "verify")); refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
	})
}

// A-4 y DD-07: en un monorepo la rama del cambio apunta a la integración; una rama de rol, a la rama
// de seguimiento, sin afirmar integración. Los pasos usan `git -C <raíz git>` absoluta (también con el
// proceso en otro directorio), pathspec explícito, la rama concreta y ningún `&&` ni orden de
// plataforma.
func TestHandOffDeliversTheChangeBranchToTheIntegrationAndARoleBranchToTheTracking(t *testing.T) {
	repo := handOffMonorepo(t, "core", "web")
	base := filepath.Base(repo)

	t.Run("rama del cambio", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "x", base)
		plan, refusal := handOff(t, handOffRequest(worktree, "design", "", "tasks"))
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		if plan.Topology != workspace.TopologyMonorepoEmbedded || !plan.MergeRequired || len(plan.Deliveries) != 1 {
			t.Fatalf("plan = %+v", plan)
		}
		delivery := plan.Deliveries[0]
		wantSameDirectory(t, "gitRoot", delivery.GitRoot, worktree)
		if delivery.Branch != "feat/x" || delivery.PRTarget != "main" || delivery.InPlace {
			t.Errorf("entrega = %+v, se esperaba feat/x hacia main", delivery)
		}
		if want := (&branch.HandOffPR{Head: "feat/x", Base: "main"}); !reflect.DeepEqual(delivery.PR, want) {
			t.Errorf("pr = %+v, se esperaba %+v", delivery.PR, want)
		}
		root := pathquote.Quote(delivery.GitRoot)
		want := []string{
			"git -C " + root + " add -- " + handOffGates,
			"git -C " + root + ` commit -m "chore(sdd): aprobar design de x y ceder tasks" -- ` + handOffGates,
			"git -C " + root + " push -u origin feat/x",
		}
		if !reflect.DeepEqual(delivery.Steps, want) {
			t.Errorf("pasos:\n%q\nse esperaba:\n%q", delivery.Steps, want)
		}
		wantNoCompoundOrPlatformCommands(t, delivery.Steps)
	})

	t.Run("rama de rol con el proceso en otro directorio", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "x-core", base)
		plan, refusal := handOff(t, handOffRequest(worktree, "role-apply:core", "core", "apply:web"))
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		delivery := plan.Deliveries[0]
		wantSameDirectory(t, "gitRoot", delivery.GitRoot, worktree)
		if delivery.Branch != "feat/x-core" || delivery.PRTarget != "feat/x" {
			t.Errorf("entrega = %+v, se esperaba la rama de rol hacia la de seguimiento", delivery)
		}
		if want := (&branch.HandOffPR{Head: "feat/x-core", Base: "feat/x"}); !reflect.DeepEqual(delivery.PR, want) {
			t.Errorf("pr = %+v, se esperaba %+v", delivery.PR, want)
		}
		if last := delivery.Steps[len(delivery.Steps)-1]; last != "git -C "+pathquote.Quote(delivery.GitRoot)+" push -u origin feat/x-core" {
			t.Errorf("el push debe nombrar la rama concreta: %q", last)
		}
	})
}

func TestHandOffQuotesAGitRootWithSpaces(t *testing.T) {
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	spaced := filepath.Join(t.TempDir(), "carpeta con espacios")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", spaced, "main")
	sealChangeAt(t, spaced, "x", "core")
	runGit(t, spaced, "add", "openspec")
	commit(t, spaced, "kickoff")

	plan, refusal := handOff(t, handOffRequest(spaced, "spec", "", "design"))

	if refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	for _, step := range plan.Deliveries[0].Steps {
		if !strings.Contains(step, pathquote.Quote(plan.Deliveries[0].GitRoot)) {
			t.Errorf("el paso %q debe citar la raíz git con espacios", step)
		}
	}
}

// Matriz «Selección de repositorio git» y «Estado del push»: el cálculo no ejecuta ninguna
// operación remota ni `commit`.
func TestHandOffNeverRunsRemoteOperationsOrCommit(t *testing.T) {
	repo := handOffMonorepo(t, "core")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	seen := observe(t)

	if _, refusal := handOff(t, handOffRequest(worktree, "spec", "", "design")); refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}

	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) || invocation[1] == "commit" || invocation[1] == "add" {
			t.Errorf("HandOff lanzó %v", invocation)
		}
	}
}

// REQ-26.5, segundo escenario: el repositorio de specs de un multirepo se trabaja in situ; la
// entrega es un commit y el push de la rama de integración, sin rama de cambio ni PR.
func TestHandOffOfTheSpecsRepositoryOfAMultirepoIsInPlaceWithoutAPR(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	runGit(t, specs, "add", "-A")
	commit(t, specs, "kickoff de x")
	request := branch.HandOffRequest{
		CWD:         master,
		Change:      "x",
		ChangeRoot:  filepath.Join(specs, "openspec", "changes", "x"),
		Gate:        "spec",
		HandedOffTo: "design",
		Exempt:      []string{"gates.yaml"},
	}

	plan, refusal := handOff(t, request)

	if refusal != nil {
		t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
	}
	if plan.Topology != workspace.TopologyMultirepo || !plan.MergeRequired || len(plan.Deliveries) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	delivery := plan.Deliveries[0]
	wantSameDirectory(t, "gitRoot", delivery.GitRoot, specs)
	if !delivery.InPlace || delivery.Branch != "main" || delivery.PRTarget != "" || delivery.PR != nil {
		t.Errorf("entrega = %+v, se esperaba in situ en main, sin PR ni destino", delivery)
	}
	root := pathquote.Quote(delivery.GitRoot)
	want := []string{
		"git -C " + root + " add -- " + handOffGates,
		"git -C " + root + ` commit -m "chore(sdd): aprobar spec de x y ceder design" -- ` + handOffGates,
		"git -C " + root + " push origin main",
	}
	if !reflect.DeepEqual(delivery.Steps, want) {
		t.Errorf("pasos:\n%q\nse esperaba:\n%q", delivery.Steps, want)
	}
	for _, step := range delivery.Steps {
		if strings.Contains(step, "feat/") {
			t.Errorf("el repositorio de specs no tiene rama de cambio: %q", step)
		}
	}

	t.Run("un fichero de la carpeta sin confirmar se rechaza", func(t *testing.T) {
		writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "spec.md"), "spec\n")
		t.Cleanup(func() { resetTree(t, specs) })
		_, refusal := handOff(t, request)
		wantUncommitted(t, refusal, "openspec/changes/x/spec.md")
	})
	t.Run("fuera de la integración no se presenta un push de la rama equivocada", func(t *testing.T) {
		runGit(t, specs, "switch", "-q", "-c", "otra")
		t.Cleanup(func() { runGit(t, specs, "switch", "-q", "main") })
		_, refusal := handOff(t, request)
		if refusal == nil || refusal.Code != branch.CodeWrongBranch || refusal.Continuation == "" {
			t.Fatalf("rechazo = %+v, se esperaba wrong_branch con continuación", refusal)
		}
	})
}

// DD-07, role-apply en un multirepo: la entrega del repositorio de specs más la de cada repositorio
// de código del rol, con su propia raíz git y su propio destino; el trabajo del rol sin confirmar en
// el repositorio de código lo impide.
func TestHandOffOfARoleApplyGateInAMultirepoDeliversEveryRoleRepository(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	runGit(t, specs, "add", "-A")
	commit(t, specs, "kickoff de x")
	wantReady(t, ensureInPlace(t, master, "x"))
	backend := filepath.Join(master, "repo-backend")
	request := branch.HandOffRequest{
		CWD:         master,
		Change:      "x",
		ChangeRoot:  filepath.Join(specs, "openspec", "changes", "x"),
		Role:        "backend",
		Gate:        "role-apply:backend",
		HandedOffTo: "verify",
		Exempt:      []string{"gates.yaml"},
	}

	t.Run("trabajo del rol sin confirmar en el repositorio de código", func(t *testing.T) {
		writeFile(t, filepath.Join(backend, "api.go"), "package api\n")
		t.Cleanup(func() { resetTree(t, backend) })
		_, refusal := handOff(t, request)
		wantUncommitted(t, refusal, "api.go")
		if !strings.Contains(refusal.Continuation, pathquote.Quote(refusal.Details["repository"])) {
			t.Errorf("la continuación %q debe usar la raíz git del repositorio de código", refusal.Continuation)
		}
	})
	t.Run("con todo confirmado hay una entrega por repositorio", func(t *testing.T) {
		plan, refusal := handOff(t, request)
		if refusal != nil {
			t.Fatalf("rechazo %q: %s", refusal.Code, refusal.Message)
		}
		if len(plan.Deliveries) != 2 {
			t.Fatalf("entregas = %+v, se esperaban la de specs y la de repo-backend", plan.Deliveries)
		}
		specsDelivery, code := plan.Deliveries[0], plan.Deliveries[1]
		if !specsDelivery.InPlace || specsDelivery.Repository != "repo-specs" {
			t.Errorf("la primera entrega es la de specs in situ: %+v", specsDelivery)
		}
		wantSameDirectory(t, "gitRoot de repo-backend", code.GitRoot, backend)
		if code.Branch != "feat/x" || code.PRTarget != "main" || code.InPlace {
			t.Errorf("entrega de código = %+v", code)
		}
		if want := []string{"git -C " + pathquote.Quote(code.GitRoot) + " push -u origin feat/x"}; !reflect.DeepEqual(code.Steps, want) {
			t.Errorf("pasos del código = %q, se esperaba %q (sin commit: el árbol ya está limpio)", code.Steps, want)
		}
		if want := (&branch.HandOffPR{Head: "feat/x", Base: "main"}); !reflect.DeepEqual(code.PR, want) {
			t.Errorf("pr = %+v", code.PR)
		}
	})
}

func wantNoCompoundOrPlatformCommands(t *testing.T, steps []string) {
	t.Helper()
	for _, step := range steps {
		if strings.Contains(step, "&&") || strings.Contains(step, "gh ") || strings.Contains(step, "glab ") {
			t.Errorf("el paso %q no puede ser compuesto ni llamar a la plataforma", step)
		}
		if strings.Contains(step, " -a ") || strings.Contains(step, " HEAD") {
			t.Errorf("el paso %q no puede usar `-a` ni `HEAD`", step)
		}
	}
}
