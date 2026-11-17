package sddstatus

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

func contextWorkspace(t *testing.T, blocks string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+blocks)
	return ws
}

func TestResolveWorkspaceTechnicalContextProjectsDeclaredModes(t *testing.T) {
	cases := map[string]struct {
		blocks         string
		wantMaintained *ContextModeProjection
		wantReference  *ContextModeProjection
	}{
		"dos modos": {
			blocks:         declBothModes,
			wantMaintained: &ContextModeProjection{Entry: "context/TECHNICAL_CONTEXT.md", Paths: []string{"context"}},
			wantReference:  &ContextModeProjection{Paths: []string{"00_Glosario.md"}},
		},
		"solo mantenido": {
			blocks:         "technical_context:\n  maintained:\n    entry: ctx/ENTRY.md\n    paths: [ctx/]\n",
			wantMaintained: &ContextModeProjection{Entry: "ctx/ENTRY.md", Paths: []string{"ctx"}},
		},
		"solo referencia": {
			blocks:        "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n",
			wantReference: &ContextModeProjection{Paths: []string{"00_Glosario.md"}},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ws := contextWorkspace(t, tc.blocks)
			got, err := ResolveWorkspaceTechnicalContext(ws.Master)
			if err != nil {
				t.Fatal(err)
			}
			if got.SchemaName != "axiom.workspace-context" || got.SchemaVersion != 1 || got.Invalid != "" {
				t.Fatalf("cabecera inesperada: %+v", got)
			}
			if got.TechnicalContext == nil {
				t.Fatal("technicalContext no debe ser nil")
			}
			absolutize := func(mode *ContextModeProjection) *ContextModeProjection {
				if mode == nil {
					return nil
				}
				out := &ContextModeProjection{Paths: []string{}}
				if mode.Entry != "" {
					out.Entry = filepath.Join(got.WorkspaceRoot, filepath.FromSlash(mode.Entry))
				}
				for _, p := range mode.Paths {
					out.Paths = append(out.Paths, filepath.Join(got.WorkspaceRoot, filepath.FromSlash(p)))
				}
				return out
			}
			if want := absolutize(tc.wantMaintained); !reflect.DeepEqual(got.TechnicalContext.Maintained, want) {
				t.Errorf("maintained = %+v, quiero %+v", got.TechnicalContext.Maintained, want)
			}
			if want := absolutize(tc.wantReference); !reflect.DeepEqual(got.TechnicalContext.Reference, want) {
				t.Errorf("reference = %+v, quiero %+v", got.TechnicalContext.Reference, want)
			}
			if !filepath.IsAbs(got.WorkspaceRoot) {
				t.Errorf("workspaceRoot no es absoluta: %q", got.WorkspaceRoot)
			}
		})
	}
}

func TestResolveWorkspaceTechnicalContextWithoutDeclarationIsNullInJSON(t *testing.T) {
	ws := contextWorkspace(t, "")
	got, err := ResolveWorkspaceTechnicalContext(ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	if got.TechnicalContext != nil || got.Invalid != "" {
		t.Fatalf("sin declaración no hay proyección ni motivo: %+v", got)
	}
	raw, _ := json.Marshal(got)
	if !strings.Contains(string(raw), `"technicalContext":null`) || strings.Contains(string(raw), `"invalid"`) {
		t.Errorf("JSON inesperado: %s", raw)
	}
}

func TestResolveWorkspaceTechnicalContextInvalidDeclarationFailsClosed(t *testing.T) {
	ws := contextWorkspace(t, "technical_context:\n  maintained:\n    paths: []\n")
	got, err := ResolveWorkspaceTechnicalContext(ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	if got.TechnicalContext != nil {
		t.Errorf("una declaración inválida no se proyecta: %+v", got.TechnicalContext)
	}
	if !strings.Contains(got.Invalid, "paths") {
		t.Errorf("invalid debe nombrar el motivo: %q", got.Invalid)
	}

	unreadable := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(unreadable.Master, "axiom.yaml"), "workspace: [sin cerrar\n")
	got, err = ResolveWorkspaceTechnicalContext(unreadable.Master)
	if err != nil {
		t.Fatal(err)
	}
	if got.TechnicalContext != nil || got.Invalid == "" {
		t.Errorf("un axiom.yaml ilegible falla cerrado: %+v", got)
	}
}

// Verificación pendiente del plan C-04-b: sin --change, el status proyecta
// technicalContext aunque esté bloqueado por 0 o varios cambios activos.
func TestResolveWithoutChangeProjectsTechnicalContextWhenBlocked(t *testing.T) {
	cases := map[string][]string{
		"0 cambios activos": nil,
		"2 cambios activos": {"first", "second"},
	}
	for name, changes := range cases {
		t.Run(name, func(t *testing.T) {
			ws := contextWorkspace(t, declBothModes)
			mkdir(t, filepath.Join(ws.Master, "openspec", "changes"))
			for _, change := range changes {
				mkdir(t, filepath.Join(ws.Master, "openspec", "changes", change))
			}
			status, err := Resolve(ResolveOptions{CWD: ws.Master})
			if err != nil {
				t.Fatal(err)
			}
			if len(status.BlockedReasons) == 0 {
				t.Fatalf("sin --change el status debe estar bloqueado: %+v", status.BlockedReasons)
			}
			if status.TechnicalContext == nil || status.TechnicalContext.Maintained == nil || status.TechnicalContext.Reference == nil {
				t.Fatalf("technicalContext debe proyectarse con el status bloqueado: %+v", status.TechnicalContext)
			}
			wantEntry := filepath.Join(ws.Master, "context", "TECHNICAL_CONTEXT.md")
			if got := status.TechnicalContext.Maintained.Entry; got != wantEntry {
				t.Errorf("entry = %q, quiero %q", got, wantEntry)
			}
			fields := statusJSON(t, status)
			if _, present := fields["technicalContext"]; !present {
				t.Error("el JSON v2 debe llevar technicalContext")
			}
		})
	}
}
