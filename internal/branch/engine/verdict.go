package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// VerdictQuery es la consulta del veredicto de rama (design D-10, §4.2). Path es la carpeta del
// cambio, un fichero o un repositorio: de él sale el repositorio cuya rama se comprueba.
type VerdictQuery struct {
	Path       string
	Change     string
	Lane       branch.Lane
	Context    branch.Context
	Consultant branch.Consultant
	// SealRoles son los roles que se van a sellar (solo `kickoff seal`): antes del kickoff las
	// ramas de rol que serían correctas salen de ellos.
	SealRoles []string
}

// VerdictFor responde «¿esta rama es la que corresponde a este cambio en este repositorio según
// su papel?» (REQ-25.18, design D-10). Es la consulta única de las guardas A, B y D y obtiene
// todos los hechos leyendo ficheros, sin lanzar un solo proceso de git (REQ-25.21: lee `HEAD`
// sin procesos; REQ-25.42: si git no está disponible, no bloquea): `workspace.Locate`,
// `gitcommon.GitRootFromFiles`, `gitcommon.CommonDirFromFiles`, `gitcommon.ReadHead`,
// `kickoff.Load`, el registro local y la política de `axiom.yaml`.
//
// Un veredicto `not_applicable` explica su exención en Detail (un cambio con `kickoff.yaml` sin
// instantánea, REQ-25.37, y los sumideros acumulativos por nombre, REQ-25.18) y, cuando no pudo
// comprobar la rama (una ruta fuera de git, un `HEAD` ilegible), además avisa en Warning sin
// bloquear; así también un cambio sin kickoff cuyo nombre no forma un nombre de rama válido.
// El `error` se reserva para los fallos internos —configuración, kickoff o registro
// ilegibles— y es siempre un *branch.RefusalError `config_invalid` con su continuación.
//
// Los nombres esperados salen, para la rama del cambio y las de rol, del registro local
// (REQ-25.12) y, sin él, de la política; la instantánea del kickoff se suma con el escritor
// del sellado (S8-c2). No consulta los cambios activos de la rama de integración.
func (e *Engine) VerdictFor(ctx context.Context, q VerdictQuery) (branch.Verdict, error) {
	if err := ctx.Err(); err != nil {
		return branch.Verdict{}, err
	}
	in := branch.VerdictInput{Change: q.Change, Lane: q.Lane, Context: q.Context, Consultant: q.Consultant}
	if in.Lane == "" {
		in.Lane = branch.LaneSDD
	}
	if in.Context == "" {
		in.Context = branch.ContextOrdinary
	}
	if in.Consultant == "" {
		in.Consultant = branch.ConsultantWrite
	}

	if isCumulativeSink(q.Change) {
		return exempt(in, fmt.Sprintf("«%s» es un sumidero acumulativo y no un cambio: queda exento de las guardas de rama (REQ-25.18)", q.Change)), nil
	}
	start := existingDirectory(q.Path)
	root, inGit := gitcommon.GitRootFromFiles(start)
	if !inGit {
		return unchecked(in, fmt.Sprintf("%s no pertenece a ningún repositorio git", start)), nil
	}

	loc, err := workspace.Locate(start)
	if err != nil {
		return branch.Verdict{}, &branch.RefusalError{Refusal: *branch.LocateRefusal(err)}
	}
	if refusal := branch.ConfigRefusal(loc); refusal != nil {
		return branch.Verdict{}, &branch.RefusalError{Refusal: *refusal}
	}

	if in.Lane != branch.LaneODD {
		sealed, err := kickoff.Load(filepath.Join(loc.PlanningHome, "changes", q.Change))
		if err != nil {
			return branch.Verdict{}, kickoffUnreadable(loc, q.Change, err)
		}
		if sealed != nil {
			return exempt(in, "el kickoff.yaml del cambio no tiene instantánea de política git: queda exento de las guardas de rama (REQ-25.37)"), nil
		}
	}

	// Sin kickoff, el nombre esperado sale del registro y de la política, y el nombre
	// del cambio compone la rama y el fichero del registro: uno que no forma un nombre de rama válido
	// (un espacio, un carácter reservado) no tiene rama que comprobar. Es `not_applicable` con aviso,
	// no un fallo interno: la guarda no bloquea lo que no puede comprobar. Va antes de abrir el
	// registro, que no puede nombrarse con ese cambio.
	if q.Change != "" {
		if invalid := gitref.ValidBranchComponent(q.Change); invalid != nil {
			return invalidChangeName(in, invalid), nil
		}
	}

	planningRoot, ok := gitcommon.GitRootFromFiles(loc.WorkspaceRoot)
	if !ok {
		return unchecked(in, fmt.Sprintf("%s no pertenece a ningún repositorio git", loc.WorkspaceRoot)), nil
	}
	planningCommon, err := gitcommon.CommonDirFromFiles(planningRoot)
	if err != nil {
		return unchecked(in, err.Error()), nil
	}
	repoCommon := planningCommon
	isPlanning := pathidentity.SameDirectory(root, planningRoot)
	if !isPlanning {
		if repoCommon, err = gitcommon.CommonDirFromFiles(root); err != nil {
			return unchecked(in, err.Error()), nil
		}
	}
	current, detached, err := gitcommon.ReadHead(root)
	if err != nil {
		return unchecked(in, err.Error()), nil
	}

	master := projectMaster(loc, branch.RepoFacts{GitRoot: planningRoot, CommonDir: planningCommon})
	record, err := e.loadRecord(planningCommon, in.Lane, q.Change)
	if err != nil {
		return branch.Verdict{}, &branch.RefusalError{Refusal: *registryRefusal(err, master, q.Change)}
	}

	in.Role = roleOf(loc, isPlanning)
	in.Repo = filepath.Base(root)
	in.Current, in.Detached = current, detached
	in.Master = master
	in.Expected = expectedNames(q, loc, record, root, repoCommon)
	return branch.Decide(in), nil
}

