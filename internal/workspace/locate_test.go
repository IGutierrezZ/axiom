package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// treeSnapshot lista, de forma ordenada, todo lo que hay bajo root. Sirve para
// comprobar que la resolución no escribe nada (REQ-24.1).
func treeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries = append(entries, path)
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer %s: %v", root, err)
	}
	return entries
}

func mustLocate(t *testing.T, start string) Location {
	t.Helper()
	loc, err := Locate(start)
	if err != nil {
		t.Fatalf("Locate(%q) = %v, se esperaba una resolución", start, err)
	}
	return loc
}

func wantLocateError(t *testing.T, start string, kind LocateErrorKind) *LocateError {
	t.Helper()
	_, err := Locate(start)
	var locateErr *LocateError
	if !errors.As(err, &locateErr) {
		t.Fatalf("Locate(%q) = %v, se esperaba un *LocateError", start, err)
	}
	if locateErr.Kind != kind {
		t.Fatalf("Locate(%q): kind = %q, se esperaba %q (%v)", start, locateErr.Kind, kind, err)
	}
	if locateErr.Cause == "" || locateErr.Continuation == "" {
		t.Fatalf("Locate(%q): el rechazo debe nombrar causa y continuación: %+v", start, locateErr)
	}
	return locateErr
}

func TestLocateCanonicalResolvesTheSamePlanningHomeFromEveryStart(t *testing.T) {
	ws := workspacetest.Canonical(t)
	before := treeSnapshot(t, ws.Master)

	starts := []struct {
		name   string
		dir    string
		origin Origin
	}{
		{"carpeta maestra", ws.Master, OriginPointer},
		{"repositorio de especificaciones", ws.Specs, OriginConfig},
		{"repositorio de código", filepath.Join(ws.Backend, "src"), OriginPointer},
	}
	for _, tc := range starts {
		t.Run(tc.name, func(t *testing.T) {
			loc := mustLocate(t, tc.dir)
			if loc.Master != ws.Master || loc.WorkspaceRoot != ws.Specs || loc.PlanningHome != ws.Home {
				t.Fatalf("Location = master %q root %q home %q; se esperaba %q %q %q", loc.Master, loc.WorkspaceRoot, loc.PlanningHome, ws.Master, ws.Specs, ws.Home)
			}
			if loc.Origin != tc.origin {
				t.Fatalf("Origin = %q, se esperaba %q", loc.Origin, tc.origin)
			}
			if loc.Config == nil || loc.Config.Workspace.Topology != TopologyMultirepo {
				t.Fatalf("Config = %+v, se esperaba la configuración multirepo del repositorio de especificaciones", loc.Config)
			}
			if want := filepath.Join(ws.Specs, "axiom.yaml"); loc.ConfigPath != want {
				t.Fatalf("ConfigPath = %q, se esperaba %q", loc.ConfigPath, want)
			}
			if len(loc.Notes) != 0 {
				t.Fatalf("Notes = %v, se esperaba ninguna", loc.Notes)
			}
		})
	}

	if after := treeSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("la resolución modificó el árbol:\nantes: %v\ndespués: %v", before, after)
	}
}

func TestLocateDoesNotCreateAPlanningHome(t *testing.T) {
	ws := workspacetest.Canonical(t)
	if err := os.RemoveAll(ws.Home); err != nil {
		t.Fatal(err)
	}
	before := treeSnapshot(t, ws.Master)
	loc := mustLocate(t, ws.Master)
	if loc.PlanningHome != ws.Home {
		t.Fatalf("PlanningHome = %q, se esperaba %q", loc.PlanningHome, ws.Home)
	}
	if after := treeSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("la resolución creó ficheros: antes %v, después %v", before, after)
	}
}

func TestLocateStartInvalid(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "fichero.txt")
	workspacetest.WriteFile(t, file, "x")
	root := filepath.VolumeName(dir) + string(filepath.Separator)

	for name, start := range map[string]string{
		"inexistente":          filepath.Join(dir, "no-existe"),
		"inexistente anidado":  filepath.Join(dir, "a", "b"),
		"inexistente relativo": filepath.Join("no", "existe", "nunca"),
		"no es un directorio":  file,
		"raíz del sistema":     root,
	} {
		t.Run(name, func(t *testing.T) {
			locateErr := wantLocateError(t, start, LocateStartInvalid)
			if !strings.Contains(locateErr.Continuation, "--cwd") {
				t.Fatalf("la continuación debe nombrar --cwd: %q", locateErr.Continuation)
			}
		})
	}
}

func TestLocatePrecedenceBetweenMarkersOfTheSameDirectory(t *testing.T) {
	t.Run("puntero gana a configuración y openspec", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: {name: ignorado, topology: monorepo-embedded}\nroles: {x: {name: x, repositories: [{path: .}]}}\n")
		loc := mustLocate(t, ws.Master)
		if loc.Origin != OriginPointer || loc.ConfigPath != filepath.Join(ws.Specs, "axiom.yaml") {
			t.Fatalf("Origin %q ConfigPath %q: la configuración efectiva debe ser la que apunta el puntero", loc.Origin, loc.ConfigPath)
		}
	})
	t.Run("configuración gana a openspec", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		loc := mustLocate(t, ws.Master)
		if loc.Origin != OriginConfig {
			t.Fatalf("Origin = %q, se esperaba %q", loc.Origin, OriginConfig)
		}
	})
	t.Run("solo openspec", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.MkdirAll(t, filepath.Join(dir, "openspec"))
		loc := mustLocate(t, dir)
		if loc.Origin != OriginOpenSpec || loc.WorkspaceRoot != dir || loc.PlanningHome != filepath.Join(dir, "openspec") || loc.MarkerDir != dir {
			t.Fatalf("Location = %+v, se esperaba el directorio con origen openspec", loc)
		}
		if loc.Config != nil || loc.ConfigPath != "" {
			t.Fatalf("sin axiom.yaml no hay configuración: %+v", loc)
		}
	})
}

func TestLocateSubdirectoryFallbackNeverActsAsMarker(t *testing.T) {
	master := t.TempDir()
	workspacetest.WriteFile(t, filepath.Join(master, "repo-specs", "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded}\n")
	loc := mustLocate(t, master)
	if loc.Origin != OriginStart || loc.WorkspaceRoot != master || loc.MarkerDir != "" {
		t.Fatalf("Location = %+v, el fallback de subdirectorios no debe actuar como marcador", loc)
	}
}

func TestLocateWithoutMarkersRootsAtTheStart(t *testing.T) {
	dir := t.TempDir()
	before := treeSnapshot(t, dir)
	loc := mustLocate(t, dir)
	if loc.Origin != OriginStart || loc.Master != dir || loc.WorkspaceRoot != dir || loc.PlanningHome != filepath.Join(dir, "openspec") || loc.Start != dir {
		t.Fatalf("Location = %+v", loc)
	}
	if after := treeSnapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Fatalf("la resolución escribió en disco: %v", after)
	}
}

