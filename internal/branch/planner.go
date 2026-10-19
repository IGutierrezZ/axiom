package branch

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// El planificador (design D-04) es una función pura: a partir de los hechos que
// reúne el motor (topología, política, roles sellados, raíces git y preferencias
// personales) produce un plan determinista de ramas por repositorio, o el rechazo
// que lo impide. No lee el disco ni ejecuta nada, de modo que sus reglas se prueban
// con tablas.

// DefaultWorktreeRoot es la raíz de worktrees por defecto cuando el hub no declara
// una: `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`, fuera de
// cualquier repositorio del proyecto (REQ-25.11, design D-04 y D-08). master debe ser
// la carpeta maestra del checkout principal.
func DefaultWorktreeRoot(master string) string {
	master = filepath.Clean(master)
	return filepath.Join(filepath.Dir(master), filepath.Base(master)+"-wt")
}

// RepoFacts son los hechos de un repositorio git que entran al planificador.
type RepoFacts struct {
	GitRoot   string // absoluta y limpia
	CommonDir string // absoluto
	// RelPath es la ruta de la raíz git relativa a la carpeta maestra, con «/» (`.` si
	// es la propia carpeta maestra). Vacía cuando la raíz git queda fuera de la carpeta
	// maestra: entonces rige la política del workspace y ninguna entrada de
	// `repositories:`, cuyas rutas son relativas a ella.
	RelPath string
	// DeclaredPath es la ruta tal como la declara `axiom.yaml` (en un rol), de la que
	// sale el nombre `<repo>` del worktree (S-17). El repositorio del planning home
	// lleva `.`.
	DeclaredPath string
	// Roles son los roles de axiom.yaml cuyas rutas resuelven a esta raíz git. El motor
	// entrega los de los roles sellados del cambio (todos los declarados, con
	// `fullstack`) y el planificador los cruza con los roles del kickoff.
	Roles []string
}

