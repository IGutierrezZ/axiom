package cli

import (
	"context"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// GateStop es la parada de puerta de un cambio tal como la proyecta `axiom sdd status` (C-06-f, D7),
// más lo necesario para reconstruir «aprobada sin publicar» tras una recarga (D9): siempre se
// devuelven la publicación del status, el último registro del ledger y el estilo de ejecución.
type GateStop struct {
	Change string
	// WorkspaceRoot es la raíz del status (`actionContext.workspaceRoot`), la que usa `Publish`.
	WorkspaceRoot string
	// RecordRoot es la raíz en la que se registra la decisión: la del status, o el worktree de
	// archivado registrado cuando la puerta es `integration`.
	RecordRoot      string
	ExecutionStyle  string
	ArtifactStore   string
	NextRecommended string
	BlockedReasons  []string
	// Question es la pregunta tipada de la puerta, con las opciones en su orden; nil sin parada.
	Question *sddstatus.SDDGovernanceGateResult
	// Digest es el de Question.Gate calculado sobre la carpeta del cambio de RecordRoot.
	Digest       string
	Publication  *sddstatus.PublicationProjection
	LastDecision *kickoff.GateRecord
}

// GatesGateway es la pasarela que el dashboard usa para consultar, registrar y publicar la decisión
// de una puerta. No guarda estado y no añade lógica: cada operación delega en el mismo núcleo que
// la CLI.
type GatesGateway struct{}

// NewGatesGateway devuelve la pasarela.
func NewGatesGateway() *GatesGateway { return &GatesGateway{} }

// Stop resuelve el status del cambio desde cwd y devuelve su parada. Solo hay parada cuando el
// status recomienda `await-gate`; el resto de campos se devuelven siempre.
func (*GatesGateway) Stop(ctx context.Context, cwd, change string) (GateStop, error) {
	if err := ctx.Err(); err != nil {
		return GateStop{}, err
	}
	status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: cwd, ChangeName: change})
	if err != nil {
		return GateStop{}, err
	}
	name := change
	if status.ChangeName != nil && *status.ChangeName != "" {
		name = *status.ChangeName
	}
	root := status.ActionContext.WorkspaceRoot
	if root == "" {
		root = cwd
	}
	stop := GateStop{
		Change: name, WorkspaceRoot: root, RecordRoot: root,
		ArtifactStore:   string(status.ArtifactStore),
		NextRecommended: status.NextRecommended,
		BlockedReasons:  status.BlockedReasons,
		Publication:     status.Publication,
	}
	if status.Governance != nil {
		stop.ExecutionStyle = string(status.Governance.Kickoff.Config.ExecutionStyle)
	}
	if changeRoot, ok := statusChangeRoot(status, root, name); ok {
		if ledger, err := kickoff.LoadGates(changeRoot); err == nil && len(ledger.Records) > 0 {
			last := ledger.Records[len(ledger.Records)-1]
			stop.LastDecision = &last
		}
	}
	if status.NextRecommended != "await-gate" || status.GateQuestion == nil {
		return stop, nil
	}
	stop.Question = status.GateQuestion
	gate := kickoff.GateKey(status.GateQuestion.Gate)
	if gate == kickoff.GateIntegration {
		stop.RecordRoot = archiveRecordRoot(ctx, root, name)
	}
	if recordChangeRoot, ok := statusChangeRoot(status, stop.RecordRoot, name); ok {
		sealed, err := kickoff.Load(recordChangeRoot)
		if err != nil {
			return GateStop{}, err
		}
		if stop.Digest, err = gateArtifactDigest(recordChangeRoot, gate, sealed); err != nil {
			return GateStop{}, err
		}
	}
	return stop, nil
}

// Record registra la decisión con el mismo núcleo que `axiom sdd gate record`. El actor nunca viaja:
// RecordGate lo resuelve con la identidad git de la raíz, como la CLI sin --actor.
func (*GatesGateway) Record(ctx context.Context, in GateRecordInput) (GateRecordOutcome, error) {
	in.Actor = ""
	return RecordGate(ctx, in)
}

// Publish publica los documentos del cambio con la misma guarda que `axiom sdd publish`.
func (*GatesGateway) Publish(ctx context.Context, req ChangePublishRequest) (ChangePublishResult, error) {
	return PublishChange(ctx, req)
}

// statusChangeRoot es la carpeta del cambio bajo root: la que RecordGate resolvería desde esa raíz.
func statusChangeRoot(status sddstatus.Status, root, change string) (string, bool) {
	if status.ChangeRoot != nil && *status.ChangeRoot != "" && sameRoot(status.ActionContext.WorkspaceRoot, root) {
		return *status.ChangeRoot, true
	}
	dir := filepath.Join(root, "openspec", "changes", change)
	return dir, isDirectory(dir)
}

func sameRoot(a, b string) bool { return pathidentity.SameDirectory(a, b) }

// archiveRecordRoot es el worktree de archivado registrado del cambio o, si no hay, root: entonces la
// guarda A del núcleo, en contexto `archive`, niega con la continuación `axiom branch ensure --archive`.
func archiveRecordRoot(ctx context.Context, root, change string) string {
	loc, err := workspace.Locate(root)
	if err != nil {
		return root
	}
	registered, _ := engine.Default().RegisteredChangesByName(ctx, loc)
	if found, ok := registered[change]; ok && found.Kind == branch.KindArchive && found.WorktreeRoot != "" {
		return found.WorktreeRoot
	}
	return root
}
