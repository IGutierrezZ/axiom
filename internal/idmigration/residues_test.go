package idmigration_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	residuesSchema = "axiom.id-residues/v1"
	contextsSchema = "axiom.id-contexts/v1"
)

// residueEntry es una exención de la prueba de residuos (D7). Granularidad: rutas, ámbito y
// tokens; nunca números de línea.
type residueEntry struct {
	Paths    []string `yaml:"paths"`
	Scope    string   `yaml:"scope"`
	Category string   `yaml:"category"`
	Tokens   []string `yaml:"tokens"`
	Reason   string   `yaml:"reason"`
}

// contextEntry resuelve un ID ambiguo en una ruta (D3, regla 5).
type contextEntry struct {
	Path       string `yaml:"path"`
	Token      string `yaml:"token"`
	Capability string `yaml:"capability"`
	Reason     string `yaml:"reason"`
}

// Conjuntos cerrados de RF-RID-16.
var (
	residueCategories = []string{"default-pattern-fixture", "default-pattern-definition", "default-pattern-example", "legacy-citation", "dated-history"}
	residueScopes     = []string{scopeAll, scopeComment, scopeString, scopeCode}
)

const defaultPatternExample = "default-pattern-example"

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func decodeStrict(data []byte, into any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(into)
}

// parseResidues lee y valida la lista de residuos.
func parseResidues(data []byte) ([]residueEntry, error) {
	var doc struct {
		Schema  string         `yaml:"schema"`
		Entries []residueEntry `yaml:"entries"`
	}
	if err := decodeStrict(data, &doc); err != nil {
		return nil, fmt.Errorf("residues.yaml: %w", err)
	}
	if doc.Schema != residuesSchema {
		return nil, fmt.Errorf("residues.yaml: el esquema %q no es %q", doc.Schema, residuesSchema)
	}
	var errs []error
	for i, e := range doc.Entries {
		if err := validateResidue(e); err != nil {
			errs = append(errs, fmt.Errorf("entrada %d (%s): %w", i+1, strings.Join(e.Paths, ", "), err))
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return doc.Entries, nil
}

func parseContexts(data []byte) ([]contextEntry, error) {
	var doc struct {
		Schema  string         `yaml:"schema"`
		Entries []contextEntry `yaml:"entries"`
	}
	if err := decodeStrict(data, &doc); err != nil {
		return nil, fmt.Errorf("contexts.yaml: %w", err)
	}
	if doc.Schema != contextsSchema {
		return nil, fmt.Errorf("contexts.yaml: el esquema %q no es %q", doc.Schema, contextsSchema)
	}
	for i, e := range doc.Entries {
		if e.Path == "" || e.Token == "" || e.Capability == "" || strings.TrimSpace(e.Reason) == "" {
			return nil, fmt.Errorf("contexts.yaml: la entrada %d necesita path, token, capability y reason", i+1)
		}
	}
	return doc.Entries, nil
}

func allPaths(e residueEntry, ok func(string) bool) bool {
	for _, p := range e.Paths {
		if !ok(p) {
			return false
		}
	}
	return len(e.Paths) > 0
}

// tokensRequired dice si la entrada debe listar tokens: solo se excusa a los casos medidos
// (fixtures, ejemplos en specs, historia fechada y la carpeta de este cambio).
func tokensRequired(e residueEntry) bool {
	switch e.Category {
	case "default-pattern-fixture":
		if e.Scope == scopeString && allPaths(e, func(p string) bool { return strings.HasSuffix(p, "_test.go") }) {
			return false
		}
		if e.Scope == scopeAll && allPaths(e, func(p string) bool { return strings.Contains(p, "testdata/") }) {
			return false
		}
	case defaultPatternExample:
		if e.Scope == scopeCode && allPaths(e, func(p string) bool {
			return p == indexPath || (strings.HasPrefix(p, "openspec/specs/") && strings.HasSuffix(p, "/spec.md"))
		}) {
			return false
		}
	case "dated-history":
		if allPaths(e, func(p string) bool {
			return strings.HasPrefix(p, "docs/archive/") || strings.HasPrefix(p, "docs/audits/")
		}) {
			return false
		}
	case "legacy-citation":
		if allPaths(e, func(p string) bool {
			return strings.HasPrefix(p, "openspec/changes/migrar-ids-rf/") || globMatch("openspec/changes/archive/*-migrar-ids-rf/**", p)
		}) {
			return false
		}
	}
	return true
}

const assetsSkills = "internal/assets/skills/"

// validateResidue aplica la validación estructural de D7 y los huecos 8 y 9.
func validateResidue(e residueEntry) error {
	if !contains(residueCategories, e.Category) {
		return fmt.Errorf("la categoría %q no está en el conjunto cerrado %v", e.Category, residueCategories)
	}
	if !contains(residueScopes, e.Scope) {
		return fmt.Errorf("el ámbito %q no está en %v", e.Scope, residueScopes)
	}
	if strings.TrimSpace(e.Reason) == "" {
		return errors.New("el motivo está vacío")
	}
	if len(e.Paths) == 0 {
		return errors.New("no lista rutas")
	}
	if len(e.Tokens) == 0 && tokensRequired(e) {
		return errors.New("`tokens` es obligatorio en esta categoría y ámbito")
	}
	for _, p := range e.Paths {
		last := p[strings.LastIndex(p, "/")+1:]
		if (e.Scope == scopeString || e.Scope == scopeAll) && strings.HasSuffix(last, ".go") && !strings.HasSuffix(last, "_test.go") {
			return fmt.Errorf("la ruta %q alcanza literales de Go de producción: un mensaje de producción nunca es residuo", p)
		}
		if strings.HasPrefix(p, assetsSkills) && e.Category != defaultPatternExample {
			return fmt.Errorf("la ruta %q alcanza un asset embebido de producción: solo se admite el ejemplo del patrón por defecto (%s)", p, defaultPatternExample)
		}
	}
	return nil
}

// coverage devuelve la primera entrada que cubre una aparición.
func coverage(res []residueEntry, p, scope, token string) *residueEntry {
	for i := range res {
		e := &res[i]
		if e.Scope != scopeAll && e.Scope != scope {
			continue
		}
		if len(e.Tokens) > 0 && !contains(e.Tokens, token) {
			continue
		}
		for _, pat := range e.Paths {
			if globMatch(pat, p) {
				return e
			}
		}
	}
	return nil
}

// forbiddenResidue dice si una entrada que cubre una aparición es inadmisible: ni un literal de
// Go de producción ni un asset embebido de producción (salvo el ejemplo del patrón por defecto).
func forbiddenResidue(p string, kind fileKind, scope string, e *residueEntry) string {
	if kind == kindGoProd && scope == scopeString {
		return "un literal de Go de producción nunca es residuo"
	}
	if strings.HasPrefix(p, assetsSkills) && e.Category != defaultPatternExample {
		return "un asset embebido de producción solo admite el ejemplo del patrón por defecto"
	}
	return ""
}

// ---- recorrido del repositorio ----

// occurrence es una aparición de un ID heredado con su línea y su ámbito.
type occurrence struct {
	Path  string
	Line  int
	Token string
	Scope string
}

const scopeText = "text"

// occurrencesOf clasifica por ámbito cada aparición de un fichero: `go/scanner` para Go y
// tramos de código para Markdown.
func occurrencesOf(p, content string) ([]occurrence, error) {
	li := newLineIndex(content)
	var out []occurrence
	add := func(off int, tok, scope string) {
		_, n := li.at(off)
		out = append(out, occurrence{Path: p, Line: n, Token: tok, Scope: scope})
	}
	switch {
	case strings.HasSuffix(p, ".go"):
		segs, err := goSegments(content)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		covered := make([]bool, len(content))
		for _, s := range segs {
			for _, h := range findTokens(content[s.Start:s.End]) {
				add(s.Start+h.Start, h.Text, s.Scope)
			}
			for i := s.Start; i < s.End; i++ {
				covered[i] = true
			}
		}
		for _, h := range findTokens(content) {
			if !covered[h.Start] {
				add(h.Start, h.Text, "go-code")
			}
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	case strings.HasSuffix(p, ".md"):
		for _, l := range analyzeMarkdown(splitLines(content)) {
			if l.InFence {
				for _, h := range findTokens(l.Text) {
					add(l.Start+h.Start, h.Text, scopeCode)
				}
				continue
			}
			for _, s := range inlineSegments(l.Text) {
				scope := scopeText
				if s.Scope == scopeCode {
					scope = scopeCode
				}
				for _, h := range findTokens(l.Text[s.Start:s.End]) {
					add(l.Start+s.Start+h.Start, h.Text, scope)
				}
			}
		}
	default:
		for _, h := range findTokens(content) {
			add(h.Start, h.Text, scopeText)
		}
	}
	return out, nil
}

// obligationExempt: ubicaciones donde un ID heredado no necesita residuo (RF-RID-16).
func obligationExempt(p string) bool {
	return strings.HasPrefix(p, "openspec/changes/archive/") || p == idMapPath || isInputFile(p)
}

// checkResidues aplica las dos reglas de TestRepositoryLegacyIDResidues sobre el contenido de
// los ficheros versionados: obligación (toda aparición fuera de la historia y del mapa está
// cubierta y no es inadmisible) y caducidad (toda entrada y cada token de su lista casa con
// alguna aparición en cualquier fichero versionado, el archivo incluido). Devuelve los fallos.
func checkResidues(files map[string]string, res []residueEntry) []string {
	var fails []string
	type use struct{ matched map[string]bool }
	uses := make([]use, len(res))
	for i := range uses {
		uses[i].matched = map[string]bool{}
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if p == idMapPath || isInputFile(p) || strings.IndexByte(files[p], 0) >= 0 {
			continue
		}
		kind, _ := classify(p)
		occs, err := occurrencesOf(p, files[p])
		if err != nil {
			fails = append(fails, err.Error())
			continue
		}
		for _, o := range occs {
			e := coverage(res, o.Path, o.Scope, o.Token)
			// La caducidad cuenta toda aparición que case con el alcance de cada entrada,
			// aunque otra entrada anterior la cubra primero.
			for i := range res {
				if coverage(res[i:i+1], o.Path, o.Scope, o.Token) != nil {
					uses[i].matched[o.Token] = true
					uses[i].matched[""] = true
				}
			}
			if obligationExempt(p) {
				continue
			}
			switch {
			case e == nil:
				fails = append(fails, fmt.Sprintf("%s:%d %s: cita heredada sin residuo declarado", o.Path, o.Line, o.Token))
			default:
				if why := forbiddenResidue(o.Path, kind, o.Scope, e); why != "" {
					fails = append(fails, fmt.Sprintf("%s:%d %s: %s", o.Path, o.Line, o.Token, why))
				}
			}
		}
	}
	for i, e := range res {
		label := fmt.Sprintf("entrada %d (%s, %s)", i+1, strings.Join(e.Paths, ", "), e.Category)
		if !uses[i].matched[""] {
			fails = append(fails, label+": caducada, ya no casa con ninguna aparición")
			continue
		}
		for _, tok := range e.Tokens {
			if !uses[i].matched[tok] {
				fails = append(fails, fmt.Sprintf("%s: el token %s caducó, ya no aparece", label, tok))
			}
		}
	}
	return fails
}

// ---- raíz del repositorio y lista de ficheros ----

func sameDir(a, b string) bool {
	ia, err := os.Stat(a)
	if err != nil {
		return false
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ia, ib)
}

// repoRootFrom resuelve la raíz como ruta absoluta y la comprueba con
// `git -C <raíz> rev-parse --show-toplevel`. Sin git, o fuera de un repositorio, devuelve error:
// la prueba falla, nunca se omite (hueco 8).
func repoRootFrom(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	out, err := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("no se pudo ejecutar git en %s (¿falta git o no es un repositorio?): %w", abs, err)
	}
	top := filepath.Clean(filepath.FromSlash(strings.TrimSpace(string(out))))
	if !sameDir(top, abs) {
		return "", fmt.Errorf("la raíz %s no es la raíz del repositorio (git indica %s)", abs, top)
	}
	return abs, nil
}

// repoRoot es la raíz del repositorio de Axiom, calculada desde el directorio del paquete.
func repoRoot() (string, error) { return repoRootFrom(filepath.Join("..", "..")) }

// listVersioned devuelve `git -C <raíz> ls-files -z`, de solo lectura, rechazando toda ruta
// absoluta o con `..`.
func listVersioned(root string) ([]string, error) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files falló en %s: %w", root, err)
	}
	var files []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" {
			continue
		}
		if err := safeRel(p); err != nil {
			return nil, err
		}
		files = append(files, p)
	}
	return files, nil
}

