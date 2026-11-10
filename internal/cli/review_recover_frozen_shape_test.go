package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// frozenShapeRecoverArgs devuelve la recuperación escalada derivada (sin
// autorización explícita) del predecesor preparado.
func frozenShapeRecoverArgs(repo string, started ReviewFacadeStartResult, predecessor reviewtransaction.CompactRecord, successor string, extra ...string) []string {
	return append([]string{"recover", "--cwd", repo, "--predecessor-lineage", started.LineageID,
		"--expected-predecessor-revision", predecessor.Revision, "--successor-lineage", successor,
		"--disposition", "escalated"}, extra...)
}

func frozenShapePolicyFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(canonicalReviewCLITempDir(t), "policy.txt")
	if err := os.WriteFile(path, []byte("private fixture policy bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRecoverEscalatedPreservesFrozenShape comprueba que el sucesor de una
// recuperación escalada sobre el mismo árbol candidato conserva la política y
// las lentes congeladas del predecesor, y que solo recalcula riesgo y líneas.
func TestRecoverEscalatedPreservesFrozenShape(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(t *testing.T) []string
		extra func(policy string) []string
	}{
		{name: "focus-y-politica-implicitos", start: func(t *testing.T) []string {
			return []string{"--focus", "resilience", "--policy", frozenShapePolicyFile(t)}
		}},
		{name: "sobrescrituras-compatibles", start: func(t *testing.T) []string {
			return []string{"--focus", "resilience", "--policy", frozenShapePolicyFile(t)}
		}, extra: func(policy string) []string { return []string{"--focus", "resilience", "--policy", policy} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reviewEnabledHome(t)
			repo := initReviewCLIRepo(t)
			startExtra := tc.start(t)
			started, predecessor, store := setupAccountingOnlyEscalation(t, repo, startExtra)
			var extra []string
			if tc.extra != nil {
				extra = tc.extra(startExtra[3])
			}
			var output bytes.Buffer
			const successor = "frozen-shape-successor"
			if err := RunReview(frozenShapeRecoverArgs(repo, started, predecessor, successor, extra...), &output); err != nil {
				t.Fatal(err)
			}
			successorStore, err := reviewtransaction.CompactAuthoritativeStore(context.Background(), repo, successor)
			if err != nil {
				t.Fatal(err)
			}
			next, err := successorStore.Load()
			if err != nil {
				t.Fatal(err)
			}
			prior := predecessor.State
			if next.State.PolicyHash != prior.PolicyHash || !reflect.DeepEqual(next.State.FrozenPolicyContent, prior.FrozenPolicyContent) ||
				!reflect.DeepEqual(next.State.SelectedLenses, prior.SelectedLenses) {
				t.Fatalf("la recuperación no conservó política y lentes congeladas: %#v", next.State)
			}
			assessment, err := (reviewtransaction.SnapshotBuilder{Repo: repo}).AssessSnapshotRisk(context.Background(), next.State.InitialSnapshot)
			if err != nil || next.State.RiskLevel != assessment.Level || next.State.OriginalChangedLines != assessment.ChangedLines {
				t.Fatalf("riesgo y líneas no se recalcularon: %#v, %v", next.State, err)
			}
			after, err := store.Load()
			if err != nil || after.Revision != predecessor.Revision {
				t.Fatalf("la recuperación cambió el predecesor: %v", err)
			}
		})
	}
}

// TestRecoverEscalatedRefusesConflictingFrozenShape comprueba que un --focus
// o --policy que contradice la forma congelada se rechaza sin mutar y que el
// rechazo imprime el comando ligado sin esos flags, con la marca axiom.
func TestRecoverEscalatedRefusesConflictingFrozenShape(t *testing.T) {
	for _, tc := range []struct {
		name, flag string
		extra      func(t *testing.T) []string
	}{
		{name: "focus", flag: "--focus", extra: func(*testing.T) []string { return []string{"--focus", "reliability"} }},
		{name: "politica", flag: "--policy", extra: func(t *testing.T) []string {
			path := filepath.Join(canonicalReviewCLITempDir(t), "other.txt")
			if err := os.WriteFile(path, []byte("another policy\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			return []string{"--policy", path}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reviewEnabledHome(t)
			repo := initReviewCLIRepo(t)
			started, predecessor, store := setupAccountingOnlyEscalation(t, repo, []string{"--focus", "resilience", "--policy", frozenShapePolicyFile(t)})
			before, err := os.ReadFile(store.StatePath())
			if err != nil {
				t.Fatal(err)
			}
			const successor = "frozen-shape-conflict"
			err = RunReview(frozenShapeRecoverArgs(repo, started, predecessor, successor, tc.extra(t)...), &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), tc.flag+" conflicts with the frozen recovery review shape") ||
				strings.Contains(err.Error(), "private fixture policy bytes") {
				t.Fatalf("rechazo de la forma congelada = %v", err)
			}
			_, command, named := strings.Cut(err.Error(), "re-run: ")
			if !named {
				t.Fatalf("el rechazo no nombra continuación: %v", err)
			}
			words := reviewShellWords(t, command)
			if len(words) < 3 || words[0] != "axiom" || words[1] != "review" {
				t.Fatalf("la continuación no es un comando de revisión: %q", command)
			}
			if strings.Contains(command, "--focus") || strings.Contains(command, "--policy") {
				t.Fatalf("la continuación conserva las sobrescrituras: %q", command)
			}
			stores, err := reviewtransaction.DiscoverCompactStores(context.Background(), repo)
			if err != nil || len(stores) != 1 {
				t.Fatalf("el rechazo creó un sucesor: %v, %v", stores, err)
			}
			// El comando imprime --cwd entrecomillado para un intérprete POSIX; se
			// sustituye por la ruta real para no depender del escapado de Windows.
			args := []string{}
			for index := 2; index < len(words); index++ {
				if words[index] == "--cwd" {
					index++
					continue
				}
				args = append(args, words[index])
			}
			if !strings.Contains(command, "--cwd") {
				t.Fatalf("la continuación perdió el --cwd ligado: %q", command)
			}
			if err := RunReview(append(args, "--cwd", repo), &bytes.Buffer{}); err != nil {
				t.Fatalf("la continuación impresa no se ejecuta: %v; comando=%q", err, command)
			}
			after, err := os.ReadFile(store.StatePath())
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("la continuación mutó el predecesor: %v", err)
			}
		})
	}
}

// TestAccountingOnlyRecoveryWithoutFrozenPolicyContentIsRefused comprueba que
// STATUS no anuncia una recuperación accounting-only imposible (política solo
// con hash) y que RECOVER la rechaza sin mutar.
func TestAccountingOnlyRecoveryWithoutFrozenPolicyContentIsRefused(t *testing.T) {
	reviewEnabledHome(t)
	repo := initReviewCLIRepo(t)
	started, predecessor, store := setupAccountingOnlyEscalation(t, repo, nil)
	predecessor.State.FrozenPolicyContent = nil
	revision, err := reviewtransaction.CompactRevisionForState(predecessor.State)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := json.MarshalIndent(reviewtransaction.CompactRecord{
		Schema: "gentle-ai.review-state-record/v2", Revision: revision, State: predecessor.State,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.StatePath(), append(persisted, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = RunReview([]string{"status", "--cwd", repo, "--contract", ReviewIntegrationContractV2, "--next-transition", "--lineage", started.LineageID}, &output)
	var failure ReviewIntegrationFailure
	decodeStrictReviewJSON(t, output.Bytes(), &failure)
	if err == nil || !strings.Contains(failure.Cause, "no frozen policy content") {
		t.Fatalf("se anunció una recuperación accounting-only imposible: %v, %#v", err, failure)
	}
	err = RunReviewRecover([]string{"--cwd", repo, "--predecessor-lineage", started.LineageID, "--expected-predecessor-revision", revision,
		"--successor-lineage", "hash-only-successor", "--disposition", "escalated"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "no frozen policy content") {
		t.Fatalf("recuperación nativa con política solo hash = %v", err)
	}
	after, _ := os.ReadFile(store.StatePath())
	stores, _ := reviewtransaction.DiscoverCompactStores(context.Background(), repo)
	if !bytes.Equal(before, after) || len(stores) != 1 {
		t.Fatal("el rechazo de política solo hash mutó la autoridad")
	}
}
