package sddstatus

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	rfDeltaOK = "## ADDED Requirements\n\n### Requirement: Alta de usuario (RF-USU-02)\n\nEl sistema da de alta usuarios.\n\n#### Scenario: Alta correcta\n\n- **CUANDO** se da de alta\n- **ENTONCES** queda registrado\n"
	rfTasks   = "- [ ] 1.1 Trabajo\n  Cubre: RF-USU-02\n"
)

// isoProjectWith monta un proyecto embebido con el bloque `sdd:` dado, sin ningún cambio.
func isoProjectWith(t *testing.T, sddBlock string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+sddBlock)
	return ws
}

// REQ-29.3, RF-ISO-04 y RF-RID-05: con `sdd.id_pattern: "RF-<DOM>-<NN>"` un cambio ISO
// con identificadores del patrón no tiene motivos de identificador ni de cobertura.
func TestISOWithDeclaredPatternHasNoIdentifierOrCoverageReasons(t *testing.T) {
	t.Parallel()
	ws := isoProjectWith(t, "sdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"usuarios": rfDeltaOK}, rfTasks, "")
	status := resolveFrom(t, ws.Master, "demo")
	wantTokens(t, status)
	if status.NextRecommended != "apply" {
		t.Errorf("nextRecommended = %q, se esperaba apply: %v", status.NextRecommended, status.BlockedReasons)
	}
}

// REQ-29.3: un nombre sin identificador del patrón bloquea nombrando el patrón efectivo.
func TestISOWithDeclaredPatternNamesThePatternInReasons(t *testing.T) {
	t.Parallel()
	ws := isoProjectWith(t, "sdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	delta := strings.Replace(rfDeltaOK, " (RF-USU-02)", " (REQ-30.1)", 1)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"usuarios": delta}, rfTasks, "")
	status := resolveFrom(t, ws.Master, "demo")
	wantTokens(t, status, "iso_requirement_id", "iso_cubre_unknown")
	joined := strings.Join(status.BlockedReasons, "\n")
	if !strings.Contains(joined, "identificador RF-<DOM>-<NN>") || strings.Contains(joined, "identificador REQ-<n>.<m>") {
		t.Errorf("el motivo debe nombrar el patrón efectivo y no el de por defecto: %v", status.BlockedReasons)
	}
}

// RF-RID-01 (lectura en vivo, D-04): un cambio ya sellado cuyo axiom.yaml cambia de patrón se
// evalúa con el patrón nuevo y el motivo lo nombra.
func TestISOEvaluatesWithTheLivePattern(t *testing.T) {
	t.Parallel()
	ws := isoProjectWith(t, "sdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"usuarios": rfDeltaOK}, rfTasks, "")
	wantTokens(t, resolveFrom(t, ws.Master, "demo"))

	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+"sdd:\n  iso: true\n  id_pattern: \"HU-<NNN>\"\n")
	status := resolveFrom(t, ws.Master, "demo")
	wantTokens(t, status, "iso_requirement_id", "iso_cubre_unknown")
	if joined := strings.Join(status.BlockedReasons, "\n"); !strings.Contains(joined, "HU-<NNN>") {
		t.Errorf("el motivo debe nombrar HU-<NNN>: %v", status.BlockedReasons)
	}
}

// RF-CLI-11 y RF-RID-08 (D-07): con el patrón inaccesible la etapa `specs` lleva
// `iso_declaration_invalid` en lugar de las reglas de identificador, `tasks` no se evalúa y
// no hay `iso.matrix`.
func TestISOWithInaccessiblePattern(t *testing.T) {
	t.Parallel()
	for name, block := range map[string]string{
		"patrón inválido":        "sdd:\n  iso: true\n  id_pattern: \"<m><m>\"\n",
		"clave desconocida":      "sdd:\n  iso: true\n  id_patern: \"RF-<m>\"\n",
		"iso con un valor roto":  "sdd:\n  iso: quizás\n",
		"sdd que no es un mapeo": "sdd:\n  - iso\n",
	} {
		t.Run(name, func(t *testing.T) {
			ws := isoProjectWith(t, "sdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
			// El cambio se sella con un proyecto sano y el axiom.yaml se rompe después.
			seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"usuarios": rfDeltaOK}, rfTasks, "")
			workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+block)

			status := resolveFrom(t, ws.Master, "demo")
			wantTokens(t, status, "iso_declaration_invalid")
			joined := strings.Join(status.BlockedReasons, "\n")
			if strings.Contains(joined, "iso_requirement_id") || strings.Contains(joined, "iso_cubre") || strings.Contains(joined, "iso_coverage") {
				t.Errorf("no deben evaluarse las reglas de identificador ni de cobertura: %v", status.BlockedReasons)
			}
			if status.ISO == nil || status.ISO.Project != "invalid" {
				t.Fatalf("iso.project debe ser invalid: %+v", status.ISO)
			}
			if status.ISO.Matrix != nil {
				t.Errorf("sin patrón no hay iso.matrix: %+v", status.ISO.Matrix)
			}
			if status.NextRecommended != "spec" {
				t.Errorf("nextRecommended = %q, se esperaba spec", status.NextRecommended)
			}
		})
	}
}

// REQ-29.3: sin declaración los textos del motivo y de su continuación son idénticos byte a
// byte a los de antes (`REQ-<n>.<m>`).
func TestISODefaultPatternReasonTextsAreUnchanged(t *testing.T) {
	t.Parallel()
	ws := isoProject(t)
	seedISOChange(t, ws, "demo", "# Propuesta\n", map[string]string{"pagos": deltaAddedNoID}, tasksPlain, "")
	status := resolveFrom(t, ws.Master, "demo")
	wantTokens(t, status, "iso_requirement_id")
	want := "blocked(iso_requirement_id): dominio \"pagos\", sección ADDED, requisito \"Cobro con IVA\": el nombre no contiene un identificador REQ-<n>.<m>; añade o corrige en el nombre un identificador REQ-<n>.<m>"
	found := false
	for _, reason := range status.BlockedReasons {
		if reason == want {
			found = true
		}
	}
	if !found {
		t.Errorf("falta el motivo idéntico al de siempre:\n%s\nen %v", want, status.BlockedReasons)
	}
}
