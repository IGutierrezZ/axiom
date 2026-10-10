package specbaseline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Las pruebas de ApplyCompose y de la reaplicación idempotente (inc-24, REQ-24.43,
// REQ-24.46, D-13 paso 2). Todos los textos son inventados.

const (
	applyConfigTwoCompose = `workspace:
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
    - domain: reglas-de-negocio
      path: 05_Reglas_Negocio.md
      requirement:
        heading: '^#+ (?P<id>RN-\d+): (?P<title>.+)$'
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
	applyReglas      = "# Reglas de negocio\n\n## Precios\n\n### RN-001: Descuento por volumen\n\nSe aplica un descuento por volumen.\n"
	applyDeltaReglas = "## MODIFIED Requirements\n\n### Requirement: RN-001: Descuento por volumen\n\nSe aplica un descuento por volumen y por fidelidad.\n"
)

const (
	docRequisitos = "01_Requisitos_Funcionales.md"
	docReglas     = "05_Reglas_Negocio.md"
)

// preparedFixture monta el espacio con git, prepara el plan y devuelve el objetivo.
func preparedFixture(t *testing.T) (*integrationFixture, ChangeTarget, Plan) {
	t.Helper()
	f := newIntegrationFixture(t, true)
	target := f.target(t)
	plan, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	mustNoErrors(t, report)
	return f, target, plan
}

// preparedTwoCompose es como preparedFixture con dos documentos de composición.
func preparedTwoCompose(t *testing.T) (*integrationFixture, ChangeTarget, Plan) {
	t.Helper()
	f := newIntegrationFixture(t, true)
	writeRepoFile(t, f.specs, "axiom.yaml", applyConfigTwoCompose)
	writeRepoFile(t, f.specs, docReglas, applyReglas)
	f.writeChange(t, "specs/reglas-de-negocio/spec.md", applyDeltaReglas)
	commitAll(t, f.specs, "segundo documento de composición")
	target := f.target(t)
	plan, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	mustNoErrors(t, report)
	return f, target, plan
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func leftoverTemps(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var temps []string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			temps = append(temps, entry.Name())
		}
	}
	return temps
}

func TestApplyComposeWritesTheCompositionAndMarksThePlan(t *testing.T) {
	f, target, plan := preparedFixture(t)
	before := snapshot(t, f.master)

	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatalf("ApplyCompose: %v", err)
	}
	mustNoErrors(t, report)
	if report.Schema != ApplySchema || report.Change != "inc-99-ejemplo" {
		t.Errorf("informe = %+v", report)
	}

	written := readText(t, filepath.Join(f.specs, docRequisitos))
	if sha256Hex([]byte(written)) != plan.Documents[0].AfterSHA256 {
		t.Errorf("el resultado debe coincidir con la huella after del plan:\n%s", written)
	}
	if !strings.Contains(written, "### RF-008: Reintento de pago") || strings.Contains(written, "RF-9001") || !strings.Contains(written, "Véase RF-008.") {
		t.Errorf("el documento debe llevar el ID definitivo y ningún provisional:\n%s", written)
	}

	changed := changedFiles(before, snapshot(t, f.master))
	sort.Strings(changed)
	want := []string{"repo-specs/" + docRequisitos, "repo-specs/openspec/changes/inc-99-ejemplo/" + PlanFileName}
	if !reflect.DeepEqual(changed, want) {
		t.Fatalf("solo cambian el documento de composición y el plan: %v", changed)
	}

	after, err := ReadPlan(filepath.Join(f.changeRoot, PlanFileName))
	if err != nil {
		t.Fatalf("el plan debe seguir siendo válido: %v", err)
	}
	for _, document := range after.Documents {
		wantState := TraceState("")
		if document.Integration == workspace.IntegrationCompose {
			wantState = StateIntegrated
		}
		if document.State != wantState {
			t.Errorf("estado del documento %s = %q, quería %q", document.Domain, document.State, wantState)
		}
	}
	for _, entry := range after.Entries {
		wantState := StatePending
		if entry.Kind == EntryCompose {
			wantState = StateIntegrated
		}
		if entry.State != wantState {
			t.Errorf("estado de la entrada %s (%s) = %q, quería %q", entry.ID, entry.Kind, entry.State, wantState)
		}
	}
	if after.InputsSHA256 != plan.InputsSHA256 || !reflect.DeepEqual(after.Repositories, plan.Repositories) || after.VerifiedAt != nil {
		t.Errorf("la integración solo marca el estado: antes %+v, después %+v", plan, after)
	}
	if temps := leftoverTemps(t, f.specs); len(temps) != 0 {
		t.Errorf("no deben quedar temporales: %v", temps)
	}
}

func TestApplyComposeWritesEveryDocumentOfSeveralDomains(t *testing.T) {
	f, target, plan := preparedTwoCompose(t)
	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales", "reglas-de-negocio"})
	if err != nil {
		t.Fatalf("ApplyCompose: %v", err)
	}
	mustNoErrors(t, report)
	for _, document := range plan.Documents {
		if document.Integration != workspace.IntegrationCompose {
			continue
		}
		written := readText(t, filepath.Join(f.specs, filepath.FromSlash(document.Path)))
		if sha256Hex([]byte(written)) != document.AfterSHA256 {
			t.Errorf("el documento %s no coincide con su huella after", document.Path)
		}
	}
	if got := readText(t, filepath.Join(f.specs, docReglas)); !strings.Contains(got, "por fidelidad") {
		t.Errorf("el segundo documento debe estar compuesto:\n%s", got)
	}
	after, _ := ReadPlan(filepath.Join(f.changeRoot, PlanFileName))
	integrated := 0
	for _, document := range after.Documents {
		if document.State == StateIntegrated {
			integrated++
		}
	}
	if integrated != 2 {
		t.Errorf("ambas composiciones quedan integradas en el plan: %d", integrated)
	}
}

func TestApplyComposeTakesTheSupersedeOptionsFromThePlan(t *testing.T) {
	f := newIntegrationFixture(t, true)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", "## REMOVED Requirements\n\n### Requirement: RF-007: Consulta de producto\n\n(Reason: sustituido por la consulta unificada)\n")
	commitAll(t, f.specs, "delta de eliminación")
	target := f.target(t)
	if _, report, err := Prepare(context.Background(), target, PlanOptions{Supersede: true}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	written := readText(t, filepath.Join(f.specs, docRequisitos))
	if !strings.Contains(written, "### RF-007: Consulta de producto [SUPERSEDED / DEPRECADO]") || !strings.Contains(written, "> Motivo: sustituido por la consulta unificada") {
		t.Errorf("--supersede se toma del plan:\n%s", written)
	}
}

func TestApplyComposeRequiresThePlan(t *testing.T) {
	f := newIntegrationFixture(t, true)
	before := snapshot(t, f.master)
	report, err := ApplyCompose(context.Background(), f.target(t), []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "no tiene plan de integración", PlanFileName, "axiom sdd integrate prepare --cwd")
	if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("un rechazo no escribe nada: %v", changed)
	}
}

func TestApplyComposeRequiresTheProjectBaseline(t *testing.T) {
	f := newIntegrationFixture(t, false)
	writeRepoFile(t, f.specs, "axiom.yaml", "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n")
	report, err := ApplyCompose(context.Background(), f.target(t), []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "no tiene baseline del proyecto", "axiom sdd archive-compose --canonical")
}

func TestApplyComposeReportsTheArtifactStoreCause(t *testing.T) {
	f := newIntegrationFixture(t, false)
	target := f.target(t)
	target.ArtifactStore = "engram"
	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "REQ-24.62", `"engram"`)
}

// D-13, paso 2: la lista de --document coincide exactamente con los dominios de
// composición del plan.
func TestApplyComposeDomainListMustMatchThePlanExactly(t *testing.T) {
	f, target, plan := preparedTwoCompose(t)
	command := composeCommand(target, plan.Documents)
	before := snapshot(t, f.master)

	tests := []struct {
		name    string
		domains []string
		want    []string
	}{
		{name: "falta un dominio", domains: []string{"requisitos-funcionales"}, want: []string{"faltan reglas-de-negocio", "exactamente el comando", command}},
		{name: "sobra un dominio de prosa", domains: []string{"requisitos-funcionales", "reglas-de-negocio", "modelo-de-datos"}, want: []string{`"modelo-de-datos"`, "es de prosa", "edición dirigida o solo registro"}},
		{name: "dominio de solo registro", domains: []string{"requisitos-funcionales", "reglas-de-negocio", "flujos-de-negocio"}, want: []string{`"flujos-de-negocio"`, "es de prosa", "(register)"}},
		{name: "dominio no declarado", domains: []string{"requisitos-funcionales", "reglas-de-negocio", "nuevo-dominio"}, want: []string{`"nuevo-dominio"`, "'baseline.documents'", "declara el dominio", "retira la delta"}},
		{name: "dominio repetido", domains: []string{"requisitos-funcionales", "requisitos-funcionales", "reglas-de-negocio"}, want: []string{`"requisitos-funcionales" figura repetido`}},
		{name: "ninguno", domains: nil, want: []string{"faltan requisitos-funcionales, reglas-de-negocio"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := ApplyCompose(context.Background(), target, tt.domains)
			if err != nil {
				t.Fatal(err)
			}
			mustErrors(t, report, tt.want...)
			if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
				t.Fatalf("un rechazo no escribe nada: %v", changed)
			}
		})
	}
}

func TestApplyComposeRejectsAnExtraComposeDomainThatTheDeltaDoesNotHave(t *testing.T) {
	f, _, _ := preparedTwoCompose(t)
	// reglas-de-negocio está declarado con perfil, pero el plan no lo incluye si el
	// cambio no tiene su delta.
	f.removeChange(t, "specs/reglas-de-negocio")
	commitAll(t, f.specs, "sin delta de reglas")
	if err := os.Remove(filepath.Join(f.changeRoot, PlanFileName)); err != nil {
		t.Fatal(err)
	}
	target := f.target(t)
	if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales", "reglas-de-negocio"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "sobran reglas-de-negocio")
}

// REQ-24.43: reaplicar un documento ya integrado se rechaza, nombra verify y no
// modifica nada.
func TestApplyComposeRefusesAnAlreadyIntegratedDocument(t *testing.T) {
	f, target, _ := preparedFixture(t)
	if report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	after := snapshot(t, f.master)

	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "ya integrada", `"requisitos-funcionales"`, "RF-9001 -> RF-008", "axiom sdd integrate verify")
	if changed := changedFiles(after, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("reaplicar no cambia nada: %v", changed)
	}
}

// Un documento que ya coincide con su huella after se reconoce como integrado
// aunque el plan siga pendiente (por ejemplo, tras una interrupción).
func TestApplyComposeRecognizesTheAfterFingerprintWhileThePlanIsPending(t *testing.T) {
	f, target, plan := preparedFixture(t)
	if report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	composed := readText(t, filepath.Join(f.specs, docRequisitos))
	// El plan vuelve a estar pendiente: como si la escritura se hubiese interrumpido tras renombrar.
	if err := WritePlan(filepath.Join(f.changeRoot, PlanFileName), plan); err != nil {
		t.Fatal(err)
	}

	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "ya integrada", "axiom sdd integrate verify")
	if got := readText(t, filepath.Join(f.specs, docRequisitos)); got != composed {
		t.Errorf("el documento no debe cambiar")
	}
}

func TestAlreadyIntegratedHelper(t *testing.T) {
	original := []byte("original")
	composed := []byte("compuesto")
	plan := Plan{Change: "inc-99-ejemplo", IDMap: []IDAssignment{{Provisional: "RF-9001", Definitive: "RF-008", Domain: "requisitos"}}, Documents: []PlanDocument{
		{Domain: "requisitos", Integration: workspace.IntegrationCompose, BeforeSHA256: sha256Hex(original), AfterSHA256: sha256Hex(composed), State: StatePending},
		{Domain: "noop", Integration: workspace.IntegrationCompose, BeforeSHA256: sha256Hex(original), AfterSHA256: sha256Hex(original), State: StatePending},
		{Domain: "prosa", Integration: workspace.IntegrationDirected, BeforeSHA256: sha256Hex(original)},
	}}
	tests := []struct {
		name    string
		domain  string
		current []byte
		want    bool
	}{
		{"pendiente y en before", "requisitos", original, false},
		{"pendiente pero en after", "requisitos", composed, true},
		{"composición que no cambia el documento", "noop", original, false},
		{"dominio de prosa", "prosa", original, false},
		{"dominio ajeno", "otro", composed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AlreadyIntegrated(plan, tt.domain, tt.current)
			if (err != nil) != tt.want {
				t.Fatalf("AlreadyIntegrated = %v, quería rechazo = %t", err, tt.want)
			}
			if tt.want {
				var done *AlreadyIntegratedError
				if !errors.As(err, &done) || done.Domain != tt.domain || !strings.Contains(err.Error(), "RF-9001 -> RF-008") {
					t.Errorf("el rechazo debe nombrar el dominio y la correspondencia: %v", err)
				}
			}
		})
	}

	plan.Documents[0].State = StateIntegrated
	if err := AlreadyIntegrated(plan, "requisitos", original); err == nil {
		t.Error("un documento que el plan marca como integrado se rechaza aunque sus bytes difieran")
	}
	// Un plan integrado nombra el dominio, no «todas las composiciones».
	if message := AlreadyIntegrated(plan, "requisitos", original).Error(); strings.Contains(message, "todas sus composiciones") {
		t.Errorf("el rechazo por dominio no habla de todas las composiciones: %s", message)
	}
}

// R35: si los documentos o las entradas cambiaron desde prepare, se rechaza con la
// continuación y no se escribe ninguno.
func TestApplyComposeRefusesWhenTheDocumentsOrInputsChangedSincePrepare(t *testing.T) {
	t.Run("un documento cambió", func(t *testing.T) {
		f, target, _ := preparedTwoCompose(t)
		writeRepoFile(t, f.specs, docReglas, applyReglas+"\nLínea añadida a mano.\n")
		before := snapshot(t, f.master)
		report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales", "reglas-de-negocio"})
		if err != nil {
			t.Fatal(err)
		}
		mustErrors(t, report, "ya no corresponde", "git checkout -- ", docReglas, "axiom sdd integrate prepare")
		if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
			t.Fatalf("ningún documento se escribe si uno no está en su huella before (tampoco el primero): %v", changed)
		}
	})

	t.Run("un delta cambió", func(t *testing.T) {
		f, target, _ := preparedFixture(t)
		f.writeChange(t, "specs/requisitos-funcionales/spec.md", strings.Replace(fixtureDeltaRequisitos, "tres veces", "cuatro veces", 1))
		before := snapshot(t, f.master)
		report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
		if err != nil {
			t.Fatal(err)
		}
		mustErrors(t, report, "ya no corresponde", "volver a preparar")
		if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
			t.Fatalf("un rechazo no escribe nada: %v", changed)
		}
	})
}

func TestApplyComposeRefusesAnAfterFingerprintThatNoLongerMatches(t *testing.T) {
	f, target, plan := preparedFixture(t)
	plan.Documents[0].AfterSHA256 = sha256Hex([]byte("otro resultado"))
	if err := WritePlan(filepath.Join(f.changeRoot, PlanFileName), plan); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, f.master)
	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "distinto del que registró", "huella after", "axiom sdd integrate prepare")
	if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("un rechazo no escribe nada: %v", changed)
	}
}

// failRenames sustituye el renombrado para que falle según fail(n), con n el orden
// de la llamada (desde 1), y devuelve el contador de llamadas.
func failRenames(t *testing.T, fail func(n int) bool) *atomic.Int32 {
	t.Helper()
	previous := renameFile
	t.Cleanup(func() { renameFile = previous })
	var calls atomic.Int32
	renameFile = func(oldPath, newPath string) error {
		n := int(calls.Add(1))
		if fail(n) {
			return errors.New("renombrado simulado")
		}
		return previous(oldPath, newPath)
	}
	return &calls
}

// REQ-24.46: si un renombrado falla se restauran desde memoria los ya renombrados
// y ningún documento queda modificado.
func TestApplyComposeIsAllOrNothingWhenARenameFails(t *testing.T) {
	f, target, _ := preparedTwoCompose(t)
	original := snapshot(t, f.master)
	calls := failRenames(t, func(n int) bool { return n == 2 }) // el segundo documento

	_, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales", "reglas-de-negocio"})
	if err == nil {
		t.Fatal("se esperaba el error de escritura")
	}
	for _, want := range []string{docReglas, "renombrado simulado", "se han restaurado", "no se ha modificado ninguno", "axiom sdd integrate prepare"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	if calls.Load() < 3 {
		t.Errorf("la restauración del primer documento también renombra: %d llamadas", calls.Load())
	}
	if changed := changedFiles(original, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("ningún documento ni el plan quedan modificados: %v", changed)
	}
	if temps := leftoverTemps(t, f.specs); len(temps) != 0 {
		t.Errorf("no deben quedar temporales: %v", temps)
	}
}

func TestApplyComposeReportsADocumentLeftModifiedWhenTheRestoreFails(t *testing.T) {
	f, target, _ := preparedTwoCompose(t)
	failRenames(t, func(n int) bool { return n >= 2 }) // también la restauración

	_, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales", "reglas-de-negocio"})
	if err == nil {
		t.Fatal("se esperaba el error de escritura")
	}
	for _, want := range []string{"la restauración", "falló", "git checkout -- " + docRequisitos} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	if got := readText(t, filepath.Join(f.specs, docRequisitos)); !strings.Contains(got, "RF-008") {
		t.Errorf("el primer documento ya renombrado queda compuesto, y el error lo dice")
	}
	if got := readText(t, filepath.Join(f.specs, docReglas)); got != applyReglas {
		t.Errorf("el segundo documento no debe cambiar")
	}
	if temps := leftoverTemps(t, f.specs); len(temps) != 0 {
		t.Errorf("no deben quedar temporales del segundo documento: %v", temps)
	}
}

func TestApplyComposeRestoresTheDocumentsWhenThePlanCannotBeMarked(t *testing.T) {
	f, target, _ := preparedTwoCompose(t)
	original := snapshot(t, f.master)
	failRenames(t, func(n int) bool { return n == 3 }) // el plan, tras los dos documentos

	_, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales", "reglas-de-negocio"})
	if err == nil {
		t.Fatal("se esperaba el error de escritura del plan")
	}
	for _, want := range []string{PlanFileName, "restaurado", "nada ha cambiado"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	if changed := changedFiles(original, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("sin la marca en el plan no queda ningún documento modificado: %v", changed)
	}
}

// Todos los temporales se crean antes de renombrar el primero: un fallo al crear el
// segundo no deja ningún documento modificado ni temporales.
func TestApplyComposeStopsBeforeRenamingWhenATempCannotBeCreated(t *testing.T) {
	f, target, _ := preparedTwoCompose(t)
	original := snapshot(t, f.master)
	previous := createTemp
	t.Cleanup(func() { createTemp = previous })
	var calls int
	createTemp = func(dir, pattern string) (*os.File, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("disco lleno simulado")
		}
		return previous(dir, pattern)
	}

	_, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales", "reglas-de-negocio"})
	if err == nil {
		t.Fatal("se esperaba el error de preparación")
	}
	for _, want := range []string{docReglas, "disco lleno simulado", "no se ha modificado ningún documento", "axiom sdd integrate prepare"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	if changed := changedFiles(original, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("ningún documento queda modificado: %v", changed)
	}
	if temps := leftoverTemps(t, f.specs); len(temps) != 0 {
		t.Errorf("el temporal del primer documento se retira: %v", temps)
	}
}

func TestApplyComposeKeepsASymlinkedDocumentAndWritesItsTarget(t *testing.T) {
	f := newIntegrationFixture(t, true)
	real := filepath.Join(f.specs, "real", docRequisitos)
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(f.specs, docRequisitos), real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("real", docRequisitos), filepath.Join(f.specs, docRequisitos)); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos: %v", err)
	}
	commitAll(t, f.specs, "documento como enlace simbólico")
	target := f.target(t)
	if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	info, err := os.Lstat(filepath.Join(f.specs, docRequisitos))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("el enlace simbólico debe conservarse: %v %v", info, err)
	}
	if got := readText(t, real); !strings.Contains(got, "RF-008") {
		t.Errorf("el destino del enlace recibe la composición:\n%s", got)
	}
}

func TestApplyComposeRefusesASymlinkedDocumentOutsideTheMasterFolder(t *testing.T) {
	f, target, _ := preparedFixture(t)
	outside := t.TempDir()
	outsideDoc := filepath.Join(outside, docRequisitos)
	if err := os.WriteFile(outsideDoc, []byte(fixtureRequisitos), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.specs, docRequisitos)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDoc, filepath.Join(f.specs, docRequisitos)); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos: %v", err)
	}
	report, err := ApplyCompose(context.Background(), target, []string{"requisitos-funcionales"})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "fuera de la carpeta maestra", "enlace simbólico")
	if got := readText(t, outsideDoc); got != fixtureRequisitos {
		t.Errorf("el documento de fuera de la carpeta maestra no debe cambiar")
	}
}
