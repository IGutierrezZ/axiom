package changestart

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/fileprim"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// Estados de éxito de la fase 2 y del descarte (D-18). already_adopted es el mismo de la fase 1.
const (
	StatusAdopted          = "adopted"
	StatusDiscarded        = "discarded"
	StatusNothingToDiscard = "nothing_to_discard"
)

// Resultado del vínculo de integrations.yaml (paso 3) y del enlace de vuelta (paso 7) en la salida.
const (
	LinkCreated       = "created"
	LinkPresent       = "present"
	LinkNotApplicable = "not_applicable"

	BacklinkAdded    = "added"
	BacklinkPresent  = "present"
	BacklinkPending  = "pending"
	BacklinkConflict = "conflict"
)

// Valores de `backlink` en context/work-item.json.
const (
	storedBacklinkPending  = "pending"
	storedBacklinkDone     = "done"
	storedBacklinkConflict = "conflict"
)

// AdoptResult es el resultado de la fase 2. Solo lleva datos de confianza (D-20).
type AdoptResult struct {
	Status      string
	Change      string
	Lane        Lane
	Profile     Profile
	ExternalRef string
	Plugin, ID  string
	TypeKey     string
	URL         string
	Context     string // <destino>/context ya adoptado
	Link        string
	Backlink    string
	Downloaded  int
	Omitted     int
	Warnings    []string
}

// DiscardResult es el resultado del descarte.
type DiscardResult struct {
	Status   string
	Change   string
	Staging  string
	Warnings []string
}

// errTagBlocksBackLink lo devuelve el callback Allow cuando otra etiqueta axiom:* impide añadir la propia.
var errTagBlocksBackLink = errors.New("otra etiqueta axiom:* de un cambio abierto impide el enlace de vuelta")

// changeRoot es la carpeta SDD del cambio bajo el planning home.
func (e Env) changeRoot(change string) string {
	return filepath.Join(e.PlanningHome, "changes", change)
}

// contextPaths devuelve el destino de context/ del carril y el directorio de sus temporales de copia,
// ambos en el mismo árbol de trabajo y volumen (D-15): SDD <planning>/changes/<cambio>/context/ y
// .context-tmp/; ODD <ws>/odd/context/<cambio>/ y <ws>/odd/context/.tmp/<cambio>/.
func (e Env) contextPaths(lane Lane, change string) (dest, tmpBase string) {
	if lane == LaneODD {
		base := filepath.Join(e.WorkspaceRoot, "odd", "context")
		return filepath.Join(base, change), filepath.Join(base, ".tmp", change)
	}
	root := e.changeRoot(change)
	return filepath.Join(root, "context"), filepath.Join(root, ".context-tmp")
}

// adoptCommand es el comando de reintento de la fase 2 (con --lane odd en ODD).
func adoptCommand(change string, lane Lane) string {
	cmd := "axiom change start --adopt --change " + change
	if lane == LaneODD {
		cmd += " --lane odd"
	}
	return cmd
}

