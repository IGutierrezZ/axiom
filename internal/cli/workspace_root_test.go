package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

const workspaceRootAxiomYAML = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"

// workspaceRootFixture monta un directorio con la subcarpeta desde la que se
// lanza el comando y el HOME aislado. Con withAxiomYAML crea un axiom.yaml en la
// raíz; con withGit, un repositorio git. Devuelve (raíz, subcarpeta, home) y deja
// la subcarpeta como directorio actual.
func workspaceRootFixture(t *testing.T, withAxiomYAML, withGit bool) (root, sub, home string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sub = filepath.Join(root, "internal", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if withAxiomYAML {
		mustWriteFile(t, filepath.Join(root, "axiom.yaml"), []byte(workspaceRootAxiomYAML))
	}
	if withGit {
		if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	restoreHome := osUserHomeDir
	restoreBackupHome := backup.UserHomeDirFn
	restoreCommand := runCommand
	restoreLookPath := cmdLookPath
	restoreNotice := workspaceRootNoticeWriter
	osUserHomeDir = func() (string, error) { return home, nil }
	backup.UserHomeDirFn = func() (string, error) { return home, nil }
	runCommand = func(string, ...string) error { return nil }
	cmdLookPath = func(name string) (string, error) { return "/usr/local/bin/" + name, nil }
	t.Cleanup(func() {
		osUserHomeDir = restoreHome
		backup.UserHomeDirFn = restoreBackupHome
		runCommand = restoreCommand
		cmdLookPath = restoreLookPath
		workspaceRootNoticeWriter = restoreNotice
	})
	t.Chdir(sub)
	return root, sub, home
}

func assertSameDir(t *testing.T, got, want string) {
	t.Helper()
	if !pathidentity.SameDirectory(got, want) {
		t.Fatalf("directorio = %q, se esperaba %q", got, want)
	}
}

func TestResolveWorkspaceDir(t *testing.T) {
	t.Run("proyecto Axiom: la raíz del axiom.yaml con aviso", func(t *testing.T) {
		root, sub, home := workspaceRootFixture(t, true, true)
		dir, notice := resolveWorkspaceDir(ScopeWorkspace, home)
		assertSameDir(t, dir, root)
		for _, want := range []string{"Aviso", "raíz del workspace", "proyecto Axiom", sub} {
			if !strings.Contains(notice, want) {
				t.Errorf("el aviso %q debe contener %q", notice, want)
			}
		}
	})

	t.Run("solo repositorio git: la raíz git", func(t *testing.T) {
		root, _, home := workspaceRootFixture(t, false, true)
		dir, notice := resolveWorkspaceDir(ScopeWorkspace, home)
		assertSameDir(t, dir, root)
		if !strings.Contains(notice, "repositorio git") {
			t.Errorf("el aviso %q debe nombrar el repositorio git", notice)
		}
	})

	t.Run("fuera de todo proyecto: la carpeta actual sin aviso", func(t *testing.T) {
		_, sub, home := workspaceRootFixture(t, false, false)
		dir, notice := resolveWorkspaceDir(ScopeWorkspace, home)
		assertSameDir(t, dir, sub)
		if notice != "" {
			t.Errorf("no debe haber aviso con la raíz en la carpeta actual: %q", notice)
		}
	})

	t.Run("desde la propia raíz no hay aviso", func(t *testing.T) {
		root, _, home := workspaceRootFixture(t, true, true)
		t.Chdir(root)
		dir, notice := resolveWorkspaceDir(ScopeWorkspace, home)
		assertSameDir(t, dir, root)
		if notice != "" {
			t.Errorf("no debe haber aviso desde la raíz: %q", notice)
		}
	})

	t.Run("ámbito global conserva la carpeta actual", func(t *testing.T) {
		_, sub, home := workspaceRootFixture(t, true, true)
		dir, notice := resolveWorkspaceDir(ScopeGlobal, home)
		assertSameDir(t, dir, sub)
		if notice != "" {
			t.Errorf("el ámbito global no reinterpreta el workspace: %q", notice)
		}
	})
}

type workspaceRootCase struct {
	name          string
	withAxiomYAML bool
	withGit       bool
}

var workspaceRootCases = []workspaceRootCase{
	{"proyecto con axiom.yaml", true, false},
	{"solo repositorio git", false, true},
}

// TestRunInstallWorkspaceScopeWritesToTheProjectRoot comprueba que lanzar la
// instalación desde una subcarpeta escribe en la raíz del proyecto y no deja
// `.mcp.json` ni `.claude` en la subcarpeta.
func TestRunInstallWorkspaceScopeWritesToTheProjectRoot(t *testing.T) {
	for _, tc := range workspaceRootCases {
		t.Run(tc.name, func(t *testing.T) {
			root, sub, home := workspaceRootFixture(t, tc.withAxiomYAML, tc.withGit)
			var notice bytes.Buffer
			workspaceRootNoticeWriter = &notice

			result, err := RunInstall([]string{
				"--scope", "workspace",
				"--agent", "claude-code",
				"--component", "engram",
				"--preset", "custom",
			}, system.DetectionResult{})
			if err != nil {
				t.Fatalf("RunInstall() error = %v", err)
			}
			if !result.Verify.Ready {
				t.Fatalf("la verificación posterior falló: %#v", result.Verify)
			}

			if _, err := os.Stat(filepath.Join(root, ".mcp.json")); err != nil {
				t.Errorf("falta .mcp.json en la raíz del proyecto: %v", err)
			}
			for _, absent := range []string{
				filepath.Join(sub, ".mcp.json"),
				filepath.Join(sub, ".claude"),
				filepath.Join(home, ".mcp.json"),
				filepath.Join(home, ".claude.json"),
			} {
				if _, statErr := os.Stat(absent); !os.IsNotExist(statErr) {
					t.Errorf("no debe existir %q; stat = %v", absent, statErr)
				}
			}
			if !strings.Contains(notice.String(), "raíz del workspace") {
				t.Errorf("falta el aviso de raíz distinta de la carpeta actual: %q", notice.String())
			}
		})
	}
}

func TestRunInstallWorkspaceScopeOutsideAnyProjectUsesTheCurrentDirectory(t *testing.T) {
	_, sub, _ := workspaceRootFixture(t, false, false)
	var notice bytes.Buffer
	workspaceRootNoticeWriter = &notice

	if _, err := RunInstall([]string{
		"--scope", "workspace",
		"--agent", "claude-code",
		"--component", "engram",
		"--preset", "custom",
	}, system.DetectionResult{}); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(sub, ".mcp.json")); err != nil {
		t.Errorf("fuera de todo proyecto debe escribir en la carpeta actual: %v", err)
	}
	if notice.Len() != 0 {
		t.Errorf("no debe haber aviso: %q", notice.String())
	}
}

func writeWorkspaceRootSyncState(t *testing.T, home string) {
	t.Helper()
	if err := state.Write(home, state.InstallState{
		InstalledAgents:     []string{"claude-code"},
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentEngram},
		Persona:             "neutral",
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}
}

// TestRunSyncWorkspaceScopeWritesToTheProjectRoot es el equivalente de sync: la
// raíz del informe y los ficheros gestionados son los del proyecto, no los de la
// subcarpeta desde la que se lanzó.
func TestRunSyncWorkspaceScopeWritesToTheProjectRoot(t *testing.T) {
	for _, tc := range workspaceRootCases {
		t.Run(tc.name, func(t *testing.T) {
			root, sub, home := workspaceRootFixture(t, tc.withAxiomYAML, tc.withGit)
			var notice bytes.Buffer
			workspaceRootNoticeWriter = &notice
			writeWorkspaceRootSyncState(t, home)

			result, err := RunSync([]string{"--scope", "workspace", "--agents", "claude-code"})
			if err != nil {
				t.Fatalf("RunSync() error = %v", err)
			}

			assertSameDir(t, result.ScopeRoot, root)
			if _, err := os.Stat(filepath.Join(root, ".mcp.json")); err != nil {
				t.Errorf("falta .mcp.json en la raíz del proyecto: %v", err)
			}
			for _, absent := range []string{
				filepath.Join(sub, ".mcp.json"),
				filepath.Join(sub, ".claude"),
			} {
				if _, statErr := os.Stat(absent); !os.IsNotExist(statErr) {
					t.Errorf("no debe existir %q; stat = %v", absent, statErr)
				}
			}
			if !strings.Contains(notice.String(), "raíz del workspace") {
				t.Errorf("falta el aviso de raíz distinta de la carpeta actual: %q", notice.String())
			}
		})
	}
}

func TestRunSyncWorkspaceScopeOutsideAnyProjectUsesTheCurrentDirectory(t *testing.T) {
	_, sub, home := workspaceRootFixture(t, false, false)
	var notice bytes.Buffer
	workspaceRootNoticeWriter = &notice
	writeWorkspaceRootSyncState(t, home)

	result, err := RunSync([]string{"--scope", "workspace", "--agents", "claude-code"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	assertSameDir(t, result.ScopeRoot, sub)
	if notice.Len() != 0 {
		t.Errorf("no debe haber aviso: %q", notice.String())
	}
}

// TestRunSyncWithSelectionScopedExplicitWorkspaceDirWins asegura que un
// directorio de workspace dado de forma explícita no se reescribe a la raíz del
// proyecto: solo la resolución implícita (cadena vacía) busca la raíz.
func TestRunSyncWithSelectionScopedExplicitWorkspaceDirWins(t *testing.T) {
	root, sub, home := workspaceRootFixture(t, true, true)
	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentClaudeCode},
		Components: []model.ComponentID{model.ComponentEngram},
		Persona:    model.PersonaNeutral,
	}

	result, err := RunSyncWithSelectionScoped(home, sub, ScopeWorkspace, selection)
	if err != nil {
		t.Fatalf("RunSyncWithSelectionScoped() error = %v", err)
	}
	assertSameDir(t, result.ScopeRoot, sub)
	if _, err := os.Stat(filepath.Join(sub, ".mcp.json")); err != nil {
		t.Errorf("el directorio explícito debe respetarse: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(statErr) {
		t.Errorf("no debe escribirse en la raíz cuando el directorio es explícito; stat = %v", statErr)
	}
}
