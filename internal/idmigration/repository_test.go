package idmigration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Pruebas permanentes de la migración de identificadores (C-03-e, commit e). A diferencia de
// TestMigrateRepository, siempre están activas: protegen el mapa, los residuos y el patrón RF
// una vez que la herramienta ya no se ejecuta.

// legacyPrefix evita escribir un token heredado literal en este fichero de pruebas.
const legacyPrefix = "REQ-"

// idMapEntries es el número de entradas del mapa en el momento de la migración. Solo cambia con
// una versión nueva del esquema del mapa.
const idMapEntries = 513

// idMapSHA256 es la huella del mapa tal como lo generó la migración (con saltos de línea LF).
const idMapSHA256 = "197be163ae5ef98e35f8c9de87cb36c660fcc9798cbba73d9ac0b2eeb98620c1"

// withdrawnCubre lista las citas `Cubre:` de documentos vivos que apuntan a un requisito
// retirado y siguen siendo válidas a propósito: ID -> motivo. Está vacía mientras ningún
// documento vivo cite un requisito retirado; añadir una entrada exige un motivo explícito.
var withdrawnCubre = map[string]string{}

// fenceMarker devuelve el delimitador de una línea de valla de código, o "".
func fenceMarker(line string) string {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return ""
	}
	for _, m := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, m) {
			return m
		}
	}
	return ""
}

// requirementTitleIDs devuelve los IDs RF de los títulos de requisito de un spec.md, fuera de
// vallas.
func requirementTitleIDs(content string) []string {
	var ids []string
	open := ""
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if m := fenceMarker(line); m != "" {
			switch {
			case open == "":
				open = m
			case open == m:
				open = ""
			}
			continue
		}
		if open != "" || !strings.HasPrefix(line, "### Requirement:") {
			continue
		}
		if id := rfPattern.InTitle(strings.TrimSpace(strings.TrimPrefix(line, "### Requirement:"))); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// livingSpecIDs lee openspec/specs/*/spec.md: capacidad -> conjunto de IDs.
func livingSpecIDs(t *testing.T, root string) map[string]map[string]bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "openspec", "specs", "*", "spec.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no hay specs vivas en %s: %v", root, err)
	}
	out := map[string]map[string]bool{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		set := map[string]bool{}
		for _, id := range requirementTitleIDs(string(data)) {
			set[id] = true
		}
		out[filepath.Base(filepath.Dir(f))] = set
	}
	return out
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err) // sin git o fuera de un repositorio la prueba falla, nunca se omite
	}
	return root
}

// TestIDMapMatchesLivingSpecs comprueba solo mapa -> spec viva: cada entrada existe en la spec
// de su capacidad con su ID nuevo. No comprueba spec -> mapa: los requisitos posteriores a la
// migración (y los que asigna el archivado de este cambio) no están ni deben estar en el mapa.
func TestIDMapMatchesLivingSpecs(t *testing.T) {
	root := mustRepoRoot(t)
	entries, err := readMapFile(root) // incluye legacyid.Load con el patrón RF
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != idMapEntries {
		t.Errorf("el mapa tiene %d entradas, esperadas %d: una entrada perdida o añadida a mano", len(entries), idMapEntries)
	}
	live := livingSpecIDs(t, root)
	inMap := map[string]bool{}
	for _, e := range entries {
		inMap[e.New] = true
		set, ok := live[e.Capability]
		if !ok {
			t.Errorf("%s -> %s: la capacidad %q no tiene spec viva", strings.Join(e.Old, ","), e.New, e.Capability)
			continue
		}
		if !set[e.New] {
			t.Errorf("%s -> %s: la spec viva de %q no tiene un requisito con ese ID", strings.Join(e.Old, ","), e.New, e.Capability)
		}
	}
	// Los IDs que asigna el archivado de este cambio nunca entran en el mapa.
	for _, id := range []string{"RF-TRC-16", "RF-TRC-17", "RF-RID-16"} {
		if inMap[id] {
			t.Errorf("%s no debe estar en el mapa: lo asigna el archivado de migrar-ids-rf", id)
		}
	}
	// Los requisitos que modifica la delta de este cambio existen en la spec viva.
	for _, id := range []string{"RF-TRC-02", "RF-TRC-04"} {
		if !live["requirement-trace"][id] {
			t.Errorf("%s no existe en la spec viva requirement-trace", id)
		}
	}
}

// TestIDMapFingerprint fija la huella del mapa (inmutabilidad, hueco 2).
func TestIDMapFingerprint(t *testing.T) {
	root := mustRepoRoot(t)
	data, err := os.ReadFile(mapFilePath(root))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(strings.ReplaceAll(string(data), "\r\n", "\n")))
	if got := hex.EncodeToString(sum[:]); got != idMapSHA256 {
		t.Errorf("openspec/id-map.yaml cambió (sha256 %s, esperado %s). El mapa es inmutable una vez "+
			"publicado: es el contrato con el que `axiom trace` traduce los IDs heredados. Un cambio "+
			"legítimo exige actualizar idMapSHA256 de forma deliberada y revisada, en el mismo commit "+
			"que explica por qué cambia el mapa.", got, idMapSHA256)
	}
}

