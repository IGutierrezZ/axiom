package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	validateEmbeddedBase = "workspace:\n  name: ejemplo-embebido\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	validateBaseline     = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n"
	validateContext      = "technical_context:\n  maintained:\n    entry: context/TECHNICAL_CONTEXT.md\n    paths: [context/]\n  reference:\n    paths: [00_Glosario.md]\n"
)

// validateWorkspace monta un proyecto embebido con la declaración dada, el
// documento de la baseline y, si config no está vacío, openspec/config.yaml.
func validateWorkspace(t *testing.T, blocks, openSpecConfig string) string {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), validateEmbeddedBase+blocks)
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "03_Modelo_Datos.md"), "# modelo\n")
	if openSpecConfig != "" {
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), openSpecConfig)
	}
	return ws.Master
}

func wantOutput(t *testing.T, output string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, output)
		}
	}
}

// workspace-topology REQ-1.2 y REQ-24.17: una declaración válida y compatible
// con el almacén finaliza con COMPLIANT y código 0.
func TestWorkspaceValidateAcceptsAValidDeclaration(t *testing.T) {
	for name, openSpecConfig := range map[string]string{
		"sin almacen declarado": "",
		"almacen openspec":      "artifact_store: openspec\n",
		"almacen hybrid":        "artifact_store: hybrid\n",
	} {
		t.Run(name, func(t *testing.T) {
			master := validateWorkspace(t, validateBaseline+validateContext, openSpecConfig)
			workspacetest.WriteFile(t, filepath.Join(master, "context", "TECHNICAL_CONTEXT.md"), "# ctx\n")
			workspacetest.WriteFile(t, filepath.Join(master, "00_Glosario.md"), "# glosario\n")
			output, code := runAxiom(t, "workspace", "validate", "--path", master)
			if code != 0 {
				t.Fatalf("workspace validate salió con %d:\n%s", code, output)
			}
			wantOutput(t, output, "COMPLIANT")
			if strings.Contains(output, "Advertencias") {
				t.Errorf("sin rutas inexistentes no debe haber advertencias:\n%s", output)
			}
		})
	}
}

// REQ-24.27 y workspace-topology REQ-1.2: una ruta de contexto inexistente solo
// avisa, y las advertencias se imprimen también con COMPLIANT.
func TestWorkspaceValidatePrintsWarningsWithCompliant(t *testing.T) {
	master := validateWorkspace(t, validateContext, "")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("una ruta inexistente no altera el código de salida; salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "Advertencias:", "'context/' de 'technical_context.maintained.paths' no existe todavía", "'00_Glosario.md'", "COMPLIANT")
	if strings.Contains(output, "NON-COMPLIANT") {
		t.Errorf("el resultado debe ser conforme:\n%s", output)
	}
	if strings.Index(output, "Advertencias:") > strings.Index(output, "Resultado: COMPLIANT") {
		t.Errorf("las advertencias van antes del resultado:\n%s", output)
	}
}

// REQ-24.14 y REQ-24.17: un error por cada ruta inválida, con código 1.
func TestWorkspaceValidateReportsOneErrorPerInvalidPath(t *testing.T) {
	blocks := "baseline:\n  documents:\n    - domain: uno\n      path: ../fuera.md\n    - domain: dos\n      path: /abs/dos.md\n    - domain: tres\n      path: tres.md\n"
	master := validateWorkspace(t, blocks, "")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 1 {
		t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
	}
	wantOutput(t, output,
		"No se pudo cargar la configuración de Axiom",
		"'../fuera.md' sale de workspaceRoot",
		"'/abs/dos.md'",
		"debe ser relativa a workspaceRoot",
	)
	lines := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "el dominio 'uno'") || strings.Contains(line, "el dominio 'dos'") {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("se esperaba una línea por ruta inválida, hubo %d:\n%s", lines, output)
	}
}

