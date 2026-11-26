package model

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func codexHelperCommand(ctx context.Context, mode, output string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCodexDiscoveryHelperProcess$", "--")
	cmd.Env = append(os.Environ(), "GO_WANT_CODEX_HELPER=1", "GO_CODEX_HELPER_MODE="+mode, "GO_CODEX_HELPER_OUTPUT="+output)
	return cmd
}

func TestCodexDiscoveryHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CODEX_HELPER") != "1" {
		return
	}

	switch os.Getenv("GO_CODEX_HELPER_MODE") {
	case "error":
		os.Exit(1)
	case "wait":
		select {}
	case "output-limit":
		_, _ = os.Stdout.WriteString(strings.Repeat("x", codexModelDiscoveryOutputLimit+1))
	default:
		_, _ = os.Stdout.WriteString(os.Getenv("GO_CODEX_HELPER_OUTPUT"))
	}
	os.Exit(0)
}
