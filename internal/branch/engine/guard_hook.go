package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// El borde de proceso de la guarda D (design D-18): las cargas de los tres runtimes con hooks, el
// presupuesto de tiempo y el protocolo de salida. Nada de esto interpreta ni ejecuta el contenido
// de la carga: solo extrae el evento, la herramienta, `cwd` y las rutas destino.

// Los runtimes con hook de `axiom branch guard` (`--agent`).
const (
	GuardAgentClaudeCode = "claude-code"
	GuardAgentCodex      = "codex"
	GuardAgentOpenCode   = "opencode"
	// GuardAgentVSCode es VS Code Copilot (el arnés de Copilot, hooks de .github/hooks/*.json).
	GuardAgentVSCode = "vscode-copilot"
)

// GuardAgents son los valores admitidos de `--agent`.
var GuardAgents = []string{GuardAgentClaudeCode, GuardAgentCodex, GuardAgentOpenCode, GuardAgentVSCode}

// GuardAgentNeedsEvent dice si la carga del runtime no trae el evento y la orden del hook debe
// declararlo con `--event pre|post` (VS Code Copilot).
func GuardAgentNeedsEvent(agent string) bool {
	return agent == GuardAgentVSCode
}

const (
	// GuardMaxPayload es el tamaño máximo de la carga que se lee de la entrada estándar: 256 KiB.
	// Una carga mayor no se lee entera y la guarda no hace nada.
	GuardMaxPayload = 256 << 10
	// GuardBudget es el presupuesto de toda la ejecución, la lectura de la carga incluida: agotado,
	// la guarda no hace nada.
	GuardBudget = time.Second
	// guardBlockExit es el código de salida con el que Claude Code y Codex bloquean (o devuelven el
	// aviso al agente): el motivo va por `stderr`.
	guardBlockExit = 2
)

