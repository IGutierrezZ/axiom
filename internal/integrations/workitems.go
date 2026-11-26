package integrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// PortWorkItems es el puerto opcional que lee un work item del tracker, descarga sus adjuntos y le
// añade la etiqueta de vuelta de un cambio (C-07-d). Como el resto de puertos opcionales, se detecta
// por aserción de tipo: un plugin sin él sigue funcionando.
const PortWorkItems = "work_items"

// Errores neutrales del puerto de work items. El adaptador los produce en la frontera del puerto; el
// núcleo no conoce ningún tracker.
var (
	ErrItemNotFound        = errors.New("el tracker no encontró el elemento: revisa el identificador con `axiom change start --from <id|url>`")
	ErrItemConflict        = errors.New("el elemento cambió mientras se escribía: repite `axiom change start --adopt --change <cambio>`")
	ErrResourceURLRejected = errors.New("URL fuera del host y la organización configurados") // refusal:by-design world-action: la URL la decide el contenido del tracker; el adjunto se omite y se lista
	ErrAttachmentTooLarge  = errors.New("el adjunto supera el límite de descarga")           // refusal:by-design world-action: el tamaño lo decide el tracker; el adjunto se omite y se lista
)

// ResourceURLRejection explica sin red por qué una URL no recibe la credencial. Reason es un valor
// cerrado: scheme, userinfo, host, organization o path. errors.Is(err, ErrResourceURLRejected).
type ResourceURLRejection struct {
	Reason string
}

func (e *ResourceURLRejection) Error() string {
	return fmt.Sprintf("%s (motivo: %s)", ErrResourceURLRejected.Error(), e.Reason)
}

func (e *ResourceURLRejection) Is(target error) bool { return target == ErrResourceURLRejected }

// ItemRelation es un elemento relacionado (padre o Related) con su tipo y título, si se pudieron leer.
type ItemRelation struct{ ID, Type, Title string }

// ItemAttachment es un adjunto de un elemento. Size es el tamaño declarado; 0 si se desconoce.
type ItemAttachment struct {
	Name string
	URL  string
	Size int64
}

// ExternalItem es la lectura neutral de un elemento del tracker. Los campos HTML llegan tal cual. No
// lleva la categoría del estado: la da StateCategory.
type ExternalItem struct {
	ID, Type, Title, State, URL                 string
	Description, ReproSteps, AcceptanceCriteria string
	Tags                                        []string
	Parent                                      *ItemRelation
	Related                                     []ItemRelation
	Attachments                                 []ItemAttachment
	Warnings                                    []string // degradaciones saneadas (p. ej. relaciones sin título)
}

// ItemRequest pide la lectura del elemento ID.
type ItemRequest struct {
	PortRequest
	ID string
}

// StateRequest pide la categoría de State en el tipo Type, con Type en la grafía configurada.
type StateRequest struct {
	PortRequest
	Type, State string
}

// StateCategory es la categoría de un estado. Known es false si el estado no figura entre los de su
// tipo.
type StateCategory struct {
	Category string // Proposed | InProgress | Resolved | Completed | Removed | …
	Known    bool
}

// AttachmentRequest pide la descarga de URL con un máximo de MaxBytes.
type AttachmentRequest struct {
	PortRequest
	URL      string
	MaxBytes int64
}

// Attachment es un adjunto descargado. ContentType va normalizado y es application/octet-stream si
// el servidor no lo declaró.
type Attachment struct {
	Data        []byte
	ContentType string
}

// BackLinkRequest pide añadir Tag al elemento ID. Allow recibe las etiquetas actuales y devuelve un
// error si alguna impide añadir Tag.
type BackLinkRequest struct {
	PortRequest
	ID, Tag string
	Allow   func(tags []string) error
}

// BackLinkOutcome es el resultado del enlace de vuelta.
type BackLinkOutcome string

const (
	BackLinkAdded    BackLinkOutcome = "added"
	BackLinkPresent  BackLinkOutcome = "present"
	BackLinkConflict BackLinkOutcome = "conflict"
)

