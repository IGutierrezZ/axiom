package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Las pruebas de este fichero cubren la distribución de proyecto de Gemini CLI con
// --scope workspace. Gemini carga GEMINI.md de la raíz del proyecto y de sus
// padres, y los nombres que declara context.fileName en .gemini/settings.json; no
// carga <proyecto>/.gemini/GEMINI.md, que era donde Axiom escribía las
// instrucciones. Ahora van en .gemini/AXIOM.md, registrado en context.fileName.

func geminiSelection() model.Selection {
	return model.Selection{
		Agents:     []model.AgentID{model.AgentGeminiCLI},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentPersona, model.ComponentSDD, model.ComponentSkills},
		Skills:     []model.SkillID{model.SkillGoTesting},
		Persona:    model.PersonaNeutral,
	}
}

func contextFileNamesOf(t *testing.T, settingsPath string) []string {
	t.Helper()
	contextObject, _ := readJSONObject(t, settingsPath)["context"].(map[string]any)
	raw, _ := contextObject["fileName"].([]any)
	names := make([]string, 0, len(raw))
	for _, name := range raw {
		names = append(names, name.(string))
	}
	return names
}

func readJSONObjectFromString(t *testing.T, raw string) map[string]any {
	t.Helper()
	object, err := filemerge.UnmarshalJSONObject([]byte(raw))
	if err != nil {
		t.Fatalf("JSON de prueba inválido: %v", err)
	}
	return object
}

func captureWorkspaceNotices(t *testing.T) *bytes.Buffer {
	t.Helper()
	var notices bytes.Buffer
	restore := workspaceInstructionsNoticeWriter
	workspaceInstructionsNoticeWriter = &notices
	t.Cleanup(func() { workspaceInstructionsNoticeWriter = restore })
	return &notices
}

// El orquestador y la persona van a .gemini/AXIOM.md, context.fileName conserva
// GEMINI.md y añade ese fichero, el GEMINI.md de la raíz (del equipo) queda intacto
// y no se escribe nada en HOME ni en .gemini/GEMINI.md.
func TestRunInstallWorkspaceGeminiUsesOwnInstructionsFile(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	seedFootprintHome(t, home)
	before := homeSnapshotWithoutState(t, home)
	teamGemini := []byte("# Contexto del equipo\n\nNo tocar.\n")
	mustWriteFile(t, filepath.Join(workspace, "GEMINI.md"), teamGemini)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	instructions, err := os.ReadFile(filepath.Join(workspace, ".gemini", "AXIOM.md"))
	if err != nil {
		t.Fatalf("falta .gemini/AXIOM.md: %v", err)
	}
	for _, section := range []string{"<!-- axiom:sdd-orchestrator -->", "<!-- axiom:engram-protocol -->", "<!-- axiom:agent-routing -->"} {
		if !strings.Contains(string(instructions), section) {
			t.Errorf(".gemini/AXIOM.md no contiene %s", section)
		}
	}
	if strings.Contains(string(instructions), "{{AXIOM_SKILLS_DIR") {
		t.Errorf(".gemini/AXIOM.md tiene marcadores de ruta sin resolver")
	}
	settingsPath := filepath.Join(workspace, ".gemini", "settings.json")
	if got := contextFileNamesOf(t, settingsPath); !slices.Equal(got, []string{"GEMINI.md", ".gemini/AXIOM.md"}) {
		t.Errorf("context.fileName = %v, quiero [GEMINI.md .gemini/AXIOM.md]", got)
	}
	if mcp, _ := readJSONObject(t, settingsPath)["mcpServers"].(map[string]any); mcp["engram"] == nil {
		t.Errorf("falta mcpServers.engram en .gemini/settings.json: %v", mcp)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".agents", "skills", "sdd-init", "SKILL.md")); err != nil {
		t.Errorf("faltan las skills en .agents/skills: %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(workspace, "GEMINI.md")); !bytes.Equal(after, teamGemini) {
		t.Errorf("el GEMINI.md de la raíz cambió:\n%s", after)
	}
	assertNothingUnder(t, workspace, filepath.Join(".gemini", "GEMINI.md"))
	assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))
}

