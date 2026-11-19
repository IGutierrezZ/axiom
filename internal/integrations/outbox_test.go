package integrations

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

const validOutboxYAML = `schema: axiom.integrations-outbox/v1
change: x
operations:
  - key: gate.spec.approved/x/d0123456789abcdef
    plugin: fake
    state: pending
    attempts: 2
    enqueued_at: 2026-11-18T22:30:00Z
    last_attempt_at: 2026-11-18T22:31:00Z
    last_error: timeout del tracker
    event:
      type: gate.spec.approved
      change: x
      gate: spec
      actor: Nombre <correo@example.test>
      occurred_at: 2026-11-18T22:30:00Z
      artifact_digest: abc
      fingerprint: d0123456789abcdef
      idempotency_key: gate.spec.approved/x/d0123456789abcdef
`

func sampleEvent(key string) Event {
	return Event{Type: manifest.EventGateSpecApproved, Change: "x", Gate: "spec", Actor: "cli", OccurredAt: fixedNow, IdempotencyKey: key}
}

func TestReadOutboxAbsentFileIsEmpty(t *testing.T) {
	f := newWorkspace(t)
	ob, err := readOutbox(f.Folder())
	if err != nil {
		t.Fatalf("readOutbox sin fichero: %v", err)
	}
	if ob.Schema != OutboxSchemaV1 || ob.Change != "x" || len(ob.Operations) != 0 {
		t.Fatalf("outbox = %+v", ob)
	}
	if OutboxSchemaV1 != "axiom.integrations-outbox/v1" {
		t.Fatalf("OutboxSchemaV1 = %q", OutboxSchemaV1)
	}
}

func TestReadOutboxParsesTheExactSchema(t *testing.T) {
	f := newWorkspace(t)
	writeTestFile(t, f.Folder().path(outboxName), validOutboxYAML)
	ob, err := readOutbox(f.Folder())
	if err != nil {
		t.Fatalf("readOutbox: %v", err)
	}
	if len(ob.Operations) != 1 {
		t.Fatalf("operaciones = %+v", ob.Operations)
	}
	op := ob.Operations[0]
	if op.State != StatePending || op.Attempts != 2 || op.LastError != "timeout del tracker" || op.Event.Gate != "spec" || op.Event.Actor != "Nombre <correo@example.test>" {
		t.Fatalf("operación = %+v", op)
	}
}

func TestReadOutboxIsStrict(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		wants string
	}{
		{"esquema distinto", strings.Replace(validOutboxYAML, "outbox/v1", "outbox/v2", 1), "esquema"},
		{"clave desconocida en la raíz", validOutboxYAML + "config: {a: b}\n", "config"},
		{"clave desconocida en la operación", strings.Replace(validOutboxYAML, "    attempts: 2\n", "    attempts: 2\n    secret: abc\n", 1), "secret"},
		{"estado desconocido", strings.Replace(validOutboxYAML, "state: pending", "state: delivered", 1), "estado"},
		{"clave vacía", strings.Replace(validOutboxYAML, "  - key: gate.spec.approved/x/d0123456789abcdef", `  - key: ""`, 1), "key"},
		{"plugin vacío", strings.Replace(validOutboxYAML, "plugin: fake", `plugin: ""`, 1), "plugin"},
		{"intentos negativos", strings.Replace(validOutboxYAML, "attempts: 2", "attempts: -1", 1), "attempts"},
		{"evento fuera del catálogo", strings.Replace(validOutboxYAML, "type: gate.spec.approved", "type: gate.tasks.approved", 1), "evento"},
		{"cambio distinto del de la carpeta", strings.Replace(validOutboxYAML, "change: x\noperations", "change: otro\noperations", 1), "change"},
		{"fichero vacío", "", "vacío"},
		{"YAML roto", "schema: [sin cerrar\n", ".integrations-outbox.yaml"},
		{"operación duplicada", validOutboxYAML + strings.SplitN(validOutboxYAML, "operations:\n", 2)[1], "duplicada"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newWorkspace(t)
			writeTestFile(t, f.Folder().path(outboxName), tt.yaml)
			_, err := readOutbox(f.Folder())
			if err == nil || !strings.Contains(err.Error(), tt.wants) {
				t.Fatalf("se esperaba un error con %q, obtuvo %v", tt.wants, err)
			}
		})
	}
}

