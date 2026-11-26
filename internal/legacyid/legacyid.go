// Package legacyid lee el mapa versionado de identificadores heredados (`openspec/id-map.yaml`,
// esquema `axiom.id-map/v1`) y lo ofrece como resolutor de equivalentes de `axiom trace`
// (C-03-e, RF-TRC-02 y RF-TRC-04). El mapa asocia cada requisito vigente en la migración a su
// identificador nuevo y, si lo tenía, a su identificador antiguo `REQ-n.m`.
//
// Es un paquete de dominio sin E/S propia: lee a través de io/fs.FS, no usa git y no importa
// reqtrace; Map implementa de forma estructural reqtrace.LegacyResolver.
package legacyid

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const (
	// FileName es el nombre del mapa, relativo al planning home.
	FileName = "id-map.yaml"
	// Schema es la versión del esquema que este paquete sabe leer.
	Schema = "axiom.id-map/v1"
)

// Códigos estables de MapError (RF-TRC-17).
const (
	CodeUnreadable    = "id_map_unreadable"
	CodeMalformed     = "id_map_malformed"
	CodeSchemaUnknown = "id_map_schema_unknown"
	CodeInvariant     = "id_map_invariant"
)

// MapError es el error tipado de un mapa que existe pero no es válido.
type MapError struct {
	// Path es la ruta mostrada del fichero.
	Path string
	// Code es uno de los códigos estables Code*.
	Code string
	// Detail es el motivo concreto (la entrada y el invariante incumplido).
	Detail string
	// Err es la causa de bajo nivel, si la hay.
	Err error
}

func (e *MapError) Error() string {
	return fmt.Sprintf("el mapa de identificadores heredados %s no es válido (%s): %s; corrige el mapa o restáuralo desde git con `git checkout -- openspec/id-map.yaml`",
		e.Path, e.Code, e.Detail)
}

// Unwrap devuelve la causa de bajo nivel.
func (e *MapError) Unwrap() error { return e.Err }

// rawMap es el documento YAML tal como se escribe en disco.
type rawMap struct {
	Schema  string     `yaml:"schema"`
	Entries []rawEntry `yaml:"entries"`
}

type rawEntry struct {
	Capability string   `yaml:"capability"`
	Old        []string `yaml:"old,flow"`
	Title      string   `yaml:"title"`
	New        string   `yaml:"new"`
}

// capOld es la clave natural de una entrada: el par (capacidad, ID antiguo).
type capOld struct{ capability, old string }

// capTitle es la capacidad y el título de una entrada sin ID antiguo.
type capTitle struct{ capability, title string }

// Map es el mapa cargado e indexado.
type Map struct {
	byPair  map[capOld]string
	oldCaps map[string][]string // ID antiguo -> capacidades, ordenadas
	newOlds map[string][]string // ID nuevo -> IDs antiguos, ordenados
	titles  map[string]capTitle // ID nuevo sin ID antiguo -> (capacidad, título)
}

// rfPattern es el patrón con el que Axiom numera los IDs nuevos del mapa.
var rfPattern = func() reqid.Pattern {
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		panic(err)
	}
	return p
}()

// Load lee name de fsys. Un fichero ausente da (nil, nil). Cualquier otro fallo es un
// *MapError. Los IDs nuevos deben ser válidos bajo el patrón p de la consulta o bajo
// `RF-<DOM>-<NN>`: nunca se exige la forma `REQ-<n>.<m>` por defecto.
func Load(fsys fs.FS, name string, p reqid.Pattern) (*Map, error) {
	data, err := fs.ReadFile(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &MapError{Path: name, Code: CodeUnreadable, Detail: fmt.Sprintf("no se pudo leer el fichero: %v", err), Err: err}
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, &MapError{Path: name, Code: CodeMalformed, Detail: "el fichero está vacío"}
	}

	var head struct {
		Schema string `yaml:"schema"`
	}
	if err := yaml.Unmarshal(data, &head); err != nil {
		return nil, &MapError{Path: name, Code: CodeMalformed, Detail: fmt.Sprintf("YAML inválido: %v", err), Err: err}
	}
	if head.Schema != Schema {
		return nil, &MapError{Path: name, Code: CodeSchemaUnknown, Detail: fmt.Sprintf("el esquema %q no se conoce; se esperaba %q", head.Schema, Schema)}
	}

	var raw rawMap
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return nil, &MapError{Path: name, Code: CodeMalformed, Detail: fmt.Sprintf("el contenido no sigue el esquema: %v", err), Err: err}
	}
	return index(name, raw, p)
}

