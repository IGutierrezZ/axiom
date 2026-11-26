package reqtrace

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IGutierrezZ/axiom/v3/internal/legacyid"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const legacyMapText = `schema: axiom.id-map/v1
entries:
  - capability: autoskills
    old: [REQ-1.1]
    new: RF-ASK-01
  - capability: workspace-topology
    old: [REQ-1.1]
    new: RF-WTP-01
  - capability: pagos
    old: [REQ-24.3]
    new: RF-PAG-03
  - capability: sdd-kickoff
    old: [REQ-27.2b]
    new: RF-KCK-09
  - capability: sdd-otro
    old: [REQ-27.2]
    new: RF-OTR-02
  - capability: persona
    old: [REQ-10.1, REQ-10.2]
    new: RF-PER-08
`

func rfPattern(t *testing.T) reqid.Pattern {
	t.Helper()
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func loadLegacy(t *testing.T) *legacyid.Map {
	t.Helper()
	m, err := legacyid.Load(fstest.MapFS{legacyid.FileName: {Data: []byte(legacyMapText)}}, legacyid.FileName, rfPattern(t))
	if err != nil || m == nil {
		t.Fatalf("mapa de prueba: %v, %v", m, err)
	}
	return m
}

func legacyWarnings(ws []Warning) []Warning {
	var out []Warning
	for _, w := range ws {
		if w.Code == WarnLegacyAmbiguous {
			out = append(out, w)
		}
	}
	return out
}

func TestRequirementTraceParseIDLegacy(t *testing.T) {
	p := rfPattern(t)
	m := loadLegacy(t)
	got, err := ParseID("REQ-1.1", p, m)
	if err != nil || got != (Query{ID: "REQ-1.1", Legacy: true}) {
		t.Fatalf("antiguo del mapa = %+v, %v", got, err)
	}
	got, err = ParseID("REQ-27.2b", p, m)
	if err != nil || !got.Legacy {
		t.Fatalf("REQ-27.2b como clave antigua = %+v, %v", got, err)
	}
	if got, err = ParseID("RF-ASK-01", p, m); err != nil || got != (Query{ID: "RF-ASK-01"}) {
		t.Fatalf("ID nuevo = %+v, %v", got, err)
	}
	var idErr *IDError
	if _, err = ParseID("REQ-99.9", p, m); !errors.As(err, &idErr) || idErr.ID != "REQ-99.9" || idErr.Template != p.Template() {
		t.Fatalf("antiguo fuera del mapa debe ser *IDError con el patrón del proyecto: %v", err)
	}
	if _, err = ParseID("REQ-1.1", p, nil); !errors.As(err, &idErr) {
		t.Fatalf("sin mapa un REQ-n.m es *IDError: %v", err)
	}
	if _, err = ParseID("REQ-1.10", p, m); !errors.As(err, &idErr) {
		t.Fatalf("la consulta es exacta (REQ-1.10 no es REQ-1.1): %v", err)
	}
}

func resolveWithMap(t *testing.T, raw string, traces []traceSet) (equivalence, []Warning) {
	t.Helper()
	p, m := rfPattern(t), loadLegacy(t)
	q, err := ParseID(raw, p, m)
	if err != nil {
		t.Fatal(err)
	}
	return resolveEquivalentsLegacy(q, p, traces, m, m)
}

func TestRequirementTraceLegacyClosure(t *testing.T) {
	eq, warns := resolveWithMap(t, "REQ-1.1", nil)
	if want := []string{"REQ-1.1", "RF-ASK-01", "RF-WTP-01"}; !reflect.DeepEqual(eq.resolved, want) {
		t.Fatalf("antiguo ambiguo resuelve %v; quería %v", eq.resolved, want)
	}
	if lw := legacyWarnings(warns); len(lw) != 1 || !strings.Contains(lw[0].Message, "RF-ASK-01") || !strings.Contains(lw[0].Message, "RF-WTP-01") {
		t.Fatalf("legacy_ambiguous = %+v", warns)
	}

	eq, warns = resolveWithMap(t, "REQ-24.3", nil)
	if want := []string{"REQ-24.3", "RF-PAG-03"}; !reflect.DeepEqual(eq.resolved, want) || len(warns) != 0 {
		t.Fatalf("antiguo único resuelve %v con avisos %v", eq.resolved, warns)
	}

	// Un ID nuevo resuelve a su antiguo sin arrastrar los demás nuevos de ese antiguo.
	eq, warns = resolveWithMap(t, "RF-ASK-01", nil)
	if want := []string{"REQ-1.1", "RF-ASK-01"}; !reflect.DeepEqual(eq.resolved, want) || len(warns) != 0 {
		t.Fatalf("nuevo resuelve %v con avisos %v; quería %v", eq.resolved, warns, want)
	}

	// Dos antiguos de un mismo nuevo son equivalentes entre sí.
	eq, _ = resolveWithMap(t, "REQ-10.1", nil)
	if want := []string{"REQ-10.1", "REQ-10.2", "RF-PER-08"}; !reflect.DeepEqual(eq.resolved, want) {
		t.Fatalf("dos antiguos = %v; quería %v", eq.resolved, want)
	}
}

func TestRequirementTraceLegacyIDsInTraceEntries(t *testing.T) {
	traces := []traceSet{{Path: "a", Entries: []livespec.TraceEntry{
		{Provisional: "prov:x:1", Definitive: "REQ-24.3", Capability: "pagos", Requirement: "R"},
	}}}
	eq, _ := resolveWithMap(t, "RF-PAG-03", traces)
	if want := []string{"REQ-24.3", "RF-PAG-03", "prov:x:1"}; !reflect.DeepEqual(eq.resolved, want) {
		t.Fatalf("entrada con definitivo heredado: %v; quería %v", eq.resolved, want)
	}
}

// sddLegacy compone lectura y clasificación SDD con el mapa, como hará Collect.
func sddLegacy(t *testing.T, files fs.FS, raw string) ([]candidate, []Warning) {
	t.Helper()
	p, m := rfPattern(t), loadLegacy(t)
	q, err := ParseID(raw, p, m)
	if err != nil {
		t.Fatal(err)
	}
	src := Source{Files: files, GitPrefix: "openspec"}
	folders, warnings := listSDDFolders(src)
	traces, w := readSDDTraces(src, folders)
	warnings = append(warnings, w...)
	eq, w := resolveEquivalentsLegacy(q, p, traces, m, m)
	warnings = append(warnings, w...)
	var out []candidate
	for _, f := range folders {
		c, w := classifySDD(src, p, eq, f)
		out = append(out, c...)
		warnings = append(warnings, w...)
	}
	return out, warnings
}

func paths(cs []candidate) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func TestRequirementTraceLegacyMatchScopedByCapability(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/specs/autoskills/spec.md":         deltaSpec("MODIFIED", "Cobro (REQ-1.1)"),
		"changes/archive/2026-01-02-b/specs/workspace-topology/spec.md": deltaSpec("MODIFIED", "Otro (REQ-1.1)"),
	}
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"RF-ASK-01", []string{"a"}},
		{"RF-WTP-01", []string{"b"}},
		{"REQ-1.1", []string{"a", "b"}},
	} {
		got, warns := sddLegacy(t, files, tc.query)
		if !reflect.DeepEqual(paths(got), tc.want) {
			t.Errorf("consulta %s: incrementos %v; quería %v", tc.query, paths(got), tc.want)
		}
		for _, c := range got {
			if c.Role != RoleModified || !reflect.DeepEqual(c.IDs, []string{"REQ-1.1"}) {
				t.Errorf("consulta %s: %+v", tc.query, c.Increment)
			}
		}
		if tc.query != "REQ-1.1" && len(legacyWarnings(warns)) != 0 {
			t.Errorf("consulta %s: el contexto de capacidad no debe avisar: %v", tc.query, warns)
		}
	}
}

func TestRequirementTraceLegacyNewIDBringsOldIDIncrements(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/specs/pagos/spec.md": deltaSpec("MODIFIED", "Cobro (REQ-24.3)"),
		"changes/archive/2026-01-02-b/specs/pagos/spec.md": deltaSpec("ADDED", "Cobro nuevo (RF-PAG-03)"),
	}
	got, _ := sddLegacy(t, files, "RF-PAG-03")
	if len(got) != 2 || got[0].Role != RoleModified && got[1].Role != RoleModified {
		t.Fatalf("incrementos = %v", got)
	}
	byRole := map[Role]string{}
	for _, c := range got {
		byRole[c.Role] = c.Name
	}
	if byRole[RoleModified] != "a" || byRole[RoleCreated] != "b" {
		t.Fatalf("roles = %v", byRole)
	}
}

func TestRequirementTraceLegacyCubreWithAndWithoutContext(t *testing.T) {
	files := fstest.MapFS{
		// Con contexto: el cambio tiene specs/autoskills.
		"changes/archive/2026-01-01-con/specs/autoskills/spec.md": deltaSpec("ADDED", "Algo (RF-ASK-02)"),
		"changes/archive/2026-01-01-con/tasks.md":                 {Data: []byte("- [x] 1.1 x\n  Cubre: REQ-1.1\n")},
		// Sin contexto: ningún specs/.
		"changes/archive/2026-01-02-sin/tasks.md": {Data: []byte("- [x] 1.1 x\n  Cubre: REQ-1.1\n- [x] 1.2 y\n  Cubre: REQ-24.3\n")},
	}
	got, warns := sddLegacy(t, files, "RF-WTP-01")
	if len(got) != 0 {
		t.Fatalf("RF-WTP-01 no debe casar ni con contexto de otra capacidad ni sin contexto ambiguo: %v", paths(got))
	}
	lw := legacyWarnings(warns)
	if len(lw) != 1 || lw[0].Path != "openspec/changes/archive/2026-01-02-sin" || !strings.Contains(lw[0].Message, "autoskills") || !strings.Contains(lw[0].Message, "workspace-topology") {
		t.Fatalf("sin contexto útil debe avisar con la ruta y las candidatas: %+v", warns)
	}

	got, _ = sddLegacy(t, files, "RF-ASK-01")
	if !reflect.DeepEqual(paths(got), []string{"con"}) || got[0].Role != RoleCovered {
		t.Fatalf("con contexto de capacidad = %v", paths(got))
	}

	// Sin contexto, un antiguo de una sola capacidad casa con su ID nuevo.
	got, warns = sddLegacy(t, files, "RF-PAG-03")
	if !reflect.DeepEqual(paths(got), []string{"sin"}) || len(legacyWarnings(warns)) != 0 {
		t.Fatalf("antiguo de una capacidad sin contexto = %v, %v", paths(got), warns)
	}

	// Sin contexto, la consulta literal del ID antiguo casa.
	got, _ = sddLegacy(t, files, "REQ-1.1")
	if !reflect.DeepEqual(paths(got), []string{"con", "sin"}) {
		t.Fatalf("consulta literal del antiguo = %v", paths(got))
	}
}

// Hueco 7: REQ-27.2b es un token completo, no REQ-27.2 seguido de «b».
func TestRequirementTraceLegacySuffixedTokenIsWholeToken(t *testing.T) {
	files := fstest.MapFS{
		"changes/archive/2026-01-01-a/specs/sdd-kickoff/spec.md": deltaSpec("MODIFIED", "Arranque (REQ-27.2b)"),
	}
	got, _ := sddLegacy(t, files, "REQ-27.2b")
	if !reflect.DeepEqual(paths(got), []string{"a"}) || !reflect.DeepEqual(got[0].IDs, []string{"REQ-27.2b"}) {
		t.Fatalf("REQ-27.2b se consulta y se encuentra como token completo: %v", got)
	}
	got, _ = sddLegacy(t, files, "RF-KCK-09")
	if !reflect.DeepEqual(paths(got), []string{"a"}) {
		t.Fatalf("el ID nuevo trae el incremento de REQ-27.2b: %v", paths(got))
	}
	got, _ = sddLegacy(t, files, "REQ-27.2")
	if len(got) != 0 {
		t.Fatalf("REQ-27.2 no debe casar con REQ-27.2b: %v", paths(got))
	}
}

func oddLegacy(t *testing.T, files fs.FS, raw string) ([]candidate, []Warning) {
	t.Helper()
	p, m := rfPattern(t), loadLegacy(t)
	q, err := ParseID(raw, p, m)
	if err != nil {
		t.Fatal(err)
	}
	src := Source{Files: files, GitPrefix: "odd/tasks"}
	docs, warnings := listODDDocs(src)
	traces, w := readODDTraces(docs)
	warnings = append(warnings, w...)
	eq, w := resolveEquivalentsLegacy(q, p, traces, m, m)
	warnings = append(warnings, w...)
	var out []candidate
	for _, d := range docs {
		c, w := classifyODD(p, eq, d)
		out = append(out, c...)
		warnings = append(warnings, w...)
	}
	return out, warnings
}

func TestRequirementTraceLegacyODDCapabilityContext(t *testing.T) {
	doc := "## Spec viva\n\n### Capacidad: pagos\n\n```openspec\n## MODIFIED Requirements\n\n### Requirement: Cobro (REQ-24.3)\n\nEl sistema MUST.\n\n#### Scenario: x\n\n- **DADO** a\n```\n"
	files := fstest.MapFS{"cobro.md": oddDocText("", doc)}
	got, _ := oddLegacy(t, files, "RF-PAG-03")
	if len(got) != 1 || got[0].Role != RoleModified {
		t.Fatalf("ODD con ### Capacidad: pagos = %v", got)
	}

	// Capacidad del documento fuera del mapa para ese token: sin contexto útil y ambiguo.
	files = fstest.MapFS{"viejo.md": oddDocText("", oddSpecViva)}
	got, warns := oddLegacy(t, files, "RF-ASK-01")
	if len(got) != 0 || len(legacyWarnings(warns)) != 1 {
		t.Fatalf("sin contexto útil: %v, %v", got, warns)
	}
	got, _ = oddLegacy(t, files, "REQ-1.1")
	if len(got) != 1 {
		t.Fatalf("consulta literal del antiguo en ODD = %v", got)
	}
}
