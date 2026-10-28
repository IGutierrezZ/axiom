package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Pruebas del aviso de la guarda D con git real, entorno aislado y sin red (REQ-25.21 escenarios 3
// a 8, design D-18 paso 4): con el evento `post` y un fichero que no es un documento de Axiom, avisa
// sin bloquear cuando el checkout que lo contiene no está en la rama de ningún cambio activo
// registrado por `ensure`, y no hace nada en cualquier otro caso. Decide solo con lecturas de
// ficheros: ninguna de estas pruebas observa un proceso de git durante la guarda.

func postWrite(paths ...string) GuardPayload {
	return GuardPayload{Event: GuardEventPost, Tool: "Write", Paths: paths}
}

func codeFileOf(dir string) string { return filepath.Join(dir, "internal", "foo", "foo.go") }

func wantWarned(t *testing.T, decision GuardDecision, wants ...string) {
	t.Helper()
	if decision.Action != GuardActionWarn || decision.Message == "" {
		t.Fatalf("decisión %+v, se esperaba un aviso con mensaje", decision)
	}
	for _, want := range wants {
		if !strings.Contains(decision.Message, want) {
			t.Errorf("el aviso %q debe contener %q", decision.Message, want)
		}
	}
}

// registeredWorktree es el worktree que el registro guarda para la rama del cambio.
func registeredWorktree(t *testing.T, repo, change string) string {
	t.Helper()
	for _, repository := range loadRecord(t, repo, change).Repositories {
		for _, registered := range repository.Branches {
			if registered.Kind == branch.KindChange && registered.Worktree != "" {
				return registered.Worktree
			}
		}
	}
	t.Fatalf("el registro de %s no guarda el worktree de su rama", change)
	return ""
}

// inPlaceChange es un monorepo con el cambio `x` registrado en modo `in_place` y desplegado de
// nuevo en `main`: no cuesta ningún worktree y deja el checkout en la rama equivocada.
func inPlaceChange(t *testing.T) string {
	t.Helper()
	repo := embeddedCoreProject(t)
	wantReady(t, ensureInPlace(t, repo, "x"))
	runGit(t, repo, "switch", "-q", "main")
	return repo
}

// REQ-25.21, escenarios 3, 4 y 6: un fichero de código escrito en el checkout principal en `main`
// avisa sin bloquear con el worktree correcto; con varios cambios activos nombra los de todos; y no
// avisa dentro del worktree del cambio, cuyo HEAD es la rama registrada.
func TestGuardWarnNamesTheCorrectWorktreeOfEveryActiveChange(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktreeX := registeredWorktree(t, repo, "x")
	file := codeFileOf(repo)

	t.Run("un cambio activo y el checkout principal en main", func(t *testing.T) {
		decision := guardOf(t, postWrite(file))

		wantWarned(t, decision, file, "«main»", "cambio «x»", worktreeX, "«feat/x»")
		if decision.Action == GuardActionBlock {
			t.Errorf("el aviso no bloquea: %+v", decision)
		}
	})
	t.Run("sin aviso si el fichero está en el worktree correcto", func(t *testing.T) {
		wantNothing(t, guardOf(t, postWrite(codeFileOf(worktreeX))))
	})
	t.Run("con varios cambios activos nombra los worktrees de todos", func(t *testing.T) {
		wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "y"}))
		worktreeY := registeredWorktree(t, repo, "y")

		wantWarned(t, guardOf(t, postWrite(file)), worktreeX, worktreeY, "cambio «x»", "cambio «y»")
		// El worktree de uno de los cambios es el correcto para ese cambio: no se avisa.
		wantNothing(t, guardOf(t, postWrite(codeFileOf(worktreeX))))
		wantNothing(t, guardOf(t, postWrite(codeFileOf(worktreeY))))
	})
}

// REQ-25.21, escenario 5 («En modo `in_place` el aviso nombra la rama esperada»): sin worktree
// que nombrar, el aviso dice la rama esperada; en la rama del cambio no avisa, y en otra rama o con
// un HEAD desacoplado sí.
func TestGuardWarnNamesTheExpectedBranchInPlace(t *testing.T) {
	t.Parallel()
	repo := inPlaceChange(t)
	file := codeFileOf(repo)

	decision := guardOf(t, postWrite(file))
	wantWarned(t, decision, file, "«main»", "cambio «x»", "rama esperada «feat/x»", "in_place")
	if strings.Contains(decision.Message, "worktree") {
		t.Errorf("en in_place no hay worktree que nombrar: %q", decision.Message)
	}

	runGit(t, repo, "switch", "-q", "feat/x")
	wantNothing(t, guardOf(t, postWrite(file)))
	runGit(t, repo, "switch", "-q", "-c", "otra")
	wantWarned(t, guardOf(t, postWrite(file)), "«otra»", "rama esperada «feat/x»")
	runGit(t, repo, "switch", "-q", "--detach", "HEAD")
	wantWarned(t, guardOf(t, postWrite(file)), "HEAD desacoplado", "rama esperada «feat/x»")
}

