package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// traceSchema identifica la salida JSON de `axiom sdd trace`.
const traceSchema = "axiom.sdd-trace/v1"

// RunSDDTrace es `axiom sdd trace --cwd <ruta> --change <c> [--integration-branch
// <ref>] [--json]` (REQ-29.9, design D14 y §6.4). Es de solo lectura: deriva con
// `git log --first-parent` el commit de integración del código y el del archivado
// del cambio, sin estado persistido y con el mismo resultado con squash, merge o
// rebase, y los contrasta con la evidencia `pr_merged` de la compuerta
// `integration`. Muestra además la cadena requisito, `Cubre:`, fila de la matriz
// de los artefactos del cambio (activo o archivado).
//
// Selección del repositorio git (design §8): el nombre del cambio y la referencia
// se validan antes de invocar git; git se ejecuta solo a través de
// gitcommon.Run, con el directorio absoluto del repositorio que contiene el
// planning home de `--cwd`, nunca con el directorio del proceso.
func RunSDDTrace(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderSDDTraceHelp(stdout)
	}
	flags := flag.NewFlagSet("sdd-trace", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", "", "Directorio desde el que resolver el workspaceRoot")
	change := flags.String("change", "", "Cambio cuya traza se deriva")
	integrationBranch := flags.String("integration-branch", "", "Rama de integración; por defecto la de la instantánea del kickoff o la de axiom.yaml")
	asJSON := flags.Bool("json", false, "Salida JSON (esquema "+traceSchema+")")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("sdd trace: %w; ejecuta `axiom sdd trace --help`", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("sdd trace: argumento inesperado %q; ejecuta `axiom sdd trace --cwd <dir> --change <cambio> [--json]`", flags.Arg(0))
	}
	if strings.TrimSpace(*change) == "" {
		return errors.New("sdd trace requiere --change; ejecuta `axiom sdd trace --cwd <dir> --change <cambio> [--json]`")
	}
	if err := gitref.ValidBranchComponent(*change); err != nil || *change == "archive" {
		reason := "es un nombre reservado"
		if err != nil {
			reason = err.Error()
		}
		return fmt.Errorf("sdd trace: --change %q no es un nombre de cambio válido: %s; usa el nombre de la carpeta del cambio que proyecta `axiom sdd status --cwd <dir> --json`", *change, reason)
	}
	if *integrationBranch != "" {
		if err := gitref.ValidRefName(*integrationBranch); err != nil {
			return fmt.Errorf("sdd trace: --integration-branch %q no es una referencia válida: %w", *integrationBranch, err)
		}
	}

	start := *cwd
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return fmt.Errorf("sdd trace: %w", err)
	}
	ctx := context.Background()
	gitRoot, err := gitcommon.TopLevel(ctx, loc.PlanningHome)
	if err != nil {
		return fmt.Errorf("sdd trace: no se puede localizar el repositorio git de %s: %w; la traza se deriva del historial git del planning home", loc.PlanningHome, err)
	}
	prefix, err := planningHomePrefix(gitRoot, loc.PlanningHome)
	if err != nil {
		return fmt.Errorf("sdd trace: %w", err)
	}

	changeRoot, source := resolveTraceChangeRoot(loc.PlanningHome, *change)
	var sealed *kickoff.Kickoff
	var ledger kickoff.GateLedger
	if changeRoot != "" {
		sealed, _ = kickoff.Load(changeRoot)
		ledger, _ = kickoff.LoadGates(changeRoot)
	}
	ref := traceIntegrationBranch(*integrationBranch, sealed, loc.Declarations)
	if err := gitref.ValidRefName(ref); err != nil {
		return fmt.Errorf("sdd trace: la rama de integración %q no es una referencia válida: %w; indícala con --integration-branch", ref, err)
	}

	derived, err := isoevidence.DeriveTrace(ctx, gitHistory{root: gitRoot}, isoevidence.TraceInput{
		Change: *change, Ref: ref, PlanningHomePrefix: prefix, EvidenceCommit: prMergedEvidence(ledger),
	})
	if err != nil {
		return fmt.Errorf("sdd trace: %w; si la rama %q no existe en este clon, indica otra con --integration-branch", err, ref)
	}

	report := traceReport{
		Schema: traceSchema, Change: *change, IntegrationBranch: ref,
		CodeCommit: traceCommit(derived.CodeCommit), ArchiveCommit: traceCommit(derived.ArchiveCommit),
		ArchiveFolder: derived.ArchiveFolder, Discrepancy: derived.Discrepancy,
		ChainSource: source, Chain: []traceChainEntry{},
	}
	if changeRoot != "" {
		report.Chain = traceChain(loc.PlanningHome, changeRoot, sealed)
	}
	return renderTrace(stdout, *asJSON, report)
}

