// axiom:managed axiom-branch-guard/v2
// Sin verificar en un runtime real de OpenCode v2: la forma de las llamadas de herramienta sigue la
// de los demás plugins gestionados de esta generación (ctx.tool.hook con execute.before y
// execute.after) y los nombres de herramienta son los de la generación v1.
import { Plugin } from "@opencode/plugin"
import { spawn } from "node:child_process"

// Capa D de la guarda de rama (inc-25, design D-18): un adaptador mínimo que transporta la ruta
// destino de una escritura del agente a la orden `axiom branch guard` (protocolo de OpenCode) y aplica la
// decisión que devuelve Go. Toda la clasificación vive en Go; este plugin no lee ficheros ni
// interpreta rutas.
//
// - execute.before (evento "pre"): si Go responde "block", lanza un Error con su mensaje.
// - execute.after (evento "post"): si Go responde "warn", añade el aviso al contenido de la
//   herramienta. La escritura ya ocurrió: el aviso nunca bloquea.
//
// La orden es un array de argumentos fijo, sin shell, y la carga viaja solo por stdin. Cualquier
// fallo (el binario no existe, JSON inválido, salida distinta de 0, salida desmesurada o tiempo
// agotado) falla en abierto: no hace nada. Mejor esfuerzo (R10): el bloqueo no está verificado en
// todos los carriles de OpenCode; las garantías de fondo son las de las capas A y B.

const GUARD_COMMAND = "axiom"
const GUARD_ARGS = ["branch", "guard", "--agent", "opencode"]
const GUARD_TOOLS = new Set(["write", "edit", "patch"])
// La guarda tiene un presupuesto de 1 s; el plugin espera algo más para no cortarla antes de tiempo.
const GUARD_TIMEOUT_MS = 4000
const GUARD_MAX_OUTPUT = 64 * 1024

type GuardEvent = "pre" | "post"
type GuardDecision = { action: "block" | "warn"; message: string }

// Interpreta la salida de la guarda. Cualquier cosa que no sea exactamente un bloqueo o un aviso
// con mensaje no es una decisión (falla en abierto).
function parseDecision(stdout: string): GuardDecision | undefined {
  try {
    const parsed: unknown = JSON.parse(stdout.trim())
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) return undefined
    const action = Reflect.get(parsed, "action")
    const message = Reflect.get(parsed, "message")
    if ((action !== "block" && action !== "warn") || typeof message !== "string" || message === "") return undefined
    return { action, message }
  } catch {
    return undefined
  }
}

// Lanza la guarda una vez y resuelve siempre, sin lanzar nunca: la decisión, o undefined si algo falla.
function runGuard(event: GuardEvent, tool: string, filePath: string, cwd: string): Promise<GuardDecision | undefined> {
  return new Promise((resolve) => {
    let settled = false
    let timer: ReturnType<typeof setTimeout> | undefined
    let child: ReturnType<typeof spawn> | undefined
    const finish = (decision?: GuardDecision) => {
      if (settled) return
      settled = true
      if (timer !== undefined) clearTimeout(timer)
      try { child?.kill() } catch { /* el proceso ya terminó */ }
      resolve(decision)
    }
    try {
      timer = setTimeout(() => finish(), GUARD_TIMEOUT_MS)
      child = spawn(GUARD_COMMAND, GUARD_ARGS, { cwd, stdio: ["pipe", "pipe", "ignore"], windowsHide: true })
      const chunks: Buffer[] = []
      let size = 0
      child.stdout?.on("data", (chunk: Buffer) => {
        size += chunk.length
        if (size > GUARD_MAX_OUTPUT) { finish(); return }
        chunks.push(chunk)
      })
      child.on("error", () => finish())
      child.stdin?.on("error", () => { /* la salida decide; el tiempo límite acota */ })
      child.on("close", (code: number | null) => {
        if (code !== 0) { finish(); return }
        finish(parseDecision(Buffer.concat(chunks).toString("utf8")))
      })
      child.stdin?.end(JSON.stringify({ event, tool, filePath, cwd }))
    } catch {
      finish()
    }
  })
}

// La ruta destino que declara la herramienta; sin ella no hay nada que clasificar.
function targetPath(input: unknown): string | undefined {
  if (input === null || typeof input !== "object") return undefined
  const filePath = Reflect.get(input, "filePath")
  return typeof filePath === "string" && filePath !== "" ? filePath : undefined
}

export default Plugin.define({
  id: "gentle-ai.axiom-branch-guard",
  async setup(ctx) {
    const registrations: Array<{ dispose(): Promise<void> }> = []
    const cwd = ctx.location.directory
    try {
      registrations.push(await ctx.tool.hook("execute.before", async (call) => {
        if (!GUARD_TOOLS.has(call.tool)) return
        const filePath = targetPath(call.input)
        if (filePath === undefined) return
        const decision = await runGuard("pre", call.tool, filePath, cwd)
        if (decision?.action === "block") throw new Error(decision.message)
      }))
      registrations.push(await ctx.tool.hook("execute.after", async (call) => {
        if (call.status !== "completed" || !GUARD_TOOLS.has(call.tool)) return
        const filePath = targetPath(call.input)
        if (filePath === undefined) return
        const decision = await runGuard("post", call.tool, filePath, cwd)
        const content: unknown = call.result?.content
        if (decision?.action === "warn" && typeof content === "string") call.result = { ...call.result, content: `${content}\n\n${decision.message}` }
      }))
    } catch (cause) {
      await Promise.all(registrations.map((registration) => registration.dispose()))
      throw cause
    }
    return async () => {
      await Promise.all(registrations.map((registration) => registration.dispose()))
    }
  },
})
