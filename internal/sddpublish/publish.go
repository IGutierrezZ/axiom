package sddpublish

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/docpaths"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Targets calcula el destino de la publicación (DD-07): la raíz git, la rama que se publica y la
// de integración. Lo implementa `branch/engine.Engine` y es una lectura que no publica nada.
type Targets interface {
	PublicationTarget(ctx context.Context, req branch.PublicationRequest) (branch.PublicationTarget, *branch.Refusal)
}

// Publisher ejecuta el algoritmo de `axiom sdd publish` (DD-09). Se compone con un puerto de git y
// el cálculo del destino, de modo que las pruebas usan git real sobre remotos desnudos locales.
type Publisher struct {
	Git     Git
	Targets Targets
}

// Mensajes de commit Conventional, en castellano (DD-09, paso 7).
const (
	commitMessageChange  = "docs(sdd): publicar los documentos de %s"
	commitMessageArchive = "docs(sdd): publicar el archivado de %s"
)

// shownCodePaths es el máximo de rutas de código que cita `code_commits_unpublished`.
const shownCodePaths = 20

// Run publica los documentos de un cambio en `origin/<rama>` (DD-09). Los pasos 1 a 6 solo leen,
// y una negativa en ellos no escribe nada; después solo quedan dos efectos, el commit de la
// documentación y el push de la rama, y ambos se informan, también cuando una negativa posterior
// los sigue. Nunca hace `fetch`, nunca fuerza, nunca se salta hooks y nunca sube commits de
// código: los informa como `code_commits_unpublished` para que los suba la persona.
func (p *Publisher) Run(ctx context.Context, req Request) Result {
	result := Result{Schema: Schema, Change: req.Change, Context: ContextChange, Status: StatusRefused, Remote: Remote}
	if req.Archive {
		result.Context = ContextArchive
	}
	refused := func(refusal *Refusal) Result {
		result.Status, result.Refusal = StatusRefused, refusal
		return result
	}
	gitFailed := func(operation string, err error) Result {
		return refused(refuse(Code(branch.CodeGitFailed),
			fmt.Sprintf("no se pudo %s: %v", operation, err),
			"comprueba que git funciona en el repositorio del cambio y repite la orden", nil))
	}

	// 1. Almacén: solo un almacén con ficheros se publica. Sin declaración cuenta como `openspec`.
	location, err := workspace.Locate(req.CWD)
	if err != nil {
		return refused(fromBranch(branch.LocateRefusal(err)))
	}
	if store, declared := workspace.DeclaredArtifactStore(location.WorkspaceRoot); declared && (store == "engram" || store == "none") {
		return refused(refuse(CodeStoreWithoutFiles,
			fmt.Sprintf("el almacén de artefactos %q no guarda el cambio en ficheros: no hay documentos que publicar", store),
			"con este almacén no hay publicación: sigue con el cambio en esta sesión o cambia `artifact_store` a `openspec` o `hybrid`", nil))
	}

	// 2. Cambio: su carpeta existe en disco (o, con Archive, la carpeta archivada). Con Archive y sin
	// ninguna de las dos, los borrados de `changes/<x>/` que git aún ve se comprueban tras el paso 3.
	found, err := changeOnDisk(location.PlanningHome, req.Change, req.Archive)
	if err != nil {
		return gitFailed("comprobar la carpeta del cambio", err)
	}
	if !found && !req.Archive {
		// Tras archivar, `changes/<x>/` ya no existe: si el repositorio está en la rama de archivado,
		// lo que falta es el flag, y `wrong_branch` lo nombra. Es otra lectura del destino.
		if _, archiveRefusal := p.Targets.PublicationTarget(ctx, branch.PublicationRequest{CWD: req.CWD, Change: req.Change, Archive: true, Branch: req.Branch}); archiveRefusal == nil {
			if _, wrong := p.Targets.PublicationTarget(ctx, branch.PublicationRequest{CWD: req.CWD, Change: req.Change, Branch: req.Branch}); wrong != nil {
				return refused(fromBranch(wrong))
			}
		}
		return refused(changeNotFound(req.Change, location.PlanningHome))
	}

	// 3. Destino.
	target, refusal := p.Targets.PublicationTarget(ctx, branch.PublicationRequest{CWD: req.CWD, Change: req.Change, Archive: req.Archive, Branch: req.Branch})
	if refusal != nil {
		return refused(fromBranch(refusal))
	}
	result.Repository = &RepositoryRef{Name: target.Repository, GitRoot: target.GitRoot}
	result.Branch, result.InPlace = target.Branch, target.InPlace
	dir := target.GitRoot

	set, err := composeSet(location, target, req.Change)
	if err != nil {
		return refused(refuse(Code(branch.CodeConfigInvalid),
			fmt.Sprintf("las rutas de documentación declaradas en axiom.yaml no son válidas: %v", err),
			"corrige `baseline` y `technical_context` en axiom.yaml y comprueba con `axiom workspace validate`", nil))
	}
	if !found {
		deleted, err := p.Git.DirtyPaths(ctx, dir, []string{path.Join(target.PlanningRel, "changes", req.Change)})
		if err != nil {
			return gitFailed("leer el estado de la carpeta del cambio", err)
		}
		if len(deleted) == 0 {
			return refused(changeNotFound(req.Change, location.PlanningHome))
		}
	}

	// 4. Remoto: va antes del commit a propósito, porque sin `origin` no se crea ningún commit.
	configured, err := p.Git.RemoteConfigured(ctx, dir, Remote)
	if err != nil {
		return gitFailed("leer la configuración del remoto origin", err)
	}
	if !configured {
		return refused(refuse(CodeRemoteMissing,
			"el repositorio no tiene configurado el remoto origin: no hay dónde publicar; no se ha escrito nada",
			fmt.Sprintf("añade el remoto con `git -C %s remote add origin <url>` y repite la orden", pathquote.Quote(dir)), nil))
	}

	// 5. Rango sin subir: la referencia superior es la rama remota si existe y, si no, la de
	// integración remota o local (REQ-27.8). Lo que ya está en la integración remota no es trabajo
	// del cambio, así que se excluye también (una fusión de `main` en la rama no da negativas falsas).
	_, remoteBranchExists, err := p.Git.ResolveCommit(ctx, dir, "refs/remotes/origin/"+target.Branch)
	if err != nil {
		return gitFailed("resolver la rama remota", err)
	}
	remoteIntegration := "refs/remotes/origin/" + target.IntegrationBranch
	_, remoteIntegrationExists, err := p.Git.ResolveCommit(ctx, dir, remoteIntegration)
	if err != nil {
		return gitFailed("resolver la integración remota", err)
	}
	var exclude []string
	switch {
	case remoteBranchExists:
		exclude = append(exclude, "refs/remotes/origin/"+target.Branch)
		if remoteIntegrationExists {
			exclude = append(exclude, remoteIntegration)
		}
	case remoteIntegrationExists:
		exclude = append(exclude, remoteIntegration)
	default:
		localIntegration := "refs/heads/" + target.IntegrationBranch
		_, localExists, err := p.Git.ResolveCommit(ctx, dir, localIntegration)
		if err != nil {
			return gitFailed("resolver la rama de integración", err)
		}
		if !localExists {
			return refused(refuse(CodeUnpublishedRangeUnknown,
				fmt.Sprintf("no existe %s, ni en el remoto ni en local, y no se puede calcular qué commits aún no están publicados; no se ha escrito nada", target.IntegrationBranch),
				fmt.Sprintf("trae la integración con `git -C %s fetch origin` o crea la rama %s, y repite la orden", pathquote.Quote(dir), target.IntegrationBranch), nil))
		}
		exclude = append(exclude, localIntegration)
	}

	// 6. Conjunto de documentación sucio: lo que la lista cerrada acepta, sin el estado local.
	dirty, err := p.Git.DirtyPaths(ctx, dir, set.Candidates())
	if err != nil {
		return gitFailed("leer el estado de las rutas de documentación", err)
	}
	var docs []string
	roots := map[string]bool{}
	for _, rel := range dirty {
		root, isDoc := set.Root(rel)
		if !isDoc || set.LocalState(rel) {
			continue
		}
		docs = append(docs, rel)
		roots[root] = true
	}
	slices.Sort(docs)

	// 7. Commit, solo si hay documentación sucia.
	if len(docs) > 0 {
		identity, err := p.Git.IdentityConfigured(ctx, dir)
		if err != nil {
			return gitFailed("comprobar la identidad de git", err)
		}
		if !identity {
			return refused(refuse(CodeIdentityMissing,
				"git no tiene identidad con la que firmar el commit de los documentos; no se ha confirmado nada",
				"configura `git config user.name` y `git config user.email` y repite la orden", nil))
		}
		message := fmt.Sprintf(commitMessageChange, req.Change)
		if req.Archive {
			message = fmt.Sprintf(commitMessageArchive, req.Change)
		}
		sha, err := p.Git.Commit(ctx, dir, message, slices.Sorted(mapKeys(roots)), []string{set.LocalStateGlob()})
		if err != nil {
			var failure *CommandError
			if !errors.As(err, &failure) {
				return gitFailed("confirmar los documentos", err)
			}
			return refused(classifyCommit(failure, p.commitHookPresent(ctx, dir)))
		}
		result.Commit = &CommitResult{SHA: sha, Message: message, Paths: docs}
	}

	// 8. Commits sin subir: los de código no los sube Axiom. El commit del paso 7, si se creó, se
	// conserva y se informa.
	unpublished, err := p.Git.UnpublishedCommits(ctx, dir, "HEAD", exclude, maxUnpublishedCommits)
	if err != nil {
		return gitFailed("listar los commits sin subir", err)
	}
	if len(unpublished) > maxUnpublishedCommits {
		return refused(refuse(CodeUnpublishedRangeTooBig,
			fmt.Sprintf("hay más de %d commits sin subir en %s: no se examinan uno a uno", maxUnpublishedCommits, target.Branch),
			fmt.Sprintf("sube la rama tú con `%s` tras revisarla", pushCommand(dir, target.Branch)),
			map[string]any{"limit": maxUnpublishedCommits}))
	}
	result.UnpublishedCommits = unpublished
	var codeCommits, codePaths []string
	for _, sha := range unpublished {
		touched, err := p.Git.CommitPaths(ctx, dir, sha)
		if err != nil {
			return gitFailed("leer las rutas del commit "+shortSHA(sha), err)
		}
		offending := false
		for _, rel := range touched {
			if !set.Contains(rel) {
				offending = true
				if !slices.Contains(codePaths, rel) {
					codePaths = append(codePaths, rel)
				}
			}
		}
		if offending {
			codeCommits = append(codeCommits, shortSHA(sha))
		}
	}
	if len(codeCommits) > 0 {
		return refused(codeCommitsUnpublished(dir, target.Branch, codeCommits, codePaths, result.Commit != nil))
	}

	// 9. Nada que subir.
	if len(unpublished) == 0 && result.Commit == nil {
		result.Status = StatusUpToDate
		result.Refusal = nil
		result.UnpublishedCommits = nil
		result.Resume = ResumeSteps(req.Change)
		return result
	}

	// 10. Push de la rama, sin forzar ni saltarse hooks. El seguimiento solo se configura fuera de
	// `InPlace`: la integración in situ ya lo tiene.
	prePush, err := p.Git.HookPresent(ctx, dir, "pre-push")
	if err != nil {
		prePush = false
	}
	outcome, err := p.Git.Push(ctx, dir, Remote, target.Branch, !target.InPlace)
	if err != nil {
		return gitFailed("subir la rama "+target.Branch, err)
	}
	if outcome.TimedOut || outcome.ExitCode != 0 {
		return refused(classifyPush(outcome, prePush, pushTimeout, target.Branch))
	}
	pushed, _, err := p.Git.ResolveCommit(ctx, dir, "HEAD")
	if err != nil {
		return gitFailed("resolver el commit publicado", err)
	}
	result.Status, result.Refusal = StatusPublished, nil
	result.Push = &PushResult{Refspec: "refs/heads/" + target.Branch + ":refs/heads/" + target.Branch, SHA: pushed, Created: !remoteBranchExists}
	result.Resume = ResumeSteps(req.Change)
	return result
}

