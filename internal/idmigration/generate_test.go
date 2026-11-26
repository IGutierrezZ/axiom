package idmigration_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/legacyid"
)

// mapEntry es una entrada de openspec/id-map.yaml (esquema axiom.id-map/v1).
type mapEntry struct {
	Capability string   `yaml:"capability"`
	Old        []string `yaml:"old,flow"`
	Title      string   `yaml:"title"`
	New        string   `yaml:"new"`
}

const mapHeader = "# Mapa de identificadores heredados de Axiom (C-03-e). Lo genera internal/idmigration; no se edita a mano.\nschema: " + legacyid.Schema + "\nentries:\n"

// ---- títulos ----

var (
	titleRe    = regexp.MustCompile(`^(\s*###\s+Requirement:\s*)(.*?)\s*$`)
	oldGroupRe = regexp.MustCompile(`\(\s*(REQ-\d+\.\d+[a-z]?(?:\s*(?:&|,|y)\s*REQ-\d+\.\d+[a-z]?)*)\s*\)`)
	otherIDRe  = regexp.MustCompile(`\([A-Z][A-Z0-9]*-[^)]*\)`)
)

// titleParts es la lectura de un título `### Requirement:` (D6).
type titleParts struct {
	Name       string
	Olds       []string
	GroupStart int // relativos al nombre; -1 sin grupo
	GroupEnd   int
}

// parseTitle lee el último grupo entre paréntesis formado solo por IDs heredados y separadores.
func parseTitle(name string) titleParts {
	tp := titleParts{Name: name, GroupStart: -1, GroupEnd: -1}
	ms := oldGroupRe.FindAllStringSubmatchIndex(name, -1)
	if len(ms) == 0 {
		return tp
	}
	m := ms[len(ms)-1]
	tp.GroupStart, tp.GroupEnd = m[0], m[1]
	for _, h := range findTokens(name[m[2]:m[3]]) {
		tp.Olds = append(tp.Olds, h.Text)
	}
	return tp
}

// titleLine es un título de requisito en un fuente Markdown, fuera de vallas.
type titleLine struct {
	Line        int
	Start       int // inicio absoluto del nombre
	Parts       titleParts
	LineStartAt int
}

// specTitles devuelve los títulos `### Requirement:` de una spec, en orden de aparición y fuera
// de las vallas de código.
func specTitles(content string) []titleLine {
	var out []titleLine
	for i, l := range analyzeMarkdown(splitLines(content)) {
		if l.InFence {
			continue
		}
		m := titleRe.FindStringSubmatch(l.Text)
		if m == nil {
			continue
		}
		out = append(out, titleLine{Line: i + 1, Start: l.Start + len(m[1]), Parts: parseTitle(m[2]), LineStartAt: l.Start})
	}
	return out
}

// ---- índice del mapa ----

type oldRef struct {
	Old    string
	Series string
	Num    int
	NewNum int
}

type pair struct{ capability, old string }

// mapIndex indexa las entradas del mapa para el reescritor.
type mapIndex struct {
	entries []mapEntry
	byPair  map[pair]*mapEntry
	oldCaps map[string][]string
	byNew   map[string]*mapEntry
	byTitle map[pair]*mapEntry
	caps    map[string]bool
	capOlds map[string][]oldRef
}

func newMapIndex(entries []mapEntry) *mapIndex {
	ix := &mapIndex{
		entries: entries, byPair: map[pair]*mapEntry{}, oldCaps: map[string][]string{},
		byNew: map[string]*mapEntry{}, byTitle: map[pair]*mapEntry{}, caps: map[string]bool{},
		capOlds: map[string][]oldRef{},
	}
	for i := range entries {
		e := &entries[i]
		ix.caps[e.Capability] = true
		ix.byNew[e.New] = e
		if len(e.Old) == 0 && e.Title != "" {
			ix.byTitle[pair{e.Capability, e.Title}] = e
		}
		newNum, _ := rfPattern.Number(e.New)
		for _, old := range e.Old {
			ix.byPair[pair{e.Capability, old}] = e
			ix.oldCaps[old] = append(ix.oldCaps[old], e.Capability)
			if m := tokenRe.FindStringSubmatch(old); m != nil {
				n, _ := strconv.Atoi(m[2])
				ix.capOlds[e.Capability] = append(ix.capOlds[e.Capability], oldRef{Old: old, Series: m[1], Num: n, NewNum: newNum})
			}
		}
	}
	for old := range ix.oldCaps {
		sort.Strings(ix.oldCaps[old])
	}
	return ix
}

// ambiguous dice si el ID antiguo aparece en los títulos de dos o más capacidades.
func (ix *mapIndex) ambiguous(old string) bool { return len(ix.oldCaps[old]) >= 2 }

