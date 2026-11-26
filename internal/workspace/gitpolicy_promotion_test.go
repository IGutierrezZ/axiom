package workspace

import (
	"strings"
	"testing"
)

// RF-TOP-09: `main_branch` y `promotion` se declaran en `git:` y se conservan tal
// como se escribieron; sin ellas, la política resuelta las deja vacías (el defecto
// `branch` lo aplica el motor, que así conserva el origen).
func TestGitPromotionKeysAreDeclaredWithoutDefaults(t *testing.T) {
	tests := []struct {
		name     string
		block    string
		declared GitConfig
	}{
		{"solo la rama principal", "git:\n  main_branch: master\n", GitConfig{MainBranch: "master"}},
		{"solo la promoción", "git:\n  promotion: release\n", GitConfig{Promotion: "release"}},
		{"promoción ask", "git:\n  promotion: ask\n", GitConfig{Promotion: "ask"}},
		{"las cuatro claves", "git:\n  integration_branch: develop\n  branch_prefix: inc\n  main_branch: master\n  promotion: branch\n",
			GitConfig{IntegrationBranch: "develop", BranchPrefix: "inc", MainBranch: "master", Promotion: "branch"}},
		{"rama principal con barra", "git:\n  main_branch: release/1.x\n", GitConfig{MainBranch: "release/1.x"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil || declarations.GitErr != nil {
				t.Fatalf("el bloque es válido: base=%v git=%v", baseErr, declarations.GitErr)
			}
			if declarations.Git == nil || *declarations.Git != tc.declared {
				t.Errorf("Declarations.Git = %+v, se esperaba %+v", declarations.Git, tc.declared)
			}
			if cfg.Git.MainBranch != tc.declared.MainBranch || cfg.Git.Promotion != tc.declared.Promotion {
				t.Errorf("la política resuelta = %+v, se esperaba main_branch=%q y promotion=%q", cfg.Git, tc.declared.MainBranch, tc.declared.Promotion)
			}
		})
	}
}

// resolveGit no rellena ni `main_branch` ni `promotion`: vacío significa una etapa
// y defecto del motor, respectivamente.
func TestResolveGitLeavesPromotionKeysWithoutDefault(t *testing.T) {
	for name, declared := range map[string]*GitConfig{
		"sin bloque":    nil,
		"bloque vacío":  {},
		"solo prefijo":  {BranchPrefix: "inc"},
		"solo rama int": {IntegrationBranch: "develop"},
	} {
		t.Run(name, func(t *testing.T) {
			resolved := resolveGit(declared)
			if resolved.MainBranch != "" || resolved.Promotion != "" {
				t.Errorf("resolveGit(%+v) = %+v, main_branch y promotion deben quedar vacíos", declared, resolved)
			}
		})
	}
}

// RF-TOP-09 (escenario «Valor inválido»): `promotion` solo admite tres valores,
// distingue mayúsculas y su error enumera los tres.
func TestGitPromotionInvalidValuesAreRejected(t *testing.T) {
	tests := []struct {
		name  string
		block string
		wants []string
	}{
		{"mayúscula inicial", "git:\n  promotion: Branch\n", []string{"git.promotion", "'Branch'", "'branch'", "'release'", "'ask'"}},
		{"valor desconocido", "git:\n  promotion: trunk\n", []string{"git.promotion", "'trunk'", "'branch'", "'release'", "'ask'"}},
		{"valor vacío", "git:\n  promotion: \"\"\n", []string{"git.promotion", "''"}},
		{"valor nulo", "git:\n  promotion:\n", []string{"git.promotion", "nulo"}},
		{"rama principal inválida", "git:\n  main_branch: \"a b\"\n", []string{"git.main_branch", "'a b'", "espacio"}},
		{"rama principal vacía", "git:\n  main_branch: \"\"\n", []string{"git.main_branch", "no puede estar vacío"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.GitErr, tc.wants...)
			if declarations.Git != nil || cfg.Git != nil {
				t.Errorf("un bloque inválido no se proyecta: %+v / %+v", declarations.Git, cfg.Git)
			}
		})
	}
}

// Los errores acumulados del bloque nombran las cuatro claves admitidas.
func TestGitUnknownKeyErrorNamesTheFourKeys(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "git:\n  promotion: Branch\n  main_branch: \"a b\"\n  worktree_mode: worktree\n")
	wantErrorContains(t, declarations.GitErr,
		"git.promotion", "'Branch'",
		"git.main_branch", "'a b'",
		"worktree_mode", "integration_branch, branch_prefix, main_branch, promotion")
}

