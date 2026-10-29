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

// INC-25 llevó al dashboard antiguo cinco funciones de ramas y worktrees; el rediseño las conserva en
// módulos nuevos. Estas pruebas fijan, con el estilo del resto de assets (comprobaciones estructurales
// y, para la lógica, ejecución en Node de los módulos copiados a `.mjs`), dónde vive cada una:
//
//   - avisos del listado de incrementos (`X-Axiom-Notes`), insignias de pendiente de archivar y de rama:
//     js/screens/incrementos.js, js/core/branches.js y la lectura de cabeceras de js/core/api.js;
//   - política de ramas efectiva en solo lectura y preferencias personales de worktree del proyecto
//     activo: js/screens/ecosystem.js (tarjetas Topología y Preferencias de worktree);
//   - modo y raíz de worktree al registrar un proyecto: js/projects.js;
//   - diálogo de decisión del 409 `decision_required` al crear un incremento: js/screens/incrementos.js.
//
// Su contrato con el servidor lo cubren las pruebas HTTP de listing_test.go, server_test.go y
// service_branch_test.go (`decision_required` tipado, `branchPolicy`, `pendingArchive`, `X-Axiom-Notes`).

// funcBody devuelve el texto de una función de primer nivel (hasta su `}` en la columna 0).
func funcBody(t *testing.T, src, name string) string {
	t.Helper()
	re := regexp.MustCompile(`(?s)(?:async )?function ` + regexp.QuoteMeta(name) + `\(.*?\n}\n`)
	body := re.FindString(src)
	if body == "" {
		t.Fatalf("no se encuentra la función %s", name)
	}
	return body
}

// runNodeModules copia los módulos js/core/<nombre>.js indicados a un directorio temporal como `.mjs`
// (reescribiendo sus importaciones relativas) y ejecuta el guion con Node. Sin Node se omite la prueba.
// El guion termina imprimiendo un único JSON por la salida estándar, que se devuelve.
func runNodeModules(t *testing.T, modules []string, script string) []byte {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite la prueba de ejecución")
	}
	dir := t.TempDir()
	for _, name := range modules {
		src := readAsset(t, "js/core/"+name+".js")
		src = regexp.MustCompile(`from '\./([a-z]+)\.js'`).ReplaceAllString(src, "from './$1.mjs'")
		if err := os.WriteFile(filepath.Join(dir, name+".mjs"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "run.mjs")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node falló: %v\n%s", err, out)
	}
	return out
}

// REQ-25.34: la tarjeta de un cambio muestra la insignia de pendiente de archivar y la rama de su
// worktree, y los avisos de `X-Axiom-Notes` llegan a una región visible y anunciada del listado.
func TestIncrementsListShowsNotesPendingArchiveAndBranch(t *testing.T) {
	html, js := readAsset(t, "index.html"), readAsset(t, "js/screens/incrementos.js")

	box := regexp.MustCompile(`<div[^>]*id="increments-notes"[^>]*>`).FindString(html)
	if box == "" {
		t.Fatal("index.html debe tener el contenedor `increments-notes` de los avisos del listado")
	}
	if !strings.Contains(box, `role="status"`) || !strings.Contains(box, "hidden") {
		t.Errorf("los avisos son una región `role=\"status\"` oculta mientras no hay avisos: %s", box)
	}
	for _, marker := range []string{"apiWithHeaders('/api/increments')", "X-Axiom-Notes", "parseNotes(", "renderNotes(", "pendingArchiveBadge(inc)", "branchBadge(inc)"} {
		if !strings.Contains(js, marker) {
			t.Errorf("incrementos.js debe contener %q", marker)
		}
	}
	// El aviso lo redacta el servidor y puede nombrar rutas: se pinta como texto, nunca como HTML.
	notes := funcBody(t, js, "renderNotes")
	if !strings.Contains(notes, "textContent") || strings.Contains(notes, "innerHTML") {
		t.Errorf("renderNotes debe pintar cada aviso con textContent:\n%s", notes)
	}
	// La tarjeta y la fila de la lista llevan ambas insignias.
	for _, fn := range []string{"cardHtml", "listHtml"} {
		body := funcBody(t, js, fn)
		if !strings.Contains(body, "pendingArchiveBadge(inc)") || !strings.Contains(body, "branchBadge(inc)") {
			t.Errorf("%s debe mostrar las insignias de pendiente de archivar y de rama", fn)
		}
	}
}

