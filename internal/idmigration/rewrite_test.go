package idmigration_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---- regiones ----

// region es un tramo del fichero que el reescritor trata con una misma regla.
type region struct {
	Start, End int
	Scope      string
	Keep       bool     // se conserva automáticamente (D4)
	Title      bool     // línea `### Requirement:` (D6)
	Cap        string   // capacidad vigente (títulos)
	Caps       []string // capacidades de contexto de fichero (D3, regla 4)
}

type mdMode int

const (
	mdSpec mdMode = iota
	mdODD
	mdText
)

var capHeadingRe = regexp.MustCompile(`^\s*###\s+Capacidad:\s*([A-Za-z0-9._-]+)`)

func firstWord(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

func lineRegions(l mdLine, allowTitle bool, capability string, caps []string, keepCode bool) []region {
	if allowTitle && titleRe.MatchString(l.Text) {
		return []region{{Start: l.Start, End: l.End, Scope: scopeAll, Title: true, Cap: capability}}
	}
	var out []region
	for _, s := range inlineSegments(l.Text) {
		out = append(out, region{
			Start: l.Start + s.Start, End: l.Start + s.End, Scope: s.Scope,
			Keep: keepCode && s.Scope == scopeCode, Cap: capability, Caps: caps,
		})
	}
	return out
}

// mdRegions divide un fuente Markdown según el modo: specs vivas (código y vallas se
// conservan), documentos ODD (solo se reescriben las vallas `openspec` de una capacidad viva y
// el texto fuera de vallas) y Markdown de texto (todo se reescribe salvo residuo).
func mdRegions(content string, mode mdMode, specCap string, ix *mapIndex) []region {
	infos := analyzeMarkdown(splitLines(content))
	var fileCaps []string
	if mode == mdODD {
		seen := map[string]bool{}
		for _, l := range infos {
			if l.InFence {
				continue
			}
			if m := capHeadingRe.FindStringSubmatch(l.Text); m != nil && !seen[m[1]] {
				seen[m[1]] = true
				fileCaps = append(fileCaps, m[1])
			}
		}
	}
	// El cuerpo de una valla `openspec` es Markdown de spec: sus vallas anidadas se conservan.
	inner := map[int]mdLine{}
	for i := 0; i < len(infos); i++ {
		if !infos[i].Open || firstWord(infos[i].Info) != "openspec" {
			continue
		}
		j := i + 1
		var body []line
		for ; j < len(infos) && !infos[j].Marker; j++ {
			body = append(body, infos[j].line)
		}
		for k, bl := range analyzeMarkdown(body) {
			inner[i+1+k] = bl
		}
		i = j
	}
	cur := specCap
	var out []region
	for i, l := range infos {
		if l.InFence {
			switch mode {
			case mdSpec:
				out = append(out, region{Start: l.Start, End: l.End, Scope: scopeCode, Keep: true})
			case mdODD:
				if l.Marker {
					continue
				}
				if firstWord(l.Info) == "openspec" && ix.caps[cur] {
					if in, ok := inner[i]; ok && in.InFence {
						out = append(out, region{Start: l.Start, End: l.End, Scope: scopeCode, Keep: true})
						continue
					}
					out = append(out, lineRegions(l, true, cur, []string{cur}, true)...)
				} else {
					out = append(out, region{Start: l.Start, End: l.End, Scope: scopeCode, Keep: true})
				}
			default:
				out = append(out, region{Start: l.Start, End: l.End, Scope: scopeCode})
			}
			continue
		}
		switch mode {
		case mdSpec:
			out = append(out, lineRegions(l, true, specCap, []string{specCap}, true)...)
		case mdODD:
			if m := capHeadingRe.FindStringSubmatch(l.Text); m != nil {
				cur = m[1]
			}
			out = append(out, lineRegions(l, false, "", fileCaps, false)...)
		default:
			out = append(out, lineRegions(l, false, "", nil, false)...)
		}
	}
	return out
}

func goRegions(content string) ([]region, error) {
	segs, err := goSegments(content)
	if err != nil {
		return nil, err
	}
	out := make([]region, len(segs))
	for i, s := range segs {
		out[i] = region{Start: s.Start, End: s.End, Scope: s.Scope}
	}
	return out, nil
}

// ---- reescritor ----

// fileEnv son las entradas de la reescritura de un fichero.
type fileEnv struct {
	idx      *mapIndex
	ctxs     []contextEntry
	res      []residueEntry
	pinned   map[string]bool // tokens fijados por un mensaje de producción (Go de pruebas)
	testKeep map[string]bool // tokens en literales de prueba no fijados (Go de pruebas)
}

// fileOut es el resultado de reescribir un fichero.
type fileOut struct {
	edits     []edit
	fails     []string
	unmapped  []string
	keptRes   int
	keptAuto  int
	subs      int
	rewritten map[string]bool
}

type fileCtx struct {
	env     fileEnv
	path    string
	kind    fileKind
	content string
	li      lineIndex
	out     *fileOut
}

func (c *fileCtx) fail(format string, args ...any) {
	c.out.fails = append(c.out.fails, fmt.Sprintf(format, args...))
}

func (c *fileCtx) add(start, end int, repl string) {
	c.out.edits = append(c.out.edits, edit{Start: start, End: end, Repl: repl})
	c.out.subs++
}

func (c *fileCtx) where(off int) string {
	_, n := c.li.at(off)
	return fmt.Sprintf("%s:%d", c.path, n)
}

func containsWord(text, word string) bool {
	for from := 0; from < len(text); {
		i := strings.Index(text[from:], word)
		if i < 0 {
			return false
		}
		s, e := from+i, from+i+len(word)
		before := s == 0 || !isNameByte(text[s-1])
		after := e == len(text) || !isNameByte(text[e])
		if before && after {
			return true
		}
		from = s + 1
	}
	return false
}

func isNameByte(b byte) bool {
	return b == '-' || b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// resolve aplica la cadena de contexto de D3 a un ID antiguo con varias capacidades candidatas:
// la misma línea nombra una sola; el fichero (o la capacidad vigente) aporta una sola; la
// tabla de contextos curada fija una.
func (c *fileCtx) resolve(tok, lineText string, caps, cands []string) (string, bool) {
	if len(cands) == 1 {
		return cands[0], true
	}
	pick := func(match func(string) bool) (string, bool) {
		var m []string
		for _, cd := range cands {
			if match(cd) {
				m = append(m, cd)
			}
		}
		if len(m) == 1 {
			return m[0], true
		}
		return "", false
	}
	if cp, ok := pick(func(cd string) bool { return containsWord(lineText, cd) }); ok {
		return cp, true
	}
	if cp, ok := pick(func(cd string) bool { return contains(caps, cd) }); ok {
		return cp, true
	}
	for _, ce := range c.env.ctxs {
		if ce.Token == tok && globMatch(ce.Path, c.path) && contains(cands, ce.Capability) {
			return ce.Capability, true
		}
	}
	return "", false
}

func rangeContiguous(ix *mapIndex, capability, series string, from, to int) error {
	if from > to {
		return fmt.Errorf("los extremos están invertidos")
	}
	seen := map[int]bool{}
	var nums []int
	for _, o := range ix.capOlds[capability] {
		if o.Series == series && o.Num >= from && o.Num <= to && !seen[o.NewNum] {
			seen[o.NewNum] = true
			nums = append(nums, o.NewNum)
		}
	}
	sort.Ints(nums)
	for i := 1; i < len(nums); i++ {
		if nums[i] != nums[i-1]+1 {
			return fmt.Errorf("los IDs nuevos del intervalo no son contiguos (%d y %d)", nums[i-1], nums[i])
		}
	}
	return nil
}

func (c *fileCtx) noteRewritten(r region, tok string) {
	switch c.kind {
	case kindGoProd:
		if r.Scope == scopeString {
			c.out.rewritten[tok] = true
		}
	case kindGoTest:
	default:
		c.out.rewritten[tok] = true
	}
}

func tokenNumber(tok string) int {
	m := tokenRe.FindStringSubmatch(tok)
	if m == nil {
		return -1
	}
	n, _ := strconv.Atoi(m[2])
	return n
}

func (c *fileCtx) processTitle(r region) {
	l, _ := c.li.at(r.Start)
	m := titleRe.FindStringSubmatch(l.Text)
	if m == nil {
		return
	}
	ix := c.env.idx
	nameStart := l.Start + len(m[1])
	nameEnd := nameStart + len(m[2])
	where := c.where(r.Start)
	unmapped := func(format string, args ...any) {
		if c.kind == kindSpec {
			c.fail(where+": "+format, args...)
			return
		}
		c.out.unmapped = append(c.out.unmapped, where+" "+fmt.Sprintf(format, args...))
	}
	if id := rfPattern.InTitle(m[2]); id != "" {
		if e := ix.byNew[id]; e == nil || e.Capability != r.Cap {
			c.fail("%s: título con ID distinto: %q lleva %s, que el mapa no asigna a %s", where, m[2], id, r.Cap)
		}
		return
	}
	tp := parseTitle(m[2])
	if tp.GroupStart < 0 {
		e := ix.byTitle[pair{r.Cap, strings.TrimSpace(m[2])}]
		if e == nil {
			unmapped("el título sin ID %q no tiene entrada en el mapa de %s", m[2], r.Cap)
			return
		}
		c.add(nameEnd, nameEnd, " ("+e.New+")")
		return
	}
	var entry *mapEntry
	for _, old := range tp.Olds {
		e := ix.byPair[pair{r.Cap, old}]
		if e == nil {
			unmapped("el título %q cita %s, que el mapa no tiene en %s", m[2], old, r.Cap)
			return
		}
		if entry != nil && e != entry {
			c.fail("%s: título con ID distinto: %q cita IDs de entradas diferentes", where, m[2])
			return
		}
		entry = e
	}
	c.add(nameStart+tp.GroupStart, nameStart+tp.GroupEnd, "("+entry.New+")")
}

func (c *fileCtx) processRegion(r region) {
	if r.Title {
		c.processTitle(r)
		return
	}
	text := c.content[r.Start:r.End]
	ix := c.env.idx
	for _, h := range findTokens(text) {
		abs := r.Start + h.Start
		l, _ := c.li.at(abs)
		where := c.where(abs)
		tok := h.Text
		if r.Keep {
			c.out.keptAuto++
			continue
		}
		if c.kind == kindGoTest {
			if (r.Scope == scopeString && !c.env.pinned[tok]) || (r.Scope == scopeComment && c.env.testKeep[tok]) {
				c.out.keptAuto++
				continue
			}
		}
		if e := coverage(c.env.res, c.path, r.Scope, tok); e != nil {
			if why := forbiddenResidue(c.path, c.kind, r.Scope, e); why != "" {
				c.fail("%s %s: %s", where, tok, why)
				continue
			}
			c.out.keptRes++
			continue
		}
		cands := ix.oldCaps[tok]
		if len(cands) == 0 {
			c.out.unmapped = append(c.out.unmapped, where+" "+tok)
			if c.kind == kindGoProd && r.Scope == scopeString {
				c.fail("%s %s: literal de producción sin equivalente vivo; corrígelo a mano antes de migrar", where, tok)
			}
			continue
		}
		capName, ok := c.resolve(tok, l.Text, r.Caps, cands)
		if !ok {
			c.fail("%s %s: ambiguo sin resolver [%s]", where, tok, strings.Join(cands, ", "))
			continue
		}
		c.add(abs, abs+len(tok), ix.byPair[pair{capName, tok}].New)
		c.noteRewritten(r, tok)

		prev := tok
		for _, e := range continuations(text, h) {
			eTok := e.token()
			switch e.Sep {
			case "/", ", ", " y ":
				ne := ix.byPair[pair{capName, eTok}]
				if ne == nil {
					cs := ix.oldCaps[eTok]
					if len(cs) == 0 {
						c.out.unmapped = append(c.out.unmapped, where+" "+eTok)
						prev = eTok
						continue
					}
					cc, ok := c.resolve(eTok, l.Text, r.Caps, cs)
					if !ok {
						c.fail("%s %s: ambiguo sin resolver [%s]", where, eTok, strings.Join(cs, ", "))
						prev = eTok
						continue
					}
					ne = ix.byPair[pair{cc, eTok}]
				}
				c.add(r.Start+e.Start, r.Start+e.End, ne.New)
				c.noteRewritten(r, eTok)
			default: // rango
				left, right := ix.byPair[pair{capName, prev}], ix.byPair[pair{capName, eTok}]
				if left == nil || right == nil {
					c.fail("%s rango %s%s%s: un extremo no está en el mapa de %s", where, prev, e.Sep, eTok, capName)
					prev = eTok
					continue
				}
				if err := rangeContiguous(ix, capName, e.Series, tokenNumber(prev), tokenNumber(eTok)); err != nil {
					c.fail("%s rango %s%s%s: %v", where, prev, e.Sep, eTok, err)
					prev = eTok
					continue
				}
				sep := e.Sep
				if sep == "-" || sep == "–" {
					sep = "–"
				}
				c.add(r.Start+e.SepStart, r.Start+e.End, sep+right.New)
				c.noteRewritten(r, eTok)
			}
			prev = eTok
		}
	}
}

// rewriteFile calcula la reescritura de un fichero y devuelve su contenido nuevo.
func rewriteFile(env fileEnv, p string, kind fileKind, specCap, content string) (string, *fileOut) {
	out := &fileOut{rewritten: map[string]bool{}}
	c := &fileCtx{env: env, path: p, kind: kind, content: content, li: newLineIndex(content), out: out}
	var regions []region
	switch kind {
	case kindSpec:
		regions = mdRegions(content, mdSpec, specCap, env.idx)
	case kindODD:
		regions = mdRegions(content, mdODD, "", env.idx)
	case kindGoProd, kindGoTest:
		var err error
		if regions, err = goRegions(content); err != nil {
			c.fail("%s: no se puede analizar el Go: %v", p, err)
			return content, out
		}
	default:
		if strings.HasSuffix(p, ".md") {
			regions = mdRegions(content, mdText, "", env.idx)
		} else {
			regions = []region{{Start: 0, End: len(content), Scope: scopeAll}}
		}
	}
	for _, r := range regions {
		c.processRegion(r)
	}
	if len(out.edits) == 0 {
		return content, out
	}
	updated, err := applyEdits(content, out.edits)
	if err != nil {
		c.fail("%s: %v", p, err)
		return content, out
	}
	if kind == kindGoProd || kind == kindGoTest {
		if updated, err = formatGo(content, updated); err != nil {
			c.fail("%s: %v", p, err)
			return content, out
		}
	}
	return updated, out
}

// ---- migración de un árbol ----

type migrateIn struct {
	Root           string
	Files          []string
	Entries        []mapEntry
	Ctxs           []contextEntry
	Res            []residueEntry
	DeclarePattern bool // insertar `sdd.id_pattern` en axiom.yaml (D10)
}

type result struct {
	Files     map[string]string // solo los ficheros cuyo contenido cambia
	Subs      int
	Failures  []string
	Unmapped  []string
	KeptRes   int
	KeptAuto  int
	Scanned   int
	Unchanged int
}

func under(q, dir string) bool { return dir == "." || strings.HasPrefix(q, dir+"/") }

// migrate calcula en memoria toda la reescritura: nunca escribe. Los ficheros listados que no
// existen, no son regulares o son binarios se omiten sin error.
func migrate(in migrateIn) *result {
	res := &result{Files: map[string]string{}}
	ix := newMapIndex(in.Entries)
	contents := map[string]string{}
	var paths []string
	for _, p := range in.Files {
		if err := safeRel(p); err != nil {
			res.Failures = append(res.Failures, err.Error())
			continue
		}
		if skipPath(p) {
			continue
		}
		abs := filepath.Join(in.Root, filepath.FromSlash(p))
		fi, err := os.Lstat(abs)
		if err != nil || !fi.Mode().IsRegular() {
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				res.Failures = append(res.Failures, fmt.Sprintf("%s: %v", p, err))
			}
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			res.Failures = append(res.Failures, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		head := data
		if len(head) > 8000 {
			head = head[:8000]
		}
		if bytes.IndexByte(head, 0) >= 0 {
			continue
		}
		contents[p] = string(data)
		paths = append(paths, p)
	}
	sort.Strings(paths)
	res.Scanned = len(paths)

	updated := map[string]string{}
	outs := map[string]*fileOut{}
	base := fileEnv{idx: ix, ctxs: in.Ctxs, res: in.Res}
	run := func(p string, env fileEnv) {
		kind, specCap := classify(p)
		content := contents[p]
		switch {
		case kind == kindSpec || kind == kindODD:
		case !strings.Contains(content, "REQ-"):
			return
		}
		nc, out := rewriteFile(env, p, kind, specCap, content)
		outs[p] = out
		updated[p] = nc
	}
	for _, p := range paths {
		if kind, _ := classify(p); kind != kindGoTest {
			run(p, base)
		}
	}

	// Fijados: tokens que esta ejecución reescribe en un fichero que no es de prueba bajo el
	// directorio del paquete (literal de Go de producción o asset embebido).
	byDir := map[string][]string{}
	for _, p := range paths {
		if kind, _ := classify(p); kind == kindGoTest {
			byDir[path.Dir(p)] = append(byDir[path.Dir(p)], p)
		}
	}
	for dir, tests := range byDir {
		pinned := map[string]bool{}
		for q, o := range outs {
			if under(q, dir) {
				for tok := range o.rewritten {
					pinned[tok] = true
				}
			}
		}
		keep := map[string]bool{}
		for _, tp := range tests {
			segs, err := goSegments(contents[tp])
			if err != nil {
				continue
			}
			for _, s := range segs {
				if s.Scope != scopeString {
					continue
				}
				for _, h := range findTokens(contents[tp][s.Start:s.End]) {
					if !pinned[h.Text] {
						keep[h.Text] = true
					}
				}
			}
		}
		env := base
		env.pinned, env.testKeep = pinned, keep
		for _, tp := range tests {
			run(tp, env)
		}
	}

	if in.DeclarePattern {
		if cur, ok := contents["axiom.yaml"]; ok {
			if nu, ok2 := updated["axiom.yaml"]; ok2 {
				cur = nu
			}
			nc, err := ensureSDDBlock(cur)
			if err != nil {
				res.Failures = append(res.Failures, "axiom.yaml: "+err.Error())
			} else {
				updated["axiom.yaml"] = nc
			}
		}
	}

	for _, p := range paths {
		if o := outs[p]; o != nil {
			res.Failures = append(res.Failures, o.fails...)
			res.Unmapped = append(res.Unmapped, o.unmapped...)
			res.KeptRes += o.keptRes
			res.KeptAuto += o.keptAuto
		}
		if nc, ok := updated[p]; ok && nc != contents[p] {
			res.Files[p] = nc
			if o := outs[p]; o != nil {
				res.Subs += o.subs
			}
		} else {
			res.Unchanged++
		}
	}
	return res
}

// ---- pruebas ----

func mkIdx(entries ...mapEntry) *mapIndex { return newMapIndex(entries) }

func rwEnv(entries ...mapEntry) fileEnv { return fileEnv{idx: mkIdx(entries...)} }

func asEntry(capability, newID string, old ...string) mapEntry {
	return mapEntry{Capability: capability, Old: old, New: newID}
}

func mustRewrite(t *testing.T, env fileEnv, p, content string) (string, *fileOut) {
	t.Helper()
	kind, specCap := classify(p)
	got, out := rewriteFile(env, p, kind, specCap, content)
	return got, out
}

func TestRewriteWholeTokenOnly(t *testing.T) {
	env := rwEnv(asEntry("autoskills", "RF-ASK-01", "REQ-1.1"))
	in := "REQ-1.1, REQ-1.10, REQ-11.1, xREQ-1.1, REQ-1.1.2, (REQ-1.1) y REQ-1.1."
	got, out := mustRewrite(t, env, "docs/a.txt", in)
	want := "RF-ASK-01, REQ-1.10, REQ-11.1, xREQ-1.1, REQ-1.1.2, (RF-ASK-01) y RF-ASK-01."
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if len(out.fails) != 0 {
		t.Fatalf("fallos: %v", out.fails)
	}
}

func TestRewriteSuffixedTokenIsOneToken(t *testing.T) {
	env := rwEnv(asEntry("sdd-lane-kickoff", "RF-KOF-01", "REQ-27.2"), asEntry("sdd-lane-kickoff", "RF-KOF-02", "REQ-27.2b"))
	got, _ := mustRewrite(t, env, "docs/a.txt", "REQ-27.2b y REQ-27.2 y REQ-27.2c")
	if got != "RF-KOF-02 y RF-KOF-01 y REQ-27.2c" {
		t.Fatalf("REQ-27.2b es un token y no REQ-27.2 seguido de b: %q", got)
	}
}

func TestRewriteTitlesD6(t *testing.T) {
	env := rwEnv(
		asEntry("workspace-topology", "RF-TOP-01", "REQ-2.3"),
		asEntry("persona-behavior-contract", "RF-PER-01", "REQ-10.1", "REQ-10.2"),
		asEntry("sdd-lane-kickoff", "RF-KOF-01", "REQ-27.2b"),
		mapEntry{Capability: "workspace-topology", Title: "Untitled Behaviour", New: "RF-TOP-02"},
	)
	cases := []struct{ p, in, want string }{
		{"openspec/specs/workspace-topology/spec.md", "### Requirement: Legacy (REQ-2.3) [SUPERSEDED / DEPRECADO]\n", "### Requirement: Legacy (RF-TOP-01) [SUPERSEDED / DEPRECADO]\n"},
		{"openspec/specs/workspace-topology/spec.md", "### Requirement: Legacy (REQ-2.3) — RETIRADO\n", "### Requirement: Legacy (RF-TOP-01) — RETIRADO\n"},
		{"openspec/specs/persona-behavior-contract/spec.md", "### Requirement: Tone (REQ-10.1 & REQ-10.2)\n", "### Requirement: Tone (RF-PER-01)\n"},
		{"openspec/specs/sdd-lane-kickoff/spec.md", "### Requirement: Kick (REQ-27.2b)\n", "### Requirement: Kick (RF-KOF-01)\n"},
		{"openspec/specs/workspace-topology/spec.md", "### Requirement: Untitled Behaviour\n", "### Requirement: Untitled Behaviour (RF-TOP-02)\n"},
		{"openspec/specs/workspace-topology/spec.md", "### Requirement: Untitled Behaviour   \r\n", "### Requirement: Untitled Behaviour (RF-TOP-02)   \r\n"},
		// idempotencia: un título que ya lleva su ID no se toca
		{"openspec/specs/workspace-topology/spec.md", "### Requirement: Legacy (RF-TOP-01)\n", "### Requirement: Legacy (RF-TOP-01)\n"},
	}
	for _, c := range cases {
		got, out := mustRewrite(t, env, c.p, c.in)
		if len(out.fails) != 0 {
			t.Errorf("%q: fallos %v", c.in, out.fails)
		}
		// el caso de espacios finales conserva los espacios: el sufijo se añade tras el nombre
		if c.in == "### Requirement: Untitled Behaviour   \r\n" {
			if !strings.Contains(got, "Untitled Behaviour (RF-TOP-02)") || !strings.HasSuffix(got, "\r\n") {
				t.Errorf("%q -> %q", c.in, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("%q\n got %q\nwant %q", c.in, got, c.want)
		}
	}

	_, out := mustRewrite(t, env, "openspec/specs/workspace-topology/spec.md", "### Requirement: Otro (RF-TOP-02)\n")
	_ = out
	got, out := mustRewrite(t, env, "openspec/specs/workspace-topology/spec.md", "### Requirement: Otro (RF-PER-01)\n")
	if len(out.fails) != 1 || !strings.Contains(out.fails[0], "título con ID distinto") || got != "### Requirement: Otro (RF-PER-01)\n" {
		t.Fatalf("un título con otro ID debe fallar: %v", out.fails)
	}
	_, out = mustRewrite(t, env, "openspec/specs/workspace-topology/spec.md", "### Requirement: Ajeno (REQ-9.9)\n")
	if len(out.fails) != 1 {
		t.Fatalf("un título sin entrada en una spec viva es un fallo: %v", out.fails)
	}
}

func TestRewriteContinuationsAndRangesD5(t *testing.T) {
	env := rwEnv(
		asEntry("pre", "RF-PRE-01", "REQ-21.5"), asEntry("pre", "RF-PRE-02", "REQ-21.6"),
		asEntry("pre", "RF-PRE-03", "REQ-21.8"), asEntry("pre", "RF-PRE-04", "REQ-21.9"),
		asEntry("pre", "RF-PRE-05", "REQ-21.10"), asEntry("pre", "RF-PRE-06", "REQ-21.12"),
		asEntry("odd", "RF-ODD-01", "REQ-24.26"), asEntry("odd", "RF-ODD-02", "REQ-24.27"),
		asEntry("adp", "RF-ADP-01", "REQ-30.9"), asEntry("adp", "RF-ADP-02", "REQ-30.10"), asEntry("adp", "RF-ADP-06", "REQ-30.14"),
		asEntry("gap", "RF-GAP-01", "REQ-50.1"), asEntry("gap", "RF-GAP-03", "REQ-50.3"),
	)
	cases := []struct{ in, want string }{
		{"REQ-21.5/21.6", "RF-PRE-01/RF-PRE-02"},          // lista con /
		{"REQ-24.26 y 24.27", "RF-ODD-01 y RF-ODD-02"},    // lista con y
		{"REQ-21.5, 21.6", "RF-PRE-01, RF-PRE-02"},        // lista con coma
		{"REQ-21.8-21.12", "RF-PRE-03–RF-PRE-06"},         // rango con guion, contiguo (21.11 retirado)
		{"REQ-21.8–21.12", "RF-PRE-03–RF-PRE-06"},         // rango con guion largo
		{"REQ-21.8 a 21.12", "RF-PRE-03 a RF-PRE-06"},     // rango con ` a `
		{"REQ-21.8 al 21.12", "RF-PRE-03 al RF-PRE-06"},   // rango con ` al `
		{"REQ-30.9 a REQ-30.14", "RF-ADP-01 a RF-ADP-06"}, // extremos con prefijo completo
		{"REQ-21.5/21.7", "RF-PRE-01/21.7"},               // continuación sin mapa: se conserva
	}
	for _, c := range cases {
		got, out := mustRewrite(t, env, "docs/a.txt", c.in)
		if len(out.fails) != 0 {
			t.Errorf("%q: fallos %v", c.in, out.fails)
		}
		if got != c.want {
			t.Errorf("%q\n got %q\nwant %q", c.in, got, c.want)
		}
	}
	// Rango no contiguo: RF-GAP-01 y RF-GAP-03 con un hueco en medio.
	got, out := mustRewrite(t, env, "docs/a.txt", "REQ-50.1-50.3")
	if len(out.fails) != 1 || !strings.Contains(out.fails[0], "contiguos") || strings.Contains(out.fails[0], "docs/a.txt:0") {
		t.Fatalf("un rango no contiguo es un fallo con ubicación: %v", out.fails)
	}
	_ = got
	// Un rango con un extremo sin mapear.
	_, out = mustRewrite(t, env, "docs/a.txt", "REQ-50.1-50.2")
	if len(out.fails) != 1 || !strings.Contains(out.fails[0], "extremo") {
		t.Fatalf("un extremo sin mapear es un fallo: %v", out.fails)
	}
}

func TestRewriteAmbiguousContextChainD3(t *testing.T) {
	entries := []mapEntry{
		asEntry("autoskills", "RF-ASK-01", "REQ-1.1"),
		asEntry("workspace-topology", "RF-TOP-01", "REQ-1.1"),
		asEntry("multi-project-hub", "RF-HUB-01", "REQ-1.1"),
	}
	env := rwEnv(entries...)

	// regla 3: la misma línea nombra exactamente una capacidad candidata
	got, out := mustRewrite(t, env, "docs/a.md", "Ver REQ-1.1 de `workspace-topology`.\nVer REQ-1.1 de autoskills.\n")
	if got != "Ver RF-TOP-01 de `workspace-topology`.\nVer RF-ASK-01 de autoskills.\n" || len(out.fails) != 0 {
		t.Fatalf("regla de la misma línea: %q %v", got, out.fails)
	}
	// una línea que nombra dos candidatas no resuelve
	_, out = mustRewrite(t, env, "docs/a.md", "REQ-1.1 entre autoskills y workspace-topology\n")
	if len(out.fails) != 1 || !strings.Contains(out.fails[0], "ambiguo sin resolver") {
		t.Fatalf("dos candidatas en la línea no resuelven: %v", out.fails)
	}
	// regla 4: en una spec, su propia capacidad
	got, out = mustRewrite(t, env, "openspec/specs/multi-project-hub/spec.md", "# Hub\n\nVer REQ-1.1 aquí.\n")
	if got != "# Hub\n\nVer RF-HUB-01 aquí.\n" || len(out.fails) != 0 {
		t.Fatalf("contexto de fichero en spec: %q %v", got, out.fails)
	}
	// regla 4: en ODD, la única capacidad candidata entre sus `### Capacidad:`
	odd := "# F\n\n### Capacidad: autoskills\n\n```openspec\ntexto\n```\n\n### Capacidad: otra-capacidad\n\nCubre: REQ-1.1\n"
	got, out = mustRewrite(t, env, "odd/tasks/f.md", odd)
	if !strings.Contains(got, "Cubre: RF-ASK-01") || len(out.fails) != 0 {
		t.Fatalf("contexto de fichero en ODD: %q %v", got, out.fails)
	}
	// regla 5: tabla de contexto curada, solo si la capacidad es candidata
	env5 := env
	env5.ctxs = []contextEntry{{Path: "internal/hub/*.go", Token: "REQ-1.1", Capability: "multi-project-hub", Reason: "el comentario nombra el hub"}}
	got, out = mustRewrite(t, env5, "internal/hub/init.go", "package hub\n\n// REQ-1.1 registra el proyecto.\n")
	if !strings.Contains(got, "// RF-HUB-01 registra") || len(out.fails) != 0 {
		t.Fatalf("tabla de contexto: %q %v", got, out.fails)
	}
	env5.ctxs[0].Capability = "capacidad-inexistente"
	_, out = mustRewrite(t, env5, "internal/hub/init.go", "package hub\n\n// REQ-1.1 registra el proyecto.\n")
	if len(out.fails) != 1 {
		t.Fatalf("una entrada de contexto con una capacidad no candidata no se aplica: %v", out.fails)
	}
	// regla 6: nada resuelve; el fallo nombra ruta:línea, token y candidatas
	_, out = mustRewrite(t, env, "internal/x/y.go", "package y\n\n// REQ-1.1 sin contexto\n")
	if len(out.fails) != 1 || !strings.Contains(out.fails[0], "internal/x/y.go:3 REQ-1.1") || !strings.Contains(out.fails[0], "autoskills, multi-project-hub, workspace-topology") {
		t.Fatalf("fallo sin contexto: %v", out.fails)
	}
}

func TestRewriteGoCommentsLiteralsAndFormat(t *testing.T) {
	env := rwEnv(asEntry("workspace-topology", "RF-TOP-01", "REQ-2.3"), asEntry("sdd-lane-kickoff", "RF-KOF-02", "REQ-27.2b"))
	prod := "package foo\n\n// Ver REQ-2.3 de la topología.\nconst msg = \"falta REQ-27.2b\" // REQ-27.2b\n"
	got, out := mustRewrite(t, env, "internal/foo/foo.go", prod)
	if want := "package foo\n\n// Ver RF-TOP-01 de la topología.\nconst msg = \"falta RF-KOF-02\" // RF-KOF-02\n"; got != want || len(out.fails) != 0 {
		t.Fatalf("Go de producción:\n got %q\nwant %q\n%v", got, want, out.fails)
	}
	if !out.rewritten["REQ-27.2b"] || out.rewritten["REQ-2.3"] {
		t.Fatalf("solo el literal fija el token: %v", out.rewritten)
	}

	// gofmt: el reescrito se realinea si el original estaba formateado
	aligned := "package foo\n\nvar (\n\ta   = 1 // REQ-2.3\n\tbcd = 2\n)\n"
	got, _ = mustRewrite(t, env, "internal/foo/a.go", aligned)
	if want := "package foo\n\nvar (\n\ta   = 1 // RF-TOP-01\n\tbcd = 2\n)\n"; got != want {
		t.Fatalf("gofmt: %q", got)
	}
	// un original sin formatear no se reformatea (se conservan sus bytes)
	messy := "package foo\r\n\r\n//   REQ-2.3   x\r\nvar   a  =  1\r\n"
	got, _ = mustRewrite(t, env, "internal/foo/m.go", messy)
	if want := "package foo\r\n\r\n//   RF-TOP-01   x\r\nvar   a  =  1\r\n"; got != want {
		t.Fatalf("un Go sin formatear conserva sus bytes: %q", got)
	}

	// un literal de producción sin equivalente es un fallo; en un comentario solo se informa
	_, out = mustRewrite(t, env, "internal/foo/b.go", "package foo\n\nconst m = \"REQ-24.62\"\n")
	if len(out.fails) != 1 || !strings.Contains(out.fails[0], "internal/foo/b.go:3 REQ-24.62") {
		t.Fatalf("literal de producción no mapeado: %v", out.fails)
	}
	_, out = mustRewrite(t, env, "internal/foo/c.go", "package foo\n\n// REQ-24.62 retirado\n")
	if len(out.fails) != 0 || len(out.unmapped) != 1 {
		t.Fatalf("un comentario sin equivalente se informa sin fallar: %v %v", out.fails, out.unmapped)
	}

	// Go con un literal sin cerrar: fallo, no pánico
	_, out = mustRewrite(t, env, "internal/foo/d.go", "package foo\n\nvar x = \"REQ-2.3\n")
	if len(out.fails) != 1 {
		t.Fatalf("un fuente ilegible es un fallo: %v", out.fails)
	}
}

func TestRewriteGoTestsPinnedLiteralsAndFixtureComments(t *testing.T) {
	files := map[string]string{
		"internal/foo/foo.go":      "package foo\n\nconst msg = \"falta REQ-27.2b\"\n",
		"internal/foo/foo_test.go": "package foo\n\n// REQ-1.10 describe el fixture; REQ-2.3 es una cita.\nvar _ = \"REQ-27.2b\"\nvar fixture = \"## REQ-1.10 en un fixture\"\n",
	}
	root := t.TempDir()
	for p, c := range files {
		writeFile(t, root, p, c)
	}
	entries := []mapEntry{asEntry("sdd-lane-kickoff", "RF-KOF-02", "REQ-27.2b"), asEntry("workspace-topology", "RF-TOP-01", "REQ-2.3"), asEntry("autoskills", "RF-ASK-03", "REQ-1.10")}
	res := migrate(migrateIn{Root: root, Files: []string{"internal/foo/foo.go", "internal/foo/foo_test.go"}, Entries: entries})
	if len(res.Failures) != 0 {
		t.Fatalf("fallos: %v", res.Failures)
	}
	if got := res.Files["internal/foo/foo.go"]; !strings.Contains(got, "RF-KOF-02") {
		t.Fatalf("producción: %q", got)
	}
	got := res.Files["internal/foo/foo_test.go"]
	// pinned: el literal con REQ-27.2b se reescribe; el fixture REQ-1.10 y su comentario se conservan;
	// la cita REQ-2.3 del comentario (que no está en ningún literal) se reescribe.
	if !strings.Contains(got, `var _ = "RF-KOF-02"`) || !strings.Contains(got, `"## REQ-1.10 en un fixture"`) ||
		!strings.Contains(got, "// REQ-1.10 describe el fixture; RF-TOP-01 es una cita.") {
		t.Fatalf("Go de pruebas: %q", got)
	}
}

func TestRewriteODDFencesAndRetiredCapabilities(t *testing.T) {
	env := rwEnv(asEntry("multi-project-hub", "RF-HUB-01", "REQ-2.1"), asEntry("multi-project-hub", "RF-HUB-02", "REQ-2.2"))
	odd := "# F\n\nCubre: REQ-2.2 y `REQ-2.1`.\n\n## Spec viva\n\n### Capacidad: multi-project-hub\n\n````openspec\n## MODIFIED Requirements\n\n### Requirement: Hub Registration (REQ-2.1)\n\nEl hub cita REQ-2.2 y `REQ-2.1`.\n\n```text\nREQ-2.1 anidado\n```\n````\n\n### Capacidad: knowledge-query-engine\n\n```openspec\n### Requirement: Retirada (REQ-7.7)\n\nCita REQ-2.1.\n```\n\n```go\nvar x = \"REQ-2.1\"\n```\n"
	got, out := mustRewrite(t, env, "odd/tasks/f.md", odd)
	if len(out.fails) != 0 {
		t.Fatalf("fallos: %v", out.fails)
	}
	for _, want := range []string{
		"Cubre: RF-HUB-02 y `RF-HUB-01`.",                      // fuera de vallas se reescribe también el código en línea
		"### Requirement: Hub Registration (RF-HUB-01)",        // título de la delta
		"El hub cita RF-HUB-02 y `REQ-2.1`.",                   // regla de specs: código en línea conservado
		"REQ-2.1 anidado",                                      // valla anidada: se conserva
		"### Requirement: Retirada (REQ-7.7)\n\nCita REQ-2.1.", // capacidad retirada: valla entera conservada
		"var x = \"REQ-2.1\"",                                  // vallas que no son openspec: se conservan
	} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en:\n%s", want, got)
		}
	}
}

func TestRewriteSpecMarkdownKeepsInlineCodeAndFences(t *testing.T) {
	env := rwEnv(asEntry("autoskills", "RF-ASK-01", "REQ-1.1"))
	spec := "# Autoskills\n\nCubre REQ-1.1 y el ejemplo `REQ-1.1`.\n\n### Requirement: Detección (REQ-1.1)\n\n```markdown\n### Requirement: Falso (REQ-1.1)\nREQ-1.1\n```\n"
	got, out := mustRewrite(t, env, "openspec/specs/autoskills/spec.md", spec)
	want := "# Autoskills\n\nCubre RF-ASK-01 y el ejemplo `REQ-1.1`.\n\n### Requirement: Detección (RF-ASK-01)\n\n```markdown\n### Requirement: Falso (REQ-1.1)\nREQ-1.1\n```\n"
	if got != want || len(out.fails) != 0 {
		t.Fatalf("got  %q\nwant %q\n%v", got, want, out.fails)
	}
	if out.keptAuto != 3 {
		t.Errorf("deben conservarse automáticamente 3 apariciones (código en línea y vallas): %d", out.keptAuto)
	}
}

func TestRewriteResidueKeepsCoveredTokens(t *testing.T) {
	env := rwEnv(asEntry("autoskills", "RF-ASK-01", "REQ-1.1"), asEntry("autoskills", "RF-ASK-02", "REQ-30.4"))
	env.res = []residueEntry{residue([]string{"docs/guia.md"}, scopeAll, defaultPatternExample, "ejemplo", "REQ-30.4")}
	got, out := mustRewrite(t, env, "docs/guia.md", "Usa REQ-1.1 y el ejemplo REQ-30.4.\n")
	if got != "Usa RF-ASK-01 y el ejemplo REQ-30.4.\n" || out.keptRes != 1 {
		t.Fatalf("%q keptRes=%d", got, out.keptRes)
	}
	// un residuo que cubre un literal de producción es un fallo
	env.res = []residueEntry{{Paths: []string{"internal/foo/foo.go"}, Scope: scopeAll, Category: "legacy-citation", Reason: "x", Tokens: []string{"REQ-1.1"}}}
	_, out = mustRewrite(t, env, "internal/foo/foo.go", "package foo\n\nconst m = \"REQ-1.1\"\n")
	if len(out.fails) != 1 {
		t.Fatalf("un residuo sobre un literal de producción es un fallo: %v", out.fails)
	}
}

// ---- árbol miniatura: bytes, idempotencia y fallos sin escribir ----

func TestRewriteMiniatureIsIdempotent(t *testing.T) {
	root := miniRoot(t)
	env := runEnv{Root: root, List: walkList(root), Logf: t.Logf}
	if err := runMode("map", env); err != nil {
		t.Fatal(err)
	}
	if err := runMode("write", env); err != nil {
		t.Fatalf("write: %v", err)
	}
	entries, _ := readMapFile(root)
	files, _ := walkList(root)()
	ctxs, res := loadMeta(t, root)
	second := migrate(migrateIn{Root: root, Files: files, Entries: entries, Ctxs: ctxs, Res: res, DeclarePattern: true})
	if len(second.Files) != 0 || len(second.Failures) != 0 {
		t.Fatalf("la segunda pasada debe ser un no-op: %v %v", second.Failures, keys(second.Files))
	}
}

func TestRewritePreservesCRLFAndBOMAndLeavesUntouchedFilesAlone(t *testing.T) {
	root := miniRoot(t)
	if err := runMode("map", runEnv{Root: root, List: walkList(root), Logf: t.Logf}); err != nil {
		t.Fatal(err)
	}
	crlf := "# Guía\r\n\r\nVer REQ-1.2 de autoskills.\r\nSegunda línea REQ-2.3.\r\n"
	bom := "\ufeffGuía con BOM: REQ-27.2b y REQ-1.2 de autoskills.\n"
	clean := "Sin citas heredadas.\r\n"
	writeFile(t, root, "docs/crlf.md", crlf)
	writeFile(t, root, "docs/bom.md", bom)
	writeFile(t, root, "docs/clean.md", clean)
	if err := os.WriteFile(filepath.Join(root, "assets", "blob.bin"), []byte("REQ-1.2\x00\x01REQ-1.2"), 0o644); err != nil {
		t.Fatal(err)
	}
	cleanPath := filepath.Join(root, "docs", "clean.md")
	old := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(cleanPath, old, old); err != nil {
		t.Fatal(err)
	}
	list := walkList(root)
	listWithGhost := func() ([]string, error) {
		files, err := list()
		return append(files, "docs/no-existe.md", "assets/ausente.bin"), err
	}
	if err := runMode("write", runEnv{Root: root, List: listWithGhost, Logf: t.Logf}); err != nil {
		t.Fatalf("write: %v", err)
	}

	gotCRLF := readFile(t, root, "docs/crlf.md")
	if want := strings.NewReplacer("REQ-1.2", "RF-ASK-02", "REQ-2.3", "RF-TOP-03").Replace(crlf); gotCRLF != want {
		t.Fatalf("CRLF no preservado:\n got %q\nwant %q", gotCRLF, want)
	}
	gotBOM := readFile(t, root, "docs/bom.md")
	if want := strings.NewReplacer("REQ-1.2", "RF-ASK-02", "REQ-27.2b", "RF-KOF-02").Replace(bom); gotBOM != want || !strings.HasPrefix(gotBOM, "\ufeff") {
		t.Fatalf("BOM no preservado:\n got %q\nwant %q", gotBOM, want)
	}
	if readFile(t, root, "docs/clean.md") != clean {
		t.Fatal("un fichero sin sustituciones cambió de contenido")
	}
	fi, err := os.Stat(cleanPath)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.ModTime().Equal(old) {
		t.Fatalf("un fichero sin sustituciones no se reescribe: mtime %v, quería %v", fi.ModTime(), old)
	}
	if data, _ := os.ReadFile(filepath.Join(root, "assets", "blob.bin")); string(data) != "REQ-1.2\x00\x01REQ-1.2" {
		t.Fatal("un fichero binario no se toca")
	}
}

func TestRewriteFailsWithoutWritingAnythingOnUnresolvedAmbiguity(t *testing.T) {
	root := miniRoot(t)
	env := runEnv{Root: root, List: walkList(root), Logf: t.Logf}
	if err := runMode("map", env); err != nil {
		t.Fatal(err)
	}
	// REQ-1.1 está en autoskills y workspace-topology; este texto no da contexto alguno.
	writeFile(t, root, "docs/ambiguo.md", "Sin contexto: REQ-1.1.\n")
	before := snapshot(t, root)
	for _, mode := range []string{"plan", "write"} {
		err := runMode(mode, env)
		if err == nil || !strings.Contains(err.Error(), "docs/ambiguo.md:1 REQ-1.1") {
			t.Fatalf("%s debe fallar nombrando ruta:línea y token: %v", mode, err)
		}
		if after := snapshot(t, root); !equalMaps(before, after) {
			t.Fatalf("%s escribió algo pese al fallo", mode)
		}
	}
}

func TestRewriteFailsWithoutWritingOnNonContiguousRangeProductionLiteralAndTitleMismatch(t *testing.T) {
	cases := map[string]struct{ path, content, want string }{
		"rango no contiguo":     {"docs/rango.md", "Ver REQ-1.1-1.10 de autoskills.\n", "contiguos"},
		"literal de producción": {"internal/foo/lit.go", "package foo\n\nconst m = \"REQ-24.62\"\n", "internal/foo/lit.go:3 REQ-24.62"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := miniRoot(t)
			env := runEnv{Root: root, List: walkList(root), Logf: t.Logf}
			if err := runMode("map", env); err != nil {
				t.Fatal(err)
			}
			if name == "rango no contiguo" {
				// REQ-1.1, REQ-1.2 y REQ-1.10 de autoskills dan RF-ASK-01, 02 y 03 (contiguos): se
				// abre un hueco añadiendo al mapa un requisito intermedio con el número 09.
				entries, _ := readMapFile(root)
				entries = append(entries, asEntry("autoskills", "RF-ASK-09", "REQ-1.5"))
				text, _ := emitMap(entries)
				writeFile(t, root, "openspec/id-map.yaml", text)
			}
			writeFile(t, root, c.path, c.content)
			before := snapshot(t, root)
			err := runMode("write", env)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("write debe fallar con %q: %v", c.want, err)
			}
			if !equalMaps(before, snapshot(t, root)) {
				t.Fatal("write escribió algo pese al fallo")
			}
		})
	}

	t.Run("título con otro ID", func(t *testing.T) {
		root := miniRoot(t)
		env := runEnv{Root: root, List: walkList(root), Logf: t.Logf}
		if err := runMode("map", env); err != nil {
			t.Fatal(err)
		}
		spec := readFile(t, root, "openspec/specs/autoskills/spec.md")
		writeFile(t, root, "openspec/specs/autoskills/spec.md", strings.Replace(spec, "(REQ-1.2)", "(RF-ASK-09)", 1))
		before := snapshot(t, root)
		err := runMode("write", env)
		if err == nil || !strings.Contains(err.Error(), "título con ID distinto") {
			t.Fatalf("un título con un ID distinto del mapa debe fallar: %v", err)
		}
		if !equalMaps(before, snapshot(t, root)) {
			t.Fatal("write escribió algo pese al fallo")
		}
	})
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
