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
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Reglas de Check. Todas se acumulan: el informe lista cada fallo, no solo el primero.
const (
	RuleFormat           = "format"
	RuleUniqueness       = "uniqueness"
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
	Change string `json:"change"`
	// IDPattern es la plantilla del patrón efectivo con el que se mide el cambio e
	// IDPatternDeclared dice si el proyecto la declara en axiom.yaml (`sdd.id_pattern`) o es
	// el patrón por defecto.
	IDPattern         string    `json:"id_pattern"`
	IDPatternDeclared bool      `json:"id_pattern_declared"`
	Passed            bool      `json:"passed"`
	Failures          []Failure `json:"failures"`
}

var scenarioHeading = regexp.MustCompile(`^####[ \t]+Scenario:`)

var fenceLine = regexp.MustCompile("^ {0,3}(```+|~~~+)")

// addedRequirement es un requisito ADDED con la delta de la que sale.
type addedRequirement struct {
	req  openspecdoc.Requirement
	file string // specs/<capacidad>/spec.md, relativo a la carpeta del cambio
	id   string // identificador definitivo del título, "" si no lleva uno válido
	prov string // token provisional compatible del título, "" si no lleva uno
}

// Check verifica de forma determinista, sin modelo y sin escribir, el cambio de adopción
// de loc con el patrón de identificadores efectivo del proyecto (el declarado en
// `sdd.id_pattern` o el defecto): formato, unicidad por serie y número, escenarios,
// procedencia, cobertura del inventario y motivos de los descartes (REQ-30.11 y
// REQ-30.12). No decide nada de la compuerta `spec` ni juzga la fidelidad semántica de la
// migración (REQ-30.13).
func Check(loc workspace.Location, change string, pattern reqid.Pattern) (CheckReport, error) {
	changeRoot := filepath.Join(loc.PlanningHome, "changes", change)
	if info, err := os.Stat(changeRoot); err != nil || !info.IsDir() {
		return CheckReport{}, fmt.Errorf("no existe el cambio activo %q en %s; comprueba el nombre con `axiom sdd status --cwd %s`",
			change, pathquote.Quote(filepath.Join(loc.PlanningHome, "changes")), pathquote.Quote(loc.WorkspaceRoot))
	}
	scanCommand := fmt.Sprintf("axiom archive adopt scan --change %s", pathquote.ShellWord(change))
	c := &checker{loc: loc, change: change, changeRoot: changeRoot, scanCommand: scanCommand, pattern: pattern}

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

	report := CheckReport{
		Change:            change,
		IDPattern:         pattern.Template(),
		IDPatternDeclared: loc.Declarations.SDD != nil && loc.Declarations.SDD.IDPattern != nil,
		Failures:          c.failures,
	}
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
	pattern     reqid.Pattern
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
			a := addedRequirement{req: r, file: rel}
			a.id, a.prov = c.classifyTitle(a)
			c.added = append(c.added, a)
		}
	}
	if !found {
		c.fail(RuleFormat, "specs/<capacidad>/spec.md", "el cambio no tiene ninguna delta en specs/<capacidad>/spec.md; redacta la delta de adopción con `## ADDED Requirements`")
	}
	return nil
}

// lastParen extrae el último "(…)" de un título, sin los paréntesis.
var lastParen = regexp.MustCompile(`\(([^()]*)\)`)

