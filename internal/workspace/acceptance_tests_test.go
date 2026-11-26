package workspace

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// RF-TOP-19: un bloque válido se proyecta tal cual, con los entornos en orden.
func TestAcceptanceTestsDecodesAValidBlock(t *testing.T) {
	cfg, baseErr, declarations := parseDeclarations(t, "acceptance_tests:\n  enabled: true\n  environments: [local, dev, pre]\n")
	if baseErr != nil || declarations.AcceptanceTestsErr != nil {
		t.Fatalf("el bloque debe ser válido: base=%v acceptance_tests=%v", baseErr, declarations.AcceptanceTestsErr)
	}
	want := &AcceptanceTestsConfig{Enabled: true, Environments: []string{"local", "dev", "pre"}}
	if !reflect.DeepEqual(declarations.AcceptanceTests, want) || !reflect.DeepEqual(cfg.AcceptanceTests, want) {
		t.Errorf("declarations=%+v cfg=%+v, se esperaba %+v", declarations.AcceptanceTests, cfg.AcceptanceTests, want)
	}
}

// RF-TOP-19: sin bloque y con `enabled: false` la función no está activa; con
// `enabled: false` los entornos son opcionales.
func TestAcceptanceTestsAbsentOrDisabledIsInactive(t *testing.T) {
	cfg, baseErr, declarations := parseDeclarations(t, "")
	if baseErr != nil || declarations.AcceptanceTests != nil || declarations.AcceptanceTestsErr != nil || cfg.AcceptanceTests != nil {
		t.Fatalf("sin bloque no hay declaración ni error: %v %+v", baseErr, declarations)
	}
	got, err := declarations.EffectiveAcceptanceTests(TopologyMultirepo)
	if err != nil || got.Enabled || len(got.Environments) != 0 {
		t.Errorf("sin bloque el accesor devuelve el valor cero: %+v, %v", got, err)
	}

	_, baseErr, declarations = parseDeclarations(t, "acceptance_tests: { enabled: false }\n")
	if baseErr != nil || declarations.AcceptanceTestsErr != nil || declarations.AcceptanceTests == nil || declarations.AcceptanceTests.Enabled {
		t.Fatalf("enabled:false sin entornos es válido y no activa: %v %+v", baseErr, declarations)
	}
	got, err = declarations.EffectiveAcceptanceTests(TopologyMonorepoEmbedded)
	if err != nil || got.Enabled {
		t.Errorf("enabled:false es válido en cualquier topología: %+v, %v", got, err)
	}
}

// RF-TOP-19: cada violación del decodificador estricto es un error con su causa.
func TestAcceptanceTestsRejectsInvalidBlocks(t *testing.T) {
	cases := []struct {
		name  string
		block string
		wants []string
	}{
		{"clave desconocida", "acceptance_tests: { enabled: true, environments: [dev], modo: x }\n", []string{"clave desconocida 'modo'", "línea", "enabled, environments"}},
		{"clave duplicada", "acceptance_tests:\n  enabled: true\n  enabled: false\n  environments: [dev]\n", []string{"clave duplicada 'enabled'"}},
		{"enabled entre comillas", "acceptance_tests: { enabled: \"sí\" }\n", []string{"acceptance_tests.enabled", "'sí'"}},
		{"enabled texto", "acceptance_tests: { enabled: si, environments: [dev] }\n", []string{"acceptance_tests.enabled"}},
		{"enabled nulo", "acceptance_tests: { enabled: , environments: [dev] }\n", []string{"acceptance_tests.enabled"}},
		{"enabled ausente", "acceptance_tests: { environments: [dev] }\n", []string{"requiere la clave 'enabled'"}},
		{"bloque nulo", "acceptance_tests:\n", []string{"no puede ser nulo"}},
		{"bloque no mapeo", "acceptance_tests: [x]\n", []string{"debe ser un mapeo"}},
		{"punto y coma", "acceptance_tests: { enabled: true, environments: [\"dev;pre\"] }\n", []string{"';'", "dev;pre"}},
		{"entorno vacío", "acceptance_tests: { enabled: true, environments: [\"\"] }\n", []string{"no puede estar vacío"}},
		{"entorno no cadena", "acceptance_tests: { enabled: true, environments: [1] }\n", []string{"cada entorno debe ser una cadena"}},
		{"activo sin entornos", "acceptance_tests: { enabled: true }\n", []string{"'acceptance_tests.environments' es obligatorio"}},
		{"activo con lista vacía", "acceptance_tests: { enabled: true, environments: [] }\n", []string{"debe ser no vacío"}},
		{"entornos duplicados", "acceptance_tests: { enabled: true, environments: [dev, pre, dev] }\n", []string{"'dev' está duplicado"}},
		{"entornos no lista", "acceptance_tests: { enabled: true, environments: dev }\n", []string{"debe ser una lista de cadenas"}},
		{"entornos nulos", "acceptance_tests: { enabled: false, environments: }\n", []string{"acceptance_tests.environments no puede ser un valor nulo"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, _, declarations := parseDeclarations(t, tc.block)
			wantErrorContains(t, declarations.AcceptanceTestsErr, tc.wants...)
			if declarations.AcceptanceTests != nil || cfg != nil && cfg.AcceptanceTests != nil {
				t.Errorf("un bloque inválido no se proyecta: %+v", declarations.AcceptanceTests)
			}
		})
	}
}

