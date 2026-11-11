package permissions

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

type InjectionResult struct {
	Changed bool
	Files   []string
}

// TargetPath returns the file path that permission injection creates or updates
// for the adapter under rootDir, or an empty string when the agent has no
// supported permission injection target. rootDir is the install root chosen by
// the caller for the active scope (the user's home for a global install, the
// workspace root for a workspace install); this package never picks it.
// Codex has no target: Axiom relies on Codex's built-in default permissions and
// does not write its permissions config at all.
//
// The overlays only carry protective rules. Axiom never writes an
// auto-approval mode (Claude bypassPermissions, Gemini/Qwen auto_edit, VS Code
// chat.tools.autoApprove, an OpenCode/Kilo catch-all allow) and never removes
// one the user already has, so agents whose only overlay was such a mode have
// no target at all.
func TargetPath(rootDir string, adapter agents.Adapter) string {
	if agentOverlay(adapter.Agent()) == nil {
		return ""
	}
	return adapter.SettingsPath(rootDir)
}

// claudeCodeOverlayJSON declares Claude Code's protective deny rules only. It
// sets no defaultMode: the user's own mode is left untouched.
var claudeCodeOverlayJSON = []byte(`{
  "permissions": {
    "deny": [
      "Bash(rm -rf /)",
      "Bash(sudo rm -rf /)",
      "Bash(rm -rf ~)",
      "Bash(sudo rm -rf ~)",
      "Read(.env)",
      "Read(.env.*)",
      "Edit(.env)",
      "Edit(.env.*)",
      "Read(~/.ssh/**)",
      "Edit(~/.ssh/**)",
      "Read(~/.credentials/**)",
      "Edit(~/.credentials/**)",
      "Read(~/Library/Keychains/**)",
      "Edit(~/Library/Keychains/**)",
      "Read(~/.aws/credentials)",
      "Edit(~/.aws/credentials)",
      "Read(~/.config/gh/hosts.yml)",
      "Edit(~/.config/gh/hosts.yml)",
      "Read(**/*.pem)",
      "Edit(**/*.pem)",
      "Read(**/*.key)",
      "Edit(**/*.key)",
      "Read(**/secrets/*)",
      "Edit(**/secrets/*)"
    ]
  }
}
`)

// openCodeOverlayJSON uses the OpenCode "permission" key with bash/read
// granularity. It declares only "ask" and "deny" rules: no catch-all "allow" is
// written, so the user's own default for bash and read stays in force.
var openCodeOverlayJSON = []byte(`{
  "permission": {
    "bash": {
      "git commit *": "ask",
      "git push *": "ask",
      "git push": "ask",
      "git push --force *": "ask",
      "git rebase *": "ask",
      "git reset --hard *": "ask",
      "ssh": "ask",
      "ssh *": "ask",
      "scp": "ask",
      "scp *": "ask",
      "sftp": "ask",
      "sftp *": "ask",
      "rsync": "ask",
      "rsync *": "ask"
    },
    "read": {
      "*.env": "deny",
      "*.env.*": "deny",
      "**/.env": "deny",
      "**/.env.*": "deny",
      "**/secrets/**": "deny",
      "**/credentials.json": "deny",
      "**/.ssh/**": "deny",
      "**/.credentials/**": "deny",
      "**/Library/Keychains/**": "deny",
      "**/.aws/credentials": "deny",
      "**/.config/gh/hosts.yml": "deny",
      "**/*.pem": "deny",
      "**/*.key": "deny"
    }
  }
}
`)

// unionPaths lists, per agent, the overlay arrays that are merged as a union
// with the user's own array instead of replacing it (mergeObjects replaces
// arrays). Only these declared paths get that treatment.
var unionPaths = map[model.AgentID][][]string{
	model.AgentClaudeCode: {{"permissions", "deny"}},
}

// Overlay devuelve una copia del JSON de reglas protectoras que Axiom fusiona en
// la configuración del agente, o nil si no escribe ninguna. Lo usa la
// desinstalación para retirar exactamente esas entradas y ninguna otra: ni el
// resto de la lista deny del usuario ni modos que Axiom ya no impone.
func Overlay(id model.AgentID) []byte {
	overlay := agentOverlay(id)
	if overlay == nil {
		return nil
	}
	return append([]byte(nil), overlay...)
}