// REQ-24.17: una baseline sin documentos o con un perfil sin grupos finaliza con 1.
func TestWorkspaceValidateRejectsAnInvalidBaseline(t *testing.T) {
	for name, tc := range map[string]struct {
		blocks string
		wants  []string
	}{
		"sin documentos":         {"baseline: {}\n", []string{"la baseline declarada requiere al menos un documento"}},
		"clave desconocida":      {"baseline:\n  documentss: []\n", []string{"clave desconocida 'documentss'", "claves admitidas: documents"}},
		"perfil sin los grupos":  {"baseline:\n  documents:\n    - domain: rf\n      path: rf.md\n      requirement:\n        heading: '^### (RF-\\d+): (.+)$'\n        level: 3\n        render: '### {id}: {title}'\n", []string{"el dominio 'rf'", "los grupos con nombre 'id' y 'title'"}},
		"identidad desconocida":  {"baseline:\n  documents:\n    - domain: rf\n      path: rf.md\n      requirement:\n        heading: '^### (?P<id>RF-\\d+): (?P<title>.+)$'\n        level: 3\n        identity: nombre\n        render: '### {id}: {title}'\n", []string{"el dominio 'rf'", "'id' o 'text', no 'nombre'"}},
		"contexto sin rutas":     {"technical_context:\n  maintained:\n    paths: []\n", []string{"requiere al menos una ruta"}},
		"modo desconocido":       {"technical_context:\n  readonly:\n    paths: [a.md]\n", []string{"clave desconocida 'readonly'", "claves admitidas: maintained, reference"}},
		"nivel con include":      {"baseline:\n  documents:\n    - domain: rf\n      path: rf.md\n      requirement:\n        heading: '^#### (?P<id>RF-\\d+): (?P<title>.+)$'\n        level: 4\n        render: '#### {id}: {title}'\n", []string{"el dominio 'rf'", "'scenarios: omit' o un 'level' menor o igual que 3"}},
		"baseline y contexto":    {"baseline: {}\ntechnical_context: {}\n", []string{"la baseline declarada requiere", "requiere al menos el modo"}},
		"ruta compartida (modo)": {"baseline:\n  documents:\n    - domain: a\n      path: a.md\ntechnical_context:\n  reference:\n    paths: [a.md]\n", []string{"solapa con el documento 'a.md'"}},
	} {
		t.Run(name, func(t *testing.T) {
			master := validateWorkspace(t, tc.blocks, "")
			output, code := runAxiom(t, "workspace", "validate", "--path", master)
			if code != 1 {
				t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
			}
			wantOutput(t, output, tc.wants...)
		})
	}
}

// REQ-24.14: un documento declarado que no existe es un error nombrando el
// dominio y la ruta; los demás documentos válidos no lo ocultan.
func TestWorkspaceValidateReportsAMissingDocument(t *testing.T) {
	blocks := "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: ausente\n      path: docs/ausente.md\n"
	master := validateWorkspace(t, blocks, "")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 1 {
		t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
	}
	wantOutput(t, output, "NO CONFORME", "el documento 'docs/ausente.md' del dominio 'ausente' no existe", "NON-COMPLIANT")
	if strings.Contains(output, "dominio 'modelo-de-datos'") {
		t.Errorf("el documento existente no debe reportarse:\n%s", output)
	}
}

// REQ-24.62: con almacén declarado engram o none, un error por declaración
// afectada con la continuación, y código 1.
func TestWorkspaceValidateRejectsAFileDeclarationWithAFilelessStore(t *testing.T) {
	for _, store := range []string{"engram", "none"} {
		t.Run(store, func(t *testing.T) {
			master := validateWorkspace(t, validateBaseline+"technical_context:\n  maintained:\n    paths: [context/]\n", "artifact_store: "+store+"\n")
			workspacetest.WriteFile(t, filepath.Join(master, "context", "a.md"), "# ctx\n")
			output, code := runAxiom(t, "workspace", "validate", "--path", master)
			if code != 1 {
				t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
			}
			wantOutput(t, output,
				"la declaración 'baseline' exige un almacén de artefactos de ficheros",
				"la declaración 'technical_context.maintained' exige un almacén de artefactos de ficheros",
				"artifact_store '"+store+"'",
				"Declara 'artifact_store: openspec' o 'hybrid' en openspec/config.yaml",
			)
		})
	}

	t.Run("solo contexto de referencia no exige almacen de ficheros", func(t *testing.T) {
		master := validateWorkspace(t, "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n", "artifact_store: engram\n")
		workspacetest.WriteFile(t, filepath.Join(master, "00_Glosario.md"), "# glosario\n")
		output, code := runAxiom(t, "workspace", "validate", "--path", master)
		if code != 0 {
			t.Fatalf("salió con %d, se esperaba 0:\n%s", code, output)
		}
	})
}

