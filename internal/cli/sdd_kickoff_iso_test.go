package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// REQ-29.12 y REQ-29.13: `seal` prepara la carpeta del cambio tras todas las
// validaciones y copia `sdd.iso` a la instantánea. Los nombres de las pruebas
// empiezan por TestRunSDDKickoffSeal para entrar en el mismo patrón de `-run`.

// sealPlain son los argumentos mínimos de un sellado válido.
var sealPlain = []string{"--execution-style", "continuous", "--handoff-policy", "none"}

func writeSDDAxiomYAML(t *testing.T, root, blocks string) {
	t.Helper()
	writeCLIFixture(t, filepath.Join(root, "axiom.yaml"), "workspace:\n"+
		"  name: test-workspace\n"+
		"  topology: monorepo-embedded\n"+
		"roles:\n"+
		"  core:\n"+
		"    name: core\n"+
		"    repositories:\n"+
		"      - path: .\n"+blocks)
}

// listDir devuelve los nombres de las entradas de dir, o nil si no existe.
func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// REQ-29.12: un cambio sin carpeta se sella creándola.
func TestRunSDDKickoffSealOnNonexistentChangeCreatesTheFolder(t *testing.T) {
	root := t.TempDir()
	mustMkdirAllCLI(t, filepath.Join(root, "openspec", "changes"))
	changeRoot := filepath.Join(root, "openspec", "changes", "does-not-exist")

	var out bytes.Buffer
	if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "does-not-exist"}, sealPlain...), &out); err != nil {
		t.Fatalf("RunSDDKickoff() error = %v, un cambio sin carpeta se sella creándola", err)
	}
	if _, err := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); err != nil {
		t.Fatalf("kickoff.yaml no se escribió en la carpeta creada: %v", err)
	}
	if strings.Contains(out.String(), "iso=") {
		t.Errorf("un cambio no ISO no menciona iso en el resumen:\n%s", out.String())
	}
}

// REQ-29.12: la carpeta existente queda intacta y un segundo sellado no reescribe nada.
func TestRunSDDKickoffSealKeepsAnExistingFolderAndIsIdempotent(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	writeCLIFixture(t, filepath.Join(changeRoot, "proposal.md"), "# propuesta\n")

	if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "inc-99-example"}, sealPlain...), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var second bytes.Buffer
	if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--json"}, sealPlain...), &second); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
	if string(before) != string(after) || !strings.Contains(second.String(), `"wrote": false`) {
		t.Fatalf("el segundo sellado no reescribe y lo dice:\n%s", second.String())
	}
	if got := listDir(t, changeRoot); len(got) != 2 {
		t.Errorf("la carpeta solo lleva proposal.md y kickoff.yaml: %v", got)
	}
	if data, _ := os.ReadFile(filepath.Join(changeRoot, "proposal.md")); string(data) != "# propuesta\n" {
		t.Errorf("proposal.md debe quedar intacto: %q", data)
	}
}

// REQ-29.12 (amenaza 9.1): un nombre de cambio peligroso se rechaza sin crear nada,
// ni dentro ni fuera de openspec/changes/.
func TestRunSDDKickoffSealRejectsUnsafeChangeNamesWithoutCreatingAnything(t *testing.T) {
	for _, name := range []string{"../x", "archive", "a:(b", "a*b", "a..b", "..", "x/y", `x\y`, "a b", ".oculto", "x.lock", "con"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			changes := filepath.Join(root, "openspec", "changes")
			mustMkdirAllCLI(t, changes)

			err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", name}, sealPlain...), &bytes.Buffer{})
			if err == nil {
				t.Fatalf("--change %q debe rechazarse", name)
			}
			if got := listDir(t, changes); len(got) != 0 {
				t.Errorf("el rechazo no crea nada en openspec/changes/: %v", got)
			}
			if got := listDir(t, filepath.Join(root, "openspec")); len(got) != 1 {
				t.Errorf("el rechazo no crea nada en openspec/: %v", got)
			}
			if got := listDir(t, root); len(got) != 1 {
				t.Errorf("el rechazo no crea nada en la raíz: %v", got)
			}
		})
	}
}

