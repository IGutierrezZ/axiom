package livespec

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de Integration.Apply (REQ-30.2, REQ-30.3, D10): todo o nada, idempotencia
// byte a byte y regeneración del índice solo si ya existe.

// applyProject monta un proyecto embebido con la especificación de facturación y
// devuelve su Location real (con WorkspaceRoot y PlanningHome).
func applyProject(t *testing.T, files map[string]string) workspace.Location {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	writeTree(t, ws.Home, files)
	loc, err := workspace.Locate(ws.Master)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	return loc
}

func planAndApply(t *testing.T, loc workspace.Location, doc FeatureDoc) (Integration, []string, error) {
	t.Helper()
	reg, err := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanIntegration(loc, doc, reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	written, err := plan.Apply()
	return plan, written, err
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(root, path)
		if entry.IsDir() {
			files[filepath.ToSlash(rel)+"/"] = ""
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func sameTree(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if other, ok := b[k]; !ok || other != v {
			return false
		}
	}
	return true
}

func twoCapabilityDoc(t *testing.T) FeatureDoc {
	t.Helper()
	return docWith(t, "",
		[2]string{"facturacion", "## ADDED Requirements\n\n" + addedBlock("Nota de abono", "REQ-31.5", "El sistema DEBE emitir notas de abono.")},
		[2]string{"pagos", "## ADDED Requirements\n\n" + addedBlock("Cobro", "REQ-32.1", "El sistema DEBE cobrar.")},
	)
}

func TestApplyIntegratingTwiceLeavesIdenticalBytes(t *testing.T) {
	loc := applyProject(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	doc := twoCapabilityDoc(t)

	_, written, err := planAndApply(t, loc, doc)
	if err != nil {
		t.Fatalf("primera integración: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("written = %v, se esperaban dos especificaciones", written)
	}
	after := snapshotTree(t, loc.PlanningHome)
	if !strings.Contains(readFile(t, filepath.Join(loc.PlanningHome, "specs", "facturacion", "spec.md")), "Nota de abono (REQ-31.5)") {
		t.Fatal("la primera integración debe añadir el requisito")
	}

	second, written, err := planAndApply(t, loc, doc)
	if err != nil {
		t.Fatalf("segunda integración: %v", err)
	}
	if len(written) != 0 || !second.NothingToIntegrate() {
		t.Errorf("la segunda integración no escribe: written=%v plan=%+v", written, second)
	}
	if !sameTree(after, snapshotTree(t, loc.PlanningHome)) {
		t.Error("integrar dos veces debe dejar los bytes idénticos")
	}
	for _, c := range second.Capabilities {
		if c.Outcome != OutcomeAlready {
			t.Errorf("%s: Outcome = %q, se esperaba already_integrated", c.Name, c.Outcome)
		}
	}
}

func TestApplyCreatesTheCapabilityFolderAndFile(t *testing.T) {
	loc := applyProject(t, nil)
	plan, written, err := planAndApply(t, loc, twoCapabilityDoc(t))
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	path := filepath.Join(loc.PlanningHome, "specs", "pagos", "spec.md")
	if got := readFile(t, path); got != plan.Capabilities[1].Content {
		t.Errorf("contenido = %q", got)
	}
	if len(written) != 2 || written[1] != path {
		t.Errorf("written = %v", written)
	}
	for _, name := range []string{".spec.md.axiom-tmp"} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no debe quedar el temporal %s (%v)", name, err)
		}
	}
}

// Un fallo al escribir la segunda especificación restaura la primera y quita lo creado.
func TestApplyAllOrNothingWithAnInjectedRenameFailure(t *testing.T) {
	loc := applyProject(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	before := snapshotTree(t, loc.PlanningHome)

	calls := 0
	restore := renameFile
	renameFile = func(from, to string) error {
		calls++
		if calls == 2 {
			return errors.New("disco lleno simulado")
		}
		return os.Rename(from, to)
	}
	t.Cleanup(func() { renameFile = restore })

	_, written, err := planAndApply(t, loc, twoCapabilityDoc(t))
	if err == nil || !strings.Contains(err.Error(), "disco lleno simulado") {
		t.Fatalf("error = %v, se esperaba el fallo inyectado", err)
	}
	if len(written) != 0 {
		t.Errorf("written = %v: un Apply fallido no informa de escrituras", written)
	}
	if !sameTree(before, snapshotTree(t, loc.PlanningHome)) {
		t.Errorf("tras el fallo el árbol debe quedar como estaba:\nantes   %v\ndespués %v", before, snapshotTree(t, loc.PlanningHome))
	}
	if !strings.Contains(err.Error(), "axiom archive integrate --odd") {
		t.Errorf("el error debe nombrar la continuación: %v", err)
	}
}

func TestApplyAllOrNothingWithAnInjectedTemporaryWriteFailure(t *testing.T) {
	loc := applyProject(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	before := snapshotTree(t, loc.PlanningHome)

	calls := 0
	restore := writeTempFile
	writeTempFile = func(path string, data []byte, perm os.FileMode) error {
		calls++
		if calls == 2 {
			return errors.New("sin espacio simulado")
		}
		return os.WriteFile(path, data, perm)
	}
	t.Cleanup(func() { writeTempFile = restore })

	_, _, err := planAndApply(t, loc, twoCapabilityDoc(t))
	if err == nil || !strings.Contains(err.Error(), "sin espacio simulado") {
		t.Fatalf("error = %v", err)
	}
	if !sameTree(before, snapshotTree(t, loc.PlanningHome)) {
		t.Error("tras el fallo no debe quedar ningún temporal ni carpeta creada")
	}
}

func TestApplyRegeneratesTheIndexOnlyWhenItAlreadyExists(t *testing.T) {
	t.Run("sin índice no se crea", func(t *testing.T) {
		loc := applyProject(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
		_, written, err := planAndApply(t, loc, twoCapabilityDoc(t))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(loc.PlanningHome, "INDEX.md")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("INDEX.md no debe crearse (%v)", err)
		}
		if len(written) != 2 {
			t.Errorf("written = %v", written)
		}
	})
	t.Run("con índice se regenera", func(t *testing.T) {
		loc := applyProject(t, map[string]string{
			"specs/facturacion/spec.md": facturacionSpec,
			"INDEX.md":                  "# Índice viejo\n",
		})
		_, written, err := planAndApply(t, loc, twoCapabilityDoc(t))
		if err != nil {
			t.Fatal(err)
		}
		index := readFile(t, filepath.Join(loc.PlanningHome, "INDEX.md"))
		if strings.Contains(index, "Índice viejo") || !strings.Contains(index, "pagos") {
			t.Errorf("el índice debe regenerarse con la capacidad nueva:\n%s", index)
		}
		if last := written[len(written)-1]; last != filepath.Join(loc.PlanningHome, "INDEX.md") {
			t.Errorf("written = %v: el índice figura como escrito", written)
		}
	})
	t.Run("una integración sin cambios no toca el índice", func(t *testing.T) {
		loc := applyProject(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
		doc := twoCapabilityDoc(t)
		if _, _, err := planAndApply(t, loc, doc); err != nil {
			t.Fatal(err)
		}
		writeTree(t, loc.PlanningHome, map[string]string{"INDEX.md": "# Índice\n"})
		_, written, err := planAndApply(t, loc, doc)
		if err != nil || len(written) != 0 {
			t.Fatalf("written=%v err=%v", written, err)
		}
		if got := readFile(t, filepath.Join(loc.PlanningHome, "INDEX.md")); got != "# Índice\n" {
			t.Errorf("el índice no debe regenerarse si no se escribió ninguna especificación: %q", got)
		}
	})
}

func TestApplyRestoresTheSpecsAndTheIndexWhenTheIndexRegenerationFails(t *testing.T) {
	loc := applyProject(t, map[string]string{
		"specs/facturacion/spec.md": facturacionSpec,
		"INDEX.md":                  "# Índice viejo\n",
	})
	before := snapshotTree(t, loc.PlanningHome)

	restore := syncIndex
	syncIndex = func(workspace.Location) error { return errors.New("índice ilegible simulado") }
	t.Cleanup(func() { syncIndex = restore })

	_, _, err := planAndApply(t, loc, twoCapabilityDoc(t))
	if err == nil || !strings.Contains(err.Error(), "índice ilegible simulado") {
		t.Fatalf("error = %v", err)
	}
	if !sameTree(before, snapshotTree(t, loc.PlanningHome)) {
		t.Error("un fallo del índice debe deshacer también las especificaciones")
	}
}

func TestApplyWithNothingToIntegrateDoesNotTouchTheDisk(t *testing.T) {
	loc := applyProject(t, nil)
	reg, _ := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	plan, err := PlanIntegration(loc, parseOK(t, featureDoc("", "")), reg)
	if err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, loc.PlanningHome)
	written, err := plan.Apply()
	if err != nil || len(written) != 0 {
		t.Fatalf("written=%v err=%v", written, err)
	}
	if !sameTree(before, snapshotTree(t, loc.PlanningHome)) {
		t.Error("no debe escribirse nada")
	}
}
