package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Alta de un cambio desde un work item (D-08, prov:WEB:9): `POST /api/integrations/start` y
// `.../start/discard`. Usan git real con `ensure` de verdad y un doble de IntegrationsGateway para las
// fases 1 y 2: nunca llaman a un tracker. Ninguna llama a t.Parallel (estado de proceso).

const (
	wiChange = "us-12-alta"
	wiBranch = "feat/us-12-alta"
	wiHostil = "TEXTO-HOSTIL-DEL-WORK-ITEM"
)

type wiEnv struct {
	ws  workspacetest.Workspace
	svc *Service
	srv *Server
	h   http.Handler
	gw  *fakeGateway
}

func newWIEnv(t *testing.T, mode string) *wiEnv {
	t.Helper()
	// El estado (hub con las preferencias de worktree) se aísla antes de preparar el proyecto.
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	ws := embeddedProject(t, mode)
	t.Setenv(apiEnvVar, apiSentinel)
	integrationstest.Install(t, integrationstest.NewFake())
	svc := NewService(ws.Master)
	gw := &fakeGateway{applic: cli.WorkItemApplicability{Applicable: true, Plugin: "fake", RefAlias: "fk"}}
	svc.SetIntegrationsGateway(gw)
	srv := NewServer(svc)
	return &wiEnv{ws: ws, svc: svc, srv: srv, h: srv.handler(), gw: gw}
}

// api reutiliza el cliente de las pruebas de integraciones sobre la cadena completa.
func (e *wiEnv) api() *apiEnv { return &apiEnv{ws: e.ws, svc: e.svc, srv: e.srv, h: e.h, gw: e.gw} }

func (e *wiEnv) start(body string) apiResp { return e.api().post("/api/integrations/start", body) }
func (e *wiEnv) discard(body string) apiResp {
	return e.api().post("/api/integrations/start/discard", body)
}

// untouched exige que no haya quedado ni rama ni carpeta del cambio.
func (e *wiEnv) untouched(t *testing.T) {
	t.Helper()
	if branchExists(t, e.ws.Master, wiBranch) {
		t.Errorf("no debía crearse la rama %s", wiBranch)
	}
	wantAbsent(t, filepath.Join(e.ws.Home, "changes", wiChange))
}

func actionsOf(t *testing.T, body map[string]any) []string {
	t.Helper()
	raw, _ := body["actions"].([]any)
	out := make([]string, 0, len(raw))
	for _, a := range raw {
		out = append(out, a.(string))
	}
	return out
}

func wantStatus(t *testing.T, r apiResp, status int, code string) map[string]any {
	t.Helper()
	if r.Code != status {
		t.Fatalf("estado = %d, se esperaba %d: %s", r.Code, status, r.body())
	}
	body := r.json(t)
	if code != "" && body["code"] != code {
		t.Fatalf("code = %v, se esperaba %s: %s", body["code"], code, r.body())
	}
	assertNoStore(t, "respuesta", r)
	return body
}

func TestWorkItemStartAdoptsInTheEnsureWorktree(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{Change: wiChange, ExternalRef: "fk:12", Status: "staged", Warnings: []string{"aviso de la fase 1"}}, nil
	}
	e.gw.adopt = func(context.Context, string, string) (cli.WorkItemAdoptOutcome, error) {
		return cli.WorkItemAdoptOutcome{Change: wiChange, Status: "adopted", Warnings: []string{"el enlace de vuelta quedó pendiente"}}, nil
	}

	r := e.start(`{"ref":" 12 ","intent":"Dar de alta usuarios"}`)

	body := wantStatus(t, r, http.StatusOK, "")
	if body["success"] != true || body["name"] != wiChange || body["externalRef"] != "fk:12" {
		t.Errorf("respuesta = %v", body)
	}
	warnings, _ := body["warnings"].([]any)
	if len(warnings) != 2 {
		t.Errorf("warnings = %v, se esperaban los de las dos fases", body["warnings"])
	}
	// La fase 1 recibe el proyecto; la fase 2, la raíz del worktree que preparó ensure.
	wantSameDirectory(t, "cwd de la fase 1", e.gw.startCwd.Load().(string), e.ws.Master)
	worktree := worktreeOfBranch(t, e.ws.Master, wiBranch)
	wantSameDirectory(t, "cwd de la fase 2", e.gw.adoptCwd.Load().(string), worktree)
	proposal := wantFile(t, filepath.Join(worktree, "openspec", "changes", wiChange, "proposal.md"))
	if !strings.Contains(proposal, "Dar de alta usuarios") {
		t.Errorf("proposal.md sin la intención tecleada:\n%s", proposal)
	}
	// El dashboard nunca sella el kickoff (H-20) ni escribe en el checkout principal.
	wantAbsent(t, filepath.Join(worktree, "openspec", "changes", wiChange, "kickoff.yaml"))
	wantAbsent(t, filepath.Join(e.ws.Home, "changes", wiChange))
	if got := runGit(t, e.ws.Master, "branch", "--show-current"); got != "main" {
		t.Errorf("el checkout principal debía seguir en main, está en %q", got)
	}
}

