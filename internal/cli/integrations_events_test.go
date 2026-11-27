package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const integrationsBase = "workspace:\n  name: test-workspace\n  topology: monorepo-embedded\nroles:\n  core:\n    name: core\n    repositories:\n      - path: .\n"

const fakeBlock = "integrations:\n  fake:\n    project: p\n"

// integrationsWorkspace monta un proyecto embebido con el bloque dado y el cambio x, e instala el
// doble. Devuelve la raíz, la carpeta del cambio y el doble.
func integrationsWorkspace(t *testing.T, block string) (root, changeRoot string, fake *integrationstest.Fake) {
	t.Helper()
	root, changeRoot = newGovernanceWorkspace(t, "x")
	writeCLIFixture(t, filepath.Join(root, "axiom.yaml"), integrationsBase+block)
	quiet := writerBranchWarnings
	writerBranchWarnings = io.Discard
	t.Cleanup(func() { writerBranchWarnings = quiet })
	fake = integrationstest.NewFake()
	integrationstest.Install(t, fake)
	return root, changeRoot, fake
}

func captureIntegrationWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := integrationWarnings
	integrationWarnings = &buf
	t.Cleanup(func() { integrationWarnings = previous })
	return &buf
}

func locateIntegrations(t *testing.T, root string) workspace.Location {
	t.Helper()
	loc, err := workspace.Locate(root)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	return loc
}

func specEventFor(change string) integrations.Event {
	return integrations.Event{Type: manifest.EventGateSpecApproved, Change: change, Gate: "spec", Actor: "cli", OccurredAt: time.Now().UTC(), IdempotencyKey: "gate.spec.approved/" + change + "/dabc"}
}

func assertEmptyChangeRoot(t *testing.T, changeRoot string) {
	t.Helper()
	entries, _ := os.ReadDir(changeRoot)
	for _, e := range entries {
		t.Errorf("no debe crearse ningún fichero en la carpeta del cambio: %s", e.Name())
	}
}

func fileExistsCLI(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestDispatchIntegrationEventInvalidBlockWarnsAndDoesNotDispatch(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, "integrations:\n  azure:\n    project: p\n")
	warnings := captureIntegrationWarnings(t)
	built := false

	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, manifest.EventGateSpecApproved, func() integrations.Event {
		built = true
		return integrations.Event{}
	})

	out := warnings.String()
	for _, want := range []string{"Aviso:", "plugin desconocido 'azure'", "gate.spec.approved", "axiom workspace validate", "axiom integrations emit"} {
		if !strings.Contains(out, want) {
			t.Errorf("el aviso debe contener %q:\n%s", want, out)
		}
	}
	if strings.Count(strings.TrimSpace(out), "\n") != 0 {
		t.Errorf("una sola línea de aviso:\n%s", out)
	}
	if built || fake.Calls() != 0 {
		t.Fatal("no se construye ni se despacha el evento")
	}
	assertEmptyChangeRoot(t, changeRoot)
}

// failingStore falla la prueba si se le pide un secreto.
type failingStore struct{ t *testing.T }

func (f failingStore) Get(context.Context, secrets.Ref) (secrets.Secret, secrets.Source, error) {
	f.t.Fatal("no se debe resolver ningún secreto")
	return secrets.Secret{}, secrets.SourceMissing, nil
}

func (f failingStore) Set(context.Context, secrets.Ref, secrets.Secret) error {
	f.t.Fatal("set")
	return nil
}

func (f failingStore) Delete(context.Context, secrets.Ref) error {
	f.t.Fatal("delete")
	return nil
}

func TestDispatchIntegrationEventWithoutListenersTouchesNothingAndNeverBuilds(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	secrets.SetDefaultForTest(t, failingStore{t})
	identityResolved := false
	original := resolveActorIdentity
	resolveActorIdentity = func(ctx context.Context, dir string) gitidentity.Identity {
		identityResolved = true
		return original(ctx, dir)
	}
	t.Cleanup(func() { resolveActorIdentity = original })
	warnings := captureIntegrationWarnings(t)
	built := false

	// El doble no escucha change.start.
	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, manifest.EventChangeStart, func() integrations.Event {
		built = true
		return integrations.Event{}
	})
	// Sin bloque tampoco hay oyentes.
	noBlock := locateIntegrations(t, root)
	noBlock.Declarations.Integrations = nil
	dispatchIntegrationEvent(noBlock, changeRoot, manifest.EventGateSpecApproved, func() integrations.Event {
		built = true
		return integrations.Event{}
	})

	if built || identityResolved || fake.Calls() != 0 || warnings.Len() != 0 {
		t.Fatalf("sin oyentes no se construye, no se resuelve la identidad y no se avisa: built=%v identity=%v calls=%d warnings=%q", built, identityResolved, fake.Calls(), warnings)
	}
	assertEmptyChangeRoot(t, changeRoot)
}

