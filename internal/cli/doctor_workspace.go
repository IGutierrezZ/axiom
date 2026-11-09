package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Comprobaciones de la configuración del proyecto (axiom.yaml): topología, repositorios
// de rol y repositorio de specs. Solo leen ficheros bajo la raíz dada; "es git" se
// decide por la existencia de `.git` (directorio o fichero, como en un worktree), sin
// lanzar procesos.

const workspaceFixRemedy = "Fix the workspace layout declared in axiom.yaml (see 'axiom workspace validate')"

// workspaceDoctorChecks devuelve las comprobaciones de axiom.yaml de la raíz. Sin raíz
// o sin axiom.yaml no hay configuración que mirar y no se añade ninguna: que un
// proyecto no esté configurado no es un fallo del doctor.
func workspaceDoctorChecks(opts doctor.Options) []doctor.Check {
	root := strings.TrimSpace(opts.Root)
	if root == "" {
		return nil
	}
	loc, err := workspace.LocateAt(root)
	project := func(id doctor.CheckID, run func() doctor.Result) doctor.Check {
		return doctor.Check{ID: id, Scope: doctor.ScopeProject, Run: func(context.Context) doctor.Result { return run() }}
	}
	if err != nil {
		// La resolución rechaza una configuración con el repositorio de specs ausente o
		// con un planning home ambiguo: la causa y su continuación son el diagnóstico.
		return []doctor.Check{project(doctor.CheckProjectWorkspace, func() doctor.Result {
			return doctor.Result{Status: doctor.StatusFail, Detail: err.Error(),
				Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, workspaceFixRemedy)}
		})}
	}
	if loc.ConfigPath == "" && loc.ConfigErr == nil {
		return nil
	}
	if loc.Config == nil {
		return []doctor.Check{project(doctor.CheckProjectWorkspace, func() doctor.Result {
			detail := "axiom.yaml could not be loaded"
			if loc.ConfigErr != nil {
				detail += ": " + loc.ConfigErr.Error()
			}
			return doctor.Result{Status: doctor.StatusFail, Detail: detail,
				Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, "Fix axiom.yaml (see 'axiom workspace validate')")}
		})}
	}
	return []doctor.Check{
		project(doctor.CheckProjectWorkspace, func() doctor.Result { return checkWorkspaceConfig(loc) }),
		project(doctor.CheckProjectTopology, func() doctor.Result { return checkWorkspaceTopology(loc) }),
		project(doctor.CheckProjectRoleRepos, func() doctor.Result { return checkRoleRepos(loc) }),
		project(doctor.CheckProjectSpecsRepo, func() doctor.Result { return checkSpecsRepo(loc) }),
	}
}

// checkWorkspaceConfig aplica la validación de la topología a la configuración. Los
// repositorios de rol se validan aparte (checkRoleRepos), así que aquí se omiten para
// no duplicar el mismo fallo.
func checkWorkspaceConfig(loc workspace.Location) doctor.Result {
	cfg := *loc.Config
	cfg.Roles = nil
	report, err := workspace.Validate(workspace.DefaultFS(), &cfg, loc.Master)
	if err != nil {
		return doctor.Result{Status: doctor.StatusFail, Detail: "axiom.yaml could not be validated: " + err.Error(),
			Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, workspaceFixRemedy)}
	}
	if len(report.Errors) > 0 {
		return doctor.Result{Status: doctor.StatusFail, Detail: strings.Join(report.Errors, "; "),
			Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, workspaceFixRemedy)}
	}
	if len(report.Warnings) > 0 {
		return doctor.Result{Status: doctor.StatusWarn, Detail: strings.Join(report.Warnings, "; "),
			Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, workspaceFixRemedy)}
	}
	return doctor.Result{Status: doctor.StatusPass, Detail: "axiom.yaml is valid"}
}

