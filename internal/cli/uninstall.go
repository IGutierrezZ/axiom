package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	componentuninstall "github.com/IGutierrezZ/axiom/v3/internal/components/uninstall"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

type UninstallFlags struct {
	Agents     []string
	Components []string
	All        bool
	Yes        bool
	// Scope es el ámbito (--scope): workspace revierte el proyecto y global, la
	// carpeta de usuario. Vacío usa el mismo valor por defecto que install.
	Scope string
	// PurgeState borra además la raíz de estado de Axiom en una desinstalación
	// global completa; PurgeHub, también el registro del hub aunque tenga proyectos.
	PurgeState bool
	PurgeHub   bool
}

func ParseUninstallFlags(args []string) (UninstallFlags, error) {
	var opts UninstallFlags

	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})
	registerListFlag(fs, "agent", &opts.Agents)
	registerListFlag(fs, "agents", &opts.Agents)
	registerListFlag(fs, "component", &opts.Components)
	registerListFlag(fs, "components", &opts.Components)
	fs.BoolVar(&opts.All, "all", false, "remove managed configuration for all supported agents")
	fs.BoolVar(&opts.Yes, "yes", false, "skip confirmation prompt")
	fs.BoolVar(&opts.Yes, "y", false, "skip confirmation prompt")
	fs.StringVar(&opts.Scope, "scope", "", "uninstall scope: workspace (default) or global")
	fs.BoolVar(&opts.PurgeState, "purge-state", false, "also delete the Axiom state directory (global scope, with --all)")
	fs.BoolVar(&opts.PurgeHub, "purge-hub", false, "with --purge-state, also delete the hub registry even if it lists other projects")

	if err := fs.Parse(args); err != nil {
		return UninstallFlags{}, err
	}
	if fs.NArg() > 0 {
		return UninstallFlags{}, fmt.Errorf("unexpected uninstall argument %q", fs.Arg(0))
	}
	if opts.All && (len(opts.Agents) > 0 || len(opts.Components) > 0) {
		return UninstallFlags{}, fmt.Errorf("--all cannot be combined with --agent/--agents or --component/--components")
	}
	if !opts.All && len(opts.Agents) == 0 {
		return UninstallFlags{}, fmt.Errorf("partial uninstall requires at least one --agent/--agents or use --all")
	}
	if _, err := ResolveInstallScope(opts.Scope); err != nil {
		return UninstallFlags{}, err
	}
	if opts.PurgeHub && !opts.PurgeState {
		return UninstallFlags{}, fmt.Errorf("--purge-hub requiere --purge-state: añádelo, por ejemplo `axiom uninstall --all --purge-state --purge-hub`")
	}
	if opts.PurgeState && !opts.All {
		return UninstallFlags{}, fmt.Errorf("--purge-state solo se admite con --all: borra el estado de toda la máquina; usa `axiom uninstall --all --purge-state`")
	}

	return opts, nil
}

// uninstallOptions traduce el ámbito y las purgas de la línea de comandos a las
// opciones del servicio de desinstalación.
func uninstallOptions(scope InstallScope, purgeState, purgeHub bool) componentuninstall.Options {
	opts := componentuninstall.Options{
		Scope:                  componentuninstall.ScopeGlobal,
		AgentSupportsWorkspace: agentSupportsWorkspace,
	}
	if scope == ScopeWorkspace {
		opts.Scope = componentuninstall.ScopeWorkspace
		return opts
	}
	opts.PurgeState = purgeState
	opts.PurgeHub = purgeHub
	return opts
}

// resolveUninstallRoots resuelve el ámbito y la raíz sobre la que opera la
// desinstalación, igual que install y sync: el ámbito sale de --scope o de
// AXIOM_INSTALL_SCOPE (workspace por defecto) y, en workspace, la raíz es la del
// proyecto aunque el comando se lance desde una subcarpeta.
func resolveUninstallRoots(scopeFlag string) (scope InstallScope, homeDir, workspaceDir string, err error) {
	scope, err = ResolveInstallScope(scopeFlag)
	if err != nil {
		return "", "", "", err
	}
	homeDir, err = osUserHomeDir()
	if err != nil {
		return "", "", "", fmt.Errorf("resolve home directory: %w", err)
	}
	workspaceDir, notice := resolveWorkspaceDir(scope, homeDir)
	if workspaceDir == "" {
		return "", "", "", fmt.Errorf("no se pudo resolver la carpeta del proyecto (la carpeta actual no se puede leer); vuelve a lanzar `axiom uninstall` desde una carpeta que exista")
	}
	noteWorkspaceRoot(notice)
	return scope, homeDir, workspaceDir, nil
}

