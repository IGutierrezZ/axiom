package kickoff

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// TestParseSealArgsKnownAndUnknownFlags is the base RED case for 8.1: a
// fully-formed, valid `kickoff seal` invocation must parse without error,
// and a single unrecognised flag must be rejected — mirroring
// sddstatus.ParseCommandArgs' own "unknown argument" behaviour.
func TestParseSealArgsKnownAndUnknownFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "flags conocidas completas",
			args: []string{
				"--cwd", "/repo", "--change", "inc-99-example",
				"--flow-mode", "sdd", "--execution-style", "checkpointed",
				"--handoff-policy", "per_checkpoint",
				"--role", "core:blocking", "--role", "qa:deferred",
				"--deployment-target", "staging", "--json",
			},
			wantErr: false,
		},
		{
			name:    "bandera desconocida",
			args:    []string{"--change", "inc-99-example", "--execution-style", "continuous", "--handoff-policy", "none", "--bogus", "x"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSealArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSealArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseSealArgsRequiresExecutionStyleOrSessionPace covers REQ-21.5's
// second scenario: an ambiguous or absent modality must never fall back to
// a silent default.
func TestParseSealArgsRequiresExecutionStyleOrSessionPace(t *testing.T) {
	_, err := ParseSealArgs([]string{"--cwd", "/repo", "--change", "inc-99-example", "--handoff-policy", "none"})
	if err == nil {
		t.Fatal("ParseSealArgs() = nil error, se esperaba rechazo sin --execution-style ni --from-session-pace")
	}
}

// TestParseSealArgsExecutionStyleFromSessionPace is D-03's derivation
// table: interactive -> checkpointed, auto -> continuous.
func TestParseSealArgsExecutionStyleFromSessionPace(t *testing.T) {
	tests := []struct {
		pace string
		want ExecutionStyle
	}{
		{pace: "interactive", want: ExecutionCheckpointed},
		{pace: "auto", want: ExecutionContinuous},
	}
	for _, tt := range tests {
		t.Run(tt.pace, func(t *testing.T) {
			parsed, err := ParseSealArgs([]string{
				"--change", "inc-99-example", "--from-session-pace", tt.pace, "--handoff-policy", "none",
			})
			if err != nil {
				t.Fatalf("ParseSealArgs() error = %v", err)
			}
			if parsed.ExecutionStyle != tt.want {
				t.Fatalf("ExecutionStyle = %q, want %q", parsed.ExecutionStyle, tt.want)
			}
			if parsed.ExecutionStyleSource != "session_pace" {
				t.Fatalf("ExecutionStyleSource = %q, want %q", parsed.ExecutionStyleSource, "session_pace")
			}
			if parsed.ExecutionStyleOverride {
				t.Fatal("ExecutionStyleOverride = true sin --execution-style explicito")
			}
		})
	}
}

// TestParseSealArgsExecutionStyleExplicitOverridesSessionPace is D-03's
// explicit override rule: when both flags are given, --execution-style
// wins, and the override is recorded as deliberate.
func TestParseSealArgsExecutionStyleExplicitOverridesSessionPace(t *testing.T) {
	parsed, err := ParseSealArgs([]string{
		"--change", "inc-99-example",
		"--from-session-pace", "auto",
		"--execution-style", "checkpointed",
		"--handoff-policy", "none",
	})
	if err != nil {
		t.Fatalf("ParseSealArgs() error = %v", err)
	}
	if parsed.ExecutionStyle != ExecutionCheckpointed {
		t.Fatalf("ExecutionStyle = %q, want %q (el explicito debe ganar sobre session-pace=auto)", parsed.ExecutionStyle, ExecutionCheckpointed)
	}
	if parsed.ExecutionStyleSource != "explicit" {
		t.Fatalf("ExecutionStyleSource = %q, want %q", parsed.ExecutionStyleSource, "explicit")
	}
	if !parsed.ExecutionStyleOverride {
		t.Fatal("ExecutionStyleOverride = false, se esperaba true: sobrescritura deliberada de --from-session-pace (D-03)")
	}
}

// TestParseSealArgsDefaultsRoleToFullstackBlocking covers REQ-21.6: no
// --role flags declared seals exactly one role, fullstack:blocking.
func TestParseSealArgsDefaultsRoleToFullstackBlocking(t *testing.T) {
	parsed, err := ParseSealArgs([]string{"--change", "inc-99-example", "--execution-style", "continuous", "--handoff-policy", "none"})
	if err != nil {
		t.Fatalf("ParseSealArgs() error = %v", err)
	}
	if len(parsed.Roles) != 1 || parsed.Roles[0].Role != "fullstack" || parsed.Roles[0].GatePolicy != multirole.PolicyBlocking {
		t.Fatalf("Roles = %+v, se esperaba exactamente [{fullstack blocking}]", parsed.Roles)
	}
}

// TestParseSealArgsRoleFlagPolicyParsing covers --role <id>[:policy]
// parsing: a bare id defaults to blocking, an explicit policy is honoured,
// and an unknown policy is rejected.
func TestParseSealArgsRoleFlagPolicyParsing(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantRole   string
		wantPolicy multirole.GatePolicy
		wantErr    bool
	}{
		{name: "solo id, blocking por defecto", value: "core", wantRole: "core", wantPolicy: multirole.PolicyBlocking},
		{name: "id con politica deferred", value: "qa:deferred", wantRole: "qa", wantPolicy: multirole.PolicyDeferred},
		{name: "id con politica optional", value: "docs:optional", wantRole: "docs", wantPolicy: multirole.PolicyOptional},
		{name: "politica desconocida", value: "core:bogus", wantErr: true},
		{name: "sin identificador de rol", value: ":blocking", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSealArgs([]string{
				"--change", "inc-99-example", "--execution-style", "continuous", "--handoff-policy", "none", "--role", tt.value,
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSealArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(parsed.Roles) != 1 || parsed.Roles[0].Role != tt.wantRole || parsed.Roles[0].GatePolicy != tt.wantPolicy {
				t.Fatalf("Roles = %+v, want [{%s %s}]", parsed.Roles, tt.wantRole, tt.wantPolicy)
			}
		})
	}
}

// TestParseSealArgsInferSkipsRequiredFlags covers the --infer branch: none
// of execution-style/handoff-policy/role is required, since InferKickoff
// derives them all.
func TestParseSealArgsInferSkipsRequiredFlags(t *testing.T) {
	parsed, err := ParseSealArgs([]string{"--change", "inc-99-example", "--infer"})
	if err != nil {
		t.Fatalf("ParseSealArgs() error = %v, se esperaba exito con --infer sin las demas banderas", err)
	}
	if !parsed.Infer {
		t.Fatal("Infer = false, se esperaba true")
	}
}

// TestParseSealArgsCWDAcceptsAnyShape confirms --cwd is captured verbatim
// regardless of shape (relative, absolute, or a path that does not exist):
// containment and existence for --cwd are enforced later, by the CLI
// adapter, before any write — never inside this pure parser (T-2).
func TestParseSealArgsCWDAcceptsAnyShape(t *testing.T) {
	tests := []string{"relative/path", "/absolute/path", "/does/not/exist/anywhere"}
	for _, cwd := range tests {
		t.Run(cwd, func(t *testing.T) {
			parsed, err := ParseSealArgs([]string{
				"--cwd", cwd, "--change", "inc-99-example", "--execution-style", "continuous", "--handoff-policy", "none",
			})
			if err != nil {
				t.Fatalf("ParseSealArgs() error = %v", err)
			}
			if parsed.CWD != cwd {
				t.Fatalf("CWD = %q, want %q", parsed.CWD, cwd)
			}
		})
	}
}

// TestChangeNameContainmentRejectsAllTenVectors is T-2/T-7's full table
// against the single shared validator every parser in this file reuses
// (task 8.3): none of these strings may ever reach filepath.Join. Renamed
// from ...AllEightVectors when an independent validator found two more
// adversarial vectors this table did not yet cover (remediation, post
// Phase 8): a bare Windows drive-relative segment, and a reserved device
// name whose trailing space defeats the first-dot truncation the reserved
// check used to rely on. Neither escaped openspec/changes/ on its own —
// this is containment hardening, not a live break — but Phase 8 owns this
// validator, so the fix and its regression coverage land here.
func TestChangeNameContainmentRejectsAllTenVectors(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "traversal ..", value: ".."},
		{name: "separador /", value: "foo/bar"},
		{name: "separador \\", value: `foo\bar`},
		{name: "ruta absoluta posix", value: "/etc/passwd"},
		{name: "nombre reservado windows con", value: "con"},
		{name: "nombre reservado windows nul", value: "nul"},
		{name: "cadena vacia", value: ""},
		{name: "cadena de 300 caracteres", value: strings.Repeat("a", 300)},
		{name: "segmento relativo de unidad windows", value: "C:foo"},
		{name: "nombre reservado windows con espacio final sin extension", value: "con "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateChangeName(tt.value); err == nil {
				t.Fatalf("validateChangeName(%q) = nil, se esperaba rechazo por contencion", tt.value)
			}
		})
	}
}

