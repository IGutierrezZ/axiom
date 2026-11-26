package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/fileprim"
)

// Sello del último fetch correcto del autofetch (RF-BRA-39, design D-06): estado derivado y
// local a la máquina, por repositorio de specs, en el mismo sitio que el registro de ramas
// (`<commonDir>/axiom/...`). Solo `run` lo escribe, y solo tras un fetch correcto.

// Cotas del ciclo. Son variables de paquete para que las pruebas las reduzcan, como
// gitcommon.runTimeout. autofetchFetchTimeout (8 s por fetch) está en ensure.go y la comparte
// el fetch implícito de `ensure`.
var (
	// autofetchInterval es el tiempo mínimo entre dos fetch de `run`: dentro de él solo se
	// omite la red.
	autofetchInterval = 5 * time.Minute
	// autofetchRunBudget acota el ciclo entero de `run` (design D-09): 8 s de fetch más unos
	// 2 s para las lecturas locales y el avance.
	autofetchRunBudget = 10 * time.Second
	// autofetchAdvanceReserve es lo que debe quedar del presupuesto para iniciar el avance
	// por fast-forward (fila 8 de D-08).
	autofetchAdvanceReserve = time.Second
)

// autofetchStamp es el contenido de `state.json`.
type autofetchStamp struct {
	LastFetch string `json:"lastFetch"`
}

// autofetchStampPath es el fichero del sello bajo el git common dir del repositorio de specs.
func autofetchStampPath(commonDir string) string {
	return filepath.Join(commonDir, "axiom", "autofetch", "v1", "state.json")
}

// clock es el reloj del motor: el inyectado en las pruebas o el del sistema.
func (e *Engine) clock() time.Time {
	if e.now != nil {
		return e.now()
	}
	return time.Now()
}

// readAutofetchStamp lee el instante del último fetch correcto. Un sello ausente, ilegible, con
// una fecha que no es RFC 3339 o con una fecha futura cuenta como «sin fetch previo» (ok falso).
func readAutofetchStamp(commonDir string, now time.Time) (last time.Time, ok bool) {
	payload, err := os.ReadFile(autofetchStampPath(commonDir))
	if err != nil {
		return time.Time{}, false
	}
	var stamp autofetchStamp
	if err := json.Unmarshal(payload, &stamp); err != nil {
		return time.Time{}, false
	}
	last, err = time.Parse(time.RFC3339, stamp.LastFetch)
	if err != nil || last.After(now) {
		return time.Time{}, false
	}
	return last, true
}

// writeAutofetchStamp escribe el sello de forma atómica (temporal y renombrado).
func writeAutofetchStamp(commonDir string, at time.Time) error {
	payload, err := json.Marshal(autofetchStamp{LastFetch: at.UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	path := autofetchStampPath(commonDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, err := temporary.Write(payload); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := fileprim.ReplaceAtomic(temporaryPath, path); err != nil {
		return fmt.Errorf("sustituir %s: %w", path, err)
	}
	return nil
}