func RunUninstall(args []string, stdout io.Writer) (componentuninstall.Result, error) {
	return runUninstallWithInput(args, stdout, os.Stdin)
}

// RunUninstallWithSelection desinstala los agentes y componentes elegidos en el
// ámbito dado: con workspace, workspaceDir es la raíz del proyecto.
func RunUninstallWithSelection(scope InstallScope, homeDir, workspaceDir string, agentIDs []model.AgentID, componentIDs []model.ComponentID) (componentuninstall.Result, error) {
	agents := make([]string, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		agents = append(agents, string(agentID))
	}
	components := make([]string, 0, len(componentIDs))
	for _, componentID := range componentIDs {
		components = append(components, string(componentID))
	}
	return componentuninstall.PartialUninstallScoped(homeDir, workspaceDir, AppVersion, uninstallOptions(scope, false, false), agents, components)
}

// RunUninstallWithSelectionAndProfiles es RunUninstallWithSelection con la
// selección de perfiles de SDD y el ámbito de los datos de Engram.
func RunUninstallWithSelectionAndProfiles(scope InstallScope, homeDir, workspaceDir string, agentIDs []model.AgentID, componentIDs []model.ComponentID, profileNames []string, engramScope model.EngramUninstallScope) (componentuninstall.Result, error) {
	agents := make([]string, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		agents = append(agents, string(agentID))
	}
	components := make([]string, 0, len(componentIDs))
	for _, componentID := range componentIDs {
		components = append(components, string(componentID))
	}
	return componentuninstall.PartialUninstallWithProfileSelectionScoped(homeDir, workspaceDir, AppVersion, uninstallOptions(scope, false, false), agents, components, profileNames, engramScope)
}

func RenderUninstallReport(result componentuninstall.Result) string {
	var b strings.Builder

	// The header states what actually happened. A batch that failed for one
	// agent still commits the agents that succeeded (see Result.FailedAgents),
	// so "complete" would be a lie the user reads before the failure detail
	// printed further down under manual cleanup.
	if len(result.FailedAgents) > 0 {
		_, _ = fmt.Fprintf(&b, "Managed uninstall partially complete: %s failed\n", strings.Join(agentLabels(result.FailedAgents), ", "))
	} else if len(result.RetainedPiResources) > 0 {
		_, _ = fmt.Fprintln(&b, "Managed uninstall finished; Pi resources retained for review")
	} else {
		_, _ = fmt.Fprintln(&b, "Managed uninstall complete")
	}
	if result.Manifest.ID != "" {
		_, _ = fmt.Fprintf(&b, "Backup: %s (%s)\n", result.Manifest.ID, result.Manifest.DisplayLabel())
		_, _ = fmt.Fprintf(&b, "Backup path: %s\n", result.BackupPath)
	}
	_, _ = fmt.Fprintf(&b, "Changed files: %d\n", len(result.ChangedFiles))
	_, _ = fmt.Fprintf(&b, "Removed files: %d\n", len(result.RemovedFiles))
	_, _ = fmt.Fprintf(&b, "Removed directories: %d\n", len(result.RemovedDirectories))
	if len(result.AgentsRemovedFromState) > 0 {
		_, _ = fmt.Fprintf(&b, "Updated state.json: removed %s\n", strings.Join(agentLabels(result.AgentsRemovedFromState), ", "))
	}
	appendPathSection(&b, "Rewritten files", result.ChangedFiles)
	appendPathSection(&b, "Deleted files", result.RemovedFiles)
	appendPathSection(&b, "Deleted directories", result.RemovedDirectories)
	appendPathSection(&b, "Removed PATH entries", result.RemovedPathEntries)
	appendPathSection(&b, "Purged state (--purge-state)", result.PurgedPaths)
	appendPathSection(&b, "Retained Pi resources (not deleted)", result.RetainedPiResources)
	appendOptionalPiPackageCleanup(&b, result.OptionalPiPackageCleanupCommands)
	appendPathSection(&b, "Manual cleanup required", result.ManualActions)

	return strings.TrimRight(b.String(), "\n")
}

