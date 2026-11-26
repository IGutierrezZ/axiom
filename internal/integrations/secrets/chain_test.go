package secrets

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var ref = Ref{Plugin: "azure-devops", Name: "pat"}

// failingStore simula un keyring que falla con una causa distinta de «no existe».
type failingStore struct{ err error }

func (f failingStore) Get(context.Context, Ref) (Secret, Source, error) {
	return Secret{}, SourceMissing, f.err
}
func (f failingStore) Set(context.Context, Ref, Secret) error { return f.err }
func (f failingStore) Delete(context.Context, Ref) error      { return f.err }

func envWith(values map[string]string) Env {
	return Env{Getenv: func(k string) string { return values[k] }}
}

func TestChainEnvWinsOverSecondLink(t *testing.T) {
	mem := NewMemory()
	_ = mem.Set(context.Background(), ref, NewSecret("del-keyring"))
	chain := NewChain(envWith(map[string]string{"AXIOM_INTEGRATION_AZURE_DEVOPS_PAT": "del-entorno"}), mem)

	got, src, err := chain.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Reveal() != "del-entorno" || src != SourceEnv {
		t.Fatalf("el entorno debe ganar: valor=%q fuente=%q", got.Reveal(), src)
	}
}

func TestChainFallsBackToSecondLinkWithKeyringSource(t *testing.T) {
	mem := NewMemory()
	_ = mem.Set(context.Background(), ref, NewSecret("del-keyring"))
	chain := NewChain(envWith(nil), mem)

	got, src, err := chain.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Reveal() != "del-keyring" || src != SourceKeyring {
		t.Fatalf("valor=%q fuente=%q", got.Reveal(), src)
	}
}

func TestChainEmptyEnvCountsAsMissing(t *testing.T) {
	mem := NewMemory()
	_ = mem.Set(context.Background(), ref, NewSecret("del-keyring"))
	chain := NewChain(envWith(map[string]string{"AXIOM_INTEGRATION_AZURE_DEVOPS_PAT": ""}), mem)
	_, src, err := chain.Get(context.Background(), ref)
	if err != nil || src != SourceKeyring {
		t.Fatalf("una variable vacía cae al segundo eslabón: fuente=%q err=%v", src, err)
	}
}

func TestChainNotFoundIsMissingErrorWithoutValue(t *testing.T) {
	chain := NewChain(envWith(nil), NewMemory())
	got, src, err := chain.Get(context.Background(), ref)
	var missing *MissingError
	if !errors.As(err, &missing) {
		t.Fatalf("esperaba *MissingError, obtuvo %T %v", err, err)
	}
	if src != SourceMissing || !got.IsZero() {
		t.Fatalf("sin valor y con fuente missing, obtuvo %q %q", got.Reveal(), src)
	}
	msg := err.Error()
	for _, want := range []string{"axiom integrations secret set", "AXIOM_INTEGRATION_AZURE_DEVOPS_PAT", "azure-devops", "pat"} {
		if !strings.Contains(msg, want) {
			t.Errorf("el mensaje debe nombrar %q: %s", want, msg)
		}
	}
}

func TestChainWithoutSecondLinkIsMissing(t *testing.T) {
	chain := NewChain(envWith(nil), nil)
	_, src, err := chain.Get(context.Background(), ref)
	var missing *MissingError
	if !errors.As(err, &missing) || src != SourceMissing {
		t.Fatalf("sin segundo eslabón y sin variable: %v %q", err, src)
	}
}

func TestChainGenericErrorIsKeyringUnavailableAndCreatesNoFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cause := errors.New("sin servicio de secretos")
	chain := NewChain(envWith(nil), failingStore{err: cause})
	got, src, err := chain.Get(context.Background(), ref)

	var unavailable *KeyringUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("esperaba *KeyringUnavailableError, obtuvo %T %v", err, err)
	}
	if !errors.Is(err, cause) {
		t.Fatal("el error debe envolver la causa")
	}
	if src != SourceKeyringUnavailable || !got.IsZero() {
		t.Fatalf("fuente=%q valor=%q", src, got.Reveal())
	}
	if !strings.Contains(err.Error(), "AXIOM_INTEGRATION_AZURE_DEVOPS_PAT") {
		t.Fatalf("el mensaje debe nombrar la variable de entorno: %s", err)
	}
	if unavailable.EnvVar != "AXIOM_INTEGRATION_AZURE_DEVOPS_PAT" {
		t.Fatalf("EnvVar = %q", unavailable.EnvVar)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("nunca se crea un fichero de respaldo, hay %d entradas en %s", len(entries), filepath.Base(dir))
	}
}

