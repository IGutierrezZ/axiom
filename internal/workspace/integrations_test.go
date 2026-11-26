package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// integrationsManifest es el plugin de prueba de este fichero. No se importa integrationstest
// porque depende de internal/integrations, que a su vez importa workspace.
func integrationsManifest() manifest.Manifest {
	return manifest.Manifest{
		Schema:       manifest.SchemaV1,
		ID:           "fake",
		Title:        "Plugin de prueba",
		Capabilities: []manifest.Capability{manifest.CapItemsRead},
		Events:       []manifest.EventType{manifest.EventArchive},
		Config: []manifest.ConfigField{
			{Key: "project", Type: manifest.FieldString, Required: true},
			{Key: "area", Type: manifest.FieldString, Enum: []string{"alfa", "beta"}},
			{Key: "retries", Type: manifest.FieldInt, Default: 3},
			{Key: "dry_run", Type: manifest.FieldBool},
			{Key: "labels", Type: manifest.FieldStringList, Enum: []string{"uno", "dos"}},
			{Key: "mode", Type: manifest.FieldString, Default: "normal"},
		},
		Secrets:  []manifest.SecretSpec{{Name: "token", Required: true}},
		Identity: manifest.IdentitySpec{Source: manifest.IdentityNone},
	}
}

func otherIntegrationsManifest() manifest.Manifest {
	m := integrationsManifest()
	m.ID = "azure-devops"
	m.Config = nil
	m.Secrets = nil
	return m
}

func installCatalog(t *testing.T) {
	t.Helper()
	manifest.SetCatalogForTest(t, integrationsManifest(), otherIntegrationsManifest())
}

func integrationsNode(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(source), &doc); err != nil {
		t.Fatalf("YAML de prueba inválido: %v", err)
	}
	node, ok := mappingValue(doc.Content[0], "integrations")
	if !ok {
		t.Fatalf("el YAML de prueba no tiene bloque integrations")
	}
	return node
}

func TestDecodeIntegrationsValid(t *testing.T) {
	installCatalog(t)

	t.Run("bloque vacío no configura nada", func(t *testing.T) {
		got, err := decodeIntegrations(integrationsNode(t, "integrations: {}\n"))
		if err != nil || got == nil || len(got.Plugins) != 0 {
			t.Fatalf("decodeIntegrations({}) = %+v, %v", got, err)
		}
		if len(got.Active()) != 0 {
			t.Fatal("sin plugins no hay activos")
		}
	})

	t.Run("plugin con valores tipados y defaults aplicados", func(t *testing.T) {
		got, err := decodeIntegrations(integrationsNode(t, `integrations:
  fake:
    project: marco
    area: alfa
    dry_run: true
    labels: [uno, dos]
`))
		if err != nil {
			t.Fatalf("decodeIntegrations: %v", err)
		}
		if len(got.Plugins) != 1 {
			t.Fatalf("plugins = %+v", got.Plugins)
		}
		p := got.Plugins[0]
		if p.ID != "fake" || !p.Enabled || p.Line != 2 {
			t.Fatalf("plugin = %+v", p)
		}
		want := map[string]any{
			"project": "marco",
			"area":    "alfa",
			"dry_run": true,
			"labels":  []string{"uno", "dos"},
			"retries": 3,        // default
			"mode":    "normal", // default
		}
		if !reflect.DeepEqual(p.Values, want) {
			t.Fatalf("Values = %#v, quería %#v", p.Values, want)
		}
	})

	t.Run("valores declarados pisan los defaults", func(t *testing.T) {
		got, err := decodeIntegrations(integrationsNode(t, "integrations:\n  fake:\n    project: p\n    retries: 7\n"))
		if err != nil || got.Plugins[0].Values["retries"] != 7 {
			t.Fatalf("retries = %+v, %v", got, err)
		}
	})

	t.Run("enabled false no exige obligatorias y Active lo filtra", func(t *testing.T) {
		got, err := decodeIntegrations(integrationsNode(t, "integrations:\n  fake:\n    enabled: false\n  azure-devops:\n    enabled: true\n"))
		if err != nil {
			t.Fatalf("decodeIntegrations: %v", err)
		}
		if len(got.Plugins) != 2 || got.Plugins[0].Enabled {
			t.Fatalf("plugins = %+v", got.Plugins)
		}
		active := got.Active()
		if len(active) != 1 || active[0].ID != "azure-devops" {
			t.Fatalf("Active() = %+v", active)
		}
	})

	t.Run("plugin sin claves ni nada que exigir", func(t *testing.T) {
		got, err := decodeIntegrations(integrationsNode(t, "integrations:\n  azure-devops: {}\n"))
		if err != nil || len(got.Plugins) != 1 || !got.Plugins[0].Enabled {
			t.Fatalf("decodeIntegrations = %+v, %v", got, err)
		}
	})
}