func invariant(name, format string, args ...any) error {
	return &MapError{Path: name, Code: CodeInvariant, Detail: fmt.Sprintf(format, args...)}
}

// index valida los invariantes de RF-TRC-16 y construye los índices.
func index(name string, raw rawMap, p reqid.Pattern) (*Map, error) {
	if len(raw.Entries) == 0 {
		return nil, invariant(name, "el mapa no tiene entradas")
	}
	m := &Map{byPair: map[capOld]string{}, oldCaps: map[string][]string{}, newOlds: map[string][]string{}, titles: map[string]capTitle{}}
	for i, e := range raw.Entries {
		at := fmt.Sprintf("entrada %d", i+1)
		if strings.TrimSpace(e.Capability) == "" {
			return nil, invariant(name, "%s: la capacidad está vacía", at)
		}
		if strings.TrimSpace(e.New) == "" {
			return nil, invariant(name, "%s (%s): falta el ID nuevo", at, e.Capability)
		}
		if !p.Valid(e.New) && !rfPattern.Valid(e.New) {
			return nil, invariant(name, "%s (%s): el ID nuevo %q no es válido bajo el patrón %s ni bajo %s", at, e.Capability, e.New, p.Template(), rfPattern.Template())
		}
		if _, dup := m.newOlds[e.New]; dup {
			return nil, invariant(name, "%s (%s): el ID nuevo %q está repetido", at, e.Capability, e.New)
		}
		if len(e.Old) == 0 && strings.TrimSpace(e.Title) == "" {
			return nil, invariant(name, "%s (%s, %s): no lleva ni old ni title", at, e.Capability, e.New)
		}
		olds := append([]string(nil), e.Old...)
		sort.Strings(olds)
		m.newOlds[e.New] = olds
		if len(e.Old) == 0 {
			m.titles[e.New] = capTitle{e.Capability, strings.TrimSpace(e.Title)}
		}
		for _, old := range e.Old {
			if strings.TrimSpace(old) == "" || strings.TrimSpace(old) != old {
				return nil, invariant(name, "%s (%s): el ID antiguo %q no es válido", at, e.Capability, old)
			}
			key := capOld{e.Capability, old}
			if _, dup := m.byPair[key]; dup {
				return nil, invariant(name, "%s: el ID antiguo %s está repetido en la capacidad %s", at, old, e.Capability)
			}
			m.byPair[key] = e.New
			m.oldCaps[old] = append(m.oldCaps[old], e.Capability)
		}
	}
	for old := range m.oldCaps {
		sort.Strings(m.oldCaps[old])
	}
	return m, nil
}

// Aliases devuelve los equivalentes de id: de un ID antiguo, todos sus IDs nuevos (uno por
// capacidad); de un ID nuevo, sus IDs antiguos. Ordenados y sin repetidos.
func (m *Map) Aliases(id string) []string {
	set := map[string]bool{}
	for _, c := range m.oldCaps[id] {
		set[m.byPair[capOld{c, id}]] = true
	}
	for _, old := range m.newOlds[id] {
		set[old] = true
	}
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// Legacy dice si id es una clave antigua del mapa en alguna capacidad. La comparación es
// exacta.
func (m *Map) Legacy(id string) bool { return len(m.oldCaps[id]) > 0 }

// Known dice si id es una clave antigua o un ID nuevo del mapa.
func (m *Map) Known(id string) bool {
	if m.Legacy(id) {
		return true
	}
	_, ok := m.newOlds[id]
	return ok
}

// InCapability devuelve el ID nuevo del par (capacidad, ID antiguo), o "" si no existe.
func (m *Map) InCapability(capability, old string) string { return m.byPair[capOld{capability, old}] }

// Capabilities devuelve, ordenadas, las capacidades en las que old es una clave antigua.
func (m *Map) Capabilities(old string) []string { return append([]string(nil), m.oldCaps[old]...) }

// Title devuelve la capacidad y el título de la entrada sin ID antiguo cuyo ID nuevo es id, o
// dos cadenas vacías si id no es una de ellas (RF-TRC-04, regla 5).
func (m *Map) Title(id string) (capability, title string) {
	t := m.titles[id]
	return t.capability, t.title
}
