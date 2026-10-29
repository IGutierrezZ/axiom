package sdd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/cursor"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kiro"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
)

// skillsScopeAdapters son los adaptadores con subagentes o prompts que antes
// codificaban la ruta de skills del home del usuario.
func skillsScopeAdapters() map[string]agents.Adapter {
	return map[string]agents.Adapter{
		"claude": claudeAdapter(),
		"cursor": cursor.NewAdapter(),
		"kimi":   kimiAdapter(),
		"kiro":   kiro.NewAdapter(),
	}
}

// skillsRelDir devuelve el directorio de skills del adaptador relativo a su
// raíz, con separador `/`, que es la forma que ven los subagentes.
func skillsRelDir(adapter agents.Adapter) string {
	return filepath.ToSlash(adapter.SkillsDir(""))
}

// injectedPromptFiles recorre los ficheros que la inyección de SDD escribe bajo
// root. Excluye el contenido de las skills (su documentación lista a propósito
// los directorios de usuario de todos los agentes) salvo el flujo perezoso del
// orquestador, que sí lleva rutas de skills resueltas.
func injectedPromptFiles(t *testing.T, root string, adapter agents.Adapter) map[string]string {
	t.Helper()
	skillsDir := adapter.SkillsDir(root)
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasPrefix(path, skillsDir+string(filepath.Separator)) &&
			!strings.HasSuffix(path, "sdd-orchestrator-workflow.md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		files[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return files
}

// En workspace no puede quedar una ruta de skills del home salvo en las líneas
// de doble resolución («en el workspace primero, si no en el home» o «(global) o
// (workspace)»), que ya nombran también la alternativa de workspace. En ningún ámbito puede quedar un marcador
// sin resolver.
func TestInjectedSkillsPathsFollowInstallScope(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	for name, adapter := range skillsScopeAdapters() {
		rel := skillsRelDir(adapter)
		home := "~/" + rel

		t.Run(name+"/workspace", func(t *testing.T) {
			root := t.TempDir()
			if _, err := Inject(root, adapter, "", InjectOptions{WorkspaceDir: root}); err != nil {
				t.Fatalf("Inject(workspace): %v", err)
			}
			files := injectedPromptFiles(t, root, adapter)
			if len(files) == 0 {
				t.Fatal("la inyección no escribió ningún fichero")
			}
			resolved := false
			for path, content := range files {
				if strings.Contains(content, "{{AXIOM_SKILLS_DIR") {
					t.Errorf("%s: marcador de ruta de skills sin resolver", path)
				}
				if strings.Contains(content, `%USERPROFILE%\\`+strings.ReplaceAll(rel, "/", `\\`)) {
					t.Errorf("%s: ruta de skills de Windows del home en una instalación workspace", path)
				}
				for _, line := range strings.Split(content, "\n") {
					if !strings.Contains(line, home) {
						continue
					}
					if strings.Count(line, rel) <= strings.Count(line, home) && !strings.Contains(line, "(workspace)") {
						t.Errorf("%s: ruta de skills del home sin alternativa de workspace: %s", path, strings.TrimSpace(line))
					}
				}
				if strings.Contains(content, rel+"/sdd-") || strings.Contains(content, rel+"/_shared/") {
					resolved = true
				}
			}
			if !resolved {
				t.Errorf("ningún fichero referencia %s/... relativo al proyecto", rel)
			}
		})

		t.Run(name+"/global", func(t *testing.T) {
			root := t.TempDir()
			if _, err := Inject(root, adapter, "", InjectOptions{}); err != nil {
				t.Fatalf("Inject(global): %v", err)
			}
			files := injectedPromptFiles(t, root, adapter)
			resolved := false
			for path, content := range files {
				if strings.Contains(content, "{{AXIOM_SKILLS_DIR") {
					t.Errorf("%s: marcador de ruta de skills sin resolver", path)
				}
				if strings.Contains(content, home+"/sdd-") || strings.Contains(content, home+"/_shared/") {
					resolved = true
				}
			}
			if !resolved {
				t.Errorf("la instalación global debe conservar %s/...", home)
			}
		})
	}
}

// Los subagentes de Claude son el caso que motivó el cambio: en workspace leen
// la skill de fase en `.claude/skills`, y en global en `~/.claude/skills`.
func TestClaudeSubAgentsReadSkillsFromInstallScope(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	adapter := claudeAdapter()
	cases := []struct {
		name      string
		workspace bool
		want      string
		forbidden string
	}{
		{"workspace", true, "`.claude/skills/sdd-apply/SKILL.md`", "~/.claude/skills"},
		{"global", false, "`~/.claude/skills/sdd-apply/SKILL.md`", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			opts := InjectOptions{}
			if c.workspace {
				opts.WorkspaceDir = root
			}
			if _, err := Inject(root, adapter, "", opts); err != nil {
				t.Fatalf("Inject: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(adapter.SubAgentsDir(root), "sdd-apply.md"))
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)
			if !strings.Contains(content, c.want) {
				t.Errorf("sdd-apply.md no contiene %s:\n%s", c.want, content)
			}
			if c.forbidden != "" && strings.Contains(content, c.forbidden) {
				t.Errorf("sdd-apply.md contiene %q en una instalación workspace", c.forbidden)
			}
		})
	}
}

// Kiro conserva las dos grafías (POSIX y Windows) y ambas siguen el ámbito.
func TestKiroSubAgentsResolveBothPathSpellings(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	adapter := kiro.NewAdapter()

	workspaceRoot := t.TempDir()
	if _, err := Inject(workspaceRoot, adapter, "", InjectOptions{WorkspaceDir: workspaceRoot}); err != nil {
		t.Fatalf("Inject(workspace): %v", err)
	}
	data, err := os.ReadFile(filepath.Join(adapter.SubAgentsDir(workspaceRoot), "sdd-apply.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"`.kiro/skills/sdd-apply/SKILL.md`", `.kiro\\skills\\sdd-apply\\SKILL.md`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("workspace: falta %s", want)
		}
	}
	for _, forbidden := range []string{"~/.kiro", "%USERPROFILE%"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("workspace: no debe quedar %s", forbidden)
		}
	}

	globalRoot := t.TempDir()
	if _, err := Inject(globalRoot, adapter, "", InjectOptions{}); err != nil {
		t.Fatalf("Inject(global): %v", err)
	}
	data, err = os.ReadFile(filepath.Join(adapter.SubAgentsDir(globalRoot), "sdd-apply.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"`~/.kiro/skills/sdd-apply/SKILL.md`", `%USERPROFILE%\\.kiro\\skills\\sdd-apply\\SKILL.md`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("global: falta %s", want)
		}
	}
}

