package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Las pruebas de este fichero cubren las pantallas de Conocimiento (T5): Skills,
// Specs vivas y Código y grafo. Son estructurales (no hay motor de JavaScript en
// `go test`) salvo el analizador de escenarios y el router, que se ejecutan con Node
// si está disponible. Comprueban las reglas de D3 (sin historial ni specs afectadas),
// D5 (aprobar y rechazar piden confirmación), accesibilidad y que los filtros de
// símbolos viajan al servidor.

func TestKnowledgeAssetsAreServed(t *testing.T) {
	h := assetServer(t)
	for _, target := range []string{
		"/js/core/spec.js", "/js/core/format.js",
		"/js/screens/skills.js", "/js/screens/livingdoc.js", "/js/screens/semantic.js",
	} {
		rr := getAsset(t, h, target)
		if rr.Code != http.StatusOK {
			t.Errorf("GET %s = %d, se esperaba 200", target, rr.Code)
			continue
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
			t.Errorf("GET %s: Content-Type = %q, se esperaba JavaScript", target, ct)
		}
	}
}

func TestSpecsDomainRouteIsWired(t *testing.T) {
	router := readAsset(t, "js/core/router.js")
	for _, marker := range []string{"head === 'specs' && parts.length >= 2", "name: 'specs', domain", "#/specs/${encodeURIComponent(params.domain)}", "hrefFor(route.name, route)"} {
		if !strings.Contains(router, marker) {
			t.Errorf("router.js no contiene %q", marker)
		}
	}
	main := readAsset(t, "js/main.js")
	for _, marker := range []string{"enterSpecs(route)", "import { enterSpecs, initLivingDocs, loadLivingDocs }"} {
		if !strings.Contains(main, marker) {
			t.Errorf("main.js no contiene %q", marker)
		}
	}
	shell := readAsset(t, "js/shell.js")
	if !strings.Contains(shell, "route.name === 'specs' && route.domain") {
		t.Errorf("la miga debe mostrar el dominio abierto de Specs vivas")
	}
	// Cada dominio de la lista es un enlace de ruta, no un botón: atrás y adelante funcionan.
	live := readAsset(t, "js/screens/livingdoc.js")
	for _, marker := range []string{"hrefFor('specs', { domain: d.domain })", `aria-current="page"`, "export function enterSpecs"} {
		if !strings.Contains(live, marker) {
			t.Errorf("livingdoc.js no contiene %q", marker)
		}
	}
}

// Aprobar y rechazar pasan por una confirmación antes de llamar a la API (D5).
func TestSkillDecisionsGoThroughAConfirmation(t *testing.T) {
	js := readAsset(t, "js/screens/skills.js")
	for _, fn := range []struct{ name, endpoint string }{
		{"async function approve(", "api('/api/skills/approve'"},
		{"async function reject(", "api('/api/skills/reject'"},
	} {
		start := strings.Index(js, fn.name)
		if start < 0 {
			t.Fatalf("skills.js no define %s", fn.name)
		}
		body := js[start:]
		confirm, call := strings.Index(body, "await confirmDialog("), strings.Index(body, fn.endpoint)
		if confirm < 0 || call < 0 || confirm > call {
			t.Errorf("%s debe pedir confirmación (confirmDialog) antes de llamar a la API", fn.name)
		}
		if !strings.Contains(body[:call], "if (!ok) return;") {
			t.Errorf("%s debe salir si el usuario cancela", fn.name)
		}
	}
	for _, marker := range []string{
		"danger: true",               // rechazar es destructivo
		"refreshOverview(['inbox'])", // la insignia de la barra lateral
		"Promise.all([loadSkillsInbox(), loadSkills()",
		"/\\(aviso:/i.test(message)", // el aviso de la aprobación no se pierde
		"stateEmpty('Buzón vacío'",
		"api('/api/skills/scan'",
	} {
		if !strings.Contains(js, marker) {
			t.Errorf("skills.js no contiene %q", marker)
		}
	}
	// La vista previa muestra los campos del DTO y renderiza SKILL.md con el renderizador seguro.
	for _, field := range []string{"p.origin", "p.source", "p.verified", "p.role", "p.detected_by", "p.justification", "p.created_at", "p.skill_md", "p.sha256", "renderMarkdown(p.skill_md)", "shortHash("} {
		if !strings.Contains(js, field) {
			t.Errorf("la vista previa de skills no usa %q", field)
		}
	}
	// Las acciones son botones con texto; no hay controles de solo icono sin nombre.
	if strings.Contains(js, "innerHTML = p.") {
		t.Errorf("skills.js no debe volcar campos de la API sin escapar")
	}
}

