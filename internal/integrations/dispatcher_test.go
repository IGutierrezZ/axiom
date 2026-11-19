package integrations

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

var durable = DispatchOptions{Durable: true, PersistOutcome: true}

func specEvent(key string) Event {
	return Event{Type: manifest.EventGateSpecApproved, Change: "x", Gate: "spec", Actor: "Prueba <p@example.test>", OccurredAt: fixedNow, IdempotencyKey: key}
}

func setup(t *testing.T) (workspaceFixture, *testPlugin, *Dispatcher) {
	t.Helper()
	f := newGitWorkspace(t)
	plugin := newTestPlugin()
	installTestPlugin(t, plugin)
	return f, plugin, newTestDispatcher(f, "fake")
}

func TestDispatchEnqueuesBeforeTheImmediateAttemptAndRemovesOnSuccess(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setLinks(Link{Kind: "work_item", ID: "12345", URL: "https://tracker.example/12345"})

	var queuedDuringDelivery bool
	plugin.hook = func(Delivery) {
		ob, err := readOutbox(f.Folder())
		queuedDuringDelivery = err == nil && len(ob.Operations) == 1 && ob.Operations[0].State == StatePending
	}

	report := d.Dispatch(context.Background(), f.Root, specEvent("gate.spec.approved/x/dabc"), durable)

	if report.Err != nil || len(report.Delivered) != 1 || len(report.Pending) != 0 || len(report.Abandoned) != 0 {
		t.Fatalf("informe = %+v", report)
	}
	if !queuedDuringDelivery {
		t.Fatal("la operación debe estar en la cola (duradera) antes del intento inmediato")
	}
	if fileExists(f.Folder().path(outboxName)) {
		t.Fatal("entregada y retirada, la cola vacía se borra")
	}
	doc, err := readLinks(f.Folder())
	if err != nil || len(doc.Links) != 1 || doc.Links[0].ID != "12345" || doc.Links[0].Source != "gate.spec.approved/x/dabc" || doc.Links[0].Plugin != "fake" {
		t.Fatalf("los vínculos devueltos se fusionan en integrations.yaml: %+v %v", doc, err)
	}
	if keys := plugin.deliveredKeys(); len(keys) != 1 || keys[0] != "gate.spec.approved/x/dabc" {
		t.Fatalf("el plugin recibió %v", keys)
	}
}

func TestDispatchPassesConfigLinksAndScopedSecretsToThePlugin(t *testing.T) {
	f, plugin, d := setup(t)
	writeTestFile(t, f.Folder().path(LinksFile), validLinksYAML)

	d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	got := plugin.received
	if len(got) != 1 {
		t.Fatalf("entregas = %d", len(got))
	}
	delivery := got[0]
	if delivery.Config["project"] != "p" {
		t.Fatalf("Config = %v", delivery.Config)
	}
	if _, has := delivery.Config["enabled"]; has {
		t.Fatalf("Config no incluye la clave reservada enabled: %v", delivery.Config)
	}
	if len(delivery.Links) != 1 || delivery.Links[0].ID != "12345" {
		t.Fatalf("Links es la instantánea de integrations.yaml: %+v", delivery.Links)
	}
	if delivery.Secrets == nil || delivery.Event.IdempotencyKey != "k1" {
		t.Fatalf("la entrega lleva el evento y el origen de secretos: %+v", delivery)
	}
}

func TestDispatchFailureCountsAnAttemptAndKeepsThePendingOperation(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setFailAlways(errors.New("timeout del tracker\ncon dos líneas"))

	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	if len(report.Pending) != 1 || len(report.Delivered) != 0 || report.Pending[0].Attempts != 1 {
		t.Fatalf("informe = %+v", report)
	}
	ob, err := readOutbox(f.Folder())
	if err != nil || len(ob.Operations) != 1 {
		t.Fatalf("la operación sigue en la cola: %+v %v", ob, err)
	}
	op := ob.Operations[0]
	if op.State != StatePending || op.Attempts != 1 || !op.LastAttemptAt.Equal(fixedNow) {
		t.Fatalf("operación = %+v", op)
	}
	if op.LastError != "timeout del tracker con dos líneas" {
		t.Fatalf("last_error saneado a una línea: %q", op.LastError)
	}
	if report.QueuePath != f.Folder().path(outboxName) {
		t.Fatalf("QueuePath = %q", report.QueuePath)
	}
}

