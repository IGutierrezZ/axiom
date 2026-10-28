# Axiom — Regla local de worktrees para Claude Code

> **Alcance:** política local para desarrollar *este* repositorio. No describe el comportamiento del producto: en otros proyectos Axiom prepara las ramas y los worktrees con `axiom branch` según la topología y el modo de trabajo personal de cada usuario (en multirepo, el repositorio de specs trabaja en la rama de integración).

## Regla

La topología de este repositorio es `monorepo-embedded` (`axiom.yaml`): `openspec/` comparte repositorio con el código. Por tanto, **todo cambio** (código, artefactos de `openspec/` y documentos `odd/tasks/`) nace **desde la primera escritura** en un worktree propio que prepara `axiom branch ensure`. El envoltorio local lo invoca:

```
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/axiom-worktree.ps1 new <slug> [-Prefix feat|inc|fix|refactor]
```

- Ejecuta `axiom branch ensure --cwd <checkout principal> --change <slug> --mode worktree --json` y muestra la ruta del worktree: `C:\repos\axiom-wt\<slug>\axiom`, sobre la rama `<prefijo>/<slug>` (el prefijo sale de `git.branch_prefix` en `axiom.yaml`, `inc`, salvo `-Prefix`) desde `origin/main`, sin red.
- PR: `... scripts/axiom-worktree.ps1 pr <slug>` (la base sale de `prTarget` en `axiom branch status --json`).
- Tras el merge: `... scripts/axiom-worktree.ps1 done <slug>` (`axiom branch done` y avance rápido de `main`).
- Necesita un `axiom` con `axiom branch` en el `PATH` (`go install ./cmd/axiom`).

## Checkout principal `C:\repos\axiom`

- Está **siempre en `main`** y solo recibe fast-forwards.
- **Prohibido** ahí: `git checkout -b`, `git switch -c`, `git checkout <rama>`, hacer commits y editar ficheros versionados.
- El paso de implementación de ODD (`axiom branch ensure --lane odd --change <feature>`) se cumple **creando el worktree** (`--mode worktree`), nunca ramificando en el directorio actual.
- Si la sesión arranca en el checkout principal, crea primero el worktree y trabaja con las rutas absolutas del JSON de `ensure` (`--cwd`, `git -C`).
- `EnterWorktree` / `ExitWorktree` están prohibidos: crean rutas en `.claude/worktrees/`, dejan procesos huérfanos y no son interoperables con otros clientes (véase `AGENTS.md`).

## Guarda mecánica

El hook `scripts/claude-hooks/guard-primary-checkout.ps1` deniega escrituras (`Write|Edit|NotebookEdit`) dentro del checkout principal y avisa si un comando lo deja fuera de `main` o con commits locales no presentes en `origin/main`. Complementa a `axiom branch guard` (el hook del producto que instala `axiom sync` y bloquea los documentos de Axiom en la rama equivocada), que no cubre Bash ni PowerShell. En otra máquina hay que registrarlo en `.claude/settings.json` (local, no versionado); el fragmento exacto está en la cabecera del script.

Reglas completas del ciclo de PRs: `AGENTS.md` (sección ODD-5).
