# Guarda del checkout principal de axiom

- **Rama:** `fix/guarda-checkout-principal` · **Worktree:** `axiom-wt/guarda-checkout-principal`
- **Carril:** ODD · **TDD:** no configurado (`axiom.yaml` sin clave de pruebas) → comprobaciones funcionales
- **Entrega:** `ask-on-risk`; previsión ≈ 250 líneas → un único PR

## Objetivo

El checkout principal `C:\repos\axiom` queda siempre en `main`. Todo cambio de este repositorio (código y artefactos `openspec/`, porque la topología es `monorepo-embedded`) nace desde el principio en un worktree `axiom-wt/<slug>` creado con `scripts/axiom-worktree.ps1 new`.

## Problema y causa

Algunas sesiones de Claude Code hacen `git checkout -b` en el checkout principal (reflog: `fix/query-readonly` 2026-09-29, `feat/inc-24-spec-baseline-profile` 2026-10-03, `docs/readme-contributors-attribution` 2026-10-12).

- La regla de worktrees solo vive en `AGENTS.md` y `GEMINI.md`; Claude Code no los lee y no hay `CLAUDE.md` en la raíz (`.claude/` está en `.gitignore`).
- Lo que Claude sí lee (paso 6 de ODD) dice «branch first when on the default branch», que se ejecuta en el cwd.
- `AGENTS.md`/`GEMINI.md` dicen que `specs_repository` trabaja directo en `main`; en este monorepo eso choca con la regla, porque `openspec/` comparte repositorio con el código.
- No hay guarda mecánica.

## Alcance y restricciones

- **Solo política local de este repositorio.** No se toca el comportamiento del producto: `internal/components/agentguidance/routing.go` e `internal/assets/skills/**` siguen igual. En otros proyectos el worktree sigue siendo configurable y se crea en `apply`, porque en multirepo el repositorio de specs trabaja en `main`.
- Sin `EnterWorktree`/`ExitWorktree`.
- El directorio `.claude/` del checkout principal está excluido de la guarda: es configuración local, no versionada.

## Tareas

- [x] **T1** — `CLAUDE.md` en la raíz con la regla local, y aclaración del caso monorepo en `AGENTS.md` y `GEMINI.md`. Ruta: delegada (escritor único; disparador: 2+ ficheros no triviales junto con T2). Commit `7eeb4c3d`.
- [x] **T2** — Guarda para hooks de Claude Code `scripts/claude-hooks/guard-primary-checkout.ps1` y su registro en el `.claude/settings.json` local del checkout principal. Ruta: delegada (mismo escritor).
  - `PreToolUse` en `Write|Edit|NotebookEdit`: deniega rutas dentro del checkout principal, salvo `.claude/`.
  - `PostToolUse` en `Bash|PowerShell`: si el checkout principal no está en `main`, o si `main` tiene commits que no están en `origin/main`, devuelve el error a Claude con la forma de corregirlo.

## Criterios de aceptación

- Una sesión de Claude Code abierta en `C:\repos\axiom` ve la regla (`CLAUDE.md` raíz) y la guarda la frena al editar en el checkout principal o al dejarlo fuera de `main`.
- Las rutas `C:\repos\axiom-wt\...` no se confunden con el checkout principal: el prefijo se compara con separador final.
- La guarda no falla de forma bloqueante si no puede leer el estado de git: en ese caso avisa sin bloquear.

## Comprobaciones

- Pruebas de la guarda con JSON de ejemplo por stdin: escritura dentro y fuera del principal, `.claude/`, `axiom-wt`, rutas `C:/` y `/c/`, principal en `main` y fuera de `main`.
- `go build ./...` (sin cambios de Go, solo como control).

## Progreso

- Worktree creado desde `origin/main` (`e7d4104c`).
- T1 hecha en `7eeb4c3d` (`CLAUDE.md` raíz; aclaración multirepo/monorepo en `AGENTS.md` y `GEMINI.md`). Ruta: delegada.
- T2 hecha: `scripts/claude-hooks/guard-primary-checkout.ps1` (compatible con Windows PowerShell 5.1, con BOM UTF-8 para los acentos) y registro de los hooks `PreToolUse` (`Write|Edit|NotebookEdit`) y `PostToolUse` (`Bash|PowerShell`) en el `.claude/settings.json` local del checkout principal, sin tocar el resto de hooks. Ruta: delegada.
- Evidencia de comprobaciones: controlador de pruebas fuera del repositorio con 28 casos, todos conformes (escrituras en el principal con `C:\`, `C:/` y `/c/` → 2; `.claude/`, `axiom-wt\...`, hermano `axiom2` y fuera de repositorio → 0; ruta relativa con cwd principal → 2 y con cwd worktree → 0; JSON malformado o vacío → 0). El estado del principal se probó en un repositorio desechable con worktree enlazado: rama distinta → 2, HEAD desacoplado → 2, `main` por delante de `origin/main` → 2, restaurado y sin `origin/main` → 0. `go build ./...` sin errores.
- **Activación:** la ruta registrada en el `settings.json` local apunta a `C:/repos/axiom/scripts/claude-hooks/...`, que solo existirá en el checkout principal tras el merge y el `done`/fast-forward. Hasta entonces el hook falla de forma no bloqueante (el script no existe).
