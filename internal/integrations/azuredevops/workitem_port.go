package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

var _ integrations.WorkItemSource = (*Plugin)(nil)

// Campos de un work item que lee el puerto.
const (
	fieldType       = "System.WorkItemType"
	fieldTitle      = "System.Title"
	fieldState      = "System.State"
	fieldTags       = "System.Tags"
	fieldDesc       = "System.Description"
	fieldRepro      = "Microsoft.VSTS.TCM.ReproSteps"
	fieldAcceptance = "Microsoft.VSTS.Common.AcceptanceCriteria"
)

// Tipos de relación que traduce el puerto; el resto se ignora.
const (
	relParent   = "System.LinkTypes.Hierarchy-Reverse"
	relRelated  = "System.LinkTypes.Related"
	relAttached = "AttachedFile"
)

// maxRelationBatch es el máximo de elementos relacionados cuyo tipo y título se piden.
const maxRelationBatch = 50

var relationItemPath = []string{"_apis", "wit", "workitems"}

// ReadItem implementa integrations.WorkItemSource. Hace como mucho dos peticiones: el elemento con
// $expand=relations y un único lote con los identificadores del padre y de los Related. Solo el 404
// de la primera es integrations.ErrItemNotFound; un fallo del lote degrada a un aviso y deja las
// relaciones con su identificador. La URL del elemento sale de la configuración, nunca del contenido.
func (p *Plugin) ReadItem(ctx context.Context, req integrations.ItemRequest) (integrations.ExternalItem, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.ExternalItem{}, err
	}
	id, err := strconv.Atoi(req.ID)
	if err != nil || id <= 0 {
		return integrations.ExternalItem{}, invalidWorkItemID(id)
	}
	wi, err := client.GetWorkItem(ctx, id)
	if err != nil {
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			return integrations.ExternalItem{}, fmt.Errorf("%w: %w", integrations.ErrItemNotFound, err)
		}
		return integrations.ExternalItem{}, err
	}

	item := integrations.ExternalItem{
		ID:                 strconv.Itoa(wi.ID),
		Type:               fieldString(wi.Fields, fieldType),
		Title:              fieldString(wi.Fields, fieldTitle),
		State:              fieldString(wi.Fields, fieldState),
		URL:                client.workItemURL(wi.ID),
		Description:        fieldString(wi.Fields, fieldDesc),
		ReproSteps:         fieldString(wi.Fields, fieldRepro),
		AcceptanceCriteria: fieldString(wi.Fields, fieldAcceptance),
		Tags:               splitTags(fieldString(wi.Fields, fieldTags)),
	}

	var parent string
	var related []string
	for _, rel := range wi.Relations {
		switch rel.Rel {
		case relParent:
			if rid, ok := client.relatedItemID(rel.URL); ok && parent == "" {
				parent = rid
			}
		case relRelated:
			if rid, ok := client.relatedItemID(rel.URL); ok {
				related = append(related, rid)
			}
		case relAttached:
			item.Attachments = append(item.Attachments, integrations.ItemAttachment{
				Name: attributeString(rel.Attributes, "name"),
				URL:  rel.URL,
				Size: attributeInt(rel.Attributes, "resourceSize"),
			})
		}
	}

	titles := client.relationTitles(ctx, parent, related, &item)
	if parent != "" {
		r := titles.relation(parent)
		item.Parent = &r
	}
	for _, rid := range related {
		item.Related = append(item.Related, titles.relation(rid))
	}
	return item, nil
}

// relationInfo guarda el tipo y el título leídos de los elementos relacionados.
type relationInfo map[string]integrations.ItemRelation

func (r relationInfo) relation(id string) integrations.ItemRelation {
	if rel, ok := r[id]; ok {
		return rel
	}
	return integrations.ItemRelation{ID: id}
}

// relationTitles lee de una sola vez el tipo y el título de los relacionados. Cualquier fallo del
// lote (incluido un 404) degrada a un aviso fijo de Axiom en item.Warnings.
func (c *Client) relationTitles(ctx context.Context, parent string, related []string, item *integrations.ExternalItem) relationInfo {
	seen := map[string]bool{}
	var ids []int
	for _, raw := range append([]string{parent}, related...) {
		n, err := strconv.Atoi(raw)
		if raw == "" || err != nil || seen[raw] || len(ids) >= maxRelationBatch {
			continue
		}
		seen[raw] = true
		ids = append(ids, n)
	}
	info := relationInfo{}
	if len(ids) == 0 {
		return info
	}
	items, err := c.GetWorkItems(ctx, ids)
	if err != nil {
		item.Warnings = append(item.Warnings, "no se pudieron leer el tipo y el título de los elementos relacionados: quedan solo con su identificador")
		return info
	}
	for _, wi := range items {
		id := strconv.Itoa(wi.ID)
		info[id] = integrations.ItemRelation{ID: id, Type: fieldString(wi.Fields, fieldType), Title: fieldString(wi.Fields, fieldTitle)}
	}
	return info
}

