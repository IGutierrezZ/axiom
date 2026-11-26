package integrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Puertos opcionales de un plugin (RF-INT-23). El puerto Plugin (Manifest, Deliver) no cambia: un
// plugin implementa estos solo si los ofrece, y las funciones ResolveIdentity y Discover los detectan
// por aserción de tipo.

// Nombres de los puertos, tal como los nombran los errores tipados.
const (
	PortIdentity  = "identity"
	PortDiscovery = "discovery"
)

// PortRequest es lo que recibe un puerto opcional: la configuración del plugin y sus secretos.
type PortRequest struct {
	Config  map[string]any // copia de integrations.<plugin>, sin `enabled`
	Secrets SecretSource   // acotada al manifiesto y rastreada para la redacción
}

// IdentityRequest es la petición del puerto de identidad.
type IdentityRequest struct {
	PortRequest
	// User es el identificador elegido de forma explícita (--user); vacío es la vía automática del
	// plugin.
	User string
}

// TrackerIdentity es la identidad de la persona en el tracker. Es serializable en JSON.
type TrackerIdentity struct {
	Account     string `json:"account"` // clave de indexado: la organización en Azure DevOps
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Source      string `json:"source"` // lo define el plugin: connection_data | member_selection
}

// IdentityResolver es el puerto opcional que resuelve la identidad de la persona en el tracker.
type IdentityResolver interface {
	ResolveIdentity(ctx context.Context, req IdentityRequest) (TrackerIdentity, error)
}

// Discoverer es el puerto opcional que describe el proceso del tracker.
type Discoverer interface {
	Discover(ctx context.Context, req PortRequest) (Discovery, error)
}

// DiscoverySchemaV1 es el esquema del documento de descubrimiento.
const DiscoverySchemaV1 = "axiom.integration-discovery/v1"

// Discovery es el proceso del tracker: tipos de elemento con sus estados y campos, iteraciones y
// miembros. Es neutro respecto al tracker; solo Scope y los valores de Source son del plugin.
type Discovery struct {
	Schema     string            `json:"schema"`
	Plugin     string            `json:"plugin"`
	Scope      map[string]string `json:"scope"` // en Azure DevOps: organization, project, team
	ItemTypes  []ItemType        `json:"item_types"`
	Iterations []Iteration       `json:"iterations"`
	Members    []Member          `json:"members"`
}

// ItemType es un tipo de elemento del tracker (en Azure DevOps, un tipo de work item).
type ItemType struct {
	Name          string      `json:"name"`
	ReferenceName string      `json:"reference_name"`
	States        []ItemState `json:"states"`
	Fields        []ItemField `json:"fields"`
}

// ItemState es un estado de un tipo de elemento.
type ItemState struct {
	Name     string `json:"name"`
	Category string `json:"category"` // tal como lo devuelve el tracker, sin normalizar
	Color    string `json:"color,omitempty"`
}

// ItemField es un campo de un tipo de elemento.
type ItemField struct {
	ReferenceName  string `json:"reference_name"`
	Name           string `json:"name"`
	AlwaysRequired bool   `json:"always_required"`
}

// Iteration es una iteración (sprint) del equipo.
type Iteration struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	StartDate  string `json:"start_date,omitempty"` // texto de la API, sin reinterpretar
	FinishDate string `json:"finish_date,omitempty"`
	TimeFrame  string `json:"time_frame,omitempty"` // past | current | future
}

// Member es un miembro del equipo. Hace también de candidato de identidad.
type Member struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// Errores tipados de los puertos.
var (
	// ErrPortUnsupported indica que el plugin no ofrece el puerto pedido.
	ErrPortUnsupported = errors.New("el plugin no ofrece el puerto")
	// ErrIdentitySelectionRequired indica que la identidad no se pudo resolver sola y hace falta
	// elegir el usuario del tracker con --user.
	ErrIdentitySelectionRequired = errors.New("hace falta elegir el usuario del tracker con --user")
	// ErrInvalidPortResult indica que el plugin devolvió un resultado que no cumple el contrato del
	// puerto.
	ErrInvalidPortResult = errors.New("el plugin devolvió un resultado inválido")
)

// PortUnsupportedError nombra el plugin y el puerto que falta. errors.Is(err, ErrPortUnsupported).
type PortUnsupportedError struct{ Plugin, Port string }

func (e *PortUnsupportedError) Error() string {
	return fmt.Sprintf("el plugin %s no ofrece el puerto de %s", e.Plugin, portLabel(e.Port))
}

func (e *PortUnsupportedError) Is(target error) bool { return target == ErrPortUnsupported }

func portLabel(port string) string {
	switch port {
	case PortIdentity:
		return "identidad"
	case PortDiscovery:
		return "descubrimiento"
	case PortWorkItems:
		return "work items"
	case PortTasks:
		return "Tasks"
	}
	return port
}

// IdentitySelectionError indica que la identidad no se resolvió sola. Candidates son los miembros
// del equipo con su id y nombre visible. errors.Is(err, ErrIdentitySelectionRequired).
type IdentitySelectionError struct {
	Plugin, Account string
	// User vacío: no se indicó --user; no vacío: no coincide con ningún miembro.
	User string
	// Reason es por qué no sirvió la vía automática (saneado).
	Reason     string
	Candidates []Member
}