// SkippedPath es una ruta declarada por un rol que se omite del plan sin abortarlo:
// no existe o no pertenece a ningún repositorio git (REQ-25.6).
type SkippedPath struct {
	Path    string `json:"path"` // absoluta
	Role    string `json:"role"`
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

// PlanInput es todo lo que el planificador necesita. Las excepciones por cambio
// (`--prefix`, `--branch`, `--base`) van encima de la política (REQ-25.3).
type PlanInput struct {
	// Operation es la orden de `axiom branch` que repiten las continuaciones de los
	// rechazos; Master, la carpeta maestra que estas citan con `--cwd`.
	Operation string
	Lane      Lane
	Change    string
	Topology  workspace.TopologyType
	Master    string
	// WorkspaceRoot es el del checkout; WorkspaceRel, su ruta relativa a la raíz git
	// del planning home, con «/» (`.` si coinciden), para situarlo dentro del worktree.
	WorkspaceRoot string
	WorkspaceRel  string
	PlanningRepo  RepoFacts
	CodeRepos     []RepoFacts
	Skipped       []SkippedPath
	// HasKickoff dice si el cambio tiene `kickoff.yaml`; SealedRoles son sus roles.
	// Antes del sellado el plan solo contiene la rama del cambio en el planning home.
	HasKickoff   bool
	SealedRoles  []string
	Policy       Policy
	Exceptions   Exceptions
	WorktreeRoot string
}

// ChangePlan es el plan de un cambio: sus repositorios en orden estable (primero el
// del planning home y después por nombre) y, en cada uno, sus ramas (la del cambio y
// las de rol, estas en orden alfabético).
type ChangePlan struct {
	Lane         Lane
	Change       string
	WorktreeRoot string
	// WorkspaceRoot y PlanningHome son los del worktree de la rama del cambio del
	// planning home; con el repositorio de specs de un multirepo, que se trabaja in
	// situ, son los del checkout.
	WorkspaceRoot string
	PlanningHome  string
	OddDocument   string // solo en el carril ODD
	Repositories  []RepoPlan
	Skipped       []SkippedPath
	Warnings      []string
}

// RepoPlan es el plan de un repositorio.
type RepoPlan struct {
	Name      string // `<repo>` en la ruta del worktree (REQ-25.11, S-17)
	GitRoot   string
	CommonDir string
	RelPath   string
	Role      RepoRole
	// Roles son los roles del cambio que tocan este repositorio (los sellados que lo
	// declaran, o `fullstack`), en orden alfabético.
	Roles             []string
	IntegrationBranch string
	// BaseRef es la base fijada por la excepción `--base`; vacía, la resuelve el motor
	// desde la rama de integración (design D-06).
	BaseRef  string
	Branches []BranchPlan
}

// BranchPlan es una rama del plan.
type BranchPlan struct {
	Name   string
	Kind   BranchKind
	RoleID string // solo para Kind == KindRole
	// From es la rama de la que sale una rama de rol (la de seguimiento). En la rama
	// del cambio queda vacía: sale de la base del repositorio.
	From     string
	Worktree string // absoluta
	PRTarget string
}

// Plan calcula el plan de un cambio en monorepo embebido o en multirepo (REQ-25.4,
// REQ-25.5, REQ-25.11, REQ-25.13). Es determinista: la misma entrada da el mismo plan
// sea cual sea el orden de los roles y de los repositorios que se le entregan, y no
// altera su entrada. Devuelve un rechazo, sin plan, ante un nombre inválido
// (`invalid_name`) o un valor inválido de la política (`config_invalid`).
func Plan(in PlanInput) (ChangePlan, *Refusal) {
	if refusal := ValidateChangeName(in.Operation, in.Master, in.Change); refusal != nil {
		return ChangePlan{}, refusal
	}
	if refusal := validateExceptions(in); refusal != nil {
		return ChangePlan{}, refusal
	}

	sealed := sealedRoles(in)
	plan := ChangePlan{
		Lane:         in.Lane,
		Change:       in.Change,
		WorktreeRoot: in.WorktreeRoot,
		Skipped:      sortedSkipped(in.Skipped),
		Warnings:     untouchedRoleWarnings(in, sealed),
	}
	for _, selection := range selectRepositories(in, sealed) {
		repo, refusal := planRepository(in, selection)
		if refusal != nil {
			return ChangePlan{}, refusal
		}
		plan.Repositories = append(plan.Repositories, repo)
		if tracking := trackingBranch(repo); tracking != "" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("Publica la rama de seguimiento %s del repositorio %s en el remoto antes del primer PR de rol; axiom branch nunca ejecuta push.", tracking, repo.Name))
		}
	}
	plan.WorkspaceRoot, plan.PlanningHome = plannedWorkspace(in, plan.Repositories[0])
	if in.Lane == LaneODD {
		plan.OddDocument = filepath.Join(plan.WorkspaceRoot, "odd", "tasks", in.Change+".md")
	}
	return plan, nil
}

// repoSelection es un repositorio que entra al plan con los roles del cambio que lo
// tocan.
type repoSelection struct {
	facts    RepoFacts
	planning bool
	roles    []string
}

// selectRepositories elige los repositorios del plan. El del planning home entra
// siempre; los demás solo cuando algún rol sellado (o `fullstack`) los toca, y solo
// después del sellado y en el carril SDD: el carril ODD no divide en roles y, antes
// del sellado, los roles del cambio aún no se conocen (S-1).
func selectRepositories(in PlanInput, sealed []string) []repoSelection {
	selections := []repoSelection{{facts: in.PlanningRepo, planning: true, roles: touchingRoles(in.PlanningRepo, sealed)}}
	if len(sealed) == 0 {
		return selections
	}
	code := slices.Clone(in.CodeRepos)
	slices.SortFunc(code, func(a, b RepoFacts) int {
		if byName := strings.Compare(repoName(a), repoName(b)); byName != 0 {
			return byName
		}
		return strings.Compare(a.GitRoot, b.GitRoot)
	})
	for _, facts := range code {
		if roles := touchingRoles(facts, sealed); len(roles) > 0 {
			selections = append(selections, repoSelection{facts: facts, roles: roles})
		}
	}
	return selections
}

