package dashboard

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Las pruebas de la pantalla «Integraciones» (prov:WEB:7) inspeccionan los assets: no hay
// ejecutor de JavaScript en el arnés (H-05), de modo que se fijan por texto los invariantes de
// seguridad y de estructura. La comprobación visual queda para verify.

var integrationsScreenModules = []string{"js/screens/integraciones.js", "js/core/integrations.js"}

func TestIntegrationsScreenAssetsAreServedAndResolved(t *testing.T) {
	t.Parallel()
	h := assetServer(t)
	for _, f := range integrationsScreenModules {
		rr := getAsset(t, h, "/"+f)
		if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/javascript") {
			t.Errorf("GET /%s = %d, %q", f, rr.Code, rr.Header().Get("Content-Type"))
		}
	}
	main := readAsset(t, "js/main.js")
	for _, marker := range []string{"./screens/integraciones.js", "integraciones: 'view-integraciones'", "initIntegrations", "loadIntegrations"} {
		if !strings.Contains(main, marker) {
			t.Errorf("main.js no contiene %q", marker)
		}
	}
	if !strings.Contains(readAsset(t, "js/screens/integraciones.js"), "../core/integrations.js") {
		t.Error("la pantalla no importa core/integrations.js")
	}
}

func TestIntegrationsScreenAssetsRouteAndNavigation(t *testing.T) {
	t.Parallel()
	if !strings.Contains(readAsset(t, "js/core/router.js"), "'integraciones'") {
		t.Error("router.js no define la ruta 'integraciones'")
	}
	if !strings.Contains(readAsset(t, "js/shell.js"), "integraciones: 'Integraciones'") {
		t.Error("shell.js no etiqueta la ruta integraciones")
	}
	index := readAsset(t, "index.html")
	for _, marker := range []string{`href="#/integraciones"`, `data-nav="integraciones"`, `id="view-integraciones"`, `id="integrations-body"`} {
		if !strings.Contains(index, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}
}

func TestIntegrationsScreenAssetsPATInputIsHardened(t *testing.T) {
	t.Parallel()
	screen := readAsset(t, "js/screens/integraciones.js") + readAsset(t, "js/core/integrations.js")
	for _, marker := range []string{`type="password"`, `autocomplete="off"`, "data-secret-input", "confirm: true"} {
		if !strings.Contains(screen, marker) {
			t.Errorf("la pantalla no contiene %q", marker)
		}
	}
	// El campo del PAT no lleva data-key: el lote de PUT /api/settings solo recoge [data-key].
	inputTag := regexp.MustCompile(`<input[^>]*data-secret-input[^>]*>`).FindString(screen)
	if inputTag == "" || strings.Contains(inputTag, "data-key") {
		t.Errorf("el campo del PAT debe existir y no llevar data-key: %q", inputTag)
	}
	if !strings.Contains(inputTag, `type="password"`) || !strings.Contains(inputTag, `autocomplete="off"`) {
		t.Errorf("el campo del PAT debe ser password y autocomplete=off: %q", inputTag)
	}
	if !strings.Contains(screen, "value = ''") {
		t.Error("la pantalla no vacía el campo del PAT tras guardar")
	}
}

func TestIntegrationsScreenAssetsNeverPersistOrInterpolateUnsafely(t *testing.T) {
	t.Parallel()
	for _, f := range integrationsScreenModules {
		src := readAsset(t, f)
		for _, banned := range []string{"localStorage", "sessionStorage", "document.cookie", "eval(", "insertAdjacentHTML"} {
			if strings.Contains(src, banned) {
				t.Errorf("%s usa %q", f, banned)
			}
		}
		// Un href nunca se interpola con datos: los enlaces externos se crean con a.href = ….
		if regexp.MustCompile(`href="[^"]*\$\{`).MatchString(src) {
			t.Errorf("%s interpola datos dentro de un href en una plantilla HTML", f)
		}
	}
	all := readAsset(t, "js/screens/integraciones.js") + readAsset(t, "js/core/integrations.js")
	for _, marker := range []string{"escapeHtml", "textContent", "safeExternalURL", "noopener noreferrer", "#/ajustes"} {
		if !strings.Contains(all, marker) {
			t.Errorf("la pantalla no contiene %q", marker)
		}
	}
	if !strings.Contains(readAsset(t, "js/core/integrations.js"), "export function safeExternalURL") {
		t.Error("core/integrations.js no exporta safeExternalURL")
	}
}

func TestIntegrationsScreenAssetsReuseTheSettingsFlow(t *testing.T) {
	t.Parallel()
	screen := readAsset(t, "js/screens/integraciones.js")
	for _, marker := range []string{
		"controlHtml", "collectChanges", "'/api/settings'", "method: 'PUT'", "revision",
		"/api/integrations/secret", "/api/integrations/secret/delete", "/api/integrations/connect", "/api/integrations/discover",
		"writeBlockedReason", "documentErrors", "shadowedByEnv", "<datalist", "no encontrado en el tracker",
	} {
		if !strings.Contains(screen+readAsset(t, "js/core/integrations.js"), marker) {
			t.Errorf("la pantalla no contiene %q", marker)
		}
	}
}

func TestIntegrationsScreenAssetsWorkItemStartForm(t *testing.T) {
	t.Parallel()
	inc := readAsset(t, "js/screens/incrementos.js")
	for _, marker := range []string{
		"/api/integrations", "/api/integrations/start", "/api/integrations/start/discard", "confirm: true",
		"confirmDialog", "startFromWorkItem", "workItemsApplicable", "Reintentar", "Descartar", "textContent",
	} {
		if !strings.Contains(inc, marker) {
			t.Errorf("incrementos.js no contiene %q", marker)
		}
	}
	for _, banned := range []string{"workitem", "work-item", "WorkItem"} {
		// El work item nunca se recuerda en el navegador: ninguna escritura de preferencias lo nombra.
		if regexp.MustCompile(`writePref\([^)]*` + banned).MatchString(inc) {
			t.Errorf("incrementos.js guarda el work item con writePref (%s)", banned)
		}
	}
	// Ningún dato del tracker entra en innerHTML.
	if regexp.MustCompile(`innerHTML\s*=[^;]*(err\.|failure|warning|message)`).MatchString(inc[strings.Index(inc, "// ---- Modal"):]) {
		t.Error("el modal de alta interpola datos del servidor en innerHTML")
	}
	index := readAsset(t, "index.html")
	for _, marker := range []string{`id="field-inc-workitem"`, `id="input-inc-workitem"`, `id="new-inc-recovery"`, `id="btn-wi-retry"`, `id="btn-wi-discard"`, `id="new-inc-notes"`} {
		if !strings.Contains(index, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}
	if !strings.Contains(readAsset(t, "js/core/branches.js"), "export async function startFromWorkItem") {
		t.Error("core/branches.js no exporta startFromWorkItem")
	}
}
