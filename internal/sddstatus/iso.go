package sddstatus

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// bugProfile es el resultado de leer el perfil de bug sin delta de proposal.md
// (REQ-29.15, design D9). Es el adaptador entre el status y isoevidence: este
// paquete lee el fichero y isoevidence solo analiza texto.
type bugProfile struct {
	// declaration es lo que declara proposal.md. Con Kind "none" el estado es
	// idéntico al anterior a este cambio.
	declaration isoevidence.BugDeclaration
	// findings son los motivos de una declaración incompleta, uno por cada cosa
	// que falta.
	findings []isoevidence.Finding
}

// declared dice si proposal.md declara alguna de las dos secciones del perfil.
func (b bugProfile) declared() bool {
	return b.declaration.Kind != isoevidence.BugKindNone
}

// satisfiesSpecs dice si la declaración, completa, cumple el artefacto de specs
// sin delta.
func (b bugProfile) satisfiesSpecs() bool {
	return b.declared() && b.declaration.Valid
}

// evaluateBugProfile aplica D9: solo cuando el artefacto de specs está `missing`
// (sin delta ni ficheros en specs/) se analiza proposal.md. Con delta (`done` o
// `partial`) las secciones se ignoran sin aviso (regla 1 de REQ-29.15). Vale en
// todo proyecto, sea o no ISO.
func evaluateBugProfile(specs ArtifactState, proposalPaths []string) bugProfile {
	if specs != ArtifactMissing {
		return bugProfile{declaration: isoevidence.BugDeclaration{Kind: isoevidence.BugKindNone}}
	}
	declaration := isoevidence.ParseBugDeclaration(readText(firstPath(proposalPaths)))
	return bugProfile{declaration: declaration, findings: declaration.Findings()}
}

// bugDeclaredWithoutDelta dice si un cambio sin ningún fichero de specs declara
// un bug completo en proposal.md: el criterio con el que la compuerta `spec`
// juzga proposal.md (D20) y con el que lo mismo calcula `axiom sdd gate record`.
func bugDeclaredWithoutDelta(specPaths, proposalPaths []string) bool {
	if len(specPaths) != 0 {
		return false
	}
	return evaluateBugProfile(ArtifactMissing, proposalPaths).satisfiesSpecs()
}

// findingReasons compone una línea de blockedReasons por hallazgo (REQ-29.16).
func findingReasons(findings []isoevidence.Finding) []string {
	reasons := make([]string, 0, len(findings))
	for _, finding := range findings {
		reasons = append(reasons, finding.Reason())
	}
	return reasons
}

// Valores de ISOProjection.Project y ISOProjection.Change (design §5.2).
const (
	isoValueISO     = "iso"
	isoValueNotISO  = "not_iso"
	isoValueInvalid = "invalid"
)

// ISOProjection es la proyección aditiva de REQ-29.2. Se omite cuando axiom.yaml
// no tiene bloque `sdd:` y el cambio seleccionado no es ISO, de modo que un
// consumidor anterior no observa ninguna diferencia.
type ISOProjection struct {
	// Project: "iso" (sdd.iso: true), "not_iso" (false o bloque sin la clave) o
	// "invalid" (bloque inválido, contradictorio con workflow.lane, o axiom.yaml
	// ilegible).
	Project string `json:"project"`
	// Reason: solo con Project "invalid"; el error y su continuación. Está
	// presente con o sin cambio seleccionado.
	Reason string `json:"reason,omitempty"`
	// Change: "iso" o "not_iso", leído de la instantánea de kickoff.yaml. Ausente
	// sin cambio seleccionado.
	Change string `json:"change,omitempty"`
	// Matrix: solo en un cambio ISO con apply terminado.
	Matrix *ISOMatrixProjection `json:"matrix,omitempty"`
}

// ISOMatrixProjection es el resultado de validar la matriz de verify de un cambio
// ISO, para que el orquestador decida si el verify es limpio sin leer el informe.
type ISOMatrixProjection struct {
	// State: "valid", "invalid", "missing" o "not_required" (bug sin impacto sin
	// matriz).
	State string `json:"state"`
	// Successful: la matriz es válida (o no requerida) y ninguna fila falla.
	Successful bool `json:"successful"`
	// Findings es el número de hallazgos; el detalle va en blockedReasons.
	Findings int `json:"findings"`
}

