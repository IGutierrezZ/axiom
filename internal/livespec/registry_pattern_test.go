package livespec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func mustPattern(t testing.TB, template string) reqid.Pattern {
	t.Helper()
	p, err := reqid.Parse(template)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// patternFixture monta un planning home con los ficheros dados y su registro con el patrón.
func patternFixture(t *testing.T, template string, files map[string]string) (workspace.Location, *Registry) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "openspec")
	writeTree(t, home, files)
	reg, err := LoadRegistry(home, RegistryOptions{Pattern: mustPattern(t, template)})
	if err != nil {
		t.Fatal(err)
	}
	return workspace.Location{WorkspaceRoot: root, PlanningHome: home}, reg
}

func specWith(capability string, reqs ...string) string {
	out := "# Especificación: " + capability + "\n\n## Requirements\n\n"
	for _, r := range reqs {
		out += "### Requirement: " + r + "\n\nEl sistema DEBE algo.\n\n"
	}
	return out
}

// REQ-30.4 y RF-RID-03: el siguiente libre usa la serie y el relleno del patrón.
func TestRegistryNextFreeWithDeclaredPattern(t *testing.T) {
	_, reg := patternFixture(t, "RF-<DOM>-<NN>", map[string]string{
		"specs/usuarios/spec.md": specWith("usuarios", "Alta (RF-USU-01)", "Baja (RF-USU-05)"),
		"specs/facturas/spec.md": specWith("facturas", "Emisión (RF-FAC-02)"),
		"specs/limites/spec.md":  specWith("limites", "Nueve (RF-LIM-09)", "Cien (RF-LIM-99)"),
		"specs/grandes/spec.md":  specWith("grandes", "Cien (RF-BIG-100)"),
	})
	p := reg.Pattern()
	if p.Template() != "RF-<DOM>-<NN>" {
		t.Fatalf("Pattern() = %q", p.Template())
	}
	for id, want := range map[string]string{
		"RF-USU-01":  "RF-USU-06", // los huecos no se rellenan
		"RF-FAC-02":  "RF-FAC-03", // otra serie no cuenta
		"RF-LIM-09":  "RF-LIM-100",
		"RF-BIG-100": "RF-BIG-101",
	} {
		series, ok := p.Series(id)
		if !ok {
			t.Fatalf("Series(%s) no ok", id)
		}
		if got := reg.NextFree(series); got != want {
			t.Errorf("NextFree(%s) = %s, quería %s", id, got, want)
		}
	}
	if refs := reg.LookupIdentity("RF-USU-05"); len(refs) != 1 || refs[0].ID != "RF-USU-05" {
		t.Errorf("LookupIdentity(RF-USU-05) = %+v", refs)
	}
	if refs := reg.LookupIdentity("RF-FAC-05"); len(refs) != 0 {
		t.Errorf("otra serie no comparte clave: %+v", refs)
	}
}

func TestRegistryNextFreeSingleSeriesAndDefault(t *testing.T) {
	_, reg := patternFixture(t, "RF-<m>", map[string]string{
		"specs/heredado/spec.md": specWith("heredado", "Uno (RF-120)", "Dos (RF-121)"),
	})
	series, _ := reg.Pattern().Series("RF-120")
	if got := reg.NextFree(series); got != "RF-122" {
		t.Errorf("NextFree = %s, quería RF-122", got)
	}

	_, def := patternFixture(t, "REQ-<n>.<m>", map[string]string{
		"specs/pagos/spec.md": specWith("pagos", "Cobro (REQ-30.4)"),
	})
	s, _ := def.Pattern().Series("REQ-30.4")
	if got := def.NextFree(s); got != "REQ-30.5" {
		t.Errorf("NextFree = %s, quería REQ-30.5", got)
	}
	// Sin Pattern, el registro usa el defecto.
	zero, err := LoadRegistry(t.TempDir(), RegistryOptions{})
	if err != nil || zero.Pattern() != reqid.Default() {
		t.Errorf("el registro sin patrón debe usar el defecto: %v", err)
	}
}

func rfDoc(t *testing.T, capability, delta string) FeatureDoc {
	t.Helper()
	return docWith(t, "", [2]string{capability, delta})
}

// REQ-30.4: la colisión propone el siguiente libre saltando los usados en el documento.
func TestPlanIntegrationCollisionProposesNextFreeWithDeclaredPattern(t *testing.T) {
	loc, reg := patternFixture(t, "RF-<DOM>-<NN>", map[string]string{
		"specs/usuarios/spec.md": specWith("usuarios", "Alta (RF-USU-01)"),
	})
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta masiva", "RF-USU-02", "El sistema DEBE dar de alta en lote.") +
		"\n" + addedBlock("Otra alta", "RF-USU-01", "El sistema DEBE otra cosa.")
	_, err := PlanIntegration(loc, rfDoc(t, "usuarios", delta), reg)
	var collision *CollisionError
	if !errors.As(err, &collision) {
		t.Fatalf("error = %v, se esperaba *CollisionError", err)
	}
	if collision.ID != "RF-USU-01" || collision.NextFree != "RF-USU-03" || !strings.Contains(collision.Where, "usuarios") {
		t.Errorf("collision = %+v", collision)
	}
}

