package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// D2: el modo RDD es una preferencia del usuario en cada proyecto, guardada en el hub y
// compartida por los worktrees del proyecto. Estas pruebas recorren la CLI en proceso.

// reviewModeProject prepara un hogar aislado (con su hub) y un repositorio que es un proyecto
// Axiom con el marcador versionado. Si register, el proyecto queda registrado en el hub.
func reviewModeProject(t *testing.T, register bool) (repo string, manager *hub.Manager) {
	t.Helper()
	reviewModeHome(t)
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	repo = initReviewCLIRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "axiom.yaml"), []byte("workspace:\n  name: p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", "axiom.yaml")
	runReviewCLIGit(t, repo, "commit", "-qm", "project marker")
	manager, err := hub.NewManager("")
	if err != nil {
		t.Fatal(err)
	}
	if register {
		if _, err := manager.Register(repo, "p", ""); err != nil {
			t.Fatal(err)
		}
	}
	return repo, manager
}

func runReviewMode(t *testing.T, args ...string) (ReviewModeResult, error) {
	t.Helper()
	var output bytes.Buffer
	err := RunReviewMode(append(args, "--json"), &output)
	if output.Len() == 0 {
		return ReviewModeResult{}, err
	}
	return decodeReviewModeResult(t, output.Bytes()), err
}

func TestReviewModeStatusResolvesProjectBeforeGlobalBeforeDefault(t *testing.T) {
	for _, test := range []struct {
		name      string
		project   string
		global    string
		effective reviewtransaction.RDDMode
		source    reviewtransaction.RDDModeSource
	}{
		{name: "project on over global off", project: "on", global: "off", effective: reviewtransaction.RDDModeOn, source: reviewtransaction.RDDModeSourceProject},
		{name: "project off over global on", project: "off", global: "on", effective: reviewtransaction.RDDModeOff, source: reviewtransaction.RDDModeSourceProject},
		{name: "global on without project value", global: "on", effective: reviewtransaction.RDDModeOn, source: reviewtransaction.RDDModeSourceGlobal},
		{name: "global off without project value", global: "off", effective: reviewtransaction.RDDModeOff, source: reviewtransaction.RDDModeSourceGlobal},
		{name: "default when nothing is set", effective: reviewtransaction.RDDModeOff, source: reviewtransaction.RDDModeSourceDefault},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, manager := reviewModeProject(t, true)
			if test.project != "" {
				if _, err := manager.SetRDDMode(repo, test.project); err != nil {
					t.Fatal(err)
				}
			}
			if test.global != "" {
				if err := state.Write(os.Getenv("HOME"), state.InstallState{RDDMode: test.global}); err != nil {
					t.Fatal(err)
				}
			}
			result, err := runReviewMode(t, "status", "--cwd", repo)
			if err != nil {
				t.Fatalf("status error = %v", err)
			}
			if result.Status.Effective != test.effective || result.Status.Source != test.source {
				t.Fatalf("status = %#v, want effective=%s source=%s", result.Status, test.effective, test.source)
			}
		})
	}
}

func TestReviewModeCloneOffStillWinsOverAProjectOn(t *testing.T) {
	repo, manager := reviewModeProject(t, true)
	if _, err := manager.SetRDDMode(repo, "on"); err != nil {
		t.Fatal(err)
	}
	if _, err := runReviewMode(t, "disable", "--cwd", repo, "--scope", "clone"); err != nil {
		t.Fatalf("clone disable error = %v", err)
	}
	result, err := runReviewMode(t, "status", "--cwd", repo)
	if err != nil {
		t.Fatalf("status error = %v", err)
	}
	if result.Status.Effective != reviewtransaction.RDDModeOff || result.Status.Source != reviewtransaction.RDDModeSourceCloneLocal ||
		result.Status.Project != reviewtransaction.RDDModeOn {
		t.Fatalf("a clone off must win and still report the project value: %#v", result.Status)
	}
}

func TestReviewModeEnableAndDisableDefaultToTheProjectInsideARegisteredProject(t *testing.T) {
	repo, manager := reviewModeProject(t, true)

	result, err := runReviewMode(t, "enable", "--cwd", repo)
	if err != nil {
		t.Fatalf("enable error = %v", err)
	}
	if result.Scope != reviewModeScopeProject || result.Status.Effective != reviewtransaction.RDDModeOn ||
		result.Status.Source != reviewtransaction.RDDModeSourceProject {
		t.Fatalf("enable result = %#v", result)
	}
	if mode, _, err := manager.LookupRDDMode(repo); err != nil || mode != hub.RDDModeOn {
		t.Fatalf("el hub guarda %q, %v; se esperaba on", mode, err)
	}
	if _, err := os.Stat(state.Path(os.Getenv("HOME"))); err == nil {
		t.Fatal("un enable de proyecto no debe escribir el estado global")
	}

	result, err = runReviewMode(t, "disable", "--cwd", repo)
	if err != nil {
		t.Fatalf("disable error = %v", err)
	}
	if result.Status.Effective != reviewtransaction.RDDModeOff || result.Status.Source != reviewtransaction.RDDModeSourceProject {
		t.Fatalf("disable result = %#v", result)
	}
}

