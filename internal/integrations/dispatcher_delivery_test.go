package integrations

import (
	"context"
	"strings"
	"testing"
)

func TestDispatchPropagatesSanitizedOutcomeWarnings(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.warnings = []string{"task_state_kept: la Task sigue en Tech Review", "  assignment_skipped:\n sin   identidad "}

	report := d.Dispatch(context.Background(), f.Root, specEvent("k-warn"), durable)

	if report.Err != nil || len(report.Delivered) != 1 {
		t.Fatalf("una entrega con avisos sigue siendo una entrega: %+v", report)
	}
	want := []string{"task_state_kept: la Task sigue en Tech Review", "assignment_skipped: sin identidad"}
	if strings.Join(report.Warnings, "|") != strings.Join(want, "|") {
		t.Fatalf("avisos = %q, se esperaba %q", report.Warnings, want)
	}
}

type identityLookupFunc func(plugin, account string) (TrackerIdentity, bool, error)

func (f identityLookupFunc) Lookup(plugin, account string) (TrackerIdentity, bool, error) {
	return f(plugin, account)
}

func TestDispatchPassesDelivererAndIdentitiesToThePlugin(t *testing.T) {
	f, plugin, d := setup(t)
	d.Deliverer = "Ana <ana@example.test>"
	d.Identities = identityLookupFunc(func(_, account string) (TrackerIdentity, bool, error) {
		return TrackerIdentity{Account: account, ID: "guid-1"}, true, nil
	})

	d.Dispatch(context.Background(), f.Root, specEvent("k-id"), durable)

	if len(plugin.received) != 1 {
		t.Fatalf("entregas = %d", len(plugin.received))
	}
	got := plugin.received[0]
	if got.Deliverer != "Ana <ana@example.test>" || got.Identities == nil {
		t.Fatalf("Deliverer=%q Identities=%v", got.Deliverer, got.Identities)
	}
	if id, ok, err := got.Identities.Lookup("fake", "org"); err != nil || !ok || id.ID != "guid-1" {
		t.Fatalf("Lookup = %+v %v %v", id, ok, err)
	}

	// Sin configurar nada, el plugin no recibe identidades.
	d.Deliverer, d.Identities = "", nil
	d.Dispatch(context.Background(), f.Root, specEvent("k-id2"), durable)
	if last := plugin.received[len(plugin.received)-1]; last.Deliverer != "" || last.Identities != nil {
		t.Fatalf("sin identidades: %+v", last)
	}
}