func TestDispatchAbandonsAtTheTenthAttempt(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setFailAlways(errors.New("sigue caído"))

	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)
		ob, _ := readOutbox(f.Folder())
		if len(ob.Operations) != 1 {
			t.Fatalf("intento %d: la operación se deduplica, hay %d", attempt, len(ob.Operations))
		}
		op := ob.Operations[0]
		if op.Attempts != attempt {
			t.Fatalf("intento %d: attempts = %d", attempt, op.Attempts)
		}
		if attempt < MaxAttempts && (op.State != StatePending || len(report.Pending) != 1) {
			t.Fatalf("intento %d: debe seguir pendiente: %+v", attempt, op)
		}
		if attempt == MaxAttempts && (op.State != StateAbandoned || len(report.Abandoned) != 1) {
			t.Fatalf("el décimo intento abandona: %+v %+v", op, report)
		}
	}

	// Una abandonada no se vuelve a intentar con un despacho nuevo.
	calls := plugin.calls()
	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)
	if plugin.calls() != calls || len(report.Abandoned) != 1 {
		t.Fatalf("una abandonada no se reintenta sola: llamadas %d -> %d, informe %+v", calls, plugin.calls(), report)
	}
}

func TestDispatchPermanentErrorAbandonsImmediately(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setFailAlways(Permanent(errors.New("401 no autorizado")))

	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	if len(report.Abandoned) != 1 || len(report.Pending) != 0 || report.Abandoned[0].Attempts != 1 {
		t.Fatalf("un error permanente abandona al primer intento: %+v", report)
	}
	ob, _ := readOutbox(f.Folder())
	if ob.Operations[0].State != StateAbandoned {
		t.Fatalf("estado = %s", ob.Operations[0].State)
	}
}

func TestDispatchRecoversFromAPluginPanicAsATransientFailure(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.panicWith, plugin.hasPanic = "kaboom", true

	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	if len(report.Pending) != 1 || report.Pending[0].Attempts != 1 {
		t.Fatalf("un pánico cuenta como fallo transitorio: %+v", report)
	}
	if !strings.Contains(report.Pending[0].LastError, "kaboom") {
		t.Fatalf("el error recoge el valor del pánico: %q", report.Pending[0].LastError)
	}
}

func TestDispatchTimeoutLeavesTheOperationPendingAndDoesNotWaitForAPluginThatIgnoresTheContext(t *testing.T) {
	f, plugin, d := setup(t)
	release := make(chan struct{})
	plugin.block = release
	t.Cleanup(func() { close(release) })
	d.Immediate = 60 * time.Millisecond

	start := time.Now()
	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)
	elapsed := time.Since(start)

	if elapsed > 3*time.Second {
		t.Fatalf("un plugin que ignora el contexto no debe bloquear más allá del plazo: tardó %s", elapsed)
	}
	if len(report.Pending) != 1 || report.Pending[0].Attempts != 1 {
		t.Fatalf("informe = %+v", report)
	}
	if !strings.Contains(report.Pending[0].LastError, "tiempo agotado") {
		t.Fatalf("el error indica el tiempo agotado: %q", report.Pending[0].LastError)
	}
	ob, _ := readOutbox(f.Folder())
	if len(ob.Operations) != 1 || ob.Operations[0].State != StatePending {
		t.Fatalf("la operación queda pendiente: %+v", ob)
	}
}

func TestDispatchPluginNotListeningGetsNothingAndCreatesNothing(t *testing.T) {
	f, plugin, d := setup(t)
	ev := specEvent("change.start/x")
	ev.Type, ev.Gate = manifest.EventChangeStart, "" // el doble no escucha change.start

	report := d.Dispatch(context.Background(), f.Root, ev, durable)

	if !report.NothingToDo || report.Err != nil {
		t.Fatalf("informe = %+v", report)
	}
	if plugin.calls() != 0 {
		t.Fatal("un plugin que no escucha el evento no recibe nada")
	}
	assertNoIntegrationsFiles(t, f)
}

func TestDispatchDisabledOrAbsentConfigDoesNothing(t *testing.T) {
	f, plugin, d := setup(t)
	d.Config.Plugins[0].Enabled = false
	if report := d.Dispatch(context.Background(), f.Root, specEvent("k"), durable); !report.NothingToDo {
		t.Fatalf("plugin desactivado: %+v", report)
	}
	d.Config = nil
	if report := d.Dispatch(context.Background(), f.Root, specEvent("k"), durable); !report.NothingToDo {
		t.Fatalf("sin configuración: %+v", report)
	}
	if plugin.calls() != 0 {
		t.Fatal("no se entrega nada")
	}
	assertNoIntegrationsFiles(t, f)
}

