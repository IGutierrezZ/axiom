package adoptcheck

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
)

const (
	originRF001 = "Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)"
	originRF002 = "Origen: docs/requisitos.md#rf-002-baja-de-usuario (RF-002)"
)

// useDoc sustituye el documento de requisitos del fixture, regenera su inventario y deja
// como únicos descartes los indicados (`- <ruta>#<ancla>: <motivo>`).
func (f *checkFixture) useDoc(t *testing.T, doc string, discards ...string) {
	t.Helper()
	writeTree(t, f.loc.WorkspaceRoot, map[string]string{"docs/requisitos.md": doc})
	f.writeInventory(t)
	f.write(t, adoption.DiscardsFileName, "# Adopción\n\n## Descartado\n\n"+strings.Join(discards, "\n")+"\n")
}

func (f *checkFixture) delta(t *testing.T, text string) {
	t.Helper()
	f.write(t, "specs/proyecto/spec.md", text)
}

const docConGlosario = "# Requisitos\n\n## RF-001 Alta de usuario\n\nAlta.\n\n## RF-002 Baja de usuario\n\nBaja.\n\n## Glosario\n\nTérminos sin identificador.\n"

// deltaOf compone una delta ADDED con requisitos (título, líneas Origen:).
func deltaOf(reqs ...[2]string) string {
	var b strings.Builder
	b.WriteString("# Delta\n\n## ADDED Requirements\n\n")
	for _, r := range reqs {
		b.WriteString("### Requirement: " + r[0] + "\n\nEl sistema DEBE algo.\n\n" + r[1] + "\n\n#### Scenario: Ok\n- **DADO** x\n- **CUANDO** y\n- **ENTONCES** z\n\n")
	}
	return b.String()
}

// REQ-30.11 f: el `Origen:` debe coincidir con el ID propio de su unidad.
func TestCheck_OriginID_OrigenContraLaUnidad(t *testing.T) {
	f := newCheckFixture(t)
	f.delta(t, strings.Replace(validDelta, originRF001, "Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-002)", 1))
	requireFailure(t, f.check(t), RuleOriginID, "Alta de usuario", "RF-002", "RF-001", "docs/requisitos.md#rf-001-alta-de-usuario")
}

func TestCheck_OriginID_ProvisionalConIDEsperadoSeRechaza(t *testing.T) {
	f := newCheckFixture(t)
	f.delta(t, strings.Replace(validDelta, "Alta de usuario (RF-001)", "Alta de usuario (prov::1)", 1))
	requireFailure(t, f.check(t), RuleOriginID, "Alta de usuario", "conserva el identificador del cliente `RF-001`")
}

func TestCheck_OriginID_DefinitivoSinIDEnLaUnidad(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, docConGlosario, "- docs/requisitos.md#requisitos: es el título")
	f.delta(t, validDelta+deltaOf([2]string{"Glosario (RF-003)", "Origen: docs/requisitos.md#glosario"})[len("# Delta\n\n## ADDED Requirements\n\n"):])
	requireFailure(t, f.check(t), RuleOriginID, "Glosario", "no trae identificador", "usa un provisional")
}

func TestCheck_OriginID_ProvisionalConUnidadSinIDPasa(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, docConGlosario, "- docs/requisitos.md#requisitos: es el título")
	f.delta(t, validDelta+deltaOf([2]string{"Glosario (prov::1)", "Origen: docs/requisitos.md#glosario"})[len("# Delta\n\n## ADDED Requirements\n\n"):])
	report := f.check(t)
	if got := failuresOf(report, RuleOriginID); len(got) != 0 {
		t.Fatalf("un provisional sobre una unidad sin ID no falla: %+v", got)
	}
	if !report.Passed {
		t.Fatalf("el cambio debe pasar: %+v", report.Failures)
	}
}

func TestCheck_OriginID_DefinitivoQueNoCoincide(t *testing.T) {
	f := newCheckFixture(t)
	f.delta(t, strings.Replace(validDelta, "Alta de usuario (RF-001)", "Alta de usuario (RF-009)", 1))
	requireFailure(t, f.check(t), RuleOriginID, "Alta de usuario", "RF-009", "RF-001")
}

