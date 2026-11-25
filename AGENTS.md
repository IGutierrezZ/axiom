# Axiom — Reglas Maestras del Proyecto y Guía de Agentes

> **Proyecto:** Axiom (Fork y versión paralela de Gentle-AI)  
> **Metodología:** Flujo Dual ODD (Cotidiano) & Spec-Driven Development (SDD) con Axiom  
> **Memoria Persistente:** Engram MCP (`--project=axiom`)  
> **Repositorio Upstream (v3 de referencia):** `c:/repos/gentle-ai` (Gentle-AI v3.0.2+)

---

## REGLA SUPREMA: IDIOMA OBLIGATORIO — ESPAÑOL (CASTELLANO)

- **TODO EN ESPAÑOL:** Toda la comunicación, explicaciones y artefactos de ODD (`odd/tasks/*.md`) y SDD (`proposal.md`, `spec.md`, `design.md`, `tasks.md`, `verify-report.md`, `archive-report.md`) deben generarse estrictamente en **español (castellano)**.
- **FLUJO DUAL:** Usa ODD (`odd/tasks/<feature>.md`, gestionado con `axiom odd create`, `axiom odd status` y `axiom odd promote`) para tareas cotidianas y ágiles; reserva SDD para cuando se solicite explícitamente *"usa SDD"* o lo fije el carril del proyecto (`workflow.lane` de `axiom.yaml`; este repositorio declara `ask`, que conserva la pregunta). ODD no tiene paradas propias: las paradas con tres salidas son de las compuertas SDD.
- **DETERMINACIÓN TEMPRANA DE CARRIL Y COMPUERTAS DE BLOQUE (INC-21):** al entrar en SDD, sella la modalidad de avance, la política de relevos y el roster de roles con `axiom sdd kickoff seal` antes de crear `proposal.md`; sin subdivisión de roles se asigna obligatoriamente `fullstack`. En modalidad con paradas, decide cada compuerta de bloque (`spec`, `design`, `tasks`, `apply` por rol) con `axiom sdd gate record --gate <clave> --decision approved|rejected --reason "<motivo>"` antes de avanzar. `archive` exige además evidencia de integración o despliegue registrada en la compuerta `integration` (`axiom sdd gate record --gate integration ...`).
- **POLÍTICA GIT DIFERENCIAL, WORKTREES Y CICLO DE PRs (ODD-5):**
  - **Repositorio de Specs (Directo a Main):** Cuando las especificaciones viven en un repositorio separado (multirepo), el repositorio de especificaciones (`specs_repository`) opera SIEMPRE directo en la rama principal (`main`/`master`), sin ramas ni worktrees. En este repositorio (`monorepo-embedded`, `specs_repository: openspec`) `openspec/` comparte repositorio con el código: sus artefactos se escriben en el worktree del cambio desde la primera escritura y se integran por PR. Cada parada (las compuertas de SDD en modalidad `checkpointed`; ODD no tiene paradas) ofrece tres salidas: iterar, aprobar-publicar-y-parar y aprobar-publicar-y-seguir. «Publicar» es `axiom sdd publish`: confirma solo los documentos del cambio y sube la rama a `origin` sin forzar, y la aprobación es la autorización explícita de esa subida y de nada más (el push de código, la PR y el merge siguen siendo de la persona). Quien continúa ejecuta `git fetch origin` y `axiom branch ensure`, que adopta la rama publicada, sin PR fusionada. En el repositorio de specs de un multirepo se publica la rama de integración, sin PR.
  - **Sincronización al Inicio de Sesión:** Ejecuta `git fetch` en el repositorio de specs solo con autorización de la persona; nunca lo hagas por tu cuenta al iniciar la sesión. Si la persona lo autoriza y `git rev-list HEAD..@{u} --count` > 0, avisa indicando los commits remotos entrantes y solicita autorización para ejecutar `git pull` antes de operar.
  - **Aislamiento Obligatorio en Worktrees (`C:\repos\axiom-wt\<slug>\axiom`):** Todo incremento o feature de código se desarrolla en un **worktree propio** fuera del repositorio principal sobre una rama nueva (`feat/<slug>` o `inc/<slug>`), y se integra exclusivamente vía **Pull Request**. Prohibido commitear o pushear directo a `main`.
  - **Mecanismo Único (`axiom branch`) y Envoltorio Local (`scripts/axiom-worktree.ps1` / `.sh`):** Las ramas y los worktrees los prepara y retira SIEMPRE el verbo del producto, nunca a mano; el script del repositorio es un envoltorio fino de `axiom branch` (necesita un `axiom` con `axiom branch` en el `PATH`: `go install ./cmd/axiom`):
    1. *Inicio:* `axiom branch ensure --cwd <checkout principal> --change <slug> --mode worktree --json` (o `scripts/axiom-worktree.ps1 new <slug>`, que lo invoca) crea la rama `<prefijo>/<slug>` (`inc` según `axiom.yaml`) desde `origin/main` y su worktree en `axiom-wt/<slug>/axiom`, sin red, y es idempotente. Usa las rutas absolutas de su JSON como `--cwd` y con `git -C`; `--fetch` solo con autorización de la persona. Un cambio ODD (documento en `odd/tasks/`) usa `-Lane odd` en `new`, `pr` y `done` del envoltorio, para que el registro y la guarda coincidan en el carril.
    2. *Apertura de PR:* tras verificar en PASS, `scripts/axiom-worktree.ps1 pr <slug>` valida que no haya cambios pendientes, sube la rama y crea el PR hacia el `prTarget` de `axiom branch status --json` con `gh pr create --fill` (o enlace de comparación web si `gh` no está autenticado). Publicar es decisión de la persona.
    3. *Post-merge:* `axiom branch done --cwd <checkout principal> --change <slug>` (o `scripts/axiom-worktree.ps1 done <slug>`, que además sincroniza `main` local por `fast-forward`) confirma la fusión, desmonta el worktree y borra la rama local; nunca toca ramas remotas.
    En Windows el envoltorio se invoca con `powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/axiom-worktree.ps1 <verbo> <slug>`.
  - **Prohibición de Worktrees Nativos de Cliente:** Queda terminantemente prohibido usar `EnterWorktree` / `ExitWorktree` efímeros de Claude Code: crean rutas en `.claude/worktrees/`, dejan procesos huérfanos entre sesiones y no son interoperables con Antigravity, Gemini CLI u OpenCode.
