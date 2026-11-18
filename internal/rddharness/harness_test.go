package rddharness

import "testing"

func TestSetForTestingRestoresPreviousValue(t *testing.T) {
	restoreOn := SetForTesting(true)
	if !Enabled() {
		t.Fatal("el punto de inyección activo debe habilitar RDD")
	}
	restoreOff := SetForTesting(false)
	if compiled != Enabled() {
		t.Fatalf("Enabled() = %v con la inyección apagada, quiere %v", Enabled(), compiled)
	}
	restoreOff()
	if !Enabled() {
		t.Fatal("restore debe devolver el valor anterior (activo)")
	}
	restoreOn()
}
