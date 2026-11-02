package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Los hooks del registro de skills se instalan en la máquina del usuario y corren
// en cualquier repositorio. Estas pruebas fijan que fuera de un proyecto Axiom (un
// axiom.yaml en el directorio o en un ancestro, o un puntero .axiom-workspace que
// lleva a uno) el refresco no escribe nada.

// writeAxiomYAML marca dir como raíz de un proyecto Axiom.
func writeAxiomYAML(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte("workspace:\n  name: Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// foreignRepo monta un repositorio git que no es de Axiom: con skills de
// proyecto, un AGENTS.md del usuario con una sección ## Skills escrita a mano y un
// openspec/ de otra herramienta. Devuelve el repositorio y los bytes de AGENTS.md
// que el refresco no debe tocar.
func foreignRepo(t *testing.T) (string, []byte) {
	t.Helper()
	repo := t.TempDir()
	for _, dir := range []string{".git", "openspec/changes/x", "skills/demo"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	skill := "---\nname: demo\ndescription: \"Trigger: demo.\"\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(repo, "skills", "demo", "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	agents := []byte("# Reglas del repositorio\n\n## Skills\n\nSección escrita a mano.\n")
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), agents, 0o644); err != nil {
		t.Fatal(err)
	}
	return repo, agents
}

// assertNothingWritten comprueba que el refresco fue inerte: sin .atl, sin
// .gitignore, con los bytes de AGENTS.md intactos y sin ningún tema en Engram.
func assertNothingWritten(t *testing.T, repo string, agents []byte, mirrorCalls *int) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repo, ".atl")); !os.IsNotExist(err) {
		t.Fatalf("fuera de un proyecto Axiom no se debe crear .atl: stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("fuera de un proyecto Axiom no se debe crear .gitignore: stat err = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, agents) {
		t.Fatalf("fuera de un proyecto Axiom no se debe tocar AGENTS.md, ahora es %q", got)
	}
	if *mirrorCalls != 0 {
		t.Fatalf("fuera de un proyecto Axiom no se debe escribir en Engram, llamadas al espejo = %d", *mirrorCalls)
	}
}

// El comando exacto de los hooks, en un repositorio ajeno, sale con 0 sin imprimir
// nada y sin escribir en ningún sitio.
func TestSkillRegistryRefreshIsInertInAForeignRepository(t *testing.T) {
	setFakeHome(t, t.TempDir())
	calls := stubMirror(t, false)
	repo, agents := foreignRepo(t)

	var buf bytes.Buffer
	// Con y sin --no-gitignore: la guarda va delante del paso del .gitignore.
	for _, args := range [][]string{
		{"--quiet", "--no-gitignore", "--cwd", repo},
		{"--quiet", "--cwd", repo},
	} {
		if err := runSkillRegistryRefresh(args, &buf); err != nil {
			t.Fatalf("refresh %v debe omitirse con salida 0, error: %v", args, err)
		}
		if buf.Len() != 0 {
			t.Fatalf("refresh %v no debe imprimir nada, imprimió %q", args, buf.String())
		}
	}
	assertNothingWritten(t, repo, agents, calls)
}

func TestSkillIndexRefreshIsInertInAForeignRepository(t *testing.T) {
	setFakeHome(t, t.TempDir())
	calls := stubMirror(t, false)
	repo, agents := foreignRepo(t)

	var stdout, stderr bytes.Buffer
	if code := RunSkillIndex([]string{"refresh", "--force", "--cwd", repo}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "axiom.yaml") || !strings.Contains(out, repo) || strings.Count(out, "\n") != 1 {
		t.Fatalf("el aviso debe ser una línea que nombre el axiom.yaml ausente y la ruta, fue %q", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertNothingWritten(t, repo, agents, calls)
}

func TestSkillRegistryRefreshIsInertInAPlainFolder(t *testing.T) {
	setFakeHome(t, t.TempDir())
	calls := stubMirror(t, false)
	plain := t.TempDir()
	notes := []byte("# Notas\n")
	if err := os.WriteFile(filepath.Join(plain, "AGENTS.md"), notes, 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runSkillRegistryRefresh([]string{"--quiet", "--cwd", plain}, &buf); err != nil {
		t.Fatalf("refresh en una carpeta sin git debe omitirse, error: %v", err)
	}
	assertNothingWritten(t, plain, notes, calls)
}

func TestSkillRegistryRefreshIsInertWithOnlyOpenSpec(t *testing.T) {
	setFakeHome(t, t.TempDir())
	calls := stubMirror(t, false)
	repo := t.TempDir()
	for _, dir := range []string{"openspec/changes/x", ".git"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	notes := []byte("# Notas\n")
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), notes, 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runSkillRegistryRefresh([]string{"--quiet", "--cwd", repo}, &buf); err != nil {
		t.Fatalf("refresh con solo openspec/ debe omitirse, error: %v", err)
	}
	assertNothingWritten(t, repo, notes, calls)
}

// La otra mitad del contrato: con axiom.yaml, el registro, la sección gestionada
// de AGENTS.md y el espejo de Engram se comportan como antes.
func TestSkillIndexRefreshStillWritesInAnAxiomProject(t *testing.T) {
	setFakeHome(t, t.TempDir())
	calls := stubMirror(t, false)
	repo, agents := foreignRepo(t)
	writeAxiomYAML(t, repo)

	var stdout, stderr bytes.Buffer
	if code := RunSkillIndex([]string{"refresh", "--no-gitignore", "--cwd", repo}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0 (stdout=%q stderr=%q)", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".atl", "skill-registry.md")); err != nil {
		t.Fatalf("el registro debe escribirse en un proyecto Axiom: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, agents) {
		t.Fatalf("la sección gestionada ## Skills de AGENTS.md debe refrescarse en un proyecto Axiom")
	}
	if *calls == 0 {
		t.Fatalf("el espejo de Engram debe ejecutarse en un proyecto Axiom")
	}
}

// Un axiom.yaml por encima del directorio que resolvió el hook (la carpeta de un
// servicio con su propio checkout git) basta.
func TestSkillRegistryRefreshInASubdirectoryOfAnAxiomProject(t *testing.T) {
	setFakeHome(t, t.TempDir())
	stubMirror(t, false)
	project := t.TempDir()
	writeAxiomYAML(t, project)
	sub := filepath.Join(project, "services", "api")
	if err := os.MkdirAll(filepath.Join(sub, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runSkillRegistryRefresh([]string{"--quiet", "--no-gitignore", "--cwd", sub}, &buf); err != nil {
		t.Fatalf("refresh en un subdirectorio de un proyecto Axiom debe continuar, error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sub, ".atl", "skill-registry.md")); err != nil {
		t.Fatalf("el registro debe escribirse en el subdirectorio: %v", err)
	}
}

// Un repositorio de código de un multirepo no lleva axiom.yaml propio: solo el
// puntero .axiom-workspace de la carpeta maestra lleva hasta el del repositorio de
// especificaciones.
func TestSkillRegistryRefreshFollowsTheMultirepoPointer(t *testing.T) {
	setFakeHome(t, t.TempDir())
	stubMirror(t, false)
	ws := workspacetest.Canonical(t)
	if err := os.MkdirAll(filepath.Join(ws.Backend, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runSkillRegistryRefresh([]string{"--quiet", "--no-gitignore", "--cwd", ws.Backend}, &buf); err != nil {
		t.Fatalf("refresh en un repositorio de código de un multirepo debe continuar, error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.Backend, ".atl", "skill-registry.md")); err != nil {
		t.Fatalf("el registro debe escribirse a través del puntero: %v", err)
	}
}