func TestLocateNearestMarkerWins(t *testing.T) {
	t.Run("proyecto anidado frente al puntero", func(t *testing.T) {
		ws := workspacetest.Canonical(t, workspacetest.WithBackendOpenSpec())
		loc := mustLocate(t, filepath.Join(ws.Backend, "src"))
		if loc.Origin != OriginOpenSpec || loc.WorkspaceRoot != ws.Backend || loc.PlanningHome != filepath.Join(ws.Backend, "openspec") {
			t.Fatalf("Location = %+v, se esperaba el openspec propio de repo-backend", loc)
		}
	})
	t.Run("subdirectorio de un proyecto embebido", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		sub := filepath.Join(ws.Master, "internal", "pkg")
		workspacetest.MkdirAll(t, sub)
		loc := mustLocate(t, sub)
		if loc.WorkspaceRoot != ws.Master || loc.PlanningHome != ws.Home {
			t.Fatalf("Location = %+v, la raíz es la del proyecto y no el subdirectorio", loc)
		}
	})
	t.Run("el directorio de partida se evalúa antes que sus ancestros", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		sub := filepath.Join(ws.Master, "sub")
		workspacetest.MkdirAll(t, filepath.Join(sub, "openspec"))
		loc := mustLocate(t, sub)
		if loc.WorkspaceRoot != sub || loc.Origin != OriginOpenSpec {
			t.Fatalf("Location = %+v, se esperaba sub con origen openspec", loc)
		}
	})
}

func TestLocateEmbeddedLayoutIsInvariantUnderSpecsRepository(t *testing.T) {
	for _, specsRepository := range []string{"", ".", "openspec", "specs"} {
		t.Run("specs_repository="+specsRepository, func(t *testing.T) {
			ws := workspacetest.Embedded(t, specsRepository)
			loc := mustLocate(t, ws.Master)
			if loc.WorkspaceRoot != ws.Master || loc.PlanningHome != ws.Home || loc.Master != ws.Master {
				t.Fatalf("Location = %+v, el layout embebido debe ser <raíz>/openspec", loc)
			}
		})
	}
	t.Run("sin axiom.yaml", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.MkdirAll(t, filepath.Join(dir, "openspec", "changes"))
		loc := mustLocate(t, dir)
		if loc.WorkspaceRoot != dir || loc.PlanningHome != filepath.Join(dir, "openspec") {
			t.Fatalf("Location = %+v", loc)
		}
	})
	t.Run("directorio con el nombre de specs_repository", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "specs")
		workspacetest.WriteFile(t, filepath.Join(root, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded, specs_repository: specs}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
		workspacetest.MkdirAll(t, filepath.Join(root, "openspec"))
		loc := mustLocate(t, root)
		if loc.Master != root || loc.WorkspaceRoot != root {
			t.Fatalf("Location = %+v, en monorepo-embedded no se aplica la deducción por sufijo", loc)
		}
	})
	t.Run("puntero hacia el propio axiom.yaml embebido", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, ".axiom-workspace"), "config: axiom.yaml\n")
		loc := mustLocate(t, ws.Master)
		if loc.Origin != OriginPointer || loc.PlanningHome != ws.Home {
			t.Fatalf("Location = %+v", loc)
		}
	})
}

func TestLocateDeducesTheMasterFolderFromTheSpecsRepository(t *testing.T) {
	config := func(topology, specsRepository string) string {
		return "workspace: {name: x, topology: " + topology + ", specs_repository: " + specsRepository + "}\nroles: {a: {name: a, repositories: [{path: .}]}}\n"
	}
	t.Run("varios segmentos", func(t *testing.T) {
		master := t.TempDir()
		specs := filepath.Join(master, "repos", "specs")
		workspacetest.WriteFile(t, filepath.Join(specs, "axiom.yaml"), config("multirepo", "repos/specs"))
		loc := mustLocate(t, specs)
		if loc.Master != master || loc.WorkspaceRoot != specs {
			t.Fatalf("Location = %+v, la carpeta maestra es %s", loc, master)
		}
	})
	t.Run("separador invertido", func(t *testing.T) {
		master := t.TempDir()
		specs := filepath.Join(master, "repos", "specs")
		workspacetest.WriteFile(t, filepath.Join(specs, "axiom.yaml"), config("multirepo", `'repos\specs'`))
		if loc := mustLocate(t, specs); loc.Master != master {
			t.Fatalf("Master = %q, se esperaba %q: / y \\ son equivalentes", loc.Master, master)
		}
	})
	t.Run("el directorio contiene specs_repository", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "openspec")
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), config("multirepo", "openspec"))
		workspacetest.MkdirAll(t, filepath.Join(dir, "openspec"))
		loc := mustLocate(t, dir)
		if loc.Master != dir || loc.PlanningHome != filepath.Join(dir, "openspec") {
			t.Fatalf("Location = %+v", loc)
		}
	})
	t.Run("sin coincidencia de sufijo", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), config("multirepo", "especificacion"))
		workspacetest.MkdirAll(t, filepath.Join(dir, "especificacion"))
		loc := mustLocate(t, dir)
		if loc.Master != dir || loc.WorkspaceRoot != filepath.Join(dir, "especificacion") || loc.PlanningHome != filepath.Join(dir, "especificacion", "openspec") {
			t.Fatalf("Location = %+v", loc)
		}
	})
	t.Run("specs_repository vacío o punto", func(t *testing.T) {
		for _, value := range []string{`""`, `"."`} {
			dir := t.TempDir()
			workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), config("multirepo", value))
			loc := mustLocate(t, dir)
			if loc.Master != dir || loc.WorkspaceRoot != dir {
				t.Fatalf("specs_repository %s: Location = %+v", value, loc)
			}
		}
	})
	t.Run("workspace.root no interviene", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded, root: '..'}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
		if loc := mustLocate(t, ws.Master); loc.WorkspaceRoot != ws.Master {
			t.Fatalf("WorkspaceRoot = %q, workspace.root no debe intervenir", loc.WorkspaceRoot)
		}
	})
}

