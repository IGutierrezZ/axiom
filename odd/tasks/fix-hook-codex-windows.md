# ODD: hooks de Codex que salen con 0 en Windows (fix-hook-codex-windows)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/fix-hook-codex-windows.md`. Espejo Engram: `odd/fix-hook-codex-windows/tasks`.
> Rama: `fix/fix-hook-codex-windows` · worktree `C:\repos\axiom-wt\fix-hook-codex-windows\axiom`.

Tipo: bug

## Objetivo

Que los hooks `SessionStart` que `axiom sync` instala en el `hooks.json` de Codex terminen siempre con 0 también en Windows, como exige RF-BRA-40.

## Problema y porqué

En Windows, Codex ejecuta cada hook con `%COMSPEC%` (`cmd.exe /C`); fuente: `codex-rs/hooks/src/engine/command_runner.rs`, `default_shell_program`. Las órdenes que instala Axiom para Codex usan sintaxis POSIX:

- autofetch: `axiom autofetch run --agent codex || true`;
- skill-registry: `axiom skill-registry refresh --quiet --no-gitignore --cwd "$PWD" || true`.

`cmd.exe` no tiene `true` ni expande `$PWD`. Con un PATH de Windows limpio y `axiom` ausente (o sin el verbo), `cmd /C "... || true"` sale con 1 y Codex muestra «hook exited with code 1» en cada arranque. Solo funciona donde Git Bash añade `/usr/bin/true` al PATH. No bloquea la sesión, pero incumple RF-BRA-40 («termina siempre en 0»).

Codex admite `commandWindows` en cada handler (`codex-rs/core/config.schema.json`, `HookHandlerConfig`; `codex-rs/hooks/src/engine/discovery.rs` usa `command_windows.unwrap_or(command)`).

## Decisiones

- **`commandWindows` en los handlers de Codex** con una forma de `cmd.exe` que sale siempre con 0: `<orden> 2>nul & exit /b 0` (comprobada a mano). `command` conserva la forma POSIX, así que la misma entrada sirve en Linux, macOS y Windows.
- **skill-registry sin `--cwd` en Windows.** `refresh` toma `os.Getwd()` cuando falta `--cwd` (`resolveSkillRegistryDirs`, `internal/app/app.go`), que es lo mismo que `%CD%`. Pasar `--cwd "%CD%"` rompe en la raíz de una unidad: `"C:\"` se lee como comilla escapada al trocear los argumentos.
- **La guarda de rama de Codex no cambia.** `axiom branch guard --agent codex` no lleva sintaxis POSIX, `cmd.exe` la ejecuta tal cual y bloquea con el código 2 (protocolo de `branch_guard.go`). Forzar 0 desactivaría el bloqueo; no aplica RF-BRA-40.
- **Identidad de la entrada.** Una entrada de autofetch solo es canónica si su `commandWindows` coincide con el esperado (vacío en Claude Code). Así `axiom sync` poda la forma anterior sin `commandWindows` y añade la nueva, y una segunda ejecución no cambia nada. En skill-registry, la entrada existente recibe el campo en su sitio.
- **Desinstalación y poda** reconocen el marcador también en `commandWindows`.

## Alcance autorizado

`internal/components/sdd/inject.go` y sus pruebas, `internal/components/uninstall/service.go` y sus pruebas, y este documento.

## Modo TDD

Sin configuración de TDD en el proyecto ni en la sesión: comprobaciones funcionales. Ejecutor: `go test` localizado con `-run` en `./internal/components/sdd/`, `./internal/components/uninstall/` y `go test ./internal/components -run TestGolden`. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:bug`.

## Tareas

- [x] T1 · `commandWindows` en los hooks `SessionStart` de autofetch y skill-registry de Codex, con migración idempotente de instalaciones existentes, poda y desinstalación que reconocen el campo, y pruebas. Ruta: inline (unidad pequeña, sin decisiones abiertas). Evidencia: `go test ./internal/components/sdd/ -run 'Autofetch|Codex|SkillRegistry|BranchGuard'` ok (incluye `TestCodexHookCommandWindowsExitsZeroInCmdWithoutAxiom`, solo en Windows: con `axiom` ausente y PATH de `System32`, las dos formas `commandWindows` salen con 0 y la forma `|| true` no); `go test ./internal/components/uninstall/ -run 'Autofetch|SkillRegistry|Hook|BranchGuard'` ok; `go test ./internal/components -run TestGolden` ok sin regenerar; `go test ./internal/cli/ -run Codex` ok. `axiom archive integrate --odd fix-hook-codex-windows`: sin delta, impacto declarado.
  Cubre: RF-BRA-40

## Progreso

T1 hecha en `ffc676210`; `go test ./internal/idmigration/` ok tras el commit. Siguiente paso: PR con etiqueta `type:bug`.

Nota: `json.MarshalIndent` escribe `>` y `&` como `>` y `&` en `hooks.json`; es JSON equivalente y Codex lo lee igual.

## Criterio de terminado

- El `hooks.json` de Codex lleva `commandWindows` con `& exit /b 0` en los handlers de autofetch y skill-registry; `command` sigue en forma POSIX.
- Un `hooks.json` previo sin el campo queda actualizado tras `axiom sync`, y una segunda ejecución no cambia nada.
- La desinstalación retira los handlers con el campo nuevo.
- Las pruebas localizadas pasan y la CI está en verde.

## Impacto en la spec

Sin impacto en la spec
Motivo: restaura RF-BRA-40 de `change-branch-lifecycle` («termina siempre en 0»), que ya existe y no cambia.
