package cli

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/identitystore"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

const (
	adoPAT    = "ficticio-pat-connect-0123456789"
	adoPATEnv = "AXIOM_INTEGRATION_AZURE_DEVOPS_PAT"

	adoConnectionDataRoute = "/contoso/_apis/connectionData"
	adoTypesRoute          = "/contoso/Fabrikam/_apis/wit/workitemtypes"
	adoIterationsRoute     = "/contoso/Fabrikam/Fabrikam Team/_apis/work/teamsettings/iterations"
	adoMembersRoute        = "/contoso/_apis/projects/Fabrikam/teams/Fabrikam Team/members"
)

// adoEnv monta un workspace con el bloque dado, un servidor Azure DevOps de prueba instalado en las
// costuras, el PAT ficticio en el entorno y un directorio de estado temporal.
type adoEnv struct {
	root     string
	stateDir string
	srv      *adotest.Server
}

func newADOEnv(t *testing.T, block string) *adoEnv {
	t.Helper()
	root, _ := newGovernanceWorkspace(t, "x")
	writeCLIFixture(t, filepath.Join(root, "axiom.yaml"), integrationsBase+block)
	quiet := writerBranchWarnings
	writerBranchWarnings = io.Discard
	t.Cleanup(func() { writerBranchWarnings = quiet })

	stateDir := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	t.Setenv(adoPATEnv, adoPAT)

	srv := adotest.NewServer(t, nil)
	adotest.Install(t, srv)
	return &adoEnv{root: root, stateDir: stateDir, srv: srv}
}

// serveProcess programa un proceso completo: dos tipos, dos iteraciones y tres miembros.
func (e *adoEnv) serveProcess(connection adotest.Response) {
	for _, route := range adoProcessRoutes(connection) {
		e.srv.Reply("GET", route.path, route.response)
	}
}

// adoRoute es una ruta GET del servidor de prueba con su respuesta.
type adoRoute struct {
	path     string
	response adotest.Response
}

// adoProcessRoutes devuelve las rutas del proceso completo: connectionData con la respuesta dada,
// dos tipos con sus estados y campos, dos iteraciones y tres miembros.
func adoProcessRoutes(connection adotest.Response) []adoRoute {
	list := func(items ...map[string]any) adotest.Response {
		return adotest.Response{JSON: map[string]any{"value": items}}
	}
	return []adoRoute{
		{adoConnectionDataRoute, connection},
		{adoTypesRoute, list(
			map[string]any{"name": "Bug", "referenceName": "Microsoft.VSTS.WorkItemTypes.Bug"},
			map[string]any{"name": "Task", "referenceName": "Microsoft.VSTS.WorkItemTypes.Task"},
		)},
		{adoTypesRoute + "/Bug/states", list(
			map[string]any{"name": "New", "category": "Proposed"},
			map[string]any{"name": "Active", "category": "InProgress"},
			map[string]any{"name": "Removed", "category": "Removed"},
		)},
		{adoTypesRoute + "/Task/states", list(map[string]any{"name": "To Do", "category": "Proposed"})},
		{adoTypesRoute + "/Bug/fields", list(map[string]any{"referenceName": "System.Title", "name": "Title", "alwaysRequired": true})},
		{adoTypesRoute + "/Task/fields", list(map[string]any{"referenceName": "System.Title", "name": "Title", "alwaysRequired": true})},
		{adoIterationsRoute, list(
			map[string]any{"id": "it-1", "name": "Sprint 1", "path": `Fabrikam\Sprint 1`,
				"attributes": map[string]any{"startDate": "2026-11-01T00:00:00Z", "finishDate": "2026-11-14T00:00:00Z", "timeFrame": "current"}},
			map[string]any{"id": "it-2", "name": "Sprint 2", "path": `Fabrikam\Sprint 2`},
		)},
		{adoMembersRoute, list(adoMember("abc-123", "Ana"), adoMember("def-456", "Berto"), adoMember("ghi-789", "Carla"))},
	}
}

func adoMember(id, name string) map[string]any {
	return map[string]any{"identity": map[string]any{"id": id, "displayName": name}}
}

