package dashboard

import (
	"context"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Sin t.Parallel: comparten el HOME de pruebas, repos git y servidores httptest.

// onMain deja el repositorio de specs en la rama `main` que espera la publicación (sin commits aún).
func (e *marksEnv) onMain() {
	e.t.Helper()
	runGit(e.t, e.ws.Specs, "symbolic-ref", "HEAD", "refs/heads/main")
}

// publishToRemote confirma todo el repositorio de specs y lo sube a un remoto desnudo local, de
// modo que refs/remotes/origin/main refleja lo publicado.
func (e *marksEnv) publishToRemote() {
	e.t.Helper()
	e.onMain()
	bare := filepath.Join(e.t.TempDir(), "remote.git")
	runGit(e.t, e.ws.Master, "init", "-q", "--bare", bare)
	runGit(e.t, e.ws.Specs, "remote", "add", "origin", bare)
	runGit(e.t, e.ws.Specs, "add", "--all")
	runGit(e.t, e.ws.Specs, "-c", "user.name=T", "-c", "user.email=t@example.org", "commit", "-q", "-m", "docs: base")
	runGit(e.t, e.ws.Specs, "push", "-q", "origin", "main")
}

func publishBody(change string) map[string]any {
	return map[string]any{"change": change, "confirm": true, "branch": "main"}
}

func TestMarksRecordAppendsOnlyAndReturnsTheMatrix(t *testing.T) {
	e := newMarksEnv(t)
	resp, body := e.mark("c", "P-01", "dev", "ok", "primera")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("estado = %d: %v", resp.StatusCode, body)
	}
	first := e.marksFile()
	if first == "" {
		t.Fatal("la marca no se escribió")
	}
	entry, _ := body["entry"].(map[string]any)
	if entry["by"] != "Ana Pérez <ana@example.org>" || entry["case"] != "P-01" || entry["plan_digest"] == "" {
		t.Errorf("entrada = %v", entry)
	}
	if resp, body = e.mark("c", "P-02", "dev", "ko", "segunda"); resp.StatusCode != http.StatusCreated {
		t.Fatalf("estado = %d: %v", resp.StatusCode, body)
	}
	second := e.marksFile()
	if !strings.HasPrefix(second, first) || second == first {
		t.Errorf("el contenido previo debe ser prefijo exacto:\n%s\n---\n%s", first, second)
	}
	dto := e.detailMarks()
	if !dto.Available || len(dto.Cells) != 3 || dto.NoteMax != 500 || dto.CommitMax != 100 {
		t.Fatalf("DTO = %+v", dto)
	}
	if dto.Cells[0].State != "ok" || dto.Cells[0].Current == nil || dto.Cells[2].State != "ko" {
		t.Errorf("celdas = %+v", dto.Cells)
	}
	if dto.SummaryText != "dev: 1/2 ok, 1 ko; pre: 0/1 ok, 1 sin marcar" {
		t.Errorf("summaryText = %q", dto.SummaryText)
	}
}

func TestMarksValidationRejectionsWriteNothing(t *testing.T) {
	e := newMarksEnv(t)
	long := strings.Repeat("x", 501)
	cases := []struct {
		label, code, caseID, env, result, note, commit string
	}{
		{"caso desconocido", "marks_case_unknown", "P-99", "dev", "ok", "", ""},
		{"entorno no declarado", "marks_environment_invalid", "P-01", "prod", "ok", "", ""},
		{"entorno no aplicable", "marks_environment_invalid", "P-02", "pre", "ok", "", ""},
		{"resultado fuera de dominio", "marks_result_invalid", "P-01", "dev", "yes", "", ""},
		{"nota demasiado larga", "marks_note_too_long", "P-01", "dev", "ok", long, ""},
		{"nota con salto de línea", "marks_note_invalid", "P-01", "dev", "ok", "a\nb", ""},
		{"commit con caracteres no permitidos", "marks_commit_invalid", "P-01", "dev", "ok", "", "a b;rm"},
	}
	for _, c := range cases {
		resp, body := e.post("/api/increments/marks", map[string]any{"change": "c", "case": c.caseID,
			"environment": c.env, "result": c.result, "note": c.note, "commit": c.commit})
		wantMarksRefusal(t, c.label, resp, body, http.StatusUnprocessableEntity, c.code)
		if got := e.marksFile(); got != "" {
			t.Fatalf("%s: se escribió algo: %q", c.label, got)
		}
	}
	if identity, _ := e.fake.calls(); identity != 0 {
		t.Errorf("la identidad se consultó %d veces antes de validar", identity)
	}
}

