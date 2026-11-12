package livespec

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/techcontext"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ReferencePathError es el rechazo de una rama que modifica, crea o elimina ficheros bajo
// `technical_context.reference.paths` (REQ-30.6 a, REQ-24.30). Files son esos ficheros,
// relativos a la raíz del workspace y con `/`.
type ReferencePathError struct {
	Files []string
}

func (e *ReferencePathError) Error() string {
	return fmt.Sprintf("la rama modifica ficheros de referencia (modo reference de `technical_context` en axiom.yaml), que nunca se modifican: %s; revierte esos cambios (por ejemplo con `git restore --source=<base> -- <fichero>` o `git rm` si los creaste) y vuelve a ejecutar `axiom archive integrate --odd <feature>`; si el fichero debe poder editarse, muévelo a `technical_context.maintained.paths` en axiom.yaml",
		strings.Join(e.Files, ", "))
}

// BaseUnresolvedError es el rechazo, con fallo cerrado, de una rama cuya base no puede
// determinarse: sin ella no se puede garantizar la regla de referencia (REQ-30.6).
type BaseUnresolvedError struct {
	// Repository es la raíz del repositorio git cuya base no se resuelve.
	Repository string
	Feature    string
	Workspace  string
	Cause      error
}

func (e *BaseUnresolvedError) Error() string {
	feature, cwd := e.Feature, "<ruta>"
	if feature == "" {
		feature = "<feature>"
	}
	if e.Workspace != "" {
		cwd = pathquote.Quote(e.Workspace)
	}
	return fmt.Sprintf("no se puede determinar la base de la rama en el repositorio %s: %v; sin ella no se puede garantizar que la rama no modifique `technical_context.reference`; comprueba el plan con `axiom branch status --lane odd --change %s --cwd %s` y, si el repositorio no forma parte del plan de la feature, inclúyelo con `axiom branch ensure --lane odd --change %s --repo <ruta> --cwd %s`",
		pathquote.Quote(e.Repository), e.Cause, feature, cwd, feature, cwd)
}

func (e *BaseUnresolvedError) Unwrap() error { return e.Cause }

// repositoryBase resuelve la referencia git que sirve de base de la rama en un
// repositorio (D7). Es una variable de paquete solo para que las pruebas fijen la base
// sin montar el plan de ramas completo.
var repositoryBase = engineBase

// engineBase pide al motor de ramas el plan de la feature en el carril ODD y elige la
// base del repositorio indicado. Solo lee; nunca ejecuta `fetch`.
func engineBase(ctx context.Context, loc workspace.Location, feature, repoRoot string) (string, error) {
	result := engine.Default().Status(ctx, loc.WorkspaceRoot, feature, branch.LaneODD)
	if result.Status != branch.StatusReady && result.Status != branch.StatusNotStarted {
		reason := result.Message
		if reason == "" {
			reason = "el estado de las ramas de la feature es " + string(result.Status)
		}
		return "", errors.New(reason)
	}
	for _, repo := range result.Repositories {
		if pathidentity.SameDirectory(repo.GitRoot, repoRoot) {
			return baseOf(ctx, repoRoot, repo)
		}
	}
	return "", fmt.Errorf("el repositorio no forma parte del plan de la feature %s", feature)
}

// baseOf elige la base del repositorio entre las referencias locales (D7): `@{upstream}`
// si HEAD ya es la rama de integración (el repositorio de specs de un multirepo); si no,
// `origin/<prTarget>` si existe y, si no, `<prTarget>`, con el `prTarget` de la rama de la
// feature o, sin rama, la rama de integración. Nunca consulta el remoto.
func baseOf(ctx context.Context, repoRoot string, repo branch.RepositoryResult) (string, error) {
	unresolved := func(cause error) error { return &BaseUnresolvedError{Repository: repoRoot, Cause: cause} }

	if repo.CurrentBranch != "" && repo.CurrentBranch == repo.IntegrationBranch {
		if !commitResolves(ctx, repoRoot, "@{upstream}") {
			return "", unresolved(fmt.Errorf("HEAD ya es la rama de integración %q y no tiene upstream configurado", repo.IntegrationBranch))
		}
		return "@{upstream}", nil
	}

	target := repo.IntegrationBranch
	for _, candidate := range repo.Branches {
		if candidate.PRTarget == "" {
			continue
		}
		target = candidate.PRTarget
		if candidate.Name == repo.CurrentBranch {
			break
		}
	}
	if target == "" {
		return "", unresolved(errors.New("el plan no indica la rama destino ni la rama de integración"))
	}
	for _, ref := range []string{"origin/" + target, target} {
		if commitResolves(ctx, repoRoot, ref) {
			return ref, nil
		}
	}
	return "", unresolved(fmt.Errorf("ni origin/%s ni %s existen en local (Axiom no ejecuta `fetch`)", target, target))
}

