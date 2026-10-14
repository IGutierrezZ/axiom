package sddstatus

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// decorateWorkspaceDeclarations proyecta sobre el status lo que la resolución
// del workspace (workspace.Locate) tiene que comunicar: los avisos de Location
// (un axiom.yaml directo que no se pudo cargar y un planning home divergente de
// <maestra>/openspec) y las declaraciones válidas de axiom.yaml, `baseline:` y
// `technical_context:`, como campos aditivos y opcionales (D-05, REQ-24.19,
// REQ-24.28). Es un decorador que actúa fuera de baseStatus, de modo que la
// firma que usa status_v2_clean_break_test no cambia.
//
// Un bloque inválido no se proyecta, ni parcial ni por defecto (REQ-24.18), y sin
// declaración las dos claves no existen en el JSON: un proyecto sin axiom.yaml, o
// con uno que no declara los bloques, produce el mismo status que antes.
//
// Un aviso es informativo: nunca modifica la ruta recomendada ni las
// dependencias. Si el status ya lleva instrucciones de fase, se vuelven a
// generar para que reflejen lo añadido.
func decorateWorkspaceDeclarations(status *Status, location workspace.Location) {
	status.Baseline = projectBaseline(location)
	status.TechnicalContext = projectTechnicalContext(location)

	changed := len(location.Notes) > 0
	status.Notes = append(status.Notes, location.Notes...)
	if blockArchiveOnDeclarations(status, location) {
		changed = true
	}
	if note, ok := siblingRepositoryPathNote(status, location); ok {
		status.Notes = append(status.Notes, note)
		changed = true
	}
	archiveLine := integratePrepareLine(status, location)
	if status.PhaseInstructions == nil || (!changed && archiveLine == "") {
		return
	}
	instructions := renderPhaseInstructions(*status)
	if archiveLine != "" {
		instructions.Archive = append(instructions.Archive, archiveLine)
	}
	status.PhaseInstructions = &instructions
}

// projectBaseline devuelve la proyección de la baseline declarada, o nil si no
// hay declaración o no es válida. Las rutas son absolutas: las declaradas son
// relativas a workspaceRoot.
func projectBaseline(location workspace.Location) *BaselineProjection {
	declarations := location.Declarations
	if declarations.BaselineErr != nil || declarations.Baseline == nil {
		return nil
	}
	documents := make([]BaselineDocumentProjection, 0, len(declarations.Baseline.Documents))
	for _, document := range declarations.Baseline.Documents {
		documents = append(documents, BaselineDocumentProjection{
			Domain:      document.Domain,
			Path:        absoluteDeclaredPath(location.WorkspaceRoot, document.Path),
			Integration: string(document.EffectiveIntegration()),
		})
	}
	return &BaselineProjection{Documents: documents}
}

// projectTechnicalContext devuelve la proyección del contexto técnico declarado,
// o nil si no hay declaración o no es válida. Un modo no declarado no aparece.
func projectTechnicalContext(location workspace.Location) *TechnicalContextProjection {
	declarations := location.Declarations
	if declarations.TechnicalContextErr != nil || declarations.TechnicalContext == nil {
		return nil
	}
	projectMode := func(mode *workspace.ContextMode) *ContextModeProjection {
		if mode == nil {
			return nil
		}
		projected := &ContextModeProjection{Paths: make([]string, 0, len(mode.Paths))}
		for _, declared := range mode.Paths {
			projected.Paths = append(projected.Paths, absoluteDeclaredPath(location.WorkspaceRoot, declared))
		}
		if strings.TrimSpace(mode.Entry) != "" {
			projected.Entry = absoluteDeclaredPath(location.WorkspaceRoot, mode.Entry)
		}
		return projected
	}
	return &TechnicalContextProjection{
		Maintained: projectMode(declarations.TechnicalContext.Maintained),
		Reference:  projectMode(declarations.TechnicalContext.Reference),
	}
}

