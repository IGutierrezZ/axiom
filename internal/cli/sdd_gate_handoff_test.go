package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de la salida 2 de una parada, `axiom sdd gate record --hand-off` (REQ-26.1, REQ-26.3 a
// REQ-26.6; DD-02, DD-07, DD-08). Con git real, entorno aislado y sin red, sobre el punto de
// entrada en proceso `RunSDDGate`.

var (
	handOffGatesRel = "openspec/changes/x/gates.yaml"
	continuousArgs  = []string{"--execution-style", "continuous", "--handoff-policy", "none"}
)

// handOffProject es un monorepo embebido con el cambio x sellado `checkpointed` con los roles core
// y qa, en la rama `feat/x` y con sus artefactos confirmados: la forma en la que se presenta una
// parada.
func handOffProject(t *testing.T, sealArgs []string) workspacetest.Workspace {
	t.Helper()
	ws := guardProjectSealedWith(t, sealArgs)
	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	changeRoot := guardChangeRoot(ws)
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "spec.md"), "# spec\n")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "design.md"), "# diseño\n")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "tasks.core.md"), "- [x] 1.1 hecho\n")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "tasks.qa.md"), "- [x] 1.1 hecho\n")
	integrateCommit(t, ws.Master, "artefactos del cambio")
	return ws
}

func handOffLedger(t *testing.T, ws workspacetest.Workspace) kickoff.GateLedger {
	t.Helper()
	ledger, err := kickoff.LoadGates(guardChangeRoot(ws))
	if err != nil {
		t.Fatalf("gates.yaml: %v", err)
	}
	return ledger
}

// wantRefusedWithoutWriting comprueba que la salida 2 se rechazó con la continuación esperada y sin
// dejar ningún rastro en la carpeta del cambio (REQ-26.3, REQ-26.4, REQ-26.6).
func wantRefusedWithoutWriting(t *testing.T, ws workspacetest.Workspace, args []string, wantFragments ...string) {
	t.Helper()
	before := consentCLISnapshot(t, guardChangeRoot(ws))
	out, err := gateRecord(t, ws, args...)
	if err == nil {
		t.Fatalf("la salida 2 debía rechazarse; salida:\n%s", out)
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("el rechazo %q debe contener %q", err.Error(), fragment)
		}
	}
	if !strings.Contains(err.Error(), "axiom ") && !strings.Contains(err.Error(), "git -C") {
		t.Errorf("el rechazo %q debe llevar una continuación ejecutable", err.Error())
	}
	wantNoWrite(t, ws, before)
	if out != "" {
		t.Errorf("un rechazo no imprime la confirmación de la compuerta: %q", out)
	}
}

// REQ-26.3, REQ-26.5 y REQ-27.12: la salida 2 en un monorepo escribe `approved` con la marca y
// devuelve la orden de publicación atada a la rama, el destino de su PR y la reanudación (`git
// fetch origin` y `axiom branch ensure`), sin exigir una PR fusionada ni órdenes de commit o push.
func TestGateRecordHandOffInAMonorepoWritesTheMarkAndReturnsTheDelivery(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	out, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--hand-off")

	if err != nil {
		t.Fatalf("gate record --hand-off: %v", err)
	}
	for _, want := range []string{
		`Compuerta "spec" registrada (approved, fase siguiente cedida: design) para "x".`,
		"axiom sdd publish --cwd ",
		" --change x --branch feat/x",
		"rama feat/x en origin; destino de su PR: main",
		"`git fetch origin` y `axiom branch ensure --change x --json`",
		"no hace falta una PR fusionada",
		"Axiom no ejecuta ninguna orden de git ni de plataforma",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{" add -- ", " commit -m ", " push ", "fusionada en", "no recupera trabajo", "Aviso:"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("la cesión ya no entrega órdenes de documentación ni exige fusionar; la salida no puede contener %q:\n%s", forbidden, out)
		}
	}
	for _, forbidden := range []string{"&&", "gh ", "glab ", " -a "} {
		if strings.Contains(out, forbidden) {
			t.Errorf("la salida no puede contener %q:\n%s", forbidden, out)
		}
	}
	ledger := handOffLedger(t, ws)
	if len(ledger.Records) != 1 || !ledger.Records[0].HandedOff || ledger.Records[0].Decision != kickoff.DecisionApproved || ledger.Records[0].ArtifactDigest == "" {
		t.Fatalf("registros = %+v, se esperaba un approved con la marca y el digest", ledger.Records)
	}
	data, readErr := os.ReadFile(filepath.Join(guardChangeRoot(ws), "gates.yaml"))
	if readErr != nil || !strings.Contains(string(data), "handed_off: true") {
		t.Errorf("gates.yaml debe llevar handed_off: true (%v):\n%s", readErr, data)
	}
}

