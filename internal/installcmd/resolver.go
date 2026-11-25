package installcmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// cmdLookPath, osStat, osGetenv, and cmdGoVersion are package-level vars for testability.
var cmdLookPath = exec.LookPath
var osStat = os.Stat
var osGetenv = os.Getenv
var cmdGoVersion = func() ([]byte, error) {
	return exec.Command("go", "version").Output()
}

// CommandSequence represents an ordered list of commands to run in sequence.
// Each inner slice is a single command with its arguments (e.g., ["brew", "install", "engram"]).
// Multi-step installs (e.g., tap + install) are expressed as multiple entries.
type CommandSequence = [][]string

type Resolver interface {
	ResolveAgentInstall(profile system.PlatformProfile, agent model.AgentID) (CommandSequence, error)
	ResolveComponentInstall(profile system.PlatformProfile, component model.ComponentID) (CommandSequence, error)
	ResolveDependencyInstall(profile system.PlatformProfile, dependency string) (CommandSequence, error)
}

type profileResolver struct{}

func NewResolver() Resolver {
	return profileResolver{}
}

func (profileResolver) ResolveAgentInstall(profile system.PlatformProfile, agent model.AgentID) (CommandSequence, error) {
	switch agent {
	case model.AgentClaudeCode:
		return resolveClaudeCodeInstall(profile), nil
	case model.AgentOpenCode:
		return resolveOpenCodeInstall(profile)
	case model.AgentGitHubCopilotCLI:
		return resolveCopilotCLIInstall(profile), nil
	default:
		return nil, fmt.Errorf("install command is not supported for agent %q", agent)
	}
}

// resolveClaudeCodeInstall returns the npm install command sequence gentle-ai
// shows for Claude Code — display text only, never executed by gentle-ai
// (see agentInstallStep in internal/cli/run.go). On Linux with system npm,
// sudo is required. With nvm/fnm/volta, it is not. On Windows and macOS,
// sudo is never needed.
//
// --ignore-scripts blocks postinstall hooks, the primary supply-chain attack
// vector for npm packages. The version advises "latest" rather than a pin:
// a pin only guarded against a tampered "latest" tag when gentle-ai itself
// ran the command unattended. Now a human reads and runs it, and a stale
// hardcoded version goes wrong the moment a newer release ships (the same
// drift this shape fixed for Codex's GPT-5.6 update advice).
func resolveClaudeCodeInstall(profile system.PlatformProfile) CommandSequence {
	const pkg = "@anthropic-ai/claude-code@latest"
	if profile.OS == "linux" && !profile.NpmWritable {
		return CommandSequence{{"sudo", "npm", "install", "-g", "--ignore-scripts", pkg}}
	}
	return CommandSequence{{"npm", "install", "-g", "--ignore-scripts", pkg}}
}

// resolveCopilotCLIInstall devuelve el comando npm que se muestra para GitHub
// Copilot CLI; es solo texto, Axiom no lo ejecuta. En Linux con npm del sistema
// hace falta sudo; con nvm/fnm/volta, no. En Windows y macOS nunca.
//
// --ignore-scripts bloquea los scripts de instalación, el vector habitual de los
// ataques a la cadena de suministro. @github/copilot no declara ninguno (sus
// binarios por plataforma llegan como optionalDependencies), así que no se pierde
// nada. La versión es "latest" porque lo lee y lo ejecuta una persona.
func resolveCopilotCLIInstall(profile system.PlatformProfile) CommandSequence {
	const pkg = "@github/copilot@latest"
	if profile.OS == "linux" && !profile.NpmWritable {
		return CommandSequence{{"sudo", "npm", "install", "-g", "--ignore-scripts", pkg}}
	}
	return CommandSequence{{"npm", "install", "-g", "--ignore-scripts", pkg}}
}

