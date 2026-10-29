package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Listados que agregan los cambios de los worktrees registrados y marcan los pendientes de archivar
// (INC-25 S10-c3; REQ-25.34 y REQ-25.20, design D-16, D-17 y D-20). Pruebas con git real, entorno
// aislado y sin red: el cambio se crea con `ensure` en modo `worktree`, de modo que el registro local
// y el worktree son los que produce el motor; solo el worktree de la rama de archivado, que exige un
// cambio fusionado, se registra a mano en ese mismo registro.

// listingResponse es la respuesta de `GET /api/increments` ya decodificada: el cuerpo (un array), los
// avisos de la cabecera `X-Axiom-Notes` (un array JSON) y la cabecera tal como viajó.
type listingResponse struct {
	list      []IncrementSummaryDTO
	raw       []map[string]any
	notes     []string
	noteBytes string
}

func getListing(t *testing.T, svc *Service) listingResponse {
	t.Helper()
	rr := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/increments", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/increments = %d: %s", rr.Code, rr.Body.String())
	}
	response := listingResponse{noteBytes: rr.Header().Get(notesHeaderName)}
	if err := json.Unmarshal(rr.Body.Bytes(), &response.list); err != nil {
		t.Fatalf("el cuerpo de GET /api/increments debe seguir siendo un array de incrementos: %v\n%s", err, rr.Body.String())
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response.raw); err != nil {
		t.Fatalf("el cuerpo debe ser un array de objetos: %v", err)
	}
	if response.noteBytes != "" {
		if err := json.Unmarshal([]byte(response.noteBytes), &response.notes); err != nil {
			t.Fatalf("%s debe ser un array JSON de cadenas: %v\n%s", notesHeaderName, err, response.noteBytes)
		}
	}
	return response
}

// summary devuelve el incremento de ese nombre y tipo, y exige que figure una sola vez.
func (r listingResponse) summary(t *testing.T, name, kind string) IncrementSummaryDTO {
	t.Helper()
	var found []IncrementSummaryDTO
	for _, increment := range r.list {
		if increment.Name == name && increment.Type == kind {
			found = append(found, increment)
		}
	}
	if len(found) != 1 {
		t.Fatalf("el %s %q debía figurar una sola vez, figura %d (listado: %+v)", kind, name, len(found), r.list)
	}
	return found[0]
}

func (r listingResponse) rawSummary(t *testing.T, name string) map[string]any {
	t.Helper()
	for _, increment := range r.raw {
		if increment["name"] == name {
			return increment
		}
	}
	t.Fatalf("%q no figura en el cuerpo: %v", name, r.raw)
	return nil
}

// noteMentioning devuelve el aviso que contiene todos los textos dados, o falla.
func (r listingResponse) noteMentioning(t *testing.T, parts ...string) string {
	t.Helper()
	for _, note := range r.notes {
		matches := true
		for _, part := range parts {
			matches = matches && strings.Contains(note, part)
		}
		if matches {
			return note
		}
	}
	t.Fatalf("ningún aviso de %s contiene %q: %q", notesHeaderName, parts, r.notes)
	return ""
}

// worktreeChange crea el cambio con `ensure` en modo `worktree` y devuelve el worktree de `feat/<name>`.
func worktreeChange(t *testing.T, svc *Service, ws workspacetest.Workspace, name string) string {
	t.Helper()
	if _, err := svc.CreateIncrement(CreateIncrementRequest{Name: name, Intent: "Propósito " + name}); err != nil {
		t.Fatalf("crear el cambio %s: %v", name, err)
	}
	return worktreeOfBranch(t, ws.Master, "feat/"+name)
}

