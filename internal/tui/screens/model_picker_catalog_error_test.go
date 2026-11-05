package screens

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
)

func degradedPickerState(t *testing.T, discoveryErr error) ModelPickerState {
	t.Helper()
	state := NewRuntimeModelPickerStateWithDiscoverer(filepath.Join(t.TempDir(), "missing-opencode.json"), func(context.Context, string) (map[string]opencode.Provider, error) {
		return nil, discoveryErr
	})
	state.ConfiguredProviders = map[string]opencode.Provider{
		"configured": {ID: "configured", Name: "Configured", Models: map[string]opencode.Model{
			"configured/model": {ID: "configured/model", ToolCall: true},
		}},
	}
	return state.Update(state.StartRuntimeCatalogDiscovery(1, "active-project")().(RuntimeCatalogDiscoveryMsg))
}

func TestPickerKeepsTypedCatalogErrorAndClearsItOnSuccess(t *testing.T) {
	state := degradedPickerState(t, &opencode.CatalogError{Kind: opencode.CatalogErrorTimeout})
	var catalogErr *opencode.CatalogError
	if !errors.As(state.CatalogError, &catalogErr) || catalogErr.Kind != opencode.CatalogErrorTimeout {
		t.Fatalf("CatalogError = %v, want the typed timeout", state.CatalogError)
	}
	state = state.Update(RuntimeCatalogDiscoveryMsg{RequestID: 1, ProjectDir: "active-project", Providers: map[string]opencode.Provider{
		"custom": {ID: "custom", Models: map[string]opencode.Model{"m": {ID: "m", ToolCall: true}}},
	}})
	if state.CatalogError != nil {
		t.Fatalf("CatalogError = %v after a successful discovery, want nil", state.CatalogError)
	}
	if strings.Contains(RenderModelPicker(nil, state, 0), "Se muestran solo los modelos configurados") {
		t.Fatal("a successful discovery must not render the degraded-catalog warning")
	}
}

func TestPickerWarnsOfDegradedCatalogInPhaseListAndProviderSelect(t *testing.T) {
	state := degradedPickerState(t, &opencode.CatalogError{Kind: opencode.CatalogErrorOutputTooLarge})
	if state.CatalogStatus != RuntimeCatalogReady {
		t.Fatalf("CatalogStatus = %v, want ready with configured models", state.CatalogStatus)
	}
	const tail = "Se muestran solo los modelos configurados"
	if got := RenderModelPicker(nil, state, 0); !strings.Contains(got, "supera el tamaño máximo") || !strings.Contains(got, tail) {
		t.Fatalf("phase list does not warn of the degraded catalog:\n%s", got)
	}
	state.Mode = ModeProviderSelect
	if got := RenderModelPicker(nil, state, 0); !strings.Contains(got, "supera el tamaño máximo") || !strings.Contains(got, tail) {
		t.Fatalf("provider selector does not warn of the degraded catalog:\n%s", got)
	}
}

func TestPickerCatalogFailureExplanationIsTruthfulPerKind(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    string
		notWant string
	}{
		{"binario ausente", &opencode.CatalogError{Kind: opencode.CatalogErrorMissingBinary}, "No se encontró el ejecutable", ""},
		{"plazo", &opencode.CatalogError{Kind: opencode.CatalogErrorTimeout}, "a tiempo", "instalado"},
		{"código de salida", &opencode.CatalogError{Kind: opencode.CatalogErrorCommandFailed}, "terminó con error", "instalado"},
		{"demasiado grande", &opencode.CatalogError{Kind: opencode.CatalogErrorOutputTooLarge}, "tamaño máximo", "instalado"},
		{"JSON inválido", &opencode.CatalogError{Kind: opencode.CatalogErrorMalformed}, "No se pudo interpretar", "instalado"},
		{"esquema no soportado", &opencode.CatalogError{Kind: opencode.CatalogErrorUnsupportedSchema}, "No se pudo interpretar", "instalado"},
		{"envuelto", fmt.Errorf("descubrimiento: %w", &opencode.CatalogError{Kind: opencode.CatalogErrorTimeout}), "a tiempo", ""},
		{"sin tipo", errors.New("secreto interno"), "No se pudo descubrir", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headline, next := modelPickerCatalogFailureExplanation(tt.err)
			text := headline + " " + next
			if !strings.Contains(text, tt.want) {
				t.Fatalf("explanation = %q, want %q", text, tt.want)
			}
			if tt.notWant != "" && strings.Contains(text, tt.notWant) {
				t.Fatalf("explanation = %q, must not claim %q", text, tt.notWant)
			}
			if strings.Contains(text, "secreto interno") || strings.Contains(text, "goroutine") {
				t.Fatalf("explanation = %q leaks error detail", text)
			}
		})
	}
}

func TestPickerFailedPhaseListNamesTheTypedCause(t *testing.T) {
	state := NewRuntimeModelPickerStateWithDiscoverer(filepath.Join(t.TempDir(), "missing-opencode.json"), func(context.Context, string) (map[string]opencode.Provider, error) {
		return nil, &opencode.CatalogError{Kind: opencode.CatalogErrorMalformed}
	})
	state = state.Update(state.StartRuntimeCatalogDiscovery(1, "active-project")().(RuntimeCatalogDiscoveryMsg))
	if state.CatalogStatus != RuntimeCatalogFailed {
		t.Fatalf("CatalogStatus = %v, want failed without configured models", state.CatalogStatus)
	}
	got := RenderModelPicker(nil, state, 0)
	if !strings.Contains(got, "No se pudo interpretar el catálogo") || strings.Contains(got, "instalado") {
		t.Fatalf("failed phase list must name the parse failure, not blame the installation:\n%s", got)
	}
}
