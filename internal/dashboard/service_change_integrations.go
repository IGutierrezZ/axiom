package dashboard

import (
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// Pestaña «Integraciones» del cambio (D-07). Es de solo lectura: no ejecuta ni reintenta la outbox
// ni modifica vínculos. Cada fuente se lee por separado y su fallo deja vacía solo su sección.

const (
	// maxChangeOutbox acota las operaciones de la outbox que se publican.
	maxChangeOutbox = 100
	// maxChangeGateRecords acota las decisiones de puerta que se publican (las más recientes).
	maxChangeGateRecords = 100
)

// safeExternalURL dice si raw es un enlace externo seguro (D-09): sin blancos ni caracteres de
// control, analizable, esquema http o https (sin distinguir mayúsculas), con host y sin userinfo.
// Rechaza javascript:, data:, relativas y //host. Equivale a `safeExternalURL` de core/integrations.js.
func safeExternalURL(raw string) bool {
	if raw == "" {
		return false
	}
	for _, r := range raw {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	return parsed.Hostname() != "" && parsed.User == nil
}

// changeFolderFor devuelve la carpeta de integraciones de un cambio ya localizado. WorkspaceRoot es
// el padre del directorio `openspec` que la contiene (en un worktree, el del propio worktree).
func changeFolderFor(changeRoot string) (integrations.Folder, bool) {
	for dir := changeRoot; ; {
		parent := filepath.Dir(dir)
		if filepath.Base(dir) == "openspec" {
			return integrations.Folder{WorkspaceRoot: parent, Root: changeRoot}, true
		}
		if parent == dir {
			return integrations.Folder{}, false
		}
		dir = parent
	}
}

// GetChangeIntegrations arma la pestaña del cambio. Un nombre inválido o un cambio inexistente son
// fallos tipados; cualquier lectura que falle es un error parcial en la propia respuesta.
func (s *Service) GetChangeIntegrations(name string) (*ChangeIntegrationsDTO, *integrationsError) {
	if !validExplainerChangeName(name) {
		return nil, integrationsFail(400, "invalid_change_name", "el nombre del cambio no es válido")
	}
	location, err := s.locateIncrement(name)
	if err != nil {
		return nil, integrationsFail(404, "change_not_found", "no se encontró el cambio indicado")
	}
	if info, statErr := os.Stat(location.path); statErr != nil || !info.IsDir() {
		return nil, integrationsFail(404, "change_not_found", "no se encontró el cambio indicado")
	}
	dto := &ChangeIntegrationsDTO{
		Name:      name,
		Archived:  location.kind == "archived",
		WorkItems: []ChangeLinkDTO{},
		Roles:     []ChangeRoleDTO{},
		Outbox:    []ChangeOperationDTO{},
		Gates:     []ChangeGateActorDTO{},
		Errors:    []string{},
	}
	if folder, ok := changeFolderFor(location.path); ok {
		fillChangeLinks(dto, folder)
		fillChangeOutbox(dto, folder)
	} else {
		dto.Errors = append(dto.Errors, "la carpeta del cambio no cuelga de un directorio openspec: no se pueden leer sus vínculos ni su outbox")
	}
	fillChangeActors(dto, location.path)
	dto.Empty = len(dto.WorkItems) == 0 && len(dto.Roles) == 0 && len(dto.Outbox) == 0
	return dto, nil
}

func changeLinkDTO(l integrations.Link) ChangeLinkDTO {
	out := ChangeLinkDTO{Plugin: l.Plugin, PluginTitle: l.Plugin, Kind: l.Kind, ID: l.ID, EstimateHours: l.EstimateHours}
	if m, found := manifest.Lookup(l.Plugin); found && strings.TrimSpace(m.Title) != "" {
		out.PluginTitle = m.Title
	}
	if safeExternalURL(l.URL) {
		out.URL = l.URL
	}
	if l.StartedAt != nil {
		out.StartedAt = l.StartedAt.UTC().Format(time.RFC3339)
	}
	return out
}

// fillChangeLinks lee integrations.yaml y el diario de tiempo pendiente. Cada rol conocido por el
// kickoff, una Task, una imputación o un tiempo pendiente aparece una vez; sin Task, Task es nil.
func fillChangeLinks(dto *ChangeIntegrationsDTO, folder integrations.Folder) {
	const unreadable = "integrations.yaml no se pudo leer (corrupto o de una versión desconocida): no se muestran vínculos ni imputaciones"
	roles := map[string]*ChangeRoleDTO{}
	role := func(name string) *ChangeRoleDTO {
		if r, found := roles[name]; found {
			return r
		}
		r := &ChangeRoleDTO{Role: name, Entries: []ChangeTimeEntryDTO{}}
		roles[name] = r
		return r
	}
	if loaded, err := kickoff.Load(folder.Root); err == nil && loaded != nil {
		for _, kr := range loaded.Config.Roles {
			role(kr.Role)
		}
	}
	workItems, err := integrations.WorkItemLinks(folder)
	if err != nil {
		dto.Errors = append(dto.Errors, unreadable)
		return
	}
	for _, l := range workItems {
		dto.WorkItems = append(dto.WorkItems, changeLinkDTO(l))
	}
	tasks, err := integrations.TaskLinks(folder)
	if err != nil {
		dto.Errors = append(dto.Errors, unreadable)
		return
	}
	for _, l := range tasks {
		link := changeLinkDTO(l)
		role(l.Role).Task = &link
	}
	entries, err := integrations.TimeEntries(folder)
	if err != nil {
		dto.Errors = append(dto.Errors, unreadable)
		return
	}
	for _, e := range entries {
		r := role(e.Role)
		r.Entries = append(r.Entries, ChangeTimeEntryDTO{Plugin: e.Plugin, ItemID: e.ItemID, Minutes: e.Minutes, Date: e.Date, Actor: e.Actor})
	}
	names := make([]string, 0, len(roles))
	for name := range roles {
		names = append(names, name)
	}
	sort.Strings(names)
	pendingFailed := false
	for _, name := range names {
		pending, found, perr := integrations.PendingTimeFor(folder, name)
		if perr != nil {
			pendingFailed = true
			break
		}
		if found {
			roles[name].Pending = &ChangePendingTimeDTO{Minutes: pending.Minutes, Date: pending.Date, ItemID: pending.ItemID}
		}
	}
	if pendingFailed {
		dto.Errors = append(dto.Errors, "el tiempo pendiente de imputar no se pudo leer (fichero corrupto o de una versión desconocida)")
	}
	for _, name := range names {
		r := roles[name]
		// El grupo sin rol solo aparece si tiene contenido; un rol del kickoff sin Task se muestra «sin Task».
		if name == "" && r.Task == nil && len(r.Entries) == 0 && r.Pending == nil {
			continue
		}
		dto.Roles = append(dto.Roles, *r)
	}
}

func fillChangeOutbox(dto *ChangeIntegrationsDTO, folder integrations.Folder) {
	views, err := integrations.OutboxView(folder)
	if err != nil {
		dto.Errors = append(dto.Errors, "la outbox de integraciones no se pudo leer (fichero corrupto o de una versión desconocida)")
		return
	}
	dto.OutboxTotal = len(views)
	if len(views) > maxChangeOutbox {
		views = views[:maxChangeOutbox]
		dto.OutboxTruncated = true
	}
	for _, v := range views {
		op := ChangeOperationDTO{
			Plugin: v.Plugin, Event: string(v.Event), Role: v.Role, Gate: v.Gate,
			State: string(v.State), Attempts: v.Attempts, Reason: v.Reason,
			EnqueuedAt: v.EnqueuedAt.UTC().Format(time.RFC3339),
		}
		if !v.LastAttemptAt.IsZero() {
			op.LastAttemptAt = v.LastAttemptAt.UTC().Format(time.RFC3339)
		}
		dto.Outbox = append(dto.Outbox, op)
	}
}

// fillChangeActors lee quién selló el kickoff y quién decidió cada puerta. Un cambio sin
// kickoff.yaml (creado desde el dashboard) no es un error: Sealed queda en false.
func fillChangeActors(dto *ChangeIntegrationsDTO, changeRoot string) {
	loaded, err := kickoff.Load(changeRoot)
	switch {
	case err != nil:
		dto.Errors = append(dto.Errors, "kickoff.yaml no se pudo leer (corrupto o de una versión desconocida): se desconoce quién lo selló")
	case loaded != nil:
		dto.Kickoff = ChangeKickoffActorDTO{Sealed: true, SealedBy: loaded.SealedBy}
	}
	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		dto.Errors = append(dto.Errors, "gates.yaml no se pudo leer (corrupto o de una versión desconocida): no se muestran las decisiones de puerta")
		return
	}
	records := ledger.Records
	if len(records) > maxChangeGateRecords {
		records = records[len(records)-maxChangeGateRecords:]
	}
	for _, r := range records {
		g := ChangeGateActorDTO{Gate: string(r.Gate), Decision: string(r.Decision), Actor: r.Actor}
		if !r.RecordedAt.IsZero() {
			g.RecordedAt = r.RecordedAt.UTC().Format(time.RFC3339)
		}
		dto.Gates = append(dto.Gates, g)
	}
}