func checkWorkspaceTopology(loc workspace.Location) doctor.Result {
	cfg := loc.Config
	detail := fmt.Sprintf("topology %s; planning home %s", cfg.Workspace.Topology, loc.PlanningHome)
	if cfg.Workspace.Topology == workspace.TopologyMultirepo {
		detail += "; master folder " + loc.Master
	}
	return doctor.Result{Status: doctor.StatusPass, Detail: detail}
}

// repoPath resuelve la ruta de un repositorio declarado: las relativas parten de la
// carpeta maestra, como en workspace.Validate.
func repoPath(master, declared string) string {
	clean := filepath.Clean(strings.TrimSpace(declared))
	if filepath.IsAbs(clean) {
		return clean
	}
	return filepath.Join(master, clean)
}

func isGitCheckout(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func isDoctorDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// checkRoleRepos: un repositorio de rol que falta es un fallo; uno que existe pero no
// es git, un aviso. No se mira la rama: sin un cambio activo no hay rama esperada.
func checkRoleRepos(loc workspace.Location) doctor.Result {
	type entry struct{ role, declared, path string }
	var repos []entry
	for role, cfg := range loc.Config.Roles {
		for _, repo := range cfg.Repositories {
			repos = append(repos, entry{role, repo.Path, repoPath(loc.Master, repo.Path)})
		}
	}
	if len(repos) == 0 {
		return doctor.Result{Status: doctor.StatusPass, Detail: "no role repositories declared"}
	}
	sort.Slice(repos, func(i, j int) bool {
		if repos[i].role != repos[j].role {
			return repos[i].role < repos[j].role
		}
		return repos[i].declared < repos[j].declared
	})
	var missing, notGit []string
	for _, r := range repos {
		label := fmt.Sprintf("%s (role %s)", r.declared, r.role)
		switch {
		case !isDoctorDir(r.path):
			missing = append(missing, label)
		case !isGitCheckout(r.path):
			notGit = append(notGit, label)
		}
	}
	switch {
	case len(missing) > 0:
		detail := "role repositories missing: " + strings.Join(missing, ", ")
		if len(notGit) > 0 {
			detail += "; not git repositories: " + strings.Join(notGit, ", ")
		}
		return doctor.Result{Status: doctor.StatusFail, Detail: detail,
			Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, "Clone or create the missing role repositories, or correct their path in axiom.yaml")}
	case len(notGit) > 0:
		return doctor.Result{Status: doctor.StatusWarn, Detail: "role repositories that are not git repositories: " + strings.Join(notGit, ", "),
			Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, "Initialise git in those folders or correct their path in axiom.yaml")}
	}
	return doctor.Result{Status: doctor.StatusPass, Detail: fmt.Sprintf("%d role repositories exist and are git repositories", len(repos))}
}

// checkSpecsRepo: en multirepo el repositorio de specs es una carpeta propia bajo la
// maestra; en monorepo-embedded las specs viven en el propio repositorio del proyecto.
func checkSpecsRepo(loc workspace.Location) doctor.Result {
	cfg := loc.Config
	declared := strings.TrimSpace(cfg.Workspace.SpecsRepository)
	target := loc.Master
	if declared != "" && declared != "." {
		target = repoPath(loc.Master, declared)
	}
	if !isDoctorDir(target) {
		return doctor.Result{Status: doctor.StatusFail, Detail: "specs repository " + target + " does not exist",
			Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, "Create or clone the specs repository, or correct 'specs_repository' in axiom.yaml")}
	}
	gitDir := target
	if cfg.Workspace.Topology == workspace.TopologyMonorepoEmbedded {
		gitDir = loc.Master
	}
	if !isGitCheckout(gitDir) {
		return doctor.Result{Status: doctor.StatusWarn, Detail: gitDir + " holds the specs but is not a git repository",
			Remedy: doctor.NewRemedy(doctor.RemedyFixWorkspace, "Initialise git there so the specs are versioned")}
	}
	return doctor.Result{Status: doctor.StatusPass, Detail: "specs repository " + target + " exists and is a git repository"}
}
