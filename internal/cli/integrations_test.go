package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

const integrationsCanary = "s3cr3t-CANARIO-9f8e7d"

// runIntegrationsCLI ejecuta RunIntegrations con stdin dado y devuelve código, stdout y stderr.
func runIntegrationsCLI(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := RunIntegrations(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestRunIntegrationsUsageAndUnknownSubcommand(t *testing.T) {
	if code, out, _ := runIntegrationsCLI(t, ""); code != 2 || !strings.Contains(out, "sync") {
		t.Fatalf("sin argumentos: código %d, salida %q", code, out)
	}
	code, out, errOut := runIntegrationsCLI(t, "", "inspect")
	if code == 0 || !strings.Contains(out, "emit") || !strings.Contains(errOut, "inspect") {
		t.Fatalf("subcomando desconocido: código %d, stdout %q, stderr %q", code, out, errOut)
	}
	if code, out, _ := runIntegrationsCLI(t, "", "--help"); code != 0 || !strings.Contains(out, "secret") {
		t.Fatalf("--help: %d %q", code, out)
	}
	for _, sub := range []string{"sync", "emit", "secret"} {
		if code, out, _ := runIntegrationsCLI(t, "", sub, "--help"); code != 0 || !strings.Contains(out, "Uso: axiom integrations "+sub) {
			t.Errorf("%s --help: %d %q", sub, code, out)
		}
	}
	if code, out, _ := runIntegrationsCLI(t, "", "secret", "set", "--help"); code != 0 || !strings.Contains(out, "stdin") {
		t.Errorf("secret set --help: %d %q", code, out)
	}
	if code, _, _ := runIntegrationsCLI(t, "", "secret", "inspect"); code != 2 {
		t.Errorf("secret con subcomando desconocido sale con 2, salió con %d", code)
	}
	if code, _, errOut := runIntegrationsCLI(t, "", "sync", "--nada"); code != 2 || !strings.Contains(errOut, "nada") {
		t.Errorf("bandera desconocida: %d %q", code, errOut)
	}
}

// ---- sync

func TestIntegrationsSyncWithoutBlockOrQueueExitsZeroAndCreatesNothing(t *testing.T) {
	root, changeRoot, _ := integrationsWorkspace(t, "")
	code, out, errOut := runIntegrationsCLI(t, "", "sync", "--cwd", root)
	if code != 0 || errOut != "" || !strings.Contains(out, "No hay colas") {
		t.Fatalf("sin bloque: %d %q %q", code, out, errOut)
	}
	assertEmptyChangeRoot(t, changeRoot)

	root, changeRoot, _ = integrationsWorkspace(t, fakeBlock)
	if code, _, _ := runIntegrationsCLI(t, "", "sync", "--cwd", root, "--change", "x"); code != 0 {
		t.Fatalf("sin cola: %d", code)
	}
	assertEmptyChangeRoot(t, changeRoot)
}

func TestIntegrationsSyncInvalidBlockIsAnError(t *testing.T) {
	root, _, _ := integrationsWorkspace(t, "integrations:\n  azure: {}\n")
	code, _, errOut := runIntegrationsCLI(t, "", "sync", "--cwd", root)
	if code != 1 || !strings.Contains(errOut, "plugin desconocido") {
		t.Fatalf("bloque inválido: %d %q", code, errOut)
	}
}

// failDelivery deja una operación pendiente en la cola de x con el doble fallando.
func failDelivery(t *testing.T, root, changeRoot string, fake *integrationstest.Fake) {
	t.Helper()
	captureIntegrationWarnings(t)
	fake.FailAlways(errors.New("tracker caído"))
	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, "gate.spec.approved", func() integrations.Event { return specEventFor("x") })
	if !fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatal("la operación debía quedar en la cola")
	}
}