// readFiles lee a memoria los ficheros de la lista; omite los inexistentes y los no regulares.
func readFiles(root string, files []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range files {
		fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		out[p] = string(data)
	}
	return out, nil
}

// ---- pruebas ----

func residue(paths []string, scope, category, reason string, tokens ...string) residueEntry {
	return residueEntry{Paths: paths, Scope: scope, Category: category, Reason: reason, Tokens: tokens}
}

func TestResidueUndeclaredCitationFailsWithPathAndLine(t *testing.T) {
	files := map[string]string{
		"docs/guia.md":    "uno\ndos REQ-5.5 tres\n",
		"internal/a/a.go": "package a\n\n// ok\nvar x = \"REQ-7.7\"\n",
		"internal/a/b.go": "package a\n",
		"notas/otra.txt":  "sin citas\n",
	}
	fails := checkResidues(files, nil)
	want := []string{"docs/guia.md:2 REQ-5.5", "internal/a/a.go:4 REQ-7.7"}
	for _, w := range want {
		if !anyContains(fails, w) {
			t.Errorf("falta el fallo %q en %v", w, fails)
		}
	}
	if len(fails) != 2 {
		t.Errorf("fallos inesperados: %v", fails)
	}
}

func TestResidueDeclaredResiduesAreAccepted(t *testing.T) {
	files := map[string]string{
		"docs/guia.md":                 "ejemplo REQ-30.4 y `REQ-31.1`\n",
		"internal/x/x_test.go":         "package x\n\nvar f = \"REQ-1.1\"\n",
		"internal/x/testdata/f.txt":    "REQ-2.2\n",
		"docs/archive/viejo.md":        "REQ-9.9\n",
		"openspec/changes/archive/a/b": "REQ-4.4\n",
		"openspec/id-map.yaml":         "old: [REQ-1.1]\n",
		"openspec/specs/x/spec.md":     "Ejemplo `REQ-40.1`.\n",
	}
	res := []residueEntry{
		residue([]string{"docs/guia.md"}, scopeAll, defaultPatternExample, "ejemplo para otros proyectos", "REQ-30.4"),
		residue([]string{"docs/guia.md"}, scopeCode, defaultPatternExample, "ejemplo en código", "REQ-31.1"),
		residue([]string{"internal/x/*_test.go"}, scopeString, "default-pattern-fixture", "fixture del patrón por defecto"),
		residue([]string{"**/testdata/**"}, scopeAll, "default-pattern-fixture", "fixtures"),
		residue([]string{"docs/archive/**"}, scopeAll, "dated-history", "historia fechada"),
		residue([]string{"openspec/specs/**/spec.md"}, scopeCode, defaultPatternExample, "ejemplos del patrón por defecto"),
	}
	if fails := checkResidues(files, res); len(fails) != 0 {
		t.Fatalf("los residuos declarados no deben fallar: %v", fails)
	}
}

