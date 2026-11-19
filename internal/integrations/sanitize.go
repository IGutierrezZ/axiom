package integrations

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

const (
	// maxErrorChars es el tope de caracteres de un texto de error antes de guardarlo o imprimirlo.
	maxErrorChars = 512
	// minTrackedSecretLen es la longitud mínima de un valor para redactarlo: un valor más corto
	// borraría fragmentos de texto corriente.
	minTrackedSecretLen = 4
	redactedMarker      = "[REDACTED]"
)

// secretTracker apunta en memoria cada valor de secreto resuelto durante una entrega, para limpiar
// de cualquier texto de error los valores que un plugin haya envuelto en él (D-12).
type secretTracker struct {
	mu     sync.Mutex
	values map[string]struct{}
}

func newSecretTracker() *secretTracker { return &secretTracker{values: map[string]struct{}{}} }

func (t *secretTracker) track(value string) {
	if t == nil || len(value) < minTrackedSecretLen {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.values[value] = struct{}{}
}

// ordered devuelve los valores rastreados del más largo al más corto: así uno contenido en otro no
// deja un resto del mayor.
func (t *secretTracker) ordered() []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	values := make([]string, 0, len(t.values))
	for v := range t.values {
		values = append(values, v)
	}
	sort.Slice(values, func(i, j int) bool {
		if len(values[i]) != len(values[j]) {
			return len(values[i]) > len(values[j])
		}
		return values[i] < values[j]
	})
	return values
}

// sanitizeText sustituye por [REDACTED] cada valor rastreado, aplana el texto a una línea y lo
// recorta a maxErrorChars caracteres. La redacción va antes del recorte para que un valor a
// caballo del límite no salga ni en prefijo.
func sanitizeText(text string, tracker *secretTracker) string {
	for _, value := range tracker.ordered() {
		text = strings.ReplaceAll(text, value, redactedMarker)
	}
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) > maxErrorChars {
		runes := []rune(text)
		text = string(runes[:maxErrorChars])
	}
	return text
}

// sanitizeError es sanitizeText sobre el mensaje de err; un error nulo da la cadena vacía.
func sanitizeError(err error, tracker *secretTracker) string {
	if err == nil {
		return ""
	}
	return sanitizeText(err.Error(), tracker)
}

// scopedSecrets es el SecretSource que recibe un plugin: acotado a los secretos de su propio
// manifiesto y con rastreo de cada valor resuelto.
type scopedSecrets struct {
	plugin   string
	declared []string
	store    secrets.Store
	tracker  *secretTracker
}

// newScopedSecrets acota store a los secretos que declara m.
func newScopedSecrets(m manifest.Manifest, store secrets.Store, tracker *secretTracker) SecretSource {
	declared := make([]string, len(m.Secrets))
	for i, s := range m.Secrets {
		declared[i] = s.Name
	}
	return &scopedSecrets{plugin: m.ID, declared: declared, store: store, tracker: tracker}
}

func (s *scopedSecrets) Secret(ctx context.Context, name string) (secrets.Secret, error) {
	known := false
	for _, d := range s.declared {
		if d == name {
			known = true
			break
		}
	}
	if !known {
		list := "ninguno"
		if len(s.declared) > 0 {
			list = strings.Join(s.declared, ", ")
		}
		return secrets.Secret{}, fmt.Errorf("el plugin %s no declara el secreto %q; secretos declarados: %s", s.plugin, name, list)
	}
	secret, _, err := s.store.Get(ctx, secrets.Ref{Plugin: s.plugin, Name: name})
	if err != nil {
		return secrets.Secret{}, err
	}
	s.tracker.track(secret.Reveal())
	return secret, nil
}
