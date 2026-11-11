package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// projectModeFixture aísla el HOME y el registro del hub, crea un repositorio y lo
// registra como proyecto del hub.
func projectModeFixture(t *testing.T) (home, repo string, mgr *hub.Manager) {
	t.Helper()
	home = reviewModeHome(t)
	t.Setenv("AXIOM_STATE_DIR", "")
	repo = initReviewCLIRepo(t)
	var err error
	if mgr, err = hub.NewManager(""); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Register(repo, "proyecto", ""); err != nil {
		t.Fatal(err)
	}
	return home, repo, mgr
}

func runProjectMode(t *testing.T, repo string, args ...string) (ReviewModeResult, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunReviewMode(append(args, "--cwd", repo, "--json"), &out)
	if out.Len() == 0 {
		return ReviewModeResult{}, err
	}
	return decodeReviewModeResult(t, out.Bytes()), err
}

// D2: la precedencia completa. Un «off» del clon gana; si no, manda la preferencia
// del proyecto en ambos sentidos; si tampoco la hay, el interruptor global; sin
// ninguna opinión, apagado por defecto.
func TestReviewModePrecedenceMatrix(t *testing.T) {
	type want struct {
		effective reviewtransaction.RDDMode
		source    reviewtransaction.RDDModeSource
	}
	expected := func(global, project string, cloneOff bool) want {
		switch {
		case cloneOff:
			return want{reviewtransaction.RDDModeOff, reviewtransaction.RDDModeSourceCloneLocal}
		case project == "on":
			return want{reviewtransaction.RDDModeOn, reviewtransaction.RDDModeSourceProject}
		case project == "off":
			return want{reviewtransaction.RDDModeOff, reviewtransaction.RDDModeSourceProject}
		case global == "on":
			return want{reviewtransaction.RDDModeOn, reviewtransaction.RDDModeSourceGlobal}
		case global == "off":
			return want{reviewtransaction.RDDModeOff, reviewtransaction.RDDModeSourceGlobal}
		default:
			return want{reviewtransaction.RDDModeOff, reviewtransaction.RDDModeSourceDefault}
		}
	}
	for _, global := range []string{"", "on", "off"} {
		for _, project := range []string{"", "on", "off"} {
			for _, cloneOff := range []bool{false, true} {
				name := fmt.Sprintf("global=%s/project=%s/clone_off=%v", global, project, cloneOff)
				t.Run(name, func(t *testing.T) {
					_, repo, mgr := projectModeFixture(t)
					ctx := context.Background()
					switch global {
					case "on":
						if _, err := SetGlobalReviewMode(ctx, repo, true); err != nil {
							t.Fatal(err)
						}
					case "off":
						if _, err := SetGlobalReviewMode(ctx, repo, false); err != nil {
							t.Fatal(err)
						}
					}
					if _, err := mgr.SetProjectRDD("proyecto", project); err != nil {
						t.Fatal(err)
					}
					if cloneOff {
						if _, err := runProjectMode(t, repo, "disable", "--scope", "clone"); err != nil {
							t.Fatal(err)
						}
					}

					status, err := ReviewModeStatus(ctx, repo)
					if err != nil {
						t.Fatalf("ReviewModeStatus() error = %v", err)
					}
					w := expected(global, project, cloneOff)
					if status.Effective != w.effective || status.Source != w.source {
						t.Fatalf("efectivo/origen = %s/%s, se esperaba %s/%s (estado %+v)", status.Effective, status.Source, w.effective, w.source, status)
					}
					if string(status.Project) != project {
						t.Errorf("status.Project = %q, se esperaba %q", status.Project, project)
					}
					// La puerta de las operaciones usa la misma resolución.
					err = authorizeReviewRDDOperation(ctx, repo, reviewtransaction.RDDOperationStart)
					if (err == nil) != (w.effective == reviewtransaction.RDDModeOn) {
						t.Errorf("authorize start err = %v con efectivo %s", err, w.effective)
					}
				})
			}
		}
	}
}

