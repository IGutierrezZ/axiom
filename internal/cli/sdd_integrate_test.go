package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/specbaseline"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom sdd integrate check` y `prepare` (inc-24, REQ-24.42,
// REQ-24.54, D-13). Todos los textos son inventados.

const integrateChange = "inc-99-ejemplo"

const integrateConfig = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
baseline:
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      requirement:
        heading: '^#+ (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: 03_Modelo_Datos.md
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register
technical_context:
  maintained:
    paths:
      - context/
      - ../repo-backend/docs/
  reference:
    paths:
      - referencia/
`

const (
	integrateRequisitos = "# Requisitos funcionales\n\n## Catálogo\n\n### RF-001: Alta de producto\n\nEl sistema DEBE permitir dar de alta un producto.\n\n### RF-007: Consulta de producto\n\nEl sistema DEBE devolver los datos de un producto.\n"
	integrateModelo     = "# Modelo de datos\n\n## Entidades\n\n### Cliente\n\nTiene nombre y correo.\n"
	integrateFlujos     = "# Flujos de negocio\n\n## Pedidos\n\nEl pedido pasa por tres estados.\n"
	integrateContext    = "# Arquitectura\n\n## Componentes\n\n### Servicio de catálogo\n\nGestiona los productos.\n\n### Servicio de pedidos\n\nGestiona los pedidos.\n"
	integrateAPI        = "# API\n\n## Pagos\n\nTexto de pagos.\n"

	integrateDeltaRequisitos = `## ADDED Requirements

### Requirement: RF-9001: Reintento de pago
(Placement: after RF-007)

El sistema DEBE reintentar el cobro fallido hasta tres veces.

#### Scenario: Primer reintento
- **CUANDO** falla el cobro
- **ENTONCES** se reintenta

## MODIFIED Requirements

### Requirement: RF-001: Alta de producto con imagen

El sistema DEBE permitir adjuntar una imagen. Véase RF-9001.
`
	integrateDeltaModelo = "## ADDED Requirements\n\n### Requirement: Entidad Reintento\n\nCada reintento se asocia a un pedido (ver RF-9001).\n"
	integrateDeltaFlujos = "## ADDED Requirements\n\n### Requirement: Flujo de devolución\n\nDescribe la devolución.\n"

	integrateContextImpact = `# Delta de contexto: inc-99-ejemplo

## Entrada: context/ARQUITECTURA.md

- Ancla: Componentes > Servicio de pedidos
- Acción: modify
- Motivo: se añade el reintento de pago (RF-9001)

Texto nuevo para la sección (RF-9001).

## Entrada: ../repo-backend/docs/API.md

