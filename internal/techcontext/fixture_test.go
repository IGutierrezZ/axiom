package techcontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const fixtureConfig = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
technical_context:
  maintained:
    paths:
      - context/
      - ../repo-backend/docs/
  reference:
    paths:
      - referencia/
`

// configWithoutContext es la configuración del espacio sin `technical_context`.
const configWithoutContext = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
`

const (
	fixtureContext = "# Arquitectura\n\n## Componentes\n\n### Servicio de catálogo\n\nGestiona los productos.\n\n### Servicio de pedidos\n\nGestiona los pedidos.\n"
	fixtureAPI     = "# API\n\n## Pagos\n\nTexto de pagos.\n"

	fixtureContextImpact = `# Delta de contexto: inc-99-ejemplo

## Entrada: context/ARQUITECTURA.md

- Ancla: Componentes > Servicio de pedidos
- Acción: modify
- Motivo: se añade el reintento de pago

Texto nuevo para la sección.

## Entrada: ../repo-backend/docs/API.md

- Ancla: Errores
- Acción: add
- Motivo: documenta el error del cobro reintentado
`
)

// fixture es un espacio canónico sintético: la carpeta maestra con el puntero, el
// repositorio de especificaciones con el contexto técnico declarado (sin
// baseline), un repositorio de código hermano con documentación mantenida y el
// cambio inc-99-ejemplo con su delta de contexto.
type fixture struct {
	master, specs, backend, changeRoot string
}

var fixtureTemplate = &treeTemplate{name: "espacio-contexto"}

func newFixtureAt(master string) *fixture {
	return &fixture{
		master:     master,
		specs:      filepath.Join(master, "repo-specs"),
		backend:    filepath.Join(master, "repo-backend"),
		changeRoot: filepath.Join(master, "repo-specs", "openspec", "changes", "inc-99-ejemplo"),
	}
}

// newFixture monta el espacio en una carpeta temporal. Con git, ambos
// repositorios quedan inicializados y con todo confirmado.
func newFixture(t *testing.T, withGit bool) *fixture {
	t.Helper()
	if withGit {
		skipWithoutGit(t)
	}
	master, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := newFixtureAt(master)
	if withGit {
		fixtureTemplate.materialize(t, master, func(t testing.TB, dir string) {
			buildFixture(t, newFixtureAt(dir), true)
		})
		return f
	}
	buildFixture(t, f, false)
	return f
}

func skipWithoutGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
}

func initRepo(t testing.TB, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "core.autocrlf", "false")
}

func commitAll(t testing.TB, dir, message string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", message)
	return strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))
}

func buildFixture(t testing.TB, f *fixture, withGit bool) {
	t.Helper()
	if withGit {
		initRepo(t, f.specs)
		initRepo(t, f.backend)
	}
	writeTestFile(t, filepath.Join(f.master, ".axiom-workspace"), "config: repo-specs/axiom.yaml\n")
	writeTestFile(t, filepath.Join(f.specs, "axiom.yaml"), fixtureConfig)
	writeTestFile(t, filepath.Join(f.specs, "context", "ARQUITECTURA.md"), fixtureContext)
	writeTestFile(t, filepath.Join(f.specs, "referencia", "legado", "VIEJO.md"), "# Viejo\n")
	writeTestFile(t, filepath.Join(f.backend, "docs", "API.md"), fixtureAPI)
	f.writeChange(t, "specs/pagos/spec.md", "## ADDED Requirements\n\n### Requirement: Reintento (REQ-1.1)\n\nEl sistema DEBE reintentar.\n")
	f.writeChange(t, DeltaFileName, fixtureContextImpact)
	if withGit {
		commitAll(t, f.specs, "especificaciones")
		commitAll(t, f.backend, "backend")
	}
}

func (f *fixture) writeChange(t testing.TB, rel, content string) {
	t.Helper()
	writeTestFile(t, filepath.Join(f.changeRoot, filepath.FromSlash(rel)), content)
}

func (f *fixture) removeChange(t *testing.T, rel string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(f.changeRoot, filepath.FromSlash(rel))); err != nil {
		t.Fatal(err)
	}
}

// target resuelve de nuevo la ubicación, de modo que un axiom.yaml modificado
// por la prueba se vea en la declaración.
func (f *fixture) target(t *testing.T) Target {
	t.Helper()
	loc, err := workspace.Locate(f.master)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	return Target{
		Location:      loc,
		Change:        "inc-99-ejemplo",
		ChangeRoot:    f.changeRoot,
		AllowedRoots:  []string{loc.WorkspaceRoot, f.backend},
		ArtifactStore: "openspec",
	}
}

func (f *fixture) check(t *testing.T) Report {
	t.Helper()
	report, err := Check(context.Background(), f.target(t))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return report
}

func sha256Of(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// snapshot devuelve la huella de cada fichero bajo root, sin los directorios .git.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		relative, _ := filepath.Rel(root, path)
		files[filepath.ToSlash(relative)] = sha256Of(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func changedFiles(before, after map[string]string) []string {
	var changed []string
	for path, sum := range after {
		if before[path] != sum {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed = append(changed, path+" (borrado)")
		}
	}
	return changed
}

func mustErrors(t *testing.T, report Report, fragments ...string) {
	t.Helper()
	joined := strings.Join(report.Errors, "\n")
	if len(report.Errors) == 0 {
		t.Fatalf("el informe debía tener errores con %q", fragments)
	}
	for _, fragment := range fragments {
		if !strings.Contains(joined, fragment) {
			t.Errorf("los errores deben contener %q:\n%s", fragment, joined)
		}
	}
}

func mustNoErrors(t *testing.T, report Report) {
	t.Helper()
	if len(report.Errors) != 0 {
		t.Fatalf("errores inesperados:\n%s", strings.Join(report.Errors, "\n"))
	}
}
