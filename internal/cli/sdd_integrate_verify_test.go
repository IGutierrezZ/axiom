package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/specbaseline"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom sdd integrate verify` y `show` (inc-24, REQ-24.51,
// REQ-24.52, REQ-24.57, D-13 pasos 4 a 6). Todos los textos son inventados.

const (
	integrateModeloSection = "\n## inc-99-ejemplo: Entidad Reintento\n\nCada reintento se asocia a un pedido (ver RF-008).\n"
	integrateAPISection    = "\n## Errores\n\nEl cobro reintentado devuelve el error 402.\n"
)

func appendIntegrate(t *testing.T, path, text string) {
	t.Helper()
	handle, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if _, err := handle.WriteString(text); err != nil {
		t.Fatal(err)
	}
}

func replaceIntegrate(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(old)) {
		t.Fatalf("%s no contiene %q", path, old)
	}
	if err := os.WriteFile(path, bytes.Replace(data, []byte(old), []byte(replacement), 1), 0o644); err != nil {
		t.Fatal(err)
	}
}

// applyDirectedEdits simula la edición dirigida del agente de archive: la sección
// del cambio del documento de prosa y las dos entradas del delta de contexto.
func (f *integrateFixture) applyDirectedEdits(t *testing.T) {
	t.Helper()
	appendIntegrate(t, filepath.Join(f.ws.Specs, "03_Modelo_Datos.md"), integrateModeloSection)
	replaceIntegrate(t, filepath.Join(f.ws.Specs, "context", "ARQUITECTURA.md"), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta el cobro (RF-008).")
	appendIntegrate(t, filepath.Join(f.ws.Backend, "docs", "API.md"), integrateAPISection)
}

// integrateCycleFixture lleva el cambio al punto de `verify`: prepare, el comando de
// composición que imprime y la edición dirigida simulada.
func integrateCycleFixture(t *testing.T) *integrateFixture {
	t.Helper()
	f, command := preparedIntegrateFixture(t)
	if _, stderr, err := runPrintedCommand(t, command); err != nil {
		t.Fatalf("archive-compose --in-place: %v\n%s", err, stderr)
	}
	f.applyDirectedEdits(t)
	return f
}

// traceHasRow dice si la tabla de la traza tiene una fila con ese principio y ese
// final: la columna del ancla de una entrada de composición es el encabezado del
// requisito y no se fija aquí.
func traceHasRow(markdown, prefix, suffix string) bool {
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(line, prefix) && strings.HasSuffix(line, suffix) {
			return true
		}
	}
	return false
}

func decodeVerifyReport(t *testing.T, stdout string) specbaseline.VerifyReport {
	t.Helper()
	var report specbaseline.VerifyReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("la salida no es un informe de verificación JSON: %v\n%s", err, stdout)
	}
	return report
}

func verifyArgs(f *integrateFixture, extra ...string) []string {
	return append([]string{"verify", "--cwd", f.ws.Master, "--change", integrateChange}, extra...)
}

func showArgs(f *integrateFixture, extra ...string) []string {
	return append([]string{"show", "--cwd", f.ws.Master, "--change", integrateChange}, extra...)
}

// moveToArchive mueve el cambio a openspec/changes/archive/<nombre>, como el
// Step 3 de la skill de archive.
func (f *integrateFixture) moveToArchive(t *testing.T, name string) string {
	t.Helper()
	archive := filepath.Join(f.ws.Home, "changes", "archive")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(archive, name)
	if err := os.Rename(f.changeRoot, target); err != nil {
		t.Fatal(err)
	}
	return target
}

// REQ-24.51: con la edición dentro de alcance, verify sale con código 0, en texto y
// en JSON, y solo escribe el plan.
func TestIntegrateVerifyExitsZeroAndOnlyWritesThePlan(t *testing.T) {
	f := integrateCycleFixture(t)
	before := f.snapshot(t)

	stdout, err := runIntegrate(t, verifyArgs(f)...)
	if err != nil {
		t.Fatalf("integrate verify: %v\n%s", err, stdout)
	}
	for _, want := range []string{
		"Verificación de la integración del cambio inc-99-ejemplo: superada",
		"integration.yaml",
		"axiom sdd integrate show --cwd",
		"--format markdown",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, stdout)
		}
	}
	changed := changedIntegrateFiles(before, f.snapshot(t))
	if len(changed) != 1 || !strings.HasSuffix(changed[0], "changes/inc-99-ejemplo/integration.yaml") {
		t.Errorf("verify solo escribe el plan: %v", changed)
	}

	// En JSON: axiom.sdd-integration-verify/v1, sin violaciones.
	stdout, err = runIntegrate(t, verifyArgs(f, "--json")...)
	if err != nil {
		t.Fatalf("integrate verify --json: %v\n%s", err, stdout)
	}
	report := decodeVerifyReport(t, stdout)
	if report.Schema != "axiom.sdd-integration-verify/v1" || report.Change != integrateChange || !report.Passed || len(report.Violations) != 0 {
		t.Errorf("informe = %+v", report)
	}
}

