package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas del avance de la integración del autofetch (RF-BRA-39 paso 2, design D-08), con git
// real y remotos desnudos locales: ninguna es paralela porque comparten el observador global de
// gitcommon y, alguna, variables de paquete.

// advanceBehind es un proyecto con `origin/main` un commit por delante de la `main` local, que
// está desplegada en el checkout principal. Devuelve el repositorio, el remoto y el SHA remoto.
func advanceBehind(t *testing.T) (repo, remote, remoteSHA string) {
	t.Helper()
	repo, remote = relevanceProject(t)
	remoteSHA = publishFromClone(t, remote, "main", map[string]string{"docs/nota.md": "nota\n"}, "nota")
	return repo, remote, remoteSHA
}

// gitAttempt ejecuta git y devuelve el error en lugar de abortar: para provocar conflictos.
func gitAttempt(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = exec.Command("git", append([]string{"-C", dir}, args...)...).Run()
}

func noticeKinds(result branch.AutofetchResult) []string {
	var kinds []string
	for _, notice := range result.Notices {
		kinds = append(kinds, notice.Kind)
	}
	return kinds
}

func integrationOf(t *testing.T, result branch.AutofetchResult) *branch.AutofetchIntegration {
	t.Helper()
	if result.Integration == nil {
		t.Fatalf("el resultado no describe la integración: %+v", result)
	}
	return result.Integration
}

// Filas 9 y 1: con la integración por detrás y desplegada con el árbol limpio avanza por
// fast-forward sin crear commits ni fusiones; el ciclo siguiente la encuentra al día.
func TestAutofetchAdvanceFastForwardsACleanDeployedIntegration(t *testing.T) {
	repo, _, remoteSHA := advanceBehind(t)
	before := runGit(t, repo, "rev-parse", "main")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	state := integrationOf(t, result)
	if !state.Advanced || state.State != "advanced" || state.Before != before || state.After != remoteSHA {
		t.Errorf("integración %+v, se esperaba avanzada de %s a %s", state, before, remoteSHA)
	}
	if got := runGit(t, repo, "rev-parse", "main"); got != remoteSHA {
		t.Errorf("main en %s, se esperaba el SHA remoto %s", got, remoteSHA)
	}
	if count := runGit(t, repo, "rev-list", "--count", "main..origin/main"); count != "0" {
		t.Errorf("quedan %s commits sin avanzar", count)
	}
	if merges := runGit(t, repo, "rev-list", "--merges", "--count", "main"); merges != "0" {
		t.Errorf("el avance creó %s fusiones", merges)
	}
	if len(result.Notices) != 0 {
		t.Errorf("un avance correcto no avisa: %v", noticeKinds(result))
	}

	again := autofetchRun(t, autofetchOn, repo, relevanceClock.Add(autofetchInterval))
	if state := integrationOf(t, again); state.State != "up_to_date" || state.Advanced {
		t.Errorf("segundo ciclo %+v, se esperaba up_to_date", state)
	}
}

// Fila 1 (sin R, sin L, L == R) y fila 2 (local por delante): no hay aviso aunque el árbol esté
// sucio (regla previa de RF-BRA-39 paso 3).
func TestAutofetchAdvanceNothingToDoNeverNoticesEvenWithADirtyTree(t *testing.T) {
	t.Run("al día con el árbol sucio", func(t *testing.T) {
		repo, _ := relevanceProject(t)
		writeFile(t, filepath.Join(repo, "sucio.txt"), "x\n")
		result := autofetchRun(t, autofetchOn, repo, relevanceClock)
		if state := integrationOf(t, result); state.State != "up_to_date" || len(result.Notices) != 0 {
			t.Errorf("integración %+v y avisos %v, se esperaba al día y sin avisos", state, noticeKinds(result))
		}
	})
	t.Run("local por delante con el árbol sucio", func(t *testing.T) {
		repo, _ := relevanceProject(t)
		commit(t, repo, "local")
		writeFile(t, filepath.Join(repo, "sucio.txt"), "x\n")
		head := runGit(t, repo, "rev-parse", "main")
		result := autofetchRun(t, autofetchOn, repo, relevanceClock)
		if state := integrationOf(t, result); state.State != "ahead" || len(result.Notices) != 0 {
			t.Errorf("integración %+v y avisos %v, se esperaba ahead y sin avisos", state, noticeKinds(result))
		}
		if runGit(t, repo, "rev-parse", "main") != head {
			t.Error("una rama por delante no se mueve")
		}
	})
	t.Run("sin referencia remota", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		runGit(t, repo, "remote", "add", "origin", t.TempDir())
		result := autofetchRun(t, autofetchOn, repo, relevanceClock)
		if len(result.Notices) != 1 || result.Notices[0].Kind != branch.AutofetchNoticeFetchFailed {
			t.Fatalf("avisos %v, se esperaba solo el del fetch fallido", noticeKinds(result))
		}
		if result.Integration != nil {
			t.Errorf("tras un fetch fallido no se evalúa el avance: %+v", result.Integration)
		}
	})
	t.Run("sin rama local", func(t *testing.T) {
		repo, _, _ := advanceBehind(t)
		runGit(t, repo, "checkout", "-q", "-b", "feat/z")
		runGit(t, repo, "branch", "-q", "-D", "main")
		result := autofetchRun(t, autofetchOn, repo, relevanceClock)
		if state := integrationOf(t, result); state.State != "no_local_branch" || len(result.Notices) != 0 {
			t.Errorf("integración %+v y avisos %v, se esperaba no_local_branch y sin avisos", state, noticeKinds(result))
		}
	})
}

