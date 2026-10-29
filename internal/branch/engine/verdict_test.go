package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Pruebas de `VerdictFor` y de la guarda A con git real, entorno aislado y sin red (REQ-25.18,
// REQ-25.21, REQ-25.42, design D-10 y D-11). `VerdictFor` responde leyendo ficheros: ninguna
// de estas pruebas observa un solo proceso de git durante la consulta.

func verdictFor(t *testing.T, q VerdictQuery) branch.Verdict {
	t.Helper()
	if q.Lane == "" {
		q.Lane = branch.LaneSDD
	}
	if q.Context == "" {
		q.Context = branch.ContextOrdinary
	}
	if q.Consultant == "" {
		q.Consultant = branch.ConsultantWrite
	}
	verdict, err := Default().VerdictFor(context.Background(), q)
	if err != nil {
		t.Fatalf("VerdictFor(%+v): %v", q, err)
	}
	return verdict
}

func wantKind(t *testing.T, got branch.Verdict, want branch.VerdictKind) {
	t.Helper()
	if got.Kind != want {
		t.Fatalf("veredicto %q (%s), se esperaba %q", got.Kind, got.Detail, want)
	}
}

// changeRootOf es la carpeta del cambio x bajo un repositorio de specs embebido.
func changeRootOf(repo string) string {
	return filepath.Join(repo, "openspec", "changes", "x")
}

// continuationCWD extrae la ruta citada con `--cwd` de una continuación.
func continuationCWD(t *testing.T, continuation string) string {
	t.Helper()
	match := regexp.MustCompile(`--cwd "([^"]*)"`).FindStringSubmatch(continuation)
	if match == nil {
		t.Fatalf("la continuación %q no cita --cwd con pathquote.Quote", continuation)
	}
	return match[1]
}

// REQ-25.18 y REQ-25.21: sin kickoff, los nombres esperados salen de la política y el HEAD, de
// los ficheros: no se lanza ningún proceso (el observador queda vacío). La prueba mueve HEAD
// entre las dos consultas con git directo, que no pasa por gitcommon.Run ni por el observador.
func TestVerdictForReadsOnlyFilesAndLaunchesNoProcess(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	seen := observe(t)

	onChange := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})
	runGit(t, repo, "switch", "-q", "main")
	onIntegration := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})
	if len(*seen) != 0 {
		t.Fatalf("VerdictFor lanzó procesos de git: %v", *seen)
	}

	wantKind(t, onChange, branch.VerdictCorrect)
	wantKind(t, onIntegration, branch.VerdictWrongBranch)
	if onIntegration.Current != "main" || len(onIntegration.Expected) != 1 || onIntegration.Expected[0] != "feat/x" {
		t.Errorf("rama actual %q y esperada %v, se esperaba main y [feat/x]", onIntegration.Current, onIntegration.Expected)
	}
	if !strings.HasPrefix(onIntegration.Continuation, "axiom branch ensure --cwd ") || !strings.HasSuffix(onIntegration.Continuation, `" --change x`) ||
		!pathidentity.SameDirectory(continuationCWD(t, onIntegration.Continuation), repo) {
		t.Errorf("continuación = %q, se esperaba `axiom branch ensure --cwd %s --change x`", onIntegration.Continuation, repo)
	}
}

// REQ-25.18 (S-7): sin `kickoff.yaml` los nombres salen de la política de `axiom.yaml`, con su
// prefijo y su integración.
func TestVerdictForTakesTheNamesFromThePolicyWhenThereIsNoKickoff(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  integration_branch: develop\n  branch_prefix: inc\n"))
	runGit(t, repo, "switch", "-q", "-c", "inc/x")

	wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"}), branch.VerdictCorrect)

	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	wrong := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})
	wantKind(t, wrong, branch.VerdictWrongBranch)
	if len(wrong.Expected) != 1 || wrong.Expected[0] != "inc/x" {
		t.Errorf("rama esperada = %v, se esperaba [inc/x] por el prefijo de la política", wrong.Expected)
	}
}

