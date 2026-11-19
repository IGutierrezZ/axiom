// Package integrationstest ofrece un plugin de integración en memoria y su instalación en las
// costuras de prueba (catálogo, registro y almacén de secretos). Es el doble con el que se prueba
// todo el marco sin ningún tracker real.
//
// Install toca estado de paquete (catálogo, registro y almacén por defecto): las pruebas que lo
// usan NO deben llamar a t.Parallel().
package integrationstest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// FakeID es el identificador del plugin de prueba.
const FakeID = "fake"

// FakeTokenSecret es el nombre del único secreto del manifiesto del doble.
const FakeTokenSecret = "token"

// FakeManifest devuelve el manifiesto del ejemplo del diseño (§5.1): id fake, secreto token e
// identidad git_actor.
func FakeManifest() manifest.Manifest {
	return manifest.Manifest{
		Schema:       manifest.SchemaV1,
		ID:           FakeID,
		Title:        "Plugin de prueba en memoria",
		Capabilities: []manifest.Capability{manifest.CapItemsRead, manifest.CapItemsTransition},
		Events: []manifest.EventType{
			manifest.EventGateSpecApproved,
			manifest.EventGateDesignApproved,
			manifest.EventRoleStart,
			manifest.EventRoleApplyApproved,
			manifest.EventArchive,
		},
		Config: []manifest.ConfigField{
			{Key: "project", Type: manifest.FieldString, Required: true, Description: "Proyecto del tracker"},
		},
		Secrets:  []manifest.SecretSpec{{Name: FakeTokenSecret, Description: "Token de acceso", Required: true}},
		Identity: manifest.IdentitySpec{Source: manifest.IdentityGitActor},
	}
}

// Fake es un plugin en memoria. Registra los eventos que recibe con su clave, devuelve vínculos
// configurables, puede fallar siempre, N veces o por evento, y es idempotente por clave.
type Fake struct {
	mu sync.Mutex

	manifest manifest.Manifest

	received   []integrations.Delivery // todas las entregas recibidas, también las fallidas
	delivered  []integrations.Event    // eventos entregados con éxito, una vez por clave
	byKey      map[string]integrations.Outcome
	links      []integrations.Link
	failAlways error
	failTimes  int
	failErr    error
	failEvents map[manifest.EventType]error
	readToken  bool
	leakToken  bool
	block      <-chan struct{}
	panicWith  any
	hasPanic   bool
}

// NewFake crea el doble con el manifiesto de FakeManifest.
func NewFake() *Fake {
	return &Fake{manifest: FakeManifest(), byKey: map[string]integrations.Outcome{}, failEvents: map[manifest.EventType]error{}}
}

// Manifest implementa integrations.Plugin.
func (f *Fake) Manifest() manifest.Manifest { return f.manifest.Clone() }

// ReturnLinks fija los vínculos que devuelven las entregas con éxito (los pisa).
func (f *Fake) ReturnLinks(links ...integrations.Link) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.links = append([]integrations.Link(nil), links...)
	return f
}

// FailAlways hace que toda entrega falle con err hasta llamar a Recover.
func (f *Fake) FailAlways(err error) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failAlways = err
	return f
}

// FailTimes hace que las próximas n entregas fallen con err y las siguientes tengan éxito.
func (f *Fake) FailTimes(n int, err error) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failTimes, f.failErr = n, err
	return f
}

// FailOnEvent hace que toda entrega del evento t falle con err.
func (f *Fake) FailOnEvent(t manifest.EventType, err error) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failEvents[t] = err
	return f
}

// Recover quita todos los fallos programados.
func (f *Fake) Recover() *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failAlways, f.failTimes, f.failErr = nil, 0, nil
	f.failEvents = map[manifest.EventType]error{}
	return f
}

// ReadToken hace que cada entrega pida el secreto token a su SecretSource. Si no se puede
// resolver, la entrega falla con ese error.
func (f *Fake) ReadToken() *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readToken = true
	return f
}