func TestResidueExpiredEntryAndTokenFail(t *testing.T) {
	files := map[string]string{"docs/guia.md": "ejemplo REQ-30.4\n"}
	res := []residueEntry{
		residue([]string{"docs/guia.md"}, scopeAll, defaultPatternExample, "vivo", "REQ-30.4", "REQ-30.5"),
		residue([]string{"docs/otra.md"}, scopeAll, defaultPatternExample, "entrada obsoleta", "REQ-1.1"),
	}
	fails := checkResidues(files, res)
	if !anyContains(fails, "REQ-30.5 caducó") || !anyContains(fails, "entrada 2") || !anyContains(fails, "caducada") {
		t.Fatalf("faltan la caducidad del token y de la entrada: %v", fails)
	}
	if anyContains(fails, "REQ-30.4 caducó") {
		t.Fatalf("el token vivo no caduca: %v", fails)
	}
}

func TestResidueExpiryCountsTheArchive(t *testing.T) {
	// Tras archivar, la carpeta del cambio se mueve a archive/: su entrada sigue viva.
	files := map[string]string{"openspec/changes/archive/2026-11-30-migrar-ids-rf/proposal.md": "cita REQ-1.1\n"}
	res := []residueEntry{
		residue([]string{"openspec/changes/migrar-ids-rf/**", "openspec/changes/archive/*-migrar-ids-rf/**"}, scopeAll, "legacy-citation", "documentos de este cambio"),
	}
	if fails := checkResidues(files, res); len(fails) != 0 {
		t.Fatalf("la caducidad debe contar el archivo: %v", fails)
	}
	if fails := checkResidues(map[string]string{}, res); !anyContains(fails, "caducada") {
		t.Fatalf("sin ninguna aparición la entrada caduca: %v", fails)
	}
}

