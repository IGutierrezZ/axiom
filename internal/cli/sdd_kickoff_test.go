package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

func mustMkdirAllCLI(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

// newGovernanceWorkspace builds a minimal workspace + active change
// directory under t.TempDir(), the fixture shape every kickoff/gate CLI
// test in this file and in sdd_gate_test.go shares.
func newGovernanceWorkspace(t *testing.T, change string) (root, changeRoot string) {
	t.Helper()
	root = t.TempDir()
	changeRoot = filepath.Join(root, "openspec", "changes", change)
	mustMkdirAllCLI(t, changeRoot)
	return root, changeRoot
}

// writeRepoLikeAxiomYAML mirrors this repository's own axiom.yaml shape
// closely enough for multirole.DetectRoles: roles core and e2e declared, no
// fullstack (H-4), matching internal/kickoff/infer_test.go's
// repoLikeWorkspaceConfig fixture.
func writeRepoLikeAxiomYAML(t *testing.T, root string) {
	t.Helper()
	writeCLIFixture(t, filepath.Join(root, "axiom.yaml"), "workspace:\n"+
		"  name: test-workspace\n"+
		"  topology: monorepo-embedded\n"+
		"roles:\n"+
		"  core:\n"+
		"    name: core\n"+
		"    repositories:\n"+
		"      - path: .\n"+
		"  e2e:\n"+
		"    name: e2e\n"+
		"    repositories:\n"+
		"      - path: .\n")
}

// TestRunSDDKickoffSealWritesSummaryAndKickoffFile is task 9.1's base case:
// a correct seal exits 0 with a summary, and kickoff.yaml actually exists
// afterwards.
func TestRunSDDKickoffSealWritesSummaryAndKickoffFile(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")

	var out bytes.Buffer
	err := RunSDDKickoff([]string{
		"seal", "--cwd", root, "--change", "inc-99-example",
		"--execution-style", "checkpointed", "--handoff-policy", "per_checkpoint",
	}, &out)
	if err != nil {
		t.Fatalf("RunSDDKickoff() error = %v", err)
	}
	if !strings.Contains(out.String(), "inc-99-example") || !strings.Contains(out.String(), "checkpointed") {
		t.Fatalf("la salida no resume el sello:\n%s", out.String())
	}
	if _, statErr := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); statErr != nil {
		t.Fatalf("kickoff.yaml no se escribio: %v", statErr)
	}
}

// TestRunSDDKickoffSealOnAlreadySealedChangeReturnsWinnerWithoutRewriting
// covers D-01/D-02's single-write rule end to end through the CLI: a second
// seal with different flags exits 0, shows the WINNING (first) config, and
// never rewrites kickoff.yaml.
func TestRunSDDKickoffSealOnAlreadySealedChangeReturnsWinnerWithoutRewriting(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	sealedPath := filepath.Join(changeRoot, "kickoff.yaml")

	var first bytes.Buffer
	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "checkpointed", "--handoff-policy", "none"}, &first); err != nil {
		t.Fatalf("primer seal error = %v", err)
	}
	before, err := os.ReadFile(sealedPath)
	if err != nil {
		t.Fatalf("leer kickoff.yaml: %v", err)
	}

	var second bytes.Buffer
	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "continuous", "--handoff-policy", "per_checkpoint"}, &second); err != nil {
		t.Fatalf("segundo seal error = %v", err)
	}
	after, err := os.ReadFile(sealedPath)
	if err != nil {
		t.Fatalf("releer kickoff.yaml: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("el segundo seal reescribio kickoff.yaml:\nantes:\n%s\ndespues:\n%s", before, after)
	}
	if !strings.Contains(second.String(), "checkpointed") {
		t.Fatalf("la salida del segundo seal no muestra la configuracion ganadora (checkpointed):\n%s", second.String())
	}
}