func TestLocateRootsByTopology(t *testing.T) {
	config := func(topology, specsRepository string) string {
		return "workspace: {name: x, topology: " + topology + ", specs_repository: " + specsRepository + "}\nroles: {a: {name: a, repositories: [{path: .}]}}\n"
	}
	tests := []struct {
		name            string
		topology        string
		specsRepository string
		mkdir           string
		wantRoot        string // relativo a la carpeta maestra
		wantHome        string
	}{
		{"multirepo con repositorio de especificaciones propio", "multirepo", "especificacion", "especificacion", "especificacion", "especificacion/openspec"},
		{"perfil knowledge terminado en openspec", "multirepo", "openspec", "openspec", ".", "openspec"},
		{"varios segmentos terminado en openspec", "multirepo", "docs/openspec", "docs", "docs", "docs/openspec"},
		{"openspec sin crear", "multirepo", "openspec", "", ".", "openspec"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			master := t.TempDir()
			workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), config(tc.topology, tc.specsRepository))
			if tc.mkdir != "" {
				workspacetest.MkdirAll(t, filepath.Join(master, tc.mkdir))
			}
			before := treeSnapshot(t, master)
			loc := mustLocate(t, master)
			if want := filepath.Join(master, filepath.FromSlash(tc.wantRoot)); loc.WorkspaceRoot != want {
				t.Fatalf("WorkspaceRoot = %q, se esperaba %q", loc.WorkspaceRoot, want)
			}
			if want := filepath.Join(master, filepath.FromSlash(tc.wantHome)); loc.PlanningHome != want {
				t.Fatalf("PlanningHome = %q, se esperaba %q", loc.PlanningHome, want)
			}
			if after := treeSnapshot(t, master); !reflect.DeepEqual(before, after) {
				t.Fatalf("la resolución escribió en disco: %v", after)
			}
		})
	}
}

func TestLocateAmbiguousPlanningHome(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithMasterOpenSpec())
	before := treeSnapshot(t, ws.Master)
	for name, start := range map[string]string{
		"carpeta maestra":                 ws.Master,
		"repositorio de especificaciones": ws.Specs,
		"repositorio de código":           filepath.Join(ws.Backend, "src"),
	} {
		t.Run(name, func(t *testing.T) {
			locateErr := wantLocateError(t, start, LocateAmbiguousPlanningHome)
			message := locateErr.Error()
			for _, want := range []string{filepath.Join(ws.Master, "openspec"), ws.Home, WorkspacePointerFilename} {
				if !strings.Contains(message, want) {
					t.Fatalf("el rechazo debe nombrar %q: %s", want, message)
				}
			}
		})
	}
	if after := treeSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("el rechazo escribió en disco: %v", after)
	}

	t.Run("el mismo planning home no es ambiguo", func(t *testing.T) {
		master := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(master, ".axiom-workspace"), "config: axiom.yaml\n")
		workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), "workspace: {name: x, topology: multirepo, specs_repository: openspec}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
		workspacetest.MkdirAll(t, filepath.Join(master, "openspec"))
		if loc := mustLocate(t, master); loc.PlanningHome != filepath.Join(master, "openspec") {
			t.Fatalf("PlanningHome = %q", loc.PlanningHome)
		}
	})
	t.Run("sin puntero no hay ambigüedad", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		if loc := mustLocate(t, ws.Master); len(loc.Notes) != 0 {
			t.Fatalf("Notes = %v", loc.Notes)
		}
	})
}

func TestLocateRejectsUnresolvableConfiguration(t *testing.T) {
	t.Run("puntero roto", func(t *testing.T) {
		ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
		before := treeSnapshot(t, ws.Master)
		locateErr := wantLocateError(t, ws.Master, LocatePointerBroken)
		for _, want := range []string{filepath.Join(ws.Master, ".axiom-workspace"), "repo-inexistente"} {
			if !strings.Contains(locateErr.Error(), want) {
				t.Fatalf("el rechazo debe nombrar %q: %s", want, locateErr)
			}
		}
		if after := treeSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
			t.Fatalf("el rechazo escribió en disco: %v", after)
		}
	})
	t.Run("puntero sin config ni specs", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, ".axiom-workspace"), "")
		locateErr := wantLocateError(t, dir, LocatePointerBroken)
		if !strings.Contains(locateErr.Error(), "config") || !strings.Contains(locateErr.Error(), "specs") {
			t.Fatalf("el rechazo debe indicar que el puntero define config o specs: %s", locateErr)
		}
	})
	t.Run("destino con YAML ilegible", func(t *testing.T) {
		ws := workspacetest.Canonical(t, workspacetest.WithUnreadableConfig())
		locateErr := wantLocateError(t, ws.Master, LocateConfigUnloadable)
		if !strings.Contains(locateErr.Continuation, "axiom workspace validate") {
			t.Fatalf("la continuación debe nombrar axiom workspace validate: %q", locateErr.Continuation)
		}
	})
	t.Run("destino con base inválida", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), "workspace: {name: x, topology: desconocida}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
		locateErr := wantLocateError(t, ws.Master, LocateConfigUnloadable)
		if !strings.Contains(locateErr.Error(), "desconocida") || !strings.Contains(locateErr.Error(), "axiom workspace validate") {
			t.Fatalf("el rechazo debe nombrar la topología inválida y axiom workspace validate: %s", locateErr)
		}
	})
	t.Run("repositorio de especificaciones inexistente", func(t *testing.T) {
		ws := workspacetest.Canonical(t, workspacetest.WithoutSpecsRepository())
		locateErr := wantLocateError(t, ws.Master, LocateSpecsRepositoryMissing)
		if !strings.Contains(locateErr.Error(), ws.Specs) || !strings.Contains(locateErr.Error(), "specs_repository") {
			t.Fatalf("el rechazo debe nombrar %s y specs_repository: %s", ws.Specs, locateErr)
		}
	})
	t.Run("axiom.yaml directo sin repositorio de especificaciones", func(t *testing.T) {
		master := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), "workspace: {name: x, topology: multirepo, specs_repository: repo-specs}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
		locateErr := wantLocateError(t, master, LocateSpecsRepositoryMissing)
		if !strings.Contains(locateErr.Error(), filepath.Join(master, "repo-specs")) {
			t.Fatalf("el rechazo debe nombrar la ruta esperada: %s", locateErr)
		}
	})
}

func TestLocateKeepsTheEmbeddedResolutionForAnUnloadableDirectConfig(t *testing.T) {
	dir := t.TempDir()
	workspacetest.MkdirAll(t, filepath.Join(dir, "openspec"))
	workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded}\n")

	loc := mustLocate(t, dir)
	if loc.WorkspaceRoot != dir || loc.PlanningHome != filepath.Join(dir, "openspec") || loc.Origin != OriginConfig {
		t.Fatalf("Location = %+v, se conserva la resolución embebida", loc)
	}
	if loc.Config != nil || loc.ConfigErr == nil {
		t.Fatalf("Config = %+v ConfigErr = %v, se esperaba configuración nula y causa", loc.Config, loc.ConfigErr)
	}
	if len(loc.Notes) != 1 || !strings.Contains(loc.Notes[0], "axiom workspace validate") || !strings.Contains(loc.Notes[0], "roles") {
		t.Fatalf("Notes = %v, se esperaba un aviso que nombre la causa y axiom workspace validate", loc.Notes)
	}
}

