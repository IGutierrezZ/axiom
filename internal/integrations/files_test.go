package integrations

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/fileprim"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

func TestLockPathLivesInTheGitCommonDir(t *testing.T) {
	f := newGitWorkspace(t)

	got, err := f.Folder().lockPath()
	if err != nil {
		t.Fatalf("lockPath: %v", err)
	}
	wantDir := filepath.Join(f.WS, ".git", "axiom", "integrations", "v1", "locks")
	if !pathidentity.SameDirectory(filepath.Dir(got), wantDir) && filepath.Dir(got) != wantDir {
		t.Fatalf("el cerrojo debe vivir en %s, está en %s", wantDir, filepath.Dir(got))
	}
	if filepath.Base(got) != "x.lock" {
		t.Fatalf("el cerrojo se llama como el cambio: %s", got)
	}
}

func TestLockPathIsSharedByLinkedWorktrees(t *testing.T) {
	f := newGitWorkspace(t)
	wtParent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(wtParent, "wt")
	runGit(t, f.WS, "worktree", "add", "-q", "-b", "feat/x", wt)
	wtRoot := filepath.Join(wt, "openspec", "changes", "x")
	if err := os.MkdirAll(wtRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	main, err := f.Folder().lockPath()
	if err != nil {
		t.Fatal(err)
	}
	linked, err := (Folder{WorkspaceRoot: wt, Root: wtRoot}).lockPath()
	if err != nil {
		t.Fatalf("lockPath en el worktree enlazado: %v", err)
	}
	if !sameLockPath(main, linked) {
		t.Fatalf("clon y worktree enlazado comparten el cerrojo del common dir:\n%s\n%s", main, linked)
	}
}

func TestLockPathFallsBackToTheChangeFolderWithoutGitForOrdinaryFolders(t *testing.T) {
	f := newWorkspace(t)
	got, err := f.Folder().lockPath()
	if err != nil {
		t.Fatalf("lockPath sin git: %v", err)
	}
	if want := filepath.Join(f.Root, ".integrations.lock"); got != want {
		t.Fatalf("lockPath = %s, quería %s", got, want)
	}
}

func TestLockPathDoesNotFallBackForArchivedFolders(t *testing.T) {
	f := newWorkspace(t)
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	if _, err := archived.lockPath(); !errors.Is(err, ErrArchivedWithoutGit) {
		t.Fatalf("sin git no se encola en la archivada: %v", err)
	}
	if _, err := archived.acquire(); !errors.Is(err, ErrArchivedWithoutGit) {
		t.Fatalf("acquire sin git en la archivada: %v", err)
	}
}

func TestArchivedFolderSharesTheLockOfItsChange(t *testing.T) {
	f := newGitWorkspace(t)
	archived := archivedFolderOf(t, f, "2026-11-19-x")
	if got := archived.Change(); got != "x" {
		t.Fatalf("Change() de la archivada = %q", got)
	}
	if !archived.Archived() || f.Folder().Archived() {
		t.Fatal("Archived() distingue la carpeta archivada de la ordinaria")
	}
	ordinary, _ := f.Folder().lockPath()
	arch, err := archived.lockPath()
	if err != nil {
		t.Fatal(err)
	}
	if ordinary != arch {
		t.Fatalf("la carpeta archivada comparte el cerrojo del cambio:\n%s\n%s", ordinary, arch)
	}
}

func TestAcquireSerializesAndReleases(t *testing.T) {
	f := newGitWorkspace(t)
	release, err := f.Folder().acquire()
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	var slept atomic.Int32
	original := sleepBeforeLockRetry
	sleepBeforeLockRetry = func(time.Duration) {
		if slept.Add(1) == 3 {
			release() // el primero libera mientras el segundo reintenta
		}
	}
	t.Cleanup(func() { sleepBeforeLockRetry = original })

	second, err := f.Folder().acquire()
	if err != nil {
		t.Fatalf("el segundo acquire debe tener éxito tras reintentar: %v", err)
	}
	second()
	if slept.Load() < 3 {
		t.Fatalf("debía reintentar con la costura sleepBeforeLockRetry, durmió %d veces", slept.Load())
	}
}

func TestAcquireGivesUpAfterTheAttemptBudget(t *testing.T) {
	f := newGitWorkspace(t)
	release, err := f.Folder().acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	var slept atomic.Int32
	original := sleepBeforeLockRetry
	sleepBeforeLockRetry = func(time.Duration) { slept.Add(1) }
	t.Cleanup(func() { sleepBeforeLockRetry = original })

	_, err = f.Folder().acquire()
	if err == nil || !errors.Is(err, fileprim.ErrLockContended) {
		t.Fatalf("con el cerrojo ocupado debe agotar los intentos: %v", err)
	}
	if slept.Load() != lockAcquireAttempts {
		t.Fatalf("debía dormir %d veces, durmió %d", lockAcquireAttempts, slept.Load())
	}
}

func TestAcquireReturnsNonContentionErrorsImmediately(t *testing.T) {
	f := newGitWorkspace(t)
	boom := errors.New("fallo del sistema de ficheros")
	original := acquireIntegrationsLock
	acquireIntegrationsLock = func(string) (*fileprim.Lock, error) { return nil, boom }
	t.Cleanup(func() { acquireIntegrationsLock = original })

	if _, err := f.Folder().acquire(); !errors.Is(err, boom) {
		t.Fatalf("un error que no es contención se devuelve sin reintentar: %v", err)
	}
}

func TestLockRetryBackoffIsBoundedAndNeverBelowTheBase(t *testing.T) {
	original := randomInt63n
	randomInt63n = func(n int64) int64 { return n - 1 } // jitter máximo
	t.Cleanup(func() { randomInt63n = original })
	for attempt := 0; attempt < 300; attempt++ {
		d := lockRetryBackoff(attempt)
		if d < lockRetryDelay || d > lockMaxRetryDelay {
			t.Fatalf("intento %d: %s fuera de [%s, %s]", attempt, d, lockRetryDelay, lockMaxRetryDelay)
		}
	}
	randomInt63n = func(int64) int64 { return 0 } // sin jitter
	if got := lockRetryBackoff(0); got != lockRetryDelay {
		t.Fatalf("el primer intento espera el retardo base, esperó %s", got)
	}
	if got := lockRetryBackoff(250); got != lockMaxRetryDelay {
		t.Fatalf("tras crecer, el retardo toca el tope sin jitter: %s", got)
	}
}

func TestWriteAtomicKeepsPreviousContentWhenTheReplaceFails(t *testing.T) {
	f := newWorkspace(t)
	folder := f.Folder()
	if err := folder.writeAtomic(LinksFile, linksTempPattern, []byte("antiguo\n")); err != nil {
		t.Fatalf("primera escritura: %v", err)
	}

	boom := errors.New("fallo al reemplazar")
	original := replaceIntegrationsFile
	replaceIntegrationsFile = func(string, string) error { return boom }
	t.Cleanup(func() { replaceIntegrationsFile = original })

	err := folder.writeAtomic(LinksFile, linksTempPattern, []byte("nuevo\n"))
	if !errors.Is(err, boom) {
		t.Fatalf("el fallo del reemplazo se propaga: %v", err)
	}
	if got := readTestFile(t, folder.path(LinksFile)); got != "antiguo\n" {
		t.Fatalf("el fichero conserva su contenido anterior íntegro, tiene %q", got)
	}
	entries, _ := os.ReadDir(f.Root)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".integrations-") {
			t.Errorf("no debe quedar ningún temporal: %s", e.Name())
		}
	}
}