func TestIntegrationsSyncRetriesDeliversAndExitsByWhatRemains(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	failDelivery(t, root, changeRoot, fake)

	code, _, errOut := runIntegrationsCLI(t, "", "sync", "--cwd", root)
	if code != 1 || !strings.Contains(errOut, "gate.spec.approved/x/dabc") || !strings.Contains(errOut, "tracker caído") {
		t.Fatalf("con algo pendiente sale con 1 y resume por stderr: %d %q", code, errOut)
	}

	fake.Recover()
	code, out, errOut := runIntegrationsCLI(t, "", "sync", "--cwd", root, "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("sync entrega y sale con 0: %d %q", code, errOut)
	}
	var decoded struct {
		Schema    string `json:"schema"`
		Remaining bool   `json:"remaining"`
		Changes   []struct {
			Change    string           `json:"change"`
			Delivered []map[string]any `json:"delivered"`
			Pending   []map[string]any `json:"pending"`
			Abandoned []map[string]any `json:"abandoned"`
			Discarded []map[string]any `json:"discarded"`
		} `json:"changes"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if decoded.Schema != "axiom.integrations-sync/v1" || decoded.Remaining || len(decoded.Changes) != 1 || len(decoded.Changes[0].Delivered) != 1 || decoded.Changes[0].Pending == nil || decoded.Changes[0].Abandoned == nil || decoded.Changes[0].Discarded == nil {
		t.Fatalf("JSON = %+v\n%s", decoded, out)
	}
	if fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatal("la cola se borra")
	}
}

func TestIntegrationsSyncRetryAbandonedAndDiscard(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	captureIntegrationWarnings(t)
	fake.FailAlways(errors.New("sigue caído"))
	loc := locateIntegrations(t, root)
	for i := 0; i < 10; i++ { // diez intentos abandonan la operación
		dispatchIntegrationEvent(loc, changeRoot, "gate.spec.approved", func() integrations.Event { return specEventFor("x") })
	}
	fake.Recover()

	code, out, _ := runIntegrationsCLI(t, "", "sync", "--cwd", root, "--json")
	if code != 1 || !strings.Contains(out, `"abandoned"`) || !strings.Contains(out, "gate.spec.approved/x/dabc") {
		t.Fatalf("sin --retry-abandoned sigue abandonada: %d %s", code, out)
	}
	if code, _, _ := runIntegrationsCLI(t, "", "sync", "--cwd", root, "--discard", "gate.spec.approved/x/dabc"); code != 0 {
		t.Fatalf("--discard retira la operación y no queda nada: %d", code)
	}
	if fake.Calls() != 10 {
		t.Fatalf("lo descartado no se entrega (llamadas %d)", fake.Calls())
	}

	fake.FailAlways(errors.New("caído otra vez"))
	for i := 0; i < 10; i++ {
		dispatchIntegrationEvent(loc, changeRoot, "gate.spec.approved", func() integrations.Event { return specEventFor("x") })
	}
	fake.Recover()
	if code, _, _ := runIntegrationsCLI(t, "", "sync", "--cwd", root, "--retry-abandoned"); code != 0 {
		t.Fatalf("--retry-abandoned la reactiva y la entrega: %d", code)
	}
}

// ---- emit

func TestIntegrationsEmitRejections(t *testing.T) {
	root, _, _ := integrationsWorkspace(t, fakeBlock)
	sealForGateTest(t, root, "x")
	tests := []struct {
		name  string
		args  []string
		wants string
	}{
		{"change.start", []string{"--change", "x", "--event", "change.start"}, "C-07-d"},
		{"evento fuera del catálogo", []string{"--change", "x", "--event", "gate.tasks.approved"}, "eventos admitidos"},
		{"rol sin --role", []string{"--change", "x", "--event", "role.start"}, "--role"},
		{"rol fuera del roster", []string{"--change", "x", "--event", "role.start", "--role", "inexistente"}, "roster"},
		{"cambio inexistente", []string{"--change", "no-existe", "--event", "role.start", "--role", "fullstack"}, "no existe"},
		{"archive sin carpeta archivada", []string{"--change", "x", "--event", "archive"}, "carpeta archivada"},
		{"compuerta sin decisión", []string{"--change", "x", "--event", "gate.spec.approved"}, "no fabrica aprobaciones"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureIntegrationWarnings(t)
			code, _, errOut := runIntegrationsCLI(t, "", append([]string{"emit", "--cwd", root}, tt.args...)...)
			if code != 1 || !strings.Contains(errOut, tt.wants) {
				t.Fatalf("código %d, stderr %q; se esperaba un error con %q", code, errOut, tt.wants)
			}
		})
	}
	if code, _, _ := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x"); code != 2 {
		t.Fatalf("sin --event es un error de uso: %d", code)
	}
}

func TestIntegrationsEmitGateEventsRequireTheLastDecisionApproved(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	captureIntegrationWarnings(t)
	digest := "0123456789abcdef0123456789abcdef"

	approved := approvedRecord(kickoff.GateSpec, digest)
	rejected := approvedRecord(kickoff.GateSpec, digest)
	rejected.Decision = kickoff.DecisionRejected
	for _, r := range []kickoff.GateRecord{approved, rejected} {
		if err := kickoff.AppendGate(changeRoot, r); err != nil {
			t.Fatal(err)
		}
	}
	code, _, errOut := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "gate.spec.approved")
	if code != 1 || !strings.Contains(errOut, "no approved") {
		t.Fatalf("la última decisión es rejected: %d %q", code, errOut)
	}
	if fake.Calls() != 0 {
		t.Fatal("no se entrega nada")
	}

	again := approvedRecord(kickoff.GateSpec, digest)
	if err := kickoff.AppendGate(changeRoot, again); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "gate.spec.approved", "--json")
	if code != 0 || errOut != "" {
		t.Fatalf("con la última aprobada se emite: %d %q", code, errOut)
	}
	var decoded struct {
		Schema      string           `json:"schema"`
		Key         string           `json:"key"`
		Delivered   []map[string]any `json:"delivered"`
		NothingToDo bool             `json:"nothing_to_do"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Schema != "axiom.integrations-emit/v1" || decoded.Key != "gate.spec.approved/x/d0123456789abcdef" || len(decoded.Delivered) != 1 || decoded.NothingToDo {
		t.Fatalf("JSON = %+v", decoded)
	}
	if ev := fake.Delivered()[0]; ev.Actor != "Ana <ana@example.test>" || ev.ArtifactDigest != digest {
		t.Fatalf("toma digest y actor del registro: %+v", ev)
	}
}

func TestIntegrationsEmitRoleStartAndPendingExit(t *testing.T) {
	root, _, fake := integrationsWorkspace(t, fakeBlock)
	sealForGateTest(t, root, "x")
	captureIntegrationWarnings(t)

	code, out, _ := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "role.start", "--role", "Fullstack", "--actor", "Eva")
	if code != 0 || !strings.Contains(out, "role.start/x/fullstack") {
		t.Fatalf("role.start: %d %q", code, out)
	}
	if ev := fake.Delivered()[0]; ev.Actor != "Eva" || ev.Role != "fullstack" {
		t.Fatalf("evento = %+v", ev)
	}

	fake.FailAlways(errors.New("caído"))
	code, _, errOut := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "role.start", "--role", "fullstack")
	if code != 1 || !strings.Contains(errOut, "role.start/x/fullstack") {
		t.Fatalf("con el doble fallando queda pendiente y sale con 1: %d %q", code, errOut)
	}
}

