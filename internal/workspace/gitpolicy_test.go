package workspace

import (
	"os"
	"os/exec"
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

// wantPreviousBinaryToleratesTheBlock reproduce la carga de un binario anterior a
// los bloques `git:` y `repositories:`: decodifica la base con una estructura que
// no los conoce, así que los ignora como claves desconocidas de primer nivel y la
// configuración base sigue siendo válida.
func wantPreviousBinaryToleratesTheBlock(t *testing.T, block string) {
	t.Helper()
	type previousConfig struct {
		Workspace  WorkspaceSection      `yaml:"workspace"`
		Roles      map[string]RoleConfig `yaml:"roles"`
		Governance GovernanceConfig      `yaml:"governance,omitempty"`
	}
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
	for name, block := range map[string]string{
		"válido":                "git:\n  integration_branch: develop\n  branch_prefix: inc\n",
		"con clave desconocida": "git:\n  worktree_mode: worktree\n",
		"con valores inválidos": "git:\n  integration_branch: \"a b:c\"\n  branch_prefix: \"inc//\"\n",
		"que no es un mapeo":    "git: main\n",
		"nulo":                  "git:\n",
	} {
		t.Run(name, func(t *testing.T) {
			wantPreviousBinaryToleratesTheBlock(t, block)
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

// uncommentedExampleBlock devuelve el bloque de primer nivel `<key>:` que
// axiom.example.yaml documenta comentado (`# <key>:` seguido de líneas
// `#   ...`), ya descomentado y listo para validar.
func uncommentedExampleBlock(t *testing.T, key string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "axiom.example.yaml"))
	if err != nil {
		t.Fatalf("no se pudo leer axiom.example.yaml: %v", err)
	}
	var block []string
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if len(block) == 0 {
			if line == "# "+key+":" {
				block = append(block, key+":")
			}
			continue
		}
		if !strings.HasPrefix(line, "#   ") {
			break // fin del bloque comentado
		}
		block = append(block, strings.TrimPrefix(line, "# "))
	}
	if len(block) < 3 {
		t.Fatalf("axiom.example.yaml debe documentar un bloque `# %s:` con sus claves, y se halló %q", key, block)
	}
	return strings.Join(block, "\n") + "\n"
}

// REQ-25.26: el bloque `git:` documentado en axiom.example.yaml, una vez
// descomentado, es válido (la plantilla no puede desviarse del esquema).
func TestExampleGitBlockIsValidOnceUncommented(t *testing.T) {
	block := uncommentedExampleBlock(t, "git")
	cfg, baseErr, declarations := parseDeclarations(t, block)
	if baseErr != nil || declarations.GitErr != nil {
		t.Fatalf("el bloque de la plantilla no es válido: base=%v git=%v\n%s", baseErr, declarations.GitErr, block)
	}
	if declarations.Git == nil || declarations.Git.IntegrationBranch == "" || declarations.Git.BranchPrefix == "" {
		t.Errorf("la plantilla debe documentar las dos claves: %+v", declarations.Git)
	}
	if cfg.Git == nil {
		t.Errorf("la política debe quedar resuelta: %+v", cfg)
	}
}

// wantRepositories comprueba la lista de entradas de `repositories:` tal como se
// declararon: lo no escrito queda vacío y se hereda del bloque `git:`.
func wantRepositories(t *testing.T, got []RepositoryPolicy, want ...RepositoryPolicy) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("repositories = %+v, se esperaban %d entradas: %+v", got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("repositories[%d] = %+v, se esperaba %+v", i, got[i], want[i])
		}
	}
}

// REQ-2.1 (escenario «Los bloques opcionales ausentes no alteran la validación») y
// REQ-25.27 (escenario «Sin bloque `repositories:` todos heredan»): sin el bloque
// no hay entradas ni error, y la política de cualquier repositorio es la de `git:`.
func TestRepositoriesAbsentBlockLeavesThemEmptyAndEveryRepositoryInherits(t *testing.T) {
	for name, blocks := range map[string]string{
		"sin ningún bloque":     "",
		"con un git: declarado": "git:\n  branch_prefix: inc\n",
		"con otros bloques":     "baseline:\n  documents:\n    - domain: modelo\n      path: modelo.md\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, blocks)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			if declarations.Repositories != nil || declarations.RepositoriesErr != nil {
				t.Errorf("sin bloque no hay declaración ni error: %+v", declarations)
			}
			if cfg == nil || cfg.Repositories != nil {
				t.Fatalf("sin bloque la configuración no lleva entradas: %+v", cfg)
			}
			// El `repositories:` de cada rol no es el bloque de primer nivel.
			if len(cfg.Roles["core"].Repositories) != 1 {
				t.Errorf("los repositorios del rol siguen siendo los suyos: %+v", cfg.Roles)
			}
		})
	}

	cfg, _, _ := parseDeclarations(t, "git:\n  branch_prefix: inc\n")
	wantGit(t, cfg, "main", "inc") // la política que heredan todos los repositorios
}