// REQ-25.34 escenarios «Un cambio de un worktree figura en el listado» y «El detalle de un cambio en
// worktree se resuelve»: el checkout principal no tiene el cambio, el listado lo da como activo con la
// ruta de su worktree y su rama, y el detalle, los roles y el relevo se leen del worktree.
func TestIncrementsListsAChangeThatLivesOnlyInItsWorktree(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktree := worktreeChange(t, svc, ws, "x")
	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	workspacetest.WriteFile(t, filepath.Join(worktree, "openspec", "changes", "x", "design.md"), "# Diseño de x\n")

	listing := getListing(t, svc)

	x := listing.summary(t, "x", "active")
	wantSameDirectory(t, "worktreePath", x.WorktreePath, worktree)
	if x.Branch != "feat/x" {
		t.Errorf("branch = %q, se esperaba feat/x", x.Branch)
	}
	raw := listing.rawSummary(t, "x")
	if _, present := raw["worktreePath"]; !present {
		t.Errorf("el campo de INC-25 es worktreePath (camelCase): %v", raw)
	}
	if _, present := raw["pendingArchive"]; present {
		t.Errorf("pendingArchive es omitempty y x no está pendiente: %v", raw)
	}

	detail := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/increments/x", nil))
	if detail.Code != http.StatusOK {
		t.Fatalf("GET /api/increments/x = %d: %s", detail.Code, detail.Body.String())
	}
	var dto IncrementDetailDTO
	if err := json.Unmarshal(detail.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if !dto.HasProposal || !strings.Contains(dto.Proposal, "Propósito x") {
		t.Errorf("el detalle debía leer proposal.md del worktree: %+v", dto)
	}
	if dto.Summary.Branch != "feat/x" {
		t.Errorf("el detalle debía indicar la rama del worktree: %+v", dto.Summary)
	}

	roles := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(roles, httptest.NewRequest(http.MethodGet, "/api/roles?change=x", nil))
	if roles.Code != http.StatusOK {
		t.Errorf("GET /api/roles?change=x = %d, design.md solo existe en el worktree: %s", roles.Code, roles.Body.String())
	}

	handoffs := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(handoffs, httptest.NewRequest(http.MethodGet, "/api/handoffs?change=x", nil))
	if strings.Contains(handoffs.Body.String(), "no encontrado") {
		t.Errorf("GET /api/handoffs?change=x debía resolver el cambio del worktree: %s", handoffs.Body.String())
	}
}

// REQ-25.34 escenario «El worktree prevalece sobre el checkout principal»: el cambio figura una sola
// vez y con los datos del worktree.
func TestIncrementsWorktreePrevailsOverTheMainCheckout(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktree := worktreeChange(t, svc, ws, "x")
	workspacetest.WriteFile(t, filepath.Join(worktree, "openspec", "changes", "x", "tasks.md"), "- [x] 1.1 hecha\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "x", "proposal.md"), "# x en el checkout principal\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "x", "tasks.md"), "- [ ] 1.1 a\n- [ ] 1.2 b\n")

	listing := getListing(t, svc)

	x := listing.summary(t, "x", "active")
	if x.TasksTotal != 1 || x.TasksCompleted != 1 {
		t.Errorf("tareas = %d/%d, se esperaban las del worktree (1/1)", x.TasksCompleted, x.TasksTotal)
	}
	wantSameDirectory(t, "worktreePath", x.WorktreePath, worktree)

	detail := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/increments/x", nil))
	var dto IncrementDetailDTO
	if err := json.Unmarshal(detail.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dto.TasksContent, "1.1 hecha") {
		t.Errorf("el detalle debía leerse del worktree y no del checkout principal: %q", dto.TasksContent)
	}
}

// REQ-25.34 escenario «El worktree de la rama de archivado prevalece sobre el de la rama del cambio».
func TestIncrementsArchiveWorktreePrevailsOverTheChangeWorktree(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktreeChange(t, svc, ws, "x")
	archiveWorktree := filepath.Join(t.TempDir(), "x-archive")
	workspacetest.WriteFile(t, filepath.Join(archiveWorktree, "openspec", "changes", "x", "tasks.md"), "- [x] 1.1 a\n- [x] 1.2 b\n")
	registerArchiveWorktree(t, ws, "x", archiveWorktree)

	listing := getListing(t, svc)

	x := listing.summary(t, "x", "active")
	if x.Branch != "feat/x-archive" {
		t.Errorf("branch = %q, se esperaba la de archivado feat/x-archive", x.Branch)
	}
	wantSameDirectory(t, "worktreePath", x.WorktreePath, archiveWorktree)
	if x.TasksTotal != 2 || x.TasksCompleted != 2 {
		t.Errorf("tareas = %d/%d, se esperaban las del worktree de archivado (2/2)", x.TasksCompleted, x.TasksTotal)
	}
}

// registerArchiveWorktree añade al registro local del cambio una rama de archivado desplegada en dir.
func registerArchiveWorktree(t *testing.T, ws workspacetest.Workspace, change, dir string) {
	t.Helper()
	common, err := gitcommon.CommonDirFromFiles(ws.Master)
	if err != nil {
		t.Fatalf("git common dir de %s: %v", ws.Master, err)
	}
	store := registry.New()
	records, err := store.List(common, branch.LaneSDD)
	if err != nil {
		t.Fatalf("leer el registro: %v", err)
	}
	for _, record := range records {
		if record.Change != change {
			continue
		}
		record.Repositories[0].Branches = append(record.Repositories[0].Branches,
			branch.RecordBranch{Name: "feat/" + change + "-archive", Kind: branch.KindArchive, Worktree: dir})
		if err := store.Save(common, record); err != nil {
			t.Fatalf("guardar el registro de %s: %v", change, err)
		}
		return
	}
	t.Fatalf("el registro no tiene el cambio %s: %+v", change, records)
}

// REQ-25.34 escenario «Un worktree desaparecido se omite con aviso»: 200 sin ese cambio y un aviso en
// `X-Axiom-Notes` que nombra la ruta ausente. La cabecera viaja en ASCII (los acentos, escapados).
func TestIncrementsOmitsAMissingWorktreeWithANote(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktree := worktreeChange(t, svc, ws, "x")
	if err := os.RemoveAll(worktree); err != nil {
		t.Fatal(err)
	}

	listing := getListing(t, svc)

	for _, increment := range listing.list {
		if increment.Name == "x" {
			t.Errorf("x vive en un worktree desaparecido y no debía listarse: %+v", increment)
		}
	}
	note := listing.noteMentioning(t, "ya no existe", "x")
	if !strings.Contains(note, filepath.Base(worktree)) {
		t.Errorf("el aviso debía nombrar la ruta ausente %s: %q", worktree, note)
	}
	for i := 0; i < len(listing.noteBytes); i++ {
		if listing.noteBytes[i] >= 0x80 {
			t.Fatalf("la cabecera debía viajar en ASCII (acentos escapados): %q", listing.noteBytes)
		}
	}
	if !strings.Contains(note, "restáuralo") {
		t.Errorf("el aviso decodificado debía conservar los acentos: %q", note)
	}
}

// REQ-25.34 y REQ-25.20 escenario «Un cambio fusionado y sin archivar se muestra como pendiente»: el
// listado marca `y` y la cabecera lo nombra con su continuación; la respuesta sigue siendo 200 y un
// cambio que vive solo en su worktree no se marca.
func TestIncrementsMarksAMergedUnarchivedChangeAsPendingAndNotesIt(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "y", "proposal.md"), "# y fusionado\n")
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktreeChange(t, svc, ws, "x")

	listing := getListing(t, svc)

	y := listing.summary(t, "y", "active")
	if !y.PendingArchive {
		t.Errorf("y está en la rama de integración sin archivar y debía marcarse pendingArchive: %+v", y)
	}
	if raw := listing.rawSummary(t, "y"); raw["pendingArchive"] != true {
		t.Errorf("pendingArchive debía viajar como true: %v", raw)
	}
	if x := listing.summary(t, "x", "active"); x.PendingArchive {
		t.Errorf("x solo vive en su worktree y no está pendiente: %+v", x)
	}
	listing.noteMentioning(t, "pendientes de archivar", "y", "axiom branch ensure --change y --archive")
	for _, note := range listing.notes {
		if strings.Contains(note, "--change x ") {
			t.Errorf("x no debía figurar entre los pendientes: %q", note)
		}
	}
}