func adoConnectionOK() adotest.Response {
	return adotest.Response{JSON: map[string]any{"authenticatedUser": map[string]any{
		"id": "guid-1", "customDisplayName": "Ana Pérez", "providerDisplayName": "ana",
	}}}
}

func (e *adoEnv) identityPath() string { return filepath.Join(e.stateDir, identitystore.FileName) }

func (e *adoEnv) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	return runIntegrationsCLI(t, "", append(args, "--cwd", e.root)...)
}

// snapshotFiles devuelve el contenido de cada fichero bajo dir, por ruta relativa.
func snapshotFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(dir, path)
		files[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func assertSameSnapshot(t *testing.T, label string, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Errorf("%s: cambió el conjunto de ficheros: antes %d, después %d (%v)", label, len(before), len(after), after)
		return
	}
	for name, content := range before {
		if got, ok := after[name]; !ok || got != content {
			t.Errorf("%s: el fichero %s cambió", label, name)
		}
	}
}

func readIdentityFile(t *testing.T, path string) identitystore.File {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer el fichero de identidad: %v", err)
	}
	var file identitystore.File
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	return file
}

// ---- connect

func TestIntegrationsConnectRequiresPlugin(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	code, _, errOut := e.run(t, "connect")
	if code != 2 || !strings.Contains(errOut, "--plugin") {
		t.Fatalf("sin --plugin: %d %q", code, errOut)
	}
	code, _, errOut = e.run(t, "discover")
	if code != 2 || !strings.Contains(errOut, "--plugin") {
		t.Fatalf("discover sin --plugin: %d %q", code, errOut)
	}
}

func TestIntegrationsUnknownPluginListsTheCatalogBeforeReadingTheWorkspace(t *testing.T) {
	newADOEnv(t, adoBlock)
	missing := filepath.Join(t.TempDir(), "no-existe")
	for _, verb := range []string{"connect", "discover"} {
		code, _, errOut := runIntegrationsCLI(t, "", verb, "--plugin", "jira", "--cwd", missing)
		if code != 1 || !strings.Contains(errOut, "plugin desconocido") || !strings.Contains(errOut, "azure-devops") {
			t.Errorf("%s con plugin desconocido: %d %q (debe nombrar el catálogo y fallar antes de leer el workspace)", verb, code, errOut)
		}
	}
}

func TestIntegrationsConnectAndDiscoverRejectUnconfiguredOrDisabledPlugins(t *testing.T) {
	tests := []struct {
		name, block, wants string
	}{
		{"bloque ausente", "", "no está configurado"},
		{"plugin distinto configurado", "integrations: {}\n", "no está configurado"},
		{"desactivado", "integrations:\n  azure-devops:\n    enabled: false\n", "desactivado"},
	}
	for _, tt := range tests {
		for _, verb := range []string{"connect", "discover"} {
			t.Run(tt.name+" "+verb, func(t *testing.T) {
				e := newADOEnv(t, tt.block)
				e.serveProcess(adoConnectionOK())

				code, out, errOut := e.run(t, verb, "--plugin", "azure-devops")

				if code != 1 || !strings.Contains(errOut, tt.wants) || out != "" {
					t.Fatalf("%s: código %d, stdout %q, stderr %q", tt.name, code, out, errOut)
				}
				if len(e.srv.Requests()) != 0 {
					t.Errorf("sin plugin activo no hay peticiones (%d)", len(e.srv.Requests()))
				}
				if _, err := os.Stat(e.identityPath()); err == nil {
					t.Error("no se guarda ninguna identidad")
				}
			})
		}
	}
}

func TestIntegrationsConnectSucceedsAndStoresTheIdentity(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())

	code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops")

	if code != 0 || errOut != "" {
		t.Fatalf("connect: %d %q", code, errOut)
	}
	for _, want := range []string{"azure-devops", "organización contoso", "Ana Pérez", "guid-1", "connection_data", e.identityPath()} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida debe contener %q: %s", want, out)
		}
	}
	file := readIdentityFile(t, e.identityPath())
	entry := file.Plugins["azure-devops"]["contoso"]
	if entry.ID != "guid-1" || entry.DisplayName != "Ana Pérez" || entry.Source != "connection_data" || entry.ResolvedAt.IsZero() {
		t.Errorf("entrada guardada = %+v", entry)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(e.identityPath())
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("el fichero de identidad debe ser 0600: %v %v", info, err)
		}
	}
	if n := e.srv.Count("GET", adoMembersRoute); n != 0 {
		t.Errorf("con connectionData utilizable no se consultan los miembros (%d)", n)
	}
}

