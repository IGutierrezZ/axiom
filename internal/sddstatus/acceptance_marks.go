package sddstatus

import (
	"context"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// publishedFileReader lee la versión publicada de un fichero del cambio (D11). Es la costura de
// las pruebas; en producción es el motor de ramas, solo con lecturas y sin red.
type publishedFileReader func(ctx context.Context, req branch.PublicationRequest, name string) (branch.PublishedChangeFile, *branch.Refusal)

// publishedMarksReader es la variable de prueba del lector: nula en producción.
var publishedMarksReader publishedFileReader

func defaultPublishedReader() (publishedFileReader, bool) {
	if publishedMarksReader != nil {
		return publishedMarksReader, true
	}
	eng := publicationEngine()
	if eng == nil {
		return nil, false
	}
	return eng.PublishedChangeFile, true
}

// AcceptanceMarksResult es el resultado común de `axiom sdd status` y del dashboard para las
// marcas de ejecución de `pruebas.md` (RF-KOF-22, prov:KOF:1): una sola lectura y un solo
// cálculo para las tres presentaciones. Ningún fallo es un error de Go.
type AcceptanceMarksResult struct {
	// Applicable es verdadero con multirepo, la función activa, `pruebas.md` y un
	// `pruebas-estado.yaml` presente. Sin él no hay nada que proyectar.
	Applicable bool
	// Path es `<changeRoot>/pruebas-estado.yaml`.
	Path string
	// Error es un fichero ilegible o que no es una secuencia, o una batería ilegible; con él no
	// hay recuentos ni pendientes.
	Error string
	// Digest es el digest actual de `pruebas.md` (kickoff.ArtifactDigest).
	Digest string
	// File son las entradas válidas, en el orden del fichero, y las inválidas aparte.
	File marks.File
	// Summary es el resumen por entorno, con Invalid ya rellenado.
	Summary marks.Summary
	// PendingKnown es verdadero si la lectura de la versión publicada fue posible.
	PendingKnown bool
	// Pending son las entradas locales ausentes de la versión publicada, en el orden local.
	Pending []marks.Entry
	// PendingRefusal es el rechazo tipado del motor cuando PendingKnown es falso (con código,
	// mensaje y continuación); nulo si el motor no está disponible.
	PendingRefusal *branch.Refusal
	// Remote y Branch son el remoto y la rama contra los que se midieron las pendientes; solo
	// con PendingKnown.
	Remote, Branch string
}

// AcceptanceMarksFor lee `pruebas-estado.yaml` del cambio, calcula el resumen contra la batería y
// el digest actuales y obtiene las pendientes de publicar contra la versión publicada, solo con
// lecturas. changeRoot es la carpeta del cambio en el planning home in situ. tests es el
// resultado de AcceptanceTestsFor para esa carpeta.
func AcceptanceMarksFor(ctx context.Context, location workspace.Location, changeRoot, change string, tests AcceptanceTestsResult) AcceptanceMarksResult {
	var result AcceptanceMarksResult
	if topology, _ := location.GoverningTopology(); topology != workspace.TopologyMultirepo || !tests.Active || !tests.Present {
		return result
	}
	result.Path = filepath.Join(changeRoot, marks.FileName)
	content, err := os.ReadFile(result.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			result.Applicable = true
			result.Error = "no se puede leer " + result.Path + ": " + err.Error()
		}
		return result
	}
	result.Applicable = true
	if tests.Error != "" {
		result.Error = tests.Error
		return result
	}
	file, err := marks.Parse(content)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	digest, err := kickoff.ArtifactDigest([]string{filepath.Join(changeRoot, acceptancetests.FileName)})
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Digest = digest
	result.File = file
	result.Summary = marks.Summarize(tests.Report.Battery, tests.Environments, file.Entries, digest)
	result.Summary.Invalid = len(file.Invalid)
	if len(file.Entries) == 0 {
		return result
	}

	read, ok := defaultPublishedReader()
	if !ok {
		return result
	}
	published, refusal := read(ctx, branch.PublicationRequest{CWD: location.WorkspaceRoot, Change: change}, marks.FileName)
	if refusal != nil {
		result.PendingRefusal = refusal
		return result
	}
	remote, err := marks.Parse(published.Content)
	if err != nil {
		// Una versión publicada ilegible no permite saber qué falta: todo lo local es pendiente.
		remote = marks.File{}
	}
	result.PendingKnown = true
	result.Remote, result.Branch = "origin", published.Target.Branch
	result.Pending = marks.Pending(file.Entries, remote.Entries)
	return result
}

// AcceptanceMarksProjection es AcceptanceTestsProjection.Marks (D12): informativa, nunca cambia
// nextRecommended, las dependencias, blockedReasons, las puertas ni sus digests.
type AcceptanceMarksProjection struct {
	Path         string             `json:"path"`
	Summary      string             `json:"summary,omitempty"`
	Environments []marks.EnvSummary `json:"environments,omitempty"`
	Orphans      int                `json:"orphans"`
	Invalid      int                `json:"invalid"`
	// Pending solo aparece si la lectura publicada fue posible; si no, PendingUnavailable.
	Pending            *int   `json:"pending,omitempty"`
	PendingUnavailable string `json:"pendingUnavailable,omitempty"`
	// Error (fichero ilegible) sustituye a los recuentos y es informativo.
	Error string `json:"error,omitempty"`
}

// projectMarks devuelve la proyección de result, o nil si no hay nada que proyectar: sin
// fichero, o con un fichero vacío o sin ninguna entrada válida (y sin error).
func projectMarks(result AcceptanceMarksResult) *AcceptanceMarksProjection {
	if !result.Applicable {
		return nil
	}
	projection := &AcceptanceMarksProjection{Path: result.Path}
	if result.Error != "" {
		projection.Error = result.Error
		return projection
	}
	if len(result.File.Entries) == 0 {
		return nil
	}
	projection.Summary = result.Summary.Text()
	projection.Environments = result.Summary.Environments
	projection.Orphans = result.Summary.Orphans
	projection.Invalid = result.Summary.Invalid
	switch {
	case result.PendingKnown:
		pending := len(result.Pending)
		projection.Pending = &pending
	case result.PendingRefusal != nil:
		projection.PendingUnavailable = string(result.PendingRefusal.Code)
	}
	return projection
}