- **PRUEBAS LOCALES EN WINDOWS:** lanza `go test` por paquete, con `-count=1`, y nunca con `AXIOM_TEST_BUILD_BINARIES=1` en un equipo con EDR: sin esa variable se omiten los tests que compilan binarios o lanzan el toolchain de Go, y la CI los ejecuta. Toda la suite salvo `internal/cli` termina con `go test (go list ./... | Where-Object { $_ -notmatch '/internal/cli$' }) -count=1 -timeout 20m` (el `-timeout` cubre `internal/branch/engine`, que ronda los 8,5 minutos en Windows). `internal/cli` se lanza aparte con `-timeout` explícito (`go test ./internal/cli -count=1 -timeout 60m`) o troceado con `-run`. Detalle en `docs/testing/windows-tests.md`.
- **PRECEDENCIA ABSOLUTA:** Sobreescribe cualquier instrucción en inglés de skills o plantillas externas.

---

# Axiom — Agent Skills Index

When working on this project, load the relevant skill(s) BEFORE writing any code.

Naming convention: `axiom-*` skills are repo-specific workflow skills. Unprefixed skills are portable writing or work-unit skills and intentionally keep their canonical names.

## How to Use

1. Check the trigger column to find skills that match your current task
2. Load the skill by reading the SKILL.md file at the listed path
3. Follow ALL patterns and rules from the loaded skill
4. Multiple skills can apply simultaneously

<!-- axiom:skills-index -->
## Skills