func TestWorkItemStartProposalNeverCopiesTrackerText(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{Change: wiChange, ExternalRef: "fk:12", Status: "staged", Warnings: []string{wiHostil}}, nil
	}

	r := e.start(`{"ref":"12"}`)

	wantStatus(t, r, http.StatusOK, "")
	proposal := wantFile(t, filepath.Join(worktreeOfBranch(t, e.ws.Master, wiBranch), "openspec", "changes", wiChange, "proposal.md"))
	if strings.Contains(proposal, wiHostil) {
		t.Errorf("proposal.md no puede llevar texto del work item:\n%s", proposal)
	}
	// Sin intención tecleada, el texto genérico cita la referencia externa.
	if !strings.Contains(proposal, "fk:12") {
		t.Errorf("el texto genérico debe citar la referencia externa fk:12:\n%s", proposal)
	}
}

func TestWorkItemStartRejectsInvalidChangeName(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	absolute := filepath.Join(e.ws.Master, "x")
	for _, change := range []string{"../x", "a/b", `a\b`, absolute, "archive", ".oculto", "con espacio"} {
		payload, _ := json.Marshal(map[string]string{"change": change})
		r := e.start(string(payload))
		body := wantStatus(t, r, http.StatusBadRequest, "invalid_request")
		if strings.TrimSpace(body["error"].(string)) == "" {
			t.Errorf("%q: sin mensaje", change)
		}
		d := e.discard(`{"change":` + string(mustJSON(change)) + `,"confirm":true}`)
		wantStatus(t, d, http.StatusBadRequest, "invalid_request")
	}
	// Un nombre derivado inválido de la fase 1 tampoco llega a una ruta.
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{Change: "../x", ExternalRef: "fk:12", Status: "staged"}, nil
	}
	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusUnprocessableEntity, "invalid_change_name")
	if e.gw.stageds.Load() != 0 || e.gw.adopts.Load() != 0 || e.gw.discards.Load() != 0 {
		t.Errorf("un nombre inválido no debe llegar a la pasarela: staged %d adopt %d discard %d", e.gw.stageds.Load(), e.gw.adopts.Load(), e.gw.discards.Load())
	}
	e.untouched(t)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestWorkItemStartValidatesTheRequest(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	bigIntent := strings.Repeat("a", 4097)
	cases := map[string]string{
		"sin ref ni change":   `{}`,
		"ref y change":        `{"ref":"12","change":"us-12-alta"}`,
		"ref en blanco":       `{"ref":"   "}`,
		"ref de control":      `{"ref":"12\n13"}`,
		"ref demasiado larga": `{"ref":"` + strings.Repeat("1", 513) + `"}`,
		"intención grande":    `{"ref":"12","intent":"` + bigIntent + `"}`,
		"tipo desconocido":    `{"ref":"12","type":"cualquiera"}`,
		"campo desconocido":   `{"ref":"12","pat":"x"}`,
		"cuerpo sobre 16 KiB": `{"ref":"12","intent":"` + strings.Repeat("a", 17<<10) + `"}`,
		"JSON con basura":     `{"ref":"12"} {}`,
	}
	for _, body := range cases {
		wantStatus(t, e.start(body), http.StatusBadRequest, "invalid_request")
	}
	if e.gw.starts.Load() != 0 {
		t.Errorf("una petición inválida no debe llegar a la fase 1: %d llamadas", e.gw.starts.Load())
	}
	api := e.api()
	if r := api.request(http.MethodGet, "/api/integrations/start", ""); r.Code != http.StatusMethodNotAllowed || r.code(t) != "method_not_allowed" {
		t.Errorf("GET = %d %s", r.Code, r.body())
	}
	if r := api.post("/api/integrations/start?ref=12", `{"ref":"12"}`); r.Code != http.StatusBadRequest {
		t.Errorf("query = %d", r.Code)
	}
	if r := api.request(http.MethodPost, "/api/integrations/start", `{"ref":"12"}`, func(req *http.Request) { req.Header.Set("Content-Type", "text/plain") }); r.Code != http.StatusBadRequest {
		t.Errorf("tipo de contenido = %d", r.Code)
	}
	e.untouched(t)
}

