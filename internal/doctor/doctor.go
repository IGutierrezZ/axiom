// Package doctor defines presentation-independent health check results and execution.
package doctor

import "context"

// CheckID is a stable identifier for a health check.
type CheckID string

const (
	CheckStateJSON             CheckID = "state:json"
	CheckInstalledAssetVersion CheckID = "installed:asset_version"
	CheckEngramReachable       CheckID = "engram:reachable"
	CheckDiskSpace             CheckID = "disk:space"
	CheckProjectAssets         CheckID = "project:assets"
	CheckProjectAdapters       CheckID = "project:adapters"
	CheckProjectWorkspace      CheckID = "project:workspace"
	CheckProjectTopology       CheckID = "project:topology"
	CheckProjectRoleRepos      CheckID = "project:role_repos"
	CheckProjectSpecsRepo      CheckID = "project:specs_repo"
)

// ToolCheckID returns the stable check identifier for a tool binary.
func ToolCheckID(tool string) CheckID { return CheckID("tool:" + tool) }

// Status is the outcome of a health check.
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// RemedyID identifies a remedy that a future executor may implement.
type RemedyID string

const (
	RemedyInstallTool      RemedyID = "install-tool"
	RemedyRemoveDuplicates RemedyID = "remove-duplicate-tools"
	RemedyReorderPath      RemedyID = "reorder-path"
	RemedyInstall          RemedyID = "install"
	RemedyRepairState      RemedyID = "repair-state"
	RemedySync             RemedyID = "sync"
	RemedyStartEngram      RemedyID = "start-engram"
	RemedyInspectEngram    RemedyID = "inspect-engram"
	RemedyFreeDiskSpace    RemedyID = "free-disk-space"
	RemedyFixWorkspace     RemedyID = "fix-workspace"
)

// RemedyCategory groups remedies by the resource they concern.
type RemedyCategory string

const (
	RemedyCategoryInstall       RemedyCategory = "install"
	RemedyCategoryConfiguration RemedyCategory = "configuration"
	RemedyCategoryEnvironment   RemedyCategory = "environment"
	RemedyCategoryService       RemedyCategory = "service"
	RemedyCategoryStorage       RemedyCategory = "storage"
)

// ActionMode describes the interaction required before a remedy may run.
type ActionMode string

const (
	ActionAutomatic    ActionMode = "automatic"
	ActionConfirmation ActionMode = "confirmation"
	ActionManualOnly   ActionMode = "manual-only"
)

// Platform identifies an operating environment supported by a remedy.
type Platform string

const (
	PlatformWindows Platform = "windows"
	PlatformMacOS   Platform = "macos"
	PlatformLinux   Platform = "linux"
	PlatformWSL     Platform = "wsl"
)

// Remedy describes a possible response without performing it.
type Remedy struct {
	ID                 RemedyID       `json:"id"`
	Description        string         `json:"description"`
	Category           RemedyCategory `json:"category"`
	ActionMode         ActionMode     `json:"action_mode"`
	Eligible           bool           `json:"eligible"`
	EligibilityReason  string         `json:"eligibility_reason,omitempty"`
	SupportedPlatforms []Platform     `json:"supported_platforms,omitempty"`
}

// NewRemedy classifies a remedy without making it executable.
func NewRemedy(id RemedyID, description string) *Remedy {
	r := &Remedy{ID: id, Description: description, ActionMode: ActionManualOnly, SupportedPlatforms: supportedPlatforms()}
	switch id {
	case RemedyInstallTool, RemedyInstall:
		r.Category, r.EligibilityReason = RemedyCategoryInstall, "no bounded managed install was identified"
	case RemedyRemoveDuplicates:
		r.Category, r.EligibilityReason = RemedyCategoryEnvironment, "binary ownership is unknown"
	case RemedyReorderPath:
		r.Category, r.EligibilityReason = RemedyCategoryEnvironment, "PATH order is user-owned"
	case RemedyRepairState:
		r.Category, r.EligibilityReason = RemedyCategoryConfiguration, "no safe recovery source was identified"
	case RemedySync:
		r.Category, r.ActionMode, r.Eligible = RemedyCategoryConfiguration, ActionConfirmation, true
		r.EligibilityReason = "requires managed state and existing sync safeguards"
	case RemedyStartEngram, RemedyInspectEngram:
		r.Category, r.EligibilityReason = RemedyCategoryService, "external process and log ownership is unknown"
	case RemedyFixWorkspace:
		r.Category, r.EligibilityReason = RemedyCategoryConfiguration, "the workspace layout is user-owned"
	case RemedyFreeDiskSpace:
		r.Category, r.EligibilityReason = RemedyCategoryStorage, "disk cleanup is user-owned"
	default:
		r.EligibilityReason = "remedy is not allowlisted"
	}
	return r
}

func supportedPlatforms() []Platform {
	return []Platform{PlatformWindows, PlatformMacOS, PlatformLinux, PlatformWSL}
}

// Scope tells whether a check looks at the machine or at one project.
type Scope string

const (
	// ScopeGlobal groups the checks that do not depend on the open project:
	// tools, Engram, disk and the user-level state.
	ScopeGlobal Scope = "global"
	// ScopeProject groups the checks that read the project root.
	ScopeProject Scope = "project"
)

// Options say what a doctor run looks at, so that it never depends on the
// working directory of the process.
type Options struct {
	// Home is the user's home directory.
	Home string
	// Root is the project root; empty means "no project".
	Root string
}

// Result contains evidence produced by one check.
type Result struct {
	Name   CheckID `json:"name"`
	Scope  Scope   `json:"scope"`
	Status Status  `json:"status"`
	Detail string  `json:"detail"`
	Remedy *Remedy `json:"remedy,omitempty"`
}

// Report preserves checks in execution order.
type Report struct {
	Checks []Result `json:"checks"`
}

// Failed reports whether any check ended in StatusFail.
func (r Report) Failed() bool {
	for _, c := range r.Checks {
		if c.Status == StatusFail {
			return true
		}
	}
	return false
}

// Check is a read-only diagnostic operation. An empty Scope means ScopeGlobal.
type Check struct {
	ID    CheckID
	Scope Scope
	Run   func(context.Context) Result
}

// Runner executes checks deterministically in declaration order.
type Runner struct {
	Checks []Check
}

// Run executes every check in declaration order and isolates panics per check.
func (r Runner) Run(ctx context.Context) Report {
	report := Report{Checks: make([]Result, 0, len(r.Checks))}
	for _, check := range r.Checks {
		result := runCheck(ctx, check)
		result.Name = check.ID
		result.Scope = check.Scope
		if result.Scope == "" {
			result.Scope = ScopeGlobal
		}
		report.Checks = append(report.Checks, result)
	}
	return report
}

func runCheck(ctx context.Context, check Check) (result Result) {
	defer func() {
		if recover() != nil {
			result = Result{Status: StatusFail, Detail: "check failed unexpectedly"}
		}
	}()
	return check.Run(ctx)
}
