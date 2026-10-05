package sdd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

// telemetryHookEvents are the only hook events earlier versions wrote the
// runtime telemetry command under.
var telemetryHookEvents = []string{"Stop", "SubagentStop"}

// retiredTelemetryCommands lists the exact commands earlier versions installed
// for agent ("claude" or "codex"), under the current and the legacy binary name.
func retiredTelemetryCommands(agent string) []string {
	return []string{
		"axiom telemetry runtime " + agent + " --json",
		"gentle-ai telemetry runtime " + agent + " --json",
	}
}

// retireRuntimeTelemetryHooks removes the retired runtime telemetry hooks for
// agent from the Stop and SubagentStop events of hooksMap. Matching is by exact
// command string only, so user hooks, even in the same event or matcher entry,
// are never touched. A matcher entry left without hooks, and an event left
// without entries, are dropped. It reports whether hooksMap changed and is a
// no-op on a second run.
func retireRuntimeTelemetryHooks(hooksMap map[string]any, agent string) bool {
	retired := retiredTelemetryCommands(agent)
	changed := false
	for _, event := range telemetryHookEvents {
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

func isRetiredTelemetryCommand(command string, retired []string) bool {
	for _, candidate := range retired {
		if command == candidate {
			return true
		}
	}
	return false
}

// retireClaudeTelemetryHooks removes the retired Claude runtime telemetry hooks
// from settingsPath. It writes only when something was removed and never
// creates the file.
func retireClaudeTelemetryHooks(settingsPath string) (bool, error) {
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
	if !ok || !retireRuntimeTelemetryHooks(hooksMap, "claude") {
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
