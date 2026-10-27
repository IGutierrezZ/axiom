package assets

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Arnés en proceso del plugin axiom-branch-guard.ts (REQ-25.22; R10): Node ejecuta el plugin real
// con `spawn` sustituido por un doble que registra la orden, los argumentos y la carga por stdin.
// No se lanza ningún proceso `axiom` y no se necesita OpenCode instalado.

// branchGuardPrelude define el doble de `spawn` y los temporizadores controlados. `script` decide,
// para cada proceso simulado, qué responde la guarda.
const branchGuardPrelude = `
import { EventEmitter } from "node:events"
const calls = []
const timers = []
let script = () => ({ stdout: JSON.stringify({ action: "none", message: "" }), code: 0 })
globalThis.setTimeout = (fn, ms) => { timers.push({ fn, ms, cleared: false }); return timers.length - 1 }
globalThis.clearTimeout = (id) => { if (timers[id]) timers[id].cleared = true }
globalThis.__spawn = (command, args, options) => {
  const call = { command, args, options, stdin: "", ended: false, killed: false }
  calls.push(call)
  const behavior = script(call)
  if (behavior.throws) throw new Error("spawn failed")
  const child = new EventEmitter()
  child.stdout = new EventEmitter()
  child.stderr = new EventEmitter()
  child.stdin = new EventEmitter()
  child.stdin.write = (data) => { call.stdin += data; return true }
  child.stdin.end = (data) => {
    if (data !== undefined) call.stdin += data
    call.ended = true
    queueMicrotask(() => {
      if (behavior.hang) return
      if (behavior.error) { child.emit("error", new Error(behavior.error)); return }
      if (behavior.stdout !== undefined) child.stdout.emit("data", Buffer.from(behavior.stdout))
      child.emit("close", behavior.code ?? 0)
    })
  }
  child.kill = () => { call.killed = true }
  return child
}
const assert = (condition, message) => { if (!condition) throw new Error(message) }
const rejects = async (fn) => { try { await fn() } catch (cause) { return cause } throw new Error("se esperaba un rechazo") }
const decision = (action, message) => ({ stdout: JSON.stringify({ action, message }), code: 0 })
const ARGS = JSON.stringify(["branch", "guard", "--agent", "opencode"])
const failOpenCases = [
  { name: "error de spawn", behavior: { error: "ENOENT" } },
  { name: "spawn lanza", behavior: { throws: true } },
  { name: "JSON inválido", behavior: { stdout: "no es JSON" } },
  { name: "acción desconocida", behavior: { stdout: JSON.stringify({ action: "explode", message: "x" }) } },
  { name: "bloqueo sin mensaje", behavior: { stdout: JSON.stringify({ action: "block" }) } },
  { name: "salida distinta de 0", behavior: { stdout: JSON.stringify({ action: "block", message: "x" }), code: 2 } },
  { name: "salida vacía", behavior: { stdout: "" } },
  { name: "salida desmesurada", behavior: { stdout: JSON.stringify({ action: "block", message: "x".repeat(200000) }) } },
]
`

// runOpenCodeBranchGuardV1 ejecuta el plugin de OpenCode v1 con el arnés dado.
func runOpenCodeBranchGuardV1(t *testing.T, harness string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	source, err := Read("opencode/plugins/axiom-branch-guard.ts")
	if err != nil {
		t.Fatal(err)
	}
	source = strings.Replace(source, `import { spawn } from "node:child_process"`, `const spawn = (...args: any[]) => (globalThis as any).__spawn(...args)`, 1)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.mts"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "harness.mjs"), []byte("import plugin from './plugin.mts'\n"+branchGuardPrelude+harness), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "--experimental-strip-types", filepath.Join(dir, "harness.mjs"))
	cmd.Env = append(os.Environ(), "HOME="+dir, "USERPROFILE="+dir, "XDG_CONFIG_HOME="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("arnés de axiom-branch-guard v1: %v\n%s", err, out)
	}
}

