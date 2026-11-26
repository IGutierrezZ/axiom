package state

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureChangeContextIgnoredIdempotent(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	exclude := filepath.Join(root, ".git", "info", "exclude")

	if err := EnsureChangeContextIgnored(root); err != nil {
		t.Fatal(err)
	}
	first := readBytes(t, exclude)
	if err := EnsureChangeContextIgnored(root); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, readBytes(t, exclude)) {
		t.Errorf("la segunda llamada ha reescrito info/exclude:\n%s\n---\n%s", first, readBytes(t, exclude))
	}
	for _, entry := range changeContextExcludeEntries("/") {
		if strings.Count(string(first), entry+"\n") != 1 {
			t.Errorf("la entrada %q debe aparecer una sola vez:\n%s", entry, first)
		}
	}
	for _, rel := range []string{
		"openspec/changes/x/context/work-item.md",
		"openspec/changes/x/.context-tmp/abc/work-item.md",
		"openspec/changes/archive/2026-11-26-x/context/work-item.json",
		"odd/context/x/work-item.md",
		"odd/context/.tmp/x/abc/work-item.md",
	} {
		if !gitIgnored(t, root, rel) {
			t.Errorf("git no ignora %s", rel)
		}
	}
	if gitIgnored(t, root, "openspec/changes/x/proposal.md") {
		t.Error("el resto del cambio no debe quedar excluido")
	}
}

func TestEnsureChangeContextIgnoredKeepsUserLines(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	exclude := filepath.Join(root, ".git", "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exclude, []byte("*.secreto"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := EnsureChangeContextIgnored(root); err != nil {
			t.Fatal(err)
		}
	}
	got := string(readBytes(t, exclude))
	if !strings.HasPrefix(got, "*.secreto\n") {
		t.Errorf("la línea ajena debe conservarse al principio: %q", got)
	}
	if strings.Count(got, "/odd/context/\n") != 1 || strings.Count(got, contextExcludeHeader) != 1 {
		t.Errorf("las entradas y la cabecera deben aparecer una sola vez: %q", got)
	}
}

func TestEnsureChangeContextIgnoredWorkspacePrefix(t *testing.T) {
	gitAvailable(t)
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	nested := filepath.Join(repo, "servicios", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := EnsureChangeContextIgnored(nested); err != nil {
		t.Fatal(err)
	}
	if !gitIgnored(t, repo, "servicios/api/openspec/changes/x/context/work-item.md") {
		t.Error("git no ignora el contexto del workspace anidado")
	}
	if !gitIgnored(t, repo, "servicios/api/odd/context/x/work-item.md") {
		t.Error("git no ignora el contexto ODD del workspace anidado")
	}
	if gitIgnored(t, repo, "odd/context/x/work-item.md") || gitIgnored(t, repo, "openspec/changes/x/context/work-item.md") {
		t.Error("el prefijo anidado no debe excluir el contexto de la raíz del repositorio")
	}
}

func TestEnsureChangeContextIgnoredLinkedWorktreeUsesCommonDir(t *testing.T) {
	gitAvailable(t)
	main := t.TempDir()
	runGit(t, main, "init", "-q")
	runGit(t, main, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "init")
	linked := filepath.Join(t.TempDir(), "wt")
	runGit(t, main, "worktree", "add", "-q", "-b", "rama", linked)
	if err := EnsureChangeContextIgnored(linked); err != nil {
		t.Fatal(err)
	}
	if !gitIgnored(t, linked, "odd/context/x/work-item.md") || !gitIgnored(t, main, "odd/context/x/work-item.md") {
		t.Error("la exclusión debe valer en todos los worktrees")
	}
}

func TestEnsureChangeContextIgnoredOutsideGitDoesNothing(t *testing.T) {
	root := t.TempDir()
	if err := EnsureChangeContextIgnored(root); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Errorf("fuera de git no debe crearse nada: %v", entries)
	}
}
