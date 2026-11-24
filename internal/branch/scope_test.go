package branch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func yes() *bool { v := true; return &v }
func no() *bool  { v := false; return &v }

func scopeEntry(name, outcome string, merged *bool) RepositoryScope {
	repo := RepositoryScope{Name: name, Branch: &ScopeRef{Name: "feat/" + name}, Merged: merged}
	repo.SetOutcome(outcome)
	return repo
}

func unverifiableEntry(name, reason string) RepositoryScope {
	repo := scopeEntry(name, ScopeOutcomeUnverifiable, nil)
	repo.SetUnverifiable(reason, "mensaje", "", nil)
	return repo
}

// REQ-25.54: precedencia `unverifiable` > `pending_merge` > `nothing_to_verify` > `offerable`;
// `empty` y `exempt` no son pendientes y `pending` nunca es nulo.
func TestSummarizeScopeOfferPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		repos       []RepositoryScope
		wantOffer   string
		wantPending []ScopePending
	}{
		{"todo fusionado", []RepositoryScope{scopeEntry("api", ScopeOutcomeOK, yes())}, ScopeOfferOfferable, nil},
		{"dos fusionados y uno exento", []RepositoryScope{scopeEntry("api", ScopeOutcomeOK, yes()), scopeEntry("web", ScopeOutcomeOK, yes()), scopeEntry("specs", ScopeOutcomeExempt, nil)}, ScopeOfferOfferable, nil},
		{"una rama sin fusionar", []RepositoryScope{scopeEntry("api", ScopeOutcomeOK, no())}, ScopeOfferPendingMerge, []ScopePending{{Repository: "api", Branch: "feat/api", Reason: ScopeReasonNotMerged}}},
		{"ok sin indicador de fusión cuenta como sin fusionar", []RepositoryScope{scopeEntry("api", ScopeOutcomeOK, nil)}, ScopeOfferPendingMerge, []ScopePending{{Repository: "api", Branch: "feat/api", Reason: ScopeReasonNotMerged}}},
		{"todo vacío", []RepositoryScope{scopeEntry("api", ScopeOutcomeEmpty, nil), scopeEntry("web", ScopeOutcomeEmpty, nil)}, ScopeOfferNothingToVerify, nil},
		{"solo exentos", []RepositoryScope{scopeEntry("specs", ScopeOutcomeExempt, nil)}, ScopeOfferNothingToVerify, nil},
		{"sin repositorios", nil, ScopeOfferNothingToVerify, nil},
		{"un repositorio no verificable", []RepositoryScope{unverifiableEntry("web", ScopeReasonBranchMissing), scopeEntry("api", ScopeOutcomeOK, yes())}, ScopeOfferUnverifiable, []ScopePending{{Repository: "web", Branch: "feat/web", Reason: ScopeReasonBranchMissing}}},
		{"no verificable domina a sin fusionar y lista ambos", []RepositoryScope{unverifiableEntry("web", ScopeReasonBaseMissing), scopeEntry("api", ScopeOutcomeOK, no())}, ScopeOfferUnverifiable, []ScopePending{{Repository: "web", Branch: "feat/web", Reason: ScopeReasonBaseMissing}, {Repository: "api", Branch: "feat/api", Reason: ScopeReasonNotMerged}}},
		{"sin fusionar domina a vacío", []RepositoryScope{scopeEntry("api", ScopeOutcomeEmpty, nil), scopeEntry("web", ScopeOutcomeOK, no())}, ScopeOfferPendingMerge, []ScopePending{{Repository: "web", Branch: "feat/web", Reason: ScopeReasonNotMerged}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeScope(tc.repos)
			if got.Offer != tc.wantOffer || got.OfferLabel != Label(tc.wantOffer) || got.OfferLabel == "" {
				t.Errorf("Offer = %q (%q), se esperaba %q con su etiqueta", got.Offer, got.OfferLabel, tc.wantOffer)
			}
			if got.Pending == nil {
				t.Fatal("Pending es nulo: debe ser siempre una lista")
			}
			if len(got.Pending) != len(tc.wantPending) {
				t.Fatalf("Pending = %+v, se esperaban %d entradas", got.Pending, len(tc.wantPending))
			}
			for i, want := range tc.wantPending {
				p := got.Pending[i]
				if p.Repository != want.Repository || p.Branch != want.Branch || p.Reason != want.Reason || p.ReasonLabel != Label(want.Reason) || p.ReasonLabel == "" {
					t.Errorf("Pending[%d] = %+v, se esperaba %+v con su etiqueta", i, p, want)
				}
			}
		})
	}
}

