package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// REQ-21.13: el aviso de último rol se emite al aprobarse el role-apply del último
// rol blocking, nombra los roles deferred y optional aún abiertos y no se repite
// cuando uno de ellos cierra después. Los nombres empiezan por
// TestGateRecordLastRoleNotice para entrar en el mismo patrón de `-run`.

const noticeChange = "inc-99-example"

// noticeRecord registra una compuerta de rol aprobada y devuelve la salida.
func noticeRecord(t *testing.T, root, role string, extra ...string) string {
	t.Helper()
	var out bytes.Buffer
	args := append([]string{
		"record", "--cwd", root, "--change", noticeChange,
		"--gate", "role-apply:" + role, "--decision", "approved", "--reason", "listo",
	}, extra...)
	if err := RunSDDGate(args, &out); err != nil {
		t.Fatalf("RunSDDGate(role-apply:%s) error = %v\n%s", role, err, out.String())
	}
	return out.String()
}

// noticeJSON decodifica la salida de `gate record --json`.
func noticeJSON(t *testing.T, out string) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal([]byte(out), &document); err != nil {
		t.Fatalf("salida JSON no válida: %v\n%s", err, out)
	}
	return document
}

func TestGateRecordLastRoleNoticeFiresWhenTheLastBlockingRoleClosesWithADeferredPending(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, noticeChange)
	sealForGateTest(t, root, noticeChange, "--role", "core:blocking", "--role", "e2e:deferred")

	out := noticeRecord(t, root, "core")
	if got := strings.Count(out, lastRoleNoticeMarker); got != 1 {
		t.Fatalf("el aviso aparece %d veces, se esperaba 1:\n%s", got, out)
	}
	for _, want := range []string{"e2e", "deferred", "pendiente", "no bloquean la verificacion global"} {
		if !strings.Contains(out, want) {
			t.Errorf("el aviso no incluye %q:\n%s", want, out)
		}
	}

	// El cierre posterior del rol deferred no repite el aviso.
	if later := noticeRecord(t, root, "e2e"); strings.Contains(later, lastRoleNoticeMarker) || strings.Contains(later, "Aviso:") {
		t.Errorf("el cierre de un rol deferred repite el aviso:\n%s", later)
	}
}

func TestGateRecordLastRoleNoticeNamesAnOptionalRoleWithoutDecision(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, noticeChange)
	sealForGateTest(t, root, noticeChange, "--role", "core:blocking", "--role", "docs:optional")

	out := noticeRecord(t, root, "core")
	if !strings.Contains(out, lastRoleNoticeMarker) || !strings.Contains(out, "docs") || !strings.Contains(out, "optional") {
		t.Errorf("el aviso debe nombrar a docs como rol optional pendiente:\n%s", out)
	}
}

func TestGateRecordLastRoleNoticeWaitsForTheOtherBlockingRole(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, noticeChange)
	sealForGateTest(t, root, noticeChange, "--role", "core:blocking", "--role", "web:blocking", "--role", "e2e:deferred")

	if first := noticeRecord(t, root, "core"); strings.Contains(first, lastRoleNoticeMarker) {
		t.Fatalf("aviso con web (blocking) aún pendiente:\n%s", first)
	}
	// Aprobar antes el deferred tampoco cierra nada.
	if deferred := noticeRecord(t, root, "e2e"); strings.Contains(deferred, lastRoleNoticeMarker) {
		t.Fatalf("aviso al aprobar un rol deferred con web aún pendiente:\n%s", deferred)
	}
	last := noticeRecord(t, root, "web")
	if got := strings.Count(last, lastRoleNoticeMarker); got != 1 {
		t.Fatalf("el aviso aparece %d veces al cerrar el último blocking, se esperaba 1:\n%s", got, last)
	}
	if strings.Contains(last, "no bloquean la verificacion global") {
		t.Errorf("con el deferred ya aprobado no queda ningún rol no bloqueante que nombrar:\n%s", last)
	}
}

