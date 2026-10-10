package specbaseline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// VerifySchema es el esquema del informe de `integrate verify` (§4.4).
const VerifySchema = "axiom.sdd-integration-verify/v1"

// Comprobaciones de la verificación nativa E2 (REQ-24.51, D-15) que nombra
// Violation.Check.
const (
	// ViolationPresence: una entrada esperada no figura en el plan, o su documento
	// no aparece en el diff de archive (a).
	ViolationPresence = "presence"
	// ViolationAnchor: un ancla no existe tras la edición exactamente una vez, o
	// existe una acción `remove` (b).
	ViolationAnchor = "anchor"
	// ViolationScope: el diff de un repositorio afectado toca algo fuera del
	// conjunto permitido (c).
	ViolationScope = "scope"
	// ViolationTrace: el plan de integración no existe, no es analizable o no
	// declara el esquema esperado (d).
	ViolationTrace = "trace"
	// ViolationIDs: el texto añadido conserva un ID provisional distinto de su ID
	// definitivo (e).
	ViolationIDs = "ids"
	// ViolationCompose: un documento de composición no está en su huella after.
	ViolationCompose = "compose"
)

// Violation es un fallo de la verificación: la comprobación, la entrada y el
// fichero que la provocan y su explicación, con la resolución cuando la hay.
type Violation struct {
	Check  string `json:"check"` // presence | anchor | scope | trace | ids | compose
	Entry  string `json:"entry,omitempty"`
	File   string `json:"file,omitempty"`
	Detail string `json:"detail"`
}

// VerifyReport es la salida de `integrate verify` (axiom.sdd-integration-verify/v1).
// Passed es verdadero solo sin violaciones; Violations nunca es nil.
type VerifyReport struct {
	Schema     string      `json:"schema"`
	Change     string      `json:"change"`
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations"`
}