// RF-TOP-19: con enabled:false las entradas no se exigen, pero su tipo sí se valida.
func TestAcceptanceTestsDisabledStillChecksTypes(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "acceptance_tests: { enabled: false, environments: [dev, dev] }\n")
	if declarations.AcceptanceTestsErr != nil {
		t.Errorf("con enabled:false los duplicados no se exigen: %v", declarations.AcceptanceTestsErr)
	}
	_, _, declarations = parseDeclarations(t, "acceptance_tests: { enabled: false, environments: [1] }\n")
	if declarations.AcceptanceTestsErr == nil {
		t.Error("un entorno que no es cadena es un error aunque enabled sea false")
	}
}

// RF-TOP-19: el accesor nunca asume un valor con una declaración inválida o
// una topología que la regla rechaza.
func TestAcceptanceTestsAccessorNeverAssumesAValue(t *testing.T) {
	parseErr := errors.New("yaml ilegible")
	blockErr := errors.New("bloque roto")

	if _, err := (Declarations{ParseErr: parseErr}).EffectiveAcceptanceTests(TopologyMultirepo); !errors.Is(err, parseErr) {
		t.Errorf("con ParseErr el error debe envolverlo: %v", err)
	}
	if _, err := (Declarations{AcceptanceTestsErr: blockErr}).EffectiveAcceptanceTests(TopologyMultirepo); !errors.Is(err, blockErr) {
		t.Errorf("con AcceptanceTestsErr el error debe envolverlo: %v", err)
	}

	active := Declarations{AcceptanceTests: &AcceptanceTestsConfig{Enabled: true, Environments: []string{"dev"}}}
	got, err := active.EffectiveAcceptanceTests(TopologyMonorepoEmbedded)
	if err == nil || got.Enabled || len(got.Environments) != 0 {
		t.Errorf("enabled:true fuera de multirepo es error y no devuelve valor: %+v, %v", got, err)
	}
	wantErrorContains(t, err, "acceptance_tests", "monorepo-embedded")

	got, err = active.EffectiveAcceptanceTests(TopologyMultirepo)
	if err != nil || !got.Enabled || !reflect.DeepEqual(got.Environments, []string{"dev"}) {
		t.Errorf("en multirepo el accesor devuelve lo declarado: %+v, %v", got, err)
	}
	// La copia no comparte la lista con la declaración.
	got.Environments[0] = "otro"
	if active.AcceptanceTests.Environments[0] != "dev" {
		t.Error("el accesor debe devolver una copia de los entornos")
	}
}

