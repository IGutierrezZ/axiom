package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/components/skills"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Con --scope workspace, los agentes que leen .agents/skills (Codex, VS Code Copilot,
// OpenCode, Kilo, Gemini CLI y Cursor) comparten una sola copia de las skills de Axiom;
// Claude Code, que no la lee, conserva .claude/skills. Las pruebas de este fichero fijan
// ese reparto, la migración de las carpetas propias de instalaciones anteriores y que
// el global no cambia.

// legacyAgentSkillDirs son las carpetas de skills propias de cada agente que dejaban las
// instalaciones workspace anteriores.
var legacyAgentSkillDirs = []string{
	filepath.Join(".github", "skills"),
	filepath.Join(".gemini", "skills"),
	filepath.Join(".opencode", "skills"),
	filepath.Join(".kilo", "skills"),
	filepath.Join(".cursor", "skills"),
	filepath.Join(".codex", "skills"),
}

func sharedSkillsSelection(agentIDs ...model.AgentID) model.Selection {
	return model.Selection{
		Agents:     agentIDs,
		Components: []model.ComponentID{model.ComponentPersona, model.ComponentSDD, model.ComponentSkills},
		Skills:     []model.SkillID{model.SkillGoTesting},
		Persona:    model.PersonaNeutral,
	}
}

func assertSharedSkillsCopy(t *testing.T, workspace string) {
	t.Helper()
	for _, rel := range []string{
		filepath.Join("sdd-init", "SKILL.md"),
		filepath.Join("go-testing", "SKILL.md"),
		filepath.Join("_shared", "skill-resolver.md"),
	} {
		if _, err := os.Stat(filepath.Join(workspace, ".agents", "skills", rel)); err != nil {
			t.Errorf("falta .agents/skills/%s: %v", rel, err)
		}
	}
}

func assertNoDuplicates(t *testing.T, label string, paths []string) {
	t.Helper()
	seen := map[string]struct{}{}
	for _, path := range paths {
		key := filepath.Clean(path)
		if _, dup := seen[key]; dup {
			t.Errorf("%s: ruta repetida %s", label, path)
		}
		seen[key] = struct{}{}
	}
}

// Codex, VS Code, Gemini y OpenCode en el mismo proyecto: una sola copia en
// .agents/skills y ninguna en las carpetas propias. La copia de seguridad, la lista de
// verificación y los ficheros cambiados no repiten rutas, y tras dos sync el tercero no
// cambia nada.
func TestWorkspaceSharedSkillsSingleCopyForAgentsReadingAgentsDir(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	before := homeSnapshotWithoutState(t, home)
	agentIDs := []model.AgentID{model.AgentCodex, model.AgentVSCodeCopilot, model.AgentGeminiCLI, model.AgentOpenCode}

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "codex,vscode-copilot,gemini-cli,opencode"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertSharedSkillsCopy(t, workspace)
	assertNothingUnder(t, workspace, append([]string{filepath.Join(".claude", "skills")}, legacyAgentSkillDirs...)...)
	// Contenido idéntico para todos: sin marcadores sin resolver.
	ledger, err := os.ReadFile(filepath.Join(workspace, ".agents", "skills", "_shared", "review-ledger-contract.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ledger), "{{") {
		t.Errorf("la copia compartida tiene marcadores sin resolver")
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))

	selection := sharedSkillsSelection(agentIDs...)
	adapters := resolveAdaptersScoped(ScopeWorkspace, workspace, agentIDs)
	for _, component := range []model.ComponentID{model.ComponentSDD, model.ComponentSkills} {
		assertNoDuplicates(t, "ruta de verificación de "+string(component), componentPathsWithWorkspaceScoped(home, workspace, ScopeWorkspace, selection, adapters, component))
	}
	installTargets, err := adapterSkillBackupTargets(home, workspace, ScopeWorkspace, selection, adapters)
	if err != nil {
		t.Fatal(err)
	}
	assertNoDuplicates(t, "copia de seguridad de install", installTargets)
	syncTargets, err := syncAdapterSkillBackupTargetsScoped(home, workspace, ScopeWorkspace, selection, adapters)
	if err != nil {
		t.Fatal(err)
	}
	assertNoDuplicates(t, "copia de seguridad de sync", syncTargets)

	first, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection)
	if err != nil {
		t.Fatalf("sync 1: %v", err)
	}
	assertNoDuplicates(t, "ficheros cambiados por el sync", first.ChangedFiles)
	// Los ficheros de instrucciones los declaran a la vez varios componentes (no es cosa
	// de las skills): solo se exige que cada skill se verifique una vez.
	var skillChecks []string
	for _, check := range first.Verify.Checks {
		if strings.Contains(filepath.ToSlash(check.ID), "/skills/") {
			skillChecks = append(skillChecks, check.ID)
		}
	}
	if len(skillChecks) == 0 {
		t.Errorf("la verificación no comprueba ninguna skill")
	}
	assertNoDuplicates(t, "comprobaciones de verificación de skills", skillChecks)
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("sync 2: %v", err)
	}
	snapshot := permissionsTreeSnapshot(t, workspace)
	third, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection)
	if err != nil {
		t.Fatalf("sync 3: %v", err)
	}
	if !third.NoOp || third.FilesChanged != 0 {
		t.Errorf("el tercer sync cambió %d ficheros (noop=%v): %v", third.FilesChanged, third.NoOp, third.ChangedFiles)
	}
	assertSameTree(t, "workspace", snapshot, permissionsTreeSnapshot(t, workspace))
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
}

