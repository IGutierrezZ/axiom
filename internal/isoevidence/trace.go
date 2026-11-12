package isoevidence

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// History es el puerto de solo lectura sobre el historial de la rama de
// integración (REQ-29.9). Las implementaciones leen con `git log
// --first-parent`, `git ls-tree` y `git merge-base --is-ancestor`; este paquete
// no ejecuta ninguna orden.
type History interface {
	// FirstParentAdded devuelve, del más antiguo al más reciente, los commits
	// de la línea `--first-parent` de ref en los que aparece path (un fichero, o
	// un directorio con algún fichero dentro) sin existir en el commit anterior
	// de esa línea. La implementación debe desactivar la detección de
	// renombrados, para que una carpeta archivada con `git mv` cuente como
	// añadida. Sin coincidencias devuelve una lista vacía y no un error.
	FirstParentAdded(ctx context.Context, ref, path string) ([]string, error)
	// TreeHas dice si path existe en el árbol de commit.
	TreeHas(ctx context.Context, commit, path string) (bool, error)
	// TreeEntries devuelve los nombres (no las rutas) de las entradas
	// inmediatas del directorio dir en el árbol de ref, que puede ser una rama
	// o un commit. Un directorio ausente devuelve una lista vacía.
	TreeEntries(ctx context.Context, ref, dir string) ([]string, error)
	// IsAncestor dice si ancestor es el propio commit descendant o un ancestro
	// suyo.
	IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error)
}

// TraceInput son los datos de una derivación. Change y PlanningHomePrefix los
// valida quien llama; aquí se rechazan además los que no pueden formar una
// ruta de git segura.
type TraceInput struct {
	Change string
	Ref    string
	// PlanningHomePrefix es la ruta del planning home relativa a la raíz git,
	// con `/` (por ejemplo `openspec`).
	PlanningHomePrefix string
	// EvidenceCommit es el commit de la evidencia `pr_merged` de la compuerta
	// `integration`; vacío si no hay evidencia.
	EvidenceCommit string
}

// CommitRef es un commit derivado, o la constancia de que no existe todavía.
type CommitRef struct {
	Commit string
	Found  bool
}

// Trace es el resultado de la derivación: los dos commits de la línea
// principal y la discrepancia con la evidencia, si la hay.
type Trace struct {
	CodeCommit    CommitRef
	ArchiveCommit CommitRef
	// ArchiveFolder es el nombre `<fecha>-<cambio>` de la carpeta archivada, o
	// vacío si no existe todavía en la rama.
	ArchiveFolder string
	// Discrepancy describe un desacuerdo con la evidencia `pr_merged`; vacío si
	// coincide o no hay evidencia. Nunca es un error de ejecución.
	Discrepancy string
}

// ErrInvalidTraceInput se devuelve cuando la entrada no puede formar una ruta
// de git segura. Envuelve el motivo.
var ErrInvalidTraceInput = errors.New("entrada de traza no válida")

// DeriveTrace deriva, con el algoritmo de design §6.4 y sin estado persistido,
// el commit de integración del código (el primero de la línea `--first-parent`
// que incorpora la carpeta del cambio con sus `tasks*.md` y su
// `verify-report.md`) y el commit de archivado (el primero que incorpora la
// carpeta `changes/archive/<fecha>-<cambio>/`). La ausencia de cualquiera de
// los dos se informa sin error.
func DeriveTrace(ctx context.Context, h History, in TraceInput) (Trace, error) {
	if err := validateTraceInput(in); err != nil {
		return Trace{}, err
	}
	changesDir := joinPath(in.PlanningHomePrefix, "changes")
	changeDir := joinPath(changesDir, in.Change)
	var trace Trace

	candidates, err := h.FirstParentAdded(ctx, in.Ref, joinPath(changeDir, "verify-report.md"))
	if err != nil {
		return Trace{}, fmt.Errorf("no se pudo leer el historial de verify-report.md: %w", err)
	}
	for _, commit := range candidates {
		entries, err := h.TreeEntries(ctx, commit, changeDir)
		if err != nil {
			return Trace{}, fmt.Errorf("no se pudo leer el árbol del commit %s: %w", commit, err)
		}
		if hasTasksFile(entries) {
			trace.CodeCommit = CommitRef{Commit: commit, Found: true}
			break
		}
	}

	folder, err := archivedFolder(ctx, h, in.Ref, changesDir, in.Change)
	if err != nil {
		return Trace{}, err
	}
	if folder != "" {
		trace.ArchiveFolder = folder
		folderDir := joinPath(joinPath(changesDir, "archive"), folder)
		commits, err := h.FirstParentAdded(ctx, in.Ref, joinPath(folderDir, "kickoff.yaml"))
		if err == nil && len(commits) == 0 {
			commits, err = h.FirstParentAdded(ctx, in.Ref, folderDir)
		}
		if err != nil {
			return Trace{}, fmt.Errorf("no se pudo leer el historial de la carpeta archivada: %w", err)
		}
		if len(commits) > 0 {
			trace.ArchiveCommit = CommitRef{Commit: commits[0], Found: true}
		}
	}

	if in.EvidenceCommit != "" {
		trace.Discrepancy = contrastEvidence(ctx, h, trace.CodeCommit, in.EvidenceCommit)
	}
	return trace, nil
}