// sealedRoles devuelve los roles del kickoff del cambio, sin repetidos (sin distinguir
// mayúsculas) y en orden alfabético, o ninguno si el carril es ODD o el cambio no tiene
// kickoff.
func sealedRoles(in PlanInput) []string {
	if in.Lane == LaneODD || !in.HasKickoff {
		return nil
	}
	seen := map[string]bool{}
	var roles []string
	for _, role := range in.SealedRoles {
		if key := strings.ToLower(role); role != "" && !seen[key] {
			seen[key] = true
			roles = append(roles, role)
		}
	}
	slices.SortFunc(roles, func(a, b string) int {
		if byFold := strings.Compare(strings.ToLower(a), strings.ToLower(b)); byFold != 0 {
			return byFold
		}
		return strings.Compare(a, b)
	})
	return roles
}

// touchingRoles son los roles sellados que tocan un repositorio (design D-04, S-3):
// los que lo declaran en axiom.yaml, y `fullstack`, que toca todos. Conserva el orden
// de sealed.
func touchingRoles(facts RepoFacts, sealed []string) []string {
	var roles []string
	for _, role := range sealed {
		if multirole.IsReservedRole(role) || declaresRole(facts.Roles, role) {
			roles = append(roles, role)
		}
	}
	return roles
}

func declaresRole(declared []string, role string) bool {
	return slices.ContainsFunc(declared, func(candidate string) bool { return strings.EqualFold(candidate, role) })
}

// untouchedRoleWarnings avisa de cada rol sellado que no toca ningún repositorio
// declarado: un rol desconocido para axiom.yaml, o que no declara repositorios,
// no aporta ramas (REQ-25.4). Un rol cuyas rutas se omitieron ya figura en `skipped`.
func untouchedRoleWarnings(in PlanInput, sealed []string) []string {
	var warnings []string
	for _, role := range sealed {
		if multirole.IsReservedRole(role) || declaresRole(in.PlanningRepo.Roles, role) {
			continue
		}
		if slices.ContainsFunc(in.CodeRepos, func(facts RepoFacts) bool { return declaresRole(facts.Roles, role) }) {
			continue
		}
		if slices.ContainsFunc(in.Skipped, func(skipped SkippedPath) bool { return strings.EqualFold(skipped.Role, role) }) {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("El rol «%s» del kickoff no toca ningún repositorio declarado en axiom.yaml: no aporta ramas.", role))
	}
	return warnings
}

// sortedSkipped devuelve una copia ordenada de las rutas omitidas, o nil si no hay.
func sortedSkipped(skipped []SkippedPath) []SkippedPath {
	if len(skipped) == 0 {
		return nil
	}
	sorted := slices.Clone(skipped)
	slices.SortFunc(sorted, func(a, b SkippedPath) int {
		if byPath := strings.Compare(a.Path, b.Path); byPath != 0 {
			return byPath
		}
		return strings.Compare(a.Role, b.Role)
	})
	return sorted
}

// planRepository planifica las ramas de un repositorio (REQ-25.4): con un rol (o con
// ninguno), solo `<prefijo>/<cambio>`; con dos o más, esa es la rama de seguimiento y
// cada rol saca `<prefijo>/<cambio>-<rol>` desde ella. El repositorio de specs de un
// multirepo se trabaja in situ y no recibe ramas (REQ-25.5).
func planRepository(in PlanInput, selection repoSelection) (RepoPlan, *Refusal) {
	facts := selection.facts
	name := repoName(facts)
	effective := repoPolicy(in.Policy, facts)
	if refusal := validatePolicyValue(in, name, "integration_branch", effective.IntegrationBranch, gitref.ValidRefName); refusal != nil {
		return RepoPlan{}, refusal
	}
	repo := RepoPlan{
		Name:              name,
		GitRoot:           facts.GitRoot,
		CommonDir:         facts.CommonDir,
		RelPath:           facts.RelPath,
		Role:              repositoryRole(in.Topology, selection.planning),
		Roles:             slices.Clone(selection.roles),
		IntegrationBranch: effective.IntegrationBranch,
		BaseRef:           in.Exceptions.Base,
	}
	if repo.Role == RepoSpecsInPlace {
		return repo, nil
	}

	// La excepción `--prefix` sustituye al prefijo de la política, que entonces no se
	// usa y no hace falta validar.
	prefix := in.Exceptions.Prefix
	if prefix == "" {
		prefix = effective.BranchPrefix
		if refusal := validatePolicyValue(in, name, "branch_prefix", prefix, gitref.ValidPrefix); refusal != nil {
			return RepoPlan{}, refusal
		}
	}
	changeBranch := prefix + "/" + in.Change
	if in.Exceptions.Branch != "" {
		changeBranch = in.Exceptions.Branch
	}
	repo.Branches = []BranchPlan{{
		Name:     changeBranch,
		Kind:     KindChange,
		PRTarget: effective.IntegrationBranch,
		Worktree: filepath.Join(in.WorktreeRoot, in.Change, name),
	}}
	if len(selection.roles) < 2 {
		return repo, nil
	}
	for _, role := range selection.roles {
		if refusal := validateRoleName(in, role, changeBranch); refusal != nil {
			return RepoPlan{}, refusal
		}
		repo.Branches = append(repo.Branches, BranchPlan{
			Name:     changeBranch + "-" + role,
			Kind:     KindRole,
			RoleID:   role,
			From:     changeBranch,
			PRTarget: changeBranch,
			Worktree: filepath.Join(in.WorktreeRoot, in.Change+"-"+role, name),
		})
	}
	return repo, nil
}

