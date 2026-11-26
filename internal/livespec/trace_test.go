package livespec

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Las pruebas de la traza de identificadores asignados (D12, REQ-47.6). Todos los textos
// son inventados.

const goodTrace = `schema: axiom.requirement-ids/v1
pattern: REQ-<n>.<m>
assignments:
  - provisional: prov:30:1
    definitive: REQ-30.7
    capability: pagos
    requirement: Alta de cobro (REQ-30.7)
`

func wantPlanError(t *testing.T, err error, code string) *IDPlanError {
	t.Helper()
	var planErr *IDPlanError
	if !errors.As(err, &planErr) {
		t.Fatalf("error = %v, quería un *IDPlanError con código %s", err, code)
	}
	if planErr.Code != code {
		t.Fatalf("código = %q, quería %q (%v)", planErr.Code, code, err)
	}
	if planErr.Continuation == "" {
		t.Errorf("el error %s no lleva continuación", code)
	}
	return planErr
}

func TestTraceParsesAndValidatesAGoodTrace(t *testing.T) {
	trace, err := ParseTrace([]byte(goodTrace), "ids.yaml")
	if err != nil {
		t.Fatalf("ParseTrace: %v", err)
	}
	if err := trace.validate(reqid.Default(), "ids.yaml"); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(trace.Assignments) != 1 || trace.Assignments[0].Definitive != "REQ-30.7" || trace.Assignments[0].Capability != "pagos" {
		t.Fatalf("trace = %+v", trace)
	}
}

func TestTraceRejectsAnUnreadableOrUnknownTrace(t *testing.T) {
	for name, text := range map[string]string{
		"yaml ilegible":       "schema: [axiom\n  : :",
		"esquema desconocido": strings.Replace(goodTrace, "axiom.requirement-ids/v1", "axiom.requirement-ids/v9", 1),
		"sin esquema":         "pattern: REQ-<n>.<m>\nassignments: []\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseTrace([]byte(text), "ids.yaml")
			planErr := wantPlanError(t, err, CodeIDTraceInvalid)
			if planErr.File != "ids.yaml" {
				t.Errorf("File = %q", planErr.File)
			}
		})
	}
}

func TestTraceRejectsAPatternMismatchAndAnInvalidDefinitive(t *testing.T) {
	other, err := ParseTrace([]byte(strings.Replace(goodTrace, "REQ-<n>.<m>", "RF-<DOM>-<NN>", 1)), "ids.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wantPlanError(t, other.validate(reqid.Default(), "ids.yaml"), CodeIDTracePatternMismatch)

	bad, err := ParseTrace([]byte(strings.Replace(goodTrace, "REQ-30.7\n", "FOO-1\n", 1)), "ids.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wantPlanError(t, bad.validate(reqid.Default(), "ids.yaml"), CodeIDTraceInvalid)
}

func TestTraceRenderWritesNothingWithoutAssignmentsAndRoundTrips(t *testing.T) {
	empty := Trace{Schema: traceSchema, Pattern: "REQ-<n>.<m>"}
	if got := empty.Render(); got != nil {
		t.Fatalf("Render sin asignaciones = %q, quería nada", got)
	}
	trace, err := ParseTrace([]byte(goodTrace), "ids.yaml")
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseTrace(trace.Render(), "ids.yaml")
	if err != nil {
		t.Fatalf("la traza renderizada no se lee: %v", err)
	}
	if !reflect.DeepEqual(again.Assignments, trace.Assignments) {
		t.Fatalf("ida y vuelta: %+v != %+v", again, trace)
	}
}

func TestTraceReadFileOfAMissingTraceIsTheEmptyTrace(t *testing.T) {
	trace, err := ReadTraceFile(t.TempDir() + "/ids.yaml")
	if err != nil || len(trace.Assignments) != 0 {
		t.Fatalf("ReadTraceFile = %+v, %v", trace, err)
	}
}

const oddDocument = "# Mi feature\n\nTexto.\n\n## Spec viva\n\n### Capacidad: pagos\n\n```openspec\n## ADDED Requirements\n\n### Requirement: Alta (prov:30:1)\n\nCuerpo.\n```\n"

func oddTrace() Trace {
	trace, _ := ParseTrace([]byte(goodTrace), "doc")
	return trace
}

func TestODDTraceSectionIsAppendedAndReplacedWithoutTouchingTheRest(t *testing.T) {
	withTrace, err := SetODDTrace(oddDocument, oddTrace())
	if err != nil {
		t.Fatalf("SetODDTrace: %v", err)
	}
	if !strings.HasPrefix(withTrace, oddDocument) {
		t.Fatalf("el resto del documento cambió:\n%s", withTrace)
	}
	if !strings.Contains(withTrace, "\n## Identificadores asignados\n\n```yaml\nschema: axiom.requirement-ids/v1\n") {
		t.Fatalf("falta la sección de traza:\n%s", withTrace)
	}
	got, found, err := ExtractODDTrace(withTrace, "doc")
	if err != nil || !found || got.Assignments[0].Definitive != "REQ-30.7" {
		t.Fatalf("ExtractODDTrace = %+v, %v, %v", got, found, err)
	}

	changed := oddTrace()
	changed.Assignments[0].Definitive = "REQ-30.9"
	replaced, err := SetODDTrace(withTrace+"\n## Otra sección\n\nTexto final.\n", changed)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(replaced, "## Identificadores asignados") != 1 || !strings.Contains(replaced, "definitive: REQ-30.9") || strings.Contains(replaced, "definitive: REQ-30.7") {
		t.Fatalf("la sección no se reemplazó:\n%s", replaced)
	}
	if !strings.HasSuffix(replaced, "\n## Otra sección\n\nTexto final.\n") {
		t.Fatalf("la sección posterior cambió:\n%s", replaced)
	}
}

func TestODDTraceWithoutAssignmentsLeavesTheDocumentAndFindsNothing(t *testing.T) {
	same, err := SetODDTrace(oddDocument, Trace{Schema: traceSchema, Pattern: "REQ-<n>.<m>"})
	if err != nil || same != oddDocument {
		t.Fatalf("SetODDTrace vacío = %q, %v", same, err)
	}
	_, found, err := ExtractODDTrace(oddDocument, "doc")
	if err != nil || found {
		t.Fatalf("ExtractODDTrace sin sección = %v, %v", found, err)
	}
}

func TestODDTraceRejectsADuplicatedSectionAndAnIllegibleBlock(t *testing.T) {
	section := "\n## Identificadores asignados\n\n```yaml\n" + goodTrace + "```\n"
	_, _, err := ExtractODDTrace(oddDocument+section+section, "doc")
	wantPlanError(t, err, CodeIDTraceInvalid)
	_, _, err = ExtractODDTrace(oddDocument+"\n## Identificadores asignados\n\n```yaml\n: : [\n```\n", "doc")
	wantPlanError(t, err, CodeIDTraceInvalid)
	_, _, err = ExtractODDTrace(oddDocument+"\n## Identificadores asignados\n\nSin bloque.\n", "doc")
	wantPlanError(t, err, CodeIDTraceInvalid)
	// Un encabezado dentro de una valla no es una sección.
	_, found, err := ExtractODDTrace(oddDocument+"\n```\n## Identificadores asignados\n```\n", "doc")
	if err != nil || found {
		t.Fatalf("encabezado vallado = %v, %v", found, err)
	}
}
