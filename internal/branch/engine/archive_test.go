package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de `ensure --archive` (REQ-25.45, REQ-25.7, REQ-25.8, REQ-25.9, REQ-25.11, REQ-25.12,
// design D-25 y D-06) con git real, entorno aislado y remotos desnudos locales: la rama de
// archivado sale de la integración con el cambio ya fusionado, sin tocar las ramas del cambio ni
// las de rol y sin publicar nada. Cada `git worktree add` cuesta segundos en Windows, de modo
// que las pruebas comparten repositorio y usan `in_place` cuando el worktree no es lo que se
// comprueba.

// mergeChange confirma en main, con un merge commit, el cambio dado bajo
// `openspec/changes/<cambio>/`: es como si el PR del cambio ya se hubiera fusionado.
func mergeChange(t *testing.T, repo, change string) {
	t.Helper()
	runGit(t, repo, "switch", "-q", "-c", "fusion-"+change)
	writeFile(t, filepath.Join(repo, "openspec", "changes", change, "proposal.md"), "propuesta\n")
	runGit(t, repo, "add", "-A")
	commit(t, repo, "cambio "+change)
	runGit(t, repo, "switch", "-q", "main")
	commit(t, repo, "avance de main")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "fusiona "+change, "fusion-"+change)
	runGit(t, repo, "branch", "-q", "-D", "fusion-"+change)
}

// publishChange publica en el remoto desnudo, desde otro clon, un commit de main que contiene el
// cambio, sin que repo lo vea, y devuelve el commit al que apunta main en el remoto.
func publishChange(t *testing.T, remote, change string) string {
	t.Helper()
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	runGit(t, scratch, "clone", "-q", remote, clone)
	writeFile(t, filepath.Join(clone, "openspec", "changes", change, "proposal.md"), "propuesta\n")
	runGit(t, clone, "add", "-A")
	commit(t, clone, "fusiona "+change)
	runGit(t, clone, "push", "-q", "origin", "main")
	return runGit(t, clone, "rev-parse", "HEAD")
}

// archiveRequest es la petición de `ensure --archive` del cambio x en el carril SDD.
func archiveRequest(cwd string) EnsureRequest {
	return EnsureRequest{CWD: cwd, Change: "x", Lane: branch.LaneSDD, Archive: true}
}

// ensureArchive ejecuta `ensure --archive` con un hub que no pregunta (origen `hub`).
func ensureArchive(t *testing.T, request EnsureRequest) branch.Result {
	t.Helper()
	return ensureWith(t, hubWorktree, request)
}

func archiveWorktree(t *testing.T, repo string) string {
	t.Helper()
	return plannedWorktree(t, repo, "x-archive", filepath.Base(repo))
}

func wantOnlyArchiveBranch(t *testing.T, result branch.Result, name string) branch.BranchResult {
	t.Helper()
	repository := onlyRepository(t, result)
	if len(repository.Branches) != 1 {
		t.Fatalf("el resultado de --archive describe solo la rama de archivado, hay %+v", repository.Branches)
	}
	archive := repository.Branches[0]
	if archive.Name != name || archive.Kind != branch.KindArchive || archive.KindLabel != "rama de archivado" || !archive.Exists || archive.Role != "" {
		t.Fatalf("rama = %+v, se esperaba la rama de archivado %s", archive, name)
	}
	return archive
}