// Sin GEMINI.md en la raíz, instalar no lo crea.
func TestRunInstallWorkspaceGeminiDoesNotCreateRootGeminiMD(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertNothingUnder(t, workspace, "GEMINI.md", filepath.Join(".gemini", "GEMINI.md"))
}

// context.fileName del usuario, como cadena o como lista, se conserva: el valor del
// proyecto sustituye al del usuario, así que Axiom solo añade su fichero. Las demás
// claves se mantienen y un tercer sync no cambia ningún byte.
func TestWorkspaceGeminiKeepsUserContextFileNamesAndIsIdempotent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		want     []string
	}{
		{
			name:     "cadena",
			settings: `{"theme":"GitHub","context":{"fileName":"CONTEXTO.md","includeDirectories":["docs"]},"mcpServers":{"propio":{"command":"propio"}}}`,
			want:     []string{"CONTEXTO.md", ".gemini/AXIOM.md"},
		},
		{
			name:     "lista",
			settings: `{"theme":"GitHub","context":{"fileName":["AGENTS.md","GEMINI.md"],"includeDirectories":["docs"]},"mcpServers":{"propio":{"command":"propio"}}}`,
			want:     []string{"AGENTS.md", "GEMINI.md", ".gemini/AXIOM.md"},
		},
		{
			name:     "sin fileName",
			settings: `{"theme":"GitHub","context":{"includeDirectories":["docs"]},"mcpServers":{"propio":{"command":"propio"}}}`,
			want:     []string{"GEMINI.md", ".gemini/AXIOM.md"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AXIOM_STATE_DIR", t.TempDir())
			home, workspace, _ := footprintEnv(t)
			seedFootprintHome(t, home)
			before := homeSnapshotWithoutState(t, home)
			settingsPath := filepath.Join(workspace, ".gemini", "settings.json")
			mustWriteFile(t, settingsPath, []byte(tc.settings))

			if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli"}, macOSDetectionResult()); err != nil {
				t.Fatalf("RunInstall() error = %v", err)
			}
			for i := 0; i < 2; i++ {
				if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, geminiSelection()); err != nil {
					t.Fatalf("sync %d: %v", i+1, err)
				}
			}

			if got := contextFileNamesOf(t, settingsPath); !slices.Equal(got, tc.want) {
				t.Errorf("context.fileName = %v, quiero %v", got, tc.want)
			}
			settings := readJSONObject(t, settingsPath)
			if settings["theme"] != "GitHub" {
				t.Errorf("se perdió una clave del usuario: %v", settings)
			}
			contextObject, _ := settings["context"].(map[string]any)
			if dirs, _ := contextObject["includeDirectories"].([]any); len(dirs) != 1 || dirs[0] != "docs" {
				t.Errorf("context.includeDirectories cambió: %v", contextObject)
			}
			if mcp, _ := settings["mcpServers"].(map[string]any); mcp["propio"] == nil || mcp["engram"] == nil {
				t.Errorf("mcpServers debe conservar el servidor del usuario y añadir engram: %v", mcp)
			}
			assertNothingUnder(t, workspace, filepath.Join(".gemini", "GEMINI.md"))
			assertSameTree(t, "home", before, homeSnapshotWithoutState(t, home))

			snapshot := permissionsTreeSnapshot(t, workspace)
			if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, geminiSelection()); err != nil {
				t.Fatalf("sync 3: %v", err)
			}
			assertSameTree(t, "workspace", snapshot, permissionsTreeSnapshot(t, workspace))
		})
	}
}

