package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain deja a los procesos git de las pruebas sin acceso a la configuración del
// usuario, con un autor fijo y sin red (tasks.md, regla 9), con el mismo patrón que
// internal/specbaseline, y fija el techo de descubrimiento de proyectos de las pruebas
// (workspace.DiscoveryCeilingOverride).
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

// isolatedGitConfig es el gitconfig global de las pruebas: sin gc ni mantenimiento
// automático en segundo plano (que compite con la limpieza del directorio temporal) y
// sin fsmonitor, caché de ficheros sin seguimiento ni fsync, que solo encarecen cada
// proceso git en Windows.
const isolatedGitConfig = `[gc]
	auto = 0
	autoDetach = false
[maintenance]
	auto = false
[core]
	fsmonitor = false
	untrackedCache = false
	fsync = none
`

func runTests(m *testing.M) int {
	root, err := os.MkdirTemp("", "registry-pruebas-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: crear la carpeta temporal de las pruebas: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(root) }()

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "registry: crear el HOME aislado: %v\n", err)
		return 1
	}
	gitConfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gitConfig, []byte(isolatedGitConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "registry: escribir el gitconfig aislado: %v\n", err)
		return 1
	}
	templateDir := filepath.Join(root, "git-template")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "registry: crear la plantilla vacía de git: %v\n", err)
		return 1
	}
	for key, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"GIT_CONFIG_GLOBAL":   gitConfig,
		"GIT_TEMPLATE_DIR":    templateDir,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME":     "Autor de Prueba",
		"GIT_AUTHOR_EMAIL":    "autor@ejemplo.invalid",
		"GIT_COMMITTER_NAME":  "Autor de Prueba",
		"GIT_COMMITTER_EMAIL": "autor@ejemplo.invalid",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "registry: fijar %s: %v\n", key, err)
			return 1
		}
	}
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	return m.Run()
}

func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
}

// runGit ejecuta git directamente, sin pasar por gitcommon.Run, para que el oráculo de
// una prueba no dependa del código que prueba.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v en %s: %v\n%s", args, dir, err, output)
	}
}

// newRepository crea un repositorio real en la rama main y con un commit, y devuelve su
// raíz y su git common dir.
func newRepository(t *testing.T) (root, commonDir string) {
	t.Helper()
	requireGit(t)
	root = t.TempDir()
	runGit(t, root, "init", "-q", "-b", "main")
	runGit(t, root, "commit", "-q", "--allow-empty", "-m", "inicial")
	return root, commonDirOf(t, root)
}

func commonDirOf(t *testing.T, dir string) string {
	t.Helper()
	common, err := gitcommon.CommonDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("git common dir de %s: %v", dir, err)
	}
	return common
}

