package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// canonicalService monta el espacio canónico (carpeta maestra con puntero,
// repositorio de especificaciones y repositorios de código) y un servicio cuya
// ruta registrada es start. Las pruebas de este fichero no usan Hub: solo
// ejercitan la resolución del planning home.
func canonicalService(t *testing.T, start func(workspacetest.Workspace) string, variants ...workspacetest.Variant) (*Service, workspacetest.Workspace) {
	t.Helper()
	ws := workspacetest.Canonical(t, variants...)
	return NewServiceWithHub(start(ws), nil), ws
}

func startAtMaster(ws workspacetest.Workspace) string  { return ws.Master }
func startAtBackend(ws workspacetest.Workspace) string { return ws.Backend }

// requireSameDir falla si got y want no son el mismo directorio (insensible a
// los enlaces y a los nombres cortos de Windows).
func requireSameDir(t *testing.T, label, got, want string) {
	t.Helper()
	if got == "" {
		t.Fatalf("%s: ruta vacía, se esperaba %q", label, want)
	}
	if !sameDirectory(t, got, want) {
		t.Errorf("%s = %q, se esperaba %q", label, got, want)
	}
}

// requireNoOpenSpecAt falla si alguna de las rutas contiene un directorio
// openspec/ o un repositorio de especificaciones con cambios: las operaciones del
// dashboard no deben crear nada fuera del planning home.
func requireNoOpenSpecAt(t *testing.T, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, "openspec")); !os.IsNotExist(err) {
			t.Errorf("no debe existir openspec/ en %s (err=%v)", dir, err)
		}
	}
}

// TestPlanningHomeResolvesFromTheRegisteredPath cubre REQ-24.10: con la ruta
// registrada en la carpeta maestra o en un repositorio de código, los cambios se
// leen de <planning home>/changes, y rootPath sigue siendo la ruta registrada.
func TestPlanningHomeResolvesFromTheRegisteredPath(t *testing.T) {
	for _, tt := range []struct {
		name  string
		start func(workspacetest.Workspace) string
	}{
		{"carpeta maestra", startAtMaster},
		{"repositorio de código", startAtBackend},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, ws := canonicalService(t, tt.start)
			writeFixtureFile(t, ws.Home, "changes/archive/2026-01-01-inc-98-viejo/proposal.md", "# Viejo\n")
			registered := tt.start(ws)

			list, err := svc.GetIncrements()
			if err != nil {
				t.Fatalf("GetIncrements: %v", err)
			}
			kinds := map[string]string{}
			for _, inc := range list {
				kinds[inc.Name] = inc.Type
			}
			if kinds["inc-99-ejemplo"] != "active" || kinds["2026-01-01-inc-98-viejo"] != "archived" {
				t.Errorf("GetIncrements = %v, se esperaba el activo y el archivado del planning home", kinds)
			}

			path, kind, err := svc.FindIncrementPath("inc-99-ejemplo")
			if err != nil || kind != "active" {
				t.Fatalf("FindIncrementPath activo: %q %q %v", path, kind, err)
			}
			requireSameDir(t, "ruta del activo", path, filepath.Join(ws.Home, "changes", "inc-99-ejemplo"))

			path, kind, err = svc.FindIncrementPath("inc-98-viejo")
			if err != nil || kind != "archived" {
				t.Fatalf("FindIncrementPath archivado por sufijo: %q %q %v", path, kind, err)
			}
			requireSameDir(t, "ruta del archivado", path, filepath.Join(ws.Home, "changes", "archive", "2026-01-01-inc-98-viejo"))

			if got := svc.getRootPath(); got != registered {
				t.Errorf("rootPath = %q, debe seguir siendo la ruta registrada %q", got, registered)
			}
		})
	}
}