// REQ-25.45 (escenario «Crea la rama de archivado desde la integración con el cambio fusionado»),
// REQ-25.11 (ruta de la rama de archivado) y REQ-25.4: con el cambio fusionado con un merge commit
// en `origin/main` (referencia local), nace `feat/x-archive` desde `origin/main` con su worktree
// en `<raíz>/x-archive/<repo>`, la base es `origin/main` con `fetched=false`, la rama del cambio
// no se crea, el checkout principal queda como estaba y el registro guarda la rama con `kind:
// archive`. Repetirlo es idempotente (REQ-25.7): el mismo JSON, el mismo estado, ninguna escritura
// y ninguna operación remota, ni `push`.
func TestEnsureArchiveCreatesTheArchiveBranchFromTheIntegrationWithItsWorktree(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	remoteWithMain(t, repo)
	integration := runGit(t, repo, "rev-parse", "refs/remotes/origin/main")
	before := checkoutState(t, repo)

	result := ensureArchive(t, archiveRequest(repo))

	wantReady(t, result)
	if result.Operation != "ensure" || result.Change != "x" || result.Lane != branch.LaneSDD {
		t.Errorf("operación %q, cambio %q, carril %q; el resultado de --archive es el de `ensure`", result.Operation, result.Change, result.Lane)
	}
	wantMode(t, result, branch.ModeWorktree, branch.ModeSourceHub)
	repository := onlyRepository(t, result)
	if repository.Base == nil || *repository.Base != (branch.BaseResult{Ref: "origin/main", SHA: integration, Source: "remote-tracking", Fetched: false}) {
		t.Errorf("base = %+v, se esperaba origin/main en %s sin fetch", repository.Base, integration)
	}
	archive := wantOnlyArchiveBranch(t, result, "feat/x-archive")
	if archive.PRTarget != "main" || archive.Dirty {
		t.Errorf("rama de archivado = %+v, se esperaba el destino main", archive)
	}
	worktree := archiveWorktree(t, repo)
	wantSameDirectory(t, "worktree de feat/x-archive", archive.Worktree, worktree)
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x-archive"); got != integration {
		t.Errorf("feat/x-archive apunta a %s, se esperaba la base %s", got, integration)
	}
	if got := runGit(t, worktree, "symbolic-ref", "--short", "HEAD"); got != "feat/x-archive" {
		t.Errorf("el worktree tiene %s, se esperaba feat/x-archive", got)
	}
	if _, err := os.Stat(filepath.Join(worktree, "openspec", "changes", "x", "proposal.md")); err != nil {
		t.Errorf("el worktree de archivado parte de la integración, que contiene el cambio: %v", err)
	}
	if runGit(t, repo, "branch", "--list", "feat/x") != "" {
		t.Error("`--archive` no crea la rama del cambio")
	}
	if after := checkoutState(t, repo); after != before {
		t.Errorf("el checkout principal cambió:\nantes:\n%s\ndespués:\n%s", before, after)
	}
	record := loadRecord(t, repo, "x")
	if record.Mode != branch.ModeWorktree || record.ModeSource != branch.ModeSourceHub || record.ArchivePrefix != "" || len(record.Repositories) != 1 {
		t.Fatalf("registro = %+v", record)
	}
	registered := record.Repositories[0].Branches
	if len(registered) != 1 || registered[0].Name != "feat/x-archive" || registered[0].Kind != branch.KindArchive || registered[0].RoleID != "" {
		t.Fatalf("ramas registradas = %+v, se esperaba solo la de archivado con kind archive", registered)
	}
	wantSameDirectory(t, "worktree registrado", registered[0].Worktree, worktree)
	firstJSON := resultJSON(t, result)
	for _, want := range []string{`"kind": "archive"`, `"kindLabel": "rama de archivado"`, `"fetched": false`} {
		if !strings.Contains(firstJSON, want) {
			t.Errorf("el JSON debe contener %s:\n%s", want, firstJSON)
		}
	}

	t.Run("repetirlo es idempotente y no publica nada", func(t *testing.T) {
		state := repositoryState(t, repo)
		recordBytes, err := os.ReadFile(recordFile(commonDir(repo), "sdd", "x"))
		if err != nil {
			t.Fatal(err)
		}
		seen := observe(t)

		again := ensureArchive(t, archiveRequest(repo))

		wantReady(t, again)
		if got := resultJSON(t, again); got != firstJSON {
			t.Errorf("la repetición cambió el JSON:\n%s\n---\n%s", firstJSON, got)
		}
		if got := repositoryState(t, repo); got != state {
			t.Errorf("la repetición cambió el repositorio:\n%s\n---\n%s", state, got)
		}
		if after, err := os.ReadFile(recordFile(commonDir(repo), "sdd", "x")); err != nil || string(after) != string(recordBytes) {
			t.Errorf("el registro cambió entre ejecuciones: %q, %v", after, err)
		}
		for _, invocation := range *seen {
			if isWriteInvocation(invocation) || remoteOperation(invocation[1]) {
				t.Errorf("la repetición no escribe ni toca la red, pero lanzó %v", invocation)
			}
		}
	})
}

// REQ-25.45 y REQ-25.7: una rama de archivado que ya existe se reutiliza sin moverla (solo se
// añade su worktree) y la repetición no la publica.
func TestEnsureArchiveReusesAnExistingArchiveBranchWithoutMovingIt(t *testing.T) {
	repo := embeddedCoreProject(t)
	old := runGit(t, repo, "rev-parse", "main")
	runGit(t, repo, "branch", "feat/x-archive", "main")
	mergeChange(t, repo, "x")
	if runGit(t, repo, "rev-parse", "main") == old {
		t.Fatal("la preparación debía adelantar main")
	}
	seen := observe(t)

	result := ensureArchive(t, archiveRequest(repo))

	wantReady(t, result)
	wantSameDirectory(t, "worktree", wantOnlyArchiveBranch(t, result, "feat/x-archive").Worktree, archiveWorktree(t, repo))
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x-archive"); got != old {
		t.Errorf("la rama existente se movió a %s, debía seguir en %s", got, old)
	}
	for _, invocation := range *seen {
		if invocation[1] == "branch" {
			t.Errorf("la rama ya existía: no se crea ni se mueve ninguna rama, pero se lanzó %v", invocation)
		}
	}
}

