package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/docpaths"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Destino de la publicación de los documentos de un cambio (RF-PUB-03, DD-07). El dominio solo
// calcula y describe: `Engine.PublicationTarget` es una lectura (sin cerrojo, sin registro y sin
// `fetch`) y nunca ejecuta `commit` ni `push`; quien publica es `axiom sdd publish`, en su propio
// paquete (RF-PUB-07, RF-BRA-01).

// PublicationRequest es lo que `Engine.PublicationTarget` necesita de la puerta o de `publish`.
type PublicationRequest struct {
	CWD     string // workspaceRoot resuelto por la CLI, nunca el directorio del proceso
	Change  string
	Archive bool   // plan de archivado (rama `-archive` o integración in situ)
	Branch  string // opcional: la rama que autorizó la persona; si no coincide, wrong_branch
}

// PublicationTarget es el destino de la publicación: el repositorio del planning home, la rama
// que se publica y la rama de integración. Branch es siempre la rama actual del repositorio, y
// solo si el plan la permite.
type PublicationTarget struct {
	Repository        string // nombre del repositorio del planning home
	GitRoot           string // raíz git absoluta del worktree donde vive la carpeta del cambio
	Branch            string // rama que se publica: la actual, que el plan permite
	IntegrationBranch string
	InPlace           bool   // repositorio de specs de un multirepo (también tras archivar)
	PRTarget          string // BranchPlan.PRTarget de Branch, el `prTarget` de `axiom branch status`; vacío si InPlace
	Topology          workspace.TopologyType
	PlanningRel       string // planning home relativo a GitRoot, con «/»; vacío si es la raíz
}

// PublicationDetachedHead es el rechazo `detached_head`: el repositorio no está en ninguna rama y
// no hay rama que publicar. No ha escrito nada.
func PublicationDetachedHead(change, master string, repo RepoPlan) *Refusal {
	return &Refusal{
		Code:         CodeDetachedHead,
		Message:      fmt.Sprintf("%s: el repositorio %s tiene HEAD desacoplado y no hay rama que publicar; no se ha escrito nada", CodeDetachedHead, repo.Name),
		Continuation: fmt.Sprintf("pon el repositorio en la rama que corresponde con `%s` y repite la orden", ensureInvocation("ensure", change, master, LaneSDD, Exceptions{})),
		Details:      map[string]string{"change": change, "repository": repo.GitRoot},
	}
}

// PublicationIntegrationBranch es el rechazo `integration_branch`: la rama actual es la de
// integración de un repositorio que no es el de specs de un multirepo, y ahí la publicación no
// sube nada (S9). No ha escrito nada.
func PublicationIntegrationBranch(change, master string, repo RepoPlan, current string) *Refusal {
	return &Refusal{
		Code: CodeIntegrationBranch,
		Message: fmt.Sprintf("%s: el repositorio %s está en su rama de integración %s y los documentos de un cambio no se publican ahí; no se ha escrito nada",
			CodeIntegrationBranch, repo.Name, current),
		Continuation: fmt.Sprintf("pon el repositorio en la rama del cambio con `%s` y repite la orden", ensureInvocation("ensure", change, master, LaneSDD, Exceptions{})),
		Details:      map[string]string{"change": change, "repository": repo.GitRoot, "integration": current},
	}
}

// PublicationWrongBranch es el rechazo `wrong_branch` de la publicación: la rama actual no es la
// que el plan permite o no es la que autorizó la persona. current vacío significa HEAD desacoplado.
// archive dice si se pidió el plan de archivado, y flagHint nombra el flag `--archive` cuando es lo
// que falta (o sobra) para que la rama actual sea la permitida. No ha escrito nada.
func PublicationWrongBranch(change, master string, repo RepoPlan, current string, expected []string, archive bool, flagHint string) *Refusal {
	shown := current
	if shown == "" {
		shown = "HEAD desacoplado"
	}
	operation := "ensure"
	if archive {
		operation = ensureOperation + " --" + archiveRole
	}
	continuation := fmt.Sprintf("pon el repositorio en la rama que corresponde con `%s` y repite la orden", ensureInvocation(operation, change, master, LaneSDD, Exceptions{}))
	if flagHint != "" {
		continuation = flagHint + "; " + continuation
	}
	return &Refusal{
		Code: CodeWrongBranch,
		Message: fmt.Sprintf("%s: el repositorio %s está en %s y la publicación espera %s; no se ha escrito nada",
			CodeWrongBranch, repo.Name, shown, strings.Join(expected, " o ")),
		Continuation: continuation,
		Details:      map[string]string{"change": change, "repository": repo.GitRoot, "current": current, "expected": strings.Join(expected, ", ")},
	}
}

// DocPaths compone la lista cerrada de documentación del cambio (RF-PUB-01) en el repositorio del
// destino: desempaqueta `technical_context` de las declaraciones de axiom.yaml, cuyas rutas son
// relativas al workspaceRoot, y delega en `docpaths.FromWorkspace`. Es el único punto de
// composición, compartido por `axiom sdd publish` y por el estado de publicación del status.
func (t PublicationTarget) DocPaths(declarations workspace.Declarations, change string) (docpaths.Set, error) {
	var maintained, reference []string
	if technical := declarations.TechnicalContext; technical != nil {
		if technical.Maintained != nil {
			maintained = technical.Maintained.Paths
		}
		if technical.Reference != nil {
			reference = technical.Reference.Paths
		}
	}
	return docpaths.FromWorkspace(t.PlanningRel, change, maintained, reference)
}

// PublicationCodeRangeUnknown es el código con el que `axiom sdd publish` se niega cuando no existe
// la rama de integración ni en el remoto ni en local y no puede calcular qué commits faltan por
// subir. El estado de publicación del status lo proyecta como `unavailable` con el mismo código.
// Vale lo mismo que `sddpublish.CodeUnpublishedRangeUnknown`; una prueba fija la igualdad.
const PublicationCodeRangeUnknown Code = "unpublished_range_unknown"

// PublicationMaxUnpushed es el máximo de commits sin subir que se examinan uno a uno: por encima, el
// rango es demasiado grande y se deja a la persona. Es el mismo umbral que usa `axiom sdd publish`.
const PublicationMaxUnpushed = 500

// PublicationState son los hechos, de solo lectura y sin red, que dicen si hay algo que
// `axiom sdd publish` haría ahora: documentación sin confirmar o commits sin subir. Lo calcula
// `Engine.PublicationState` con las mismas reglas que `publish` (RF-PUB-01 y RF-PUB-04): la lista
// cerrada de rutas y el mismo rango de exclusiones. No decide qué hacer con ellos: eso es del status.
type PublicationState struct {
	Target        PublicationTarget
	RemoteMissing bool     // origin sin URL: no hay publicación posible
	RangeUnknown  bool     // ni origin/<integración> ni la rama de integración local
	RemoteBranch  bool     // existe refs/remotes/origin/<Branch>
	DirtyDocs     []string // documentación sin confirmar, sin estado local, ordenada
	Unpushed      int      // commits de HEAD fuera de las exclusiones de RF-PUB-04
	CodeCommits   []string // SHA cortos (7) de los commits sin subir que tocan código, del más antiguo al más reciente
	RangeTooBig   bool     // Unpushed > PublicationMaxUnpushed
}