func TestGateRecordHandOffReasonIsOptionalAndKept(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	if _, err := gateRecord(t, ws, "--gate", "design", "--decision", "approved", "--hand-off", "--reason", "diseño conforme"); err != nil {
		t.Fatalf("gate record --hand-off con motivo: %v", err)
	}

	ledger := handOffLedger(t, ws)
	if len(ledger.Records) != 1 || ledger.Records[0].Reason != "diseño conforme" || !ledger.Records[0].HandedOff {
		t.Fatalf("registros = %+v", ledger.Records)
	}
}

// Matriz «Estado del push» y «Estado del commit»: el observador del único punto que ejecuta git no
// ve push, fetch, pull, ls-remote ni commit durante toda la orden.
func TestGateRecordHandOffNeverRunsRemoteGitOrCommit(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)
	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) { seen = append(seen, append([]string{dir}, args...)) }
	t.Cleanup(func() { gitcommon.Observe = nil })

	if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--hand-off"); err != nil {
		t.Fatalf("gate record --hand-off: %v", err)
	}

	if len(seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range seen {
		switch invocation[1] {
		case "push", "fetch", "pull", "ls-remote", "commit", "add":
			t.Errorf("gate record --hand-off lanzó %v", invocation)
		}
	}
}

// REQ-26.4, último escenario: la guarda `wrong_branch` se aplica antes que la comprobación de
// artefactos, aunque estos también estén sin confirmar.
func TestGateRecordHandOffReportsWrongBranchBeforeUncommittedArtifacts(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)
	integrateGit(t, ws.Master, "switch", "-q", "main")
	workspacetest.WriteFile(t, filepath.Join(guardChangeRoot(ws), "spec.md"), "# spec editado\n")
	before := consentCLISnapshot(t, guardChangeRoot(ws))

	_, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--hand-off")

	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
	wantNoWrite(t, ws, before)
}