// ---- generador ----

// generateMap calcula las entradas del mapa desde las specs vivas de root: una por título,
// numeradas por aparición con el código de dominio de la capacidad. La ambigüedad no se
// codifica: la calcula newMapIndex a partir de los pares (capacidad, ID antiguo).
func generateMap(root string, dom map[string]string) ([]mapEntry, error) {
	dirs, err := os.ReadDir(filepath.Join(root, "openspec", "specs"))
	if err != nil {
		return nil, fmt.Errorf("no se pueden leer las specs vivas: %w", err)
	}
	var out []mapEntry
	var errs []error
	for _, d := range dirs { // os.ReadDir ordena por nombre, es decir, por bytes
		if !d.IsDir() {
			continue
		}
		capability := d.Name()
		data, err := os.ReadFile(filepath.Join(root, "openspec", "specs", capability, "spec.md"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", capability, err))
			continue
		}
		code, ok := dom[capability]
		if !ok {
			errs = append(errs, fmt.Errorf("la capacidad %s no tiene código de dominio en la tabla aprobada", capability))
			continue
		}
		series, ok := rfPattern.Series("RF-" + code + "-01")
		if !ok {
			errs = append(errs, fmt.Errorf("el código %s no forma serie bajo %s", code, rfTemplate))
			continue
		}
		seenOld, seenTitle := map[string]bool{}, map[string]bool{}
		for n, tl := range specTitles(string(data)) {
			e := mapEntry{Capability: capability, New: rfPattern.Format(series, n+1)}
			where := fmt.Sprintf("%s:%d", capability, tl.Line)
			rest := tl.Parts.Name
			if tl.Parts.GroupStart >= 0 {
				rest = tl.Parts.Name[:tl.Parts.GroupStart] + tl.Parts.Name[tl.Parts.GroupEnd:]
				e.Old = tl.Parts.Olds
			}
			if id := rfPattern.InTitle(tl.Parts.Name); id != "" || otherIDRe.MatchString(rest) {
				errs = append(errs, fmt.Errorf("%s: el título %q ya lleva un ID que no es REQ-n.m", where, tl.Parts.Name))
				continue
			}
			if len(findTokens(rest)) > 0 {
				errs = append(errs, fmt.Errorf("%s: el título %q cita un ID heredado fuera de un grupo entre paréntesis", where, tl.Parts.Name))
				continue
			}
			if len(e.Old) == 0 {
				e.Title = strings.TrimSpace(tl.Parts.Name)
				if seenTitle[e.Title] {
					errs = append(errs, fmt.Errorf("%s: el título sin ID %q está repetido en la capacidad", where, e.Title))
					continue
				}
				seenTitle[e.Title] = true
			}
			for _, old := range e.Old {
				if seenOld[old] {
					errs = append(errs, fmt.Errorf("%s: el ID %s está repetido en la capacidad", where, old))
				}
				seenOld[old] = true
			}
			out = append(out, e)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// scalar emite un escalar YAML en una línea con yaml.Marshal.
func scalar(s string) (string, error) {
	b, err := yaml.Marshal(s)
	if err != nil {
		return "", err
	}
	out := strings.TrimSuffix(string(b), "\n")
	if strings.ContainsAny(out, "\n\r") {
		return "", fmt.Errorf("el escalar %q ocupa más de una línea", s)
	}
	return out, nil
}

// emitMap serializa las entradas de forma determinista: cabecera fija, LF y salto final.
func emitMap(entries []mapEntry) (string, error) {
	var b strings.Builder
	b.WriteString(mapHeader)
	for _, e := range entries {
		c, err := scalar(e.Capability)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "  - capability: %s\n", c)
		if len(e.Old) > 0 {
			olds := make([]string, len(e.Old))
			for i, o := range e.Old {
				if olds[i], err = scalar(o); err != nil {
					return "", err
				}
			}
			fmt.Fprintf(&b, "    old: [%s]\n", strings.Join(olds, ", "))
		}
		if e.Title != "" {
			t, err := scalar(e.Title)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "    title: %s\n", t)
		}
		n, err := scalar(e.New)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "    new: %s\n", n)
	}
	return b.String(), nil
}

// readMapFile lee el mapa de root: lo valida con legacyid.Load y decodifica sus entradas.
func readMapFile(root string) ([]mapEntry, error) {
	dir := filepath.Join(root, "openspec")
	if _, err := legacyid.Load(os.DirFS(dir), legacyid.FileName, rfPattern); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, legacyid.FileName))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Schema  string     `yaml:"schema"`
		Entries []mapEntry `yaml:"entries"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return doc.Entries, nil
}

// ---- pruebas ----

func miniRoot(t *testing.T) string {
	t.Helper()
	return copyTree(t, filepath.Join("testdata", "mini"))
}

func entriesFor(t *testing.T, root string) []mapEntry {
	t.Helper()
	entries, err := generateMap(root, domains)
	if err != nil {
		t.Fatalf("generateMap: %v", err)
	}
	return entries
}

func find(entries []mapEntry, capability, newID string) *mapEntry {
	for i := range entries {
		if entries[i].Capability == capability && entries[i].New == newID {
			return &entries[i]
		}
	}
	return nil
}

func TestGenerateNumbersByAppearanceAndComputesAmbiguity(t *testing.T) {
	root := miniRoot(t)
	entries := entriesFor(t, root)

	want := []struct {
		capability, newID string
		old               []string
		title             string
	}{
		{"autoskills", "RF-ASK-01", []string{"REQ-1.1"}, ""},
		{"autoskills", "RF-ASK-02", []string{"REQ-1.2"}, ""},
		{"autoskills", "RF-ASK-03", []string{"REQ-1.10"}, ""},
		{"multi-project-hub", "RF-HUB-01", []string{"REQ-2.1"}, ""},
		{"persona-behavior-contract", "RF-PER-01", []string{"REQ-10.1", "REQ-10.2"}, ""}, // (A & B): una entrada con dos antiguos
		{"persona-behavior-contract", "RF-PER-02", []string{"REQ-10.3"}, ""},
		{"sdd-lane-kickoff", "RF-KOF-01", []string{"REQ-27.2"}, ""},
		{"sdd-lane-kickoff", "RF-KOF-02", []string{"REQ-27.2b"}, ""}, // el token con sufijo es una clave más
		{"workspace-topology", "RF-TOP-01", []string{"REQ-1.1"}, ""},
		{"workspace-topology", "RF-TOP-02", []string{"REQ-2.1"}, ""},
		{"workspace-topology", "RF-TOP-03", []string{"REQ-2.3"}, ""},   // título no final: `(REQ-2.3) [SUPERSEDED / DEPRECADO]`
		{"workspace-topology", "RF-TOP-04", nil, "Untitled Behaviour"}, // sin ID: lleva el título original
	}
	for _, w := range want {
		e := find(entries, w.capability, w.newID)
		if e == nil {
			t.Errorf("falta %s/%s en %v", w.capability, w.newID, entries)
			continue
		}
		if strings.Join(e.Old, ",") != strings.Join(w.old, ",") || e.Title != w.title {
			t.Errorf("%s/%s = %+v; quiero old=%v title=%q", w.capability, w.newID, *e, w.old, w.title)
		}
	}

	ix := newMapIndex(entries)
	for old, wantCaps := range map[string]string{
		"REQ-1.1":   "autoskills,workspace-topology",
		"REQ-2.1":   "multi-project-hub,workspace-topology",
		"REQ-1.2":   "autoskills",
		"REQ-27.2":  "sdd-lane-kickoff",
		"REQ-27.2b": "sdd-lane-kickoff",
	} {
		if got := strings.Join(ix.oldCaps[old], ","); got != wantCaps {
			t.Errorf("capacidades de %s = %q; quiero %q", old, got, wantCaps)
		}
	}
	if !ix.ambiguous("REQ-1.1") || ix.ambiguous("REQ-1.2") {
		t.Error("la ambigüedad debe calcularse por el número de capacidades del token")
	}
}

func TestGenerateTwiceIsIdenticalByteForByte(t *testing.T) {
	root := miniRoot(t)
	a, err := emitMap(entriesFor(t, root))
	if err != nil {
		t.Fatal(err)
	}
	b, err := emitMap(entriesFor(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("dos generaciones difieren:\n%s\n---\n%s", a, b)
	}
}

func TestGenerateEmitterIsDeterministicLF(t *testing.T) {
	text, err := emitMap([]mapEntry{
		{Capability: "autoskills", Old: []string{"REQ-1.1"}, New: "RF-ASK-01"},
		{Capability: "workspace-topology", Title: `Título: con "comillas" y # almohadilla`, New: "RF-TOP-01"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(text, mapHeader) || !strings.HasSuffix(text, "\n") || strings.Contains(text, "\r") {
		t.Fatalf("cabecera fija, LF y salto final: %q", text)
	}
	if !strings.Contains(text, "    old: [REQ-1.1]\n") || !strings.Contains(text, "    new: RF-TOP-01\n") {
		t.Fatalf("old en estilo flujo y new al final: %q", text)
	}
	if _, err := legacyid.Load(fstest.MapFS{legacyid.FileName: {Data: []byte(text)}}, legacyid.FileName, rfPattern); err != nil {
		t.Fatalf("un título con caracteres especiales debe sobrevivir a la ida y vuelta: %v", err)
	}
}

