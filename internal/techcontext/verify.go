package techcontext

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// VerifySchema es el esquema del informe de `archive context verify`.
const VerifySchema = "axiom.archive-context-verify/v1"

// Comprobaciones de la verificación nativa (REQ-24.51) que nombra Violation.Check.
const (
	// ViolationPresence: una entrada esperada no figura en el plan, o su documento
	// no aparece en el diff de archive.
	ViolationPresence = "presence"
	// ViolationAnchor: un ancla no existe tras la edición exactamente una vez, o
	// existe una acción `remove`.
	ViolationAnchor = "anchor"
	// ViolationScope: el diff de un repositorio afectado toca algo fuera del
	// conjunto permitido.
	ViolationScope = "scope"
	// ViolationTrace: el plan de integración no existe, no es analizable, no es un
	// plan de contexto o no es del cambio.
	ViolationTrace = "trace"
)

// Violation es un fallo de la verificación: la comprobación, la entrada y el
// fichero que la provocan y su explicación, con la resolución cuando la hay.
type Violation struct {
	Check  string `json:"check"` // presence | anchor | scope | trace
	Entry  string `json:"entry,omitempty"`
	File   string `json:"file,omitempty"`
	Detail string `json:"detail"`
}

// VerifyReport es la salida de `archive context verify`. Passed es verdadero solo
// sin violaciones; Violations nunca es nil.
type VerifyReport struct {
	Schema     string      `json:"schema"`
	Change     string      `json:"change"`
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations"`
}

// Verify es la verificación nativa de la integración del contexto técnico
// (REQ-24.51, REQ-24.57): determinista, sin modelo y con git de solo lectura.
// Comprueba, contra el plan que fijó `prepare`:
//
//   - trace: `integration.yaml` existe, se decodifica de forma estricta, es un plan
//     de contexto y es del cambio;
//   - presence: las entradas esperadas, recalculadas desde los ficheros actuales del
//     cambio, coinciden con las del plan, y cada documento con entradas (salvo
//     `remove` y «Sin impacto») aparece en el diff de archive;
//   - anchor: cada ancla de contexto resuelve exactamente una sección (`add` y
//     `modify`) o ninguna (`remove`);
//   - scope: cada ruta del diff de archive de cada repositorio afectado pertenece al
//     conjunto permitido, por identidad y nunca por texto.
//
// El diff de archive es el del repositorio contra el HEAD del plan (no el actual: un
// commit entre `prepare` y `verify` no oculta nada), sin las entradas de la foto
// previa cuya huella no ha cambiado. Si todo pasa marca las entradas como integradas,
// registra `created` en las entradas `add` que crearon el documento y escribe
// `verified_at`: solo toca `integration.yaml`. Si algo falla lista todas las
// violaciones y no modifica nada. El error devuelto es solo de ejecución.
func Verify(ctx context.Context, t Target) (VerifyReport, error) {
	report := VerifyReport{Schema: VerifySchema, Change: t.Change, Violations: []Violation{}}
	planPath := filepath.Join(t.ChangeRoot, TraceFileName)
	prepare := prepareCommand(t)

	trace, err := ReadTrace(planPath)
	var foreign *ForeignPlanError
	switch {
	case errors.Is(err, fs.ErrNotExist):
		report.add(ViolationTrace, "", TraceFileName, "el cambio no tiene plan de integración: la verificación compara contra lo que fijó `prepare`. Continuación: ejecuta `%s`", prepare)
		return report, ctx.Err()
	case errors.As(err, &foreign):
		report.add(ViolationTrace, "", TraceFileName, "%v", err)
		return report, ctx.Err()
	case err != nil:
		report.add(ViolationTrace, "", TraceFileName, "el plan de integración no se puede leer: %v. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `%s`", err, TraceFileName, prepare)
		return report, ctx.Err()
	case trace.Change != t.Change:
		report.add(ViolationTrace, "", TraceFileName, "el plan es del cambio %q y se verifica el cambio %q. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `%s`", trace.Change, t.Change, TraceFileName, prepare)
		return report, ctx.Err()
	}

	v := &verifier{
		ctx:     ctx,
		t:       t,
		trace:   trace,
		master:  resolveDeepest(filepath.Clean(t.Location.Master)),
		scope:   ScopeOf(t.Location),
		report:  &report,
		prepare: prepare,
	}
	v.checkExpectedEntries()
	if err := v.collectChanges(); err != nil {
		return report, err
	}
	v.checkPresence()
	v.checkAnchors()
	v.checkScope()
	if len(report.Violations) > 0 {
		return report, ctx.Err()
	}

	updated, err := v.markVerified()
	if err != nil {
		return report, err
	}
	if err := WriteTrace(planPath, updated); err != nil {
		return report, err
	}
	report.Passed = true
	return report, ctx.Err()
}