// REQ-29.12: wrong_branch no deja carpeta ni kickoff.yaml aunque el cambio no exista.
func TestRunSDDKickoffSealWrongBranchLeavesNoFolder(t *testing.T) {
	captureSealWarnings(t)
	isolateIntegrateGit(t)
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "README.md"), "# base\n")
	integrateInitRepo(t, ws.Master)
	integrateCommit(t, ws.Master, "base")
	changeRoot := filepath.Join(ws.Home, "changes", sealChange)

	_, err := runSeal(t, ws.Master, sealPlain...)
	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
	if _, statErr := os.Stat(changeRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("wrong_branch no debe dejar la carpeta del cambio (%v)", statErr)
	}
}

// REQ-29.1 y REQ-29.12: un `sdd:` inválido o contradictorio con workflow.lane se
// rechaza sin carpeta ni kickoff.yaml, con la carpeta ausente y con ella presente.
func TestRunSDDKickoffSealRejectsAnInvalidOrContradictorySDDDeclaration(t *testing.T) {
	cases := map[string]struct {
		blocks string
		wants  []string
	}{
		"iso no booleano":   {"sdd:\n  iso: quizás\n", []string{"sdd.iso"}},
		"clave desconocida": {"sdd:\n  archive: true\n", []string{"archive"}},
		"contradicción":     {"sdd:\n  iso: true\nworkflow:\n  lane: odd\n", []string{"sdd.iso", "workflow.lane"}},
	}
	for name, tc := range cases {
		t.Run(name+"/sin carpeta", func(t *testing.T) {
			root := t.TempDir()
			mustMkdirAllCLI(t, filepath.Join(root, "openspec", "changes"))
			writeSDDAxiomYAML(t, root, tc.blocks)

			err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "nuevo"}, sealPlain...), &bytes.Buffer{})
			if err == nil {
				t.Fatal("la declaración sdd: inválida debe rechazar el sellado")
			}
			for _, want := range tc.wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe contener %q", err, want)
				}
			}
			if got := listDir(t, filepath.Join(root, "openspec", "changes")); len(got) != 0 {
				t.Errorf("no debe crearse la carpeta del cambio: %v", got)
			}
		})
		t.Run(name+"/con carpeta", func(t *testing.T) {
			root, changeRoot := newGovernanceWorkspace(t, "existente")
			writeSDDAxiomYAML(t, root, tc.blocks)

			if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "existente"}, sealPlain...), &bytes.Buffer{}); err == nil {
				t.Fatal("la declaración sdd: inválida debe rechazar el sellado")
			}
			if _, statErr := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("no debe escribirse kickoff.yaml (%v)", statErr)
			}
		})
	}
}

// REQ-29.13: con sdd.iso true el sellado copia la instantánea una sola vez y el
// resumen, en texto y JSON, lo dice; con iso false o sin bloque no aparece nada.
func TestRunSDDKickoffSealCopiesSDDISOToTheSnapshot(t *testing.T) {
	t.Run("iso true crea la carpeta y copia la instantánea", func(t *testing.T) {
		root := t.TempDir()
		mustMkdirAllCLI(t, filepath.Join(root, "openspec", "changes"))
		writeSDDAxiomYAML(t, root, "sdd:\n  iso: true\n")

		var text bytes.Buffer
		if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "cobro-iva"}, sealPlain...), &text); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(text.String(), "iso=true") {
			t.Errorf("el resumen de texto debe llevar iso=true:\n%s", text.String())
		}
		sealed, err := kickoff.Load(filepath.Join(root, "openspec", "changes", "cobro-iva"))
		if err != nil || sealed == nil || !sealed.IsISO() {
			t.Fatalf("kickoff.yaml debe llevar la instantánea ISO: %+v, %v", sealed, err)
		}

		var asJSON bytes.Buffer
		if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "cobro-iva", "--json"}, sealPlain...), &asJSON); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(asJSON.String(), `"iso": true`) {
			t.Errorf("el JSON debe llevar iso: true:\n%s", asJSON.String())
		}
	})

	for name, blocks := range map[string]string{"iso false": "sdd:\n  iso: false\n", "sin bloque": ""} {
		t.Run(name+" no escribe sdd ni iso", func(t *testing.T) {
			root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
			writeSDDAxiomYAML(t, root, blocks)

			var out bytes.Buffer
			if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--json"}, sealPlain...), &out); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			for _, unwanted := range []string{"sdd:", "iso"} {
				if strings.Contains(string(data), unwanted) || strings.Contains(out.String(), unwanted) {
					t.Errorf("no debe aparecer %q:\n%s\n%s", unwanted, data, out.String())
				}
			}
		})
	}
}

