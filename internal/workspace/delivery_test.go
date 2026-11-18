package workspace

import (
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

func deliveryNode(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(source), &doc); err != nil {
		t.Fatalf("YAML de prueba inválido: %v", err)
	}
	node, ok := mappingValue(doc.Content[0], "delivery")
	if !ok {
		t.Fatalf("el YAML de prueba no tiene bloque delivery")
	}
	return node
}

// REQ-32.1: el entero es estricto y la lista de claves, cerrada.
func TestDecodeDelivery(t *testing.T) {
	valid := []struct {
		name   string
		source string
		want   int
	}{
		{"seiscientas", "delivery:\n  split_hint_lines: 600\n", 600},
		{"una", "delivery:\n  split_hint_lines: 1\n", 1},
		{"mapeo vacío", "delivery: {}\n", 0},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeDelivery(deliveryNode(t, tc.source))
			if err != nil || got == nil || got.SplitHintLines != tc.want {
				t.Fatalf("decodeDelivery = %+v, %v; se esperaba split_hint_lines=%d", got, err, tc.want)
			}
		})
	}

	invalid := []struct {
		name   string
		source string
		wants  []string
	}{
		{"cero", "delivery:\n  split_hint_lines: 0\n", []string{"delivery.split_hint_lines", "mayor que cero"}},
		{"negativo", "delivery:\n  split_hint_lines: -1\n", []string{"delivery.split_hint_lines", "mayor que cero"}},
		{"entre comillas", "delivery:\n  split_hint_lines: \"600\"\n", []string{"delivery.split_hint_lines", "sin comillas"}},
		{"notación científica", "delivery:\n  split_hint_lines: 6e2\n", []string{"delivery.split_hint_lines", "6e2"}},
		{"decimal", "delivery:\n  split_hint_lines: 1.5\n", []string{"delivery.split_hint_lines", "1.5"}},
		{"booleano", "delivery:\n  split_hint_lines: true\n", []string{"delivery.split_hint_lines", "true"}},
		{"nulo", "delivery:\n  split_hint_lines:\n", []string{"delivery.split_hint_lines", "nulo"}},
		{"lista", "delivery:\n  split_hint_lines: [600]\n", []string{"delivery.split_hint_lines", "lista"}},
		{"mapeo como valor", "delivery:\n  split_hint_lines: {a: b}\n", []string{"delivery.split_hint_lines", "mapeo"}},
		{"desbordamiento", "delivery:\n  split_hint_lines: 9223372036854775808\n", []string{"delivery.split_hint_lines"}},
		{"clave desconocida", "delivery:\n  budget: 400\n", []string{"budget", "claves admitidas: split_hint_lines"}},
		{"clave duplicada", "delivery:\n  split_hint_lines: 600\n  split_hint_lines: 700\n", []string{"clave duplicada 'split_hint_lines'"}},
		{"bloque nulo", "delivery:\n", []string{"bloque 'delivery'", "mapeo"}},
		{"bloque escalar", "delivery: 600\n", []string{"bloque 'delivery'", "mapeo"}},
		{"bloque lista", "delivery:\n  - split_hint_lines\n", []string{"bloque 'delivery'", "mapeo"}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeDelivery(deliveryNode(t, tc.source))
			if got != nil {
				t.Errorf("un bloque inválido no devuelve configuración: %+v", got)
			}
			wantErrorContains(t, err, tc.wants...)
		})
	}
}

// REQ-32.1: carga por capas (parseLayered, ParseConfig, ReadDeclarations y
// Locate) de un bloque `delivery:` válido, inválido y ausente.
func TestDeliveryLoad(t *testing.T) {
	t.Run("sin bloque nada cambia", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(declarationsBase))
		if baseErr != nil || declarations.Delivery != nil || declarations.DeliveryErr != nil {
			t.Fatalf("sin bloque no hay declaración ni error: base=%v %+v", baseErr, declarations)
		}
		if cfg == nil || cfg.Delivery != nil {
			t.Errorf("sin bloque WorkspaceConfig.Delivery es nil: %+v", cfg)
		}
		if _, err := ParseConfig([]byte(declarationsBase)); err != nil {
			t.Fatalf("ParseConfig sin bloque: %v", err)
		}
	})

	t.Run("valido se proyecta", func(t *testing.T) {
		blocks := "delivery:\n  split_hint_lines: 600\n"
		cfg, baseErr, declarations := parseDeclarations(t, blocks)
		if baseErr != nil || declarations.DeliveryErr != nil || declarations.Delivery == nil || declarations.Delivery.SplitHintLines != 600 {
			t.Fatalf("base=%v delivery=%v declarado=%+v", baseErr, declarations.DeliveryErr, declarations.Delivery)
		}
		if cfg == nil || cfg.Delivery == nil || cfg.Delivery.SplitHintLines != 600 {
			t.Errorf("WorkspaceConfig.Delivery debe proyectar lo declarado: %+v", cfg)
		}
		if _, err := ParseConfig([]byte(declarationsBase + blocks)); err != nil {
			t.Fatalf("ParseConfig debe aceptarlo: %v", err)
		}
	})

	t.Run("invalido deja nil y falla la carga", func(t *testing.T) {
		blocks := "delivery:\n  split_hint_lines: \"600\"\n"
		cfg, baseErr, declarations := parseDeclarations(t, blocks)
		if baseErr != nil || declarations.Delivery != nil || declarations.DeliveryErr == nil {
			t.Fatalf("base=%v %+v", baseErr, declarations)
		}
		if cfg == nil || cfg.Delivery != nil {
			t.Errorf("con bloque inválido WorkspaceConfig.Delivery es nil: %+v", cfg)
		}
		_, err := ParseConfig([]byte(declarationsBase + blocks))
		wantErrorContains(t, err, "delivery.split_hint_lines", "sin comillas")
	})

	t.Run("errores de delivery y de otro bloque se acumulan", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + "delivery:\n  split_hint_lines: 0\nworkflow:\n  lane: nope\n"))
		wantErrorContains(t, err, "delivery.split_hint_lines", "lane")
	})

	t.Run("ReadDeclarations y Locate", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		configPath := filepath.Join(ws.Master, "axiom.yaml")
		workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"delivery:\n  split_hint_lines: 600\n")
		got := ReadDeclarations(configPath)
		if got.DeliveryErr != nil || got.Delivery == nil || got.Delivery.SplitHintLines != 600 {
			t.Fatalf("ReadDeclarations: %+v", got)
		}
		loc := mustLocate(t, ws.Master)
		if loc.Config == nil || loc.ConfigErr != nil || loc.Config.Delivery == nil || loc.Config.Delivery.SplitHintLines != 600 {
			t.Fatalf("Locate debe proyectar delivery: %+v", loc)
		}
	})
}

// REQ-32.1: workspace validate carga con LoadConfig y falla con un `delivery:`
// inválido; con un bloque válido carga sin error.
func TestWorkspaceValidateDelivery(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	configPath := filepath.Join(ws.Master, "axiom.yaml")

	workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"delivery:\n  split_hint_lines: 600\n")
	if _, err := LoadConfig(configPath); err != nil {
		t.Fatalf("un delivery válido debe cargar: %v", err)
	}
	for name, blocks := range map[string]string{
		"cero":              "delivery:\n  split_hint_lines: 0\n",
		"clave desconocida": "delivery:\n  budget: 400\n",
		"bloque nulo":       "delivery:\n",
	} {
		workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+blocks)
		if _, err := LoadConfig(configPath); err == nil {
			t.Errorf("%s: LoadConfig debe fallar", name)
		}
	}
}
