package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// projectStateFixture aísla el estado global (sin AXIOM_STATE_DIR) y monta dos
// proyectos con `.git` que comparten un mismo HOME. Deja el primero como
// directorio actual.
func projectStateFixture(t *testing.T) (rootA, rootB, home string) {
	t.Helper()
	t.Setenv("AXIOM_STATE_DIR", "")
	rootA, _, home = workspaceRootFixture(t, false, true)
	rootB, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(rootB, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(rootA)
	return rootA, rootB, home
}

func installInProject(t *testing.T, root string, extra ...string) {
	t.Helper()
	t.Chdir(root)
	args := append([]string{"--scope", "workspace", "--agent", "claude-code", "--preset", "custom"}, extra...)
	result, err := RunInstall(args, system.DetectionResult{})
	if err != nil {
		t.Fatalf("RunInstall(%s) error = %v", root, err)
	}
	if !result.Verify.Ready {
		t.Fatalf("la verificación posterior falló en %s: %#v", root, result.Verify)
	}
}

func readProjectState(t *testing.T, home, root string) state.InstallState {
	t.Helper()
	got, err := state.ProjectStore(home, root).Read()
	if err != nil {
		t.Fatalf("leer el estado de %s: %v", root, err)
	}
	return got
}

func TestInstallWorkspaceStateIsPerProject(t *testing.T) {
	rootA, rootB, home := projectStateFixture(t)

	installInProject(t, rootA, "--component", "engram", "--persona", "neutral")
	installInProject(t, rootB, "--component", "context7", "--persona", "gentleman")

	a, b := readProjectState(t, home, rootA), readProjectState(t, home, rootB)
	if a.Persona != "neutral" || len(a.Components) != 1 || a.Components[0] != model.ComponentEngram {
		t.Errorf("el estado de A = persona %q componentes %v, want neutral/[engram]", a.Persona, a.Components)
	}
	if b.Persona != "gentleman" || len(b.Components) != 1 || b.Components[0] != model.ComponentContext7 {
		t.Errorf("el estado de B = persona %q componentes %v, want gentleman/[context7]", b.Persona, b.Components)
	}
	for _, root := range []string{rootA, rootB} {
		if _, err := os.Stat(state.ProjectPath(root)); err != nil {
			t.Errorf("falta el estado del proyecto %s: %v", root, err)
		}
	}
	if _, err := state.Read(home); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("instalar en workspace no debe crear el estado global; Read() error = %v", err)
	}
}

func TestInstallWorkspaceDoesNotModifyTheGlobalState(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{"codex"}, Persona: "neutral", RDDMode: "off"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(state.Path(home))
	if err != nil {
		t.Fatal(err)
	}

	installInProject(t, rootA, "--component", "engram")

	after, err := os.ReadFile(state.Path(home))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("el estado global ha cambiado:\n%s\n---\n%s", before, after)
	}
	// La semilla global no aporta agentes: el proyecto solo registra lo que se instaló en él.
	if got := readProjectState(t, home, rootA); !slices.Equal(got.InstalledAgents, []string{"claude-code"}) {
		t.Errorf("agentes del proyecto = %v, want solo [claude-code]", got.InstalledAgents)
	}
}

func TestInstallGlobalScopeKeepsUsingTheUserState(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	t.Chdir(rootA)

	result, err := RunInstall([]string{"--scope", "global", "--agent", "claude-code", "--component", "engram", "--preset", "custom"}, system.DetectionResult{})
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	if !result.Verify.Ready {
		t.Fatalf("la verificación posterior falló: %#v", result.Verify)
	}

	got, err := state.Read(home)
	if err != nil {
		t.Fatalf("el ámbito global debe seguir escribiendo el estado de usuario: %v", err)
	}
	if len(got.InstalledAgents) != 1 || got.InstalledAgents[0] != "claude-code" {
		t.Errorf("agentes globales = %v, want [claude-code]", got.InstalledAgents)
	}
	if _, err := os.Stat(state.ProjectPath(rootA)); !os.IsNotExist(err) {
		t.Errorf("con --scope global no debe crearse estado de proyecto; stat = %v", err)
	}
}

