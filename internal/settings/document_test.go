package settings

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RF-TOP-22: Read devuelve el valor tal como está escrito, aunque el documento sea
// inválido, de modo que la pantalla permita corregirlo.
func TestReadReturnsValuesAsWrittenInAnInvalidDocument(t *testing.T) {
	doc := monoDoc + "workflow:\n  lane: zzz\ngit:\n  promotion: siempre\n"
	snap := readSnapshot(t, doc)
	if got := mustEntry(t, snap, "workflow.lane").Value; got != "zzz" {
		t.Errorf("lane = %v, quiero el valor escrito", got)
	}
	if e := mustEntry(t, snap, "git.promotion"); e.Value != "siempre" || !e.Declared {
		t.Errorf("promotion = %+v", e)
	}
	if len(snap.DocumentErrors) == 0 {
		t.Error("DocumentErrors debe traer las violaciones de ParseConfig")
	}
	if !snap.Writable {
		t.Errorf("un documento semánticamente inválido sigue siendo escribible: %s", snap.WriteBlockedReason)
	}
	if e := mustEntry(t, snap, "git.integration_branch"); e.Declared || e.Value != nil {
		t.Errorf("una clave sin declarar no tiene valor: %+v", e)
	}
}

// RF-TOP-22: valores tipados, revisión SHA-256, ruta, topología e ISO.
func TestReadTypedValuesRevisionAndMetadata(t *testing.T) {
	doc := monoDoc + "delivery:\n  split_hint_lines: 400\nsdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n"
	path := writeConfig(t, doc)
	snap, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ConfigPath != path {
		t.Errorf("ConfigPath = %q", snap.ConfigPath)
	}
	if want := revisionOf([]byte(doc)); snap.Revision != want || len(want) != 64 {
		t.Errorf("Revision = %q, quiero %q", snap.Revision, want)
	}
	if snap.Topology != workspace.TopologyMonorepoEmbedded || !snap.ISO || !snap.Writable {
		t.Errorf("snapshot = %+v", snap)
	}
	if got := mustEntry(t, snap, "delivery.split_hint_lines").Value; got != 400 {
		t.Errorf("split_hint_lines = %#v", got)
	}
	if got := mustEntry(t, snap, "sdd.iso").Value; got != true {
		t.Errorf("iso = %#v", got)
	}
	if got := mustEntry(t, snap, "sdd.id_pattern").Value; got != "RF-<DOM>-<NN>" {
		t.Errorf("id_pattern = %#v", got)
	}
	multi := readSnapshot(t, multiDoc+"acceptance_tests:\n  enabled: true\n  environments: [dev, pro]\n")
	if multi.Topology != workspace.TopologyMultirepo {
		t.Errorf("Topology = %q", multi.Topology)
	}
	env := mustEntry(t, multi, "acceptance_tests.environments").Value
	if list, ok := env.([]string); !ok || len(list) != 2 || list[1] != "pro" {
		t.Errorf("environments = %#v", env)
	}
}

// RF-TOP-22: el valor de tipo distinto del esperado se devuelve en crudo.
func TestReadReturnsRawValueOfAWrongType(t *testing.T) {
	snap := readSnapshot(t, monoDoc+"delivery:\n  split_hint_lines: muchas\nsdd:\n  iso: \"true\"\n")
	if got := mustEntry(t, snap, "delivery.split_hint_lines").Value; got != "muchas" {
		t.Errorf("split_hint_lines = %#v", got)
	}
	if got := mustEntry(t, snap, "sdd.iso").Value; got != "true" {
		t.Errorf("iso entre comillas = %#v", got)
	}
	if snap.ISO {
		t.Error("`iso: \"true\"` entre comillas no es ISO")
	}
}

// RF-TOP-22: multidocumento, anclas, alias, `<<` y claves repetidas dejan el
// documento sin permiso de escritura, con la construcción y su línea.
func TestReadBlocksUnsupportedConstructs(t *testing.T) {
	cases := []struct {
		name, doc, wantReason string
	}{
		{"multidocumento", monoDoc + "---\nworkspace:\n  name: otro\n", "más de un documento"},
		{"ancla", monoDoc + "git:\n  integration_branch: &rama main\n", "ancla &rama (línea"},
		{"alias", monoDoc + "git:\n  integration_branch: &rama main\n  main_branch: *rama\n", "ancla"},
		{"mezcla", monoDoc + "base: &b\n  a: 1\nextra:\n  <<: *b\n", "ancla"},
		{"clave repetida", monoDoc + "git:\n  branch_prefix: a\n  branch_prefix: b\n", "repite la clave \"branch_prefix\" (línea"},
		{"yaml roto", "workspace: [\n", "no es YAML válido"},
		{"vacío", "", "vacío"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := readSnapshot(t, tc.doc)
			if snap.Writable {
				t.Fatal("el documento no debería ser escribible")
			}
			if !strings.Contains(snap.WriteBlockedReason, tc.wantReason) {
				t.Errorf("WriteBlockedReason = %q, debe contener %q", snap.WriteBlockedReason, tc.wantReason)
			}
		})
	}
	merge := readSnapshot(t, "base: &b\n  a: 1\nextra:\n  <<: *b\n")
	if merge.Writable {
		t.Error("`<<` debe bloquear la escritura")
	}
	if !strings.Contains(parseDocument([]byte("extra:\n  <<: {a: 1}\n")).blocked, "clave de mezcla `<<` (línea 2)") {
		t.Error("`<<` debe nombrarse con su línea")
	}
}

// RF-TOP-22: el bloque `integrations` no se recorre, así que sus construcciones no
// bloquean ni sus valores llegan al catálogo.
func TestReadDoesNotWalkIntegrations(t *testing.T) {
	doc := monoDoc + "integrations:\n  jira: &j\n    token: secreto-uno\n  otra: *j\n"
	snap := readSnapshot(t, doc)
	if !snap.Writable {
		t.Errorf("integrations no debe bloquear: %s", snap.WriteBlockedReason)
	}
	for _, e := range snap.Entries {
		if strings.Contains(e.Key, "integrations") {
			t.Errorf("entrada de integrations en el catálogo: %+v", e)
		}
	}
}

// RF-TOP-22: un fichero inexistente es config_missing.
func TestReadMissingFileIsConfigMissing(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "axiom.yaml"))
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != CodeConfigMissing {
		t.Fatalf("err = %v, quiero config_missing", err)
	}
	if typed.Message == "" {
		t.Error("el mensaje debe estar en castellano")
	}
}

// RF-TOP-21: entradas de repositories con ruta normalizada, unión con las de los
// roles y orden estable.
func TestReadExpandsRepositoriesFromEntriesAndRoles(t *testing.T) {
	doc := multiDoc + "repositories:\n  - path: .\\repo-backend\\\n    promotion: release\n  - path: extra\n"
	snap := readSnapshot(t, doc)
	if e := mustEntry(t, snap, "repositories[repo-backend].promotion"); e.Value != "release" || !e.Declared {
		t.Errorf("promotion = %+v", e)
	}
	mustEntry(t, snap, "repositories[extra].branch_prefix")
	mustEntry(t, snap, "repositories[repo-frontend].branch_prefix")
}