func TestWriteAtomicReplacesAndSyncsTheDirectory(t *testing.T) {
	f := newWorkspace(t)
	var synced []string
	original := syncIntegrationsDir
	syncIntegrationsDir = func(dir string) error { synced = append(synced, dir); return nil }
	t.Cleanup(func() { syncIntegrationsDir = original })

	if err := f.Folder().writeAtomic(LinksFile, linksTempPattern, []byte("uno\n")); err != nil {
		t.Fatal(err)
	}
	if err := f.Folder().writeAtomic(LinksFile, linksTempPattern, []byte("dos\n")); err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, f.Folder().path(LinksFile)); got != "dos\n" {
		t.Fatalf("contenido = %q", got)
	}
	if len(synced) != 2 || synced[0] != f.Root {
		t.Fatalf("debe sincronizar el directorio tras cada reemplazo: %v", synced)
	}
}

func TestWriteAtomicAndRemoveGoThroughTheArchivedGuard(t *testing.T) {
	f := newWorkspace(t)
	archived := archivedFolderOf(t, f, "2026-11-19-x")

	for _, name := range []string{LinksFile, "gates.yaml", ".gates.lock", ".otro"} {
		if err := archived.writeAtomic(name, outboxTempPattern, []byte("x\n")); err == nil || !strings.Contains(err.Error(), "RF-ARC-03") {
			t.Errorf("escribir %s en la archivada se rechaza con el mensaje de RF-ARC-03: %v", name, err)
		}
		if err := archived.remove(name); err == nil || !strings.Contains(err.Error(), "RF-ARC-03") {
			t.Errorf("borrar %s en la archivada se rechaza con el mensaje de RF-ARC-03: %v", name, err)
		}
	}
	if err := archived.writeAtomic(LinksFile, linksTempPattern, []byte("x\n")); err == nil {
		t.Error("el temporal de integrations.yaml tampoco se autoriza en la archivada")
	}

	if err := archived.writeAtomic(outboxName, outboxTempPattern, []byte("cola\n")); err != nil {
		t.Fatalf("la cola sí se escribe en la archivada: %v", err)
	}
	if got := readTestFile(t, archived.path(outboxName)); got != "cola\n" {
		t.Fatalf("cola = %q", got)
	}
	if err := archived.remove(outboxName); err != nil {
		t.Fatalf("la cola sí se borra en la archivada: %v", err)
	}
	if fileExists(archived.path(outboxName)) {
		t.Fatal("la cola debe haberse borrado")
	}
	if err := archived.remove(outboxName); err != nil {
		t.Fatalf("borrar lo que no existe no es un error: %v", err)
	}
}

func TestGitStatusShowsNoLockNorQueueOnceTheQueueIsEmpty(t *testing.T) {
	f := newGitWorkspace(t)
	folder := f.Folder()

	release, err := folder.acquire()
	if err != nil {
		t.Fatal(err)
	}
	if err := folder.writeAtomic(outboxName, outboxTempPattern, []byte("cola\n")); err != nil {
		t.Fatal(err)
	}
	release()
	if status := runGit(t, f.WS, "status", "--porcelain", "--untracked-files=all"); !strings.Contains(status, ".integrations-outbox.yaml") {
		t.Fatalf("con la cola presente el árbol está sucio:\n%s", status)
	}
	if err := folder.remove(outboxName); err != nil {
		t.Fatal(err)
	}
	if status := runGit(t, f.WS, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("vacía la cola, ni cerrojo ni cola ensucian el árbol:\n%s", status)
	}
}

// sameLockPath compara dos rutas de cerrojo que aún no existen en disco (así que no se puede
// comparar por identidad): igualdad textual, sin distinguir mayúsculas en Windows.
func sameLockPath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
