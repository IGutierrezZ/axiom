package reviewtransaction

import (
	"context"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
)

// RDD suspendido (C-01-c): sin la etiqueta rdd_harness las preferencias "on"
// guardadas, en cualquier alcance, no se aplican.
func TestResolveRDDModeIsOffWithStoredOnPreferencesWhileSuspended(t *testing.T) {
	restore := rddharness.SetForTesting(false)
	t.Cleanup(restore)
	if rddharness.Enabled() {
		t.Skip("binario compilado con rdd_harness: el apagado no aplica")
	}
	ctx := context.Background()
	repo := initSnapshotRepo(t)

	cases := map[string]RDDGlobalMode{
		"global":  {Value: string(RDDModeOn)},
		"project": {Project: string(RDDModeOn)},
		"ambos":   {Value: string(RDDModeOn), Project: string(RDDModeOn)},
	}
	for name, global := range cases {
		for label, resolve := range map[string]func(context.Context, string, RDDGlobalMode) (RDDModeStatus, error){
			"ResolveRDDMode":       ResolveRDDMode,
			"ResolveRDDModeStrict": ResolveRDDModeStrict,
		} {
			status, err := resolve(ctx, repo, global)
			if err != nil {
				t.Fatalf("%s/%s: error = %v", name, label, err)
			}
			if status.Enabled() || status.Effective != RDDModeOff || status.Source != RDDModeSourceDefault || !status.Resolved() {
				t.Fatalf("%s/%s: estado = %#v; quiere off/default resuelto", name, label, status)
			}
		}
	}
}

// El resolutor estricto no lee la anulación del clon mientras RDD está
// suspendido: una ubicación ilegible no lo hace fallar.
func TestResolveRDDModeStrictDoesNotReadTheCloneOverrideWhileSuspended(t *testing.T) {
	restore := rddharness.SetForTesting(false)
	t.Cleanup(restore)
	if rddharness.Enabled() {
		t.Skip("binario compilado con rdd_harness: el apagado no aplica")
	}
	notARepo := t.TempDir()
	status, err := ResolveRDDModeStrict(context.Background(), notARepo, RDDGlobalMode{Value: string(RDDModeOn)})
	if err != nil || status.Effective != RDDModeOff || status.Source != RDDModeSourceDefault {
		t.Fatalf("estado = %#v, error = %v; quiere off/default sin error", status, err)
	}
}

// Con el punto de inyección activo las preferencias "on" se siguen respetando.
func TestResolveRDDModeHonoursStoredOnPreferencesWithTheHarness(t *testing.T) {
	restore := rddharness.SetForTesting(true)
	t.Cleanup(restore)
	repo := initSnapshotRepo(t)
	status, err := ResolveRDDMode(context.Background(), repo, RDDGlobalMode{Value: string(RDDModeOn)})
	if err != nil || !status.Enabled() || status.Source != RDDModeSourceGlobal {
		t.Fatalf("estado = %#v, error = %v; quiere on/global", status, err)
	}
}

func TestResolveRDDModeCancelledContextStillFailsClosedWhileSuspended(t *testing.T) {
	restore := rddharness.SetForTesting(false)
	t.Cleanup(restore)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	status, err := ResolveRDDMode(cancelled, t.TempDir(), RDDGlobalMode{})
	if err == nil || status.Resolved() {
		t.Fatalf("estado = %#v, error = %v; quiere error y proyección sin resolver", status, err)
	}
}
