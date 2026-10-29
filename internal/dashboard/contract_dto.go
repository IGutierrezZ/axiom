package dashboard

import (
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/handoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// NewBarrierDTO convierte el informe de barrera del dominio al contrato JSON del dashboard.
// Devuelve nil si el informe es nil. Las listas vacías se serializan como [] y no como null.
func NewBarrierDTO(r *multirole.BarrierReport) *BarrierDTO {
	if r == nil {
		return nil
	}
	dto := &BarrierDTO{
		Satisfied:     r.Satisfied,
		Change:        r.Change,
		Blockers:      nonNilStrings(r.Blockers),
		Warnings:      nonNilStrings(r.Warnings),
		Roles:         make([]RoleStatusDTO, 0, len(r.Roles)),
		DeferredTasks: make([]DeferredTaskDTO, 0, len(r.DeferredTasks)),
	}
	for _, rs := range r.Roles {
		dto.Roles = append(dto.Roles, RoleStatusDTO{
			Role:         rs.Assignment.Role,
			Name:         rs.Assignment.Name,
			GatePolicy:   string(rs.Assignment.GatePolicy),
			Repositories: nonNilStrings(rs.Assignment.Repositories),
			Deliverables: nonNilStrings(rs.Assignment.Deliverables),
			TasksFound:   rs.TasksFound,
			Tasks: TaskProgressDTO{
				Total:     rs.Tasks.Total,
				Completed: rs.Tasks.Completed,
				Pending:   rs.Tasks.Pending,
				Percent:   rs.Tasks.Percent,
			},
			ApplyDone:  rs.ApplyDone,
			VerifyDone: rs.VerifyDone,
			Verdict:    rs.Verdict,
			Compliant:  rs.Compliant,
		})
	}
	for _, dt := range r.DeferredTasks {
		dto.DeferredTasks = append(dto.DeferredTasks, DeferredTaskDTO{
			Change:       dt.Change,
			Role:         dt.Role,
			TaskText:     dt.TaskText,
			SpecRef:      dt.SpecRef,
			ChangeDirRef: dt.ChangeDirRef,
		})
	}
	return dto
}

// NewHandoffDTO convierte un relevo del dominio al contrato JSON del dashboard.
// Devuelve nil si el relevo es nil.
func NewHandoffDTO(h *handoff.Handoff) *HandoffDTO {
	if h == nil {
		return nil
	}
	ts := ""
	if !h.Metadata.Timestamp.IsZero() {
		ts = h.Metadata.Timestamp.Format(time.RFC3339)
	}
	return &HandoffDTO{
		Metadata: HandoffMetadataDTO{
			Change:    h.Metadata.Change,
			FromPhase: string(h.Metadata.FromPhase),
			ToPhase:   string(h.Metadata.ToPhase),
			FromRole:  h.Metadata.FromRole,
			ToRole:    h.Metadata.ToRole,
			Timestamp: ts,
			Status:    string(h.Metadata.Status),
		},
		Sections: HandoffSectionsDTO{
			ExecutiveSummary:   h.Sections.ExecutiveSummary,
			Artifacts:          h.Sections.Artifacts,
			Decisions:          h.Sections.Decisions,
			RisksAndBlockers:   h.Sections.RisksAndBlockers,
			DirectInstructions: h.Sections.DirectInstructions,
		},
	}
}

// nonNilStrings garantiza que una lista vacía se serialice como [] en JSON.
func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