// REQ-25.27 (escenario «Entrada con rama de integración propia») y S-13: una
// entrada puede llevar su `integration_branch` y su `branch_prefix`; lo que no
// escribe queda vacío y se hereda del bloque `git:`.
func TestRepositoriesEntriesCarryTheirOwnBranchSettings(t *testing.T) {
	tests := []struct {
		name  string
		block string
		want  []RepositoryPolicy
	}{
		{"rama de integración propia", "repositories:\n  - path: repo-backend\n    integration_branch: release\n",
			[]RepositoryPolicy{{Path: "repo-backend", IntegrationBranch: "release"}}},
		{"prefijo propio", "repositories:\n  - path: repo-frontend\n    branch_prefix: ui\n",
			[]RepositoryPolicy{{Path: "repo-frontend", BranchPrefix: "ui"}}},
		{"las dos claves", "repositories:\n  - path: repo-backend\n    integration_branch: release/1.x\n    branch_prefix: equipo/inc\n",
			[]RepositoryPolicy{{Path: "repo-backend", IntegrationBranch: "release/1.x", BranchPrefix: "equipo/inc"}}},
		{"solo la ruta hereda todo", "repositories:\n  - path: repo-infra\n",
			[]RepositoryPolicy{{Path: "repo-infra"}}},
		{"varias entradas en el orden declarado", "repositories:\n  - path: b\n    branch_prefix: ui\n  - path: a\n  - path: c\n    integration_branch: develop\n",
			[]RepositoryPolicy{{Path: "b", BranchPrefix: "ui"}, {Path: "a"}, {Path: "c", IntegrationBranch: "develop"}}},
		{"estilo de flujo", "repositories: [{path: a}, {path: b, branch_prefix: ui}]\n",
			[]RepositoryPolicy{{Path: "a"}, {Path: "b", BranchPrefix: "ui"}}},
		{"valores entre comillas", "repositories:\n  - path: \"repo-backend\"\n    integration_branch: 'release'\n",
			[]RepositoryPolicy{{Path: "repo-backend", IntegrationBranch: "release"}}},
		{"la propia carpeta maestra", "repositories:\n  - path: .\n    branch_prefix: inc\n",
			[]RepositoryPolicy{{Path: ".", BranchPrefix: "inc"}}},
		{"rutas anidadas con barras", "repositories:\n  - path: app/web\n  - path: app/api\n",
			[]RepositoryPolicy{{Path: "app/web"}, {Path: "app/api"}}},
		{"ruta con ./ inicial, tal como se declaró", "repositories:\n  - path: ./app\n",
			[]RepositoryPolicy{{Path: "./app"}}},
		{"lista vacía", "repositories: []\n", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil || declarations.RepositoriesErr != nil {
				t.Fatalf("el bloque es válido: base=%v repositories=%v", baseErr, declarations.RepositoriesErr)
			}
			wantRepositories(t, cfg.Repositories, tc.want...)
			wantRepositories(t, declarations.Repositories, tc.want...)

			viaParse, err := ParseConfig([]byte(declarationsBase + tc.block))
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			wantRepositories(t, viaParse.Repositories, tc.want...)
		})
	}

	t.Run("convive con git: y la entrada solo sustituye lo que declara", func(t *testing.T) {
		cfg, baseErr, declarations := parseDeclarations(t,
			"git:\n  integration_branch: main\nrepositories:\n  - path: repo-backend\n    integration_branch: release\n")
		if baseErr != nil || declarations.GitErr != nil || declarations.RepositoriesErr != nil {
			t.Fatalf("base=%v git=%v repositories=%v", baseErr, declarations.GitErr, declarations.RepositoriesErr)
		}
		wantGit(t, cfg, "main", "feat") // repo-frontend, sin entrada, usa main
		wantRepositories(t, cfg.Repositories, RepositoryPolicy{Path: "repo-backend", IntegrationBranch: "release"})
		if cfg.Repositories[0].BranchPrefix != "" {
			t.Errorf("repo-backend no declara prefijo y hereda el de git: %+v", cfg.Repositories[0])
		}
	})
}

