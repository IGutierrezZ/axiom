package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

// Exclusiones MCP personales de Claude Code (feature contexto-por-proyecto): Claude Code
// fusiona `deniedMcpServers` de todos los ficheros de settings. Axiom lo escribe solo en
// `.claude/settings.local.json` (personal y fuera de git), nunca en `.claude/settings.json`
// (compartido con el equipo) ni en `~/.claude.json` (guarda la sesión OAuth).

const (
	deniedMcpServersKey = "deniedMcpServers"
	serverNameKey       = "serverName"
)

// LocalSettingsPath es el `settings.local.json` de proyecto de Claude Code en dir.
func LocalSettingsPath(dir string) string {
	return filepath.Join(dir, ".claude", "settings.local.json")
}

// ApplyDeniedMCPServers fusiona en `<dir>/.claude/settings.local.json` la lista
// `deniedMcpServers`: añade `{"serverName": n}` por cada nombre de add que aún no tenga
// una entrada con ese serverName (las que ya existen, p. ej. puestas a mano, se adoptan) y
// retira las entradas que son exactamente `{"serverName": n}` para cada nombre de remove.
// Conserva el resto de claves y las entradas ajenas (otros nombres, `serverUrl`,
// `serverCommand`) y su orden; los comentarios del fichero no se conservan y las claves
// de primer nivel se reescriben ordenadas. Si la lista queda vacía se quita la clave.
// Sin fichero y sin nada que añadir no crea nada. Un JSON ilegible es un error que nombra
// el fichero y lo deja intacto. Devuelve si escribió el fichero.
func ApplyDeniedMCPServers(dir string, add, remove []string) (bool, error) {
	path := LocalSettingsPath(dir)
	raw, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("no se pudo leer %s: %w", path, err)
	}
	if !exists && len(add) == 0 {
		return false, nil
	}

	root := map[string]any{}
	if exists {
		root, err = filemerge.UnmarshalJSONObject(raw)
		if err != nil {
			return false, fmt.Errorf("%s no es un JSON válido y no se modifica: %w", path, err)
		}
	}
	var entries []any
	if value, present := root[deniedMcpServersKey]; present {
		list, ok := value.([]any)
		if !ok {
			return false, fmt.Errorf("%s: %s no es una lista y no se modifica", path, deniedMcpServersKey)
		}
		entries = list
	}

	updated, modified := mergeDeniedEntries(entries, add, remove)
	if !modified {
		return false, nil
	}
	if len(updated) == 0 {
		delete(root, deniedMcpServersKey)
	} else {
		root[deniedMcpServersKey] = updated
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(root); err != nil {
		return false, fmt.Errorf("no se pudo codificar %s: %w", path, err)
	}
	if _, err := filemerge.WriteFileAtomic(path, buf.Bytes(), 0o644); err != nil {
		return false, fmt.Errorf("no se pudo escribir %s: %w", path, err)
	}
	return true, nil
}

func mergeDeniedEntries(entries []any, add, remove []string) ([]any, bool) {
	modified := false
	kept := make([]any, 0, len(entries)+len(add))
	for _, entry := range entries {
		if name, only := bareServerName(entry); only && slices.Contains(remove, name) && !slices.Contains(add, name) {
			modified = true
			continue
		}
		kept = append(kept, entry)
	}
	for _, name := range add {
		if hasServerName(kept, name) {
			continue
		}
		kept = append(kept, map[string]any{serverNameKey: name})
		modified = true
	}
	return kept, modified
}

// bareServerName devuelve el nombre de una entrada que es exactamente
// `{"serverName": n}`; cualquier otra forma (con serverUrl, serverCommand…) es ajena.
func bareServerName(entry any) (string, bool) {
	object, ok := entry.(map[string]any)
	if !ok || len(object) != 1 {
		return "", false
	}
	name, ok := object[serverNameKey].(string)
	return name, ok
}

func hasServerName(entries []any, name string) bool {
	for _, entry := range entries {
		if object, ok := entry.(map[string]any); ok && object[serverNameKey] == name {
			return true
		}
	}
	return false
}
