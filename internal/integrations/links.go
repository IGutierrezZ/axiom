package integrations

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

const (
	// LinksFile es el documento versionado con los vínculos e imputaciones de un cambio.
	LinksFile = "integrations.yaml"
	// LinksSchemaV1 es el esquema original: se sigue leyendo, y se reescribe como v2 solo cuando una
	// fusión cambia algo.
	LinksSchemaV1 = "axiom.integrations/v1"
	// LinksSchemaV2 añade a los vínculos `estimate_hours` y `started_at` y a las imputaciones `actor`.
	LinksSchemaV2 = "axiom.integrations/v2"

	maxLinkIDLength = 128

	// Orígenes (source) de los vínculos de Task y prefijo de las imputaciones de cierre de rol.
	TasksCreateSource = "tasks.create"
	TasksBindSource   = "tasks.bind"
	roleCloseSource   = "role.close/"
	// TaskLinkKind es el kind de los vínculos de la Task de un rol.
	TaskLinkKind = "task"
)

var (
	linkKindPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	dateOnly        = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// LinksDoc es el contenido de integrations.yaml. Nunca lleva intentos, errores ni operaciones.
type LinksDoc struct {
	Schema      string      `yaml:"schema"`
	Change      string      `yaml:"change"`
	Links       []Link      `yaml:"links,omitempty"`
	TimeEntries []TimeEntry `yaml:"time_entries,omitempty"`
}

// readLinks lee integrations.yaml con lectura estricta (esquema exacto, claves desconocidas
// rechazadas). Sin fichero devuelve un documento vacío. Funciona también en una carpeta archivada,
// donde solo se lee.
func readLinks(f Folder) (LinksDoc, error) {
	path := f.path(LinksFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LinksDoc{Schema: LinksSchemaV1, Change: f.Change()}, nil
		}
		return LinksDoc{}, fmt.Errorf("leer %s en %q: %w", LinksFile, path, err)
	}
	doc, err := parseLinks(data, f.Change())
	if err != nil {
		return LinksDoc{}, fmt.Errorf("%s inválido en %q: %w", LinksFile, path, err)
	}
	return doc, nil
}

func parseLinks(data []byte, change string) (LinksDoc, error) {
	// Primero solo el esquema, sin lectura estricta: un esquema ajeno se rechaza nombrándolo antes de
	// que sus campos desconocidos den un error confuso.
	var head struct {
		Schema string `yaml:"schema"`
	}
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&head); err != nil {
		if errors.Is(err, io.EOF) {
			return LinksDoc{}, errors.New("el fichero está vacío")
		}
		return LinksDoc{}, err
	}
	if head.Schema != LinksSchemaV1 && head.Schema != LinksSchemaV2 {
		return LinksDoc{}, fmt.Errorf("esquema %q no admitido; admitidos: %s, %s", head.Schema, LinksSchemaV1, LinksSchemaV2)
	}
	var doc LinksDoc
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&doc); err != nil {
		return LinksDoc{}, err
	}
	if doc.Change != change {
		return LinksDoc{}, fmt.Errorf("el campo change es %q y la carpeta es la del cambio %q", doc.Change, change)
	}
	for i, l := range doc.Links {
		if err := validateLink(l); err != nil {
			return LinksDoc{}, fmt.Errorf("vínculo %d: %w", i+1, err)
		}
		if strings.TrimSpace(l.Plugin) == "" || strings.TrimSpace(l.Source) == "" {
			return LinksDoc{}, fmt.Errorf("vínculo %d: plugin y source son obligatorios", i+1)
		}
	}
	for i, e := range doc.TimeEntries {
		if err := validateTimeEntry(e); err != nil {
			return LinksDoc{}, fmt.Errorf("imputación %d: %w", i+1, err)
		}
		if strings.TrimSpace(e.Plugin) == "" || strings.TrimSpace(e.Source) == "" {
			return LinksDoc{}, fmt.Errorf("imputación %d: plugin y source son obligatorios", i+1)
		}
	}
	return doc, nil
}

// validateLink comprueba los campos que controla el plugin (y se escriben en un documento
// versionado): kind, id y url. Una URL con userinfo se rechaza para que ninguna credencial llegue a
// disco; el error nunca repite la URL.
func validateLink(l Link) error {
	if !linkKindPattern.MatchString(l.Kind) {
		return fmt.Errorf("kind %q debe cumplir %s", l.Kind, linkKindPattern.String())
	}
	if err := validateItemID("id", l.ID); err != nil {
		return err
	}
	if l.EstimateHours != nil {
		if err := validateEstimateHours(*l.EstimateHours); err != nil {
			return err
		}
	}
	if l.StartedAt != nil {
		if _, offset := l.StartedAt.Zone(); offset != 0 {
			return errors.New("started_at debe estar en UTC")
		}
	}
	if l.URL != "" {
		parsed, err := url.Parse(l.URL)
		switch {
		case err != nil:
			return errors.New("url no es una URL válida")
		case parsed.Scheme != "http" && parsed.Scheme != "https":
			return errors.New("url debe ser http o https")
		case parsed.User != nil:
			return errors.New("url lleva userinfo (usuario o contraseña): se rechaza para que ninguna credencial llegue al documento versionado")
		case parsed.Host == "":
			return errors.New("url debe llevar host")
		}
	}
	return nil
}