// recordFile es la ruta del registro de un cambio, escrita como la fija el diseño
// (§4.4): `<common dir>/axiom/branch/v1/<carril>/<cambio>.json`. La prueba no la toma
// del código que prueba.
func recordFile(commonDir, lane, change string) string {
	return filepath.Join(commonDir, "axiom", "branch", "v1", lane, change+".json")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// validRecord es el registro de un cambio materializado en modo worktree, con rutas
// absolutas.
func validRecord(root, commonDir string) branch.Record {
	return branch.Record{
		Schema:        branch.RegistrySchema,
		Lane:          branch.LaneSDD,
		Change:        "x",
		Mode:          branch.ModeWorktree,
		ModeSource:    branch.ModeSourceAnswer,
		Exceptions:    branch.Exceptions{Branch: "inc/x"},
		ArchivePrefix: "",
		PlanningRepo:  root,
		Repositories: []branch.RecordRepo{{
			Name:      "axiom",
			GitRoot:   root,
			CommonDir: commonDir,
			Role:      branch.RepoChangeBranch,
			Branches: []branch.RecordBranch{
				{Name: "inc/x", Kind: branch.KindChange, Worktree: filepath.Join(filepath.Dir(root), "x", "axiom")},
				{Name: "inc/x-core", Kind: branch.KindRole, RoleID: "core", Worktree: filepath.Join(filepath.Dir(root), "x-core", "axiom")},
			},
		}},
	}
}

func writeRecord(t *testing.T, commonDir, lane string, record branch.Record) string {
	t.Helper()
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := recordFile(commonDir, lane, record.Change)
	writeFile(t, path, string(data)+"\n")
	return path
}

func load(t *testing.T, commonDir string, lane branch.Lane, change string) (*branch.Record, error) {
	t.Helper()
	return New().Load(commonDir, lane, change)
}

// REQ-25.14 y design D-07: sin fichero, el registro no existe y no es un error
// (`status` informa `registry.present: false`); leerlo no crea nada.
func TestLoadOfAnAbsentRecordReturnsNilWithoutCreatingAnything(t *testing.T) {
	_, commonDir := newRepository(t)

	t.Run("sin carpeta del registro", func(t *testing.T) {
		record, err := load(t, commonDir, branch.LaneSDD, "x")
		if record != nil || err != nil {
			t.Fatalf("Load = %+v, %v; se esperaba nil, nil", record, err)
		}
		if _, statErr := os.Stat(filepath.Join(commonDir, "axiom")); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("leer un registro ausente no debe crear %s (%v)", filepath.Join(commonDir, "axiom"), statErr)
		}
	})
	t.Run("con la carpeta del carril pero sin el fichero del cambio", func(t *testing.T) {
		writeFile(t, recordFile(commonDir, "sdd", "otro"), "{}")
		record, err := load(t, commonDir, branch.LaneSDD, "x")
		if record != nil || err != nil {
			t.Fatalf("Load = %+v, %v; se esperaba nil, nil", record, err)
		}
	})
}

// REQ-25.14: `Load` lee `<common dir>/axiom/branch/v1/<carril>/<cambio>.json`; cada
// carril tiene sus propios registros, aunque el cambio se llame igual.
func TestLoadReadsTheRecordOfTheChangeAndLane(t *testing.T) {
	root, commonDir := newRepository(t)
	want := validRecord(root, commonDir)
	writeRecord(t, commonDir, "sdd", want)

	got, err := load(t, commonDir, branch.LaneSDD, "x")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, &want) {
		t.Errorf("Load = %+v\nse esperaba %+v", got, &want)
	}

	t.Run("el carril odd no ve el registro del carril sdd", func(t *testing.T) {
		if record, err := load(t, commonDir, branch.LaneODD, "x"); record != nil || err != nil {
			t.Fatalf("Load odd = %+v, %v; se esperaba nil, nil", record, err)
		}
		odd := validRecord(root, commonDir)
		odd.Lane = branch.LaneODD
		odd.Mode, odd.ModeSource = branch.ModeInPlace, branch.ModeSourceFlag
		writeRecord(t, commonDir, "odd", odd)
		record, err := load(t, commonDir, branch.LaneODD, "x")
		if err != nil || record == nil || record.Mode != branch.ModeInPlace {
			t.Fatalf("Load odd = %+v, %v; se esperaba el registro in_place", record, err)
		}
	})
	t.Run("otro cambio no ve el registro", func(t *testing.T) {
		if record, err := load(t, commonDir, branch.LaneSDD, "y"); record != nil || err != nil {
			t.Fatalf("Load y = %+v, %v; se esperaba nil, nil", record, err)
		}
	})
	t.Run("tolera campos que no conoce", func(t *testing.T) {
		data, err := json.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		extended := strings.Replace(string(data), `{"schema"`, `{"futuro":{"a":1},"schema"`, 1)
		writeFile(t, recordFile(commonDir, "sdd", "x"), extended)
		got, err := load(t, commonDir, branch.LaneSDD, "x")
		if err != nil || got == nil || got.Change != "x" {
			t.Fatalf("Load = %+v, %v; un campo desconocido no invalida el registro", got, err)
		}
	})
}

