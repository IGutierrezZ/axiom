package kickoff

import (
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// sealArgs parsea un sellado explícito y falla la prueba si la sintaxis no es válida.
func sealArgs(t *testing.T, extra ...string) SealArgs {
	t.Helper()
	parsed, err := ParseSealArgs(append([]string{"--change", "inc-99-example"}, extra...))
	if err != nil {
		t.Fatalf("ParseSealArgs(%v) error = %v", extra, err)
	}
	return parsed
}

func sealRoleNames(parsed SealArgs) string {
	names := make([]string, 0, len(parsed.Roles))
	for _, r := range parsed.Roles {
		names = append(names, r.Role)
	}
	return strings.Join(names, ",")
}

// REQ-28.19: en multirepo el sellado deriva checkpointed y rechaza continuous.
func TestSealArgsApplyTopologyMultirepo(t *testing.T) {
	t.Run("sin estilo deriva checkpointed con origen topology", func(t *testing.T) {
		parsed := sealArgs(t)
		if err := parsed.ApplyTopology(workspace.TopologyMultirepo); err != nil {
			t.Fatalf("ApplyTopology() error = %v", err)
		}
		if parsed.ExecutionStyle != ExecutionCheckpointed || parsed.ExecutionStyleSource != "topology" {
			t.Fatalf("estilo = %q origen = %q, se esperaba checkpointed/topology", parsed.ExecutionStyle, parsed.ExecutionStyleSource)
		}
	})

	forbidden := []struct {
		name string
		args []string
		want string
	}{
		{"continuous explicito", []string{"--execution-style", "continuous"}, "--execution-style continuous"},
		{"ritmo auto", []string{"--from-session-pace", "auto"}, "--from-session-pace auto"},
		{"continuous explicito sobre el ritmo", []string{"--from-session-pace", "interactive", "--execution-style", "continuous"}, "--execution-style continuous"},
	}
	for _, tt := range forbidden {
		t.Run("rechaza "+tt.name, func(t *testing.T) {
			parsed := sealArgs(t, tt.args...)
			err := parsed.ApplyTopology(workspace.TopologyMultirepo)
			if err == nil || !errors.Is(err, ErrTopologyForbidsContinuous) {
				t.Fatalf("ApplyTopology() error = %v, se esperaba ErrTopologyForbidsContinuous", err)
			}
			for _, want := range []string{"multirepo", "checkpointed", tt.want} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, se esperaba que nombrara %q", err, want)
				}
			}
		})
	}

	accepted := []struct {
		name  string
		args  []string
		style ExecutionStyle
		roles string
	}{
		{"checkpointed con core y web", []string{"--execution-style", "checkpointed", "--role", "core", "--role", "web"}, ExecutionCheckpointed, "core,web"},
		{"ritmo interactive", []string{"--from-session-pace", "interactive", "--role", "core"}, ExecutionCheckpointed, "core"},
		{"checkpointed explicito sobre auto", []string{"--from-session-pace", "auto", "--execution-style", "checkpointed", "--role", "core"}, ExecutionCheckpointed, "core"},
	}
	for _, tt := range accepted {
		t.Run("acepta "+tt.name, func(t *testing.T) {
			parsed := sealArgs(t, tt.args...)
			if err := parsed.ApplyTopology(workspace.TopologyMultirepo); err != nil {
				t.Fatalf("ApplyTopology() error = %v", err)
			}
			if parsed.ExecutionStyle != tt.style || sealRoleNames(parsed) != tt.roles {
				t.Fatalf("estilo = %q roles = %q, se esperaba %q / %q", parsed.ExecutionStyle, sealRoleNames(parsed), tt.style, tt.roles)
			}
		})
	}
}