func (r *VerifyReport) add(check, entry, file, format string, args ...any) {
	r.Violations = append(r.Violations, Violation{Check: check, Entry: entry, File: file, Detail: fmt.Sprintf(format, args...)})
}

// verifiedRepository es un repositorio de la línea base del plan, abierto de nuevo.
type verifiedRepository struct {
	name  string
	repo  repository
	head  string
	photo map[string]string // huella de la foto previa por ruta del repositorio
}

// changedFile es una ruta del diff de archive.
type changedFile struct {
	repo     *verifiedRepository
	rel      string // relativa a la raíz del repositorio, con '/'
	abs      string
	deleted  bool
	isDir    bool // un repositorio anidado sin seguimiento
	inChange bool // dentro de la carpeta del cambio
}

type verifier struct {
	ctx     context.Context
	t       Target
	trace   Trace
	master  string
	scope   Scope
	report  *VerifyReport
	prepare string

	changed []*changedFile
}

func (v *verifier) violate(check, entry, file, format string, args ...any) {
	v.report.add(check, entry, file, format, args...)
}

func (v *verifier) display(abs string) string { return displayPath(v.master, abs) }

// entryKey identifica una entrada de la traza sin su estado. La acción `created`,
// que verify registra, equivale a `added`, que es la del delta.
func entryKey(entry TraceEntry) string {
	action := entry.Action
	if action == ActionCreated {
		action = ActionAdded
	}
	return strings.Join([]string{string(entry.Kind), entry.Document, entry.Anchor, string(action), entry.ID, entry.Reason}, "\x1f")
}

func describeEntry(entry TraceEntry) string {
	return fmt.Sprintf("%s (%s, documento %s)", entry.ID, entry.Kind, entry.Document)
}

func traceAction(action Action) TraceAction {
	switch action {
	case Add:
		return ActionAdded
	case Remove:
		return ActionRemoved
	default:
		return ActionModified
	}
}

// checkExpectedEntries recalcula las entradas esperadas desde los ficheros actuales
// del cambio (el delta de contexto y la declaración) y comprueba que coinciden con
// las del plan.
func (v *verifier) checkExpectedEntries() {
	scratch := newReport(CheckSchema, v.t.Change)
	g, ok := gather(v.t, &scratch)
	if !ok || len(scratch.Errors) > 0 {
		for _, problem := range scratch.Errors {
			v.violate(ViolationPresence, "", "", "no se pueden recalcular las entradas esperadas desde los ficheros actuales del cambio: %s", problem)
		}
		return
	}
	digest := inputsDigest(g.contextData, declarationSnapshot(v.t.Location.Declarations))
	if digest != v.trace.InputsSHA256 {
		v.violate(ViolationPresence, "", TraceFileName, "el delta de contexto o la declaración de axiom.yaml cambiaron desde `prepare`: las entradas esperadas ya no corresponden al plan. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `%s`", TraceFileName, v.prepare)
	}

	expected := map[string]TraceEntry{}
	var order []string
	if g.maintained && g.contextPresent {
		delta, err := ParseDelta(g.contextData)
		if err != nil {
			v.violate(ViolationPresence, "", DeltaFileName, "%s ya no es válido: %v. Continuación: corrige el delta de contexto y vuelve a preparar con `%s`", DeltaFileName, err, v.prepare)
		} else if !delta.NoImpact {
			for _, entry := range delta.Entries {
				candidate := TraceEntry{ID: entry.ID, Kind: EntryContext, Document: entry.Document, Anchor: entry.Anchor.String(), Action: traceAction(entry.Action), Reason: entry.Reason}
				key := entryKey(candidate)
				if _, seen := expected[key]; !seen {
					order = append(order, key)
				}
				expected[key] = candidate
			}
		}
	}

	recorded := map[string]TraceEntry{}
	var recordedOrder []string
	for _, entry := range v.trace.Entries {
		key := entryKey(entry)
		if _, seen := recorded[key]; !seen {
			recordedOrder = append(recordedOrder, key)
		}
		recorded[key] = entry
	}
	for _, key := range order {
		if _, ok := recorded[key]; !ok {
			entry := expected[key]
			v.violate(ViolationPresence, entry.ID, entry.Document, "la entrada %s figura en los ficheros del cambio y no en el plan de integración. Continuación: vuelve a preparar con `%s`", describeEntry(entry), v.prepare)
		}
	}
	for _, key := range recordedOrder {
		if _, ok := expected[key]; !ok {
			entry := recorded[key]
			v.violate(ViolationPresence, entry.ID, entry.Document, "la entrada %s del plan ya no está en los ficheros del cambio. Continuación: vuelve a preparar con `%s`", describeEntry(entry), v.prepare)
		}
	}
}

