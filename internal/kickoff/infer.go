package kickoff

import (
	"errors"
	"os"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// InferKickoff computes the conservative retro-seal default for a
// pre-existing change that has artifacts but no sealed kickoff.yaml
// (REQ-21.4, design.md S8.1 — the corrected rule that supersedes the
// original spec.md:102 wording; see O-1). It is pure with respect to
// writes: it never seals anything. The caller passes the returned value to
// Seal once it has filled in Change.
//
// Three cases, in this order:
//
//  1. design.md exists and multirole.DetectRoles resolves roles without
//     error: freeze EXACTLY those roles, with execution_style: continuous,
//     execution_style_source: inferred, sealed_by: inferred. It infers no
//     handoff policy: handoff_policy is deprecated (REQ-26.9) and is written
//     as none only as a compatibility value for older binaries. This NEVER introduces the "fullstack" identity — doing so
//     would silently change which tasks.<role>.md file the existing
//     multirole barrier consults for that change.
//  2. design.md does not exist: nothing reads roles yet, so there is no
//     prior behaviour to preserve. Seal the single "fullstack" role with
//     tasks.md / verify-report.md.
//  3. DetectRoles itself errors (e.g. a role declared in design.md is
//     absent from axiom.yaml): infer nothing, propagate the error
//     unchanged, and return a non-blocking note. Sealing "fullstack" here
//     would mask a real configuration error behind an inference.
//
// The retro-seal is deliberately exempt from the continuous-roster rule that
// explicit sealing enforces (P2, REQ-21.4): it keeps sealing continuous with
// the roles a change already declares, marked sealed_by: inferred, and
// ValidateForSeal accepts that marker.
//
// designPath is the path to the change's design.md; wsConfig is the
// workspace's axiom.yaml configuration (may be nil).
//
// The only I/O InferKickoff performs is the read already encapsulated
// inside multirole.DetectRoles: it distinguishes case (b) from case (c) by
// inspecting DetectRoles' own error with errors.Is(err, os.ErrNotExist)
// instead of probing the filesystem a second time.
func InferKickoff(designPath string, wsConfig *workspace.WorkspaceConfig) (Kickoff, string, error) {
	roles, detectErr := multirole.DetectRoles(designPath, wsConfig)
	if detectErr != nil {
		if errors.Is(detectErr, os.ErrNotExist) {
			// Case (b): no design.md yet. Nothing reads roles today, so
			// there is no prior behaviour to preserve.
			return inferredKickoff([]multirole.RoleAssignment{
				{Role: "fullstack", GatePolicy: multirole.PolicyBlocking},
			}), "", nil
		}
		// Case (c): DetectRoles rejects for a real reason (e.g. a role
		// declared in design.md is absent from axiom.yaml). Infer
		// nothing; propagate the existing error intact. Sealing
		// "fullstack" here would mask a real configuration error behind
		// an inference.
		const note = "no se pudo inferir el kickoff: la deteccion de roles ya vigente devolvio un error; corrige la configuracion de axiom.yaml antes de sellar"
		return Kickoff{}, note, detectErr
	}
	// Case (a): design.md exists and DetectRoles resolved roles without
	// error. Freeze exactly those roles: never fullstack.
	return inferredKickoff(roles), "", nil
}

// inferredKickoff assembles the retro-sealed document body shared by cases
// (a) and (b): only the role list differs between callers. Role file names
// come from defaultRoleArtifactFiles (types.go) — the same convention an
// explicit `axiom sdd kickoff seal` applies via SealArgs.ToKickoff (args.go)
// — so the retro-seal never invents a naming convention the barrier does
// not already use, and never redirects it to a plain tasks.md it did not
// ask for (REQ-21.4, third scenario).
func inferredKickoff(roles []multirole.RoleAssignment) Kickoff {
	kickoffRoles := make([]KickoffRole, 0, len(roles))
	for _, r := range roles {
		tasksFile, verifyFile := defaultRoleArtifactFiles(r.Role)
		kickoffRoles = append(kickoffRoles, KickoffRole{
			Role:       r.Role,
			GatePolicy: r.GatePolicy,
			TasksFile:  tasksFile,
			VerifyFile: verifyFile,
		})
	}
	return Kickoff{
		Schema:   KickoffSchemaV1,
		SealedBy: SealedByInferred,
		Config: FlowConfig{
			FlowMode:             FlowSDD,
			ExecutionStyle:       ExecutionContinuous,
			ExecutionStyleSource: "inferred",
			HandoffPolicy:        HandoffNone, // compatibility value (REQ-26.9)
			Roles:                kickoffRoles,
		},
		Lifecycle: Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	}
}
