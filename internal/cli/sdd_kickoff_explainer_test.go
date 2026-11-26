package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Estas pruebas sustituyen kickoffSealWarnings, una variable global: no usan t.Parallel.

// captureExplainerWarnings redirige los avisos de seal a un búfer durante la prueba.
func captureExplainerWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var warnings bytes.Buffer
	previous := kickoffSealWarnings
	kickoffSealWarnings = &warnings
	t.Cleanup(func() { kickoffSealWarnings = previous })
	return &warnings
}

// explainerWorkspace crea un workspace con el bloque sdd: indicado (vacío: sin bloque).
func explainerWorkspace(t *testing.T, blocks string) string {
	t.Helper()
	root := t.TempDir()
	mustMkdirAllCLI(t, filepath.Join(root, "openspec", "changes"))
	writeSDDAxiomYAML(t, root, blocks)
	return root
}

func readSealedKickoff(t *testing.T, root, change string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "openspec", "changes", change, "kickoff.yaml"))
	if err != nil {
		t.Fatalf("leer kickoff.yaml: %v", err)
	}
	return string(data)
}

func TestRunSDDKickoffSealExplainerHTMLProjectValue(t *testing.T) {
	tests := []struct {
		name     string
		blocks   string
		flag     []string
		want     string // valor sellado esperado
		wantWarn bool
	}{
		{"always sin bandera sella yes", "sdd:\n  explainer_html: always\n", nil, "yes", false},
		{"always con yes", "sdd:\n  explainer_html: always\n", []string{"--explainer-html", "yes"}, "yes", false},
		{"never sin bandera sella no explícito", "sdd:\n  explainer_html: never\n", nil, "no", false},
		{"never con no", "sdd:\n  explainer_html: never\n", []string{"--explainer-html", "no"}, "no", false},
		{"ask con yes", "sdd:\n  explainer_html: ask\n", []string{"--explainer-html", "yes"}, "yes", false},
		{"ask con no", "", []string{"--explainer-html", "no"}, "no", false},
		{"ask sin bandera sella no y avisa", "", nil, "no", true},
		{"ask explícito sin bandera sella no y avisa", "sdd:\n  explainer_html: ask\n", nil, "no", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := captureExplainerWarnings(t)
			root := explainerWorkspace(t, tt.blocks)
			args := append(append([]string{"seal", "--cwd", root, "--change", "nuevo"}, sealPlain...), tt.flag...)
			var out bytes.Buffer
			if err := RunSDDKickoff(args, &out); err != nil {
				t.Fatalf("RunSDDKickoff() error = %v", err)
			}
			if got := readSealedKickoff(t, root, "nuevo"); !strings.Contains(got, `explainer_html: "`+tt.want+`"`) {
				t.Fatalf("kickoff.yaml debe llevar explainer_html %q:\n%s", tt.want, got)
			}
			if !strings.Contains(out.String(), "explainer_html="+tt.want) {
				t.Fatalf("el resumen debe mostrar explainer_html=%s:\n%s", tt.want, out.String())
			}
			hasWarning := strings.Contains(warnings.String(), "sdd.explainer_html es ask y no se ha preguntado")
			if hasWarning != tt.wantWarn {
				t.Fatalf("aviso = %v, se esperaba %v; stderr:\n%s", hasWarning, tt.wantWarn, warnings.String())
			}
			if tt.wantWarn && !strings.Contains(warnings.String(), "--explainer-html yes") {
				t.Fatalf("el aviso debe indicar cómo activarlo:\n%s", warnings.String())
			}
		})
	}
}

// Una contradicción o una declaración inválida se rechazan antes de crear nada: ni
// carpeta del cambio, ni kickoff.yaml, ni avisos.
func TestRunSDDKickoffSealExplainerHTMLRejectionsWriteNothing(t *testing.T) {
	tests := []struct {
		name   string
		blocks string
		flag   []string
		want   []string
	}{
		{"always contradice no", "sdd:\n  explainer_html: always\n", []string{"--explainer-html", "no"},
			[]string{"--explainer-html no contradice sdd.explainer_html: always de axiom.yaml; omite la bandera o cambia la declaración"}},
		{"never contradice yes", "sdd:\n  explainer_html: never\n", []string{"--explainer-html", "yes"},
			[]string{"--explainer-html yes contradice sdd.explainer_html: never de axiom.yaml; omite la bandera o cambia la declaración"}},
		{"declaración inválida", "sdd:\n  explainer_html: quizás\n", nil,
			[]string{"sdd.explainer_html", `"quizás"`, "usa ask, always o never"}},
		{"valor de bandera inválido", "", []string{"--explainer-html", "quizás"},
			[]string{"--explainer-html desconocido", "yes, no"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := captureExplainerWarnings(t)
			root := explainerWorkspace(t, tt.blocks)
			args := append(append([]string{"seal", "--cwd", root, "--change", "nuevo"}, sealPlain...), tt.flag...)
			err := RunSDDKickoff(args, &bytes.Buffer{})
			if err == nil {
				t.Fatal("se esperaba un rechazo")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe contener %q", err, want)
				}
			}
			if got := listDir(t, filepath.Join(root, "openspec", "changes")); len(got) != 0 {
				t.Errorf("un rechazo no debe crear la carpeta del cambio: %v", got)
			}
			if warnings.Len() != 0 {
				t.Errorf("un rechazo no emite avisos: %q", warnings.String())
			}
		})
	}
}

