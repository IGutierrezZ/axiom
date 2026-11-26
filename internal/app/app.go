package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/skillregistry"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/statecoord"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/update"
	"github.com/IGutierrezZ/axiom/v3/internal/update/upgrade"
	"github.com/IGutierrezZ/axiom/v3/internal/verify"
)

// Version is set from main via ldflags at build time.
var Version = "dev"

var (
	updateCheckAll            = update.CheckAll
	updateCheckFiltered       = update.CheckFiltered
	upgradeExecute            = upgrade.Execute
	upgradeExecuteWithOptions = upgrade.ExecuteWithOptions
	selfUpdateFn              = selfUpdate
	ensureCurrentOSSupported  = system.EnsureCurrentOSSupported
	detectSystem              = system.Detect
	// deferredSyncFn is the function called when PendingSync=true is found on
	// launch. Swappable for tests; production value calls cli.RunSync directly.
	// cli.RunSync is idempotent (re-reads state + re-applies configs each call),
	// so retrying on failure (spec scenario "deferred sync fails → retry") is safe.
	deferredSyncFn = func() error {
		args := []string(nil)
		if h, err := os.UserHomeDir(); err == nil {
			args = syncArgsForDiscoveredAgents(h)
		}
		_, err := cli.RunSync(args)
		return err
	}
)

func Run() error {
	return RunArgs(os.Args[1:], os.Stdout)
}

// nonInteractiveStartError: `axiom` sin argumentos abre la UI y necesita terminal.
const nonInteractiveStartError = "axiom requires both stdin and stdout to be terminals (TTYs); use --version, axiom update, or --help for non-interactive use"

// clearPendingSyncAfterDeferredSync clears PendingSync under the canonical
// install-state lock. It re-reads the latest state inside the lock so changes
// written by a concurrent writer between the deferred sync and this clear are
// preserved; the fallback is used only when the state file does not exist yet.
func clearPendingSyncAfterDeferredSync(homeDir string, fallback state.InstallState) error {
	return statecoord.WithLock(homeDir, func() error {
		current, readErr := state.Read(homeDir)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				current = fallback
			} else {
				return readErr
			}
		}
		current.PendingSync = false
		return state.Write(homeDir, current)
	})
}

