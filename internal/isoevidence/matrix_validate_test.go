package isoevidence

import (
	"fmt"
	"strings"
	"testing"
)

// fakeTree es un TreeResolver simulado.
type fakeTree struct {
	pathSymbols map[string]ResolveOutcome // clave "ruta|símbolo"
	goTests     map[string]bool
	err         error
	calls       []string
}

func (f *fakeTree) ResolvePathSymbol(rel, symbol string) (ResolveOutcome, error) {
	f.calls = append(f.calls, "path:"+rel+"|"+symbol)
	if f.err != nil {
		return "", f.err
	}
	if o, ok := f.pathSymbols[rel+"|"+symbol]; ok {
		return o, nil
	}
	return OutcomeFileMissing, nil
}

func (f *fakeTree) ResolveGoTest(name string) (bool, error) {
	f.calls = append(f.calls, "test:"+name)
	return f.goTests[name], f.err
}

func newFakeTree() *fakeTree {
	return &fakeTree{
		pathSymbols: map[string]ResolveOutcome{
			"internal/pagos/cobro_test.go|TestCobraIVA": OutcomeResolved,
			"e2e/checkout.spec.ts|pagaConTarjeta":       OutcomeResolved,
			"e2e/checkout.spec.ts|ausente":              OutcomeSymbolMissing,
			"../otro/prueba_test.go|TestX":              OutcomeOutsideTree,
			"tests/test_pago.py|test_cobra_iva":         OutcomeResolved,
		},
		goTests: map[string]bool{"TestArchiveBlocksOnMatrix": true},
	}
}

// matrixOf construye un Matrix a partir de filas ya valoradas.
func matrixOf(rows ...MatrixRow) Matrix {
	for i := range rows {
		if rows[i].Line == 0 {
			rows[i].Line = 10 + i
		}
	}
	return Matrix{Present: true, Rows: rows}
}

func autoRow(req, scenario, ref string) MatrixRow {
	return MatrixRow{RequirementID: req, Scenario: scenario, Type: TypeUnit, Reference: ref, Command: "go test ./...", ExitCode: "0", Result: ResultPass}
}

func manualRow(req, scenario string) MatrixRow {
	return MatrixRow{RequirementID: req, Scenario: scenario, Type: TypeManual, Owner: "Ana Pérez", Date: "2026-11-12", Rationale: "No hay emulador", Result: ResultPass}
}

