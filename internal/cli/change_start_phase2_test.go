package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const startItemPath = "/contoso/Fabrikam/_apis/wit/workitems/2525"

// stagedChange deja el WI 2525 en staging con la fase 1 y devuelve el nombre derivado.
func (e *adoEnv) stagedChange(t *testing.T) string {
	t.Helper()
	e.seedStory(2525, "Alta de usuarios", "New")
	e.srv.SeedWorkItem(2525, 1, map[string]any{
		"System.WorkItemType": "User Story", "System.Title": "Alta de usuarios", "System.State": "New",
		"System.Description": "<p>Hola <b>mundo</b></p>", "System.Tags": "a; b",
	})
	code, out, errOut := e.startCLI(t, "--from", "2525", "--json")
	doc := decodeStart(t, out)
	if code != 0 || doc.Status != "staged" {
		t.Fatalf("fase 1: código %d, doc %+v, error %q", code, doc, errOut)
	}
	return doc.Change
}

// prepareChange deja el cambio preparado como lo haría `axiom branch ensure`: registro de ramas en el
// carril, HEAD en la rama del cambio y, en SDD, la carpeta del cambio.
func (e *adoEnv) prepareChange(t *testing.T, change string, lane branch.Lane, worktree string) {
	t.Helper()
	common := filepath.Join(e.root, ".git")
	if err := os.WriteFile(filepath.Join(common, "HEAD"), []byte("ref: refs/heads/inc/"+change+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := branch.RecordRepo{
		Name: "axiom", GitRoot: e.root, CommonDir: common, Role: branch.RepoChangeBranch,
		Branches: []branch.RecordBranch{{Name: "inc/" + change, Kind: branch.KindChange, Worktree: worktree}},
	}
	rec := branch.Record{
		Schema: branch.RegistrySchema, Lane: lane, Change: change, Mode: branch.ModeWorktree,
		ModeSource: branch.ModeSourceDefault, PlanningRepo: e.root, Repositories: []branch.RecordRepo{repo},
	}
	if err := registry.New().Save(common, rec); err != nil {
		t.Fatal(err)
	}
	if lane == branch.LaneSDD {
		if err := os.MkdirAll(filepath.Join(e.root, "openspec", "changes", change), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestChangeStartAdoptEndToEnd(t *testing.T) {
	e := newStartEnv(t)
	change := e.stagedChange(t)
	e.prepareChange(t, change, branch.LaneSDD, "")

	code, out, errOut := e.startCLI(t, "--adopt", "--change", change, "--json")
	doc := decodeStart(t, out)

	if code != 0 || doc.Phase != "adopt" || doc.Status != "adopted" || doc.Lane != "sdd" || doc.Link != "created" || doc.Backlink != "added" ||
		doc.ExternalRef != "ado:2525" || doc.Change != change {
		t.Fatalf("código %d, doc %+v, error %q", code, doc, errOut)
	}
	if want := filepath.Join(e.root, "openspec", "changes", change, "context"); filepath.Clean(doc.Context) != want {
		t.Fatalf("context = %q, quiero %q", doc.Context, want)
	}
	if _, err := os.Stat(filepath.Join(doc.Context, "work-item.md")); err != nil {
		t.Fatalf("work-item.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.root, ".git", "axiom", "change-start", "v1", change)); !os.IsNotExist(err) {
		t.Fatalf("el staging debe retirarse: %v", err)
	}
	links, err := os.ReadFile(filepath.Join(e.root, "openspec", "changes", change, "integrations.yaml"))
	if err != nil || !strings.Contains(string(links), "kind: work_item") || !strings.Contains(string(links), "source: change.start") {
		t.Fatalf("integrations.yaml: %v\n%s", err, links)
	}
	if got, _ := e.srv.WorkItem(2525); got.Fields["System.Tags"] != "a; b; axiom:"+change {
		t.Fatalf("etiquetas = %q", got.Fields["System.Tags"])
	}
	if e.srv.Count("PATCH", startItemPath) != 1 {
		t.Fatalf("una sola escritura en el tracker: %v", e.srv.Requests())
	}

	// Segunda ejecución: already_adopted, sin ninguna petición ni escritura.
	requests := len(e.srv.Requests())
	before := snapshotFiles(t, filepath.Join(e.root, "openspec"))
	code, out, _ = e.startCLI(t, "--adopt", "--change", change, "--json")
	again := decodeStart(t, out)
	if code != 0 || again.Status != "already_adopted" || again.Warnings == nil {
		t.Fatalf("segunda adopción: código %d, doc %+v", code, again)
	}
	if len(e.srv.Requests()) != requests {
		t.Fatalf("already_adopted no usa la red: %v", e.srv.Requests()[requests:])
	}
	assertSameSnapshot(t, "segunda adopción", before, snapshotFiles(t, filepath.Join(e.root, "openspec")))

	// El texto de la salida no incluye texto libre del tracker.
	code, textOut, _ := e.startCLI(t, "--adopt", "--change", change)
	if code != 0 || !strings.Contains(textOut, change) || strings.Contains(textOut, "Hola") {
		t.Fatalf("salida de texto: código %d, %q", code, textOut)
	}
}

func TestChangeStartAdoptODD(t *testing.T) {
	e := newStartEnv(t)
	change := e.stagedChange(t)
	e.prepareChange(t, change, branch.LaneODD, "")

	code, out, errOut := e.startCLI(t, "--adopt", "--change", change, "--lane", "odd", "--json")
	doc := decodeStart(t, out)

	if code != 0 || doc.Status != "adopted" || doc.Lane != "odd" || doc.Link != "not_applicable" || doc.Backlink != "added" {
		t.Fatalf("código %d, doc %+v, error %q", code, doc, errOut)
	}
	if _, err := os.Stat(filepath.Join(e.root, "odd", "context", change, "work-item.md")); err != nil {
		t.Fatalf("contexto ODD: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.root, "openspec", "changes", change)); !os.IsNotExist(err) {
		t.Fatalf("ODD no crea nada bajo openspec/changes/: %v", err)
	}
}

func TestChangeStartAdoptTrackerDownWarnsExitZero(t *testing.T) {
	e := newStartEnv(t)
	change := e.stagedChange(t)
	e.prepareChange(t, change, branch.LaneSDD, "")
	e.srv.Reply("PATCH", startItemPath, adotest.Response{Status: 503})

	code, out, errOut := e.startCLI(t, "--adopt", "--change", change, "--json")
	doc := decodeStart(t, out)

	if code != 0 || doc.Status != "adopted" || doc.Backlink != "pending" || doc.Refusal != nil {
		t.Fatalf("código %d, doc %+v, error %q", code, doc, errOut)
	}
	if joined := strings.Join(doc.Warnings, "\n"); !strings.Contains(joined, "axiom change start --adopt --change "+change) {
		t.Fatalf("el aviso no nombra el reintento: %v", doc.Warnings)
	}
	if _, err := os.Stat(filepath.Join(doc.Context, "work-item.md")); err != nil {
		t.Fatalf("el contexto queda aplicado: %v", err)
	}
}

func TestChangeStartAdoptNotPreparedContinuations(t *testing.T) {
	// Solo existe el registro ODD y se pide el carril sdd por defecto: la continuación incluye --lane odd.
	e := newStartEnv(t)
	change := e.stagedChange(t)
	worktree := t.TempDir()
	e.prepareChange(t, change, branch.LaneODD, worktree)
	before := snapshotFiles(t, filepath.Join(e.root, ".git", "axiom", "change-start", "v1", change))

	code, out, _ := e.startCLI(t, "--adopt", "--change", change, "--json")
	ref := decodeStart(t, out).Refusal
	if code != 1 || ref == nil || ref.Code != "change_not_prepared" ||
		!strings.Contains(ref.Continuation, "--lane odd") || !strings.Contains(ref.Continuation, "--cwd") {
		t.Fatalf("código %d, rechazo %+v", code, ref)
	}
	assertSameSnapshot(t, "staging", before, snapshotFiles(t, filepath.Join(e.root, ".git", "axiom", "change-start", "v1", change)))

	// Sin registro alguno: la continuación es `axiom branch ensure`.
	other := newStartEnv(t)
	name := other.stagedChange(t)
	code, out, _ = other.startCLI(t, "--adopt", "--change", name, "--lane", "odd", "--json")
	ref = decodeStart(t, out).Refusal
	if code != 1 || ref == nil || ref.Continuation != "axiom branch ensure --change "+name+" --lane odd --json" {
		t.Fatalf("código %d, rechazo %+v", code, ref)
	}

	// Registro presente con el HEAD en otra rama y worktree: el mensaje nombra el --adopt con --cwd.
	head := newStartEnv(t)
	hn := head.stagedChange(t)
	head.prepareChange(t, hn, branch.LaneSDD, worktree)
	if err := os.WriteFile(filepath.Join(head.root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = head.startCLI(t, "--adopt", "--change", hn, "--json")
	ref = decodeStart(t, out).Refusal
	if code != 1 || ref == nil || !strings.Contains(ref.Message, "axiom change start --adopt --change "+hn) || !strings.Contains(ref.Message, "--cwd") {
		t.Fatalf("código %d, rechazo %+v", code, ref)
	}
}

func TestChangeStartAdoptStagingNotFound(t *testing.T) {
	e := newStartEnv(t)
	e.seedStory(2525, "Alta de usuarios", "New")
	e.prepareChange(t, "US-2525-alta-de-usuarios", branch.LaneSDD, "")
	code, out, _ := e.startCLI(t, "--adopt", "--change", "US-2525-alta-de-usuarios", "--json")
	ref := decodeStart(t, out).Refusal
	if code != 1 || ref == nil || ref.Code != "staging_not_found" || ref.Continuation != "axiom change start --from <id|url>" {
		t.Fatalf("código %d, rechazo %+v", code, ref)
	}
}

func TestChangeStartDiscardEndToEnd(t *testing.T) {
	e := newStartEnv(t)
	change := e.stagedChange(t)
	requests := len(e.srv.Requests())
	staging := filepath.Join(e.root, ".git", "axiom", "change-start", "v1", change)

	code, out, errOut := e.startCLI(t, "--discard", "--change", change, "--json")
	doc := decodeStart(t, out)

	if code != 0 || doc.Phase != "discard" || doc.Status != "discarded" || doc.Change != change {
		t.Fatalf("código %d, doc %+v, error %q", code, doc, errOut)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatalf("el staging debe desaparecer: %v", err)
	}
	if len(e.srv.Requests()) != requests {
		t.Fatalf("--discard no usa la red: %v", e.srv.Requests()[requests:])
	}

	code, out, _ = e.startCLI(t, "--discard", "--change", change, "--json")
	again := decodeStart(t, out)
	if code != 0 || again.Status != "nothing_to_discard" || len(again.Warnings) == 0 {
		t.Fatalf("sin staging: código %d, doc %+v", code, again)
	}
}

func TestChangeStartDiscardTraversalUsage(t *testing.T) {
	e := newStartEnv(t)
	change := e.stagedChange(t)
	staging := filepath.Join(e.root, ".git", "axiom", "change-start", "v1", change)
	before := snapshotFiles(t, filepath.Join(e.root, ".git", "axiom"))

	for _, bad := range []string{"../../algo", "..", "a/b", `a\b`, "archive"} {
		code, out, _ := e.startCLI(t, "--discard", "--change", bad, "--json")
		doc := decodeStart(t, out)
		if code != 2 || doc.Refusal == nil || doc.Refusal.Code != "usage_invalid" {
			t.Fatalf("%q: código %d, doc %+v", bad, code, doc)
		}
	}
	if _, err := os.Stat(staging); err != nil {
		t.Fatalf("no se borra nada: %v", err)
	}
	assertSameSnapshot(t, "common dir", before, snapshotFiles(t, filepath.Join(e.root, ".git", "axiom")))
}
