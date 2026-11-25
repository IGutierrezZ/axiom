# Revisión de cobertura: spec de `retirar-rdd`

Informativa: no bloquea fases ni archivo y no corrige la spec. Con el estilo `continuous`, los huecos pasan al resumen final.

## Ronda 1 — 2026-11-25

Entradas: `proposal.md` y los 35 deltas de `specs/`. Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Ambos devolvieron un resultado utilizable.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | requisito sin cubrir | `specs/` (sin capacidad) — `bench/` | Ningún requisito exige que `bench/`, `results.json` y `skills/axiom-bench` no existan ni que la CI no los invoque. Tampoco recoge el ratchet sin `bench/classify.go` ni la decisión sobre los ejes de `bench/` que no son RDD. Solo cubre `bench-model-picker` (REQ-46.1). | A, B |
| 2 | requisito sin cubrir | `sdd-independent-review` | La propuesta anuncia un MODIFIED que quite las 12 menciones RDD, pero el delta solo trae ADDED REQ-28.43 y REMOVED REQ-28.11. El propósito conserva «Son ajenas a RDD». | A |
| 3 | requisito sin cubrir | `axiom-distribution-identity` (notas de la versión) | Ningún requisito obliga a publicar la migración manual en las notas de la versión ni a declarar retirado `review-provider-contract` en la PR y en las notas. | A, B |
| 4 | caso límite | `sdd-orchestrator-assets` REQ-46.4 | Faltan escenarios de restos heredados: `uninstall` ya no borra los `review-*` instalados (consecuencia aceptada sin escenario); un hook `Stop` residual falla con verbo desconocido; un `opencode.json` u overlay con `review-*` no rompe `sync`; se conservan los hooks ajenos de `settings.json`; y existe el plugin `review-result-artifacts.ts` heredado. | A, B |
| 5 | caso límite | `axiom-user-state-and-env` REQ-46.5 | La lectura tolerante no cubre las asignaciones de modelo de roles `rdd-*`/`review-*` (Codex, Claude, OpenCode), un hub con `RDDMode`, un `rdd_mode` de tipo inesperado ni un JSON parcial. | A, B |
| 6 | requisito sin cubrir | `axiom-sdd-cli-integration` / sin capacidad | Ningún requisito normativo recoge el criterio `rg -l 'reviewtransaction\|reviewerprovider\|rddharness'` sin resultados fuera de los históricos (sin lápidas), ni la retirada de la prosa RDD de `README.md`, `AGENTS.md` y las docs vivas. | A, B |
| 7 | caso límite | `change-branch-lifecycle` | No hay escenario de que `done` termine bien e ignore sin aviso un almacén residual `<git common dir>/gentle-ai/...` de RDD, ni de que desaparezcan `LineageProbe`/`DefaultWithLineages`. | A |
| 8 | requisito sin cubrir | sin capacidad (`sddstatus` `CommonDir`) | El paso de `CompactAuthoritativeStore` a `gitcommon.CommonDir` exige equivalencia de ruta (absoluta, worktrees enlazados, enlaces simbólicos) y ningún delta lo protege. El ledger de Judgment Day podría cambiar de ubicación sin que nada lo detecte. | A, B |
| 9 | requisito sin cubrir | sin capacidad (CI) | No hay requisito de que los shards de `windows-full-suite.yml` cubran todos los tests que quedan, de que `opencode-v2-contracts` conserve el `tsc` de los plugins no RDD ni de que la CI deje de ejecutar `-count=100`, `rdd_harness` y `bench_fixture`. | A, B |
| 10 | caso límite | `local-web-dashboard` REQ-46.6 / TUI | Faltan: la respuesta 404 (no 500) del cliente antiguo a `/api/project/review-mode`; `hub` sin `RDDMode`; y la ausencia de las pantallas `review_mode`, `install_review_mode` y `review_store_reset`, distinta de la del menú de bienvenida (REQ-16.2). | A, B |
| 11 | caso límite | `sdd-orchestrator-assets` | Falta un escenario de que se conservan `ScreenReview` («Review and Confirm»), `sdd-attempt grant`, `consentenvelope` y `kickoff.CommitReview*`. Un borrado demasiado agresivo no lo detectaría ninguna spec. | A, B |
| 12 | caso límite | `sdd-orchestrator-assets` REQ-46.3 | Solo `rdd-defect-workflow` tiene requisito de ausencia. Faltan `rdd-advisory-transport` y `axiom-bench` en el catálogo, los presets y `skill_picker`, y la ausencia de los agentes `review-*` embebidos: el escenario solo mira el directorio de salida. | A, B |
| 13 | requisito sin cubrir | `rdd-decoupling-v3-stability` REQ-18.1 | REQ-18.1 se retira sin traslado y contiene comportamiento SDD que sobrevive: `axiom sdd status --json` no emite `reviewOffer`, y `dependencies.archive` es `ready` sin depender de ninguna revisión de código. | B |
| 14 | caso límite | `rdd-sdd-receipt-consumption` | Se retira la lectura compatible de `reviewGate` v1 y `ReceiptRef` sin requisito que asegure que un ledger de intentos SDD antiguo con esos campos sigue cargando o se ignora. | B |

### Tratamiento en esta corrida

Con el estilo `continuous` no se relanza `sdd-spec` por estos huecos. El diseño y las tareas cubren con pruebas los que son comprobables en código: la equivalencia de `CommonDir` (8), la carga tolerante (5 y 14), los shards (9), la ausencia por `rg` (6) y la conservación de `ScreenReview` y `sdd-attempt` (11). El resumen final los lista para que la persona decida si pide otra corrida de la spec.