// TestRunSDDKickoffSealInferThreeCases exercises design.md S8.1's three
// retro-seal cases (Phase 3) end to end through `seal --infer`.
func TestRunSDDKickoffSealInferThreeCases(t *testing.T) {
	t.Run("caso a: design.md con roles explicitos", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		writeRepoLikeAxiomYAML(t, root)
		writeCLIFixture(t, filepath.Join(changeRoot, "design.md"),
			"# Diseno\n\n```yaml\nroles:\n  - role: core\n    gate_policy: blocking\n  - role: e2e\n    gate_policy: deferred\n```\n")

		var out bytes.Buffer
		if err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "inc-99-example"}, &out); err != nil {
			t.Fatalf("RunSDDKickoff() error = %v", err)
		}
		if !strings.Contains(out.String(), "core") || !strings.Contains(out.String(), "e2e") || strings.Contains(out.String(), "fullstack") {
			t.Fatalf("la salida no refleja los roles inferidos del design.md (core, e2e, nunca fullstack):\n%s", out.String())
		}
	})

	t.Run("caso b: sin design.md sella fullstack", func(t *testing.T) {
		root, _ := newGovernanceWorkspace(t, "inc-99-example")

		var out bytes.Buffer
		if err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "inc-99-example"}, &out); err != nil {
			t.Fatalf("RunSDDKickoff() error = %v", err)
		}
		if !strings.Contains(out.String(), "fullstack") {
			t.Fatalf("la salida no refleja el rol fullstack por defecto:\n%s", out.String())
		}
	})

	t.Run("caso c: rol ausente de axiom.yaml propaga el error", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		writeRepoLikeAxiomYAML(t, root)
		writeCLIFixture(t, filepath.Join(changeRoot, "design.md"),
			"# Diseno\n\n```yaml\nroles:\n  - role: database\n    gate_policy: blocking\n```\n")

		err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "inc-99-example"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("RunSDDKickoff() = nil error, se esperaba propagar el error de DetectRoles (rol ausente de axiom.yaml)")
		}
		if _, statErr := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); !os.IsNotExist(statErr) {
			t.Fatalf("kickoff.yaml se escribio pese al error de deteccion de roles: statErr = %v", statErr)
		}
	})
}

// TestRunSDDKickoffSealOnArchivedRootRefusesNamingBugOrNewIncrement is
// D-14/REQ-21.18: a seal targeting a change root already under
// openspec/changes/archive/ is refused, naming the bug/new-increment path.
func TestRunSDDKickoffSealOnArchivedRootRefusesNamingBugOrNewIncrement(t *testing.T) {
	root := t.TempDir()
	archivedRoot := filepath.Join(root, "openspec", "changes", "archive", "inc-01-old")
	mustMkdirAllCLI(t, archivedRoot)

	err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-01-old", "--execution-style", "continuous", "--handoff-policy", "none"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunSDDKickoff() = nil error, se esperaba rechazo por raiz archivada (D-14)")
	}
	if !strings.Contains(err.Error(), "REQ-21.18") {
		t.Fatalf("error = %v, se esperaba que nombrase REQ-21.18 (bug o nuevo incremento)", err)
	}
	if _, statErr := os.Stat(filepath.Join(archivedRoot, "kickoff.yaml")); !os.IsNotExist(statErr) {
		t.Fatalf("kickoff.yaml se escribio pese al rechazo por raiz archivada: statErr = %v", statErr)
	}
}

// TestRunSDDKickoffUnknownSubcommand is T-8: an unrecognised kickoff
// subverb is refused.
func TestRunSDDKickoffUnknownSubcommand(t *testing.T) {
	err := RunSDDKickoff([]string{"bogus"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunSDDKickoff() = nil error, se esperaba rechazo de subcomando desconocido")
	}
}

// TestRunSDDKickoffNoSubcommand covers the empty-args edge of T-8.
func TestRunSDDKickoffNoSubcommand(t *testing.T) {
	err := RunSDDKickoff(nil, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunSDDKickoff() = nil error, se esperaba exigir un subcomando")
	}
}

// TestRunSDDKickoffHelp is T-8: --help and -h both exit 0 and mention both
// subcommands.
func TestRunSDDKickoffHelp(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var out bytes.Buffer
			if err := RunSDDKickoff([]string{flag}, &out); err != nil {
				t.Fatalf("RunSDDKickoff(%q) error = %v", flag, err)
			}
			if !strings.Contains(out.String(), "seal") || !strings.Contains(out.String(), "show") {
				t.Fatalf("la ayuda no menciona los subcomandos seal/show:\n%s", out.String())
			}
		})
	}
}

