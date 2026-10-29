package sddstatus

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const planningHomeTasks = "- [ ] 1.1 Trabajo de ejemplo\n"

// statusJSONKeys devuelve las claves de primer nivel del status v2 proyectado.
func statusJSONKeys(t *testing.T, status Status) []string {
	t.Helper()
	projected, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatalf("ProjectStatusV2: %v", err)
	}
	raw, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func resolveFrom(t *testing.T, cwd, change string) Status {
	t.Helper()
	status, err := Resolve(ResolveOptions{CWD: cwd, ChangeName: change})
	if err != nil {
		t.Fatalf("Resolve(%q) = %v", cwd, err)
	}
	return status
}

// REQ-24.4: el layout embebido resuelve igual con cualquier specs_repository y
// también sin axiom.yaml, sin claves nuevas, sin notas y sin bloquear archive.
func TestResolveEmbeddedLayoutIsEquivalentForEverySpecsRepository(t *testing.T) {
	type layout struct {
		name string
		root string
	}
	var layouts []layout
	for _, specsRepository := range []string{"", ".", "openspec", "specs"} {
		ws := workspacetest.Embedded(t, specsRepository)
		seedReadyChange(t, ws.Master, "demo", planningHomeTasks)
		layouts = append(layouts, layout{"specs_repository=" + specsRepository, ws.Master})
	}
	bare := t.TempDir()
	seedReadyChange(t, bare, "demo", planningHomeTasks)
	layouts = append(layouts, layout{"sin axiom.yaml", bare})

	reference := resolveFrom(t, layouts[len(layouts)-1].root, "demo")
	referenceKeys := statusJSONKeys(t, reference)
	for _, key := range referenceKeys {
		if key == "baseline" || key == "technicalContext" {
			t.Fatalf("el status no debe proyectar %q sin declaración: %v", key, referenceKeys)
		}
	}
	for _, tc := range layouts {
		t.Run(tc.name, func(t *testing.T) {
			status := resolveFrom(t, tc.root, "demo")
			if status.ActionContext.WorkspaceRoot != tc.root {
				t.Fatalf("workspaceRoot = %q, se esperaba %q", status.ActionContext.WorkspaceRoot, tc.root)
			}
			if want := filepath.Join(tc.root, "openspec"); status.PlanningHome.Path != want || status.PlanningHome.Mode != ActionModeRepoLocal {
				t.Fatalf("planningHome = %+v, se esperaba %q en modo repo-local", status.PlanningHome, want)
			}
			if len(status.Notes) != 0 {
				t.Fatalf("notes = %v, el layout embebido no emite avisos", status.Notes)
			}
			if !reflect.DeepEqual(status.Dependencies, reference.Dependencies) {
				t.Fatalf("dependencies = %+v, se esperaba lo mismo que sin axiom.yaml: %+v", status.Dependencies, reference.Dependencies)
			}
			if keys := statusJSONKeys(t, status); !reflect.DeepEqual(keys, referenceKeys) {
				t.Fatalf("claves = %v, se esperaban las de siempre: %v", keys, referenceKeys)
			}
		})
	}
}

// REQ-24.8: el mismo planning home desde la carpeta maestra, el repositorio de
// especificaciones y un repositorio de código del espacio canónico.
func TestResolveCanonicalWorkspaceFromThreeStartingPoints(t *testing.T) {
	ws := workspacetest.Canonical(t)
	seedReadyChange(t, ws.Specs, "inc-demo", planningHomeTasks)

	starts := map[string]string{
		"carpeta maestra":                 ws.Master,
		"repositorio de especificaciones": ws.Specs,
		"repositorio de código":           filepath.Join(ws.Backend, "src"),
	}
	for name, start := range starts {
		t.Run(name, func(t *testing.T) {
			status := resolveFrom(t, start, "inc-demo")
			if status.PlanningHome.Path != ws.Home || status.PlanningHome.Mode != ActionModeRepoLocal {
				t.Fatalf("planningHome = %+v, se esperaba %q", status.PlanningHome, ws.Home)
			}
			if status.ActionContext.WorkspaceRoot != ws.Specs {
				t.Fatalf("workspaceRoot = %q, se esperaba %q", status.ActionContext.WorkspaceRoot, ws.Specs)
			}
			if want := []string{ws.Specs}; !reflect.DeepEqual(status.ActionContext.AllowedEditRoots, want) {
				t.Fatalf("allowedEditRoots = %v, se esperaba %v (los repositorios de código no se autorizan implícitamente)", status.ActionContext.AllowedEditRoots, want)
			}
			wantChangeRoot := filepath.Join(ws.Home, "changes", "inc-demo")
			if status.ChangeRoot == nil || *status.ChangeRoot != wantChangeRoot {
				t.Fatalf("changeRoot = %v, se esperaba %q", status.ChangeRoot, wantChangeRoot)
			}
		})
	}

	t.Run("raíz explícita", func(t *testing.T) {
		status, err := Resolve(ResolveOptions{WorkspaceRoot: ws.Specs, ChangeName: "inc-demo"})
		if err != nil {
			t.Fatal(err)
		}
		if status.PlanningHome.Path != ws.Home || len(status.Notes) != 0 {
			t.Fatalf("planningHome = %+v notes = %v", status.PlanningHome, status.Notes)
		}
	})
}

