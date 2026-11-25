package mcpexclusion

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
)

// UsageWindow es la ventana en la que se cuenta el uso de un servidor MCP: un servidor
// cargado sin ninguna llamada en este tiempo se señala como «sin uso».
const UsageWindow = 30 * 24 * time.Hour

// maxTranscriptLine acota la memoria por línea de transcripción: una línea mayor se
// descarta entera (no puede ser una llamada a una herramienta MCP corriente).
const maxTranscriptLine = 32 << 20

// ServerUsage es el uso de un servidor MCP candidato en la ventana.
type ServerUsage struct {
	Name     string     `json:"name"`
	Uses     int        `json:"uses"`
	LastUsed *time.Time `json:"lastUsed,omitempty"`
	// Unused es verdadero si hay transcripciones del proyecto, el servidor no está
	// excluido y no se usó en la ventana.
	Unused bool `json:"unused"`
}

// UsageAnalysis es el resultado de Analyze. HasData es falso cuando Claude Code no
// guarda ninguna transcripción del proyecto: entonces nada se marca como sin uso, porque
// no hay datos de los que concluirlo («sin datos»).
type UsageAnalysis struct {
	HasData bool
	Servers []ServerUsage
}

// SegmentUse es el uso agregado de un segmento de nombre de herramienta MCP.
type SegmentUse struct {
	Count int
	Last  time.Time
}

// EncodeProjectDir devuelve el nombre de carpeta con que Claude Code guarda las
// transcripciones de un directorio en ~/.claude/projects: cada carácter que no es
// [A-Za-z0-9] pasa a «-» (`C:\repos\axiom` → `C--repos-axiom`; un carácter fuera del plano
// básico ocupa dos, como las unidades UTF-16 de JavaScript).
func EncodeProjectDir(dir string) string {
	var b strings.Builder
	for _, r := range dir {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r > 0xFFFF:
			b.WriteString("--")
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

// ServerSegment devuelve el segmento que Claude Code usa en `mcp__<segmento>__<tool>`
// para un servidor configurado: todo carácter fuera de [A-Za-z0-9_-] pasa a «_»
// (`claude.ai Microsoft 365` → `claude_ai_Microsoft_365`).
//
// No se pueden casar por nombre los conectores de claude.ai que la aplicación de
// escritorio nombra por su id (`b67fe70e-…`) ni los servidores de un plugin
// (`plugin_<plugin>_<servidor>`): su uso queda sin atribuir y el servidor puede salir
// como sin uso aunque se use. Por eso es una sugerencia y nunca una acción.
func ServerSegment(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// ScanUsage recorre las transcripciones de Claude Code de los directorios dados
// (~/.claude/projects/<dir codificado>/*.jsonl) modificadas dentro de UsageWindow y
// cuenta, por segmento de servidor, las llamadas a herramientas `mcp__<servidor>__*`.
// Solo lee el nombre de las llamadas: ningún contenido de mensaje sale de esta función.
// hasData dice si existe alguna transcripción de esos directorios, de cualquier edad.
func ScanUsage(home string, dirs []string, now time.Time) (usage map[string]SegmentUse, hasData bool) {
	usage = map[string]SegmentUse{}
	var seen []string
	for _, dir := range dirs {
		folder := filepath.Join(home, ".claude", "projects", EncodeProjectDir(dir))
		if slices.Contains(seen, folder) {
			continue
		}
		seen = append(seen, folder)
		files, err := filepath.Glob(filepath.Join(folder, "*.jsonl"))
		if err != nil {
			continue
		}
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil || info.IsDir() {
				continue
			}
			hasData = true
			if now.Sub(info.ModTime()) > UsageWindow {
				continue
			}
			scanTranscript(file, info.ModTime(), now, usage)
		}
	}
	return usage, hasData
}

func scanTranscript(path string, fallback, now time.Time, usage map[string]SegmentUse) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	reader := bufio.NewReaderSize(f, 64<<10)
	for {
		line, err := readLine(reader)
		if len(line) > 0 {
			countToolUses(line, fallback, now, usage)
		}
		if err != nil {
			return
		}
	}
}

// readLine lee una línea completa sin el límite del Scanner; una línea que supera
// maxTranscriptLine se consume entera y se devuelve vacía.
func readLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	tooLong := false
	for {
		chunk, err := reader.ReadSlice('\n')
		if !tooLong {
			line = append(line, chunk...)
			if len(line) > maxTranscriptLine {
				line, tooLong = nil, true
			}
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		return line, err
	}
}

var (
	toolUseMarker = []byte(`"tool_use"`)
	mcpMarker     = []byte(`mcp__`)
)

// countToolUses suma al uso los `tool_use` MCP de una línea dentro de la ventana. El
// filtro por subcadena evita decodificar el JSON de las líneas que no pueden contener
// una llamada MCP.
func countToolUses(line []byte, fallback, now time.Time, usage map[string]SegmentUse) {
	if !bytes.Contains(line, toolUseMarker) || !bytes.Contains(line, mcpMarker) {
		return
	}
	var entry struct {
		Timestamp string `json:"timestamp"`
		Message   struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &entry) != nil {
		return
	}
	var items []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if json.Unmarshal(entry.Message.Content, &items) != nil {
		return
	}
	when := fallback
	if ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
		when = ts
	}
	if now.Sub(when) > UsageWindow {
		return
	}
	for _, item := range items {
		if item.Type != "tool_use" || !strings.HasPrefix(item.Name, "mcp__") {
			continue
		}
		segment, _, ok := strings.Cut(strings.TrimPrefix(item.Name, "mcp__"), "__")
		if !ok || segment == "" {
			continue
		}
		current := usage[segment]
		current.Count++
		if when.After(current.Last) {
			current.Last = when
		}
		usage[segment] = current
	}
}

