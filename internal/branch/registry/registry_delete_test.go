package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// design D-09: `done` borra el registro de un cambio cuando ya no le queda ninguna rama. Borrarlo
// deja el resto de la carpeta como estaba (el bloqueo del cambio y los registros de los demás
// cambios y carriles) y es idempotente: un registro que no existe no es un error.
func TestDeleteRemovesOnlyTheRecordOfTheChangeAndIsIdempotent(t *testing.T) {
	root, commonDir := newRepository(t)
	store := New()
	for _, change := range []string{"x", "y"} {
		if err := store.Save(commonDir, recordOf(root, commonDir, branch.LaneSDD, change)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Save(commonDir, recordOf(root, commonDir, branch.LaneODD, "x")); err != nil {
		t.Fatal(err)
	}
	unlock, err := store.Lock(t.Context(), commonDir, branch.LaneSDD, "x")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	if err := store.Delete(commonDir, branch.LaneSDD, "x"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if record, err := load(t, commonDir, branch.LaneSDD, "x"); err != nil || record != nil {
		t.Errorf("el registro de x debía desaparecer: %+v, %v", record, err)
	}
	for path, what := range map[string]string{
		recordFile(commonDir, "sdd", "y"):                                        "el registro de otro cambio",
		recordFile(commonDir, "odd", "x"):                                        "el registro del mismo cambio en otro carril",
		filepath.Join(filepath.Dir(recordFile(commonDir, "sdd", "x")), "x.lock"): "el bloqueo del cambio",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Delete no debe tocar %s: %v", what, err)
		}
	}
	if err := store.Delete(commonDir, branch.LaneSDD, "x"); err != nil {
		t.Errorf("borrar un registro que no existe no es un error: %v", err)
	}
	if err := New().Delete(commonDir, branch.LaneSDD, "nunca-existio"); err != nil {
		t.Errorf("borrar el registro de un cambio sin carpeta de carril tampoco: %v", err)
	}
}

// Una clave que no puede componer la ruta de un registro se rechaza sin tocar nada, como en Load
// y Save: Delete nunca sale de la carpeta del registro.
func TestDeleteRefusesAKeyThatCannotNameARecord(t *testing.T) {
	root, commonDir := newRepository(t)
	outside := filepath.Join(commonDir, "axiom", "branch", "v1", "sdd", "x.json")
	if err := New().Save(commonDir, recordOf(root, commonDir, branch.LaneSDD, "x")); err != nil {
		t.Fatal(err)
	}

	for name, call := range map[string]func() error{
		"nombre con separadores": func() error { return New().Delete(commonDir, branch.LaneSDD, "../x") },
		"carril desconocido":     func() error { return New().Delete(commonDir, branch.Lane("otro"), "x") },
		"common dir relativo":    func() error { return New().Delete("relativo", branch.LaneSDD, "x") },
	} {
		if err := call(); err == nil {
			t.Errorf("%s: Delete debía rechazar la clave", name)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("los rechazos no deben borrar el registro: %v", err)
	}
}
