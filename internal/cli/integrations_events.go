package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// integrationWarnings recibe los avisos del despacho de eventos de integración. Es propio, como
// actorIdentityWarnings: los verbos de origen no cambian su stdout y las pruebas lo recogen.
var integrationWarnings io.Writer = os.Stderr

func integrationWarn(format string, args ...any) {
	text := strings.Join(strings.Fields(fmt.Sprintf(format, args...)), " ")
	_, _ = fmt.Fprintln(integrationWarnings, "Aviso: "+text)
}

// newIntegrationDispatcher compone el despachador con el bloque, el registro cerrado, el almacén de
// secretos y la guarda A de ramas (contexto ordinary o archive según la carpeta).
func newIntegrationDispatcher(loc workspace.Location) (*integrations.Dispatcher, error) {
	registry, err := integrations.DefaultRegistry()
	if err != nil {
		return nil, err
	}
	return &integrations.Dispatcher{
		Registry:      registry,
		Config:        loc.Declarations.Integrations,
		Secrets:       secrets.Default(),
		WorkspaceRoot: loc.WorkspaceRoot,
		Guard: func(folder integrations.Folder) error {
			writerContext := branch.ContextOrdinary
			if folder.Archived() {
				writerContext = branch.ContextArchive
			}
			return integrationWriterGuard(folder.Root, folder.Change(), "integrations dispatch", writerContext)
		},
	}, nil
}

// dispatchIntegrationEvent entrega un evento del ciclo de vida a los plugins configurados (D-11).
// Nunca devuelve error ni cambia lo que imprime el verbo de origen: todo problema es un aviso.
//
//  1. Un bloque `integrations:` inválido avisa y no despacha.
//  2. Sin plugin activo que escuche el evento vuelve sin leer ni escribir disco, sin resolver la
//     identidad git y sin llamar a build (por eso build es diferido).
//  3. La guarda A y el despacho los hace el Dispatcher; lo que no se entrega queda en la cola con
//     un aviso que nombra su ruta y `axiom integrations sync`.
func dispatchIntegrationEvent(loc workspace.Location, changeRoot string, eventType manifest.EventType, build func() integrations.Event) {
	change := filepath.Base(changeRoot)
	if loc.Declarations.IntegrationsErr != nil {
		integrationWarn("el bloque `integrations:` de axiom.yaml no es válido (%v); no se despacha el evento %s del cambio %s. Corrígelo con `axiom workspace validate` y emite el evento a mano con `axiom integrations emit --cwd %s --change %s --event %s`.",
			loc.Declarations.IntegrationsErr, eventType, change, loc.WorkspaceRoot, change, eventType)
		return
	}
	if !integrations.HasListeners(loc.Declarations.Integrations, eventType) {
		return
	}
	dispatcher, err := newIntegrationDispatcher(loc)
	if err != nil {
		integrationWarn("no se puede despachar el evento %s a las integraciones: %v", eventType, err)
		return
	}
	ev := build()
	report := dispatcher.Dispatch(context.Background(), changeRoot, ev, integrations.DispatchOptions{
		Durable:        true,
		PersistOutcome: integrations.PolicyFor(eventType).PersistOutcome,
	})
	reportDispatch(loc, report, eventType)
}

// reportDispatch convierte el resultado de un despacho en avisos.
func reportDispatch(loc workspace.Location, report integrations.Report, eventType manifest.EventType) {
	if report.Err != nil {
		integrationWarn("no se ha podido despachar el evento %s del cambio %s a las integraciones: %v. Reintenta con `axiom integrations emit --cwd %s --change %s --event %s`.",
			eventType, report.Change, report.Err, loc.WorkspaceRoot, report.Change, eventType)
	}
	for _, warning := range report.Warnings {
		integrationWarn("%s", warning)
	}
	for _, op := range report.Undelivered() {
		detail := op.LastError
		if op.Reason != "" {
			detail = op.Reason
		}
		integrationWarn("la entrega %s al plugin %s no se completó (%s, intentos: %d%s); queda en la cola %s. Reintenta con `axiom integrations sync --cwd %s --change %s`.",
			op.Key, op.Plugin, op.State, op.Attempts, errorSuffix(detail), report.QueuePath, loc.WorkspaceRoot, report.Change)
	}
}

func errorSuffix(detail string) string {
	if detail == "" {
		return ""
	}
	return ": " + detail
}

