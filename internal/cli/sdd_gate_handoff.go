package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// gateRecordSchema es el esquema del `--json` de `axiom sdd gate record` (§4.4 del diseño): se
// emite en todas las salidas de la compuerta, no solo en la de la cesión.
const gateRecordSchema = "axiom.sdd-gate-record/v1"

// Comprobaciones de la salida 2 de una parada, `gate record --hand-off` (REQ-26.1, REQ-26.3,
// REQ-26.6; DD-02, DD-12). runSDDGateRecord las llama en dos puntos, siempre ANTES de escribir
// gates.yaml: requireHandOffStops tras cargar el kickoff (paso 4) y prepareHandOff una vez
// resuelto el digest de la puerta (pasos 6 a 9). Todas son lecturas y cada rechazo nombra la orden
// ejecutable que sigue siendo válida; ninguna deja rastro en el disco.

// requireHandOffStops es el paso 4 de DD-02: la cesión solo existe en un cambio sellado con
// paradas (`checkpointed`). En uno continuo, o sin sellar, no hay parada que ceder.
func requireHandOffStops(parsed kickoff.GateRecordArgs, sealed *kickoff.Kickoff, workspaceRoot string) error {
	if sealed != nil && sealed.Config.ExecutionStyle == kickoff.ExecutionCheckpointed {
		return nil
	}
	return fmt.Errorf("el cambio %q no tiene paradas: `--hand-off` solo existe en un cambio sellado con `--execution-style checkpointed`, y este es continuo o no está sellado; registra la decisión sin cederla con `%s`",
		parsed.Change, gateApprovedInvocation(parsed, workspaceRoot))
}

// handOffPrepared es lo que las comprobaciones de la salida 2 dejan listo para escribir y
// presentar: la fase que se cede y la entrega calculada por el motor de ramas.
type handOffPrepared struct {
	handedOffTo string
	plan        branch.HandOffPlan
	parsed      kickoff.GateRecordArgs
	workspace   string
}

// prepareHandOff implementa los pasos 6 a 9 de DD-02: almacén con ficheros, artefacto presente,
// puerta aún sin aprobar y los datos de entrega. Ya no comprueba qué está confirmado (REQ-26.4,
// DD-12): los documentos pendientes los confirma `axiom sdd publish`, que informa del código sin subir.
func prepareHandOff(parsed kickoff.GateRecordArgs, loc workspace.Location, changeRoot string, roster []multirole.RoleAssignment, digest string) (*handOffPrepared, error) {
	workspaceRoot := loc.WorkspaceRoot

	// Sin declaración cuenta como `openspec`: la carpeta del cambio existe en disco, que es la
	// condición de resolveGovernanceChangeRoot.
	if store, declared := workspace.DeclaredArtifactStore(workspaceRoot); declared && (store == "engram" || store == "none") {
		return nil, fmt.Errorf("`--hand-off` no está disponible con el almacén de artefactos %q: el cambio no vive en ficheros que otra persona pueda recibir; las salidas disponibles son iterar con `%s` y aprobar y seguir con `%s`",
			store, gateRejectedInvocation(parsed, workspaceRoot), gateApprovedInvocation(parsed, workspaceRoot))
	}

	// No se cede una fase cuyo artefacto no existe o está vacío.
	if digest == "" {
		return nil, fmt.Errorf("no se puede ceder la fase siguiente de la compuerta %q del cambio %q: su artefacto no existe o está vacío; produce el artefacto (`axiom sdd status %s --cwd %s` indica qué falta) y repite la orden",
			parsed.Gate, parsed.Change, pathquote.ShellWord(parsed.Change), pathquote.Quote(workspaceRoot))
	}

	// Una aprobación posterior a la primera quedaría oculta tras ella: la marca de cesión sería
	// invisible. Se comprueba aquí y otra vez bajo el cerrojo del ledger (AppendGateChecked).
	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		return nil, err
	}
	if _, resolved := ledger.Approval(parsed.Gate); resolved {
		return nil, gateAlreadyResolved(parsed, workspaceRoot)
	}

	handedOffTo, err := handedOffToFor(parsed.Gate, roster, ledger)
	if err != nil {
		return nil, err
	}
	role := ""
	if isRoleApplyGate(parsed.Gate) {
		role = strings.TrimPrefix(string(parsed.Gate), "role-apply:")
	}
	plan, refusal := engine.Default().HandOff(context.Background(), branch.HandOffRequest{
		CWD:         workspaceRoot,
		Change:      parsed.Change,
		ChangeRoot:  changeRoot,
		Role:        role,
		Gate:        string(parsed.Gate),
		HandedOffTo: handedOffTo,
	})
	if refusal != nil {
		return nil, &branch.RefusalError{Refusal: *refusal}
	}
	return &handOffPrepared{handedOffTo: handedOffTo, plan: plan, parsed: parsed, workspace: workspaceRoot}, nil
}