// REQ-25.18 y REQ-25.12: el registro local manda sobre la política. Con una excepción `--prefix`
// registrada la rama del cambio es la del registro, vista igual desde el checkout principal y
// desde el worktree enlazado (el registro vive en el common dir); la de archivado deriva de ella.
func TestVerdictForTakesTheNamesFromTheRegistryFromTheMainCheckoutAndFromALinkedWorktree(t *testing.T) {
	repo := embeddedCoreProject(t)
	result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x", Exceptions: branch.Exceptions{Prefix: "fix"}})
	wantReady(t, result)
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	seen := observe(t)

	fromWorktree := verdictFor(t, VerdictQuery{Path: worktree, Change: "x"})
	fromMain := verdictFor(t, VerdictQuery{Path: repo, Change: "x"})
	archive := verdictFor(t, VerdictQuery{Path: worktree, Change: "x", Context: branch.ContextArchive})
	if len(*seen) != 0 {
		t.Fatalf("VerdictFor lanzó procesos de git: %v", *seen)
	}

	wantKind(t, fromWorktree, branch.VerdictCorrect)
	wantKind(t, fromMain, branch.VerdictWrongBranch)
	if len(fromMain.Expected) != 1 || fromMain.Expected[0] != "fix/x" {
		t.Errorf("rama esperada = %v, se esperaba [fix/x] por el registro", fromMain.Expected)
	}
	wantKind(t, archive, branch.VerdictWrongBranch)
	if len(archive.Expected) != 1 || archive.Expected[0] != "fix/x-archive" {
		t.Errorf("rama de archivado esperada = %v, se esperaba [fix/x-archive]", archive.Expected)
	}
	if !strings.HasSuffix(archive.Continuation, "--change x --archive") {
		t.Errorf("la continuación de archivado %q debe terminar en --change x --archive", archive.Continuation)
	}
}

// REQ-25.18 (continuación): desde un worktree enlazado en la rama equivocada la orden de
// `ensure` apunta a la carpeta maestra del checkout principal, no a la del worktree.
func TestVerdictForPointsTheContinuationAtTheMainCheckout(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureOf(t, repo, "x"))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	runGit(t, worktree, "switch", "-q", "-c", "otra")

	verdict := verdictFor(t, VerdictQuery{Path: worktree, Change: "x"})

	wantKind(t, verdict, branch.VerdictWrongBranch)
	if cwd := continuationCWD(t, verdict.Continuation); !pathidentity.SameDirectory(cwd, repo) {
		t.Errorf("--cwd = %q, se esperaba la carpeta maestra del checkout principal %q", cwd, repo)
	}
}

// REQ-25.18: las ramas de rol que se van a sellar (SealRoles) son correctas aunque aún no haya
// kickoff; sin ellas esa misma rama es incorrecta.
func TestVerdictForAcceptsTheRoleBranchesAboutToBeSealed(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	runGit(t, repo, "switch", "-q", "-c", "feat/x-qa")

	wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"}), branch.VerdictWrongBranch)
	withRoles := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x", SealRoles: []string{"core", "qa"}})
	wantKind(t, withRoles, branch.VerdictCorrect)

	// Un único rol no genera ramas de rol (REQ-25.4): `feat/x-qa` sigue siendo ajena.
	oneRole := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x", SealRoles: []string{"qa"}})
	wantKind(t, oneRole, branch.VerdictWrongBranch)
}

// REQ-25.37: un cambio cuyo `kickoff.yaml` existe y no tiene instantánea queda exento, y
// también los sumideros acumulativos por nombre; en los dos casos aunque la rama sea la de
// integración, sin continuación ni aviso.
func TestVerdictForExemptsAChangeWithoutSnapshotAndTheCumulativeSinks(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	sealChange(t, repo, "x", "core")
	commitAll(t, repo, "kickoff sin instantánea")

	sealed := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})
	wantKind(t, sealed, branch.VerdictNotApplicable)
	if sealed.Continuation != "" || sealed.Warning != "" || sealed.Detail == "" {
		t.Errorf("la exención explica el motivo, sin continuación ni aviso: %+v", sealed)
	}
	for _, change := range []string{"e2e-cumulative", "cumulative-core", "cumulative-frontend-2"} {
		sink := verdictFor(t, VerdictQuery{Path: filepath.Join(repo, "openspec", "changes", change), Change: change})
		wantKind(t, sink, branch.VerdictNotApplicable)
		if sink.Continuation != "" || sink.Warning != "" {
			t.Errorf("el sumidero %s no lleva continuación ni aviso: %+v", change, sink)
		}
	}
	// Un nombre que solo se parece al de un sumidero no está exento.
	wantKind(t, verdictFor(t, VerdictQuery{Path: filepath.Join(repo, "openspec", "changes", "cumulativo"), Change: "cumulativo"}), branch.VerdictWrongBranch)
}

