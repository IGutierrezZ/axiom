package integrations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const (
	// MaxAttempts es el máximo de intentos de una operación (incluido el inmediato): al alcanzarlo
	// pasa a abandonada y queda en la cola, visible (D-10).
	MaxAttempts = 10
	// DefaultImmediate es el plazo total del intento inmediato de un despacho.
	DefaultImmediate = 10 * time.Second
	// DefaultPerOp es el plazo por operación de sync.
	DefaultPerOp = 30 * time.Second
)

// Estados de una operación en un informe.
const (
	ReportDelivered = "delivered"
	ReportPending   = "pending"
	ReportAbandoned = "abandoned"
	ReportDiscarded = "discarded"
)

// DispatchOptions ajusta un despacho.
type DispatchOptions struct {
	Durable        bool // encola antes de intentar (todos los eventos, incluido archive)
	PersistOutcome bool // fusiona vínculos en integrations.yaml (falso para archive)
}

// Dispatcher entrega los eventos del ciclo de vida a los plugins configurados.
type Dispatcher struct {
	Registry *Registry
	Config   *workspace.IntegrationsConfig
	Secrets  secrets.Store
	// WorkspaceRoot es el padre del planning home: la guarda de inmutabilidad de las carpetas
	// archivadas (D-18) resuelve la carpeta del cambio respecto a él.
	WorkspaceRoot string
	Now           func() time.Time
	Immediate     time.Duration // 10 s si es cero
	PerOp         time.Duration // 30 s si es cero (sync)
	// Guard, si no es nil, se llama con la carpeta antes de tomar su cerrojo: un error la rechaza
	// (la guarda A de ramas, RF-BRA-20, la pone la capa de verbos). Dispatch lo deja en Report.Err y
	// Sync salta la carpeta con un aviso.
	Guard func(Folder) error
	// Deliverer es la identidad git de quien entrega; Identities consulta las identidades guardadas
	// del tracker. Ambos viajan en cada Delivery y solo se leen (nil/vacío: sin identidad).
	Deliverer  string
	Identities IdentityLookup
}

