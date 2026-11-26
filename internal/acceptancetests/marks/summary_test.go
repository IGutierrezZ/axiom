package marks

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
)

const otherDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func withDigest(e Entry, d string) Entry { e.PlanDigest = d; return e }

func sumOf(t *testing.T, s Summary, env string) EnvSummary {
	t.Helper()
	for _, e := range s.Environments {
		if e.Environment == env {
			return e
		}
	}
	t.Fatalf("falta el entorno %s", env)
	return EnvSummary{}
}

func checkSum(t *testing.T, s Summary) {
	t.Helper()
	for _, e := range s.Environments {
		if got := e.OK + e.KO + e.Blocked + e.NA + e.Stale + e.Unmarked; got != e.Total {
			t.Errorf("%s: la suma %d no es el total %d", e.Environment, got, e.Total)
		}
	}
}

func TestSummarizeScenarioFromSpec(t *testing.T) {
	var cases []acceptancetests.Case
	for _, id := range []string{"P-01", "P-02", "P-03", "P-04", "P-05"} {
		cases = append(cases, acceptancetests.Case{ID: id, Environments: []string{"pre"}})
	}
	cases[0].Environments = []string{"pre", "local"}
	cases[1].Environments = []string{"pre", "local"}
	var entries []Entry
	for _, id := range []string{"P-01", "P-02", "P-03", "P-04", "P-05"} {
		entries = append(entries, entry(id, "pre", ResultOK, "2026-11-26T19:00:00.000Z"))
	}
	entries = append(entries,
		entry("P-01", "local", ResultOK, "2026-11-26T19:00:00.000Z"),
		withDigest(entry("P-02", "local", ResultOK, "2026-11-26T19:00:00.000Z"), otherDigest),
	)
	s := Summarize(acceptancetests.Battery{Cases: cases}, testEnvs2(), entries, testDigest)
	checkSum(t, s)
	if got, want := s.Text(), "pre: 5/5 ok; local: 1/2 ok, 1 caducada"; got != want {
		t.Fatalf("Text = %q, quería %q", got, want)
	}
}

func testEnvs2() []string { return []string{"pre", "local"} }

func TestSummarizeCategoriesAndNAApart(t *testing.T) {
	b := acceptancetests.Battery{Cases: []acceptancetests.Case{
		{ID: "P-01", Environments: []string{"pre"}}, {ID: "P-02", Environments: []string{"pre"}},
		{ID: "P-03", Environments: []string{"pre"}}, {ID: "P-04", Environments: []string{"pre"}},
		{ID: "P-05", Environments: []string{"pre"}}, {ID: "P-06", Environments: []string{"pre"}},
	}}
	entries := []Entry{
		entry("P-01", "pre", ResultOK, "2026-11-26T19:00:00.000Z"),
		entry("P-02", "pre", ResultKO, "2026-11-26T19:00:00.000Z"),
		entry("P-03", "pre", ResultBlocked, "2026-11-26T19:00:00.000Z"),
		entry("P-04", "pre", ResultNA, "2026-11-26T19:00:00.000Z"),
		withDigest(entry("P-05", "pre", ResultOK, "2026-11-26T19:00:00.000Z"), otherDigest),
	}
	s := Summarize(b, []string{"pre"}, entries, testDigest)
	checkSum(t, s)
	e := sumOf(t, s, "pre")
	if e.Total != 6 || e.OK != 1 || e.KO != 1 || e.Blocked != 1 || e.NA != 1 || e.Stale != 1 || e.Unmarked != 1 {
		t.Fatalf("recuento = %+v", e)
	}
	if got, want := s.Text(), "pre: 1/6 ok, 1 ko, 1 bloqueado, 1 n/a, 1 sin marcar, 1 caducada"; got != want {
		t.Fatalf("Text = %q", got)
	}
}