// Adopt ejecuta la fase 2 (D-12, D-14, D-15, D-25): bajo el cerrojo del cambio, comprueba que el cambio
// está preparado, copia y verifica context/ desde el staging (que queda intacto), vincula el work item
// (SDD), asegura la exclusión de git, registra el carril y el enlace pendiente, retira el staging y
// añade la etiqueta de vuelta. Es reintentable e idempotente. No toma el cerrojo del work item, no
// ejecuta git y no escribe en el repositorio fuera de los destinos de D-15.
func Adopt(ctx context.Context, env Env, tr Tracker, change string, lane Lane) (AdoptResult, error) {
	env = env.withDefaults()
	if err := ValidChangeName(change); err != nil {
		return AdoptResult{}, refuse(CodeUsageInvalid, err)
	}
	if lane != LaneSDD && lane != LaneODD {
		return AdoptResult{}, refuse(CodeUsageInvalid, errors.New("carril no válido"))
	}
	unlock, rf := env.lockChange(change)
	if rf != nil {
		return AdoptResult{}, rf
	}
	defer unlock()

	// Paso 1: preparación y qué adoptar.
	warnings, rf := env.prepare(change, lane)
	if rf != nil {
		return AdoptResult{}, rf
	}
	dest, tmpBase := env.contextPaths(lane, change)
	destJSON := filepath.Join(dest, "work-item.json")
	destDoc, destReadable := readContextDocOK(destJSON)
	_, statErr := os.Lstat(dest)
	if statErr != nil && !isNotExist(statErr) {
		return AdoptResult{}, refuse(CodeWriteFailed, statErr, "step", "destination")
	}
	destExists := statErr == nil
	staging, hasStaging := env.stagingFor(change)

	var doc ContextDoc
	copyNeeded := false
	switch {
	case hasStaging:
		doc = staging.Doc
		if destExists {
			if !destReadable || destDoc.Plugin != doc.Plugin || destDoc.ID != doc.ID {
				return AdoptResult{}, refuse(CodeStagingConflict, nil, "change", change)
			}
		} else {
			copyNeeded = true
		}
	case destReadable:
		doc = destDoc
		if doc.Backlink == storedBacklinkDone {
			return env.alreadyAdopted(doc, change, lane, dest, warnings), nil
		}
	default:
		return AdoptResult{}, refuse(CodeStagingNotFound, nil, "change", change)
	}
	if doc.Plugin != tr.PluginID() {
		return AdoptResult{}, refuse(CodeStagingConflict, nil, "change", change)
	}
	warnings = append(warnings, env.sweepTemps(change)...)
	if err := env.FS.RemoveAll(tmpBase); err != nil {
		warnings = append(warnings, "no se pudieron barrer los temporales de una adopción anterior del cambio; se barrerán en la siguiente operación")
	}

	// Paso 2: copiar y verificar context/ en el destino; el staging no se toca.
	createdHere := false
	if copyNeeded {
		if step, err := env.copyContext(filepath.Join(staging.Dir, "context"), dest, tmpBase); err != nil {
			return AdoptResult{}, refuse(CodeWriteFailed, err, "step", step)
		}
		createdHere = true
	}
	undo := func() {
		if createdHere {
			env.undoContext(dest, tmpBase)
		}
	}

	res := AdoptResult{Status: StatusAdopted, Change: change, Lane: lane, Link: LinkNotApplicable, Context: dest}

	// Paso 3: vínculo kind work_item en integrations.yaml (solo SDD).
	if lane == LaneSDD {
		folder := integrations.Folder{WorkspaceRoot: env.WorkspaceRoot, Root: env.changeRoot(change)}
		changed, err := env.FS.LinkWorkItem(folder, doc.Plugin, integrations.Link{Kind: "work_item", ID: doc.ID, URL: doc.URL}, env.Now())
		if err != nil {
			undo()
			if errors.Is(err, fileprim.ErrLockContended) {
				return AdoptResult{}, refuse(CodeChangeStartBusy, err)
			}
			return AdoptResult{}, refuse(CodeWriteFailed, err, "step", "link")
		}
		res.Link = LinkPresent
		if changed {
			res.Link = LinkCreated
		}
	}

	// Paso 4: exclusión de context/ en info/exclude.
	if err := env.FS.EnsureIgnored(env.WorkspaceRoot); err != nil {
		undo()
		return AdoptResult{}, refuse(CodeWriteFailed, err, "step", "exclude")
	}

	// Paso 5: carril y enlace pendiente en el work-item.json del destino.
	cur, err := readContextDoc(destJSON)
	if err != nil {
		return AdoptResult{}, refuse(CodeWriteFailed, err, "step", "workitem_json")
	}
	cur.Lane = string(lane)
	if cur.AdoptedAt == "" {
		cur.AdoptedAt = env.Now().UTC().Format("2006-01-02T15:04:05Z")
	}
	if cur.Backlink != storedBacklinkDone {
		cur.Backlink = storedBacklinkPending
	}
	if err := env.writeContextDoc(destJSON, cur); err != nil {
		return AdoptResult{}, refuse(CodeWriteFailed, err, "step", "workitem_json")
	}
	res.fill(cur)

	// Paso 6: retirar el staging. Un fallo avisa y sigue: la adopción ya está completa.
	if hasStaging {
		if _, err := env.retireReporting(staging.Dir, change); err != nil {
			warnings = append(warnings, "staging_not_retired: no se pudo retirar el staging del cambio "+change+" ("+filepath.ToSlash(staging.Dir)+"); la adopción está completa y `"+adoptCommand(change, lane)+"` lo retira al repetirlo")
		}
	}

	// Pasos 7 y 8: enlace de vuelta y su resultado. Con el enlace ya hecho no hay petición.
	if cur.Backlink == storedBacklinkDone {
		res.Backlink = BacklinkPresent
		res.Warnings = warnings
		return res, nil
	}
	outcome, warning := env.linkBack(ctx, tr, cur.ID, change, lane)
	if warning != "" {
		warnings = append(warnings, warning)
	}
	res.Backlink = outcome.reported
	if outcome.stored != cur.Backlink {
		cur.Backlink = outcome.stored
		if err := env.writeContextDoc(destJSON, cur); err != nil {
			warnings = append(warnings, "no se pudo registrar el resultado del enlace de vuelta en context/work-item.json: queda pendiente y `"+adoptCommand(change, lane)+"` lo completa")
		}
	}
	res.Warnings = warnings
	return res, nil
}

