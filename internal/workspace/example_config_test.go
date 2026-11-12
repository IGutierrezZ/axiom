package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// carril-sdd-versatil U24 (REQ-29.17): la plantilla axiom.example.yaml y el
// manual documentan el modo ISO sin desviarse del esquema.

func readExampleConfig(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "axiom.example.yaml"))
	if err != nil {
		t.Fatalf("no se pudo leer axiom.example.yaml: %v", err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

// exampleBlock devuelve el bloque de primer nivel `<key>:` comentado en la
// plantilla, descomentado, sin exigir un mínimo de claves.
func exampleBlock(t *testing.T, example, key string) string {
	t.Helper()
	var block []string
	for _, line := range strings.Split(example, "\n") {
		if len(block) == 0 {
			if line == "# "+key+":" {
				block = append(block, key+":")
			}
			continue
		}
		if !strings.HasPrefix(line, "#   ") {
			break
		}
		block = append(block, strings.TrimPrefix(line, "# "))
	}
	if len(block) < 2 {
		t.Fatalf("axiom.example.yaml debe documentar un bloque `# %s:` con sus claves, y se halló %q", key, block)
	}
	return strings.Join(block, "\n") + "\n"
}

// La plantilla carga sin error con su bloque comentado, y el bloque `sdd:` que
// documenta es válido una vez descomentado.
func TestExampleConfigLoads(t *testing.T) {
	example := readExampleConfig(t)

	if _, baseErr, declarations := parseLayered([]byte(example)); baseErr != nil || declarations.SDDErr != nil || declarations.WorkflowErr != nil {
		t.Fatalf("la plantilla debe cargar sin errores: base=%v sdd=%v workflow=%v", baseErr, declarations.SDDErr, declarations.WorkflowErr)
	}

	block := exampleBlock(t, example, "sdd")
	_, baseErr, declarations := parseLayered([]byte(example + "\n" + block))
	if baseErr != nil || declarations.SDDErr != nil {
		t.Fatalf("el bloque sdd documentado debe ser válido: base=%v sdd=%v", baseErr, declarations.SDDErr)
	}
	if declarations.SDD == nil || !declarations.SDD.ISO {
		t.Fatalf("el bloque documentado declara iso: true: %+v", declarations.SDD)
	}
}

// `iso: true` no convive con `workflow.lane: odd`, ni siquiera con la plantilla.
func TestExampleISOBlockRejectsTheODDLane(t *testing.T) {
	example := readExampleConfig(t)
	block := exampleBlock(t, example, "sdd")

	_, _, declarations := parseLayered([]byte(example + "\n" + block + "workflow:\n  lane: odd\n"))
	if !errors.Is(declarations.SDDErr, ErrISOWorkflowContradiction) {
		t.Fatalf("iso: true con workflow.lane: odd debe rechazarse con ErrISOWorkflowContradiction, y se obtuvo %v", declarations.SDDErr)
	}

	// Con cualquier otro carril es válido.
	for _, lane := range []string{"sdd", "ask"} {
		_, _, declarations := parseLayered([]byte(example + "\n" + block + "workflow:\n  lane: " + lane + "\n"))
		if declarations.SDDErr != nil || declarations.WorkflowErr != nil {
			t.Errorf("iso: true con workflow.lane: %s es válido: sdd=%v workflow=%v", lane, declarations.SDDErr, declarations.WorkflowErr)
		}
	}
}

// El manual documenta la declaración, el perfil de bug con sus tres rutas, la
// comprobación previa al archivado y la traza.
func TestManualDocumentsTheISOMode(t *testing.T) {
	doc := readManualDeInicio(t)
	assertDocContains(t, doc,
		"[Modo ISO y Perfil de Bug (`sdd:`)](#modo-iso-y-perfil-de-bug-sdd)",
		"### Modo ISO y Perfil de Bug (`sdd:`)",
		"iso: true",
		"`axiom workspace validate` la rechaza",
		"`## Informe de bug`",
		"`## Impacto en la spec`",
		"`Sin impacto en la spec`",
		"`## Matriz de verificación`",
		"`Cubre: REQ-x[, REQ-y ...]`",
		"axiom sdd archive-check",
		"`alreadyComposed`",
		"axiom sdd trace",
	)
}
