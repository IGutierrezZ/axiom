package integrations

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// TrackerNotesFile es el diario local de avisos de un cambio `continuous`: rechazos, avisos y
	// éxitos de los verbos de tracker, que el archivo lee para componer el seguimiento final. Es un
	// fichero con punto en la raíz del cambio: estado local que nunca se publica
	// (docpaths.LocalState) y cuyo nombre no casa con .integrations-*.yaml.
	TrackerNotesFile = ".tracker-notes.yaml"
	// TrackerNotesSchemaV1 es su esquema.
	TrackerNotesSchemaV1 = "axiom.integrations-tracker-notes/v1"

	notesTempPattern = ".tracker-notes-*.yaml"
)

// TrackerNote es un suceso de un verbo de tracker. Code `done` marca un verbo terminado con éxito:
// resuelve los sucesos anteriores del mismo verbo y rol.
type TrackerNote struct {
	At           time.Time `yaml:"at"`
	Verb         string    `yaml:"verb"`
	Role         string    `yaml:"role,omitempty"`
	Code         string    `yaml:"code"`
	Message      string    `yaml:"message,omitempty"`
	Continuation string    `yaml:"continuation,omitempty"`
}

type notesDoc struct {
	Schema string        `yaml:"schema"`
	Change string        `yaml:"change"`
	Notes  []TrackerNote `yaml:"notes,omitempty"`
}

func readNotes(f Folder) (notesDoc, error) {
	path := f.path(TrackerNotesFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return notesDoc{Schema: TrackerNotesSchemaV1, Change: f.Change()}, nil
		}
		return notesDoc{}, fmt.Errorf("leer %s en %q: %w", TrackerNotesFile, path, err)
	}
	var doc notesDoc
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return notesDoc{}, fmt.Errorf("%s inválido en %q: el fichero está vacío", TrackerNotesFile, path)
		}
		return notesDoc{}, fmt.Errorf("%s inválido en %q: %w", TrackerNotesFile, path, err)
	}
	if doc.Schema != TrackerNotesSchemaV1 {
		return notesDoc{}, fmt.Errorf("%s inválido en %q: esquema %q no admitido; admitido: %s", TrackerNotesFile, path, doc.Schema, TrackerNotesSchemaV1)
	}
	if doc.Change != f.Change() {
		return notesDoc{}, fmt.Errorf("%s inválido en %q: el campo change es %q y la carpeta es la del cambio %q", TrackerNotesFile, path, doc.Change, f.Change())
	}
	return doc, nil
}

// AppendTrackerNote añade un suceso al final del diario bajo el cerrojo del cambio. Solo añade:
// nunca reordena ni retira nada. note.At vacío se completa con la hora actual; la hora siempre se
// guarda en UTC.
func AppendTrackerNote(f Folder, note TrackerNote) error {
	if note.Verb == "" || note.Code == "" {
		return errors.New("el suceso del diario exige verb y code")
	}
	if note.At.IsZero() {
		note.At = time.Now()
	}
	note.At = note.At.UTC()

	release, err := f.acquire()
	if err != nil {
		return err
	}
	defer release()
	doc, err := readNotes(f)
	if err != nil {
		return err
	}
	doc.Notes = append(doc.Notes, note)
	doc.Schema = TrackerNotesSchemaV1
	doc.Change = f.Change()
	payload, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("serializar %s: %w", TrackerNotesFile, err)
	}
	return f.writeAtomic(TrackerNotesFile, notesTempPattern, payload)
}