// readContextDocOK es readContextDoc sin el error.
func readContextDocOK(path string) (ContextDoc, bool) {
	doc, err := readContextDoc(path)
	return doc, err == nil
}

func (r *AdoptResult) fill(doc ContextDoc) {
	r.Profile = Profile(doc.Profile)
	r.ExternalRef = doc.ExternalRef
	r.Plugin, r.ID = doc.Plugin, doc.ID
	r.TypeKey = doc.TypeKey
	r.URL = doc.URL
	r.Downloaded, r.Omitted = doc.Attachments.Downloaded, doc.Attachments.Omitted
}

// alreadyAdopted compone el resultado de una adopción completa: sin red y sin escrituras (D-13).
func (e Env) alreadyAdopted(doc ContextDoc, change string, lane Lane, dest string, warnings []string) AdoptResult {
	res := AdoptResult{Status: StatusAlreadyAdopted, Change: change, Lane: lane, Context: dest, Warnings: warnings}
	res.fill(doc)
	return res
}

// writeContextDoc escribe work-item.json con la costura de escritura atómica.
func (e Env) writeContextDoc(path string, doc ContextDoc) error {
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return e.FS.WriteFile(path, append(payload, '\n'), contextFileMode)
}

// copyContext copia src a un temporal del mismo árbol que dest, lo verifica y lo publica con un único
// renombrado (D-15). Cualquier fallo borra el temporal y deja el destino sin cambios; step nombra el
// paso que falló (copy, verify, rename).
func (e Env) copyContext(src, dest, tmpBase string) (step string, err error) {
	tmp := filepath.Join(tmpBase, randomName())
	defer func() {
		if err != nil {
			_ = e.FS.RemoveAll(tmp)
		}
		removeEmptyDirs(tmpBase)
	}()
	if err = e.FS.CopyTree(src, tmp); err != nil {
		return "copy", err
	}
	if err = e.FS.VerifyTree(src, tmp); err != nil {
		return "verify", err
	}
	if err = e.FS.MkdirAll(filepath.Dir(dest), contextDirMode); err != nil {
		return "rename", err
	}
	if err = e.FS.Rename(tmp, dest); err != nil {
		return "rename", err
	}
	return "", nil
}

// undoContext retira el context/ que esta ejecución acaba de crear (D-25): lo renombra a un temporal
// del mismo árbol y lo borra, para que git no lo vea nunca a medio borrar.
func (e Env) undoContext(dest, tmpBase string) {
	parking := filepath.Join(tmpBase, randomName())
	if e.FS.MkdirAll(tmpBase, contextDirMode) == nil && e.FS.Rename(dest, parking) == nil {
		_ = e.FS.RemoveAll(parking)
	} else {
		_ = e.FS.RemoveAll(dest)
	}
	removeEmptyDirs(tmpBase)
}

// removeEmptyDirs retira, si quedaron vacíos, el directorio de temporales y su padre `.tmp` de ODD.
func removeEmptyDirs(tmpBase string) {
	_ = os.Remove(tmpBase)
	if parent := filepath.Dir(tmpBase); filepath.Base(parent) == ".tmp" {
		_ = os.Remove(parent)
	}
}

// backLinkOutcome es el resultado del paso 7: lo que se informa y lo que se guarda.
type backLinkOutcome struct {
	reported string // added | present | pending | conflict
	stored   string // pending | done | conflict
}

