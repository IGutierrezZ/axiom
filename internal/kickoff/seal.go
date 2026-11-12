package kickoff

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
	"gopkg.in/yaml.v3"
)

// KickoffFileName is the sealed document's file name inside a change root.
const KickoffFileName = "kickoff.yaml"

// nowFunc is the injectable clock Seal uses for SealedAt. Production always
// resolves to time.Now().UTC(); tests substitute it to assert on a fixed
// timestamp without depending on wall-clock time.
var nowFunc = func() time.Time { return time.Now().UTC() }

// publishKickoff is a test seam over reviewtransaction.PublishFileNoReplace,
// mirroring the seam sddstatus.ensureChangeInstanceMarker already uses for
// its own single-write publication (edit_authority_consent.go:91).
var publishKickoff = reviewtransaction.PublishFileNoReplace

func kickoffPath(changeRoot string) string {
	return filepath.Join(changeRoot, KickoffFileName)
}

// Load reads the sealed kickoff document for a change. Absence of the file
// is not an error: it means "not sealed yet", so Load returns (nil, nil).
// Any other read or parse failure is a named error — a corrupt document is
// never treated as an unsealed change (T-10).
func Load(changeRoot string) (*Kickoff, error) {
	path := kickoffPath(changeRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("leer kickoff sellado en %q: %w", path, err)
	}
	parsed, err := ParseKickoff(data)
	if err != nil {
		return nil, fmt.Errorf("kickoff sellado invalido en %q: %w", path, err)
	}
	return &parsed, nil
}

// Seal writes k as the sealed kickoff.yaml for changeRoot exactly once.
// The first successful publication wins for the life of the change
// instance (D-01, D-02): a later Seal call with different content never
// overwrites it. Instead, Seal re-reads and returns the winner already on
// disk with ok=false, so the caller knows its own value was not persisted.
//
// The implementation mirrors sddstatus.ensureChangeInstanceMarker: publish
// a temporary file in changeRoot via PublishFileNoReplace, and on
// os.ErrExist, re-read the document that won the race instead of failing.
func Seal(changeRoot string, k Kickoff) (Kickoff, bool, error) {
	if k.SealedAt.IsZero() {
		k.SealedAt = nowFunc()
	}
	if err := k.Validate(); err != nil {
		return Kickoff{}, false, err
	}

	payload, err := yaml.Marshal(k)
	if err != nil {
		return Kickoff{}, false, fmt.Errorf("serializar kickoff.yaml: %w", err)
	}

	path := kickoffPath(changeRoot)
	temporary, err := os.CreateTemp(changeRoot, ".kickoff-*.yaml")
	if err != nil {
		return Kickoff{}, false, fmt.Errorf("crear publicacion temporal de kickoff.yaml: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := temporary.Write(payload); err != nil {
		_ = temporary.Close()
		return Kickoff{}, false, fmt.Errorf("escribir publicacion temporal de kickoff.yaml: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return Kickoff{}, false, fmt.Errorf("cerrar publicacion temporal de kickoff.yaml: %w", err)
	}

	if err := publishKickoff(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			winner, readErr := Load(changeRoot)
			if readErr != nil {
				return Kickoff{}, false, readErr
			}
			if winner == nil {
				return Kickoff{}, false, fmt.Errorf("kickoff.yaml existe pero no se pudo releer en %q", path)
			}
			return *winner, false, nil
		}
		return Kickoff{}, false, fmt.Errorf("publicar kickoff.yaml: %w", err)
	}

	winner, err := Load(changeRoot)
	if err != nil {
		return Kickoff{}, false, err
	}
	if winner == nil {
		return Kickoff{}, false, fmt.Errorf("kickoff.yaml publicado pero ausente en la relectura de %q", path)
	}
	return *winner, true, nil
}

// PrepareChangeRoot crea changeRoot y sus padres ausentes, uno a uno, y devuelve una
// función que borra, en orden inverso, solo los directorios que creó y solo si están
// vacíos (REQ-29.12). Si changeRoot ya existe no crea nada y la retirada no hace nada.
// La retirada nunca borra ficheros: un directorio que ya no está vacío se respeta.
//
// Cada nivel se crea con os.Mkdir, no con os.MkdirAll, para saber con exactitud cuáles
// existían antes y cuáles son propios. Quien llama valida antes el nombre del cambio:
// PrepareChangeRoot no interpreta changeRoot, solo lo crea.
func PrepareChangeRoot(changeRoot string) (rollback func(), err error) {
	noop := func() {}
	cleaned := filepath.Clean(changeRoot)
	info, statErr := os.Stat(cleaned)
	switch {
	case statErr == nil && info.IsDir():
		return noop, nil
	case statErr == nil:
		return noop, fmt.Errorf("la ruta del cambio %q existe y no es una carpeta", cleaned)
	case !os.IsNotExist(statErr):
		return noop, fmt.Errorf("comprobar la carpeta del cambio %q: %w", cleaned, statErr)
	}

	// Recoge los niveles ausentes, del más profundo al más superficial.
	var missing []string
	for current := cleaned; ; {
		if _, err := os.Stat(current); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return noop, fmt.Errorf("comprobar %q: %w", current, err)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	var created []string
	rollback = func() {
		for i := len(created) - 1; i >= 0; i-- {
			// os.Remove falla en un directorio no vacío: es la comprobación que se busca.
			_ = os.Remove(created[i])
		}
		created = nil
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0o755); err != nil {
			if os.IsExist(err) {
				continue // otra orden lo creó entre medias: no es propio
			}
			rollback()
			return noop, fmt.Errorf("crear la carpeta del cambio %q: %w", missing[i], err)
		}
		created = append(created, missing[i])
	}
	return rollback, nil
}