// Verify es la verificación nativa E2 de la integración dirigida (REQ-24.51,
// REQ-24.53, REQ-24.57, REQ-24.59 (e), D-15): determinista, sin modelo y con git de
// solo lectura. Comprueba, contra el plan que fijó `prepare`:
//
//   - (d) traza: `integration.yaml` existe, se decodifica de forma estricta y es del
//     cambio;
//   - las composiciones, por la huella after del plan y no por ancla, porque son
//     deterministas;
//   - (a) presencia: las entradas esperadas, recalculadas desde los ficheros actuales
//     del cambio, coinciden con las del plan, y cada documento con entradas dirigidas
//     o de contexto (salvo `remove` y «Sin impacto») aparece en el diff de archive;
//   - (b) anclas: la sección del cambio de cada documento dirigido aparece
//     exactamente una vez, y cada ancla de contexto resuelve una sección (`add` y
//     `modify`) o ninguna (`remove`);
//   - (c) alcance: cada ruta del diff de archive de cada repositorio afectado
//     pertenece al conjunto permitido, por identidad y nunca por texto;
//   - (e) IDs: ningún documento cambiado tiene más tokens de un ID provisional
//     (distinto de su definitivo) que el commit que registró `prepare`.
//
// El diff de archive es el del árbol de trabajo contra el HEAD del plan (no el
// actual: un commit entre `prepare` y `verify` no oculta nada) más los ficheros sin
// seguimiento, sin las entradas de la foto previa cuya huella no ha cambiado.
//
// Si todo pasa marca como integradas las entradas dirigidas y de contexto (y las
// composiciones que quedaron pendientes tras una interrupción), registra `created`
// en las entradas `add` que crearon el documento y escribe `verified_at`: solo toca
// `integration.yaml`. Si algo falla lista todas las violaciones y no modifica ni los
// documentos ni el plan. Las violaciones van en el informe; el error devuelto es solo
// de ejecución (git no disponible, tiempo agotado, fallo de escritura).
func Verify(ctx context.Context, t ChangeTarget) (VerifyReport, error) {
	report := VerifyReport{Schema: VerifySchema, Change: t.Change, Violations: []Violation{}}
	planPath := filepath.Join(t.ChangeRoot, PlanFileName)
	prepareCommand := fmt.Sprintf("axiom sdd integrate prepare --cwd %s --change %s", pathquote.Quote(t.Location.WorkspaceRoot), pathquote.ShellWord(t.Change))

	plan, err := ReadPlan(planPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		report.add(ViolationTrace, "", PlanFileName, "el cambio no tiene plan de integración: la verificación compara contra lo que fijó `prepare`. Continuación: ejecuta `%s`", prepareCommand)
		return report, ctx.Err()
	case err != nil:
		report.add(ViolationTrace, "", PlanFileName, "el plan de integración no se puede leer: %v. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `%s`", err, PlanFileName, prepareCommand)
		return report, ctx.Err()
	case plan.Change != t.Change:
		report.add(ViolationTrace, "", PlanFileName, "el plan es del cambio %q y se verifica el cambio %q. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `%s`", plan.Change, t.Change, PlanFileName, prepareCommand)
		return report, ctx.Err()
	}

	v := &verifier{
		ctx:            ctx,
		t:              t,
		plan:           plan,
		master:         resolveDeepest(filepath.Clean(t.Location.Master)),
		scope:          contextScope(t.Location),
		ids:            newIDMap(plan.IDMap),
		report:         &report,
		prepareCommand: prepareCommand,
	}
	v.checkCompositions()
	v.checkExpectedEntries()
	if err := v.collectChanges(); err != nil {
		return report, err
	}
	v.checkPresence()
	v.checkAnchors()
	v.checkScope()
	if err := v.checkIDs(); err != nil {
		return report, err
	}
	if len(report.Violations) > 0 {
		return report, ctx.Err()
	}

	updated, err := v.markVerified()
	if err != nil {
		return report, err
	}
	if err := WritePlan(planPath, updated); err != nil {
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
	repo  gitRepository
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
	allowed  bool
}

type verifier struct {
	ctx            context.Context
	t              ChangeTarget
	plan           Plan
	master         string
	scope          ContextScope
	ids            IDMap
	report         *VerifyReport
	prepareCommand string

	changed []*changedFile
}

func (v *verifier) violate(check, entry, file, format string, args ...any) {
	v.report.add(check, entry, file, format, args...)
}

func (v *verifier) workspacePath(rel string) string {
	return filepath.Join(v.t.Location.WorkspaceRoot, filepath.FromSlash(rel))
}

func (v *verifier) display(abs string) string { return displayPath(v.master, abs) }

// checkCompositions comprueba por huella que cada documento de composición está
// compuesto y no se retocó después. Un documento en su huella after con el plan aún
// pendiente (una interrupción tras renombrar) cuenta como integrado, salvo que after
// sea igual a before: entonces la huella no distingue, y solo vale el estado del plan.
func (v *verifier) checkCompositions() {
	for _, document := range v.plan.Documents {
		if document.Integration != workspace.IntegrationCompose {
			continue
		}
		abs := v.workspacePath(document.Path)
		fingerprint, err := workingTreeFingerprint(abs)
		if err != nil {
			v.violate(ViolationCompose, document.Domain, document.Path, "no se puede leer el documento compuesto del dominio %q: %v", document.Domain, err)
			continue
		}
		matchesAfter := document.AfterSHA256 != "" && fingerprint == document.AfterSHA256
		switch {
		case document.State == StateIntegrated && matchesAfter:
		case document.State == StateIntegrated:
			v.violate(ViolationCompose, document.Domain, document.Path, "el documento del dominio %q figura como integrado y ya no coincide con la huella after que fijó `prepare`: se modificó después de componerse. Continuación: devuelve el documento con `git checkout -- %s` y vuelve a componerlo con `axiom sdd archive-compose` (el comando que imprime `axiom sdd integrate prepare`)", document.Domain, pathquote.ShellWord(document.Path))
		case matchesAfter && document.AfterSHA256 != document.BeforeSHA256:
			// Interrupción tras renombrar: el documento ya está compuesto y el plan sigue
			// pendiente. markVerified lo marca como integrado.
		default:
			v.violate(ViolationCompose, document.Domain, document.Path, "el documento del dominio %q no está compuesto: no coincide con la huella after que fijó `prepare`. Continuación: ejecuta `%s`", document.Domain, composeCommand(v.t, v.plan.Documents))
		}
	}
}

// entryKey identifica una entrada de la traza sin su estado. La acción `created`,
// que verify registra, equivale a `added`, que es la del delta.
func entryKey(entry TraceEntry) string {
	action := entry.Action
	if action == ActionCreated {
		action = ActionAdded
	}
	return strings.Join([]string{string(entry.Kind), entry.Domain, entry.Document, entry.Anchor, string(action), entry.ID, entry.Reason}, "\x1f")
}

func describeEntry(entry TraceEntry) string {
	return fmt.Sprintf("%s (%s, documento %s)", entry.ID, entry.Kind, entry.Document)
}

func contextTraceAction(action ContextAction) TraceAction {
	switch action {
	case ContextAdd:
		return ActionAdded
	case ContextRemove:
		return ActionRemoved
	default:
		return ActionModified
	}
}

// checkExpectedEntries recalcula las entradas esperadas desde los ficheros actuales
// del cambio (los deltas, el delta de contexto y la declaración) y comprueba que
// coinciden con las del plan (a). Las entradas de composición no se recalculan: el
// documento ya se compuso y su resultado se comprueba por huella after.
func (v *verifier) checkExpectedEntries() {
	scratch := newReport(CheckSchema, v.t.Change)
	g, ok := gather(v.t, &scratch)
	if !ok || len(scratch.Errors) > 0 {
		for _, problem := range scratch.Errors {
			v.violate(ViolationPresence, "", "", "no se pueden recalcular las entradas esperadas desde los ficheros actuales del cambio: %s", problem)
		}
		return
	}
	digest := inputsDigest(g.deltaInputs(), g.contextData, declarationSnapshot(v.t.Location.Declarations))
	if digest != v.plan.InputsSHA256 {
		v.violate(ViolationPresence, "", PlanFileName, "los deltas, el delta de contexto o la declaración de axiom.yaml cambiaron desde `prepare`: las entradas esperadas ya no corresponden al plan. Continuación: devuelve los documentos integrados con `git checkout -- <documentos>`, retira %s de la carpeta del cambio y vuelve a preparar con `%s`", PlanFileName, v.prepareCommand)
	}

	expected := map[string]TraceEntry{}
	var order []string
	remember := func(entry TraceEntry) {
		key := entryKey(entry)
		if _, seen := expected[key]; !seen {
			order = append(order, key)
		}
		expected[key] = entry
	}
	for _, document := range g.documents {
		if document.mode == workspace.IntegrationCompose {
			continue
		}
		for _, entry := range directedEntries(document, parseDelta(document.delta), v.ids, document.mode, v.t.Change) {
			remember(entry)
		}
	}
	if g.maintained && g.contextPresent {
		delta, err := ParseContextDelta(g.contextData)
		if err != nil {
			v.violate(ViolationPresence, "", ContextDeltaFileName, "%s ya no es válido: %v. Continuación: corrige el delta de contexto y vuelve a preparar con `%s`", ContextDeltaFileName, err, v.prepareCommand)
		} else if !delta.NoImpact {
			for _, entry := range delta.Entries {
				remember(TraceEntry{ID: entry.ID, Kind: EntryContext, Document: entry.Document, Anchor: entry.Anchor.String(), Action: contextTraceAction(entry.Action), Reason: v.ids.Substitute(entry.Reason)})
			}
		}
	}

	recorded := map[string]TraceEntry{}
	var recordedOrder []string
	for _, entry := range v.plan.Entries {
		if entry.Kind == EntryCompose {
			continue
		}
		key := entryKey(entry)
		if _, seen := recorded[key]; !seen {
			recordedOrder = append(recordedOrder, key)
		}
		recorded[key] = entry
	}
	for _, key := range order {
		if _, ok := recorded[key]; !ok {
			entry := expected[key]
			v.violate(ViolationPresence, entry.ID, entry.Document, "la entrada %s figura en los ficheros del cambio y no en el plan de integración. Continuación: vuelve a preparar con `%s`", describeEntry(entry), v.prepareCommand)
		}
	}
	for _, key := range recordedOrder {
		if _, ok := expected[key]; !ok {
			entry := recorded[key]
			v.violate(ViolationPresence, entry.ID, entry.Document, "la entrada %s del plan ya no está en los ficheros del cambio. Continuación: vuelve a preparar con `%s`", describeEntry(entry), v.prepareCommand)
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
// archive: el árbol de trabajo contra el HEAD registrado más los ficheros sin
// seguimiento, sin las entradas de la foto previa cuya huella no ha cambiado
// (D-15). Un repositorio sin commits se compara con el árbol vacío.
func (v *verifier) collectChanges() error {
	for _, recorded := range v.plan.Repositories {
		dir := filepath.Join(v.master, filepath.FromSlash(recorded.Repo))
		repo, err := openRepository(v.ctx, dir)
		var notGit *NotGitRepositoryError
		switch {
		case errors.As(err, &notGit):
			v.violate(ViolationScope, "", recorded.Repo, "el repositorio %s que registró `prepare` ya no es un repositorio git: el alcance de la integración solo se puede comprobar por diff. Continuación: restablece el repositorio y vuelve a ejecutar `axiom sdd integrate verify`, o vuelve a preparar con `%s`", recorded.Repo, v.prepareCommand)
			continue
		case err != nil:
			return err
		}
		if !pathidentity.SameDirectory(repo.root, dir) {
			v.violate(ViolationScope, "", recorded.Repo, "el directorio %s que registró `prepare` ya no es la raíz de un repositorio git (ahora pertenece al repositorio de %s). Continuación: restablece el repositorio y vuelve a ejecutar `axiom sdd integrate verify`, o vuelve a preparar con `%s`", recorded.Repo, pathquote.Quote(repo.root), v.prepareCommand)
			continue
		}
		if recorded.Head != "" {
			exists, err := repo.commitExists(v.ctx, recorded.Head)
			if err != nil {
				return err
			}
			if !exists {
				v.violate(ViolationScope, "", recorded.Repo, "el commit %s que registró `prepare` en el repositorio %s ya no existe (¿se reescribió la historia?): sin él no se puede calcular el diff de archive. Continuación: vuelve a preparar con `%s`", recorded.Head, recorded.Repo, v.prepareCommand)
				continue
			}
		}

		verified := &verifiedRepository{name: recorded.Repo, repo: repo, head: recorded.Head, photo: map[string]string{}}
		for _, entry := range recorded.Dirty {
			verified.photo[pathKey(entry.Path)] = entry.SHA256
		}

		type candidate struct {
			rel     string
			deleted bool
		}
		var candidates []candidate
		seen := map[string]bool{}
		add := func(rel string, deleted bool) {
			if key := pathKey(rel); !seen[key] {
				seen[key] = true
				candidates = append(candidates, candidate{rel: rel, deleted: deleted})
			}
		}
		if recorded.Head == "" {
			files, err := repo.listFiles(v.ctx, "--cached", "--others", "--exclude-standard")
			if err != nil {
				return err
			}
			for _, rel := range files {
				add(rel, false)
			}
		} else {
			changes, err := repo.diffNames(v.ctx, recorded.Head)
			if err != nil {
				return err
			}
			for _, change := range changes {
				add(change.Path, strings.HasPrefix(change.Status, "D"))
			}
			untracked, err := repo.listFiles(v.ctx, "--others", "--exclude-standard")
			if err != nil {
				return err
			}
			for _, rel := range untracked {
				add(rel, false)
			}
		}

		for _, c := range candidates {
			rel := strings.TrimSuffix(c.rel, "/")
			abs := filepath.Join(repo.root, filepath.FromSlash(rel))
			if recorded.Head == "" {
				// Contra el árbol vacío, un fichero que está en el índice y ya no en el árbol
				// de trabajo no es un cambio.
				if _, err := os.Lstat(abs); err != nil {
					continue
				}
			}
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
				deleted:  c.deleted,
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

// checkPresence exige que cada documento con entradas dirigidas o de contexto
// aparezca en el diff de archive (a). Las entradas `remove` de contexto, el modo
// solo registro y «Sin impacto» no lo exigen.
func (v *verifier) checkPresence() {
	for _, entry := range v.plan.Entries {
		switch {
		case entry.Kind == EntryDirected:
			if abs := v.workspacePath(entry.Document); !v.appears(abs) {
				v.violate(ViolationPresence, entry.ID, entry.Document, "la edición dirigida de la entrada %q no aparece en el diff de archive: el documento %s no se modificó. Continuación: integra la sección del cambio en el documento y vuelve a ejecutar `axiom sdd integrate verify`", entry.ID, entry.Document)
			}
		case entry.Kind == EntryContext && entry.Action != ActionRemoved:
			if abs := v.scope.Resolve(entry.Document); !v.appears(abs) {
				v.violate(ViolationPresence, entry.ID, entry.Document, "la entrada de contexto %s no aparece en el diff de archive: el documento %s no se modificó (o no se creó). Continuación: edita el documento según la entrada y vuelve a ejecutar `axiom sdd integrate verify`", entry.ID, entry.Document)
			}
		}
	}
}

// startsWithToken dice si el título empieza por token como token completo: lo que
// sigue no es un carácter de identificador.
func startsWithToken(title, token string) bool {
	if !strings.HasPrefix(title, token) {
		return false
	}
	rest := title[len(token):]
	if rest == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return !isIDRune(r)
}

// checkAnchors comprueba las anclas (b): la sección del cambio de cada documento
// dirigido existe exactamente una vez, y cada ancla de contexto resuelve
// exactamente una sección con `add` y `modify` y ninguna con `remove`.
func (v *verifier) checkAnchors() {
	read := func(abs string) ([]byte, bool, error) {
		data, err := os.ReadFile(abs)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return nil, false, nil
		case err != nil:
			return nil, false, err
		}
		return data, true, nil
	}
	for _, entry := range v.plan.Entries {
		switch entry.Kind {
		case EntryDirected:
			data, exists, err := read(v.workspacePath(entry.Document))
			if err != nil || !exists {
				v.violate(ViolationAnchor, entry.ID, entry.Document, "no se puede comprobar la sección %q de la entrada %q: el documento %s no se puede leer", v.plan.Change, entry.ID, entry.Document)
				continue
			}
			var sections []string
			for _, heading := range ParseDocument(data).Headings() {
				if startsWithToken(heading.Title, v.plan.Change) {
					sections = append(sections, fmt.Sprintf("línea %d %q", heading.Index+1, heading.Title))
				}
			}
			switch len(sections) {
			case 1:
			case 0:
				v.violate(ViolationAnchor, entry.ID, entry.Document, "el documento %s no contiene ninguna sección de la entrada %q cuyo título empiece por el identificador del cambio %q como token completo (REQ-24.50). Continuación: crea o reutiliza esa sección en el documento", entry.Document, entry.ID, v.plan.Change)
			default:
				v.violate(ViolationAnchor, entry.ID, entry.Document, "el documento %s contiene %d secciones de la entrada %q cuyo título empieza por el identificador del cambio %q (%s) y debe haber exactamente una. Continuación: reutiliza una sola sección", entry.Document, len(sections), entry.ID, v.plan.Change, strings.Join(sections, "; "))
			}
		case EntryContext:
			anchor, err := ParseAnchor(entry.Anchor)
			if err != nil {
				v.violate(ViolationAnchor, entry.ID, entry.Document, "el ancla %q del plan no es válida: %v", entry.Anchor, err)
				continue
			}
			data, exists, err := read(v.scope.Resolve(entry.Document))
			if err != nil {
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
}

// scopeSets son los conjuntos que deciden el alcance (c): se construyen una vez.
type scopeSets struct {
	documents []string // documentos de los dominios del cambio con integración compose o directed
	register  []namedPath
	context   []string // documentos nombrados en las entradas del delta de contexto
	defaults  []string // con la baseline por defecto: specs de los dominios del cambio e INDEX.md
	hasIndex  bool
}

type namedPath struct {
	abs    string
	domain string
}

func (v *verifier) scopeSets() scopeSets {
	var sets scopeSets
	for _, document := range v.plan.Documents {
		abs := v.workspacePath(document.Path)
		if document.Integration == workspace.IntegrationRegister {
			sets.register = append(sets.register, namedPath{abs: abs, domain: document.Domain})
		} else {
			sets.documents = append(sets.documents, abs)
		}
	}
	for _, entry := range v.plan.Entries {
		if entry.Kind == EntryContext {
			sets.context = append(sets.context, v.scope.Resolve(entry.Document))
		}
	}
	if v.t.Location.Declarations.Baseline == nil {
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
		sets.hasIndex = true
	}
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

// checkScope comprueba el alcance (c): cada ruta del diff de archive pertenece al
// conjunto permitido, y la pertenencia se decide por identidad (os.SameFile para
// ficheros, pathidentity.Contains para directorios), nunca comparando cadenas. Un
// fichero borrado fuera de la carpeta del cambio es siempre una violación, porque
// archive nunca borra documentos. Hay errores explícitos para las rutas de
// referencia, los documentos de solo registro y las rutas mantenidas con «Sin
// impacto» (REQ-24.57).
func (v *verifier) checkScope() {
	sets := v.scopeSets()
	verify := fmt.Sprintf("axiom sdd integrate verify --cwd %s --change %s", pathquote.Quote(v.t.Location.WorkspaceRoot), pathquote.ShellWord(v.t.Change))
	for _, c := range v.changed {
		file := v.display(c.abs)
		switch {
		case c.inChange:
			c.allowed = true
		case c.deleted:
			v.violate(ViolationScope, "", file, "se ha borrado %s fuera de la carpeta del cambio: archive nunca borra documentos. Continuación: restáuralo con `git checkout -- %s` y vuelve a ejecutar `%s`", file, pathquote.ShellWord(c.rel), verify)
		case anyUnder(sets.documents, c.abs) || anyUnder(sets.context, c.abs) || anyUnder(sets.defaults, c.abs):
			c.allowed = true
		default:
			v.violate(ViolationScope, "", file, "%s", v.outOfScope(c, sets, file, verify))
		}
	}
}

// outOfScope explica por qué una ruta queda fuera del conjunto permitido, con el
// error más específico que corresponde: referencia, solo registro, «Sin impacto» o
// el genérico.
func (v *verifier) outOfScope(c *changedFile, sets scopeSets, file, verify string) string {
	resolution := fmt.Sprintf("revierte o retira ese cambio y vuelve a ejecutar `%s`", verify)
	for _, declared := range v.scope.Reference {
		if underPath(v.scope.Resolve(declared), c.abs) {
			return fmt.Sprintf("%s está bajo la ruta de referencia %q (modo reference): las rutas de referencia nunca se modifican (REQ-24.30). Continuación: %s", file, declared, resolution)
		}
	}
	for _, registered := range sets.register {
		if underPath(registered.abs, c.abs) {
			return fmt.Sprintf("%s es el documento del dominio %q con integración register: archive no lo edita, solo registra la entrada como pendiente (REQ-24.53). Continuación: %s", file, registered.domain, resolution)
		}
	}
	if v.plan.Context.Decision == ContextNoImpact {
		for _, declared := range v.scope.Maintained {
			if underPath(v.scope.Resolve(declared), c.abs) {
				return fmt.Sprintf("%s está bajo una ruta mantenida y el delta de contexto declara «Sin impacto» (%s): la edición contradice la decisión registrada (REQ-24.57). Continuación: %s, o cambia la decisión en %s y vuelve a preparar con `%s`", file, v.plan.Context.Reason, resolution, ContextDeltaFileName, v.prepareCommand)
			}
		}
	}
	allowed := "los documentos de los dominios del cambio con integración compose o directed, los documentos nombrados en el delta de contexto y la carpeta del cambio"
	if sets.hasIndex {
		allowed += ", y con la baseline por defecto las specs de los dominios del cambio e INDEX.md"
	}
	return fmt.Sprintf("%s no pertenece al conjunto permitido de la integración (%s). Continuación: %s", file, allowed, resolution)
}

// checkIDs comprueba la sustitución de IDs (e): por cada documento cambiado dentro
// de alcance y cada ID provisional con definitivo distinto, el número de tokens en el
// contenido actual no supera el del commit que registró `prepare` (`git show
// <head>:<ruta>`, vacío si el fichero no existía). Contar tokens en lugar de comparar
// líneas tolera el reajuste del texto preexistente y no depende del final de línea.
func (v *verifier) checkIDs() error {
	if len(v.plan.IDMap) == 0 {
		return nil
	}
	for _, c := range v.changed {
		if !c.allowed || c.inChange || c.deleted || c.isDir {
			continue
		}
		data, err := os.ReadFile(c.abs)
		if err != nil {
			v.violate(ViolationIDs, "", v.display(c.abs), "no se puede leer %s para comprobar los IDs provisionales: %v", v.display(c.abs), err)
			continue
		}
		current := v.ids.CountProvisional(string(data))
		if len(current) == 0 {
			continue
		}
		previous := map[string]int{}
		if c.repo.head != "" {
			committed, _, err := c.repo.repo.committedContent(v.ctx, c.repo.head, c.rel)
			if err != nil {
				return err
			}
			previous = v.ids.CountProvisional(string(committed))
		}
		provisionals := make([]string, 0, len(current))
		for provisional := range current {
			provisionals = append(provisionals, provisional)
		}
		sort.Strings(provisionals)
		for _, provisional := range provisionals {
			if current[provisional] <= previous[provisional] {
				continue
			}
			assignment, _ := v.ids.Lookup(provisional)
			v.violate(ViolationIDs, provisional, v.display(c.abs), "el ID provisional %s (ID definitivo %s) aparece %d vez/veces en %s y el commit registrado por `prepare` tenía %d: el texto añadido conserva un ID provisional sin sustituir (REQ-24.59). Continuación: sustituye %s por %s en el texto que añadiste",
				provisional, assignment.Definitive, current[provisional], v.display(c.abs), previous[provisional], provisional, assignment.Definitive)
		}
	}
	return nil
}

// markVerified devuelve el plan con las entradas dirigidas y de contexto, y las
// composiciones, integradas; con `created` en las entradas `add` que crearon su
// documento (el documento no existía en el commit registrado); y con verified_at.
// Las entradas de solo registro siguen pendientes (REQ-24.53).
func (v *verifier) markVerified() (Plan, error) {
	updated := v.plan
	updated.Documents = append([]PlanDocument(nil), v.plan.Documents...)
	for i := range updated.Documents {
		if updated.Documents[i].Integration == workspace.IntegrationCompose {
			updated.Documents[i].State = StateIntegrated
		}
	}
	updated.Entries = append([]TraceEntry(nil), v.plan.Entries...)
	for i := range updated.Entries {
		entry := &updated.Entries[i]
		switch entry.Kind {
		case EntryCompose, EntryDirected:
			entry.State = StateIntegrated
		case EntryContext:
			entry.State = StateIntegrated
			if entry.Action == ActionAdded {
				created, err := v.createdDocument(v.scope.Resolve(entry.Document))
				if err != nil {
					return Plan{}, err
				}
				if created {
					entry.Action = ActionCreated
				}
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
		_, existed, err := c.repo.repo.committedContent(v.ctx, c.repo.head, c.rel)
		if err != nil {
			return false, err
		}
		return !existed, nil
	}
	return false, nil
}

// commitExists dice si el commit existe en el repositorio.
func (r gitRepository) commitExists(ctx context.Context, commit string) (bool, error) {
	result, err := execGit(ctx, r.root, "cat-file", "-e", commit+"^{commit}")
	if err != nil {
		return false, err
	}
	return result.exit == 0, nil
}

// nameStatus es una entrada de `git diff --name-status`.
type nameStatus struct {
	Status string
	Path   string // relativa a la raíz del repositorio, con '/'
}

// diffNames devuelve los ficheros del árbol de trabajo que difieren del commit, con
// o sin stage: `git diff --name-status -z --no-renames <commit>`. Con
// `core.autocrlf` git ya normaliza los finales de línea, de modo que un fichero que
// solo difiere en ellos no aparece. Solo lee: `git diff` refresca el índice y lo
// reescribe si sus marcas de tiempo están desfasadas, de modo que se ejecuta con
// GIT_INDEX_FILE en una copia temporal del índice (`withScratchIndex`) y el índice
// real no se toca.
func (r gitRepository) diffNames(ctx context.Context, commit string) ([]nameStatus, error) {
	var result gitResult
	err := r.withScratchIndex(ctx, func(indexFile string) error {
		var diffErr error
		result, diffErr = execGitEnv(ctx, r.root, []string{"GIT_INDEX_FILE=" + indexFile}, "-c", "core.fsmonitor=false", "diff", "--no-ext-diff", "--no-textconv", "--no-renames", "--name-status", "-z", commit, "--")
		return diffErr
	})
	if err != nil {
		return nil, err
	}
	if result.exit != 0 {
		return nil, fmt.Errorf("git diff contra %s en %s terminó con el código %d: %s", commit, r.root, result.exit, result.stderr)
	}
	var entries []nameStatus
	fields := bytes.Split(result.stdout, []byte{0})
	for i := 0; i < len(fields); {
		if len(fields[i]) == 0 {
			i++
			continue
		}
		if i+1 >= len(fields) {
			return nil, fmt.Errorf("git diff en %s devolvió una entrada incompleta: %q", r.root, fields[i])
		}
		entries = append(entries, nameStatus{Status: string(fields[i]), Path: string(fields[i+1])})
		i += 2
	}
	return entries, nil
}

// listFiles devuelve los ficheros que `git ls-files -z` lista con las opciones
// indicadas, sin entrecomillado. Los repositorios anidados sin seguimiento llevan
// una `/` final.
func (r gitRepository) listFiles(ctx context.Context, options ...string) ([]string, error) {
	args := append([]string{"ls-files", "-z"}, options...)
	result, err := execGit(ctx, r.root, args...)
	if err != nil {
		return nil, err
	}
	if result.exit != 0 {
		return nil, fmt.Errorf("git ls-files en %s terminó con el código %d: %s", r.root, result.exit, result.stderr)
	}
	var files []string
	for _, field := range bytes.Split(result.stdout, []byte{0}) {
		if len(field) > 0 {
			files = append(files, string(field))
		}
	}
	return files, nil
}

// committedContent devuelve el contenido de un fichero en el commit, el mismo que
// da `git show <commit>:<ruta>`, y si existía; un fichero que no existía devuelve
// contenido vacío. La existencia se consulta con `git ls-tree`, que no depende del
// idioma de los mensajes de git.
func (r gitRepository) committedContent(ctx context.Context, commit, rel string) ([]byte, bool, error) {
	tree, err := execGit(ctx, r.root, "ls-tree", "-z", "--name-only", commit, "--", rel)
	if err != nil {
		return nil, false, err
	}
	if tree.exit != 0 {
		return nil, false, fmt.Errorf("git ls-tree %s en %s terminó con el código %d: %s", commit, r.root, tree.exit, tree.stderr)
	}
	if len(bytes.Trim(tree.stdout, "\x00")) == 0 {
		return nil, false, nil
	}
	blob, err := execGit(ctx, r.root, "cat-file", "blob", commit+":"+rel)
	if err != nil {
		return nil, false, err
	}
	if blob.exit != 0 {
		return nil, false, fmt.Errorf("git cat-file %s:%s en %s terminó con el código %d: %s", commit, rel, r.root, blob.exit, blob.stderr)
	}
	return blob.stdout, true, nil
}