// La lógica pura de los avisos y de las insignias se ejecuta de verdad (si hay Node) con entradas hostiles.
func TestNotesAndLocationBadgesRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"api", "icons", "components", "branches"},
		"import { parseNotes, pendingArchiveBadge, branchBadge } from './branches.mjs';\n"+
			"const header = '[\"cambios pendientes de archivar (1): inc-1 (axiom branch ensure --change inc-1 --archive)\",\"worktree desaparecido: C:\\\\\\\\wt\\\\\\\\x\",\"\\\\u00e1\\\\u00e9\"]';\n"+
			"console.log(JSON.stringify({\n"+
			"  parsed: parseNotes(header),\n"+
			"  none: [parseNotes(null), parseNotes(''), parseNotes('no es json'), parseNotes('{\"a\":1}'), parseNotes('[1,null,\"\",\"  \",\"ok\"]')],\n"+
			"  pending: pendingArchiveBadge({ name: 'inc-1', pendingArchive: true }),\n"+
			"  noPending: [pendingArchiveBadge({ name: 'x' }), pendingArchiveBadge({ name: 'x', pendingArchive: false }), pendingArchiveBadge(null)],\n"+
			"  branch: branchBadge({ branch: 'inc/<b>x</b>', worktreePath: 'C:\\\\wt\\\\\"x' }),\n"+
			"  noBranch: [branchBadge({}), branchBadge(undefined)],\n"+
			"}));\n")
	var res struct {
		Parsed    []string   `json:"parsed"`
		None      [][]string `json:"none"`
		Pending   string     `json:"pending"`
		NoPending []string   `json:"noPending"`
		Branch    string     `json:"branch"`
		NoBranch  []string   `json:"noBranch"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if len(res.Parsed) != 3 || !strings.HasPrefix(res.Parsed[0], "cambios pendientes de archivar (1)") || res.Parsed[2] != "áé" {
		t.Errorf("parseNotes no decodifica el array de avisos (acentos escapados incluidos): %q", res.Parsed)
	}
	for i, got := range res.None {
		want := 0
		if i == 4 {
			want = 1 // `[1,null,"","  ","ok"]` conserva solo el texto no vacío
		}
		if len(got) != want {
			t.Errorf("parseNotes caso %d = %q, se esperaban %d avisos", i, got, want)
		}
	}
	if res.None[4][0] != "ok" {
		t.Errorf("parseNotes debe descartar lo que no es texto: %q", res.None[4])
	}
	if !strings.Contains(res.Pending, "Pendiente de archivar") || !strings.Contains(res.Pending, "axiom branch ensure --change inc-1 --archive") || !strings.Contains(res.Pending, "<svg") {
		t.Errorf("la insignia de pendiente de archivar debe llevar icono, texto y la orden que archiva: %s", res.Pending)
	}
	for i, got := range res.NoPending {
		if got != "" {
			t.Errorf("sin pendingArchive no hay insignia (caso %d): %s", i, got)
		}
	}
	if strings.Contains(res.Branch, "<b>") || !strings.Contains(res.Branch, "inc/&lt;b&gt;x&lt;/b&gt;") || !strings.Contains(res.Branch, "&quot;x") {
		t.Errorf("la rama y la ruta del worktree se escapan: %s", res.Branch)
	}
	if !strings.Contains(res.Branch, "Worktree:") {
		t.Errorf("la insignia de rama indica el worktree en su título: %s", res.Branch)
	}
	for i, got := range res.NoBranch {
		if got != "" {
			t.Errorf("sin rama no hay insignia (caso %d): %s", i, got)
		}
	}
}

// api.js expone las cabeceras de la respuesta (los avisos viajan en `X-Axiom-Notes`) sin romper el
// agrupamiento de GET simultáneos, y el cuerpo de un 409 llega entero en el ApiError.
func TestAPIClientExposesHeadersAndConflictBodies(t *testing.T) {
	out := runNodeModules(t, []string{"api"},
		"import { api, apiWithHeaders, ApiError } from './api.mjs';\n"+
			"let calls = 0;\n"+
			"const reply = (status, body, headers = {}) => new Response(body === null ? null : JSON.stringify(body), { status, headers });\n"+
			"globalThis.fetch = async (path, init) => {\n"+
			"  calls++;\n"+
			"  if (path === '/list') { await new Promise(r => setTimeout(r, 10)); return reply(200, [{ name: 'a' }], { 'X-Axiom-Notes': '[\"aviso\"]' }); }\n"+
			"  if (path === '/decision') return reply(409, { status: 'decision_required', statusLabel: 'Hace falta una decisión', code: 'decision_required', decision: { question: '¿Cómo?', choices: [{ answer: 'worktree', label: 'W', effect: 'e' }] } });\n"+
			"  if (path === '/refusal') return reply(409, { code: 'wrong_branch', error: 'Estás en otra rama.', continuation: 'git switch x' });\n"+
			"  if (path === '/label') return reply(409, { statusLabel: 'Solo la etiqueta' });\n"+
			"  return reply(200, null);\n"+
			"};\n"+
			"const [a, b, c] = await Promise.all([apiWithHeaders('/list'), apiWithHeaders('/list'), api('/list')]);\n"+
			"const shared = { calls, notes: a.headers.get('X-Axiom-Notes'), data: a.data, same: a === b, plain: c };\n"+
			"const fail = async (path) => { try { await api(path, { body: {} }); return null; } catch (e) { return { isApiError: e instanceof ApiError, status: e.status, message: e.message, code: e.data && e.data.code, choices: e.data && e.data.decision ? e.data.decision.choices.length : 0, continuation: e.data && e.data.continuation }; } };\n"+
			"const nobody = await apiWithHeaders('/vacio');\n"+
			"console.log(JSON.stringify({ shared, decision: await fail('/decision'), refusal: await fail('/refusal'), label: await fail('/label'), nobody: { data: nobody.data, hasHeaders: typeof nobody.headers.get === 'function' } }));\n")
	var res struct {
		Shared struct {
			Calls int                 `json:"calls"`
			Notes string              `json:"notes"`
			Data  []map[string]string `json:"data"`
			Same  bool                `json:"same"`
			Plain []map[string]string `json:"plain"`
		} `json:"shared"`
		Decision, Refusal, Label *struct {
			IsAPIError   bool   `json:"isApiError"`
			Status       int    `json:"status"`
			Message      string `json:"message"`
			Code         string `json:"code"`
			Choices      int    `json:"choices"`
			Continuation string `json:"continuation"`
		}
		Nobody struct {
			Data       any  `json:"data"`
			HasHeaders bool `json:"hasHeaders"`
		} `json:"nobody"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if res.Shared.Calls != 1 || res.Shared.Notes != `["aviso"]` || len(res.Shared.Data) != 1 || len(res.Shared.Plain) != 1 {
		t.Errorf("tres GET simultáneos de la misma ruta comparten una petición y conservan las cabeceras: %+v", res.Shared)
	}
	if d := res.Decision; d == nil || !d.IsAPIError || d.Status != 409 || d.Code != "decision_required" || d.Choices != 1 || d.Message != "Hace falta una decisión" {
		t.Errorf("el 409 decision_required debe llegar como ApiError con su cuerpo y su etiqueta como mensaje: %+v", d)
	}
	if r := res.Refusal; r == nil || r.Message != "Estás en otra rama." || r.Continuation != "git switch x" || r.Code != "wrong_branch" {
		t.Errorf("el 409 de un rechazo conserva el error y la continuación: %+v", r)
	}
	if l := res.Label; l == nil || l.Message != "Solo la etiqueta" {
		t.Errorf("sin `error` ni `message` se usa `statusLabel`: %+v", l)
	}
	if res.Nobody.Data != nil || !res.Nobody.HasHeaders {
		t.Errorf("una respuesta sin cuerpo devuelve data null y sus cabeceras: %+v", res.Nobody)
	}
}