// TestRunSDDKickoffShowOnUnsealedChangeStatesExplicitly is task 9.1's
// "show sobre un cambio sin sellar" scenario: an explicit, unambiguous "no
// kickoff" statement, never a silent empty success.
func TestRunSDDKickoffShowOnUnsealedChangeStatesExplicitly(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, "inc-99-example")

	var out bytes.Buffer
	if err := RunSDDKickoff([]string{"show", "--cwd", root, "--change", "inc-99-example"}, &out); err != nil {
		t.Fatalf("RunSDDKickoff() error = %v", err)
	}
	if !strings.Contains(strings.ToLower(out.String()), "sin kickoff sellado") {
		t.Fatalf("la salida no declara explicitamente la ausencia de kickoff:\n%s", out.String())
	}
}

// TestRunSDDKickoffShowOnSealedChangeDisplaysIt confirms the mirror case:
// a sealed change's show output actually surfaces its configuration.
func TestRunSDDKickoffShowOnSealedChangeDisplaysIt(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, "inc-99-example")
	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "checkpointed", "--handoff-policy", "none"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("seal error = %v", err)
	}

	var out bytes.Buffer
	if err := RunSDDKickoff([]string{"show", "--cwd", root, "--change", "inc-99-example"}, &out); err != nil {
		t.Fatalf("RunSDDKickoff() error = %v", err)
	}
	if !strings.Contains(out.String(), "checkpointed") {
		t.Fatalf("la salida de show no refleja la configuracion sellada:\n%s", out.String())
	}
}

// La prueba que exigía que la carpeta del cambio existiera
// (TestRunSDDKickoffSealOnNonexistentChangeNamesResolvedPath) se invirtió por
// REQ-29.12: `seal` crea la carpeta tras las validaciones. Su sustituta es
// TestRunSDDKickoffSealOnNonexistentChangeCreatesTheFolder
// (sdd_kickoff_iso_test.go); `--infer` conserva el rechazo.

// --- REQ-26.9: handoff_policy is deprecated. The flag is accepted and ignored,
// never required, never named by help, errors or text, and the JSON echoes the
// value only when the file carries one.

const deprecatedPolicyKickoffYAML = `schema: axiom.sdd-kickoff/v1
change: inc-99-example
sealed_at: 2026-09-21T15:25:00Z
sealed_by: cli
kickoff:
  flow_mode: sdd
  execution_style: checkpointed
  execution_style_source: explicit
%s  roles:
    - role: fullstack
      gate_policy: blocking
      tasks_file: tasks.md
      verify_file: verify-report.md
lifecycle:
  deployment_target: local
  post_archive_policy: bug_only
`