func TestSpecsViewerOmitsHistoryAndAffectedSpecs(t *testing.T) {
	html := readAsset(t, "index.html")
	start, end := strings.Index(html, `id="view-specs"`), strings.Index(html, `id="view-skills"`)
	if start < 0 || end < start {
		t.Fatalf("index.html: no se localiza la vista de specs vivas")
	}
	specs := strings.ToLower(html[start:end] + readAsset(t, "js/screens/livingdoc.js"))
	for _, forbidden := range []string{"historial", "cambios en curso", "modificará", "modificara", "specs afectadas"} {
		if strings.Contains(specs, forbidden) {
			t.Errorf("Specs vivas no debe incluir %q (D3: el servidor no expone esos datos)", forbidden)
		}
	}
	live := readAsset(t, "js/screens/livingdoc.js")
	for _, marker := range []string{"api('/api/archive/sync', { method: 'POST' })", "renderSpec(content)", "Fallo al sincronizar", "last_sync", "total_requirements"} {
		if !strings.Contains(live, marker) {
			t.Errorf("livingdoc.js no contiene %q", marker)
		}
	}
}

// Los filtros de símbolos (texto y tipo) viajan al servidor como query y kind.
func TestSymbolFiltersAreServerSide(t *testing.T) {
	js := readAsset(t, "js/screens/semantic.js")
	for _, marker := range []string{"params.append('query', query)", "params.append('kind', kind)", "/api/semantic/symbols${qs ?", "setTimeout(", "250", "ctx.symbolsToken"} {
		if !strings.Contains(js, marker) {
			t.Errorf("semantic.js no contiene %q", marker)
		}
	}
	for _, marker := range []string{"native_ast_ready", "codegraph_installed", "serena_installed", "api('/api/semantic/reindex', { method: 'POST' })", "api('/api/semantic/dependencies')", "data.agents"} {
		if !strings.Contains(js, marker) {
			t.Errorf("semantic.js no contiene %q", marker)
		}
	}
	// El servidor acepta realmente el filtro por tipo.
	root := t.TempDir()
	writeFixtureFile(t, root, "sample/sample.go", "package sample\n\ntype Service struct{}\n\nfunc Helper() {}\n")
	router := NewServer(NewService(root)).Router()
	for kind, want := range map[string]string{"struct": "Service", "func": "Helper"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/semantic/symbols?kind="+kind, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("GET /api/semantic/symbols?kind=%s = %d: %s", kind, rr.Code, rr.Body.String())
		}
		var symbols []struct{ Name, Kind string }
		if err := json.Unmarshal(rr.Body.Bytes(), &symbols); err != nil {
			t.Fatalf("JSON inválido: %v", err)
		}
		if len(symbols) == 0 {
			t.Errorf("kind=%s no devolvió símbolos", kind)
		}
		for _, s := range symbols {
			if s.Kind != kind {
				t.Errorf("kind=%s devolvió el símbolo %q de tipo %q", kind, s.Name, s.Kind)
			}
		}
		found := false
		for _, s := range symbols {
			found = found || s.Name == want
		}
		if !found {
			t.Errorf("kind=%s no incluye %q: %+v", kind, want, symbols)
		}
	}
}