func assertNoIntegrationsFiles(t *testing.T, f workspaceFixture) {
	t.Helper()
	entries, _ := os.ReadDir(f.Root)
	for _, e := range entries {
		t.Errorf("no debe crearse ningún fichero en la carpeta del cambio: %s", e.Name())
	}
	if fileExists(filepath.Join(f.WS, ".git", "axiom", "integrations")) {
		t.Error("no debe crearse el directorio del cerrojo")
	}
}

func TestDispatchConfiguredButUnregisteredPluginStaysPendingWithoutCountingAnAttempt(t *testing.T) {
	f := newGitWorkspace(t)
	manifest.SetCatalogForTest(t, testManifest())
	SetDefaultRegistryForTest(t) // registro vacío: el plugin está en el catálogo y configurado, pero no registrado
	d := newTestDispatcher(f, "fake")

	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	if len(report.Pending) != 1 || report.Pending[0].Attempts != 0 || report.Pending[0].Reason != ReasonPluginNotConfigured {
		t.Fatalf("informe = %+v", report)
	}
	ob, _ := readOutbox(f.Folder())
	if len(ob.Operations) != 1 || ob.Operations[0].Reason != "plugin_not_configured" || ob.Operations[0].Attempts != 0 || ob.Operations[0].State != StatePending {
		t.Fatalf("cola = %+v", ob)
	}
}

func TestDispatchSecretsAreSanitizedInQueueAndReport(t *testing.T) {
	f := newGitWorkspace(t)
	plugin := newTestPlugin()
	mem := installTestPlugin(t, plugin)
	_ = mem.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(canarySecret))
	d := newTestDispatcher(f, "fake")
	plugin.readToken, plugin.leakToken = true, true

	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	if len(report.Pending) != 1 {
		t.Fatalf("informe = %+v", report)
	}
	for _, text := range []string{report.Pending[0].LastError, readTestFile(t, f.Folder().path(outboxName))} {
		if strings.Contains(text, canarySecret) {
			t.Fatalf("el valor del secreto no puede salir en la cola ni en el informe: %q", text)
		}
	}
	if !strings.Contains(report.Pending[0].LastError, "[REDACTED]") {
		t.Fatalf("last_error = %q", report.Pending[0].LastError)
	}
}

func TestDispatchArchiveDoesNotPersistLinksAndKeepsTheQueueInTheArchivedRoot(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setLinks(Link{Kind: "work_item", ID: "99"})
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	writeTestFile(t, archived.path(LinksFile), validLinksYAML)
	writeTestFile(t, archived.path("proposal.md"), "propuesta\n")
	linksBefore := readTestFile(t, archived.path(LinksFile))

	var queuedInArchivedRoot bool
	plugin.hook = func(Delivery) { queuedInArchivedRoot = fileExists(archived.path(outboxName)) }
	ev := Event{Type: manifest.EventArchive, Change: "x", Actor: "cli", OccurredAt: fixedNow, IdempotencyKey: "archive/x/2026-11-19-x"}

	report := d.Dispatch(context.Background(), archived.Root, ev, DispatchOptions{Durable: true, PersistOutcome: false})

	if report.Err != nil || len(report.Delivered) != 1 {
		t.Fatalf("informe = %+v", report)
	}
	if !queuedInArchivedRoot {
		t.Fatal("la operación de archive se encola en la raíz de la carpeta archivada")
	}
	if len(report.Warnings) == 0 || !strings.Contains(strings.Join(report.Warnings, " "), "descart") {
		t.Fatalf("los vínculos devueltos por archive se descartan con un aviso: %v", report.Warnings)
	}
	if got := readTestFile(t, archived.path(LinksFile)); got != linksBefore {
		t.Fatal("integrations.yaml archivado no cambia")
	}
	if fileExists(archived.path(outboxName)) {
		t.Fatal("la cola archivada se borra al vaciarse")
	}
	entries, _ := os.ReadDir(archived.Root)
	if len(entries) != 2 {
		t.Fatalf("la carpeta archivada conserva solo sus ficheros: %v", entries)
	}
}

func TestDispatchArchiveFailureLeavesTheQueueInTheArchivedRoot(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setFailAlways(errors.New("tracker caído"))
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	ev := Event{Type: manifest.EventArchive, Change: "x", Actor: "cli", OccurredAt: fixedNow, IdempotencyKey: "archive/x/2026-11-19-x"}

	report := d.Dispatch(context.Background(), archived.Root, ev, DispatchOptions{Durable: true})

	if len(report.Pending) != 1 {
		t.Fatalf("informe = %+v", report)
	}
	ob, err := readOutbox(archived)
	if err != nil || len(ob.Operations) != 1 || ob.Operations[0].Key != "archive/x/2026-11-19-x" {
		t.Fatalf("cola archivada = %+v %v", ob, err)
	}
	if report.QueuePath != archived.path(outboxName) {
		t.Fatalf("QueuePath = %q", report.QueuePath)
	}
	if status := runGit(t, f.WS, "status", "--porcelain", "--untracked-files=all"); !strings.Contains(status, "2026-11-19-x/.integrations-outbox.yaml") {
		t.Fatalf("la cola archivada ensucia el árbol (protege la cola en done):\n%s", status)
	}
}