func retiredTopologyConfig(specsRepository string) string {
	return "workspace: {name: x, topology: monorepo-decoupled, specs_repository: " + specsRepository + "}\nroles: {a: {name: a, repositories: [{path: .}]}}\n"
}

// workspace-topology REQ-25.25 (escenario «Un puntero hacia una configuración con
// la topología retirada rechaza la resolución») y planning-home-resolution
// REQ-24.7 (caso 2): la configuración base no se carga, así que el planning home
// es desconocido y no hay resolución que devolver.
func TestLocateRejectsAPointerToARetiredTopology(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), retiredTopologyConfig("repo-specs"))
	before := treeSnapshot(t, ws.Master)

	locateErr := wantLocateError(t, ws.Master, LocateConfigUnloadable)
	for _, want := range []string{"`monorepo-decoupled` fue retirada", "`multirepo`", "axiom workspace validate"} {
		if !strings.Contains(locateErr.Error(), want) {
			t.Errorf("el rechazo debe contener %q: %s", want, locateErr)
		}
	}
	if after := treeSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("el rechazo escribió en disco: %v", after)
	}
}

// REQ-24.7 (caso 3) y REQ-25.25: un axiom.yaml directo con la topología retirada
// no se rechaza. Conserva la resolución embebida y avisa, sin darle la semántica
// de repositorio de especificaciones separado que tenía antes (REQ-24.3).
func TestLocateKeepsTheEmbeddedResolutionForADirectConfigWithARetiredTopology(t *testing.T) {
	master := t.TempDir()
	workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), retiredTopologyConfig("especificacion"))
	workspacetest.MkdirAll(t, filepath.Join(master, "especificacion"))

	loc := mustLocate(t, master)
	if loc.WorkspaceRoot != master || loc.PlanningHome != filepath.Join(master, "openspec") || loc.Master != master {
		t.Fatalf("Location = %+v, se esperaba la resolución embebida y no el repositorio especificacion", loc)
	}
	if loc.Config != nil || loc.ConfigErr == nil {
		t.Fatalf("Config = %+v ConfigErr = %v, se esperaba configuración nula y la causa", loc.Config, loc.ConfigErr)
	}
	if len(loc.Notes) != 1 {
		t.Fatalf("Notes = %v, se esperaba un único aviso", loc.Notes)
	}
	for _, want := range []string{"`monorepo-decoupled` fue retirada", "`multirepo`", "axiom workspace validate"} {
		if !strings.Contains(loc.Notes[0], want) {
			t.Errorf("el aviso debe contener %q: %s", want, loc.Notes[0])
		}
	}
}

// REQ-24.3 y REQ-24.5: el enraizado por repositorio de especificaciones y la
// deducción de la carpeta maestra por sufijo rigen solo para multirepo. Un valor
// retirado que llegara a una configuración construida a mano tampoco los recibe.
func TestOnlyMultirepoSeparatesTheSpecsRepository(t *testing.T) {
	master := filepath.Join(string(filepath.Separator), "w")
	dir := filepath.Join(master, "especificacion")
	for _, tc := range []struct {
		name         string
		topology     TopologyType
		wantSeparate bool
	}{
		{"multirepo", TopologyMultirepo, true},
		{"embebido", TopologyMonorepoEmbedded, false},
		{"valor retirado", TopologyType("monorepo-decoupled"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &WorkspaceConfig{Workspace: WorkspaceSection{Topology: tc.topology, SpecsRepository: "especificacion"}}
			if got := separatesSpecs(cfg); got != tc.wantSeparate {
				t.Fatalf("separatesSpecs = %v, se esperaba %v", got, tc.wantSeparate)
			}
			root, home := Roots(cfg, master)
			wantRoot, wantHome := master, filepath.Join(master, "openspec")
			if tc.wantSeparate {
				wantRoot, wantHome = dir, filepath.Join(dir, "openspec")
			}
			if root != wantRoot || home != wantHome {
				t.Fatalf("Roots = (%q, %q), se esperaba (%q, %q)", root, home, wantRoot, wantHome)
			}
			wantMaster := dir
			if tc.wantSeparate {
				wantMaster = master
			}
			if got := deduceMaster(cfg, dir); got != wantMaster {
				t.Fatalf("deduceMaster = %q, se esperaba %q", got, wantMaster)
			}
		})
	}
}

func TestLocateWarnsAboutADivergentMasterOpenSpecWithoutPointer(t *testing.T) {
	master := t.TempDir()
	workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), "workspace: {name: x, topology: multirepo, specs_repository: especificacion}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
	workspacetest.MkdirAll(t, filepath.Join(master, "especificacion"))
	workspacetest.MkdirAll(t, filepath.Join(master, "openspec", "changes", "antiguo"))

	loc := mustLocate(t, master)
	wantHome := filepath.Join(master, "especificacion", "openspec")
	if loc.PlanningHome != wantHome {
		t.Fatalf("PlanningHome = %q, se esperaba %q", loc.PlanningHome, wantHome)
	}
	if len(loc.Notes) != 1 || !strings.Contains(loc.Notes[0], filepath.Join(master, "openspec")) || !strings.Contains(loc.Notes[0], wantHome) {
		t.Fatalf("Notes = %v, se esperaba un aviso que nombre las dos rutas", loc.Notes)
	}
}

func TestLocateHonoursTheTestCeiling(t *testing.T) {
	outer := t.TempDir()
	workspacetest.MkdirAll(t, filepath.Join(outer, "openspec"))
	inner := filepath.Join(outer, "proyecto", "sub")
	workspacetest.MkdirAll(t, inner)

	// El techo de TestMain es el directorio temporal del sistema, por encima de
	// outer: el openspec de outer sí es el marcador más cercano.
	if loc := mustLocate(t, inner); loc.Origin != OriginOpenSpec || loc.WorkspaceRoot != outer {
		t.Fatalf("Location = %+v, el openspec del ancestro debería ser el marcador", loc)
	}

	previous := DiscoveryCeilingOverride
	t.Cleanup(func() { DiscoveryCeilingOverride = previous })
	DiscoveryCeilingOverride = []string{outer}
	if loc := mustLocate(t, inner); loc.Origin != OriginStart || loc.WorkspaceRoot != inner {
		t.Fatalf("Location = %+v, el techo impide evaluar el directorio del techo y sus ancestros", loc)
	}
}