// TestRepositoryLegacyIDResidues recorre `git ls-files` y exige que toda cita heredada fuera de
// la historia archivada esté declarada en residues.yaml, que ninguna entrada ni token haya
// caducado (el archivo cuenta) y que ningún literal de Go de producción ni asset embebido sea
// residuo. Siempre activa: sin git, o fuera de un repositorio, falla.
func TestRepositoryLegacyIDResidues(t *testing.T) {
	root := mustRepoRoot(t)
	files, err := listVersioned(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("git ls-files no devolvió ningún fichero")
	}
	content, err := readFiles(root, files)
	if err != nil {
		t.Fatal(err)
	}
	_, res, err := loadMetaFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("residues.yaml no declara ninguna entrada")
	}
	for i, e := range res {
		if err := validateResidue(e); err != nil {
			t.Errorf("residues.yaml entrada %d: %v", i+1, err)
		}
	}
	for _, f := range checkResidues(content, res) {
		t.Error(f)
	}
}

// cubreProblems valida las líneas `Cubre:` de un conjunto de documentos: cada ID es válido bajo
// RF-<DOM>-<NN> y existe en la spec viva, salvo los retirados declarados con motivo.
func cubreProblems(docs map[string]string, live map[string]bool, withdrawn map[string]string) []string {
	var out []string
	paths := make([]string, 0, len(docs))
	for p := range docs {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		for _, line := range isoevidence.ParseCubre(p, docs[p]) {
			for _, id := range line.IDs {
				switch {
				case !rfPattern.Valid(id):
					out = append(out, fmt.Sprintf("%s:%d: %q no es un ID RF-<DOM>-<NN>", p, line.Line, id))
				case live[id]:
				case withdrawn[id] != "":
				default:
					out = append(out, fmt.Sprintf("%s:%d: %s no existe en la spec viva ni está declarado como retirado", p, line.Line, id))
				}
			}
		}
	}
	return out
}

// TestCubreLinesUseExistingRFIDs revisa los documentos vivos (odd/tasks/*.md y las specs); no
// inspecciona openspec/changes/archive/**.
func TestCubreLinesUseExistingRFIDs(t *testing.T) {
	root := mustRepoRoot(t)
	live := map[string]bool{}
	for _, set := range livingSpecIDs(t, root) {
		for id := range set {
			live[id] = true
		}
	}
	for id, why := range withdrawnCubre {
		if strings.TrimSpace(why) == "" {
			t.Errorf("withdrawnCubre[%s] no tiene motivo", id)
		}
	}
	var patterns = []string{
		filepath.Join(root, "odd", "tasks", "*.md"),
		filepath.Join(root, "openspec", "specs", "*", "spec.md"),
	}
	docs := map[string]string{}
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil {
				t.Fatal(err)
			}
			rel, _ := filepath.Rel(root, m)
			docs[filepath.ToSlash(rel)] = string(data)
		}
	}
	if len(docs) == 0 {
		t.Fatal("no hay documentos vivos que revisar")
	}
	for _, p := range cubreProblems(docs, live, withdrawnCubre) {
		t.Error(p)
	}
}

func TestCubreProblemsClassifiesCitations(t *testing.T) {
	live := map[string]bool{"RF-TRC-02": true}
	withdrawn := map[string]string{"RF-OLD-01": "requisito retirado, se cita como historia"}
	docs := map[string]string{
		"a.md": "- Cubre: RF-TRC-02, RF-OLD-01\n" +
			"  Cubre: " + legacyPrefix + "1.1\n" +
			"Cubre: RF-TRC-99\n" +
			"```\nCubre: RF-NOP-01\n```\n",
	}
	got := strings.Join(cubreProblems(docs, live, withdrawn), "\n")
	for _, want := range []string{
		`a.md:2: "` + legacyPrefix + `1.1" no es un ID RF-<DOM>-<NN>`,
		"a.md:3: RF-TRC-99 no existe en la spec viva",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("falta el problema %q en:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"RF-TRC-02", "RF-OLD-01", "RF-NOP-01"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("%s no debía reportarse:\n%s", unwanted, got)
		}
	}
}

// TestAxiomYAMLDeclaresRFPattern fija RF-RID-01 para este repositorio: axiom.yaml declara el
// patrón propio de Axiom.
func TestAxiomYAMLDeclaresRFPattern(t *testing.T) {
	root := mustRepoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		SDD struct {
			IDPattern string `yaml:"id_pattern"`
		} `yaml:"sdd"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.SDD.IDPattern != rfTemplate {
		t.Fatalf("axiom.yaml: sdd.id_pattern = %q, esperado %q", cfg.SDD.IDPattern, rfTemplate)
	}
	if _, err := reqid.Parse(cfg.SDD.IDPattern); err != nil {
		t.Fatalf("sdd.id_pattern no es un patrón válido: %v", err)
	}
}
