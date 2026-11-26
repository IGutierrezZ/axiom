package integrations

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const viewOutboxYAML = `schema: axiom.integrations-outbox/v1
change: x
operations:
  - key: CLAVE-SECRETA/b
    plugin: fake
    state: pending
    attempts: 3
    enqueued_at: 2026-11-18T22:40:00Z
    last_attempt_at: 2026-11-18T22:41:00Z
    last_error: ERROR-CRUDO con token
    reason: plugin_not_configured
    event:
      type: gate.spec.approved
      change: x
      gate: spec
      actor: ACTOR-PRIVADO <a@example.test>
      occurred_at: 2026-11-18T22:40:00Z
      artifact_digest: DIGEST-PRIVADO
      idempotency_key: CLAVE-SECRETA/b
  - key: CLAVE-SECRETA/a
    plugin: fake
    state: abandoned
    attempts: 1
    enqueued_at: 2026-11-18T22:30:00Z
    event:
      type: role.apply.approved
      change: x
      role: web
      gate: role-apply:web
      actor: cli
      occurred_at: 2026-11-18T22:30:00Z
      idempotency_key: CLAVE-SECRETA/a
`

func TestOutboxViewOrdersByEnqueuedAt(t *testing.T) {
	fx := newWorkspace(t)
	writeTestFile(t, filepath.Join(fx.Root, outboxName), viewOutboxYAML)
	views, err := OutboxView(fx.Folder())
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("vistas = %d, quiero 2", len(views))
	}
	if views[0].Role != "web" || views[0].State != StateAbandoned || views[1].Reason != "plugin_not_configured" {
		t.Errorf("orden o campos inesperados: %+v", views)
	}
	if views[1].Attempts != 3 || !views[1].LastAttemptAt.Equal(time.Date(2026, 11, 18, 22, 41, 0, 0, time.UTC)) {
		t.Errorf("intentos o último intento inesperados: %+v", views[1])
	}
}

func TestOutboxViewNeverCarriesKeyErrorActorOrDigest(t *testing.T) {
	fx := newWorkspace(t)
	writeTestFile(t, filepath.Join(fx.Root, outboxName), viewOutboxYAML)
	views, err := OutboxView(fx.Folder())
	if err != nil {
		t.Fatal(err)
	}
	dump := strings.Join(flattenView(views), "|")
	for _, secret := range []string{"CLAVE-SECRETA", "ERROR-CRUDO", "ACTOR-PRIVADO", "DIGEST-PRIVADO", "token"} {
		if strings.Contains(dump, secret) {
			t.Errorf("la vista filtra %q: %s", secret, dump)
		}
	}
	typ := reflect.TypeOf(OperationView{})
	for _, banned := range []string{"Key", "LastError", "Actor", "ArtifactDigest", "Fingerprint", "IdempotencyKey"} {
		if _, found := typ.FieldByName(banned); found {
			t.Errorf("OperationView no puede tener el campo %s", banned)
		}
	}
}

func TestOutboxViewWithoutFileIsEmpty(t *testing.T) {
	fx := newWorkspace(t)
	views, err := OutboxView(fx.Folder())
	if err != nil || views == nil || len(views) != 0 {
		t.Fatalf("OutboxView sin fichero = %v, %v; quiero lista vacía no nula y sin error", views, err)
	}
}

func TestOutboxViewReportsACorruptFile(t *testing.T) {
	fx := newWorkspace(t)
	writeTestFile(t, filepath.Join(fx.Root, outboxName), "schema: otro\n")
	if _, err := OutboxView(fx.Folder()); err == nil {
		t.Fatal("un fichero corrupto debe devolver error")
	}
}

func TestOutboxViewReadsAnArchivedFolder(t *testing.T) {
	fx := newWorkspace(t)
	folder := archivedFolderOf(t, fx, "2026-11-01-x")
	if err := os.WriteFile(filepath.Join(folder.Root, outboxName), []byte(viewOutboxYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	views, err := OutboxView(folder)
	if err != nil || len(views) != 2 {
		t.Fatalf("archivada: %v, %v", views, err)
	}
}

func flattenView(views []OperationView) []string {
	var out []string
	for _, v := range views {
		out = append(out, v.Plugin, string(v.Event), v.Role, v.Gate, string(v.State), v.Reason)
	}
	return out
}
