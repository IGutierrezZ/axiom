package reqtrace

import (
	"errors"
	"fmt"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// roleOrder es el orden estable de los roles.
var roleOrder = []Role{RoleCreated, RoleModified, RoleRemoved, RoleRenamed, RoleCovered}

func roleRank(r Role) int {
	for i, role := range roleOrder {
		if role == r {
			return i
		}
	}
	return len(roleOrder)
}

// candidate es un incremento todavía sin anclar.
type candidate struct {
	Increment
	// change es el nombre del cambio SDD (sin fecha) o la feature ODD.
	change string
}

// roleHit son los IDs, las capacidades y las tareas con las que una fuente toca el requisito
// en un rol.
type roleHit struct {
	ids   map[string]bool
	caps  map[string]bool
	tasks []Task
}

// hits agrupa las apariciones de una fuente por rol.
type hits map[Role]*roleHit

func (h hits) record(role Role, ids []string, capability string) {
	hit := h[role]
	if hit == nil {
		hit = &roleHit{ids: map[string]bool{}, caps: map[string]bool{}}
		h[role] = hit
	}
	for _, id := range ids {
		hit.ids[id] = true
	}
	if capability != "" {
		hit.caps[capability] = true
	}
}

func (h hits) addTask(role Role, ids []string, task Task) {
	h.record(role, ids, "")
	h[role].tasks = append(h[role].tasks, task)
}

// candidates convierte las apariciones en un incremento por rol, en orden de roles, con los
// IDs, las capacidades y las tareas ordenados y sin repetidos.
func (h hits) candidates(base candidate) []candidate {
	var out []candidate
	for _, role := range roleOrder {
		hit := h[role]
		if hit == nil {
			continue
		}
		c := base
		c.Role = role
		c.IDs = sortedKeys(hit.ids)
		c.Capabilities = sortedKeys(hit.caps)
		c.Tasks = sortedTasks(hit.tasks)
		out = append(out, c)
	}
	return out
}

func sortedTasks(tasks []Task) []Task {
	seen := map[Task]bool{}
	var out []Task
	for _, t := range tasks {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// idTokens devuelve los identificadores del texto (válidos y provisionales) que coinciden
// exactamente con el conjunto: `REQ-1.1` nunca casa con `REQ-1.10` ni con `REQ-11.1`.
func idTokens(text string, p reqid.Pattern, match map[string]bool) []string {
	valid, provisional, _ := isoevidence.RequirementKeys(text, p)
	var out []string
	for _, token := range append(valid, provisional...) {
		if match[token] {
			out = append(out, token)
		}
	}
	return out
}

// classifyDelta anota en h los requisitos de la delta de una capacidad que citan el ID: un
// título ADDED es created; MODIFIED, REMOVED y RENAMED, sus roles (en RENAMED, tanto el
// nombre origen como el destino). Devuelve si hubo alguna aparición.
func classifyDelta(h hits, capability string, d openspecdoc.Delta, p reqid.Pattern, match map[string]bool) bool {
	found := false
	apply := func(role Role, reqs []openspecdoc.Requirement) {
		for _, r := range reqs {
			var ids []string
			names := []string{r.Name}
			if role == RoleRenamed {
				if oldName, newName, ok := openspecdoc.SplitRename(r.Name); ok {
					names = []string{oldName, newName}
				}
			}
			for _, name := range names {
				ids = append(ids, idTokens(name, p, match)...)
			}
			if len(ids) > 0 {
				h.record(role, ids, capability)
				found = true
			}
		}
	}
	apply(RoleCreated, d.Added)
	apply(RoleModified, d.Modified)
	apply(RoleRemoved, d.Removed)
	apply(RoleRenamed, d.Renamed)
	return found
}

// classifyCubre anota en h las líneas `Cubre:` de un texto que citan el ID.
func classifyCubre(h hits, file, text string, p reqid.Pattern, match map[string]bool) {
	for _, line := range isoevidence.ParseCubre(file, text) {
		var ids []string
		for _, raw := range line.IDs {
			ids = append(ids, idTokens(raw, p, match)...)
		}
		if len(ids) > 0 {
			h.addTask(RoleCovered, ids, Task{File: line.File, Line: line.Line})
		}
	}
}

// classifySDD clasifica una carpeta archivada: roles por título de requisito de su delta,
// covered por las líneas `Cubre:` de sus tasks*.md, created por una entrada de su ids.yaml
// sin aparición en la delta, y el tipo por la declaración de bug de su proposal.md. Un
// fichero ausente cuenta como vacío; uno ilegible se informa y la búsqueda sigue.
func classifySDD(src Source, p reqid.Pattern, eq equivalence, f sddFolder) ([]candidate, []Warning) {
	match := eq.matchSet(f.Path)
	h := hits{}
	var warnings []Warning

	deltaFound := false
	specsDir := joinPath(f.Dir, "specs")
	capabilities, w := readDirOrEmpty(src.Files, specsDir, joinPath(f.Path, "specs"))
	warnings = append(warnings, w...)
	for _, capDir := range capabilities {
		if !capDir.IsDir() {
			continue
		}
		specGit := joinPath(joinPath(joinPath(f.Path, "specs"), capDir.Name()), "spec.md")
		data, err := fs.ReadFile(src.Files, joinPath(joinPath(specsDir, capDir.Name()), "spec.md"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			warnings = append(warnings, unreadable(specGit, err))
			continue
		}
		delta, _ := openspecdoc.ParseChangeSpec(string(data))
		if classifyDelta(h, capDir.Name(), delta, p, match) {
			deltaFound = true
		}
	}

	entries, w := readDirOrEmpty(src.Files, f.Dir, f.Path)
	warnings = append(warnings, w...)
	for _, entry := range entries {
		if entry.IsDir() || !tasksFileName.MatchString(entry.Name()) {
			continue
		}
		fileGit := joinPath(f.Path, entry.Name())
		data, err := fs.ReadFile(src.Files, joinPath(f.Dir, entry.Name()))
		if err != nil {
			warnings = append(warnings, unreadable(fileGit, err))
			continue
		}
		classifyCubre(h, fileGit, string(data), p, match)
	}

	if !deltaFound {
		for _, entry := range eq.touched[f.Path] {
			h.record(RoleCreated, []string{entry.Definitive}, entry.Capability)
		}
	}
	if len(h) == 0 {
		return nil, warnings
	}

	kind := KindIncrement
	if data, err := fs.ReadFile(src.Files, joinPath(f.Dir, "proposal.md")); err == nil && isoevidence.ParseBugDeclaration(string(data)).Kind != isoevidence.BugKindNone {
		kind = KindBug
	}
	base := candidate{
		Increment: Increment{Lane: LaneSDD, Name: f.Change, Path: f.Path, Type: kind, Assignments: eq.assignments(f.Path)},
		change:    f.Change,
	}
	return h.candidates(base), warnings
}

// classifyODD clasifica un documento ODD: con ParseFeatureDoc correcto, roles por la delta de
// cada `### Capacidad:`, covered por las líneas `Cubre:` y created por una entrada de la
// traza sin aparición en la delta; el tipo es bug con `Tipo: bug`. Si el parseo estricto
// falla y el documento cita el ID, añade odd_unparseable y busca en el texto.
func classifyODD(p reqid.Pattern, eq equivalence, d oddDoc) ([]candidate, []Warning) {
	match := eq.matchSet(d.Path)
	h := hits{}
	var warnings []Warning
	kind := KindIncrement
	deltaFound := false

	doc, err := livespec.ParseFeatureDoc(d.Name, []byte(d.Text))
	if err == nil {
		if doc.Kind == livespec.KindBug {
			kind = KindBug
		}
		for _, c := range doc.Capabilities {
			if classifyDelta(h, c.Name, c.Delta, p, match) {
				deltaFound = true
			}
		}
		classifyCubre(h, d.Path, d.Text, p, match)
	} else {
		mentioned := len(idTokens(d.Text, p, match)) > 0
		if mentioned {
			warnings = append(warnings, Warning{
				Code: WarnODDUnparseable, Path: d.Path,
				Message: fmt.Sprintf("el documento %s no cumple el formato de feature (%s); se busca el ID en su texto: corrígelo para una traza fiable", d.Path, firstLine(err.Error())),
			})
		}
		kind, deltaFound = tolerantODD(h, d, p, match)
		if mentioned && len(h) == 0 && len(eq.touched[d.Path]) == 0 {
			warnings = append(warnings, Warning{
				Code: WarnUnclassifiedMention, Path: d.Path,
				Message: fmt.Sprintf("el documento %s cita el ID solo en prosa, fuera de un título de requisito o de una línea Cubre:; no se clasifica ningún incremento", d.Path),
			})
		}
	}
	if !deltaFound {
		for _, entry := range eq.touched[d.Path] {
			h.record(RoleCreated, []string{entry.Definitive}, entry.Capability)
		}
	}
	if len(h) == 0 {
		return nil, warnings
	}
	base := candidate{
		Increment: Increment{Lane: LaneODD, Name: d.Name, Path: d.Path, Type: kind, Assignments: eq.assignments(d.Path)},
		change:    d.Name,
	}
	return h.candidates(base), warnings
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

var (
	oddBugLine        = regexp.MustCompile(`^Tipo:[ \t]*bug[ \t]*$`)
	oddCapabilityLine = regexp.MustCompile(`^### Capacidad:[ \t]*(\S+)[ \t]*$`)
)

// tolerantODD es la búsqueda textual de un documento que no parsea (decisión 4): la delta de
// cada bloque vallado `openspec` (o el texto entero si no hay bloques), las líneas `Cubre:` y
// el tipo por una línea `Tipo: bug` antes del primer `## `. Devuelve el tipo y si hubo
// aparición en una delta.
func tolerantODD(h hits, d oddDoc, p reqid.Pattern, match map[string]bool) (Kind, bool) {
	type block struct {
		capability string
		text       string
	}
	var blocks []block
	kind := KindIncrement
	seenSection := false
	capability := ""
	inFence, fenceChar, fenceLen := false, byte(0), 0
	inBlock := false
	var current []string
	for _, line := range strings.Split(strings.ReplaceAll(d.Text, "\r\n", "\n"), "\n") {
		ch, n, hasInfo, ok := openspecdoc.FenceMarker(line)
		switch {
		case !inFence && ok:
			inFence, fenceChar, fenceLen = true, ch, n
			fields := strings.Fields(line[n:])
			inBlock = len(fields) > 0 && fields[0] == "openspec"
			current = nil
		case inFence && ok && ch == fenceChar && n >= fenceLen && !hasInfo:
			if inBlock {
				blocks = append(blocks, block{capability: capability, text: strings.Join(current, "\n") + "\n"})
			}
			inFence, inBlock = false, false
		case inFence:
			if inBlock {
				current = append(current, line)
			}
		case strings.HasPrefix(line, "## "):
			seenSection = true
		case !seenSection && oddBugLine.MatchString(line):
			kind = KindBug
		default:
			if m := oddCapabilityLine.FindStringSubmatch(line); m != nil {
				capability = m[1]
			}
		}
	}
	if inBlock {
		blocks = append(blocks, block{capability: capability, text: strings.Join(current, "\n") + "\n"})
	}
	if len(blocks) == 0 {
		blocks = []block{{text: d.Text}}
	}
	deltaFound := false
	for _, b := range blocks {
		if classifyDelta(h, b.capability, openspecdoc.ParseDelta(b.text), p, match) {
			deltaFound = true
		}
	}
	classifyCubre(h, d.Path, d.Text, p, match)
	return kind, deltaFound
}
