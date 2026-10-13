package testenv

import (
	"strings"
	"testing"
	"time"
)

func TestParseTimeoutScale(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		goos    string
		want    float64
		wantErr bool
	}{
		{"unset on windows", "", "windows", 2, false},
		{"unset on linux", "", "linux", 1, false},
		{"blank on windows", "  ", "windows", 2, false},
		{"explicit overrides the default", "3.5", "windows", 3.5, false},
		{"explicit 1 on windows disables the widening", "1", "windows", 1, false},
		{"explicit on linux", "4", "linux", 4, false},
		{"below one is rejected", "0.5", "linux", 0, true},
		{"zero is rejected", "0", "linux", 0, true},
		{"negative is rejected", "-2", "windows", 0, true},
		{"not a number is rejected", "fast", "linux", 0, true},
		{"NaN is rejected", "NaN", "linux", 0, true},
		{"Inf is rejected", "Inf", "linux", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimeoutScale(tt.raw, tt.goos)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTimeoutScale(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), TimeoutScaleEnv) {
				t.Fatalf("error %q does not name %s", err, TimeoutScaleEnv)
			}
			if got != tt.want {
				t.Fatalf("ParseTimeoutScale(%q, %q) = %v, want %v", tt.raw, tt.goos, got, tt.want)
			}
		})
	}
}

func TestScaledReadsTheEnvironment(t *testing.T) {
	t.Setenv(TimeoutScaleEnv, "2.5")
	got, err := Scaled(4 * time.Second)
	if err != nil {
		t.Fatalf("Scaled() error = %v", err)
	}
	if want := 10 * time.Second; got != want {
		t.Fatalf("Scaled(4s) = %v, want %v", got, want)
	}
	if got := MustScaled(time.Second); got != 2500*time.Millisecond {
		t.Fatalf("MustScaled(1s) = %v, want 2.5s", got)
	}
}

func TestScaledRejectsAnInvalidScaleInsteadOfShrinking(t *testing.T) {
	t.Setenv(TimeoutScaleEnv, "0.1")
	if _, err := Scaled(time.Minute); err == nil {
		t.Fatal("Scaled accepted a scale below 1")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustScaled did not abort on an invalid scale")
		}
	}()
	MustScaled(time.Minute)
}

func TestScaledNeverReturnsLessThanTheBudget(t *testing.T) {
	for _, raw := range []string{"", "1", "1.0000001", "2", "10"} {
		t.Setenv(TimeoutScaleEnv, raw)
		got, err := Scaled(90 * time.Second)
		if err != nil {
			t.Fatalf("Scaled with %q: %v", raw, err)
		}
		if got < 90*time.Second {
			t.Fatalf("Scaled with %q = %v, shrank the budget", raw, got)
		}
	}
}