// REQ-25.32: ante un 409 `decision_required` la interfaz presenta la decisión en un diálogo accesible y
// repite la creación con `mode`.
func TestNewIncrementAsksTheDecisionInAnAccessibleDialog(t *testing.T) {
	html, js := readAsset(t, "index.html"), readAsset(t, "js/screens/incrementos.js")

	dialog := regexp.MustCompile(`<dialog[^>]*id="dlg-increment-decision"[^>]*>`).FindString(html)
	if dialog == "" {
		t.Fatal("index.html debe contener el <dialog> de decisión `dlg-increment-decision`")
	}
	if !strings.Contains(dialog, "aria-labelledby=") {
		t.Errorf("el diálogo de decisión debe tener nombre accesible: %s", dialog)
	}
	for _, id := range []string{"inc-decision-title", "inc-decision-question", "inc-decision-choices", "btn-cancel-inc-decision"} {
		if !strings.Contains(html, `id="`+id+`"`) {
			t.Errorf("index.html debe contener #%s del diálogo de decisión", id)
		}
	}
	if !strings.Contains(html, "¿Cómo quieres trabajar este cambio?") {
		t.Error("el diálogo de decisión debe titularse «¿Cómo quieres trabajar este cambio?»")
	}

	// El alta usa la creación con reintento y el servidor es quien decide (D5: elegir es un botón).
	submit := funcBody(t, js, "initNewIncrement")
	if !strings.Contains(submit, "createWithDecision(") || !strings.Contains(submit, "askIncrementDecision") {
		t.Errorf("el alta del incremento debe usar createWithDecision con askIncrementDecision:\n%s", submit)
	}
	// La decisión viene del servidor: se pinta con textContent y el foco entra en el diálogo.
	ask := funcBody(t, js, "askIncrementDecision")
	for _, marker := range []string{"decision.question", "decision.choices", "choice.answer", "choice.label", "choice.effect", "textContent", "openDialog(", "dataset.answer"} {
		if !strings.Contains(ask, marker) {
			t.Errorf("askIncrementDecision debe contener %q:\n%s", marker, ask)
		}
	}
	if strings.Contains(ask, "innerHTML") {
		t.Errorf("askIncrementDecision no debe pintar la decisión como HTML:\n%s", ask)
	}
	// Cancelar, Escape y pulsar fuera resuelven con «sin respuesta».
	if !strings.Contains(ask, "'close'") || !strings.Contains(ask, "resolve(") {
		t.Errorf("askIncrementDecision debe resolver al cerrarse el diálogo:\n%s", ask)
	}
	// No queda ningún `fetch` propio ni la lógica del reintento duplicada en la pantalla.
	if strings.Contains(js, "fetch(") || strings.Contains(js, "decision_required") {
		t.Errorf("incrementos.js delega la creación con reintento en core/branches.js")
	}
}