func TestMarksHostileNoteTravelsLiteral(t *testing.T) {
	e := newMarksEnv(t)
	note := `a: b # "c" - <script>alert(1)</script> ñ`
	if resp, body := e.mark("c", "P-01", "dev", "ok", note); resp.StatusCode != http.StatusCreated {
		t.Fatalf("estado = %d: %v", resp.StatusCode, body)
	}
	dto := e.detailMarks()
	if dto.Cells[0].Current == nil || dto.Cells[0].Current.Note != note {
		t.Errorf("la nota debe viajar literal: %+v", dto.Cells[0].Current)
	}
	file, err := marks.Parse([]byte(e.marksFile()))
	if err != nil || len(file.Entries) != 1 || file.Entries[0].Note != note {
		t.Errorf("ida y vuelta: %+v %v", file, err)
	}
}

func TestMarksIdentityMissingNamesKeysAndCommands(t *testing.T) {
	e := newMarksEnv(t)
	e.fake.actor, e.fake.missing = "", []string{"user.email"}
	resp, body := e.mark("c", "P-01", "dev", "ok", "")
	wantMarksRefusal(t, "sin identidad", resp, body, http.StatusConflict, "identity_missing")
	text, _ := body["continuation"].(string)
	if !strings.Contains(text, "git -C") || !strings.Contains(text, "config user.email") || strings.Contains(text, "config user.name") {
		t.Errorf("continuación = %q", text)
	}
	if e.marksFile() != "" {
		t.Error("no debe escribirse nada sin identidad")
	}
}

func TestMarksConcurrentRecordsStayIntact(t *testing.T) {
	e := newMarksEnv(t)
	const n = 12
	var wg sync.WaitGroup
	statuses := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, _ := e.mark("c", "P-01", "dev", "ok", "nota "+string(rune('a'+i)))
			statuses[i] = resp.StatusCode
		}(i)
	}
	wg.Wait()
	for i, status := range statuses {
		if status != http.StatusCreated {
			t.Fatalf("la marca %d dio %d", i, status)
		}
	}
	file, err := marks.Parse([]byte(e.marksFile()))
	if err != nil || len(file.Entries) != n || len(file.Invalid) != 0 {
		t.Fatalf("fichero: %d entradas, %d inválidas, %v", len(file.Entries), len(file.Invalid), err)
	}
	seen := map[string]bool{}
	for _, entry := range file.Entries {
		if seen[entry.At] {
			t.Errorf("`at` repetido: %s", entry.At)
		}
		seen[entry.At] = true
	}
}

func TestMarksBusyWhilePublishing(t *testing.T) {
	e := newMarksEnv(t)
	e.onMain()
	if resp, body := e.mark("c", "P-01", "dev", "ok", ""); resp.StatusCode != http.StatusCreated {
		t.Fatalf("estado = %d: %v", resp.StatusCode, body)
	}
	e.fake.enter, e.fake.release = make(chan struct{}), make(chan struct{})
	done := make(chan struct{})
	var first *http.Response
	var firstBody map[string]any
	go func() {
		defer close(done)
		first, firstBody = e.post("/api/increments/marks/publish", publishBody("c"))
	}()
	<-e.fake.enter
	resp, body := e.mark("c", "P-02", "dev", "ok", "")
	wantMarksRefusal(t, "marca durante la publicación", resp, body, http.StatusConflict, "marks_busy")
	resp, body = e.post("/api/increments/marks/publish", publishBody("c"))
	wantMarksRefusal(t, "segunda publicación", resp, body, http.StatusConflict, "marks_busy")
	close(e.fake.release)
	<-done
	if first.StatusCode != http.StatusOK || firstBody["status"] != "published" {
		t.Fatalf("la primera publicación: %d %v", first.StatusCode, firstBody)
	}
	if _, publish := e.fake.calls(); publish != 1 {
		t.Errorf("el publicador se llamó %d veces", publish)
	}
	if resp, body = e.mark("c", "P-02", "dev", "ok", ""); resp.StatusCode != http.StatusCreated {
		t.Errorf("tras publicar el cerrojo debe liberarse: %d %v", resp.StatusCode, body)
	}
}