func TestResidueInputFilesNeitherKeepEntriesAliveNorNeedResidues(t *testing.T) {
	files := map[string]string{
		residuesPath:   "tokens: [REQ-30.4]\n",
		contextsPath:   "token: REQ-30.4\n",
		idMapPath:      "old: [REQ-30.4]\n",
		"docs/guia.md": "sin citas\n",
	}
	res := []residueEntry{residue([]string{"docs/guia.md"}, scopeAll, defaultPatternExample, "ejemplo", "REQ-30.4")}
	fails := checkResidues(files, res)
	if len(fails) == 0 {
		t.Fatal("citar el token solo en la lista de residuos, el mapa o los contextos no mantiene viva la entrada")
	}
}

func TestResidueRejectsProductionLiteralsAndEmbeddedAssets(t *testing.T) {
	prod := residue([]string{"internal/techcontext/plan.go"}, scopeString, "legacy-citation", "cita", "REQ-24.62")
	if err := validateResidue(prod); err == nil || !strings.Contains(err.Error(), "producción") {
		t.Fatalf("una entrada sobre un literal de Go de producción es inválida: %v", err)
	}
	all := residue([]string{"internal/foo/*.go"}, scopeAll, "legacy-citation", "cita", "REQ-1.1")
	if err := validateResidue(all); err == nil {
		t.Fatal("`all` sobre un .go de producción es inválido")
	}
	if err := validateResidue(residue([]string{"internal/foo/*_test.go"}, scopeString, "legacy-citation", "cita", "REQ-1.1")); err != nil {
		t.Fatalf("un fichero de prueba sí puede declararse: %v", err)
	}
	asset := residue([]string{"internal/assets/skills/_shared/x.md"}, scopeAll, "legacy-citation", "cita", "REQ-1.1")
	if err := validateResidue(asset); err == nil || !strings.Contains(err.Error(), "asset") {
		t.Fatalf("un asset embebido de producción no admite una cita heredada: %v", err)
	}
	example := residue([]string{"internal/assets/skills/_shared/sdd-iso-convention.md"}, scopeAll, defaultPatternExample, "ejemplo del patrón por defecto", "REQ-1.1")
	if err := validateResidue(example); err != nil {
		t.Fatalf("el ejemplo explícito del patrón por defecto sí se admite en un asset: %v", err)
	}

	// Aunque la entrada pase la estructura, la prueba dinámica veta el literal de producción.
	files := map[string]string{"internal/foo/foo.go": "package foo\n\nvar msg = \"REQ-1.1\"\n"}
	res := []residueEntry{residue([]string{"internal/foo/foo.go"}, scopeComment, "legacy-citation", "cita", "REQ-1.1")}
	if fails := checkResidues(files, res); !anyContains(fails, "internal/foo/foo.go:3 REQ-1.1") {
		t.Fatalf("un literal de producción sin residuo válido debe fallar: %v", fails)
	}
	res = []residueEntry{{Paths: []string{"internal/foo/foo.go"}, Scope: scopeAll, Category: "legacy-citation", Reason: "x", Tokens: []string{"REQ-1.1"}}}
	if fails := checkResidues(files, res); !anyContains(fails, "nunca es residuo") {
		t.Fatalf("una entrada que cubre un literal de producción se veta: %v", fails)
	}
	files = map[string]string{"internal/assets/skills/a.md": "ver REQ-1.1\n"}
	res = []residueEntry{{Paths: []string{"internal/assets/skills/*.md"}, Scope: scopeAll, Category: "legacy-citation", Reason: "x", Tokens: []string{"REQ-1.1"}}}
	if fails := checkResidues(files, res); !anyContains(fails, "asset embebido") {
		t.Fatalf("un asset embebido de producción se veta: %v", fails)
	}
}

