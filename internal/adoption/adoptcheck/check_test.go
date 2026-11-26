package adoptcheck

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const (
	checkChange = "adopcion-requisitos"

	validDelta = `# Delta para proyecto

## ADDED Requirements

### Requirement: Alta de usuario (RF-001)

El sistema DEBE permitir el alta de un usuario.

Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)

#### Scenario: Alta correcta
- **DADO** un usuario nuevo
- **CUANDO** se da de alta
- **ENTONCES** queda registrado

### Requirement: Baja de usuario (RF-002)

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
	pattern    reqid.Pattern
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
	withDeclaredPattern(t, f, "RF-<NNN>")
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
	report, err := Check(f.loc, checkChange, f.pattern)
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
	f.write(t, "specs/proyecto/spec.md", validDelta+"\n## MODIFIED Requirements\n\n### Requirement: Otro (RF-003)\n\nTexto.\n\nOrigen: docs/requisitos.md#requisitos (RF-1)\n\n#### Scenario: x\n- **DADO** a\n")
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
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "Alta de usuario (RF-001)", "Alta de usuario", 1))
	report := f.check(t)
	requireFailure(t, report, RuleFormat, "Alta de usuario", "(RF-NNN)")
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
	f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "(RF-002)", "(RF-001)", 1))
	report := f.check(t)
	requireFailure(t, report, RuleUniqueness, "RF-001", "Baja de usuario")
}

func TestCheck_IdentificadorQueYaEstaEnLasSpecs(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"specs/viejo/spec.md": "# Especificación: viejo\n\n## Requirements\n\n### Requirement: Algo (RF-001)\n\nTexto.\n",
	})
	report := f.check(t)
	requireFailure(t, report, RuleUniqueness, "RF-001", "la capacidad viejo")
}

func TestCheck_ColisionConOtroCambioActivo(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"changes/otro/specs/cap/spec.md": "## ADDED Requirements\n\n### Requirement: Algo (RF-002)\n\nTexto.\n",
	})
	report := f.check(t)
	requireFailure(t, report, RuleUniqueness, "RF-002", "el cambio activo otro")
}

// RF-ADP-04 (C-03-c): ya no hay regla de serie. Un cambio puede mezclar series y usar una
// serie que ya existe en las specs; solo cuentan el formato y la unicidad por serie y número.
func TestCheck_YaNoHayReglaDeSerie(t *testing.T) {
	f := newCheckFixture(t)
	withDeclaredPattern(t, f, "RF-<DOM>-<NN>")
	f.write(t, "specs/proyecto/spec.md", strings.ReplaceAll(strings.ReplaceAll(validDelta, "(RF-001)", "(RF-USU-01)"), "(RF-002)", "(RF-FAC-01)"))
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"specs/vieja/spec.md": "# Especificación: vieja\n\n## Requirements\n\n### Requirement: Algo (RF-USU-77)\n\nTexto.\n",
	})
	report := f.check(t)
	for _, f := range report.Failures {
		if f.Rule == RuleFormat || f.Rule == RuleUniqueness || f.Rule == "series" {
			t.Fatalf("mezclar series o reutilizar una existente no es un fallo: %+v", report.Failures)
		}
	}
}

func TestCheck_UnaSolaSerieNuevaNoFalla(t *testing.T) {
	f := newCheckFixture(t)
	writeTree(t, f.loc.PlanningHome, map[string]string{
		"specs/otra/spec.md": "# Especificación: otra\n\n## Requirements\n\n### Requirement: Algo (RF-028)\n\nTexto.\n",
	})
	if got := failuresOf(f.check(t), RuleUniqueness); len(got) != 0 {
		t.Fatalf("no debería haber fallos de unicidad: %+v", got)
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
	if _, err := Check(f.loc, "no-existe", f.pattern); err == nil || !strings.Contains(err.Error(), "no-existe") {
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
		if !strings.Contains(r.Name, "(RF-00") && !strings.Contains(r.Name, "(REQ-40.9)") {
			t.Errorf("la identidad conservada del requisito es el ID del cliente: %q", r.Name)
		}
	}
}

// RF-ADP-01: `proposed_id_pattern` no altera la obsolescencia del inventario (solo cuenta
// el SHA-256 de cada origen) ni el resumen, y editar axiom.yaml tampoco la altera.
func TestCheck_ElPatronPropuestoNoAlteraLaObsolescencia(t *testing.T) {
	f := newCheckFixture(t)
	inv := mustScan(t, f.loc.WorkspaceRoot)
	if inv.ProposedIDPattern == nil {
		t.Fatal("el inventario del fixture debería llevar la propuesta")
	}
	withField := f.check(t)
	sumWith := adoption.Summarize(inv)

	stripped := inv
	stripped.ProposedIDPattern = nil
	data, err := stripped.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	f.write(t, adoption.InventoryFileName, string(data))
	writeTree(t, f.loc.WorkspaceRoot, map[string]string{"axiom.yaml": "sdd:\n  id_pattern: \"RF-<NNN>\"\n"})
	withoutField := f.check(t)

	if !reflect.DeepEqual(withField, withoutField) {
		t.Fatalf("el campo no debe alterar el resultado:\ncon: %+v\nsin: %+v", withField, withoutField)
	}
	if got := failuresOf(withoutField, RuleInventoryStale); len(got) != 0 {
		t.Fatalf("no debería haber obsolescencia: %+v", got)
	}
	if !reflect.DeepEqual(sumWith, adoption.Summarize(stripped)) {
		t.Fatal("Summarize no debe depender del campo")
	}
}

// RF-ADP-02 y RF-ADP-03: formato del título (D4): definitivo, provisional compatible o rechazo.
func TestCheck_FormatoDelTitulo(t *testing.T) {
	retitle := func(from, to string) string { return strings.Replace(validDelta, "Alta de usuario (RF-001)", to, 1) }
	cases := []struct {
		name     string
		pattern  string
		title    string
		wantFail []string // subcadenas del fallo `format`; vacío = sin fallo
	}{
		{"definitivo válido", "RF-<NNN>", "Alta de usuario (RF-001)", nil},
		{"provisional compatible con serie fija", "RF-<NNN>", "Alta de usuario (prov::1)", nil},
		{"provisional con dominio", "RF-<DOM>-<NN>", "Alta de usuario (prov:USU:1)", nil},
		{"provisional por defecto con serie", "", "Alta de usuario (prov:40:1)", nil},
		{"provisional por defecto con etiqueta", "", "Alta de usuario (prov:nueva:1)", nil},
		{"provisional incompatible", "RF-<NNN>", "Alta de usuario (prov:40:1)", []string{"Alta de usuario", "provisional_malformed"}},
		{"etiqueta sin serie nueva", "RF-<NNN>", "Alta de usuario (prov:nueva:1)", []string{"Alta de usuario", reqid.CodeProvisionalNewSeriesUnsupported}},
		{"sin identificador", "RF-<NNN>", "Alta de usuario (algo)", []string{"Alta de usuario", "(algo)", "RF-NNN", reqid.ProvisionalText("<clave>", "<ordinal>")}},
		{"definitivo y provisional", "RF-<NNN>", "Alta de usuario (RF-009) (prov::1)", []string{"Alta de usuario", "ambiguo"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCheckFixture(t)
			if tc.pattern == "" {
				withoutDeclaredPattern(f)
			} else {
				withDeclaredPattern(t, f, tc.pattern)
			}
			f.write(t, "specs/proyecto/spec.md", retitle("", tc.title))
			// Solo se evalúa el primer requisito: el segundo conserva su (RF-002) de fixture.
			report := f.check(t)
			var got []Failure
			for _, fl := range failuresOf(report, RuleFormat) {
				if strings.HasPrefix(fl.Subject, "Alta de usuario") {
					got = append(got, fl)
				}
			}
			if len(tc.wantFail) == 0 {
				if len(got) != 0 {
					t.Fatalf("no debería haber fallos de formato: %+v", got)
				}
				return
			}
			requireFailure(t, report, RuleFormat, tc.wantFail...)
		})
	}
}

// RF-ADP-03 y RF-ADP-04: unicidad por serie y número (D4).
func TestCheck_UnicidadPorClave(t *testing.T) {
	t.Run("misma clave con distinto ancho", func(t *testing.T) {
		f := newCheckFixture(t)
		withDeclaredPattern(t, f, "RF-<m>")
		f.write(t, "specs/proyecto/spec.md", strings.Replace(strings.Replace(validDelta, "(RF-001)", "(RF-1)", 1), "(RF-002)", "(RF-001)", 1))
		requireFailure(t, f.check(t), RuleUniqueness, "Alta de usuario", "Baja de usuario", "serie `RF-*`, número 1")
	})
	t.Run("mismo ID conservado en dos requisitos", func(t *testing.T) {
		f := newCheckFixture(t)
		f.write(t, "specs/proyecto/spec.md", strings.Replace(validDelta, "(RF-002)", "(RF-001)", 1))
		requireFailure(t, f.check(t), RuleUniqueness, "RF-001", "Baja de usuario", "provisional")
	})
	t.Run("mismo provisional", func(t *testing.T) {
		f := newCheckFixture(t)
		f.write(t, "specs/proyecto/spec.md", strings.Replace(strings.Replace(validDelta, "(RF-001)", "(prov::1)", 1), "(RF-002)", "(prov::1)", 1))
		requireFailure(t, f.check(t), RuleUniqueness, "prov::1", "Baja de usuario")
	})
	t.Run("contra el registro", func(t *testing.T) {
		f := newCheckFixture(t)
		writeTree(t, f.loc.PlanningHome, map[string]string{
			"specs/viva/spec.md": "# Especificación: viva\n\n## Requirements\n\n### Requirement: Algo (RF-001)\n\nTexto.\n",
		})
		report := f.check(t)
		requireFailure(t, report, RuleUniqueness, "RF-001", "la capacidad viva")
		if got := failuresOf(report, RuleUniqueness); len(got) != 1 {
			t.Fatalf("solo choca RF-001; RF-002 está libre: %+v", got)
		}
	})
	t.Run("con RF-<m> el registro encuentra RF-1", func(t *testing.T) {
		f := newCheckFixture(t)
		withDeclaredPattern(t, f, "RF-<m>")
		writeTree(t, f.loc.PlanningHome, map[string]string{
			"specs/viva/spec.md": "# Especificación: viva\n\n## Requirements\n\n### Requirement: Algo (RF-1)\n\nTexto.\n",
		})
		requireFailure(t, f.check(t), RuleUniqueness, "la capacidad viva")
	})
}