- Ancla: Errores
- Acción: add
- Motivo: documenta el error del cobro reintentado
`
)

// integrateFixture es un espacio canónico con la baseline del proyecto, el
// contexto técnico mantenido y el cambio inc-99-ejemplo con sus deltas.
type integrateFixture struct {
	ws         workspacetest.Workspace
	changeRoot string
}

func isolateIntegrateGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

func integrateGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

func integrateInitRepo(t *testing.T, dir string) {
	t.Helper()
	integrateGit(t, dir, "init", "-q")
	integrateGit(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
	integrateGit(t, dir, "config", "core.autocrlf", "false")
}

func integrateCommit(t *testing.T, dir, message string) {
	t.Helper()
	integrateGit(t, dir, "add", "-A")
	integrateGit(t, dir, "commit", "-q", "-m", message)
}

func newIntegrateFixture(t *testing.T, withGit bool) *integrateFixture {
	t.Helper()
	if withGit {
		isolateIntegrateGit(t)
	}
	ws := workspacetest.Canonical(t)
	f := &integrateFixture{ws: ws, changeRoot: filepath.Join(ws.Home, "changes", integrateChange)}
	write := func(dir, rel, content string) {
		workspacetest.WriteFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content)
	}
	write(ws.Specs, "axiom.yaml", integrateConfig)
	write(ws.Specs, "01_Requisitos_Funcionales.md", integrateRequisitos)
	write(ws.Specs, "03_Modelo_Datos.md", integrateModelo)
	write(ws.Specs, "04_Flujos_Negocio.md", integrateFlujos)
	write(ws.Specs, "context/ARQUITECTURA.md", integrateContext)
	write(ws.Specs, "referencia/legado/VIEJO.md", "# Viejo\n")
	write(ws.Backend, "docs/API.md", integrateAPI)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", integrateDeltaRequisitos)
	f.writeChange(t, "specs/modelo-de-datos/spec.md", integrateDeltaModelo)
	f.writeChange(t, "specs/flujos-de-negocio/spec.md", integrateDeltaFlujos)
	f.writeChange(t, specbaseline.ContextDeltaFileName, integrateContextImpact)
	if withGit {
		for _, dir := range []string{ws.Specs, ws.Backend} {
			integrateInitRepo(t, dir)
			integrateCommit(t, dir, "base")
		}
	}
	return f
}

func (f *integrateFixture) writeChange(t *testing.T, rel, content string) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(f.changeRoot, filepath.FromSlash(rel)), content)
}

// snapshot devuelve la huella de cada fichero bajo la carpeta maestra, sin los
// directorios .git.
func (f *integrateFixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(f.ws.Master, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(data)
		relative, _ := filepath.Rel(f.ws.Master, path)
		files[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func changedIntegrateFiles(before, after map[string]string) []string {
	var changed []string
	for path, sum := range after {
		if before[path] != sum {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed = append(changed, path+" (borrado)")
		}
	}
	return changed
}

func runIntegrate(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunSDDIntegrate(args, &out)
	return out.String(), err
}

// withIntegrationStatus sustituye la costura con el status por una función que
// parte del status real y lo ajusta.
func withIntegrationStatus(t *testing.T, adjust func(sddstatus.Status) sddstatus.Status) {
	t.Helper()
	previous := resolveIntegrationStatus
	t.Cleanup(func() { resolveIntegrationStatus = previous })
	resolveIntegrationStatus = func(options sddstatus.ResolveOptions) (sddstatus.Status, error) {
		status, err := previous(options)
		if err != nil {
			return status, err
		}
		return adjust(status), nil
	}
}

// withFakeIntegrationStatus sustituye la costura por un status fijo, sin leer el
// disco ni Engram.
func withFakeIntegrationStatus(t *testing.T, status sddstatus.Status) {
	t.Helper()
	previous := resolveIntegrationStatus
	t.Cleanup(func() { resolveIntegrationStatus = previous })
	resolveIntegrationStatus = func(sddstatus.ResolveOptions) (sddstatus.Status, error) { return status, nil }
}

func decodeIntegrateReport(t *testing.T, stdout string) specbaseline.Report {
	t.Helper()
	var report specbaseline.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("la salida no es un informe JSON: %v\n%s", err, stdout)
	}
	return report
}

// REQ-24.42 y REQ-24.54: durante la fase spec, con los deltas sin confirmar, sin
// compuerta aprobada y sin autorizaciones, check informa y no escribe nada.
func TestIntegrateCheckDuringTheSpecPhaseNeedsNoGateNorAuthorization(t *testing.T) {
	f := newIntegrateFixture(t, false)
	// Un kickoff por bloques sin ninguna compuerta aprobada: check no depende de ellas.
	if err := RunSDDKickoff([]string{"seal", "--cwd", f.ws.Specs, "--change", integrateChange,
		"--execution-style", "checkpointed", "--handoff-policy", "none"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("kickoff seal: %v", err)
	}
	before := f.snapshot(t)

	for name, start := range map[string]string{
		"carpeta maestra":                 f.ws.Master,
		"repositorio de especificaciones": f.ws.Specs,
		"repositorio de código":           filepath.Join(f.ws.Backend, "src"),
	} {
		t.Run(name, func(t *testing.T) {
			stdout, err := runIntegrate(t, "check", "--cwd", start, "--change", integrateChange, "--json")
			if err != nil {
				t.Fatalf("integrate check: %v\n%s", err, stdout)
			}
			report := decodeIntegrateReport(t, stdout)
			if report.Schema != "axiom.sdd-integration-check/v1" || report.Change != integrateChange || report.Baseline != "project" {
				t.Fatalf("informe = %+v", report)
			}
			if len(report.Errors) != 0 {
				t.Fatalf("errores inesperados: %v", report.Errors)
			}
			if report.ComposeCommand != "" || len(report.Directed) != 0 || len(report.ContextItems) != 0 {
				t.Errorf("check no emite el comando ni los trabajos de edición: %+v", report)
			}
			if len(report.IDMap) != 1 || report.IDMap[0].Provisional != "RF-9001" || report.IDMap[0].Definitive != "RF-008" {
				t.Errorf("idMap = %+v", report.IDMap)
			}
		})
	}
	if changed := changedIntegrateFiles(before, f.snapshot(t)); len(changed) != 0 {
		t.Fatalf("check no escribe nada, tampoco en la carpeta del cambio: %v", changed)
	}
	if _, err := os.Stat(filepath.Join(f.changeRoot, specbaseline.PlanFileName)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("check no crea %s: %v", specbaseline.PlanFileName, err)
	}
}

func TestIntegrateCheckTextReport(t *testing.T) {
	f := newIntegrateFixture(t, false)
	stdout, err := runIntegrate(t, "check", "--cwd", f.ws.Specs, "--change", integrateChange)
	if err != nil {
		t.Fatalf("integrate check: %v\n%s", err, stdout)
	}
	for _, want := range []string{
		"Integración del cambio inc-99-ejemplo (baseline del proyecto): check",
		"requisitos-funcionales (compose): 01_Requisitos_Funcionales.md",
		"ADDED: RF-9001",
		"MODIFIED: RF-001",
		"familia RF-: máximo RF-007, definitivos RF-008..RF-008, inicio provisional sugerido RF-9001",
		"RF-9001 -> RF-008 (dominio \"requisitos-funcionales\")",
		"Contexto técnico mantenido: presente=true, decisión=entries",
		"C1 modify: context/ARQUITECTURA.md > Componentes > Servicio de pedidos",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("falta %q en el informe:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Errores:") {
		t.Errorf("sin errores no hay sección de errores:\n%s", stdout)
	}
}

// REQ-24.42: un encabezado conflictivo en el cuerpo es un error con código distinto
// de cero, y el informe lo nombra con su línea y la continuación.
func TestIntegrateCheckExitsNonZeroOnAConflictingHeading(t *testing.T) {
	f := newIntegrateFixture(t, false)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", `## ADDED Requirements

