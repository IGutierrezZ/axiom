package techcontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Esquemas de los informes de `check`, `prepare` y `verify`.
const (
	// CheckSchema es el esquema del informe de `archive context check`.
	CheckSchema = "axiom.archive-context-check/v1"
	// PrepareSchema es el esquema del informe de `archive context prepare`.
	PrepareSchema = "axiom.archive-context-plan/v1"
)

// now es el reloj de `prepare`; las pruebas lo fijan.
var now = time.Now

// Target es el cambio sobre el que actúan `check`, `prepare` y `verify`: lo
// resuelve el adaptador de la CLI a partir de la ubicación del workspace y del
// status SDD.
type Target struct {
	Location      workspace.Location
	Change        string
	ChangeRoot    string   // carpeta del cambio, absoluta
	AllowedRoots  []string // actionContext.allowedEditRoots del status
	ArtifactStore string   // almacén efectivo que el adaptador toma del status
}

// ContextEntryReport es una entrada del delta de contexto tal como la ve `check`.
type ContextEntryReport struct {
	ID       string `json:"id"` // C1..Cn
	Document string `json:"document"`
	Anchor   string `json:"anchor"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Create   bool   `json:"create,omitempty"` // la entrada `add` crea el documento
}

// ContextReport es el estado del delta de contexto: si el contexto mantenido está
// declarado, si el fichero existe, la decisión y sus entradas.
type ContextReport struct {
	Declared bool                 `json:"declared"`
	Present  bool                 `json:"present"`
	Decision string               `json:"decision"` // entries | no_impact | none
	Reason   string               `json:"reason,omitempty"`
	Entries  []ContextEntryReport `json:"entries,omitempty"`
}

// ContextItem es el trabajo de edición de una entrada del delta de contexto
// (REQ-24.59): el documento, el ancla, la acción, el motivo y el texto que integrar.
type ContextItem struct {
	ID       string `json:"id"`
	Document string `json:"document"`
	Anchor   string `json:"anchor"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Text     string `json:"text,omitempty"`
	Create   bool   `json:"create,omitempty"`
}

// Report es la salida de `check` (axiom.archive-context-check/v1) y de `prepare`
// (axiom.archive-context-plan/v1). Los errores impiden integrar; con errores quien
// llama sale con un código distinto de cero.
type Report struct {
	Schema       string        `json:"schema"`
	Change       string        `json:"change"`
	Context      ContextReport `json:"context"`
	ContextItems []ContextItem `json:"contextItems,omitempty"`
	Warnings     []string      `json:"warnings"`
	Errors       []string      `json:"errors"`
}

func newReport(schema, change string) Report {
	return Report{
		Schema:   schema,
		Change:   change,
		Context:  ContextReport{Decision: ContextNone},
		Warnings: []string{},
		Errors:   []string{},
	}
}

func (r *Report) fail(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// gathered es lo que `check` y `prepare` leen del disco antes de evaluar nada.
type gathered struct {
	maintained     bool
	contextData    []byte // nil si no hay delta de contexto
	contextPresent bool
}

// gather lee la declaración y el delta de contexto (REQ-24.62, REQ-30.19,
// REQ-30.30). Devuelve false si el resto de comprobaciones no tiene sentido:
// declaración ilegible o inválida, o un almacén sin ficheros.
func gather(t Target, report *Report) (*gathered, bool) {
	loc := t.Location
	declarations := loc.Declarations
	g := &gathered{}
	switch {
	case declarations.ParseErr != nil:
		report.fail("%s no se puede leer como YAML: %v. Continuación: corrige el YAML y vuelve a ejecutar el comando (`axiom workspace validate --path %s` comprueba la declaración)", pathquote.Quote(loc.ConfigPath), declarations.ParseErr, pathquote.Quote(loc.Master))
		return g, false
	case declarations.TechnicalContextErr != nil:
		report.fail("la declaración de %s no es válida: %v. Continuación: corrígela con `axiom workspace validate --path %s` y vuelve a ejecutar el comando", pathquote.Quote(loc.ConfigPath), declarations.TechnicalContextErr, pathquote.Quote(loc.Master))
		return g, false
	}

	g.maintained = declarations.TechnicalContext != nil && declarations.TechnicalContext.Maintained != nil
	report.Context.Declared = g.maintained

	if g.maintained && (t.ArtifactStore == "engram" || t.ArtifactStore == "none") {
		report.fail("el contexto técnico mantenido ('technical_context.maintained') exige un almacén de artefactos de ficheros y el almacén de este cambio es %q: sin la carpeta del cambio en ficheros no hay delta de contexto ni traza que integrar (REQ-24.62). Continuación: declara `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retira la declaración", t.ArtifactStore)
		return g, false
	}

	if g.maintained {
		data, err := os.ReadFile(filepath.Join(t.ChangeRoot, DeltaFileName))
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			report.fail("no se puede leer %s: %v", DeltaFileName, err)
		default:
			g.contextData, g.contextPresent = data, true
		}
	}
	report.Context.Present = g.contextPresent
	return g, true
}