// planningHomePrefix es la ruta del planning home relativa a la raíz git, con `/`;
// vacía si coinciden. Resuelve enlaces y nombres cortos para que ambas rutas sean
// comparables aunque git y el sistema las escriban distinto.
func planningHomePrefix(gitRoot, planningHome string) (string, error) {
	resolve := func(path string) string {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			return resolved
		}
		return path
	}
	relative, err := filepath.Rel(resolve(gitRoot), resolve(planningHome))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("el planning home %s no está dentro del repositorio git %s; ejecuta `axiom sdd trace` con un `--cwd` dentro del repositorio que contiene el planning home", planningHome, gitRoot)
	}
	relative = filepath.ToSlash(relative)
	if relative == "." {
		return "", nil
	}
	return relative, nil
}

// resolveTraceChangeRoot localiza la carpeta del cambio: la activa o, si no
// existe, la archivada `<fecha>-<cambio>` más reciente. Vacía si no está en el
// árbol de trabajo (la traza de git sigue siendo válida sin ella).
func resolveTraceChangeRoot(planningHome, change string) (root, source string) {
	active := filepath.Join(planningHome, "changes", change)
	if isDirectory(active) {
		return active, "active"
	}
	if archived, ok := kickoff.ArchivedChangeRoot(filepath.Dir(planningHome), change); ok {
		return archived, "archived"
	}
	return "", "none"
}

// traceIntegrationBranch elige la rama de integración: la indicada, la de la
// instantánea git del kickoff, la de axiom.yaml o la de por defecto.
func traceIntegrationBranch(flagValue string, sealed *kickoff.Kickoff, declarations workspace.Declarations) string {
	switch {
	case flagValue != "":
		return flagValue
	case sealed != nil && sealed.Git != nil && sealed.Git.IntegrationBranch != "":
		return sealed.Git.IntegrationBranch
	case declarations.Git != nil && declarations.Git.IntegrationBranch != "":
		return declarations.Git.IntegrationBranch
	}
	return workspace.DefaultIntegrationBranch
}

// commitPattern son los commits que se entregan a git como revisión: hexadecimal.
var commitPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)

// prMergedEvidence devuelve el commit del último registro `approved` de la
// compuerta `integration` con evidencia `pr_merged`, o vacío si no hay o no es un
// commit (lo que no es hexadecimal nunca llega a git).
func prMergedEvidence(ledger kickoff.GateLedger) string {
	evidence := ""
	for _, record := range ledger.Records {
		if record.Gate == kickoff.GateIntegration && record.Decision == kickoff.DecisionApproved && record.EvidenceKind == kickoff.EvidencePRMerged {
			evidence = record.EvidenceRef
		}
	}
	if !commitPattern.MatchString(evidence) {
		return ""
	}
	return evidence
}

// gitHistory implementa isoevidence.History sobre gitcommon.Run, el único punto de
// ejecución de git. root es la raíz absoluta del repositorio. Solo se ejecutan
// lecturas, con revisiones validadas (hexadecimal o nombre de referencia) y rutas
// que salen del nombre de un cambio ya validado.
type gitHistory struct{ root string }

func (h gitHistory) run(ctx context.Context, args ...string) (gitcommon.Result, error) {
	return gitcommon.Run(ctx, h.root, args...)
}

// revision admite una revisión que sea un commit (hexadecimal) o un nombre de
// referencia válido: nunca una expresión de revisión.
func revision(rev string) error {
	if commitPattern.MatchString(rev) {
		return nil
	}
	return gitref.ValidRefName(rev)
}

func gitFailure(result gitcommon.Result, what string) error {
	// refusal:by-design world-action: el fallo lo produce git y su salida de error, incluida en el mensaje, nombra la causa y su remedio.
	return fmt.Errorf("%s falló (salida %d): %s", what, result.ExitCode, strings.TrimSpace(result.Stderr))
}

func (h gitHistory) FirstParentAdded(ctx context.Context, ref, path string) ([]string, error) {
	if err := revision(ref); err != nil {
		return nil, err
	}
	result, err := h.run(ctx, "log", "--first-parent", "--reverse", "--no-renames", "--diff-filter=A", "--format=%H", ref, "--", path)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, gitFailure(result, "git log")
	}
	return strings.Fields(string(result.Stdout)), nil
}