// Un adaptador sin modo workspace recibe el home aunque la instalación pida
// workspace: sus rutas de skills deben seguir apuntando al home.
func TestInstallsInWorkspaceOnlyWhenRootIsWorkspace(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "proyecto")
	home := filepath.Join(t.TempDir(), "home")
	cases := []struct {
		name string
		root string
		opts InjectOptions
		want bool
	}{
		{"raiz igual al workspace", workspace, InjectOptions{WorkspaceDir: workspace}, true},
		{"raiz con separadores redundantes", workspace + string(filepath.Separator), InjectOptions{WorkspaceDir: workspace}, true},
		{"adaptador sin modo workspace usa el home", home, InjectOptions{WorkspaceDir: workspace}, false},
		{"instalación global", home, InjectOptions{}, false},
		{"raíz vacía", "", InjectOptions{WorkspaceDir: workspace}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := installsInWorkspace(c.root, c.opts); got != c.want {
				t.Fatalf("installsInWorkspace(%q) = %v, quiero %v", c.root, got, c.want)
			}
		})
	}
}

// Guarda de regresión: los assets que llevan rutas de skills usan el marcador.
// La única ruta del home admitida en crudo es la de doble resolución, que
// nombra también la alternativa de workspace.
func TestSubAgentAndCommandAssetsUseSkillsDirPlaceholder(t *testing.T) {
	dirs := []string{
		"claude/agents", "claude/commands", "cursor/agents", "kimi/agents", "kiro/agents",
	}
	for _, dir := range dirs {
		entries, err := fs.ReadDir(assets.FS, dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content := assets.MustRead(dir + "/" + entry.Name())
			for _, line := range strings.Split(content, "\n") {
				for _, home := range []string{"~/.claude/skills", "~/.cursor/skills", "~/.config/agents/skills", "~/.kiro/skills", `%USERPROFILE%\\.kiro`} {
					if strings.Contains(line, home) && !strings.Contains(line, "if absent") {
						t.Errorf("%s/%s codifica %s sin marcador: %s", dir, entry.Name(), home, strings.TrimSpace(line))
					}
				}
			}
		}
	}
}

func TestResolveSkillsDirPlaceholdersLeavesUnrelatedContentUntouched(t *testing.T) {
	const plain = "sin marcadores: ~/.claude/skills/_shared/x.md"
	if got := resolveSkillsDirPlaceholders(plain, claudeAdapter(), true); got != plain {
		t.Fatalf("contenido sin marcador modificado: %q", got)
	}
}
