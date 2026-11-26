package reqtrace

import (
	"context"
	"fmt"
	"sort"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
)

// RefError es el error de una rama de integración que no se puede leer en el repositorio de
// una fuente. Err conserva la causa de `git ls-tree` para distinguir una referencia
// inexistente de otros fallos.
type RefError struct {
	Lane Lane
	// Repo identifica el repositorio de la fuente (Source.Repo).
	Repo string
	Ref  string
	Err  error
}

func (e *RefError) Error() string {
	repo := ""
	if e.Repo != "" {
		repo = " en el repositorio " + e.Repo
	}
	return fmt.Sprintf("no se puede leer la rama de integración %q (fuente %s)%s: %v", e.Ref, e.Lane, repo, e.Err)
}

// Unwrap devuelve la causa del fallo de git.
func (e *RefError) Unwrap() error { return e.Err }

// checkRef comprueba la referencia en el repositorio de la fuente con `TreeHas` sobre su
// directorio (`git ls-tree` falla con una referencia inexistente y no con una ruta ausente).
// Una fuente ausente o sin historia no se comprueba.
func checkRef(ctx context.Context, ref string, lane Lane, src Source, path string) error {
	if src.Files == nil || src.History == nil {
		return nil
	}
	if path == "" {
		path = "."
	}
	if _, err := src.History.TreeHas(ctx, ref, path); err != nil {
		return &RefError{Lane: lane, Repo: src.Repo, Ref: ref, Err: err}
	}
	return nil
}

// sddAnchor son los commits de una carpeta archivada.
type sddAnchor struct {
	integration Commit
	archive     Commit
}

// anchorSDD ancla una carpeta archivada con el algoritmo de DeriveTrace (decisión 6): el
// commit de archivado es el primero que añade `kickoff.yaml` de la carpeta (o la carpeta); el
// de código, el más reciente que añade `verify-report.md` con `tasks*.md` en el árbol y que
// es ancestro del archivado. Sin código, el de archivado ancla el incremento; sin ninguno,
// found es false. Dos carpetas con el mismo nombre de cambio toman cada una su commit.
func anchorSDD(ctx context.Context, h isoevidence.History, ref, prefix string, c candidate) (sddAnchor, error) {
	var anchor sddAnchor
	commits, err := h.FirstParentAdded(ctx, ref, joinPath(c.Path, "kickoff.yaml"))
	if err == nil && len(commits) == 0 {
		commits, err = h.FirstParentAdded(ctx, ref, c.Path)
	}
	if err != nil {
		return anchor, fmt.Errorf("no se pudo leer el historial de la carpeta archivada %s: %w", c.Path, err)
	}
	if len(commits) > 0 {
		anchor.archive = Commit{Commit: commits[0], Found: true}
	}

	changeDir := joinPath(prefix, "changes/"+c.change)
	candidates, err := h.FirstParentAdded(ctx, ref, joinPath(changeDir, "verify-report.md"))
	if err != nil {
		return anchor, fmt.Errorf("no se pudo leer el historial de verify-report.md de %s: %w", c.change, err)
	}
	for i := len(candidates) - 1; i >= 0; i-- {
		commit := candidates[i]
		entries, err := h.TreeEntries(ctx, commit, changeDir)
		if err != nil {
			return anchor, fmt.Errorf("no se pudo leer el árbol del commit %s: %w", commit, err)
		}
		if !hasTasksFile(entries) {
			continue
		}
		if anchor.archive.Found {
			ok, err := h.IsAncestor(ctx, commit, anchor.archive.Commit)
			if err != nil {
				return anchor, fmt.Errorf("no se pudo comparar el commit %s con el de archivado: %w", commit, err)
			}
			if !ok {
				continue
			}
		}
		anchor.integration = Commit{Commit: commit, Found: true, Anchor: AnchorCode}
		break
	}
	if !anchor.integration.Found && anchor.archive.Found {
		anchor.integration = Commit{Commit: anchor.archive.Commit, Found: true, Anchor: AnchorArchive}
	}
	return anchor, nil
}

func hasTasksFile(entries []string) bool {
	for _, e := range entries {
		if tasksFileName.MatchString(e) {
			return true
		}
	}
	return false
}