// absoluteDeclaredPath resuelve una ruta declarada, relativa a workspaceRoot y
// con «/» o «\» como separadores, a una ruta absoluta y limpia del sistema.
func absoluteDeclaredPath(workspaceRoot, declared string) string {
	slashed := strings.ReplaceAll(strings.TrimSpace(declared), `\`, "/")
	return filepath.Clean(filepath.Join(workspaceRoot, filepath.FromSlash(slashed)))
}

// contextImpactFile es el delta de contexto de un cambio con contexto técnico
// mantenido (REQ-24.31). Solo se comprueba su existencia: no se proyecta en el
// status ni altera los localizadores de artefactos (REQ-24.28).
const contextImpactFile = "context-impact.md"

// blockArchiveOnDeclarations aplica las causas de D-05 a un status con cambio
// seleccionado y no archivado: una declaración inválida o un axiom.yaml
// ilegible, la ausencia del delta de contexto y un almacén sin ficheros con una
// baseline del proyecto o un contexto mantenido. Cada causa bloquea solo el
// archivado (dependencies.archive en blocked) y se informa por dos canales:
//
//   - notes, siempre;
//   - blockedReasons, solo cuando, sin las causas, la ruta sería archive. El
//     contrato de los consumidores prohíbe apply con blockedReasons no vacío, así
//     que una declaración mal escrita no puede bloquear la planificación ni la
//     implementación (REQ-24.18). Sigue el patrón de archiveDependencyFromGovernance.
//
// Si la ruta era archive, pasa a resolve-blockers, que es lo que
// resolveNextRecommended devuelve con archive bloqueado y todo lo demás hecho.
// Devuelve si añadió alguna causa.
func blockArchiveOnDeclarations(status *Status, location workspace.Location) bool {
	if status.ChangeName == nil || status.Archived != nil {
		return false
	}
	causes := declarationCauses(status, location)
	if len(causes) == 0 {
		return false
	}
	status.Dependencies.Archive = DependencyBlocked
	status.Notes = append(status.Notes, causes...)
	if status.NextRecommended == string(PhaseArchive) {
		status.BlockedReasons = append(status.BlockedReasons, causes...)
		status.NextRecommended = "resolve-blockers"
	}
	return true
}

// declarationCauses devuelve, con los textos de cara al usuario, las causas que
// impiden archivar. Sin axiom.yaml, o con uno que no declara los bloques nuevos,
// no hay ninguna: un proyecto solo de openspec/ nunca se bloquea por esta vía (R1).
func declarationCauses(status *Status, location workspace.Location) []string {
	var causes []string
	declarations := location.Declarations

	if invalid := invalidDeclarationErrors(declarations); len(invalid) > 0 {
		causes = append(causes, fmt.Sprintf(
			"La declaración de axiom.yaml no es válida: %s. Ejecuta `axiom workspace validate --path %s` y corrígela antes de archivar.",
			strings.Join(invalid, "; "), pathquote.Quote(location.Master)))
	}

	maintained := status.TechnicalContext != nil && status.TechnicalContext.Maintained != nil
	projectBaseline := status.Baseline != nil
	if !maintained && !projectBaseline {
		return causes
	}
	store := effectiveArtifactStore(status, location)
	switch store {
	case ArtifactStoreOpenSpec, ArtifactStoreHybrid:
		if maintained && status.ChangeRoot != nil {
			delta := filepath.Join(*status.ChangeRoot, contextImpactFile)
			if _, err := os.Stat(delta); errors.Is(err, os.ErrNotExist) {
				causes = append(causes, fmt.Sprintf(
					"Falta el delta de contexto %s: registra las entradas del impacto o «Sin impacto: <motivo>».",
					pathquote.Quote(delta)))
			}
		}
	case ArtifactStoreEngram, ArtifactStoreNone:
		var subjects []string
		if projectBaseline {
			subjects = append(subjects, "la baseline del proyecto (`baseline`)")
		}
		if maintained {
			subjects = append(subjects, "el contexto técnico mantenido (`technical_context.maintained`)")
		}
		verb := "exige"
		if len(subjects) > 1 {
			verb = "exigen"
		}
		causes = append(causes, fmt.Sprintf(
			"%s %s un almacén de ficheros y el almacén es `%s`: declara `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retira la declaración.",
			capitalize(strings.Join(subjects, " y ")), verb, store))
	}
	return causes
}

// effectiveArtifactStore es el almacén de REQ-24.62: el que declara
// openspec/config.yaml y, solo si no declara ninguno, el que resolvió el status.
// El declarado prevalece porque el resolutor informa openspec para un none
// declarado cuando el cambio está en disco.
func effectiveArtifactStore(status *Status, location workspace.Location) ArtifactStore {
	root := location.WorkspaceRoot
	if root == "" {
		root = status.ActionContext.WorkspaceRoot
	}
	if declared, ok := workspace.DeclaredArtifactStore(root); ok {
		return ArtifactStore(declared)
	}
	return status.ArtifactStore
}

// invalidDeclarationErrors reúne, en líneas sueltas, las causas por las que las
// declaraciones no son válidas: un axiom.yaml ilegible o un bloque que no supera
// la validación. Con axiom.yaml ausente o sin bloques nuevos, es vacío.
func invalidDeclarationErrors(declarations workspace.Declarations) []string {
	var lines []string
	for _, err := range []error{declarations.ParseErr, declarations.BaselineErr, declarations.TechnicalContextErr} {
		if err != nil {
			lines = append(lines, errorLines(err)...)
		}
	}
	return lines
}