// REQ-25.27: `path` absoluta, con unidad de disco o que sale de la carpeta maestra
// se rechaza en la carga, nombrando la entrada, la ruta y la regla incumplida.
func TestRepositoriesInvalidPathsAreLoadErrors(t *testing.T) {
	tests := []struct {
		name  string
		block string
		wants []string
	}{
		{"absoluta posix", "repositories:\n  - path: /opt/repo\n", []string{"repositories[0].path", "'/opt/repo'", "debe ser relativa a la carpeta maestra"}},
		{"con unidad de disco", "repositories:\n  - path: 'C:\\repos\\app'\n", []string{"repositories[0].path", `'C:\repos\app'`, "debe ser relativa a la carpeta maestra"}},
		{"con barra invertida inicial", "repositories:\n  - path: '\\app'\n", []string{"repositories[0].path", `'\app'`, "debe ser relativa a la carpeta maestra"}},
		{"sale de la carpeta maestra", "repositories:\n  - path: ../fuera\n", []string{"repositories[0].path", "'../fuera'", "sale de la carpeta maestra"}},
		{"sale tras normalizar", "repositories:\n  - path: app/../../fuera\n", []string{"repositories[0].path", "'app/../../fuera'", "sale de la carpeta maestra"}},
		{"sube a la carpeta padre", "repositories:\n  - path: ..\n", []string{"repositories[0].path", "'..'", "sale de la carpeta maestra"}},
		{"vacía", "repositories:\n  - path: \"\"\n", []string{"repositories[0].path", "''", "la ruta está vacía"}},
		{"solo espacios", "repositories:\n  - path: \"   \"\n", []string{"repositories[0].path", "la ruta está vacía"}},
		{"nula", "repositories:\n  - path:\n", []string{"repositories[0].path", "nulo"}},
		{"lista", "repositories:\n  - path: [a, b]\n", []string{"repositories[0].path", "cadena"}},
		{"mapeo", "repositories:\n  - path: {a: b}\n", []string{"repositories[0].path", "cadena"}},
		{"segunda entrada", "repositories:\n  - path: ok\n  - path: ../fuera\n", []string{"repositories[1].path", "'../fuera'"}},
		{"sin path", "repositories:\n  - integration_branch: develop\n", []string{"repositories[0]", "no declara 'path'"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.RepositoriesErr, tc.wants...)
			if declarations.Repositories != nil || cfg == nil || cfg.Repositories != nil {
				t.Errorf("un bloque inválido no se proyecta: %+v / %+v", declarations.Repositories, cfg)
			}
			_, err := ParseConfig([]byte(declarationsBase + tc.block))
			wantErrorContains(t, err, tc.wants...)
		})
	}
}

// REQ-25.27: los valores propios de una entrada siguen las reglas de `git:`
// (S-13) y el error nombra la entrada, la clave y el valor.
func TestRepositoriesInvalidBranchValuesNameTheEntryTheKeyAndTheValue(t *testing.T) {
	tests := []struct {
		name  string
		block string
		wants []string
	}{
		{"rama con espacio y dos puntos", "repositories:\n  - path: a\n    integration_branch: \"a b:c\"\n", []string{"repositories[0].integration_branch", "'a b:c'", "espacio"}},
		{"rama vacía", "repositories:\n  - path: a\n    integration_branch: \"\"\n", []string{"repositories[0].integration_branch", "''", "no puede estar vacío"}},
		{"prefijo con barras finales", "repositories:\n  - path: a\n    branch_prefix: \"inc//\"\n", []string{"repositories[0].branch_prefix", "'inc//'", "«/»"}},
		{"prefijo con barra inicial", "repositories:\n  - path: a\n    branch_prefix: /inc\n", []string{"repositories[0].branch_prefix", "'/inc'", "«/»"}},
		{"valor nulo", "repositories:\n  - path: a\n    branch_prefix:\n", []string{"repositories[0].branch_prefix", "nulo"}},
		{"valor que es una lista", "repositories:\n  - path: a\n    integration_branch: [x]\n", []string{"repositories[0].integration_branch", "cadena"}},
		{"en la segunda entrada", "repositories:\n  - path: a\n  - path: b\n    branch_prefix: a..b\n", []string{"repositories[1].branch_prefix", "'a..b'", "«..»"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.RepositoriesErr, tc.wants...)
			if declarations.Repositories != nil {
				t.Errorf("un bloque inválido no se proyecta: %+v", declarations.Repositories)
			}
		})
	}
}

// REQ-2.1 (escenario «Clave desconocida dentro de un bloque nuevo») y REQ-25.27:
// una clave desconocida dentro de una entrada es un error que la nombra.
func TestRepositoriesUnknownKeysAreRejected(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "repositories:\n  - path: a\n    worktree_mode: worktree\n")
	wantErrorContains(t, declarations.RepositoriesErr, "worktree_mode", "'repositories[0]'", "path", "integration_branch", "branch_prefix")

	_, _, declarations = parseDeclarations(t, "repositories:\n  - path: a\n  - path: b\n    root: /tmp/x\n")
	wantErrorContains(t, declarations.RepositoriesErr, "root", "'repositories[1]'")
	if declarations.Repositories != nil {
		t.Errorf("una clave desconocida invalida el bloque entero: %+v", declarations.Repositories)
	}
}