func TestMarksPublishWithoutConfirmTouchesNothing(t *testing.T) {
	e := newMarksEnv(t)
	for _, body := range []map[string]any{
		{"change": "c", "branch": "main"},
		{"change": "c", "confirm": false, "branch": "main"},
		{"change": "no-existe", "branch": "main"},
	} {
		resp, out := e.post("/api/increments/marks/publish", body)
		wantMarksRefusal(t, "sin confirm", resp, out, http.StatusUnprocessableEntity, "marks_confirm_missing")
	}
	if identity, publish := e.fake.calls(); identity != 0 || publish != 0 {
		t.Errorf("el puerto se usó: identidad %d, publicación %d", identity, publish)
	}
}

func TestMarksNothingToPublishSkipsThePublisher(t *testing.T) {
	e := newMarksEnv(t)
	resp, body := e.post("/api/increments/marks/publish", publishBody("c"))
	if resp.StatusCode != http.StatusOK || body["status"] != "nothing_to_publish" {
		t.Fatalf("sin fichero: %d %v", resp.StatusCode, body)
	}
	if r, b := e.mark("c", "P-01", "dev", "ok", ""); r.StatusCode != http.StatusCreated {
		t.Fatalf("marca: %d %v", r.StatusCode, b)
	}
	e.publishToRemote()
	if dto := e.detailMarks(); len(dto.Pending) != 0 || dto.PendingUnavailable != nil || dto.Branch != "main" || dto.Remote != "origin" {
		t.Fatalf("tras publicar: %+v", dto)
	}
	resp, body = e.post("/api/increments/marks/publish", publishBody("c"))
	if resp.StatusCode != http.StatusOK || body["status"] != "nothing_to_publish" {
		t.Fatalf("publicado: %d %v", resp.StatusCode, body)
	}
	if _, publish := e.fake.calls(); publish != 0 {
		t.Errorf("el publicador se llamó %d veces sin pendientes", publish)
	}
}

func TestMarksPublishDelegatesWithTheBranchTheUserSaw(t *testing.T) {
	e := newMarksEnv(t)
	e.publishToRemote()
	if r, b := e.mark("c", "P-01", "dev", "ko", "falla"); r.StatusCode != http.StatusCreated {
		t.Fatalf("marca: %d %v", r.StatusCode, b)
	}
	dto := e.detailMarks()
	if len(dto.Pending) != 1 || dto.Pending[0].Result != "ko" || dto.Branch != "main" {
		t.Fatalf("pendientes = %+v", dto)
	}
	e.fake.result = cli.MarksPublishResult{Status: "published", Remote: "origin", Branch: "main", Commit: "abc1234",
		PushedSHA: "def5678", Replayed: true, UncommittedDocs: []string{"changes/c/design.md"}}
	resp, body := e.post("/api/increments/marks/publish", publishBody("c"))
	if resp.StatusCode != http.StatusOK || body["status"] != "published" || body["pushed"] != "def5678" ||
		body["replayed"] != true || body["branch"] != "main" || body["remote"] != "origin" {
		t.Fatalf("respuesta = %d %v", resp.StatusCode, body)
	}
	if docs, _ := body["uncommittedDocs"].([]any); len(docs) != 1 {
		t.Errorf("uncommittedDocs = %v", body["uncommittedDocs"])
	}
	if _, ok := body["marks"].(map[string]any); !ok {
		t.Errorf("falta marks: %v", body)
	}
	e.fake.mu.Lock()
	got := e.fake.lastRequest
	e.fake.mu.Unlock()
	if got.Change != "c" || got.Branch != "main" || got.CWD != e.ws.Specs {
		t.Errorf("petición al publicador = %+v", got)
	}
}