func TestOpenCodeBranchGuardPluginBlocksBeforeWriteTools(t *testing.T) {
	runOpenCodeBranchGuardV1(t, `
const hooks = await plugin({ directory: "/repo", worktree: "/worktree" })
const before = (tool, args) => hooks["tool.execute.before"]({ tool, sessionID: "s", callID: "c" }, { args })

const dangerous = "/repo/openspec/changes/x/$(touch pwned); rm -rf x.md"
script = () => decision("block", "Rama equivocada: ejecuta axiom branch ensure --change x")
for (const tool of ["write", "edit", "patch"]) {
  calls.length = 0
  const error = await rejects(() => before(tool, { filePath: dangerous, content: "x" }))
  assert(error instanceof Error && error.message === "Rama equivocada: ejecuta axiom branch ensure --change x", tool + ": mensaje de bloqueo " + error.message)
  assert(calls.length === 1, tool + ": un único proceso")
  const call = calls[0]
  assert(call.command === "axiom", tool + ": orden " + call.command)
  assert(JSON.stringify(call.args) === ARGS, tool + ": argumentos " + JSON.stringify(call.args))
  assert(!call.options.shell, tool + ": sin shell")
  assert(call.options.cwd === "/repo", tool + ": cwd " + call.options.cwd)
  assert(call.ended, tool + ": stdin cerrado")
  const payload = JSON.parse(call.stdin)
  assert(payload.event === "pre" && payload.tool === tool && payload.filePath === dangerous && payload.cwd === "/repo", tool + ": carga " + call.stdin)
  assert(timers.at(-1).cleared === true, tool + ": temporizador liberado")
}

// Una decisión de aviso o ninguna no bloquea en before y no toca los argumentos.
for (const reply of [decision("warn", "aviso"), decision("none", "")]) {
  script = () => reply
  const args = { filePath: "/repo/a.go" }
  await before("write", args)
  assert(JSON.stringify(args) === JSON.stringify({ filePath: "/repo/a.go" }), "before no modifica los argumentos")
}

// Otras herramientas no lanzan ningún proceso; patch sin filePath tampoco (la guarda no tiene ruta que clasificar).
script = () => decision("block", "no debe llegar")
calls.length = 0
await before("bash", { command: "ls", filePath: "/repo/openspec/changes/x/a.md" })
await before("read", { filePath: "/repo/openspec/changes/x/a.md" })
await before("patch", { patchText: "*** Begin Patch" })
await before("write", {})
assert(calls.length === 0, "procesos lanzados sin herramienta de escritura o sin ruta: " + calls.length)
`)
}

func TestOpenCodeBranchGuardPluginWarnsAfterWriteTools(t *testing.T) {
	runOpenCodeBranchGuardV1(t, `
const hooks = await plugin({ directory: "/repo", worktree: "/worktree" })
const after = (tool, args, output) => hooks["tool.execute.after"]({ tool, sessionID: "s", callID: "c", args }, output)

script = () => decision("warn", "Aviso: estás en main; el cambio x vive en el worktree C:/wt/x")
for (const tool of ["write", "edit", "patch"]) {
  calls.length = 0
  const output = { output: "Wrote file successfully.", metadata: {} }
  await after(tool, { filePath: "/repo/internal/a.go" }, output)
  assert(output.output === "Wrote file successfully.\n\nAviso: estás en main; el cambio x vive en el worktree C:/wt/x", tool + ": salida " + JSON.stringify(output.output))
  assert(calls.length === 1 && calls[0].command === "axiom" && JSON.stringify(calls[0].args) === ARGS, tool + ": proceso")
  const payload = JSON.parse(calls[0].stdin)
  assert(payload.event === "post" && payload.tool === tool && payload.filePath === "/repo/internal/a.go" && payload.cwd === "/repo", tool + ": carga " + calls[0].stdin)
}

// Sin aviso, con un bloqueo (after nunca bloquea) o con una salida que no es texto, no se toca nada.
for (const reply of [decision("none", ""), decision("block", "no aplica en after")]) {
  script = () => reply
  const output = { output: "ok", metadata: {} }
  await after("write", { filePath: "/repo/a.go" }, output)
  assert(output.output === "ok", "after sin aviso no modifica la salida")
}
script = () => decision("warn", "aviso")
const structured = { output: { ok: true }, metadata: {} }
await after("write", { filePath: "/repo/a.go" }, structured)
assert(JSON.stringify(structured.output) === JSON.stringify({ ok: true }), "una salida que no es texto no se modifica")

// Otras herramientas y after sin ruta no lanzan procesos.
calls.length = 0
await after("bash", { filePath: "/repo/a.go" }, { output: "ok" })
await after("write", {}, { output: "ok" })
assert(calls.length === 0, "procesos lanzados sin herramienta de escritura o sin ruta")
`)
}