// REQ-30.4: un ADDED con el formato por defecto se rechaza nombrando el patrón del proyecto.
func TestPlanIntegrationRejectsTheDefaultFormatWithDeclaredPattern(t *testing.T) {
	loc, reg := patternFixture(t, "RF-<DOM>-<NN>", nil)
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta", "REQ-31.1", "El sistema DEBE dar de alta.")
	_, err := PlanIntegration(loc, rfDoc(t, "usuarios", delta), reg)
	var docErr *DocError
	if !errors.As(err, &docErr) || len(docErr.Problems) != 1 || docErr.Problems[0].Code != "added_without_id" {
		t.Fatalf("error = %v, se esperaba added_without_id", err)
	}
	if !strings.Contains(docErr.Problems[0].Message, "RF-<DOM>-<NN>") {
		t.Errorf("el motivo debe nombrar el patrón del proyecto: %s", docErr.Problems[0].Message)
	}
}

// REQ-30.4 y RF-ARC-11: con el defecto el motivo es idéntico al de antes.
func TestPlanIntegrationAddedWithoutIDMessageIsUnchangedWithTheDefault(t *testing.T) {
	loc, reg := planFixture(t, nil)
	delta := "## ADDED Requirements\n\n### Requirement: Alta\n\nEl sistema DEBE dar de alta.\n"
	_, err := PlanIntegration(loc, rfDoc(t, "facturacion", delta), reg)
	var docErr *DocError
	if !errors.As(err, &docErr) {
		t.Fatalf("error = %v", err)
	}
	want := "el requisito ADDED \"Alta\" de la capacidad facturacion no lleva un identificador `(REQ-n.m)` en el título; añade el siguiente número libre de la serie, p. ej. `### Requirement: Alta (REQ-n.m)`, o un provisional `(prov:<serie>:<ordinal>)` cuyo número asigna la integración"
	if got := docErr.Problems[0].Message; got != want {
		t.Errorf("mensaje = %q\nquería %q", got, want)
	}
}