func TestCreateIncrementWritesUnderThePlanningHome(t *testing.T) {
	for _, tt := range []struct {
		name  string
		start func(workspacetest.Workspace) string
	}{
		{"carpeta maestra", startAtMaster},
		{"repositorio de código", startAtBackend},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, ws := canonicalService(t, tt.start)
			// REQ-25.32: `ensure` precede a la escritura. En un multirepo el repositorio de specs se
			// trabaja in situ, en su rama de integración (REQ-25.5), de modo que solo necesita git.
			commitGitRepo(t, ws.Specs)

			res, err := svc.CreateIncrement(CreateIncrementRequest{Name: "mi-cambio", Intent: "Propósito X"})
			if err != nil {
				t.Fatalf("CreateIncrement: %v", err)
			}
			want := filepath.Join(ws.Home, "changes", "mi-cambio")
			requireSameDir(t, "directorio del cambio", filepath.Dir(res.Path), want)
			if _, err := os.Stat(filepath.Join(want, "proposal.md")); err != nil {
				t.Errorf("proposal.md debe estar en el planning home: %v", err)
			}
			requireNoOpenSpecAt(t, ws.Master, ws.Backend, ws.Frontend)

			if _, err := svc.CreateIncrement(CreateIncrementRequest{Name: "mi-cambio"}); err == nil {
				t.Error("un segundo CreateIncrement con el mismo nombre debe detectar el cambio existente")
			}
		})
	}
}

func TestMigrateIncompleteTasksUsesTheCumulativeOfThePlanningHome(t *testing.T) {
	svc, ws := canonicalService(t, startAtMaster)
	writeFixtureFile(t, ws.Home, "changes/inc-99-ejemplo/tasks.md",
		"# Tareas\n- [x] 1.1 hecha\n- [ ] 1.2 pendiente\n")

	moved, err := svc.MigrateIncompleteTasksToCumulative("inc-99-ejemplo", "backend")
	if err != nil {
		t.Fatalf("MigrateIncompleteTasksToCumulative: %v", err)
	}
	if moved != 1 {
		t.Errorf("tareas migradas = %d, se esperaba 1", moved)
	}

	cumulative := filepath.Join(ws.Home, "changes", "cumulative-backend", "tasks.md")
	data, err := os.ReadFile(cumulative)
	if err != nil {
		t.Fatalf("el acumulativo debe estar en <planning home>/changes/cumulative-<rol>: %v", err)
	}
	if !strings.Contains(string(data), "1.2 pendiente") {
		t.Errorf("el acumulativo no recoge la tarea pendiente:\n%s", data)
	}
	// La ubicación anterior colgaba de <raíz registrada>/<specs_repository>.
	if _, err := os.Stat(filepath.Join(ws.Master, "repo-specs", "changes")); !os.IsNotExist(err) {
		t.Errorf("no debe crearse changes/ junto al repositorio de especificaciones (err=%v)", err)
	}
}

// TestCreateHandoffTextUsesThePlanningHome cubre el texto de CreateHandoff: la
// ruta de artefactos por defecto es la del planning home, relativa a la ruta
// registrada cuando cuelga de ella y absoluta cuando queda fuera.
func TestCreateHandoffTextUsesThePlanningHome(t *testing.T) {
	req := CreateHandoffRequest{Change: "inc-99-ejemplo", FromPhase: "design", ToPhase: "tasks", ExecutiveSummary: "Resumen de prueba"}

	t.Run("relativa a la carpeta maestra", func(t *testing.T) {
		svc, ws := canonicalService(t, startAtMaster)
		res, err := svc.CreateHandoff(req)
		if err != nil {
			t.Fatalf("CreateHandoff: %v", err)
		}
		requireSameDir(t, "directorio del handoff", filepath.Dir(res.FilePath), filepath.Join(ws.Home, "changes", "inc-99-ejemplo"))
		data, err := os.ReadFile(res.FilePath)
		if err != nil {
			t.Fatal(err)
		}
		if want := "repo-specs/openspec/changes/inc-99-ejemplo"; !strings.Contains(string(data), want) {
			t.Errorf("handoff.md debe citar %q:\n%s", want, data)
		}
		if strings.Contains(string(data), "\n- openspec/changes/") {
			t.Errorf("handoff.md no debe citar la ruta antigua openspec/changes/...:\n%s", data)
		}
	})

	t.Run("absoluta desde un repositorio de código", func(t *testing.T) {
		svc, ws := canonicalService(t, startAtBackend)
		res, err := svc.CreateHandoff(req)
		if err != nil {
			t.Fatalf("CreateHandoff: %v", err)
		}
		data, err := os.ReadFile(res.FilePath)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.ToSlash(filepath.Join(ws.Home, "changes", "inc-99-ejemplo"))
		if !strings.Contains(string(data), want) {
			t.Errorf("handoff.md debe citar la ruta absoluta %q:\n%s", want, data)
		}
	})

	t.Run("un cambio archivado no admite handoff", func(t *testing.T) {
		svc, ws := canonicalService(t, startAtMaster)
		writeFixtureFile(t, ws.Home, "changes/archive/2026-01-01-inc-98-viejo/proposal.md", "# Viejo\n")
		_, err := svc.CreateHandoff(CreateHandoffRequest{Change: "inc-98-viejo", ExecutiveSummary: "x"})
		if err == nil || !strings.Contains(err.Error(), "no existe como cambio activo") {
			t.Errorf("err = %v, se esperaba el rechazo de cambio no activo", err)
		}
	})
}