// La decisión es independiente de iso y del carril, y vale con checkpointed.
func TestRunSDDKickoffSealExplainerHTMLIsIndependentOfISOAndStyle(t *testing.T) {
	captureExplainerWarnings(t)
	root := explainerWorkspace(t, "sdd:\n  iso: true\n  explainer_html: always\n")
	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "nuevo", "--execution-style", "checkpointed"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSDDKickoff() error = %v", err)
	}
	got := readSealedKickoff(t, root, "nuevo")
	if !strings.Contains(got, `explainer_html: "yes"`) || !strings.Contains(got, "iso: true") {
		t.Fatalf("el sellado debe llevar yes y la instantánea ISO:\n%s", got)
	}
}

// El aviso solo se emite al escribir, y la carpeta nueva con yes se crea.
func TestRunSDDKickoffSealExplainerHTMLWarnsOnlyWhenWriting(t *testing.T) {
	warnings := captureExplainerWarnings(t)
	root := explainerWorkspace(t, "")
	seal := append([]string{"seal", "--cwd", root, "--change", "nuevo"}, sealPlain...)
	if err := RunSDDKickoff(seal, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warnings.String(), "se sella explainer_html: no") {
		t.Fatalf("la primera escritura debe avisar:\n%s", warnings.String())
	}
	warnings.Reset()

	var second bytes.Buffer
	if err := RunSDDKickoff(seal, &second); err != nil {
		t.Fatal(err)
	}
	if warnings.Len() != 0 || !strings.Contains(second.String(), "Kickoff ya estaba sellado") {
		t.Fatalf("un seal sobre un cambio sellado no escribe ni avisa:\n%q\n%s", warnings.String(), second.String())
	}
}

// Un seal sobre un cambio ya sellado con otro --explainer-html se comporta como hoy:
// devuelve el valor ganador sin reescribir el fichero.
func TestRunSDDKickoffSealExplainerHTMLOnSealedChangeKeepsTheWinner(t *testing.T) {
	captureExplainerWarnings(t)
	root := explainerWorkspace(t, "")
	base := append([]string{"seal", "--cwd", root, "--change", "nuevo"}, sealPlain...)
	if err := RunSDDKickoff(append(base, "--explainer-html", "no"), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	before := readSealedKickoff(t, root, "nuevo")

	var out bytes.Buffer
	if err := RunSDDKickoff(append(base, "--explainer-html", "yes"), &out); err != nil {
		t.Fatalf("el segundo seal se comporta como hoy: %v", err)
	}
	if after := readSealedKickoff(t, root, "nuevo"); after != before {
		t.Fatalf("el segundo seal reescribió kickoff.yaml:\n%s\n---\n%s", before, after)
	}
	if !strings.Contains(out.String(), "explainer_html=no") {
		t.Fatalf("debe mostrar el valor ganador:\n%s", out.String())
	}
}

// El JSON lleva explainerHtml solo cuando el fichero lo lleva.
func TestRunSDDKickoffSealExplainerHTMLJSON(t *testing.T) {
	warnings := captureExplainerWarnings(t)
	root := explainerWorkspace(t, "")
	var out bytes.Buffer
	args := append([]string{"seal", "--cwd", root, "--change", "nuevo", "--json", "--explainer-html", "yes"}, sealPlain...)
	if err := RunSDDKickoff(args, &out); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Wrote   bool `json:"wrote"`
		Kickoff struct {
			ExplainerHTML string `json:"explainerHtml"`
		} `json:"kickoff"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("el JSON de stdout debe seguir siendo válido: %v\n%s", err, out.String())
	}
	if !result.Wrote || result.Kickoff.ExplainerHTML != "yes" {
		t.Fatalf("resultado JSON inesperado: %+v\n%s", result, out.String())
	}
	if warnings.Len() != 0 {
		t.Fatalf("con la bandera no hay aviso: %q", warnings.String())
	}

	// ask sin bandera con --json: el aviso va a stderr y stdout conserva su forma.
	warnings.Reset()
	other := explainerWorkspace(t, "")
	out.Reset()
	if err := RunSDDKickoff(append([]string{"seal", "--cwd", other, "--change", "nuevo", "--json"}, sealPlain...), &out); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Kickoff.ExplainerHTML != "no" {
		t.Fatalf("JSON con ask sin bandera: %v %+v\n%s", err, result, out.String())
	}
	if !strings.Contains(warnings.String(), "sdd.explainer_html es ask") {
		t.Fatalf("el aviso se emite también con --json: %q", warnings.String())
	}
}

func TestRunSDDKickoffSealInferNeverWritesExplainerHTML(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	writeSDDAxiomYAML(t, root, "sdd:\n  explainer_html: always\n")
	writeCLIFixture(t, filepath.Join(changeRoot, "design.md"), "# Diseno\n")
	if err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "inc-99-example"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("seal --infer error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
	if err != nil || strings.Contains(string(data), "explainer_html") {
		t.Fatalf("el retrosellado no lleva explainer_html: %v\n%s", err, data)
	}
}

func TestRunSDDKickoffSealHelpMentionsExplainerHTML(t *testing.T) {
	var out bytes.Buffer
	if err := RunSDDKickoff([]string{"--help"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[--explainer-html yes|no]", "sdd.explainer_html"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("la ayuda debe mencionar %q:\n%s", want, out.String())
		}
	}
}
