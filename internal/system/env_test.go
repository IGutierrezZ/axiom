package system

import "testing"

func TestLookupEnv(t *testing.T) {
	const axiomKey = "AXIOM_TEST_VAR"
	const otherKey = "OTHER_TEST_VAR"

	t.Run("axiom priority when both set", func(t *testing.T) {
		t.Setenv(axiomKey, "val_axiom")
		t.Setenv(otherKey, "val_other")

		got, ok := LookupEnv(axiomKey, otherKey)
		if !ok || got != "val_axiom" {
			t.Errorf("LookupEnv() = (%q, %v), want (%q, true)", got, ok, "val_axiom")
		}
	})

	t.Run("second key when first unset", func(t *testing.T) {
		const unsetAxiom = "AXIOM_UNSET_KEY"
		const setKey = "OTHER_SET_KEY"
		t.Setenv(setKey, "val_other")

		got, ok := LookupEnv(unsetAxiom, setKey)
		if !ok || got != "val_other" {
			t.Errorf("LookupEnv() = (%q, %v), want (%q, true)", got, ok, "val_other")
		}
	})

	t.Run("returns false when both unset", func(t *testing.T) {
		const unset1 = "AXIOM_UNSET_1"
		const unset2 = "AXIOM_UNSET_2"

		got, ok := LookupEnv(unset1, unset2)
		if ok || got != "" {
			t.Errorf("LookupEnv() = (%q, %v), want (%q, false)", got, ok, "")
		}
	})

	t.Run("no fallback to GENTLE_AI_ counterpart", func(t *testing.T) {
		const unsetAxiom = "AXIOM_NO_FALLBACK"
		const setGentle = "GENTLE_AI_NO_FALLBACK"
		t.Setenv(setGentle, "legacy_value")

		got, ok := LookupEnv(unsetAxiom)
		if ok || got != "" {
			t.Errorf("LookupEnv(%s) = (%q, %v), want (%q, false) without fallback", unsetAxiom, got, ok, "")
		}
	})
}

func TestGetenv(t *testing.T) {
	const axiomKey = "AXIOM_GETENV_TEST"
	const otherKey = "OTHER_GETENV_TEST"

	t.Run("axiom takes precedence", func(t *testing.T) {
		t.Setenv(axiomKey, "1")
		t.Setenv(otherKey, "0")

		if got := Getenv(axiomKey, otherKey); got != "1" {
			t.Errorf("Getenv() = %q, want %q", got, "1")
		}
	})

	t.Run("evaluates second key when first empty", func(t *testing.T) {
		t.Setenv(axiomKey, "")
		t.Setenv(otherKey, "0")

		if got := Getenv(axiomKey, otherKey); got != "0" {
			t.Errorf("Getenv() = %q, want %q", got, "0")
		}
	})

	t.Run("returns empty when neither set", func(t *testing.T) {
		const k1 = "AXIOM_NONE_SET_1"
		const k2 = "AXIOM_NONE_SET_2"

		if got := Getenv(k1, k2); got != "" {
			t.Errorf("Getenv() = %q, want %q", got, "")
		}
	})

	t.Run("no automatic fallback to GENTLE_AI_ key", func(t *testing.T) {
		const unsetAxiom = "AXIOM_AUTO_FALLBACK_UNSET"
		const setGentle = "GENTLE_AI_AUTO_FALLBACK_UNSET"
		t.Setenv(setGentle, "fallback_value")

		if got := Getenv(unsetAxiom); got != "" {
			t.Errorf("Getenv(%s) = %q, want empty string (no fallback)", unsetAxiom, got)
		}
	})
}
