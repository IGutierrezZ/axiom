package cli

import (
	"encoding/json"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

func TestSDDStatusDualSerialization(t *testing.T) {
	baseStatus := sddstatus.Status{
		SchemaVersion:   sddstatus.SchemaVersion,
		ArtifactStore:   sddstatus.ArtifactStoreOpenSpec,
		ApplyState:      sddstatus.ApplyReady,
		NextRecommended: "tasks",
		Artifacts: map[string]sddstatus.ArtifactState{
			"proposal":      sddstatus.ArtifactDone,
			"specs":         sddstatus.ArtifactDone,
			"design":        sddstatus.ArtifactDone,
			"tasks":         sddstatus.ArtifactMissing,
			"applyProgress": sddstatus.ArtifactMissing,
			"verifyReport":  sddstatus.ArtifactMissing,
		},
	}

	tests := []struct {
		name       string
		schemaName string
		contract   string
		wantSchema string
	}{
		{
			name:       "SDD Status con contrato canonico Axiom v2",
			schemaName: sddstatus.AxiomSchemaName,
			contract:   sddstatus.AxiomStatusContractV2,
			wantSchema: sddstatus.AxiomSchemaName,
		},
		{
			name:       "SDD Status con contrato canonico Axiom v1",
			schemaName: sddstatus.AxiomSchemaName,
			contract:   sddstatus.AxiomStatusContractV1,
			wantSchema: sddstatus.AxiomSchemaName,
		},
		{
			name:       "SDD Status con contrato legado Gentle AI v2",
			schemaName: sddstatus.SchemaName,
			contract:   sddstatus.StatusContractV2,
			wantSchema: sddstatus.SchemaName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := baseStatus
			s.SchemaName = tt.schemaName
			proj, err := sddstatus.ProjectStatusV2(s)
			if err != nil {
				t.Fatalf("ProjectStatusV2 error: %v", err)
			}
			if proj.SchemaName != tt.wantSchema {
				t.Errorf("proj.SchemaName = %q, want %q", proj.SchemaName, tt.wantSchema)
			}

			// Serializar a JSON
			data, err := json.Marshal(proj)
			if err != nil {
				t.Fatalf("marshal ProjectStatusV2: %v", err)
			}

			// Deserializar comprobando fidelidad
			var decoded sddstatus.StatusV2Projection
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal StatusV2Projection: %v", err)
			}
			if decoded.SchemaName != tt.wantSchema {
				t.Errorf("roundtrip schemaName = %q, want %q", decoded.SchemaName, tt.wantSchema)
			}
			if decoded.ApplyState != sddstatus.ApplyReady {
				t.Errorf("roundtrip ApplyState = %q, want %q", decoded.ApplyState, sddstatus.ApplyReady)
			}
			if decoded.Artifacts["proposal"] != sddstatus.ArtifactDone {
				t.Errorf("roundtrip Artifacts[proposal] = %q, want %q", decoded.Artifacts["proposal"], sddstatus.ArtifactDone)
			}
		})
	}
}