// REQ-25.14, escenario «El registro es visible desde un worktree enlazado»: el git
// common dir es el mismo desde el checkout principal y desde cualquier worktree
// enlazado, de modo que el registro escrito desde uno se lee desde el otro.
func TestTheRecordIsVisibleFromALinkedWorktree(t *testing.T) {
	root, commonDir := newRepository(t)
	linked := filepath.Join(t.TempDir(), "x", "axiom")
	runGit(t, root, "worktree", "add", "-q", "-b", "inc/x", linked, "main")

	linkedCommon := commonDirOf(t, linked)
	if !pathidentity.SameDirectory(commonDir, linkedCommon) {
		t.Fatalf("el common dir del worktree enlazado (%s) debe ser el del checkout principal (%s)", linkedCommon, commonDir)
	}
	want := validRecord(root, commonDir)
	writeRecord(t, commonDir, "sdd", want)

	got, err := load(t, linkedCommon, branch.LaneSDD, "x")
	if err != nil || got == nil {
		t.Fatalf("Load desde el worktree enlazado = %+v, %v", got, err)
	}
	if !reflect.DeepEqual(got, &want) {
		t.Errorf("el registro leído desde el worktree enlazado difiere:\n%+v\n%+v", got, &want)
	}
}

// REQ-25.14 y design D-07: un fichero corrupto es `config_invalid` para quien lo
// consume: el error nombra el fichero y la causa. El registro es estado local
// derivado: `ensure` lo reconstruye, así que nada lo interpreta con indulgencia.
func TestLoadRejectsACorruptRecordNamingTheFile(t *testing.T) {
	root, commonDir := newRepository(t)
	valid := validRecord(root, commonDir)

	mutate := func(change func(*branch.Record)) string {
		record := valid
		record.Repositories = append([]branch.RecordRepo(nil), valid.Repositories...)
		record.Repositories[0].Branches = append([]branch.RecordBranch(nil), valid.Repositories[0].Branches...)
		change(&record)
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	tests := []struct {
		name    string
		content string
		wantIn  string
	}{
		{"JSON mal formado", `{"schema": "axiom.branch-registry/v1", `, "JSON"},
		{"fichero vacío", ``, "JSON"},
		{"datos detrás del documento", mutate(func(*branch.Record) {}) + `{}`, "JSON"},
		{"esquema desconocido", mutate(func(r *branch.Record) { r.Schema = "axiom.branch-registry/v2" }), "esquema"},
		{"sin esquema", mutate(func(r *branch.Record) { r.Schema = "" }), "esquema"},
		{"carril distinto del que se pidió", mutate(func(r *branch.Record) { r.Lane = branch.LaneODD }), "carril"},
		{"cambio distinto del que se pidió", mutate(func(r *branch.Record) { r.Change = "otro" }), "cambio"},
		{"modo desconocido", mutate(func(r *branch.Record) { r.Mode = "ask" }), "modo"},
		{"modo vacío", mutate(func(r *branch.Record) { r.Mode = "" }), "modo"},
		{"origen del modo desconocido", mutate(func(r *branch.Record) { r.ModeSource = "azar" }), "origen del modo"},
		{"repositorio del planning home relativo", mutate(func(r *branch.Record) { r.PlanningRepo = "axiom" }), "planningRepo"},
		{"raíz git relativa", mutate(func(r *branch.Record) { r.Repositories[0].GitRoot = "axiom" }), "gitRoot"},
		{"common dir relativo", mutate(func(r *branch.Record) { r.Repositories[0].CommonDir = ".git" }), "commonDir"},
		{"worktree relativo", mutate(func(r *branch.Record) { r.Repositories[0].Branches[0].Worktree = "x/axiom" }), "worktree"},
		{"prefijo de excepción inválido", mutate(func(r *branch.Record) { r.Exceptions.Prefix = "mi prefijo//" }), "prefix"},
		{"rama de excepción inválida", mutate(func(r *branch.Record) { r.Exceptions.Branch = "a..b" }), "branch"},
		{"prefijo de archivado inválido", mutate(func(r *branch.Record) { r.ArchivePrefix = "mi prefijo//" }), "archivePrefix"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := recordFile(commonDir, "sdd", "x")
			writeFile(t, path, tc.content)

			record, err := load(t, commonDir, branch.LaneSDD, "x")

			if record != nil || err == nil {
				t.Fatalf("Load = %+v, %v; se esperaba un error", record, err)
			}
			var failure *FileError
			if !errors.As(err, &failure) {
				t.Fatalf("el error %T (%v) debe ser un *FileError", err, err)
			}
			if failure.Path != path || !strings.Contains(err.Error(), path) {
				t.Errorf("el error debe nombrar el fichero %s: path %q, mensaje %q", path, failure.Path, err.Error())
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Errorf("el mensaje %q debe nombrar la causa (%q)", err.Error(), tc.wantIn)
			}
			if failure.Unwrap() == nil {
				t.Error("el error conserva su causa")
			}
		})
	}

	t.Run("un directorio en lugar del fichero no se puede leer", func(t *testing.T) {
		path := recordFile(commonDir, "sdd", "dir")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := load(t, commonDir, branch.LaneSDD, "dir")
		var failure *FileError
		if !errors.As(err, &failure) || failure.Path != path {
			t.Fatalf("Load = %v; se esperaba un *FileError que nombre %s", err, path)
		}
	})
}

