package hub

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	initEmbeddedConfig = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	initMultirepoOpen  = "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	initValidBaseline  = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: flujos-de-negocio\n      path: 04_Flujos_Negocio.md\n"
	initInvalidBase    = "baseline:\n  documents: []\n"
	initValidContext   = "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n"
	initInvalidContext = "technical_context:\n  maintained:\n    paths: []\n"
	// Base sin roles: error ajeno a los bloques nuevos.
	initRolelessConfig = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: openspec\n"
)

// initSandbox es un inicializador con un registro del Hub en un directorio
// temporal, de modo que las pruebas pueden comprobar que el registro no cambia.
type initSandbox struct {
	registryPath string
	manager      *Manager
	initializer  *Initializer
}

func newInitSandbox(t *testing.T) initSandbox {
	t.Helper()
	registryPath := filepath.Join(t.TempDir(), "hub", "workspaces.json")
	manager, err := NewManager(registryPath)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return initSandbox{registryPath: registryPath, manager: manager, initializer: NewInitializer(manager, NewDetector())}
}

// registered devuelve cuántos proyectos hay en el registro del sandbox.
func (s initSandbox) registered(t *testing.T) int {
	t.Helper()
	cfg, err := s.manager.Load()
	if err != nil {
		t.Fatalf("cargar el registro del Hub: %v", err)
	}
	return len(cfg.Workspaces)
}

// requireUntouchedRegistry comprueba que el registro ni siquiera se creó.
func (s initSandbox) requireUntouchedRegistry(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(s.registryPath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("el registro del Hub no debe tocarse (stat = %v)", err)
	}
}

// snapshotTree devuelve el contenido de todo lo que hay bajo root: directorios
// y ficheros con sus bytes. Sirve para comprobar que un rechazo no escribe nada.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			tree[path] = "<dir>"
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		tree[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer %s: %v", root, err)
	}
	return tree
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	return string(data)
}

func notesContain(notes []string, fragment string) bool {
	for _, note := range notes {
		if strings.Contains(note, fragment) {
			return true
		}
	}
	return false
}

// requireNote falla si ninguna nota contiene fragment.
func requireNote(t *testing.T, notes []string, fragment string) {
	t.Helper()
	if !notesContain(notes, fragment) {
		t.Errorf("ninguna nota contiene %q; notas: %q", fragment, notes)
	}
}

// multi-project-hub REQ-2.2: un puntero propio hacia otro planning home rechaza
// la inicialización, con cualquier perfil, sin escribir ni registrar.
func TestInitRejectsAnOwnPointerToAnotherPlanningHome(t *testing.T) {
	for _, profile := range []string{"", "full", "knowledge"} {
		t.Run("perfil "+profile, func(t *testing.T) {
			ws := workspacetest.Canonical(t)
			sandbox := newInitSandbox(t)
			before := snapshotTree(t, ws.Master)

			res, err := sandbox.initializer.Init(InitOptions{Path: ws.Master, Profile: profile})

			var foreign *ForeignPlanningHomeError
			if !errors.As(err, &foreign) {
				t.Fatalf("Init = (%+v, %v), se esperaba un *ForeignPlanningHomeError", res, err)
			}
			if res != nil {
				t.Errorf("un rechazo no devuelve resultado, obtuvo %+v", res)
			}
			if foreign.PointerPath != filepath.Join(ws.Master, ".axiom-workspace") || foreign.PlanningHome != ws.Home || foreign.WorkspaceRoot != ws.Specs {
				t.Errorf("campos = %+v, se esperaban el puntero, %s y %s", foreign, ws.Home, ws.Specs)
			}
			for _, want := range []string{".axiom-workspace", ws.Home, "axiom init --path", ws.Specs, "receta"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el mensaje no contiene %q:\n%v", want, err)
				}
			}
			if after := snapshotTree(t, ws.Master); !reflect.DeepEqual(before, after) {
				t.Errorf("el rechazo no debe escribir nada: antes %d entradas, después %d", len(before), len(after))
			}
			sandbox.requireUntouchedRegistry(t)
		})
	}
}