// Con Claude Code además: .claude/skills (Claude no lee .agents/skills) y .agents/skills,
// y nada más.
func TestWorkspaceSharedSkillsWithClaudeAddsOnlyClaudeDir(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code,opencode,gemini-cli"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertSharedSkillsCopy(t, workspace)
	if _, err := os.Stat(filepath.Join(workspace, ".claude", "skills", "sdd-init", "SKILL.md")); err != nil {
		t.Errorf("falta .claude/skills/sdd-init: %v", err)
	}
	assertNothingUnder(t, workspace, legacyAgentSkillDirs...)
}

// Solo Claude Code: solo .claude/skills; no se crea .agents/skills.
func TestWorkspaceSharedSkillsClaudeOnlyDoesNotCreateAgentsDir(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(workspace, ".claude", "skills", "sdd-init", "SKILL.md")); err != nil {
		t.Errorf("falta .claude/skills/sdd-init: %v", err)
	}
	assertNothingUnder(t, workspace, append([]string{filepath.Join(".agents", "skills")}, legacyAgentSkillDirs...)...)
}

// seedLegacyAgentSkills deja en dir una copia de las skills de Axiom como la dejaba una
// instalación workspace anterior de ese agente.
func seedLegacyAgentSkills(t *testing.T, dir string, agent model.AgentID) {
	t.Helper()
	if _, err := sdd.InjectSkillDirectoryForAgent(dir, "", agent); err != nil {
		t.Fatal(err)
	}
	if _, err := skills.InjectDirectoryWithCapability(dir, []model.SkillID{model.SkillGoTesting}, ""); err != nil {
		t.Fatal(err)
	}
}

// Migración: la copia idéntica a lo que Axiom genera se retira (con su carpeta si queda
// vacía); la modificada por el usuario se conserva con aviso; una skill ajena no se toca.
func TestWorkspaceSharedSkillsMigratesPreviousPerAgentCopies(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	before := homeSnapshotWithoutState(t, home)

	opencodeSkills := filepath.Join(workspace, ".opencode", "skills")
	seedLegacyAgentSkills(t, opencodeSkills, model.AgentOpenCode)
	edited := filepath.Join(opencodeSkills, "go-testing", "SKILL.md")
	original, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	editedContent := append(append([]byte(nil), original...), []byte("\nAjuste del usuario.\n")...)
	mustWriteFile(t, edited, editedContent)

	githubSkills := filepath.Join(workspace, ".github", "skills")
	seedLegacyAgentSkills(t, githubSkills, model.AgentVSCodeCopilot)
	foreign := filepath.Join(githubSkills, "mi-skill", "SKILL.md")
	mustWriteFile(t, foreign, []byte("---\nname: mi-skill\ndescription: mía\n---\n"))

	cursorSkills := filepath.Join(workspace, ".cursor", "skills")
	seedLegacyAgentSkills(t, cursorSkills, model.AgentCursor)

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "opencode,vscode-copilot,cursor"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertSharedSkillsCopy(t, workspace)
	// La modificada se conserva con su contenido, y se avisa.
	if got, _ := os.ReadFile(edited); string(got) != string(editedContent) {
		t.Errorf("la skill modificada cambió:\n%s", got)
	}
	notes := strings.Join(result.Execution.ManualActions, "\n")
	if !strings.Contains(notes, filepath.Join(opencodeSkills, "go-testing")) || !strings.Contains(notes, ".agents") {
		t.Errorf("falta el aviso de la copia modificada:\n%s", notes)
	}
	// Las copias sin cambios de Axiom se retiran.
	for _, gone := range []string{
		filepath.Join(opencodeSkills, "sdd-init"),
		filepath.Join(opencodeSkills, "_shared"),
		filepath.Join(githubSkills, "sdd-init"),
		filepath.Join(githubSkills, "go-testing"),
		filepath.Join(githubSkills, "_shared"),
	} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("la copia de Axiom %s debería haberse retirado", gone)
		}
	}
	// La ajena queda intacta y por eso su carpeta contenedora se conserva.
	if got, err := os.ReadFile(foreign); err != nil || !strings.Contains(string(got), "mi-skill") {
		t.Errorf("la skill ajena cambió o desapareció: %v", err)
	}
	// La carpeta que quedó vacía se borra; .cursor (del agente) no.
	if _, err := os.Stat(cursorSkills); err == nil {
		t.Errorf(".cursor/skills debería haberse borrado al quedar vacía")
	}
	if _, err := os.Stat(filepath.Join(workspace, ".cursor")); err != nil {
		t.Errorf(".cursor no debe borrarse: %v", err)
	}
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))

	// Un sync posterior no vuelve a escribir copias propias.
	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, sharedSkillsSelection(model.AgentOpenCode, model.AgentVSCodeCopilot, model.AgentCursor)); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, err := os.Stat(filepath.Join(opencodeSkills, "sdd-init")); err == nil {
		t.Errorf("el sync volvió a escribir .opencode/skills/sdd-init")
	}
}

