package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de los valores de las variables de plantilla y de la instantánea con plantilla (C-07-f,
// design D-06 y D-13), con git real y entorno aislado.

// stageWorkItem escribe el staging de `change start` de un cambio con el id y el perfil dados.
func stageWorkItem(t *testing.T, repo, change, id, profile string) {
	t.Helper()
	doc := `{"schema":"axiom.change-start-context/v1","change":"` + change + `","plugin":"azure-devops","id":"` + id + `","profile":"` + profile + `","attachments":{"downloaded":0,"omitted":0}}`
	writeFile(t, filepath.Join(commonDir(repo), "axiom", "change-start", "v1", change, "context", "work-item.json"), doc)
}

func gitTemplate(raw string) string {
	return "git:\n  integration_branch: main\n  branch_template: \"" + raw + "\"\n"
}

// RF-KOF-06 y D-13: con el staging de un work item `bug`, la plantilla renderiza `fix` y el id, y la
// instantánea congela la plantilla, `kind` y `ref`.
func TestSnapshotFreezesTheTemplateAndTheValuesOfTheWorkItem(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, gitTemplate("{kind}/{ref}-{slug}")))
	stageWorkItem(t, repo, "us-2525-login", "2525", "bug")

	got := snapshotOf(t, repo, "us-2525-login", "fullstack")

	if got.ChangeBranch != "fix/2525-login" {
		t.Errorf("rama del cambio = %q", got.ChangeBranch)
	}
	branches := got.Repositories[0].Branches
	if branches[0].Name != "fix/2525-login" || branches[0].Template != "{kind}/{ref}-{slug}" {
		t.Errorf("ramas = %+v", branches)
	}
	if got.TemplateValues == nil || got.TemplateValues.Kind != "fix" || got.TemplateValues.Ref != "2525" {
		t.Errorf("template_values = %+v", got.TemplateValues)
	}
	if err := got.Validate(); err != nil {
		t.Errorf("la instantánea debe ser válida: %v", err)
	}
}

// Sin plantilla, la instantánea no lleva ninguna clave nueva.
func TestSnapshotWithoutTemplateCarriesNoTemplateFields(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, ""))
	stageWorkItem(t, repo, "x", "2525", "bug")

	got := snapshotOf(t, repo, "x", "fullstack")

	if got.TemplateValues != nil || got.Repositories[0].PendingBranches != nil || got.Repositories[0].Branches[0].Template != "" {
		t.Errorf("sin plantilla no hay campos de plantilla: %+v", got)
	}
}

// D-06: `kind` se congela siempre (feature sin perfil bug) y `ref` solo si se conoce; la rama con
// `{task_ref}` no lleva nombre sino una entrada pendiente con su prefijo.
func TestSnapshotRecordsPendingBranchesWithTheirPrefix(t *testing.T) {
	t.Parallel()
	roles := map[string][]string{"backend": {"repo-backend"}}
	master, _ := newMultirepo(t, roles, "repo-backend")
	writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", roles,
		"git:\n  integration_branch: main\n  branch_prefix: inc\nrepositories:\n  - path: repo-backend\n    branch_template: \"feature/TASK-{task_ref}\"\n"))

	got := snapshotOf(t, master, "x", "backend")

	code := got.Repositories[1]
	if len(code.Branches) != 0 || len(code.PendingBranches) != 1 {
		t.Fatalf("repositorio de código = %+v, se esperaba una entrada pendiente y ninguna rama", code)
	}
	if pending := code.PendingBranches[0]; pending.Kind != "change" || pending.Role != "backend" || pending.Template != "feature/TASK-{task_ref}" || pending.Prefix != "inc" {
		t.Errorf("pendiente = %+v", pending)
	}
	if got.TemplateValues == nil || got.TemplateValues.Kind != "feature" || got.TemplateValues.Ref != "" {
		t.Errorf("template_values = %+v, se esperaba solo kind feature", got.TemplateValues)
	}
	if got.ChangeBranch != "" {
		t.Errorf("ChangeBranch = %q: ningún repositorio tiene aún la rama del cambio", got.ChangeBranch)
	}
	if err := got.Validate(); err != nil {
		t.Errorf("la instantánea debe ser válida: %v", err)
	}
}

// S5: el sellado propaga el rechazo del plan y no devuelve ninguna instantánea.
func TestSnapshotPropagatesTemplateRefusals(t *testing.T) {
	t.Parallel()
	t.Run("ref sin work item", func(t *testing.T) {
		t.Parallel()
		repo := newProject(t, configWithRoles(map[string][]string{"fullstack": {"."}}, gitTemplate("{ref}/{change}")))
		wantSnapshotRefusal(t, repo, branch.CodeBranchTemplateUnresolved, "fullstack")
	})
	t.Run("rama compartida con role", func(t *testing.T) {
		t.Parallel()
		roles := map[string][]string{"api": {"repo-backend"}, "jobs": {"repo-backend"}}
		master, _ := newMultirepo(t, roles, "repo-backend")
		writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", roles,
			"repositories:\n  - path: repo-backend\n    branch_template: \"{role}/{change}\"\n"))
		wantSnapshotRefusal(t, master, branch.CodeBranchTemplateNotApplicable, "api", "jobs")
	})
}

