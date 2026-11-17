package sddstatus

// ChangeArtifactPaths devuelve los ficheros de artefacto de un cambio según la regla canónica de
// `axiom sdd status`: todas las specs bajo `specs/**` (o el `spec.md` plano con contenido si no hay
// ninguna) y los ficheros de tareas por rol del kickoff sellado (o `tasks.md`). Es la regla
// compartida con el dashboard; el digest de las puertas (`internal/cli/sdd_gate_digest.go`) es
// espejo exacto de ella. Las rutas son absolutas y van en orden determinista.
func ChangeArtifactPaths(changeRoot string) (ArtifactPaths, error) {
	return resolveArtifactPaths(changeRoot)
}