// Un parche con varios ficheros del mismo checkout avisa una sola vez. Un documento de Axiom del
// parche no cuenta como fichero de código.
func TestGuardWarnWarnsOncePerCheckoutOfAPatch(t *testing.T) {
	t.Parallel()
	repo := inPlaceChange(t)

	decision := guardOf(t, postWrite(codeFileOf(repo), filepath.Join(repo, "internal", "bar.go"), documentOf(repo)))

	wantWarned(t, decision, "cambio «x»")
	if count := strings.Count(decision.Message, "aviso:"); count != 1 {
		t.Errorf("el aviso aparece %d veces en %q, se esperaba una", count, decision.Message)
	}
}

// Design D-18, paso 5: cada evento tiene una única responsabilidad. `pre` con otro fichero y `post`
// con un documento de Axiom no hacen nada, de modo que el aviso nunca se duplica con el bloqueo.
func TestGuardWarnEachEventHasASingleResponsibility(t *testing.T) {
	t.Parallel()
	repo := inPlaceChange(t)
	file := codeFileOf(repo)
	document := documentOf(repo)

	wantNothing(t, guardOf(t, preWrite(file)))
	wantNothing(t, guardOf(t, postWrite(document)))
	wantNothing(t, guardOf(t, postWrite(filepath.Join(repo, "odd", "tasks", "x.md"))))
	// Una carpeta `openspec/changes/<x>/` de un fixture no es un documento de Axiom del proyecto: es
	// otro fichero, y en `main` avisa como cualquier otro.
	wantWarned(t, guardOf(t, postWrite(filepath.Join(repo, "internal", "testdata", "openspec", "changes", "x", "proposal.md"))), "cambio «x»")
	// El mismo documento en `pre` bloquea y en `post` no avisa: el bloqueo y el aviso no se solapan.
	wantBlocked(t, guardOf(t, preWrite(document)))
	wantWarned(t, guardOf(t, postWrite(file)), "cambio «x»")
	unknown := postWrite(file)
	unknown.Event = ""
	wantNothing(t, guardOf(t, unknown))
}

// REQ-25.21, escenario 7: sin cambio activo registrado en el repositorio la guarda no hace nada
// con el código: ni sin registro, ni con el registro de otro repositorio.
func TestGuardWarnDoesNothingWithoutAnActiveChangeRegisteredInTheRepository(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	file := codeFileOf(repo)

	t.Run("sin ningún registro", func(t *testing.T) {
		wantNothing(t, guardOf(t, postWrite(file)))
	})
	t.Run("con el registro de otro repositorio", func(t *testing.T) {
		record := registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, Worktree: filepath.Join(t.TempDir(), "x")})
		record.Repositories[0].CommonDir = filepath.Join(t.TempDir(), ".git")
		writeRegistry(t, commonDir(repo), record)

		wantNothing(t, guardOf(t, postWrite(file)))
	})
	t.Run("con un registro sin ramas", func(t *testing.T) {
		writeRegistry(t, commonDir(repo), registeredRecord(repo))

		wantNothing(t, guardOf(t, postWrite(file)))
	})
}