// pathKey normaliza una ruta de git para compararla sin distinguir mayúsculas donde
// el sistema de ficheros no las distingue.
func pathKey(p string) string {
	if samePath("a", "A") {
		return strings.ToLower(p)
	}
	return p
}

// collectChanges calcula, en cada repositorio de la línea base del plan, el diff de
// archive con ChangedSince desde el HEAD registrado, sin las entradas de la foto
// previa cuya huella no ha cambiado. Un repositorio sin commits se compara con el
// árbol vacío.
func (v *verifier) collectChanges() error {
	for _, recorded := range v.trace.Repositories {
		dir := filepath.Join(v.master, filepath.FromSlash(recorded.Repo))
		repo, err := openRepository(v.ctx, dir)
		switch {
		case errors.Is(err, gitcommon.ErrNotGitRepository):
			v.violate(ViolationScope, "", recorded.Repo, "el repositorio %s que registró `prepare` ya no es un repositorio git: el alcance de la integración solo se puede comprobar por diff. Continuación: restablece el repositorio y vuelve a ejecutar `%s`, o vuelve a preparar con `%s`", recorded.Repo, v.verifyCommand(), v.prepare)
			continue
		case err != nil:
			return err
		}
		if !pathidentity.SameDirectory(repo.root, dir) {
			v.violate(ViolationScope, "", recorded.Repo, "el directorio %s que registró `prepare` ya no es la raíz de un repositorio git (ahora pertenece al repositorio de %s). Continuación: restablece el repositorio y vuelve a ejecutar `%s`, o vuelve a preparar con `%s`", recorded.Repo, pathquote.Quote(repo.root), v.verifyCommand(), v.prepare)
			continue
		}
		if recorded.Head != "" {
			exists, err := repo.commitExists(v.ctx, recorded.Head)
			if err != nil {
				return err
			}
			if !exists {
				v.violate(ViolationScope, "", recorded.Repo, "el commit %s que registró `prepare` en el repositorio %s ya no existe (¿se reescribió la historia?): sin él no se puede calcular el diff de archive. Continuación: vuelve a preparar con `%s`", recorded.Head, recorded.Repo, v.prepare)
				continue
			}
		}

		verified := &verifiedRepository{name: recorded.Repo, repo: repo, head: recorded.Head, photo: map[string]string{}}
		for _, entry := range recorded.Dirty {
			verified.photo[pathKey(entry.Path)] = entry.SHA256
		}
		paths, err := ChangedSince(v.ctx, repo.root, recorded.Head)
		if err != nil {
			return err
		}
		for _, abs := range paths {
			rel, err := filepath.Rel(repo.root, abs)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if sum, ok := verified.photo[pathKey(rel)]; ok {
				if current, err := workingTreeFingerprint(abs); err == nil && current == sum {
					continue // suciedad previa que no ha cambiado: no es del archivado
				}
			}
			info, statErr := os.Lstat(abs)
			v.changed = append(v.changed, &changedFile{
				repo:     verified,
				rel:      rel,
				abs:      abs,
				deleted:  errors.Is(statErr, fs.ErrNotExist),
				isDir:    statErr == nil && info.IsDir(),
				inChange: pathidentity.Contains(v.t.ChangeRoot, filepath.Dir(abs)),
			})
		}
	}
	sort.SliceStable(v.changed, func(i, j int) bool {
		if v.changed[i].repo.name != v.changed[j].repo.name {
			return v.changed[i].repo.name < v.changed[j].repo.name
		}
		return v.changed[i].rel < v.changed[j].rel
	})
	return nil
}

