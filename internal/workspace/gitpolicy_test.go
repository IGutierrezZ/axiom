package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// wantGit comprueba la política de ramas resuelta de una configuración cargada.
func wantGit(t *testing.T, cfg *WorkspaceConfig, integration, prefix string) {
	t.Helper()
	if cfg == nil || cfg.Git == nil {
		t.Fatalf("la configuración debe llevar la política de ramas resuelta: %+v", cfg)
	}
	if cfg.Git.IntegrationBranch != integration || cfg.Git.BranchPrefix != prefix {
		t.Errorf("política = {integración: %q, prefijo: %q}, se esperaba {%q, %q}",
			cfg.Git.IntegrationBranch, cfg.Git.BranchPrefix, integration, prefix)
	}
}

// REQ-25.26 (escenario «Sin bloque rigen los valores por defecto»): sin `git:`
// no hay declaración, pero la política resuelta es `main` y `feat`.
func TestGitAbsentBlockAppliesTheDefaults(t *testing.T) {
	cfg, baseErr, declarations := parseLayered([]byte(declarationsBase))
	if baseErr != nil {
		t.Fatalf("baseErr = %v, se esperaba nil", baseErr)
	}
	if declarations.Git != nil || declarations.GitErr != nil {
		t.Fatalf("sin bloque no hay declaración ni error de git: %+v", declarations)
	}
	wantGit(t, cfg, "main", "feat")

	viaParse, err := ParseConfig([]byte(declarationsBase))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	wantGit(t, viaParse, "main", "feat")
}

// REQ-25.26 (escenario «Bloque parcial completa con los valores por defecto»):
// la declaración conserva lo escrito y la política resuelta completa el resto.
func TestGitPartialBlockIsCompletedWithTheDefaults(t *testing.T) {
	tests := []struct {
		name        string
		block       string
		integration string
		prefix      string
		declared    GitConfig
	}{
		{"solo la rama de integración", "git:\n  integration_branch: develop\n", "develop", "feat", GitConfig{IntegrationBranch: "develop"}},
		{"solo el prefijo", "git:\n  branch_prefix: inc\n", "main", "inc", GitConfig{BranchPrefix: "inc"}},
		{"las dos claves", "git:\n  integration_branch: release/1.x\n  branch_prefix: equipo/inc\n", "release/1.x", "equipo/inc", GitConfig{IntegrationBranch: "release/1.x", BranchPrefix: "equipo/inc"}},
		{"en estilo de flujo", "git: {integration_branch: develop}\n", "develop", "feat", GitConfig{IntegrationBranch: "develop"}},
		{"valores entre comillas", "git:\n  integration_branch: \"develop\"\n  branch_prefix: 'inc'\n", "develop", "inc", GitConfig{IntegrationBranch: "develop", BranchPrefix: "inc"}},
		{"valor que YAML lee como número", "git:\n  branch_prefix: 2024\n", "main", "2024", GitConfig{BranchPrefix: "2024"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil || declarations.GitErr != nil {
				t.Fatalf("el bloque es válido: base=%v git=%v", baseErr, declarations.GitErr)
			}
			wantGit(t, cfg, tc.integration, tc.prefix)
			if declarations.Git == nil || *declarations.Git != tc.declared {
				t.Errorf("Declarations.Git = %+v, se esperaba lo declarado %+v", declarations.Git, tc.declared)
			}
		})
	}
}