// REQ-25.45 (escenarios «La base local sin el cambio fusionado se rechaza y sugiere `--fetch`» y
// «`--fetch` explícito permite alcanzar el cambio fusionado»): con el cambio solo en el remoto,
// `ensure --archive` rechaza con `change_not_integrated` nombrando la base evaluada (origin/main)
// y la carpeta que falta, sugiere `--fetch` con una orden ejecutable, no crea ninguna rama,
// worktree ni registro y no hace ninguna operación remota; con `--fetch` explícito, se ejecuta un
// único `fetch origin` antes de evaluar la base, `fetched` es `true` y la rama sale del commit
// traído.
func TestEnsureArchiveRefusesAChangeTheBaseDoesNotContainAndFetchReachesIt(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	advanced := publishChange(t, remote, "x")
	state := repositoryState(t, repo)
	seen := observe(t)

	result := ensureArchive(t, archiveRequest(repo))

	wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
	for _, want := range []string{"origin/main", "openspec/changes/x/", "x"} {
		if !strings.Contains(result.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", result.Message, want)
		}
	}
	if want := "axiom branch ensure --cwd " + pathquote.Quote(repo) + " --change x --archive --fetch"; !strings.Contains(result.Continuation, want) {
		t.Errorf("la continuación %q debe sugerir --fetch con la orden %q", result.Continuation, want)
	}
	if result.Details["base"] != "origin/main" || result.Details["path"] != "openspec/changes/x/" {
		t.Errorf("details = %v, se esperaba la base origin/main y la carpeta openspec/changes/x/", result.Details)
	}
	wantNothingCreated(t, repo, state, "x")
	wantNoRegistryFile(t, repo, "x")
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) || remoteOperation(invocation[1]) {
			t.Errorf("el rechazo no escribe ni toca la red, pero se lanzó %v", invocation)
		}
	}

	t.Run("con --fetch la base alcanza el cambio fusionado", func(t *testing.T) {
		request := archiveRequest(repo)
		request.Fetch = true
		seen := observe(t)

		result := ensureArchive(t, request)

		wantReady(t, result)
		base := onlyRepository(t, result).Base
		if base == nil || *base != (branch.BaseResult{Ref: "origin/main", SHA: advanced, Source: "remote-tracking", Fetched: true}) {
			t.Errorf("base = %+v, se esperaba origin/main en %s con fetched=true", base, advanced)
		}
		wantOnlyArchiveBranch(t, result, "feat/x-archive")
		if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x-archive"); got != advanced {
			t.Errorf("feat/x-archive apunta a %s, se esperaba el commit traído %s", got, advanced)
		}
		fetches := fetchInvocations(*seen)
		if len(fetches) != 1 || !reflect.DeepEqual(fetches[0][1:], []string{"fetch", "origin"}) {
			t.Errorf("se esperaba un único `git fetch origin`: %v", *seen)
		}
		fetchAt := indexOfInvocation(*seen, func(invocation []string) bool { return invocation[1] == "fetch" })
		writeAt := indexOfInvocation(*seen, isWriteInvocation)
		if fetchAt < 0 || writeAt < 0 || fetchAt > writeAt {
			t.Errorf("el fetch (%d) va antes de toda escritura (%d)", fetchAt, writeAt)
		}
		for _, invocation := range *seen {
			if invocation[1] == "push" || invocation[1] == "pull" || invocation[1] == "ls-remote" {
				t.Errorf("ninguna operación remota distinta de `fetch origin`: %v", invocation)
			}
		}
	})
}

// REQ-25.10, supuesto S-8: con `--fetch` y un remoto inalcanzable, `fetch_failed` y no se crea
// nada, ni registro.
func TestEnsureArchiveFetchFailureLeavesNoEffects(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe"))
	state := repositoryState(t, repo)
	request := archiveRequest(repo)
	request.Fetch = true

	result := ensureArchive(t, request)

	wantRefusal(t, result, branch.CodeFetchFailed, branch.StatusRefused)
	if !strings.Contains(result.Continuation, "axiom branch ensure --archive --change x --fetch") {
		t.Errorf("la continuación %q debe repetir la orden de archivado", result.Continuation)
	}
	wantNothingCreated(t, repo, state, "x")
	wantNoRegistryFile(t, repo, "x")
}