// TestRunSDDKickoffSealWithoutHandoffPolicyWritesNoneAndNeverMentionsIt seals
// with no --handoff-policy: it succeeds, kickoff.yaml carries the compatibility
// value none, and neither the text summary nor show mentions the policy.
func TestRunSDDKickoffSealWithoutHandoffPolicyWritesNoneAndNeverMentionsIt(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")

	var sealOut bytes.Buffer
	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "checkpointed"}, &sealOut); err != nil {
		t.Fatalf("seal sin --handoff-policy error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
	if err != nil {
		t.Fatalf("leer kickoff.yaml: %v", err)
	}
	if !strings.Contains(string(raw), "handoff_policy: none") {
		t.Fatalf("kickoff.yaml sin handoff_policy: none:\n%s", raw)
	}

	var showOut bytes.Buffer
	if err := RunSDDKickoff([]string{"show", "--cwd", root, "--change", "inc-99-example"}, &showOut); err != nil {
		t.Fatalf("show error = %v", err)
	}
	for name, text := range map[string]string{"seal": sealOut.String(), "show": showOut.String()} {
		if strings.Contains(strings.ToLower(text), "handoff") {
			t.Fatalf("la salida de texto de %s menciona la politica de relevos:\n%s", name, text)
		}
	}
}

// TestRunSDDKickoffSealIgnoresDeprecatedFlagAndRejectsUnknownValue covers the
// flag as an accepted no-op: per_checkpoint never reaches kickoff.yaml and an
// unknown value is still refused without writing anything (A-6).
func TestRunSDDKickoffSealIgnoresDeprecatedFlagAndRejectsUnknownValue(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")

	err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "checkpointed", "--handoff-policy", "otra"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("seal --handoff-policy otra = nil error, se esperaba rechazo")
	}
	if _, statErr := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); !os.IsNotExist(statErr) {
		t.Fatalf("kickoff.yaml se escribio pese al rechazo del valor desconocido: statErr = %v", statErr)
	}

	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "checkpointed", "--handoff-policy", "per_checkpoint"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("seal --handoff-policy per_checkpoint error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
	if err != nil {
		t.Fatalf("leer kickoff.yaml: %v", err)
	}
	if strings.Contains(string(raw), "per_checkpoint") || !strings.Contains(string(raw), "handoff_policy: none") {
		t.Fatalf("el flag deprecado llego a kickoff.yaml:\n%s", raw)
	}
}

// TestRunSDDKickoffSealInferWritesNoneAsCompatibilityValue is REQ-21.4's
// retro-seal without any policy inference: none is only the compatibility value.
func TestRunSDDKickoffSealInferWritesNoneAsCompatibilityValue(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")

	var out bytes.Buffer
	if err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "inc-99-example"}, &out); err != nil {
		t.Fatalf("seal --infer error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
	if err != nil {
		t.Fatalf("leer kickoff.yaml: %v", err)
	}
	if !strings.Contains(string(raw), "handoff_policy: none") {
		t.Fatalf("el retro-sello no escribio handoff_policy: none:\n%s", raw)
	}
	if strings.Contains(strings.ToLower(out.String()), "handoff") {
		t.Fatalf("la salida del retro-sello menciona la politica de relevos:\n%s", out.String())
	}
}

// TestRunSDDKickoffHelpAndGateErrorDoNotNameHandoffPolicy covers the help and
// the sealing hint of `gate record`: neither suggests --handoff-policy.
func TestRunSDDKickoffHelpAndGateErrorDoNotNameHandoffPolicy(t *testing.T) {
	var help bytes.Buffer
	if err := RunSDDKickoff([]string{"--help"}, &help); err != nil {
		t.Fatalf("--help error = %v", err)
	}
	if strings.Contains(help.String(), "handoff-policy") {
		t.Fatalf("la ayuda de kickoff nombra --handoff-policy:\n%s", help.String())
	}

	root, _ := newGovernanceWorkspace(t, "inc-99-example")
	err := RunSDDGate([]string{"record", "--cwd", root, "--change", "inc-99-example", "--gate", "role-apply:core", "--decision", "approved"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("gate record role-apply sin kickoff = nil error, se esperaba rechazo")
	}
	if strings.Contains(err.Error(), "handoff-policy") {
		t.Fatalf("el mensaje de error sugiere --handoff-policy: %v", err)
	}
}

// TestRunSDDKickoffJSONEmitsHandoffPolicyOnlyWhenTheFileCarriesIt covers
// `kickoff show --json` and `kickoff seal --json`: handoffPolicy is omitted
// when empty and echoed as read otherwise.
func TestRunSDDKickoffJSONEmitsHandoffPolicyOnlyWhenTheFileCarriesIt(t *testing.T) {
	writeKickoff := func(t *testing.T, policyLine string) string {
		t.Helper()
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		writeCLIFixture(t, filepath.Join(changeRoot, "kickoff.yaml"), strings.Replace(deprecatedPolicyKickoffYAML, "%s", policyLine, 1))
		return root
	}
	show := func(t *testing.T, root string) string {
		t.Helper()
		var out bytes.Buffer
		if err := RunSDDKickoff([]string{"show", "--cwd", root, "--change", "inc-99-example", "--json"}, &out); err != nil {
			t.Fatalf("show --json error = %v", err)
		}
		return out.String()
	}

	if out := show(t, writeKickoff(t, "")); strings.Contains(out, "handoffPolicy") {
		t.Fatalf("show --json emite handoffPolicy con la clave ausente:\n%s", out)
	}
	if out := show(t, writeKickoff(t, "  handoff_policy: per_checkpoint\n")); !strings.Contains(out, `"handoffPolicy": "per_checkpoint"`) {
		t.Fatalf("show --json no devuelve el valor leido:\n%s", out)
	}

	root, _ := newGovernanceWorkspace(t, "inc-99-example")
	var sealOut bytes.Buffer
	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "checkpointed", "--json"}, &sealOut); err != nil {
		t.Fatalf("seal --json error = %v", err)
	}
	if !strings.Contains(sealOut.String(), `"handoffPolicy": "none"`) {
		t.Fatalf("seal --json no refleja el valor de compatibilidad escrito:\n%s", sealOut.String())
	}
}