// REQ-25.21: la rama registrada de cualquier tipo —cambio, rol o archivado— y de cualquier cambio
// activo es una rama correcta y no avisa; otra sí, y nombra los worktrees de ambos carriles, SDD y
// ODD, sin dar el de archivado como lugar de escritura del código.
func TestGuardWarnRegisteredBranchesOfAnyKindAndLaneDoNotWarn(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	worktrees := t.TempDir()
	worktreeOf := func(name string) string { return filepath.Join(worktrees, name) }
	writeRegistry(t, commonDir(repo), registeredRecord(repo,
		branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, Worktree: worktreeOf("x")},
		branch.RecordBranch{Name: "feat/x-core", Kind: branch.KindRole, RoleID: "core", Worktree: worktreeOf("x-core")},
		branch.RecordBranch{Name: "feat/x-archive", Kind: branch.KindArchive, Worktree: worktreeOf("x-archive")},
	))
	odd := registeredRecord(repo, branch.RecordBranch{Name: "feat/y", Kind: branch.KindChange, Worktree: worktreeOf("y")})
	odd.Lane, odd.Change = branch.LaneODD, "y"
	writeRegistry(t, commonDir(repo), odd)
	file := codeFileOf(repo)

	for _, name := range []string{"feat/x", "feat/x-core", "feat/x-archive", "feat/y"} {
		runGit(t, repo, "switch", "-q", "-c", name)
		wantNothing(t, guardOf(t, postWrite(file)))
	}

	runGit(t, repo, "switch", "-q", "-c", "otra")
	decision := guardOf(t, postWrite(file))
	wantWarned(t, decision, "cambio «x»", worktreeOf("x"), worktreeOf("x-core"), "cambio ODD «y»", worktreeOf("y"))
	if strings.Contains(decision.Message, worktreeOf("x-archive")) {
		t.Errorf("el worktree de archivado no es donde se escribe el código: %q", decision.Message)
	}
}

// REQ-25.21, escenario 8: el repositorio de specs de un multirepo queda exento del aviso aunque no
// esté en su rama de integración (no del bloqueo de documentos); un repositorio de código con el
// cambio registrado sí avisa.
func TestGuardWarnExemptsTheSpecsRepositoryOfAMultirepo(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: master, Change: "x", Mode: branch.ModeInPlace}))
	backend := filepath.Join(master, "repo-backend")
	runGit(t, backend, "switch", "-q", "main")

	t.Run("el repositorio de specs en su integración", func(t *testing.T) {
		wantNothing(t, guardOf(t, postWrite(filepath.Join(specs, "docs", "guia.txt"))))
	})
	t.Run("el repositorio de specs fuera de su integración", func(t *testing.T) {
		runGit(t, specs, "switch", "-q", "-c", "otra")
		wantNothing(t, guardOf(t, postWrite(filepath.Join(specs, "docs", "guia.txt"))))
		runGit(t, specs, "switch", "-q", "main")
	})
	t.Run("un repositorio de código con el cambio registrado", func(t *testing.T) {
		file := codeFileOf(backend)

		wantWarned(t, guardOf(t, postWrite(file)), file, "«main»", "cambio «x»", "rama esperada «feat/x»")
		runGit(t, backend, "switch", "-q", "feat/x")
		wantNothing(t, guardOf(t, postWrite(file)))
	})
}

// La exención del repositorio de specs es de la guarda y no un efecto de que el registro no lo
// conozca (`ensure` nunca registra ramas del repositorio de specs, que se trabaja in situ): aunque
// un registro nombrara una rama suya, escribir en él no avisa. Sin `openspec/` en el repositorio de
// specs, `Locate` asciende hasta la carpeta maestra y resuelve el multirepo.
func TestGuardWarnExemptsTheSpecsRepositoryEvenIfARecordNamesIt(t *testing.T) {
	t.Parallel()
	_, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	record := registeredRecord(specs, branch.RecordBranch{Name: "feat/z", Kind: branch.KindChange, InPlace: true})
	record.Change = "z"
	record.Repositories[0].Role = branch.RepoSpecsInPlace
	writeRegistry(t, commonDir(specs), record)
	file := filepath.Join(specs, "docs", "guia.txt")

	wantNothing(t, guardOf(t, postWrite(file)))
	runGit(t, specs, "switch", "-q", "-c", "otra")
	wantNothing(t, guardOf(t, postWrite(file)))
}