func (h gitHistory) TreeHas(ctx context.Context, commit, path string) (bool, error) {
	if err := revision(commit); err != nil {
		return false, err
	}
	result, err := h.run(ctx, "ls-tree", "-z", "--name-only", commit, "--", path)
	if err != nil {
		return false, err
	}
	if result.ExitCode != 0 {
		return false, gitFailure(result, "git ls-tree")
	}
	return len(strings.TrimRight(string(result.Stdout), "\x00")) > 0, nil
}

func (h gitHistory) TreeEntries(ctx context.Context, ref, dir string) ([]string, error) {
	if err := revision(ref); err != nil {
		return nil, err
	}
	prefix := strings.TrimSuffix(dir, "/") + "/"
	result, err := h.run(ctx, "ls-tree", "-z", "--name-only", ref, "--", prefix)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, gitFailure(result, "git ls-tree")
	}
	var names []string
	for _, entry := range strings.Split(string(result.Stdout), "\x00") {
		if name := strings.TrimPrefix(entry, prefix); entry != "" && name != entry && name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func (h gitHistory) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	for _, rev := range []string{ancestor, descendant} {
		if err := revision(rev); err != nil {
			return false, err
		}
	}
	result, err := h.run(ctx, "merge-base", "--is-ancestor", ancestor, descendant)
	if err != nil {
		return false, err
	}
	switch result.ExitCode {
	case 0:
		return true, nil
	case 1:
		return false, nil
	}
	return false, gitFailure(result, "git merge-base")
}

// traceReport es la salida de `axiom sdd trace`, en texto y en JSON.
type traceReport struct {
	Schema            string            `json:"schema"`
	Change            string            `json:"change"`
	IntegrationBranch string            `json:"integrationBranch"`
	CodeCommit        traceCommitJSON   `json:"codeCommit"`
	ArchiveCommit     traceCommitJSON   `json:"archiveCommit"`
	ArchiveFolder     string            `json:"archiveFolder,omitempty"`
	Discrepancy       string            `json:"discrepancy,omitempty"`
	ChainSource       string            `json:"chainSource"`
	Chain             []traceChainEntry `json:"chain"`
}

type traceCommitJSON struct {
	Commit string `json:"commit,omitempty"`
	Found  bool   `json:"found"`
}

func traceCommit(ref isoevidence.CommitRef) traceCommitJSON {
	return traceCommitJSON{Commit: ref.Commit, Found: ref.Found}
}

// traceChainEntry es un requisito con las líneas `Cubre:` que lo citan y las
// filas de la matriz que lo prueban.
type traceChainEntry struct {
	Requirement string           `json:"requirement"`
	Domain      string           `json:"domain,omitempty"`
	Section     string           `json:"section,omitempty"`
	Name        string           `json:"name,omitempty"`
	Cubre       []traceCubre     `json:"cubre"`
	Rows        []traceMatrixRow `json:"rows"`
}

type traceCubre struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type traceMatrixRow struct {
	Scenario  string `json:"scenario"`
	Type      string `json:"type"`
	Reference string `json:"reference,omitempty"`
	Result    string `json:"result"`
	Commit    string `json:"commit,omitempty"`
}

// traceChain construye la cadena de los artefactos del cambio: los requisitos de
// la delta (ADDED y MODIFIED) y los citados en `Cubre:`, cada uno con sus líneas
// `Cubre:` y sus filas de matriz, ordenados por identificador.
func traceChain(planningHome, changeRoot string, sealed *kickoff.Kickoff) []traceChainEntry {
	tasksFiles, verifyFiles := []string{"tasks.md"}, []string{"verify-report.md"}
	if sealed != nil {
		if perRole := kickoff.RoleTaskFiles(sealed.Config.Roles); perRole != nil {
			tasksFiles = perRole
		}
		verifyFiles = verifyFiles[:0]
		for _, role := range sealed.Config.Roles {
			verifyFiles = append(verifyFiles, role.VerifyFile)
		}
		if len(verifyFiles) == 0 {
			verifyFiles = []string{"verify-report.md"}
		}
	}

	entries := map[string]*traceChainEntry{}
	entry := func(id string) *traceChainEntry {
		if entries[id] == nil {
			entries[id] = &traceChainEntry{Requirement: id, Cubre: []traceCubre{}, Rows: []traceMatrixRow{}}
		}
		return entries[id]
	}

	delta, _, err := sddstatus.ISOChangeRequirements(planningHome, changeRoot)
	if err == nil {
		for _, requirement := range delta {
			if requirement.Section != isoevidence.SectionAdded && requirement.Section != isoevidence.SectionModified {
				continue
			}
			if ids, _ := isoevidence.RequirementIDs(requirement.Name); len(ids) == 1 {
				e := entry(ids[0])
				e.Domain, e.Section, e.Name = requirement.Domain, string(requirement.Section), requirement.Name
			}
		}
	}
	for _, name := range tasksFiles {
		for _, line := range isoevidence.ParseCubre(name, readTraceFile(changeRoot, name)) {
			for _, raw := range line.IDs {
				if ids, _ := isoevidence.RequirementIDs(raw); len(ids) == 1 {
					e := entry(ids[0])
					e.Cubre = append(e.Cubre, traceCubre{File: line.File, Line: line.Line})
				}
			}
		}
	}
	for _, name := range verifyFiles {
		for _, row := range isoevidence.ParseMatrix(readTraceFile(changeRoot, name)).Rows {
			if ids, _ := isoevidence.RequirementIDs(row.RequirementID); len(ids) == 1 {
				e := entry(ids[0])
				e.Rows = append(e.Rows, traceMatrixRow{Scenario: row.Scenario, Type: row.Type, Reference: row.Reference, Result: row.Result, Commit: row.Commit})
			}
		}
	}

	chain := make([]traceChainEntry, 0, len(entries))
	for _, e := range entries {
		chain = append(chain, *e)
	}
	sort.Slice(chain, func(i, j int) bool { return chain[i].Requirement < chain[j].Requirement })
	return chain
}

