package dashboard

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Las pruebas de este fichero cubren la pantalla de Ecosistema (T6) y la
// retirada del CSS heredado (T7). La lógica pura (traducción al castellano,
// orden del Doctor, tamaños y cadena upgrade -> sync) se ejecuta con Node; lo
// demás es estructural, porque `go test` no trae un motor de DOM.

func TestPlatformAssetsAreServed(t *testing.T) {
	h := assetServer(t)
	for _, target := range []string{"/js/core/ecosystem.js", "/js/screens/ecosystem.js"} {
		rr := getAsset(t, h, target)
		if rr.Code != 200 {
			t.Errorf("GET %s = %d, se esperaba 200", target, rr.Code)
			continue
		}
		if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") {
			t.Errorf("GET %s: Content-Type = %q, un módulo ES exige text/javascript", target, got)
		}
	}
}

// legacy.css desaparece del todo: del paquete, del HTML y de cualquier referencia.
func TestLegacyStylesAreRemoved(t *testing.T) {
	for _, f := range embeddedFiles(t) {
		if strings.Contains(strings.ToLower(f), "legacy") {
			t.Errorf("el fichero embebido %s no debería existir: el CSS heredado se retiró", f)
		}
	}
	h := assetServer(t)
	if rr := getAsset(t, h, "/legacy.css"); rr.Code != 404 {
		t.Errorf("GET /legacy.css = %d, se esperaba 404", rr.Code)
	}
	for _, f := range embeddedFiles(t) {
		if !strings.HasSuffix(f, ".html") && !strings.HasSuffix(f, ".css") && !strings.HasSuffix(f, ".js") {
			continue
		}
		if strings.Contains(readAsset(t, f), "legacy.css") {
			t.Errorf("%s todavía menciona legacy.css", f)
		}
	}
	// Las clases que solo existían en legacy.css no se usan en el marcado ni en los módulos.
	for _, class := range []string{"legacy-card", "stat-cards-row", "stat-card", "stat-label", "stat-value", "badge-phase", "text-accent", "subsection-title"} {
		if strings.Contains(readAsset(t, "index.html"), class) || strings.Contains(readAllJS(t), class) {
			t.Errorf("la clase heredada %q sigue en uso", class)
		}
	}
	// Solo tokens: ningún color fijo en la hoja de estilos, y ninguna variable --color-* de la interfaz antigua.
	css := readAsset(t, "style.css")
	if loc := regexp.MustCompile(`#[0-9A-Fa-f]{3,8}\b`).FindString(css); loc != "" {
		t.Errorf("style.css contiene el color fijo %q; usa los tokens de tokens.css", loc)
	}
	if strings.Contains(css, "--color-") {
		t.Errorf("style.css usa variables --color-* de la interfaz antigua")
	}
}

// Todo id que consulta el JavaScript existe en el HTML (o lo genera el propio módulo), y las
// clases del bloque de Ecosistema están definidas en style.css.
func TestEcosystemMarkupMatchesTheScreen(t *testing.T) {
	html := readAsset(t, "index.html")
	js := readAsset(t, "js/screens/ecosystem.js")
	start, end := strings.Index(html, `id="view-ecosistema"`), strings.Index(html, "</main>")
	if start < 0 || end < start {
		t.Fatalf("index.html: no se localiza la vista de Ecosistema")
	}
	view := html[start:end]
	for _, m := range regexp.MustCompile(`\$\('#([\w-]+)'\)`).FindAllStringSubmatch(js, -1) {
		if !strings.Contains(view, `id="`+m[1]+`"`) && !strings.Contains(js, `id="`+m[1]+`"`) {
			t.Errorf("ecosystem.js consulta #%s, que no existe en el marcado", m[1])
		}
	}
	css := readAsset(t, "style.css")
	for _, class := range regexp.MustCompile(`class="(eco-[\w-]+)`).FindAllStringSubmatch(view+js, -1) {
		if !strings.Contains(css, "."+class[1]) {
			t.Errorf("la clase %s se usa pero style.css no la define", class[1])
		}
	}
	for _, marker := range []string{
		"btn-eco-sync", "btn-eco-upgrade", "btn-eco-create-backup", "select-eco-channel", "eco-last", "eco-health",
		"eco-doctor-body", "eco-topology-body", "eco-models-body", "eco-backups-body", "eco-persona",
	} {
		if !strings.Contains(view, `id="`+marker+`"`) {
			t.Errorf("la vista de Ecosistema no contiene #%s", marker)
		}
	}
	// Los botones de operación y la última operación son nativos y accesibles.
	for _, marker := range []string{"<details id=\"eco-last\"", "role=\"status\" aria-live=\"polite\"", "data-eco-lock"} {
		if !strings.Contains(view, marker) {
			t.Errorf("la vista de Ecosistema no contiene %q", marker)
		}
	}
}

