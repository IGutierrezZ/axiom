# ODD: Sincronización Upstream Fase 3 — Roles Nativos en Model Picker TUI (Claude, OpenCode, Codex)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/sync-upstream-phase3-models.md`.  
> Espejo de recuperación en Engram: topic `odd/sync-upstream-phase3-models/tasks`, proyecto `axiom`.

## Objetivo

Portar selectivamente desde upstream Gentle AI (v3.4.0..v3.7.0+, PR #4912) las capacidades de configuración de roles nativos en la interfaz TUI (Model Picker) a lo largo de los tres entornos de agentes principales (Claude Code, OpenCode y Codex), preservando los contratos canónicos de Axiom y la soberanía del flujo dual (ODD + SDD):

1. **Claude Model Picker:**
   - Incorporar en el picker custom de Claude la selección de modelos para los seis roles RDD (`rdd-risk`, `rdd-readability`, `rdd-reliability`, `rdd-resilience`, `rdd-refuter`, `rdd-validator`).
   - Implementar viewport scroll con seguimiento de cursor para navegación fluida en terminales de 12 líneas.
   - Pasar el modelo configurado al proceso aislado de Claude (`--model <model>`) en el adaptador de revisión RDD (`internal/reviewerprovider/claude_adapter.go`).
2. **OpenCode Model Picker:**
   - Permitir editar individualmente los agentes nativos gestionados `general` y `explore` en el selector de modelos global de OpenCode, preservando perfiles SDD y acciones masivas.
3. **Codex Model Picker:**
   - Añadir roles ODD (Explorer, Worker, Verify) y los seis roles RDD con persistencia y defaults seguros basados en la familia GPT-6 (`gpt-6-astra`, `gpt-6-sol`, `gpt-6-luna`).
   - Cablear la resolución de modelos de roles RDD hacia `codex exec --model` en el adaptador de revisor Codex.

---

## Tareas

- [x] **T1 · Roles RDD en Claude Model Picker y Adaptador de Revisor**
  - Extender `internal/tui/screens/claude_model_picker.go` con las seis filas RDD y viewport con cursor-following en terminales cortos.
  - Actualizar `internal/reviewerprovider/claude_adapter.go` para transmitir `--model <modelo>` al subproceso de Claude.
  - Ejecutar y verificar tests en `internal/tui/screens/claude_model_picker_test.go`, `internal/reviewerprovider/claude_adapter_test.go`, `internal/cli/review_provider_runtime_model_test.go` e `internal/components/sdd/inject_test.go`.
- [ ] **T2 · Roles Nativos General y Explore en OpenCode Model Picker**
  - Extender `internal/tui/screens/model_picker.go` e `internal/components/sdd/inject.go` para descubrir y editar los agentes nativos `general` y `explore`.
  - Ejecutar y verificar tests en `internal/tui/screens/model_picker_test.go` e `internal/components/sdd/inject_test.go`.
- [ ] **T3 · Roles ODD/RDD en Codex Model Picker, Presets GPT-6 y Adaptador**
  - Actualizar `internal/model/codex_model.go` y `internal/tui/screens/codex_model_picker.go` con roles ODD, 6 roles RDD y defaults GPT-6.
  - Conectar los modelos configurados en `internal/reviewerprovider/codex_adapter.go` y `internal/cli/review_provider_runtime.go`.
  - Ejecutar y verificar tests en `internal/model/codex_model_test.go`, `internal/tui/screens/codex_model_picker_test.go` y `internal/reviewerprovider/codex_adapter_test.go`.
- [ ] **T4 · Verificación de Calidad y Pruebas Globales**
  - Ejecutar tests de todos los paquetes modificados.
  - Ejecutar `go vet` sobre los paquetes afectados.
  - Compilar binario `cmd/axiom`.

---

## Verificación Ejecutable

- **T1 Claude Model Picker & Review Adapter Tests:**
  - `go test -v ./internal/tui/screens -run "TestClaude"`: PASS (`TestClaudeShortTerminalShowsFocusedRDDRowsAndConfirm`, `TestClaudeModelPickerPlacesResearchAfterExplore`, `TestClaudePickerSavesAndReopensNativeReviewRoles`).
  - `go test -v ./internal/reviewerprovider -run "TestClaude"`: PASS (`TestClaudeAdapterReturnsNoBytesWhenUnavailable`, `TestClaudeAdapterHelperProcess`).
  - `go test -v ./internal/cli -run "TestClaudeReviewAdapter"`: PASS (`TestClaudeReviewAdapterUsesSavedModelForEachNativeRole`, `TestClaudeReviewAdapterMissingAndInvalidAssignmentsUseNativeDefault`).
  - `go test -v ./internal/components/sdd -run "TestInjectClaude.*Review"`: PASS (`TestInjectClaudeNativeReviewAgentsUseSavedRoleModels`, `TestInjectClaudeReviewAgentFallsBackForInvalidRole`).
