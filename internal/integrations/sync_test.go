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
)

// seedOutbox escribe una cola con las operaciones dadas (plugin fake) en la carpeta, en ese orden.
func seedOutbox(t *testing.T, folder Folder, events ...Event) {
	t.Helper()
	ob := Outbox{Schema: OutboxSchemaV1, Change: folder.Change()}
	for _, ev := range events {
		ob.enqueue("fake", ev, fixedNow)
	}
	if err := writeOutbox(folder, ob); err != nil {
		t.Fatalf("sembrar la cola: %v", err)
	}
}

func eventOf(t manifest.EventType, key string) Event {
	return Event{Type: t, Change: "x", Actor: "cli", OccurredAt: fixedNow, IdempotencyKey: key}
}

func TestSyncRetriesPendingInEnqueueOrderAndRemovesDelivered(t *testing.T) {
	f, plugin, d := setup(t)
	seedOutbox(t, f.Folder(),
		eventOf(manifest.EventGateSpecApproved, "k1"),
		eventOf(manifest.EventGateDesignApproved, "k2"),
		eventOf(manifest.EventArchive, "k3"),
	)

	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if got := plugin.deliveredKeys(); strings.Join(got, ",") != "k1,k2,k3" {
		t.Fatalf("orden de entrega = %v", got)
	}
	if report.Remaining || len(report.Changes) != 1 || len(report.Changes[0].Delivered) != 3 {
		t.Fatalf("informe = %+v", report)
	}
	if fileExists(f.Folder().path(outboxName)) {
		t.Fatal("entregado todo, la cola se borra")
	}
}

func TestSyncAFailureDoesNotStopTheFollowingOperations(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.failEvents[manifest.EventRoleStart] = errors.New("tracker rechaza role.start")
	start := eventOf(manifest.EventRoleStart, "k2")
	start.Role = "backend"
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"), start, eventOf(manifest.EventGateDesignApproved, "k3"))

	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := plugin.deliveredKeys(); strings.Join(got, ",") != "k1,k3" {
		t.Fatalf("un fallo no impide las siguientes: %v", got)
	}
	if !report.Remaining || len(report.Changes[0].Pending) != 1 || report.Changes[0].Pending[0].Key != "k2" || report.Changes[0].Pending[0].Attempts != 1 {
		t.Fatalf("informe = %+v", report)
	}
	ob, _ := readOutbox(f.Folder())
	if len(ob.Operations) != 1 || ob.Operations[0].Key != "k2" {
		t.Fatalf("queda solo la fallida: %+v", ob.Operations)
	}
}

func TestSyncRetryAbandonedResetsAttempts(t *testing.T) {
	f, plugin, d := setup(t)
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"))
	ob, _ := readOutbox(f.Folder())
	ob.Operations[0].State, ob.Operations[0].Attempts, ob.Operations[0].LastError = StateAbandoned, MaxAttempts, "sigue caído"
	if err := writeOutbox(f.Folder(), ob); err != nil {
		t.Fatal(err)
	}

	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil || plugin.calls() != 0 || !report.Remaining || len(report.Changes[0].Abandoned) != 1 {
		t.Fatalf("sin la bandera una abandonada no se reintenta: calls=%d report=%+v err=%v", plugin.calls(), report, err)
	}

	report, err = d.Sync(context.Background(), SyncOptions{RetryAbandoned: true})
	if err != nil || report.Remaining || len(report.Changes[0].Delivered) != 1 {
		t.Fatalf("--retry-abandoned la reactiva y entrega: %+v %v", report, err)
	}
}