// createWithDecision se ejecuta de verdad: un único reintento con `mode`, y cada rechazo con su texto.
func TestCreateWithDecisionRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"api", "icons", "components", "branches"},
		"import { ApiError } from './api.mjs';\n"+
			"import { createWithDecision } from './branches.mjs';\n"+
			"const decision = { question: '¿Cómo?', choices: [{ answer: 'worktree', label: 'W', effect: 'e' }, { answer: 'in_place', label: 'I', effect: 'f' }] };\n"+
			"const needs = () => new ApiError('Hace falta una decisión', 409, { code: 'decision_required', decision });\n"+
			"const run = async (replies, answer) => {\n"+
			"  const posted = []; const asked = [];\n"+
			"  const post = async (payload) => { posted.push(payload); const r = replies.shift(); if (r instanceof Error) throw r; return r; };\n"+
			"  const ask = async (d) => { asked.push(d.question); return answer; };\n"+
			"  let result = null; let error = null;\n"+
			"  try { result = await createWithDecision(post, ask, { name: 'inc-1', intent: 'x' }); } catch (e) { error = e.message; }\n"+
			"  return { posted, asked, result, error };\n"+
			"};\n"+
			"console.log(JSON.stringify({\n"+
			"  direct: await run([{ success: true, path: '/p' }], 'worktree'),\n"+
			"  retry: await run([needs(), { success: true, path: '/q' }], 'in_place'),\n"+
			"  cancelled: await run([needs()], null),\n"+
			"  again: await run([needs(), needs()], 'worktree'),\n"+
			"  refusal: await run([new ApiError('Estás en otra rama.', 409, { code: 'wrong_branch', error: 'Estás en otra rama.', continuation: 'git switch x' })], 'worktree'),\n"+
			"  refusalAfter: await run([needs(), new ApiError('Fallo', 409, { error: 'Fallo', continuation: 'axiom branch status' })], 'worktree'),\n"+
			"  plain: await run([new ApiError('No se pudo', 400, { error: 'No se pudo' })], 'worktree'),\n"+
			"  badBody: await run([new ApiError('Conflict', 409, { code: 'decision_required' })], 'worktree'),\n"+
			"  notOk: await run([{ success: false, error: 'ya existe' }], 'worktree'),\n"+
			"}));\n")
	type outcome struct {
		Posted []map[string]string `json:"posted"`
		Asked  []string            `json:"asked"`
		Result map[string]any      `json:"result"`
		Error  string              `json:"error"`
	}
	var res map[string]outcome
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if d := res["direct"]; len(d.Posted) != 1 || len(d.Asked) != 0 || d.Error != "" || d.Posted[0]["mode"] != "" {
		t.Errorf("sin decisión pendiente se crea una sola vez y sin preguntar: %+v", d)
	}
	if r := res["retry"]; len(r.Posted) != 2 || len(r.Asked) != 1 || r.Posted[1]["mode"] != "in_place" || r.Posted[1]["name"] != "inc-1" || r.Posted[0]["mode"] != "" || r.Error != "" || r.Result["path"] != "/q" {
		t.Errorf("tras el 409 se repite una vez con `mode` y el resto del cuerpo intacto: %+v", r)
	}
	if c := res["cancelled"]; len(c.Posted) != 1 || !strings.Contains(c.Error, "Creación cancelada") {
		t.Errorf("sin respuesta no se repite y se avisa de la cancelación: %+v", c)
	}
	if a := res["again"]; len(a.Posted) != 2 || !strings.Contains(a.Error, "volvió a pedir la decisión") {
		t.Errorf("si el servidor vuelve a pedir la decisión no se entra en bucle: %+v", a)
	}
	if r := res["refusal"]; len(r.Posted) != 1 || !strings.Contains(r.Error, "Estás en otra rama.") || !strings.Contains(r.Error, "Continuación: git switch x") {
		t.Errorf("un rechazo 409 muestra su error y su continuación: %+v", r)
	}
	if r := res["refusalAfter"]; len(r.Posted) != 2 || !strings.Contains(r.Error, "Continuación: axiom branch status") {
		t.Errorf("un rechazo tras la decisión también muestra su continuación: %+v", r)
	}
	if p := res["plain"]; len(p.Posted) != 1 || p.Error != "No se pudo" {
		t.Errorf("un error sin continuación pasa tal cual: %+v", p)
	}
	if b := res["badBody"]; len(b.Asked) != 0 || b.Error != "Conflict" {
		t.Errorf("un decision_required sin decisión utilizable no abre el diálogo: %+v", b)
	}
	if n := res["notOk"]; n.Error != "ya existe" {
		t.Errorf("success:false se trata como error con el texto del servidor: %+v", n)
	}
}

