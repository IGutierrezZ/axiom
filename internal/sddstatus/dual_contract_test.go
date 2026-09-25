package sddstatus

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/consentenvelope"
)

func TestValidateStatusContractDualSupport(t *testing.T) {
	tests := []struct {
		name     string
		contract string
		wantErr  bool
	}{
		{
			name:     "contrato legado v2 valido",
			contract: StatusContractV2,
			wantErr:  false,
		},
		{
			name:     "contrato canonico axiom v2 valido",
			contract: AxiomStatusContractV2,
			wantErr:  false,
		},
		{
			name:     "contrato canonico axiom v1 valido",
			contract: AxiomStatusContractV1,
			wantErr:  false,
		},
		{
			name:     "contrato legado v1 no soportado",
			contract: "gentle-ai.sdd-status/v1",
			wantErr:  true,
		},
		{
			name:     "contrato desconocido rechazado",
			contract: "unsupported.sdd-status/v9",
			wantErr:  true,
		},
		{
			name:     "contrato vacio rechazado",
			contract: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStatusContract(tt.contract)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateStatusContract(%q) error = %v, wantErr %v", tt.contract, err, tt.wantErr)
			}
		})
	}
}

func TestProjectStatusV2SchemaDualSupport(t *testing.T) {
	baseStatus := Status{
		SchemaVersion:   SchemaVersion,
		ArtifactStore:   ArtifactStoreOpenSpec,
		ApplyState:      ApplyReady,
		NextRecommended: "tasks",
		Artifacts: map[string]ArtifactState{
			"proposal":      ArtifactDone,
			"specs":         ArtifactDone,
			"design":        ArtifactDone,
			"tasks":         ArtifactMissing,
			"applyProgress": ArtifactMissing,
			"verifyReport":  ArtifactMissing,
		},
	}

	tests := []struct {
		name           string
		schemaName     string
		wantSchemaName string
		wantErr        bool
	}{
		{
			name:           "schema legado gentle-ai aceptado",
			schemaName:     LegacySchemaName,
			wantSchemaName: LegacySchemaName,
			wantErr:        false,
		},
		{
			name:           "schema canonico axiom aceptado",
			schemaName:     AxiomSchemaName,
			wantSchemaName: AxiomSchemaName,
			wantErr:        false,
		},
		{
			name:           "schema desconocido rechazado",
			schemaName:     "other.sdd-status",
			wantSchemaName: "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := baseStatus
			s.SchemaName = tt.schemaName
			projected, err := ProjectStatusV2(s)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ProjectStatusV2() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && projected.SchemaName != tt.wantSchemaName {
				t.Errorf("ProjectStatusV2().SchemaName = %q, want %q", projected.SchemaName, tt.wantSchemaName)
			}
		})
	}
}

