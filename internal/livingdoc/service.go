package livingdoc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Service provee la interfaz unificada para operaciones de documentación viva.
//
// El servicio no guarda rutas: conserva solo el directorio de partida y resuelve
// el planning home con workspace.Locate en cada llamada pública (RF-PLH-10,
// D-07). El dashboard y la TUI viven mucho tiempo, y una raíz cacheada en el
// constructor quedaría obsoleta si el proyecto cambia entre dos llamadas.
type Service struct {
	start       string
	indexer     *Indexer
	synthesizer *Synthesizer
}

// NewService inicializa una nueva instancia del servicio de documentación viva.
// start es el directorio de partida de la resolución del planning home; no es
// necesario que sea la raíz del workspace.
func NewService(start string, indexer *Indexer, synthesizer *Synthesizer) *Service {
	if start == "" {
		start = "."
	}
	if indexer == nil {
		indexer = NewIndexer()
	}
	if synthesizer == nil {
		synthesizer = NewSynthesizer(indexer)
	}

	return &Service{
		start:       start,
		indexer:     indexer,
		synthesizer: synthesizer,
	}
}

// resolve localiza el planning home desde el directorio de partida. Un fallo de
// Locate se propaga tal cual: nombra la causa y una continuación accionable.
func (s *Service) resolve() (workspace.Location, error) {
	return workspace.Locate(s.start)
}

// indexerFor devuelve el indexador del servicio con el patrón de identificadores del
// workspace de loc (RF-RID-01). livingdoc es informativo y nunca falla por el patrón: si no es
// accesible (bloque `sdd:` inválido o configuración ilegible) indexa con el patrón por
// defecto y devuelve un aviso que nombra el error y la continuación (RF-RID-08).
func (s *Service) indexerFor(loc workspace.Location) (*Indexer, string) {
	pattern, err := loc.Declarations.IDPattern()
	if err != nil {
		master := loc.Master
		if master == "" {
			master = loc.WorkspaceRoot
		}
		warning := fmt.Sprintf("El patrón de identificadores de requisito no es accesible (%v); se indexa con el patrón por defecto %s hasta que lo corrijas con `axiom workspace validate --path %s`",
			err, reqid.Default().Template(), pathquote.Quote(master))
		return s.indexer.WithPattern(reqid.Default()), warning
	}
	return s.indexer.WithPattern(pattern), ""
}

func specsRootOf(loc workspace.Location) string {
	return filepath.Join(loc.PlanningHome, "specs")
}

func indexPathOf(loc workspace.Location) string {
	return filepath.Join(loc.PlanningHome, "INDEX.md")
}

// projectNameOf devuelve workspace.name de la configuración resuelta o, sin él, el
// nombre del directorio de la carpeta maestra (m11).
func projectNameOf(loc workspace.Location) string {
	if loc.Config != nil {
		if name := strings.TrimSpace(loc.Config.Workspace.Name); name != "" {
			return name
		}
	}
	return filepath.Base(loc.Master)
}

// Operaciones que escriben <planning home>/specs o <planning home>/INDEX.md, tal
// como las nombra el rechazo de la guarda (RF-TOP-12).
const (
	opSync      = "la sincronización del catálogo de especificaciones vivas (`axiom archive sync`)"
	opColdStart = "la adopción orgánica (`axiom archive coldstart`)"
)

// GetCatalog devuelve el catálogo completo de especificaciones vivas consolidadas.
// Lee solo <planning home>/specs; si existe la ubicación antigua divergente
// (Q14), la nombra en Warnings.
//
// Un `baseline:` residual (RF-TOP-12) no oculta el catálogo: se lee `<planning
// home>/specs` como siempre, porque leer no escribe, y se avisa de la causa y de
// la migración.
func (s *Service) GetCatalog(ctx context.Context) (*LivingCatalog, error) {
	loc, err := s.resolve()
	if err != nil {
		return nil, err
	}
	indexer, patternWarning := s.indexerFor(loc)
	catalog, _, err := indexer.ScanSpecs(specsRootOf(loc))
	if err != nil {
		return catalog, err
	}
	catalog.ProjectName = projectNameOf(loc)
	if patternWarning != "" {
		catalog.Warnings = append(catalog.Warnings, patternWarning)
	}
	if retired := loc.Declarations.BaselineRetiredErr; retired != nil {
		catalog.Warnings = append(catalog.Warnings, retired.Error())
	}
	if warning, ok := loc.LegacySpecsDivergence(); ok {
		catalog.Warnings = append(catalog.Warnings, warning)
	}
	return catalog, nil
}

// GetSpecDetail obtiene los metadatos y el contenido Markdown completo de una especificación viva.
func (s *Service) GetSpecDetail(domain string) (*LivingSpecEntry, string, error) {
	loc, err := s.resolve()
	if err != nil {
		return nil, "", err
	}
	specPath := filepath.Join(specsRootOf(loc), domain, "spec.md")
	contentBytes, err := os.ReadFile(specPath)
	if err != nil {
		return nil, "", fmt.Errorf("especificación viva para dominio '%s' no encontrada: %w", domain, err)
	}

	fi, _ := os.Stat(specPath)
	modTime := fi.ModTime()

	indexer, _ := s.indexerFor(loc)
	entry, _ := indexer.ParseSpecContent(string(contentBytes), domain, specPath, modTime)
	return entry, string(contentBytes), nil
}

// Sync ejecuta el re-escaneo y regenera el archivo INDEX.md del planning home.
// Con un `baseline:` residual se rechaza sin escribir nada con un
// *workspace.BaselineRetiredError (RF-TOP-12).
func (s *Service) Sync(ctx context.Context) (*SyncReport, error) {
	loc, err := s.resolve()
	if err != nil {
		return nil, err
	}
	if err := loc.GuardSpecsWrite(opSync); err != nil {
		return nil, err
	}
	indexer, patternWarning := s.indexerFor(loc)
	report, err := indexer.SyncIndexForProject(projectNameOf(loc), specsRootOf(loc), indexPathOf(loc))
	if err != nil {
		return nil, err
	}
	if patternWarning != "" {
		report.Warnings = append(report.Warnings, patternWarning)
	}
	if warning, ok := loc.LegacySpecsDivergence(); ok {
		report.Warnings = append(report.Warnings, warning)
	}
	return report, nil
}

// ColdStart ejecuta la adopción orgánica para un cambio y sincroniza el catálogo.
// Con un `baseline:` residual se rechaza antes de escribir nada con un
// *workspace.BaselineRetiredError (RF-TOP-12).
func (s *Service) ColdStart(changeName, domain string) (*LivingSpecEntry, error) {
	loc, err := s.resolve()
	if err != nil {
		return nil, err
	}
	if err := loc.GuardSpecsWrite(opColdStart); err != nil {
		return nil, err
	}
	entry, err := s.synthesizer.ColdStart(loc.WorkspaceRoot, changeName, domain)
	if err != nil {
		return nil, err
	}

	// Sincronizar catálogo maestro
	_, _ = s.Sync(context.Background())

	return entry, nil
}

// GetSpecsRoot retorna el directorio donde residen las especificaciones vivas.
func (s *Service) GetSpecsRoot() (string, error) {
	loc, err := s.resolve()
	if err != nil {
		return "", err
	}
	return specsRootOf(loc), nil
}

// GetIndexPath retorna la ruta al archivo INDEX.md.
func (s *Service) GetIndexPath() (string, error) {
	loc, err := s.resolve()
	if err != nil {
		return "", err
	}
	return indexPathOf(loc), nil
}
