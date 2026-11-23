// Package integrations es el marco de plugins de integración: define el puerto Plugin, el registro
// cerrado, los eventos del ciclo de vida con sus claves de idempotencia, la cola local de
// operaciones, los vínculos versionados y el despachador con reintento. No conoce ningún tracker
// concreto: los adaptadores reales viven en subpaquetes (internal/integrations/azuredevops) y los
// ensambla internal/integrations/builtin.
package integrations

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// EventType es el nombre de un evento del catálogo cerrado.
type EventType = manifest.EventType

// fingerprintHexLen es la cantidad de hex del digest que forma la huella (D-09).
const fingerprintHexLen = 16

// Event es un hecho del ciclo de vida de un cambio que se entrega a los plugins.
type Event struct {
	Type           EventType `json:"type" yaml:"type"`
	Change         string    `json:"change" yaml:"change"`
	Role           string    `json:"role,omitempty" yaml:"role,omitempty"`
	Gate           string    `json:"gate,omitempty" yaml:"gate,omitempty"` // spec, design, role-apply:<rol>
	Actor          string    `json:"actor" yaml:"actor"`                   // C-07-a: identidad git, --actor o "cli"
	OccurredAt     time.Time `json:"occurred_at" yaml:"occurred_at"`
	ArtifactDigest string    `json:"artifact_digest,omitempty" yaml:"artifact_digest,omitempty"`
	Fingerprint    string    `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	IdempotencyKey string    `json:"idempotency_key" yaml:"idempotency_key"`
}

// ErrUnknownEvent identifica un nombre de evento fuera del catálogo cerrado. errors.Is lo reconoce
// dentro de *UnknownEventError.
var ErrUnknownEvent = errors.New("evento de integración fuera del catálogo")

// UnknownEventError es el error tipado de NewEvent: nombra el evento rechazado y los admitidos.
type UnknownEventError struct {
	Name string
}

func (e *UnknownEventError) Error() string {
	return fmt.Sprintf("evento %q fuera del catálogo; eventos admitidos: %s", e.Name, manifest.EventNames())
}

func (e *UnknownEventError) Is(target error) bool { return target == ErrUnknownEvent }

// NewEvent es el único constructor de eventos a partir de un nombre en texto: rechaza lo que no
// esté en el catálogo cerrado y un cambio vacío. El llamador completa el resto de campos.
func NewEvent(name, change string) (Event, error) {
	if !manifest.ValidEvent(EventType(name)) {
		return Event{}, &UnknownEventError{Name: name}
	}
	if strings.TrimSpace(change) == "" {
		return Event{}, errors.New("el evento necesita el nombre del cambio")
	}
	return Event{Type: EventType(name), Change: change}, nil
}

// Fingerprint devuelve la huella de una aprobación (D-09): "d" más los 16 primeros hex del digest
// del artefacto o, si el digest está vacío, "o<n>" con n el ordinal (desde 1) de esa aprobación
// entre las de la misma compuerta.
func Fingerprint(digest string, ordinal int) string {
	if digest == "" {
		return fmt.Sprintf("o%d", ordinal)
	}
	if len(digest) > fingerprintHexLen {
		digest = digest[:fingerprintHexLen]
	}
	return "d" + digest
}

// GateApprovedKey es la clave de gate.spec.approved y gate.design.approved.
func GateApprovedKey(t EventType, change, fingerprint string) string {
	return fmt.Sprintf("%s/%s/%s", t, change, fingerprint)
}

// RoleApplyApprovedKey es la clave de role.apply.approved.
func RoleApplyApprovedKey(change, role, fingerprint string) string {
	return fmt.Sprintf("%s/%s/%s/%s", manifest.EventRoleApplyApproved, change, role, fingerprint)
}

// RoleStartKey es la clave de role.start (sin huella).
func RoleStartKey(change, role string) string {
	return fmt.Sprintf("%s/%s/%s", manifest.EventRoleStart, change, role)
}

// ArchiveKey es la clave de archive; archivedFolder es el nombre <fecha>-<cambio> de la carpeta.
func ArchiveKey(change, archivedFolder string) string {
	return fmt.Sprintf("%s/%s/%s", manifest.EventArchive, change, archivedFolder)
}

// EventPolicy resume las reglas propias de cada evento.
type EventPolicy struct {
	// Emittable indica si Axiom puede emitirlo en esta versión (change.start solo es catálogo).
	Emittable bool
	// NeedsRole indica que el evento lleva un rol del roster sellado.
	NeedsRole bool
	// NeedsApprovedGate indica que nace de una compuerta con decisión approved en gates.yaml.
	NeedsApprovedGate bool
	// PersistOutcome indica si los vínculos devueltos se fusionan en integrations.yaml. Es falso
	// para archive: integrations.yaml archivado nunca se reescribe.
	PersistOutcome bool
}

// PolicyFor devuelve la política del evento; un evento desconocido tiene la política vacía.
func PolicyFor(t EventType) EventPolicy {
	switch t {
	case manifest.EventChangeStart:
		return EventPolicy{PersistOutcome: true}
	case manifest.EventGateSpecApproved, manifest.EventGateDesignApproved:
		return EventPolicy{Emittable: true, NeedsApprovedGate: true, PersistOutcome: true}
	case manifest.EventRoleStart:
		return EventPolicy{Emittable: true, NeedsRole: true, PersistOutcome: true}
	case manifest.EventRoleApplyApproved:
		return EventPolicy{Emittable: true, NeedsRole: true, NeedsApprovedGate: true, PersistOutcome: true}
	case manifest.EventArchive:
		return EventPolicy{Emittable: true}
	}
	return EventPolicy{}
}
