package livingdoc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

func patternIndexer(t *testing.T, template string) *Indexer {
	t.Helper()
	p, err := reqid.Parse(template)
	if err != nil {
		t.Fatal(err)
	}
	return NewIndexer().WithPattern(p)
}

func requirementIDs(entry *LivingSpecEntry) []string {
	ids := make([]string, 0, len(entry.Requirements))
	for _, r := range entry.Requirements {
		ids = append(ids, r.ID)
	}
	return ids
}

func parseIDs(t *testing.T, idx *Indexer, domain, content string) []string {
	t.Helper()
	entry, _ := idx.ParseSpecContent(content, domain, domain+"/spec.md", time.Time{})
	return requirementIDs(entry)
}

// REQ-44.6: un encabezado con identificador lo conserva; sin él recibe un sintético que
// nunca es válido para el patrón, y la lectura laxa no valida.
func TestIndexerKeepsHeadingIDsAndSynthesizesOutsideTheValidSpace(t *testing.T) {
	spec := "# Cobros\n\n" +
		"### Requirement: Con ID (REQ-30.1)\n\n#### Scenario: A\n\n" +
		"### Requirement: Sin ID\n\n#### Scenario: B\n\n" +
		"### Requirement: Mal formado (REQ-x)\n\n#### Scenario: C\n"
	got := parseIDs(t, NewIndexer(), "facturacion", spec)
	want := []string{"REQ-30.1", "REQ-facturacion-2", "REQ-x"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("defecto: IDs = %v, quería %v", got, want)
	}

	rf := "# Usuarios\n\n" +
		"### Requirement: Con ID (RF-USU-01)\n\n#### Scenario: A\n\n" +
		"### Requirement: Sin ID\n\n#### Scenario: B\n\n" +
		"### Requirement: Otro patrón (REQ-30.1)\n\n#### Scenario: C\n\n" +
		"### Requirement: Mal formado (RF-USU-1)\n\n#### Scenario: D\n"
	got = parseIDs(t, patternIndexer(t, "RF-<DOM>-<NN>"), "usuarios", rf)
	want = []string{"RF-USU-01", "REQ-usuarios-2", "REQ-usuarios-3", "RF-USU-1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("RF: IDs = %v, quería %v", got, want)
	}
	valid := patternIndexer(t, "RF-<DOM>-<NN>").pattern
	for _, id := range got[1:3] {
		if valid.Valid(id) {
			t.Errorf("el sintético %q no puede ser un identificador válido", id)
		}
	}
}

func TestIndexerSyntheticWithoutLowercaseDomainStaysOutsideTheValidSpace(t *testing.T) {
	idx := patternIndexer(t, "RF-<DOM>-<NN>")
	got := parseIDs(t, idx, "USU", "# U\n\n### Requirement: Sin ID\n\n#### Scenario: A\n")
	if len(got) != 1 || got[0] != "req-USU-1" || idx.pattern.Valid(got[0]) {
		t.Errorf("IDs = %v, se esperaba req-USU-1", got)
	}
}

// REQ-44.6: el sintético es estable entre ejecuciones.
func TestIndexerSyntheticIDsAreStableAcrossRuns(t *testing.T) {
	spec := "# X\n\n### Requirement: Uno\n\n### Requirement: Dos\n"
	first := parseIDs(t, NewIndexer(), "pagos", spec)
	second := parseIDs(t, NewIndexer(), "pagos", spec)
	if strings.Join(first, ",") != strings.Join(second, ",") || len(first) != 2 || first[0] != "REQ-pagos-1" || first[1] != "REQ-pagos-2" {
		t.Errorf("IDs = %v / %v", first, second)
	}
}

// WithPattern devuelve una copia: el indexador original no cambia.
func TestIndexerWithPatternCopies(t *testing.T) {
	base := NewIndexer()
	withRF := base.WithPattern(patternIndexer(t, "RF-<DOM>-<NN>").pattern)
	if withRF == base {
		t.Fatal("WithPattern debe devolver una copia")
	}
	spec := "# U\n\n### Requirement: Alta (RF-USU-01)\n"
	if got := parseIDs(t, base, "usuarios", spec); got[0] != "REQ-usuarios-1" {
		// Con el defecto, `(RF-USU-01)` no es un candidato: el requisito recibe sintético.
		t.Errorf("el indexador original conserva el defecto: %v", got)
	}
	if got := parseIDs(t, withRF, "usuarios", spec); got[0] != "RF-USU-01" {
		t.Errorf("la copia usa RF-<DOM>-<NN>: %v", got)
	}
}

func writeProject(t *testing.T, sddBlock, spec string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	config := "workspace:\n  name: demo\n  topology: monorepo-embedded\nroles:\n  fullstack:\n    name: Fullstack\n    repositories:\n      - path: .\n" + sddBlock
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), config)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "specs", "usuarios", "spec.md"), spec)
	return ws
}

// REQ-44.6: el servicio indexa con el patrón del workspace.
func TestServiceIndexesWithTheWorkspacePattern(t *testing.T) {
	spec := "# Usuarios\n\n### Requirement: Alta (RF-USU-01)\n\n#### Scenario: A\n- DADO x\n"
	ws := writeProject(t, "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n", spec)
	svc := NewService(ws.Master, nil, nil)
	catalog, err := svc.GetCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Specs) != 1 || catalog.Specs[0].Requirements[0].ID != "RF-USU-01" {
		t.Fatalf("catálogo = %+v", catalog.Specs)
	}
	if len(catalog.Warnings) != 0 {
		t.Errorf("sin errores de patrón no hay avisos: %v", catalog.Warnings)
	}
	report, err := svc.Sync(context.Background())
	if err != nil || len(report.Warnings) != 0 {
		t.Fatalf("Sync = %+v, %v", report, err)
	}
}

// REQ-44.6 y REQ-44.9: con el patrón inaccesible livingdoc, que es informativo, indexa con el
// defecto y avisa nombrando el error del patrón, sin fallar.
func TestServiceWithAnInaccessiblePatternIndexesWithTheDefaultAndWarns(t *testing.T) {
	spec := "# Usuarios\n\n### Requirement: Alta (REQ-30.1)\n\n#### Scenario: A\n- DADO x\n"
	ws := writeProject(t, "sdd:\n  id_pattern: \"<m><m>\"\n", spec)
	svc := NewService(ws.Master, nil, nil)

	catalog, err := svc.GetCatalog(context.Background())
	if err != nil {
		t.Fatalf("GetCatalog no debe fallar: %v", err)
	}
	if got := catalog.Specs[0].Requirements[0].ID; got != "REQ-30.1" {
		t.Errorf("con el defecto el ID es REQ-30.1: %q", got)
	}
	if joined := strings.Join(catalog.Warnings, "\n"); !strings.Contains(joined, "sdd.id_pattern") || !strings.Contains(joined, `"<m><m>"`) {
		t.Errorf("el aviso debe nombrar el error del patrón: %v", catalog.Warnings)
	}

	report, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync no debe fallar: %v", err)
	}
	if joined := strings.Join(report.Warnings, "\n"); !strings.Contains(joined, "sdd.id_pattern") {
		t.Errorf("SyncReport.Warnings debe nombrar el error del patrón: %v", report.Warnings)
	}
}