// linkBack añade la etiqueta axiom:<cambio> con la regla de D-11 para las etiquetas ajenas. Un fallo del
// tracker no es un fallo del verbo: devuelve pending con un aviso accionable (prov:74:15).
func (e Env) linkBack(ctx context.Context, tr Tracker, id, change string, lane Lane) (backLinkOutcome, string) {
	local := newLocalState(e)
	var blocking []string
	allow := func(tags []string) error {
		blocking = local.blockingTags(tags, change)
		if len(blocking) > 0 {
			return errTagBlocksBackLink
		}
		return nil
	}
	back, err := tr.LinkBack(ctx, id, axiomTagPrefix+change, allow)
	if err != nil {
		return backLinkOutcome{BacklinkPending, storedBacklinkPending},
			"no se pudo añadir la etiqueta " + axiomTagPrefix + change + " al work item (el tracker no respondió o rechazó la escritura): el contexto y el vínculo ya están aplicados; repite `" + adoptCommand(change, lane) + "` para completar el enlace"
	}
	switch back.Outcome {
	case integrations.BackLinkAdded:
		return backLinkOutcome{BacklinkAdded, storedBacklinkDone}, ""
	case integrations.BackLinkPresent:
		return backLinkOutcome{BacklinkPresent, storedBacklinkDone}, ""
	case integrations.BackLinkConflict:
		names := strings.Join(blocking, ", ")
		if names == "" {
			names = "de otro cambio"
		}
		return backLinkOutcome{BacklinkConflict, storedBacklinkConflict},
			"el work item ya lleva la etiqueta " + names + " de otro cambio abierto: no se añadió " + axiomTagPrefix + change + "; retira esa etiqueta en el tracker o cierra ese cambio y repite `" + adoptCommand(change, lane) + "`"
	}
	return backLinkOutcome{BacklinkPending, storedBacklinkPending},
		"el tracker devolvió un resultado desconocido al añadir la etiqueta: repite `" + adoptCommand(change, lane) + "` para completar el enlace"
}

// prepare comprueba que el cambio está preparado (D-14), sin escribir y sin ejecutar git: registro del
// carril, carpeta del cambio (SDD) y HEAD leído de ficheros. Un HEAD ilegible (reftable) se acepta con
// aviso. Los rechazos llevan en Fields el motivo y, si los hay, el carril del otro registro y el
// worktree, para que la CLI componga la continuación.
func (e Env) prepare(change string, lane Lane) ([]string, *Refusal) {
	store := registry.New()
	rec, err := store.Load(e.CommonDir, branch.Lane(lane), change)
	if err != nil {
		return nil, refuse(CodeChangeNotPrepared, err, "reason", "record_unreadable", "lane", string(lane), "change", change, "detail", err.Error())
	}
	if rec == nil {
		otherLane := LaneODD
		if lane == LaneODD {
			otherLane = LaneSDD
		}
		if other, oerr := store.Load(e.CommonDir, branch.Lane(otherLane), change); oerr == nil && other != nil {
			return nil, refuse(CodeChangeNotPrepared, nil, "reason", "lane_mismatch", "lane", string(lane), "change", change,
				"record_lane", string(otherLane), "worktree", recordWorktree(*other))
		}
		return nil, refuse(CodeChangeNotPrepared, nil, "reason", "no_record", "lane", string(lane), "change", change)
	}
	repo, ok := planningRepo(*rec)
	if !ok {
		return nil, refuse(CodeChangeNotPrepared, nil, "reason", "record_unreadable", "lane", string(lane), "change", change,
			"detail", "el registro no identifica el repositorio de especificaciones")
	}
	worktree := recordWorktree(*rec)
	var warnings []string
	if repo.Role != branch.RepoSpecsInPlace {
		head, detached, err := gitcommon.ReadHead(e.GitRoot)
		switch {
		case err != nil:
			warnings = append(warnings, "no se pudo leer el HEAD del checkout sin ejecutar git: se comprobó solo el registro y la carpeta del cambio")
		case detached || !branchRegistered(repo, head):
			return nil, refuse(CodeChangeNotPrepared, nil, "reason", "head_other_branch", "lane", string(lane), "change", change, "worktree", worktree)
		}
	}
	if lane == LaneSDD && !dirExists(e.changeRoot(change)) {
		return nil, refuse(CodeChangeNotPrepared, nil, "reason", "folder_missing", "lane", string(lane), "change", change, "worktree", worktree)
	}
	return warnings, nil
}

// branchRegistered dice si head es la rama del cambio o de uno de sus roles en el repositorio.
func branchRegistered(repo branch.RecordRepo, head string) bool {
	for _, b := range repo.Branches {
		if (b.Kind == branch.KindChange || b.Kind == branch.KindRole) && b.Name == head {
			return true
		}
	}
	return false
}

// recordWorktree devuelve el worktree de la rama del cambio del repositorio de especificaciones, o vacío
// si no lo hay o ya no existe.
func recordWorktree(r branch.Record) string {
	repo, ok := planningRepo(r)
	if !ok {
		return ""
	}
	for _, b := range repo.Branches {
		if b.Kind == branch.KindChange && b.Worktree != "" && dirExists(b.Worktree) {
			return b.Worktree
		}
	}
	return ""
}
