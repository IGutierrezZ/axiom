package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pruebas de `axiom branch guard` (REQ-25.21, REQ-25.1): el verbo, sus errores de uso y el
// protocolo de salida de cada runtime con git real, entorno aislado y la carga simulada por la
// entrada estándar. La decisión de la guarda tiene su batería en internal/branch/engine; aquí se
// prueba lo que añade la CLI. Llevan el prefijo propio `TestBranchGuard` para filtrarlas sin
// arrastrar las pruebas ajenas que contienen «Guard» (algunas de revisión tardan minutos).

// runBranchGuardCLI ejecuta `axiom branch guard` con la entrada estándar simulada.
func runBranchGuardCLI(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = runBranch(append([]string{"guard"}, args...), strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

// branchGuardProject es un proyecto embebido de un rol, con git real, en la rama main.
func branchGuardProject(t *testing.T) string {
	t.Helper()
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, "")
	return repo
}

func guardPayloads(t *testing.T, repo string) map[string]string {
	t.Helper()
	document := filepath.Join(repo, "openspec", "changes", "x", "proposal.md")
	marshal := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	return map[string]string{
		"claude-code": marshal(map[string]any{"cwd": repo, "hook_event_name": "PreToolUse", "tool_name": "Write", "tool_input": map[string]any{"file_path": document}}),
		"codex": marshal(map[string]any{"cwd": repo, "hook_event_name": "PreToolUse", "tool_name": "apply_patch", "tool_input": map[string]any{
			"command": "*** Begin Patch\n*** Add File: " + document + "\n+hola\n*** End Patch\n",
		}}),
		"opencode": marshal(map[string]any{"event": "pre", "tool": "write", "filePath": document, "cwd": repo}),
	}
}

func TestBranchGuardHelpDescribesTheHookAndExitsZero(t *testing.T) {
	t.Parallel()
	for _, flagName := range []string{"--help", "-h"} {
		code, stdout, stderr := runBranchGuardCLI(t, "", flagName)

		if code != 0 || stderr != "" {
			t.Errorf("%s: código %d, stderr %q; se esperaba 0 y stderr vacío", flagName, code, stderr)
		}
		for _, want := range []string{"--agent <runtime>", "claude-code, codex u opencode", "falla en\nabierto", "256 KiB", "axiom branch ensure", "Código 2", "opencode"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("%s: la ayuda debe contener %q:\n%s", flagName, want, stdout)
			}
		}
	}
}

// Un error de uso sale con el código 2 de la familia y, como todo rechazo, nombra la continuación;
// no llega a leer la entrada estándar.
func TestBranchGuardRejectsUsageErrorsWithCode2WithoutReadingTheInput(t *testing.T) {
	t.Parallel()
	cases := map[string][]string{
		"sin --agent":                    {},
		"--agent vacío":                  {"--agent", ""},
		"--agent desconocido":            {"--agent", "vim"},
		"--agent de otro runtime (pi)":   {"--agent", "pi"},
		"argumento suelto":               {"--agent", "codex", "sobra"},
		"bandera desconocida":            {"--agent", "codex", "--json"},
		"--agent sin valor":              {"--agent"},
		"--agent con mayúsculas":         {"--agent", "Claude-Code"},
		"--agent con espacios alrededor": {"--agent", " codex "},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var out, errOut bytes.Buffer
			code := runBranch(append([]string{"guard"}, args...), failingReader{t: t}, &out, &errOut)

			if code != 2 || out.Len() != 0 {
				t.Errorf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, out.String())
			}
			if !strings.HasPrefix(errOut.String(), "Error: ") || !strings.Contains(errOut.String(), "axiom branch guard --help") {
				t.Errorf("stderr %q debe empezar por «Error: » y nombrar `axiom branch guard --help`", errOut.String())
			}
		})
	}
}

// failingReader es una entrada estándar que no debe leerse.
type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Error("un error de uso no debe leer la entrada estándar")
	return 0, io.EOF
}

