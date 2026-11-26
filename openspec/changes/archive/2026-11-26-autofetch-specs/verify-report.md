# Informe de verificación: autofetch-specs (C-05-d)

> Rol único `fullstack`. Alcance: commits propios de `inc/autofetch-specs` (T1 66776395, T2 dd82958e, T3 869dc142, T4 81af3089, T5 d28adf84, T6 08f351ca, T7 e56a87bd, T8 ba164c0d, T9 94d0ba9d, T10 7224026f), excluida la integración de `origin/main` (c82fd3cf). Modo TDD estándar. Solo pruebas localizadas.

## Estado de las tareas

T1 a T10 marcadas `[x]` en `tasks.md` (10/10), con evidencia por tarea y coherente con el apply-progress de Engram (`sdd/autofetch-specs/apply-progress`, #2679). No hay trabajo sin terminar.

## Cumplimiento técnico

| Dimensión | Resultado |
|-----------|-----------|
| Diseño | Conforme: único punto de ejecución de git (`gitcommon.RunWithTimeout`, guardas `TestArchitecture` e `TestIsolation` en verde); identidad del proyecto solo por ficheros; sello atómico en `<commonDir>/axiom/autofetch/v1/state.json`; verbo `axiom autofetch` fuera de la familia `branch`. |
| Spec | Conforme en RF-BRA-39, RF-BRA-40, RF-BRA-01/10/13 (modificados), RF-HUB-07/13 (ver matriz). |
| Build | `go vet` de los 12 paquetes tocados: sin salida (exit 0). `gofmt -l internal cmd`: vacío. |
| Pruebas | Todas en verde (lista en la matriz). |
| Cobertura | No medida (no se pidió `-cover`); la cobertura funcional se evalúa por escenario en la matriz. |
| Estilo | `gofmt` limpio; comentarios y mensajes en castellano; identificadores en inglés; sin atribución de IA en los commits. |

### Comandos ejecutados (todos exit 0)

- `go vet ./internal/hub ./internal/branch/... ./internal/cli ./cmd/axiom ./internal/components/sdd ./internal/components/uninstall ./internal/components/agentguidance ./internal/assets ./internal/workspace ./internal/gitcommon ./internal/sddpublish`
- `go test ./internal/hub ./internal/gitcommon ./internal/sddpublish ./internal/assets ./internal/components/agentguidance ./internal/workspace ./internal/components/uninstall`: ok (paquetes completos, no solo `-run`).
- `go test ./internal/branch ./internal/branch/gitcli`: ok.
- `go test ./internal/branch/engine -run "Autofetch|Prefs"`: ok (128 s).
- `go test ./internal/branch/engine -run "EnsureAutofetch|EnsureFetch"`: ok.
- `go test ./internal/cli -run Autofetch`: ok.
- `go test ./cmd/axiom -run "Autofetch|BranchVerbs"`: ok.
- `go test ./internal/components -run TestGolden`: ok (en caché, mismos fuentes que T9).
- `go test ./internal/components/sdd -run "Autofetch|BranchGuard|TestSyncNever"`: ok.

No ejecutado: `internal/branch/engine -run "Ensure|Archive"` completo (unos 4 min; T3 lo registra en verde y `EnsureAutofetch|EnsureFetch` pasa ahora) ni la suite entera (regla del proyecto).

## Puntos de atención

- (a) `WaitDelay` de 500 ms en `gitcommon.RunWithTimeout`: correcto y seguro. `ErrWaitDelay` solo se trata como éxito si git salió con código 0 y `runCtx.Err() == nil`; un código de salida distinto de cero sigue llegando como `ExitError`, y un tiempo agotado o cancelación se clasifican después (`ErrTimeout` o interrupción). Afecta a `guard_warn`, `gitidentity`, `sddpublish` (push) y `FetchWithin`: en todos el único cambio es que un hijo colgado ya no retiene la llamada; no hay falso éxito. Sin prueba propia en `internal/gitcommon` (SUGGESTION 1).
- (b) `engine.SetAutofetchRunBudget`: exportada solo para pruebas (comentario lo declara), sin uso de producción (grep: solo `autofetch_test.go:593`). Aceptable; WARNING menor de superficie pública (SUGGESTION 2).
- (c) Cotas: fetch 8 s (`autofetchFetchTimeout`, recortado al resto del presupuesto), presupuesto global 10 s con `context.WithTimeout` y `recover`, reserva de 1 s para el avance; hook con `|| true` / `; exit 0` y la prueba del hook con remoto que no responde termina en 0 dentro de la cota. Verificado por pruebas (`AutofetchBudget`, CLI) y lectura. No se midió con la cota real de 10 s (se redujo a 600 ms en la prueba).
- (d) Interruptor apagado: `autofetchEnabled`/`Autofetch` salen antes de cualquier git; pruebas con `Observe` vacío (run, hook, `ensure` byte a byte) en verde. `scope`, `done` y `status` sin red.
- (e) `FastForward` es solo `merge --ff-only`; no hay `push`, `pull`, `ls-remote` ni `--prune` en el código (grep) ni en las trazas de `Observe` de las pruebas; ramas de cambio, de archivado, `main_branch` distinta y repositorios de código no se mueven (pruebas T5/T6).
- (f) Doctrina: la misma frase y el mismo párrafo previo a cada parada aparecen en `routing.go`, `sdd-orchestrator-sections.md`, `sdd-apply/SKILL.md`, AGENTS.md, GEMINI.md y el manual; los 10 goldens cambian exactamente +2 líneas cada uno, sin deriva.

## Matriz requisito → escenario → prueba

| Requisito | Escenarios | Prueba | Resultado |
|-----------|-----------|--------|-----------|
| RF-BRA-39 (ciclo, avance, avisos, prohibiciones, intervalo, remoto inalcanzable) | 22 escenarios | `engine -run Autofetch(Cycle\|Stamp\|Project\|Budget\|Relevance\|Advance)`, `cli -run Autofetch` | PASS |
| RF-BRA-40 (hook, inercia, doctrina) | 8 escenarios | `sdd -run Autofetch`, `uninstall` (paquete), `cli` modo hook, `assets`, `agentguidance`, `TestGolden` | PASS (R-5 Codex solo manual, declarado) |
| RF-BRA-01 (familia, único punto) | 10 escenarios | `cmd/axiom BranchVerbs`, `gitcommon TestArchitecture`, `sddpublish TestIsolation` | PASS |
| RF-BRA-10 (fetch implícito) | 16 escenarios | `engine -run EnsureAutofetch\|EnsureFetch`, `gitcli FetchWithin` | PASS |
| RF-BRA-13 (`--archive`) | 16 escenarios | `engine EnsureAutofetch`, `branch ChangeNotIntegrated` | PASS (la batería `Archive` completa no se relanzó) |
| RF-HUB-07 (preferencia) | 14 escenarios | `hub` (paquete), `engine -run Prefs`, `cli` | PASS |
| RF-HUB-13 (`enable`/`disable`/`status`) | 8 escenarios | `cli -run Autofetch`, `hub` | PASS |

## Hallazgos

- CRITICAL: ninguno.
- WARNING: ninguno.
- SUGGESTION 1: añadir una prueba en `internal/gitcommon` que fije `WaitDelay` y el tratamiento de `ErrWaitDelay` (hoy solo la cubren indirectamente las pruebas del hook y de `FetchWithin`).
- SUGGESTION 2: valorar mover `SetAutofetchRunBudget` a un fichero de soporte de pruebas o paquete `internal`, para no exponer API pública solo de pruebas.
- SUGGESTION 3: no se verificó con la cota real de 10 s ni el contrato `SessionStart` de Codex (R-5, manual).

**Estado:** limpio
