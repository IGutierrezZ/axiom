package gitcli

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// RemoteConfigured lee la configuración local: con y sin `origin`, sin tocar la red.
func TestRemoteConfigured(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	git := New()

	if ok, err := git.RemoteConfigured(ctx, repo, "origin"); err != nil || ok {
		t.Fatalf("RemoteConfigured sin origin = %v, %v; se esperaba false y sin error", ok, err)
	}
	runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe.git"))
	if ok, err := git.RemoteConfigured(ctx, repo, "origin"); err != nil || !ok {
		t.Fatalf("RemoteConfigured con origin = %v, %v; se esperaba true y sin error", ok, err)
	}
	if ok, err := git.RemoteConfigured(ctx, repo, "otro"); err != nil || ok {
		t.Errorf("RemoteConfigured de un remoto distinto = %v, %v; se esperaba false", ok, err)
	}
	if _, err := git.RemoteConfigured(ctx, repo, "--bad"); err == nil {
		t.Error("RemoteConfigured aceptó un nombre de remoto que parece una opción")
	}
}

// DirtyPathsUnder expande lo que no tiene seguimiento, ve lo preparado, lo modificado y lo borrado,
// toma las rutas como literales y no ve nada fuera de las rutas candidatas.
func TestDirtyPathsUnder(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	git := New()
	writeFile(t, filepath.Join(repo, "docs", "a.md"), "a")
	writeFile(t, filepath.Join(repo, "docs", "b.md"), "b")
	writeFile(t, filepath.Join(repo, "docs", "con espacio.md"), "c")
	writeFile(t, filepath.Join(repo, "fuera", "x.go"), "x")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "base")

	t.Run("limpio no devuelve nada", func(t *testing.T) {
		got, err := git.DirtyPathsUnder(ctx, repo, []string{"docs"})
		if err != nil || len(got) != 0 {
			t.Fatalf("DirtyPathsUnder = %v, %v; se esperaba vacío", got, err)
		}
	})

	t.Run("expande la carpeta sin seguimiento a ficheros", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "nueva", "k.yaml"), "k")
		writeFile(t, filepath.Join(repo, "nueva", ".gates.lock"), "")
		got, err := git.DirtyPathsUnder(ctx, repo, []string{"nueva"})
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(got)
		if want := []string{"nueva/.gates.lock", "nueva/k.yaml"}; !slices.Equal(got, want) {
			t.Errorf("DirtyPathsUnder = %v, se esperaba %v (cada fichero, sin la carpeta «nueva/»)", got, want)
		}
	})

	t.Run("cuenta preparados, sin preparar y borrados", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "docs", "a.md"), "a2")
		runGit(t, repo, "add", "docs/a.md")
		writeFile(t, filepath.Join(repo, "docs", "b.md"), "b2")
		if err := os.Remove(filepath.Join(repo, "docs", "con espacio.md")); err != nil {
			t.Fatal(err)
		}
		got, err := git.DirtyPathsUnder(ctx, repo, []string{"docs"})
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(got)
		want := []string{"docs/a.md", "docs/b.md", "docs/con espacio.md"}
		if !slices.Equal(got, want) {
			t.Errorf("DirtyPathsUnder = %v, se esperaba %v", got, want)
		}
	})

	t.Run("una ruta con espacios o asterisco es literal", func(t *testing.T) {
		// El asterisco no puede ser parte de un nombre en Windows: basta con que no actúe de patrón.
		for _, pattern := range []string{"docs/*.md", "docs/[ab].md", "docs/?.md"} {
			if got, err := git.DirtyPathsUnder(ctx, repo, []string{pattern}); err != nil || len(got) != 0 {
				t.Errorf("DirtyPathsUnder(%q) = %v, %v; se esperaba vacío (la ruta es literal)", pattern, got, err)
			}
		}
		got, err := git.DirtyPathsUnder(ctx, repo, []string{"docs/con espacio.md"})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"docs/con espacio.md"}; !slices.Equal(got, want) {
			t.Errorf("DirtyPathsUnder = %v, se esperaba %v", got, want)
		}
	})

	t.Run("no ve nada fuera de las rutas candidatas", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "fuera", "x.go"), "x2")
		got, err := git.DirtyPathsUnder(ctx, repo, []string{"nueva"})
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(got, "fuera/x.go") {
			t.Errorf("DirtyPathsUnder = %v, no debería ver fuera/x.go", got)
		}
	})

	t.Run("rechaza rutas que git leería de otro modo", func(t *testing.T) {
		for _, bad := range [][]string{nil, {""}, {"."}, {"../x"}, {"/abs"}, {"-o"}, {":(glob)x"}, {"a\nb"}} {
			if _, err := git.DirtyPathsUnder(ctx, repo, bad); err == nil {
				t.Errorf("DirtyPathsUnder(%q) no devolvió error", bad)
			}
		}
	})
}
