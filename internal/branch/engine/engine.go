// Package engine es el motor de `axiom branch` (INC-25, design D-01 y §4.2): reúne los
// hechos de solo lectura de un cambio (el proyecto, su kickoff, sus repositorios y su
// estado en git), se los entrega al planificador puro de internal/branch y describe el
// resultado. Esta rebanada trae `Status`; las operaciones que escriben llegan con las
// siguientes.
//
// No lanza procesos ni abre la red: toda lectura de git pasa por gitcommon.Run a través
// de internal/branch/gitcli (REQ-25.1). Los textos para el usuario están en castellano.
package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// gitReader son las lecturas de git que usa el motor en esta rebanada. Es un subconjunto
// del puerto branch.Git: cada método se añade a él con el primer llamador que lo usa
// (tasks.md, decisión T1), y el motor pasa a depender de branch.Git entero cuando
// existan las escrituras de `ensure`.
type gitReader interface {
	TopLevel(ctx context.Context, dir string) (string, error)
	CommonDir(ctx context.Context, dir string) (string, error)
	ResolveCommit(ctx context.Context, dir, ref string) (sha string, ok bool, err error)
	BranchExists(ctx context.Context, dir, name string) (bool, error)
	Worktrees(ctx context.Context, dir string) ([]branch.WorktreeEntry, error)
	DirtyPaths(ctx context.Context, dir string) ([]string, error)
	AheadBehind(ctx context.Context, dir, base, branch string) (ahead, behind int, err error)
}

var _ gitReader = (*gitcli.Git)(nil)

// Engine es el motor de ramas.
type Engine struct {
	git gitReader
}

// Default monta el motor con las lecturas reales de git.
func Default() *Engine {
	return &Engine{git: gitcli.New()}
}

// planInput reúne los hechos de solo lectura de un cambio y compone la entrada del
// planificador (design §3.1): el proyecto que resuelve cwd, el kickoff del cambio, y las
// raíces git del planning home y de los repositorios que declaran los roles sellados.
// Devuelve el rechazo que impide planificar, siempre antes de leer nada con el nombre
// del cambio.
func (e *Engine) planInput(ctx context.Context, operation, cwd, change string, lane branch.Lane) (branch.PlanInput, *branch.Refusal) {
	if reason := directoryProblem(cwd); reason != "" {
		return branch.PlanInput{}, branch.UnusableDirectory(operation, reason)
	}
	if refusal := branch.ValidateChangeName(operation, cwd, change); refusal != nil {
		return branch.PlanInput{}, refusal
	}
	location, err := workspace.Locate(cwd)
	if err != nil {
		return branch.PlanInput{}, branch.LocateRefusal(err)
	}
	if refusal := branch.ConfigRefusal(location); refusal != nil {
		return branch.PlanInput{}, refusal
	}

	topology := workspace.TopologyMonorepoEmbedded
	if location.Config != nil {
		topology = location.Config.Workspace.Topology
	}
	planning, refusal := e.repositoryFacts(ctx, operation, location, location.WorkspaceRoot, ".", location.WorkspaceRoot)
	if refusal != nil {
		return branch.PlanInput{}, refusal
	}
	workspaceRel, inside := relativeWithin(planning.GitRoot, location.WorkspaceRoot)
	if !inside {
		workspaceRel = "."
	}

	hasKickoff, sealed, refusal := sealedRoles(cwd, location, change, lane)
	if refusal != nil {
		return branch.PlanInput{}, refusal
	}
	code, skipped, refusal := e.declaredRepositories(ctx, operation, location, &planning, sealed)
	if refusal != nil {
		return branch.PlanInput{}, refusal
	}

	return branch.PlanInput{
		Operation:     operation,
		Lane:          lane,
		Change:        change,
		Topology:      topology,
		Master:        location.Master,
		WorkspaceRoot: location.WorkspaceRoot,
		WorkspaceRel:  workspaceRel,
		PlanningRepo:  planning,
		CodeRepos:     code,
		Skipped:       skipped,
		HasKickoff:    hasKickoff,
		SealedRoles:   sealed,
		Policy:        branch.ResolvePolicy(location.Config),
		WorktreeRoot:  branch.DefaultWorktreeRoot(location.Master),
	}, nil
}

