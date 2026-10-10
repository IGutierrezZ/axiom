package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/specbaseline"
)

// Las pruebas del modo cambio de `axiom sdd archive-compose` (`--change
// --document ... --in-place`) y de la detección de «ya integrado» del modo
// documento (inc-24, REQ-24.43, REQ-24.46, REQ-24.47, D-13). Todos los textos son
// inventados.

// splitCommandLine separa una línea de comando con las comillas dobles que emite
// pathquote: las comillas agrupan y no hay escapes.
func splitCommandLine(line string) []string {
	var words []string
	var current strings.Builder
	inQuotes, started := false, false
	for _, r := range line {
		switch {
		case r == '"':
			inQuotes, started = !inQuotes, true
		case r == ' ' && !inQuotes:
			if started {
				words = append(words, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(r)
			started = true
		}
	}
	if started {
		words = append(words, current.String())
	}
	return words
}

func sha256Of(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestSplitCommandLine(t *testing.T) {
	got := splitCommandLine(`axiom sdd archive-compose --cwd "C:\mi carpeta\repo" --change inc-1 --in-place`)
	want := []string{"axiom", "sdd", "archive-compose", "--cwd", `C:\mi carpeta\repo`, "--change", "inc-1", "--in-place"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("palabras = %q, quería %q", got, want)
	}
}

// preparedIntegrateFixture monta el espacio con git, autoriza el repositorio de
// código por cambio y ejecuta `integrate prepare`: devuelve el comando de
// composición que imprime.
func preparedIntegrateFixture(t *testing.T) (*integrateFixture, string) {
	t.Helper()
	f := newIntegrateFixture(t, true)
	withIntegrationStatus(t, func(status sddstatus.Status) sddstatus.Status {
		status.ActionContext.AllowedEditRoots = append(append([]string{}, status.ActionContext.AllowedEditRoots...), f.ws.Backend)
		return status
	})
	stdout, err := runIntegrate(t, "prepare", "--cwd", f.ws.Master, "--change", integrateChange, "--json")
	if err != nil {
		t.Fatalf("integrate prepare: %v\n%s", err, stdout)
	}
	command := decodeIntegrateReport(t, stdout).ComposeCommand
	if command == "" {
		t.Fatalf("prepare debe imprimir el comando de composición:\n%s", stdout)
	}
	return f, command
}

// runPrintedCommand ejecuta tal cual el comando de composición que imprime prepare.
func runPrintedCommand(t *testing.T, command string) (stdout, stderr string, err error) {
	t.Helper()
	words := splitCommandLine(command)
	if len(words) < 4 || strings.Join(words[:3], " ") != "axiom sdd archive-compose" {
		t.Fatalf("comando inesperado: %s", command)
	}
	return runCompose(t, words[3:]...)
}

// Arnés 6.19 en prueba: el comando que imprime prepare se ejecuta sin cambios, y una
// segunda ejecución se rechaza como ya integrada.
func TestInPlaceAppliesThePlanPrepareFixedAndRefusesAReapplication(t *testing.T) {
	f, command := preparedIntegrateFixture(t)
	document := filepath.Join(f.ws.Specs, "01_Requisitos_Funcionales.md")
	plan, err := specbaseline.ReadPlan(filepath.Join(f.changeRoot, specbaseline.PlanFileName))
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runPrintedCommand(t, command)
	if err != nil {
		t.Fatalf("archive-compose --in-place: %v\n%s", err, stderr)
	}
	for _, want := range []string{"Integración del cambio inc-99-ejemplo: 1 documento(s) compuesto(s)", "requisitos-funcionales: 01_Requisitos_Funcionales.md", "axiom sdd integrate verify --cwd"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("el resumen no contiene %q:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stderr, `Correspondencia de IDs: RF-9001 -> RF-008 (dominio "requisitos-funcionales")`) {
		t.Errorf("la correspondencia de IDs va por la salida de error:\n%s", stderr)
	}
	written, err := os.ReadFile(document)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Of(written); got != plan.Documents[0].AfterSHA256 {
		t.Errorf("el documento debe coincidir con la huella after del plan:\n%s", written)
	}
	after, err := specbaseline.ReadPlan(filepath.Join(f.changeRoot, specbaseline.PlanFileName))
	if err != nil || after.Documents[0].State != specbaseline.StateIntegrated {
		t.Errorf("el plan debe marcar la composición como integrada: %v %+v", err, after.Documents)
	}

	_, _, err = runPrintedCommand(t, command)
	if err == nil {
		t.Fatal("la segunda ejecución debe rechazarse como ya integrada")
	}
	for _, want := range []string{"ya integrada", "RF-9001 -> RF-008", "axiom sdd integrate verify", "no se escribió ningún documento"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo no contiene %q: %v", want, err)
		}
	}
	again, _ := os.ReadFile(document)
	if !bytes.Equal(again, written) {
		t.Errorf("reaplicar no debe cambiar el documento")
	}
}

func TestInPlaceRejectsIncompatibleOrMissingOptions(t *testing.T) {
	f := newIntegrateFixture(t, false)
	base := []string{"--cwd", f.ws.Specs}
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"--in-place sin --change", []string{"--document", "requisitos-funcionales", "--in-place"}, []string{"--in-place exige --change", "axiom sdd integrate prepare"}},
		{"--change sin --in-place", []string{"--change", integrateChange, "--document", "requisitos-funcionales"}, []string{"--change solo se usa con --in-place", "--in-place"}},
		{"con --output", []string{"--change", integrateChange, "--document", "requisitos-funcionales", "--in-place", "--output", "salida.md"}, []string{"--in-place excluye --output", "las opciones de supersede se toman del plan"}},
		{"con --delta", []string{"--change", integrateChange, "--document", "requisitos-funcionales", "--in-place", "--delta", "delta.md"}, []string{"--in-place excluye --delta"}},
		{"con --supersede", []string{"--change", integrateChange, "--document", "requisitos-funcionales", "--in-place", "--supersede"}, []string{"--in-place excluye --supersede"}},
		{"con --superseded-requirements", []string{"--change", integrateChange, "--document", "requisitos-funcionales", "--in-place", "--superseded-requirements", "RF-001"}, []string{"--in-place excluye --superseded-requirements"}},
		{"con varias opciones excluidas", []string{"--change", integrateChange, "--document", "requisitos-funcionales", "--in-place", "--output", "x", "--supersede"}, []string{"--in-place excluye --output, --supersede"}},
		{"con --canonical", []string{"--change", integrateChange, "--document", "requisitos-funcionales", "--in-place", "--canonical", "c.md"}, []string{"--in-place excluye --canonical"}},
		{"sin --document", []string{"--change", integrateChange, "--in-place"}, []string{"--in-place exige al menos un --document", "coincidir exactamente"}},
		{"cambio con ruta", []string{"--change", "../otro", "--document", "requisitos-funcionales", "--in-place"}, []string{"no es un nombre de cambio válido", "axiom sdd archive-compose --cwd"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := runCompose(t, append(append([]string{}, base...), tt.args...)...)
			if err == nil {
				t.Fatal("se esperaba un rechazo")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error no contiene %q: %v", want, err)
				}
			}
			if stdout != "" {
				t.Errorf("un rechazo no escribe nada por la salida estándar: %q", stdout)
			}
		})
	}
}

