package sddpublish

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/docpaths"
	"github.com/IGutierrezZ/axiom/v3/internal/fileprim"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// commitMessageOnly es el mensaje Conventional del commit de la publicación acotada.
const commitMessageOnly = "docs(sdd): publicar las marcas de pruebas de %s"

// validateOnly comprueba, antes de cualquier operación de git, que la petición acotada es
// coherente: Replay exige Only, Only no admite Archive y cada nombre es el exacto que acepta
// docpaths.ExactAllowed. Sin Only ni Replay devuelve nil.
func validateOnly(req Request) *Refusal {
	if len(req.Only) == 0 {
		if req.Replay == nil {
			return nil
		}
		return refuse(CodeOnlyPathRejected,
			"la reproducción solo se admite junto con la lista acotada de ficheros; no se ha escrito nada",
			"publica con `axiom sdd publish --change "+req.Change+"` o pide la publicación de las marcas desde el dashboard", nil)
	}
	if req.Archive {
		return refuse(CodeOnlyPathRejected,
			"la publicación acotada no admite el plan de archivado; no se ha escrito nada",
			"publica el archivado con `axiom sdd publish --change "+req.Change+" --archive`", nil)
	}
	for _, name := range req.Only {
		if !docpaths.ExactAllowed(name) {
			return refuse(CodeOnlyPathRejected,
				fmt.Sprintf("la ruta %q no es publicable en modo acotado: solo se admite pruebas-estado.yaml en la raíz de la carpeta del cambio; no se ha escrito nada", name),
				"publica los demás documentos con `axiom sdd publish --change "+req.Change+"`", nil)
		}
	}
	return nil
}

// onlyRun agrupa lo que el tramo acotado hereda de los pasos 1 a 5 de Run.
type onlyRun struct {
	req                Request
	result             Result
	target             branch.PublicationTarget
	set                docpaths.Set
	dir                string
	exclude            []string
	remoteBranchExists bool
	// rangeBase es la referencia (`origin/<rama>` o la de integración) que cita la continuación
	// de other_commits_unpublished.
	rangeBase string
}

