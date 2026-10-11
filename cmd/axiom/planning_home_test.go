package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// helperProcessEnv activa TestAxiomHelperProcess: el binario de pruebas se
// relanza a sí mismo y ejecuta main() con los argumentos que siguen a "--". Los
// verbos handoff y role terminan con os.Exit, así que se comprueban como
// subproceso sin necesidad de compilar otro binario.
const helperProcessEnv = "AXIOM_TEST_HELPER_PROCESS"

func TestAxiomHelperProcess(t *testing.T) {
	if os.Getenv(helperProcessEnv) != "1" {
		t.Skip("subproceso auxiliar: solo se ejecuta desde runAxiom")
	}
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 {
		t.Fatal("falta el separador -- con los argumentos del CLI")
	}
	os.Args = append([]string{"axiom"}, os.Args[separator+1:]...)
	main()
}

// runAxiom ejecuta el CLI como subproceso y devuelve su salida combinada y su
// código de salida.
func runAxiom(t *testing.T, args ...string) (string, int) {
	t.Helper()
	command := exec.Command(os.Args[0], append([]string{"-test.run=^TestAxiomHelperProcess$", "--"}, args...)...)
	command.Env = append(os.Environ(), helperProcessEnv+"=1")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return output.String(), 0
	case errors.As(err, &exitErr):
		return output.String(), exitErr.ExitCode()
	default:
		t.Fatalf("no se pudo ejecutar el subproceso: %v", err)
		return "", -1
	}
}

const planningHomeChange = "inc-99-ejemplo"

func canonicalStarts(ws workspacetest.Workspace) map[string]string {
	return map[string]string{
		"carpeta maestra":                 ws.Master,
		"repositorio de especificaciones": ws.Specs,
		"repositorio de código":           filepath.Join(ws.Backend, "src"),
	}
}

func canonicalChangeDir(ws workspacetest.Workspace) string {
	return filepath.Join(ws.Home, "changes", planningHomeChange)
}

func assertNoOpenSpecOutsideSpecsRepository(t *testing.T, ws workspacetest.Workspace) {
	t.Helper()
	for _, dir := range []string{ws.Master, ws.Backend, ws.Frontend} {
		if _, err := os.Stat(filepath.Join(dir, "openspec")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("no debe crearse openspec/ fuera del repositorio de especificaciones: %s (%v)", dir, err)
		}
	}
}

// writeDesign declara en design.md del cambio los roles indicados.
func writeDesign(t *testing.T, ws workspacetest.Workspace, roles ...string) {
	t.Helper()
	var design strings.Builder
	design.WriteString("# Diseno\n\n```yaml\nroles:\n")
	for _, role := range roles {
		design.WriteString("  - role: " + role + "\n    gate_policy: blocking\n")
	}
	design.WriteString("```\n")
	workspacetest.WriteFile(t, filepath.Join(canonicalChangeDir(ws), "design.md"), design.String())
}

// REQ-24.8: `role list` lee los roles de la configuración que apunta el puntero
// y el cambio del repositorio de especificaciones, desde cualquier punto de
// partida. La carpeta maestra no tiene axiom.yaml propio.
func TestRoleListHonoursThePointerFromEveryStart(t *testing.T) {
	ws := workspacetest.Canonical(t)
	writeDesign(t, ws, "backend", "frontend")
	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "role", "list", "--change", planningHomeChange, "--path", start)
			if code != 0 {
				t.Fatalf("role list salió con %d:\n%s", code, output)
			}
			for _, want := range []string{"Total de roles asignados: 2", "backend", "frontend"} {
				if !strings.Contains(output, want) {
					t.Fatalf("la salida no contiene %q:\n%s", want, output)
				}
			}
		})
	}
	assertNoOpenSpecOutsideSpecsRepository(t, ws)

	t.Run("un rol ausente de la configuración del puntero se rechaza", func(t *testing.T) {
		other := workspacetest.Canonical(t)
		writeDesign(t, other, "database")
		output, code := runAxiom(t, "role", "list", "--change", planningHomeChange, "--path", other.Master)
		if code == 0 || !strings.Contains(output, "database") {
			t.Fatalf("se esperaba el error de DetectRoles (salida %d):\n%s", code, output)
		}
	})
}

