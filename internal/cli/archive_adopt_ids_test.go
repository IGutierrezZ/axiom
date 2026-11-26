package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/adoption/adoptcheck"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Prueba de integración del criterio de terminado de C-03-c (REQ-30.9 a REQ-30.13): una
// adopción con IDs del cliente `RF-001…RF-120` los conserva hasta la spec viva y un
// provisional posterior recibe `RF-121`. Todos los textos son inventados.

const (
	adoptIDsChange   = sealChange // el cambio sellado por guardProject
	adoptIDsDiscards = "# Adopción\n\n## Descartado\n\n- docs/requisitos.md#requisitos: es el título del documento\n"
	adoptIDsDeclare  = "sdd:\n  id_pattern: \"RF-<NNN>\"\n"
)

// adoptIDsDoc compone un documento con `total` secciones `## RF-NNN Requisito NNN`.
func adoptIDsDoc(total int) string {
	var b strings.Builder
	b.WriteString("# Requisitos\n\nIntroducción.\n\n")
	for i := 1; i <= total; i++ {
		fmt.Fprintf(&b, "## RF-%03d Requisito %03d\n\nTexto del requisito %d.\n\n", i, i, i)
	}
	return b.String()
}

func adoptIDsAnchor(i int) string {
	return fmt.Sprintf("docs/requisitos.md#rf-%03d-requisito-%03d", i, i)
}

func adoptIDsOrigin(i int) string {
	return fmt.Sprintf("Origen: %s (RF-%03d)", adoptIDsAnchor(i), i)
}

// adoptIDsDelta compone una delta ADDED con un requisito por unidad `RF-001…RF-total`.
func adoptIDsDelta(total int) string {
	var b strings.Builder
	b.WriteString("# Delta\n\n## ADDED Requirements\n\n")
	for i := 1; i <= total; i++ {
		b.WriteString(adoptIDsRequirement(fmt.Sprintf("Requisito %03d (RF-%03d)", i, i), adoptIDsOrigin(i)))
	}
	return b.String()
}

func adoptIDsRequirement(title, origin string) string {
	out := "### Requirement: " + title + "\n\nEl sistema DEBE cumplir el requisito.\n\n"
	if origin != "" {
		out += origin + "\n\n"
	}
	return out + "#### Scenario: Cumple\n- **DADO** un contexto\n- **CUANDO** se ejecuta\n- **ENTONCES** se cumple\n\n"
}

// adoptIDsProject monta el proyecto sellado, deja el checkout en la rama del cambio y escribe
// el documento del cliente; con `declare` añade `sdd.id_pattern` a axiom.yaml.
func adoptIDsProject(t *testing.T, doc, declare string) workspacetest.Workspace {
	t.Helper()
	ws := guardProject(t)
	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), doc)
	if declare != "" {
		declareAdoptPattern(t, ws, declare)
	}
	return ws
}

func adoptIDsRoot(ws workspacetest.Workspace) string {
	return filepath.Join(ws.Home, "changes", adoptIDsChange)
}

// adoptIDsScanAndWrite hace `scan --change` y escribe la delta y los descartes del cambio.
func adoptIDsScanAndWrite(t *testing.T, ws workspacetest.Workspace, delta string) {
	t.Helper()
	if out, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", adoptIDsChange); err != nil {
		t.Fatalf("scan --change: %v\n%s", err, out)
	}
	workspacetest.WriteFile(t, filepath.Join(adoptIDsRoot(ws), "specs", "proyecto", "spec.md"), delta)
	workspacetest.WriteFile(t, filepath.Join(adoptIDsRoot(ws), adoption.DiscardsFileName), adoptIDsDiscards)
}

func adoptIDsCheck(t *testing.T, ws workspacetest.Workspace) (adoptcheck.CheckReport, error) {
	t.Helper()
	stdout, err := runArchiveAdoptCLI(t, "check", "--path", ws.Master, "--change", adoptIDsChange, "--json")
	var report adoptcheck.CheckReport
	if jerr := json.Unmarshal([]byte(stdout), &report); jerr != nil {
		t.Fatalf("check --json no es un informe: %v\n%s", jerr, stdout)
	}
	return report, err
}

