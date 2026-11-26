package reqtrace

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func collectFixture() (Input, *fakeHistory) {
	h := &fakeHistory{
		line: []string{"c1", "c2", "c3", "c4", "c5"},
		added: map[string][]string{
			"openspec/changes/alta/verify-report.md":                   {"c1"},
			"openspec/changes/archive/2026-01-01-alta/kickoff.yaml":    {"c2"},
			"openspec/changes/arreglo/verify-report.md":                {"c3"},
			"openspec/changes/archive/2026-02-01-arreglo/kickoff.yaml": {"c4"},
			"odd/tasks/cobro.md":                                       {"c5"},
		},
		entries: map[string][]string{
			"c1|openspec/changes/alta":    {"tasks.md"},
			"c3|openspec/changes/arreglo": {"tasks.md"},
		},
	}
	specs := fstest.MapFS{
		"changes/archive/2026-01-01-alta/specs/pagos/spec.md": deltaSpec("ADDED", "Cobro (REQ-30.1)"),
		"changes/archive/2026-01-01-alta/ids.yaml":            {Data: []byte(validTrace)},
		"changes/archive/2026-02-01-arreglo/tasks.md":         {Data: []byte("- [x] 1.1 x\n  Cubre: REQ-30.1\n")},
		"changes/archive/2026-02-01-arreglo/proposal.md":      {Data: []byte("# P\n\n## Informe de bug\n\nObservado: x\nEsperado: y\n")},
	}
	odd := fstest.MapFS{"cobro.md": oddDocText("", strings.ReplaceAll(oddSpecViva, "REQ-1.1", "REQ-30.1"))}
	return Input{
		Pattern: reqid.Default(),
		Ref:     "main",
		SDD:     Source{Files: specs, GitPrefix: "openspec", History: h, Repo: "r"},
		ODD:     Source{Files: odd, GitPrefix: "odd/tasks", History: h, Repo: "r"},
	}, h
}

func query(t *testing.T, raw string) Query {
	t.Helper()
	q, err := ParseID(raw, reqid.Default())
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestRequirementTraceCollectEndToEnd(t *testing.T) {
	in, _ := collectFixture()
	in.Query = query(t, "REQ-30.1")
	res, err := Collect(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Schema != Schema || res.ID != "REQ-30.1" || res.IntegrationBranch != "main" || len(res.Warnings) != 0 {
		t.Fatalf("resultado = %+v", res)
	}
	if !reflect.DeepEqual(res.ResolvedIDs, []string{"REQ-30.1", "prov:trace:1"}) {
		t.Fatalf("resueltos = %v", res.ResolvedIDs)
	}
	type row struct {
		lane   Lane
		role   Role
		kind   Kind
		commit string
		anchor Anchor
	}
	var got []row
	for _, i := range res.Increments {
		got = append(got, row{i.Lane, i.Role, i.Type, i.Integration.Commit, i.Integration.Anchor})
	}
	want := []row{
		{LaneSDD, RoleCreated, KindIncrement, "c1", AnchorCode},
		{LaneSDD, RoleCovered, KindBug, "c3", AnchorCode},
		{LaneODD, RoleModified, KindIncrement, "c5", AnchorDocument},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("incrementos = %+v\nquería %+v", got, want)
	}
	if a := res.Increments[0].Assignments; len(a) != 1 || a[0].Provisional != "prov:trace:1" || a[0].Definitive != "REQ-30.1" {
		t.Fatalf("asignación = %+v", a)
	}
	if res.Increments[1].Tasks[0].File != "openspec/changes/archive/2026-02-01-arreglo/tasks.md" {
		t.Fatalf("tareas = %+v", res.Increments[1].Tasks)
	}
}

func TestRequirementTraceCollectProvisionalAndDefinitiveGiveSameResult(t *testing.T) {
	in, _ := collectFixture()
	in.Query = query(t, "REQ-30.1")
	byDef, err := Collect(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	in.Query = query(t, "prov:trace:1")
	byProv, err := Collect(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if byDef.ID == byProv.ID {
		t.Fatal("el ID consultado debe conservarse tal cual")
	}
	byProv.ID = byDef.ID
	a, _ := json.Marshal(byDef)
	b, _ := json.Marshal(byProv)
	if string(a) != string(b) {
		t.Fatalf("resultados distintos:\n%s\n%s", a, b)
	}
}

func TestRequirementTraceCollectValidIDWithoutAppearancesHasEmptyLists(t *testing.T) {
	in, _ := collectFixture()
	in.Query = query(t, "REQ-99.9")
	res, err := Collect(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(res)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"increments", "warnings"} {
		list, ok := doc[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("%s = %#v; quería una lista vacía", key, doc[key])
		}
	}
}

func TestRequirementTraceCollectMissingRefIsRefError(t *testing.T) {
	in, h := collectFixture()
	h.badRef = "main"
	in.Query = query(t, "REQ-30.1")
	_, err := Collect(context.Background(), in)
	var refErr *RefError
	if !errors.As(err, &refErr) || refErr.Ref != "main" || refErr.Lane != LaneSDD {
		t.Fatalf("error = %v", err)
	}
}

func TestRequirementTraceCollectODDWithoutRepositoryWarnsAndSkips(t *testing.T) {
	in, _ := collectFixture()
	in.ODD.History = nil
	in.Query = query(t, "REQ-30.1")
	res, err := Collect(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Code != WarnODDRepoUnavailable || res.Warnings[0].Path != "odd/tasks" || len(res.Increments) != 2 {
		t.Fatalf("resultado = %+v", res)
	}
}

func TestRequirementTraceCollectIsDeterministic(t *testing.T) {
	in, _ := collectFixture()
	in.Query = query(t, "REQ-30.1")
	first, _ := Collect(context.Background(), in)
	second, _ := Collect(context.Background(), in)
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatalf("dos ejecuciones difieren:\n%s\n%s", a, b)
	}
}