// agentOverlay returns the correct permission overlay for the given agent,
// or nil if the agent has nothing protective to inject.
func agentOverlay(id model.AgentID) []byte {
	switch id {
	case model.AgentClaudeCode:
		return claudeCodeOverlayJSON
	case model.AgentOpenCode, model.AgentKilocode:
		return openCodeOverlayJSON
	case model.AgentGeminiCLI, model.AgentQwenCode, model.AgentVSCodeCopilot:
		// Their only overlay was an auto-approval mode (auto_edit,
		// chat.tools.autoApprove). Axiom does not impose those modes, and a
		// value the user already set is never touched, so nothing is injected.
		return nil
	case model.AgentGitHubCopilotCLI:
		// Copilot CLI gestiona los permisos por herramienta con sus propias opciones
		// y con lo que el usuario apruebe en la sesión. Axiom no tiene un formato de
		// reglas que escribir en su settings.json, así que no inyecta nada: lo que el
		// usuario haya decidido queda intacto.
		return nil
	case model.AgentAntigravity:
		// Antigravity manages permissions via IDE UI (Artifact Review Policy /
		// Terminal Command Auto Execution). No injectable settings.json schema.
		return nil
	case model.AgentCursor:
		// Cursor manages permissions via cli-config.json, not settings.json.
		return nil
	case model.AgentCodex:
		// Codex relies on its built-in default permissions. gentle-ai writes
		// nothing to Codex's permissions config — not a profile, and not the
		// legacy cleanup that used to strip one. Codex refuses to load a config
		// that defines a [permissions.*] profile without default_permissions,
		// so a cleanup removing the pointer while a user profile survived left
		// Codex unable to start (#1794). An old gentle-dev profile stays until
		// its owner removes it.
		return nil
	case model.AgentHermes:
		// Hermes permission format is undocumented — no overlay is injected (§14).
		return nil
	default:
		return nil
	}
}

// Inject merges the protective overlay of the adapter into its settings file
// under rootDir. The caller resolves rootDir for the install scope; an agent
// that cannot hold permissions under that root must not be passed at all, so a
// workspace install never falls back to the user's home.
func Inject(rootDir string, adapter agents.Adapter) (InjectionResult, error) {
	settingsPath := TargetPath(rootDir, adapter)
	if settingsPath == "" {
		return InjectionResult{}, nil
	}

	overlay := agentOverlay(adapter.Agent())
	if overlay == nil {
		return InjectionResult{}, nil
	}

	defaults := adapter.Agent() == model.AgentOpenCode || adapter.Agent() == model.AgentKilocode
	writeResult, err := mergeJSONFile(settingsPath, overlay, defaults, unionPaths[adapter.Agent()])
	if err != nil {
		return InjectionResult{}, err
	}

	return InjectionResult{Changed: writeResult.Changed, Files: []string{settingsPath}}, nil
}