// REQ-24.6 y REQ-24.7: un puntero propio que no resuelve rechaza con el mismo
// error del primitivo de resolución.
func TestInitRejectsAnOwnPointerThatCannotBeResolved(t *testing.T) {
	tests := []struct {
		name    string
		variant workspacetest.Variant
		kind    workspace.LocateErrorKind
	}{
		{"puntero roto", workspacetest.WithBrokenPointer(), workspace.LocatePointerBroken},
		{"axiom.yaml apuntado ilegible", workspacetest.WithUnreadableConfig(), workspace.LocateConfigUnloadable},
		{"repositorio de especificaciones inexistente", workspacetest.WithoutSpecsRepository(), workspace.LocateSpecsRepositoryMissing},
		{"openspec propio que no es su planning home", workspacetest.WithMasterOpenSpec(), workspace.LocateAmbiguousPlanningHome},
	}
	for _, tt := range tests {
		for _, profile := range []string{"full", "knowledge"} {
			t.Run(tt.name+" perfil "+profile, func(t *testing.T) {
				ws := workspacetest.Canonical(t, tt.variant)
				sandbox := newInitSandbox(t)
				before := snapshotTree(t, ws.Master)

				_, err := sandbox.initializer.Init(InitOptions{Path: ws.Master, Profile: profile})

				var locateErr *workspace.LocateError
				if !errors.As(err, &locateErr) || locateErr.Kind != tt.kind {
					t.Fatalf("Init = %v, se esperaba un *workspace.LocateError %s", err, tt.kind)
				}
				if locateErr.Continuation == "" {
					t.Errorf("el rechazo debe nombrar la continuación: %+v", locateErr)
				}
				var foreign *ForeignPlanningHomeError
				if errors.As(err, &foreign) {
					t.Errorf("el error debe ser el del primitivo, no el rechazo de planning home ajeno")
				}
				if after := snapshotTree(t, ws.Master); !reflect.DeepEqual(before, after) {
					t.Errorf("el rechazo no debe escribir nada")
				}
				sandbox.requireUntouchedRegistry(t)
			})
		}
	}
}

// REQ-2.2: un puntero que resuelve a <directorio>/openspec no se rechaza, aunque
// ese directorio no exista todavía. No se reescribe el axiom.yaml del puntero.
func TestInitAcceptsAnOwnPointerToItsOwnPlanningHome(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"monorepo-embedded", initEmbeddedConfig},
		{"multirepo con specs_repository openspec", initMultirepoOpen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			master := t.TempDir()
			workspacetest.WriteFile(t, filepath.Join(master, ".axiom-workspace"), "config: axiom.yaml\n")
			workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"), tt.config)
			sandbox := newInitSandbox(t)

			res, err := sandbox.initializer.Init(InitOptions{Path: master})
			if err != nil {
				t.Fatalf("Init = %v, no debía rechazar: el planning home del puntero es <directorio>/openspec", err)
			}
			if !res.AlreadyExisted || res.ConfigPath != filepath.Join(master, "axiom.yaml") {
				t.Errorf("resultado = %+v, se esperaba el axiom.yaml del puntero como existente", res)
			}
			if got := readFile(t, filepath.Join(master, "axiom.yaml")); got != tt.config {
				t.Errorf("el axiom.yaml existente no se sobrescribe:\n%s", got)
			}
			for _, dir := range []string{"openspec/specs", "openspec/changes", ".axiom/inbox/skills"} {
				if !exists(filepath.Join(master, filepath.FromSlash(dir))) {
					t.Errorf("falta %s", dir)
				}
			}
			if sandbox.registered(t) != 1 {
				t.Errorf("el proyecto debe quedar registrado")
			}
			if len(res.Notes) != 0 {
				t.Errorf("sin declaraciones ni violaciones no hay notas: %q", res.Notes)
			}
		})
	}
}

// El axiom.yaml que init considera existente es el que indica el puntero propio:
// no escribe <ruta>/axiom.yaml cuando el puntero señala otro fichero.
func TestInitUsesThePointedConfigAsTheExistingOne(t *testing.T) {
	master := t.TempDir()
	workspacetest.WriteFile(t, filepath.Join(master, ".axiom-workspace"), "config: config/axiom.yaml\n")
	configPath := filepath.Join(master, "config", "axiom.yaml")
	workspacetest.WriteFile(t, configPath, initMultirepoOpen+initValidBaseline)
	sandbox := newInitSandbox(t)

	res, err := sandbox.initializer.Init(InitOptions{Path: master})
	if err != nil {
		t.Fatalf("Init = %v", err)
	}
	if res.ConfigPath != configPath || !res.AlreadyExisted {
		t.Errorf("ConfigPath/AlreadyExisted = %q / %v, se esperaba %q existente", res.ConfigPath, res.AlreadyExisted, configPath)
	}
	if exists(filepath.Join(master, "axiom.yaml")) {
		t.Error("no debe escribirse <ruta>/axiom.yaml cuando el puntero indica otro fichero")
	}
	if res.BaselineOwnership != workspace.BaselineProject || exists(filepath.Join(master, "openspec", "specs")) {
		t.Errorf("la consulta de baseline es la del fichero apuntado: %q", res.BaselineOwnership)
	}
}

