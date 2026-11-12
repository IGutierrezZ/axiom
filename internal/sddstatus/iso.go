package sddstatus

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// bugProfile es el resultado de leer el perfil de bug sin delta de proposal.md
// (REQ-29.15, design D9). Es el adaptador entre el status y isoevidence: este
// paquete lee el fichero y isoevidence solo analiza texto.
type bugProfile struct {
	// declaration es lo que declara proposal.md. Con Kind "none" el estado es
	// idéntico al anterior a este cambio.
	declaration isoevidence.BugDeclaration
	// findings son los motivos de una declaración incompleta, uno por cada cosa
	// que falta.
	findings []isoevidence.Finding
}

// declared dice si proposal.md declara alguna de las dos secciones del perfil.
func (b bugProfile) declared() bool {
	return b.declaration.Kind != isoevidence.BugKindNone
}

// satisfiesSpecs dice si la declaración, completa, cumple el artefacto de specs
// sin delta.
func (b bugProfile) satisfiesSpecs() bool {
	return b.declared() && b.declaration.Valid
}

// evaluateBugProfile aplica D9: solo cuando el artefacto de specs está `missing`
// (sin delta ni ficheros en specs/) se analiza proposal.md. Con delta (`done` o
// `partial`) las secciones se ignoran sin aviso (regla 1 de REQ-29.15). Vale en
// todo proyecto, sea o no ISO.
func evaluateBugProfile(specs ArtifactState, proposalPaths []string) bugProfile {
	if specs != ArtifactMissing {
		return bugProfile{declaration: isoevidence.BugDeclaration{Kind: isoevidence.BugKindNone}}
	}
	declaration := isoevidence.ParseBugDeclaration(readText(firstPath(proposalPaths)))
	return bugProfile{declaration: declaration, findings: declaration.Findings()}
}

// bugDeclaredWithoutDelta dice si un cambio sin ningún fichero de specs declara
// un bug completo en proposal.md: el criterio con el que la compuerta `spec`
// juzga proposal.md (D20) y con el que lo mismo calcula `axiom sdd gate record`.
func bugDeclaredWithoutDelta(specPaths, proposalPaths []string) bool {
	if len(specPaths) != 0 {
		return false
	}
	return evaluateBugProfile(ArtifactMissing, proposalPaths).satisfiesSpecs()
}

// findingReasons compone una línea de blockedReasons por hallazgo (REQ-29.16).
func findingReasons(findings []isoevidence.Finding) []string {
	reasons := make([]string, 0, len(findings))
	for _, finding := range findings {
		reasons = append(reasons, finding.Reason())
	}
	return reasons
}

// isoInputs es lo que las reglas ISO leen de un cambio: la delta y la baseline
// con el lector nativo de OpenSpec (D2), la declaración de bug de proposal.md y
// las líneas `Cubre:` de todos los ficheros de tareas. El paquete isoevidence
// no lee ficheros: este adaptador los lee y le entrega el modelo ya construido.
type isoInputs struct {
	// hasDelta dice si el cambio lleva al menos un fichero de specs.
	hasDelta bool
	delta    []isoevidence.Requirement
	baseline []isoevidence.Requirement
	index    isoevidence.BaselineIndex
	bug      isoevidence.BugDeclaration
	// tasksComplete dice si existen todos los ficheros de tareas del cambio
	// (tasks.md o los de cada rol): sin ellos no se evalúa la cobertura.
	tasksComplete bool
	cubre         []isoevidence.CubreLine
}

// loadISOInputs lee los artefactos del cambio. Un fichero ilegible cuenta como
// vacío: la ausencia de un artefacto la proyectan ya los estados ordinarios.
func loadISOInputs(planningHome, changeRoot string, paths ArtifactPaths, artifacts map[string]ArtifactState) isoInputs {
	in := isoInputs{hasDelta: len(paths.Specs) > 0}
	in.delta = readDeltaRequirements(changeRoot, paths.Specs)
	in.baseline = readBaselineRequirements(planningHome)
	in.index = isoevidence.NewBaselineIndex(in.baseline)
	if !in.hasDelta {
		in.bug = isoevidence.ParseBugDeclaration(readText(firstPath(paths.Proposal)))
	}
	in.tasksComplete = artifacts["tasks"] == ArtifactDone && len(paths.Tasks) > 0
	// Las líneas `Cubre:` se leen de todos los ficheros de tareas que existen: la
	// etapa `tasks` solo se evalúa con todos presentes (tasksComplete), pero el
	// archivado necesita los requisitos citados por un bug con requisito existente.
	for _, path := range paths.Tasks {
		in.cubre = append(in.cubre, isoevidence.ParseCubre(filepath.Base(path), readText(path))...)
	}
	return in
}

