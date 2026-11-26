// Package trackersync reúne la lógica pura de los verbos de sincronización con el tracker
// (eventos-ado, D4): el modelo de la spec de un cambio, el reloj y las estimaciones. No
// hace red; solo lee los artefactos del cambio.
package trackersync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/specsummary"
)

// CodeProvisionalIDsUnresolved es el código de un cambio archivado cuyas deltas conservan
// un provisional que `ids.yaml` no traduce.
const CodeProvisionalIDsUnresolved = "provisional_ids_unresolved"

// SpecModelError es el error del modelo de la spec: Code es el código estable y Reason el
// motivo, con su continuación.
type SpecModelError struct {
	Code   string
	Reason string
}

func (e *SpecModelError) Error() string { return e.Code + ": " + e.Reason }

// SpecModelOptions configura SpecModel. Pattern es el patrón de identificadores del
// proyecto (el valor cero es el predeterminado). Archived indica que la carpeta es la de un
// cambio archivado: sus IDs definitivos salen de las deltas reescritas y un `prov:`
// residual se traduce con `ids.yaml`.
type SpecModelOptions struct {
	Pattern  reqid.Pattern
	Archived bool
}

// SpecModelResult es el modelo de la spec y los IDs de los requisitos que el cambio crea,
// modifica o renombra (los REMOVED no llevan ID), sin repetidos y en el orden del resumen.
type SpecModelResult struct {
	Model specsummary.Model
	IDs   []string
}

// SpecModel recorre las deltas de `<changeRoot>/specs/<capacidad>/spec.md` con
// openspecdoc.ParseChangeSpec y devuelve el modelo del renderizador: los requisitos de
// todas las secciones reordenados por su línea (el orden del fichero) con su operación. Solo
// lee: ni la spec viva ni la traza se modifican. Sin carpeta `specs/` devuelve el resultado
// vacío.
func SpecModel(changeRoot string, opts SpecModelOptions) (SpecModelResult, error) {
	specsDir := filepath.Join(changeRoot, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return SpecModelResult{}, nil
		}
		return SpecModelResult{}, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos de la carpeta", specsDir, err)
	}

	subst, known, err := traceSubstitution(changeRoot, opts.Archived)
	if err != nil {
		return SpecModelResult{}, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var out SpecModelResult
	seen := map[string]bool{}
	for _, name := range names {
		path := filepath.Join(specsDir, name, "spec.md")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return SpecModelResult{}, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero", path, err)
		}
		delta, _ := openspecdoc.ParseChangeSpec(string(data))
		if delta.Empty() {
			continue
		}
		reqs, err := orderedRequirements(delta, subst, known, opts)
		if err != nil {
			return SpecModelResult{}, fmt.Errorf("%s: %w", path, err)
		}
		out.Model.Capabilities = append(out.Model.Capabilities, specsummary.Capability{
			Name:         name,
			Requirements: modelRequirements(reqs),
		})
		for _, r := range reqs {
			if r.op == specsummary.OpRemoved || r.id == "" || seen[r.id] {
				continue
			}
			seen[r.id] = true
			out.IDs = append(out.IDs, r.id)
		}
	}
	return out, nil
}

// traceSubstitution lee `ids.yaml` solo en un cambio archivado: el mapa de sustitución de
// provisional a definitivo y el conjunto de provisionales con entrada.
func traceSubstitution(changeRoot string, archived bool) (reqid.IDMap, map[string]bool, error) {
	if !archived {
		return reqid.IDMap{}, nil, nil
	}
	trace, err := livespec.ReadTraceFile(filepath.Join(changeRoot, "ids.yaml"))
	if err != nil {
		return reqid.IDMap{}, nil, err
	}
	pairs := make([]reqid.Assignment, 0, len(trace.Assignments))
	known := map[string]bool{}
	for _, a := range trace.Assignments {
		if a.Provisional == "" || a.Definitive == "" {
			continue
		}
		pairs = append(pairs, reqid.Assignment{Provisional: a.Provisional, Definitive: a.Definitive})
		known[a.Provisional] = true
	}
	return reqid.NewIDMap(pairs), known, nil
}

// orderedReq es un requisito ya resuelto, con su línea para el orden.
type orderedReq struct {
	line        int
	op          specsummary.Operation
	title       string
	renamedFrom string
	paragraph   string
	id          string
}

// orderedRequirements junta las cuatro secciones y las ordena por línea del fichero.
func orderedRequirements(d openspecdoc.Delta, subst reqid.IDMap, known map[string]bool, opts SpecModelOptions) ([]orderedReq, error) {
	var reqs []orderedReq
	add := func(rs []openspecdoc.Requirement, op specsummary.Operation) error {
		for _, r := range rs {
			title, from := r.Name, ""
			if op == specsummary.OpRenamed {
				if oldName, newName, ok := openspecdoc.SplitRename(r.Name); ok {
					title, from = newName, oldName
				}
			}
			if opts.Archived {
				for _, text := range []string{title, from} {
					if token, ok := reqid.ProvisionalInTitle(text); ok && !known[token] {
						if _, err := opts.Pattern.ParseProvisional(token); err == nil {
							return &SpecModelError{Code: CodeProvisionalIDsUnresolved,
								Reason: fmt.Sprintf("el provisional %s sigue en la delta archivada y `ids.yaml` no lo traduce; restaura `ids.yaml` con git o reintegra el cambio con `axiom sdd archive-integrate --check`", token)}
						}
					}
				}
				title, from = subst.Substitute(title), subst.Substitute(from)
			}
			reqs = append(reqs, orderedReq{
				line: r.Line, op: op, title: title, renamedFrom: from,
				paragraph: specsummary.FirstParagraph(r.Body),
				id:        idOfTitle(title, from, opts.Pattern),
			})
		}
		return nil
	}
	for _, s := range []struct {
		rs []openspecdoc.Requirement
		op specsummary.Operation
	}{
		{d.Added, specsummary.OpAdded}, {d.Modified, specsummary.OpModified},
		{d.Removed, specsummary.OpRemoved}, {d.Renamed, specsummary.OpRenamed},
	} {
		if err := add(s.rs, s.op); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(reqs, func(i, j int) bool { return reqs[i].line < reqs[j].line })
	return reqs, nil
}

// idOfTitle devuelve el identificador del título: el provisional que el patrón admite o el
// último identificador válido; en un RENAMED sin ID en el nombre nuevo, el del anterior.
func idOfTitle(title, from string, p reqid.Pattern) string {
	for _, text := range []string{title, from} {
		if text == "" {
			continue
		}
		if token, ok := reqid.ProvisionalInTitle(text); ok {
			if _, err := p.ParseProvisional(token); err == nil {
				return token
			}
			continue
		}
		if id := p.InTitle(text); id != "" {
			return id
		}
	}
	return ""
}

// modelRequirements convierte los requisitos resueltos al modelo del renderizador.
func modelRequirements(reqs []orderedReq) []specsummary.Requirement {
	out := make([]specsummary.Requirement, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, specsummary.Requirement{
			Title: r.title, Paragraph: r.paragraph, Operation: r.op, RenamedFrom: r.renamedFrom,
		})
	}
	return out
}