// REQ-2.2: un puntero solo en un directorio ancestro no impide inicializar; el
// marcador más cercano gana y el nuevo axiom.yaml pasa a serlo.
func TestInitIgnoresAPointerInAnAncestor(t *testing.T) {
	ws := workspacetest.Canonical(t)
	sandbox := newInitSandbox(t)

	res, err := sandbox.initializer.Init(InitOptions{Path: ws.Backend, Name: "backend"})
	if err != nil {
		t.Fatalf("Init = %v, no debía rechazar por el puntero de la carpeta maestra", err)
	}
	if res.AlreadyExisted || res.ConfigPath != filepath.Join(ws.Backend, "axiom.yaml") {
		t.Errorf("resultado = %+v, se esperaba un axiom.yaml nuevo en el propio destino", res)
	}
	for _, dir := range []string{"openspec/specs", "openspec/changes", ".axiom/inbox/skills"} {
		if !exists(filepath.Join(ws.Backend, filepath.FromSlash(dir))) {
			t.Errorf("falta %s", dir)
		}
	}
	if sandbox.registered(t) != 1 {
		t.Errorf("el proyecto debe quedar registrado")
	}
	if res.BaselineOwnership != workspace.BaselineDefault || len(res.Notes) != 0 {
		t.Errorf("sin declaraciones: ownership %q, notas %q", res.BaselineOwnership, res.Notes)
	}
}

// Un puntero roto solo en un ancestro tampoco impide inicializar.
func TestInitIgnoresABrokenPointerInAnAncestor(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	sandbox := newInitSandbox(t)
	if _, err := sandbox.initializer.Init(InitOptions{Path: ws.Backend, Name: "backend"}); err != nil {
		t.Fatalf("Init = %v, el puntero roto de un ancestro no debe detenerlo", err)
	}
	if sandbox.registered(t) != 1 {
		t.Errorf("el proyecto debe quedar registrado")
	}
}

// REQ-24.23: con baseline del proyecto, init no crea openspec/specs/ ni
// openspec/INDEX.md, conserva el resto del andamiaje de su perfil, informa y no
// falla. El axiom.yaml existente no se sobrescribe.
func TestInitWithAProjectBaselineSkipsTheSpecs(t *testing.T) {
	tests := []struct {
		name        string
		profile     string
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name:        "perfil completo",
			profile:     "full",
			wantPresent: []string{"openspec/changes", ".axiom/inbox/skills"},
			wantAbsent:  []string{"openspec/specs", "openspec/INDEX.md", ".mcp.json"},
		},
		{
			name:        "perfil knowledge",
			profile:     "knowledge",
			wantPresent: []string{"openspec/changes", ".mcp.json"},
			wantAbsent:  []string{"openspec/specs", "openspec/INDEX.md", ".axiom/inbox/skills"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			config := initEmbeddedConfig + initValidBaseline
			workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), config)
			sandbox := newInitSandbox(t)

			res, err := sandbox.initializer.Init(InitOptions{Path: dir, Profile: tt.profile})
			if err != nil {
				t.Fatalf("Init = %v, con baseline del proyecto completa con código 0", err)
			}
			for _, rel := range tt.wantPresent {
				if !exists(filepath.Join(dir, filepath.FromSlash(rel))) {
					t.Errorf("debe existir %s", rel)
				}
			}
			for _, rel := range tt.wantAbsent {
				if exists(filepath.Join(dir, filepath.FromSlash(rel))) {
					t.Errorf("no debe existir %s", rel)
				}
			}
			if got := readFile(t, filepath.Join(dir, "axiom.yaml")); got != config {
				t.Errorf("el axiom.yaml existente no se sobrescribe:\n%s", got)
			}
			if res.BaselineOwnership != workspace.BaselineProject || !res.AlreadyExisted {
				t.Errorf("ownership/AlreadyExisted = %q / %v", res.BaselineOwnership, res.AlreadyExisted)
			}
			requireNote(t, res.Notes, "baseline de especificaciones es del proyecto")
			requireNote(t, res.Notes, "modelo-de-datos, flujos-de-negocio")
			requireNote(t, res.Notes, "no se crean openspec/specs/ ni openspec/INDEX.md")
			if sandbox.registered(t) != 1 {
				t.Errorf("el proyecto debe quedar registrado")
			}
		})
	}
}