// readDeltaRequirements lee los requisitos de cada delta con openspecdoc.ParseDelta,
// la misma función que usa el compositor de archivado. El dominio es el de la
// carpeta `specs/<dominio>/`; una delta plana no tiene dominio.
func readDeltaRequirements(changeRoot string, specFiles []string) []isoevidence.Requirement {
	var delta []isoevidence.Requirement
	for _, path := range specFiles {
		domain := ""
		if dir := filepath.Dir(path); filepath.Clean(dir) != filepath.Clean(changeRoot) {
			domain = filepath.Base(dir)
		}
		parsed := openspecdoc.ParseDelta(readText(path))
		add := func(section isoevidence.Section, blocks []openspecdoc.Requirement) {
			for _, block := range blocks {
				requirement := isoevidence.Requirement{Domain: domain, Section: section, Name: block.Name, Text: block.Body}
				if section == isoevidence.SectionRenamed {
					if oldName, newName, ok := openspecdoc.SplitRename(block.Name); ok {
						requirement.OldName, requirement.Name = oldName, newName
					}
				}
				delta = append(delta, requirement)
			}
		}
		add(isoevidence.SectionAdded, parsed.Added)
		add(isoevidence.SectionModified, parsed.Modified)
		add(isoevidence.SectionRemoved, parsed.Removed)
		add(isoevidence.SectionRenamed, parsed.Renamed)
	}
	return delta
}

// readBaselineRequirements lee los requisitos de `<planning home>/specs/*/spec.md`
// con openspecdoc.ParseCanonical, el lector nativo (D2). El texto de cada requisito es el
// del segmento, que es lo que compara la tolerancia del archivado interrumpido.
func readBaselineRequirements(planningHome string) []isoevidence.Requirement {
	files, err := filepath.Glob(filepath.Join(planningHome, "specs", "*", "spec.md"))
	if err != nil {
		return nil
	}
	sort.Strings(files)
	var baseline []isoevidence.Requirement
	for _, path := range files {
		domain := filepath.Base(filepath.Dir(path))
		for _, req := range openspecdoc.ParseCanonical(readText(path)) {
			baseline = append(baseline, isoevidence.Requirement{Domain: domain, Name: req.Name, Text: req.Body})
		}
	}
	return baseline
}

// specsFindings es la etapa `specs` de un cambio con delta: identificadores
// (REQ-29.3) y escenarios (REQ-29.10 y nombres repetidos de REQ-29.6). Un bug
// sin delta no tiene requisitos propios que comprobar.
func (in isoInputs) specsFindings() []isoevidence.Finding {
	if !in.hasDelta {
		return nil
	}
	findings := isoevidence.CheckRequirementIDs(in.delta, in.baseline)
	return append(findings, isoevidence.CheckScenarios(in.delta)...)
}

// tasksFindings es la etapa `tasks`: cobertura (REQ-29.4) con delta, y las
// comprobaciones del bug sin delta (REQ-29.5 y REQ-29.19) cuando su declaración
// es completa. Solo se evalúa cuando existen todos los ficheros de tareas.
func (in isoInputs) tasksFindings() []isoevidence.Finding {
	if !in.tasksComplete {
		return nil
	}
	switch {
	case in.hasDelta:
		return isoevidence.CheckCoverage(in.delta, in.index, in.cubre)
	case in.bug.Kind == isoevidence.BugKindExistingRequirement && in.bug.Valid:
		findings := isoevidence.CheckCoverage(nil, in.index, in.cubre)
		return append(findings, isoevidence.CheckExistingRequirementBug(in.index, in.cubre)...)
	case in.bug.Kind == isoevidence.BugKindNoImpact && in.bug.Valid:
		return isoevidence.CheckNoImpactBug(in.cubre)
	}
	return nil
}

// isoEvaluation es el resultado de las reglas ISO de un cambio ISO, por etapa
// (design D6).
type isoEvaluation struct {
	inputs isoInputs
	specs  []isoevidence.Finding
	tasks  []isoevidence.Finding
	// archive son los hallazgos de la etapa `archive` (escenarios y matriz), y
	// matrix el resultado de validar la matriz. Los rellena evaluateArchive solo
	// cuando el archivado estaría disponible (D12).
	archive []isoevidence.Finding
	matrix  *isoevidence.MatrixResult
}

