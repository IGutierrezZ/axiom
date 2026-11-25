// Package mcpexclusion aplica las exclusiones personales de servidores MCP de un
// proyecto (feature contexto-por-proyecto) a la configuración local de los agentes, en la
// raíz del proyecto y en cada uno de sus worktrees. Vive fuera de `internal/dashboard` y
// de `internal/cli` para que el guardado del dashboard, `axiom sync` y `axiom branch
// ensure` compartan la misma lógica y se pueda probar sin esas capas.
package mcpexclusion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ClaudeAgent es el único agente que soporta exclusiones en esta versión: los demás no
// cargan conectores de claude.ai.
const ClaudeAgent = "claude-code"

// Estados de un destino en el informe.
const (
	StatusWritten   = "written"   // se escribió o modificó el fichero
	StatusUnchanged = "unchanged" // ya estaba como pide la preferencia
	StatusSkipped   = "skipped"   // no se tocó, con el motivo en Reason
	StatusFailed    = "failed"    // error al aplicar, con el detalle en Reason
)

// Target es el resultado de aplicar las exclusiones en un directorio.
type Target struct {
	Dir    string `json:"dir"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Report resume una aplicación: un resultado por directorio y los agentes con
// exclusiones que no se pudieron aplicar por no estar soportados.
type Report struct {
	Targets     []Target `json:"targets"`
	Unsupported []string `json:"unsupported,omitempty"`
}

// Failed devuelve los destinos que fallaron.
func (r Report) Failed() []Target {
	var failed []Target
	for _, target := range r.Targets {
		if target.Status == StatusFailed {
			failed = append(failed, target)
		}
	}
	return failed
}

// Warnings devuelve los avisos que merece el informe, en castellano: un aviso por
// destino fallido y uno por los agentes no soportados. Está pensado para que `sync` y
// `ensure` los muestren sin fallar.
func (r Report) Warnings() []string {
	var warnings []string
	for _, target := range r.Failed() {
		warnings = append(warnings, fmt.Sprintf("No se pudieron aplicar las exclusiones de servidores MCP en %s: %s", target.Dir, target.Reason))
	}
	if len(r.Unsupported) > 0 {
		warnings = append(warnings, fmt.Sprintf("Las exclusiones de servidores MCP de %s no se aplican: agente no soportado", strings.Join(r.Unsupported, ", ")))
	}
	return warnings
}

// Apply aplica las exclusiones de claude-code de current en cada directorio del proyecto
// de record (más extraDirs) y retira de ellos las de previous que ya no están en current.
// Con previous nil solo añade. Los demás agentes con exclusiones se informan como no
// soportados. Nunca falla: cada error queda en su destino del informe.
func Apply(record *hub.WorkspaceRecord, previous, current map[string][]string, extraDirs ...string) Report {
	var report Report
	if record == nil {
		return report
	}
	for _, agent := range unsupportedAgents(previous, current) {
		report.Unsupported = append(report.Unsupported, agent)
	}

	add := current[ClaudeAgent]
	var remove []string
	for _, name := range previous[ClaudeAgent] {
		if !slices.Contains(add, name) {
			remove = append(remove, name)
		}
	}
	if len(add) == 0 && len(remove) == 0 {
		return report
	}

	for _, dir := range ProjectDirs(record.Path, extraDirs...) {
		report.Targets = append(report.Targets, applyDir(dir, add, remove))
	}
	return report
}

// ApplyCurrent aplica la preferencia guardada del proyecto que contiene repo (solo
// añade, no retira nada) en el proyecto y en extraDirs. Un proyecto sin registrar o sin
// exclusiones no es un error: devuelve un informe vacío.
func ApplyCurrent(manager *hub.Manager, repo string, extraDirs ...string) (Report, error) {
	current, record, err := manager.LookupProjectMCPExclusions(repo)
	if err != nil {
		return Report{}, fmt.Errorf("no se pudieron leer las exclusiones de servidores MCP: %w", err)
	}
	if record == nil || len(current) == 0 {
		return Report{}, nil
	}
	return Apply(record, nil, current, extraDirs...), nil
}

func applyDir(dir string, add, remove []string) Target {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return Target{Dir: dir, Status: StatusSkipped, Reason: "el directorio no existe"}
	}
	changed, err := claude.ApplyDeniedMCPServers(dir, add, remove)
	switch {
	case err != nil:
		return Target{Dir: dir, Status: StatusFailed, Reason: err.Error()}
	case changed:
		return Target{Dir: dir, Status: StatusWritten}
	default:
		return Target{Dir: dir, Status: StatusUnchanged}
	}
}

func unsupportedAgents(previous, current map[string][]string) []string {
	var agents []string
	for _, set := range []map[string][]string{current, previous} {
		for agent, names := range set {
			if agent != ClaudeAgent && len(names) > 0 && !slices.Contains(agents, agent) {
				agents = append(agents, agent)
			}
		}
	}
	slices.Sort(agents)
	return agents
}

// ProjectDirs devuelve los directorios donde se aplican las exclusiones de un proyecto,
// sin repetidos y en orden estable: la raíz, los worktrees de git de cada repositorio
// y, en multirepo, los repositorios de los roles que declara axiom.yaml (sus rutas
// relativas se resuelven desde la carpeta maestra) con sus worktrees. Más extraDirs.
// Lo que no es un repositorio git aporta solo su propio directorio.
func ProjectDirs(root string, extraDirs ...string) []string {
	var repos []string
	if root != "" {
		repos = append(repos, root)
		if loc, err := workspace.Locate(root); err == nil && loc.Config != nil {
			for _, key := range sortedRoleKeys(loc.Config.Roles) {
				for _, repo := range loc.Config.Roles[key].Repositories {
					path := strings.TrimSpace(repo.Path)
					if path == "" {
						continue
					}
					if !filepath.IsAbs(path) {
						path = filepath.Join(loc.Master, path)
					}
					repos = append(repos, path)
				}
			}
		}
	}

	var dirs []string
	add := func(dir string) {
		dir = filepath.Clean(dir)
		if !slices.ContainsFunc(dirs, func(seen string) bool { return pathidentity.SameDirectory(seen, dir) }) {
			dirs = append(dirs, dir)
		}
	}
	git := &gitcli.Git{}
	for _, repo := range repos {
		add(repo)
		if _, statErr := os.Stat(filepath.Join(repo, ".git")); statErr != nil {
			continue // no es la raíz de un repositorio: git subiría a otro ajeno
		}
		if entries, err := git.Worktrees(context.Background(), repo); err == nil {
			for _, entry := range entries {
				add(entry.Path)
			}
		}
	}
	for _, dir := range extraDirs {
		if strings.TrimSpace(dir) != "" {
			add(dir)
		}
	}
	return dirs
}

func sortedRoleKeys(roles map[string]workspace.RoleConfig) []string {
	keys := make([]string, 0, len(roles))
	for key := range roles {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
