// Package identitystore guarda la identidad de la persona en cada tracker: un fichero propio en
// ~/.axiom (respetando AXIOM_STATE_DIR), junto a workspaces.json, con esquema versionado, escritura
// atómica y modo 0600, indexado por plugin y cuenta (en Azure DevOps, la organización).
//
// Es un paquete hoja: solo importa la biblioteca estándar, internal/system e
// internal/components/filemerge. El fichero nunca contiene secretos.
package identitystore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

const (
	// SchemaV1 es el esquema del fichero de identidad.
	SchemaV1 = "axiom.integration-identities/v1"
	// FileName es el nombre del fichero, junto a workspaces.json.
	FileName = "integration-identities.json"

	fileMode = 0o600
)

// writeAtomic es la costura de las pruebas de fallo de escritura.
var writeAtomic = filemerge.WriteFileAtomicMode

// Entry es la identidad de la persona en una cuenta del tracker.
type Entry struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Source      string    `json:"source"`      // connection_data | member_selection
	ResolvedAt  time.Time `json:"resolved_at"` // UTC
}

// File es el contenido del fichero: plugin → cuenta (organización) → entrada.
type File struct {
	Schema  string                      `json:"schema"`
	Plugins map[string]map[string]Entry `json:"plugins"`
}

// Store es el almacén de identidades en Path.
type Store struct{ Path string }

// CorruptError indica que el fichero existe pero no se puede interpretar (JSON inválido o esquema
// desconocido). El almacén no lo sobrescribe: la persona lo revisa o lo borra a mano.
type CorruptError struct {
	Path  string
	Cause error
}

func (e *CorruptError) Error() string {
	return fmt.Sprintf("el fichero de identidad %s no se puede leer (%v): revísalo o bórralo a mano; Axiom no lo sobrescribe", e.Path, e.Cause)
}

func (e *CorruptError) Unwrap() error { return e.Cause }

// DefaultPath devuelve la ruta del fichero bajo el directorio de estado de Axiom.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil && system.Getenv("AXIOM_STATE_DIR") == "" {
		return "", fmt.Errorf("no se pudo localizar el directorio personal: %w", err)
	}
	return filepath.Join(system.AxiomDir(home), FileName), nil
}

// normalizeAccount normaliza la clave de cuenta: las organizaciones de Azure DevOps no distinguen
// mayúsculas.
func normalizeAccount(account string) string { return strings.ToLower(strings.TrimSpace(account)) }

func emptyFile() File { return File{Schema: SchemaV1, Plugins: map[string]map[string]Entry{}} }

// Load lee el fichero. Uno inexistente es un File vacío; uno ilegible o de otro esquema es
// *CorruptError.
func (s Store) Load() (File, error) {
	if s.Path == "" {
		return File{}, errors.New("el almacén de identidad no tiene ruta")
	}
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyFile(), nil
	}
	if err != nil {
		return File{}, fmt.Errorf("leer el fichero de identidad %s: %w", s.Path, err)
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, &CorruptError{Path: s.Path, Cause: err}
	}
	if file.Schema != SchemaV1 {
		return File{}, &CorruptError{Path: s.Path, Cause: fmt.Errorf("esquema %q desconocido; se esperaba %s", file.Schema, SchemaV1)}
	}
	if file.Plugins == nil {
		file.Plugins = map[string]map[string]Entry{}
	}
	return file, nil
}

// Get devuelve la entrada de un plugin y una cuenta.
func (s Store) Get(plugin, account string) (Entry, bool, error) {
	file, err := s.Load()
	if err != nil {
		return Entry{}, false, err
	}
	entry, ok := file.Plugins[plugin][normalizeAccount(account)]
	return entry, ok, nil
}

// Put guarda la entrada de un plugin y una cuenta: lee el fichero, sustituye solo esa entrada y lo
// reescribe de forma atómica con modo 0600. Las demás entradas no cambian. Un fichero corrupto no se
// sobrescribe.
func (s Store) Put(plugin, account string, e Entry) error {
	key := normalizeAccount(account)
	switch {
	case strings.TrimSpace(plugin) == "":
		return errors.New("el plugin de la identidad no puede estar vacío")
	case key == "":
		return errors.New("la cuenta de la identidad no puede estar vacía")
	case e.ID == "":
		return errors.New("la identidad no trae identificador")
	}
	file, err := s.Load()
	if err != nil {
		return err
	}
	if file.Plugins[plugin] == nil {
		file.Plugins[plugin] = map[string]Entry{}
	}
	e.ResolvedAt = e.ResolvedAt.UTC()
	file.Plugins[plugin][key] = e

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar el fichero de identidad: %w", err)
	}
	if _, err := writeAtomic(s.Path, append(data, '\n'), fileMode); err != nil {
		return fmt.Errorf("guardar el fichero de identidad %s: %w", s.Path, err)
	}
	return nil
}
