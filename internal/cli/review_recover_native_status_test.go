package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// TestStatusRecoverNativeFixtureIsPinnedAndValid fija el fixture nuevo de
// recuperación nativa (cuatro entradas requeridas) y comprueba que valida contra
// el esquema status-v9 publicado y contra la validación de Go. Los fixtures v1
// de recuperación, fijados por hash, no se tocan: describen la forma explícita
// de siete entradas.
func TestStatusRecoverNativeFixtureIsPinnedAndValid(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration")
	payload, err := os.ReadFile(filepath.Join(root, "v2", "fixtures", "status-v9-recover-native.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	if actual, want := hex.EncodeToString(digest[:]), "3fe0151d302e48e6732026a4f9df811e20a375b12190635e4d8244fd5457c5b0"; actual != want {
		t.Fatalf("status-v9-recover-native.fixture.json digest = %s, want %s", actual, want)
	}
	validatePublishedReviewSchema(t, compileWholeNativeStatusSchema(t, "status-v9.schema.json"), payload)
	var status ReviewTargetStatusResult
	decodeStrictReviewJSON(t, payload, &status)
	if err := status.Validate(); err != nil {
		t.Fatalf("native recover fixture: %v", err)
	}
	if status.Eligibility == nil || len(status.Eligibility.AllowedActions) != 1 ||
		!reflect.DeepEqual(status.Eligibility.AllowedActions[0].RequiredInputs, []string{"predecessor_lineage", "expected_predecessor_revision", "successor_lineage", "disposition"}) {
		t.Fatalf("native recover fixture required inputs = %#v", status.Eligibility)
	}
	// El fixture v1 original sigue describiendo la forma explícita de siete entradas.
	legacy, err := os.ReadFile(filepath.Join(root, "v1", "fixtures", "status-v2-recover.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(legacy, []byte(`"maintainer_authorization"`)) {
		t.Fatal("legacy recover fixture changed")
	}
}

// TestNativeRecoveryIgnoresUnrelatedAuthorityDamage comprueba que una
// autoridad ajena y dañada no bloquea una recuperación nativa sana ni oculta a
// un hijo ya registrado.
func TestNativeRecoveryIgnoresUnrelatedAuthorityDamage(t *testing.T) {
	reviewEnabledHome(t)
	repo, predecessor := invalidatedRecoverySelfDerivationPredecessor(t, "healthy-recovery")
	store, err := reviewtransaction.CompactAuthoritativeStore(t.Context(), repo, predecessor.State.LineageID)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := reviewtransaction.CompactAuthoritativeStore(t.Context(), repo, "unrelated-damaged")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(foreign.StatePath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign.StatePath(), []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := foreign.LoadContext(t.Context()); err == nil || reviewtransaction.IsCompactAuthorityOperationalFailure(err) {
		t.Fatalf("el fixture debe ser un daño de contenido en cuarentena: %v", err)
	}
	status := selectorTransitionStatus(t, repo, "--lineage", predecessor.State.LineageID)
	if status.Action != reviewtransaction.TargetStatusActionRecover || status.NextTransition.Execute == nil || len(status.NextTransition.Execute.Arguments) != 4 {
		t.Fatalf("un daño ajeno bloqueó la recuperación nativa sana: %#v", status.NextTransition)
	}
	var recovered ReviewRecoverResult
	decodeStrictReviewJSON(t, executeSelectorTransition(t, repo, status), &recovered)
	after, err := os.ReadFile(store.StatePath())
	foreignAfter, foreignErr := os.ReadFile(foreign.StatePath())
	if err != nil || foreignErr != nil || !bytes.Equal(before, after) || string(foreignAfter) != "not-json\n" || recovered.TargetIdentity != status.TargetIdentity {
		t.Fatalf("la recuperación impresa cambió el predecesor, la autoridad dañada o el target: %#v, %v, %v", recovered, err, foreignErr)
	}
	var output bytes.Buffer
	err = RunReview([]string{"status", "--cwd", repo, "--contract", ReviewIntegrationContractV2, "--next-transition", "--lineage", predecessor.State.LineageID}, &output)
	var failure ReviewIntegrationFailure
	decodeStrictReviewJSON(t, output.Bytes(), &failure)
	if err == nil || !strings.Contains(failure.Cause, "predecessor already has successor "+recovered.LineageID) {
		t.Fatalf("el daño ajeno ocultó a un hijo válido: %v, %#v", err, failure)
	}
}

// TestNativeRecoveryPropagatesOperationalAuthorityFailure comprueba que no
// poder observar una autoridad ajena se propaga en lugar de tratarse como
// ausencia.
func TestNativeRecoveryPropagatesOperationalAuthorityFailure(t *testing.T) {
	reviewEnabledHome(t)
	repo, predecessor := invalidatedRecoverySelfDerivationPredecessor(t, "healthy-operational-probe")
	status := selectorTransitionStatus(t, repo, "--lineage", predecessor.State.LineageID)
	store, err := reviewtransaction.CompactAuthoritativeStore(t.Context(), repo, "unobservable-foreign")
	if err != nil {
		t.Fatal(err)
	}
	// Un directorio en la ruta del registro hace fallar ReadFile de forma
	// determinista, también con permisos elevados.
	if err := os.MkdirAll(store.StatePath(), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadContext(t.Context()); err == nil || !reviewtransaction.IsCompactAuthorityOperationalFailure(err) {
		t.Fatalf("el fixture debe impedir la observación: %v", err)
	}
	successor, err := reviewStatusRecoverySuccessor(t.Context(), repo, status.NextTransition.Execute.Binding, "")
	if successor != "" || err == nil || !reviewtransaction.IsCompactAuthorityOperationalFailure(err) {
		t.Fatalf("el escaneo de propiedad ignoró el fallo operativo: successor=%q err=%v", successor, err)
	}
}

// TestNativeRecoveryStatusRefusalsAreReadOnlyAndDiagnosticRuns comprueba que
// cada rechazo de STATUS no muta, nombra el diagnóstico de solo lectura con la
// marca axiom y que ese diagnóstico se ejecuta.
func TestNativeRecoveryStatusRefusalsAreReadOnlyAndDiagnosticRuns(t *testing.T) {
	reviewEnabledHome(t)
	repo, predecessor := invalidatedRecoverySelfDerivationPredecessor(t, "native-recovery-refusals")
	t.Chdir(repo)
	store, _ := reviewtransaction.CompactAuthoritativeStore(t.Context(), repo, predecessor.State.LineageID)
	before, _ := os.ReadFile(store.StatePath())
	assertRefusal := func(extra ...string) {
		t.Helper()
		var output bytes.Buffer
		err := RunReview(append([]string{"status", "--contract", ReviewIntegrationContractV2, "--next-transition", "--lineage", predecessor.State.LineageID}, extra...), &output)
		if err == nil {
			t.Fatalf("STATUS aceptó %v: %s", extra, output.String())
		}
		var failure ReviewIntegrationFailure
		decodeStrictReviewJSON(t, output.Bytes(), &failure)
		if err := failure.Validate(); err != nil || failure.Code != "invalid_request" || failure.Phase != "preflight" {
			t.Fatalf("sobre de rechazo inválido: %#v, %v", failure, err)
		}
		_, command, named := strings.Cut(failure.Cause, "re-run: ")
		if !named || command != "axiom review inspect-authority" {
			t.Fatalf("el rechazo no nombra el diagnóstico de solo lectura: %v", failure.Cause)
		}
		if err := RunReview(reviewShellWords(t, command)[2:], io.Discard); err != nil {
			t.Fatalf("el diagnóstico impreso se rechazó: %v", err)
		}
		after, _ := os.ReadFile(store.StatePath())
		if !bytes.Equal(before, after) {
			t.Fatal("el STATUS rechazado o su diagnóstico mutó el predecesor")
		}
	}
	for _, extra := range [][]string{
		{"--recovery-authorization="}, {"--recovery-authorization=wrong"},
		{"--recovery-actor=maintainer"}, {"--recovery-reason=only-reason"}, {"--recovery-actor="},
		{"--recovery-successor-lineage=" + predecessor.State.LineageID},
	} {
		assertRefusal(extra...)
	}
	status := selectorTransitionStatus(t, repo, "--lineage", predecessor.State.LineageID)
	derived := selectorTransitionArguments(t, status)["successor-lineage"]
	// Una autoridad ajena en el nombre derivado no debe provocar búsqueda de sufijo.
	if err := runLegacyFacadeStartForTest(t, []string{"--cwd", repo, "--lineage", derived}, io.Discard); err != nil {
		t.Fatal(err)
	}
	assertRefusal()
	assertRefusal("--recovery-successor-lineage=" + derived)
	available := selectorTransitionStatus(t, repo, "--lineage", predecessor.State.LineageID, "--recovery-successor-lineage=explicit-available")
	executeSelectorTransition(t, repo, available)
	assertRefusal("--recovery-successor-lineage=another-name")
	stores, err := reviewtransaction.DiscoverCompactStores(t.Context(), repo)
	if err != nil || len(stores) != 3 {
		t.Fatalf("los rechazos bifurcaron la autoridad: %d almacenes, %v", len(stores), err)
	}
}