func TestInPlaceRefusesWithoutAPlanAndWithAMismatchedList(t *testing.T) {
	f := newIntegrateFixture(t, true)

	_, _, err := runCompose(t, "--cwd", f.ws.Specs, "--change", integrateChange, "--document", "requisitos-funcionales", "--in-place")
	if err == nil || !strings.Contains(err.Error(), "no tiene plan de integración") || !strings.Contains(err.Error(), "axiom sdd integrate prepare --cwd") {
		t.Fatalf("sin plan debe nombrar prepare: %v", err)
	}

	preparedFixtureWith := func() {
		withIntegrationStatus(t, func(status sddstatus.Status) sddstatus.Status {
			status.ActionContext.AllowedEditRoots = append(append([]string{}, status.ActionContext.AllowedEditRoots...), f.ws.Backend)
			return status
		})
		if stdout, err := runIntegrate(t, "prepare", "--cwd", f.ws.Specs, "--change", integrateChange); err != nil {
			t.Fatalf("integrate prepare: %v\n%s", err, stdout)
		}
	}
	preparedFixtureWith()

	documentPath := filepath.Join(f.ws.Specs, "01_Requisitos_Funcionales.md")
	before, _ := os.ReadFile(documentPath)
	_, _, err = runCompose(t, "--cwd", f.ws.Specs, "--change", integrateChange, "--in-place", "--document", "requisitos-funcionales", "--document", "modelo-de-datos")
	if err == nil || !strings.Contains(err.Error(), "es de prosa") || !strings.Contains(err.Error(), "edición dirigida o solo registro") {
		t.Errorf("un dominio de prosa se rechaza indicando cómo se integra: %v", err)
	}
	_, _, err = runCompose(t, "--cwd", f.ws.Specs, "--change", integrateChange, "--in-place", "--document", "requisitos-funcionales", "--document", "nuevo-dominio")
	if err == nil || !strings.Contains(err.Error(), "'baseline.documents'") {
		t.Errorf("un dominio no declarado se rechaza (REQ-24.48): %v", err)
	}
	_, _, err = runCompose(t, "--cwd", f.ws.Specs, "--change", integrateChange, "--in-place", "--document", "requisitos-funcionales")
	if err != nil {
		t.Fatalf("la lista exacta sí se acepta: %v", err)
	}
	if after, _ := os.ReadFile(documentPath); bytes.Equal(before, after) {
		t.Errorf("la lista exacta compone el documento")
	}
}

