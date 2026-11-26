package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const rfCanonicalSpec = "## Requirements\n\n### Requirement: Alta (RF-USU-01)\n\nEl sistema da de alta.\n\n### Requirement: Baja (RF-USU-02)\n\nEl sistema da de baja.\n"

const rfModifiedDelta = "## MODIFIED Requirements\n\n### Requirement: Alta masiva (RF-USU-01)\n\nEl sistema da de alta en lote.\n"

func wantValidateContinuation(t *testing.T, err error, wants ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("se esperaba una negativa")
	}
	for _, want := range append([]string{"axiom workspace validate --path"}, wants...) {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error %q debe contener %q", err, want)
		}
	}
}

// RF-RID-08: con el patrón inaccesible `archive integrate` se niega sin escribir y con la
// continuación `axiom workspace validate --path <proyecto>`; un `sdd.iso` roto también.
func TestArchiveIntegrateRefusesAnInaccessiblePattern(t *testing.T) {
	for name, block := range map[string]string{
		"patrón inválido": "sdd:\n  id_pattern: \"<m><m>\"\n",
		"iso inválido":    "sdd:\n  iso: quizás\n",
	} {
		t.Run(name, func(t *testing.T) {
			f := newODDFixture(t, true)
			workspacetest.WriteFile(t, filepath.Join(f.root, "axiom.yaml"), readODDFile(t, filepath.Join(f.root, "axiom.yaml"))+block)
			f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
			before := oddSnapshot(t, f.root)

			_, err := f.run(t)
			wantValidateContinuation(t, err, "sdd.id_pattern")
			if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
				t.Error("la negativa no escribe nada")
			}
		})
	}
}

// RF-RID-07 y RF-OLS-04: un proyecto con `RF-<DOM>-<NN>` integra un ADDED con su identificador y
// rechaza uno con el formato por defecto.
func TestArchiveIntegrateUsesTheDeclaredPattern(t *testing.T) {
	f := newODDFixture(t, true)
	workspacetest.WriteFile(t, filepath.Join(f.root, "axiom.yaml"), readODDFile(t, filepath.Join(f.root, "axiom.yaml"))+"sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	rfDelta := strings.Replace(oddAddedDelta, "Nota de abono (REQ-31.5)", "Nota de abono (RF-FAC-02)", 1)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"usuarios", rfDelta})))
	if _, err := f.run(t); err != nil {
		t.Fatalf("un ADDED con RF-FAC-02 debe integrarse: %v", err)
	}
	if got := readODDFile(t, f.specPath("usuarios")); !strings.Contains(got, "Nota de abono (RF-FAC-02)") {
		t.Errorf("la spec debe llevar el requisito:\n%s", got)
	}

	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"otra", oddAddedDelta})))
	before := oddSnapshot(t, f.root)
	_, err := f.run(t)
	if err == nil || !strings.Contains(err.Error(), "RF-<DOM>-<NN>") {
		t.Fatalf("un ADDED con REQ-31.5 debe rechazarse nombrando el patrón: %v", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("el rechazo no escribe nada")
	}
}

// RF-RID-08: `sdd-archive-compose` se niega sin escribir con el patrón inaccesible.
func TestSDDArchiveComposeRefusesAnInaccessiblePatternWithoutWriting(t *testing.T) {
	root := t.TempDir()
	writeSDDAxiomYAML(t, root, "sdd:\n  id_pattern: \"<m><m>\"\n")
	canonical, delta, out := filepath.Join(root, "spec.md"), filepath.Join(root, "delta.md"), filepath.Join(root, "out.md")
	writeCLIFixture(t, canonical, rfCanonicalSpec)
	writeCLIFixture(t, delta, rfModifiedDelta)

	for _, args := range [][]string{
		{"--canonical", canonical, "--delta", delta, "--output", out},
		{"--cwd", root, "--canonical", "spec.md", "--delta", "delta.md", "--output", "out.md"},
	} {
		var buf bytes.Buffer
		err := runSDDArchiveCompose(args, &buf)
		wantValidateContinuation(t, err, "sdd.id_pattern")
		if _, statErr := os.Stat(out); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("la negativa no escribe %s (%v)", out, statErr)
		}
	}
}

// RF-ARC-11 y RF-RID-02: con workspace, el compositor usa el patrón del workspace de `--cwd` o
// del que contiene la delta; sin workspace usa el defecto.
func TestSDDArchiveComposeUsesTheWorkspacePattern(t *testing.T) {
	root := t.TempDir()
	writeSDDAxiomYAML(t, root, "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	canonical, delta := filepath.Join(root, "spec.md"), filepath.Join(root, "delta.md")
	writeCLIFixture(t, canonical, rfCanonicalSpec)
	writeCLIFixture(t, delta, rfModifiedDelta)

	for name, args := range map[string][]string{
		"con --cwd":             {"--cwd", root, "--canonical", "spec.md", "--delta", "delta.md"},
		"workspace de la delta": {"--canonical", canonical, "--delta", delta},
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := runSDDArchiveCompose(args, &buf); err != nil {
				t.Fatalf("el MODIFIED por ID con título cambiado debe componerse: %v", err)
			}
			if !strings.Contains(buf.String(), "Alta masiva (RF-USU-01)") || strings.Contains(buf.String(), "### Requirement: Alta (RF-USU-01)") {
				t.Errorf("salida inesperada:\n%s", buf.String())
			}
		})
	}

	// Sin workspace rige el defecto: RF-USU-01 no es un identificador y el título cambiado no se
	// resuelve.
	bare := t.TempDir()
	bareCanonical, bareDelta := filepath.Join(bare, "spec.md"), filepath.Join(bare, "delta.md")
	writeCLIFixture(t, bareCanonical, rfCanonicalSpec)
	writeCLIFixture(t, bareDelta, rfModifiedDelta)
	var buf bytes.Buffer
	if err := runSDDArchiveCompose([]string{"--canonical", bareCanonical, "--delta", bareDelta}, &buf); err == nil {
		t.Fatal("sin workspace rige el patrón por defecto y el título cambiado no se resuelve")
	}
}

// RF-ADP-01, RF-ADP-03 y RF-RID-04: la ayuda de `archive adopt` describe el patrón declarado, la
// conservación del ID del cliente y la propuesta del escaneo, sin literal de forma fijo.
func TestArchiveAdoptHelpDescribesTheDeclaredPattern(t *testing.T) {
	var out bytes.Buffer
	if err := RunArchiveAdopt([]string{"--help"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"un identificador del patrón declarado (sdd.id_pattern; sin declaración, (REQ-n.m))",
		"o un provisional (prov:<clave>:<ordinal>)",
		"el ID del cliente conservado (origin_id)", "el patrón declarado (pattern_undeclared)",
		"propone el patrón de identificadores (sdd.id_pattern), sin escribir axiom.yaml",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("la ayuda debe contener %q:\n%s", want, out.String())
		}
	}
	for _, gone := range []string{"de una serie nueva", "C-03-c"} {
		if strings.Contains(out.String(), gone) {
			t.Errorf("la ayuda ya no debe contener %q:\n%s", gone, out.String())
		}
	}

	var integrate bytes.Buffer
	if err := RunArchiveIntegrate([]string{"--help"}, &integrate); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(integrate.String(), "Un `### Requirement:` ADDED lleva su `(REQ-n.m)` en el título; si el identificador ya existe") {
		t.Errorf("la ayuda de integrate debe conservar la frase de siempre:\n%s", integrate.String())
	}
}
