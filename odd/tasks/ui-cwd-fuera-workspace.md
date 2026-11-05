# ODD: el servidor de `axiom ui` no retiene el workspace como directorio de trabajo

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/ui-cwd-fuera-workspace.md`.
> Espejo de recuperación en Engram: topic `odd/ui-cwd-fuera-workspace/tasks`, proyecto `axiom`.
> Rama: `fix/ui-cwd-fuera-workspace` (worktree `axiom-wt/ui-cwd-fuera-workspace`).

## Objetivo

Que el proceso servidor de `axiom ui` (en primer plano o desacoplado con `--detach`) no fije su directorio de trabajo dentro del workspace que sirve.

## Problema y porqué

- `runUI` (`cmd/axiom/main.go`) hace `os.Chdir(baseDir)` y `startDetachedServer` (`cmd/axiom/ui_launch.go`) lanza el hijo de `--detach` con `cmd.Dir = baseDir`.
- En Windows un proceso cuyo cwd está dentro de un directorio impide borrarlo: un servidor `axiom ui --detach` lanzado desde un worktree hace fallar `axiom branch done` / `git worktree remove` con "Permission denied". Ocurrió al cerrar la PR #222.

## Hallazgos de exploración

- `internal/dashboard` trabaja con `rootPath` absoluto (`NewService*` aplica `filepath.Abs`); las llamadas a `git` fijan `cmd.Dir`.
- Únicas dependencias del cwd: `RunSync`, que ya hace su propio `Chdir` temporal a `rootPath` y restaura el anterior, y `BrowseDirectories`, que solo usa `os.Getwd` como último recurso si no hay workspace ni home.
- `RunUpgradeReportWithChannel` solo actúa sobre el binario; no depende del cwd.

## Decisiones

- **Directorio neutro**: el home del usuario; si no se puede resolver o cae dentro del workspace, el directorio temporal; si también cae dentro, la raíz del volumen del workspace.
- El hijo de `--detach` arranca con `cmd.Dir` en ese directorio neutro y `runUI` hace `Chdir` a él en lugar de a `baseDir`. `baseDir` sigue llegando explícito al servicio y al hijo (`--path`).

## Alcance autorizado

`cmd/axiom` (`runUI`, `ui_launch.go` y sus tests).

## Restricciones

- No compilar binarios sueltos: verificación solo con `go test` y `go vet`.
- TDD: desactivado (sin configuración de TDD en `axiom.yaml`). Runner: `go test`. Comprobaciones: `go test ./cmd/axiom/... ./internal/dashboard/...` y `go vet ./...`.
- RDD: desactivada por defecto.

## Tareas

- [x] **T1 · Directorio de trabajo neutro para el servidor** — ruta: en línea (1 fichero de código y su test, ya entendidos).
  - `neutralWorkDir(baseDir, candidatos...)` y su uso en `startDetachedServer` y `runUI`.
  - Tests: elección del candidato, exclusión de candidatos dentro del workspace, raíz como último recurso y `cmd.Dir` del hijo fuera del workspace.
- [x] **T2 · PR contra `main`** con etiqueta `type:bug`.

## Criterios de aceptación

- El servidor no retiene `baseDir` como cwd.
- `go test ./cmd/axiom/... ./internal/dashboard/...` y `go vet ./...` en verde.

## Progreso y evidencias

- T1: `go test ./cmd/axiom/... ./internal/dashboard/...` → `ok` en ambos paquetes; `go vet ./...` sin hallazgos; `gofmt -l cmd/axiom` vacío. Tests nuevos: `TestNeutralWorkDir`, `TestServerWorkDirOutsideWorkspace`, `TestDetachedServerCommandWorkDir`. Commit: `fix(ui): servidor de axiom ui con cwd fuera del workspace`.
- T2: PR IGutierrezZ/axiom#223 abierta contra `main` con `type:bug`; CI en verde (9 correctos, 5 omitidos, 0 fallidos).
- RDD: desactivada; sin evaluación de riesgo.
- Siguiente paso: fusión (decisión del usuario) y `scripts/axiom-worktree.ps1 done ui-cwd-fuera-workspace`.

## Seguimiento fuera de alcance

- `RunSync` (`internal/dashboard/service.go`) hace `os.Chdir` global y temporal al workspace dentro de un servidor HTTP concurrente; si `os.Getwd` falla, no restaura. Conviene pasar el directorio explícito a `sync` en lugar de cambiar el cwd del proceso.
