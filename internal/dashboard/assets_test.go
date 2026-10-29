package dashboard

import (
	"io/fs"
	"mime"
	"net/http"
	"net/http/httptest"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Las pruebas de este fichero cubren los recursos estáticos del rediseño: que se
// sirven con su Content-Type, que los módulos ES se resuelven entre sí, que la
// estructura (barra lateral, rutas, tema) existe y que no hay recursos externos.
// No inspeccionan estilos concretos: eso lo cubren la revisión visual y
// tokens_test.go (contraste).

func assetServer(t *testing.T) http.Handler {
	t.Helper()
	return NewServer(NewService(t.TempDir())).Router()
}

func getAsset(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
	return rr
}

// embeddedFiles devuelve las rutas (relativas a assets/) de todos los ficheros embebidos.
func embeddedFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := fs.WalkDir(AssetsFS, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, strings.TrimPrefix(p, "assets/"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer los recursos embebidos: %v", err)
	}
	sort.Strings(files)
	return files
}

func readAsset(t *testing.T, name string) string {
	t.Helper()
	raw, err := AssetsFS.ReadFile("assets/" + name)
	if err != nil {
		t.Fatalf("leer assets/%s: %v", name, err)
	}
	return string(raw)
}

// readAllJS concatena todo el JavaScript embebido, para comprobaciones de
// comportamiento que no dependen de en qué módulo viva la lógica.
func readAllJS(t *testing.T) string {
	t.Helper()
	var sb strings.Builder
	for _, f := range embeddedFiles(t) {
		if strings.HasSuffix(f, ".js") {
			sb.WriteString(readAsset(t, f))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func TestAssetsAreServedWithTheirContentType(t *testing.T) {
	h := assetServer(t)

	want := map[string]string{
		"/":            "text/html",
		"/tokens.css":  "text/css",
		"/style.css":   "text/css",
		"/legacy.css":  "text/css",
		"/js/main.js":  "text/javascript",
		"/index.html":  "", // el FileServer redirige /index.html a /
		"/no-existe.x": "",
	}
	for target, typ := range want {
		rr := getAsset(t, h, target)
		switch target {
		case "/index.html":
			if rr.Code != http.StatusMovedPermanently {
				t.Errorf("GET %s = %d, se esperaba la redirección a /", target, rr.Code)
			}
		case "/no-existe.x":
			if rr.Code != http.StatusNotFound {
				t.Errorf("GET %s = %d, se esperaba 404", target, rr.Code)
			}
		default:
			if rr.Code != http.StatusOK {
				t.Errorf("GET %s = %d, se esperaba 200", target, rr.Code)
			}
			if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, typ) {
				t.Errorf("GET %s: Content-Type = %q, se esperaba %q", target, got, typ)
			}
		}
	}

	// Todo fichero embebido (cada módulo JS incluido) debe ser alcanzable.
	for _, f := range embeddedFiles(t) {
		if f == "index.html" {
			continue
		}
		rr := getAsset(t, h, "/"+f)
		if rr.Code != http.StatusOK {
			t.Errorf("GET /%s = %d, se esperaba 200", f, rr.Code)
			continue
		}
		got := rr.Header().Get("Content-Type")
		switch {
		case strings.HasSuffix(f, ".js") && !strings.HasPrefix(got, "text/javascript"):
			t.Errorf("GET /%s: Content-Type = %q; un módulo ES exige text/javascript", f, got)
		case strings.HasSuffix(f, ".css") && !strings.HasPrefix(got, "text/css"):
			t.Errorf("GET /%s: Content-Type = %q, se esperaba text/css", f, got)
		}
	}
}

// Las fuentes .woff2 se añadirán en assets/fonts/: el tipo MIME debe estar
// registrado de antemano para que se sirvan como font/woff2.
func TestWoff2MimeTypeIsRegistered(t *testing.T) {
	if got := mime.TypeByExtension(".woff2"); got != "font/woff2" {
		t.Errorf("mime.TypeByExtension(.woff2) = %q, se esperaba font/woff2", got)
	}
}

var importPattern = regexp.MustCompile(`(?m)^\s*import\s[^;]*?from\s+'([^']+)'`)

// Cada import relativo de un módulo debe apuntar a un fichero embebido: sin
// navegador en las pruebas, es la comprobación que atrapa una ruta rota.
func TestJSModuleImportsResolve(t *testing.T) {
	modules := 0
	for _, f := range embeddedFiles(t) {
		if !strings.HasSuffix(f, ".js") {
			continue
		}
		modules++
		src := readAsset(t, f)
		for _, m := range importPattern.FindAllStringSubmatch(src, -1) {
			spec := m[1]
			if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
				t.Errorf("%s importa %q: solo se admiten rutas relativas, sin dependencias externas", f, spec)
				continue
			}
			target := path.Join(path.Dir(f), spec)
			if _, err := fs.Stat(AssetsFS, "assets/"+target); err != nil {
				t.Errorf("%s importa %q, que no existe (%s)", f, spec, target)
			}
		}
	}
	if modules < 8 {
		t.Errorf("se esperaban al menos 8 módulos JS embebidos y hay %d", modules)
	}
}

func TestIndexHTMLShell(t *testing.T) {
	h := assetServer(t)
	rr := getAsset(t, h, "/")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET / = %d", rr.Code)
	}
	body := rr.Body.String()

	// Hojas de estilo y módulo de entrada, con tokens.css antes que style.css.
	order := []string{`href="/tokens.css"`, `href="/style.css"`, `<script type="module" src="/js/main.js">`}
	last := -1
	for _, marker := range order {
		idx := strings.Index(body, marker)
		if idx < 0 {
			t.Errorf("index.html no contiene %q", marker)
			continue
		}
		if idx < last {
			t.Errorf("%q aparece fuera de orden", marker)
		}
		last = idx
	}

	// Script anti-parpadeo: antes de la primera hoja de estilo y con la lógica de tema.
	script := strings.Index(body, "axiom.theme")
	if script < 0 || script > strings.Index(body, `href="/tokens.css"`) {
		t.Errorf("el script anti-parpadeo debe estar en <head> antes de las hojas de estilo")
	}
	for _, marker := range []string{"matchMedia('(prefers-color-scheme: dark)')", "dataset.theme", "try {"} {
		if !strings.Contains(body, marker) {
			t.Errorf("el script anti-parpadeo no contiene %q", marker)
		}
	}

	// Estructura: tres grupos, rutas por hash, miga, Actualizar con aria-label y selector de tema.
	for _, marker := range []string{
		`aria-label="Navegación principal"`,
		"Trabajo", "Conocimiento", "Plataforma",
		`href="#/inicio"`, `href="#/incrementos"`, `href="#/specs"`, `href="#/skills"`, `href="#/codigo"`, `href="#/ecosistema"`,
		`aria-label="Ruta"`,
		`id="btn-refresh" class="btn btn-icon" aria-label="Actualizar datos"`,
		"Nuevo incremento",
		"Tema · se guarda para ti",
		`data-theme-pref="light"`, `data-theme-pref="dark"`, `data-theme-pref="system"`,
		`id="project-trigger"`, `aria-haspopup="true"`, `aria-expanded="false"`,
		`id="zero-config-hero"`,
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("index.html no contiene %q", marker)
		}
	}

	// Lo antiguo ya no existe: sin pestañas planas ni clase de tema fija.
	for _, gone := range []string{"modern-light-theme", `class="nav-tab`, "data-tab=\"tab-", "role-change-select", "handoff-change-select"} {
		if strings.Contains(body, gone) {
			t.Errorf("index.html todavía contiene %q", gone)
		}
	}

	// Cada diálogo es un <dialog> con nombre accesible.
	for _, id := range []string{"modal-folder-picker", "modal-add-project", "modal-new-increment", "modal-create-handoff", "dlg-confirm"} {
		re := regexp.MustCompile(`<dialog id="` + id + `"[^>]*aria-labelledby=`)
		if !re.MatchString(body) {
			t.Errorf("el diálogo %q debe ser un <dialog> con aria-labelledby", id)
		}
	}
	for _, marker := range []string{"btn-browse-project-path", "btn-browse-header-project"} {
		if !strings.Contains(body, marker) {
			t.Errorf("index.html debe contener %q (selector de carpetas)", marker)
		}
	}
}