func TestResidueStructuralValidation(t *testing.T) {
	cases := map[string]struct {
		e    residueEntry
		want string
	}{
		"categoría desconocida": {residue([]string{"docs/a.md"}, scopeAll, "inventada", "m", "REQ-1.1"), "categoría"},
		"ámbito desconocido":    {residue([]string{"docs/a.md"}, "todo", "legacy-citation", "m", "REQ-1.1"), "ámbito"},
		"motivo vacío":          {residue([]string{"docs/a.md"}, scopeAll, "legacy-citation", " ", "REQ-1.1"), "motivo"},
		"tokens obligatorio":    {residue([]string{"docs/a.md"}, scopeAll, "legacy-citation", "m"), "tokens"},
		"tokens en doc ejemplo": {residue([]string{"docs/a.md"}, scopeCode, defaultPatternExample, "m"), "tokens"},
		"sin rutas":             {residue(nil, scopeAll, "legacy-citation", "m", "REQ-1.1"), "rutas"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateResidue(c.e); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("quería un error con %q: %v", c.want, err)
			}
		})
	}
	ok := []residueEntry{
		residue([]string{"internal/x/*_test.go"}, scopeString, "default-pattern-fixture", "m"),
		residue([]string{"**/testdata/**"}, scopeAll, "default-pattern-fixture", "m"),
		residue([]string{"openspec/specs/**/spec.md"}, scopeCode, defaultPatternExample, "m"),
		residue([]string{"docs/archive/**", "docs/audits/**"}, scopeAll, "dated-history", "m"),
		residue([]string{"openspec/changes/migrar-ids-rf/**", "openspec/changes/archive/*-migrar-ids-rf/**"}, scopeAll, "legacy-citation", "m"),
	}
	for _, e := range ok {
		if err := validateResidue(e); err != nil {
			t.Errorf("%v debe aceptarse sin tokens: %v", e.Paths, err)
		}
	}
}

