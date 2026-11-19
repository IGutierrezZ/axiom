package secrets

import (
	"context"
	"errors"
	"testing"
)

func TestRefEnvVar(t *testing.T) {
	if got := (Ref{Plugin: "fake", Name: "token"}).EnvVar(); got != "AXIOM_INTEGRATION_FAKE_TOKEN" {
		t.Fatalf("EnvVar = %q", got)
	}
	if got := (Ref{Plugin: "fake", Name: "token"}).String(); got != "fake/token" {
		t.Fatalf("String = %q", got)
	}
}

func TestEnvAdapter(t *testing.T) {
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", "desde-os")
	s, src, err := Env{}.Get(context.Background(), Ref{Plugin: "fake", Name: "token"})
	if err != nil || s.Reveal() != "desde-os" || src != SourceEnv {
		t.Fatalf("Env{} debe leer os.Getenv: %v %q %q", err, s.Reveal(), src)
	}
	if _, src, err := (Env{}).Get(context.Background(), Ref{Plugin: "fake", Name: "otro"}); !errors.Is(err, ErrNotFound) || src != SourceMissing {
		t.Fatalf("variable ausente: %v %q", err, src)
	}
}

func TestMissingErrorMatchesNotFound(t *testing.T) {
	err := error(&MissingError{Ref: Ref{Plugin: "fake", Name: "token"}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("un *MissingError debe cumplir errors.Is(ErrNotFound)")
	}
}

func TestMemoryStore(t *testing.T) {
	m := NewMemory()
	r := Ref{Plugin: "fake", Name: "token"}
	if _, _, err := m.Get(context.Background(), r); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get vacío: %v", err)
	}
	if err := m.Set(context.Background(), r, NewSecret("v")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if s, src, err := m.Get(context.Background(), r); err != nil || s.Reveal() != "v" || src != SourceKeyring {
		t.Fatalf("Get: %v %q %q", err, s.Reveal(), src)
	}
	if err := m.Delete(context.Background(), r); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var zero Memory
	if err := zero.Set(context.Background(), r, NewSecret("v")); err != nil {
		t.Fatalf("el valor cero de Memory también sirve: %v", err)
	}
}

func TestDefaultReadsEnvAndCanBeOverridden(t *testing.T) {
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", "x")
	if s, src, err := Default().Get(context.Background(), Ref{Plugin: "fake", Name: "token"}); err != nil || s.Reveal() != "x" || src != SourceEnv {
		t.Fatalf("Default() lee el entorno: %v %q %q", err, s.Reveal(), src)
	}

	inner := &fakeT{}
	override := NewMemory()
	_ = override.Set(context.Background(), Ref{Plugin: "fake", Name: "token"}, NewSecret("m"))
	SetDefaultForTest(inner, NewChain(Env{Getenv: func(string) string { return "" }}, override))
	if s, _, err := Default().Get(context.Background(), Ref{Plugin: "fake", Name: "token"}); err != nil || s.Reveal() != "m" {
		t.Fatalf("SetDefaultForTest: %v %q", err, s.Reveal())
	}
	inner.run()
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", "otro")
	if s, _, err := Default().Get(context.Background(), Ref{Plugin: "fake", Name: "token"}); err != nil || s.Reveal() != "otro" {
		t.Fatal("Cleanup debe restaurar el almacén de producción")
	}
}

type fakeT struct{ cleanups []func() }

func (f *fakeT) Helper()           {}
func (f *fakeT) Cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }
func (f *fakeT) run() {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
}