func TestRoots(t *testing.T) {
	master := filepath.Join(string(filepath.Separator), "w")
	abs := filepath.Join(t.TempDir(), "specs-externos")
	cfg := func(topology TopologyType, specsRepository string) *WorkspaceConfig {
		return &WorkspaceConfig{Workspace: WorkspaceSection{Topology: topology, SpecsRepository: specsRepository}}
	}
	tests := []struct {
		name     string
		cfg      *WorkspaceConfig
		wantRoot string
		wantHome string
	}{
		{"sin configuración", nil, master, filepath.Join(master, "openspec")},
		{"embebido ignora specs_repository", cfg(TopologyMonorepoEmbedded, "specs"), master, filepath.Join(master, "openspec")},
		{"multirepo con repositorio", cfg(TopologyMultirepo, "repo-specs"), filepath.Join(master, "repo-specs"), filepath.Join(master, "repo-specs", "openspec")},
		{"multirepo con varios segmentos terminado en openspec", cfg(TopologyMultirepo, "docs/openspec"), filepath.Join(master, "docs"), filepath.Join(master, "docs", "openspec")},
		{"vacío es la maestra", cfg(TopologyMultirepo, ""), master, filepath.Join(master, "openspec")},
		{"punto es la maestra", cfg(TopologyMultirepo, "."), master, filepath.Join(master, "openspec")},
		{"ruta absoluta", cfg(TopologyMultirepo, abs), abs, filepath.Join(abs, "openspec")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, home := Roots(tc.cfg, master)
			if root != tc.wantRoot || home != tc.wantHome {
				t.Fatalf("Roots = (%q, %q), se esperaba (%q, %q)", root, home, tc.wantRoot, tc.wantHome)
			}
			if home != filepath.Join(root, "openspec") {
				t.Fatalf("invariante B' rota: %q no es %q/openspec", home, root)
			}
		})
	}
}

func TestResolveOwnPointer(t *testing.T) {
	t.Run("sin puntero no hay resolución", func(t *testing.T) {
		dir := t.TempDir()
		if _, found, err := ResolveOwnPointer(dir); found || err != nil {
			t.Fatalf("found = %v err = %v, se esperaba (false, nil)", found, err)
		}
	})
	t.Run("no asciende", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		if _, found, err := ResolveOwnPointer(filepath.Join(ws.Backend, "src")); found || err != nil {
			t.Fatalf("found = %v err = %v: el puntero de un ancestro no cuenta", found, err)
		}
	})
	t.Run("resuelve el puntero propio", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		loc, found, err := ResolveOwnPointer(ws.Master)
		if err != nil || !found {
			t.Fatalf("found = %v err = %v", found, err)
		}
		if loc.Master != ws.Master || loc.WorkspaceRoot != ws.Specs || loc.PlanningHome != ws.Home || loc.Origin != OriginPointer {
			t.Fatalf("Location = %+v", loc)
		}
	})
	t.Run("planning home inexistente no es error", func(t *testing.T) {
		master := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(master, ".axiom-workspace"), "config: axiom.yaml\n")
		workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), "workspace: {name: x, topology: multirepo, specs_repository: openspec}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
		loc, found, err := ResolveOwnPointer(master)
		if err != nil || !found || loc.PlanningHome != filepath.Join(master, "openspec") {
			t.Fatalf("Location = %+v found = %v err = %v", loc, found, err)
		}
		if _, statErr := os.Stat(filepath.Join(master, "openspec")); !errors.Is(statErr, fs.ErrNotExist) {
			t.Fatalf("la resolución creó openspec/: %v", statErr)
		}
	})
	t.Run("rechazos tipados", func(t *testing.T) {
		broken := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
		_, found, err := ResolveOwnPointer(broken.Master)
		var locateErr *LocateError
		if !found || !errors.As(err, &locateErr) || locateErr.Kind != LocatePointerBroken {
			t.Fatalf("found = %v err = %v, se esperaba pointer_broken", found, err)
		}
		ambiguous := workspacetest.Canonical(t, workspacetest.WithMasterOpenSpec())
		_, _, err = ResolveOwnPointer(ambiguous.Master)
		if !errors.As(err, &locateErr) || locateErr.Kind != LocateAmbiguousPlanningHome {
			t.Fatalf("err = %v, se esperaba ambiguous_planning_home", err)
		}
	})
}

func TestLocateAtEvaluatesOnlyTheGivenDirectory(t *testing.T) {
	ws := workspacetest.Canonical(t)
	t.Run("raíz explícita del espacio canónico", func(t *testing.T) {
		loc, err := LocateAt(ws.Specs)
		if err != nil {
			t.Fatal(err)
		}
		if loc.WorkspaceRoot != ws.Specs || loc.PlanningHome != ws.Home || loc.Master != ws.Master {
			t.Fatalf("Location = %+v", loc)
		}
	})
	t.Run("sin marcador no asciende", func(t *testing.T) {
		start := filepath.Join(ws.Backend, "src")
		loc, err := LocateAt(start)
		if err != nil {
			t.Fatal(err)
		}
		if loc.Origin != OriginStart || loc.Master != start || loc.WorkspaceRoot != start {
			t.Fatalf("Location = %+v, LocateAt no debe ascender hasta el puntero", loc)
		}
	})
	t.Run("directorio inválido", func(t *testing.T) {
		if _, err := LocateAt(filepath.Join(ws.Master, "no-existe")); err == nil {
			t.Fatal("se esperaba el rechazo start_invalid")
		}
	})
}

