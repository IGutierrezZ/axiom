package reqtrace

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// AliasResolver es el punto de extensión de equivalentes: una capacidad futura (el mapa de
// IDs antiguos de C-03-e) añade equivalentes sin modificar la consulta.
type AliasResolver interface{ Aliases(id string) []string }

// LegacyResolver añade a AliasResolver la resolución por capacidad de los IDs heredados
// (RF-TRC-04): lo implementa el mapa versionado de legacyid.
type LegacyResolver interface {
	AliasResolver
	// Legacy dice si id es una clave antigua del mapa en alguna capacidad.
	Legacy(id string) bool
	// Known dice si id es una clave antigua o un ID nuevo del mapa.
	Known(id string) bool
	// InCapability devuelve el ID nuevo del par (capacidad, ID antiguo), o "".
	InCapability(capability, old string) string
	// Capabilities devuelve, ordenadas, las capacidades con old como clave antigua.
	Capabilities(old string) []string
}

// NoAliases es la implementación por defecto: no añade nada más allá de las trazas.
type NoAliases struct{}

// Aliases no devuelve ningún equivalente.
func (NoAliases) Aliases(string) []string { return nil }

// traceSet es la traza `axiom.requirement-ids/v1` de una fuente: Path identifica la carpeta
// archivada o el documento ODD (relativo a la raíz git) y es la clave del conjunto de
// coincidencia de esa fuente.
type traceSet struct {
	Path    string
	Entries []livespec.TraceEntry
}

