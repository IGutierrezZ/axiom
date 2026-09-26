# ODD: Sincronización Upstream Fase 1 — Estabilidad en Windows y Concurrencia de Estado

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/sync-upstream-phase1-windows-state.md`.  
> Espejo de recuperación en Engram: topic `odd/sync-upstream-phase1-windows-state/tasks`, proyecto `axiom`.

## Objetivo

Portar selectivamente desde upstream Gentle AI (v3.4.0..v3.7.0+) los tres parches críticos de estabilidad en entorno Windows y concurrencia de estado, preservando de forma absoluta la soberanía de Axiom, su módulo canónico (`github.com/IGutierrezZ/axiom/v3`), sus contratos desacoplados y su Flujo Dual:

1. **Serialización atómica de escritores de estado (`internal/app/app.go`, `selfupdate.go`):**
   - Asegurar que `persistAssignments`, `RunArgs` (limpieza de `PendingSync` post-sync diferido) y `markPendingSyncAfterSelfUpdate` ejecuten su ciclo de lectura-modificación-escritura íntegramente bajo el cerrojo canónico `statecoord.WithLock`.
   - Evitar condiciones de carrera donde escrituras concurrentes sobrescriban o descarten selecciones de modelo o marcas de estado.
2. **Hook de Claude `UserPromptSubmit` compatible con PowerShell en Windows (`internal/components/sdd/inject.go`):**
   - Corregir el comando de hook para Claude Code en Windows, sustituyendo la sintaxis bash no soportada (`|| true`, `${CLAUDE_PROJECT_DIR:-$PWD}`) por un comando seguro ejecutado en PowerShell que compruebe la variable de entorno, comille la ruta `$dir` y termine limpiamente con `exit 0`.
   - Implementar migración automática que elimine ganchos POSIX antiguos antes de registrar el canónico de Windows.
3. **Cancelación limpia de árboles de subprocesos en Windows (`internal/opencode/`):**
   - Implementar en `catalog_process_windows.go` la asignación del proceso hijo a un Windows Job Object con `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`.
   - Garantizar que al expirar el deadline o cancelar el contexto de descubrimiento de modelos en OpenCode se termine todo el árbol de procesos huérfanos sin bloquear descriptores de pipe.

---

## Tareas

- [x] **T1 · Serialización de Escritores de Estado bajo `statecoord.WithLock` (`internal/app/`)**
  - Implementar `clearPendingSyncAfterDeferredSync` bajo `statecoord.WithLock` en `internal/app/app.go`.
  - Envolver la lectura y mutación de `persistAssignments` dentro de `statecoord.WithLock` en `internal/app/app.go`.
  - Implementar `markPendingSyncAfterSelfUpdate` bajo lock en `internal/app/selfupdate.go`.
  - Añadir suite de pruebas de contención y preservación en `internal/app/state_lock_test.go`.
- [x] **T2 · Hook de Claude `UserPromptSubmit` seguro para PowerShell en Windows (`internal/components/sdd/`)**
  - Adaptar `ensureClaudeSkillRegistryHook` en `internal/components/sdd/inject.go` para generar el comando PowerShell en Windows y POSIX en Unix.
  - Implementar migración y purga de hooks POSIX preexistentes en Windows.
  - Añadir tests unitarios de generación y migración en `internal/components/sdd/inject_test.go`.
- [ ] **T3 · Terminación de Árbol de Procesos en Windows con Job Objects (`internal/opencode/`)**
  - Implementar `configureProcessGroup` con Windows Job Object en `internal/opencode/catalog_process_windows.go`.
  - Actualizar `catalog_process_unix.go` y `catalog.go` para enlazar los ganchos `afterStart` y `release`.
  - Añadir test `TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows` en `catalog_process_windows_test.go`.
- [ ] **T4 · Verificación de Calidad y Pruebas Globales**
  - Ejecutar `go test ./internal/app/...` -> PASS.
  - Ejecutar `go test ./internal/components/sdd/... -run "ClaudeSkillRegistryHook"` -> PASS.
  - Ejecutar `go test ./internal/opencode/... -run "CatalogCommand"` -> PASS.
  - Ejecutar `go test ./...` y `go vet ./...`.

---

## Verificación Ejecutable
(Se registrará con las salidas reales de las pruebas)
