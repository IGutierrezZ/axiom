import type { Plugin } from "@opencode-ai/plugin"
import { readFile, stat } from "node:fs/promises"
import { dirname, isAbsolute, join, resolve } from "node:path"

const TASK_RESULT = /^<task id="[^"\r\n]+" state="completed">\n(?:<summary>[^<>\r\n]+<\/summary>\n)?<task_result>\n([\s\S]*?)\n<\/task_result>\n<\/task>$/
const TASK_TAG = /<\/?(?:task|task_result|summary)(?:\s|>)/
const SDD_PHASES = ["sdd-init", "sdd-explore", "sdd-research", "sdd-propose", "sdd-spec", "sdd-design", "sdd-tasks", "sdd-apply", "sdd-verify", "sdd-archive", "sdd-onboard"]
const SDD_TASK_FAILURE_PREFIX = "AXIOM_SDD_FAILURE "
const SDD_TASK_ROUTE_TOKEN = /^[A-Za-z0-9][A-Za-z0-9._:@-]{0,127}$/
const SDD_PREFLIGHT_QUESTION_PREFIX = "Axiom SDD preflight "
// Solo se acepta el prefijo de Axiom: las preguntas con el prefijo del nombre anterior
// dejan de reconocerse; `axiom sync` reinstala los plugins y las instrucciones.
const SDD_PREFLIGHT_ACCEPTED_PREFIXES = [SDD_PREFLIGHT_QUESTION_PREFIX]
const SDD_PREFLIGHT_HEADING = "## SDD Session Preflight"
// #2855: host cwd does not identify the coordinator's selected change/store.
const SDD_TASK_CONTINUATION_GUIDANCE = "Return to the active SDD coordinator and inspect only its retained structured status for the selected change and artifact store. If that status is unavailable, report this terminal failure and ask the user to select the change and artifact store. Do not infer either, run unscoped status discovery, retry, or launch another phase."

type SDDTaskFailure = { phase: string, code: string, handoff: string }
type SDDTaskFailureError = Error & { sddFailure: SDDTaskFailure }
type SDDTopology = "monorepo-embedded" | "multirepo"
type SDDPreflightQuestion = { question: string, options: Array<{ label: string }>, multiple?: boolean }
type SDDPreflightChoice = { label: string, description: string, token: string }
type SDDPreflightGroup = { id: string, header: string, legacyNumber: number, requiredIn: SDDTopology[], options: SDDPreflightChoice[] }
type SDDPreflightBlockField = { label: string, groupId: string, default: string }

// SDD preflight contract. It mirrors the Go hook (internal/cli/sdd_preflight_hook.go): the same groups,
// markers, labels, tokens and block lines, so a session is governed the same way under Claude Code and
// OpenCode. The runtime — not the model — owns these labels and descriptions, so a user who picks a
// rendered option is always accepted downstream regardless of the model's phrasing or locale.
//
// `id` is the stable nominal marker (`<prefix><id>:`); `legacyNumber` is the position in the numbered
// marker older installs still write (`<prefix>N/3:`, 0 when the group never had one); `requiredIn` lists
// the topologies that ask the group. A group with no topology (the legacy Artifacts and PR strategy ones)
// is only recognized so that its answer is tolerated and ignored. The groups a session must answer are the ones
// its topology requires, so their number varies and may be zero.
const SDD_PREFLIGHT_GROUPS: SDDPreflightGroup[] = [
  {
    id: "pace", header: "Pace", legacyNumber: 1, requiredIn: ["monorepo-embedded"],
    options: [
      { label: "Interactive", description: "Confirm before each SDD phase advances.", token: "interactive" },
      { label: "Automatic", description: "Advance through SDD phases without per-phase confirmation.", token: "auto" },
    ],
  },
  {
    id: "artifacts", header: "Artifacts", legacyNumber: 2, requiredIn: [],
    options: [
      { label: "OpenSpec", description: "Track this change with OpenSpec proposal, spec, design, and task files.", token: "openspec" },
      { label: "Engram", description: "Track this change with Engram memory topics.", token: "engram" },
      { label: "Both", description: "Track this change with OpenSpec files and Engram memory together.", token: "hybrid" },
    ],
  },
  {
    id: "pr-strategy", header: "PR strategy", legacyNumber: 3, requiredIn: [],
    options: [
      { label: "Ask me", description: "Ask before opening a pull request when review risk is high.", token: "ask-on-risk" },
      { label: "Single PR", description: "Deliver the change as a single pull request.", token: "single-pr" },
      { label: "Auto", description: "Chain pull requests automatically as work completes.", token: "auto-chain" },
    ],
  },
]
// Lines of the block, in order. The artifact store is never asked and is always hybrid; a group the
// topology does not ask contributes its conservative default (interactive, single-pr).
const SDD_PREFLIGHT_BLOCK_FIELDS: SDDPreflightBlockField[] = [
  { label: "Pace", groupId: "pace", default: "interactive" },
  { label: "Artifact store", groupId: "", default: "hybrid" },
  { label: "Delivery strategy", groupId: "", default: "single-pr" },
]
const SDD_PREFLIGHT_AUTHORITY = "Parent-confirmed by the runtime; models and child agents cannot create or modify this block."
const SDD_PREFLIGHT_RETRY_GUIDANCE = "ask the required SDD preflight groups again with the question tool; typed chat answers cannot create preflight authority"