// readSDDTrace lee el `ids.yaml` de una carpeta archivada. Un fichero ausente es una traza
// vacía; uno ilegible se ignora con una advertencia.
func readSDDTrace(files fs.FS, dir, gitPath string) (traceSet, []Warning) {
	set := traceSet{Path: gitPath}
	data, err := fs.ReadFile(files, joinPath(dir, "ids.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return set, nil
	}
	if err != nil {
		return set, []Warning{{Code: WarnUnreadable, Path: joinPath(gitPath, "ids.yaml"), Message: fmt.Sprintf("no se pudo leer la traza ids.yaml y se omite: %v; comprueba los permisos del fichero", err)}}
	}
	trace, err := livespec.ParseTrace(data, joinPath(gitPath, "ids.yaml"))
	if err != nil {
		return set, []Warning{traceInvalid(joinPath(gitPath, "ids.yaml"), err)}
	}
	set.Entries = trace.Assignments
	return set, nil
}

// readODDTrace lee la sección `## Identificadores asignados` de un documento ODD. Sin
// sección es una traza vacía; una sección ilegible se ignora con una advertencia.
func readODDTrace(doc, gitPath string) (traceSet, []Warning) {
	set := traceSet{Path: gitPath}
	trace, found, err := livespec.ExtractODDTrace(doc, gitPath)
	if err != nil {
		return set, []Warning{traceInvalid(gitPath, err)}
	}
	if found {
		set.Entries = trace.Assignments
	}
	return set, nil
}

func traceInvalid(path string, err error) Warning {
	return Warning{Code: WarnTraceInvalid, Path: path, Message: fmt.Sprintf("la traza de identificadores es ilegible y se ignora: %v", err)}
}

// equivalence es el resultado del cierre de equivalentes de un ID consultado.
type equivalence struct {
	query   Query
	closure map[string]bool
	// legacy es el resolutor de IDs heredados; nil si no hay mapa.
	legacy LegacyResolver
	// resolved son los IDs resueltos, en orden de bytes y sin repetidos.
	resolved []string
	// touched son, por traza, las entradas que tocan el cierre.
	touched map[string][]livespec.TraceEntry
}

// resolveEquivalents calcula el cierre de equivalentes de q sobre las trazas (decisión 3):
// semillas, cierre hasta punto fijo con los Previous y el AliasResolver, y los provisionales
// de las entradas que lo tocan. Un provisional con varios definitivos distintos los resuelve
// todos y emite provisional_ambiguous.
func resolveEquivalents(q Query, p reqid.Pattern, traces []traceSet, aliases AliasResolver) (equivalence, []Warning) {
	return resolveEquivalentsLegacy(q, p, traces, aliases, nil)
}

// translateLegacyTraces sustituye, en las entradas con capacidad, los IDs heredados por el ID
// nuevo de (capacidad, ID antiguo): así el cierre no arrastra los IDs nuevos de otras
// capacidades que usaban el mismo ID antiguo.
func translateLegacyTraces(traces []traceSet, legacy LegacyResolver) []traceSet {
	translate := func(capability, id string) string {
		if capability == "" || !legacy.Legacy(id) {
			return id
		}
		if n := legacy.InCapability(capability, id); n != "" {
			return n
		}
		return id
	}
	out := make([]traceSet, len(traces))
	for i, t := range traces {
		out[i] = traceSet{Path: t.Path, Entries: make([]livespec.TraceEntry, len(t.Entries))}
		for j, e := range t.Entries {
			e.Definitive = translate(e.Capability, e.Definitive)
			if len(e.Previous) > 0 {
				prev := make([]string, len(e.Previous))
				for k, id := range e.Previous {
					prev[k] = translate(e.Capability, id)
				}
				e.Previous = prev
			}
			out[i].Entries[j] = e
		}
	}
	return out
}

// resolveEquivalentsLegacy es resolveEquivalents con el resolutor de IDs heredados (nil sin
// mapa). Con él, un ID antiguo consultado resuelve a todos sus IDs nuevos (legacy_ambiguous si
// son varios) y un ID nuevo a sus antiguos; un antiguo alcanzado desde un nuevo no vuelve a
// expandirse, para no arrastrar los IDs nuevos de otras capacidades.
func resolveEquivalentsLegacy(q Query, p reqid.Pattern, traces []traceSet, aliases AliasResolver, legacy LegacyResolver) (equivalence, []Warning) {
	if aliases == nil {
		if legacy != nil {
			aliases = legacy
		} else {
			aliases = NoAliases{}
		}
	}
	if legacy != nil {
		traces = translateLegacyTraces(traces, legacy)
	}
	closure := map[string]bool{}
	var queue []string
	add := func(id string) bool {
		if id == "" || closure[id] {
			return false
		}
		closure[id] = true
		queue = append(queue, id)
		return true
	}

	var warnings []Warning
	if !q.Provisional {
		add(q.ID)
		if legacy != nil && legacy.Legacy(q.ID) {
			if caps := legacy.Capabilities(q.ID); len(caps) > 1 {
				news := make([]string, len(caps))
				for i, c := range caps {
					news[i] = legacy.InCapability(c, q.ID)
				}
				warnings = append(warnings, Warning{
					Code: WarnLegacyAmbiguous,
					Message: fmt.Sprintf("el identificador heredado %s corresponde a %d identificadores nuevos (%s) en las capacidades %s; se resuelven todos: consulta por el identificador nuevo para acotar",
						q.ID, len(caps), strings.Join(news, ", "), strings.Join(caps, ", ")),
				})
			}
		}
	} else {
		definitives := map[string]bool{}
		sources := map[string]bool{}
		for _, t := range traces {
			for _, e := range t.Entries {
				if e.Provisional == q.ID && e.Definitive != "" {
					definitives[e.Definitive] = true
					sources[t.Path] = true
				}
			}
		}
		if len(definitives) == 0 {
			add(q.ID)
		}
		for _, d := range sortedKeys(definitives) {
			add(d)
		}
		if len(definitives) > 1 {
			warnings = append(warnings, Warning{
				Code: WarnProvisionalAmbiguous,
				Message: fmt.Sprintf("el provisional %s está asignado a %d identificadores distintos (%s) en %s; se resuelven todos: consulta por el definitivo para acotar",
					q.ID, len(definitives), strings.Join(sortedKeys(definitives), ", "), strings.Join(sortedKeys(sources), ", ")),
			})
		}
	}

	touches := func(e livespec.TraceEntry) bool {
		if closure[e.Definitive] {
			return true
		}
		for _, prev := range e.Previous {
			if closure[prev] {
				return true
			}
		}
		return false
	}
	for progressed := true; progressed; {
		progressed = false
		for _, t := range traces {
			for _, e := range t.Entries {
				if !touches(e) {
					continue
				}
				for _, id := range append([]string{e.Definitive}, e.Previous...) {
					progressed = add(id) || progressed
				}
			}
		}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			if legacy != nil && id != q.ID && legacy.Legacy(id) {
				continue
			}
			for _, alias := range aliases.Aliases(id) {
				if acceptableAlias(alias, p, legacy) {
					progressed = add(alias) || progressed
				}
			}
		}
	}

	eq := equivalence{query: q, closure: closure, legacy: legacy, touched: map[string][]livespec.TraceEntry{}}
	resolved := map[string]bool{q.ID: true}
	for id := range closure {
		resolved[id] = true
	}
	for _, t := range traces {
		for _, e := range t.Entries {
			if touches(e) {
				eq.touched[t.Path] = append(eq.touched[t.Path], e)
				resolved[e.Provisional] = true
			}
		}
	}
	delete(resolved, "")
	eq.resolved = sortedKeys(resolved)
	return eq, warnings
}