// El índice de skills de AGENTS.md apunta a la copia compartida .agents/skills y no a la
// primera carpeta de agente (.opencode/skills) como pasaba con varios agentes.
func TestWorkspaceSharedSkillsIndexPointsToAgentsDir(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	mustWriteFile(t, filepath.Join(workspace, "axiom.yaml"), []byte("workspace:\n  name: indice\n  topology: monorepo-embedded\n"))
	agentsPath := filepath.Join(workspace, "AGENTS.md")
	mustWriteFile(t, agentsPath, []byte("# Proyecto\n\n## Skills\n\n<!-- axiom:skills-index -->\n<!-- /axiom:skills-index -->\n"))

	selection := sharedSkillsSelection(model.AgentOpenCode, model.AgentGeminiCLI, model.AgentCodex)
	result, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !result.SkillRegistryRefreshed || result.SkillsIndexed == 0 {
		t.Fatalf("el índice no se regeneró: refrescado=%v indexadas=%d error=%q", result.SkillRegistryRefreshed, result.SkillsIndexed, result.SkillRegistryError)
	}
	raw, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	index := filepath.ToSlash(string(raw))
	if !strings.Contains(index, ".agents/skills/go-testing/SKILL.md") {
		t.Errorf("el índice no apunta a .agents/skills:\n%s", index)
	}
	for _, unwanted := range []string{".opencode/skills", ".gemini/skills", ".github/skills"} {
		if strings.Contains(index, unwanted) {
			t.Errorf("el índice no debe apuntar a %s:\n%s", unwanted, index)
		}
	}
}

// Con el ámbito global cada agente sigue escribiendo en su carpeta de usuario: la
// copia compartida es solo del proyecto.
func TestGlobalSkillsStayInEachAgentUserDir(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	agentIDs := []model.AgentID{model.AgentOpenCode, model.AgentGeminiCLI, model.AgentCursor, model.AgentCodex}

	if _, err := RunInstall([]string{"--scope", "global", "--agent", "opencode,gemini-cli,cursor,codex"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	for _, id := range agentIDs {
		adapter, err := agents.NewAdapter(id)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(adapter.SkillsDir(home), "sdd-init", "SKILL.md")
		if _, err := os.Stat(want); err != nil {
			t.Errorf("%s: falta %s: %v", id, want, err)
		}
	}
	assertNothingUnder(t, workspace, filepath.Join(".agents", "skills"))
}

// Una carpeta de skills compartida se reclama una sola vez por ejecución en workspace: el
// primer agente la escribe y los siguientes la omiten. Con el ámbito global nunca se
// deduplica, y una carpeta vacía no se reclama.
func TestWorkspaceSkillsDirsClaimOncePerFolderInWorkspaceOnly(t *testing.T) {
	root := t.TempDir()
	build := func(ids ...model.AgentID) []agents.Adapter {
		return resolveAdaptersScoped(ScopeWorkspace, root, ids)
	}
	adapters := build(model.AgentOpenCode, model.AgentGeminiCLI, model.AgentClaudeCode, model.AgentCodex)

	workspaceDirs := newWorkspaceSkillsDirs(ScopeWorkspace)
	var written []model.AgentID
	for _, adapter := range adapters {
		if !workspaceDirs.claimSkills(adapter, root) {
			written = append(written, adapter.Agent())
		}
	}
	// OpenCode abre .agents/skills; Gemini y Codex la comparten; Claude tiene la suya.
	if want := []model.AgentID{model.AgentOpenCode, model.AgentClaudeCode}; len(written) != len(want) || written[0] != want[0] || written[1] != want[1] {
		t.Errorf("escriben %v, quiero %v", written, want)
	}

	globalDirs := newWorkspaceSkillsDirs(ScopeGlobal)
	for _, adapter := range adapters {
		if globalDirs.claimSkills(adapter, root) || globalDirs.claimSkills(adapter, root) {
			t.Errorf("%s: con el ámbito global no se deduplica nada", adapter.Agent())
		}
	}
	if workspaceDirs.claim("") {
		t.Errorf("una carpeta vacía no se reclama")
	}
}
