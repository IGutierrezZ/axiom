package specbaseline

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const contextDeltaExample = `# Delta de contexto: inc-99-ejemplo

## Entrada: context/ARQUITECTURA.md

- Ancla: Componentes > Servicio de pagos
- Acción: modify
- Motivo: se añade el reintento con retroceso exponencial (RF-9001)

Texto que integrar en la sección.

Segundo párrafo.

## Entrada: ` + "`context\\PAGOS.md`" + `

- Ancla: Reintentos #2
- Acción: add
- Motivo: documento nuevo del servicio

## Entrada: ../repo-backend/docs/API.md

- Ancla: Errores
- Acción: remove
- Motivo: el endpoint deja de existir
`

func TestParseContextDeltaWithEntries(t *testing.T) {
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		delta, err := ParseContextDelta(v.apply(contextDeltaExample))
		if err != nil {
			t.Fatalf("ParseContextDelta: %v", err)
		}
		if delta.NoImpact || len(delta.Entries) != 3 {
			t.Fatalf("delta = %+v, quería 3 entradas y sin «Sin impacto»", delta)
		}
		first, second, third := delta.Entries[0], delta.Entries[1], delta.Entries[2]
		if first.ID != "C1" || second.ID != "C2" || third.ID != "C3" {
			t.Errorf("identificadores %q, %q, %q; quería C1..C3 por orden de aparición", first.ID, second.ID, third.ID)
		}
		if first.Document != "context/ARQUITECTURA.md" || first.Action != ContextModify || first.Anchor.String() != "Componentes > Servicio de pagos" {
			t.Errorf("primera entrada = %+v", first)
		}
		if first.Reason != "se añade el reintento con retroceso exponencial (RF-9001)" {
			t.Errorf("motivo = %q", first.Reason)
		}
		if want := "Texto que integrar en la sección.\n\nSegundo párrafo."; first.Body != want {
			t.Errorf("cuerpo = %q, quería %q (sin EOL ajeno ni blancos en los extremos)", first.Body, want)
		}
		if first.Line != 3 {
			t.Errorf("línea = %d, quería 3", first.Line)
		}
		if second.Document != "context/PAGOS.md" || second.Anchor.Ordinal != 2 || second.Action != ContextAdd || second.Body != "" {
			t.Errorf("segunda entrada (comillas invertidas, `\\` y ordinal) = %+v", second)
		}
		if third.Document != "../repo-backend/docs/API.md" || third.Action != ContextRemove {
			t.Errorf("tercera entrada (repositorio hermano) = %+v", third)
		}
	})
}

func TestParseContextDeltaNoImpact(t *testing.T) {
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		delta, err := ParseContextDelta(v.apply("# Delta de contexto: inc-99-ejemplo\n\nSin impacto: el cambio solo corrige una errata en un mensaje de error\n"))
		if err != nil {
			t.Fatalf("ParseContextDelta: %v", err)
		}
		if !delta.NoImpact || len(delta.Entries) != 0 || delta.Reason != "el cambio solo corrige una errata en un mensaje de error" {
			t.Fatalf("delta = %+v", delta)
		}
	})
	if delta, err := ParseContextDelta([]byte("Sin impacto: sin encabezado inicial")); err != nil || !delta.NoImpact {
		t.Fatalf("el encabezado `# ...` es opcional: %+v %v", delta, err)
	}
}

func TestParseContextDeltaFencesAreBodyNotStructure(t *testing.T) {
	data := "## Entrada: context/A.md\n\n- Ancla: Uno\n- Acción: add\n- Motivo: ejemplo\n\n```markdown\n## Entrada: context/Falsa.md\n- Campo: no es un campo\nSin impacto: tampoco\n```\n\nCierre.\n"
	delta, err := ParseContextDelta([]byte(data))
	if err != nil {
		t.Fatalf("ParseContextDelta: %v", err)
	}
	if len(delta.Entries) != 1 || delta.NoImpact {
		t.Fatalf("las vallas no abren entradas ni declaran «Sin impacto»: %+v", delta)
	}
	if !strings.Contains(delta.Entries[0].Body, "## Entrada: context/Falsa.md") || !strings.HasSuffix(delta.Entries[0].Body, "Cierre.") {
		t.Errorf("el cuerpo conserva el contenido de la valla: %q", delta.Entries[0].Body)
	}
}

