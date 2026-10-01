//go:build windows

package opencode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows proves
// that a grandchild inheriting stdout and stderr does not extend discovery past
// the context deadline: cancellation terminates the whole Job Object tree, so
// pipes close as soon as the deadline fires instead of staying open for the
// grandchild's 30s sleep.
func TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows(t *testing.T) {
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
