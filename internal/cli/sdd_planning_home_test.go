package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const planningHomeChange = "inc-99-ejemplo"

// canonicalStarts son los tres puntos de partida del espacio canónico desde los
// que los verbos SDD deben resolver el mismo planning home (REQ-24.8).
func canonicalStarts(ws workspacetest.Workspace) map[string]string {
	return map[string]string{
		"carpeta maestra":                 ws.Master,
		"repositorio de especificaciones": ws.Specs,
		"repositorio de código":           filepath.Join(ws.Backend, "src"),
	}
}

func assertNoOpenSpecOutsideSpecsRepository(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	for _, dir := range []string{ws.Master, ws.Backend, ws.Frontend} {
		if _, err := os.Stat(filepath.Join(dir, "openspec")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("no debe crearse openspec/ fuera del repositorio de especificaciones: %s (%v)", dir, err)
		}
	}
}

// REQ-24.8 y REQ-24.9: kickoff y gate leen y escriben el cambio del repositorio
// de especificaciones desde cualquier punto de partida del espacio canónico.
func TestGovernanceVerbsOperateOnTheSamePlanningHomeFromEveryStart(t *testing.T) {
	ws := workspacetest.Canonical(t)
	changeRoot := filepath.Join(ws.Home, "changes", planningHomeChange)

	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			var sealOut bytes.Buffer
			if err := RunSDDKickoff([]string{"seal", "--cwd", start, "--change", planningHomeChange,
				"--execution-style", "checkpointed", "--handoff-policy", "none"}, &sealOut); err != nil {
				t.Fatalf("kickoff seal desde %s: %v", start, err)
			}
			if _, err := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); err != nil {
				t.Fatalf("kickoff.yaml debe estar en el cambio del repositorio de especificaciones: %v", err)
			}

			var showOut bytes.Buffer
			if err := RunSDDKickoff([]string{"show", "--cwd", start, "--change", planningHomeChange}, &showOut); err != nil {
				t.Fatalf("kickoff show desde %s: %v", start, err)
			}
			if !strings.Contains(showOut.String(), "checkpointed") {
				t.Fatalf("kickoff show no lee el kickoff sellado:\n%s", showOut.String())
			}

			if err := RunSDDGate([]string{"record", "--cwd", start, "--change", planningHomeChange,
				"--gate", "spec", "--decision", "approved", "--reason", "aprobada desde " + name}, &bytes.Buffer{}); err != nil {
				t.Fatalf("gate record desde %s: %v", start, err)
			}
			var gateOut bytes.Buffer
			if err := RunSDDGate([]string{"show", "--cwd", start, "--change", planningHomeChange}, &gateOut); err != nil {
				t.Fatalf("gate show desde %s: %v", start, err)
			}
			if !strings.Contains(gateOut.String(), "aprobada desde "+name) {
				t.Fatalf("gate show no lista la decisión registrada desde %s:\n%s", name, gateOut.String())
			}
		})
	}
	assertNoOpenSpecOutsideSpecsRepository(t, ws)
}

// REQ-24.8: la configuración que consulta `kickoff seal --infer` es la que
// resuelve el primitivo (a través del puntero), no un axiom.yaml de la raíz.
func TestKickoffSealInferReadsTheConfigurationBehindThePointer(t *testing.T) {
	ws := workspacetest.Canonical(t)
	changeRoot := filepath.Join(ws.Home, "changes", planningHomeChange)
	start := filepath.Join(ws.Backend, "src")

	t.Run("rol declarado en la configuración del puntero", func(t *testing.T) {
		writeCLIFixture(t, filepath.Join(changeRoot, "design.md"),
			"# Diseno\n\n```yaml\nroles:\n  - role: backend\n    gate_policy: blocking\n```\n")
		var out bytes.Buffer
		if err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", start, "--change", planningHomeChange}, &out); err != nil {
			t.Fatalf("seal --infer: %v", err)
		}
		if !strings.Contains(out.String(), "backend") {
			t.Fatalf("la salida no refleja el rol inferido:\n%s", out.String())
		}
	})
	t.Run("rol ausente de la configuración del puntero", func(t *testing.T) {
		other := workspacetest.Canonical(t)
		otherChange := filepath.Join(other.Home, "changes", planningHomeChange)
		writeCLIFixture(t, filepath.Join(otherChange, "design.md"),
			"# Diseno\n\n```yaml\nroles:\n  - role: database\n    gate_policy: blocking\n```\n")
		err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", filepath.Join(other.Backend, "src"), "--change", planningHomeChange}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("se esperaba el error de DetectRoles: la configuración del puntero no declara el rol database")
		}
		if _, statErr := os.Stat(filepath.Join(otherChange, "kickoff.yaml")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("kickoff.yaml se escribió pese al error: %v", statErr)
		}
	})
}

