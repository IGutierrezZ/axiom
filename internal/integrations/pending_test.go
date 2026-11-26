package integrations

import (
	"strings"
	"sync"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/docpaths"
)

func samplePending(role, item string) PendingTime {
	return PendingTime{
		Role: role, ItemID: item, Hours: 1.5, Minutes: 90, Actor: "ana", Date: "2026-11-18",
		Source: "role.close/" + role + "/1", BaseRev: 4, ExpectedCompleted: 3.5, ExpectedRemaining: 2,
	}
}

func TestPendingTimeRoundTripOnePerRole(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()

	if _, ok, err := PendingTimeFor(folder, "be"); err != nil || ok {
		t.Fatalf("sin diario: ok=%v err=%v", ok, err)
	}
	if err := PutPendingTime(folder, samplePending("be", "205")); err != nil {
		t.Fatal(err)
	}
	if err := PutPendingTime(folder, samplePending("fe", "300")); err != nil {
		t.Fatal(err)
	}
	// Sustituir la del mismo rol no duplica.
	if err := PutPendingTime(folder, samplePending("be", "206")); err != nil {
		t.Fatal(err)
	}
	got, ok, err := PendingTimeFor(folder, "be")
	if err != nil || !ok || got.ItemID != "206" || got.BaseRev != 4 || got.ExpectedCompleted != 3.5 {
		t.Fatalf("be = %+v ok=%v err=%v", got, ok, err)
	}
	doc, _ := readPending(folder)
	if len(doc.Pending) != 2 || doc.Schema != PendingTimeSchemaV1 {
		t.Fatalf("un elemento por rol: %+v", doc)
	}

	if err := ClearPendingTime(folder, "be"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := PendingTimeFor(folder, "be"); ok {
		t.Fatal("be debe haberse borrado")
	}
	if _, ok, _ := PendingTimeFor(folder, "fe"); !ok {
		t.Fatal("fe se conserva")
	}
	// Borrar sin intención no hace nada; borrar la última retira el fichero.
	if err := ClearPendingTime(folder, "be"); err != nil {
		t.Fatal(err)
	}
	if err := ClearPendingTime(folder, "fe"); err != nil {
		t.Fatal(err)
	}
	if fileExists(folder.path(PendingTimeFile)) {
		t.Fatal("con el diario vacío se retira el fichero")
	}
}

func TestPendingTimeValidatesAndRejectsCorruptFiles(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	bad := samplePending("be", "205")
	bad.Minutes = 0
	if err := PutPendingTime(folder, bad); err == nil {
		t.Fatal("una intención sin minutos se rechaza")
	}
	if fileExists(folder.path(PendingTimeFile)) {
		t.Fatal("nada se escribe si es inválida")
	}
	writeTestFile(t, folder.path(PendingTimeFile), "schema: axiom.integrations-time-pending/v9\nchange: x\n")
	if _, _, err := PendingTimeFor(folder, "be"); err == nil || !strings.Contains(err.Error(), "v9") {
		t.Fatalf("esquema ajeno: %v", err)
	}
}

func TestPendingTimeWritesAreSerializedByTheChangeLock(t *testing.T) {
	f := newGitWorkspace(t)
	folder := f.Folder()
	roles := []string{"a", "b", "c", "d", "e", "f"}
	var wg sync.WaitGroup
	for _, r := range roles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := PutPendingTime(folder, samplePending(r, "7")); err != nil {
				t.Errorf("rol %s: %v", r, err)
			}
		}()
	}
	wg.Wait()
	doc, err := readPending(folder)
	if err != nil || len(doc.Pending) != len(roles) {
		t.Fatalf("ninguna escritura concurrente se pierde: %d de %d (%v)", len(doc.Pending), len(roles), err)
	}
}

func TestPendingTimeInArchivedFolderIsRefused(t *testing.T) {
	f := newGitWorkspace(t)
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	if err := PutPendingTime(archived, samplePending("be", "1")); err == nil || !strings.Contains(err.Error(), "RF-ARC-03") {
		t.Fatalf("la archivada es inmutable: %v", err)
	}
}

func TestLocalJournalsAreLocalStateAndNotIntegrationsTemporaries(t *testing.T) {
	set, err := docpaths.New("openspec", "x", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{
		PendingTimeFile, TrackerNotesFile,
		strings.Replace(pendingTempPattern, "*", "123", 1),
		strings.Replace(notesTempPattern, "*", "123", 1),
	}
	for _, name := range names {
		if !set.LocalState("openspec/changes/x/" + name) {
			t.Errorf("%s debe ser estado local no publicado", name)
		}
		if strings.HasPrefix(name, ".integrations-") {
			t.Errorf("%s no debe casar con .integrations-*.yaml", name)
		}
	}
}