// workItemURL construye la URL de un elemento desde la configuración, nunca desde el contenido.
func (c *Client) workItemURL(id int) string {
	return strings.TrimRight(c.endpoint.String(), "/") + "/" + url.PathEscape(c.target.Organization) + "/" +
		url.PathEscape(c.target.Project) + "/_workitems/edit/" + strconv.Itoa(id)
}

// relatedItemID extrae el identificador de la URL de una relación, y solo si es del host y la
// organización configurados y tiene la forma {org}/_apis/wit/workItems/{id}. Una relación de otro host
// u organización, o ilegible, se ignora: no genera ninguna petición.
func (c *Client) relatedItemID(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || !strings.EqualFold(u.Scheme, c.endpoint.Scheme) || !strings.EqualFold(u.Host, c.endpoint.Host) {
		return "", false
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if prefix := strings.Trim(c.endpoint.Path, "/"); prefix != "" {
		if len(segments) == 0 || strings.Join(segments[:min(len(segments), len(strings.Split(prefix, "/")))], "/") != prefix {
			return "", false
		}
		segments = segments[len(strings.Split(prefix, "/")):]
	}
	// {org}[/{project}]/_apis/wit/workItems/{id}
	if len(segments) < 5 || !strings.EqualFold(segments[0], c.target.Organization) {
		return "", false
	}
	tail := segments[len(segments)-4:]
	for i, want := range relationItemPath {
		if !strings.EqualFold(tail[i], want) {
			return "", false
		}
	}
	if !validItemID(tail[3]) {
		return "", false
	}
	return tail[3], true
}

// StateCategory implementa integrations.WorkItemSource con un único GET
// {project}/_apis/wit/workitemtypes/{tipo}/states. El tipo llega en la grafía configurada y se escapa
// como un solo segmento. Cualquier error, incluido un 404, se devuelve tal cual (no es ErrItemNotFound).
// Un estado que no figura entre los del tipo (sin distinguir mayúsculas) da Known=false sin error.
func (p *Plugin) StateCategory(ctx context.Context, req integrations.StateRequest) (integrations.StateCategory, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.StateCategory{}, err
	}
	var states processStatesResponse
	if err := client.do(ctx, request{
		method:    "GET",
		segments:  []string{client.target.Project, "_apis", "wit", "workitemtypes", req.Type, "states"},
		scope:     scopeWorkWrite,
		operation: "leer los estados del tipo de work item",
		retryable: true,
	}, &states); err != nil {
		return integrations.StateCategory{}, err
	}
	for _, s := range states.Value {
		if strings.EqualFold(s.Name, strings.TrimSpace(req.State)) {
			return integrations.StateCategory{Category: s.Category, Known: true}, nil
		}
	}
	return integrations.StateCategory{}, nil
}

// DownloadAttachment implementa integrations.WorkItemSource. Traduce al error neutral
// *integrations.ResourceURLRejection el rechazo previo a la red de una URL ajena.
func (p *Plugin) DownloadAttachment(ctx context.Context, req integrations.AttachmentRequest) (integrations.Attachment, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.Attachment{}, err
	}
	att, err := client.Download(ctx, req.URL, req.MaxBytes)
	if err != nil {
		var foreign *ForeignURLError
		if errors.As(err, &foreign) {
			return integrations.Attachment{}, &integrations.ResourceURLRejection{Reason: foreign.Reason}
		}
		return integrations.Attachment{}, err
	}
	return att, nil
}

func fieldString(fields map[string]any, name string) string {
	s, _ := fields[name].(string)
	return s
}

func attributeString(attrs map[string]any, name string) string {
	s, _ := attrs[name].(string)
	return s
}

func attributeInt(attrs map[string]any, name string) int64 {
	switch n := attrs[name].(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	}
	return 0
}

// splitTags separa System.Tags por ";" y recorta cada etiqueta, descartando las vacías.
func splitTags(raw string) []string {
	var tags []string
	for _, t := range strings.Split(raw, ";") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
