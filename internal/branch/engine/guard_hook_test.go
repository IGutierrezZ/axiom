package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Matriz de amenazas «Integración de proceso (hook)» (REQ-25.21, REQ-25.22, design D-18): una carga
// mal formada o enorme, un `..`, un tiempo agotado y un pánico no hacen nada (fallo en abierto), no
// se lanza ningún proceso y el protocolo de salida es el de cada runtime. Las pruebas funcionales
// usan un presupuesto holgado: lo que prueban es el protocolo, no la velocidad de la máquina.

const testGuardBudget = 30 * time.Second

func runGuardFor(t *testing.T, agent, input string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = runGuard(Default(), testGuardBudget, agent, strings.NewReader(input), &out, &errOut)
	return code, out.String(), errOut.String()
}

func claudeWrite(t *testing.T, event, cwd, path string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"cwd": cwd, "hook_event_name": event, "tool_name": "Write", "tool_input": map[string]any{"file_path": path}})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func openCodeWrite(t *testing.T, event, cwd, path string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"event": event, "tool": "write", "filePath": path, "cwd": cwd})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func codexPatch(t *testing.T, event, cwd, path string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"cwd": cwd, "hook_event_name": event, "tool_name": "apply_patch",
		"tool_input": map[string]any{"command": "*** Begin Patch\n*** Add File: " + path + "\n+hola\n*** End Patch\n"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// REQ-25.21 y D-18, protocolo de salida: Claude Code y Codex bloquean con salida 2 y el motivo por
// `stderr`; OpenCode sale siempre con 0 y entrega `{"action","message"}` por `stdout`.
func TestGuardHookOutputProtocolOfEachRuntime(t *testing.T) {
	repo := embeddedCoreProject(t)
	document := documentOf(repo)
	payloads := map[string]func(event string) string{
		GuardAgentClaudeCode: func(event string) string {
			return claudeWrite(t, map[string]string{GuardEventPre: "PreToolUse", GuardEventPost: "PostToolUse"}[event], repo, document)
		},
		GuardAgentCodex: func(event string) string {
			return codexPatch(t, map[string]string{GuardEventPre: "PreToolUse", GuardEventPost: "PostToolUse"}[event], repo, document)
		},
		GuardAgentOpenCode: func(event string) string { return openCodeWrite(t, event, repo, document) },
	}

	for _, agent := range GuardAgents {
		t.Run(agent+" bloquea en la rama equivocada", func(t *testing.T) {
			code, stdout, stderr := runGuardFor(t, agent, payloads[agent](GuardEventPre))
			if agent == GuardAgentOpenCode {
				var out guardOutput
				if code != 0 || stderr != "" || json.Unmarshal([]byte(stdout), &out) != nil || out.Action != GuardActionBlock ||
					!strings.Contains(out.Message, "axiom branch ensure --cwd ") {
					t.Fatalf("código %d, stdout %q, stderr %q; se esperaba 0, stdout con action=block y stderr vacío", code, stdout, stderr)
				}
				return
			}
			if code != 2 || stdout != "" || !strings.Contains(stderr, "axiom branch ensure --cwd ") {
				t.Fatalf("código %d, stdout %q, stderr %q; se esperaba 2, stdout vacío y el motivo por stderr", code, stdout, stderr)
			}
		})
		t.Run(agent+" no hace nada con el evento post de un documento", func(t *testing.T) {
			code, stdout, stderr := runGuardFor(t, agent, payloads[agent](GuardEventPost))
			if code != 0 || stderr != "" {
				t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
			}
			if agent == GuardAgentOpenCode {
				if want := `{"action":"none","message":""}` + "\n"; stdout != want {
					t.Errorf("stdout %q, se esperaba %q", stdout, want)
				}
			} else if stdout != "" {
				t.Errorf("stdout %q, se esperaba vacío", stdout)
			}
		})
	}

	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	for _, agent := range GuardAgents {
		t.Run(agent+" permite en la rama del cambio", func(t *testing.T) {
			code, _, stderr := runGuardFor(t, agent, payloads[agent](GuardEventPre))
			if code != 0 || stderr != "" {
				t.Errorf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
			}
		})
	}
}

// A-2 (spike-guard-runtimes.md): la carga documentada de Codex para `apply_patch` completa, con
// `tool_input.command` como cadena (`*** Begin Patch` … `*** End Patch`), la envoltura de sesión y,
// en `PostToolUse`, `tool_response`. La guarda solo lee del parche; el resto de la carga no cambia
// las rutas, y con `PreToolUse` sobre un documento en la rama equivocada se bloquea con salida 2.
func TestGuardHookCodexDocumentedApplyPatchEnvelope(t *testing.T) {
	repo := embeddedCoreProject(t)
	document := documentOf(repo)
	patch := "*** Begin Patch\n*** Add File: " + filepath.ToSlash(document) + "\n+hola\n*** Update File: internal/a.go\n@@\n-a\n+b\n*** End Patch\n"
	envelope := func(event string, withResponse bool) string {
		body := map[string]any{
			"session_id": "019a-test", "transcript_path": filepath.Join(repo, "transcript.jsonl"), "cwd": repo,
			"hook_event_name": event, "model": "gpt-5", "permission_mode": "default",
			"tool_name": "apply_patch", "tool_use_id": "call_1", "tool_input": map[string]any{"command": patch},
		}
		if withResponse {
			body["tool_response"] = map[string]any{"output": "Success. Updated the following files:\nA " + document + "\n*** Add File: otra.md"}
		}
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	for event, want := range map[string]string{"PreToolUse": GuardEventPre, "PostToolUse": GuardEventPost} {
		got, err := ParseGuardPayload(GuardAgentCodex, strings.NewReader(envelope(event, event == "PostToolUse")))
		if err != nil {
			t.Fatalf("%s: ParseGuardPayload: %v", event, err)
		}
		if got.Event != want || got.Tool != "apply_patch" || len(got.Paths) != 2 ||
			!strings.EqualFold(filepath.ToSlash(got.Paths[0]), filepath.ToSlash(document)) ||
			!strings.EqualFold(filepath.ToSlash(got.Paths[1]), filepath.ToSlash(filepath.Join(repo, "internal", "a.go"))) {
			t.Errorf("%s: carga = %+v; se esperaban el evento %q, la herramienta apply_patch y las dos rutas del parche, sin leer tool_response", event, got, want)
		}
	}

	code, stdout, stderr := runGuardFor(t, GuardAgentCodex, envelope("PreToolUse", false))
	if code != 2 || stdout != "" || !strings.Contains(stderr, "axiom branch ensure --cwd ") {
		t.Fatalf("código %d, stdout %q, stderr %q; se esperaba 2, stdout vacío y el motivo con la continuación por stderr", code, stdout, stderr)
	}
}

// El protocolo de un aviso (`warn`, que llega con S11-c2): Claude Code y Codex lo devuelven con
// salida 2 y el texto por `stderr`, que el runtime entrega al agente sin bloquear una escritura que
// ya ocurrió; OpenCode, por `stdout`.
func TestGuardHookOutputProtocolOfAWarning(t *testing.T) {
	t.Parallel()
	warning := GuardDecision{Action: GuardActionWarn, Message: "estás en la rama equivocada"}

	var out, errOut bytes.Buffer
	if code := writeGuardDecision(GuardAgentClaudeCode, warning, &out, &errOut); code != 2 || out.Len() != 0 || strings.TrimSpace(errOut.String()) != warning.Message {
		t.Errorf("Claude Code: código %d, stdout %q, stderr %q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := writeGuardDecision(GuardAgentOpenCode, warning, &out, &errOut); code != 0 || errOut.Len() != 0 ||
		out.String() != `{"action":"warn","message":"estás en la rama equivocada"}`+"\n" {
		t.Errorf("OpenCode: código %d, stdout %q, stderr %q", code, out.String(), errOut.String())
	}
}

// Fail-open: una carga de 1 MiB (aunque sea JSON válido de un documento en la rama equivocada), JSON
// inválido, vacío, un agente desconocido y una herramienta sin ruta no hacen nada.
func TestGuardHookDoesNothingWithAnOversizedOrMalformedPayload(t *testing.T) {
	repo := embeddedCoreProject(t)
	document := documentOf(repo)

	if code, _, stderr := runGuardFor(t, GuardAgentClaudeCode, claudeWrite(t, "PreToolUse", repo, document)); code != 2 {
		t.Fatalf("control: la carga sin relleno debía bloquear, código %d (%q)", code, stderr)
	}

	oversized, err := json.Marshal(map[string]any{
		"cwd": repo, "hook_event_name": "PreToolUse", "tool_name": "Write",
		"tool_input": map[string]any{"file_path": document, "content": strings.Repeat("a", 1<<20)},
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct{ agent, input string }{
		"1 MiB":                        {GuardAgentClaudeCode, string(oversized)},
		"1 MiB en Codex":               {GuardAgentCodex, string(oversized)},
		"un byte más del máximo":       {GuardAgentClaudeCode, `{"pad":"` + strings.Repeat("a", GuardMaxPayload) + `"}`},
		"JSON inválido":                {GuardAgentClaudeCode, `{"hook_event_name": "PreToolUse", `},
		"JSON inválido en OpenCode":    {GuardAgentOpenCode, `no es json`},
		"entrada vacía":                {GuardAgentClaudeCode, ``},
		"JSON que no es un objeto":     {GuardAgentCodex, `[1, 2, 3]`},
		"agente desconocido":           {"vim", claudeWrite(t, "PreToolUse", repo, document)},
		"herramienta sin ruta":         {GuardAgentClaudeCode, `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`},
		"ruta en blanco de OpenCode":   {GuardAgentOpenCode, `{"event":"pre","tool":"write","filePath":""}`},
		"evento desconocido de Claude": {GuardAgentClaudeCode, claudeWrite(t, "Notification", repo, document)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runGuardFor(t, tc.agent, tc.input)
			if code != 0 || stderr != "" {
				t.Errorf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
			}
			if tc.agent == GuardAgentOpenCode && strings.Contains(stdout, GuardActionBlock) {
				t.Errorf("stdout %q no debe bloquear", stdout)
			}
		})
	}
}

// El límite de la carga se aplica al leer: ParseGuardPayload no devuelve nada de una carga que lo
// supera ni lee de más.
func TestParseGuardPayloadRefusesMoreThanTheLimit(t *testing.T) {
	t.Parallel()
	reader := &countingReader{reader: strings.NewReader(strings.Repeat("a", 4*GuardMaxPayload))}

	if _, err := ParseGuardPayload(GuardAgentClaudeCode, reader); err == nil {
		t.Fatal("una carga mayor que el límite debe ser un error")
	}
	if reader.read > GuardMaxPayload+1+4096 {
		t.Errorf("se leyeron %d bytes, no debía pasar de %d", reader.read, GuardMaxPayload+1)
	}
}

type countingReader struct {
	reader io.Reader
	read   int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += n
	return n, err
}

// Fail-open por tiempo: con la entrada estándar bloqueada el hook termina dentro del presupuesto y
// no hace nada, y con una carga que sí cabe pero un presupuesto agotado tampoco.
func TestGuardHookDoesNothingWhenTheBudgetIsExhausted(t *testing.T) {
	t.Parallel()
	blocked, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })

	var out, errOut bytes.Buffer
	started := time.Now()
	code := runGuard(Default(), 50*time.Millisecond, GuardAgentClaudeCode, blocked, &out, &errOut)

	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Errorf("el hook tardó %s: debe terminar con el presupuesto", elapsed)
	}
	if code != 0 || out.Len() != 0 || errOut.Len() != 0 {
		t.Errorf("código %d, stdout %q, stderr %q; se esperaba 0 y nada escrito", code, out.String(), errOut.String())
	}
}

// Fail-open ante un pánico: la guarda recupera el pánico del trabajo y no hace nada, sin dejar de
// responder.
func TestGuardWithinRecoversAPanicAndDoesNothing(t *testing.T) {
	t.Parallel()

	decision := guardWithin(time.Second, func(context.Context) GuardDecision { panic("fallo interno") })

	wantNothing(t, decision)
}

// `..` que sale de la carpeta del cambio: la ruta se limpia al leer la carga y es otro fichero, que
// la guarda no bloquea aunque el checkout esté en la rama equivocada.
func TestGuardHookCleansDotDotPathsThatLeaveTheChange(t *testing.T) {
	repo := embeddedCoreProject(t)
	// Sin filepath.Join, que ya limpia: la carga lleva los `..` tal cual.
	separator := string(filepath.Separator)
	escaping := changeRootOf(repo) + separator + ".." + separator + ".." + separator + "internal" + separator + "a.go"

	payload, err := ParseGuardPayload(GuardAgentClaudeCode, strings.NewReader(claudeWrite(t, "PreToolUse", repo, escaping)))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(repo, "openspec", "internal", "a.go"); len(payload.Paths) != 1 || payload.Paths[0] != want {
		t.Fatalf("rutas = %v, se esperaba la ruta limpia %q", payload.Paths, want)
	}
	if code, _, stderr := runGuardFor(t, GuardAgentClaudeCode, claudeWrite(t, "PreToolUse", repo, escaping)); code != 0 || stderr != "" {
		t.Errorf("código %d, stderr %q; una ruta que sale del cambio es otro fichero y no se bloquea", code, stderr)
	}
}

// REQ-25.21, escenario «El hook no lanza git ni ningún otro proceso»: ni al bloquear ni al dejar
// pasar se registra una sola invocación de git.
func TestGuardHookLaunchesNoProcess(t *testing.T) {
	repo := embeddedCoreProject(t)
	seen := observe(t)

	if code, _, _ := runGuardFor(t, GuardAgentClaudeCode, claudeWrite(t, "PreToolUse", repo, documentOf(repo))); code != 2 {
		t.Fatalf("control: la escritura debía bloquearse, código %d", code)
	}
	if code, _, _ := runGuardFor(t, GuardAgentOpenCode, openCodeWrite(t, "pre", repo, filepath.Join(repo, "internal", "a.go"))); code != 0 {
		t.Fatalf("control: otro fichero no se bloquea, código %d", code)
	}
	if len(*seen) != 0 {
		t.Errorf("el hook lanzó procesos de git: %v", *seen)
	}
}

// La guarda no puede lanzar procesos ni abrir la red por construcción: sus fuentes no importan
// `os/exec` ni `net` (la arquitectura de internal/branch/** ya prohíbe todo salvo gitcommon).
func TestGuardSourcesImportNoProcessOrNetworkPackage(t *testing.T) {
	t.Parallel()
	for _, file := range []string{"guard.go", "guard_classify.go", "guard_hook.go"} {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if path == "os/exec" || path == "net" || strings.HasPrefix(path, "net/") || path == "syscall" {
				t.Errorf("%s importa %q: la guarda no lanza procesos ni abre la red", file, path)
			}
		}
	}
}
