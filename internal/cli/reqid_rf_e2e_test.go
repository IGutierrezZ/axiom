package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas extremo a extremo de un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`
// (RF-RID-07, RF-RID-08). Todo corre en proceso con repositorios git temporales: nunca se
// compila ni se lanza un binario.

const rfSDDBlock = "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n"

const rfISOBlock = "sdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n"

func rfPattern(t testing.TB) reqid.Pattern {
	t.Helper()
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func rfRequirement(name, id, body string) string {
	return "### Requirement: " + name + " (" + id + ")\n\n" + body + "\n\n#### Scenario: Caso\n\n- **CUANDO** ocurre\n- **ENTONCES** se cumple\n"
}

// TestReqidRFOddFlow cubre los puntos 1 a 4 y 8: integración ODD, colisión con el siguiente
// libre, ADDED con el formato por defecto, composición con título cambiado y el estado
// transitorio de una spec que aún lleva el formato anterior.
func TestReqidRFOddFlow(t *testing.T) {
	f := newODDFixture(t, true)
	p := rfPattern(t)
	workspacetest.WriteFile(t, filepath.Join(f.root, "axiom.yaml"), readODDFile(t, filepath.Join(f.root, "axiom.yaml"))+rfSDDBlock)
	workspacetest.WriteFile(t, f.specPath("usuarios"), "# Especificación: usuarios\n\n## Requirements\n\n"+rfRequirement("Alta", "RF-USU-01", "El sistema da de alta."))

	// (2) Un ADDED que repite RF-USU-01 se rechaza nombrando el identificador, la capacidad y
	// el siguiente libre, sin escribir.
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"usuarios", "## ADDED Requirements\n\n" + rfRequirement("Otra alta", "RF-USU-01", "El sistema hace otra cosa.")})))
	before := oddSnapshot(t, f.root)
	_, err := f.run(t)
	if err == nil {
		t.Fatal("un ADDED con un identificador existente debe rechazarse")
	}
	for _, want := range []string{"RF-USU-01", "usuarios", "RF-USU-02"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo %q debe nombrar %q", err, want)
		}
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("el rechazo no escribe nada")
	}

	// (1) Un ADDED con el siguiente identificador escribe la spec.
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"usuarios", "## ADDED Requirements\n\n" + rfRequirement("Alta masiva", "RF-USU-02", "El sistema da de alta en lote.")})))
	if _, err := f.run(t); err != nil {
		t.Fatalf("la integración de RF-USU-02 debe tener éxito: %v", err)
	}
	spec := readODDFile(t, f.specPath("usuarios"))
	if !strings.Contains(spec, "Alta (RF-USU-01)") || !strings.Contains(spec, "Alta masiva (RF-USU-02)") {
		t.Fatalf("la spec debe llevar los dos requisitos:\n%s", spec)
	}

	// (3) Un ADDED con el formato por defecto se rechaza nombrando el patrón.
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"usuarios", "## ADDED Requirements\n\n" + rfRequirement("Alta masiva", "REQ-31.1", "Texto.")})))
	_, err = f.run(t)
	if err == nil || !strings.Contains(err.Error(), "RF-<DOM>-<NN>") {
		t.Fatalf("un ADDED con REQ-31.1 debe rechazarse nombrando el patrón: %v", err)
	}

	// (4) sdd-archive-compose --cwd compone un ADDED RF-USU-03 y un MODIFIED con el título
	// cambiado y el mismo RF-USU-01.
	delta := "## ADDED Requirements\n\n" + rfRequirement("Baja", "RF-USU-03", "El sistema da de baja.") +
		"\n## MODIFIED Requirements\n\n" + rfRequirement("Alta con validación", "RF-USU-01", "El sistema da de alta y valida.")
	deltaPath := filepath.Join(f.root, "delta-usuarios.md")
	workspacetest.WriteFile(t, deltaPath, delta)
	var composed bytes.Buffer
	if err := runSDDArchiveCompose([]string{"--cwd", f.root, "--canonical", f.specPath("usuarios"), "--delta", deltaPath}, &composed); err != nil {
		t.Fatalf("sdd-archive-compose: %v", err)
	}
	out := composed.String()
	for _, want := range []string{"Alta con validación (RF-USU-01)", "Baja (RF-USU-03)", "Alta masiva (RF-USU-02)"} {
		if !strings.Contains(out, want) {
			t.Errorf("la composición debe contener %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "### Requirement: Alta (RF-USU-01)") {
		t.Errorf("el requisito RF-USU-01 se reemplaza por su título nuevo:\n%s", out)
	}

	// (8) Estado transitorio: la spec de facturación aún lleva `REQ-31.1`, que en este proyecto
	// no es un identificador. Un MODIFIED de título exacto se empareja por título y el
	// identificador no se resuelve en la baseline.
	modified := "## MODIFIED Requirements\n\n" + rfRequirement("Alta de factura", "REQ-31.1", "El sistema da de alta una factura y la notifica.")
	factDelta := filepath.Join(f.root, "delta-facturacion.md")
	workspacetest.WriteFile(t, factDelta, modified)
	var factComposed bytes.Buffer
	if err := runSDDArchiveCompose([]string{"--cwd", f.root, "--canonical", f.specPath("facturacion"), "--delta", factDelta}, &factComposed); err != nil {
		t.Fatalf("el MODIFIED de título exacto se empareja por título: %v", err)
	}
	if !strings.Contains(factComposed.String(), "y la notifica") {
		t.Errorf("el MODIFIED debe aplicarse:\n%s", factComposed.String())
	}
	var baseline []isoevidence.Requirement
	for _, r := range openspecdoc.ParseCanonical(readODDFile(t, f.specPath("facturacion"))) {
		baseline = append(baseline, isoevidence.Requirement{Domain: "facturacion", Name: r.Name, Text: r.Body})
	}
	if got := isoevidence.NewBaselineIndex(baseline, p)["REQ-31.1"]; len(got) != 0 {
		t.Errorf("REQ-31.1 no se resuelve en la baseline de un proyecto RF: %v", got)
	}
}

// TestReqidRFArchiveCheckResumesAnInterruptedArchive cubre el punto 5: alreadyComposed pasa
// de falso a verdadero al componer cada dominio y un archivado interrumpido y reanudado ni
// duplica ni pierde requisitos.
func TestReqidRFArchiveCheckResumesAnInterruptedArchive(t *testing.T) {
	p := rfPattern(t)
	const usuarios = "# usuarios\n\n## Requirements\n\n"
	const facturas = "# facturas\n\n## Requirements\n\n"
	canonicalUsuarios := usuarios + rfRequirement("Alta", "RF-USU-01", "El sistema da de alta.")
	canonicalFacturas := facturas + rfRequirement("Emisión", "RF-FAC-01", "El sistema emite facturas.")
	deltaUsuarios := "## ADDED Requirements\n\n" + rfRequirement("Baja", "RF-USU-02", "El sistema da de baja.") +
		"\n## MODIFIED Requirements\n\n" + rfRequirement("Alta masiva", "RF-USU-01", "El sistema da de alta en lote.")
	deltaFacturas := "## REMOVED Requirements\n\n### Requirement: Emisión antigua (RF-FAC-01)\n\n(Reason: ya no aplica)\n"

	root, _ := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", deltas: map[string]string{"usuarios": deltaUsuarios, "facturas": deltaFacturas},
		tasks: "- [x] 1.1 T\n  Cubre: RF-USU-01, RF-USU-02\n", verify: archiveCheckVerify(), iso: true,
	})
	writeSDDAxiomYAML(t, root, rfISOBlock)
	specs := filepath.Join(root, "openspec", "specs")
	writeCLIFixture(t, filepath.Join(specs, "usuarios", "spec.md"), canonicalUsuarios)
	writeCLIFixture(t, filepath.Join(specs, "facturas", "spec.md"), canonicalFacturas)

	domains := func() map[string]bool {
		t.Helper()
		out, _ := runArchiveCheck(root, "demo", "--json")
		var report struct {
			Domains []struct {
				Domain          string `json:"domain"`
				AlreadyComposed bool   `json:"alreadyComposed"`
			} `json:"domains"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("salida JSON no válida: %v\n%s", err, out)
		}
		got := map[string]bool{}
		for _, d := range report.Domains {
			got[d.Domain] = d.AlreadyComposed
		}
		return got
	}
	if got := domains(); got["usuarios"] || got["facturas"] || len(got) != 2 {
		t.Fatalf("antes de componer = %v", got)
	}

	// El archivado compone un dominio y se interrumpe antes del siguiente.
	options := sddstatus.ComposeOptions{IDPattern: p}
	composedUsuarios, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(canonicalUsuarios, deltaUsuarios, options)
	if err != nil {
		t.Fatalf("compose usuarios: %v", err)
	}
	writeCLIFixture(t, filepath.Join(specs, "usuarios", "spec.md"), composedUsuarios)
	if got := domains(); !got["usuarios"] || got["facturas"] {
		t.Fatalf("tras componer usuarios = %v, se esperaba usuarios=true y facturas=false", got)
	}

	// La reanudación solo compone lo que sigue en falso.
	composedFacturas, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(canonicalFacturas, deltaFacturas, options)
	if err != nil {
		t.Fatalf("compose facturas: %v", err)
	}
	writeCLIFixture(t, filepath.Join(specs, "facturas", "spec.md"), composedFacturas)
	if got := domains(); !got["usuarios"] || !got["facturas"] {
		t.Fatalf("tras reanudar = %v, se esperaba todo compuesto", got)
	}

	finalUsuarios := readODDFile(t, filepath.Join(specs, "usuarios", "spec.md"))
	if strings.Count(finalUsuarios, "(RF-USU-02)") != 1 || strings.Count(finalUsuarios, "(RF-USU-01)") != 1 || !strings.Contains(finalUsuarios, "Alta masiva (RF-USU-01)") {
		t.Errorf("sin duplicar ni perder requisitos:\n%s", finalUsuarios)
	}
	if finalFacturas := readODDFile(t, filepath.Join(specs, "facturas", "spec.md")); strings.Contains(finalFacturas, "RF-FAC-01") {
		t.Errorf("el REMOVED por identificador borra el requisito:\n%s", finalFacturas)
	}
}

