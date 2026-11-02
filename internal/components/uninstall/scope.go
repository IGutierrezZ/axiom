package uninstall

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// statePath es el documento de estado que esta desinstalación actualiza: el del
// proyecto en el ámbito workspace y el de usuario en el global.
func (s *Service) statePath() string {
	if s.workspace {
		return state.ProjectPath(s.workspaceDir)
	}
	return state.Path(s.homeDir)
}

// readInstallState lee el estado de instalación del ámbito. En workspace es solo
// el del proyecto: el global actúa de semilla de lectura para install y sync, pero
// una desinstalación no debe deducir qué hay en el proyecto de lo que hay en otros.
// Sin documento devuelve un error que cumple errors.Is(err, fs.ErrNotExist).
func (s *Service) readInstallState() (state.InstallState, error) {
	if !s.workspace {
		return state.Read(s.homeDir)
	}
	if _, err := os.Stat(state.ProjectPath(s.workspaceDir)); err != nil {
		return state.InstallState{}, err
	}
	return state.ProjectStore(s.homeDir, s.workspaceDir).Read()
}

// updateState registra en el estado del ámbito que los agentes de toRemove ya no
// están instalados. En el global conserva el documento y quita los agentes. En el
// proyecto, cuando no queda ningún agente instalado, borra el estado de proyecto y
// su cerrojo, retira de `.git/info/exclude` las líneas que Axiom añadió y quita la
// carpeta `.axiom` si queda vacía: el proyecto vuelve a estar como antes de instalar.
func (s *Service) updateState(toRemove []model.AgentID, result *Result) ([]model.AgentID, error) {
	if !s.workspace {
		return updateStateAfterUninstall(s.homeDir, toRemove)
	}
	if len(toRemove) == 0 {
		return nil, nil
	}
	path := state.ProjectPath(s.workspaceDir)
	current, err := s.readInstallState()
	if err != nil {
		if isStateNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read install state: %w", err)
	}

	removeSet := make(map[string]struct{}, len(toRemove))
	for _, agentID := range toRemove {
		removeSet[string(agentID)] = struct{}{}
	}
	kept := make([]string, 0, len(current.InstalledAgents))
	removed := make([]model.AgentID, 0, len(toRemove))
	for _, installed := range current.InstalledAgents {
		if _, ok := removeSet[installed]; ok {
			removed = append(removed, model.AgentID(installed))
			continue
		}
		kept = append(kept, installed)
	}

	if len(kept) > 0 {
		updated := current
		updated.InstalledAgents = kept
		if slices.Contains(toRemove, model.AgentOpenCode) {
			updated.BackgroundIntent = ""
		}
		if err := state.ProjectStore(s.homeDir, s.workspaceDir).Write(updated); err != nil {
			return nil, fmt.Errorf("write install state: %w", err)
		}
		return removed, nil
	}

	// Sin agentes en el proyecto no hay nada que recordar: se retira todo lo que
	// escribió Axiom para el estado.
	for _, target := range []string{path, path + ".lock"} {
		if err := os.Remove(target); err == nil {
			result.RemovedFiles = append(result.RemovedFiles, target)
		} else if !os.IsNotExist(err) {
			return removed, fmt.Errorf("remove project state %q: %w", target, err)
		}
	}
	if err := os.Remove(filepath.Dir(path)); err == nil {
		result.RemovedDirectories = append(result.RemovedDirectories, filepath.Dir(path))
	}
	excludePath, lines, err := state.RemoveProjectStateIgnored(s.workspaceDir)
	if err != nil {
		return removed, err
	}
	if len(lines) > 0 {
		result.ChangedFiles = append(result.ChangedFiles, excludePath)
	}
	return removed, nil
}
