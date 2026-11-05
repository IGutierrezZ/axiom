//go:build windows

package opencode

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"github.com/IGutierrezZ/axiom/v3/internal/testenv"
)

// TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows proves
// that a grandchild inheriting stdout and stderr does not extend discovery past
// the context deadline: cancellation terminates the whole Job Object tree, so
// pipes close as soon as the deadline fires instead of staying open for the
// grandchild's 30s sleep.
func TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows(t *testing.T) {
	testenv.RequireBinaryBuilds(t)
	dir := t.TempDir()
	helper := filepath.Join(dir, "descendant-helper.exe")
	source := filepath.Join(dir, "main.go")
	src := `package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "grandchild" {
		time.Sleep(30 * time.Second)
		return
	}
	grandchild := exec.Command(os.Args[0], "grandchild")
	grandchild.Stdout = os.Stdout
	grandchild.Stderr = os.Stderr
	_ = grandchild.Start()
	fmt.Println("custom/model")
	fmt.Println("{\"id\":\"model\",\"name\":\"Model\",\"capabilities\":{\"toolcall\":true}}")
	time.Sleep(25 * time.Millisecond)
}
`
	if err := os.WriteFile(source, []byte(src), 0o600); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	buildCmd := exec.Command("go", "build", "-o", helper, source)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\noutput: %s", err, string(out))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, _ = runCatalogCommand(ctx, Command{Path: helper, OutputLimit: 1 << 20})
	elapsed := time.Since(started)

	if elapsed > 10*time.Second {
		t.Fatalf("discovery stayed blocked for %v; want bounded by the 300ms deadline", elapsed)
	}
}

// TestCatalogDescendantHelperProcess no es una prueba: es el cuerpo del hijo
// (parent) y del nieto (grandchild) que lanzan las pruebas siguientes, sin
// compilar ningún ejecutable: se relanza el propio binario de test.
func TestCatalogDescendantHelperProcess(t *testing.T) {
	switch os.Getenv("AXIOM_CATALOG_DESCENDANT") {
	case "parent":
		// Margen para que el padre asigne este proceso al Job Object antes de
		// que nazca el nieto, que hereda la pertenencia al job.
		time.Sleep(300 * time.Millisecond)
		grandchild := exec.Command(os.Args[0], "-test.run=^TestCatalogDescendantHelperProcess$")
		grandchild.Env = append(os.Environ(), "AXIOM_CATALOG_DESCENDANT=grandchild")
		grandchild.Stdout, grandchild.Stderr = os.Stdout, os.Stderr
		_ = grandchild.Start()
		time.Sleep(time.Minute)
	case "grandchild":
		_ = os.WriteFile(os.Getenv("AXIOM_CATALOG_PIDFILE"), []byte(strconv.Itoa(os.Getpid())), 0o600)
		time.Sleep(time.Minute)
	}
}

// grandchildPID espera a que el nieto publique su PID.
func grandchildPID(t *testing.T, pidFile string) uint32 {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(string(data)); err == nil && pid > 0 {
				return uint32(pid)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("el nieto no llegó a arrancar")
	return 0
}

func processExited(pid uint32) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return true
	}
	defer windows.CloseHandle(h)
	event, _ := windows.WaitForSingleObject(h, 0)
	return event == windows.WAIT_OBJECT_0
}

// TestCatalogCancellationLeavesNoOrphanGrandchildWindows prueba, con las rutas
// reales de DiscoverCatalog (tubería y fichero), que cancelar un hijo que lanzó
// un nieto no deja al nieto vivo.
func TestCatalogCancellationLeavesNoOrphanGrandchildWindows(t *testing.T) {
	for _, tt := range []struct {
		name   string
		major  RuntimeMajor
		runner CatalogRunner
	}{
		{"streamCatalogCommand", RuntimeV1, streamCatalogCommand},
		{"fileCatalogCommand", RuntimeV2, fileCatalogCommand},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
			runner := func(ctx context.Context, _ Command) (io.Reader, error) {
				return tt.runner(ctx, Command{
					Path: os.Args[0],
					Args: []string{"-test.run=^TestCatalogDescendantHelperProcess$"},
					Env:  []string{"AXIOM_CATALOG_DESCENDANT=parent", "AXIOM_CATALOG_PIDFILE=" + pidFile},
				})
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := discoverCatalogForMajor(ctx, "project", tt.major, runner)
				result <- err
			}()

			pid := grandchildPID(t, pidFile)
			if processExited(pid) {
				t.Fatal("el nieto debía estar vivo antes de cancelar")
			}
			cancel()
			select {
			case err := <-result:
				wantCatalogKind(t, err, CatalogErrorTimeout)
			case <-time.After(15 * time.Second):
				t.Fatal("el descubrimiento siguió bloqueado tras cancelar")
			}
			for deadline := time.Now().Add(5 * time.Second); !processExited(pid); {
				if time.Now().After(deadline) {
					t.Fatalf("el nieto %d sigue vivo tras cancelar el descubrimiento", pid)
				}
				time.Sleep(20 * time.Millisecond)
			}
		})
	}
}