// TestReqidRFISOStatusAndBugQuery cubre los puntos 6 y 7: un cambio ISO con identificadores
// del patrón no tiene motivos de identificador ni de cobertura, y la consulta de un bug con
// `Cubre: RF-USU-01` encuentra el requisito existente de la baseline.
func TestReqidRFISOStatusAndBugQuery(t *testing.T) {
	isoTokens := func(reasons []string) string {
		var tokens []string
		for _, reason := range reasons {
			for _, token := range []string{"iso_requirement_id", "iso_requirement_scenarios", "iso_coverage", "iso_cubre_unknown", "iso_bug_requirement", "iso_declaration_invalid"} {
				if strings.Contains(reason, "blocked("+token+")") {
					tokens = append(tokens, token)
				}
			}
		}
		return strings.Join(tokens, ",")
	}

	t.Run("cambio ISO con delta", func(t *testing.T) {
		delta := "## ADDED Requirements\n\n" + rfRequirement("Alta", "RF-USU-02", "El sistema da de alta.") + "\n" + rfRequirement("Baja", "RF-USU-03", "El sistema da de baja.")
		root, _ := archiveCheckWorkspace(t, archiveCheckChange{
			name: "demo", proposal: "# P\n", deltas: map[string]string{"usuarios": delta},
			tasks: "- [ ] 1.1 T\n  Cubre: RF-USU-02, RF-USU-03\n", iso: true,
		})
		writeSDDAxiomYAML(t, root, rfISOBlock)
		status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "demo"})
		if err != nil {
			t.Fatal(err)
		}
		if got := isoTokens(status.BlockedReasons); got != "" {
			t.Errorf("sin motivos de identificador ni de cobertura, y hay %s: %v", got, status.BlockedReasons)
		}
	})

	t.Run("bug con requisito existente", func(t *testing.T) {
		baseline := "# usuarios\n\n## Requirements\n\n" + rfRequirement("Alta", "RF-USU-01", "El sistema da de alta.")
		for name, tc := range map[string]struct{ cubre, want string }{
			"cita un requisito existente": {"RF-USU-01", ""},
			"cita uno desconocido":        {"RF-USU-99", "iso_cubre_unknown"},
		} {
			t.Run(name, func(t *testing.T) {
				root, _ := archiveCheckWorkspace(t, archiveCheckChange{
					name: "demo", proposal: archiveCheckBugReport,
					tasks: "- [ ] 1.1 T\n  Cubre: " + tc.cubre + "\n", iso: true,
				})
				writeSDDAxiomYAML(t, root, rfISOBlock)
				writeCLIFixture(t, filepath.Join(root, "openspec", "specs", "usuarios", "spec.md"), baseline)
				status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "demo"})
				if err != nil {
					t.Fatal(err)
				}
				if got := isoTokens(status.BlockedReasons); got != tc.want {
					t.Errorf("motivos = %q, quería %q: %v", got, tc.want, status.BlockedReasons)
				}
			})
		}
	})
}