// classifyTitle aplica la regla de formato (REQ-30.10 y REQ-30.11 a) al título de un
// requisito ADDED y devuelve su identificador definitivo o su provisional compatible con
// el patrón efectivo; ambos vacíos si el título incumple (el fallo ya consta).
func (c *checker) classifyTitle(a addedRequirement) (id, prov string) {
	name := a.req.Name
	id = c.pattern.InTitle(name)
	token, hasProv := reqid.ProvisionalInTitle(name)
	switch {
	case id != "" && hasProv:
		c.fail(RuleFormat, name, "el título del requisito (%s, línea %d) lleva un identificador `(%s)` y un provisional `(%s)` a la vez, lo que es ambiguo; deja solo uno", a.file, a.req.Line, id, token)
		return "", ""
	case id != "":
		return id, ""
	case hasProv:
		if _, err := c.pattern.ParseProvisional(token); err != nil {
			code, reason := "provisional_invalid", err.Error()
			var perr *reqid.ProvisionalError
			if errors.As(err, &perr) {
				code, reason = perr.Code, perr.Reason
			}
			c.fail(RuleFormat, name, "el provisional `(%s)` del título (%s, línea %d) no es compatible con el patrón `%s`: %s (código %s); escríbelo como %s",
				token, a.file, a.req.Line, c.pattern.Template(), reason, code, reqid.ProvisionalText("<clave>", "<ordinal>"))
			return "", ""
		}
		return "", token
	}
	found := ""
	if ms := lastParen.FindAllStringSubmatch(name, -1); len(ms) > 0 {
		found = fmt.Sprintf(" (el último paréntesis, `(%s)`, no es un identificador del patrón)", ms[len(ms)-1][1])
	}
	c.fail(RuleFormat, name, "el título del requisito (%s, línea %d) no lleva un identificador `(%s)` del patrón efectivo ni un provisional `(%s)`%s; añade uno al título",
		a.file, a.req.Line, c.pattern.Compact(), reqid.ProvisionalText("<clave>", "<ordinal>"), found)
	return "", ""
}

// identityKey es la identidad de un identificador definitivo: serie y número del patrón.
type identityKey struct {
	series reqid.Series
	number int
}

// checkIdentifiers aplica la unicidad (REQ-30.11 b): dentro del cambio por serie y número,
// los provisionales por token y contra el registro de specs y cambios activos con
// LookupIdentity.
func (c *checker) checkIdentifiers() error {
	byKey := map[identityKey][]addedRequirement{}
	var keys []identityKey
	byToken := map[string][]addedRequirement{}
	var tokens []string
	for _, a := range c.added {
		switch {
		case a.id != "":
			series, ok := c.pattern.Series(a.id)
			number, okN := c.pattern.Number(a.id)
			if !ok || !okN {
				continue
			}
			key := identityKey{series: series, number: number}
			if len(byKey[key]) == 0 {
				keys = append(keys, key)
			}
			byKey[key] = append(byKey[key], a)
		case a.prov != "":
			if len(byToken[a.prov]) == 0 {
				tokens = append(tokens, a.prov)
			}
			byToken[a.prov] = append(byToken[a.prov], a)
		}
	}

	all, err := livespec.LoadRegistry(c.loc.PlanningHome, livespec.RegistryOptions{ExcludeChange: c.change, Pattern: c.pattern})
	if err != nil {
		return err
	}
	where := func(list []addedRequirement) string {
		parts := make([]string, len(list))
		for i, a := range list {
			parts[i] = fmt.Sprintf("«%s» (%s, línea %d)", a.req.Name, a.file, a.req.Line)
		}
		return strings.Join(parts, " y ")
	}
	for _, key := range keys {
		list := byKey[key]
		first := list[0].id
		if len(list) > 1 {
			same := true
			for _, a := range list {
				same = same && a.id == first
			}
			if same {
				c.fail(RuleUniqueness, first, "el identificador se repite dentro del cambio: %s; asigna un identificador distinto a cada requisito o, si fusiona unidades del cliente, usa un provisional para uno de ellos", where(list))
			} else {
				c.fail(RuleUniqueness, first, "dos requisitos comparten serie y número (serie `%s`, número %d): %s; asigna un número distinto a cada requisito", key.series.String(), key.number, where(list))
			}
		}
		for _, ref := range all.LookupIdentity(first) {
			c.fail(RuleUniqueness, first, "el identificador ya existe en %s (%s); elige un número libre de la serie, por ejemplo con el siguiente de `openspec/specs/`", ref.Where(), filepath.ToSlash(ref.Path))
		}
	}
	for _, token := range tokens {
		if list := byToken[token]; len(list) > 1 {
			c.fail(RuleUniqueness, token, "el provisional se repite dentro del cambio: %s; usa un ordinal distinto en cada requisito", where(list))
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