// errorLines aplana un error, incluidos los acumulados con errors.Join, en una
// lista de mensajes de una sola línea cada uno.
func errorLines(err error) []string {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		var lines []string
		for _, inner := range joined.Unwrap() {
			lines = append(lines, errorLines(inner)...)
		}
		return lines
	}
	return strings.Split(strings.TrimSpace(err.Error()), "\n")
}

func capitalize(text string) string {
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + text[1:]
}

// integratePrepareLine es la línea que gana phaseInstructions.archive cuando el
// cambio seleccionado tiene baseline del proyecto o contexto mantenido: nombra el
// verbo que prepara la integración. Vacía en los demás casos.
func integratePrepareLine(status *Status, location workspace.Location) string {
	if status.ChangeName == nil || status.Archived != nil {
		return ""
	}
	maintained := status.TechnicalContext != nil && status.TechnicalContext.Maintained != nil
	if status.Baseline == nil && !maintained {
		return ""
	}
	root := location.WorkspaceRoot
	if root == "" {
		root = status.ActionContext.WorkspaceRoot
	}
	return fmt.Sprintf(
		"Con baseline del proyecto o contexto técnico mantenido, prepara la integración antes de archivar: `axiom sdd integrate prepare --cwd %s --change %s`.",
		pathquote.Quote(root), *status.ChangeName)
}

// backtickedPath recoge los fragmentos entre comillas invertidas de tasks.md.
var backtickedPath = regexp.MustCompile("`([^`\\n]+)`")

// siblingRepositoryPathNote avisa (R30) de que tasks.md cita, entre comillas
// invertidas, rutas relativas cuyo primer segmento es el nombre de un repositorio
// de código hermano de workspaceRoot y que no existen bajo workspaceRoot: en
// multirepo los repositorios de código se nombran como `../<repo>/...` o con ruta
// absoluta. Es solo informativo y de solo lectura.
func siblingRepositoryPathNote(status *Status, location workspace.Location) (string, bool) {
	if status.ChangeName == nil || status.Archived != nil || location.Config == nil {
		return "", false
	}
	siblings := siblingRepositoryNames(location)
	if len(siblings) == 0 {
		return "", false
	}
	// Per-role changes keep one plan per role file, so the note reads them all.
	var tasksTexts []string
	for _, path := range status.ArtifactPaths.Tasks {
		tasksTexts = append(tasksTexts, readText(path))
	}
	tasks := strings.Join(tasksTexts, "\n")
	if tasks == "" {
		return "", false
	}

	var offending []string
	seen := map[string]bool{}
	for _, match := range backtickedPath.FindAllStringSubmatch(tasks, -1) {
		cited := strings.TrimSpace(match[1])
		slashed := strings.ReplaceAll(cited, `\`, "/")
		if !strings.Contains(slashed, "/") || strings.HasPrefix(slashed, "/") || strings.HasPrefix(slashed, ".") ||
			strings.HasPrefix(slashed, "~") || filepath.IsAbs(cited) || seen[cited] {
			continue
		}
		first := strings.SplitN(slashed, "/", 2)[0]
		if !siblings[first] {
			continue
		}
		if _, err := os.Stat(filepath.Join(location.WorkspaceRoot, filepath.FromSlash(slashed))); err == nil {
			continue
		}
		seen[cited] = true
		offending = append(offending, cited)
	}
	if len(offending) == 0 {
		return "", false
	}
	examples := offending
	if len(examples) > 3 {
		examples = examples[:3]
	}
	return fmt.Sprintf(
		"tasks.md cita %d ruta(s) relativa(s) que no existen bajo %s pero cuyo primer segmento es un repositorio de código hermano (%s). En multirepo los repositorios de código se nombran como `../<repo>/...` o con ruta absoluta.",
		len(offending), pathquote.Quote(location.WorkspaceRoot), "`"+strings.Join(examples, "`, `")+"`"), true
}

// siblingRepositoryNames devuelve los nombres de directorio de los repositorios
// de rol que son hermanos de workspaceRoot (comparten su directorio padre).
func siblingRepositoryNames(location workspace.Location) map[string]bool {
	siblings := map[string]bool{}
	parent := filepath.Dir(location.WorkspaceRoot)
	for _, role := range location.Config.Roles {
		for _, repository := range role.Repositories {
			declared := filepath.FromSlash(strings.ReplaceAll(strings.TrimSpace(repository.Path), `\`, "/"))
			if declared == "" {
				continue
			}
			path := declared
			if !filepath.IsAbs(path) {
				path = filepath.Join(location.Master, path)
			}
			path = filepath.Clean(path)
			if path == filepath.Clean(location.WorkspaceRoot) {
				continue
			}
			if filepath.Dir(path) == parent || pathidentity.SameDirectory(filepath.Dir(path), parent) {
				siblings[filepath.Base(path)] = true
			}
		}
	}
	return siblings
}
