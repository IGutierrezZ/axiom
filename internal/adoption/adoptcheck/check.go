package adoptcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Reglas de Check. Todas se acumulan: el informe lista cada fallo, no solo el primero.
const (
	RuleFormat           = "format"
	RuleUniqueness       = "uniqueness"
	RuleSeries           = "series"
	RuleScenarios        = "scenarios"
	RuleOrigin           = "origin"
	RuleCoverage         = "coverage"
	RuleDiscardReason    = "discard_reason"
	RuleInventoryMissing = "inventory_missing"
	RuleInventoryStale   = "inventory_stale"
)

// Failure es un fallo tipado: la regla incumplida, el requisito o la unidad que la
// incumple y el detalle con la continuación.
type Failure struct {
	Rule    string `json:"rule"`
	Subject string `json:"subject"`
	Detail  string `json:"detail"`
}

// CheckReport es el informe de `axiom archive adopt check`.
type CheckReport struct {
	Change   string    `json:"change"`
	Passed   bool      `json:"passed"`
	Failures []Failure `json:"failures"`
}

var scenarioHeading = regexp.MustCompile(`^####[ \t]+Scenario:`)

var fenceLine = regexp.MustCompile("^ {0,3}(```+|~~~+)")

// addedRequirement es un requisito ADDED con la delta de la que sale.
type addedRequirement struct {
	req  openspecdoc.Requirement
	file string // specs/<capacidad>/spec.md, relativo a la carpeta del cambio
}

// Check verifica de forma determinista, sin modelo y sin escribir, el cambio de adopción
// de loc: formato, unicidad, serie, escenarios, procedencia, cobertura del inventario y
// motivos de los descartes (REQ-30.11 y REQ-30.12). No decide nada de la compuerta
// `spec` ni juzga la fidelidad semántica de la migración (REQ-30.13).
func Check(loc workspace.Location, change string) (CheckReport, error) {
	changeRoot := filepath.Join(loc.PlanningHome, "changes", change)
	if info, err := os.Stat(changeRoot); err != nil || !info.IsDir() {
		return CheckReport{}, fmt.Errorf("no existe el cambio activo %q en %s; comprueba el nombre con `axiom sdd status --cwd %s`",
			change, pathquote.Quote(filepath.Join(loc.PlanningHome, "changes")), pathquote.Quote(loc.WorkspaceRoot))
	}
	scanCommand := fmt.Sprintf("axiom archive adopt scan --change %s", pathquote.ShellWord(change))
	c := &checker{loc: loc, change: change, changeRoot: changeRoot, scanCommand: scanCommand}

	if err := c.loadDeltas(); err != nil {
		return CheckReport{}, err
	}
	if err := c.checkIdentifiers(); err != nil {
		return CheckReport{}, err
	}
	c.checkScenarios()
	covered := c.checkOrigins()
	discards, err := adoption.ReadDiscards(changeRoot)
	if err != nil {
		return CheckReport{}, err
	}
	c.checkCoverage(covered, discards)

	report := CheckReport{Change: change, Failures: c.failures}
	if report.Failures == nil {
		report.Failures = []Failure{}
	}
	report.Passed = len(report.Failures) == 0
	return report, nil
}

type checker struct {
	loc         workspace.Location
	change      string
	changeRoot  string
	scanCommand string
	added       []addedRequirement
	failures    []Failure
	units       map[string]map[string]bool // ruta → anclas existentes (nil si no se puede resolver)
}

func (c *checker) fail(rule, subject, format string, args ...any) {
	c.failures = append(c.failures, Failure{Rule: rule, Subject: subject, Detail: fmt.Sprintf(format, args...)})
}

// loadDeltas lee specs/<capacidad>/spec.md del cambio y aplica la regla de formato.
func (c *checker) loadDeltas() error {
	specsDir := filepath.Join(c.changeRoot, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no se puede leer %s: %w; comprueba los permisos de la carpeta", specsDir, err)
	}
	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		file := filepath.Join(specsDir, entry.Name(), "spec.md")
		data, err := os.ReadFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero", file, err)
		}
		found = true
		rel := "specs/" + entry.Name() + "/spec.md"
		delta := openspecdoc.ParseDelta(string(data))
		for _, section := range []struct {
			name string
			reqs []openspecdoc.Requirement
		}{{"MODIFIED", delta.Modified}, {"REMOVED", delta.Removed}, {"RENAMED", delta.Renamed}} {
			if len(section.reqs) > 0 {
				c.fail(RuleFormat, rel+" ## "+section.name+" Requirements",
					"la delta contiene `## %s Requirements` y un cambio de adopción solo admite `## ADDED Requirements`; mueve esos requisitos a ADDED o retíralos", section.name)
			}
		}
		for _, heading := range delta.Unknown {
			c.fail(RuleFormat, rel+" "+heading,
				"el encabezado `%s` no es una sección de delta; un cambio de adopción solo admite `## ADDED Requirements`", heading)
		}
		for _, r := range delta.Added {
			if r.ID == "" {
				c.fail(RuleFormat, r.Name, "el título del requisito (%s, línea %d) no lleva un identificador `(REQ-n.m)`; añádelo al título", rel, r.Line)
			}
			c.added = append(c.added, addedRequirement{req: r, file: rel})
		}
	}
	if !found {
		c.fail(RuleFormat, "specs/<capacidad>/spec.md", "el cambio no tiene ninguna delta en specs/<capacidad>/spec.md; redacta la delta de adopción con `## ADDED Requirements`")
	}
	return nil
}

