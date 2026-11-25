package dashboard

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// La ruta del interruptor RDD ya no existe: ni GET ni POST la atienden.
func TestRetiredReviewModeRouteIsNotFound(t *testing.T) {
	router, _, _ := newPrefsServer(t, t.TempDir())

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		rr := request(router, method, "/api/project/review-mode", `{"mode":"on"}`)
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s /api/project/review-mode = %d, quiero %d: %s", method, rr.Code, http.StatusNotFound, rr.Body.String())
		}
	}
}

// Un cliente antiguo que todavía envía `review_mode` en el alta no recibe error: el campo
// se ignora y el proyecto se crea igual.
func TestProjectsInitIgnoresTheRetiredReviewModeField(t *testing.T) {
	router, _, _ := newPrefsServer(t, t.TempDir())
	dir := filepath.Join(t.TempDir(), "cliente-antiguo")
	writeAdoptionFile(t, dir, "README.md", "# Proyecto\n")

	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`,"review_mode":"on"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init con review_mode = %d: %s", rr.Code, rr.Body.String())
	}
	if body := rr.Body.String(); strings.Contains(body, "review_mode") {
		t.Errorf("la respuesta no debe mencionar el campo retirado: %s", body)
	}
}
