package changestart

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

const (
	// ContextSchema es el esquema de context/work-item.json.
	ContextSchema = "axiom.change-start-context/v1"

	contextDirMode  = 0o755
	contextFileMode = 0o644
)

// fsSeams son las costuras de prueba del sistema de ficheros: escritura, renombrado y borrado.
// Los fallos de E/S de D-25 se inyectan aquí, nunca con permisos del sistema de ficheros. Una costura
// ausente usa la implementación real.
type fsSeams struct {
	MkdirAll  func(path string, perm os.FileMode) error
	WriteFile func(path string, data []byte, perm os.FileMode) error
	Rename    func(oldPath, newPath string) error
	RemoveAll func(path string) error
	// CopyTree y VerifyTree son los pasos de copia y verificación de context/ de la fase 2 (D-15).
	CopyTree   func(src, dst string) error
	VerifyTree func(src, dst string) error
	// LinkWorkItem y EnsureIgnored son los pasos 3 y 4 de la fase 2: el vínculo en integrations.yaml y la
	// exclusión de context/ en info/exclude (D-25).
	LinkWorkItem  func(f integrations.Folder, pluginID string, l integrations.Link, now time.Time) (bool, error)
	EnsureIgnored func(workspaceRoot string) error
}

func (f fsSeams) withDefaults() fsSeams {
	if f.CopyTree == nil {
		seams := f
		f.CopyTree = func(src, dst string) error { return copyTree(seams, src, dst) }
	}
	if f.VerifyTree == nil {
		f.VerifyTree = verifyTree
	}
	if f.LinkWorkItem == nil {
		f.LinkWorkItem = integrations.LinkWorkItem
	}
	if f.EnsureIgnored == nil {
		f.EnsureIgnored = state.EnsureChangeContextIgnored
	}
	if f.MkdirAll == nil {
		f.MkdirAll = os.MkdirAll
	}
	if f.WriteFile == nil {
		f.WriteFile = func(path string, data []byte, perm os.FileMode) error {
			if err := os.MkdirAll(filepath.Dir(path), contextDirMode); err != nil {
				return err
			}
			_, err := filemerge.WriteFileAtomic(path, data, perm)
			return err
		}
	}
	if f.Rename == nil {
		f.Rename = os.Rename
	}
	if f.RemoveAll == nil {
		f.RemoveAll = os.RemoveAll
	}
	return f
}

// Env es el entorno de las operaciones. La CLI lo construye con D-23: GitRoot y CommonDir son los del
// repositorio que contiene WorkspaceRoot.
type Env struct {
	WorkspaceRoot, PlanningHome, GitRoot, CommonDir string
	Now                                             func() time.Time
	FS                                              fsSeams
	// LockWait acota la espera de un cerrojo ocupado; cero usa 5 s.
	LockWait time.Duration
}

func (e Env) withDefaults() Env {
	if e.Now == nil {
		e.Now = time.Now
	}
	e.FS = e.FS.withDefaults()
	return e
}

// v1 es la raíz del staging: <common>/axiom/change-start/v1.
func (e Env) v1() string { return filepath.Join(e.CommonDir, "axiom", "change-start", "v1") }

// StagingDir es <v1>/<cambio>; su context/ es el contenido importado.
func (e Env) StagingDir(change string) string { return filepath.Join(e.v1(), change) }

func (e Env) tmpDir(change string) string { return filepath.Join(e.v1(), ".tmp", change) }

// ContextDoc es context/work-item.json: solo datos de confianza (D-12, D-20).
type ContextDoc struct {
	Schema      string      `json:"schema"`
	Change      string      `json:"change"`
	Plugin      string      `json:"plugin"`
	ID          string      `json:"id"`
	URL         string      `json:"url,omitempty"`
	ExternalRef string      `json:"external_ref,omitempty"`
	Profile     string      `json:"profile,omitempty"`
	TypeKey     string      `json:"type_key,omitempty"`
	StagedAt    string      `json:"staged_at,omitempty"`
	Lane        string      `json:"lane,omitempty"`
	AdoptedAt   string      `json:"adopted_at,omitempty"`
	Backlink    string      `json:"backlink,omitempty"`
	Attachments Attachments `json:"attachments"`
}

// Attachments cuenta los recursos importados y los omitidos.
type Attachments struct {
	Downloaded int `json:"downloaded"`
	Omitted    int `json:"omitted"`
}