// REQ-25.27: dos entradas que declaran la misma ruta se rechazan nombrando las
// dos, aunque la escriban de forma distinta; que dos rutas distintas resuelvan a
// la misma raíz git lo comprueba `workspace validate` en disco.
func TestRepositoriesDuplicatedPathsAreRejectedNamingBothEntries(t *testing.T) {
	tests := []struct {
		name  string
		block string
		wants []string
	}{
		{"mismo texto", "repositories:\n  - path: app\n  - path: app\n", []string{"repositories[0]", "repositories[1]", "'app'", "misma ruta"}},
		{"con ./ inicial", "repositories:\n  - path: app\n  - path: ./app\n", []string{"repositories[0] ('app')", "repositories[1] ('./app')", "misma ruta 'app'"}},
		{"con barra final", "repositories:\n  - path: app/\n  - path: app\n", []string{"repositories[0]", "repositories[1]", "misma ruta 'app'"}},
		{"con segmentos redundantes", "repositories:\n  - path: a/../app\n  - path: app\n", []string{"repositories[0]", "repositories[1]", "misma ruta 'app'"}},
		{"con barras invertidas", "repositories:\n  - path: 'app\\web'\n  - path: app/web\n", []string{"repositories[0]", "repositories[1]", "misma ruta 'app/web'"}},
		{"distinta capitalización", "repositories:\n  - path: App\n  - path: app\n", []string{"repositories[0] ('App')", "repositories[1] ('app')", "misma ruta"}},
		{"la raíz repetida", "repositories:\n  - path: .\n  - path: ./\n", []string{"repositories[0]", "repositories[1]", "misma ruta '.'"}},
		{"la tercera repite la primera", "repositories:\n  - path: a\n  - path: b\n  - path: a\n", []string{"repositories[0]", "repositories[2]", "misma ruta 'a'"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.RepositoriesErr, tc.wants...)
			if declarations.Repositories != nil || cfg.Repositories != nil {
				t.Errorf("un bloque inválido no se proyecta: %+v / %+v", declarations.Repositories, cfg.Repositories)
			}
		})
	}

	t.Run("rutas distintas dentro de un mismo repositorio son válidas en la carga", func(t *testing.T) {
		cfg, _, declarations := parseDeclarations(t, "repositories:\n  - path: app/web\n  - path: app/api\n")
		if declarations.RepositoriesErr != nil {
			t.Fatalf("la forma es válida; la raíz git compartida la comprueba workspace validate: %v", declarations.RepositoriesErr)
		}
		wantRepositories(t, cfg.Repositories, RepositoryPolicy{Path: "app/web"}, RepositoryPolicy{Path: "app/api"})
	})
}

// Los errores del bloque se acumulan: una sola ejecución de `workspace validate`
// los muestra todos (REQ-24.17).
func TestRepositoriesErrorsAreAccumulated(t *testing.T) {
	_, _, declarations := parseDeclarations(t, `repositories:
  - path: /opt/repo
    integration_branch: "a b"
  - path: ok
    worktree_mode: worktree
    branch_prefix: "inc//"
  - integration_branch: main
  - path: dup
  - path: ./dup
`)
	wantErrorContains(t, declarations.RepositoriesErr,
		"repositories[0].path", "'/opt/repo'",
		"repositories[0].integration_branch", "'a b'",
		"worktree_mode", "repositories[1].branch_prefix", "'inc//'",
		"repositories[2]", "no declara 'path'",
		"repositories[3]", "repositories[4]", "misma ruta 'dup'")
}

func TestRepositoriesBlockMustBeAListOfMappings(t *testing.T) {
	for name, tc := range map[string]struct {
		block string
		wants []string
	}{
		"escalar":                 {"repositories: repo-backend\n", []string{"'repositories'", "lista"}},
		"nulo":                    {"repositories:\n", []string{"'repositories'", "lista"}},
		"mapeo":                   {"repositories: {path: a}\n", []string{"'repositories'", "lista"}},
		"entrada escalar":         {"repositories:\n  - repo-backend\n", []string{"repositories[0]", "mapeo"}},
		"entrada nula":            {"repositories:\n  -\n", []string{"repositories[0]", "mapeo"}},
		"entrada que es lista":    {"repositories:\n  - [a]\n", []string{"repositories[0]", "mapeo"}},
		"segunda entrada escalar": {"repositories:\n  - path: a\n  - b\n", []string{"repositories[1]", "mapeo"}},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.RepositoriesErr, tc.wants...)
			if declarations.Repositories != nil || cfg.Repositories != nil {
				t.Errorf("un bloque que no es una lista de mapeos no se proyecta: %+v / %+v", declarations.Repositories, cfg.Repositories)
			}
		})
	}
}

func TestRepositoriesDuplicatedKeyIsRejected(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "repositories:\n  - path: a\n    path: b\n")
	wantErrorContains(t, declarations.RepositoriesErr, "path", "duplicada", "'repositories[0]'")
}

