package acceptancetests

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestCheckFullCoverage(t *testing.T) {
	delta := []isoevidence.Requirement{
		req("d", "ADDED", "Uno REQ-1.1", "A", "B", "C"),
		req("d", "MODIFIED", "Dos REQ-1.2", "D", "E"),
	}
	text := header +
		row("ID", "P-01", "Cubre", "REQ-1.1#A; REQ-1.1#B") +
		row("ID", "P-02", "Cubre", "REQ-1.1#C; REQ-1.2#D") +
		row("ID", "P-03", "Cubre", "REQ-1.2#E") +
		row("ID", "P-04", "Cubre", "REQ-1.2#E")
	r := Check(text, baseInput(delta...))
	if !r.OK() || r.Cases != 4 || r.Scenarios != (Coverage{5, 5}) || r.Requirements != (Coverage{2, 2}) {
		t.Fatalf("informe = %+v", r)
	}
}

func TestCheckLooseIDCoversRequirementNotScenarios(t *testing.T) {
	r := Check(header+row("Cubre", "REQ-1.1"), baseInput(req("d", "ADDED", "Uno REQ-1.1", "A", "B")))
	mustOnly(t, r, CodeUncoveredScenario, CodeUncoveredScenario)
	if r.Requirements != (Coverage{1, 1}) || r.Scenarios != (Coverage{0, 2}) {
		t.Fatalf("informe = %+v", r)
	}
}

func TestCheckUncoveredRequirement(t *testing.T) {
	r := Check(header+row("Cubre", "REQ-1.1#A"),
		baseInput(req("d", "ADDED", "Uno REQ-1.1", "A"), req("e", "ADDED", "Dos REQ-2.1", "X", "Y")))
	mustOnly(t, r, CodeUncoveredScenario, CodeUncoveredScenario, CodeUncoveredReq)
	if r.Findings[2].File != "specs/e/spec.md" || r.Findings[2].Line != 0 {
		t.Fatalf("hallazgo de delta = %+v", r.Findings[0])
	}
}

