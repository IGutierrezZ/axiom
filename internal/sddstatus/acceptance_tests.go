package sddstatus

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TasksGateArtifactPaths returns the single set of files the shared "tasks"
// gate digest covers. Both `axiom sdd gate record` (internal/cli) and the
// status governance read (governanceArtifactInputs) call it, so the two
// digests cannot drift.
//
// The set is tasks.md, or every role's own TasksFile when the sealed roster
// declares them (kickoff.RoleTaskFiles), plus pruebas.md only when it exists and
// is not blank. Without pruebas.md the list is exactly the previous one, so the
// digest of an existing change does not change. The per-role and
// role-apply:<role> digests never include pruebas.md.
func TasksGateArtifactPaths(changeRoot string, roles []kickoff.KickoffRole) []string {
	paths := roleTaskPaths(changeRoot, roles)
	if len(paths) == 0 {
		paths = []string{filepath.Join(changeRoot, "tasks.md")}
	}
	if acceptancePath := filepath.Join(changeRoot, acceptancetests.FileName); hasContent(acceptancePath) {
		paths = append(paths, acceptancePath)
	}
	return paths
}

// AcceptanceTestsResult es el resultado común de `axiom sdd status`, de `axiom sdd tests check`
// y del dashboard (RF-KOF-21, RF-KOF-22). Ningún fallo es un error de Go: la declaración
// inválida va a DeclarationError y los fallos de lectura del cambio a Error, nunca a un informe
// parcial.
type AcceptanceTestsResult struct {
	// Active es verdadero con `acceptance_tests.enabled: true` válido en una topología admitida.
	Active bool
	// Environments es acceptance_tests.environments; solo con Active.
	Environments []string
	// DeclarationError es el error del bloque, de la topología o de axiom.yaml, con la
	// continuación `corrígela con axiom workspace validate --path <master>`. Active queda falso.
	DeclarationError string
	// Present indica que existe pruebas.md en la carpeta del cambio.
	Present bool
	// Missing indica función activa, tareas presentes y pruebas.md ausente (`pruebas_ausente`).
	Missing bool
	// Error es un kickoff ilegible, un patrón inaccesible, una delta ausente o un pruebas.md
	// ilegible. Con Error no hay informe.
	Error string
	// Report es el resultado de acceptancetests.Check; vacío sin Present o con Error.
	Report acceptancetests.Report
	// Findings son los hallazgos del informe más `pruebas_ausente` cuando Missing.
	Findings []acceptancetests.Finding
}

// AcceptanceTestsProjection es Status.AcceptanceTests. Es informativa: nunca cambia
// nextRecommended, las dependencias, blockedReasons, las puertas ni sus digests.
type AcceptanceTestsProjection struct {
	// Active es la activación efectiva; omitir la clave equivale a false.
	Active bool `json:"active"`
	// Environments solo acompaña a Active.
	Environments []string `json:"environments,omitempty"`
	// Error es el error del bloque o de la topología, con su continuación.
	Error string `json:"error,omitempty"`
	// Warning es el aviso informativo `pruebas_ausente`.
	Warning string `json:"warning,omitempty"`
	// Battery solo existe cuando el cambio seleccionado tiene pruebas.md.
	Battery *AcceptanceTestsBattery `json:"battery,omitempty"`
	// Marks es el resumen de las marcas de ejecución de `pruebas-estado.yaml` (C-06-d). Solo con
	// multirepo, la función activa, `pruebas.md` y el fichero con entradas válidas (o ilegible).
	Marks *AcceptanceMarksProjection `json:"marks,omitempty"`
}

// AcceptanceTestsBattery resume el resultado de acceptancetests.Check sobre pruebas.md.
type AcceptanceTestsBattery struct {
	Path         string                   `json:"path"`
	Cases        int                      `json:"cases"`
	Scenarios    acceptancetests.Coverage `json:"scenarios"`
	Requirements acceptancetests.Coverage `json:"requirements"`
	Findings     int                      `json:"findings"`
	// Error es el fallo de lectura del cambio; con él no hay recuentos.
	Error string `json:"error,omitempty"`
}

