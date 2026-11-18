package sdd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
)

// reviewSubAgentNames son los agentes nativos de revisión de RDD. Con RDD
// suspendido (C-01-c) no se instalan y `sync` retira los que dejó una versión
// anterior. Los ficheros embebidos se conservan: uninstall los lee para borrar.
var reviewSubAgentNames = []string{
	"review-readability",
	"review-refuter",
	"review-reliability",
	"review-resilience",
	"review-risk",
}

// isReviewSubAgentFile dice si el fichero embebido es la definición de un
// agente de revisión (.md o .yaml), por nombre exacto.
func isReviewSubAgentFile(fileName string) bool {
	base := strings.TrimSuffix(strings.TrimSuffix(fileName, ".md"), ".yaml")
	for _, name := range reviewSubAgentNames {
		if base == name {
			return true
		}
	}
	return false
}

// IsRetiredReviewSubAgentPath dice si path es un agente de revisión que, con
// RDD suspendido, install y sync retiran en lugar de escribir; la verificación
// posterior debe comprobar su ausencia. Se limita al directorio de subagentes
// (`agents`, el que usan todos los adaptadores con subagentes) para no tratar
// como retirado un fichero homónimo de otra ruta gestionada.
func IsRetiredReviewSubAgentPath(path string) bool {
	if rddharness.Enabled() {
		return false
	}
	path = filepath.Clean(path)
	if filepath.Base(filepath.Dir(path)) != "agents" {
		return false
	}
	return isReviewSubAgentFile(filepath.Base(path))
}

// retireReviewSubAgentFile borra de agentsDir el agente de revisión con el
// nombre de la plantilla embebida. Solo actúa sobre nombres exactos de Axiom
// (el mismo criterio que usa uninstall); un fichero con otro nombre, aunque
// comparta el prefijo, no se toca. Informa si borró algo y no hace nada si el
// fichero no existe.
func retireReviewSubAgentFile(agentsDir, fileName string) (bool, error) {
	path := filepath.Join(agentsDir, fileName)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect retired review agent %q: %w", path, err)
	}
	if info.IsDir() {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("remove retired review agent %q: %w", path, err)
	}
	return true, nil
}

// retiredReviewStopHookCommands son las órdenes exactas que versiones
// anteriores registraron para el hook de parada de revisión de agentID, con el
// nombre actual y el heredado del binario.
func retiredReviewStopHookCommands(agentID string) []string {
	return []string{
		"axiom review stop-hook --agent " + agentID,
		"gentle-ai review stop-hook --agent " + agentID,
	}
}

// reviewStopHookEvents son los únicos eventos bajo los que se registró.
var reviewStopHookEvents = []string{"Stop", "SessionStart"}

// retireReviewStopHooks quita de Stop y SessionStart de hooksMap el hook de
// parada de revisión de agentID. Compara por cadena exacta, así que los hooks
// del usuario, incluso en la misma entrada, no se tocan. Una entrada sin hooks
// y un evento sin entradas se eliminan. Informa si hooksMap cambió.
func retireReviewStopHooks(hooksMap map[string]any, agentID string) bool {
	retired := retiredReviewStopHookCommands(agentID)
	changed := false
	for _, event := range reviewStopHookEvents {
		entries, ok := hooksMap[event].([]any)
		if !ok {
			continue
		}
		eventChanged := false
		kept := make([]any, 0, len(entries))
		for _, entry := range entries {
			entryMap, isMap := entry.(map[string]any)
			hooks, hasHooks := entryMap["hooks"].([]any)
			if !isMap || !hasHooks {
				kept = append(kept, entry)
				continue
			}
			remaining := make([]any, 0, len(hooks))
			for _, hook := range hooks {
				hookMap, _ := hook.(map[string]any)
				if command, _ := hookMap["command"].(string); isRetiredTelemetryCommand(command, retired) {
					eventChanged = true
					continue
				}
				remaining = append(remaining, hook)
			}
			if len(remaining) == len(hooks) {
				kept = append(kept, entry)
				continue
			}
			if len(remaining) == 0 {
				continue
			}
			entryMap["hooks"] = remaining
			kept = append(kept, entryMap)
		}
		if !eventChanged {
			continue
		}
		changed = true
		if len(kept) == 0 {
			delete(hooksMap, event)
		} else {
			hooksMap[event] = kept
		}
	}
	return changed
}

// retireClaudeReviewStopHook retira de settingsPath el hook de parada de
// revisión de Claude. Solo escribe si quitó algo y nunca crea el fichero.
func retireClaudeReviewStopHook(settingsPath, agentID string) (bool, error) {
	data, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return false, nil
	}
	root := map[string]any{}
	if err := json.Unmarshal(data, &root); err != nil {
		return false, fmt.Errorf("parse Claude settings %q: %w", settingsPath, err)
	}
	hooksMap, ok := root["hooks"].(map[string]any)
	if !ok || !retireReviewStopHooks(hooksMap, agentID) {
		return false, nil
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	out = append(out, '\n')
	wr, err := filemerge.WriteFileAtomic(settingsPath, out, 0o644)
	if err != nil {
		return false, err
	}
	return wr.Changed, nil
}