// REQ-25.34 escenario «Sin cambios pendientes no hay aviso de pendientes»: `changes/` solo tiene
// `archive/` y los sumideros exentos, y la cabecera ni siquiera viaja.
func TestIncrementsWithoutPendingChangesHaveNoNote(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "archive", "2026-01-01-viejo", "proposal.md"), "# viejo\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "cumulative-core", "tasks.md"), "- [ ] 1.1 a\n")
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)

	listing := getListing(t, NewService(ws.Master))

	if listing.noteBytes != "" || len(listing.notes) != 0 {
		t.Errorf("sin pendientes no debía haber aviso, hay %q", listing.noteBytes)
	}
	for _, increment := range listing.list {
		if increment.PendingArchive {
			t.Errorf("ningún cambio está pendiente: %+v", increment)
		}
	}
}

// REQ-25.34 y D-17: sin la referencia de integración la lista de pendientes se omite con un aviso y sin
// error; la respuesta sigue siendo 200.
func TestIncrementsNoteTheMissingIntegrationReference(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), `workspace:
  name: ejemplo-embebido
  topology: monorepo-embedded
  specs_repository: "."
roles:
  core:
    name: Core
    repositories:
      - path: .
git:
  integration_branch: develop
`)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "y", "proposal.md"), "# y\n")
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)

	listing := getListing(t, NewService(ws.Master))

	listing.noteMentioning(t, "pendientes de archivar", "develop")
	if y := listing.summary(t, "y", "active"); y.PendingArchive {
		t.Errorf("sin referencia de integración no se evalúa nada y y no debía marcarse: %+v", y)
	}
}