// REQ-25.45, G-13: si la base ya contiene el cambio archivado (`changes/archive/<fecha>-<cambio>/`
// y no `changes/<cambio>/`), el mismo código `change_not_integrated` con un mensaje que nombra la
// entrada archivada y dice que no hay nada que archivar; una carpeta de otro cambio o sin fecha no
// cuenta como la entrada archivada de este.
func TestEnsureArchiveNamesTheArchivedEntryWhenTheBaseAlreadyHoldsTheArchivedChange(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	for _, entry := range []string{"2026-10-01-x", "2026-09-30-xy", "x-2026-10-01", "x"} {
		writeFile(t, filepath.Join(repo, "openspec", "changes", "archive", entry, "proposal.md"), "archivado\n")
	}
	runGit(t, repo, "add", "-A")
	commit(t, repo, "archivados")
	state := repositoryState(t, repo)

	result := ensureArchive(t, archiveRequest(repo))

	wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
	for _, want := range []string{"openspec/changes/archive/2026-10-01-x/", "nada que archivar", "main"} {
		if !strings.Contains(result.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", result.Message, want)
		}
	}
	if result.Details["archivedEntry"] != "openspec/changes/archive/2026-10-01-x/" {
		t.Errorf("details = %v, se esperaba la entrada archivada", result.Details)
	}
	if want := "axiom branch status --change x --cwd " + pathquote.Quote(repo); !strings.Contains(result.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
	}
	wantNothingCreated(t, repo, state, "x")
	wantNoRegistryFile(t, repo, "x")

	t.Run("sin entrada archivada de este cambio es el rechazo ordinario", func(t *testing.T) {
		result := ensureArchive(t, EnsureRequest{CWD: repo, Change: "y", Lane: branch.LaneSDD, Archive: true})

		wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
		if strings.Contains(result.Message, "nada que archivar") || result.Details["archivedEntry"] != "" {
			t.Errorf("el cambio y no está archivado: %q %v", result.Message, result.Details)
		}
		if !strings.Contains(result.Continuation, "--fetch") {
			t.Errorf("la continuación %q debe sugerir --fetch", result.Continuation)
		}
	})
}

// REQ-25.45 (escenarios «Multirepo no tiene rama de archivado» y «El carril ODD no tiene rama de
// archivado»): `archive_branch_not_applicable` sin crear nada, también con un hub en `ask`: la
// aplicabilidad se decide antes de preguntar por el modo.
func TestEnsureArchiveIsNotApplicableToMultirepoOrTheODDLane(t *testing.T) {
	t.Parallel()
	t.Run("multirepo", func(t *testing.T) {
		master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
		sealChangeAt(t, specs, "x", "backend")
		backend := filepath.Join(master, "repo-backend")
		specsState, backendState := repositoryState(t, specs), repositoryState(t, backend)

		result := ensureWith(t, hubAsk, archiveRequest(master))

		wantRefusal(t, result, branch.CodeArchiveBranchNotApplicable, branch.StatusRefused)
		if !strings.Contains(result.Message, "multirepo") {
			t.Errorf("el mensaje %q debe nombrar el multirepo", result.Message)
		}
		if got := repositoryState(t, specs); got != specsState {
			t.Errorf("el repositorio de specs cambió:\n%s\n---\n%s", specsState, got)
		}
		if got := repositoryState(t, backend); got != backendState {
			t.Errorf("el repositorio de código cambió:\n%s\n---\n%s", backendState, got)
		}
		if _, err := os.Stat(branch.DefaultWorktreeRoot(canonicalDirectory(t, master))); err == nil {
			t.Error("el rechazo no debe crear la raíz de worktrees")
		}
	})
	t.Run("carril ODD", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		mergeChange(t, repo, "mi-feature")
		state := repositoryState(t, repo)

		result := ensureWith(t, hubAsk, EnsureRequest{CWD: repo, Change: "mi-feature", Lane: branch.LaneODD, Archive: true})

		wantRefusal(t, result, branch.CodeArchiveBranchNotApplicable, branch.StatusRefused)
		if !strings.Contains(result.Message, "ODD") {
			t.Errorf("el mensaje %q debe nombrar el carril ODD", result.Message)
		}
		wantNothingCreated(t, repo, state, "mi-feature")
		wantNoRegistryFile(t, repo, "mi-feature")
	})
}

// REQ-25.12, supuesto S-16 (escenario «`--branch` y `--base` no se admiten con `--archive`»):
// `plan_conflict` sin crear nada ni registrar nada, también con un hub en `ask`; un `--prefix`
// inválido es `invalid_name` como en `ensure`.
func TestEnsureArchiveRejectsBranchAndBaseAndAnInvalidPrefix(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	state := repositoryState(t, repo)

	for name, tc := range map[string]struct {
		exceptions branch.Exceptions
		code       branch.Code
		want       string
	}{
		"--branch":          {branch.Exceptions{Branch: "doc/x"}, branch.CodePlanConflict, "--branch"},
		"--base":            {branch.Exceptions{Base: "release/2"}, branch.CodePlanConflict, "--base"},
		"--prefix inválido": {branch.Exceptions{Prefix: "a b"}, branch.CodeInvalidName, "--prefix"},
	} {
		t.Run(name, func(t *testing.T) {
			request := archiveRequest(repo)
			request.Exceptions = tc.exceptions

			result := ensureWith(t, hubAsk, request)

			wantRefusal(t, result, tc.code, branch.StatusRefused)
			if !strings.Contains(result.Message, tc.want) {
				t.Errorf("el mensaje %q debe nombrar %s", result.Message, tc.want)
			}
			wantNothingCreated(t, repo, state, "x")
			wantNoRegistryFile(t, repo, "x")
		})
	}
}