// REQ-1.1 y REQ-25.33: la política de ramas y los ajustes por repositorio se muestran en solo lectura.
func TestBranchPolicyIsShownReadOnlyInEcosystem(t *testing.T) {
	html, eco, branches := readAsset(t, "index.html"), readAsset(t, "js/screens/ecosystem.js"), readAsset(t, "js/core/branches.js")

	topology := funcBody(t, eco, "topologyHtml")
	if !strings.Contains(topology, "branchPolicyRows(ws)") || !strings.Contains(topology, "repoPolicyHtml(ws)") {
		t.Errorf("la tarjeta Topología debe mostrar la política de ramas y los ajustes por repositorio:\n%s", topology)
	}
	for _, marker := range []string{"ws.branchPolicy", "integrationBranch", "branchPrefix", "ws.repositories", "hereda del workspace"} {
		if !strings.Contains(branches, marker) {
			t.Errorf("branches.js debe contener %q", marker)
		}
	}
	// Solo lectura: ni controles de edición de `git:` ni una escritura de la política.
	if strings.Contains(html, "integration_branch") || strings.Contains(html, "branch_prefix") || strings.Contains(branches, "integration_branch") {
		t.Error("la interfaz no debe ofrecer controles de edición de `git:` (REQ-25.33)")
	}
	if loc := regexp.MustCompile(`(?s)<(input|select|textarea)[^>]*id="[^"]*(integration|branch-prefix)[^"]*"`).FindString(html); loc != "" {
		t.Errorf("la política de ramas es de solo lectura, no debe tener controles: %s", loc)
	}
	if regexp.MustCompile(`method:\s*'(PUT|PATCH|DELETE)'`).MatchString(branches) || strings.Contains(branches, "api(") {
		t.Errorf("branches.js no escribe la política de ramas")
	}
}