// runOnly es el tramo acotado de Run (D9): confirma solo las rutas de Only, deja el resto del
// índice y del árbol intactos, avisa de los demás documentos sin confirmar y sube la rama; ante
// `push_non_fast_forward` con Replay hace un único reintento por reproducción (D10).
func (p *Publisher) runOnly(ctx context.Context, o onlyRun) Result {
	result, req, dir, target := o.result, o.req, o.dir, o.target
	refused := func(refusal *Refusal) Result {
		result.Status, result.Refusal = StatusRefused, refusal
		return result
	}
	gitFailed := func(operation string, err error) Result {
		return refused(refuse(Code(branch.CodeGitFailed),
			fmt.Sprintf("no se pudo %s: %v", operation, err),
			"comprueba que git funciona en el repositorio del cambio y repite la orden", nil))
	}

	onlyPaths := make([]string, 0, len(req.Only))
	for _, name := range req.Only {
		onlyPaths = append(onlyPaths, path.Join(target.PlanningRel, "changes", req.Change, name))
	}

	// Sucios: los de Only se confirman; los demás documentos de la lista cerrada se avisan.
	dirty, err := p.Git.DirtyPaths(ctx, dir, o.set.Candidates())
	if err != nil {
		return gitFailed("leer el estado de las rutas de documentación", err)
	}
	var onlyDirty []string
	for _, rel := range dirty {
		if slices.Contains(onlyPaths, rel) {
			onlyDirty = append(onlyDirty, rel)
			continue
		}
		if _, isDoc := o.set.Root(rel); isDoc && !o.set.LocalState(rel) {
			result.UncommittedDocs = append(result.UncommittedDocs, rel)
		}
	}
	slices.Sort(onlyDirty)
	slices.Sort(result.UncommittedDocs)

	// El rango sin subir solo puede contener commits que toquen las rutas de Only: se comprueba
	// antes de crear ningún commit, para no crear uno que no se podría subir.
	unpublished, err := p.Git.UnpublishedCommits(ctx, dir, "HEAD", o.exclude, maxUnpublishedCommits)
	if err != nil {
		return gitFailed("listar los commits sin subir", err)
	}
	if len(unpublished) > maxUnpublishedCommits {
		return refused(refuse(CodeUnpublishedRangeTooBig,
			fmt.Sprintf("hay más de %d commits sin subir en %s: no se examinan uno a uno", maxUnpublishedCommits, target.Branch),
			fmt.Sprintf("sube la rama tú con `%s` tras revisarla", pushCommand(dir, target.Branch)),
			map[string]any{"limit": maxUnpublishedCommits}))
	}
	var others []string
	for _, sha := range unpublished {
		touched, err := p.Git.CommitPaths(ctx, dir, sha)
		if err != nil {
			return gitFailed("leer las rutas del commit "+shortSHA(sha), err)
		}
		for _, rel := range touched {
			if !slices.Contains(onlyPaths, rel) {
				others = append(others, shortSHA(sha))
				break
			}
		}
	}
	if len(others) > 0 {
		return refused(otherCommitsUnpublished(dir, target.Branch, o.rangeBase, req.Change, others))
	}

	// Commit de las rutas de Only, solo si hay algo sucio.
	if len(onlyDirty) > 0 {
		identity, err := p.Git.IdentityConfigured(ctx, dir)
		if err != nil {
			return gitFailed("comprobar la identidad de git", err)
		}
		if !identity {
			return refused(refuse(CodeIdentityMissing,
				"git no tiene identidad con la que firmar el commit de las marcas; no se ha confirmado nada",
				"configura `git config user.name` y `git config user.email` en el repositorio de specs y repite la orden", nil))
		}
		message := fmt.Sprintf(commitMessageOnly, req.Change)
		sha, err := p.Git.Commit(ctx, dir, message, onlyDirty, nil)
		if err != nil {
			var failure *CommandError
			if !errors.As(err, &failure) {
				return gitFailed("confirmar las marcas", err)
			}
			return refused(classifyCommit(failure, p.commitHookPresent(ctx, dir)))
		}
		result.Commit = &CommitResult{SHA: sha, Message: message, Paths: onlyDirty}
		unpublished = append(unpublished, sha)
	}
	result.UnpublishedCommits = unpublished

	if len(unpublished) == 0 {
		result.Status, result.Refusal, result.UnpublishedCommits = StatusUpToDate, nil, nil
		result.Resume = ResumeSteps(req.Change)
		return result
	}

	prePush, err := p.Git.HookPresent(ctx, dir, "pre-push")
	if err != nil {
		prePush = false
	}
	outcome, err := p.Git.Push(ctx, dir, Remote, target.Branch, false)
	if err != nil {
		return gitFailed("subir la rama "+target.Branch, err)
	}
	if outcome.TimedOut || outcome.ExitCode != 0 {
		refusal := classifyPush(outcome, prePush, pushTimeout, target.Branch)
		if refusal.Code != CodePushNonFastForward || req.Replay == nil {
			return refused(refusal)
		}
		return p.replay(ctx, o, result, onlyPaths, prePush)
	}
	return p.published(ctx, o, result)
}

// published cierra una publicación con éxito: resuelve el commit subido y rellena Push.
func (p *Publisher) published(ctx context.Context, o onlyRun, result Result) Result {
	pushed, _, err := p.Git.ResolveCommit(ctx, o.dir, "HEAD")
	if err != nil {
		result.Status = StatusRefused
		result.Refusal = refuse(Code(branch.CodeGitFailed),
			fmt.Sprintf("no se pudo resolver el commit publicado: %v", err),
			"comprueba que git funciona en el repositorio del cambio y repite la orden", nil)
		return result
	}
	result.Status, result.Refusal = StatusPublished, nil
	result.Push = &PushResult{Refspec: "refs/heads/" + o.target.Branch + ":refs/heads/" + o.target.Branch, SHA: pushed, Created: !o.remoteBranchExists}
	result.Resume = ResumeSteps(o.req.Change)
	return result
}

