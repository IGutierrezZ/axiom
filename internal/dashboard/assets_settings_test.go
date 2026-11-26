package dashboard

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Ajustes del proyecto (C-06-e): la estructura de la pantalla se comprueba sobre las cadenas de los
// assets, como el resto de pruebas de assets, y la lógica pura de core/settings.js se ejecuta de
// verdad en Node (se omite sin Node). El contrato con el servidor lo cubre settings_api_test.go.

// RF-WEB-18: la ruta `ajustes` está registrada en el router, el mapa de vistas y las etiquetas.
func TestSettingsAssetsRouteIsWired(t *testing.T) {
	t.Parallel()
	router, main, shell, html := readAsset(t, "js/core/router.js"), readAsset(t, "js/main.js"), readAsset(t, "js/shell.js"), readAsset(t, "index.html")

	if !regexp.MustCompile(`SIMPLE_ROUTES = \[[^\]]*'ajustes'`).MatchString(router) || !strings.Contains(router, "#/ajustes") {
		t.Error("router.js debe registrar la ruta 'ajustes' en SIMPLE_ROUTES y en el comentario de rutas")
	}
	for _, marker := range []string{"ajustes: 'view-ajustes'", "initSettings()", "loadSettings()", "from './screens/ajustes.js'"} {
		if !strings.Contains(main, marker) {
			t.Errorf("main.js debe contener %q", marker)
		}
	}
	if !strings.Contains(shell, "ajustes: 'Ajustes del proyecto'") {
		t.Error("shell.js debe etiquetar la ruta como «Ajustes del proyecto»")
	}
	if !strings.Contains(readAsset(t, "js/core/icons.js"), "settings:") {
		t.Error("icons.js debe tener el icono `settings` en línea")
	}
	nav := regexp.MustCompile(`<a class="nav-item" href="#/ajustes" data-nav="ajustes">[^\n]*</a>`).FindString(html)
	if nav == "" || !strings.Contains(nav, `data-icon="settings"`) {
		t.Errorf("index.html debe tener la entrada de navegación a #/ajustes con su icono: %q", nav)
	}
	if platform := html[strings.Index(html, `<span class="nav-label">Plataforma</span>`):]; strings.Index(platform, `data-nav="ajustes"`) < 0 || strings.Index(platform, `data-nav="ajustes"`) > strings.Index(platform, "</nav>") {
		t.Error("la entrada de Ajustes va en el grupo «Plataforma»")
	}
}

// RF-WEB-18: la vista lleva el aviso «Editar solo afecta a cambios nuevos», el contenedor del
// catálogo y la tarjeta de preferencias personales; ningún control de RDD ni de `integrations`.
func TestSettingsAssetsViewStructure(t *testing.T) {
	t.Parallel()
	html := readAsset(t, "index.html")
	start := strings.Index(html, `id="view-ajustes"`)
	end := strings.Index(html[start:], "</main>")
	if start < 0 || end < 0 {
		t.Fatal("index.html: no se localiza la vista de Ajustes")
	}
	view := html[start : start+end]

	if !strings.Contains(view, "Editar solo afecta a cambios nuevos") {
		t.Error("la vista debe avisar de que «Editar solo afecta a cambios nuevos»")
	}
	for _, id := range []string{"ajustes-title", "settings-body", "settings-config-path", "settings-announce", "form-autofetch", "input-autofetch", "btn-save-autofetch", "autofetch-status", "settings-personal-state"} {
		if !strings.Contains(view, `id="`+id+`"`) {
			t.Errorf("la vista de Ajustes no contiene #%s", id)
		}
	}
	if !strings.Contains(view, `for="input-autofetch"`) {
		t.Error("el control de autofetch necesita su <label for>")
	}
	if !strings.Contains(html, `id="ajustes-title" class="page-title" tabindex="-1"`) {
		t.Error("el título lleva tabindex=-1 para recibir el foco al navegar")
	}
	// El editor de worktree salió de Ecosistema: vive solo en Ajustes.
	eco := html[strings.Index(html, `id="view-ecosistema"`):start]
	for _, id := range []string{"form-worktree-prefs", "select-worktree-prefs-mode", "input-worktree-prefs-root"} {
		if strings.Contains(eco, `id="`+id+`"`) {
			t.Errorf("#%s ya no debe estar en la vista de Ecosistema", id)
		}
	}
	// Ni controles de RDD ni del bloque integrations, ni claves concretas en la pantalla.
	rdd := regexp.MustCompile(`(?i)\brdd\b`)
	for name, text := range map[string]string{
		"la vista":           view,
		"screens/ajustes.js": readAsset(t, "js/screens/ajustes.js"),
		"core/settings.js":   readAsset(t, "js/core/settings.js"),
	} {
		if rdd.MatchString(text) {
			t.Errorf("%s no debe ofrecer controles de RDD", name)
		}
		if strings.Contains(strings.ToLower(text), "integrations") {
			t.Errorf("%s no debe tratar el bloque integrations", name)
		}
	}
	if strings.Contains(html, "integration_branch") || strings.Contains(html, "branch_prefix") {
		t.Error("index.html no debe contener claves de git: la pantalla se genera del catálogo (RF-WEB-10)")
	}
	js := readAsset(t, "js/screens/ajustes.js") + readAsset(t, "js/core/settings.js")
	for _, key := range []string{"workflow.lane", "git.promotion", "sdd.iso", "gate_policy"} {
		if strings.Contains(js, key) {
			t.Errorf("la pantalla no debe conocer la clave %q: sale del catálogo", key)
		}
	}
}

