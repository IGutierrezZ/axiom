package persona

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

type InjectionResult struct {
	Changed bool
	Files   []string
}

// PiPersonaConfigPath returns the managed Pi persona state path beneath rootDir.
// Pi resolves this file relative to the active agent configuration root, which
// is the user's home for global install and the workspace for workspace install.
func PiPersonaConfigPath(rootDir string) string {
	return filepath.Join(rootDir, ".pi", "gentle-ai", "persona.json")
}

// InjectPiPersona writes the small runtime config consumed by gentle-pi. El
// modo es siempre el de Axiom.
func InjectPiPersona(rootDir string) (InjectionResult, error) {
	if strings.TrimSpace(rootDir) == "" {
		return InjectionResult{}, fmt.Errorf("Pi persona root must not be empty")
	}
	mode := "axiom"
	content, err := json.MarshalIndent(struct {
		Mode string `json:"mode"`
	}{Mode: mode}, "", "  ")
	if err != nil {
		return InjectionResult{}, fmt.Errorf("encode Pi persona config: %w", err)
	}
	content = append(content, '\n')

	path := PiPersonaConfigPath(rootDir)
	writeResult, err := filemerge.WriteFileAtomic(path, content, 0o644)
	if err != nil {
		return InjectionResult{}, err
	}
	return InjectionResult{Changed: writeResult.Changed, Files: []string{path}}, nil
}

// outputStyleOverlayJSON is the settings.json overlay to enable the selected
// managed Claude Code output style.
func outputStyleOverlayJSON(name string) []byte {
	return []byte(fmt.Sprintf("{\n  \"outputStyle\": %q\n}\n", name))
}

// Inject applies the Axiom persona: the marker-bound markdown block, the
// retirement of a residual OpenCode `agent.gentleman` key and the Claude Code
// output-style overlay. Used by `axiom install`.
func Inject(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	return inject(homeDir, adapter)
}

// InjectForSync regenerates the persona assets `axiom sync` manages. Install and
// sync apply the same persona, so it delegates to Inject.
func InjectForSync(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	return inject(homeDir, adapter)
}