// El Doctor se lee de la caché compartida; la pantalla no lo consulta por su cuenta.
func TestEcosystemUsesTheSharedDoctorCache(t *testing.T) {
	js := readAsset(t, "js/screens/ecosystem.js")
	if strings.Contains(js, "api('/api/ecosystem/doctor')") {
		t.Errorf("ecosystem.js consulta el Doctor directamente; debe usar la caché de overview.js")
	}
	for _, marker := range []string{"state.overview.doctor", "refreshOverview(['doctor'])", "on('overview'", "sortChecks(", "statusView(", "categoryText("} {
		if !strings.Contains(js, marker) {
			t.Errorf("ecosystem.js no contiene %q", marker)
		}
	}
	// Las rutas de los demás bloques y de las operaciones siguen siendo las del servidor.
	for _, route := range []string{"/api/workspace", "/api/ecosystem/models", "/api/ecosystem/backups", "/api/ecosystem/sync", "/api/ecosystem/upgrade", "/api/ecosystem/backups/create", "/api/ecosystem/backups/restore"} {
		if !strings.Contains(js, "'"+route+"'") {
			t.Errorf("ecosystem.js no usa %s", route)
		}
	}
	// Un bloque que falla muestra su error y no vacía el resto: cada carga captura el suyo.
	for _, fn := range []string{"loadTopology", "loadModels", "loadBackups"} {
		body := functionBody(t, js, "async function "+fn+"(")
		if !strings.Contains(body, "catch (err)") || !strings.Contains(body, "stateError(") {
			t.Errorf("%s debe capturar su error y mostrarlo con stateError", fn)
		}
	}
}

// Restaurar un respaldo pide confirmación explícita, con las consecuencias, antes de llamar a la API.
func TestRestoreBackupAsksForConfirmationBeforeCallingTheAPI(t *testing.T) {
	js := readAsset(t, "js/screens/ecosystem.js")
	body := functionBody(t, js, "async function restoreBackup(")
	confirm := strings.Index(body, "confirmDialog(")
	call := strings.Index(body, "'/api/ecosystem/backups/restore'")
	if confirm < 0 || call < 0 || confirm > call {
		t.Fatalf("restoreBackup debe llamar a confirmDialog antes que a la API (confirm=%d, api=%d)", confirm, call)
	}
	if !strings.Contains(body, "if (!ok) return;") || strings.Index(body, "if (!ok) return;") > call {
		t.Errorf("cancelar la confirmación debe salir antes de llamar a la API")
	}
	if !strings.Contains(body, "danger: true") {
		t.Errorf("la confirmación de restaurar debe ser un diálogo de acción peligrosa")
	}
	msg := functionBody(t, js, "function restoreMessage(")
	for _, want := range []string{"sustituirán", "se perderán", "no se deshace"} {
		if !strings.Contains(msg, want) {
			t.Errorf("el mensaje de confirmación debe explicar las consecuencias (%q)", want)
		}
	}
	// El botón no llama a la API por sí mismo: solo la delegación abre la confirmación.
	if strings.Count(js, "'/api/ecosystem/backups/restore'") != 1 {
		t.Errorf("la ruta de restaurar debe usarse en un único punto")
	}
	// Mientras corre una operación, el resto de botones quedan deshabilitados.
	for _, marker := range []string{"setLocked(true)", "setLocked(false)", "setBusy(button, busyLabel)", "data-eco-lock"} {
		if !strings.Contains(js, marker) {
			t.Errorf("ecosystem.js no contiene %q", marker)
		}
	}
}