func TestDecodeIntegrationsErrors(t *testing.T) {
	installCatalog(t)

	tests := []struct {
		name   string
		source string
		wants  []string
		not    []string
	}{
		{"bloque nulo", "integrations:\n", []string{"bloque 'integrations'", "mapeo"}, nil},
		{"bloque que no es mapeo", "integrations: [x]\n", []string{"bloque 'integrations'", "mapeo"}, nil},
		{"plugin desconocido", "integrations:\n  azure:\n    project: p\n", []string{"plugin desconocido 'azure'", "línea 2", "plugins admitidos: fake, azure-devops"}, nil},
		{"valor de plugin nulo", "integrations:\n  fake:\n", []string{"integrations.fake", "mapeo"}, nil},
		{"valor de plugin no mapeo", "integrations:\n  fake: [a]\n", []string{"integrations.fake", "mapeo"}, nil},
		{"clave desconocida", "integrations:\n  fake:\n    project: p\n    clave_inexistente: 1\n", []string{"clave desconocida 'clave_inexistente'", "línea 4", "claves admitidas: enabled, project, area, retries, dry_run, labels, mode"}, nil},
		{"clave duplicada", "integrations:\n  fake:\n    project: p\n    project: q\n", []string{"clave duplicada 'project'", "línea 4"}, nil},
		{"plugin duplicado", "integrations:\n  fake:\n    project: p\n  fake:\n    project: q\n", []string{"plugin duplicado 'fake'", "línea 4"}, nil},
		{"tipo incorrecto en cadena", "integrations:\n  fake:\n    project: 5\n", []string{"integrations.fake.project", "cadena"}, nil},
		{"tipo incorrecto en entero", "integrations:\n  fake:\n    project: p\n    retries: muchos\n", []string{"integrations.fake.retries", "entero"}, nil},
		{"tipo incorrecto en booleano", "integrations:\n  fake:\n    project: p\n    dry_run: \"true\"\n", []string{"integrations.fake.dry_run", "true o false"}, nil},
		{"tipo incorrecto en lista", "integrations:\n  fake:\n    project: p\n    labels: uno\n", []string{"integrations.fake.labels", "lista"}, nil},
		{"elemento de lista que no es cadena", "integrations:\n  fake:\n    project: p\n    labels: [uno, 2]\n", []string{"integrations.fake.labels", "cadenas"}, nil},
		{"fuera del enumerado", "integrations:\n  fake:\n    project: p\n    area: gamma\n", []string{"integrations.fake.area", "'gamma'", "alfa, beta"}, nil},
		{"elemento de lista fuera del enumerado", "integrations:\n  fake:\n    project: p\n    labels: [uno, tres]\n", []string{"integrations.fake.labels", "'tres'", "uno, dos"}, nil},
		{"obligatoria ausente", "integrations:\n  fake:\n    area: alfa\n", []string{"integrations.fake.project", "obligatoria", "línea 2"}, nil},
		{"enabled no booleano", "integrations:\n  fake:\n    enabled: quizás\n    project: p\n", []string{"integrations.fake.enabled", "true o false"}, nil},
		{"valor nulo", "integrations:\n  fake:\n    project:\n", []string{"integrations.fake.project", "nulo"}, nil},
		{"secreto como clave es desconocida y no se imprime", "integrations:\n  fake:\n    project: p\n    token: abc-valor-secreto\n", []string{"clave desconocida 'token'", "integrations.fake"}, []string{"abc-valor-secreto"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeIntegrations(integrationsNode(t, tt.source))
			if got != nil {
				t.Errorf("un bloque inválido no devuelve configuración: %+v", got)
			}
			wantErrorContains(t, err, tt.wants...)
			for _, n := range tt.not {
				if err != nil && strings.Contains(err.Error(), n) {
					t.Errorf("el error no debe contener %q:\n%v", n, err)
				}
			}
		})
	}

	t.Run("sin catálogo el mensaje dice que no hay plugins en esta versión", func(t *testing.T) {
		manifest.SetCatalogForTest(t) // catálogo vacío
		_, err := decodeIntegrations(integrationsNode(t, "integrations:\n  fake:\n    project: p\n"))
		wantErrorContains(t, err, "plugin desconocido 'fake'", "ninguno en esta versión")
	})

	t.Run("acumula todas las violaciones", func(t *testing.T) {
		_, err := decodeIntegrations(integrationsNode(t, "integrations:\n  azure: {}\n  fake:\n    area: gamma\n    nueva: 1\n"))
		wantErrorContains(t, err, "plugin desconocido 'azure'", "'gamma'", "clave desconocida 'nueva'", "integrations.fake.project")
	})
}