// evaluation es el resultado de evaluar el delta de contexto.
type evaluation struct {
	decision     ContextDecision
	contextItems []ContextItem
	creates      map[string]bool // documentos que archive crea, por ruta relativa
}

// evaluate valida el delta de contexto (REQ-24.31, REQ-24.32, REQ-24.30): su
// formato, que cada documento esté bajo una ruta mantenida y no bajo una de
// referencia, y que cada ancla resuelva según su acción. Prepara los trabajos de
// edición. No escribe nada.
func evaluate(t Target, g *gathered, report *Report) evaluation {
	ev := evaluation{decision: ContextDecision{Decision: ContextNone}, creates: map[string]bool{}}
	if !g.maintained || !g.contextPresent {
		return ev
	}
	delta, err := ParseDelta(g.contextData)
	if err != nil {
		report.fail("%s (REQ-24.31): %v. Continuación: corrige el formato del delta de contexto (`## Entrada: <ruta>` con `- Ancla:`, `- Acción:` y `- Motivo:`, o `Sin impacto: <motivo>`)", DeltaFileName, err)
	}
	report.Context.Reason = delta.Reason
	switch {
	case err != nil:
		return ev
	case delta.NoImpact:
		report.Context.Decision = ContextNoImpact
		ev.decision = ContextDecision{Decision: ContextNoImpact, Reason: delta.Reason}
		return ev
	}
	report.Context.Decision = ContextEntries
	ev.decision = ContextDecision{Decision: ContextEntries}

	scope := ScopeOf(t.Location)
	for _, entry := range delta.Entries {
		report.Context.Entries = append(report.Context.Entries, ContextEntryReport{ID: entry.ID, Document: entry.Document, Anchor: entry.Anchor.String(), Action: string(entry.Action), Reason: entry.Reason})
		if err := scope.CheckDocument(entry); err != nil {
			report.fail("%v", err)
			continue
		}
		data, readErr := os.ReadFile(scope.Resolve(entry.Document))
		exists := readErr == nil
		if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
			report.fail("la entrada %s (línea %d): no se puede leer %s: %v", entry.ID, entry.Line, entry.Document, readErr)
			continue
		}
		create, err := CheckAnchor(entry, data, exists)
		if err != nil {
			report.fail("%v", err)
			continue
		}
		if create {
			ev.creates[entry.Document] = true
			report.Context.Entries[len(report.Context.Entries)-1].Create = true
		}
		ev.contextItems = append(ev.contextItems, ContextItem{
			ID:       entry.ID,
			Document: entry.Document,
			Anchor:   entry.Anchor.String(),
			Action:   string(entry.Action),
			Reason:   entry.Reason,
			Text:     entry.Body,
			Create:   create,
		})
	}
	return ev
}

// entries son las entradas de la traza del delta de contexto: una por entrada.
func (ev evaluation) entries() []TraceEntry {
	var entries []TraceEntry
	for _, item := range ev.contextItems {
		action := ActionModified
		switch Action(item.Action) {
		case Add:
			action = ActionAdded
		case Remove:
			action = ActionRemoved
		}
		entries = append(entries, TraceEntry{ID: item.ID, Kind: EntryContext, Document: item.Document, Anchor: item.Anchor, Action: action, State: StatePending, Reason: item.Reason})
	}
	return entries
}