func TestIntegrationsEmitNothingToDoWithoutListeners(t *testing.T) {
	root, changeRoot, _ := integrationsWorkspace(t, "")
	sealForGateTest(t, root, "x")
	code, out, _ := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "role.start", "--role", "fullstack")
	if code != 0 || !strings.Contains(out, "Ningún plugin activo escucha") {
		t.Fatalf("sin oyentes: %d %q", code, out)
	}
	if fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatal("no se encola nada")
	}
	code, out, _ = runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "role.start", "--role", "fullstack", "--json")
	if code != 0 || !strings.Contains(out, `"nothing_to_do": true`) {
		t.Fatalf("JSON sin oyentes: %d %s", code, out)
	}
}

func TestIntegrationsEmitUndeliveredExitsOneWithQueuePath(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	sealForGateTest(t, root, "x")
	captureIntegrationWarnings(t)
	fake.FailAlways(errors.New("tracker caído"))
	code, _, errOut := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "role.start", "--role", "fullstack")
	if code != 1 || !strings.Contains(errOut, filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatalf("pendiente: %d %q", code, errOut)
	}
}

// ---- secret

func newSecretWorkspace(t *testing.T) *secrets.Memory {
	t.Helper()
	fake := integrationstest.NewFake()
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", "")
	return integrationstest.Install(t, fake)
}