// readContextDoc lee un work-item.json y lo valida: esquema, plugin e id presentes. Cualquier fallo
// significa «no legible» y el llamador trata el directorio como inexistente (D-12).
func readContextDoc(path string) (ContextDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ContextDoc{}, err
	}
	var doc ContextDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return ContextDoc{}, err
	}
	if doc.Schema != ContextSchema || doc.Plugin == "" || doc.ID == "" {
		return ContextDoc{}, errors.New("work-item.json no es un contexto completo")
	}
	return doc, nil
}

// stagingEntry es un staging con su work-item.json legible.
type stagingEntry struct {
	Name string
	Dir  string
	Doc  ContextDoc
}

// stagingFor lee el staging de un cambio; ok es false si no existe o no es legible.
func (e Env) stagingFor(change string) (stagingEntry, bool) {
	dir := e.StagingDir(change)
	doc, err := readContextDoc(filepath.Join(dir, "context", "work-item.json"))
	if err != nil {
		return stagingEntry{}, false
	}
	return stagingEntry{Name: change, Dir: dir, Doc: doc}, true
}

// stagingsOf devuelve los stagings válidos del mismo (plugin, id), de cualquier nombre.
func (e Env) stagingsOf(plugin, id string) []stagingEntry {
	entries, err := os.ReadDir(e.v1())
	if err != nil {
		return nil
	}
	var out []stagingEntry
	for _, ent := range entries {
		if !ent.IsDir() || strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		if s, ok := e.stagingFor(ent.Name()); ok && s.Doc.Plugin == plugin && s.Doc.ID == id {
			out = append(out, s)
		}
	}
	return out
}

// randomName es un sufijo aleatorio para temporales; los barre el dueño del cerrojo del cambio.
func randomName() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "t" + hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}

// retire retira un directorio sin dejar nunca uno a medio borrar con work-item.json legible: primero
// lo renombra atómicamente a <v1>/.tmp/<cambio>/<aleatorio>/ y solo después lo borra (D-12). Un fallo
// al renombrar se devuelve; un fallo al borrar el renombrado no es un fallo (lo barre la siguiente
// operación del cambio).
func (e Env) retire(dir, change string) error {
	_, err := e.retireReporting(dir, change)
	return err
}

// retireReporting es retire con el fallo del borrado final, que no es un fallo de la retirada: lo
// devuelve aparte para que el llamador lo avise si quiere (descarte, D-25).
func (e Env) retireReporting(dir, change string) (removeErr, err error) {
	parking := filepath.Join(e.tmpDir(change), randomName())
	if err := e.FS.MkdirAll(filepath.Dir(parking), contextDirMode); err != nil {
		return nil, err
	}
	if err := e.FS.Rename(dir, parking); err != nil {
		return nil, err
	}
	return e.FS.RemoveAll(parking), nil
}

func isNotExist(err error) bool { return errors.Is(err, os.ErrNotExist) }

// WorkItemOrigin son los datos del work item de origen de un cambio que necesita el motor de ramas
// para renderizar `{ref}` y `{kind}` (C-07-f): el plugin, el identificador y el perfil declarado
// (`bug` da `{kind}` = `fix`).
type WorkItemOrigin struct {
	Plugin, ID, Profile string
}

// WorkItemFacts lee, solo de ficheros, el work item de origen de un cambio: primero su staging
// (<commonDir>/axiom/change-start/v1/<cambio>, antes de `--adopt`) y, sin él, el work-item.json que
// ya adoptó el cambio en adoptedDir (SDD: <raíz del cambio>/context; ODD: odd/context/<cambio>). ok es
// false si no hay ninguno legible. No toma cerrojos ni escribe nada.
func WorkItemFacts(commonDir, change, adoptedDir string) (WorkItemOrigin, bool) {
	if commonDir != "" {
		if staging, ok := (Env{CommonDir: commonDir}).stagingFor(change); ok {
			return originOf(staging.Doc), true
		}
	}
	if adoptedDir != "" {
		if doc, err := readContextDoc(filepath.Join(adoptedDir, "work-item.json")); err == nil {
			return originOf(doc), true
		}
	}
	return WorkItemOrigin{}, false
}

func originOf(doc ContextDoc) WorkItemOrigin {
	return WorkItemOrigin{Plugin: doc.Plugin, ID: doc.ID, Profile: doc.Profile}
}