// commitHookPresent dice si hay un hook de git que pudo rechazar el commit.
func (p *Publisher) commitHookPresent(ctx context.Context, dir string) bool {
	for _, hook := range []string{"pre-commit", "prepare-commit-msg", "commit-msg"} {
		if present, err := p.Git.HookPresent(ctx, dir, hook); err == nil && present {
			return true
		}
	}
	return false
}

// changeOnDisk dice si la carpeta del cambio existe en el planning home o, con archive, si existe
// su carpeta archivada `changes/archive/<fecha>-<cambio>/`.
func changeOnDisk(planningHome, change string, archive bool) (bool, error) {
	if info, err := os.Stat(filepath.Join(planningHome, "changes", change)); err == nil && info.IsDir() {
		return true, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if !archive {
		return false, nil
	}
	entries, err := os.ReadDir(filepath.Join(planningHome, "changes", "archive"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	archived := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-` + regexp.QuoteMeta(change) + `$`)
	for _, entry := range entries {
		if entry.IsDir() && archived.MatchString(entry.Name()) {
			return true, nil
		}
	}
	return false, nil
}

func changeNotFound(change, planningHome string) *Refusal {
	return refuse(CodeChangeNotFound,
		fmt.Sprintf("el cambio %q no existe en %s: no hay documentos que publicar", change, planningHome),
		fmt.Sprintf("comprueba el nombre con `axiom sdd status --cwd %s` y repite la orden", pathquote.Quote(filepath.Dir(planningHome))), nil)
}

// composeSet compone la lista cerrada de documentación (DD-06) con las rutas de
// `technical_context` de axiom.yaml, que son relativas al workspaceRoot, convertidas a relativas a
// la raíz git del destino.
func composeSet(location workspace.Location, target branch.PublicationTarget, change string) (docpaths.Set, error) {
	wsRel := workspaceRelative(target.PlanningRel)
	var maintained, reference []string
	declarations := location.Declarations
	if technical := declarations.TechnicalContext; technical != nil {
		if technical.Maintained != nil {
			for _, declared := range technical.Maintained.Paths {
				if rel, ok := fromWorkspace(wsRel, declared); ok {
					maintained = append(maintained, rel)
				}
			}
		}
		if technical.Reference != nil {
			for _, declared := range technical.Reference.Paths {
				if rel, ok := fromWorkspace(wsRel, declared); ok {
					reference = append(reference, rel)
				}
			}
		}
	}
	return docpaths.New(target.PlanningRel, change, maintained, reference)
}

// workspaceRelative es el workspaceRoot relativo a la raíz git, derivado del planning home, que es
// siempre `<workspaceRoot>/openspec`: vacío si ambos coinciden con la raíz.
func workspaceRelative(planningRel string) string {
	if planningRel == "" {
		return ""
	}
	if dir := path.Dir(planningRel); dir != "." {
		return dir
	}
	return ""
}

// fromWorkspace convierte una ruta declarada, relativa al workspaceRoot, a una relativa a la raíz
// git. Las que caen fuera de la raíz git o son la propia raíz se descartan.
func fromWorkspace(wsRel, declared string) (string, bool) {
	clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(declared), `\`, "/"))
	switch {
	case clean == "." || clean == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../"):
		return "", false
	case len(clean) > 1 && clean[1] == ':':
		return "", false
	}
	joined := path.Join(wsRel, clean)
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return "", false
	}
	return joined, true
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// pushCommand es la orden que sube la rama a mano: `git -C <raíz> push origin <rama>:<rama>`.
func pushCommand(dir, branchName string) string {
	return fmt.Sprintf("git -C %s push origin %s:%s", pathquote.Quote(dir), pathquote.ShellWord(branchName), pathquote.ShellWord(branchName))
}

func codeCommitsUnpublished(dir, branchName string, commits, paths []string, committed bool) *Refusal {
	shown, more := paths, 0
	if len(paths) > shownCodePaths {
		shown, more = paths[:shownCodePaths], len(paths)-shownCodePaths
	}
	message := fmt.Sprintf("la rama %s tiene commits de código sin subir (%s) que tocan %s", branchName, strings.Join(commits, ", "), strings.Join(shown, ", "))
	if more > 0 {
		message += fmt.Sprintf(" y %d rutas más", more)
	}
	if committed {
		message += "; el commit de los documentos se conserva en local"
	}
	message += "; Axiom no sube código"
	details := map[string]any{"commits": commits, "paths": shown, "pathCount": len(paths)}
	return refuse(CodeCodeCommitsUnpublished, message,
		fmt.Sprintf("sube la rama tú, con el código revisado, con `%s`; la publicación no deshace la aprobación", pushCommand(dir, branchName)), details)
}

func mapKeys(m map[string]bool) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for key := range m {
			if !yield(key) {
				return
			}
		}
	}
}