// REQ-25.12 (escenario «`--prefix` fija el prefijo de la rama de archivado») y REQ-25.45: con
// `--prefix doc` la rama es `doc/x-archive` con su worktree en `<raíz>/x-archive/<repo>`, la rama
// del cambio no se crea ni se contradice, el registro guarda el prefijo de archivado y repetir
// (con el mismo prefijo o sin él) es idempotente; otro prefijo daría otro nombre y es
// `plan_conflict` sin tocar nada.
func TestEnsureArchivePrefixNamesTheBranchAndIsRegistered(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	request := archiveRequest(repo)
	request.Exceptions = branch.Exceptions{Prefix: "doc"}

	result := ensureArchive(t, request)

	wantReady(t, result)
	archive := wantOnlyArchiveBranch(t, result, "doc/x-archive")
	wantSameDirectory(t, "worktree", archive.Worktree, archiveWorktree(t, repo))
	if runGit(t, repo, "branch", "--list", "feat/x") != "" || runGit(t, repo, "branch", "--list", "feat/x-archive") != "" {
		t.Error("`--prefix` solo nombra la rama de archivado: no se crea ninguna rama con el prefijo del cambio")
	}
	record := loadRecord(t, repo, "x")
	if record.ArchivePrefix != "doc" || record.Exceptions != (branch.Exceptions{}) {
		t.Errorf("registro: archivePrefix %q, excepciones %+v; se esperaba doc y ninguna excepción del cambio", record.ArchivePrefix, record.Exceptions)
	}
	first := resultJSON(t, result)
	state := repositoryState(t, repo)

	t.Run("repetir con el mismo prefijo o sin él es idempotente", func(t *testing.T) {
		for name, exceptions := range map[string]branch.Exceptions{"mismo prefijo": {Prefix: "doc"}, "sin prefijo": {}} {
			again := archiveRequest(repo)
			again.Exceptions = exceptions

			result := ensureArchive(t, again)

			wantReady(t, result)
			if got := resultJSON(t, result); got != first {
				t.Errorf("%s: el JSON cambió:\n%s\n---\n%s", name, first, got)
			}
			if got := repositoryState(t, repo); got != state {
				t.Errorf("%s: cambió el repositorio", name)
			}
		}
	})
	t.Run("otro prefijo es plan_conflict y no toca nada", func(t *testing.T) {
		other := archiveRequest(repo)
		other.Exceptions = branch.Exceptions{Prefix: "otro"}

		result := ensureArchive(t, other)

		wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
		for _, want := range []string{"doc/x-archive", "otro/x-archive"} {
			if !strings.Contains(result.Message, want) {
				t.Errorf("el mensaje %q debe nombrar %s", result.Message, want)
			}
		}
		if want := "axiom branch ensure --archive --change x --cwd " + pathquote.Quote(repo); !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
		if got := repositoryState(t, repo); got != state {
			t.Error("el rechazo cambió el repositorio")
		}
		if got := loadRecord(t, repo, "x"); got.ArchivePrefix != "doc" {
			t.Errorf("el registro cambió: %+v", got)
		}
	})
}

// REQ-25.9 (escenarios «Modo `in_place` hace checkout de la rama de archivado» y «Modo `in_place`
// con árbol sucio se rechaza») y REQ-25.8: con `in_place` y el árbol sucio, `dirty_worktree` sin
// modificar nada (con una continuación que repite `--archive`); con el árbol limpio, `git switch`
// a la rama de archivado y ningún worktree; ya en ella con el árbol sucio es un no-op.
func TestEnsureArchiveInPlaceSwitchesToTheArchiveBranchAndRefusesADirtyTree(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	request := archiveRequest(repo)
	request.Mode = branch.ModeInPlace
	writeFile(t, filepath.Join(repo, "suelto.txt"), "sin añadir\n")
	before := inPlaceState(t, repo)

	result := ensureArchive(t, request)

	wantRefusal(t, result, branch.CodeDirtyWorktree, branch.StatusRefused)
	if !strings.Contains(result.Message, "suelto.txt") || !strings.Contains(result.Message, "feat/x-archive") {
		t.Errorf("el mensaje %q debe nombrar el fichero y la rama de archivado", result.Message)
	}
	if want := "axiom branch ensure --archive --change x --mode in_place --cwd " + pathquote.Quote(repo); !strings.Contains(result.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
	}
	if after := inPlaceState(t, repo); after != before {
		t.Errorf("el rechazo cambió el repositorio:\nantes:\n%s\ndespués:\n%s", before, after)
	}
	wantNoRegistryFile(t, repo, "x")

	resetTree(t, repo)
	result = ensureArchive(t, request)

	wantReady(t, result)
	wantMode(t, result, branch.ModeInPlace, branch.ModeSourceFlag)
	archive := wantOnlyArchiveBranch(t, result, "feat/x-archive")
	if archive.Worktree == "" || !pathidentity.SameDirectory(archive.Worktree, repo) {
		t.Errorf("en in_place la rama de archivado está desplegada en el propio repositorio: %q", archive.Worktree)
	}
	if got := currentBranch(t, repo); got != "feat/x-archive" {
		t.Errorf("el repositorio está en %s, se esperaba feat/x-archive", got)
	}
	if listed := runGit(t, repo, "worktree", "list", "--porcelain"); strings.Count(listed, "worktree ") != 1 {
		t.Errorf("in_place no crea worktrees:\n%s", listed)
	}
	record := loadRecord(t, repo, "x")
	if record.Mode != branch.ModeInPlace || len(record.Repositories[0].Branches) != 1 || !record.Repositories[0].Branches[0].InPlace || record.Repositories[0].Branches[0].Kind != branch.KindArchive {
		t.Errorf("registro = %+v, se esperaba la rama de archivado inPlace", record)
	}

	writeFile(t, filepath.Join(repo, "suelto.txt"), "sin añadir\n")
	seen := observe(t)
	again := ensureArchive(t, request)
	wantReady(t, again)
	wantNoWrites(t, *seen, "ya en la rama de archivado con el árbol sucio es un no-op")
}

