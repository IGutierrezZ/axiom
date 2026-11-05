package dashboard

import (
	"strings"
	"testing"
)

// La recarga tras actualizar no puede depender de que un sondeo llegue a ver la fase
// «restarting» (dura menos que el intervalo de sondeo): también se recarga cuando
// responde un proceso distinto (otro from_version) o tras fallos de red previos.
func TestUpdateJobUIReloadDoesNotDependOnObservingRestarting(t *testing.T) {
	src := readAsset(t, "js/update.js")
	for _, marker := range []string{
		"function newProcessAnswered(job)",
		"updateStarted && (downSeen || (job.from_version || '') !== startedFrom)",
		"trackUpdate(job)",
		"downSeen = true",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("js/update.js no contiene %q", marker)
		}
	}
	// El chequeo ocurre en el sondeo normal, antes de pintar y de mirar la fase.
	poll := src[strings.Index(src, "async function pollUpdate()"):]
	check := strings.Index(poll, "newProcessAnswered(job)")
	phase := strings.Index(poll, "job.phase === 'restarting'")
	if check < 0 || phase < 0 || check > phase {
		t.Fatalf("pollUpdate debe comprobar newProcessAnswered(job) antes de depender de la fase restarting (%d, %d)", check, phase)
	}
	if !strings.Contains(poll[check:], "window.location.reload()") {
		t.Error("pollUpdate debe recargar al detectar el proceso nuevo")
	}
}
