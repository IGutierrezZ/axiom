package hub

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// requireViolation comprueba que err es un *InitValidationError con alguna
// violación que contiene want, y devuelve el error tipado.
func requireViolation(t *testing.T, err error, want string) *InitValidationError {
	t.Helper()
	var validation *InitValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("se esperaba *InitValidationError con %q, se obtuvo %v", want, err)
	}
	for _, v := range validation.Violations {
		if strings.Contains(v, want) {
			return validation
		}
	}
	t.Fatalf("ninguna violación contiene %q: %q", want, validation.Violations)
	return nil
}

// requireNothingWritten falla si path existe: el alta rechazada no debe haber
// creado ni el directorio destino.
func requireNothingWritten(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("el alta rechazada no debía escribir nada y existe %s (err=%v)", path, err)
	}
}

func TestInitRejectsInvalidConfigurationsWithoutWriting(t *testing.T) {
	cases := []struct {
		name  string
		roles []RoleInput
		want  string
	}{
		{
			name: "clave de rol repetida",
			roles: []RoleInput{
				{Key: "backend", Name: "Backend", Repositories: []string{"."}},
				{Key: "backend", Name: "Otro", Repositories: []string{"."}},
			},
			want: `la clave de rol "backend" está repetida`,
		},
		{
			name: "claves que solo difieren en mayúsculas",
			roles: []RoleInput{
				{Key: "Api", Name: "A", Repositories: []string{"."}},
				{Key: "api", Name: "B", Repositories: []string{"."}},
			},
			want: "solo se distinguen por las mayúsculas",
		},
		{
			name:  "rol sin clave ni nombre",
			roles: []RoleInput{{Repositories: []string{"."}}},
			want:  "sin clave ni nombre",
		},
		{
			name:  "repositorio con ruta absoluta",
			roles: []RoleInput{{Key: "core", Name: "Core", Repositories: []string{absoluteSample()}}},
			want:  "es una ruta absoluta",
		},
		{
			name:  "repositorio que sale del proyecto",
			roles: []RoleInput{{Key: "core", Name: "Core", Repositories: []string{"../fuera"}}},
			want:  "sale de la carpeta del proyecto",
		},
		{
			name:  "repositorio que sale del proyecto tras normalizar",
			roles: []RoleInput{{Key: "core", Name: "Core", Repositories: []string{"src/../../fuera"}}},
			want:  "sale de la carpeta del proyecto",
		},
		{
			name:  "repositorio vacío",
			roles: []RoleInput{{Key: "core", Name: "Core", Repositories: []string{" "}}},
			want:  "ruta vacía",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "nuevo")
			_, err := NewInitializer(nil, nil).Init(InitOptions{Path: target, Name: "Prueba", Roles: tc.roles})
			requireViolation(t, err, tc.want)
			requireNothingWritten(t, target)

			// ValidateInit da el mismo veredicto sin escribir.
			requireViolation(t, NewInitializer(nil, nil).ValidateInit(InitOptions{Path: target, Name: "Prueba", Roles: tc.roles}), tc.want)
			requireNothingWritten(t, target)
		})
	}
}

// absoluteSample es una ruta absoluta válida en el sistema operativo de la prueba.
func absoluteSample() string {
	abs, err := filepath.Abs(filepath.Join(os.TempDir(), "otro-repo"))
	if err != nil {
		return "/otro-repo"
	}
	return abs
}

func TestInitReportsEveryViolationTogether(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nuevo")
	err := NewInitializer(nil, nil).ValidateInit(InitOptions{
		Path: target,
		Name: "Prueba",
		// Los roles especializados solo se admiten en multirepo.
		Topology: "multirepo",
		Roles: []RoleInput{
			{Key: "a", Name: "A", Repositories: []string{"../fuera"}},
			{Key: "a", Name: "A2", Repositories: []string{"."}},
			{Key: "b", Name: "B", Repositories: []string{absoluteSample()}},
		},
	})
	var validation *InitValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("se esperaba *InitValidationError, se obtuvo %v", err)
	}
	if len(validation.Violations) != 3 {
		t.Fatalf("se esperaban 3 violaciones (repetida, fuera, absoluta), hay %d: %q", len(validation.Violations), validation.Violations)
	}
	for _, line := range validation.Violations {
		if strings.Contains(line, "no existe") {
			t.Errorf("una ruta inadmisible no debe arrastrar además un error de existencia: %q", line)
		}
	}
	requireNothingWritten(t, target)
	if msg := validation.Error(); !strings.Contains(msg, "\n- ") {
		t.Errorf("Error() debe listar las violaciones una por línea: %q", msg)
	}
}

