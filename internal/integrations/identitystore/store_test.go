package identitystore

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

const plugin = "azure-devops"

var resolvedAt = time.Date(2026, 11, 22, 9, 30, 0, 0, time.UTC)

func entry(id, name, source string) Entry {
	return Entry{ID: id, DisplayName: name, Source: source, ResolvedAt: resolvedAt}
}

func newStore(t *testing.T) Store {
	t.Helper()
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	return Store{Path: path}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func TestDefaultPathHonoursAxiomStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", dir)

	path, err := DefaultPath()

	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, FileName) {
		t.Errorf("ruta = %q, quería %q", path, filepath.Join(dir, FileName))
	}
}

func TestLoadOfAMissingFileIsEmpty(t *testing.T) {
	store := newStore(t)

	file, err := store.Load()

	if err != nil {
		t.Fatal(err)
	}
	if file.Schema != SchemaV1 || len(file.Plugins) != 0 || file.Plugins == nil {
		t.Errorf("fichero vacío esperado, obtuvo %+v", file)
	}
	if _, found, err := store.Get(plugin, "contoso"); err != nil || found {
		t.Errorf("Get sobre fichero inexistente = %v, %v", found, err)
	}
}

func TestFirstPutCreatesTheFileWithTheEntry(t *testing.T) {
	store := newStore(t)

	if err := store.Put(plugin, "contoso", entry("guid-1", "Ana", "connection_data")); err != nil {
		t.Fatal(err)
	}

	got, found, err := store.Get(plugin, "contoso")
	if err != nil || !found {
		t.Fatalf("Get = %v, %v", found, err)
	}
	if got != entry("guid-1", "Ana", "connection_data") {
		t.Errorf("entrada = %+v", got)
	}
	data, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{SchemaV1, `"contoso"`, `"id": "guid-1"`, `"display_name": "Ana"`, `"source": "connection_data"`, `"resolved_at": "2026-11-22T09:30:00Z"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("el fichero debe contener %s:\n%s", want, data)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(store.Path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("modo = %o, quería 600", info.Mode().Perm())
		}
	}
}

func TestPutKeepsOtherOrganizationsUntouched(t *testing.T) {
	store := newStore(t)
	if err := store.Put(plugin, "contoso", entry("guid-1", "Ana", "connection_data")); err != nil {
		t.Fatal(err)
	}

	if err := store.Put(plugin, "fabrikam", entry("guid-2", "Ana F", "member_selection")); err != nil {
		t.Fatal(err)
	}

	contoso, _, _ := store.Get(plugin, "contoso")
	fabrikam, _, _ := store.Get(plugin, "fabrikam")
	if contoso != entry("guid-1", "Ana", "connection_data") || fabrikam != entry("guid-2", "Ana F", "member_selection") {
		t.Errorf("contoso = %+v, fabrikam = %+v", contoso, fabrikam)
	}
}

func TestPutReplacesTheEntryOfThatOrganizationOnly(t *testing.T) {
	store := newStore(t)
	_ = store.Put(plugin, "contoso", entry("guid-1", "Ana", "member_selection"))
	_ = store.Put(plugin, "fabrikam", entry("guid-2", "Ana F", "member_selection"))
	later := Entry{ID: "guid-1", DisplayName: "Ana", Source: "connection_data", ResolvedAt: resolvedAt.Add(time.Hour)}

	if err := store.Put(plugin, "contoso", later); err != nil {
		t.Fatal(err)
	}

	contoso, _, _ := store.Get(plugin, "contoso")
	fabrikam, _, _ := store.Get(plugin, "fabrikam")
	if contoso != later {
		t.Errorf("contoso debe pasar a connection_data con la fecha nueva: %+v", contoso)
	}
	if fabrikam != entry("guid-2", "Ana F", "member_selection") {
		t.Errorf("fabrikam no debe cambiar: %+v", fabrikam)
	}
}

func TestOrganizationKeyIsNormalizedToLowerCase(t *testing.T) {
	store := newStore(t)

	if err := store.Put(plugin, "Contoso", entry("guid-1", "Ana", "connection_data")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(plugin, "CONTOSO", entry("guid-1", "Ana Pérez", "connection_data")); err != nil {
		t.Fatal(err)
	}

	file, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Plugins[plugin]) != 1 || file.Plugins[plugin]["contoso"].DisplayName != "Ana Pérez" {
		t.Fatalf("las organizaciones no distinguen mayúsculas: %+v", file.Plugins)
	}
	if _, found, _ := store.Get(plugin, "cOnToSo"); !found {
		t.Error("Get también normaliza la clave")
	}
}

func TestFailedWriteThroughTheSeamLeavesThePreviousFileAndNothingElse(t *testing.T) {
	store := newStore(t)
	if err := store.Put(plugin, "contoso", entry("guid-1", "Ana", "connection_data")); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.Path)
	namesBefore := dirNames(t, filepath.Dir(store.Path))

	boom := errors.New("fallo de escritura simulado")
	original := writeAtomic
	t.Cleanup(func() { writeAtomic = original })
	writeAtomic = func(string, []byte, os.FileMode) (filemerge.WriteResult, error) { return filemerge.WriteResult{}, boom }

	err := store.Put(plugin, "fabrikam", entry("guid-2", "Ana F", "member_selection"))

	if !errors.Is(err, boom) {
		t.Fatalf("Put = %v, quería el fallo de escritura", err)
	}
	after, _ := os.ReadFile(store.Path)
	if !bytes.Equal(before, after) {
		t.Errorf("el fichero anterior debe quedar byte a byte:\n%s\n---\n%s", before, after)
	}
	if got := dirNames(t, filepath.Dir(store.Path)); strings.Join(got, ",") != strings.Join(namesBefore, ",") {
		t.Errorf("no deben aparecer ficheros nuevos: antes %v, después %v", namesBefore, got)
	}
}

// failRealWrite provoca un fallo de la escritura atómica real (sin la costura): en Windows mantiene
// abierto el destino, lo que impide el renombrado que publica el temporal; en los demás sistemas
// quita el permiso de escritura del directorio. Devuelve false si no se puede provocar.
func failRealWrite(t *testing.T, path string) bool {
	t.Helper()
	if runtime.GOOS == "windows" {
		held, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = held.Close() })
		return true
	}
	if os.Geteuid() == 0 {
		return false // root ignora los permisos del directorio
	}
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o500); err != nil {
		return false
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	return true
}

func TestFailedWriteThroughTheRealAtomicWriterLeavesThePreviousFileAndNoTemporary(t *testing.T) {
	store := newStore(t)
	if err := store.Put(plugin, "contoso", entry("guid-1", "Ana", "connection_data")); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.Path)
	namesBefore := dirNames(t, filepath.Dir(store.Path))
	if !failRealWrite(t, store.Path) {
		t.Skip("no se puede provocar un fallo de la escritura atómica real en este entorno")
	}

	err := store.Put(plugin, "fabrikam", entry("guid-2", "Ana F", "member_selection"))

	if err == nil {
		t.Skip("el sistema de ficheros permitió la escritura pese a la inyección del fallo")
	}
	after, readErr := os.ReadFile(store.Path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("tras el fallo el fichero anterior debe quedar íntegro:\n%s\n---\n%s", before, after)
	}
	if got := dirNames(t, filepath.Dir(store.Path)); strings.Join(got, ",") != strings.Join(namesBefore, ",") {
		t.Errorf("no debe quedar ningún temporal: antes %v, después %v", namesBefore, got)
	}
}

func TestCorruptFileIsReportedAndNeverOverwritten(t *testing.T) {
	tests := map[string]string{
		"JSON inválido":       "{esto no es json",
		"esquema desconocido": `{"schema":"axiom.integration-identities/v99","plugins":{}}`,
		"sin esquema":         `{"plugins":{}}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			store := newStore(t)
			if err := os.WriteFile(store.Path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}

			err := store.Put(plugin, "contoso", entry("guid-1", "Ana", "connection_data"))

			var corrupt *CorruptError
			if !errors.As(err, &corrupt) || corrupt.Path != store.Path {
				t.Fatalf("se esperaba *CorruptError con la ruta, obtuvo %v", err)
			}
			if !strings.Contains(err.Error(), store.Path) {
				t.Errorf("el error debe nombrar el fichero: %q", err)
			}
			after, _ := os.ReadFile(store.Path)
			if string(after) != content {
				t.Errorf("el contenido ilegible no se sobrescribe: %q", after)
			}
			if _, _, err := store.Get(plugin, "contoso"); !errors.As(err, &corrupt) {
				t.Errorf("Get sobre un fichero corrupto = %v", err)
			}
		})
	}
}

func TestPutNeverTouchesWorkspacesJSON(t *testing.T) {
	store := newStore(t)
	workspaces := filepath.Join(filepath.Dir(store.Path), "workspaces.json")
	content := []byte(`{"schema":"x","projects":[]}`)
	if err := os.WriteFile(workspaces, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := store.Put(plugin, "contoso", entry("guid-1", "Ana", "connection_data")); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(workspaces)
	if err != nil || !bytes.Equal(got, content) {
		t.Errorf("workspaces.json debe quedar intacto: %q, %v", got, err)
	}
}

func TestPutRejectsEmptyKeys(t *testing.T) {
	store := newStore(t)

	if err := store.Put("", "contoso", entry("guid-1", "Ana", "connection_data")); err == nil {
		t.Error("un plugin vacío es un error")
	}
	if err := store.Put(plugin, "  ", entry("guid-1", "Ana", "connection_data")); err == nil {
		t.Error("una cuenta vacía es un error")
	}
	if err := store.Put(plugin, "contoso", Entry{DisplayName: "Ana"}); err == nil {
		t.Error("una entrada sin id es un error")
	}
	if _, err := os.Stat(store.Path); err == nil {
		t.Error("una petición rechazada no crea el fichero")
	}
}