func prepareCommand(t Target) string {
	return fmt.Sprintf("axiom archive context prepare --path %s --change %s", pathquote.Quote(t.Location.WorkspaceRoot), pathquote.ShellWord(t.Change))
}

// Check hace las validaciones del contexto técnico sin exigir git ni autorización
// de edición, y no escribe nada, tampoco en la carpeta del cambio (REQ-24.42,
// REQ-30.15). Un delta de contexto ausente es un aviso. Quien llama sale con un
// código distinto de cero si Report.Errors no está vacío; el error devuelto es
// solo de ejecución.
func Check(ctx context.Context, t Target) (Report, error) {
	report := newReport(CheckSchema, t.Change)
	g, ok := gather(t, &report)
	if !ok {
		return report, ctx.Err()
	}
	evaluate(t, g, &report)
	if g.maintained && !g.contextPresent {
		report.warn("el cambio no lleva delta de contexto %s y el contexto técnico mantenido está declarado: registra el impacto previsto o «Sin impacto: <motivo>» antes de archivar; la ausencia nunca se asume como «Sin impacto» (REQ-24.36)", DeltaFileName)
	}
	return report, ctx.Err()
}

// inputsDigest calcula `inputs_sha256` con la misma fórmula que `axiom sdd
// integrate` cuando no hay deltas de dominio: así cada verbo reconoce el plan del
// otro. Un delta de contexto nil (ausente) y uno vacío producen huellas distintas.
func inputsDigest(contextDelta, declaration []byte) string {
	hash := sha256.New()
	frame := func(tag string, data []byte) {
		fmt.Fprintf(hash, "%s\x00%d\x00", tag, len(data))
		hash.Write(data)
	}
	if contextDelta == nil {
		frame("context-absent", nil)
	} else {
		frame("context", contextDelta)
	}
	frame("declaration", declaration)
	return hex.EncodeToString(hash.Sum(nil))
}

// declarationSnapshot es la instantánea determinista de las declaraciones de
// axiom.yaml que influyen en la integración. La clave `baseline` se serializa
// siempre vacía: este paquete no integra baselines.
func declarationSnapshot(declarations workspace.Declarations) []byte {
	data, err := yaml.Marshal(struct {
		Baseline         *struct{}                         `yaml:"baseline"`
		TechnicalContext *workspace.TechnicalContextConfig `yaml:"technical_context"`
	}{nil, declarations.TechnicalContext})
	if err != nil {
		return []byte("declaración no serializable: " + err.Error())
	}
	return data
}

// reconcile decide qué hace `prepare` cuando ya existe un plan: si todas sus
// entradas figuran como integradas rechaza como «ya integrado»; si las entradas
// cambiaron desde `prepare` rechaza con la continuación; en otro caso lo reemite
// con su HEAD y su foto previa, que mantienen la protección de `verify`.
func reconcile(t Target, existing Trace, digest string) (Trace, error) {
	integrated := len(existing.Entries) > 0
	for _, entry := range existing.Entries {
		integrated = integrated && entry.State == StateIntegrated
	}
	if integrated {
		return Trace{}, &AlreadyIntegratedError{Change: existing.Change, Target: t}
	}
	if existing.InputsSHA256 != digest {
		return Trace{}, &PlanChangedError{Change: existing.Change, Target: t}
	}
	return existing, nil
}

// AlreadyIntegratedError es el rechazo de preparar de nuevo un cambio cuyas
// entradas de contexto ya figuran como integradas.
type AlreadyIntegratedError struct {
	Change string
	Target Target
}

func (e *AlreadyIntegratedError) Error() string {
	return fmt.Sprintf("el cambio %q figura ya integrado: todas sus entradas de contexto están marcadas como integradas en %s y no se vuelven a aplicar. Continuación: comprueba la integración con `axiom archive context verify --path %s --change %s` o consulta la traza con `axiom archive context show --path %s --change %s`",
		e.Change, TraceFileName, pathquote.Quote(e.Target.Location.WorkspaceRoot), pathquote.ShellWord(e.Change), pathquote.Quote(e.Target.Location.WorkspaceRoot), pathquote.ShellWord(e.Change))
}