// El JSON conserva lastRoleClosed, que significa «esta aprobación cerró el último
// rol blocking» (el aviso en forma de campo), y añade pendingNonBlockingRoles solo
// entonces.
func TestGateRecordLastRoleNoticeJSONMeaning(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, noticeChange)
	sealForGateTest(t, root, noticeChange, "--role", "core:blocking", "--role", "e2e:deferred", "--role", "docs:optional")

	closing := noticeJSON(t, noticeRecord(t, root, "core", "--json"))
	if closing["lastRoleClosed"] != true {
		t.Fatalf("lastRoleClosed = %v, se esperaba true al cerrar el último blocking", closing["lastRoleClosed"])
	}
	pending, _ := closing["pendingNonBlockingRoles"].([]any)
	if len(pending) != 2 {
		t.Fatalf("pendingNonBlockingRoles = %v, se esperaban e2e y docs", closing["pendingNonBlockingRoles"])
	}
	first, _ := pending[0].(map[string]any)
	second, _ := pending[1].(map[string]any)
	if first["role"] != "docs" && first["role"] != "e2e" {
		t.Errorf("rol inesperado: %v", first)
	}
	seen := map[string]string{}
	for _, entry := range []map[string]any{first, second} {
		seen[entry["role"].(string)] = entry["policy"].(string) + "/" + entry["status"].(string)
	}
	if seen["e2e"] != "deferred/pending" || seen["docs"] != "optional/pending" {
		t.Errorf("pendingNonBlockingRoles = %v", seen)
	}

	// La aprobación posterior del rol deferred no cerró nada: false y sin lista.
	later := noticeJSON(t, noticeRecord(t, root, "e2e", "--json"))
	if later["lastRoleClosed"] != false {
		t.Errorf("lastRoleClosed = %v al aprobar un rol deferred, se esperaba false (sin segundo aviso)", later["lastRoleClosed"])
	}
	if _, present := later["pendingNonBlockingRoles"]; present {
		t.Errorf("pendingNonBlockingRoles no debe salir fuera del cierre: %v", later)
	}
}

// Con todos los roles blocking, o con el rol único fullstack, ni el texto ni el JSON
// cambian: ninguna clave nueva y el aviso de siempre.
func TestGateRecordLastRoleNoticeAllBlockingIsUnchanged(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, noticeChange)
	sealForGateTest(t, root, noticeChange, "--role", "core", "--role", "web")

	noticeRecord(t, root, "core")
	text := noticeRecord(t, root, "web")
	if strings.Count(text, lastRoleNoticeMarker) != 1 || strings.Contains(text, "no bloqueantes") {
		t.Errorf("el aviso con todos los roles blocking debe ser el de siempre:\n%s", text)
	}

	jsonRoot, _ := newGovernanceWorkspace(t, noticeChange)
	sealForGateTest(t, jsonRoot, noticeChange, "--role", "core", "--role", "web")
	noticeRecord(t, jsonRoot, "core")
	document := noticeJSON(t, noticeRecord(t, jsonRoot, "web", "--json"))
	if document["lastRoleClosed"] != true {
		t.Errorf("lastRoleClosed = %v, se esperaba true", document["lastRoleClosed"])
	}
	if _, present := document["pendingNonBlockingRoles"]; present {
		t.Errorf("con todos los roles blocking no debe salir pendingNonBlockingRoles: %v", document)
	}

	soloRoot, _ := newGovernanceWorkspace(t, noticeChange)
	sealForGateTest(t, soloRoot, noticeChange)
	solo := noticeJSON(t, noticeRecord(t, soloRoot, "fullstack", "--json"))
	if solo["lastRoleClosed"] != true {
		t.Errorf("rol único fullstack: lastRoleClosed = %v, se esperaba true", solo["lastRoleClosed"])
	}
	if _, present := solo["pendingNonBlockingRoles"]; present {
		t.Errorf("rol único fullstack: no debe salir pendingNonBlockingRoles: %v", solo)
	}
}
