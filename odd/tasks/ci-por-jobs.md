# ODD: CI de PR por jobs y shards de Windows rebalanceados (ci-por-jobs)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/ci-por-jobs.md`. Espejo Engram: `odd/ci-por-jobs/tasks`.
> Rama: `inc/ci-por-jobs` · worktree `C:\repos\axiom-wt\ci-por-jobs\axiom`. Cambio C-08-i del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A4; P21 resuelta (C-01-e retiró `bench/` entero).

## Objetivo

Que el check obligatorio `Unit Tests` deje de ser un único job secuencial y que la suite completa de Windows quede equilibrada tras retirar RDD.

## Problema y porqué

- En la CI de una PR, `Unit Tests` encadena `go test ./...`, la comprobación del árbol limpio, el humo del binario y `deadcode-ratchet` en un solo job que además espera a `Go Format`. Medido en la PR #310 (ejecución 45714038862): 4:02 de pared; `Unit Tests` 3:23, de los que `go test ./...` son 2:58.
- En Windows (ejecución 45650722391, tras retirar RDD) el shard `rest` tarda 7:50 en el paso de pruebas y marca el tiempo de pared (unos 9 min); `cli-n-o` tarda 0:21. En `rest` pesan `internal/branch/engine` (431 s) e `internal/components/sdd` (291 s).

## Decisiones

- **Check obligatorio.** El ruleset de `main` exige `Go Format` y `Unit Tests`. Ninguno se renombra: `Unit Tests` pasa a ser un job agregador que falla si `Go Format` o cualquier job hijo no termina en `success`. Un PR solo de artefactos (`artifacts_only`) sigue omitiendo hijos y agregador, y la omisión cuenta como superada.
- **Hijos sin esperar a `Go Format`.** Los jobs hijos solo dependen de `changes` y arrancan en paralelo con el formato; el agregador conserva la guarda de formato (`TestRequiredChecksFailClosedWhenFormatFails`).
- **Partición de Linux.** `go test` en cuatro grupos por coste medido: `cli`, `app-assets-update`, `engine-sdd` y `rest`. `rest` se deriva de `go list ./...` menos lo que nombran los demás, así que un paquete nuevo queda cubierto el día que se añade. Humo del binario y `deadcode-ratchet`, cada uno en su job.
- **Partición de Windows.** `cli` en dos shards (`A-O`, `P-Z`), `internal/branch/engine` en dos por nombre (`A-E`, `F-Z`), `internal/components/sdd` en uno propio y `rest` con el resto. `rest` excluye lo que nombran los demás shards mediante el campo `exclude` de la matriz, no una regex fija en el paso.
- **Validación de la partición.** `TestWindowsFullSuiteShardsCoverEveryTestName` se conserva y se amplía: comprueba además que `exclude` de `rest` es exactamente la unión de los paquetes de los demás shards. Un test hermano hace lo mismo con la matriz de Linux.
- **`bench/` (P21).** Ya no queda ningún paso en los workflows (comprobado con `grep -i bench .github/workflows`).
- Fuera de alcance: `darwin-runtime`, `organic-runtime-e2e` y `e2e-tests` (solo manuales).

## Pasos de `Unit Tests` antes y después

| Antes (un job) | Después |
|---|---|
| Require successful Go format | `Unit Tests` (agregador): misma guarda |
| `go test ./...` | `Go Tests (<grupo>)`: `go test` del grupo, 4 jobs |
| Verify the checkout is clean after the unit tests | En cada `Go Tests (<grupo>)` |
| Smoke the canonical binary surface | Job `Binary Smoke`; compara con árbol limpio porque ya no le preceden pruebas |
| Check for new unreachable functions | Job `Deadcode Ratchet` |
| — | `Unit Tests`: exige `success` de los tres jobs anteriores |

## Alcance autorizado

`.github/workflows/ci.yml`, `.github/workflows/windows-full-suite.yml`, `internal/assets/formatter_ordering_test.go`, `internal/update/windows_distribution_policy_test.go` (exige la cadena `go test ./...` en `ci.yml`) y `docs/testing/windows-tests.md` si hace falta.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false`. Comprobaciones: `go test ./internal/assets ./internal/update -run` de los tests tocados y la propia CI de la PR (con `ci:windows` para validar los shards). Sin batería completa local ni binarios sueltos.

## Entrega

`single-pr`, un commit por tarea. Etiquetas `type:chore` y `ci:windows`.

## Tareas

- [x] T1 · Partir `Unit Tests` en jobs paralelos con agregador (`ci.yml` y tests de guarda).
- [ ] T2 · Rebalancear `windows-full-suite.yml` y ampliar la guarda de la partición.
- [ ] T3 · Medir la CI de la PR (antes y después) y registrar el resultado aquí.

Ruta de todas las tareas: inline (unidades pequeñas sobre ficheros de CI ya leídos).

## Criterio de terminado

- Tiempo de pared de la CI de una PR, antes y después, registrado aquí.
- `Unit Tests` conserva el nombre y falla si falla cualquier job hijo.
- El atajo `artifacts_only` (`odd/`, `openspec/`, `docs/plan/`) sigue omitiendo hijos y agregador.
- Lista de pasos antes y después sin pérdidas no justificadas (tabla de arriba).

## Progreso

- T1: agregador `Unit Tests` y jobs `Go Tests (cli | app-assets-update | engine-sdd | rest)`, `Binary Smoke` y `Deadcode Ratchet`. Guardas nuevas en `internal/assets/workflow_shards_test.go`: `TestUnitTestShardsPartitionEveryPackage` (comprobada con la mutación que quita `./internal/update` de `exclude`: falla) y `TestUnitTestsAggregatorRequiresEveryUnitJob`. `windows_distribution_policy_test.go` exige ahora `go list ./...` en lugar de `go test ./...`. `go vet` y `go test ./internal/assets ./internal/update -run` de los tests tocados, en verde.
