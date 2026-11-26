package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// InstallScope controls where agent-scoped config files (system prompts, skills/, agents/, etc.) are written.
// ScopeGlobal writes to the user's global config root for each selected agent.
// ScopeWorkspace writes to the current workspace config root for each selected agent.
type InstallScope string

const (
	// ScopeGlobal writes to the global agent config dir.
	ScopeGlobal InstallScope = "global"
	// ScopeWorkspace writes to the current workspace config root for each selected agent (default).
	ScopeWorkspace InstallScope = "workspace"

	ScopeAxiomEnvVar = "AXIOM_INSTALL_SCOPE"
	scopeEnvVar      = ScopeAxiomEnvVar
)

// ScopeEnvGlobalOverride permite que AXIOM_INSTALL_SCOPE=global elija el ámbito
// global. Es solo para las pruebas: los arneses de cli y app fijan la variable para
// trabajar sobre un HOME aislado sin pasar --scope en cada llamada. En producción
// vale false y el ámbito global solo se elige con --scope global explícito.
var ScopeEnvGlobalOverride = false

// errGlobalScopeFromEnv es el rechazo de AXIOM_INSTALL_SCOPE=global: el ámbito
// global escribe en la configuración de usuario de cada agente y afecta a todos
// los proyectos de la máquina, así que tiene que pedirse en la propia orden.
var errGlobalScopeFromEnv = fmt.Errorf("%s=global no elige el ámbito global: pídelo de forma explícita con --scope global", ScopeAxiomEnvVar)

// ResolveInstallScope resuelve el ámbito a partir de la opción y de la variable.
// Prioridad: --scope explícito > AXIOM_INSTALL_SCOPE > workspace (predeterminado).
// Un flagValue vacío significa que no se pasó la opción. La variable solo admite
// workspace: el ámbito global exige --scope global (véase ScopeEnvGlobalOverride).
func ResolveInstallScope(flagValue string) (InstallScope, error) {
	if raw := strings.TrimSpace(flagValue); raw != "" {
		return parseInstallScope(raw)
	}
	raw := strings.TrimSpace(system.Getenv(ScopeAxiomEnvVar))
	if raw == "" {
		return ScopeWorkspace, nil
	}
	scope, err := parseInstallScope(raw)
	if err != nil {
		return "", err
	}
	if scope == ScopeGlobal && !ScopeEnvGlobalOverride {
		return "", errGlobalScopeFromEnv
	}
	return scope, nil
}

func parseInstallScope(raw string) (InstallScope, error) {
	switch InstallScope(raw) {
	case ScopeGlobal, ScopeWorkspace:
		return InstallScope(raw), nil
	default:
		return "", fmt.Errorf("unsupported scope %q (valid: global, workspace)", raw)
	}
}

// ResolveAgentConfigDir returns the directory to use as the agent config root.
// When scope is ScopeWorkspace, workspaceDir is returned; otherwise homeDir is used.
// Both homeDir and workspaceDir must be non-empty absolute paths.
func ResolveAgentConfigDir(scope InstallScope, homeDir, workspaceDir string) string {
	if scope == ScopeWorkspace && strings.TrimSpace(workspaceDir) != "" {
		return workspaceDir
	}
	return homeDir
}

// workspaceRootNoticeWriter recibe el aviso de que la raíz del workspace no es la
// carpeta actual. Es una variable para que las pruebas lo capturen.
var workspaceRootNoticeWriter io.Writer = os.Stderr

// resolveWorkspaceDir devuelve el directorio que hace de raíz del workspace y, si
// difiere de la carpeta actual, el aviso en castellano para el usuario.
//
// Con ámbito global devuelve la carpeta actual sin cambios: ahí el workspace solo
// interviene en rutas secundarias y no se reinterpreta. Con ámbito workspace
// aplica workspace.ResolveRoot, de modo que lanzar el comando desde una subcarpeta
// del proyecto no cree allí `.claude`, `.mcp.json` ni el resto de ficheros.
//
// Si no se puede leer la carpeta actual devuelve "": los consumidores ya tratan la
// cadena vacía como «sin workspace».
func resolveWorkspaceDir(scope InstallScope, homeDir string) (dir, notice string) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", ""
	}
	if scope != ScopeWorkspace {
		return cwd, ""
	}
	root, source := workspace.ResolveRoot(cwd, homeDir)
	if pathidentity.SameDirectory(root, cwd) {
		return cwd, ""
	}
	return root, fmt.Sprintf("Aviso: la raíz del workspace es %s (%s), no la carpeta actual %s; los ficheros se escriben en la raíz.\n", root, source, cwd)
}

// ResolveWorkspaceDir expone resolveWorkspaceDir a quien lanza la sincronización
// fuera de este paquete (la TUI), que decide si muestra el aviso.
func ResolveWorkspaceDir(scope InstallScope, homeDir string) (dir, notice string) {
	return resolveWorkspaceDir(scope, homeDir)
}

func noteWorkspaceRoot(notice string) {
	if notice != "" {
		fmt.Fprint(workspaceRootNoticeWriter, notice)
	}
}