// adoptIDsArchive pasa a la rama de archivado y ejecuta `archive-integrate`.
func adoptIDsArchive(t *testing.T, ws workspacetest.Workspace) string {
	t.Helper()
	integrateGit(t, ws.Master, "switch", "-q", "-c", guardArchiveBranch)
	if out, err := runSDDIntegrate(t, ws, "--json"); err != nil {
		t.Fatalf("archive-integrate: %v\n%s", err, out)
	}
	return readODDFile(t, filepath.Join(ws.Home, "specs", "proyecto", "spec.md"))
}

func lastChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// REQ-30.9, REQ-30.11, REQ-30.12 y REQ-30.13: de `scan` a la spec viva con los IDs conservados.
func TestAdoptIDsCriterion(t *testing.T) {
	const total = 120
	ws := adoptIDsProject(t, adoptIDsDoc(total), adoptIDsDeclare)
	axiomYaml := filepath.Join(ws.Master, "axiom.yaml")
	axiomBefore := readODDFile(t, axiomYaml)

	// (2) scan --change propone RF-<NNN> sin tocar axiom.yaml.
	if out, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", adoptIDsChange); err != nil {
		t.Fatalf("scan --change: %v\n%s", err, out)
	}
	inv, err := adoption.ReadInventory(adoptIDsRoot(ws))
	if err != nil || inv.ProposedIDPattern == nil {
		t.Fatalf("inventario = %+v, err = %v", inv, err)
	}
	if inv.ProposedIDPattern.Template != "RF-<NNN>" || inv.ProposedIDPattern.IDs != total || len(inv.ProposedIDPattern.Conflicts) != 0 {
		t.Fatalf("propuesta = %+v", inv.ProposedIDPattern)
	}
	if readODDFile(t, axiomYaml) != axiomBefore {
		t.Fatal("scan no escribe axiom.yaml")
	}

	// (3) la delta con los 120 ADDED y el título del documento descartado.
	adoptIDsScanAndWrite(t, ws, adoptIDsDelta(total))

	// (4) check pasa con el patrón declarado.
	report, err := adoptIDsCheck(t, ws)
	if err != nil || !report.Passed || report.IDPattern != "RF-<NNN>" || !report.IDPatternDeclared {
		t.Fatalf("check: %v, informe = %+v", err, report)
	}

	// (5) archive-integrate conserva los IDs y los Origen: byte a byte.
	spec := adoptIDsArchive(t, ws)
	for i := 1; i <= total; i++ {
		for _, want := range []string{fmt.Sprintf("(RF-%03d)", i), adoptIDsOrigin(i)} {
			if !strings.Contains(spec, want) {
				t.Fatalf("la spec viva pierde %q", want)
			}
		}
	}
	if strings.Contains(spec, "prov:") {
		t.Errorf("la spec viva no lleva provisionales:\n%s", lastChars(spec, 600))
	}

	// 5.2: un ADDED provisional posterior recibe RF-121 con el mismo motor que archive-integrate.
	adoptIDsIntegrateSecond(t, ws, "y", adoptIDsRequirement("Nuevo requisito (prov::1)", ""))
	after := readODDFile(t, filepath.Join(ws.Home, "specs", "proyecto", "spec.md"))
	if !strings.Contains(after, "Nuevo requisito (RF-121)") || strings.Contains(after, "prov:") {
		t.Fatalf("el provisional debe recibir RF-121:\n%s", lastChars(after, 600))
	}
	if !strings.Contains(after, "(RF-120)") {
		t.Error("los definitivos previos se conservan")
	}
}