// OperationReport describe una operación en un informe. Nunca lleva secretos: LastError ya está
// saneado.
type OperationReport struct {
	Key       string `json:"key"`
	Plugin    string `json:"plugin"`
	State     string `json:"state"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"last_error,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// Report es el resultado de un despacho.
type Report struct {
	Change      string
	Folder      string
	QueuePath   string // ruta de la cola si queda algo en ella
	Delivered   []OperationReport
	Pending     []OperationReport
	Abandoned   []OperationReport
	Warnings    []string
	NothingToDo bool  // ningún plugin activo escucha el evento: no se ha leído ni escrito nada
	Err         error // fallo que impide despachar (cerrojo, cola inválida, evento inválido...)
}

// Undelivered devuelve las operaciones que no se entregaron: pendientes y abandonadas.
func (r Report) Undelivered() []OperationReport {
	return append(append([]OperationReport(nil), r.Pending...), r.Abandoned...)
}

func (d *Dispatcher) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d *Dispatcher) immediate() time.Duration {
	if d.Immediate > 0 {
		return d.Immediate
	}
	return DefaultImmediate
}

func (d *Dispatcher) perOp() time.Duration {
	if d.PerOp > 0 {
		return d.PerOp
	}
	return DefaultPerOp
}

// listening devuelve los plugins activos del bloque cuyo manifiesto del catálogo escucha t.
func (d *Dispatcher) listening(t EventType) []workspace.IntegrationPluginConfig {
	var out []workspace.IntegrationPluginConfig
	for _, p := range d.Config.Active() {
		m, ok := manifest.Lookup(p.ID)
		if !ok {
			continue
		}
		for _, e := range m.Events {
			if e == t {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// HasListeners indica si algún plugin activo del bloque escucha t según el catálogo cerrado. Es la
// comprobación barata que permite a un verbo no leer ni escribir nada (ni construir el evento)
// cuando nadie escucha.
func HasListeners(config *workspace.IntegrationsConfig, t EventType) bool {
	return (&Dispatcher{Config: config}).listening(t) != nil
}

// activeConfig devuelve la configuración activa del plugin id.
func (d *Dispatcher) activeConfig(id string) (workspace.IntegrationPluginConfig, bool) {
	for _, p := range d.Config.Active() {
		if p.ID == id {
			return p, true
		}
	}
	return workspace.IntegrationPluginConfig{}, false
}

func opReport(op Operation, state string) OperationReport {
	return OperationReport{Key: op.Key, Plugin: op.Plugin, State: state, Attempts: op.Attempts, LastError: op.LastError, Reason: op.Reason}
}

// Dispatch entrega ev a los plugins activos que lo escuchan. Sin ninguno, vuelve sin leer ni
// escribir disco. Con alguno: toma el cerrojo del cambio, encola (plugin, clave) si no está ya,
// intenta la entrega de inmediato con un plazo total compartido y retira lo entregado. Nunca
// entra en pánico ni devuelve error: lo que impide despachar queda en Report.Err.
func (d *Dispatcher) Dispatch(ctx context.Context, changeRoot string, ev Event, opts DispatchOptions) Report {
	folder := Folder{WorkspaceRoot: d.WorkspaceRoot, Root: changeRoot}
	report := Report{Change: folder.Change(), Folder: changeRoot}

	if !manifest.ValidEvent(ev.Type) {
		report.Err = &UnknownEventError{Name: string(ev.Type)}
		return report
	}
	if ev.IdempotencyKey == "" {
		report.Err = errors.New("el evento no lleva clave de idempotencia")
		return report
	}
	listening := d.listening(ev.Type)
	if len(listening) == 0 {
		report.NothingToDo = true
		return report
	}
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = d.now()
	}
	if d.Guard != nil {
		if err := d.Guard(folder); err != nil {
			report.Err = err
			return report
		}
	}

	release, err := folder.acquire()
	if err != nil {
		report.Err = err
		return report
	}
	defer release()

	outbox := Outbox{Schema: OutboxSchemaV1, Change: folder.Change()}
	if opts.Durable {
		if outbox, err = readOutbox(folder); err != nil {
			report.Err = err
			return report
		}
	}
	created := false
	for _, p := range listening {
		if _, isNew := outbox.enqueue(p.ID, ev, d.now()); isNew {
			created = true
		}
	}
	if opts.Durable && created {
		if err := writeOutbox(folder, outbox); err != nil {
			report.Err = fmt.Errorf("encolar la entrega: %w", err)
			return report
		}
	}

	deliverCtx, cancel := context.WithTimeout(ctx, d.immediate())
	defer cancel()

	var delivered []string
	var ids []string // plugins del evento, en orden de declaración
	for _, p := range listening {
		ids = append(ids, p.ID)
	}
	for _, id := range ids {
		op := outbox.find(id, ev.IdempotencyKey)
		if op == nil {
			continue
		}
		if op.State == StateAbandoned {
			continue // visible, pero solo sync --retry-abandoned la reactiva
		}
		persist := opts.PersistOutcome && PolicyFor(op.Event.Type).PersistOutcome && !folder.Archived()
		if d.attempt(deliverCtx, folder, op, persist, &report.Warnings) {
			delivered = append(delivered, id)
		}
	}

	for _, id := range ids {
		op := outbox.find(id, ev.IdempotencyKey)
		if op == nil {
			continue
		}
		switch {
		case containsString(delivered, id):
			report.Delivered = append(report.Delivered, opReport(*op, ReportDelivered))
		case op.State == StateAbandoned:
			report.Abandoned = append(report.Abandoned, opReport(*op, ReportAbandoned))
		default:
			report.Pending = append(report.Pending, opReport(*op, ReportPending))
		}
	}
	for _, id := range delivered {
		outbox.remove(id, ev.IdempotencyKey)
	}

	if opts.Durable {
		if err := writeOutbox(folder, outbox); err != nil {
			report.Err = fmt.Errorf("actualizar la cola tras la entrega: %w", err)
		} else if len(outbox.Operations) > 0 {
			report.QueuePath = folder.path(outboxName)
		}
	}
	return report
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

type callResult struct {
	outcome Outcome
	err     error
}

// call ejecuta Deliver en su propia gorutina: recupera un pánico como fallo transitorio y deja de
// esperar al vencer ctx aunque el plugin lo ignore. El canal tiene hueco para el resultado tardío,
// que nadie recoge: el plugin ya no puede escribir nada en la cola.
func call(ctx context.Context, plugin Plugin, delivery Delivery) callResult {
	done := make(chan callResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- callResult{err: fmt.Errorf("el plugin entró en pánico: %v", r)}
			}
		}()
		outcome, err := plugin.Deliver(ctx, delivery)
		done <- callResult{outcome: outcome, err: err}
	}()
	select {
	case result := <-done:
		return result
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return callResult{err: fmt.Errorf("tiempo agotado esperando al plugin: %w", ctx.Err())}
		}
		return callResult{err: fmt.Errorf("entrega cancelada: %w", ctx.Err())}
	}
}

// attempt intenta entregar una operación y actualiza su contabilidad. Devuelve true si se entregó
// (y los vínculos, si toca, ya están en integrations.yaml): quien llama la retira de la cola. Un
// plugin que ya no está configurado, activo o registrado deja la operación pendiente con el motivo
// plugin_not_configured y sin contar un intento.
func (d *Dispatcher) attempt(ctx context.Context, folder Folder, op *Operation, persist bool, warnings *[]string) bool {
	cfg, configured := d.activeConfig(op.Plugin)
	var plugin Plugin
	registered := false
	if d.Registry != nil {
		plugin, registered = d.Registry.Get(op.Plugin)
	}
	if !configured || !registered {
		op.Reason = ReasonPluginNotConfigured
		return false
	}
	op.Reason = ""

	tracker := newSecretTracker()
	fail := func(err error) bool {
		op.Attempts++
		op.LastAttemptAt = d.now()
		op.LastError = sanitizeError(err, tracker)
		if IsPermanent(err) || op.Attempts >= MaxAttempts {
			op.State = StateAbandoned
		}
		return false
	}

	links, err := readLinks(folder)
	if err != nil {
		return fail(err)
	}
	values := make(map[string]any, len(cfg.Values))
	for k, v := range cfg.Values {
		values[k] = v
	}
	delivery := Delivery{
		Event:   op.Event,
		Config:  values,
		Links:   append([]Link(nil), links.Links...),
		Secrets: newScopedSecrets(plugin.Manifest(), d.Secrets, tracker),

		Deliverer:  d.Deliverer,
		Identities: d.Identities,
	}

	result := call(ctx, plugin, delivery)
	if result.err != nil {
		return fail(result.err)
	}
	if persist {
		if _, err := mergeLinks(folder, op.Plugin, op.Key, d.now(), result.outcome); err != nil {
			return fail(err)
		}
	} else if n := len(result.outcome.Links) + len(result.outcome.TimeEntries); n > 0 {
		*warnings = append(*warnings, fmt.Sprintf("se descartan %d vínculo(s) o imputación(es) devueltos por %s: este evento no escribe %s", n, op.Event.Type, LinksFile))
	}
	// Los avisos solo cuentan si la entrega se completó (también la fusión de vínculos).
	for _, w := range result.outcome.Warnings {
		if text := sanitizeText(w, tracker); text != "" {
			*warnings = append(*warnings, text)
		}
	}
	return true
}