// BackLink es el resultado de LinkBack. Conflicting son las etiquetas que impidieron añadir la
// propia.
type BackLink struct {
	Outcome     BackLinkOutcome
	Conflicting []string
}

// WorkItemSource es el puerto opcional de work items.
type WorkItemSource interface {
	// ParseItemRef valida y normaliza la referencia (id o URL) sin red y devuelve el identificador.
	ParseItemRef(cfg map[string]any, raw string) (string, error)
	ReadItem(ctx context.Context, req ItemRequest) (ExternalItem, error)
	StateCategory(ctx context.Context, req StateRequest) (StateCategory, error)
	DownloadAttachment(ctx context.Context, req AttachmentRequest) (Attachment, error)
	LinkBack(ctx context.Context, req BackLinkRequest) (BackLink, error)
}

// workItemSource detecta el puerto por aserción de tipo.
func workItemSource(p Plugin) (WorkItemSource, error) {
	if src, ok := p.(WorkItemSource); ok {
		return src, nil
	}
	return nil, &PortUnsupportedError{Plugin: p.Manifest().ID, Port: PortWorkItems}
}

// ParseItemRef interpreta la referencia de --from sin red ni secretos. Un plugin sin el puerto
// devuelve *PortUnsupportedError.
func ParseItemRef(p Plugin, cfg workspace.IntegrationPluginConfig, raw string) (string, error) {
	src, err := workItemSource(p)
	if err != nil {
		return "", err
	}
	tracker := newSecretTracker()
	values := portRequest(p, cfg, nil, tracker).Config
	return invokePort(tracker, p.Manifest().ID, PortWorkItems, func() (string, error) {
		return src.ParseItemRef(values, raw)
	})
}

// ReadItem lee el elemento id con las garantías de invokePort.
func ReadItem(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store, id string) (ExternalItem, error) {
	src, err := workItemSource(p)
	if err != nil {
		return ExternalItem{}, err
	}
	tracker := newSecretTracker()
	req := ItemRequest{PortRequest: portRequest(p, cfg, store, tracker), ID: id}
	return invokePort(tracker, p.Manifest().ID, PortWorkItems, func() (ExternalItem, error) {
		return src.ReadItem(ctx, req)
	})
}

// ReadStateCategory consulta la categoría del estado state del tipo typ (grafía configurada).
func ReadStateCategory(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store, typ, state string) (StateCategory, error) {
	src, err := workItemSource(p)
	if err != nil {
		return StateCategory{}, err
	}
	tracker := newSecretTracker()
	req := StateRequest{PortRequest: portRequest(p, cfg, store, tracker), Type: typ, State: state}
	return invokePort(tracker, p.Manifest().ID, PortWorkItems, func() (StateCategory, error) {
		return src.StateCategory(ctx, req)
	})
}

// DownloadAttachment descarga url con un máximo de max bytes.
func DownloadAttachment(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store, url string, max int64) (Attachment, error) {
	src, err := workItemSource(p)
	if err != nil {
		return Attachment{}, err
	}
	tracker := newSecretTracker()
	req := AttachmentRequest{PortRequest: portRequest(p, cfg, store, tracker), URL: url, MaxBytes: max}
	return invokePort(tracker, p.Manifest().ID, PortWorkItems, func() (Attachment, error) {
		return src.DownloadAttachment(ctx, req)
	})
}

// LinkBack añade la etiqueta tag al elemento id. allow decide si las etiquetas actuales impiden
// añadirla; un pánico de allow lo recupera invokePort.
func LinkBack(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store, id, tag string, allow func([]string) error) (BackLink, error) {
	src, err := workItemSource(p)
	if err != nil {
		return BackLink{}, err
	}
	tracker := newSecretTracker()
	req := BackLinkRequest{PortRequest: portRequest(p, cfg, store, tracker), ID: id, Tag: tag, Allow: allow}
	return invokePort(tracker, p.Manifest().ID, PortWorkItems, func() (BackLink, error) {
		return src.LinkBack(ctx, req)
	})
}