// REQ-26.1 (modalidad continua), REQ-26.6 (almacén sin ficheros), REQ-26.4 (artefacto ausente o sin
// confirmar) y REQ-26.3 (puerta ya resuelta): cada rechazo lleva una continuación ejecutable y deja
// la carpeta del cambio intacta.
func TestGateRecordHandOffRefusals(t *testing.T) {
	handOff := []string{"--decision", "approved", "--hand-off"}
	withGate := func(gate string) []string { return append([]string{"--gate", gate}, handOff...) }

	t.Run("un cambio continuo no tiene paradas", func(t *testing.T) {
		ws := handOffProject(t, continuousArgs)
		wantRefusedWithoutWriting(t, ws, withGate("spec"), "no tiene paradas", "--decision approved")
		if strings.Contains(consentCLISnapshot(t, guardChangeRoot(ws))[filepath.Join(guardChangeRoot(ws), "gates.yaml")], "handed_off") {
			t.Error("no se escribe gates.yaml")
		}
	})

	for _, store := range []string{"engram", "none"} {
		t.Run("el almacén "+store+" no tiene ficheros que ceder", func(t *testing.T) {
			ws := handOffProject(t, guardSealArgs)
			workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), "artifact_store: "+store+"\n")
			integrateCommit(t, ws.Master, "almacén")
			wantRefusedWithoutWriting(t, ws, withGate("spec"), `"`+store+`"`, "--decision rejected", "--decision approved")
		})
	}

	t.Run("un almacén hybrid sí permite la salida 2", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), "artifact_store: hybrid\n")
		integrateCommit(t, ws.Master, "almacén")
		if _, err := gateRecord(t, ws, withGate("spec")...); err != nil {
			t.Fatalf("hybrid debe permitir la salida 2: %v", err)
		}
	})

	t.Run("un artefacto ausente no se cede", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		if err := os.Remove(filepath.Join(guardChangeRoot(ws), "design.md")); err != nil {
			t.Fatal(err)
		}
		integrateCommit(t, ws.Master, "sin diseño")
		wantRefusedWithoutWriting(t, ws, withGate("design"), "su artefacto no existe o está vacío", "axiom sdd status")
	})

	t.Run("una puerta ya aprobada con la salida 3 no admite la 2", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved"); err != nil {
			t.Fatalf("salida 3: %v", err)
		}
		wantRefusedWithoutWriting(t, ws, withGate("spec"), "ya está resuelta", "axiom sdd status")
		if ledger := handOffLedger(t, ws); len(ledger.Records) != 1 || ledger.Records[0].HandedOff {
			t.Errorf("registros = %+v, se esperaba solo el approved sin marca", ledger.Records)
		}
	})

	// REQ-26.4 y DD-12: la cesión ya no comprueba qué está confirmado. Los documentos pendientes los
	// confirma `axiom sdd publish`, y `uncommitted_artifacts` deja de existir.
	t.Run("un artefacto modificado sin confirmar ya no se rechaza", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		workspacetest.WriteFile(t, filepath.Join(guardChangeRoot(ws), "spec.md"), "# spec editado\n")
		out, err := gateRecord(t, ws, withGate("spec")...)
		if err != nil {
			t.Fatalf("la salida 2 no comprueba lo confirmado: %v", err)
		}
		if strings.Contains(out, "uncommitted_artifacts") || !strings.Contains(out, "axiom sdd publish") {
			t.Errorf("la cesión debe entregar la orden de publicación:\n%s", out)
		}
	})

	t.Run("un fichero sin seguimiento ya no se rechaza", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		workspacetest.WriteFile(t, filepath.Join(guardChangeRoot(ws), "specs", "demo", "spec.md"), "# delta\n")
		if _, err := gateRecord(t, ws, withGate("spec")...); err != nil {
			t.Fatalf("la salida 2 no comprueba lo confirmado: %v", err)
		}
	})

	t.Run("el propio gates.yaml sin confirmar no cuenta", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved"); err != nil {
			t.Fatalf("salida 3 previa: %v", err)
		}
		if _, err := gateRecord(t, ws, withGate("design")...); err != nil {
			t.Fatalf("gates.yaml sin confirmar no impide la salida 2: %v", err)
		}
	})

	t.Run("integration no admite la salida 2 y no escribe", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		wantRefusedWithoutWriting(t, ws, withGate("integration"), "no está disponible para la compuerta integration")
	})

	t.Run("--hand-off con rejected se rechaza", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "spec", "--decision", "rejected", "--reason", "x", "--hand-off"}, "solo acompaña a `--decision approved`")
	})
}

// REQ-26.3, REQ-21.13 (cuarto escenario) y DD-06: en role-apply la fase cedida es `apply:<roles que
// faltan>` y, con el último rol, `verify` con el aviso de último rol.
func TestGateRecordHandOffOfARoleApplyNamesWhatIsHandedOver(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	out, err := gateRecord(t, ws, "--gate", "role-apply:core", "--decision", "approved", "--hand-off")
	if err != nil {
		t.Fatalf("role-apply:core --hand-off: %v", err)
	}
	if !strings.Contains(out, "fase siguiente cedida: apply:qa") {
		t.Errorf("la fase cedida es apply:qa:\n%s", out)
	}
	if strings.Contains(out, lastRoleNoticeMarker) {
		t.Errorf("con un rol pendiente no hay aviso de último rol:\n%s", out)
	}

	out, err = gateRecord(t, ws, "--gate", "role-apply:qa", "--decision", "approved", "--hand-off")
	if err != nil {
		t.Fatalf("role-apply:qa --hand-off: %v", err)
	}
	if !strings.Contains(out, "fase siguiente cedida: verify") {
		t.Errorf("con el último rol se cede verify:\n%s", out)
	}
	if strings.Count(out, lastRoleNoticeMarker) != 1 {
		t.Errorf("la salida 2 del último rol emite el aviso exactamente una vez:\n%s", out)
	}

	// W-01: cada registro persiste la fase que cedió al escribirse, no la que resulte después.
	ledger := handOffLedger(t, ws)
	if len(ledger.Records) != 2 || ledger.Records[0].HandedOffTo != "apply:qa" || ledger.Records[1].HandedOffTo != "verify" {
		t.Errorf("registros = %+v, se esperaba handed_off_to apply:qa y verify", ledger.Records)
	}
}