func wantSnapshotRefusal(t *testing.T, dir string, code branch.Code, roles ...string) {
	t.Helper()
	snapshot, err := Default().Snapshot(context.Background(), locationOf(t, dir), "x", roles)
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != code || refusal.Continuation == "" || snapshot != nil {
		t.Fatalf("snapshot = %+v, error = %v; se esperaba un rechazo %s con continuación y sin instantánea", snapshot, err, code)
	}
}

// ODD (S8): `{ref}` sale del staging; sin work item, `branch_template_unresolved`.
func TestEnsureOddRendersTheTemplateFromTheStagingOrRefuses(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, gitTemplate("{kind}/{ref}-{slug}")))
	refused := ensureOdd(t, repo, "mi-feature", EnsureRequest{})
	if refused.Code != branch.CodeBranchTemplateUnresolved {
		t.Fatalf("código = %q (%s), se esperaba branch_template_unresolved", refused.Code, refused.Message)
	}

	stageWorkItem(t, repo, "mi-feature", "88", "")
	result := ensureOdd(t, repo, "mi-feature", EnsureRequest{})
	wantReady(t, result)
	branchByName(t, onlyRepository(t, result), "feature/88-mi-feature")
}

// D-13: un cambio sellado lee `kind` y `ref` de su instantánea; sin `template_values`, `{kind}` es
// feature aunque el staging diga bug (G9).
func TestResolveTemplateValuesOfASealedChangeReadsOnlyTheSnapshot(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	stageWorkItem(t, repo, "x", "2525", "bug")
	pending := kickoff.GitSnapshotRepo{Path: ".", PendingBranches: []kickoff.GitSnapshotPendingBranch{{Kind: "change", Role: "core", Template: "{kind}/{ref}/{task_ref}", Prefix: "feat"}}}
	input := func(values *kickoff.GitTemplateValues) branch.PlanInput {
		return branch.PlanInput{
			Change: "x", Lane: branch.LaneSDD, HasKickoff: true,
			PlanningRepo: branch.RepoFacts{CommonDir: commonDir(repo)},
			Snapshot:     &kickoff.GitSnapshot{Repositories: []kickoff.GitSnapshotRepo{pending}, TemplateValues: values},
		}
	}
	location := workspace.Location{WorkspaceRoot: repo, PlanningHome: filepath.Join(repo, "openspec")}

	got := resolveTemplateValues(input(nil), location)
	if got.Kind != "feature" || !got.KindFrozen || got.Ref != "2525" || got.RefFrozen {
		t.Errorf("sin template_values = %+v: kind feature congelado; ref del staging sin congelar", got)
	}

	got = resolveTemplateValues(input(&kickoff.GitTemplateValues{Kind: "fix", Ref: "77"}), location)
	if got.Kind != "fix" || got.Ref != "77" || !got.KindFrozen || !got.RefFrozen {
		t.Errorf("con template_values = %+v", got)
	}
}

// D-13: el vínculo `work_item` de integrations.yaml manda sobre el staging y la Task del rol sale
// de su vínculo `task`, sin distinguir mayúsculas; un integrations.yaml ilegible se entrega como
// LinksErr sin rechazar.
func TestResolveTemplateValuesReadsTheLinksOfTheActivePlugin(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	stageWorkItem(t, repo, "x", "STAGING", "")
	home := filepath.Join(repo, "openspec")
	links := filepath.Join(home, "changes", "x", "integrations.yaml")
	writeFile(t, links, "schema: axiom.integrations/v1\nchange: x\nlinks:\n"+
		"  - {plugin: azure-devops, kind: work_item, id: \"300\", source: change.start, recorded_at: 2026-01-01T00:00:00Z}\n"+
		"  - {plugin: azure-devops, kind: task, id: \"1235\", role: Backend, source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n")
	location := workspace.Location{
		WorkspaceRoot: repo, PlanningHome: home,
		Declarations: workspace.Declarations{Integrations: &workspace.IntegrationsConfig{Plugins: []workspace.IntegrationPluginConfig{{ID: "azure-devops", Enabled: true}}}},
	}
	input := branch.PlanInput{
		Change: "x", Lane: branch.LaneSDD, PlanningRepo: branch.RepoFacts{CommonDir: commonDir(repo)},
		Policy: branch.ResolvePolicy(&workspace.WorkspaceConfig{Git: &workspace.GitConfig{BranchTemplate: "feature/{ref}-{task_ref}"}}),
	}

	got := resolveTemplateValues(input, location)
	if got.Ref != "300" || got.TaskRefs["backend"] != "1235" || got.LinksErr != nil {
		t.Errorf("valores = %+v", got)
	}

	if err := os.WriteFile(links, []byte("schema: ajeno\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = resolveTemplateValues(input, location)
	if got.LinksErr == nil || got.Ref != "STAGING" {
		t.Errorf("con el fichero ilegible: %+v", got)
	}
}

// Sin plantilla en juego no se lee nada.
func TestResolveTemplateValuesReadsNothingWithoutTemplates(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
	stageWorkItem(t, repo, "x", "2525", "bug")
	input := branch.PlanInput{Change: "x", Lane: branch.LaneSDD, PlanningRepo: branch.RepoFacts{CommonDir: commonDir(repo)}, Policy: branch.ResolvePolicy(nil)}

	got := resolveTemplateValues(input, workspace.Location{WorkspaceRoot: repo, PlanningHome: filepath.Join(repo, "openspec")})

	if got.Ref != "" || got.Kind != "" || got.TaskRefs != nil || got.LinksErr != nil {
		t.Errorf("valores = %+v, se esperaba el valor cero", got)
	}
}
