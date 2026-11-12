package sddstatus

import "github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"

// OpenSpecComposeSchema names the sdd-archive canonical composition contract
// exposed to the sdd-archive skill and its CLI wrapper.
const OpenSpecComposeSchema = "gentle-ai.sdd-archive-compose/v1"

// UnappliedDeltaError names exactly one delta requirement that
// ComposeOpenSpecCanonicalSpec could not apply. Callers MUST refuse — never
// report success on a partial composition (#4119). El análisis y la composición
// viven en internal/openspecdoc (REQ-30.22 a REQ-30.24); este es un alias de tipo.
type UnappliedDeltaError = openspecdoc.UnappliedDeltaError

// ComposeOptions configura el comportamiento opcional de poda progresiva y archivado (ODD-4.1).
type ComposeOptions = openspecdoc.ComposeOptions

// ComposeOpenSpecCanonicalSpec merges an OpenSpec delta spec into a canonical
// spec's byte content; see openspecdoc.Compose.
func ComposeOpenSpecCanonicalSpec(canonical, delta string) (string, error) {
	return openspecdoc.Compose(canonical, delta, ComposeOptions{})
}

// ComposeOpenSpecCanonicalSpecWithOptions soporta opciones adicionales como la
// preservación de requisitos deprecados ([SUPERSEDED / DEPRECADO]) para poda progresiva.
func ComposeOpenSpecCanonicalSpecWithOptions(canonical, delta string, opts ComposeOptions) (string, error) {
	return openspecdoc.Compose(canonical, delta, opts)
}