// W-01: tras ceder core con la salida 2 (apply:qa) y aprobar qa con la salida 3, el ledger
// conserva apply:qa en el registro de core y el de qa no lleva ninguna clave de cesión.
func TestGateRecordHandOffPhaseSurvivesALaterApprovalOfTheOtherRole(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)
	if _, err := gateRecord(t, ws, "--gate", "role-apply:core", "--decision", "approved", "--hand-off"); err != nil {
		t.Fatalf("role-apply:core --hand-off: %v", err)
	}
	if _, err := gateRecord(t, ws, "--gate", "role-apply:qa", "--decision", "approved"); err != nil {
		t.Fatalf("role-apply:qa salida 3: %v", err)
	}

	ledger := handOffLedger(t, ws)
	if len(ledger.Records) != 2 {
		t.Fatalf("registros = %+v", ledger.Records)
	}
	if core := ledger.Records[0]; !core.HandedOff || core.HandedOffTo != "apply:qa" {
		t.Errorf("registro de core = %+v, se esperaba la cesión apply:qa", core)
	}
	if qa := ledger.Records[1]; qa.HandedOff || qa.HandedOffTo != "" {
		t.Errorf("registro de qa = %+v, la salida 3 no cede nada", qa)
	}
	data, err := os.ReadFile(filepath.Join(guardChangeRoot(ws), "gates.yaml"))
	if err != nil || strings.Count(string(data), "handed_off_to:") != 1 {
		t.Errorf("gates.yaml debe llevar handed_off_to una sola vez (%v):\n%s", err, data)
	}
}

// REQ-26.4 y DD-12: con código del rol sin confirmar la cesión de un role-apply ya no se rechaza.
// `gate record` no inspecciona el trabajo del rol; el código sin subir lo informa después
// `axiom sdd publish`.
func TestGateRecordHandOffOfARoleApplyDoesNotInspectTheRoleWork(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "internal", "core.go"), "package core\n")

	out, err := gateRecord(t, ws, "--gate", "role-apply:core", "--decision", "approved", "--hand-off")
	if err != nil {
		t.Fatalf("la cesión de un role-apply no inspecciona el trabajo del rol: %v", err)
	}
	if !strings.Contains(out, "axiom sdd publish") {
		t.Errorf("la cesión debe entregar la orden de publicación:\n%s", out)
	}
	if ledger := handOffLedger(t, ws); len(ledger.Records) != 1 || !ledger.Records[0].HandedOff {
		t.Errorf("registros = %+v, se esperaba el approved con la marca", ledger.Records)
	}
}

// REQ-26.1 y REQ-26.3: las salidas 1 y 3 no escriben la marca.
func TestGateRecordExitsOneAndThreeNeverWriteTheHandOffMark(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "rejected", "--reason", "falta un caso"); err != nil {
		t.Fatalf("salida 1: %v", err)
	}
	if _, err := gateRecord(t, ws, "--gate", "design", "--decision", "approved"); err != nil {
		t.Fatalf("salida 3: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(guardChangeRoot(ws), "gates.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "handed_off") {
		t.Errorf("las salidas 1 y 3 no escriben la marca:\n%s", data)
	}
}

