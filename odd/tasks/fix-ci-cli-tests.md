# ODD: Corrección de Tests de Telemetría, Doctor y Sincronización de Codex en CLI (fix-ci-cli-tests)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/fix-ci-cli-tests.md`.  
> Rama de trabajo: `fix/fix-ci-cli-tests`.

## Objetivo

Corregir los fallos de tests unitarios en el paquete `internal/cli` detectados en la CI de GitHub Actions (Run 36608862386):
1. **Actualizar aserciones de telemetría a `AXIOM_TELEMETRY`**: En `internal/cli/telemetry_test.go`, actualizar los casos de prueba de opt-out y saneamiento de entorno para verificar la variable canónica `AXIOM_TELEMETRY` tras el desacople de `GENTLE_AI_*`.
2. **Excluir Codex de scope workspace en `adapterSupportsWorkspace`**: Codex CLI (`codex-cli`) opera estrictamente con configuración global en `~/.codex` y no admite configuración a nivel de directorio de proyecto. Añadir `model.AgentCodex` a `adapterSupportsWorkspace` para asegurar que `componentInjectionDirScoped` derive siempre a `homeDir` independientemente del scope por defecto de instalación/sincronización (`ScopeWorkspace`), garantizando la convergencia de perfiles y `hooks.json`.
3. **Actualizar aserciones y rutas de `doctor_test.go`**: Actualizar los fixtures que asumían rutas obsoletas `.gentle-ai` a la ruta canónica `.axiom`, y alinear las aserciones de texto con `axiom doctor`, `axiom sync` y `tool:axiom`.
4. **Verificación integral en verde**: Asegurar que toda la suite de `internal/cli` pase sin regresiones antes de abrir PR.

---

## Tareas

- [x] **T1 · Actualizar casos de prueba de telemetría en `internal/cli/telemetry_test.go`**
  - Actualizar `TestTelemetryPolicyReadOnly` para evaluar `AXIOM_TELEMETRY` en `optout` y `nonzero optout`.
  - Incluir saneamiento explícito de `AXIOM_TELEMETRY` en `enableTelemetryForTest` y `telemetryTestHome`.
- [x] **T2 · Excluir `model.AgentCodex` de scope workspace en `internal/cli/run.go`**
  - Añadir `model.AgentCodex` al switch de `adapterSupportsWorkspace`.
- [x] **T3 · Actualizar aserciones y fixtures en `internal/cli/doctor_test.go`**
  - Migrar fixtures de estado de `.gentle-ai/state.json` a `.axiom/state.json`.
  - Actualizar cadenas esperadas de `gentle-ai doctor` / `gentle-ai sync` a `axiom doctor` / `axiom sync`.
  - Actualizar `coreTools` en `TestCheckToolBinaries` para validar `tool:axiom`.
- [x] **T4 · Verificación de suites y pruebas de regresión**
  - Ejecutar `go test -v -run TestTelemetryPolicyReadOnly ./internal/cli` (PASS).
  - Ejecutar `go test -v -run "TestRunSync.*Codex" ./internal/cli` (PASS).
  - Ejecutar `go test -v -run "Test.*Doctor|TestCheck.*|TestRenderDoctor.*" ./internal/cli` (PASS).

---

## Verificación Ejecutable

- `go test -v -run TestTelemetryPolicyReadOnly ./internal/cli` (0.08s - PASS)
- `go test -v -run TestRunSyncCodexVerificationMatchesRuntimeProfileOutput ./internal/cli` (1.86s - PASS)
- `go test -v -run TestRunSync_RestoresCodexEffortAssignments ./internal/cli` (0.92s - PASS)
- `go test -v -run TestRunSync_RestoresCodexPhaseModelAssignments ./internal/cli` (0.86s - PASS)
- `go test -v -run "Test.*Doctor|TestCheck.*|TestRenderDoctor.*" ./internal/cli` (5.43s - PASS)