func TestDispatchArchiveWithoutGitDoesNotQueue(t *testing.T) {
	f := newWorkspace(t) // sin git
	plugin := newTestPlugin()
	installTestPlugin(t, plugin)
	d := newTestDispatcher(f, "fake")
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	ev := Event{Type: manifest.EventArchive, Change: "x", Actor: "cli", OccurredAt: fixedNow, IdempotencyKey: "archive/x/2026-11-19-x"}

	report := d.Dispatch(context.Background(), archived.Root, ev, DispatchOptions{Durable: true})

	if !errors.Is(report.Err, ErrArchivedWithoutGit) {
		t.Fatalf("sin git no se encola en la archivada: %+v", report)
	}
	if plugin.calls() != 0 || fileExists(archived.path(outboxName)) {
		t.Fatal("no se entrega ni se crea nada")
	}
}

func TestDispatchInvalidEventAndInvalidQueueAreErrorsNotDeliveries(t *testing.T) {
	f, plugin, d := setup(t)

	if report := d.Dispatch(context.Background(), f.Root, Event{Type: "gate.tasks.approved", Change: "x", IdempotencyKey: "k"}, durable); !errors.Is(report.Err, ErrUnknownEvent) {
		t.Fatalf("evento fuera del catálogo: %+v", report)
	}
	if report := d.Dispatch(context.Background(), f.Root, Event{Type: manifest.EventArchive, Change: "x"}, durable); report.Err == nil {
		t.Fatal("un evento sin clave de idempotencia es un error")
	}

	writeTestFile(t, f.Folder().path(outboxName), "esto: no es una cola\n")
	report := d.Dispatch(context.Background(), f.Root, specEvent("k"), durable)
	if report.Err == nil || !strings.Contains(report.Err.Error(), outboxName) {
		t.Fatalf("una cola inválida es un error que la nombra: %+v", report)
	}
	if plugin.calls() != 0 {
		t.Fatal("con la cola inválida no se entrega nada")
	}
}

func TestDispatchRepeatedKeyAfterDeliveryIsDeliveredAgainButThePluginIsIdempotent(t *testing.T) {
	f, plugin, d := setup(t)
	d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)
	d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	if plugin.calls() != 2 {
		t.Fatalf("entrega al menos una vez: se vuelve a entregar la misma clave (llamadas %d)", plugin.calls())
	}
	if keys := plugin.deliveredKeys(); len(keys) != 1 {
		t.Fatalf("el plugin es idempotente por clave: %v", keys)
	}
}

func TestDispatchNonDurableDoesNotWriteTheQueue(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setFailAlways(errors.New("caído"))
	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), DispatchOptions{Durable: false, PersistOutcome: true})
	if len(report.Pending) != 1 {
		t.Fatalf("informe = %+v", report)
	}
	if fileExists(f.Folder().path(outboxName)) {
		t.Fatal("sin Durable no se escribe la cola")
	}
}

func TestDispatchPersistsLinksBeforeRemovingTheOperation(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setLinks(Link{Kind: "work_item", ID: "1"})

	// Orden de escrituras ante una caída: primero integrations.yaml, después la cola. Cuando se
	// escribe integrations.yaml la operación sigue en la cola: una caída entre ambas reentrega.
	var queuedWhenLinksWritten bool
	original := replaceIntegrationsFile
	replaceIntegrationsFile = func(src, dst string) error {
		if filepath.Base(dst) == LinksFile {
			queuedWhenLinksWritten = fileExists(f.Folder().path(outboxName))
		}
		return original(src, dst)
	}
	t.Cleanup(func() { replaceIntegrationsFile = original })

	report := d.Dispatch(context.Background(), f.Root, specEvent("k1"), durable)

	if report.Err != nil || len(report.Delivered) != 1 {
		t.Fatalf("informe = %+v", report)
	}
	if !queuedWhenLinksWritten {
		t.Fatal("integrations.yaml debe escribirse antes de retirar la operación de la cola")
	}
	if fileExists(f.Folder().path(outboxName)) {
		t.Fatal("y después la cola se retira")
	}
}