func TestMarksPublishRefusalsKeepCodeAndConcreteContinuation(t *testing.T) {
	e := newMarksEnv(t)
	e.publishToRemote()
	if r, b := e.mark("c", "P-01", "dev", "ok", ""); r.StatusCode != http.StatusCreated {
		t.Fatalf("marca: %d %v", r.StatusCode, b)
	}
	continuation := "ejecuta `git -C \"x\" log origin/main..HEAD --stat` y sube esos commits antes de repetir"
	e.fake.result = cli.MarksPublishResult{Status: "refused", Refusal: &cli.MarksPublishRefusal{
		Code: "other_commits_unpublished", Label: "hay otros commits sin subir", Message: "hay commits ajenos",
		Continuation: continuation, Details: map[string]any{"commits": []string{"abc"}}}}
	resp, body := e.post("/api/increments/marks/publish", publishBody("c"))
	wantMarksRefusal(t, "otros commits", resp, body, http.StatusConflict, "other_commits_unpublished")
	if body["status"] != "refused" || body["label"] != "hay otros commits sin subir" || body["continuation"] != continuation || body["details"] == nil {
		t.Errorf("cuerpo = %v", body)
	}
}

func TestMarksPendingUnavailableCarriesTheTypedRefusal(t *testing.T) {
	e := newMarksEnv(t) // el repo está en master y la publicación espera main
	if r, b := e.mark("c", "P-01", "dev", "ok", ""); r.StatusCode != http.StatusCreated {
		t.Fatalf("marca: %d %v", r.StatusCode, b)
	}
	dto := e.detailMarks()
	un := dto.PendingUnavailable
	if un == nil || un.Code != "wrong_branch" || un.Error == "" || un.Continuation == "" || dto.Remote != "origin" || len(dto.Pending) != 0 {
		t.Fatalf("pendingUnavailable = %+v (DTO %+v)", un, dto)
	}
	if dto.SummaryText == "" {
		t.Error("el resumen debe mantenerse aunque no se puedan medir las pendientes")
	}
	resp, body := e.post("/api/increments/marks/publish", publishBody("c"))
	wantMarksRefusal(t, "pendientes desconocidas", resp, body, http.StatusConflict, "wrong_branch")
	if _, publish := e.fake.calls(); publish != 0 {
		t.Errorf("el publicador se llamó %d veces", publish)
	}
}

