package reqtrace

import (
	"errors"
	"io/fs"
	"sort"

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
