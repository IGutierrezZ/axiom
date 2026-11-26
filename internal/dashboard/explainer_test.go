package dashboard

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/explainer"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Sin t.Parallel en estas pruebas: montan proyectos git completos.

// explainerChange monta un cambio activo "c" con los ficheros dados y el kickoff sellado con el
// valor indicado ("" no lo informa). Devuelve el servicio y la carpeta del cambio.
func explainerChange(t *testing.T, sealed kickoff.ExplainerHTML, files map[string]string) (*Service, string) {
	t.Helper()
	ws := embeddedProject(t, hub.WorktreeModeInPlace)
	root := filepath.Join(ws.Home, "changes", "c")
	workspacetest.WriteFile(t, filepath.Join(root, "proposal.md"), "# c\n")
	for rel, content := range files {
		workspacetest.WriteFile(t, filepath.Join(root, filepath.FromSlash(rel)), content)
	}
	_, ok, err := kickoff.Seal(root, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: "c", SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			ExplainerHTML: sealed,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
	return NewService(ws.Master), root
}

func explainerGet(t *testing.T, srv *httptest.Server, method, query string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+"/api/explainer"+query, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func wantSecurityHeaders(t *testing.T, label string, resp *http.Response) {
	t.Helper()
	if got := resp.Header.Get("Content-Security-Policy"); got != explainerCSP {
		t.Errorf("%s: CSP = %q, quería %q", label, got, explainerCSP)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("%s: X-Content-Type-Options = %q", label, got)
	}
}

func TestExplainerEndpointServesWhitelistedHTML(t *testing.T) {
	svc, root := explainerChange(t, "", map[string]string{"resumen-spec.html": "<html>spec</html>"})
	workspacetest.WriteFile(t, filepath.Join(filepath.Dir(root), "resumen-spec.html"), "<html>hermano</html>")
	srv := httptest.NewServer(NewServer(svc).Router())
	defer srv.Close()

	resp, body := explainerGet(t, srv, http.MethodGet, "?change=c&kind=spec")
	if resp.StatusCode != http.StatusOK || body != "<html>spec</html>" {
		t.Fatalf("GET = %d %q", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := resp.Header.Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q", got)
	}
	wantSecurityHeaders(t, "200", resp)

	resp, body = explainerGet(t, srv, http.MethodHead, "?change=c&kind=spec")
	if resp.StatusCode != http.StatusOK || body != "" {
		t.Errorf("HEAD = %d %q", resp.StatusCode, body)
	}

	resp, _ = explainerGet(t, srv, http.MethodGet, "?change=c&kind=design")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("design ausente = %d, quería 404", resp.StatusCode)
	}
	wantSecurityHeaders(t, "404", resp)

	resp, _ = explainerGet(t, srv, http.MethodGet, "?change=nope&kind=spec")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cambio inexistente = %d, quería 404", resp.StatusCode)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		resp, _ = explainerGet(t, srv, method, "?change=c&kind=spec")
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s = %d, quería 405", method, resp.StatusCode)
		}
	}
}

func TestExplainerEndpointRejectsBadParameters(t *testing.T) {
	svc, _ := explainerChange(t, "", map[string]string{"resumen-spec.html": "x", "spec.md": "x"})
	srv := httptest.NewServer(NewServer(svc).Router())
	defer srv.Close()

	changes := []string{"..", "../x", "x/..", "a/b", `a\b`, "%2e%2e", "a%2fb", "%5c", "C:x", "%00", "", "."}
	for _, c := range changes {
		resp, _ := explainerGet(t, srv, http.MethodGet, "?change="+c+"&kind=spec")
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("change=%q = %d, quería 400", c, resp.StatusCode)
		}
		wantSecurityHeaders(t, "400 "+c, resp)
	}
	for _, k := range []string{"spec.md", "kickoff.yaml", "tasks", "", "SPEC"} {
		resp, _ := explainerGet(t, srv, http.MethodGet, "?change=c&kind="+url.QueryEscape(k))
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("kind=%q = %d, quería 400", k, resp.StatusCode)
		}
	}
}