// REQ-24.47 y REQ-24.43: el modo documento rechaza como ya integrado un documento
// que el plan del cambio marca como integrado o que ya coincide con su huella after.
func TestDocumentModeRefusesAnAlreadyIntegratedDocumentAndNamesTheCorrespondence(t *testing.T) {
	f, command := preparedIntegrateFixture(t)
	deltaRelative := "openspec/changes/" + integrateChange + "/specs/requisitos-funcionales/spec.md"
	preview := []string{"--cwd", f.ws.Specs, "--document", "requisitos-funcionales", "--delta", deltaRelative}

	// Con el plan vigente y sin aplicar, el modo documento previsualiza.
	preview1, _, err := runCompose(t, preview...)
	if err != nil {
		t.Fatalf("la previsualización antes de aplicar: %v", err)
	}

	if _, stderr, err := runPrintedCommand(t, command); err != nil {
		t.Fatalf("archive-compose --in-place: %v\n%s", err, stderr)
	}
	written, _ := os.ReadFile(filepath.Join(f.ws.Specs, "01_Requisitos_Funcionales.md"))
	if string(written) != preview1 {
		t.Errorf("la previsualización coincide con lo que escribe --in-place")
	}

	stdout, _, err := runCompose(t, preview...)
	if err == nil {
		t.Fatal("el modo documento debe rechazar un documento ya integrado")
	}
	for _, want := range []string{"ya integrada", `"requisitos-funcionales"`, "RF-9001 -> RF-008", "axiom sdd integrate verify"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo no contiene %q: %v", want, err)
		}
	}
	if stdout != "" {
		t.Errorf("un rechazo no emite el documento: %q", stdout)
	}

	// Aunque el plan siga pendiente, un documento que ya coincide con su huella after
	// se reconoce como integrado.
	planPath := filepath.Join(f.changeRoot, specbaseline.PlanFileName)
	plan, err := specbaseline.ReadPlan(planPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range plan.Documents {
		plan.Documents[i].State = ""
		if plan.Documents[i].AfterSHA256 != "" {
			plan.Documents[i].State = specbaseline.StatePending
		}
	}
	if err := specbaseline.WritePlan(planPath, plan); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runCompose(t, preview...); err == nil || !strings.Contains(err.Error(), "ya integrada") {
		t.Errorf("el documento que coincide con su huella after se rechaza: %v", err)
	}
}

// Un plan ilegible no se ignora en silencio en el modo documento.
func TestDocumentModeSurfacesAnUnreadablePlan(t *testing.T) {
	f := newIntegrateFixture(t, false)
	if err := os.WriteFile(filepath.Join(f.changeRoot, specbaseline.PlanFileName), []byte("no: [es un plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := runCompose(t, "--cwd", f.ws.Specs, "--document", "requisitos-funcionales", "--delta", "openspec/changes/"+integrateChange+"/specs/requisitos-funcionales/spec.md")
	if err == nil {
		t.Fatal("se esperaba un rechazo por el plan ilegible")
	}
	for _, want := range []string{"no se puede leer el plan de integración", specbaseline.PlanFileName, "axiom sdd integrate prepare --cwd"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
}

func TestArchiveComposeHelpDocumentsTheChangeMode(t *testing.T) {
	var out bytes.Buffer
	if err := runSDDArchiveComposeWithStderr([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--change <change> --document <domain>", "--in-place", "axiom sdd integrate prepare", "Un documento ya integrado se rechaza"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("la ayuda no contiene %q:\n%s", want, out.String())
		}
	}
}
