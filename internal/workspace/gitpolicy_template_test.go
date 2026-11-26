package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// multirepoBase es una configuración multirepo con un repositorio de specs
// relativo (`especificacion`) y un rol con un repositorio de código.
const multirepoBase = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: especificacion
roles:
  backend:
    name: Backend
    repositories:
      - path: backend-api
`

// RF-TOP-10, prov:TOP:1: las tres claves `branch_template` se aceptan y se
// proyectan tal como se declararon.
func TestBranchTemplateKeysAreAcceptedAndProjected(t *testing.T) {
	doc := multirepoBase +
		"git:\n  branch_template: \"{prefix}/{change}\"\n" +
		"repositories:\n  - path: backend-api\n    branch_template: \"feature/TASK-{task_ref}\"\n"
	doc = strings.Replace(doc, "    name: Backend\n", "    name: Backend\n    branch_template: \"feature/{role}-{task_ref}\"\n", 1)

	cfg, err := ParseConfig([]byte(doc))
	if err != nil {
		t.Fatalf("las tres plantillas son válidas: %v", err)
	}
	if cfg.Git == nil || cfg.Git.BranchTemplate != "{prefix}/{change}" {
		t.Errorf("git.branch_template no se proyecta: %+v", cfg.Git)
	}
	if len(cfg.Repositories) != 1 || cfg.Repositories[0].BranchTemplate != "feature/TASK-{task_ref}" {
		t.Errorf("repositories[0].branch_template no se proyecta: %+v", cfg.Repositories)
	}
	if got := cfg.Roles["backend"].BranchTemplate; got != "feature/{role}-{task_ref}" {
		t.Errorf("roles.backend.branch_template = %q", got)
	}
}

// Las claves desconocidas siguen rechazándose y el mensaje enumera `branch_template`.
func TestBranchTemplateIsListedAmongTheAdmittedKeys(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "git:\n  worktree_mode: worktree\n")
	wantErrorContains(t, declarations.GitErr, "worktree_mode", "branch_template")

	_, _, declarations = parseDeclarations(t, "repositories:\n  - path: a\n    nope: 1\n")
	wantErrorContains(t, declarations.RepositoriesErr, "nope", "branch_template")
}

// Los errores se acumulan y cada uno nombra la ruta de su clave.
func TestBranchTemplateErrorsAreAccumulatedAndNameTheKeyPath(t *testing.T) {
	doc := declarationsBase +
		"git:\n  branch_template: \"{prefix}/{kind}\"\n" +
		"repositories:\n  - path: a\n    branch_template: \"{desconocida}\"\n  - path: b\n    branch_template: \"-{ref}\"\n"
	doc = strings.Replace(doc, "    name: Core\n", "    name: Core\n    branch_template: \"{change\"\n", 1)

	_, baseErr, declarations := parseLayered([]byte(doc))
	if baseErr != nil {
		t.Fatalf("la base es válida: %v", baseErr)
	}
	wantErrorContains(t, declarations.GitErr, "git.branch_template", "{prefix}/{kind}")
	wantErrorContains(t, declarations.RepositoriesErr, "repositories[0].branch_template", "{desconocida}", "repositories[1].branch_template", "-{ref}")
	wantErrorContains(t, declarations.BranchTemplatesErr, "roles.core.branch_template", "{change")

	_, err := ParseConfig([]byte(doc))
	wantErrorContains(t, err, "git.branch_template", "repositories[0].branch_template", "roles.core.branch_template")
}

// prov:TOP:1: una plantilla sin variable distintiva es `config_invalid`.
func TestBranchTemplateWithoutDistinguishingVariableIsRejected(t *testing.T) {
	for _, block := range []string{
		"git:\n  branch_template: \"{prefix}/{kind}\"\n",
		"git:\n  branch_template: \"constante\"\n",
		"repositories:\n  - path: a\n    branch_template: \"{prefix}/{role}\"\n",
	} {
		_, err := ParseConfig([]byte(declarationsBase + block))
		wantErrorContains(t, err, "branch_template")
	}
}

// Una plantilla válida con cada variable distintiva se acepta.
func TestBranchTemplateAcceptsEveryDistinguishingVariable(t *testing.T) {
	for _, raw := range []string{"{prefix}/{change}", "{kind}/{slug}", "{kind}/{ref}", "feature/{task_ref}"} {
		// `{task_ref}` no puede estar en la plantilla que aloja `openspec/`: se
		// declara en una entrada de código de un multirepo.
		doc := multirepoBase + "repositories:\n  - path: backend-api\n    branch_template: \"" + raw + "\"\n"
		if _, err := ParseConfig([]byte(doc)); err != nil {
			t.Errorf("la plantilla %q es válida: %v", raw, err)
		}
	}
}

// RF-BRA-05 (S11, G6): `{role}` y `{task_ref}` no se admiten en la plantilla de la
// rama que aloja `openspec/`.
func TestBranchTemplateOpenspecBranchRule(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		wantErr []string // vacío: debe cargar sin error
	}{
		{
			"monorepo: git con {task_ref} y sin entrada del repositorio",
			declarationsBase + "git:\n  branch_template: \"feature/{task_ref}\"\n",
			[]string{"git.branch_template", "task_ref"},
		},
		{
			"monorepo: git con {role} y sin entrada del repositorio",
			declarationsBase + "git:\n  branch_template: \"{role}/{change}\"\n",
			[]string{"git.branch_template", "role"},
		},
		{
			"monorepo: entrada `.` con {task_ref}",
			declarationsBase + "repositories:\n  - path: ./\n    branch_template: \"feature/{task_ref}\"\n",
			[]string{"repositories[0].branch_template", "task_ref"},
		},
		{
			"monorepo: la entrada `.` declara y sustituye a git",
			declarationsBase + "git:\n  branch_template: \"feature/{task_ref}\"\nrepositories:\n  - path: .\n    branch_template: \"{prefix}/{change}\"\n",
			nil,
		},
		{
			"monorepo: entrada `.` solo con prefijo no sustituye a git",
			declarationsBase + "git:\n  branch_template: \"feature/{task_ref}\"\nrepositories:\n  - path: .\n    branch_prefix: ui\n",
			[]string{"git.branch_template", "task_ref"},
		},
		{
			"monorepo: otra entrada con {task_ref} no aloja openspec",
			declarationsBase + "repositories:\n  - path: otro\n    branch_template: \"feature/{task_ref}\"\n",
			nil,
		},
		{
			"multirepo: git con {task_ref} y specs sin plantilla propia",
			multirepoBase + "git:\n  branch_template: \"feature/{task_ref}\"\n",
			[]string{"git.branch_template", "task_ref"},
		},
		{
			"multirepo: git con {task_ref} válido si specs declara la suya",
			multirepoBase + "git:\n  branch_template: \"feature/{task_ref}\"\nrepositories:\n  - path: especificacion\n    branch_template: \"{prefix}/{change}\"\n",
			nil,
		},
		{
			"multirepo: specs solo con branch_prefix no sustituye a git",
			multirepoBase + "git:\n  branch_template: \"feature/{task_ref}\"\nrepositories:\n  - path: especificacion\n    branch_prefix: spec\n",
			[]string{"git.branch_template", "task_ref"},
		},
		{
			"multirepo: specs con {role} en su propia plantilla",
			multirepoBase + "repositories:\n  - path: especificacion\n    branch_template: \"{role}/{change}\"\n",
			[]string{"repositories[0].branch_template", "role"},
		},
		{
			"multirepo: openspec como último segmento identifica su padre",
			strings.Replace(multirepoBase, "specs_repository: especificacion", "specs_repository: especificacion/openspec", 1) +
				"repositories:\n  - path: especificacion\n    branch_template: \"feature/{task_ref}\"\n",
			[]string{"repositories[0].branch_template", "task_ref"},
		},
		{
			"multirepo: specs_repository absoluto toma git",
			strings.Replace(multirepoBase, "specs_repository: especificacion", "specs_repository: /srv/especificacion", 1) +
				"git:\n  branch_template: \"feature/{task_ref}\"\nrepositories:\n  - path: especificacion\n    branch_template: \"{prefix}/{change}\"\n",
			[]string{"git.branch_template", "task_ref"},
		},
		{
			"multirepo: repositorio de código con {task_ref} es válido",
			multirepoBase + "repositories:\n  - path: backend-api\n    branch_template: \"feature/TASK-{task_ref}\"\n",
			nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, baseErr, declarations := parseLayered([]byte(tc.doc))
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			if len(tc.wantErr) == 0 {
				if declarations.BranchTemplatesErr != nil || declarations.GitErr != nil || declarations.RepositoriesErr != nil {
					t.Fatalf("debía cargar sin error: %v / %v / %v", declarations.BranchTemplatesErr, declarations.GitErr, declarations.RepositoriesErr)
				}
				if _, err := ParseConfig([]byte(tc.doc)); err != nil {
					t.Fatalf("ParseConfig: %v", err)
				}
				return
			}
			wantErrorContains(t, declarations.BranchTemplatesErr, tc.wantErr...)
			_, err := ParseConfig([]byte(tc.doc))
			wantErrorContains(t, err, tc.wantErr...)
		})
	}
}

// `roles.<id>.branch_template` con {role} o {task_ref} es válida: las ramas de rol
// las usan.
func TestBranchTemplateRoleTemplateMayUseRoleAndTaskRef(t *testing.T) {
	doc := strings.Replace(multirepoBase, "    name: Backend\n", "    name: Backend\n    branch_template: \"feature/{role}-{task_ref}\"\n", 1)
	if _, err := ParseConfig([]byte(doc)); err != nil {
		t.Fatalf("la plantilla de rol es válida: %v", err)
	}
}

// Una plantilla de rol sin variable distintiva se rechaza con la ruta del rol.
func TestBranchTemplateRoleTemplateWithoutDistinguishingVariableIsRejected(t *testing.T) {
	doc := strings.Replace(multirepoBase, "    name: Backend\n", "    name: Backend\n    branch_template: \"{prefix}/{role}\"\n", 1)
	_, err := ParseConfig([]byte(doc))
	wantErrorContains(t, err, "roles.backend.branch_template", "{prefix}/{role}")
}

// Una plantilla de rol rota no invalida la base: `sdd status` y otras órdenes
// siguen leyendo la configuración base.
func TestBranchTemplateRoleErrorDoesNotInvalidateTheBase(t *testing.T) {
	doc := strings.Replace(multirepoBase, "    name: Backend\n", "    name: Backend\n    branch_template: \"{x}\"\n", 1)
	cfg, baseErr, declarations := parseLayered([]byte(doc))
	if baseErr != nil || cfg == nil {
		t.Fatalf("la base sigue siendo legible: cfg=%v err=%v", cfg, baseErr)
	}
	if declarations.BranchTemplatesErr == nil {
		t.Fatal("se esperaba BranchTemplatesErr")
	}
}

// Aviso de `workspace validate`: la plantilla de rol no tiene efecto en
// monorepo-embedded.
func TestBranchTemplateWarningsRoleTemplateWithoutEffectInMonorepo(t *testing.T) {
	doc := strings.Replace(declarationsBase, "    name: Core\n", "    name: Core\n    branch_template: \"feature/{role}-{task_ref}\"\n", 1)
	cfg, err := ParseConfig([]byte(doc))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	warnings := BranchTemplateWarnings(cfg)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "roles.core.branch_template") || !strings.Contains(warnings[0], "monorepo-embedded") {
		t.Fatalf("avisos = %v", warnings)
	}

	multi := strings.Replace(multirepoBase, "    name: Backend\n", "    name: Backend\n    branch_template: \"feature/{role}-{task_ref}\"\n", 1)
	cfgMulti, err := ParseConfig([]byte(multi))
	if err != nil {
		t.Fatalf("ParseConfig multirepo: %v", err)
	}
	if got := BranchTemplateWarnings(cfgMulti); len(got) != 0 {
		t.Errorf("en multirepo la plantilla de rol tiene efecto: %v", got)
	}
	if got := BranchTemplateWarnings(nil); got != nil {
		t.Errorf("sin configuración no hay avisos: %v", got)
	}
}

// Sin plantillas, nada cambia: ni errores, ni avisos, ni campos rellenos.
func TestBranchTemplateAbsentLeavesEverythingUntouched(t *testing.T) {
	cfg, baseErr, declarations := parseDeclarations(t, "git:\n  branch_prefix: inc\n")
	if baseErr != nil || declarations.BranchTemplatesErr != nil {
		t.Fatalf("sin plantilla no hay error: %v / %v", baseErr, declarations.BranchTemplatesErr)
	}
	if cfg.Git.BranchTemplate != "" {
		t.Errorf("sin plantilla la política queda vacía: %q", cfg.Git.BranchTemplate)
	}
	if got := BranchTemplateWarnings(cfg); len(got) != 0 {
		t.Errorf("avisos = %v", got)
	}
}

// ValidateBranchTemplate reúne la sintaxis, la variable distintiva y la muestra.
func TestValidateBranchTemplate(t *testing.T) {
	for _, raw := range []string{"{prefix}/{change}", "feature/{task_ref}", "{kind}/{ref}-{slug}"} {
		if err := ValidateBranchTemplate(raw); err != nil {
			t.Errorf("%q debe ser válida: %v", raw, err)
		}
	}
	for _, raw := range []string{"", "{prefix}/{kind}", "{change", "{nope}", "-{ref}", "{prefix}/", "a b/{change}"} {
		if err := ValidateBranchTemplate(raw); err == nil {
			t.Errorf("%q debe rechazarse", raw)
		}
	}
}

// axiom.example.yaml documenta las tres claves y, descomentadas, validan (S12).
func TestExampleBranchTemplateDocumentationIsValidOnceUncommented(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "axiom.example.yaml"))
	if err != nil {
		t.Fatalf("no se pudo leer axiom.example.yaml: %v", err)
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")

	if !strings.Contains(text, "#   branch_template: \"{prefix}/{change}\"") {
		t.Error("el bloque `git:` del ejemplo debe documentar `branch_template: \"{prefix}/{change}\"`")
	}
	if !strings.Contains(text, "#     branch_template: \"feature/TASK-{task_ref}\"") {
		t.Error("el bloque `repositories:` del ejemplo debe documentar `branch_template: \"feature/TASK-{task_ref}\"`")
	}
	const roleLine = "    # # branch_template: \"feature/{role}-{task_ref}\""
	if !strings.Contains(text, roleLine) {
		t.Fatalf("el rol `backend` del ejemplo debe llevar la línea doblemente comentada %q", roleLine)
	}
	for _, variable := range []string{"{prefix}", "{kind}", "{change}", "{slug}", "{role}", "{ref}", "{task_ref}"} {
		if !strings.Contains(text, variable) {
			t.Errorf("el ejemplo debe explicar la variable %s", variable)
		}
	}

	// Descomenta el rol y añade los bloques `git:` y `repositories:` ya descomentados.
	live := strings.Replace(text, roleLine, "    branch_template: \"feature/{role}-{task_ref}\"", 1)
	live += "\n" + uncommentedExampleBlock(t, "git") + uncommentedExampleBlock(t, "repositories")
	if _, err := ParseConfig([]byte(live)); err != nil {
		t.Fatalf("el ejemplo con todas las plantillas descomentadas no es válido: %v", err)
	}
}
