package livespec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ChangeReportSchema es el esquema del informe de `axiom sdd archive-integrate --json`.
const ChangeReportSchema = "axiom.sdd-archive-integrate/v1"

// traceFileName es la traza de identificadores de un cambio SDD (D12).
const traceFileName = "ids.yaml"

// ChangeOptions configura PlanChange. Supersede y Superseded son las opciones de superación
// de `archive-compose` aplicadas a todo el cambio (D4); Check solo marca el informe: el plan
// es el mismo y quien no llama a Apply no escribe nada.
type ChangeOptions struct {
	Supersede  bool
	Superseded []string
	Check      bool
}

// ChangeIntegration es el resultado, calculado en memoria, de integrar un cambio SDD entero:
// la spec viva de cada capacidad, las reescrituras de los artefactos del cambio y la traza.
// No se ha escrito nada hasta que se llama a Apply.
type ChangeIntegration struct {
	Change       string
	Check        bool
	Capabilities []CapabilityPlan
	// Requirements son los requisitos que las deltas del cambio tocan y crean.
	Requirements openspecdoc.Declaration
	Assignments  []PlannedEntry
	Duplicates   []Duplicate

	loc   workspace.Location
	files []fileWrite
}

// NothingToIntegrate dice si el paso no cambia ningún fichero.
func (c ChangeIntegration) NothingToIntegrate() bool { return len(c.files) == 0 }

// Apply escribe todos los ficheros del plan o ninguno (RF-RID-12) con writeSet. Devuelve las
// rutas absolutas escritas. Un plan sin cambios no toca el disco.
func (c ChangeIntegration) Apply() ([]string, error) {
	return writeSet(c.files, writeOptions{Retry: "axiom sdd archive-integrate --change " + c.Change})
}

// changeFileKind es la clase de un artefacto del cambio que el paso reescribe.
type changeFileKind int

const (
	kindTasks changeFileKind = iota
	kindVerify
	kindAcceptanceTests
)