// isoDeclaration es lo que decorateISO sabe de la declaración `sdd:` del proyecto.
type isoDeclaration struct {
	// invalidText es el error de la declaración sin su continuación; vacío si es
	// válida. Una declaración inválida falla cerrada (D8): nunca se asume no ISO.
	invalidText string
	// contradiction marca la contradicción con workflow.lane (D18), cuyo error ya
	// lleva su corrección.
	contradiction bool
	// declared dice si hay un bloque `sdd:` válido; iso es su valor.
	declared, iso bool
}

// readISODeclaration clasifica Declarations.SDD, SDDErr y ParseErr. Un axiom.yaml
// ilegible (ParseErr) cuenta como inválido: no se puede saber si declara el
// bloque, así que la proyección falla cerrada.
func readISODeclaration(declarations workspace.Declarations) isoDeclaration {
	switch {
	case declarations.SDDErr != nil:
		text := strings.Join(errorLines(declarations.SDDErr), "; ")
		contradiction := errors.Is(declarations.SDDErr, workspace.ErrISOWorkflowContradiction)
		if contradiction {
			text = strings.TrimSuffix(text, ": "+workspace.ErrISOWorkflowContradiction.Error())
		}
		return isoDeclaration{invalidText: text, contradiction: contradiction}
	case declarations.ParseErr != nil:
		return isoDeclaration{invalidText: strings.Join(errorLines(declarations.ParseErr), "; ")}
	case declarations.SDD != nil:
		return isoDeclaration{declared: true, iso: declarations.SDD.ISO}
	}
	return isoDeclaration{}
}

// reason es ISOProjection.Reason: el error y, salvo en la contradicción (cuyo
// texto ya nombra las dos correcciones), la continuación.
func (d isoDeclaration) reason(master string) string {
	if d.contradiction {
		return d.invalidText
	}
	return fmt.Sprintf("%s; corrígela con `axiom workspace validate --path %s`", d.invalidText, pathquote.Quote(master))
}

// finding es el hallazgo `iso_declaration_invalid` que estrecha el archivado.
func (d isoDeclaration) finding(master string) isoevidence.Finding {
	return isoevidence.Finding{
		Code:         isoevidence.CodeDeclarationInvalid,
		Stage:        isoevidence.StageArchive,
		Subject:      "sdd",
		Detail:       "la declaración `sdd:` de axiom.yaml no es válida: " + d.invalidText,
		Continuation: "ejecuta `axiom workspace validate --path " + pathquote.Quote(master) + "` y corrígela",
	}
}

// note es la línea de `notes` de D8, que se añade con o sin cambio seleccionado.
func (d isoDeclaration) note(master string) string {
	return fmt.Sprintf(
		"La declaración sdd: de axiom.yaml no es válida: %s. Corrígela con axiom workspace validate --path %s; mientras tanto ningún cambio puede archivarse.",
		d.invalidText, pathquote.Quote(master))
}

// changeKickoff devuelve el kickoff sellado del cambio seleccionado, o nil sin
// cambio, con el cambio archivado o sin sellar. El resolutor OpenSpec ya lo
// cargó en Governance; en los almacenes sin ficheros se lee de la carpeta del
// cambio (D21). Un kickoff ilegible no se proyecta aquí: el resolutor OpenSpec ya
// falla con su error nombrado (T-10).
func changeKickoff(status *Status) *kickoff.Kickoff {
	if status.ChangeName == nil || status.Archived != nil {
		return nil
	}
	if status.Governance != nil {
		sealed := status.Governance.Kickoff
		return &sealed
	}
	root := ""
	switch {
	case status.ChangeRoot != nil:
		root = *status.ChangeRoot
	case status.PlanningHome.Path != "":
		root = filepath.Join(status.PlanningHome.Path, "changes", *status.ChangeName)
	default:
		return nil
	}
	sealed, err := kickoff.Load(root)
	if err != nil {
		return nil
	}
	return sealed
}

