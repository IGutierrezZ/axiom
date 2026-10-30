package testenv

import (
	"runtime"
	"strings"
	"testing"
)

// fakeTB registra la omisión sin detener la prueba real. Skip imita a
// testing.T terminando la goroutine con runtime.Goexit.
type fakeTB struct {
	testing.TB
	skipped bool
	message string
}

func (f *fakeTB) Helper() {}

func (f *fakeTB) Skip(args ...any) {
	f.skipped = true
	for _, arg := range args {
		if s, ok := arg.(string); ok {
			f.message += s
		}
	}
	runtime.Goexit()
}

// runRequire ejecuta RequireBinaryBuilds en su propia goroutine y devuelve el
// fakeTB una vez terminada, haya omitido o no.
func runRequire() *fakeTB {
	fake := &fakeTB{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		RequireBinaryBuilds(fake)
	}()
	<-done
	return fake
}

func TestBinaryBuildsEnabled(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{" 1 ", true},
		{"", false},
		{"0", false},
		{"false", false},
		{"yes", false},
		{"2", false},
	}
	for _, tt := range tests {
		if got := binaryBuildsEnabled(tt.raw); got != tt.want {
			t.Errorf("binaryBuildsEnabled(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestRequireBinaryBuildsSkipsWhenUnset(t *testing.T) {
	t.Setenv(BuildBinariesEnv, "")
	fake := runRequire()
	if !fake.skipped {
		t.Fatal("RequireBinaryBuilds did not skip with the variable unset")
	}
	if !strings.Contains(fake.message, BuildBinariesEnv+"=1") {
		t.Fatalf("skip message %q does not name %s=1", fake.message, BuildBinariesEnv)
	}
}

func TestRequireBinaryBuildsSkipsWhenNotEnabled(t *testing.T) {
	t.Setenv(BuildBinariesEnv, "0")
	if !runRequire().skipped {
		t.Fatal("RequireBinaryBuilds did not skip with the variable set to 0")
	}
}

func TestRequireBinaryBuildsRunsWhenEnabled(t *testing.T) {
	for _, raw := range []string{"1", "true"} {
		t.Setenv(BuildBinariesEnv, raw)
		if runRequire().skipped {
			t.Fatalf("RequireBinaryBuilds skipped with %s=%s", BuildBinariesEnv, raw)
		}
	}
}

func TestRequireBinaryBuildsIgnoresCI(t *testing.T) {
	t.Setenv(BuildBinariesEnv, "")
	t.Setenv("CI", "true")
	if !runRequire().skipped {
		t.Fatal("RequireBinaryBuilds was enabled implicitly by CI")
	}
}