func (v *verifier) verifyCommand() string {
	return fmt.Sprintf("axiom archive context verify --path %s --change %s", pathquote.Quote(v.t.Location.WorkspaceRoot), pathquote.ShellWord(v.t.Change))
}

// appears dice si el documento está entre los ficheros del diff de archive que
// siguen existiendo, por identidad.
func (v *verifier) appears(abs string) bool {
	for _, c := range v.changed {
		if !c.deleted && underPath(abs, c.abs) {
			return true
		}
	}
	return false
}

// checkPresence exige que cada documento con entradas de contexto aparezca en el
// diff de archive. Las entradas `remove` y «Sin impacto» no lo exigen.
func (v *verifier) checkPresence() {
	for _, entry := range v.trace.Entries {
		if entry.Action == ActionRemoved {
			continue
		}
		if abs := v.scope.Resolve(entry.Document); !v.appears(abs) {
			v.violate(ViolationPresence, entry.ID, entry.Document, "la entrada de contexto %s no aparece en el diff de archive: el documento %s no se modificó (o no se creó). Continuación: edita el documento según la entrada y vuelve a ejecutar `%s`", entry.ID, entry.Document, v.verifyCommand())
		}
	}
}

// checkAnchors comprueba las anclas: cada ancla de contexto resuelve exactamente
// una sección con `add` y `modify` y ninguna con `remove`.
func (v *verifier) checkAnchors() {
	for _, entry := range v.trace.Entries {
		anchor, err := ParseAnchor(entry.Anchor)
		if err != nil {
			v.violate(ViolationAnchor, entry.ID, entry.Document, "el ancla %q del plan no es válida: %v", entry.Anchor, err)
			continue
		}
		data, err := os.ReadFile(v.scope.Resolve(entry.Document))
		exists := err == nil
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			v.violate(ViolationAnchor, entry.ID, entry.Document, "no se puede leer %s para comprobar el ancla %q: %v", entry.Document, entry.Anchor, err)
			continue
		}
		var found []HeadingRef
		if exists {
			found = ResolveAnchor(data, anchor)
		}
		describe := func() string {
			parts := make([]string, len(found))
			for i, ref := range found {
				parts[i] = fmt.Sprintf("línea %d %q", ref.Line, ref.FullPath())
			}
			return strings.Join(parts, "; ")
		}
		switch {
		case entry.Action == ActionRemoved && len(found) > 0:
			v.violate(ViolationAnchor, entry.ID, entry.Document, "el ancla %q sigue existiendo en %s (%s) y la acción `remove` exige que ya no exista. Continuación: elimina esa sección", entry.Anchor, entry.Document, describe())
		case entry.Action == ActionRemoved:
		case !exists:
			v.violate(ViolationAnchor, entry.ID, entry.Document, "el documento %s no existe y la entrada %s (acción %q) exige la sección %q. Continuación: crea el documento con esa sección", entry.Document, entry.ID, entry.Action, entry.Anchor)
		case len(found) == 0:
			v.violate(ViolationAnchor, entry.ID, entry.Document, "el ancla %q no resuelve a ninguna sección de %s tras la edición. Continuación: crea o conserva esa sección", entry.Anchor, entry.Document)
		case len(found) > 1:
			v.violate(ViolationAnchor, entry.ID, entry.Document, "el ancla %q resuelve a %d secciones de %s (%s) y debe resolver exactamente una. Continuación: deja una sola o desambigua el ancla del delta de contexto", entry.Anchor, len(found), entry.Document, describe())
		}
	}
}

// scopeSets son los conjuntos que deciden el alcance: se construyen una vez.
type scopeSets struct {
	context  []string // documentos nombrados en las entradas del plan
	defaults []string // specs de los dominios del cambio e INDEX.md
}

func (v *verifier) scopeSets() scopeSets {
	var sets scopeSets
	for _, entry := range v.trace.Entries {
		sets.context = append(sets.context, v.scope.Resolve(entry.Document))
	}
	specs := filepath.Join(v.t.ChangeRoot, "specs")
	if entries, err := os.ReadDir(specs); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(specs, entry.Name(), "spec.md")); err == nil {
				sets.defaults = append(sets.defaults, filepath.Join(v.t.Location.PlanningHome, "specs", entry.Name(), "spec.md"))
			}
		}
	}
	sets.defaults = append(sets.defaults, filepath.Join(v.t.Location.PlanningHome, "INDEX.md"))
	return sets
}