func RunArgs(args []string, stdout io.Writer) error {
	if len(args) == 0 && (!isattyFn(os.Stdin.Fd()) || !isattyFn(os.Stdout.Fd())) {
		return errors.New(nonInteractiveStartError)
	}

	// Propagate the build-time version to the CLI and upgrade layers so backup
	// manifests record which version of axiom created them.
	cli.AppVersion = Version
	upgrade.AppVersion = Version

	// --yes as a global CLI flag for self-update is handled via AXIOM_YES=1.
	// Per-subcommand --yes flags (e.g. restore --yes) are parsed by each subcommand.

	// Platform-independent commands: no system detection, self-update, or
	// platform validation.
	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-v":
			_, _ = fmt.Fprintf(stdout, "axiom %s\n", Version)
			return nil
		case "help", "--help", "-h":
			printHelp(stdout, Version)
			return nil
		case "uninstall":
			if len(args) >= 2 && args[1] == "opencode-plugin" {
				_, err := cli.RunUninstallOpenCodePlugin(args[2:], stdout)
				return err
			}
			return runUninstall(args[1:], stdout)
		case "skill-registry":
			return runSkillRegistry(args[1:], stdout)
		case "sdd-status":
			return cli.RunSDDStatus(args[1:], stdout)
		case "sdd-continue":
			return cli.RunSDDContinue(args[1:], stdout)
		case "sdd-attempt":
			return cli.RunSDDAttempt(args[1:], stdout)
		case "sdd-archive-compose":
			return cli.RunSDDArchiveCompose(args[1:], stdout)
		case "sdd-task-result":
			return cli.RunSDDTaskResult(args[1:], stdout)
		case "sdd-preflight-hook":
			return cli.RunSDDPreflightHook(args[1:], stdout)
		case "codegraph":
			return cli.RunCodeGraph(args[1:], stdout)
		case "install":
			if hasHelpFlag(args[1:]) {
				cli.PrintInstallHelp(stdout)
				return nil
			}
		case "setup":
			if hasHelpFlag(args[1:]) {
				cli.PrintInstallHelp(stdout)
				return nil
			}
		case "sync":
			if hasHelpFlag(args[1:]) {
				cli.PrintSyncHelp(stdout)
				return nil
			}
		}
	}

	// Issue #535: parse the upgrade command's arguments exactly once, before
	// any platform validation, system detection, self-update, update check,
	// backup, or execution effect. Unsupported pre-delimiter dash arguments
	// are rejected here with zero effects. The parsed value is forwarded to
	// runUpgrade below so the executor never reparses raw CLI args. The
	// help selection branch is added in the follow-up help-behavior change.
	var parsedUpgrade *upgradeArgs
	if len(args) > 0 && args[0] == "upgrade" {
		parsed, err := parseUpgradeArgs(args[1:])
		if err != nil {
			return err
		}
		parsedUpgrade = &parsed
	}

	if err := ensureCurrentOSSupported(); err != nil {
		return err
	}

	result, err := detectSystem(context.Background())
	if err != nil {
		return fmt.Errorf("detect system: %w", err)
	}

	if !result.System.Supported {
		return system.EnsureSupportedPlatform(result.System.Profile)
	}

	var (
		profile         system.PlatformProfile
		profileResolved bool
	)
	resolveProfile := func() system.PlatformProfile {
		if !profileResolved {
			profile = cli.ResolveInstallProfile(result)
			profileResolved = true
		}
		return profile
	}

	// Self-update: check for a newer axiom release and apply it before
	// CLI dispatch. Errors are non-fatal — logged and swallowed.
	// Skip auto-upgrade on bare start (len(args) == 0) to avoid silently
	// replacing the binary while the user expects a clean UI launch (#696).
	isBareStart := len(args) == 0
	if !isBareStart && !isExplicitUpdateFlow(args) {
		if err := selfUpdateFn(context.Background(), Version, resolveProfile(), stdout); err != nil {
			_, _ = fmt.Fprintf(stdout, "Warning: self-update failed: %v\n", err)
		}
	}

	if len(args) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve user home directory: %w", err)
		}

		// Load persisted state to read PendingSync. Unreadable state cannot
		// safely be treated as empty.
		installedState, err := state.Read(homeDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read install state: %w", err)
		}
		// Con el ámbito workspace se lee el estado del proyecto
		// (D2); los campos de máquina, como PendingSync, siguen siendo los del estado
		// global.
		if store := installStateStore(homeDir); store.IsProject() {
			projectState, projectErr := store.Read()
			if projectErr != nil && !os.IsNotExist(projectErr) {
				return fmt.Errorf("read install state: %w", projectErr)
			}
			if projectErr == nil {
				installedState = state.WithMachineFields(projectState, installedState)
			}
		}

		// Deferred sync: if a previous axiom self-upgrade set PendingSync=true,
		// run sync now with the new binary before opening the UI. On success,
		// clear the flag. On failure, log and leave the flag set for idempotent
		// retry on the next launch (per spec scenario "deferred sync fails → retry").
		// This is non-fatal — a sync failure must never block the UI from opening.
		if installedState.PendingSync {
			if err := deferredSyncFn(); err != nil {
				_, _ = fmt.Fprintf(stdout, "Warning: deferred sync failed: %v\n", err)
				// Leave PendingSync=true so the next launch retries.
			} else {
				// Re-read the latest state inside the canonical lock so a
				// concurrent writer's changes survive the PendingSync clear.
				if writeErr := clearPendingSyncAfterDeferredSync(homeDir, installedState); writeErr != nil {
					// Best-effort: surface the failure so it's not silently swallowed.
					// Idempotent re-sync on the next launch is acceptable.
					_, _ = fmt.Fprintf(stdout, "Warning: failed to clear PendingSync flag: %v\n", writeErr)
				}
			}
			// Self-update path: the previous launch completed a axiom
			// self-upgrade under the old binary and set PendingSync=true. We are
			// now running under the new binary; print the doctor advisory so the
			// user can verify ecosystem health against the post-upgrade state.
			// Print regardless of sync outcome — the advisory is informational.
			printPostUpgradeDoctorAdvisory(stdout)
		}

		// Abrir la UI lo hace main tras volver de aquí sin error.
		return nil
	}

	switch args[0] {
	case "update":
		return runUpdate(context.Background(), Version, resolveProfile(), stdout)
	case "upgrade":
		return runUpgrade(context.Background(), *parsedUpgrade, result, stdout)
	case "install":
		installResult, err := cli.RunInstall(args[1:], result)
		if err != nil {
			return err
		}

		if installResult.DryRun {
			_, _ = fmt.Fprintln(stdout, cli.RenderDryRun(installResult))
		} else {
			_, _ = fmt.Fprint(stdout, verify.RenderReport(installResult.Verify))
			_, _ = fmt.Fprint(stdout, cli.RenderInstallManualActions(installResult))
		}

		return nil
	case "setup":
		setupArgs := append([]string{"--scope", "workspace"}, args[1:]...)
		installResult, err := cli.RunInstall(setupArgs, result)
		if err != nil {
			return err
		}

		if installResult.DryRun {
			_, _ = fmt.Fprintln(stdout, cli.RenderDryRun(installResult))
		} else {
			_, _ = fmt.Fprint(stdout, verify.RenderReport(installResult.Verify))
			_, _ = fmt.Fprint(stdout, cli.RenderInstallManualActions(installResult))
		}

		return nil
	case "sync":
		syncResult, err := cli.RunSync(args[1:])
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintln(stdout, cli.RenderSyncReport(syncResult))
		return nil
	case "restore":
		return cli.RunRestore(args[1:], stdout)
	case "doctor":
		return cli.RunDoctor(context.Background(), stdout)
	default:
		return fmt.Errorf("unknown command %q — run 'axiom help' for available commands", args[0])
	}
}