// notApprovedCheck es la comprobación del paso 10 de DD-02, bajo el cerrojo del ledger: la puerta
// sigue sin aprobar cuando se va a anexar el registro.
func (h *handOffPrepared) notApprovedCheck(ledger kickoff.GateLedger) error {
	if _, resolved := ledger.Approval(h.parsed.Gate); resolved {
		return gateAlreadyResolved(h.parsed, h.workspace)
	}
	return nil
}

// handedOffToFor es la fase que cede aprobar gate: lo que bloquea una puerta fija o, en un
// role-apply, `verify` con el último rol y los roles que faltan en otro caso.
func handedOffToFor(gate kickoff.GateKey, roster []multirole.RoleAssignment, ledger kickoff.GateLedger) (string, error) {
	if !isRoleApplyGate(gate) {
		return kickoff.HandedOffTo(kickoff.GateState{Key: gate}, roster, nil), nil
	}
	rolePending := make(map[string]int, len(roster))
	for _, r := range roster {
		rolePending[r.Role] = 0
	}
	states, err := kickoff.EvaluateGates(kickoff.Inputs{
		Execution:   kickoff.ExecutionCheckpointed,
		Roles:       roster,
		RolePending: rolePending,
		Ledger:      ledger,
	})
	if err != nil {
		return "", err
	}
	return kickoff.HandedOffTo(kickoff.GateState{Key: gate}, roster, states), nil
}

func gateAlreadyResolved(parsed kickoff.GateRecordArgs, workspaceRoot string) error {
	return fmt.Errorf("la compuerta %q del cambio %q ya está resuelta (approved): no se puede registrar otra aprobación con cesión sobre ella; consulta su estado con `axiom sdd status %s --cwd %s`",
		parsed.Gate, parsed.Change, pathquote.ShellWord(parsed.Change), pathquote.Quote(workspaceRoot))
}

// gateApprovedInvocation y gateRejectedInvocation son las órdenes de las salidas 3 y 1 de la parada
// para la compuerta de la invocación, las que siguen valiendo cuando la salida 2 no se puede usar.
func gateApprovedInvocation(parsed kickoff.GateRecordArgs, workspaceRoot string) string {
	return fmt.Sprintf("axiom sdd gate record --cwd %s --change %s --gate %s --decision approved",
		pathquote.Quote(workspaceRoot), pathquote.ShellWord(parsed.Change), pathquote.ShellWord(string(parsed.Gate)))
}

func gateRejectedInvocation(parsed kickoff.GateRecordArgs, workspaceRoot string) string {
	return fmt.Sprintf("axiom sdd gate record --cwd %s --change %s --gate %s --decision rejected --reason <motivo>",
		pathquote.Quote(workspaceRoot), pathquote.ShellWord(parsed.Change), pathquote.ShellWord(string(parsed.Gate)))
}