func TestSummarizeCurrentByLatestAtThenFileOrder(t *testing.T) {
	b := acceptancetests.Battery{Cases: []acceptancetests.Case{{ID: "P-01", Environments: []string{"pre"}}}}
	// La última del fichero es más antigua: gana la de mayor `at`.
	s := Summarize(b, []string{"pre"}, []Entry{
		entry("P-01", "pre", ResultKO, "2026-11-26T19:00:05.000Z"),
		entry("P-01", "pre", ResultOK, "2026-11-26T19:00:01.000Z"),
	}, testDigest)
	if e := sumOf(t, s, "pre"); e.KO != 1 || e.OK != 0 {
		t.Fatalf("debía ganar ko por at: %+v", e)
	}
	// Empate de `at`: gana la última del fichero.
	s = Summarize(b, []string{"pre"}, []Entry{
		entry("P-01", "pre", ResultKO, "2026-11-26T19:00:05.000Z"),
		entry("P-01", "pre", ResultOK, "2026-11-26T19:00:05.000Z"),
	}, testDigest)
	if e := sumOf(t, s, "pre"); e.OK != 1 || e.KO != 0 {
		t.Fatalf("debía ganar la última por orden: %+v", e)
	}
	// Vigente ko + caducada ok anterior: cuenta ko, no caducada.
	s = Summarize(b, []string{"pre"}, []Entry{
		withDigest(entry("P-01", "pre", ResultOK, "2026-11-26T19:00:01.000Z"), otherDigest),
		entry("P-01", "pre", ResultKO, "2026-11-26T19:00:02.000Z"),
	}, testDigest)
	if e := sumOf(t, s, "pre"); e.KO != 1 || e.Stale != 0 {
		t.Fatalf("vigente ko con caducada: %+v", e)
	}
}

func TestSummarizeTotalZeroOrphansAndRepeatedCase(t *testing.T) {
	b := acceptancetests.Battery{Cases: []acceptancetests.Case{
		{ID: "P-01", Environments: []string{"pre"}},
		{ID: "P-01", Environments: []string{"pre", "local"}}, // repetido: cuenta una vez
		{ID: "malo", Environments: []string{"pre"}},          // ID inválido: no es caso
	}}
	entries := []Entry{
		entry("P-01", "pre", ResultOK, "2026-11-26T19:00:00.000Z"),
		entry("P-07", "pre", ResultOK, "2026-11-26T19:00:00.000Z"),  // caso inexistente
		entry("P-01", "prod", ResultOK, "2026-11-26T19:00:00.000Z"), // entorno no declarado
		entry("malo", "pre", ResultOK, "2026-11-26T19:00:00.000Z"),  // caso no válido
	}
	s := Summarize(b, []string{"pre", "local", "qa"}, entries, testDigest)
	checkSum(t, s)
	if s.Orphans != 3 {
		t.Fatalf("huérfanas = %d", s.Orphans)
	}
	if e := sumOf(t, s, "pre"); e.Total != 1 || e.OK != 1 {
		t.Fatalf("pre = %+v", e)
	}
	if e := sumOf(t, s, "local"); e.Total != 1 || e.Unmarked != 1 {
		t.Fatalf("local = %+v", e)
	}
	if got, want := s.Text(), "pre: 1/1 ok; local: 0/1 ok, 1 sin marcar; qa: sin casos"; got != want {
		t.Fatalf("Text = %q", got)
	}
}

func TestSummaryTextPlurals(t *testing.T) {
	s := Summary{Environments: []EnvSummary{{Environment: "pre", Total: 5, Blocked: 2, Stale: 2, KO: 1}}}
	if got, want := s.Text(), "pre: 0/5 ok, 1 ko, 2 bloqueados, 2 caducadas"; got != want {
		t.Fatalf("Text = %q", got)
	}
	if (Summary{}).Text() != "" {
		t.Error("sin entornos el texto es vacío")
	}
}

func TestSummarizeToleratesCRLFBattery(t *testing.T) {
	md := strings.Join([]string{
		"| ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones | Pasos | Resultado esperado |",
		"|----|-------|-----|------|-----------|----------|----------------|-------|--------------------|",
		"| P-01 | RF-X-01 | fullstack | manual | alta | pre; local | - | - | ok |",
		"| P-02 | RF-X-01 | fullstack | manual | alta | pre | - | - | ok |",
		"",
	}, "\r\n")
	b := acceptancetests.Parse(md)
	s := Summarize(b, []string{"pre", "local"}, nil, testDigest)
	if got, want := s.Text(), "pre: 0/2 ok, 2 sin marcar; local: 0/1 ok, 1 sin marcar"; got != want {
		t.Fatalf("Text = %q (casos=%d)", got, len(b.Cases))
	}
}