// Un nombre de cambio o un carril inválidos nunca llegan a componer una ruta: el motor
// los valida antes, y el registro no sale de su carpeta aunque se le entregue uno.
func TestLoadNeverLeavesTheRegistryFolder(t *testing.T) {
	_, commonDir := newRepository(t)
	outside := filepath.Join(commonDir, "axiom", "branch", "v1", "fuera.json")
	writeFile(t, outside, "{}")

	for _, tc := range []struct {
		lane   branch.Lane
		change string
	}{
		{branch.LaneSDD, "../fuera"},
		{branch.LaneSDD, `..\fuera`},
		{branch.LaneSDD, ""},
		{branch.LaneSDD, "a/b"},
		{"../v1", "x"},
		{"", "x"},
	} {
		record, err := load(t, commonDir, tc.lane, tc.change)
		if record != nil || err == nil {
			t.Errorf("Load(%q, %q) = %+v, %v; se esperaba un error", tc.lane, tc.change, record, err)
			continue
		}
		var failure *FileError
		if errors.As(err, &failure) {
			t.Errorf("Load(%q, %q): un argumento inválido no es un fallo de fichero: %v", tc.lane, tc.change, err)
		}
	}
}

// recordOf es el registro de un cambio y carril dados, derivado del válido de la prueba.
func recordOf(root, commonDir string, lane branch.Lane, change string) branch.Record {
	record := validRecord(root, commonDir)
	record.Lane, record.Change = lane, change
	return record
}

func changesOf(records []branch.Record) []string {
	var changes []string
	for _, record := range records {
		changes = append(changes, record.Change)
	}
	return changes
}

