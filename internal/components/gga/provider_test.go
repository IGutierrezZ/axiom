package gga

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func writeExistingConfig(t *testing.T, home, content string) string {
	t.Helper()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	path := ConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestInjectPreservesConfiguredProvider(t *testing.T) {
	// Se ejecuta dos veces: la primera es el install y la segunda el sync.
	custom := "# mi config\nexport PROVIDER='ollama:llama3'   # local\nTIMEOUT=\"60\"\nFILE_PATTERNS=\"*.rs\"\n"
	home := t.TempDir()
	path := writeExistingConfig(t, home, custom)

	for _, run := range []string{"install", "sync"} {
		result, err := Inject(home, []model.AgentID{model.AgentClaudeCode})
		if err != nil {
			t.Fatalf("%s: Inject() error = %v", run, err)
		}
		if result.ConfigChanged || result.ConfigFile != "" {
			t.Fatalf("%s: config reportado como escrito: %+v", run, result)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: leer config: %v", run, err)
		}
		if string(got) != custom {
			t.Fatalf("%s: el config cambió:\n%s", run, got)
		}
		// La plantilla AGENTS.md sí se escribe.
		if result.AgentsFile == "" {
			t.Fatalf("%s: AgentsFile vacío", run)
		}
	}
}

func TestInjectComputesProviderWhenMissing(t *testing.T) {
	tests := []struct {
		name     string
		existing *string
	}{
		{name: "sin config", existing: nil},
		{name: "config vacío", existing: strPtr("")},
		{name: "config sin proveedor", existing: strPtr("# solo comentarios\nTIMEOUT=\"60\"\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
			if tt.existing != nil {
				writeExistingConfig(t, home, *tt.existing)
			}

			result, err := Inject(home, []model.AgentID{model.AgentOpenCode})
			if err != nil {
				t.Fatalf("Inject() error = %v", err)
			}
			if !result.ConfigChanged || result.ConfigFile == "" {
				t.Fatalf("config no escrito: %+v", result)
			}
			got, err := os.ReadFile(ConfigPath(home))
			if err != nil {
				t.Fatalf("leer config: %v", err)
			}
			if string(got) != string(BuildConfig("opencode")) {
				t.Fatalf("config inesperado:\n%s", got)
			}
		})
	}
}

func TestInjectKeepsUnparseableConfig(t *testing.T) {
	custom := "PROVIDER=\"$(detect)\"\n"
	home := t.TempDir()
	path := writeExistingConfig(t, home, custom)

	result, err := Inject(home, []model.AgentID{model.AgentClaudeCode})
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if result.ConfigChanged {
		t.Fatal("ConfigChanged = true para un config ininterpretable")
	}
	got, _ := os.ReadFile(path)
	if string(got) != custom {
		t.Fatalf("el config cambió: %s", got)
	}
}

func TestExistingProvider(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantWord string
		wantSafe bool
	}{
		{"vacío", "", "", true},
		{"comillas dobles", `PROVIDER="claude"`, `"claude"`, true},
		{"comillas simples con export", "export PROVIDER='codex'", `'codex'`, true},
		{"sin comillas y comentario", "PROVIDER=gemini # nota", "gemini", true},
		{"ollama con modelo", `PROVIDER="ollama:llama3"`, `"ollama:llama3"`, true},
		{"otras claves sin proveedor", "TIMEOUT=\"300\"\n", "", true},
		{"proveedor vacío", `PROVIDER=""`, "", false},
		{"expansión", `PROVIDER="$P"`, "", false},
		{"sustitución de comando", "PROVIDER=$(x)", "", false},
		{"duplicado", "PROVIDER=a\nPROVIDER=b", "", false},
		{"línea que no es asignación", "if true; then\nfi", "", false},
		{"comilla sin cerrar", `PROVIDER="claude`, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			word, safe := existingProvider([]byte(tt.content))
			if word != tt.wantWord || safe != tt.wantSafe {
				t.Fatalf("existingProvider(%q) = (%q, %v), want (%q, %v)", tt.content, word, safe, tt.wantWord, tt.wantSafe)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