// decorateISO proyecta la declaración ISO del proyecto y la instantánea del
// cambio (REQ-29.1, REQ-29.2, REQ-29.13), y estrecha el archivado ante una
// declaración inválida (D8) o un almacén sin ficheros (D21). Es aditivo: sin
// bloque `sdd:` y con un cambio no ISO no escribe nada, y el status de un cambio
// no ISO es idéntico al de antes.
//
// Las reglas del cambio se leen de su instantánea y nunca de axiom.yaml en vivo.
// Cada hallazgo de archivado va a `notes` y, solo cuando la ruta sería archive, a
// `blockedReasons` (el contrato de los consumidores prohíbe apply con motivos).
func decorateISO(status *Status, location workspace.Location) {
	declaration := readISODeclaration(location.Declarations)
	sealed := changeKickoff(status)
	changeISO := sealed != nil && sealed.IsISO()
	invalid := declaration.invalidText != ""
	if !declaration.declared && !invalid && !changeISO {
		return
	}

	projection := &ISOProjection{Project: isoValueNotISO}
	switch {
	case invalid:
		projection.Project = isoValueInvalid
		projection.Reason = declaration.reason(location.Master)
	case declaration.iso:
		projection.Project = isoValueISO
	}
	if status.ChangeName != nil && status.Archived == nil {
		projection.Change = isoValueNotISO
		if changeISO {
			projection.Change = isoValueISO
		}
	}
	projection.Matrix = status.isoMatrix
	status.ISO = projection

	changed := false
	if invalid {
		status.Notes = append(status.Notes, declaration.note(location.Master))
		changed = true
	} else if sealed != nil && changeISO != declaration.iso {
		status.Notes = append(status.Notes, fmt.Sprintf(
			"La instantánea ISO del cambio (sdd.iso: %t) difiere de la declaración vigente del proyecto (sdd.iso: %t); las reglas del cambio siguen su instantánea.",
			changeISO, declaration.iso))
		changed = true
	}

	if status.ChangeName != nil && status.Archived == nil {
		var findings, noted []isoevidence.Finding
		if invalid {
			findings = append(findings, declaration.finding(location.Master))
		}
		if changeISO {
			switch store := effectiveArtifactStore(status, location); store {
			case ArtifactStoreEngram, ArtifactStoreNone:
				finding := isoevidence.Finding{
					Code:         isoevidence.CodeStoreUnsupported,
					Stage:        isoevidence.StageArchive,
					Subject:      string(store),
					Detail:       fmt.Sprintf("el cambio es ISO y el almacén de artefactos es `%s`, que no guarda ficheros: no se puede evaluar la matriz de verificación", store),
					Continuation: "declara `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`",
				}
				findings = append(findings, finding)
				noted = append(noted, finding)
			}
		}
		if len(findings) > 0 {
			narrowArchive(status, findings, noted)
			changed = true
		}
	}

	if changed && status.PhaseInstructions != nil {
		refreshPhaseInstructions(status, location)
	}
}

// narrowArchive estrecha `archive` a `blocked` y, solo cuando la ruta sería
// archive, vuelca los motivos en blockedReasons y la lleva a resolve-blockers
// (lo que resolveNextRecommended devuelve con archive bloqueado y todo lo demás
// hecho). Los hallazgos de noted además van a `notes`.
func narrowArchive(status *Status, findings, noted []isoevidence.Finding) {
	status.Dependencies.Archive = DependencyBlocked
	status.Notes = append(status.Notes, findingReasons(noted)...)
	if status.NextRecommended != string(PhaseArchive) {
		return
	}
	status.BlockedReasons = append(status.BlockedReasons, findingReasons(findings)...)
	status.NextRecommended = "resolve-blockers"
}

// refreshPhaseInstructions vuelve a generar las instrucciones de fase para que
// reflejen la ruta y las notas añadidas, conservando la línea de archivado de la
// integración que decorateWorkspaceDeclarations ya había añadido.
func refreshPhaseInstructions(status *Status, location workspace.Location) {
	instructions := renderPhaseInstructions(*status)
	if line := integratePrepareLine(status, location); line != "" {
		instructions.Archive = append(instructions.Archive, line)
	}
	status.PhaseInstructions = &instructions
}
