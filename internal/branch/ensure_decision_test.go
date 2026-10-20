package branch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos y decisión del modo de trabajo de `ensure` (REQ-25.8, design §4.3): la decisión
// tipada de `ask` y los rechazos del modo, con la continuación como orden ejecutable.

const modeTestCWD = `C:\repos\mi proyecto`

func TestDecisionRequiredAskCarriesTheQuestionAndBothChoices(t *testing.T) {
	refusal := DecisionRequired("ensure", "x", modeTestCWD, LaneSDD)

	if refusal.Code != CodeDecisionRequired || refusal.Message == "" || refusal.Continuation == "" {
		t.Fatalf("rechazo = %+v, se esperaba decision_required con mensaje y continuación", refusal)
	}
	decision := refusal.Decision
	if decision == nil || !strings.Contains(decision.Question, "cambio x") || len(decision.Choices) != 2 {
		t.Fatalf("decisión = %+v, se esperaba la pregunta del cambio x con dos opciones", decision)
	}
	for i, mode := range []Mode{ModeWorktree, ModeInPlace} {
		choice := decision.Choices[i]
		want := "axiom branch ensure --change x --mode " + string(mode) + " --cwd " + pathquote.Quote(modeTestCWD)
		if choice.Answer != string(mode) || choice.Label != Label(string(mode)) || choice.Effect == "" || choice.Invocation != want {
			t.Errorf("opción %d = %+v, se esperaba la respuesta %s, su etiqueta, un efecto y la invocación %q", i, choice, mode, want)
		}
		if !strings.Contains(refusal.Continuation, choice.Invocation) {
			t.Errorf("la continuación %q debe contener la invocación %q", refusal.Continuation, choice.Invocation)
		}
	}
}

func TestDecisionRequiredAskKeepsTheLaneInTheInvocations(t *testing.T) {
	for _, choice := range DecisionRequired("ensure", "mi-feature", modeTestCWD, LaneODD).Decision.Choices {
		if !strings.Contains(choice.Invocation, "--change mi-feature --lane odd --mode "+choice.Answer) {
			t.Errorf("la invocación %q debe repetir --lane odd", choice.Invocation)
		}
	}
}

// El resultado de un `decision_required` lleva su estado propio, el código, la decisión y,
// como cualquier resultado que no es «listo», sale con 1 y escribe por stderr.
func TestResultRefusedWithADecisionRequiredCarriesTheDecision(t *testing.T) {
	refusal := DecisionRequired("ensure", "x", modeTestCWD, LaneSDD)

	result := NewResult("ensure").Refused(refusal)

	if result.Status != StatusDecisionRequired || result.StatusLabel != "decisión requerida" || result.Code != CodeDecisionRequired || result.ExitCode() != ExitRefused {
		t.Errorf("estado %q (%q), código %q, salida %d", result.Status, result.StatusLabel, result.Code, result.ExitCode())
	}
	if result.Decision != refusal.Decision {
		t.Errorf("el resultado debe llevar la decisión del rechazo")
	}

	var out bytes.Buffer
	if err := result.WriteJSON(&out); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if _, present := decoded["decision"]; !present {
		t.Errorf("el JSON debe llevar `decision`:\n%s", out.String())
	}

	var stdout, stderr bytes.Buffer
	result.WriteText(&stdout, &stderr)
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "decision_required") || !strings.Contains(stderr.String(), "--mode worktree") || !strings.Contains(stderr.String(), "--mode in_place") {
		t.Errorf("el texto va a stderr con el código y las dos órdenes:\nstdout %q\nstderr %q", stdout.String(), stderr.String())
	}
}

// Un resultado sin decisión no lleva la clave: los demás resultados no cambian.
func TestResultWithoutADecisionDoesNotCarryTheKey(t *testing.T) {
	var out bytes.Buffer
	if err := NewResult("ensure").WriteJSON(&out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "decision") {
		t.Errorf("un resultado sin decisión no lleva `decision`:\n%s", out.String())
	}
}

func TestModeRefusalsNameAnExecutableContinuation(t *testing.T) {
	tests := []struct {
		name         string
		refusal      *Refusal
		wantCode     Code
		wantMessage  []string
		wantContinue string
	}{
		{
			name:         "un modo distinto del materializado",
			refusal:      ModeConflict("ensure", "x", modeTestCWD, LaneSDD, ModeWorktree, ModeInPlace),
			wantCode:     CodePlanConflict,
			wantMessage:  []string{"cambio x", "worktree", "in_place"},
			wantContinue: "axiom branch ensure --change x --mode worktree --cwd " + pathquote.Quote(modeTestCWD),
		},
		{
			name:         "un modo que no es una respuesta",
			refusal:      InvalidMode("ensure", "x", modeTestCWD, LaneSDD, ModeAsk),
			wantCode:     CodeConfigInvalid,
			wantMessage:  []string{"ask", "worktree", "in_place"},
			wantContinue: "axiom branch ensure --change x --mode worktree --cwd " + pathquote.Quote(modeTestCWD),
		},
		{
			name:         "in_place todavía no disponible",
			refusal:      InPlaceNotAvailable("ensure", "x", modeTestCWD, LaneSDD),
			wantCode:     CodePlanConflict,
			wantMessage:  []string{"in_place"},
			wantContinue: "axiom branch ensure --change x --mode worktree --cwd " + pathquote.Quote(modeTestCWD),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.refusal.Code != tc.wantCode {
				t.Errorf("código = %q, se esperaba %q", tc.refusal.Code, tc.wantCode)
			}
			for _, want := range tc.wantMessage {
				if !strings.Contains(tc.refusal.Message, want) {
					t.Errorf("el mensaje %q debe contener %q", tc.refusal.Message, want)
				}
			}
			if !strings.Contains(tc.refusal.Continuation, tc.wantContinue) {
				t.Errorf("la continuación %q debe contener la orden %q", tc.refusal.Continuation, tc.wantContinue)
			}
			if tc.refusal.Decision != nil {
				t.Errorf("solo decision_required lleva decisión: %+v", tc.refusal.Decision)
			}
		})
	}
}
