package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Ciclo de `axiom autofetch run` (RF-BRA-39, RF-HUB-07, design D-01, D-03, D-06 y D-09): lee el
// interruptor del hub por la identidad del proyecto, trae `origin` solo en el repositorio de
// specs con un intervalo mínimo entre fetch, y devuelve un resultado tipado. Nunca publica ni
// poda (no hay `push`, `pull`, `ls-remote` ni `--prune`) y nunca falla: lo que no puede hacer
// queda como aviso. Toda la ejecución va acotada por autofetchRunBudget.
//
// Puntos de extensión de las tareas siguientes, ya situados en autofetchCycle: la relevancia
// del cambio actual y los avisos (a) y (b) de D-07 (T5), y el avance por fast-forward estricto
// con la tabla de D-08 (T6). Ambos se calculan después del fetch y bajo el mismo presupuesto.

// AutofetchRequest es la petición de un ciclo: el directorio desde el que se ejecuta.
type AutofetchRequest struct{ CWD string }

// AutofetchProject es la identidad del proyecto de un ciclo, resuelta solo con ficheros.
type AutofetchProject struct {
	// Location es el proyecto que resolvió cwd.
	Location workspace.Location
	// Master es la carpeta maestra del checkout principal: la identidad del proyecto en el hub.
	Master string
	// GitRoot es la raíz git del repositorio de specs vista desde cwd (puede ser un worktree
	// enlazado) y CommonDir, su git common dir.
	GitRoot   string
	CommonDir string
	// Checkout es el checkout principal del repositorio de specs: ahí van el fetch y, más
	// adelante, las lecturas, de modo que un worktree de feature nunca se toca.
	Checkout string
}

// AutofetchProject resuelve el proyecto de cwd solo por ficheros, sin lanzar ningún proceso de
// git (D-03): `IsAxiomProject`, `Locate`, `GitRootFromFiles`, `CommonDirFromFiles` y
// `MainWorktreeFromFiles`. Así, fuera de un proyecto Axiom o con el interruptor apagado no se
// lanza git. Un `GIT_DIR` del entorno no cambia nada: git no se consulta.
func (e *Engine) AutofetchProject(cwd string) (AutofetchProject, *branch.Refusal) {
	const operation = branch.AutofetchOperation
	if reason := directoryProblem(cwd); reason != "" {
		return AutofetchProject{}, branch.UnusableDirectory(operation, reason)
	}
	if _, ok := workspace.IsAxiomProject(cwd); !ok {
		return AutofetchProject{}, &branch.Refusal{
			Code:         branch.CodeConfigInvalid,
			Message:      fmt.Sprintf("%s no pertenece a ningún proyecto Axiom", cwd),
			Continuation: "ejecuta la orden dentro de un proyecto Axiom (con axiom.yaml)",
		}
	}
	location, err := workspace.Locate(cwd)
	if err != nil {
		return AutofetchProject{}, branch.LocateRefusal(err)
	}
	root, ok := gitcommon.GitRootFromFiles(location.WorkspaceRoot)
	if !ok {
		return AutofetchProject{}, branch.NotAGitRepository(operation, location.WorkspaceRoot)
	}
	commonDir, err := gitcommon.CommonDirFromFiles(root)
	if err != nil {
		return AutofetchProject{}, &branch.Refusal{
			Code:         branch.CodeGitFailed,
			Message:      fmt.Sprintf("no se pudo localizar el directorio común de git de %s: %v", root, err),
			Continuation: "comprueba que el repositorio no está dañado",
			Details:      map[string]string{"path": root},
		}
	}
	checkout := root
	if main, known := gitcommon.MainWorktreeFromFiles(commonDir); known && directoryProblem(main) == "" {
		checkout = main
	}
	planning := branch.RepoFacts{GitRoot: root, CommonDir: commonDir}
	return AutofetchProject{
		Location:  location,
		Master:    resolveDirectory(projectMaster(location, planning)),
		GitRoot:   root,
		CommonDir: commonDir,
		Checkout:  checkout,
	}, nil
}

// Autofetch ejecuta un ciclo y devuelve su resultado. Con el interruptor apagado, el proyecto
// sin registrar o el hub ilegible no hace red ni escribe nada. Un pánico se devuelve como aviso
// del resultado parcial, nunca se propaga.
func (e *Engine) Autofetch(ctx context.Context, r AutofetchRequest) (result branch.AutofetchResult) {
	result = branch.NewAutofetchResult()
	defer func() {
		if recovered := recover(); recovered != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("el autofetch se interrumpió por un error interno: %v", recovered))
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, autofetchRunBudget)
	defer cancel()
	e.autofetchCycle(ctx, r, &result)
	return result
}

