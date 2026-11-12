package techcontext

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	testHead   = "0123456789abcdef0123456789abcdef01234567"
	testDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// planV1 es el plan que escribe `axiom sdd integrate prepare` sin baseline
// declarada: las claves de documentos, de correspondencia de IDs y de opciones
// están presentes y vacías. El decodificador de la traza debe tolerarlas.
const planV1 = `schema: axiom.sdd-integration/v1
change: inc-99-ejemplo
prepared_at: 2026-01-02T03:04:05Z
inputs_sha256: ` + testDigest + `
options:
  supersede: false
  superseded_requirements: []
repositories:
  - repo: .
    head: ` + testHead + `
    dirty: []
id_map: []
documents: []
context:
  decision: entries
entries:
  - id: C1
    kind: context
    document: context/ARQUITECTURA.md
    anchor: Componentes > Pagos
    action: modified
    state: pending
    reason: se añade el reintento
`

func TestReadTraceToleratesTheEmptyBaselineFieldsOfAV1Plan(t *testing.T) {
	trace, err := DecodeTrace([]byte(planV1))
	if err != nil {
		t.Fatalf("DecodeTrace: %v", err)
	}
	if trace.Change != "inc-99-ejemplo" || trace.Schema != PlanSchema || len(trace.Entries) != 1 {
		t.Fatalf("traza = %+v", trace)
	}
	entry := trace.Entries[0]
	if entry.Kind != EntryContext || entry.Action != ActionModified || entry.State != StatePending || entry.Anchor != "Componentes > Pagos" {
		t.Errorf("entrada = %+v", entry)
	}
	if trace.Context.Decision != ContextEntries {
		t.Errorf("decisión = %q", trace.Context.Decision)
	}
}

func TestDecodeTraceRejectsForeignPlans(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{"documentos", func(s string) string {
			return strings.Replace(s, "documents: []", "documents:\n  - domain: modelo\n    path: 03_Modelo.md\n    integration: compose\n    before_sha256: "+testDigest, 1)
		}, "documents"},
		{"correspondencia de IDs", func(s string) string {
			return strings.Replace(s, "id_map: []", "id_map:\n  - provisional: RF-9001\n    definitive: RF-008\n    domain: modelo", 1)
		}, "id_map"},
		{"supersede", func(s string) string { return strings.Replace(s, "supersede: false", "supersede: true", 1) }, "supersede"},
		{"entrada de composición", func(s string) string { return strings.Replace(s, "kind: context", "kind: compose", 1) }, "compose"},
		{"entrada dirigida", func(s string) string { return strings.Replace(s, "kind: context", "kind: directed", 1) }, "directed"},
		{"entrada de registro", func(s string) string { return strings.Replace(s, "kind: context", "kind: register", 1) }, "register"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeTrace([]byte(tt.mutate(planV1)))
			var foreign *ForeignPlanError
			if !errors.As(err, &foreign) {
				t.Fatalf("quería *ForeignPlanError, obtuve %T: %v", err, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("el error %q debe nombrar %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), "axiom sdd integrate") {
				t.Errorf("el error debe nombrar el verbo retirado `axiom sdd integrate`: %v", err)
			}
		})
	}
}

func TestDecodeTraceRejectsMalformedPlans(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"vacío", "  \n", "vacío"},
		{"esquema", strings.Replace(planV1, PlanSchema, "axiom.sdd-integration/v2", 1), "schema"},
		{"clave desconocida", planV1 + "extra: 1\n", "extra"},
		{"dos documentos YAML", planV1 + "---\nschema: otro\n", "un solo documento"},
		{"huella", strings.Replace(planV1, testDigest, "xyz", 1), "inputs_sha256"},
		{"decisión", strings.Replace(planV1, "decision: entries", "decision: quizá", 1), "context.decision"},
		{"ruta con barra invertida", strings.Replace(planV1, "document: context/ARQUITECTURA.md", `document: context\ARQUITECTURA.md`, 1), "document"},
		{"estado", strings.Replace(planV1, "state: pending", "state: hecho", 1), "state"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeTrace([]byte(tt.text))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("el error %v debe contener %q", err, tt.want)
			}
			var foreign *ForeignPlanError
			if errors.As(err, &foreign) {
				t.Errorf("un plan mal formado no es un plan ajeno: %v", err)
			}
		})
	}
}

func TestTraceMarshalIsDeterministicAndRoundTrips(t *testing.T) {
	trace, err := DecodeTrace([]byte(planV1))
	if err != nil {
		t.Fatal(err)
	}
	first, err := MarshalTrace(trace)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := MarshalTrace(trace)
	if string(first) != string(second) {
		t.Fatal("la misma traza debe producir los mismos bytes")
	}
	if string(first) != planV1 {
		t.Errorf("la serialización no reproduce el plan v1:\n%s\n--- frente a ---\n%s", first, planV1)
	}
	again, err := DecodeTrace(first)
	if err != nil {
		t.Fatalf("la traza serializada no se vuelve a leer: %v", err)
	}
	if again.Change != trace.Change || len(again.Entries) != 1 {
		t.Errorf("ida y vuelta = %+v", again)
	}
}

