package idmigration_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IGutierrezZ/axiom/v3/internal/legacyid"
	"github.com/IGutierrezZ/axiom/v3/internal/livingdoc"
)

// runEnv es el entorno de una ejecución de la herramienta: la raíz, la lista de ficheros
// (git ls-files en el repositorio real, un recorrido en el árbol miniatura) y el registro.
type runEnv struct {
	Root string
	List func() ([]string, error)
	Logf func(string, ...any)
}

// runMode ejecuta uno de los tres modos de la herramienta (D1):
//
//	map:   genera openspec/id-map.yaml (falla si ya existe)
//	plan:  calcula toda la reescritura en memoria, informa y no escribe
//	write: las mismas comprobaciones que plan y, si todo resuelve, escribe
func runMode(mode string, env runEnv) error {
	switch mode {
	case "map":
		return runMap(env)
	case "plan":
		return runRewrite(false, env)
	case "write":
		return runRewrite(true, env)
	}
	return fmt.Errorf("AXIOM_IDMIGRATION=%q no es válido: usa map, plan o write", mode)
}

func mapFilePath(root string) string { return filepath.Join(root, "openspec", legacyid.FileName) }

func runMap(env runEnv) error {
	target := mapFilePath(env.Root)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s ya existe: bórralo de forma explícita para regenerarlo", target)
	}
	entries, err := generateMap(env.Root, domains)
	if err != nil {
		return err
	}
	text, err := emitMap(entries)
	if err != nil {
		return err
	}
	if _, err := legacyid.Load(fstest.MapFS{legacyid.FileName: {Data: []byte(text)}}, legacyid.FileName, rfPattern); err != nil {
		return fmt.Errorf("el mapa generado no supera legacyid.Load: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, []byte(text), 0o644); err != nil {
		return err
	}
	env.Logf("mapa generado: %d entradas en %s", len(entries), target)
	return nil
}

// loadMetaFrom lee las entradas curadas (contextos y residuos) de root; si no existen, vacías.
func loadMetaFrom(root string) ([]contextEntry, []residueEntry, error) {
	var ctxs []contextEntry
	var res []residueEntry
	if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(contextsPath))); err == nil {
		if ctxs, err = parseContexts(data); err != nil {
			return nil, nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, err
	}
	if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(residuesPath))); err == nil {
		if res, err = parseResidues(data); err != nil {
			return nil, nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, err
	}
	return ctxs, res, nil
}

func loadMeta(t *testing.T, root string) ([]contextEntry, []residueEntry) {
	t.Helper()
	ctxs, res, err := loadMetaFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	return ctxs, res
}

func runRewrite(write bool, env runEnv) error {
	_, statErr := os.Stat(mapFilePath(env.Root))
	var entries []mapEntry
	var err error
	switch {
	case statErr == nil:
		if entries, err = readMapFile(env.Root); err != nil {
			return err
		}
	case write:
		return fmt.Errorf("falta %s: ejecuta antes AXIOM_IDMIGRATION=map", mapFilePath(env.Root))
	default:
		env.Logf("no hay %s: plan usa el mapa que se generaría ahora, en memoria", legacyid.FileName)
		if entries, err = generateMap(env.Root, domains); err != nil {
			return err
		}
	}
	ctxs, res, err := loadMetaFrom(env.Root)
	if err != nil {
		return err
	}
	files, err := env.List()
	if err != nil {
		return err
	}
	r := migrate(migrateIn{Root: env.Root, Files: files, Entries: entries, Ctxs: ctxs, Res: res, DeclarePattern: true})
	report(env, r)
	if len(r.Failures) > 0 {
		return failuresError(r.Failures)
	}
	if !write {
		return nil
	}

	paths := make([]string, 0, len(r.Files))
	for p := range r.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		abs := filepath.Join(env.Root, filepath.FromSlash(p))
		perm := os.FileMode(0o644)
		if fi, err := os.Stat(abs); err == nil {
			perm = fi.Mode().Perm()
		}
		if err := os.WriteFile(abs, []byte(r.Files[p]), perm); err != nil {
			return fmt.Errorf("no se pudo escribir %s: %w", p, err)
		}
	}
	if _, err := os.Stat(filepath.Join(env.Root, "openspec", "specs")); err == nil {
		changed, err := syncIndex(env.Root)
		if err != nil {
			return fmt.Errorf("no se pudo regenerar INDEX.md: %w", err)
		}
		env.Logf("INDEX.md regenerado: %v", changed)
	}

	// Segunda pasada en memoria sobre lo escrito: debe ser un no-op (idempotencia).
	again := migrate(migrateIn{Root: env.Root, Files: files, Entries: entries, Ctxs: ctxs, Res: res, DeclarePattern: true})
	if len(again.Failures) > 0 || len(again.Files) > 0 {
		return fmt.Errorf("la segunda pasada no es idempotente: %d fallos, %d ficheros cambian (%v)", len(again.Failures), len(again.Files), firstN(keys(again.Files), 5))
	}
	return nil
}