func TestKnowledgeScreensUseNoLegacyStyles(t *testing.T) {
	legacy := []string{"skill-card", "inbox-card", "inbox-grid", "inbox-section", "livingdoc-domain-item", "livingdoc-viewer-box", "livingdoc-stats-grid", "livingdoc-layout", "livingdoc-list", "badge-origin-", "stat-card", "legacy-card", "filter-btn", "agent-item", "dep-chip", "badge-connector", "badge-kind", "symbols-toolbar", "search-box", "loading-state", "empty-state", "error-banner", "subsection-title", "btn-preview-skill"}
	html := readAsset(t, "index.html")
	start, end := strings.Index(html, `id="view-specs"`), strings.Index(html, `id="view-ecosistema"`)
	if start < 0 || end < start {
		t.Fatalf("index.html: no se localizan las vistas de conocimiento")
	}
	sources := map[string]string{
		"index.html (specs, skills y código)": html[start:end],
		"skills.js":                           readAsset(t, "js/screens/skills.js"),
		"livingdoc.js":                        readAsset(t, "js/screens/livingdoc.js"),
		"semantic.js":                         readAsset(t, "js/screens/semantic.js"),
		"spec.js":                             readAsset(t, "js/core/spec.js"),
	}
	for name, src := range sources {
		for _, class := range legacy {
			if strings.Contains(src, class) {
				t.Errorf("%s usa la clase heredada %q", name, class)
			}
		}
		if loc := regexp.MustCompile(`#[0-9A-Fa-f]{6}\b|#[0-9A-Fa-f]{3}\b`).FindString(strings.ReplaceAll(src, `href="#/`, "")); loc != "" && !strings.HasPrefix(name, "index.html") {
			t.Errorf("%s contiene el color fijo %q; usa los tokens", name, loc)
		}
	}

	css := readAsset(t, "style.css")
	for _, marker := range []string{".kn-split", ".kn-layout-specs", ".kn-table-scroll", ".kn-search", ".spec-step", ".spec-kw", ".spec-must", ".kn-pp[aria-pressed=\"true\"]", ".kn-domain[aria-current=\"page\"]", "@media (max-width: 859px)", "@media (max-width: 480px)"} {
		if !strings.Contains(css, marker) {
			t.Errorf("style.css no contiene %q", marker)
		}
	}
	// Los buscadores conservan el foco visible: ninguna regla del bloque nuevo lo quita.
	block := css[strings.Index(css, "Conocimiento (T5)"):]
	if strings.Contains(block, "outline: 0") || strings.Contains(block, "outline: none") {
		t.Errorf("el bloque de Conocimiento no debe quitar el contorno de foco")
	}
	// Sin scroll horizontal de página: las cajas anchas desplazan por dentro.
	if !strings.Contains(block, ".kn-table-scroll { max-height: 420px; overflow: auto; }") {
		t.Errorf("la tabla de símbolos debe desplazarse dentro de su caja")
	}
}

func TestKnowledgeScreensAreAccessible(t *testing.T) {
	html := readAsset(t, "index.html")
	for _, id := range []string{"specs-filter", "skills-filter", "semantic-search-input"} {
		if !regexp.MustCompile(`<label class="sr-only" for="` + id + `">[^<]+</label>`).MatchString(html) {
			t.Errorf("el buscador %q necesita una <label for> accesible", id)
		}
	}
	for _, marker := range []string{
		`aria-label="Dominios de especificación"`,
		`id="semantic-kind-filters" role="group" aria-label="Filtrar por tipo de símbolo"`,
		`id="deps-internal-toggle" aria-pressed="false"`,
		`id="semantic-symbols-count" role="status" aria-live="polite"`,
		`id="scan-feedback" class="hidden" role="status" aria-live="polite"`,
		`id="reindex-feedback" class="hidden" role="status" aria-live="polite"`,
		`aria-label="Salida de la reindexación"`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}
	// El estado se dice con texto (y icono), no solo con color.
	sem := readAsset(t, "js/screens/semantic.js")
	for _, text := range []string{"Instalado (CLI en PATH)", "No está en PATH", "Configurado en el workspace", "Sin configurar en el workspace", "AST nativo listo", "AST nativo no disponible"} {
		if !strings.Contains(sem, text) {
			t.Errorf("semantic.js no expresa con texto el estado %q", text)
		}
	}
	skills := readAsset(t, "js/screens/skills.js")
	for _, text := range []string{"SHA-256 verificado", "Sin verificar", "aria-pressed=\"${current}\"", "role=\"group\" aria-label=\"Formato de SKILL.md\""} {
		if !strings.Contains(skills, text) {
			t.Errorf("skills.js no contiene %q", text)
		}
	}
	// Las cajas con desplazamiento propio son enfocables con el teclado.
	if !strings.Contains(sem, `role="region" aria-label="Símbolos" tabindex="0"`) {
		t.Errorf("la tabla de símbolos debe ser una región enfocable con nombre")
	}
}

