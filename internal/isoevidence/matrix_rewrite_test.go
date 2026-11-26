package isoevidence

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const rewriteMatrixHeader = "| Requisito | Escenario | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado | Commit |\n" +
	"|---|---|---|---|---|---|---|---|---|---|---|\n"

func rewriteMatrixRow(requirement, scenario string) string {
	return "| " + requirement + " | " + scenario + " | unitario | `internal/x/y_test.go` | `go test` | 0 | ana | 2026-11-26 | prueba de prov:30:1 | pass | abc1234 |\n"
}

// REQ-47.4 destino 5: RewriteMatrix solo toca la columna Requisito de las filas de la matriz.
func TestRewriteMatrix(t *testing.T) {
	m := reqid.NewIDMap([]reqid.Assignment{{Provisional: "prov:30:1", Definitive: "REQ-31.2"}})
	report := "# Informe\n\nMenciona prov:30:1 en el texto.\n\n" +
		"## Matriz de verificación\n\n" + rewriteMatrixHeader +
		rewriteMatrixRow("prov:30:1", "Con IVA") +
		rewriteMatrixRow("`prov:30:1`", "Sin IVA") +
		rewriteMatrixRow("REQ-29.1", "Otro") +
		rewriteMatrixRow("prov:30:11", "Intacto") +
		"\n## Otra sección\n\n| Requisito | x |\n|---|---|\n| prov:30:1 | y |\n"
	want := strings.Replace(report, "| prov:30:1 | Con IVA", "| REQ-31.2 | Con IVA", 1)
	want = strings.Replace(want, "| `prov:30:1` | Sin IVA", "| `REQ-31.2` | Sin IVA", 1)

	got := RewriteMatrix(report, m)
	if got != want {
		t.Errorf("RewriteMatrix =\n%s\nquería\n%s", got, want)
	}
	if !strings.Contains(got, "prueba de prov:30:1") || !strings.Contains(got, "| prov:30:1 | y |") || !strings.Contains(got, "Menciona prov:30:1 en el texto") {
		t.Error("solo cambia la columna Requisito de la matriz")
	}
	if once := RewriteMatrix(report, m); RewriteMatrix(once, m) != once {
		t.Error("la reescritura es idempotente")
	}
	if got := RewriteMatrix(report, reqid.NewIDMap(nil)); got != report {
		t.Error("un mapa vacío no reescribe nada")
	}
	crlf := strings.ReplaceAll(report, "\n", "\r\n")
	if got := RewriteMatrix(crlf, m); got != strings.ReplaceAll(want, "\n", "\r\n") {
		t.Error("los finales de línea CRLF se conservan")
	}
}

func TestRewriteMatrixLeavesAmbiguousOrMissingMatrixAlone(t *testing.T) {
	m := reqid.NewIDMap([]reqid.Assignment{{Provisional: "prov:30:1", Definitive: "REQ-31.2"}})
	row := rewriteMatrixRow("prov:30:1", "S")
	twice := "## Matriz de verificación\n\n" + rewriteMatrixHeader + row + "\n## Matriz de verificación\n\n" + rewriteMatrixHeader + row
	if got := RewriteMatrix(twice, m); got != twice {
		t.Error("una matriz ambigua no se reescribe")
	}
	if got := RewriteMatrix("sin matriz prov:30:1\n", m); got != "sin matriz prov:30:1\n" {
		t.Error("sin matriz no se reescribe")
	}
	noColumn := "## Matriz de verificación\n\n| Escenario | Tipo |\n|---|---|\n| prov:30:1 | x |\n"
	if got := RewriteMatrix(noColumn, m); got != noColumn {
		t.Error("sin columna Requisito no se reescribe")
	}
	fenced := "```\n## Matriz de verificación\n\n" + rewriteMatrixHeader + row + "```\n"
	if got := RewriteMatrix(fenced, m); got != fenced {
		t.Error("una matriz dentro de una valla no se reescribe")
	}
}

func TestRewriteMatrixEscapedPipeInEarlierCell(t *testing.T) {
	m := reqid.NewIDMap([]reqid.Assignment{{Provisional: "prov:30:1", Definitive: "REQ-31.2"}})
	header := "| Escenario | Requisito | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado |\n|---|---|---|---|---|---|---|---|---|---|\n"
	row := "| a \\| b | prov:30:1 | unitario | r | c | 0 | o | 2026-11-26 | j | pass |\n"
	got := RewriteMatrix("## Matriz de verificación\n\n"+header+row, m)
	if !strings.Contains(got, "| a \\| b | REQ-31.2 | unitario") {
		t.Errorf("la columna Requisito se localiza tras una barra escapada: %s", got)
	}
}

// REQ-29.3, REQ-47.4: la fila que cita el provisional cuenta como la del escenario afectado
// hasta archivar, y la fila reescrita sigue contando con el definitivo.
func TestMatrixCitesProvisionalUntilArchiveThenRewritten(t *testing.T) {
	p := reqid.Default()
	deltaReqs := []Requirement{req(SectionAdded, "pagos", "Cobro con IVA (prov:30:1)", "Con IVA")}
	report := "## Matriz de verificación\n\n" + rewriteMatrixHeader + rewriteMatrixRow("prov:30:1", "Con IVA")
	tree := newFakeTree()
	tree.pathSymbols["internal/x/y_test.go|"] = OutcomeResolved

	before := ValidateMatrix(ParseMatrix(report), AffectedFromDelta(deltaReqs, p), false, nil)
	for _, f := range before.Findings {
		if strings.Contains(f.Detail, "no tiene fila") || strings.Contains(f.Detail, "no figura entre los afectados") {
			t.Errorf("la fila del provisional debe contar como la del escenario afectado: %s", f.Reason())
		}
	}

	m := reqid.NewIDMap([]reqid.Assignment{{Provisional: "prov:30:1", Definitive: "REQ-31.2"}})
	deltaAfter := []Requirement{req(SectionAdded, "pagos", "Cobro con IVA (REQ-31.2)", "Con IVA")}
	after := ValidateMatrix(ParseMatrix(RewriteMatrix(report, m)), AffectedFromDelta(deltaAfter, p), false, nil)
	for _, f := range after.Findings {
		if strings.Contains(f.Detail, "no tiene fila") || strings.Contains(f.Detail, "no figura entre los afectados") {
			t.Errorf("tras reescribir, la fila cita el definitivo y cuenta: %s", f.Reason())
		}
	}
}
