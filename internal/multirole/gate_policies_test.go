package multirole

import (
	"slices"
	"testing"
)

// RF-TOP-21: GatePolicies expone el vocabulario cerrado de `gate_policy` y devuelve
// una copia nueva por llamada.
func TestGatePoliciesReturnsAFreshCopy(t *testing.T) {
	got := GatePolicies()
	if want := []GatePolicy{"blocking", "deferred", "optional"}; !slices.Equal(got, want) {
		t.Fatalf("GatePolicies() = %v, quiero %v", got, want)
	}
	got[0] = "zzz"
	if GatePolicies()[0] != PolicyBlocking {
		t.Fatal("GatePolicies debe devolver una copia nueva por llamada")
	}
}
