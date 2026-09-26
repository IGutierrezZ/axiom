# ODD: Aislamiento y Exclusión de Worktrees en el Hub de Proyectos de Axiom

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/worktree-hub-isolation.md`.  
> Espejo de recuperación en Engram: topic `odd/worktree-hub-isolation/tasks`, proyecto `axiom`.

## Objetivo

Evitar que los worktrees git temporales (por ejemplo `C:\repos\axiom-wt\<slug>`) se registren como proyectos independientes en el catálogo global de workspaces de Axiom (`~/.axiom/workspaces.json`), y garantizar que el ciclo de limpieza (`scripts/axiom-worktree.ps1 done` y `scripts/axiom-worktree.sh done`) desvincule cualquier registro si por alguna razón hubiese sido registrado.

1. **Detección Confiable de Worktrees (`internal/hub`):**
   - Función canónica `IsGitWorktree(path string) bool` que analiza si `.git` es un archivo con puntero `gitdir:` (característica estándar de los worktrees vinculados) o si la ruta pertenece al contenedor de worktrees de Axiom (`axiom-wt`).
2. **Defensa en Registro y Saneamiento (`internal/hub/manager.go`):**
   - `Manager.Register`: rechazar explícitamente el registro de worktrees con error descriptivo.
   - `Manager.Prune`: nuevo método para purgar del catálogo global rutas inexistentes o worktrees huérfanos.
3. **Prevención de Auto-Registro en Dashboard (`internal/dashboard/service.go`):**
   - `NewService`: comprobar `!hub.IsGitWorktree(rootPath)` antes de auto-registrar cualquier espacio de trabajo que contenga `axiom.yaml`.
   - `RegisterProject`: rechazar explícitamente si se intenta registrar un worktree vía API REST / Dashboard.
4. **Comando CLI de Purga (`cmd/axiom/main.go`):**
   - `axiom project prune`: comando para limpiar entradas inválidas o worktrees residuales en `workspaces.json`.
5. **Limpieza en Scripts de Ciclo de Vida (`scripts/axiom-worktree.ps1` y `.sh`):**
   - En el verbo `done`, ejecutar `axiom project remove <slug>` y `<wtPath>` para garantizar que ningún worktree desmantelado quede en el Hub.
6. **Verificación y Pruebas Unitarias:**
   - Tests exhaustivos en `internal/hub` e `internal/dashboard`.

---

## Tareas

- [x] **T1 · Detección de Worktrees (`internal/hub/detector.go` & `types.go`)**
  - Implementar `IsGitWorktree(path string) bool` y pruebas unitarias asociadas.
- [x] **T2 · Bloqueo de Registro y Método `Prune()` (`internal/hub/manager.go`)**
  - Impedir que `Register` acepte un worktree.
  - Implementar `Prune()` en `Manager` y tests en `internal/hub/hub_test.go`.
- [x] **T3 · Protección contra Auto-Registro en Servicio de Dashboard (`internal/dashboard/service.go`)**
  - Condicionar el auto-registro en `NewService` con `!hub.IsGitWorktree(rootPath)`.
  - Proteger `AddProject`.
- [x] **T4 · Comando CLI `axiom project prune` (`cmd/axiom/main.go`)**
  - Registrar el comando en la CLI de Axiom con salida formateada.
- [x] **T5 · Saneamiento en Scripts de Worktrees (`scripts/axiom-worktree.ps1` y `.sh`)**
  - Añadir paso de desvinculación en el verbo `done`.
- [x] **T6 · Batería de Pruebas y Verificación Integral**
  - Ejecutar tests de Go y validar con fixtures reales de worktrees.

---

## Verificación Ejecutable

- `go test -v ./internal/hub/... -run "TestIsGitWorktree|TestManagerWorktreeIsolationAndPrune"` -> PASS
- `go test -v ./internal/dashboard/... -run TestDashboardWorktreeExclusion` -> PASS
- `go test ./internal/hub/... ./cmd/axiom/...` -> PASS
- `go vet ./internal/hub/... ./internal/dashboard/... ./cmd/axiom/...` -> PASS (0 advertencias)
- `go run ./cmd/axiom project add .` (dentro de worktree) -> Bloqueado con error descriptivo: `la ruta '...' corresponde a un worktree git; los worktrees no deben registrarse como proyectos en el Hub`.
- `go run ./cmd/axiom project prune` -> Operativo y reporta catálogo limpio.
