package azuredevops

import (
	"encoding/base64"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestIsRevTestRejection(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   errorBody
		want   bool
	}{
		{"412 siempre", 412, errorBody{}, true},
		{"412 con cualquier cuerpo", 412, errorBody{Message: "otra cosa"}, true},
		{"400 con typeKey", 400, errorBody{TypeKey: "TestOperationFailedException"}, true},
		{"400 con typeKey en minúsculas", 400, errorBody{TypeKey: "testoperationfailedexception"}, true},
		{"400 con el mensaje de test operation", 400, errorBody{Message: "The Test Operation at path '/rev' failed"}, true},
		{"409 con typeKey", 409, errorBody{TypeKey: "TestOperationFailedException"}, true},
		{"409 con el mensaje", 409, errorBody{Message: "test operation falló"}, true},
		{"400 ajeno (campo inexistente)", 400, errorBody{Message: "TF51535: Cannot find field System.Nope.", TypeKey: "RuleValidationException"}, false},
		{"409 ajeno", 409, errorBody{Message: "conflicto de otra clase"}, false},
		{"400 sin cuerpo", 400, errorBody{}, false},
		{"404 no es rechazo", 404, errorBody{TypeKey: "TestOperationFailedException"}, false},
		{"500 no es rechazo", 500, errorBody{Message: "test operation"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRevTestRejection(tt.status, tt.body); got != tt.want {
				t.Fatalf("isRevTestRejection(%d, %+v) = %v, esperaba %v", tt.status, tt.body, got, tt.want)
			}
		})
	}
}

func TestParseErrorBody(t *testing.T) {
	if got := parseErrorBody([]byte(`{"message":"hola","typeKey":"K"}`)); got.Message != "hola" || got.TypeKey != "K" {
		t.Fatalf("JSON de error = %+v", got)
	}
	if got := parseErrorBody([]byte("texto plano")); got.Message != "texto plano" || got.TypeKey != "" {
		t.Fatalf("texto plano = %+v", got)
	}
	if got := parseErrorBody([]byte(`{"otra":"cosa"}`)); got.Message != `{"otra":"cosa"}` {
		t.Fatalf("JSON ajeno se conserva como texto = %+v", got)
	}
}

func TestRedactor(t *testing.T) {
	const pat = "secreto-ficticio-0001"
	b64 := base64.StdEncoding.EncodeToString([]byte(":" + pat))
	r := newRedactor(pat)

	tests := map[string]string{
		"el PAT suelto":          "valor " + pat + " fin",
		"el base64":              "valor " + b64 + " fin",
		"la cabecera completa":   "Authorization: Basic " + b64,
		"varias apariciones":     pat + " " + pat + " " + b64,
		"mezclado en una cadena": "a" + pat + "b" + b64 + "c",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			got := r.redact(input)
			for _, leaked := range []string{pat, b64} {
				if strings.Contains(got, leaked) {
					t.Fatalf("queda %q en %q", leaked, got)
				}
			}
		})
	}
	if got := r.redact("Authorization: Basic " + b64); strings.Contains(got, "Basic") {
		t.Fatalf("la forma «Basic <b64>» se redacta entera, sin dejar el prefijo: %q", got)
	}
	if got := r.redact("texto sin secretos"); got != "texto sin secretos" {
		t.Fatalf("no debe alterar texto limpio: %q", got)
	}
}

func TestRedactorIgnoresValuesShorterThanFourCharacters(t *testing.T) {
	r := newRedactor("abc")
	if got := r.redact("abc def"); !strings.Contains(got, "abc") {
		t.Fatalf("un valor de menos de 4 caracteres no se redacta (borraría texto legítimo): %q", got)
	}
}

func TestDetailOfFlattensAndTruncates(t *testing.T) {
	r := newRedactor("pat-ficticio")
	if got := r.detailOf("una\n\n  línea\tcon   espacios"); got != "una línea con espacios" {
		t.Fatalf("detailOf = %q", got)
	}
	long := strings.Repeat("é", 2000)
	got := r.detailOf(long)
	if n := len([]rune(got)); n != maxErrorDetail+1 || !strings.HasSuffix(got, "…") {
		t.Fatalf("detalle largo: %d runas, sufijo %q", n, got[len(got)-3:])
	}
}

func TestBackoff(t *testing.T) {
	want := map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 4: 8 * time.Second, 5: 16 * time.Second, 6: 16 * time.Second, 20: 16 * time.Second}
	for attempt, d := range want {
		if got := backoff(attempt); got != d {
			t.Errorf("backoff(%d) = %v, esperaba %v", attempt, got, d)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		value  string
		want   time.Duration
		wantOK bool
	}{
		{"", 0, false},
		{"2", 2 * time.Second, true},
		{" 30 ", 30 * time.Second, true},
		{"0", 0, true},
		{"-1", 0, false},
		{"mañana", 0, false},
		// Valores que desbordarían time.Duration se saturan al máximo en lugar
		// de dar la vuelta a una duración negativa o pequeña (JD-1).
		{"9223372037", time.Duration(math.MaxInt64), true},
		{"10000000000", time.Duration(math.MaxInt64), true},
		{"99999999999999999999", time.Duration(math.MaxInt64), true},
		{"-99999999999999999999", 0, false},
		{now.Add(7 * time.Second).Format(http.TimeFormat), 7 * time.Second, true},
		{now.Add(-7 * time.Second).Format(http.TimeFormat), 0, true},
	}
	for _, tt := range tests {
		got, ok := parseRetryAfter(tt.value, now)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("parseRetryAfter(%q) = %v, %v; esperaba %v, %v", tt.value, got, ok, tt.want, tt.wantOK)
		}
	}
}
