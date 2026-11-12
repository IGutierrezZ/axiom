// Package adoption inventaría los orígenes de requisitos previos de un proyecto y
// verifica el cambio SDD que los migra a OpenSpec (REQ-30.9 a REQ-30.14). Es de solo
// lectura salvo la escritura del inventario que hace la CLI con `--change`.
package adoption

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// InventoryFileName es el inventario que `adopt scan --change` escribe en la carpeta
	// del cambio.
	InventoryFileName = "adoption-inventory.yaml"
	// DiscardsFileName es el documento del cambio cuya sección `## Descartado` lista las
	// unidades que no se migran.
	DiscardsFileName = "adoption.md"
	// InventorySchema identifica el formato del inventario.
	InventorySchema = "axiom.openspec-adoption-inventory/v1"

	// KindMarkdown es un origen con encabezados; KindFile, uno de fichero entero.
	KindMarkdown = "markdown"
	KindFile     = "file"

	// ModeMaintained y ModeReference son los modos de `technical_context`.
	ModeMaintained = "maintained"
	ModeReference  = "reference"
)

// Inventory es el inventario determinista de orígenes y unidades de un árbol.
type Inventory struct {
	Schema            string             `yaml:"schema" json:"schema"`
	Sources           []Source           `yaml:"sources" json:"sources"`
	ContextCandidates []ContextCandidate `yaml:"context_candidates" json:"context_candidates"`
	Warnings          []string           `yaml:"warnings" json:"warnings"`
}

// Source es un fichero de requisitos previos. Path va relativo al workspaceRoot y con `/`.
type Source struct {
	Path   string `yaml:"path" json:"path"`
	SHA256 string `yaml:"sha256" json:"sha256"`
	Kind   string `yaml:"kind" json:"kind"` // markdown|file
	Units  []Unit `yaml:"units" json:"units"`
}

// Unit es una parte direccionable de un origen: un encabezado (con ancla) o el fichero
// entero (ancla vacía).
type Unit struct {
	Anchor     string   `yaml:"anchor" json:"anchor"`
	Title      string   `yaml:"title" json:"title"`
	Line       int      `yaml:"line" json:"line"`
	ForeignIDs []string `yaml:"foreign_ids,omitempty" json:"foreign_ids,omitempty"`
}

// ContextCandidate es una ruta que el escaneo propone declarar en `technical_context`.
type ContextCandidate struct {
	Path   string `yaml:"path" json:"path"`
	Mode   string `yaml:"mode" json:"mode"` // maintained|reference
	Reason string `yaml:"reason" json:"reason"`
}

// Summary es la vista del hub: orígenes, recuento de unidades y candidatos.
type Summary struct {
	SourceCount       int                `json:"source_count"`
	UnitCount         int                `json:"unit_count"`
	Sources           []SourceSummary    `json:"sources"`
	ContextCandidates []ContextCandidate `json:"context_candidates"`
	Warnings          []string           `json:"warnings"`
}

// SourceSummary es un origen con su recuento de unidades.
type SourceSummary struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Units int    `json:"units"`
}

// Summarize resume un inventario para el hub. Las listas nunca son nulas.
func Summarize(inv Inventory) Summary {
	sum := Summary{
		Sources:           []SourceSummary{},
		ContextCandidates: append([]ContextCandidate{}, inv.ContextCandidates...),
		Warnings:          append([]string{}, inv.Warnings...),
	}
	for _, s := range inv.Sources {
		sum.SourceCount++
		sum.UnitCount += len(s.Units)
		sum.Sources = append(sum.Sources, SourceSummary{Path: s.Path, Kind: s.Kind, Units: len(s.Units)})
	}
	return sum
}

// Empty dice si el resumen no propone ningún origen ni candidato.
func (s Summary) Empty() bool {
	return s.SourceCount == 0 && len(s.ContextCandidates) == 0
}

// Marshal serializa el inventario en YAML.
func (inv Inventory) Marshal() ([]byte, error) {
	data, err := yaml.Marshal(inv)
	if err != nil {
		return nil, fmt.Errorf("no se puede serializar el inventario de adopción: %w", err)
	}
	return data, nil
}

// ReadInventory lee el inventario de la carpeta de un cambio. El error envuelve
// fs.ErrNotExist si no existe.
func ReadInventory(changeRoot string) (Inventory, error) {
	path := filepath.Join(changeRoot, InventoryFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Inventory{}, err
	}
	var inv Inventory
	if err := yaml.Unmarshal(data, &inv); err != nil {
		return Inventory{}, fmt.Errorf("el inventario %s no es un YAML válido: %w", path, err)
	}
	if inv.Schema != InventorySchema {
		return Inventory{}, fmt.Errorf("el inventario %s declara el esquema %q y se esperaba %q", path, inv.Schema, InventorySchema)
	}
	return inv, nil
}
