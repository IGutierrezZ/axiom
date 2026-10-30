package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// retiredAgents son los agentes que se conservan en el código pero no se pueden
// elegir al instalar (decisión de producto, 2026-10-30).
var retiredAgents = []model.AgentID{model.AgentQwenCode, model.AgentHermes, model.AgentKimi}

// allowUnselectableAgentsForTest vuelve a admitir Qwen Code, Hermes y Kimi durante
// la prueba. Solo lo usan las pruebas que ejercen el código de esos adaptadores
// (hub de Kimi, ajustes de Qwen): el código sigue ahí para retomarlos.
func allowUnselectableAgentsForTest(t *testing.T) {
	t.Helper()
	restore := isSelectableAgent
	isSelectableAgent = func(model.AgentID) bool { return true }
	t.Cleanup(func() { isSelectableAgent = restore })
}

// seedRetiredAgentsHome siembra en HOME las carpetas que el descubrimiento por
// ficheros reconocería de Qwen Code, Hermes y Kimi.
func seedRetiredAgentsHome(t *testing.T, home string) {
	t.Helper()
	for _, dir := range []string{".qwen", ".hermes", ".kimi"} {
		mustWriteFile(t, filepath.Join(home, dir, "usuario.txt"), []byte("del usuario\n"))
	}
}

func TestCatalogRetiredAgentsStayKnownButNotSelectable(t *testing.T) {
	for _, id := range retiredAgents {
		if !catalog.IsSupportedAgent(id) {
			t.Errorf("%s debe seguir siendo un agente conocido", id)
		}
		if catalog.IsSelectableAgent(id) {
			t.Errorf("%s no debe ser seleccionable", id)
		}
	}
	for _, agent := range catalog.SelectableAgents() {
		if slices.Contains(retiredAgents, agent.ID) {
			t.Errorf("SelectableAgents() no debe incluir %s", agent.ID)
		}
	}
}

// El valor por defecto de install (sin --agent y sin detección) no incluye los
// agentes retirados.
func TestNormalizeInstallFlagsDefaultsExcludeRetiredAgents(t *testing.T) {
	input, err := NormalizeInstallFlags(InstallFlags{}, system.DetectionResult{})
	if err != nil {
		t.Fatalf("NormalizeInstallFlags() error = %v", err)
	}
	for _, id := range retiredAgents {
		if slices.Contains(input.Selection.Agents, id) {
			t.Errorf("la selección por defecto no debe incluir %s: %v", id, input.Selection.Agents)
		}
	}
}

// Con las carpetas de los agentes retirados detectadas, la selección por defecto
// tampoco las adopta.
func TestDefaultAgentsFromDetectionIgnoresRetiredAgents(t *testing.T) {
	detection := makeDetectionWithAgents("kimi", "qwen-code", "hermes", "claude-code")
	got := defaultAgentsFromDetection(detection)
	if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(got, want) {
		t.Errorf("defaultAgentsFromDetection() = %v, want %v", got, want)
	}

	// Si solo hay agentes retirados detectados se cae al catálogo seleccionable.
	got = defaultAgentsFromDetection(makeDetectionWithAgents("kimi", "qwen-code", "hermes"))
	if len(got) == 0 {
		t.Fatal("sin agentes seleccionables detectados debe usarse el catálogo seleccionable")
	}
	for _, id := range retiredAgents {
		if slices.Contains(got, id) {
			t.Errorf("el valor por defecto no debe incluir %s: %v", id, got)
		}
	}
}

// --agent con un agente retirado falla con un error claro, también si va junto a
// otros agentes, y no instala nada.
func TestRunInstallRejectsRetiredAgents(t *testing.T) {
	for _, id := range retiredAgents {
		for _, scope := range []string{"global", "workspace"} {
			t.Run(string(id)+"/"+scope, func(t *testing.T) {
				home, workspace, commands := footprintEnv(t)
				t.Setenv("AXIOM_STATE_DIR", "")
				before := permissionsTreeSnapshot(t, home)

				_, err := RunInstall([]string{"--scope", scope, "--agent", "claude-code," + string(id)}, macOSDetectionResult())
				if err == nil {
					t.Fatalf("RunInstall() con %s debe fallar", id)
				}
				for _, want := range []string{string(id), "no está disponible de momento"} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("el error %q debe contener %q", err, want)
					}
				}
				assertSameTree(t, "home", before, permissionsTreeSnapshot(t, home))
				if len(commands.get()) != 0 {
					t.Errorf("no debe lanzarse ningún comando, se lanzaron %v", commands.get())
				}
				if entries, _ := os.ReadDir(workspace); len(entries) != 0 {
					t.Errorf("el proyecto debe quedar vacío, contiene %v", entries)
				}
			})
		}
	}
}

