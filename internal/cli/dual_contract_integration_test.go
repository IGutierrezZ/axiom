package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/reviewtransaction"
	"github.com/gentleman-programming/gentle-ai/v3/internal/sddstatus"
)

func TestDualContract_ReviewCapabilitiesNegotiation(t *testing.T) {
	tests := []struct {
		name         string
		contract     string
		wantContract string
		wantMajor    int
		wantMinor    int
		wantErr      bool
	}{
		{
			name:         "Negociacion canonica directa con Axiom v2",
			contract:     reviewtransaction.AxiomReviewIntegrationV2Contract,
			wantContract: reviewtransaction.AxiomReviewIntegrationV2Contract,
			wantMajor:    2,
			wantMinor:    6,
			wantErr:      false,
		},
		{
			name:         "Negociacion retrocompatible con Gentle AI v2",
			contract:     ReviewIntegrationContractV2,
			wantContract: ReviewIntegrationContractV2,
			wantMajor:    2,
			wantMinor:    6,
			wantErr:      false,
		},
		{
			name:         "Negociacion retrocompatible con Gentle AI v1",
			contract:     ReviewIntegrationContractV1,
			wantContract: ReviewIntegrationContractV1,
			wantMajor:    1,
			wantMinor:    5,
			wantErr:      false,
		},
		{
			name:         "Contrato desconocido rechazado tipificadamente",
			contract:     "unsupported.review-integration/v99",
			wantContract: "",
			wantErr:      true,
		},
		{
			name:         "Contrato vacio rechazado",
			contract:     "",
			wantContract: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			args := []string{"--contract", tt.contract}
			err := RunReviewCapabilities(args, &stdout)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunReviewCapabilities(--contract %q) error = %v, wantErr %v", tt.contract, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			var result ReviewCapabilitiesResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("unmarshal ReviewCapabilitiesResult: %v", err)
			}
			if result.Contract != tt.wantContract {
				t.Errorf("contract = %q, want %q", result.Contract, tt.wantContract)
			}
			if result.Protocol.Major != tt.wantMajor || result.Protocol.Minor != tt.wantMinor {
				t.Errorf("protocol = %d.%d, want %d.%d", result.Protocol.Major, result.Protocol.Minor, tt.wantMajor, tt.wantMinor)
			}
			if err := result.Validate(); err != nil {
				t.Errorf("result.Validate() failed: %v", err)
			}
		})
	}
}

func TestDualContract_ReviewOperationNegotiationAndValidation(t *testing.T) {
	testValidateResult := ReviewValidateResult{
		Schema:  ReviewValidateSchema,
		Result:  reviewtransaction.GateAllow,
		Allowed: true,
		Action:  "continue",
		Reason:  "all checks passed",
		Context: reviewtransaction.GateContext{
			Gate: reviewtransaction.GatePostApply,
		},
	}

	tests := []struct {
		name         string
		contract     string
		wantSchema   string
		wantContract string
	}{
		{
			name:         "Operacion negociada Axiom v2",
			contract:     AxiomReviewIntegrationContractV2,
			wantSchema:   reviewtransaction.AxiomReviewOperationV2Contract,
			wantContract: AxiomReviewIntegrationContractV2,
		},
		{
			name:         "Operacion negociada Gentle AI v2",
			contract:     ReviewIntegrationContractV2,
			wantSchema:   ReviewIntegrationOperationSchemaV2,
			wantContract: ReviewIntegrationContractV2,
		},
		{
			name:         "Operacion negociada Gentle AI v1",
			contract:     ReviewIntegrationContractV1,
			wantSchema:   ReviewIntegrationOperationSchema,
			wantContract: ReviewIntegrationContractV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			err := encodeReviewIntegrationOperation(&stdout, true, "review.validate", testValidateResult, testValidateResult, tt.contract)
			if err != nil {
				t.Fatalf("encodeReviewIntegrationOperation error: %v", err)
			}

			var envelope ReviewIntegrationOperationResult
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatalf("unmarshal ReviewIntegrationOperationResult: %v", err)
			}

			if envelope.Schema != tt.wantSchema {
				t.Errorf("schema = %q, want %q", envelope.Schema, tt.wantSchema)
			}
			if envelope.Contract != tt.wantContract {
				t.Errorf("contract = %q, want %q", envelope.Contract, tt.wantContract)
			}
			if err := envelope.Validate(); err != nil {
				t.Errorf("envelope.Validate() error: %v", err)
			}

			// Validar deserializacion estricta del resultado
			var decoded ReviewValidateResult
			if err := json.Unmarshal(envelope.Result, &decoded); err != nil {
				t.Fatalf("unmarshal envelope payload: %v", err)
			}
			if !decoded.Allowed || decoded.Action != "continue" {
				t.Errorf("decoded.Allowed = %v, Action = %q", decoded.Allowed, decoded.Action)
			}
		})
	}
}

func TestDualContract_ReviewFailureNegotiation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantSchema   string
		wantContract string
	}{
		{
			name:         "Fallo con contrato canonico Axiom v2",
			args:         []string{"validate", "--contract=" + AxiomReviewIntegrationContractV2},
			wantSchema:   reviewtransaction.AxiomReviewFailureV2Contract,
			wantContract: AxiomReviewIntegrationContractV2,
		},
		{
			name:         "Fallo con contrato legado Gentle AI v2",
			args:         []string{"validate", "--contract=" + ReviewIntegrationContractV2},
			wantSchema:   ReviewIntegrationFailureSchemaV2,
			wantContract: ReviewIntegrationContractV2,
		},
		{
			name:         "Fallo con contrato por defecto v1",
			args:         []string{"validate"},
			wantSchema:   ReviewIntegrationFailureSchema,
			wantContract: ReviewIntegrationContractV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := newReviewIntegrationFailure("review.validate", tt.args, errors.New("simulated error"))
			if failure.Schema != tt.wantSchema {
				t.Errorf("failure.Schema = %q, want %q", failure.Schema, tt.wantSchema)
			}
			if failure.Contract != tt.wantContract {
				t.Errorf("failure.Contract = %q, want %q", failure.Contract, tt.wantContract)
			}
		})
	}
}