func firstN(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func failuresError(f []string) error {
	shown := f
	if len(shown) > 50 {
		shown = shown[:50]
	}
	return fmt.Errorf("la migración no puede continuar, nada se escribe: %d fallos\n%s", len(f), strings.Join(shown, "\n"))
}

var tokenInEntry = regexp.MustCompile(`REQ-\d+\.\d+[a-z]?$`)

// report publica el informe de plan/write: por fichero, el resumen y las propuestas de
// residuos para lo que se conserva sin cobertura.
func report(env runEnv, r *result) {
	paths := make([]string, 0, len(r.Files))
	for p := range r.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		env.Logf("reescribe %s", p)
	}
	ambiguous := 0
	for _, f := range r.Failures {
		if strings.Contains(f, "ambiguo sin resolver") {
			ambiguous++
		}
		env.Logf("FALLO %s", f)
	}
	tokens := map[string]bool{}
	byPath := map[string][]string{}
	for _, u := range r.Unmapped {
		sp := strings.SplitN(u, " ", 2)
		loc, rest := sp[0], ""
		if len(sp) > 1 {
			rest = sp[1]
		}
		path := loc[:strings.LastIndex(loc, ":")]
		tok := tokenInEntry.FindString(rest)
		if tok != "" {
			tokens[tok] = true
			byPath[path] = append(byPath[path], tok)
		}
		env.Logf("sin equivalente %s", u)
	}
	var props []string
	for p := range byPath {
		props = append(props, p)
	}
	sort.Strings(props)
	for _, p := range props {
		toks := dedupe(byPath[p])
		env.Logf("candidato a residuo: paths: [%q] tokens: [%s]", p, strings.Join(toks, ", "))
	}
	env.Logf("RESUMEN ficheros analizados=%d reescritos=%d sustituciones=%d conservados(residuo=%d automático=%d) ambiguos sin resolver=%d fallos=%d tokens sin equivalente=%d apariciones sin equivalente=%d ficheros candidatos a residuo=%d",
		r.Scanned, len(r.Files), r.Subs, r.KeptRes, r.KeptAuto, ambiguous, len(r.Failures), len(tokens), len(r.Unmapped), len(props))
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// ---- axiom.yaml e INDEX.md ----

// ensureSDDBlock inserta tras `workflow:` el bloque `sdd:` con el patrón de Axiom (D10). Si ya
// declara ese mismo patrón no toca nada; con otro valor, falla.
func ensureSDDBlock(content string) (string, error) {
	ls := splitLines(content)
	nl := "\n"
	if strings.Contains(content, "\r\n") {
		nl = "\r\n"
	}
	sddAt, wfAt := -1, -1
	for i, l := range ls {
		switch {
		case strings.HasPrefix(l.Text, "sdd:"):
			sddAt = i
		case strings.HasPrefix(l.Text, "workflow:"):
			wfAt = i
		}
	}
	if sddAt >= 0 {
		for j := sddAt + 1; j < len(ls); j++ {
			t := ls[j].Text
			if t != "" && t[0] != ' ' && t[0] != '\t' && t[0] != '#' {
				break
			}
			trimmed := strings.TrimSpace(t)
			if !strings.HasPrefix(trimmed, "id_pattern:") {
				continue
			}
			v := strings.TrimSpace(strings.TrimPrefix(trimmed, "id_pattern:"))
			if i := strings.Index(v, " #"); i >= 0 {
				v = strings.TrimSpace(v[:i])
			}
			if v = strings.Trim(v, `"'`); v == rfTemplate {
				return content, nil
			}
			return "", fmt.Errorf("`sdd.id_pattern` ya vale %q y no %q", v, rfTemplate)
		}
		return "", errors.New("el bloque `sdd:` existe sin `id_pattern`: decláralo a mano")
	}
	if wfAt < 0 {
		return "", errors.New("no hay bloque `workflow:` tras el que insertar `sdd:`")
	}
	last := wfAt
	for j := wfAt + 1; j < len(ls); j++ {
		t := ls[j].Text
		if t == "" || t[0] == '#' {
			continue
		}
		if t[0] == ' ' || t[0] == '\t' {
			last = j
			continue
		}
		break
	}
	block := "# Patrón de identificadores de requisito propio de Axiom (C-03-e)." + nl +
		"sdd:" + nl + "  id_pattern: \"" + rfTemplate + "\"" + nl
	off := ls[last].End
	switch {
	case strings.HasPrefix(content[off:], "\r\n"):
		off += 2
	case strings.HasPrefix(content[off:], "\n"):
		off++
	default:
		return content + nl + nl + block, nil
	}
	return content[:off] + nl + block + content[off:], nil
}

var syncDateRe = regexp.MustCompile(`(?m)^> \*\*Última Sincronización:\*\*.*$`)

// syncIndex regenera openspec/INDEX.md con livingdoc.Sync y devuelve si cambió algo más que la
// línea de fecha; si solo cambió la fecha, restaura el contenido anterior.
func syncIndex(root string) (bool, error) {
	target := filepath.Join(root, "openspec", "INDEX.md")
	old, _ := os.ReadFile(target)
	if _, err := livingdoc.NewService(root, nil, nil).Sync(context.Background()); err != nil {
		return false, err
	}
	now, err := os.ReadFile(target)
	if err != nil {
		return false, err
	}
	if len(old) > 0 && syncDateRe.ReplaceAllString(string(old), "") == syncDateRe.ReplaceAllString(string(now), "") {
		return false, os.WriteFile(target, old, 0o644)
	}
	return true, nil
}

// ---- auxiliares de prueba ----

func walkList(root string) func() ([]string, error) {
	return func() ([]string, error) {
		var out []string
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
			return nil
		})
		sort.Strings(out)
		return out, err
	}
}

