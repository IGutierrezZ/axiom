package changestart

import (
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

const axiomTagPrefix = "axiom:"

// axiomTagChange interpreta una etiqueta del work item. Solo cuenta como etiqueta de Axiom si empieza
// por axiom: (sin distinguir mayúsculas) y el resto supera ValidChangeName; si no, es una etiqueta
// ajena que nunca se usa en rutas, mensajes ni continuaciones (D-11, D-28).
func axiomTagChange(tag string) (string, bool) {
	t := strings.TrimSpace(tag)
	if len(t) < len(axiomTagPrefix) || !strings.EqualFold(t[:len(axiomTagPrefix)], axiomTagPrefix) {
		return "", false
	}
	other := strings.TrimSpace(t[len(axiomTagPrefix):])
	if ValidChangeName(other) != nil {
		return "", false
	}
	return other, true
}

// blockingTags devuelve, con la grafía axiom:<otro>, las etiquetas de tags que impiden al cambio own
// añadir la suya: las de Axiom con un nombre válido, distintas de own (sin distinguir mayúsculas) y
// cuyo cambio no está cerrado localmente (D-11, prov:74:15). Las ajenas, las de nombre inválido y las
// de cambios archivados o terminados se toleran.
func (l *localState) blockingTags(tags []string, own string) []string {
	var out []string
	for _, tag := range tags {
		other, ok := axiomTagChange(tag)
		if !ok || strings.EqualFold(other, own) || l.closedLocally(other) {
			continue
		}
		out = append(out, axiomTagPrefix+other)
	}
	return out
}

// selfAdoption es la adopción de este mismo work item por el cambio del nombre derivado.
type selfAdoption struct {
	Lane     Lane
	Context  string
	Backlink string
}

// linkScan es el resultado del escaneo de vínculos sin rechazo.
type linkScan struct {
	Self     *selfAdoption
	Warnings []string
}

// scanLinks aplica D-11 sin red: primero las etiquetas del WI, que mandan sobre el registro local, y
// después los vínculos de los cambios abiertos del registro. Rechaza con work_item_linked o devuelve
// la adopción propia, si la hay.
func (l *localState) scanLinks(item integrations.ExternalItem, plugin, id, name string) (linkScan, *Refusal) {
	var scan linkScan
	for _, tag := range item.Tags {
		other, ok := axiomTagChange(tag)
		if !ok || strings.EqualFold(other, name) || l.closedLocally(other) {
			continue
		}
		known := "false"
		if len(l.recordsOf(other)) > 0 {
			known = "true"
		}
		return scan, refuse(CodeWorkItemLinked, nil, "tag", axiomTagPrefix+other, "change", other, "local", known)
	}
	if l.warning != "" {
		scan.Warnings = append(scan.Warnings, l.warning)
	}
	unreadable := false
	for _, r := range l.records {
		for _, base := range checkouts(r) {
			root, ctxFile := l.changeLocations(base, r)
			match, doc, err := adoptsItem(r, root, ctxFile, plugin, id)
			if err != nil {
				unreadable = true
				continue
			}
			if !match {
				continue
			}
			if !strings.EqualFold(r.Change, name) {
				return scan, refuse(CodeWorkItemLinked, nil, "change", r.Change, "local", "true")
			}
			if scan.Self == nil {
				scan.Self = &selfAdoption{Lane: Lane(r.Lane), Context: filepath.Dir(ctxFile), Backlink: doc.Backlink}
			}
		}
	}
	if unreadable {
		scan.Warnings = append(scan.Warnings, "no se pudo leer el vínculo local de algún cambio abierto: el escaneo local fue incompleto")
	}
	return scan, nil
}

// changeLocations devuelve la carpeta del cambio y su context/work-item.json en un checkout (D-11):
// SDD <planning>/changes/<x>/context/ y ODD <ws>/odd/context/<x>/.
func (l *localState) changeLocations(base string, r branch.Record) (changeRoot, contextFile string) {
	if r.Lane == branch.LaneODD {
		dir := filepath.Join(base, l.env.workspaceRel(), "odd", "context", r.Change)
		return dir, filepath.Join(dir, "work-item.json")
	}
	dir := filepath.Join(base, l.env.planningRel(), "changes", r.Change)
	return dir, filepath.Join(dir, "context", "work-item.json")
}

// adoptsItem dice si el cambio ya adoptó el work item: en SDD por su vínculo kind work_item en
// integrations.yaml o por su context/work-item.json; en ODD, solo por este último.
func adoptsItem(r branch.Record, changeRoot, contextFile, plugin, id string) (bool, ContextDoc, error) {
	var doc ContextDoc
	if d, err := readContextDoc(contextFile); err == nil {
		doc = d
		if d.Plugin == plugin && d.ID == id {
			return true, doc, nil
		}
	} else if !isNotExist(err) {
		return false, doc, err
	}
	if r.Lane == branch.LaneODD {
		return false, doc, nil
	}
	folder := integrations.Folder{WorkspaceRoot: filepath.Dir(filepath.Dir(changeRoot)), Root: changeRoot}
	links, err := integrations.WorkItemLinks(folder)
	if err != nil {
		return false, doc, err
	}
	for _, link := range links {
		if link.Plugin == plugin && link.ID == id {
			return true, doc, nil
		}
	}
	return false, doc, nil
}