// Un `repositories:` inválido no oculta los errores de la base ni los de los demás
// bloques, y estos no ocultan los suyos (canales propios, D-03).
func TestRepositoriesErrorsShareTheLoadWithTheOtherBlocksAndTheBase(t *testing.T) {
	const invalidRepositories = "repositories:\n  - path: ../fuera\n"

	t.Run("con git: inválido", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + "git:\n  branch_prefix: \"inc//\"\n" + invalidRepositories))
		wantErrorContains(t, err, "git.branch_prefix", "repositories[0].path")
	})

	t.Run("con otro bloque inválido", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + invalidRepositories + "baseline:\n  documents: []\n"))
		wantErrorContains(t, err, "repositories[0].path", "baseline.documents")
	})

	t.Run("con la base inválida", func(t *testing.T) {
		data := []byte(locateInvalidBase + invalidRepositories)
		cfg, baseErr, declarations := parseLayered(data)
		if cfg != nil || baseErr == nil {
			t.Fatalf("la base es inválida: cfg=%+v err=%v", cfg, baseErr)
		}
		wantErrorContains(t, declarations.RepositoriesErr, "repositories[0].path")

		_, err := ParseConfig(data)
		wantErrorContains(t, err, "roles", "repositories[0].path")
	})

	t.Run("un bloque válido no se ve afectado por otro inválido", func(t *testing.T) {
		cfg, baseErr, declarations := parseDeclarations(t, "repositories:\n  - path: a\n    branch_prefix: ui\ngit:\n  branch_prefix: \"inc//\"\n")
		if baseErr != nil || declarations.RepositoriesErr != nil || declarations.GitErr == nil {
			t.Fatalf("base=%v repositories=%v git=%v", baseErr, declarations.RepositoriesErr, declarations.GitErr)
		}
		wantRepositories(t, cfg.Repositories, RepositoryPolicy{Path: "a", BranchPrefix: "ui"})
		if cfg.Git != nil {
			t.Errorf("el git: inválido no se proyecta: %+v", cfg.Git)
		}
	})
}

// REQ-2.1 (escenario «Una versión anterior del binario tolera los bloques
// nuevos»): el binario que no conoce `repositories:` decodifica la base ignorando
// esa clave de primer nivel, aunque el bloque sea inválido.
func TestRepositoriesBlockIsToleratedByThePreviousBinary(t *testing.T) {
	for name, block := range map[string]string{
		"válido":                "repositories:\n  - path: repo-backend\n    integration_branch: release\n",
		"con clave desconocida": "repositories:\n  - path: a\n    worktree_mode: worktree\n",
		"con valores inválidos": "repositories:\n  - path: ../fuera\n    branch_prefix: \"inc//\"\n",
		"que no es una lista":   "repositories: repo-backend\n",
		"nulo":                  "repositories:\n",
		"junto a git:":          "git:\n  branch_prefix: inc\nrepositories:\n  - path: a\n",
	} {
		t.Run(name, func(t *testing.T) {
			wantPreviousBinaryToleratesTheBlock(t, block)
		})
	}

	// La decodificación base actual tampoco proyecta `repositories:` en la
	// configuración: el campo lleva `yaml:"-"` y solo parseLayered lo rellena.
	var current WorkspaceConfig
	if err := yaml.Unmarshal([]byte(declarationsBase+"repositories:\n  - path: a\n"), &current); err != nil {
		t.Fatalf("decodificación base: %v", err)
	}
	if current.Repositories != nil {
		t.Errorf("la decodificación base no debe rellenar Repositories: %+v", current.Repositories)
	}
}

// REQ-25.27 y D-03: `Locate` nunca rechaza por un `repositories:` inválido; solo
// fallan la carga (`LoadConfig`, `ParseConfig`) y `workspace validate`.
func TestLocateIsNeverRejectedByAnInvalidRepositoriesBlock(t *testing.T) {
	const invalidRepositories = "repositories:\n  - path: /opt/repo\n"

	t.Run("axiom.yaml directo", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+invalidRepositories)
		loc := mustLocate(t, ws.Master)
		if loc.Config == nil || loc.ConfigErr != nil || len(loc.Notes) != 0 {
			t.Fatalf("la base válida carga y el bloque inválido queda en su canal: %+v", loc)
		}
		if loc.Declarations.RepositoriesErr == nil || loc.Declarations.Repositories != nil || loc.Config.Repositories != nil {
			t.Fatalf("el error va a Declarations.RepositoriesErr y no se proyecta: %+v", loc.Declarations)
		}
	})

	t.Run("puntero", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateCanonicalConfig+invalidRepositories)
		loc := mustLocate(t, ws.Master)
		if loc.Master != ws.Master || loc.WorkspaceRoot != ws.Specs || loc.PlanningHome != ws.Home {
			t.Fatalf("la resolución no cambia con un repositories: inválido: %+v", loc)
		}
		if loc.Declarations.RepositoriesErr == nil || len(loc.Config.Roles) != 2 || len(loc.Notes) != 0 {
			t.Fatalf("RepositoriesErr = %v, roles = %d, notas = %v", loc.Declarations.RepositoriesErr, len(loc.Config.Roles), loc.Notes)
		}
	})

	t.Run("bloque válido", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+"repositories:\n  - path: .\n    branch_prefix: inc\n")
		loc := mustLocate(t, ws.Master)
		if loc.Declarations.RepositoriesErr != nil {
			t.Fatalf("Declarations = %+v", loc.Declarations)
		}
		wantRepositories(t, loc.Declarations.Repositories, RepositoryPolicy{Path: ".", BranchPrefix: "inc"})
		wantRepositories(t, loc.Config.Repositories, RepositoryPolicy{Path: ".", BranchPrefix: "inc"})
	})

	t.Run("sin bloque", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		loc := mustLocate(t, ws.Master)
		if loc.Declarations.Repositories != nil || loc.Declarations.RepositoriesErr != nil || loc.Config.Repositories != nil {
			t.Fatalf("sin bloque no hay entradas ni error: %+v", loc.Declarations)
		}
	})
}