func TestResolveRejectsAnAmbiguousPlanningHomeFromEveryStartingPoint(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithMasterOpenSpec())
	for name, start := range map[string]string{
		"carpeta maestra":       ws.Master,
		"repositorio de código": filepath.Join(ws.Backend, "src"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Resolve(ResolveOptions{CWD: start})
			var locateErr *workspace.LocateError
			if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocateAmbiguousPlanningHome {
				t.Fatalf("Resolve = %v, se esperaba ambiguous_planning_home", err)
			}
		})
	}
}

func TestResolveRejectsAnInvalidStartingDirectory(t *testing.T) {
	_, err := Resolve(ResolveOptions{CWD: filepath.Join(t.TempDir(), "no-existe")})
	var locateErr *workspace.LocateError
	if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocateStartInvalid {
		t.Fatalf("Resolve = %v, se esperaba start_invalid", err)
	}
}

// REQ-24.7, caso 3: un axiom.yaml directo que no se puede cargar conserva la
// resolución embebida y el status lo avisa sin bloquear.
func TestResolveReportsLocationNotesWithoutBlocking(t *testing.T) {
	t.Run("axiom.yaml directo no cargable", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded}\n")
		seedReadyChange(t, dir, "demo", planningHomeTasks)

		status := resolveFrom(t, dir, "demo")
		if status.ActionContext.WorkspaceRoot != dir || status.PlanningHome.Path != filepath.Join(dir, "openspec") {
			t.Fatalf("status = %+v, se conserva la resolución embebida", status.PlanningHome)
		}
		if len(status.Notes) != 1 || !strings.Contains(status.Notes[0], "axiom workspace validate") {
			t.Fatalf("notes = %v, se esperaba un aviso que nombre axiom workspace validate", status.Notes)
		}
		if status.ApplyState != ApplyReady {
			t.Fatalf("applyState = %q, el aviso no debe bloquear", status.ApplyState)
		}
	})

	t.Run("planning home divergente sin puntero", func(t *testing.T) {
		master := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), "workspace: {name: x, topology: multirepo, specs_repository: especificacion}\nroles: {a: {name: a, repositories: [{path: .}]}}\n")
		specs := filepath.Join(master, "especificacion")
		seedReadyChange(t, specs, "demo", planningHomeTasks)
		workspacetest.MkdirAll(t, filepath.Join(master, "openspec", "changes", "antiguo"))

		status := resolveFrom(t, master, "demo")
		if status.PlanningHome.Path != filepath.Join(specs, "openspec") {
			t.Fatalf("planningHome = %q", status.PlanningHome.Path)
		}
		if len(status.Notes) != 1 || !strings.Contains(status.Notes[0], filepath.Join(master, "openspec")) {
			t.Fatalf("notes = %v, se esperaba el aviso del planning home divergente", status.Notes)
		}
	})

	t.Run("las instrucciones de fase incluyen el aviso", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded}\n")
		seedReadyChange(t, dir, "demo", planningHomeTasks)
		status, err := Resolve(ResolveOptions{CWD: dir, ChangeName: "demo", IncludeInstructions: true})
		if err != nil {
			t.Fatal(err)
		}
		if status.PhaseInstructions == nil || !strings.Contains(RenderMarkdown(status), "axiom workspace validate") {
			t.Fatalf("el render con instrucciones debe incluir el aviso: %v", status.Notes)
		}
	})
}