// PlanChangedError es el rechazo de reemitir un plan cuyas entradas ya no son las
// actuales: el delta de contexto o la declaración cambiaron desde `prepare`.
type PlanChangedError struct {
	Change string
	Target Target
}

func (e *PlanChangedError) Error() string {
	return fmt.Sprintf("el plan de integración del cambio %q (%s) ya no corresponde a las entradas actuales: cambiaron el delta de contexto o la declaración de axiom.yaml desde `prepare`. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `%s`", e.Change, TraceFileName, TraceFileName, prepareCommand(e.Target))
}

// Prepare valida el delta, comprueba las precondiciones de la integración
// (REQ-24.49) y la autorización de edición de las rutas mantenidas fuera del
// repositorio de especificaciones (REQ-24.37), registra el HEAD y la foto de
// suciedad de cada repositorio afectado y escribe `<carpeta del cambio>/
// integration.yaml`: nunca escribe en las rutas mantenidas. Con un plan existente
// lo reemite si las entradas coinciden, rechaza como «ya integrado» si todas lo
// están, y rechaza con la continuación si las entradas cambiaron. Los rechazos de
// validación van en Report.Errors, sin plan; el error devuelto es solo de
// ejecución.
func Prepare(ctx context.Context, t Target) (Trace, Report, error) {
	report := newReport(PrepareSchema, t.Change)
	g, ok := gather(t, &report)
	if !ok || len(report.Errors) > 0 {
		return Trace{}, report, ctx.Err()
	}

	digest := inputsDigest(g.contextData, declarationSnapshot(t.Location.Declarations))
	planPath := filepath.Join(t.ChangeRoot, TraceFileName)
	existing, readErr := ReadTrace(planPath)
	reuse := readErr == nil
	var foreign *ForeignPlanError
	switch {
	case errors.As(readErr, &foreign):
		report.fail("%v", readErr)
		return Trace{}, report, ctx.Err()
	case readErr != nil && !errors.Is(readErr, fs.ErrNotExist):
		report.fail("no se puede leer el plan existente: %v. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `%s`", readErr, TraceFileName, prepareCommand(t))
		return Trace{}, report, ctx.Err()
	}
	if reuse {
		kept, err := reconcile(t, existing, digest)
		if err != nil {
			report.fail("%v", err)
			return Trace{}, report, ctx.Err()
		}
		existing = kept
	}

	ev := evaluate(t, g, &report)
	if g.maintained && !g.contextPresent {
		report.fail("el cambio no lleva delta de contexto %s y el contexto técnico mantenido está declarado: archive se detiene hasta que se registre el impacto previsto o «Sin impacto: <motivo>»; la ausencia nunca se asume como «Sin impacto» (REQ-24.36, REQ-24.56). Continuación: redacta %s en la carpeta del cambio", DeltaFileName, DeltaFileName)
	}
	if len(report.Errors) > 0 {
		return Trace{}, report, ctx.Err()
	}
	report.ContextItems = ev.contextItems
	if reuse {
		return existing, report, ctx.Err()
	}

	authorizeEdits(ctx, t, ev, &report)
	if len(report.Errors) > 0 {
		report.ContextItems = nil
		return Trace{}, report, ctx.Err()
	}

	capture, err := captureRepositories(ctx, t, ev)
	var stop *PreconditionError
	if errors.As(err, &stop) {
		for _, problem := range stop.Problems {
			report.fail("%s", problem)
		}
		report.ContextItems = nil
		return Trace{}, report, ctx.Err()
	}
	if err != nil {
		report.fail("%v", err)
		report.ContextItems = nil
		return Trace{}, report, ctx.Err()
	}
	for _, warning := range capture.Warnings {
		report.warn("%s", warning)
	}

	trace := Trace{
		Schema:       PlanSchema,
		Change:       t.Change,
		PreparedAt:   now().UTC().Truncate(time.Second),
		InputsSHA256: digest,
		Options:      traceOptions{SupersededRequirements: []string{}},
		Repositories: capture.Repositories,
		IDMap:        []any{},
		Documents:    []any{},
		Context:      ev.decision,
		Entries:      ev.entries(),
	}
	if err := WriteTrace(planPath, trace); err != nil {
		return Trace{}, report, err
	}
	return trace, report, ctx.Err()
}

