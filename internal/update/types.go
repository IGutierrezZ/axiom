package update

import "strings"

// SelfToolName is the registry name of the primary CLI product (this binary).
// Every upgrade safeguard MUST route through IsSelfToolName instead of comparing
// against a literal (RF-UPD-03).
const SelfToolName = "axiom"

// IsSelfToolName reports whether name identifies the primary CLI product.
// Matching is case- and edge-whitespace-insensitive.
func IsSelfToolName(name string) bool {
	return strings.ToLower(strings.TrimSpace(name)) == SelfToolName
}

// IsSelfTool reports whether tool is the primary CLI product (D-01).
func IsSelfTool(tool ToolInfo) bool { return IsSelfToolName(tool.Name) }

// UpdateStatus represents the outcome of a single tool version check.
type UpdateStatus string

const (
	UpToDate        UpdateStatus = "up-to-date"
	UpdateAvailable UpdateStatus = "update-available"
	NotInstalled    UpdateStatus = "not-installed"
	// RegisteredNotMaterialized means an OpenCode community plugin is listed in
	// ~/.config/opencode/tui.json, but OpenCode has not yet materialized it under
	// ~/.config/opencode/node_modules/<pkg>/package.json.
	RegisteredNotMaterialized UpdateStatus = "registered-not-materialized"
	VersionUnknown            UpdateStatus = "version-unknown"
	CheckFailed               UpdateStatus = "check-failed"
	// DevBuild is used when the installed version is the sentinel "dev" string,
	// indicating a source-built binary. Such builds are not auto-targeted for upgrade.
	DevBuild UpdateStatus = "dev-build"
)

// InstallMethod describes how a managed tool is installed on the current platform.
// Used by the upgrade executor to choose the correct upgrade strategy.
type InstallMethod string

const (
	InstallBrew      InstallMethod = "brew"
	InstallGoInstall InstallMethod = "go-install"
	InstallBinary    InstallMethod = "binary"
	// InstallScript downloads and executes the project's install.sh via pipe.
	// Used for tools that distribute via shell scripts rather than pre-built binaries.
	InstallScript InstallMethod = "script"
	// InstallOpenCodePlugin is a manual upgrade method: Axiom registers the
	// package in tui.json, and OpenCode owns package resolution on restart/reload.
	InstallOpenCodePlugin InstallMethod = "opencode-plugin"
	// InstallSourceBuild compiles the tool from a controlled source clone
	// (git clone of the exact tag, then go build). Used as the resilient
	// Windows path when `go install` is not resolvable (RF-UPD-01, D-02).
	InstallSourceBuild InstallMethod = "source-build"
)

// ToolInfo describes a managed tool that can be checked for updates.
type ToolInfo struct {
	Name              string        // human-readable name (e.g., "axiom")
	Owner             string        // GitHub repository owner
	Repo              string        // GitHub repository name
	DetectCmd         []string      // command to detect installed version; nil = use build var
	VersionPrefix     string        // prefix to strip from version output (e.g., "v")
	ReleaseTagPattern string        // optional regexp for selecting the correct GitHub release channel
	InstallMethod     InstallMethod // how this tool is installed (used by upgrade executor)
	GoImportPath      string        // for go-install tools (e.g. "github.com/.../cmd/engram")
	NpmPackage        string        // for OpenCode community plugins installed in ~/.config/opencode/node_modules

	// GoModulePath is the `module` directive the published source actually
	// declares. It is the single source of truth for whether `go install
	// <GoImportPath>@ver` is resolvable by the toolchain (RF-UPD-02).
	GoModulePath string

	// FallbackPaths returns a list of absolute paths to check when exec.LookPath
	// fails. This covers the Windows scenario where AddToUserPath updates the
	// registry but the running process PATH is stale after install. When a path
	// is found on disk, detectInstalledVersion runs the detect command using that
	// full path rather than the bare binary name.
	//
	// The function receives the user home directory and the value of LOCALAPPDATA
	// (empty on non-Windows). May be nil when no fallback is needed.
	FallbackPaths func(homeDir, localAppData string) []string
}

// GoInstallResolvable reports whether a `go install` naming GoImportPath can be
// resolved against the declared module path. It reads ONLY GoModulePath and
// GoImportPath; it never derives the module from Owner/Repo (D-01). For the
// fork this is false while go.mod declares the upstream module, and the system
// MUST NOT emit or run any `go install` for it (RF-UPD-02).
func (t ToolInfo) GoInstallResolvable() bool {
	m := strings.TrimSpace(t.GoModulePath)
	p := strings.TrimSpace(t.GoImportPath)
	if m == "" || p == "" {
		return false
	}
	if p == m {
		return true
	}
	if !strings.HasPrefix(p, m+"/") {
		return false
	}
	// A major-version suffix (v2, v3, ...) immediately after the declared
	// module makes `go install` resolve a DIFFERENT module (m + "/" + suffix).
	// `.../axiom` therefore does not make `.../axiom/v3/...` resolvable.
	rest := p[len(m)+1:]
	seg, _, _ := strings.Cut(rest, "/")
	return !isGoMajorVersionSuffix(seg)
}

// isGoMajorVersionSuffix reports whether seg is a Go major-version module
// suffix (v2, v3, ...). v0 and v1 are un-suffixed and never match.
func isGoMajorVersionSuffix(seg string) bool {
	if len(seg) < 2 || seg[0] != 'v' {
		return false
	}
	for _, c := range seg[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return seg != "v0" && seg != "v1"
}

// UpdateResult holds the result of checking a single tool for updates.
type UpdateResult struct {
	Tool             ToolInfo
	InstalledVersion string
	LatestVersion    string
	Status           UpdateStatus
	ReleaseURL       string
	UpdateHint       string
	Err              error
}

// SelectSelfToolResult returns the update result of the self-tool (this binary),
// or nil when results holds none. It matches through IsSelfTool so the lookup
// follows the registry name instead of a literal: a literal that drifted from the
// registry once left the start-up self-update without its own tool.
func SelectSelfToolResult(results []UpdateResult) *UpdateResult {
	for i := range results {
		if IsSelfTool(results[i].Tool) {
			return &results[i]
		}
	}
	return nil
}
