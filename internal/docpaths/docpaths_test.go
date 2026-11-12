package docpaths

import (
	"reflect"
	"testing"
)

func mustNew(t *testing.T, planning, change string, maintained, reference []string) Set {
	t.Helper()
	set, err := New(planning, change, maintained, reference)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return set
}

// Matriz de amenazas «Rutas que parecen documentación»: solo cuenta la lista cerrada,
// por raíz de ruta y no por extensión.
func TestDocPathsClassifiesByRootNotExtension(t *testing.T) {
	set := mustNew(t, "openspec", "x",
		[]string{"docs/contexto"},
		[]string{"docs/contexto/referencia"})

	cases := []struct {
		rel  string
		root string
		want bool
	}{
		// Código aunque parezca documentación.
		{"requirements.txt", "", false},
		{"CMakeLists.txt", "", false},
		{"docs/x.mdx", "", false},
		{"README.sh", "", false},
		{"README.md", "", false},
		{"internal/cli/sdd_publish.go", "", false},
		// El mismo nombre bajo la carpeta del cambio es documentación y nunca se ejecuta.
		{"openspec/changes/x/README.sh", "openspec/changes/x", true},
		{"openspec/changes/x/specs/a/spec.md", "openspec/changes/x", true},
		{"openspec/changes/x/gates.yaml", "openspec/changes/x", true},
		// Otro cambio: no es de este.
		{"openspec/changes/foo-x/proposal.md", "", false},
		{"openspec/changes/xx/proposal.md", "", false},
		{"openspec/changes/x", "", false},
		// Carpeta archivada de este cambio y no de otro de nombre parecido.
		{"openspec/changes/archive/2026-11-09-x/proposal.md", "openspec/changes/archive/2026-11-09-x", true},
		{"openspec/changes/archive/2026-11-09-foo-x/proposal.md", "", false},
		{"openspec/changes/archive/2026-11-09-x-y/proposal.md", "", false},
		{"openspec/changes/archive/x/proposal.md", "", false},
		{"openspec/changes/archive/2026-11-09-x", "", false},
		// Specs, índice y contexto mantenido.
		{"openspec/specs/a/spec.md", "openspec/specs", true},
		{"openspec/INDEX.md", "openspec/INDEX.md", true},
		{"openspec/INDEX.md.bak", "", false},
		{"docs/contexto", "docs/contexto", true},
		{"docs/contexto/a.md", "docs/contexto", true},
		{"docs/contexto-otro/a.md", "", false},
		// reference gana sobre cualquier otra regla.
		{"docs/contexto/referencia/a.md", "", false},
		{"docs/contexto/referencia", "", false},
		// odd/tasks no está en la lista.
		{"odd/tasks/feature.md", "", false},
		// Las comparaciones distinguen mayúsculas, como git.
		{"OpenSpec/changes/x/a.md", "", false},
		{"openspec/changes/X/a.md", "", false},
		{"openspec/CHANGES/x/a.md", "", false},
		// Rutas mal formadas nunca son documentación.
		{"", "", false},
		{"/openspec/changes/x/a.md", "", false},
		{"openspec/changes/x/../../../secret", "", false},
		{`openspec\changes\x\a.md`, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.rel, func(t *testing.T) {
			if got := set.Contains(tc.rel); got != tc.want {
				t.Errorf("Contains(%q) = %v, se esperaba %v", tc.rel, got, tc.want)
			}
			root, ok := set.Root(tc.rel)
			if ok != tc.want || root != tc.root {
				t.Errorf("Root(%q) = %q, %v; se esperaba %q, %v", tc.rel, root, ok, tc.root, tc.want)
			}
		})
	}
}

// El repositorio de specs de un multirepo tiene el planning home en la raíz.
func TestDocPathsEmptyPlanningHome(t *testing.T) {
	set := mustNew(t, "", "x", nil, nil)
	for rel, want := range map[string]bool{
		"changes/x/design.md":               true,
		"specs/a/spec.md":                   true,
		"INDEX.md":                          true,
		"changes/archive/2026-01-02-x/p.md": true,
		"main.go":                           false,
		"changes/y/design.md":               false,
	} {
		if got := set.Contains(rel); got != want {
			t.Errorf("Contains(%q) = %v, se esperaba %v", rel, got, want)
		}
	}
	want := []string{"changes/x", "changes/archive", "specs", "INDEX.md"}
	if got := set.Candidates(); !reflect.DeepEqual(got, want) {
		t.Errorf("Candidates = %v, se esperaba %v", got, want)
	}
}

func TestDocPathsLocalState(t *testing.T) {
	set := mustNew(t, "openspec", "x", nil, nil)
	cases := map[string]bool{
		"openspec/changes/x/.gates.lock":                    true,
		"openspec/changes/x/.axiom-instance":                true,
		"openspec/changes/x/.gates.yaml.tmp1":               true,
		"openspec/changes/x/gates.yaml":                     false,
		"openspec/changes/x/specs/.hidden":                  false,
		"openspec/changes/y/.gates.lock":                    false,
		"openspec/changes/archive/2026-01-01-x/.gates.lock": false,
		".gates.lock": false,
	}
	for rel, want := range cases {
		if got := set.LocalState(rel); got != want {
			t.Errorf("LocalState(%q) = %v, se esperaba %v", rel, got, want)
		}
	}
	if got, want := set.LocalStateGlob(), "openspec/changes/x/.*"; got != want {
		t.Errorf("LocalStateGlob = %q, se esperaba %q", got, want)
	}
}

func TestDocPathsCandidatesAreStableAndUnique(t *testing.T) {
	set := mustNew(t, "openspec", "x", []string{"docs/ctx", "docs/a.md", "docs/a.md"}, []string{"docs/ref"})
	want := []string{"openspec/changes/x", "openspec/changes/archive", "openspec/specs", "openspec/INDEX.md", "docs/ctx", "docs/a.md"}
	if got := set.Candidates(); !reflect.DeepEqual(got, want) {
		t.Errorf("Candidates = %v, se esperaba %v", got, want)
	}
}

func TestDocPathsNewRejectsUnusableInput(t *testing.T) {
	for name, call := range map[string]func() error{
		"cambio vacío":          func() error { _, err := New("openspec", "", nil, nil); return err },
		"cambio con barra":      func() error { _, err := New("openspec", "a/b", nil, nil); return err },
		"cambio punto punto":    func() error { _, err := New("openspec", "..", nil, nil); return err },
		"planning fuera":        func() error { _, err := New("../x", "x", nil, nil); return err },
		"maintained absoluta":   func() error { _, err := New("", "x", []string{"/etc/passwd"}, nil); return err },
		"maintained raíz":       func() error { _, err := New("", "x", []string{"."}, nil); return err },
		"reference fuera":       func() error { _, err := New("", "x", nil, []string{"../a"}); return err },
		"maintained con unidad": func() error { _, err := New("", "x", []string{`C:\a`}, nil); return err },
		"maintained vacía":      func() error { _, err := New("", "x", []string{""}, nil); return err },
	} {
		if call() == nil {
			t.Errorf("%s: se esperaba un error", name)
		}
	}
}

func TestDocPathsZeroValueRecognisesNothing(t *testing.T) {
	var set Set
	if set.Contains("openspec/changes/x/a.md") || set.LocalState("a") || set.Candidates() != nil {
		t.Error("el valor cero no debe reconocer ninguna ruta")
	}
}