// El espacio canónico: --path es la carpeta maestra, los documentos se resuelven
// contra el repositorio de especificaciones que apunta el puntero.
func TestWorkspaceValidateResolvesDeclarationsAgainstTheSpecsRepository(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"),
		"workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"+
			validateBaseline+"technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "03_Modelo_Datos.md"), "# modelo\n")

	output, code := runAxiom(t, "workspace", "validate", "--path", ws.Master)
	if code != 0 {
		t.Fatalf("salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT", "Advertencias:", "'../repo-backend/docs/' de 'technical_context.maintained.paths'")

	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "docs", "arquitectura.md"), "# arq\n")
	output, code = runAxiom(t, "workspace", "validate", "--path", ws.Master)
	if code != 0 || strings.Contains(output, "Advertencias") {
		t.Fatalf("con la ruta mantenida existente no debe haber advertencias (salió con %d):\n%s", code, output)
	}
}

// workspace-topology REQ-25.25 (escenario «La validación termina con código 1 y
// la corrección») y REQ-1.2 (escenario «Topología retirada finaliza con error
// accionable»): la topología retirada no se acepta y el mensaje nombra multirepo.
func TestWorkspaceValidateRejectsTheRetiredTopology(t *testing.T) {
	master := t.TempDir()
	workspacetest.WriteFile(t, filepath.Join(master, "axiom.yaml"),
		"workspace:\n  name: ejemplo-retirado\n  topology: monorepo-decoupled\n  specs_repository: especificacion\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n")
	workspacetest.MkdirAll(t, filepath.Join(master, "especificacion"))

	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 1 {
		t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
	}
	wantOutput(t, output,
		"No se pudo cargar la configuración de Axiom",
		"la topología `monorepo-decoupled` fue retirada: sustitúyela por `multirepo` en `workspace.topology`",
	)
	if strings.Contains(output, "COMPLIANT") && !strings.Contains(output, "NON-COMPLIANT") {
		t.Errorf("la topología retirada no puede dar un resultado conforme:\n%s", output)
	}
}

// Sin bloques nuevos el comando se comporta como antes.
func TestWorkspaceValidateWithoutDeclarationsIsUnchanged(t *testing.T) {
	master := validateWorkspace(t, "", "artifact_store: none\n")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT")
	if strings.Contains(output, "Advertencias") {
		t.Errorf("sin declaraciones no hay advertencias:\n%s", output)
	}
}

// requireRealGit omite la prueba con -short o sin git instalado: los repositorios
// de estas pruebas se crean con `git init` real.
func requireRealGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
}

// outsideAnyGitRepository omite la prueba si el directorio temporal del sistema
// está dentro de un repositorio git: una ruta «que no es un repositorio» no sería
// fiable en esa máquina.
func outsideAnyGitRepository(t *testing.T, dir string) {
	t.Helper()
	for current := dir; ; {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			t.Skipf("el directorio temporal %s está dentro de un repositorio git", dir)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return
		}
		current = parent
	}
}

