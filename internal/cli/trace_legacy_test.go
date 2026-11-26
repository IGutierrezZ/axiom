package cli

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/legacyid"
	"github.com/IGutierrezZ/axiom/v3/internal/reqtrace"
)

// C-03-e: `axiom trace` con IDs heredados y el mapa versionado `openspec/id-map.yaml`. Los
// nombres empiezan por TestAxiomTrace para entrar en el mismo patrón de `-run`.

const axLegacyMap = "schema: axiom.id-map/v1\nentries:\n" +
	"  - capability: pagos\n    old: [REQ-24.3]\n    new: RF-PAG-03\n" +
	"  - capability: autoskills\n    old: [REQ-1.1]\n    new: RF-ASK-01\n" +
	"  - capability: workspace-topology\n    old: [REQ-1.1]\n    new: RF-WTP-01\n"

// newAxLegacyProject monta un proyecto con el patrón RF y un cambio archivado de `pagos` que
// modifica REQ-24.3. mapText vacío no escribe mapa.
func newAxLegacyProject(t *testing.T, mapText string) string {
	t.Helper()
	repo := axiomTraceProject(t, "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	files := map[string]string{
		"openspec/changes/archive/2026-11-12-alta/specs/pagos/spec.md": axTraceSpec("MODIFIED", "Cobro (REQ-24.3)"),
	}
	if mapText != "" {
		files["openspec/id-map.yaml"] = mapText
	}
	axTraceCommit(t, repo, "alta archivada", files)
	return repo
}

func TestAxiomTraceLegacyIDAcceptedAndResolvedWithMap(t *testing.T) {
	repo := newAxLegacyProject(t, axLegacyMap)
	for _, id := range []string{"REQ-24.3", "RF-PAG-03"} {
		result := axTraceRun(t, id, "--cwd", repo)
		if !reflect.DeepEqual(result.ResolvedIDs, []string{"REQ-24.3", "RF-PAG-03"}) {
			t.Errorf("%s: IDs resueltos = %v", id, result.ResolvedIDs)
		}
		if len(result.Increments) != 1 || result.Increments[0].Name != "alta" || result.Increments[0].Role != reqtrace.RoleModified {
			t.Errorf("%s: incrementos = %+v", id, result.Increments)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("%s: avisos = %v", id, result.Warnings)
		}
	}
}

func TestAxiomTraceLegacyAmbiguousIDWarnsAndResolvesAll(t *testing.T) {
	repo := newAxLegacyProject(t, axLegacyMap)
	result := axTraceRun(t, "REQ-1.1", "--cwd", repo)
	if !reflect.DeepEqual(result.ResolvedIDs, []string{"REQ-1.1", "RF-ASK-01", "RF-WTP-01"}) {
		t.Fatalf("IDs resueltos = %v", result.ResolvedIDs)
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != reqtrace.WarnLegacyAmbiguous {
		t.Fatalf("avisos = %v; quería un legacy_ambiguous", result.Warnings)
	}
}

func TestAxiomTraceLegacyIDOutsideTheMapFailsWithoutGit(t *testing.T) {
	repo := newAxLegacyProject(t, axLegacyMap)
	calls := observeGit(t)
	_, err := axiomTrace("REQ-99.9", "--cwd", repo)
	if err == nil {
		t.Fatal("un REQ-n.m fuera del mapa debe rechazarse")
	}
	for _, want := range []string{"REQ-99.9", "RF-<DOM>-<NN>", "axiom trace --help"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("se llamó a git con un ID fuera del mapa: %v", *calls)
	}
}

func TestAxiomTraceWithoutMapHasNoAliasesNorWarning(t *testing.T) {
	repo := newAxLegacyProject(t, "")
	result := axTraceRun(t, "RF-PAG-03", "--cwd", repo)
	if !reflect.DeepEqual(result.ResolvedIDs, []string{"RF-PAG-03"}) || len(result.Warnings) != 0 || len(result.Increments) != 0 {
		t.Fatalf("sin mapa: resueltos %v, avisos %v, incrementos %v", result.ResolvedIDs, result.Warnings, result.Increments)
	}
	if _, err := axiomTrace("REQ-24.3", "--cwd", repo); err == nil {
		t.Fatal("sin mapa un REQ-n.m debe rechazarse")
	}
}

func TestAxiomTraceInvalidMapFailsTypedWithoutGit(t *testing.T) {
	for name, content := range map[string]string{
		"YAML inválido":       "schema: [\n",
		"esquema desconocido": "schema: axiom.id-map/v9\nentries: []\n",
		"invariante":          "schema: axiom.id-map/v1\nentries:\n  - capability: a\n    old: [REQ-1.1]\n    new: nada\n",
	} {
		t.Run(name, func(t *testing.T) {
			repo := newAxLegacyProject(t, content)
			calls := observeGit(t)
			// El mapa inválido gana siempre sobre un ID de consulta inválido.
			for _, id := range []string{"RF-PAG-03", "REQ-99.9", "no-es-un-id"} {
				_, err := axiomTrace(id, "--cwd", repo)
				var mapErr *legacyid.MapError
				if !errors.As(err, &mapErr) {
					t.Fatalf("%s: quería *legacyid.MapError, obtuve %v", id, err)
				}
				if !strings.Contains(err.Error(), filepath.Join("openspec", "id-map.yaml")) || !strings.Contains(err.Error(), mapErr.Code) {
					t.Errorf("%s: el error debe nombrar el fichero y el código: %v", id, err)
				}
			}
			if len(*calls) != 0 {
				t.Fatalf("se llamó a git con un mapa inválido: %v", *calls)
			}
		})
	}
}