// stripOpenCodeReviewAgents quita del overlay de OpenCode los agentes de
// revisión de RDD y sus permisos de delegación. Los agentes de Judgment Day
// no se tocan.
func stripOpenCodeReviewAgents(agentsMap map[string]any) {
	for _, name := range opencode.ReviewPhases() {
		delete(agentsMap, name)
	}
	for _, raw := range agentsMap {
		deleteReviewTaskPermissions(raw)
	}
}

func deleteReviewTaskPermissions(raw any) bool {
	agent, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	permission, ok := agent["permission"].(map[string]any)
	if !ok {
		return false
	}
	task, ok := permission["task"].(map[string]any)
	if !ok {
		return false
	}
	if replacement, ok := task["__replace__"].(map[string]any); ok {
		task = replacement
	}
	changed := false
	for _, name := range opencode.ReviewPhases() {
		if _, has := task[name]; has {
			delete(task, name)
			changed = true
		}
	}
	return changed
}

// retireOpenCodeReviewAgents borra de la configuración instalada de OpenCode
// los agentes de revisión que Axiom gestiona (nombre y forma de la definición
// que escribe el overlay) y sus permisos de delegación en los orquestadores
// gestionados. Una definición del usuario con ese nombre pero otra forma se
// conserva. Solo escribe si quitó algo y nunca crea el fichero.
func retireOpenCodeReviewAgents(settingsPath string) (bool, error) {
	data, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return false, nil
	}
	root, err := filemerge.UnmarshalJSONObject(data)
	if err != nil {
		return false, nil
	}
	agentsMap, ok := root["agent"].(map[string]any)
	if !ok {
		return false, nil
	}
	changed := false
	for _, name := range opencode.ReviewPhases() {
		def, ok := agentsMap[name].(map[string]any)
		if !ok || !opencode.IsManagedAgent(name, def) {
			continue
		}
		delete(agentsMap, name)
		changed = true
	}
	for key, raw := range agentsMap {
		if !opencode.IsManagedAgentName(key) {
			continue
		}
		if deleteReviewTaskPermissions(raw) {
			changed = true
		}
	}
	if !changed {
		return false, nil
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	out = append(out, '\n')
	wr, err := filemerge.WriteFileAtomic(settingsPath, out, 0o644)
	if err != nil {
		return false, err
	}
	return wr.Changed, nil
}

// stripPreservedOpenCodeReviewExecutionContract quita de un prompt de
// orquestador de OpenCode preservado el bloque «Review Execution Contract»
// que dejó una instalación antigua, tanto el delimitado por los marcadores de
// migración como el que solo conserva la cabecera. Sin bloque devuelve el
// prompt tal cual.
func stripPreservedOpenCodeReviewExecutionContract(prompt string) string {
	const (
		startMarker = "<!-- gentle-ai:review-execution-contract-migration -->"
		endMarker   = "<!-- /gentle-ai:review-execution-contract-migration -->"
		heading     = "#### Review Execution Contract"
	)
	if markerStart := strings.Index(prompt, startMarker); markerStart >= 0 {
		if relativeEnd := strings.Index(prompt[markerStart:], endMarker); relativeEnd >= 0 {
			markerEnd := markerStart + relativeEnd + len(endMarker)
			return joinPreservedPrompt(prompt, markerStart, markerEnd)
		}
	}
	headingStart := strings.Index(prompt, heading)
	if headingStart < 0 {
		return prompt
	}
	headingEnd := len(prompt)
	remainder := prompt[headingStart+len(heading):]
	for _, candidate := range []string{"\n#### Cost and Context Balance", "\n#### ", "\n## ", "\n# "} {
		if relativeEnd := strings.Index(remainder, candidate); relativeEnd >= 0 {
			candidateEnd := headingStart + len(heading) + relativeEnd + 1
			if candidateEnd < headingEnd {
				headingEnd = candidateEnd
			}
		}
	}
	return joinPreservedPrompt(prompt, headingStart, headingEnd)
}

func joinPreservedPrompt(prompt string, start, end int) string {
	prefix := strings.TrimRight(prompt[:start], "\n")
	suffix := strings.TrimLeft(prompt[end:], "\n")
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix + "\n"
	}
	return prefix + "\n\n" + suffix
}

// suspendedSharedSkillContent adapta el contenido de un fichero compartido de
// skills mientras RDD está suspendido (C-01-c): quita de las secciones del
// orquestador el bloque del contrato de ejecución de revisión, que ningún
// orquestador renderiza. El resto de ficheros no cambia.
func suspendedSharedSkillContent(fileName string, content []byte) []byte {
	if rddharness.Enabled() || fileName != "sdd-orchestrator-sections.md" {
		return content
	}
	const (
		startMarker = "<!-- sdd-orchestrator-section:Review Execution Contract:start -->"
		endMarker   = "<!-- sdd-orchestrator-section:Review Execution Contract:end -->"
	)
	text := string(content)
	start := strings.Index(text, startMarker)
	if start < 0 {
		return content
	}
	relativeEnd := strings.Index(text[start:], endMarker)
	if relativeEnd < 0 {
		return content
	}
	end := start + relativeEnd + len(endMarker)
	return []byte(text[:start] + strings.TrimLeft(text[end:], "\r\n"))
}