// RF-TOP-19: Validate rechaza enabled:true fuera de multirepo, en el paso 3b, y
// admite enabled:false y la ausencia del bloque.
func TestValidateAcceptanceTestsTopology(t *testing.T) {
	baseDir := filepath.Clean("/workspace")
	newFS := func() *mockFS {
		fs := newMockFS()
		fs.addDir(baseDir)
		fs.addDir(filepath.Join(baseDir, "especificacion"))
		fs.addDir(filepath.Join(baseDir, "app"))
		return fs
	}
	role := map[string]RoleConfig{"fullstack": {Name: "Fullstack", Repositories: []RepositoryEntry{{Path: "app"}}}}
	embedded := func(acceptance *AcceptanceTestsConfig) *WorkspaceConfig {
		return &WorkspaceConfig{
			Workspace:       WorkspaceSection{Name: "App", Topology: TopologyMonorepoEmbedded, SpecsRepository: "."},
			Roles:           role,
			AcceptanceTests: acceptance,
		}
	}
	multirepo := func(acceptance *AcceptanceTestsConfig) *WorkspaceConfig {
		return &WorkspaceConfig{
			Workspace:       WorkspaceSection{Name: "App", Topology: TopologyMultirepo, SpecsRepository: "especificacion"},
			Roles:           role,
			AcceptanceTests: acceptance,
		}
	}
	enabled := &AcceptanceTestsConfig{Enabled: true, Environments: []string{"dev"}}

	t.Run("monorepo rechaza enabled true", func(t *testing.T) {
		report, err := Validate(newFS(), embedded(enabled), baseDir)
		if err != nil || report.Valid {
			t.Fatalf("se esperaba un informe inválido: %+v, %v", report, err)
		}
		assertContains(t, report.Errors, "acceptance_tests")
		assertContains(t, report.Errors, "monorepo-embedded")
	})
	t.Run("monorepo admite enabled false y la ausencia", func(t *testing.T) {
		for _, acceptance := range []*AcceptanceTestsConfig{{Enabled: false}, nil} {
			report, err := Validate(newFS(), embedded(acceptance), baseDir)
			if err != nil || !report.Valid {
				t.Errorf("se esperaba válido con %+v: %+v, %v", acceptance, report.Errors, err)
			}
		}
	})
	t.Run("multirepo admite enabled true", func(t *testing.T) {
		report, err := Validate(newFS(), multirepo(enabled), baseDir)
		if err != nil || !report.Valid {
			t.Errorf("se esperaba válido: %+v, %v", report.Errors, err)
		}
	})
}

// RF-TOP-20: Locate no rechaza por un bloque inválido y expone el error de
// RF-TOP-19 en su canal, sin proyectarlo.
func TestLocateExposesAnInvalidAcceptanceTestsBlock(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateCanonicalConfig+"acceptance_tests: { enabled: true }\n")
	loc := mustLocate(t, ws.Master)
	if loc.Declarations.AcceptanceTestsErr == nil || loc.Declarations.AcceptanceTests != nil {
		t.Fatalf("el bloque inválido debe quedar en su canal sin proyectarse: %+v", loc.Declarations)
	}
	wantErrorContains(t, loc.Declarations.AcceptanceTestsErr, "environments")
	if loc.Config == nil || len(loc.Config.Roles) == 0 {
		t.Errorf("la configuración base sigue disponible: %+v", loc.Config)
	}
	if _, err := loc.Declarations.EffectiveAcceptanceTests(TopologyMultirepo); err == nil {
		t.Error("el accesor no debe asumir un valor con el bloque inválido")
	}
}

// RF-TOP-19: la carga estricta (ParseConfig) falla con el error del bloque.
func TestAcceptanceTestsParseConfigIsStrict(t *testing.T) {
	_, err := ParseConfig([]byte(declarationsBase + "acceptance_tests: { enabled: true, environments: [dev], modo: x }\n"))
	wantErrorContains(t, err, "clave desconocida 'modo'")
}
