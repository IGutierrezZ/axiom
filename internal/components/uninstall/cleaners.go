package uninstall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

type jsonPath []string

const maxManagedFileSize = 16 << 20

var managedPersonaFingerprints = []string{
	"## Personality",
	"Senior Architect",
	"## Rules",
}

func removeMarkdownSections(content string, sectionIDs ...string) (string, bool) {
	updated := content
	changed := false
	for _, sectionID := range sectionIDs {
		next := filemerge.InjectMarkdownSection(updated, sectionID, "")
		if next != updated {
			changed = true
			updated = next
		}
	}
	return updated, changed
}

func removeManagedPersonaPreamble(content string) (string, bool) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	markerIdx := strings.Index(normalized, "<!-- axiom:")
	if legacyIdx := strings.Index(normalized, "<!-- gentle-ai:"); legacyIdx >= 0 && (markerIdx < 0 || legacyIdx < markerIdx) {
		markerIdx = legacyIdx
	}

	prefix := normalized
	suffix := ""
	if markerIdx >= 0 {
		prefix = normalized[:markerIdx]
		suffix = normalized[markerIdx:]
	}

	if strings.TrimSpace(prefix) == "" || !looksLikeManagedPersonaPrefix(prefix) {
		return content, false
	}

	if markerIdx < 0 {
		return content, false
	}

	suffix = strings.TrimLeft(suffix, "\n")
	return suffix, true
}

var (
	embeddedPersonasOnce     sync.Once
	embeddedPersonasContents []string
)

// embeddedPersonas devuelve el contenido, sin espacios en los extremos y con saltos
// de línea LF, de cada persona que Axiom incrusta (persona-*.md de cualquier agente).
func embeddedPersonas() []string {
	embeddedPersonasOnce.Do(func() {
		_ = fs.WalkDir(assets.FS, ".", func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			name := entry.Name()
			if !strings.HasPrefix(name, "persona-") || !strings.HasSuffix(name, ".md") {
				return nil
			}
			content, readErr := assets.FS.ReadFile(path)
			if readErr != nil {
				return nil
			}
			embeddedPersonasContents = append(embeddedPersonasContents,
				strings.TrimSpace(strings.ReplaceAll(string(content), "\r\n", "\n")))
			return nil
		})
	})
	return embeddedPersonasContents
}

func looksLikeManagedPersonaPrefix(prefix string) bool {
	if (strings.Contains(prefix, "name: Gentle AI Persona") || strings.Contains(prefix, "name: Axiom Persona")) && strings.Contains(prefix, "description: Teaching-oriented persona") {
		return true
	}
	// El agente con estrategia de sustitución de fichero (Codex, Gemini) recibe la
	// persona entera al principio del fichero, sin marcadores: es de Axiom si es
	// byte a byte una de las que Axiom incrusta.
	if trimmed := strings.TrimSpace(strings.ReplaceAll(prefix, "\r\n", "\n")); trimmed != "" && slices.Contains(embeddedPersonas(), trimmed) {
		return true
	}

	for _, fingerprint := range managedPersonaFingerprints {
		if !strings.Contains(prefix, fingerprint) {
			return false
		}
	}

	return true
}

func removeJSONPaths(raw []byte, paths ...jsonPath) ([]byte, bool, error) {
	root, err := unmarshalJSONObject(raw)
	if err != nil {
		return nil, false, err
	}

	changed := false
	for _, path := range paths {
		if deleteJSONPath(root, path) {
			changed = true
		}
	}
	if !changed {
		return raw, false, nil
	}

	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("marshal json after cleanup: %w", err)
	}

	eol := detectEOL(string(raw))
	encoded = bytes.ReplaceAll(encoded, []byte("\n"), []byte(eol))
	return append(encoded, []byte(eol)...), true, nil
}

func deleteJSONPath(root map[string]any, path jsonPath) bool {
	if len(path) == 0 {
		return false
	}

	key := path[0]
	value, ok := root[key]
	if !ok {
		return false
	}

	if len(path) == 1 {
		delete(root, key)
		return true
	}

	child, ok := value.(map[string]any)
	if !ok {
		return false
	}

	changed := deleteJSONPath(child, path[1:])
	if changed && len(child) == 0 {
		delete(root, key)
	}
	return changed
}

func jsonIsEmptyObject(raw []byte) bool {
	root, err := unmarshalJSONObject(raw)
	if err != nil {
		return false
	}
	return len(root) == 0
}

// cleanCodexTOML retira de ~/.codex/config.toml lo que escribe el componente Engram:
// su tabla de MCP, las claves de los ficheros de instrucciones y, si valen lo que
// escribe el instalador, features.multi_agent y agents.max_threads/max_depth (los
// valores distintos son del usuario).
func cleanCodexTOML(content string) (string, bool) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	updated := filemerge.RemoveTOMLTable(normalized, "mcp_servers.engram")
	updated = removeTopLevelTOMLKeys(updated, "model_instructions_file", "experimental_compact_prompt_file")
	updated = removeCodexMultiAgentKeys(updated)
	updated = strings.TrimSpace(updated)
	if updated != "" {
		updated += "\n"
	}
	return updated, updated != normalized
}