func TestRouterDefinesEveryRoute(t *testing.T) {
	router := readAsset(t, "js/core/router.js")
	for _, marker := range []string{
		"'inicio'", "'incrementos'", "'specs'", "'skills'", "'codigo'", "'ecosistema'",
		"'artefactos'", "'roles'", "'handoff'",
		"#/inicio", "hashchange", "#/incrementos/",
	} {
		if !strings.Contains(router, marker) {
			t.Errorf("router.js no contiene %q", marker)
		}
	}
}

func TestThemeModule(t *testing.T) {
	theme := readAsset(t, "js/core/theme.js")
	for _, marker := range []string{
		"'axiom.theme'", "'light'", "'dark'", "'system'",
		"prefers-color-scheme: dark", "addEventListener('change'", "aria-pressed", "dataset.theme",
	} {
		if !strings.Contains(theme, marker) {
			t.Errorf("theme.js no contiene %q", marker)
		}
	}
	// Todo acceso a localStorage va protegido.
	if got, want := strings.Count(theme, "localStorage"), strings.Count(theme, "try {"); got > want+1 {
		t.Errorf("theme.js: %d accesos a localStorage para %d bloques try", got, want)
	}
}

// El dashboard debe funcionar sin red: ningún recurso apunta a un servidor externo.
func TestAssetsHaveNoExternalResources(t *testing.T) {
	external := regexp.MustCompile(`(?i)(href|src)\s*=\s*["']https?://|url\(\s*["']?https?://|@import\s+["']?https?://|fonts\.googleapis|cdn\.|unpkg\.com|jsdelivr`)
	for _, f := range embeddedFiles(t) {
		if !strings.HasSuffix(f, ".html") && !strings.HasSuffix(f, ".css") && !strings.HasSuffix(f, ".js") {
			continue
		}
		if loc := external.FindString(readAsset(t, f)); loc != "" {
			t.Errorf("%s referencia un recurso externo: %q", f, loc)
		}
	}
}