func TestDualContract_ReviewPreflightFailureRejection(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantCode  string
		wantValid bool
	}{
		{
			name:      "Contrato canonico Axiom valido",
			args:      []string{"validate", "--contract", AxiomReviewIntegrationContractV2},
			wantValid: true,
		},
		{
			name:      "Contrato legado Gentle AI valido",
			args:      []string{"validate", "--contract", ReviewIntegrationContractV2},
			wantValid: true,
		},
		{
			name:      "Contrato desconocido rechazado",
			args:      []string{"validate", "--contract", "invalid.review/v99"},
			wantCode:  "unsupported_contract",
			wantValid: false,
		},
		{
			name:      "Contrato vacio rechazado",
			args:      []string{"validate", "--contract="},
			wantCode:  "empty_contract",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, handled, failure := reviewIntegrationFailureRoute(tt.args)
			if failure != nil && tt.wantValid {
				t.Fatalf("unexpected failure: %#v", failure)
			}
			if !tt.wantValid {
				if !handled {
					t.Fatal("expected handled failure route, got handled=false")
				}
				if failure == nil {
					t.Fatal("expected failure envelope, got nil")
				}
				if failure.Code != tt.wantCode {
					t.Errorf("failure.Code = %q, want %q", failure.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestDualContract_SchemaValidationWithCompiledSchemas(t *testing.T) {
	opSchema := compileWholePublishedReviewSchema(t, "v2", "operation.schema.json")
	failureSchema := compileWholePublishedReviewSchema(t, "v2", "failure.schema.json")
	capSchema := compileWholePublishedReviewSchema(t, "v2", "capabilities-v2.6.schema.json")

	validResultBytes, err := json.Marshal(ReviewValidateResult{
		Schema:  ReviewValidateSchema,
		Result:  reviewtransaction.GateAllow,
		Allowed: true,
		Action:  "continue",
		Reason:  "all checks passed",
		Context: reviewtransaction.GateContext{
			Gate: reviewtransaction.GatePostApply,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Validar operacion con Axiom contra schema v2 compilado
	axiomOp := ReviewIntegrationOperationResult{
		Schema:    reviewtransaction.AxiomReviewOperationV2Contract,
		Contract:  AxiomReviewIntegrationContractV2,
		Operation: "review.validate",
		Result:    validResultBytes,
	}
	axiomOpBytes, err := json.Marshal(axiomOp)
	if err != nil {
		t.Fatal(err)
	}
	validatePublishedReviewSchema(t, opSchema, axiomOpBytes)

	// 2. Validar operacion con Gentle AI contra schema v2 compilado
	legacyOp := ReviewIntegrationOperationResult{
		Schema:    ReviewIntegrationOperationSchemaV2,
		Contract:  ReviewIntegrationContractV2,
		Operation: "review.validate",
		Result:    validResultBytes,
	}
	legacyOpBytes, err := json.Marshal(legacyOp)
	if err != nil {
		t.Fatal(err)
	}
	validatePublishedReviewSchema(t, opSchema, legacyOpBytes)

	// 3. Validar fallo con Axiom contra schema v2 compilado
	axiomFailure := ReviewIntegrationFailure{
		Schema:                 reviewtransaction.AxiomReviewFailureV2Contract,
		Contract:               AxiomReviewIntegrationContractV2,
		Operation:              "review.validate",
		Phase:                  "preflight",
		Code:                   "unsupported_contract",
		Message:                "The requested contract is not supported.",
		MutationOutcome:        ReviewMutationNotStarted,
		AuthorityApplicability: "not_evaluated",
		RetrySafe:              true,
		Replayability:          reviewtransaction.ReplayabilityNotReplayable,
		RequiredInputs:         []string{},
		NextAction:             "correct_request",
	}
	axiomFailureBytes, err := json.Marshal(axiomFailure)
	if err != nil {
		t.Fatal(err)
	}
	validatePublishedReviewSchema(t, failureSchema, axiomFailureBytes)

	// 4. Validar capacidades v2.6 negociadas con Axiom
	surface := reviewCapabilitiesStaticSurface(AxiomReviewIntegrationContractV2)
	surface.Package = ReviewCapabilitiesPackage{Name: "gentle-ai", Version: "3.1.0", ReleaseChannel: "stable"}
	surface.Build = ReviewCapabilitiesBuild{
		ID:            reviewCapabilitiesBuildDigest("3.1.0", ReviewCapabilitiesBuild{GoVersion: "go1.25.0", VCSModified: "false"}),
		GoVersion:     "go1.25.0",
		ModuleVersion: "v3.1.0",
		VCS:           "git",
		VCSRevision:   "0123456789abcdef0123456789abcdef01234567",
		VCSTime:       "2026-09-25T12:00:00Z",
		VCSModified:   "false",
	}
	surface.Executable = ReviewCapabilitiesExecutable{
		SHA256:       "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Evidence:     "self-reported",
		Verification: "compare-with-published-manifest",
	}
	surfaceBytes, err := json.Marshal(surface)
	if err != nil {
		t.Fatal(err)
	}
	validatePublishedReviewSchema(t, capSchema, surfaceBytes)
}

func TestDualContract_SDDStatusDualSerialization(t *testing.T) {
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
