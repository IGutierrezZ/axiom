package techcontext

import (
	"errors"
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

func TestParseDeltaWithEntries(t *testing.T) {
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		delta, err := ParseDelta(v.apply(contextDeltaExample))
		if err != nil {
			t.Fatalf("ParseDelta: %v", err)
		}
		if delta.NoImpact || len(delta.Entries) != 3 {
			t.Fatalf("delta = %+v, quería 3 entradas y sin «Sin impacto»", delta)
		}
		first, second, third := delta.Entries[0], delta.Entries[1], delta.Entries[2]
		if first.ID != "C1" || second.ID != "C2" || third.ID != "C3" {
			t.Errorf("identificadores %q, %q, %q; quería C1..C3 por orden de aparición", first.ID, second.ID, third.ID)
		}
		if first.Document != "context/ARQUITECTURA.md" || first.Action != Modify || first.Anchor.String() != "Componentes > Servicio de pagos" {
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
		if second.Document != "context/PAGOS.md" || second.Anchor.Ordinal != 2 || second.Action != Add || second.Body != "" {
			t.Errorf("segunda entrada (comillas invertidas, `\\` y ordinal) = %+v", second)
		}
		if third.Document != "../repo-backend/docs/API.md" || third.Action != Remove {
			t.Errorf("tercera entrada (repositorio hermano) = %+v", third)
		}
	})
}

func TestParseDeltaNoImpact(t *testing.T) {
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		delta, err := ParseDelta(v.apply("# Delta de contexto: inc-99-ejemplo\n\nSin impacto: el cambio solo corrige una errata en un mensaje de error\n"))
		if err != nil {
			t.Fatalf("ParseDelta: %v", err)
		}
		if !delta.NoImpact || len(delta.Entries) != 0 || delta.Reason != "el cambio solo corrige una errata en un mensaje de error" {
			t.Fatalf("delta = %+v", delta)
		}
	})
	if delta, err := ParseDelta([]byte("Sin impacto: sin encabezado inicial")); err != nil || !delta.NoImpact {
		t.Fatalf("el encabezado `# ...` es opcional: %+v %v", delta, err)
	}
}

func TestParseDeltaFencesAreBodyNotStructure(t *testing.T) {
	data := "## Entrada: context/A.md\n\n- Ancla: Uno\n- Acción: add\n- Motivo: ejemplo\n\n```markdown\n## Entrada: context/Falsa.md\n- Campo: no es un campo\nSin impacto: tampoco\n```\n\nCierre.\n"
	delta, err := ParseDelta([]byte(data))
	if err != nil {
		t.Fatalf("ParseDelta: %v", err)
	}
	if len(delta.Entries) != 1 || delta.NoImpact {
		t.Fatalf("las vallas no abren entradas ni declaran «Sin impacto»: %+v", delta)
	}
	if !strings.Contains(delta.Entries[0].Body, "## Entrada: context/Falsa.md") || !strings.HasSuffix(delta.Entries[0].Body, "Cierre.") {
		t.Errorf("el cuerpo conserva el contenido de la valla: %q", delta.Entries[0].Body)
	}
}

func TestParseDeltaEntryWithoutBodyAndWithOtherHeadings(t *testing.T) {
	data := "## Entrada: context/A.md\n- Ancla: Uno\n- Acción: add\n- Motivo: sin línea en blanco tras los campos\n## Entrada: context/B.md\n\n- Ancla: Dos\n- Acción: add\n- Motivo: m\n\n## Notas\n\nUn encabezado que no es una entrada forma parte del cuerpo.\n"
	delta, err := ParseDelta([]byte(data))
	if err != nil {
		t.Fatalf("ParseDelta: %v", err)
	}
	if len(delta.Entries) != 2 || delta.Entries[0].Body != "" {
		t.Fatalf("delta = %+v", delta)
	}
	if want := "## Notas\n\nUn encabezado que no es una entrada forma parte del cuerpo."; delta.Entries[1].Body != want {
		t.Errorf("cuerpo = %q, quería %q", delta.Entries[1].Body, want)
	}
}

func TestParseDeltaRejections(t *testing.T) {
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
			_, err := ParseDelta([]byte(tt.data))
			var parseErr *DeltaError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %v, quería *DeltaError", err)
			}
			for _, fragment := range tt.want {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("el error %q debe contener %q", err, fragment)
				}
			}
		})
	}
}

func TestParseDeltaSameAnchorWithDifferentOrdinalIsNotARepeat(t *testing.T) {
	data := "## Entrada: context/A.md\n\n- Ancla: Notas #1\n- Acción: modify\n- Motivo: a\n\n## Entrada: context/A.md\n\n- Ancla: Notas #2\n- Acción: modify\n- Motivo: b\n\n## Entrada: context/B.md\n\n- Ancla: Notas #1\n- Acción: modify\n- Motivo: c\n"
	if _, err := ParseDelta([]byte(data)); err != nil {
		t.Fatalf("un ordinal distinto, o un documento distinto, no repite el par: %v", err)
	}
}

func TestParseDeltaReportsEveryProblem(t *testing.T) {
	data := "## Entrada: context/A.md\n\n- Ancla: Uno\n- Acción: replace\n\n## Entrada: context/B.md\n\n- Ancla: Dos\n- Acción: add\n- Motivo: m\n- Extra: x\n"
	_, err := ParseDelta([]byte(data))
	var parseErr *DeltaError
	if !errors.As(err, &parseErr) || len(parseErr.Problems) < 3 {
		t.Fatalf("debían reportarse todos los defectos, no solo el primero: %v", err)
	}
}

// contextTree monta workspaceRoot (<tmp>/especificaciones) con un repositorio