// replay es el reintento único de D10 tras `push_non_fast_forward`: trae la rama (R1), lee la
// versión remota y la local (R2), fusiona (R3), mueve la rama al remoto sin pisar nada (R4),
// escribe la fusión (R5) y confirma y sube (R6). Todo fallo anterior a R4 deja el estado
// intacto; el segundo rechazo del push no tiene tercer intento y conserva el commit nuevo.
func (p *Publisher) replay(ctx context.Context, o onlyRun, result Result, onlyPaths []string, prePush bool) Result {
	dir, target, req := o.dir, o.target, o.req
	rel := onlyPaths[0]
	remoteRef := "refs/remotes/" + Remote + "/" + target.Branch
	failed := func(step, message, continuation string) Result {
		result.Status = StatusRefused
		result.Refusal = refuse(CodeReplayFailed, "reproducción interrumpida en "+step+": "+message, continuation, map[string]any{"step": step})
		return result
	}
	keepNote := "el commit de las marcas se conserva en local"

	// R1: fetch de la única rama.
	fetched, err := p.Git.Fetch(ctx, dir, Remote, target.Branch)
	if err != nil {
		return failed("R1", fmt.Sprintf("no se pudo traer %s: %v; no ha cambiado nada y %s", target.Branch, err, keepNote),
			"comprueba que git funciona y repite la publicación")
	}
	if fetched.TimedOut || fetched.ExitCode != 0 {
		result.Status, result.Refusal = StatusRefused, classifyFetch(fetched, target.Branch)
		return result
	}

	// R2: versión remota y local.
	remote, _, err := p.Git.FileAt(ctx, dir, remoteRef, rel)
	if err != nil {
		return failed("R2", fmt.Sprintf("no se pudo leer %s en %s: %v; no ha cambiado nada y %s", rel, remoteRef, err, keepNote),
			"repite la publicación")
	}
	local, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return failed("R2", fmt.Sprintf("no se pudo leer el fichero local %s: %v; no ha cambiado nada y %s", rel, err, keepNote),
			"repite la publicación")
	}

	// R3: fusión.
	merged, err := req.Replay(local, remote)
	if err != nil {
		return failed("R3", fmt.Sprintf("no se pudo fusionar la versión local con la remota: %v; no ha cambiado nada y %s", err, keepNote),
			fmt.Sprintf("corrige %s (la versión remota se lee con `git -C %s show %s:%s`) y repite la publicación", rel, pathquote.Quote(dir), remoteRef, rel))
	}

	// R4: la rama pasa al remoto conservando cualquier cambio local que no choque.
	if err := p.Git.ResetKeep(ctx, dir, remoteRef); err != nil {
		return failed("R4", fmt.Sprintf("git no pudo mover la rama a %s sin pisar cambios locales: %v; no ha cambiado nada y %s", remoteRef, err, keepNote),
			fmt.Sprintf("confirma o guarda los demás cambios del repositorio de specs (`git -C %s status`) y repite la publicación", pathquote.Quote(dir)))
	}

	// R5: escritura atómica de la fusión.
	if err := writeAtomic(filepath.Join(dir, filepath.FromSlash(rel)), merged); err != nil {
		return failed("R5", fmt.Sprintf("no se pudo escribir %s tras mover la rama: %v", rel, err),
			fmt.Sprintf("las marcas locales siguen en el commit anterior: recupéralas con `git -C %s show ORIG_HEAD:%s`", pathquote.Quote(dir), rel))
	}

	// R6: confirmar y subir, salvo que todo estuviera ya publicado.
	result.Replayed = true
	result.Commit, result.UnpublishedCommits = nil, nil
	dirty, err := p.Git.DirtyPaths(ctx, dir, []string{rel})
	if err != nil {
		return failed("R6", fmt.Sprintf("no se pudo leer el estado de %s: %v", rel, err),
			fmt.Sprintf("las marcas fusionadas están en %s; repite la publicación", rel))
	}
	if len(dirty) == 0 {
		result.Status, result.Refusal = StatusUpToDate, nil
		result.Resume = ResumeSteps(req.Change)
		return result
	}
	message := fmt.Sprintf(commitMessageOnly, req.Change)
	sha, err := p.Git.Commit(ctx, dir, message, []string{rel}, nil)
	if err != nil {
		var failure *CommandError
		if !errors.As(err, &failure) {
			return failed("R6", fmt.Sprintf("no se pudo confirmar la fusión: %v", err),
				fmt.Sprintf("las marcas fusionadas están en %s sin confirmar; repite la publicación", rel))
		}
		result.Status, result.Refusal = StatusRefused, classifyCommit(failure, p.commitHookPresent(ctx, dir))
		return result
	}
	result.Commit = &CommitResult{SHA: sha, Message: message, Paths: []string{rel}}
	result.UnpublishedCommits = []string{sha}
	outcome, err := p.Git.Push(ctx, dir, Remote, target.Branch, false)
	if err != nil {
		return failed("R6", fmt.Sprintf("no se pudo subir la rama: %v; el commit nuevo se conserva", err), "repite la publicación")
	}
	if outcome.TimedOut || outcome.ExitCode != 0 {
		result.Status, result.Refusal = StatusRefused, classifyPush(outcome, prePush, pushTimeout, target.Branch)
		return result
	}
	return p.published(ctx, o, result)
}