// exempt es el veredicto de una consulta exenta, sin aviso.
func exempt(in branch.VerdictInput, reason string) branch.Verdict {
	in.Exempt = reason
	return branch.Decide(in)
}

// unchecked es el veredicto de una consulta que no pudo comprobar la rama (una ruta fuera de
// git, un `HEAD` ilegible): `not_applicable`, con el aviso que lo dice, porque la guarda no
// bloquea lo que no puede comprobar (REQ-25.42).
func unchecked(in branch.VerdictInput, reason string) branch.Verdict {
	verdict := exempt(in, "no se comprobó la rama del cambio: "+reason)
	verdict.Warning = fmt.Sprintf("no se pudo comprobar la rama del cambio %s: %s; no se bloquea", in.Change, reason)
	return verdict
}

// invalidChangeName es el veredicto de un cambio sin kickoff cuyo nombre no forma un nombre de rama
// válido (invalid dice la regla incumplida): `not_applicable`, con el aviso que lo dice, sin
// continuación y sin bloquear.
func invalidChangeName(in branch.VerdictInput, invalid error) branch.Verdict {
	reason := fmt.Sprintf("el nombre del cambio %q no es un nombre de rama válido (%v)", in.Change, invalid)
	verdict := exempt(in, reason+": no se comprueba la rama del cambio")
	verdict.Warning = reason + "; no se comprueba la rama"
	return verdict
}

// isCumulativeSink dice si name es un sumidero acumulativo, `e2e-cumulative` o `cumulative-<rol>`:
// viven en `changes/` pero no son cambios y quedan exentos por nombre (REQ-25.18).
func isCumulativeSink(name string) bool {
	return name == "e2e-cumulative" || cumulativeSink.MatchString(name)
}

// existingDirectory devuelve el directorio existente más cercano a path: él mismo si es un
// directorio, su carpeta si es un fichero, y el primer ascendiente que existe si la ruta aún no
// existe (la carpeta de un cambio que se va a crear, un documento ODD nuevo).
func existingDirectory(path string) string {
	current := filepath.Clean(path)
	for {
		if info, err := os.Stat(current); err == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return current
		}
		current = parent
	}
}

// kickoffUnreadable es el fallo interno de un `kickoff.yaml` que no se puede leer: no se trata
// como «sin sellar» (T-10), la guarda falla en cerrado.
func kickoffUnreadable(loc workspace.Location, change string, cause error) *branch.RefusalError {
	path := filepath.Join(loc.PlanningHome, "changes", change, kickoff.KickoffFileName)
	return &branch.RefusalError{Refusal: branch.Refusal{
		Code:         branch.CodeConfigInvalid,
		Message:      fmt.Sprintf("el kickoff del cambio %s no se puede leer: %v", change, cause),
		Continuation: fmt.Sprintf("restaura o corrige %s y comprueba con `axiom sdd status %s --cwd %s`", pathquote.Quote(path), change, pathquote.Quote(loc.Master)),
		Details:      map[string]string{"path": path},
	}}
}

// loadRecord lee el registro local del cambio, o nil si no hay (o el motor se montó sin
// registro, como en las pruebas que lo construyen a mano).
func (e *Engine) loadRecord(commonDir string, lane branch.Lane, change string) (*branch.Record, error) {
	if e.registry == nil {
		return nil, nil
	}
	return e.registry.Load(commonDir, lane, change)
}

// roleOf es el papel del repositorio consultado según la topología (REQ-25.5): en monorepo
// embebido todos reciben el modelo de ramas del cambio; en multirepo, el repositorio del
// planning home (el de specs) se trabaja in situ y los demás son de código.
func roleOf(loc workspace.Location, isPlanning bool) branch.RepoRole {
	switch {
	case topologyOf(loc) != workspace.TopologyMultirepo:
		return branch.RepoChangeBranch
	case isPlanning:
		return branch.RepoSpecsInPlace
	default:
		return branch.RepoCode
	}
}

