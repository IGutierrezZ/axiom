package branch

import (
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const policyBase = `workspace:
  name: ejemplo
  topology: monorepo-embedded
  specs_repository: .
roles:
  core:
    name: Core
    repositories:
      - path: .
`

// loadPolicy carga policyBase más el bloque dado con el cargador real de axiom.yaml y
// resuelve la política de ramas, de modo que las pruebas cubren la unión entre lo que
// declara el fichero y lo que consume el dominio.
func loadPolicy(t *testing.T, blocks string) Policy {
	t.Helper()
	cfg, err := workspace.ParseConfig([]byte(policyBase + blocks))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	return ResolvePolicy(cfg)
}

func wantRepoPolicy(t *testing.T, what string, got RepoPolicy, integration, prefix string) {
	t.Helper()
	if got.IntegrationBranch != integration || got.BranchPrefix != prefix {
		t.Errorf("%s = {integración: %q, prefijo: %q}, se esperaba {%q, %q}", what, got.IntegrationBranch, got.BranchPrefix, integration, prefix)
	}
}

// REQ-25.3, escenario «Sin configuración se aplican los valores por defecto».
func TestPolicyWithoutConfigurationAppliesTheDefaults(t *testing.T) {
	tests := map[string]Policy{
		"sin configuración":                  ResolvePolicy(nil),
		"configuración construida a mano":    ResolvePolicy(&workspace.WorkspaceConfig{}),
		"axiom.yaml sin bloques de política": loadPolicy(t, ""),
	}
	for name, policy := range tests {
		t.Run(name, func(t *testing.T) {
			if policy.IntegrationBranch != "main" || policy.BranchPrefix != "feat" {
				t.Errorf("política = {%q, %q}, se esperaba {main, feat}", policy.IntegrationBranch, policy.BranchPrefix)
			}
			wantRepoPolicy(t, "For(.)", policy.For("."), "main", "feat")
			wantRepoPolicy(t, "For(app)", policy.For("app"), "main", "feat")
		})
	}
}

// REQ-25.3, escenario «El bloque `git:` sustituye los valores por defecto», y la
// decisión de tratar `cfg.Git == nil` (configuración a mano o `git:` inválido) con
// los defectos en lugar de fallar.
func TestPolicyGitBlockReplacesTheDefaults(t *testing.T) {
	policy := loadPolicy(t, "git:\n  integration_branch: develop\n  branch_prefix: inc\n")
	if policy.IntegrationBranch != "develop" || policy.BranchPrefix != "inc" {
		t.Errorf("política = {%q, %q}, se esperaba {develop, inc}", policy.IntegrationBranch, policy.BranchPrefix)
	}
	wantRepoPolicy(t, "For(.)", policy.For("."), "develop", "inc")

	partial := loadPolicy(t, "git:\n  integration_branch: develop\n")
	if partial.IntegrationBranch != "develop" || partial.BranchPrefix != "feat" {
		t.Errorf("un bloque parcial completa con los defectos: {%q, %q}", partial.IntegrationBranch, partial.BranchPrefix)
	}

	byHand := ResolvePolicy(&workspace.WorkspaceConfig{Git: &workspace.GitConfig{IntegrationBranch: "release"}})
	if byHand.IntegrationBranch != "release" || byHand.BranchPrefix != "feat" {
		t.Errorf("una política a mano incompleta se completa: {%q, %q}", byHand.IntegrationBranch, byHand.BranchPrefix)
	}
}

// REQ-25.3, escenario «Un repositorio con ajustes propios prevalece sobre `git:`».
func TestPolicyRepositoryEntryOverridesTheGitBlockOnlyForThatRepository(t *testing.T) {
	policy := loadPolicy(t, `git:
  integration_branch: main
repositories:
  - path: repo-backend
    integration_branch: release
  - path: repo-prefijo
    branch_prefix: fix
`)

	wantRepoPolicy(t, "repo-backend", policy.For("repo-backend"), "release", "feat")
	wantRepoPolicy(t, "repo-frontend", policy.For("repo-frontend"), "main", "feat")
	// Una entrada que solo declara el prefijo hereda la rama de integración de `git:`.
	wantRepoPolicy(t, "repo-prefijo", policy.For("repo-prefijo"), "main", "fix")
	// El valor del workspace no cambia por las entradas.
	if policy.IntegrationBranch != "main" || policy.BranchPrefix != "feat" {
		t.Errorf("política del workspace = {%q, %q}, se esperaba {main, feat}", policy.IntegrationBranch, policy.BranchPrefix)
	}
}

// Policy.For recibe la ruta relativa con barras y la compara normalizada: la entrada
// escrita con barras invertidas, `./` o separadores repetidos casa igual.
func TestPolicyForNormalisesTheRelativePath(t *testing.T) {
	policy := loadPolicy(t, `repositories:
  - path: services\api
    integration_branch: api-main
  - path: ./web/
    integration_branch: web-main
  - path: .
    integration_branch: root-main
`)

	tests := []struct {
		lookup string
		want   string
	}{
		{"services/api", "api-main"},
		{`services\api`, "api-main"},
		{"./services//api/", "api-main"},
		{"web", "web-main"},
		{".", "root-main"},
		{"", "root-main"},
		{"services", "main"},
		{"services/api/src", "main"},
	}
	for _, tc := range tests {
		t.Run(tc.lookup, func(t *testing.T) {
			wantRepoPolicy(t, "For("+tc.lookup+")", policy.For(tc.lookup), tc.want, "feat")
		})
	}
}

// ResolvePolicy no comparte estado con la configuración: modificar la política
// devuelta no cambia lo cargado.
func TestResolvePolicyDoesNotAliasTheConfiguration(t *testing.T) {
	cfg := &workspace.WorkspaceConfig{
		Git:          &workspace.GitConfig{IntegrationBranch: "main", BranchPrefix: "feat"},
		Repositories: []workspace.RepositoryPolicy{{Path: "a", IntegrationBranch: "x"}},
	}
	policy := ResolvePolicy(cfg)
	policy.PerRepository["a"] = RepoPolicy{IntegrationBranch: "cambiado"}
	policy.IntegrationBranch = "cambiado"

	if cfg.Git.IntegrationBranch != "main" || cfg.Repositories[0].IntegrationBranch != "x" {
		t.Errorf("la configuración cambió: %+v %+v", cfg.Git, cfg.Repositories)
	}
}

func configLocation(gitErr, repositoriesErr, configErr error) workspace.Location {
	return workspace.Location{
		Master:       `C:\proyecto con espacios`,
		ConfigPath:   `C:\proyecto con espacios\axiom.yaml`,
		ConfigErr:    configErr,
		Declarations: workspace.Declarations{GitErr: gitErr, RepositoriesErr: repositoriesErr},
	}
}

// REQ-25.3: un valor inválido o una configuración ilegible en `axiom.yaml` se
// rechaza con `config_invalid` (nunca con `invalid_name`, que es de la invocación) y
// la continuación ejecutable cita la carpeta maestra.
func TestPolicyConfigRefusalIsConfigInvalidAndNamesTheCause(t *testing.T) {
	tests := []struct {
		name      string
		location  workspace.Location
		wantCause []string
	}{
		{"bloque git inválido", configLocation(errors.New("git.branch_prefix: valor «a b»"), nil, nil), []string{"git.branch_prefix"}},
		{"bloque repositories inválido", configLocation(nil, errors.New("repositories[0].path: sale de la carpeta maestra"), nil), []string{"repositories[0].path"}},
		{"configuración ilegible", configLocation(nil, nil, errors.New("error al parsear axiom.yaml")), []string{"error al parsear axiom.yaml"}},
		{"varias causas", configLocation(errors.New("causa-git\ncausa-git-2"), errors.New("causa-repos"), nil), []string{"causa-git", "causa-git-2", "causa-repos"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			refusal := ConfigRefusal(tc.location)
			if refusal == nil {
				t.Fatal("se esperaba un rechazo")
			}
			if refusal.Code != CodeConfigInvalid || refusal.Code == CodeInvalidName {
				t.Errorf("code = %s, se esperaba config_invalid", refusal.Code)
			}
			for _, want := range append(tc.wantCause, tc.location.ConfigPath) {
				if !strings.Contains(refusal.Message, want) {
					t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
				}
			}
			if strings.Contains(refusal.Message, "\n") {
				t.Errorf("el mensaje es de una sola línea: %q", refusal.Message)
			}
			for _, want := range []string{"axiom workspace validate --path", pathquote.Quote(tc.location.Master)} {
				if !strings.Contains(refusal.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
				}
			}
			if refusal.Details["path"] != tc.location.ConfigPath {
				t.Errorf("details = %v, se esperaba la ruta %s", refusal.Details, tc.location.ConfigPath)
			}
		})
	}
}

func TestPolicyConfigRefusalIsNilWhenTheConfigurationIsUsable(t *testing.T) {
	if refusal := ConfigRefusal(configLocation(nil, nil, nil)); refusal != nil {
		t.Errorf("una configuración válida no se rechaza: %+v", refusal)
	}
	// Sin axiom.yaml no hay nada que rechazar: rigen los valores por defecto.
	if refusal := ConfigRefusal(workspace.Location{Master: `C:\sin config`}); refusal != nil {
		t.Errorf("sin axiom.yaml no se rechaza: %+v", refusal)
	}
	// Los errores de otras declaraciones (baseline, technical_context) no impiden planificar ramas.
	other := configLocation(nil, nil, nil)
	other.Declarations.BaselineErr = errors.New("baseline inválida")
	other.Declarations.TechnicalContextErr = errors.New("contexto inválido")
	if refusal := ConfigRefusal(other); refusal != nil {
		t.Errorf("un baseline inválido no es asunto de la política de ramas: %+v", refusal)
	}
}

// La resolución del planning home que falla (puntero roto, configuración que no se
// puede cargar) también impide planificar: conserva la causa y la continuación
// tipadas de Locate.
func TestPolicyLocateRefusalKeepsTheTypedCauseAndContinuation(t *testing.T) {
	located := &workspace.LocateError{
		Kind:         workspace.LocateConfigUnloadable,
		Path:         `C:\p\axiom.yaml`,
		Cause:        "el puntero no se puede cargar",
		Continuation: "Ejecuta `axiom workspace validate --path \"C:\\p\"` y corrige la configuración",
	}
	refusal := LocateRefusal(located)

	if refusal.Code != CodeConfigInvalid || refusal.Message != located.Cause || refusal.Continuation != located.Continuation {
		t.Errorf("rechazo = %+v; se esperaba config_invalid con la causa y la continuación de Locate", refusal)
	}
	if refusal.Details["path"] != located.Path {
		t.Errorf("details = %v, se esperaba la ruta %s", refusal.Details, located.Path)
	}

	generic := LocateRefusal(errors.New("fallo inesperado"))
	if generic.Code != CodeConfigInvalid || !strings.Contains(generic.Message, "fallo inesperado") {
		t.Errorf("un error que no es de Locate conserva su texto: %+v", generic)
	}
}
