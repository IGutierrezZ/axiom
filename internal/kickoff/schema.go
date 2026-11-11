package kickoff

import (
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"gopkg.in/yaml.v3"
)

// Closed vocabularies for every enum field this package owns. A value
// absent from its set is rejected with a named error; none of these maps
// degrades an unrecognised value to its nearest known member.
var (
	validFlowModes = map[FlowMode]bool{
		FlowODD: true,
		FlowSDD: true,
	}

	validExecutionStyles = map[ExecutionStyle]bool{
		ExecutionContinuous:   true,
		ExecutionCheckpointed: true,
	}

	validHandoffPolicies = map[HandoffPolicy]bool{
		HandoffNone:          true,
		HandoffPerCheckpoint: true,
	}

	validDocsPublications = map[DocsPublication]bool{
		DocsPublicationAuto: true,
	}

	validGatePolicies = map[multirole.GatePolicy]bool{
		multirole.PolicyBlocking: true,
		multirole.PolicyDeferred: true,
		multirole.PolicyOptional: true,
	}

	validDeploymentTargets = map[string]bool{
		"local":      true,
		"staging":    true,
		"production": true,
	}

	validEvidenceKinds = map[EvidenceKind]bool{
		EvidencePRMerged:    true,
		EvidenceDeployment:  true,
		EvidenceAttestation: true,
	}

	validGateDecisions = map[GateDecision]bool{
		DecisionApproved: true,
		DecisionRejected: true,
	}

	fixedGateKeys = map[GateKey]bool{
		GateSpec:        true,
		GateDesign:      true,
		GateTasks:       true,
		GateIntegration: true,
	}
)

// ParseKickoff decodes and validates a kickoff.yaml document. It never
// returns a zero-value Kickoff on malformed input: both a decode failure
// and a failed Validate() surface as a named error, never a silent zero
// value that would pass through as "no gates configured" (T-10).
func ParseKickoff(data []byte) (Kickoff, error) {
	var k Kickoff
	if err := yaml.Unmarshal(data, &k); err != nil {
		return Kickoff{}, fmt.Errorf("decodificar kickoff.yaml: %w", err)
	}
	if err := k.Validate(); err != nil {
		return Kickoff{}, err
	}
	return k, nil
}

// Validate checks that every enum field of the sealed document belongs to
// its closed vocabulary declared above. An unrecognised value is always
// rejected, never coerced to the nearest known one.
func (k Kickoff) Validate() error {
	if k.Schema != KickoffSchemaV1 {
		return fmt.Errorf("kickoff: esquema desconocido %q (se esperaba %q)", k.Schema, KickoffSchemaV1)
	}
	if !validFlowModes[k.Config.FlowMode] {
		return fmt.Errorf("kickoff: flow_mode desconocido %q", k.Config.FlowMode)
	}
	if !validExecutionStyles[k.Config.ExecutionStyle] {
		return fmt.Errorf("kickoff: execution_style desconocido %q", k.Config.ExecutionStyle)
	}
	// handoff_policy is deprecated (REQ-26.9): an empty value is valid, a
	// value outside the closed vocabulary is still rejected.
	if k.Config.HandoffPolicy != "" && !validHandoffPolicies[k.Config.HandoffPolicy] {
		return fmt.Errorf("kickoff: handoff_policy desconocido %q", k.Config.HandoffPolicy)
	}
	// docs_publication (REQ-27.4): vacio o auto, y auto solo acompana a continuous.
	if k.Config.DocsPublication != "" {
		if !validDocsPublications[k.Config.DocsPublication] {
			return fmt.Errorf("kickoff: docs_publication desconocido %q", k.Config.DocsPublication)
		}
		if k.Config.ExecutionStyle != ExecutionContinuous {
			return fmt.Errorf("kickoff: docs_publication %q solo es valido con execution_style continuous", k.Config.DocsPublication)
		}
	}
	for _, role := range k.Config.Roles {
		if !validGatePolicies[role.GatePolicy] {
			return fmt.Errorf("kickoff: gate_policy desconocido %q para el rol %q", role.GatePolicy, role.Role)
		}
	}
	if !validDeploymentTargets[k.Lifecycle.DeploymentTarget] {
		return fmt.Errorf("kickoff: deployment_target desconocido %q", k.Lifecycle.DeploymentTarget)
	}
	if k.Git != nil {
		return k.Git.Validate()
	}
	return nil
}