// LoadConfig es la carga de `workspace validate`: un `repositories:` inválido la
// hace fallar nombrando la entrada y la ruta, y uno válido se carga (REQ-1.2).
func TestLoadConfigAppliesAndValidatesTheRepositoriesBlock(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "axiom.yaml")

	if err := os.WriteFile(file, []byte(declarationsBase+"repositories:\n  - path: repo-backend\n    integration_branch: release\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("LoadConfig con un bloque válido: %v", err)
	}
	wantRepositories(t, cfg.Repositories, RepositoryPolicy{Path: "repo-backend", IntegrationBranch: "release"})

	if err := os.WriteFile(file, []byte(declarationsBase+"repositories:\n  - path: ../fuera\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadConfig(file)
	wantErrorContains(t, err, "repositories[0].path", "'../fuera'", "sale de la carpeta maestra")
}

// REQ-25.27: el bloque `repositories:` documentado en axiom.example.yaml, una vez
// descomentado, es válido (la plantilla no puede desviarse del esquema).
func TestExampleRepositoriesBlockIsValidOnceUncommented(t *testing.T) {
	block := uncommentedExampleBlock(t, "repositories")
	cfg, baseErr, declarations := parseDeclarations(t, block)
	if baseErr != nil || declarations.RepositoriesErr != nil {
		t.Fatalf("el bloque de la plantilla no es válido: base=%v repositories=%v\n%s", baseErr, declarations.RepositoriesErr, block)
	}
	if len(cfg.Repositories) < 2 {
		t.Fatalf("la plantilla debe documentar al menos dos entradas: %+v", cfg.Repositories)
	}
	for _, entry := range cfg.Repositories {
		if entry.IntegrationBranch != "" || entry.BranchPrefix != "" {
			return
		}
	}
	t.Errorf("la plantilla debe mostrar una entrada con ajustes propios: %+v", cfg.Repositories)
}

// repositoriesConfig carga declarationsBase más el bloque `repositories:` dado.
func repositoriesConfig(t *testing.T, block string) *WorkspaceConfig {
	t.Helper()
	cfg, err := ParseConfig([]byte(declarationsBase + block))
	if err != nil {
		t.Fatalf("la configuración de la prueba debe ser válida: %v", err)
	}
	return cfg
}

// requireRealGit omite la prueba con -short o sin git instalado: ejecuta
// `git init` real (skill go-testing, regla 9 de tasks.md).
func requireRealGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
}

// gitRepositoryAt crea dir y lo inicializa como repositorio git real, con el
// entorno aislado de la configuración del desarrollador.
func gitRepositoryAt(t *testing.T, dir string) string {
	t.Helper()
	workspacetest.MkdirAll(t, dir)
	workspacetest.InitGit(t, dir)
	return dir
}

// outsideAnyGitRepository devuelve un directorio temporal sin ningún `.git` en sus
// ancestros, u omite la prueba si el directorio temporal del sistema está dentro
// de un repositorio git (una prueba de «no es un repositorio» no sería fiable).
func outsideAnyGitRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for current := dir; ; {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			t.Skipf("el directorio temporal %s está dentro de un repositorio git", dir)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return dir
		}
		current = parent
	}
}

// wantOneMessage comprueba que exactamente un mensaje contiene todos los
// fragmentos.
func wantOneMessage(t *testing.T, messages []string, wants ...string) {
	t.Helper()
	matches := 0
	for _, message := range messages {
		all := true
		for _, want := range wants {
			all = all && strings.Contains(message, want)
		}
		if all {
			matches++
		}
	}
	if matches != 1 {
		t.Errorf("se esperaba un mensaje con %q, y hay %d entre %q", wants, matches, messages)
	}
}

// REQ-25.27: sin entradas, o sin configuración, no hay nada que comprobar en disco.
func TestValidateRepositoryPoliciesWithoutEntriesChecksNothing(t *testing.T) {
	for name, cfg := range map[string]*WorkspaceConfig{
		"configuración nula": nil,
		"sin bloque":         repositoriesConfig(t, ""),
		"lista vacía":        repositoriesConfig(t, "repositories: []\n"),
	} {
		t.Run(name, func(t *testing.T) {
			errs, warnings := ValidateRepositoryPolicies(cfg, t.TempDir())
			if errs != nil || warnings != nil {
				t.Errorf("errs = %q, warnings = %q, se esperaba nil", errs, warnings)
			}
		})
	}
}

