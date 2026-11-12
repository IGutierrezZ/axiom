package techcontext

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/specbaseline"
)

// Pruebas cruzadas D2/D3 (REQ-30.18): `techcontext` y `specbaseline` conviven
// durante U1–U5 sin ventana rota: el plan que escribe uno lo verifica el otro, con
// el mismo `integration.yaml`. Se retiran en U6, cuando `specbaseline` desaparece.

func (f *fixture) baselineTarget(t *testing.T) specbaseline.ChangeTarget {
	t.Helper()
	target := f.target(t)
	return specbaseline.ChangeTarget{
		Location:      target.Location,
		Change:        target.Change,
		ChangeRoot:    target.ChangeRoot,
		AllowedRoots:  target.AllowedRoots,
		ArtifactStore: target.ArtifactStore,
	}
}

func TestCrossAPlanWrittenBySddIntegratePrepareIsVerifiedByArchiveContext(t *testing.T) {
	f := newFixture(t, true)
	plan, report, err := specbaseline.Prepare(context.Background(), f.baselineTarget(t), specbaseline.PlanOptions{})
	if err != nil || len(report.Errors) != 0 {
		t.Fatalf("specbaseline.Prepare: %v %v", err, report.Errors)
	}
	trace, err := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
	if err != nil {
		t.Fatalf("ReadTrace no lee el plan de sdd integrate: %v", err)
	}
	if trace.InputsSHA256 != plan.InputsSHA256 || len(trace.Entries) != 2 {
		t.Fatalf("traza = %+v, plan = %+v", trace, plan)
	}

	f.applyEdits(t)
	verified := f.verify(t)
	if !verified.Passed {
		t.Fatalf("archive context verify debe aceptar el plan de sdd integrate prepare: %+v", verified.Violations)
	}
	after, err := specbaseline.ReadPlan(filepath.Join(f.changeRoot, TraceFileName))
	if err != nil {
		t.Fatalf("el plan que reescribe archive context no se lee con sdd integrate: %v", err)
	}
	if after.VerifiedAt == nil || after.Entries[0].State != specbaseline.StateIntegrated {
		t.Errorf("plan verificado = %+v", after)
	}
}

func TestCrossAPlanWrittenByArchiveContextIsVerifiedBySddIntegrate(t *testing.T) {
	f := newFixture(t, true)
	if _, report := prepare(t, f); len(report.Errors) != 0 {
		t.Fatal(report.Errors)
	}
	if _, err := specbaseline.ReadPlan(filepath.Join(f.changeRoot, TraceFileName)); err != nil {
		t.Fatalf("sdd integrate no lee el plan de archive context: %v", err)
	}
	f.applyEdits(t)
	verified, err := specbaseline.Verify(context.Background(), f.baselineTarget(t))
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Passed {
		t.Fatalf("sdd integrate verify debe aceptar el plan de archive context prepare: %+v", verified.Violations)
	}
	trace, err := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
	if err != nil || trace.VerifiedAt == nil {
		t.Fatalf("ReadTrace tras la verificación de sdd integrate: %+v, %v", trace, err)
	}
}

// El mismo cambio produce el mismo plan con cualquiera de los dos verbos: la
// conducta de contexto es idéntica salvo la hora de preparación.
func TestCrossBothVerbsPrepareTheSamePlan(t *testing.T) {
	a := newFixture(t, true)
	b := newFixture(t, true)
	theirs, report, err := specbaseline.Prepare(context.Background(), a.baselineTarget(t), specbaseline.PlanOptions{})
	if err != nil || len(report.Errors) != 0 {
		t.Fatalf("specbaseline.Prepare: %v %v", err, report.Errors)
	}
	ours, _ := prepare(t, b)

	if ours.InputsSHA256 != theirs.InputsSHA256 {
		t.Errorf("inputs_sha256 = %s, sdd integrate escribe %s", ours.InputsSHA256, theirs.InputsSHA256)
	}
	if ours.Context.Decision != theirs.Context.Decision || ours.Context.Reason != theirs.Context.Reason {
		t.Errorf("decisión = %+v frente a %+v", ours.Context, theirs.Context)
	}
	var ourEntries, theirEntries []string
	for _, entry := range ours.Entries {
		ourEntries = append(ourEntries, strings.Join([]string{entry.ID, string(entry.Kind), entry.Document, entry.Anchor, string(entry.Action), string(entry.State), entry.Reason}, "|"))
	}
	for _, entry := range theirs.Entries {
		theirEntries = append(theirEntries, strings.Join([]string{entry.ID, string(entry.Kind), entry.Document, entry.Anchor, string(entry.Action), string(entry.State), entry.Reason}, "|"))
	}
	if !reflect.DeepEqual(ourEntries, theirEntries) {
		t.Errorf("entradas = %v frente a %v", ourEntries, theirEntries)
	}
	var ourRepos, theirRepos []string
	for _, repo := range ours.Repositories {
		ourRepos = append(ourRepos, repo.Repo)
	}
	for _, repo := range theirs.Repositories {
		theirRepos = append(theirRepos, repo.Repo)
	}
	if !reflect.DeepEqual(ourRepos, theirRepos) {
		t.Errorf("repositorios = %v frente a %v", ourRepos, theirRepos)
	}
}

// `show` da los mismos bytes que `sdd integrate show` para el mismo plan.
func TestCrossRenderTraceMatchesSddIntegrateShow(t *testing.T) {
	f := newFixture(t, true)
	plan, report, err := specbaseline.Prepare(context.Background(), f.baselineTarget(t), specbaseline.PlanOptions{})
	if err != nil || len(report.Errors) != 0 {
		t.Fatalf("specbaseline.Prepare: %v %v", err, report.Errors)
	}
	f.applyEdits(t)
	if verified, err := specbaseline.Verify(context.Background(), f.baselineTarget(t)); err != nil || !verified.Passed {
		t.Fatalf("Verify: %v %+v", err, verified.Violations)
	}
	plan, err = specbaseline.ReadPlan(filepath.Join(f.changeRoot, TraceFileName))
	if err != nil {
		t.Fatal(err)
	}
	trace, err := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{TraceFormatMarkdown, TraceFormatJSON} {
		want, err := specbaseline.RenderTrace(plan, format)
		if err != nil {
			t.Fatal(err)
		}
		got, err := RenderTrace(trace, format)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("formato %s:\n%s\n--- frente a sdd integrate show ---\n%s", format, got, want)
		}
	}
	// El plan reescrito por archive context sigue siendo byte a byte el de un plan v1.
	rewritten, _ := os.ReadFile(filepath.Join(f.changeRoot, TraceFileName))
	again, err := MarshalTrace(trace)
	if err != nil || string(again) != string(rewritten) {
		t.Errorf("la serialización de la traza no reproduce el plan escrito")
	}
	original, _ := specbaseline.MarshalPlan(plan)
	if string(original) != string(again) {
		t.Errorf("MarshalTrace y MarshalPlan difieren:\n%s\n---\n%s", again, original)
	}
}
