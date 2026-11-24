package reqid_test

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestSynthetic(t *testing.T) {
	if got := reqid.Synthetic("facturacion", 3); got != "REQ-facturacion-3" {
		t.Errorf("Synthetic = %q", got)
	}
	if got := reqid.Synthetic("USU", 1); got != "req-USU-1" {
		t.Errorf("Synthetic = %q", got)
	}
	if got := reqid.Synthetic("sdd-iso", 12); got != "REQ-sdd-iso-12" {
		t.Errorf("Synthetic = %q", got)
	}
}