func runUninstall(args []string, stdout io.Writer) error {
	result, err := cli.RunUninstall(args, stdout)
	if result.Manifest.ID != "" {
		_, _ = fmt.Fprintln(stdout, cli.RenderUninstallReport(result))
	}
	return err
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}

	return false
}

func runSkillRegistry(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: axiom skill-registry <refresh|list> [flags]")
	}
	switch args[0] {
	case "refresh":
		return runSkillRegistryRefresh(args[1:], stdout)
	case "list":
		return runSkillRegistryList(args[1:], stdout)
	default:
		return fmt.Errorf("unknown skill-registry command %q (want refresh or list)", args[0])
	}
}

// resolveSkillRegistryDirs resolves the working directory (defaulting to the
// process cwd) and the user home directory used to locate skills. An empty
// home resolves the process user home directory. A relative cwd (for
// example "--cwd .") is made absolute: otherwise the root guard sees
// filepath.Dir(".") == "." and skips the refresh as a filesystem root.
func resolveSkillRegistryDirs(cwd, home string) (string, string, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("resolve cwd: %w", err)
		}
	} else {
		// Fold the separators before deciding whether --cwd is absolute: on a
		// non-Windows host `\tmp\ws` is not absolute, and filepath.Abs would join
		// it to the process cwd instead of resolving it to /tmp/ws.
		cwd = skillregistry.CleanPathArg(cwd)
		if !filepath.IsAbs(cwd) {
			abs, err := filepath.Abs(cwd)
			if err != nil {
				return "", "", fmt.Errorf("resolve cwd %q: %w", cwd, err)
			}
			cwd = abs
		}
	}
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("resolve home directory: %w", err)
		}
	}
	return cwd, home, nil
}

// runSkillRegistryRefresh keeps the legacy surface byte-identical (RF-SKI-05):
// primary line only, no per-destination lines and no mirror warning. The
// engine underneath is the shared unified one (D-13).
func runSkillRegistryRefresh(args []string, stdout io.Writer) error {
	parsed, err := parseSkillIndexRefreshFlags(skillIndexSurfaceLegacy, args)
	if err != nil {
		return err
	}
	result, skipped, err := runSharedRefresh(parsed, stdout)
	if err != nil {
		return err
	}
	if skipped || parsed.quiet {
		return nil
	}
	writePrimaryRefreshLine(stdout, result)
	return nil
}

// runSkillRegistryList keeps the legacy list output byte-identical
// (RF-SKI-05) over the shared read-only engine.
func runSkillRegistryList(args []string, stdout io.Writer) error {
	return writeSkillIndexList(skillIndexSurfaceLegacy, args, stdout)
}