// RF-TOP-10: una entrada de `repositories:` admite `promotion` (vacío hereda) pero
// no `main_branch`, que solo existe en `git:`.
func TestRepositoriesEntriesCarryTheirOwnPromotion(t *testing.T) {
	block := "repositories:\n  - path: a\n    promotion: ask\n  - path: b\n    promotion: release\n    branch_prefix: ui\n  - path: c\n"
	cfg, baseErr, declarations := parseDeclarations(t, block)
	if baseErr != nil || declarations.RepositoriesErr != nil {
		t.Fatalf("el bloque es válido: base=%v repositories=%v", baseErr, declarations.RepositoriesErr)
	}
	want := []RepositoryPolicy{
		{Path: "a", Promotion: "ask"},
		{Path: "b", BranchPrefix: "ui", Promotion: "release"},
		{Path: "c"},
	}
	wantRepositories(t, cfg.Repositories, want...)
	wantRepositories(t, declarations.Repositories, want...)
}

func TestRepositoriesRejectMainBranchAndInvalidPromotion(t *testing.T) {
	tests := []struct {
		name  string
		block string
		wants []string
	}{
		{"main_branch por repositorio", "repositories:\n  - path: a\n    main_branch: master\n", []string{"main_branch", "'repositories[0]'", "clave desconocida", "path, integration_branch, branch_prefix, promotion"}},
		{"promotion con mayúscula", "repositories:\n  - path: a\n    promotion: Branch\n", []string{"repositories[0].promotion", "'Branch'", "'branch'", "'release'", "'ask'"}},
		{"promotion desconocida en la segunda entrada", "repositories:\n  - path: a\n  - path: b\n    promotion: trunk\n", []string{"repositories[1].promotion", "'trunk'"}},
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

// PromotionWarnings: avisa, sin invalidar, de una promoción que no tiene una segunda
// rama a la que promover.
func TestPromotionWarnings(t *testing.T) {
	tests := []struct {
		name  string
		block string
		count int      // número de avisos esperados
		wants []string // fragmentos que deben aparecer en los avisos
	}{
		{"sin bloques", "", 0, nil},
		{"dos etapas coherentes", "git:\n  integration_branch: develop\n  main_branch: master\n  promotion: branch\n", 0, nil},
		{"promotion sin main_branch", "git:\n  promotion: branch\n", 1, []string{"'git.promotion' (branch)", "git.main_branch"}},
		{"main_branch igual a la integración por defecto", "git:\n  main_branch: main\n  promotion: release\n", 1, []string{"'git.promotion' (release)", "es igual a 'git.integration_branch'"}},
		{"main_branch igual a la integración declarada", "git:\n  integration_branch: develop\n  main_branch: develop\n  promotion: ask\n", 1, []string{"'git.promotion' (ask)", "'develop'"}},
		{"solo main_branch no avisa", "git:\n  integration_branch: develop\n  main_branch: master\n", 0, nil},
		{"entrada con promotion sin main_branch", "repositories:\n  - path: a\n    promotion: branch\n", 1, []string{"'repositories[0].promotion' (branch)", "falta 'git.main_branch'"}},
		{"entrada que iguala la integración", "git:\n  integration_branch: develop\n  main_branch: master\nrepositories:\n  - path: a\n    integration_branch: master\n    promotion: branch\n", 1,
			[]string{"'repositories[0].promotion' (branch)", "'master'"}},
		{"entrada que hereda la integración distinta de la principal", "git:\n  integration_branch: develop\n  main_branch: master\nrepositories:\n  - path: a\n    promotion: ask\n", 0, nil},
		{"git y entrada a la vez", "git:\n  promotion: release\nrepositories:\n  - path: a\n    promotion: branch\n  - path: b\n", 2,
			[]string{"'git.promotion' (release)", "'repositories[0].promotion' (branch)"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(declarationsBase + tc.block))
			if err != nil {
				t.Fatalf("la configuración es válida: %v", err)
			}
			warnings := PromotionWarnings(cfg)
			if len(warnings) != tc.count {
				t.Fatalf("avisos = %q, se esperaban %d", warnings, tc.count)
			}
			joined := strings.Join(warnings, "\n")
			for _, want := range tc.wants {
				if !strings.Contains(joined, want) {
					t.Errorf("los avisos %q deben contener %q", warnings, want)
				}
			}
		})
	}

	if got := PromotionWarnings(nil); got != nil {
		t.Errorf("PromotionWarnings(nil) = %q, se esperaba nil", got)
	}
	if got := PromotionWarnings(&WorkspaceConfig{}); got != nil {
		t.Errorf("sin política resuelta no hay avisos: %q", got)
	}
}