// writePlan escribe el contenido que el plan calculó para una capacidad.
func writePlan(t *testing.T, plan Integration) {
	t.Helper()
	for _, c := range plan.Capabilities {
		if c.Outcome == OutcomeAlready {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(c.Path, []byte(c.Content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// RF-ARC-11 y REQ-30.2: classify empareja MODIFIED y REMOVED por identificador y la segunda
// ejecución es idempotente (`already`).
func TestPlanIntegrationClassifiesByIDAndIsIdempotent(t *testing.T) {
	for _, tc := range []struct {
		name, template, modifiedID, removedID string
	}{
		{"RF-<DOM>-<NN>", "RF-<DOM>-<NN>", "RF-USU-01", "RF-USU-02"},
		{"REQ-<n>.<m>", "REQ-<n>.<m>", "REQ-31.1", "REQ-31.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loc, reg := patternFixture(t, tc.template, map[string]string{
				"specs/usuarios/spec.md": specWith("usuarios", "Alta ("+tc.modifiedID+")", "Baja ("+tc.removedID+")"),
			})
			delta := "## MODIFIED Requirements\n\n" + addedBlock("Alta nueva", tc.modifiedID, "El sistema DEBE dar de alta con validación.") +
				"\n## REMOVED Requirements\n\n### Requirement: Baja antigua (" + tc.removedID + ")\n\n(Reason: ya no aplica)\n"
			doc := rfDoc(t, "usuarios", delta)

			plan, err := PlanIntegration(loc, doc, reg)
			if err != nil {
				t.Fatalf("PlanIntegration: %v", err)
			}
			got := plan.Capabilities[0]
			if got.Outcome != OutcomeComposed {
				t.Fatalf("outcome = %s, quería composed", got.Outcome)
			}
			if !strings.Contains(got.Content, "### Requirement: Alta nueva ("+tc.modifiedID+")") || strings.Contains(got.Content, "("+tc.removedID+")") {
				t.Fatalf("el MODIFIED por ID reemplaza al requisito y el REMOVED lo borra:\n%s", got.Content)
			}

			writePlan(t, plan)
			reg2, err := LoadRegistry(loc.PlanningHome, RegistryOptions{Pattern: reg.Pattern()})
			if err != nil {
				t.Fatal(err)
			}
			again, err := PlanIntegration(loc, doc, reg2)
			if err != nil {
				t.Fatalf("segunda PlanIntegration: %v", err)
			}
			if again.Capabilities[0].Outcome != OutcomeAlready {
				t.Errorf("la segunda ejecución debe ser already_integrated: %s", again.Capabilities[0].Outcome)
			}
		})
	}
}

// El identificador de un requisito se deriva del título con el patrón (RF-RID-04): sin
// openspecdoc.Requirement.ID, la composición y la clasificación coinciden.
func TestClassifyAndComposeAgreeOnResolution(t *testing.T) {
	names := []string{"Alta (RF-USU-01)", "Baja (RF-USU-02)"}
	idx, match := openspecdoc.Resolve(names, "Baja vieja (RF-USU-02)", mustPattern(t, "RF-<DOM>-<NN>"))
	if match != openspecdoc.MatchFound || idx != 1 {
		t.Fatalf("Resolve = %d, %v", idx, match)
	}
}

// RF-ADP-03 y RF-RID-04: LookupIdentity compara por serie y número, no por cadena.
func TestRegistryLookupIdentityMatchesByKeyNotByString(t *testing.T) {
	_, reg := patternFixture(t, "RF-<m>", map[string]string{
		"specs/b/spec.md":                   specWith("b", "Uno (RF-001)"),
		"specs/a/spec.md":                   specWith("a", "Otro (RF-1)"),
		"changes/inc-1/specs/c/spec.md":     "## ADDED Requirements\n\n### Requirement: Tercero (RF-01)\n\nCuerpo.\n",
		"changes/inc-2/specs/d/spec.md":     "## ADDED Requirements\n\n### Requirement: Cuarto (RF-002)\n\nCuerpo.\n",
		"changes/archive/x/specs/e/spec.md": "## ADDED Requirements\n\n### Requirement: Archivado (RF-1)\n\nCuerpo.\n",
	})
	refs := reg.LookupIdentity("RF-1")
	if len(refs) != 3 {
		t.Fatalf("LookupIdentity(RF-1) = %+v, se esperaban 3 (RF-1, RF-001 y RF-01, incluido el cambio activo)", refs)
	}
	// Orden estable: ruta y después ID.
	for i := 1; i < len(refs); i++ {
		if refs[i-1].Path > refs[i].Path || (refs[i-1].Path == refs[i].Path && refs[i-1].ID > refs[i].ID) {
			t.Errorf("orden inestable: %+v", refs)
		}
	}
	if got := reg.LookupIdentity("RF-001"); len(got) != 3 {
		t.Errorf("LookupIdentity(RF-001) = %d referencias", len(got))
	}
	if got := reg.LookupIdentity("RF-3"); len(got) != 0 {
		t.Errorf("RF-3 está libre: %+v", got)
	}
}

func TestRegistryLookupIdentityFixedWidthRejectsOtherFormats(t *testing.T) {
	_, reg := patternFixture(t, "RF-<NNN>", map[string]string{
		"specs/a/spec.md": specWith("a", "Uno (RF-001)"),
	})
	if got := reg.LookupIdentity("RF-001"); len(got) != 1 {
		t.Errorf("LookupIdentity(RF-001) = %+v", got)
	}
	if got := reg.LookupIdentity("RF-1"); len(got) != 0 {
		t.Errorf("RF-1 no es válido con RF-<NNN>: %+v", got)
	}
}

func TestRegistryLookupIdentityHonoursExcludeChange(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "openspec")
	writeTree(t, home, map[string]string{
		"changes/propio/specs/a/spec.md": "## ADDED Requirements\n\n### Requirement: Uno (RF-1)\n\nCuerpo.\n",
		"changes/otro/specs/a/spec.md":   "## ADDED Requirements\n\n### Requirement: Dos (RF-001)\n\nCuerpo.\n",
	})
	reg, err := LoadRegistry(home, RegistryOptions{Pattern: mustPattern(t, "RF-<m>"), ExcludeChange: "propio"})
	if err != nil {
		t.Fatal(err)
	}
	refs := reg.LookupIdentity("RF-1")
	if len(refs) != 1 || refs[0].Change != "otro" {
		t.Fatalf("ExcludeChange debe dejar fuera el cambio propio: %+v", refs)
	}
}

func TestRegistryLookupIdentityDefaultPatternBehavesLikeLookup(t *testing.T) {
	reg, err := LoadRegistry(registryFixture(t), RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"REQ-31.1", "REQ-31.2", "REQ-99.1", "REQ-31.5"} {
		a, b := reg.Lookup(id), reg.LookupIdentity(id)
		if len(a) != len(b) {
			t.Errorf("%s: Lookup=%d LookupIdentity=%d", id, len(a), len(b))
		}
	}
	if got := reg.LookupIdentity("no es un id"); got != nil {
		t.Errorf("un ID inválido no tiene clave: %+v", got)
	}
}
