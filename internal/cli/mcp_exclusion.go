package cli

import (
	"fmt"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/mcpexclusion"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// Aplicación de las exclusiones personales de servidores MCP del proyecto (feature
// contexto-por-proyecto) desde `axiom sync` y `axiom branch ensure`. La lógica vive en
// internal/mcpexclusion; aquí solo se enganchan los dos disparadores. Un fallo al
// aplicarlas nunca hace fallar ni sync ni ensure: se muestra como aviso.

func mcpExclusionManager(homeDir string) (*hub.Manager, error) {
	if homeDir == "" {
		return hub.NewManager("")
	}
	return hub.NewManager(filepath.Join(system.AxiomDir(homeDir), "workspaces.json"))
}

// applyMCPExclusions aplica la preferencia guardada del proyecto que contiene repo (y
// los directorios extra) y devuelve los avisos en castellano; nil si no hay nada que
// avisar, incluido el caso de un proyecto sin registrar o sin preferencia.
func applyMCPExclusions(homeDir, repo string, extraDirs ...string) []string {
	manager, err := mcpExclusionManager(homeDir)
	if err != nil {
		return []string{fmt.Sprintf("No se pudieron aplicar las exclusiones de servidores MCP: %v", err)}
	}
	report, err := mcpexclusion.ApplyCurrent(manager, repo, extraDirs...)
	if err != nil {
		return []string{fmt.Sprintf("No se pudieron aplicar las exclusiones de servidores MCP: %v", err)}
	}
	return report.Warnings()
}

// mcpExclusionSyncStep aplica en `axiom sync` la preferencia vigente del proyecto que se
// sincroniza (solo añade, nunca retira). No guarda copia de seguridad ni cuenta como
// fichero cambiado: `.claude/settings.local.json` es personal, la operación es aditiva e
// idempotente y no forma parte de los activos gestionados. Nunca devuelve error.
type mcpExclusionSyncStep struct {
	id           string
	homeDir      string
	workspaceDir string
	notes        *[]string
}

func (s mcpExclusionSyncStep) ID() string { return s.id }

func (s mcpExclusionSyncStep) Run() error {
	if warnings := applyMCPExclusions(s.homeDir, s.workspaceDir); s.notes != nil {
		*s.notes = append(*s.notes, warnings...)
	}
	return nil
}

// applyMCPExclusionsAfterEnsure aplica la preferencia a los worktrees que el resultado
// de `ensure` informa y añade los avisos al resultado. Solo actúa con un resultado
// listo; sin worktrees (modo in_place) no hace nada.
func applyMCPExclusionsAfterEnsure(result *branch.Result, cwd string) {
	if result.ExitCode() != branch.ExitReady {
		return
	}
	var worktrees []string
	for _, repository := range result.Repositories {
		for _, created := range repository.Branches {
			if created.Worktree != "" {
				worktrees = append(worktrees, created.Worktree)
			}
		}
	}
	if len(worktrees) == 0 {
		return
	}
	result.Warnings = append(result.Warnings, applyMCPExclusions("", cwd, worktrees...)...)
}