func TestDispatchIntegrationEventGuardRejectionWarnsAndReturns(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	warnings := captureIntegrationWarnings(t)
	var contexts []branch.Context
	original := integrationWriterGuard
	integrationWriterGuard = func(_, _, _ string, ctx branch.Context) error {
		contexts = append(contexts, ctx)
		return errors.New("rama equivocada")
	}
	t.Cleanup(func() { integrationWriterGuard = original })

	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, manifest.EventGateSpecApproved, func() integrations.Event {
		return specEventFor("x")
	})

	if !strings.Contains(warnings.String(), "Aviso:") || !strings.Contains(warnings.String(), "rama equivocada") {
		t.Fatalf("la guarda rechaza con un aviso: %q", warnings)
	}
	if fake.Calls() != 0 || len(contexts) != 1 || contexts[0] != branch.ContextOrdinary {
		t.Fatalf("guarda A en contexto ordinary, sin entrega: calls=%d contexts=%v", fake.Calls(), contexts)
	}
	assertEmptyChangeRoot(t, changeRoot)
}

func TestDispatchIntegrationEventUndeliveredNamesQueueAndSync(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	fake.FailAlways(errors.New("tracker caído"))
	warnings := captureIntegrationWarnings(t)

	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, manifest.EventGateSpecApproved, func() integrations.Event {
		return specEventFor("x")
	})

	out := warnings.String()
	queue := filepath.Join(changeRoot, ".integrations-outbox.yaml")
	for _, want := range []string{"Aviso:", "tracker caído", queue, "axiom integrations sync --cwd " + root + " --change x"} {
		if !strings.Contains(out, want) {
			t.Errorf("el aviso debe contener %q:\n%s", want, out)
		}
	}
	if !fileExistsCLI(queue) {
		t.Fatal("la operación queda en la cola")
	}
}

func TestDispatchIntegrationEventDeliversAndStaysSilent(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	fake.ReturnLinks(integrations.Link{Kind: "work_item", ID: "1"})
	warnings := captureIntegrationWarnings(t)

	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, manifest.EventGateSpecApproved, func() integrations.Event {
		return specEventFor("x")
	})

	if warnings.Len() != 0 {
		t.Fatalf("una entrega correcta no avisa: %q", warnings)
	}
	if got := fake.DeliveredKeys(); len(got) != 1 {
		t.Fatalf("entregas = %v", got)
	}
	if !fileExistsCLI(filepath.Join(changeRoot, "integrations.yaml")) || fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) {
		t.Fatal("vínculos escritos y cola retirada")
	}
}

func TestDispatchIntegrationEventNeverFailsAndSanitizesWarnings(t *testing.T) {
	const canary = "s3cr3t-CANARIO-9f8e7d"
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	fake.LeakTokenInError()
	if err := secrets.Default().Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(canary)); err != nil {
		t.Fatal(err)
	}
	warnings := captureIntegrationWarnings(t)

	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, manifest.EventGateSpecApproved, func() integrations.Event {
		return specEventFor("x")
	})

	if strings.Contains(warnings.String(), canary) || !strings.Contains(warnings.String(), "[REDACTED]") {
		t.Fatalf("los avisos pasan por la limpieza de secretos: %q", warnings)
	}
	queue, _ := os.ReadFile(filepath.Join(changeRoot, ".integrations-outbox.yaml"))
	if strings.Contains(string(queue), canary) {
		t.Fatal("la cola tampoco puede llevar el secreto")
	}

	// Una cola inválida es un aviso, nunca un error ni un pánico.
	if err := os.WriteFile(filepath.Join(changeRoot, ".integrations-outbox.yaml"), []byte("esto: no es una cola\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	warnings.Reset()
	dispatchIntegrationEvent(locateIntegrations(t, root), changeRoot, manifest.EventGateSpecApproved, func() integrations.Event { return specEventFor("x") })
	if !strings.Contains(warnings.String(), "Aviso:") || !strings.Contains(warnings.String(), "no se ha podido despachar") {
		t.Fatalf("aviso por cola inválida: %q", warnings)
	}
}

func approvedRecord(gate kickoff.GateKey, digest string) kickoff.GateRecord {
	return kickoff.GateRecord{Gate: gate, Decision: kickoff.DecisionApproved, Reason: "ok", ArtifactDigest: digest, Actor: "Ana <ana@example.test>", RecordedAt: time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)}
}

func TestEmitGateIntegrationEventBuildsTheD09Key(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	captureIntegrationWarnings(t)
	digest := "0123456789abcdef0123456789abcdef"

	for _, record := range []kickoff.GateRecord{
		approvedRecord(kickoff.GateSpec, digest),
		approvedRecord(kickoff.GateDesign, ""),
		approvedRecord(kickoff.RoleApplyGate("core"), digest),
	} {
		if err := kickoff.AppendGate(changeRoot, record); err != nil {
			t.Fatal(err)
		}
		emitGateIntegrationEvent(locateIntegrations(t, root), changeRoot, record, func(text string) { integrationWarn("%s", text) })
	}

	want := []string{
		"gate.spec.approved/x/d0123456789abcdef",
		"gate.design.approved/x/o1",
		"role.apply.approved/x/core/d0123456789abcdef",
	}
	got := fake.DeliveredKeys()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("claves = %v, se esperaban %v", got, want)
	}
	if first := fake.Delivered()[0]; first.Actor != "Ana <ana@example.test>" || first.ArtifactDigest != digest || first.Gate != "spec" {
		t.Fatalf("el evento toma actor, digest y compuerta del registro: %+v", first)
	}
}

