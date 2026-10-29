package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// La guarda D (REQ-25.21, design D-18): `axiom branch guard` es el hook que instala `axiom sync`
// en los runtimes con hooks. Aquí vive la decisión; la carga de cada runtime, el presupuesto de
// tiempo y el protocolo de salida están en guard_hook.go y la clasificación léxica de las rutas,
// en guard_classify.go.
//
// Decide solo con lecturas de ficheros: no lanza ni un proceso (`VerdictFor` lee `HEAD` con
// gitcommon.ReadHead), no escribe nada y falla en abierto, de modo que cualquier error interno, un
// pánico o un proyecto que no es de Axiom terminan en «no hacer nada». Solo afecta a las
// escrituras del agente a través del runtime, nunca al editor humano ni a las órdenes de la CLI.

// Acciones de una decisión de la guarda y eventos de la carga.
const (
	GuardActionNone  = "none"
	GuardActionBlock = "block"
	GuardActionWarn  = "warn"

	// GuardEventPre es el evento previo a la escritura (`PreToolUse`, `tool.execute.before`):
	// el único que bloquea, y solo en los documentos de Axiom.
	GuardEventPre = "pre"
	// GuardEventPost es el evento posterior a la escritura (`PostToolUse`, `tool.execute.after`):
	// el único que avisa, y solo en el resto de ficheros.
	GuardEventPost = "post"
)

// GuardPayload es lo que la guarda necesita de la carga de un runtime: el evento (`pre` o `post`;
// vacío si el runtime envió otro), la herramienta y las rutas que va a escribir el agente, ya
// absolutas y limpias (contra `CWD` las relativas).
type GuardPayload struct {
	Event string
	Tool  string
	Paths []string
	CWD   string
}

// GuardDecision es la decisión de la guarda: `none`, `block` o `warn` y el mensaje, en castellano,
// que el runtime entrega al agente (con la continuación ejecutable en un bloqueo).
type GuardDecision struct {
	Action  string
	Message string
}

var guardNone = GuardDecision{Action: GuardActionNone}

// Guard decide qué hace el hook con la carga (design D-18, §4.2). Nunca falla ni lanza procesos:
// con cualquier error o pánico recuperado devuelve `none`.
//
// Bloqueo (solo evento `pre` y documento de Axiom): `Locate` sobre ficheros —con `OriginStart`,
// fuera de un proyecto Axiom, nada—, contención de la ruta bajo `<planning home>/changes` o
// `<raíz del proyecto>/odd/tasks` y `VerdictFor` con el consultante `guard` evaluado en los dos
// contextos: bloquea solo si es `wrong_branch` en ambos (G-11: la guarda no conoce la fase, así que
// da por buena la rama que lo sea en cualquiera de los dos contextos; la exactitud por fase la
// garantizan las guardas A y B). Rige sin cambio registrado, con los nombres de la política (S-18).
// Cualquier otra combinación de evento y ruta no hace nada; el aviso del resto de ficheros llega
// con el evento `post` (guardWarn, en guard_warn.go): cada evento tiene una única responsabilidad,
// así que el aviso nunca se duplica.
func (e *Engine) Guard(ctx context.Context, payload GuardPayload) (decision GuardDecision) {
	decision = guardNone
	defer func() {
		if recover() != nil {
			decision = guardNone
		}
	}()
	if payload.Event == GuardEventPost {
		return e.guardWarn(ctx, payload)
	}
	if payload.Event != GuardEventPre {
		return decision
	}
	for _, path := range payload.Paths {
		if ctx.Err() != nil {
			return guardNone
		}
		document, ok := classifyAxiomDocument(path, runtime.GOOS == "windows")
		if !ok {
			continue
		}
		if blocked, ok := e.guardDocument(ctx, document); ok {
			return blocked
		}
	}
	return decision
}

// guardDocument evalúa un documento de Axiom y devuelve el bloqueo si lo hay. Un fallo en
// cualquier paso —proyecto ilegible, veredicto con error— no bloquea (falla en abierto).
func (e *Engine) guardDocument(ctx context.Context, document axiomDocument) (GuardDecision, bool) {
	loc, err := workspace.Locate(existingDirectory(document.Path))
	if err != nil || loc.Origin == workspace.OriginStart || !containsDocument(loc, document) {
		return guardNone, false
	}

	query := VerdictQuery{
		Path:       document.Path,
		Change:     document.Change,
		Lane:       document.Lane,
		Context:    branch.ContextOrdinary,
		Consultant: branch.ConsultantGuard,
	}
	ordinary, err := e.VerdictFor(ctx, query)
	if err != nil || ordinary.Kind != branch.VerdictWrongBranch {
		return guardNone, false
	}
	query.Context = branch.ContextArchive
	archive, err := e.VerdictFor(ctx, query)
	if err != nil || archive.Kind != branch.VerdictWrongBranch {
		return guardNone, false
	}
	return GuardDecision{Action: GuardActionBlock, Message: blockMessage(document, ordinary, archive)}, true
}

// containsDocument confirma que la ruta, que la clasificación léxica solo ha reconocido por sus
// segmentos, cae de verdad bajo las carpetas que `Locate` resolvió para el proyecto: la de los
// cambios del planning home o `odd/tasks` de la raíz del proyecto. Así una carpeta
// `openspec/changes/x/` de un fixture ajeno no es un documento de este proyecto.
func containsDocument(loc workspace.Location, document axiomDocument) bool {
	root := filepath.Join(loc.PlanningHome, "changes")
	if document.Lane == branch.LaneODD {
		root = filepath.Join(loc.WorkspaceRoot, "odd", "tasks")
	}
	return within(root, document.Path)
}

// within dice si path es root o cae bajo él: por identidad de directorio, que resuelve enlaces y
// nombres cortos 8.3 y exige que root exista, o, si no existe aún (el primer cambio del proyecto),
// por texto limpio, que en Windows no distingue mayúsculas.
func within(root, path string) bool {
	if pathidentity.Contains(root, path) {
		return true
	}
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// blockMessage es el motivo del bloqueo que ve el agente: nombra el documento, el porqué (el
// detalle del veredicto, con la rama actual y la esperada) y la orden de `ensure` que lo arregla;
// fuera del carril ODD añade la de archivado, porque la guarda no sabe en qué fase está el agente.
func blockMessage(document axiomDocument, ordinary, archive branch.Verdict) string {
	message := fmt.Sprintf("axiom branch guard: escritura bloqueada en %s: %s. Pon el repositorio en la rama que corresponde con `%s` y repite la escritura",
		document.Path, ordinary.Detail, ordinary.Continuation)
	if document.Lane != branch.LaneODD && archive.Continuation != "" && archive.Continuation != ordinary.Continuation {
		message += fmt.Sprintf("; si estás archivando el cambio, con `%s`", archive.Continuation)
	}
	return message + "."
}