// validateEstimateHours exige un valor positivo con, como mucho, dos decimales.
func validateEstimateHours(h float64) error {
	if math.IsNaN(h) || math.IsInf(h, 0) || h <= 0 {
		return errors.New("estimate_hours debe ser positivo")
	}
	if math.Abs(h*100-math.Round(h*100)) > 1e-6 {
		return errors.New("estimate_hours admite como mucho dos decimales")
	}
	return nil
}

func validateTimeEntry(e TimeEntry) error {
	if err := validateItemID("item_id", e.ItemID); err != nil {
		return err
	}
	if strings.HasPrefix(e.Source, roleCloseSource) && (strings.TrimSpace(e.Actor) == "" || strings.TrimSpace(e.Role) == "") {
		return fmt.Errorf("una imputación %s* exige actor y role no vacíos", roleCloseSource)
	}
	if e.Minutes <= 0 {
		return errors.New("minutes debe ser positivo")
	}
	if !dateOnly.MatchString(e.Date) {
		return fmt.Errorf("date %q debe tener la forma AAAA-MM-DD", e.Date)
	}
	if _, err := time.Parse("2006-01-02", e.Date); err != nil {
		return fmt.Errorf("date %q no es una fecha válida", e.Date)
	}
	return nil
}

func validateItemID(field, value string) error {
	switch {
	case strings.TrimSpace(value) == "":
		return fmt.Errorf("%s no puede estar vacío", field)
	case len(value) > maxLinkIDLength:
		return fmt.Errorf("%s supera los %d caracteres", field, maxLinkIDLength)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s no puede llevar caracteres de control", field)
		}
	}
	return nil
}

type linkIdentity struct{ plugin, kind, id, role string }
type entryIdentity struct{ plugin, source, itemID string }

// mergeLinks fusiona el resultado de una entrega en integrations.yaml de forma idempotente: un
// vínculo se identifica por (plugin, kind, id, role) y una imputación por (plugin, source, item_id);
// lo ya presente se conserva intacto. Un Outcome vacío no crea el fichero. Devuelve si el fichero
// cambió. Quien llama sostiene el cerrojo del cambio.
//
// Los campos que controla el plugin se validan y cualquier incumplimiento es un error permanente.
// En una carpeta archivada cualquier intento de escritura se rechaza con el mensaje de RF-ARC-03.
func mergeLinks(f Folder, pluginID, key string, now time.Time, out Outcome) (bool, error) {
	if len(out.Links) == 0 && len(out.TimeEntries) == 0 {
		return false, nil
	}
	if err := f.guard(LinksFile); err != nil {
		return false, err
	}

	links, entries, err := normalizeOutcome(pluginID, key, now, out)
	if err != nil {
		return false, Permanent(err)
	}
	doc, err := readLinks(f)
	if err != nil {
		return false, err
	}

	haveLinks := map[linkIdentity]int{}
	for i, l := range doc.Links {
		haveLinks[linkIdentity{l.Plugin, l.Kind, l.ID, l.Role}] = i
	}
	haveEntries := map[entryIdentity]bool{}
	for _, e := range doc.TimeEntries {
		haveEntries[entryIdentity{e.Plugin, e.Source, e.ItemID}] = true
	}
	changed := false
	for _, l := range links {
		id := linkIdentity{l.Plugin, l.Kind, l.ID, l.Role}
		at, present := haveLinks[id]
		if !present {
			haveLinks[id] = len(doc.Links)
			doc.Links = append(doc.Links, l)
			changed = true
			continue
		}
		// Un vínculo ya presente se conserva, salvo dos campos: la estimación entrante no nula y
		// distinta lo actualiza, y el inicio del rol solo se fija si aún no constaba.
		existing := &doc.Links[at]
		if l.EstimateHours != nil && (existing.EstimateHours == nil || *existing.EstimateHours != *l.EstimateHours) {
			hours := *l.EstimateHours
			existing.EstimateHours = &hours
			changed = true
		}
		if l.StartedAt != nil && existing.StartedAt == nil {
			started := *l.StartedAt
			existing.StartedAt = &started
			changed = true
		}
	}
	for _, e := range entries {
		id := entryIdentity{e.Plugin, e.Source, e.ItemID}
		if !haveEntries[id] {
			haveEntries[id] = true
			doc.TimeEntries = append(doc.TimeEntries, e)
			changed = true
		}
	}
	if !changed {
		return false, nil
	}
	if err := writeLinksDoc(f, doc); err != nil {
		return false, err
	}
	return true, nil
}

