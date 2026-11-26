package sddstatus

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	reqDeltaPagos = "## ADDED Requirements\n\n### Requirement: Cobro con IVA (prov:30:1)\n\nCobra con IVA.\n\n#### Scenario: Cobro\n\n- **CUANDO** se cobra\n- **ENTONCES** incluye IVA\n\n" +
		"### Requirement: Recargo (REQ-30.9)\n\nCobra recargo.\n\n#### Scenario: Recargo\n\n- **CUANDO** hay recargo\n- **ENTONCES** se aplica\n\n" +
		"## MODIFIED Requirements\n\n### Requirement: Recibo (REQ-5.1)\n\nEmite recibo.\n\n#### Scenario: Recibo\n\n- **CUANDO** se paga\n- **ENTONCES** se emite\n\n" +
		"### Requirement: Cobro antiguo\n\nTexto.\n\n#### Scenario: Antiguo\n\n- **CUANDO** x\n- **ENTONCES** y\n\n" +
		"## REMOVED Requirements\n\n### Requirement: Descuento (REQ-5.2)\n\n## RENAMED Requirements\n\n### Requirement: Cobro viejo (REQ-5.3) → Cobro nuevo (REQ-5.3)\n"
	reqDeltaAlfa = "## ADDED Requirements\n\n### Requirement: Alta (REQ-31.1)\n\nDa de alta.\n\n#### Scenario: Alta\n\n- **CUANDO** se da de alta\n- **ENTONCES** queda registrado\n"
	reqFullSpec  = "# nueva\n\n## Requirements\n\n### Requirement: Nuevo (prov:a:1)\n\nTexto.\n\n#### Scenario: S\n\n- **CUANDO** x\n- **ENTONCES** y\n"
)

func requirementsJSON(t *testing.T, status Status) string {
	t.Helper()
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// RF-CLI-14: modifies y creates ordenados, sin id para un nombre sin ID, provisional marcado,
// RENAMED por su nombre antiguo y capacidad nueva como spec completa que cuenta como ADDED.
func TestRequirementsProjectsModifiesAndCreates(t *testing.T) {
	t.Parallel()
	ws := isoDefaultProject(t)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": reqDeltaPagos, "alfa": reqDeltaAlfa, "nueva": reqFullSpec}, provTasks, "")
	status := resolveFrom(t, ws.Master, "demo")
	if status.Requirements == nil {
		t.Fatal("requirements ausente con deltas legibles")
	}
	type row struct {
		cap, id, op, name string
		prov              bool
	}
	flat := func(items []RequirementItem) []row {
		var out []row
		for _, i := range items {
			out = append(out, row{i.Capability, i.ID, i.Operation, i.Requirement, i.Provisional})
		}
		return out
	}
	wantCreates := []row{
		{"alfa", "REQ-31.1", "ADDED", "Alta (REQ-31.1)", false},
		{"nueva", "prov:a:1", "ADDED", "Nuevo (prov:a:1)", true},
		{"pagos", "prov:30:1", "ADDED", "Cobro con IVA (prov:30:1)", true},
		{"pagos", "REQ-30.9", "ADDED", "Recargo (REQ-30.9)", false},
	}
	if got := flat(status.Requirements.Creates); !reflect.DeepEqual(got, wantCreates) {
		t.Errorf("creates = %+v\nquería %+v", got, wantCreates)
	}
	wantModifies := []row{
		{"pagos", "REQ-5.1", "MODIFIED", "Recibo (REQ-5.1)", false},
		{"pagos", "", "MODIFIED", "Cobro antiguo", false},
		{"pagos", "REQ-5.2", "REMOVED", "Descuento (REQ-5.2)", false},
		{"pagos", "REQ-5.3", "RENAMED", "Cobro viejo (REQ-5.3)", false},
	}
	if got := flat(status.Requirements.Modifies); !reflect.DeepEqual(got, wantModifies) {
		t.Errorf("modifies = %+v\nquería %+v", got, wantModifies)
	}
	wire := requirementsJSON(t, status)
	if !strings.Contains(wire, `"requirements":{"modifies":[`) || !strings.Contains(wire, `"provisional":true`) {
		t.Errorf("JSON sin el campo esperado: %s", wire)
	}
	if strings.Contains(wire, `"id":""`) {
		t.Errorf("un nombre sin ID no lleva id: %s", wire)
	}
}

// RF-CLI-14: sin deltas legibles el campo se omite y la salida no lo nombra.
func TestRequirementsOmittedWithoutReadableDeltas(t *testing.T) {
	t.Parallel()
	ws := isoDefaultProject(t)
	seedISOChange(t, ws, "sin-specs", "# Propuesta\n", nil, provTasks, "")
	seedISOChange(t, ws, "vacio", "# Propuesta\n", map[string]string{"pagos": "# solo texto\n"}, provTasks, "")
	for _, change := range []string{"sin-specs", "vacio"} {
		status := resolveFrom(t, ws.Master, change)
		if status.Requirements != nil {
			t.Errorf("%s: requirements = %+v, debía omitirse", change, status.Requirements)
		}
		if strings.Contains(requirementsJSON(t, status), `"requirements"`) {
			t.Errorf("%s: el JSON no debe nombrar requirements", change)
		}
	}
}

// RF-CLI-14: con un cambio archivado el campo se omite.
func TestRequirementsOmittedForArchivedChange(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, filepath.Join(root, "openspec", "changes", "archive", "2026-08-30-demo", "proposal.md"), "# Propuesta\n")
	write(t, filepath.Join(root, "openspec", "changes", "archive", "2026-08-30-demo", "specs", "pagos", "spec.md"), reqDeltaPagos)
	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Archived == nil || status.Requirements != nil {
		t.Errorf("archived = %v, requirements = %+v", status.Archived, status.Requirements)
	}
}

// RF-CLI-14: con el patrón inaccesible el campo se omite.
func TestRequirementsOmittedWithInaccessiblePattern(t *testing.T) {
	t.Parallel()
	ws := isoDefaultProject(t)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": reqDeltaPagos}, provTasks, "")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: true\n  id_pattern: \"<m><m>\"\n")
	status := resolveFrom(t, ws.Master, "demo")
	if status.Requirements != nil {
		t.Errorf("requirements = %+v, debía omitirse con el patrón inaccesible", status.Requirements)
	}
}

// RF-CLI-14: con el almacén engram o none no hay ficheros de delta y el campo se omite; y el
// campo nunca altera nextRecommended, dependencies ni motivos.
func TestRequirementsStoreAndRoutingIndependence(t *testing.T) {
	t.Parallel()
	ws := isoDefaultProject(t)
	root := seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": reqDeltaPagos}, provTasks, "")
	status := resolveFrom(t, ws.Master, "demo")
	if status.Requirements == nil {
		t.Fatal("requirements ausente")
	}
	before := status
	before.Requirements = nil
	for _, store := range []ArtifactStore{ArtifactStoreEngram, ArtifactStoreNone} {
		copied := before
		copied.ArtifactStore = store
		decorateRequirements(&copied, workspace.Location{WorkspaceRoot: filepath.Join(root, "ausente")})
		if copied.Requirements != nil {
			t.Errorf("%s: requirements = %+v, debía omitirse", store, copied.Requirements)
		}
	}
	if status.NextRecommended != before.NextRecommended || !reflect.DeepEqual(status.BlockedReasons, before.BlockedReasons) || status.Dependencies != before.Dependencies {
		t.Errorf("el campo cambió el enrutado: %q %v", status.NextRecommended, status.BlockedReasons)
	}
}