// REQ-25.21, escenarios «Bloquea un documento de Axiom en rama equivocada» y «Permite un documento
// de Axiom en la rama correcta», con el protocolo de salida de cada runtime: Claude Code y Codex
// salen con 2 y el motivo por stderr; OpenCode siempre con 0 y el JSON por stdout.
func TestBranchGuardBlocksADocumentOnTheWrongBranchWithTheProtocolOfEachRuntime(t *testing.T) {
	repo := branchGuardProject(t)
	payloads := guardPayloads(t, repo)

	for _, agent := range []string{"claude-code", "codex", "opencode"} {
		t.Run(agent+" en main", func(t *testing.T) {
			code, stdout, stderr := runBranchGuardCLI(t, payloads[agent], "--agent", agent)
			if agent == "opencode" {
				var out struct{ Action, Message string }
				if err := json.Unmarshal([]byte(stdout), &out); err != nil || code != 0 || stderr != "" || out.Action != "block" ||
					!strings.Contains(out.Message, "axiom branch ensure --cwd ") || !strings.Contains(out.Message, "--change x") {
					t.Fatalf("código %d, stdout %q, stderr %q; se esperaba 0 y un bloqueo con la continuación de ensure", code, stdout, stderr)
				}
				return
			}
			if code != 2 || stdout != "" || !strings.Contains(stderr, "axiom branch ensure --cwd ") || !strings.Contains(stderr, "--change x") {
				t.Fatalf("código %d, stdout %q, stderr %q; se esperaba 2 y el motivo con la continuación por stderr", code, stdout, stderr)
			}
		})
	}

	branchTestGit(t, repo, "switch", "-q", "-c", "feat/x")
	for _, agent := range []string{"claude-code", "codex", "opencode"} {
		t.Run(agent+" en la rama del cambio", func(t *testing.T) {
			code, stdout, stderr := runBranchGuardCLI(t, payloads[agent], "--agent", agent)
			if code != 0 || stderr != "" {
				t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
			}
			if agent == "opencode" && stdout != `{"action":"none","message":""}`+"\n" {
				t.Errorf("stdout %q, se esperaba la acción none", stdout)
			}
			if agent != "opencode" && stdout != "" {
				t.Errorf("stdout %q, se esperaba vacío", stdout)
			}
		})
	}
}

// El hook falla en abierto: una carga ilegible o de un proyecto que no es de Axiom no bloquea, no
// avisa y no escribe nada.
func TestBranchGuardFailsOpenAndWritesNothing(t *testing.T) {
	repo := branchGuardProject(t)
	elsewhere := branchOutsideGit(t)
	outsidePayload := guardPayloads(t, elsewhere)
	before := listFiles(t, elsewhere)

	cases := map[string]string{
		"carga que no es JSON":     "esto no es json",
		"entrada vacía":            "",
		"carga mayor de 256 KiB":   `{"pad":"` + strings.Repeat("a", 300<<10) + `"}`,
		"proyecto que no es Axiom": outsidePayload["claude-code"],
		"herramienta sin ruta":     `{"cwd":` + jsonString(t, repo) + `,"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`,
		"evento posterior (aviso)": strings.Replace(guardPayloads(t, repo)["claude-code"], "PreToolUse", "PostToolUse", 1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runBranchGuardCLI(t, input, "--agent", "claude-code")
			if code != 0 || stdout != "" || stderr != "" {
				t.Errorf("código %d, stdout %q, stderr %q; se esperaba 0 sin nada escrito", code, stdout, stderr)
			}
		})
	}
	if after := listFiles(t, elsewhere); strings.Join(after, "|") != strings.Join(before, "|") {
		t.Errorf("la guarda no debe escribir ficheros: antes %v, después %v", before, after)
	}
}

func jsonString(t *testing.T, value string) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var names []string
	if err := filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
		names = append(names, path)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return names
}