func TestCheck_OriginID_ConVariosOrigenesBastaUnoQueCoincida(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, docConGlosario, "- docs/requisitos.md#requisitos: es el título")
	f.delta(t, deltaOf(
		[2]string{"Alta de usuario (RF-001)", originRF001 + "\nOrigen: docs/requisitos.md#glosario (RF-002)"},
		[2]string{"Baja de usuario (RF-002)", originRF002},
	))
	if got := failuresOf(f.check(t), RuleOriginID); len(got) != 0 {
		t.Fatalf("basta que el título coincida con algún Origen: %+v", got)
	}
}

func TestCheck_OriginID_CompararPorClave(t *testing.T) {
	f := newCheckFixture(t)
	withDeclaredPattern(t, f, "RF-<m>")
	f.delta(t, strings.Replace(strings.Replace(validDelta, "Alta de usuario (RF-001)", "Alta de usuario (RF-1)", 1), "Baja de usuario (RF-002)", "Baja de usuario (RF-2)", 1))
	report := f.check(t)
	if !report.Passed {
		t.Fatalf("RF-1 y RF-001 son la misma clave con RF-<m>: %+v", report.Failures)
	}
}

func TestCheck_OriginID_ConIDsInvalidosCompararPorCadena(t *testing.T) {
	f := newCheckFixture(t)
	withoutDeclaredPattern(f)
	f.delta(t, strings.Replace(validDelta, "Alta de usuario (RF-001)", "Alta de usuario (REQ-40.1)", 1))
	requireFailure(t, f.check(t), RuleOriginID, "Alta de usuario", "REQ-40.1", "RF-001")
}

func TestCheck_OriginID_SinInventarioSoloOriginalID(t *testing.T) {
	f := newCheckFixture(t)
	f.write(t, adoption.InventoryFileName, "")
	f.delta(t, strings.Replace(validDelta, "Alta de usuario (RF-001)", "Alta de usuario (RF-009)", 1))
	report := f.check(t)
	requireFailure(t, report, RuleOriginID, "RF-009", "RF-001")
	requireFailure(t, report, RuleInventoryMissing)
}

// REQ-30.10 y REQ-30.11: un `Origen:` con varios IDs o con texto que no es un ID se rechaza.
func TestCheck_OriginID_OrigenConVariosIDsOTextoNoID(t *testing.T) {
	for _, text := range []string{"RF-001, RF-002", "n/d"} {
		t.Run(text, func(t *testing.T) {
			f := newCheckFixture(t)
			f.delta(t, strings.Replace(validDelta, originRF001, "Origen: docs/requisitos.md#rf-001-alta-de-usuario ("+text+")", 1))
			requireFailure(t, f.check(t), RuleOriginID, "un único identificador o ninguno", text)
		})
	}
}

func TestCheck_OriginID_OrigenSinParentesisUsaLaUnidad(t *testing.T) {
	f := newCheckFixture(t)
	f.delta(t, strings.Replace(validDelta, originRF001, "Origen: docs/requisitos.md#rf-001-alta-de-usuario", 1))
	report := f.check(t)
	if !report.Passed {
		t.Fatalf("sin ID escrito, E sale de la unidad: %+v", report.Failures)
	}
	f.delta(t, strings.Replace(strings.Replace(validDelta, originRF001, "Origen: docs/requisitos.md#rf-001-alta-de-usuario", 1), "Alta de usuario (RF-001)", "Alta de usuario (RF-009)", 1))
	requireFailure(t, f.check(t), RuleOriginID, "RF-009", "RF-001")
}

// REQ-30.10 y REQ-30.11: fusión y división.
func TestCheck_OriginID_FusionDeUnidades(t *testing.T) {
	f := newCheckFixture(t)
	f.delta(t, deltaOf([2]string{"Alta y baja (RF-001)", "Origen: docs/requisitos.md#rf-001-alta-de-usuario\nOrigen: docs/requisitos.md#rf-002-baja-de-usuario"}))
	requireFailure(t, f.check(t), RuleOriginID, "Alta y baja", "RF-002", "un requisito por unidad", "provisional")
}