func mergeJSONFile(path string, overlay []byte, defaults bool, union [][]string) (filemerge.WriteResult, error) {
	baseJSON, err := osReadFile(path)
	if err != nil {
		return filemerge.WriteResult{}, err
	}

	if len(union) > 0 {
		// La unión sustituye la lista deny del usuario: se reescriben antes sus
		// reglas antiguas de Axiom para que no convivan con las ancladas al HOME.
		overlay, err = filemerge.UnionJSONArrays(withAnchoredClaudeCodeDeny(baseJSON), overlay, union...)
		if err != nil {
			return filemerge.WriteResult{}, err
		}
	}

	merge := filemerge.MergeJSONObjectsForPath
	if defaults {
		// Solo OpenCode/Kilo Code: el overlay no relaja el valor por defecto del usuario.
		overlay = withoutRulesLooserThanUserDefault(baseJSON, overlay)
		merge = filemerge.MergeJSONDefaultsForPath
	}
	merged, err := merge(path, baseJSON, overlay)
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

// staleClaudeCodeDenyRules asocia las reglas relativas al proyecto que escribían
// versiones anteriores de Axiom con su equivalente anclado al HOME. Claude Code
// resuelve una ruta sin "~/" respecto al proyecto, así que aquellas reglas no
// protegían los secretos del HOME.
var staleClaudeCodeDenyRules = map[string]string{
	"Read(.ssh/*)":               "Read(~/.ssh/**)",
	"Edit(.ssh/*)":               "Edit(~/.ssh/**)",
	"Read(.credentials/*)":       "Read(~/.credentials/**)",
	"Edit(.credentials/*)":       "Edit(~/.credentials/**)",
	"Read(Library/Keychains/*)":  "Read(~/Library/Keychains/**)",
	"Edit(Library/Keychains/*)":  "Edit(~/Library/Keychains/**)",
	"Read(.aws/credentials)":     "Read(~/.aws/credentials)",
	"Edit(.aws/credentials)":     "Edit(~/.aws/credentials)",
	"Read(.config/gh/hosts.yml)": "Read(~/.config/gh/hosts.yml)",
	"Edit(.config/gh/hosts.yml)": "Edit(~/.config/gh/hosts.yml)",
}

// LegacyOverlay devuelve el JSON con las reglas que versiones anteriores de
// Axiom escribían y que ya no forman parte del Overlay, o nil si el agente no
// tiene ninguna. La desinstalación las retira junto a las actuales para no dejar
// reglas propias de Axiom en instalaciones antiguas.
func LegacyOverlay(id model.AgentID) []byte {
	if id != model.AgentClaudeCode {
		return nil
	}
	stale := make([]string, 0, len(staleClaudeCodeDenyRules))
	for rule := range staleClaudeCodeDenyRules {
		stale = append(stale, rule)
	}
	sort.Strings(stale)
	encoded, err := json.Marshal(map[string]any{"permissions": map[string]any{"deny": stale}})
	if err != nil {
		return nil
	}
	return encoded
}

// withAnchoredClaudeCodeDeny devuelve base con las reglas antiguas de su lista
// permissions.deny ancladas al HOME. Solo se usa como entrada de la unión: el
// documento que se escribe sale de la fusión sobre el original. Si base no es un
// objeto JSON o no tiene la lista, la devuelve intacta.
func withAnchoredClaudeCodeDeny(base []byte) []byte {
	root, err := filemerge.UnmarshalJSONObject(base)
	if err != nil {
		return base
	}
	permissions, _ := root["permissions"].(map[string]any)
	deny, ok := permissions["deny"].([]any)
	if !ok {
		return base
	}
	permissions["deny"] = anchorStaleClaudeCodeDenyRules(deny)
	encoded, err := json.Marshal(root)
	if err != nil {
		return base
	}
	return encoded
}

// anchorStaleClaudeCodeDenyRules reescribe en su sitio cada coincidencia exacta
// de las reglas antiguas de Axiom. Si la regla anclada ya figura, descarta la
// antigua. No toca ninguna otra entrada ni su orden.
func anchorStaleClaudeCodeDenyRules(deny []any) []any {
	present := map[string]bool{}
	for _, rule := range deny {
		if name, ok := rule.(string); ok {
			present[name] = true
		}
	}
	anchored := make([]any, 0, len(deny))
	for _, rule := range deny {
		name, _ := rule.(string)
		if replacement, stale := staleClaudeCodeDenyRules[name]; stale {
			if present[replacement] {
				continue
			}
			present[replacement] = true
			rule = replacement
		}
		anchored = append(anchored, rule)
	}
	return anchored
}

// withoutRulesLooserThanUserDefault descarta del overlay las reglas de
// herramienta menos estrictas que el permission "*" raíz del usuario (deny es
// más estricto que ask, y ask que allow). Fusionada en un mapa de herramienta
// que el usuario acotó, una regla más laxa sigue a la raíz y, como última
// coincidencia, relajaría su valor por defecto. Sin "*" raíz, o con allow, el
// overlay queda intacto.
func withoutRulesLooserThanUserDefault(base, overlay []byte) []byte {
	strictness := map[string]int{"allow": 0, "ask": 1, "deny": 2}
	root, err := filemerge.UnmarshalJSONObject(base)
	if err != nil {
		return overlay
	}
	permission, _ := root["permission"].(map[string]any)
	rootAction, _ := permission["*"].(string)
	if patterns, ok := permission["*"].(map[string]any); ok {
		rootAction, _ = patterns["*"].(string)
	}
	floor, ok := strictness[rootAction]
	if !ok || floor == 0 {
		return overlay
	}
	defaults, err := filemerge.UnmarshalJSONObject(overlay)
	if err != nil {
		return overlay
	}
	rules, _ := defaults["permission"].(map[string]any)
	for _, value := range rules {
		patterns, _ := value.(map[string]any)
		for pattern, action := range patterns {
			if name, _ := action.(string); strictness[name] < floor {
				delete(patterns, pattern)
			}
		}
	}
	encoded, err := json.Marshal(defaults)
	if err != nil {
		return overlay
	}
	return encoded
}
