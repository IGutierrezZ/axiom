package sddstatus

import (
	"path/filepath"
	"sort"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RequirementItem es un requisito que la delta del cambio crea o toca (REQ-47.9).
type RequirementItem = openspecdoc.DeclaredRequirement

// RequirementsProjection es el campo aditivo `requirements` de `axiom sdd status --json`:
// `modifies` con los MODIFIED, REMOVED y RENAMED (nombre antiguo) y `creates` con los ADDED,
// ordenados por capacidad y, dentro de ella, por orden de aparición en la delta.
type RequirementsProjection struct {
	Modifies []RequirementItem `json:"modifies"`
	Creates  []RequirementItem `json:"creates"`
}

// decorateRequirements proyecta lo que declaran las deltas del cambio (REQ-47.9, D9). Es
// aditivo y no toca nextRecommended, dependencies ni motivos. El campo se omite, sin dejar
// rastro en el JSON, cuando el cambio está archivado, el almacén no guarda ficheros
// (`engram` o `none`), el patrón de identificadores es inaccesible o las deltas no declaran
// ningún requisito. No lee la canónica: un nombre sin identificador no lleva `id`.
func decorateRequirements(status *Status, location workspace.Location) {
	if status.ChangeName == nil || status.Archived != nil || len(status.ArtifactPaths.Specs) == 0 {
		return
	}
	switch effectiveArtifactStore(status, location) {
	case ArtifactStoreEngram, ArtifactStoreNone:
		return
	}
	pattern, err := location.Declarations.IDPattern()
	if err != nil {
		return
	}
	paths := append([]string(nil), status.ArtifactPaths.Specs...)
	sort.Strings(paths)
	capabilityOf := func(path string) string {
		if status.ChangeRoot != nil && filepath.Clean(filepath.Dir(path)) == filepath.Clean(*status.ChangeRoot) {
			return ""
		}
		return filepath.Base(filepath.Dir(path))
	}
	var projection RequirementsProjection
	for _, path := range paths {
		delta, _ := openspecdoc.ParseChangeSpec(readText(path))
		declared := openspecdoc.Declared(capabilityOf(path), delta, pattern)
		projection.Modifies = append(projection.Modifies, declared.Modifies...)
		projection.Creates = append(projection.Creates, declared.Creates...)
	}
	if len(projection.Modifies) == 0 && len(projection.Creates) == 0 {
		return
	}
	byCapability := func(items []RequirementItem) func(i, j int) bool {
		return func(i, j int) bool { return items[i].Capability < items[j].Capability }
	}
	sort.SliceStable(projection.Modifies, byCapability(projection.Modifies))
	sort.SliceStable(projection.Creates, byCapability(projection.Creates))
	if projection.Modifies == nil {
		projection.Modifies = []RequirementItem{}
	}
	if projection.Creates == nil {
		projection.Creates = []RequirementItem{}
	}
	status.Requirements = &projection
}