func scopeJSON(t *testing.T, result ScopeResult) (string, map[string]any) {
	t.Helper()
	var buffer bytes.Buffer
	if err := result.WriteJSON(&buffer); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("la salida no es JSON: %v\n%s", err, buffer.String())
	}
	return buffer.String(), decoded
}

// REQ-25.51: el contrato lleva su propio esquema, tokens con etiqueta y `pending` como lista aun
// vacía; dos escrituras del mismo resultado dan los mismos bytes.
func TestScopeResultJSONIsStableAndLabelled(t *testing.T) {
	repo := scopeEntry("api", ScopeOutcomeOK, yes())
	repo.GitRoot = "/repos/api"
	repo.Role = RepoChangeBranch
	repo.RoleLabel = Label(string(RepoChangeBranch))
	repo.Start = NewScopeStart("aaa", ScopeStartFirstParent)
	repo.Commits = []ScopeCommit{{SHA: "bbb"}, {SHA: "ccc", Merge: true}}
	repo.SetMerged(true, LayerAncestry)
	exempt := scopeEntry("specs", ScopeOutcomeExempt, nil)
	result := NewScopeResult()
	result.Change = "x"
	result.Repositories = []RepositoryScope{repo, exempt}
	summary := SummarizeScope(result.Repositories)
	result.Summary = &summary

	raw, decoded := scopeJSON(t, result)
	again, _ := scopeJSON(t, result)
	if raw != again {
		t.Errorf("dos escrituras del mismo resultado difieren:\n%s\n%s", raw, again)
	}
	for key, want := range map[string]any{"schema": "axiom.branch-scope/v1", "operation": "scope", "status": "ready", "statusLabel": "listo"} {
		if decoded[key] != want {
			t.Errorf("%s = %v, se esperaba %v", key, decoded[key], want)
		}
	}
	summaryJSON := decoded["summary"].(map[string]any)
	if pending, ok := summaryJSON["pending"].([]any); !ok || len(pending) != 0 {
		t.Errorf("pending = %v, se esperaba una lista vacía y no nula", summaryJSON["pending"])
	}
	if summaryJSON["offer"] != "offerable" || summaryJSON["offerLabel"] == "" {
		t.Errorf("offer = %v / %v, se esperaba offerable con etiqueta", summaryJSON["offer"], summaryJSON["offerLabel"])
	}
	first := decoded["repositories"].([]any)[0].(map[string]any)
	if first["outcome"] != "ok" || first["outcomeLabel"] == "" || first["mergedBy"] != "ancestry" || first["mergedByLabel"] != "ascendencia" {
		t.Errorf("entrada ok = %v, se esperaban resultado y capa de fusión con su etiqueta", first)
	}
	if first["merged"] != true {
		t.Errorf("merged = %v, se esperaba true", first["merged"])
	}
	start := first["start"].(map[string]any)
	if start["source"] != "first_parent" || start["sourceLabel"] != Label("first_parent") {
		t.Errorf("start = %v, se esperaba la fuente first_parent con etiqueta", start)
	}
	second := decoded["repositories"].([]any)[1].(map[string]any)
	for _, absent := range []string{"merged", "start", "commits", "files", "range"} {
		if _, present := second[absent]; present {
			t.Errorf("una entrada exenta no lleva %q: %v", absent, second)
		}
	}
	if strings.Contains(raw, "\"reason\"") {
		t.Errorf("solo unverifiable lleva reason: %s", raw)
	}
}