func TestEmitGateIntegrationEventIgnoresRejectedTasksAndIntegration(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	captureIntegrationWarnings(t)
	rejected := approvedRecord(kickoff.GateSpec, "abc")
	rejected.Decision = kickoff.DecisionRejected
	for _, record := range []kickoff.GateRecord{rejected, approvedRecord(kickoff.GateTasks, "abc"), approvedRecord(kickoff.GateIntegration, "")} {
		emitGateIntegrationEvent(locateIntegrations(t, root), changeRoot, record, func(text string) { integrationWarn("%s", text) })
	}
	if fake.Calls() != 0 {
		t.Fatalf("rejected, tasks e integration no emiten (llamadas %d)", fake.Calls())
	}
	assertEmptyChangeRoot(t, changeRoot)
}

func TestEmitRoleStartEventKeyAndActor(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	captureIntegrationWarnings(t)
	emitRoleStartEvent(locateIntegrations(t, root), changeRoot, "backend")
	emitRoleStartEvent(locateIntegrations(t, root), changeRoot, "backend")

	if keys := fake.DeliveredKeys(); len(keys) != 1 || keys[0] != "role.start/x/backend" {
		t.Fatalf("claves = %v", keys)
	}
	if ev := fake.Delivered()[0]; ev.Role != "backend" || ev.Actor == "" || ev.Type != manifest.EventRoleStart {
		t.Fatalf("evento = %+v", ev)
	}
}

func TestApprovalOrdinalCountsApprovalsOfTheSameGate(t *testing.T) {
	_, changeRoot, _ := integrationsWorkspace(t, fakeBlock)
	first := approvedRecord(kickoff.GateSpec, "")
	second := approvedRecord(kickoff.GateSpec, "")
	for _, r := range []kickoff.GateRecord{first, second} {
		if err := kickoff.AppendGate(changeRoot, r); err != nil {
			t.Fatal(err)
		}
	}
	if got := approvalOrdinal(changeRoot, second); got != 2 {
		t.Fatalf("ordinal = %d, se esperaba 2", got)
	}
}

// adoBlock es un bloque integrations.azure-devops válido (RF-ADO-02).
const adoBlock = "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Fabrikam Team\n"

// RF-ADO-14: con el plugin azure-devops configurado y activo, `gate record` produce la misma salida
// y el mismo código que sin el bloque, y no crea integrations.yaml ni la cola: el manifiesto no
// escucha el evento de compuerta de spec.
func TestGateRecordIsIdenticalWithAndWithoutAzureDevOpsBlock(t *testing.T) {
	run := func(t *testing.T, block string) (stdout, stderr string, code int, changeRoot string) {
		t.Helper()
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		writeCLIFixture(t, filepath.Join(root, "axiom.yaml"), integrationsBase+block)
		var warnings bytes.Buffer
		previousWriter, previousIntegrations := writerBranchWarnings, integrationWarnings
		writerBranchWarnings, integrationWarnings = &warnings, &warnings
		t.Cleanup(func() { writerBranchWarnings, integrationWarnings = previousWriter, previousIntegrations })
		secrets.SetDefaultForTest(t, failingStore{t}) // ningún secreto se resuelve sin oyentes

		var out bytes.Buffer
		if err := RunSDDGate([]string{"record", "--cwd", root, "--change", "inc-99-example", "--gate", "spec", "--decision", "approved", "--reason", "ok", "--actor", "Ana <ana@example.test>"}, &out); err != nil {
			code = 1
			warnings.WriteString("error: " + err.Error())
		}
		normalize := func(s string) string { return strings.ReplaceAll(s, root, "<root>") }
		return normalize(out.String()), normalize(warnings.String()), code, changeRoot
	}

	withoutOut, withoutErr, withoutCode, _ := run(t, "")
	withOut, withErr, withCode, withRoot := run(t, adoBlock)

	if withOut != withoutOut || withErr != withoutErr || withCode != withoutCode {
		t.Fatalf("la salida debe coincidir con y sin el bloque:\nsin: %q %q %d\ncon: %q %q %d", withoutOut, withoutErr, withoutCode, withOut, withErr, withCode)
	}
	if withCode != 0 {
		t.Fatalf("el registro de la compuerta debe tener éxito (no basta con coincidir): %q %q %d", withOut, withErr, withCode)
	}
	for _, name := range []string{"integrations.yaml", ".integrations-outbox.yaml"} {
		if fileExistsCLI(filepath.Join(withRoot, name)) {
			t.Errorf("no debe crearse %s", name)
		}
	}
}
