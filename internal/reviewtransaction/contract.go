package reviewtransaction

import (
	"errors"
	"fmt"
	"strings"
)

// Identificadores canónicos de Axiom para contratos de revisión RDD.
const (
	AxiomReviewIntegrationV2Contract        = "axiom.review-integration/v2"
	AxiomReviewIntegrationConsentV3Contract = "axiom.review-integration.consent/v3"
	AxiomReviewAssessmentV1Contract         = "axiom.review-assessment/v1"
	AxiomReviewAcknowledgedV1Contract       = "axiom.review-acknowledged/v1"
	AxiomReviewAuthorityStatusV1Contract    = "axiom.review-authority-status/v1"

	AxiomReviewOperationV2Contract = "axiom.review-integration.operation/v2"
	AxiomReviewFailureV2Contract   = "axiom.review-integration.failure/v2"
	AxiomReviewStartV3Contract     = "axiom.review-integration.start/v3"
	AxiomReviewStartV4Contract     = "axiom.review-integration.start/v4"
	AxiomReviewStatusV9Contract    = "axiom.review-integration.status/v9"
	AxiomReviewRepairV2Contract    = "axiom.review-integration.repair/v2"
)

// Identificadores legados para retrocompatibilidad.
const (
	LegacyReviewIntegrationV2Contract        = "gentle-ai.review-integration/v2"
	LegacyReviewIntegrationV1Contract        = "gentle-ai.review-integration/v1"
	LegacyReviewIntegrationConsentV3Contract = "gentle-ai.review-integration.consent/v3"
	LegacyReviewAssessmentV1Contract         = "gentle-ai.review-assessment/v1"
	LegacyReviewAcknowledgedV1Contract       = "gentle-ai.review-acknowledged/v1"
	LegacyReviewAuthorityStatusV1Contract    = "gentle-ai.review-authority-status/v1"

	LegacyReviewOperationV2Contract = "gentle-ai.review-integration.operation/v2"
	LegacyReviewFailureV2Contract   = "gentle-ai.review-integration.failure/v2"
	LegacyReviewStartV3Contract     = "gentle-ai.review-integration.start/v3"
	LegacyReviewStartV4Contract     = "gentle-ai.review-integration.start/v4"
	LegacyReviewStatusV9Contract    = "gentle-ai.review-integration.status/v9"
	LegacyReviewRepairV2Contract    = "gentle-ai.review-integration.repair/v2"
)

// Sentinel errors para resolución de contratos.
var (
	ErrEmptyContract       = errors.New("the review integration contract cannot be empty") // refusal:by-design world-action: caller must provide a valid non-empty contract identifier
	ErrUnsupportedContract = errors.New("unsupported review integration contract")         // refusal:by-design world-action: caller must negotiate a supported review integration contract
)

// ResolveReviewContract resuelve cualquier identificador de contrato de revisión
// (canónico de Axiom o alias legado de Gentle AI) retornando la forma canónica,
// si era una solicitud de dialecto legado, o un error si no está soportado.
func ResolveReviewContract(requested string) (canonical string, isLegacy bool, err error) {
	if requested == "" {
		return "", false, ErrEmptyContract
	}

	switch requested {
	case AxiomReviewIntegrationV2Contract:
		return AxiomReviewIntegrationV2Contract, false, nil
	case LegacyReviewIntegrationV2Contract:
		return AxiomReviewIntegrationV2Contract, true, nil
	case LegacyReviewIntegrationV1Contract:
		return LegacyReviewIntegrationV1Contract, true, nil

	case AxiomReviewIntegrationConsentV3Contract:
		return AxiomReviewIntegrationConsentV3Contract, false, nil
	case LegacyReviewIntegrationConsentV3Contract:
		return AxiomReviewIntegrationConsentV3Contract, true, nil

	case AxiomReviewAssessmentV1Contract:
		return AxiomReviewAssessmentV1Contract, false, nil
	case LegacyReviewAssessmentV1Contract:
		return AxiomReviewAssessmentV1Contract, true, nil

	case AxiomReviewAcknowledgedV1Contract:
		return AxiomReviewAcknowledgedV1Contract, false, nil
	case LegacyReviewAcknowledgedV1Contract:
		return AxiomReviewAcknowledgedV1Contract, true, nil

	case AxiomReviewAuthorityStatusV1Contract:
		return AxiomReviewAuthorityStatusV1Contract, false, nil
	case LegacyReviewAuthorityStatusV1Contract:
		return AxiomReviewAuthorityStatusV1Contract, true, nil

	case AxiomReviewOperationV2Contract:
		return AxiomReviewOperationV2Contract, false, nil
	case LegacyReviewOperationV2Contract:
		return AxiomReviewOperationV2Contract, true, nil

	case AxiomReviewFailureV2Contract:
		return AxiomReviewFailureV2Contract, false, nil
	case LegacyReviewFailureV2Contract:
		return AxiomReviewFailureV2Contract, true, nil

	case AxiomReviewStartV3Contract:
		return AxiomReviewStartV3Contract, false, nil
	case LegacyReviewStartV3Contract:
		return AxiomReviewStartV3Contract, true, nil

	case AxiomReviewStartV4Contract:
		return AxiomReviewStartV4Contract, false, nil
	case LegacyReviewStartV4Contract:
		return AxiomReviewStartV4Contract, true, nil

	case AxiomReviewStatusV9Contract:
		return AxiomReviewStatusV9Contract, false, nil
	case LegacyReviewStatusV9Contract:
		return AxiomReviewStatusV9Contract, true, nil

	case AxiomReviewRepairV2Contract:
		return AxiomReviewRepairV2Contract, false, nil
	case LegacyReviewRepairV2Contract:
		return AxiomReviewRepairV2Contract, true, nil

	default:
		return "", false, fmt.Errorf("%w %q; supported contracts: %s, %s, %s",
			ErrUnsupportedContract, requested,
			AxiomReviewIntegrationV2Contract,
			LegacyReviewIntegrationV2Contract,
			LegacyReviewIntegrationV1Contract)
	}
}

// MatchReviewDialect formatea el identificador dado (schema o contract) al dialecto
// especificado: canónico de Axiom (isLegacy=false) o legado de Gentle AI (isLegacy=true).
func MatchReviewDialect(identifier string, isLegacy bool) string {
	if isLegacy {
		if strings.HasPrefix(identifier, "axiom.") {
			return "gentle-ai." + strings.TrimPrefix(identifier, "axiom.")
		}
		return identifier
	}
	if strings.HasPrefix(identifier, "gentle-ai.") {
		return "axiom." + strings.TrimPrefix(identifier, "gentle-ai.")
	}
	return identifier
}

// IsReviewContractSupported comprueba rápidamente si un contrato es reconocido.
func IsReviewContractSupported(contract string) bool {
	_, _, err := ResolveReviewContract(contract)
	return err == nil
}
