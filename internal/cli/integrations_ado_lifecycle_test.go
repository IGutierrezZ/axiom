package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const adoLifecycleTask = 77

// seedADOTaskProcess siembra los estados del tipo Task y la Task 77 en el estado dado.
func seedADOTaskProcess(e *adoEnv, state string) {
	e.srv.SeedStates("Task",
		adotest.State{Name: "To Do", Category: "Proposed"},
		adotest.State{Name: "Active", Category: "InProgress"},
		adotest.State{Name: "Tech Review", Category: "InProgress"},
		adotest.State{Name: "Done", Category: "Completed"})
	e.srv.SeedWorkItem(adoLifecycleTask, 1, map[string]any{
		"System.WorkItemType": "Task", "System.Title": "cambio / fullstack", "System.State": state,
	})
}

// bindADOTask escribe en integrations.yaml el vínculo de la Task 77 al rol fullstack.
func bindADOTask(t *testing.T, root, changeRoot string) {
	t.Helper()
	link := integrations.Link{Plugin: "azure-devops", Kind: integrations.TaskLinkKind, ID: "77", Role: "fullstack", Source: integrations.TasksBindSource}
	if _, err := integrations.BindTask(integrations.Folder{WorkspaceRoot: root, Root: changeRoot}, "azure-devops", link, time.Now().UTC()); err != nil {
		t.Fatalf("BindTask: %v", err)
	}
}

func adoTaskState(t *testing.T, e *adoEnv) string {
	t.Helper()
	wi, ok := e.srv.WorkItem(adoLifecycleTask)
	if !ok {
		t.Fatal("falta la Task")
	}
	state, _ := wi.Fields["System.State"].(string)
	return state
}

// RF-ADO-14: con el bloque azure-devops activo, `gate record` de un apply aprobado entrega el evento
// al tracker (la Task pasa a Tech Review) y produce el mismo stdout y el mismo código que sin el
// bloque.
func TestGateRecordADORoleApplyDeliversWithoutChangingOutputOrCode(t *testing.T) {
	gate := []string{"--gate", "role-apply:fullstack", "--decision", "approved", "--reason", "ok", "--actor", "Ana"}

	base := newADOEnv(t, "")
	baseRoot := base.root
	sealForGateTest(t, baseRoot, "x")
	baseText, err := gateRecordRun(t, baseRoot, gate...)
	if err != nil {
		t.Fatal(err)
	}
	baseJSON, err := gateRecordRun(t, baseRoot, append(gate, "--json")...)
	if err != nil {
		t.Fatal(err)
	}

	e := newADOEnv(t, adoBlock)
	seedADOTaskProcess(e, "Active")
	changeRoot := filepath.Join(e.root, "openspec", "changes", "x")
	bindADOTask(t, e.root, changeRoot)
	sealForGateTest(t, e.root, "x")
	var warnings bytes.Buffer
	previous := integrationWarnings
	integrationWarnings = &warnings
	t.Cleanup(func() { integrationWarnings = previous })

	text, err := gateRecordRun(t, e.root, gate...)
	if err != nil {
		t.Fatalf("con el bloque activo el verbo sale con éxito: %v", err)
	}
	jsonOut, err := gateRecordRun(t, e.root, append(gate, "--json")...)
	if err != nil {
		t.Fatal(err)
	}

	if text != baseText || jsonOut != baseJSON {
		t.Fatalf("la salida no cambia con el bloque:\n%s\n---\n%s\n%s\n---\n%s", text, baseText, jsonOut, baseJSON)
	}
	if got := adoTaskState(t, e); got != "Tech Review" {
		t.Fatalf("la Task debía pasar a Tech Review, está en %q (avisos: %s)", got, warnings.String())
	}
	if strings.Contains(warnings.String(), adoPAT) {
		t.Fatalf("los avisos no filtran el PAT: %s", warnings.String())
	}
}

// Un fallo del tracker no cambia la salida ni el código de `gate record`: queda un aviso y la
// operación en la cola.
func TestGateRecordADOTrackerFailureKeepsOutputAndExitsZero(t *testing.T) {
	gate := []string{"--gate", "role-apply:fullstack", "--decision", "approved", "--reason", "ok", "--actor", "Ana"}
	base := newADOEnv(t, "")
	baseRoot := base.root
	sealForGateTest(t, baseRoot, "x")
	baseText, err := gateRecordRun(t, baseRoot, gate...)
	if err != nil {
		t.Fatal(err)
	}

	e := newADOEnv(t, adoBlock)
	seedADOTaskProcess(e, "Active")
	changeRoot := filepath.Join(e.root, "openspec", "changes", "x")
	bindADOTask(t, e.root, changeRoot)
	e.srv.Reply("GET", "/contoso/Fabrikam/_apis/wit/workitems/77", adotest.Response{Status: 500, Body: "caído"})
	sealForGateTest(t, e.root, "x")
	warnings := captureIntegrationWarnings(t)

	text, err := gateRecordRun(t, e.root, gate...)

	if err != nil || text != baseText {
		t.Fatalf("salida = %q (%v), esperaba %q", text, err, baseText)
	}
	if !strings.Contains(warnings.String(), "axiom integrations sync") {
		t.Fatalf("el fallo avisa con la continuación: %q", warnings)
	}
}

// `sync` muestra los avisos de entrega de las operaciones entregadas y los abandonos tipados.
func TestIntegrationsSyncADOShowsDeliveryWarningsAndTypedAbandonments(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	seedADOTaskProcess(e, "Tech Review")
	changeRoot := filepath.Join(e.root, "openspec", "changes", "x")
	captureIntegrationWarnings(t)
	loc := locateIntegrations(t, e.root)

	// role.start sin vínculo: la entrega queda abandonada con el error tipado y su salida.
	emitRoleStartEvent(loc, changeRoot, "fullstack")
	code, _, errOut := runIntegrationsCLI(t, "", "sync", "--cwd", e.root)
	if code != 1 || !strings.Contains(errOut, "abandoned") || !strings.Contains(errOut, "task_not_bound") ||
		!strings.Contains(errOut, "axiom integrations sync --retry-abandoned") {
		t.Fatalf("abandono tipado: código %d, stderr %q", code, errOut)
	}

	// Con el vínculo, la reactivación entrega y muestra los avisos de la entrega.
	bindADOTask(t, e.root, changeRoot)
	code, out, errOut := runIntegrationsCLI(t, "", "sync", "--cwd", e.root, "--retry-abandoned")
	if code != 0 {
		t.Fatalf("sync --retry-abandoned: código %d, stdout %q, stderr %q", code, out, errOut)
	}
	for _, want := range []string{"entregadas 1", "Aviso: task_state_kept: ", "Aviso: assignment_skipped: "} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout debe contener %q:\n%s", want, out)
		}
	}
	if got := adoTaskState(t, e); got != "Tech Review" {
		t.Fatalf("la Task no retrocede: %q", got)
	}
}
