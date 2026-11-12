package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// REQ-29.10 y REQ-29.11: archive-compose se niega a componer la delta de un cambio
// ISO que no cumple la regla del archivado. Los nombres empiezan por
// TestArchiveCompose para entrar en el mismo patrón de `-run`.

func TestArchiveComposeISORefusal(t *testing.T) {
	tests := []struct {
		name   string
		change archiveCheckChange
		// token es el motivo esperado en el rechazo; vacío si se acepta.
		token string
	}{
		{
			name:   "ISO con escenarios y matriz válida",
			change: archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(archiveCheckRowPass), iso: true},
		},
		{
			name:   "ISO con un ADDED sin escenarios",
			change: archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(), iso: true},
			token:  "iso_requirement_scenarios",
		},
		{
			name:   "ISO con la matriz ausente",
			change: archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone, iso: true},
			token:  "iso_matrix_missing",
		},
		{
			name:   "ISO con la matriz inválida",
			change: archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(), iso: true},
			token:  "iso_matrix_invalid",
		},
		{
			name:   "cambio no ISO nunca se rechaza",
			change: archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, changeRoot := archiveCheckWorkspace(t, tt.change)
			deltaPath := filepath.Join(changeRoot, "specs", "pagos", "spec.md")
			loc, err := workspace.Locate(root)
			if err != nil {
				t.Fatal(err)
			}

			for mode, refuse := range map[string]func() error{
				"documento":   func() error { return refuseISOCompose(loc, true, deltaPath, "pagos") },
				"--canonical": func() error { return refuseISOCompose(workspace.Location{}, false, deltaPath, "") },
			} {
				err := refuse()
				switch {
				case tt.token == "" && err != nil:
					t.Errorf("modo %s: error = %v, se esperaba aceptar", mode, err)
				case tt.token != "" && err == nil:
					t.Errorf("modo %s: se esperaba rechazar con %s", mode, tt.token)
				case tt.token != "" && (!strings.Contains(err.Error(), "blocked("+tt.token+")") || !strings.Contains(err.Error(), "archive-check")):
					t.Errorf("modo %s: el rechazo no nombra el motivo ni la continuación: %v", mode, err)
				}
			}
		})
	}
}

// Sin cambio determinable no se aplica nada (último escenario de REQ-29.10): una
// delta suelta, fuera de changes/<cambio>/specs/<dominio>/, o fuera de todo espacio.
func TestArchiveComposeISORefusalWithoutChangeContext(t *testing.T) {
	root, changeRoot := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone, iso: true,
	})
	loc, err := workspace.Locate(root)
	if err != nil {
		t.Fatal(err)
	}

	loose := filepath.Join(root, "delta-suelta", "pagos", "spec.md")
	writeCLIFixture(t, loose, archiveCheckDeltaNoScene)
	if err := refuseISOCompose(loc, true, loose, "pagos"); err != nil {
		t.Errorf("delta suelta (modo documento): error = %v, sin cambio no se aplica nada", err)
	}
	if err := refuseISOCompose(workspace.Location{}, false, loose, ""); err != nil {
		t.Errorf("delta suelta (--canonical): error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "pagos", "spec.md")
	writeCLIFixture(t, outside, archiveCheckDeltaNoScene)
	if err := refuseISOCompose(workspace.Location{}, false, outside, ""); err != nil {
		t.Errorf("delta fuera de todo espacio: error = %v", err)
	}
	// El dominio que se indica debe ser el de la carpeta de la delta.
	if err := refuseISOCompose(loc, true, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), "otro"); err != nil {
		t.Errorf("dominio que no es el de la carpeta de la delta: error = %v", err)
	}
}
