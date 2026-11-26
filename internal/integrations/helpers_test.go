package integrations

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// fixedNow es el reloj fijo de las pruebas del despachador.
var fixedNow = time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)

func nowFunc() time.Time { return fixedNow }

// workspaceFixture es un espacio de trabajo temporal con la carpeta ordinaria del cambio x.
type workspaceFixture struct {
	WS     string
	Change string
	Root   string // <ws>/openspec/changes/x
}

func newWorkspace(t *testing.T) workspaceFixture {
	t.Helper()
	ws, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := workspaceFixture{WS: ws, Change: "x", Root: filepath.Join(ws, "openspec", "changes", "x")}
	if err := os.MkdirAll(f.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

// newGitWorkspace es newWorkspace con un repositorio git inicializado en la raíz.
func newGitWorkspace(t *testing.T) workspaceFixture {
	t.Helper()
	f := newWorkspace(t)
	gitIsolated(t)
	runGit(t, f.WS, "init", "-q", "-b", "main")
	runGit(t, f.WS, "config", "user.name", "Prueba")
	runGit(t, f.WS, "config", "user.email", "prueba@example.test")
	writeTestFile(t, filepath.Join(f.WS, "README.md"), "x\n")
	runGit(t, f.WS, "add", "-A")
	runGit(t, f.WS, "commit", "-q", "-m", "base")
	return f
}

// archivedFolderOf crea (y devuelve) la carpeta archivada <fecha>-<cambio> del espacio.
func archivedFolderOf(t *testing.T, f workspaceFixture, name string) Folder {
	t.Helper()
	root := filepath.Join(f.WS, "openspec", "changes", "archive", name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return Folder{WorkspaceRoot: f.WS, Root: root}
}

func (f workspaceFixture) Folder() Folder { return Folder{WorkspaceRoot: f.WS, Root: f.Root} }

func gitIsolated(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR"} {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	return string(data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// testPlugin es un plugin programable para las pruebas internas del paquete. Replica el doble de
// integrationstest, que no se puede importar desde aquí: integrationstest importa este paquete y
// Go no admite ese ciclo en las pruebas internas.
type testPlugin struct {
	mu sync.Mutex

	m          manifest.Manifest
	received   []Delivery
	delivered  []Event
	byKey      map[string]Outcome
	links      []Link
	failAlways error
	failTimes  int
	failErr    error
	failEvents map[manifest.EventType]error
	readToken  bool
	leakToken  bool
	block      <-chan struct{}
	panicWith  any
	hasPanic   bool
	warnings   []string
	hook       func(Delivery) // se llama al empezar cada entrega, antes de cualquier fallo programado
}

func testManifest() manifest.Manifest {
	return manifest.Manifest{
		Schema:       manifest.SchemaV1,
		ID:           "fake",
		Title:        "Plugin de prueba en memoria",
		Capabilities: []manifest.Capability{manifest.CapItemsRead, manifest.CapItemsTransition},
		Events: []manifest.EventType{
			manifest.EventGateSpecApproved,
			manifest.EventGateDesignApproved,
			manifest.EventRoleStart,
			manifest.EventRoleApplyApproved,
			manifest.EventArchive,
		},
		Config:   []manifest.ConfigField{{Key: "project", Type: manifest.FieldString, Required: true, Description: "Proyecto del tracker"}},
		Secrets:  []manifest.SecretSpec{{Name: "token", Description: "Token de acceso", Required: true}},
		Identity: manifest.IdentitySpec{Source: manifest.IdentityGitActor},
	}
}

func newTestPlugin() *testPlugin {
	return &testPlugin{m: testManifest(), byKey: map[string]Outcome{}, failEvents: map[manifest.EventType]error{}}
}

func (p *testPlugin) Manifest() manifest.Manifest { return p.m.Clone() }

func (p *testPlugin) Deliver(ctx context.Context, d Delivery) (Outcome, error) {
	p.mu.Lock()
	snapshot := d
	snapshot.Links = append([]Link(nil), d.Links...)
	p.received = append(p.received, snapshot)
	block, hasPanic, panicVal, readToken, leakToken := p.block, p.hasPanic, p.panicWith, p.readToken, p.leakToken
	hook := p.hook
	p.mu.Unlock()

	if hook != nil {
		hook(d)
	}
	if hasPanic {
		panic(panicVal)
	}
	if block != nil {
		<-block
	}
	if err := ctx.Err(); err != nil {
		return Outcome{}, err
	}
	var token string
	if readToken {
		secret, err := d.Secrets.Secret(ctx, "token")
		if err != nil {
			return Outcome{}, err
		}
		token = secret.Reveal()
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.failEvents[d.Event.Type]; err != nil {
		return Outcome{}, err
	}
	if p.failAlways != nil {
		return Outcome{}, p.failAlways
	}
	if p.failTimes > 0 {
		p.failTimes--
		return Outcome{}, p.failErr
	}
	if leakToken {
		return Outcome{}, fmt.Errorf("el tracker rechazó la credencial %s", token)
	}
	if out, seen := p.byKey[d.Event.IdempotencyKey]; seen {
		return out, nil
	}
	out := Outcome{Links: append([]Link(nil), p.links...), Warnings: append([]string(nil), p.warnings...)}
	p.byKey[d.Event.IdempotencyKey] = out
	p.delivered = append(p.delivered, d.Event)
	return out, nil
}

func (p *testPlugin) calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.received)
}

func (p *testPlugin) deliveredKeys() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	keys := make([]string, len(p.delivered))
	for i, e := range p.delivered {
		keys[i] = e.IdempotencyKey
	}
	return keys
}

func (p *testPlugin) setFailAlways(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failAlways = err
}

func (p *testPlugin) recover() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failAlways, p.failTimes, p.failErr = nil, 0, nil
	p.failEvents = map[manifest.EventType]error{}
}

func (p *testPlugin) setLinks(links ...Link) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.links = append([]Link(nil), links...)
}

// installTestPlugin instala el catálogo, el registro y un almacén de secretos en memoria con el
// plugin, y lo restaura al terminar. Devuelve el almacén en memoria.
func installTestPlugin(t *testing.T, plugins ...*testPlugin) *secrets.Memory {
	t.Helper()
	manifests := make([]manifest.Manifest, len(plugins))
	list := make([]Plugin, len(plugins))
	for i, p := range plugins {
		manifests[i] = p.Manifest()
		list[i] = p
	}
	manifest.SetCatalogForTest(t, manifests...)
	SetDefaultRegistryForTest(t, list...)
	memory := secrets.NewMemory()
	secrets.SetDefaultForTest(t, secrets.NewChain(secrets.Env{Getenv: func(string) string { return "" }}, memory))
	return memory
}

// newTestDispatcher compone un despachador con el reloj fijo y la configuración de los plugins
// dados activos (project: p). Los plazos son holgados salvo que la prueba los cambie.
func newTestDispatcher(f workspaceFixture, ids ...string) *Dispatcher {
	registry, _ := ResolveRegistry(nil)
	config := &workspace.IntegrationsConfig{}
	for _, id := range ids {
		config.Plugins = append(config.Plugins, workspace.IntegrationPluginConfig{ID: id, Enabled: true, Values: map[string]any{"project": "p"}, Line: 2})
	}
	return &Dispatcher{
		Registry:      registry,
		Config:        config,
		Secrets:       secrets.Default(),
		WorkspaceRoot: f.WS,
		Now:           nowFunc,
		Immediate:     10 * time.Second,
		PerOp:         30 * time.Second,
	}
}
