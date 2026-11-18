package gitidentity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"recorte", "  Ana Pérez \n", "Ana Pérez"},
		{"solo espacios", "   \t ", ""},
		{"CR", "Ana\rPérez", ""},
		{"LF interno", "Ana\nPérez", ""},
		{"TAB", "Ana\tPérez", ""},
		{"ESC", "Ana\x1b[31mPérez", ""},
		{"DEL", "Ana\x7fPérez", ""},
		{"C1", "Ana\u0085Pérez", ""},
		{"UTF-8 no válido", "Ana\xff\xfePérez", ""},
		{"angulares", "Ana <admin> Pérez", "Ana admin Pérez"},
		{"solo angulares", " <> ", ""},
		{"acentos y almohadilla", "Íñigo: Gutiérrez # Zufia", "Íñigo: Gutiérrez # Zufia"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Sanitize(tc.in); got != tc.want {
				t.Errorf("Sanitize(%q) = %q, quiero %q", tc.in, got, tc.want)
			}
		})
	}

	t.Run("truncado a 120 runes", func(t *testing.T) {
		got := Sanitize(strings.Repeat("ñ", 500))
		if n := utf8.RuneCountInString(got); n != MaxPartRunes {
			t.Fatalf("%d runes, quiero %d", n, MaxPartRunes)
		}
		if !utf8.ValidString(got) {
			t.Fatal("el truncado cortó una rune")
		}
	})
	t.Run("truncado sin espacio final", func(t *testing.T) {
		in := strings.Repeat("a", MaxPartRunes-1) + " " + strings.Repeat("b", 50)
		got := Sanitize(in)
		if got != strings.Repeat("a", MaxPartRunes-1) {
			t.Fatalf("Sanitize = %q (%d runes)", got, utf8.RuneCountInString(got))
		}
	})
}

func TestIdentityActorAndMissing(t *testing.T) {
	full := New("Ana Pérez", "ana@example.com")
	if !full.Complete() {
		t.Fatal("la identidad completa debe estar completa")
	}
	actor, ok := full.Actor()
	if !ok || actor != "Ana Pérez <ana@example.com>" {
		t.Fatalf("Actor() = %q, %v", actor, ok)
	}
	if full.Missing() != nil {
		t.Fatalf("Missing() = %v, quiero vacío", full.Missing())
	}

	long, ok := New(strings.Repeat("n", 500), strings.Repeat("e", 500)).Actor()
	if !ok || utf8.RuneCountInString(long) > 256 {
		t.Fatalf("actor largo: %d runes, ok=%v", utf8.RuneCountInString(long), ok)
	}

	hostile, ok := New("Ana <admin>", "<a@b>").Actor()
	if !ok || strings.Count(hostile, "<") != 1 || strings.Count(hostile, ">") != 1 {
		t.Fatalf("el actor debe llevar un solo '<' y un solo '>': %q", hostile)
	}
	for _, reserved := range []string{"cli", "inferred"} {
		if actor == reserved || hostile == reserved {
			t.Fatalf("el actor coincide con el valor reservado %q", reserved)
		}
	}

	cases := []struct {
		name     string
		id       Identity
		complete bool
		missing  []string
	}{
		{"ninguna", New("", ""), false, []string{KeyName, KeyEmail}},
		{"solo nombre", New("Ana", ""), false, []string{KeyEmail}},
		{"solo email", New("", "ana@example.com"), false, []string{KeyName}},
		{"parte vacía por saneado", New("Ana", "   "), false, []string{KeyEmail}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.id.Complete() != tc.complete {
				t.Errorf("Complete() = %v", tc.id.Complete())
			}
			if got, ok := tc.id.Actor(); ok || got != "" {
				t.Errorf("Actor() = %q, %v; una identidad parcial es ausente", got, ok)
			}
			if !reflect.DeepEqual(tc.id.Missing(), tc.missing) {
				t.Errorf("Missing() = %v, quiero %v", tc.id.Missing(), tc.missing)
			}
		})
	}
}

// isolateGit aísla git de la máquina: HOME temporal, configuración global
// con el contenido indicado, sin configuración del sistema y sin variables de
// autor ni de redirección del repositorio.
func isolateGit(t *testing.T, globalConfig string) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
	home := t.TempDir()
	global := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(global, []byte(globalConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	for _, name := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "GIT_DIR", "GIT_WORK_TREE", "GIT_CEILING_DIRECTORIES"} {
		// t.Setenv registra la restauración; después se retira la variable.
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func newRepo(t *testing.T, name, email string) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	if name != "" {
		runGit(t, dir, "config", "--local", "user.name", name)
	}
	if email != "" {
		runGit(t, dir, "config", "--local", "user.email", email)
	}
	return dir
}

const globalIdentityA = "[user]\n\tname = Global A\n\temail = a@example.com\n"

func TestResolveLocalConfig(t *testing.T) {
	isolateGit(t, "")
	repo := newRepo(t, "Ana Pérez", "ana@example.com")
	if got := Resolve(context.Background(), repo); got != New("Ana Pérez", "ana@example.com") {
		t.Fatalf("Resolve = %+v", got)
	}
}

func TestResolveLocalOverridesGlobal(t *testing.T) {
	isolateGit(t, globalIdentityA)
	repo := newRepo(t, "Local B", "b@example.com")
	if got := Resolve(context.Background(), repo); got != New("Local B", "b@example.com") {
		t.Fatalf("la configuración local debe prevalecer: %+v", got)
	}
}

func TestResolveNonRepositoryUsesGlobal(t *testing.T) {
	isolateGit(t, globalIdentityA)
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	if got := Resolve(context.Background(), dir); got != New("Global A", "a@example.com") {
		t.Fatalf("Resolve = %+v", got)
	}
}

func TestResolveIgnoresGitAuthorEnv(t *testing.T) {
	isolateGit(t, "")
	t.Setenv("GIT_AUTHOR_NAME", "Autor Env")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Comitente Env")
	t.Setenv("GIT_COMMITTER_EMAIL", "comitente@example.com")
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	if got := Resolve(context.Background(), dir); got != (Identity{}) {
		t.Fatalf("las variables GIT_AUTHOR_*/GIT_COMMITTER_* no son identidad: %+v", got)
	}
}

func TestResolveIgnoresRedirectingGitDir(t *testing.T) {
	isolateGit(t, "")
	other := newRepo(t, "Otro Repositorio", "otro@example.com")
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	if got := Resolve(context.Background(), dir); got != (Identity{}) {
		t.Fatalf("GIT_DIR no debe redirigir la resolución: %+v", got)
	}
}

func TestResolveWithoutGitIsAbsent(t *testing.T) {
	isolateGit(t, globalIdentityA)
	t.Setenv("PATH", t.TempDir())
	if got := Resolve(context.Background(), t.TempDir()); got != (Identity{}) {
		t.Fatalf("sin git la identidad está ausente: %+v", got)
	}
}

func TestResolveRelativeDirIsAbsent(t *testing.T) {
	isolateGit(t, globalIdentityA)
	if got := Resolve(context.Background(), filepath.Join("relativo", "ruta")); got != (Identity{}) {
		t.Fatalf("un directorio relativo deja la identidad ausente: %+v", got)
	}
}