func inject(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	if !adapter.SupportsSystemPrompt() {
		return InjectionResult{}, nil
	}

	files := make([]string, 0, 3)
	changed := false

	content := personaContent(adapter.Agent(), residualChannel(adapter))
	if content == "" {
		return InjectionResult{}, nil
	}

	// 1. Inject persona content based on system prompt strategy.

	switch adapter.SystemPromptStrategy() {
	case model.StrategyMarkdownSections:
		promptPath := adapter.SystemPromptFile(homeDir)
		existing, err := readFileOrEmpty(promptPath)
		if err != nil {
			return InjectionResult{}, err
		}

		// Strip legacy Agent Teams Lite block (standalone ATL installer leftover).
		healed := filemerge.StripLegacyATLBlock(existing)

		updated := filemerge.InjectMarkdownSection(healed, filemerge.PersonaSectionID, content)

		writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(updated), 0o644)
		if err != nil {
			return InjectionResult{}, err
		}
		changed = changed || writeResult.Changed
		files = append(files, promptPath)

	case model.StrategyFileReplace:
		promptPath := adapter.SystemPromptFile(homeDir)

		if adapter.Agent() == model.AgentOpenCode {
			existing, err := readFileOrEmpty(promptPath)
			if err != nil {
				return InjectionResult{}, err
			}

			healed := filemerge.StripLegacyATLBlock(existing)
			updated := filemerge.InjectMarkdownSection(healed, filemerge.PersonaSectionID, content)

			writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(updated), 0o644)
			if err != nil {
				return InjectionResult{}, err
			}
			changed = changed || writeResult.Changed
			files = append(files, promptPath)
			break
		}

		// Escribir solo el contenido de la persona destruiría las secciones de
		// SDD/Engram que se inyectan después en el pipeline. Si el fichero ya existe
		// y tiene secciones gestionadas, se conservan y solo se sustituye la parte
		// de la persona al principio.
		existing, readErr := readFileOrEmpty(promptPath)
		if readErr != nil {
			return InjectionResult{}, readErr
		}

		if preserved, ok := preserveManagedSections(existing, content); ok {
			writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(preserved), 0o644)
			if err != nil {
				return InjectionResult{}, err
			}
			changed = changed || writeResult.Changed
			files = append(files, promptPath)
			break
		}

		writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(content), 0o644)
		if err != nil {
			return InjectionResult{}, err
		}
		changed = changed || writeResult.Changed
		files = append(files, promptPath)

	case model.StrategyInstructionsFile:
		promptPath := adapter.SystemPromptFile(homeDir)

		// Se conservan las secciones gestionadas (misma lógica que en
		// StrategyFileReplace).
		existing, readErr := readFileOrEmpty(promptPath)
		if readErr != nil {
			return InjectionResult{}, readErr
		}

		if preserved, ok := preserveManagedSections(existing, wrapInstructionsFile(content)); ok {
			writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(preserved), 0o644)
			if err != nil {
				return InjectionResult{}, err
			}
			changed = changed || writeResult.Changed
			files = append(files, promptPath)
			break
		}

		// Write the new instructions file (with YAML frontmatter) to the current path.
		// WriteFileAtomic compares bytes, so it is naturally idempotent: it rewrites
		// whenever the on-disk content differs from instructionsContent, which covers
		// the case where an older install wrote persona content without frontmatter.
		instructionsContent := wrapInstructionsFile(content)
		writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(instructionsContent), 0o644)
		if err != nil {
			return InjectionResult{}, err
		}
		changed = changed || writeResult.Changed
		files = append(files, promptPath)

	case model.StrategyAppendToFile:
		promptPath := adapter.SystemPromptFile(homeDir)

		existing, err := readFileOrEmpty(promptPath)
		if err != nil {
			return InjectionResult{}, err
		}

		// Append-style agents still need marker-bound persona sections so sync can
		// replace managed content without duplicating it or disturbing user-authored
		// rules in the shared prompt file.
		healed := filemerge.StripLegacyATLBlock(existing)
		updated := filemerge.InjectMarkdownSection(healed, filemerge.PersonaSectionID, content)

		writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(updated), 0o644)
		if err != nil {
			return InjectionResult{}, err
		}
		changed = changed || writeResult.Changed
		files = append(files, promptPath)

	}

	// 2. OpenCode: la persona de Axiom no define agente propio, así que se retira
	// el agent.gentleman residual de una instalación anterior. Solo se borra la
	// subclave "gentleman": el resto de agentes del usuario se conservan.
	if adapter.Agent() == model.AgentOpenCode {
		settingsPath := adapter.SettingsPath(homeDir)
		if settingsPath != "" {
			removed, err := removeJSONNestedSubKey(settingsPath, "agent", "gentleman")
			if err != nil {
				return InjectionResult{}, fmt.Errorf("clean agent.gentleman from settings: %w", err)
			}
			if removed {
				changed = true
				files = append(files, settingsPath)
			}
		}
	}

	// 3. Write the managed output style. The backup/verify callers derive their
	// paths from the same plan.
	plan := ResourcePlanFor()
	style := plan.OutputStyle()
	if adapter.SupportsOutputStyles() {
		stylePaths := plan.OutputStylePaths(adapter.OutputStyleDir(homeDir))
		if stylePaths.Write != "" {
			styleResult, err := filemerge.WriteFileAtomic(stylePaths.Write, []byte(assets.MustRead(style.AssetPath)), 0o644)
			if err != nil {
				return InjectionResult{}, err
			}
			changed = changed || styleResult.Changed
			files = append(files, stylePaths.Write)
		}

		settingsPath := adapter.SettingsPath(homeDir)
		if settingsPath != "" {
			settingsResult, err := mergeJSONFile(settingsPath, outputStyleOverlayJSON(style.Name))
			if err != nil {
				return InjectionResult{}, err
			}
			changed = changed || settingsResult.Changed
			files = append(files, settingsPath)
		}
	}

	return InjectionResult{Changed: changed, Files: files}, nil
}