// REQ-25.26 (escenario «Valor inválido»): un valor inválido es un error de carga
// que nombra la clave, el valor y la regla incumplida.
func TestGitInvalidValuesAreLoadErrorsNamingTheKeyAndTheValue(t *testing.T) {
	tests := []struct {
		name  string
		block string
		wants []string
	}{
		{"prefijo con barras finales", "git:\n  branch_prefix: \"inc//\"\n", []string{"git.branch_prefix", "'inc//'", "«/»"}},
		{"prefijo con barra inicial", "git:\n  branch_prefix: /inc\n", []string{"git.branch_prefix", "'/inc'", "«/»"}},
		{"prefijo con espacio y barras (diseño §7.1)", "git:\n  branch_prefix: \"mi prefijo//\"\n", []string{"git.branch_prefix", "'mi prefijo//'"}},
		{"prefijo vacío", "git:\n  branch_prefix: \"\"\n", []string{"git.branch_prefix", "''", "no puede estar vacío"}},
		{"prefijo con dos puntos seguidos", "git:\n  branch_prefix: a..b\n", []string{"git.branch_prefix", "'a..b'", "«..»"}},
		{"rama con espacio y dos puntos (diseño §7.1)", "git:\n  integration_branch: \"a b:c\"\n", []string{"git.integration_branch", "'a b:c'", "espacio"}},
		{"rama vacía", "git:\n  integration_branch: \"\"\n", []string{"git.integration_branch", "''", "no puede estar vacío"}},
		{"rama que termina en .lock", "git:\n  integration_branch: main.lock\n", []string{"git.integration_branch", "'main.lock'", "«.lock»"}},
		{"rama con circunflejo", "git:\n  integration_branch: \"a^b\"\n", []string{"git.integration_branch", "'a^b'", "«^»"}},
		{"valor nulo", "git:\n  integration_branch:\n", []string{"git.integration_branch", "nulo"}},
		{"valor que es una lista", "git:\n  branch_prefix: [inc, feat]\n", []string{"git.branch_prefix", "cadena"}},
		{"valor que es un mapeo", "git:\n  integration_branch: {a: b}\n", []string{"git.integration_branch", "cadena"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.GitErr, tc.wants...)
			if declarations.Git != nil {
				t.Errorf("un bloque inválido no se proyecta: Declarations.Git = %+v", declarations.Git)
			}
			if cfg == nil || cfg.Git != nil {
				t.Errorf("un bloque inválido deja la política sin resolver (nil), no %+v", cfg)
			}

			_, err := ParseConfig([]byte(declarationsBase + tc.block))
			wantErrorContains(t, err, tc.wants...)
		})
	}
}

// REQ-25.26 (escenario «Clave desconocida en `git:`»): el modo de trabajo y la
// raíz de los worktrees viven en el hub personal, no en axiom.yaml.
func TestGitUnknownKeysAreRejected(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "git:\n  worktree_mode: worktree\n")
	wantErrorContains(t, declarations.GitErr, "worktree_mode", "'git'", "integration_branch", "branch_prefix")

	_, _, declarations = parseDeclarations(t, "git:\n  integration_branch: main\n  worktree_root: /tmp/x\n")
	wantErrorContains(t, declarations.GitErr, "worktree_root")
	if declarations.Git != nil {
		t.Errorf("una clave desconocida invalida el bloque entero: %+v", declarations.Git)
	}
}

// Los errores del bloque se acumulan: una sola ejecución de `workspace validate`
// los muestra todos (REQ-24.17).
func TestGitErrorsAreAccumulated(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "git:\n  integration_branch: \"a b\"\n  branch_prefix: \"inc//\"\n  worktree_mode: worktree\n  extra: 1\n")
	wantErrorContains(t, declarations.GitErr,
		"git.integration_branch", "'a b'",
		"git.branch_prefix", "'inc//'",
		"worktree_mode", "extra")
}

func TestGitBlockMustBeAMapping(t *testing.T) {
	for name, block := range map[string]string{
		"escalar": "git: main\n",
		"nulo":    "git:\n",
		"lista":   "git: [main]\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.GitErr, "'git'", "mapeo", "integration_branch")
			if declarations.Git != nil || cfg.Git != nil {
				t.Errorf("un bloque que no es un mapeo no se proyecta: %+v / %+v", declarations.Git, cfg.Git)
			}
		})
	}
}

func TestGitDuplicatedKeyIsRejected(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "git:\n  branch_prefix: inc\n  branch_prefix: feat\n")
	wantErrorContains(t, declarations.GitErr, "branch_prefix", "duplicada")
}