func TestWorkItemStartPhase1RefusalLeavesNoTrace(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{}, &cli.WorkItemRefusal{Code: "work_item_closed", Message: "El work item está cerrado.", Continuation: "Reabre el work item", Fields: map[string]string{"reason": "closed"}}
	}

	r := e.start(`{"ref":"12"}`)

	body := wantStatus(t, r, http.StatusUnprocessableEntity, "work_item_closed")
	if body["error"] != "El work item está cerrado." || body["continuation"] != "Reabre el work item" || body["stage"] != "start" {
		t.Errorf("cuerpo = %v", body)
	}
	if _, has := body["actions"]; has {
		t.Errorf("tras un rechazo de la fase 1 no hay nada que reintentar ni descartar: %v", body)
	}
	if e.gw.adopts.Load() != 0 {
		t.Error("no debe adoptarse tras un rechazo de la fase 1")
	}
	e.untouched(t)
}

func TestWorkItemStartRefusalFromAnotherChangeIsAConflict(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{}, &cli.WorkItemRefusal{Code: "work_item_linked", Message: "Ya está enlazado.", Continuation: "Cierra el otro cambio"}
	}
	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusUnprocessableEntity, "work_item_linked")
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{}, &cli.WorkItemRefusal{Code: "change_start_busy", Message: "Otro proceso.", Continuation: "Reintenta"}
	}
	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusConflict, "change_start_busy")
}

func TestWorkItemStartDecisionRequiredCarriesChangeAndMode(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeAsk)

	first := e.start(`{"ref":"12"}`)

	body := wantStatus(t, first, http.StatusConflict, "decision_required")
	if body["change"] != wiChange || body["decision"] == nil || body["status"] != "decision_required" {
		t.Fatalf("la decisión debe llevar el cambio y la decisión: %v", body)
	}
	e.untouched(t)
	if e.gw.starts.Load() != 1 || e.gw.adopts.Load() != 0 {
		t.Fatalf("fase 1 %d veces y fase 2 %d", e.gw.starts.Load(), e.gw.adopts.Load())
	}

	// La repetición con {change, mode} no repite la fase 1 y completa el alta.
	second := e.start(`{"change":"` + wiChange + `","mode":"worktree"}`)

	wantStatus(t, second, http.StatusOK, "")
	if e.gw.starts.Load() != 1 || e.gw.stageds.Load() != 1 || e.gw.adopts.Load() != 1 {
		t.Errorf("fase 1 %d, staging %d, fase 2 %d", e.gw.starts.Load(), e.gw.stageds.Load(), e.gw.adopts.Load())
	}
	wantFile(t, filepath.Join(worktreeOfBranch(t, e.ws.Master, wiBranch), "openspec", "changes", wiChange, "proposal.md"))
}

func TestWorkItemStartRetryWithoutStagingCreatesNothing(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.staged = func(string, string) (cli.WorkItemStagedOutcome, error) {
		return cli.WorkItemStagedOutcome{}, &cli.WorkItemRefusal{Code: "staging_not_found", Message: "No hay staging.", Continuation: "Vuelve a dar el work item"}
	}

	r := e.start(`{"change":"` + wiChange + `"}`)

	wantStatus(t, r, http.StatusConflict, "staging_not_found")
	if e.gw.starts.Load() != 0 || e.gw.adopts.Load() != 0 {
		t.Error("sin staging no debe llamarse a la fase 1 ni a la 2")
	}
	e.untouched(t)
}