func TestParseContextDeltaEntryWithoutBodyAndWithOtherHeadings(t *testing.T) {
	data := "## Entrada: context/A.md\n- Ancla: Uno\n- Acción: add\n- Motivo: sin línea en blanco tras los campos\n## Entrada: context/B.md\n\n- Ancla: Dos\n- Acción: add\n- Motivo: m\n\n## Notas\n\nUn encabezado que no es una entrada forma parte del cuerpo.\n"
	delta, err := ParseContextDelta([]byte(data))
	if err != nil {
		t.Fatalf("ParseContextDelta: %v", err)
	}
	if len(delta.Entries) != 2 || delta.Entries[0].Body != "" {
		t.Fatalf("delta = %+v", delta)
	}
	if want := "## Notas\n\nUn encabezado que no es una entrada forma parte del cuerpo."; delta.Entries[1].Body != want {
		t.Errorf("cuerpo = %q, quería %q", delta.Entries[1].Body, want)
	}
}

func TestParseContextDeltaRejections(t *testing.T) {
	entry := func(fields ...string) string {
		return "## Entrada: context/A.md\n\n" + strings.Join(fields, "\n") + "\n"
	}
	valid := []string{"- Ancla: Uno", "- Acción: modify", "- Motivo: razón"}
	tests := []struct {
		name string
		data string
		want []string
	}{
		{name: "fichero vacío", data: "", want: []string{"no contiene ninguna entrada", "Sin impacto"}},
		{name: "solo el encabezado", data: "# Delta de contexto: inc-99\n", want: []string{"no contiene ninguna entrada"}},
		{name: "«Sin impacto» sin motivo", data: "Sin impacto:\n", want: []string{"«Sin impacto» requiere un motivo"}},
		{name: "«Sin impacto» sin dos puntos", data: "Sin impacto\n", want: []string{"requiere un motivo"}},
		{name: "«Sin impacto» con entradas", data: "Sin impacto: nada\n\n" + entry(valid...), want: []string{"no pueden coexistir"}},
		{name: "«Sin impacto» repetido", data: "Sin impacto: uno\nSin impacto: dos\n", want: []string{"más de una vez", "línea 1"}},
		{name: "texto suelto antes de las entradas", data: "Una introducción.\n\n" + entry(valid...), want: []string{"línea 1", "texto no reconocido"}},
		{name: "valla fuera de una entrada", data: "```\ncódigo\n```\nSin impacto: x\n", want: []string{"línea 1", "bloque de código fuera de una entrada"}},
		{name: "campo desconocido", data: entry("- Ancla: Uno", "- Acción: modify", "- Motivo: m", "- Prioridad: alta"), want: []string{`campo desconocido "Prioridad"`, "línea 6"}},
		{name: "campo sin dos puntos", data: entry("- Ancla: Uno", "- Acción: modify", "- Motivo: m", "- suelto"), want: []string{"no es un campo"}},
		{name: "falta el ancla", data: entry("- Acción: modify", "- Motivo: m"), want: []string{"falta el campo `- Ancla:`"}},
		{name: "falta la acción", data: entry("- Ancla: Uno", "- Motivo: m"), want: []string{"falta el campo `- Acción:`"}},
		{name: "falta el motivo", data: entry("- Ancla: Uno", "- Acción: add"), want: []string{"falta el campo `- Motivo:`"}},
		{name: "sin campos", data: "## Entrada: context/A.md\n\nSolo cuerpo.\n", want: []string{"falta el campo `- Ancla:`", "falta el campo `- Acción:`", "falta el campo `- Motivo:`"}},
		{name: "campo repetido", data: entry("- Ancla: Uno", "- Ancla: Dos", "- Acción: add", "- Motivo: m"), want: []string{`el campo "Ancla" está repetido`}},
		{name: "campo vacío", data: entry("- Ancla: Uno", "- Acción: add", "- Motivo:"), want: []string{`el campo "Motivo" no puede estar vacío`}},
		{name: "acción desconocida", data: entry("- Ancla: Uno", "- Acción: replace", "- Motivo: m"), want: []string{`la acción "replace" no es válida`}},
		{name: "el campo exacto lleva tilde", data: entry("- Ancla: Uno", "- Accion: add", "- Motivo: m"), want: []string{`campo desconocido "Accion"`}},
		{name: "ancla inválida", data: entry("- Ancla: Pagos >  > Notas", "- Acción: add", "- Motivo: m"), want: []string{"elemento vacío"}},
		{name: "ordinal inválido", data: entry("- Ancla: Notas #0", "- Acción: add", "- Motivo: m"), want: []string{"mayor que cero"}},
		{name: "ruta vacía", data: "## Entrada:\n\n- Ancla: Uno\n- Acción: add\n- Motivo: m\n", want: []string{"no indica la ruta del documento"}},
		{name: "ruta absoluta", data: "## Entrada: /etc/context.md\n\n- Ancla: Uno\n- Acción: add\n- Motivo: m\n", want: []string{"debe ser relativa a workspaceRoot"}},
		{name: "ruta con unidad de disco", data: "## Entrada: C:\\docs\\a.md\n\n- Ancla: Uno\n- Acción: add\n- Motivo: m\n", want: []string{"debe ser relativa a workspaceRoot"}},
		{name: "ruta de directorio", data: "## Entrada: context/\n\n- Ancla: Uno\n- Acción: add\n- Motivo: m\n", want: []string{"no a un directorio"}},
		{
			name: "par documento y ancla repetido",
			data: "## Entrada: context/A.md\n\n- Ancla: Uno\n- Acción: add\n- Motivo: m\n\n## Entrada: context\\a.md\n\n- Ancla: Uno\n- Acción: modify\n- Motivo: n\n",
			want: []string{"la entrada C2 repite el documento", `la entrada C1 (línea 1)`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseContextDelta([]byte(tt.data))
			var parseErr *ContextDeltaError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %v, quería *ContextDeltaError", err)
			}
			for _, fragment := range tt.want {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("el error %q debe contener %q", err, fragment)
				}
			}
		})
	}
}