func runUninstallWithInput(args []string, stdout io.Writer, stdin io.Reader) (componentuninstall.Result, error) {
	flags, err := ParseUninstallFlags(args)
	if err != nil {
		return componentuninstall.Result{}, err
	}

	scope, homeDir, workspaceDir, err := resolveUninstallRoots(flags.Scope)
	if err != nil {
		return componentuninstall.Result{}, err
	}

	if !flags.Yes {
		confirmed, err := promptUninstallConfirm(flags, scope, workspaceDir, stdout, stdin)
		if err != nil {
			return componentuninstall.Result{}, err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(stdout, "uninstall cancelled")
			return componentuninstall.Result{}, nil
		}
	}

	opts := uninstallOptions(scope, flags.PurgeState, flags.PurgeHub)
	if flags.All {
		return componentuninstall.CompleteUninstallScoped(homeDir, workspaceDir, AppVersion, opts)
	}
	return componentuninstall.PartialUninstallScoped(homeDir, workspaceDir, AppVersion, opts, flags.Agents, flags.Components)
}

func promptUninstallConfirm(flags UninstallFlags, scope InstallScope, workspaceDir string, stdout io.Writer, stdin io.Reader) (bool, error) {
	if scope == ScopeWorkspace {
		_, _ = fmt.Fprintf(stdout, "Ámbito: workspace (%s). Solo se revierte lo que Axiom gestiona en el proyecto; tu carpeta de usuario no se toca.\n", workspaceDir)
	} else {
		_, _ = fmt.Fprintln(stdout, "Ámbito: global (tu configuración de usuario y las entradas de PATH que gestiona Axiom). Los proyectos no se tocan: ejecuta `axiom uninstall --scope workspace` en cada uno.")
		if flags.PurgeState {
			_, _ = fmt.Fprintln(stdout, "--purge-state: se borrará el directorio de estado de Axiom, copias de seguridad incluidas.")
		}
	}
	if flags.All {
		_, _ = fmt.Fprintln(stdout, "This will remove axiom managed configuration from all supported agents.")
	} else {
		_, _ = fmt.Fprintf(stdout, "This will remove axiom managed configuration from: %s\n", strings.Join(agentLabelsFromStrings(flags.Agents), ", "))
	}
	if len(flags.Components) > 0 {
		_, _ = fmt.Fprintf(stdout, "Components: %s\n", strings.Join(flags.Components, ", "))
	} else {
		_, _ = fmt.Fprintln(stdout, "Components: all managed uninstallable components")
	}
	_, _ = fmt.Fprintln(stdout, "A backup snapshot will be created before any file is modified.")
	_, _ = fmt.Fprint(stdout, "Type 'yes' to confirm: ")

	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("read uninstall confirmation: %w", err)
		}
		return false, fmt.Errorf("no confirmation provided (use --yes to skip prompt)")
	}
	return strings.EqualFold(strings.TrimSpace(scanner.Text()), "yes"), nil
}

func appendOptionalPiPackageCleanup(b *strings.Builder, commands []string) {
	if len(commands) == 0 {
		return
	}

	_, _ = fmt.Fprintln(b, "\nOptional Pi package cleanup:")
	_, _ = fmt.Fprintln(b, "  Review shared or user-modified packages/resources before removing them:")
	for _, command := range commands {
		_, _ = fmt.Fprintf(b, "  - %s\n", command)
	}
}

func appendPathSection(b *strings.Builder, title string, paths []string) {
	if len(paths) == 0 {
		return
	}

	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	cwd, cwdErr := os.Getwd()
	_, _ = fmt.Fprintf(b, "\n%s:\n", title)
	for _, path := range sorted {
		rel := path
		if cwdErr == nil {
			if r, relErr := filepath.Rel(cwd, path); relErr == nil && !strings.HasPrefix(r, "..") {
				rel = r
			}
		}
		_, _ = fmt.Fprintf(b, "  - %s\n", rel)
	}
}

func agentLabels(agentIDs []model.AgentID) []string {
	labels := make([]string, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		labels = append(labels, agentLabel(agentID))
	}
	return labels
}

func agentLabelsFromStrings(agentIDs []string) []string {
	labels := make([]string, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		labels = append(labels, agentLabel(model.AgentID(agentID)))
	}
	return labels
}

func agentLabel(agentID model.AgentID) string {
	for _, agent := range catalog.AllAgents() {
		if agent.ID == agentID {
			return agent.Name
		}
	}
	return string(agentID)
}
