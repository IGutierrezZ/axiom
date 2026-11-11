package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// G4.6: InstallInWorkspace instala en la raíz explícita aunque el proceso esté en
// otra carpeta, sin cambiar de directorio, y registra los agentes en el estado del
// proyecto, igual que `axiom install --scope workspace`.
func TestInstallInWorkspaceInstallsInTheExplicitRootWhateverTheCwd(t *testing.T) {
	_, sub, home := workspaceRootFixture(t, true, false) // el cwd es otro proyecto, con su axiom.yaml
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(project, "axiom.yaml"), []byte(workspaceRootAxiomYAML))

	result, err := InstallInWorkspace(project, []string{"--agent", "claude-code", "--component", "engram", "--preset", "custom"}, system.DetectionResult{})
	if err != nil {
		t.Fatalf("InstallInWorkspace() error = %v", err)
	}
	if !result.Verify.Ready {
		t.Fatalf("la verificación posterior falló: %#v", result.Verify)
	}

	if _, err := os.Stat(filepath.Join(project, ".mcp.json")); err != nil {
		t.Errorf("falta .mcp.json en la raíz explícita: %v", err)
	}
	for _, absent := range []string{
		filepath.Join(sub, ".mcp.json"),
		filepath.Join(filepath.Dir(filepath.Dir(sub)), ".mcp.json"),
		filepath.Join(home, ".mcp.json"),
		filepath.Join(home, ".claude.json"),
	} {
		if _, statErr := os.Stat(absent); !os.IsNotExist(statErr) {
			t.Errorf("no debe existir %q; stat = %v", absent, statErr)
		}
	}

	st, err := StateStoreFor(ScopeWorkspace, home, project).Read()
	if err != nil {
		t.Fatalf("el estado del proyecto no se escribió: %v", err)
	}
	if len(st.InstalledAgents) != 1 || st.InstalledAgents[0] != "claude-code" {
		t.Errorf("InstalledAgents = %v, se esperaba [claude-code]", st.InstalledAgents)
	}
	if _, err := os.Stat(filepath.Join(project, ".axiom", "state.json")); err != nil {
		t.Errorf("falta .axiom/state.json en el proyecto: %v", err)
	}
	if _, err := StateStoreFor(ScopeGlobal, home, "").Read(); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("el estado global no debe tocarse: err = %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil || !pathidentity.SameDirectory(cwd, sub) {
		t.Errorf("la instalación no debe cambiar el directorio del proceso: cwd = %q (err=%v), quiero %q", cwd, err, sub)
	}
}

func TestInstallInWorkspaceRejectsBadInput(t *testing.T) {
	workspaceRootFixture(t, false, false)
	missing := filepath.Join(t.TempDir(), "no-existe")
	file := filepath.Join(t.TempDir(), "fichero")
	mustWriteFile(t, file, []byte("x"))

	for name, tc := range map[string]struct {
		dir  string
		args []string
		want string
	}{
		"raíz vacía":       {"", []string{"--agent", "claude-code"}, "raíz del proyecto inválida"},
		"raíz inexistente": {missing, []string{"--agent", "claude-code"}, "no existe o no es un directorio"},
		"raíz es fichero":  {file, []string{"--agent", "claude-code"}, "no existe o no es un directorio"},
		"con --scope":      {t.TempDir(), []string{"--scope", "global", "--agent", "claude-code"}, "quita --scope"},
		"agente inválido":  {t.TempDir(), []string{"--agent", "no-existe"}, "no-existe"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := InstallInWorkspace(tc.dir, tc.args, system.DetectionResult{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, quiero que contenga %q", err, tc.want)
			}
		})
	}
}