// Un fallo sale con código 1, lista todas las violaciones, nombra la continuación y
// no modifica documentos ni plan.
func TestIntegrateVerifyExitsOneListingEveryViolationAndChangesNothing(t *testing.T) {
	f := integrateCycleFixture(t)
	workspacetest.WriteFile(t, filepath.Join(f.ws.Specs, "referencia", "legado", "README.sh"), "#!/bin/sh\n")
	appendIntegrate(t, filepath.Join(f.ws.Specs, "04_Flujos_Negocio.md"), "\nretoque\n")
	replaceIntegrate(t, filepath.Join(f.ws.Specs, "03_Modelo_Datos.md"), "## inc-99-ejemplo: Entidad Reintento", "## Entidad Reintento")
	before := f.snapshot(t)

	stdout, err := runIntegrate(t, verifyArgs(f)...)
	if err == nil {
		t.Fatalf("verify debe salir con código distinto de cero:\n%s", stdout)
	}
	for _, want := range []string{
		"no superada",
		"[scope]", "repo-specs/referencia/legado/README.sh", "referencia",
		"repo-specs/04_Flujos_Negocio.md", "register",
		"[anchor]", "Entidad Reintento",
	} {
		if !strings.Contains(stdout, want) && !strings.Contains(strings.ReplaceAll(stdout, `\`, "/"), want) {
			t.Errorf("la salida no contiene %q:\n%s", want, stdout)
		}
	}
	for _, want := range []string{"no se superó", "violación(es)", "axiom sdd integrate verify --cwd"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	if changed := changedIntegrateFiles(before, f.snapshot(t)); len(changed) != 0 {
		t.Errorf("un fallo no modifica documentos ni plan: %v", changed)
	}

	jsonOut, jsonErr := runIntegrate(t, verifyArgs(f, "--json")...)
	if jsonErr == nil {
		t.Fatal("verify --json también sale con código distinto de cero")
	}
	report := decodeVerifyReport(t, jsonOut)
	if report.Passed || len(report.Violations) < 3 {
		t.Errorf("informe = %+v", report)
	}
}

func TestIntegrateVerifyWithoutAPlanReportsTheTraceViolation(t *testing.T) {
	f := newIntegrateFixture(t, true)
	stdout, err := runIntegrate(t, verifyArgs(f)...)
	if err == nil {
		t.Fatalf("sin plan, verify sale con código distinto de cero:\n%s", stdout)
	}
	for _, want := range []string{"[trace]", "integration.yaml", "axiom sdd integrate prepare"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, stdout)
		}
	}
}

// REQ-24.52: show da la traza en Markdown, con la tabla y el identificador
// provisional a definitivo, y en JSON.
func TestIntegrateShowRendersTheTraceInMarkdownAndJSON(t *testing.T) {
	f := integrateCycleFixture(t)
	if stdout, err := runIntegrate(t, verifyArgs(f)...); err != nil {
		t.Fatalf("integrate verify: %v\n%s", err, stdout)
	}
	before := f.snapshot(t)

	markdown, err := runIntegrate(t, showArgs(f)...)
	if err != nil {
		t.Fatalf("integrate show: %v\n%s", err, markdown)
	}
	explicit, err := runIntegrate(t, showArgs(f, "--format", "markdown")...)
	if err != nil || explicit != markdown {
		t.Fatalf("markdown es el formato por defecto: %v\n%s\n---\n%s", err, markdown, explicit)
	}
	for _, want := range []string{
		"\n## Traza de integración\n",
		"| Identificador | Documento | Ancla | Acción | Estado |",
		"| Entidad Reintento | 03_Modelo_Datos.md | inc-99-ejemplo | added | integrated |",
		"| C1 | context/ARQUITECTURA.md | Componentes > Servicio de pedidos | modified | integrated |",
		"| Flujo de devolución | 04_Flujos_Negocio.md | inc-99-ejemplo | added | pending |",
		"Cambio: `inc-99-ejemplo`.",
	} {
		if !strings.Contains(markdown, want) {
			t.Errorf("la traza no contiene %q:\n%s", want, markdown)
		}
	}

	if !traceHasRow(markdown, "| RF-9001 → RF-008 | 01_Requisitos_Funcionales.md | ", " | added | integrated |") {
		t.Errorf("la traza lleva la correspondencia RF-9001 → RF-008 del ADDED integrado:\n%s", markdown)
	}

	jsonOut, err := runIntegrate(t, showArgs(f, "--format", "json")...)
	if err != nil {
		t.Fatalf("integrate show --format json: %v\n%s", err, jsonOut)
	}
	var view specbaseline.TraceView
	if err := json.Unmarshal([]byte(jsonOut), &view); err != nil {
		t.Fatalf("la salida no es JSON: %v\n%s", err, jsonOut)
	}
	if view.Schema != specbaseline.TraceSchema || view.Change != integrateChange || view.VerifiedAt == nil || len(view.Entries) != 6 {
		t.Errorf("vista = %+v", view)
	}
	if changed := changedIntegrateFiles(before, f.snapshot(t)); len(changed) != 0 {
		t.Errorf("show solo lee: %v", changed)
	}
}

// «Sin impacto» y los avisos aparecen en la traza con su motivo.
func TestIntegrateShowRecordsTheNoImpactDecisionWithItsReason(t *testing.T) {
	f := newIntegrateFixture(t, true)
	f.writeChange(t, specbaseline.ContextDeltaFileName, "Sin impacto: el cambio solo corrige una errata en un mensaje de error\n")
	integrateCommit(t, f.ws.Specs, "sin impacto")
	if stdout, err := runIntegrate(t, "prepare", "--cwd", f.ws.Master, "--change", integrateChange); err != nil {
		t.Fatalf("integrate prepare: %v\n%s", err, stdout)
	}
	markdown, err := runIntegrate(t, showArgs(f)...)
	if err != nil {
		t.Fatalf("integrate show: %v\n%s", err, markdown)
	}
	for _, want := range []string{"### Decisión sobre el contexto técnico", "Sin impacto: el cambio solo corrige una errata en un mensaje de error", "Verificada: pendiente"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("la traza no contiene %q:\n%s", want, markdown)
		}
	}
}

// Arnés 6.26 en prueba: el ciclo completo de D-13. prepare, el comando de
// composición que imprime, la edición dirigida simulada, verify, mover el cambio a
// archive/<fecha>-<cambio> y show, cuya salida se añade al informe de archive.
func TestIntegrateFullCycleEndsWithTheTraceOfTheArchivedChange(t *testing.T) {
	f := integrateCycleFixture(t)
	if stdout, err := runIntegrate(t, verifyArgs(f)...); err != nil {
		t.Fatalf("integrate verify: %v\n%s", err, stdout)
	}
	active, err := runIntegrate(t, showArgs(f)...)
	if err != nil {
		t.Fatalf("integrate show (activo): %v", err)
	}

	archived := f.moveToArchive(t, "2026-10-10-"+integrateChange)
	got, err := runIntegrate(t, showArgs(f)...)
	if err != nil {
		t.Fatalf("integrate show sobre el cambio archivado con prefijo de fecha: %v\n%s", err, got)
	}
	if got != active {
		t.Errorf("la traza de un cambio archivado es la misma:\n%s\n---\n%s", active, got)
	}
	if _, err := os.Stat(filepath.Join(archived, specbaseline.PlanFileName)); err != nil {
		t.Fatalf("la traza viaja con el cambio: %v", err)
	}

	// `show --format markdown >> archive-report.md`.
	report := filepath.Join(archived, "archive-report.md")
	workspacetest.WriteFile(t, report, "# Informe de archive\n\nResumen del archivado.\n")
	appendIntegrate(t, report, got)
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "Resumen del archivado.\n\n## Traza de integración\n") {
		t.Errorf("la traza se separa del informe con una línea en blanco:\n%s", text)
	}
	if strings.Count(text, "| RF-9001 → RF-008 |") != 1 {
		t.Errorf("el informe contiene la traza una vez:\n%s", text)
	}

	// Con el cambio ya archivado, verify no lo vuelve a integrar.
	if stdout, err := runIntegrate(t, verifyArgs(f)...); err == nil {
		t.Errorf("verify sobre un cambio archivado debe rechazarse:\n%s", stdout)
	}
}

func TestIntegrateShowFindsAnArchivedChangeByItsExactName(t *testing.T) {
	f := integrateCycleFixture(t)
	f.moveToArchive(t, integrateChange)
	stdout, err := runIntegrate(t, showArgs(f, "--format", "json")...)
	if err != nil {
		t.Fatalf("integrate show: %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, `"change": "inc-99-ejemplo"`) {
		t.Errorf("salida = %s", stdout)
	}
}

func TestIntegrateShowExitsOneWithoutATrace(t *testing.T) {
	f := newIntegrateFixture(t, true)

	t.Run("cambio activo sin plan", func(t *testing.T) {
		_, err := runIntegrate(t, showArgs(f)...)
		if err == nil {
			t.Fatal("se esperaba un rechazo")
		}
		for _, want := range []string{"no tiene traza de integración", "integration.yaml", "axiom sdd integrate prepare"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el error no contiene %q: %v", want, err)
			}
		}
	})
	t.Run("cambio archivado sin plan", func(t *testing.T) {
		f.moveToArchive(t, "2026-10-10-"+integrateChange)
		_, err := runIntegrate(t, showArgs(f)...)
		if err == nil || !strings.Contains(err.Error(), "no tiene traza de integración") {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("cambio inexistente", func(t *testing.T) {
		_, err := runIntegrate(t, "show", "--cwd", f.ws.Master, "--change", "inc-00-no-existe")
		if err == nil {
			t.Fatal("se esperaba un rechazo")
		}
		for _, want := range []string{"inc-00-no-existe", "ni activo", "ni archivado", "axiom sdd status --cwd"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el error no contiene %q: %v", want, err)
			}
		}
	})
}

func TestIntegrateShowRejectsAnUnreadablePlanNamingTheContinuation(t *testing.T) {
	f := newIntegrateFixture(t, true)
	f.writeChange(t, specbaseline.PlanFileName, "schema: [\n")
	_, err := runIntegrate(t, showArgs(f)...)
	if err == nil || !strings.Contains(err.Error(), "no se puede leer") || !strings.Contains(err.Error(), "axiom sdd integrate prepare") {
		t.Errorf("error = %v", err)
	}
}

func TestIntegrateVerifyAndShowArgumentErrorsNameTheirContinuation(t *testing.T) {
	cases := map[string][]string{
		"show con un formato desconocido": {"show", "--change", integrateChange, "--format", "yaml"},
		"show no admite --json":           {"show", "--change", integrateChange, "--json"},
		"verify no admite --format":       {"verify", "--change", integrateChange, "--format", "json"},
		"verify no admite --supersede":    {"verify", "--change", integrateChange, "--supersede"},
		"show sin --change":               {"show"},
		"verify sin --change":             {"verify", "--cwd", "."},
		"verify con argumento posicional": {"verify", "--change", integrateChange, "sobra"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := runIntegrate(t, args...)
			if err == nil {
				t.Fatal("se esperaba un error de argumentos")
			}
			if !strings.Contains(err.Error(), "axiom sdd integrate") {
				t.Errorf("el error debe nombrar la continuación `axiom sdd integrate ...`: %v", err)
			}
		})
	}
}

func TestIntegrateHelpListsVerifyAndShow(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"verify", "-h"}, {"show", "--help"}} {
		stdout, err := runIntegrate(t, args...)
		if err != nil {
			t.Fatalf("integrate %v: %v", args, err)
		}
		for _, want := range []string{"Uso: axiom sdd integrate <check|prepare|verify|show>", "verify  --cwd", "show    --cwd", "--format markdown|json", "código 1"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("la ayuda no contiene %q:\n%s", want, stdout)
			}
		}
	}
}

func TestIntegrateUnknownSubcommandListsTheFourVerbs(t *testing.T) {
	_, err := runIntegrate(t, "borrar")
	if err == nil || !strings.Contains(err.Error(), "check, prepare, verify, show") {
		t.Errorf("error = %v", err)
	}
	_, err = runIntegrate(t)
	if err == nil || !strings.Contains(err.Error(), "check, prepare, verify o show") {
		t.Errorf("error = %v", err)
	}
}

// REQ-24.9 (A1): una compuerta sobre un cambio archivado con prefijo de fecha se
// rechaza como archivado, nombrando REQ-21.18.
func TestGateRecordRefusesAChangeArchivedUnderADatedFolder(t *testing.T) {
	root := t.TempDir()
	mustMkdirAllCLI(t, filepath.Join(root, "openspec", "changes", "archive", "2026-10-10-inc-01-old"))

	err := RunSDDGate([]string{"record", "--cwd", root, "--change", "inc-01-old", "--gate", "spec", "--decision", "approved", "--reason", "x"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunSDDGate() = nil error, se esperaba rechazo por raíz archivada (D-14)")
	}
	if !strings.Contains(err.Error(), "REQ-21.18") {
		t.Fatalf("error = %v, se esperaba que nombrase REQ-21.18", err)
	}
}
