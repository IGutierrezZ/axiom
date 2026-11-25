# Informe de verificación: retirar-rdd

Rol: `fullstack` (verify del rol = verify del cambio). Alcance: diff `3c520778..HEAD` (rama `inc/retirar-rdd`, HEAD `c3c1a123`; 1093 ficheros, +5269/-237898). Estilo `continuous`, almacén openspec.

## Resumen

La retirada de RDD está completa y coherente con proposal, specs, diseño y tareas (78/78 tareas marcadas, 0 pendientes). Build, vet, gofmt, tidy y toda la suite pasan; dos pruebas fallaron solo bajo la carga de la ejecución paralela completa y pasan al repetirlas aisladas (paquetes no tocados por el diff). 0 CRITICAL, 2 WARNING, 3 SUGGESTION.

## Cumplimiento técnico

| Comprobación | Resultado |
|---|---|
| `go build ./... && go vet ./...` | exit 0 |
| `go test ./internal/cli` en 4 shards paralelos (`^Test[A-M]`, `^Test[N-O]`, `^Test[P-Q]\|^Test[S-Z]`, `^TestR`; `-count=1 -timeout 120m`) | los cuatro `ok` (sin los fallos de entorno conocidos) |
| `go test $(go list ./... \| grep -v '/internal/cli$') -count=1 -timeout 180m` | 106 paquetes `ok`; 2 fallos bajo carga (ver W1, W2), ambos `ok` aislados |
| `gofmt -l .` | vacío |
| `go mod tidy` + `git diff --exit-code go.mod go.sum` | sin diff residual (exit 0) |
| `go list ./... \| grep -E 'reviewtransaction\|reviewerprovider\|rddharness'` | vacío |
| `git grep` de `reviewtransaction\|reviewerprovider\|rddharness\|bench-model-picker\|bench_fixture\|rdd_harness` fuera de `openspec/changes`, `openspec/specs`, `docs/plan`, `docs/archive`, `odd` | solo `docs/audits/2026-07-23-*` y `2026-07-27-*`, `docs/releases/v2.2.0-closure-ledger.md`, `docs/upstream-absorption-ledger.md` (históricos conservados por D14) y `docs/releases/notas-retirada-rdd.md` (nota de migración, que debe nombrarlos) |
| Pruebas de seguridad de release (`internal/update/{release_security,windows_distribution_policy}_test.go`) | 13 + 6 = 19 funciones `Test` presentes (fixup `d79331b3`) y en `ok` |

## Conservaciones

- `ScreenReview` (`internal/tui/model.go`), `axiom sdd-attempt grant` (`cmd/axiom/main.go`, `internal/app/app.go`), `internal/consentenvelope` (con tests en `ok`), `kickoff.CommitReview*` (`internal/kickoff/commit_review_test.go`, `machine.go`, `schema.go`) y Judgment Day con `review-ledger` (`internal/assets/skills/judgment-day/references/prompts-and-formats.md:59`, `internal/assets/claude/agents/jd-judge-{a,b}.md:50`): presentes.
- No queda prosa `axiom review` en `judgment-day`.

## Matriz de requisitos

| Requisito | Implementación / prueba | Resultado |
|---|---|---|
| REQ-46.1 sin `axiom review*`, `stop-hook`, `bench-model-picker` | verbos y paquetes borrados; ausencia de paquetes verificada; tests de `internal/cli` | cumple |
| REQ-46.2 release sin bundle `review-provider-contract` | C5; `internal/update`, `releasepolicy`, `releaseprovenance*` en `ok` | cumple |
| REQ-46.3 doctrina sin prosa RDD | tests de assets/`components/sdd` en `ok` | cumple |
| REQ-46.4 `sync` sin agentes/plugins/skill/hook RDD | tests de `sync` (selección persistida ignorada) en shards `ok` | cumple |
| REQ-46.5 estado tolera restos RDD | `TestLegacyRDDSwitchFieldsAreIgnoredAndDroppedOnTheNextWrite` (C10) en `ok` | cumple |
| REQ-46.6 dashboard sin modo de revisión | `review_mode_retired_test` (C7), `internal/dashboard` en `ok` | cumple |
| REQ-46.7 rutas Windows en JSON de SDD | trasladado desde requisito previo; `internal/sddstatus` en `ok` | cumple |
| REQ-46.8 aislamiento de CWD | trasladado; `internal/cli` en `ok` | cumple |
| REQ-46.9 preset de skills saneado | `TestSelectedSkillIDsDropsRetired` | cumple |
| REQ-46.10 Engram con proyecto ambiguo | trasladado; assets en `ok` | cumple |
| REQ-28.43 juicio con dos jueces ciegos | Judgment Day conservado | cumple |
| MODIFIED/REMOVED (35 deltas) | comportamiento y código ya inexistentes (`go list` vacío, `git grep` limpio) | cumple |