// REQ-25.27 (escenarios «Dos entradas de la misma raíz git» y «Ruta que no es un
// repositorio git solo avisa»): cada entrada se comprueba en disco; repositorios
// reales y distintos, o una subcarpeta de uno de ellos, no dan errores ni avisos.
func TestValidateRepositoryPoliciesAcceptsRealRepositories(t *testing.T) {
	requireRealGit(t)
	master := t.TempDir()
	gitRepositoryAt(t, filepath.Join(master, "repo-backend"))
	gitRepositoryAt(t, filepath.Join(master, "repo-frontend"))
	workspacetest.MkdirAll(t, filepath.Join(master, "repo-infra", "terraform"))
	gitRepositoryAt(t, filepath.Join(master, "repo-infra"))

	cfg := repositoriesConfig(t, "repositories:\n  - path: repo-backend\n    integration_branch: release\n  - path: repo-frontend\n    branch_prefix: ui\n  - path: repo-infra/terraform\n")
	errs, warnings := ValidateRepositoryPolicies(cfg, master)
	if len(errs) != 0 || len(warnings) != 0 {
		t.Errorf("repositorios reales y distintos: errs = %q, warnings = %q", errs, warnings)
	}
}

func TestValidateRepositoryPoliciesAcceptsTheMasterFolderItself(t *testing.T) {
	requireRealGit(t)
	master := gitRepositoryAt(t, t.TempDir())
	errs, warnings := ValidateRepositoryPolicies(repositoriesConfig(t, "repositories:\n  - path: .\n    branch_prefix: inc\n"), master)
	if len(errs) != 0 || len(warnings) != 0 {
		t.Errorf("la carpeta maestra es un repositorio: errs = %q, warnings = %q", errs, warnings)
	}
}

// REQ-25.27 (escenario «Dos entradas de la misma raíz git»): dos rutas dentro de
// un mismo repositorio son un error que nombra ambas y la raíz compartida.
func TestValidateRepositoryPoliciesRejectsTwoEntriesOfTheSameGitRoot(t *testing.T) {
	requireRealGit(t)
	master := t.TempDir()
	app := gitRepositoryAt(t, filepath.Join(master, "app"))
	workspacetest.MkdirAll(t, filepath.Join(app, "web"))
	workspacetest.MkdirAll(t, filepath.Join(app, "api"))
	workspacetest.MkdirAll(t, filepath.Join(app, "worker"))
	gitRepositoryAt(t, filepath.Join(master, "otro"))

	t.Run("dos entradas", func(t *testing.T) {
		cfg := repositoriesConfig(t, "repositories:\n  - path: app/web\n  - path: otro\n  - path: app/api\n")
		errs, warnings := ValidateRepositoryPolicies(cfg, master)
		if len(errs) != 1 || len(warnings) != 0 {
			t.Fatalf("errs = %q, warnings = %q, se esperaba un error y ningún aviso", errs, warnings)
		}
		wantOneMessage(t, errs, "'app/web'", "repositories[0]", "'app/api'", "repositories[2]", "mismo repositorio git", app)
	})

	t.Run("la tercera entrada se compara con la primera", func(t *testing.T) {
		cfg := repositoriesConfig(t, "repositories:\n  - path: app/web\n  - path: app/api\n  - path: app/worker\n")
		errs, _ := ValidateRepositoryPolicies(cfg, master)
		if len(errs) != 2 {
			t.Fatalf("errs = %q, se esperaban dos errores", errs)
		}
		wantOneMessage(t, errs, "'app/web'", "'app/api'", app)
		wantOneMessage(t, errs, "'app/web'", "'app/worker'", app)
	})

	t.Run("la raíz del repositorio y una subcarpeta suya", func(t *testing.T) {
		cfg := repositoriesConfig(t, "repositories:\n  - path: app\n  - path: app/web\n")
		errs, _ := ValidateRepositoryPolicies(cfg, master)
		wantOneMessage(t, errs, "'app'", "'app/web'", app)
	})
}

// El mismo directorio con otra grafía no esquiva la regla: las raíces se comparan
// por identidad, no por texto. Las entradas solo necesitan una entrada `.git`, así
// que no hace falta git.
func TestValidateRepositoryPoliciesComparesGitRootsByIdentity(t *testing.T) {
	t.Run("mediante un enlace simbólico", func(t *testing.T) {
		master := t.TempDir()
		app := filepath.Join(master, "app")
		workspacetest.MkdirAll(t, filepath.Join(app, ".git"))
		linkDirOrSkip(t, app, filepath.Join(master, "alias"))

		errs, _ := ValidateRepositoryPolicies(repositoriesConfig(t, "repositories:\n  - path: app\n  - path: alias\n"), master)
		wantOneMessage(t, errs, "'app'", "'alias'", "mismo repositorio git")
	})

	// En Windows y macOS `app` y `APP` son el mismo directorio y las raíces solo
	// difieren en el texto; en un sistema que distingue mayúsculas son dos
	// directorios distintos y la prueba no tiene sentido.
	t.Run("con otra capitalización en un sistema que no distingue mayúsculas", func(t *testing.T) {
		master := t.TempDir()
		app := filepath.Join(master, "app")
		workspacetest.MkdirAll(t, filepath.Join(app, ".git"))
		workspacetest.MkdirAll(t, filepath.Join(app, "api"))
		workspacetest.MkdirAll(t, filepath.Join(app, "web"))
		if _, err := os.Stat(filepath.Join(master, "APP")); err != nil {
			t.Skip("el sistema de ficheros distingue mayúsculas: `app` y `APP` son directorios distintos")
		}

		errs, _ := ValidateRepositoryPolicies(repositoriesConfig(t, "repositories:\n  - path: app/web\n  - path: APP/api\n"), master)
		wantOneMessage(t, errs, "'app/web'", "'APP/api'", "mismo repositorio git")
	})
}

