# ODD: Sincronización Upstream Fase 2 — Motor RDD: Detección de Riesgos en Líneas Añadidas y Convergencia de Autoridad

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/sync-upstream-phase2-rdd.md`.  
> Espejo de recuperación en Engram: topic `odd/sync-upstream-phase2-rdd/tasks`, proyecto `axiom`.

## Objetivo

Portar selectivamente desde upstream Gentle AI (v3.4.0..v3.7.0+) las optimizaciones críticas del motor de revisión RDD (`internal/reviewtransaction/`), preservando la soberanía de Axiom y sus contratos formales:

1. **Convergencia de Bloqueo de Autoridad en RAR (`internal/reviewtransaction/rar_authority_repository.go`):**
   - Corregir el comportamiento ante agotamiento del cerrojo consultivo del repositorio RAR (`LOCK`): si una publicación concurrente idéntica ya publicó con éxito el par inmutable recibo+resultado solicitado y este sigue siendo válido, converger hacia la autoridad publicada en lugar de fallar por timeout espurio (#3239).
   - Asegurar que la liberación del cerrojo de mantenimiento del recibo nativo ocurra antes de resolver la liveness para evitar deadlocks de re-adquisición.
2. **Clasificación de Riesgos sobre Líneas Añadidas y Catálogo de Sumideros Peligrosos (`internal/reviewtransaction/`):**
   - Incorporar `SignalDangerousSink` y el catálogo tipado de patrones de sumideros peligrosos (`risk_dangerous_sink.go`) clasificados por lenguaje y CWE (CWE-295 TLS inseguro, CWE-502 deserialización no confiable, CWE-94 eval/código dinámico, CWE-78 inyección de shell, CWE-327 hashes débiles de credenciales, CWE-732 permisos mundiales, etc.).
   - Modificar `processBoundaryRiskReasons` en `risk.go` para analizar las llamadas de spawn y los sumideros peligrosos **únicamente sobre las líneas añadidas (`+`)** del diff unificado, ignorando archivos de test (`isTestRiskPath`).
   - Evitar falsos positivos de riesgo alto en código preexistente no modificado, manteniendo escaneos acotados y fail-closed ante diffs desmedidos.

---

## Tareas

- [x] **T1 · Convergencia de Bloqueo de Autoridad RAR (`internal/reviewtransaction/rar_authority_repository.go`)**
  - Implementar predicado unificado `converge` en `Publish`.
  - Asegurar liberación previa `releaseOnce` del lock de recibo nativo antes de evaluar convergencia bajo contención de `LOCK`.
  - Portar y ejecutar suite de pruebas de convergencia en `rar_authority_repository_test.go`.
- [ ] **T2 · Catálogo de Sumideros Peligrosos y Escaneo en Líneas Añadidas (`internal/reviewtransaction/`)**
  - Implementar `internal/reviewtransaction/risk_dangerous_sink.go` con `dangerousSinkCatalog`, `dangerousSinkLine` y detector de YAML inseguro.
  - Actualizar `internal/reviewtransaction/risk.go` con `SignalDangerousSink`, `RiskReasonDangerousSink`, `isTestRiskPath` y análisis del diff acotado sobre líneas añadidas.
  - Incorporar pruebas unitarias en `risk_dangerous_sink_test.go` y `risk_process_boundary_test.go`.
- [ ] **T3 · Verificación de Calidad y Pruebas Globales**
  - Ejecutar `go test -v ./internal/reviewtransaction/... -run "TestRARVerificationAuthorityConverges|TestRARVerificationAuthorityLockExhaustion"` -> PASS.
  - Ejecutar `go test -v ./internal/reviewtransaction/... -run "TestDangerousSink|TestProcessBoundary"` -> PASS.
  - Ejecutar `go test ./internal/reviewtransaction/...` -> PASS.
  - Ejecutar `go vet ./internal/reviewtransaction/...` -> PASS.
  - Compilar `cmd/axiom`.

---

## Verificación Ejecutable

### T1: Convergencia de Bloqueo de Autoridad RAR
```
=== RUN   TestRARVerificationAuthorityConvergesOnExhaustedRepositoryLock
--- PASS: TestRARVerificationAuthorityConvergesOnExhaustedRepositoryLock (5.93s)
=== RUN   TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped
=== RUN   TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/no_published_pair
=== RUN   TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/divergent_contracts_at_the_exact_pair
=== RUN   TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/native_receipt_lock_held_at_the_exact_pair
=== RUN   TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/stale_live_receipt_refuses_convergence_at_the_exact_pair
--- PASS: TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped (18.21s)
    --- PASS: TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/no_published_pair (3.79s)
    --- PASS: TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/divergent_contracts_at_the_exact_pair (5.17s)
    --- PASS: TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/native_receipt_lock_held_at_the_exact_pair (6.69s)
    --- PASS: TestRARVerificationAuthorityLockExhaustionWithoutConvergentPairStaysTyped/stale_live_receipt_refuses_convergence_at_the_exact_pair (2.56s)
PASS
ok  	github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction	24.272s
```