// commitResolves dice si la referencia se resuelve a un commit del repositorio.
func commitResolves(ctx context.Context, repoRoot, ref string) bool {
	result, err := gitcommon.Run(ctx, repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil && result.ExitCode == 0
}

// checkContext es la comprobación estructural del contexto técnico de la rama (REQ-30.6).
// Sin `technical_context` declarado no produce informe ni error. Con él calcula, por cada
// repositorio que contiene una ruta declarada y contra su base (D7), los ficheros que la
// rama cambió (confirmados, en el índice, sin confirmar y no seguidos) y los clasifica
// con techcontext.Scope: cualquiera bajo una ruta de referencia es un rechazo; los de las
// rutas mantenidas se informan sin bloquear. No juzga el contenido de la edición.
func checkContext(ctx context.Context, loc workspace.Location, feature string) (*ContextReport, error) {
	if loc.Declarations.TechnicalContext == nil {
		return nil, nil
	}
	scope := techcontext.ScopeOf(loc)

	var repositories []string
	seen := map[string]bool{}
	for _, declared := range append(append([]string{}, scope.Maintained...), scope.Reference...) {
		root, err := techcontext.RepositoryRoot(ctx, scope.Resolve(declared))
		if err != nil {
			return nil, &BaseUnresolvedError{Repository: scope.Resolve(declared), Feature: feature, Workspace: loc.WorkspaceRoot, Cause: err}
		}
		if !seen[root] {
			seen[root] = true
			repositories = append(repositories, root)
		}
	}

	maintained, reference := map[string]bool{}, map[string]bool{}
	workspaceRoot := techcontext.RealPath(loc.WorkspaceRoot)
	for _, root := range repositories {
		base, err := repositoryBase(ctx, loc, feature, root)
		if err == nil && !commitResolves(ctx, root, base) {
			err = fmt.Errorf("la base %q no se resuelve a un commit", base)
		}
		if err != nil {
			var unresolved *BaseUnresolvedError
			if errors.As(err, &unresolved) {
				unresolved.Feature, unresolved.Workspace = feature, loc.WorkspaceRoot
				return nil, unresolved
			}
			return nil, &BaseUnresolvedError{Repository: root, Feature: feature, Workspace: loc.WorkspaceRoot, Cause: err}
		}
		changed, err := techcontext.ChangedSince(ctx, root, base)
		if err != nil {
			return nil, fmt.Errorf("no se pueden calcular los ficheros que la rama cambió en %s frente a %s: %w; comprueba el repositorio con `git -C %s status` y vuelve a ejecutar `axiom archive integrate --odd %s`",
				pathquote.Quote(root), base, err, pathquote.Quote(root), pathquote.ShellWord(feature))
		}
		for _, file := range changed {
			switch scope.Classify(file) {
			case techcontext.Reference:
				reference[relativeTo(workspaceRoot, file)] = true
			case techcontext.Maintained:
				maintained[relativeTo(workspaceRoot, file)] = true
			}
		}
	}

	if len(reference) > 0 {
		return nil, &ReferencePathError{Files: sortedSet(reference)}
	}
	return &ContextReport{Declared: true, MaintainedTouched: sortedSet(maintained), ReferenceTouched: []string{}}, nil
}

// relativeTo da la ruta de file relativa a la raíz del workspace con `/`, o la absoluta si
// no se puede relativizar (otra unidad).
func relativeTo(workspaceRoot, file string) string {
	rel, err := filepath.Rel(workspaceRoot, file)
	if err != nil {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}

func sortedSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
