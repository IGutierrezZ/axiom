package isoevidence

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const matrixHeading = "## Matriz de verificación"

// Columnas obligatorias de la matriz, por nombre (design §5.5). La columna
// Commit es opcional e informativa: nunca se valida.
var matrixRequiredColumns = []string{
	"Requisito", "Escenario", "Tipo", "Referencia", "Comando", "Salida",
	"Responsable", "Fecha", "Justificación", "Resultado",
}

// Tipos de fila admitidos, exactos.
const (
	TypeUnit        = "unitario"
	TypeIntegration = "integración"
	TypeE2E         = "e2e"
	TypeManual      = "manual"
)

// Resultados admitidos, exactos.
const (
	ResultPass = "pass"
	ResultFail = "fail"
)

// MatrixState es el estado de una matriz tras validarla.
type MatrixState string

const (
	MatrixValid       MatrixState = "valid"
	MatrixInvalid     MatrixState = "invalid"
	MatrixMissing     MatrixState = "missing"
	MatrixNotRequired MatrixState = "not_required"
)

// Matrix es la matriz leída de un verify-report.md.
type Matrix struct {
	Present bool
	Rows    []MatrixRow
	// Issues son los problemas de forma: matriz ambigua, cabeceras que faltan,
	// filas con más celdas que cabeceras y líneas `verdict:`.
	Issues []Finding
}

// MatrixRow es una fila de la matriz con sus celdas ya recortadas.
type MatrixRow struct {
	Line          int
	RequirementID string
	Scenario      string
	Type          string
	Reference     string
	Command       string
	ExitCode      string
	Owner         string
	Date          string
	Rationale     string
	Result        string
	Commit        string
}

func matrixIssue(subject, detail, continuation string) Finding {
	return Finding{
		Code:         CodeMatrixInvalid,
		Stage:        StageArchive,
		Subject:      subject,
		Detail:       detail,
		Continuation: continuation,
	}
}

const continuationMatrix = "ejecuta la fase `sdd-verify` para corregir `" + matrixHeading + "` en verify-report.md"

// ParseMatrix lee la sección `## Matriz de verificación` de un verify-report.md.
// Solo cuenta un encabezado de nivel 2 fuera de vallas; una matriz histórica a
// nivel 3 no coincide con él. Dos encabezados de nivel 2 son una matriz
// ambigua e inválida. La sección termina en el siguiente `#` o `##`.
func ParseMatrix(report string) Matrix {
	lines := splitLines(report)

	var starts []int
	for i, l := range lines {
		if !l.Fenced && strings.TrimRight(l.Text, " \t") == matrixHeading {
			starts = append(starts, i)
		}
	}
	if len(starts) == 0 {
		return Matrix{}
	}
	m := Matrix{Present: true}
	if len(starts) > 1 {
		m.Issues = append(m.Issues, matrixIssue(matrixHeading,
			fmt.Sprintf("la sección `%s` aparece %d veces: la matriz es ambigua", matrixHeading, len(starts)),
			"deja una sola sección de nivel 2; mueve la matriz anterior al contexto histórico con un encabezado de nivel 3 y fecha"))
		return m
	}

	var section []textLine
	for _, l := range lines[starts[0]+1:] {
		if !l.Fenced && (strings.HasPrefix(l.Text, "# ") || strings.HasPrefix(l.Text, "## ")) {
			break
		}
		if l.Fenced {
			continue
		}
		section = append(section, l)
	}

	for _, l := range section {
		if strings.HasPrefix(strings.TrimLeft(l.Text, " \t"), "verdict:") {
			m.Issues = append(m.Issues, matrixIssue(fmt.Sprintf("línea %d", l.Num),
				fmt.Sprintf("la línea %d de la matriz empieza por `verdict:`, que está reservada a la barrera de sincronización", l.Num),
				"retira esa línea de la matriz"))
		}
	}

	var headerIdx = -1
	for i, l := range section {
		if isTableLine(l.Text) {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return m
	}

	columns := splitCells(section[headerIdx].Text)
	position := map[string]int{}
	for i, c := range columns {
		key := strings.ToLower(c)
		if _, dup := position[key]; !dup {
			position[key] = i
		}
	}
	for _, required := range matrixRequiredColumns {
		if _, ok := position[strings.ToLower(required)]; !ok {
			m.Issues = append(m.Issues, matrixIssue("cabecera",
				fmt.Sprintf("la tabla de la matriz no tiene la columna `%s`", required),
				"añade la columna con ese nombre; las cabeceras se reconocen por nombre, no por posición"))
		}
	}

	body := section[headerIdx+1:]
	if len(body) > 0 && isSeparatorLine(body[0].Text) {
		body = body[1:]
	}
	cell := func(cells []string, name string) string {
		i, ok := position[strings.ToLower(name)]
		if !ok || i >= len(cells) {
			return ""
		}
		return cells[i]
	}
	for _, l := range body {
		if !isTableLine(l.Text) {
			break
		}
		cells := splitCells(l.Text)
		if len(cells) > len(columns) {
			m.Issues = append(m.Issues, matrixIssue(fmt.Sprintf("línea %d", l.Num),
				fmt.Sprintf("la fila de la línea %d tiene %d celdas y la cabecera %d", l.Num, len(cells), len(columns)),
				"escapa con `\\|` cada barra vertical literal de una celda"))
		}
		m.Rows = append(m.Rows, MatrixRow{
			Line:          l.Num,
			RequirementID: cell(cells, "Requisito"),
			Scenario:      cell(cells, "Escenario"),
			Type:          cell(cells, "Tipo"),
			Reference:     cell(cells, "Referencia"),
			Command:       cell(cells, "Comando"),
			ExitCode:      cell(cells, "Salida"),
			Owner:         cell(cells, "Responsable"),
			Date:          cell(cells, "Fecha"),
			Rationale:     cell(cells, "Justificación"),
			Result:        cell(cells, "Resultado"),
			Commit:        cell(cells, "Commit"),
		})
	}
	return m
}

func isTableLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "|")
}