func TestLocationLegacySpecsDivergence(t *testing.T) {
	// legacyDir es la ubicación antigua <raíz>/<specs_repository>/specs.
	tests := []struct {
		name            string
		specsRepository string
		legacyDir       string // relativo a la raíz; "" no crea ubicación antigua
		wantWarning     bool
	}{
		{name: "specs_repository vacío", specsRepository: "", legacyDir: filepath.Join("specs")},
		{name: "specs_repository punto", specsRepository: ".", legacyDir: filepath.Join("specs")},
		{name: "specs_repository openspec", specsRepository: "openspec", legacyDir: filepath.Join("openspec", "specs")},
		{name: "specs_repository ./openspec", specsRepository: "./openspec", legacyDir: filepath.Join("openspec", "specs")},
		{name: "specs divergente con ubicación antigua", specsRepository: "specs", legacyDir: filepath.Join("specs", "specs"), wantWarning: true},
		{name: "specs con barra invertida y subcarpeta", specsRepository: `docs\specs`, legacyDir: filepath.Join("docs", "specs", "specs"), wantWarning: true},
		{name: "specs divergente sin ubicación antigua", specsRepository: "specs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := workspacetest.Embedded(t, tt.specsRepository)
			if tt.legacyDir != "" {
				workspacetest.MkdirAll(t, filepath.Join(ws.Master, tt.legacyDir))
			}
			loc := mustLocate(t, ws.Master)
			warning, ok := loc.LegacySpecsDivergence()
			if ok != tt.wantWarning {
				t.Fatalf("LegacySpecsDivergence() = (%q, %v), se esperaba aviso=%v", warning, ok, tt.wantWarning)
			}
			if !ok {
				if warning != "" {
					t.Fatalf("sin aviso, el texto debe estar vacío: %q", warning)
				}
				return
			}
			legacy := filepath.Join(ws.Master, tt.legacyDir)
			current := filepath.Join(ws.Home, "specs")
			for _, want := range []string{legacy, current, "workspace.specs_repository"} {
				if !strings.Contains(warning, want) {
					t.Errorf("el aviso debe nombrar %q y la acción: %q", want, warning)
				}
			}
		})
	}

	t.Run("solo en monorepo-embedded", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.MkdirAll(t, filepath.Join(ws.Specs, "repo-specs", "specs"))
		loc := mustLocate(t, ws.Master)
		if warning, ok := loc.LegacySpecsDivergence(); ok {
			t.Fatalf("el multirepo no emite el aviso Q14: %q", warning)
		}
	})
	t.Run("sin configuración cargada", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.MkdirAll(t, filepath.Join(dir, "openspec", "changes"))
		workspacetest.MkdirAll(t, filepath.Join(dir, "specs", "specs"))
		if warning, ok := mustLocate(t, dir).LegacySpecsDivergence(); ok {
			t.Fatalf("sin axiom.yaml no hay specs_repository que divergir: %q", warning)
		}
	})
	t.Run("ruta equivalente a openspec tras normalizar", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "./openspec/../openspec")
		workspacetest.MkdirAll(t, filepath.Join(ws.Home, "specs"))
		if warning, ok := mustLocate(t, ws.Master).LegacySpecsDivergence(); ok {
			t.Fatalf("la misma carpeta no diverge de sí misma: %q", warning)
		}
	})
}

const (
	// Declaración inválida (baseline sin documentos) con base válida.
	locateInvalidBaseline = "baseline:\n  documents: []\n"
	// Declaración válida de un documento de prosa.
	locateValidBaseline = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n"
	// Base inválida: sin roles.
	locateInvalidBase = "workspace: {name: x, topology: monorepo-embedded}\n"

	locateCanonicalConfig = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n  frontend:\n    name: Frontend\n    repositories:\n      - path: repo-frontend\n"
	locateEmbeddedConfig  = "workspace:\n  name: ejemplo-embebido\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
)

// REQ-24.18 y D-03: Locate rellena las declaraciones con la carga por capas y,
// con una base válida, las declaraciones inválidas nunca hacen rechazar.
func TestLocateFillsTheDeclarations(t *testing.T) {
	t.Run("puntero con declaraciones validas", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateCanonicalConfig+locateValidBaseline+"technical_context:\n  reference:\n    paths: [00_Glosario.md]\n")
		for name, start := range map[string]string{"carpeta maestra": ws.Master, "repositorio de especificaciones": ws.Specs} {
			loc := mustLocate(t, start)
			if loc.Declarations.BaselineErr != nil || loc.Declarations.TechnicalContextErr != nil || loc.Declarations.ParseErr != nil {
				t.Fatalf("%s: declaraciones válidas con error: %+v", name, loc.Declarations)
			}
			if loc.Declarations.Baseline == nil || len(loc.Declarations.Baseline.Documents) != 1 || loc.Declarations.TechnicalContext == nil {
				t.Fatalf("%s: Declarations = %+v, se esperaban los dos bloques", name, loc.Declarations)
			}
			if loc.Config == nil || loc.Config.Baseline != loc.Declarations.Baseline {
				t.Fatalf("%s: la configuración base debe llevar el mismo bloque válido", name)
			}
		}
	})

	t.Run("sin bloques nuevos son el valor cero", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		loc := mustLocate(t, ws.Master)
		if loc.Declarations.ParseErr != nil || loc.Declarations.Baseline != nil || loc.Declarations.BaselineErr != nil ||
			loc.Declarations.TechnicalContext != nil || loc.Declarations.TechnicalContextErr != nil {
			t.Fatalf("Declarations = %+v, se esperaba el valor cero", loc.Declarations)
		}
	})

	t.Run("sin axiom.yaml ni puntero son el valor cero", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.MkdirAll(t, filepath.Join(dir, "openspec"))
		for _, start := range []string{dir, t.TempDir()} {
			loc := mustLocate(t, start)
			if loc.Declarations.ParseErr != nil || loc.Declarations.Baseline != nil || loc.Declarations.BaselineErr != nil {
				t.Fatalf("Declarations = %+v, sin axiom.yaml se esperaba el valor cero", loc.Declarations)
			}
		}
	})

	t.Run("puntero con declaracion invalida y base valida no rechaza", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateCanonicalConfig+locateInvalidBaseline+"technical_context: {}\n")
		before := treeSnapshot(t, ws.Master)
		loc := mustLocate(t, ws.Master)
		if loc.Master != ws.Master || loc.WorkspaceRoot != ws.Specs || loc.PlanningHome != ws.Home {
			t.Fatalf("la resolución no cambia con una declaración inválida: %+v", loc)
		}
		if loc.Declarations.BaselineErr == nil || loc.Declarations.TechnicalContextErr == nil {
			t.Fatalf("las dos declaraciones inválidas deben quedar en su canal: %+v", loc.Declarations)
		}
		if loc.Declarations.Baseline != nil || loc.Declarations.TechnicalContext != nil || loc.Declarations.ParseErr != nil {
			t.Fatalf("un bloque inválido no se proyecta y ParseErr es solo para el YAML ilegible: %+v", loc.Declarations)
		}
		if loc.Config == nil || len(loc.Config.Roles) != 2 || loc.Config.Baseline != nil {
			t.Fatalf("los roles siguen disponibles en la configuración base: %+v", loc.Config)
		}
		if len(loc.Notes) != 0 {
			t.Fatalf("una declaración inválida no genera aviso de resolución: %v", loc.Notes)
		}
		if after := treeSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
			t.Fatalf("Locate escribió en disco: %v", after)
		}
	})

	t.Run("puntero con base invalida rechaza aunque la declaracion sea valida", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateInvalidBase+locateValidBaseline)
		locateErr := wantLocateError(t, ws.Master, LocateConfigUnloadable)
		if !strings.Contains(locateErr.Continuation, "axiom workspace validate") {
			t.Fatalf("la continuación debe nombrar axiom workspace validate: %q", locateErr.Continuation)
		}
	})

	t.Run("puntero con declaracion invalida y base invalida rechaza por la base", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateInvalidBase+locateInvalidBaseline)
		locateErr := wantLocateError(t, ws.Master, LocateConfigUnloadable)
		if !strings.Contains(locateErr.Error(), "roles") {
			t.Fatalf("el rechazo debe nombrar la causa de la base: %s", locateErr)
		}
	})

	t.Run("sin puntero la base invalida conserva la resolucion embebida y evalua la declaracion", func(t *testing.T) {
		for name, tc := range map[string]struct {
			baseline string
			valid    bool
		}{"declaracion valida": {locateValidBaseline, true}, "declaracion invalida": {locateInvalidBaseline, false}} {
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				workspacetest.MkdirAll(t, filepath.Join(dir, "openspec"))
				workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), locateInvalidBase+tc.baseline)
				loc := mustLocate(t, dir)
				if loc.WorkspaceRoot != dir || loc.Origin != OriginConfig || loc.Config != nil || loc.ConfigErr == nil || len(loc.Notes) != 1 {
					t.Fatalf("se conserva la resolución embebida con aviso: %+v", loc)
				}
				if loc.Declarations.ParseErr != nil {
					t.Fatalf("el YAML es legible: ParseErr = %v", loc.Declarations.ParseErr)
				}
				if tc.valid != (loc.Declarations.Baseline != nil) || tc.valid != (loc.Declarations.BaselineErr == nil) {
					t.Fatalf("la declaración se evalúa aunque la base falle (válida=%v): %+v", tc.valid, loc.Declarations)
				}
			})
		}
	})

	t.Run("sin puntero un YAML ilegible deja la causa en ParseErr", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.MkdirAll(t, filepath.Join(dir, "openspec"))
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), "workspace: [sin cerrar\n")
		loc := mustLocate(t, dir)
		if loc.Declarations.ParseErr == nil || loc.ConfigErr == nil || loc.Config != nil || len(loc.Notes) != 1 {
			t.Fatalf("un YAML ilegible se resuelve embebido con aviso y ParseErr: %+v", loc)
		}
	})

	t.Run("sin puntero una declaracion invalida con base valida no rechaza", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+locateInvalidBaseline)
		loc := mustLocate(t, ws.Master)
		if loc.Config == nil || loc.ConfigErr != nil || loc.Declarations.BaselineErr == nil || len(loc.Notes) != 0 {
			t.Fatalf("la base válida carga y la declaración inválida queda en su canal: %+v", loc)
		}
	})
}

