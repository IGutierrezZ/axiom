package integrations

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// PendingTimeFile es el diario local de imputaciones pendientes de un cambio. Es un fichero con
	// punto en la raíz de la carpeta del cambio: estado local que nunca se publica
	// (docpaths.LocalState) y cuyo nombre no casa con .integrations-*.yaml.
	PendingTimeFile = ".time-pending.yaml"
	// PendingTimeSchemaV1 es su esquema.
	PendingTimeSchemaV1 = "axiom.integrations-time-pending/v1"

	pendingTempPattern = ".time-pending-*.yaml"
)

// PendingTime es la intención de imputar horas de un rol, escrita antes de tocar el tracker. Una
// intención que sobrevive (fallo local, resultado incierto, proceso interrumpido) la reconcilia el
// siguiente `time record` de ese rol releyendo la Task de ItemID (la de la intención, no la del
// vínculo vigente).
type PendingTime struct {
	Role              string  `yaml:"role"`
	ItemID            string  `yaml:"item_id"`
	Hours             float64 `yaml:"hours"`
	Minutes           int     `yaml:"minutes"`
	Actor             string  `yaml:"actor"`
	Date              string  `yaml:"date"` // AAAA-MM-DD
	Source            string  `yaml:"source"`
	BaseRev           int     `yaml:"base_rev"`
	ExpectedCompleted float64 `yaml:"expected_completed"`
	ExpectedRemaining float64 `yaml:"expected_remaining"`
}

// pendingDoc es el contenido de .time-pending.yaml: un elemento por rol.
type pendingDoc struct {
	Schema  string        `yaml:"schema"`
	Change  string        `yaml:"change"`
	Pending []PendingTime `yaml:"pending,omitempty"`
}

func readPending(f Folder) (pendingDoc, error) {
	path := f.path(PendingTimeFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return pendingDoc{Schema: PendingTimeSchemaV1, Change: f.Change()}, nil
		}
		return pendingDoc{}, fmt.Errorf("leer %s en %q: %w", PendingTimeFile, path, err)
	}
	var doc pendingDoc
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return pendingDoc{}, fmt.Errorf("%s inválido en %q: el fichero está vacío", PendingTimeFile, path)
		}
		return pendingDoc{}, fmt.Errorf("%s inválido en %q: %w", PendingTimeFile, path, err)
	}
	if doc.Schema != PendingTimeSchemaV1 {
		return pendingDoc{}, fmt.Errorf("%s inválido en %q: esquema %q no admitido; admitido: %s", PendingTimeFile, path, doc.Schema, PendingTimeSchemaV1)
	}
	if doc.Change != f.Change() {
		return pendingDoc{}, fmt.Errorf("%s inválido en %q: el campo change es %q y la carpeta es la del cambio %q", PendingTimeFile, path, doc.Change, f.Change())
	}
	return doc, nil
}

func validatePending(p PendingTime) error {
	if strings.TrimSpace(p.Role) == "" {
		return errors.New("la intención pendiente exige role")
	}
	if err := validateItemID("item_id", p.ItemID); err != nil {
		return err
	}
	if strings.TrimSpace(p.Source) == "" || strings.TrimSpace(p.Actor) == "" {
		return errors.New("la intención pendiente exige source y actor")
	}
	if p.Minutes <= 0 {
		return errors.New("minutes debe ser positivo")
	}
	return nil
}

// PutPendingTime escribe la intención de un rol bajo el cerrojo del cambio, sustituyendo la que ya
// hubiera de ese rol. Un fallo se devuelve a quien llama: `time record` no toca el tracker si no
// puede dejarla escrita.
func PutPendingTime(f Folder, p PendingTime) error {
	if err := validatePending(p); err != nil {
		return err
	}
	release, err := f.acquire()
	if err != nil {
		return err
	}
	defer release()
	doc, err := readPending(f)
	if err != nil {
		return err
	}
	replaced := false
	for i := range doc.Pending {
		if doc.Pending[i].Role == p.Role {
			doc.Pending[i] = p
			replaced = true
		}
	}
	if !replaced {
		doc.Pending = append(doc.Pending, p)
	}
	return writePending(f, doc)
}

// PendingTimeFor devuelve la intención pendiente del rol, si la hay. Lectura sin cerrojo: la
// escritura es atómica.
func PendingTimeFor(f Folder, role string) (PendingTime, bool, error) {
	doc, err := readPending(f)
	if err != nil {
		return PendingTime{}, false, err
	}
	for _, p := range doc.Pending {
		if p.Role == role {
			return p, true, nil
		}
	}
	return PendingTime{}, false, nil
}

// ClearPendingTime borra la intención del rol bajo el cerrojo. Solo debe llamarse cuando consta que
// la imputación no se aplicó o ya se registró: la reconciliación la hace el llamante, sobre
// PendingTime.ItemID. Sin intención no hace nada; con el diario vacío, retira el fichero.
func ClearPendingTime(f Folder, role string) error {
	release, err := f.acquire()
	if err != nil {
		return err
	}
	defer release()
	doc, err := readPending(f)
	if err != nil {
		return err
	}
	var kept []PendingTime
	for _, p := range doc.Pending {
		if p.Role != role {
			kept = append(kept, p)
		}
	}
	if len(kept) == len(doc.Pending) {
		return nil
	}
	doc.Pending = kept
	if len(kept) == 0 {
		return f.remove(PendingTimeFile)
	}
	return writePending(f, doc)
}

func writePending(f Folder, doc pendingDoc) error {
	doc.Schema = PendingTimeSchemaV1
	doc.Change = f.Change()
	payload, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("serializar %s: %w", PendingTimeFile, err)
	}
	return f.writeAtomic(PendingTimeFile, pendingTempPattern, payload)
}
