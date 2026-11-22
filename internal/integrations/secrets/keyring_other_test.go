//go:build !windows && !darwin

package secrets

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// En Linux y el resto de sistemas el único almacén es la variable de entorno: `set` se niega con un
// error tipado que remite a la variable, sin escribir nada.
func TestDefaultHasNoKeyringOutsideWindowsAndDarwin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", "")
	ref := Ref{Plugin: "fake", Name: "token"}

	err := Default().Set(context.Background(), ref, NewSecret("v"))
	var unavailable *KeyringUnavailableError
	if !errors.As(err, &unavailable) || !errors.Is(err, ErrNoWritableStore) || !strings.Contains(err.Error(), "AXIOM_INTEGRATION_FAKE_TOKEN") {
		t.Fatalf("set debe negarse con un error tipado que nombre la variable: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("nunca se escribe texto plano: %v", entries)
	}
	if _, src, err := Default().Get(context.Background(), ref); src != SourceMissing || err == nil {
		t.Fatalf("sin variable el secreto falta: %v %q", err, src)
	}
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", "x")
	if got, src, err := Default().Get(context.Background(), ref); err != nil || got.Reveal() != "x" || src != SourceEnv {
		t.Fatalf("con variable: %v %q", err, src)
	}
}