// repositoryRole es el papel de un repositorio según la topología (REQ-25.5): en
// monorepo embebido todos reciben el modelo de ramas del cambio; en multirepo, el
// repositorio de specs se trabaja in situ y los demás son de código.
func repositoryRole(topology workspace.TopologyType, planning bool) RepoRole {
	switch {
	case topology != workspace.TopologyMultirepo:
		return RepoChangeBranch
	case planning:
		return RepoSpecsInPlace
	default:
		return RepoCode
	}
}

// repoPolicy es la política efectiva de un repositorio (REQ-25.3): la entrada de
// `repositories:` o, sin ella, la del workspace. Un repositorio fuera de la carpeta
// maestra solo hereda la del workspace.
func repoPolicy(policy Policy, facts RepoFacts) RepoPolicy {
	if facts.RelPath == "" {
		return RepoPolicy{IntegrationBranch: policy.IntegrationBranch, BranchPrefix: policy.BranchPrefix}
	}
	return policy.For(facts.RelPath)
}

// repoName es el `<repo>` de la ruta del worktree (REQ-25.11, S-17): el último
// segmento de la ruta declarada y, para `.`, el nombre del directorio de la raíz git.
func repoName(facts RepoFacts) string {
	declared := path.Clean(strings.ReplaceAll(strings.TrimSpace(facts.DeclaredPath), `\`, "/"))
	if name := path.Base(declared); name != "." && name != ".." && name != "/" {
		return name
	}
	return filepath.Base(facts.GitRoot)
}

// trackingBranch devuelve la rama de seguimiento del repositorio, o la cadena vacía si
// no tiene ramas de rol.
func trackingBranch(repo RepoPlan) string {
	for _, branch := range repo.Branches {
		if branch.Kind == KindRole {
			return repo.Branches[0].Name
		}
	}
	return ""
}

// plannedWorkspace sitúa el workspaceRoot y el planning home del plan: dentro del
// worktree de la rama del cambio del planning home, con la ruta relativa que tienen en
// el checkout, o en el propio checkout cuando el repositorio no tiene worktree (el de
// specs de un multirepo, que se trabaja in situ).
func plannedWorkspace(in PlanInput, planning RepoPlan) (workspaceRoot, planningHome string) {
	workspaceRoot = in.WorkspaceRoot
	if len(planning.Branches) > 0 && planning.Branches[0].Worktree != "" {
		workspaceRoot = planning.Branches[0].Worktree
		if rel := in.WorkspaceRel; rel != "" && rel != "." {
			workspaceRoot = filepath.Join(workspaceRoot, filepath.FromSlash(rel))
		}
	}
	// Sin configuración, Roots devuelve el planning home literal bajo la raíz.
	_, planningHome = workspace.Roots(nil, workspaceRoot)
	return workspaceRoot, planningHome
}