func TestOpenCodeBranchGuardPluginFailsOpen(t *testing.T) {
	runOpenCodeBranchGuardV1(t, `
const hooks = await plugin({ directory: "/repo", worktree: "/worktree" })
for (const { name, behavior } of failOpenCases) {
  script = () => behavior
  await hooks["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: { filePath: "/repo/openspec/changes/x/a.md" } })
  const output = { output: "ok", metadata: {} }
  await hooks["tool.execute.after"]({ tool: "write", sessionID: "s", callID: "c", args: { filePath: "/repo/a.go" } }, output)
  assert(output.output === "ok", name + ": la salida no cambia")
}

// Tiempo agotado: el temporizador es acotado, el proceso se mata y el hook termina sin bloquear.
script = () => ({ hang: true })
calls.length = 0
const timersBefore = timers.length
const pending = hooks["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: { filePath: "/repo/openspec/changes/x/a.md" } })
const timer = timers[timersBefore]
assert(timer !== undefined && timer.ms > 0 && timer.ms <= 5000, "tiempo límite acotado: " + timer?.ms)
timer.fn()
await pending
assert(calls[0].killed, "el proceso se mata al agotar el tiempo")

const pendingAfter = hooks["tool.execute.after"]({ tool: "edit", sessionID: "s", callID: "c", args: { filePath: "/repo/a.go" } }, { output: "ok" })
timers.at(-1).fn()
await pendingAfter
`)
}

func TestOpenCodeV2BranchGuardPlugin(t *testing.T) {
	runV2Plugin(t, "axiom-branch-guard", branchGuardPrelude+`
const hooks = {}
let disposed = 0
const cleanup = await plugin.setup({ location: { directory: "/repo" }, tool: { async hook(name, fn) { hooks[name] = fn; return { async dispose() { disposed++ } } } } })
assert(typeof hooks["execute.before"] === "function" && typeof hooks["execute.after"] === "function", "hooks registrados")

script = () => decision("block", "Rama equivocada")
const dangerous = "/repo/openspec/changes/x/$(touch pwned).md"
for (const tool of ["write", "edit", "patch"]) {
  calls.length = 0
  const error = await rejects(() => hooks["execute.before"]({ tool, sessionID: "s", id: "c", input: { filePath: dangerous } }))
  assert(error instanceof Error && error.message === "Rama equivocada", tool + ": bloqueo")
  assert(calls.length === 1 && calls[0].command === "axiom" && JSON.stringify(calls[0].args) === ARGS && !calls[0].options.shell && calls[0].ended, tool + ": proceso")
  const payload = JSON.parse(calls[0].stdin)
  assert(payload.event === "pre" && payload.tool === tool && payload.filePath === dangerous && payload.cwd === "/repo", tool + ": carga " + calls[0].stdin)
}
calls.length = 0
await hooks["execute.before"]({ tool: "bash", sessionID: "s", id: "c", input: { filePath: dangerous } })
await hooks["execute.before"]({ tool: "write", sessionID: "s", id: "c", input: {} })
assert(calls.length === 0, "sin herramienta de escritura o sin ruta no hay proceso")

script = () => decision("warn", "Aviso de rama")
const call = { tool: "edit", sessionID: "s", id: "c", status: "completed", input: { filePath: "/repo/a.go" }, result: { output: { ok: true }, content: "hecho" } }
await hooks["execute.after"](call)
assert(call.result.content === "hecho\n\nAviso de rama" && call.result.output.ok === true, "aviso añadido al contenido: " + JSON.stringify(call.result))
assert(JSON.parse(calls.at(-1).stdin).event === "post", "after envía el evento post")
const failed = { tool: "edit", sessionID: "s", id: "c", status: "error", input: { filePath: "/repo/a.go" }, error: { message: "x" } }
calls.length = 0
await hooks["execute.after"](failed)
assert(calls.length === 0 && failed.result === undefined, "una herramienta fallida no se avisa")

for (const { name, behavior } of failOpenCases) {
  script = () => behavior
  await hooks["execute.before"]({ tool: "write", sessionID: "s", id: "c", input: { filePath: "/repo/openspec/changes/x/a.md" } })
  const done = { tool: "write", sessionID: "s", id: "c", status: "completed", input: { filePath: "/repo/a.go" }, result: { content: "ok" } }
  await hooks["execute.after"](done)
  assert(done.result.content === "ok", name + ": el contenido no cambia")
}

await cleanup()
assert(disposed === 2, "las dos registraciones se liberan: " + disposed)
`)
}
