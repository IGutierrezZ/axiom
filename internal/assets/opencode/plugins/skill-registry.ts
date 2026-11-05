/**
 * skill-registry
 * Refreshes Gentle AI's project skill registry when OpenCode starts.
 *
 * Codex and Claude Code use native startup hooks for the same command. OpenCode
 * loads plugins at startup, so this plugin provides the equivalent behavior
 * without depending on shell interpolation or command-file parse-time cwd.
 *
 * Failure policy: the plugin is best-effort and must never block OpenCode
 * startup. When the refresh command fails we emit one actionable single line
 * (no stack) that distinguishes a spawn-or-absent-syscall ENOENT, whose
 * missing resource is not identified, from an unreachable working directory.
 */

import type { Plugin } from "@opencode-ai/plugin"
import { execFile } from "child_process"
import { access } from "fs/promises"
import { homedir } from "os"
import { join, parse } from "path"
import { promisify } from "util"

const execFileAsync = promisify(execFile)

// Mirrors the CLI guard's markers (.git, .atl, and ProjectSkillDirs in internal/skillregistry/registry.go); a Go parity test pins this list.
const PROJECT_MARKERS = [".git", ".atl", "skills", ".opencode/skills", ".claude/skills", ".gemini/skills", ".cursor/skills", ".github/skills", ".codex/skills", ".qwen/skills", ".kiro/skills", ".openclaw/skills", ".pi/skills", ".agent/skills", ".agents/skills", ".atl/skills", ".hermes/skills"]

async function pathExists(path: string): Promise<boolean> {
  try {
    await access(path)
    return true
  } catch {
    return false
  }
}

/**
 * OpenCode started in a brand-new non-project directory can resolve the
 * working directory to "/", the user's home directory, or a markerless
 * scratch folder. Refreshing there would initialize a stray .atl registry
 * (or fail loudly on a read-only root) at every startup. The CLI refuses
 * those locations too; this guard skips the spawn entirely.
 */
async function isProjectRoot(cwd: string): Promise<boolean> {
  if (!cwd) return false
  if (cwd === parse(cwd).root) return false
  if (cwd === homedir()) return false
  for (const marker of PROJECT_MARKERS) if (await pathExists(join(cwd, ...marker.split("/")))) return true
  return false
}

/**
 * Sanitize a working-directory string for safe single-line logging.
 * `JSON.stringify` escapes control characters and never embeds a literal newline.
 */
function quoteCwd(cwd: string): string {
  return JSON.stringify(cwd)
}

/** Collapse CR/LF and NUL so a diagnostic built from hostile input stays on one row. */
function singleLine(s: string): string {
  return s.replace(/[\r\n\0]+/g, " ")
}

/** Extract a readable message from any thrown value, including plain objects. */
function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message
  if (typeof err === "object" && err !== null) {
    const message = (err as { message?: unknown }).message
    if (typeof message === "string") return message
  }
  return String(err)
}

/**
 * Classify an `execFileAsync` failure for `axiom skill-registry refresh` into
 * a single actionable log line.
 *
 * - `ENOENT` from a `spawn axiom` syscall (or with no syscall field): the
 *   runtime could not complete the refresh and the error does not identify the
 *   missing resource. Never assert PATH absence or blame the binary: it can be
 *   present while the runtime environment is not.
 * - `ENOENT` from any other syscall (typically `access`/`stat`): the working
 *   directory itself is invalid, so name the cwd instead of the binary.
 * - Anything else: error code (when available) and message, without the stack.
 *
 * Exported so a Node harness can drive the string generation directly.
 */
export function describeRefreshFailure(err: unknown, cwd: string): string {
  const code = (err as NodeJS.ErrnoException | undefined)?.code
  const syscall = (err as NodeJS.ErrnoException | undefined)?.syscall
  const cwdExample = quoteCwd(cwd)
  if (code === "ENOENT" && (!syscall || syscall.startsWith("spawn"))) {
    return singleLine(
      `[skill-registry] the OpenCode runtime could not complete the axiom skill-registry refresh (ENOENT); ` +
      `the missing resource was not identified. ` +
      `Skipping the skill-registry refresh for ${cwdExample}. ` +
      `Once the OpenCode runtime environment is valid, run ` +
      `\`axiom skill-registry refresh --no-gitignore --cwd <project>\` from a working shell, then re-launch OpenCode. ` +
      `Plugin stays best-effort and does not block startup.`,
    )
  }
  if (code === "ENOENT") {
    return singleLine(
      `[skill-registry] axiom skill-registry refresh could not access the working directory ${cwdExample}: ` +
      `ENOENT reached execFile (syscall=${syscall ?? "unknown"}). ` +
      `Verify the directory exists and is reachable from the OpenCode process. ` +
      `Plugin stays best-effort and does not block startup.`,
    )
  }
  const codeTag = code ? ` code=${code}` : ""
  return singleLine(`[skill-registry] axiom skill-registry refresh failed for ${cwdExample}${codeTag}: ${errorMessage(err)}`)
}

export const SkillRegistryPlugin: Plugin = async (input) => {
  async function refreshSkillRegistry() {
    const cwd = input.worktree || input.directory || process.cwd()

    if (!(await isProjectRoot(cwd))) {
      // Startup hooks must not scream: a non-project directory is a normal
      // situation, not an error. Log to stderr — stdout belongs to commands
      // like `opencode models --verbose`, whose output axiom parses.
      console.error("[skill-registry] skipping refresh: not a project root:", cwd)
      return
    }

    try {
      await execFileAsync(
        "axiom",
        ["skill-registry", "refresh", "--quiet", "--no-gitignore", "--cwd", cwd],
        { timeout: 30_000 },
      )
    } catch (err) {
      console.error(describeRefreshFailure(err, cwd))
    }
  }

  // Don't await — keep OpenCode startup responsive. The command is
  // fingerprint-cached, so normal startup stays cheap.
  refreshSkillRegistry().catch((err) => {
    let cwd = "<unknown>"
    try {
      cwd = input.worktree || input.directory || process.cwd()
    } catch {
      // cwd selection itself failed; keep "<unknown>".
    }
    console.error(describeRefreshFailure(err, cwd))
  })

  return {}
}

export default SkillRegistryPlugin
