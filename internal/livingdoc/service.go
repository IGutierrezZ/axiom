package livingdoc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Service provee la interfaz unificada para operaciones de documentación viva.
//
// El servicio no guarda rutas: conserva solo el directorio de partida y resuelve
// el planning home con workspace.Locate en cada llamada pública (REQ-24.10,
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

// GetCatalog devuelve el catálogo completo de especificaciones vivas consolidadas.
// Lee solo <planning home>/specs; si existe la ubicación antigua divergente
// (Q14), la nombra en Warnings.
func (s *Service) GetCatalog(ctx context.Context) (*LivingCatalog, error) {
	loc, err := s.resolve()
	if err != nil {
		return nil, err
	}
	catalog, _, err := s.indexer.ScanSpecs(specsRootOf(loc))
	if err != nil {
		return catalog, err
	}
	catalog.ProjectName = projectNameOf(loc)
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

	entry, _ := s.indexer.ParseSpecContent(string(contentBytes), domain, specPath, modTime)
	return entry, string(contentBytes), nil
}

// Sync ejecuta el re-escaneo y regenera el archivo INDEX.md del planning home.
func (s *Service) Sync(ctx context.Context) (*SyncReport, error) {
	loc, err := s.resolve()
	if err != nil {
		return nil, err
	}
	report, err := s.indexer.SyncIndexForProject(projectNameOf(loc), specsRootOf(loc), indexPathOf(loc))
	if err != nil {
		return nil, err
	}
	if warning, ok := loc.LegacySpecsDivergence(); ok {
		report.Warnings = append(report.Warnings, warning)
	}
	return report, nil
}

// ColdStart ejecuta la adopción orgánica para un cambio y sincroniza el catálogo.
func (s *Service) ColdStart(changeName, domain string) (*LivingSpecEntry, error) {
	loc, err := s.resolve()
	if err != nil {
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