func TestWorkItemStartEnsureRefusalOffersRetryAndDiscard(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeInPlace)
	if err := os.WriteFile(filepath.Join(e.ws.Master, "stray.txt"), []byte("sin confirmar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := e.start(`{"ref":"12"}`)

	body := wantStatus(t, r, http.StatusConflict, "dirty_worktree")
	if body["change"] != wiChange || body["stage"] != "ensure" || strings.TrimSpace(body["continuation"].(string)) == "" {
		t.Errorf("cuerpo = %v", body)
	}
	if got := actionsOf(t, body); len(got) != 2 || got[0] != "retry" || got[1] != "discard" {
		t.Errorf("actions = %v", got)
	}
	if e.gw.adopts.Load() != 0 {
		t.Error("no debe adoptarse sin rama preparada")
	}
}

// Hueco 8: los rechazos de change_not_prepared llegan con su reason y su mensaje propio.
func TestWorkItemStartPhase2RefusalKeepsItsReasonAndMessage(t *testing.T) {
	for _, reason := range []string{"head_other_branch", "folder_missing", "lane_mismatch", "record_unreadable"} {
		t.Run(reason, func(t *testing.T) {
			e := newWIEnv(t, hub.WorktreeModeWorktree)
			message := "mensaje propio del motivo " + reason
			e.gw.adopt = func(context.Context, string, string) (cli.WorkItemAdoptOutcome, error) {
				return cli.WorkItemAdoptOutcome{}, &cli.WorkItemRefusal{
					Code: "change_not_prepared", Message: message, Continuation: "axiom branch ensure --change " + wiChange,
					Fields: map[string]string{"reason": reason},
				}
			}

			r := e.start(`{"ref":"12"}`)

			body := wantStatus(t, r, http.StatusConflict, "change_not_prepared")
			if body["reason"] != reason || body["error"] != message || body["change"] != wiChange || body["stage"] != "adopt" {
				t.Errorf("cuerpo = %v", body)
			}
			if got := actionsOf(t, body); len(got) != 2 || got[0] != "retry" || got[1] != "discard" {
				t.Errorf("actions = %v", got)
			}
			// El fallo posterior a la fase 1 deja el staging: el descarte nunca es automático.
			if e.gw.discards.Load() != 0 {
				t.Error("el descarte solo se ejecuta cuando la persona lo pide")
			}
		})
	}
}

func TestWorkItemStartPhase2StagingNotFoundOffersNoActions(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.adopt = func(context.Context, string, string) (cli.WorkItemAdoptOutcome, error) {
		return cli.WorkItemAdoptOutcome{}, &cli.WorkItemRefusal{Code: "staging_not_found", Message: "No hay staging.", Continuation: "Vuelve a dar el work item"}
	}

	body := wantStatus(t, e.start(`{"ref":"12"}`), http.StatusConflict, "staging_not_found")

	if got := actionsOf(t, body); len(got) != 0 {
		t.Errorf("sin staging no hay nada que reintentar ni descartar: %v", got)
	}
}

func TestWorkItemStartAdoptWarningsDoNotStopTheFlow(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.adopt = func(context.Context, string, string) (cli.WorkItemAdoptOutcome, error) {
		return cli.WorkItemAdoptOutcome{Change: wiChange, Status: "adopted", Warnings: []string{"backlink pendiente", "staging sin retirar"}}, nil
	}

	body := wantStatus(t, e.start(`{"ref":"12"}`), http.StatusOK, "")

	warnings, _ := body["warnings"].([]any)
	if len(warnings) != 2 || body["success"] != true {
		t.Errorf("los avisos de la fase 2 viajan con el éxito: %v", body)
	}
}

// Hueco 4: reanudar con el mismo work item reutiliza la carpeta que dejó el intento fallido.
func TestWorkItemStartResumeReusesTheFolderOfAFailedAttempt(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.adopt = func(context.Context, string, string) (cli.WorkItemAdoptOutcome, error) {
		return cli.WorkItemAdoptOutcome{}, &cli.WorkItemRefusal{Code: "write_failed", Message: "No se pudo adoptar.", Continuation: "Reintenta"}
	}
	wantStatus(t, e.start(`{"ref":"12","intent":"Primera intención"}`), http.StatusConflict, "write_failed")
	proposal := filepath.Join(worktreeOfBranch(t, e.ws.Master, wiBranch), "openspec", "changes", wiChange, "proposal.md")
	if err := os.WriteFile(proposal, []byte("# editada a mano\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Segundo intento: el mismo work item, ya con staging (la fase 1 dice already_staged).
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{Change: wiChange, ExternalRef: "fk:12", Status: "already_staged"}, nil
	}
	e.gw.adopt = nil

	r := e.start(`{"ref":"12","intent":"Segunda intención"}`)

	wantStatus(t, r, http.StatusOK, "")
	if got := wantFile(t, proposal); got != "# editada a mano\n" {
		t.Errorf("la reanudación no debe reescribir un proposal.md existente:\n%s", got)
	}
}

func TestWorkItemStartResumeByChangeReusesTheFolder(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.adopt = func(context.Context, string, string) (cli.WorkItemAdoptOutcome, error) {
		return cli.WorkItemAdoptOutcome{}, &cli.WorkItemRefusal{Code: "change_start_busy", Message: "Otro proceso.", Continuation: "Reintenta"}
	}
	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusConflict, "change_start_busy")
	e.gw.adopt = nil

	r := e.start(`{"change":"` + wiChange + `"}`)

	wantStatus(t, r, http.StatusOK, "")
	if e.gw.starts.Load() != 1 {
		t.Errorf("el reintento por change no repite la fase 1: %d", e.gw.starts.Load())
	}
}

// Hueco 4: un cambio activo AJENO con el mismo nombre derivado no se reutiliza en silencio.
func TestWorkItemStartDoesNotReuseAForeignActiveChange(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	foreign := filepath.Join(e.ws.Home, "changes", wiChange)
	workspacetest.WriteFile(t, filepath.Join(foreign, "proposal.md"), "# ajeno\n")
	runGit(t, e.ws.Master, "add", "-A")
	runGit(t, e.ws.Master, "commit", "-q", "-m", "cambio ajeno")

	r := e.start(`{"ref":"12"}`)

	body := wantStatus(t, r, http.StatusConflict, "change_exists")
	if body["change"] != wiChange || strings.TrimSpace(body["error"].(string)) == "" {
		t.Errorf("cuerpo = %v", body)
	}
	if got := wantFile(t, filepath.Join(foreign, "proposal.md")); got != "# ajeno\n" {
		t.Errorf("el cambio ajeno no debe tocarse:\n%s", got)
	}
	if branchExists(t, e.ws.Master, wiBranch) {
		t.Error("no debía crearse la rama sobre un cambio ajeno")
	}
	// El staging recién creado por esta petición se retira: un «Reintentar» no puede reutilizar lo ajeno.
	if e.gw.discards.Load() != 1 || e.gw.adopts.Load() != 0 {
		t.Errorf("descartes %d, fase 2 %d", e.gw.discards.Load(), e.gw.adopts.Load())
	}
}

func TestWorkItemStartRefusesAnArchivedChangeWithTheSameName(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	workspacetest.WriteFile(t, filepath.Join(e.ws.Home, "changes", "archive", "2026-01-01-"+wiChange, "proposal.md"), "# archivado\n")
	runGit(t, e.ws.Master, "add", "-A")
	runGit(t, e.ws.Master, "commit", "-q", "-m", "archivado")

	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusConflict, "change_exists")

	if branchExists(t, e.ws.Master, wiBranch) {
		t.Error("no debía crearse la rama")
	}
}