// TestRunSDDKickoffSealDocsPublication cubre REQ-27.4 de extremo a extremo: `continuous` con
// `--docs-publication auto` escribe el campo; con checkpointed o --infer se rechaza sin escribir
// kickoff.yaml; sellar sin el flag no escribe el campo; un segundo seal no lo reescribe.
func TestRunSDDKickoffSealDocsPublication(t *testing.T) {
	t.Run("continuous con auto escribe el campo", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		var out bytes.Buffer
		if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "continuous", "--docs-publication", "auto"}, &out); err != nil {
			t.Fatalf("RunSDDKickoff() error = %v", err)
		}
		data, err := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "docs_publication: auto") {
			t.Fatalf("kickoff.yaml sin docs_publication: auto:\n%s", data)
		}
		if !strings.Contains(out.String(), "docs_publication=auto") {
			t.Fatalf("el resumen no muestra la autorizacion:\n%s", out.String())
		}
	})
	t.Run("sin el flag no se escribe el campo", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "continuous"}, &bytes.Buffer{}); err != nil {
			t.Fatalf("RunSDDKickoff() error = %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
		if strings.Contains(string(data), "docs_publication") {
			t.Fatalf("kickoff.yaml no debe llevar docs_publication:\n%s", data)
		}
	})
	for name, args := range map[string][]string{
		"checkpointed": {"--execution-style", "checkpointed", "--docs-publication", "auto"},
		"infer":        {"--infer", "--docs-publication", "auto"},
		"otro valor":   {"--execution-style", "continuous", "--docs-publication", "otro"},
	} {
		t.Run("rechazo con "+name, func(t *testing.T) {
			root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
			full := append([]string{"seal", "--cwd", root, "--change", "inc-99-example"}, args...)
			if err := RunSDDKickoff(full, &bytes.Buffer{}); err == nil {
				t.Fatal("se esperaba un rechazo")
			}
			if _, err := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); err == nil {
				t.Fatal("un rechazo no debe escribir kickoff.yaml")
			}
		})
	}
	t.Run("el segundo seal no reescribe", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "continuous"}, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		var second bytes.Buffer
		if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "continuous", "--docs-publication", "auto"}, &second); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(filepath.Join(changeRoot, "kickoff.yaml"))
		if strings.Contains(string(data), "docs_publication") || !strings.Contains(second.String(), "Kickoff ya estaba sellado") {
			t.Fatalf("el segundo seal no debe reescribir:\n%s\n%s", data, second.String())
		}
	})
}

