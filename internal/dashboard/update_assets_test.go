package dashboard

import (
	"strings"
	"testing"
)

// Pruebas estructurales del aviso de versión nueva (P2, T2). No hay motor de
// JavaScript en `go test`: comprueban que los recursos se sirven y que contienen
// las piezas que dan el comportamiento (condiciones del aviso, botones, textos en
// castellano, enlace seguro).

func TestUpdateBannerAssetsAreServed(t *testing.T) {
	h := assetServer(t)
	if rr := getAsset(t, h, "/js/update.js"); rr.Code != 200 {
		t.Fatalf("GET /js/update.js = %d, se esperaba 200", rr.Code)
	}
	index := getAsset(t, h, "/").Body.String()
	if !strings.Contains(index, `id="update-banner"`) {
		t.Error("index.html debe contener el contenedor #update-banner")
	}
	// Oculto por defecto: sin versión nueva no se ve nada.
	if !strings.Contains(index, `id="update-banner" class="alert alert-warn update-banner" role="status" hidden`) {
		t.Error("#update-banner debe estar oculto por defecto y ser una región role=status")
	}
	if main := readAsset(t, "js/main.js"); !strings.Contains(main, "initUpdate()") {
		t.Error("js/main.js debe arrancar initUpdate()")
	}
}

func TestUpdateBannerOnlyShowsForAvailableUpdate(t *testing.T) {
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		// Solo se muestra con update_available; al día, dev_build y check_failed lo ocultan.
		"info.status !== 'update_available'",
		"banner.hidden = true",
		"/api/system/version",
		"/api/system/version?refresh=1",
		// El fallo de la comprobación no interrumpe el dashboard.
		"catch (err)",
		"return null",
		// Botones y textos en castellano.
		"Hay una versión nueva de Axiom",
		"Actualizar", "Comprobar de nuevo", "Ver la release",
		`data-action="update-now"`, `data-action="update-recheck"`,
		// El enlace solo admite https y se abre sin dar acceso a la ventana de origen.
		`/^https:\/\//i`,
		`rel="noopener noreferrer"`,
		"escapeHtml(info.current)", "escapeHtml(info.latest)",
		"POST",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
}

// El botón «Actualizar» lanza un trabajo asíncrono y la UI consulta su estado: la
// petición no depende del tiempo de espera del servidor y cada fase se ve.
func TestUpdateJobUIPollsStatusAndShowsEveryPhase(t *testing.T) {
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		"/api/system/update/status",
		"/api/system/update'",
		"POLL_MS",
		"setTimeout(pollUpdate, POLL_MS)",
		// Fases en castellano.
		"Comprobando", "Actualizando", "Reiniciando", "Sincronizando",
		"checking", "upgrading", "restarting", "syncing", "failed",
		// Un fallo muestra el motivo y el comando manual, con copiar y reintentar.
		"La actualización ha fallado", "job.manual_hint", "Copiar orden", "Reintentar",
		"escapeHtml(job.manual_hint)", "escapeHtml(job.message)",
		// Al recargar se recupera el trabajo en curso; el almacenamiento va protegido.
		"resumeUpdate()", "try {",
		// Los fallos de red al consultar se toleran un margen antes de rendirse.
		"MAX_POLL_ERRORS",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
}
