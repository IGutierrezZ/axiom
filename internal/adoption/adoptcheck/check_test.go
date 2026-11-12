package adoptcheck

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const (
	checkChange = "adopcion-requisitos"

	validDelta = `# Delta para proyecto

## ADDED Requirements

### Requirement: Alta de usuario (REQ-40.1)

El sistema DEBE permitir el alta de un usuario.

Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)

#### Scenario: Alta correcta
- **DADO** un usuario nuevo
- **CUANDO** se da de alta
- **ENTONCES** queda registrado

### Requirement: Baja de usuario (REQ-40.2)

El sistema DEBE permitir la baja de un usuario.

Origen: docs/requisitos.md#rf-002-baja-de-usuario (RF-002)

#### Scenario: Baja correcta
- **DADO** un usuario existente
- **CUANDO** se da de baja
- **ENTONCES** deja de existir
`
	validDiscards = "# Adopción\n\n## Descartado\n\n- docs/requisitos.md#requisitos: es el título del documento\n"
)

type checkFixture struct {
	loc        workspace.Location
	changeRoot string
}

// newCheckFixture crea un workspace con docs/requisitos.md, su inventario y un cambio de
// adopción válido (delta y adoption.md).
func newCheckFixture(t *testing.T) *checkFixture {
	t.Helper()
	root := t.TempDir()
	writeTree(t, root, map[string]string{"docs/requisitos.md": requisitosDoc})
	f := &checkFixture{
		loc:        workspace.Location{WorkspaceRoot: root, PlanningHome: filepath.Join(root, "openspec")},
		changeRoot: filepath.Join(root, "openspec", "changes", checkChange),
	}
	f.writeInventory(t)
	f.write(t, "specs/proyecto/spec.md", validDelta)
	f.write(t, adoption.DiscardsFileName, validDiscards)
	return f
}

func (f *checkFixture) write(t *testing.T, rel, content string) {
	t.Helper()
	writeTree(t, f.changeRoot, map[string]string{rel: content})
}

func (f *checkFixture) writeInventory(t *testing.T) {
	t.Helper()
	inv := mustScan(t, f.loc.WorkspaceRoot)
	data, err := inv.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	f.write(t, adoption.InventoryFileName, string(data))
}

func (f *checkFixture) check(t *testing.T) CheckReport {
	t.Helper()
	report, err := Check(f.loc, checkChange)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return report
}

// failuresOf devuelve los fallos de una regla.
func failuresOf(report CheckReport, rule string) []Failure {
	var out []Failure
	for _, f := range report.Failures {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func requireFailure(t *testing.T, report CheckReport, rule string, contains ...string) {
	t.Helper()
	for _, f := range failuresOf(report, rule) {
		text := f.Subject + " " + f.Detail
		ok := true
		for _, want := range contains {
			if !strings.Contains(text, want) {
				ok = false
			}
		}
		if ok {
			return
		}
	}
	t.Fatalf("falta un fallo %q que contenga %q; fallos: %+v", rule, contains, report.Failures)
}

func TestCheck_UnCambioValidoPasa(t *testing.T) {
	f := newCheckFixture(t)
	report := f.check(t)
	if !report.Passed || len(report.Failures) != 0 || report.Change != checkChange {
		t.Fatalf("informe = %+v", report)
	}
}

func TestCheck_UnModifiedSeRechaza(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", validDelta+"\n## MODIFIED Requirements\n\n### Requirement: Otro (REQ-40.3)\n\nTexto.\n\nOrigen: docs/requisitos.md#requisitos (RF-1)\n\n#### Scenario: x\n- **DADO** a\n")
	report := f.check(t)
	if report.Passed {
		t.Fatal("un MODIFIED no debe pasar")
	}
	requireFailure(t, report, RuleFormat, "MODIFIED Requirements", "ADDED")
}

func TestCheck_EncabezadoDeNivel2DesconocidoSeRechaza(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "## ADDED Requirements", "## ADDED Requirement", 1))
	report := f.check(t)
	requireFailure(t, report, RuleFormat, "## ADDED Requirement")
}

func TestCheck_RequisitoSinIdentificadorEnElTitulo(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "Alta de usuario (REQ-40.1)", "Alta de usuario", 1))
	report := f.check(t)
	requireFailure(t, report, RuleFormat, "Alta de usuario", "(REQ-n.m)")
}

func TestCheck_CambioSinDelta(t *testing.T) {
	f := newCheckFixture(t)
	if err := os.RemoveAll(filepath.Join(f.changeRoot, "specs")); err != nil {
		t.Fatal(err)
	}
	report := f.check(t)
	requireFailure(t, report, RuleFormat, "specs/<capacidad>/spec.md")
}

