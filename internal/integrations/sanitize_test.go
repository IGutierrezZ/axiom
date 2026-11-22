package integrations

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

const canarySecret = "s3cr3t-CANARIO-9f8e7d"

func TestSanitizeReplacesTrackedValues(t *testing.T) {
	tr := newSecretTracker()
	tr.track(canarySecret)
	tr.track("otro-valor-largo")

	got := sanitizeText("falló con "+canarySecret+" y de nuevo "+canarySecret+"; otro-valor-largo", tr)
	if strings.Contains(got, canarySecret) || strings.Contains(got, "otro-valor-largo") {
		t.Fatalf("no debe quedar ningún valor rastreado: %q", got)
	}
	if want := "falló con [REDACTED] y de nuevo [REDACTED]; [REDACTED]"; got != want {
		t.Fatalf("sanitizeText = %q, quería %q", got, want)
	}
}

func TestSanitizeIgnoresValuesShorterThanFourCharacters(t *testing.T) {
	tr := newSecretTracker()
	tr.track("abc")
	tr.track("")
	tr.track("abcd")
	got := sanitizeText("abc abcd", tr)
	if got != "abc [REDACTED]" {
		t.Fatalf("solo se redactan los valores de al menos 4 caracteres: %q", got)
	}
}

func TestSanitizeReplacesTheLongestValueFirst(t *testing.T) {
	tr := newSecretTracker()
	tr.track("abcd")
	tr.track("abcdefgh")
	if got := sanitizeText("clave abcdefgh", tr); got != "clave [REDACTED]" {
		t.Fatalf("un valor contenido en otro no debe dejar un resto del mayor: %q", got)
	}
}

func TestSanitizeFlattensToOneLineAndTrimsTo512Characters(t *testing.T) {
	tr := newSecretTracker()
	got := sanitizeText("línea uno\nlínea dos\r\n\tlínea tres\n\n", tr)
	if got != "línea uno línea dos línea tres" {
		t.Fatalf("aplana a una línea con espacios simples: %q", got)
	}

	long := strings.Repeat("ñ", 600)
	got = sanitizeText(long, tr)
	if utf8.RuneCountInString(got) > maxErrorChars || !utf8.ValidString(got) {
		t.Fatalf("recorta a %d caracteres sin partir un carácter, tiene %d", maxErrorChars, utf8.RuneCountInString(got))
	}
	if maxErrorChars != 512 {
		t.Fatalf("maxErrorChars = %d, el diseño fija 512", maxErrorChars)
	}
}

func TestSanitizeRedactsBeforeTrimmingSoNothingLeaksAtTheCut(t *testing.T) {
	tr := newSecretTracker()
	tr.track(canarySecret)
	// El valor cae justo a caballo del límite de 512 caracteres.
	text := strings.Repeat("x", maxErrorChars-5) + canarySecret + " fin"
	got := sanitizeText(text, tr)
	if strings.Contains(got, canarySecret[:6]) {
		t.Fatalf("ni siquiera un prefijo del valor puede salir: %q", got[len(got)-30:])
	}
	if utf8.RuneCountInString(got) > maxErrorChars {
		t.Fatalf("supera el máximo: %d", utf8.RuneCountInString(got))
	}
}

func TestSanitizeErrorHandlesNil(t *testing.T) {
	if got := sanitizeError(nil, newSecretTracker()); got != "" {
		t.Fatalf("sanitizeError(nil) = %q", got)
	}
	if got := sanitizeError(errors.New("hola\nmundo"), nil); got != "hola mundo" {
		t.Fatalf("sin rastreador solo aplana: %q", got)
	}
}

func TestScopedSecretsResolveOnlyDeclaredSecretsAndTrackThem(t *testing.T) {
	mem := secrets.NewMemory()
	_ = mem.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(canarySecret))
	_ = mem.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "otro"}, secrets.NewSecret("no-declarado-valor"))
	store := secrets.NewChain(secrets.Env{Getenv: func(string) string { return "" }}, mem)

	tr := newSecretTracker()
	source := newScopedSecrets(testManifest(), store, tr)

	got, err := source.Secret(context.Background(), "token")
	if err != nil || got.Reveal() != canarySecret {
		t.Fatalf("Secret(token) = %v, %v", got, err)
	}
	if sanitizeText("error "+canarySecret, tr) != "error [REDACTED]" {
		t.Fatal("el valor resuelto debe quedar rastreado")
	}

	_, err = source.Secret(context.Background(), "otro")
	if err == nil || !strings.Contains(err.Error(), "otro") || !strings.Contains(err.Error(), "token") {
		t.Fatalf("un secreto no declarado da error nombrándolo y listando los declarados: %v", err)
	}
	if strings.Contains(sanitizeText("no-declarado-valor", tr), "[REDACTED]") {
		t.Fatal("un secreto no declarado no se resuelve ni se rastrea")
	}
}

func TestScopedSecretsPropagateMissingWithoutValue(t *testing.T) {
	store := secrets.NewChain(secrets.Env{Getenv: func(string) string { return "" }}, secrets.NewMemory())
	source := newScopedSecrets(testManifest(), store, newSecretTracker())
	_, err := source.Secret(context.Background(), "token")
	var missing *secrets.MissingError
	if !errors.As(err, &missing) {
		t.Fatalf("un secreto ausente da *secrets.MissingError: %v", err)
	}
}

func TestAFakeThatLeaksTheTokenInItsErrorCannotLeakItOut(t *testing.T) {
	mem := secrets.NewMemory()
	_ = mem.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(canarySecret))
	store := secrets.NewChain(secrets.Env{Getenv: func(string) string { return "" }}, mem)
	tr := newSecretTracker()
	source := newScopedSecrets(testManifest(), store, tr)

	_, err := leakingPlugin().Deliver(context.Background(), Delivery{Secrets: source})
	if err == nil || !strings.Contains(err.Error(), canarySecret) {
		t.Fatalf("el doble debe filtrar el valor en su error crudo: %v", err)
	}
	if got := sanitizeError(err, tr); strings.Contains(got, canarySecret) || !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("el error saneado no puede llevar el valor: %q", got)
	}
}

func leakingPlugin() *testPlugin {
	p := newTestPlugin()
	p.readToken, p.leakToken = true, true
	return p
}