// REQ-25.8 (escenario de la respuesta a `ask` reutilizada) y D-25 paso 6: con un hub en `ask` y
// sin respuesta, `decision_required` cuya continuación repite `--archive` y sin crear nada; con el
// cambio ya registrado (respuesta `in_place`), `--archive` la reutiliza sin preguntar y conserva
// su origen; sin registro, `--mode` es la respuesta y se registra.
func TestEnsureArchiveReusesTheRegisteredModeAndAsksOnlyWithoutAnAnswer(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	state := inPlaceState(t, repo)

	t.Run("sin respuesta registrada pregunta y no crea nada", func(t *testing.T) {
		result := ensureWith(t, hubAsk, archiveRequest(repo))

		if result.Status != branch.StatusDecisionRequired || result.Code != branch.CodeDecisionRequired || result.Decision == nil || result.ExitCode() != branch.ExitRefused {
			t.Fatalf("status %q, código %q, decisión %+v; se esperaba decision_required", result.Status, result.Code, result.Decision)
		}
		for _, choice := range result.Decision.Choices {
			if want := "axiom branch ensure --archive --change x --mode " + choice.Answer + " --cwd " + pathquote.Quote(repo); choice.Invocation != want {
				t.Errorf("invocación de %s = %q, se esperaba %q", choice.Answer, choice.Invocation, want)
			}
		}
		if after := inPlaceState(t, repo); after != state {
			t.Errorf("la pregunta cambió el repositorio:\n%s\n---\n%s", state, after)
		}
		wantNoRegistryFile(t, repo, "x")
	})
	t.Run("con el cambio registrado reutiliza la respuesta", func(t *testing.T) {
		record := registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange})
		record.Mode, record.ModeSource = branch.ModeInPlace, branch.ModeSourceAnswer
		writeRegistry(t, commonDir(repo), record)

		result := ensureWith(t, hubAsk, archiveRequest(repo))

		wantReady(t, result)
		wantMode(t, result, branch.ModeInPlace, branch.ModeSourceAnswer)
		wantOnlyArchiveBranch(t, result, "feat/x-archive")
		if got := currentBranch(t, repo); got != "feat/x-archive" {
			t.Errorf("el repositorio está en %s, se esperaba feat/x-archive", got)
		}
		saved := loadRecord(t, repo, "x")
		if saved.Mode != branch.ModeInPlace || saved.ModeSource != branch.ModeSourceAnswer {
			t.Errorf("modo registrado = %q (%q), debía conservarse", saved.Mode, saved.ModeSource)
		}
		var kinds []branch.BranchKind
		for _, recorded := range saved.Repositories[0].Branches {
			kinds = append(kinds, recorded.Kind)
		}
		if !reflect.DeepEqual(kinds, []branch.BranchKind{branch.KindChange, branch.KindArchive}) {
			t.Errorf("el registro conserva la rama del cambio y añade la de archivado: %v", kinds)
		}
		// Un `--mode` distinto del registrado es plan_conflict también con --archive.
		other := archiveRequest(repo)
		other.Mode = branch.ModeWorktree
		wantRefusal(t, ensureWith(t, hubAsk, other), branch.CodePlanConflict, branch.StatusRefused)
	})
	t.Run("sin registro, --mode es la respuesta a ask", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		mergeChange(t, repo, "x")
		request := archiveRequest(repo)
		request.Mode = branch.ModeInPlace

		result := ensureWith(t, hubAsk, request)

		wantReady(t, result)
		wantMode(t, result, branch.ModeInPlace, branch.ModeSourceAnswer)
		if record := loadRecord(t, repo, "x"); record.Mode != branch.ModeInPlace || record.ModeSource != branch.ModeSourceAnswer {
			t.Errorf("registro = %q (%q), se esperaba in_place con origen answer", record.Mode, record.ModeSource)
		}
	})
}