// Fila 3: divergencia, solo aviso con el rebase y sin mover nada.
func TestAutofetchAdvanceDivergedIsOnlyANotice(t *testing.T) {
	repo, _, _ := advanceBehind(t)
	commit(t, repo, "local propio")
	local := runGit(t, repo, "rev-parse", "main")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if kinds := noticeKinds(result); !slices.Equal(kinds, []string{branch.AutofetchNoticeIntegrationDiverged}) {
		t.Fatalf("avisos %v, se esperaba integration_diverged", kinds)
	}
	want := "git -C " + pathquote.Quote(resolveDirectory(repo)) + " rebase origin/main"
	if got := result.Notices[0].Command; got != want {
		t.Errorf("orden %q, se esperaba %q", got, want)
	}
	if runGit(t, repo, "rev-parse", "main") != local || integrationOf(t, result).Advanced {
		t.Error("una rama divergida no se mueve")
	}
}

// inProgressProject deja `main` un commit por detrás del remoto y con un conflicto preparado:
// la rama `otra` toca el mismo fichero que el último commit local.
func inProgressProject(t *testing.T) string {
	t.Helper()
	repo, remote := relevanceProject(t)
	runGit(t, repo, "branch", "otra")
	runGit(t, repo, "checkout", "-q", "otra")
	writeFile(t, filepath.Join(repo, "f.txt"), "otra\n")
	runGit(t, repo, "add", "f.txt")
	commit(t, repo, "otra")
	runGit(t, repo, "checkout", "-q", "main")
	writeFile(t, filepath.Join(repo, "f.txt"), "main\n")
	runGit(t, repo, "add", "f.txt")
	commit(t, repo, "main")
	runGit(t, repo, "push", "-q", "origin", "main")
	publishFromClone(t, remote, "main", map[string]string{"docs/nota.md": "nota\n"}, "nota")
	return repo
}

// Fila 4: un merge, un rebase o un cherry-pick en curso sobre la integración dejan solo el
// aviso con `status`, sin tocar el índice ni los ficheros, antes que las filas 5 a 7.
func TestAutofetchAdvanceOperationInProgressIsOnlyANotice(t *testing.T) {
	for _, tc := range []struct {
		name      string
		operation string
		start     []string
	}{
		{"merge", "merge", []string{"merge", "otra"}},
		{"rebase", "rebase", []string{"rebase", "otra"}},
		{"cherry-pick", "cherry-pick", []string{"cherry-pick", "otra"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := inProgressProject(t)
			gitAttempt(t, repo, tc.start...)
			status := runGit(t, repo, "status", "--porcelain")
			local := runGit(t, repo, "rev-parse", "main")
			file, err := os.ReadFile(filepath.Join(repo, "f.txt"))
			if err != nil {
				t.Fatal(err)
			}

			result := autofetchRun(t, autofetchOn, repo, relevanceClock)

			if kinds := noticeKinds(result); !slices.Equal(kinds, []string{branch.AutofetchNoticeOperationInProgress}) {
				t.Fatalf("avisos %v, se esperaba integration_operation_in_progress", kinds)
			}
			notice := result.Notices[0]
			if !strings.Contains(notice.Message, tc.operation) || notice.Command != "git -C "+pathquote.Quote(resolveDirectory(repo))+" status" {
				t.Errorf("aviso %+v, se esperaba la operación %s y la orden status", notice, tc.operation)
			}
			after, _ := os.ReadFile(filepath.Join(repo, "f.txt"))
			if runGit(t, repo, "status", "--porcelain") != status || string(after) != string(file) || runGit(t, repo, "rev-parse", "main") != local {
				t.Error("el ciclo no debe tocar el índice, los ficheros ni la rama")
			}
			if integrationOf(t, result).Advanced {
				t.Error("con una operación en curso no se avanza")
			}
		})
	}
}