// evalTempDir devuelve un directorio temporal sin enlaces simbólicos ni nombres
// cortos 8.3, de modo que las rutas esperadas coinciden con las que produce la
// resolución.
func evalTempDir(t *testing.T) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolver el directorio temporal: %v", err)
	}
	return resolved
}

// linkWorktree convierte worktree en un worktree enlazado del repositorio cuyo
// common dir es commonDir, solo con ficheros (sin lanzar git): su `.git` es un
// fichero `gitdir:` y su git dir, `<common dir>/worktrees/<name>`, contiene el
// fichero `commondir`. Es la misma evidencia que lee gitcommon.LinkedWorktreeRoot.
func linkWorktree(t *testing.T, commonDir, worktree, name string) {
	t.Helper()
	gitDir := filepath.Join(commonDir, "worktrees", name)
	workspacetest.WriteFile(t, filepath.Join(gitDir, "commondir"), "../..\n")
	workspacetest.WriteFile(t, filepath.Join(worktree, ".git"), "gitdir: "+filepath.ToSlash(gitDir)+"\n")
}

// linkWorktreeOfCheckout monta un worktree enlazado cuyo checkout principal es
// main (con un `.git` directorio, como un clon normal).
func linkWorktreeOfCheckout(t *testing.T, main, worktree, name string) {
	t.Helper()
	linkWorktree(t, filepath.Join(main, ".git"), worktree, name)
}

// REQ-25.28 (escenario «Un worktree de un monorepo embebido resuelve su propio
// planning home»): con marcador propio dentro del worktree, gana el más cercano y
// el planning home es el del worktree, no el del checkout principal.
func TestLocateALinkedWorktreeOfAnEmbeddedMonorepoResolvesItsOwnPlanningHome(t *testing.T) {
	main := workspacetest.Embedded(t, "openspec")
	worktree := filepath.Join(evalTempDir(t), "x", "repo")
	workspacetest.WriteFile(t, filepath.Join(worktree, "axiom.yaml"), embeddedConfigOf(t, main.Master))
	workspacetest.MkdirAll(t, filepath.Join(worktree, "openspec", "changes", "x"))
	workspacetest.MkdirAll(t, filepath.Join(worktree, "internal"))
	linkWorktreeOfCheckout(t, main.Master, worktree, "x")

	loc := mustLocate(t, filepath.Join(worktree, "internal"))
	if loc.WorkspaceRoot != worktree || loc.PlanningHome != filepath.Join(worktree, "openspec") || loc.Master != worktree {
		t.Fatalf("Location = master %q root %q home %q; se esperaba el propio worktree %q", loc.Master, loc.WorkspaceRoot, loc.PlanningHome, worktree)
	}
	if loc.Origin != OriginConfig || loc.Config == nil || loc.ConfigPath != filepath.Join(worktree, "axiom.yaml") {
		t.Fatalf("Origin %q ConfigPath %q: se esperaba el axiom.yaml del propio worktree", loc.Origin, loc.ConfigPath)
	}
	if len(loc.Notes) != 0 {
		t.Fatalf("con marcador propio no hay nota: %v", loc.Notes)
	}
}

// embeddedConfigOf devuelve el contenido del axiom.yaml del checkout principal,
// que el worktree versiona igual.
func embeddedConfigOf(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "axiom.yaml"))
	if err != nil {
		t.Fatalf("leer el axiom.yaml de %s: %v", root, err)
	}
	return string(data)
}

// REQ-25.28 (escenario «Un worktree de un repositorio de código de un multirepo
// resuelve el planning home de la carpeta maestra»): sin marcador en el worktree
// se resuelve como desde el checkout principal, y la ascensión no sale del
// worktree: un marcador en el directorio que lo contiene no cuenta.
func TestLocateALinkedWorktreeWithoutMarkerResolvesFromTheMainCheckout(t *testing.T) {
	ws := workspacetest.Canonical(t)
	outside := evalTempDir(t)
	worktree := filepath.Join(outside, "x", "repo-backend")
	workspacetest.MkdirAll(t, filepath.Join(worktree, "src"))
	linkWorktreeOfCheckout(t, ws.Backend, worktree, "x")
	workspacetest.MkdirAll(t, filepath.Join(ws.Backend, ".git"))
	// Un marcador señuelo fuera del worktree, en su directorio padre.
	workspacetest.MkdirAll(t, filepath.Join(outside, "x", "openspec"))

	before := treeSnapshot(t, outside)
	beforeMaster := treeSnapshot(t, ws.Master)
	direct := mustLocate(t, ws.Backend)
	for _, start := range []string{worktree, filepath.Join(worktree, "src")} {
		loc := mustLocate(t, start)
		if loc.Master != ws.Master || loc.WorkspaceRoot != ws.Specs || loc.PlanningHome != ws.Home {
			t.Fatalf("desde %s: master %q root %q home %q; se esperaba %q %q %q", start, loc.Master, loc.WorkspaceRoot, loc.PlanningHome, ws.Master, ws.Specs, ws.Home)
		}
		if loc.Origin != direct.Origin || loc.ConfigPath != direct.ConfigPath || loc.MarkerDir != direct.MarkerDir {
			t.Fatalf("desde %s la resolución difiere de la del checkout principal: %+v frente a %+v", start, loc, direct)
		}
		if loc.Start != start {
			t.Fatalf("Start = %q, se esperaba el directorio de partida %q", loc.Start, start)
		}
		if len(loc.Notes) != 1 || !strings.Contains(loc.Notes[0], ws.Backend) {
			t.Fatalf("Notes = %v, se esperaba una nota que nombre el checkout principal %q", loc.Notes, ws.Backend)
		}
	}
	if after := treeSnapshot(t, outside); !reflect.DeepEqual(before, after) {
		t.Fatalf("la resolución modificó el worktree: %v", after)
	}
	if after := treeSnapshot(t, ws.Master); !reflect.DeepEqual(beforeMaster, after) {
		t.Fatalf("la resolución modificó la carpeta maestra: %v", after)
	}
}

