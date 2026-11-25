package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckEngramReachable_ProjectMCPApproval cubre el caso real del 2026-11-25:
// tras retirar el Engram de usuario de Claude Code, el único que quedaba era el de
// <raíz>/.mcp.json, que Claude Code no había aprobado (enabledMcpjsonServers solo
// tenía context7), y el doctor daba pass. Si el Engram que usaría Claude Code solo
// está en .mcp.json y sin aprobar, engram:reachable avisa (warn, no fail). El
// doctor solo lee: nunca aprueba nada.
func TestCheckEngramReachable_ProjectMCPApproval(t *testing.T) {
	const projectMCP = `{"mcpServers":{"engram":{"command":"engram","args":["mcp","--tools=agent"]}}}`

	cases := []struct {
		name          string
		localSettings string // .claude/settings.local.json del proyecto ("" = no existe)
		sharedProject string // .claude/settings.json del proyecto
		userSettings  string // ~/.claude/settings.json
		userScope     bool   // ~/.claude.json declara mcpServers.engram
		wantStatus    CheckStatus
	}{
		{name: "sin aprobación registrada", wantStatus: CheckStatusWarn},
		{name: "aprobado solo context7 (caso real)", localSettings: `{"enabledMcpjsonServers":["context7"]}`, wantStatus: CheckStatusWarn},
		{name: "engram aprobado en settings.local.json", localSettings: `{"enabledMcpjsonServers":["context7","engram"]}`, wantStatus: CheckStatusPass},
		{name: "todos aprobados en el settings.json del proyecto", sharedProject: `{"enableAllProjectMcpServers":true}`, wantStatus: CheckStatusPass},
		{name: "engram aprobado en el settings.json de usuario", userSettings: `{"enabledMcpjsonServers":["engram"]}`, wantStatus: CheckStatusPass},
		{name: "rechazado gana a aprobado", localSettings: `{"enabledMcpjsonServers":["engram"],"disabledMcpjsonServers":["engram"]}`, wantStatus: CheckStatusWarn},
		{name: "registro de usuario presente", userScope: true, wantStatus: CheckStatusPass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(engramHealthEnvVar, "")
			setStdioProbeForTest(t, nil)
			homeDir := t.TempDir()
			workspaceDir := t.TempDir()
			mcpPath := filepath.Join(workspaceDir, ".mcp.json")
			writeApprovalFixture(t, mcpPath, projectMCP)
			localPath := filepath.Join(workspaceDir, ".claude", "settings.local.json")
			writeApprovalFixture(t, localPath, tc.localSettings)
			writeApprovalFixture(t, filepath.Join(workspaceDir, ".claude", "settings.json"), tc.sharedProject)
			writeApprovalFixture(t, filepath.Join(homeDir, ".claude", "settings.json"), tc.userSettings)
			if tc.userScope {
				writeDoctorEngramConfig(t, homeDir, "engram", []string{"mcp", "--tools=agent"})
			}

			got := checkEngramReachable(context.Background(), homeDir, workspaceDir, []string{"claude-code"})

			if got.Status != tc.wantStatus {
				t.Fatalf("engram:reachable status = %q, detail = %q; want %q", got.Status, got.Detail, tc.wantStatus)
			}
			if tc.wantStatus == CheckStatusWarn {
				for _, want := range []string{mcpPath, "not approved", "enabledMcpjsonServers"} {
					if !strings.Contains(got.Detail, want) {
						t.Fatalf("detail = %q, want it to mention %q", got.Detail, want)
					}
				}
				if got.Remedy == nil {
					t.Fatalf("remedy = nil, want how to approve it")
				}
			}
			if tc.localSettings != "" {
				after, err := os.ReadFile(localPath)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(after, []byte(tc.localSettings)) {
					t.Fatalf("settings.local.json changed to %s; the doctor must only read", after)
				}
			}
		})
	}
}

// TestCheckEngramReachable_ProjectMCPPendingApprovalOnlyForClaude comprueba que el
// aviso es de Claude Code: otro agente con su Engram de proyecto no lo provoca.
func TestCheckEngramReachable_ProjectMCPPendingApprovalOnlyForClaude(t *testing.T) {
	t.Setenv(engramHealthEnvVar, "")
	setStdioProbeForTest(t, nil)
	workspaceDir := t.TempDir()
	writeApprovalFixture(t, filepath.Join(workspaceDir, ".mcp.json"), `{"mcpServers":{"engram":{"command":"engram","args":["mcp"]}}}`)
	writeApprovalFixture(t, filepath.Join(workspaceDir, ".gemini", "settings.json"), `{"mcpServers":{"engram":{"command":"engram","args":["mcp"]}}}`)

	got := checkEngramReachable(context.Background(), t.TempDir(), workspaceDir, []string{"gemini-cli"})

	if got.Status == CheckStatusWarn && strings.Contains(got.Detail, "not approved") {
		t.Fatalf("engram:reachable = %q (%q); the approval warning only applies to Claude Code", got.Status, got.Detail)
	}
}

func writeApprovalFixture(t *testing.T, path, content string) {
	t.Helper()
	if content == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
