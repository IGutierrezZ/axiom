package changestart

import (
	"context"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// Tracker es el plugin activo ligado a su configuración y a los secretos (D-01, D-18). Lo implementa
// la CLI sobre las funciones del puerto de work items; changestart no conoce ningún tracker.
type Tracker interface {
	PluginID() string
	RefAlias() string
	ItemKeys() []manifest.TypePrefix
	ParseRef(raw string) (string, error)
	Read(ctx context.Context, id string) (integrations.ExternalItem, error)
	StateCategory(ctx context.Context, configuredType, state string) (integrations.StateCategory, error)
	Download(ctx context.Context, url string, max int64) (integrations.Attachment, error)
	LinkBack(ctx context.Context, id, tag string, allow func([]string) error) (integrations.BackLink, error)
}