// AcceptanceTestsFor lee la declaración acceptance_tests de axiom.yaml y, con changeRoot, la
// batería pruebas.md del cambio. Sin changeRoot devuelve solo la declaración. Con la función no
// activa (o con un bloque inválido) y pruebas.md presente ejecuta Check igualmente, pero sin
// validar `Entornos` porque no hay dominio declarado (RF-KOF-21): el resto de comprobaciones sí.
func AcceptanceTestsFor(location workspace.Location, changeRoot string) AcceptanceTestsResult {
	var result AcceptanceTestsResult

	// Sin topología gobernante la declaración no es evaluable y no está activa: un bloque
	// `enabled: true` da el error de topología.
	topology, _ := location.GoverningTopology()
	settings, err := location.Declarations.EffectiveAcceptanceTests(topology)
	if err != nil {
		result.DeclarationError = err.Error() + "; corrígela con `axiom workspace validate --path " + pathquote.Quote(location.Master) + "`"
	} else if settings.Enabled {
		result.Active = true
		result.Environments = settings.Environments
	}
	if changeRoot == "" {
		return result
	}

	roles, kickoffErr := sealedRoles(changeRoot)
	file := filepath.Join(changeRoot, acceptancetests.FileName)
	content, readErr := os.ReadFile(file)
	switch {
	case readErr == nil:
		result.Present = true
	case errors.Is(readErr, fs.ErrNotExist):
		if result.Active && kickoffErr == nil && tasksPresent(changeRoot, roles) {
			result.Missing = true
			result.Findings = []acceptancetests.Finding{acceptancetests.MissingFinding()}
		}
		return result
	default:
		result.Error = "no se puede leer " + file + ": " + readErr.Error()
		return result
	}

	if kickoffErr != nil {
		result.Error = kickoffErr.Error()
		return result
	}
	pattern, err := location.Declarations.IDPattern()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	paths, err := ChangeArtifactPaths(changeRoot)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if len(paths.Specs) == 0 {
		result.Error = "el cambio no tiene specs delta contra las que cruzar " + acceptancetests.FileName
		return result
	}
	roleNames := make([]string, 0, len(roles))
	for _, role := range roles {
		roleNames = append(roleNames, role.Role)
	}
	result.Report = acceptancetests.Check(string(content), acceptancetests.Input{
		Delta:            readDeltaRequirements(changeRoot, paths.Specs),
		Pattern:          pattern,
		Environments:     result.Environments,
		Roles:            roleNames,
		SkipEnvironments: !result.Active,
	})
	result.Findings = result.Report.Findings
	return result
}

// sealedRoles devuelve los roles sellados del cambio; nil si no hay kickoff sellado.
func sealedRoles(changeRoot string) ([]kickoff.KickoffRole, error) {
	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		return nil, err
	}
	if sealed == nil {
		return nil, nil
	}
	return sealed.Config.Roles, nil
}

// tasksPresent indica que todos los ficheros de tareas esperados existen y no están vacíos.
// Se calcula sobre TasksGateArtifactPaths, que sin pruebas.md es solo el conjunto de tareas.
func tasksPresent(changeRoot string, roles []kickoff.KickoffRole) bool {
	for _, path := range TasksGateArtifactPaths(changeRoot, roles) {
		if !hasContent(path) {
			return false
		}
	}
	return true
}

// decorateAcceptanceTests proyecta acceptance_tests de axiom.yaml y la batería pruebas.md del
// cambio seleccionado. Es aditivo: la clave se omite salvo que axiom.yaml declare el bloque
// (válido o inválido) o sea ilegible, o que el cambio seleccionado, no archivado, tenga
// pruebas.md; omitirla equivale a active: false. No toca NextRecommended, Dependencies,
// BlockedReasons, Notes, ISO ni Governance.Gates, y por tanto tampoco los digests.
func decorateAcceptanceTests(status *Status, location workspace.Location) {
	declarations := location.Declarations
	declared := declarations.AcceptanceTests != nil || declarations.AcceptanceTestsErr != nil || declarations.ParseErr != nil

	var result AcceptanceTestsResult
	selected := status.ChangeName != nil && status.Archived == nil && status.ChangeRoot != nil
	if selected {
		result = AcceptanceTestsFor(location, *status.ChangeRoot)
	} else if declared {
		result = AcceptanceTestsFor(location, "")
	}
	if !declared && !result.Present {
		return
	}

	projection := &AcceptanceTestsProjection{
		Active:       result.Active,
		Environments: result.Environments,
		Error:        result.DeclarationError,
	}
	if result.Missing {
		projection.Warning = string(acceptancetests.CodeMissing) + ": " + acceptancetests.MissingFinding().Message
	}
	if result.Present {
		battery := &AcceptanceTestsBattery{Path: filepath.Join(*status.ChangeRoot, acceptancetests.FileName), Error: result.Error}
		if result.Error == "" {
			battery.Cases = result.Report.Cases
			battery.Scenarios = result.Report.Scenarios
			battery.Requirements = result.Report.Requirements
			battery.Findings = len(result.Report.Findings)
		}
		projection.Battery = battery
		if selected {
			projection.Marks = projectMarks(AcceptanceMarksFor(context.Background(), location, *status.ChangeRoot, *status.ChangeName, result))
		}
	} else if result.Error != "" {
		projection.Error = strings.TrimSpace(projection.Error + " " + result.Error)
	}
	status.AcceptanceTests = projection
}
