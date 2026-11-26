package engine

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Pruebas de la rama de archivado en `status` (RF-BRA-16 escenario «La rama de archivado
// figura en el plan», RF-BRA-04 escenario «La rama de archivado no pertenece al modelo de
// roles», RF-BRA-13), con git real y sin red. La rama de archivado figura cuando existe, con
// los mismos datos que las demás y el destino de su PR en la rama de integración.

// fullstackProject es un monorepo con el rol único `fullstack` sellado en el cambio `x`: en
// `monorepo-embedded` no existen ramas de rol (RF-TOP-04 de `workspace-topology`).
func fullstackProject(t *testing.T) string {
	t.Helper()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))
	sealChange(t, repo, "x", "fullstack")
	return repo
}

func branchNamesOf(repository branch.RepositoryResult) []string {
	var names []string
	for _, described := range repository.Branches {
		names = append(names, described.Name)
	}
	return names
}

// RF-BRA-16: con la rama de archivado y su worktree, `status` la incluye con la ruta absoluta
// de su worktree, su tipo (con etiqueta), su estado respecto de la base y el destino de su PR,
// que es la integración; la rama del cambio sigue como estaba.
func TestStatusIncludesTheArchiveBranchWithItsWorktreeAndTheIntegrationAsPRTarget(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	runGit(t, repo, "branch", "feat/x", "main")
	worktree := filepath.Join(branch.DefaultWorktreeRoot(canonicalDirectory(t, repo)), "x-archive", filepath.Base(repo))
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x-archive", worktree, "main")
	commit(t, worktree, "documentación del archivado")

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	repository := onlyRepository(t, result)
	if want := []string{"feat/x", "feat/x-archive"}; !reflect.DeepEqual(branchNamesOf(repository), want) {
		t.Fatalf("ramas = %v, se esperaba %v (la del cambio y después la de archivado)", branchNamesOf(repository), want)
	}
	archive := branchByName(t, repository, "feat/x-archive")
	if archive.Kind != branch.KindArchive || archive.KindLabel != "rama de archivado" || !archive.Exists || archive.Dirty || archive.Role != "" {
		t.Errorf("rama de archivado = %+v", archive)
	}
	wantSameDirectory(t, "worktree de feat/x-archive", archive.Worktree, worktree)
	if archive.PRTarget != "main" {
		t.Errorf("prTarget = %q, se esperaba la rama de integración main", archive.PRTarget)
	}
	if archive.ComparedTo != "main" || intValue(t, "adelanto", archive.Ahead) != 1 || intValue(t, "atraso", archive.Behind) != 0 {
		t.Errorf("comparada con %q: adelanto %v, atraso %v; se esperaba main con 1 de adelanto", archive.ComparedTo, archive.Ahead, archive.Behind)
	}
	if change := branchByName(t, repository, "feat/x"); change.Kind != branch.KindChange || change.PRTarget != "main" || change.Worktree != "" {
		t.Errorf("la rama del cambio no cambia por existir la de archivado: %+v", change)
	}
}

// RF-BRA-04, escenario «La rama de archivado no pertenece al modelo de roles»: con el rol único
// `fullstack` de un monorepo (sin ramas de rol), el destino del PR de `feat/x` es la integración,
// el de la rama de archivado `feat/x-archive` también y el plan no contiene ninguna rama de rol.
func TestStatusArchiveBranchDoesNotBelongToTheRoleModel(t *testing.T) {
	t.Parallel()
	repo := fullstackProject(t)
	for _, name := range []string{"feat/x", "feat/x-archive"} {
		runGit(t, repo, "branch", name, "main")
	}

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	repository := onlyRepository(t, result)
	if want := []string{"feat/x", "feat/x-archive"}; !reflect.DeepEqual(branchNamesOf(repository), want) {
		t.Fatalf("ramas = %v, se esperaba %v", branchNamesOf(repository), want)
	}
	for _, described := range repository.Branches {
		if described.Kind == branch.KindRole {
			t.Errorf("un monorepo con fullstack no tiene ramas de rol: %+v", described)
		}
	}
	for name, wantTarget := range map[string]string{"feat/x": "main", "feat/x-archive": "main"} {
		if got := branchByName(t, repository, name).PRTarget; got != wantTarget {
			t.Errorf("prTarget de %s = %q, se esperaba %q", name, got, wantTarget)
		}
	}
	if archive := branchByName(t, repository, "feat/x-archive"); archive.Kind != branch.KindArchive || archive.Role != "" {
		t.Errorf("la rama de archivado no es de rol: %+v", archive)
	}
}

// RF-BRA-16: sin rama de archivado, `status` no la inventa: ni se lista ni cuenta para que el
// cambio esté iniciado.
func TestStatusWithoutAnArchiveBranchDoesNotListOne(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	t.Run("cambio sin ramas", func(t *testing.T) {
		result := status(t, repo, "x", branch.LaneSDD)

		if result.Status != branch.StatusNotStarted {
			t.Fatalf("status %q, se esperaba not_started", result.Status)
		}
		if want := []string{"feat/x"}; !reflect.DeepEqual(branchNamesOf(onlyRepository(t, result)), want) {
			t.Errorf("ramas = %v, se esperaba solo %v", branchNamesOf(onlyRepository(t, result)), want)
		}
	})
	t.Run("cambio con su rama", func(t *testing.T) {
		runGit(t, repo, "branch", "feat/x", "main")

		result := status(t, repo, "x", branch.LaneSDD)

		wantReady(t, result)
		if want := []string{"feat/x"}; !reflect.DeepEqual(branchNamesOf(onlyRepository(t, result)), want) {
			t.Errorf("ramas = %v, se esperaba solo %v", branchNamesOf(onlyRepository(t, result)), want)
		}
	})
}

