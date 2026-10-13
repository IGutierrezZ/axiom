package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// multirepoMasterFixtures are the two shapes of a multirepo master folder the
// lifecycle must never turn into a repository: one marked by .axiom-workspace
// and one recognized only by the repositories it holds.
var multirepoMasterFixtures = []struct {
	name  string
	build func(t *testing.T, master string)
}{
	{
		name: "workspace marker",
		build: func(t *testing.T, master string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(master, ".axiom-workspace"), []byte("planning_home: .\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
	},
	{
		name: "child repositories",
		build: func(t *testing.T, master string) {
			t.Helper()
			runReviewCLIGit(t, master, "init", "-q", "api")
			if err := os.MkdirAll(filepath.Join(master, "web"), 0o755); err != nil {
				t.Fatal(err)
			}
			// A gitfile, as a worktree or submodule checkout leaves behind.
			if err := os.WriteFile(filepath.Join(master, "web", ".git"), []byte("gitdir: ../api/.git\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
	},
}

func TestReviewLifecycleRefusesMultirepoMasterWithoutInitializingGit(t *testing.T) {
	modes := []struct {
		name string
		home func(*testing.T) string
	}{
		{name: "receipt-driven development off", home: reviewModeHome},
		{name: "receipt-driven development on", home: reviewEnabledHome},
	}
	operations := []struct {
		name       string
		args       func(master string) []string
		negotiated bool
	}{
		{
			name: "negotiated status",
			args: func(master string) []string {
				return []string{"status", "--cwd", master, "--contract", ReviewIntegrationContractV2, "--next-transition"}
			},
			negotiated: true,
		},
		{
			name: "plain status",
			args: func(master string) []string { return []string{"status", "--cwd", master} },
		},
		{
			name: "direct start",
			args: func(master string) []string { return []string{"start", "--cwd", master} },
		},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			for _, fixture := range multirepoMasterFixtures {
				t.Run(fixture.name, func(t *testing.T) {
					for _, operation := range operations {
						t.Run(operation.name, func(t *testing.T) {
							mode.home(t)
							master := canonicalReviewCLITempDir(t)
							fixture.build(t, master)

							var output bytes.Buffer
							err := RunReview(operation.args(master), &output)
							if err == nil {
								t.Fatalf("review accepted a multirepo master folder:\n%s", output.String())
							}
							if _, statErr := os.Lstat(filepath.Join(master, ".git")); !errors.Is(statErr, os.ErrNotExist) {
								t.Fatalf("refused review left Git metadata in the master folder: %v", statErr)
							}

							if operation.negotiated {
								// A negotiated caller reads the typed failure envelope;
								// the detail travels in its scrubbed cause.
								failure := decodeReviewIntegrationFailure(t, output.Bytes())
								if failure.Code != reviewIntegrationInvalidRequestCode || failure.NextAction != "correct_request" ||
									failure.MutationOutcome != ReviewMutationNotStarted {
									t.Fatalf("negotiated failure = %#v, want a not_started invalid_request that asks to correct the request", failure)
								}
								for _, want := range []string{"multirepo workspace", "axiom review status --cwd <repo>"} {
									if !strings.Contains(failure.Cause, want) {
										t.Fatalf("negotiated failure cause = %q, want it to contain %q", failure.Cause, want)
									}
								}
								return
							}

							var preflight *reviewIntegrationPreflightError
							if !errors.As(err, &preflight) {
								t.Fatalf("refusal = %v (%T), want a typed preflight refusal rather than an unexpected fault", err, err)
							}
							if code := preflight.classification().Code; code != reviewIntegrationInvalidRequestCode {
								t.Fatalf("refusal code = %q, want %q", code, reviewIntegrationInvalidRequestCode)
							}
							for _, want := range []string{"multirepo workspace", "axiom review status --cwd <repo>"} {
								if !strings.Contains(err.Error(), want) {
									t.Fatalf("refusal = %q, want it to contain %q", err.Error(), want)
								}
							}
						})
					}
				})
			}
		})
	}
}

func TestReviewAssessNeverInitializesGit(t *testing.T) {
	fixtures := []struct {
		name  string
		build func(t *testing.T, dir string)
	}{
		{
			name: "plain unversioned directory",
			build: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "candidate.txt"), []byte("candidate\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{name: "multirepo master with marker", build: multirepoMasterFixtures[0].build},
		{name: "multirepo master with child repositories", build: multirepoMasterFixtures[1].build},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			reviewEnabledHome(t)
			dir := canonicalReviewCLITempDir(t)
			fixture.build(t, dir)

			var output bytes.Buffer
			err := RunReview([]string{"assess", "--cwd", dir, "--json"}, &output)
			if err == nil {
				t.Fatalf("review assess accepted a directory outside Git:\n%s", output.String())
			}
			if output.Len() != 0 {
				t.Fatalf("review assess printed output before refusing: %q", output.String())
			}
			if _, statErr := os.Lstat(filepath.Join(dir, ".git")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("review assess initialized Git: %v", statErr)
			}

			var preflight *reviewIntegrationPreflightError
			if !errors.As(err, &preflight) {
				t.Fatalf("assess refusal = %v (%T), want a typed preflight refusal", err, err)
			}
			if code := preflight.classification().Code; code != reviewIntegrationInvalidRequestCode {
				t.Fatalf("assess refusal code = %q, want %q", code, reviewIntegrationInvalidRequestCode)
			}
			for _, want := range []string{"never initializes Git", "axiom review assess --cwd <repo>"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("assess refusal = %q, want it to contain %q", err.Error(), want)
				}
			}
		})
	}
}
