package isoevidence

import (
	"strings"
	"testing"
)

const matrixHeader = "| Requisito | Escenario | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado | Commit |\n" +
	"|---|---|---|---|---|---|---|---|---|---|---|\n"

func TestParseMatrix(t *testing.T) {
	t.Run("sin sección: matriz ausente", func(t *testing.T) {
		m := ParseMatrix("# Informe\n\n## Cumplimiento técnico\n\nTodo bien.\n")
		if m.Present || len(m.Rows) != 0 || len(m.Issues) != 0 {
			t.Fatalf("matriz = %+v, quería ausente", m)
		}
	})

	t.Run("informe vacío", func(t *testing.T) {
		if m := ParseMatrix(""); m.Present {
			t.Fatalf("un informe vacío no tiene matriz: %+v", m)
		}
	})

	t.Run("tabla completa con filas automática y manual", func(t *testing.T) {
		report := "# Informe\n\nverdict: pass\n\n## Matriz de verificación\n\n" + matrixHeader +
			"| REQ-30.1 | Cobro con IVA general | unitario | `internal/pagos/cobro_test.go:TestCobraIVA` | `go test ./internal/pagos -run TestCobraIVA -count=1` | 0 | | | | pass | 1a2b3c4 |\n" +
			"| REQ-30.2 | Recibo impreso | manual | acta-12 | | | Ana Pérez | 2026-11-12 | La impresora fiscal no tiene emulador | pass | |\n" +
			"\n## Otra sección\n\n| REQ-99.9 | Ajena | unitario | x | x | 0 | | | | pass | |\n"
		m := ParseMatrix(report)
		if !m.Present || len(m.Issues) != 0 {
			t.Fatalf("matriz = %+v", m)
		}
		if len(m.Rows) != 2 {
			t.Fatalf("filas = %d, quería 2: %+v", len(m.Rows), m.Rows)
		}
		got := m.Rows[0]
		want := MatrixRow{
			Line: 9, RequirementID: "REQ-30.1", Scenario: "Cobro con IVA general", Type: "unitario",
			Reference: "internal/pagos/cobro_test.go:TestCobraIVA",
			Command:   "go test ./internal/pagos -run TestCobraIVA -count=1", ExitCode: "0", Result: "pass", Commit: "1a2b3c4",
		}
		if got != want {
			t.Errorf("fila 0 = %+v, quería %+v", got, want)
		}
		manual := m.Rows[1]
		if manual.Owner != "Ana Pérez" || manual.Date != "2026-11-12" || manual.Rationale != "La impresora fiscal no tiene emulador" || manual.Command != "" || manual.ExitCode != "" {
			t.Errorf("fila manual = %+v", manual)
		}
	})

	t.Run("las cabeceras se reconocen por nombre, sin importar orden ni mayúsculas", func(t *testing.T) {
		report := "## Matriz de verificación\n\n" +
			"| RESULTADO | escenario | REQUISITO | Salida | Tipo | referencia | COMANDO | responsable | fecha | justificación |\n" +
			"|:--|:-:|--:|---|---|---|---|---|---|---|\n" +
			"| pass | Esc | REQ-1.1 | 0 | unitario | TestX | go test | | | |\n"
		m := ParseMatrix(report)
		if len(m.Issues) != 0 || len(m.Rows) != 1 {
			t.Fatalf("matriz = %+v", m)
		}
		r := m.Rows[0]
		if r.Result != "pass" || r.Scenario != "Esc" || r.RequirementID != "REQ-1.1" || r.ExitCode != "0" || r.Type != "unitario" || r.Reference != "TestX" || r.Command != "go test" {
			t.Errorf("fila = %+v", r)
		}
		if r.Commit != "" {
			t.Errorf("sin columna Commit, la celda es vacía: %q", r.Commit)
		}
	})

	t.Run("la barra escapada es literal", func(t *testing.T) {
		report := "## Matriz de verificación\n\n" + matrixHeader +
			"| REQ-1.1 | Esc | unitario | TestX | `go test -run 'A\\|B' ./...` | 0 | | | | pass | |\n"
		m := ParseMatrix(report)
		if len(m.Issues) != 0 || len(m.Rows) != 1 {
			t.Fatalf("matriz = %+v", m)
		}
		if got := m.Rows[0].Command; got != "go test -run 'A|B' ./..." {
			t.Errorf("Command = %q", got)
		}
	})

	t.Run("una barra sin escapar da una fila con más celdas", func(t *testing.T) {
		report := "## Matriz de verificación\n\n" + matrixHeader +
			"| REQ-1.1 | Esc | unitario | TestX | go test -run A|B | 0 | | | | pass | |\n"
		m := ParseMatrix(report)
		if len(m.Issues) != 1 || m.Issues[0].Code != CodeMatrixInvalid || !strings.Contains(m.Issues[0].Reason(), "celdas") {
			t.Fatalf("issues = %v", reasons(m.Issues))
		}
	})

	t.Run("las celdas con guion o raya no aplican", func(t *testing.T) {
		report := "## Matriz de verificación\n\n" + matrixHeader +
			"| REQ-2.1 | Esc | manual | - | — | - | Ana | 2026-11-12 | Motivo | pass | — |\n"
		m := ParseMatrix(report)
		r := m.Rows[0]
		if r.Reference != "" || r.Command != "" || r.ExitCode != "" || r.Commit != "" {
			t.Errorf("fila = %+v", r)
		}
	})

	t.Run("dos encabezados de nivel 2: matriz ambigua", func(t *testing.T) {
		report := "## Matriz de verificación\n\n" + matrixHeader + "\n## Matriz de verificación\n\n" + matrixHeader
		m := ParseMatrix(report)
		if !m.Present || len(m.Issues) != 1 || len(m.Rows) != 0 {
			t.Fatalf("matriz = %+v", m)
		}
		if !strings.Contains(m.Issues[0].Reason(), "ambigua") {
			t.Errorf("el motivo no dice que es ambigua: %q", m.Issues[0].Reason())
		}
	})

	t.Run("una matriz histórica de nivel 3 se ignora", func(t *testing.T) {
		report := "## Matriz de verificación\n\n" + matrixHeader +
			"| REQ-1.1 | Esc | unitario | TestX | go test | 0 | | | | pass | |\n\n" +
			"## Contexto histórico\n\n### Matriz de verificación (histórica, 2026-10-01)\n\n" + matrixHeader +
			"| REQ-0.9 | Vieja | unitario | TestVieja | go test | 1 | | | | fail | |\n"
		m := ParseMatrix(report)
		if len(m.Issues) != 0 || len(m.Rows) != 1 || m.Rows[0].RequirementID != "REQ-1.1" {
			t.Fatalf("matriz = %+v", m)
		}
	})

	t.Run("solo la matriz histórica: no hay matriz vigente", func(t *testing.T) {
		report := "## Contexto histórico\n\n### Matriz de verificación (histórica, 2026-10-01)\n\n" + matrixHeader
		if m := ParseMatrix(report); m.Present {
			t.Fatalf("la matriz de nivel 3 no cuenta: %+v", m)
		}
	})

	t.Run("un encabezado dentro de una valla no cuenta", func(t *testing.T) {
		report := "```markdown\n## Matriz de verificación\n\n" + matrixHeader + "```\n"
		if m := ParseMatrix(report); m.Present {
			t.Fatalf("la valla no es una matriz: %+v", m)
		}
	})

	t.Run("sección presente y vacía", func(t *testing.T) {
		m := ParseMatrix("## Matriz de verificación\n\n" + matrixHeader)
		if !m.Present || len(m.Rows) != 0 || len(m.Issues) != 0 {
			t.Fatalf("matriz = %+v", m)
		}
	})

	t.Run("sección sin tabla", func(t *testing.T) {
		m := ParseMatrix("## Matriz de verificación\n\nNo aplica.\n")
		if !m.Present || len(m.Rows) != 0 || len(m.Issues) != 0 {
			t.Fatalf("matriz = %+v", m)
		}
	})

	t.Run("falta una cabecera obligatoria", func(t *testing.T) {
		report := "## Matriz de verificación\n\n| Requisito | Escenario | Tipo |\n|---|---|---|\n| REQ-1.1 | Esc | unitario |\n"
		m := ParseMatrix(report)
		var named []string
		for _, f := range m.Issues {
			named = append(named, f.Reason())
		}
		joined := strings.Join(named, "\n")
		for _, col := range []string{"Referencia", "Comando", "Salida", "Responsable", "Fecha", "Justificación", "Resultado"} {
			if !strings.Contains(joined, "`"+col+"`") {
				t.Errorf("no se reporta la columna %s:\n%s", col, joined)
			}
		}
		if strings.Contains(joined, "`Commit`") {
			t.Error("Commit es opcional y no debe reportarse")
		}
	})

	t.Run("final de línea CRLF", func(t *testing.T) {
		report := strings.ReplaceAll("## Matriz de verificación\n\n"+matrixHeader+
			"| REQ-1.1 | Esc | unitario | TestX | go test | 0 | | | | pass | |\n", "\n", "\r\n")
		m := ParseMatrix(report)
		if len(m.Issues) != 0 || len(m.Rows) != 1 || m.Rows[0].Result != "pass" {
			t.Fatalf("matriz = %+v", m)
		}
	})

	t.Run("una línea verdict: dentro de la sección invalida la matriz", func(t *testing.T) {
		report := "## Matriz de verificación\n\nverdict: pass\n\n" + matrixHeader
		m := ParseMatrix(report)
		if len(m.Issues) != 1 || !strings.Contains(m.Issues[0].Reason(), "verdict:") {
			t.Fatalf("issues = %v", reasons(m.Issues))
		}
	})

	t.Run("ninguna línea de la matriz empieza por verdict:", func(t *testing.T) {
		report := "verdict: pass\n\n## Matriz de verificación\n\n" + matrixHeader +
			"| REQ-1.1 | Esc | unitario | TestX | go test | 0 | | | | pass | |\n"
		m := ParseMatrix(report)
		if len(m.Issues) != 0 {
			t.Fatalf("la línea verdict: previa, fuera de la sección, no debe afectar a la matriz: %v", reasons(m.Issues))
		}
		inSection := false
		for _, line := range strings.Split(report, "\n") {
			if strings.HasPrefix(line, "## ") {
				inSection = line == matrixHeading
				continue
			}
			if inSection && strings.HasPrefix(line, "verdict:") {
				t.Errorf("la sección contiene una línea verdict:: %q", line)
			}
		}
	})
}