// Hueco 5: sin plugin de work items aplicable o sin PAT, un código tipado y nada creado.
func TestWorkItemStartWithoutApplicablePluginCreatesNothing(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.applic = cli.WorkItemApplicability{Reason: "no hay ningún plugin de integración activo"}

	body := wantStatus(t, e.start(`{"ref":"12"}`), http.StatusUnprocessableEntity, "work_items_not_applicable")

	if !strings.Contains(body["error"].(string), "no hay ningún plugin de integración activo") {
		t.Errorf("el mensaje debe explicar el motivo: %v", body)
	}
	if e.gw.starts.Load() != 0 {
		t.Error("sin plugin aplicable no se llama a la fase 1")
	}
	e.untouched(t)
	wantStatus(t, e.start(`{"change":"`+wiChange+`"}`), http.StatusUnprocessableEntity, "work_items_not_applicable")
	wantStatus(t, e.discard(`{"change":"`+wiChange+`","confirm":true}`), http.StatusUnprocessableEntity, "work_items_not_applicable")
	if e.gw.stageds.Load() != 0 || e.gw.discards.Load() != 0 {
		t.Error("sin plugin aplicable no se llama a la pasarela")
	}
}

func TestWorkItemStartWithoutPATCreatesNothing(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	t.Setenv(apiEnvVar, "")

	body := wantStatus(t, e.start(`{"ref":"12"}`), http.StatusUnprocessableEntity, "secret_missing")

	if body["envVar"] != apiEnvVar {
		t.Errorf("envVar = %v", body["envVar"])
	}
	if e.gw.starts.Load() != 0 {
		t.Error("sin PAT no se llama al tracker")
	}
	e.untouched(t)
}

func TestWorkItemStartWithoutGatewayAnswers503(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.svc.SetIntegrationsGateway(nil)

	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusServiceUnavailable, "integrations_gateway_missing")
	wantStatus(t, e.discard(`{"change":"`+wiChange+`","confirm":true}`), http.StatusServiceUnavailable, "integrations_gateway_missing")
	e.untouched(t)
}