func TestReadAndWriteTrace(t *testing.T) {
	file := filepath.Join(t.TempDir(), TraceFileName)
	if _, err := ReadTrace(file); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("un plan que no existe envuelve fs.ErrNotExist: %v", err)
	}
	trace, err := DecodeTrace([]byte(planV1))
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteTrace(file, trace); err != nil {
		t.Fatalf("WriteTrace: %v", err)
	}
	read, err := ReadTrace(file)
	if err != nil || read.Change != trace.Change {
		t.Fatalf("ReadTrace = %+v, %v", read, err)
	}
	bad := trace
	bad.Change = ""
	if err := WriteTrace(file, bad); err == nil || !strings.Contains(err.Error(), "change") {
		t.Fatalf("una traza inválida no se escribe: %v", err)
	}
}

func TestWriteTraceIsAtomic(t *testing.T) {
	file := filepath.Join(t.TempDir(), TraceFileName)
	trace, _ := DecodeTrace([]byte(planV1))
	if err := WriteTrace(file, trace); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(file)

	original := renameFile
	renameFile = func(string, string) error { return errors.New("fallo inyectado") }
	defer func() { renameFile = original }()
	updated := trace
	updated.Change = "otro-cambio"
	if err := WriteTrace(file, updated); err == nil {
		t.Fatal("el fallo del renombrado debe llegar a quien llama")
	}
	after, _ := os.ReadFile(file)
	if string(before) != string(after) {
		t.Errorf("un fallo a mitad de la escritura no debe alterar el plan existente")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(file), ".*.tmp-*"))
	if len(leftovers) != 0 {
		t.Errorf("quedaron temporales: %v", leftovers)
	}
}

func TestRenderTrace(t *testing.T) {
	trace, _ := DecodeTrace([]byte(planV1))
	t.Run("markdown", func(t *testing.T) {
		data, err := RenderTrace(trace, TraceFormatMarkdown)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{"\n## Traza de integración\n", "Cambio: `inc-99-ejemplo`", "Verificada: pendiente", "| Identificador | Documento | Ancla | Acción | Estado |", "| C1 | context/ARQUITECTURA.md | Componentes > Pagos | modified | pending |"} {
			if !strings.Contains(text, want) {
				t.Errorf("el Markdown no contiene %q:\n%s", want, text)
			}
		}
	})
	t.Run("sin impacto", func(t *testing.T) {
		quiet := trace
		quiet.Entries = nil
		quiet.Context = ContextDecision{Decision: ContextNoImpact, Reason: "solo cambia código interno"}
		data, _ := RenderTrace(quiet, TraceFormatMarkdown)
		for _, want := range []string{"La integración no registra entradas.", "### Decisión sobre el contexto técnico", "Sin impacto: solo cambia código interno"} {
			if !strings.Contains(string(data), want) {
				t.Errorf("falta %q:\n%s", want, data)
			}
		}
	})
	t.Run("json", func(t *testing.T) {
		data, err := RenderTrace(trace, TraceFormatJSON)
		if err != nil {
			t.Fatal(err)
		}
		var view map[string]any
		if err := json.Unmarshal(data, &view); err != nil {
			t.Fatalf("JSON inválido: %v", err)
		}
		if view["schema"] != TraceSchema || view["change"] != "inc-99-ejemplo" {
			t.Errorf("vista = %v", view)
		}
		if entries, _ := view["entries"].([]any); len(entries) != 1 {
			t.Errorf("entradas = %v", view["entries"])
		}
		if ids, ok := view["idMap"].([]any); !ok || len(ids) != 0 {
			t.Errorf("idMap = %v, quería una lista vacía", view["idMap"])
		}
	})
	t.Run("formato desconocido", func(t *testing.T) {
		if _, err := RenderTrace(trace, "xml"); err == nil || !strings.Contains(err.Error(), "markdown") {
			t.Fatalf("el formato desconocido nombra los válidos: %v", err)
		}
	})
}

func TestTraceTimesAreUTC(t *testing.T) {
	trace, _ := DecodeTrace([]byte(planV1))
	zone := time.FixedZone("x", 3600)
	trace.PreparedAt = time.Date(2026, 1, 2, 4, 4, 5, 0, zone)
	verified := time.Date(2026, 1, 2, 5, 4, 5, 0, zone)
	trace.VerifiedAt = &verified
	data, err := MarshalTrace(trace)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "prepared_at: 2026-01-02T03:04:05Z") || !strings.Contains(string(data), "verified_at: 2026-01-02T04:04:05Z") {
		t.Errorf("las horas se escriben en UTC:\n%s", data)
	}
}