// classifyFetch clasifica el fallo del `git fetch` de la reproducción.
func classifyFetch(outcome PushOutcome, branchName string) *Refusal {
	tail := stderrTail(outcome.Stderr)
	if outcome.TimedOut {
		return refuse(CodeRemoteUnavailable,
			fmt.Sprintf("el fetch de %s desde origin superó el tiempo máximo de %s; no ha cambiado nada", branchName, pushTimeout),
			"comprueba la conexión y las credenciales del remoto y repite la publicación; el commit de las marcas se conserva en local",
			map[string]any{"timeout": pushTimeout.String()})
	}
	if outcome.ExitCode == 128 {
		for _, signal := range unavailableSignals {
			if strings.Contains(outcome.Stderr, signal) {
				return refuse(CodeRemoteUnavailable,
					fmt.Sprintf("origin no está disponible o no hay credenciales para traer %s: %s; no ha cambiado nada", branchName, tail),
					"comprueba la URL de `origin` y las credenciales de git ya configuradas y repite la publicación; el commit de las marcas se conserva en local",
					map[string]any{"stderr": tail})
			}
		}
	}
	return refuse(CodeReplayFailed,
		fmt.Sprintf("reproducción interrumpida en R1: el fetch de %s falló (código %d): %s; no ha cambiado nada", branchName, outcome.ExitCode, tail),
		"repite la publicación; el commit de las marcas se conserva en local",
		map[string]any{"step": "R1", "exitCode": outcome.ExitCode, "stderr": tail})
}

// otherCommitsUnpublished es la negativa de la publicación acotada cuando hay commits sin subir
// que tocan algo más que las rutas acotadas. No ha creado ningún commit.
func otherCommitsUnpublished(dir, branchName, rangeBase, change string, commits []string) *Refusal {
	log := fmt.Sprintf("git -C %s log %s..HEAD --stat", pathquote.Quote(dir), rangeBase)
	return refuse(CodeOtherCommitsUnpublished,
		fmt.Sprintf("la rama %s tiene commits sin subir (%s) que tocan algo más que las marcas de pruebas; no se ha creado ningún commit", branchName, strings.Join(commits, ", ")),
		fmt.Sprintf("revisa esos commits con `%s` y súbelos con `axiom sdd publish --change %s` (si son documentos) o con `%s` antes de repetir; las marcas siguen pendientes",
			log, change, pushCommand(dir, branchName)),
		map[string]any{"commits": commits})
}

// writeAtomic escribe content en target con un temporal con punto en la misma carpeta y
// fileprim.ReplaceAtomic, de modo que nunca queda un fichero a medias.
func writeAtomic(target string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".pruebas-estado-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := fileprim.ReplaceAtomic(name, target); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
