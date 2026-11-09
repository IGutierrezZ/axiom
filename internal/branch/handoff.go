package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Entrega de la salida 2 de una parada («aprobar, publicar y parar», REQ-26.4, REQ-26.5, REQ-27.12,
// DD-12). El dominio solo calcula y describe: nunca ejecuta `commit`, `push` ni abre una PR (el
// puerto Git no tiene método que publique, REQ-25.1). Los documentos del cambio los sube
// `axiom sdd publish` con la invocación que lleva el plan; lo que sigue siendo de la persona es el
// push del código de los repositorios del rol, con una orden exacta, un solo comando sin `&&` ni
// órdenes de plataforma (`gh`, `glab`), con `git -C <raíz git absoluta>` y las rutas citadas por
// pathquote.Quote, para que la persona la copie tal cual en Windows y en POSIX.

// HandOffRequest es lo que `engine.HandOff` necesita de la orden `gate record --hand-off`. La
// cesión no inspecciona el trabajo del rol ni lo que está confirmado (REQ-26.4): el código sin subir
// lo informa después `axiom sdd publish`.
type HandOffRequest struct {
	CWD        string // workspaceRoot resuelto por la CLI, nunca el directorio del proceso
	Change     string
	ChangeRoot string // carpeta del cambio, absoluta
	Role       string // vacío en spec, design y tasks; el rol en role-apply
	// Gate y HandedOffTo identifican la puerta que se aprueba y la fase (o fase de rol) que se cede.
	Gate        string
	HandedOffTo string
}

// HandOffPublish es el destino de los documentos de la cesión: lo sube `axiom sdd publish` con
// Invocation. PRTarget es el `prTarget` del plan para Branch (informativo: la PR es decisión de la
// persona) y queda vacío en el repositorio de specs de un multirepo, que se trabaja in situ en la
// rama de integración y no lleva PR.
type HandOffPublish struct {
	Remote     string `json:"remote"`
	Branch     string `json:"branch"`
	PRTarget   string `json:"prTarget,omitempty"`
	InPlace    bool   `json:"inPlace,omitempty"`
	Invocation string `json:"invocation"`
}

// HandOffDelivery es la entrega de código de un repositorio del rol distinto del planning home: los
// pasos exactos, en orden, que la persona ejecuta con su propia herramienta. GitRoot es la raíz git
// absoluta que usan sus órdenes (`git -C`). PRTarget es informativo: Axiom no abre la PR.
type HandOffDelivery struct {
	Repository string   `json:"repository"`
	GitRoot    string   `json:"gitRoot"`
	Branch     string   `json:"branch"`
	PRTarget   string   `json:"prTarget,omitempty"`
	Steps      []string `json:"steps"`
}

// HandOffPlan es la entrega completa de la salida 2: dónde se publican los documentos, el push de
// código que sigue siendo de la persona y lo que ejecuta quien recoja el cambio.
type HandOffPlan struct {
	Topology   workspace.TopologyType `json:"topology"`
	Publish    HandOffPublish         `json:"publish"`
	Deliveries []HandOffDelivery      `json:"deliveries,omitempty"`
	Resume     []string               `json:"resume"`
}

// PublishRemote es el único remoto al que publica `axiom sdd publish`.
const PublishRemote = "origin"

// PublishInvocation es la orden exacta, atada a la rama con `--branch`, que publica los documentos
// del cambio (REQ-27.7). La comparten la pregunta de la puerta, `gate record` y la cesión, de modo
// que la rama que se nombra es exactamente la que se sube.
func PublishInvocation(workspaceRoot, change, branchName string) string {
	return fmt.Sprintf("axiom sdd publish --cwd %s --change %s --branch %s",
		pathquote.Quote(workspaceRoot), pathquote.ShellWord(change), pathquote.ShellWord(branchName))
}

// HandOffResume son las órdenes que ejecuta quien recoja el cambio: traer lo publicado y dejar que
// `ensure` adopte la rama (REQ-27.15). No llevan `-C`, porque la ruta de quien siga no se conoce.
func HandOffResume(change string) []string {
	return []string{"git fetch origin", fmt.Sprintf("axiom branch ensure --change %s --json", pathquote.ShellWord(change))}
}

// HandOffPushStep es el paso que publica la rama concreta, nunca `HEAD`. Con setUpstream usa `-u
// origin`, como el primer push de una rama del cambio; sin él, `push origin <rama>`.
func HandOffPushStep(gitRoot, branchName string, setUpstream bool) string {
	flag := ""
	if setUpstream {
		flag = "-u "
	}
	return fmt.Sprintf("git -C %s push %sorigin %s", pathquote.Quote(gitRoot), flag, pathquote.ShellWord(branchName))
}

// HandOffWrongBranch es el rechazo `wrong_branch` que la entrega devuelve si la rama actual del
// repositorio no es ninguna de las que el plan espera para el cambio. La guarda de escritura de la
// CLI lo detecta antes; esto cierra el caso en falso para no presentar la publicación de la rama
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