func TestIntegrationsSecretSetReadsOnlyStdin(t *testing.T) {
	mem := newSecretWorkspace(t)

	code, out, errOut := runIntegrationsCLI(t, integrationsCanary+"\n", "secret", "set", "--plugin", "fake", "--name", "token")
	if code != 0 {
		t.Fatalf("set: %d %q", code, errOut)
	}
	if strings.Contains(out+errOut, integrationsCanary) {
		t.Fatal("stdout y stderr no contienen el valor")
	}
	got, _, err := mem.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"})
	if err != nil || got.Reveal() != integrationsCanary {
		t.Fatalf("el valor queda en el almacén sin el salto de línea final: %q %v", got.Reveal(), err)
	}

	// Solo se quita un único salto de línea final.
	runIntegrationsCLI(t, "dos\n\n", "secret", "set", "--plugin", "fake", "--name", "token")
	got, _, _ = mem.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"})
	if got.Reveal() != "dos\n" {
		t.Fatalf("un único salto de línea final fuera: %q", got.Reveal())
	}
	runIntegrationsCLI(t, "tres\r\n", "secret", "set", "--plugin", "fake", "--name", "token")
	got, _, _ = mem.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"})
	if got.Reveal() != "tres" {
		t.Fatalf("CRLF final fuera: %q", got.Reveal())
	}
}

func TestIntegrationsSecretSetRejections(t *testing.T) {
	mem := newSecretWorkspace(t)
	untouched := func() {
		t.Helper()
		if _, _, err := mem.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}); !errors.Is(err, secrets.ErrNotFound) {
			t.Fatalf("no debe escribirse nada: %v", err)
		}
	}

	t.Run("--value es una opción desconocida", func(t *testing.T) {
		code, _, errOut := runIntegrationsCLI(t, "", "secret", "set", "--plugin", "fake", "--name", "token", "--value", integrationsCanary)
		if code != 2 || strings.Contains(errOut, integrationsCanary) {
			t.Fatalf("--value se rechaza sin repetir el valor: %d %q", code, errOut)
		}
		untouched()
	})
	t.Run("valor vacío", func(t *testing.T) {
		for _, in := range []string{"", "\n", "\r\n"} {
			if code, _, errOut := runIntegrationsCLI(t, in, "secret", "set", "--plugin", "fake", "--name", "token"); code != 1 || !strings.Contains(errOut, "vacío") {
				t.Fatalf("%q: %d %q", in, code, errOut)
			}
		}
		untouched()
	})
	t.Run("más de 4 KiB", func(t *testing.T) {
		code, _, errOut := runIntegrationsCLI(t, strings.Repeat("a", 4097), "secret", "set", "--plugin", "fake", "--name", "token")
		if code != 1 || !strings.Contains(errOut, "4096") {
			t.Fatalf("%d %q", code, errOut)
		}
		if code, _, _ := runIntegrationsCLI(t, strings.Repeat("a", 4096), "secret", "set", "--plugin", "fake", "--name", "token"); code != 0 {
			t.Fatalf("4 KiB exactos caben: %d", code)
		}
		_ = mem.Delete(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"})
	})
	t.Run("terminal", func(t *testing.T) {
		original := stdinIsTerminal
		stdinIsTerminal = func(io.Reader) bool { return true }
		t.Cleanup(func() { stdinIsTerminal = original })
		code, _, errOut := runIntegrationsCLI(t, integrationsCanary, "secret", "set", "--plugin", "fake", "--name", "token")
		if code != 1 || !strings.Contains(errOut, "Canaliza") || !strings.Contains(errOut, "Read-Host -MaskInput") {
			t.Fatalf("terminal: %d %q", code, errOut)
		}
		untouched()
	})
	t.Run("secreto no declarado", func(t *testing.T) {
		code, _, errOut := runIntegrationsCLI(t, "x", "secret", "set", "--plugin", "fake", "--name", "otro")
		if code != 1 || !strings.Contains(errOut, "secretos declarados: token") {
			t.Fatalf("%d %q", code, errOut)
		}
		code, _, errOut = runIntegrationsCLI(t, "x", "secret", "set", "--plugin", "azure", "--name", "token")
		if code != 1 || !strings.Contains(errOut, "plugins admitidos: fake") {
			t.Fatalf("plugin desconocido: %d %q", code, errOut)
		}
		untouched()
	})
	t.Run("faltan banderas", func(t *testing.T) {
		if code, _, _ := runIntegrationsCLI(t, "x", "secret", "set", "--plugin", "fake"); code != 2 {
			t.Fatalf("sin --name es un error de uso: %d", code)
		}
	})
}

