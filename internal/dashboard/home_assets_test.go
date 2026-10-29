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

// Las pruebas de este fichero cubren Inicio y las insignias de la barra lateral
// (T4). Como no hay motor de JavaScript en `go test`, las estructurales miran
// las piezas que dan el comportamiento y, si hay Node, la regla de atención, los
// indicadores y la caché con deduplicación se ejecutan de verdad.

func TestHomeAssetsAreServed(t *testing.T) {
	h := assetServer(t)
	for _, target := range []string{"/js/core/overview.js", "/js/screens/inicio.js", "/js/core/increment.js"} {
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

// La regla «necesita atención» se define una sola vez, en increment.js, y Inicio
// la consume sin repetir las banderas del resumen.
func TestHomeAttentionRuleIsDefinedOnce(t *testing.T) {
	shared := readAsset(t, "js/core/increment.js")
	if got := strings.Count(shared, "export function needsAttention("); got != 1 {
		t.Errorf("increment.js define needsAttention %d veces, se esperaba 1", got)
	}
	if !strings.Contains(shared, "statusFlag(inc) !== null") {
		t.Errorf("needsAttention debe derivarse de statusFlag para no duplicar las banderas")
	}
	// Prioridad de la lista: verificación global, roles, archivado, y spec/design al final.
	rank := regexp.MustCompile(`ATTENTION_RANK = \{ gverify: 0, waiting: 1, archive: 2, spec: 3, design: 3 \}`)
	if !rank.MatchString(shared) {
		t.Errorf("increment.js no declara la prioridad gverify > waiting > archive > spec/design")
	}

	home := readAsset(t, "js/screens/inicio.js")
	for _, flag := range []string{"ready_for_global_verify", "waiting_roles", "ready_for_archive", "pending_spec", "ready_for_design"} {
		if strings.Contains(home, flag) {
			t.Errorf("inicio.js repite la bandera %q: la regla vive en increment.js", flag)
		}
	}
	for _, marker := range []string{"attentionItems(", "incrementStats(", "deriveNextStep(", "doctorSummary("} {
		if !strings.Contains(home, marker) {
			t.Errorf("inicio.js no usa %q", marker)
		}
	}
}

// D3: ni actividad reciente ni «Ver como» ni edades relativas inventadas.
func TestHomeOmitsRecentActivityAndLens(t *testing.T) {
	html := readAsset(t, "index.html")
	start, end := strings.Index(html, `id="view-inicio"`), strings.Index(html, `id="view-incrementos"`)
	if start < 0 || end < start {
		t.Fatalf("index.html: no se localiza la vista de Inicio")
	}
	sources := map[string]string{
		"index.html (Inicio)": html[start:end],
		"inicio.js":           readAsset(t, "js/screens/inicio.js"),
	}
	for name, src := range sources {
		low := strings.ToLower(src)
		for _, banned := range []string{"actividad reciente", "ver como", "hace 2 h", "hace unos"} {
			if strings.Contains(low, banned) {
				t.Errorf("%s contiene %q, bloque omitido por D3", name, banned)
			}
		}
	}
}

// Cada bloque pinta su carga, su vacío y su error; un fallo no vacía el resto.
func TestHomeBlocksHaveLoadingEmptyAndErrorStates(t *testing.T) {
	home := readAsset(t, "js/screens/inicio.js")
	for _, marker := range []string{
		"stateLoading(", "stateError(", "stateEmpty(", "data-action=\"retry\"",
		"source('increments')", "source('doctor')", "source('sync')", "check('inbox'", "check('semantic'", "check('specs'",
		"incrementos necesitan", "Nada necesita atención", "No se pudo consultar",
		"home-attention", "home-flow", "home-kpis", "home-help", "home-sync",
		"flow-list", "flow-seg", "is-bug", "sr-only", "aria-label=\"Incrementos activos por fase SDD\"",
		"href: '#/skills'", "href: '#/codigo'", "href: '#/specs'", "href: '#/ecosistema'",
		"card.hidden = rows.length === 0 && waiting === 0",
	} {
		if !strings.Contains(home, marker) {
			t.Errorf("inicio.js no contiene %q", marker)
		}
	}
	// Los cuatro indicadores enlazan a su pantalla y llevan texto.
	if got := strings.Count(home, "href: '#/incrementos'"); got < 3 {
		t.Errorf("los indicadores de incrementos deben enlazar a #/incrementos (%d enlaces)", got)
	}
	if !strings.Contains(home, "href: '#/ecosistema'") {
		t.Errorf("el indicador de salud debe enlazar a #/ecosistema")
	}
	// Todo dato de la API entra escapado y no hay inserciones sin escapar de campos crudos.
	if !strings.Contains(home, "escapeHtml(") {
		t.Errorf("inicio.js debe escapar los datos de la API")
	}
	for _, banned := range []string{"eval(", "new Function", "document.write", "insertAdjacentHTML"} {
		if strings.Contains(home, banned) {
			t.Errorf("inicio.js no debe usar %s", banned)
		}
	}
}

// D5 en Inicio: Continuar es botón; las fases de agente salen como Recordatorio.
func TestHomeActionsFollowD5(t *testing.T) {
	home := readAsset(t, "js/screens/inicio.js")
	if !strings.Contains(home, "reminder({ ...next.reminder, compact: true })") {
		t.Errorf("las fases de agente deben pintarse con el componente reminder() en modo compacto")
	}
	if !strings.Contains(home, "continueIncrement(") || !strings.Contains(home, "data-action=\"continue\"") {
		t.Errorf("Continuar debe usar el helper compartido continueIncrement")
	}
	if !strings.Contains(home, "tab: 'roles'") {
		t.Errorf("la fila en espera de roles debe enlazar a la barrera (#/incrementos/<name>/roles)")
	}
	for _, banned := range []string{`data-action="verify"`, `data-action="archive"`, "/api/increments/verify", "/api/increments/continue"} {
		if strings.Contains(home, banned) {
			t.Errorf("inicio.js no debe contener %q (D5: verify y archive son recordatorios; la llamada vive en increment.js)", banned)
		}
	}
	comp := readAsset(t, "js/core/components.js")
	if !strings.Contains(comp, "compact = false") || !strings.Contains(comp, "reminder-compact") {
		t.Errorf("components.js debe ofrecer reminder() compacto")
	}
	// El pull reutiliza la lógica de la barra lateral.
	if !strings.Contains(home, "pullSpecs(") || !strings.Contains(readAsset(t, "js/shell.js"), "export async function pullSpecs(") {
		t.Errorf("el pull de Inicio debe reutilizar pullSpecs de shell.js")
	}
}

// Las insignias de la barra lateral llevan texto accesible y no se refrescan al navegar.
func TestSidebarBadgesAreAccessibleAndCached(t *testing.T) {
	html := readAsset(t, "index.html")
	for _, id := range []string{`id="badge-incrementos"`, `id="badge-skills"`, `id="badge-ecosistema"`} {
		if !strings.Contains(html, id) {
			t.Errorf("index.html no contiene %s", id)
		}
	}
	shell := readAsset(t, "js/shell.js")
	for _, marker := range []string{
		"renderNavBadges", "class=\"sr-only\"", "aria-hidden=\"true\"", "'incremento activo'", "'propuesta pendiente'",
		"'aviso'", "nav-dot", "on('overview', renderNavBadges)", "document.hidden", "POLLED_SOURCES",
	} {
		if !strings.Contains(shell, marker) {
			t.Errorf("shell.js no contiene %q", marker)
		}
	}

	// Navegar no consulta: el manejador de rutas no toca la caché.
	mainJS := readAsset(t, "js/main.js")
	show := mainJS[strings.Index(mainJS, "function showView("):strings.Index(mainJS, "function boot(")]
	for _, banned := range []string{"refreshOverview", "loadAllData", "fetch(", "api("} {
		if strings.Contains(show, banned) {
			t.Errorf("showView no debe llamar a %s: las insignias no se refrescan al cambiar de ruta", banned)
		}
	}
	if !strings.Contains(mainJS, "refreshOverview()") {
		t.Errorf("Actualizar (loadAllData) debe refrescar la caché de Inicio")
	}
	if strings.Contains(shell, "setInterval(checkSpecsSync") {
		t.Errorf("el sondeo de 60 s debe refrescar la caché completa, no solo el aviso de specs")
	}
}

// La caché reparte las fuentes y api() comparte los GET simultáneos.
func TestHomeSharedCacheAndDedupe(t *testing.T) {
	over := readAsset(t, "js/core/overview.js")
	for _, route := range []string{
		"/api/increments", "/api/ecosystem/doctor", "/api/workspace/specs/sync-status",
		"/api/skills/inbox", "/api/semantic/status", "/api/archive/specs",
	} {
		if !strings.Contains(over, "'"+route+"'") {
			t.Errorf("overview.js no declara la fuente %s", route)
		}
	}
	if !strings.Contains(over, "export const POLLED_SOURCES") || !strings.Contains(over, "export function refreshOverview(") {
		t.Errorf("overview.js debe exportar POLLED_SOURCES y refreshOverview")
	}
	apiJS := readAsset(t, "js/core/api.js")
	for _, marker := range []string{"const inflight = new Map()", "inflight.delete(path)", "inflight.set(path, pending)"} {
		if !strings.Contains(apiJS, marker) {
			t.Errorf("api.js no contiene %q", marker)
		}
	}
	// Inicio no hace fetch propio de las fuentes cacheadas.
	home := readAsset(t, "js/screens/inicio.js")
	if loc := regexp.MustCompile(`api\('/api/(increments|ecosystem|workspace|skills|semantic|archive)`).FindString(home); loc != "" {
		t.Errorf("inicio.js consulta %q directamente; debe usar la caché de overview.js", loc)
	}
}

// El CSS nuevo solo usa tokens y define los estados accesibles de Inicio.
func TestHomeStylesUseTokensAndKeepStatesVisible(t *testing.T) {
	css := readAsset(t, "style.css")
	for _, marker := range []string{
		".kpi-link", ".home-grid", ".home-main", ".att-row", ".reminder-compact", ".flow-list", ".flow-seg.is-bug",
		".flow-legend", ".help-row", ".nav-dot.is-bad", "repeating-linear-gradient",
	} {
		if !strings.Contains(css, marker) {
			t.Errorf("style.css no contiene %q", marker)
		}
	}
	if loc := regexp.MustCompile(`#[0-9A-Fa-f]{3,8}\b`).FindString(css); loc != "" {
		t.Errorf("style.css contiene el color fijo %q; usa los tokens de tokens.css", loc)
	}
	// Los bugs del flujo se distinguen por trama y por texto, no solo por color.
	if !strings.Contains(readAsset(t, "js/screens/inicio.js"), "flow-bug") {
		t.Errorf("el flujo SDD debe nombrar los bugs con texto")
	}
}

// Con Node se ejecutan de verdad: la regla de atención y su prioridad, los
// indicadores, el resumen del doctor, la caché por fuente y la deduplicación.
func TestHomeLogicRuntime(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite la prueba de ejecución de la lógica de Inicio")
	}
	dir := t.TempDir()
	for _, name := range []string{"api", "icons", "increment", "state", "overview"} {
		src := readAsset(t, "js/core/"+name+".js")
		src = regexp.MustCompile(`from '\./([a-z]+)\.js'`).ReplaceAllString(src, "from './$1.mjs'")
		if err := os.WriteFile(filepath.Join(dir, name+".mjs"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := `import { attentionItems, needsAttention, incrementStats } from './increment.mjs';
import { api } from './api.mjs';
import { state } from './state.mjs';
import { doctorSummary, refreshOverview, sourceData } from './overview.mjs';
console.debug = () => {};

const incs = [
  { name: 'z-spec', type: 'active', phase: 'spec', pending_spec: true },
  { name: 'a-design', type: 'active', phase: 'design', ready_for_design: true },
  { name: 'arch', type: 'archived', phase: 'archive', ready_for_archive: true },
  { name: 'wait', type: 'active', phase: 'apply', waiting_roles: true, pending_roles: ['fe'], tasks_total: 4, tasks_completed: 2 },
  { name: 'gv', type: 'active', phase: 'verify', ready_for_global_verify: true, is_bug: true, change_type: 'fix', tasks_total: 6, tasks_completed: 6 },
  { name: 'ar', type: 'active', phase: 'verify', ready_for_archive: true, tasks_total: 2, tasks_completed: 2 },
  { name: 'quiet', type: 'active', phase: 'apply', tasks_total: 3, tasks_completed: 1 },
];
const routes = {
  '/api/increments': JSON.stringify(incs),
  '/api/skills/inbox': 'null',
  '/api/ecosystem/doctor': JSON.stringify({ healthy: false, checks: [{ name: 'a', status: 'ok' }, { name: 'b', status: 'warning' }, { name: 'c', status: 'error' }] }),
  '/api/workspace/specs/sync-status': JSON.stringify({ is_git_repo: true, behind: 2 }),
  '/api/archive/specs': JSON.stringify({ specs: [] }),
};
const calls = [];
globalThis.fetch = async (path) => {
  calls.push(path);
  await new Promise(r => setTimeout(r, 5));
  const body = routes[path];
  if (body === undefined) return { ok: false, status: 500, statusText: 'x', text: async () => JSON.stringify({ error: 'boom' }) };
  return { ok: true, status: 200, text: async () => body };
};

const out = {};
out.order = attentionItems(incs).map(i => i.inc.name);
out.needs = { quiet: needsAttention(incs[6]), arch: needsAttention(incs[2]), gv: needsAttention(incs[4]) };
out.stats = incrementStats(incs);
out.doctor = doctorSummary(JSON.parse(routes['/api/ecosystem/doctor']));

await Promise.all([api('/api/increments'), api('/api/increments'), refreshOverview(['increments'])]);
out.simultaneous = calls.filter(c => c === '/api/increments').length;
await api('/api/increments');
out.sequential = calls.filter(c => c === '/api/increments').length;

calls.length = 0;
await refreshOverview();
out.full = calls.length;
out.overview = Object.fromEntries(Object.entries(state.overview).map(([k, v]) => [k, v.status]));
out.inbox = sourceData('inbox');
out.semanticError = state.overview.semantic.error;
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
	var res struct {
		Order []string        `json:"order"`
		Needs map[string]bool `json:"needs"`
		Stats struct {
			Active          int `json:"active"`
			Bugs            int `json:"bugs"`
			Archived        int `json:"archived"`
			TasksDone       int `json:"tasksDone"`
			TasksTotal      int `json:"tasksTotal"`
			Pct             int `json:"pct"`
			Verifying       int `json:"verifying"`
			ReadyForArchive int `json:"readyForArchive"`
			ByPhase         []struct {
				ID    string `json:"id"`
				Count int    `json:"count"`
				Bugs  int    `json:"bugs"`
			} `json:"byPhase"`
		} `json:"stats"`
		Doctor struct {
			NotOk    int `json:"notOk"`
			Errors   int `json:"errors"`
			Warnings int `json:"warnings"`
		} `json:"doctor"`
		Simultaneous  int               `json:"simultaneous"`
		Sequential    int               `json:"sequential"`
		Full          int               `json:"full"`
		Overview      map[string]string `json:"overview"`
		Inbox         []any             `json:"inbox"`
		SemanticError string            `json:"semanticError"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, raw)
	}

	// Prioridad: verificación global, roles, archivado, y spec/design al final por nombre.
	if want := "gv,wait,ar,a-design,z-spec"; strings.Join(res.Order, ",") != want {
		t.Errorf("orden de atención = %v, se esperaba %s", res.Order, want)
	}
	if res.Needs["quiet"] || res.Needs["arch"] || !res.Needs["gv"] {
		t.Errorf("needsAttention = %v: un incremento sin bandera o archivado no necesita atención", res.Needs)
	}
	s := res.Stats
	if s.Active != 6 || s.Bugs != 1 || s.Archived != 1 || s.TasksDone != 11 || s.TasksTotal != 15 || s.Pct != 73 || s.Verifying != 2 || s.ReadyForArchive != 1 {
		t.Errorf("indicadores inesperados: %+v", s)
	}
	counts := map[string]int{}
	for _, p := range s.ByPhase {
		counts[p.ID] = p.Count
		if p.ID == "verify" && p.Bugs != 1 {
			t.Errorf("la fase verify debe contar 1 bug, cuenta %d", p.Bugs)
		}
	}
	if counts["spec"] != 1 || counts["design"] != 1 || counts["apply"] != 2 || counts["verify"] != 2 || counts["explore"] != 0 {
		t.Errorf("conteo por fase inesperado: %v", counts)
	}
	if res.Doctor.NotOk != 2 || res.Doctor.Errors != 1 || res.Doctor.Warnings != 1 {
		t.Errorf("resumen del doctor inesperado: %+v", res.Doctor)
	}

	// Una sola petición por ruta mientras haya otra en vuelo, y nueva consulta después.
	if res.Simultaneous != 1 {
		t.Errorf("3 consultas simultáneas a /api/increments hicieron %d peticiones, se esperaba 1", res.Simultaneous)
	}
	if res.Sequential != 2 {
		t.Errorf("tras terminar, una consulta nueva debe repetir la petición (total %d, se esperaba 2)", res.Sequential)
	}
	if res.Full != 6 {
		t.Errorf("refreshOverview() hizo %d peticiones, se esperaban 6 (una por fuente)", res.Full)
	}
	// Una fuente que falla no afecta a las demás; null de una lista pasa a [].
	if res.Overview["semantic"] != "error" || res.SemanticError != "boom" {
		t.Errorf("la fuente semantic debe quedar en error con su mensaje: %v / %q", res.Overview["semantic"], res.SemanticError)
	}
	for _, key := range []string{"increments", "doctor", "sync", "inbox", "specs"} {
		if res.Overview[key] != "ready" {
			t.Errorf("la fuente %s debe quedar lista aunque otra falle (estado %q)", key, res.Overview[key])
		}
	}
	if res.Inbox == nil || len(res.Inbox) != 0 {
		t.Errorf("el buzón null debe normalizarse a una lista vacía, es %v", res.Inbox)
	}
}
