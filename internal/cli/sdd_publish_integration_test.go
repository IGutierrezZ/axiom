package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
)

const archivedFolderName = "2026-11-19-x"

// archiveProject monta el repositorio de `newPublishProject` con el bloque dado (y el doble
// instalado) y deja el cambio x movido a su carpeta archivada, en la rama de archivado. Si
// withLinks, el cambio llevaba un integrations.yaml versionado.
func archiveProject(t *testing.T, block string, withLinks bool) (repo, archived string, fake *integrationstest.Fake) {
	t.Helper()
	repo = newBranchRepository(t)
	writeBranchConfig(t, repo, block)
	for name, content := range map[string]string{"proposal.md": "propuesta\n", "design.md": "diseño\n"} {
		writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", name), content)
	}
	if withLinks {
		writePublishFile(t, filepath.Join(repo, "openspec", "changes", "x", "integrations.yaml"), "schema: axiom.integrations/v1\nchange: x\nlinks:\n  - plugin: fake\n    kind: work_item\n    id: \"7\"\n    source: gate.spec.approved/x/dabc\n    recorded_at: 2026-11-18T22:30:00Z\n")
	}
	branchTestGit(t, repo, "add", "-A")
	branchTestGit(t, repo, "commit", "-q", "-m", "cambio x")
	bare := t.TempDir()
	branchTestGit(t, bare, "init", "-q", "--bare", "-b", "main")
	branchTestGit(t, repo, "remote", "add", "origin", bare)
	branchTestGit(t, repo, "push", "-q", "origin", "main")
	branchTestGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	changes := filepath.Join(repo, "openspec", "changes")
	if err := os.MkdirAll(filepath.Join(changes, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	archived = filepath.Join(changes, "archive", archivedFolderName)
	if err := os.Rename(filepath.Join(changes, "x"), archived); err != nil {
		t.Fatal(err)
	}
	fake = integrationstest.NewFake()
	integrationstest.Install(t, fake)
	quiet := writerBranchWarnings
	writerBranchWarnings = discardWriter{}
	t.Cleanup(func() { writerBranchWarnings = quiet })
	return repo, archived, fake
}

func publishArchive(t *testing.T, repo string) publishJSON {
	t.Helper()
	stdout, err := runPublish(t, "--cwd", repo, "--change", "x", "--archive", "--json")
	if err != nil {
		t.Fatalf("RunSDDPublish --archive: %v\n%s", err, stdout)
	}
	return decodePublish(t, stdout)
}

// comparablePublish deja de un resultado lo que no depende del repositorio concreto.
func comparablePublish(p publishJSON) publishJSON {
	p.Repository = nil
	if p.Commit != nil {
		p.Commit = &struct {
			SHA   string   `json:"sha"`
			Paths []string `json:"paths"`
		}{Paths: p.Commit.Paths}
	}
	return p
}

func TestSDDPublishArchiveIntegrationDeliversArchiveAndKeepsThePublishResult(t *testing.T) {
	baseRepo, _, _ := archiveProject(t, "", false)
	base := comparablePublish(publishArchive(t, baseRepo))

	repo, _, fake := archiveProject(t, fakeBlock, false)
	warnings := captureIntegrationWarnings(t)
	got := comparablePublish(publishArchive(t, repo))

	if !reflect.DeepEqual(got, base) {
		t.Fatalf("el resultado, el código y el JSON de publish no cambian con la integración:\n%+v\n%+v", got, base)
	}
	if keys := fake.DeliveredKeys(); len(keys) != 1 || keys[0] != "archive/x/"+archivedFolderName {
		t.Fatalf("el doble recibe archive con su clave: %v", keys)
	}
	if warnings.Len() != 0 {
		t.Fatalf("una entrega correcta no avisa: %q", warnings)
	}
}

func TestSDDPublishArchiveIntegrationFailureQueuesInTheArchivedRootAndTouchesNothingVersioned(t *testing.T) {
	repo, archived, fake := archiveProject(t, fakeBlock, true)
	fake.FailAlways(errors.New("tracker caído"))
	warnings := captureIntegrationWarnings(t)
	linksBefore, _ := os.ReadFile(filepath.Join(archived, "integrations.yaml"))

	published := publishArchive(t, repo)
	if published.Status != "published" || published.Context != "archive" {
		t.Fatalf("publicación = %+v", published)
	}

	queue := filepath.Join(archived, ".integrations-outbox.yaml")
	if !fileExistsCLI(queue) {
		t.Fatal("la cola queda en la raíz de la carpeta archivada")
	}
	if after, _ := os.ReadFile(filepath.Join(archived, "integrations.yaml")); !bytes.Equal(after, linksBefore) {
		t.Fatal("integrations.yaml archivado no cambia byte a byte")
	}
	status := publishGitOutput(t, repo, "status", "--porcelain", "--untracked-files=all")
	if strings.TrimSpace(status) != "?? openspec/changes/archive/"+archivedFolderName+"/.integrations-outbox.yaml" {
		t.Fatalf("solo la cola, sin seguimiento, ensucia el árbol:\n%s", status)
	}
	if !strings.Contains(warnings.String(), "axiom integrations sync") {
		t.Fatalf("el aviso nombra sync: %q", warnings)
	}

	// Un publish --archive repetido con la cola pendiente no la confirma.
	again := publishArchive(t, repo)
	if again.Status != "up_to_date" || again.Commit != nil {
		t.Fatalf("segunda publicación = %+v", again)
	}
	if !fileExistsCLI(queue) {
		t.Fatal("la cola sigue pendiente")
	}
}

func TestSDDPublishArchiveIntegrationDiscardsLinksReturnedByArchive(t *testing.T) {
	repo, archived, fake := archiveProject(t, fakeBlock, true)
	fake.ReturnLinks(integrations.Link{Kind: "work_item", ID: "99"})
	warnings := captureIntegrationWarnings(t)
	linksBefore, _ := os.ReadFile(filepath.Join(archived, "integrations.yaml"))

	publishArchive(t, repo)

	if after, _ := os.ReadFile(filepath.Join(archived, "integrations.yaml")); !bytes.Equal(after, linksBefore) {
		t.Fatal("los vínculos devueltos por archive no se escriben")
	}
	if !strings.Contains(warnings.String(), "descart") {
		t.Fatalf("se descartan con un aviso por stderr: %q", warnings)
	}
	if fileExistsCLI(filepath.Join(archived, ".integrations-outbox.yaml")) {
		t.Fatal("entregado, la cola se borra")
	}
}

func TestSDDPublishArchiveIntegrationRefusedOrWithoutArchiveEmitsNothing(t *testing.T) {
	repo, archived, fake := archiveProject(t, fakeBlock, false)

	// Sin --archive en la rama de archivado: se rechaza y no se emite.
	if _, err := runPublish(t, "--cwd", repo, "--change", "x", "--json"); err == nil {
		t.Fatal("sin --archive la rama de archivado se rechaza")
	}
	// Con --archive pero en una rama que no es la de archivado: rechazo tipado.
	branchTestGit(t, repo, "switch", "-q", "main")
	if _, err := runPublish(t, "--cwd", repo, "--change", "x", "--archive", "--json"); err == nil {
		t.Fatal("--archive fuera de la rama de archivado se rechaza")
	}
	if fake.Calls() != 0 || fileExistsCLI(filepath.Join(archived, ".integrations-outbox.yaml")) {
		t.Fatalf("un publish rechazado no emite ni crea cola (llamadas %d)", fake.Calls())
	}
}

func TestSDDPublishArchiveIntegrationSyncDeliversAndRemovesTheArchivedQueue(t *testing.T) {
	for _, withChange := range []bool{true, false} {
		repo, archived, fake := archiveProject(t, fakeBlock, false)
		fake.FailAlways(errors.New("tracker caído"))
		captureIntegrationWarnings(t)
		publishArchive(t, repo)
		queue := filepath.Join(archived, ".integrations-outbox.yaml")
		if !fileExistsCLI(queue) {
			t.Fatal("la cola archivada debía quedar pendiente")
		}

		fake.Recover()
		args := []string{"sync", "--cwd", repo}
		if withChange {
			args = append(args, "--change", "x")
		}
		code, _, errOut := runIntegrationsCLI(t, "", args...)
		if code != 0 {
			t.Fatalf("sync (--change=%v): %d %q", withChange, code, errOut)
		}
		if fileExistsCLI(queue) {
			t.Fatalf("sync (--change=%v) entrega y borra la cola archivada", withChange)
		}
		if keys := fake.DeliveredKeys(); len(keys) != 1 || keys[0] != "archive/x/"+archivedFolderName {
			t.Fatalf("entregas = %v", keys)
		}
	}
}

func TestSDDPublishArchiveIntegrationWithoutGitDoesNotQueueAndNamesEmit(t *testing.T) {
	root, _, fake := integrationsWorkspace(t, fakeBlock)
	archived := filepath.Join(root, "openspec", "changes", "archive", archivedFolderName)
	if err := os.MkdirAll(archived, 0o755); err != nil {
		t.Fatal(err)
	}
	warnings := captureIntegrationWarnings(t)

	emitArchiveEvent(locateIntegrations(t, root), archived)

	if fileExistsCLI(filepath.Join(archived, ".integrations-outbox.yaml")) || fake.Calls() != 0 {
		t.Fatal("sin git no se encola ni se entrega")
	}
	if !strings.Contains(warnings.String(), "axiom integrations emit") || !strings.Contains(warnings.String(), "--event archive") {
		t.Fatalf("el aviso deja la orden emit: %q", warnings)
	}
}