// §4.4: `--json` emite `axiom.sdd-gate-record/v1` en todas las salidas; `delivery`, `handedOff` y
// `handedOffTo` solo en la salida 2, y `lastRoleClosed` solo en una compuerta de rol.
func TestGateRecordJSONInEveryExit(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)
	decode := func(t *testing.T, out string) map[string]any {
		t.Helper()
		var document map[string]any
		if err := json.Unmarshal([]byte(out), &document); err != nil {
			t.Fatalf("la salida no es JSON: %v\n%s", err, out)
		}
		if document["schema"] != "axiom.sdd-gate-record/v1" || document["change"] != "x" {
			t.Errorf("documento = %v", document)
		}
		return document
	}

	t.Run("salida 1", func(t *testing.T) {
		out, err := gateRecord(t, ws, "--gate", "spec", "--decision", "rejected", "--reason", "falta", "--json")
		if err != nil {
			t.Fatal(err)
		}
		document := decode(t, out)
		for _, key := range []string{"delivery", "handedOff", "handedOffTo", "lastRoleClosed", "publish", "publishUnavailable"} {
			if _, present := document[key]; present {
				t.Errorf("la salida 1 no lleva %q: %v", key, document)
			}
		}
		if document["gate"] != "spec" || document["decision"] != "rejected" {
			t.Errorf("documento = %v", document)
		}
	})
	t.Run("salida 3", func(t *testing.T) {
		out, err := gateRecord(t, ws, "--gate", "design", "--decision", "approved", "--json")
		if err != nil {
			t.Fatal(err)
		}
		document := decode(t, out)
		for _, key := range []string{"delivery", "handedOff", "handedOffTo"} {
			if _, present := document[key]; present {
				t.Errorf("la salida 3 no lleva %q: %v", key, document)
			}
		}
		// REQ-27.12: toda aprobaciÃ³n calcula la orden de publicaciÃ³n y no la ejecuta.
		publish, _ := document["publish"].(map[string]any)
		if publish["branch"] != "feat/x" || publish["prTarget"] != "main" || !strings.Contains(publish["invocation"].(string), "--branch feat/x") {
			t.Errorf("la salida 3 lleva publish con la rama real: %v", document)
		}
	})
	t.Run("salida 2", func(t *testing.T) {
		out, err := gateRecord(t, ws, "--gate", "tasks", "--decision", "approved", "--hand-off", "--json")
		if err != nil {
			t.Fatal(err)
		}
		document := decode(t, out)
		if document["handedOff"] != true || document["handedOffTo"] != "apply" {
			t.Errorf("documento = %v", document)
		}
		delivery, ok := document["delivery"].(map[string]any)
		if !ok || delivery["topology"] != "monorepo-embedded" {
			t.Fatalf("delivery = %v", document["delivery"])
		}
		// REQ-27.12: se retira mergeRequired y la cesión no exige PR fusionada.
		if _, present := delivery["mergeRequired"]; present {
			t.Errorf("delivery ya no lleva mergeRequired: %v", delivery)
		}
		if _, present := delivery["deliveries"]; present {
			t.Errorf("sin repositorios de código del rol no hay deliveries: %v", delivery)
		}
		publish, _ := delivery["publish"].(map[string]any)
		if publish["remote"] != "origin" || publish["branch"] != "feat/x" || publish["prTarget"] != "main" {
			t.Errorf("delivery.publish = %v", publish)
		}
		invocation, _ := publish["invocation"].(string)
		if !strings.HasPrefix(invocation, "axiom sdd publish --cwd ") || !strings.HasSuffix(invocation, " --change x --branch feat/x") {
			t.Errorf("la invocación debe estar atada a la rama con --branch: %q", invocation)
		}
		if topLevel, _ := document["publish"].(map[string]any); topLevel["invocation"] != invocation || topLevel["prTarget"] != "main" {
			t.Errorf("publish de nivel superior = %v", document["publish"])
		}
		resume, _ := delivery["resume"].([]any)
		if len(resume) != 2 || resume[0] != "git fetch origin" || resume[1] != "axiom branch ensure --change x --json" {
			t.Errorf("resume = %v", delivery["resume"])
		}
		if strings.Contains(invocation, "&&") || strings.Contains(invocation, "gh ") || strings.Contains(invocation, "glab ") {
			t.Errorf("invocación no admisible: %q", invocation)
		}
	})
	t.Run("compuerta de rol", func(t *testing.T) {
		out, err := gateRecord(t, ws, "--gate", "role-apply:core", "--decision", "approved", "--json")
		if err != nil {
			t.Fatal(err)
		}
		document := decode(t, out)
		if closed, present := document["lastRoleClosed"]; !present || closed != false {
			t.Errorf("lastRoleClosed = %v (presente %v), se esperaba false con qa pendiente", closed, present)
		}
		out, err = gateRecord(t, ws, "--gate", "role-apply:qa", "--decision", "approved", "--hand-off", "--json")
		if err != nil {
			t.Fatal(err)
		}
		document = decode(t, out)
		if document["lastRoleClosed"] != true || document["handedOffTo"] != "verify" {
			t.Errorf("documento = %v", document)
		}
		if strings.Contains(out, lastRoleNoticeMarker) {
			t.Errorf("en --json el aviso de último rol es el campo lastRoleClosed, no texto:\n%s", out)
		}
	})
}