// REQ-25.7, REQ-25.14 y REQ-25.12: `--archive` no altera las ramas del cambio ni las de rol, y el
// `ensure` ordinario posterior (aquí con `--branch`, que un registro con solo la rama de archivado
// no contradice) conserva la rama de archivado y su prefijo en el registro; `status` las muestra
// juntas, con la de archivado nombrada por el registro aunque la del cambio use otro nombre.
func TestEnsureAfterArchiveKeepsTheArchiveBranchInTheRegistryAndStatus(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	request := archiveRequest(repo)
	request.Exceptions = branch.Exceptions{Prefix: "doc"}
	wantReady(t, ensureArchive(t, request))
	archiveHead := runGit(t, repo, "rev-parse", "refs/heads/doc/x-archive")

	result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD, Exceptions: branch.Exceptions{Branch: "inc/x"}})

	wantReady(t, result)
	if got := branchNamesOf(onlyRepository(t, result)); !reflect.DeepEqual(got, []string{"inc/x"}) {
		t.Errorf("el `ensure` ordinario describe solo el plan del cambio: %v", got)
	}
	record := loadRecord(t, repo, "x")
	if record.Exceptions.Branch != "inc/x" || record.ArchivePrefix != "doc" {
		t.Errorf("registro: excepciones %+v, prefijo de archivado %q", record.Exceptions, record.ArchivePrefix)
	}
	var names []string
	for _, recorded := range record.Repositories[0].Branches {
		names = append(names, recorded.Name+":"+string(recorded.Kind))
	}
	if want := []string{"inc/x:change", "doc/x-archive:archive"}; !reflect.DeepEqual(names, want) {
		t.Errorf("ramas registradas = %v, se esperaba %v", names, want)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/doc/x-archive"); got != archiveHead {
		t.Errorf("el `ensure` ordinario movió la rama de archivado: %s -> %s", archiveHead, got)
	}

	listed := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, listed)
	if got := branchNamesOf(onlyRepository(t, listed)); !reflect.DeepEqual(got, []string{"inc/x", "doc/x-archive"}) {
		t.Errorf("status = %v, se esperaba la rama del cambio y la de archivado", got)
	}
	wantSameDirectory(t, "worktree de la rama de archivado", branchByName(t, onlyRepository(t, listed), "doc/x-archive").Worktree, archiveWorktree(t, repo))

	// Una repetición de `--archive` sigue idempotente aunque la rama del cambio use otro nombre.
	state := repositoryState(t, repo)
	wantReady(t, ensureArchive(t, archiveRequest(repo)))
	if got := repositoryState(t, repo); got != state {
		t.Error("repetir --archive no debe cambiar el repositorio")
	}
}

// design D-25 paso 3: sin `--prefix`, la rama de archivado deriva de la rama del cambio
// registrada (`--branch inc/inc-25-x` da `inc/inc-25-x-archive`) y la de la política cuando no
// hay registro.
func TestEnsureArchiveDerivesTheNameFromTheRegisteredChangeBranch(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	record := registeredRecord(repo, branch.RecordBranch{Name: "inc/inc-25-x", Kind: branch.KindChange})
	record.Exceptions = branch.Exceptions{Branch: "inc/inc-25-x"}
	record.Mode, record.ModeSource = branch.ModeInPlace, branch.ModeSourceFlag
	writeRegistry(t, commonDir(repo), record)

	result := ensureArchive(t, archiveRequest(repo))

	wantReady(t, result)
	wantOnlyArchiveBranch(t, result, "inc/inc-25-x-archive")
	if saved := loadRecord(t, repo, "x"); saved.Exceptions.Branch != "inc/inc-25-x" || len(saved.Repositories[0].Branches) != 2 {
		t.Errorf("registro = %+v, se esperaba la excepción del cambio intacta y dos ramas", saved)
	}
}

// design D-25 paso 4 y D-06: con `--base` fijada en el cambio registrado, la rama de archivado
// sigue saliendo de la rama de integración.
func TestEnsureArchiveIgnoresTheChangeBaseExceptionAndUsesTheIntegration(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	runGit(t, repo, "branch", "release/2", "main")
	mergeChange(t, repo, "x")
	record := registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange})
	record.Exceptions = branch.Exceptions{Base: "release/2"}
	record.Mode, record.ModeSource = branch.ModeInPlace, branch.ModeSourceFlag
	writeRegistry(t, commonDir(repo), record)
	integration := runGit(t, repo, "rev-parse", "main")

	result := ensureArchive(t, archiveRequest(repo))

	wantReady(t, result)
	if base := onlyRepository(t, result).Base; base == nil || base.Ref != "main" || base.SHA != integration {
		t.Errorf("base = %+v, se esperaba la rama de integración main en %s", base, integration)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x-archive"); got != integration {
		t.Errorf("feat/x-archive apunta a %s, se esperaba %s", got, integration)
	}
	if saved := loadRecord(t, repo, "x"); saved.Exceptions.Base != "release/2" {
		t.Errorf("la excepción del cambio debía conservarse: %+v", saved.Exceptions)
	}
}