func TestWorkItemStartOverlappingRequestsAnswer409(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	entered, release := make(chan struct{}), make(chan struct{})
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		close(entered)
		<-release
		return cli.WorkItemStartOutcome{Change: wiChange, ExternalRef: "fk:12", Status: "staged"}, nil
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var first apiResp
	go func() {
		defer wg.Done()
		first = e.start(`{"ref":"12"}`)
	}()
	<-entered

	wantStatus(t, e.start(`{"ref":"13"}`), http.StatusConflict, "integration_busy")
	wantStatus(t, e.discard(`{"change":"`+wiChange+`","confirm":true}`), http.StatusConflict, "integration_busy")
	close(release)
	wg.Wait()

	wantStatus(t, first, http.StatusOK, "")
	if e.gw.starts.Load() != 1 {
		t.Errorf("la petición solapada no debe llegar a la fase 1: %d", e.gw.starts.Load())
	}
}

func TestWorkItemStartTimeoutAnswers504AndReleasesTheLock(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.srv.workItemBudget = 40 * time.Millisecond
	e.gw.startWI = func(ctx context.Context, _, _ string) (cli.WorkItemStartOutcome, error) {
		<-ctx.Done()
		return cli.WorkItemStartOutcome{}, ctx.Err()
	}
	started := time.Now()

	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusGatewayTimeout, "integration_timeout")

	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Errorf("tardó %v", elapsed)
	}
	e.untouched(t)
	e.gw.startWI = nil
	e.srv.workItemBudget = defaultWorkItemBudget
	wantStatus(t, e.start(`{"ref":"12"}`), http.StatusOK, "")
}

func TestWorkItemStartGatewayErrorsAreRedacted(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.startWI = func(context.Context, string, string) (cli.WorkItemStartOutcome, error) {
		return cli.WorkItemStartOutcome{}, errors.New("el tracker devolvió " + apiSentinel)
	}

	r := e.start(`{"ref":"12"}`)

	wantStatus(t, r, http.StatusBadGateway, "integration_failed")
	assertNoSentinel(t, "alta", r)
	e.untouched(t)
}

func TestWorkItemDiscardRequiresConfirmAndOnlyDiscardsTheStaging(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)

	for _, body := range []string{`{"change":"` + wiChange + `"}`, `{"change":"` + wiChange + `","confirm":false}`} {
		wantStatus(t, e.discard(body), http.StatusBadRequest, "invalid_request")
	}
	if e.gw.discards.Load() != 0 {
		t.Fatal("sin confirm no se descarta")
	}

	r := e.discard(`{"change":"` + wiChange + `","confirm":true}`)

	body := wantStatus(t, r, http.StatusOK, "")
	if body["change"] != wiChange || body["status"] != "discarded" {
		t.Errorf("respuesta = %v", body)
	}
	if _, ok := body["warnings"].([]any); !ok {
		t.Errorf("warnings debe ser una lista: %v", body)
	}
	e.untouched(t)
	for name, payload := range map[string]string{
		"campo desconocido": `{"change":"` + wiChange + `","confirm":true,"force":true}`,
		"sin change":        `{"confirm":true}`,
	} {
		if r := e.discard(payload); r.Code != http.StatusBadRequest {
			t.Errorf("%s: estado %d", name, r.Code)
		}
	}
	if r := e.api().request(http.MethodGet, "/api/integrations/start/discard", ""); r.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET = %d", r.Code)
	}
}

func TestWorkItemDiscardRefusalIsTyped(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	e.gw.discard = func(string, string) (cli.WorkItemDiscardOutcome, error) {
		return cli.WorkItemDiscardOutcome{}, &cli.WorkItemRefusal{Code: "change_start_busy", Message: "Otro proceso.", Continuation: "Reintenta"}
	}

	body := wantStatus(t, e.discard(`{"change":"`+wiChange+`","confirm":true}`), http.StatusConflict, "change_start_busy")

	if body["error"] != "Otro proceso." || body["continuation"] != "Reintenta" {
		t.Errorf("cuerpo = %v", body)
	}
}

// Hueco 7: el paso 3 respeta el contexto de la petición: con un plazo vencido no crea nada.
func TestEnsureIncrementBranchHonoursTheRequestContext(t *testing.T) {
	e := newWIEnv(t, hub.WorktreeModeWorktree)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.svc.ensureIncrementBranch(ctx, wiChange, "")

	if err == nil {
		t.Fatal("con el contexto cancelado ensure no debía completarse")
	}
	e.untouched(t)
}
