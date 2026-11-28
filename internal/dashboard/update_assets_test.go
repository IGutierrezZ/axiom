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
	t.Parallel()
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
	t.Parallel()
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		// Solo se muestra con update_available o restart_required; al día, dev_build y
		// check_failed lo ocultan.
		"info.status === 'update_available' && info.update_available",
		"if (!restartOnly && !updatable)",
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

// Con un binario instalado distinto del proceso y nada más reciente que instalar, el
// aviso ofrece reiniciar y no actualizar (RF-UPD-09); la orden manual sigue el canal.
func TestUpdateBannerOffersRestartWhenTheInstalledBinaryDiffers(t *testing.T) {
	t.Parallel()
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		"info.status === 'restart_required'",
		"escapeHtml(info.installed)",
		"Reinicia el dashboard para usarla.",
		`label: 'Reiniciar'`,
		"info.status === 'update_available' || info.status === 'restart_required'",
		"const from = info.installed || info.current;",
		"axiom@main",
		"latest.startsWith('main@')",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
}

// El botón permanente «Reiniciar con la última versión» lanza el reinicio forzado y
// la página se recarga cuando responde otra instancia del servidor, aunque la versión
// no cambie (RF-UPD-10).
func TestRestartLatestButtonIsWired(t *testing.T) {
	t.Parallel()
	index := getAsset(t, assetServer(t), "/").Body.String()
	for _, marker := range []string{
		`id="btn-axiom-restart"`,
		"Reiniciar con la última versión",
		`id="axiom-running-version"`,
	} {
		if !strings.Contains(index, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		"'/api/system/restart'",
		"startRestart(restartButton)",
		"restartInstance = currentInstance();",
		"info.instance !== restartInstance",
		"on('version', renderRunningVersion)",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
}

// «Actualizar» de Ecosistema sigue el mismo trabajo en segundo plano que el aviso y el
// reinicio: la tarjeta espera su final (o el reinicio) y el aviso superior no repite
// lo que la tarjeta ya pinta.
func TestEcosystemUpgradeUIFollowsTheBackgroundJob(t *testing.T) {
	t.Parallel()
	update := readAsset(t, "js/update.js")
	for _, marker := range []string{
		"export async function startEcosystemUpgrade(channel)",
		"api('/api/ecosystem/upgrade', { body: { channel } })",
		"on('updateJob', settleEcosystemWaiter)",
		"!ownedByEcosystem(job)",
	} {
		if !strings.Contains(update, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
	screen := readAsset(t, "js/screens/ecosystem.js")
	for _, marker := range []string{
		"startEcosystemUpgrade(channel).then(upgradeJobResult)",
		"presentError: presentUpgradeError",
	} {
		if !strings.Contains(screen, marker) {
			t.Errorf("js/screens/ecosystem.js no contiene %q", marker)
		}
	}
	if strings.Contains(screen, "api('/api/ecosystem/upgrade'") {
		t.Error("la pantalla no debe llamar al upgrade esperando la respuesta: lo lanza startEcosystemUpgrade")
	}
}

// El botón «Actualizar» lanza un trabajo asíncrono y la UI consulta su estado: la
// petición no depende del tiempo de espera del servidor y cada fase se ve.
func TestUpdateJobUIPollsStatusAndShowsEveryPhase(t *testing.T) {
	t.Parallel()
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

// Durante el reinicio la UI sigue consultando la versión hasta que el servidor
// responde con otra distinta y entonces recarga; si no vuelve, muestra el comando.
func TestUpdateJobUIReloadsWhenTheNewVersionAnswers(t *testing.T) {
	t.Parallel()
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		"job.phase === 'restarting'",
		"beginRestartWait(job)",
		"job.from_version",
		"pollRestart",
		"info.current !== restartFrom",
		"window.location.reload()",
		"RESTART_TIMEOUT_MS",
		"El dashboard no volvió a arrancar con la versión nueva.",
		"axiom ui",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
}

// Tras recargar, la UI recupera el estado del trabajo y muestra el resultado de la
// sincronización posterior a la actualización.
func TestUpdateJobUIShowsThePostUpdateSyncResult(t *testing.T) {
	t.Parallel()
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		"function syncSummary(sync)",
		"!sync.executed",
		"Sincronización del workspace completada.",
		"La sincronización del workspace falló",
		"Ver el detalle de la sincronización",
		"escapeHtml(lines.join(",
		"resumeUpdate()",
		"syncSummary(job.sync)",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
}