// REQ-25.7 y matriz de amenazas «Estado del push»: ninguna invocación de `--archive` contiene
// `push` y el puerto de escritura no tiene ningún método que publique.
func TestEnsureArchiveNeverPublishes(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeChange(t, repo, "x")
	remote := remoteWithMain(t, repo)
	seen := observe(t)
	request := archiveRequest(repo)
	request.Mode = branch.ModeInPlace

	wantReady(t, ensureArchive(t, request))

	for _, invocation := range *seen {
		if invocation[1] == "push" {
			t.Errorf("`ensure --archive` nunca publica: %v", invocation)
		}
	}
	if remote := runGit(t, remote, "branch", "--list"); strings.Contains(remote, "archive") {
		t.Errorf("la rama de archivado no debe llegar al remoto:\n%s", remote)
	}
}

// REQ-25.7 (escenario «`--archive` no altera las ramas del cambio») y REQ-25.4: con las ramas
// `feat/x`, `feat/x-core` y `feat/x-qa` ya creadas y el PR fusionado, `ensure --archive` crea
// `feat/x-archive` y el commit al que apunta cada una de las ramas anteriores no cambia, ni
// aparecen otras ramas; el plan de archivado tampoco se ve afectado por los roles sellados.
func TestEnsureArchiveDoesNotAlterTheChangeAndRoleBranches(t *testing.T) {
	t.Parallel()
	repo := twoRolesProject(t)
	for _, name := range []string{"feat/x", "feat/x-core", "feat/x-qa"} {
		runGit(t, repo, "branch", name, "main")
	}
	mergeChange(t, repo, "x")
	heads := runGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads/feat")
	request := archiveRequest(repo)
	request.Mode = branch.ModeInPlace

	result := ensureArchive(t, request)

	wantReady(t, result)
	wantOnlyArchiveBranch(t, result, "feat/x-archive")
	after := runGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads/feat")
	archiveHead := runGit(t, repo, "rev-parse", "refs/heads/feat/x-archive")
	for _, line := range strings.Split(heads, "\n") {
		if !strings.Contains(after, line) {
			t.Errorf("la rama %q cambió de commit:\nantes:\n%s\ndespués:\n%s", line, heads, after)
		}
	}
	if want := runGit(t, repo, "rev-parse", "main"); archiveHead != want {
		t.Errorf("feat/x-archive apunta a %s, se esperaba la integración %s", archiveHead, want)
	}
	if got := strings.Count(after, "\n") + 1; got != 4 {
		t.Errorf("se esperaban las tres ramas anteriores y la de archivado:\n%s", after)
	}
	record := loadRecord(t, repo, "x")
	if len(record.Repositories[0].Branches) != 1 || record.Repositories[0].Branches[0].Kind != branch.KindArchive {
		t.Errorf("registro = %+v, se esperaba solo la rama de archivado", record.Repositories)
	}
}

// REQ-25.14 y REQ-25.45: `ensure --archive` ejecutado desde el worktree enlazado del cambio (la
// raíz git es la del worktree, el git common dir es el del checkout principal) encuentra el
// registro, deriva el nombre de la rama del cambio registrada, comprueba la integración sobre el
// árbol de la base y crea la rama de archivado y su worktree junto a los de la raíz de worktrees
// del proyecto, sin tocar la rama del cambio.
func TestEnsureArchiveFromTheLinkedWorktreeOfTheChange(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureOf(t, repo, "x"))
	mergeChange(t, repo, "x")
	linked := plannedWorktree(t, repo, "x", filepath.Base(repo))
	changeHead := runGit(t, repo, "rev-parse", "refs/heads/feat/x")

	result := ensureArchive(t, archiveRequest(linked))

	wantReady(t, result)
	archive := wantOnlyArchiveBranch(t, result, "feat/x-archive")
	wantSameDirectory(t, "worktree de la rama de archivado", archive.Worktree, archiveWorktree(t, repo))
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != changeHead {
		t.Errorf("la rama del cambio se movió: %s -> %s", changeHead, got)
	}
	record := loadRecord(t, repo, "x")
	var names []string
	for _, recorded := range record.Repositories[0].Branches {
		names = append(names, recorded.Name+":"+string(recorded.Kind))
	}
	if want := []string{"feat/x:change", "feat/x-archive:archive"}; !reflect.DeepEqual(names, want) {
		t.Errorf("ramas registradas = %v, se esperaba %v", names, want)
	}
}