// residualChannel reports whether the adapter already delivers tone/language/
// philosophy content through an active output-style channel that loads every
// session, making the system-prompt persona section redundant for that
// content. Claude Code is gated on SupportsOutputStyles().
func residualChannel(adapter agents.Adapter) bool {
	return adapter.SupportsOutputStyles()
}

// personaContent returns the Axiom persona asset for the given agent.
func personaContent(agent model.AgentID, residualContentAvailable bool) string {
	if residualContentAvailable {
		switch agent {
		case model.AgentClaudeCode:
			return assets.MustRead("claude/persona-axiom.md")
		}
	}
	switch agent {
	case model.AgentOpenCode:
		return assets.MustRead("opencode/persona-axiom.md")
	default:
		return assets.MustRead("generic/persona-axiom.md")
	}
}

func mergeJSONFile(path string, overlay []byte) (filemerge.WriteResult, error) {
	baseJSON, err := osReadFile(path)
	if err != nil {
		return filemerge.WriteResult{}, err
	}
	merged, err := filemerge.MergeJSONObjectsForPath(path, baseJSON, overlay)
	if err != nil {
		return filemerge.WriteResult{}, err
	}

	return filemerge.WriteFileAtomic(path, merged, 0o644)
}

var osReadFile = func(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read json file %q: %w", path, err)
	}

	return content, nil
}

// preserveManagedSections checks whether the existing file content has
// gentle-ai managed sections (SDD orchestrator, engram protocol, etc.) and
// returns new content that preserves those sections while replacing only the
// persona text before them. Returns ("", false) when no preservation is needed
// (empty file or no managed markers found).
func preserveManagedSections(existing, newPersona string) (string, bool) {
	if existing == "" {
		return "", false
	}

	idxAxiom := strings.Index(existing, "<!-- axiom:")
	idxLegacy := strings.Index(existing, "<!-- gentle-ai:")
	idx := -1
	if idxAxiom >= 0 && idxLegacy >= 0 {
		if idxAxiom < idxLegacy {
			idx = idxAxiom
		} else {
			idx = idxLegacy
		}
	} else if idxAxiom >= 0 {
		idx = idxAxiom
	} else {
		idx = idxLegacy
	}
	if idx < 0 {
		return "", false
	}

	managedSuffix := existing[idx:]
	updated := newPersona
	if !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	if idx > 0 {
		// There was persona content before the markers — add a blank line separator.
		updated += "\n"
	}
	updated += managedSuffix

	return updated, true
}

func readFileOrEmpty(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read file %q: %w", path, err)
	}
	return string(data), nil
}

func wrapInstructionsFile(content string) string {
	frontmatter := "---\n" +
		"name: Axiom Persona\n" +
		"description: Teaching-oriented persona with SDD orchestration and Engram protocol\n" +
		"applyTo: \"**\"\n" +
		"---\n\n"

	return frontmatter + content
}

// removeJSONNestedSubKey reads the JSON object at path and removes subKey from
// within the top-level parentKey object. Only the named subKey is deleted —
// sibling keys inside parentKey are preserved. If the file does not exist, the
// parentKey is absent, or subKey is not present, it is a no-op and returns false.
func removeJSONNestedSubKey(path, parentKey, subKey string) (bool, error) {
	raw, err := osReadFile(path)
	if err != nil {
		return false, err
	}
	if len(raw) == 0 {
		return false, nil
	}

	root := map[string]any{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return false, nil
	}

	parent, ok := root[parentKey]
	if !ok {
		return false, nil
	}
	parentMap, ok := parent.(map[string]any)
	if !ok {
		return false, nil
	}
	if _, exists := parentMap[subKey]; !exists {
		return false, nil
	}

	delete(parentMap, subKey)
	if len(parentMap) == 0 {
		delete(root, parentKey)
	} else {
		root[parentKey] = parentMap
	}

	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal settings after cleanup: %w", err)
	}
	encoded = append(encoded, '\n')

	if _, err := filemerge.WriteFileAtomic(path, encoded, 0o644); err != nil {
		return false, err
	}
	return true, nil
}
