package integrations

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTrackerNotesAreAppendOnlyAndOrdered(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()

	if notes, err := TrackerNotes(folder); err != nil || len(notes) != 0 {
		t.Fatalf("sin diario: %v %v", notes, err)
	}
	events := []TrackerNote{
		{At: fixedNow, Verb: "summary push", Code: "tracker_rejected", Message: "sin edición", Continuation: "axiom integrations summary push --change x"},
		{At: fixedNow.Add(time.Minute), Verb: "time record", Role: "be", Code: "time_pending"},
		{At: fixedNow.Add(2 * time.Minute), Verb: "summary push", Code: "done"},
	}
	for _, n := range events {
		if err := AppendTrackerNote(folder, n); err != nil {
			t.Fatal(err)
		}
	}
	got, err := TrackerNotes(folder)
	if err != nil || len(got) != 3 {
		t.Fatalf("notes = %+v, %v", got, err)
	}
	for i, n := range events {
		if got[i].Code != n.Code || got[i].Verb != n.Verb || !got[i].At.Equal(n.At) || got[i].At.Location() != time.UTC {
			t.Fatalf("suceso %d = %+v, se esperaba %+v", i, got[i], n)
		}
	}
	if got[0].Continuation == "" || got[1].Role != "be" {
		t.Fatalf("se conservan rol y continuación: %+v", got)
	}

	// Solo se añade: los sucesos anteriores quedan intactos y en su orden.
	if err := AppendTrackerNote(folder, TrackerNote{At: fixedNow.Add(time.Hour), Verb: "time propose", Code: "done"}); err != nil {
		t.Fatal(err)
	}
	after, _ := TrackerNotes(folder)
	if len(after) != 4 || after[3].Verb != "time propose" {
		t.Fatalf("el nuevo suceso va al final: %+v", after)
	}
	for i := range got {
		if after[i] != got[i] {
			t.Fatalf("el suceso %d cambió: %+v -> %+v", i, got[i], after[i])
		}
	}
}

func TestAppendTrackerNoteValidatesAndStampsTime(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	if err := AppendTrackerNote(folder, TrackerNote{Verb: "x"}); err == nil {
		t.Fatal("el suceso exige verb y code")
	}
	if fileExists(folder.path(TrackerNotesFile)) {
		t.Fatal("un suceso inválido no escribe")
	}
	if err := AppendTrackerNote(folder, TrackerNote{Verb: "tasks create", Code: "done"}); err != nil {
		t.Fatal(err)
	}
	notes, _ := TrackerNotes(folder)
	if len(notes) != 1 || notes[0].At.IsZero() {
		t.Fatalf("la hora vacía se completa: %+v", notes)
	}
}

func TestTrackerNotesUnderTheChangeLockAreNotLost(t *testing.T) {
	f := newGitWorkspace(t)
	folder := f.Folder()
	const writers = 8
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := AppendTrackerNote(folder, TrackerNote{Verb: "time record", Code: "done"}); err != nil {
				t.Errorf("añadir: %v", err)
			}
		}()
	}
	wg.Wait()
	notes, err := TrackerNotes(folder)
	if err != nil || len(notes) != writers {
		t.Fatalf("ningún suceso se pierde: %d de %d (%v)", len(notes), writers, err)
	}
}

func TestTrackerNotesRejectCorruptAndArchivedFolders(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	writeTestFile(t, folder.path(TrackerNotesFile), "schema: axiom.integrations-tracker-notes/v9\nchange: x\n")
	if _, err := TrackerNotes(folder); err == nil || !strings.Contains(err.Error(), "v9") {
		t.Fatalf("esquema ajeno: %v", err)
	}
	archived := archivedFolderOf(t, newGitWorkspace(t), "2026-11-19-x")
	if err := AppendTrackerNote(archived, TrackerNote{Verb: "summary push", Code: "done"}); err == nil || !strings.Contains(err.Error(), "RF-ARC-03") {
		t.Fatalf("la archivada es inmutable: %v", err)
	}
}
