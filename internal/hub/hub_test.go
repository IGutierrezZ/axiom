package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

func TestDetector(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axiom-detector-test-*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewDetector()

	// 1. Proyecto Go
	goDir := filepath.Join(tempDir, "go-proj")
	_ = os.MkdirAll(goDir, 0755)
	_ = os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module example.com/test\ngo 1.25\n"), 0644)

	detGo, err := detector.Detect(goDir)
	if err != nil {
		t.Fatalf("error detectando Go: %v", err)
	}
	if detGo.PrimaryLanguage != "go" {
		t.Errorf("esperado 'go', obtenido '%s'", detGo.PrimaryLanguage)
	}
	if !detGo.HasTests {
		t.Errorf("esperado HasTests=true para Go")
	}

	// 2. Proyecto .NET / C#
	dotnetDir := filepath.Join(tempDir, "dotnet-proj")
	_ = os.MkdirAll(dotnetDir, 0755)
	_ = os.WriteFile(filepath.Join(dotnetDir, "App.csproj"), []byte("<Project></Project>"), 0644)

	detDotnet, err := detector.Detect(dotnetDir)
	if err != nil {
		t.Fatalf("error detectando dotnet: %v", err)
	}
	if detDotnet.PrimaryLanguage != "csharp" {
		t.Errorf("esperado 'csharp', obtenido '%s'", detDotnet.PrimaryLanguage)
	}

	// 3. Proyecto Node / React
	nodeDir := filepath.Join(tempDir, "node-proj")
	_ = os.MkdirAll(nodeDir, 0755)
	_ = os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte(`{"dependencies": {"react": "^18.0.0"}, "devDependencies": {"vitest": "^1.0.0"}}`), 0644)
	_ = os.WriteFile(filepath.Join(nodeDir, "tsconfig.json"), []byte(`{}`), 0644)

	detNode, err := detector.Detect(nodeDir)
	if err != nil {
		t.Fatalf("error detectando node: %v", err)
	}
	if detNode.PrimaryLanguage != "typescript" {
		t.Errorf("esperado 'typescript', obtenido '%s'", detNode.PrimaryLanguage)
	}
	hasReact := false
	for _, f := range detNode.Frameworks {
		if f == "react" {
			hasReact = true
			break
		}
	}
	if !hasReact {
		t.Errorf("esperado framework 'react' detectado")
	}
	if !detNode.HasTests {
		t.Errorf("esperado HasTests=true para vitest")
	}
}

func TestManagerOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axiom-manager-test-*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, ".axiom", "workspaces.json")
	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("error creando Manager: %v", err)
	}

	// 1. Cargar configuración inicial vacía
	cfg, err := mgr.Load()
	if err != nil {
		t.Fatalf("error en Load(): %v", err)
	}
	if len(cfg.Workspaces) != 0 {
		t.Errorf("esperado 0 workspaces, obtenido %d", len(cfg.Workspaces))
	}

	// 2. Registrar primer proyecto
	proj1 := filepath.Join(tempDir, "proj1")
	_ = os.MkdirAll(proj1, 0755)
	_ = os.WriteFile(filepath.Join(proj1, "axiom.yaml"), []byte("workspace:\n  name: Proj1\n"), 0644)

	rec1, err := mgr.Register(proj1, "Proyecto 1", "monorepo-embedded")
	if err != nil {
		t.Fatalf("error registrando proj1: %v", err)
	}
	if rec1.ID != "proyecto-1" {
		t.Errorf("esperado id 'proyecto-1', obtenido '%s'", rec1.ID)
	}
	if !rec1.IsConfigured {
		t.Errorf("esperado IsConfigured=true para proj1")
	}

	// 3. Obtener activo (debe ser rec1 automáticamente)
	active, err := mgr.GetActive()
	if err != nil {
		t.Fatalf("error en GetActive(): %v", err)
	}
	if active.ID != rec1.ID {
		t.Errorf("esperado activo '%s', obtenido '%s'", rec1.ID, active.ID)
	}

	// 4. Registrar segundo proyecto (sin axiom.yaml)
	proj2 := filepath.Join(tempDir, "proj2")
	_ = os.MkdirAll(proj2, 0755)

	rec2, err := mgr.Register(proj2, "Proyecto 2", "monorepo-embedded")
	if err != nil {
		t.Fatalf("error registrando proj2: %v", err)
	}
	if rec2.IsConfigured {
		t.Errorf("esperado IsConfigured=false para proj2")
	}

	// 5. Conmutar activo a proj2
	updatedActive, err := mgr.SetActive(rec2.ID)
	if err != nil {
		t.Fatalf("error en SetActive: %v", err)
	}
	if updatedActive.ID != rec2.ID {
		t.Errorf("esperado nuevo activo '%s', obtenido '%s'", rec2.ID, updatedActive.ID)
	}

	// 6. Listar
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("error en List(): %v", err)
	}
	if len(list) != 2 {
		t.Errorf("esperado 2 workspaces, obtenidos %d", len(list))
	}

	// 7. Idempotencia de registro (mismo path, actualiza nombre)
	rec1Updated, err := mgr.Register(proj1, "Proyecto 1 Renombrado", "monorepo-embedded")
	if err != nil {
		t.Fatalf("error re-registrando proj1: %v", err)
	}
	if rec1Updated.Name != "Proyecto 1 Renombrado" {
		t.Errorf("esperado nombre actualizado, obtenido '%s'", rec1Updated.Name)
	}
	listAfterReReg, _ := mgr.List()
	if len(listAfterReReg) != 2 {
		t.Errorf("re-registro no debe duplicar la entrada: esperado 2, obtenido %d", len(listAfterReReg))
	}

	// 8. Desregistrar
	if err := mgr.Unregister(rec2.ID); err != nil {
		t.Fatalf("error desregistrando: %v", err)
	}
	listAfterDel, _ := mgr.List()
	if len(listAfterDel) != 1 {
		t.Errorf("esperado 1 workspace tras desregistro, obtenido %d", len(listAfterDel))
	}
}