// ParseGuardPayload lee la carga del runtime agent de r, con un máximo de GuardMaxPayload bytes, y
// devuelve el evento, la herramienta y las rutas destino, limpias y absolutas contra `cwd`. event
// es el que declara la orden del hook para los runtimes cuya carga no lo trae
// (GuardAgentNeedsEvent); con los demás se ignora y se pasa vacío.
//
//   - Claude Code: `hook_event_name` (`PreToolUse` o `PostToolUse`), `tool_name`, `cwd` y
//     `tool_input.file_path` o `tool_input.notebook_path`.
//   - Codex: la misma envoltura y, de `apply_patch`, las cabeceras `*** Add File:`,
//     `*** Update File:`, `*** Delete File:` y `*** Move to:` del parche, que se buscan en todo
//     `tool_input` porque la forma de la carga, tomada de la documentación oficial de Codex, no
//     está verificada en una ejecución real (A-2, spike-guard-runtimes.md).
//   - OpenCode: `{event, tool, filePath, cwd}`, que envía su plugin (`event` es `pre` o `post`).
//
// Una carga demasiado grande, JSON inválido o un agente desconocido es un error: quien llama falla
// en abierto.
//
// VS Code Copilot (hooks camelCase `preToolUse` y `postToolUse` del arnés de Copilot,
// docs.github.com/en/copilot/reference/hooks-reference) envía `{sessionId, timestamp, cwd,
// toolName, toolArgs}`, y en `postToolUse` además `toolResult`. `toolArgs` llega como cadena con
// JSON dentro (o como objeto), y la documentación no fija los campos de cada herramienta: se leen
// `path`, `file_path` y `filePath`, y las cabeceras de parche de `apply_patch`, venga el parche en
// un campo o como cadena entera. Solo cuentan las herramientas de escritura del arnés (`create`,
// `edit`, `str_replace_editor` salvo su orden `view`, `apply_patch`); con cualquier otra no hay
// rutas y la guarda no hace nada.
func ParseGuardPayload(agent, event string, r io.Reader) (GuardPayload, error) {
	data, err := io.ReadAll(io.LimitReader(r, GuardMaxPayload+1))
	if err != nil {
		return GuardPayload{}, fmt.Errorf("no se pudo leer la carga del hook: %w", err)
	}
	if len(data) > GuardMaxPayload {
		return GuardPayload{}, fmt.Errorf("la carga del hook supera %d bytes", GuardMaxPayload)
	}

	var payload GuardPayload
	var paths []string
	switch agent {
	case GuardAgentClaudeCode, GuardAgentCodex:
		var raw struct {
			CWD   string          `json:"cwd"`
			Event string          `json:"hook_event_name"`
			Tool  string          `json:"tool_name"`
			Input json.RawMessage `json:"tool_input"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return GuardPayload{}, fmt.Errorf("la carga del hook no es JSON válido: %w", err)
		}
		payload = GuardPayload{CWD: raw.CWD, Tool: raw.Tool, Event: claudeEvent(raw.Event)}
		if agent == GuardAgentClaudeCode {
			paths = claudePaths(raw.Input)
		} else {
			paths = codexPaths(raw.Input)
		}
	case GuardAgentOpenCode:
		var raw struct {
			Event    string `json:"event"`
			Tool     string `json:"tool"`
			FilePath string `json:"filePath"`
			CWD      string `json:"cwd"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return GuardPayload{}, fmt.Errorf("la carga del hook no es JSON válido: %w", err)
		}
		payload = GuardPayload{CWD: raw.CWD, Tool: raw.Tool}
		if raw.Event == GuardEventPre || raw.Event == GuardEventPost {
			payload.Event = raw.Event
		}
		paths = []string{raw.FilePath}
	case GuardAgentVSCode:
		var raw struct {
			CWD  string          `json:"cwd"`
			Tool string          `json:"toolName"`
			Args json.RawMessage `json:"toolArgs"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return GuardPayload{}, fmt.Errorf("la carga del hook no es JSON válido: %w", err)
		}
		payload = GuardPayload{CWD: raw.CWD, Tool: raw.Tool}
		if event == GuardEventPre || event == GuardEventPost {
			payload.Event = event
		}
		paths = vscodePaths(raw.Tool, raw.Args)
	default:
		return GuardPayload{}, fmt.Errorf("el agente %q no es uno de %s", agent, strings.Join(GuardAgents, ", "))
	}

	for _, path := range paths {
		if absolute, ok := absolutePath(path, payload.CWD); ok && !slices.Contains(payload.Paths, absolute) {
			payload.Paths = append(payload.Paths, absolute)
		}
	}
	return payload, nil
}

// claudeEvent traduce el `hook_event_name` de Claude Code y de Codex al evento de la guarda; otro
// evento no es de la guarda.
func claudeEvent(name string) string {
	switch name {
	case "PreToolUse":
		return GuardEventPre
	case "PostToolUse":
		return GuardEventPost
	}
	return ""
}

// claudePaths son los destinos de una herramienta de edición de Claude Code (`Write`, `Edit`,
// `MultiEdit` usan `file_path`; `NotebookEdit`, `notebook_path`).
func claudePaths(input json.RawMessage) []string {
	var tool struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	}
	if json.Unmarshal(input, &tool) != nil {
		return nil
	}
	return []string{tool.FilePath, tool.NotebookPath}
}

// codexPaths son los ficheros que nombran las cabeceras de un parche de `apply_patch`, en el orden
// en que aparecen. Busca en todos los textos de tool_input (la orden puede venir como cadena, como
// lista de cadenas o dentro de un objeto) y no interpreta nada más.
func codexPaths(input json.RawMessage) []string {
	var decoded any
	if json.Unmarshal(input, &decoded) != nil {
		return nil
	}
	return patchPathsIn(decoded)
}

// vscodeWriteTools son las herramientas de escritura del arnés de Copilot que vigila la guarda.
var vscodeWriteTools = []string{"create", "edit", "str_replace_editor", "apply_patch"}

// vscodePaths son los destinos de una herramienta de escritura de VS Code Copilot. args es
// `toolArgs`: una cadena con JSON dentro, un objeto o, en `apply_patch`, el parche en crudo.
func vscodePaths(tool string, args json.RawMessage) []string {
	if !slices.Contains(vscodeWriteTools, tool) {
		return nil
	}
	var decoded any
	if json.Unmarshal(args, &decoded) != nil {
		return nil
	}
	if text, ok := decoded.(string); ok {
		var inner any
		if json.Unmarshal([]byte(text), &inner) != nil {
			return patchHeaders(text)
		}
		decoded = inner
	}
	var paths []string
	if object, ok := decoded.(map[string]any); ok {
		if command, _ := object["command"].(string); tool == "str_replace_editor" && command == "view" {
			return nil
		}
		for _, key := range []string{"path", "file_path", "filePath"} {
			if path, ok := object[key].(string); ok {
				paths = append(paths, path)
			}
		}
	}
	return append(paths, patchPathsIn(decoded)...)
}

// patchPathsIn son los ficheros que nombran las cabeceras de parche de todos los textos de un
// valor JSON decodificado, en orden estable.
func patchPathsIn(decoded any) []string {
	var paths []string
	var walk func(value any)
	walk = func(value any) {
		switch typed := value.(type) {
		case string:
			paths = append(paths, patchHeaders(typed)...)
		case []any:
			for _, element := range typed {
				walk(element)
			}
		case map[string]any:
			for _, key := range slices.Sorted(maps.Keys(typed)) {
				walk(typed[key])
			}
		}
	}
	walk(decoded)
	return paths
}

// patchHeaderPrefixes son las cabeceras de `apply_patch` que nombran un fichero.
var patchHeaderPrefixes = []string{"*** Add File: ", "*** Update File: ", "*** Delete File: ", "*** Move to: "}

// patchHeaders devuelve los ficheros de las cabeceras de un parche, una por línea.
func patchHeaders(patch string) []string {
	var paths []string
	scanner := bufio.NewScanner(strings.NewReader(patch))
	scanner.Buffer(nil, GuardMaxPayload)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		for _, prefix := range patchHeaderPrefixes {
			if rest, ok := strings.CutPrefix(line, prefix); ok {
				if rest = strings.TrimSpace(rest); rest != "" {
					paths = append(paths, rest)
				}
				break
			}
		}
	}
	return paths
}

// absolutePath devuelve path limpio y absoluto: una ruta relativa se resuelve contra cwd y, si el
// runtime no lo envió, contra el directorio del proceso (que hereda el del runtime).
func absolutePath(path, cwd string) (string, bool) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) {
		return "", false
	}
	if !filepath.IsAbs(path) && cwd != "" {
		path = filepath.Join(cwd, path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return filepath.Clean(absolute), true
}

// guardOutput es la salida de `--agent opencode`, que interpreta su plugin.
type guardOutput struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

// RunGuard ejecuta el hook de `agent`: lee la carga de stdin, decide y escribe el resultado según
// el protocolo del runtime, todo dentro de GuardBudget y en abierto. Devuelve el código de salida:
// con `--agent claude-code` y `--agent codex`, 2 con el motivo por `stderr` si bloquea o avisa y 0
// en otro caso; con `--agent opencode`, siempre 0 con `{"action":"none|block|warn","message":"…"}`
// por `stdout`; con `--agent vscode-copilot`, siempre 0 (en `preToolUse` del arnés de Copilot,
// cualquier código distinto de 0 deniega la escritura): el bloqueo va por `stdout` como
// `{"permissionDecision":"deny","permissionDecisionReason":"…"}`, el aviso como
// `{"additionalContext":"…"}`, que el modelo lee tras el resultado de la herramienta, y sin nada
// que decir no escribe nada. event es el que declara la orden del hook (`pre` o `post`) para los
// runtimes cuya carga no lo trae; con los demás va vacío.
func RunGuard(agent, event string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runGuard(Default(), GuardBudget, agent, event, stdin, stdout, stderr)
}

func runGuard(e *Engine, budget time.Duration, agent, event string, stdin io.Reader, stdout, stderr io.Writer) int {
	decision := guardWithin(budget, func(ctx context.Context) GuardDecision {
		payload, err := ParseGuardPayload(agent, event, stdin)
		if err != nil {
			return guardNone
		}
		return e.Guard(ctx, payload)
	})
	return writeGuardDecision(agent, decision, stdout, stderr)
}

// vscodeGuardOutput es la salida de `--agent vscode-copilot`: la decisión de `preToolUse` o el
// contexto adicional de `postToolUse` del arnés de Copilot.
type vscodeGuardOutput struct {
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
}

// guardWithin ejecuta work con un presupuesto de tiempo: si se agota, o si work entra en pánico,
// devuelve `none` (fallo en abierto). work sigue ejecutándose en su gorrutina hasta que termine el
// proceso, que es corto.
func guardWithin(budget time.Duration, work func(context.Context) GuardDecision) GuardDecision {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	done := make(chan GuardDecision, 1)
	go func() {
		defer func() {
			if recover() != nil {
				done <- guardNone
			}
		}()
		done <- work(ctx)
	}()
	select {
	case decision := <-done:
		return decision
	case <-ctx.Done():
		return guardNone
	}
}

// writeGuardDecision escribe la decisión con el protocolo del runtime y devuelve el código de
// salida (véase RunGuard).
func writeGuardDecision(agent string, decision GuardDecision, stdout, stderr io.Writer) int {
	if agent == GuardAgentVSCode {
		var output vscodeGuardOutput
		switch decision.Action {
		case GuardActionBlock:
			output = vscodeGuardOutput{PermissionDecision: "deny", PermissionDecisionReason: decision.Message}
		case GuardActionWarn:
			output = vscodeGuardOutput{AdditionalContext: decision.Message}
		default:
			return 0
		}
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		if encoder.Encode(output) == nil {
			_, _ = stdout.Write(out.Bytes())
		}
		return 0
	}
	if agent == GuardAgentOpenCode {
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		if encoder.Encode(guardOutput{Action: decision.Action, Message: decision.Message}) == nil {
			_, _ = stdout.Write(out.Bytes())
		}
		return 0
	}
	if decision.Action == GuardActionNone {
		return 0
	}
	fmt.Fprintln(stderr, decision.Message)
	return guardBlockExit
}