// Un `git:` inválido no oculta los errores de la base ni los de los demás
// bloques, y estos no ocultan los suyos (canales propios, D-03).
func TestGitErrorsShareTheLoadWithTheOtherBlocksAndTheBase(t *testing.T) {
	t.Run("con otro bloque inválido", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + "git:\n  branch_prefix: \"inc//\"\nbaseline:\n  documents: []\n"))
		wantErrorContains(t, err, "git.branch_prefix", "baseline.documents")
	})

	t.Run("con la base inválida", func(t *testing.T) {
		data := []byte(locateInvalidBase + "git:\n  branch_prefix: \"inc//\"\n")
		cfg, baseErr, declarations := parseLayered(data)
		if cfg != nil || baseErr == nil {
			t.Fatalf("la base es inválida: cfg=%+v err=%v", cfg, baseErr)
		}
		wantErrorContains(t, declarations.GitErr, "git.branch_prefix")

		_, err := ParseConfig(data)
		wantErrorContains(t, err, "roles", "git.branch_prefix")
	})

	t.Run("un bloque válido no se ve afectado por otro inválido", func(t *testing.T) {
		cfg, baseErr, declarations := parseDeclarations(t, "git:\n  branch_prefix: inc\nbaseline:\n  documents: []\n")
		if baseErr != nil || declarations.GitErr != nil || declarations.BaselineErr == nil {
			t.Fatalf("base=%v git=%v baseline=%v", baseErr, declarations.GitErr, declarations.BaselineErr)
		}
		wantGit(t, cfg, "main", "inc")
	})
}

// REQ-25.26 (compatibilidad con el binario anterior): el binario que no conoce
// `git:` decodifica la base ignorando esa clave, aunque el bloque sea inválido.
func TestGitBlockIsToleratedByThePreviousBinary(t *testing.T) {
	type previousConfig struct {
		Workspace  WorkspaceSection      `yaml:"workspace"`
		Roles      map[string]RoleConfig `yaml:"roles"`
		Governance GovernanceConfig      `yaml:"governance,omitempty"`
	}
	for name, block := range map[string]string{
		"válido":                "git:\n  integration_branch: develop\n  branch_prefix: inc\n",
		"con clave desconocida": "git:\n  worktree_mode: worktree\n",
		"con valores inválidos": "git:\n  integration_branch: \"a b:c\"\n  branch_prefix: \"inc//\"\n",
		"que no es un mapeo":    "git: main\n",
		"nulo":                  "git:\n",
	} {
		t.Run(name, func(t *testing.T) {
			var previous previousConfig
			if err := yaml.Unmarshal([]byte(declarationsBase+block), &previous); err != nil {
				t.Fatalf("el binario anterior debía tolerar el bloque: %v", err)
			}
			cfg := WorkspaceConfig{Workspace: previous.Workspace, Roles: previous.Roles, Governance: previous.Governance}
			if err := validateBaseConfig(&cfg); err != nil {
				t.Fatalf("la configuración base sigue siendo válida para el binario anterior: %v", err)
			}
			if cfg.Workspace.Name != "ejemplo" {
				t.Errorf("workspace.name = %q, se esperaba ejemplo", cfg.Workspace.Name)
			}
		})
	}

	// La decodificación base actual tampoco proyecta `git:` en la configuración:
	// el campo lleva `yaml:"-"` y solo parseLayered lo rellena.
	var current WorkspaceConfig
	if err := yaml.Unmarshal([]byte(declarationsBase+"git:\n  branch_prefix: inc\n"), &current); err != nil {
		t.Fatalf("decodificación base: %v", err)
	}
	if current.Git != nil {
		t.Errorf("la decodificación base no debe rellenar Git: %+v", current.Git)
	}
}

