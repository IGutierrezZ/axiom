package consentenvelope

import "testing"

func validCore() Core {
	return Core{
		Headline: "Something needs a human decision.",
		Reason:   "the subject is blocked until a human decides.",
		Value:    "Deciding here keeps authority with the human.",
		Evidence: []string{},
		Choices: []Choice{
			{Answer: "granted", Label: "Grant", Effect: "Grants once.", Invocation: "gentle-ai example --consent granted"},
			{Answer: "declined", Label: "Decline", Effect: "Stays blocked.", Invocation: "gentle-ai example --consent declined"},
		},
		OffPath: OffPath{Note: "The documented alternative.", Command: "gentle-ai example off"},
	}
}

// threeChoiceCore is a valid envelope declaring three tokens, the shape the
// planning-gate question uses.
func threeChoiceCore() Core {
	core := validCore()
	core.Choices = []Choice{
		{Answer: "rejected", Label: "Iterate", Effect: "Keeps the gate open.", Invocation: "example --decision rejected"},
		{Answer: "hand_off", Label: "Hand off", Effect: "Approves and hands off.", Invocation: "example --decision approved --hand-off"},
		{Answer: "approved", Label: "Continue", Effect: "Approves and continues.", Invocation: "example --decision approved"},
	}
	return core
}

func TestValidateCompletenessDeclaredAnswers(t *testing.T) {
	if err := threeChoiceCore().ValidateCompleteness("rejected", "hand_off", "approved"); err != nil {
		t.Fatalf("valid three-choice core: %v", err)
	}
	tests := []struct {
		name    string
		core    Core
		answers []string
	}{
		{name: "fewer choices than declared", core: validCore(), answers: []string{"granted", "declined", "maybe"}},
		{name: "more choices than declared", core: threeChoiceCore(), answers: []string{"rejected", "approved"}},
		{name: "declared order differs", core: threeChoiceCore(), answers: []string{"approved", "hand_off", "rejected"}},
		{name: "declared token differs", core: threeChoiceCore(), answers: []string{"rejected", "handoff", "approved"}},
		{name: "duplicate declared tokens", core: threeChoiceCore(), answers: []string{"rejected", "rejected", "approved"}},
		{name: "a single declared token", core: validCore(), answers: []string{"granted"}},
		{name: "no declared tokens", core: validCore(), answers: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.core.ValidateCompleteness(tt.answers...); err == nil {
				t.Fatal("ValidateCompleteness() accepted a mismatched answer list")
			}
		})
	}
	// The two-token consumers (edit authority, RDD) keep rejecting a third
	// choice: the arity lives in each caller's own list.
	if err := threeChoiceCore().ValidateCompleteness("granted", "declined"); err == nil {
		t.Fatal("a three-choice envelope validated against the grant/decline pair")
	}
}

func TestValidateCompleteness(t *testing.T) {
	if err := validCore().ValidateCompleteness("granted", "declined"); err != nil {
		t.Fatalf("valid core: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(core *Core)
	}{
		{name: "empty headline", mutate: func(core *Core) { core.Headline = "" }},
		{name: "empty reason", mutate: func(core *Core) { core.Reason = "" }},
		{name: "empty value", mutate: func(core *Core) { core.Value = "" }},
		{name: "nil evidence encodes as null", mutate: func(core *Core) { core.Evidence = nil }},
		{name: "one choice", mutate: func(core *Core) { core.Choices = core.Choices[:1] }},
		{name: "three choices", mutate: func(core *Core) { core.Choices = append(core.Choices, Choice{Answer: "maybe"}) }},
		{name: "swapped answer tokens", mutate: func(core *Core) {
			core.Choices[0].Answer, core.Choices[1].Answer = core.Choices[1].Answer, core.Choices[0].Answer
		}},
		{name: "empty label", mutate: func(core *Core) { core.Choices[0].Label = "" }},
		{name: "empty effect", mutate: func(core *Core) { core.Choices[1].Effect = "" }},
		{name: "empty invocation", mutate: func(core *Core) { core.Choices[0].Invocation = "" }},
		{name: "empty off-path note", mutate: func(core *Core) { core.OffPath.Note = "" }},
		{name: "empty off-path command", mutate: func(core *Core) { core.OffPath.Command = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core := validCore()
			tt.mutate(&core)
			if err := core.ValidateCompleteness("granted", "declined"); err == nil {
				t.Fatal("ValidateCompleteness() accepted an incomplete envelope")
			}
		})
	}
}
