package sddstatus

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/explainer"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ExplainerProjection es Status.Explainer. Es informativa: nunca cambia
// nextRecommended, las dependencias, blockedReasons, la proyección iso ni el digest de una
// puerta (decorateExplainer no toca ninguno de esos campos).
type ExplainerProjection struct {
	// Project es sdd.explainer_html del proyecto: ask, always, never o invalid.
	Project string `json:"project"`
	// Reason solo acompaña a invalid: el error y su continuación.
	Reason string `json:"reason,omitempty"`
	// Error es el fallo al descubrir los md del cambio o al leer su kickoff.
	Error  string                       `json:"error,omitempty"`
	Spec   *ExplainerArtifactProjection `json:"spec,omitempty"`
	Design *ExplainerArtifactProjection `json:"design,omitempty"`
}

// ExplainerArtifactProjection es la frescura del HTML de un tipo.
type ExplainerArtifactProjection struct {
	// State es fresh, stale o missing; vacío solo junto a Error.
	State string `json:"state,omitempty"`
	// Path es la ruta absoluta de resumen-<kind>.html.
	Path string `json:"path"`
	// ExpectedDigest es el sha256:<hex> de los md actuales: el valor que la fase copia al <meta>.
	ExpectedDigest string `json:"expectedDigest,omitempty"`
	// Error es el motivo de un HTML ilegible o mayor que el límite, o de un md ilegible.
	Error string `json:"error,omitempty"`
}

// ExplainerFreshnessResult es el resultado común de `axiom sdd status` y del dashboard.
type ExplainerFreshnessResult struct {
	// Sealed es el valor sellado efectivo: yes o no (la clave ausente es no).
	Sealed kickoff.ExplainerHTML
	// Enabled es verdadero con Sealed yes y un almacén de ficheros (openspec o hybrid).
	Enabled bool
	// Error es un kickoff ilegible o un fallo al descubrir los md del cambio.
	Error string
	// Spec y Design son nil sin md de origen y con Enabled falso.
	Spec   *ExplainerArtifactProjection
	Design *ExplainerArtifactProjection
}

// DeclaredArtifactStore es el envoltorio exportado de declaredArtifactStore: el almacén que
// declara openspec/config.yaml del workspace, y si lo declara.
func DeclaredArtifactStore(workspaceRoot string) (ArtifactStore, bool) {
	return declaredArtifactStore(workspaceRoot)
}

// ExplainerFreshness lee el kickoff.yaml de la carpeta del cambio, aplica la condición (sellado
// yes y almacén openspec o hybrid) y calcula la frescura de cada HTML. Ningún fallo es un error
// de Go: se devuelve en Error o en el Error del tipo afectado.
func ExplainerFreshness(changeRoot string, store ArtifactStore) ExplainerFreshnessResult {
	result := ExplainerFreshnessResult{Sealed: kickoff.ExplainerHTMLNo}
	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if sealed == nil {
		return result
	}
	result.Sealed = kickoff.EffectiveExplainerHTML(sealed.Config.ExplainerHTML)
	result.Enabled = result.Sealed == kickoff.ExplainerHTMLYes &&
		(store == ArtifactStoreOpenSpec || store == ArtifactStoreHybrid)
	if !result.Enabled {
		return result
	}

	paths, err := ChangeArtifactPaths(changeRoot)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Spec = explainerArtifactFreshness(changeRoot, explainer.KindSpec, paths.Specs)
	result.Design = explainerArtifactFreshness(changeRoot, explainer.KindDesign, paths.Design)
	return result
}

// explainerArtifactFreshness aplica la regla por tipo: sin md de origen el tipo se omite (nil).
func explainerArtifactFreshness(changeRoot string, kind explainer.Kind, sources []string) *ExplainerArtifactProjection {
	if len(sources) == 0 {
		return nil
	}
	projection := &ExplainerArtifactProjection{Path: filepath.Join(changeRoot, explainer.FileName(kind))}
	digest, err := kickoff.ArtifactDigest(sources)
	if err != nil {
		projection.Error = err.Error()
		return projection
	}
	projection.ExpectedDigest = explainer.FormatDigest(digest)

	html, err := explainer.ReadFile(projection.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		projection.State = string(explainer.StateMissing)
	case err != nil:
		projection.Error = err.Error()
	default:
		if actual, ok := explainer.SourceDigest(html); ok && actual == projection.ExpectedDigest {
			projection.State = string(explainer.StateFresh)
		} else {
			projection.State = string(explainer.StateStale)
		}
	}
	return projection
}

// decorateExplainer proyecta sdd.explainer_html y la frescura del HTML del cambio seleccionado.
// Es aditivo: el bloque se omite salvo que axiom.yaml tenga bloque `sdd:` (válido o inválido) o
// sea ilegible, o que el cambio seleccionado, no archivado, esté sellado con explainer_html yes.
// La mera existencia de un resumen-*.html no lo hace aparecer. No toca NextRecommended,
// Dependencies, BlockedReasons, Notes, ISO ni Governance.Gates.
func decorateExplainer(status *Status, location workspace.Location) {
	declarations := location.Declarations
	declared := declarations.SDD != nil || declarations.ParseErr != nil || declarations.SDDErr != nil

	var result ExplainerFreshnessResult
	if status.ChangeName != nil && status.Archived == nil && status.ChangeRoot != nil {
		result = ExplainerFreshness(*status.ChangeRoot, status.ArtifactStore)
	}
	if !declared && !result.Enabled && result.Error == "" {
		return
	}

	projection := &ExplainerProjection{Project: string(workspace.ExplainerHTMLAsk)}
	mode, err := declarations.ExplainerHTML()
	if err != nil {
		projection.Project = "invalid"
		projection.Reason = err.Error() + "; corrígela con `axiom workspace validate --path " + pathquote.Quote(location.Master) + "`"
	} else {
		projection.Project = string(mode)
	}
	projection.Error = result.Error
	projection.Spec = result.Spec
	projection.Design = result.Design
	status.Explainer = projection
}
