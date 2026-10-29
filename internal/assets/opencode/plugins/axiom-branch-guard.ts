import type { Plugin } from "@opencode-ai/plugin"
import { spawn } from "node:child_process"

// Capa D de la guarda de rama (inc-25, design D-18): un adaptador mínimo que transporta la ruta
// destino de una escritura del agente a la orden `axiom branch guard` (protocolo de OpenCode) y aplica la
// decisión que devuelve Go. Toda la clasificación (documento de Axiom o código, rama correcta o
// equivocada) vive en Go; este plugin no lee ficheros ni interpreta rutas.
//
// - tool.execute.before (evento "pre"): si Go responde "block", lanza un Error con su mensaje.
// - tool.execute.after (evento "post"): si Go responde "warn", añade el aviso a la salida de la
//   herramienta. La escritura ya ocurrió: el aviso nunca bloquea.
//
// La orden es un array de argumentos fijo, sin shell, y la carga viaja solo por stdin: ningún dato
// de la carga llega a la línea de órdenes. Cualquier fallo (el binario no existe, JSON inválido,
// salida distinta de 0, salida desmesurada o tiempo agotado) falla en abierto: no hace nada.
//
// Mejor esfuerzo (R10): OpenCode puede tragarse la excepción de "before" en algunos carriles, de
// modo que el bloqueo no está verificado en todos ellos. Las garantías de fondo son las de las
// capas A y B; esta capa es complementaria.

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
function targetPath(args: unknown): string | undefined {
  if (args === null || typeof args !== "object") return undefined
  const filePath = Reflect.get(args, "filePath")
  return typeof filePath === "string" && filePath !== "" ? filePath : undefined
}

const AxiomBranchGuardPlugin: Plugin = async ({ directory, worktree }) => {
  const cwd = () => directory || worktree || process.cwd()
  return {
    "tool.execute.before": async (input, output) => {
      if (!GUARD_TOOLS.has(input.tool)) return
      const filePath = targetPath(output.args)
      if (filePath === undefined) return
      const decision = await runGuard("pre", input.tool, filePath, cwd())
      if (decision?.action === "block") throw new Error(decision.message)
    },
    "tool.execute.after": async (input, output) => {
      if (!GUARD_TOOLS.has(input.tool)) return
      const filePath = targetPath(input.args)
      if (filePath === undefined) return
      const decision = await runGuard("post", input.tool, filePath, cwd())
      if (decision?.action === "warn" && typeof output.output === "string") output.output = `${output.output}\n\n${decision.message}`
    },
  }
}

export default AxiomBranchGuardPlugin