func TestParseContextDeltaSameAnchorWithDifferentOrdinalIsNotARepeat(t *testing.T) {
	data := "## Entrada: context/A.md\n\n- Ancla: Notas #1\n- Acción: modify\n- Motivo: a\n\n## Entrada: context/A.md\n\n- Ancla: Notas #2\n- Acción: modify\n- Motivo: b\n\n## Entrada: context/B.md\n\n- Ancla: Notas #1\n- Acción: modify\n- Motivo: c\n"
	if _, err := ParseContextDelta([]byte(data)); err != nil {
		t.Fatalf("un ordinal distinto, o un documento distinto, no repite el par: %v", err)
	}
}

func TestParseContextDeltaReportsEveryProblem(t *testing.T) {
	data := "## Entrada: context/A.md\n\n- Ancla: Uno\n- Acción: replace\n\n## Entrada: context/B.md\n\n- Ancla: Dos\n- Acción: add\n- Motivo: m\n- Extra: x\n"
	_, err := ParseContextDelta([]byte(data))
	var parseErr *ContextDeltaError
	if !errors.As(err, &parseErr) || len(parseErr.Problems) < 3 {
		t.Fatalf("debían reportarse todos los defectos, no solo el primero: %v", err)
	}
}

// contextTree monta workspaceRoot (<tmp>/especificaciones) con un repositorio
// hermano (<tmp>/repo-backend) y devuelve el ámbito de las pruebas.
func contextTree(t *testing.T) (ContextScope, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "especificaciones")
	for _, dir := range []string{
		filepath.Join(root, "context"), filepath.Join(root, "referencia", "legado"),
		filepath.Join(base, "repo-backend", "docs"), filepath.Join(base, "ajeno"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{
		filepath.Join(root, "01_Requisitos.md"), filepath.Join(root, "context", "ARQUITECTURA.md"),
		filepath.Join(root, "context", "NOTAS.txt"), filepath.Join(root, "referencia", "legado", "VIEJO.md"),
		filepath.Join(base, "repo-backend", "docs", "API.md"), filepath.Join(base, "ajeno", "X.md"),
	} {
		if err := os.WriteFile(file, []byte("# x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return ContextScope{
		WorkspaceRoot:     root,
		Maintained:        []string{"context/", "../repo-backend/docs/", "NOTAS-RAIZ.md"},
		Reference:         []string{"referencia/"},
		BaselineDocuments: []string{"01_Requisitos.md"},
	}, base
}

func scopeEntry(document string) ContextEntry {
	return ContextEntry{ID: "C4", Document: document, Line: 12}
}

func TestContextScopeAcceptsMaintainedDocuments(t *testing.T) {
	scope, _ := contextTree(t)
	for _, document := range []string{
		"context/ARQUITECTURA.md",            // existente
		"context/PAGOS.md",                   // por crear (REQ-24.55)
		"context/nuevo/dir/SERVICIO.md",      // por crear, con directorios intermedios
		"../repo-backend/docs/API.md",        // repositorio hermano con `..`
		"../repo-backend/docs/NUEVO.md",      // por crear en el repositorio hermano
		"context/../context/ARQUITECTURA.md", // la ruta se limpia antes de decidir
	} {
		if err := scope.CheckDocument(scopeEntry(document)); err != nil {
			t.Errorf("%s debía estar permitido: %v", document, err)
		}
	}
}

func TestContextScopeRejectsDocumentsOutsideTheMaintainedPaths(t *testing.T) {
	scope, _ := contextTree(t)
	tests := []struct {
		document string
		want     []string
	}{
		{document: "referencia/legado/VIEJO.md", want: []string{"C4", "línea 12", "ruta de referencia", "modo reference"}},
		{document: "referencia/NUEVO.md", want: []string{"modo reference"}},
		{document: "01_Requisitos.md", want: []string{"'baseline.documents'"}},
		{document: "../ajeno/X.md", want: []string{"no está bajo ninguna ruta mantenida", `"context/"`, `"../repo-backend/docs/"`}},
		{document: "docs/otro.md", want: []string{"no está bajo ninguna ruta mantenida"}},
		{document: "context-ajeno/X.md", want: []string{"no está bajo ninguna ruta mantenida"}}, // prefijo textual, otro directorio
	}
	for _, tt := range tests {
		err := scope.CheckDocument(scopeEntry(tt.document))
		if err == nil {
			t.Errorf("%s debía rechazarse", tt.document)
			continue
		}
		for _, fragment := range tt.want {
			if !strings.Contains(err.Error(), fragment) {
				t.Errorf("%s: el error %q debe contener %q", tt.document, err, fragment)
			}
		}
	}
}

func TestContextScopeDecidesByIdentityNotByText(t *testing.T) {
	scope, _ := contextTree(t)
	// Una ruta mantenida declarada como un fichero concreto solo permite ese fichero.
	if err := os.WriteFile(scope.Resolve("NOTAS-RAIZ.md"), []byte("# n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := scope.CheckDocument(scopeEntry("NOTAS-RAIZ.md")); err != nil {
		t.Errorf("el fichero mantenido debía permitirse: %v", err)
	}
	if err := scope.CheckDocument(scopeEntry("NOTAS-RAIZ.md.bak")); err == nil {
		t.Error("un fichero con el mismo prefijo no es el fichero mantenido")
	}
	// Con una ruta de referencia que existe, lo que resuelve a ella es referencia
	// aunque se escriba con otra grafía.
	if err := scope.CheckDocument(scopeEntry("context/../referencia/legado/VIEJO.md")); err == nil || !strings.Contains(err.Error(), "referencia") {
		t.Errorf("la grafía con `..` sigue siendo referencia: %v", err)
	}
}

func TestContextScopeRejectsASymlinkThatLeavesTheMaintainedPath(t *testing.T) {
	scope, base := contextTree(t)
	link := filepath.Join(scope.WorkspaceRoot, "context", "salida")
	if err := os.Symlink(filepath.Join(base, "ajeno"), link); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos en este sistema: %v", err)
	}
	if err := scope.CheckDocument(scopeEntry("context/salida/X.md")); err == nil {
		t.Fatal("un enlace simbólico bajo la ruta mantenida que sale de ella no debe permitir el documento")
	}
	if err := scope.CheckDocument(scopeEntry("context/salida/NUEVO.md")); err == nil {
		t.Fatal("tampoco un documento por crear tras el enlace")
	}
}

func TestContextScopeComparesAnUndeclaredPathLexically(t *testing.T) {
	scope, _ := contextTree(t)
	// La ruta mantenida aún no existe: no tiene identidad, así que se compara de
	// forma léxica y el primer documento de un directorio nuevo se permite.
	scope.Maintained = []string{"contexto-nuevo/"}
	if err := scope.CheckDocument(scopeEntry("contexto-nuevo/PRIMERO.md")); err != nil {
		t.Errorf("el primer documento de una ruta mantenida inexistente debía permitirse: %v", err)
	}
	if err := scope.CheckDocument(scopeEntry("contexto-nuevo-otro/X.md")); err == nil {
		t.Error("un directorio hermano con el mismo prefijo no está bajo la ruta mantenida")
	}
}

func TestContextScopeWithoutMaintainedPaths(t *testing.T) {
	scope, _ := contextTree(t)
	scope.Maintained = nil
	err := scope.CheckDocument(scopeEntry("context/ARQUITECTURA.md"))
	if err == nil || !strings.Contains(err.Error(), "(ninguna)") {
		t.Fatalf("sin rutas mantenidas se rechaza nombrando que no hay ninguna: %v", err)
	}
}