func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files, err := walkList(root)()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, p := range files {
		out[p] = readFile(t, root, p)
	}
	return out
}

// ---- pruebas ----

// TestMigrateRepository ejecuta la herramienta sobre el repositorio real. Sin la variable de
// entorno se omite; con ella:
//
//	AXIOM_IDMIGRATION=map|plan|write go test ./internal/idmigration -run TestMigrateRepository -count=1 -v
func TestMigrateRepository(t *testing.T) {
	mode := os.Getenv("AXIOM_IDMIGRATION")
	if mode == "" {
		t.Skip("AXIOM_IDMIGRATION=map|plan|write ejecuta la herramienta sobre el repositorio real")
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := runMode(mode, runEnv{Root: root, List: func() ([]string, error) { return listVersioned(root) }, Logf: t.Logf}); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateModeRejectsUnknownValue(t *testing.T) {
	if err := runMode("destruir", runEnv{Root: t.TempDir(), List: func() ([]string, error) { return nil, nil }, Logf: t.Logf}); err == nil {
		t.Fatal("un modo desconocido debe fallar")
	}
}

func TestMigrateMiniatureMapPlanWriteEndToEnd(t *testing.T) {
	root := miniRoot(t)
	var logs []string
	env := runEnv{Root: root, List: walkList(root), Logf: func(f string, a ...any) { logs = append(logs, fmt.Sprintf(f, a...)) }}
	before := snapshot(t, root)

	// plan sin mapa: usa el mapa que se generaría y no escribe nada.
	if err := runMode("plan", env); err != nil {
		t.Fatalf("plan sin mapa: %v", err)
	}
	if !equalMaps(before, snapshot(t, root)) {
		t.Fatal("plan escribió algo")
	}
	if !anyContains(logs, "no hay id-map.yaml") || !anyContains(logs, "RESUMEN") || !anyContains(logs, "sin equivalente docs/guide.md:4 REQ-24.99") {
		t.Fatalf("el informe de plan es incompleto:\n%s", strings.Join(logs, "\n"))
	}

	// write exige el mapa generado antes.
	if err := runMode("write", env); err == nil || !strings.Contains(err.Error(), "AXIOM_IDMIGRATION=map") {
		t.Fatalf("write sin mapa debe pedir map: %v", err)
	}

	if err := runMode("map", env); err != nil {
		t.Fatalf("map: %v", err)
	}
	mapped := snapshot(t, root)
	if len(mapped) != len(before)+1 || mapped["openspec/id-map.yaml"] == "" {
		t.Fatalf("map solo debe crear id-map.yaml: %v", keys(mapped))
	}

	logs = nil
	if err := runMode("plan", env); err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !equalMaps(mapped, snapshot(t, root)) {
		t.Fatal("plan escribió algo")
	}

	if err := runMode("write", env); err != nil {
		t.Fatalf("write: %v", err)
	}
	after := snapshot(t, root)

	expect := map[string][]string{
		"openspec/specs/autoskills/spec.md": {
			"> Cubre RF-ASK-01 a RF-ASK-02 de esta capacidad; no toca RF-ASK-03 ni REQ-11.1.",
			"### Requirement: Skill Detection (RF-ASK-01)",
			"Ejemplo del patrón por defecto: `REQ-1.1`.", // código en línea conservado
			"Continúa en RF-ASK-01/RF-ASK-02 y se amplía en RF-ASK-01–RF-ASK-02.",
			"### Requirement: Falso en valla (REQ-9.9)\nREQ-9.9", // valla conservada
		},
		"openspec/specs/workspace-topology/spec.md": {
			"### Requirement: Topology Detection (RF-TOP-01)", "Ver RF-TOP-02 y RF-TOP-03.",
			"### Requirement: Legacy Topology (RF-TOP-03) [SUPERSEDED / DEPRECADO]",
			"### Requirement: Untitled Behaviour (RF-TOP-04)",
		},
		"openspec/specs/persona-behavior-contract/spec.md": {"### Requirement: Tone Contract (RF-PER-01)"},
		"openspec/specs/sdd-lane-kickoff/spec.md":          {"Distinto de RF-KOF-01: RF-KOF-02 es un token propio."},
		"odd/tasks/feature.md": {
			"Cubre: RF-HUB-01", "### Requirement: Hub Registration (RF-HUB-01)", "Ejemplo `REQ-2.1`.", "REQ-2.1 anidado",
			"### Requirement: Retirada (REQ-7.7)\n\nCita REQ-2.1 de una capacidad retirada.",
		},
		"internal/foo/foo.go":      {"// Ver RF-TOP-03 de la topología.", `const msg = "falta RF-KOF-02"`},
		"internal/foo/foo_test.go": {"// REQ-1.10 describe el fixture; RF-TOP-03 es una cita.", `var _ = "RF-KOF-02"`, `"## REQ-1.10 en un fixture"`},
		"docs/guide.md":            {"REQ-1.1 de autoskills", "REQ-24.99", "REQ-30.4"},
	}
	// docs/guide.md: REQ-1.1 con `autoskills` en la línea → autoskills
	expect["docs/guide.md"] = []string{"Detección de skills: RF-ASK-01 de autoskills.", "REQ-24.99 se conserva", "Ejemplo para otros proyectos: REQ-30.4."}
	for p, wants := range expect {
		for _, w := range wants {
			if !strings.Contains(after[p], w) {
				t.Errorf("%s no contiene %q:\n%s", p, w, after[p])
			}
		}
	}
	if after["assets/blob.bin"] != before["assets/blob.bin"] {
		t.Error("un binario no se toca")
	}
	// axiom.yaml: bloque sdd: insertado y tokens sin equivalente (REQ-25.24) conservados;
	// REQ-27.1 no está en el mapa miniatura y también se conserva.
	if !strings.Contains(after["axiom.yaml"], "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"") ||
		!strings.Contains(after["axiom.yaml"], "(REQ-25.24)") || !strings.Contains(after["axiom.yaml"], "(REQ-27.1)") {
		t.Errorf("axiom.yaml:\n%s", after["axiom.yaml"])
	}
	if idx := after["openspec/INDEX.md"]; !strings.Contains(idx, "RF-ASK-01") {
		t.Errorf("INDEX.md regenerado debe usar los IDs nuevos:\n%s", idx)
	}

	// Una segunda ejecución de write no cambia nada y no escribe INDEX solo por la fecha.
	if err := runMode("write", env); err != nil {
		t.Fatalf("segundo write: %v", err)
	}
	if again := snapshot(t, root); !equalMaps(after, again) {
		t.Fatal("la segunda ejecución de write cambió ficheros")
	}
}

func TestMigrateAxiomYAMLDeclaresPatternOnce(t *testing.T) {
	base := "workflow:\n  lane: \"ask\"\n\nroles:\n  a: 1\n"
	got, err := ensureSDDBlock(base)
	if err != nil {
		t.Fatal(err)
	}
	want := "workflow:\n  lane: \"ask\"\n\n# Patrón de identificadores de requisito propio de Axiom (C-03-e).\nsdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n\nroles:\n  a: 1\n"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if again, err := ensureSDDBlock(got); err != nil || again != got {
		t.Fatalf("segunda inserción: %q %v", again, err)
	}
	if crlf, err := ensureSDDBlock(strings.ReplaceAll(base, "\n", "\r\n")); err != nil || !strings.Contains(crlf, "sdd:\r\n  id_pattern") || strings.Contains(strings.ReplaceAll(crlf, "\r\n", ""), "\n") {
		t.Fatalf("CRLF: %q %v", crlf, err)
	}
	if _, err := ensureSDDBlock("workflow:\n  lane: ask\nsdd:\n  id_pattern: \"REQ-<n>.<m>\"\n"); err == nil {
		t.Fatal("un `sdd.id_pattern` distinto debe fallar")
	}
	if _, err := ensureSDDBlock("roles:\n  a: 1\n"); err == nil {
		t.Fatal("sin `workflow:` no hay dónde insertar")
	}
	if got, err := ensureSDDBlock("workflow:\n  lane: ask"); err != nil || !strings.HasSuffix(got, "id_pattern: \"RF-<DOM>-<NN>\"\n") {
		t.Fatalf("fichero sin salto final: %q %v", got, err)
	}
}

func TestMigrateFailsWithoutWritingWhenAxiomYAMLHasAnotherPattern(t *testing.T) {
	root := miniRoot(t)
	env := runEnv{Root: root, List: walkList(root), Logf: t.Logf}
	if err := runMode("map", env); err != nil {
		t.Fatal(err)
	}
	yaml := readFile(t, root, "axiom.yaml")
	writeFile(t, root, "axiom.yaml", yaml+"\nsdd:\n  id_pattern: \"REQ-<n>.<m>\"\n")
	before := snapshot(t, root)
	if err := runMode("write", env); err == nil || !strings.Contains(err.Error(), "axiom.yaml") {
		t.Fatalf("write debe fallar por el patrón distinto: %v", err)
	}
	if !equalMaps(before, snapshot(t, root)) {
		t.Fatal("write escribió algo pese al fallo")
	}
}