func TestChainSetAndDeleteGoOnlyToSecondLink(t *testing.T) {
	mem := NewMemory()
	chain := NewChain(envWith(nil), mem)
	if err := chain.Set(context.Background(), ref, NewSecret("v")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got, _, err := mem.Get(context.Background(), ref); err != nil || got.Reveal() != "v" {
		t.Fatalf("Set debe llegar al segundo eslabón: %v", err)
	}
	if err := chain.Delete(context.Background(), ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := chain.Delete(context.Background(), ref); err != nil {
		t.Fatalf("Delete es idempotente: %v", err)
	}
	if _, _, err := mem.Get(context.Background(), ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("tras Delete no debe existir: %v", err)
	}
}

func TestChainWithoutSecondLinkRejectsWrites(t *testing.T) {
	chain := NewChain(envWith(nil), nil)
	var unavailable *KeyringUnavailableError
	if err := chain.Set(context.Background(), ref, NewSecret("v")); !errors.As(err, &unavailable) {
		t.Fatalf("Set sin segundo eslabón: %v", err)
	}
	if err := chain.Delete(context.Background(), ref); !errors.As(err, &unavailable) {
		t.Fatalf("Delete sin segundo eslabón: %v", err)
	}
}

func TestChainWriteFailureIsKeyringUnavailable(t *testing.T) {
	chain := NewChain(envWith(nil), failingStore{err: errors.New("denegado")})
	var unavailable *KeyringUnavailableError
	if err := chain.Set(context.Background(), ref, NewSecret("v")); !errors.As(err, &unavailable) {
		t.Fatalf("Set con fallo: %v", err)
	}
	if err := chain.Delete(context.Background(), ref); !errors.As(err, &unavailable) {
		t.Fatalf("Delete con fallo: %v", err)
	}
}

// Un valor por encima de MaxValueBytes (cabe en todos los almacenes del sistema) falla con un error
// tipado que guía hacia la variable de entorno y nunca refleja el valor.
func TestChainSetRejectsAValueAboveMaxValueBytesWithATypedError(t *testing.T) {
	ref := Ref{Plugin: "fake", Name: "token"}
	mem := NewMemory()
	chain := NewChain(Env{}, mem)
	value := strings.Repeat("x", MaxValueBytes+1)

	err := chain.Set(context.Background(), ref, NewSecret(value))

	var tooLarge *ValueTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("debe devolver *ValueTooLargeError, no %T: %v", err, err)
	}
	if tooLarge.Max != MaxValueBytes || tooLarge.EnvVar != ref.EnvVar() {
		t.Errorf("error = %+v", tooLarge)
	}
	msg := err.Error()
	if strings.Contains(msg, value) || !strings.Contains(msg, ref.EnvVar()) {
		t.Errorf("el mensaje debe guiar hacia %s y no contener el valor: %q", ref.EnvVar(), msg)
	}
	if _, _, getErr := mem.Get(context.Background(), ref); !errors.Is(getErr, ErrNotFound) {
		t.Error("un valor rechazado no se guarda")
	}
}

func TestChainSetAcceptsAValueOfExactlyMaxValueBytes(t *testing.T) {
	ref := Ref{Plugin: "fake", Name: "token"}
	chain := NewChain(Env{}, NewMemory())

	if err := chain.Set(context.Background(), ref, NewSecret(strings.Repeat("x", MaxValueBytes))); err != nil {
		t.Fatalf("el límite exacto debe caber: %v", err)
	}
}