func TestCheck_IdentificadorDuplicadoEnElCambio(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "(REQ-40.2)", "(REQ-40.1)", 1))
	report := f.check(t)
	requireFailure(t, report, RuleUniqueness, "REQ-40.1", "Baja de usuario")
}

func TestCheck_IdentificadorQueYaEstaEnLasSpecs(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"specs/viejo/spec.md": "# Especificación: viejo\n\n## Requirements\n\n### Requirement: Algo (REQ-40.1)\n\nTexto.\n",
	})
	report := f.check(t)
	requireFailure(t, report, RuleUniqueness, "REQ-40.1", "la capacidad viejo")
}

func TestCheck_ColisionConOtroCambioActivo(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"changes/otro/specs/cap/spec.md": "## ADDED Requirements\n\n### Requirement: Algo (REQ-40.2)\n\nTexto.\n",
	})
	report := f.check(t)
	requireFailure(t, report, RuleUniqueness, "REQ-40.2", "el cambio activo otro")
}

func TestCheck_MezclaDeSeries(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "(REQ-40.2)", "(REQ-41.1)", 1))
	report := f.check(t)
	requireFailure(t, report, RuleSeries, "40", "41")
}

func TestCheck_SerieYaExistenteEnLasSpecs(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"specs/vieja/spec.md": "# Especificación: vieja\n\n## Requirements\n\n### Requirement: Algo (REQ-40.77)\n\nTexto.\n",
	})
	report := f.check(t)
	requireFailure(t, report, RuleSeries, "serie 40", "la capacidad vieja")
}

func TestCheck_UnaSolaSerieNuevaNoFalla(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"specs/otra/spec.md": "# Especificación: otra\n\n## Requirements\n\n### Requirement: Algo (REQ-28.1)\n\nTexto.\n",
	})
	if got := failuresOf(f.check(t), RuleSeries); len(got) != 0 {
		t.Fatalf("no debería haber fallos de serie: %+v", got)
	}
}

func TestCheck_RequisitoSinEscenario(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "#### Scenario: Baja correcta", "Nota sin escenario", 1))
	report := f.check(t)
	requireFailure(t, report, RuleScenarios, "Baja de usuario")
	if got := failuresOf(report, RuleScenarios); len(got) != 1 {
		t.Fatalf("solo el segundo requisito carece de escenario: %+v", got)
	}
}

func TestCheck_OrigenQueNoResuelve(t *testing.T) {
	f := newCheckFixture(t)
	delta := strings.Replace(validDelta, "docs/requisitos.md#rf-001-alta-de-usuario (RF-001)", "docs/no-existe.md#rf-001 (RF-001)", 1)
	delta = strings.Replace(delta, "docs/requisitos.md#rf-002-baja-de-usuario (RF-002)", "docs/requisitos.md#ancla-inexistente (RF-002)", 1)
	f.write(t, "specs/proyecto/spec.md", delta)
	report := f.check(t)
	requireFailure(t, report, RuleOrigin, "docs/no-existe.md", "no existe")
	requireFailure(t, report, RuleOrigin, "docs/requisitos.md#ancla-inexistente", "ancla")
}

func TestCheck_OrigenFueraDelWorkspace(t *testing.T) {
	f := newCheckFixture(t)
	delta := strings.Replace(validDelta, "docs/requisitos.md#rf-001-alta-de-usuario (RF-001)", "../fuera.md#x (RF-001)", 1)
	f.write(t, "specs/proyecto/spec.md", delta)
	requireFailure(t, f.check(t), RuleOrigin, "../fuera.md", "fuera")
}

func TestCheck_RequisitoSinOrigen(t *testing.T) {
	f := newCheckFixture(t)
	delta := strings.Replace(validDelta, "Origen: docs/requisitos.md#rf-002-baja-de-usuario (RF-002)\n", "", 1)
	f.write(t, "specs/proyecto/spec.md", delta)
	report := f.check(t)
	requireFailure(t, report, RuleOrigin, "Baja de usuario", "Origen:")
	requireFailure(t, report, RuleCoverage, "rf-002-baja-de-usuario")
}

func TestCheck_OrigenesMultiplesCubrenVariasUnidades(t *testing.T) {
	f := newCheckFixture(t)
	delta := strings.Replace(validDelta, "Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)\n",
		"Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)\nOrigen: docs/requisitos.md#requisitos\n", 1)
	f.write(t, "specs/proyecto/spec.md", delta)
	f.write(t, adoption.DiscardsFileName, "## Descartado\n")
	if report := f.check(t); !report.Passed {
		t.Fatalf("dos Origen: cubren dos unidades: %+v", report.Failures)
	}
}

