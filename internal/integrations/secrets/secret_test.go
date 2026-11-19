package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const canary = "s3cr3t-CANARIO-9f8e7d"

func TestSecretNeverLeaksInAnyFormat(t *testing.T) {
	s := NewSecret(canary)

	type holder struct {
		Token Secret `json:"token" yaml:"token"`
	}
	h := holder{Token: s}

	outputs := map[string]string{
		"%s":                fmt.Sprintf("%s", s),
		"%v":                fmt.Sprintf("%v", s),
		"%+v":               fmt.Sprintf("%+v", s),
		"%#v":               fmt.Sprintf("%#v", s),
		"%q":                fmt.Sprintf("%q", s),
		"%x":                fmt.Sprintf("%x", s),
		"%d":                fmt.Sprintf("%d", s),
		"%v de puntero":     fmt.Sprintf("%v", &s),
		"%+v de estructura": fmt.Sprintf("%+v", h),
		"%#v de estructura": fmt.Sprintf("%#v", h),
		"String":            s.String(),
		"GoString":          s.GoString(),
		"Sprint":            fmt.Sprint(s),
		"Sprintln":          fmt.Sprintln(s),
		"Errorf":            fmt.Errorf("fallo con %v", s).Error(),
	}

	mj, err := s.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	outputs["MarshalJSON"] = string(mj)

	hj, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	outputs["json.Marshal de estructura"] = string(hj)

	my, err := s.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	outputs["MarshalYAML"] = fmt.Sprint(my)

	hy, err := yaml.Marshal(h)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	outputs["yaml.Marshal de estructura"] = string(hy)

	mt, err := s.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	outputs["MarshalText"] = string(mt)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("entrega", "secreto", s)
	outputs["slog JSON"] = buf.String()
	buf.Reset()
	slog.New(slog.NewTextHandler(&buf, nil)).Info("entrega", "secreto", s)
	outputs["slog texto"] = buf.String()
	outputs["LogValue"] = s.LogValue().String()

	for name, out := range outputs {
		if strings.Contains(out, canary) {
			t.Errorf("%s filtra el valor del secreto: %q", name, out)
		}
		// %x y %d no imprimen el marcador por convención de fmt, pero Format los cubre igualmente.
		if !strings.Contains(out, redacted) {
			t.Errorf("%s debe contener %s: %q", name, redacted, out)
		}
	}
}

func TestSecretRevealReturnsValue(t *testing.T) {
	if got := NewSecret(canary).Reveal(); got != canary {
		t.Fatalf("Reveal() = %q, quería el valor", got)
	}
}

func TestSecretZeroValueIsRedactedAndEmpty(t *testing.T) {
	var s Secret
	if s.Reveal() != "" {
		t.Fatal("el valor cero revela cadena vacía")
	}
	if got := fmt.Sprintf("%v", s); got != redacted {
		t.Fatalf("el valor cero también se redacta, obtuvo %q", got)
	}
	if !s.IsZero() {
		t.Fatal("IsZero debe ser verdadero para el valor cero")
	}
	if NewSecret("x").IsZero() {
		t.Fatal("IsZero debe ser falso con valor")
	}
}