func TestReviewModeGlobalScopeInsideAProjectKeepsSettingTheMachineDefault(t *testing.T) {
	repo, manager := reviewModeProject(t, true)

	result, err := runReviewMode(t, "enable", "--cwd", repo, "--scope", "global")
	if err != nil {
		t.Fatalf("enable --scope global error = %v", err)
	}
	if result.Scope != reviewModeScopeGlobal || result.Status.Source != reviewtransaction.RDDModeSourceGlobal || !result.Status.Enabled() {
		t.Fatalf("result = %#v", result)
	}
	if mode, _, err := manager.LookupRDDMode(repo); err != nil || mode != "" {
		t.Fatalf("--scope global no debe tocar el hub: %q, %v", mode, err)
	}
}

func TestReviewModeProjectScopeRefusesAnUnregisteredProjectNamingTheGlobalExit(t *testing.T) {
	repo, manager := reviewModeProject(t, false)

	_, err := runReviewMode(t, "enable", "--cwd", repo)
	if err == nil || !strings.Contains(err.Error(), "--scope global") || !strings.Contains(err.Error(), "axiom project add") {
		t.Fatalf("error = %v, se esperaba un rechazo que nombre `axiom project add` y `--scope global`", err)
	}
	if _, statErr := os.Stat(state.Path(os.Getenv("HOME"))); statErr == nil {
		t.Fatal("el rechazo no debe escribir el estado global en silencio")
	}
	if cfg, loadErr := manager.Load(); loadErr != nil || len(cfg.Workspaces) != 0 {
		t.Fatalf("el rechazo no debe registrar el proyecto: %+v, %v", cfg, loadErr)
	}
}

func TestReviewModeOutsideAProjectKeepsTheGlobalDefaultScope(t *testing.T) {
	reviewModeHome(t)
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	repo := initReviewCLIRepo(t)

	result, err := runReviewMode(t, "enable", "--cwd", repo)
	if err != nil {
		t.Fatalf("enable error = %v", err)
	}
	if result.Scope != reviewModeScopeGlobal || result.Status.Source != reviewtransaction.RDDModeSourceGlobal || !result.Status.Enabled() {
		t.Fatalf("fuera de un proyecto el enable debía fijar el global: %#v", result)
	}
}

func TestReviewModeProjectPreferenceIsSharedByLinkedWorktrees(t *testing.T) {
	repo, manager := reviewModeProject(t, true)
	linked := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-wt")
	runReviewCLIGit(t, repo, "worktree", "add", "-q", "-b", "rama", linked)

	if _, err := runReviewMode(t, "enable", "--cwd", linked); err != nil {
		t.Fatalf("enable desde el worktree enlazado error = %v", err)
	}
	if mode, _, err := manager.LookupRDDMode(repo); err != nil || mode != hub.RDDModeOn {
		t.Fatalf("el hub debía guardar on para la carpeta maestra: %q, %v", mode, err)
	}
	cfg, err := manager.Load()
	if err != nil || len(cfg.Workspaces) != 1 || !pathidentity.SameDirectory(cfg.Workspaces[0].Path, repo) {
		t.Fatalf("el hub no debe ganar una entrada por worktree: %+v, %v", cfg, err)
	}
	for name, start := range map[string]string{"checkout principal": repo, "worktree enlazado": linked} {
		result, err := runReviewMode(t, "status", "--cwd", start)
		if err != nil || !result.Status.Enabled() || result.Status.Source != reviewtransaction.RDDModeSourceProject {
			t.Errorf("%s: status = %#v, %v; se esperaba on decidido por el proyecto", name, result.Status, err)
		}
	}
}

func TestReviewModeFailsClosedOnAnInvalidStoredProjectValue(t *testing.T) {
	repo, manager := reviewModeProject(t, true)
	cfg, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Workspaces[0].RDDMode = "sometimes"
	if err := manager.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if err := state.Write(os.Getenv("HOME"), state.InstallState{RDDMode: "on"}); err != nil {
		t.Fatal(err)
	}

	result, err := runReviewMode(t, "status", "--cwd", repo)
	if err == nil || !strings.Contains(err.Error(), "axiom review mode enable") {
		t.Fatalf("error = %v, se esperaba un fallo que nombre el comando que lo repara", err)
	}
	if result.Status.Enabled() {
		t.Fatalf("un valor inválido no puede dejar el RDD activo: %#v", result.Status)
	}
	// Un interruptor ilegible no es un interruptor apagado: los puntos de aplicación lo tratan
	// como activo (reviewDrivenDevelopmentDisabled falla cerrado a «no desactivado»).
	if disabled, _ := reviewDrivenDevelopmentDisabled(context.Background(), repo); disabled {
		t.Fatal("un valor de proyecto ilegible no puede contar como RDD desactivado")
	}

	// Sobrescribirlo con un enable de proyecto es la reparación que el mensaje promete.
	if _, err := runReviewMode(t, "enable", "--cwd", repo); err != nil {
		t.Fatalf("la reparación con enable falló: %v", err)
	}
	if mode, _, err := manager.LookupRDDMode(repo); err != nil || mode != hub.RDDModeOn {
		t.Fatalf("tras la reparación el hub guarda %q, %v", mode, err)
	}
}

func TestReviewModeProjectScopeOutsideAProjectIsRefused(t *testing.T) {
	reviewModeHome(t)
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	repo := initReviewCLIRepo(t)
	_, err := runReviewMode(t, "enable", "--cwd", repo, "--scope", "project")
	if err == nil || !strings.Contains(err.Error(), "--scope global") {
		t.Fatalf("error = %v, se esperaba un rechazo que nombre --scope global", err)
	}
}