// REQ-24.18 y REQ-24.23: con una baseline indeterminada, init omite la baseline,
// informa de la causa y completa con código 0.
func TestInitWithAnIndeterminateBaselineSkipsTheSpecs(t *testing.T) {
	tests := []struct {
		name   string
		config string
		cause  string
	}{
		{"baseline invalida", initEmbeddedConfig + initInvalidBase, "al menos un documento"},
		{"yaml ilegible", "workspace: [sin cerrar\n", "error al parsear axiom.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), tt.config)
			sandbox := newInitSandbox(t)

			res, err := sandbox.initializer.Init(InitOptions{Path: dir})
			if err != nil {
				t.Fatalf("Init = %v, con baseline indeterminada completa con código 0", err)
			}
			if exists(filepath.Join(dir, "openspec", "specs")) || exists(filepath.Join(dir, "openspec", "INDEX.md")) {
				t.Error("no debe crearse openspec/specs/ ni INDEX.md con una baseline indeterminada")
			}
			if !exists(filepath.Join(dir, "openspec", "changes")) || !exists(filepath.Join(dir, ".axiom", "inbox", "skills")) {
				t.Error("el resto del andamiaje se conserva")
			}
			if res.BaselineOwnership != workspace.BaselineIndeterminate {
				t.Errorf("ownership = %q", res.BaselineOwnership)
			}
			requireNote(t, res.Notes, "No se puede determinar")
			requireNote(t, res.Notes, tt.cause)
			requireNote(t, res.Notes, "axiom workspace validate --path")
			if got := readFile(t, filepath.Join(dir, "axiom.yaml")); got != tt.config {
				t.Errorf("el axiom.yaml existente no se sobrescribe:\n%s", got)
			}
		})
	}
}

// REQ-2.2, punto 4: init valida el axiom.yaml que encuentra, informa de cada
// violación con la continuación y no cambia el código de salida. Solo la baseline
// decide si se escribe openspec/specs/.
func TestInitReportsEveryViolationOfAnExistingConfig(t *testing.T) {
	t.Run("technical_context invalido no impide la baseline por defecto", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), initEmbeddedConfig+initInvalidContext)
		res, err := newInitSandbox(t).initializer.Init(InitOptions{Path: dir})
		if err != nil {
			t.Fatalf("Init = %v", err)
		}
		if !exists(filepath.Join(dir, "openspec", "specs")) {
			t.Error("un technical_context inválido solo se informa: openspec/specs/ se crea")
		}
		requireNote(t, res.Notes, "no es válido")
		requireNote(t, res.Notes, "technical_context")
		requireNote(t, res.Notes, "axiom workspace validate --path")
		if res.BaselineOwnership != workspace.BaselineDefault {
			t.Errorf("ownership = %q", res.BaselineOwnership)
		}
	})
	t.Run("error de la configuracion base solo se informa", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), initRolelessConfig)
		res, err := newInitSandbox(t).initializer.Init(InitOptions{Path: dir})
		if err != nil {
			t.Fatalf("Init = %v", err)
		}
		if !exists(filepath.Join(dir, "openspec", "specs")) {
			t.Error("un error ajeno al bloque baseline no cambia la baseline por defecto")
		}
		requireNote(t, res.Notes, "al menos un rol")
		requireNote(t, res.Notes, "axiom workspace validate --path")
	})
	t.Run("una violacion por nota", func(t *testing.T) {
		dir := t.TempDir()
		workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), initEmbeddedConfig+initInvalidBase+initInvalidContext)
		res, err := newInitSandbox(t).initializer.Init(InitOptions{Path: dir})
		if err != nil {
			t.Fatalf("Init = %v", err)
		}
		violations := 0
		for _, note := range res.Notes {
			if strings.HasPrefix(note, "El axiom.yaml") {
				violations++
			}
		}
		if violations < 2 {
			t.Errorf("se esperaba una nota por violación (baseline y technical_context), notas: %q", res.Notes)
		}
	})
	t.Run("sin axiom.yaml no hay violaciones", func(t *testing.T) {
		res, err := newInitSandbox(t).initializer.Init(InitOptions{Path: t.TempDir()})
		if err != nil {
			t.Fatalf("Init = %v", err)
		}
		if len(res.Notes) != 0 {
			t.Errorf("un proyecto nuevo no genera notas: %q", res.Notes)
		}
	})
}

