package hub

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
)

// Exclusiones personales de servidores MCP por proyecto (feature contexto-por-proyecto):
// el usuario decide, por proyecto y por agente, qué servidores MCP y conectores no debe
// cargar el agente al trabajar en él (por ejemplo, `claude.ai Microsoft 365` en un
// repositorio de código). Viven en la entrada del proyecto de `~/.axiom/workspaces.json`,
// como el resto de preferencias personales, y nunca en `axiom.yaml`: un repositorio no
// puede imponerlas. Aplicarlas en la configuración de cada agente es cosa de su
// adaptador; el hub solo las guarda.

// maxMCPServerNameLen acota el nombre de un servidor: los nombres reales son cortos y
// un límite evita guardar basura pegada por error.
const maxMCPServerNameLen = 200

// NormalizeMCPExclusions valida y normaliza unas exclusiones: recorta los nombres de
// agente y de servidor, descarta los vacíos y los repetidos, ordena cada lista y quita
// los agentes que se quedan sin ninguna. Devuelve nil si no queda nada. Un nombre con
// caracteres de control o demasiado largo es un error que lo nombra.
func NormalizeMCPExclusions(in map[string][]string) (map[string][]string, error) {
	out := map[string][]string{}
	for agent, names := range in {
		agent = strings.TrimSpace(agent)
		if agent == "" {
			if len(names) > 0 {
				return nil, fmt.Errorf("las exclusiones de servidores MCP necesitan el agente al que se aplican")
			}
			continue
		}
		var clean []string
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if err := validateMCPServerName(name); err != nil {
				return nil, err
			}
			if !slices.Contains(clean, name) {
				clean = append(clean, name)
			}
		}
		if len(clean) == 0 {
			continue
		}
		// Dos claves que solo difieren en espacios son el mismo agente: se juntan aquí y
		// se ordenan y deduplican abajo.
		out[agent] = append(out[agent], clean...)
	}
	for agent, names := range out {
		slices.Sort(names)
		out[agent] = slices.Compact(names)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func validateMCPServerName(name string) error {
	if runes := []rune(name); len(runes) > maxMCPServerNameLen {
		return fmt.Errorf("el nombre de servidor MCP «%s…» supera %d caracteres", string(runes[:40]), maxMCPServerNameLen)
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fmt.Errorf("el nombre de servidor MCP %q tiene caracteres de control", name)
	}
	return nil
}

// LookupProjectMCPExclusions devuelve una copia de las exclusiones guardadas para el
// proyecto que contiene repo (también desde un worktree enlazado) y su entrada, o nil
// si no hay ninguna o el repo no está en un proyecto registrado. No modifica nada.
func (m *Manager) LookupProjectMCPExclusions(repo string) (map[string][]string, *WorkspaceRecord, error) {
	record, err := m.ProjectForRepo(repo)
	if err != nil || record == nil {
		return nil, record, err
	}
	return cloneMCPExclusions(record.MCPExclusions), record, nil
}

// SetProjectMCPExclusions guarda las exclusiones del proyecto registrado que idOrPath
// nombra, por id o por ruta, y devuelve las que había antes (para que quien las aplica
// retire las que ya no están) y la entrada actualizada. Unas exclusiones vacías retiran
// el campo del JSON. Valida antes de leer el hub; un proyecto sin registrar es un error
// que lo dice y no modifica nada.
func (m *Manager) SetProjectMCPExclusions(idOrPath string, exclusions map[string][]string) (previous map[string][]string, record *WorkspaceRecord, err error) {
	normalized, err := NormalizeMCPExclusions(exclusions)
	if err != nil {
		return nil, nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := m.loadUnlocked()
	if err != nil {
		return nil, nil, err
	}
	for i := range cfg.Workspaces {
		if cfg.Workspaces[i].ID != idOrPath && !sameProject(cfg.Workspaces[i].Path, idOrPath) {
			continue
		}
		previous = cloneMCPExclusions(cfg.Workspaces[i].MCPExclusions)
		cfg.Workspaces[i].MCPExclusions = normalized
		if err := m.saveUnlocked(cfg); err != nil {
			return nil, nil, err
		}
		return previous, &cfg.Workspaces[i], nil
	}
	return nil, nil, fmt.Errorf("el proyecto «%s» no está registrado en %s", idOrPath, m.configPath)
}

func cloneMCPExclusions(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for _, agent := range slices.Sorted(maps.Keys(in)) {
		out[agent] = slices.Clone(in[agent])
	}
	return out
}
