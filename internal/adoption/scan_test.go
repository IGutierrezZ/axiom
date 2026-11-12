package adoption

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeTree escribe ficheros (ruta con `/` → contenido) bajo root.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func sourcePaths(inv Inventory) []string {
	paths := make([]string, 0, len(inv.Sources))
	for _, s := range inv.Sources {
		paths = append(paths, s.Path)
	}
	return paths
}

func mustScan(t *testing.T, root string) Inventory {
	t.Helper()
	inv, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return inv
}

const requisitosDoc = "# Requisitos\n\nIntroducción.\n\n## RF-001 Alta de usuario\n\nEl sistema permite el alta.\n\n## RF-002 Baja de usuario\n\nEl sistema permite la baja.\n"

// Amenaza «rutas tipo documentación» (REQ-30.9): una prueba por clase.
func TestScan_RutasQueNoSonOrigen(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"README.sh":           "#!/bin/sh\n# Requisitos\n# RF-001\n",
		"CMakeLists.txt":      "project(requisitos)\n# RF-002\n",
		"requirements.txt":    "flask==2.0\n",
		"docs/guia.mdx":       "# RF-003 Requisito\n",
		"docs/requisitos.mdx": "# RF-004 Requisito\n",
		"src/main.go":         "package main // RF-005\n",
	})
	inv := mustScan(t, root)
	if len(inv.Sources) != 0 {
		t.Fatalf("no debería haber orígenes, hay %v", sourcePaths(inv))
	}
}

func TestScan_EnlaceSimbolicoQueSaleDelRootNoSeSigue(t *testing.T) {
	outside := t.TempDir()
	writeTree(t, outside, map[string]string{"requisitos.md": requisitosDoc, "ext/prd.md": requisitosDoc})
	root := t.TempDir()
	writeTree(t, root, map[string]string{"docs/local.md": "# Nota\n"})
	if err := os.Symlink(filepath.Join(outside, "requisitos.md"), filepath.Join(root, "docs", "requisitos.md")); err != nil {
		t.Skipf("la plataforma no permite crear enlaces simbólicos: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "ext"), filepath.Join(root, "ext")); err != nil {
		t.Skipf("la plataforma no permite crear enlaces simbólicos: %v", err)
	}
	inv := mustScan(t, root)
	if len(inv.Sources) != 0 {
		t.Fatalf("un enlace que sale de workspaceRoot no se sigue; hay orígenes %v", sourcePaths(inv))
	}
}

func TestScan_FicheroMayorDe2MiBVaAWarnings(t *testing.T) {
	root := t.TempDir()
	big := "# RF-001 Grande\n" + strings.Repeat("x", 2<<20)
	writeTree(t, root, map[string]string{"docs/requisitos.md": big, "docs/prd.md": requisitosDoc})
	inv := mustScan(t, root)
	if got := sourcePaths(inv); !reflect.DeepEqual(got, []string{"docs/prd.md"}) {
		t.Fatalf("orígenes = %v, quería solo docs/prd.md", got)
	}
	if len(inv.Warnings) != 1 || !strings.Contains(inv.Warnings[0], "docs/requisitos.md") || !strings.Contains(inv.Warnings[0], "2 MiB") {
		t.Fatalf("avisos = %v, quería uno que nombre docs/requisitos.md y 2 MiB", inv.Warnings)
	}
}

func TestScan_MasDe2000FicherosVaAWarnings(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{}
	for i := 0; i < maxFiles+5; i++ {
		files[fmt.Sprintf("docs/doc%04d.md", i)] = "# RF-001 x\n"
	}
	writeTree(t, root, files)
	inv := mustScan(t, root)
	if len(inv.Sources) != maxFiles {
		t.Fatalf("orígenes = %d, quería %d", len(inv.Sources), maxFiles)
	}
	if len(inv.Warnings) == 0 || !strings.Contains(inv.Warnings[len(inv.Warnings)-1], "2000") {
		t.Fatalf("avisos = %v, quería uno con el límite de 2000 ficheros", inv.Warnings)
	}
}

func TestScan_DirectoriosExcluidos(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"docs/requisitos.md": requisitosDoc}
	for _, dir := range []string{"openspec", ".openspec", "odd", ".git", "node_modules", "vendor", "testdata", ".oculto", "docs/vendor", "a/node_modules/b"} {
		files[dir+"/requisitos.md"] = requisitosDoc
	}
	writeTree(t, root, files)
	inv := mustScan(t, root)
	if got := sourcePaths(inv); !reflect.DeepEqual(got, []string{"docs/requisitos.md"}) {
		t.Fatalf("orígenes = %v, quería solo docs/requisitos.md", got)
	}
}