// adoptIDsIntegrateSecond integra la delta `body` de la capacidad `proyecto` de otro cambio
// con el motor de archive-integrate (PlanChange + Apply), sin un segundo cambio sellado.
func adoptIDsIntegrateSecond(t *testing.T, ws workspacetest.Workspace, change, body string) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", change, "specs", "proyecto", "spec.md"), "## ADDED Requirements\n\n"+body)
	loc, err := workspace.Locate(ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	pattern, err := projectIDPattern("test", loc)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := livespec.LoadRegistry(loc.PlanningHome, livespec.RegistryOptions{Pattern: pattern, ExcludeChange: change})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := livespec.PlanChange(loc, change, livespec.ChangeOptions{}, reg)
	if err != nil {
		t.Fatalf("PlanChange: %v", err)
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

// REQ-30.12: los definitivos ocupan primero; un provisional anterior en la delta recibe máximo+1.
func TestAdoptIDsProvisionalBeforeTheHighestDefinitive(t *testing.T) {
	const total = 120
	ws := adoptIDsProject(t, adoptIDsDoc(total), adoptIDsDeclare)
	var b strings.Builder
	b.WriteString("# Delta\n\n## ADDED Requirements\n\n")
	for i := 1; i <= total; i++ {
		if i == total {
			// El provisional va antes del definitivo más alto; cita la unidad sin ID del documento.
			b.WriteString(adoptIDsRequirement("Extra sin origen (prov::1)", "Origen: docs/requisitos.md#requisitos"))
		}
		b.WriteString(adoptIDsRequirement(fmt.Sprintf("Requisito %03d (RF-%03d)", i, i), adoptIDsOrigin(i)))
	}
	adoptIDsScanAndWrite(t, ws, b.String())
	if report, err := adoptIDsCheck(t, ws); err != nil || !report.Passed {
		t.Fatalf("check: %v, informe = %+v", err, report)
	}
	spec := adoptIDsArchive(t, ws)
	if !strings.Contains(spec, "Extra sin origen (RF-121)") || !strings.Contains(spec, "Requisito 120 (RF-120)") {
		t.Fatalf("el provisional recibe RF-121 aunque RF-120 vaya después:\n%s", lastChars(spec, 800))
	}
}

// REQ-30.9 y REQ-30.11: sin el patrón declarado el cambio no pasa y el fallo cita la línea a copiar.
func TestAdoptIDsPatternUndeclaredFails(t *testing.T) {
	const total = 120
	ws := adoptIDsProject(t, adoptIDsDoc(total), "")
	adoptIDsScanAndWrite(t, ws, adoptIDsDelta(total))
	report, err := adoptIDsCheck(t, ws)
	if err == nil || report.Passed {
		t.Fatalf("sin id_pattern el check debe fallar: %+v", report)
	}
	found := false
	var rules []string
	for _, f := range report.Failures {
		rules = append(rules, f.Rule)
		if f.Rule == adoptcheck.RulePatternUndeclared {
			found = true
			if !strings.Contains(f.Detail, `id_pattern: "RF-<NNN>"`) {
				t.Errorf("el fallo cita la línea a copiar: %s", f.Detail)
			}
		}
	}
	if !found {
		t.Fatalf("falta %s; reglas = %v", adoptcheck.RulePatternUndeclared, rules)
	}
}

// REQ-30.12: un origen sin IDs, sin patrón declarado, usa provisionales y el patrón por defecto.
func TestAdoptIDsOriginWithoutIDsUsesTheDefaultPattern(t *testing.T) {
	doc := "# Requisitos\n\n## Alta de usuario\n\nTexto.\n\n## Baja de usuario\n\nTexto.\n"
	ws := adoptIDsProject(t, doc, "")
	delta := "# Delta\n\n## ADDED Requirements\n\n" +
		adoptIDsRequirement("Alta de usuario (prov:40:1)", "Origen: docs/requisitos.md#alta-de-usuario") +
		adoptIDsRequirement("Baja de usuario (prov:40:2)", "Origen: docs/requisitos.md#baja-de-usuario")
	adoptIDsScanAndWrite(t, ws, delta)
	inv, err := adoption.ReadInventory(adoptIDsRoot(ws))
	if err != nil || inv.ProposedIDPattern != nil {
		t.Fatalf("sin IDs no hay propuesta: %+v (%v)", inv.ProposedIDPattern, err)
	}
	report, err := adoptIDsCheck(t, ws)
	if err != nil || !report.Passed || report.IDPatternDeclared || report.IDPattern != reqid.Default().Template() {
		t.Fatalf("check: %v, informe = %+v", err, report)
	}
	spec := adoptIDsArchive(t, ws)
	if !strings.Contains(spec, "Alta de usuario (REQ-40.1)") || !strings.Contains(spec, "Baja de usuario (REQ-40.2)") || strings.Contains(spec, "prov:") {
		t.Fatalf("la spec viva usa REQ-40.1 y REQ-40.2:\n%s", spec)
	}
}
