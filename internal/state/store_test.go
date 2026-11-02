package state

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// isolateGlobalState aparta el estado global de cualquier AXIOM_STATE_DIR del
// entorno: estas pruebas fijan el estado de usuario bajo el HOME temporal.
func isolateGlobalState(t *testing.T) {
	t.Helper()
	t.Setenv("AXIOM_STATE_DIR", "")
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return data
}

func TestProjectStorePathLivesUnderTheWorkspaceRoot(t *testing.T) {
	isolateGlobalState(t)
	home, root := t.TempDir(), t.TempDir()

	store := ProjectStore(home, root)

	if !store.IsProject() {
		t.Fatal("IsProject() = false, want true")
	}
	if want := filepath.Join(root, ".axiom", "state.json"); store.Path() != want {
		t.Errorf("Path() = %q, want %q", store.Path(), want)
	}
	if want := Path(home); GlobalStore(home).Path() != want {
		t.Errorf("global Path() = %q, want %q", GlobalStore(home).Path(), want)
	}
	if ProjectStore(home, "").IsProject() {
		t.Error("ProjectStore con raíz vacía debe ser el almacén global")
	}
}

func TestProjectStoresOfTwoProjectsDoNotOverwriteEachOther(t *testing.T) {
	isolateGlobalState(t)
	home := t.TempDir()
	rootA, rootB := t.TempDir(), t.TempDir()
	storeA, storeB := ProjectStore(home, rootA), ProjectStore(home, rootB)

	a := InstallState{InstalledAgents: []string{"claude-code"}, Persona: "neutral", ClaudeModelAssignments: map[string]string{"sdd-apply": "sonnet"}}
	a.SetSelection(model.Selection{Components: []model.ComponentID{model.ComponentEngram}})
	b := InstallState{InstalledAgents: []string{"opencode"}, Persona: "gentleman", ClaudeModelAssignments: map[string]string{"sdd-apply": "opus"}}
	b.SetSelection(model.Selection{Components: []model.ComponentID{model.ComponentSDD}})
	if err := storeA.Write(a); err != nil {
		t.Fatalf("Write A: %v", err)
	}
	if err := storeB.Write(b); err != nil {
		t.Fatalf("Write B: %v", err)
	}

	gotA, err := storeA.Read()
	if err != nil {
		t.Fatalf("Read A: %v", err)
	}
	gotB, err := storeB.Read()
	if err != nil {
		t.Fatalf("Read B: %v", err)
	}
	if gotA.Persona != "neutral" || gotA.InstalledAgents[0] != "claude-code" || gotA.ClaudeModelAssignments["sdd-apply"] != "sonnet" || gotA.Components[0] != model.ComponentEngram {
		t.Errorf("el estado de A se ha contaminado: %#v", gotA)
	}
	if gotB.Persona != "gentleman" || gotB.InstalledAgents[0] != "opencode" || gotB.ClaudeModelAssignments["sdd-apply"] != "opus" || gotB.Components[0] != model.ComponentSDD {
		t.Errorf("el estado de B se ha contaminado: %#v", gotB)
	}
	if _, err := os.Stat(Path(home)); !os.IsNotExist(err) {
		t.Errorf("escribir estado de proyecto no debe crear el global; stat = %v", err)
	}
}

func TestProjectStoreSeedsFromGlobalWithoutModifyingIt(t *testing.T) {
	isolateGlobalState(t)
	home, root := t.TempDir(), t.TempDir()
	recorded := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	checked := recorded.Add(-time.Hour)
	global := InstallState{
		InstalledAgents:     []string{"claude-code", "codex"},
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentEngram},
		Persona:             "neutral",
		RDDMode:             "on",
		RDDModeRecordedAt:   &recorded,
		LastUpdateCheck:     &checked,
		PendingSync:         true,
	}
	if err := Write(home, global); err != nil {
		t.Fatal(err)
	}
	before := readBytes(t, Path(home))
	store := ProjectStore(home, root)

	seed, err := store.Read()
	if err != nil {
		t.Fatalf("Read() = %v, want la semilla global", err)
	}

	if seed.InstalledAgents != nil || seed.Persona != "neutral" || !seed.SelectionConfigured {
		t.Errorf("la semilla no refleja la selección global: %#v", seed)
	}
	if seed.RDDMode != "" || seed.RDDModeRecordedAt != nil || seed.LastUpdateCheck != nil || seed.PendingSync {
		t.Errorf("la semilla no debe arrastrar campos de máquina: %#v", seed)
	}
	if _, err := os.Stat(store.Path()); !os.IsNotExist(err) {
		t.Errorf("leer la semilla no debe crear el estado del proyecto; stat = %v", err)
	}
	if !bytes.Equal(before, readBytes(t, Path(home))) {
		t.Error("leer la semilla ha modificado el estado global")
	}
}