// Un cambio sin kickoff cuyo nombre no forma un nombre de rama válido (aquí, con un espacio) no
// tiene rama que comprobar: `not_applicable` con aviso en castellano, sin continuación y sin
// error, aunque la rama actual sea la de integración. Con un nombre válido sigue bloqueando.
func TestVerdictForAnInvalidChangeNameWithoutKickoffIsNotApplicableWithAWarning(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
		verdict := verdictFor(t, VerdictQuery{Path: filepath.Join(repo, "openspec", "changes", "phase one"), Change: "phase one", Lane: lane})

		wantKind(t, verdict, branch.VerdictNotApplicable)
		if verdict.Continuation != "" || verdict.Detail == "" {
			t.Errorf("carril %s: la exención explica el motivo y no lleva continuación: %+v", lane, verdict)
		}
		for _, want := range []string{`"phase one"`, "no es un nombre de rama válido", "espacios", "no se comprueba la rama"} {
			if !strings.Contains(verdict.Warning, want) {
				t.Errorf("carril %s: el aviso %q debe contener %q", lane, verdict.Warning, want)
			}
		}
	}
	wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"}), branch.VerdictWrongBranch)
}

// REQ-25.18 y REQ-25.42: fuera de un repositorio git la consulta no bloquea y avisa, sin
// buscar nada más.
func TestVerdictForOutsideGitIsNotApplicableWithAWarning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "openspec", "changes", "x"), 0o755); err != nil {
		t.Fatal(err)
	}

	verdict := verdictFor(t, VerdictQuery{Path: changeRootOf(dir), Change: "x"})

	wantKind(t, verdict, branch.VerdictNotApplicable)
	if !strings.Contains(verdict.Warning, "git") || verdict.Continuation != "" {
		t.Errorf("fuera de git se avisa y no se bloquea: %+v", verdict)
	}
}

// REQ-25.21 (HEAD ilegible): un `HEAD` con la forma del backend reftable, que no se puede leer
// sin git, es `not_applicable` con aviso: la consulta no bloquea.
func TestVerdictForAReftableHeadIsNotApplicableWithAWarning(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	writeFile(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/.invalid\n")

	verdict := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})

	wantKind(t, verdict, branch.VerdictNotApplicable)
	if !strings.Contains(verdict.Warning, "HEAD") {
		t.Errorf("el aviso debe nombrar el HEAD ilegible: %q", verdict.Warning)
	}
}

// REQ-25.18: un HEAD desacoplado es `wrong_branch` y el veredicto lo dice.
func TestVerdictForADetachedHeadIsWrong(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	runGit(t, repo, "switch", "-q", "--detach", "HEAD")

	verdict := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})

	wantKind(t, verdict, branch.VerdictWrongBranch)
	if !verdict.Detached || !strings.Contains(verdict.Detail, "desacoplado") {
		t.Errorf("el veredicto debe decir que el HEAD está desacoplado: %+v", verdict)
	}
}

// REQ-25.18, escenario «Los cambios activos en la rama de integración no alteran el
// veredicto»: un cambio `y` fusionado en `main` no hace correcta a `main` para `x` ni incorrecta
// a `feat/x`.
func TestVerdictForIgnoresTheActiveChangesOfTheIntegrationBranch(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	writeFile(t, filepath.Join(repo, "openspec", "changes", "y", "proposal.md"), "# y\n")
	commitAll(t, repo, "cambio y fusionado")

	wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"}), branch.VerdictWrongBranch)
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"}), branch.VerdictCorrect)
}

// REQ-25.18 (archivado): el registro guarda el prefijo con el que se creó la rama de archivado y
// el veredicto de archivado lo usa; sin él, la rama de archivado es `<rama del cambio>-archive`.
func TestVerdictForTakesTheArchiveBranchFromTheRegistryPrefix(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	record := registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, InPlace: true})
	record.ArchivePrefix = "doc"
	if err := registry.New().Save(commonDir(repo), record); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "switch", "-q", "-c", "doc/x-archive")

	wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x", Context: branch.ContextArchive}), branch.VerdictCorrect)
	ordinary := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})
	wantKind(t, ordinary, branch.VerdictWrongBranch)
	if len(ordinary.Expected) != 1 || ordinary.Expected[0] != "feat/x" {
		t.Errorf("rama esperada = %v, se esperaba [feat/x]", ordinary.Expected)
	}
}

