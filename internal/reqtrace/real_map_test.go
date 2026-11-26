package reqtrace

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/legacyid"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// realProject carga el mapa y el patrón reales del repositorio. Las pruebas leen con os porque
// el límite de arquitectura de reqtrace (imports_test.go) solo vigila los ficheros de producción.
func realProject(t *testing.T) (reqid.Pattern, *legacyid.Map) {
	t.Helper()
	data, err := os.ReadFile("../../axiom.yaml")
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
	p, err := reqid.Parse(cfg.SDD.IDPattern)
	if err != nil {
		t.Fatalf("sdd.id_pattern de axiom.yaml (%q): %v", cfg.SDD.IDPattern, err)
	}
	m, err := legacyid.Load(os.DirFS("../../openspec"), legacyid.FileName, p)
	if err != nil || m == nil {
		t.Fatalf("mapa real openspec/%s: %v, %v", legacyid.FileName, m, err)
	}
	return p, m
}

// TestTraceRealMapResolvesLegacyID es la verificación en proceso de que `axiom trace` de un ID
// heredado responde con el ID nuevo, sin depender de la historia git (la CI usa un clon
// superficial).
func TestTraceRealMapResolvesLegacyID(t *testing.T) {
	p, m := realProject(t)

	resolve := func(raw string) (Query, []string, []Warning) {
		t.Helper()
		q, err := ParseID(raw, p, m)
		if err != nil {
			t.Fatalf("ParseID(%s): %v", raw, err)
		}
		if !q.Legacy {
			t.Fatalf("ParseID(%s) = %+v: debe ser una consulta heredada", raw, q)
		}
		eq, w := resolveEquivalentsLegacy(q, p, nil, m, m)
		var news []string
		for _, id := range eq.resolved {
			if !m.Legacy(id) {
				news = append(news, id)
			}
		}
		return q, news, w
	}

	if _, got, w := resolve("REQ-24.3"); !reflect.DeepEqual(got, []string{"RF-PLH-03"}) || len(legacyWarnings(w)) != 0 {
		t.Errorf("REQ-24.3 resuelve a %v con avisos %v, esperado [RF-PLH-03] sin legacy_ambiguous", got, w)
	}

	_, got, w := resolve("REQ-1.1")
	if len(got) != 7 {
		t.Errorf("REQ-1.1 resuelve a %d IDs nuevos %v, esperados siete", len(got), got)
	}
	if len(legacyWarnings(w)) != 1 {
		t.Errorf("REQ-1.1 debe emitir un aviso legacy_ambiguous, avisos: %v", w)
	}

	newID := m.InCapability("axiom-sdd-cli-integration", "REQ-27.2b")
	if newID == "" {
		t.Fatal("el mapa real no resuelve REQ-27.2b en axiom-sdd-cli-integration")
	}
	if _, got, _ := resolve("REQ-27.2b"); !reflect.DeepEqual(got, []string{newID}) {
		t.Errorf("REQ-27.2b resuelve a %v, esperado [%s]", got, newID)
	}
}

// TestTraceRealMapFindsTitleEntryIncrements comprueba con el mapa y los cambios archivados
// reales que un requisito sin ID antiguo (RF-ENG-01) encuentra por título los incrementos
// anteriores a la migración (RF-TRC-04, regla 5).
func TestTraceRealMapFindsTitleEntryIncrements(t *testing.T) {
	p, m := realProject(t)
	if c, title := m.Title("RF-ENG-01"); c != "engram-protocol-injection" || title != "Single canonical Codex protocol asset" {
		t.Fatalf("Title(RF-ENG-01) = %q, %q", c, title)
	}
	q, err := ParseID("RF-ENG-01", p, m)
	if err != nil {
		t.Fatal(err)
	}
	src := Source{Files: os.DirFS("../../openspec"), GitPrefix: "openspec"}
	folders, _ := listSDDFolders(src)
	traces, _ := readSDDTraces(src, folders)
	eq, _ := resolveEquivalentsLegacy(q, p, traces, m, m)
	found := 0
	for _, f := range folders {
		c, _ := classifySDD(src, p, eq, f)
		found += len(c)
	}
	if found == 0 {
		t.Fatal("RF-ENG-01 no encuentra ningún cambio archivado por su título")
	}
}
