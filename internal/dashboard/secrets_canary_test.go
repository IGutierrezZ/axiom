package dashboard

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const dashboardCanary = "s3cr3t-CANARIO-9f8e7d"

// Rutas de solo lectura que el canario recorre. Cada una se pide con GET y su cuerpo no puede
// contener el valor de ningún secreto de integración (REQ-43.16).
var coveredRoutes = map[string]string{
	"/api/health":        "/api/health",
	"/api/projects":      "/api/projects",
	"/api/workspace":     "/api/workspace",
	"/api/increments":    "/api/increments",
	"/api/increments/":   "/api/increments/x",
	"/api/roles":         "/api/roles",
	"/api/handoffs":      "/api/handoffs",
	"/api/skills":        "/api/skills",
	"/api/archive/specs": "/api/archive/specs",
}

// Rutas registradas que el canario no recorre, cada una con su motivo. Ninguna lee el bloque
// integrations: de axiom.yaml, los ficheros de integraciones de la carpeta del cambio ni el
// almacén de secretos: o mutan estado (POST) o consultan otra fuente.
var excludedRoutes = map[string]string{
	"/api/projects/switch":                "mutación (POST): cambia el proyecto activo",
	"/api/projects/add":                   "mutación (POST): registra un proyecto en el hub",
	"/api/projects/init":                  "mutación (POST): inicializa un proyecto",
	"/api/adapters":                       "lista los adaptadores de agentes del sistema; no lee el proyecto",
	"/api/project/review-mode":            "lee y fija el interruptor de revisión del hub, no axiom.yaml",
	"/api/projects/active/worktree-prefs": "preferencias personales de worktrees del hub",
	"/api/fs/directories":                 "exploración de directorios del sistema de ficheros",
	"/api/fs/native-picker":               "selector nativo de carpetas del sistema",
	"/api/workspace/specs/sync-status":    "estado de git del repositorio de specs; no lee integraciones",
	"/api/workspace/specs/pull":           "mutación (POST): git pull del repositorio de specs",
	"/api/increments/continue":            "mutación (POST): ejecuta la transición del despachador",
	"/api/increments/verify":              "mutación (POST): lanza la verificación",
	"/api/increments/migrate-cumulative":  "mutación (POST): migra el sumidero acumulativo",
	"/api/skills/inbox":                   "buzón de skills propuestas; no lee integraciones",
	"/api/skills/scan":                    "mutación (POST): escanea tecnologías",
	"/api/skills/approve":                 "mutación (POST): aprueba una skill",
	"/api/skills/reject":                  "mutación (POST): descarta una skill",
	"/api/semantic/status":                "diagnóstico de conectores semánticos",
	"/api/semantic/symbols":               "símbolos de código del índice semántico",
	"/api/semantic/dependencies":          "grafo de dependencias del índice semántico",
	"/api/semantic/reindex":               "mutación (POST): reindexa CodeGraph",
	"/api/archive/specs/":                 "especificación viva por dominio de openspec/specs; no lee integraciones",
	"/api/archive/sync":                   "mutación (POST): regenera INDEX.md",
	"/api/ecosystem/doctor":               "diagnóstico del ecosistema de la máquina",
	"/api/ecosystem/sync":                 "mutación (POST): sincroniza agentes",
	"/api/ecosystem/upgrade":              "mutación (POST): actualiza herramientas",
	"/api/ecosystem/backups":              "lista copias de seguridad de la máquina",
	"/api/ecosystem/backups/create":       "mutación (POST): crea una copia",
	"/api/ecosystem/backups/restore":      "mutación (POST): restaura una copia",
	"/api/ecosystem/models":               "catálogo de modelos de agentes",
	"/api/system/version":                 "versión del binario",
	"/api/system/update":                  "mutación (POST): actualiza Axiom",
	"/api/system/update/status":           "estado de una actualización en curso",
	"/icons/icon-192.png":                 "recurso estático",
	"/icons/icon-512.png":                 "recurso estático",
	"/icons/icon-maskable-512.png":        "recurso estático",
	"/":                                   "recursos estáticos del panel",
}

// registeredRoutes lee las rutas que registra server.go (s.mux.HandleFunc / s.mux.Handle).
func registeredRoutes(t *testing.T) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "server.go", nil, 0)
	if err != nil {
		t.Fatalf("parsear server.go: %v", err)
	}
	var routes []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (selector.Sel.Name != "HandleFunc" && selector.Sel.Name != "Handle") {
			return true
		}
		if literal, ok := call.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
			if route, err := strconv.Unquote(literal.Value); err == nil {
				routes = append(routes, route)
			}
		}
		return true
	})
	sort.Strings(routes)
	return routes
}

// La guarda falla si aparece una ruta nueva sin clasificar como cubierta o excluida con motivo.
func TestEveryDashboardRouteIsClassifiedForTheSecretsCanary(t *testing.T) {
	routes := registeredRoutes(t)
	if len(routes) < 20 {
		t.Fatalf("la lectura de server.go solo encontró %d rutas: %v", len(routes), routes)
	}
	for _, route := range routes {
		_, covered := coveredRoutes[route]
		reason, excluded := excludedRoutes[route]
		switch {
		case covered && excluded:
			t.Errorf("la ruta %s está en las dos listas", route)
		case !covered && !excluded:
			t.Errorf("la ruta %s no está clasificada: añádela a coveredRoutes (y al canario) o a excludedRoutes con su motivo", route)
		case excluded && strings.TrimSpace(reason) == "":
			t.Errorf("la ruta %s excluida necesita un motivo", route)
		}
	}
	registered := map[string]bool{}
	for _, route := range routes {
		registered[route] = true
	}
	for route := range coveredRoutes {
		if !registered[route] {
			t.Errorf("la ruta cubierta %s ya no existe en server.go", route)
		}
	}
	for route := range excludedRoutes {
		if !registered[route] {
			t.Errorf("la ruta excluida %s ya no existe en server.go", route)
		}
	}
}

func TestNoReadOnlyDashboardRouteReturnsAnIntegrationSecret(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"),
		"workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\nintegrations:\n  fake:\n    project: p\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "x", "proposal.md"), "# x\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "x", ".integrations-outbox.yaml"), "schema: axiom.integrations-outbox/v1\nchange: x\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "archive", "2026-01-01-viejo", "proposal.md"), "# viejo\n")

	memory := integrationstest.Install(t, integrationstest.NewFake())
	if err := memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(dashboardCanary)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", dashboardCanary)

	handler := NewServer(NewService(ws.Master)).Router()
	paths := make([]string, 0, len(coveredRoutes))
	for _, path := range coveredRoutes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if body := recorder.Body.String(); strings.Contains(body, dashboardCanary) {
			t.Errorf("GET %s devuelve el valor del secreto de integración", path)
		}
		for key, values := range recorder.Header() {
			for _, value := range values {
				if strings.Contains(value, dashboardCanary) {
					t.Errorf("GET %s lleva el secreto en la cabecera %s", path, key)
				}
			}
		}
	}
}