// ProjectMCPNames devuelve, ordenados, los nombres de los servidores de `<root>/.mcp.json`
// (las claves de `mcpServers`); un fichero ausente o ilegible da una lista vacía.
func ProjectMCPNames(root string) []string {
	raw, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	names := make([]string, 0, len(doc.MCPServers))
	for name := range doc.MCPServers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Analyze calcula el uso en los últimos 30 días de los servidores names (sin repetidos)
// en los directorios del proyecto de root (la raíz, sus worktrees y repositorios de
// roles). Un servidor excluido se informa pero nunca se marca como sin uso.
func Analyze(home, root string, names, excluded []string, now time.Time) UsageAnalysis {
	usage, hasData := ScanUsage(home, ProjectDirs(root), now)
	analysis := UsageAnalysis{HasData: hasData}
	sorted := slices.Clone(names)
	slices.Sort(sorted)
	for _, name := range slices.Compact(sorted) {
		use := usage[ServerSegment(name)]
		server := ServerUsage{Name: name, Uses: use.Count}
		if use.Count > 0 {
			last := use.Last
			server.LastUsed = &last
		}
		server.Unused = hasData && use.Count == 0 && !slices.Contains(excluded, name)
		analysis.Servers = append(analysis.Servers, server)
	}
	return analysis
}

// UnusedCandidates devuelve los servidores candidatos de un proyecto (los de usuario de
// Claude Code y los de su `.mcp.json`) cargados y sin uso en la ventana, descontando los
// ya excluidos. Sin transcripciones del proyecto devuelve nil: «sin datos».
func UnusedCandidates(home, root string, excluded []string, now time.Time) []ServerUsage {
	names := append(claude.UserMCPServerNames(home), ProjectMCPNames(root)...)
	var unused []ServerUsage
	for _, server := range Analyze(home, root, names, excluded, now).Servers {
		if server.Unused {
			unused = append(unused, server)
		}
	}
	return unused
}