// approvalOrdinal cuenta, desde 1, la aprobación de record entre las aprobaciones de su compuerta en
// el ledger ya escrito (la huella de respaldo de D-09 cuando el digest está vacío).
func approvalOrdinal(changeRoot string, record kickoff.GateRecord) int {
	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		return 1
	}
	ordinal := 0
	for _, r := range ledger.Records {
		if r.Gate == record.Gate && r.Decision == kickoff.DecisionApproved {
			ordinal++
		}
	}
	if ordinal == 0 {
		return 1
	}
	return ordinal
}

// gateEventFor traduce una compuerta aprobada al evento del catálogo; tasks e integration no emiten.
func gateEventFor(gate kickoff.GateKey) (manifest.EventType, string, bool) {
	switch gate {
	case kickoff.GateSpec:
		return manifest.EventGateSpecApproved, "", true
	case kickoff.GateDesign:
		return manifest.EventGateDesignApproved, "", true
	}
	if role, ok := kickoff.RoleFromGate(gate); ok {
		return manifest.EventRoleApplyApproved, role, true
	}
	return "", "", false
}

// gateIntegrationEvent construye el evento de una aprobación ya registrada en gates.yaml (D-09).
func gateIntegrationEvent(changeRoot string, eventType manifest.EventType, role string, record kickoff.GateRecord) integrations.Event {
	change := filepath.Base(changeRoot)
	fingerprint := integrations.Fingerprint(record.ArtifactDigest, approvalOrdinal(changeRoot, record))
	key := integrations.GateApprovedKey(eventType, change, fingerprint)
	if eventType == manifest.EventRoleApplyApproved {
		key = integrations.RoleApplyApprovedKey(change, role, fingerprint)
	}
	return integrations.Event{
		Type:           eventType,
		Change:         change,
		Role:           role,
		Gate:           string(record.Gate),
		Actor:          record.Actor,
		OccurredAt:     record.RecordedAt,
		ArtifactDigest: record.ArtifactDigest,
		Fingerprint:    fingerprint,
		IdempotencyKey: key,
	}
}

// emitGateIntegrationEvent despacha gate.spec.approved, gate.design.approved o role.apply.approved
// tras registrar una aprobación. Las decisiones rejected, tasks e integration no emiten.
func emitGateIntegrationEvent(loc workspace.Location, changeRoot string, record kickoff.GateRecord) {
	if record.Decision != kickoff.DecisionApproved {
		return
	}
	eventType, role, ok := gateEventFor(record.Gate)
	if !ok {
		return
	}
	dispatchIntegrationEvent(loc, changeRoot, eventType, func() integrations.Event {
		return gateIntegrationEvent(changeRoot, eventType, role, record)
	})
}

// emitRoleStartEvent despacha role.start con la clave role.start/<cambio>/<rol>.
func emitRoleStartEvent(loc workspace.Location, changeRoot, role string) {
	dispatchIntegrationEvent(loc, changeRoot, manifest.EventRoleStart, func() integrations.Event {
		actor, _ := defaultActor(loc.WorkspaceRoot, gateActorTarget)
		change := filepath.Base(changeRoot)
		return integrations.Event{
			Type:           manifest.EventRoleStart,
			Change:         change,
			Role:           role,
			Actor:          actor,
			OccurredAt:     time.Now().UTC(),
			IdempotencyKey: integrations.RoleStartKey(change, role),
		}
	})
}

// emitArchiveEvent despacha archive sobre la carpeta archivada, con la clave
// archive/<cambio>/<carpeta archivada>.
func emitArchiveEvent(loc workspace.Location, archivedRoot string) {
	folder := integrations.Folder{WorkspaceRoot: loc.WorkspaceRoot, Root: archivedRoot}
	dispatchIntegrationEvent(loc, archivedRoot, manifest.EventArchive, func() integrations.Event {
		actor, _ := defaultActor(loc.WorkspaceRoot, gateActorTarget)
		return integrations.Event{
			Type:           manifest.EventArchive,
			Change:         folder.Change(),
			Actor:          actor,
			OccurredAt:     time.Now().UTC(),
			IdempotencyKey: integrations.ArchiveKey(folder.Change(), filepath.Base(archivedRoot)),
		}
	})
}

// integrationWriterGuard es la guarda A que protege la escritura de la cola y de los vinculos;
// es una variable para que las pruebas simulen su rechazo.
var integrationWriterGuard = guardChangeWriter