// LeakTokenInError hace que, tras pedir el secreto token, la entrega falle con un error que lo
// incluye en el mensaje (para probar la limpieza de secretos). Implica ReadToken.
func (f *Fake) LeakTokenInError() *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readToken, f.leakToken = true, true
	return f
}

// BlockUntil hace que Deliver espere a que se cierre ch IGNORANDO el contexto (un plugin mal
// escrito que no respeta el plazo).
func (f *Fake) BlockUntil(ch <-chan struct{}) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.block = ch
	return f
}

// PanicWith hace que Deliver entre en pánico con v.
func (f *Fake) PanicWith(v any) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.panicWith, f.hasPanic = v, true
	return f
}

// Deliver implementa integrations.Plugin.
func (f *Fake) Deliver(ctx context.Context, d integrations.Delivery) (integrations.Outcome, error) {
	f.mu.Lock()
	f.received = append(f.received, cloneDelivery(d))
	block, hasPanic, panicVal := f.block, f.hasPanic, f.panicWith
	readToken, leakToken := f.readToken, f.leakToken
	f.mu.Unlock()

	if hasPanic {
		panic(panicVal)
	}
	if block != nil {
		<-block
	}
	if err := ctx.Err(); err != nil {
		return integrations.Outcome{}, err
	}

	var token string
	if readToken {
		if d.Secrets == nil {
			return integrations.Outcome{}, errors.New("el doble pidió el secreto token pero la entrega no trae SecretSource")
		}
		secret, err := d.Secrets.Secret(ctx, FakeTokenSecret)
		if err != nil {
			return integrations.Outcome{}, err
		}
		token = secret.Reveal()
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.programmedFailure(d.Event); err != nil {
		return integrations.Outcome{}, err
	}
	if leakToken {
		return integrations.Outcome{}, fmt.Errorf("el tracker rechazó la credencial %s", token)
	}

	if out, seen := f.byKey[d.Event.IdempotencyKey]; seen {
		return cloneOutcome(out), nil // idempotente por clave: no se registra otra vez
	}
	out := integrations.Outcome{Links: append([]integrations.Link(nil), f.links...)}
	f.byKey[d.Event.IdempotencyKey] = out
	f.delivered = append(f.delivered, d.Event)
	return cloneOutcome(out), nil
}

// programmedFailure devuelve el fallo que toca, o nil. Se llama con f.mu tomado.
func (f *Fake) programmedFailure(ev integrations.Event) error {
	if err := f.failEvents[ev.Type]; err != nil {
		return err
	}
	if f.failAlways != nil {
		return f.failAlways
	}
	if f.failTimes > 0 {
		f.failTimes--
		return f.failErr
	}
	return nil
}

// Delivered devuelve los eventos entregados con éxito, una vez por clave y en orden de llegada.
func (f *Fake) Delivered() []integrations.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]integrations.Event(nil), f.delivered...)
}

// DeliveredKeys devuelve las claves de los eventos entregados con éxito.
func (f *Fake) DeliveredKeys() []string {
	events := f.Delivered()
	keys := make([]string, len(events))
	for i, e := range events {
		keys[i] = e.IdempotencyKey
	}
	return keys
}

// Received devuelve todas las entregas recibidas (también las que fallaron), con copias de la
// configuración y de los vínculos.
func (f *Fake) Received() []integrations.Delivery {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]integrations.Delivery, len(f.received))
	copy(out, f.received)
	return out
}

// Calls devuelve cuántas veces se llamó a Deliver.
func (f *Fake) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.received)
}

func cloneDelivery(d integrations.Delivery) integrations.Delivery {
	out := d
	out.Links = append([]integrations.Link(nil), d.Links...)
	if d.Config != nil {
		out.Config = make(map[string]any, len(d.Config))
		for k, v := range d.Config {
			out.Config[k] = v
		}
	}
	return out
}

func cloneOutcome(o integrations.Outcome) integrations.Outcome {
	return integrations.Outcome{
		Links:       append([]integrations.Link(nil), o.Links...),
		TimeEntries: append([]integrations.TimeEntry(nil), o.TimeEntries...),
	}
}