// REQ-25.18 y REQ-25.38 (multirepo): el repositorio de specs exige su integración, también en
// el contexto de archivado, y una rama de rol es incorrecta; el de código solo responde al
// consultante `apply`.
func TestVerdictForAMultirepoChecksTheSpecsRepositoryAndTheCodeOnlyWhenApplying(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	backend := filepath.Join(master, "repo-backend")
	specsChange := filepath.Join(specs, "openspec", "changes", "x")
	seen := observe(t)

	for _, context := range []branch.Context{branch.ContextOrdinary, branch.ContextArchive} {
		wantKind(t, verdictFor(t, VerdictQuery{Path: specsChange, Change: "x", Context: context}), branch.VerdictCorrect)
	}
	if len(*seen) != 0 {
		t.Fatalf("VerdictFor lanzó procesos de git: %v", *seen)
	}

	runGit(t, specs, "switch", "-q", "-c", "feat/x-backend")
	for _, context := range []branch.Context{branch.ContextOrdinary, branch.ContextArchive} {
		verdict := verdictFor(t, VerdictQuery{Path: specsChange, Change: "x", Context: context})
		wantKind(t, verdict, branch.VerdictWrongBranch)
		if len(verdict.Expected) != 1 || verdict.Expected[0] != "main" {
			t.Errorf("contexto %s: esperada %v, se esperaba [main]", context, verdict.Expected)
		}
	}

	// repo-backend sigue en main: solo es incorrecto cuando se aplica una tarea que lo toca.
	for _, consultant := range []branch.Consultant{branch.ConsultantWrite, branch.ConsultantStatus, branch.ConsultantGuard} {
		wantKind(t, verdictFor(t, VerdictQuery{Path: backend, Change: "x", Consultant: consultant, SealRoles: []string{"backend"}}), branch.VerdictNotApplicable)
	}
	applying := verdictFor(t, VerdictQuery{Path: backend, Change: "x", Consultant: branch.ConsultantApply, SealRoles: []string{"backend"}})
	wantKind(t, applying, branch.VerdictWrongBranch)
	if len(applying.Expected) != 1 || applying.Expected[0] != "feat/x" || applying.Repo != "repo-backend" {
		t.Errorf("repositorio %q, esperada %v; se esperaba repo-backend con [feat/x]", applying.Repo, applying.Expected)
	}
	runGit(t, backend, "switch", "-q", "-c", "feat/x")
	wantKind(t, verdictFor(t, VerdictQuery{Path: backend, Change: "x", Consultant: branch.ConsultantApply, SealRoles: []string{"backend"}}), branch.VerdictCorrect)
}

// El carril ODD no tiene kickoff, ni instantánea ni archive: el contexto de archivado se
// evalúa como el ordinario, la rama es `<prefijo>/<feature>` y la continuación repite el carril.
func TestVerdictForTheODDLaneNeedsNoKickoffAndRepeatsTheLane(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	odd := VerdictQuery{Path: filepath.Join(repo, "odd", "tasks", "x.md"), Change: "x", Lane: branch.LaneODD}

	wrong := verdictFor(t, odd)
	wantKind(t, wrong, branch.VerdictWrongBranch)
	if !strings.HasSuffix(wrong.Continuation, "--change x --lane odd") {
		t.Errorf("la continuación %q debe repetir --lane odd", wrong.Continuation)
	}
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	odd.Context = branch.ContextArchive
	wantKind(t, verdictFor(t, odd), branch.VerdictCorrect)
}