// gatePublishJSON es el destino de la publicación de los documentos que `gate record` calcula y no
// ejecuta (REQ-27.12, DD-11): la misma forma que `gateQuestion.publication`. Invocation es la orden
// de `axiom sdd publish` atada a la rama con `--branch`, que el orquestador ejecuta una vez tras el
// registro. PRTarget es informativo y se omite en la integración in situ de un multirepo.
type gatePublishJSON struct {
	Remote     string `json:"remote"`
	Branch     string `json:"branch"`
	PRTarget   string `json:"prTarget,omitempty"`
	Invocation string `json:"invocation"`
}

// gatePublish es el resultado del cálculo del destino de la publicación de una aprobación: el
// destino, o el código de la negativa del motor de ramas cuando no se puede calcular. Ambos vacíos
// significan que la decisión no publica (rechazo, integración o almacén sin ficheros).
type gatePublish struct {
	target      *gatePublishJSON
	unavailable string
}

// resolveGatePublish calcula, solo con lecturas, dónde publica una aprobación (REQ-27.12): en toda
// aprobación distinta de `integration` con un almacén que guarda ficheros. En la cesión usa el plan
// que ya calculó el motor, y en las demás aprobaciones el destino de `PublicationTarget`. Nunca
// ejecuta `commit` ni `push`: `gate record` solo registra (REQ-26.5).
func resolveGatePublish(parsed kickoff.GateRecordArgs, workspaceRoot string, handOff *handOffPrepared) gatePublish {
	if parsed.Decision != kickoff.DecisionApproved || parsed.Gate == kickoff.GateIntegration {
		return gatePublish{}
	}
	if handOff != nil {
		publish := handOff.plan.Publish
		return gatePublish{target: &gatePublishJSON{Remote: publish.Remote, Branch: publish.Branch, PRTarget: publish.PRTarget, Invocation: publish.Invocation}}
	}
	if store, declared := workspace.DeclaredArtifactStore(workspaceRoot); declared && (store == "engram" || store == "none") {
		return gatePublish{}
	}
	target, refusal := engine.Default().PublicationTarget(context.Background(), branch.PublicationRequest{CWD: workspaceRoot, Change: parsed.Change})
	if refusal != nil {
		return gatePublish{unavailable: string(refusal.Code)}
	}
	return gatePublish{target: &gatePublishJSON{
		Remote:     branch.PublishRemote,
		Branch:     target.Branch,
		PRTarget:   target.PRTarget,
		Invocation: branch.PublishInvocation(workspaceRoot, parsed.Change, target.Branch),
	}}
}

// gateRecordJSON es el `--json` de `gate record` (§4.4). delivery, handedOff y handedOffTo solo
// salen con la salida 2; publish y publishUnavailable, en toda aprobación distinta de integration;
// lastRoleClosed solo en una compuerta de rol.
type gateRecordJSON struct {
	Schema             string              `json:"schema"`
	Change             string              `json:"change"`
	Gate               string              `json:"gate"`
	Decision           string              `json:"decision"`
	HandedOff          bool                `json:"handedOff,omitempty"`
	HandedOffTo        string              `json:"handedOffTo,omitempty"`
	LastRoleClosed     *bool               `json:"lastRoleClosed,omitempty"`
	Publish            *gatePublishJSON    `json:"publish,omitempty"`
	PublishUnavailable string              `json:"publishUnavailable,omitempty"`
	Delivery           *branch.HandOffPlan `json:"delivery,omitempty"`
}

