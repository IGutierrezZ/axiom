package legacyid

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const header = "schema: axiom.id-map/v1\nentries:\n"

func mapFS(content string) fstest.MapFS {
	return fstest.MapFS{FileName: {Data: []byte(content)}}
}

func loadText(t *testing.T, content string, p reqid.Pattern) (*Map, error) {
	t.Helper()
	return Load(mapFS(content), FileName, p)
}

func mustLoad(t *testing.T, content string) *Map {
	t.Helper()
	m, err := loadText(t, content, reqid.Default())
	if err != nil || m == nil {
		t.Fatalf("Load = %v, %v; quería un mapa", m, err)
	}
	return m
}

const sample = header + `
  - capability: autoskills
    old: [REQ-1.1]
    new: RF-ASK-01
  - capability: workspace-topology
    old: [REQ-1.1]
    new: RF-WTP-01
  - capability: persona-behavior-contract
    old: [REQ-10.1, REQ-10.2]
    new: RF-PER-08
  - capability: antigravity-support
    title: "Antigravity Adapter Registration"
    new: RF-AGV-01
  - capability: sdd-kickoff
    old: [REQ-27.2b]
    new: RF-KCK-09
`

func TestLegacyIDLoadAbsentFileIsNilNil(t *testing.T) {
	m, err := Load(fstest.MapFS{}, FileName, reqid.Default())
	if m != nil || err != nil {
		t.Fatalf("Load sin fichero = %v, %v; quería nil, nil", m, err)
	}
}

type unreadableFS struct{}

func (unreadableFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

func TestLegacyIDLoadErrorCodes(t *testing.T) {
	rf, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		fsys    fs.FS
		pattern reqid.Pattern
		code    string
		detail  string
	}{
		{"ilegible", unreadableFS{}, reqid.Default(), CodeUnreadable, ""},
		{"vacío", mapFS(""), reqid.Default(), CodeMalformed, ""},
		{"YAML inválido", mapFS("schema: [\n"), reqid.Default(), CodeMalformed, ""},
		{"campo desconocido", mapFS(header + "  - capability: a\n    old: [REQ-1.1]\n    new: RF-AAA-01\n    extra: 1\n"), reqid.Default(), CodeMalformed, "extra"},
		{"esquema desconocido", mapFS("schema: axiom.id-map/v9\nentries: []\n"), reqid.Default(), CodeSchemaUnknown, "axiom.id-map/v9"},
		{"sin esquema", mapFS("entries: []\n"), reqid.Default(), CodeSchemaUnknown, ""},
		{"sin entradas", mapFS("schema: axiom.id-map/v1\nentries: []\n"), reqid.Default(), CodeInvariant, "entradas"},
		{"ID nuevo repetido", mapFS(header + "  - capability: a\n    old: [REQ-1.1]\n    new: RF-AAA-01\n  - capability: b\n    old: [REQ-2.1]\n    new: RF-AAA-01\n"), reqid.Default(), CodeInvariant, "RF-AAA-01"},
		{"ID nuevo inválido", mapFS(header + "  - capability: a\n    old: [REQ-1.1]\n    new: nada\n"), reqid.Default(), CodeInvariant, "nada"},
		{"ID nuevo vacío", mapFS(header + "  - capability: a\n    old: [REQ-1.1]\n"), reqid.Default(), CodeInvariant, "ID nuevo"},
		{"antiguo repetido en la capacidad", mapFS(header + "  - capability: a\n    old: [REQ-1.1]\n    new: RF-AAA-01\n  - capability: a\n    old: [REQ-1.1]\n    new: RF-AAA-02\n"), reqid.Default(), CodeInvariant, "REQ-1.1"},
		{"antiguo repetido en la misma entrada", mapFS(header + "  - capability: a\n    old: [REQ-1.1, REQ-1.1]\n    new: RF-AAA-01\n"), rf, CodeInvariant, "REQ-1.1"},
		{"sin old ni title", mapFS(header + "  - capability: a\n    new: RF-AAA-01\n"), reqid.Default(), CodeInvariant, "old"},
		{"capacidad vacía", mapFS(header + "  - old: [REQ-1.1]\n    new: RF-AAA-01\n"), reqid.Default(), CodeInvariant, "capacidad"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Load(tc.fsys, FileName, tc.pattern)
			var me *MapError
			if m != nil || !errors.As(err, &me) {
				t.Fatalf("Load = %v, %v; quería *MapError", m, err)
			}
			if me.Code != tc.code {
				t.Fatalf("código = %q, quería %q (%v)", me.Code, tc.code, err)
			}
			text := err.Error()
			for _, want := range []string{FileName, tc.code, "git checkout -- openspec/id-map.yaml", tc.detail} {
				if !strings.Contains(text, want) {
					t.Errorf("Error() = %q; falta %q", text, want)
				}
			}
		})
	}
}