func TestInitializer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axiom-init-test-*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, ".axiom", "workspaces.json")
	mgr, _ := NewManager(configPath)
	det := NewDetector()
	ini := NewInitializer(mgr, det)

	// Inicializar en una carpeta con un go.mod
	targetProj := filepath.Join(tempDir, "mi-servicio-go")
	_ = os.MkdirAll(targetProj, 0755)
	_ = os.WriteFile(filepath.Join(targetProj, "go.mod"), []byte("module mi-servicio\ngo 1.25\n"), 0644)

	res, err := ini.Init(InitOptions{
		Path: targetProj,
		Name: "Mi Servicio Go",
	})
	if err != nil {
		t.Fatalf("error en Init: %v", err)
	}

	if res.AlreadyExisted {
		t.Errorf("no debía existir previamente")
	}

	// Verificar axiom.yaml
	yamlBytes, err := os.ReadFile(res.ConfigPath)
	if err != nil {
		t.Fatalf("error leyendo axiom.yaml generado: %v", err)
	}
	content := string(yamlBytes)
	if !strings.Contains(content, "name: \"Mi Servicio Go\"") {
		t.Errorf("contenido no incluye el nombre esperado: %s", content)
	}
	if !strings.Contains(content, "\"go\"") {
		t.Errorf("contenido no detectó la tecnología 'go': %s", content)
	}

	// Verificar carpetas creadas
	dirsToCheck := []string{
		filepath.Join(targetProj, ".axiom", "inbox", "skills"),
		filepath.Join(targetProj, "openspec", "specs"),
		filepath.Join(targetProj, "openspec", "changes"),
	}
	for _, d := range dirsToCheck {
		info, err := os.Stat(d)
		if err != nil || !info.IsDir() {
			t.Errorf("directorio requerido %s no fue creado", d)
		}
	}

	// Verificar registro en Manager
	active, err := mgr.GetActive()
	if err != nil {
		t.Fatalf("error recuperando activo del hub: %v", err)
	}
	if active.Path != targetProj {
		t.Errorf("esperado path '%s', obtenido '%s'", targetProj, active.Path)
	}

	// Re-ejecutar Init sin force: debe reportar AlreadyExisted=true y no sobreescribir
	res2, err := ini.Init(InitOptions{
		Path: targetProj,
	})
	if err != nil {
		t.Fatalf("error en segundo Init: %v", err)
	}
	if !res2.AlreadyExisted {
		t.Errorf("esperado AlreadyExisted=true en segunda llamada")
	}
}

