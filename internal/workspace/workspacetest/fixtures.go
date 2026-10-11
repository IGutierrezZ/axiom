// Package workspacetest construye, bajo un directorio temporal, los espacios de
// trabajo sintéticos que usan las pruebas de resolución del planning home. Es un
// paquete solo de pruebas: ningún binario de producción lo importa.
package workspacetest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// Workspace reúne las rutas absolutas de un espacio de trabajo sintético.
type Workspace struct {
	Master   string // carpeta maestra (no versionada en el canónico)
	Specs    string // repositorio de especificaciones (workspaceRoot esperado)
	Home     string // planning home esperado: <Specs>/openspec
	Backend  string // repositorio de código backend ("" en el embebido)
	Frontend string // repositorio de código frontend ("" en el embebido)
}

// Variant ajusta el espacio de trabajo tras montarlo.
type Variant func(t testing.TB, ws *Workspace)

const canonicalConfig = `workspace:
  name: ejemplo-canonico
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
  frontend:
    name: Frontend
    repositories:
      - path: repo-frontend
`

const embeddedConfigTemplate = `workspace:
  name: ejemplo-embebido
  topology: monorepo-embedded
  specs_repository: %s
roles:
  core:
    name: Core
    repositories:
      - path: .
`

// Canonical monta el espacio canónico: una carpeta maestra con .axiom-workspace
// (config: repo-specs/axiom.yaml), el repositorio de especificaciones con su
// axiom.yaml multirepo y el cambio inc-99-ejemplo, y los repositorios de código
// repo-backend/src y repo-frontend.
func Canonical(t testing.TB, variants ...Variant) Workspace {
	t.Helper()
	master := evalDir(t, t.TempDir())
	ws := Workspace{
		Master:   master,
		Specs:    filepath.Join(master, "repo-specs"),
		Home:     filepath.Join(master, "repo-specs", "openspec"),
		Backend:  filepath.Join(master, "repo-backend"),
		Frontend: filepath.Join(master, "repo-frontend"),
	}
	WriteFile(t, filepath.Join(master, ".axiom-workspace"), "config: repo-specs/axiom.yaml\n")
	WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), canonicalConfig)
	MkdirAll(t, filepath.Join(ws.Home, "changes", "inc-99-ejemplo"))
	MkdirAll(t, filepath.Join(ws.Backend, "src"))
	MkdirAll(t, ws.Frontend)
	for _, variant := range variants {
		variant(t, &ws)
	}
	return ws
}

// Embedded monta un proyecto monorepo-embedded con axiom.yaml y openspec/. El
// parámetro specsRepository se escribe tal cual en workspace.specs_repository.
func Embedded(t testing.TB, specsRepository string, variants ...Variant) Workspace {
	t.Helper()
	root := evalDir(t, t.TempDir())
	ws := Workspace{Master: root, Specs: root, Home: filepath.Join(root, "openspec")}
	WriteFile(t, filepath.Join(root, "axiom.yaml"), embeddedConfig(specsRepository))
	MkdirAll(t, filepath.Join(ws.Home, "changes"))
	for _, variant := range variants {
		variant(t, &ws)
	}
	return ws
}

func embeddedConfig(specsRepository string) string {
	return fmt.Sprintf(embeddedConfigTemplate, strconv.Quote(specsRepository))
}

// WithGit inicializa un repositorio git en el repositorio de especificaciones y
// en los de código del canónico, para las pruebas que consultan git.
func WithGit() Variant {
	return func(t testing.TB, ws *Workspace) {
		t.Helper()
		for _, dir := range []string{ws.Specs, ws.Backend, ws.Frontend} {
			if dir != "" {
				InitGit(t, dir)
			}
		}
	}
}

// WithMasterOpenSpec crea un openspec/ propio en la carpeta maestra: con el
// puntero, la ambigüedad de REQ-24.6.
func WithMasterOpenSpec() Variant {
	return func(t testing.TB, ws *Workspace) {
		t.Helper()
		MkdirAll(t, filepath.Join(ws.Master, "openspec", "changes", "cambio-antiguo"))
	}
}

// WithBackendOpenSpec crea un openspec/ propio en repo-backend: el marcador más
// cercano desde repo-backend/src (REQ-24.2).
func WithBackendOpenSpec() Variant {
	return func(t testing.TB, ws *Workspace) {
		t.Helper()
		MkdirAll(t, filepath.Join(ws.Backend, "openspec"))
	}
}

// WithBrokenPointer apunta el puntero a un axiom.yaml inexistente.
func WithBrokenPointer() Variant {
	return func(t testing.TB, ws *Workspace) {
		t.Helper()
		WriteFile(t, filepath.Join(ws.Master, ".axiom-workspace"), "config: repo-inexistente/axiom.yaml\n")
	}
}

// WithUnreadableConfig sustituye el axiom.yaml del repositorio de
// especificaciones por un YAML con error de sintaxis.
func WithUnreadableConfig() Variant {
	return func(t testing.TB, ws *Workspace) {
		t.Helper()
		WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), "workspace: [sin cerrar\n")
	}
}

// WithoutSpecsRepository retira el repositorio de especificaciones, pero deja
// el puntero apuntando a su axiom.yaml en otra ubicación (specs_repository
// ausente en disco).
func WithoutSpecsRepository() Variant {
	return func(t testing.TB, ws *Workspace) {
		t.Helper()
		configCopy := filepath.Join(ws.Master, "config-externa", "axiom.yaml")
		WriteFile(t, configCopy, canonicalConfig)
		WriteFile(t, filepath.Join(ws.Master, ".axiom-workspace"), "config: config-externa/axiom.yaml\n")
		if err := os.RemoveAll(ws.Specs); err != nil {
			t.Fatalf("retirar el repositorio de especificaciones: %v", err)
		}
	}
}

// WithArchivedChange añade el cambio archivado con prefijo de fecha
// archive/2026-10-04-inc-98-previo al planning home.
func WithArchivedChange() Variant {
	return func(t testing.TB, ws *Workspace) {
		t.Helper()
		MkdirAll(t, filepath.Join(ws.Home, "changes", "archive", "2026-10-04-inc-98-previo"))
	}
}

// WriteFile escribe content en path, creando los directorios intermedios.
func WriteFile(t testing.TB, path, content string) {
	t.Helper()
	MkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir %s: %v", path, err)
	}
}

// MkdirAll crea dir y sus ancestros.
func MkdirAll(t testing.TB, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("crear %s: %v", dir, err)
	}
}

// InitGit ejecuta git init en dir con un entorno aislado de la configuración
// del desarrollador.
func InitGit(t testing.TB, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init en %s: %v\n%s", dir, err, out)
	}
}

// evalDir resuelve los enlaces simbólicos y los nombres cortos de dir, de modo
// que las rutas esperadas coincidan con las que produce la resolución.
func evalDir(t testing.TB, dir string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolver %s: %v", dir, err)
	}
	return resolved
}