// writeLinksDoc escribe doc como v2: el esquema nuevo solo se materializa cuando algo cambia.
func writeLinksDoc(f Folder, doc LinksDoc) error {
	doc.Schema = LinksSchemaV2
	payload, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("serializar %s: %w", LinksFile, err)
	}
	return f.writeAtomic(LinksFile, linksTempPattern, payload)
}

// normalizeOutcome completa y valida lo que devuelve el plugin: el plugin, el origen (la clave del
// evento) y la fecha de registro.
func normalizeOutcome(pluginID, key string, now time.Time, out Outcome) ([]Link, []TimeEntry, error) {
	recorded := now.UTC()
	links := make([]Link, 0, len(out.Links))
	for i, l := range out.Links {
		if l.Plugin == "" {
			l.Plugin = pluginID
		}
		if l.Plugin != pluginID {
			return nil, nil, fmt.Errorf("vínculo %d: pertenece al plugin %q y lo devolvió %q", i+1, l.Plugin, pluginID)
		}
		if l.Source == "" {
			l.Source = key
		}
		if l.RecordedAt.IsZero() {
			l.RecordedAt = recorded
		}
		l.RecordedAt = l.RecordedAt.UTC()
		if l.StartedAt != nil {
			started := l.StartedAt.UTC()
			l.StartedAt = &started
		}
		if err := validateLink(l); err != nil {
			return nil, nil, fmt.Errorf("vínculo %d: %w", i+1, err)
		}
		links = append(links, l)
	}
	entries := make([]TimeEntry, 0, len(out.TimeEntries))
	for i, e := range out.TimeEntries {
		if e.Plugin == "" {
			e.Plugin = pluginID
		}
		if e.Plugin != pluginID {
			return nil, nil, fmt.Errorf("imputación %d: pertenece al plugin %q y lo devolvió %q", i+1, e.Plugin, pluginID)
		}
		if e.Source == "" {
			e.Source = key
		}
		if e.RecordedAt.IsZero() {
			e.RecordedAt = recorded
		}
		e.RecordedAt = e.RecordedAt.UTC()
		if err := validateTimeEntry(e); err != nil {
			return nil, nil, fmt.Errorf("imputación %d: %w", i+1, err)
		}
		entries = append(entries, e)
	}
	return links, entries, nil
}

// WorkItemLinkSource es el origen (source) del vínculo que escribe la adopción de un work item.
const WorkItemLinkSource = "change.start"

// LinkWorkItem escribe en integrations.yaml el vínculo de un work item con el cambio (C-07-d). Toma
// el cerrojo del cambio y reutiliza la fusión idempotente de las entregas con la clave change.start:
// los vínculos e imputaciones ya presentes se conservan y repetir la llamada no cambia el fichero.
// Devuelve si el fichero cambió.
func LinkWorkItem(f Folder, pluginID string, l Link, now time.Time) (bool, error) {
	release, err := f.acquire()
	if err != nil {
		return false, err
	}
	defer release()
	return mergeLinks(f, pluginID, WorkItemLinkSource, now, Outcome{Links: []Link{l}})
}

// WorkItemLinks devuelve los vínculos kind work_item de integrations.yaml. Sin fichero, ninguno.
func WorkItemLinks(f Folder) ([]Link, error) {
	doc, err := readLinks(f)
	if err != nil {
		return nil, err
	}
	var out []Link
	for _, l := range doc.Links {
		if l.Kind == "work_item" {
			out = append(out, l)
		}
	}
	return out, nil
}

// BindTask escribe en integrations.yaml el vínculo de la Task de un rol. l.Source debe ser
// tasks.create o tasks.bind. Toma el cerrojo del cambio y reutiliza la fusión idempotente: repetir la
// llamada no cambia el fichero, y una estimación entrante no nula y distinta actualiza
// estimate_hours del vínculo ya presente. Devuelve si el fichero cambió.
func BindTask(f Folder, pluginID string, l Link, now time.Time) (bool, error) {
	if l.Kind != TaskLinkKind {
		return false, fmt.Errorf("BindTask exige un vínculo kind %q y recibió %q", TaskLinkKind, l.Kind)
	}
	if l.Source != TasksCreateSource && l.Source != TasksBindSource {
		return false, fmt.Errorf("BindTask exige source %q o %q y recibió %q", TasksCreateSource, TasksBindSource, l.Source)
	}
	release, err := f.acquire()
	if err != nil {
		return false, err
	}
	defer release()
	return mergeLinks(f, pluginID, l.Source, now, Outcome{Links: []Link{l}})
}

