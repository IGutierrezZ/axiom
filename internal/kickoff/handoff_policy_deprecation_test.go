package kickoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// REQ-26.9: handoff_policy is deprecated. The flag is accepted, validated and
// ignored; a freshly sealed kickoff always carries the compatibility value
// "none" so that older binaries (which reject a kickoff without the key) keep
// reading it.

func sealedKickoffFromArgs(t *testing.T, args []string) (Kickoff, string) {
	t.Helper()
	parsed, err := ParseSealArgs(args)
	if err != nil {
		t.Fatalf("ParseSealArgs(%v) error = %v", args, err)
	}
	changeRoot := t.TempDir()
	if _, ok, err := Seal(changeRoot, parsed.ToKickoff("inc-99-example")); err != nil || !ok {
		t.Fatalf("Seal() = (_, %v, %v)", ok, err)
	}
	onDisk, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil {
		t.Fatalf("leer kickoff.yaml: %v", err)
	}
	k, err := ParseKickoff(onDisk)
	if err != nil {
		t.Fatalf("ParseKickoff(sellado) error = %v", err)
	}
	return k, string(onDisk)
}

func TestSealWithoutHandoffPolicyWritesNone(t *testing.T) {
	k, raw := sealedKickoffFromArgs(t, []string{
		"--cwd", "/repo", "--change", "inc-99-example", "--execution-style", "checkpointed",
	})
	if k.Config.HandoffPolicy != HandoffNone {
		t.Fatalf("HandoffPolicy = %q, se esperaba none", k.Config.HandoffPolicy)
	}
	if !strings.Contains(raw, "handoff_policy: none") {
		t.Fatalf("kickoff.yaml no contiene handoff_policy: none:\n%s", raw)
	}
}

func TestSealIgnoresDeprecatedHandoffPolicyFlag(t *testing.T) {
	for _, value := range []string{"none", "per_checkpoint"} {
		t.Run(value, func(t *testing.T) {
			k, raw := sealedKickoffFromArgs(t, []string{
				"--cwd", "/repo", "--change", "inc-99-example", "--execution-style", "checkpointed",
				"--handoff-policy", value,
			})
			if k.Config.HandoffPolicy != HandoffNone {
				t.Fatalf("HandoffPolicy = %q, se esperaba none: el flag no debe llegar a kickoff.yaml", k.Config.HandoffPolicy)
			}
			if strings.Contains(raw, "per_checkpoint") {
				t.Fatalf("kickoff.yaml conserva el valor del flag deprecado:\n%s", raw)
			}
		})
	}
}

func TestSealRejectsUnknownHandoffPolicyValue(t *testing.T) {
	_, err := ParseSealArgs([]string{
		"--cwd", "/repo", "--change", "inc-99-example", "--execution-style", "checkpointed",
		"--handoff-policy", "otra",
	})
	if err == nil {
		t.Fatal("ParseSealArgs() = nil error, se esperaba rechazo de --handoff-policy otra (A-6)")
	}
	if !strings.Contains(err.Error(), "--handoff-policy desconocido") {
		t.Fatalf("error = %q, se esperaba el rechazo del valor desconocido", err.Error())
	}
}

func TestSealRequiresNoHandoffPolicy(t *testing.T) {
	_, err := ParseSealArgs([]string{
		"--cwd", "/repo", "--change", "inc-99-example", "--execution-style", "continuous",
	})
	if err != nil {
		t.Fatalf("ParseSealArgs() sin --handoff-policy error = %v; el flag ya no es obligatorio", err)
	}
}

func TestKickoffValidateAcceptsHandoffPolicyVocabularyAndEmpty(t *testing.T) {
	tests := []struct {
		name    string
		value   HandoffPolicy
		wantErr bool
	}{
		{name: "vacio", value: "", wantErr: false},
		{name: "none", value: HandoffNone, wantErr: false},
		{name: "per_checkpoint", value: HandoffPerCheckpoint, wantErr: false},
		{name: "otra", value: HandoffPolicy("otra"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := sampleKickoff("inc-99-example")
			k.Config.HandoffPolicy = tt.value
			err := k.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseKickoffReadsLegacyPerCheckpointAndMissingKey(t *testing.T) {
	legacy, err := ParseKickoff([]byte(validKickoffYAML()))
	if err != nil {
		t.Fatalf("ParseKickoff(per_checkpoint antiguo) error = %v", err)
	}
	if legacy.Config.HandoffPolicy != HandoffPerCheckpoint {
		t.Fatalf("HandoffPolicy = %q, se esperaba per_checkpoint leido tal cual", legacy.Config.HandoffPolicy)
	}

	withoutKey := strings.Replace(validKickoffYAML(), "  handoff_policy: per_checkpoint\n", "", 1)
	if strings.Contains(withoutKey, "handoff_policy") {
		t.Fatal("la fixture sin clave sigue conteniendo handoff_policy")
	}
	k, err := ParseKickoff([]byte(withoutKey))
	if err != nil {
		t.Fatalf("ParseKickoff(sin clave) error = %v; el valor vacio es valido", err)
	}
	if k.Config.HandoffPolicy != "" {
		t.Fatalf("HandoffPolicy = %q, se esperaba vacio", k.Config.HandoffPolicy)
	}
}

func TestInferredKickoffWritesNoneWithoutInferringPolicy(t *testing.T) {
	changeRoot := t.TempDir()
	designPath := writeDesignMD(t, changeRoot, "# Diseno\n")
	k, _, err := InferKickoff(designPath, repoLikeWorkspaceConfig())
	if err != nil {
		t.Fatalf("InferKickoff() error = %v", err)
	}
	if k.Config.HandoffPolicy != HandoffNone {
		t.Fatalf("HandoffPolicy = %q, se esperaba none (valor de compatibilidad)", k.Config.HandoffPolicy)
	}
}
