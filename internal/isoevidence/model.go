package isoevidence

import "fmt"

// Section es la sección de la delta a la que pertenece un requisito.
type Section string

// Secciones de una delta. Un requisito de la baseline no tiene sección.
const (
	SectionAdded    Section = "ADDED"
	SectionModified Section = "MODIFIED"
	SectionRemoved  Section = "REMOVED"
	SectionRenamed  Section = "RENAMED"
)

// Requirement es un requisito leído por el adaptador con el lector nativo de
// OpenSpec. El paquete no lee ficheros: recibe este modelo ya construido.
type Requirement struct {
	Domain  string
	Section Section // vacío para un requisito de la baseline
	Name    string  // en un RENAMED, el nombre nuevo
	OldName string  // solo en un RENAMED
	// Text es el bloque completo del requisito, con su encabezado
	// `### Requirement:`, tal como lo devuelve el lector nativo.
	Text string
}

// Stage es la etapa del status en la que se evalúa un hallazgo.
type Stage string

// Etapas de evaluación (design D6).
const (
	StageSpecs   Stage = "specs"
	StageTasks   Stage = "tasks"
	StageArchive Stage = "archive"
)

// Code es el token estable de un motivo tipado.
type Code string

// Lista cerrada de tokens (design §5.3).
const (
	CodeBugReportIncomplete   Code = "bug_report_incomplete"
	CodeBugNoImpactIncomplete Code = "bug_no_impact_incomplete"
	CodeDeclarationInvalid    Code = "iso_declaration_invalid"
	CodeRequirementID         Code = "iso_requirement_id"
	CodeRequirementScenarios  Code = "iso_requirement_scenarios"
	CodeCoverage              Code = "iso_coverage"
	CodeCubreUnknown          Code = "iso_cubre_unknown"
	CodeBugRequirement        Code = "iso_bug_requirement"
	CodeBugContradiction      Code = "iso_bug_contradiction"
	CodeMatrixMissing         Code = "iso_matrix_missing"
	CodeMatrixInvalid         Code = "iso_matrix_invalid"
	CodeMatrixFailed          Code = "iso_matrix_failed"
	CodeStoreUnsupported      Code = "iso_store_unsupported"
)

// AllCodes devuelve la lista cerrada de tokens, en el orden de design §5.3.
// La prueba del contrato de status la compara con la tabla publicada.
func AllCodes() []Code {
	return []Code{
		CodeBugReportIncomplete,
		CodeBugNoImpactIncomplete,
		CodeDeclarationInvalid,
		CodeRequirementID,
		CodeRequirementScenarios,
		CodeCoverage,
		CodeCubreUnknown,
		CodeBugRequirement,
		CodeBugContradiction,
		CodeMatrixMissing,
		CodeMatrixInvalid,
		CodeMatrixFailed,
		CodeStoreUnsupported,
	}
}

// Finding es un hallazgo de una regla ISO. Cada hallazgo produce un motivo.
type Finding struct {
	Code         Code
	Stage        Stage
	Subject      string // requisito, fila, fichero o identificador afectado
	Detail       string
	Continuation string
}

// Reason compone la línea de `blockedReasons`:
// `blocked(<token>): <detalle>; <continuación>`, la misma forma que
// `blocked(wrong_branch)`. Sin continuación omite el último tramo.
func (f Finding) Reason() string {
	reason := fmt.Sprintf("blocked(%s): %s", f.Code, f.Detail)
	if f.Continuation != "" {
		reason += "; " + f.Continuation
	}
	return reason
}
