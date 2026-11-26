package changestart

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/changestart/htmlmd"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

// Estados de éxito de la fase 1 (D-18).
const (
	StatusStaged         = "staged"
	StatusAlreadyStaged  = "already_staged"
	StatusAlreadyAdopted = "already_adopted"
)

// StartResult es el resultado de la fase 1. Solo lleva datos de confianza (D-20): el nombre derivado,
// el perfil, la referencia, los identificadores, la URL que construyó el adaptador, la clave de tipo
// configurada, las rutas, los contadores y los avisos de texto fijo.
type StartResult struct {
	Status      string
	Change      string
	Profile     Profile
	ExternalRef string
	Plugin, ID  string
	TypeKey     string
	URL         string
	Staging     string // <common>/axiom/change-start/v1/<cambio>
	Context     string // context/ ya adoptado (already_adopted)
	Lane        Lane
	Downloaded  int
	Omitted     int
	Warnings    []string
}

// Start ejecuta la fase 1 (D-12, D-13): lee el work item una vez, lo valida en el orden de prov:74:6,
// deriva el nombre, comprueba los stagings existentes y publica un staging atómico bajo el common dir.
// No escribe en el árbol de trabajo ni en el work item. El error es un *Refusal, un error de uso del
// puerto (integrations.ErrPortUnsupported) o, nunca, un error de E/S sin tipar.
func Start(ctx context.Context, env Env, tr Tracker, ref string) (StartResult, error) {
	env = env.withDefaults()
	plugin := tr.PluginID()

	id, err := tr.ParseRef(ref)
	if err != nil {
		if errors.Is(err, integrations.ErrPortUnsupported) {
			return StartResult{}, err
		}
		return StartResult{}, refuse(CodeWorkItemReferenceInvalid, err)
	}

	unlockItem, rf := env.lockItem(plugin, id)
	if rf != nil {
		return StartResult{}, rf
	}
	defer unlockItem()

	item, err := tr.Read(ctx, id)
	if err != nil {
		if errors.Is(err, integrations.ErrItemNotFound) {
			return StartResult{}, refuse(CodeWorkItemNotFound, err)
		}
		return StartResult{}, trackerUnavailable(err, plugin)
	}
	tp, ok := mapType(tr.ItemKeys(), item.Type)
	if !ok {
		return StartResult{}, refuse(CodeWorkItemTypeUnmapped, nil, "type", inlineNeutral(item.Type), "plugin", plugin)
	}
	cat, err := tr.StateCategory(ctx, tp.Type, item.State)
	if err != nil {
		return StartResult{}, trackerUnavailable(err, plugin)
	}
	var warnings []string
	if !cat.Known {
		warnings = append(warnings, "no se pudo determinar si el work item está cerrado: su estado no figura entre los estados de su tipo")
	}
	if closedCategory(cat.Category) {
		return StartResult{}, refuse(CodeWorkItemClosed, nil, "category", cat.Category)
	}
	warnings = append(warnings, item.Warnings...)

	name := DeriveName(tp.Prefix, id, item.Title)
	profile := profileFor(item.Type)
	externalRef := tr.RefAlias() + ":" + id
	scan, rf := newLocalState(env).scanLinks(item, plugin, id, name)
	if rf != nil {
		return StartResult{}, rf
	}
	warnings = append(warnings, scan.Warnings...)

	unlockChange, rf := env.lockChange(name)
	if rf != nil {
		return StartResult{}, rf
	}
	defer unlockChange()
	warnings = append(warnings, env.sweepTemps(name)...)

	res := StartResult{
		Change: name, Profile: profile, ExternalRef: externalRef, Plugin: plugin, ID: id,
		TypeKey: tp.Type, URL: item.URL,
	}

	// D-13, punto 1: staging válido de ese nombre y, si no, del mismo work item con otro nombre.
	if s, ok := env.stagingFor(name); ok {
		if s.Doc.Plugin != plugin || s.Doc.ID != id {
			return StartResult{}, refuse(CodeStagingConflict, nil, "change", name)
		}
		return alreadyStaged(res, s, warnings), nil
	}
	if others := env.stagingsOf(plugin, id); len(others) > 0 {
		return alreadyStaged(res, others[0], warnings), nil
	}
	// D-13, punto 2: adopción propia.
	if scan.Self != nil {
		res.Status = StatusAlreadyAdopted
		res.Context = scan.Self.Context
		res.Lane = scan.Self.Lane
		if scan.Self.Backlink != "done" {
			cmd := "axiom change start --adopt --change " + name
			if scan.Self.Lane == LaneODD {
				cmd += " --lane odd"
			}
			warnings = append(warnings, "el enlace de vuelta con el work item no está completo: completa la adopción con `"+cmd+"`")
		}
		res.Warnings = warnings
		return res, nil
	}

	downloaded, omitted, truncated, err := env.publishStaging(ctx, tr, item, res, cat.Category)
	if err != nil {
		return StartResult{}, refuse(CodeWriteFailed, err, "step", "staging")
	}
	if truncated {
		warnings = append(warnings, "el contenido de algún campo del work item era demasiado grande y se truncó")
	}
	res.Status = StatusStaged
	res.Staging = env.StagingDir(name)
	res.Downloaded, res.Omitted = downloaded, omitted
	res.Warnings = warnings
	return res, nil
}