func TestGenerateRoundTripWithLegacyidLoad(t *testing.T) {
	root := miniRoot(t)
	text, err := emitMap(entriesFor(t, root))
	if err != nil {
		t.Fatal(err)
	}
	m, err := legacyid.Load(fstest.MapFS{legacyid.FileName: {Data: []byte(text)}}, legacyid.FileName, rfPattern)
	if err != nil {
		t.Fatalf("legacyid.Load(Generate(x)): %v", err)
	}
	if got := m.InCapability("workspace-topology", "REQ-1.1"); got != "RF-TOP-01" {
		t.Errorf("InCapability = %q", got)
	}
	// REQ-27.2b es un token del mapa distinto de REQ-27.2 (hueco 7).
	if !m.Legacy("REQ-27.2b") || m.InCapability("sdd-lane-kickoff", "REQ-27.2b") != "RF-KOF-02" || m.InCapability("sdd-lane-kickoff", "REQ-27.2") != "RF-KOF-01" {
		t.Errorf("REQ-27.2b y REQ-27.2 deben ser claves distintas: %v", m.Aliases("REQ-27.2b"))
	}
	if got := m.Aliases("REQ-10.2"); len(got) != 1 || got[0] != "RF-PER-01" {
		t.Errorf("un título con dos antiguos resuelve ambos al mismo nuevo: %v", got)
	}
}