// El analizador de escenarios y el router se ejecutan de verdad (si hay Node) con entradas hostiles.
func TestSpecParserAndRouterRuntime(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite la prueba de ejecución del visor de specs")
	}
	dir := t.TempDir()
	for _, name := range []string{"api", "icons", "markdown", "spec", "router"} {
		src := readAsset(t, "js/core/"+name+".js")
		src = regexp.MustCompile(`from '\./([a-z]+)\.js'`).ReplaceAllString(src, "from './$1.mjs'")
		if err := os.WriteFile(filepath.Join(dir, name+".mjs"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := "import { parseSpec, renderSpec, specStats, emphasizeModals } from './spec.mjs';\n" +
		"import { parseHash, hrefFor } from './router.mjs';\n" +
		"const hostile = [\n" +
		"  '# Título <script>alert(1)</script>',\n" +
		"  '',\n" +
		"  '### Requirement: Sesión <img src=x onerror=alert(1)> (REQ-1.1)',\n" +
		"  '',\n" +
		"  'El sistema **DEBE** crear y NO DEBE borrar. También MUST cumplir `DEBE` en código.',\n" +
		"  '',\n" +
		"  '#### Scenario: Carrito <b>válido</b>',\n" +
		"  '- **DADO** un carrito [x](javascript:alert(1))',\n" +
		"  '- **CUANDO** el usuario inicia el pago',\n" +
		"  '- **ENTONCES** se crea una sesión <svg onload=alert(1)>',\n" +
		"  '- **Y** se registra',\n" +
		"  '  continuación del paso',\n" +
		"  '',\n" +
		"  '#### Scenario: English',\n" +
		"  '- **GIVEN** a user',\n" +
		"  '- **WHEN** it logs in',\n" +
		"  '- **THEN** it works and MUST succeed',\n" +
		"  '',\n" +
		"  '---',\n" +
		"  '',\n" +
		"  '### Requirement: Sin escenarios (REQ-1.2)',\n" +
		"  '',\n" +
		"  'Texto plano con DEBE.',\n" +
		"  '',\n" +
		"  '```',\n" +
		"  '### Requirement: Dentro de un bloque de código',\n" +
		"  '```',\n" +
		"].join('\\n');\n" +
		"const blocks = parseSpec(hostile);\n" +
		"const reqs = blocks.filter(b => b.type === 'requirement');\n" +
		"console.log(JSON.stringify({\n" +
		"  types: blocks.map(b => b.type),\n" +
		"  ids: reqs.map(r => r.id),\n" +
		"  titles: reqs.map(r => r.title),\n" +
		"  kinds: reqs[0].scenarios.map(s => s.steps.map(x => x.kind + ':' + x.kw)),\n" +
		"  lastStep: reqs[0].scenarios[0].steps[3].text,\n" +
		"  stats: specStats(hostile),\n" +
		"  html: renderSpec(hostile),\n" +
		"  emph: emphasizeModals('<a href=\"x-MUST-y\">DEBE</a> DEBE <code>MUST</code> <strong>DEBE</strong> NO DEBE'),\n" +
		"  empty: renderSpec(''),\n" +
		"  plain: renderSpec('Solo texto\\n\\n- **DADO** suelto sin escenario'),\n" +
		"  route: [parseHash('#/specs'), parseHash('#/specs/auth'), parseHash('#/specs/a%20b'), parseHash('#/specs/x/extra'), parseHash('#/specs/%E0%A4%A')],\n" +
		"  hrefs: [hrefFor('specs'), hrefFor('specs', { domain: 'a b/c' })],\n" +
		"}));\n"
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
		Types    []string   `json:"types"`
		IDs      []string   `json:"ids"`
		Titles   []string   `json:"titles"`
		Kinds    [][]string `json:"kinds"`
		LastStep string     `json:"lastStep"`
		Stats    struct{ Requirements, Scenarios int }
		HTML     string `json:"html"`
		Emph     string `json:"emph"`
		Empty    string `json:"empty"`
		Plain    string `json:"plain"`
		Route    []struct {
			Name      string `json:"name"`
			Domain    string `json:"domain"`
			Canonical bool   `json:"canonical"`
		} `json:"route"`
		Hrefs []string `json:"hrefs"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}

	// Estructura: un bloque de texto inicial y dos requisitos; lo de dentro del bloque de código no abre otro.
	if got := strings.Join(res.Types, ","); got != "md,requirement,requirement" {
		t.Errorf("bloques = %s", got)
	}
	if got := strings.Join(res.IDs, ","); got != "REQ-1.1,REQ-1.2" {
		t.Errorf("ids = %s", got)
	}
	if len(res.Titles) != 2 || !strings.HasPrefix(res.Titles[1], "Sin escenarios") {
		t.Errorf("títulos = %v", res.Titles)
	}
	wantKinds := [][]string{{"given:DADO", "when:CUANDO", "then:ENTONCES", "and:Y"}, {"given:GIVEN", "when:WHEN", "then:THEN"}}
	if len(res.Kinds) != 2 || strings.Join(res.Kinds[0], "|") != strings.Join(wantKinds[0], "|") || strings.Join(res.Kinds[1], "|") != strings.Join(wantKinds[1], "|") {
		t.Errorf("pasos = %v, se esperaba %v", res.Kinds, wantKinds)
	}
	if res.LastStep != "se registra continuación del paso" {
		t.Errorf("la línea con sangría debe continuar el paso anterior: %q", res.LastStep)
	}
	if res.Stats.Requirements != 2 || res.Stats.Scenarios != 2 {
		t.Errorf("specStats = %+v", res.Stats)
	}

	// Escape: nada del documento se inyecta como HTML.
	for _, bad := range []string{"<script", "<img", "<svg", "<b>", "javascript:", "onerror=alert(1)>", `<strong><strong`, "<code><strong"} {
		if strings.Contains(res.HTML, bad) {
			t.Errorf("el HTML del visor contiene %q sin escapar: %s", bad, res.HTML)
		}
	}
	for _, good := range []string{
		"&lt;script&gt;", "&lt;img", "&lt;svg", "&lt;b&gt;",
		`<span class="spec-kw mono">DADO</span>`, `<span class="spec-kw mono">CUANDO</span>`, `<span class="spec-kw mono">ENTONCES</span>`,
		`<span class="spec-kw mono">GIVEN</span>`, `spec-step-and`, `role="group" aria-label="Escenario:`,
		`<strong class="spec-must">NO DEBE</strong>`, `<strong class="spec-must">MUST</strong>`,
	} {
		if !strings.Contains(res.HTML, good) {
			t.Errorf("el HTML del visor no contiene %q: %s", good, res.HTML)
		}
	}
	// DEBE/MUST solo se resaltan en texto: no en código, enlaces ni donde ya hay negrita.
	if want := `<a href="x-MUST-y">DEBE</a> <strong class="spec-must">DEBE</strong> <code>MUST</code> <strong class="spec-must">DEBE</strong> <strong class="spec-must">NO DEBE</strong>`; res.Emph != want {
		t.Errorf("emphasizeModals = %q, se esperaba %q", res.Emph, want)
	}
	if res.Empty != "" {
		t.Errorf("un documento vacío no debe pintar nada: %q", res.Empty)
	}
	if strings.Contains(res.Plain, "spec-step") || !strings.Contains(res.Plain, "<strong>DADO</strong>") {
		t.Errorf("fuera de un escenario, DADO es Markdown normal: %s", res.Plain)
	}

	// Router: #/specs/<dominio> es una ruta, atrás y adelante la restauran, y lo raro cae a inicio.
	if len(res.Route) != 5 {
		t.Fatalf("rutas = %+v", res.Route)
	}
	if r := res.Route[0]; r.Name != "specs" || r.Domain != "" || !r.Canonical {
		t.Errorf("#/specs = %+v", r)
	}
	if r := res.Route[1]; r.Name != "specs" || r.Domain != "auth" || !r.Canonical {
		t.Errorf("#/specs/auth = %+v", r)
	}
	if r := res.Route[2]; r.Domain != "a b" {
		t.Errorf("#/specs/a%%20b = %+v", r)
	}
	if r := res.Route[3]; r.Name != "specs" || r.Domain != "x" || r.Canonical {
		t.Errorf("#/specs/x/extra debe normalizarse a #/specs/x: %+v", r)
	}
	if r := res.Route[4]; r.Name != "inicio" {
		t.Errorf("un dominio mal codificado cae a inicio: %+v", r)
	}
	if len(res.Hrefs) != 2 || res.Hrefs[0] != "#/specs" || res.Hrefs[1] != "#/specs/a%20b%2Fc" {
		t.Errorf("hrefFor = %v", res.Hrefs)
	}
}

// Todos los módulos embebidos se analizan con Node (sin ejecutarlos): un error de
// sintaxis rompería la carga de la interfaz entera y las pruebas estructurales no lo verían.
func TestEmbeddedModulesParse(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite el análisis de sintaxis de los módulos")
	}
	dir := t.TempDir()
	for _, f := range embeddedFiles(t) {
		if !strings.HasSuffix(f, ".js") {
			continue
		}
		target := filepath.Join(dir, strings.ReplaceAll(strings.TrimSuffix(f, ".js"), "/", "_")+".mjs")
		if err := os.WriteFile(target, []byte(readAsset(t, f)), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(node, "--check", target).CombinedOutput(); err != nil {
			t.Errorf("%s no es JavaScript válido: %v\n%s", f, err, out)
		}
	}
}

// T9: Código (símbolos y dependencias, miles de filas) se carga al entrar por primera
// vez en #/codigo; «Actualizar» solo lo repite si ya se abrió.
func TestCodeScreenLoadsLazily(t *testing.T) {
	main := readAsset(t, "js/main.js")
	if strings.Contains(main, "loadSemanticData") {
		t.Errorf("main.js no debe pedir los datos de Código al arrancar (loadSemanticData)")
	}
	for _, want := range []string{"refreshSemantic()", "if (route.name === 'codigo') enterSemantic();"} {
		if !strings.Contains(main, want) {
			t.Errorf("main.js debe contener %q", want)
		}
	}

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite la prueba de ejecución de la carga diferida")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lazy.mjs"), []byte(readAsset(t, "js/core/lazy.js")), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "import { lazyLoader } from './lazy.mjs';\n" +
		"let calls = 0;\n" +
		"const l = lazyLoader(async () => { calls++; });\n" +
		"const fail = (m) => { console.error(m); process.exit(1); };\n" +
		"await l.refresh();\n" +
		"if (calls !== 0 || l.started) fail('refresh antes de entrar no debe cargar');\n" +
		"await l.enter();\n" +
		"if (calls !== 1 || !l.started) fail('la primera entrada debe cargar una vez');\n" +
		"await l.enter();\n" +
		"if (calls !== 1) fail('volver a entrar no debe recargar');\n" +
		"await l.refresh();\n" +
		"if (calls !== 2) fail('refresh tras entrar debe recargar');\n"
	if err := os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "run.mjs")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("carga diferida: %v\n%s", err, out)
	}
}
