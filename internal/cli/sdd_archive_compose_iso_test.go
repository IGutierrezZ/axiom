package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
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
				"documento":   func() error { return refuseISOCompose(loc, deltaPath, []string{"pagos"}) },
				"--canonical": func() error { return refuseISOCompose(workspace.Location{}, deltaPath, nil) },
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
	if err := refuseISOCompose(loc, loose, []string{"pagos"}); err != nil {
		t.Errorf("delta suelta (modo documento): error = %v, sin cambio no se aplica nada", err)
	}
	if err := refuseISOCompose(workspace.Location{}, loose, nil); err != nil {
		t.Errorf("delta suelta (--canonical): error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "pagos", "spec.md")
	writeCLIFixture(t, outside, archiveCheckDeltaNoScene)
	if err := refuseISOCompose(workspace.Location{}, outside, nil); err != nil {
		t.Errorf("delta fuera de todo espacio: error = %v", err)
	}
	// El dominio que se indica debe ser el de la carpeta de la delta.
	if err := refuseISOCompose(loc, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), []string{"otro"}); err != nil {
		t.Errorf("dominio que no es el de la carpeta de la delta: error = %v", err)
	}
}

// archiveComposeCanonical es la canónica del dominio `pagos` sobre la que componen
// las pruebas de extremo a extremo.
const archiveComposeCanonical = "# pagos\n\n## Requirements\n\n### Requirement: REQ-5.1 Recibo\n\nEl sistema emite recibo.\n\n#### Scenario: Recibo emitido\n\n- **CUANDO** se paga\n"

// composeISOFixture monta el cambio y la canónica y devuelve las rutas de ambas.
func composeISOFixture(t *testing.T, change archiveCheckChange) (root, canonical, delta string) {
	t.Helper()
	root, changeRoot := archiveCheckWorkspace(t, change)
	canonical = filepath.Join(root, "openspec", "specs", "pagos", "spec.md")
	writeCLIFixture(t, canonical, archiveComposeCanonical)
	return root, canonical, filepath.Join(changeRoot, "specs", "pagos", "spec.md")
}

// REQ-29.10 y REQ-29.11, de extremo a extremo: el rechazo no escribe nada, ni en
// la salida ni en la canónica.
func TestArchiveComposeISORefusesEndToEndWithoutWriting(t *testing.T) {
	change := archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(), iso: true}
	root, canonical, delta := composeISOFixture(t, change)
	before := treeSnapshot(t, root)

	for name, args := range map[string][]string{
		"--canonical": {"--canonical", canonical, "--delta", delta},
	} {
		var stdout, stderr bytes.Buffer
		err := runSDDArchiveComposeWithStderr(args, &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "blocked(iso_requirement_scenarios)") {
			t.Fatalf("modo %s: error = %v, se esperaba el rechazo ISO", name, err)
		}
		if stdout.Len() != 0 || stderr.Len() != 0 {
			t.Errorf("modo %s: el rechazo escribió salida: %q %q", name, stdout.String(), stderr.String())
		}
	}
	after := treeSnapshot(t, root)
	for path, text := range before {
		if after[path] != text {
			t.Errorf("el rechazo modificó %s", path)
		}
	}
	if len(after) != len(before) {
		t.Errorf("el rechazo creó ficheros: antes %d, después %d", len(before), len(after))
	}
}

// REQ-24.38: la salida de una composición aceptada de un cambio ISO y la de un
// cambio no ISO son idénticas byte a byte, y coinciden con el motor nativo.
func TestArchiveComposeISOAcceptedOutputIsByteIdentical(t *testing.T) {
	isoChange := archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaOK}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(archiveCheckRowPass), iso: true}
	plainChange := isoChange
	plainChange.iso = false

	compose := func(change archiveCheckChange) string {
		t.Helper()
		_, canonical, delta := composeISOFixture(t, change)
		var stdout, stderr bytes.Buffer
		if err := runSDDArchiveComposeWithStderr([]string{"--canonical", canonical, "--delta", delta}, &stdout, &stderr); err != nil {
			t.Fatalf("iso=%v: error = %v", change.iso, err)
		}
		return stdout.String()
	}
	native, err := sddstatus.ComposeOpenSpecCanonicalSpec(archiveComposeCanonical, archiveCheckDeltaOK)
	if err != nil {
		t.Fatal(err)
	}
	iso, plain := compose(isoChange), compose(plainChange)
	if iso != plain || iso != native {
		t.Errorf("la salida difiere\nISO:\n%s\nno ISO:\n%s\nnativa:\n%s", iso, plain, native)
	}
}

// Un cambio no ISO sigue componiendo un ADDED sin escenarios, como antes.
func TestArchiveComposeNonISOKeepsComposingWithoutScenarios(t *testing.T) {
	change := archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone}
	_, canonical, delta := composeISOFixture(t, change)
	var stdout, stderr bytes.Buffer
	if err := runSDDArchiveComposeWithStderr([]string{"--canonical", canonical, "--delta", delta}, &stdout, &stderr); err != nil {
		t.Fatalf("un cambio no ISO compone sin escenarios: %v", err)
	}
	if !strings.Contains(stdout.String(), "REQ-30.1 Cobro con IVA") {
		t.Errorf("la composición no incluye el requisito añadido:\n%s", stdout.String())
	}
}

// Los modos retirados con la baseline ajena (`--document`, `--change`, `--in-place`) se
// rechazan con un motivo tipado, antes de la regla ISO y sin escribir nada
// (REQ-30.22).
func TestArchiveComposeRetiredModesAreRejectedWithATypedError(t *testing.T) {
	change := archiveCheckChange{name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(), iso: true}
	root, canonical, delta := composeISOFixture(t, change)
	before := treeSnapshot(t, root)

	for name, tc := range map[string]struct {
		args  []string
		flags []string
	}{
		"--document": {[]string{"--cwd", root, "--document", "pagos", "--delta", delta}, []string{"--document"}},
		"--in-place": {[]string{"--cwd", root, "--change", "demo", "--document", "pagos", "--in-place"}, []string{"--document", "--change", "--in-place"}},
		"mezclado":   {[]string{"--canonical", canonical, "--delta", delta, "--document", "pagos"}, []string{"--document"}},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := runSDDArchiveComposeWithStderr(tc.args, &stdout, &stderr)
			var retired *ComposeModeRetiredError
			if !errors.As(err, &retired) {
				t.Fatalf("error = %v, se esperaba un *ComposeModeRetiredError", err)
			}
			for _, flag := range tc.flags {
				if !strings.Contains(err.Error(), flag) {
					t.Errorf("el rechazo no nombra %s: %v", flag, err)
				}
			}
			for _, want := range []string{"axiom sdd archive-compose --canonical", "axiom archive context", "axiom archive adopt scan"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el rechazo no nombra la continuación %q: %v", want, err)
				}
			}
			if strings.Contains(err.Error(), "blocked(iso_") {
				t.Errorf("un modo retirado no pasa por la regla ISO: %v", err)
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Errorf("el rechazo escribió salida: %q %q", stdout.String(), stderr.String())
			}
		})
	}
	after := treeSnapshot(t, root)
	if len(after) != len(before) {
		t.Errorf("el rechazo creó o borró ficheros: antes %d, después %d", len(before), len(after))
	}
	for path, text := range before {
		if after[path] != text {
			t.Errorf("el rechazo modificó %s", path)
		}
	}
}
