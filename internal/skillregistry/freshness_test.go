package skillregistry

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFreshnessSkill(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryFreshnessAbsentFreshStale(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	skill := filepath.Join(cwd, "skills", "demo", "SKILL.md")
	writeFreshnessSkill(t, skill, "---\nname: demo\ndescription: \"A. Trigger: b\"\n---\n")

	if got := RegistryFreshness(cwd, home); got != FreshnessAbsent {
		t.Fatalf("sin registro = %q, want absent", got)
	}
	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}
	if got := RegistryFreshness(cwd, home); got != FreshnessFresh {
		t.Fatalf("tras regenerar = %q, want fresh", got)
	}
	writeFreshnessSkill(t, skill, "---\nname: demo\ndescription: \"A. Trigger: otro\"\n---\n")
	if got := RegistryFreshness(cwd, home); got != FreshnessStale {
		t.Fatalf("tras editar = %q, want stale", got)
	}
}

func TestChosenCopyPrefiereLaCompartidaDelProyecto(t *testing.T) {
	cwd := t.TempDir()
	perAgent := filepath.Join(cwd, ".claude", "skills", "x", "SKILL.md")
	shared := filepath.Join(cwd, ".agents", "skills", "x", "SKILL.md")
	user := filepath.Join(t.TempDir(), "x", "SKILL.md")

	if got := ChosenCopy(cwd, []string{perAgent, shared, user}); got != shared {
		t.Errorf("ChosenCopy = %q, want %q", got, shared)
	}
	if got := ChosenCopy(cwd, []string{user, perAgent}); got != perAgent {
		t.Errorf("la copia del proyecto gana a la de usuario: %q", got)
	}
	if got := ChosenCopy(cwd, nil); got != "" {
		t.Errorf("sin copias = %q", got)
	}
}

func TestIsReservedNameMatchesRegistryExclusions(t *testing.T) {
	for _, n := range []string{"_shared", "skill-registry", "sdd-apply"} {
		if !IsReservedName(n) {
			t.Errorf("%q debe ser reservado", n)
		}
	}
	if IsReservedName("go-testing") {
		t.Error("go-testing no es reservado")
	}
}