### Requirement: RF-9001: Reintento de pago
(Placement: after RF-007)

Texto.

### Subtítulo que cerraría el bloque

Más texto.

## MODIFIED Requirements

### Requirement: RF-001: Alta de producto

Texto.
`)

	stdout, err := runIntegrate(t, "check", "--cwd", f.ws.Specs, "--change", integrateChange, "--json")
	if err == nil {
		t.Fatal("se esperaba un código distinto de cero con un encabezado conflictivo")
	}
	for _, want := range []string{"integrate check", "1 error(es)", "axiom sdd integrate check --cwd"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	report := decodeIntegrateReport(t, stdout)
	if len(report.Errors) != 1 {
		t.Fatalf("errores = %v", report.Errors)
	}
	for _, want := range []string{`dominio "requisitos-funcionales"`, "sección ADDED", "### Subtítulo que cerraría el bloque", "línea 8", "cerraría el bloque", "Continuación"} {
		if !strings.Contains(report.Errors[0], want) {
			t.Errorf("el error del informe no contiene %q: %s", want, report.Errors[0])
		}
	}

	text, err := runIntegrate(t, "check", "--cwd", f.ws.Specs, "--change", integrateChange)
	if err == nil || !strings.Contains(text, "Errores:") || !strings.Contains(text, "### Subtítulo que cerraría el bloque") {
		t.Errorf("el informe de texto debe listar los errores (err = %v):\n%s", err, text)
	}
}

// D-13, paso 1: prepare emite el plan con el comando literal, los trabajos de
// edición y la correspondencia, y solo escribe integration.yaml.
func TestIntegratePrepareEmitsThePlanAndWritesOnlyTheIntegrationFile(t *testing.T) {
	f := newIntegrateFixture(t, true)
	// El repositorio de código hermano está autorizado por cambio (REQ-24.37).
	withIntegrationStatus(t, func(status sddstatus.Status) sddstatus.Status {
		status.ActionContext.AllowedEditRoots = append(append([]string{}, status.ActionContext.AllowedEditRoots...), f.ws.Backend)
		return status
	})
	before := f.snapshot(t)

	stdout, err := runIntegrate(t, "prepare", "--cwd", f.ws.Master, "--change", integrateChange, "--json")
	if err != nil {
		t.Fatalf("integrate prepare: %v\n%s", err, stdout)
	}
	report := decodeIntegrateReport(t, stdout)
	if report.Schema != "axiom.sdd-integration-plan/v1" || report.Change != integrateChange || report.Baseline != "project" {
		t.Fatalf("informe = %+v", report)
	}
	wantCommand := "axiom sdd archive-compose --cwd " + pathquote.Quote(f.ws.Specs) + " --change " + integrateChange + " --document requisitos-funcionales --in-place"
	if report.ComposeCommand != wantCommand {
		t.Errorf("composeCommand = %q\nquería     %q", report.ComposeCommand, wantCommand)
	}
	if len(report.IDMap) != 1 || report.IDMap[0].Definitive != "RF-008" {
		t.Errorf("idMap = %+v", report.IDMap)
	}
	if len(report.Directed) != 1 || report.Directed[0].Domain != "modelo-de-datos" || report.Directed[0].Section != integrateChange {
		t.Fatalf("directed = %+v", report.Directed)
	}
	if text := report.Directed[0].Entries[0].Text; !strings.Contains(text, "RF-008") || strings.Contains(text, "RF-9001") {
		t.Errorf("el texto dirigido debe llevar el ID definitivo: %q", text)
	}
	if len(report.ContextItems) != 2 {
		t.Fatalf("contextItems = %+v", report.ContextItems)
	}
	if first := report.ContextItems[0]; first.ID != "C1" || !strings.Contains(first.Reason, "RF-008") || !strings.Contains(first.Text, "RF-008") || strings.Contains(first.Text, "RF-9001") {
		t.Errorf("el trabajo de contexto debe llevar el ID definitivo: %+v", first)
	}
	if len(report.Register) != 1 || report.Register[0].Domain != "flujos-de-negocio" {
		t.Errorf("register = %+v", report.Register)
	}
	if report.Warnings == nil || report.Errors == nil {
		t.Errorf("warnings y errors son siempre listas: %+v", report)
	}

	changed := changedIntegrateFiles(before, f.snapshot(t))
	want := []string{"repo-specs/openspec/changes/" + integrateChange + "/" + specbaseline.PlanFileName}
	if !reflect.DeepEqual(changed, want) {
		t.Fatalf("prepare solo escribe integration.yaml: cambió %v", changed)
	}
	plan, err := specbaseline.ReadPlan(filepath.Join(f.changeRoot, specbaseline.PlanFileName))
	if err != nil {
		t.Fatalf("el plan escrito no se lee: %v", err)
	}
	if plan.Change != integrateChange || len(plan.Repositories) != 2 {
		t.Errorf("plan = %+v", plan)
	}

	// Una segunda ejecución reemite el mismo plan sin cambiar nada.
	again, err := runIntegrate(t, "prepare", "--cwd", f.ws.Master, "--change", integrateChange, "--json")
	if err != nil {
		t.Fatalf("segunda ejecución de prepare: %v\n%s", err, again)
	}
	reissued := decodeIntegrateReport(t, again)
	if reissued.ComposeCommand != report.ComposeCommand || !reflect.DeepEqual(reissued.Directed, report.Directed) ||
		!reflect.DeepEqual(reissued.ContextItems, report.ContextItems) || !reflect.DeepEqual(reissued.IDMap, report.IDMap) {
		t.Errorf("prepare con un plan vigente debe reemitir el mismo plan:\n%+v\n%+v", report, reissued)
	}
	if changed := changedIntegrateFiles(before, f.snapshot(t)); len(changed) != 1 {
		t.Errorf("la reemisión no escribe nada más: %v", changed)
	}
}

func TestIntegratePrepareTextReportNamesThePlanAndTheCommand(t *testing.T) {
	f := newIntegrateFixture(t, true)
	withIntegrationStatus(t, func(status sddstatus.Status) sddstatus.Status {
		status.ActionContext.AllowedEditRoots = append(append([]string{}, status.ActionContext.AllowedEditRoots...), f.ws.Backend)
		return status
	})
	stdout, err := runIntegrate(t, "prepare", "--cwd", f.ws.Specs, "--change", integrateChange)
	if err != nil {
		t.Fatalf("integrate prepare: %v\n%s", err, stdout)
	}
	for _, want := range []string{
		"Comando de composición:\n  axiom sdd archive-compose --cwd ",
		"--document requisitos-funcionales --in-place",
		"Edición dirigida:",
		"modelo-de-datos",
		"Edición del contexto:",
		"Solo registro:",
		"Plan de integración: " + filepath.Join(f.changeRoot, specbaseline.PlanFileName),
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("falta %q en el informe:\n%s", want, stdout)
		}
	}
}

// REQ-24.37: sin la autorización por cambio del repositorio hermano, que el status
// reporta en allowedEditRoots, prepare rechaza sin escribir y nombra el grant.
func TestIntegratePrepareTakesTheEditRootsFromTheStatus(t *testing.T) {
	f := newIntegrateFixture(t, true)
	before := f.snapshot(t)

	stdout, err := runIntegrate(t, "prepare", "--cwd", f.ws.Specs, "--change", integrateChange, "--json")
	if err == nil {
		t.Fatal("se esperaba un rechazo: el repositorio de código no está autorizado")
	}
	report := decodeIntegrateReport(t, stdout)
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "axiom sdd-attempt grant") || !strings.Contains(report.Errors[0], "allowedEditRoots") {
		t.Fatalf("errores = %v", report.Errors)
	}
	if report.ComposeCommand != "" || len(report.Directed) != 0 {
		t.Errorf("con errores no hay comando ni trabajos: %+v", report)
	}
	if changed := changedIntegrateFiles(before, f.snapshot(t)); len(changed) != 0 {
		t.Fatalf("un rechazo no escribe nada: %v", changed)
	}
}

func TestIntegratePrepareRefusesWithoutGit(t *testing.T) {
	f := newIntegrateFixture(t, false)
	withIntegrationStatus(t, func(status sddstatus.Status) sddstatus.Status {
		status.ActionContext.AllowedEditRoots = append(append([]string{}, status.ActionContext.AllowedEditRoots...), f.ws.Backend)
		return status
	})
	stdout, err := runIntegrate(t, "prepare", "--cwd", f.ws.Specs, "--change", integrateChange, "--json")
	if err == nil {
		t.Fatal("se esperaba un rechazo: los documentos no están en un repositorio git")
	}
	report := decodeIntegrateReport(t, stdout)
	if len(report.Errors) == 0 || !strings.Contains(strings.Join(report.Errors, "\n"), "repositorio git") {
		t.Errorf("errores = %v", report.Errors)
	}
	if _, statErr := os.Stat(filepath.Join(f.changeRoot, specbaseline.PlanFileName)); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("un rechazo no escribe %s: %v", specbaseline.PlanFileName, statErr)
	}
}

func TestIntegrateSupersedeOptionsAreTakenByPrepareOnly(t *testing.T) {
	f := newIntegrateFixture(t, true)
	withIntegrationStatus(t, func(status sddstatus.Status) sddstatus.Status {
		status.ActionContext.AllowedEditRoots = append(append([]string{}, status.ActionContext.AllowedEditRoots...), f.ws.Backend)
		return status
	})
	if _, err := runIntegrate(t, "prepare", "--cwd", f.ws.Specs, "--change", integrateChange, "--supersede", "--superseded-requirements", "RF-007, RF-001", "--json"); err != nil {
		t.Fatalf("integrate prepare: %v", err)
	}
	plan, err := specbaseline.ReadPlan(filepath.Join(f.changeRoot, specbaseline.PlanFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Options.Supersede || !reflect.DeepEqual(plan.Options.SupersededRequirements, []string{"RF-007", "RF-001"}) {
		t.Errorf("opciones del plan = %+v", plan.Options)
	}

	_, err = runIntegrate(t, "check", "--cwd", f.ws.Specs, "--change", integrateChange, "--supersede")
	if err == nil || !strings.Contains(err.Error(), "axiom sdd integrate --help") {
		t.Errorf("check no admite --supersede (err = %v)", err)
	}
}

// REQ-24.62: con la baseline declarada y un almacén sin ficheros, check y prepare
// informan de la causa y salen con código distinto de cero.
func TestIntegrateReportsTheArtifactStoreCause(t *testing.T) {
	f := newIntegrateFixture(t, false)
	for _, store := range []sddstatus.ArtifactStore{sddstatus.ArtifactStoreEngram, sddstatus.ArtifactStoreNone} {
		t.Run(string(store), func(t *testing.T) {
			withFakeIntegrationStatus(t, sddstatus.Status{ArtifactStore: store})
			for _, verb := range []string{"check", "prepare"} {
				stdout, err := runIntegrate(t, verb, "--cwd", f.ws.Specs, "--change", integrateChange, "--json")
				if err == nil {
					t.Fatalf("%s: se esperaba un rechazo con el almacén %s", verb, store)
				}
				report := decodeIntegrateReport(t, stdout)
				joined := strings.Join(report.Errors, "\n")
				for _, want := range []string{"REQ-24.62", string(store), "artifact_store: openspec"} {
					if !strings.Contains(joined, want) {
						t.Errorf("%s: los errores no contienen %q:\n%s", verb, want, joined)
					}
				}
			}
		})
	}
}

func TestIntegrateRefusesAChangeThatIsNotActive(t *testing.T) {
	f := newIntegrateFixture(t, false)

	t.Run("inexistente", func(t *testing.T) {
		withFakeIntegrationStatus(t, sddstatus.Status{ArtifactStore: sddstatus.ArtifactStoreOpenSpec})
		_, err := runIntegrate(t, "check", "--cwd", f.ws.Specs, "--change", "inc-00-no-existe")
		if err == nil {
			t.Fatal("se esperaba un rechazo")
		}
		for _, want := range []string{"inc-00-no-existe", "axiom sdd status --cwd"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el error no contiene %q: %v", want, err)
			}
		}
	})

	t.Run("archivado", func(t *testing.T) {
		withFakeIntegrationStatus(t, sddstatus.Status{ArtifactStore: sddstatus.ArtifactStoreOpenSpec, Archived: &sddstatus.ArchivedProjection{Path: "openspec/changes/archive/2026-10-04-inc-99-ejemplo"}})
		_, err := runIntegrate(t, "prepare", "--cwd", f.ws.Specs, "--change", integrateChange)
		if err == nil || !strings.Contains(err.Error(), "ya está archivado") || !strings.Contains(err.Error(), "axiom sdd status") {
			t.Errorf("el rechazo debe nombrar el archivado y la continuación: %v", err)
		}
	})

	t.Run("no se pudo resolver", func(t *testing.T) {
		previous := resolveIntegrationStatus
		t.Cleanup(func() { resolveIntegrationStatus = previous })
		resolveIntegrationStatus = func(sddstatus.ResolveOptions) (sddstatus.Status, error) {
			return sddstatus.Status{}, errors.New("fallo simulado")
		}
		_, err := runIntegrate(t, "check", "--cwd", f.ws.Specs, "--change", integrateChange)
		if err == nil || !strings.Contains(err.Error(), "fallo simulado") || !strings.Contains(err.Error(), "axiom sdd status") {
			t.Errorf("el rechazo debe envolver la causa y nombrar la continuación: %v", err)
		}
	})
}

func TestIntegrateRefusesAnUnlocatableWorkspace(t *testing.T) {
	_, err := runIntegrate(t, "check", "--cwd", filepath.Join(t.TempDir(), "no-existe"), "--change", integrateChange)
	if err == nil {
		t.Fatal("se esperaba un rechazo de Locate")
	}
}

func TestIntegrateArgumentErrorsNameTheirContinuation(t *testing.T) {
	cases := map[string][]string{
		"sin subcomando":       {},
		"subcomando ajeno":     {"borrar"},
		"sin --change":         {"check", "--cwd", "."},
		"cambio con ruta":      {"prepare", "--change", "../otro"},
		"cambio con separador": {"check", "--change", `a\b`},
		"argumento posicional": {"check", "--change", integrateChange, "sobra"},
		"flag desconocido":     {"prepare", "--change", integrateChange, "--nada"},
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

func TestIntegrateHelpListsCheckAndPrepare(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"check", "-h"}} {
		stdout, err := runIntegrate(t, args...)
		if err != nil {
			t.Fatalf("integrate %v: %v", args, err)
		}
		for _, want := range []string{"Uso: axiom sdd integrate <check|prepare>", "--supersede", "integration.yaml", "archive-compose --change"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("la ayuda no contiene %q:\n%s", want, stdout)
			}
		}
	}
}