// REQ-25.15 (escenario «`status` sin `--change` lista registrados y pendientes») y
// design D-07: `List` devuelve los registros del carril en un orden estable (por cambio,
// no por nombre de fichero: `a-b.json` precede a `a.json`), sin mezclar carriles, y sin
// crear nada.
func TestListReturnsTheRecordsOfTheLaneInAStableOrder(t *testing.T) {
	root, commonDir := newRepository(t)

	t.Run("sin carpeta del registro no hay nada que listar y no se crea nada", func(t *testing.T) {
		records, err := New().List(commonDir, branch.LaneSDD)
		if err != nil || len(records) != 0 {
			t.Fatalf("List = %+v, %v; se esperaba una lista vacía sin error", records, err)
		}
		if _, statErr := os.Stat(filepath.Join(commonDir, "axiom")); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("listar no debe crear %s (%v)", filepath.Join(commonDir, "axiom"), statErr)
		}
	})

	for _, change := range []string{"b", "a-b", "a"} {
		writeRecord(t, commonDir, "sdd", recordOf(root, commonDir, branch.LaneSDD, change))
	}
	writeRecord(t, commonDir, "odd", recordOf(root, commonDir, branch.LaneODD, "z"))

	t.Run("ordena por cambio", func(t *testing.T) {
		records, err := New().List(commonDir, branch.LaneSDD)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got, want := changesOf(records), []string{"a", "a-b", "b"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambios = %v, se esperaba %v", got, want)
		}
		want := recordOf(root, commonDir, branch.LaneSDD, "a")
		if !reflect.DeepEqual(records[0], want) {
			t.Errorf("el registro leído difiere:\n%+v\n%+v", records[0], want)
		}
	})
	t.Run("cada carril lista solo los suyos", func(t *testing.T) {
		records, err := New().List(commonDir, branch.LaneODD)
		if err != nil || !reflect.DeepEqual(changesOf(records), []string{"z"}) {
			t.Fatalf("List odd = %v, %v; se esperaba solo z", changesOf(records), err)
		}
	})
	t.Run("lo que no es un registro se ignora", func(t *testing.T) {
		writeFile(t, filepath.Join(commonDir, "axiom", "branch", "v1", "sdd", "notas.txt"), "no es un registro")
		if err := os.MkdirAll(filepath.Join(commonDir, "axiom", "branch", "v1", "sdd", "carpeta"), 0o755); err != nil {
			t.Fatal(err)
		}
		records, err := New().List(commonDir, branch.LaneSDD)
		if err != nil || !reflect.DeepEqual(changesOf(records), []string{"a", "a-b", "b"}) {
			t.Fatalf("List = %v, %v; un fichero que no es .json y una carpeta no son registros", changesOf(records), err)
		}
	})
	t.Run("es visible desde un worktree enlazado", func(t *testing.T) {
		linked := filepath.Join(t.TempDir(), "x", "axiom")
		runGit(t, root, "worktree", "add", "-q", "-b", "inc/x", linked, "main")
		records, err := New().List(commonDirOf(t, linked), branch.LaneSDD)
		if err != nil || !reflect.DeepEqual(changesOf(records), []string{"a", "a-b", "b"}) {
			t.Fatalf("List desde el worktree enlazado = %v, %v", changesOf(records), err)
		}
	})
}

// design D-07: un fichero que existe y no es un registro válido es un *FileError que lo
// nombra, como en `Load`, y no se devuelve una lista a medias: el registro es estado
// derivado y nada lo interpreta con indulgencia.
func TestListRejectsAnInvalidRecordNamingTheFile(t *testing.T) {
	root, commonDir := newRepository(t)
	writeRecord(t, commonDir, "sdd", recordOf(root, commonDir, branch.LaneSDD, "bueno"))

	tests := []struct {
		name   string
		setup  func(t *testing.T) string
		wantIn string
	}{
		{"JSON mal formado", func(t *testing.T) string {
			path := recordFile(commonDir, "sdd", "roto")
			writeFile(t, path, `{"schema": `)
			return path
		}, "JSON"},
		{"un directorio en lugar del fichero", func(t *testing.T) string {
			path := recordFile(commonDir, "sdd", "carpeta")
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			return path
		}, "registro local"},
		{"un nombre que no puede ser el de un cambio", func(t *testing.T) string {
			path := recordFile(commonDir, "sdd", "a b")
			writeFile(t, path, "{}")
			return path
		}, "nombre de cambio"},
		{"un registro de otro cambio", func(t *testing.T) string {
			path := recordFile(commonDir, "sdd", "x")
			data, err := json.Marshal(recordOf(root, commonDir, branch.LaneSDD, "otro"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, string(data))
			return path
		}, "cambio"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			t.Cleanup(func() { _ = os.RemoveAll(path) })

			records, err := New().List(commonDir, branch.LaneSDD)

			if records != nil || err == nil {
				t.Fatalf("List = %v, %v; se esperaba un error sin lista", changesOf(records), err)
			}
			var failure *FileError
			if !errors.As(err, &failure) || failure.Path != path {
				t.Fatalf("el error %T (%v) debe ser un *FileError que nombre %s", err, err, path)
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Errorf("el mensaje %q debe nombrar la causa (%q)", err.Error(), tc.wantIn)
			}
		})
	}
}

