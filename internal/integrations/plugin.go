package integrations

import (
	"context"
	"errors"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

// SecretSource entrega a un plugin sus secretos. Está acotado a los secretos del manifiesto del
// propio plugin: pedir uno no declarado es un error.
type SecretSource interface {
	Secret(ctx context.Context, name string) (secrets.Secret, error)
}

// Delivery es lo que recibe el plugin en cada entrega.
type Delivery struct {
	Event   Event
	Config  map[string]any // valores de integrations.<plugin> (copia), sin `enabled`
	Links   []Link         // instantánea de integrations.yaml en el momento de la entrega
	Secrets SecretSource
}

// Outcome es el resultado de una entrega con éxito: lo que el plugin quiere dejar vinculado.
type Outcome struct {
	Links       []Link
	TimeEntries []TimeEntry
}

// Link vincula un cambio (o uno de sus roles) con un elemento del tracker.
type Link struct {
	Plugin     string    `json:"plugin" yaml:"plugin"`
	Kind       string    `json:"kind" yaml:"kind"` // ^[a-z][a-z0-9_]*$, definido por el plugin
	ID         string    `json:"id" yaml:"id"`
	URL        string    `json:"url,omitempty" yaml:"url,omitempty"` // opcional; http(s), sin userinfo
	Role       string    `json:"role,omitempty" yaml:"role,omitempty"`
	Source     string    `json:"source" yaml:"source"` // clave de idempotencia del evento que lo produjo
	RecordedAt time.Time `json:"recorded_at" yaml:"recorded_at"`
}

// TimeEntry es una imputación de tiempo en un elemento del tracker. Nadie la produce aún (C-07-e).
type TimeEntry struct {
	Plugin     string    `json:"plugin" yaml:"plugin"`
	Role       string    `json:"role,omitempty" yaml:"role,omitempty"`
	ItemID     string    `json:"item_id" yaml:"item_id"`
	Minutes    int       `json:"minutes" yaml:"minutes"`
	Date       string    `json:"date" yaml:"date"` // AAAA-MM-DD
	Source     string    `json:"source" yaml:"source"`
	RecordedAt time.Time `json:"recorded_at" yaml:"recorded_at"`
}

// Plugin es el puerto de un adaptador de tracker.
type Plugin interface {
	Manifest() manifest.Manifest
	// Deliver entrega un evento. DEBE ser idempotente por Event.IdempotencyKey (entrega al menos
	// una vez), respetar ctx y no incluir valores de secretos en el error.
	Deliver(ctx context.Context, d Delivery) (Outcome, error)
}

// ErrPermanent marca un fallo que no se arregla reintentando (configuración inválida, 4xx de
// autorización). La operación pasa directamente a abandonada.
var ErrPermanent = errors.New("fallo permanente de la integración")

// Permanent envuelve err con ErrPermanent.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// IsPermanent indica si err (o algo que envuelve) es un fallo permanente.
func IsPermanent(err error) bool { return errors.Is(err, ErrPermanent) }

type permanentError struct{ err error }

func (e *permanentError) Error() string        { return e.err.Error() }
func (e *permanentError) Unwrap() error        { return e.err }
func (e *permanentError) Is(target error) bool { return target == ErrPermanent }
