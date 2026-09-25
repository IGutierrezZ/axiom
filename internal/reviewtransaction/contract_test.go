package reviewtransaction

import (
	"errors"
	"testing"
)

func TestResolveReviewContract(t *testing.T) {
	tests := []struct {
		name          string
		requested     string
		wantCanonical string
		wantLegacy    bool
		wantErr       bool
		errTarget     error
	}{
		{
			name:          "Axiom canónico review-integration/v2",
			requested:     "axiom.review-integration/v2",
			wantCanonical: AxiomReviewIntegrationV2Contract,
			wantLegacy:    false,
			wantErr:       false,
		},
		{
			name:          "Legado gentle-ai review-integration/v2",
			requested:     "gentle-ai.review-integration/v2",
			wantCanonical: AxiomReviewIntegrationV2Contract,
			wantLegacy:    true,
			wantErr:       false,
		},
		{
			name:          "Legado gentle-ai review-integration/v1",
			requested:     "gentle-ai.review-integration/v1",
			wantCanonical: LegacyReviewIntegrationV1Contract,
			wantLegacy:    true,
			wantErr:       false,
		},
		{
			name:          "Axiom consent v3",
			requested:     "axiom.review-integration.consent/v3",
			wantCanonical: AxiomReviewIntegrationConsentV3Contract,
			wantLegacy:    false,
			wantErr:       false,
		},
		{
			name:          "Legado consent v3",
			requested:     "gentle-ai.review-integration.consent/v3",
			wantCanonical: AxiomReviewIntegrationConsentV3Contract,
			wantLegacy:    true,
			wantErr:       false,
		},
		{
			name:          "Axiom assessment v1",
			requested:     "axiom.review-assessment/v1",
			wantCanonical: AxiomReviewAssessmentV1Contract,
			wantLegacy:    false,
			wantErr:       false,
		},
		{
			name:          "Legado assessment v1",
			requested:     "gentle-ai.review-assessment/v1",
			wantCanonical: AxiomReviewAssessmentV1Contract,
			wantLegacy:    true,
			wantErr:       false,
		},
		{
			name:          "Axiom acknowledged v1",
			requested:     "axiom.review-acknowledged/v1",
			wantCanonical: AxiomReviewAcknowledgedV1Contract,
			wantLegacy:    false,
			wantErr:       false,
		},
		{
			name:          "Legado acknowledged v1",
			requested:     "gentle-ai.review-acknowledged/v1",
			wantCanonical: AxiomReviewAcknowledgedV1Contract,
			wantLegacy:    true,
			wantErr:       false,
		},
		{
			name:          "Contrato vacío",
			requested:     "",
			wantCanonical: "",
			wantLegacy:    false,
			wantErr:       true,
			errTarget:     ErrEmptyContract,
		},
		{
			name:          "Contrato con espacios",
			requested:     " axiom.review-integration/v2",
			wantCanonical: "",
			wantLegacy:    false,
			wantErr:       true,
			errTarget:     ErrUnsupportedContract,
		},
		{
			name:          "Contrato no soportado",
			requested:     "invalid.review-integration/v99",
			wantCanonical: "",
			wantLegacy:    false,
			wantErr:       true,
			errTarget:     ErrUnsupportedContract,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCanonical, gotLegacy, err := ResolveReviewContract(tt.requested)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveReviewContract(%q) error = %v, wantErr %v", tt.requested, err, tt.wantErr)
			}
			if tt.wantErr && tt.errTarget != nil && !errors.Is(err, tt.errTarget) {
				t.Errorf("ResolveReviewContract(%q) error = %v, want target %v", tt.requested, err, tt.errTarget)
			}
			if gotCanonical != tt.wantCanonical {
				t.Errorf("ResolveReviewContract(%q) canonical = %v, esperado %v", tt.requested, gotCanonical, tt.wantCanonical)
			}
			if gotLegacy != tt.wantLegacy {
				t.Errorf("ResolveReviewContract(%q) isLegacy = %v, esperado %v", tt.requested, gotLegacy, tt.wantLegacy)
			}
		})
	}
}

func TestMatchReviewDialect(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		isLegacy   bool
		expected   string
	}{
		{
			name:       "convertir canónico a legado",
			identifier: "axiom.review-integration/v2",
			isLegacy:   true,
			expected:   "gentle-ai.review-integration/v2",
		},
		{
			name:       "convertir legado a canónico",
			identifier: "gentle-ai.review-integration/v2",
			isLegacy:   false,
			expected:   "axiom.review-integration/v2",
		},
		{
			name:       "preservar canónico cuando no es legado",
			identifier: "axiom.review-integration.status/v9",
			isLegacy:   false,
			expected:   "axiom.review-integration.status/v9",
		},
		{
			name:       "preservar legado cuando es legado",
			identifier: "gentle-ai.review-integration.status/v9",
			isLegacy:   true,
			expected:   "gentle-ai.review-integration.status/v9",
		},
		{
			name:       "identificador neutro sin prefijo conocido",
			identifier: "other-identifier",
			isLegacy:   true,
			expected:   "other-identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchReviewDialect(tt.identifier, tt.isLegacy)
			if got != tt.expected {
				t.Errorf("MatchReviewDialect(%q, %v) = %v, esperado %v", tt.identifier, tt.isLegacy, got, tt.expected)
			}
		})
	}
}