// REQ-24.9, escenario 3: un cambio archivado con prefijo de fecha se rechaza por
// archivado y no con «no existe».
func TestGovernanceVerbsRejectAnArchivedChangeWithADatePrefix(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithArchivedChange())
	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			err := RunSDDGate([]string{"record", "--cwd", start, "--change", "inc-98-previo",
				"--gate", "spec", "--decision", "approved", "--reason", "x"}, &bytes.Buffer{})
			if err == nil {
				t.Fatal("se esperaba el rechazo del cambio archivado")
			}
			if !strings.Contains(err.Error(), "archivado") || !strings.Contains(err.Error(), "REQ-21.18") {
				t.Fatalf("el rechazo debe ser por cambio archivado: %v", err)
			}
			if strings.Contains(err.Error(), "no existe") {
				t.Fatalf("el cambio archivado no debe responder «no existe»: %v", err)
			}
		})
	}
}

func TestGovernanceVerbsRejectAnUnresolvableWorkspaceWithoutWriting(t *testing.T) {
	t.Run("planning home ambiguo", func(t *testing.T) {
		ws := workspacetest.Canonical(t, workspacetest.WithMasterOpenSpec())
		err := RunSDDKickoff([]string{"show", "--cwd", filepath.Join(ws.Backend, "src"), "--change", planningHomeChange}, &bytes.Buffer{})
		var locateErr *workspace.LocateError
		if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocateAmbiguousPlanningHome {
			t.Fatalf("error = %v, se esperaba ambiguous_planning_home", err)
		}
	})
	t.Run("directorio de partida inexistente", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "no-existe")
		err := RunSDDGate([]string{"show", "--cwd", missing, "--change", planningHomeChange}, &bytes.Buffer{})
		var locateErr *workspace.LocateError
		if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocateStartInvalid || !strings.Contains(err.Error(), "--cwd") {
			t.Fatalf("error = %v, se esperaba start_invalid con la continuación --cwd", err)
		}
	})
	t.Run("cambio inexistente", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		err := RunSDDKickoff([]string{"show", "--cwd", ws.Master, "--change", "inc-00-no-existe"}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "no existe") || !strings.Contains(err.Error(), ws.Specs) {
			t.Fatalf("error = %v, se esperaba «no existe» con la ruta del repositorio de especificaciones", err)
		}
	})
}

// REQ-24.9 y REQ-24.12: la autorización por cambio opera sobre el repositorio de
// especificaciones, también cuando se lanza desde un repositorio de código.
func TestSDDAttemptGrantOperatesOnTheSpecsRepositoryFromACodeRepository(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithGit())
	for _, args := range [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "core.autocrlf", "false"},
	} {
		runReviewCLIGit(t, ws.Specs, args...)
	}
	sibling, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	seedSDDStatusReadyChange(t, ws.Specs, "cli-grant", "- [ ] 1.1 Update `"+filepath.Join(sibling, "main.go")+"`\n")
	runReviewCLIGit(t, ws.Specs, "add", "-A")
	runReviewCLIGit(t, ws.Specs, "commit", "-qm", "base")

	start := filepath.Join(ws.Backend, "src")
	var continued bytes.Buffer
	if err := RunSDDContinue([]string{"cli-grant", "--cwd", start, "--json"}, &continued); err != nil {
		t.Fatalf("sdd continue desde el repositorio de código: %v", err)
	}
	marker, err := os.ReadFile(filepath.Join(ws.Home, "changes", "cli-grant", ".axiom-instance"))
	if err != nil {
		t.Fatalf("el marcador de instancia debe estar en el repositorio de especificaciones: %v", err)
	}

	granted := runSDDAttemptStatus(t, []string{
		"grant", "--cwd", start, "--change", "cli-grant", "--root", sibling,
		"--change-instance", strings.TrimSpace(string(marker)), "--actor", "maintainer",
		"--reason", "despliegue secuencial multirrepositorio", "--request-id", "planning-home-grant-1",
	})
	if len(granted.GrantedRoots) != 1 || granted.GrantedRoots[0] != sibling || granted.Revision == "" {
		t.Fatalf("grant = %#v, se esperaba la raíz %q concedida con una revisión", granted, sibling)
	}
	assertNoOpenSpecOutsideSpecsRepository(t, ws)
}