func TestReviewModeProjectScopeEnableDisableInherit(t *testing.T) {
	_, repo, mgr := projectModeFixture(t)
	axiomYAML := []byte("workspace:\n  name: p\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n")
	if err := os.WriteFile(filepath.Join(repo, "axiom.yaml"), axiomYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := runProjectMode(t, repo, "enable", "--scope", "project")
	if err != nil {
		t.Fatalf("enable --scope project: %v", err)
	}
	if result.Scope != "project" || result.Status.Effective != reviewtransaction.RDDModeOn ||
		result.Status.Source != reviewtransaction.RDDModeSourceProject || result.Status.Global != reviewtransaction.RDDModeUnset {
		t.Fatalf("enable: %+v", result)
	}
	if mode, _, err := mgr.LookupProjectRDD(repo); err != nil || mode != "on" {
		t.Errorf("el hub debe guardar on: %q (err=%v)", mode, err)
	}

	result, err = runProjectMode(t, repo, "disable", "--scope", "project")
	if err != nil || result.Status.Effective != reviewtransaction.RDDModeOff || result.Status.Source != reviewtransaction.RDDModeSourceProject {
		t.Fatalf("disable: %+v (err=%v)", result, err)
	}

	result, err = runProjectMode(t, repo, "inherit", "--scope", "project")
	if err != nil || result.Status.Source != reviewtransaction.RDDModeSourceDefault || result.Status.Project != reviewtransaction.RDDModeUnset {
		t.Fatalf("inherit: %+v (err=%v)", result, err)
	}
	if record, _, err := mgr.LookupProjectRDD(repo); err != nil || record != "" {
		t.Errorf("inherit debe retirar la preferencia: %q (err=%v)", record, err)
	}
	raw, _ := os.ReadFile(mgr.GetConfigPath())
	if strings.Contains(string(raw), "rdd_mode") {
		t.Errorf("sin preferencia el campo se retira del JSON:\n%s", raw)
	}

	// Nada de esto toca el repositorio: ni axiom.yaml ni el estado del proyecto.
	if _, err := runProjectMode(t, repo, "enable", "--scope", "project"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(repo, "axiom.yaml"))
	if err != nil || !bytes.Equal(after, axiomYAML) {
		t.Errorf("axiom.yaml cambió: err=%v\n%s", err, after)
	}
	if _, err := os.Stat(filepath.Join(repo, ".axiom", "state.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("la preferencia no debe escribir .axiom/state.json (err=%v)", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "gentle-ai")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("la preferencia no debe crear estado del repositorio (err=%v)", err)
	}
}

func TestReviewModeProjectScopeOutsideARegisteredProjectFails(t *testing.T) {
	reviewModeHome(t)
	t.Setenv("AXIOM_STATE_DIR", "")
	repo := initReviewCLIRepo(t) // no registrado en el hub

	for _, op := range []string{"enable", "disable", "inherit"} {
		_, err := runProjectMode(t, repo, op, "--scope", "project")
		if err == nil || !strings.Contains(err.Error(), "no está en ningún proyecto registrado") {
			t.Errorf("%s fuera de un proyecto registrado debe fallar con un mensaje claro: %v", op, err)
		}
	}
	if _, err := runProjectMode(t, repo, "inherit", "--scope", "global"); err == nil {
		t.Error("inherit solo admite --scope project")
	}
	// status sigue funcionando y no inventa preferencia.
	result, err := runProjectMode(t, repo, "status")
	if err != nil || result.Status.Project != reviewtransaction.RDDModeUnset {
		t.Errorf("status sin proyecto: %+v (err=%v)", result, err)
	}
}

// Un repositorio sigue sin poder forzar el encendido: el override de clon solo
// admite apagar, también cuando hay una preferencia de proyecto encendida.
func TestReviewModeRepositoryStillCannotForceOnWithAProjectPreference(t *testing.T) {
	_, repo, mgr := projectModeFixture(t)
	if _, err := mgr.SetProjectRDD("proyecto", "on"); err != nil {
		t.Fatal(err)
	}
	global, err := readUserRDDMode(repo, mgr)
	if err != nil || global.Project != "on" {
		t.Fatalf("readUserRDDMode = %+v (err=%v)", global, err)
	}
	_, err = reviewtransaction.SetCloneLocalRDDMode(context.Background(), repo, reviewtransaction.RDDModeOn, "", global)
	if !errors.Is(err, reviewtransaction.ErrRDDModeRepositoryForcedOn) {
		t.Fatalf("el clon no puede forzar on: %v", err)
	}
}

// Limpiar el override de clon no enciende nada mientras la preferencia de proyecto
// esté apagada, y el rechazo nombra el origen que manda.
func TestReviewModeClearingCloneOverrideRefusesWhileProjectIsOff(t *testing.T) {
	_, repo, mgr := projectModeFixture(t)
	if _, err := mgr.SetProjectRDD("proyecto", "off"); err != nil {
		t.Fatal(err)
	}
	if _, err := runProjectMode(t, repo, "disable", "--scope", "clone"); err != nil {
		t.Fatal(err)
	}
	_, err := runProjectMode(t, repo, "enable", "--scope", "clone")
	var disabled *reviewtransaction.RDDDisabledError
	if !errors.As(err, &disabled) || disabled.Source != reviewtransaction.RDDModeSourceProject {
		t.Fatalf("se esperaba RDDDisabledError con origen project: %v", err)
	}
	if !strings.Contains(err.Error(), "axiom review mode enable --scope=project") {
		t.Errorf("el rechazo debe nombrar el comando del origen que decide: %v", err)
	}
}

func TestReviewModeUnreadableProjectPreferenceFailsClosedNamingTheFile(t *testing.T) {
	_, repo, mgr := projectModeFixture(t)
	raw, err := os.ReadFile(mgr.GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(raw), `"is_configured"`, `"rdd_mode": "quizás", "is_configured"`, 1)
	if broken == string(raw) {
		t.Fatalf("no se pudo corromper el hub:\n%s", raw)
	}
	if err := os.WriteFile(mgr.GetConfigPath(), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = ReviewModeStatus(context.Background(), repo)
	if err == nil || !strings.Contains(err.Error(), mgr.GetConfigPath()) {
		t.Fatalf("un valor inválido debe fallar cerrado y nombrar el fichero: %v", err)
	}
	if authErr := authorizeReviewRDDOperation(context.Background(), repo, reviewtransaction.RDDOperationStart); authErr == nil {
		t.Error("con la preferencia ilegible no se debe autorizar el inicio")
	}
}