// npmBasedAgents is the set of agents whose auto-install runs npm commands.
// When any of these agents is selected, npm (and therefore Node.js) must be
// present before the pipeline reaches the agent install step.
//
// AgentPi is included because InstallCommand always runs engramInitCommand(),
// which executes either `pnpm dlx` or `npm exec` (both require Node.js). The
// npm-presence check is a sound proxy for Node.js availability.
var npmBasedAgents = map[model.AgentID]struct{}{
	model.AgentClaudeCode: {},
	model.AgentOpenCode:   {},
	model.AgentGeminiCLI:  {},
	model.AgentCodex:      {},
	model.AgentQwenCode:   {},
	model.AgentPi:         {},

	model.AgentGitHubCopilotCLI: {},
}

// ValidateAgentInstallPreflight validates agent-specific prerequisites that must
// exist before running installation commands.
func ValidateAgentInstallPreflight(profile system.PlatformProfile, agent model.AgentID) error {
	if _, ok := npmBasedAgents[agent]; ok {
		if err := validateNpmInstallPreflight(profile); err != nil {
			return err
		}
	}
	switch agent {
	case model.AgentPi:
		return validatePiInstallPreflight()
	default:
		return nil
	}
}

func validatePiInstallPreflight() error {
	if _, err := cmdLookPath("pi"); err != nil {
		return fmt.Errorf("Pi requires the `pi` executable in PATH before installing Axiom Pi packages")
	}

	return nil
}

