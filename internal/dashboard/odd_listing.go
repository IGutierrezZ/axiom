package dashboard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Listado de las features ODD en el tablero de Incrementos. Una feature ODD es un documento vivo
// `odd/tasks/<feature>.md`. Las que están en curso viven en un worktree registrado y se detectan por
// su contenido (el documento difiere de la base de integración); las del checkout principal ya están
// integradas y se listan como archivadas. La lectura de git la hace engine.ODDWorktrees.

// oddDoc es lo que el listado necesita de un documento ODD.
type oddDoc struct {
	Title     string
	Total     int
	Completed int
}

var (
	oddCheckboxLine    = regexp.MustCompile(`^- \[([ xX])\]`)
	oddCheckboxHeading = regexp.MustCompile(`^#{2,6}\s+\[([ xX])\]`)
	oddTitlePrefix     = regexp.MustCompile(`(?i)^ODD\s*[:·\-–—]\s*`)
)

// parseODDDoc extrae el título y el progreso de un documento ODD. El título es el primer encabezado
// `# …` sin el prefijo `ODD:` / `ODD ·`. Las tareas son las casillas `- [ ]` / `- [x]` sin sangría y los
// encabezados `### [x] T1 · …`; las casillas con sangría son subpasos y solo cuentan si el documento no
// tiene ninguna de primer nivel.
func parseODDDoc(content string) oddDoc {
	var doc oddDoc
	var nestedTotal, nestedCompleted int
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line = strings.TrimRight(line, " \t")
		if doc.Title == "" && strings.HasPrefix(line, "# ") {
			doc.Title = strings.TrimSpace(oddTitlePrefix.ReplaceAllString(strings.TrimPrefix(line, "# "), ""))
			continue
		}
		match := oddCheckboxLine.FindStringSubmatch(line)
		if match == nil {
			match = oddCheckboxHeading.FindStringSubmatch(line)
		}
		if match != nil {
			doc.Total++
			if match[1] != " " {
				doc.Completed++
			}
			continue
		}
		if nested := oddCheckboxLine.FindStringSubmatch(strings.TrimLeft(line, " \t")); nested != nil {
			nestedTotal++
			if nested[1] != " " {
				nestedCompleted++
			}
		}
	}
	if doc.Total == 0 {
		doc.Total, doc.Completed = nestedTotal, nestedCompleted
	}
	return doc
}

// oddPhase es la fase de una feature ODD en curso en las columnas del tablero: sin tareas, `explore`;
// con tareas pendientes, `apply`; con todas marcadas, `verify`.
func oddPhase(doc oddDoc) string {
	switch {
	case doc.Total == 0:
		return "explore"
	case doc.Completed < doc.Total:
		return "apply"
	}
	return "verify"
}

func oddSummary(name, kind, phase string, doc oddDoc) IncrementSummaryDTO {
	pct := 0
	if doc.Total > 0 {
		pct = doc.Completed * 100 / doc.Total
	}
	return IncrementSummaryDTO{
		Name: name, Type: kind, Phase: phase, TasksTotal: doc.Total, TasksCompleted: doc.Completed,
		ProgressPct: pct, ChangeType: "odd", Title: doc.Title,
	}
}

func readODDDoc(path string) (oddDoc, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return oddDoc{}, false
	}
	return parseODDDoc(string(data)), true
}

// oddIncrements devuelve las features ODD: las activas de los worktrees registrados y, como
// archivadas, las del checkout principal que no estén activas en un worktree (el worktree prevalece,
// como en SDD). También devuelve los avisos: el de cada worktree ya integrado y los del motor.
func oddIncrements(loc workspace.Location) ([]IncrementSummaryDTO, []string) {
	var list []IncrementSummaryDTO
	var notes []string
	active := map[string]bool{}

	worktrees, warnings := engine.Default().ODDWorktrees(context.Background(), loc)
	notes = append(notes, warnings...)
	for _, worktree := range worktrees {
		if worktree.Integrated {
			notes = append(notes, fmt.Sprintf(
				"El worktree %s (rama %s) ya está integrado en %s; ciérralo con «axiom branch done --change %s».",
				worktree.Change, worktree.Branch, worktree.Base, worktree.Change))
			continue
		}
		for _, change := range worktree.Docs {
			doc, ok := readODDDoc(change.Path)
			if !ok || active[change.Name] {
				continue
			}
			active[change.Name] = true
			summary := oddSummary(change.Name, "active", oddPhase(doc), doc)
			summary.WorktreePath = worktree.Root
			summary.Branch = worktree.Branch
			list = append(list, summary)
		}
	}

	mainRoot, ok := gitcommon.GitRootFromFiles(loc.WorkspaceRoot)
	if !ok {
		mainRoot = loc.Master
	}
	entries, err := os.ReadDir(filepath.Join(mainRoot, filepath.FromSlash(engine.ODDTasksDir)))
	if err != nil {
		return list, dedupeNotes(notes)
	}
	for _, entry := range entries {
		name, isDoc := strings.CutSuffix(entry.Name(), ".md")
		if entry.IsDir() || !isDoc || active[name] {
			continue
		}
		doc, ok := readODDDoc(filepath.Join(mainRoot, filepath.FromSlash(engine.ODDTasksDir), entry.Name()))
		if !ok {
			continue
		}
		list = append(list, oddSummary(name, "archived", "archive", doc))
	}
	return list, dedupeNotes(notes)
}

func dedupeNotes(notes []string) []string {
	var unique []string
	for _, note := range notes {
		if !slices.Contains(unique, note) {
			unique = append(unique, note)
		}
	}
	return unique
}
