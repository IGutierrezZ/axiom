package reqtrace

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func entry(prov, def string, prev ...string) livespec.TraceEntry {
	return livespec.TraceEntry{Provisional: prov, Definitive: def, Capability: "pagos", Requirement: "R", Previous: prev}
}

type mapAliases map[string][]string

func (m mapAliases) Aliases(id string) []string { return m[id] }

func resolveIDs(t *testing.T, raw string, traces []traceSet, aliases AliasResolver) (equivalence, []Warning) {
	t.Helper()
	q, err := ParseID(raw, reqid.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return resolveEquivalents(q, reqid.Default(), traces, aliases)
}

func TestRequirementTraceEquivalentsProvisionalAndDefinitiveMatch(t *testing.T) {
	traces := []traceSet{{Path: "a", Entries: []livespec.TraceEntry{entry("prov:trace:1", "REQ-30.1")}}}
	byDef, _ := resolveIDs(t, "REQ-30.1", traces, nil)
	byProv, warns := resolveIDs(t, "prov:trace:1", traces, nil)
	want := []string{"REQ-30.1", "prov:trace:1"}
	if !reflect.DeepEqual(byDef.resolved, want) || !reflect.DeepEqual(byProv.resolved, want) || len(warns) != 0 {
		t.Fatalf("definitivo=%v provisional=%v warns=%v; quería %v", byDef.resolved, byProv.resolved, warns, want)
	}
	if got := byProv.assignments("a"); len(got) != 1 || got[0].Provisional != "prov:trace:1" || got[0].Definitive != "REQ-30.1" {
		t.Fatalf("asignaciones = %+v", got)
	}
}

func TestRequirementTraceEquivalentsChainCycleAndSharedPrevious(t *testing.T) {
	chain := []traceSet{
		{Path: "a", Entries: []livespec.TraceEntry{entry("prov:a:1", "REQ-2.1", "REQ-1.1")}},
		{Path: "b", Entries: []livespec.TraceEntry{entry("prov:b:1", "REQ-3.1", "REQ-2.1")}},
	}
	eq, _ := resolveIDs(t, "REQ-1.1", chain, nil)
	if got := eq.resolved; !reflect.DeepEqual(got, []string{"REQ-1.1", "REQ-2.1", "REQ-3.1", "prov:a:1", "prov:b:1"}) {
		t.Fatalf("cadena A→B→C = %v", got)
	}
	cycle := []traceSet{
		{Path: "a", Entries: []livespec.TraceEntry{entry("prov:a:1", "REQ-1.1", "REQ-2.1")}},
		{Path: "b", Entries: []livespec.TraceEntry{entry("prov:b:1", "REQ-2.1", "REQ-1.1")}},
	}
	if eq, _ := resolveIDs(t, "REQ-2.1", cycle, nil); !reflect.DeepEqual(eq.resolved, []string{"REQ-1.1", "REQ-2.1", "prov:a:1", "prov:b:1"}) {
		t.Fatalf("ciclo = %v", eq.resolved)
	}
	shared := []traceSet{
		{Path: "a", Entries: []livespec.TraceEntry{entry("prov:a:1", "REQ-5.1", "REQ-4.1")}},
		{Path: "b", Entries: []livespec.TraceEntry{entry("prov:b:1", "REQ-6.1", "REQ-4.1")}},
	}
	if eq, _ := resolveIDs(t, "REQ-5.1", shared, nil); !reflect.DeepEqual(eq.resolved, []string{"REQ-4.1", "REQ-5.1", "REQ-6.1", "prov:a:1", "prov:b:1"}) {
		t.Fatalf("Previous compartido = %v", eq.resolved)
	}
}

func TestRequirementTraceEquivalentsAliasResolver(t *testing.T) {
	aliases := mapAliases{"REQ-30.1": {"REQ-9.9", "no-es-un-id"}}
	eq, _ := resolveIDs(t, "REQ-30.1", nil, aliases)
	if want := []string{"REQ-30.1", "REQ-9.9"}; !reflect.DeepEqual(eq.resolved, want) {
		t.Fatalf("alias inyectado = %v; quería %v (el que no cumple el patrón se descarta)", eq.resolved, want)
	}
	if !eq.matchSet("x")["REQ-9.9"] {
		t.Fatal("el alias debe estar en el conjunto de coincidencia")
	}
	if eq, _ := resolveIDs(t, "REQ-30.1", nil, NoAliases{}); !reflect.DeepEqual(eq.resolved, []string{"REQ-30.1"}) {
		t.Fatalf("NoAliases = %v", eq.resolved)
	}
	if eq, _ := resolveIDs(t, "REQ-30.1", nil, nil); !reflect.DeepEqual(eq.resolved, []string{"REQ-30.1"}) {
		t.Fatalf("sin resolutor = %v", eq.resolved)
	}
}

func TestRequirementTraceEquivalentsProvisionalAmbiguousAndPerSourceMatch(t *testing.T) {
	traces := []traceSet{
		{Path: "openspec/changes/archive/2026-01-01-a", Entries: []livespec.TraceEntry{entry("prov:x:1", "REQ-1.1")}},
		{Path: "openspec/changes/archive/2026-02-01-b", Entries: []livespec.TraceEntry{entry("prov:x:1", "REQ-2.1")}},
		{Path: "odd/tasks/c.md", Entries: []livespec.TraceEntry{entry("prov:x:2", "REQ-1.1")}},
	}
	eq, warns := resolveIDs(t, "prov:x:1", traces, nil)
	if len(warns) != 1 || warns[0].Code != WarnProvisionalAmbiguous ||
		!strings.Contains(warns[0].Message, "2026-01-01-a") || !strings.Contains(warns[0].Message, "2026-02-01-b") {
		t.Fatalf("advertencias = %+v", warns)
	}
	if want := []string{"REQ-1.1", "REQ-2.1", "prov:x:1", "prov:x:2"}; !reflect.DeepEqual(eq.resolved, want) {
		t.Fatalf("resueltos = %v; quería %v", eq.resolved, want)
	}
	// El provisional de otra fuente no cuenta: prov:x:2 es solo del documento ODD.
	if eq.matchSet("openspec/changes/archive/2026-01-01-a")["prov:x:2"] {
		t.Fatal("el provisional de otra fuente no debe casar")
	}
	if !eq.matchSet("odd/tasks/c.md")["prov:x:2"] {
		t.Fatal("el provisional de la propia fuente debe casar")
	}
}

func TestRequirementTraceEquivalentsUnknownIDResolvesToItself(t *testing.T) {
	eq, warns := resolveIDs(t, "REQ-77.7", []traceSet{{Path: "a", Entries: []livespec.TraceEntry{entry("prov:a:1", "REQ-1.1")}}}, nil)
	if !reflect.DeepEqual(eq.resolved, []string{"REQ-77.7"}) || len(warns) != 0 {
		t.Fatalf("ID sin traza = %v %v", eq.resolved, warns)
	}
	eq, _ = resolveIDs(t, "prov:q:1", nil, nil)
	if !reflect.DeepEqual(eq.resolved, []string{"prov:q:1"}) {
		t.Fatalf("provisional sin traza = %v", eq.resolved)
	}
}

const validTrace = `schema: axiom.requirement-ids/v1
pattern: REQ-<n>.<m>
assignments:
  - provisional: prov:trace:1
    definitive: REQ-30.1
    capability: pagos
    requirement: Cobro
`

func TestRequirementTraceReadTraces(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/ids.yaml": {Data: []byte(validTrace)},
		"changes/archive/2026-01-02-b/ids.yaml": {Data: []byte("schema: otro/v9\n")},
	}
	set, warns := readSDDTrace(files, "changes/archive/2026-01-01-a", "openspec/changes/archive/2026-01-01-a")
	if len(warns) != 0 || len(set.Entries) != 1 || set.Entries[0].Definitive != "REQ-30.1" {
		t.Fatalf("traza válida = %+v %v", set, warns)
	}
	set, warns = readSDDTrace(files, "changes/archive/2026-01-02-b", "openspec/changes/archive/2026-01-02-b")
	if len(warns) != 1 || warns[0].Code != WarnTraceInvalid || len(set.Entries) != 0 ||
		warns[0].Path != "openspec/changes/archive/2026-01-02-b/ids.yaml" {
		t.Fatalf("traza ilegible = %+v %v", set, warns)
	}
	if set, warns = readSDDTrace(files, "changes/archive/2026-01-03-c", "x"); len(warns) != 0 || len(set.Entries) != 0 {
		t.Fatalf("sin ids.yaml = %+v %v", set, warns)
	}

	doc := "# Doc\n\n## Identificadores asignados\n\n```yaml\n" + validTrace + "```\n"
	set, warns = readODDTrace(doc, "odd/tasks/f.md")
	if len(warns) != 0 || len(set.Entries) != 1 {
		t.Fatalf("traza ODD = %+v %v", set, warns)
	}
	broken := "## Identificadores asignados\n\nsin bloque yaml\n"
	if _, warns = readODDTrace(broken, "odd/tasks/g.md"); len(warns) != 1 || warns[0].Code != WarnTraceInvalid || warns[0].Path != "odd/tasks/g.md" {
		t.Fatalf("traza ODD ilegible = %v", warns)
	}
	if set, warns = readODDTrace("# sin traza\n", "odd/tasks/h.md"); len(warns) != 0 || len(set.Entries) != 0 {
		t.Fatalf("ODD sin traza = %+v %v", set, warns)
	}
}