// PlanChange calcula en memoria la integración de todas las capacidades de un cambio SDD
// (D3, D4): el plan único de asignación (PlanIDs) con la validación previa de todo el
// cambio, la composición de cada capacidad con la delta entera, la reanudación con solo lo
// pendiente o la copia de una capacidad nueva, los duplicados entre capacidades (RF-RID-13) y
// las reescrituras de las deltas, de las líneas `Cubre:` de las tareas, de la matriz de los
// informes de verify y de la traza `ids.yaml`, solo con el mapa no vacío. reg es el registro
// sin el propio cambio. El primer error rechaza todo el plan.
func PlanChange(loc workspace.Location, change string, opts ChangeOptions, reg *Registry) (ChangeIntegration, error) {
	pattern := reg.Pattern()
	changeRoot := filepath.Join(loc.PlanningHome, "changes", change)
	specsDir := filepath.Join(changeRoot, "specs")
	retry := "axiom sdd archive-integrate --change " + change
	plan := ChangeIntegration{Change: change, Check: opts.Check, loc: loc}
	relative := func(path string) string {
		if rel, err := filepath.Rel(loc.WorkspaceRoot, path); err == nil {
			return filepath.ToSlash(rel)
		}
		return filepath.ToSlash(path)
	}

	flat := ""
	flatData, err := readOptional(filepath.Join(specsDir, "spec.md"))
	if err != nil {
		return ChangeIntegration{}, err
	}
	if flatData != nil {
		flat = relative(filepath.Join(specsDir, "spec.md"))
	}
	names, err := subdirectories(specsDir)
	if err != nil {
		return ChangeIntegration{}, err
	}
	var sources []CapabilitySource
	deltaPaths := map[string]string{}
	for _, name := range names {
		deltaPath := filepath.Join(specsDir, name, "spec.md")
		data, err := readOptional(deltaPath)
		if err != nil {
			return ChangeIntegration{}, err
		}
		if data == nil {
			continue
		}
		text := string(data)
		delta, _ := openspecdoc.ParseChangeSpec(text)
		path, canonical, existed, err := readCanonical(loc, name, retry)
		if err != nil {
			return ChangeIntegration{}, err
		}
		sources = append(sources, CapabilitySource{
			CapabilityDelta: CapabilityDelta{Name: name, Delta: delta, Text: text},
			File:            relative(deltaPath), Path: path, Canonical: canonical, Existed: existed,
		})
		deltaPaths[name] = deltaPath
		declared := openspecdoc.Declared(name, delta, pattern)
		plan.Requirements.Modifies = append(plan.Requirements.Modifies, declared.Modifies...)
		plan.Requirements.Creates = append(plan.Requirements.Creates, declared.Creates...)
	}

	tracePath := filepath.Join(changeRoot, traceFileName)
	traceData, err := readOptional(tracePath)
	if err != nil {
		return ChangeIntegration{}, err
	}
	var trace Trace
	if traceData != nil {
		if trace, err = ParseTrace(traceData, relative(tracePath)); err != nil {
			return ChangeIntegration{}, err
		}
	}
	artifacts, citations, err := changeArtifacts(changeRoot, relative)
	if err != nil {
		return ChangeIntegration{}, err
	}

	idPlan, err := PlanIDs(IDPlanInput{
		Pattern: pattern, Capabilities: sources, Registry: reg, Trace: trace, Citations: citations,
		Superseded: opts.Superseded, FlatDelta: flat,
	})
	if err != nil {
		return ChangeIntegration{}, err
	}
	plan.Assignments = idPlan.Entries

	var specFiles, deltaFiles []fileWrite
	resulting := map[string]string{}
	for _, s := range sources {
		substituted := s.Text
		if idPlan.Map.Count() > 0 {
			substituted = idPlan.Map.Substitute(s.Text)
			if substituted != s.Text {
				deltaFiles = append(deltaFiles, fileWrite{Path: deltaPaths[s.Name], Content: []byte(substituted), Name: s.Name})
			}
		}
		step, err := composeChangeCapability(s, substituted, idPlan, opts, pattern)
		if err != nil {
			return ChangeIntegration{}, err
		}
		plan.Capabilities = append(plan.Capabilities, step)
		if step.Outcome != OutcomeAlready {
			resulting[step.Name] = step.Content
			specFiles = append(specFiles, fileWrite{Path: step.Path, Content: []byte(step.Content), Name: step.Name})
		}
	}
	if plan.Duplicates, err = checkDuplicates(reg, resulting, idPlan.Written); err != nil {
		return ChangeIntegration{}, err
	}

	plan.files = append(plan.files, specFiles...)
	plan.files = append(plan.files, deltaFiles...)
	if idPlan.Map.Count() > 0 {
		for _, a := range artifacts {
			rewritten := a.text
			switch a.kind {
			case kindTasks:
				rewritten = isoevidence.RewriteCubre(a.text, idPlan.Map)
			case kindAcceptanceTests:
				rewritten = acceptancetests.RewriteCubre(a.text, idPlan.Map)
			default:
				rewritten = isoevidence.RewriteMatrix(a.text, idPlan.Map)
			}
			if rewritten != a.text {
				plan.files = append(plan.files, fileWrite{Path: a.path, Content: []byte(rewritten), Name: filepath.Base(a.path)})
			}
		}
	}
	if rendered := idPlan.Trace(pattern).Render(); rendered != nil && !bytes.Equal(rendered, traceData) {
		plan.files = append(plan.files, fileWrite{Path: tracePath, Content: rendered, Name: traceFileName})
	}
	return plan, nil
}

// changeArtifact es un fichero de tareas o un informe de verify del cambio.
type changeArtifact struct {
	path string
	text string
	kind changeFileKind
}