// Fila 5: sin desplegar y con HEAD desacoplado en el checkout principal, la orden es `switch`
// y después `merge --ff-only`.
func TestAutofetchAdvanceDetachedHeadIsOnlyANotice(t *testing.T) {
	repo, _, _ := advanceBehind(t)
	runGit(t, repo, "checkout", "-q", "--detach")
	local := runGit(t, repo, "rev-parse", "main")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if kinds := noticeKinds(result); !slices.Equal(kinds, []string{branch.AutofetchNoticeIntegrationDetached}) {
		t.Fatalf("avisos %v, se esperaba integration_detached", kinds)
	}
	quoted := pathquote.Quote(resolveDirectory(repo))
	want := "git -C " + quoted + " switch main && git -C " + quoted + " merge --ff-only origin/main"
	if got := result.Notices[0].Command; got != want {
		t.Errorf("orden %q, se esperaba %q", got, want)
	}
	if runGit(t, repo, "rev-parse", "main") != local {
		t.Error("una rama no desplegada no se mueve")
	}
}

// Fila 6: no desplegada y el checkout principal en otra rama, la orden es el `fetch .` sin `+`.
func TestAutofetchAdvanceNotDeployedIsOnlyANotice(t *testing.T) {
	repo, _, _ := advanceBehind(t)
	runGit(t, repo, "checkout", "-q", "-b", "feat/z")
	local := runGit(t, repo, "rev-parse", "main")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if kinds := noticeKinds(result); !slices.Equal(kinds, []string{branch.AutofetchNoticeNotDeployed}) {
		t.Fatalf("avisos %v, se esperaba integration_not_deployed", kinds)
	}
	want := "git -C " + pathquote.Quote(resolveDirectory(repo)) + " fetch . refs/remotes/origin/main:refs/heads/main"
	if got := result.Notices[0].Command; got != want {
		t.Errorf("orden %q, se esperaba %q", got, want)
	}
	if runGit(t, repo, "rev-parse", "main") != local {
		t.Error("una rama no desplegada no se mueve")
	}
}

// Fila 7: el árbol sucio (fichero sin seguimiento o preparado) impide el avance.
func TestAutofetchAdvanceDirtyTreeIsOnlyANotice(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stage bool
	}{{"sin seguimiento", false}, {"preparado", true}} {
		t.Run(tc.name, func(t *testing.T) {
			repo, _, _ := advanceBehind(t)
			writeFile(t, filepath.Join(repo, "sucio.txt"), "x\n")
			if tc.stage {
				runGit(t, repo, "add", "sucio.txt")
			}
			local := runGit(t, repo, "rev-parse", "main")

			result := autofetchRun(t, autofetchOn, repo, relevanceClock)

			if kinds := noticeKinds(result); !slices.Equal(kinds, []string{branch.AutofetchNoticeIntegrationDirty}) {
				t.Fatalf("avisos %v, se esperaba integration_dirty", kinds)
			}
			want := "git -C " + pathquote.Quote(resolveDirectory(repo)) + " merge --ff-only origin/main"
			if got := result.Notices[0].Command; got != want {
				t.Errorf("orden %q, se esperaba %q", got, want)
			}
			if runGit(t, repo, "rev-parse", "main") != local {
				t.Error("con el árbol sucio no se avanza")
			}
		})
	}
}

// Fila 8: sin presupuesto para iniciar el avance queda el aviso con el motivo y la orden de la
// fila 7.
func TestAutofetchAdvanceWithoutBudgetIsANotice(t *testing.T) {
	repo, _, _ := advanceBehind(t)
	local := runGit(t, repo, "rev-parse", "main")
	previous := autofetchAdvanceReserve
	autofetchAdvanceReserve = time.Hour
	t.Cleanup(func() { autofetchAdvanceReserve = previous })

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if kinds := noticeKinds(result); !slices.Equal(kinds, []string{branch.AutofetchNoticeFastForwardFailed}) {
		t.Fatalf("avisos %v, se esperaba integration_ff_failed", kinds)
	}
	if notice := result.Notices[0]; !strings.Contains(notice.Message, "sin tiempo") || !strings.Contains(notice.Command, "merge --ff-only origin/main") {
		t.Errorf("aviso %+v, se esperaba el motivo de tiempo y la orden", notice)
	}
	if runGit(t, repo, "rev-parse", "main") != local {
		t.Error("sin presupuesto no se avanza")
	}
}