// Un carril o un git common dir que no pueden componer la carpeta del registro no
// llegan a leer nada: error corriente, no de fichero.
func TestListNeverLeavesTheRegistryFolder(t *testing.T) {
	_, commonDir := newRepository(t)
	writeFile(t, filepath.Join(commonDir, "axiom", "branch", "v1", "fuera", "x.json"), "{}")

	for _, tc := range []struct {
		name      string
		commonDir string
		lane      branch.Lane
	}{
		{"carril que sube de carpeta", commonDir, "../v1"},
		{"carril vacío", commonDir, ""},
		{"carril desconocido", commonDir, "fuera"},
		{"git common dir relativo", ".git", branch.LaneSDD},
		{"git common dir vacío", "", branch.LaneSDD},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records, err := New().List(tc.commonDir, tc.lane)
			if records != nil || err == nil {
				t.Fatalf("List = %v, %v; se esperaba un error", changesOf(records), err)
			}
			var failure *FileError
			if errors.As(err, &failure) {
				t.Errorf("un argumento inválido no es un fallo de fichero: %v", err)
			}
		})
	}
}

// REQ-25.14 y design D-07: `Save` escribe el registro con una serialización estable (JSON
// con sangrado de dos espacios y `\n`, sin `\r` ni marcas de tiempo), en la ruta que fija el
// diseño, sin dejar ficheros temporales, y lo que escribe lo lee `Load` igual.
func TestSaveWritesTheStableSerializationWithNewlines(t *testing.T) {
	root, commonDir := newRepository(t)
	want := validRecord(root, commonDir)

	if err := New().Save(commonDir, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := recordFile(commonDir, "sdd", "x")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("el registro debe estar en %s: %v", path, err)
	}
	expected, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(expected)+"\n" {
		t.Errorf("serialización:\n%s\nse esperaba JSON con sangrado de dos espacios y un único `\\n` final:\n%s\n", data, expected)
	}
	if strings.Contains(string(data), "\r") {
		t.Errorf("el registro se escribe con `\\n`, también en Windows: %q", data)
	}
	got, err := load(t, commonDir, branch.LaneSDD, "x")
	if err != nil || !reflect.DeepEqual(got, &want) {
		t.Fatalf("Load tras Save = %+v, %v; se esperaba %+v", got, err, &want)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "x.json" {
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("la carpeta del registro solo debe tener x.json, tiene %v (sin temporales)", names)
	}
}

// design D-05 (fase 5): «solo se escribe si cambian sus bytes». Un registro idéntico no
// toca el fichero (su fecha de modificación no cambia); uno distinto lo reemplaza.
func TestSaveDoesNotWriteWhenTheBytesDoNotChange(t *testing.T) {
	root, commonDir := newRepository(t)
	record := validRecord(root, commonDir)
	if err := New().Save(commonDir, record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	path := recordFile(commonDir, "sdd", "x")
	old := time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	t.Run("bytes idénticos", func(t *testing.T) {
		if err := New().Save(commonDir, record); err != nil {
			t.Fatalf("Save: %v", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.ModTime().Equal(old) {
			t.Errorf("fecha de modificación %v, se esperaba %v: un registro idéntico no se reescribe", info.ModTime(), old)
		}
	})
	t.Run("bytes distintos", func(t *testing.T) {
		changed := record
		changed.Mode, changed.ModeSource = branch.ModeInPlace, branch.ModeSourceFlag
		if err := New().Save(commonDir, changed); err != nil {
			t.Fatalf("Save: %v", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.ModTime().Equal(old) {
			t.Error("un registro distinto debe reescribir el fichero")
		}
		got, err := load(t, commonDir, branch.LaneSDD, "x")
		if err != nil || got == nil || got.Mode != branch.ModeInPlace || got.ModeSource != branch.ModeSourceFlag {
			t.Fatalf("Load = %+v, %v; se esperaba el registro in_place", got, err)
		}
	})
}

// El registro es estado derivado y `Load` no lo interpreta con indulgencia: `Save` tampoco
// escribe nada que `Load` rechazaría, y un rechazo no crea ni la carpeta del registro.
func TestSaveRefusesARecordThatLoadWouldRejectWithoutWritingAnything(t *testing.T) {
	root, commonDir := newRepository(t)
	tests := []struct {
		name      string
		commonDir string
		mutate    func(*branch.Record)
	}{
		{"esquema desconocido", commonDir, func(r *branch.Record) { r.Schema = "otro/v9" }},
		{"carril inválido", commonDir, func(r *branch.Record) { r.Lane = "xyz" }},
		{"cambio que no puede nombrar un registro", commonDir, func(r *branch.Record) { r.Change = "a b" }},
		{"modo sin materializar", commonDir, func(r *branch.Record) { r.Mode = branch.ModeAsk }},
		{"ruta del worktree relativa", commonDir, func(r *branch.Record) { r.Repositories[0].Branches[0].Worktree = "x/axiom" }},
		{"prefijo de archivado inválido", commonDir, func(r *branch.Record) { r.ArchivePrefix = "a b" }},
		{"git common dir relativo", ".git", func(*branch.Record) {}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := validRecord(root, commonDir)
			tc.mutate(&record)

			if err := New().Save(tc.commonDir, record); err == nil {
				t.Fatal("Save debía rechazar el registro")
			}
			if _, err := os.Stat(filepath.Join(commonDir, "axiom")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("un registro rechazado no debe crear nada (%v)", err)
			}
		})
	}
}

// design D-07, «Coexistencia con gentle-ai/»: espacios de nombres separados. Guardar y
// bloquear el registro nunca toca `<git common dir>/gentle-ai/`.
func TestTheRegistryNeverTouchesTheGentleAIFolder(t *testing.T) {
	root, commonDir := newRepository(t)
	lineage := filepath.Join(commonDir, "gentle-ai", "review-transactions", "linaje-1", "estado.json")
	writeFile(t, lineage, `{"linaje":"abierto"}`)
	old := time.Date(2002, time.March, 4, 5, 6, 7, 0, time.UTC)
	if err := os.Chtimes(lineage, old, old); err != nil {
		t.Fatal(err)
	}

	unlock, err := New().Lock(context.Background(), commonDir, branch.LaneSDD, "x")
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	saveErr := New().Save(commonDir, validRecord(root, commonDir))
	unlock()
	if saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	data, err := os.ReadFile(lineage)
	if err != nil || string(data) != `{"linaje":"abierto"}` {
		t.Fatalf("el linaje de RDD cambió: %q, %v", data, err)
	}
	if info, err := os.Stat(lineage); err != nil || !info.ModTime().Equal(old) {
		t.Errorf("el linaje de RDD no debe tocarse (%v, %v)", info, err)
	}
	entries, err := os.ReadDir(filepath.Join(commonDir, "gentle-ai"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "review-transactions" {
		t.Errorf("gentle-ai/ no debe ganar ni perder entradas: %v, %v", entries, err)
	}
	if _, err := os.Stat(recordFile(commonDir, "sdd", "x")); err != nil {
		t.Errorf("el registro vive en axiom/branch/v1: %v", err)
	}
}

// REQ-25.14 y design D-05: el bloqueo es exclusivo por cambio y carril, no es reentrante
// (un segundo `ensure` del mismo cambio lo encuentra ocupado), no bloquea otros cambios ni
// el otro carril, se libera al terminar y liberarlo dos veces es inocuo.
func TestLockIsExclusivePerChangeAndLane(t *testing.T) {
	_, commonDir := newRepository(t)
	store := New()

	unlock, err := store.Lock(context.Background(), commonDir, branch.LaneSDD, "x")
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}

	if _, err := store.Lock(context.Background(), commonDir, branch.LaneSDD, "x"); !errors.Is(err, ErrBusy) {
		t.Errorf("segundo Lock del mismo cambio = %v; se esperaba ErrBusy", err)
	}
	for name, other := range map[string]func() (func(), error){
		"otro cambio": func() (func(), error) { return store.Lock(context.Background(), commonDir, branch.LaneSDD, "y") },
		"otro carril": func() (func(), error) { return store.Lock(context.Background(), commonDir, branch.LaneODD, "x") },
	} {
		release, err := other()
		if err != nil {
			t.Errorf("%s: el bloqueo de x/sdd no debe bloquearlo: %v", name, err)
			continue
		}
		release()
	}

	unlock()
	unlock()
	again, err := store.Lock(context.Background(), commonDir, branch.LaneSDD, "x")
	if err != nil {
		t.Fatalf("tras liberar, el cambio vuelve a poder bloquearse: %v", err)
	}
	again()
	if _, err := os.Stat(filepath.Join(commonDir, "axiom", "branch", "v1", "sdd", "x.lock")); err != nil {
		t.Errorf("el bloqueo es `<cambio>.lock` junto al registro: %v", err)
	}
}

// El bloqueo, como el resto del almacén, nunca sale de la carpeta del registro y rechaza
// sin crear nada una clave que no puede componerla o un contexto ya cancelado.
func TestLockRefusesAKeyThatCannotNameARecordWithoutCreatingAnything(t *testing.T) {
	_, commonDir := newRepository(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	for _, tc := range []struct {
		name      string
		ctx       context.Context
		commonDir string
		lane      branch.Lane
		change    string
	}{
		{"cambio con espacios", context.Background(), commonDir, branch.LaneSDD, "a b"},
		{"cambio que sube de carpeta", context.Background(), commonDir, branch.LaneSDD, "../x"},
		{"carril desconocido", context.Background(), commonDir, "fuera", "x"},
		{"git common dir relativo", context.Background(), ".git", branch.LaneSDD, "x"},
		{"contexto cancelado", cancelled, commonDir, branch.LaneSDD, "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unlock, err := New().Lock(tc.ctx, tc.commonDir, tc.lane, tc.change)
			if err == nil {
				unlock()
				t.Fatal("Lock debía rechazar la petición")
			}
			if errors.Is(err, ErrBusy) {
				t.Errorf("una clave inválida no es un bloqueo ocupado: %v", err)
			}
			if _, statErr := os.Stat(filepath.Join(commonDir, "axiom")); !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("un rechazo no debe crear nada (%v)", statErr)
			}
		})
	}
}

// REQ-25.14, REQ-25.45 y design §4.4: el registro guarda la rama de archivado con `kind:
// archive` y el prefijo de archivado, y los lee tal cual; repetir el guardado no reescribe el
// fichero.
func TestSaveAndLoadKeepTheArchiveBranchAndItsPrefix(t *testing.T) {
	root, commonDir := newRepository(t)
	record := validRecord(root, commonDir)
	record.ArchivePrefix = "doc"
	record.Repositories[0].Branches = append(record.Repositories[0].Branches,
		branch.RecordBranch{Name: "doc/x-archive", Kind: branch.KindArchive, Worktree: filepath.Join(filepath.Dir(root), "x-archive", "axiom")})

	if err := New().Save(commonDir, record); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(recordFile(commonDir, "sdd", "x"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"archivePrefix": "doc"`, `"name": "doc/x-archive"`, `"kind": "archive"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("el registro debe contener %s:\n%s", want, data)
		}
	}
	got, err := load(t, commonDir, branch.LaneSDD, "x")
	if err != nil || got == nil || !reflect.DeepEqual(*got, record) {
		t.Fatalf("Load = %+v, %v; se esperaba el mismo registro %+v", got, err, record)
	}
	if err := New().Save(commonDir, record); err != nil {
		t.Fatal(err)
	}
	if again, err := os.ReadFile(recordFile(commonDir, "sdd", "x")); err != nil || !bytes.Equal(again, data) {
		t.Errorf("repetir el guardado no cambia los bytes (%v)", err)
	}
}