func TestSmartAdoptionWithExistingSDDAndAgents(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axiom-smart-adopt-test-*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Crear estructura simulada tipo Ludeka
	openSpecDir := filepath.Join(tempDir, "openspec")
	_ = os.MkdirAll(filepath.Join(openSpecDir, "specs"), 0755)
	_ = os.MkdirAll(filepath.Join(openSpecDir, "changes"), 0755)

	configYamlContent := `schema: "1.0"
context: "Glosario y reglas del juego Ludeka."
projects:
  - path: "src/Ludeka.Web"
    stack: ".NET 10 Blazor Web App"
  - path: "src/Ludeka.Core"
    stack: ".NET 10 Class Library"
  - path: "tests/Ludeka.UnitTests"
    stack: ".NET 10 xUnit"
`
	_ = os.WriteFile(filepath.Join(openSpecDir, "config.yaml"), []byte(configYamlContent), 0644)

	// Crear agentes y skills preexistentes
	skillsDir := filepath.Join(tempDir, ".agents", "skills")
	_ = os.MkdirAll(filepath.Join(skillsDir, "sdd-explore"), 0755)
	_ = os.WriteFile(filepath.Join(skillsDir, "sdd-explore", "SKILL.md"), []byte("description: \"Fase de exploración inicial\"\n"), 0644)

	_ = os.MkdirAll(filepath.Join(skillsDir, "fluentui-blazor"), 0755)
	_ = os.WriteFile(filepath.Join(skillsDir, "fluentui-blazor", "SKILL.md"), []byte("description: \"Componentes Fluent UI Blazor\"\n"), 0644)

	githubDir := filepath.Join(tempDir, ".github")
	_ = os.MkdirAll(githubDir, 0755)
	_ = os.WriteFile(filepath.Join(githubDir, "copilot-instructions.md"), []byte("# Copilot Rules\n"), 0644)

	detector := NewDetector()
	det, err := detector.Detect(tempDir)
	if err != nil {
		t.Fatalf("error en Detect: %v", err)
	}

	if !det.HasExistingSDD {
		t.Errorf("esperado HasExistingSDD=true")
	}
	if det.SpecsRepository != "openspec" {
		t.Errorf("esperado SpecsRepository='openspec', obtenido '%s'", det.SpecsRepository)
	}
	if !strings.Contains(det.DomainContext, "Ludeka") {
		t.Errorf("esperado DomainContext conteniendo 'Ludeka', obtenido '%s'", det.DomainContext)
	}
	if len(det.Projects) != 3 {
		t.Errorf("esperado 3 proyectos detectados, obtenidos %d", len(det.Projects))
	}
	if len(det.ConfiguredRoles) != 3 {
		t.Errorf("esperado 3 roles inferidos (core, web, qa), obtenidos %d", len(det.ConfiguredRoles))
	}
	if len(det.AdoptedSkills) < 3 {
		t.Errorf("esperado al menos 3 skills/agentes detectados, obtenidos %d", len(det.AdoptedSkills))
	}

	// Inicializar con adopción inteligente
	ini := NewInitializer(nil, detector)
	res, err := ini.Init(InitOptions{
		Path: tempDir,
		Name: "Ludeka Adopted",
	})
	if err != nil {
		t.Fatalf("error en Init con adopción: %v", err)
	}

	yamlBytes, err := os.ReadFile(res.ConfigPath)
	if err != nil {
		t.Fatalf("error leyendo axiom.yaml generado: %v", err)
	}
	content := string(yamlBytes)

	// Verificar asignación exacta de repositorios a roles
	if !strings.Contains(content, "src/Ludeka.Web") {
		t.Errorf("axiom.yaml debe contener la ruta 'src/Ludeka.Web': %s", content)
	}
	if !strings.Contains(content, "src/Ludeka.Core") {
		t.Errorf("axiom.yaml debe contener la ruta 'src/Ludeka.Core': %s", content)
	}
	if !strings.Contains(content, "tests/Ludeka.UnitTests") {
		t.Errorf("axiom.yaml debe contener la ruta 'tests/Ludeka.UnitTests': %s", content)
	}
	if !strings.Contains(content, "Glosario y reglas del juego Ludeka.") {
		t.Errorf("axiom.yaml debe contener el contexto de gobernanza: %s", content)
	}
}

func TestInteractiveRolesAndNonBlocking(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axiom-interactive-roles-*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ini := NewInitializer(nil, nil)
	res, err := ini.Init(InitOptions{
		Path: tempDir,
		Name: "Custom Roles Proj",
		Roles: []RoleInput{
			{
				Key:          "backend",
				Name:         "Backend Engine",
				Repositories: []string{"server"},
				NonBlocking:  false,
				Tech:         []string{"go"},
			},
			{
				Key:          "docs",
				Name:         "Documentation",
				Repositories: []string{"docs"},
				NonBlocking:  true,
				Tech:         []string{"markdown"},
			},
		},
	})
	if err != nil {
		t.Fatalf("error en Init con roles personalizados: %v", err)
	}

	yamlBytes, _ := os.ReadFile(res.ConfigPath)
	content := string(yamlBytes)

	if !strings.Contains(content, "backend:") || !strings.Contains(content, "gate_policy: \"blocking\"") {
		t.Errorf("esperado rol 'backend' con gate_policy blocking: %s", content)
	}
	if !strings.Contains(content, "docs:") || !strings.Contains(content, "gate_policy: \"advisory\"") {
		t.Errorf("esperado rol 'docs' con gate_policy advisory: %s", content)
	}
}

// writeGitFile escribe un fichero de git sintético (`.git`, `commondir`) creando sus
// directorios.
func writeGitFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("crear el directorio de %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir %s: %v", path, err)
	}
}

// syntheticLinkedWorktree crea, sin git, un worktree enlazado: su `.git` es un fichero que
// apunta a `<principal>/.git/worktrees/<nombre>`, cuyo git dir declara su `commondir`. Devuelve
// la raíz del worktree y la del checkout principal.
func syntheticLinkedWorktree(t *testing.T, base, name string) (linked, main string) {
	t.Helper()
	main = filepath.Join(base, name+"-principal")
	gitDir := filepath.Join(main, ".git", "worktrees", name)
	writeGitFile(t, filepath.Join(gitDir, "commondir"), "../..\n")
	linked = filepath.Join(base, name)
	writeGitFile(t, filepath.Join(linked, ".git"), "gitdir: "+filepath.ToSlash(gitDir)+"\n")
	return linked, main
}

