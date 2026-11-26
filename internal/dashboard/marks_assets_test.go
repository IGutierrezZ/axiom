package dashboard

import (
	"strings"
	"testing"
)

// TestAcceptanceMarksAssets fija en el JS embebido de la pestaña «Pruebas» las garantías de la UI de
// marcas: texto de usuario siempre escapado, publicación solo tras confirmDialog con confirm y rama,
// sin control en celdas no aplicables y rechazos tipados visibles con su continuación.
func TestAcceptanceMarksAssets(t *testing.T) {
	t.Parallel()
	src := readAsset(t, "js/screens/detalle-pruebas.js")

	for _, marker := range []string{
		"escapeHtml(cur.note)", "escapeHtml(cur.commit)", "escapeHtml(entryText(e))",
		"caducada: ${escapeHtml(cell.stale.result)}",
		"marks.noteMax", "marks.commitMax",
		"'/api/increments/marks/publish'", "'/api/increments/marks'",
		"confirm: true, branch: m.branch",
		"Remoto: ${m.remote}", "Publicar marcas",
		"marks.pendingUnavailable", "refusalHtml(marks.pendingUnavailable)", "continuation",
		"marks.unavailable", "uncommittedDocs",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("detalle-pruebas.js no contiene %q", marker)
		}
	}

	// confirmDialog precede a la petición de publicar.
	dlg := strings.Index(src, "await confirmDialog(")
	pub := strings.Index(src, "api('/api/increments/marks/publish'")
	if dlg < 0 || pub < 0 || dlg > pub {
		t.Errorf("la publicación debe pedirse tras confirmDialog (dlg=%d, pub=%d)", dlg, pub)
	}
	// El botón se deshabilita sin pendientes o con pendingUnavailable.
	if !strings.Contains(src, "pending.length === 0 || !!marks.pendingUnavailable") {
		t.Error("el botón de publicar debe deshabilitarse con 0 pendientes o con pendingUnavailable")
	}
	// Celda no aplicable: «—» sin formulario.
	if !strings.Contains(src, "cell ? cellHtml(cell, marks) : '—'") {
		t.Error("las celdas no aplicables deben pintarse como «—» sin control")
	}
	// Ninguna nota ni commit se interpola sin escapar.
	for _, raw := range []string{"${cur.note}", "${cur.commit}", "${e.note}", "${e.commit}", "innerHTML = res"} {
		if strings.Contains(src, raw) {
			t.Errorf("detalle-pruebas.js interpola sin escapar: %q", raw)
		}
	}
}
