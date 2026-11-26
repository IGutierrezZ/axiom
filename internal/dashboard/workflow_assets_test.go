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
	t.Parallel()
	h := assetServer(t)
	for _, target := range []string{
		"/js/core/markdown.js", "/js/core/increment.js",
		"/js/screens/incrementos.js", "/js/screens/detalle.js",
		"/js/screens/detalle-roles.js", "/js/screens/detalle-handoff.js", "/js/screens/detalle-explicacion.js",
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
	t.Parallel()
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
	// Los nueve filtros (los ocho de SDD y el de las features ODD).
	for _, id := range []string{"'all'", "'bug'", "'odd'", "'spec'", "'design'", "'waiting'", "'gverify'", "'archive'", "'archived'"} {
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
	for _, marker := range []string{"'Bug'", "'Feature'", "'Arquitectura'", "'Refactor'", "'ODD'", "type-chip", "role=\"progressbar\"", "is-done"} {
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
	t.Parallel()
	html := readAsset(t, "index.html")
	for _, tab := range []string{"artefactos", "roles", "handoff", "explicacion"} {
		if !strings.Contains(html, `data-tab="`+tab+`"`) {
			t.Errorf("index.html no contiene la pestaña %q", tab)
		}
		if !strings.Contains(html, `data-panel="`+tab+`"`) {
			t.Errorf("index.html no contiene el panel %q", tab)
		}
	}
	for _, marker := range []string{`aria-label="Secciones del incremento"`, `id="inc-stepper"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}
	// La tarjeta «Siguiente paso» y su consola se retiraron: el stepper y la orden de la cabecera bastan.
	for _, gone := range []string{`id="inc-next-body"`, `id="inc-console"`, "Siguiente paso"} {
		if strings.Contains(html, gone) {
			t.Errorf("index.html todavía contiene %q", gone)
		}
	}

	detail := readAsset(t, "js/screens/detalle.js")
	for _, marker := range []string{
		"aria-current", "hrefFor('incremento'", "data-artifact", "data-format", "aria-pressed",
		"detail.specs", "detail.tasks", "CSS.escape", "verify_report", "archive_report", "no existe todavía",
		"sectionStats", "withFocus", "ctx.token",
		"commandChip(nextCommand(s))", "aria-current=\"step\"",
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

// Las fases de agente se muestran como una orden copiable, nunca como un botón que
// las ejecute; y las órdenes citadas existen de verdad en el repositorio.
func TestAgentPhasesAreCopyableCommandsNotButtons(t *testing.T) {
	t.Parallel()
	all := readAllJS(t)
	for _, banned := range []string{"/api/increments/verify", "/api/increments/continue", "continueIncrement", "Continuar con Axiom"} {
		if strings.Contains(all, banned) {
			t.Errorf("la interfaz ya no debe contener %q: el dashboard no ejecuta fases ni el despachador", banned)
		}
	}

	shared := readAsset(t, "js/core/increment.js")
	if !strings.Contains(shared, "export function nextCommand(") {
		t.Errorf("increment.js debe exportar nextCommand, la única fuente de la orden de agente")
	}
	comp := readAsset(t, "js/core/components.js")
	if !strings.Contains(comp, "export function commandChip(") || !strings.Contains(comp, "data-copy=") {
		t.Errorf("components.js debe ofrecer commandChip con data-copy (lo atiende bindCopyButtons)")
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
	// El chip explica que la orden se lanza fuera del dashboard.
	if !strings.Contains(comp, "IDE o CLI") {
		t.Errorf("commandChip debe decir que la orden se lanza desde el IDE o la CLI")
	}
	if strings.Contains(shared, "data-action=\"verify\"") || strings.Contains(shared, "data-action=\"archive\"") {
		t.Errorf("verify y archive no pueden ser botones de ejecución")
	}
}

// Multirol: el paso Apply en curso desglosa cada rol de la barrera y enlaza a su
// pestaña; con un único rol (fullstack) el paso queda como estaba.
func TestApplyStepShowsPerRoleProgressWhenMultirole(t *testing.T) {
	t.Parallel()
	shared := readAsset(t, "js/core/increment.js")
	for _, marker := range []string{
		"apply.state === 'current' && roles.length > 1", "function roleProgress(", "barrier.satisfied === false",
		"r.verify_done", "r.tasks_found", "'verificado'", "'sin tasks'",
	} {
		if !strings.Contains(shared, marker) {
			t.Errorf("increment.js no contiene %q", marker)
		}
	}
	detail := readAsset(t, "js/screens/detalle.js")
	for _, marker := range []string{
		"function stepRoles(", "stepRoles(st, change)", `aria-label="Avance por rol"`, "is-waiting",
		"tab: 'roles'", "Barrera cerrada", "progressBar(r.pct", "escapeHtml(r.label)", "escapeHtml(st.blocker)",
	} {
		if !strings.Contains(detail, marker) {
			t.Errorf("detalle.js no contiene %q", marker)
		}
	}
	css := readAsset(t, "style.css")
	// Clases propias del paso: .dt-role ya es la tarjeta de la pestaña «Roles y barrera».
	for _, marker := range []string{".dt-step-roles", ".dt-step-role ", ".dt-step.is-waiting .dt-step-dot", ".dt-step-role-link"} {
		if !strings.Contains(css, marker) {
			t.Errorf("style.css no contiene %q", marker)
		}
	}
}

// La hoja de estilos de incrementos define sus componentes y solo usa tokens.
func TestIncrementStylesDefineTheirComponentsWithTokens(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// Las features ODD llevan su propio chip «ODD», no enlazan a un detalle SDD y no cuentan en el flujo
// SDD de Inicio; el subtítulo del tablero ya no habla solo de SDD.
func TestIncrementsODDChipAndHomeStatsRuntime(t *testing.T) {
	t.Parallel()
	out := runNodeModules(t, []string{"api", "icons", "increment"},
		"import { typeInfo, typeChip, incrementStats, statusFlag } from './increment.mjs';\n"+
			"const odd = { name: 'mi-feature', type: 'active', phase: 'apply', change_type: 'odd', tasks_total: 4, tasks_completed: 1 };\n"+
			"const oddDone = { name: 'viejo', type: 'archived', phase: 'archive', change_type: 'odd', tasks_total: 2, tasks_completed: 2 };\n"+
			"const sdd = { name: 'x', type: 'active', phase: 'spec', pending_spec: true, change_type: 'feature', tasks_total: 2, tasks_completed: 1 };\n"+
			"const withOdd = incrementStats([odd, oddDone, sdd]);\n"+
			"const withoutOdd = incrementStats([sdd]);\n"+
			"console.log(JSON.stringify({\n"+
			"  info: typeInfo(odd), chip: typeChip(odd), feature: typeInfo(sdd).id,\n"+
			"  flag: statusFlag(odd),\n"+
			"  same: JSON.stringify(withOdd) === JSON.stringify(withoutOdd), active: withOdd.active, archived: withOdd.archived, tasks: withOdd.tasksTotal,\n"+
			"  empty: incrementStats(null).active,\n"+
			"}));\n")
	var res struct {
		Info struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"info"`
		Chip    string `json:"chip"`
		Feature string `json:"feature"`
		Flag    any    `json:"flag"`
		Same    bool   `json:"same"`
		Active  int    `json:"active"`
		Archive int    `json:"archived"`
		Tasks   int    `json:"tasks"`
		Empty   int    `json:"empty"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if res.Info.ID != "odd" || res.Info.Label != "ODD" || res.Feature != "feature" {
		t.Errorf("typeInfo: %+v / %s", res.Info, res.Feature)
	}
	if !strings.Contains(res.Chip, "type-odd") || !strings.Contains(res.Chip, ">ODD<") && !strings.Contains(res.Chip, "ODD</span>") {
		t.Errorf("el chip debía ser type-odd con el texto ODD: %s", res.Chip)
	}
	if res.Flag != nil {
		t.Errorf("una feature ODD no tiene bandera de estado SDD: %v", res.Flag)
	}
	if !res.Same || res.Active != 1 || res.Archive != 0 || res.Tasks != 2 || res.Empty != 0 {
		t.Errorf("las features ODD no deben alterar los indicadores SDD de Inicio: %+v", res)
	}

	inc := readAsset(t, "js/screens/incrementos.js")
	for _, marker := range []string{"id: 'odd', label: 'ODD'", "nameHtml(", "change_type === 'odd'"} {
		if !strings.Contains(inc, marker) {
			t.Errorf("incrementos.js no contiene %q", marker)
		}
	}
	if html := readAsset(t, "index.html"); !strings.Contains(html, "cada feature ODD") || strings.Contains(html, "Ciclo de vida SDD de cada cambio") {
		t.Errorf("el subtítulo del tablero debía mencionar las features ODD")
	}
	if css := readAsset(t, "style.css"); !strings.Contains(css, ".type-odd") {
		t.Errorf("style.css no define .type-odd")
	}
}

// La pestaña «Explicación» sigue a «Handoff», siempre es visible y enmarca el HTML en un iframe
// de origen opaco, sin controles de edición.
func TestExplainerTab(t *testing.T) {
	t.Parallel()
	html := readAsset(t, "index.html")
	handoff := strings.Index(html, `data-tab="handoff"`)
	tab := strings.Index(html, `data-tab="explicacion"`)
	if handoff < 0 || tab < handoff {
		t.Errorf("la pestaña «Explicación» debe ir tras «Handoff» (handoff=%d, explicacion=%d)", handoff, tab)
	}
	if !regexp.MustCompile(`data-tab="explicacion"[^>]*>Explicación<`).MatchString(html) {
		t.Error("la pestaña «Explicación» no tiene su rótulo")
	}
	if regexp.MustCompile(`data-tab="explicacion"[^>]*hidden`).MatchString(html) {
		t.Error("la pestaña «Explicación» debe ser siempre visible")
	}
	if !strings.Contains(html, `data-panel="explicacion"`) {
		t.Error("index.html no contiene el panel explicacion")
	}

	src := readAsset(t, "js/screens/detalle-explicacion.js")
	if !strings.Contains(src, `sandbox=""`) {
		t.Error(`el iframe debe llevar sandbox="" literal`)
	}
	for _, forbidden := range []string{"allow-same-origin", "allow-scripts", "<button", "<textarea", "contenteditable", "method: 'POST'", "Regenerar", "Guardar"} {
		if strings.Contains(src, forbidden) {
			t.Errorf("detalle-explicacion.js no debe contener %q", forbidden)
		}
	}
	for _, marker := range []string{
		"no está activada", "Todavía no hay documento de origen", "No hay explicación generada", "desactualizado",
		"/api/explainer?change=", "encodeURIComponent(change)",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("detalle-explicacion.js no contiene %q", marker)
		}
	}
	if !strings.Contains(readAsset(t, "js/screens/detalle.js"), "renderExplainer(") {
		t.Error("detalle.js no pinta la pestaña «Explicación»")
	}
}