func TestValidateMatrix(t *testing.T) {
	affected := []ScenarioKey{{"REQ-30.1", "A"}, {"REQ-30.1", "B"}, {"REQ-30.2", "C"}}
	okA := autoRow("REQ-30.1", "A", "internal/pagos/cobro_test.go:TestCobraIVA")
	okB := autoRow("REQ-30.1", "B", "TestArchiveBlocksOnMatrix")
	okC := manualRow("REQ-30.2", "C")

	cases := []struct {
		name      string
		matrix    Matrix
		affected  []ScenarioKey
		noImpact  bool
		wantState MatrixState
		wantOK    bool
		want      [][]string // fragmentos de cada hallazgo, en orden; nil = ninguno
		wantCodes []Code
	}{
		{
			name:      "matriz completa y válida",
			matrix:    matrixOf(okA, okB, okC),
			affected:  affected,
			wantState: MatrixValid, wantOK: true,
		},
		{
			name:      "matriz ausente en ISO",
			matrix:    Matrix{},
			affected:  affected,
			wantState: MatrixMissing,
			want:      [][]string{{"blocked(iso_matrix_missing)", "Matriz de verificación"}},
			wantCodes: []Code{CodeMatrixMissing},
		},
		{
			name:      "un escenario sin fila",
			matrix:    matrixOf(okA, okC),
			affected:  affected,
			wantState: MatrixInvalid,
			want:      [][]string{{"escenario \"B\"", "REQ-30.1", "no tiene fila"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "una fila a un escenario inexistente",
			matrix:    matrixOf(okA, okB, okC, autoRow("REQ-30.1", "Z", "TestArchiveBlocksOnMatrix")),
			affected:  affected,
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 13", "escenario \"Z\"", "no figura entre los afectados"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "una fila a un requisito que no es afectado",
			matrix:    matrixOf(okA, okB, okC, autoRow("REQ-99.9", "A", "TestArchiveBlocksOnMatrix")),
			affected:  affected,
			wantState: MatrixInvalid,
			want:      [][]string{{"REQ-99.9", "no figura entre los afectados"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "dos filas para el mismo escenario",
			matrix:    matrixOf(okA, okA, okB, okC),
			affected:  affected,
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 11", "duplicado"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name: "resultado vacío y resultado desconocido",
			matrix: matrixOf(
				func() MatrixRow { r := okA; r.Result = ""; return r }(),
				func() MatrixRow { r := okB; r.Result = "quizás"; return r }(),
				okC),
			affected:  affected,
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 10", "resultado está vacío"}, {"línea 11", `"quizás"`}},
			wantCodes: []Code{CodeMatrixInvalid, CodeMatrixInvalid},
		},
		{
			name:      "tipo desconocido",
			matrix:    matrixOf(okA, func() MatrixRow { r := okB; r.Type = "smoke"; return r }(), okC),
			affected:  affected,
			wantState: MatrixInvalid,
			want:      [][]string{{"tipo", `"smoke"`}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "tipo sin tilde no es exacto",
			matrix:    matrixOf(okA, func() MatrixRow { r := okB; r.Type = "integracion"; return r }(), okC),
			affected:  affected,
			wantState: MatrixInvalid,
			want:      [][]string{{`"integracion"`}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:   "una fila fallida: válida pero no de éxito",
			matrix: matrixOf(okA, func() MatrixRow { r := okB; r.Result = ResultFail; r.ExitCode = "1"; return r }(), okC),
			// valid pero no de éxito
			affected:  affected,
			wantState: MatrixValid, wantOK: false,
			want:      [][]string{{"blocked(iso_matrix_failed)", "línea 11", "`fail`"}},
			wantCodes: []Code{CodeMatrixFailed},
		},
		{
			name:      "la columna Commit no se valida",
			matrix:    matrixOf(func() MatrixRow { r := okA; r.Commit = "no-existe-en-git"; return r }(), okB, okC),
			affected:  affected,
			wantState: MatrixValid, wantOK: true,
		},
		{
			name:      "referencia de fila automática con TestX resuelta",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "TestArchiveBlocksOnMatrix")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixValid, wantOK: true,
		},
		{
			name:      "TestX que ningún fichero declara",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "TestInexistente")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 10", "TestInexistente", "ningún `*_test.go`"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "fichero inexistente",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "e2e/otro.spec.ts:pagaConTarjeta")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 10", "e2e/otro.spec.ts", "no existe"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "símbolo ausente del fichero",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "e2e/checkout.spec.ts:ausente")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"símbolo \"ausente\"", "identificador completo"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "ruta que sale del árbol",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "../otro/prueba_test.go:TestX")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"fuera del árbol", "../otro/prueba_test.go"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "ruta absoluta de Windows",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", `C:\x:Sym`)),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"fuera del árbol"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "ruta UNC",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", `\\srv\x:Sym`)),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"fuera del árbol"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "ruta absoluta de Unix",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "/etc/passwd:root")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"fuera del árbol"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "referencia en Python con la forma general",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "tests/test_pago.py:test_cobra_iva")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixValid, wantOK: true,
		},
		{
			name:      "referencia sin ninguna forma admitida",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "la prueba de pagos")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"la prueba de pagos", "formas admitidas", "ruta/fichero:símbolo", "TestX"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "referencia vacía",
			matrix:    matrixOf(autoRow("REQ-1.1", "S", "")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"no tiene referencia"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name: "sin comando y sin código de salida",
			matrix: matrixOf(
				func() MatrixRow { r := autoRow("REQ-1.1", "S1", "TestArchiveBlocksOnMatrix"); r.Command = ""; return r }(),
				func() MatrixRow {
					r := autoRow("REQ-1.1", "S2", "TestArchiveBlocksOnMatrix")
					r.ExitCode = ""
					return r
				}(),
			),
			affected:  []ScenarioKey{{"REQ-1.1", "S1"}, {"REQ-1.1", "S2"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 10", "comando"}, {"línea 11", "código de salida"}},
			wantCodes: []Code{CodeMatrixInvalid, CodeMatrixInvalid},
		},
		{
			name: "código de salida que no es un entero",
			matrix: matrixOf(
				func() MatrixRow {
					r := autoRow("REQ-1.1", "S", "TestArchiveBlocksOnMatrix")
					r.ExitCode = "ok"
					return r
				}(),
			),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{`"ok"`, "no es un entero"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name: "pass con código de salida distinto de cero",
			matrix: matrixOf(
				func() MatrixRow {
					r := autoRow("REQ-1.1", "S", "TestArchiveBlocksOnMatrix")
					r.ExitCode = "1"
					return r
				}(),
			),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 10", "`pass`", "código de salida es 1", "incoherente"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "fila manual completa",
			matrix:    matrixOf(manualRow("REQ-1.1", "S")),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixValid, wantOK: true,
		},
		{
			name: "fila manual sin responsable, fecha ni justificación",
			matrix: matrixOf(
				func() MatrixRow { r := manualRow("REQ-1.1", "S"); r.Owner, r.Date, r.Rationale = "", "", ""; return r }(),
			),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{"Responsable"}, {"Fecha"}, {"Justificación"}},
			wantCodes: []Code{CodeMatrixInvalid, CodeMatrixInvalid, CodeMatrixInvalid},
		},
		{
			name: "fecha no válida",
			matrix: matrixOf(
				func() MatrixRow { r := manualRow("REQ-1.1", "S"); r.Date = "ayer"; return r }(),
			),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{`"ayer"`, "AAAA-MM-DD"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name: "fecha con un día que no existe",
			matrix: matrixOf(
				func() MatrixRow { r := manualRow("REQ-1.1", "S"); r.Date = "2026-02-30"; return r }(),
			),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixInvalid,
			want:      [][]string{{`"2026-02-30"`}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name: "la referencia de una fila manual no se resuelve",
			matrix: matrixOf(
				func() MatrixRow { r := manualRow("REQ-1.1", "S"); r.Reference = "acta-de-pruebas-12"; return r }(),
			),
			affected:  []ScenarioKey{{"REQ-1.1", "S"}},
			wantState: MatrixValid, wantOK: true,
		},
		{
			name:      "una incidencia de forma de ParseMatrix invalida la matriz",
			matrix:    Matrix{Present: true, Issues: []Finding{matrixIssue("cabecera", "falta la columna `Tipo`", "")}},
			affected:  nil,
			wantState: MatrixInvalid,
			want:      [][]string{{"columna `Tipo`"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
		{
			name:      "bug sin impacto: matriz ausente es válida y de éxito",
			matrix:    Matrix{},
			noImpact:  true,
			wantState: MatrixNotRequired, wantOK: true,
		},
		{
			name:      "bug sin impacto: matriz presente y vacía es válida y de éxito",
			matrix:    Matrix{Present: true},
			noImpact:  true,
			wantState: MatrixNotRequired, wantOK: true,
		},
		{
			name:      "bug sin impacto: cualquier fila invalida la matriz",
			matrix:    matrixOf(autoRow("REQ-13.1", "X", "TestArchiveBlocksOnMatrix")),
			noImpact:  true,
			wantState: MatrixInvalid,
			want:      [][]string{{"línea 10", "REQ-13.1", `"X"`, "no figura entre los afectados"}},
			wantCodes: []Code{CodeMatrixInvalid},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateMatrix(tc.matrix, tc.affected, tc.noImpact, newFakeTree())
			if got.State != tc.wantState {
				t.Fatalf("estado = %q, quería %q (hallazgos: %v)", got.State, tc.wantState, reasons(got.Findings))
			}
			if got.Successful != tc.wantOK {
				t.Errorf("Successful = %v, quería %v", got.Successful, tc.wantOK)
			}
			if len(got.Findings) != len(tc.want) {
				t.Fatalf("hallazgos = %v, quería %d", reasons(got.Findings), len(tc.want))
			}
			for i, f := range got.Findings {
				if f.Code != tc.wantCodes[i] || f.Stage != StageArchive {
					t.Errorf("hallazgo %d: %s/%s, quería %s/%s", i, f.Code, f.Stage, tc.wantCodes[i], StageArchive)
				}
				for _, fragment := range tc.want[i] {
					if !strings.Contains(f.Reason(), fragment) {
						t.Errorf("hallazgo %d: %q no contiene %q", i, f.Reason(), fragment)
					}
				}
			}
		})
	}
}

// TestValidateMatrixKeepsManualRowsOffTheTree: una fila manual no consulta el
// árbol y el resultado de resolver nunca se pide para una referencia manual.
func TestValidateMatrixKeepsManualRowsOffTheTree(t *testing.T) {
	tree := newFakeTree()
	row := manualRow("REQ-1.1", "S")
	row.Reference = "acta-12"
	ValidateMatrix(matrixOf(row), []ScenarioKey{{"REQ-1.1", "S"}}, false, tree)
	if len(tree.calls) != 0 {
		t.Fatalf("una fila manual no debe resolver nada contra el árbol: %v", tree.calls)
	}
}

func TestValidateMatrixResolverFailure(t *testing.T) {
	tree := newFakeTree()
	tree.err = fmt.Errorf("disco ilegible")
	got := ValidateMatrix(
		matrixOf(autoRow("REQ-1.1", "S", "e2e/checkout.spec.ts:pagaConTarjeta")),
		[]ScenarioKey{{"REQ-1.1", "S"}}, false, tree)
	if got.State != MatrixInvalid || len(got.Findings) != 1 || !strings.Contains(got.Findings[0].Reason(), "disco ilegible") {
		t.Fatalf("un fallo del resolutor debía invalidar con su causa: %+v", got)
	}
}

func TestValidateMatrixWithoutTree(t *testing.T) {
	got := ValidateMatrix(
		matrixOf(autoRow("REQ-1.1", "S", "TestX")),
		[]ScenarioKey{{"REQ-1.1", "S"}}, false, nil)
	if got.State != MatrixInvalid {
		t.Fatalf("sin árbol una fila automática no puede resolver: %+v", got)
	}
}

// TestValidateMatrixFromReport recorre el camino completo: texto del informe,
// ParseMatrix y ValidateMatrix.
func TestValidateMatrixFromReport(t *testing.T) {
	report := "## Matriz de verificación\n\n" + matrixHeader +
		"| REQ-30.1 | Cobro con IVA general | unitario | `internal/pagos/cobro_test.go:TestCobraIVA` | `go test ./internal/pagos -run TestCobraIVA -count=1` | 0 | | | | pass | 1a2b3c4 |\n" +
		"| REQ-30.2 | Recibo impreso | manual | acta-12 | | | Ana Pérez | 2026-11-12 | La impresora fiscal no tiene emulador | pass | |\n"
	affected := []ScenarioKey{{"REQ-30.1", "Cobro con IVA general"}, {"REQ-30.2", "Recibo impreso"}}
	got := ValidateMatrix(ParseMatrix(report), affected, false, newFakeTree())
	if got.State != MatrixValid || !got.Successful {
		t.Fatalf("resultado = %+v (%v)", got, reasons(got.Findings))
	}
}