// repositoryFacts lee la raíz git y el git common dir del repositorio que contiene dir
// (REQ-25.6: con las consultas de git, nunca suponiendo un `.git` literal). declared es
// la ruta tal como se declaró. errorDir es el directorio que citan los rechazos.
func (e *Engine) repositoryFacts(ctx context.Context, operation string, location workspace.Location, dir, declared, errorDir string) (branch.RepoFacts, *branch.Refusal) {
	root, err := e.git.TopLevel(ctx, dir)
	if err != nil {
		return branch.RepoFacts{}, gitRefusal(operation, errorDir, "localizar la raíz del repositorio", err)
	}
	return e.completeFacts(ctx, operation, location, dir, root, declared, errorDir)
}

// completeFacts completa los hechos de un repositorio cuya raíz git ya se conoce: su
// git common dir y su ruta relativa a la carpeta maestra.
func (e *Engine) completeFacts(ctx context.Context, operation string, location workspace.Location, dir, root, declared, errorDir string) (branch.RepoFacts, *branch.Refusal) {
	common, err := e.git.CommonDir(ctx, dir)
	if err != nil {
		return branch.RepoFacts{}, gitRefusal(operation, errorDir, "localizar el directorio común de git", err)
	}
	facts := branch.RepoFacts{GitRoot: root, CommonDir: common, DeclaredPath: declared}
	// Un repositorio fuera de la carpeta maestra no puede tener entrada propia en
	// `repositories:` (sus rutas son relativas a ella): rige la política del workspace.
	if relative, inside := relativeWithin(location.Master, root); inside {
		facts.RelPath = relative
	}
	return facts, nil
}

// sealedRoles devuelve los roles sellados en el kickoff del cambio. El carril ODD no
// tiene kickoff (REQ-25.13) y un cambio sin sellar no tiene roles aún: ambos devuelven
// hasKickoff = false. Un kickoff que no se puede leer no se trata como «sin sellar»
// (T-10): es un rechazo.
func sealedRoles(cwd string, location workspace.Location, change string, lane branch.Lane) (hasKickoff bool, roles []string, refusal *branch.Refusal) {
	if lane == branch.LaneODD {
		return false, nil, nil
	}
	changeRoot := filepath.Join(location.PlanningHome, "changes", change)
	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		kickoffPath := filepath.Join(changeRoot, kickoff.KickoffFileName)
		return false, nil, &branch.Refusal{
			Code:         branch.CodeConfigInvalid,
			Message:      fmt.Sprintf("el kickoff del cambio %s no se puede leer: %v", change, err),
			Continuation: fmt.Sprintf("restaura o corrige %s y comprueba con `axiom sdd status %s --cwd %s`", pathquote.Quote(kickoffPath), change, pathquote.Quote(cwd)),
			Details:      map[string]string{"path": kickoffPath},
		}
	}
	if sealed == nil {
		return false, nil, nil
	}
	for _, role := range sealed.Config.Roles {
		roles = append(roles, role.Role)
	}
	return true, roles, nil
}

// declaredRepository es una ruta que declara un rol en axiom.yaml.
type declaredRepository struct{ role, path string }

