package dashboard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Escritores guardados del dashboard y `ensure` al crear un cambio (INC-25, REQ-25.19 y
// REQ-25.32, design D-11 y D-20; decisión T4 de tasks.md). Todo escritor de la carpeta de un
// cambio llama a la guarda A (cli.GuardChangeWriter, contexto `ordinary`) antes de escribir y,
// fuera de la rama del cambio, devuelve el *branch.RefusalError de la guarda sin crear nada.
//
// El dashboard solo crea cambios por CreateIncrement: ejecuta `ensure` con el motor de la CLI y
// llama a writeIncrement con el planning home que devuelve (en modo `worktree`, el del worktree
// del cambio). La respuesta HTTP tipada de un rechazo (409) es de S10-c1; hasta entonces el
// rechazo sale como error con su código, su mensaje y su continuación.

// ensureIncrementBranch ejecuta `ensure` para el cambio name del carril SDD y devuelve el
// planning home donde hay que escribir. Sin respuesta de modo (`CreateIncrementRequest` aún no
// tiene `mode`, S10-c1) rige el modo efectivo por defecto que resuelve `ensure`: el del
// registro, el de las preferencias del hub o `worktree`. Cualquier resultado que no sea `ready`
// (incluido `decision_required`) se devuelve como *branch.RefusalError, sin escribir nada.
func (s *Service) ensureIncrementBranch(name string) (string, error) {
	cwd, err := filepath.Abs(s.getRootPath())
	if err != nil {
		return "", fmt.Errorf("no se pudo resolver la ruta del proyecto: %w", err)
	}
	result := engine.Default().Ensure(context.Background(), engine.EnsureRequest{
		CWD:    cwd,
		Change: name,
		Lane:   branch.LaneSDD,
	})
	if result.Status != branch.StatusReady {
		return "", ensureRefusal(result, name, cwd)
	}
	if result.PlanningHome == "" {
		return "", fmt.Errorf("ensure no devolvió el planning home del cambio %s; comprueba el estado con `axiom branch status --change %s --cwd %s`",
			name, pathquote.ShellWord(name), pathquote.Quote(cwd))
	}
	return result.PlanningHome, nil
}

// ensureRefusal es el rechazo tipado de un resultado de `ensure` que no está `ready`: conserva el
// código, el mensaje, la continuación, los detalles y, en `decision_required`, la decisión.
func ensureRefusal(result branch.Result, name, cwd string) *branch.RefusalError {
	refusal := branch.Refusal{
		Code:         result.Code,
		Message:      result.Message,
		Continuation: result.Continuation,
		Details:      result.Details,
		Decision:     result.Decision,
	}
	if refusal.Message == "" {
		// Un resultado sin rechazo que tampoco está `ready` (`not_started`): nada quedó preparado.
		refusal.Message = fmt.Sprintf("ensure no dejó preparado el cambio %s (estado %s)", name, result.Status)
		refusal.Continuation = fmt.Sprintf("comprueba el estado del cambio con `axiom branch status --change %s --cwd %s` y repite la creación",
			pathquote.ShellWord(name), pathquote.Quote(cwd))
	}
	return &branch.RefusalError{Refusal: refusal}
}

// writeIncrement es el escritor guardado de la creación de un incremento (REQ-25.19, D-11):
// extraído del cuerpo de escritura de CreateIncrement, escribe `proposal.md` bajo
// `<planningHome>/changes/<name>`. Consulta la guarda A antes de crear la carpeta: invocado sin
// `ensure` en una rama que no es la del cambio, se niega con `wrong_branch` y no crea nada.
func (s *Service) writeIncrement(planningHome, name string, req CreateIncrementRequest) (*CreateIncrementResponse, error) {
	activePath := filepath.Join(planningHome, "changes", name)
	if err := cli.GuardChangeWriter(activePath, name, "crear el incremento"); err != nil {
		return nil, err
	}
	if _, err := os.Stat(activePath); err == nil {
		return nil, fmt.Errorf("el incremento %q ya existe como cambio activo", name)
	}
	if err := os.MkdirAll(activePath, 0755); err != nil {
		return nil, fmt.Errorf("error al crear el directorio del incremento: %w", err)
	}

	intent := strings.TrimSpace(req.Intent)
	if intent == "" {
		intent = fmt.Sprintf("Implementación e integración de la funcionalidad %s bajo la metodología SDD.", name)
	}

	title := humanizeName(name)
	changeType := strings.TrimSpace(req.Type)
	if changeType == "" {
		changeType = "feature"
	}

	proposalContent := fmt.Sprintf(`# Propuesta: %s (%s)

## Propósito (Intent)

%s

---

## Alcance (Scope)

### Dentro de Alcance (In Scope)
- Diseño, implementación y verificación de las capacidades de %s.
- Cobertura de pruebas unitarias y validación formal de requerimientos.

### Fuera de Alcance (Out of Scope)
- Cambios no relacionados directamente con los objetivos de esta iteración.

---

## Capacidades (Capabilities)

### Nuevas Capacidades
- `+"`%s`"+`: Funcionalidad principal introducida por el cambio.

---

## Enfoque de Implementación (Approach)
1. Exploración y definición de especificaciones con escenarios BDD en spec.md.
2. Diseño técnico detallado y arquitectura en design.md.
3. Desglose y seguimiento de tareas en tasks.md.
4. Verificación formal y consolidación de documentación viva en archive.
`, title, name, intent, name, name)

	if strings.TrimSpace(req.ProposalBody) != "" {
		proposalContent = req.ProposalBody // INC-19: bytes exactos del renderizador ODD, sin normalizar [D-02]
	}

	proposalPath := filepath.Join(activePath, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte(proposalContent), 0644); err != nil {
		return nil, fmt.Errorf("error al escribir proposal.md: %w", err)
	}

	return &CreateIncrementResponse{
		Success: true,
		Name:    name,
		Path:    proposalPath,
		Message: fmt.Sprintf("Incremento %s inicializado correctamente con proposal.md", name),
	}, nil
}

// guardIncrementFolder es la guarda A de los escritores del dashboard que modifican la carpeta de
// un cambio ya existente (migración al acumulativo y handoff), en el contexto `ordinary`. Falla en
// cerrado ante un fallo interno de la consulta y no bloquea lo exento ni lo que no puede comprobar.
func guardIncrementFolder(changeRoot, change, op string) error {
	return cli.GuardChangeWriter(changeRoot, change, op)
}

// planningIntegration es la rama de integración resuelta del proyecto registrado (design D-22,
// REQ-25.23): `git.integration_branch` de su axiom.yaml o, sin él o sin poder localizar el
// espacio de trabajo, el valor por defecto de branch.ResolvePolicy. Sustituye al literal `main`
// que GetSpecsSyncStatus informaba cuando git no sabe decir la rama actual.
func (s *Service) planningIntegration() string {
	loc, err := workspace.Locate(s.getRootPath())
	if err != nil {
		return branch.ResolvePolicy(nil).IntegrationBranch
	}
	return engine.PlanningIntegration(loc)
}
