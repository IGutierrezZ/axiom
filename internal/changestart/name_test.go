package changestart

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

func TestDeriveNameCanonical(t *testing.T) {
	if got := DeriveName("US", "2525", "Alta de usuarios"); got != "US-2525-alta-de-usuarios" {
		t.Fatalf("nombre = %q", got)
	}
}

func TestDeriveNameAccentsAndSigns(t *testing.T) {
	got := DeriveName("BUG", "7", "Fallo en la  pestaña “Señalización” (¡urgente!)")
	const want = "BUG-7-fallo-en-la-pestana-senalizacion-urgente"
	if got != want {
		t.Fatalf("nombre = %q, quiero %q", got, want)
	}
	if strings.Contains(got, "--") {
		t.Fatalf("guiones dobles en %q", got)
	}
}

func TestDeriveNameTruncated(t *testing.T) {
	title := strings.Repeat("palabra ", 40)
	got := DeriveName("US", "2525", title)
	if len(got) > 48 {
		t.Fatalf("longitud %d > 48: %q", len(got), got)
	}
	if strings.HasSuffix(got, "-") {
		t.Fatalf("termina en guion: %q", got)
	}
	if !strings.HasSuffix(got, "-palabra") {
		t.Fatalf("debe cortar en el último guion que quepa: %q", got)
	}
	if err := ValidChangeName(got); err != nil {
		t.Fatalf("inválido: %v", err)
	}
	// Sin guiones en el slug se corta en seco.
	got = DeriveName("US", "2525", strings.Repeat("a", 300))
	if len(got) != 48 || err2(ValidChangeName(got)) {
		t.Fatalf("corte en seco = %q (%d)", got, len(got))
	}
}

func err2(err error) bool { return err != nil }

func TestDeriveNameEmptySlug(t *testing.T) {
	for _, title := range []string{"", "¡¿?!", "---", "😀😀", "“”"} {
		if got := DeriveName("US", "12", title); got != "US-12" {
			t.Fatalf("título %q: nombre = %q", title, got)
		}
	}
}

func TestDeriveNameGitDangerous(t *testing.T) {
	for _, title := range []string{"a..b", "x~1", "y^z", "ref:name", "log@{1}", "a.lock", "../../etc", "con"} {
		got := DeriveName("US", "5", title)
		if err := gitref.ValidBranchComponent(got); err != nil {
			t.Fatalf("título %q: %q no es un componente de rama válido: %v", title, got, err)
		}
		if err := ValidChangeName(got); err != nil {
			t.Fatalf("título %q: %q: %v", title, got, err)
		}
	}
}

func TestValidChangeNameRejectsReserved(t *testing.T) {
	for _, name := range []string{"archive", "Archive", "", "a b", "../x", "-x", "a..b"} {
		if ValidChangeName(name) == nil {
			t.Fatalf("%q debe rechazarse", name)
		}
	}
	for _, name := range []string{"US-1-x", "otro-cambio", "a.b"} {
		if err := ValidChangeName(name); err != nil {
			t.Fatalf("%q debe aceptarse: %v", name, err)
		}
	}
}

func FuzzDeriveName(f *testing.F) {
	for _, s := range []string{"", "Alta de usuarios", "a..b", "ñandú ß æ", "\x00\xff", strings.Repeat("x", 500), "@{", "con.lock"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, title string) {
		got := DeriveName("US", "2525", title)
		if err := ValidChangeName(got); err != nil {
			t.Fatalf("título %q: %q inválido: %v", title, got, err)
		}
		if len(got) > 48 || !utf8.ValidString(got) {
			t.Fatalf("título %q: %q fuera de límites", title, got)
		}
	})
}