// acceptableAlias descarta los alias que no son un ID válido ni un provisional válido del
// patrón; un alias que proviene del mapa de IDs heredados se acepta siempre.
func acceptableAlias(id string, p reqid.Pattern, legacy LegacyResolver) bool {
	if p.Valid(id) || (legacy != nil && legacy.Known(id)) {
		return true
	}
	_, err := p.ParseProvisional(id)
	return err == nil
}

// matchSet es el conjunto de coincidencia de la fuente cuya traza es key: el cierre, el ID
// consultado y los provisionales de las entradas de esa misma fuente que tocan el cierre. El
// provisional de otra fuente no cuenta, para que un provisional repetido en otro cambio no
// traiga falsos positivos.
//
// Con mapa, los IDs heredados quedan fuera: se buscan aparte y acotados por capacidad
// (matcher), para que un `REQ-1.1` de otra capacidad no cuente como incremento de este ID.
func (e equivalence) matchSet(key string) map[string]bool {
	set := map[string]bool{}
	if !e.isLegacy(e.query.ID) {
		set[e.query.ID] = true
	}
	for id := range e.closure {
		if !e.isLegacy(id) {
			set[id] = true
		}
	}
	for _, entry := range e.touched[key] {
		if entry.Provisional != "" {
			set[entry.Provisional] = true
		}
	}
	return set
}

func (e equivalence) isLegacy(id string) bool { return e.legacy != nil && e.legacy.Legacy(id) }

// assignments devuelve las asignaciones de la traza key que tocan el cierre, en orden estable.
func (e equivalence) assignments(key string) []Assignment {
	var out []Assignment
	for _, entry := range e.touched[key] {
		a := Assignment{Provisional: entry.Provisional, Definitive: entry.Definitive, Capability: entry.Capability}
		if len(entry.Previous) > 0 {
			a.Previous = append([]string(nil), entry.Previous...)
			sort.Strings(a.Previous)
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provisional != out[j].Provisional {
			return out[i].Provisional < out[j].Provisional
		}
		if out[i].Definitive != out[j].Definitive {
			return out[i].Definitive < out[j].Definitive
		}
		return out[i].Capability < out[j].Capability
	})
	return out
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// joinPath une dos rutas relativas con `/`, sin barra doble y sin barra inicial.
func joinPath(base, elem string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return elem
	}
	return base + "/" + elem
}
