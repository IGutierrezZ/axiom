package hub

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Preferencia personal de revisión (RDD) de un proyecto (D2 de alta-proyecto-ui): el
// usuario decide por proyecto si el desarrollo dirigido por recibos está encendido
// («on»), apagado («off») o si hereda el interruptor global («»). Vive en la entrada
// del proyecto de `~/.axiom/workspaces.json`, junto a las preferencias de worktree, y
// nunca en `axiom.yaml` ni en `.axiom/state.json`: un repositorio no puede imponerla.

// Valores de la preferencia de revisión de un proyecto. El vacío es «sin preferencia».
const (
	RDDModeOn  = "on"
	RDDModeOff = "off"
)

// ValidateRDDMode comprueba el valor de la preferencia: vacío, «on» u «off».
func ValidateRDDMode(mode string) error {
	switch mode {
	case "", RDDModeOn, RDDModeOff:
		return nil
	default:
		return fmt.Errorf("el modo de revisión «%s» no es válido: usa %s, %s o déjalo vacío para heredar el global", mode, RDDModeOn, RDDModeOff)
	}
}

// ProjectForRepo devuelve la entrada del hub del proyecto que contiene repo, o nil si
// ninguno lo contiene. Un proyecto contiene su carpeta maestra y todo lo que cuelga de
// ella: en multirepo, los repositorios de los roles y el de especificaciones. Si repo es
// un worktree enlazado o queda fuera de la maestra, se resuelve además por la carpeta
// maestra que localiza workspace.Locate. Con varios candidatos gana el más profundo.
func (m *Manager) ProjectForRepo(repo string) (*WorkspaceRecord, error) {
	cfg, err := m.Load()
	if err != nil {
		return nil, err
	}
	candidates := []string{}
	if abs, absErr := filepath.Abs(repo); absErr == nil {
		candidates = append(candidates, abs)
	}
	if loc, locErr := workspace.Locate(repo); locErr == nil && loc.Master != "" {
		candidates = append(candidates, loc.Master)
	}

	var best *WorkspaceRecord
	for i := range cfg.Workspaces {
		record := &cfg.Workspaces[i]
		for _, candidate := range candidates {
			if !sameProject(record.Path, candidate) && !pathidentity.Contains(record.Path, candidate) {
				continue
			}
			if best == nil || len(filepath.Clean(record.Path)) > len(filepath.Clean(best.Path)) {
				best = record
			}
			break
		}
	}
	return best, nil
}

// LookupProjectRDD devuelve la preferencia de revisión guardada para el proyecto que
// contiene repo («on», «off» o «» si no hay preferencia o el repo no está en ningún
// proyecto registrado) y la entrada del proyecto, si la hay. Un hub que no se puede leer
// o un valor guardado inválido es un error que nombra el fichero; no modifica nada.
func (m *Manager) LookupProjectRDD(repo string) (string, *WorkspaceRecord, error) {
	record, err := m.ProjectForRepo(repo)
	if err != nil {
		return "", nil, err
	}
	if record == nil {
		return "", nil, nil
	}
	if err := ValidateRDDMode(record.RDDMode); err != nil {
		return "", record, fmt.Errorf("la preferencia de revisión guardada en %s para %s: %w", m.configPath, record.Path, err)
	}
	return record.RDDMode, record, nil
}

// SetProjectRDD guarda la preferencia de revisión del proyecto registrado que idOrPath
// nombra, por id o por ruta, y devuelve su entrada actualizada. «» la retira del JSON.
// Valida antes de leer el hub; un proyecto sin registrar es un error que lo dice y no
// modifica nada.
func (m *Manager) SetProjectRDD(idOrPath, mode string) (*WorkspaceRecord, error) {
	if err := ValidateRDDMode(mode); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := m.loadUnlocked()
	if err != nil {
		return nil, err
	}
	for i := range cfg.Workspaces {
		if cfg.Workspaces[i].ID != idOrPath && !sameProject(cfg.Workspaces[i].Path, idOrPath) {
			continue
		}
		cfg.Workspaces[i].RDDMode = strings.TrimSpace(mode)
		if err := m.saveUnlocked(cfg); err != nil {
			return nil, err
		}
		return &cfg.Workspaces[i], nil
	}
	return nil, fmt.Errorf("el proyecto «%s» no está registrado en %s", idOrPath, m.configPath)
}