func TestIntegrationsSecretStatusNeverRevealsValueOrLength(t *testing.T) {
	mem := newSecretWorkspace(t)

	code, out, _ := runIntegrationsCLI(t, "", "secret", "status", "--plugin", "fake")
	if code != 0 || !strings.Contains(out, "fake/token: missing") {
		t.Fatalf("missing: %d %q", code, out)
	}

	_ = mem.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(integrationsCanary))
	code, out, _ = runIntegrationsCLI(t, "", "secret", "status", "--plugin", "fake", "--json")
	if code != 0 || strings.Contains(out, integrationsCanary) || strings.Contains(out, "length") || strings.Contains(out, "len") {
		t.Fatalf("status no revela valor ni longitud: %d %s", code, out)
	}
	var decoded integrationsSecretStatusJSON
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Schema != "axiom.integrations-secret-status/v1" || decoded.Plugin != "fake" || len(decoded.Secrets) != 1 || decoded.Secrets[0].Source != "keyring" {
		t.Fatalf("JSON = %+v", decoded)
	}

	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", integrationsCanary)
	_, out, _ = runIntegrationsCLI(t, "", "secret", "status", "--plugin", "fake", "--name", "token", "--json")
	if !strings.Contains(out, `"source": "env"`) || strings.Contains(out, integrationsCanary) {
		t.Fatalf("env: %s", out)
	}

	if code, _, errOut := runIntegrationsCLI(t, "", "secret", "status", "--plugin", "fake", "--name", "otro"); code != 1 || !strings.Contains(errOut, "token") {
		t.Fatalf("nombre no declarado: %d %q", code, errOut)
	}
	if code, _, _ := runIntegrationsCLI(t, "", "secret", "status"); code != 2 {
		t.Fatalf("sin --plugin: %d", code)
	}
}

func TestIntegrationsSecretDeleteIsIdempotent(t *testing.T) {
	mem := newSecretWorkspace(t)
	_ = mem.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret("v"))
	for i := 0; i < 2; i++ {
		code, out, errOut := runIntegrationsCLI(t, "", "secret", "delete", "--plugin", "fake", "--name", "token")
		if code != 0 || errOut != "" || !strings.Contains(out, "AXIOM_INTEGRATION_FAKE_TOKEN") {
			t.Fatalf("delete %d: %d %q %q", i, code, out, errOut)
		}
	}
	if _, _, err := mem.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}); !errors.Is(err, secrets.ErrNotFound) {
		t.Fatalf("el secreto ya no existe: %v", err)
	}
	if code, _, _ := runIntegrationsCLI(t, "", "secret", "delete", "--plugin", "fake", "--name", "otro"); code != 1 {
		t.Fatalf("secreto no declarado: %d", code)
	}
}