// evaluateISOArchive aplica la regla del archivado ISO sobre los artefactos del
// cambio: los escenarios de todas las deltas (REQ-29.10) y la matriz de verify,
// válida y de éxito (REQ-29.11, REQ-29.6 a REQ-29.8 y REQ-29.19). No evalúa
// REQ-29.3: la unicidad de un ADDED es de la etapa `specs`, y tras componer un
// dominio dejaría de cumplirse (D17). master es la raíz del árbol contra la que
// resuelven las referencias de prueba (D11); con una raíz que no se puede abrir
// no hay árbol y las filas automáticas quedan inválidas, nunca aprobadas.
func evaluateISOArchive(in isoInputs, verifyReport, master string) ([]isoevidence.Finding, isoevidence.MatrixResult) {
	var tree isoevidence.TreeResolver
	if resolver, err := isoevidence.NewTreeResolver(master); err == nil {
		defer resolver.Close()
		tree = resolver
	}

	noImpact := !in.hasDelta && in.bug.Kind == isoevidence.BugKindNoImpact && in.bug.Valid
	var affected []isoevidence.ScenarioKey
	switch {
	case in.hasDelta:
		affected = isoevidence.AffectedFromDelta(in.delta)
	case in.bug.Kind == isoevidence.BugKindExistingRequirement && in.bug.Valid:
		var cited []string
		for _, line := range in.cubre {
			cited = append(cited, line.IDs...)
		}
		affected = isoevidence.AffectedFromBaseline(in.index, cited)
	}
	result := isoevidence.ValidateMatrix(isoevidence.ParseMatrix(verifyReport), affected, noImpact, tree)

	var findings []isoevidence.Finding
	if in.hasDelta {
		findings = append(findings, isoevidence.CheckScenarios(in.delta)...)
	}
	return append(findings, result.Findings...), result
}

// ISOArchiveRefusal es la regla del archivado de un cambio ISO (design D13): la
// declaración `sdd:` válida (D8), los escenarios de todas las deltas y la matriz
// de verify válida y de éxito, con independencia de cuántas tareas queden
// pendientes (D12). La comparten `axiom sdd status`, `axiom sdd archive-check` y
// `axiom sdd archive-compose`, de modo que ninguno discrepa. Un cambio sin
// instantánea ISO, sin kickoff o con un kickoff ilegible no tiene hallazgos: el
// kickoff ilegible ya lo reporta el status con su error nombrado.
func ISOArchiveRefusal(workspaceRoot, changeRoot string) []isoevidence.Finding {
	sealed, err := kickoff.Load(changeRoot)
	if err != nil || sealed == nil || !sealed.IsISO() {
		return nil
	}
	location, _ := workspace.LocateAt(workspaceRoot)
	master := location.Master
	if master == "" {
		master = workspaceRoot
	}

	var findings []isoevidence.Finding
	if declaration := readISODeclaration(location.Declarations); declaration.invalidText != "" {
		findings = append(findings, declaration.finding(master))
	}
	paths, err := resolveArtifactPaths(changeRoot)
	if err != nil {
		return findings
	}
	inputs := loadISOInputs(filepath.Join(workspaceRoot, "openspec"), changeRoot, paths, nil)
	archive, _ := evaluateISOArchive(inputs, readText(firstPath(paths.VerifyReport)), master)
	return append(findings, archive...)
}

// ISOChangeRequirements devuelve los requisitos de las deltas del cambio y los de
// la baseline de planningHome, leídos con el mismo lector nativo que usa el
// compositor (D2). Los usa `axiom sdd archive-check` para decidir, por dominio,
// si la composición ya está hecha (D17). No evalúa ninguna regla.
func ISOChangeRequirements(planningHome, changeRoot string) (delta, baseline []isoevidence.Requirement, err error) {
	paths, err := resolveArtifactPaths(changeRoot)
	if err != nil {
		return nil, nil, err
	}
	return readDeltaRequirements(changeRoot, paths.Specs), readBaselineRequirements(planningHome), nil
}

// evaluateArchive rellena la etapa `archive` y la proyección de la matriz.
func (e *isoEvaluation) evaluateArchive(verifyPaths []string, master string) {
	findings, result := evaluateISOArchive(e.inputs, readText(firstPath(verifyPaths)), master)
	e.archive = findings
	e.matrix = &result
}

