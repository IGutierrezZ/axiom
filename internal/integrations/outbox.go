package integrations

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// OutboxSchemaV1 es el esquema de la cola local de operaciones.
const OutboxSchemaV1 = "axiom.integrations-outbox/v1"

// outboxName es el nombre de la cola. El literal vive en kickoff (D-18): no hay un segundo.
const outboxName = kickoff.IntegrationsOutboxFile

// ReasonPluginNotConfigured es el motivo de una operación cuyo plugin ya no está configurado,
// activo o registrado: queda pendiente y no cuenta un intento.
const ReasonPluginNotConfigured = "plugin_not_configured"

// OperationState es el estado de una operación en la cola.
type OperationState string

const (
	StatePending   OperationState = "pending"
	StateAbandoned OperationState = "abandoned"
)

// Operation es una entrega pendiente: el par (plugin, clave) identifica la operación lógica. Nunca
// guarda secretos ni configuración del plugin: se leen de axiom.yaml y del almacén en cada intento.
type Operation struct {
	Key           string         `yaml:"key"`
	Plugin        string         `yaml:"plugin"`
	State         OperationState `yaml:"state"`
	Attempts      int            `yaml:"attempts"`
	EnqueuedAt    time.Time      `yaml:"enqueued_at"`
	LastAttemptAt time.Time      `yaml:"last_attempt_at,omitempty"`
	LastError     string         `yaml:"last_error,omitempty"` // saneado, ≤ 512 caracteres, una línea
	Reason        string         `yaml:"reason,omitempty"`
	Event         Event          `yaml:"event"`
}

// Outbox es el contenido de .integrations-outbox.yaml.
type Outbox struct {
	Schema     string      `yaml:"schema"`
	Change     string      `yaml:"change"`
	Operations []Operation `yaml:"operations"`
}

// readOutbox lee la cola de la carpeta con lectura estricta. Sin fichero devuelve una cola vacía.
func readOutbox(f Folder) (Outbox, error) {
	path := f.path(outboxName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Outbox{Schema: OutboxSchemaV1, Change: f.Change()}, nil
		}
		return Outbox{}, fmt.Errorf("leer %s en %q: %w", outboxName, path, err)
	}
	ob, err := parseOutbox(data, f.Change())
	if err != nil {
		return Outbox{}, fmt.Errorf("%s inválido en %q: %w", outboxName, path, err)
	}
	return ob, nil
}

func parseOutbox(data []byte, change string) (Outbox, error) {
	var ob Outbox
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&ob); err != nil {
		if errors.Is(err, io.EOF) {
			return Outbox{}, errors.New("el fichero está vacío")
		}
		return Outbox{}, err
	}
	if ob.Schema != OutboxSchemaV1 {
		return Outbox{}, fmt.Errorf("esquema %q no admitido: debe ser %q", ob.Schema, OutboxSchemaV1)
	}
	if ob.Change != change {
		return Outbox{}, fmt.Errorf("el campo change es %q y la carpeta es la del cambio %q", ob.Change, change)
	}
	seen := map[string]bool{}
	for i, op := range ob.Operations {
		if err := validateOperation(op); err != nil {
			return Outbox{}, fmt.Errorf("operación %d: %w", i+1, err)
		}
		id := op.Plugin + "\x00" + op.Key
		if seen[id] {
			return Outbox{}, fmt.Errorf("operación duplicada (plugin %s, clave %s)", op.Plugin, op.Key)
		}
		seen[id] = true
	}
	return ob, nil
}

func validateOperation(op Operation) error {
	switch {
	case strings.TrimSpace(op.Key) == "":
		return errors.New("la clave (key) no puede estar vacía")
	case strings.TrimSpace(op.Plugin) == "":
		return errors.New("el plugin no puede estar vacío")
	case op.State != StatePending && op.State != StateAbandoned:
		return fmt.Errorf("estado %q no admitido; admitidos: %s, %s", op.State, StatePending, StateAbandoned)
	case op.Attempts < 0:
		return errors.New("attempts no puede ser negativo")
	case !manifest.ValidEvent(op.Event.Type):
		return fmt.Errorf("evento %q fuera del catálogo; admitidos: %s", op.Event.Type, manifest.EventNames())
	}
	return nil
}

// find devuelve la operación (plugin, clave) de la cola, si está.
func (o *Outbox) find(plugin, key string) *Operation {
	for i := range o.Operations {
		if o.Operations[i].Plugin == plugin && o.Operations[i].Key == key {
			return &o.Operations[i]
		}
	}
	return nil
}

// enqueue encola la operación (plugin, clave del evento) si no hay ya una con ese par. Devuelve la
// operación y si se creó. Mientras una operación sigue en la cola (pendiente o abandonada) no se
// duplica; tras entregarse y retirarse, la misma clave puede volver a encolarse.
func (o *Outbox) enqueue(plugin string, ev Event, now time.Time) (*Operation, bool) {
	if existing := o.find(plugin, ev.IdempotencyKey); existing != nil {
		return existing, false
	}
	o.Operations = append(o.Operations, Operation{
		Key:        ev.IdempotencyKey,
		Plugin:     plugin,
		State:      StatePending,
		EnqueuedAt: now.UTC(),
		Event:      ev,
	})
	return &o.Operations[len(o.Operations)-1], true
}

// remove retira la operación (plugin, clave).
func (o *Outbox) remove(plugin, key string) {
	kept := o.Operations[:0]
	for _, op := range o.Operations {
		if op.Plugin == plugin && op.Key == key {
			continue
		}
		kept = append(kept, op)
	}
	o.Operations = kept
}

// writeOutbox escribe la cola de forma atómica, o borra el fichero si no queda ninguna operación
// (así el árbol solo está sucio mientras hay algo pendiente o abandonado).
func writeOutbox(f Folder, o Outbox) error {
	if len(o.Operations) == 0 {
		return f.remove(outboxName)
	}
	o.Schema = OutboxSchemaV1
	if o.Change == "" {
		o.Change = f.Change()
	}
	payload, err := yaml.Marshal(o)
	if err != nil {
		return fmt.Errorf("serializar %s: %w", outboxName, err)
	}
	return f.writeAtomic(outboxName, outboxTempPattern, payload)
}