function sddPreflightGuidance(message: string): Error {
  return new Error(`${message}; ${SDD_PREFLIGHT_RETRY_GUIDANCE}`)
}

function isSDDPreflightQuestion(question: unknown): boolean {
  return !!question && typeof question === "object" && !Array.isArray(question) && typeof (question as Record<string, unknown>).question === "string" && SDD_PREFLIGHT_ACCEPTED_PREFIXES.some((prefix) => ((question as Record<string, unknown>).question as string).startsWith(prefix))
}

// The groups a topology asks, in table order.
function sddPreflightRequiredGroups(topology: SDDTopology): SDDPreflightGroup[] {
  return SDD_PREFLIGHT_GROUPS.filter((group) => group.requiredIn.includes(topology))
}

// The group a question's marker names under any accepted prefix: the nominal `<id>:` marker or the
// legacy `N/3:` one. A question with no marker, or an unknown one, names no group.
function sddPreflightQuestionGroup(text: string): SDDPreflightGroup | undefined {
  for (const prefix of SDD_PREFLIGHT_ACCEPTED_PREFIXES) {
    if (!text.startsWith(prefix)) continue
    const rest = text.slice(prefix.length)
    for (const group of SDD_PREFLIGHT_GROUPS) {
      if (rest.startsWith(`${group.id}:`)) return group
      if (group.legacyNumber > 0 && rest.startsWith(`${group.legacyNumber}/3:`)) return group
    }
  }
  return undefined
}

// Binds the questions of one call to the groups the topology requires, by marker and never by position.
// A question whose marker names no required group (unknown name, the legacy Artifacts group, or a group
// another topology asks) is ignored. A required group asked twice or not asked is refused naming it.
function sddPreflightBoundQuestions(questions: unknown[], topology: SDDTopology): Array<{ group: SDDPreflightGroup, index: number }> {
  const required = sddPreflightRequiredGroups(topology)
  const bound = new Map<string, number>()
  questions.forEach((raw, index) => {
    const text = raw && typeof raw === "object" && !Array.isArray(raw) ? (raw as Record<string, unknown>).question : undefined
    const group = typeof text === "string" ? sddPreflightQuestionGroup(text) : undefined
    if (!group || !required.includes(group)) return
    if (bound.has(group.id)) throw sddPreflightGuidance(`SDD preflight group ${group.id} (${group.header}) appears in more than one question`)
    bound.set(group.id, index)
  })
  return required.map((group) => {
    const index = bound.get(group.id)
    if (index === undefined) throw sddPreflightGuidance(`topology ${topology} requires the SDD preflight group ${group.id} (${group.header}), but no question starts with its "${SDD_PREFLIGHT_QUESTION_PREFIX}${group.id}:" marker`)
    return { group, index }
  })
}

// The refusal text for a dispatch with no preflight authority, naming the groups the topology requires.
function sddPreflightMissingAuthority(topology: SDDTopology): string {
  const ids = sddPreflightRequiredGroups(topology).map((group) => `\`${group.id}\``).join(", ")
  return `parent-confirmed SDD preflight is missing; ask the SDD preflight groups that topology ${topology} requires (${ids}) with the question tool; typed chat answers cannot create preflight authority`
}