// alreadyStaged compone el resultado de un staging existente, que no se toca (D-13). El nombre
// devuelto es el del staging.
func alreadyStaged(res StartResult, s stagingEntry, warnings []string) StartResult {
	res.Status = StatusAlreadyStaged
	res.Change = s.Name
	res.Staging = s.Dir
	if s.Doc.Profile != "" {
		res.Profile = Profile(s.Doc.Profile)
	}
	if s.Doc.ExternalRef != "" {
		res.ExternalRef = s.Doc.ExternalRef
	}
	res.Downloaded, res.Omitted = s.Doc.Attachments.Downloaded, s.Doc.Attachments.Omitted
	res.Warnings = append(warnings,
		"ya existe un staging de este work item para el cambio "+s.Name+" en "+s.Dir+": no se ha modificado ni se ha descargado nada")
	return res
}

// trackerUnavailable clasifica cualquier fallo del tracker o de la credencial (D-03). Un secreto
// ausente se distingue para que la continuación nombre cómo guardarlo. Nunca copia el texto del fallo.
func trackerUnavailable(err error, plugin string) *Refusal {
	r := refuse(CodeTrackerUnavailable, err, "plugin", plugin)
	var missing *secrets.MissingError
	if errors.As(err, &missing) {
		r.Fields["secret"] = missing.Ref.Name
	}
	return r
}

// publishStaging importa el work item en <v1>/.tmp/<cambio>/<aleatorio>/, escribe work-item.json en
// último lugar y publica con un único os.Rename (D-12). Cualquier fallo borra el temporal: nunca queda
// un staging parcial adoptable.
func (e Env) publishStaging(ctx context.Context, tr Tracker, item integrations.ExternalItem, res StartResult, category string) (downloaded, omitted int, truncated bool, err error) {
	tmp := filepath.Join(e.tmpDir(res.Change), randomName())
	defer func() {
		_ = e.FS.RemoveAll(tmp)
		_ = os.Remove(e.tmpDir(res.Change)) // solo si quedó vacío
	}()

	pipeline := newAttachmentPipeline(tr.Download)
	pipeline.importAttached(ctx, item.Attachments)
	convert := func(src string) string {
		r := htmlmd.Convert(src, func(img htmlmd.Image) string { return pipeline.image(ctx, img) })
		truncated = truncated || r.Truncated
		return r.Markdown
	}
	description := convert(item.Description)
	repro := convert(item.ReproSteps)
	acceptance := convert(item.AcceptanceCriteria)

	now := e.Now()
	entries := pipeline.entries()
	markdown := renderWorkItem(renderInput{
		Plugin: res.Plugin, ID: res.ID, URL: res.URL, ExternalRef: res.ExternalRef, Profile: string(res.Profile),
		Imported: now, Item: item, Category: category, Description: description, ReproSteps: repro, AcceptanceCriteria: acceptance,
		Attachments: entries,
	})
	for _, entry := range entries {
		if entry.File == "" {
			omitted++
		}
	}
	downloaded = len(pipeline.files())

	doc := ContextDoc{
		Schema: ContextSchema, Change: res.Change, Plugin: res.Plugin, ID: res.ID, URL: res.URL,
		ExternalRef: res.ExternalRef, Profile: string(res.Profile), TypeKey: res.TypeKey,
		StagedAt: now.UTC().Format("2006-01-02T15:04:05Z"), Attachments: Attachments{Downloaded: downloaded, Omitted: omitted},
	}
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return 0, 0, false, err
	}

	ctxDir := filepath.Join(tmp, "context")
	if err = e.FS.MkdirAll(ctxDir, contextDirMode); err != nil {
		return 0, 0, false, err
	}
	if err = e.FS.WriteFile(filepath.Join(ctxDir, "work-item.md"), []byte(markdown), contextFileMode); err != nil {
		return 0, 0, false, err
	}
	if err = writeAttachmentFiles(e.FS, filepath.Join(ctxDir, "attachments"), pipeline.files()); err != nil {
		return 0, 0, false, err
	}
	// work-item.json va en último lugar: con él legible, el staging está completo.
	if err = e.FS.WriteFile(filepath.Join(ctxDir, "work-item.json"), append(payload, '\n'), contextFileMode); err != nil {
		return 0, 0, false, err
	}

	target := e.StagingDir(res.Change)
	if _, statErr := os.Lstat(target); statErr == nil {
		// Un directorio sin work-item.json legible no lo dejó una fase 1 completa: se sustituye.
		if err = e.retire(target, res.Change); err != nil {
			return 0, 0, false, err
		}
	}
	if err = e.FS.MkdirAll(filepath.Dir(target), contextDirMode); err != nil {
		return 0, 0, false, err
	}
	if err = e.FS.Rename(tmp, target); err != nil {
		return 0, 0, false, err
	}
	return downloaded, omitted, truncated, nil
}