// REQ-25.21, escenarios «Inerte fuera de un proyecto Axiom» y «Fail-open»: configuración o registro
// ilegibles, un HEAD que no se puede leer, un contexto cancelado y un pánico recuperado no avisan.
func TestGuardWarnFailsOpen(t *testing.T) {
	t.Parallel()
	cases := map[string]func(t *testing.T) (repo string){
		"configuración ilegible": func(t *testing.T) string {
			repo := inPlaceChange(t)
			writeFile(t, filepath.Join(repo, "axiom.yaml"), "workspace: [sin cerrar\n")
			return repo
		},
		"registro del cambio ilegible": func(t *testing.T) string {
			repo := inPlaceChange(t)
			writeFile(t, recordFile(commonDir(repo), "sdd", "x"), "{ no es json\n")
			return repo
		},
		"registro de otro cambio ilegible": func(t *testing.T) string {
			repo := inPlaceChange(t)
			writeFile(t, recordFile(commonDir(repo), "sdd", "z"), "{ no es json\n")
			return repo
		},
		"registro del carril ODD ilegible": func(t *testing.T) string {
			repo := inPlaceChange(t)
			writeFile(t, recordFile(commonDir(repo), "odd", "z"), "{ no es json\n")
			return repo
		},
		"HEAD ilegible": func(t *testing.T) string {
			repo := inPlaceChange(t)
			writeFile(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/.invalid\n")
			return repo
		},
		"HEAD ausente": func(t *testing.T) string {
			repo := inPlaceChange(t)
			if err := os.Remove(filepath.Join(repo, ".git", "HEAD")); err != nil {
				t.Fatal(err)
			}
			return repo
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo := setup(t)
			wantNothing(t, guardOf(t, postWrite(codeFileOf(repo))))
		})
	}

	t.Run("fuera de un proyecto Axiom", func(t *testing.T) {
		t.Parallel()
		requireGit(t)
		dir := t.TempDir()
		runGit(t, dir, "init", "-q", "-b", "main")
		commit(t, dir, "inicial")
		before := listDirectory(t, dir)

		wantNothing(t, guardOf(t, postWrite(codeFileOf(dir))))
		if after := listDirectory(t, dir); len(after) != len(before) {
			t.Errorf("la guarda no debe escribir ficheros: antes %v, después %v", before, after)
		}
	})
	t.Run("contexto cancelado y pánico recuperado", func(t *testing.T) {
		t.Parallel()
		repo := inPlaceChange(t)
		payload := postWrite(codeFileOf(repo))
		wantWarned(t, guardOf(t, payload))

		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		wantNothing(t, Default().Guard(cancelled, payload))
		var unmounted *Engine
		wantNothing(t, unmounted.Guard(context.Background(), payload))
	})
}

// REQ-25.21, escenario «El hook no lanza git ni ningún otro proceso»: ni el aviso ni el silencio
// lanzan una sola invocación de git. No es paralela: el observador de procesos es global.
func TestGuardWarnLaunchesNoProcess(t *testing.T) {
	repo := inPlaceChange(t)
	file := codeFileOf(repo)
	seen := observe(t)

	wantWarned(t, guardOf(t, postWrite(file)))
	wantNothing(t, guardOf(t, preWrite(file)))
	wantNothing(t, guardOf(t, postWrite(documentOf(repo))))
	wantNothing(t, guardOf(t, postWrite(filepath.Join(t.TempDir(), "fuera.go"))))

	if len(*seen) != 0 {
		t.Errorf("la guarda lanzó procesos de git: %v", *seen)
	}
}

// El aviso llega por el canal de cada runtime sin bloquear: Claude Code y Codex lo devuelven con
// salida 2 y el texto por `stderr`, que el runtime entrega al agente (la escritura ya ocurrió);
// OpenCode, por `stdout` con `action: warn`.
func TestGuardWarnReachesEachRuntimeThroughItsChannel(t *testing.T) {
	t.Parallel()
	repo := inPlaceChange(t)
	file := codeFileOf(repo)

	for agent, payload := range map[string]string{
		GuardAgentClaudeCode: claudeWrite(t, "PostToolUse", repo, file),
		GuardAgentCodex:      codexPatch(t, "PostToolUse", repo, file),
		GuardAgentOpenCode:   openCodeWrite(t, GuardEventPost, repo, file),
	} {
		t.Run(agent, func(t *testing.T) {
			code, stdout, stderr := runGuardFor(t, agent, payload)
			if agent == GuardAgentOpenCode {
				if code != 0 || stderr != "" || !strings.HasPrefix(stdout, `{"action":"warn","message":"axiom branch guard: aviso:`) {
					t.Fatalf("código %d, stdout %q, stderr %q; se esperaba 0 y action=warn por stdout", code, stdout, stderr)
				}
				return
			}
			if code != 2 || stdout != "" || !strings.Contains(stderr, "axiom branch guard: aviso:") || !strings.Contains(stderr, "«feat/x»") {
				t.Fatalf("código %d, stdout %q, stderr %q; se esperaba 2 con el aviso por stderr", code, stdout, stderr)
			}
		})
	}
}