// The block built from the answered tokens (keyed by group id); a group with no answer takes its default.
function sddPreflightBuildBlock(values: Record<string, string>): string {
  const lines = [SDD_PREFLIGHT_HEADING, SDD_PREFLIGHT_AUTHORITY]
  for (const field of SDD_PREFLIGHT_BLOCK_FIELDS) {
    const token = field.groupId === "" ? undefined : values[field.groupId]
    lines.push(`- ${field.label}: ${token ?? field.default}`)
  }
  return lines.join("\n")
}

// Index of the canonical choice a label names, or -1.
function sddPreflightChoice(group: SDDPreflightGroup, label: string): number {
  const normalized = normalizeSDDPreflightAnswer(label)
  return group.options.findIndex((option) => normalizeSDDPreflightAnswer(option.label) === normalized)
}

function sddPreflightLabels(group: SDDPreflightGroup): string {
  return group.options.map((option) => option.label).join(", ")
}

// Canonicalizes the groups of a grouped SDD preflight before OpenCode renders it: forces the nominal
// host marker, single-select, non-empty headers, and the canonical option labels in canonical order, in
// place. Only the questions whose marker names a group the topology requires are touched; unmarked,
// unknown and legacy Artifacts questions stay as the model wrote them. A description the model supplied
// is kept only for the canonical label it was written for (matched tolerantly by label, never by array
// position), and only when every canonical label finds exactly one such option; otherwise the built-in
// descriptions are used so a label is never shown next to another option's description. A required group
// that is missing or repeated is refused here, naming it. This runs for every session (root or child)
// because it only shapes what the user sees; authority recording stays root-only.
function canonicalizeSDDPreflightQuestions(args: unknown, topology: SDDTopology): void {
  if (!args || typeof args !== "object" || Array.isArray(args)) return
  const questions = (args as Record<string, unknown>).questions
  if (!Array.isArray(questions)) return
  if (!questions.some((question) => isSDDPreflightQuestion(question))) return
  for (const { group, index } of sddPreflightBoundQuestions(questions, topology)) {
    const question = questions[index] as Record<string, unknown>
    const marker = `${SDD_PREFLIGHT_QUESTION_PREFIX}${group.id}:`
    const existing = new RegExp(`^Axiom SDD preflight (?:${group.id}|${group.legacyNumber}/3):\\s*`)
    question.question = `${marker} ${(question.question as string).replace(existing, "")}`.trim()
    question.multiple = false
    if (typeof question.header !== "string" || question.header.trim() === "") question.header = group.header
    const supplied = Array.isArray(question.options) ? (question.options as unknown[]) : []
    const described = group.options.map((canonical) => {
      const matches = supplied.filter((option) => !!option && typeof option === "object" && !Array.isArray(option)
        && typeof (option as Record<string, unknown>).label === "string"
        && normalizeSDDPreflightAnswer((option as Record<string, unknown>).label as string) === normalizeSDDPreflightAnswer(canonical.label))
      if (matches.length !== 1) return undefined
      const description = (matches[0] as Record<string, unknown>).description
      return typeof description === "string" && description.trim() !== "" ? description : undefined
    })
    const keepDescriptions = supplied.length === group.options.length && described.every((description) => description !== undefined)
    question.options = group.options.map((canonical, optionIndex) => ({
      label: canonical.label,
      description: keepDescriptions ? (described[optionIndex] as string) : canonical.description,
    }))
  }
}

// Normalizes a user-facing answer or option label for tolerant matching:
// trims, collapses internal whitespace, lowercases, then strips diacritics
// via NFD decomposition so a typed or localized answer still binds.
function normalizeSDDPreflightAnswer(value: string): string {
  return value.trim().replace(/\s+/g, " ").toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "")
}

// axiom-project-gate:begin
// Axiom project gate (D3). This plugin lives in the user's OpenCode configuration, so it also loads
// in projects that have nothing to do with Axiom. Outside an Axiom project it stays inert: no
// preflight canonicalization, no SDD dispatch refusal and no task-result validation. It mirrors
// workspace.IsAxiomProject (internal/workspace/project.go): an `axiom.yaml` in the directory or an
// ancestor, or an `.axiom-workspace` pointer that resolves to an existing file. A bare `openspec/`
// does not count. Any error answers "not a project", so the gate fails open. A Go parity test pins
// this block against the Go criterion; change both together.
const AXIOM_CONFIG_FILE = "axiom.yaml"
const AXIOM_WORKSPACE_POINTER = ".axiom-workspace"

