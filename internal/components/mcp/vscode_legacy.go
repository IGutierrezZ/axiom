package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

// context7RemoteURL es el servidor remoto de Context7 que las instalaciones
// anteriores declaraban para VS Code en .vscode/mcp.json.
const context7RemoteURL = "https://mcp.context7.com/mcp"

// RetireLegacyVSCodeServer retira el servidor name (engram o context7) de
// <proyecto>/.vscode/mcp.json, donde lo declaraban las instalaciones anteriores con la
// clave `servers`, del arnés Local de VS Code. Ahora va en <proyecto>/.mcp.json.
//
// Solo retira la entrada que escribió Axiom: la misma forma, sin claves añadidas por
// el usuario. El resto del fichero conserva su texto y sus comentarios. Si el
// documento se queda sin miembros, se borra, y con él la carpeta .vscode si queda
// vacía. Un fichero ausente, ilegible o con otra definición no se toca: la entrada
// del usuario sigue en su sitio y VS Code la reenvía al arnés de Copilot.
func RetireLegacyVSCodeServer(path, name string) (bool, error) {
	if path == "" {
		return false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("leer %s: %w", path, err)
	}
	root, err := filemerge.UnmarshalJSONObject(raw)
	if err != nil {
		return false, nil
	}
	servers, _ := root["servers"].(map[string]any)
	if !isLegacyVSCodeServer(name, servers[name]) {
		return false, nil
	}
	updated, changed, err := filemerge.RemoveJSONCObjectMember(raw, "servers", name)
	if err != nil || !changed {
		return false, nil
	}
	if rest, err := filemerge.UnmarshalJSONObject(updated); err == nil && len(rest) == 0 {
		if err := os.Remove(path); err != nil {
			return false, fmt.Errorf("borrar %s: %w", path, err)
		}
		// La carpeta .vscode solo se quita si queda vacía: casi siempre es del equipo.
		_ = os.Remove(filepath.Dir(path))
		return true, nil
	}
	if _, err := filemerge.WriteFileAtomic(path, updated, 0o644); err != nil {
		return false, fmt.Errorf("reescribir %s: %w", path, err)
	}
	return true, nil
}

// isLegacyVSCodeServer dice si la entrada tiene la forma que Axiom escribía para VS
// Code en .vscode/mcp.json, sin claves de más: Engram, una orden engram con `mcp`
// como primer argumento; Context7, el servidor remoto por http o el paquete de npx.
func isLegacyVSCodeServer(name string, value any) bool {
	entry, ok := value.(map[string]any)
	if !ok {
		return false
	}
	switch name {
	case "engram":
		if !onlyKeys(entry, "type", "command", "args") || !typeIs(entry, "stdio") {
			return false
		}
		command, _ := entry["command"].(string)
		args, _ := entry["args"].([]any)
		base := strings.ToLower(filepath.Base(strings.ReplaceAll(command, `\`, "/")))
		return (base == "engram" || base == "engram.exe") && len(args) > 0 && args[0] == "mcp"
	case "context7":
		if url, ok := entry["url"].(string); ok {
			return onlyKeys(entry, "type", "url") && typeIs(entry, "http") && strings.TrimRight(url, "/") == context7RemoteURL
		}
		if !onlyKeys(entry, "type", "command", "args") || !typeIs(entry, "stdio") || entry["command"] != "npx" {
			return false
		}
		args, _ := entry["args"].([]any)
		hasPackage, hasBinary := false, false
		for _, arg := range args {
			text, _ := arg.(string)
			hasPackage = hasPackage || strings.HasPrefix(text, "--package=@upstash/context7-mcp@")
			hasBinary = hasBinary || text == "context7-mcp"
		}
		return hasPackage && hasBinary
	}
	return false
}

// onlyKeys dice si entry no tiene más claves que las permitidas.
func onlyKeys(entry map[string]any, allowed ...string) bool {
	for key := range entry {
		found := false
		for _, candidate := range allowed {
			found = found || key == candidate
		}
		if !found {
			return false
		}
	}
	return true
}

// typeIs dice si entry no declara `type` o declara el esperado.
func typeIs(entry map[string]any, want string) bool {
	value, declared := entry["type"]
	return !declared || value == want
}