// autofetchCycle es el ciclo; escribe en result a medida que avanza, de modo que un pánico
// deja el resultado parcial.
func (e *Engine) autofetchCycle(ctx context.Context, r AutofetchRequest, result *branch.AutofetchResult) {
	project, refusal := e.AutofetchProject(r.CWD)
	if refusal != nil {
		result.Warnings = append(result.Warnings, refusal.Message)
		return
	}
	result.Project = project.Master
	if e.prefs == nil {
		return
	}
	enabled, _, warning := e.prefs.Autofetch(project.Master)
	if warning != "" {
		result.Warnings = append(result.Warnings, warning)
	}
	if !enabled {
		return
	}
	result.Autofetch = branch.AutofetchStateEnabled
	result.Repository = project.Checkout

	e.autofetchFetch(ctx, project, result)
	// Relevancia (D-07), calculada antes del avance: primero los avisos de la integración y,
	// tras los de no avance de T6 (D-08), los de las ramas del cambio.
	relevance := e.autofetchRelevance(ctx, project, result)
	result.Notices = append(result.Notices, relevance.incoming...)
	// T6: avance por fast-forward estricto y avisos de no avance (D-08), aquí.
	result.Notices = append(result.Notices, relevance.branches...)
}

// autofetchFetch trae `origin` del repositorio de specs respetando el intervalo mínimo (D-06).
// Dentro del intervalo solo se omite la red. El sello se escribe solo tras un fetch correcto;
// si no se puede escribir, es un aviso y el ciclo sigue.
func (e *Engine) autofetchFetch(ctx context.Context, project AutofetchProject, result *branch.AutofetchResult) {
	now := e.clock()
	if last, ok := readAutofetchStamp(project.CommonDir, now); ok {
		result.LastFetch = last.UTC().Format(time.RFC3339)
		if now.Sub(last) < autofetchInterval {
			result.Fetch = branch.AutofetchFetchSkippedInterval
			return
		}
	}
	configured, err := e.git.RemoteConfigured(ctx, project.Checkout, originRemote)
	if err == nil && !configured {
		result.Fetch = branch.AutofetchFetchNoOrigin
		return
	}
	if err == nil {
		err = e.writer.FetchWithin(ctx, project.Checkout, originRemote, fetchTimeoutWithin(ctx))
	}
	if err != nil {
		e.autofetchFetchFailed(project, result, err)
		return
	}
	result.Fetch = branch.AutofetchFetchDone
	result.LastFetch = now.UTC().Format(time.RFC3339)
	if err := writeAutofetchStamp(project.CommonDir, now); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudo escribir el sello del último fetch (%v): el próximo ciclo volverá a traer origin", err))
	}
}

// autofetchFetchFailed anota el fallo del fetch como aviso: `timeout` si se agotó el tiempo del
// fetch o el presupuesto del ciclo, `failed` en cualquier otro caso. El ciclo no avanza nada
// tras un fetch fallido (T6 lo respeta por el valor de result.Fetch).
func (e *Engine) autofetchFetchFailed(project AutofetchProject, result *branch.AutofetchResult, err error) {
	kind, fetch, text := branch.AutofetchNoticeFetchFailed, branch.AutofetchFetchFailed, "no pudo traer"
	if errors.Is(err, gitcommon.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		kind, fetch, text = branch.AutofetchNoticeFetchTimeout, branch.AutofetchFetchTimeout, "agotó el tiempo al traer"
	}
	result.Fetch = fetch
	result.Notices = append(result.Notices, branch.AutofetchNotice{
		Kind:    kind,
		Branch:  originRemote,
		Message: fmt.Sprintf("el autofetch %s %s en %s (%v): se sigue con las referencias locales", text, originRemote, pathquote.Quote(project.Checkout), err),
	})
}

// fetchTimeoutWithin es el tiempo máximo del fetch: autofetchFetchTimeout, o lo que quede del
// presupuesto del ciclo si es menos. Con el plazo ya vencido devuelve un instante, de modo que
// el fetch se corta de inmediato y el fallo cuenta como tiempo agotado.
func fetchTimeoutWithin(ctx context.Context) time.Duration {
	timeout := autofetchFetchTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = max(remaining, time.Nanosecond)
		}
	}
	return timeout
}