func TestInitWritesRoleKeysSafely(t *testing.T) {
	for _, key := range []string{"web:app", "qa team", "yes", "123", "con#almohadilla", `con"comillas`, "api.v2"} {
		t.Run(key, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "nuevo")
			res, err := NewInitializer(nil, nil).Init(InitOptions{
				Path: target,
				Name: "Prueba",
				// Las claves de rol libres solo se admiten en multirepo.
				Topology: "multirepo",
				Roles:    []RoleInput{{Key: key, Name: "Rol", Repositories: []string{"."}}},
			})
			if err != nil {
				t.Fatalf("la clave %q debe escribirse de forma segura: %v", key, err)
			}
			cfg, err := workspace.LoadConfig(res.ConfigPath)
			if err != nil {
				t.Fatalf("el axiom.yaml generado con la clave %q no carga: %v", key, err)
			}
			if _, ok := cfg.Roles[key]; !ok || len(cfg.Roles) != 1 {
				t.Errorf("el rol %q no se conserva intacto: %v", key, cfg.Roles)
			}
		})
	}
}

func TestInitAcceptsValidConfigurationOnNewProject(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nuevo")
	roles := []RoleInput{
		{Key: "backend", Name: "Backend", Repositories: []string{"server", "./libs/../shared"}, Tech: []string{"go"}},
		{Key: "docs", Name: "Docs", Repositories: []string{"docs"}, NonBlocking: true},
	}
	if err := NewInitializer(nil, nil).ValidateInit(InitOptions{Path: target, Name: "Prueba", Topology: "multirepo", Roles: roles}); err != nil {
		t.Fatalf("una configuración válida no debe rechazarse: %v", err)
	}
	requireNothingWritten(t, target)

	res, err := NewInitializer(nil, nil).Init(InitOptions{Path: target, Name: "Prueba", Topology: "multirepo", Roles: roles})
	if err != nil {
		t.Fatalf("Init con una configuración válida falló: %v", err)
	}
	if _, err := workspace.LoadConfig(res.ConfigPath); err != nil {
		t.Fatalf("el axiom.yaml generado no carga: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "openspec", "changes")); err != nil {
		t.Errorf("Init debía crear openspec/changes: %v", err)
	}
}

func TestInitDefaultAndKnowledgeProfilesStillPassValidation(t *testing.T) {
	for _, profile := range []string{"", "knowledge", "spec-only"} {
		target := filepath.Join(t.TempDir(), "nuevo")
		if _, err := NewInitializer(nil, nil).Init(InitOptions{Path: target, Name: "Prueba", Profile: profile}); err != nil {
			t.Errorf("perfil %q: el alta por defecto no debe rechazarse: %v", profile, err)
		}
	}
}

func TestValidateInitIgnoresGeneratedConfigWhenExistingOneIsKept(t *testing.T) {
	target := t.TempDir()
	existing := "workspace:\n  name: \"Previo\"\n  topology: \"monorepo-embedded\"\n  specs_repository: \"openspec\"\nroles:\n  core:\n    name: \"Core\"\n    repositories:\n      - path: \".\"\n"
	if err := os.WriteFile(filepath.Join(target, "axiom.yaml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := []RoleInput{{Key: "x", Name: "X", Repositories: []string{"../fuera"}}}

	// Sin --force Init conserva el axiom.yaml y no escribe nada generado.
	if err := NewInitializer(nil, nil).ValidateInit(InitOptions{Path: target, Roles: bad}); err != nil {
		t.Fatalf("con un axiom.yaml conservado no hay configuración generada que validar: %v", err)
	}
	if _, err := NewInitializer(nil, nil).Init(InitOptions{Path: target, Roles: bad}); err != nil {
		t.Fatalf("Init sobre un axiom.yaml existente debe conservarlo: %v", err)
	}

	// Con --force y sin declaraciones que perder se regenera, así que se valida.
	requireViolation(t, NewInitializer(nil, nil).ValidateInit(InitOptions{Path: target, Roles: bad, Force: true}), "sale de la carpeta del proyecto")
	got, _ := os.ReadFile(filepath.Join(target, "axiom.yaml"))
	if string(got) != existing {
		t.Errorf("el rechazo no debe tocar el axiom.yaml existente:\n%s", got)
	}
}

func TestPolicyViolationsUseTheMultiroleVocabulary(t *testing.T) {
	cfg := &workspace.WorkspaceConfig{Roles: map[string]workspace.RoleConfig{
		"a": {Name: "A"},
		"b": {Name: "B", GatePolicy: "blocking"},
		"c": {Name: "C", GatePolicy: "Deferred"},
		"d": {Name: "D", GatePolicy: "optional"},
		"e": {Name: "E", GatePolicy: "advisory"},
	}}
	got := policyViolations(cfg)
	if len(got) != 1 || !strings.Contains(got[0], `"advisory"`) || !strings.Contains(got[0], `"e"`) {
		t.Fatalf("solo advisory debe rechazarse, se obtuvo %q", got)
	}
}

func TestYAMLKeyQuotesOnlyWhenNeeded(t *testing.T) {
	cases := map[string]string{
		"backend":  "backend",
		"api.v2":   "api.v2",
		"web:app":  `"web:app"`,
		"yes":      `"yes"`,
		"123":      `"123"`,
		"qa team":  `"qa team"`,
		"con#hash": `"con#hash"`,
	}
	for key, want := range cases {
		if got := yamlKey(key); got != want {
			t.Errorf("yamlKey(%q) = %s, se esperaba %s", key, got, want)
		}
	}
}

// writeDetectedProjects deja en dir un openspec/config.yaml con proyectos de
// los que el detector infiere los roles core, web y qa.
func writeDetectedProjects(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "openspec", "changes"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "schema: \"1.0\"\nprojects:\n  - path: \"src/App.Web\"\n    stack: \".NET Blazor\"\n  - path: \"src/App.Core\"\n    stack: \".NET Class Library\"\n  - path: \"tests/App.UnitTests\"\n    stack: \".NET xUnit\"\n"
	if err := os.WriteFile(filepath.Join(dir, "openspec", "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
}

func roleKeysOf(t *testing.T, configPath string) []string {
	t.Helper()
	cfg, err := workspace.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("el axiom.yaml generado no carga: %v", err)
	}
	keys := make([]string, 0, len(cfg.Roles))
	for key := range cfg.Roles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestPlanConfigMonorepoIgnoresDetectedRoles(t *testing.T) {
	target := t.TempDir()
	writeDetectedProjects(t, target)
	ini := NewInitializer(nil, NewDetector())
	opts := InitOptions{Path: target, Name: "Mono"}

	if err := ini.ValidateInit(opts); err != nil {
		t.Fatalf("el alta monorepo con roles detectados no debe rechazarse: %v", err)
	}
	res, err := ini.Init(opts)
	if err != nil {
		t.Fatalf("Init monorepo falló: %v", err)
	}
	if got := roleKeysOf(t, res.ConfigPath); len(got) != 1 || got[0] != "fullstack" {
		t.Errorf("monorepo-embedded solo debe escribir fullstack, hay %v", got)
	}
}

func TestPlanConfigMultirepoAppliesDetectedRoles(t *testing.T) {
	target := t.TempDir()
	writeDetectedProjects(t, target)
	res, err := NewInitializer(nil, NewDetector()).Init(InitOptions{Path: target, Name: "Multi", Topology: "multirepo"})
	if err != nil {
		t.Fatalf("Init multirepo falló: %v", err)
	}
	want := []string{"core", "qa", "web"}
	if got := roleKeysOf(t, res.ConfigPath); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("multirepo debe aplicar los roles detectados %v, hay %v", want, got)
	}
}

func TestValidateInitRejectsExplicitRolesInMonorepo(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nuevo")
	err := NewInitializer(nil, nil).ValidateInit(InitOptions{
		Path:  target,
		Name:  "Prueba",
		Roles: []RoleInput{{Key: "core", Name: "Core", Repositories: []string{"."}}},
	})
	requireViolation(t, err, "solo admite el rol 'fullstack'")
	requireNothingWritten(t, target)

	// El rol fullstack explícito sí es conforme.
	ok := InitOptions{Path: target, Name: "Prueba", Roles: []RoleInput{{Key: "fullstack", Name: "Full", Repositories: []string{"."}}}}
	if err := NewInitializer(nil, nil).ValidateInit(ok); err != nil {
		t.Errorf("fullstack explícito en monorepo debe aceptarse: %v", err)
	}
}

func TestPlanConfigKnowledgeProfileForcesMultirepoKnowledge(t *testing.T) {
	for _, profile := range []string{"knowledge", "spec-only"} {
		target := t.TempDir()
		writeDetectedProjects(t, target)
		res, err := NewInitializer(nil, NewDetector()).Init(InitOptions{Path: target, Name: "Know", Profile: profile})
		if err != nil {
			t.Fatalf("perfil %q: Init falló: %v", profile, err)
		}
		cfg, err := workspace.LoadConfig(res.ConfigPath)
		if err != nil {
			t.Fatalf("perfil %q: el axiom.yaml no carga: %v", profile, err)
		}
		if cfg.Workspace.Topology != workspace.TopologyMultirepo {
			t.Errorf("perfil %q: topología %q, se esperaba multirepo", profile, cfg.Workspace.Topology)
		}
		if _, ok := cfg.Roles["knowledge"]; !ok || len(cfg.Roles) != 1 {
			t.Errorf("perfil %q: roles %v, se esperaba solo knowledge", profile, cfg.Roles)
		}
	}
}