// REQ-28.19 y REQ-21.6: en monorepo-embedded solo se sella fullstack.
func TestSealArgsApplyTopologyMonorepo(t *testing.T) {
	rejected := []struct {
		name string
		args []string
		want string
	}{
		{"core y web", []string{"--execution-style", "checkpointed", "--role", "core", "--role", "web"}, "core, web"},
		{"fullstack y qa", []string{"--execution-style", "checkpointed", "--role", "fullstack", "--role", "qa"}, "fullstack, qa"},
		{"un rol especializado con ritmo interactive", []string{"--from-session-pace", "interactive", "--role", "core"}, "core"},
	}
	for _, tt := range rejected {
		t.Run("rechaza "+tt.name, func(t *testing.T) {
			parsed := sealArgs(t, tt.args...)
			err := parsed.ApplyTopology(workspace.TopologyMonorepoEmbedded)
			if err == nil || !errors.Is(err, ErrTopologyRequiresFullstack) {
				t.Fatalf("ApplyTopology() error = %v, se esperaba ErrTopologyRequiresFullstack", err)
			}
			for _, want := range []string{"monorepo-embedded", "fullstack", tt.want, "multirepo"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, se esperaba que nombrara %q", err, want)
				}
			}
		})
	}

	t.Run("sin --role sella fullstack", func(t *testing.T) {
		parsed := sealArgs(t, "--execution-style", "checkpointed")
		if err := parsed.ApplyTopology(workspace.TopologyMonorepoEmbedded); err != nil {
			t.Fatalf("ApplyTopology() error = %v", err)
		}
		if sealRoleNames(parsed) != "fullstack" {
			t.Fatalf("roles = %q, se esperaba fullstack", sealRoleNames(parsed))
		}
	})

	t.Run("continuous con fullstack y politica deferred se acepta", func(t *testing.T) {
		parsed := sealArgs(t, "--execution-style", "continuous", "--role", "fullstack:deferred")
		if err := parsed.ApplyTopology(workspace.TopologyMonorepoEmbedded); err != nil {
			t.Fatalf("ApplyTopology() error = %v", err)
		}
		if parsed.ExecutionStyle != ExecutionContinuous {
			t.Fatalf("estilo = %q, se esperaba continuous", parsed.ExecutionStyle)
		}
	})

	t.Run("sin estilo sigue siendo obligatorio", func(t *testing.T) {
		parsed := sealArgs(t)
		err := parsed.ApplyTopology(workspace.TopologyMonorepoEmbedded)
		if err == nil || !strings.Contains(err.Error(), "REQ-21.5") {
			t.Fatalf("ApplyTopology() error = %v, se esperaba el rechazo de REQ-21.5", err)
		}
	})
}

// La regla de topología gana a la de continuous cuando ambas aplican.
func TestSealArgsApplyTopologyPrecedence(t *testing.T) {
	parsed := sealArgs(t, "--execution-style", "continuous", "--role", "core")
	err := parsed.ApplyTopology(workspace.TopologyMonorepoEmbedded)
	if err == nil || !errors.Is(err, ErrTopologyRequiresFullstack) {
		t.Fatalf("ApplyTopology() error = %v, se esperaba el mensaje de topología", err)
	}
	if errors.Is(err, ErrContinuousRequiresFullstack) {
		t.Fatalf("error = %v, no debe ganar la regla de continuous", err)
	}
}

// REQ-21.5, REQ-28.12 y REQ-28.13 siguen intactas sin axiom.yaml (topología "").
func TestSealArgsApplyTopologyWithoutTopology(t *testing.T) {
	t.Run("estilo obligatorio", func(t *testing.T) {
		parsed := sealArgs(t)
		err := parsed.ApplyTopology("")
		if err == nil || !strings.Contains(err.Error(), "REQ-21.5") {
			t.Fatalf("ApplyTopology() error = %v, se esperaba REQ-21.5", err)
		}
	})
	t.Run("continuous con roles se rechaza por la regla de roster", func(t *testing.T) {
		parsed := sealArgs(t, "--execution-style", "continuous", "--role", "core", "--role", "web")
		err := parsed.ApplyTopology("")
		if err == nil || !errors.Is(err, ErrContinuousRequiresFullstack) || !strings.Contains(err.Error(), "--execution-style checkpointed") {
			t.Fatalf("ApplyTopology() error = %v, se esperaba ErrContinuousRequiresFullstack", err)
		}
	})
	t.Run("ritmo auto con roles nombra la derivacion", func(t *testing.T) {
		parsed := sealArgs(t, "--from-session-pace", "auto", "--role", "core")
		err := parsed.ApplyTopology("")
		if err == nil || !errors.Is(err, ErrContinuousRequiresFullstack) || !strings.Contains(err.Error(), "--from-session-pace auto deriva continuous") {
			t.Fatalf("ApplyTopology() error = %v", err)
		}
	})
	t.Run("checkpointed con roles sella", func(t *testing.T) {
		parsed := sealArgs(t, "--from-session-pace", "interactive", "--role", "core")
		if err := parsed.ApplyTopology(""); err != nil {
			t.Fatalf("ApplyTopology() error = %v", err)
		}
	})
}
