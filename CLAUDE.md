# Axiom — Regla local de worktrees para Claude Code

> **Alcance:** política local para desarrollar *este* repositorio. No describe el comportamiento del producto: en otros proyectos Axiom mantiene el worktree configurable y lo crea en `apply` (en multirepo, el repositorio de specs trabaja en `main`).

## Regla

La topología de este repositorio es `monorepo-embedded` (`axiom.yaml`): `openspec/` comparte repositorio con el código. Por tanto, **todo cambio** (código, artefactos de `openspec/` y documentos `odd/tasks/`) nace **desde la primera escritura** en un worktree propio:

```
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/axiom-worktree.ps1 new <slug> [-Prefix feat|inc|fix|refactor]
```

- Crea `C:\repos\axiom-wt\<slug>` sobre una rama nueva desde `origin/main`.
- PR: `... scripts/axiom-worktree.ps1 pr <slug>`.
- Tras el merge: `... scripts/axiom-worktree.ps1 done <slug>`.

## Checkout principal `C:\repos\axiom`

- Está **siempre en `main`** y solo recibe fast-forwards.
- **Prohibido** ahí: `git checkout -b`, `git switch -c`, `git checkout <rama>`, hacer commits y editar ficheros versionados.
- La instrucción de ODD «branch first when on the default branch» se cumple **creando el worktree**, nunca ramificando en el directorio actual.
- Si la sesión arranca en el checkout principal, crea primero el worktree y trabaja con rutas absolutas dentro de él (o con `git -C`).
- `EnterWorktree` / `ExitWorktree` están prohibidos: crean rutas en `.claude/worktrees/`, dejan procesos huérfanos y no son interoperables con otros clientes (véase `AGENTS.md`).

## Guarda mecánica

El hook `scripts/claude-hooks/guard-primary-checkout.ps1` deniega escrituras (`Write|Edit|NotebookEdit`) dentro del checkout principal y avisa si un comando lo deja fuera de `main` o con commits locales no presentes en `origin/main`. En otra máquina hay que registrarlo en `.claude/settings.json` (local, no versionado); el fragmento exacto está en la cabecera del script.

Reglas completas del ciclo de PRs: `AGENTS.md` (sección ODD-5).
