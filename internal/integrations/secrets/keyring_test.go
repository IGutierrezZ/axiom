//go:build windows || darwin

package secrets

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

var kref = Ref{Plugin: "fake", Name: "token"}

func TestKeyringSetGetDeleteUsesServiceAxiomAndAccountPluginSecret(t *testing.T) {
	keyring.MockInit()
	k := Keyring{}
	ctx := context.Background()

	if err := k.Set(ctx, kref, NewSecret("valor")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if raw, err := keyring.Get("axiom", "fake/token"); err != nil || raw != "valor" {
		t.Fatalf("servicio axiom y cuenta <plugin>/<secreto>: %q %v", raw, err)
	}
	got, src, err := k.Get(ctx, kref)
	if err != nil || got.Reveal() != "valor" || src != SourceKeyring {
		t.Fatalf("Get: %v %v %q", got, err, src)
	}
	if err := k.Delete(ctx, kref); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := k.Delete(ctx, kref); err != nil {
		t.Fatalf("Delete de un secreto inexistente es idempotente: %v", err)
	}
	if _, _, err := k.Get(ctx, kref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("tras Delete: %v", err)
	}
}

func TestChainWithKeyringNotFoundIsMissingError(t *testing.T) {
	keyring.MockInit()
	chain := NewChain(Env{Getenv: func(string) string { return "" }}, Keyring{})
	_, src, err := chain.Get(context.Background(), kref)
	var missing *MissingError
	if !errors.As(err, &missing) || src != SourceMissing {
		t.Fatalf("ErrNotFound da *MissingError: %v %q", err, src)
	}
}

func TestChainWithFailingKeyringIsUnavailableAndFallsBackToNothing(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	keyring.MockInitWithError(errors.New("sin servicio"))
	chain := NewChain(Env{Getenv: func(string) string { return "" }}, Keyring{})

	_, src, err := chain.Get(context.Background(), kref)
	var unavailable *KeyringUnavailableError
	if !errors.As(err, &unavailable) || src != SourceKeyringUnavailable || unavailable.EnvVar != "AXIOM_INTEGRATION_FAKE_TOKEN" {
		t.Fatalf("keyring caído: %v %q", err, src)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("nunca se crea un fichero de respaldo: %v", entries)
	}
}

func TestEnvStillWinsOverKeyring(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("axiom", "fake/token", "del-keyring")
	chain := NewChain(Env{Getenv: func(string) string { return "del-entorno" }}, Keyring{})
	got, src, err := chain.Get(context.Background(), kref)
	if err != nil || got.Reveal() != "del-entorno" || src != SourceEnv {
		t.Fatalf("el entorno gana: %v %q", err, src)
	}
}

func TestDefaultComposesEnvThenKeyring(t *testing.T) {
	keyring.MockInit()
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", "")
	if err := Default().Set(context.Background(), kref, NewSecret("v")); err != nil {
		t.Fatalf("Default().Set: %v", err)
	}
	got, src, err := Default().Get(context.Background(), kref)
	if err != nil || got.Reveal() != "v" || src != SourceKeyring {
		t.Fatalf("Default().Get: %v %q", err, src)
	}
}