// matrixProjection es `iso.matrix`, que usa el orquestador para decidir si el
// verify es limpio sin leer el informe (REQ-29.16).
func (e *isoEvaluation) matrixProjection() *ISOMatrixProjection {
	if e == nil || e.matrix == nil {
		return nil
	}
	return &ISOMatrixProjection{State: string(e.matrix.State), Successful: e.matrix.Successful, Findings: len(e.matrix.Findings)}
}

// archiveDependencyFromISO solo estrecha `archive` de `ready` a `blocked`: nunca
// lo mejora, y no añade ningún motivo propio (van a notes y blockedReasons).
func archiveDependencyFromISO(archive DependencyState, e *isoEvaluation) DependencyState {
	if e == nil || archive != DependencyReady || len(e.archive) == 0 {
		return archive
	}
	return DependencyBlocked
}

// verifyRoute aplica D19: con apply terminado y la matriz fallando, la ruta
// ordinaria (resolve-blockers, o await-gate de la compuerta `integration`, porque
// la matriz forma parte de la PR que esa compuerta fusiona) se sustituye por
// `verify` si la matriz falta o es inválida, y por resolve-blockers si tiene
// filas `fail`. Las compuertas spec, design, tasks y role-apply conservan su
// precedencia.
func (e *isoEvaluation) verifyRoute(next string, applyState ApplyState, governance *Governance) string {
	if e == nil || e.matrix == nil || e.matrix.Successful || len(e.archive) == 0 || applyState != ApplyAllDone {
		return next
	}
	switch next {
	case "resolve-blockers":
	case "await-gate":
		if gate, ok := firstOpenGate(governance); !ok || gate.Key != kickoff.GateIntegration {
			return next
		}
	default:
		return next
	}
	if e.matrix.State == isoevidence.MatrixMissing || e.matrix.State == isoevidence.MatrixInvalid {
		return string(PhaseVerify)
	}
	return "resolve-blockers"
}

// evaluateISOChange evalúa las etapas `specs` y `tasks` de un cambio ISO, leído
// de su instantánea. Con otro cambio no se llama: fuera de ISO ninguna de estas
// reglas proyecta ni bloqueo ni aviso (REQ-29.16).
func evaluateISOChange(planningHome, changeRoot string, paths ArtifactPaths, artifacts map[string]ArtifactState) *isoEvaluation {
	inputs := loadISOInputs(planningHome, changeRoot, paths, artifacts)
	return &isoEvaluation{inputs: inputs, specs: inputs.specsFindings(), tasks: inputs.tasksFindings()}
}

// blocksCore dice si alguna etapa de planificación tiene hallazgos: `coreReady`
// pasa a falso y la ruta vuelve a la fase que los corrige. Es seguro con nil.
func (e *isoEvaluation) blocksCore() bool {
	return e != nil && len(e.specs)+len(e.tasks) > 0
}

// narrow estrecha las dependencias `specs` y `tasks` de las etapas con hallazgos.
func (e *isoEvaluation) narrow(dependencies *Dependencies) {
	if e == nil {
		return
	}
	if len(e.specs) > 0 {
		dependencies.Specs = DependencyBlocked
	}
	if len(e.tasks) > 0 {
		dependencies.Tasks = DependencyBlocked
	}
}

// planningReasons son los motivos de las etapas `specs` y `tasks`, un motivo por
// hallazgo, para el canal `genuine` que la ruta de planificación conserva.
func (e *isoEvaluation) planningReasons() []string {
	if e == nil {
		return nil
	}
	return append(findingReasons(e.specs), findingReasons(e.tasks)...)
}

// Valores de ISOProjection.Project y ISOProjection.Change (design §5.2).
const (
	isoValueISO     = "iso"
	isoValueNotISO  = "not_iso"
	isoValueInvalid = "invalid"
)

// ISOProjection es la proyección aditiva de REQ-29.2. Se omite cuando axiom.yaml
// no tiene bloque `sdd:` y el cambio seleccionado no es ISO, de modo que un
// consumidor anterior no observa ninguna diferencia.
type ISOProjection struct {
	// Project: "iso" (sdd.iso: true), "not_iso" (false o bloque sin la clave) o
	// "invalid" (bloque inválido, contradictorio con workflow.lane, o axiom.yaml
	// ilegible).
	Project string `json:"project"`
	// Reason: solo con Project "invalid"; el error y su continuación. Está
	// presente con o sin cambio seleccionado.
	Reason string `json:"reason,omitempty"`
	// Change: "iso" o "not_iso", leído de la instantánea de kickoff.yaml. Ausente
	// sin cambio seleccionado.
	Change string `json:"change,omitempty"`
	// Matrix: solo en un cambio ISO con apply terminado.
	Matrix *ISOMatrixProjection `json:"matrix,omitempty"`
}