func TestRoleStatusAndBarrierResolveTheChangeFromACodeRepository(t *testing.T) {
	ws := workspacetest.Canonical(t)
	writeDesign(t, ws, "backend")
	start := filepath.Join(ws.Backend, "src")

	output, code := runAxiom(t, "role", "status", "--change", planningHomeChange, "--path", start)
	if code != 0 || !strings.Contains(output, "Axiom Estado de Roles: "+planningHomeChange) {
		t.Fatalf("role status salió con %d:\n%s", code, output)
	}
	output, code = runAxiom(t, "role", "barrier", "--change", planningHomeChange, "--path", start)
	if code == 0 || !strings.Contains(output, "BARRIER BLOCKED") {
		t.Fatalf("role barrier debía bloquear por falta de tareas y verificación (salida %d):\n%s", code, output)
	}
}

// REQ-24.8: handoff create, validate y show operan sobre el planning home del
// espacio canónico con independencia del punto de partida.
func TestHandoffVerbsUseThePlanningHomeFromEveryStart(t *testing.T) {
	ws := workspacetest.Canonical(t)
	handoffPath := filepath.Join(canonicalChangeDir(ws), "handoff.md")

	output, code := runAxiom(t, "handoff", "create", "--change", planningHomeChange,
		"--from", "spec", "--to", "design", "--from-role", "backend", "--to-role", "frontend",
		"--path", filepath.Join(ws.Backend, "src"))
	if code != 0 {
		t.Fatalf("handoff create salió con %d:\n%s", code, output)
	}
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("el relevo debe escribirse en el repositorio de especificaciones: %v", err)
	}

	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "handoff", "validate", "--change", planningHomeChange, "--path", start)
			if code != 0 || !strings.Contains(output, "HANDOFF VALID") {
				t.Fatalf("handoff validate salió con %d:\n%s", code, output)
			}
			output, code = runAxiom(t, "handoff", "show", "--change", planningHomeChange, "--path", start)
			if code != 0 || !strings.Contains(output, handoffPath) {
				t.Fatalf("handoff show debe nombrar %s (salida %d):\n%s", handoffPath, code, output)
			}
		})
	}
	assertNoOpenSpecOutsideSpecsRepository(t, ws)
}

func TestResolveChangeDirAndHandoffFileUseTheResolvedWorkspaceRoot(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(canonicalChangeDir(ws), "handoff.md"), "# Relevo\n")

	for name, start := range canonicalStarts(ws) {
		t.Run(name, func(t *testing.T) {
			loc, change, changeDir, err := resolveChangeDir(start, planningHomeChange)
			if err != nil {
				t.Fatal(err)
			}
			if loc.WorkspaceRoot != ws.Specs || loc.Config == nil || change != planningHomeChange || changeDir != canonicalChangeDir(ws) {
				t.Fatalf("resolveChangeDir = (%+v, %q, %q)", loc, change, changeDir)
			}
			// Sin --change se elige el único cambio activo del planning home.
			if _, only, _, err := resolveChangeDir(start, ""); err != nil || only != planningHomeChange {
				t.Fatalf("resolveChangeDir sin cambio = (%q, %v)", only, err)
			}
			file, err := resolveHandoffFile(loc.WorkspaceRoot, "")
			if err != nil || file != filepath.Join(canonicalChangeDir(ws), "handoff.md") {
				t.Fatalf("resolveHandoffFile = (%q, %v)", file, err)
			}
		})
	}

	t.Run("planning home ambiguo", func(t *testing.T) {
		ambiguous := workspacetest.Canonical(t, workspacetest.WithMasterOpenSpec())
		_, _, _, err := resolveChangeDir(ambiguous.Master, planningHomeChange)
		var locateErr *workspace.LocateError
		if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocateAmbiguousPlanningHome {
			t.Fatalf("error = %v, se esperaba ambiguous_planning_home", err)
		}
	})
}