func anyUnder(declared []string, target string) bool {
	for _, d := range declared {
		if underPath(d, target) {
			return true
		}
	}
	return false
}

// checkScope comprueba el alcance: cada ruta del diff de archive pertenece al
// conjunto permitido, y la pertenencia se decide por identidad (os.SameFile para
// ficheros, pathidentity.Contains para directorios), nunca comparando cadenas. Un
// fichero borrado fuera de la carpeta del cambio es siempre una violación, porque
// archive nunca borra documentos. Hay errores explícitos para las rutas de
// referencia y las rutas mantenidas con «Sin impacto» (REQ-24.57).
func (v *verifier) checkScope() {
	sets := v.scopeSets()
	verify := v.verifyCommand()
	for _, c := range v.changed {
		file := v.display(c.abs)
		switch {
		case c.inChange:
		case c.deleted:
			v.violate(ViolationScope, "", file, "se ha borrado %s fuera de la carpeta del cambio: archive nunca borra documentos. Continuación: restáuralo con `git checkout -- %s` y vuelve a ejecutar `%s`", file, pathquote.ShellWord(c.rel), verify)
		case anyUnder(sets.context, c.abs) || anyUnder(sets.defaults, c.abs):
		default:
			v.violate(ViolationScope, "", file, "%s", v.outOfScope(c, file, verify))
		}
	}
}

// outOfScope explica por qué una ruta queda fuera del conjunto permitido, con el
// error más específico que corresponde: referencia, «Sin impacto» o el genérico.
func (v *verifier) outOfScope(c *changedFile, file, verify string) string {
	resolution := fmt.Sprintf("revierte o retira ese cambio y vuelve a ejecutar `%s`", verify)
	switch v.scope.Classify(c.abs) {
	case Reference:
		return fmt.Sprintf("%s está bajo una ruta de referencia (modo reference): las rutas de referencia nunca se modifican (REQ-24.30). Continuación: %s", file, resolution)
	case Maintained:
		if v.trace.Context.Decision == ContextNoImpact {
			return fmt.Sprintf("%s está bajo una ruta mantenida y el delta de contexto declara «Sin impacto» (%s): la edición contradice la decisión registrada (REQ-24.57). Continuación: %s, o cambia la decisión en %s y vuelve a preparar con `%s`", file, v.trace.Context.Reason, resolution, DeltaFileName, v.prepare)
		}
	}
	return fmt.Sprintf("%s no pertenece al conjunto permitido de la integración (los documentos nombrados en el delta de contexto, la carpeta del cambio y las specs de los dominios del cambio e INDEX.md). Continuación: %s", file, resolution)
}

// markVerified devuelve la traza con las entradas integradas, con `created` en las
// entradas `add` que crearon su documento (el documento no existía en el commit
// registrado) y con verified_at.
func (v *verifier) markVerified() (Trace, error) {
	updated := v.trace
	updated.Entries = append([]TraceEntry(nil), v.trace.Entries...)
	for i := range updated.Entries {
		entry := &updated.Entries[i]
		entry.State = StateIntegrated
		if entry.Action == ActionAdded {
			created, err := v.createdDocument(v.scope.Resolve(entry.Document))
			if err != nil {
				return Trace{}, err
			}
			if created {
				entry.Action = ActionCreated
			}
		}
	}
	verifiedAt := now().UTC().Truncate(time.Second)
	updated.VerifiedAt = &verifiedAt
	return updated, nil
}

// createdDocument dice si archive creó el documento: aparece en el diff y no
// existía en el commit que registró `prepare`.
func (v *verifier) createdDocument(abs string) (bool, error) {
	for _, c := range v.changed {
		if c.deleted || c.isDir || !underPath(abs, c.abs) {
			continue
		}
		if c.repo.head == "" {
			return true, nil
		}
		existed, err := c.repo.repo.committedExists(v.ctx, c.repo.head, c.rel)
		if err != nil {
			return false, err
		}
		return !existed, nil
	}
	return false, nil
}
