package acceptancetests

import (
	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// FileName es el nombre del artefacto de la batería dentro de la carpeta del cambio.
const FileName = "pruebas.md"

// Code es el token estable de un hallazgo (RF-KOF-20).
type Code string

// Códigos de hallazgo de RF-KOF-20.
const (
	CodeMissing            Code = "pruebas_ausente"
	CodeInvalidTable       Code = "tabla_invalida"
	CodeInvalidCell        Code = "celda_invalida"
	CodeInvalidCaseID      Code = "id_caso_invalido"
	CodeDuplicateCaseID    Code = "id_caso_duplicado"
	CodeNoCubre            Code = "caso_sin_cubre"
	CodeInvalidEnvironment Code = "entorno_invalido"
	CodeInvalidRole        Code = "rol_invalido"
	CodeUnknownID          Code = "id_desconocido"
	CodeUnknownScenario    Code = "escenario_desconocido"
	CodeRequirementNoID    Code = "requisito_sin_id"
	CodeDuplicateDeltaID   Code = "id_duplicado_en_delta"
	CodeAmbiguousScenario  Code = "escenario_ambiguo"
	CodeUncoveredReq       Code = "requisito_sin_cubrir"
	CodeUncoveredScenario  Code = "escenario_sin_cubrir"
)

// columns son las nueve columnas de la tabla, con su nombre exacto (RF-KOF-18).
var columns = []string{
	"ID", "Cubre", "Rol", "Tipo", "Prioridad", "Entornos",
	"Precondiciones", "Pasos", "Resultado esperado",
}

// CubreEntry es una entrada de `Cubre`. Scenario vacío cita el requisito entero. Una entrada
// vacía (`;;` o `ID#` sin nombre) se representa con ID vacío.
type CubreEntry struct{ ID, Scenario string }

// Case es una fila de la tabla.
type Case struct {
	Line          int
	ID            string
	Role          string
	Type          string
	Priority      string
	Preconditions string
	Steps         string
	Expected      string
	Cubre         []CubreEntry
	CubreRaw      string
	Environments  []string
}

// Battery es el resultado de parsear `pruebas.md`: los casos y los hallazgos de forma.
type Battery struct {
	Cases  []Case
	Issues []Finding
}

// Finding es un hallazgo. Los de la batería llevan File = FileName; los de la delta,
// `specs/<dominio>/spec.md` con Line 0.
type Finding struct {
	Code    Code
	File    string
	Line    int
	Subject string
	Message string
}

// Coverage es un recuento cubiertos sobre total.
type Coverage struct {
	Covered int `json:"covered"`
	Total   int `json:"total"`
}

// Input es lo que Check recibe ya leído por el adaptador.
type Input struct {
	// Delta son los requisitos de las specs delta; Check solo usa ADDED y MODIFIED.
	Delta []isoevidence.Requirement
	// Pattern es el patrón de identificadores del proyecto.
	Pattern reqid.Pattern
	// Environments son los entornos declarados en acceptance_tests.environments.
	Environments []string
	// Roles son los roles sellados del cambio.
	Roles []string
	// SkipEnvironments omite la validación de `Entornos` (función no activa: no hay dominio).
	SkipEnvironments bool
}

// Report es el informe de Check.
type Report struct {
	Battery      Battery
	Cases        int
	Scenarios    Coverage
	Requirements Coverage
	Findings     []Finding
}

// OK indica que no hay ningún hallazgo.
func (r Report) OK() bool { return len(r.Findings) == 0 }

// MissingFinding es el hallazgo `pruebas_ausente`, que Check nunca emite: lo añade el
// adaptador cuando la función está activa, hay tareas y falta el fichero.
func MissingFinding() Finding {
	return Finding{
		Code:    CodeMissing,
		File:    FileName,
		Message: "falta " + FileName + ": la función está activa y el cambio ya tiene tareas",
	}
}
