package adoption

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseDiscards(t *testing.T) {
	text := "# Adopción\n\nTexto previo.\n\n## Descartado\n\n" +
		"- docs/requisitos.md#glosario: no es un requisito\n" +
		"- `docs/mis requisitos.md#intro`: es una introducción: sin requisitos\n" +
		"* docs/historias.docx: documento obsoleto\n" +
		"- docs/requisitos.md#vacio:\n" +
		"- docs/requisitos.md#sin-dos-puntos\n" +
		"Línea que no es una viñeta\n" +
		"\n## Otra sección\n\n- docs/otra.md#x: no cuenta\n"
	got := ParseDiscards(text)
	want := []Discard{
		{Path: "docs/requisitos.md", Anchor: "glosario", Reason: "no es un requisito", Line: 7},
		{Path: "docs/mis requisitos.md", Anchor: "intro", Reason: "es una introducción: sin requisitos", Line: 8},
		{Path: "docs/historias.docx", Anchor: "", Reason: "documento obsoleto", Line: 9},
		{Path: "docs/requisitos.md", Anchor: "vacio", Reason: "", Line: 10},
		{Path: "docs/requisitos.md", Anchor: "sin-dos-puntos", Reason: "", Line: 11},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("descartes = %+v\nquería %+v", got, want)
	}
}

func TestParseDiscards_SinSeccion(t *testing.T) {
	if got := ParseDiscards("# Adopción\n\n- docs/x.md#y: no cuenta\n"); len(got) != 0 {
		t.Fatalf("sin `## Descartado` no hay descartes: %+v", got)
	}
}

func TestParseDiscards_IgnoraVallas(t *testing.T) {
	text := "## Descartado\n\n```\n- docs/x.md#y: en una valla\n```\n- docs/z.md#w: fuera\n"
	got := ParseDiscards(text)
	if len(got) != 1 || got[0].Path != "docs/z.md" {
		t.Fatalf("descartes = %+v", got)
	}
}

func TestReadDiscards(t *testing.T) {
	dir := t.TempDir()
	if got, err := ReadDiscards(dir); err != nil || len(got) != 0 {
		t.Fatalf("sin adoption.md: %v, %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(dir, DiscardsFileName), []byte("## Descartado\n\n- a.md#b: c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadDiscards(dir)
	if err != nil || len(got) != 1 || got[0].Reason != "c" {
		t.Fatalf("descartes = %+v, err = %v", got, err)
	}
}

func TestParseOrigins(t *testing.T) {
	body := "### Requirement: Alta (REQ-40.1)\n\nEl sistema DEBE permitir el alta.\n\n" +
		"Origen: docs/requisitos.md#rf-001 (RF-001)\n" +
		"- Origen: `docs/mis requisitos.md#intro` (US-4)\n" +
		"Origen: docs/historias.docx\n" +
		"Origen: docs/requisitos.md#rf-002 ()\n" +
		"```\nOrigen: docs/en-valla.md#x (RF-9)\n```\n" +
		"Texto con Origen: en medio\n\n#### Scenario: x\n"
	got := ParseOrigins(body)
	want := []Origin{
		{Path: "docs/requisitos.md", Anchor: "rf-001", OriginalID: "RF-001", Line: 5},
		{Path: "docs/mis requisitos.md", Anchor: "intro", OriginalID: "US-4", Line: 6},
		{Path: "docs/historias.docx", Anchor: "", OriginalID: "", Line: 7},
		{Path: "docs/requisitos.md", Anchor: "rf-002", OriginalID: "", Line: 8},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orígenes = %+v\nquería %+v", got, want)
	}
}

func TestParseOrigins_VariasLineasCubrenVariasUnidades(t *testing.T) {
	body := "Origen: a.md#uno (RF-1)\nOrigen: a.md#dos (RF-2)\n"
	if got := ParseOrigins(body); len(got) != 2 || got[0].Anchor != "uno" || got[1].Anchor != "dos" {
		t.Fatalf("orígenes = %+v", got)
	}
}