// declaredRepositories resuelve a raíces git, sin repetidos, los repositorios que
// declaran los roles sellados del cambio (`fullstack` toca los de todos los roles), y
// asocia a cada raíz los roles que la declaran. Los que resuelven a la raíz del
// planning home se anotan en planning. Una ruta que no existe o no es un repositorio
// git se omite sin abortar (REQ-25.6).
func (e *Engine) declaredRepositories(ctx context.Context, operation string, location workspace.Location, planning *branch.RepoFacts, sealed []string) ([]branch.RepoFacts, []branch.SkippedPath, *branch.Refusal) {
	var code []*branch.RepoFacts
	var skipped []branch.SkippedPath
	for _, declared := range declaredPaths(location.Config, sealed) {
		dir := declared.path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(location.Master, filepath.FromSlash(strings.ReplaceAll(dir, `\`, "/")))
		}
		dir = filepath.Clean(dir)
		if reason := missingPath(dir); reason != "" {
			skipped = append(skipped, skip(dir, declared.role, reason))
			continue
		}
		root, err := e.git.TopLevel(ctx, dir)
		if errors.Is(err, gitcommon.ErrNotGitRepository) {
			skipped = append(skipped, skip(dir, declared.role, "no pertenece a ningún repositorio git"))
			continue
		}
		if err != nil {
			return nil, nil, gitRefusal(operation, location.Master, "localizar la raíz git de "+dir, err)
		}

		repository := repositoryAt(append([]*branch.RepoFacts{planning}, code...), root)
		if repository == nil {
			facts, refusal := e.completeFacts(ctx, operation, location, dir, root, declared.path, location.Master)
			if refusal != nil {
				return nil, nil, refusal
			}
			repository = &facts
			code = append(code, repository)
		}
		if !slices.Contains(repository.Roles, declared.role) {
			repository.Roles = append(repository.Roles, declared.role)
		}
	}
	values := make([]branch.RepoFacts, 0, len(code))
	for _, repository := range code {
		values = append(values, *repository)
	}
	return values, skipped, nil
}

// declaredPaths son las rutas que declaran, en axiom.yaml, los roles sellados del
// cambio, en orden estable (por rol y, dentro de cada rol, por orden de declaración).
// `fullstack` cuenta como todos los roles declarados (REQ-25.4, S-3).
func declaredPaths(cfg *workspace.WorkspaceConfig, sealed []string) []declaredRepository {
	if cfg == nil || len(sealed) == 0 {
		return nil
	}
	fullstack := slices.ContainsFunc(sealed, multirole.IsReservedRole)
	var declared []declaredRepository
	for _, role := range slices.Sorted(maps.Keys(cfg.Roles)) {
		if !fullstack && !slices.ContainsFunc(sealed, func(candidate string) bool { return strings.EqualFold(candidate, role) }) {
			continue
		}
		for _, entry := range cfg.Roles[role].Repositories {
			declared = append(declared, declaredRepository{role: role, path: entry.Path})
		}
	}
	return declared
}

// repositoryAt devuelve el repositorio de la lista cuya raíz git es root (por identidad
// de ruta, no por texto), o nil.
func repositoryAt(repositories []*branch.RepoFacts, root string) *branch.RepoFacts {
	for _, repository := range repositories {
		if pathidentity.SameDirectory(repository.GitRoot, root) {
			return repository
		}
	}
	return nil
}

// missingPath explica por qué una ruta declarada no se puede usar como directorio, o
// devuelve la cadena vacía si es un directorio al que se puede acceder.
func missingPath(dir string) string {
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "no existe"
	case err != nil:
		return fmt.Sprintf("no es accesible (%v)", err)
	case !info.IsDir():
		return "no es un directorio"
	}
	return ""
}

// skip compone la ruta omitida: nombra la ruta, el rol y el motivo (REQ-25.6).
func skip(dir, role, reason string) branch.SkippedPath {
	return branch.SkippedPath{
		Path:    dir,
		Role:    role,
		Code:    branch.CodeNotAGitRepository,
		Message: fmt.Sprintf("la ruta %s declarada por el rol %s %s; se omite del plan", dir, role, reason),
	}
}

// directoryProblem explica por qué dir no sirve como directorio de partida, o devuelve
// la cadena vacía si es un directorio al que se puede acceder.
func directoryProblem(dir string) string {
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("el directorio %s no existe", dir)
	case err != nil:
		return fmt.Sprintf("no se pudo acceder al directorio %s: %v", dir, err)
	case !info.IsDir():
		return fmt.Sprintf("%s no es un directorio", dir)
	}
	return ""
}

// gitRefusal traduce un fallo de las lecturas de git: «no es un repositorio» es un
// rechazo (`not_a_git_repository`); cualquier otro es `git_failed`, que nombra la
// operación y resume el error de git (REQ-25.2).
func gitRefusal(operation, dir, gitOperation string, err error) *branch.Refusal {
	if errors.Is(err, gitcommon.ErrNotGitRepository) {
		return branch.NotAGitRepository(operation, dir)
	}
	return branch.GitFailed(operation, dir, gitOperation, err)
}

// relativeWithin devuelve la ruta de descendant relativa a ancestor, con «/» (`.` si es
// el propio ancestor), y false si descendant no está dentro de él. Asciende desde
// descendant comparando cada directorio con ancestor por identidad, no por texto: git
// devuelve la grafía larga de Windows y la carpeta maestra puede llevar el nombre corto
// 8.3 (tasks.md, regla 10).
func relativeWithin(ancestor, descendant string) (string, bool) {
	var segments []string
	for current := filepath.Clean(descendant); ; {
		if pathidentity.SameDirectory(current, ancestor) {
			if len(segments) == 0 {
				return ".", true
			}
			slices.Reverse(segments)
			return path.Join(segments...), true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		segments = append(segments, filepath.Base(current))
		current = parent
	}
}
