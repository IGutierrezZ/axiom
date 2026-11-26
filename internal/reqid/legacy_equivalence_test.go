package reqid_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Las expresiones heredadas se copian aquí: son la referencia con la que el
// defecto debe coincidir byte a byte (RF-RID-02).
var (
	legacyCandidate = regexp.MustCompile(`\bREQ-[0-9A-Za-z.]*[0-9A-Za-z]`)
	legacyValid     = regexp.MustCompile(`^REQ-\d+\.\d+$`)
	legacyInTitle   = regexp.MustCompile(`\((REQ-\d+\.\d+)\)`)
	legacyTrailing  = regexp.MustCompile(`\((REQ-[^)]+)\)\s*$`)
)

func legacyInTitleID(title string) string {
	m := legacyInTitle.FindAllStringSubmatch(title, -1)
	if len(m) == 0 {
		return ""
	}
	return m[len(m)-1][1]
}

func legacyScan(text string) (valid, malformed []string) {
	seenV, seenM := map[string]bool{}, map[string]bool{}
	for _, tok := range legacyCandidate.FindAllString(text, -1) {
		if legacyValid.MatchString(tok) {
			if !seenV[tok] {
				seenV[tok] = true
				valid = append(valid, tok)
			}
			continue
		}
		if !seenM[tok] {
			seenM[tok] = true
			malformed = append(malformed, tok)
		}
	}
	return valid, malformed
}

// overflowFree excluye la única divergencia permitida: los números que desbordan int.
func overflowFree(s string) bool { return !strings.Contains(s, "99999999999999999999") }

func checkLegacy(t *testing.T, text string) {
	t.Helper()
	if !overflowFree(text) {
		return
	}
	p := reqid.Default()
	if got, want := p.InTitle(text), legacyInTitleID(text); got != want {
		t.Errorf("InTitle(%q) = %q, heredada %q", text, got, want)
	}
	gv, gm := p.Scan(text)
	wv, wm := legacyScan(text)
	if strings.Join(gv, "|") != strings.Join(wv, "|") || strings.Join(gm, "|") != strings.Join(wm, "|") {
		t.Errorf("Scan(%q) = %v %v, heredada %v %v", text, gv, gm, wv, wm)
	}
	if got, want := p.Valid(text), legacyValid.MatchString(text); got != want {
		t.Errorf("Valid(%q) = %v, heredada %v", text, got, want)
	}
	_, id, ok := p.TrailingCandidate(text)
	m := legacyTrailing.FindStringSubmatch(text)
	if m != nil && strings.TrimSpace(text[:strings.LastIndex(text, "("+m[1]+")")]) != "" {
		if !ok || id != m[1] {
			t.Errorf("TrailingCandidate(%q) = %q, %v, heredada %q", text, id, ok, m[1])
		}
	} else if ok {
		t.Errorf("TrailingCandidate(%q) = %q, la heredada no encaja", text, id)
	}
}

func TestLegacyEquivalence(t *testing.T) {
	edges := []string{
		"REQ-30", "REQ-30.1,", "REQ-30.1.", "REQ-x.1", "REQ-030.01", "(REQ-1.1) (REQ-2.2)",
		"REQ-30.4.1", "REQ-30.1abc", "XREQ-30.1", "req-30.4", "REQ-30.4", "Cobro (REQ-30.4)",
		"Cobro (REQ-30)", "Cobro (REQ-30.4) ", "Cobro REQ-30.4 y (REQ-1.2)", "", "REQ-", "(REQ-)",
	}
	for _, e := range edges {
		checkLegacy(t, e)
	}
	// Tras C-03-e la spec viva usa RF-<DOM>-<NN>; el corpus de la gramática heredada son los
	// títulos `REQ-n.m` de las deltas archivadas, que la historia no reescribe (D11).
	files, err := filepath.Glob(filepath.Join("..", "..", "openspec", "changes", "archive", "*", "specs", "*", "spec.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no hay specs archivadas: %v %v", files, err)
	}
	titles := 0
	for _, f := range files {
		for _, title := range requirementTitles(t, f) {
			checkLegacy(t, title)
			titles++
		}
	}
	if titles == 0 {
		t.Fatal("ningún título de requisito en openspec/changes/archive")
	}
}

func requirementTitles(t *testing.T, file string) []string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "### Requirement:") {
			titles = append(titles, strings.TrimSpace(strings.TrimPrefix(line, "### Requirement:")))
		}
	}
	return titles
}

// TestRFCorpusTitlesHaveIDs: todo requisito de la spec viva lleva un ID RF-<DOM>-<NN> en el
// título y ninguno conserva un REQ-n.m (C-03-e, D11).
func TestRFCorpusTitlesHaveIDs(t *testing.T) {
	rf, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join("..", "..", "openspec", "specs", "*", "spec.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no hay specs vivas: %v %v", files, err)
	}
	titles := 0
	for _, f := range files {
		for _, title := range requirementTitles(t, f) {
			titles++
			if rf.InTitle(title) == "" {
				t.Errorf("%s: el título %q no lleva un ID RF-<DOM>-<NN>", f, title)
			}
			if reqid.Default().InTitle(title) != "" {
				t.Errorf("%s: el título %q conserva un ID heredado REQ-n.m", f, title)
			}
		}
	}
	if titles == 0 {
		t.Fatal("ningún título de requisito en openspec/specs")
	}
}