func TestCheck_OriginID_DivisionConElMismoID(t *testing.T) {
	f := newCheckFixture(t)
	f.delta(t, deltaOf(
		[2]string{"Alta (RF-001)", "Origen: docs/requisitos.md#rf-001-alta-de-usuario"},
		[2]string{"Alta bis (RF-001)", "Origen: docs/requisitos.md#rf-001-alta-de-usuario\nOrigen: docs/requisitos.md#rf-002-baja-de-usuario"},
	))
	requireFailure(t, f.check(t), RuleUniqueness, "RF-001", "provisional")
}

// REQ-30.9 y REQ-30.11: `pattern_undeclared`.
func TestCheck_PatternUndeclared_SinPatronDeclarado(t *testing.T) {
	f := newCheckFixture(t)
	withoutDeclaredPattern(f)
	report := f.check(t)
	requireFailure(t, report, RulePatternUndeclared, "sdd.id_pattern", "por defecto", "2 identificador", "sdd:", `  id_pattern: "RF-<NNN>"`, "rama de trabajo de la adopción", "multirepo", "adopt check")
}

func TestCheck_PatternUndeclared_PatronDeclaradoQueNoCubre(t *testing.T) {
	f := newCheckFixture(t)
	withDeclaredPattern(t, f, "HU-<NNN>")
	requireFailure(t, f.check(t), RulePatternUndeclared, "HU-<NNN>", "declarado en axiom.yaml", `"RF-<NNN>"`)
}

func TestCheck_PatternUndeclared_ConInventarioV1(t *testing.T) {
	f := newCheckFixture(t)
	withoutDeclaredPattern(f)
	inv := mustScan(t, f.loc.WorkspaceRoot)
	inv.ProposedIDPattern = nil
	data, err := inv.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	f.write(t, adoption.InventoryFileName, string(data))
	requireFailure(t, f.check(t), RulePatternUndeclared, `"RF-<NNN>"`)
}

func TestCheck_PatternUndeclared_SinIDsNoExigePatron(t *testing.T) {
	f := newCheckFixture(t)
	withoutDeclaredPattern(f)
	f.useDoc(t, "# Requisitos\n\n## Alta de usuario\n\nAlta.\n", "- docs/requisitos.md#requisitos: es el título")
	f.delta(t, deltaOf([2]string{"Alta (prov:40:1)", "Origen: docs/requisitos.md#alta-de-usuario"}))
	report := f.check(t)
	if got := failuresOf(report, RulePatternUndeclared); len(got) != 0 {
		t.Fatalf("sin IDs ajenos no se exige patrón: %+v", got)
	}
	if !report.Passed {
		t.Fatalf("debe pasar: %+v", report.Failures)
	}
}

func TestCheck_PatternUndeclared_ConflictoDeEsquema(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, "# Requisitos\n\n## RF-001 Alta\n\nA.\n\n## RNF-001 Rendimiento\n\nB.\n", "- docs/requisitos.md#requisitos: es el título")
	f.delta(t, deltaOf(
		[2]string{"Alta (RF-001)", "Origen: docs/requisitos.md#rf-001-alta"},
		[2]string{"Rendimiento (prov::1)", "Origen: docs/requisitos.md#rnf-001-rendimiento (RNF-001)"},
	))
	report := f.check(t)
	requireFailure(t, report, RulePatternUndeclared, "mixed_prefixes", "normaliza a mano los identificadores del origen", "axiom archive adopt scan --change "+checkChange)
	for _, fl := range failuresOf(report, RulePatternUndeclared) {
		if strings.Contains(fl.Detail, `id_pattern: "`) {
			t.Fatalf("con conflicto de esquema no hay líneas para copiar: %s", fl.Detail)
		}
	}
}