// REQ-25.42 y D-10: el `error` se reserva para los fallos internos (configuración, kickoff o
// registro ilegibles) y es un rechazo tipado `config_invalid` con su continuación.
func TestVerdictForReportsAnUnreadableConfigurationKickoffOrRegistryAsConfigInvalid(t *testing.T) {
	t.Parallel()
	corrupt := map[string]func(t *testing.T) (repo string){
		"configuración ilegible": func(t *testing.T) string { return newProject(t, "workspace: [sin cerrar\n") },
		"prefijo inválido": func(t *testing.T) string {
			return newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  branch_prefix: \"mi prefijo//\"\n"))
		},
		"kickoff ilegible": func(t *testing.T) string {
			repo := embeddedCoreProject(t)
			writeFile(t, filepath.Join(changeRootOf(repo), "kickoff.yaml"), "schema: [sin cerrar\n")
			return repo
		},
		"registro ilegible": func(t *testing.T) string {
			repo := embeddedCoreProject(t)
			writeFile(t, recordFile(commonDir(repo), "sdd", "x"), "{ no es json\n")
			return repo
		},
	}
	for name, setup := range corrupt {
		t.Run(name, func(t *testing.T) {
			repo := setup(t)
			_, err := Default().VerdictFor(context.Background(), VerdictQuery{Path: changeRootOf(repo), Change: "x", Lane: branch.LaneSDD})
			var refusal *branch.RefusalError
			if !errors.As(err, &refusal) || refusal.Code != branch.CodeConfigInvalid || refusal.Continuation == "" {
				t.Fatalf("error = %v, se esperaba un rechazo tipado config_invalid con continuación", err)
			}
		})
	}
}

// REQ-25.19 y D-11: la guarda A deja pasar `correct` y `not_applicable` devolviendo el aviso, se
// niega con `wrong_branch` (rechazo tipado con la continuación) y falla en cerrado ante un
// fallo interno, con `config_invalid`.
func TestRequireChangeBranchAllowsCorrectAndNotApplicableAndRefusesWrongBranch(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	target := WriteTarget{ChangeRoot: changeRootOf(repo), Change: "x", Lane: branch.LaneSDD, Context: branch.ContextOrdinary, Op: "kickoff seal"}

	_, err := Default().RequireChangeBranch(context.Background(), target)
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeWrongBranch {
		t.Fatalf("en main: error = %v, se esperaba el rechazo tipado wrong_branch", err)
	}
	for _, want := range []string{"«main»", "«feat/x»", "axiom branch ensure --cwd ", `" --change x`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo %q debe contener %q", err.Error(), want)
		}
	}
	if cwd := continuationCWD(t, err.Error()); !pathidentity.SameDirectory(cwd, repo) {
		t.Errorf("--cwd = %q, se esperaba la carpeta maestra %q", cwd, repo)
	}

	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	if warning, err := Default().RequireChangeBranch(context.Background(), target); err != nil || warning != "" {
		t.Errorf("en feat/x: (%q, %v), se esperaba (\"\", nil)", warning, err)
	}

	outside := t.TempDir()
	warning, err := Default().RequireChangeBranch(context.Background(), WriteTarget{ChangeRoot: changeRootOf(outside), Change: "x", Lane: branch.LaneSDD, Context: branch.ContextOrdinary, Op: "kickoff seal"})
	if err != nil || !strings.Contains(warning, "git") {
		t.Errorf("fuera de git: (%q, %v), se esperaba no bloquear con aviso", warning, err)
	}
}

// D-11 (falla en cerrado): un fallo interno, o un contexto ya cancelado, no deja escribir.
func TestRequireChangeBranchFailsClosedOnInternalFailures(t *testing.T) {
	t.Parallel()
	repo := newProject(t, "workspace: [sin cerrar\n")
	target := WriteTarget{ChangeRoot: changeRootOf(repo), Change: "x", Lane: branch.LaneSDD, Context: branch.ContextOrdinary, Op: "gate record"}

	_, err := Default().RequireChangeBranch(context.Background(), target)
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeConfigInvalid {
		t.Fatalf("configuración ilegible: error = %v, se esperaba config_invalid", err)
	}

	healthy := embeddedCoreProject(t)
	runGit(t, healthy, "switch", "-q", "-c", "feat/x")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Default().RequireChangeBranch(cancelled, WriteTarget{ChangeRoot: changeRootOf(healthy), Change: "x", Lane: branch.LaneSDD, Context: branch.ContextOrdinary, Op: "gate record"})
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeConfigInvalid {
		t.Fatalf("contexto cancelado: error = %v, se esperaba config_invalid (falla en cerrado)", err)
	}
	for _, want := range []string{"gate record", "axiom branch status"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el fallo interno %q debe nombrar %q", err.Error(), want)
		}
	}
}