// checkIdentifiers aplica las reglas de unicidad (REQ-30.11 b) y de serie (REQ-30.12).
func (c *checker) checkIdentifiers() error {
	byID := map[string][]addedRequirement{}
	var ids []string
	seriesSet := map[int]bool{}
	for _, a := range c.added {
		if a.req.ID == "" {
			continue
		}
		if len(byID[a.req.ID]) == 0 {
			ids = append(ids, a.req.ID)
		}
		byID[a.req.ID] = append(byID[a.req.ID], a)
		if series, ok := livespec.SeriesOf(a.req.ID); ok {
			seriesSet[series] = true
		}
	}

	all, err := livespec.LoadRegistry(c.loc.PlanningHome, livespec.RegistryOptions{ExcludeChange: c.change})
	if err != nil {
		return err
	}
	for _, id := range ids {
		if list := byID[id]; len(list) > 1 {
			var where []string
			for _, a := range list {
				where = append(where, fmt.Sprintf("«%s» (%s, línea %d)", a.req.Name, a.file, a.req.Line))
			}
			c.fail(RuleUniqueness, id, "el identificador se repite dentro del cambio: %s; asigna un identificador distinto a cada requisito", strings.Join(where, " y "))
		}
		for _, ref := range all.Lookup(id) {
			c.fail(RuleUniqueness, id, "el identificador ya existe en %s (%s); elige un número libre de la serie, por ejemplo con el siguiente de `openspec/specs/`", ref.Where(), filepath.ToSlash(ref.Path))
		}
	}

	var series []int
	for s := range seriesSet {
		series = append(series, s)
	}
	sort.Ints(series)
	if len(series) > 1 {
		names := make([]string, len(series))
		for i, s := range series {
			names[i] = fmt.Sprint(s)
		}
		c.fail(RuleSeries, "series", "el cambio mezcla las series %s; usa una única serie `REQ-n.*` nueva para todo el proyecto migrado", strings.Join(names, " y "))
	}
	if len(series) > 0 {
		specs, err := livespec.LoadRegistry(c.loc.PlanningHome, livespec.RegistryOptions{SpecsOnly: true})
		if err != nil {
			return err
		}
		for _, s := range series {
			caps := map[string]bool{}
			for _, ref := range specs.SeriesRefs(s) {
				caps[ref.Capability] = true
			}
			if len(caps) == 0 {
				continue
			}
			var names []string
			for name := range caps {
				names = append(names, "la capacidad "+name)
			}
			sort.Strings(names)
			c.fail(RuleSeries, fmt.Sprintf("serie %d", s), "la serie ya existe en openspec/specs/ (%s); elige un número de serie que no se use", strings.Join(names, ", "))
		}
	}
	return nil
}

func (c *checker) checkScenarios() {
	for _, a := range c.added {
		if countScenarios(a.req.Body) == 0 {
			c.fail(RuleScenarios, a.req.Name, "el requisito (%s, línea %d) no tiene ningún `#### Scenario:`; añade al menos un escenario", a.file, a.req.Line)
		}
	}
}

func countScenarios(body string) int {
	count, fence := 0, ""
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if m := fenceLine.FindStringSubmatch(line); m != nil {
			switch marker := m[1][:1]; {
			case fence == "":
				fence = marker
			case fence == marker:
				fence = ""
			}
			continue
		}
		if fence == "" && scenarioHeading.MatchString(line) {
			count++
		}
	}
	return count
}

// checkOrigins aplica la regla de procedencia (REQ-30.11 d) y devuelve las unidades que
// citan los `Origen:` (clave `<ruta>#<ancla>`).
func (c *checker) checkOrigins() map[string]bool {
	covered := map[string]bool{}
	c.units = map[string]map[string]bool{}
	for _, a := range c.added {
		origins := adoption.ParseOrigins(a.req.Body)
		if len(origins) == 0 {
			c.fail(RuleOrigin, a.req.Name, "el requisito (%s, línea %d) no tiene ninguna línea `Origen: <ruta>#<ancla> (<ID original>)`; añade al menos una", a.file, a.req.Line)
			continue
		}
		for _, o := range origins {
			rel := c.checkOrigin(a, o)
			if rel != "" {
				covered[unitKey(rel, o.Anchor)] = true
			}
		}
	}
	return covered
}