// Un rechazo del verbo lleva el código tipado, sale con 1 y no lleva resumen ni repositorios.
func TestScopeResultRefusalCarriesTheTypedCodeAndExitsWithOne(t *testing.T) {
	refusal := &Refusal{Code: CodeScopeNotApplicable, Message: "no aplica", Continuation: "usa el carril sdd", Details: map[string]string{"change": "x"}}
	result := NewScopeResult().Refused(refusal)
	_, decoded := scopeJSON(t, result)
	if decoded["status"] != "refused" || decoded["code"] != "scope_not_applicable" || decoded["continuation"] != "usa el carril sdd" {
		t.Errorf("rechazo = %v, se esperaba refused con scope_not_applicable y su continuación", decoded)
	}
	if _, present := decoded["summary"]; present {
		t.Error("un rechazo no lleva resumen")
	}
	if result.ExitCode() != ExitRefused {
		t.Errorf("ExitCode = %d, se esperaba %d", result.ExitCode(), ExitRefused)
	}
	if git := NewScopeResult().Refused(&Refusal{Code: CodeGitFailed, Message: "m", Continuation: "c"}); git.Status != StatusError || git.ExitCode() != ExitRefused {
		t.Errorf("git_failed = %s / %d, se esperaba error y 1", git.Status, git.ExitCode())
	}
}

// REQ-25.2: una operación calculada sale con 0 aunque haya repositorios `unverifiable`.
func TestScopeResultExitsWithZeroWhenTheScopeWasComputedEvenWithUnverifiable(t *testing.T) {
	result := NewScopeResult()
	result.Repositories = []RepositoryScope{unverifiableEntry("web", ScopeReasonBranchMissing)}
	summary := SummarizeScope(result.Repositories)
	result.Summary = &summary
	if result.ExitCode() != ExitReady {
		t.Errorf("ExitCode = %d, se esperaba 0", result.ExitCode())
	}
}

// El texto del verbo es una línea por repositorio y la oferta, en castellano.
func TestScopeResultWriteTextListsEachRepositoryAndTheOffer(t *testing.T) {
	result := NewScopeResult()
	result.Change = "x"
	web := unverifiableEntry("web", ScopeReasonBranchMissing)
	web.Message = "la rama ya no está o falta un fetch autorizado"
	result.Repositories = []RepositoryScope{scopeEntry("api", ScopeOutcomeOK, yes()), web}
	summary := SummarizeScope(result.Repositories)
	result.Summary = &summary
	var stdout, stderr bytes.Buffer
	result.WriteText(&stdout, &stderr)
	out := stdout.String()
	for _, want := range []string{"Cambio: x", "api", "web", "la rama ya no está o falta un fetch autorizado", "Oferta: " + Label(ScopeOfferUnverifiable)} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
	refused := NewScopeResult().Refused(&Refusal{Code: CodeKickoffNotSealed, Message: "sin kickoff", Continuation: "sella"})
	stdout.Reset()
	refused.WriteText(&stdout, &stderr)
	if !strings.Contains(stderr.String(), "kickoff_not_sealed") || stdout.Len() != 0 {
		t.Errorf("un rechazo va a stderr con su código: stdout %q, stderr %q", stdout.String(), stderr.String())
	}
}

// Cada token nuevo del contrato lleva su etiqueta en castellano.
func TestEveryScopeTokenHasASpanishLabel(t *testing.T) {
	for _, token := range []string{
		ScopeOutcomeOK, ScopeOutcomeEmpty, ScopeOutcomeUnverifiable, ScopeOutcomeExempt,
		ScopeReasonBranchMissing, ScopeReasonBaseMissing, ScopeReasonForkPointUndetermined, ScopeReasonNotMerged,
		ScopeOfferOfferable, ScopeOfferPendingMerge, ScopeOfferUnverifiable, ScopeOfferNothingToVerify,
		ScopeStartFirstParent, ScopeStartReflog,
		ScopeWarningForeignCommitsPossible, string(CodeScopeNotApplicable),
	} {
		if label, known := labels[token]; !known || label == "" || label == token {
			t.Errorf("el token %q no tiene etiqueta en castellano", token)
		}
	}
}

// La fuente de una referencia lleva su etiqueta en castellano aunque su token tenga guion.
func TestScopeRefLabelsItsSource(t *testing.T) {
	for source, want := range map[string]string{ScopeSourceLocal: "rama local", ScopeSourceRemoteTracking: "referencia remota ya descargada"} {
		if got := NewScopeRef("feat/x", "refs/x", source, "abc"); got.SourceLabel != want {
			t.Errorf("SourceLabel de %q = %q, se esperaba %q", source, got.SourceLabel, want)
		}
	}
}
