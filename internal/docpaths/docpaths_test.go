package docpaths

import (
	"reflect"
	"strings"
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
		"openspec/changes/archive/2026-01-01-x/.gates.lock": true, // MODIFIED RF-PUB-01: estado local del cambio archivado
		".gates.lock": false,
	}
	for rel, want := range cases {
		if got := set.LocalState(rel); got != want {
			t.Errorf("LocalState(%q) = %v, se esperaba %v", rel, got, want)
		}
	}
}

// MODIFIED RF-PUB-01: un fichero con punto directamente en la raíz de la carpeta archivada del cambio
// es estado local; el anidado, el de otro cambio y el de una carpeta que no es archivada, no.
func TestLocalStateArchivedRoot(t *testing.T) {
	set := mustNew(t, "openspec", "x", nil, nil)
	cases := map[string]bool{
		"openspec/changes/archive/2026-11-19-x/.integrations-outbox.yaml":     true,  // raíz archivada del cambio
		"openspec/changes/archive/2026-11-19-x/.integrations-outbox-123.yaml": true,  // temporal de escritura atómica
		"openspec/changes/x/.integrations-outbox.yaml":                        true,  // raíz ordinaria, como hoy
		"openspec/changes/archive/2026-11-19-x/specs/.nota":                   false, // anidado: sigue siendo documentación
		"openspec/changes/archive/2026-11-19-x/.dir/fichero":                  false, // dentro de una carpeta con punto
		"openspec/changes/archive/2026-11-01-y/.x":                            false, // carpeta archivada de otro cambio
		"openspec/changes/archive/2026-11-19-x-y/.x":                          false, // nombre que solo empieza igual
		"openspec/changes/archive/x/.x":                                       false, // sin fecha
		"openspec/changes/archive/.x":                                         false, // archive sin carpeta de cambio
		"openspec/changes/archive/2026-11-19-x/integrations.yaml":             false, // sin punto: documentación
		"openspec/changes/archive/2026-11-19-x":                               false, // la carpeta misma
		"openspec/changes/archive/2026-11-19-x/":                              false,
		"openspec/changes/archive/2026-11-19-x/.gates.lock/":                  false,
		"openspec/specs/.x":                                                   false,
	}
	for rel, want := range cases {
		if got := set.LocalState(rel); got != want {
			t.Errorf("LocalState(%q) = %v, se esperaba %v", rel, got, want)
		}
	}

	want := []string{
		"openspec/changes/x/.*",
		"openspec/changes/archive/[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-x/.*",
		"openspec/changes/x/context/**",
		"openspec/changes/x/.context-tmp/**",
		"openspec/changes/archive/[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-x/context/**",
		"openspec/changes/archive/[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-x/.context-tmp/**",
	}
	if got := set.LocalStateGlobs(); !reflect.DeepEqual(got, want) {
		t.Errorf("LocalStateGlobs = %v, se esperaba %v", got, want)
	}

	// Una raíz de planning vacía (repositorio de specs de un multirepo) cuelga de changes/ sin prefijo.
	root := mustNew(t, "", "x", nil, nil)
	if !root.LocalState("changes/archive/2026-11-19-x/.integrations-outbox.yaml") {
		t.Error("la raíz archivada también es estado local con planning home vacío")
	}
	if got := root.LocalStateGlobs()[1]; got != "changes/archive/[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-x/.*" {
		t.Errorf("glob archivado con planning home vacío = %q", got)
	}
}