func TestResidueParseFilesStrictly(t *testing.T) {
	good := "schema: axiom.id-residues/v1\nentries:\n  - paths: [\"docs/a.md\"]\n    scope: all\n    category: default-pattern-example\n    tokens: [REQ-30.4]\n    reason: \"ejemplo\"\n"
	if es, err := parseResidues([]byte(good)); err != nil || len(es) != 1 {
		t.Fatalf("lista válida: %v %v", es, err)
	}
	for name, bad := range map[string]string{
		"esquema":     strings.Replace(good, "axiom.id-residues/v1", "otro/v9", 1),
		"campo ajeno": good + "    extra: 1\n",
		"categoría":   strings.Replace(good, "default-pattern-example", "inventada", 1),
	} {
		if _, err := parseResidues([]byte(bad)); err == nil {
			t.Errorf("%s: debe fallar", name)
		}
	}
	ctx := "schema: axiom.id-contexts/v1\nentries:\n  - path: internal/hub/init.go\n    token: REQ-2.2\n    capability: multi-project-hub\n    reason: \"el comentario la nombra\"\n"
	if es, err := parseContexts([]byte(ctx)); err != nil || len(es) != 1 {
		t.Fatalf("contextos válidos: %v %v", es, err)
	}
	if _, err := parseContexts([]byte(strings.Replace(ctx, "multi-project-hub", "", 1))); err == nil {
		t.Error("un contexto sin capacidad debe fallar")
	}
}