// m13: con --force, init lee y valida el axiom.yaml existente antes de
// sobrescribirlo. Si declara baseline: o technical_context: (válidos o no), lo
// conserva íntegro, lo explica y completa con código 0; sin esos bloques lo
// sobrescribe como antes.
func TestInitForceKeepsAConfigWithDeclarations(t *testing.T) {
	tests := []struct {
		name          string
		config        string
		wantSpecs     bool // si openspec/specs/ debe crearse (baseline por defecto)
		wantOwnership workspace.BaselineOwnership
	}{
		{"baseline valida", initEmbeddedConfig + initValidBaseline, false, workspace.BaselineProject},
		{"baseline invalida", initEmbeddedConfig + initInvalidBase, false, workspace.BaselineIndeterminate},
		{"technical_context valido", initEmbeddedConfig + initValidContext, true, workspace.BaselineDefault},
		{"technical_context invalido", initEmbeddedConfig + initInvalidContext, true, workspace.BaselineDefault},
		{"yaml ilegible", "workspace: [sin cerrar\n", false, workspace.BaselineIndeterminate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "axiom.yaml")
			workspacetest.WriteFile(t, configPath, tt.config)
			sandbox := newInitSandbox(t)

			res, err := sandbox.initializer.Init(InitOptions{Path: dir, Name: "Nuevo", Force: true})
			if err != nil {
				t.Fatalf("Init --force = %v, debe completar con código 0", err)
			}
			if got := readFile(t, configPath); got != tt.config {
				t.Errorf("--force no sobrescribe un axiom.yaml con declaraciones:\n%s", got)
			}
			if !res.AlreadyExisted || res.ConfigPath != configPath {
				t.Errorf("AlreadyExisted/ConfigPath = %v / %q", res.AlreadyExisted, res.ConfigPath)
			}
			requireNote(t, res.Notes, "Se conserva")
			requireNote(t, res.Notes, "--force")
			if exists(filepath.Join(dir, "openspec", "specs")) != tt.wantSpecs {
				t.Errorf("openspec/specs existe = %v, se esperaba %v", !tt.wantSpecs, tt.wantSpecs)
			}
			if res.BaselineOwnership != tt.wantOwnership {
				t.Errorf("ownership = %q, se esperaba %q", res.BaselineOwnership, tt.wantOwnership)
			}
			if sandbox.registered(t) != 1 {
				t.Errorf("el proyecto debe quedar registrado")
			}
		})
	}
}

func TestInitForceOverwritesAConfigWithoutDeclarations(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "axiom.yaml")
	workspacetest.WriteFile(t, configPath, initRolelessConfig)

	res, err := newInitSandbox(t).initializer.Init(InitOptions{Path: dir, Name: "Nuevo Nombre", Force: true})
	if err != nil {
		t.Fatalf("Init --force = %v", err)
	}
	if res.AlreadyExisted {
		t.Error("con --force y sin declaraciones el fichero se sobrescribe, como antes")
	}
	got := readFile(t, configPath)
	if got == initRolelessConfig || !strings.Contains(got, `name: "Nuevo Nombre"`) {
		t.Errorf("el axiom.yaml debe regenerarse:\n%s", got)
	}
	if notesContain(res.Notes, "Se conserva") {
		t.Errorf("no hay nada que conservar: %q", res.Notes)
	}
	if !exists(filepath.Join(dir, "openspec", "specs")) {
		t.Error("sin baseline del proyecto se crea openspec/specs/")
	}
}

// Sin --force, un axiom.yaml existente nunca se sobrescribe y el resto del
// comportamiento previo se conserva.
func TestInitWithoutForceKeepsTheExistingConfig(t *testing.T) {
	dir := t.TempDir()
	workspacetest.WriteFile(t, filepath.Join(dir, "axiom.yaml"), initEmbeddedConfig)
	res, err := newInitSandbox(t).initializer.Init(InitOptions{Path: dir})
	if err != nil {
		t.Fatalf("Init = %v", err)
	}
	if !res.AlreadyExisted || readFile(t, filepath.Join(dir, "axiom.yaml")) != initEmbeddedConfig {
		t.Errorf("sin --force el axiom.yaml existente se conserva: %+v", res)
	}
	if notesContain(res.Notes, "Se conserva") {
		t.Errorf("sin --force no hay nota de conservación: %q", res.Notes)
	}
}