// TestPlanningRootIsResolvedOnEveryCall cubre que el servicio no guarda el
// resultado de Locate: un espacio cuyo puntero se repara entre dos llamadas pasa
// de rechazarse a resolverse con el mismo servicio.
func TestPlanningRootIsResolvedOnEveryCall(t *testing.T) {
	svc, ws := canonicalService(t, startAtMaster, workspacetest.WithBrokenPointer())

	var locateErr *workspace.LocateError
	if _, err := svc.GetIncrements(); !errors.As(err, &locateErr) {
		t.Fatalf("con el puntero roto se esperaba un *LocateError, obtenido %v", err)
	}

	writeFixtureFile(t, ws.Master, ".axiom-workspace", "config: repo-specs/axiom.yaml\n")
	list, err := svc.GetIncrements()
	if err != nil {
		t.Fatalf("tras reparar el puntero, GetIncrements debe resolver: %v", err)
	}
	if len(list) != 1 || list[0].Name != "inc-99-ejemplo" {
		t.Errorf("GetIncrements tras reparar = %+v", list)
	}
}

// TestUnresolvablePlanningHomeIsPropagated cubre que un fallo de Locate llega
// intacto (causa y continuación) y que ningún método escribe nada.
func TestUnresolvablePlanningHomeIsPropagated(t *testing.T) {
	svc, ws := canonicalService(t, startAtMaster, workspacetest.WithBrokenPointer())

	calls := map[string]func() error{
		"GetIncrements": func() error { _, err := svc.GetIncrements(); return err },
		"FindIncrementPath": func() error {
			_, _, err := svc.FindIncrementPath("inc-99-ejemplo")
			return err
		},
		"CreateIncrement": func() error {
			_, err := svc.CreateIncrement(CreateIncrementRequest{Name: "mi-cambio"})
			return err
		},
		"MigrateIncompleteTasksToCumulative": func() error {
			_, err := svc.MigrateIncompleteTasksToCumulative("inc-99-ejemplo", "backend")
			return err
		},
		"CreateHandoff": func() error {
			_, err := svc.CreateHandoff(CreateHandoffRequest{Change: "inc-99-ejemplo", ExecutiveSummary: "x"})
			return err
		},
		"ContinueIncrement": func() error {
			_, err := svc.ContinueIncrement("inc-99-ejemplo")
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			var locateErr *workspace.LocateError
			err := call()
			if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocatePointerBroken {
				t.Fatalf("err = %v, se esperaba el *LocateError pointer_broken", err)
			}
			if !strings.Contains(err.Error(), locateErr.Continuation) || locateErr.Continuation == "" {
				t.Errorf("el error debe llevar la continuación accionable: %v", err)
			}
		})
	}
	requireNoOpenSpecAt(t, ws.Master, ws.Backend, ws.Frontend)
}

// TestIncrementsEndpointListsThePlanningHome es el arnés HTTP: GET /api/increments
// sobre el canónico devuelve los cambios del planning home, y con el puntero roto
// responde 500 con el mensaje de Locate.
func TestIncrementsEndpointListsThePlanningHome(t *testing.T) {
	svc, _ := canonicalService(t, startAtMaster)
	rr := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/increments", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/increments = %d: %s", rr.Code, rr.Body.String())
	}
	var list []IncrementSummaryDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("respuesta no válida: %v", err)
	}
	if len(list) != 1 || list[0].Name != "inc-99-ejemplo" || list[0].Type != "active" {
		t.Errorf("GET /api/increments = %+v", list)
	}

	broken, _ := canonicalService(t, startAtMaster, workspacetest.WithBrokenPointer())
	rr = httptest.NewRecorder()
	NewServer(broken).Router().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/increments", nil))
	if rr.Code != http.StatusInternalServerError || !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("con el puntero roto: %d %s", rr.Code, rr.Body.String())
	}
}