async function isRegularFile(path: string): Promise<boolean> {
  try { return (await stat(path)).isFile() } catch { return false }
}

// The pointer is a flat YAML with two optional scalar keys (`config`, `specs`), so a line match is
// enough here and no YAML parser is needed.
function pointerScalar(text: string, key: "config" | "specs"): string | undefined {
  const match = new RegExp(`^${key}[ \t]*:(.*)$`, "m").exec(text)
  if (!match) return undefined
  let value = match[1].trim()
  const quote = value.charAt(0)
  if (quote === '"' || quote === "'") {
    const end = value.indexOf(quote, 1)
    value = end > 0 ? value.slice(1, end) : value.slice(1)
  } else {
    value = value.replace(/\s+#.*$/, "")
  }
  value = value.trim()
  return value === "" ? undefined : value
}

async function pointerResolvesToConfig(pointer: string): Promise<boolean> {
  try {
    const text = await readFile(pointer, "utf8")
    const base = dirname(pointer)
    const config = pointerScalar(text, "config")
    if (config !== undefined) return await isRegularFile(isAbsolute(config) ? config : join(base, config))
    const specs = pointerScalar(text, "specs")
    return specs !== undefined && await isRegularFile(join(base, specs, AXIOM_CONFIG_FILE))
  } catch {
    return false
  }
}

async function isAxiomProject(start: string): Promise<boolean> {
  try {
    if (!start) return false
    let dir = resolve(start)
    if (!(await stat(dir)).isDirectory() || dirname(dir) === dir) return false
    for (;;) {
      if (await isRegularFile(join(dir, AXIOM_CONFIG_FILE))) return true
      const pointer = join(dir, AXIOM_WORKSPACE_POINTER)
      if (await isRegularFile(pointer) && await pointerResolvesToConfig(pointer)) return true
      const parent = dirname(dir)
      if (parent === dir) return false
      dir = parent
    }
  } catch {
    return false
  }
}
// axiom-project-gate:end

// axiom-topology:begin
// Axiom topology reader (D7). The preflight groups a session must answer depend on `workspace.topology`
// of the project's axiom.yaml, so the plugin reads it in process. It mirrors workspace.Locate and
// Location.GoverningTopology (internal/workspace/locate.go): the nearest marker wins (a pointer over an
// axiom.yaml over a bare openspec/), a broken pointer or a start that is not a directory resolves to
// nothing, and anything that is not a readable `multirepo` or `monorepo-embedded` (no file, no topology,
// unknown value, unreadable YAML) governs as monorepo-embedded, the widest question set: it asks too
// much, never too little. It reuses isRegularFile and pointerScalar from the project gate above. A Go
// parity test pins this block against the binary; change both together. Known, accepted divergences: the
// line reader below does not interpret YAML flow style (`workspace: {topology: multirepo}`) and treats it
// as unreadable, which is the safe side (the axiom.yaml files Axiom generates use block style); and it
// does not check that the specs repository of a multirepo exists, which the binary does when it resolves.
const AXIOM_DEFAULT_TOPOLOGY: SDDTopology = "monorepo-embedded"

async function isDirectory(path: string): Promise<boolean> {
  try { return (await stat(path)).isDirectory() } catch { return false }
}

// The axiom.yaml a pointer resolves to, or undefined when the pointer is broken.
async function pointerConfigPath(pointer: string): Promise<string | undefined> {
  try {
    const text = await readFile(pointer, "utf8")
    const base = dirname(pointer)
    const config = pointerScalar(text, "config")
    if (config !== undefined) {
      const target = isAbsolute(config) ? config : join(base, config)
      return await isRegularFile(target) ? target : undefined
    }
    const specs = pointerScalar(text, "specs")
    if (specs === undefined) return undefined
    const target = join(base, specs, AXIOM_CONFIG_FILE)
    return await isRegularFile(target) ? target : undefined
  } catch {
    return undefined
  }
}

// The effective axiom.yaml of the directory, or undefined when there is none to read.
async function findAxiomConfig(start: string): Promise<string | undefined> {
  try {
    if (!start) return undefined
    let dir = resolve(start)
    if (!(await isDirectory(dir)) || dirname(dir) === dir) return undefined
    for (;;) {
      const pointer = join(dir, AXIOM_WORKSPACE_POINTER)
      if (await isRegularFile(pointer)) return await pointerConfigPath(pointer)
      const config = join(dir, AXIOM_CONFIG_FILE)
      if (await isRegularFile(config)) return config
      if (await isDirectory(join(dir, "openspec"))) return undefined
      const parent = dirname(dir)
      if (parent === dir) return undefined
      dir = parent
    }
  } catch {
    return undefined
  }
}

// A YAML scalar: quoted (the quote closes the value) or plain with a trailing comment removed.
function yamlScalar(raw: string): string | undefined {
  let value = raw.trim()
  const quote = value.charAt(0)
  if (quote === '"' || quote === "'") {
    const end = value.indexOf(quote, 1)
    value = end > 0 ? value.slice(1, end) : value.slice(1)
  } else {
    value = value.replace(/\s+#.*$/, "")
  }
  value = value.trim()
  return value === "" ? undefined : value
}

// `topology` of the top-level `workspace:` block mapping: the key at the indentation of the block's first
// entry, never a deeper one nor one of another section.
function workspaceTopologyScalar(text: string): string | undefined {
  let inside = false
  let childIndent = -1
  for (const line of text.split(/\r?\n/)) {
    if (/^\s*(?:#.*)?$/.test(line)) continue
    const indent = line.length - line.trimStart().length
    if (!inside) {
      if (indent === 0 && /^workspace[ \t]*:[ \t]*(?:#.*)?$/.test(line)) inside = true
      continue
    }
    if (indent === 0) return undefined
    if (childIndent < 0) childIndent = indent
    if (indent !== childIndent) continue
    const match = /^[ \t]+topology[ \t]*:(.*)$/.exec(line)
    if (match) return yamlScalar(match[1])
  }
  return undefined
}

async function readAxiomTopology(start: string): Promise<SDDTopology> {
  try {
    const path = await findAxiomConfig(start)
    if (path === undefined) return AXIOM_DEFAULT_TOPOLOGY
    const value = workspaceTopologyScalar(await readFile(path, "utf8"))
    return value === "multirepo" || value === "monorepo-embedded" ? value : AXIOM_DEFAULT_TOPOLOGY
  } catch {
    return AXIOM_DEFAULT_TOPOLOGY
  }
}
// axiom-topology:end

function isSDDPhase(agent: string): boolean {
  return SDD_PHASES.some((phase) => agent === phase || agent.startsWith(phase + "-"))
}

function isBackgroundTask(args: unknown): boolean {
  return !!args && typeof args === "object" && !Array.isArray(args) && (args as Record<string, unknown>).background === true
}

async function isRootSession(client: any, sessionID: string): Promise<boolean> {
  try {
    const result = await client.session.get({ path: { id: sessionID } })
    const info = result?.data ?? result
    return !!info && info.id === sessionID && (info.parentID === undefined || info.parentID === null)
  } catch {
    return false
  }
}

// sddPreflightBlock builds the parent-confirmed preflight block from a
// question tool call and its answers for the groups the topology requires.
// The before-hook already canonicalizes a recognized preflight, so these
// structural checks are a safety net for a host that skipped it or a caller
// that bypassed it. Each question is bound to its group by its marker and each
// option and answer to a canonical choice by its label, never by position (the
// answers of the host are positional to the questions, so the answer of a
// group is the one at its question's index); only answer matching against the
// labels is intentionally tolerant (trim, whitespace collapse, case,
// diacritics) so a typed or slightly-off answer still binds to the option the
// user meant. Every error names the group and what to correct. A topology that
// requires no group needs no question: the block is entirely derived.
function sddPreflightBlock(args: unknown, metadata: unknown, topology: SDDTopology): string | undefined {
  if (!args || typeof args !== "object" || Array.isArray(args)) return undefined
  const questions = (args as Record<string, unknown>).questions
  if (!Array.isArray(questions)) return undefined
  const recognized = questions.some((question) => isSDDPreflightQuestion(question))
  if (!recognized) return undefined
  const bound = sddPreflightBoundQuestions(questions, topology)
  if (bound.length === 0) return sddPreflightBuildBlock({})
  if (!metadata || typeof metadata !== "object" || Array.isArray(metadata)) throw sddPreflightGuidance("SDD preflight response metadata is missing")
  const answers = (metadata as Record<string, unknown>).answers
  if (!Array.isArray(answers)) throw sddPreflightGuidance("SDD preflight response must contain the answers")
  const values: Record<string, string> = {}
  for (const { group, index } of bound) {
    const name = `SDD preflight question ${group.id} (${group.header})`
    const question = questions[index] as SDDPreflightQuestion
    if (question.multiple === true) throw sddPreflightGuidance(`${name} must be single-select`)
    const offered = new Set<number>()
    for (const option of Array.isArray(question.options) ? question.options : []) {
      const label = option && typeof option.label === "string" ? option.label : undefined
      const choice = label === undefined ? -1 : sddPreflightChoice(group, label)
      if (choice < 0) throw sddPreflightGuidance(`${name} offers option ${JSON.stringify(label ?? option)}, which is not one of its labels: ${sddPreflightLabels(group)}`)
      if (offered.has(choice)) throw sddPreflightGuidance(`${name} offers ${JSON.stringify(group.options[choice].label)} more than once`)
      offered.add(choice)
    }
    if (offered.size !== group.options.length) throw sddPreflightGuidance(`${name} must offer every one of its labels: ${sddPreflightLabels(group)}`)
    const answer = answers[index]
    if (!Array.isArray(answer) || answer.length === 0) throw sddPreflightGuidance(`${name} has no answer`)
    if (answer.length !== 1 || typeof answer[0] !== "string") throw sddPreflightGuidance(`${name} answer is not single-select`)
    const choice = sddPreflightChoice(group, answer[0])
    if (choice < 0) throw sddPreflightGuidance(`${name} answer ${JSON.stringify(answer[0])} is not one of its labels: ${sddPreflightLabels(group)}`)
    values[group.id] = group.options[choice].token
  }
  return sddPreflightBuildBlock(values)
}

function taskResult(output: unknown): void {
  if (typeof output !== "string" || output.trim() === "") {
    throw Object.assign(new Error("SDD phase output must not be empty"), { sddClass: "empty_result" })
  }
  const trimmed = output.trim()
  const envelope = TASK_RESULT.exec(trimmed)
  if (!envelope) {
    if (TASK_TAG.test(trimmed)) throw Object.assign(new Error("SDD phase output contains a malformed task result envelope"), { sddClass: "malformed_result" })
    return
  }
  if (envelope[1].trim() === "") throw Object.assign(new Error("SDD phase task result is empty"), { sddClass: "empty_result" })
  if (TASK_TAG.test(envelope[1])) throw Object.assign(new Error("SDD phase task result contains a nested task envelope"), { sddClass: "malformed_result" })
}

function taskRouteModel(metadata: unknown): string | undefined {
  if (!metadata || typeof metadata !== "object" || Array.isArray(metadata)) return undefined
  const model = (metadata as Record<string, unknown>).model
  if (!model || typeof model !== "object" || Array.isArray(model)) return undefined
  const providerID = (model as Record<string, unknown>).providerID
  const modelID = (model as Record<string, unknown>).modelID
  if (typeof providerID !== "string" || typeof modelID !== "string") return undefined
  if (!SDD_TASK_ROUTE_TOKEN.test(providerID) || !SDD_TASK_ROUTE_TOKEN.test(modelID)) return undefined
  return `${providerID}/${modelID}`
}

function sddTaskFailure(phase: string, cause: unknown, metadata?: unknown): SDDTaskFailureError {
  const empty = (cause as Record<string, unknown> | null)?.sddClass === "empty_result"
  const code = empty ? "sdd_task_result_empty" : "sdd_task_result_malformed"
  const taskModel = taskRouteModel(metadata)
  const guidance = "Do not retry or advance SDD; inspect the existing artifact state and surface the terminal failure to the user."
  const summary = empty
    ? `${phase} produced no task output at all. The child task returned nothing, which most often means the provider rejected the request before generation (authentication, region, or model access), the task was interrupted, or the phase genuinely wrote nothing. ${guidance}`
    : `${phase} returned no valid task result. ${guidance}`
  const failure: SDDTaskFailure = {
    phase,
    code,
    handoff: SDD_TASK_FAILURE_PREFIX + JSON.stringify({
      schemaName: "axiom.sdd-task-result-failure/v1",
      status: "blocked",
      code,
      phase,
      ...(taskModel === undefined ? {} : { taskModel }),
      summary,
      continuation: SDD_TASK_CONTINUATION_GUIDANCE,
    }),
  }
  return Object.assign(new Error(failure.handoff), { sddFailure: failure }) as SDDTaskFailureError
}

function sddDispatchLatched(requested: string, failure: SDDTaskFailure): Error {
  return new Error(SDD_TASK_FAILURE_PREFIX + JSON.stringify({
    schemaName: "axiom.sdd-task-result-failure/v1",
    status: "blocked",
    code: "sdd_task_dispatch_latched",
    phase: requested,
    latchedPhase: failure.phase,
    latchedCode: failure.code,
    summary: `${requested} was not dispatched. Earlier in this session ${failure.phase} returned ${failure.code}, and SDD launches stay latched afterwards so a failed phase is never silently retried and no later phase advances on top of it. No provider call, no subagent, and no artifact write happened for this launch, so it produced no new evidence about the original failure.`,
    continuation: SDD_TASK_CONTINUATION_GUIDANCE,
    exit: "Inspect the artifact state the original failure left, surface it to the user, and start a new session to launch SDD phases again. Relaunching in this session cannot dispatch.",
  }))
}

const SDDTaskResultArtifactsPlugin: Plugin = async ({ client, directory, worktree }) => {
  // Same session directory the branch guard plugin uses.
  const projectDir = () => directory || worktree || process.cwd()
  const failedSDDSessions = new Map<string, SDDTaskFailure>()
  const confirmedPreflights = new Map<string, string>()
  return {
    dispose: async () => { failedSDDSessions.clear(); confirmedPreflights.clear() },
    event: async ({ event }) => {
      if (event.type === "session.deleted") {
        failedSDDSessions.delete(event.properties.info.id)
        confirmedPreflights.delete(event.properties.info.id)
      }
    },
    "tool.execute.before": async (input, output) => {
      if (input.tool === "question") {
        if (!(await isAxiomProject(projectDir()))) return
        canonicalizeSDDPreflightQuestions(output.args, await readAxiomTopology(projectDir()))
        return
      }
      if (input.tool !== "task" || typeof output.args?.subagent_type !== "string") return
      const subagent = output.args.subagent_type
      if (!isSDDPhase(subagent)) return
      if (!(await isAxiomProject(projectDir()))) return
      if (!(await isRootSession(client, input.sessionID))) throw new Error("SDD child dispatch refused: only the interactive root session may carry parent-confirmed SDD preflight authority")
      const failure = failedSDDSessions.get(input.sessionID)
      if (failure) throw sddDispatchLatched(subagent, failure)
      if (typeof output.args.prompt !== "string") throw new Error("SDD child dispatch refused: task prompt is unavailable")
      if (output.args.prompt.includes(SDD_PREFLIGHT_HEADING)) throw new Error("SDD child dispatch refused: model-authored preflight text cannot create parent-confirmed authority")
      // A topology that requires no group has nothing to corroborate: the runtime derives the whole block.
      const topology = await readAxiomTopology(projectDir())
      const preflight = confirmedPreflights.get(input.sessionID) ?? (sddPreflightRequiredGroups(topology).length === 0 ? sddPreflightBuildBlock({}) : undefined)
      if (!preflight) throw new Error(`SDD child dispatch refused: ${sddPreflightMissingAuthority(topology)}`)
      output.args.prompt = `${preflight}\n\n${output.args.prompt}`
    },
    "tool.execute.after": async (input, output) => {
      if (input.tool === "question") {
        if (!(await isAxiomProject(projectDir()))) return
        if (!(await isRootSession(client, input.sessionID))) return
        try {
          const block = sddPreflightBlock(input.args, output.metadata, await readAxiomTopology(projectDir()))
          if (block !== undefined) confirmedPreflights.set(input.sessionID, block)
        } catch (cause) {
          confirmedPreflights.delete(input.sessionID)
          throw cause
        }
        return
      }
      if (input.tool !== "task" || typeof input.args?.subagent_type !== "string") return
      const subagent = input.args.subagent_type
      if (!isSDDPhase(subagent)) return
      if (!(await isAxiomProject(projectDir()))) return
      // OpenCode invokes this hook for the background launch acknowledgement while
      // the child is still running. That signal has no terminal task result to
      // validate; artifact/status ownership observes eventual completion.
      if (isBackgroundTask(input.args)) return
      try {
        taskResult(output.output)
      } catch (cause) {
        const failure = sddTaskFailure(subagent, cause, output.metadata)
        failedSDDSessions.set(input.sessionID, failure.sddFailure)
        throw failure
      }
    },
  }
}

export default SDDTaskResultArtifactsPlugin
