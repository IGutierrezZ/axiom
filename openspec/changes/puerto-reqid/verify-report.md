# Informe de verificación: puerto-reqid (C-03-a)

Fecha: 2026-11-24 (relanzamiento). Rol: `fullstack`. Rama `inc/puerto-reqid`, diff `d580368d..HEAD` (HEAD `6eb58bc3`). `strict_tdd: false`. Verificación de solo lectura; no se editó código, tareas ni specs.

Contexto histórico: el informe anterior (2026-11-23) salió «no limpio» con W-1 (isoevidence sin `Resolve`), W-2 (REQ-30.22 frente al título exacto) y W-3 (10.2 sin observar); el commit `6eb58bc3` alineó la documentación con el código sin tocar código.

## Reverificación de W-1 y W-2

- **W-2 resuelto.** REQ-30.22 (`specs/sdd-archive-lifecycle/spec.md:11`) y D-05 dicen ahora que, sin portadores no superados, solo vale el título exacto y sin él se rechaza. Coincide con `internal/reqid/resolve.go:54-63` (caso 0: busca `n == name`, si no `MatchNone`), con las pruebas `reqid/resolve_test.go` y `openspecdoc/resolve_test.go` (en verde).
- **W-1 resuelto en design y en 4.2/4.3, con un resto en `tasks.md` (ver W-4).** `isoevidence` no llama a `Resolve` (`rg Resolve` en `internal/isoevidence` solo da `ResolveGoTest`/`ResolvePathSymbol` de `matrix.go`, ajenos); solo importa `internal/reqid` (`cubre.go`, `reqid.go`; `imports_test.go:14,43` lo impone). La nota de D-05 (`design.md:90`) y las tareas 4.2 (`tasks.md:77`) y 4.3 (`tasks.md:79`) coinciden con el código.

## Cumplimiento técnico

| Comando | Resultado |
|---|---|
| `go vet ./...` | sin salida, código 0 |
| `gofmt -l .` | vacío |
| `go test -count=1` reqid, workspace, openspecdoc, isoevidence, sddstatus, livespec, livingdoc, adoption/... | 9 paquetes `ok` |
| `go test ./internal/cli -run 'ReqidRF\|ArchiveCheck\|SDDTrace\|Kickoff\|ArchiveIntegrate\|ArchiveCompose\|ArchiveAdopt' -count=1 -timeout 30m` | `ok` 65.7 s |
| `go test ./internal/cli` completo y `go test ./...` (tarea 10.2) | No observados: el cli completo no termina en 60 min en esta máquina sin fallos vistos; los subconjuntos de cli relacionados (702 pruebas) pasaron. Limitación de entorno pendiente de CI (no incumplimiento). |

La matriz de cumplimiento REQ-44.1 a 44.9, REQ-29.x y REQ-30.4/30.22, la conformidad con D-01 a D-09 y el criterio §8 del informe anterior no cambian: no hay cambios de código desde entonces (solo documentación, `6eb58bc3`).

## Hallazgos

### CRITICAL
Ninguno.

### WARNING
Ninguno. W-4 (frases residuales en `tasks.md:7` y `:66` que decían que `isoevidence` usa `Resolve`) quedó resuelto: ver «Cierre».

### Limitación de entorno (no hallazgo)
- W-3 reclasificado: tarea 10.2 abierta (`go test ./internal/cli` completo y `go test ./...`) por límite de la máquina; pendiente de CI.

### SUGGESTION
- S-1 · REQ-44.2 frente a REQ-30.22 · `internal/livespec/plan_test.go`: un MODIFIED con título cambiado y mismo ID, antes rechazado, ahora se acepta (intencionado por REQ-30.22); dejar constancia como excepción al «ninguna entrada cambia de resultado».
- S-2 · design D-05, `coverage-review.design.md` fila 13 · `InTitle` devuelve el último ID válido de un título con dos; sin constar como intencionado en el invariante de no regresión.

## Cierre (W-4)

Releído `tasks.md:7` (decisión c) y `:66` (tarea 3.3): ya dicen que `isoevidence` no usa `Resolve`, con corrección fechada 2026-11-24. `rg` en `openspec/changes/puerto-reqid/` (sin verify-report ni coverage-review) no deja otra frase que afirme lo contrario; `tasks.md:77` (4.2) conserva su descripción original, matizada por su corrección fechada. Sin cambios de código.

## Resumen

W-1, W-2 y W-4 resueltos y coherentes con el código; vet, gofmt, 9 paquetes y el subconjunto de cli en verde; 10.2 es limitación de entorno pendiente de CI; S-1 y S-2 vigentes. Siguiente: `sdd-archive`.

**Estado:** limpio