// TestChangeNameContainmentAcceptsOrdinaryNames is the mirror case: a
// legitimate bare change directory name must never be rejected.
func TestChangeNameContainmentAcceptsOrdinaryNames(t *testing.T) {
	for _, value := range []string{"inc-21-upfront-flow-governance", "add-widget-tagging", "x"} {
		if err := validateChangeName(value); err != nil {
			t.Fatalf("validateChangeName(%q) error = %v, se esperaba aceptacion", value, err)
		}
	}
}

// TestParseSealArgsRejectsChangeContainmentVector confirms the shared
// validator is actually wired into ParseSealArgs, not just defined.
func TestParseSealArgsRejectsChangeContainmentVector(t *testing.T) {
	_, err := ParseSealArgs([]string{"--change", "../escape", "--execution-style", "continuous", "--handoff-policy", "none"})
	if err == nil {
		t.Fatal("ParseSealArgs() = nil error, se esperaba rechazo por contencion de --change")
	}
}

// TestParseGateRecordArgsKnownAndUnknownFlags mirrors the seal-args base
// case for `gate record`.
func TestParseGateRecordArgsKnownAndUnknownFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "flags conocidas completas",
			args: []string{
				"--cwd", "/repo", "--change", "inc-99-example",
				"--gate", "spec", "--decision", "approved", "--reason", "cubre el caso borde",
				"--actor", "maintainer", "--json",
			},
			wantErr: false,
		},
		{
			name:    "bandera desconocida",
			args:    []string{"--change", "inc-99-example", "--gate", "spec", "--decision", "approved", "--bogus", "x"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGateRecordArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGateRecordArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseGateRecordArgsRejectedRequiresReason is REQ-21.12's hard rule:
// a rejection without a recorded reason is refused.
func TestParseGateRecordArgsRejectedRequiresReason(t *testing.T) {
	_, err := ParseGateRecordArgs([]string{"--change", "inc-99-example", "--gate", "spec", "--decision", "rejected"})
	if err == nil {
		t.Fatal("ParseGateRecordArgs() = nil error, se esperaba rechazo: --decision rejected exige --reason")
	}
}

// TestParseGateRecordArgsApprovedDoesNotRequireReason confirms the reason
// requirement is specific to rejected, matching design.md S5.7's CLI
// contract table (it names only the rejected-without-reason row).
func TestParseGateRecordArgsApprovedDoesNotRequireReason(t *testing.T) {
	_, err := ParseGateRecordArgs([]string{"--change", "inc-99-example", "--gate", "spec", "--decision", "approved"})
	if err != nil {
		t.Fatalf("ParseGateRecordArgs() error = %v, se esperaba exito: approved no exige --reason", err)
	}
}

// TestParseGateRecordArgsHandOff fija la validación de forma de --hand-off (REQ-26.1, REQ-26.3,
// DD-01): solo acompaña a approved, no con integration ni con evidencia, y no exige --reason. Los
// registros sin la bandera se siguen leyendo con HandOff == false.
func TestParseGateRecordArgsHandOff(t *testing.T) {
	base := []string{"--change", "inc-99-example"}
	with := func(extra ...string) []string { return append(append([]string{}, base...), extra...) }

	t.Run("approved con --hand-off sin motivo", func(t *testing.T) {
		parsed, err := ParseGateRecordArgs(with("--gate", "spec", "--decision", "approved", "--hand-off"))
		if err != nil {
			t.Fatalf("ParseGateRecordArgs() error = %v", err)
		}
		if !parsed.HandOff || parsed.Decision != DecisionApproved || parsed.Gate != GateSpec {
			t.Fatalf("parsed = %+v, se esperaba HandOff con approved sobre spec", parsed)
		}
	})
	t.Run("approved con --hand-off y motivo en un role-apply", func(t *testing.T) {
		parsed, err := ParseGateRecordArgs(with("--gate", "role-apply:core", "--decision", "approved", "--hand-off", "--reason", "listo"))
		if err != nil || !parsed.HandOff || parsed.Reason != "listo" {
			t.Fatalf("parsed = %+v, error = %v", parsed, err)
		}
	})
	t.Run("sin la bandera no hay cesion", func(t *testing.T) {
		for _, args := range [][]string{
			with("--gate", "spec", "--decision", "approved"),
			with("--gate", "spec", "--decision", "rejected", "--reason", "x"),
		} {
			parsed, err := ParseGateRecordArgs(args)
			if err != nil || parsed.HandOff {
				t.Errorf("args %v: parsed = %+v, error = %v; se esperaba HandOff == false", args, parsed, err)
			}
		}
	})

	for name, tc := range map[string]struct {
		args []string
		want []string
	}{
		"con rejected": {
			args: with("--gate", "spec", "--decision", "rejected", "--reason", "x", "--hand-off"),
			want: []string{"solo acompaña a `--decision approved`", "--decision rejected --reason"},
		},
		"con integration": {
			args: with("--gate", "integration", "--decision", "approved", "--hand-off"),
			want: []string{"no está disponible para la compuerta integration", "--evidence-kind"},
		},
		"con evidencia": {
			args: with("--gate", "spec", "--decision", "approved", "--evidence-kind", "attestation", "--evidence", "x", "--hand-off"),
			want: []string{"no admite `--evidence-kind`", "--decision approved --hand-off"},
		},
	} {
		t.Run("rechaza "+name, func(t *testing.T) {
			_, err := ParseGateRecordArgs(tc.args)
			if err == nil {
				t.Fatal("ParseGateRecordArgs() = nil, se esperaba un error de uso")
			}
			for _, fragment := range tc.want {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("el error %q debe contener %q", err, fragment)
				}
			}
			if !strings.Contains(err.Error(), "axiom sdd gate record") {
				t.Errorf("el error %q debe llevar una continuación ejecutable", err)
			}
		})
	}
}

// TestParseGateRecordArgsLeavesBaseRefToTheCaller fija D-22 (REQ-25.23, escenario 5): el parseo
// de argumentos no conoce la rama de integración del proyecto, así que ya no inventa `main` ante
// un --base-ref vacío; lo completa quien conoce el repositorio (runSDDGateRecord). Un --base-ref
// explícito se conserva tal cual.
func TestParseGateRecordArgsLeavesBaseRefToTheCaller(t *testing.T) {
	base := []string{"--change", "inc-99-example", "--gate", "integration", "--decision", "approved",
		"--evidence-kind", "pr_merged", "--commit", "abc123"}
	tests := []struct {
		name  string
		extra []string
		want  string
	}{
		{name: "sin --base-ref queda vacío", want: ""},
		{name: "--base-ref explícito se conserva", extra: []string{"--base-ref", "develop"}, want: "develop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseGateRecordArgs(append(append([]string{}, base...), tt.extra...))
			if err != nil {
				t.Fatalf("ParseGateRecordArgs() error = %v", err)
			}
			if parsed.BaseRef != tt.want {
				t.Fatalf("BaseRef = %q, se esperaba %q", parsed.BaseRef, tt.want)
			}
		})
	}
}