func TestScan_UnidadesYIdentificadoresAjenos(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"docs/requisitos.md": requisitosDoc})
	inv := mustScan(t, root)
	if inv.Schema != InventorySchema {
		t.Fatalf("schema = %q", inv.Schema)
	}
	if len(inv.Sources) != 1 {
		t.Fatalf("orígenes = %v", sourcePaths(inv))
	}
	src := inv.Sources[0]
	if src.Path != "docs/requisitos.md" || src.Kind != KindMarkdown || len(src.SHA256) != 64 {
		t.Fatalf("origen inesperado: %+v", src)
	}
	want := []Unit{
		{Anchor: "requisitos", Title: "Requisitos", Line: 1},
		{Anchor: "rf-001-alta-de-usuario", Title: "RF-001 Alta de usuario", Line: 5, ForeignIDs: []string{"RF-001"}},
		{Anchor: "rf-002-baja-de-usuario", Title: "RF-002 Baja de usuario", Line: 9, ForeignIDs: []string{"RF-002"}},
	}
	if !reflect.DeepEqual(src.Units, want) {
		t.Fatalf("unidades = %+v\nquería %+v", src.Units, want)
	}
}

func TestScan_OrigenPorNombreOPorIdentificador(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/Especificación.md":  "# Cosas\n",
		"docs/historias.txt":      "# Historia\n",
		"docs/notas.md":           "# Notas\n\nVer US-12 y NFR-PER-3.\n",
		"docs/otro.md":            "# Otro\n\nNada que ver.\n",
		"docs/Requisitos.docx":    "binario",
		"docs/informe-prd.pdf":    "binario",
		"docs/hoja.xlsx":          "binario sin patrón",
		"docs/use-cases.markdown": "# Casos\n",
	})
	inv := mustScan(t, root)
	want := []string{"docs/Especificación.md", "docs/Requisitos.docx", "docs/historias.txt", "docs/informe-prd.pdf", "docs/notas.md", "docs/use-cases.markdown"}
	if got := sourcePaths(inv); !reflect.DeepEqual(got, want) {
		t.Fatalf("orígenes = %v\nquería %v", got, want)
	}
	for _, s := range inv.Sources {
		if strings.HasSuffix(s.Path, ".docx") || strings.HasSuffix(s.Path, ".pdf") {
			if s.Kind != KindFile || len(s.Units) != 1 || s.Units[0].Anchor != "" {
				t.Errorf("%s debe ser una unidad de fichero entero: %+v", s.Path, s)
			}
		}
	}
}

func TestScan_IgnoraIdentificadoresDeOpenSpec(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/notas.md": "# Notas\n\nVéase REQ-30.9 y REQ-4.12.\n",
		"docs/real.md":  "# Real\n\nVéase REQ-12 y RF-AB-0034, REQ_7.\n",
	})
	inv := mustScan(t, root)
	if got := sourcePaths(inv); !reflect.DeepEqual(got, []string{"docs/real.md"}) {
		t.Fatalf("orígenes = %v, quería solo docs/real.md", got)
	}
	if ids := inv.Sources[0].Units[0].ForeignIDs; !reflect.DeepEqual(ids, []string{"REQ-12", "REQ_7", "RF-AB-0034"}) {
		t.Fatalf("identificadores = %v", ids)
	}
}

func TestScan_EncabezadosFueraDeVallasYSinEncabezados(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/requisitos.md":  "# Real\n\n```md\n# No es un encabezado\n```\n\n#### Demasiado profundo\n\n### Sí\n",
		"docs/requisitos2.md": "Solo texto con RF-9.\n",
	})
	inv := mustScan(t, root)
	if len(inv.Sources) != 2 {
		t.Fatalf("orígenes = %v", sourcePaths(inv))
	}
	var anchors []string
	for _, u := range inv.Sources[0].Units {
		anchors = append(anchors, u.Anchor)
	}
	if !reflect.DeepEqual(anchors, []string{"real", "sí"}) {
		t.Fatalf("anclas = %v", anchors)
	}
	whole := inv.Sources[1].Units
	if len(whole) != 1 || whole[0].Anchor != "" || !reflect.DeepEqual(whole[0].ForeignIDs, []string{"RF-9"}) {
		t.Fatalf("un fichero sin encabezados es una unidad de fichero entero: %+v", whole)
	}
}

