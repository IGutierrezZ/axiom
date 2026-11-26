package integrationstest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"gopkg.in/yaml.v3"
)

// TrackerNotes lee los sucesos del diario `.tracker-notes.yaml` de una carpeta de cambio en el orden
// en que se añadieron, sin tomar el cerrojo. Sin fichero, ninguno. Es un ayudante de pruebas: la
// producción solo añade al diario y el orquestador lo lee como fichero, así que el lector no vive en
// el paquete de producción (dejaría una función exportada que nada alcanza desde cmd/axiom).
func TrackerNotes(f integrations.Folder) ([]integrations.TrackerNote, error) {
	data, err := os.ReadFile(filepath.Join(f.Root, integrations.TrackerNotesFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("leer %s: %w", integrations.TrackerNotesFile, err)
	}
	var doc struct {
		Notes []integrations.TrackerNote `yaml:"notes"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s inválido: %w", integrations.TrackerNotesFile, err)
	}
	return doc.Notes, nil
}