// ReplaceTaskLink sustituye el vínculo de la Task del rol por l. Es la única función que retira un
// vínculo de integrations.yaml. Bajo el cerrojo quita el vínculo kind task del rol (del plugin) y
// añade l; copia el started_at anterior si l no trae uno y no toca las imputaciones. Si el vínculo
// vigente ya apunta a la misma Task, equivale a BindTask. Devuelve si el fichero cambió.
func ReplaceTaskLink(f Folder, pluginID, role string, l Link, now time.Time) (bool, error) {
	if l.Kind != TaskLinkKind || l.Role != role {
		return false, fmt.Errorf("ReplaceTaskLink exige un vínculo kind %q del rol %q", TaskLinkKind, role)
	}
	if l.Source == "" {
		l.Source = TasksBindSource
	}
	release, err := f.acquire()
	if err != nil {
		return false, err
	}
	defer release()
	if err := f.guard(LinksFile); err != nil {
		return false, err
	}
	doc, err := readLinks(f)
	if err != nil {
		return false, err
	}

	var previous *Link
	kept := make([]Link, 0, len(doc.Links))
	for i := range doc.Links {
		existing := doc.Links[i]
		if existing.Plugin == pluginID && existing.Kind == TaskLinkKind && existing.Role == role {
			if existing.ID == l.ID {
				// Misma Task: no hay nada que retirar, solo actualizar estimación e inicio.
				return mergeLinks(f, pluginID, l.Source, now, Outcome{Links: []Link{l}})
			}
			previous = &doc.Links[i]
			continue
		}
		kept = append(kept, existing)
	}
	normalized, _, err := normalizeOutcome(pluginID, l.Source, now, Outcome{Links: []Link{l}})
	if err != nil {
		return false, Permanent(err)
	}
	replacement := normalized[0]
	if replacement.StartedAt == nil && previous != nil && previous.StartedAt != nil {
		started := *previous.StartedAt
		replacement.StartedAt = &started
	}
	doc.Links = append(kept, replacement)
	if err := writeLinksDoc(f, doc); err != nil {
		return false, err
	}
	return true, nil
}

// MarkRoleStarted fija started_at en el vínculo de la Task del rol solo si aún no constaba (el
// inicio es de escritura única). Devuelve si cambió. Sin vínculo de Task del rol es un error: la
// barrera de ensure --role lo impide antes.
func MarkRoleStarted(f Folder, pluginID, role string, at time.Time) (bool, error) {
	release, err := f.acquire()
	if err != nil {
		return false, err
	}
	defer release()
	if err := f.guard(LinksFile); err != nil {
		return false, err
	}
	doc, err := readLinks(f)
	if err != nil {
		return false, err
	}
	for i := range doc.Links {
		l := &doc.Links[i]
		if l.Plugin != pluginID || l.Kind != TaskLinkKind || l.Role != role {
			continue
		}
		if l.StartedAt != nil {
			return false, nil
		}
		started := at.UTC()
		l.StartedAt = &started
		if err := writeLinksDoc(f, doc); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, fmt.Errorf("no hay vínculo de Task del rol %q del plugin %q en %s", role, pluginID, LinksFile)
}

// RecordTimeEntry añade una imputación a integrations.yaml, idempotente por (plugin, source,
// item_id). e.Source es obligatorio (role.close/<rol>/<n>). Devuelve si el fichero cambió.
func RecordTimeEntry(f Folder, pluginID string, e TimeEntry, now time.Time) (bool, error) {
	if strings.TrimSpace(e.Source) == "" {
		return false, errors.New("RecordTimeEntry exige source")
	}
	release, err := f.acquire()
	if err != nil {
		return false, err
	}
	defer release()
	return mergeLinks(f, pluginID, e.Source, now, Outcome{TimeEntries: []TimeEntry{e}})
}

// TaskLinks devuelve los vínculos kind task de integrations.yaml, sin tomar el cerrojo (la
// escritura es atómica). Sin fichero, ninguno.
func TaskLinks(f Folder) ([]Link, error) {
	doc, err := readLinks(f)
	if err != nil {
		return nil, err
	}
	var out []Link
	for _, l := range doc.Links {
		if l.Kind == TaskLinkKind {
			out = append(out, l)
		}
	}
	return out, nil
}

// TimeEntries devuelve las imputaciones de integrations.yaml, sin tomar el cerrojo.
func TimeEntries(f Folder) ([]TimeEntry, error) {
	doc, err := readLinks(f)
	if err != nil {
		return nil, err
	}
	return append([]TimeEntry(nil), doc.TimeEntries...), nil
}