func TestGenerateMapModeFailsWhenMapExists(t *testing.T) {
	root := miniRoot(t)
	if err := os.WriteFile(filepath.Join(root, "openspec", "id-map.yaml"), []byte("existente\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runMode("map", runEnv{Root: root, List: walkList(root), Logf: t.Logf})
	if err == nil || !strings.Contains(err.Error(), "ya existe") {
		t.Fatalf("map no debe sobrescribir un mapa existente: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(root, "openspec", "id-map.yaml")); string(data) != "existente\n" {
		t.Fatalf("el mapa existente se modificó: %q", data)
	}
}

func TestGenerateMapModeWritesMapOnce(t *testing.T) {
	root := miniRoot(t)
	if err := runMode("map", runEnv{Root: root, List: walkList(root), Logf: t.Logf}); err != nil {
		t.Fatalf("map: %v", err)
	}
	entries, err := readMapFile(root)
	if err != nil {
		t.Fatalf("el mapa generado no se puede leer: %v", err)
	}
	if len(entries) == 0 || find(entries, "autoskills", "RF-ASK-01") == nil {
		t.Fatalf("mapa incompleto: %v", entries)
	}
}

func TestGenerateFailsWhenTitleAlreadyHasNonLegacyID(t *testing.T) {
	for name, title := range map[string]string{
		"RF ya asignado":  "Skill Detection (RF-ASK-09)",
		"otro formato":    "Skill Detection (ABC-12)",
		"ID fuera grupo":  "Skill Detection REQ-1.1",
		"RF tras antiguo": "Skill Detection (REQ-1.1) (RF-ASK-01)",
	} {
		t.Run(name, func(t *testing.T) {
			root := miniRoot(t)
			writeFile(t, root, "openspec/specs/autoskills/spec.md", "# Autoskills\n\n### Requirement: "+title+"\n\nTexto.\n")
			if _, err := generateMap(root, domains); err == nil {
				t.Fatalf("un título %q debe hacer fallar la generación", title)
			}
		})
	}
}

func TestGenerateFailsWithoutDomainCodeOrWithRepeatedKeys(t *testing.T) {
	root := miniRoot(t)
	writeFile(t, root, "openspec/specs/capacidad-sin-codigo/spec.md", "# X\n\n### Requirement: Algo (REQ-1.1)\n")
	if _, err := generateMap(root, domains); err == nil || !strings.Contains(err.Error(), "capacidad-sin-codigo") {
		t.Fatalf("una capacidad sin código de dominio debe fallar y nombrarse: %v", err)
	}

	root = miniRoot(t)
	writeFile(t, root, "openspec/specs/autoskills/spec.md", "# A\n\n### Requirement: Uno (REQ-1.1)\n\n### Requirement: Dos (REQ-1.1)\n")
	if _, err := generateMap(root, domains); err == nil {
		t.Fatal("un ID antiguo repetido en una capacidad debe fallar")
	}
}

func TestGenerateIgnoresRequirementHeadingsInsideFences(t *testing.T) {
	root := miniRoot(t)
	entries := entriesFor(t, root)
	for _, e := range entries {
		if e.Capability == "autoskills" && (strings.Contains(strings.Join(e.Old, ","), "REQ-9.9") || e.Title == "Falso en valla") {
			t.Fatalf("un título dentro de una valla no es un requisito: %+v", e)
		}
	}
}