// workspace-topology REQ-1.2 (escenario «Bloque `git:` o `repositories:` inválido
// finaliza con error»), REQ-25.26 y REQ-25.27: un bloque inválido, de forma o de
// sistema de ficheros, termina con código 1 y detalla en castellano la clave o la
// ruta y la regla incumplida.
func TestWorkspaceValidateRejectsAnInvalidGitOrRepositoriesBlock(t *testing.T) {
	for name, tc := range map[string]struct {
		blocks string
		wants  []string
	}{
		"prefijo inválido": {"git:\n  branch_prefix: \"a b//\"\n",
			[]string{"No se pudo cargar la configuración de Axiom", "git.branch_prefix", "'a b//'"}},
		"rama de integración inválida": {"git:\n  integration_branch: \"a b:c\"\n",
			[]string{"git.integration_branch", "'a b:c'", "espacio"}},
		"clave desconocida en git": {"git:\n  worktree_mode: worktree\n",
			[]string{"clave desconocida 'worktree_mode'", "claves admitidas: integration_branch, branch_prefix"}},
		"ruta que sale de la carpeta maestra": {"repositories:\n  - path: ../fuera\n",
			[]string{"repositories[0].path", "'../fuera'", "sale de la carpeta maestra"}},
		"ruta absoluta": {"repositories:\n  - path: /opt/repo\n",
			[]string{"repositories[0].path", "'/opt/repo'", "debe ser relativa a la carpeta maestra"}},
		"clave desconocida en una entrada": {"repositories:\n  - path: .\n    worktree_mode: worktree\n",
			[]string{"clave desconocida 'worktree_mode'", "'repositories[0]'"}},
		"dos entradas con la misma ruta": {"repositories:\n  - path: app\n  - path: ./app\n",
			[]string{"repositories[0] ('app')", "repositories[1] ('./app')", "misma ruta 'app'"}},
		"ruta inexistente": {"repositories:\n  - path: repo-inexistente\n",
			[]string{"NO CONFORME", "repositories[0].path", "'repo-inexistente'", "no existe como directorio", "NON-COMPLIANT"}},
		"git y repositories inválidos a la vez": {"git:\n  branch_prefix: \"inc//\"\nrepositories:\n  - path: ../fuera\n",
			[]string{"git.branch_prefix", "'inc//'", "repositories[0].path", "'../fuera'"}},
	} {
		t.Run(name, func(t *testing.T) {
			master := validateWorkspace(t, tc.blocks, "")
			output, code := runAxiom(t, "workspace", "validate", "--path", master)
			if code != 1 {
				t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
			}
			wantOutput(t, output, tc.wants...)
			if strings.Contains(output, "COMPLIANT") && !strings.Contains(output, "NON-COMPLIANT") {
				t.Errorf("un bloque inválido no puede dar un resultado conforme:\n%s", output)
			}
		})
	}
}

// REQ-25.27 (escenario «Dos entradas de la misma raíz git»): dos rutas de un mismo
// repositorio real son un error que nombra ambas y la raíz compartida.
func TestWorkspaceValidateRejectsTwoEntriesOfTheSameGitRoot(t *testing.T) {
	requireRealGit(t)
	master := validateWorkspace(t, "repositories:\n  - path: app/web\n  - path: app/api\n", "")
	workspacetest.MkdirAll(t, filepath.Join(master, "app", "web"))
	workspacetest.MkdirAll(t, filepath.Join(master, "app", "api"))
	workspacetest.InitGit(t, filepath.Join(master, "app"))

	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 1 {
		t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
	}
	wantOutput(t, output, "NO CONFORME", "'app/web'", "'app/api'", "mismo repositorio git", filepath.Join(master, "app"), "NON-COMPLIANT")
}

// REQ-1.2 y REQ-25.27: bloques `git:` y `repositories:` válidos, con repositorios
// reales, terminan con COMPLIANT, sin advertencias y con código 0.
func TestWorkspaceValidateAcceptsValidGitAndRepositoriesBlocks(t *testing.T) {
	requireRealGit(t)
	master := validateWorkspace(t, "git:\n  integration_branch: develop\n  branch_prefix: inc\nrepositories:\n  - path: repo-backend\n    integration_branch: release\n  - path: repo-frontend\n    branch_prefix: ui\n", "")
	workspacetest.MkdirAll(t, filepath.Join(master, "repo-backend"))
	workspacetest.MkdirAll(t, filepath.Join(master, "repo-frontend"))
	workspacetest.InitGit(t, filepath.Join(master, "repo-backend"))
	workspacetest.InitGit(t, filepath.Join(master, "repo-frontend"))

	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT")
	if strings.Contains(output, "Advertencias") {
		t.Errorf("con repositorios reales no hay advertencias:\n%s", output)
	}
}