// ErrContinuousRequiresFullstack identifica el rechazo de sellado de D7/D8:
// execution_style continuous solo admite el roster exacto [fullstack]. Se
// envuelve con el roster recibido y las dos salidas; errors.Is lo reconoce.
var ErrContinuousRequiresFullstack = errors.New("execution_style continuous solo admite el rol único fullstack")

// checkContinuousRoster es la regla pura de sellado: nil si style no es
// continuous o si roles es exactamente ["fullstack"]. La política de compuerta
// no se comprueba (DD-02): continuous no tiene compuertas. fromSessionPace solo
// cambia el texto del mensaje.
func checkContinuousRoster(style ExecutionStyle, roles []string, fromSessionPace bool) error {
	if style != ExecutionContinuous {
		return nil
	}
	if len(roles) == 1 && roles[0] == "fullstack" {
		return nil
	}
	prefix := ""
	if fromSessionPace {
		prefix = "--from-session-pace auto deriva continuous, que solo admite el rol único fullstack: "
	}
	return fmt.Errorf("kickoff seal: %s%w (roles recibidos: %s); sella con --execution-style checkpointed para conservar esos roles, o quita las banderas --role para sellar con fullstack",
		prefix, ErrContinuousRequiresFullstack, strings.Join(roles, ", "))
}

// ValidateForSeal es Validate() más la regla de sellado de continuous. Solo la
// usa el camino de sellado explícito: ParseKickoff y Load siguen con Validate()
// para leer cualquier kickoff ya sellado o archivado (D7). El retrosellado
// (SealedBy == SealedByInferred) queda exento (P2, REQ-21.4).
func (k Kickoff) ValidateForSeal() error {
	if err := k.Validate(); err != nil {
		return err
	}
	if k.SealedBy == SealedByInferred {
		return nil
	}
	roles := make([]string, 0, len(k.Config.Roles))
	for _, role := range k.Config.Roles {
		roles = append(roles, role.Role)
	}
	return checkContinuousRoster(k.Config.ExecutionStyle, roles, false)
}

// ParseGateLedger decodes and validates a gates.yaml document, applying the
// same "never a silent zero value" discipline as ParseKickoff.
func ParseGateLedger(data []byte) (GateLedger, error) {
	var l GateLedger
	if err := yaml.Unmarshal(data, &l); err != nil {
		return GateLedger{}, fmt.Errorf("decodificar gates.yaml: %w", err)
	}
	if err := l.Validate(); err != nil {
		return GateLedger{}, err
	}
	return l, nil
}

// Validate checks the ledger schema and every record it holds.
func (l GateLedger) Validate() error {
	if l.Schema != GateLedgerSchemaV1 {
		return fmt.Errorf("gates: esquema desconocido %q (se esperaba %q)", l.Schema, GateLedgerSchemaV1)
	}
	for i, record := range l.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("gates: registro %d invalido: %w", i, err)
		}
	}
	return nil
}

// Validate checks one gate record: its gate key belongs to the closed
// vocabulary (the four fixed keys, or a role-apply:<role> key built by
// RoleApplyGate), its decision is a recognised verdict, and, when present,
// its evidence kind is one of the three declared classes.
func (r GateRecord) Validate() error {
	if r.Gate == "" {
		return fmt.Errorf("gate record: la clave de compuerta esta vacia")
	}
	if !fixedGateKeys[r.Gate] && !strings.HasPrefix(string(r.Gate), roleApplyGatePrefix) {
		return fmt.Errorf("gate record: clave de compuerta desconocida %q", r.Gate)
	}
	if !validGateDecisions[r.Decision] {
		return fmt.Errorf("gate record: decision desconocida %q", r.Decision)
	}
	if r.EvidenceKind != "" && !validEvidenceKinds[r.EvidenceKind] {
		return fmt.Errorf("gate record: evidence_kind desconocido %q", r.EvidenceKind)
	}
	return nil
}