// authorizeEdits comprueba que cada documento que edita el delta de contexto está
// bajo alguna raíz de edición autorizada del cambio (actionContext.allowedEditRoots).
// El repositorio de especificaciones siempre lo está; los de código hermanos solo
// con una autorización por cambio, que nunca se concede de forma implícita
// (REQ-24.37).
func authorizeEdits(ctx context.Context, t Target, ev evaluation, report *Report) {
	allowed := t.AllowedRoots
	if len(allowed) == 0 {
		allowed = []string{t.Location.WorkspaceRoot}
	}
	scope := ScopeOf(t.Location)
	for _, item := range ev.contextItems {
		target := resolveDeepest(scope.Resolve(item.Document))
		authorized := false
		for _, root := range allowed {
			if pathidentity.Contains(root, target) {
				authorized = true
				break
			}
		}
		if authorized {
			continue
		}
		root := deepestExistingDir(target)
		if repo, err := openRepository(ctx, target); err == nil {
			root = repo.root
		}
		report.fail("la entrada %s edita %s, que no está bajo ninguna raíz de edición autorizada del cambio (actionContext.allowedEditRoots): falta la autorización de edición de %s. Continuación: concédela por cambio con `axiom sdd-attempt grant --cwd %s --change %s --root %s` (más `--change-instance`, `--request-id`, `--actor` y `--reason`); nunca se concede de forma implícita (REQ-24.37)",
			item.ID, item.Document, pathquote.Quote(root), pathquote.Quote(t.Location.WorkspaceRoot), pathquote.ShellWord(t.Change), pathquote.Quote(root))
	}
}

// affectedPath es un documento que la integración edita. Creates indica que
// archive lo crea (REQ-24.55): queda excluido de la precondición de cambios sin
// confirmar (REQ-24.49).
type affectedPath struct {
	Path    string // absoluta
	Creates bool
}

// PreconditionError es la parada de REQ-24.49: un documento afectado no está en
// un repositorio git, o tiene cambios sin confirmar o sin seguimiento, o hay
// demasiada suciedad ajena. Cada problema nombra los ficheros y la continuación.
type PreconditionError struct {
	Problems []string
}

func (e *PreconditionError) Error() string { return strings.Join(e.Problems, "; ") }

type repositoryCapture struct {
	Repositories []TraceRepository
	Warnings     []string
}

// repoWork reúne un repositorio con los documentos afectados que contiene.
type repoWork struct {
	repo  repository
	files []affectedPath
}