// RF-BRA-12 y RF-BRA-13: la rama de archivado deriva de la rama del cambio registrada
// (`--branch inc/inc-25-x` da `inc/inc-25-x-archive`) y, con un prefijo de archivado
// registrado, de él (`doc/x-archive`), sin `plan_conflict` con el prefijo del cambio.
func TestStatusNamesTheArchiveBranchFromTheRegistry(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	for _, name := range []string{"inc/inc-25-x", "inc/inc-25-x-archive", "doc/x-archive", "feat/x-archive"} {
		runGit(t, repo, "branch", name, "main")
	}

	t.Run("deriva de la rama del cambio registrada", func(t *testing.T) {
		record := registeredRecord(repo, branch.RecordBranch{Name: "inc/inc-25-x", Kind: branch.KindChange})
		record.Exceptions = branch.Exceptions{Branch: "inc/inc-25-x"}
		writeRegistry(t, commonDir(repo), record)

		repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

		if want := []string{"inc/inc-25-x", "inc/inc-25-x-archive"}; !reflect.DeepEqual(branchNamesOf(repository), want) {
			t.Errorf("ramas = %v, se esperaba %v", branchNamesOf(repository), want)
		}
	})
	t.Run("deriva del prefijo de archivado registrado", func(t *testing.T) {
		record := registeredRecord(repo,
			branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange},
			branch.RecordBranch{Name: "doc/x-archive", Kind: branch.KindArchive})
		record.ArchivePrefix = "doc"
		writeRegistry(t, commonDir(repo), record)

		repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

		if want := []string{"feat/x", "doc/x-archive"}; !reflect.DeepEqual(branchNamesOf(repository), want) {
			t.Errorf("ramas = %v, se esperaba %v", branchNamesOf(repository), want)
		}
		if archive := branchByName(t, repository, "doc/x-archive"); archive.Kind != branch.KindArchive || archive.PRTarget != "main" {
			t.Errorf("rama de archivado = %+v", archive)
		}
	})
	t.Run("una rama de archivado registrada que ya no existe se informa como inexistente", func(t *testing.T) {
		runGit(t, repo, "branch", "-D", "doc/x-archive")
		record := registeredRecord(repo,
			branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange},
			branch.RecordBranch{Name: "doc/x-archive", Kind: branch.KindArchive})
		record.ArchivePrefix = "doc"
		writeRegistry(t, commonDir(repo), record)

		repository := onlyRepository(t, status(t, repo, "x", branch.LaneSDD))

		if archive := branchByName(t, repository, "doc/x-archive"); archive.Exists {
			t.Errorf("la rama ya no existe en git y status no debe decir que existe: %+v", archive)
		}
	})
}

// RF-BRA-13: el carril ODD y el multirepo no tienen rama de archivado: `status` no la muestra
// aunque exista una rama con ese nombre.
func TestStatusNeverListsAnArchiveBranchForTheODDLaneOrAMultirepo(t *testing.T) {
	t.Parallel()
	t.Run("carril ODD", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		runGit(t, repo, "branch", "feat/mi-feature", "main")
		runGit(t, repo, "branch", "feat/mi-feature-archive", "main")

		result := status(t, repo, "mi-feature", branch.LaneODD)

		wantReady(t, result)
		if want := []string{"feat/mi-feature"}; !reflect.DeepEqual(branchNamesOf(onlyRepository(t, result)), want) {
			t.Errorf("ramas = %v, se esperaba solo %v", branchNamesOf(onlyRepository(t, result)), want)
		}
	})
	t.Run("multirepo", func(t *testing.T) {
		master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
		sealChangeAt(t, specs, "x", "backend")
		backend := filepath.Join(master, "repo-backend")
		runGit(t, backend, "branch", "feat/x", "main")
		runGit(t, backend, "branch", "feat/x-archive", "main")

		result := status(t, master, "x", branch.LaneSDD)

		wantReady(t, result)
		for _, repository := range result.Repositories {
			for _, described := range repository.Branches {
				if described.Kind == branch.KindArchive {
					t.Errorf("%s: un multirepo no tiene rama de archivado: %+v", repository.Name, described)
				}
			}
		}
	})
}

// RF-BRA-01 y RF-BRA-16: incluir la rama de archivado solo lee con la lista cerrada de
// subcomandos y nunca hace una operación remota.
func TestStatusArchiveBranchLaunchesNoRemoteOrWriteOperation(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "branch", "feat/x", "main")
	runGit(t, repo, "branch", "feat/x-archive", "main")
	seen := observe(t)

	result := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, result)
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) || isWriteInvocation(invocation) {
			t.Errorf("`status` no puede lanzar %v", invocation)
		}
	}
}