// TestParseGateRecordArgsGateVocabulary covers the closed --gate vocabulary:
// the four fixed keys, a well-formed role-apply:<rol>, an empty role name
// after the role-apply: prefix, and a totally unknown key.
func TestParseGateRecordArgsGateVocabulary(t *testing.T) {
	tests := []struct {
		name    string
		gate    string
		wantErr bool
	}{
		{name: "spec", gate: "spec"},
		{name: "design", gate: "design"},
		{name: "tasks", gate: "tasks"},
		{name: "integration", gate: "integration"},
		{name: "role-apply con rol", gate: "role-apply:core"},
		{name: "role-apply sin nombre de rol", gate: "role-apply:", wantErr: true},
		{name: "clave desconocida", gate: "bogus", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGateRecordArgs([]string{"--change", "inc-99-example", "--gate", tt.gate, "--decision", "approved"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGateRecordArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseGateRecordArgsDecisionVocabulary covers the closed --decision
// vocabulary.
func TestParseGateRecordArgsDecisionVocabulary(t *testing.T) {
	tests := []struct {
		name     string
		decision string
		wantErr  bool
	}{
		{name: "approved", decision: "approved"},
		{name: "rejected con reason", decision: "rejected"},
		{name: "desconocida", decision: "bogus", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"--change", "inc-99-example", "--gate", "spec", "--decision", tt.decision}
			if tt.decision == "rejected" {
				args = append(args, "--reason", "motivo de prueba")
			}
			_, err := ParseGateRecordArgs(args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGateRecordArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseGateRecordArgsRejectsChangeContainmentVector confirms the shared
// validator is wired into ParseGateRecordArgs too.
func TestParseGateRecordArgsRejectsChangeContainmentVector(t *testing.T) {
	_, err := ParseGateRecordArgs([]string{"--change", "con", "--gate", "spec", "--decision", "approved"})
	if err == nil {
		t.Fatal("ParseGateRecordArgs() = nil error, se esperaba rechazo por contencion de --change")
	}
}

// TestParseShowArgsKnownAndUnknownFlags covers the shared show parser used
// by both `kickoff show` and `gate show`.
func TestParseShowArgsKnownAndUnknownFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "flags conocidas", args: []string{"--cwd", "/repo", "--change", "inc-99-example", "--json"}},
		{name: "sin --change", args: []string{"--cwd", "/repo"}, wantErr: true},
		{name: "bandera desconocida", args: []string{"--change", "inc-99-example", "--bogus"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseShowArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseShowArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseShowArgsRejectsChangeContainmentVector confirms the shared
// validator is wired into ParseShowArgs too (task 8.3: all three parsers
// share one containment check).
func TestParseShowArgsRejectsChangeContainmentVector(t *testing.T) {
	_, err := ParseShowArgs([]string{"--change", "nul"})
	if err == nil {
		t.Fatal("ParseShowArgs() = nil error, se esperaba rechazo por contencion de --change")
	}
}

// TestParseSealArgsDocsPublication cubre REQ-27.4: --docs-publication solo admite `auto`, solo
// acompaña a continuous (también el derivado de --from-session-pace) y nunca a --infer.
func TestParseSealArgsDocsPublication(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    DocsPublication
		wantErr string
	}{
		{name: "continuous con auto", args: []string{"--change", "x", "--execution-style", "continuous", "--docs-publication", "auto"}, want: DocsPublicationAuto},
		{name: "pace auto con auto", args: []string{"--change", "x", "--from-session-pace", "auto", "--docs-publication", "auto"}, want: DocsPublicationAuto},
		{name: "sin el flag no se escribe", args: []string{"--change", "x", "--execution-style", "continuous"}, want: ""},
		{name: "checkpointed se rechaza", args: []string{"--change", "x", "--execution-style", "checkpointed", "--docs-publication", "auto"}, wantErr: "continuous"},
		{name: "pace interactive se rechaza", args: []string{"--change", "x", "--from-session-pace", "interactive", "--docs-publication", "auto"}, wantErr: "continuous"},
		{name: "infer se rechaza", args: []string{"--change", "x", "--infer", "--docs-publication", "auto"}, wantErr: "--infer"},
		{name: "otro valor se rechaza", args: []string{"--change", "x", "--execution-style", "continuous", "--docs-publication", "otro"}, wantErr: "desconocido"},
		{name: "valor ausente se rechaza", args: []string{"--change", "x", "--execution-style", "continuous", "--docs-publication"}, wantErr: "--docs-publication"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSealArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseSealArgs() error = %v, se esperaba que mencionara %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSealArgs() error = %v", err)
			}
			if parsed.DocsPublication != tt.want {
				t.Fatalf("DocsPublication = %q, want %q", parsed.DocsPublication, tt.want)
			}
			if got := parsed.ToKickoff("x").Config.DocsPublication; got != tt.want {
				t.Fatalf("ToKickoff().Config.DocsPublication = %q, want %q", got, tt.want)
			}
		})
	}
}