func (e *IdentitySelectionError) Error() string {
	msg := ErrIdentitySelectionRequired.Error()
	if e.User != "" {
		msg = fmt.Sprintf("el usuario %q no coincide con ningún miembro del equipo", e.User)
	}
	if e.Reason != "" {
		msg += ": " + e.Reason
	}
	return fmt.Sprintf("%s (%d candidato(s))", msg, len(e.Candidates))
}

func (e *IdentitySelectionError) Is(target error) bool { return target == ErrIdentitySelectionRequired }

// PortError envuelve el fallo de un puerto: Error() es el texto saneado con los secretos rastreados
// y Unwrap() conserva la causa para errors.Is y errors.As.
type PortError struct {
	Plugin, Port, Message string
	Err                   error
}

func (e *PortError) Error() string { return e.Message }
func (e *PortError) Unwrap() error { return e.Err }

// ResolveIdentity invoca el puerto de identidad de p. Lo detecta por aserción de tipo antes de
// resolver ningún secreto: un plugin sin puerto devuelve *PortUnsupportedError sin efectos. Copia la
// configuración, acota y rastrea los secretos, recupera un pánico como error y sanea el texto del
// error con los valores de secreto resueltos; la causa se conserva en Unwrap.
func ResolveIdentity(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store, user string) (TrackerIdentity, error) {
	m := p.Manifest()
	resolver, ok := p.(IdentityResolver)
	if !ok {
		return TrackerIdentity{}, &PortUnsupportedError{Plugin: m.ID, Port: PortIdentity}
	}
	tracker := newSecretTracker()
	req := IdentityRequest{PortRequest: portRequest(p, cfg, store, tracker), User: user}

	identity, err := invokePort(tracker, m.ID, PortIdentity, func() (TrackerIdentity, error) {
		return resolver.ResolveIdentity(ctx, req)
	})
	if err != nil {
		return TrackerIdentity{}, err
	}
	if identity.ID == "" || identity.Account == "" {
		return TrackerIdentity{}, portFailure(tracker, m.ID, PortIdentity,
			fmt.Errorf("%w: la identidad no trae id o cuenta", ErrInvalidPortResult))
	}
	return identity, nil
}

// Discover invoca el puerto de descubrimiento de p, con las mismas garantías que ResolveIdentity.
// Fija Schema y Plugin y convierte las listas nulas en vacías: el JSON nunca lleva null en listas.
func Discover(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store) (Discovery, error) {
	m := p.Manifest()
	discoverer, ok := p.(Discoverer)
	if !ok {
		return Discovery{}, &PortUnsupportedError{Plugin: m.ID, Port: PortDiscovery}
	}
	tracker := newSecretTracker()
	req := portRequest(p, cfg, store, tracker)

	discovery, err := invokePort(tracker, m.ID, PortDiscovery, func() (Discovery, error) {
		return discoverer.Discover(ctx, req)
	})
	if err != nil {
		return Discovery{}, err
	}
	discovery.Schema = DiscoverySchemaV1
	discovery.Plugin = m.ID
	return discovery.withNonNilLists(), nil
}

// withNonNilLists devuelve el descubrimiento con todas las listas y el mapa de ámbito no nulos.
func (d Discovery) withNonNilLists() Discovery {
	if d.Scope == nil {
		d.Scope = map[string]string{}
	}
	if d.Iterations == nil {
		d.Iterations = []Iteration{}
	}
	if d.Members == nil {
		d.Members = []Member{}
	}
	types := make([]ItemType, len(d.ItemTypes))
	for i, t := range d.ItemTypes {
		if t.States == nil {
			t.States = []ItemState{}
		}
		if t.Fields == nil {
			t.Fields = []ItemField{}
		}
		types[i] = t
	}
	d.ItemTypes = types
	return d
}

// portRequest compone la petición de un puerto: copia de la configuración y fuente de secretos
// acotada al manifiesto y rastreada.
func portRequest(p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store, tracker *secretTracker) PortRequest {
	values := make(map[string]any, len(cfg.Values))
	for k, v := range cfg.Values {
		values[k] = v
	}
	return PortRequest{Config: values, Secrets: newScopedSecrets(p.Manifest(), store, tracker)}
}

// invokePort ejecuta call recuperando un pánico y envuelve el fallo en *PortError saneado.
func invokePort[T any](tracker *secretTracker, plugin, port string, call func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			result, err = zero, portFailure(tracker, plugin, port, fmt.Errorf("el puerto entró en pánico: %v", r))
		}
	}()
	result, err = call()
	if err != nil {
		var zero T
		return zero, portFailure(tracker, plugin, port, err)
	}
	return result, nil
}

// portFailure construye el *PortError de un fallo: texto saneado y causa conservada. El motivo de un
// IdentitySelectionError también se sanea, porque la CLI lo imprime desde su campo.
func portFailure(tracker *secretTracker, plugin, port string, err error) *PortError {
	cause := err
	var selection *IdentitySelectionError
	if errors.As(err, &selection) {
		clean := *selection
		clean.Reason = sanitizeText(selection.Reason, tracker)
		cause = &clean
	}
	return &PortError{Plugin: plugin, Port: port, Message: sanitizeError(cause, tracker), Err: cause}
}