// REQ-26.3, A-1: la salida 2 y la salida 3 de la misma puerta pendiente, a la vez. Si la salida 2
// se registra es porque su marca es la del primer approved; si se rechaza es porque la salida 3 ganó
// y ningún registro lleva marca: nunca queda una cesión invisible tras otra aprobación.
func TestGateRecordConcurrentHandOffAndPlainApprovalNeverHideTheMark(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, extra := range [][]string{
		{"--gate", "spec", "--decision", "approved", "--hand-off"},
		{"--gate", "spec", "--decision", "approved"},
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out bytes.Buffer
			errs[i] = RunSDDGate(append([]string{"record", "--cwd", ws.Master, "--change", sealChange}, extra...), &out)
		}()
	}
	wg.Wait()

	if errs[1] != nil {
		t.Fatalf("la salida 3 no se rechaza nunca: %v", errs[1])
	}
	ledger := handOffLedger(t, ws)
	first, approved := ledger.Approval(kickoff.GateSpec)
	if !approved {
		t.Fatalf("registros = %+v, la puerta debía quedar aprobada", ledger.Records)
	}
	if errs[0] == nil && !first.HandedOff {
		t.Errorf("la salida 2 se registró pero el primer approved no lleva la marca: %+v", ledger.Records)
	}
	if errs[0] != nil {
		if !strings.Contains(errs[0].Error(), "ya está resuelta") {
			t.Errorf("el rechazo de la salida 2 debe decir que la puerta ya está resuelta: %v", errs[0])
		}
		for _, record := range ledger.Records {
			if record.HandedOff {
				t.Errorf("la salida 2 se rechazó y aun así hay una marca: %+v", ledger.Records)
			}
		}
	}
}

// REQ-26.5, segundo escenario: en el repositorio de specs de un multirepo, que se trabaja in situ en
// la integración, la salida 2 entrega la publicaciÃ³n de la rama `main`, sin rama de cambio
// ni destino de PR (REQ-27.12).
func TestGateRecordHandOffInTheSpecsRepositoryOfAMultirepoIsInPlace(t *testing.T) {
	captureSealWarnings(t)
	captureWriterWarnings(t)
	ws := multirepoSealProject(t)
	change := planningHomeChange
	args := []string{"seal", "--cwd", ws.Master, "--change", change, "--role", "backend", "--role", "frontend", "--execution-style", "checkpointed", "--handoff-policy", "none"}
	if err := RunSDDKickoff(args, &bytes.Buffer{}); err != nil {
		t.Fatalf("sellar el multirepo: %v", err)
	}
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", change, "spec.md"), "# spec\n")
	integrateCommit(t, ws.Specs, "kickoff y spec")

	var out bytes.Buffer
	err := RunSDDGate([]string{"record", "--cwd", ws.Master, "--change", change, "--gate", "spec", "--decision", "approved", "--hand-off"}, &out)

	if err != nil {
		t.Fatalf("gate record --hand-off en el repositorio de specs: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"fase siguiente cedida: design",
		"axiom sdd publish --cwd ",
		" --change " + change + " --branch main",
		"rama de integración main en origin",
		"se trabaja in situ en la integración, sin PR",
		"no hace falta una PR fusionada",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"feat/", "cabecera", "push -u", "destino de su PR", "gh ", "glab "} {
		if strings.Contains(text, forbidden) {
			t.Errorf("el repositorio de specs no tiene rama de cambio ni PR; la salida no puede contener %q:\n%s", forbidden, text)
		}
	}
}