// D-16: el contexto importado de un work item y sus temporales son estado local, activo y archivado.
func TestLocalStateIncludesContext(t *testing.T) {
	set := mustNew(t, "openspec", "x", nil, nil)
	cases := map[string]bool{
		"openspec/changes/x/context/work-item.md":                      true,
		"openspec/changes/x/context/attachments/tasks.md":              true,
		"openspec/changes/x/.context-tmp/abc/work-item.json":           true,
		"openspec/changes/archive/2026-11-26-x/context/work-item.json": true,
		"openspec/changes/archive/2026-11-26-x/.context-tmp/abc/a.png": true,
		"openspec/changes/archive/2026-11-26-y/context/work-item.json": false, // otro cambio
		"openspec/changes/y/context/work-item.md":                      false, // otro cambio
		"openspec/changes/x/contexto/a.md":                             false, // solo la carpeta exacta
		"openspec/changes/x/specs/context/a.md":                        false, // anidado: documentación
		"openspec/changes/x/proposal.md":                               false,
		"openspec/changes/archive/2026-11-26-x/proposal.md":            false,
		"openspec/changes/archive/2026-11-26-x/specs/context/a.md":     false,
		"openspec/changes/archive/x/context/a.md":                      false, // sin fecha
		"openspec/changes/x/context":                                   false, // la carpeta sin contenido no es un fichero
	}
	for rel, want := range cases {
		if got := set.LocalState(rel); got != want {
			t.Errorf("LocalState(%q) = %v, se esperaba %v", rel, got, want)
		}
	}
	joined := strings.Join(set.LocalStateGlobs(), "\n")
	for _, glob := range []string{"openspec/changes/x/context/**", "openspec/changes/x/.context-tmp/**"} {
		if !strings.Contains(joined, glob) {
			t.Errorf("LocalStateGlobs no contiene %q: %v", glob, set.LocalStateGlobs())
		}
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

// FromWorkspace es el único punto de composición de la lista cerrada: debe dar el mismo conjunto
// que componía `sddpublish` (rutas declaradas relativas al workspaceRoot, derivado del planning
// home) y descartar lo que cae fuera de la raíz git.
func TestFromWorkspaceRebasesDeclaredPathsOntoTheGitRoot(t *testing.T) {
	cases := []struct {
		name       string
		planning   string
		maintained []string
		reference  []string
		wantDocs   []string // deben ser documentación
		wantCode   []string // deben ser código
		candidates []string
	}{
		{
			name:       "planning home en la raíz",
			planning:   "",
			maintained: []string{"docs/contexto"},
			reference:  []string{"docs/contexto/ref"},
			wantDocs:   []string{"changes/x/design.md", "docs/contexto/a.md"},
			wantCode:   []string{"docs/contexto/ref/b.md", "docs/otro.md"},
			candidates: []string{"changes/x", "changes/archive", "specs", "INDEX.md", "docs/contexto"},
		},
		{
			name:       "openspec en la raíz git",
			planning:   "openspec",
			maintained: []string{"docs/contexto"},
			wantDocs:   []string{"openspec/changes/x/spec.md", "docs/contexto/a.md"},
			wantCode:   []string{"requirements.txt"},
			candidates: []string{"openspec/changes/x", "openspec/changes/archive", "openspec/specs", "openspec/INDEX.md", "docs/contexto"},
		},
		{
			name:       "workspace en un subdirectorio",
			planning:   "sub/openspec",
			maintained: []string{"docs/contexto"},
			reference:  []string{"docs/contexto/ref"},
			wantDocs:   []string{"sub/openspec/changes/x/spec.md", "sub/docs/contexto/a.md"},
			wantCode:   []string{"docs/contexto/a.md", "sub/docs/contexto/ref/b.md"},
			candidates: []string{"sub/openspec/changes/x", "sub/openspec/changes/archive", "sub/openspec/specs", "sub/openspec/INDEX.md", "sub/docs/contexto"},
		},
		{
			name:       "rutas declaradas fuera de la raíz se descartan",
			planning:   "sub/openspec",
			maintained: []string{"../fuera", "/abs", `C:\abs`, ".", "docs/ok"},
			wantDocs:   []string{"sub/docs/ok/a.md"},
			wantCode:   []string{"fuera/a.md", "sub/docs/a.md"},
			candidates: []string{"sub/openspec/changes/x", "sub/openspec/changes/archive", "sub/openspec/specs", "sub/openspec/INDEX.md", "sub/docs/ok"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set, err := FromWorkspace(tc.planning, "x", tc.maintained, tc.reference)
			if err != nil {
				t.Fatalf("FromWorkspace: %v", err)
			}
			for _, rel := range tc.wantDocs {
				if !set.Contains(rel) {
					t.Errorf("Contains(%q) = false, se esperaba documentación", rel)
				}
			}
			for _, rel := range tc.wantCode {
				if set.Contains(rel) {
					t.Errorf("Contains(%q) = true, se esperaba código", rel)
				}
			}
			if got := set.Candidates(); !reflect.DeepEqual(got, tc.candidates) {
				t.Errorf("Candidates() = %v, se esperaba %v", got, tc.candidates)
			}
			// Con el workspace en la raíz git, es el mismo conjunto que componer a mano con New.
			if tc.planning == "" || tc.planning == "openspec" {
				manual := mustNew(t, tc.planning, "x", tc.maintained, tc.reference)
				if !reflect.DeepEqual(manual.Candidates(), set.Candidates()) {
					t.Errorf("Candidates difiere de New: %v frente a %v", set.Candidates(), manual.Candidates())
				}
			}
		})
	}
}

func TestFromWorkspaceRejectsAnUnusableChangeName(t *testing.T) {
	if _, err := FromWorkspace("openspec", "a/b", nil, nil); err == nil {
		t.Fatal("FromWorkspace aceptó un nombre de cambio con barra")
	}
}