// RF-WEB-18: la pantalla guarda con un PUT por lote, conserva lo tecleado al fallar y copia la ruta.
func TestSettingsAssetsScreenContract(t *testing.T) {
	t.Parallel()
	js := readAsset(t, "js/screens/ajustes.js")
	save := funcBody(t, js, "saveSettings")
	for _, marker := range []string{"/api/settings", "method: 'PUT'", "revision: current.revision", "collectChanges(", "errorView(err)", "toast("} {
		if !strings.Contains(save, marker) {
			t.Errorf("saveSettings debe contener %q:\n%s", marker, save)
		}
	}
	// Un fallo no repinta el formulario: renderSettings solo se llama en el camino correcto.
	if catchPart := save[strings.Index(save, "} catch"):]; strings.Contains(catchPart, "renderSettings(") {
		t.Errorf("un fallo no debe repintar el formulario (se perdería lo tecleado):\n%s", catchPart)
	}
	for _, marker := range []string{"/api/settings/personal", "/api/projects/active/worktree-prefs", "Ruta copiada al portapapeles", "copyText(", "data-settings-reload"} {
		if !strings.Contains(js, marker) {
			t.Errorf("ajustes.js debe contener %q", marker)
		}
	}
	if strings.Contains(js, "bindCopyButtons") {
		t.Error("«Copiar ruta» no debe reutilizar bindCopyButtons: su aviso dice «Orden copiada»")
	}
	// Cada bloque carga y falla por separado.
	if load := funcBody(t, js, "loadSettings"); !strings.Contains(load, "loadCatalog()") || !strings.Contains(load, "loadPersonal()") || !strings.Contains(load, "loadWorktreePrefs()") {
		t.Errorf("loadSettings debe cargar el catálogo y las dos preferencias personales:\n%s", load)
	}
	if !strings.Contains(js, "export function initSettings()") {
		t.Error("ajustes.js debe exportar initSettings()")
	}
	if strings.Contains(js, "innerHTML = err.message") || strings.Contains(js, "innerHTML = `${err.message}") {
		t.Error("los mensajes del servidor no se pintan como HTML sin escapar")
	}
}

