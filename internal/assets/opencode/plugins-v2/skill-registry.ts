// gentle-ai:managed skill-registry/v2
import { Plugin } from "@opencode/plugin"
import { execFile } from "node:child_process"
import { access } from "node:fs/promises"
import { homedir } from "node:os"
import { join, parse } from "node:path"

const PROJECT_MARKERS = [".git", ".atl", "skills", ".opencode/skills", ".claude/skills", ".gemini/skills", ".cursor/skills", ".github/skills", ".codex/skills", ".qwen/skills", ".kiro/skills", ".openclaw/skills", ".pi/skills", ".agent/skills", ".agents/skills", ".atl/skills", ".hermes/skills"]

// One single line, no stack. ENOENT from spawn does not identify the missing
// resource, so neither PATH absence nor the binary is asserted.
// execFile reports ExecFileException, whose code may be a number or null.
type RefreshFailure = { code?: string | number | null; syscall?: string; message: string }

function describeRefreshFailure(err: RefreshFailure, cwd: string): string {
  const where = JSON.stringify(cwd)
  const syscall = err.syscall
  const text = err.code === "ENOENT" && (!syscall || syscall.startsWith("spawn"))
    ? `the OpenCode runtime could not complete the axiom skill-registry refresh (ENOENT); the missing resource was not identified. Skipping the refresh for ${where}. Once the OpenCode runtime environment is valid, run \`axiom skill-registry refresh --no-gitignore --cwd <project>\` from a working shell, then re-launch OpenCode.`
    : err.code === "ENOENT"
      ? `axiom skill-registry refresh could not access the working directory ${where}: ENOENT reached execFile (syscall=${syscall ?? "unknown"}). Verify the directory exists and is reachable from the OpenCode process.`
      : `axiom skill-registry refresh failed for ${where}${err.code ? ` code=${err.code}` : ""}: ${err.message}`
  return `[skill-registry] ${text} Plugin stays best-effort and does not block startup.`.replace(/[\r\n\0]+/g, " ")
}

export default Plugin.define({
  id: "gentle-ai.skill-registry",
  async setup(ctx) {
    // The loaded location, never process.cwd or the containing project's root.
    const cwd = ctx.location.directory
    if (!cwd || cwd === parse(cwd).root || cwd === homedir()) return
    let project = false
    for (const marker of PROJECT_MARKERS) {
      if (await access(join(cwd, marker)).then(() => true, () => false)) { project = true; break }
    }
    if (!project) return
    try {
      let child: ReturnType<typeof execFile> | undefined
      child = execFile("axiom", ["skill-registry", "refresh", "--quiet", "--no-gitignore", "--cwd", cwd],
        { cwd, timeout: 30_000, maxBuffer: 1024, windowsHide: true }, (err) => {
          child = undefined
          if (err) console.error(describeRefreshFailure(err, cwd))
        })
      return () => { child?.kill("SIGKILL"); child = undefined }
    } catch (err) { console.error(describeRefreshFailure(err as RefreshFailure, cwd)) }
  },
})