func removeTopLevelTOMLKeys(content string, keys ...string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	cleaned := make([]string, 0, len(lines))
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inTable = true
			cleaned = append(cleaned, line)
			continue
		}
		if inTable {
			cleaned = append(cleaned, line)
			continue
		}
		remove := false
		for key := range keySet {
			if strings.HasPrefix(trimmed, key+" ") || strings.HasPrefix(trimmed, key+"=") {
				remove = true
				break
			}
		}
		if !remove {
			cleaned = append(cleaned, line)
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func unmarshalJSONObject(raw []byte) (map[string]any, error) {
	object := map[string]any{}
	if len(bytes.TrimSpace(raw)) == 0 {
		return object, nil
	}

	if err := decodeJSONObject(raw, &object); err == nil {
		return object, nil
	}

	normalized := normalizeJSON(raw)
	if err := decodeJSONObject(normalized, &object); err != nil {
		return nil, fmt.Errorf("unmarshal json object: %w", err)
	}

	return object, nil
}

func decodeJSONObject(raw []byte, target *map[string]any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON payload")
		}
		return err
	}
	return nil
}

func detectEOL(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func restoreEOL(content, eol string) string {
	if eol == "\n" {
		return content
	}
	return strings.ReplaceAll(content, "\n", eol)
}

func readManagedFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("stat file %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to read symlink %q", path)
	}
	if info.Size() > maxManagedFileSize {
		return nil, fmt.Errorf("file %q exceeds max managed size %d bytes", path, maxManagedFileSize)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxManagedFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	if len(data) > maxManagedFileSize {
		return nil, fmt.Errorf("file %q exceeds max managed size %d bytes", path, maxManagedFileSize)
	}
	return data, nil
}

func normalizeJSON(raw []byte) []byte {
	withoutComments := stripJSONComments(raw)
	return stripTrailingCommas(withoutComments)
}

func stripJSONComments(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	inString := false
	escaped := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				out = append(out, ch)
			}
			continue
		}

		if inBlockComment {
			if ch == '*' && i+1 < len(raw) && raw[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			out = append(out, ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out = append(out, ch)
			continue
		}

		if ch == '/' && i+1 < len(raw) {
			next := raw[i+1]
			if next == '/' {
				inLineComment = true
				i++
				continue
			}
			if next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		out = append(out, ch)
	}

	return out
}

func stripTrailingCommas(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	inString := false
	escaped := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if inString {
			out = append(out, ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out = append(out, ch)
			continue
		}

		if ch == ',' {
			j := i + 1
			for j < len(raw) {
				next := raw[j]
				if next == ' ' || next == '\t' || next == '\n' || next == '\r' {
					j++
					continue
				}
				if next == '}' || next == ']' {
					ch = 0
				}
				break
			}
		}

		if ch != 0 {
			out = append(out, ch)
		}
	}

	return out
}

// removeOverlayEntries retira de raw las entradas que declara overlay y solo
// ellas: las cadenas de una lista (permissions.deny) y las claves cuyo valor
// coincide exactamente con el que escribe Axiom. El resto de la lista, las claves
// con otro valor y las que Axiom no escribe se conservan tal cual, y un objeto
// que Axiom dejó vacío al retirar sus entradas desaparece con ellas.
func removeOverlayEntries(raw []byte, overlay []byte) ([]byte, bool, error) {
	root, err := unmarshalJSONObject(raw)
	if err != nil {
		return nil, false, err
	}
	var declared map[string]any
	if err := json.Unmarshal(overlay, &declared); err != nil {
		return nil, false, fmt.Errorf("decode overlay: %w", err)
	}
	if !removeOverlayFrom(root, declared) {
		return raw, false, nil
	}
	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("marshal json after cleanup: %w", err)
	}
	eol := detectEOL(string(raw))
	encoded = bytes.ReplaceAll(encoded, []byte("\n"), []byte(eol))
	return append(encoded, []byte(eol)...), true, nil
}

// removeOverlayFrom aplica removeOverlayEntries sobre un objeto ya decodificado.
func removeOverlayFrom(root, overlay map[string]any) bool {
	changed := false
	for key, declared := range overlay {
		current, ok := root[key]
		if !ok {
			continue
		}
		switch want := declared.(type) {
		case map[string]any:
			child, ok := current.(map[string]any)
			if !ok {
				continue
			}
			if removeOverlayFrom(child, want) {
				changed = true
				if len(child) == 0 {
					delete(root, key)
				}
			}
		case []any:
			list, ok := current.([]any)
			if !ok {
				continue
			}
			kept := make([]any, 0, len(list))
			for _, item := range list {
				if !slices.ContainsFunc(want, func(candidate any) bool { return reflect.DeepEqual(candidate, item) }) {
					kept = append(kept, item)
				}
			}
			if len(kept) == len(list) {
				continue
			}
			changed = true
			if len(kept) == 0 {
				delete(root, key)
			} else {
				root[key] = kept
			}
		default:
			if reflect.DeepEqual(want, current) {
				delete(root, key)
				changed = true
			}
		}
	}
	return changed
}