func TestIntegrationsConfigActiveOnNil(t *testing.T) {
	var c *IntegrationsConfig
	if got := c.Active(); len(got) != 0 {
		t.Fatalf("Active() sobre nil = %+v", got)
	}
}

func TestParseLayeredLoadsIntegrationsBlock(t *testing.T) {
	installCatalog(t)

	cfg, baseErr, declarations := parseDeclarations(t, "integrations:\n  fake:\n    project: marco\n")
	if baseErr != nil || declarations.IntegrationsErr != nil {
		t.Fatalf("base=%v integraciones=%v", baseErr, declarations.IntegrationsErr)
	}
	if declarations.Integrations == nil || len(declarations.Integrations.Plugins) != 1 {
		t.Fatalf("Declarations.Integrations = %+v", declarations.Integrations)
	}
	if cfg.Integrations != declarations.Integrations {
		t.Fatalf("WorkspaceConfig.Integrations debe proyectar lo declarado: %+v", cfg.Integrations)
	}
}

func TestSameInvalidYAMLFailsEverywhereWithTheSameMessage(t *testing.T) {
	installCatalog(t)
	const invalid = declarationsBase + "integrations:\n  fake:\n    project: p\n    clave_inexistente: 1\n"
	wants := []string{"clave desconocida 'clave_inexistente'", "línea 13", "claves admitidas: enabled, project"}

	_, parseErr := ParseConfig([]byte(invalid))
	wantErrorContains(t, parseErr, wants...)

	dir := t.TempDir()
	file := filepath.Join(dir, "axiom.yaml")
	if err := os.WriteFile(file, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	_, loadErr := LoadConfig(file)
	wantErrorContains(t, loadErr, wants...)

	ws := workspacetest.Embedded(t, ".")
	if err := os.WriteFile(filepath.Join(ws.Specs, "axiom.yaml"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	loc, locateErr := Locate(ws.Specs)
	if locateErr != nil {
		t.Fatalf("Locate no rechaza una declaración inválida (D-03): %v", locateErr)
	}
	wantErrorContains(t, loc.Declarations.IntegrationsErr, wants...)

	if parseErr.Error() != loadErr.Error() {
		t.Fatalf("ParseConfig y LoadConfig deben dar el mismo mensaje:\n%v\n%v", parseErr, loadErr)
	}
	if !strings.Contains(loadErr.Error(), loc.Declarations.IntegrationsErr.Error()) {
		t.Fatalf("Locate expone el mismo error de integraciones:\n%v\n%v", loadErr, loc.Declarations.IntegrationsErr)
	}
}

func TestWithoutIntegrationsBlockNothingChanges(t *testing.T) {
	installCatalog(t)
	cfg, baseErr, declarations := parseDeclarations(t, "")
	if baseErr != nil || declarations.Integrations != nil || declarations.IntegrationsErr != nil {
		t.Fatalf("sin bloque no hay declaración: %v %+v %v", baseErr, declarations.Integrations, declarations.IntegrationsErr)
	}
	if cfg.Integrations != nil {
		t.Fatalf("sin bloque WorkspaceConfig.Integrations es nil: %+v", cfg.Integrations)
	}
}

func TestLocateLoadsValidIntegrations(t *testing.T) {
	installCatalog(t)
	ws := workspacetest.Embedded(t, ".")
	content := declarationsBase + "integrations:\n  fake:\n    project: marco\n"
	if err := os.WriteFile(filepath.Join(ws.Specs, "axiom.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	loc, err := Locate(ws.Specs)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if loc.Declarations.IntegrationsErr != nil || loc.Declarations.Integrations == nil || len(loc.Declarations.Integrations.Active()) != 1 {
		t.Fatalf("Declarations = %+v", loc.Declarations)
	}
	if errors.Is(loc.ConfigErr, os.ErrNotExist) {
		t.Fatal("la configuración debe cargar")
	}
}

// Con el catálogo por defecto (sin costura), azure-devops es el único plugin admitido (RF-ADO-02).
func TestDecodeIntegrationsAzureDevOpsWithDefaultCatalog(t *testing.T) {
	const valid = "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Fabrikam Team\n"

	t.Run("bloque válido", func(t *testing.T) {
		got, err := decodeIntegrations(integrationsNode(t, valid))
		if err != nil || got == nil || len(got.Plugins) != 1 {
			t.Fatalf("decodeIntegrations = %+v, %v", got, err)
		}
		p := got.Plugins[0]
		want := map[string]any{"organization": "contoso", "project": "Fabrikam", "team": "Fabrikam Team", "item_keys": []string{"User Story=US", "Bug=BUG"}}
		if p.ID != "azure-devops" || !p.Enabled || !reflect.DeepEqual(p.Values, want) {
			t.Fatalf("plugin = %+v", p)
		}
	})

	t.Run("clave desconocida base_url", func(t *testing.T) {
		_, err := decodeIntegrations(integrationsNode(t, valid+"    base_url: https://example.test\n"))
		wantErrorContains(t, err, "clave desconocida 'base_url'", "claves admitidas: enabled, organization, project, team, item_keys")
	})

	t.Run("pat no es una clave de configuración", func(t *testing.T) {
		_, err := decodeIntegrations(integrationsNode(t, valid+"    pat: valor-ficticio-no-secreto\n"))
		wantErrorContains(t, err, "clave desconocida 'pat'")
		if err != nil && strings.Contains(err.Error(), "valor-ficticio-no-secreto") {
			t.Fatalf("el error no debe imprimir el valor: %v", err)
		}
	})

	t.Run("falta team", func(t *testing.T) {
		_, err := decodeIntegrations(integrationsNode(t, "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n"))
		wantErrorContains(t, err, "integrations.azure-devops.team", "obligatoria")
	})

	t.Run("enabled false no exige claves", func(t *testing.T) {
		got, err := decodeIntegrations(integrationsNode(t, "integrations:\n  azure-devops:\n    enabled: false\n"))
		if err != nil || len(got.Plugins) != 1 || got.Plugins[0].Enabled {
			t.Fatalf("decodeIntegrations = %+v, %v", got, err)
		}
	})

	t.Run("jira desconocido nombra azure-devops entre los disponibles", func(t *testing.T) {
		_, err := decodeIntegrations(integrationsNode(t, "integrations:\n  jira:\n    project: p\n"))
		wantErrorContains(t, err, "plugin desconocido 'jira'", "plugins admitidos: azure-devops")
	})
}

// item_keys del plugin azure-devops (RF-ADO-02): lista de pares Tipo=PREFIJO validada al cargar.
func TestItemKeysValid(t *testing.T) {
	const head = "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Equipo\n"
	got, err := decodeIntegrations(integrationsNode(t, head+"    item_keys:\n      - Product Backlog Item=PBI\n      - Bug=BUG\n"))
	if err != nil {
		t.Fatalf("decodeIntegrations: %v", err)
	}
	want := []string{"Product Backlog Item=PBI", "Bug=BUG"}
	if !reflect.DeepEqual(got.Plugins[0].Values["item_keys"], want) {
		t.Fatalf("item_keys = %v, esperaba %v", got.Plugins[0].Values["item_keys"], want)
	}
}

func TestItemKeysDefaultApplied(t *testing.T) {
	got, err := decodeIntegrations(integrationsNode(t, "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Equipo\n"))
	if err != nil {
		t.Fatalf("decodeIntegrations: %v", err)
	}
	want := []string{"User Story=US", "Bug=BUG"}
	if !reflect.DeepEqual(got.Plugins[0].Values["item_keys"], want) {
		t.Fatalf("item_keys = %v, esperaba el valor por defecto %v", got.Plugins[0].Values["item_keys"], want)
	}
}

func TestItemKeysMalformed(t *testing.T) {
	const head = "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Equipo\n"
	_, err := decodeIntegrations(integrationsNode(t, head+"    item_keys:\n      - Bug=BUG\n      - SinIgual\n"))
	wantErrorContains(t, err, "integrations.azure-devops.item_keys", "SinIgual", "línea 8")
}

func TestItemKeysDuplicate(t *testing.T) {
	const head = "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Equipo\n"
	_, err := decodeIntegrations(integrationsNode(t, head+"    item_keys:\n      - Bug=BUG\n      - bug=B2\n"))
	wantErrorContains(t, err, "integrations.azure-devops.item_keys", "bug=B2", "repetido")
	_, err = decodeIntegrations(integrationsNode(t, head+"    item_keys:\n      - Bug=BUG\n      - Task=Bug\n"))
	wantErrorContains(t, err, "Task=Bug", "repetido")
}

func TestItemKeysPrefixInvalid(t *testing.T) {
	const head = "integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Equipo\n"
	_, err := decodeIntegrations(integrationsNode(t, head+"    item_keys:\n      - Bug=B-G\n"))
	wantErrorContains(t, err, "Bug=B-G", "prefijo")
}
