package integrations

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	// LinksSchemaV1 es su esquema.
	LinksSchemaV1 = "axiom.integrations/v1"

	maxLinkIDLength = 128
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
	var doc LinksDoc
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return LinksDoc{}, errors.New("el fichero está vacío")
		}
		return LinksDoc{}, err
	}
	if doc.Schema != LinksSchemaV1 {
		return LinksDoc{}, fmt.Errorf("esquema %q no admitido: debe ser %q", doc.Schema, LinksSchemaV1)
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

func validateTimeEntry(e TimeEntry) error {
	if err := validateItemID("item_id", e.ItemID); err != nil {
		return err
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
// En una carpeta archivada cualquier intento de escritura se rechaza con el mensaje de REQ-21.18.
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

	haveLinks := map[linkIdentity]bool{}
	for _, l := range doc.Links {
		haveLinks[linkIdentity{l.Plugin, l.Kind, l.ID, l.Role}] = true
	}
	haveEntries := map[entryIdentity]bool{}
	for _, e := range doc.TimeEntries {
		haveEntries[entryIdentity{e.Plugin, e.Source, e.ItemID}] = true
	}
	changed := false
	for _, l := range links {
		id := linkIdentity{l.Plugin, l.Kind, l.ID, l.Role}
		if !haveLinks[id] {
			haveLinks[id] = true
			doc.Links = append(doc.Links, l)
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

	payload, err := yaml.Marshal(doc)
	if err != nil {
		return false, fmt.Errorf("serializar %s: %w", LinksFile, err)
	}
	if err := f.writeAtomic(LinksFile, linksTempPattern, payload); err != nil {
		return false, err
	}
	return true, nil
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