// TestRunSDDKickoffSealContinuousRosterRule cubre REQ-28.12 y REQ-21.4 de extremo a extremo: continuous
// solo se sella con el rol único fullstack, el rechazo no escribe nada, un seal repetido con los
// argumentos antiguos se rechaza dejando el kickoff intacto y el retrosellado sigue admitiendo roles.
func TestRunSDDKickoffSealContinuousRosterRule(t *testing.T) {
	t.Run("continuous con roles falla sin crear kickoff.yaml", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "continuous", "--role", "core", "--role", "web"}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "--execution-style checkpointed") {
			t.Fatalf("error = %v, se esperaba el rechazo con la salida checkpointed", err)
		}
		if _, statErr := os.Stat(filepath.Join(changeRoot, "kickoff.yaml")); !os.IsNotExist(statErr) {
			t.Fatalf("kickoff.yaml no debía existir tras el rechazo: %v", statErr)
		}
	})

	t.Run("un seal repetido con continuous y roles se rechaza y deja el fichero intacto", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		sealed := kickoff.Kickoff{
			Schema:   kickoff.KickoffSchemaV1,
			Change:   "inc-99-example",
			SealedBy: "prueba",
			Config: kickoff.FlowConfig{
				FlowMode:       kickoff.FlowSDD,
				ExecutionStyle: kickoff.ExecutionContinuous,
				Roles: []kickoff.KickoffRole{
					{Role: "core", GatePolicy: multirole.PolicyBlocking, TasksFile: "tasks.core.md", VerifyFile: "verify-report.core.md"},
					{Role: "web", GatePolicy: multirole.PolicyBlocking, TasksFile: "tasks.web.md", VerifyFile: "verify-report.web.md"},
				},
			},
			Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		}
		if _, _, err := kickoff.Seal(changeRoot, sealed); err != nil {
			t.Fatalf("sellado previo: %v", err)
		}
		path := filepath.Join(changeRoot, "kickoff.yaml")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("leer kickoff.yaml: %v", err)
		}
		err = RunSDDKickoff([]string{"seal", "--cwd", root, "--change", "inc-99-example", "--execution-style", "continuous", "--role", "core", "--role", "web"}, &bytes.Buffer{})
		if err == nil || !errors.Is(err, kickoff.ErrContinuousRequiresFullstack) {
			t.Fatalf("error = %v, se esperaba ErrContinuousRequiresFullstack", err)
		}
		after, err := os.ReadFile(path)
		if err != nil || string(before) != string(after) {
			t.Fatalf("kickoff.yaml cambió tras el rechazo (err=%v)", err)
		}
		var show bytes.Buffer
		if err := RunSDDKickoff([]string{"show", "--cwd", root, "--change", "inc-99-example"}, &show); err != nil || !strings.Contains(show.String(), "core") {
			t.Fatalf("show debía seguir leyendo el kickoff sellado (err=%v):\n%s", err, show.String())
		}
	})

	t.Run("seal --infer con roles detectados sigue sellando continuous", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		writeRepoLikeAxiomYAML(t, root)
		writeCLIFixture(t, filepath.Join(changeRoot, "design.md"),
			"# Diseno\n\n```yaml\nroles:\n  - role: core\n    gate_policy: blocking\n  - role: e2e\n    gate_policy: deferred\n```\n")
		var out bytes.Buffer
		if err := RunSDDKickoff([]string{"seal", "--infer", "--cwd", root, "--change", "inc-99-example"}, &out); err != nil {
			t.Fatalf("seal --infer error = %v", err)
		}
		if !strings.Contains(out.String(), "execution_style=continuous") || !strings.Contains(out.String(), "sealed_by=inferred") {
			t.Fatalf("el retrosellado debía sellar continuous inferido:\n%s", out.String())
		}
	})

	t.Run("la ayuda enuncia la regla", func(t *testing.T) {
		var help bytes.Buffer
		if err := RunSDDKickoff([]string{"--help"}, &help); err != nil {
			t.Fatalf("--help error = %v", err)
		}
		if !strings.Contains(help.String(), "continuous solo con el rol fullstack") {
			t.Fatalf("la ayuda no enuncia la regla:\n%s", help.String())
		}
	})
}
