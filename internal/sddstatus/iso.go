package sddstatus

import (
	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
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