func TestScan_CandidatosAContextoTecnico(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"ARCHITECTURE.md":               "# Arquitectura\n",
		"docs/arquitectura-general.md":  "# A\n",
		"docs/architecture/overview.md": "# O\n",
		"docs/design-notes.md":          "# D\n",
		"docs/adr/0001-x.md":            "# ADR\n",
		"adr/0002-y.md":                 "# ADR\n",
		"docs/decisions/d1.md":          "# D\n",
		"docs/otra.md":                  "# Otra\n",
	})
	inv := mustScan(t, root)
	want := []ContextCandidate{
		{Path: "ARCHITECTURE.md", Mode: ModeMaintained, Reason: "documento de arquitectura"},
		{Path: "adr/", Mode: ModeReference, Reason: "registro de decisiones de arquitectura"},
		{Path: "docs/adr/", Mode: ModeReference, Reason: "registro de decisiones de arquitectura"},
		{Path: "docs/architecture/", Mode: ModeMaintained, Reason: "documentación de arquitectura o diseño"},
		{Path: "docs/arquitectura-general.md", Mode: ModeMaintained, Reason: "documentación de arquitectura o diseño"},
		{Path: "docs/decisions/", Mode: ModeReference, Reason: "registro de decisiones de arquitectura"},
		{Path: "docs/design-notes.md", Mode: ModeMaintained, Reason: "documentación de arquitectura o diseño"},
	}
	if !reflect.DeepEqual(inv.ContextCandidates, want) {
		t.Fatalf("candidatos = %+v\nquería %+v", inv.ContextCandidates, want)
	}
}

func TestScan_EsDeterministaYNoEscribe(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/requisitos.md": requisitosDoc,
		"docs/b.md":          "# Glosario\n\n# Glosario\n\nRF-5\n",
		"ARCHITECTURE.md":    "# A\n",
	})
	before := snapshotTree(t, root)
	a, b := mustScan(t, root), mustScan(t, root)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("dos pasadas distintas:\n%+v\n%+v", a, b)
	}
	if after := snapshotTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("Scan escribió en el árbol:\nantes %v\ndespués %v", before, after)
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			out[rel] = "<dir>"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestScan_DuplicadosRecibenSufijoEnLasAnclas(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"docs/b.md": "# Glosario\n\n# Glosario\n\nRF-5\n"})
	inv := mustScan(t, root)
	if len(inv.Sources) != 1 {
		t.Fatalf("orígenes = %v", sourcePaths(inv))
	}
	units := inv.Sources[0].Units
	if len(units) != 2 || units[0].Anchor != "glosario" || units[1].Anchor != "glosario-1" {
		t.Fatalf("unidades = %+v", units)
	}
}

func TestScan_NoLeeOpenspec(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"openspec/specs/x/spec.md":         "# RF-001\n",
		"openspec/changes/y/requisitos.md": "# RF-002\n",
	})
	inv := mustScan(t, root)
	if len(inv.Sources) != 0 || len(inv.ContextCandidates) != 0 {
		t.Fatalf("no debería ver nada: %+v", inv)
	}
}

func TestScan_RaizInexistente(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "no-existe")); err == nil {
		t.Fatal("esperaba un error para una raíz inexistente")
	}
}

func TestSummarize(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/requisitos.md": requisitosDoc,
		"ARCHITECTURE.md":    "# A\n",
	})
	sum := Summarize(mustScan(t, root))
	if sum.SourceCount != 1 || sum.UnitCount != 3 || len(sum.Sources) != 1 || sum.Sources[0].Path != "docs/requisitos.md" || sum.Sources[0].Units != 3 {
		t.Fatalf("resumen = %+v", sum)
	}
	if len(sum.ContextCandidates) != 1 || sum.ContextCandidates[0].Path != "ARCHITECTURE.md" {
		t.Fatalf("candidatos = %+v", sum.ContextCandidates)
	}
	if empty := Summarize(Inventory{}); empty.SourceCount != 0 || empty.UnitCount != 0 || empty.Sources == nil || empty.ContextCandidates == nil {
		t.Fatalf("un inventario vacío da un resumen vacío con listas no nulas: %+v", empty)
	}
}