// Los dos bloques de la política se pintan con datos hostiles y con los huecos que deja el servidor.
func TestBranchPolicyRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"api", "icons", "components", "branches"},
		"import { branchPolicyRows, repoPolicyHtml } from './branches.mjs';\n"+
			"const ws = { branchPolicy: { integrationBranch: 'develop', branchPrefix: 'inc/' }, repositories: [\n"+
			"  { path: 'repo-specs', integrationBranch: 'release/<i>' },\n"+
			"  { path: 'src/web', branchPrefix: 'web/' },\n"+
			"  { path: '.' } ] };\n"+
			"console.log(JSON.stringify({\n"+
			"  rows: branchPolicyRows(ws),\n"+
			"  repos: repoPolicyHtml(ws),\n"+
			"  emptyRows: [branchPolicyRows({}), branchPolicyRows(null), branchPolicyRows({ branchPolicy: {} })],\n"+
			"  noRepos: [repoPolicyHtml({ repositories: [] }), repoPolicyHtml({}), repoPolicyHtml(null)],\n"+
			"}));\n")
	var res struct {
		Rows      string   `json:"rows"`
		Repos     string   `json:"repos"`
		EmptyRows []string `json:"emptyRows"`
		NoRepos   []string `json:"noRepos"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	for _, want := range []string{"Rama de integración", "<code>develop</code>", "Prefijo de ramas de cambio", "<code>inc/</code>"} {
		if !strings.Contains(res.Rows, want) {
			t.Errorf("la política efectiva debe contener %q: %s", want, res.Rows)
		}
	}
	for _, want := range []string{"Ajustes de ramas por repositorio", "<code>repo-specs</code>", "release/&lt;i&gt;", "<code>web/</code>", "<table"} {
		if !strings.Contains(res.Repos, want) {
			t.Errorf("la tabla por repositorio debe contener %q: %s", want, res.Repos)
		}
	}
	if strings.Contains(res.Repos, "<i>") {
		t.Errorf("los nombres de rama se escapan: %s", res.Repos)
	}
	if got := strings.Count(res.Repos, "hereda del workspace"); got != 4 {
		t.Errorf("cada hueco hereda del workspace (4 huecos), hay %d: %s", got, res.Repos)
	}
	for i, got := range res.EmptyRows {
		if !strings.Contains(got, "Rama de integración") || !strings.Contains(got, "—") || strings.Contains(got, "undefined") {
			t.Errorf("sin política el valor es «—» (caso %d): %s", i, got)
		}
	}
	for i, got := range res.NoRepos {
		if got != "" {
			t.Errorf("sin repositorios no hay tabla (caso %d): %s", i, got)
		}
	}
}

// REQ-25.29 y REQ-25.33: las preferencias personales de worktree del proyecto activo se editan y se
// guardan en el Hub desde Ecosistema, y el alta de un proyecto las pregunta.
func TestWorktreePrefsEditorInEcosystem(t *testing.T) {
	html, eco := readAsset(t, "index.html"), readAsset(t, "js/screens/ecosystem.js")
	start, end := strings.Index(html, `id="view-ecosistema"`), strings.Index(html, "</main>")
	if start < 0 || end < start {
		t.Fatal("index.html: no se localiza la vista de Ecosistema")
	}
	view := html[start:end]

	selectBlock := regexp.MustCompile(`(?s)<select id="select-worktree-prefs-mode".*?</select>`).FindString(view)
	if selectBlock == "" {
		t.Fatal("la vista de Ecosistema no tiene el selector `select-worktree-prefs-mode`")
	}
	for _, option := range []string{`value=""`, `value="worktree"`, `value="in_place"`, `value="ask"`} {
		if !strings.Contains(selectBlock, option) {
			t.Errorf("el selector de modo debe contener %q:\n%s", option, selectBlock)
		}
	}
	for _, id := range []string{"input-worktree-prefs-root", "btn-save-worktree-prefs", "worktree-prefs-effective", "worktree-prefs-status", "eco-worktree-body", "form-worktree-prefs"} {
		if !strings.Contains(view, `id="`+id+`"`) {
			t.Errorf("la vista de Ecosistema no contiene #%s", id)
		}
	}
	// Cada control tiene su etiqueta, y el resultado se anuncia.
	for _, id := range []string{"select-worktree-prefs-mode", "input-worktree-prefs-root"} {
		if !strings.Contains(view, `for="`+id+`"`) {
			t.Errorf("el control #%s necesita un <label for>", id)
		}
	}
	if status := regexp.MustCompile(`<[a-z]+[^>]*id="worktree-prefs-status"[^>]*>`).FindString(view); !strings.Contains(status, `role="status"`) {
		t.Errorf("el resultado del guardado debe anunciarse con role=\"status\": %s", status)
	}
	if !strings.Contains(view, "~/.axiom/workspaces.json") || !strings.Contains(view, "axiom.yaml") {
		t.Error("la tarjeta debe decir que se guarda en el Hub y no modifica axiom.yaml")
	}

	// El servidor guarda: es un botón (D5), por PUT, y a continuación se refresca el selector de proyecto.
	save := funcBody(t, eco, "saveWorktreePrefs")
	for _, marker := range []string{"/api/projects/active/worktree-prefs", "method: 'PUT'", "loadProjects()", "toast("} {
		if !strings.Contains(save, marker) {
			t.Errorf("saveWorktreePrefs debe contener %q:\n%s", marker, save)
		}
	}
	if load := funcBody(t, eco, "loadWorktreePrefs"); !strings.Contains(load, "/api/projects/active/worktree-prefs") || !strings.Contains(load, "stateError(") || !strings.Contains(load, "404") {
		t.Errorf("loadWorktreePrefs debe leer las preferencias y distinguir un proyecto sin registrar de un fallo:\n%s", load)
	}
	if !strings.Contains(funcBody(t, eco, "loadEcosystem"), "loadWorktreePrefs()") {
		t.Error("Actualizar y cambiar de proyecto deben recargar las preferencias (loadEcosystem)")
	}
}

// REQ-25.29: el modal de registro pregunta el modo y la raíz de worktree, y la petición los envía.
func TestRegistrationModalAsksWorktreePrefsInTheRedesign(t *testing.T) {
	html, projects := readAsset(t, "index.html"), readAsset(t, "js/projects.js")

	selectBlock := regexp.MustCompile(`(?s)<select id="select-project-worktree-mode".*?</select>`).FindString(html)
	if selectBlock == "" {
		t.Fatal("index.html: el modal de registro no tiene el selector de modo `select-project-worktree-mode`")
	}
	for _, option := range []string{`value="worktree" selected`, `value="in_place"`, `value="ask"`} {
		if !strings.Contains(selectBlock, option) {
			t.Errorf("el selector de modo debe contener %q:\n%s", option, selectBlock)
		}
	}
	if got := strings.Count(selectBlock, "<option"); got != 3 {
		t.Errorf("el selector de modo tiene %d opciones, quiero 3", got)
	}
	rootInput := regexp.MustCompile(`<input[^>]*id="input-project-worktree-root"[^>]*>`).FindString(html)
	if rootInput == "" {
		t.Fatal("index.html: el modal de registro no tiene el campo `input-project-worktree-root`")
	}
	if !strings.Contains(rootInput, "placeholder=") || !strings.Contains(rootInput, "por defecto") || !strings.Contains(rootInput, "-wt") {
		t.Errorf("el campo de raíz debe mostrar su valor por defecto en el placeholder: %s", rootInput)
	}
	for _, id := range []string{"select-project-worktree-mode", "input-project-worktree-root"} {
		if !strings.Contains(html, `for="`+id+`"`) {
			t.Errorf("el control #%s necesita un <label for>", id)
		}
	}
	// Dentro del modal de alta, no en otro sitio.
	modal := html[strings.Index(html, `id="modal-add-project"`):]
	modal = modal[:strings.Index(modal, "</dialog>")]
	if !strings.Contains(modal, `id="select-project-worktree-mode"`) || !strings.Contains(modal, `id="input-project-worktree-root"`) {
		t.Error("los campos de worktree deben estar dentro del modal de registro de proyecto")
	}

	for _, marker := range []string{"registrationPrefs(", "defaultWorktreeRoot(", "#select-project-worktree-mode", "#input-project-worktree-root"} {
		if !strings.Contains(projects, marker) {
			t.Errorf("projects.js debe contener %q", marker)
		}
	}
	submit := regexp.MustCompile(`(?s)api\('/api/projects/init',.*?\}\);`).FindString(projects)
	if !strings.Contains(submit, "...registrationPrefs(") {
		t.Errorf("la petición de alta del modal debe enviar las preferencias elegidas:\n%s", submit)
	}
	if open := funcBody(t, projects, "initAddProject"); !strings.Contains(open, "'worktree'") || !strings.Contains(open, "updateWorktreeRootPlaceholder()") {
		t.Errorf("abrir el modal restablece el modo por defecto y el marcador de la raíz:\n%s", open)
	}
}

// La lógica de las preferencias se ejecuta de verdad (si hay Node).
func TestWorktreePrefsLogicRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"api", "icons", "components", "branches"},
		"import { defaultWorktreeRoot, registrationPrefs, effectivePrefsText } from './branches.mjs';\n"+
			"console.log(JSON.stringify({\n"+
			"  roots: ['C:\\\\repos\\\\mi-proyecto', 'C:\\\\repos\\\\mi-proyecto\\\\', '/home/u/proy/', '/home/u/proy', '  /a/b  ', 'C:\\\\', 'C:', '/', '', null, 'relativa', undefined].map(defaultWorktreeRoot),\n"+
			"  prefs: [registrationPrefs('worktree', ''), registrationPrefs('', ''), registrationPrefs('in_place', ''), registrationPrefs('ask', '  /x/wt '), registrationPrefs('worktree', 'C:\\\\wt'), registrationPrefs(null, null)],\n"+
			"  text: [effectivePrefsText({ mode: 'ask', root: '/x-wt' }), effectivePrefsText(null), effectivePrefsText({})],\n"+
			"}));\n")
	var res struct {
		Roots []string            `json:"roots"`
		Prefs []map[string]string `json:"prefs"`
		Text  []string            `json:"text"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	wantRoots := []string{`C:\repos\mi-proyecto-wt`, `C:\repos\mi-proyecto-wt`, "/home/u/proy-wt", "/home/u/proy-wt", "/a/b-wt", "", "", "", "", "", "", ""}
	for i, want := range wantRoots {
		if i < len(res.Roots) && res.Roots[i] != want {
			t.Errorf("defaultWorktreeRoot caso %d = %q, quiero %q", i, res.Roots[i], want)
		}
	}
	if len(res.Roots) != len(wantRoots) {
		t.Fatalf("defaultWorktreeRoot devolvió %d resultados, quiero %d", len(res.Roots), len(wantRoots))
	}
	wantPrefs := []map[string]string{
		{},
		{},
		{"worktree_mode": "in_place"},
		{"worktree_mode": "ask", "worktree_root": "/x/wt"},
		{"worktree_root": `C:\wt`},
		{},
	}
	for i, want := range wantPrefs {
		got := res.Prefs[i]
		if len(got) != len(want) {
			t.Errorf("registrationPrefs caso %d = %v, quiero %v", i, got, want)
			continue
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("registrationPrefs caso %d = %v, quiero %v", i, got, want)
			}
		}
	}
	if res.Text[0] != "Vigentes: modo ask, raíz /x-wt" || res.Text[1] != "" || res.Text[2] != "" {
		t.Errorf("effectivePrefsText = %q", res.Text)
	}
}