// RF-WEB-07, RF-WEB-18: core/settings.js se ejecuta de verdad con entradas hostiles.
func TestSettingsRuntimeControlsAndChanges(t *testing.T) {
	t.Parallel()
	out := runNodeModules(t, []string{"api", "icons", "components", "settings"},
		"import { groupBySection, controlHtml, readOnlyHtml, collectChanges, editorHref, pathActionsHtml, errorView, settingId } from './settings.mjs';\n"+
			"const enumEntry = { key: 'git.promotion', section: 'git', label: 'Promoción', description: 'Cómo se promueve.', effect: 'Solo afecta a cambios nuevos.', type: 'enum', allowed: ['branch', 'release', 'ask'], editable: true, required: false, default: 'branch', declared: false, value: null };\n"+
			"const lane = { key: 'workflow.lane', section: 'workflow', label: 'Carril', type: 'enum', allowed: ['sdd', 'ask'], editable: true, required: false, declared: true, value: 'odd', default: 'sdd' };\n"+
			"const hostile = { key: 'workspace.name', section: 'workspace', label: '<script>alert(1)</script>', description: '<img src=x onerror=alert(2)>', effect: '', type: 'string', editable: true, required: true, declared: true, value: '\"><script>alert(3)</script>', default: '' };\n"+
			"const entries = [\n"+
			"  { key: 'a.enum', type: 'enum', allowed: ['x', 'y'], editable: true, declared: true, value: 'x' },\n"+
			"  { key: 'a.text', type: 'string', editable: true, declared: true, value: 'viejo' },\n"+
			"  { key: 'a.same', type: 'string', editable: true, declared: true, value: 'igual' },\n"+
			"  { key: 'a.int', type: 'int', editable: true, declared: true, value: 400 },\n"+
			"  { key: 'a.int2', type: 'int', editable: true, declared: false, value: null },\n"+
			"  { key: 'a.bool', type: 'bool', editable: true, declared: false, value: null },\n"+
			"  { key: 'a.list', type: 'stringList', editable: true, declared: true, value: ['dev', 'pro'] },\n"+
			"  { key: 'a.list2', type: 'stringList', editable: true, declared: true, value: ['dev'] },\n"+
			"  { key: 'a.untouched', type: 'string', editable: true, declared: false, value: null },\n"+
			"  { key: 'a.readonly', type: 'string', editable: false, declared: true, value: 'fijo' },\n"+
			"];\n"+
			"const values = { 'a.enum': 'y', 'a.text': '  nuevo  ', 'a.same': 'igual', 'a.int': '', 'a.int2': '500', 'a.bool': 'true', 'a.list': 'dev\\npro\\n', 'a.list2': 'dev\\r\\nqa\\n\\n', 'a.untouched': '', 'a.readonly': 'cambiado' };\n"+
			"console.log(JSON.stringify({\n"+
			"  enumHtml: controlHtml(enumEntry),\n"+
			"  laneHtml: controlHtml(lane),\n"+
			"  hostileHtml: controlHtml(hostile),\n"+
			"  disabledHtml: controlHtml(enumEntry, 'x', { disabled: true }),\n"+
			"  changes: collectChanges(entries, values),\n"+
			"  cleared: collectChanges([{ key: 'k', type: 'enum', allowed: ['x'], editable: true, declared: true, value: 'x' }], { k: '' }),\n"+
			"  groups: groupBySection([{ section: 'sdd', key: 'a' }, { section: 'workspace', key: 'b' }, { section: 'sdd', key: 'c' }, { section: 'otra', key: 'd' }]).map(g => [g.section, g.label, g.entries.map(e => e.key)]),\n"+
			"  ids: [settingId('repositories[a/b].promotion'), settingId('roles[web].gate_policy')],\n"+
			"  hrefs: ['C:\\\\repos\\\\x\\\\axiom.yaml', '/home/u/x/axiom.yaml', 'C:\\\\repos\\\\a b#c?d&e\\\\ñ\\\\axiom.yaml', '', null, undefined].map(editorHref),\n"+
			"  actions: pathActionsHtml('C:\\\\repos\\\\\"x\\\\axiom.yaml'),\n"+
			"  readonly: readOnlyHtml({ key: 'workspace.topology', label: 'Topología', type: 'string', editable: false, value: '<b>mono</b>', readOnlyReason: 'Solo lectura.' }, 'C:\\\\repos\\\\x\\\\axiom.yaml'),\n"+
			"  readonlyEmpty: readOnlyHtml({ key: 'technical_context', label: 'Contexto', type: 'yaml', editable: false, value: 'entry: <x>' }, ''),\n"+
			"  mismatch: errorView({ message: 'El fichero cambió <ya>', status: 409, data: { code: 'revision_mismatch', error: 'x' } }),\n"+
			"  invalid: errorView({ message: 'No es válido', status: 422, data: { code: 'validation_failed', violations: ['línea 1: <mal>', 'línea 2'] } }),\n"+
			"  bare: errorView(null),\n"+
			"}));\n")
	var res struct {
		EnumHTML      string         `json:"enumHtml"`
		LaneHTML      string         `json:"laneHtml"`
		HostileHTML   string         `json:"hostileHtml"`
		DisabledHTML  string         `json:"disabledHtml"`
		Changes       map[string]any `json:"changes"`
		Cleared       map[string]any `json:"cleared"`
		Groups        [][]any        `json:"groups"`
		IDs           []string       `json:"ids"`
		Hrefs         []string       `json:"hrefs"`
		Actions       string         `json:"actions"`
		Readonly      string         `json:"readonly"`
		ReadonlyEmpty string         `json:"readonlyEmpty"`
		Mismatch      string         `json:"mismatch"`
		Invalid       string         `json:"invalid"`
		Bare          string         `json:"bare"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}

	optionValues := func(html string) []string {
		var values []string
		for _, m := range regexp.MustCompile(`<option value="([^"]*)"`).FindAllStringSubmatch(html, -1) {
			if m[1] != "" {
				values = append(values, m[1])
			}
		}
		return values
	}
	// Vocabulario cerrado: un select con exactamente los valores permitidos (más «Por defecto»).
	if !strings.Contains(res.EnumHTML, "<select") || !slices.Equal(optionValues(res.EnumHTML), []string{"branch", "release", "ask"}) {
		t.Errorf("el vocabulario cerrado es un select con exactamente allowed: %s", res.EnumHTML)
	}
	if !strings.Contains(res.EnumHTML, `<option value="" selected>Por defecto (branch)</option>`) || !strings.Contains(res.EnumHTML, `data-key="git.promotion"`) || !strings.Contains(res.EnumHTML, "Solo afecta a cambios nuevos.") {
		t.Errorf("la clave sin declarar muestra «Por defecto», su clave y su efecto: %s", res.EnumHTML)
	}
	// ISO: lane no ofrece `odd`, y el valor inválido escrito se conserva para poder corregirlo.
	if got := optionValues(res.LaneHTML); !slices.Equal(got, []string{"sdd", "ask", "odd"}) || !strings.Contains(res.LaneHTML, `<option value="odd" selected>odd (no válido)</option>`) {
		t.Errorf("un valor fuera de allowed se muestra marcado como no válido, sin ofrecerlo como permitido: %s", res.LaneHTML)
	}
	// Todo texto del servidor se escapa.
	for name, html := range map[string]string{"hostil": res.HostileHTML, "solo lectura": res.Readonly, "sin ruta": res.ReadonlyEmpty, "409": res.Mismatch, "422": res.Invalid} {
		if strings.Contains(html, "<script") || strings.Contains(html, "<img") || strings.Contains(html, "<b>") || strings.Contains(html, "<ya>") || strings.Contains(html, "<mal>") || strings.Contains(html, "<x>") {
			t.Errorf("%s: el HTML del servidor debe escaparse: %s", name, html)
		}
	}
	if !strings.Contains(res.HostileHTML, "&lt;script&gt;alert(1)&lt;/script&gt;") || strings.Contains(res.HostileHTML, `value=""><script>`) {
		t.Errorf("etiqueta y valor hostiles escapados: %s", res.HostileHTML)
	}
	if strings.Contains(res.HostileHTML, "<option") {
		t.Errorf("una cadena es un input de texto: %s", res.HostileHTML)
	}
	if !strings.Contains(res.DisabledHTML, " disabled") {
		t.Errorf("con disabled el control va deshabilitado: %s", res.DisabledHTML)
	}
	// collectChanges: solo lo cambiado, null al vaciar, sin tocar lo de solo lectura.
	want := map[string]any{
		"a.enum": "y", "a.text": "nuevo", "a.int": nil, "a.int2": float64(500), "a.bool": true,
		"a.list2": []any{"dev", "qa"},
	}
	if len(res.Changes) != len(want) {
		t.Errorf("collectChanges = %v, quiero %v", res.Changes, want)
	}
	for key, value := range want {
		got, ok := res.Changes[key]
		if !ok || !equalJSON(got, value) {
			t.Errorf("collectChanges[%s] = %v (presente: %v), quiero %v", key, got, ok, value)
		}
	}
	for _, key := range []string{"a.same", "a.list", "a.untouched", "a.readonly"} {
		if _, ok := res.Changes[key]; ok {
			t.Errorf("collectChanges no debe enviar %s", key)
		}
	}
	if v, ok := res.Cleared["k"]; !ok || v != nil {
		t.Errorf("vaciar un vocabulario declarado envía null: %v", res.Cleared)
	}
	if len(res.Groups) != 3 || res.Groups[0][0] != "workspace" || res.Groups[1][0] != "sdd" || res.Groups[2][0] != "otra" || res.Groups[2][1] != "otra" {
		t.Errorf("groupBySection ordena las secciones conocidas y deja las desconocidas al final: %v", res.Groups)
	}
	if res.IDs[0] == res.IDs[1] || regexp.MustCompile(`[^A-Za-z0-9_-]`).MatchString(res.IDs[0]) {
		t.Errorf("settingId debe dar ids válidos y distintos: %v", res.IDs)
	}
	// «Abrir en el editor»: esquema fijo y cada segmento codificado (hueco 9 de la revisión de cobertura).
	wantHrefs := []string{
		"vscode://file/C:/repos/x/axiom.yaml",
		"vscode://file/home/u/x/axiom.yaml",
		"vscode://file/C:/repos/a%20b%23c%3Fd%26e/%C3%B1/axiom.yaml",
		"", "", "",
	}
	if !slices.Equal(res.Hrefs, wantHrefs) {
		t.Errorf("editorHref = %q, quiero %q", res.Hrefs, wantHrefs)
	}
	if !strings.Contains(res.Actions, `href="vscode://file/C:/repos/%22x/axiom.yaml"`) || !strings.Contains(res.Actions, "Abrir en el editor") || !strings.Contains(res.Actions, "Copiar ruta") || !strings.Contains(res.Actions, `data-copy-path="C:\repos\&quot;x\axiom.yaml"`) {
		t.Errorf("las acciones de ruta llevan el enlace con esquema fijo y la ruta escapada: %s", res.Actions)
	}
	if !strings.Contains(res.Readonly, "Abrir en el editor") || !strings.Contains(res.Readonly, "Copiar ruta") {
		t.Errorf("cada fila de solo lectura ofrece el editor y copiar la ruta: %s", res.Readonly)
	}
	// errorView: recarga solo con revision_mismatch; las violaciones se listan.
	if !strings.Contains(res.Mismatch, "data-settings-reload") || !strings.Contains(res.Mismatch, "Recargar") {
		t.Errorf("revision_mismatch ofrece recargar: %s", res.Mismatch)
	}
	if strings.Contains(res.Invalid, "data-settings-reload") || strings.Count(res.Invalid, "<li>") != 2 || !strings.Contains(res.Invalid, "línea 1: &lt;mal&gt;") {
		t.Errorf("la validación lista sus violaciones y no ofrece recargar: %s", res.Invalid)
	}
	if !strings.Contains(res.Bare, "No se pudieron guardar los ajustes") {
		t.Errorf("un error sin datos usa el mensaje por defecto: %s", res.Bare)
	}
}

// equalJSON compara dos valores decodificados de JSON.
func equalJSON(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}
