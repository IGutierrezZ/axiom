package changestart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

var fixedNow = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

// newRepo crea un repositorio git mínimo (solo el directorio .git) y su entorno: el workspace es la
// raíz y el planning home, openspec/.
func newRepo(t *testing.T) Env {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Env{
		WorkspaceRoot: root, PlanningHome: filepath.Join(root, "openspec"),
		GitRoot: root, CommonDir: filepath.Join(root, ".git"),
		Now: func() time.Time { return fixedNow }, LockWait: 400 * time.Millisecond,
	}
}

// stubTracker es un Tracker en memoria que cuenta las llamadas.
type stubTracker struct {
	item       integrations.ExternalItem
	readErr    error
	cat        integrations.StateCategory
	catErr     error
	keys       []manifest.TypePrefix
	refErr     error
	blobs      map[string]fakeBlob
	reads      int
	stateCalls int
	stateType  string
	downloads  *fakeTracker
}

func newStub() *stubTracker {
	return &stubTracker{
		item: integrations.ExternalItem{
			ID: "2525", Type: "User Story", Title: "Alta de usuarios", State: "New",
			URL:         "https://dev.azure.com/contoso/Fabrikam/_workitems/edit/2525",
			Description: "<p>Hola <b>mundo</b></p>",
		},
		cat:  integrations.StateCategory{Category: "Proposed", Known: true},
		keys: []manifest.TypePrefix{{Type: "User Story", Prefix: "US"}, {Type: "Bug", Prefix: "BUG"}, {Type: "Defect", Prefix: "DEF"}},
	}
}

func (s *stubTracker) PluginID() string                { return "azure-devops" }
func (s *stubTracker) RefAlias() string                { return "ado" }
func (s *stubTracker) ItemKeys() []manifest.TypePrefix { return s.keys }
func (s *stubTracker) ParseRef(raw string) (string, error) {
	if s.refErr != nil {
		return "", s.refErr
	}
	raw = strings.TrimSpace(raw)
	if !regexp.MustCompile(`^[1-9][0-9]{0,9}$`).MatchString(raw) {
		return "", errors.New("referencia no válida")
	}
	return raw, nil
}

func (s *stubTracker) Read(context.Context, string) (integrations.ExternalItem, error) {
	s.reads++
	return s.item, s.readErr
}

func (s *stubTracker) StateCategory(_ context.Context, typ, _ string) (integrations.StateCategory, error) {
	s.stateCalls++
	s.stateType = typ
	return s.cat, s.catErr
}

func (s *stubTracker) Download(ctx context.Context, url string, limit int64) (integrations.Attachment, error) {
	if s.downloads == nil {
		s.downloads = newFake(s.blobs)
	}
	return s.downloads.download(ctx, url, limit)
}

func (s *stubTracker) LinkBack(context.Context, string, string, func([]string) error) (integrations.BackLink, error) {
	return integrations.BackLink{}, errors.New("la fase 1 no escribe en el tracker")
}

func mustStart(t *testing.T, env Env, tr Tracker) StartResult {
	t.Helper()
	res, err := Start(context.Background(), env, tr, "2525")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return res
}

func refusalOf(t *testing.T, err error) *Refusal {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) {
		t.Fatalf("error = %v, quiero un *Refusal", err)
	}
	return r
}

func startRefusal(t *testing.T, env Env, tr Tracker) *Refusal {
	t.Helper()
	_, err := Start(context.Background(), env, tr, "2525")
	if err == nil {
		t.Fatal("Start debía rechazar")
	}
	return refusalOf(t, err)
}

// stagings devuelve los nombres de los stagings publicados bajo <v1>.
func stagings(t *testing.T, env Env) []string {
	t.Helper()
	entries, err := os.ReadDir(env.v1())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	return out
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

func contextJSON(change, plugin, id, backlink string) string {
	doc := ContextDoc{Schema: ContextSchema, Change: change, Plugin: plugin, ID: id, Backlink: backlink}
	data, _ := json.Marshal(doc)
	return string(data)
}

// seedStaging deja un staging válido de (plugin, id) bajo el nombre dado.
func seedStaging(t *testing.T, env Env, name, id string) {
	t.Helper()
	writeFile(t, filepath.Join(env.StagingDir(name), "context", "work-item.json"), contextJSON(name, "azure-devops", id, ""))
}

// seedRecord siembra un registro de ramas abierto con su rama de cambio en worktree.
func seedRecord(t *testing.T, env Env, lane branch.Lane, change, worktree string) {
	t.Helper()
	repo := branch.RecordRepo{
		Name: "axiom", GitRoot: env.GitRoot, CommonDir: env.CommonDir, Role: branch.RepoChangeBranch,
		Branches: []branch.RecordBranch{{Name: "inc/" + change, Kind: branch.KindChange, Worktree: worktree}},
	}
	rec := branch.Record{
		Schema: branch.RegistrySchema, Lane: lane, Change: change, Mode: branch.ModeWorktree,
		ModeSource: branch.ModeSourceDefault, PlanningRepo: env.GitRoot, Repositories: []branch.RecordRepo{repo},
	}
	if err := registry.New().Save(env.CommonDir, rec); err != nil {
		t.Fatalf("sembrar registro: %v", err)
	}
}

func sddLinkYAML(change string) string {
	return fmt.Sprintf("schema: %s\nchange: %s\nlinks:\n  - plugin: azure-devops\n    kind: work_item\n    id: \"2525\"\n    source: change.start\n    recorded_at: 2026-11-26T10:00:00Z\n",
		integrations.LinksSchemaV1, change)
}