// writeGateRecordJSON emite el `--json` de una compuerta registrada. lastRoleClosed es nil fuera de
// una compuerta de rol.
func writeGateRecordJSON(stdout io.Writer, parsed kickoff.GateRecordArgs, handOff *handOffPrepared, publish gatePublish, lastRoleClosed *bool) error {
	document := gateRecordJSON{
		Schema:             gateRecordSchema,
		Change:             parsed.Change,
		Gate:               string(parsed.Gate),
		Decision:           string(parsed.Decision),
		LastRoleClosed:     lastRoleClosed,
		Publish:            publish.target,
		PublishUnavailable: publish.unavailable,
	}
	if handOff != nil {
		document.HandedOff = true
		document.HandedOffTo = handOff.handedOffTo
		plan := handOff.plan
		document.Delivery = &plan
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}

// writeGateRecordHeader es la primera línea de texto de una compuerta registrada: la de siempre y,
// con la salida 2, la fase cedida.
func writeGateRecordHeader(stdout io.Writer, parsed kickoff.GateRecordArgs, handOff *handOffPrepared) error {
	if handOff == nil {
		_, err := fmt.Fprintf(stdout, "Compuerta %q registrada (%s) para %q.\n", parsed.Gate, parsed.Decision, parsed.Change)
		return err
	}
	_, err := fmt.Fprintf(stdout, "Compuerta %q registrada (%s, fase siguiente cedida: %s) para %q.\n", parsed.Gate, parsed.Decision, handOff.handedOffTo, parsed.Change)
	return err
}

// writePublishNext presenta, en las salidas 2 y 3, la orden de publicación que sigue al registro
// (REQ-27.12): Axiom solo la calcula. Sin destino calculable no escribe nada; el código de la
// negativa va en el JSON.
func writePublishNext(stdout io.Writer, publish gatePublish) error {
	if publish.target == nil {
		return nil
	}
	_, err := fmt.Fprintf(stdout, "Siguiente paso: publica los documentos del cambio con `%s` (rama %s en origin); `gate record` solo registra la decisión y no ejecuta ni commit ni push.\n",
		publish.target.Invocation, publish.target.Branch)
	return err
}

// writeHandOffDelivery presenta la entrega de la salida 2 en castellano: la orden de publicación de
// los documentos, la reanudación de quien siga y, solo si el rol tiene repositorios de código, sus
// órdenes de push numeradas con la aclaración de que Axiom no sube código ni abre PR. No exige una PR
// fusionada y no emite ninguna orden de plataforma (REQ-26.5, REQ-27.12, DD-12).
func writeHandOffDelivery(stdout io.Writer, handOff *handOffPrepared) error {
	plan := handOff.plan
	var out strings.Builder
	switch {
	case plan.Publish.InPlace:
		fmt.Fprintf(&out, "Cesión registrada. Siguiente paso: publica los documentos con `%s` (rama de integración %s en %s; se trabaja in situ en la integración, sin PR).\n",
			plan.Publish.Invocation, plan.Publish.Branch, plan.Publish.Remote)
	default:
		fmt.Fprintf(&out, "Cesión registrada. Siguiente paso: publica los documentos con `%s` (rama %s en %s; destino de su PR: %s).\n",
			plan.Publish.Invocation, plan.Publish.Branch, plan.Publish.Remote, plan.Publish.PRTarget)
	}
	fmt.Fprintf(&out, "Quien siga ejecuta `%s` y `%s`: `ensure` adopta la rama publicada; no hace falta una PR fusionada.\n", plan.Resume[0], plan.Resume[1])
	if len(plan.Deliveries) > 0 {
		out.WriteString("Código del rol (Axiom no sube código ni abre PR: ejecútalo tú):\n")
		for _, delivery := range plan.Deliveries {
			fmt.Fprintf(&out, "Repositorio %s (rama %s hacia %s):\n", delivery.Repository, delivery.Branch, delivery.PRTarget)
			for i, step := range delivery.Steps {
				fmt.Fprintf(&out, "  %d. %s\n", i+1, step)
			}
		}
	}
	out.WriteString("Axiom no ejecuta ninguna orden de git ni de plataforma en este paso: detente aquí tras publicar, sin empezar la fase siguiente en esta sesión.\n")
	_, err := io.WriteString(stdout, out.String())
	return err
}