// REQ-29.12: si el sellado falla tras crear la carpeta, se retira lo creado; una
// carpeta que ya existía no se borra.
func TestRunSDDKickoffSealRemovesTheCreatedFolderWhenSealFails(t *testing.T) {
	failing := func(string, kickoff.Kickoff) (kickoff.Kickoff, bool, error) {
		return kickoff.Kickoff{}, false, errors.New("fallo simulado al publicar")
	}
	previous := sealKickoffFile
	sealKickoffFile = failing
	t.Cleanup(func() { sealKickoffFile = previous })

	t.Run("carpeta creada", func(t *testing.T) {
		root := t.TempDir()
		changes := filepath.Join(root, "openspec", "changes")
		mustMkdirAllCLI(t, changes)

		err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "nuevo"}, sealPlain...), &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "fallo simulado") {
			t.Fatalf("error = %v, se esperaba el fallo del sellado", err)
		}
		if got := listDir(t, changes); len(got) != 0 {
			t.Errorf("la retirada debe borrar la carpeta creada: %v", got)
		}
	})
	t.Run("carpeta preexistente", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "existente")
		writeCLIFixture(t, filepath.Join(changeRoot, "proposal.md"), "# p\n")

		if err := RunSDDKickoff(append([]string{"seal", "--cwd", root, "--change", "existente"}, sealPlain...), &bytes.Buffer{}); err == nil {
			t.Fatal("se esperaba el fallo del sellado")
		}
		if got := listDir(t, changeRoot); len(got) != 1 || got[0] != "proposal.md" {
			t.Errorf("una carpeta preexistente queda intacta: %v", got)
		}
	})
}

// REQ-29.13 y REQ-29.12: `--infer` sigue exigiendo una carpeta existente y nunca
// copia `iso`, aunque el proyecto lo declare.
func TestRunSDDKickoffSealInferNeverCreatesTheFolderNorCopiesISO(t *testing.T) {
	t.Run("sin carpeta", func(t *testing.T) {
		root := t.TempDir()
		mustMkdirAllCLI(t, filepath.Join(root, "openspec", "changes"))
		writeSDDAxiomYAML(t, root, "sdd:\n  iso: true\n")

		err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "nuevo"}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "nuevo") {
			t.Fatalf("--infer sin carpeta debe rechazarse nombrando el cambio: %v", err)
		}
		if got := listDir(t, filepath.Join(root, "openspec", "changes")); len(got) != 0 {
			t.Errorf("--infer no crea la carpeta: %v", got)
		}
	})
	t.Run("con carpeta", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		writeSDDAxiomYAML(t, root, "sdd:\n  iso: true\n")
		writeCLIFixture(t, filepath.Join(changeRoot, "design.md"), "# Diseno\n")

		if err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "inc-99-example"}, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		sealed, err := kickoff.Load(changeRoot)
		if err != nil || sealed == nil || sealed.SDD != nil || sealed.IsISO() {
			t.Fatalf("el retrosellado no copia la instantánea ISO: %+v, %v", sealed, err)
		}
	})
}