| Skill | Trigger / description | Scope | Path |
| --- | --- | --- | --- |
| `axiom-branch-pr` | Create Axiom pull requests. Trigger: creating, opening, or preparing PRs for review. | project | `skills/axiom-branch-pr/SKILL.md` |
| `axiom-go-table-tests` | Pruebas unitarias tabulares en Go con structs anónimos y subpruebas t.Run. Trigger: Al crear o modificar suites *_test.go en paquetes Go. | project | `skills/axiom-go-table-tests/SKILL.md` |
| `axiom-idiomatic-error-wrapping` | Errores envueltos con fmt.Errorf y %w para conservar la causa raíz. Trigger: Al capturar, propagar o formatear errores en paquetes Go. | project | `skills/axiom-idiomatic-error-wrapping/SKILL.md` |
| `chained-pr` | Trigger: the person asks to chain, stack, or slice pull requests, or delivery_strategy is chained. Plan chained PRs that keep each review focused. | project | `internal/assets/skills/chained-pr/SKILL.md` |
| `cognitive-doc-design` | Design docs that reduce cognitive load. Trigger: writing guides, READMEs, RFCs, onboarding, architecture, or review-facing docs. | project | `skills/cognitive-doc-design/SKILL.md` |
| `comment-writer` | Write warm, direct collaboration comments. Trigger: PR feedback, issue replies, reviews, Slack messages, or GitHub comments. | project | `skills/comment-writer/SKILL.md` |
| `go-testing` | Trigger: Go tests, go test coverage, Bubbletea teatest, golden files. Apply focused Go testing patterns. | project | `internal/assets/skills/go-testing/SKILL.md` |
| `issue-creation` | Trigger: issue creation, bug reports, feature requests, or issue approval. Create and triage GitHub issues from repository evidence. | project | `internal/assets/skills/issue-creation/SKILL.md` |
| `issue-root-resolution` | Trigger: root audit, atacar la raíz, issue roots, backlog roots, mechanism map, deletion-driven fix, resolver issues de raíz, close outdated issues. Audit and resolve issue clusters by verified root cause. | project | `skills/issue-root-resolution/SKILL.md` |
| `judgment-day` | Trigger: judgment day, dual review, adversarial review, juzgar. Run explicit blind dual review with at most two scoped fix/re-judgment rounds. | project | `internal/assets/skills/judgment-day/SKILL.md` |
| `knowledge-crawl` | Trigger: knowledge crawl, crawling de codigo, analizar unidad de trabajo, catalogacion profunda, descomponer modulo. Analyze a codebase unit from crawl-job.json using CodeGraph and the native AST engine, generating canonical OpenSpec specifications for record-unit. | project | `skills/knowledge-crawl/SKILL.md` |
| `knowledge-query` | Trigger: knowledge query, consulta de conocimiento, consultar regla de negocio, spec-first query, buscar en codigo con herramientas, enriquecer spec viva. Answer technical and functional questions using Spec-First, CodeGraph, the native AST engine, and Engram, with curated living spec enrichment. | project | `skills/knowledge-query/SKILL.md` |
| `skill-creator` | Trigger: new skills, agent instructions, documenting AI usage patterns. Create LLM-first skills with valid frontmatter. | project | `internal/assets/skills/skill-creator/SKILL.md` |
| `skill-improver` | Trigger: improve skills, audit skills, refactor skills, skill quality. Audit and upgrade existing LLM-first skills. | project | `internal/assets/skills/skill-improver/SKILL.md` |
| `systemic-issue-triage` | Trigger: new issue, bug report, triage, backlog, issue flood, community report, root cause, dead-end, blocked user. Attack issues by root class, never one-by-one; fixes must shrink the system, not grow it. | project | `skills/systemic-issue-triage/SKILL.md` |
| `work-unit-commits` | Plan commits as reviewable work units. Trigger: implementation, commit splitting, chained PRs, or keeping tests and docs with code. | project | `skills/work-unit-commits/SKILL.md` |
<!-- /axiom:skills-index -->