func TestEnqueueDeduplicatesByPluginAndKeyWhileQueued(t *testing.T) {
	ob := Outbox{Schema: OutboxSchemaV1, Change: "x"}

	first, created := ob.enqueue("fake", sampleEvent("k1"), fixedNow)
	if !created || first.State != StatePending || first.Attempts != 0 || !first.EnqueuedAt.Equal(fixedNow) || first.Key != "k1" || first.Plugin != "fake" {
		t.Fatalf("primer encolado: created=%v op=%+v", created, first)
	}
	again, created := ob.enqueue("fake", sampleEvent("k1"), fixedNow.Add(time.Hour))
	if created || again != first {
		t.Fatalf("la misma (plugin, clave) no se duplica: created=%v", created)
	}
	if _, created := ob.enqueue("otro", sampleEvent("k1"), fixedNow); !created {
		t.Fatal("otro plugin con la misma clave es otra operación")
	}
	if _, created := ob.enqueue("fake", sampleEvent("k2"), fixedNow); !created {
		t.Fatal("otra clave es otra operación")
	}
	if len(ob.Operations) != 3 || ob.Operations[0].Key != "k1" || ob.Operations[1].Plugin != "otro" || ob.Operations[2].Key != "k2" {
		t.Fatalf("orden de encolado: %+v", ob.Operations)
	}

	// Una operación abandonada sigue en la cola: no se duplica tampoco.
	ob.Operations[0].State = StateAbandoned
	if _, created := ob.enqueue("fake", sampleEvent("k1"), fixedNow); created {
		t.Fatal("una abandonada sigue ocupando su sitio hasta sync --retry-abandoned o --discard")
	}

	// Tras retirarla (entregada), la misma clave puede volver a encolarse.
	ob.remove("fake", "k1")
	if _, created := ob.enqueue("fake", sampleEvent("k1"), fixedNow); !created {
		t.Fatal("entregada y retirada, la misma clave vuelve a poder encolarse")
	}
}

func TestWriteOutboxRoundTripsAndDeletesTheFileWhenEmpty(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	ob := Outbox{Schema: OutboxSchemaV1, Change: "x"}
	ob.enqueue("fake", sampleEvent("k1"), fixedNow)

	if err := writeOutbox(folder, ob); err != nil {
		t.Fatalf("writeOutbox: %v", err)
	}
	content := readTestFile(t, folder.path(outboxName))
	if !strings.HasPrefix(content, "schema: axiom.integrations-outbox/v1\nchange: x\noperations:\n") {
		t.Fatalf("cabecera inesperada:\n%s", content)
	}
	back, err := readOutbox(folder)
	if err != nil || len(back.Operations) != 1 || back.Operations[0].Key != "k1" {
		t.Fatalf("la ida y vuelta conserva la operación: %+v %v", back, err)
	}
	if err := writeOutbox(folder, back); err != nil {
		t.Fatal(err)
	}
	if again := readTestFile(t, folder.path(outboxName)); again != content {
		t.Fatalf("reescribir lo leído da el mismo fichero:\n%s\n---\n%s", content, again)
	}

	back.remove("fake", "k1")
	if err := writeOutbox(folder, back); err != nil {
		t.Fatal(err)
	}
	if fileExists(folder.path(outboxName)) {
		t.Fatal("al quedar sin operaciones el fichero se borra")
	}
	if err := writeOutbox(folder, back); err != nil {
		t.Fatalf("borrar lo que no existe no es un error: %v", err)
	}
}

func TestOutboxNeverStoresSecretsNorPluginConfig(t *testing.T) {
	ob := Outbox{Schema: OutboxSchemaV1, Change: "x"}
	op, _ := ob.enqueue("fake", sampleEvent("k1"), fixedNow)
	op.LastError = "fallo"
	op.Attempts = 1
	op.LastAttemptAt = fixedNow
	data, err := yaml.Marshal(ob)
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := yaml.Unmarshal(data, &generic); err != nil {
		t.Fatal(err)
	}
	operation := generic["operations"].([]any)[0].(map[string]any)
	allowed := map[string]bool{"key": true, "plugin": true, "state": true, "attempts": true, "enqueued_at": true, "last_attempt_at": true, "last_error": true, "reason": true, "event": true}
	for k := range operation {
		if !allowed[k] {
			t.Errorf("la operación guarda la clave inesperada %q", k)
		}
	}
	for _, forbidden := range []string{"config", "secret", "token", "values"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("la cola no guarda %q:\n%s", forbidden, data)
		}
	}
}

func TestConcurrentDispatchesWithDifferentKeysKeepBothOperations(t *testing.T) {
	f := newGitWorkspace(t)
	fake := newTestPlugin()
	fake.setFailAlways(errors.New("tracker caído"))
	installTestPlugin(t, fake)
	d := newTestDispatcher(f, "fake")

	var wg sync.WaitGroup
	for _, key := range []string{"gate.spec.approved/x/da", "gate.spec.approved/x/db", "gate.design.approved/x/dc", "role.start/x/backend"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ev := sampleEvent(key)
			if strings.HasPrefix(key, "gate.design") {
				ev.Type = manifest.EventGateDesignApproved
			}
			if strings.HasPrefix(key, "role.start") {
				ev.Type = manifest.EventRoleStart
				ev.Role = "backend"
			}
			d.Dispatch(context.Background(), f.Root, ev, DispatchOptions{Durable: true, PersistOutcome: true})
		}()
	}
	wg.Wait()

	ob, err := readOutbox(f.Folder())
	if err != nil {
		t.Fatalf("el YAML de la cola debe seguir siendo válido: %v", err)
	}
	if len(ob.Operations) != 4 {
		t.Fatalf("las cuatro operaciones deben sobrevivir: %+v", ob.Operations)
	}
}