// readTraceFile lee un artefacto del cambio; un fichero ausente o ilegible cuenta
// como vacío, porque la traza informa de lo que hay.
func readTraceFile(changeRoot, name string) string {
	data, err := os.ReadFile(filepath.Join(changeRoot, filepath.Base(name)))
	if err != nil {
		return ""
	}
	return string(data)
}

func renderTrace(stdout io.Writer, asJSON bool, report traceReport) error {
	if asJSON {
		return encodeJSON(stdout, report)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Traza del cambio %s en %s\n", report.Change, report.IntegrationBranch)
	writeCommit := func(label string, commit traceCommitJSON, absent string) {
		if commit.Found {
			fmt.Fprintf(&b, "  %s: %s\n", label, commit.Commit)
			return
		}
		fmt.Fprintf(&b, "  %s: %s\n", label, absent)
	}
	writeCommit("commit de código", report.CodeCommit, "no hay (el cambio aún no se integró en la línea --first-parent)")
	writeCommit("commit de archivado", report.ArchiveCommit, "no hay (el cambio aún no está archivado en la rama)")
	if report.ArchiveFolder != "" {
		fmt.Fprintf(&b, "  carpeta archivada: %s\n", report.ArchiveFolder)
	}
	if report.Discrepancy != "" {
		fmt.Fprintf(&b, "  discrepancia: %s\n", report.Discrepancy)
	}
	if len(report.Chain) == 0 {
		fmt.Fprintf(&b, "Cadena requisito, Cubre:, matriz: sin datos (artefactos: %s)\n", report.ChainSource)
	} else {
		fmt.Fprintf(&b, "Cadena requisito, Cubre:, matriz (artefactos: %s):\n", report.ChainSource)
		for _, entry := range report.Chain {
			fmt.Fprintf(&b, "  %s", entry.Requirement)
			if entry.Name != "" {
				fmt.Fprintf(&b, " (%s %s, dominio %s)", entry.Section, entry.Name, entry.Domain)
			}
			b.WriteString("\n")
			for _, cubre := range entry.Cubre {
				fmt.Fprintf(&b, "    Cubre: %s:%d\n", cubre.File, cubre.Line)
			}
			for _, row := range entry.Rows {
				fmt.Fprintf(&b, "    fila: %s | %s | %s | %s\n", row.Scenario, row.Type, row.Reference, row.Result)
			}
		}
	}
	_, err := io.WriteString(stdout, b.String())
	return err
}

func renderSDDTraceHelp(stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "Uso: axiom sdd trace --cwd <dir> --change <cambio> [--integration-branch <ref>] [--json]")
	_, _ = fmt.Fprintln(stdout, "Deriva, sin escribir nada, la traza del cambio con `git log --first-parent`: el commit de integración del código,")
	_, _ = fmt.Fprintln(stdout, "el commit de archivado y la cadena requisito, Cubre:, fila de la matriz. Es independiente de la estrategia de fusión")
	_, _ = fmt.Fprintln(stdout, "(squash, merge o rebase) y la contrasta con la evidencia pr_merged de la compuerta integration.")
	_, _ = fmt.Fprintln(stdout, "  --integration-branch  Rama de integración; por defecto la del kickoff o la de axiom.yaml (main).")
	_, _ = fmt.Fprintln(stdout, "  --json                Salida JSON ("+traceSchema+").")
	return nil
}
