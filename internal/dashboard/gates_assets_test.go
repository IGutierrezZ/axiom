package dashboard

import (
	"strings"
	"testing"
)

// TestGatePanelAssets fija en el JS y el HTML embebidos del panel de puerta las garantías de la UI
// de decisiones (C-06-f, D9): diálogos propios y no confirm(), la publicación solo tras aceptar el
// diálogo, ninguna llamada hacia el push de código, el commitReview explícito al aprobar `tasks`
// y todo texto del servidor escapado.
func TestGatePanelAssets(t *testing.T) {
	t.Parallel()
	src := readAsset(t, "js/screens/detalle-puerta.js")

	for _, marker := range []string{
		"export async function renderGate", "confirmDialog", "promptDialog",
		"/api/increments/gate/stop?change=", "'/api/increments/gate'", "'/api/increments/gate/publish'",
		"stop.choices", "choice.label", "choice.effect",
		"'notify'", "'stop'", "'stop-before-commit'", "commitReview",
		"pr_merged", "baseRef", "pushCommand", "handedOffTo", "lastRoleClosed", "pendingNonBlockingRoles",
		"notices", "gate-unpublished", "aprobada sin publicar", "Remoto: ", "Rama: ",
		"escapeHtml(choice.label)", "escapeHtml(choice.effect)",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("detalle-puerta.js no contiene %q", marker)
		}
	}
	for _, forbidden := range []string{"confirm(", "fetch(", "git push", "innerHTML = res", "${choice.label}", "${choice.effect}", "${n.text}"} {
		if strings.Contains(src, forbidden) {
			t.Errorf("detalle-puerta.js no debe contener %q", forbidden)
		}
	}
	// La publicación solo se pide después del diálogo y dentro de su rama de aceptación.
	dlg := strings.Index(src, "await confirmDialog(")
	pub := strings.Index(src, "api('/api/increments/gate/publish'")
	rec := strings.Index(src, "api('/api/increments/gate'")
	if dlg < 0 || rec < 0 || pub < 0 || !(dlg < rec && rec < pub) {
		t.Errorf("el orden debe ser confirmDialog, registro, publicación (dlg=%d, rec=%d, pub=%d)", dlg, rec, pub)
	}
	if strings.Count(src, "api('/api/increments/gate/publish'") != 1 {
		t.Error("la publicación debe pedirse desde un único punto")
	}
	if !strings.Contains(src, "if (!ok) return") {
		t.Error("cancelar el diálogo no debe registrar ni publicar nada")
	}
	if !strings.Contains(src, "record.publish") {
		t.Error("la publicación solo se encadena si el registro devuelve publish")
	}
	// El commitReview viaja explícito en la aprobación de tasks.
	if !strings.Contains(src, "body.commitReview = ") {
		t.Error("la aprobación de tasks debe enviar commitReview explícito")
	}

	html := readAsset(t, "index.html")
	if !strings.Contains(html, `<section id="inc-gate" class="gate-panel" hidden`) {
		t.Error("index.html no contiene la sección #inc-gate oculta")
	}
	if strings.Index(html, `id="inc-stepper"`) > strings.Index(html, `id="inc-gate"`) ||
		strings.Index(html, `id="inc-gate"`) > strings.Index(html, `id="inc-tabs"`) {
		t.Error("#inc-gate debe estar entre #inc-stepper y #inc-tabs")
	}
	det := readAsset(t, "js/screens/detalle.js")
	if !strings.Contains(det, "renderGate(") || !strings.Contains(det, "detalle-puerta.js") {
		t.Error("detalle.js no engancha renderGate")
	}
	css := readAsset(t, "style.css")
	for _, class := range []string{".gate-panel", ".gate-choices", ".gate-choice-effect", ".gate-form", ".gate-unpublished"} {
		if !strings.Contains(css, class) {
			t.Errorf("style.css no define %s", class)
		}
	}
}

func TestGateAssetIsServed(t *testing.T) {
	t.Parallel()
	rr := getAsset(t, assetServer(t), "/js/screens/detalle-puerta.js")
	if rr.Code != 200 || !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/javascript") {
		t.Errorf("detalle-puerta.js: código %d, tipo %q", rr.Code, rr.Header().Get("Content-Type"))
	}
}

// El bloque «aprobada sin publicar» se reconstruye tras recargar desde datos persistentes
// (executionStyle y publication.state), sin estado en memoria (prov:WEB:7, D9).
func TestGateUnpublishedRebuiltFromPublicationState(t *testing.T) {
	t.Parallel()
	src := readAsset(t, "js/screens/detalle-puerta.js")
	for _, marker := range []string{
		"function pendingOf", `d.executionStyle !== "checkpointed"`, `pub.state !== "pending"`,
		"(res && res.unpublished) || pendingOf(view)", `pub.state !== "unavailable"`, "pub.unavailable",
		"code_unpublished", "pub.pushCommand",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("detalle-puerta.js no contiene %q", marker)
		}
	}
	// El reintento reutiliza el diálogo de aviso remoto y no registra nada.
	retry := src[strings.Index(src, "async function retryFlow"):]
	retry = retry[:strings.Index(retry, "async function iterateFlow")]
	for _, want := range []string{"await confirmDialog(", "Remoto: ${u.remote}", "Rama: ${u.branch}", "publishGate("} {
		if !strings.Contains(retry, want) {
			t.Errorf("retryFlow no contiene %q", want)
		}
	}
	if strings.Contains(retry, "'/api/increments/gate'") {
		t.Error("el reintento solo publica: no debe registrar la decisión")
	}
}
