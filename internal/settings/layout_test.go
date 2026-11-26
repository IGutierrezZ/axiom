package settings

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// reencode simula la serialización del escritor: decodifica a yaml.Node y vuelve a
// codificar con sangría de dos espacios, que es lo que pierde las líneas en blanco.
func reencode(t *testing.T, data []byte) []byte {
	t.Helper()
	var node yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, utf8BOM))).Decode(&node); err != nil {
		t.Fatalf("decodificar: %v", err)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		t.Fatalf("codificar: %v", err)
	}
	_ = enc.Close()
	return buf.Bytes()
}

// RF-TOP-22: la ida y vuelta de los axiom.yaml reales del repositorio, sin cambios,
// es idéntica byte a byte.
func TestLayoutRoundTripRealRepositoryFiles(t *testing.T) {
	for _, name := range []string{"axiom.yaml", "axiom.example.yaml"} {
		t.Run(name, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join("..", "..", name))
			if err != nil {
				t.Fatal(err)
			}
			got := restoreLayout(original, reencode(t, original))
			if !bytes.Equal(got, original) {
				t.Fatalf("la ida y vuelta de %s cambia el fichero:\n--- original\n%s\n--- resultado\n%s", name, original, got)
			}
		})
	}
}

// RF-TOP-22: las líneas repetidas (`- path: "."`) no confunden el alineamiento.
func TestLayoutRoundTripRepeatedLines(t *testing.T) {
	original := []byte("roles:\n  a:\n    repositories:\n      - path: \".\"\n\n  b:\n    repositories:\n      - path: \".\"\n\n  c:\n    repositories:\n      - path: \".\"\n")
	if got := restoreLayout(original, reencode(t, original)); !bytes.Equal(got, original) {
		t.Fatalf("resultado:\n%s", got)
	}
}

// RF-TOP-22: CRLF y BOM se conservan, y un parche sin efecto no altera ni un byte.
func TestLayoutRoundTripKeepsCRLFAndBOM(t *testing.T) {
	plain := "# cabecera\r\nworkspace:\r\n  name: \"x\"\r\n\r\ngit:\r\n  branch_prefix: \"inc\"\r\n"
	original := append(append([]byte{}, utf8BOM...), plain...)
	if got := restoreLayout(original, reencode(t, original)); !bytes.Equal(got, original) {
		t.Fatalf("CRLF/BOM no se conservan: %q", got)
	}
	changed := strings.Replace(string(reencode(t, original)), `"inc"`, `"feat"`, 1)
	want := append(append([]byte{}, utf8BOM...), strings.Replace(plain, `"inc"`, `"feat"`, 1)...)
	if got := restoreLayout(original, []byte(changed)); !bytes.Equal(got, want) {
		t.Fatalf("solo debe cambiar la línea editada:\n%q\n%q", got, want)
	}
}

// Hueco 4 de la revisión de cobertura: restoreLayout no duplica ni añade líneas en
// blanco dentro de un escalar literal `|` ni plegado `>`.
func TestLayoutDoesNotTouchBlankLinesInsideBlockScalars(t *testing.T) {
	original := []byte(`# cabecera

workspace:
  name: "x"
  note: |
    primera

    segunda


    tercera

other: >
  plegado uno

  plegado dos

roles:
  a:
    name: "A"
`)
	if got := restoreLayout(original, reencode(t, original)); !bytes.Equal(got, original) {
		t.Fatalf("el literal cambia:\n%s", got)
	}
	var before, after struct {
		Workspace struct{ Note string } `yaml:"workspace"`
		Other     string                `yaml:"other"`
	}
	if err := yaml.Unmarshal(original, &before); err != nil {
		t.Fatal(err)
	}
	// Cambio de una línea fuera de los literales: sus valores no se alteran.
	edited := bytes.Replace(reencode(t, original), []byte(`name: "A"`), []byte(`name: "B"`), 1)
	out := restoreLayout(original, edited)
	if err := yaml.Unmarshal(out, &after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("el valor de los escalares cambió: %q frente a %q", before, after)
	}
	if !bytes.Contains(out, []byte("    segunda\n\n\n    tercera\n")) {
		t.Fatalf("las líneas en blanco del literal no se conservan:\n%s", out)
	}
}

// RF-TOP-22: una clave de primer nivel nueva lleva una línea en blanco delante, y el
// borrado de un bloque no deja blancas dobles.
func TestLayoutBlankLineBeforeNewTopLevelKeyAndAfterDeletion(t *testing.T) {
	original := []byte("workspace:\n  name: \"x\"\n\nroles:\n  a:\n    name: \"A\"\n")
	encoded := []byte("workspace:\n  name: \"x\"\ngit:\n  branch_prefix: \"inc\"\nroles:\n  a:\n    name: \"A\"\n")
	want := "workspace:\n  name: \"x\"\n\ngit:\n  branch_prefix: \"inc\"\n\nroles:\n  a:\n    name: \"A\"\n"
	if got := restoreLayout(original, encoded); string(got) != want {
		t.Fatalf("alta de bloque:\n%s", got)
	}

	withBlock := []byte("workspace:\n  name: \"x\"\n\ndelivery:\n  split_hint_lines: 400\n\nroles:\n  a:\n    name: \"A\"\n")
	without := []byte("workspace:\n  name: \"x\"\nroles:\n  a:\n    name: \"A\"\n")
	if got := restoreLayout(withBlock, without); string(got) != string(original) {
		t.Fatalf("baja de bloque:\n%s", got)
	}
}