// changeArtifacts lee los ficheros de tareas (`tasks.md`, `tasks.<rol>.md`) y los informes de
// verify (`verify-report.md`, `verify-report.<rol>.md`) del cambio, y devuelve las citas que
// el plan valida (D6): los provisionales de las líneas `Cubre:` de un fichero de tareas y
// los de la columna `Requisito` de la matriz de cada informe de verify, y los de la parte ID
// de cada entrada de la columna `Cubre` de `pruebas.md` (RF-KOF-23). El resto de esos
// ficheros no se valida (RF-RID-12): una tabla inválida de `pruebas.md` no produce citas, no
// se reescribe y no da error propio; un provisional citado sin correspondencia lo rechaza
// PlanIDs, igual que en tareas y verify.
func changeArtifacts(changeRoot string, relative func(string) string) ([]changeArtifact, []Citation, error) {
	entries, err := os.ReadDir(changeRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos de la carpeta", changeRoot, err)
	}
	var out []changeArtifact
	var citations []Citation
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		var kind changeFileKind
		switch {
		case name == "tasks.md" || strings.HasPrefix(name, "tasks."):
			kind = kindTasks
		case name == "verify-report.md" || strings.HasPrefix(name, "verify-report."):
			kind = kindVerify
		case name == acceptancetests.FileName:
			kind = kindAcceptanceTests
		default:
			continue
		}
		path := filepath.Join(changeRoot, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero", path, err)
		}
		text := string(data)
		out = append(out, changeArtifact{path: path, text: text, kind: kind})
		file := relative(path)
		if kind == kindTasks {
			citations = append(citations, cubreCitations(file, text)...)
			continue
		}
		if kind == kindAcceptanceTests {
			for _, c := range acceptancetests.Parse(text).Cases {
				for _, entry := range c.Cubre {
					for _, span := range reqid.ScanProvisional(entry.ID) {
						citations = append(citations, Citation{File: file, Line: c.Line, Token: span.Token})
					}
				}
			}
			continue
		}
		for _, row := range isoevidence.ParseMatrix(text).Rows {
			for _, span := range reqid.ScanProvisional(row.RequirementID) {
				citations = append(citations, Citation{File: file, Line: row.Line, Token: span.Token})
			}
		}
	}
	return out, citations, nil
}

// composeChangeCapability calcula la spec viva de una capacidad del cambio (D4): copia para una
// capacidad nueva, composición con la delta entera y, si la composición entera se rechaza,
// reanudación con solo lo pendiente.
func composeChangeCapability(s CapabilitySource, substituted string, idPlan IDPlan, opts ChangeOptions, pattern reqid.Pattern) (CapabilityPlan, error) {
	step := CapabilityPlan{Name: s.Name, Path: s.Path, Existed: s.Existed}
	if !s.Existed {
		step.Outcome, step.Content = OutcomeCreated, substituted
		return step, nil
	}

	canonical := s.Canonical
	edited, wasEdited := idPlan.Canonical[s.Name]
	if wasEdited {
		canonical = edited
	}
	cd := s.CapabilityDelta
	cd.Text = substituted
	cd.Delta, _ = openspecdoc.ParseChangeSpec(substituted)
	item, err := classifyIn(cd, s.Path, true, canonical, pattern)
	if err != nil {
		return CapabilityPlan{}, err
	}
	dropTracedAdded(&item, idPlan.Entries, pattern)

	pending := item.pending
	if len(pending.Added)+len(pending.Modified)+len(pending.Removed)+len(pending.Renamed) == 0 {
		if wasEdited {
			// Solo cambió el número de un bloque propio de la spec viva (reasignación).
			step.Outcome, step.Content = OutcomeComposed, edited
			return step, nil
		}
		step.Outcome = OutcomeAlready
		return step, nil
	}

	composeOptions := openspecdoc.ComposeOptions{SupersedeRemoved: opts.Supersede, ExplicitSuperseded: idPlan.Superseded[s.Name], IDPattern: pattern}
	composed, err := openspecdoc.Compose(canonical, substituted, composeOptions)
	if err == nil {
		step.Outcome, step.Content = OutcomeComposed, composed
		return step, nil
	}
	resumed, resumeErr := openspecdoc.Compose(canonical, reducedDelta(pending), composeOptions)
	if resumeErr != nil {
		return CapabilityPlan{}, &CapabilityError{Capability: s.Name, Err: resumeErr}
	}
	step.Outcome, step.Content, step.Skipped = OutcomeResumed, resumed, skippedOf(cd.Delta, pending)
	return step, nil
}