func runUpdate(ctx context.Context, currentVersion string, profile system.PlatformProfile, stdout io.Writer) error {
	results := updateCheckAll(ctx, currentVersion, profile)
	_, _ = fmt.Fprint(stdout, update.RenderCLI(results))
	if err := updateCheckError(results); err != nil {
		return err
	}
	return nil
}

// runUpgrade handles the `axiom upgrade [--dry-run] [tool...]` command.
//
// This command:
//   - Checks for available updates for managed tools (axiom, engram)
//   - Snapshots agent config paths before execution (config preservation by design)
//   - Executes binary-only upgrades; does NOT invoke install or sync pipelines
//   - Skips axiom itself when running as a dev build (version="dev")
//   - Falls back to source-install guidance where official binaries are unavailable
//
// Issue #535: runUpgrade consumes a structured upgradeArgs value parsed once
// in RunArgs. It forwards the parsed flags and tool filters to the update
// check and executor exactly once and never reparses raw CLI arguments.
//
// The implementation lives in runUpgradeReport so the dashboard can obtain the
// structured outcome (D-06) without duplicating the run; this wrapper keeps the
// CLI error contract and the exact same stdout bytes (RF-UPD-05).
func runUpgrade(ctx context.Context, args upgradeArgs, detection system.DetectionResult, stdout io.Writer) error {
	_, err := runUpgradeReport(ctx, args, detection, stdout)
	return err
}

func updateCheckError(results []update.UpdateResult) error {
	failed := update.CheckFailures(results)
	if len(failed) == 0 {
		return nil
	}

	return fmt.Errorf("update check failed for: %s", strings.Join(failed, ", "))
}

func syncShouldIncludePermissions(agentIDs []model.AgentID) bool {
	for _, id := range agentIDs {
		if id == model.AgentCodex {
			return true
		}
	}
	return false
}

func syncArgsForDiscoveredAgents(homeDir string) []string {
	store := cli.EffectiveStateStore(homeDir)
	if persisted, err := store.Read(); err == nil && persisted.SelectionConfigured {
		if (model.Selection{Components: persisted.Components}).HasComponent(model.ComponentPermission) {
			return []string{"--include-permissions"}
		}
		return nil
	}
	if syncShouldIncludePermissions(cli.DiscoverAgents(store)) {
		return []string{"--include-permissions"}
	}
	return nil
}

// ListBackups returns all backup manifests from the backup directory.
func ListBackups() []backup.Manifest {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	roots := backup.BackupRoots(homeDir)

	manifests := make([]backup.Manifest, 0)

	for _, backupRoot := range roots {
		entries, err := os.ReadDir(backupRoot)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			manifestPath := filepath.Join(backupRoot, entry.Name(), backup.ManifestFilename)
			manifest, err := backup.ReadManifest(manifestPath)
			if err != nil {
				continue
			}
			manifest.Origin = backup.BackupOriginForRoot(homeDir, backupRoot)
			manifests = append(manifests, manifest)
		}
	}

	// Sort by creation time (newest first) — the IDs are timestamps.
	for i := 0; i < len(manifests); i++ {
		for j := i + 1; j < len(manifests); j++ {
			if manifests[j].CreatedAt.After(manifests[i].CreatedAt) {
				manifests[i], manifests[j] = manifests[j], manifests[i]
			}
		}
	}

	return manifests
}

// isExplicitUpdateFlow reports whether the current invocation is already in the
// explicit update/upgrade path. In those cases, self-update must be skipped to
// avoid preempting the user's requested command behavior.
func isExplicitUpdateFlow(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return args[0] == "update" || args[0] == "upgrade"
}

// installStateStore elige el almacén del estado de instalación para la UI según el
// ámbito de instalación: el estado del proyecto con workspace (D2). La UI no
// recibe --scope, así que no elige el ámbito global: una variable
// AXIOM_INSTALL_SCOPE no válida, o con global, deja el ámbito predeterminado.
// Se descarta el aviso de raíz.
func installStateStore(homeDir string) state.Store {
	scope, err := cli.ResolveInstallScope("")
	if err != nil {
		scope = cli.ScopeWorkspace
	}
	workspaceDir, _ := cli.ResolveWorkspaceDir(scope, homeDir)
	return cli.StateStoreFor(scope, homeDir, workspaceDir)
}