func TestSyncWorkspaceSeedsFromGlobalAndLeavesItUntouched(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	t.Chdir(rootA)
	writeWorkspaceRootSyncState(t, home)
	global, err := state.Read(home)
	if err != nil {
		t.Fatal(err)
	}
	global.RDDMode = "on"
	global.PendingSync = true
	if err := state.Write(home, global); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(state.Path(home))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := RunSync([]string{"--scope", "workspace", "--agents", "claude-code"}); err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	after, err := os.ReadFile(state.Path(home))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("sync en workspace ha modificado el estado global:\n%s\n---\n%s", before, after)
	}
	project := readProjectState(t, home, rootA)
	if project.Persona != "neutral" || len(project.Components) != 1 || project.Components[0] != model.ComponentEngram {
		t.Errorf("el proyecto no ha heredado la selección global: %#v", project)
	}
	if project.RDDMode != "" || project.PendingSync {
		t.Errorf("el proyecto no debe guardar campos de máquina: %#v", project)
	}
	if project.InstalledBinaryVersion != AppVersion || project.ManagedAssetDigest == "" {
		t.Errorf("sync debe estampar la huella de los activos en el proyecto: %#v", project)
	}
}

func TestSyncWorkspaceDiscoversAgentsFromTheProjectState(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	t.Chdir(rootA)
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{"codex"}, SelectionConfigured: true, Persona: "neutral"}); err != nil {
		t.Fatal(err)
	}
	if err := state.ProjectStore(home, rootA).Write(state.InstallState{
		InstalledAgents:     []string{"claude-code"},
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentEngram},
		Persona:             "neutral",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	if len(result.Agents) != 1 || result.Agents[0] != model.AgentClaudeCode {
		t.Errorf("agentes sincronizados = %v, want los del proyecto [claude-code]", result.Agents)
	}
}

func TestSyncOfTwoProjectsKeepsTheirOwnSelection(t *testing.T) {
	rootA, rootB, home := projectStateFixture(t)
	installInProject(t, rootA, "--component", "engram", "--persona", "neutral")
	installInProject(t, rootB, "--component", "context7", "--persona", "gentleman")

	t.Chdir(rootA)
	if _, err := RunSync([]string{"--scope", "workspace"}); err != nil {
		t.Fatalf("RunSync(A) error = %v", err)
	}
	t.Chdir(rootB)
	if _, err := RunSync([]string{"--scope", "workspace"}); err != nil {
		t.Fatalf("RunSync(B) error = %v", err)
	}

	a, b := readProjectState(t, home, rootA), readProjectState(t, home, rootB)
	if a.Persona != "neutral" || a.Components[0] != model.ComponentEngram {
		t.Errorf("sync ha pisado el estado de A: %#v", a)
	}
	if b.Persona != "gentleman" || b.Components[0] != model.ComponentContext7 {
		t.Errorf("sync ha pisado el estado de B: %#v", b)
	}
}