func TestProjectStoreFirstWriteDetachesFromGlobal(t *testing.T) {
	isolateGlobalState(t)
	home, root := t.TempDir(), t.TempDir()
	if err := Write(home, InstallState{InstalledAgents: []string{"claude-code"}, Persona: "neutral", RDDMode: "off"}); err != nil {
		t.Fatal(err)
	}
	before := readBytes(t, Path(home))
	store := ProjectStore(home, root)

	seed, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	seed.Persona = "gentleman"
	if err := store.Write(seed); err != nil {
		t.Fatalf("Write() = %v", err)
	}

	got, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if got.Persona != "gentleman" {
		t.Errorf("tras la primera escritura manda el proyecto: persona = %q, want gentleman", got.Persona)
	}
	if got.RDDMode != "" {
		t.Errorf("el documento del proyecto no debe guardar rdd_mode: %q", got.RDDMode)
	}
	if !bytes.Equal(before, readBytes(t, Path(home))) {
		t.Error("escribir el estado del proyecto ha modificado el global")
	}
	// Cambiar el global después ya no influye en el proyecto.
	if err := Write(home, InstallState{InstalledAgents: []string{"opencode"}, Persona: "custom"}); err != nil {
		t.Fatal(err)
	}
	again, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if again.Persona != "gentleman" || again.InstalledAgents != nil {
		t.Errorf("el global ha vuelto a pisar al proyecto: %#v", again)
	}
}

func TestProjectStoreMissingEverywhereReportsNotExist(t *testing.T) {
	isolateGlobalState(t)
	_, err := ProjectStore(t.TempDir(), t.TempDir()).Read()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read() error = %v, want os.ErrNotExist", err)
	}
}

func TestProjectStoreCorruptProjectStateDoesNotFallBackToGlobal(t *testing.T) {
	isolateGlobalState(t)
	home, root := t.TempDir(), t.TempDir()
	if err := Write(home, InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	store := ProjectStore(home, root)
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(), []byte("{no es json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := store.Read()
	if err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read() error = %v, want un error de decodificación sin volver a la semilla", err)
	}
}

func TestProjectStoreAtHomeSharesTheGlobalFile(t *testing.T) {
	isolateGlobalState(t)
	home := t.TempDir()
	if err := Write(home, InstallState{InstalledAgents: []string{"claude-code"}, RDDMode: "on"}); err != nil {
		t.Fatal(err)
	}

	got, err := ProjectStore(home, home).Read()
	if err != nil {
		t.Fatal(err)
	}
	if got.RDDMode != "on" || got.InstalledAgents[0] != "claude-code" {
		t.Errorf("con la raíz en HOME el almacén es el global sin recortar campos: %#v", got)
	}
}

func TestWithMachineFieldsKeepsProjectSelection(t *testing.T) {
	recorded := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	machine := InstallState{RDDMode: "on", RDDModeRecordedAt: &recorded, PendingSync: true, InstalledAgents: []string{"codex"}}
	project := InstallState{InstalledAgents: []string{"claude-code"}, Persona: "neutral"}

	got := WithMachineFields(project, machine)

	if !got.PendingSync || got.RDDMode != "on" || got.RDDModeRecordedAt == nil {
		t.Errorf("faltan los campos de máquina: %#v", got)
	}
	if got.InstalledAgents[0] != "claude-code" || got.Persona != "neutral" {
		t.Errorf("la selección debe seguir siendo la del proyecto: %#v", got)
	}
}

func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func gitIgnored(t *testing.T, dir, path string) bool {
	t.Helper()
	command := exec.Command("git", "-C", dir, "check-ignore", "-q", path)
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
	return command.Run() == nil
}

func TestWritingProjectStateExcludesItFromGit(t *testing.T) {
	gitAvailable(t)
	isolateGlobalState(t)
	home, root := t.TempDir(), t.TempDir()
	runGit(t, root, "init", "-q")
	store := ProjectStore(home, root)

	if err := store.Write(InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatalf("Write() = %v", err)
	}

	for _, name := range []string{".axiom/state.json", ".axiom/state.json.lock"} {
		if !gitIgnored(t, root, name) {
			t.Errorf("git no ignora %s", name)
		}
	}
	// El resto de `.axiom/` conserva su uso y no queda excluido.
	if gitIgnored(t, root, ".axiom/inbox/skills/x/SKILL.md") {
		t.Error("el patrón no debe excluir el resto de .axiom/")
	}
	// Ni el .gitignore del usuario ni el árbol de trabajo se tocan.
	if _, err := os.Stat(filepath.Join(root, ".gitignore")); !os.IsNotExist(err) {
		t.Errorf("no debe crearse .gitignore; stat = %v", err)
	}
}

func TestEnsureProjectStateIgnoredIsIdempotent(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	exclude := filepath.Join(root, ".git", "info", "exclude")

	if err := EnsureProjectStateIgnored(root); err != nil {
		t.Fatal(err)
	}
	first := readBytes(t, exclude)
	if err := EnsureProjectStateIgnored(root); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, readBytes(t, exclude)) {
		t.Errorf("la segunda llamada ha reescrito info/exclude:\n%s\n---\n%s", first, readBytes(t, exclude))
	}
	if strings.Count(string(first), "/.axiom/state.json\n") != 1 {
		t.Errorf("la entrada debe aparecer una sola vez:\n%s", first)
	}
}