// REQ-25.31: un worktree enlazado se reconoce por la evidencia de git (`.git` fichero cuyo git
// dir contiene `commondir`), nunca por el nombre de un segmento de su ruta.
func TestIsGitWorktree(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("directorio normal sin git y ruta vacía", func(t *testing.T) {
		if IsGitWorktree(tempDir) {
			t.Errorf("un directorio ordinario no debe ser detectado como worktree")
		}
		if IsGitWorktree("") {
			t.Errorf("la ruta vacía no es un worktree")
		}
	})

	t.Run("repositorio git normal (directorio .git)", func(t *testing.T) {
		normalRepo := filepath.Join(tempDir, "normal-repo")
		_ = os.MkdirAll(filepath.Join(normalRepo, ".git"), 0755)
		if IsGitWorktree(normalRepo) {
			t.Errorf("un repositorio ordinario con directorio .git no debe ser detectado como worktree")
		}
	})

	t.Run("worktree enlazado: .git fichero cuyo git dir tiene commondir", func(t *testing.T) {
		linked, _ := syntheticLinkedWorktree(t, tempDir, "linked-wt")
		if !IsGitWorktree(linked) {
			t.Errorf("un worktree con .git fichero y commondir debe ser detectado como worktree")
		}
		sub := filepath.Join(linked, "src")
		_ = os.MkdirAll(sub, 0755)
		if !IsGitWorktree(sub) {
			t.Errorf("una subcarpeta de un worktree enlazado pertenece a un worktree enlazado")
		}
	})

	t.Run("submódulo: .git fichero sin commondir no es un worktree enlazado", func(t *testing.T) {
		super := filepath.Join(tempDir, "super")
		_ = os.MkdirAll(filepath.Join(super, ".git", "modules", "sub"), 0755)
		sub := filepath.Join(super, "sub")
		writeGitFile(t, filepath.Join(sub, ".git"), "gitdir: ../.git/modules/sub\n")
		if IsGitWorktree(sub) {
			t.Errorf("un submódulo (gitdir sin commondir) no debe ser detectado como worktree enlazado")
		}
	})

	t.Run("un .git fichero hacia un git dir inexistente no cuenta", func(t *testing.T) {
		dangling := filepath.Join(tempDir, "dangling")
		writeGitFile(t, filepath.Join(dangling, ".git"), "gitdir: /fake/main/.git/worktrees/dangling\n")
		if IsGitWorktree(dangling) {
			t.Errorf("sin un git dir legible con commondir no hay evidencia de worktree enlazado")
		}
	})

	t.Run("el nombre de la ruta no decide: axiom-wt y -wt en un checkout principal", func(t *testing.T) {
		for _, segment := range []string{"axiom-wt", "proyecto-wt"} {
			repo := filepath.Join(tempDir, segment, "feature-x")
			_ = os.MkdirAll(filepath.Join(repo, ".git"), 0755)
			if IsGitWorktree(repo) {
				t.Errorf("un checkout principal bajo %q no es un worktree enlazado", segment)
			}
		}
		if IsGitWorktree(filepath.Join(tempDir, "axiom-wt", "no-existe")) {
			t.Errorf("una ruta bajo 'axiom-wt' sin repositorio no es un worktree")
		}
	})
}

// requireGit omite la prueba con -short o sin git instalado: las pruebas con repositorios
// reales ejecutan procesos externos (skill go-testing, regla 9 de tasks.md).
func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v en %s: %v\n%s", args, dir, err, output)
	}
}

// newGitRepository crea un checkout principal real, con un commit, en dir.
func newGitRepository(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("crear %s: %v", dir, err)
	}
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "inicial")
	return dir
}

// REQ-25.31, escenario 1: un checkout principal bajo una ruta con un segmento `axiom-wt` o
// terminado en `-wt` se registra: el nombre de la ruta no es evidencia de worktree.
func TestRegisterAcceptsAMainCheckoutUnderAWtSegment(t *testing.T) {
	requireGit(t)
	for _, segment := range []string{"axiom-wt", "proyecto-wt"} {
		t.Run(segment, func(t *testing.T) {
			base := t.TempDir()
			repo := newGitRepository(t, filepath.Join(base, segment, "proj"))
			if IsGitWorktree(repo) {
				t.Fatalf("IsGitWorktree(%q) = true; un checkout principal no es un worktree enlazado", repo)
			}
			mgr, err := NewManager(filepath.Join(base, "workspaces.json"))
			if err != nil {
				t.Fatal(err)
			}
			rec, err := mgr.Register(repo, "Proj", "monorepo-embedded")
			if err != nil {
				t.Fatalf("Register de un checkout principal bajo %q: %v", segment, err)
			}
			if !pathidentity.SameDirectory(rec.Path, repo) {
				t.Errorf("el registro guarda %q, se esperaba %q", rec.Path, repo)
			}
		})
	}
}

// quotedMainCheckout extrae la ruta citada de la continuación `axiom project add "<ruta>"` de
// un rechazo.
func quotedMainCheckout(t *testing.T, err error) string {
	t.Helper()
	match := regexp.MustCompile(`axiom project add "([^"]+)"`).FindStringSubmatch(err.Error())
	if match == nil {
		t.Fatalf("el rechazo no nombra la continuación `axiom project add \"<checkout principal>\"`: %v", err)
	}
	return match[1]
}