// functionBody devuelve el texto de una función desde su cabecera hasta la siguiente declaración de primer nivel.
func functionBody(t *testing.T, src, header string) string {
	t.Helper()
	start := strings.Index(src, header)
	if start < 0 {
		t.Fatalf("no se encuentra %q", header)
	}
	rest := src[start:]
	if end := strings.Index(rest, "\n}\n"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// La lógica pura de Ecosistema se ejecuta de verdad con Node.
func TestEcosystemLogicRuntime(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite la prueba de ejecución de la lógica de Ecosistema")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ecosystem.mjs"), []byte(readAsset(t, "js/core/ecosystem.js")), 0o600); err != nil {
		t.Fatal(err)
	}
	script := `import * as eco from './ecosystem.mjs';
const checks = [
  { name: 'ok-1', status: 'ok', category: 'agent' },
  { name: 'warn-1', status: 'warning', category: 'tool' },
  { name: 'odd', status: 'degraded', category: 'cloud' },
  { name: 'err-1', status: 'error', category: 'environment' },
  { name: 'warn-2', status: 'warning', category: 'agent' },
  { name: 'ok-2', status: 'ok', category: 'tool' },
];
const out = {};
out.status = ['ok', 'warning', 'error', 'degraded', '', undefined].map(s => eco.statusView(s));
out.category = ['agent', 'tool', 'environment', 'cloud', undefined].map(c => eco.categoryText(c));
out.order = eco.sortChecks(checks).map(c => c.name);
out.orderIsStable = eco.sortChecks(checks).filter(c => c.status === 'warning').map(c => c.name);
out.empty = [eco.sortChecks(null), eco.sortChecks(undefined)];
out.summaryOk = eco.healthSummary({ healthy: true, checks: [{ status: 'ok' }, { status: 'ok' }] });
out.summaryWarn = eco.healthSummary({ healthy: true, checks: [{ status: 'ok' }, { status: 'warning' }] });
out.summaryUnhealthy = eco.healthSummary({ healthy: false, checks: [{ status: 'ok' }] });
out.summaryErr = eco.healthSummary({ healthy: false, checks });
out.summaryNone = eco.healthSummary(null);
out.bytes = [0, undefined, -4, 512, 2048, 1258291, 1073741824].map(b => eco.formatBytes(b));
out.noPhases = [eco.formatEcosystemUpgradeSequence(null), eco.formatEcosystemUpgradeSequence({ output: ['a', 'b'] }), eco.formatEcosystemUpgradeSequence({ message: 'listo' })];
out.restart = eco.formatEcosystemUpgradeSequence({ phases: { upgrade: { status: 'updated', restart_required: true, manual_hint: '' }, sync: { executed: false, skipped_reason: 'restart-required' } } });
out.failed = eco.formatEcosystemUpgradeSequence({ phases: { upgrade: { status: 'failed', error: 'sin red', manual_hint: 'descarga el binario' }, sync: { executed: false, skipped_reason: 'upgrade-failed' } } });
out.both = eco.formatEcosystemUpgradeSequence({ phases: { upgrade: { status: 'up-to-date', output: ['ya está al día'] }, sync: { executed: true, success: true, output: ['4 ficheros'] } } });
out.syncFailed = eco.formatEcosystemUpgradeSequence({ phases: { upgrade: { status: 'ok' }, sync: { executed: true, success: false, error: 'boom' } } });
out.unknownSkip = eco.formatEcosystemUpgradeSequence({ phases: { upgrade: {}, sync: { executed: false, skipped_reason: 'otro' } } });
out.outcome = [
  eco.operationOutcome('sync', { success: true }, false),
  eco.operationOutcome('sync', { success: false }, false),
  eco.operationOutcome('upgrade', { success: true, phases: { upgrade: { restart_required: true } } }, false),
  eco.operationOutcome('upgrade', {}, true),
  eco.operationOutcome('backup', { name: 'x' }, false),
];
out.labels = eco.OPERATION_LABEL;
console.log(JSON.stringify(out));
`
	if err := os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "run.mjs")
	cmd.Dir = dir
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node falló: %v\n%s", err, raw)
	}
	type view struct {
		Text string `json:"text"`
		Tone string `json:"tone"`
		Icon string `json:"icon"`
	}
	type summary struct {
		Total    int    `json:"total"`
		OK       int    `json:"ok"`
		Warnings int    `json:"warnings"`
		Errors   int    `json:"errors"`
		Other    int    `json:"other"`
		Tone     string `json:"tone"`
		Title    string `json:"title"`
	}
	type outcome struct {
		Tone string `json:"tone"`
		Text string `json:"text"`
	}
	var res struct {
		Status           []view            `json:"status"`
		Category         []string          `json:"category"`
		Order            []string          `json:"order"`
		OrderIsStable    []string          `json:"orderIsStable"`
		Empty            [][]any           `json:"empty"`
		SummaryOK        summary           `json:"summaryOk"`
		SummaryWarn      summary           `json:"summaryWarn"`
		SummaryUnhealthy summary           `json:"summaryUnhealthy"`
		SummaryErr       summary           `json:"summaryErr"`
		SummaryNone      summary           `json:"summaryNone"`
		Bytes            []string          `json:"bytes"`
		NoPhases         []string          `json:"noPhases"`
		Restart          string            `json:"restart"`
		Failed           string            `json:"failed"`
		Both             string            `json:"both"`
		SyncFailed       string            `json:"syncFailed"`
		UnknownSkip      string            `json:"unknownSkip"`
		Outcome          []outcome         `json:"outcome"`
		Labels           map[string]string `json:"labels"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, raw)
	}

	// Traducción al castellano: los valores de la API se mapean y lo desconocido sale tal cual.
	wantStatus := []view{{"Correcto", "ok", "check"}, {"Aviso", "warn", "alert"}, {"Error", "bad", "x"}, {"degraded", "neutral", "info"}, {"Desconocido", "neutral", "info"}, {"Desconocido", "neutral", "info"}}
	for i, w := range wantStatus {
		if res.Status[i] != w {
			t.Errorf("statusView #%d = %+v, se esperaba %+v", i, res.Status[i], w)
		}
	}
	wantCat := []string{"agente", "herramienta", "entorno", "cloud", ""}
	for i, w := range wantCat {
		if res.Category[i] != w {
			t.Errorf("categoryText #%d = %q, se esperaba %q", i, res.Category[i], w)
		}
	}

	// Errores primero, luego avisos, luego lo desconocido y al final lo correcto; estable.
	if got := strings.Join(res.Order, ","); got != "err-1,warn-1,warn-2,odd,ok-1,ok-2" {
		t.Errorf("sortChecks = %s", got)
	}
	if got := strings.Join(res.OrderIsStable, ","); got != "warn-1,warn-2" {
		t.Errorf("sortChecks no es estable: %s", got)
	}
	if len(res.Empty[0]) != 0 || len(res.Empty[1]) != 0 {
		t.Errorf("sortChecks sin lista debe devolver []: %+v", res.Empty)
	}

	// Tono y titular del banner de salud.
	if s := res.SummaryOK; s.Tone != "ok" || s.Title != "Ecosistema saludable" || s.OK != 2 || s.Total != 2 {
		t.Errorf("salud correcta = %+v", s)
	}
	if s := res.SummaryWarn; s.Tone != "warn" || s.Warnings != 1 || s.Errors != 0 {
		t.Errorf("salud con avisos = %+v", s)
	}
	if s := res.SummaryUnhealthy; s.Tone != "warn" {
		t.Errorf("healthy=false sin errores debe avisar, no ser correcto: %+v", s)
	}
	if s := res.SummaryErr; s.Tone != "bad" || s.Title != "Ecosistema con errores" || s.Errors != 1 || s.Warnings != 2 || s.Other != 1 || s.OK != 2 || s.Total != 6 {
		t.Errorf("salud con errores = %+v", s)
	}
	if s := res.SummaryNone; s.Total != 0 || s.Tone != "warn" {
		t.Errorf("sin informe el banner no puede ser «saludable»: %+v", s)
	}

	// Tamaños legibles; sin tamaño del servidor no se inventa ninguno.
	wantBytes := []string{"", "", "", "512 B", "2,0 KB", "1,2 MB", "1,0 GB"}
	for i, w := range wantBytes {
		if res.Bytes[i] != w {
			t.Errorf("formatBytes #%d = %q, se esperaba %q", i, res.Bytes[i], w)
		}
	}

	// Cadena upgrade -> sync: mismos literales que la interfaz anterior.
	if res.NoPhases[0] != "" || res.NoPhases[1] != "a\nb" || res.NoPhases[2] != "listo" {
		t.Errorf("sin fases: %q", res.NoPhases)
	}
	for _, want := range []string{"Fase upgrade: updated — binario axiom reemplazado", "Fase sync: omitida — reinicia axiom para continuar; después ejecuta sync"} {
		if !strings.Contains(res.Restart, want) {
			t.Errorf("restart-required: falta %q en %q", want, res.Restart)
		}
	}
	for _, want := range []string{"Actualización manual requerida: descarga el binario", "Error de upgrade: sin red", "Fase sync: omitida — la fase upgrade falló"} {
		if !strings.Contains(res.Failed, want) {
			t.Errorf("upgrade-failed: falta %q en %q", want, res.Failed)
		}
	}
	for _, want := range []string{"Fase upgrade: up-to-date", "ya está al día", "Fase sync: ejecutada con éxito", "4 ficheros"} {
		if !strings.Contains(res.Both, want) {
			t.Errorf("ambas fases: falta %q en %q", want, res.Both)
		}
	}
	if !strings.Contains(res.SyncFailed, "Fase sync: ejecutada con errores") || !strings.Contains(res.SyncFailed, "Error de sync: boom") {
		t.Errorf("sync fallida: %q", res.SyncFailed)
	}
	if !strings.Contains(res.UnknownSkip, "Fase upgrade: desconocido") || !strings.Contains(res.UnknownSkip, "Fase sync: omitida (otro)") {
		t.Errorf("motivo de omisión desconocido: %q", res.UnknownSkip)
	}

	// Estado de «Última operación».
	wantOutcome := []outcome{{"ok", "correcta"}, {"bad", "con errores"}, {"warn", "reinicio pendiente"}, {"bad", "con errores"}, {"ok", "correcta"}}
	for i, w := range wantOutcome {
		if res.Outcome[i] != w {
			t.Errorf("operationOutcome #%d = %+v, se esperaba %+v", i, res.Outcome[i], w)
		}
	}
	if res.Labels["upgrade"] != "upgrade → sync" {
		t.Errorf("la etiqueta del upgrade debe mostrar la cadena upgrade → sync: %q", res.Labels["upgrade"])
	}
}