// REQ-25.27 (escenario «Ruta que no es un repositorio git solo avisa»): una ruta
// que existe pero no pertenece a ningún repositorio es una advertencia, con
// COMPLIANT y código 0.
func TestWorkspaceValidateOnlyWarnsAboutAPathThatIsNotAGitRepository(t *testing.T) {
	master := validateWorkspace(t, "repositories:\n  - path: repo-docs\n", "")
	outsideAnyGitRepository(t, master)
	workspacetest.MkdirAll(t, filepath.Join(master, "repo-docs"))

	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("una ruta que no es un repositorio git no altera el código de salida; salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "Advertencias:", "'repo-docs'", "repositories[0].path", "ningún repositorio git", "COMPLIANT")
	if strings.Contains(output, "NON-COMPLIANT") {
		t.Errorf("el resultado debe ser conforme:\n%s", output)
	}
}

// workspace-topology REQ-25.24 (escenario «`axiom.yaml` de este repositorio
// declara `git:`») y decisión T5: el axiom.yaml versionado declara de forma
// explícita `integration_branch: main` y `branch_prefix: inc`, y supera la
// validación con COMPLIANT. La prueba lee el fichero con una ruta relativa al
// paquete.
func TestRepositoryAxiomYamlDeclaresItsGitPolicyAndIsCompliant(t *testing.T) {
	root := filepath.Join("..", "..")
	configPath := filepath.Join(root, "axiom.yaml")

	declarations := workspace.ReadDeclarations(configPath)
	if declarations.ParseErr != nil || declarations.GitErr != nil {
		t.Fatalf("el bloque git: del axiom.yaml del repositorio debe ser válido: parse=%v git=%v", declarations.ParseErr, declarations.GitErr)
	}
	want := workspace.GitConfig{IntegrationBranch: "main", BranchPrefix: "inc"}
	if declarations.Git == nil || *declarations.Git != want {
		t.Fatalf("el axiom.yaml del repositorio debe declarar %+v (declarado, no por defecto), y declara %+v", want, declarations.Git)
	}
	cfg, err := workspace.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig(%s): %v", configPath, err)
	}
	if cfg.Git == nil || *cfg.Git != want {
		t.Errorf("la política resuelta es %+v, se esperaba %+v", cfg.Git, want)
	}

	output, code := runAxiom(t, "workspace", "validate", "--path", root)
	if code != 0 {
		t.Fatalf("workspace validate sobre el repositorio salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT")
	if strings.Contains(output, "NON-COMPLIANT") {
		t.Errorf("el repositorio debe ser conforme:\n%s", output)
	}
}

// addLinkedWorktree crea con git real un worktree enlazado de repo en worktree,
// sobre la rama nueva branch. Hace falta un commit: se crea uno vacío con una
// identidad fija y sin la configuración del desarrollador.
func addLinkedWorktree(t *testing.T, repo, worktree, branch string) {
	t.Helper()
	for _, args := range [][]string{
		{"-c", "user.name=Prueba", "-c", "user.email=prueba@example.invalid", "add", "-A"},
		{"-c", "user.name=Prueba", "-c", "user.email=prueba@example.invalid", "commit", "-q", "--allow-empty", "-m", "inicial"},
		{"worktree", "add", "-q", "-b", branch, worktree},
	} {
		command := exec.Command("git", append([]string{"-C", repo}, args...)...)
		command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v en %s: %v\n%s", args, repo, err, output)
		}
	}
}