// contrastEvidence contrasta el commit de código con la evidencia `pr_merged`:
// debe ser ese commit o un ancestro suyo. Un desacuerdo, o no poder
// comprobarlo, se informa como discrepancia y nunca como error.
func contrastEvidence(ctx context.Context, h History, code CommitRef, evidence string) string {
	if !code.Found {
		return fmt.Sprintf("la compuerta integration registra el commit %s como evidencia pr_merged, pero no se derivó ningún commit de código en la línea --first-parent", evidence)
	}
	ok, err := h.IsAncestor(ctx, code.Commit, evidence)
	switch {
	case err != nil:
		return fmt.Sprintf("no se pudo contrastar el commit de código %s con la evidencia pr_merged %s: %v", code.Commit, evidence, err)
	case !ok:
		return fmt.Sprintf("el commit de código derivado %s no es el de la evidencia pr_merged %s ni un ancestro suyo", code.Commit, evidence)
	}
	return ""
}

// archivedFolder busca `<fecha>-<cambio>` entre las carpetas archivadas. Con
// varias (un nombre de cambio reutilizado) devuelve la más reciente.
func archivedFolder(ctx context.Context, h History, ref, changesDir, change string) (string, error) {
	entries, err := h.TreeEntries(ctx, ref, joinPath(changesDir, "archive"))
	if err != nil {
		return "", fmt.Errorf("no se pudo leer la carpeta de archivo: %w", err)
	}
	pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-` + regexp.QuoteMeta(change) + `$`)
	var matches []string
	for _, e := range entries {
		if pattern.MatchString(e) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 0 {
		return "", nil
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

var roleTasksFile = regexp.MustCompile(`^tasks\..+\.md$`)

// hasTasksFile dice si entries contiene `tasks.md` o `tasks.<rol>.md`.
func hasTasksFile(entries []string) bool {
	for _, e := range entries {
		if e == "tasks.md" || roleTasksFile.MatchString(e) {
			return true
		}
	}
	return false
}

func joinPath(base, elem string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return elem
	}
	return base + "/" + elem
}

// validateTraceInput rechaza lo que no puede formar una ruta o una referencia
// de git segura: es una defensa en profundidad, porque quien llama ya valida el
// nombre del cambio y la referencia.
func validateTraceInput(in TraceInput) error {
	bad := func(what, value, why string) error {
		return fmt.Errorf("%w: %s %q %s", ErrInvalidTraceInput, what, value, why)
	}
	switch {
	case in.Change == "":
		return bad("el cambio", in.Change, "está vacío")
	case strings.ContainsAny(in.Change, `/\:*?[]`+"\x00") || strings.Contains(in.Change, "..") || strings.HasPrefix(in.Change, "-"):
		return bad("el cambio", in.Change, "no es un nombre de carpeta simple")
	case in.Ref == "":
		return bad("la referencia", in.Ref, "está vacía")
	case strings.HasPrefix(in.Ref, "-") || strings.Contains(in.Ref, "..") || strings.ContainsAny(in.Ref, " \t\x00"):
		return bad("la referencia", in.Ref, "no es un nombre de referencia seguro")
	case strings.HasPrefix(in.PlanningHomePrefix, "/") || strings.Contains(in.PlanningHomePrefix, "..") ||
		strings.ContainsAny(in.PlanningHomePrefix, `\:*?[]`+"\x00"):
		return bad("el planning home", in.PlanningHomePrefix, "no es una ruta relativa con `/` dentro del repositorio")
	}
	return nil
}