func TestEnsureProjectStateIgnoredKeepsExistingExcludeContent(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	exclude := filepath.Join(root, ".git", "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exclude, []byte("*.secreto"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureProjectStateIgnored(root); err != nil {
		t.Fatal(err)
	}

	got := string(readBytes(t, exclude))
	if !strings.HasPrefix(got, "*.secreto\n") || !strings.Contains(got, "/.axiom/state.json\n") {
		t.Errorf("info/exclude = %q, want el contenido previo y las entradas nuevas", got)
	}
}

func TestEnsureProjectStateIgnoredSkipsWhenAxiomDirIsAlreadyIgnored(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	exclude := filepath.Join(root, ".git", "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exclude, []byte(".axiom/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureProjectStateIgnored(root); err != nil {
		t.Fatal(err)
	}
	if got := string(readBytes(t, exclude)); got != ".axiom/\n" {
		t.Errorf("no debe añadir nada si .axiom/ ya está excluido: %q", got)
	}
}

func TestEnsureProjectStateIgnoredOutsideGitDoesNothing(t *testing.T) {
	root := t.TempDir()
	if err := EnsureProjectStateIgnored(root); err != nil {
		t.Fatalf("EnsureProjectStateIgnored() = %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("fuera de git no debe crearse nada: %v", entries)
	}
}

func TestEnsureProjectStateIgnoredNestedProjectUsesRelativePattern(t *testing.T) {
	gitAvailable(t)
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	nested := filepath.Join(repo, "servicios", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := EnsureProjectStateIgnored(nested); err != nil {
		t.Fatal(err)
	}

	if !gitIgnored(t, repo, "servicios/api/.axiom/state.json") {
		t.Error("git no ignora el estado del proyecto anidado")
	}
	if gitIgnored(t, repo, ".axiom/state.json") {
		t.Error("el patrón anidado no debe excluir el .axiom de la raíz del repositorio")
	}
}

func TestEnsureProjectStateIgnoredLinkedWorktreeUsesCommonDir(t *testing.T) {
	gitAvailable(t)
	main := t.TempDir()
	runGit(t, main, "init", "-q")
	runGit(t, main, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "init")
	linked := filepath.Join(t.TempDir(), "wt")
	runGit(t, main, "worktree", "add", "-q", "-b", "rama", linked)

	if err := EnsureProjectStateIgnored(linked); err != nil {
		t.Fatal(err)
	}

	if !gitIgnored(t, linked, ".axiom/state.json") {
		t.Error("git no ignora el estado en el worktree enlazado")
	}
}

func TestProjectStoreAgentsRecordedDistinguishesNilFromEmpty(t *testing.T) {
	isolateGlobalState(t)
	home, root := t.TempDir(), t.TempDir()
	project := ProjectStore(home, root)

	if !GlobalStore(home).AgentsRecorded(InstallState{}) {
		t.Error("el almacén global siempre da por registrada la lista")
	}
	if project.AgentsRecorded(InstallState{}) {
		t.Error("un estado de proyecto sin lista no ha registrado agentes")
	}
	if !project.AgentsRecorded(InstallState{InstalledAgents: []string{}}) {
		t.Error("una lista vacía no nula es una instalación sin agentes, ya registrada")
	}

	// La distinción sobrevive al disco: nil se guarda como null y [] como [].
	if err := project.Write(InstallState{SelectionConfigured: true}); err != nil {
		t.Fatal(err)
	}
	got, err := project.Read()
	if err != nil {
		t.Fatal(err)
	}
	if project.AgentsRecorded(got) {
		t.Errorf("tras escribir sin lista, la lectura no debe darla por registrada: %#v", got)
	}
	if err := project.Write(InstallState{SelectionConfigured: true, InstalledAgents: []string{}}); err != nil {
		t.Fatal(err)
	}
	got, err = project.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !project.AgentsRecorded(got) {
		t.Errorf("tras escribir una lista vacía, la lectura debe darla por registrada: %#v", got)
	}
}

func TestProjectStoreAtHomeAlwaysRecordsAgents(t *testing.T) {
	isolateGlobalState(t)
	home := t.TempDir()
	if !ProjectStore(home, home).AgentsRecorded(InstallState{}) {
		t.Error("con la raíz en HOME el fichero es el global y la lista cuenta como registrada")
	}
}