// Un context o un context.fileName de un tipo inesperado no se tocan: es del
// usuario y Axiom avisa en lugar de sustituirlo.
func TestWorkspaceGeminiUnexpectedContextTypeIsLeftUntouched(t *testing.T) {
	for name, settings := range map[string]string{
		"context no es objeto":  `{"context":"docs"}`,
		"fileName no es texto":  `{"context":{"fileName":7}}`,
		"lista con un no texto": `{"context":{"fileName":["GEMINI.md",3]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("AXIOM_STATE_DIR", t.TempDir())
			_, workspace, _ := footprintEnv(t)
			notices := captureWorkspaceNotices(t)
			settingsPath := filepath.Join(workspace, ".gemini", "settings.json")
			mustWriteFile(t, settingsPath, []byte(settings))

			if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli", "--component", "persona", "--component", "sdd"}, macOSDetectionResult()); err != nil {
				t.Fatalf("RunInstall() error = %v", err)
			}

			// El context del usuario queda como estaba; el resto del documento sí se
			// toca (mcpServers).
			want := readJSONObjectFromString(t, settings)["context"]
			if got := readJSONObject(t, settingsPath)["context"]; !reflect.DeepEqual(got, want) {
				t.Errorf("context cambió: %v, quiero %v", got, want)
			}
			if !strings.Contains(notices.String(), "context.fileName") {
				t.Errorf("falta el aviso:\n%s", notices.String())
			}
		})
	}
}

// Un settings.json que no se puede leer como JSON se rechaza antes de escribir
// nada: es un fichero del usuario y reescribirlo desde cero lo perdería.
func TestWorkspaceGeminiRefusesUnreadableSettings(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	broken := []byte("{ esto no es json")
	mustWriteFile(t, filepath.Join(workspace, ".gemini", "settings.json"), broken)

	_, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli"}, macOSDetectionResult())
	if err == nil || !strings.Contains(err.Error(), "no es JSON válido") {
		t.Fatalf("RunInstall() debe rechazar el fichero ilegible, error = %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(workspace, ".gemini", "settings.json")); !bytes.Equal(after, broken) {
		t.Errorf("settings.json se modificó:\n%s", after)
	}
	assertNothingUnder(t, workspace, filepath.Join(".gemini", "AXIOM.md"), filepath.Join(".gemini", "skills"))
}

// Migración: el .gemini/GEMINI.md de una instalación workspace anterior se retira
// solo cuando es provablemente de Axiom (únicamente secciones gestionadas y, como
// mucho, el texto de persona que genera el instalador).
func TestWorkspaceGeminiRetiresProvablyAxiomLegacyGeminiMD(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	legacy := filepath.Join(workspace, ".gemini", "GEMINI.md")
	mustWriteFile(t, legacy, []byte("<!-- axiom:persona -->\nPersona antigua.\n<!-- /axiom:persona -->\n\n<!-- axiom:sdd-orchestrator -->\nOrquestador antiguo.\n<!-- /axiom:sdd-orchestrator -->\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	assertNothingUnder(t, workspace, filepath.Join(".gemini", "GEMINI.md"))
	if _, err := os.Stat(filepath.Join(workspace, ".gemini", "AXIOM.md")); err != nil {
		t.Errorf("falta .gemini/AXIOM.md: %v", err)
	}
}

// Una instalación anterior escribía la persona sin marcadores seguida de las
// secciones gestionadas. Ese texto es el que genera el instalador, así que el
// fichero también se retira; sync lo migra igual que install.
func TestWorkspaceGeminiSyncRetiresLegacyGeminiMDWithInstallerPersona(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	legacy := filepath.Join(workspace, ".gemini", "GEMINI.md")
	mustWriteFile(t, legacy, []byte(assets.MustRead("generic/persona-neutral.md")+"\n<!-- axiom:sdd-orchestrator -->\nOrquestador antiguo.\n<!-- /axiom:sdd-orchestrator -->\n"))

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, geminiSelection()); err != nil {
		t.Fatalf("sync: %v", err)
	}

	assertNothingUnder(t, workspace, filepath.Join(".gemini", "GEMINI.md"))
	if _, err := os.Stat(filepath.Join(workspace, ".gemini", "AXIOM.md")); err != nil {
		t.Errorf("falta .gemini/AXIOM.md: %v", err)
	}
}

// Un .gemini/GEMINI.md con contenido que Axiom no escribió se deja y se avisa.
func TestWorkspaceGeminiKeepsLegacyGeminiMDWithUserContent(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	_, workspace, _ := footprintEnv(t)
	legacy := filepath.Join(workspace, ".gemini", "GEMINI.md")
	content := []byte("# Mis notas\n\n<!-- axiom:sdd-orchestrator -->\nOrquestador antiguo.\n<!-- /axiom:sdd-orchestrator -->\n")
	mustWriteFile(t, legacy, content)

	result, err := RunInstall([]string{"--scope", "workspace", "--agent", "gemini-cli"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if after, _ := os.ReadFile(legacy); !bytes.Equal(after, content) {
		t.Errorf(".gemini/GEMINI.md cambió:\n%s", after)
	}
	if !strings.Contains(strings.Join(result.Execution.ManualActions, "\n"), legacy) {
		t.Errorf("falta el aviso del fichero conservado: %v", result.Execution.ManualActions)
	}
}

// Con el ámbito global Gemini sigue escribiendo en ~/.gemini/GEMINI.md y no se
// registra nada en context.fileName.
func TestRunInstallGlobalGeminiKeepsGlobalLayout(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)

	if _, err := RunInstall([]string{"--scope", "global", "--agent", "gemini-cli", "--component", "persona", "--component", "sdd", "--component", "engram"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".gemini", "GEMINI.md")); err != nil {
		t.Errorf("falta ~/.gemini/GEMINI.md: %v", err)
	}
	assertNothingUnder(t, home, filepath.Join(".gemini", "AXIOM.md"))
	if _, declared := readJSONObject(t, filepath.Join(home, ".gemini", "settings.json"))["context"]; declared {
		t.Errorf("la instalación global no debe declarar context.fileName")
	}
	assertNothingUnder(t, workspace, ".gemini", "GEMINI.md")
}

// Backup y verify cubren el fichero propio de instrucciones y settings.json del
// proyecto, y la verificación posterior a un sync no da ningún fallo.
func TestWorkspaceGeminiBackupAndVerifyCoverOwnFiles(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	home, workspace, _ := footprintEnv(t)
	selection := geminiSelection()
	instructions := filepath.Join(workspace, ".gemini", "AXIOM.md")
	settings := filepath.Join(workspace, ".gemini", "settings.json")

	targets, err := syncBackupTargetsScoped(home, workspace, ScopeWorkspace, selection, resolveAdaptersScoped(ScopeWorkspace, workspace, selection.Agents))
	if err != nil {
		t.Fatalf("syncBackupTargetsScoped() error = %v", err)
	}
	for _, want := range []string{instructions, settings, filepath.Join(workspace, ".gemini", "GEMINI.md")} {
		if !containsPath(targets, want) {
			t.Errorf("la copia de seguridad no incluye %s: %v", want, targets)
		}
	}

	if _, err := RunSyncWithSelectionScoped(home, workspace, ScopeWorkspace, selection); err != nil {
		t.Fatalf("sync: %v", err)
	}
	report := runPostSyncVerificationScoped(home, workspace, ScopeWorkspace, selection)
	if report.Failed != 0 {
		t.Errorf("la verificación tiene %d fallos: %+v", report.Failed, report.Checks)
	}
	checked := false
	for _, check := range report.Checks {
		if strings.Contains(check.Description, instructions) || strings.Contains(check.ID, instructions) {
			checked = true
		}
	}
	if !checked {
		t.Errorf("la verificación no comprueba %s", instructions)
	}
}