// Fila 10: un `index.lock` hace fallar el fast-forward; no hay reintento y queda el aviso con la
// orden y el resumen de git.
func TestAutofetchAdvanceFailedFastForwardIsANoticeWithoutRetry(t *testing.T) {
	repo, _, _ := advanceBehind(t)
	local := runGit(t, repo, "rev-parse", "main")
	gitDir, err := gitcommon.GitDirFromFiles(repo)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "index.lock"), "")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if kinds := noticeKinds(result); !slices.Equal(kinds, []string{branch.AutofetchNoticeFastForwardFailed}) {
		t.Fatalf("avisos %v, se esperaba integration_ff_failed", kinds)
	}
	if notice := result.Notices[0]; !strings.Contains(notice.Message, "index.lock") || !strings.Contains(notice.Command, "merge --ff-only origin/main") {
		t.Errorf("aviso %+v, se esperaba el resumen de git y la orden", notice)
	}
	if runGit(t, repo, "rev-parse", "main") != local || integrationOf(t, result).Advanced {
		t.Error("el avance fallido no mueve la rama")
	}
}

// Con la integración desplegada en un worktree enlazado y el checkout principal en otra rama, el
// avance se hace en ese worktree y el checkout principal no se toca.
func TestAutofetchAdvanceFastForwardsAnIntegrationDeployedInALinkedWorktree(t *testing.T) {
	repo, _, remoteSHA := advanceBehind(t)
	runGit(t, repo, "checkout", "-q", "-b", "feat/z")
	linked := filepath.Join(t.TempDir(), "integracion")
	runGit(t, repo, "worktree", "add", "-q", linked, "main")
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if state := integrationOf(t, result); !state.Advanced || state.After != remoteSHA {
		t.Fatalf("integración %+v, se esperaba avanzada a %s", state, remoteSHA)
	}
	if runGit(t, linked, "rev-parse", "HEAD") != remoteSHA {
		t.Error("el worktree enlazado debía avanzar")
	}
	if runGit(t, repo, "rev-parse", "HEAD") != head {
		t.Error("el checkout principal en otra rama no se toca")
	}
}

// `main_branch` distinta de la integración queda intocable: solo avanza la integración.
func TestAutofetchAdvanceLeavesAMainBranchDifferentFromTheIntegrationUntouched(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, twoStageGit))
	runGit(t, repo, "branch", "-m", "master")
	runGit(t, repo, "branch", "develop")
	remote := t.TempDir()
	runGit(t, remote, "init", "-q", "--bare", "-b", "master")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-q", "origin", "master", "develop")
	runGit(t, repo, "checkout", "-q", "develop")
	publishFromClone(t, remote, "master", map[string]string{"docs/master.md": "m\n"}, "master nuevo")
	developSHA := publishFromClone(t, remote, "develop", map[string]string{"docs/develop.md": "d\n"}, "develop nuevo")
	master := runGit(t, repo, "rev-parse", "master")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	if state := integrationOf(t, result); state.Branch != "develop" || !state.Advanced || state.After != developSHA {
		t.Errorf("integración %+v, se esperaba develop avanzada a %s", state, developSHA)
	}
	if runGit(t, repo, "rev-parse", "master") != master {
		t.Error("main_branch distinta de la integración no se mueve")
	}
}

// Orden de D-07: los avisos de no avance van entre los (a) y los (b).
func TestAutofetchAdvanceNoticesGoBetweenIncomingAndBranchNotices(t *testing.T) {
	repo, remote := relevanceProject(t)
	registerChange(t, repo, branch.LaneSDD, "x", "feat/x")
	publishFromClone(t, remote, "main", map[string]string{"openspec/changes/x/proposal.md": "p\n"}, "propuesta")
	publishFromClone(t, remote, "feat/x", map[string]string{"codigo.txt": "c\n"}, "rama del cambio")
	runGit(t, repo, "branch", "feat/x", "main")
	writeFile(t, filepath.Join(repo, "sucio.txt"), "x\n")

	result := autofetchRun(t, autofetchOn, repo, relevanceClock)

	want := []string{branch.AutofetchNoticeIncomingRelevant, branch.AutofetchNoticeIntegrationDirty, branch.AutofetchNoticeChangeBranchBehind}
	if kinds := noticeKinds(result); !slices.Equal(kinds, want) {
		t.Errorf("avisos %v, se esperaba %v", kinds, want)
	}
}