func TestInstallWorkspaceExcludesProjectStateFromGit(t *testing.T) {
	rootA, _, home := projectStateFixture(t)

	installInProject(t, rootA, "--component", "engram")

	exclude, err := os.ReadFile(filepath.Join(rootA, ".git", "info", "exclude"))
	if err != nil {
		t.Fatalf("falta .git/info/exclude: %v", err)
	}
	for _, want := range []string{"/.axiom/state.json\n", "/.axiom/state.json.lock\n"} {
		if !strings.Contains(string(exclude), want) {
			t.Errorf("info/exclude no contiene %q:\n%s", want, exclude)
		}
	}
	if _, err := os.Stat(filepath.Join(rootA, ".gitignore")); !os.IsNotExist(err) {
		t.Errorf("no debe tocarse el .gitignore del proyecto; stat = %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".axiom", "state.json")); !os.IsNotExist(err) {
		t.Errorf("no debe haber estado en HOME; stat = %v", err)
	}
}

func TestEffectiveStateStorePrefersTheProjectOnceItHasState(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	t.Chdir(rootA)

	if store := EffectiveStateStore(home); store.IsProject() {
		t.Errorf("sin estado de proyecto debe usarse el global: %q", store.Path())
	}

	if err := state.ProjectStore(home, rootA).Write(state.InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	store := EffectiveStateStore(home)
	if !store.IsProject() || store.Path() != state.ProjectPath(rootA) {
		t.Errorf("con estado de proyecto debe usarse el suyo: %q", store.Path())
	}
}

func TestEffectiveStateStoreWithRootNamesTheProjectOnlyWhenItHasState(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	t.Chdir(rootA)

	if _, root := effectiveStateStoreWithRoot(home); root != "" {
		t.Errorf("sin estado de proyecto no hay raíz de proyecto: %q", root)
	}

	if err := state.ProjectStore(home, rootA).Write(state.InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	if _, root := effectiveStateStoreWithRoot(home); root != rootA {
		t.Errorf("raíz = %q, want %q", root, rootA)
	}
}

func TestManagedAssetProvenanceReadsTheProjectState(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	t.Chdir(rootA)
	if err := state.Write(home, state.InstallState{ManagedAssetDigest: "sha256:escritor-antiguo"}); err != nil {
		t.Fatal(err)
	}
	if !checkManagedReviewerAssets().stale() {
		t.Fatal("el digest global obsoleto debe reportarse como obsoleto cuando el proyecto no tiene estado")
	}
	expected, err := managedAssetDigest()
	if err != nil {
		t.Fatal(err)
	}

	if err := state.ProjectStore(home, rootA).Write(state.InstallState{ManagedAssetDigest: expected}); err != nil {
		t.Fatal(err)
	}

	if checkManagedReviewerAssets().stale() {
		t.Error("con estado de proyecto vigente, el digest obsoleto del global no debe contar")
	}
}

func TestStateStoreForFollowsTheScope(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	home, root := t.TempDir(), t.TempDir()

	if store := StateStoreFor(ScopeWorkspace, home, root); !store.IsProject() || store.Root() != root {
		t.Errorf("workspace debe dar el almacén del proyecto: %#v", store)
	}
	if store := StateStoreFor(ScopeGlobal, home, root); store.IsProject() {
		t.Errorf("global debe dar el almacén de usuario: %#v", store)
	}
	if store := StateStoreFor(ScopeWorkspace, home, ""); store.IsProject() {
		t.Errorf("sin raíz de workspace debe caer al global: %#v", store)
	}
}

// TestSyncWorkspaceWithoutProjectStateDoesNotInheritTheGlobalAgents fija que la semilla
// global solo aporta preferencias: la lista de agentes del proyecto no se deriva del
// estado global, que es la unión de todos los proyectos y de las instalaciones globales.
// Sin lista propia, sync descubre los agentes por ficheros, como con un HOME sin estado.
func TestSyncWorkspaceWithoutProjectStateDoesNotInheritTheGlobalAgents(t *testing.T) {
	rootA, _, home := projectStateFixture(t)
	t.Chdir(rootA)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := state.Write(home, state.InstallState{
		InstalledAgents:     []string{"codex"},
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentEngram},
		Persona:             "neutral",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	if len(result.Agents) != 1 || result.Agents[0] != model.AgentClaudeCode {
		t.Errorf("agentes sincronizados = %v, want los descubiertos por ficheros [claude-code] y no los del estado global", result.Agents)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex")); !os.IsNotExist(err) {
		t.Errorf("codex solo figura en el estado global y no debe sincronizarse; stat = %v", err)
	}
	project := readProjectState(t, home, rootA)
	if project.InstalledAgents != nil {
		t.Errorf("un sync no registra agentes en el proyecto: %v", project.InstalledAgents)
	}
	if project.Persona != "neutral" || len(project.Components) != 1 {
		t.Errorf("las preferencias sí vienen de la semilla: %#v", project)
	}

	// Con la lista sin registrar, el siguiente sync repite el descubrimiento por ficheros
	// en lugar de tratar el estado del proyecto como una selección vacía.
	again, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("segundo RunSync() error = %v", err)
	}
	if len(again.Agents) != 1 || again.Agents[0] != model.AgentClaudeCode {
		t.Errorf("segundo sync: agentes = %v, want [claude-code]", again.Agents)
	}
}