func TestResidueScopesFollowGoScannerAndMarkdownCode(t *testing.T) {
	occs, err := occurrencesOf("a.go", "package a\n\n// REQ-1.1 comentario\nvar s = \"REQ-2.2\"\nvar r = `REQ-3.3`\n")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, o := range occs {
		got[o.Token] = fmt.Sprintf("%s@%d", o.Scope, o.Line)
	}
	if got["REQ-1.1"] != "go-comment@3" || got["REQ-2.2"] != "go-string@4" || got["REQ-3.3"] != "go-string@5" {
		t.Fatalf("ámbitos Go: %v", got)
	}
	occs, err = occurrencesOf("a.md", "texto REQ-1.1 y `REQ-2.2`\n\n```go\nREQ-3.3\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	got = map[string]string{}
	for _, o := range occs {
		got[o.Token] = fmt.Sprintf("%s@%d", o.Scope, o.Line)
	}
	if got["REQ-1.1"] != "text@1" || got["REQ-2.2"] != "md-code@1" || got["REQ-3.3"] != "md-code@4" {
		t.Fatalf("ámbitos Markdown: %v", got)
	}
}

func TestRepoRootResolvesWithGitToplevel(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("la raíz del repositorio debe resolverse: %v", err)
	}
	if !filepath.IsAbs(root) {
		t.Errorf("la raíz debe ser absoluta: %s", root)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("la raíz no tiene go.mod: %v", err)
	}
	// Un subdirectorio de un repositorio no es su raíz: se rechaza.
	if _, err := repoRootFrom("."); err == nil {
		t.Error("un subdirectorio del repositorio no es la raíz")
	}
}

func TestRepoRootFailsOutsideARepositoryAndWithoutGit(t *testing.T) {
	if _, err := repoRootFrom(t.TempDir()); err == nil {
		t.Fatal("fuera de un repositorio debe fallar, no omitirse")
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := repoRoot(); err == nil {
		t.Fatal("sin git debe fallar, no omitirse")
	}
	if _, err := listVersioned("."); err == nil {
		t.Fatal("sin git la lista de ficheros debe fallar")
	}
}

func TestRepoRootRejectsDotDotAndAbsolutePathsFromTheList(t *testing.T) {
	for _, bad := range []string{"../fuera.txt", "a/../../fuera.txt", "/etc/passwd", `C:\Windows\x`, `..\x`, ""} {
		if err := safeRel(bad); err == nil {
			t.Errorf("safeRel(%q) debe rechazarse", bad)
		}
	}
	for _, good := range []string{"a/b.go", "docs/x..y.md", ".claude/x"} {
		if err := safeRel(good); err != nil {
			t.Errorf("safeRel(%q): %v", good, err)
		}
	}
	// Una ruta con `..` de la lista no se escribe: migrate la rechaza antes de leerla.
	root := t.TempDir()
	res := migrate(migrateIn{Root: root, Files: []string{"../fuera.md"}, Entries: []mapEntry{{Capability: "autoskills", Old: []string{"REQ-1.1"}, New: "RF-ASK-01"}}})
	if len(res.Failures) != 1 || len(res.Files) != 0 {
		t.Fatalf("una ruta con `..` debe rechazarse y no escribirse: %+v", res)
	}
}

func TestRepoRootListsVersionedFilesWithGit(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	writeFile(t, root, "a/b.txt", "x\n")
	writeFile(t, root, "c.txt", "y\n")
	run("add", "--", "a/b.txt", "c.txt")
	got, err := repoRootFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	files, err := listVersioned(got)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if strings.Join(files, ",") != "a/b.txt,c.txt" {
		t.Fatalf("ls-files = %v", files)
	}
}

func anyContains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