// Un agente mal escrito sigue dando el error de agente desconocido, no el de agente
// no disponible.
func TestAsInstallAgentIDsKeepsUnknownAgentError(t *testing.T) {
	_, err := asInstallAgentIDs([]string{"cluade"})
	if err == nil || !strings.Contains(err.Error(), "unsupported agent") {
		t.Fatalf("asInstallAgentIDs(cluade) error = %v, want unsupported agent", err)
	}
}

// El descubrimiento sin estado (sync) no adopta los agentes retirados aunque
// tengan carpeta en HOME, pero sí el resto.
func TestDiscoverAgentsIgnoresRetiredAgentFolders(t *testing.T) {
	home := t.TempDir()
	seedRetiredAgentsHome(t, home)
	seedClaudeHome(t, home)

	got := DiscoverAgents(state.GlobalStore(home))

	if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(got, want) {
		t.Errorf("DiscoverAgents() = %v, want %v", got, want)
	}
}

// sync --scope workspace sin estado de proyecto, con las carpetas de los agentes
// retirados en HOME: no los adopta, no escribe en HOME y no los persiste.
func TestRunSyncWorkspaceWithoutStateIgnoresRetiredAgentFolders(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	seedRetiredAgentsHome(t, home)
	notices := captureWorkspaceAgentsNotice(t)
	before := permissionsTreeSnapshot(t, home)

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	assertSameTree(t, "home", before, permissionsTreeSnapshot(t, home))
	if len(result.Agents) != 0 || !result.NoOp {
		t.Errorf("sin agentes seleccionables el sync no hace nada: agentes=%v noop=%v", result.Agents, result.NoOp)
	}
	if notices.Len() != 0 {
		t.Errorf("un agente que no se adopta no genera aviso:\n%s", notices)
	}
}

// Un estado de proyecto que conserva un agente retirado: sync --scope workspace lo
// omite con aviso y no escribe en HOME; el agente seleccionable sigue sincronizándose.
func TestRunSyncWorkspaceWithPersistedRetiredAgentIsOmittedWithNotice(t *testing.T) {
	for _, id := range retiredAgents {
		t.Run(string(id), func(t *testing.T) {
			root, _, home := workspaceAgentsFixture(t)
			t.Chdir(root)
			seedRetiredAgentsHome(t, home)
			if err := state.ProjectStore(home, root).Write(persistedAgents(string(id), "claude-code")); err != nil {
				t.Fatal(err)
			}
			notices := captureWorkspaceAgentsNotice(t)
			before := homeSnapshotWithoutState(t, home)

			result, err := RunSync([]string{"--scope", "workspace"})
			if err != nil {
				t.Fatalf("RunSync() error = %v", err)
			}

			assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
			if want := string(id) + " omitido: no está disponible de momento"; !strings.Contains(notices.String(), want) {
				t.Errorf("falta el aviso %q:\n%s", want, notices)
			}
			if want := []model.AgentID{model.AgentClaudeCode}; !slices.Equal(result.Agents, want) {
				t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
			}
		})
	}
}

// Una instalación global existente que ya tiene el agente retirado en el estado
// sigue sincronizándolo como hasta ahora.
func TestRunSyncGlobalWithPersistedRetiredAgentKeepsSyncing(t *testing.T) {
	for _, id := range retiredAgents {
		t.Run(string(id), func(t *testing.T) {
			root, _, home := workspaceAgentsFixture(t)
			t.Chdir(root)
			notices := captureWorkspaceAgentsNotice(t)
			if err := state.GlobalStore(home).Write(persistedAgents(string(id))); err != nil {
				t.Fatal(err)
			}

			result, err := RunSync([]string{"--scope", "global"})
			if err != nil {
				t.Fatalf("RunSync() error = %v", err)
			}

			if want := []model.AgentID{id}; !slices.Equal(result.Agents, want) {
				t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
			}
			if notices.Len() != 0 {
				t.Errorf("con --scope global no hay aviso de agentes omitidos:\n%s", notices)
			}
		})
	}
}
