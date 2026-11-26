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
	// Deliverer es la identidad git de quien entrega (quien ejecuta el despacho o la sincronización).
	// Un plugin solo asigna una Task al actor de un evento cuando coincide con ella.
	Deliverer string
	// Identities consulta las identidades guardadas de la persona en el tracker. Nil: sin identidades.
	Identities IdentityLookup
}

// IdentityLookup consulta la identidad guardada de la persona en una cuenta del tracker (la
// organización en Azure DevOps). Solo lectura: ok es falso cuando no hay identidad guardada.
type IdentityLookup interface {
	Lookup(plugin, account string) (TrackerIdentity, bool, error)
}

// Outcome es el resultado de una entrega con éxito: lo que el plugin quiere dejar vinculado.
type Outcome struct {
	Links       []Link
	TimeEntries []TimeEntry
	// Warnings son avisos de entrega («<código>: <mensaje>»): el despachador los sanea y los añade a
	// Report.Warnings. Una entrega con avisos sigue siendo una entrega con éxito.
	Warnings []string
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
	// EstimateHours es la estimación confirmada del rol (> 0, dos decimales como mucho). Un valor
	// entrante no nulo y distinto sustituye al presente.
	EstimateHours *float64 `json:"estimate_hours,omitempty" yaml:"estimate_hours,omitempty"`
	// StartedAt (UTC) es el inicio del rol: se fija una sola vez, la primera que consta.
	StartedAt *time.Time `json:"started_at,omitempty" yaml:"started_at,omitempty"`
}

// TimeEntry es una imputación de tiempo en un elemento del tracker. La escribe `axiom integrations
// time record` tras un cambio de horas con éxito; las de cierre de rol (`role.close/...`) llevan actor.
type TimeEntry struct {
	Plugin     string    `json:"plugin" yaml:"plugin"`
	Role       string    `json:"role,omitempty" yaml:"role,omitempty"`
	ItemID     string    `json:"item_id" yaml:"item_id"`
	Minutes    int       `json:"minutes" yaml:"minutes"`
	Date       string    `json:"date" yaml:"date"` // AAAA-MM-DD
	Source     string    `json:"source" yaml:"source"`
	RecordedAt time.Time `json:"recorded_at" yaml:"recorded_at"`
	// Actor es quien imputó las horas (identidad git). Obligatorio en las imputaciones `role.close/*`.
	Actor string `json:"actor,omitempty" yaml:"actor,omitempty"`
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