// captureRepositories registra la línea base de los repositorios que la
// integración toca o vigila: su HEAD y la foto de su suciedad previa. Los
// documentos afectados deben estar dentro de la carpeta maestra una vez resueltos
// los enlaces simbólicos; un repositorio vigilado que no existe o no es git se
// avisa y nunca detiene. Solo lee.
func captureRepositories(ctx context.Context, t Target, ev evaluation) (repositoryCapture, error) {
	loc := t.Location
	master := resolveDeepest(filepath.Clean(loc.Master))
	scope := ScopeOf(loc)

	var affected []affectedPath
	for _, item := range ev.contextItems {
		affected = append(affected, affectedPath{Path: scope.Resolve(item.Document), Creates: item.Create})
	}

	var works []*repoWork
	var problems, warnings []string
	workOf := func(repo repository) *repoWork {
		for _, work := range works {
			if pathidentity.SameDirectory(work.repo.root, repo.root) {
				return work
			}
		}
		work := &repoWork{repo: repo}
		works = append(works, work)
		return work
	}

	for _, document := range affected {
		resolved := resolveDeepest(filepath.Clean(document.Path))
		if !pathidentity.Contains(master, resolved) {
			return repositoryCapture{}, fmt.Errorf("la ruta %s resuelve a %s, fuera de la carpeta maestra %s: un enlace simbólico no puede sacar la integración de ella", document.Path, resolved, master)
		}
		repo, err := openRepository(ctx, resolved)
		if errors.Is(err, gitcommon.ErrNotGitRepository) {
			problems = append(problems, fmt.Sprintf("el documento %s no está dentro de un repositorio git: la verificación por diff de la integración requiere un repositorio git; versiona su directorio con `git init` y confirma su estado actual, o declara el documento en un repositorio versionado", displayPath(master, resolved)))
			continue
		}
		if err != nil {
			return repositoryCapture{}, err
		}
		work := workOf(repo)
		work.files = append(work.files, affectedPath{Path: resolved, Creates: document.Creates})
	}

	for _, dir := range watchedDirectories(loc) {
		resolved := resolveDeepest(filepath.Clean(dir))
		repo, err := openRepository(ctx, resolved)
		if errors.Is(err, gitcommon.ErrNotGitRepository) {
			warnings = append(warnings, fmt.Sprintf("el directorio %s que se vigila no es un repositorio git; se omite", displayPath(master, resolved)))
			continue
		}
		if err != nil {
			return repositoryCapture{}, err
		}
		workOf(repo)
	}

	type captured struct {
		repository TraceRepository
		warnings   []string
	}
	var captures []captured
	for _, work := range works {
		name := repositoryName(master, work.repo.root)
		head, err := work.repo.head(ctx)
		if err != nil {
			return repositoryCapture{}, err
		}
		entries, err := work.repo.status(ctx)
		if err != nil {
			return repositoryCapture{}, err
		}
		if len(entries) > MaxDirtyEntries {
			problems = append(problems, fmt.Sprintf("el repositorio %s tiene %d entradas sin confirmar, por encima del límite de %d que el plan registra: %s", name, len(entries), MaxDirtyEntries, preparePreconditionContinuation(t)))
			continue
		}
		if dirty := dirtyAffected(work, entries); len(dirty) > 0 {
			problems = append(problems, fmt.Sprintf("el repositorio %s tiene cambios sin confirmar en documentos que la integración modifica: %s; %s", name, strings.Join(dirty, ", "), preparePreconditionContinuation(t)))
		}

		photo := make([]DirtyEntry, 0, len(entries))
		for _, entry := range entries {
			entryPath := strings.TrimSuffix(entry.Path, "/")
			fingerprint, err := workingTreeFingerprint(filepath.Join(work.repo.root, filepath.FromSlash(entryPath)))
			if err != nil {
				return repositoryCapture{}, fmt.Errorf("calcular la huella de %s en el repositorio %s: %w", entryPath, name, err)
			}
			photo = append(photo, DirtyEntry{Path: entryPath, SHA256: fingerprint})
		}
		var notes []string
		if behind, note := work.repo.behind(ctx); behind > 0 {
			notes = append(notes, fmt.Sprintf("el repositorio %s va %d commit(s) por detrás de su upstream (el dato procede de las referencias locales: Axiom nunca ejecuta fetch); si otros han integrado cambios, actualiza el repositorio antes de archivar", name, behind))
		} else if note != "" {
			notes = append(notes, fmt.Sprintf("repositorio %s: %s", name, note))
		}
		captures = append(captures, captured{repository: TraceRepository{Repo: name, Head: head, Dirty: photo}, warnings: notes})
	}

	if len(problems) > 0 {
		return repositoryCapture{}, &PreconditionError{Problems: problems}
	}
	sort.SliceStable(captures, func(i, j int) bool { return captures[i].repository.Repo < captures[j].repository.Repo })
	result := repositoryCapture{Warnings: warnings}
	for _, capture := range captures {
		result.Repositories = append(result.Repositories, capture.repository)
		result.Warnings = append(result.Warnings, capture.warnings...)
	}
	return result, nil
}

func preparePreconditionContinuation(t Target) string {
	return "confirma, guarda (`git stash -u`) o descarta los cambios y vuelve a ejecutar `" + prepareCommand(t) + "`"
}