func TestCheck_UnidadSinCubrirNiDescartar(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, adoption.DiscardsFileName, "## Descartado\n")
	report := f.check(t)
	requireFailure(t, report, RuleCoverage, "docs/requisitos.md#requisitos")
	if got := failuresOf(report, RuleCoverage); len(got) != 1 {
		t.Fatalf("solo falta la unidad `requisitos`: %+v", got)
	}
}

func TestCheck_DescartadoSinMotivo(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, adoption.DiscardsFileName, "## Descartado\n\n- docs/requisitos.md#requisitos:\n")
	report := f.check(t)
	requireFailure(t, report, RuleDiscardReason, "docs/requisitos.md#requisitos", "motivo")
	requireFailure(t, report, RuleCoverage, "docs/requisitos.md#requisitos")
}

func TestCheck_SeReportanTodosLosFallos(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "#### Scenario: Alta correcta", "Sin escenario", 1))
	f.write(t, adoption.DiscardsFileName, "## Descartado\n")
	report := f.check(t)
	requireFailure(t, report, RuleScenarios, "Alta de usuario")
	requireFailure(t, report, RuleCoverage, "docs/requisitos.md#requisitos")
	if report.Passed || len(report.Failures) != 2 {
		t.Fatalf("informe = %+v", report)
	}
}

func TestCheck_SinInventario(t *testing.T) {
	f := newCheckFixture(t)
	if err := os.Remove(filepath.Join(f.changeRoot, adoption.InventoryFileName)); err != nil {
		t.Fatal(err)
	}
	report := f.check(t)
	requireFailure(t, report, RuleInventoryMissing, "axiom archive adopt scan --change "+checkChange)
	if got := failuresOf(report, RuleCoverage); len(got) != 0 {
		t.Fatalf("sin inventario no hay cobertura que evaluar: %+v", got)
	}
}

func TestCheck_InventarioObsoleto(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.WorkspaceRoot, map[string]string{"docs/requisitos.md": requisitosDoc + "\n## RF-003 Nuevo\n"})
	report := f.check(t)
	requireFailure(t, report, RuleInventoryStale, "docs/requisitos.md", "axiom archive adopt scan --change "+checkChange)
}

func TestCheck_InventarioObsoletoPorFicheroBorrado(t *testing.T) {
	f := newCheckFixture(t)
	if err := os.Remove(filepath.Join(f.loc.WorkspaceRoot, "docs", "requisitos.md")); err != nil {
		t.Fatal(err)
	}
	requireFailure(t, f.check(t), RuleInventoryStale, "docs/requisitos.md", "no existe")
}

func TestCheck_NoEscribeNada(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, adoption.DiscardsFileName, "## Descartado\n")
	before := snapshotTree(t, f.loc.WorkspaceRoot)
	f.check(t)
	if after := snapshotTree(t, f.loc.WorkspaceRoot); !reflect.DeepEqual(before, after) {
		t.Fatal("Check escribió en el árbol (REQ-30.13: no registra compuertas ni toca nada)")
	}
}

func TestCheck_CambioInexistente(t *testing.T) {
	f := newCheckFixture(t)
	if _, err := Check(f.loc, "no-existe"); err == nil || !strings.Contains(err.Error(), "no-existe") {
		t.Fatalf("esperaba un error que nombre el cambio, got %v", err)
	}
}

func TestCheck_ElOrigenSeConservaTalCualEnLaDelta(t *testing.T) {
	// REQ-30.13: la línea `Origen:` forma parte del cuerpo del requisito y la composición
	// del archivado no la trata de forma especial: la spec viva la conserva byte a byte y
	// el identificador original solo consta en esa línea.
	canonical := "# Especificación: proyecto\n\n## Requirements\n\n### Requirement: Base (REQ-40.9)\n\nTexto.\n"
	composed, err := openspecdoc.Compose(canonical, validDelta, openspecdoc.ComposeOptions{})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	for _, line := range []string{
		"Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)",
		"Origen: docs/requisitos.md#rf-002-baja-de-usuario (RF-002)",
	} {
		if !strings.Contains(composed, "\n"+line+"\n") {
			t.Errorf("la spec viva compuesta no conserva %q:\n%s", line, composed)
		}
	}
	for _, r := range openspecdoc.ParseCanonical(composed) {
		if strings.Contains(r.Name, "RF-") {
			t.Errorf("el identificador funcional no es la identidad del requisito: %q", r.Name)
		}
	}
}