// skippedOf son los requisitos de la delta que no están pendientes.
func skippedOf(all, pending openspecdoc.Delta) []SkippedRequirement {
	var out []SkippedRequirement
	for i, section := range deltaSections(all) {
		left := map[string]bool{}
		for _, r := range deltaSections(pending)[i].reqs {
			left[r.Name] = true
		}
		for _, r := range section.reqs {
			if !left[r.Name] {
				out = append(out, SkippedRequirement{Section: section.name, Requirement: r.Name})
			}
		}
	}
	return out
}

// CheckArchivedTrace comprueba, sobre un cambio ya archivado, que cada definitivo de su traza
// aparece una sola vez en la spec viva y en su capacidad (D4, RF-RID-13). El cambio archivado
// es inmutable (RF-ARC-03): el duplicado se resuelve rehaciendo el archivado.
func CheckArchivedTrace(reg *Registry, trace Trace) error {
	if err := trace.validate(reg.Pattern(), traceFileName); err != nil {
		return err
	}
	for _, e := range trace.Assignments {
		var where []string
		for _, ref := range reg.LookupIdentity(e.Definitive) {
			if ref.Kind == RefSpec {
				where = append(where, ref.Capability)
			}
		}
		if len(where) == 1 && where[0] == e.Capability {
			continue
		}
		sort.Strings(where)
		reason := fmt.Sprintf("el identificador %s asignado a la capacidad %s no aparece una sola vez en ella en la spec viva: aparece en %s", e.Definitive, e.Capability, strings.Join(where, ", "))
		if len(where) == 0 {
			reason = fmt.Sprintf("el identificador %s asignado a la capacidad %s no aparece en la spec viva", e.Definitive, e.Capability)
		}
		return &IDPlanError{Code: CodeDuplicateAssignedID, Capability: e.Capability, Token: e.Definitive, Reason: reason,
			Continuation: "el cambio archivado es inmutable: descarta la rama de archivo (`axiom branch done --force`) y rehaz el archivado desde la rama de integración, donde la carpeta del cambio conserva los provisionales y la reejecución asigna el siguiente libre"}
	}
	return nil
}

// ChangeReport es el informe `axiom.sdd-archive-integrate/v1` (D4).
type ChangeReport struct {
	Schema             string                   `json:"schema"`
	Change             string                   `json:"change"`
	Check              bool                     `json:"check"`
	Capabilities       []ChangeCapabilityReport `json:"capabilities"`
	Requirements       openspecdoc.Declaration  `json:"requirements"`
	Assignments        []AssignmentReport       `json:"assignments"`
	Duplicates         []Duplicate              `json:"duplicates"`
	Written            []string                 `json:"written"`
	NothingToIntegrate bool                     `json:"nothingToIntegrate"`
	Error              *ReportError             `json:"error,omitempty"`
}

// ChangeCapabilityReport es el resultado de una capacidad en el informe del cambio.
type ChangeCapabilityReport struct {
	Name    string               `json:"name"`
	Path    string               `json:"path"`
	Outcome Outcome              `json:"outcome"`
	Skipped []SkippedRequirement `json:"skipped,omitempty"`
}

// ReportError es el rechazo tipado dentro del informe JSON.
type ReportError struct {
	Code         string `json:"code"`
	Capability   string `json:"capability,omitempty"`
	Section      string `json:"section,omitempty"`
	Requirement  string `json:"requirement,omitempty"`
	Token        string `json:"token,omitempty"`
	File         string `json:"file,omitempty"`
	Line         int    `json:"line,omitempty"`
	Reason       string `json:"reason"`
	Continuation string `json:"continuation"`
}

func emptyChangeReport(change string, check bool) ChangeReport {
	return ChangeReport{
		Schema: ChangeReportSchema, Change: change, Check: check,
		Capabilities: []ChangeCapabilityReport{},
		Requirements: openspecdoc.Declaration{Modifies: []openspecdoc.DeclaredRequirement{}, Creates: []openspecdoc.DeclaredRequirement{}},
		Assignments:  []AssignmentReport{}, Duplicates: []Duplicate{}, Written: []string{},
	}
}