// REQ-25.27 (escenario «Ruta inexistente o fuera de la carpeta maestra»): una ruta
// que no existe, o que no es un directorio, es un error por entrada nombrando la
// ruta y la regla incumplida.
func TestValidateRepositoryPoliciesReportsMissingAndNonDirectoryPaths(t *testing.T) {
	master := t.TempDir()
	workspacetest.WriteFile(t, filepath.Join(master, "fichero.txt"), "no soy un directorio\n")

	cfg := repositoriesConfig(t, "repositories:\n  - path: repo-inexistente\n  - path: fichero.txt\n")
	errs, warnings := ValidateRepositoryPolicies(cfg, master)
	if len(errs) != 2 || len(warnings) != 0 {
		t.Fatalf("errs = %q, warnings = %q, se esperaban dos errores y ningún aviso", errs, warnings)
	}
	wantOneMessage(t, errs, "'repo-inexistente'", "repositories[0].path", "no existe como directorio", filepath.Join(master, "repo-inexistente"))
	wantOneMessage(t, errs, "'fichero.txt'", "repositories[1].path", "no es un directorio")
}

// REQ-25.27 (escenario «Ruta que no es un repositorio git solo avisa»): una ruta
// que existe pero no pertenece a ningún repositorio es una advertencia, nunca un
// error, de modo que el resultado sigue siendo conforme.
func TestValidateRepositoryPoliciesWarnsAboutAPathThatIsNotAGitRepository(t *testing.T) {
	master := outsideAnyGitRepository(t)
	workspacetest.MkdirAll(t, filepath.Join(master, "docs"))

	cfg := repositoriesConfig(t, "repositories:\n  - path: docs\n  - path: .\n")
	errs, warnings := ValidateRepositoryPolicies(cfg, master)
	if len(errs) != 0 {
		t.Errorf("una ruta que no es un repositorio git no es un error: %q", errs)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %q, se esperaban dos avisos", warnings)
	}
	wantOneMessage(t, warnings, "'docs'", "repositories[0].path", "ningún repositorio git", filepath.Join(master, "docs"))
	wantOneMessage(t, warnings, "'.'", "repositories[1].path", "ningún repositorio git")
}

// Los tres resultados conviven en una sola pasada, en el orden de las entradas.
func TestValidateRepositoryPoliciesReportsEveryFindingInOneRun(t *testing.T) {
	requireRealGit(t)
	master := outsideAnyGitRepository(t)
	app := gitRepositoryAt(t, filepath.Join(master, "app"))
	workspacetest.MkdirAll(t, filepath.Join(app, "web"))
	workspacetest.MkdirAll(t, filepath.Join(app, "api"))
	workspacetest.MkdirAll(t, filepath.Join(master, "docs"))

	cfg := repositoriesConfig(t, "repositories:\n  - path: app/web\n  - path: ausente\n  - path: docs\n  - path: app/api\n")
	errs, warnings := ValidateRepositoryPolicies(cfg, master)
	if len(errs) != 2 || len(warnings) != 1 {
		t.Fatalf("errs = %q, warnings = %q, se esperaban dos errores y un aviso", errs, warnings)
	}
	wantOneMessage(t, errs[:1], "'ausente'", "no existe como directorio")
	wantOneMessage(t, errs[1:], "'app/web'", "'app/api'", "mismo repositorio git")
	wantOneMessage(t, warnings, "'docs'", "ningún repositorio git")
}

// Si la carpeta maestra no existe, Validate ya lo reporta: repetir un error por
// cada entrada solo añadiría ruido con la misma causa.
func TestValidateRepositoryPoliciesLeavesAMissingMasterFolderToValidate(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-existe")
	errs, warnings := ValidateRepositoryPolicies(repositoriesConfig(t, "repositories:\n  - path: app\n"), missing)
	if errs != nil || warnings != nil {
		t.Errorf("errs = %q, warnings = %q, se esperaba nil", errs, warnings)
	}
}

// Una configuración construida a mano con una ruta que la carga habría rechazado
// no se comprueba en disco (ni sale de la carpeta maestra): el error ya está en la
// carga.
func TestValidateRepositoryPoliciesIgnoresPathsTheLoadRejects(t *testing.T) {
	cfg := &WorkspaceConfig{Repositories: []RepositoryPolicy{{Path: "../fuera"}, {Path: "/opt/repo"}, {Path: ""}}}
	errs, warnings := ValidateRepositoryPolicies(cfg, t.TempDir())
	if errs != nil || warnings != nil {
		t.Errorf("errs = %q, warnings = %q, se esperaba nil", errs, warnings)
	}
}