// ISOMatrixProjection es el resultado de validar la matriz de verify de un cambio
// ISO, para que el orquestador decida si el verify es limpio sin leer el informe.
type ISOMatrixProjection struct {
	// State: "valid", "invalid", "missing" o "not_required" (bug sin impacto sin
	// matriz).
	State string `json:"state"`
	// Successful: la matriz es válida (o no requerida) y ninguna fila falla.
	Successful bool `json:"successful"`
	// Findings es el número de hallazgos; el detalle va en blockedReasons.
	Findings int `json:"findings"`
}

// isoDeclaration es lo que decorateISO sabe de la declaración `sdd:` del proyecto.
type isoDeclaration struct {
	// invalidText es el error de la declaración sin su continuación; vacío si es
	// válida. Una declaración inválida falla cerrada (D8): nunca se asume no ISO.
	invalidText string
	// contradiction marca la contradicción con workflow.lane (D18), cuyo error ya
	// lleva su corrección.
	contradiction bool
	// declared dice si hay un bloque `sdd:` válido; iso es su valor.
	declared, iso bool
}

// readISODeclaration clasifica Declarations.SDD, SDDErr y ParseErr. Un axiom.yaml
// ilegible (ParseErr) cuenta como inválido: no se puede saber si declara el
// bloque, así que la proyección falla cerrada.
func readISODeclaration(declarations workspace.Declarations) isoDeclaration {
	switch {
	case declarations.SDDErr != nil:
		text := strings.Join(errorLines(declarations.SDDErr), "; ")
		contradiction := errors.Is(declarations.SDDErr, workspace.ErrISOWorkflowContradiction)
		if contradiction {
			text = strings.TrimSuffix(text, ": "+workspace.ErrISOWorkflowContradiction.Error())
		}
		return isoDeclaration{invalidText: text, contradiction: contradiction}
	case declarations.ParseErr != nil:
		return isoDeclaration{invalidText: strings.Join(errorLines(declarations.ParseErr), "; ")}
	case declarations.SDD != nil:
		return isoDeclaration{declared: true, iso: declarations.SDD.ISO}
	}
	return isoDeclaration{}
}

// reason es ISOProjection.Reason: el error y, salvo en la contradicción (cuyo
// texto ya nombra las dos correcciones), la continuación.
func (d isoDeclaration) reason(master string) string {
	if d.contradiction {
		return d.invalidText
	}
	return fmt.Sprintf("%s; corrígela con `axiom workspace validate --path %s`", d.invalidText, pathquote.Quote(master))
}

// finding es el hallazgo `iso_declaration_invalid` que estrecha el archivado.
func (d isoDeclaration) finding(master string) isoevidence.Finding {
	return isoevidence.Finding{
		Code:         isoevidence.CodeDeclarationInvalid,
		Stage:        isoevidence.StageArchive,
		Subject:      "sdd",
		Detail:       "la declaración `sdd:` de axiom.yaml no es válida: " + d.invalidText,
		Continuation: "ejecuta `axiom workspace validate --path " + pathquote.Quote(master) + "` y corrígela",
	}
}

// note es la línea de `notes` de D8, que se añade con o sin cambio seleccionado.
func (d isoDeclaration) note(master string) string {
	return fmt.Sprintf(
		"La declaración sdd: de axiom.yaml no es válida: %s. Corrígela con axiom workspace validate --path %s; mientras tanto ningún cambio puede archivarse.",
		d.invalidText, pathquote.Quote(master))
}

// changeKickoff devuelve el kickoff sellado del cambio seleccionado, o nil sin
// cambio, con el cambio archivado o sin sellar. El resolutor OpenSpec ya lo
// cargó en Governance; en los almacenes sin ficheros se lee de la carpeta del
// cambio (D21). Un kickoff ilegible no se proyecta aquí: el resolutor OpenSpec ya
// falla con su error nombrado (T-10).
func changeKickoff(status *Status) *kickoff.Kickoff {
	if status.ChangeName == nil || status.Archived != nil {
		return nil
	}
	if status.Governance != nil {
		sealed := status.Governance.Kickoff
		return &sealed
	}
	root := ""
	switch {
	case status.ChangeRoot != nil:
		root = *status.ChangeRoot
	case status.PlanningHome.Path != "":
		root = filepath.Join(status.PlanningHome.Path, "changes", *status.ChangeName)
	default:
		return nil
	}
	sealed, err := kickoff.Load(root)
	if err != nil {
		return nil
	}
	return sealed
}