// REQ-25.31, escenario 2: un worktree enlazado real no se registra y el rechazo nombra su
// checkout principal citado, de modo que `axiom project add <checkout principal>` se pueda copiar.
func TestRegisterRefusesALinkedWorktreeAndNamesTheMainCheckout(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	main := newGitRepository(t, filepath.Join(base, "proyecto-wt", "proj"))
	linked := filepath.Join(base, "enlazado")
	runGit(t, main, "worktree", "add", "-q", "-b", "inc/x", linked, "HEAD")

	if !IsGitWorktree(linked) {
		t.Fatalf("IsGitWorktree(%q) = false; un worktree enlazado real debe detectarse", linked)
	}
	mgr, err := NewManager(filepath.Join(base, "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = mgr.Register(linked, "Enlazado", "monorepo-embedded")
	if err == nil {
		t.Fatal("Register de un worktree enlazado debe fallar")
	}
	if got := quotedMainCheckout(t, err); !pathidentity.SameDirectory(got, main) {
		t.Errorf("el rechazo cita %q, se esperaba el checkout principal %q", got, main)
	}
	if !strings.Contains(err.Error(), "worktree") {
		t.Errorf("el rechazo debe decir que la ruta es un worktree: %v", err)
	}
	if list, _ := mgr.List(); len(list) != 0 {
		t.Errorf("un rechazo no debe registrar nada, hay %d entradas", len(list))
	}

	t.Run("desde una subcarpeta del worktree enlazado", func(t *testing.T) {
		sub := filepath.Join(linked, "src")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := mgr.Register(sub, "Sub", "monorepo-embedded")
		if err == nil {
			t.Fatal("Register de una subcarpeta de un worktree enlazado debe fallar")
		}
		if got := quotedMainCheckout(t, err); !pathidentity.SameDirectory(got, main) {
			t.Errorf("el rechazo cita %q, se esperaba el checkout principal %q", got, main)
		}
	})
}

// Si el checkout principal no se puede deducir (el common dir no se llama `.git`), el rechazo
// lo dice y pide registrar el checkout principal del repositorio, sin inventar una ruta.
func TestRegisterRefusalWithoutAResolvableMainCheckout(t *testing.T) {
	base := t.TempDir()
	gitDir := filepath.Join(base, "almacen", "worktrees", "x")
	writeGitFile(t, filepath.Join(gitDir, "commondir"), "../..\n")
	linked := filepath.Join(base, "x")
	writeGitFile(t, filepath.Join(linked, ".git"), "gitdir: "+filepath.ToSlash(gitDir)+"\n")

	mgr, err := NewManager(filepath.Join(base, "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = mgr.Register(linked, "X", "monorepo-embedded")
	if err == nil {
		t.Fatal("Register de un worktree enlazado debe fallar")
	}
	msg := err.Error()
	for _, want := range []string{"worktree", "checkout principal", pathquote.Quote(linked)} {
		if !strings.Contains(msg, want) {
			t.Errorf("el rechazo debe contener %q: %v", want, err)
		}
	}
	if strings.Contains(msg, `axiom project add "`) {
		t.Errorf("no debe inventar un checkout principal que no se pudo resolver: %v", err)
	}
}

// REQ-25.31, escenario 3: un submódulo (`.git` fichero sin `commondir`) no se trata como
// worktree enlazado y se registra como cualquier checkout.
func TestRegisterAcceptsASubmodule(t *testing.T) {
	base := t.TempDir()
	super := filepath.Join(base, "super")
	if err := os.MkdirAll(filepath.Join(super, ".git", "modules", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(super, "sub")
	writeGitFile(t, filepath.Join(sub, ".git"), "gitdir: ../.git/modules/sub\n")

	mgr, err := NewManager(filepath.Join(base, "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Register(sub, "Sub", "monorepo-embedded"); err != nil {
		t.Fatalf("un submódulo no es un worktree enlazado: %v", err)
	}
}

// REQ-25.31, escenario 4: crear worktrees enlazados de un proyecto registrado no altera su
// registro, no añade proyectos y `Prune` no los toca.
func TestLinkedWorktreesDoNotAlterTheRegisteredProject(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	main := newGitRepository(t, filepath.Join(base, "axiom-wt", "proj"))
	mgr, err := NewManager(filepath.Join(base, "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	rec, err := mgr.Register(main, "Proj", "monorepo-embedded")
	if err != nil {
		t.Fatal(err)
	}

	for i, name := range []string{"uno", "dos"} {
		runGit(t, main, "worktree", "add", "-q", "-b", fmt.Sprintf("inc/%d", i), filepath.Join(base, name), "HEAD")
	}

	assertOnlyTheProject := func(stage string) {
		t.Helper()
		list, err := mgr.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].ID != rec.ID || !pathidentity.SameDirectory(list[0].Path, main) {
			t.Fatalf("%s: se esperaba solo el proyecto registrado %q, hay %v", stage, main, list)
		}
		cfg, err := mgr.Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.ActiveWorkspace != rec.ID {
			t.Errorf("%s: el proyecto activo es %q, se esperaba %q", stage, cfg.ActiveWorkspace, rec.ID)
		}
	}
	assertOnlyTheProject("tras crear los worktrees")

	pruned, err := mgr.Prune()
	if err != nil {
		t.Fatal(err)
	}
	if len(pruned) != 0 {
		t.Errorf("Prune no debe purgar el proyecto ni añadir nada, purgó %v", pruned)
	}
	assertOnlyTheProject("tras Prune")
}

// Escritura de las preferencias personales de worktree (INC-25, REQ-25.29, S7): persisten en
// la entrada del proyecto sin cambiar el esquema y se validan antes de tocar el hub.

func readHubBytes(t *testing.T, config string) []byte {
	t.Helper()
	raw, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// REQ-25.29, escenario «Persistencia de las preferencias»: el modo y la raíz quedan en la
// entrada del proyecto y se recuperan; la búsqueda admite el id y la ruta.
func TestSetWorktreePrefsPersistsAndIsReadBack(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	root := filepath.Join(t.TempDir(), "worktrees", "axiom")

	rec, err := manager.SetWorktreePrefs(project, WorktreePrefs{Mode: WorktreeModeAsk, Root: root})

	if err != nil {
		t.Fatalf("SetWorktreePrefs: %v", err)
	}
	if rec.WorktreeMode != WorktreeModeAsk || rec.WorktreeRoot != root {
		t.Errorf("la entrada devuelta = (%q, %q), se esperaba (ask, %q)", rec.WorktreeMode, rec.WorktreeRoot, root)
	}
	prefs, stored, err := manager.LookupWorktreePrefs(project)
	if err != nil || !stored || prefs != (WorktreePrefs{Mode: WorktreeModeAsk, Root: root}) {
		t.Errorf("LookupWorktreePrefs = (%+v, %v, %v), se esperaba (ask, %q) guardada", prefs, stored, err, root)
	}
	if raw := string(readHubBytes(t, config)); !strings.Contains(raw, `"worktree_mode": "ask"`) {
		t.Errorf("workspaces.json debe guardar el modo en la entrada:\n%s", raw)
	}

	t.Run("por id", func(t *testing.T) {
		if _, err := manager.SetWorktreePrefs(rec.ID, WorktreePrefs{Mode: WorktreeModeInPlace}); err != nil {
			t.Fatalf("SetWorktreePrefs por id: %v", err)
		}
		prefs, _, _ := manager.LookupWorktreePrefs(project)
		if prefs.Mode != WorktreeModeInPlace {
			t.Errorf("modo = %q, se esperaba in_place", prefs.Mode)
		}
	})

	t.Run("sustituye ambos campos y vacío es sin preferencia", func(t *testing.T) {
		if _, err := manager.SetWorktreePrefs(project, WorktreePrefs{}); err != nil {
			t.Fatalf("SetWorktreePrefs vacío: %v", err)
		}
		_, stored, err := manager.LookupWorktreePrefs(project)
		if err != nil || stored {
			t.Errorf("tras vaciar, stored = %v (err %v), se esperaba sin preferencia", stored, err)
		}
		if raw := string(readHubBytes(t, config)); strings.Contains(raw, "worktree_") {
			t.Errorf("vaciar las preferencias retira los campos del JSON:\n%s", raw)
		}
	})
}

// REQ-25.29, escenarios «Modo inválido rechazado» y «Raíz relativa rechazada»: el rechazo
// está en castellano y el fichero no cambia ni un byte.
func TestSetWorktreePrefsRejectsInvalidValuesWithoutChangingTheHub(t *testing.T) {
	tests := []struct {
		name  string
		prefs WorktreePrefs
		want  []string
	}{
		{"modo siempre", WorktreePrefs{Mode: "siempre"}, []string{"«siempre»", "worktree", "in_place", "ask"}},
		{"raíz relativa", WorktreePrefs{Root: "../wt"}, []string{"ruta absoluta", "../wt"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager, project, config := newPrefsHub(t)
			before := readHubBytes(t, config)

			_, err := manager.SetWorktreePrefs(project, tc.prefs)

			if err == nil {
				t.Fatal("se esperaba un error")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe contener %q", err, want)
				}
			}
			if after := readHubBytes(t, config); string(after) != string(before) {
				t.Error("un valor inválido no debe modificar el hub")
			}
		})
	}
}

func TestSetWorktreePrefsOfAnUnregisteredProjectSaysSo(t *testing.T) {
	manager, _, config := newPrefsHub(t)
	before := readHubBytes(t, config)
	other := filepath.Join(t.TempDir(), "sin-registrar")

	_, err := manager.SetWorktreePrefs(other, WorktreePrefs{Mode: WorktreeModeAsk})

	if err == nil || !strings.Contains(err.Error(), "no está registrado") || !strings.Contains(err.Error(), other) {
		t.Errorf("error = %v, se esperaba que dijera que el proyecto %q no está registrado", err, other)
	}
	if after := readHubBytes(t, config); string(after) != string(before) {
		t.Error("un proyecto desconocido no debe modificar el hub")
	}
}

// REQ-25.29, escenario «Re-registrar conserva las preferencias»: Register no cambia su
// firma ni toca lo guardado.
func TestRegisterAgainKeepsTheWorktreePrefs(t *testing.T) {
	manager, project, _ := newPrefsHub(t)
	if _, err := manager.SetWorktreePrefs(project, WorktreePrefs{Mode: WorktreeModeInPlace}); err != nil {
		t.Fatal(err)
	}

	rec, err := manager.Register(project, "mi-proyecto", "monorepo-embedded")

	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if rec.WorktreeMode != WorktreeModeInPlace {
		t.Errorf("la entrada devuelta perdió el modo: %q", rec.WorktreeMode)
	}
	if prefs, stored, _ := manager.LookupWorktreePrefs(project); !stored || prefs.Mode != WorktreeModeInPlace {
		t.Errorf("tras re-registrar, el modo = (%q, stored %v), se esperaba in_place guardado", prefs.Mode, stored)
	}
}

// REQ-25.29, escenario «Desregistrar elimina las preferencias».
func TestUnregisterRemovesTheWorktreePrefsWithTheEntry(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	root := filepath.Join(t.TempDir(), "wt")
	rec, err := manager.SetWorktreePrefs(project, WorktreePrefs{Mode: WorktreeModeAsk, Root: root})
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.Unregister(rec.ID); err != nil {
		t.Fatalf("Unregister: %v", err)
	}

	raw := string(readHubBytes(t, config))
	for _, gone := range []string{"worktree_mode", "worktree_root", "mi-proyecto"} {
		if strings.Contains(raw, gone) {
			t.Errorf("tras desregistrar, workspaces.json no debe contener %q:\n%s", gone, raw)
		}
	}
}

// REQ-25.29, escenario «Un binario anterior ignora las preferencias»: el JSON con los campos
// nuevos carga en un tipo sin ellos, con el esquema `"1.0"` y el resto de campos intactos.
func TestAnOlderBinaryIgnoresTheWorktreePrefs(t *testing.T) {
	manager, project, config := newPrefsHub(t)
	root := filepath.Join(t.TempDir(), "wt")
	rec, err := manager.SetWorktreePrefs(project, WorktreePrefs{Mode: WorktreeModeAsk, Root: root})
	if err != nil {
		t.Fatal(err)
	}

	// El registro tal y como lo conocía el binario anterior: sin los campos de preferencias.
	type olderRecord struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Path         string   `json:"path"`
		Topology     string   `json:"topology"`
		IsConfigured bool     `json:"is_configured"`
		Tech         []string `json:"tech,omitempty"`
	}
	var older struct {
		Version         string        `json:"version"`
		ActiveWorkspace string        `json:"active_workspace"`
		Workspaces      []olderRecord `json:"workspaces"`
	}
	if err := json.Unmarshal(readHubBytes(t, config), &older); err != nil {
		t.Fatalf("un binario anterior no puede cargar el hub: %v", err)
	}

	if older.Version != "1.0" {
		t.Errorf("version = %q, se esperaba 1.0", older.Version)
	}
	if len(older.Workspaces) != 1 {
		t.Fatalf("workspaces = %d, se esperaba 1", len(older.Workspaces))
	}
	got := older.Workspaces[0]
	if got.ID != rec.ID || got.Name != "mi-proyecto" || got.Path != rec.Path || got.Topology != rec.Topology {
		t.Errorf("el resto de campos se conserva: %+v frente a %+v", got, rec)
	}
	if older.ActiveWorkspace != rec.ID {
		t.Errorf("active_workspace = %q, se esperaba %q", older.ActiveWorkspace, rec.ID)
	}
}

// REQ-25.30: el registro con preferencias valida antes de registrar, aplica solo lo que no
// está vacío (re-registrar con el modo conserva la raíz guardada) y sin preferencias no
// escribe nada más que el registro.
func TestRegisterWithWorktreePrefs(t *testing.T) {
	t.Run("un valor inválido no registra nada", func(t *testing.T) {
		config := filepath.Join(t.TempDir(), "workspaces.json")
		manager, err := NewManager(config)
		if err != nil {
			t.Fatal(err)
		}
		project := filepath.Join(t.TempDir(), "p")
		if err := os.MkdirAll(project, 0o755); err != nil {
			t.Fatal(err)
		}

		_, err = manager.RegisterWithWorktreePrefs(project, "", "", WorktreePrefs{Mode: "siempre"})

		if err == nil || !strings.Contains(err.Error(), "«siempre»") {
			t.Errorf("error = %v, se esperaba el rechazo del modo", err)
		}
		if _, statErr := os.Stat(config); !os.IsNotExist(statErr) {
			t.Error("un valor inválido no debe crear ni modificar el hub")
		}
	})

	t.Run("sin preferencias solo registra", func(t *testing.T) {
		manager, project, config := newPrefsHub(t)

		rec, err := manager.RegisterWithWorktreePrefs(project, "mi-proyecto", "", WorktreePrefs{})

		if err != nil {
			t.Fatal(err)
		}
		if rec.WorktreeMode != "" || rec.WorktreeRoot != "" {
			t.Errorf("sin preferencias no se guarda ninguna: %+v", rec)
		}
		if strings.Contains(string(readHubBytes(t, config)), "worktree_") {
			t.Error("sin preferencias el JSON no lleva los campos nuevos")
		}
	})

	t.Run("fusiona: el modo conserva la raíz guardada", func(t *testing.T) {
		manager, project, _ := newPrefsHub(t)
		root := filepath.Join(t.TempDir(), "wt")
		if _, err := manager.RegisterWithWorktreePrefs(project, "mi-proyecto", "", WorktreePrefs{Root: root}); err != nil {
			t.Fatal(err)
		}

		rec, err := manager.RegisterWithWorktreePrefs(project, "mi-proyecto", "", WorktreePrefs{Mode: WorktreeModeAsk})

		if err != nil {
			t.Fatal(err)
		}
		if rec.WorktreeMode != WorktreeModeAsk || rec.WorktreeRoot != root {
			t.Errorf("la entrada = (%q, %q), se esperaba (ask, %q)", rec.WorktreeMode, rec.WorktreeRoot, root)
		}
	})
}

// REQ-25.30: Init valida las preferencias antes de escribir ni registrar nada y, con ellas
// válidas, las guarda en la entrada del hub.
func TestInitAppliesAndValidatesTheWorktreePrefs(t *testing.T) {
	config := filepath.Join(t.TempDir(), "workspaces.json")
	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	initializer := NewInitializer(manager, nil)

	t.Run("un valor inválido no escribe nada", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "nuevo")

		_, err := initializer.Init(InitOptions{Path: target, WorktreePrefs: WorktreePrefs{Mode: "siempre"}})

		if err == nil || !strings.Contains(err.Error(), "«siempre»") {
			t.Fatalf("error = %v, se esperaba el rechazo del modo", err)
		}
		if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
			t.Error("un valor inválido no debe crear el directorio destino")
		}
		if _, statErr := os.Stat(config); !os.IsNotExist(statErr) {
			t.Error("un valor inválido no debe crear el hub")
		}
	})

	t.Run("un valor válido se guarda en la entrada", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "otro")
		root := filepath.Join(t.TempDir(), "wt")

		res, err := initializer.Init(InitOptions{Path: target, WorktreePrefs: WorktreePrefs{Mode: WorktreeModeAsk, Root: root}})

		if err != nil {
			t.Fatal(err)
		}
		if res.Record.WorktreeMode != WorktreeModeAsk || res.Record.WorktreeRoot != root {
			t.Errorf("registro = (%q, %q), se esperaba (ask, %q)", res.Record.WorktreeMode, res.Record.WorktreeRoot, root)
		}
		if prefs, stored, err := manager.LookupWorktreePrefs(target); err != nil || !stored || prefs.Mode != WorktreeModeAsk {
			t.Errorf("LookupWorktreePrefs = (%+v, %v, %v), se esperaba ask guardada", prefs, stored, err)
		}
	})
}

func TestManagerWorktreeIsolationAndPrune(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "workspaces.json")
	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("error creando manager: %v", err)
	}

	// Proyecto normal válido
	validProj := filepath.Join(tempDir, "valid-proj")
	_ = os.MkdirAll(validProj, 0755)
	recValid, err := mgr.Register(validProj, "Valid Proj", "monorepo-embedded")
	if err != nil {
		t.Fatalf("error registrando proyecto válido: %v", err)
	}

	// Intentar registrar un worktree debe fallar
	wtProj, _ := syntheticLinkedWorktree(t, tempDir, "fake-wt")

	_, err = mgr.Register(wtProj, "Worktree Proj", "monorepo-embedded")
	if err == nil {
		t.Fatalf("esperado error al registrar un worktree, pero tuvo éxito")
	}
	if !strings.Contains(err.Error(), "worktree git") {
		t.Errorf("mensaje de error inesperado: %v", err)
	}

	// Probar Prune con una entrada huérfana inyectada manualmente
	cfg, _ := mgr.Load()
	cfg.Workspaces = append(cfg.Workspaces, WorkspaceRecord{
		ID:   "deleted-proj",
		Name: "Deleted Proj",
		Path: filepath.Join(tempDir, "non-existent-dir"),
	}, WorkspaceRecord{
		ID:   "stranded-wt",
		Name: "Stranded Worktree",
		Path: filepath.Join(tempDir, "axiom-wt", "stranded"),
	})
	_ = mgr.Save(cfg)

	pruned, err := mgr.Prune()
	if err != nil {
		t.Fatalf("error en Prune(): %v", err)
	}
	if len(pruned) != 2 {
		t.Errorf("esperado 2 proyectos purgados, obtenidos %d: %v", len(pruned), pruned)
	}

	list, _ := mgr.List()
	if len(list) != 1 || list[0].ID != recValid.ID {
		t.Errorf("esperado solo 1 proyecto válido restante, obtenidos: %v", list)
	}
}
