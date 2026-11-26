package livespec

import (
	"fmt"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// oddDocRel es la ruta del documento ODD relativa a la raíz del workspace, con `/`, tal como
// la nombran los motivos.
func oddDocRel(feature string) string { return "odd/tasks/" + feature + ".md" }

// readCanonical lee la especificación canónica de una capacidad; existed es falso si no existe.
func readCanonical(loc workspace.Location, capability, retry string) (path, text string, existed bool, err error) {
	path = specPath(loc, capability)
	data, readErr := os.ReadFile(path)
	switch {
	case readErr == nil:
		return path, string(data), true, nil
	case os.IsNotExist(readErr):
		return path, "", false, nil
	default:
		return path, "", false, &CapabilityError{Capability: capability, Err: fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero y vuelve a ejecutar `%s`", path, readErr, retry)}
	}
}

// capabilityFenceLines son las líneas del documento que ocupa el contenido de las vallas
// `openspec` de las capacidades.
func capabilityFenceLines(doc FeatureDoc) map[int]bool {
	fenced := map[int]bool{}
	for _, c := range doc.Capabilities {
		for i := 0; i < strings.Count(c.Text, "\n"); i++ {
			fenced[c.TextLine+i] = true
		}
	}
	return fenced
}

// docCitations son los provisionales que el documento cita en sus líneas `Cubre:` fuera de
// vallas y fuera de la sección de traza, las mismas que reescribe la integración. Las vallas
// de las capacidades las valida el plan por su cuenta; el resto del documento no se valida,
// de modo que puede mostrar provisionales de ejemplo (REQ-47.4).
func docCitations(doc FeatureDoc, file string) ([]Citation, error) {
	lines, start, end, found, err := oddTraceSection(doc.text, file)
	if err != nil {
		return nil, err
	}
	excluded := capabilityFenceLines(doc)
	if found {
		for _, l := range lines[start:end] {
			excluded[l.n] = true
		}
	}
	var out []Citation
	for _, c := range cubreCitations(file, doc.text) {
		if !excluded[c.Line] {
			out = append(out, c)
		}
	}
	return out, nil
}

// cubreCitations son los provisionales de las líneas `Cubre:` de un texto, fuera de vallas.
func cubreCitations(file, text string) []Citation {
	var out []Citation
	for _, line := range isoevidence.ParseCubre(file, text) {
		for _, id := range line.IDs {
			for _, span := range reqid.ScanProvisional(id) {
				out = append(out, Citation{File: file, Line: line.Line, Token: span.Token})
			}
		}
	}
	return out
}

// rewriteODDDoc devuelve el documento con los identificadores asignados: sustituye el mapa
// en las vallas `openspec` de `## Spec viva` y en las líneas `Cubre:` fuera de vallas, y
// escribe la sección de traza; el resto de los bytes no cambia. Sin asignaciones devuelve el
// documento igual (REQ-30.2).
func rewriteODDDoc(doc FeatureDoc, plan IDPlan, pattern reqid.Pattern) (string, error) {
	text := doc.text
	if plan.Map.Count() > 0 {
		fenced := capabilityFenceLines(doc)
		var b strings.Builder
		for i, line := range strings.SplitAfter(text, "\n") {
			if fenced[i+1] {
				b.WriteString(plan.Map.Substitute(line))
			} else {
				b.WriteString(line)
			}
		}
		text = isoevidence.RewriteCubre(b.String(), plan.Map)
	}
	return SetODDTrace(text, plan.Trace(pattern))
}

// dropTracedAdded quita de lo pendiente el ADDED que la traza reconoce como ya integrado
// aunque su título haya cambiado: la entrada se conserva (`kept`) y la canónica ya tiene, con
// su definitivo, un requisito de cuerpo igual (REQ-30.3).
func dropTracedAdded(item *classified, entries []PlannedEntry, pattern reqid.Pattern) {
	var pending []openspecdoc.Requirement
	for _, r := range item.pending.Added {
		if !integratedUnderAnotherTitle(item, r, entries, pattern) {
			pending = append(pending, r)
		}
	}
	item.pending.Added = pending
}

func integratedUnderAnotherTitle(item *classified, r openspecdoc.Requirement, entries []PlannedEntry, pattern reqid.Pattern) bool {
	id := pattern.InTitle(r.Name)
	if id == "" {
		return false
	}
	for _, e := range entries {
		if e.State != StateKept || e.Capability != item.cap.Name || e.Definitive != id {
			continue
		}
		for _, c := range item.canonical {
			if pattern.InTitle(c.Name) == id &&
				openspecdoc.NormalizeBody(afterHeading(c.Body)) == openspecdoc.NormalizeBody(afterHeading(r.Body)) {
				return true
			}
		}
	}
	return false
}

// checkAddedWithoutID rechaza un ADDED pendiente sin identificador definitivo ni
// provisional (added_without_id): sugiere el provisional, cuyo número asigna la integración.
func checkAddedWithoutID(feature string, items []classified, pattern reqid.Pattern) error {
	for _, item := range items {
		for _, r := range item.pending.Added {
			if _, provisional := reqid.ProvisionalInTitle(r.Name); provisional || pattern.InTitle(r.Name) != "" {
				continue
			}
			return &DocError{Feature: feature, Problems: []Problem{{
				Code:    "added_without_id",
				Line:    item.cap.Line + r.Line,
				Subject: r.Name,
				Message: fmt.Sprintf("el requisito ADDED %q de la capacidad %s no lleva un identificador `(%s)` en el título; añade el siguiente número libre de la serie, p. ej. `### Requirement: %s (%s)`, o un provisional `(%s)` cuyo número asigna la integración%s", r.Name, item.cap.Name, pattern.Compact(), r.Name, pattern.Compact(), reqid.ProvisionalText("<serie>", "<ordinal>"), patternHint(pattern)),
			}}}
		}
	}
	return nil
}