// REQ-25.34 escenario «Una escritura sobre un cambio en worktree se dirige al worktree»: el relevo de
// `x` se escribe en la carpeta del cambio del worktree y el checkout principal no recibe nada.
func TestHandoffForAChangeInAWorktreeIsWrittenThere(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktree := worktreeChange(t, svc, ws, "x")
	payload, err := json.Marshal(CreateHandoffRequest{Change: "x", FromPhase: "design", ToPhase: "tasks", ExecutiveSummary: "Resumen de prueba"})
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/handoffs", bytes.NewReader(payload)))

	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /api/handoffs = %d: %s", rr.Code, rr.Body.String())
	}
	wantFile(t, filepath.Join(worktree, "openspec", "changes", "x", "handoff.md"))
	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if status := runGit(t, ws.Master, "status", "--porcelain"); status != "" {
		t.Errorf("el checkout principal no debía recibir nada:\n%s", status)
	}
}

// Arnés de S10-c3: `GET /api/increments` con un cambio solo en su worktree y otro pendiente de
// archivar, de una sola vez, como lo recibe la SPA.
func TestIncrementsWorktreeAndPendingHarness(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "y", "proposal.md"), "# y fusionado\n")
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	worktree := worktreeChange(t, svc, ws, "x")

	listing := getListing(t, svc)

	x, y := listing.summary(t, "x", "active"), listing.summary(t, "y", "active")
	wantSameDirectory(t, "worktreePath de x", x.WorktreePath, worktree)
	if y.WorktreePath != "" || y.Branch != "" || !y.PendingArchive {
		t.Errorf("y vive en el checkout principal y está pendiente: %+v", y)
	}
	if len(listing.notes) == 0 {
		t.Error("el pendiente debía avisarse en X-Axiom-Notes")
	}
}
