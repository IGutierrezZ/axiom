package sddstatus

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Estas pruebas sustituyen la variable global publishedMarksReader: no llevan t.Parallel.

func fakePublished(t *testing.T, content string, refusal *branch.Refusal) {
	t.Helper()
	publishedMarksReader = func(_ context.Context, _ branch.PublicationRequest, name string) (branch.PublishedChangeFile, *branch.Refusal) {
		if name != marks.FileName {
			t.Errorf("se pidió %q, se esperaba %q", name, marks.FileName)
		}
		if refusal != nil {
			return branch.PublishedChangeFile{}, refusal
		}
		return branch.PublishedChangeFile{
			Target: branch.PublicationTarget{Branch: "main"}, RemoteRefExists: true,
			Exists: content != "", Content: []byte(content),
		}, nil
	}
	t.Cleanup(func() { publishedMarksReader = nil })
}

const marksBattery = "REQ-1.1#A; REQ-1.1#B"

func marksChange(t *testing.T) (master, changeRoot, digest string) {
	t.Helper()
	ws, changeRoot := acceptanceChange(t, acceptanceBlockOn)
	write(t, filepath.Join(changeRoot, "pruebas.md"), acceptanceHeader+
		acceptanceRow("P-01", marksBattery, "dev; pre")+
		acceptanceRow("P-02", "REQ-1.2#C", "dev"))
	digest, err := kickoff.ArtifactDigest([]string{filepath.Join(changeRoot, "pruebas.md")})
	if err != nil {
		t.Fatal(err)
	}
	return ws.Master, changeRoot, digest
}

func marksEntry(caseID, env string, result marks.Result, at, digest string) marks.Entry {
	return marks.Entry{Case: caseID, Environment: env, Result: result, By: "Ana <ana@example.org>", At: at, PlanDigest: digest}
}

func writeMarks(t *testing.T, changeRoot string, entries ...marks.Entry) string {
	t.Helper()
	var data []byte
	for _, e := range entries {
		var err error
		if data, err = marks.Append(data, e); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(changeRoot, marks.FileName), string(data))
	return string(data)
}

func marksOf(t *testing.T, master string) *AcceptanceMarksProjection {
	t.Helper()
	got := acceptanceOf(t, resolveFrom(t, master, "inc-demo"))
	if got == nil {
		t.Fatal("falta acceptanceTests")
	}
	return got.Marks
}

func TestAcceptanceMarksProjectsSummaryAndPending(t *testing.T) {
	master, changeRoot, digest := marksChange(t)
	local := writeMarks(t, changeRoot,
		marksEntry("P-01", "dev", marks.ResultOK, "2026-11-26T10:00:00.000Z", digest),
		marksEntry("P-02", "dev", marks.ResultOK, "2026-11-26T10:00:01.000Z", strings.Repeat("a", 64)),
	)
	fakePublished(t, "", nil)
	got := marksOf(t, master)
	if got == nil || got.Path != filepath.Join(changeRoot, marks.FileName) {
		t.Fatalf("Marks = %+v", got)
	}
	if want := "dev: 1/2 ok, 1 caducada; pre: 0/1 ok, 1 sin marcar"; got.Summary != want {
		t.Fatalf("summary = %q, quería %q", got.Summary, want)
	}
	if got.Pending == nil || *got.Pending != 2 || got.PendingUnavailable != "" || got.Error != "" || len(got.Environments) != 2 {
		t.Fatalf("Marks = %+v", got)
	}

	// Con todo ya publicado no hay pendientes, pero el cero sigue proyectándose.
	fakePublished(t, local, nil)
	if got = marksOf(t, master); got.Pending == nil || *got.Pending != 0 {
		t.Fatalf("pending = %+v", got.Pending)
	}
}

func TestAcceptanceMarksPendingUnavailableKeepsTheSummary(t *testing.T) {
	master, changeRoot, digest := marksChange(t)
	writeMarks(t, changeRoot, marksEntry("P-01", "dev", marks.ResultKO, "2026-11-26T10:00:00.000Z", digest))
	fakePublished(t, "", &branch.Refusal{Code: branch.CodeWrongBranch, Message: "m", Continuation: "c"})
	got := marksOf(t, master)
	if got == nil || got.Pending != nil || got.PendingUnavailable != string(branch.CodeWrongBranch) || !strings.HasPrefix(got.Summary, "dev: 0/2 ok, 1 ko") {
		t.Fatalf("Marks = %+v", got)
	}
	result := AcceptanceMarksFor(context.Background(), locateMaster(t, master), changeRoot, "inc-demo",
		AcceptanceTestsFor(locateMaster(t, master), changeRoot))
	if result.PendingRefusal == nil || result.PendingRefusal.Continuation != "c" || result.PendingKnown {
		t.Fatalf("resultado = %+v", result)
	}
}