// workspace-planning-home REQ-25.28 (escenario «La validación desde un worktree
// externo no da falsos errores»): un worktree de repo-backend fuera de la carpeta
// maestra, sin marcador propio, valida contra la carpeta maestra resuelta desde su
// checkout principal y termina con COMPLIANT. Es el arnés de S9-c1: el CLI se
// ejecuta en un subproceso del binario de pruebas (runAxiom), sin compilar nada.
func TestWorkspaceValidateFromAnExternalLinkedWorktreeIsCompliant(t *testing.T) {
	requireRealGit(t)
	ws := workspacetest.Canonical(t)
	workspacetest.InitGit(t, ws.Backend)
	worktree := filepath.Join(t.TempDir(), "x", "repo-backend")
	addLinkedWorktree(t, ws.Backend, worktree, "inc/x")

	output, code := runAxiom(t, "workspace", "validate", "--path", worktree)
	if code != 0 {
		t.Fatalf("salió con %d, se esperaba 0:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT", "Proyecto: ejemplo-canonico", "worktree enlazado")
	if strings.Contains(output, "NON-COMPLIANT") || strings.Contains(output, "No se pudo cargar") {
		t.Errorf("la validación desde el worktree no debe dar falsos errores:\n%s", output)
	}

	// La misma carpeta maestra, desde el checkout principal, da el mismo resultado.
	direct, directCode := runAxiom(t, "workspace", "validate", "--path", ws.Master)
	if directCode != 0 {
		t.Fatalf("desde la carpeta maestra salió con %d:\n%s", directCode, direct)
	}
	wantOutput(t, direct, "Directorio base: "+ws.Master, "Total rutas verificadas: 4")
	wantOutput(t, output, "Directorio base: "+ws.Master, "Total rutas verificadas: 4")
}

// Un worktree sin marcador propio y sin planning home en su checkout principal no
// se valida contra nada: el rechazo de siempre, con código 1.
func TestWorkspaceValidateFromALinkedWorktreeWithoutAnyMarkerKeepsTheError(t *testing.T) {
	requireRealGit(t)
	repo := t.TempDir()
	outsideAnyGitRepository(t, repo)
	workspacetest.InitGit(t, repo)
	worktree := filepath.Join(t.TempDir(), "x")
	addLinkedWorktree(t, repo, worktree, "inc/x")

	output, code := runAxiom(t, "workspace", "validate", "--path", worktree)
	if code != 1 {
		t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
	}
	wantOutput(t, output, "No se pudo cargar la configuración de Axiom")
}

// REQ-25.28: `--path` sigue siendo la carpeta maestra cuando tiene marcador propio,
// también en un worktree enlazado de un monorepo embebido: no se redirige al
// checkout principal.
func TestWorkspaceValidateKeepsPathAsTheMasterWhenTheWorktreeHasItsOwnMarker(t *testing.T) {
	requireRealGit(t)
	master := validateWorkspace(t, "", "")
	workspacetest.InitGit(t, master)
	worktree := filepath.Join(t.TempDir(), "x", "repo")
	addLinkedWorktree(t, master, worktree, "inc/x")

	output, code := runAxiom(t, "workspace", "validate", "--path", worktree)
	if code != 0 {
		t.Fatalf("salió con %d, se esperaba 0:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT", "Directorio base: "+worktree)
	if strings.Contains(output, "worktree enlazado") {
		t.Errorf("con marcador propio no se redirige a la carpeta maestra del checkout principal:\n%s", output)
	}
}

// D-15: LocateAt sigue evaluando solo el directorio dado, también desde un
// worktree enlazado sin marcador (lo usan las llamadas que ya conocen la raíz).
func TestLocateAtDoesNotLeaveAnUnmarkedLinkedWorktree(t *testing.T) {
	requireRealGit(t)
	ws := workspacetest.Canonical(t)
	workspacetest.InitGit(t, ws.Backend)
	worktree := filepath.Join(t.TempDir(), "x", "repo-backend")
	addLinkedWorktree(t, ws.Backend, worktree, "inc/x")

	loc, err := workspace.LocateAt(worktree)
	if err != nil {
		t.Fatalf("LocateAt(%q) = %v", worktree, err)
	}
	if loc.Origin != workspace.OriginStart || len(loc.Notes) != 0 {
		t.Fatalf("Location = %+v, LocateAt solo evalúa el directorio dado", loc)
	}
	if !strings.EqualFold(filepath.Clean(loc.WorkspaceRoot), filepath.Clean(worktree)) {
		t.Fatalf("WorkspaceRoot = %q, se esperaba el propio worktree %q", loc.WorkspaceRoot, worktree)
	}
}