// REQ-25.26 y D-03: `Locate` nunca rechaza por un `git:` inválido; solo fallan
// la carga (`LoadConfig`, `ParseConfig`) y `workspace validate`.
func TestLocateIsNeverRejectedByAnInvalidGitBlock(t *testing.T) {
	const invalidGit = "git:\n  branch_prefix: \"inc//\"\n"

	t.Run("axiom.yaml directo", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+invalidGit)
		loc := mustLocate(t, ws.Master)
		if loc.Config == nil || loc.ConfigErr != nil || len(loc.Notes) != 0 {
			t.Fatalf("la base válida carga y el bloque inválido queda en su canal: %+v", loc)
		}
		if loc.Declarations.GitErr == nil || loc.Declarations.Git != nil || loc.Config.Git != nil {
			t.Fatalf("el error de git va a Declarations.GitErr y no se proyecta: %+v", loc.Declarations)
		}
	})

	t.Run("puntero", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateCanonicalConfig+invalidGit)
		loc := mustLocate(t, ws.Master)
		if loc.Master != ws.Master || loc.WorkspaceRoot != ws.Specs || loc.PlanningHome != ws.Home {
			t.Fatalf("la resolución no cambia con un git: inválido: %+v", loc)
		}
		if loc.Declarations.GitErr == nil || len(loc.Config.Roles) != 2 || len(loc.Notes) != 0 {
			t.Fatalf("GitErr = %v, roles = %d, notas = %v", loc.Declarations.GitErr, len(loc.Config.Roles), loc.Notes)
		}
	})

	t.Run("bloque válido", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+"git:\n  integration_branch: develop\n")
		loc := mustLocate(t, ws.Master)
		if loc.Declarations.GitErr != nil || loc.Declarations.Git == nil || loc.Declarations.Git.IntegrationBranch != "develop" {
			t.Fatalf("Declarations = %+v", loc.Declarations)
		}
		wantGit(t, loc.Config, "develop", "feat")
	})

	t.Run("sin bloque", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		loc := mustLocate(t, ws.Master)
		if loc.Declarations.Git != nil || loc.Declarations.GitErr != nil {
			t.Fatalf("sin bloque Declarations conserva el valor cero: %+v", loc.Declarations)
		}
		wantGit(t, loc.Config, "main", "feat")
	})
}

// LoadConfig es la carga de `workspace validate`: un `git:` inválido la hace
// fallar nombrando la clave y el valor, y uno válido la completa con los
// defectos (REQ-1.2 y REQ-25.26).
func TestLoadConfigAppliesAndValidatesTheGitBlock(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "axiom.yaml")

	if err := os.WriteFile(file, []byte(declarationsBase+"git:\n  branch_prefix: inc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("LoadConfig con un bloque válido: %v", err)
	}
	wantGit(t, cfg, "main", "inc")

	if err := os.WriteFile(file, []byte(declarationsBase+"git:\n  branch_prefix: \"inc//\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadConfig(file)
	wantErrorContains(t, err, "git.branch_prefix", "'inc//'")
}

// REQ-25.26: el bloque `git:` documentado en axiom.example.yaml, una vez
// descomentado, es válido (la plantilla no puede desviarse del esquema).
func TestExampleGitBlockIsValidOnceUncommented(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "axiom.example.yaml"))
	if err != nil {
		t.Fatalf("no se pudo leer axiom.example.yaml: %v", err)
	}
	var block []string
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if len(block) == 0 {
			if line == "# git:" {
				block = append(block, "git:")
			}
			continue
		}
		if !strings.HasPrefix(line, "#   ") {
			break // fin del bloque comentado
		}
		block = append(block, strings.TrimPrefix(line, "# "))
	}
	if len(block) < 3 {
		t.Fatalf("axiom.example.yaml debe documentar un bloque `# git:` con sus claves, y se halló %q", block)
	}
	cfg, baseErr, declarations := parseDeclarations(t, strings.Join(block, "\n")+"\n")
	if baseErr != nil || declarations.GitErr != nil {
		t.Fatalf("el bloque de la plantilla no es válido: base=%v git=%v\n%s", baseErr, declarations.GitErr, strings.Join(block, "\n"))
	}
	if declarations.Git == nil || declarations.Git.IntegrationBranch == "" || declarations.Git.BranchPrefix == "" {
		t.Errorf("la plantilla debe documentar las dos claves: %+v", declarations.Git)
	}
	if cfg.Git == nil {
		t.Errorf("la política debe quedar resuelta: %+v", cfg)
	}
}