// expectedNames compone los nombres esperados del repositorio cuya raíz es root y cuyo git
// common dir es repoCommon (design D-10, «Nombres esperados»): para la rama del cambio y las de
// rol, el registro (la rama registrada o las excepciones de REQ-25.12) y, sin él, la política del
// repositorio; para la de archivado, el registro si `ensure --archive` recibió `--prefix` y, si no,
// `<rama del cambio>-archive`. El carril ODD no tiene rama de archivado.
func expectedNames(q VerdictQuery, loc workspace.Location, record *branch.Record, root, repoCommon string) branch.ExpectedNames {
	policy := branch.ResolvePolicy(loc.Config)
	effective := branch.RepoPolicy{IntegrationBranch: policy.IntegrationBranch, BranchPrefix: policy.BranchPrefix}
	if relative, inside := relativeWithin(loc.Master, root); inside {
		effective = policy.For(relative)
	}

	recordedChange, recordedRoles, recordedArchive := recordedBranches(record, repoCommon)
	changeBranch := recordedChange
	if changeBranch == "" {
		switch {
		case record != nil && record.Exceptions.Branch != "":
			changeBranch = record.Exceptions.Branch
		case record != nil && record.Exceptions.Prefix != "":
			changeBranch = record.Exceptions.Prefix + "/" + q.Change
		default:
			changeBranch = effective.BranchPrefix + "/" + q.Change
		}
	}
	names := branch.ExpectedNames{Change: changeBranch, Integration: effective.IntegrationBranch}

	// Con dos o más roles que tocan el repositorio hay ramas de rol (REQ-25.4); las registradas
	// valen aunque el kickoff aún no las nombre.
	roles := make(map[string]bool, len(recordedRoles))
	for _, name := range recordedRoles {
		roles[name] = true
	}
	if touching := rolesTouching(loc, root, q.SealRoles); len(touching) >= 2 {
		for _, role := range touching {
			roles[changeBranch+"-"+role] = true
		}
	}
	for name := range roles {
		names.Roles = append(names.Roles, name)
	}
	slices.Sort(names.Roles)

	if q.Lane != branch.LaneODD {
		switch {
		case recordedArchive != "":
			names.Archive = recordedArchive
		case record != nil && record.ArchivePrefix != "":
			names.Archive = record.ArchivePrefix + "/" + q.Change + "-archive"
		default:
			names.Archive = changeBranch + "-archive"
		}
	}
	return names
}

// recordedBranches devuelve las ramas que el registro guarda para el repositorio cuyo git common
// dir es commonDir (se reconoce por identidad de ruta, que es la misma desde el checkout
// principal y desde un worktree enlazado): la del cambio, las de rol y la de archivado.
func recordedBranches(record *branch.Record, commonDir string) (change string, roles []string, archive string) {
	if record == nil {
		return "", nil, ""
	}
	for _, repo := range record.Repositories {
		if !pathidentity.SameDirectory(repo.CommonDir, commonDir) {
			continue
		}
		for _, recorded := range repo.Branches {
			switch recorded.Kind {
			case branch.KindChange:
				change = recorded.Name
			case branch.KindRole:
				roles = append(roles, recorded.Name)
			case branch.KindArchive:
				archive = recorded.Name
			}
		}
	}
	return change, roles, archive
}

// rolesTouching son los roles, de los que se van a sellar, que tocan el repositorio cuya raíz
// es root (design D-04, S-3): los que declaran en axiom.yaml una ruta que resuelve a esa raíz git
// y `fullstack`, que toca todos. Se resuelve sobre ficheros, sin lanzar git. Salen sin repetidos
// (sin distinguir mayúsculas) y en orden alfabético, como los ordena el planificador.
func rolesTouching(loc workspace.Location, root string, sealed []string) []string {
	if len(sealed) == 0 {
		return nil
	}
	declared := map[string]bool{}
	for _, entry := range declaredPaths(loc.Config, sealed) {
		dir := entry.path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(loc.Master, filepath.FromSlash(strings.ReplaceAll(dir, `\`, "/")))
		}
		if missingPath(dir) != "" {
			continue
		}
		if declaredRoot, ok := gitcommon.GitRootFromFiles(dir); ok && pathidentity.SameDirectory(declaredRoot, root) {
			declared[strings.ToLower(entry.role)] = true
		}
	}
	seen := map[string]bool{}
	var touching []string
	for _, role := range sealed {
		key := strings.ToLower(role)
		if role == "" || seen[key] || !(multirole.IsReservedRole(role) || declared[key]) {
			continue
		}
		seen[key] = true
		touching = append(touching, role)
	}
	slices.SortFunc(touching, func(a, b string) int {
		if byFold := strings.Compare(strings.ToLower(a), strings.ToLower(b)); byFold != 0 {
			return byFold
		}
		return strings.Compare(a, b)
	})
	return touching
}
