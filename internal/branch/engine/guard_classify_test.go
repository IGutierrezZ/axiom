package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Matriz de amenazas «Rutas con aspecto de documentación» (REQ-25.21, design D-18, paso 2): la
// guarda D clasifica por segmentos de ruta y nunca ejecuta ni interpreta el contenido. Son pruebas
// puras: no usan git ni tocan el disco.

func TestClassifyAxiomDocumentBySegments(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		path       string
		foldCase   bool
		wantDoc    bool
		wantChange string
		wantLane   branch.Lane
	}{
		{name: "el documento ODD exige .md", path: "/p/odd/tasks/x.md", wantDoc: true, wantChange: "x", wantLane: branch.LaneODD},
		{name: "un script bajo odd/tasks solo puede avisar", path: "/p/odd/tasks/README.sh"},
		{name: "odd/tasks/ solo cubre ficheros directos", path: "/p/odd/tasks/sub/x.md"},
		{name: "un nombre vacío no es un documento", path: "/p/odd/tasks/.md"},
		{name: "cualquier fichero de la carpeta de un cambio es un documento", path: "/p/openspec/changes/x/run.sh", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "requirements.txt dentro del cambio", path: "/p/openspec/changes/x/requirements.txt", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "CMakeLists.txt dentro del cambio", path: "/p/openspec/changes/x/CMakeLists.txt", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "MDX ejecutable dentro del cambio", path: "/p/openspec/changes/x/notes.mdx", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "una subcarpeta del cambio", path: "/p/openspec/changes/x/specs/a/spec.md", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "el nombre se toma tal cual, con metacaracteres de shell", path: "/p/openspec/changes/x/$(touch pwned).md", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "el cambio archivado es la parte tras la fecha", path: "/p/openspec/changes/archive/2026-10-17-x/proposal.md", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "el cambio archivado conserva los guiones de su nombre", path: "/p/openspec/changes/archive/2026-10-17-inc-25-ramas/design.md", wantDoc: true, wantChange: "inc-25-ramas", wantLane: branch.LaneSDD},
		{name: "la carpeta archive no es un cambio", path: "/p/openspec/changes/archive/README.md"},
		{name: "una carpeta de archivo sin fecha no es un cambio archivado", path: "/p/openspec/changes/archive/x/proposal.md"},
		{name: "la carpeta del cambio por sí sola no es un documento", path: "/p/openspec/changes/x"},
		{name: "la carpeta de los cambios no es un documento", path: "/p/openspec/changes"},
		{name: "un .. que sale de la carpeta del cambio se limpia y es otro fichero", path: "/p/openspec/changes/x/../../internal/a.go"},
		{name: "un .. que se queda en el cambio sigue siendo un documento", path: "/p/openspec/changes/y/../x/a.md", wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "requirements.txt en la raíz es otro fichero", path: "/p/requirements.txt"},
		{name: "CMakeLists.txt en la raíz es otro fichero", path: "/p/CMakeLists.txt"},
		{name: "Markdown fuera de openspec y odd es otro fichero", path: "/p/docs/guia.md"},
		{name: "openspec sin changes es otro fichero", path: "/p/openspec/specs/a/spec.md"},
		{name: "sin distinguir mayúsculas en Windows: openspec", path: "C:/P/OpenSpec/Changes/x/a.md", foldCase: true, wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "sin distinguir mayúsculas en Windows: odd", path: "C:/P/ODD/Tasks/X.MD", foldCase: true, wantDoc: true, wantChange: "X", wantLane: branch.LaneODD},
		{name: "sin distinguir mayúsculas en Windows: archive", path: "C:/P/openspec/changes/ARCHIVE/2026-10-17-x/a.md", foldCase: true, wantDoc: true, wantChange: "x", wantLane: branch.LaneSDD},
		{name: "con distinción de mayúsculas, otro nombre es otro fichero", path: "/p/OpenSpec/Changes/x/a.md"},
		{name: "con distinción de mayúsculas, odd/tasks en mayúsculas es otro fichero", path: "/p/ODD/Tasks/X.MD"},
		{name: "la ruta vacía no es un documento", path: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			document, ok := classifyAxiomDocument(tc.path, tc.foldCase)
			if ok != tc.wantDoc {
				t.Fatalf("classifyAxiomDocument(%q, fold=%v) = %+v, %v; se esperaba documento = %v", tc.path, tc.foldCase, document, ok, tc.wantDoc)
			}
			if !ok {
				return
			}
			if document.Change != tc.wantChange || document.Lane != tc.wantLane {
				t.Errorf("cambio %q y carril %q; se esperaba %q y %q", document.Change, document.Lane, tc.wantChange, tc.wantLane)
			}
		})
	}
}

// La clasificación es léxica: no crea, no lee ni ejecuta nada, ni siquiera con un nombre que parece
// una orden de shell.
func TestClassifyAxiomDocumentNeverTouchesTheDisk(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "openspec", "changes", "x", "$(touch pwned).md")

	if _, ok := classifyAxiomDocument(path, false); !ok {
		t.Fatalf("%q debía ser un documento de Axiom", path)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Errorf("la clasificación no debe crear nada: %v (%v)", entries, err)
	}
}

// Una ruta con .. que se resuelve fuera de la carpeta del cambio se clasifica ya limpia: el
// documento guardado es la ruta limpia, no la que envió el runtime.
func TestClassifyAxiomDocumentReturnsTheCleanedPath(t *testing.T) {
	t.Parallel()
	root := filepath.Join(string(filepath.Separator), "p")

	document, ok := classifyAxiomDocument(filepath.Join(root, "openspec", "changes", "y", "..", "x", "a.md"), false)

	if want := filepath.Join(root, "openspec", "changes", "x", "a.md"); !ok || document.Path != want {
		t.Errorf("ruta = %q (%v); se esperaba la ruta limpia %q", document.Path, ok, want)
	}
}