func TestLegacyIDLoadAllowsSameOldInTwoCapabilities(t *testing.T) {
	m := mustLoad(t, sample)
	if got := m.Capabilities("REQ-1.1"); !reflect.DeepEqual(got, []string{"autoskills", "workspace-topology"}) {
		t.Fatalf("Capabilities(REQ-1.1) = %v", got)
	}
}

func TestLegacyIDResolver(t *testing.T) {
	m := mustLoad(t, sample)
	if got := m.Aliases("REQ-1.1"); !reflect.DeepEqual(got, []string{"RF-ASK-01", "RF-WTP-01"}) {
		t.Errorf("Aliases(antiguo ambiguo) = %v", got)
	}
	if got := m.Aliases("RF-PER-08"); !reflect.DeepEqual(got, []string{"REQ-10.1", "REQ-10.2"}) {
		t.Errorf("Aliases(nuevo con dos antiguos) = %v", got)
	}
	if got := m.Aliases("RF-AGV-01"); len(got) != 0 {
		t.Errorf("Aliases(nuevo sin antiguo) = %v; quería vacío", got)
	}
	if got := m.Aliases("REQ-99.9"); len(got) != 0 {
		t.Errorf("Aliases(desconocido) = %v", got)
	}
	if got := m.InCapability("workspace-topology", "REQ-1.1"); got != "RF-WTP-01" {
		t.Errorf("InCapability = %q", got)
	}
	if got := m.InCapability("pagos", "REQ-1.1"); got != "" {
		t.Errorf("InCapability(par inexistente) = %q", got)
	}
	for id, want := range map[string]bool{"REQ-1.1": true, "REQ-10.2": true, "RF-ASK-01": false, "REQ-1.10": false, "Antigravity Adapter Registration": false} {
		if got := m.Legacy(id); got != want {
			t.Errorf("Legacy(%q) = %v, quería %v", id, got, want)
		}
	}
	for id, want := range map[string]bool{"REQ-1.1": true, "RF-ASK-01": true, "RF-AGV-01": true, "RF-ZZZ-99": false, "REQ-1.10": false} {
		if got := m.Known(id); got != want {
			t.Errorf("Known(%q) = %v, quería %v", id, got, want)
		}
	}
}

// Hueco 7: REQ-27.2b es una clave antigua más, de consulta exacta.
func TestLegacyIDSuffixedTokenIsExactKey(t *testing.T) {
	m := mustLoad(t, sample)
	if !m.Legacy("REQ-27.2b") || m.Legacy("REQ-27.2") || m.Legacy("REQ-27.2bb") {
		t.Fatalf("la clave REQ-27.2b debe ser exacta: Legacy(27.2b)=%v Legacy(27.2)=%v", m.Legacy("REQ-27.2b"), m.Legacy("REQ-27.2"))
	}
	if got := m.Aliases("REQ-27.2b"); !reflect.DeepEqual(got, []string{"RF-KCK-09"}) {
		t.Fatalf("Aliases(REQ-27.2b) = %v", got)
	}
}

// Hueco 4: los IDs nuevos se validan con el patrón del mapa (RF-<DOM>-<NN>), nunca exigiendo
// REQ-<n>.<m>; y el mapa inválido se detecta aunque el patrón de consulta sea el de por defecto.
func TestLegacyIDNewIDsUseRFPatternNotDefault(t *testing.T) {
	rf, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []reqid.Pattern{reqid.Default(), rf} {
		m, err := loadText(t, sample, p)
		if err != nil || m == nil {
			t.Fatalf("Load con el patrón %s falló: %v", p.Template(), err)
		}
	}
	_, err = loadText(t, header+"  - capability: a\n    old: [REQ-1.1]\n    new: REQ-5.1\n", reqid.Default())
	if err != nil {
		t.Fatalf("un ID nuevo válido bajo el patrón de consulta debe aceptarse: %v", err)
	}
	_, err = loadText(t, header+"  - capability: a\n    old: [REQ-1.1]\n    new: RF-aaa-1\n", rf)
	var me *MapError
	if !errors.As(err, &me) || me.Code != CodeInvariant {
		t.Fatalf("un ID nuevo inválido bajo ambos patrones debe rechazarse: %v", err)
	}
}
