package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Entrega de la salida 2 de una parada («aprobar, subir y parar», REQ-26.4, REQ-26.5, DD-07 y
// DD-08). El dominio solo calcula y describe: nunca ejecuta `commit`, `push` ni abre una PR (el
// puerto Git no tiene método que publique, REQ-25.1). Cada paso es una orden exacta, un solo
// comando sin `&&` ni órdenes de plataforma (`gh`, `glab`), con `git -C <raíz git absoluta>` y
// las rutas citadas por pathquote.Quote, para que la persona la copie tal cual en Windows y en
// POSIX.

// HandOffRequest es lo que `engine.HandOff` necesita de la orden `gate record --hand-off`.
type HandOffRequest struct {
	CWD        string // workspaceRoot resuelto por la CLI, nunca el directorio del proceso
	Change     string
	ChangeRoot string // carpeta del cambio, absoluta
	Role       string // vacío en spec, design y tasks; el rol en role-apply
	// Gate y HandedOffTo componen el mensaje de commit propuesto: la puerta que se aprueba y la
	// fase (o fase de rol) que se cede.
	Gate        string
	HandedOffTo string
	// Exempt son rutas, relativas a ChangeRoot, que no cuentan como «sin confirmar»: `gates.yaml`,
	// que la orden modifica y se confirma después. Las entradas de la raíz de la carpeta cuyo
	// nombre empieza por punto (estado local que nunca se versiona) están exentas siempre.
	Exempt []string
}

// HandOffPR son los datos de la PR de una entrega, sin orden de plataforma (DD-08): la rama que
// se publica y la rama destino.
type HandOffPR struct {
	Head string `json:"head"`
	Base string `json:"base"`
}

// HandOffDelivery es la entrega de un repositorio: los pasos exactos, en orden, que la persona
// ejecuta con su propia herramienta. GitRoot es la raíz git absoluta que usan sus órdenes
// (`git -C`). InPlace marca el repositorio de specs de un multirepo, que se trabaja in situ en la
// rama de integración y por eso no lleva PR ni PRTarget.
type HandOffDelivery struct {
	Repository string     `json:"repository"`
	GitRoot    string     `json:"gitRoot"`
	Branch     string     `json:"branch"`
	PRTarget   string     `json:"prTarget,omitempty"`
	InPlace    bool       `json:"inPlace,omitempty"`
	Steps      []string   `json:"steps"`
	PR         *HandOffPR `json:"pr,omitempty"`
}

// HandOffPlan es la entrega completa de la salida 2. MergeRequired es siempre verdadero: la fase
// siguiente solo empieza cuando lo entregado está integrado (REQ-26.5).
type HandOffPlan struct {
	Topology      workspace.TopologyType `json:"topology"`
	MergeRequired bool                   `json:"mergeRequired"`
	Deliveries    []HandOffDelivery      `json:"deliveries"`
}

// HandOffCommitMessage es el mensaje de commit Conventional, en castellano, que se propone para
// el `gates.yaml` de la puerta aprobada.
func HandOffCommitMessage(gate, change, handedOffTo string) string {
	return fmt.Sprintf("chore(sdd): aprobar %s de %s y ceder %s", gate, change, handedOffTo)
}

// HandOffAddStep es el paso que prepara solo la ruta dada (pathspec explícito, nunca `-A`).
func HandOffAddStep(gitRoot, relPath string) string {
	return fmt.Sprintf("git -C %s add -- %s", pathquote.Quote(gitRoot), pathspecWord(relPath))
}

// HandOffCommitStep es el paso que confirma solo la ruta dada: pathspec explícito tras `--` y
// nunca `-a`, de modo que no arrastra nada más del árbol de trabajo.
func HandOffCommitStep(gitRoot, message, relPath string) string {
	return fmt.Sprintf("git -C %s commit -m %s -- %s", pathquote.Quote(gitRoot), pathquote.Quote(message), pathspecWord(relPath))
}

// HandOffPushStep es el paso que publica la rama concreta, nunca `HEAD`. Con setUpstream usa `-u
// origin`, como el primer push de una rama del cambio; sin él, `push origin <rama>`, como el de la
// rama de integración del repositorio de specs in situ.
func HandOffPushStep(gitRoot, branchName string, setUpstream bool) string {
	flag := ""
	if setUpstream {
		flag = "-u "
	}
	return fmt.Sprintf("git -C %s push %sorigin %s", pathquote.Quote(gitRoot), flag, pathquote.ShellWord(branchName))
}

// MergeNotice es la advertencia de fusión previa de la salida 2 (REQ-26.5): quien recoge el
// cambio no puede empezar la fase siguiente hasta que la PR esté fusionada en prTarget, porque
// `axiom branch ensure` crea la rama desde la base y no recupera trabajo sin fusionar.
func MergeNotice(prTarget string) string {
	return fmt.Sprintf("la fase siguiente solo empieza cuando la PR esté fusionada en %s: `axiom branch ensure` en otra máquina crea la rama desde la base y no recupera trabajo sin fusionar", prTarget)
}

// InPlaceNotice es la advertencia del repositorio de specs de un multirepo, que se trabaja in situ
// y no tiene PR: quien recoge el cambio necesita el push de la rama de integración.
func InPlaceNotice(integration string) string {
	return fmt.Sprintf("la fase siguiente solo empieza cuando la rama de integración %s esté publicada con este commit: este repositorio se trabaja in situ y no tiene PR", integration)
}

// UncommittedArtifacts es el rechazo `uncommitted_artifacts` de la salida 2 (REQ-26.4): hay
// artefactos del cambio sin confirmar y lo que no está en un commit no viaja en la PR. Nombra las
// rutas pendientes (relativas a la raíz git del repositorio) y no ha escrito nada; la
// continuación es confirmarlas con pathspec explícito y repetir la orden.
func UncommittedArtifacts(change string, repo RepoPlan, paths []string) *Refusal {
	const shown = 10
	listed := paths
	more := ""
	if len(paths) > shown {
		listed, more = paths[:shown], fmt.Sprintf(" y %d más", len(paths)-shown)
	}
	specs := make([]string, 0, len(paths))
	for _, path := range paths {
		specs = append(specs, pathspecWord(path))
	}
	return &Refusal{
		Code: CodeUncommittedArtifacts,
		Message: fmt.Sprintf("hay artefactos del cambio %s sin confirmar en el repositorio %s: %s%s; lo que no está en un commit no viaja en la PR y no se ha escrito nada",
			change, repo.Name, strings.Join(listed, ", "), more),
		Continuation: fmt.Sprintf("confirma esas rutas con `git -C %s add -- %s` y `git -C %s commit -m %s -- %s`, y repite la orden",
			pathquote.Quote(repo.GitRoot), strings.Join(specs, " "),
			pathquote.Quote(repo.GitRoot), pathquote.Quote("docs(sdd): confirmar los artefactos de "+change), strings.Join(specs, " ")),
		Details: map[string]string{"change": change, "repository": repo.GitRoot, "paths": strings.Join(paths, ",")},
	}
}

// HandOffWrongBranch es el rechazo `wrong_branch` que la entrega devuelve si la rama actual del
// repositorio no es ninguna de las que el plan espera para el cambio. La guarda de escritura de la
// CLI lo detecta antes; esto cierra el caso en falso para no presentar un push de la rama
// equivocada. current vacío significa HEAD desacoplado.
func HandOffWrongBranch(change, master string, repo RepoPlan, current string, expected []string) *Refusal {
	shown := current
	if shown == "" {
		shown = "HEAD desacoplado"
	}
	return &Refusal{
		Code: CodeWrongBranch,
		Message: fmt.Sprintf("%s: el repositorio %s está en %s y la entrega de la salida 2 espera %s; no se ha escrito nada",
			CodeWrongBranch, repo.Name, shown, strings.Join(expected, " o ")),
		Continuation: fmt.Sprintf("pon el repositorio en la rama que corresponde con `%s` y repite la orden", ensureInvocation("ensure", change, master, LaneSDD, Exceptions{})),
		Details:      map[string]string{"change": change, "repository": repo.GitRoot, "current": current, "expected": strings.Join(expected, ", ")},
	}
}

// pathspecWord deja una ruta relativa tal cual cuando el intérprete la lee como una palabra y, si
// lleva espacios u otros caracteres que lo parten, la cita como el resto de rutas.
func pathspecWord(path string) string {
	if pathquote.ShellWord(path) == path {
		return path
	}
	return pathquote.Quote(path)
}