// REQ-25.28 (escenario «Sin marcador ni siquiera en el checkout principal»).
func TestLocateALinkedWorktreeWithoutMarkerAnywhereRootsAtTheStart(t *testing.T) {
	main := evalTempDir(t)
	worktree := filepath.Join(evalTempDir(t), "x")
	workspacetest.MkdirAll(t, worktree)
	linkWorktreeOfCheckout(t, main, worktree, "x")

	loc := mustLocate(t, worktree)
	if loc.Origin != OriginStart || loc.Master != worktree || loc.WorkspaceRoot != worktree || loc.PlanningHome != filepath.Join(worktree, "openspec") || loc.MarkerDir != "" {
		t.Fatalf("Location = %+v, se esperaba el directorio de partida con origen start", loc)
	}
	if len(loc.Notes) != 1 || !strings.Contains(loc.Notes[0], main) {
		t.Fatalf("Notes = %v, se esperaba una nota que nombre el checkout principal %q", loc.Notes, main)
	}
}

// D-15 punto 3: con un common dir que no se llama `.git` el checkout principal no
// se deduce de los nombres, y la resolución es OriginStart con nota aunque el
// repositorio tenga marcadores.
func TestLocateALinkedWorktreeWhoseCommonDirIsNotNamedGitRootsAtTheStart(t *testing.T) {
	commonDir := filepath.Join(evalTempDir(t), "separado.git")
	worktree := filepath.Join(evalTempDir(t), "x")
	workspacetest.MkdirAll(t, worktree)
	linkWorktree(t, commonDir, worktree, "x")

	loc := mustLocate(t, worktree)
	if loc.Origin != OriginStart || loc.WorkspaceRoot != worktree || loc.MarkerDir != "" {
		t.Fatalf("Location = %+v, se esperaba el directorio de partida con origen start", loc)
	}
	if len(loc.Notes) != 1 || !strings.Contains(loc.Notes[0], commonDir) {
		t.Fatalf("Notes = %v, se esperaba una nota que nombre el common dir %q", loc.Notes, commonDir)
	}
}

// REQ-25.28.3: la ubicación del worktree nunca alimenta a deduceMaster. El worktree
// se llama como el repositorio de especificaciones (el sufijo que deduceMaster
// retiraría de su ruta) y aun así la carpeta maestra es la del checkout principal.
func TestLocateALinkedWorktreeNeverMovesTheMasterFolder(t *testing.T) {
	master := evalTempDir(t)
	workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"),
		"workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n")
	workspacetest.MkdirAll(t, filepath.Join(master, "repo-specs"))
	backend := filepath.Join(master, "repo-backend")
	workspacetest.MkdirAll(t, filepath.Join(backend, ".git"))
	worktree := filepath.Join(evalTempDir(t), "wt", "repo-specs")
	workspacetest.MkdirAll(t, worktree)
	linkWorktreeOfCheckout(t, backend, worktree, "x")

	loc := mustLocate(t, worktree)
	if loc.Master != master || loc.WorkspaceRoot != filepath.Join(master, "repo-specs") {
		t.Fatalf("master %q root %q; se esperaba la carpeta maestra %q del checkout principal", loc.Master, loc.WorkspaceRoot, master)
	}
	if loc.Origin != OriginConfig || loc.Config == nil {
		t.Fatalf("Location = %+v, se esperaba el axiom.yaml directo de la carpeta maestra", loc)
	}
}

// REQ-25.28: un checkout principal, un directorio sin git y LocateAt no cambian.
func TestLocateKeepsItsBehaviourOutsideLinkedWorktrees(t *testing.T) {
	t.Run("checkout principal con marcador", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		workspacetest.MkdirAll(t, filepath.Join(ws.Master, ".git"))
		loc := mustLocate(t, ws.Master)
		if loc.Origin != OriginConfig || loc.WorkspaceRoot != ws.Master || len(loc.Notes) != 0 {
			t.Fatalf("Location = %+v", loc)
		}
	})
	t.Run("submódulo sin commondir no es un worktree enlazado", func(t *testing.T) {
		ws := workspacetest.Embedded(t, "openspec")
		sub := filepath.Join(ws.Master, "vendor", "lib")
		gitDir := filepath.Join(ws.Master, ".git", "modules", "lib")
		workspacetest.MkdirAll(t, gitDir)
		workspacetest.WriteFile(t, filepath.Join(sub, ".git"), "gitdir: "+filepath.ToSlash(gitDir)+"\n")
		loc := mustLocate(t, sub)
		if loc.Origin != OriginConfig || loc.WorkspaceRoot != ws.Master || len(loc.Notes) != 0 {
			t.Fatalf("Location = %+v, un submódulo asciende como siempre hasta el marcador del proyecto", loc)
		}
	})
	t.Run("LocateAt no asciende desde un worktree enlazado", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		worktree := filepath.Join(evalTempDir(t), "x", "repo-backend")
		workspacetest.MkdirAll(t, worktree)
		linkWorktreeOfCheckout(t, ws.Backend, worktree, "x")
		workspacetest.MkdirAll(t, filepath.Join(ws.Backend, ".git"))
		loc, err := LocateAt(worktree)
		if err != nil {
			t.Fatalf("LocateAt(%q) = %v", worktree, err)
		}
		if loc.Origin != OriginStart || loc.WorkspaceRoot != worktree || loc.Master != worktree || len(loc.Notes) != 0 {
			t.Fatalf("Location = %+v, LocateAt solo evalúa el directorio dado", loc)
		}
	})
}