func TestExplainerEndpointRefusesUnsafeFiles(t *testing.T) {
	svc, root := explainerChange(t, "yes", map[string]string{"spec.md": "x"})
	srv := httptest.NewServer(NewServer(svc).Router())
	defer srv.Close()
	html := filepath.Join(root, "resumen-spec.html")

	if err := os.Mkdir(html, 0o755); err != nil {
		t.Fatal(err)
	}
	if resp, _ := explainerGet(t, srv, http.MethodGet, "?change=c&kind=spec"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("directorio = %d, quería 404", resp.StatusCode)
	}
	if err := os.Remove(html); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(filepath.Join(root, "spec.md"), html); err == nil {
		if resp, _ := explainerGet(t, srv, http.MethodGet, "?change=c&kind=spec"); resp.StatusCode != http.StatusNotFound {
			t.Errorf("enlace simbólico = %d, quería 404", resp.StatusCode)
		}
		_ = os.Remove(html)
	} else {
		t.Logf("enlace simbólico omitido: %v", err)
	}

	big := "SECRETO-" + strings.Repeat("a", explainer.MaxBytes)
	if err := os.WriteFile(html, []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, body := explainerGet(t, srv, http.MethodGet, "?change=c&kind=spec")
	if resp.StatusCode != http.StatusInternalServerError || !strings.Contains(body, "5 MiB") || strings.Contains(body, "SECRETO") {
		t.Errorf("fichero grande = %d %.80q", resp.StatusCode, body)
	}
	wantSecurityHeaders(t, "500", resp)
}

func TestIncrementDetailExplainer(t *testing.T) {
	cases := []struct {
		name    string
		sealed  kickoff.ExplainerHTML
		files   map[string]string
		enabled bool
		items   map[string]string
	}{
		{name: "no", sealed: "no", files: map[string]string{"spec.md": "x"}, items: map[string]string{}},
		{name: "sin campo", sealed: "", files: map[string]string{"spec.md": "x", "resumen-spec.html": "<html>"}, items: map[string]string{}},
		{name: "yes sin md", sealed: "yes", enabled: true, items: map[string]string{}},
		{name: "yes con md", sealed: "yes", files: map[string]string{"spec.md": "x", "design.md": "d", "resumen-design.html": "<html>"},
			enabled: true, items: map[string]string{"spec": "missing", "design": "stale"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, root := explainerChange(t, tc.sealed, tc.files)
			dto, err := svc.GetIncrementDetail("c")
			if err != nil {
				t.Fatal(err)
			}
			if dto.Explainer.Enabled != tc.enabled || dto.Explainer.Items == nil {
				t.Fatalf("Explainer = %+v", dto.Explainer)
			}
			got := map[string]string{}
			for _, it := range dto.Explainer.Items {
				got[it.Kind] = it.State
			}
			if len(got) != len(tc.items) {
				t.Fatalf("items = %v, quería %v", got, tc.items)
			}
			for k, v := range tc.items {
				if got[k] != v {
					t.Errorf("items[%s] = %q, quería %q", k, got[k], v)
				}
			}
			// El dashboard y el status usan la misma función de frescura.
			want := sddstatus.ExplainerFreshness(root, sddstatus.ArtifactStoreOpenSpec)
			if want.Enabled != dto.Explainer.Enabled {
				t.Errorf("Enabled = %v, ExplainerFreshness = %v", dto.Explainer.Enabled, want.Enabled)
			}
			if want.Spec != nil && got["spec"] != want.Spec.State {
				t.Errorf("spec = %q, ExplainerFreshness = %q", got["spec"], want.Spec.State)
			}
			if want.Design != nil && got["design"] != want.Design.State {
				t.Errorf("design = %q, ExplainerFreshness = %q", got["design"], want.Design.State)
			}
		})
	}
}

func TestIncrementDetailExplainerArchived(t *testing.T) {
	svc, root := explainerChange(t, "yes", map[string]string{"spec.md": "x"})
	archived := filepath.Join(filepath.Dir(root), "archive", "2026-01-01-c")
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, archived); err != nil {
		t.Fatal(err)
	}
	dto, err := svc.GetIncrementDetail("c")
	if err != nil {
		t.Fatal(err)
	}
	if !dto.Explainer.Enabled || len(dto.Explainer.Items) != 1 || dto.Explainer.Items[0].State != "missing" {
		t.Errorf("archivado: Explainer = %+v", dto.Explainer)
	}
	workspacetest.WriteFile(t, filepath.Join(archived, "resumen-spec.html"), "<html>archivado</html>")
	srv := httptest.NewServer(NewServer(svc).Router())
	defer srv.Close()
	if resp, body := explainerGet(t, srv, http.MethodGet, "?change=c&kind=spec"); resp.StatusCode != http.StatusOK || body != "<html>archivado</html>" {
		t.Errorf("archivado servido = %d %q", resp.StatusCode, body)
	}
}