Nota: la trazabilidad por identificador `REQ-46.x` en los tests es parcial (solo REQ-46.3 y REQ-46.4 aparecen citados en ficheros de prueba); el resto se ha trazado por nombre de prueba y por el Registro de ejecución.

## Revisión de desviaciones

- Suelo del ratchet de rechazos 1000 → 400 (`internal/cli/refusal_resolution_ratchet_test.go:630`): coherente, el comentario documenta ~590 sitios restantes; sigue detectando un recorrido roto (S1).
- Podas en lugar de borrados, baselines (`.deadcode-baseline.txt`, ratchet, guard-population) solo con borrados: aceptable.
- `go.mod -1`, `go.sum` +8/-5: sumas nuevas de `testify`/`objx`/`go-spew`/`go-difflib` por dependencias antes transitivas; `go mod tidy` no deja diff.
- Nota `docs/releases/notas-retirada-rdd.md`: cumple; el aviso inicial exige limpiar a mano, ANTES de instalar, las entradas `axiom review stop-hook --agent claude-code` de `hooks.Stop`/`hooks.SessionStart` de `~/.claude/settings.json` (punto 1), con el motivo.
- `internal/components/uninstall/service.go:1725-1726` conserva el reconocimiento de `review stop-hook` para limpiar restos: coherente con la retirada.

## Borrados de más (lección de C5)

Se muestrearon los 398 `_test.go` borrados enteros: bench/, `internal/cli/review_*`, `reviewtransaction`, `components/sdd/rdd_*`/`review_*`, `providercontractbundle`, `tui/review_store_reset`, `dashboard/*review*`, `hub/rddpref`, `kickoff/rdd_boundary`, etc. Los no obvios (`branch_lineage_test`, `continuation_paths_test`, `components/sdd/main_test.go`, `adapter_*_guard_test`, `sdd_no_review_test`) eran todos RDD (consumían `reviewtransaction`/`rddharness`). Única pérdida colateral: ver W2/S2.

## Hallazgos

### CRITICAL
Ninguno.

### WARNING
- W1. Inestabilidad bajo carga, no atribuible al diff: `internal/components/communitytool` `TestPiCodeGraphPendingProbePreservesConfiguredFiles` (`pi_codegraph_test.go:574`, timeout de sonda MCP) e `internal/sddpublish` `TestCommitExcludesArchivedLocalState` (`gitport_test.go:477`) fallaron en la ejecución completa con cinco procesos de `go test` simultáneos y pasaron aislados (`-count=3` en el segundo). El diff no toca ninguno de los dos paquetes.
- W2. Cobertura no RDD perdida: `internal/hub/rddpref_test.go` incluía `TestProjectForRepoResolvesRoleAndSpecsReposToTheMasterRecord`, que además de `LookupProjectRDD` comprobaba la resolución de repos de rol/specs hacia el registro maestro; `ProjectForRepo` se borró con `rddpref.go` y ninguna prueba actual lo sustituye. Si `ProjectForRepo` no tiene otros consumidores (build limpio lo indica), es inocuo; si la resolución maestro/rol se necesita, queda sin cubrir.

### SUGGESTION
- S1. Considerar que el suelo del ratchet (400 vs ~590 reales) deja margen holgado; revisarlo cuando el recuento se estabilice.
- S2. Citar los identificadores `REQ-46.x` en los tests nuevos para trazabilidad directa.
- S3. `docs/audits/2026-07-23-*`/`2026-07-27-*` y `docs/upstream-absorption-ledger.md` conservan referencias textuales a paquetes retirados: correcto por D14, pero conviene que la búsqueda de ausencia de CI las excluya explícitamente.

## Limitaciones
No se ejecutó `axiom` ni se compiló ningún binario (EDR). Los fallos de entorno conocidos (symlinks de Windows, mensajes del SO en castellano) no aparecieron en esta ejecución.

**Estado:** limpio