var separatorCell = regexp.MustCompile(`^:?-{1,}:?$`)

func isSeparatorLine(line string) bool {
	if !isTableLine(line) {
		return false
	}
	cells := splitCells(line)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !separatorCell.MatchString(c) {
			return false
		}
	}
	return true
}

// splitCells divide una línea de tabla en celdas. `\|` es una barra vertical
// literal. Se recortan los espacios y un par de comillas invertidas que envuelva
// el valor; una celda con `-` o `—` no aplica y queda vacía.
func splitCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")

	var cells []string
	var cur strings.Builder
	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]
		switch {
		case c == '\\' && i+1 < len(trimmed) && trimmed[i+1] == '|':
			cur.WriteByte('|')
			i++
		case c == '|':
			cells = append(cells, cleanCell(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	// El texto tras la última barra solo es una celda si no está vacío: la
	// barra final de cierre no abre una celda nueva.
	if rest := strings.TrimSpace(cur.String()); rest != "" {
		cells = append(cells, cleanCell(rest))
	}
	return cells
}

func cleanCell(raw string) string {
	v := strings.TrimSpace(raw)
	if len(v) >= 2 && strings.HasPrefix(v, "`") && strings.HasSuffix(v, "`") {
		v = strings.TrimSpace(v[1 : len(v)-1])
	}
	if v == "-" || v == "—" {
		return ""
	}
	return v
}

// MatrixResult es el resultado de validar una matriz.
type MatrixResult struct {
	State MatrixState
	// Successful exige que la matriz sea válida (o no requerida) y que ninguna
	// fila tenga resultado `fail`.
	Successful bool
	Findings   []Finding
}

var (
	goTestShortcut = regexp.MustCompile(`^Test[^\s:/\\]*$`)
	driveReference = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
)

// ValidateMatrix valida la matriz contra los escenarios afectados (REQ-29.6 a
// REQ-29.8 y REQ-29.19). Nunca ejecuta el comando registrado de una fila: solo
// contrasta que la prueba existe en el árbol y que el comando y su código de
// salida quedaron registrados. noImpactBug indica un bug sin impacto en la
// spec, que no tiene escenarios afectados.
func ValidateMatrix(m Matrix, affected []ScenarioKey, noImpactBug bool, tree TreeResolver) MatrixResult {
	if noImpactBug {
		return validateNoImpactMatrix(m)
	}
	if !m.Present {
		return MatrixResult{
			State: MatrixMissing,
			Findings: []Finding{{
				Code:         CodeMatrixMissing,
				Stage:        StageArchive,
				Subject:      matrixHeading,
				Detail:       "verify-report.md no contiene la sección `" + matrixHeading + "`",
				Continuation: "ejecuta la fase `sdd-verify` para escribir `" + matrixHeading + "` en verify-report.md con una fila por escenario afectado",
			}},
		}
	}

	findings := append([]Finding{}, m.Issues...)
	affectedSet := map[ScenarioKey]bool{}
	for _, k := range affected {
		affectedSet[k] = true
	}
	seen := map[ScenarioKey]int{}
	var failed []Finding

	for _, row := range m.Rows {
		key := ScenarioKey{RequirementID: row.RequirementID, Scenario: row.Scenario}
		where := fmt.Sprintf("fila de la línea %d (%s, escenario %q)", row.Line, row.RequirementID, row.Scenario)
		invalid := func(detail string) {
			findings = append(findings, matrixIssue(fmt.Sprintf("línea %d", row.Line), where+": "+detail, continuationMatrix))
		}

		switch {
		case !affectedSet[key]:
			invalid("el escenario no figura entre los afectados")
		case seen[key] > 0:
			invalid("el escenario ya tiene otra fila (escenario duplicado)")
		}
		seen[key]++

		switch row.Result {
		case ResultPass, ResultFail:
		case "":
			invalid("el resultado está vacío; debe ser `pass` o `fail`")
		default:
			invalid(fmt.Sprintf("el resultado %q no es `pass` ni `fail`", row.Result))
		}

		switch row.Type {
		case TypeUnit, TypeIntegration, TypeE2E:
			validateAutomaticRow(row, tree, invalid)
		case TypeManual:
			validateManualRow(row, invalid)
		case "":
			invalid("el tipo está vacío; debe ser `unitario`, `integración`, `e2e` o `manual`")
		default:
			invalid(fmt.Sprintf("el tipo %q no es `unitario`, `integración`, `e2e` ni `manual`", row.Type))
		}

		if row.Result == ResultFail {
			failed = append(failed, Finding{
				Code:         CodeMatrixFailed,
				Stage:        StageArchive,
				Subject:      fmt.Sprintf("línea %d", row.Line),
				Detail:       where + ": el resultado es `fail`",
				Continuation: "corrige el comportamiento y repite la fase `sdd-verify`",
			})
		}
	}

	for _, k := range affected {
		if seen[k] == 0 {
			findings = append(findings, matrixIssue(k.RequirementID,
				fmt.Sprintf("el escenario %q del requisito %s no tiene fila en la matriz", k.Scenario, k.RequirementID),
				continuationMatrix))
		}
	}

	result := MatrixResult{State: MatrixValid}
	if len(findings) > 0 {
		result.State = MatrixInvalid
	}
	result.Findings = append(findings, failed...)
	result.Successful = result.State == MatrixValid && len(failed) == 0
	return result
}

// validateNoImpactMatrix aplica REQ-29.19: sin escenarios afectados, una matriz
// ausente o sin filas es válida y de éxito, y cualquier fila la invalida.
func validateNoImpactMatrix(m Matrix) MatrixResult {
	findings := append([]Finding{}, m.Issues...)
	for _, row := range m.Rows {
		findings = append(findings, matrixIssue(fmt.Sprintf("línea %d", row.Line),
			fmt.Sprintf("fila de la línea %d (%s, escenario %q): el escenario no figura entre los afectados, porque un bug sin impacto en la spec no tiene ninguno", row.Line, row.RequirementID, row.Scenario),
			"retira la fila: el `Motivo:` es la evidencia de un bug sin impacto en la spec"))
	}
	if len(findings) > 0 {
		return MatrixResult{State: MatrixInvalid, Findings: findings}
	}
	return MatrixResult{State: MatrixNotRequired, Successful: true}
}

func validateAutomaticRow(row MatrixRow, tree TreeResolver, invalid func(string)) {
	if detail := checkReference(row.Reference, tree); detail != "" {
		invalid(detail)
	}
	if row.Command == "" {
		invalid("no registra el comando ejecutado")
	}
	switch code, err := strconv.Atoi(row.ExitCode); {
	case row.ExitCode == "":
		invalid("no registra el código de salida")
	case err != nil:
		invalid(fmt.Sprintf("el código de salida %q no es un entero", row.ExitCode))
	case row.Result == ResultPass && code != 0:
		invalid(fmt.Sprintf("el resultado es `pass` pero el código de salida es %d: incoherente", code))
	}
}

// checkReference devuelve el detalle del problema de una referencia de prueba
// automática, o vacío si resuelve.
func checkReference(ref string, tree TreeResolver) string {
	formsHint := "las formas admitidas son `ruta/fichero:símbolo` y `TestX`"
	if ref == "" {
		return "no tiene referencia de prueba; " + formsHint
	}
	if tree == nil {
		return "no hay árbol de trabajo contra el que resolver la referencia " + fmt.Sprintf("%q", ref)
	}

	path, symbol, hasSymbol := splitReference(ref)
	switch {
	case driveReference.MatchString(ref) || strings.HasPrefix(ref, `\\`) || strings.HasPrefix(path, "/"):
		return fmt.Sprintf("la referencia %q está fuera del árbol: debe ser una ruta relativa con `/` dentro de él", ref)
	case !hasSymbol && goTestShortcut.MatchString(ref):
		ok, err := tree.ResolveGoTest(ref)
		if err != nil {
			return fmt.Sprintf("no se pudo comprobar la referencia %q: %v", ref, err)
		}
		if !ok {
			return fmt.Sprintf("la referencia %q no resuelve: ningún `*_test.go` del árbol declara la función `%s`", ref, ref)
		}
		return ""
	case !hasSymbol || path == "" || symbol == "":
		return fmt.Sprintf("la referencia %q no tiene una forma admitida; %s", ref, formsHint)
	}

	outcome, err := tree.ResolvePathSymbol(path, symbol)
	if err != nil {
		return fmt.Sprintf("no se pudo comprobar la referencia %q: %v", ref, err)
	}
	switch outcome {
	case OutcomeResolved:
		return ""
	case OutcomeFileMissing:
		return fmt.Sprintf("la referencia %q no resuelve: el fichero %q no existe en el árbol", ref, path)
	case OutcomeSymbolMissing:
		return fmt.Sprintf("la referencia %q no resuelve: el símbolo %q no figura en %q como identificador completo", ref, symbol, path)
	case OutcomeOutsideTree:
		return fmt.Sprintf("la referencia %q está fuera del árbol: la ruta %q debe ser relativa y no salir de él", ref, path)
	default:
		return fmt.Sprintf("la referencia %q devolvió el resultado desconocido %q", ref, outcome)
	}
}

// splitReference separa `ruta:símbolo` por el último `:`.
func splitReference(ref string) (path, symbol string, ok bool) {
	i := strings.LastIndex(ref, ":")
	if i < 0 {
		return ref, "", false
	}
	return ref[:i], ref[i+1:], true
}

func validateManualRow(row MatrixRow, invalid func(string)) {
	if row.Owner == "" {
		invalid("falta quién obtuvo la evidencia (Responsable)")
	}
	switch {
	case row.Date == "":
		invalid("falta cuándo se obtuvo la evidencia (Fecha, formato AAAA-MM-DD)")
	default:
		if _, err := time.Parse("2006-01-02", row.Date); err != nil {
			invalid(fmt.Sprintf("la fecha %q no es válida; el formato es AAAA-MM-DD", row.Date))
		}
	}
	if row.Rationale == "" {
		invalid("falta por qué no se automatiza (Justificación)")
	}
}