// checkOrigin resuelve un `Origen:` a un fichero y un ancla existentes. Devuelve la ruta
// normalizada (con `/`), o "" si ni siquiera la ruta resolvió.
func (c *checker) checkOrigin(a addedRequirement, o adoption.Origin) string {
	ref := o.Path
	if o.Anchor != "" {
		ref += "#" + o.Anchor
	}
	if strings.TrimSpace(o.Path) == "" {
		c.fail(RuleOrigin, a.req.Name, "una línea `Origen:` del requisito (%s) no indica ninguna ruta; usa `Origen: <ruta>#<ancla> (<ID original>)`", a.file)
		return ""
	}
	clean := filepath.Clean(filepath.FromSlash(o.Path))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		c.fail(RuleOrigin, ref, "el `Origen:` del requisito «%s» apunta fuera del workspace (%s); usa una ruta relativa dentro de %s", a.req.Name, o.Path, pathquote.Quote(c.loc.WorkspaceRoot))
		return ""
	}
	rel := filepath.ToSlash(clean)
	file, ok := insideWorkspace(c.loc.WorkspaceRoot, clean)
	if !ok {
		c.fail(RuleOrigin, ref, "el `Origen:` del requisito «%s»: la ruta %s no existe en el workspace (o queda fuera por un enlace simbólico); corrige la ruta", a.req.Name, rel)
		return ""
	}
	if o.Anchor == "" {
		return rel
	}
	anchors, known := c.units[rel]
	if !known {
		anchors = nil
		if ext := strings.ToLower(path.Ext(rel)); ext == ".md" || ext == ".markdown" || ext == ".txt" {
			if data, err := os.ReadFile(file); err == nil {
				anchors = map[string]bool{}
				for _, u := range adoption.Units(string(data), path.Base(rel)) {
					anchors[u.Anchor] = true
				}
			}
		}
		c.units[rel] = anchors
	}
	if anchors == nil {
		c.fail(RuleOrigin, ref, "el `Origen:` del requisito «%s» cita un ancla, pero %s es un fichero entero (o no se puede leer) y no admite ancla; quita `#%s`", a.req.Name, rel, o.Anchor)
		return rel
	}
	if !anchors[o.Anchor] {
		c.fail(RuleOrigin, ref, "el `Origen:` del requisito «%s»: el ancla `#%s` no existe en %s; copia el ancla del inventario (%s)", a.req.Name, o.Anchor, rel, adoption.InventoryFileName)
		return rel
	}
	return rel
}

// checkCoverage aplica la cobertura (REQ-30.11 e), los motivos de los descartes y la
// vigencia del inventario.
func (c *checker) checkCoverage(origins map[string]bool, discards []adoption.Discard) {
	discarded := map[string]bool{}
	for _, d := range discards {
		key := unitKey(normalizeRel(d.Path), d.Anchor)
		if strings.TrimSpace(d.Reason) != "" {
			discarded[key] = true
		}
	}

	inv, err := adoption.ReadInventory(c.changeRoot)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		c.fail(RuleInventoryMissing, adoption.InventoryFileName, "el cambio no tiene inventario de adopción; genéralo con `%s`", c.scanCommand)
	case err != nil:
		c.fail(RuleInventoryMissing, adoption.InventoryFileName, "%v; vuelve a generarlo con `%s`", err, c.scanCommand)
	default:
		for _, src := range inv.Sources {
			for _, u := range src.Units {
				key := unitKey(src.Path, u.Anchor)
				if origins[key] || discarded[key] {
					continue
				}
				c.fail(RuleCoverage, key, "la unidad «%s» del inventario no figura en ningún `Origen:` ni en `## Descartado` de %s con motivo; migra la unidad o descártala con motivo", u.Title, adoption.DiscardsFileName)
			}
		}
	}

	for _, d := range discards {
		if strings.TrimSpace(d.Reason) == "" {
			c.fail(RuleDiscardReason, unitKey(normalizeRel(d.Path), d.Anchor), "la unidad descartada (%s, línea %d) no tiene motivo; escribe `- <ruta>#<ancla>: <motivo>`", adoption.DiscardsFileName, d.Line)
		}
	}

	if err == nil {
		for _, src := range inv.Sources {
			file, ok := insideWorkspace(c.loc.WorkspaceRoot, filepath.FromSlash(src.Path))
			if !ok {
				c.fail(RuleInventoryStale, src.Path, "el origen del inventario ya no existe en el workspace; vuelve a generar el inventario con `%s`", c.scanCommand)
				continue
			}
			data, readErr := os.ReadFile(file)
			if readErr != nil {
				c.fail(RuleInventoryStale, src.Path, "el origen del inventario ya no se puede leer (%v); vuelve a generar el inventario con `%s`", readErr, c.scanCommand)
				continue
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != src.SHA256 {
				c.fail(RuleInventoryStale, src.Path, "el contenido del origen cambió desde que se generó el inventario (SHA-256 distinto); vuelve a generar el inventario con `%s`", c.scanCommand)
			}
		}
	}
}

func unitKey(rel, anchor string) string {
	if anchor == "" {
		return rel
	}
	return rel + "#" + anchor
}

func normalizeRel(p string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
}

// insideWorkspace resuelve una ruta relativa dentro de root y la acepta solo si existe,
// es un fichero y, con los enlaces simbólicos resueltos, sigue dentro de root.
func insideWorkspace(root, rel string) (string, bool) {
	abs := filepath.Join(root, rel)
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return "", false
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	if _, ok := adoption.ResolveInside(realRoot, abs); !ok {
		return "", false
	}
	return abs, true
}