// validateNpmInstallPreflight ensures npm (and therefore Node.js) is available
// before attempting any npm-based agent install. Called for all agents in
// npmBasedAgents so the user gets a clear, actionable error instead of a
// cryptic "exec: npm: executable file not found in PATH" mid-pipeline.
func validateNpmInstallPreflight(profile system.PlatformProfile) error {
	if _, err := cmdLookPath("npm"); err != nil {
		hint := system.InstallHintForDep("node", profile)
		msg := fmt.Sprintf(
			"Node.js / npm is required but `npm` was not found in PATH.\n"+
				"Install Node.js (npm is included) and retry:\n"+
				"  %s",
			hint,
		)
		if profile.PackageManager == "rpm-ostree" {
			msg += "\nIf a pending deployment prevents --apply-live, stage without it and reboot:\n  rpm-ostree install -y nodejs npm && systemctl reboot"
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (profileResolver) ResolveComponentInstall(profile system.PlatformProfile, component model.ComponentID) (CommandSequence, error) {
	switch component {
	case model.ComponentEngram:
		return resolveEngramInstall(profile)
	default:
		return nil, fmt.Errorf("install command is not supported for component %q", component)
	}
}

func (profileResolver) ResolveDependencyInstall(profile system.PlatformProfile, dependency string) (CommandSequence, error) {
	if dependency == "" {
		return nil, fmt.Errorf("dependency name is required")
	}

	switch profile.PackageManager {
	case "brew":
		return CommandSequence{{"brew", "install", dependency}}, nil
	case "apt":
		return CommandSequence{{"sudo", "apt-get", "install", "-y", dependency}}, nil
	case "pacman":
		return CommandSequence{{"sudo", "pacman", "-S", "--noconfirm", dependency}}, nil
	case "dnf":
		return CommandSequence{{"sudo", "dnf", "install", "-y", dependency}}, nil
	case "rpm-ostree":
		return CommandSequence{{"rpm-ostree", "install", "-y", "--apply-live", dependency}}, nil
	case "winget":
		return CommandSequence{{"winget", "install", "--id", dependency, "-e", "--accept-source-agreements", "--accept-package-agreements"}}, nil
	default:
		return nil, fmt.Errorf(
			"unsupported package manager %q for os=%q distro=%q",
			profile.PackageManager,
			profile.OS,
			profile.LinuxDistro,
		)
	}
}

// resolveOpenCodeInstall is display-only advice for new V2 installations.
// Released @opencode/cli needs its postinstall binary setup: never add
// --ignore-scripts. Existing V1 installations are not upgraded by this advice.
// Source: https://opencode.ai/v2/docs/migrate-v1/
func resolveOpenCodeInstall(profile system.PlatformProfile) (CommandSequence, error) {
	const pkg = "@opencode/cli@latest"
	switch profile.PackageManager {
	case "brew":
		return CommandSequence{
			{"npm", "install", "-g", pkg},
		}, nil
	case "winget":
		// On Windows, npm global installs do not require sudo.
		return CommandSequence{{"npm", "install", "-g", pkg}}, nil
	default:
		// Any package manager the system probe accepted is enough here: the
		// install runs through npm, never through the manager itself, so
		// re-enumerating managers would silently narrow the probe's list
		// (issue #2499). The gate keeps a probe-rejected Linux profile
		// (empty PackageManager) on the unsupported arm.
		if profile.OS == "linux" && profile.PackageManager != "" {
			if profile.NpmWritable {
				return CommandSequence{{"npm", "install", "-g", pkg}}, nil
			}
			return CommandSequence{{"sudo", "npm", "install", "-g", pkg}}, nil
		}
		return nil, fmt.Errorf(
			"unsupported platform for opencode: os=%q distro=%q pm=%q",
			profile.OS, profile.LinuxDistro, profile.PackageManager,
		)
	}
}

// validateGoForModuleInstall checks that Go ≥1.24 is installed and GO111MODULE is not
// disabled before attempting `go install`. Returns an actionable error if any check fails.
// MUST NOT be called for brew-based installs (brew manages Go transitively).
func validateGoForModuleInstall(profile system.PlatformProfile) error {
	if _, err := cmdLookPath("go"); err != nil {
		return fmt.Errorf(
			"Go 1.24+ is required to install Engram but was not found in PATH.\n" +
				"Please install Go from https://go.dev/dl/ and restart your terminal.")
	}

	out, err := cmdGoVersion()
	if err != nil {
		return fmt.Errorf(
			"Go 1.24+ is required but could not verify the installed version.\n" +
				"Please ensure Go is properly installed: https://go.dev/dl/")
	}

	// Parse "go version go1.XX.Y platform/arch"
	parts := strings.Fields(string(out))
	if len(parts) >= 3 {
		versionStr := strings.TrimPrefix(parts[2], "go")
		versionParts := strings.SplitN(versionStr, ".", 3)
		if len(versionParts) >= 2 {
			major, _ := strconv.Atoi(versionParts[0])
			minor, _ := strconv.Atoi(versionParts[1])
			if major < 1 || (major == 1 && minor < 24) {
				return fmt.Errorf(
					"Go 1.24+ is required to install Engram, but found go%s.\n"+
						"Please update Go: https://go.dev/dl/", versionStr)
			}
		}
	}

	if osGetenv("GO111MODULE") == "off" {
		fix := "export GO111MODULE=on  # then retry"
		if profile.OS == "windows" {
			fix = `$env:GO111MODULE = "on"  # PowerShell, then retry`
		}
		return fmt.Errorf("Go modules are disabled (GO111MODULE=off).\nRun: %s", fix)
	}

	return nil
}

// resolveEngramInstall returns the correct install command sequence for Engram per platform.
// - darwin (brew): brew tap + brew install (via Gentleman-Programming/homebrew-tap)
// - linux/windows: returns an error — callers must use engram.DownloadLatestBinary() instead.
//
// The go install method has been removed because it required Go 1.24+ which most
// users on Linux/Windows don't have. Pre-built binaries are available at:
// https://github.com/Gentleman-Programming/engram/releases
func resolveEngramInstall(profile system.PlatformProfile) (CommandSequence, error) {
	switch profile.PackageManager {
	case "brew":
		// macOS (or Linux with Homebrew): brew manages Go transitively — no preflight needed.
		return CommandSequence{
			{"brew", "tap", "Gentleman-Programming/homebrew-tap"},
			{"brew", "install", "engram"},
		}, nil
	default:
		return nil, fmt.Errorf(
			"engram on %q/%q uses direct binary download — use engram.DownloadLatestBinary() instead of CommandSequence",
			profile.OS, profile.PackageManager,
		)
	}
}