// decorateISO proyecta la declaración ISO del proyecto y la instantánea del
// cambio (REQ-29.1, REQ-29.2, REQ-29.13), y estrecha el archivado ante una
// declaración inválida (D8) o un almacén sin ficheros (D21). Es aditivo: sin
// bloque `sdd:` y con un cambio no ISO no escribe nada, y el status de un cambio
// no ISO es idéntico al de antes.
//
// Las reglas del cambio se leen de su instantánea y nunca de axiom.yaml en vivo.
// Cada hallazgo de archivado va a `notes` y, solo cuando la ruta sería archive, a
// `blockedReasons` (el contrato de los consumidores prohíbe apply con motivos).
func decorateISO(status *Status, location workspace.Location) {
	declaration := readISODeclaration(location.Declarations)
	sealed := changeKickoff(status)
	changeISO := sealed != nil && sealed.IsISO()
	invalid := declaration.invalidText != ""
	if !declaration.declared && !invalid && !changeISO {
		return
	}

	projection := &ISOProjection{Project: isoValueNotISO}
	switch {
	case invalid:
		projection.Project = isoValueInvalid
		projection.Reason = declaration.reason(location.Master)
	case declaration.iso:
		projection.Project = isoValueISO
	}
	if status.ChangeName != nil && status.Archived == nil {
		projection.Change = isoValueNotISO
		if changeISO {
			projection.Change = isoValueISO
		}
	}
	projection.Matrix = status.isoMatrix
	status.ISO = projection

	changed := false
	if invalid {
		status.Notes = append(status.Notes, declaration.note(location.Master))
		changed = true
	} else if sealed != nil && changeISO != declaration.iso {
		status.Notes = append(status.Notes, fmt.Sprintf(
			"La instantánea ISO del cambio (sdd.iso: %t) difiere de la declaración vigente del proyecto (sdd.iso: %t); las reglas del cambio siguen su instantánea.",
			changeISO, declaration.iso))
		changed = true
	}

	if status.ChangeName != nil && status.Archived == nil {
		var findings, noted []isoevidence.Finding
		if invalid {
			findings = append(findings, declaration.finding(location.Master))
		}
		if changeISO {
			switch store := effectiveArtifactStore(status, location); store {
			case ArtifactStoreEngram, ArtifactStoreNone:
				finding := isoevidence.Finding{
					Code:         isoevidence.CodeStoreUnsupported,
					Stage:        isoevidence.StageArchive,
					Subject:      string(store),
					Detail:       fmt.Sprintf("el cambio es ISO y el almacén de artefactos es `%s`, que no guarda ficheros: no se puede evaluar la matriz de verificación", store),
					Continuation: "declara `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`",
				}
				findings = append(findings, finding)
				noted = append(noted, finding)
			}
		}
		if len(findings) > 0 {
			narrowArchive(status, findings, noted)
			changed = true
		}
	}

	if changed && status.PhaseInstructions != nil {
		refreshPhaseInstructions(status, location)
	}
}

// narrowArchive estrecha `archive` a `blocked` y, solo cuando la ruta sería
// archive, vuelca los motivos en blockedReasons y la lleva a resolve-blockers
// (lo que resolveNextRecommended devuelve con archive bloqueado y todo lo demás
// hecho). Los hallazgos de noted además van a `notes`.
func narrowArchive(status *Status, findings, noted []isoevidence.Finding) {
	status.Dependencies.Archive = DependencyBlocked
	status.Notes = append(status.Notes, findingReasons(noted)...)
	if status.NextRecommended != string(PhaseArchive) {
		return
	}
	status.BlockedReasons = append(status.BlockedReasons, findingReasons(findings)...)
	status.NextRecommended = "resolve-blockers"
}

// refreshPhaseInstructions vuelve a generar las instrucciones de fase para que
// reflejen la ruta y las notas añadidas, conservando la línea de archivado de la
// integración del contexto que decorateWorkspaceDeclarations ya había añadido.
func refreshPhaseInstructions(status *Status, location workspace.Location) {
	instructions := renderPhaseInstructions(*status)
	if line := contextPrepareLine(status, location); line != "" {
		instructions.Archive = append(instructions.Archive, line)
	}
	status.PhaseInstructions = &instructions
}
