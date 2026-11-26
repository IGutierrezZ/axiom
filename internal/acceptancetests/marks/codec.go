package marks

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrNotSequence indica que el documento YAML no es una secuencia de nivel superior.
var ErrNotSequence = errors.New("pruebas-estado.yaml no es una secuencia YAML de marcas")

// Parse lee el fichero. Un contenido vacío (o solo espacios y comentarios) es un fichero sin
// entradas. YAML ilegible o que no es una secuencia devuelve error y invalida el fichero entero.
// Los elementos que son YAML válido pero tienen campos fuera de forma se devuelven en Invalid.
func Parse(content []byte) (File, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return File{}, fmt.Errorf("pruebas-estado.yaml ilegible: %w", err)
	}
	if doc.Kind == 0 {
		return File{}, nil
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.SequenceNode {
		return File{}, ErrNotSequence
	}
	var file File
	for i, node := range doc.Content[0].Content {
		var e Entry
		if node.Kind != yaml.MappingNode {
			file.Invalid = append(file.Invalid, Issue{Index: i, Reason: "el elemento no es un mapa"})
			continue
		}
		if err := node.Decode(&e); err != nil {
			file.Invalid = append(file.Invalid, Issue{Index: i, Reason: "campos con tipo incorrecto"})
			continue
		}
		if reason := entryProblem(e); reason != "" {
			file.Invalid = append(file.Invalid, Issue{Index: i, Reason: reason})
			continue
		}
		file.Entries = append(file.Entries, e)
	}
	return file, nil
}

// entryProblem devuelve el motivo por el que e está fuera de forma, o "".
func entryProblem(e Entry) string {
	switch {
	case e.Case == "":
		return "case vacío"
	case e.Environment == "":
		return "environment vacío"
	case !e.Result.Valid():
		return "resultado fuera de dominio"
	case e.By == "":
		return "by vacío"
	}
	if _, err := time.Parse(time.RFC3339Nano, e.At); err != nil {
		return "at no es una fecha válida"
	}
	if raw, err := hex.DecodeString(e.PlanDigest); err != nil || len(raw) != 32 || len(e.PlanDigest) != 64 {
		return "plan_digest no es hexadecimal de 64 caracteres"
	}
	return ""
}

func encode(entries []Entry) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	return yaml.Marshal(entries)
}

// concat devuelve base (con `\n` final garantizado si no está vacía) seguida de la
// codificación de entries. El contenido previo es prefijo exacto del resultado salvo ese `\n`.
func concat(base []byte, entries []Entry) ([]byte, error) {
	enc, err := encode(entries)
	if err != nil {
		return nil, err
	}
	out := append([]byte{}, base...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return append(out, enc...), nil
}

// Append añade e al final del contenido por concatenación: content (con `\n` final
// garantizado) seguido de la codificación de e. El contenido previo es prefijo exacto del
// resultado. Devuelve error si content no es un fichero de marcas legible, si e está fuera de
// forma o si el resultado no se lee como una entrada más.
func Append(content []byte, e Entry) ([]byte, error) {
	if reason := entryProblem(e); reason != "" {
		return nil, fmt.Errorf("marca fuera de forma: %s", reason)
	}
	before, err := Parse(content)
	if err != nil {
		return nil, err
	}
	out, err := concat(content, []Entry{e})
	if err != nil {
		return nil, err
	}
	after, err := Parse(out)
	if err != nil || len(after.Entries) != len(before.Entries)+1 || len(after.Invalid) != len(before.Invalid) {
		return nil, errors.New("pruebas-estado.yaml no admite añadir al final (¿secuencia en línea o vacía `[]`?)")
	}
	return out, nil
}

// Merge devuelve remote más las entradas de local que le faltan (multiconjunto local − remote),
// codificadas en el orden local. Nunca duplica lo que ya está en remote. Las entradas
// inválidas de local no se reproducen. Error si alguno de los dos contenidos es ilegible.
func Merge(local, remote []byte) ([]byte, error) {
	l, err := Parse(local)
	if err != nil {
		return nil, fmt.Errorf("local: %w", err)
	}
	r, err := Parse(remote)
	if err != nil {
		return nil, fmt.Errorf("remoto: %w", err)
	}
	missing := Pending(l.Entries, r.Entries)
	if len(missing) == 0 {
		return append([]byte{}, remote...), nil
	}
	out, err := concat(remote, missing)
	if err != nil {
		return nil, err
	}
	if merged, err := Parse(out); err != nil || len(merged.Entries) != len(r.Entries)+len(missing) {
		return nil, errors.New("el contenido remoto no admite añadir al final")
	}
	return out, nil
}

// Pending devuelve las entradas de local que no están en published: diferencia de
// multiconjuntos por los ocho campos, en el orden de local.
func Pending(local, published []Entry) []Entry {
	have := make(map[Entry]int, len(published))
	for _, e := range published {
		have[e]++
	}
	var out []Entry
	for _, e := range local {
		if have[e] > 0 {
			have[e]--
			continue
		}
		out = append(out, e)
	}
	return out
}