func TestAcceptanceMarksAbsentWithoutFileOrWithInactiveFunction(t *testing.T) {
	master, changeRoot, _ := marksChange(t)
	fakePublished(t, "", nil)
	if got := marksOf(t, master); got != nil {
		t.Fatalf("sin fichero no hay marks: %+v", got)
	}

	ws, inactiveRoot := acceptanceChange(t, acceptanceBlockOff)
	write(t, filepath.Join(inactiveRoot, "pruebas.md"), acceptanceHeader+acceptanceRow("P-01", marksBattery, "dev"))
	writeMarks(t, inactiveRoot, marksEntry("P-01", "dev", marks.ResultOK, "2026-11-26T10:00:00.000Z", strings.Repeat("b", 64)))
	if got := marksOf(t, ws.Master); got != nil {
		t.Fatalf("con la función inactiva no hay marks: %+v", got)
	}
	_ = changeRoot
}

func TestAcceptanceMarksEmptyOrOnlyInvalidEntriesProjectNothing(t *testing.T) {
	master, changeRoot, _ := marksChange(t)
	fakePublished(t, "", nil)
	write(t, filepath.Join(changeRoot, marks.FileName), "")
	if got := marksOf(t, master); got != nil {
		t.Fatalf("fichero vacío: %+v", got)
	}
	write(t, filepath.Join(changeRoot, marks.FileName), "- case: P-01\n  environment: dev\n  result: maybe\n")
	if got := marksOf(t, master); got != nil {
		t.Fatalf("solo entradas inválidas: %+v", got)
	}
}

func TestAcceptanceMarksUnreadableFileReportsErrorOnly(t *testing.T) {
	master, changeRoot, _ := marksChange(t)
	fakePublished(t, "", nil)
	write(t, filepath.Join(changeRoot, marks.FileName), "- case: [\n")
	got := marksOf(t, master)
	if got == nil || got.Error == "" || got.Summary != "" || got.Pending != nil || len(got.Environments) != 0 {
		t.Fatalf("Marks = %+v", got)
	}
}

func TestAcceptanceMarksInvalidEntriesAreCounted(t *testing.T) {
	master, changeRoot, digest := marksChange(t)
	fakePublished(t, "", nil)
	good := writeMarks(t, changeRoot, marksEntry("P-01", "dev", marks.ResultOK, "2026-11-26T10:00:00.000Z", digest))
	write(t, filepath.Join(changeRoot, marks.FileName), good+"- case: P-02\n  environment: dev\n  result: maybe\n")
	got := marksOf(t, master)
	if got == nil || got.Invalid != 1 || !strings.HasPrefix(got.Summary, "dev: 1/2 ok") {
		t.Fatalf("Marks = %+v", got)
	}
}

// Las marcas son informativas: el enrutado, las dependencias, los motivos, las notas y las puertas
// (con sus digests) son idénticos con y sin fichero.
func TestAcceptanceMarksNeverChangeRoutingNotesOrDigests(t *testing.T) {
	master, changeRoot, digest := marksChange(t)
	fakePublished(t, "", nil)
	without := resolveWithInstructions(t, master, "inc-demo")
	writeMarks(t, changeRoot, marksEntry("P-01", "dev", marks.ResultOK, "2026-11-26T10:00:00.000Z", digest))
	with := resolveWithInstructions(t, master, "inc-demo")

	if got := acceptanceOf(t, with); got == nil || got.Marks == nil {
		t.Fatal("se esperaban marks")
	}
	if with.NextRecommended != without.NextRecommended || !reflect.DeepEqual(with.Dependencies, without.Dependencies) ||
		!reflect.DeepEqual(with.BlockedReasons, without.BlockedReasons) || !reflect.DeepEqual(with.Notes, without.Notes) ||
		!reflect.DeepEqual(with.Governance, without.Governance) {
		t.Fatal("pruebas-estado.yaml cambió el enrutado, las notas o las puertas")
	}
	for _, path := range TasksGateArtifactPaths(changeRoot, nil) {
		if strings.HasSuffix(path, marks.FileName) {
			t.Fatalf("%s no puede formar parte del digest de la puerta tasks", path)
		}
	}
}

// El dashboard y el status comparten resultado: el texto del resumen es el mismo.
func TestAcceptanceMarksForMatchesTheProjection(t *testing.T) {
	master, changeRoot, digest := marksChange(t)
	fakePublished(t, "", nil)
	writeMarks(t, changeRoot, marksEntry("P-01", "dev", marks.ResultOK, "2026-11-26T10:00:00.000Z", digest))
	location := locateMaster(t, master)
	result := AcceptanceMarksFor(context.Background(), location, changeRoot, "inc-demo", AcceptanceTestsFor(location, changeRoot))
	if !result.Applicable || result.Digest != digest || result.Remote != "origin" || result.Branch != "main" || len(result.Pending) != 1 {
		t.Fatalf("resultado = %+v", result)
	}
	if got := marksOf(t, master); got == nil || got.Summary != result.Summary.Text() {
		t.Fatalf("la proyección %+v difiere del resumen %q", got, result.Summary.Text())
	}
}

func locateMaster(t *testing.T, master string) workspace.Location {
	t.Helper()
	location, err := workspace.Locate(master)
	if err != nil {
		t.Fatalf("Locate(%q): %v", master, err)
	}
	return location
}
