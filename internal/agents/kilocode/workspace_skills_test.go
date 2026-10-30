package kilocode

import (
	"path/filepath"
	"reflect"
	"testing"
)

// En el proyecto, las skills van a la carpeta compartida .agents/skills y la carpeta
// anterior se ofrece para migrar; en cualquier otra carpeta (la de usuario) todo sigue
// siendo global y no hay nada que migrar.
func TestWorkspaceSkillsDirIsSharedAndLegacyIsOffered(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	bound := NewAdapter().WithWorkspace(root)

	if got, want := bound.SkillsDir(root), filepath.Join(root, ".agents", "skills"); got != want {
		t.Errorf("SkillsDir(root) = %q, quiero %q", got, want)
	}
	wantLegacy := []string{filepath.Join(root, ".kilo", "skills")}
	if got := bound.LegacyWorkspaceSkillsDirs(root); !reflect.DeepEqual(got, wantLegacy) {
		t.Errorf("LegacyWorkspaceSkillsDirs(root) = %v, quiero %v", got, wantLegacy)
	}
	if got, want := bound.SkillsDir(home), NewAdapter().SkillsDir(home); got != want {
		t.Errorf("SkillsDir(home) = %q, quiero la global %q", got, want)
	}
	if got := bound.LegacyWorkspaceSkillsDirs(home); len(got) != 0 {
		t.Errorf("LegacyWorkspaceSkillsDirs(home) = %v, quiero ninguna", got)
	}
	if got, want := NewAdapter().SkillsDir(root), bound.SkillsDir(home); got == filepath.Join(root, ".agents", "skills") || got != NewAdapter().SkillsDir(root) {
		t.Errorf("el adaptador sin ligar cambió: %q (global %q)", got, want)
	}
}