func TestMarksDetailSummaryEqualsTheSharedStatusReading(t *testing.T) {
	e := newMarksEnv(t)
	e.onMain()
	for _, m := range [][3]string{{"P-01", "dev", "ok"}, {"P-01", "pre", "blocked"}, {"P-02", "dev", "n/a"}} {
		if r, b := e.mark("c", m[0], m[1], m[2], ""); r.StatusCode != http.StatusCreated {
			t.Fatalf("marca: %d %v", r.StatusCode, b)
		}
	}
	// Una marca caducada: cambia pruebas.md después de marcar.
	workspacetest.WriteFile(t, filepath.Join(e.root, "pruebas.md"),
		atHeader+marksRow("P-01", "dev; pre")+marksRow("P-02", "dev")+marksRow("P-03", "pre"))
	loc, err := workspace.Locate(e.ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	want := sddstatus.AcceptanceMarksFor(context.Background(), loc, e.root, "c", sddstatus.AcceptanceTestsFor(loc, e.root))
	dto := e.detailMarks()
	if want.Error != "" || dto.SummaryText != want.Summary.Text() || dto.Orphans != want.Summary.Orphans {
		t.Errorf("DTO %q (huérfanas %d), status %q (huérfanas %d, error %q)", dto.SummaryText, dto.Orphans, want.Summary.Text(), want.Summary.Orphans, want.Error)
	}
	if len(dto.Pending) != len(want.Pending) {
		t.Errorf("pendientes DTO %d, status %d", len(dto.Pending), len(want.Pending))
	}
}

func TestMarksChangeNameIsValidatedBeforeComposingAPath(t *testing.T) {
	e := newMarksEnv(t)
	outside := filepath.Join(filepath.Dir(e.ws.Home), "changes-escape")
	for _, change := range []string{"../x", "..", ".", "a/b", `a\b`, filepath.ToSlash(outside), outside, "", " c", "c:evil"} {
		resp, body := e.mark(change, "P-01", "dev", "ok", "")
		wantMarksRefusal(t, "mark "+change, resp, body, http.StatusBadRequest, "marks_body_invalid")
		resp, body = e.post("/api/increments/marks/publish", publishBody(change))
		wantMarksRefusal(t, "publish "+change, resp, body, http.StatusBadRequest, "marks_body_invalid")
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Errorf("se tocó el disco fuera del cambio: %v", err)
	}
	if identity, publish := e.fake.calls(); identity != 0 || publish != 0 {
		t.Errorf("el puerto se usó: %d %d", identity, publish)
	}
}

func TestMarksAvailabilityCodes(t *testing.T) {
	check := func(t *testing.T, e *marksEnv, change, code string) {
		t.Helper()
		body := map[string]any{"change": change, "case": "P-01", "environment": "dev", "result": "ok"}
		for _, path := range []string{"/api/increments/marks", "/api/increments/marks/publish"} {
			payload := body
			if strings.HasSuffix(path, "publish") {
				payload = publishBody(change)
			}
			resp, out := e.post(path, payload)
			wantMarksRefusal(t, path, resp, out, http.StatusConflict, code)
			if out["status"] != "unavailable" {
				t.Errorf("%s: status = %v", path, out["status"])
			}
		}
	}
	t.Run("monorepo sin publicador responde topología y no publicador ausente", func(t *testing.T) {
		svc, _ := explainerChange(t, "", atFiles(atHeader+atRow("P-01", "REQ-1.1#Alfa", "abrir")))
		srv := httptest.NewServer(NewServer(svc).Router())
		defer srv.Close()
		e := &marksEnv{t: t, svc: svc, srv: srv}
		check(t, e, "c", "marks_topology_unsupported")
		dto, err := svc.GetIncrementDetail("c")
		if err != nil {
			t.Fatal(err)
		}
		if m := dto.AcceptanceTests.Marks; m == nil || m.Available || m.Unavailable == nil || m.Unavailable.Code != "marks_topology_unsupported" || m.Unavailable.Continuation == "" {
			t.Errorf("DTO monorepo = %+v", m)
		}
	})
	t.Run("multirepo sin publicador", func(t *testing.T) {
		e := newMarksEnv(t)
		e.svc.SetMarksPublisher(nil)
		check(t, e, "c", "marks_publisher_missing")
	})
	t.Run("feature ODD", func(t *testing.T) {
		e := newMarksEnv(t)
		workspacetest.WriteFile(t, filepath.Join(e.ws.Specs, "odd", "tasks", "feat.md"), "# feat\n\n- [ ] 1.1 algo\n")
		check(t, e, "feat", "marks_not_sdd")
	})
	t.Run("cambio archivado", func(t *testing.T) {
		e := newMarksEnv(t)
		workspacetest.WriteFile(t, filepath.Join(e.ws.Home, "changes", "archive", "2026-01-01-viejo", "proposal.md"), "# v\n")
		check(t, e, "viejo", "marks_change_archived")
	})
	t.Run("función no activa", func(t *testing.T) {
		e := newMarksEnv(t)
		workspacetest.WriteFile(t, filepath.Join(e.ws.Specs, "axiom.yaml"), strings.Split(marksAxiomYAML, "acceptance_tests:")[0])
		check(t, e, "c", "marks_not_active")
	})
	t.Run("sin pruebas.md", func(t *testing.T) {
		e := newMarksEnv(t)
		if err := os.Remove(filepath.Join(e.root, "pruebas.md")); err != nil {
			t.Fatal(err)
		}
		check(t, e, "c", "marks_battery_missing")
	})
	t.Run("cambio inexistente es 404", func(t *testing.T) {
		e := newMarksEnv(t)
		resp, out := e.mark("nadie", "P-01", "dev", "ok", "")
		wantMarksRefusal(t, "inexistente", resp, out, http.StatusNotFound, "change_not_found")
	})
}

func TestMarksFileInvalidIsRejectedWithoutWriting(t *testing.T) {
	e := newMarksEnv(t)
	workspacetest.WriteFile(t, filepath.Join(e.root, marks.FileName), "esto: no es una secuencia\n")
	resp, body := e.mark("c", "P-01", "dev", "ok", "")
	wantMarksRefusal(t, "fichero ilegible", resp, body, http.StatusConflict, "marks_file_invalid")
	if got := e.marksFile(); got != "esto: no es una secuencia\n" {
		t.Errorf("el fichero cambió: %q", got)
	}
	if dto := e.detailMarks(); dto.Error == "" || len(dto.Cells) != 0 {
		t.Errorf("DTO con fichero ilegible = %+v", dto)
	}
	resp, body = e.post("/api/increments/marks/publish", publishBody("c"))
	wantMarksRefusal(t, "publicar con fichero ilegible", resp, body, http.StatusConflict, "marks_file_invalid")
}

func TestMarksHandlerPreflightOrder(t *testing.T) {
	e := newMarksEnv(t)
	for _, path := range []string{"/api/increments/marks", "/api/increments/marks/publish"} {
		resp, err := http.Get(e.srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("GET %s = %d", path, resp.StatusCode)
		}

		req, _ := http.NewRequest(http.MethodPost, e.srv.URL+path, strings.NewReader(`{}`))
		req.Host = "evil.example"
		req.Header.Set("Content-Type", "application/json")
		resp2, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp2.Body.Close()
		if resp2.StatusCode != http.StatusForbidden {
			t.Errorf("POST no local %s = %d", path, resp2.StatusCode)
		}

		resp3, body := e.postRaw(path, "text/plain", []byte(`{}`))
		wantMarksRefusal(t, "tipo de contenido", resp3, body, http.StatusUnsupportedMediaType, "marks_content_type")
		resp3, body = e.postRaw(path, "application/json", []byte(`{"change":"c","sorpresa":1}`))
		wantMarksRefusal(t, "campo desconocido", resp3, body, http.StatusBadRequest, "marks_body_invalid")
		huge := `{"change":"c","note":"` + strings.Repeat("x", 17<<10) + `"}`
		resp3, body = e.postRaw(path, "application/json", []byte(huge))
		wantMarksRefusal(t, "cuerpo grande", resp3, body, http.StatusBadRequest, "marks_body_invalid")
		resp3, body = e.postRaw(path, "application/json", []byte(`{"change":"c"} {"change":"c"}`))
		wantMarksRefusal(t, "dos objetos", resp3, body, http.StatusBadRequest, "marks_body_invalid")
	}
	if e.marksFile() != "" {
		t.Error("se escribió el fichero")
	}
	if identity, publish := e.fake.calls(); identity != 0 || publish != 0 {
		t.Errorf("el puerto se usó: %d %d", identity, publish)
	}
}

func TestMarksClockNeverRepeatsAt(t *testing.T) {
	var clock marksClockState
	fixed, err := time.Parse(marks.AtLayout, "2026-11-26T10:00:00.500Z")
	if err != nil {
		t.Fatal(err)
	}
	clock.now = func() time.Time { return fixed }
	first, second, third := clock.next(), clock.next(), clock.next()
	if first != "2026-11-26T10:00:00.500Z" || second != "2026-11-26T10:00:00.501Z" || third != "2026-11-26T10:00:00.502Z" {
		t.Errorf("at = %s %s %s", first, second, third)
	}
}

// TestMarksFilesRunNoGit fija la frontera de prov:WEB:3: los ficheros de marcas del dashboard no
// ejecutan git ni importan los paquetes que lo hacen.
func TestMarksFilesRunNoGit(t *testing.T) {
	forbidden := []string{"os/exec", "internal/gitcommon", "internal/gitidentity", "internal/sddpublish"}
	for _, name := range []string{"marks_port.go", "service_marks.go", "server_marks.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if path == bad || strings.HasSuffix(path, "/"+bad) {
					t.Errorf("%s importa %s", name, path)
				}
			}
		}
	}
}