// Ninguna vista usa emojis como iconos: el conjunto SVG de icons.js los sustituye.
func TestAssetsDoNotUseEmojis(t *testing.T) {
	emoji := regexp.MustCompile("[\U0001F300-\U0001FAFF☀-➿⏩-⏿]")
	for _, f := range embeddedFiles(t) {
		if !strings.HasSuffix(f, ".html") && !strings.HasSuffix(f, ".css") && !strings.HasSuffix(f, ".js") {
			continue
		}
		if loc := emoji.FindString(readAsset(t, f)); loc != "" {
			t.Errorf("%s contiene el emoji %q", f, loc)
		}
	}
}

// Se conserva la cobertura del selector de carpetas: la lógica vive ahora en
// js/projects.js y el estilo en style.css.
func TestFolderPickerIsServed(t *testing.T) {
	h := assetServer(t)

	js := getAsset(t, h, "/js/projects.js").Body.String()
	for _, marker := range []string{"openFolderPicker", "btn-browse-role-repo", "folderPickerState", "/api/fs/directories", "/api/fs/native-picker"} {
		if !strings.Contains(js, marker) {
			t.Errorf("js/projects.js debe contener %q", marker)
		}
	}
	css := getAsset(t, h, "/style.css").Body.String()
	for _, marker := range []string{".fp-list", ".fp-item", ".fp-breadcrumbs", ".project-popover"} {
		if !strings.Contains(css, marker) {
			t.Errorf("style.css debe contener %q", marker)
		}
	}
}

// Accesibilidad: nada quita el contorno de foco sin sustituirlo.
func TestStylesKeepVisibleFocus(t *testing.T) {
	css := readAsset(t, "style.css")
	if !strings.Contains(css, ":focus-visible { outline: 2px solid var(--accent)") {
		t.Errorf("style.css debe definir el foco visible de 2 px en el acento")
	}
	for _, name := range []string{"style.css", "legacy.css"} {
		if strings.Contains(readAsset(t, name), "outline: 0") || strings.Contains(readAsset(t, name), "outline: none") {
			// Se admite solo en contenedores no interactivos enfocados por programa (tabindex -1).
			for _, line := range strings.Split(readAsset(t, name), "\n") {
				if (strings.Contains(line, "outline: 0") || strings.Contains(line, "outline: none")) && !strings.Contains(line, "[tabindex=\"-1\"]") && !strings.Contains(line, ".main:focus") {
					t.Errorf("%s quita el contorno de foco sin sustituirlo: %s", name, strings.TrimSpace(line))
				}
			}
		}
	}
}

// El helper api() es el único punto de fetch del código nuevo y normaliza los errores.
func TestAPIHelperNormalisesErrors(t *testing.T) {
	api := readAsset(t, "js/core/api.js")
	for _, marker := range []string{"export async function api(", "class ApiError", "data.error", "data.message", "export function escapeHtml("} {
		if !strings.Contains(api, marker) {
			t.Errorf("api.js no contiene %q", marker)
		}
	}
}

// Los componentes comunes que reutilizarán las pantallas de T3 a T6.
func TestCommonComponentsExist(t *testing.T) {
	components := readAsset(t, "js/core/components.js")
	for _, marker := range []string{
		"export function button(", "export function iconButton(", "export function chip(", "export function badge(",
		"export function statusPill(", "export function reminder(", "function copyText(", "navigator.clipboard",
		"export function stateLoading(", "export function stateEmpty(", "export function stateError(",
		"export function tableWrap(", "export function consoleBox(", "export function confirmDialog(", "showModal",
	} {
		if !strings.Contains(components, marker) {
			t.Errorf("components.js no contiene %q", marker)
		}
	}
	icons := readAsset(t, "js/core/icons.js")
	for _, marker := range []string{`viewBox="0 0 24 24"`, `stroke="currentColor"`, "stroke-width", "aria-hidden"} {
		if !strings.Contains(icons, marker) {
			t.Errorf("icons.js no contiene %q", marker)
		}
	}
	css := readAsset(t, "style.css")
	for _, marker := range []string{".btn-primary", ".btn-danger", ".btn-ghost", ".btn-icon", ".chip", `[aria-pressed="true"]`, ".reminder", ".table-wrap", ".seg-group", "dialog.modal", ".console", ".state-error"} {
		if !strings.Contains(css, marker) {
			t.Errorf("style.css no contiene %q", marker)
		}
	}
}

// A 390 px la barra lateral se apila y se pliega, y no hay desplazamiento horizontal de página.
func TestNarrowWidthRules(t *testing.T) {
	css := readAsset(t, "style.css")
	for _, marker := range []string{"@media (max-width: 859px)", ".btn.menu-toggle { display: inline-flex; }", ".sidebar-body { display: none; }", "overflow-x: auto"} {
		if !strings.Contains(css, marker) {
			t.Errorf("style.css no contiene la regla de ancho estrecho %q", marker)
		}
	}
}
