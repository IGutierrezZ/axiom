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

// Las pruebas de este fichero cubren la pantalla de Incrementos y el detalle de
// incremento (T3). No hay motor de JavaScript en `go test`, así que son
// estructurales: comprueban que los recursos se sirven y que contienen las
// piezas que dan el comportamiento (rutas, política D5, accesibilidad, escape).
// El renderizador de Markdown, además, se ejecuta de verdad si hay Node.

func TestWorkflowAssetsAreServed(t *testing.T) {
	h := assetServer(t)
	for _, target := range []string{
		"/js/core/markdown.js", "/js/core/increment.js",
		"/js/screens/incrementos.js", "/js/screens/detalle.js",
		"/js/screens/detalle-roles.js", "/js/screens/detalle-handoff.js",
	} {
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

// El tablero cubre las siete fases, pliega las columnas vacías y recuerda la vista y el filtro.
func TestIncrementsScreenBoardListAndFilters(t *testing.T) {
	inc := readAsset(t, "js/screens/incrementos.js")
	for _, marker := range []string{
		"ix-board", "ix-col", "is-collapsed", "ALWAYS_OPEN", "Ver los ${archivedAll.length} archivados",
		"listHtml(", "boardHtml(", "tableWrap(", "miniStepper(",
		"data-view", "aria-pressed", "data-filter", "data-role",
		"'axiom.inc.view'", "'axiom.inc.filter'", "'axiom.inc.role'", "window.localStorage",
		"state.workspace.roles", "restoreFocus", "stateEmpty(", "stateError(",
	} {
		if !strings.Contains(inc, marker) {
			t.Errorf("incrementos.js no contiene %q", marker)
		}
	}
	// Todo acceso a localStorage va en try/catch.
	if got, want := strings.Count(inc, "window.localStorage"), strings.Count(inc, "try {"); got > want {
		t.Errorf("incrementos.js: %d accesos a localStorage para %d bloques try", got, want)
	}
	// Los ocho filtros pedidos.
	for _, id := range []string{"'all'", "'bug'", "'spec'", "'design'", "'waiting'", "'gverify'", "'archive'", "'archived'"} {
		if !strings.Contains(inc, "id: "+id) {
			t.Errorf("incrementos.js no define el filtro %s", id)
		}
	}

	shared := readAsset(t, "js/core/increment.js")
	for _, phase := range []string{"explore", "spec", "design", "tasks", "apply", "verify", "archive"} {
		if !strings.Contains(shared, "id: '"+phase+"'") {
			t.Errorf("increment.js no define la fase %q", phase)
		}
	}
	// Prioridad de la bandera única, de mayor a menor.
	order := []string{"ready_for_global_verify", "waiting_roles", "ready_for_archive", "pending_spec", "ready_for_design"}
	body := shared[strings.Index(shared, "export function statusFlag"):]
	last := -1
	for _, flag := range order {
		idx := strings.Index(body, "inc."+flag)
		if idx < 0 || idx < last {
			t.Errorf("increment.js: la bandera %q falta o rompe la prioridad", flag)
		}
		last = idx
	}
	// El tipo lleva texto además de icono, y no se inventan edades relativas.
	for _, marker := range []string{"'Bug'", "'Feature'", "'Arquitectura'", "'Refactor'", "type-chip", "role=\"progressbar\"", "is-done"} {
		if !strings.Contains(shared, marker) {
			t.Errorf("increment.js no contiene %q", marker)
		}
	}
	if strings.Contains(inc+shared, "hace ") {
		t.Errorf("las tarjetas no deben mostrar edades relativas inventadas («hace …»)")
	}

	html := readAsset(t, "index.html")
	for _, marker := range []string{`id="inc-view-toggle"`, `data-view="board"`, `data-view="list"`, `id="inc-filters"`, `id="inc-role-filters"`, `id="inc-live"`, `aria-live="polite"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}
}

// Las rutas del detalle, sus pestañas y su sincronización con el hash.
func TestIncrementDetailTabsAndRoutes(t *testing.T) {
	html := readAsset(t, "index.html")
	for _, tab := range []string{"artefactos", "roles", "handoff"} {
		if !strings.Contains(html, `data-tab="`+tab+`"`) {
			t.Errorf("index.html no contiene la pestaña %q", tab)
		}
		if !strings.Contains(html, `data-panel="`+tab+`"`) {
			t.Errorf("index.html no contiene el panel %q", tab)
		}
	}
	for _, marker := range []string{`aria-label="Secciones del incremento"`, `id="inc-stepper"`, `id="inc-next-body"`, `id="inc-console"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}

	detail := readAsset(t, "js/screens/detalle.js")
	for _, marker := range []string{
		"aria-current", "hrefFor('incremento'", "data-artifact", "data-format", "aria-pressed",
		"tasks_content", "verify_report", "archive_report", "no existe todavía",
		"sectionStats", "withFocus", "ctx.token",
		"/api/increments/continue", "renderConsole(", "aria-current=\"step\"",
	} {
		if !strings.Contains(detail, marker) {
			t.Errorf("detalle.js no contiene %q", marker)
		}
	}
	roles := readAsset(t, "js/screens/detalle-roles.js")
	for _, marker := range []string{
		"barrier_report", "gate_policy", "deferred_tasks", "data-migrate", "'deferred'", "confirmDialog(",
		"/api/increments/migrate-cumulative", "Rol unificado (fullstack)", "Barrera superada", "Barrera bloqueada", "Bloqueos",
	} {
		if !strings.Contains(roles, marker) {
			t.Errorf("detalle-roles.js no contiene %q", marker)
		}
	}
	// «Migrar» solo existe para roles diferidos.
	if !regexp.MustCompile(`gate_policy === 'deferred'\s*\?`).MatchString(roles) {
		t.Errorf("detalle-roles.js debe condicionar el botón Migrar a gate_policy === 'deferred'")
	}
	handoff := readAsset(t, "js/screens/detalle-handoff.js")
	for _, marker := range []string{"/api/handoffs", "renderMarkdown(", "modal-create-handoff", "executive_summary", "direct_instructions", "err.status === 404"} {
		if !strings.Contains(handoff, marker) {
			t.Errorf("detalle-handoff.js no contiene %q", marker)
		}
	}

	// Las pestañas del detalle ya no existen como pantallas globales.
	for _, gone := range []string{"btn-inc-verify", "btn-inc-continue", "Validar verificación", "roles-container", "ho-sections-container"} {
		if strings.Contains(html, gone) {
			t.Errorf("index.html todavía contiene %q", gone)
		}
	}
}

// D5: lo que exige una fase de agente es un Recordatorio con su orden, nunca un
// botón; y las órdenes citadas existen de verdad en el repositorio.
func TestAgentPhasesAreRemindersNotButtons(t *testing.T) {
	all := readAllJS(t)
	if strings.Contains(all, "/api/increments/verify") {
		t.Errorf("la interfaz ya no debe llamar a /api/increments/verify (D5: verificar es un recordatorio)")
	}

	shared := readAsset(t, "js/core/increment.js")
	detail := readAsset(t, "js/screens/detalle.js")
	if !strings.Contains(detail, "reminder(next.reminder)") {
		t.Errorf("detalle.js debe pintar la fase de agente con el componente reminder()")
	}
	// Continuar es el único botón de acción del «Siguiente paso».
	if got := strings.Count(detail, "label: 'Continuar con Axiom'"); got != 1 {
		t.Errorf("detalle.js debe tener un único botón Continuar, hay %d", got)
	}
	for _, cmd := range []string{"/sdd-continue", "/sdd-apply", "/sdd-verify", "/sdd-archive"} {
		if !strings.Contains(shared, cmd+" ${name}") {
			t.Errorf("increment.js no cita la orden %s <cambio>", cmd)
			continue
		}
		file := filepath.Join("..", "assets", "claude", "commands", strings.TrimPrefix(cmd, "/")+".md")
		if _, err := os.Stat(file); err != nil {
			t.Errorf("la orden %s no existe en el repositorio (%s): %v", cmd, file, err)
		}
	}
	for _, text := range []string{"IDE o CLI favorito", "No ejecuta fases de agente", "Recordatorio"} {
		if !strings.Contains(shared+detail, text) {
			t.Errorf("falta el texto %q del Recordatorio", text)
		}
	}
	// El texto del recordatorio no puede estar dentro de un <button> de ejecución.
	if strings.Contains(shared, "data-action=\"verify\"") || strings.Contains(shared, "data-action=\"archive\"") {
		t.Errorf("verify y archive no pueden ser botones de ejecución")
	}
}

// Las pantallas nuevas no dependen de las clases de legacy.css que sustituyen.
func TestIncrementScreensDoNotUseLegacyStyles(t *testing.T) {
	legacy := []string{"inc-card", "increments-grid", "role-card", "roles-grid", "barrier-banner", "handoff-card", "section-box", "filter-btn", "badge-feature", "badge-bug", "loading-state"}
	sources := map[string]string{"index.html (incrementos y detalle)": "", "incrementos.js": readAsset(t, "js/screens/incrementos.js"), "detalle.js": readAsset(t, "js/screens/detalle.js"),
		"detalle-roles.js": readAsset(t, "js/screens/detalle-roles.js"), "detalle-handoff.js": readAsset(t, "js/screens/detalle-handoff.js")}
	html := readAsset(t, "index.html")
	start, end := strings.Index(html, `id="view-incrementos"`), strings.Index(html, `id="view-specs"`)
	if start < 0 || end < start {
		t.Fatalf("index.html: no se localizan las vistas de incrementos")
	}
	sources["index.html (incrementos y detalle)"] = html[start:end]
	for name, src := range sources {
		for _, class := range legacy {
			if strings.Contains(src, class) {
				t.Errorf("%s usa la clase heredada %q", name, class)
			}
		}
	}

	css := readAsset(t, "style.css")
	for _, marker := range []string{".ix-board", ".ix-col.is-collapsed", ".mini-stepper", ".type-chip", ".dt-steps", ".dt-banner", ".dt-role", ".md-tasks", ".md-code", ".tab-pill", "@media (max-width: 640px)"} {
		if !strings.Contains(css, marker) {
			t.Errorf("style.css no contiene %q", marker)
		}
	}
	// Solo tokens: ningún color fijo en la hoja de estilos de la interfaz.
	if loc := regexp.MustCompile(`#[0-9A-Fa-f]{3,8}\b`).FindString(css); loc != "" {
		t.Errorf("style.css contiene el color fijo %q; usa los tokens de tokens.css", loc)
	}
}

// El renderizador de Markdown no inyecta HTML: lo escapa todo y solo enlaza http(s).
func TestMarkdownRendererIsSafeStructurally(t *testing.T) {
	md := readAsset(t, "js/core/markdown.js")
	for _, marker := range []string{"escapeHtml", "safeHref", "rel=\"noopener noreferrer\"", `(https?:\/\/|mailto:)`, "export function renderMarkdown(", "export function firstParagraph("} {
		if !strings.Contains(md, marker) {
			t.Errorf("markdown.js no contiene %q", marker)
		}
	}
	for _, banned := range []string{"innerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function"} {
		if strings.Contains(md, banned) {
			t.Errorf("markdown.js no debe usar %s", banned)
		}
	}
}

// Si hay Node, el renderizador se ejecuta de verdad sobre entradas hostiles.
func TestMarkdownRendererRuntime(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite la prueba de ejecución del renderizador")
	}
	dir := t.TempDir()
	for _, name := range []string{"api", "icons", "markdown"} {
		src := readAsset(t, "js/core/"+name+".js")
		src = regexp.MustCompile(`from '\./([a-z]+)\.js'`).ReplaceAllString(src, "from './$1.mjs'")
		if err := os.WriteFile(filepath.Join(dir, name+".mjs"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := `import { renderMarkdown, firstParagraph } from './markdown.mjs';
const hostile = [
  '# T <script>alert(1)</script>',
  '',
  'Texto <img src=x onerror=alert(1)> y [x](javascript:alert(1)) y [ok](https://example.org/a) y ![i](http://e.org/i.png)',
  '',
  '- [x] hecha',
  '- [ ] pendiente',
  '',
  '| a | b |',
  '|---|---|',
  '| <b>1</b> | 2 |',
].join('\n');
console.log(JSON.stringify({
  html: renderMarkdown(hostile, { sectionStats: true }),
  first: firstParagraph('# T\n\nPrimer **párrafo** con ` + "`código`" + `.\n\nSegundo.'),
}));
`
	if err := os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "run.mjs")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node falló: %v\n%s", err, out)
	}
	var res struct {
		HTML  string `json:"html"`
		First string `json:"first"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	for _, bad := range []string{"<script", "<img", "javascript:", `<a href="http://e.org`, "<b>"} {
		if strings.Contains(res.HTML, bad) {
			t.Errorf("el HTML renderizado contiene %q sin escapar: %s", bad, res.HTML)
		}
	}
	for _, good := range []string{"&lt;script&gt;", "&lt;img", `<a href="https://example.org/a" target="_blank" rel="noopener noreferrer">ok</a>`, `aria-label="Hecha"`, `aria-label="Pendiente"`, "1/2 hechas", "&lt;b&gt;1&lt;/b&gt;"} {
		if !strings.Contains(res.HTML, good) {
			t.Errorf("el HTML renderizado no contiene %q: %s", good, res.HTML)
		}
	}
	if res.First != "Primer párrafo con código." {
		t.Errorf("firstParagraph = %q", res.First)
	}
}