func TestCheckOneFindingPerCode(t *testing.T) {
	// Un fixture por código: cada uno aísla su causa y comprueba el código y la línea.
	delta := req("d", "ADDED", "Uno REQ-1.1", "A")
	cases := []struct {
		name  string
		text  string
		input Input
		want  Code
		line  int
	}{
		{"tabla_invalida", "sin tabla\n", baseInput(delta), CodeInvalidTable, 0},
		{"celda_invalida tipo", header + row("Tipo", "exploratoria"), baseInput(delta), CodeInvalidCell, 3},
		{"celda_invalida prioridad", header + row("Prioridad", "urgente"), baseInput(delta), CodeInvalidCell, 3},
		{"celda_invalida pasos", header + row("Pasos", ""), baseInput(delta), CodeInvalidCell, 3},
		{"celda_invalida resultado", header + row("Resultado esperado", ""), baseInput(delta), CodeInvalidCell, 3},
		{"id_caso_invalido P1", header + row("ID", "P1"), baseInput(delta), CodeInvalidCaseID, 3},
		{"id_caso_invalido p-01", header + row("ID", "p-01"), baseInput(delta), CodeInvalidCaseID, 3},
		{"id_caso_duplicado", header + row("ID", "P-03") + row("ID", "P-03"), baseInput(delta), CodeDuplicateCaseID, 4},
		{"caso_sin_cubre vacío", header + row("Cubre", ""), baseInput(delta), CodeNoCubre, 3},
		{"caso_sin_cubre ;;", header + row("Cubre", "REQ-1.1#A;;REQ-1.1"), baseInput(delta), CodeNoCubre, 3},
		{"entorno_invalido no declarado", header + row("Entornos", "prod"), baseInput(delta), CodeInvalidEnvironment, 3},
		{"entorno_invalido repetido", header + row("Entornos", "dev; dev"), baseInput(delta), CodeInvalidEnvironment, 3},
		{"entorno_invalido vacío", header + row("Entornos", ""), baseInput(delta), CodeInvalidEnvironment, 3},
		{"rol_invalido", header + row("Rol", "backend"), baseInput(delta), CodeInvalidRole, 3},
		{"id_desconocido formato", header + row("Cubre", "ABC-9"), baseInput(delta), CodeUnknownID, 3},
		{"id_desconocido ausente", header + row("Cubre", "REQ-9.9"), baseInput(delta), CodeUnknownID, 3},
		{"escenario_desconocido", header + row("Cubre", "REQ-1.1#Zzz"), baseInput(delta), CodeUnknownScenario, 3},
		{"requisito_sin_id", header + row(), baseInput(delta, req("d", "ADDED", "Sin identificador", "S")), CodeRequirementNoID, 0},
		{"id_duplicado_en_delta", header + row(), baseInput(delta, req("e", "MODIFIED", "Otra REQ-1.1", "S")), CodeDuplicateDeltaID, 0},
		{"escenario_ambiguo repetido", header + row("Cubre", "REQ-1.1#A"), baseInput(req("d", "ADDED", "Uno REQ-1.1", "A", "A")), CodeAmbiguousScenario, 0},
		{"escenario_ambiguo punto y coma", header + row("Cubre", "REQ-1.1#A"), baseInput(req("d", "ADDED", "Uno REQ-1.1", "A", "x; y")), CodeAmbiguousScenario, 0},
		{"requisito_sin_cubrir", header + row("Cubre", "REQ-1.1#A"), baseInput(delta, req("d", "ADDED", "Dos REQ-1.2")), CodeUncoveredReq, 0},
		{"escenario_sin_cubrir", header + row("Cubre", "REQ-1.1#A"), baseInput(req("d", "ADDED", "Uno REQ-1.1", "A", "B")), CodeUncoveredScenario, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Check(tc.text, tc.input)
			var found *Finding
			for i := range r.Findings {
				if r.Findings[i].Code == tc.want {
					found = &r.Findings[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("falta %s en %+v", tc.want, r.Findings)
			}
			if found.Line != tc.line || found.Message == "" {
				t.Fatalf("hallazgo = %+v, línea esperada %d", *found, tc.line)
			}
			if r.OK() {
				t.Fatal("OK() con hallazgos")
			}
		})
	}
	// pruebas_ausente no lo emite Check: lo aporta MissingFinding.
	if f := MissingFinding(); f.Code != "pruebas_ausente" || f.File != FileName {
		t.Fatalf("MissingFinding = %+v", f)
	}
	if has(Check("", baseInput(delta)), CodeMissing) {
		t.Fatal("Check no debe emitir pruebas_ausente")
	}
}

func TestCheckRoleFindingListsSealedRoles(t *testing.T) {
	in := baseInput(req("d", "ADDED", "Uno REQ-1.1", "A"))
	in.Roles = []string{"fullstack"}
	r := Check(header+row("Rol", "backend"), in)
	for _, f := range r.Findings {
		if f.Code == CodeInvalidRole && f.Message != `el rol "backend" no es un rol sellado del cambio; roles sellados: fullstack` {
			t.Fatalf("mensaje = %q", f.Message)
		}
	}
}

func TestCheckRepeatedCitationCountsOnce(t *testing.T) {
	text := header + row("ID", "P-01", "Cubre", "REQ-1.1#A; REQ-1.1#A; REQ-1.1") + row("ID", "P-02", "Cubre", "REQ-1.1#A")
	r := Check(text, baseInput(req("d", "ADDED", "Uno REQ-1.1", "A")))
	if !r.OK() || r.Scenarios != (Coverage{1, 1}) || r.Requirements != (Coverage{1, 1}) {
		t.Fatalf("informe = %+v", r)
	}
}

func TestCheckRemovedAndRenamedAreOutsideTheBase(t *testing.T) {
	delta := []isoevidence.Requirement{
		req("d", "MODIFIED", "Mod REQ-1.1", "A"),
		req("d", "REMOVED", "Rem REQ-1.2"),
		{Domain: "d", Section: "RENAMED", Name: "Nuevo REQ-1.3", OldName: "Viejo REQ-1.3"},
	}
	r := Check(header+row("Cubre", "REQ-1.1#A")+row("ID", "P-02", "Cubre", "REQ-1.2")+row("ID", "P-03", "Cubre", "REQ-1.3"), baseInput(delta...))
	if r.Requirements != (Coverage{1, 1}) || r.Scenarios != (Coverage{1, 1}) {
		t.Fatalf("base = %+v", r)
	}
	if count(r, CodeUnknownID) != 2 {
		t.Fatalf("hallazgos = %+v", r.Findings)
	}
}

func TestCheckRequirementWithoutIDCountsInTotal(t *testing.T) {
	r := Check(header+row("Cubre", "REQ-1.1#A"),
		baseInput(req("d", "ADDED", "Uno REQ-1.1", "A"), req("d", "ADDED", "Sin id", "S")))
	if r.Requirements != (Coverage{1, 2}) || r.Scenarios != (Coverage{1, 1}) || !has(r, CodeRequirementNoID) {
		t.Fatalf("informe = %+v", r)
	}
}

func TestCheckProvisionalAndProjectPattern(t *testing.T) {
	prov := reqid.ProvisionalText("1", "1")
	text := header + row("Cubre", prov+"#A") + row("ID", "P-02", "Cubre", "REQ-1.2#B")
	r := Check(text, baseInput(req("d", "ADDED", "Uno ("+prov+")", "A"), req("d", "MODIFIED", "Dos REQ-1.2", "B")))
	if !r.OK() || r.Requirements != (Coverage{2, 2}) {
		t.Fatalf("informe = %+v", r)
	}
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	in := baseInput(req("d", "MODIFIED", "Tres RF-KOF-17", "Z"))
	in.Pattern = p
	if r := Check(header+row("Cubre", "RF-KOF-17#Z"), in); !r.OK() {
		t.Fatalf("patrón del proyecto: %+v", r.Findings)
	}
	// Un provisional que el patrón no admite o un token con texto alrededor no se reconoce.
	for _, bad := range []string{"prov:x:1", "x" + prov} {
		if r := Check(header+row("Cubre", bad), baseInput()); !has(r, CodeUnknownID) {
			t.Errorf("%q debería ser id_desconocido", bad)
		}
	}
}

func TestCheckSkipEnvironments(t *testing.T) {
	in := baseInput(req("d", "ADDED", "Uno REQ-1.1", "A"))
	in.Environments = nil
	text := header + row("Cubre", "REQ-1.1#A", "Entornos", "prod") + row("ID", "P-02", "Cubre", "REQ-1.1#A", "Rol", "otro")
	if r := Check(text, in); !has(r, CodeInvalidEnvironment) {
		t.Fatal("sin SkipEnvironments debe validar Entornos")
	}
	in.SkipEnvironments = true
	r := Check(text, in)
	if has(r, CodeInvalidEnvironment) {
		t.Fatalf("SkipEnvironments no debe emitir entorno_invalido: %+v", r.Findings)
	}
	if !has(r, CodeInvalidRole) {
		t.Fatal("el resto de comprobaciones sí debe ejecutarse")
	}
}

func TestCheckEquivalentToAffectedFromDelta(t *testing.T) {
	delta := []isoevidence.Requirement{
		req("d", "ADDED", "Uno REQ-1.1", "A", "B"),
		req("d", "MODIFIED", "Dos REQ-1.2", "C"),
		req("d", "REMOVED", "Tres REQ-1.3"),
	}
	r := Check(header, baseInput(delta...))
	want := map[[2]string]bool{}
	for _, k := range isoevidence.AffectedFromDelta(delta, reqid.Default()) {
		want[[2]string{k.RequirementID, k.Scenario}] = true
	}
	got := map[[2]string]bool{}
	for _, f := range r.Findings {
		if f.Code != CodeUncoveredScenario {
			continue
		}
		for k := range want {
			if f.Subject == k[0]+"#"+k[1] {
				got[k] = true
			}
		}
	}
	if !reflect.DeepEqual(got, want) || r.Scenarios.Total != len(want) {
		t.Fatalf("Check = %v (total %d), AffectedFromDelta = %v", got, r.Scenarios.Total, want)
	}
}

func TestCheckIsDeterministicAndSorted(t *testing.T) {
	delta := []isoevidence.Requirement{req("z", "ADDED", "Uno REQ-1.1", "A"), req("a", "ADDED", "Dos REQ-1.2", "B"), req("m", "ADDED", "Sin id")}
	text := header + row("ID", "P1", "Tipo", "x") + row("ID", "P-02", "Cubre", "REQ-9.9")
	first := Check(text, baseInput(delta...))
	for i := 0; i < 5; i++ {
		if again := Check(text, baseInput(delta...)); !reflect.DeepEqual(first, again) {
			t.Fatal("el informe no es determinista")
		}
	}
	for i := 1; i < len(first.Findings); i++ {
		a, b := first.Findings[i-1], first.Findings[i]
		if a.File > b.File || (a.File == b.File && a.Line > b.Line) {
			t.Fatalf("orden roto entre %+v y %+v", a, b)
		}
	}
}