func TestIntegrationsConnectJSON(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())

	code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops", "--json")

	if code != 0 || errOut != "" {
		t.Fatalf("connect --json: %d %q", code, errOut)
	}
	var doc struct {
		Schema       string `json:"schema"`
		Plugin       string `json:"plugin"`
		Account      string `json:"account"`
		Organization string `json:"organization"`
		Status       string `json:"status"`
		Identity     struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Source      string `json:"source"`
			ResolvedAt  string `json:"resolved_at"`
		} `json:"identity"`
		IdentityFile string `json:"identity_file"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if doc.Schema != "axiom.integrations-connect/v1" || doc.Plugin != "azure-devops" || doc.Organization != "contoso" || doc.Account != "contoso" || doc.Status != "connected" {
		t.Errorf("documento = %+v", doc)
	}
	if doc.Identity.ID != "guid-1" || doc.Identity.DisplayName != "Ana Pérez" || doc.Identity.Source != "connection_data" || doc.Identity.ResolvedAt == "" {
		t.Errorf("identidad = %+v", doc.Identity)
	}
	if doc.IdentityFile != e.identityPath() {
		t.Errorf("identity_file = %q", doc.IdentityFile)
	}
	if file := readIdentityFile(t, e.identityPath()); file.Plugins["azure-devops"]["contoso"].ID != "guid-1" {
		t.Errorf("fichero = %+v", file)
	}
}

func TestIntegrationsConnectWithoutUsableIdentityListsCandidatesAndSavesNothing(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adotest.Response{Status: 401})

	code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops")

	if code != 1 || out != "" {
		t.Fatalf("selección pendiente: código %d, stdout %q", code, out)
	}
	for _, want := range []string{"--user", "abc-123  Ana", "def-456  Berto", "ghi-789  Carla", "axiom integrations connect --plugin azure-devops --user"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr debe contener %q: %s", want, errOut)
		}
	}
	if _, err := os.Stat(e.identityPath()); err == nil {
		t.Error("no se guarda ninguna identidad")
	}
}

func TestIntegrationsConnectSelectionRequiredJSON(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adotest.Response{Status: 401})

	code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops", "--json")

	if code != 1 || !strings.Contains(errOut, "--user") {
		t.Fatalf("código %d, stderr %q", code, errOut)
	}
	var doc struct {
		Schema       string `json:"schema"`
		Organization string `json:"organization"`
		Status       string `json:"status"`
		Reason       string `json:"reason"`
		Candidates   []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if doc.Schema != "axiom.integrations-connect/v1" || doc.Status != "selection_required" || doc.Organization != "contoso" || doc.Reason == "" {
		t.Errorf("documento = %+v", doc)
	}
	if len(doc.Candidates) != 3 || doc.Candidates[0].ID != "abc-123" || doc.Candidates[0].DisplayName != "Ana" {
		t.Errorf("candidatos = %+v", doc.Candidates)
	}
	if _, err := os.Stat(e.identityPath()); err == nil {
		t.Error("no se guarda ninguna identidad")
	}
}

func TestIntegrationsConnectWithUserSavesTheMemberSelection(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adotest.Response{Status: 401})

	code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops", "--user", "def-456")

	if code != 0 || errOut != "" || !strings.Contains(out, "member_selection") || !strings.Contains(out, "Berto") {
		t.Fatalf("connect --user: %d %q %q", code, out, errOut)
	}
	entry := readIdentityFile(t, e.identityPath()).Plugins["azure-devops"]["contoso"]
	if entry.ID != "def-456" || entry.Source != "member_selection" || entry.DisplayName != "Berto" {
		t.Errorf("entrada = %+v", entry)
	}
	if n := e.srv.Count("GET", adoConnectionDataRoute); n != 0 {
		t.Errorf("con --user no se consulta connectionData (%d)", n)
	}
}

func TestIntegrationsConnectWithUnknownUserNamesItAndSavesNothing(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())

	code, _, errOut := e.run(t, "connect", "--plugin", "azure-devops", "--user", "zzz")

	if code != 1 || !strings.Contains(errOut, "zzz") || !strings.Contains(errOut, "abc-123  Ana") {
		t.Fatalf("--user desconocido: %d %q", code, errOut)
	}
	if _, err := os.Stat(e.identityPath()); err == nil {
		t.Error("no se guarda ninguna identidad")
	}
}

func TestIntegrationsConnectWithAnEmptyCandidateListIsAClearError(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.srv.Reply("GET", adoConnectionDataRoute, adotest.Response{Status: 401})
	e.srv.Reply("GET", adoMembersRoute, adotest.Response{JSON: map[string]any{"value": []any{}}})

	code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops")

	if code != 1 || out != "" || !strings.Contains(errOut, "ningún miembro") {
		t.Fatalf("lista vacía: %d %q %q", code, out, errOut)
	}
	if _, err := os.Stat(e.identityPath()); err == nil {
		t.Error("no se guarda ninguna identidad")
	}
}

func TestIntegrationsConnectReportsAuthFailuresAsTypedErrors(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.srv.Reply("GET", adoConnectionDataRoute, adotest.Response{Status: 401})
	e.srv.Reply("GET", adoMembersRoute, adotest.Response{Status: 403})

	code, _, errOut := e.run(t, "connect", "--plugin", "azure-devops")

	if code != 1 || !strings.Contains(errOut, "vso.project") {
		t.Fatalf("fallo de autorización: %d %q", code, errOut)
	}
	if _, err := os.Stat(e.identityPath()); err == nil {
		t.Error("no se guarda ninguna identidad")
	}
}

func TestIntegrationsConnectNeverOverwritesACorruptIdentityFile(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())
	if err := os.WriteFile(e.identityPath(), []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}

	code, _, errOut := e.run(t, "connect", "--plugin", "azure-devops")

	if code != 1 || !strings.Contains(errOut, identitystore.FileName) {
		t.Fatalf("fichero corrupto: %d %q", code, errOut)
	}
	if data, _ := os.ReadFile(e.identityPath()); string(data) != "{no es json" {
		t.Errorf("el fichero corrupto no se sobrescribe: %q", data)
	}
}

// ---- discover

func TestIntegrationsDiscoverReadable(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())

	code, out, errOut := e.run(t, "discover", "--plugin", "azure-devops")

	if code != 0 || errOut != "" {
		t.Fatalf("discover: %d %q", code, errOut)
	}
	for _, want := range []string{
		"azure-devops — organization contoso, project Fabrikam, team Fabrikam Team",
		"Tipos de work item (2):", "Bug (Microsoft.VSTS.WorkItemTypes.Bug)",
		"Estados: New [Proposed], Active [InProgress], Removed [Removed]",
		"Campos (1): System.Title",
		"Iteraciones (2):", "Sprint 1  2026-11-01 → 2026-11-14  current",
		"Miembros (3):", "abc-123  Ana", "ghi-789  Carla",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, out)
		}
	}
}

func TestIntegrationsDiscoverJSONAndItWritesNothing(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())
	workspaceBefore, stateBefore := snapshotFiles(t, e.root), snapshotFiles(t, e.stateDir)

	code, out, errOut := e.run(t, "discover", "--plugin", "azure-devops", "--json")

	if code != 0 || errOut != "" {
		t.Fatalf("discover --json: %d %q", code, errOut)
	}
	var doc integrations.Discovery
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if doc.Schema != integrations.DiscoverySchemaV1 || doc.Plugin != "azure-devops" || len(doc.ItemTypes) != 2 || len(doc.Iterations) != 2 || len(doc.Members) != 3 {
		t.Fatalf("documento = %+v", doc)
	}
	if doc.ItemTypes[0].States[1].Category != "InProgress" || doc.ItemTypes[0].Fields[0].ReferenceName != "System.Title" {
		t.Errorf("estados y campos = %+v", doc.ItemTypes[0])
	}
	if strings.Contains(out, "null") {
		t.Errorf("el JSON no debe llevar null: %s", out)
	}
	assertSameSnapshot(t, "workspace", workspaceBefore, snapshotFiles(t, e.root))
	assertSameSnapshot(t, "estado", stateBefore, snapshotFiles(t, e.stateDir))
}

func TestIntegrationsDiscoverReadableWritesNothingEither(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adoConnectionOK())
	workspaceBefore, stateBefore := snapshotFiles(t, e.root), snapshotFiles(t, e.stateDir)

	if code, _, errOut := e.run(t, "discover", "--plugin", "azure-devops"); code != 0 {
		t.Fatalf("discover: %d %q", code, errOut)
	}

	assertSameSnapshot(t, "workspace", workspaceBefore, snapshotFiles(t, e.root))
	assertSameSnapshot(t, "estado", stateBefore, snapshotFiles(t, e.stateDir))
}

func TestIntegrationsDiscoverAuthFailureNamesTheScope(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.srv.Reply("GET", adoTypesRoute, adotest.Response{JSON: map[string]any{"value": []any{}}})
	e.srv.Reply("GET", adoIterationsRoute, adotest.Response{JSON: map[string]any{"value": []any{}}})
	e.srv.Reply("GET", adoMembersRoute, adotest.Response{Status: 403})

	code, out, errOut := e.run(t, "discover", "--plugin", "azure-devops")

	if code != 1 || out != "" || !strings.Contains(errOut, "vso.project") {
		t.Fatalf("fallo de autorización: %d %q %q", code, out, errOut)
	}
}

// ---- errores tipados y puertos

func TestIntegrationsPortVerbsReturnTypedErrors(t *testing.T) {
	verbs := map[string]func(cwd, plugin string) error{
		"connect": func(cwd, plugin string) error {
			_, err := connectIntegration(t.Context(), cwd, plugin, "")
			return err
		},
		"discover": func(cwd, plugin string) error {
			_, err := discoverIntegration(t.Context(), cwd, plugin)
			return err
		},
	}
	tests := []struct {
		name   string
		block  string
		plugin string
		setup  func(t *testing.T)
		want   error
	}{
		{"plugin desconocido", adoBlock, "jira", nil, ErrIntegrationPluginUnknown},
		{"bloque ausente", "", "azure-devops", nil, ErrIntegrationNotConfigured},
		{"plugin desactivado", "integrations:\n  azure-devops:\n    enabled: false\n", "azure-devops", nil, ErrIntegrationDisabled},
		{"configuración incompleta", "integrations:\n  azure-devops:\n    organization: contoso\n", "azure-devops", nil, ErrIntegrationConfigInvalid},
		{"PAT ausente", adoBlock, "azure-devops", func(t *testing.T) { t.Setenv(adoPATEnv, "") }, secrets.ErrNotFound},
	}
	for verb, call := range verbs {
		for _, tt := range tests {
			t.Run(verb+" "+tt.name, func(t *testing.T) {
				e := newADOEnv(t, tt.block)
				e.serveProcess(adoConnectionOK())
				if tt.setup != nil {
					tt.setup(t)
				}

				err := call(e.root, tt.plugin)

				if !errors.Is(err, tt.want) {
					t.Fatalf("%s: errors.Is(%v, %v) = false", tt.name, err, tt.want)
				}
				if tt.want == secrets.ErrNotFound && len(e.srv.Requests()) != 0 {
					t.Errorf("sin PAT no hay peticiones (%d)", len(e.srv.Requests()))
				}
				var verbErr *integrationVerbError
				if tt.want != secrets.ErrNotFound && !errors.As(err, &verbErr) {
					t.Errorf("%s debe ser un *integrationVerbError: %T", tt.name, err)
				}
			})
		}
	}
}

func TestIntegrationsPluginWithoutPortsFailsWithTheTypedErrorAndNoEffects(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	stateDir := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	workspaceBefore := snapshotFiles(t, root)

	for _, tc := range []struct{ verb, port string }{{"connect", "identidad"}, {"discover", "descubrimiento"}} {
		code, out, errOut := runIntegrationsCLI(t, "", tc.verb, "--plugin", "fake", "--cwd", root)
		if code != 1 || out != "" || !strings.Contains(errOut, "fake") || !strings.Contains(errOut, "puerto de "+tc.port) {
			t.Errorf("%s: %d %q %q", tc.verb, code, out, errOut)
		}
	}

	if _, err := connectIntegration(t.Context(), root, "fake", ""); !errors.Is(err, integrations.ErrPortUnsupported) {
		t.Errorf("connect: %v", err)
	}
	if _, err := discoverIntegration(t.Context(), root, "fake"); !errors.Is(err, integrations.ErrPortUnsupported) {
		t.Errorf("discover: %v", err)
	}
	if fake.Calls() != 0 {
		t.Errorf("ningún verbo de puertos entrega eventos (llamadas %d)", fake.Calls())
	}
	assertSameSnapshot(t, "workspace", workspaceBefore, snapshotFiles(t, root))
	if got := snapshotFiles(t, stateDir); len(got) != 0 {
		t.Errorf("no se crea ningún fichero de estado: %v", got)
	}
	assertEmptyChangeRoot(t, changeRoot)
}

// ---- ayuda (RF-INT-18)

func TestIntegrationsHelpListsExactlyTheSubcommands(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help"}, {"inspect"}} {
		code, out, _ := runIntegrationsCLI(t, "", args...)
		if args[0] == "inspect" && code == 0 {
			t.Errorf("subcomando desconocido debe salir con código distinto de cero (%d)", code)
		}
		var listed []string
		inList := false
		for _, line := range strings.Split(out, "\n") {
			switch {
			case strings.HasPrefix(line, "Subcomandos:"):
				inList = true
			case inList && strings.HasPrefix(line, "  "):
				listed = append(listed, strings.Fields(line)[0])
			case inList:
				inList = false
			}
		}
		if strings.Join(listed, ",") != "sync,emit,secret,connect,discover,tasks" {
			t.Errorf("%v: la ayuda de la familia lista %v, quería exactamente sync, emit, secret, connect, discover y tasks", args, listed)
		}
	}
	code, _, errOut := runIntegrationsCLI(t, "", "inspect")
	if code == 0 || !strings.Contains(errOut, "sync, emit, secret, connect, discover y tasks") {
		t.Errorf("el mensaje de subcomando desconocido debe enumerar todos: %d %q", code, errOut)
	}
}

func TestIntegrationsConnectAndDiscoverHelp(t *testing.T) {
	for _, verb := range []string{"connect", "discover"} {
		code, out, _ := runIntegrationsCLI(t, "", verb, "--help")
		if code != 0 || !strings.Contains(out, "Uso: axiom integrations "+verb) || !strings.Contains(out, "--plugin") || !strings.Contains(out, "--json") || !strings.Contains(out, "--cwd") {
			t.Errorf("%s --help: %d %q", verb, code, out)
		}
	}
	if _, out, _ := runIntegrationsCLI(t, "", "connect", "--help"); !strings.Contains(out, "--user") || !strings.Contains(out, "integration-identities.json") {
		t.Errorf("connect --help debe documentar --user y el fichero: %q", out)
	}
}

func TestIntegrationsReconnectReplacesTheEntryAndSaysSo(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	e.serveProcess(adotest.Response{Status: 401})
	if code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops", "--user", "def-456"); code != 0 || strings.Contains(out, "Sustituye") {
		t.Fatalf("la primera conexión no sustituye nada: %d %q %q", code, out, errOut)
	}
	// Reconexión por connectionData: la entrada de contoso pasa a connection_data.
	srv := adotest.NewServer(t, nil)
	for _, route := range adoProcessRoutes(adoConnectionOK()) {
		srv.Reply("GET", route.path, route.response)
	}
	adotest.Install(t, srv)

	code, out, errOut := e.run(t, "connect", "--plugin", "azure-devops")

	if code != 0 || errOut != "" || !strings.Contains(out, "Sustituye la identidad anterior") || !strings.Contains(out, "def-456") || !strings.Contains(out, "member_selection") {
		t.Fatalf("reconexión: %d %q %q", code, out, errOut)
	}
	entry := readIdentityFile(t, e.identityPath()).Plugins["azure-devops"]["contoso"]
	if entry.ID != "guid-1" || entry.Source != "connection_data" {
		t.Errorf("la entrada debe pasar a connection_data: %+v", entry)
	}
}
