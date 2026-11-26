package marks

import (
	"bytes"
	"strings"
	"testing"
)

const testDigest = "2fe1cfd9a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e"

func entry(caseID, env string, result Result, at string) Entry {
	return Entry{Case: caseID, Environment: env, Result: result, By: "Ana Pérez <ana@example.org>",
		At: at, PlanDigest: testDigest}
}

func TestParseValidAndEmpty(t *testing.T) {
	for _, empty := range []string{"", "\n", "# nada\n"} {
		f, err := Parse([]byte(empty))
		if err != nil || len(f.Entries) != 0 || len(f.Invalid) != 0 {
			t.Fatalf("Parse(%q) = %+v, %v", empty, f, err)
		}
	}
	data, err := Append(nil, entry("P-01", "pre", ResultOK, "2026-11-26T19:18:33.072Z"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(data)
	if err != nil || len(f.Entries) != 1 || f.Entries[0].Case != "P-01" {
		t.Fatalf("Parse = %+v, %v", f, err)
	}
}

func TestParseUnreadableAndNotSequence(t *testing.T) {
	if _, err := Parse([]byte("- case: [\n")); err == nil {
		t.Error("YAML ilegible debe dar error")
	}
	for _, doc := range []string{"case: P-01\n", "texto\n", "42\n"} {
		if _, err := Parse([]byte(doc)); err == nil {
			t.Errorf("Parse(%q) debe dar error al no ser secuencia", doc)
		}
	}
}

func TestParseInvalidEntriesApart(t *testing.T) {
	good, _ := Append(nil, entry("P-01", "pre", ResultOK, "2026-11-26T19:18:33.072Z"))
	doc := string(good) + strings.Join([]string{
		"- case: P-02", "  environment: pre", "  result: maybe", "  by: x", "  at: \"2026-11-26T19:18:33.072Z\"", "  plan_digest: " + testDigest,
		"- just a string",
		"- case: P-03", "  environment: pre", "  result: ok", "  by: x", "  at: ayer", "  plan_digest: " + testDigest,
		"- case: P-04", "  environment: pre", "  result: ok", "  by: x", "  at: \"2026-11-26T19:18:33.072Z\"", "  plan_digest: zz",
		"- case: P-05", "  environment: pre", "  result: ok", "  by: \"\"", "  at: \"2026-11-26T19:18:33.072Z\"", "  plan_digest: " + testDigest,
		"",
	}, "\n")
	f, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 || len(f.Invalid) != 5 {
		t.Fatalf("válidas=%d inválidas=%d: %+v", len(f.Entries), len(f.Invalid), f)
	}
}

func TestAppendKeepsPreviousContentAsExactPrefix(t *testing.T) {
	first, err := Append(nil, entry("P-01", "pre", ResultOK, "2026-11-26T19:18:33.072Z"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Append(first, entry("P-02", "pre", ResultKO, "2026-11-26T19:18:34.072Z"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(second, first) {
		t.Fatalf("el contenido previo no es prefijo:\n%s", second)
	}
	// Sin `\n` final en el contenido previo se garantiza uno, y el resto sigue siendo prefijo.
	trimmed := bytes.TrimRight(first, "\n")
	third, err := Append(trimmed, entry("P-03", "pre", ResultOK, "2026-11-26T19:18:35.072Z"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(third, trimmed) || !bytes.HasPrefix(third[len(trimmed):], []byte("\n")) {
		t.Fatalf("falta el salto de línea final:\n%s", third)
	}
	if f, _ := Parse(third); len(f.Entries) != 2 {
		t.Fatalf("entradas = %d", len(f.Entries))
	}
}

func TestAppendRefusesUnreadableAndInlineSequences(t *testing.T) {
	e := entry("P-01", "pre", ResultOK, "2026-11-26T19:18:33.072Z")
	if _, err := Append([]byte("- case: [\n"), e); err == nil {
		t.Error("contenido ilegible debe dar error")
	}
	if _, err := Append([]byte("[]\n"), e); err == nil {
		t.Error("secuencia en línea debe dar error")
	}
	bad := e
	bad.Result = "x"
	if _, err := Append(nil, bad); err == nil {
		t.Error("marca fuera de forma debe dar error")
	}
}

func TestMergeNeverDuplicates(t *testing.T) {
	a := entry("P-01", "pre", ResultOK, "2026-11-26T19:18:33.072Z")
	b := entry("P-02", "pre", ResultKO, "2026-11-26T19:18:34.072Z")
	c := entry("P-03", "pre", ResultOK, "2026-11-26T19:18:35.072Z")
	local, _ := Append(nil, a)
	local, _ = Append(local, b)
	local, _ = Append(local, c)
	remote, _ := Append(nil, c) // el remoto ya tiene c (subida que el cliente creyó fallida)
	remote, _ = Append(remote, entry("P-09", "pre", ResultOK, "2026-11-26T19:18:36.072Z"))

	merged, err := Merge(local, remote)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(merged, remote) {
		t.Fatal("el remoto debe ser prefijo")
	}
	f, _ := Parse(merged)
	var ids []string
	for _, e := range f.Entries {
		ids = append(ids, e.Case)
	}
	if got := strings.Join(ids, ","); got != "P-03,P-09,P-01,P-02" {
		t.Fatalf("orden = %s", got)
	}
	again, err := Merge(local, merged)
	if err != nil || !bytes.Equal(again, merged) {
		t.Fatalf("fusionar de nuevo debe ser idempotente: %v", err)
	}
	if _, err := Merge([]byte("- case: [\n"), remote); err == nil {
		t.Error("local ilegible debe dar error")
	}
	if _, err := Merge(local, []byte("x: 1\n")); err == nil {
		t.Error("remoto no secuencia debe dar error")
	}
	empty, err := Merge(local, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := Parse(empty); len(f.Entries) != 3 {
		t.Fatalf("con remoto vacío deben pasar las 3 locales, hay %d", len(f.Entries))
	}
}

func TestPendingIsMultisetDifferenceInLocalOrder(t *testing.T) {
	a := entry("P-01", "pre", ResultOK, "2026-11-26T19:18:33.072Z")
	b := entry("P-02", "pre", ResultOK, "2026-11-26T19:18:34.072Z")
	got := Pending([]Entry{a, b, a}, []Entry{a})
	if len(got) != 2 || got[0] != b || got[1] != a {
		t.Fatalf("Pending = %+v", got)
	}
	if got := Pending([]Entry{a}, []Entry{a, b}); len(got) != 0 {
		t.Fatalf("nada pendiente, hay %+v", got)
	}
	if got := Pending(nil, nil); len(got) != 0 {
		t.Fatal("vacío")
	}
}

func TestRoundTripHostileNotes(t *testing.T) {
	notes := []string{
		"con: dos puntos", "# comentario", `comillas "dobles" y 'simples'`, "- guion", "yes", "null", "true", "123",
		"<script>alert(1)</script>", "ñandú 日本語 😀", "  espacios iniciales", "{a: b}", "[x]", "a # b", "|", ">", "~", "",
	}
	var data []byte
	for i, n := range notes {
		e := entry("P-01", "pre", ResultOK, "2026-11-26T19:18:33.072Z")
		e.Note = n
		e.Commit = "build#" + string(rune('a'+i))
		var err error
		if data, err = Append(data, e); err != nil {
			t.Fatalf("Append(%q): %v", n, err)
		}
	}
	f, err := Parse(data)
	if err != nil || len(f.Entries) != len(notes) || len(f.Invalid) != 0 {
		t.Fatalf("Parse: %v %+v", err, f)
	}
	for i, n := range notes {
		if f.Entries[i].Note != n {
			t.Errorf("nota %d = %q, quería %q", i, f.Entries[i].Note, n)
		}
	}
}