func TestCheck_PatternUndeclared_IgnoraLasUnidadesDescartadas(t *testing.T) {
	for name, discards := range map[string][]string{
		"con motivo": {"- docs/requisitos.md#requisitos: titulo", "- docs/requisitos.md#glosario-rnf-001: glosario"},
		"sin motivo": {"- docs/requisitos.md#requisitos: titulo", "- docs/requisitos.md#glosario-rnf-001"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newCheckFixture(t)
			f.useDoc(t, "# Requisitos\n\n## RF-001 Alta de usuario\n\nA.\n\n## RF-002 Baja de usuario\n\nB.\n\n## Glosario RNF-001\n\nC.\n", discards...)
			report := f.check(t)
			for _, rule := range []string{RulePatternUndeclared, RuleForeignIDConflict} {
				if got := failuresOf(report, rule); len(got) != 0 {
					t.Fatalf("una unidad descartada no cuenta para %s: %+v", rule, got)
				}
			}
		})
	}
}

func TestCheck_PatternUndeclared_UnIDNoApareceEnDosMotivos(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, "# Requisitos\n\n## RF-001 Alta\n\nA.\n\n## RF-001 Alta bis\n\nB.\n\n## RNF-001 Rendimiento\n\nC.\n", "- docs/requisitos.md#requisitos: es el título")
	report := f.check(t)
	requireFailure(t, report, RuleForeignIDConflict, "RF-001")
	for _, fl := range failuresOf(report, RulePatternUndeclared) {
		if strings.Contains(fl.Detail, "RF-001") {
			t.Fatalf("RF-001 ya consta en foreign_id_conflict: %s", fl.Detail)
		}
	}
}

// REQ-30.11: `foreign_id_conflict`.
func TestCheck_ForeignIDConflict_UnidadConVariosIDs(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, "# Requisitos\n\n## RF-001 y RF-002 Alta\n\nA.\n", "- docs/requisitos.md#requisitos: es el título")
	report := f.check(t)
	requireFailure(t, report, RuleForeignIDConflict, "docs/requisitos.md#rf-001-y-rf-002-alta", "RF-001", "RF-002", "normaliza el origen o descarta la unidad")
}

func TestCheck_ForeignIDConflict_MismoIDEnDosUnidades(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, "# Requisitos\n\n## RF-001 Alta\n\nA.\n\n## RF-001 Alta bis\n\nB.\n", "- docs/requisitos.md#requisitos: es el título")
	requireFailure(t, f.check(t), RuleForeignIDConflict, "RF-001", "docs/requisitos.md#rf-001-alta", "normaliza el origen o descarta la unidad")
}

func TestCheck_ForeignIDConflict_LosConflictosDeEsquemaNoVanAqui(t *testing.T) {
	f := newCheckFixture(t)
	f.useDoc(t, "# Requisitos\n\n## RF-001 Alta\n\nA.\n\n## RNF-001 Rendimiento\n\nB.\n", "- docs/requisitos.md#requisitos: es el título")
	if got := failuresOf(f.check(t), RuleForeignIDConflict); len(got) != 0 {
		t.Fatalf("los conflictos de esquema solo van en pattern_undeclared: %+v", got)
	}
}

func TestCheck_InventoryCorrupt_UnSoloFallo(t *testing.T) {
	f := newCheckFixture(t)
	withoutDeclaredPattern(f)
	f.write(t, adoption.InventoryFileName, "sources: [\n")
	report := f.check(t)
	for _, rule := range []string{RuleOriginID, RulePatternUndeclared, RuleForeignIDConflict} {
		if got := failuresOf(report, rule); len(got) != 0 {
			t.Fatalf("un inventario ilegible no provoca cascada de %s: %+v", rule, got)
		}
	}
	if got := failuresOf(report, RuleInventoryMissing); len(got) != 1 {
		t.Fatalf("debe haber un único fallo inventory_missing: %+v", report.Failures)
	}
}

func TestCheck_OriginID_UnidadCitadaInexistenteFallaOrigin(t *testing.T) {
	f := newCheckFixture(t)
	f.delta(t, strings.Replace(validDelta, originRF001, "Origen: docs/requisitos.md#no-existe (RF-001)", 1))
	report := f.check(t)
	requireFailure(t, report, RuleOrigin, "no-existe")
	if got := failuresOf(report, RuleOriginID); len(got) != 0 {
		t.Fatalf("una unidad inexistente es de la regla origin, no de origin_id: %+v", got)
	}
}
