package reqtrace

import (
	"context"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Input son los datos de una consulta. Query y Pattern los valida quien llama con ParseID; Ref
// es la rama de integración ya validada como nombre de referencia.
type Input struct {
	Query   Query
	Pattern reqid.Pattern
	Ref     string
	// Aliases es el punto de extensión de equivalentes; nil equivale a NoAliases.
	Aliases AliasResolver
	// Legacy es el resolutor de IDs heredados (el mapa versionado); nil sin mapa.
	Legacy LegacyResolver
	// SDD tiene la raíz en el planning home (`changes/archive/...`).
	SDD Source
	// ODD tiene la raíz en `odd/tasks`; con Files y sin History se omiten sus documentos.
	ODD Source
}

// Collect responde a la consulta de solo lectura: comprueba la rama de integración en cada
// fuente, lee las trazas, resuelve los equivalentes, clasifica los incrementos SDD y ODD,
// los ancla a la línea `--first-parent` y los ordena. Un fichero ausente o mal formado es una
// advertencia y nunca aborta; una rama que no se puede leer es *RefError y un fallo de git es
// un error.
func Collect(ctx context.Context, in Input) (Result, error) {
	if err := checkRef(ctx, in.Ref, LaneSDD, in.SDD, joinPath(in.SDD.GitPrefix, "changes")); err != nil {
		return Result{}, err
	}
	if err := checkRef(ctx, in.Ref, LaneODD, in.ODD, in.ODD.GitPrefix); err != nil {
		return Result{}, err
	}

	var warnings []Warning
	folders, w := listSDDFolders(in.SDD)
	warnings = append(warnings, w...)
	traces, w := readSDDTraces(in.SDD, folders)
	warnings = append(warnings, w...)

	var docs []oddDoc
	if in.ODD.Files != nil && in.ODD.History == nil {
		warnings = append(warnings, Warning{
			Code: WarnODDRepoUnavailable, Path: in.ODD.GitPrefix,
			Message: "odd/tasks no está dentro de un repositorio git y sus documentos se omiten; muévelo al repositorio del proyecto o inicializa uno en su carpeta",
		})
	} else {
		docs, w = listODDDocs(in.ODD)
		warnings = append(warnings, w...)
		var oddTraces []traceSet
		oddTraces, w = readODDTraces(docs)
		warnings = append(warnings, w...)
		traces = append(traces, oddTraces...)
	}

	eq, w := resolveEquivalentsLegacy(in.Query, in.Pattern, traces, in.Aliases, in.Legacy)
	warnings = append(warnings, w...)

	var cands []candidate
	for _, f := range folders {
		c, w := classifySDD(in.SDD, in.Pattern, eq, f)
		cands = append(cands, c...)
		warnings = append(warnings, w...)
	}
	for _, d := range docs {
		c, w := classifyODD(in.Pattern, eq, d)
		cands = append(cands, c...)
		warnings = append(warnings, w...)
	}

	incs, w, err := anchorCandidates(ctx, in.Ref, in.SDD, in.ODD, cands)
	if err != nil {
		return Result{}, err
	}
	warnings = append(warnings, w...)
	incs, err = sortIncrements(ctx, in.SDD, in.ODD, incs)
	if err != nil {
		return Result{}, err
	}
	sortWarnings(warnings)

	result := Result{ID: in.Query.ID, ResolvedIDs: eq.resolved, IntegrationBranch: in.Ref, Increments: incs, Warnings: warnings}
	result.normalize()
	return result, nil
}