// watchedDirectories son los directorios cuyos repositorios se vigilan sin detener
// la integración: el de especificaciones, los de las rutas mantenidas y de
// referencia y los de los roles declarados. Solo los que existen.
func watchedDirectories(loc workspace.Location) []string {
	candidates := []string{loc.WorkspaceRoot}
	scope := ScopeOf(loc)
	for _, declared := range append(append([]string(nil), scope.Maintained...), scope.Reference...) {
		candidates = append(candidates, scope.Resolve(declared))
	}
	if loc.Config != nil {
		roles := make([]string, 0, len(loc.Config.Roles))
		for name := range loc.Config.Roles {
			roles = append(roles, name)
		}
		sort.Strings(roles)
		for _, name := range roles {
			for _, repo := range loc.Config.Roles[name].Repositories {
				location := filepath.Clean(strings.TrimSpace(repo.Path))
				if !filepath.IsAbs(location) {
					location = filepath.Join(loc.Master, location)
				}
				candidates = append(candidates, location)
			}
		}
	}
	var watched []string
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			candidate = filepath.Dir(candidate)
		}
		watched = append(watched, candidate)
	}
	return watched
}

// dirtyAffected devuelve, en orden y con su estado, los documentos afectados que
// tienen cualquier estado en `git status`: índice, árbol de trabajo o sin
// seguimiento. Los documentos que archive crea quedan excluidos.
func dirtyAffected(work *repoWork, entries []statusEntry) []string {
	var dirty []string
	for _, file := range work.files {
		if file.Creates {
			continue
		}
		relative, err := filepath.Rel(work.repo.root, file.Path)
		if err != nil {
			continue
		}
		relative = filepath.ToSlash(relative)
		for _, entry := range entries {
			if samePath(entry.Path, relative) {
				dirty = append(dirty, fmt.Sprintf("%s (%s)", relative, describeStatus(entry.Code)))
			}
		}
	}
	sort.Strings(dirty)
	return dirty
}

// samePath compara dos rutas relativas de git, sin distinguir mayúsculas donde el
// sistema de ficheros no las distingue.
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// describeStatus describe en castellano un código XY de `git status`.
func describeStatus(code string) string {
	if code == "??" {
		return "sin seguimiento"
	}
	staged := code[0] != ' ' && code[0] != '?'
	worktree := code[1] != ' ' && code[1] != '?'
	switch {
	case staged && worktree:
		return "cambios en el índice y en el árbol de trabajo"
	case staged:
		return "cambios en el índice"
	default:
		return "cambios en el árbol de trabajo"
	}
}

// repositoryName es el nombre de un repositorio en el plan: su ruta relativa a la
// carpeta maestra, con `/`, y `.` si es la propia carpeta maestra.
func repositoryName(master, root string) string {
	relative, err := filepath.Rel(master, root)
	if err != nil {
		return filepath.ToSlash(root)
	}
	return filepath.ToSlash(relative)
}

// displayPath muestra una ruta relativa a la carpeta maestra si está dentro de ella.
func displayPath(master, p string) string {
	if relative, err := filepath.Rel(master, p); err == nil && !strings.HasPrefix(filepath.ToSlash(relative), "../") && relative != ".." {
		return filepath.ToSlash(relative)
	}
	return p
}

// workingTreeFingerprint es la huella de un fichero del árbol de trabajo: el
// SHA-256 de su contenido, vacía si no existe (borrado). Un enlace simbólico o un
// directorio, como un repositorio anidado sin seguimiento, se identifican por su
// destino o por su naturaleza. Se calcula sobre el árbol de trabajo y nunca contra
// los bytes de un commit, que `core.autocrlf` puede normalizar.
func workingTreeFingerprint(file string) (string, error) {
	info, err := os.Lstat(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", nil
	case err != nil:
		return "", err
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(file)
		if err != nil {
			return "", err
		}
		return sha256Hex([]byte("symlink:" + target)), nil
	case info.IsDir():
		return sha256Hex([]byte("dir")), nil
	}
	handle, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer handle.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, handle); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
