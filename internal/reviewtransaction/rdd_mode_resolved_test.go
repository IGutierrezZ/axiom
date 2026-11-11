package reviewtransaction

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// #5533: the resolved default and the fail-closed projection are identical in
// every exported field, so only the explicit indicator can tell a resolution
// from a refusal. A discriminator inferred from the field values cannot.
func TestRDDModeStatusResolvedDistinguishesTheResolvedDefaultFromTheFailClosedProjection(t *testing.T) {
	resolved := rddModeStatus(RDDModeUnset, RDDModeUnset, rddModeOverrideRecord{}, false)
	failClosed := failedClosedRDDModeStatus(RDDModeSourceDefault)

	if !resolved.Resolved() {
		t.Fatalf("the resolved default reports itself as unresolved: %#v", resolved)
	}
	if failClosed.Resolved() {
		t.Fatalf("the fail-closed projection reports itself as resolved: %#v", failClosed)
	}

	// Their exported surface is byte-identical: this is the premise that makes
	// an inferred discriminator impossible.
	resolvedJSON, err := json.Marshal(resolved)
	if err != nil {
		t.Fatal(err)
	}
	failClosedJSON, err := json.Marshal(failClosed)
	if err != nil {
		t.Fatal(err)
	}
	if string(resolvedJSON) != string(failClosedJSON) {
		t.Fatalf("resolved default JSON = %s, fail-closed JSON = %s; the indicator must not reach the contract", resolvedJSON, failClosedJSON)
	}
	if strings.Contains(string(resolvedJSON), "resolved") {
		t.Fatalf("the indicator leaked into the JSON contract: %s", resolvedJSON)
	}
}

// Every status ResolveRDDMode returns beside a nil error is resolved, and every
// one it returns beside an error is not, including the tolerant path that
// swallows an override read failure while the mode is off.
func TestResolveRDDModeMarksOnlySuccessfulResolutionsAsResolved(t *testing.T) {
	ctx := context.Background()
	repo := initSnapshotRepo(t)

	status, err := ResolveRDDMode(ctx, repo, RDDGlobalMode{})
	if err != nil || !status.Resolved() {
		t.Fatalf("default resolution = %#v, %v; want resolved", status, err)
	}

	status, err = ResolveRDDMode(ctx, repo, RDDGlobalMode{Value: "not-a-mode"})
	if err == nil || status.Resolved() {
		t.Fatalf("unintelligible global = %#v, %v; want an error and an unresolved projection", status, err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	status, err = ResolveRDDMode(cancelled, repo, RDDGlobalMode{})
	if err == nil || status.Resolved() {
		t.Fatalf("cancelled resolution = %#v, %v; want an error and an unresolved projection", status, err)
	}
}
