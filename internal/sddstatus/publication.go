package sddstatus

import (
	"context"
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Publication states (RF-PUB-11). The field is absent when there is nothing to commit or push.
const (
	// publicationPending: documentation is uncommitted or commits are unpushed and `axiom sdd
	// publish` would act on them; Invocation is the command to run.
	publicationPending = "pending"
	// publicationCodeUnpublished: unpushed commits touch code, so `axiom sdd publish` would refuse;
	// it is informational and carries the push command for the person (decision D1).
	publicationCodeUnpublished = "code_unpublished"
	// publicationUnavailable: the state could not be calculated; Unavailable names the code.
	publicationUnavailable = "unavailable"
)

// Reasons of a publication state.
const (
	publicationReasonDocsUncommitted = "docs_uncommitted"
	publicationReasonBranchAhead     = "branch_ahead"
	publicationReasonRemoteBranch    = "remote_branch_missing"
	publicationReasonCodeCommits     = "code_commits"
	publicationReasonRangeTooBig     = "range_too_big"
)

// PublicationProjection is the additive `publication` field of `axiom sdd status --json`
// (RF-PUB-11): whether `axiom sdd publish` has something to do for the selected change right now.
// It lives at the root of the status, next to `gateQuestion`, and is structurally absent (nil,
// omitempty) when there is nothing to publish or the publication does not apply, so a status
// without it is byte-identical to the one that predates it. It is computed with reads only: no
// network, no writes, and it never changes `nextRecommended`, the dependencies or the blocked reasons.
type PublicationProjection struct {
	State       string   `json:"state"`                 // pending | code_unpublished | unavailable
	Reasons     []string `json:"reasons,omitempty"`     // docs_uncommitted, branch_ahead, remote_branch_missing, code_commits, range_too_big
	Archive     bool     `json:"archive,omitempty"`     // archive moment (archive branch)
	Remote      string   `json:"remote,omitempty"`      // always "origin" when there is a target
	Branch      string   `json:"branch,omitempty"`      // the branch that is published
	PRTarget    string   `json:"prTarget,omitempty"`    // informative; empty in the in-place integration
	Invocation  string   `json:"invocation,omitempty"`  // pending only: the same as gateQuestion.publication
	PushCommand string   `json:"pushCommand,omitempty"` // code_unpublished only: for the person to run
	CodeCommits []string `json:"codeCommits,omitempty"` // code_unpublished only
	Unavailable string   `json:"unavailable,omitempty"` // unavailable only
}

// publicationEngine is the branch engine the status reads from: the test variable when a test
// sets one, engine.Default() otherwise. A nil engine (tests that disable the branch guard) means
// the status projects no publication.
func publicationEngine() *engine.Engine {
	if branchEngine != nil {
		return branchEngine()
	}
	return engine.Default()
}

// decoratePublication projects `publication` after the other additive decorations (RF-PUB-11). The
// cheap checks run first, so a change that does not meet the moment of publication pays no git read.
func decoratePublication(status *Status, location workspace.Location) {
	eng := publicationEngine()
	if eng == nil {
		return
	}
	decoratePublicationWith(status, location, eng.PublicationState)
}

// publicationReader reads the publication facts of a change; the branch engine implements it.
type publicationReader func(ctx context.Context, req branch.PublicationRequest) (branch.PublicationState, *branch.Refusal)

// decoratePublicationWith is decoratePublication over an explicit reader. A refusal of the reader,
// including a failed git read (`git_failed`), projects `unavailable` with its code: the status
// never invents a state it could not read.
func decoratePublicationWith(status *Status, location workspace.Location, read publicationReader) {
	archive, due := publicationDue(status, location)
	if !due {
		return
	}
	root := status.ActionContext.WorkspaceRoot
	change := *status.ChangeName
	state, refusal := read(context.Background(), branch.PublicationRequest{CWD: root, Change: change, Archive: archive})
	if refusal != nil {
		status.Publication = &PublicationProjection{State: publicationUnavailable, Archive: archive, Unavailable: string(refusal.Code)}
		return
	}
	status.Publication = projectPublication(state, root, change, archive)
}

// publicationDue says whether the change is at a moment where `axiom sdd publish` is expected
// (RF-PUB-08) and, in that case, whether it is the archive moment. The conditions run in order and
// the first that fails ends it:
//
//  1. a change is selected;
//  2. the artifact store keeps files (not `engram` nor `none`);
//  3. the moment: in a checkpointed change, at least one gate other than `integration` is approved;
//     in a continuous change with `docsPublication: auto`, `specs`, `design` or `tasks` are done
//     before `apply` starts (no task completed, no verify report); in an archived continuous
//     change with `docsPublication: auto`, the archive. An archived checkpointed change waits for
//     the person, as the doctrine already says.
func publicationDue(status *Status, location workspace.Location) (archive, due bool) {
	if status.ChangeName == nil {
		return false, false
	}
	switch effectiveArtifactStore(status, location) {
	case ArtifactStoreEngram, ArtifactStoreNone:
		return false, false
	}
	if status.Archived != nil {
		root := location.WorkspaceRoot
		if root == "" {
			root = status.ActionContext.WorkspaceRoot
		}
		archivedRoot, found := kickoff.ArchivedChangeRoot(root, *status.ChangeName)
		if !found {
			return false, false
		}
		sealed, err := kickoff.Load(archivedRoot)
		if err != nil || sealed == nil {
			return false, false
		}
		return true, continuousAutoPublication(sealed.Config)
	}
	governance := status.Governance
	if governance == nil {
		return false, false
	}
	if governanceGated(governance) {
		// Iterar no publica nada (RF-GAT-15): con la fase de la primera compuerta abierta en
		// curso, la publicación espera a la aprobación siguiente, que vuelve a cumplir (a).
		if open, found := firstOpenGate(governance); found && phaseInProgress(status, open.Key) {
			return false, false
		}
		for _, gate := range governance.Gates {
			if gate.Status == "approved" && gate.Key != kickoff.GateIntegration {
				return false, true
			}
		}
		return false, false
	}
	if !continuousAutoPublication(governance.Kickoff.Config) {
		return false, false
	}
	planned := false
	for _, key := range []string{"specs", "design", "tasks"} {
		// Un artefacto a medio escribir es una fase en curso, no un momento de RF-PUB-08.
		if status.Artifacts[key] == ArtifactPartial {
			return false, false
		}
		planned = planned || status.Artifacts[key] == ArtifactDone
	}
	return false, planned && status.TaskProgress.Completed == 0 && status.Artifacts["verifyReport"] != ArtifactDone
}

// phaseInProgress reports whether the phase that the open gate judges has already started: its
// artifact exists (design, tasks) or apply has progress (role-apply). spec never has an approved
// gate before it, and integration only closes the change, so neither counts.
func phaseInProgress(status *Status, key kickoff.GateKey) bool {
	started := func(artifact string) bool {
		state := status.Artifacts[artifact]
		return state == ArtifactPartial || state == ArtifactDone
	}
	switch key {
	case kickoff.GateDesign:
		return started("design")
	case kickoff.GateTasks:
		return started("tasks")
	}
	if _, roleApply := kickoff.RoleFromGate(key); roleApply {
		return status.TaskProgress.Completed > 0 || started("applyProgress")
	}
	return false
}

// continuousAutoPublication reports the sealed authorization to publish without asking (RF-KOF-10).
func continuousAutoPublication(config kickoff.FlowConfig) bool {
	return config.ExecutionStyle == kickoff.ExecutionContinuous && config.DocsPublication == kickoff.DocsPublicationAuto
}

// projectPublication translates the facts into the state of the precedence table (design D1,
// option B). The first rule that holds wins; nil means there is nothing to commit or push.
func projectPublication(state branch.PublicationState, workspaceRoot, change string, archive bool) *PublicationProjection {
	if state.RemoteMissing {
		return nil
	}
	target := gatePublicationFrom(state.Target, workspaceRoot, change)
	projection := PublicationProjection{Archive: archive, Remote: target.Remote, Branch: target.Branch, PRTarget: target.PRTarget}
	if state.RangeUnknown {
		projection.State, projection.Unavailable = publicationUnavailable, string(branch.PublicationCodeRangeUnknown)
		return &projection
	}
	remoteBranchMissing := func(reasons []string) []string {
		if !state.RemoteBranch {
			return append(reasons, publicationReasonRemoteBranch)
		}
		return reasons
	}
	switch {
	case len(state.DirtyDocs) > 0:
		projection.State = publicationPending
		projection.Reasons = []string{publicationReasonDocsUncommitted}
		if state.Unpushed > 0 {
			projection.Reasons = append(projection.Reasons, publicationReasonBranchAhead)
		}
		projection.Reasons = remoteBranchMissing(projection.Reasons)
		if len(state.CodeCommits) > 0 {
			projection.Reasons = append(projection.Reasons, publicationReasonCodeCommits)
		}
	case state.RangeTooBig:
		projection.State = publicationCodeUnpublished
		projection.Reasons = []string{publicationReasonRangeTooBig}
		projection.PushCommand = publicationPushCommand(state.Target.GitRoot, state.Target.Branch)
	case len(state.CodeCommits) > 0:
		projection.State = publicationCodeUnpublished
		projection.Reasons = remoteBranchMissing([]string{publicationReasonCodeCommits})
		projection.PushCommand = publicationPushCommand(state.Target.GitRoot, state.Target.Branch)
		projection.CodeCommits = state.CodeCommits
	case state.Unpushed > 0:
		projection.State = publicationPending
		projection.Reasons = remoteBranchMissing([]string{publicationReasonBranchAhead})
	default:
		return nil
	}
	if projection.State == publicationPending {
		projection.Invocation = target.Invocation
		if archive {
			projection.Invocation = branch.PublishArchiveInvocation(workspaceRoot, change, state.Target.Branch)
		}
	}
	return &projection
}

// publicationPushCommand is the command that pushes the branch by hand, for the person: the same
// `git -C <root> push origin <branch>:<branch>` that `axiom sdd publish` names when it refuses
// unpushed code. Axiom never runs it.
func publicationPushCommand(gitRoot, branchName string) string {
	return fmt.Sprintf("git -C %s push origin %s:%s", pathquote.Quote(gitRoot), pathquote.ShellWord(branchName), pathquote.ShellWord(branchName))
}