func TestSDDIntegrationConsentResultDualValidation(t *testing.T) {
	tests := []struct {
		name     string
		envelope SDDIntegrationConsentResult
		wantErr  bool
	}{
		{
			name: "envelope legado gentle-ai valido",
			envelope: SDDIntegrationConsentResult{
				Schema:       SDDIntegrationConsentSchema,
				Contract:     SDDIntegrationContractV1,
				Operation:    sddConsentOperation,
				Action:       sddConsentActionRequired,
				Blocking:     true,
				Change:       "mi-feature",
				MissingRoots: []string{"/tmp/repo/extra"},
				Headline:     "Autorizacion requerida",
				Reason:       "Escritura fuera de raiz autorizada",
				Value:        "Se requiere confirmacion humana",
				Evidence:     []string{"Ruta detectada: /tmp/repo/extra"},
				Choices: []consentenvelope.Choice{
					{
						Answer:     sddConsentAnswerGranted,
						Label:      "Conceder",
						Effect:     "Autorizar temporalmente",
						Invocation: `gentle-ai sdd-attempt grant --cwd /tmp/repo --change mi-feature --actor user --reason test --request-id req-1 --change-instance ci-1 --root "/tmp/repo/extra"`,
					},
					{
						Answer:     sddConsentAnswerDeclined,
						Label:      "Declinar",
						Effect:     "Mantener bloqueo",
						Invocation: "gentle-ai sdd-status mi-feature --cwd /tmp/repo",
					},
				},
				OffPath: consentenvelope.OffPath{
					Command: "gentle-ai sdd-status mi-feature --cwd /tmp/repo",
					Note:    "Ajustar tasks.md para permanecer dentro de los limites",
				},
			},
			wantErr: false,
		},
		{
			name: "envelope canonico axiom valido",
			envelope: SDDIntegrationConsentResult{
				Schema:       AxiomSDDIntegrationConsentSchema,
				Contract:     AxiomSDDIntegrationContractV1,
				Operation:    sddConsentOperation,
				Action:       sddConsentActionRequired,
				Blocking:     true,
				Change:       "mi-feature",
				MissingRoots: []string{"/tmp/repo/extra"},
				Headline:     "Autorizacion requerida",
				Reason:       "Escritura fuera de raiz autorizada",
				Value:        "Se requiere confirmacion humana",
				Evidence:     []string{"Ruta detectada: /tmp/repo/extra"},
				Choices: []consentenvelope.Choice{
					{
						Answer:     sddConsentAnswerGranted,
						Label:      "Conceder",
						Effect:     "Autorizar temporalmente",
						Invocation: `axiom sdd-attempt grant --cwd /tmp/repo --change mi-feature --actor user --reason test --request-id req-1 --change-instance ci-1 --root "/tmp/repo/extra"`,
					},
					{
						Answer:     sddConsentAnswerDeclined,
						Label:      "Declinar",
						Effect:     "Mantener bloqueo",
						Invocation: "axiom sdd-status mi-feature --cwd /tmp/repo",
					},
				},
				OffPath: consentenvelope.OffPath{
					Command: "axiom sdd-status mi-feature --cwd /tmp/repo",
					Note:    "Ajustar tasks.md para permanecer dentro de los limites",
				},
			},
			wantErr: false,
		},
		{
			name: "schema desconocido rechazado",
			envelope: SDDIntegrationConsentResult{
				Schema:       "desconocido.consent/v1",
				Contract:     AxiomSDDIntegrationContractV1,
				Operation:    sddConsentOperation,
				Action:       sddConsentActionRequired,
				Blocking:     true,
				Change:       "mi-feature",
				MissingRoots: []string{"/tmp/repo/extra"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.envelope.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("SDDIntegrationConsentResult.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSDDGovernanceGateResultDualValidation(t *testing.T) {
	tests := []struct {
		name     string
		envelope SDDGovernanceGateResult
		wantErr  bool
	}{
		{
			name: "envelope gobernanza legado gentle-ai valido",
			envelope: SDDGovernanceGateResult{
				Schema:    SDDGovernanceGateSchema,
				Contract:  SDDGovernanceContractV1,
				Operation: gateOperation,
				Action:    gateActionRequired,
				Blocking:  true,
				Change:    "mi-cambio",
				Gate:      "spec",
				Criteria:  []string{"¿Cubre intencion?"},
				Headline:  "Revision de compuerta spec",
				Reason:    "Validacion de arquitectura requerida",
				Value:     "Compuerta bloqueante",
				Evidence:  []string{"Documento spec.md completo"},
				Choices: []consentenvelope.Choice{
					{
						Answer:     gateAnswerApproved,
						Label:      "Aprobar",
						Effect:     "Aprobar compuerta",
						Invocation: "axiom sdd gate record --cwd /tmp/repo --change mi-cambio --gate spec --decision approved --reason ok",
					},
					{
						Answer:     gateAnswerRejected,
						Label:      "Rechazar",
						Effect:     "Rechazar compuerta",
						Invocation: "axiom sdd gate record --cwd /tmp/repo --change mi-cambio --gate spec --decision rejected --reason necesita_cambios",
					},
				},
				OffPath: consentenvelope.OffPath{
					Command: "axiom sdd status mi-cambio --cwd /tmp/repo",
					Note:    "Inspeccionar estado",
				},
			},
			wantErr: false,
		},
		{
			name: "envelope gobernanza canonico axiom valido",
			envelope: SDDGovernanceGateResult{
				Schema:    AxiomSDDGovernanceGateSchema,
				Contract:  AxiomSDDGovernanceContractV1,
				Operation: gateOperation,
				Action:    gateActionRequired,
				Blocking:  true,
				Change:    "mi-cambio",
				Gate:      "spec",
				Criteria:  []string{"¿Cubre intencion?"},
				Headline:  "Revision de compuerta spec",
				Reason:    "Validacion de arquitectura requerida",
				Value:     "Compuerta bloqueante",
				Evidence:  []string{"Documento spec.md completo"},
				Choices: []consentenvelope.Choice{
					{
						Answer:     gateAnswerApproved,
						Label:      "Aprobar",
						Effect:     "Aprobar compuerta",
						Invocation: "axiom sdd gate record --cwd /tmp/repo --change mi-cambio --gate spec --decision approved --reason ok",
					},
					{
						Answer:     gateAnswerRejected,
						Label:      "Rechazar",
						Effect:     "Rechazar compuerta",
						Invocation: "axiom sdd gate record --cwd /tmp/repo --change mi-cambio --gate spec --decision rejected --reason necesita_cambios",
					},
				},
				OffPath: consentenvelope.OffPath{
					Command: "axiom sdd status mi-cambio --cwd /tmp/repo",
					Note:    "Inspeccionar estado",
				},
			},
			wantErr: false,
		},
		{
			name: "envelope gobernanza contrato no soportado",
			envelope: SDDGovernanceGateResult{
				Schema:    AxiomSDDGovernanceGateSchema,
				Contract:  "contrato.invalido/v1",
				Operation: gateOperation,
				Action:    gateActionRequired,
				Blocking:  true,
				Change:    "mi-cambio",
				Gate:      "spec",
				Criteria:  []string{"¿Cubre intencion?"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.envelope.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("SDDGovernanceGateResult.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