// Report construye el informe del plan; written son las rutas absolutas que Apply devolvió
// (nil si no se aplicó nada, como con `--check`).
func (c ChangeIntegration) Report(written []string) ChangeReport {
	report := emptyChangeReport(c.Change, c.Check)
	for _, p := range c.Capabilities {
		report.Capabilities = append(report.Capabilities, ChangeCapabilityReport{Name: p.Name, Path: c.relative(p.Path), Outcome: p.Outcome, Skipped: p.Skipped})
	}
	if c.Requirements.Modifies != nil {
		report.Requirements.Modifies = c.Requirements.Modifies
	}
	if c.Requirements.Creates != nil {
		report.Requirements.Creates = c.Requirements.Creates
	}
	if assignments := AssignmentReports(c.Assignments); assignments != nil {
		report.Assignments = assignments
	}
	if c.Duplicates != nil {
		report.Duplicates = c.Duplicates
	}
	for _, path := range written {
		report.Written = append(report.Written, c.relative(path))
	}
	report.NothingToIntegrate = c.NothingToIntegrate()
	return report
}

func (c ChangeIntegration) relative(path string) string {
	if rel, err := filepath.Rel(c.loc.WorkspaceRoot, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

// ErrorReport es el informe de un rechazo: el esquema con `error` tipado.
func ErrorReport(change string, check bool, err error) ChangeReport {
	report := emptyChangeReport(change, check)
	report.Error = projectError(err)
	return report
}

// ArchivedReport es el informe de la comprobación de solo lectura de un cambio ya archivado:
// el mapa guardado con todas sus entradas en estado `kept`.
func ArchivedReport(change string, trace Trace) ChangeReport {
	report := emptyChangeReport(change, true)
	report.NothingToIntegrate = true
	for _, e := range trace.Assignments {
		report.Assignments = append(report.Assignments, AssignmentReport{
			Provisional: e.Provisional, Definitive: e.Definitive, Capability: e.Capability,
			Requirement: e.Requirement, State: StateKept, Previous: e.Previous,
		})
	}
	return report
}

// projectError proyecta un rechazo del plan a su código estable (D7).
func projectError(err error) *ReportError {
	var planErr *IDPlanError
	var missing *MissingCapabilityError
	var diverged *DivergentRequirementError
	var collision *CollisionError
	var capErr *CapabilityError
	switch {
	case errors.As(err, &planErr):
		return &ReportError{Code: planErr.Code, Capability: planErr.Capability, Section: planErr.Section, Requirement: planErr.Requirement,
			Token: planErr.Token, File: planErr.File, Line: planErr.Line, Reason: planErr.Reason, Continuation: planErr.Continuation}
	case errors.As(err, &missing):
		return &ReportError{Code: "capability_missing", Capability: missing.Capability, Section: missing.Section, Requirement: missing.Requirement,
			Reason:       err.Error(),
			Continuation: "cambia la operación (solo ADDED crea una capacidad) o corrige el nombre de la capacidad"}
	case errors.As(err, &diverged):
		return &ReportError{Code: "requirement_diverged", Capability: diverged.Capability, Requirement: diverged.Requirement, Token: diverged.ID,
			Reason:       err.Error(),
			Continuation: "mueve el requisito a MODIFIED o alinea la delta con la spec viva; si editaste lo ya integrado, restaura `openspec/specs/<capacidad>/spec.md` desde la base y reintegra"}
	case errors.As(err, &collision):
		return &ReportError{Code: "id_collision", Token: collision.ID, Reason: err.Error(),
			Continuation: "usa el siguiente número libre de la serie: " + collision.NextFree}
	case errors.As(err, &capErr):
		return &ReportError{Code: "composition_failed", Capability: capErr.Capability, Reason: err.Error(),
			Continuation: "corrige la delta de esa capacidad y vuelve a ejecutar la integración"}
	default:
		return &ReportError{Code: "integration_failed", Reason: err.Error(), Continuation: "corrige la causa y vuelve a ejecutar la integración"}
	}
}