// anchorCandidates ancla cada incremento con la historia de su propia fuente (prov:trace:14).
// Un incremento SDD sin commit encontrado se informa con found false. Un documento ODD que la
// línea `--first-parent` no contiene se omite con odd_not_integrated; sin historia ODD se
// omiten todos con odd_repository_unavailable.
func anchorCandidates(ctx context.Context, ref string, sdd, odd Source, cands []candidate) ([]Increment, []Warning, error) {
	var out []Increment
	var warnings []Warning
	sddCache := map[string]sddAnchor{}
	oddCache := map[string]Commit{}
	notIntegrated := map[string]bool{}
	for _, c := range cands {
		inc := c.Increment
		switch c.Lane {
		case LaneSDD:
			archive := Commit{}
			if sdd.History != nil {
				anchor, ok := sddCache[c.Path]
				if !ok {
					var err error
					if anchor, err = anchorSDD(ctx, sdd.History, ref, sdd.GitPrefix, c); err != nil {
						return nil, nil, err
					}
					sddCache[c.Path] = anchor
				}
				inc.Integration, archive = anchor.integration, anchor.archive
			}
			inc.Archive = &archive
		case LaneODD:
			if odd.History == nil {
				continue
			}
			commit, ok := oddCache[c.Path]
			if !ok {
				commits, err := odd.History.FirstParentAdded(ctx, ref, c.Path)
				if err != nil {
					return nil, nil, fmt.Errorf("no se pudo leer el historial de %s: %w", c.Path, err)
				}
				if len(commits) > 0 {
					commit = Commit{Commit: commits[0], Found: true, Anchor: AnchorDocument}
				}
				oddCache[c.Path] = commit
			}
			if !commit.Found {
				if !notIntegrated[c.Path] {
					notIntegrated[c.Path] = true
					warnings = append(warnings, Warning{
						Code: WarnODDNotIntegrated, Path: c.Path,
						Message: fmt.Sprintf("el documento %s cita el ID pero la línea --first-parent de %s no lo contiene; se omite: intégralo en esa rama o indica otra con --integration-branch", c.Path, ref),
					})
				}
				continue
			}
			inc.Integration = commit
		}
		out = append(out, inc)
	}
	return out, warnings, nil
}

// lessKey es la clave estable de desempate: carril, ruta y rol, en orden de bytes y de roles.
func lessKey(a, b Increment) bool {
	if a.Lane != b.Lane {
		return a.Lane < b.Lane
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return roleRank(a.Role) < roleRank(b.Role)
}

// sortIncrements ordena los incrementos (decisión 9): primero los de commit encontrado por su
// posición en la línea `--first-parent`, con el comparador a ≺ b ⇔ a ≠ b ∧ IsAncestor(a, b)
// (con memoria y propagando los errores de git); los empates y los found false, al final, por
// la clave (carril, ruta, rol). Los incrementos de repositorios distintos (Source.Repo) no
// comparten historia y solo se comparan por la clave estable.
func sortIncrements(ctx context.Context, sdd, odd Source, incs []Increment) ([]Increment, error) {
	sort.SliceStable(incs, func(i, j int) bool { return lessKey(incs[i], incs[j]) })
	var found, missing []Increment
	for _, inc := range incs {
		if inc.Integration.Found {
			found = append(found, inc)
		} else {
			missing = append(missing, inc)
		}
	}

	sourceOf := func(l Lane) Source {
		if l == LaneODD {
			return odd
		}
		return sdd
	}
	type pair struct{ a, b string }
	memo := map[pair]bool{}
	var failure error
	before := func(a, b Increment) bool {
		sa, sb := sourceOf(a.Lane), sourceOf(b.Lane)
		if sa.Repo != sb.Repo || a.Integration.Commit == b.Integration.Commit || sa.History == nil {
			return lessKey(a, b)
		}
		key := pair{a.Integration.Commit, b.Integration.Commit}
		if v, ok := memo[key]; ok {
			return v
		}
		ok, err := sa.History.IsAncestor(ctx, a.Integration.Commit, b.Integration.Commit)
		if err != nil && failure == nil {
			failure = fmt.Errorf("no se pudo ordenar %s y %s por la línea --first-parent: %w", a.Integration.Commit, b.Integration.Commit, err)
		}
		memo[key] = ok
		return ok
	}
	for i := 1; i < len(found) && failure == nil; i++ {
		for j := i; j > 0 && before(found[j], found[j-1]); j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	if failure != nil {
		return nil, failure
	}
	return append(found, missing...), nil
}

// sortWarnings ordena las advertencias por ruta, código y mensaje.
func sortWarnings(ws []Warning) {
	sort.SliceStable(ws, func(i, j int) bool {
		if ws[i].Path != ws[j].Path {
			return ws[i].Path < ws[j].Path
		}
		if ws[i].Code != ws[j].Code {
			return ws[i].Code < ws[j].Code
		}
		return ws[i].Message < ws[j].Message
	})
}