func TestSyncRetryAbandonedThatFailsAgainCountsFromZero(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setFailAlways(errors.New("caído"))
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"))
	ob, _ := readOutbox(f.Folder())
	ob.Operations[0].State, ob.Operations[0].Attempts = StateAbandoned, MaxAttempts
	_ = writeOutbox(f.Folder(), ob)

	report, err := d.Sync(context.Background(), SyncOptions{RetryAbandoned: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Changes[0].Pending; len(got) != 1 || got[0].Attempts != 1 {
		t.Fatalf("tras reactivarla vuelve a contar desde 0: %+v", report.Changes[0])
	}
}

func TestSyncDiscardRemovesAPendingOrAbandonedOperationWithoutDelivering(t *testing.T) {
	f, plugin, d := setup(t)
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"), eventOf(manifest.EventGateDesignApproved, "k2"), eventOf(manifest.EventArchive, "k3"))
	ob, _ := readOutbox(f.Folder())
	ob.Operations[1].State = StateAbandoned
	_ = writeOutbox(f.Folder(), ob)

	report, err := d.Sync(context.Background(), SyncOptions{Discard: []string{"k1", "k2", "no-existe"}})
	if err != nil {
		t.Fatal(err)
	}
	change := report.Changes[0]
	if len(change.Discarded) != 2 || change.Discarded[0].Key != "k1" || change.Discarded[1].Key != "k2" {
		t.Fatalf("Discarded = %+v", change.Discarded)
	}
	if got := plugin.deliveredKeys(); len(got) != 1 || got[0] != "k3" {
		t.Fatalf("lo descartado no se entrega: %v", got)
	}
	if !strings.Contains(strings.Join(change.Warnings, " "), "no-existe") {
		t.Fatalf("una clave sin operación se avisa: %v", change.Warnings)
	}
}

func TestSyncPerOperationTimeout(t *testing.T) {
	f, plugin, d := setup(t)
	release := make(chan struct{})
	plugin.block = release
	t.Cleanup(func() { close(release) })
	d.PerOp = 60 * time.Millisecond
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"))

	start := time.Now()
	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("el plazo por operación debe cortar a un plugin que ignora el contexto")
	}
	if got := report.Changes[0].Pending; len(got) != 1 || !strings.Contains(got[0].LastError, "tiempo agotado") {
		t.Fatalf("informe = %+v", report.Changes[0])
	}
}

func TestSyncWithChangeTreatsTheOrdinaryFolderAndTheMostRecentArchivedOne(t *testing.T) {
	f, plugin, d := setup(t)
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k-ordinaria"))
	older := archivedFolderOf(t, f, "2026-10-01-x")
	newer := archivedFolderOf(t, f, "2026-11-19-x")
	seedOutbox(t, older, eventOf(manifest.EventArchive, "k-vieja"))
	seedOutbox(t, newer, eventOf(manifest.EventArchive, "k-nueva"))
	other := archivedFolderOf(t, f, "2026-11-19-y")
	seedOutbox(t, other, eventOf(manifest.EventArchive, "k-otro-cambio"))

	report, err := d.Sync(context.Background(), SyncOptions{Change: "x"})
	if err != nil {
		t.Fatal(err)
	}
	got := plugin.deliveredKeys()
	if strings.Join(got, ",") != "k-ordinaria,k-nueva" {
		t.Fatalf("con --change solo la ordinaria y la archivada más reciente: %v", got)
	}
	if len(report.Changes) != 2 || report.Changes[0].Archived || !report.Changes[1].Archived {
		t.Fatalf("informe = %+v", report.Changes)
	}
	if !fileExists(older.path(outboxName)) || !fileExists(other.path(outboxName)) {
		t.Fatal("las demás carpetas archivadas no se tocan")
	}
}

func TestSyncWithoutChangeTreatsEveryFolderWithAQueue(t *testing.T) {
	f, plugin, d := setup(t)
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k-x"))
	y := Folder{WorkspaceRoot: f.WS, Root: filepath.Join(f.WS, "openspec", "changes", "y")}
	if err := os.MkdirAll(y.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	yEvent := eventOf(manifest.EventGateSpecApproved, "k-y")
	yEvent.Change = "y"
	seedOutbox(t, y, yEvent)
	archived := archivedFolderOf(t, f, "2026-11-19-z")
	seedOutbox(t, archived, eventOf(manifest.EventArchive, "k-z"))
	quiet := Folder{WorkspaceRoot: f.WS, Root: filepath.Join(f.WS, "openspec", "changes", "w")}
	if err := os.MkdirAll(quiet.Root, 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 3 {
		t.Fatalf("solo las carpetas con cola (3), obtuvo %+v", report.Changes)
	}
	if got := plugin.deliveredKeys(); len(got) != 3 {
		t.Fatalf("entregadas = %v", got)
	}
	for _, folder := range []Folder{f.Folder(), y, archived} {
		if fileExists(folder.path(outboxName)) {
			t.Errorf("la cola de %s debe borrarse", folder.Root)
		}
	}
}

func TestSyncInArchivedFolderDoesNotMergeLinksAndWritesOnlyTheQueue(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setLinks(Link{Kind: "work_item", ID: "99"})
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	writeTestFile(t, archived.path(LinksFile), validLinksYAML)
	before := readTestFile(t, archived.path(LinksFile))
	seedOutbox(t, archived, eventOf(manifest.EventArchive, "archive/x/2026-11-19-x"))

	var written []string
	original := replaceIntegrationsFile
	replaceIntegrationsFile = func(src, dst string) error { written = append(written, filepath.Base(dst)); return original(src, dst) }
	t.Cleanup(func() { replaceIntegrationsFile = original })

	report, err := d.Sync(context.Background(), SyncOptions{Change: "x"})
	if err != nil || report.Remaining {
		t.Fatalf("Sync: %+v %v", report, err)
	}
	if got := readTestFile(t, archived.path(LinksFile)); got != before {
		t.Fatal("integrations.yaml archivado no cambia")
	}
	for _, name := range written {
		if name != outboxName {
			t.Errorf("en la archivada solo se reemplaza la cola, se reemplazó %s", name)
		}
	}
	if len(report.Changes[0].Warnings) == 0 {
		t.Fatal("los vínculos descartados se avisan")
	}
}

func TestSyncReportsRemainingWhenAnythingIsPendingOrAbandoned(t *testing.T) {
	f, plugin, d := setup(t)
	plugin.setFailAlways(errors.New("caído"))
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"))
	report, _ := d.Sync(context.Background(), SyncOptions{})
	if !report.Remaining {
		t.Fatal("con algo pendiente queda algo")
	}
	plugin.recover()
	report, _ = d.Sync(context.Background(), SyncOptions{})
	if report.Remaining {
		t.Fatal("entregado todo, no queda nada")
	}
}

func TestSyncDeliversAtLeastOnceWhenTheRemovalWasNeverWritten(t *testing.T) {
	f, plugin, d := setup(t)
	// El plugin ya entregó k1 (idempotente por clave) pero la cola conserva la operación.
	if _, err := plugin.Deliver(context.Background(), Delivery{Event: eventOf(manifest.EventGateSpecApproved, "k1")}); err != nil {
		t.Fatal(err)
	}
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"))

	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil || report.Remaining {
		t.Fatalf("Sync: %+v %v", report, err)
	}
	if plugin.calls() != 2 {
		t.Fatalf("Sync reentrega la misma clave (llamadas %d)", plugin.calls())
	}
	if len(plugin.deliveredKeys()) != 1 {
		t.Fatal("el plugin, idempotente, no duplica")
	}
	if fileExists(f.Folder().path(outboxName)) {
		t.Fatal("y la retira")
	}
}

func TestSyncWithoutQueueOrWithoutConfigurationCreatesNothing(t *testing.T) {
	f, plugin, d := setup(t)
	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil || report.Remaining || len(report.Changes) != 0 {
		t.Fatalf("sin cola: %+v %v", report, err)
	}
	report, err = d.Sync(context.Background(), SyncOptions{Change: "x"})
	if err != nil || report.Remaining || len(report.Changes) != 0 {
		t.Fatalf("sin cola y con --change: %+v %v", report, err)
	}
	if plugin.calls() != 0 {
		t.Fatal("nada que entregar")
	}
	assertNoIntegrationsFiles(t, f)
}

func TestSyncOperationWhoseCapabilityIsNoLongerConfiguredStaysPendingWithoutAnAttempt(t *testing.T) {
	f, plugin, d := setup(t)
	seedOutbox(t, f.Folder(), eventOf(manifest.EventGateSpecApproved, "k1"))
	d.Config.Plugins[0].Enabled = false

	report, err := d.Sync(context.Background(), SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got := report.Changes[0].Pending
	if len(got) != 1 || got[0].Reason != ReasonPluginNotConfigured || got[0].Attempts != 0 || !report.Remaining {
		t.Fatalf("informe = %+v", report)
	}
	if plugin.calls() != 0 {
		t.Fatal("no se entrega nada")
	}
	ob, _ := readOutbox(f.Folder())
	if ob.Operations[0].Reason != "plugin_not_configured" {
		t.Fatalf("la cola recoge el motivo: %+v", ob.Operations[0])
	}
}

func TestSyncInvalidQueueIsAnError(t *testing.T) {
	f, _, d := setup(t)
	writeTestFile(t, f.Folder().path(outboxName), "esto: no es una cola\n")
	if _, err := d.Sync(context.Background(), SyncOptions{}); err == nil || !strings.Contains(err.Error(), outboxName) {
		t.Fatalf("una cola inválida es un error que la nombra: %v", err)
	}
}
